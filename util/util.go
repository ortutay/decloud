package util

import (
	"os"
	"path"
)

func StoreAppData(filename string, data []byte, perm os.FileMode) error {
	if path.IsAbs(filename) {
	} else {
		err := makeAppDir()
		if err != nil { return err }
		if path.Dir(filename) != "." { panic("TODO implement if needed") }
		filename = appDir() + "/" + filename
	}

	f, err := os.Create(filename)
	if err != nil { return err }

	err = f.Chmod(perm)
	if err != nil { return err }

	_, err = f.Write(data)
	if err != nil { return err }

	return nil
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
