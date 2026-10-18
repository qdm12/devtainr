package setup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrSubDirectoryNotFound = errors.New("sub-directory not found in archive")
	ErrSubDirectoryEmpty    = errors.New("sub-directory is empty")
	ErrArchiveNotGzip       = errors.New("archive is not gzip compressed")
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
)

// archiveURL returns the URL of the tar.gz archive of the default branch of
// a GitHub repository, in the owner/name form.
func archiveURL(repository string) string {
	return "https://codeload.github.com/" + repository + "/tar.gz/HEAD"
}

// Entry is a file or a directory written inside the destination directory.
type Entry struct {
	// Path is the slash separated path, relative to the destination directory.
	Path string
	// IsDir is true when the entry is a directory.
	IsDir bool
}

// DownloadFiles downloads the tar.gz archive of the default branch of the
// GitHub repository (owner/name) and writes every file and directory found
// under subDirectory in it, keeping the same tree structure. It returns the
// entries written, in the order they appear in the archive.
func DownloadFiles(ctx context.Context, client *http.Client,
	repository, subDirectory, destination string,
) (entries []Entry, err error) {
	url := archiveURL(repository)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("requesting archive: %w", err)
	}
	defer func() {
		err = errors.Join(err, response.Body.Close())
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d from %s", ErrUnexpectedStatusCode, response.StatusCode, url)
	}

	return extractFiles(response.Body, subDirectory, destination)
}

// extractFiles writes, under destination, every file and directory found
// under subDirectory in the tar.gz archive read from r. The repository
// directory wrapping the content of GitHub archives is skipped, and any entry
// escaping subDirectory is ignored.
func extractFiles(archive io.Reader, subDirectory, destination string) (entries []Entry, err error) {
	subDirectory, err = cleanSubDirectory(subDirectory)
	if err != nil {
		// error already wrapped by cleanSubDirectory
		return nil, err
	}

	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArchiveNotGzip, err)
	}
	defer func() {
		err = errors.Join(err, gzipReader.Close())
	}()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}

		relativePath, inside := entryRelativePath(header.Name, subDirectory)
		if !inside {
			continue
		}

		written, err := writeEntry(destination, relativePath, header, tarReader)
		if err != nil {
			return nil, err
		}
		if !written {
			continue
		}

		entries = append(entries, Entry{Path: relativePath, IsDir: header.Typeflag == tar.TypeDir})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: no file nor directory found in archive", ErrSubDirectoryNotFound)
	}

	return entries, nil
}

// writeEntry writes a single entry of the archive under destination, and
// says if it did. The sub directory itself, as well as symlinks, devices,
// fifos and friends that have no place in a development container
// configuration directory, are skipped.
func writeEntry(destination, relativePath string,
	header *tar.Header, reader io.Reader,
) (written bool, err error) {
	if relativePath == "" {
		return false, nil
	}

	switch header.Typeflag {
	case tar.TypeDir:
		directoryPath := filepath.Join(destination, filepath.FromSlash(relativePath))
		// the owner needs to be able to create files inside of it afterwards,
		// whatever the permissions stored in the archive say
		const ownerDirectoryPerms = 0o700
		perm := header.FileInfo().Mode().Perm() | ownerDirectoryPerms
		err = os.MkdirAll(directoryPath, perm)
		if err != nil {
			return false, fmt.Errorf("creating directory: %w", err)
		}
	case tar.TypeReg:
		perm := header.FileInfo().Mode().Perm()
		err = writeFile(destination, relativePath, reader, perm)
		if err != nil {
			return false, err
		}
	default:
		return false, nil
	}

	return true, nil
}

// relativePath is cleaned by entryRelativePath: it holds no ".." and no
// leading separator, hence it cannot escape destination.
func writeFile(destination, relativePath string, r io.Reader, perm os.FileMode) error {
	path := filepath.Join(destination, filepath.FromSlash(relativePath))
	const parentDirectoriesPerms = 0o777
	err := os.MkdirAll(filepath.Dir(path), parentDirectoriesPerms)
	if err != nil {
		return fmt.Errorf("creating parent directories: %w", err)
	}

	// the owner needs to be able to read the file and to write it again
	// afterwards, whatever the permissions stored in the archive say
	const ownerFilePerms = 0o600
	perm |= ownerFilePerms

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}

	// the file is always closed, and a failing close is reported along with
	// the failing copy when both happen
	_, copyErr := io.Copy(file, r)
	closeErr := file.Close()
	if copyErr != nil {
		return errors.Join(fmt.Errorf("writing file: %w", copyErr), closeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing file: %w", closeErr)
	}

	return nil
}

func cleanSubDirectory(subDirectory string) (cleaned string, err error) {
	cleaned = strings.Trim(path.Clean("/"+subDirectory), "/")
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("cleaning sub directory: %w", ErrSubDirectoryEmpty)
	}

	return cleaned, nil
}

// entryRelativePath tells if an archive entry is inside subDirectory, and if
// so returns its path relative to subDirectory. The single top level
// directory GitHub adds around the content of its archives is skipped.
func entryRelativePath(entryName, subDirectory string) (relativePath string, inside bool) {
	name := strings.TrimPrefix(path.Clean("/"+entryName), "/")
	withoutTop := name
	if index := strings.Index(name, "/"); index >= 0 {
		withoutTop = name[index+1:]
	}

	for _, candidate := range []string{name, withoutTop} {
		switch {
		case candidate == subDirectory:
			return "", true
		case strings.HasPrefix(candidate, subDirectory+"/"):
			return strings.TrimPrefix(candidate, subDirectory+"/"), true
		}
	}

	return "", false
}
