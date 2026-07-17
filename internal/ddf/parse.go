package ddf

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

// Parse parses one DDF v2 XML document. A document may carry several root
// <Node> trees — most commonly a ./Device/Vendor/MSFT scope and a
// ./User/Vendor/MSFT scope for the same CSP — and each becomes its own CSP,
// in document order.
func Parse(data []byte) ([]*CSP, error) {
	var tree mgmtTree
	if err := xml.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("parse DDF: %w", err)
	}
	if len(tree.Nodes) == 0 {
		return nil, fmt.Errorf("parse DDF: no root Node in MgmtTree")
	}
	out := make([]*CSP, 0, len(tree.Nodes))
	for i := range tree.Nodes {
		root := &tree.Nodes[i]
		csp := &CSP{Name: root.NodeName, Path: root.Path}
		base := strings.TrimRight(root.Path, "/") + "/" + root.NodeName
		csp.Nodes = buildNodes(root.Nodes, base)
		out = append(out, csp)
	}
	return out, nil
}

func buildNodes(nodes []xmlNode, parentPath string) []Node {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]Node, 0, len(nodes))
	for i := range nodes {
		out = append(out, buildNode(&nodes[i], parentPath))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func buildNode(x *xmlNode, parentPath string) Node {
	n := Node{
		Name:        x.NodeName,
		Title:       strings.TrimSpace(x.Props.Title),
		Format:      x.Props.Format.first(),
		Access:      x.Props.Access.all(),
		Description: strings.TrimSpace(x.Props.Description),
		Default:     strings.TrimSpace(x.Props.DefaultValue),
		Occurrence:  x.Props.Occurrence.first(),
	}

	segment := n.Name
	if n.Dynamic() {
		title := n.Title
		if title == "" {
			title = "Item"
		}
		segment = "{" + title + "}"
	}
	n.Path = strings.TrimRight(parentPath, "/") + "/" + segment

	if d := x.Props.DynamicNaming; d != nil {
		switch {
		case d.ServerGenerated != nil:
			n.DynamicNaming = "ServerGeneratedUniqueIdentifier"
		case d.ClientInventory != nil:
			n.DynamicNaming = "ClientInventory"
		case d.UniqueName != "":
			n.DynamicNaming = strings.TrimSpace(d.UniqueName)
		}
	}
	if dep := x.Props.Deprecated; dep != nil {
		n.DeprecatedOSBuild = dep.OsBuildDeprecated
		if n.DeprecatedOSBuild == "" {
			n.DeprecatedOSBuild = "deprecated"
		}
	}
	if a := x.Props.Applicability; a != nil {
		app := Applicability{
			MinOSBuild:  strings.TrimSpace(a.OsBuildVersion),
			CSPVersion:  strings.TrimSpace(a.CspVersion),
			Editions:    strings.TrimSpace(a.EditionAllowList),
			RequiresAAD: a.RequiresAzureAd != nil,
		}
		if app != (Applicability{}) {
			n.Applicability = &app
		}
	}
	if av := x.Props.AllowedValues; av != nil {
		n.AllowedValues = av.convert()
	}
	if gp := x.Props.GpMapping; gp != nil {
		n.GpMapping = &GpMapping{
			EnglishName: gp.GpEnglishName,
			AreaPath:    gp.GpAreaPath,
			Element:     gp.GpElement,
		}
	}
	if rb := strings.TrimSpace(x.Props.RebootBehavior); rb != "" && rb != "None" {
		n.RebootBehavior = rb
	}
	n.AtomicRequired = x.Props.AtomicRequired != nil

	n.Children = buildNodes(x.Nodes, n.Path)
	return n
}

// mgmtTree is the root <MgmtTree> element. The DDF schema allows multiple
// top-level Node trees (one per scope); decoding into anything other than a
// slice would silently merge their children.
type mgmtTree struct {
	XMLName xml.Name  `xml:"MgmtTree"`
	Nodes   []xmlNode `xml:"Node"`
}

type xmlNode struct {
	NodeName string        `xml:"NodeName"`
	Path     string        `xml:"Path"`
	Props    xmlProperties `xml:"DFProperties"`
	Nodes    []xmlNode     `xml:"Node"`
}

type xmlProperties struct {
	Access         xmlChoice         `xml:"AccessType"`
	DefaultValue   string            `xml:"DefaultValue"`
	Description    string            `xml:"Description"`
	Format         xmlChoice         `xml:"DFFormat"`
	Occurrence     xmlChoice         `xml:"Occurrence"`
	Title          string            `xml:"DFTitle"`
	Applicability  *xmlApplicability `xml:"Applicability"`
	DynamicNaming  *xmlDynamicNaming `xml:"DynamicNodeNaming"`
	AllowedValues  *xmlAllowedValues `xml:"AllowedValues"`
	Deprecated     *xmlDeprecated    `xml:"Deprecated"`
	GpMapping      *xmlGpMapping     `xml:"GpMapping"`
	RebootBehavior string            `xml:"RebootBehavior"`
	AtomicRequired *struct{}         `xml:"AtomicRequired"`
}

// xmlChoice captures DDF elements whose payload is a set of empty child
// elements (<AccessType><Get/><Add/></AccessType>, <DFFormat><int/></DFFormat>,
// <Occurrence><One/></Occurrence>). The child *names* are the data.
type xmlChoice struct {
	Elems []xmlElem `xml:",any"`
}

type xmlElem struct {
	XMLName xml.Name
}

// first returns the first child element's local name, or "".
func (c xmlChoice) first() string {
	if len(c.Elems) == 0 {
		return ""
	}
	return c.Elems[0].XMLName.Local
}

// all returns every child element local name, sorted.
func (c xmlChoice) all() []string {
	if len(c.Elems) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.Elems))
	for _, e := range c.Elems {
		out = append(out, e.XMLName.Local)
	}
	sort.Strings(out)
	return out
}

type xmlApplicability struct {
	OsBuildVersion   string    `xml:"OsBuildVersion"`
	CspVersion       string    `xml:"CspVersion"`
	EditionAllowList string    `xml:"EditionAllowList"`
	RequiresAzureAd  *struct{} `xml:"RequiresAzureAd"`
}

type xmlDynamicNaming struct {
	ServerGenerated *struct{} `xml:"ServerGeneratedUniqueIdentifier"`
	ClientInventory *struct{} `xml:"ClientInventory"`
	UniqueName      string    `xml:"UniqueName"`
}

type xmlDeprecated struct {
	OsBuildDeprecated string `xml:"OsBuildDeprecated,attr"`
}

type xmlAllowedValues struct {
	ValueType string    `xml:"ValueType,attr"`
	Value     string    `xml:"Value"`
	Enums     []xmlEnum `xml:"Enum"`
	Admx      *xmlAdmx  `xml:"AdmxBacked"`
	List      *xmlList  `xml:"List"`
}

type xmlEnum struct {
	Value       string `xml:"Value"`
	Description string `xml:"ValueDescription"`
}

type xmlAdmx struct {
	Area string `xml:"Area,attr"`
	Name string `xml:"Name,attr"`
	File string `xml:"File,attr"`
}

type xmlList struct {
	Delimiter string `xml:"Delimiter,attr"`
}

type xmlGpMapping struct {
	GpEnglishName string `xml:"GpEnglishName,attr"`
	GpAreaPath    string `xml:"GpAreaPath,attr"`
	GpElement     string `xml:"GpElement,attr"`
}

func (a *xmlAllowedValues) convert() *AllowedValues {
	out := &AllowedValues{
		Type:  a.ValueType,
		Value: strings.TrimSpace(a.Value),
	}
	for _, e := range a.Enums {
		out.Enum = append(out.Enum, EnumValue{
			Value:       strings.TrimSpace(e.Value),
			Description: strings.TrimSpace(e.Description),
		})
	}
	if a.Admx != nil {
		out.ADMX = &ADMXBacking{Area: a.Admx.Area, Name: a.Admx.Name, File: a.Admx.File}
	}
	if a.List != nil {
		out.ListDelimiter = a.List.Delimiter
	}
	return out
}
