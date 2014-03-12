package peer

import (
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/cred"
	"code.google.com/p/leveldb-go/leveldb/db"
	"code.google.com/p/leveldb-go/leveldb/table"
)

type Peer struct {
	OcID msg.OcID
	Coins msg.BtcAddr
}

type PeerError string

const (
	UNEXPECTED PeerError = "unexpected"
	INVALID_SIGNATURE PeerError = "invalid-signature"
	COIN_REUSE PeerError = "coin-reuse"
	// TODO(ortutay): think about how to structure this...
)

func (pe PeerError) Error() string {
	return string(pe)
}

func NewPeerFromReq(req *msg.OcReq, btcConf *util.BitcoindConf) (*Peer, error) {
	ok, err := cred.VerifyOcReqSig(req, btcConf)
	if err != nil {
		return nil, UNEXPECTED
	}
	if !ok {
		return nil, INVALID_SIGNATURE
	}
	for _, coin := range req.Coins {
		ocID, err := ocIDForCoin(coin)
		if err != nil && err != db.ErrNotFound {
			return nil, UNEXPECTED
		}
		if ocID != nil && *ocID != req.ID {
			return nil, COIN_REUSE
		}
		err = setOcIDForCoin(coin, ocID)
		if err != nil {
			return nil, UNEXPECTED
		}
	}
	return nil, nil
}

func (p *Peer) BtcBalance(minConf int) (int64, error) {
	return 0, nil
}

func levelDBPath() string {
	return util.AppDir() + "/peer-leveldb.db"
}

var DBFS = db.DefaultFileSystem
func getOrCreateDB() (db.File, error) {
	_, err := DBFS.Stat(levelDBPath())
	if err == nil {
		conn, err := DBFS.Open(levelDBPath())
		if err != nil {
			return nil, err
		} else {
			return conn, nil
		}
	} else {
		conn, err := DBFS.Create(levelDBPath())
		if err != nil {
			return nil, err
		} else {
			return conn, nil
		}
	}
}

func ocIDForCoin(coin string) (*msg.OcID, error) {
	_, err := DBFS.Stat(levelDBPath())
	if err != nil {
		return nil, db.ErrNotFound
	}

	conn, err := getOrCreateDB()
	if err != nil {
		return nil, err
	}
	r := table.NewReader(conn, nil)
	defer r.Close()
	v, err := r.Get([]byte(coin), nil)
	if err != nil {
		return nil, err
	}
	id := msg.OcID(v)
	return &id, nil
}

func setOcIDForCoin(coin string, ocID *msg.OcID) error {
	conn, err := getOrCreateDB()
	if err != nil {
		return err
	}
	w := table.NewWriter(conn, nil)
	defer w.Close()
	err = w.Set([]byte(coin), []byte(*ocID), nil)
	return err
}
