package eduapi

import "testing"

func TestFindAbbreviationKeepsCyrillic(t *testing.T) {
	types := []lessonType{{Label: "Лекция", Abbreviation: "Лек."}}

	if got := findAbbreviation("Лекция", types); got != "Лек." {
		t.Errorf("findAbbreviation = %q, want %q", got, "Лек.")
	}
	if got := findAbbreviation("лекция", []lessonType{}); got != "Лекц" {
		t.Logf("fallback abbreviation = %q", got)
	}
}

func TestFormatScheduleDoesNotDuplicateType(t *testing.T) {
	end := "2026-09-04T11:50:00+03:00"
	items := []raspListItem{{
		Name:  "Лек. Проблематизация",
		Start: "2026-09-04T10:20:00+03:00",
		End:   &end,
		Info:  raspInfo{Type: "Лекция"},
	}}

	got := formatSchedule(items, []lessonType{{Label: "Лекция", Abbreviation: "Лек."}})
	if len(got) != 1 {
		t.Fatalf("got %d events", len(got))
	}
	if got[0].Summary != "Лек. Проблематизация" {
		t.Errorf("Summary = %q, want %q", got[0].Summary, "Лек. Проблематизация")
	}
}

func TestFormatScheduleSkipsEventsWithoutEnd(t *testing.T) {
	items := []raspListItem{{
		Name:  "Лек. Проблематизация",
		Start: "2026-09-04T10:20:00+03:00",
		End:   nil,
		Info:  raspInfo{Type: "Лекция"},
	}}

	if got := formatSchedule(items, nil); len(got) != 0 {
		t.Errorf("got %d events, want 0", len(got))
	}
}
