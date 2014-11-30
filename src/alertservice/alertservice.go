package alertservice

import (
	"fmt"
)

type alarm_state int

const threshold = 1.0
const timeout = (2 * 60) / 10 // == 2 min

const (
	Init alarm_state = iota
	Recovered
	High_load
)

type AlertService struct {
	currentState alarm_state
	counter      int
}

func New() *AlertService {
	return &AlertService{currentState: Init, counter: timeout}
}

func (a *AlertService) DetectAlert(value float64) (bool, string) {
	alert := false
	msg := ""

	switch a.currentState {
	case Init, Recovered:
		if value > threshold {
			a.counter--
		} else {
			a.counter = timeout
		}

		if a.counter == 0 {
			a.counter = timeout
			a.currentState = High_load
			alert = true
			msg = fmt.Sprintf("High load alert: load=%v,", value)
		}

	case High_load:
		if value <= threshold {
			a.counter--
		} else {
			a.counter = timeout
		}

		if a.counter == 0 {
			a.counter = timeout
			a.currentState = Recovered
			alert = true

			msg = fmt.Sprintf("Alert recovered:")
		}
	}

	return alert, msg
}
