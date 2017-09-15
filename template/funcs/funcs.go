package funcs

import (
	"github.com/Masterminds/sprig"
	"text/template"
)

var Funcs template.FuncMap

func init() {
	Funcs = sprig.TxtFuncMap()
	Funcs["splitPreserveQuotes"] = splitPreserveQuotes
}
