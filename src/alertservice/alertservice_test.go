package alertservice

import (
	"testing"
)

type alertResp struct {
	res   bool
	alert string
}

func TestAlertingLogic1(t *testing.T) {
	a := New()

	cpuLoad := []float64{0.1, 0.2, 0.3, 0.1, 0.8, 1.01, 1.01, 1.01, 1.01, 1.01, 1.02, 1.09, 1.11, 1.16, 1.18, 1.17, 1.11}
	expected := []alertResp{{false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {true, "High load alert: load=1.11,"}}

	for i, l := range cpuLoad {
		res, alert := a.DetectAlert(l)

		if res != expected[i].res || alert != expected[i].alert {
			t.Errorf("Error in Alertlogic: expected:%v, got:%v, %v", expected[i], res, alert)
		}
	}
}

func TestAlertingLogic2(t *testing.T) {
	a := New()

	cpuLoad := []float64{0.1, 0.2, 0.3, 0.1, 0.8, 0.91, 0.91, 0.91, 0.91, 0.91, 0.92, 0.99, 1.00, 0.90, 0.98, 0.97, 0.91}
	expected := []alertResp{{false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}}

	for i, l := range cpuLoad {
		res, alert := a.DetectAlert(l)

		if res != expected[i].res || alert != expected[i].alert {
			t.Errorf("Error in Alertlogic: expected:%v, got:%v, %v", expected[i], res, alert)
		}
	}
}

func TestAlertingLogic3(t *testing.T) {
	a := New()

	cpuLoad := []float64{0.1, 0.2, 0.3, 0.1, 0.8, 1.11, 1.31, 1.45, 1.50, 1.47, 1.44, 1.39, 1.20, 1.13, 1.18, 1.17, 1.21, 0.1, 0.1, 0.2, 0.4, 0.5, 0.6, 0.7, 0.8, 0.8, 0.2, 0.1, 0.3}
	expected := []alertResp{{false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {true, "High load alert: load=1.21,"}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {false, ""}, {true, "Alert recovered:"}}

	for i, l := range cpuLoad {
		res, alert := a.DetectAlert(l)

		if res != expected[i].res || alert != expected[i].alert {
			t.Errorf("Error in Alertlogic: expected:%v, got:%v, %v", expected[i], res, alert)
		}
	}
}
