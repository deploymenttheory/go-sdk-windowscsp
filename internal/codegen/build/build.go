// Package build flattens parsed CSP snapshots into fully-resolved view
// models. It is the only codegen stage that inspects the ddf model; every
// naming and type decision funnels through here (via the naming package) so
// templates stay decision-free.
package build

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/naming"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/view"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/ddf"
)

const commentWidth = 96

// Package builds the view model for one CSP snapshot. Returns nil when the
// CSP yields no operations (nothing to generate).
func Package(snapshotBase string, csp *ddf.CSP, release string) *view.Package {
	isPolicy := csp.PolicyArea()
	family := "csp"
	if isPolicy {
		family = "policy"
	}
	pkgName := naming.PackageName(snapshotBase)
	base := strings.TrimRight(csp.Path, "/") + "/" + csp.Name

	// Multi-scope DDF files are split into per-scope snapshots by fetchddf
	// ("VPNv2_User"); the scope becomes part of the service identity so both
	// scopes can coexist in the registry.
	serviceName := naming.ExportName(csp.Name)
	if strings.HasSuffix(strings.TrimSuffix(snapshotBase, "_AreaDDF"), "_User") {
		serviceName += "User"
	}

	p := &view.Package{
		PackageName: pkgName,
		Dir:         family + "/" + pkgName,
		ServiceName: serviceName,
		CSPName:     csp.Name,
		BasePath:    base,
		Release:     release,
		IsPolicy:    isPolicy,
	}

	b := &builder{pkg: p, usedBases: map[string]int{}, usedConsts: map[string]bool{}}
	b.addURI("Root", "the "+csp.Name+" root node.", []part{{lit: base}})
	b.claimBase("Root")
	root := node{segs: nil, parts: []part{{lit: base}}}

	// Dynamic nodes directly under the CSP root (e.g. VPNv2 profiles) are
	// enumerated with a root-level List; the walk below only sees the root's
	// children, never the root itself.
	for i := range csp.Nodes {
		if csp.Nodes[i].Dynamic() {
			b.method(view.Method{
				Recv:         p.ServiceName,
				Name:         "List",
				CommentLines: []string{"List enumerates the child nodes of " + base + "."},
				ParamStr:     "ctx context.Context",
				ReturnSig:    "([]string, error)",
				Verb:         "List",
				URIExpr:      "URIRoot",
			})
			break
		}
	}
	b.walk(csp.Nodes, root)

	if len(p.Methods) == 0 {
		return nil
	}
	return p
}

// part is one piece of an OMA-URI under construction: a literal or a
// dynamic-segment parameter reference.
type part struct {
	lit   string // literal text (empty when param is set)
	param string // Go parameter name
}

// node carries the walk state down the tree.
type node struct {
	segs   []string // naming segments below the CSP root
	params []string // dynamic parameters accumulated so far, in path order
	parts  []part   // URI fragments, starting with the base literal
}

type builder struct {
	pkg        *view.Package
	usedBases  map[string]int
	usedConsts map[string]bool
	uris       map[string]bool
}

func (b *builder) walk(nodes []ddf.Node, parent node) {
	for i := range nodes {
		b.visit(&nodes[i], parent)
	}
}

func (b *builder) visit(n *ddf.Node, parent node) {
	cur := node{
		segs:   append(slices.Clone(parent.segs), segName(n)),
		params: slices.Clone(parent.params),
		parts:  append(slices.Clone(parent.parts), part{lit: "/"}),
	}
	if n.Dynamic() {
		p := b.paramFor(n, cur.params)
		cur.params = append(cur.params, p)
		cur.parts = append(cur.parts, part{param: p})
	} else {
		cur.parts = append(cur.parts, part{lit: n.Name})
	}

	interior := !n.Leaf()
	base := b.claimBase(naming.JoinExport(cur.segs))
	access := accessSet(n.Access)

	if interior {
		emitted := false
		if hasDynamicChild(n) && access["Get"] {
			b.method(listMethod(b.pkg.ServiceName, base, n, cur))
			b.ensureURI(base, cur)
			emitted = true
		}
		if n.Dynamic() {
			if access["Add"] {
				b.method(containerMethod("Create", "Add", "client.Node()", b.pkg.ServiceName, base, n, cur))
				emitted = true
			}
			if access["Delete"] {
				b.method(containerMethod("Delete", "Delete", "", b.pkg.ServiceName, base, n, cur))
				emitted = true
			}
			if emitted {
				b.ensureURI(base, cur)
			}
		}
		b.walk(n.Children, cur)
		return
	}

	// Leaf node.
	vk := valueKind(n)
	emitted := false
	valued := (access["Get"] && n.Format != "null") || access["Add"] || access["Replace"] || access["Exec"]
	enumType := ""
	if valued {
		// The allowed-values enum type must exist before methods so their
		// signatures can use it.
		enumType = b.enumFor(base, n, vk)
	}
	if access["Get"] && n.Format != "null" {
		b.method(getMethod(b.pkg.ServiceName, base, n, cur, vk, enumType))
		emitted = true
	}
	if access["Add"] {
		b.method(setMethod("Create", "Add", b.pkg.ServiceName, base, n, cur, vk, enumType))
		emitted = true
	}
	if access["Replace"] {
		b.method(setMethod("Update", "Replace", b.pkg.ServiceName, base, n, cur, vk, enumType))
		emitted = true
	}
	if access["Delete"] {
		b.method(containerMethod("Delete", "Delete", "", b.pkg.ServiceName, base, n, cur))
		emitted = true
	}
	if access["Exec"] {
		b.method(execMethod(b.pkg.ServiceName, base, n, cur, vk, enumType))
		emitted = true
	}
	if emitted {
		b.ensureURI(base, cur)
	}
}

func segName(n *ddf.Node) string {
	if !n.Dynamic() {
		return n.Name
	}
	if n.Title != "" {
		return n.Title
	}
	return "Item"
}

// paramFor derives a unique Go parameter name for a dynamic segment.
func (b *builder) paramFor(n *ddf.Node, taken []string) string {
	seed := n.Title
	if seed == "" {
		seed = "item"
	}
	name := naming.ParamName(seed)
	if name == "" {
		name = "item"
	}
	candidate := name
	for i := 2; slices.Contains(taken, candidate); i++ {
		candidate = name + strconv.Itoa(i)
	}
	return candidate
}

// claimBase returns a package-unique naming base for a node.
func (b *builder) claimBase(base string) string {
	n := b.usedBases[base]
	b.usedBases[base] = n + 1
	if n == 0 {
		return base
	}
	return b.claimBase(base + strconv.Itoa(n+1))
}

func hasDynamicChild(n *ddf.Node) bool {
	for i := range n.Children {
		if n.Children[i].Dynamic() {
			return true
		}
	}
	return false
}

func accessSet(access []string) map[string]bool {
	out := make(map[string]bool, len(access))
	for _, a := range access {
		out[a] = true
	}
	return out
}

// ensureURI registers the URI const/builder for a node once.
func (b *builder) ensureURI(base string, cur node) {
	doc := "the " + strings.TrimPrefix(displayPath(cur.parts), b.pkg.BasePath+"/") + " node."
	b.addURIParts(base, doc, cur)
}

func (b *builder) addURI(base, doc string, parts []part) {
	b.addURIParts(base, doc, node{parts: parts})
}

func (b *builder) addURIParts(base, doc string, cur node) {
	if b.uris == nil {
		b.uris = map[string]bool{}
	}
	name := "URI" + base
	if b.uris[name] {
		return
	}
	b.uris[name] = true

	u := view.URI{Name: name, Doc: doc}
	if len(cur.params) == 0 {
		u.Literal = literalPath(cur.parts)
	} else {
		u.IsFunc = true
		u.Params = cur.params
		u.Expr = exprPath(cur.parts)
	}
	b.pkg.URIs = append(b.pkg.URIs, u)
}

func literalPath(parts []part) string {
	var s strings.Builder
	for _, p := range parts {
		s.WriteString(p.lit)
	}
	return s.String()
}

// displayPath renders the URI with {param} placeholders for docs.
func displayPath(parts []part) string {
	var s strings.Builder
	for _, p := range parts {
		if p.param != "" {
			s.WriteString("{" + p.param + "}")
			continue
		}
		s.WriteString(p.lit)
	}
	return s.String()
}

// exprPath renders a Go string-concatenation expression for a
// parameterized URI.
func exprPath(parts []part) string {
	var pieces []string
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			pieces = append(pieces, strconv.Quote(lit.String()))
			lit.Reset()
		}
	}
	for _, p := range parts {
		if p.param != "" {
			flush()
			pieces = append(pieces, p.param)
			continue
		}
		lit.WriteString(p.lit)
	}
	flush()
	return strings.Join(pieces, " + ")
}

// uriExpr is what generated method bodies pass as the uri argument.
func uriExpr(base string, cur node) string {
	if len(cur.params) == 0 {
		return "URI" + base
	}
	return "URI" + base + "(" + strings.Join(cur.params, ", ") + ")"
}

// kind describes how a leaf's DDF format maps onto Go.
type kind struct {
	goType      string
	ctor        string // client constructor applied to the value param
	accessor    string
	accessorErr bool
	zero        string
}

func valueKind(n *ddf.Node) kind {
	switch n.Format {
	case "int":
		return kind{"int64", "client.Int", "Int", true, "0"}
	case "bool":
		return kind{"bool", "client.Bool", "Bool", true, "false"}
	case "float":
		return kind{"float64", "client.Float", "Float", true, "0"}
	case "b64":
		return kind{"[]byte", "client.B64", "Bytes", true, "nil"}
	case "bin":
		return kind{"[]byte", "client.Bin", "Bytes", true, "nil"}
	case "xml":
		return kind{"string", "client.XML", "Str", false, `""`}
	case "date":
		return kind{"string", "client.Date", "Str", false, `""`}
	case "time":
		return kind{"string", "client.Time", "Str", false, `""`}
	case "null":
		return kind{"", "", "", false, ""}
	default: // chr and anything unrecognised transports as a string
		return kind{"string", "client.Chr", "Str", false, `""`}
	}
}

func paramStr(cur node, valueType string) string {
	s := "ctx context.Context"
	for _, p := range cur.params {
		s += ", " + p + " string"
	}
	if valueType != "" {
		s += ", value " + valueType
	}
	return s
}

func getMethod(recv, base string, n *ddf.Node, cur node, vk kind, enumType string) view.Method {
	name := "Get" + base
	retType := vk.goType
	if enumType != "" {
		retType = enumType
	}
	return view.Method{
		Recv:         recv,
		Name:         name,
		CommentLines: comments(name, "reads", n, cur),
		ParamStr:     paramStr(cur, ""),
		ReturnSig:    "(" + retType + ", error)",
		Verb:         "Get",
		URIExpr:      uriExpr(base, cur),
		Accessor:     vk.accessor,
		AccessorErr:  vk.accessorErr,
		Zero:         vk.zero,
		Cast:         enumType,
	}
}

// valueParam resolves the value parameter type and wire-encoding expression
// for setters: enum-typed nodes take the named type and convert back to the
// wire scalar.
func valueParam(vk kind, enumType string) (valueType, valueExpr string) {
	if vk.goType == "" {
		return "", "client.Null()"
	}
	if enumType == "" {
		return vk.goType, vk.ctor + "(value)"
	}
	return enumType, vk.ctor + "(" + vk.goType + "(value))"
}

func setMethod(prefix, verb, recv, base string, n *ddf.Node, cur node, vk kind, enumType string) view.Method {
	name := prefix + base
	valueType, valueExpr := valueParam(vk, enumType)
	action := "creates"
	if verb == "Replace" {
		action = "updates"
	}
	return view.Method{
		Recv:         recv,
		Name:         name,
		CommentLines: comments(name, action, n, cur),
		ParamStr:     paramStr(cur, valueType),
		ReturnSig:    "error",
		Verb:         verb,
		URIExpr:      uriExpr(base, cur),
		ValueExpr:    valueExpr,
	}
}

func containerMethod(prefix, verb, valueExpr, recv, base string, n *ddf.Node, cur node) view.Method {
	name := prefix + base
	action := "creates"
	if verb == "Delete" {
		action = "deletes"
	}
	return view.Method{
		Recv:         recv,
		Name:         name,
		CommentLines: comments(name, action, n, cur),
		ParamStr:     paramStr(cur, ""),
		ReturnSig:    "error",
		Verb:         verb,
		URIExpr:      uriExpr(base, cur),
		ValueExpr:    valueExpr,
	}
}

func listMethod(recv, base string, n *ddf.Node, cur node) view.Method {
	name := "List" + base
	if base == "Root" {
		name = "List"
	}
	return view.Method{
		Recv:         recv,
		Name:         name,
		CommentLines: comments(name, "lists the children of", n, cur),
		ParamStr:     paramStr(cur, ""),
		ReturnSig:    "([]string, error)",
		Verb:         "List",
		URIExpr:      uriExpr(base, cur),
	}
}

func execMethod(recv, base string, n *ddf.Node, cur node, vk kind, enumType string) view.Method {
	name := "Exec" + base
	valueType := ""
	valueExpr := "client.Null()"
	if n.Format != "null" && vk.goType != "" {
		valueType, valueExpr = valueParam(vk, enumType)
	}
	return view.Method{
		Recv:         recv,
		Name:         name,
		CommentLines: comments(name, "executes", n, cur),
		ParamStr:     paramStr(cur, valueType),
		ReturnSig:    "error",
		Verb:         "Exec",
		URIExpr:      uriExpr(base, cur),
		ValueExpr:    valueExpr,
	}
}

// comments builds the method doc comment: summary line, description,
// metadata and deprecation.
func comments(name, action string, n *ddf.Node, cur node) []string {
	lines := []string{name + " " + action + " " + displayPath(cur.parts) + "."}
	if d := naming.FirstLine(n.Description); d != "" {
		lines = append(lines, naming.WrapComment(d, commentWidth)...)
	}
	var meta []string
	if n.Default != "" {
		meta = append(meta, "Default: "+n.Default+".")
	}
	if a := n.Applicability; a != nil {
		if a.MinOSBuild != "" {
			m := "Supported from OS build " + a.MinOSBuild
			if a.CSPVersion != "" {
				m += " (CSP v" + a.CSPVersion + ")"
			}
			meta = append(meta, m+".")
		}
		if a.RequiresAAD {
			meta = append(meta, "Requires Microsoft Entra joined devices.")
		}
	}
	if n.RebootBehavior != "" {
		meta = append(meta, "Reboot behavior: "+n.RebootBehavior+".")
	}
	if len(meta) > 0 {
		lines = append(lines, "")
		lines = append(lines, meta...)
	}
	if n.DeprecatedOSBuild != "" {
		dep := "Deprecated: no longer recommended"
		if n.DeprecatedOSBuild != "deprecated" {
			dep += " since OS build " + n.DeprecatedOSBuild
		}
		lines = append(lines, "", dep+".")
	}
	return lines
}

// enumFor emits the allowed-values enum type for an ENUM/Flag leaf and
// returns its type name ("" when the node has no usable enum). Members keep
// their node-base prefix (AllowCameraNotAllowed); the type itself is named
// <base>Value.
func (b *builder) enumFor(base string, n *ddf.Node, vk kind) string {
	av := n.AllowedValues
	if av == nil || (av.Type != "ENUM" && av.Type != "Flag") || len(av.Enum) == 0 {
		return ""
	}
	typed := vk.goType == "int64"
	if !typed && vk.goType != "string" {
		return ""
	}

	baseType := "string"
	if typed {
		baseType = "int64"
	}
	typeName := b.claimBase(base + "Value")
	block := view.EnumBlock{
		TypeName: typeName,
		BaseType: baseType,
		Comment:  "allowed values for the " + n.Name + " node.",
	}
	seen := map[string]bool{}
	seenLit := map[string]bool{}
	for _, e := range av.Enum {
		lit := strconv.Quote(e.Value)
		if typed {
			v, err := strconv.ParseInt(e.Value, 0, 64)
			if err != nil {
				return "" // non-numeric member in an int enum: skip the block
			}
			lit = strconv.FormatInt(v, 10)
		}
		name := base + naming.ConstName(e.Description, e.Value)
		for i := 2; seen[name] || b.usedConsts[name]; i++ {
			name = fmt.Sprintf("%s%s%d", base, naming.ConstName(e.Description, e.Value), i)
		}
		seen[name] = true
		var comment []string
		if d := naming.FirstLine(e.Description); d != "" {
			comment = naming.WrapComment(d, commentWidth)
		}
		block.Members = append(block.Members, view.EnumMember{
			Name:         name,
			CommentLines: comment,
			Literal:      lit,
			Dup:          seenLit[lit],
		})
		seenLit[lit] = true
	}
	for _, m := range block.Members {
		b.usedConsts[m.Name] = true
	}
	b.pkg.Enums = append(b.pkg.Enums, block)
	return typeName
}

func (b *builder) method(m view.Method) {
	b.pkg.Methods = append(b.pkg.Methods, m)
}
