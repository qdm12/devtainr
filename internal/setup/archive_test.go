package setup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTopDirectory      = "godevcontainer-HEAD"
	testSubDirectory      = ".devcontainer"
	testDockerfile        = "Dockerfile"
	testDockerfileContent = "FROM alpine"
	testDockerignore      = ".dockerignore"
	testReadme            = "README.md"
	testJSON              = `{"name": "old"}`
)

type archiveEntry struct {
	name    string
	content string
	mode    int64
	isDir   bool
	isLink  bool
}

// newArchive builds a tar.gz archive the way GitHub does: every entry is
// placed under a single top level directory named after the repository.
func newArchive(t *testing.T, top string, entries []archiveEntry) []byte {
	t.Helper()

	buffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		name := entry.name
		if top != "" {
			name = top + "/" + name
		}
		if entry.isDir {
			name += "/"
		}

		header := &tar.Header{Name: name, Mode: entry.mode}
		switch {
		case entry.isDir:
			header.Typeflag = tar.TypeDir
		case entry.isLink:
			header.Typeflag = tar.TypeSymlink
			header.Linkname = "../" + entry.content
		default:
			header.Typeflag = tar.TypeReg
			header.Size = int64(len(entry.content))
		}

		require.NoError(t, tarWriter.WriteHeader(header))
		if entry.isDir || entry.isLink {
			continue
		}

		_, err := tarWriter.Write([]byte(entry.content))
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	return buffer.Bytes()
}

func Test_ExtractFiles(t *testing.T) {
	t.Parallel()
	twoEntries := []archiveEntry{
		{name: testSubDirectory, isDir: true, mode: 0755},
		{name: testSubDirectory + "/" + testDockerfile, content: testDockerfileContent, mode: 0644},
	}

	testCases := map[string]struct {
		top           string
		subDirectory  string
		entries       []archiveEntry
		expected      []Entry
		contents      map[string]string
		writableDirs  []string
		writableFiles []string
		absent        []string
		absentBeside  []string
		errTarget     error
	}{
		"files and directories at any depth": {
			top:          testTopDirectory,
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory, isDir: true, mode: 0755},
				{name: testSubDirectory + "/" + testDockerignore, content: "ignore/", mode: 0644},
				{name: testSubDirectory + "/" + testDockerfile, content: testDockerfileContent, mode: 0644},
				{name: testSubDirectory + "/docker", isDir: true, mode: 0755},
				{name: testSubDirectory + "/docker/" + devcontainerFileName, content: testJSON, mode: 0644},
				{name: testSubDirectory + "/podman/nested", isDir: true, mode: 0700},
				{name: testSubDirectory + "/podman/nested/" + devcontainerFileName, content: testJSON, mode: 0444},
				{name: testSubDirectory + "/link", content: testDockerfile, isLink: true},
				{name: testReadme, content: "not ours"},
				{name: ".github/workflows/ci.yml", content: "not ours"},
			},
			expected: []Entry{
				{Path: testDockerignore},
				{Path: testDockerfile},
				{Path: "docker", IsDir: true},
				{Path: "docker/" + devcontainerFileName},
				{Path: "podman/nested", IsDir: true},
				{Path: "podman/nested/" + devcontainerFileName},
			},
			contents: map[string]string{
				testDockerignore:                        "ignore/",
				testDockerfile:                          testDockerfileContent,
				"docker/" + devcontainerFileName:        testJSON,
				"podman/nested/" + devcontainerFileName: testJSON,
			},
			// the owner can always read and write a file again afterwards,
			// whatever the permissions stored in the archive say
			writableFiles: []string{"podman/nested/" + devcontainerFileName},
			writableDirs:  []string{"podman/nested"},
			// files outside of the wanted sub directory and links are left alone
			absent: []string{testReadme, ".github", "link"},
		},
		"directory the archive says is not writable": {
			top:          testTopDirectory,
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory + "/podman", isDir: true, mode: 0500},
				{name: testSubDirectory + "/podman/" + devcontainerFileName, content: testJSON, mode: 0444},
			},
			expected: []Entry{
				{Path: "podman", IsDir: true},
				{Path: "podman/" + devcontainerFileName},
			},
			contents: map[string]string{
				"podman/" + devcontainerFileName: testJSON,
			},
			writableDirs: []string{"podman"},
		},
		"file before its directory": {
			top:          testTopDirectory,
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory + "/podman/" + devcontainerFileName, content: testJSON, mode: 0644},
				{name: testSubDirectory + "/podman", isDir: true, mode: 0755},
			},
			expected: []Entry{
				{Path: "podman/" + devcontainerFileName},
				{Path: "podman", IsDir: true},
			},
			contents: map[string]string{
				"podman/" + devcontainerFileName: testJSON,
			},
		},
		"no top level directory": {
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory + "/" + testDockerfile, content: testDockerfileContent, mode: 0644},
			},
			expected: []Entry{{Path: testDockerfile}},
			contents: map[string]string{
				testDockerfile: testDockerfileContent,
			},
		},
		"entries escaping the sub directory": {
			top:          testTopDirectory,
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory + "/" + testDockerfile, content: testDockerfileContent, mode: 0644},
				{name: testSubDirectory + "/../../escape.txt", content: "escape", mode: 0644},
				{name: testSubDirectory + "/../" + devcontainerFileName, content: testJSON, mode: 0644},
			},
			expected: []Entry{{Path: testDockerfile}},
			contents: map[string]string{
				testDockerfile: testDockerfileContent,
			},
			absentBeside: []string{"escape.txt", devcontainerFileName},
		},
		"sub directory absent from archive": {
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testReadme, content: "hello"},
			},
			errTarget: ErrSubDirectoryNotFound,
		},
		"empty sub directory": {
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory, isDir: true, mode: 0755},
			},
			errTarget: ErrSubDirectoryNotFound,
		},
		"links only": {
			subDirectory: testSubDirectory,
			entries: []archiveEntry{
				{name: testSubDirectory + "/link", content: testDockerfile, isLink: true},
			},
			errTarget: ErrSubDirectoryNotFound,
		},
		"empty sub directory name": {
			subDirectory: "",
			entries:      twoEntries,
			errTarget:    ErrSubDirectoryEmpty,
		},
		"root sub directory name": {
			subDirectory: "/",
			entries:      twoEntries,
			errTarget:    ErrSubDirectoryEmpty,
		},
		"not a gzip archive": {
			subDirectory: testSubDirectory,
			errTarget:    ErrArchiveNotGzip,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var archive []byte
			if testCase.entries != nil {
				archive = newArchive(t, testCase.top, testCase.entries)
			}

			destination := t.TempDir()
			entries, err := extractFiles(bytes.NewReader(archive), testCase.subDirectory, destination)
			if testCase.errTarget != nil {
				assert.ErrorIs(t, err, testCase.errTarget)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, entries)

			for path, content := range testCase.contents {
				b, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(path)))
				require.NoError(t, err, path)
				assert.Equal(t, content, string(b), path)
			}
			for _, path := range testCase.absent {
				_, err := os.Lstat(filepath.Join(destination, filepath.FromSlash(path)))
				assert.ErrorIs(t, err, os.ErrNotExist, path)
			}
			for _, path := range testCase.absentBeside {
				_, err := os.Lstat(filepath.Join(filepath.Dir(destination), filepath.FromSlash(path)))
				assert.ErrorIs(t, err, os.ErrNotExist, path)
			}
			for _, path := range testCase.writableFiles {
				info, err := os.Stat(filepath.Join(destination, filepath.FromSlash(path)))
				require.NoError(t, err, path)
				assert.NotZero(t, info.Mode().Perm()&0600, path)
			}
			for _, path := range testCase.writableDirs {
				info, err := os.Stat(filepath.Join(destination, filepath.FromSlash(path)))
				require.NoError(t, err, path)
				assert.NotZero(t, info.Mode().Perm()&0700, path)
			}
		})
	}
}

func Test_entryRelativePath(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		entryName    string
		subDirectory string
		relativePath string
		inside       bool
	}{
		"file in sub directory": {
			entryName:    testTopDirectory + "/" + testSubDirectory + "/" + testDockerfile,
			subDirectory: testSubDirectory,
			relativePath: testDockerfile,
			inside:       true,
		},
		"nested file": {
			entryName:    testTopDirectory + "/" + testSubDirectory + "/podman/" + devcontainerFileName,
			subDirectory: testSubDirectory,
			relativePath: "podman/" + devcontainerFileName,
			inside:       true,
		},
		"sub directory itself": {
			entryName:    testTopDirectory + "/" + testSubDirectory,
			subDirectory: testSubDirectory,
			relativePath: "",
			inside:       true,
		},
		"file outside of sub directory": {
			entryName:    testTopDirectory + "/.github/workflows/ci.yml",
			subDirectory: testSubDirectory,
		},
		"similar name outside of sub directory": {
			entryName:    testTopDirectory + "/" + testSubDirectory + "-extra/file",
			subDirectory: testSubDirectory,
		},
		"escaping the sub directory": {
			entryName:    testTopDirectory + "/" + testSubDirectory + "/../../escape",
			subDirectory: testSubDirectory,
		},
		"no top level directory": {
			entryName:    testSubDirectory + "/" + testDockerfile,
			subDirectory: testSubDirectory,
			relativePath: testDockerfile,
			inside:       true,
		},
		"absolute entry name": {
			entryName:    "/" + testSubDirectory + "/" + testDockerfile,
			subDirectory: testSubDirectory,
			relativePath: testDockerfile,
			inside:       true,
		},
		"escaping entry name": {
			entryName:    "../../../escape",
			subDirectory: testSubDirectory,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			relativePath, inside := entryRelativePath(testCase.entryName, testCase.subDirectory)
			assert.Equal(t, testCase.relativePath, relativePath)
			assert.Equal(t, testCase.inside, inside)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newStubClient(response *http.Response, err error) *http.Client {
	return &http.Client{Transport: roundTripFunc(
		func(*http.Request) (*http.Response, error) { return response, err })}
}

func Test_DownloadFiles(t *testing.T) {
	t.Parallel()
	archive := newArchive(t, testTopDirectory, []archiveEntry{
		{name: testSubDirectory + "/" + testDockerfile, content: testDockerfileContent, mode: 0666},
		{name: testSubDirectory + "/podman/" + devcontainerFileName, content: testJSON, mode: 0666},
	})

	var requestedURL string
	client := &http.Client{Transport: roundTripFunc(
		func(request *http.Request) (*http.Response, error) {
			requestedURL = request.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(archive)),
				Header:     make(http.Header),
			}, nil
		})}

	destination := t.TempDir()
	entries, err := DownloadFiles(context.Background(), client,
		"qdm12/godevcontainer", testSubDirectory, destination)
	require.NoError(t, err)

	assert.Equal(t, "https://codeload.github.com/qdm12/godevcontainer/tar.gz/HEAD", requestedURL)
	assert.Equal(t, []Entry{
		{Path: testDockerfile},
		{Path: "podman/" + devcontainerFileName},
	}, entries)

	b, err := os.ReadFile(filepath.Join(destination, "podman", devcontainerFileName))
	require.NoError(t, err)
	assert.Equal(t, testJSON, string(b))
}

func Test_DownloadFiles_cancelledContext(t *testing.T) {
	t.Parallel()
	client := newStubClient(nil, context.Canceled)

	_, err := DownloadFiles(context.Background(), client,
		"qdm12/godevcontainer", testSubDirectory, t.TempDir())
	assert.ErrorIs(t, err, context.Canceled)
}

func Test_DownloadFiles_errors(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		response *http.Response
		target   error
	}{
		"not found": {
			response: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("404: not found")),
				Header:     make(http.Header),
			},
			target: ErrUnexpectedStatusCode,
		},
		"rate limited": {
			response: &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader("403: forbidden")),
				Header:     make(http.Header),
			},
			target: ErrUnexpectedStatusCode,
		},
		"not an archive": {
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("<html></html>")),
				Header:     make(http.Header),
			},
			target: ErrArchiveNotGzip,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := DownloadFiles(context.Background(), newStubClient(testCase.response, nil),
				"qdm12/godevcontainer", testSubDirectory, t.TempDir())
			assert.ErrorIs(t, err, testCase.target)
		})
	}
}
