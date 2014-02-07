package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
)

var _ = fmt.Println

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
		filename = fmt.Sprintf("%s/.bitcoin/bitcoin.conf", usr.HomeDir)
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

func GetAppData(filename string) (*os.File, error) {
	filename, err := normalizeFilename(filename)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filename)
	defer file.Close()
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
		filename = appDir() + "/" + filename
		return filename, nil
	}
}

func appDir() string {
	return os.Getenv("HOME") + "/.oc"
}

func makeAppDir() error {
	err := os.Mkdir(appDir(), 0775)
	if os.IsExist(err) {
		return nil
	} else {
		return err
	}
}
