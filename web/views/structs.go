package views

import "time"

type User struct {
	FirstName string
	LastName  string
	FullName  string
	Trade     string `json:"trade"`
}

type WeeklyTimeSheet struct {
	Date time.Time
	Data []CheckInData
}

type Location struct {
	Name string
	Data []CheckInData
}

type CheckInData struct {
	Date     string `json:"date"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Time     string `json:"time"`
	Location string `json:"location"`
	CheckIn  string `json:"check_in"`
	CheckOut string `json:"check_out"`
	Duration string `json:"duration"`
	Trade    string `json:"trade"`
}
