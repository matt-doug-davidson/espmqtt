package espmqtt

import (
	"fmt"
	"strings"
	"time"
)

type EspMessage struct {
	topic   string
	payload EspPayload
}

func (e *EspMessage) SetTopic(t string) {
	e.topic = t
}

func (e *EspMessage) SetDatetime(t string) {
	e.payload.datatime = t
}

func (e *EspMessage) AppendValue(field string, amount float64, attr string) {
	v := EspValues{field: field, amount: amount, attributes: attr}
	e.payload.values = append(e.payload.values, v)
}

type EspPayload struct {
	datatime string // esp
	values   []EspValues
}

type EspValues struct {
	field      string
	amount     float64
	attributes string // JSON string
}

type ESPDateTime struct {
	Local         string
	LocalTime     time.Time
	Esp           string
	Timestamp     int64
	Seconds       int
	NanoSeconds   int64
	Minutes       int
	Hours         int
	TenMinuteMark int
	Day           int
}

func after(target string, after string) string {
	// Get substring after a string.
	pos := strings.LastIndex(target, after)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(after)
	if adjustedPos >= len(target) {
		return ""
	}
	return target[adjustedPos:len(target)]

}

func FormatESPTime(timestring string) string {
	// Get the time zone. It is going to be added to the
	// time string before converting.
	now := time.Now()
	timeZone, _ := now.Zone()

	// Make sure the milliseconds have 3 digits
	decimal := after(timestring, ".")
	fmt.Println("decimal: ", decimal)
	switch len(decimal) {
	case 0:
		timestring += ".000"
	case 1:
		timestring += "00"
	case 2:
		timestring += "0"
	default:
		return "Error"
	case 3:
	}

	// Remove the T between the date and time
	timeStr := strings.Replace(timestring, "T", " ", 1)
	// Add the time zone.
	datetime := timeStr + " " + timeZone
	// Convert from current location
	t, err := time.Parse("2006-01-02 15:04:05.000 MST", datetime)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Set the location to UTC
	loc, _ := time.LoadLocation("UTC")
	// Determine the current time in the UTC timezone
	utcTime := t.In(loc)
	// Convert to ESP format.
	utcTimeStr := utcTime.Format("2006-01-02T15:04:05.000Z")
	return utcTimeStr
}
