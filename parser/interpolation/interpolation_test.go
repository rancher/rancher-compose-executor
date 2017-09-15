package interpolation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testInterpolatedLine(t *testing.T, expectedLine, interpolatedLine string, envVariables map[string]string) {
	interpolatedLine, _ = parseLine(interpolatedLine, func(s string) string {
		return envVariables[s]
	})

	assert.Equal(t, expectedLine, interpolatedLine)
}

func testInvalidInterpolatedLine(t *testing.T, line string) {
	_, success := parseLine(line, func(string) string {
		return ""
	})

	assert.Equal(t, false, success)
}

func TestParseLine(t *testing.T) {
	variables := map[string]string{
		"A":           "ABC",
		"X":           "XYZ",
		"E":           "",
		"lower":       "WORKED",
		"MiXeD":       "WORKED",
		"split_VaLue": "WORKED",
		"9aNumber":    "WORKED",
		"a9Number":    "WORKED",
		"defTest":     "WORKED",
	}

	testInterpolatedLine(t, "WORKED", "$lower", variables)
	testInterpolatedLine(t, "WORKED", "${MiXeD}", variables)
	testInterpolatedLine(t, "WORKED", "${split_VaLue}", variables)
	// Starting with a number isn't valid
	testInterpolatedLine(t, "", "$9aNumber", variables)
	testInterpolatedLine(t, "WORKED", "$a9Number", variables)

	testInterpolatedLine(t, "ABC", "$A", variables)
	testInterpolatedLine(t, "ABC", "${A}", variables)

	testInterpolatedLine(t, "ABC DE", "$A DE", variables)
	testInterpolatedLine(t, "ABCDE", "${A}DE", variables)

	testInterpolatedLine(t, "$A", "$$A", variables)
	testInterpolatedLine(t, "${A}", "$${A}", variables)

	testInterpolatedLine(t, "$ABC", "$$${A}", variables)
	testInterpolatedLine(t, "$ABC", "$$$A", variables)

	testInterpolatedLine(t, "ABC XYZ", "$A $X", variables)
	testInterpolatedLine(t, "ABCXYZ", "$A$X", variables)
	testInterpolatedLine(t, "ABCXYZ", "${A}${X}", variables)

	testInterpolatedLine(t, "", "$B", variables)
	testInterpolatedLine(t, "", "${B}", variables)
	testInterpolatedLine(t, "", "$ADE", variables)

	testInterpolatedLine(t, "", "$E", variables)
	testInterpolatedLine(t, "", "${E}", variables)

	testInvalidInterpolatedLine(t, "${df:val}")
	testInvalidInterpolatedLine(t, "${")
	testInvalidInterpolatedLine(t, "$}")
	testInvalidInterpolatedLine(t, "${}")
	testInvalidInterpolatedLine(t, "${ }")
	testInvalidInterpolatedLine(t, "${A }")
	testInvalidInterpolatedLine(t, "${ A}")
	testInvalidInterpolatedLine(t, "${A!}")
	testInvalidInterpolatedLine(t, "$!")
}
