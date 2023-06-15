package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"go.bug.st/serial"
)

// serial device
const device = "/dev/ttyACM0"

// poll interval
const interval = 60 * time.Second

// ISO8601Time utility
type ISO8601Time time.Time

// ISO8601 date time format
const ISO8601 = `2006-01-02T15:04:05.000Z07:00`

// MarshalJSON interface function
func (t ISO8601Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(ISO8601))
}

// Data - the data
type Data struct {
	CO2         int64       `json:"co2"`
	Humidity    float64     `json:"humidity"`
	Temperature float64     `json:"temperature"`
	Timestamp   ISO8601Time `json:"timestamp"`
}

// initialize and prepare the device
func prepareDevice(p serial.Port, s *bufio.Scanner) error {
	fmt.Print("I: Prepare device...:")
	defer fmt.Println("")
	for _, c := range []string{"STP", "ID?", "STA"} {
		fmt.Printf(" %v", c)
		if _, err := p.Write([]byte(c + "\r\n")); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 100) // wait
		for s.Scan() {
			t := s.Text()
			if t[:2] == `OK` {
				break
			} else if t[:2] == `NG` {
				return fmt.Errorf(" command `%v` failed", c)
			}
		}
	}
	fmt.Print(" OK.")
	return nil
}

func main() {
	port, err := serial.Open(device, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	})
	if err != nil {
		fmt.Printf("E: Opening serial: %+v: %v\n", err, device)
		os.Exit(1)
	}
	defer func() { port.Write([]byte("STP\r\n")); time.Sleep(time.Millisecond * 100); port.Close() }()

	// serial reader
	port.SetReadTimeout(time.Second * 10)
	s := bufio.NewScanner(port)
	s.Split(bufio.ScanLines)

	if err := prepareDevice(port, s); err != nil {
		fmt.Println("E: " + err.Error())
		port.Close()
		os.Exit(1)
	}

	// trap SIGINT
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	// signal handler
	go func() {
		<-sigch
		port.Write([]byte("STP\r\n"))
	}()

	// serial reader channel
	r := make(chan *Data)
	// publisher channel
	p := make(chan *Data)

	// publisher
	go func() {
		for d := range p {
			b, err := json.Marshal(d)
			if err != nil {
				fmt.Print("E: " + err.Error())
				continue
			}
			fmt.Println(string(b))
		}
	}()

	// periodical dispatcher
	go func() {
		var cur *Data // current data
		p <- <-r      // send the first data
		tick := time.Tick(interval)
		for {
			select {
			case <-tick:
				if cur == nil {
					continue
				}
				p <- cur  // publish current
				cur = nil // dismiss
			case cur = <-r:
			}
		}
	}()

	// reader (main)
	re := regexp.MustCompile(`CO2=(\d+),HUM=([\d.]+),TMP=([\d.-]+)`)
	for s.Scan() {
		d := &Data{Timestamp: ISO8601Time(time.Now())}
		text := s.Text()
		m := re.FindAllStringSubmatch(text, -1)
		if len(m) > 0 {
			d.CO2, _ = strconv.ParseInt(m[0][1], 10, 64)
			d.Humidity, _ = strconv.ParseFloat(m[0][2], 64)
			d.Temperature, _ = strconv.ParseFloat(m[0][3], 64)
			d.Timestamp = ISO8601Time(time.Now())
			r <- d
		} else if text[:6] == `OK STP` {
			return // exit 0
		} else {
			fmt.Printf("E: Read unmatched string: %v", text)
		}
	}
	if s.Err() != nil {
		fmt.Print("E: " + s.Err().Error())
	}
}
