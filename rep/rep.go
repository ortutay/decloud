package rep

import (
	"encoding/gob"
	"encoding/hex"
	"bytes"
	"strings"
	"log"
	"os"
	"fmt"
	"database/sql"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	_ "github.com/mattn/go-sqlite3"
)

type Status string

func (s Status) String() string {
	return string(s)
}

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

func Put(rec *Record) (int64, error) {
	exists := true
	if _, err := os.Stat(sqliteDBPath()); os.IsNotExist(err) {
		exists = false
	}

	db, err := sql.Open("sqlite3", sqliteDBPath())
	if err != nil {
		return 0, fmt.Errorf("error while opening db %v: %v",
			sqliteDBPath(), err.Error())
	}
	defer db.Close()

	if !exists {
		err = initTable(db)
		if err != nil {
			return 0, fmt.Errorf("error while intializing table: %v", err.Error())
		}
	}

	cmd := recordToSqlInsert(rec)
	result, err := db.Exec(cmd)
	if err != nil {
		return 0, fmt.Errorf("error while trying to insert %v: %v",
			cmd, err.Error())
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	return id, nil
}

func Reduce(selector Record, reduceFn func(matches Cursor) interface{}) error {
	return nil
}

func sqliteDBPath() string {
	return util.AppDir() + "/rep-sqlite.db"
}

func initTable(db *sql.DB) error {
	sql := `
CREATE TABLE rep (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  service TEXT,
  method TEXT,
  timestamp INTEGER,
  ocID TEXT,
  status TEXT,
  paymentType TEXT,
  paymentValueAmount INTEGER,
  paymentValueCurrency TEXT,
  perf BINARY
)`
	_, err := db.Exec(sql)
	return err
}

func recordFromSqlRow() *Record {
	return nil
}

func recordToSqlInsert(rec *Record) string {
	var perfHex string
	if rec.Perf != nil {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(rec.Perf)
		if err != nil {
			log.Fatal(err)
		}
		perfHex = hex.EncodeToString(buf.Bytes())
	}
	return fmt.Sprintf(`
INSERT INTO rep(service, method, timestamp, ocID, status, paymentType, paymentValueAmount, paymentValueCurrency, perf)
VALUES ("%s", "%s", "%d", "%s", "%s", "%s", "%d", "%s", x'%s');`, 
		qesc(rec.Service), qesc(rec.Method), rec.Timestamp, rec.OcID.String(),
		rec.Status.String(), rec.PaymentType.String(), rec.PaymentValue.Amount,
		rec.PaymentValue.Currency.String(), perfHex)
}

func qesc(s string) string {
	return strings.Replace(s, "\"", "\\\"", -1)
}
