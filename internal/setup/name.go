package setup

import (
	"errors"
	"io/ioutil"
	"os"
	"regexp"
)

var nameRegex = regexp.MustCompile(`"name"[ |\t]*:[ |\t]*".*"[ |\t]*,`)

var ErrNameNotFound = errors.New("name not found in devcontainer.json")

func ChangeName(filepath, name string) error {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	replacement := []byte(`"name": "` + name + `",`)

	found := nameRegex.Find(b)
	if len(found) == 0 {
		_ = file.Close()
		return ErrNameNotFound
	}

	b = nameRegex.ReplaceAll(b, replacement)

	file, err = os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	_, err = file.Write(b)
	if err != nil {
		_ = file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}
