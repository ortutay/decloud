package peer

import (
	"fmt"

	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

type Peer struct {
	ID    msg.OcID
	Coins []msg.BtcAddr
}

type PeerError string

const (
	UNEXPECTED        PeerError = "unexpected"
	INVALID_SIGNATURE PeerError = "invalid-signature"
	COIN_REUSE        PeerError = "coin-reuse"
	// TODO(ortutay): think about how to structure this...
)

func (pe PeerError) Error() string {
	return string(pe)
}

func NewPeerFromReq(req *msg.OcReq, btcConf *util.BitcoindConf) (*Peer, error) {
	ok, err := cred.VerifyOcReqSig(req, btcConf)
	if err != nil {
		fmt.Printf("error while verifying sig: %v\n", err)
		return nil, UNEXPECTED
	}
	if !ok {
		return nil, INVALID_SIGNATURE
	}
	coins := make([]msg.BtcAddr, 0)
	for _, coin := range req.Coins {
		ocID, err := ocIDForCoin(coin)
		if err != nil {
			fmt.Printf("error while generating ocID for coin: %v\n", err)
			return nil, UNEXPECTED
		}
		fmt.Printf("got oc ID for coin: %v\n", ocID)
		if ocID != nil && *ocID != req.ID {
			return nil, COIN_REUSE
		}
		err = setOcIDForCoin(coin, &req.ID)
		if err != nil {
			fmt.Printf("error while storing ocID for coin: %v\n", err)
			return nil, UNEXPECTED
		}
		coins = append(coins, msg.BtcAddr(coin))
	}
	return &Peer{ID: req.ID, Coins: coins}, nil
}

func (p *Peer) BtcBalance(minConf int) (int64, error) {
	return 0, nil
}

func peerDBPath() string {
	return util.AppDir() + "/peer-diskv.db"
}

func ocIDForCoin(coin string) (*msg.OcID, error) {
	fmt.Printf("get oc ID for coin: %v\n", coin)
	d := util.GetOrCreateDB(peerDBPath())
	v, _ := d.Read(coin)
	if v == nil || len(v) == 0 {
		return nil, nil
	}
	id := msg.OcID(v)
	return &id, nil
}

func setOcIDForCoin(coin string, ocID *msg.OcID) error {
	fmt.Printf("set oc ID for coin %v\n", coin)
	d := util.GetOrCreateDB(peerDBPath())
	err := d.Write(coin, []byte(ocID.String()))
	util.Ferr(err)

	return nil
}
