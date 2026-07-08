package eduapi

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"

	"t-schedule/internal/pkg/colorize"

	calendar "google.golang.org/api/calendar/v3"
)

// moscowLocation is used to interpret API timestamps that carry no explicit
// UTC offset (edu.donstu.ru returns local Moscow time).
var moscowLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("Europe/Moscow", 3*60*60)
	}
	return loc
}()

// EventDateTime mirrors the {dateTime, timeZone} shape used by the Google
// Calendar API and by the Node ScheduleFormat type.
type EventDateTime struct {
	DateTime string
	TimeZone string
}

// ScheduleFormat is a calendar-ready representation of a single lesson,
// mirroring ScheduleFormat in src/utils/format.ts.
type ScheduleFormat struct {
	ID     string
	RaspID string
	Etag   string

	Start EventDateTime
	End   EventDateTime

	Summary     string
	ColorID     string
	Location    string
	Description string
}

var (
	urlRegex  = regexp.MustCompile(`[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)?`)
	abbrRegex = regexp.MustCompile(`(^.{2,}?)[aeiouаеёиоуыэюя]`)

	moscowTZ = "Europe/Moscow"
)

// formatSchedule converts raw raspList entries into ScheduleFormat, mirroring
// formatSchedule() in format.ts.
func formatSchedule(items []raspListItem, lessonsTypes []lessonType) []ScheduleFormat {
	out := make([]ScheduleFormat, 0, len(items))

	for _, item := range items {
		endStr := item.Start
		if item.End != nil && *item.End != "" {
			endStr = *item.End
		}

		start, err1 := parseAPITime(item.Start)
		end, err2 := parseAPITime(endStr)
		if err1 != nil || err2 != nil {
			continue
		}

		startISO, endISO := isoString(start), isoString(end)

		summary := strings.TrimSpace(item.Name)

		if item.Info.Type != "" {
			typeAbbr := findAbbreviation(item.Info.Type, lessonsTypes)
			if typeAbbr != "" {
				var firstWord string
				if fields := strings.Fields(summary); len(fields) > 0 {
					firstWord = strings.ToLower(strings.ReplaceAll(fields[0], ".", ""))
				}
				if strings.ToLower(strings.ReplaceAll(typeAbbr, ".", "")) != firstWord {
					summary = typeAbbr + " " + summary
				}
			}
		}

		if item.Info.IsControlEvent {
			summary = "📝 " + summary
		}

		colorID := itoaColor(colorize.NearestColorID(item.Color))
		location := item.Info.Aud

		description := buildDescription(item.Info)

		out = append(out, ScheduleFormat{
			RaspID: hashString(startISO + "-" + endISO + "-" + summary),
			Etag:   hashString(summary + "-" + location + "-" + description),

			Start: EventDateTime{DateTime: startISO, TimeZone: moscowTZ},
			End:   EventDateTime{DateTime: endISO, TimeZone: moscowTZ},

			Summary:     summary,
			ColorID:     colorID,
			Location:    location,
			Description: description,
		})
	}

	return out
}

func findAbbreviation(lessonType string, lessonsTypes []lessonType) string {
	var typeAbbr string
	for _, lt := range lessonsTypes {
		if lt.Label == lessonType {
			typeAbbr = lt.Abbreviation
			break
		}
	}
	if typeAbbr == "" {
		if m := abbrRegex.FindStringSubmatch(lessonType); len(m) > 1 {
			typeAbbr = m[1]
		} else {
			typeAbbr = lessonType
		}
	}
	if typeAbbr == "" {
		return ""
	}
	return strings.ToUpper(typeAbbr[:1]) + typeAbbr[1:]
}

func buildDescription(info raspInfo) string {
	var lines []string

	if info.ModuleName != "" {
		lines = append(lines, info.ModuleName)
	}
	if info.Theme != "" {
		lines = append(lines, info.Theme+"\n")
	}
	if info.GroupName != "" {
		lines = append(lines, "Группа: "+info.GroupName)
	}
	if info.Link != "" && urlRegex.MatchString(info.Link) {
		links := urlRegex.FindAllString(info.Link, -1)
		lines = append(lines, "Ссылки: "+strings.Join(links, ", "))
	}
	if len(info.Teachers) > 0 {
		label := "Преподаватель"
		if len(info.Teachers) > 1 {
			label = "Преподаватели"
		}

		names := make([]string, 0, len(info.Teachers))
		for _, t := range info.Teachers {
			name := t.FullName
			if t.Email != "" {
				name += " (" + t.Email + ")"
			}
			names = append(names, name)
		}
		lines = append(lines, label+": "+strings.Join(names, ", "))
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// formatRasp converts the reserve schedule payload into ScheduleFormat,
// mirroring formatRasp() in format.ts.
func formatRasp(items []reserveRaspItem) []ScheduleFormat {
	out := make([]ScheduleFormat, 0, len(items))

	for _, item := range items {
		start, err1 := time.Parse("2006-01-02T15:04:05-07:00", item.ДатаНачала+"+03:00")
		end, err2 := time.Parse("2006-01-02T15:04:05-07:00", item.ДатаОкончания+"+03:00")
		if err1 != nil || err2 != nil {
			continue
		}

		startISO, endISO := isoString(start), isoString(end)

		summary := strings.TrimSpace(item.Дисциплина)
		location := item.Аудитория

		var lines []string
		if item.Тема != "" {
			lines = append(lines, item.Тема+"\n")
		}
		if item.Группа != "" {
			lines = append(lines, "Группа: "+item.Группа)
		}
		if item.Ссылка != "" && urlRegex.MatchString(item.Ссылка) {
			links := urlRegex.FindAllString(item.Ссылка, -1)
			lines = append(lines, "Ссылки: "+strings.Join(links, ", "))
		}
		if item.Преподаватель != "" {
			label := "Преподаватель"
			if strings.Count(item.Преподаватель, ",") > 0 {
				label = "Преподаватели"
			}
			lines = append(lines, label+": "+item.Преподаватель)
		}
		lines = append(lines, "\nЭто расписание резервного копирования на время возникновения трудностей с доступом к edu.donsu.ru. Пожалуйста, проверьте актуальное расписание на сайте университета. Извините за предоставленные неудобства.")

		description := strings.TrimSpace(strings.Join(lines, "\n"))

		out = append(out, ScheduleFormat{
			RaspID: hashString(startISO + "-" + endISO + "-" + summary),
			Etag:   hashString(summary + "-" + location + "-" + description),

			Start: EventDateTime{DateTime: startISO, TimeZone: moscowTZ},
			End:   EventDateTime{DateTime: endISO, TimeZone: moscowTZ},

			Summary:     summary,
			ColorID:     "11",
			Location:    location,
			Description: description,
		})
	}

	return out
}

// FormatEvent converts a Google Calendar event back into ScheduleFormat, for
// diffing against a freshly fetched schedule (mirrors formatEvent()).
func FormatEvent(event *calendar.Event) ScheduleFormat {
	var startDT, endDT string
	if event.Start != nil {
		startDT = event.Start.DateTime
	}
	if event.End != nil {
		endDT = event.End.DateTime
	}

	// Normalize through the same toISOString-equivalent formatting used by
	// formatSchedule/formatRasp so raspId hashes match regardless of the
	// precision/offset Google echoes back in event.start/end.DateTime.
	startISO, endISO := startDT, endDT
	if t, err := time.Parse(time.RFC3339, startDT); err == nil {
		startISO = isoString(t)
	}
	if t, err := time.Parse(time.RFC3339, endDT); err == nil {
		endISO = isoString(t)
	}

	return ScheduleFormat{
		ID:     event.Id,
		RaspID: hashString(startISO + "-" + endISO + "-" + event.Summary),
		Etag:   hashString(event.Summary + "-" + event.Location + "-" + event.Description),

		Start: EventDateTime{DateTime: startDT, TimeZone: safeTZ(event.Start)},
		End:   EventDateTime{DateTime: endDT, TimeZone: safeTZ(event.End)},

		Summary:     event.Summary,
		ColorID:     event.ColorId,
		Location:    event.Location,
		Description: event.Description,
	}
}

// isoString mirrors JS's Date.prototype.toISOString() (UTC, millisecond
// precision), used so raspId hashes stay identical between freshly fetched
// schedule items and events read back from Google Calendar.
func isoString(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func safeTZ(dt *calendar.EventDateTime) string {
	if dt == nil {
		return ""
	}
	return dt.TimeZone
}

func parseAPITime(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02T15:04:05", v, moscowLocation)
}

func hashString(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func itoaColor(id int) string {
	if id <= 0 {
		return "2"
	}
	return strconv.Itoa(id)
}
