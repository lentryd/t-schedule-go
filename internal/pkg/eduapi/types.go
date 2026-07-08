package eduapi

// Only the fields actually consumed by the bot are modeled; the upstream API
// returns much larger payloads (mirrors the trimmed usage in wrapper.ts).

type tokenAuthResponse struct {
	State int `json:"state"`
	Data  struct {
		AccessToken string `json:"accessToken"`
		Data        struct {
			AccessToken string `json:"accessToken"`
			ID          int64  `json:"id"`
		} `json:"data"`
	} `json:"data"`
}

type userInfoResponse struct {
	Data struct {
		EliteEducationID int64 `json:"eliteEducationID"`
	} `json:"data"`
}

type studentListResponse struct {
	Data struct {
		AllStudent []struct {
			StudentID int64  `json:"studentID"`
			FullName  string `json:"fullName"`
			Fio       string `json:"fio"`
			Course    int64  `json:"course"`
		} `json:"allStudent"`
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
