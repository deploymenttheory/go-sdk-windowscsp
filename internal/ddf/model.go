// Package ddf models Microsoft's DDF v2 (Device Description Framework)
// metadata for Windows Configuration Service Providers, and parses the DDF
// XML files shipped in Microsoft's DDF v2 zip drops.
//
// The model is the committed snapshot shape: cmd/fetchddf writes one JSON
// file per CSP under metadata/csp, and cmd/gencsp consumes those snapshots
// offline. Everything downstream of the parser is deterministic.
package ddf

import "strings"

// Provenance records where a snapshot tree came from.
type Provenance struct {
	Release string `json:"release"`
	Source  string `json:"source"`
	SHA256  string `json:"sha256"`
	Fetched string `json:"fetched"`
}

// CSP is one parsed DDF file: a configuration service provider or a Policy
// area.
type CSP struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Nodes []Node `json:"nodes,omitempty"`
}

// PolicyArea reports whether the CSP is a Policy CSP area (a subtree of
// ./Vendor/MSFT/Policy/Config) rather than a standalone CSP.
func (c CSP) PolicyArea() bool {
	return strings.Contains(c.Path, "/Policy/Config")
}

// Node is one node of the CSP management tree.
type Node struct {
	// Name is the OMA-DM node name. Empty for dynamic nodes, whose name is
	// chosen at runtime (see DynamicNaming and Title).
	Name string `json:"name"`
	// Title is the DFTitle, when present. For dynamic nodes it usually names
	// the placeholder (e.g. "ProfileName").
	Title string `json:"title,omitempty"`
	// Path is the full OMA-URI of the node. Dynamic segments are rendered as
	// {Title} placeholders.
	Path string `json:"path"`
	// Format is the DFFormat choice: b64, bin, bool, chr, int, node, null,
	// xml, date, time or float.
	Format string `json:"format,omitempty"`
	// Access lists the supported AccessType verbs, sorted: Add, Copy,
	// Delete, Exec, Get, Replace.
	Access      []string `json:"access,omitempty"`
	Description string   `json:"description,omitempty"`
	Default     string   `json:"default,omitempty"`
	// Occurrence is the DDF Occurrence choice (One, ZeroOrOne, ZeroOrMore,
	// OneOrMore, ZeroOrN, OneOrN), when declared.
	Occurrence string `json:"occurrence,omitempty"`
	// DynamicNaming describes how dynamic nodes are named:
	// "ServerGeneratedUniqueIdentifier", "ClientInventory", or a regex from
	// UniqueName.
	DynamicNaming string `json:"dynamicNaming,omitempty"`
	// DeprecatedOSBuild is set when the node carries MSFT:Deprecated; it
	// holds the OS build the node was deprecated in, or "deprecated" when
	// the attribute is absent.
	DeprecatedOSBuild string         `json:"deprecatedOsBuild,omitempty"`
	Applicability     *Applicability `json:"applicability,omitempty"`
	AllowedValues     *AllowedValues `json:"allowedValues,omitempty"`
	GpMapping         *GpMapping     `json:"gpMapping,omitempty"`
	RebootBehavior    string         `json:"rebootBehavior,omitempty"`
	AtomicRequired    bool           `json:"atomicRequired,omitempty"`
	Children          []Node         `json:"children,omitempty"`
}

// Dynamic reports whether the node is a dynamic (runtime-named) node.
func (n Node) Dynamic() bool { return n.Name == "" }

// Leaf reports whether the node carries a value (or is an Exec target)
// rather than grouping children.
func (n Node) Leaf() bool { return n.Format != "" && n.Format != "node" }

// GoType maps the node's DFFormat to the Go type generated accessors use.
// Interior ("node") and unknown formats return "".
func (n Node) GoType() string {
	switch n.Format {
	case "int":
		return "int64"
	case "bool":
		return "bool"
	case "float":
		return "float64"
	case "chr", "xml", "date", "time":
		return "string"
	case "b64", "bin":
		return "[]byte"
	case "null":
		// Exec-only nodes: no payload.
		return ""
	}
	return ""
}

// Applicability captures MSFT:Applicability.
type Applicability struct {
	MinOSBuild  string `json:"minOsBuild,omitempty"`
	CSPVersion  string `json:"cspVersion,omitempty"`
	Editions    string `json:"editions,omitempty"`
	RequiresAAD bool   `json:"requiresAad,omitempty"`
}

// AllowedValues captures MSFT:AllowedValues.
type AllowedValues struct {
	// Type is the ValueType attribute: XSD, RegEx, ADMX, JSON, ENUM, Flag,
	// Range, SDDL or None.
	Type string `json:"type"`
	// Value holds the single Value tag body for XSD/RegEx/JSON/Range types.
	Value string       `json:"value,omitempty"`
	Enum  []EnumValue  `json:"enum,omitempty"`
	ADMX  *ADMXBacking `json:"admx,omitempty"`
	// ListDelimiter is set when the node accepts a delimited list of values.
	ListDelimiter string `json:"listDelimiter,omitempty"`
}

// EnumValue is one ENUM/Flag member.
type EnumValue struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// ADMXBacking captures MSFT:AdmxBacked.
type ADMXBacking struct {
	Area string `json:"area"`
	Name string `json:"name"`
	File string `json:"file"`
}

// GpMapping captures MSFT:GpMapping.
type GpMapping struct {
	EnglishName string `json:"englishName,omitempty"`
	AreaPath    string `json:"areaPath,omitempty"`
	Element     string `json:"element,omitempty"`
}
