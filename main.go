package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %v /path/to/serial_device", os.Args[0])
	}
	device := os.Args[1]

	ctx := datadog.NewDefaultContext(context.Background())

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
	recordLastSubmittedAt := time.Unix(0, 0)
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

		record := CreateRecord(ts, msg)
		if err := printRecord(record); err != nil {
			log.Panicf("failed to print record: %+v", err)
		}

		// submit metrics to DataDog every minute
		if ts.Sub(recordLastSubmittedAt) > 1*time.Minute {
			if err := SubmitRecord(ctx, record); err != nil {
				log.Panicf("failed to send metrics to DataDog: %+v", err)
			}
			recordLastSubmittedAt = ts
		}
	}

	if s.Err() == nil {
		log.Panicf("failed to read from serial device: reached EOF")
	}
	log.Panicf("failed to read from serial device: %+v", s.Err())
}

func printRecord(record *Record) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	fmt.Println(string(b))

	return nil
}
