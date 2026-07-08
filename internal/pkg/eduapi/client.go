// Package eduapi is a client for the edu.donstu.ru student portal API,
// mirroring src/utils/wrapper.ts.
package eduapi

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"t-schedule/internal/store"
)

const (
	origin     = "https://edu.donstu.ru/"
	tokenURL   = origin + "api/tokenauth"
	raspURL    = origin + "api/RaspManager"
	studentURL = origin + "api/GroupManager/GetAllStudentSchoolX"
	userAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
	// Fingerprint header required by edu.donstu.ru's WAF; same value used by the Node bot.
	fingerprint = "hs%40SHzqNbc_N9O1_a730jgPIaCMnKrW5K6YyiaTh-PbJ24VPvjM%3FObXBb38EnDC1FJCC5DeAoU1UFGVmx%247reZqyiXVwiO%40%40NqEt1SasELZ9rC%40VgVWqMaWviTjhxzgYZ5HeBs0emOmS-xHfb-OM7V"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func closeBody(body io.Closer) {
	_ = body.Close()
}

// AuthResult is returned by TryAuth on success.
type AuthResult struct {
	AccessToken string
	SpaceID     int64
	StudentID   int64
}

// TryAuth attempts to authenticate against edu.donstu.ru with the given
// credentials, returning the derived access token, education space and
// student ID. Mirrors Wrapper.tryAuth().
func TryAuth(userName, password string) (*AuthResult, error) {
	body, _ := json.Marshal(map[string]string{"userName": userName, "password": password})

	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setCommonHeaders(req, "https://edu.donstu.ru/WebApp/#/login")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp.Body)

	var tokenResp tokenAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	accessToken := tokenResp.Data.Data.AccessToken
	studentID := tokenResp.Data.Data.ID * -1

	if accessToken == "" {
		return nil, fmt.Errorf("eduapi: empty access token in tokenauth response")
	}

	infoReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%sapi/UserInfo/Student?studentID=%d", origin, studentID), nil)
	if err != nil {
		return nil, err
	}
	infoReq.Header.Set("Authorization", "Bearer "+accessToken)

	infoResp, err := httpClient.Do(infoReq)
	if err != nil {
		return nil, err
	}
	defer closeBody(infoResp.Body)

	var userInfo userInfoResponse
	if err := json.NewDecoder(infoResp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	if userInfo.Data.EliteEducationID == 0 {
		return nil, fmt.Errorf("eduapi: could not resolve education space")
	}

	return &AuthResult{
		AccessToken: accessToken,
		SpaceID:     userInfo.Data.EliteEducationID,
		StudentID:   studentID,
	}, nil
}

// GetReserveRasp fetches the reserve (iCal-derived) schedule for a student,
// used as a fallback when the main API is unavailable. Mirrors
// Wrapper.getReserveRasp().
func GetReserveRasp(studentID int64) ([]ScheduleFormat, error) {
	start, end := monthWindow()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%sapi/Rasp?idStudent=%d", origin, studentID), nil)
	if err != nil {
		return nil, err
	}
	setCommonHeaders(req, "https://edu.donstu.ru/WebApp/#/Rasp/Group/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp.Body)

	var raspResp reserveRaspResponse
	if err := json.NewDecoder(resp.Body).Decode(&raspResp); err != nil {
		return nil, err
	}

	items := formatRasp(raspResp.Data.Rasp)
	return filterByWindow(items, start, end), nil
}

// GetRaspHash returns a SHA-1 hash of a student's iCal schedule feed, used to
// cheaply detect whether the schedule changed. Mirrors Wrapper.getRaspHash().
func GetRaspHash(studentID int64) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%sapi/Rasp?idStudent=%d&iCal=true", origin, studentID), nil)
	if err != nil {
		return "", err
	}
	setCommonHeaders(req, "https://edu.donstu.ru/WebApp/#/Rasp/Group/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer closeBody(resp.Body)

	h := sha1.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Client authenticates as a specific provider (a real edu.donstu.ru account
// used on behalf of the bot) to fetch schedules and student lists. Mirrors
// the instance methods of Wrapper.
type Client struct {
	spaceID     int64
	providerID  string
	userName    string
	password    string
	accessToken string

	store *store.Store
}

// NewClient creates a Client for the given provider. providerID may be empty
// when the access token doesn't need to be persisted back to Firestore.
func NewClient(st *store.Store, provider store.ProviderData, providerID string) *Client {
	return &Client{
		spaceID:     provider.EducationSpaceID,
		providerID:  providerID,
		userName:    provider.UserName,
		password:    provider.Password,
		accessToken: provider.AccessToken,
		store:       st,
	}
}

// Auth logs the provider in and stores the resulting access token. Mirrors
// Wrapper.Auth().
func (c *Client) Auth(ctx context.Context) error {
	body, _ := json.Marshal(map[string]any{
		"isParent":       false,
		"recaptchaToken": nil,
		"userName":       c.userName,
		"password":       c.password,
	})

	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)

	var tokenResp tokenAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}
	if tokenResp.Data.AccessToken == "" {
		return fmt.Errorf("eduapi: can't get token for user %s", c.userName)
	}

	c.accessToken = tokenResp.Data.AccessToken

	if c.providerID != "" && c.store != nil {
		if err := c.store.UpdateProviderAccessToken(ctx, c.providerID, c.accessToken); err != nil {
			return err
		}
	}
	return nil
}

// CheckSession reports whether the current access token is still valid.
// Mirrors Wrapper.checkSession().
func (c *Client) CheckSession() bool {
	req, err := http.NewRequest(http.MethodGet, tokenURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Cookie", "authToken="+c.accessToken)
	setCommonHeaders(req, "https://edu.donstu.ru/WebApp/#/RaspManager/Calendar")

	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer closeBody(resp.Body)

	var body struct {
		State int `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}
	return body.State == 1
}

func (c *Client) ensureSession(ctx context.Context) error {
	if c.CheckSession() {
		return nil
	}
	return c.Auth(ctx)
}

// GetLessonsTypes fetches the lesson type dictionary for the client's
// education space. Mirrors Wrapper.getLessonsTypes().
func (c *Client) GetLessonsTypes(ctx context.Context) ([]lessonType, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%sapi/SuggestedModules/LessonsTypes?educationSpaceID=%d", origin, c.spaceID), nil)
	if err != nil {
		return nil, err
	}
	c.authHeaders(req)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp.Body)

	var out lessonsTypesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data.LessonsTypes, nil
}

// GetRaspList fetches a student's schedule for the current and next month.
// Mirrors Wrapper.getRaspList().
func (c *Client) GetRaspList(ctx context.Context, studentID int64) ([]ScheduleFormat, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	lessonsTypes, err := c.GetLessonsTypes(ctx)
	if err != nil {
		return nil, err
	}

	start, end := monthWindow()

	base := fmt.Sprintf("%s?showAll=true&studentsIDs=%d&educationSpaceID=%d&showJournalFilled=false", raspURL, studentID, c.spaceID)

	var all []ScheduleFormat
	for _, month := range []int{int(start.Month()), int(end.Month())} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s&month=%d", base, month), nil)
		if err != nil {
			return nil, err
		}
		c.authHeaders(req)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		var raspResp raspListResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&raspResp)
		closeBody(resp.Body)
		if decodeErr != nil {
			return nil, decodeErr
		}

		all = append(all, formatSchedule(raspResp.Data.RaspList, lessonsTypes)...)
	}

	return filterByWindow(all, start, end), nil
}

// GetStudentList fetches the list of students visible to this provider.
// Mirrors Wrapper.getStudentList().
func (c *Client) GetStudentList(ctx context.Context) ([]store.Student, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?educationSpaceID=%d", studentURL, c.spaceID), nil)
	if err != nil {
		return nil, err
	}
	c.authHeaders(req)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp.Body)

	var out studentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	students := make([]store.Student, 0, len(out.Data.AllStudent))
	for _, s := range out.Data.AllStudent {
		if c.spaceID <= 0 {
			continue
		}
		students = append(students, store.Student{
			ID:        s.StudentID,
			Course:    s.Course,
			SpaceID:   c.spaceID,
			FullName:  s.FullName,
			ShortName: s.Fio,
		})
	}
	return students, nil
}

func (c *Client) authHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Cookie", "authToken="+c.accessToken)
	setCommonHeaders(req, "https://edu.donstu.ru/WebApp/#/RaspManager/Calendar")
}

func setCommonHeaders(req *http.Request, currentPath string) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Fp", fingerprint)
	req.Header.Set("Current-Path", currentPath)
	req.Header.Set("Referer", "https://edu.donstu.ru/WebApp/")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", userAgent)
}

// monthWindow returns [start of this month, end of next month] in Moscow time.
func monthWindow() (time.Time, time.Time) {
	now := time.Now().In(moscowLocation)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, moscowLocation)
	end := start.AddDate(0, 2, -1)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, moscowLocation)
	return start, end
}

func filterByWindow(items []ScheduleFormat, start, end time.Time) []ScheduleFormat {
	out := make([]ScheduleFormat, 0, len(items))
	for _, item := range items {
		t, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			continue
		}
		if !t.Before(start) && !t.After(end) {
			out = append(out, item)
		}
	}
	return out
}
