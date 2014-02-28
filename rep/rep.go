package rep

import (
	"fmt"
	// "database/sql"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	// _ "github.com/mattn/go-sqlite3"
	"code.google.com/p/leveldb-go/leveldb/db"
	"code.google.com/p/leveldb-go/leveldb/table"
)

type Status string

const (
	PENDING Status = "pending"
	SUCCESS = "success"
	FAILURE = "failure"
	// TODO(ortutay): additional statuses
)

type Record struct {
	Service      string
	Method       string // Is "Method" the appropriate field?
	Timestamp    int
	OcID         msg.OcID
	Status       Status
	PaymentType  msg.PaymentType
	PaymentValue msg.PaymentValue
	Perf         interface{} // Service specific
}

type Cursor interface {
	Next() *Record
	Reset()
}

var DBFS = db.DefaultFileSystem

func Put(rec Record) error {
	fmt.Printf("put: %v\n", rec)
	var dbf db.File
	dbf, err := DBFS.Open(levelDbPath())
	if err != nil {
		dbf, err = DBFS.Create(levelDbPath())
		if err != nil {
			return fmt.Errorf("could not open or create db %v: %v",
				levelDbPath(), err.Error())
		}
	}
	w := table.NewWriter(dbf, nil)
	defer w.Close()
	err = w.Set([]byte("1"), []byte("red"), nil)
	return nil
}

func Reduce(selector Record, reduceFn func(matches Cursor) interface{}) error {
	return nil
}

func levelDbPath() string {
	return util.AppDir() + "/rep-leveldb.db"
}
