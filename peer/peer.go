package peer

import (
	"fmt"
	"log"
	"encoding/json"
	"math/rand"

	"github.com/conformal/btcjson"

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

func (p *Peer) fetchNewBtcAddr(btcConf *util.BitcoindConf) (string, error) {
	cmd, err := btcjson.NewGetNewAddressCmd("")
	if err != nil {
		return "", fmt.Errorf("error while making cmd: %v", err.Error())
	}
	resp, err := util.SendBtcRpc(cmd, btcConf)
	addr, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("error during bitcoind JSON-RPC: %v", resp)
	}
	return addr, nil
}

func addrDBPath() string {
	return util.AppDir() + "/peer-addrs-diskv.db"
}

func (p *Peer) PaymentAddr(maxToMake int, btcConf *util.BitcoindConf) (string, error) {
	d := util.GetOrCreateDB(addrDBPath())
	addrsSer, _ := d.Read(p.ID.String())
	fmt.Printf("read addrs: %v\n", addrsSer)
	if addrsSer == nil || len(addrsSer) == 0 {
		fmt.Printf("no addrs read, making...\n")
		var addrs []string
		for i := 0; i < maxToMake; i++ {
			btcAddr, err := p.fetchNewBtcAddr(btcConf)
			if err != nil {
				log.Printf("error while generating addresses: %v\n", err)
				return "", err
			}
			addrs = append(addrs, btcAddr)
		}
		ser, err := json.Marshal(addrs)
		if err != nil {
			return "", err
		}
		err = d.Write(p.ID.String(), ser)
		if err != nil {
			return "", err
		}
		fmt.Printf("generated addresses: %v ser: %v\n", addrs, string(ser))
		addrsSer, _ = d.Read(p.ID.String())
	}
	var addrs []string
	err := json.Unmarshal(addrsSer, &addrs)
	if err != nil {
		return "", err
	}
	if addrs == nil || len(addrs) == 0 {
		panic("unexpected empty list")
	}
	return addrs[rand.Int()%len(addrs)], nil
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
