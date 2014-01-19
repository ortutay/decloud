package client

import (
	"oc/msg"
)

func NewCalcReq(queries []string) msg.OcReq {
	msg := msg.OcReq{
		NodeId: "TODO",
		Sig: "TODO",
		Nonce: "TODO",
		Service: "calc",
		Method: "calculate",
		Args: queries,
		PaymentType: "none",
		PaymentTxn: "",
		Body: []byte(""),
	}
	return msg
}
