package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

// nameRegex matches the name key of a development container configuration
// file. Its value is matched without greediness so that the keys coming after
// it are left untouched, and its trailing comma is not part of the match.
var nameRegex = regexp.MustCompile(`"name"[ \t]*:[ \t]*"[^"]*"`)

var ErrNameNotFound = errors.New("name not found in devcontainer.json")

// devcontainerFileName is the name of the development container
// configuration files handled by devtainr, wherever they are in the tree.
const devcontainerFileName = "devcontainer.json"

// ChangeName replaces the name of a single development container
// configuration file. Only the first name found is replaced, so that a nested
// one is left untouched. It returns ErrNameNotFound if the file holds no name.
func ChangeName(path, name string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	found := nameRegex.FindIndex(b)
	if found == nil {
		return fmt.Errorf("replacing name: %w", ErrNameNotFound)
	}

	// the name is written by hand instead of using the substitution methods
	// of the regexp package so that a name holding a $ is not expanded
	replaced := make([]byte, 0, len(b))
	replaced = append(replaced, b[:found[0]]...)
	replaced = append(replaced, `"name": "`...)
	replaced = append(replaced, name...)
	replaced = append(replaced, '"')
	replaced = append(replaced, b[found[1]:]...)
	b = replaced

	// the file exists since it has just been read, hence it is opened
	// without creation and its permissions are left untouched
	const filePerms = 0o666
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, filePerms)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}

	// the file is always closed, and a failing close is reported along with
	// the failing write when both happen
	_, writeErr := file.Write(b)
	closeErr := file.Close()
	if writeErr != nil {
		return errors.Join(fmt.Errorf("writing file: %w", writeErr), closeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing file: %w", closeErr)
	}

	return nil
}

// DevcontainerFiles returns the paths, relative to rootDirectory and in walk
// order, of every development container configuration file found recursively
// under rootDirectory. Symbolic links are ignored, hence only regular files
// are written back by [ChangeName].
func DevcontainerFiles(rootDirectory string) (found []string, err error) {
	err = filepath.WalkDir(rootDirectory, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			// wrapped by the filepath.WalkDir call below
			return err
		}
		if dirEntry.IsDir() || !dirEntry.Type().IsRegular() ||
			dirEntry.Name() != devcontainerFileName {
			return nil
		}

		relativePath, err := filepath.Rel(rootDirectory, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		found = append(found, filepath.ToSlash(relativePath))

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return found, nil
}
