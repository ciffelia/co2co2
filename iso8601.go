package main

import (
	"encoding/json"
	"time"
)

type ISO8601Time time.Time

func (t ISO8601Time) format() string {
	return time.Time(t).Format(time.RFC3339)
}

func (t ISO8601Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.format())
}
