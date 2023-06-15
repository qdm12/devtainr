package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
