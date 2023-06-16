package main

import (
	"bufio"
	"fmt"
	"go.bug.st/serial"
	"regexp"
	"strconv"
	"time"
)

type message struct {
	co2         int64
	humidity    float64
	temperature float64
}

func openSerialPort(portName string) (serial.Port, error) {
	return serial.Open(portName, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	})
}

func startDevice(p serial.Port) (*bufio.Scanner, error) {
	if err := p.SetReadTimeout(10 * time.Second); err != nil {
		return nil, err
	}

	s := bufio.NewScanner(p)

	for _, c := range []string{"STP", "STA"} {
		if err := sendCommand(p, s, c); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func stopDevice(p serial.Port, s *bufio.Scanner) error {
	return sendCommand(p, s, "STP")
}

func sendCommand(p serial.Port, s *bufio.Scanner, command string) error {
	if _, err := p.Write([]byte(command + "\r\n")); err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	for s.Scan() {
		t := s.Text()
		if t[:2] == "OK" {
			break
		} else if t[:2] == "NG" {
			return fmt.Errorf("failed to execute command `%v`", command)
		}
	}

	return nil
}

var messageRegexp = regexp.MustCompile(`CO2=(\d+),HUM=([\d.]+),TMP=([\d.-]+)`)

func parseMessage(text string) (*message, error) {
	m := messageRegexp.FindStringSubmatch(text)
	if m == nil {
		return nil, fmt.Errorf("message does not match expected pattern")
	}

	co2, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil, err
	}
	humidity, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return nil, err
	}
	temperature, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return nil, err
	}

	return &message{
		co2:         co2,
		humidity:    humidity,
		temperature: temperature,
	}, nil
}
