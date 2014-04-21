package peer

import (
	"fmt"
	"log"
	"encoding/json"
	"math/rand"

	"github.com/conformal/btcjson"

	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/rep"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

const DEFAULT_MAX_TO_MAKE = 10

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

func (p *Peer) AmountPaid(minConf int, btcConf *util.BitcoindConf) (*msg.PaymentValue, error) {
	cmd, err := btcjson.NewListReceivedByAddressCmd("", minConf, false)
	if err != nil {
		return nil, fmt.Errorf("error while making cmd: %v", err.Error())
	}
	resp, err := util.SendBtcRpc(cmd, btcConf)
	ser, ok := resp.Result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error during bitcoind JSON-RPC: %v", resp)
	}
	addrs := p.readPaymentAddrs()
	addrsMap := make(map[string]bool)
	for _, addr := range addrs {
		addrsMap[addr] = true
	}
	amt := int64(0)
	for _, r := range ser {
		result := r.(map[string]interface{})
		if addrsMap[result["address"].(string)] {
			satoshis := util.B2S(result["amount"].(float64))
			fmt.Printf("addr: %v -> %v\n", result["address"], satoshis)
			amt += satoshis
		}
	}
	fmt.Printf("my addrs: %v\n", addrs)
	return &msg.PaymentValue{Amount: amt, Currency: msg.BTC}, nil
}

func (p *Peer) AmountConsumed() (*msg.PaymentValue, error) {
	return rep.PaymentValueServedToOcID(p.ID)
}


func (p *Peer) Balance(minConf int, btcConf *util.BitcoindConf) (*msg.PaymentValue, error) {
	paidPv, err := p.AmountPaid(minConf, btcConf)
	if err != nil {
		return nil, err
	}
	consumedPv, err := p.AmountConsumed()
	if err != nil {
		return nil, err
	}
	if paidPv.Currency != msg.BTC || consumedPv.Currency != msg.BTC {
		panic("TODO: support other currencies")
	}
	pv := msg.PaymentValue{
		Amount: consumedPv.Amount - paidPv.Amount,
		Currency: msg.BTC,
	}
	return &pv, nil
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

func (p *Peer) readPaymentAddrs() ([]string) {
	d := util.GetOrCreateDB(addrDBPath())
	fmt.Printf("p: %v\n", p)
	addrsSer, _ := d.Read(p.ID.String())
	if addrsSer == nil || len(addrsSer) == 0 {
		return []string{}
	} else {
		var addrs []string
		err := json.Unmarshal(addrsSer, &addrs)
		if err != nil {
			return []string{}
		}
		return addrs
	}
}

func (p *Peer) PaymentAddr(maxToMake int, btcConf *util.BitcoindConf) (string, error) {
	if maxToMake == -1 {
		// TODO(ortutay): This is a parameter for testing. See if there is a better
		// solution.
		maxToMake = DEFAULT_MAX_TO_MAKE
	}
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
