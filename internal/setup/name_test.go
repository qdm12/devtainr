package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNewName = "new-name"

func Test_ChangeName(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		content   string
		name      string
		expected  string
		noFile    bool
		errTarget error
	}{
		"name is replaced and the other keys are kept": {
			content:  "{\n  \"name\" : \"old\",\n  \"a\": \"b\"\n}\n",
			name:     testNewName,
			expected: "{\n  \"name\": \"" + testNewName + "\",\n  \"a\": \"b\"\n}\n",
		},
		"name holding a dollar sign is not expanded": {
			content:  `{"name": "old", "a": "b"},`,
			name:     "$1project",
			expected: `{"name": "$1project", "a": "b"},`,
		},
		"only the first name is replaced": {
			content:  `{"name": "old", "a": "x, y", "tools": {"name": "nested"}, "b": "c"}`,
			name:     testNewName,
			expected: `{"name": "` + testNewName + `", "a": "x, y", "tools": {"name": "nested"}, "b": "c"}`,
		},
		"name without trailing comma": {
			content:  `{"name": "old"}`,
			name:     testNewName,
			expected: `{"name": "` + testNewName + `"}`,
		},
		"no name": {
			content:   `{"a": "b"}`,
			name:      testNewName,
			errTarget: ErrNameNotFound,
		},
		"file does not exist": {
			noFile:    true,
			name:      testNewName,
			errTarget: os.ErrNotExist,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), devcontainerFileName)
			if !testCase.noFile {
				require.NoError(t, os.WriteFile(path, []byte(testCase.content), 0600))
			}

			err := ChangeName(path, testCase.name)
			if testCase.errTarget != nil {
				assert.ErrorIs(t, err, testCase.errTarget)
				return
			}

			require.NoError(t, err)
			b, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, string(b))
		})
	}
}

func Test_DevcontainerFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	paths := []string{
		devcontainerFileName,
		"docker/" + devcontainerFileName,
		"podman/nested/" + devcontainerFileName,
		testDockerfile,
		testReadme,
	}
	for _, path := range paths {
		fullpath := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(fullpath), 0777))
		require.NoError(t, os.WriteFile(fullpath, []byte("content"), 0600))
	}

	// a symbolic link is not a configuration file to be written back
	require.NoError(t, os.Symlink(
		filepath.Join(root, testDockerfile),
		filepath.Join(root, "podman", devcontainerFileName),
	))

	found, err := DevcontainerFiles(root)
	require.NoError(t, err)
	assert.Equal(t, []string{
		devcontainerFileName,
		"docker/" + devcontainerFileName,
		"podman/nested/" + devcontainerFileName,
	}, found)
}

func Test_nameRegex(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		s     string
		found bool
	}{
		"empty string": {},
		"not containing name": {
			s: `{"a": "b"},`,
		},
		"containing name without spaces": {
			s:     `"name": "b",`,
			found: true,
		},
		"containing name with spaces": {
			s:     ` "name"  : 	 "b",`,
			found: true,
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			foundString := nameRegex.FindString(testCase.s)
			found := len(foundString) > 0
			assert.Equal(t, testCase.found, found)
		})
	}
}
