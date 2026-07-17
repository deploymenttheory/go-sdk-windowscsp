// Package render is the firewall between view models and Go source: view
// models go in, source fragments come out, and the only mechanism is
// text/template execution over the embedded .tmpl files. No naming or type
// decisions happen here.
package render

import (
	"embed"
	"strings"
	"text/template"

	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/view"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templates = template.Must(template.New("csp").Funcs(template.FuncMap{
	"join": strings.Join,
}).ParseFS(templateFS, "templates/*.tmpl"))

func execute(name string, data any) (string, error) {
	var b strings.Builder
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Doc renders the package doc comment (without the package clause).
func Doc(p *view.Package) (string, error) { return execute("doc", p) }

// Service renders the service struct and constructor.
func Service(p *view.Package) (string, error) { return execute("service", p) }

// URIConsts renders the const block of static OMA-URIs.
func URIConsts(p *view.Package) (string, error) { return execute("uriconsts", p) }

// URIFunc renders one parameterized OMA-URI builder.
func URIFunc(u *view.URI) (string, error) { return execute("urifunc", u) }

// Method renders one service method.
func Method(m *view.Method) (string, error) { return execute("method", m) }

// EnumBlock renders one allowed-values const block.
func EnumBlock(e *view.EnumBlock) (string, error) { return execute("enumblock", e) }

// Registry renders the root client and family registries.
func Registry(r *view.Registry) (string, error) { return execute("registry", r) }
