package params

import (
	"errors"
	"fmt"
)

var (
	ErrDevIsNotValid  = errors.New("dev is not valid")
	ErrPathIsNotValid = errors.New("path is not valid")
)

func GetRepository(dev string) (repository string, err error) {
	switch dev {
	case "go":
		return "qdm12/godevcontainer", nil
	case "react":
		return "qdm12/reactdevcontainer", nil
	case "rust":
		return "qdm12/rustdevcontainer", nil
	case "node":
		return "qdm12/nodedevcontainer", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrDevIsNotValid, dev)
	}
}
