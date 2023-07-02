package main

import (
	"time"
)

type Record struct {
	Timestamp   ISO8601Time `json:"timestamp"`
	Co2         int64       `json:"co2"`
	Temperature float64     `json:"temperature"`
	Humidity    float64     `json:"humidity"`
}

func CreateRecord(ts time.Time, msg *message) *Record {
	return &Record{
		Timestamp:   ISO8601Time(ts),
		Co2:         msg.co2,
		Temperature: msg.temperature,
		Humidity:    msg.humidity,
	}
}
