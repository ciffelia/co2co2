package main

import (
	"database/sql/driver"
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

func (t ISO8601Time) Value() (driver.Value, error) {
	return t.format(), nil
}
