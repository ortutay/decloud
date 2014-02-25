package rep

import (
	"cred"
	"msg"
)

type Status string

const (
	PENDING Status = "pending"
	// TODO(ortutay): additional statuses
)

type Record struct {
	Service string
	Method string // Is "Method" the appropriate field?
	Timestamp int
	OcID cred.OcID
	Status Status
	PaymentType msg.PaymentType
	PaymentValue msg.PaymentValue
	Perf interface{} // Interface specific
}

// TODO(ortutay): may want a cursor to represent a selection

func Put(rec Record) error {
	return nil
}

func Count(selector Record) int, error {
	return nil
}

func Reduce(selector Record, reducer func(result interface{}, Record)) error {
	return nil
}
