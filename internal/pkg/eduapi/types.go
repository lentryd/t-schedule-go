package eduapi

import "encoding/json"

// Only the fields actually consumed by the bot are modeled; the upstream API
// returns much larger payloads (mirrors the trimmed usage in wrapper.ts).

// tokenAuthResponse mirrors POST /api/tokenauth. On success the payload is
// double-nested - {data:{data:{accessToken, id}, accessToken}} - as described
// by wrapper.ts's TokenAuthResponse; on failure `data` collapses to the
// string "uNull", hence the json.RawMessage and the lenient decoding below.
type tokenAuthResponse struct {
	State       int             `json:"state"`
	Msg         string          `json:"msg"`
	AccessToken string          `json:"accessToken"`
	Data        json.RawMessage `json:"data"`
}

type tokenAuthData struct {
	AccessToken string         `json:"accessToken"`
	ID          int64          `json:"id"`
	Data        *tokenAuthData `json:"data"`
}

// user returns the authenticated user's token and id, taking them from the
// nested payload and falling back to the outer levels.
func (r tokenAuthResponse) user() tokenAuthData {
	var outer tokenAuthData
	if len(r.Data) > 0 {
		_ = json.Unmarshal(r.Data, &outer)
	}

	user := outer
	if outer.Data != nil {
		user = *outer.Data
	}
	if user.AccessToken == "" {
		user.AccessToken = outer.AccessToken
	}
	if user.AccessToken == "" {
		user.AccessToken = r.AccessToken
	}
	if user.ID == 0 {
		user.ID = outer.ID
	}
	return user
}

type userInfoResponse struct {
	State int    `json:"state"`
	Msg   string `json:"msg"`
	Data  struct {
		StudentID        int64 `json:"studentID"`
		EliteEducationID int64 `json:"eliteEducationID"`
	} `json:"data"`
}

type apiStudent struct {
	StudentID int64  `json:"studentID"`
	FullName  string `json:"fullName"`
	Fio       string `json:"fio"`
	Course    int64  `json:"course"`
}

type studentListResponse struct {
	Data struct {
		AllStudent []apiStudent `json:"allStudent"`
	} `json:"data"`
}

type raspTeacher struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

type raspInfo struct {
	ModuleName     string        `json:"moduleName"`
	Theme          string        `json:"theme"`
	Aud            string        `json:"aud"`
	Link           string        `json:"link"`
	GroupName      string        `json:"groupName"`
	Type           string        `json:"type"`
	IsControlEvent bool          `json:"isControlEvent"`
	Teachers       []raspTeacher `json:"teachers"`
}

type raspListItem struct {
	Name  string   `json:"name"`
	Color string   `json:"color"`
	Start string   `json:"start"`
	End   *string  `json:"end"`
	Info  raspInfo `json:"info"`
}

type raspListResponse struct {
	Data struct {
		RaspList []raspListItem `json:"raspList"`
	} `json:"data"`
}

type lessonType struct {
	Label        string `json:"label"`
	Abbreviation string `json:"abbreviation"`
}

type lessonsTypesResponse struct {
	Data struct {
		LessonsTypes []lessonType `json:"lessonsTypes"`
	} `json:"data"`
}

// reserveRaspItem mirrors one entry of the reserve (iCal-derived) schedule
// response, which uses Cyrillic field names (RaspResponse in wrapper.ts).
type reserveRaspItem struct {
	ДатаНачала    string `json:"датаНачала"`
	ДатаОкончания string `json:"датаОкончания"`
	Дисциплина    string `json:"дисциплина"`
	Аудитория     string `json:"аудитория"`
	Тема          string `json:"тема"`
	Группа        string `json:"группа"`
	Ссылка        string `json:"ссылка"`
	Преподаватель string `json:"преподаватель"`
}

type reserveRaspResponse struct {
	Data struct {
		Rasp []reserveRaspItem `json:"rasp"`
	} `json:"data"`
}
