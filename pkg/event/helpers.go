package event

type updateEvent struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}
