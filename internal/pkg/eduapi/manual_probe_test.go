package eduapi

import (
	"context"
	"os"
	"testing"

	"t-schedule/internal/store"
)

func TestManualAllMethods(t *testing.T) {
	u, p := os.Getenv("EDU_USER"), os.Getenv("EDU_PASS")
	if u == "" {
		t.Skip("no creds")
	}
	ctx := context.Background()

	res, err := TryAuth(u, p)
	if err != nil {
		t.Fatalf("TryAuth: %v", err)
	}
	t.Logf("TryAuth: studentID=%d spaceID=%d token=%d bytes", res.StudentID, res.SpaceID, len(res.AccessToken))

	c := NewClient(nil, store.ProviderData{
		EducationSpaceID: res.SpaceID, UserName: u, Password: p, AccessToken: res.AccessToken,
	}, "")

	t.Logf("CheckSession(stored token): %v", c.CheckSession())

	fresh := NewClient(nil, store.ProviderData{EducationSpaceID: res.SpaceID, UserName: u, Password: p}, "")
	if err := fresh.Auth(ctx); err != nil {
		t.Errorf("Auth: %v", err)
	} else {
		t.Logf("Auth: token=%d bytes, CheckSession=%v", len(fresh.accessToken), fresh.CheckSession())
	}

	students, err := c.GetStudentList(ctx)
	if err != nil {
		t.Errorf("GetStudentList: %v", err)
	} else {
		t.Logf("GetStudentList: %d students, first=%+v", len(students), students[0])
		found := false
		for _, s := range students {
			if s.ID == res.StudentID {
				found = true
				t.Logf("  self: %+v", s)
			}
		}
		if !found {
			t.Errorf("  self (%d) missing from list", res.StudentID)
		}
	}

	lt, err := c.GetLessonsTypes(ctx)
	if err != nil {
		t.Errorf("GetLessonsTypes: %v", err)
	} else {
		t.Logf("GetLessonsTypes: %d types, first=%+v", len(lt), lt[0])
	}

	rasp, err := c.GetRaspList(ctx, res.StudentID)
	if err != nil {
		t.Errorf("GetRaspList: %v", err)
	} else {
		t.Logf("GetRaspList: %d events", len(rasp))
		for _, e := range rasp {
			t.Logf("  summary=%q", e.Summary)
		}
	}

	reserve, err := GetReserveRasp(res.StudentID)
	if err != nil {
		t.Errorf("GetReserveRasp: %v", err)
	} else {
		t.Logf("GetReserveRasp: %d events", len(reserve))
		if len(reserve) > 0 {
			t.Logf("  first=%+v", reserve[0])
		}
	}

	h, err := GetRaspHash(res.StudentID)
	if err != nil {
		t.Errorf("GetRaspHash: %v", err)
	} else {
		t.Logf("GetRaspHash: %s", h)
	}
}
