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

// "Header" or "Record", depending on how extensible this should be
type Header struct {
	Service string
	Method string
	Timestamp int
	OcID cred.OcID
	Status Status
	PaymentType msg.PaymentType
	PaymentValue msg.PaymentValue
}

// TODO(ortutay): evaluate if this is correct structure
type Selection struct {
	// selector parameters...
}

func (s *Selection) SuccessRate() float64 {
}

// TODO(ortutay): if we omit the concept of "performance," the following is not
// needed...
type Cursor interface {
	Next() interface{}
}

type AvgPerfReducer interface {
	// ...
}

func (s *Selection) AvgPerf() interface{} {
}

