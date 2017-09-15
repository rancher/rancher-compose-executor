package funcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitPreserveQuotes(t *testing.T) {
	assert.Equal(t, []string{
		"--a",
	}, splitPreserveQuotes("--a"))
	assert.Equal(t, []string{
		"--a",
		"--b",
	}, splitPreserveQuotes("--a --b"))
	assert.Equal(t, []string{
		"--a",
		"--b='c d'",
		"--e='f'",
	}, splitPreserveQuotes("--a --b='c d' --e='f'"))
	assert.Equal(t, []string{
		"--a",
		"--b=\"c d\"",
		"--e=\"f\"",
	}, splitPreserveQuotes("--a --b=\"c d\" --e=\"f\""))
}
