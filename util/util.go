package util

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"

	"github.com/conformal/btcjson"
	"github.com/peterbourgon/diskv"
)

var _ = fmt.Println
var appDir string = "~/.decloud"

type BitcoindConf struct {
	User     string
	Password string
	Server   string
}

func LoadBitcoindConf(filename string) (*BitcoindConf, error) {
	if filename == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		candidates := []string{
			fmt.Sprintf("%s/.bitcoin/bitcoin.conf", usr.HomeDir),
			fmt.Sprintf("%s/Library/Application Support/Bitcoin/bitcoin.conf", usr.HomeDir),
			// TOOD(ortutay): Windows
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				filename = c
				break
			}
		}
	}
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	var conf BitcoindConf
	for scanner.Scan() {
		s := strings.Split(scanner.Text(), "=")
		key, value := s[0], s[1]
		switch key {
		case "rpcuser":
			conf.User = value
		case "rpcpassword":
			conf.Password = value
		case "rpcport":
			conf.Server = ":" + value
		}
	}
	if conf.User == "" || conf.Password == "" || conf.Server == "" {
		return nil, errors.New(
			fmt.Sprintf("%v missing one of rpcuser, rpcpassword, rpcport", filename))
	}
	return &conf, nil
}

func SendBtcRpc(msg btcjson.Cmd, conf *BitcoindConf) (*btcjson.Reply, error) {
	json, err := msg.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error while marshaling: %v", err.Error())
	}
	resp, err := btcjson.RpcCommand(conf.User, conf.Password, conf.Server, json)
	if err != nil {
		return nil, fmt.Errorf("error during bitcoind JSON-RPC: %v", err.Error())
	}
	return &resp, nil
}

func GetAppData(filename string) (*os.File, error) {
	filename, err := normalizeFilename(filename)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func StoreAppData(filename string, data []byte, perm os.FileMode) error {
	filename, err := normalizeFilename(filename)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}

	err = file.Chmod(perm)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func normalizeFilename(filename string) (string, error) {
	if path.IsAbs(filename) {
		return filename, nil
	} else {
		err := makeAppDir()
		if err != nil {
			return "", err
		}
		if path.Dir(filename) != "." {
			panic("TODO implement if needed")
		}
		filename = AppDir() + "/" + filename
		return filename, nil
	}
}

func AppDir() string {
	re := regexp.MustCompile("^~")
	return string(re.ReplaceAll([]byte(appDir), []byte(os.Getenv("HOME"))))
}

func SetAppDir(newAppDir string) {
	appDir = newAppDir
}

func makeAppDir() error {
	err := os.Mkdir(AppDir(), 0775)
	if os.IsExist(err) {
		return nil
	} else {
		return err
	}
}

type ByteSize int

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1000
	MB ByteSize = 1000000
	GB ByteSize = 1000000000
	TB ByteSize = 1000000000000
)

func (b ByteSize) String() string {
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2fTB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2fGB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2fMB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2fKB", float64(b)/float64(KB))
	}
	return fmt.Sprintf("%.2fB", b)
}

func (b ByteSize) Int() int {
	return int(b)
}

func (b ByteSize) Float64() float64 {
	return float64(b)
}

func ByteSizeParseString(str string) (ByteSize, error) {
	re := regexp.MustCompile("(?i)([0-9.]+) *(KB|MB|GB|TB)")
	m := re.FindStringSubmatch(str)
	if len(m) != 3 {
		return 0, fmt.Errorf("could not parse: %v", str)
	}
	r := new(big.Rat)
	_, err := fmt.Sscan(m[1], r)
	if err != nil {
		return 0, fmt.Errorf("could not parse: %v", m[1])
	}
	f, _ := r.Float64()
	unit := strings.ToUpper(m[2])
	switch unit {
	case "KB":
		return ByteSize(f * KB.Float64()), nil
	case "MB":
		return ByteSize(f * MB.Float64()), nil
	case "GB":
		return ByteSize(f * GB.Float64()), nil
	case "TB":
		return ByteSize(f * TB.Float64()), nil
	}
	return 0, fmt.Errorf("could not parse: %v", str)
}

// TODO(ortutay): for safety and convenience, we should have a single BTC type
func B2S(btc float64) int64 {
	return int64(btc * 1e8)
}

func S2B(satoshis int64) float64 {
	return float64(satoshis) / 1e8
}

func StringB2S(btc string) (int64, error) {
	r := new(big.Rat)
	_, ok := r.SetString(btc)
	if !ok {
		return 0, fmt.Errorf("could not parse: %v", btc)
	}
	fmt.Printf("r: %v %v\n", r.FloatString(10), btc)
	r.Mul(r, big.NewRat(1e8, 1))
	if !r.IsInt() {
		return 0, fmt.Errorf("more than 8 decimals: %v", btc)
	}
	return r.Num().Int64(), nil
}

func Ferr(err error) {
	if err != nil {
		panic(err)
	}
}

func GetOrCreateDB(path string) *diskv.Diskv {
	flatTransform := func(s string) []string { return []string{} }
	d := diskv.New(diskv.Options{
		BasePath:     path,
		Transform:    flatTransform,
		CacheSizeMax: 1024 * 1024,
	})
	if d == nil {
		log.Fatal("Couldn't open DB at %v", path)
	}
	return d
}
