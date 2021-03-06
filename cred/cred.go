package cred

// TODO(ortutay): different name?

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"sort"
	"strings"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

const (
	PRIVATE_KEY_FILENAME   = "nodeid-priv"
	OC_ID_PREFIX           = 'c' // "c" for open"c"loud
	NODE_ID_RAND_NUM_BYTES = 256
	SIG_RAND_NUM_BYTES     = 256
)

type Signer interface {
	SignOcReq(req *msg.OcReq) error
}

type Cred struct {
	OcCred  OcCred
	BtcConf *util.BitcoindConf
	Coins   []BtcCred
}

func (c *Cred) SignOcReq(req *msg.OcReq, bConf *util.BitcoindConf) error {
	err := c.OcCred.SignOcReq(req)
	if err != nil {
		return fmt.Errorf("error while signing: %v", err.Error())
	}

	for _, coin := range c.Coins {
		err := coin.SignOcReq(req, bConf)
		if err != nil {
			return fmt.Errorf("error while signing: %v", err.Error())
		}
	}
	return nil
}

type OcCred struct {
	Priv *ecdsa.PrivateKey // TODO(ortutay): make private field?
}

func NewOcCred() *OcCred {
	randBytes := make([]byte, NODE_ID_RAND_NUM_BYTES)
	_, err := rand.Read(randBytes)
	if err != nil {
		log.Fatal(err)
	}

	curve := elliptic.P256()
	priv, err := ecdsa.GenerateKey(curve, bytes.NewReader(randBytes))
	if err != nil {
		log.Fatal(err)
	}

	ocCred := OcCred{
		Priv: priv,
	}
	return &ocCred
}

func NewOcCredLoadOrCreate(filename string) (*OcCred, error) {
	if filename == "" {
		filename = PRIVATE_KEY_FILENAME
	}
	file, _ := util.GetAppData(filename)
	if file != nil {
		return NewOcCredLoadFromFile(filename)
	} else {
		ocCred := NewOcCred()
		err := ocCred.StorePrivateKey("")
		if err != nil {
			return nil, err
		}
		return ocCred, nil
	}
}

func NewOcCredLoadFromFile(filename string) (*OcCred, error) {
	if filename == "" {
		filename = PRIVATE_KEY_FILENAME
	}
	file, err := util.GetAppData(filename)
	if err != nil {
		return nil, fmt.Errorf("error getting app data: %v", err.Error())
	}
	var d big.Int
	fmt.Fscanf(file, "%x\n", &d)
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(d.Bytes())
	priv := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: &d,
	}

	ocCred := OcCred{
		Priv: &priv,
	}

	return &ocCred, nil
}

func getReqSigDataHash(req *msg.OcReq) ([]byte, error) {
	var buf bytes.Buffer
	req.WriteSignablePortion(&buf)

	hasher := sha256.New()
	_, err := hasher.Write(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error while hashing: %v", err.Error())
	}

	h := hasher.Sum([]byte{})
	return h, nil
}

func (o *OcCred) ID() msg.OcID {
	// TODO(ortutay): compress pub key
	return msg.OcID(fmt.Sprintf("%c%x,%x",
		OC_ID_PREFIX, o.Priv.PublicKey.X, o.Priv.PublicKey.Y))
}

func (o *OcCred) StorePrivateKey(filename string) error {
	if filename == "" {
		filename = PRIVATE_KEY_FILENAME
	}
	d := fmt.Sprintf("%x\n", o.Priv.D)
	err := util.StoreAppData(filename, []byte(d), 0600)
	if err != nil {
		return fmt.Errorf("error storing app data: %v", err.Error())
	}
	return nil
}

func (o *OcCred) SignOcReq(req *msg.OcReq) error {
	h, err := getReqSigDataHash(req)
	if err != nil {
		return err
	}

	randBytes := make([]byte, SIG_RAND_NUM_BYTES)
	_, err = rand.Read(randBytes)
	if err != nil {
		return errors.New("error generating random bytes")
	}

	r, s, err := ecdsa.Sign(bytes.NewReader(randBytes), o.Priv, h)
	if err != nil {
		return fmt.Errorf("error during ECDSA signature: %v", err.Error())
	}
	req.ID = o.ID()
	req.Sig = fmt.Sprintf("%x,%x", r, s)

	return nil
}

func VerifyOcReqSig(req *msg.OcReq, conf *util.BitcoindConf) (bool, error) {
	h, err := getReqSigDataHash(req)
	if err != nil {
		return false, err
	}

	if req.ID != "" {
		ok := verifyOcSig(h, req.ID, req.Sig)
		if !ok {
			return false, nil
		}
	}

	for i, _ := range req.Coins {
		coin := req.Coins[i]
		coinSig := req.CoinSigs[i]
		fmt.Printf("verify %v %v\n", coin, coinSig)

		switch coin[0] {
		case '1', 'm', 'n':
			if conf == nil {
				return false, errors.New("need bitcoind conf to verify btc cred")
			}
			ok, err := verifyBtcSig(h, coin, coinSig, conf)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		default:
			return false, errors.New(
				fmt.Sprintf("unexpected id prefix: %c", coin[0]))
		}
	}

	return true, nil
}

func verifyOcSig(reqHash []byte, ocID msg.OcID, sig string) bool {
	ocCredReader := strings.NewReader(ocID.String())
	var x, y, r, s big.Int
	n, err := fmt.Fscanf(ocCredReader, string(OC_ID_PREFIX)+"%x,%x", &x, &y)
	if err != nil {
		return false
	}
	n, err = ocCredReader.Read(make([]byte, 1))
	if n != 0 || err != io.EOF {
		return false
	}

	sigReader := strings.NewReader(sig)
	n, err = fmt.Fscanf(sigReader, "%x,%x", &r, &s)
	if err != nil {
		return false
	}
	n, err = sigReader.Read(make([]byte, 1))
	if n != 0 || err != io.EOF {
		return false
	}

	curve := elliptic.P256()
	pub := ecdsa.PublicKey{
		Curve: curve,
		X:     &x,
		Y:     &y,
	}
	return ecdsa.Verify(&pub, reqHash, &r, &s)
}

type BtcCred struct {
	Addr string
}

type addressBalance struct {
	Address string
	Amount  int64
}

type byAmount []addressBalance

func (a byAmount) Len() int           { return len(a) }
func (a byAmount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAmount) Less(i, j int) bool { return a[i].Amount < a[j].Amount }

func inputsInRange(unspent *[]addressBalance, min, max int64, iter int, right int) (*[]BtcCred, error) {
	// Assume list is already sorted
	if iter == 0 {
		return nil, fmt.Errorf("couldn't find matching inputs")
	}
	for i := right; i >= 0; i-- {
		u := (*unspent)[i]
		bc := BtcCred{Addr: u.Address}
		amt := u.Amount
		if amt > max {
			continue
		}
		if iter == 1 {
			if amt >= min {
				return &[]BtcCred{bc}, nil
			}
		} else {
			r, _ := inputsInRange(unspent, min-amt, max-amt, iter-1, i-1)
			if r != nil {
				result := append(*r, bc)
				return &result, nil
			}
		}
	}
	return nil, fmt.Errorf("couldn't find matching inputs")
}

func GetBtcCredInRange(min, max int64, conf *util.BitcoindConf) (*[]BtcCred, error) {
	cmd, err := btcjson.NewListUnspentCmd("")
	if err != nil {
		return nil, fmt.Errorf("error while making cmd: %v", err.Error())
	}
	resp, err := util.SendBtcRpc(cmd, conf)
	if err != nil {
		return nil, fmt.Errorf("error while making cmd: %v", err.Error())
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("error during bitcoind JSON-RPC: %v", resp.Error)
	}
	addrs := make(map[string]*addressBalance)
	unspent := resp.Result.([]btcjson.ListUnSpentResult)
	for _, u := range unspent {
		if _, ok := addrs[u.Address]; !ok {
			addrs[u.Address] = &addressBalance{
				Address: u.Address,
				Amount:  0,
			}
		}
		ab := addrs[u.Address]
		ab.Amount += util.B2S(u.Amount)
	}
	addrsList := make([]addressBalance, len(addrs))
	i := 0
	for _, v := range addrs {
		addrsList[i] = *v
		i++
	}
	sort.Sort(byAmount(addrsList))
	var use *[]BtcCred
	for iter := 1; iter <= 5; iter++ {
		use, err = inputsInRange(&addrsList, min, max, iter, len(addrsList)-1)
		if use != nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return use, nil
}

func (bc *BtcCred) SignOcReq(req *msg.OcReq, conf *util.BitcoindConf) error {
	h, err := getReqSigDataHash(req)
	if err != nil {
		return err
	}
	hb64 := base64.StdEncoding.EncodeToString(h)

	msg, err := btcjson.NewSignMessageCmd(nil, bc.Addr, hb64)
	if err != nil {
		return fmt.Errorf("error while making cmd: %v", err.Error())
	}
	json, err := msg.MarshalJSON()
	if err != nil {
		return fmt.Errorf("error while marshaling: %v", err.Error())
	}
	resp, err := btcjson.RpcCommand(conf.User, conf.Password, conf.Server, json)
	if err != nil {
		return fmt.Errorf("error while making bitcoind JSON-RPC: %v", err.Error())
	}
	sig, ok := resp.Result.(string)
	if !ok {
		return errors.New("error during bitcoind JSON-RPC")
	}

	req.Coins = append(req.Coins, fmt.Sprintf(bc.Addr))
	req.CoinSigs = append(req.CoinSigs, fmt.Sprintf(sig))

	return nil
}

func verifyBtcSig(reqHash []byte, addr string, sig string, conf *util.BitcoindConf) (bool, error) {
	hb64 := base64.StdEncoding.EncodeToString(reqHash)

	msg, err := btcjson.NewVerifyMessageCmd(nil, addr, sig, hb64)
	if err != nil {
		return false, fmt.Errorf("error while making cmd: %v", err.Error())
	}
	json, err := msg.MarshalJSON()
	if err != nil {
		return false, fmt.Errorf("error while marshaling: %v", err.Error())
	}
	resp, err := btcjson.RpcCommand(conf.User, conf.Password, conf.Server, json)
	if err != nil {
		return false, fmt.Errorf(
			"error while making bitcoind JSON-RPC: %v", err.Error())
	}
	if resp.Error != nil {
		return false, resp.Error
	}
	verifyResult, ok := resp.Result.(bool)
	if !ok {
		return false, fmt.Errorf("error during bitcoind JSON-RPC: %v", err.Error())
	}

	return verifyResult, nil
}
