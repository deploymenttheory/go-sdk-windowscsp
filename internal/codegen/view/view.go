// Package view defines the pure-data models the templates render. Every
// field is a fully-resolved fragment: templates only branch and interpolate,
// they never make naming or type decisions.
package view

// Package describes one generated CSP domain package.
type Package struct {
	PackageName string
	Dir         string // path under the output root, e.g. "csp/laps"
	ServiceName string
	CSPName     string
	BasePath    string // full OMA-URI of the CSP root
	Release     string // DDF release label from provenance
	IsPolicy    bool
	URIs        []URI
	Methods     []Method
	Enums       []EnumBlock
}

// URI is a generated OMA-URI constant (no params) or builder function.
type URI struct {
	Name    string   // e.g. "URIAllowCamera"
	Doc     string   // one-line comment
	IsFunc  bool     // const when false
	Literal string   // const value (IsFunc == false)
	Params  []string // builder parameter names (IsFunc == true)
	Expr    string   // builder return expression (IsFunc == true)
}

// Method is one generated service method.
type Method struct {
	Recv         string
	Name         string
	CommentLines []string
	ParamStr     string // full parameter list, starting with ctx
	ReturnSig    string // e.g. "(int64, error)" or "error"
	Verb         string // Get, List, Add, Replace, Delete, Exec
	URIExpr      string // "URIAllowCamera" or "URIProfileServer(profileName)"
	ValueExpr    string // e.g. "client.Int(value)"; empty for Get/List/Delete
	Accessor     string // Get only: Int, Bool, Float, Str, Bytes
	AccessorErr  bool   // Get only: accessor returns (T, error)
	Zero         string // Get only: zero literal for the error path
	// Cast is the named allowed-values type the wire value converts
	// to/from ("" when the node has no enum type).
	Cast string
}

// EnumBlock is one allowed-values enum for a node: a named type, its typed
// constants and a String method.
type EnumBlock struct {
	TypeName string
	BaseType string // "int64" or "string"
	Comment  string
	Members  []EnumMember
}

// EnumMember is one allowed-value constant.
type EnumMember struct {
	Name         string
	CommentLines []string
	Literal      string // rendered literal (already quoted for strings)
	// Dup marks members whose literal repeats an earlier member's value;
	// they are excluded from the String switch.
	Dup bool
}

// Registry describes the generated root-client registry file.
type Registry struct {
	Release  string
	Families []Family
}

// Family is one service family (standalone CSPs, Policy areas).
type Family struct {
	TypeName  string // e.g. "CSPServices"
	FieldName string // field on Client, e.g. "CSP"
	Doc       string
	Entries   []FamilyEntry
}

// FamilyEntry wires one domain package into its family struct.
type FamilyEntry struct {
	FieldName   string // e.g. "Camera"
	ServiceName string // type inside the domain package
	Alias       string // import alias
	ImportPath  string
}
