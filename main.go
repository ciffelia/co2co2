package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Record struct {
	Timestamp   ISO8601Time `json:"timestamp"`
	Co2         int64       `json:"co2"`
	Temperature float64     `json:"temperature"`
	Humidity    float64     `json:"humidity"`
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %v /path/to/serial_device", os.Args[0])
	}
	device := os.Args[1]

	p, err := openSerialPort(device)
	if err != nil {
		log.Panicf("failed to open serial device `%v`: %+v", device, err)
	}
	defer p.Close()
	log.Println("serial port opened")

	s, err := startDevice(p)
	if err != nil {
		log.Panicf("failed to start device: %+v", err)
	}
	defer func() {
		if err := stopDevice(p, s); err != nil {
			log.Panicf("failed to stop device: %+v", err)
		}
	}()
	log.Println("device started")

	// trap SIGINT
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

		<-sigCh

		// send STP command
		if _, err := p.Write([]byte("STP\r\n")); err != nil {
			panic(err)
		}
	}()

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	for s.Scan() {
		ts := time.Now().In(jst)
		text := s.Text()

		if text[:6] == "OK STP" {
			// the device was stopped by STP command (due to SIGINT)
			log.Println("device stopped")
			p.Close()
			os.Exit(130)
		}

		msg, err := parseMessage(text)
		if err != nil {
			log.Panicf("failed to parse message `%v`: %+v", text, err)
		}

		if err := handleMessage(ts, msg); err != nil {
			panic(err)
		}
	}

	if s.Err() == nil {
		log.Panicf("failed to read from serial device: reached EOF")
	}
	log.Panicf("failed to read from serial device: %+v", s.Err())
}

func handleMessage(ts time.Time, msg *message) error {
	record := &Record{
		Timestamp:   ISO8601Time(ts),
		Co2:         msg.co2,
		Temperature: msg.temperature,
		Humidity:    msg.humidity,
	}

	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	fmt.Println(string(b))

	return nil
}
