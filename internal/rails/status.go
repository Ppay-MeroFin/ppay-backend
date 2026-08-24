package rails

import "time"

type RailStatus struct {
	Name        string
	Healthy     bool
	Latency     time.Duration
	LastChecked time.Time
	Error       string
}
