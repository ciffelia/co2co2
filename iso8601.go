package main

import (
	"encoding/json"
	"time"
)

type ISO8601Time time.Time

func (t ISO8601Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format("2006-01-02T15:04:05.000Z07:00"))
}
