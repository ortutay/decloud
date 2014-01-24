package util

import (
	"os"
	"path"
)

func GetAppData(filename string) (*os.File, error) {
	filename, err := normalizeFilename(filename)
	if err != nil { return nil, err }

	file, err := os.Open(filename)
	if err != nil { return nil, err }

	return file, nil
}

func StoreAppData(filename string, data []byte, perm os.FileMode) error {
	filename, err := normalizeFilename(filename)
	if err != nil { return err }

	file, err := os.Create(filename)
	if err != nil { return err }

	err = file.Chmod(perm)
	if err != nil { return err }

	_, err = file.Write(data)
	if err != nil { return err }

	return nil
}

func normalizeFilename(filename string) (string, error) {
	if path.IsAbs(filename) {
		return filename, nil
	} else {
		err := makeAppDir()
		if err != nil { return "", err }
		if path.Dir(filename) != "." { panic("TODO implement if needed") }
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
