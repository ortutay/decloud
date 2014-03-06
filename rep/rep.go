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
	PaymentValue *msg.PaymentValue
	Perf         interface{} // Service specific
}

type Cursor interface {
	Next() *Record
	Reset()
}

func openOrCreate() (*sql.DB, error) {
	exists := true
	if _, err := os.Stat(sqliteDBPath()); os.IsNotExist(err) {
		exists = false
	}

	db, err := sql.Open("sqlite3", sqliteDBPath())
	if err != nil {
		return nil, fmt.Errorf("error while opening db %v: %v",
			sqliteDBPath(), err.Error())
	}

	if !exists {
		err = initTable(db)
		if err != nil {
			return nil, fmt.Errorf("error while intializing table: %v", err.Error())
		}
	}
	return db, nil
}

func Put(rec *Record) (int64, error) {
	db, err := openOrCreate()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	cmd := recordToSqlInsert(rec)
	fmt.Printf("insert %v\n", cmd)
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

func SuccessRate(sel *Record) (float64, error) {
	counter := make(map[string]float64)
	counter["total"] = float64(0)
	counter["success"] = float64(0)
	reducer := func(rec *Record) {
		c := counter
		if rec.Status == SUCCESS || rec.Status == FAILURE {
			c["total"]++;
		}
		if rec.Status == SUCCESS {
			c["success"]++;
		}
	}
	err := Reduce(sel, reducer)
	if err != nil {
		return 0, err
	}
	if (counter["success"]) == 0 {
		return -1, nil
	}
	return counter["success"]/counter["total"], nil
}

func Reduce(sel *Record, reduceFn func(rec *Record)) error {
	db, err := openOrCreate()
	if err != nil {
		return err
	}
	defer db.Close()

	query := selectLikeRecord(sel)
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("error while querying %v: %v", query, err.Error())
	}
	for rows.Next() {
		var service, method, ocID, status, pvType, pvCurr, timestamp, perfHex []byte
		var pvAmt int64
		err := rows.Scan(
			&service, &method, &timestamp, &ocID, &status, &pvType, &pvAmt, &pvCurr,
			&perfHex)
		var rec Record

		if len(service) != 0 {
			rec.Service = string(service)
		}
		if len(method) != 0 {
			rec.Method = string(method)
		}
		if len(ocID) != 0 {
			rec.OcID = msg.OcID(ocID)
		}
		if len(status) != 0 {
			rec.Status = Status(status)
		}
		if len(pvType) != 0 {
			rec.PaymentType = msg.PaymentType(pvType)
		}
		if len(pvCurr) != 0 {
			pv := msg.PaymentValue{
				Amount: pvAmt,
				Currency: msg.Currency(pvCurr),
			}
			rec.PaymentValue = &pv
		}
		if len(perfHex) != 0 {
			panic("TODO: implement perf decoding")
		}
		if err != nil {
			return fmt.Errorf("error scanning with %v: %v", query, err.Error())
		}
		reduceFn(&rec)
	}
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

func qesc(s string) string {
	return strings.Replace(s, "\"", "\\\"", -1)
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
	pvAmt := int64(0)
	pvCurr := ""
	if rec.PaymentValue != nil {
		pvAmt = rec.PaymentValue.Amount
		pvCurr = rec.PaymentValue.Currency.String()
	}
	return fmt.Sprintf(`
INSERT INTO rep(service, method, timestamp, ocID, status, paymentType, paymentValueAmount, paymentValueCurrency, perf)
VALUES ("%s", "%s", "%d", "%s", "%s", "%s", "%d", "%s", x'%s');`, 
		qesc(rec.Service), qesc(rec.Method), rec.Timestamp, rec.OcID.String(),
		rec.Status.String(), rec.PaymentType.String(), pvAmt, pvCurr, perfHex)
}

func selectLikeRecord(rec *Record) string {
	var buf bytes.Buffer
	buf.WriteString("SELECT service, method, timestamp, ocID, status, paymentType, paymentValueAmount, paymentValueCurrency, perf FROM rep WHERE 1")
	// buf.WriteString("SELECT status FROM rep WHERE 1")
	if rec.Service != "" {
		buf.WriteString(fmt.Sprintf(` AND service = "%s"`, qesc(rec.Service)))
	}
	if rec.Method != "" {
		buf.WriteString(fmt.Sprintf(` AND method = "%s"`, qesc(rec.Method)))
	}
	if rec.Timestamp != 0 {
		buf.WriteString(fmt.Sprintf(` AND timestamp = %d`, rec.Timestamp))
	}
	if rec.OcID.String() != "" {
		buf.WriteString(fmt.Sprintf(` AND ocID = "%s"`, rec.OcID.String()))
	}
	if rec.Status.String() != "" {
		buf.WriteString(fmt.Sprintf(` AND status = "%s"`, rec.Status.String()))
	}
	if rec.PaymentType.String() != "" {
		buf.WriteString(fmt.Sprintf(` AND paymentType = "%s"`,
			rec.PaymentType.String()))
	}
	if rec.PaymentValue != nil {
		buf.WriteString(fmt.Sprintf(` AND paymentValueAmount = %d AND paymentValueCurreny = "%s"`,
			rec.PaymentValue.Amount, rec.PaymentValue.Currency.String()))
	}
	return buf.String()
}
