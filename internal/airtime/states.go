package airtime

import "fmt"

var allowedTransitions = map[TopupStatus]map[TopupStatus]bool{
	StatusPending: {
		StatusProcessing: true,
		StatusFailed:     true,
	},
	StatusProcessing: {
		StatusSuccess: true,
		StatusFailed:  true,
	},
	StatusSuccess: {},
	StatusFailed:  {},
}

func CanTransition(from, to TopupStatus) bool {
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}

	return next[to]
}

func ValidateTransition(from, to TopupStatus) error {
	if from == to {
		return nil
	}

	if !CanTransition(from, to) {
		return fmt.Errorf("invalid airtime status transition: %s -> %s", from, to)
	}

	return nil
}
