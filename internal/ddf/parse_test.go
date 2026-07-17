package ddf

import (
	"reflect"
	"testing"
)

// fixture is a synthetic DDF v2 document exercising the parser's edge cases:
// empty-element choices, dynamic nodes, enums, applicability, deprecation,
// GP mapping and list values.
const fixture = `<?xml version="1.0" encoding="UTF-8"?>
<MgmtTree xmlns:MSFT="http://schemas.microsoft.com/MobileDevice/DM">
  <VerDTD>1.2</VerDTD>
  <MSFT:Diagnostics></MSFT:Diagnostics>
  <Node>
    <NodeName>Demo</NodeName>
    <Path>./Device/Vendor/MSFT</Path>
    <DFProperties>
      <AccessType><Get /></AccessType>
      <Description>Root node.</Description>
      <DFFormat><node /></DFFormat>
      <Occurrence><One /></Occurrence>
      <Scope><Permanent /></Scope>
      <DFType><MIME /></DFType>
      <MSFT:Applicability>
        <MSFT:OsBuildVersion>10.0.19041</MSFT:OsBuildVersion>
        <MSFT:CspVersion>1.0</MSFT:CspVersion>
        <MSFT:EditionAllowList>0x4;0x30;</MSFT:EditionAllowList>
      </MSFT:Applicability>
    </DFProperties>
    <Node>
      <NodeName>AllowThing</NodeName>
      <DFProperties>
        <AccessType><Add /><Delete /><Get /><Replace /></AccessType>
        <Description>Allows the thing. Most restricted value is 0.</Description>
        <DFFormat><int /></DFFormat>
        <Occurrence><ZeroOrOne /></Occurrence>
        <DFType><MIME>text/plain</MIME></DFType>
        <DefaultValue>1</DefaultValue>
        <MSFT:AllowedValues ValueType="ENUM">
          <MSFT:Enum>
            <MSFT:Value>0</MSFT:Value>
            <MSFT:ValueDescription>Not allowed.</MSFT:ValueDescription>
          </MSFT:Enum>
          <MSFT:Enum>
            <MSFT:Value>1</MSFT:Value>
            <MSFT:ValueDescription>Allowed.</MSFT:ValueDescription>
          </MSFT:Enum>
        </MSFT:AllowedValues>
        <MSFT:GpMapping GpEnglishName="L_AllowThing" GpAreaPath="Demo~AT~Components" />
        <MSFT:ConflictResolution>LowestValueMostSecure</MSFT:ConflictResolution>
      </DFProperties>
    </Node>
    <Node>
      <NodeName>OldSetting</NodeName>
      <DFProperties>
        <AccessType><Get /><Replace /></AccessType>
        <DFFormat><chr /></DFFormat>
        <Occurrence><One /></Occurrence>
        <DFType><MIME>text/plain</MIME></DFType>
        <MSFT:Deprecated OsBuildDeprecated="10.0.22621" />
        <MSFT:AllowedValues ValueType="None"></MSFT:AllowedValues>
      </DFProperties>
    </Node>
    <Node>
      <NodeName>Profiles</NodeName>
      <DFProperties>
        <AccessType><Get /></AccessType>
        <DFFormat><node /></DFFormat>
        <Occurrence><One /></Occurrence>
        <DFType><DDFName /></DFType>
      </DFProperties>
      <Node>
        <NodeName></NodeName>
        <DFProperties>
          <AccessType><Add /><Delete /><Get /></AccessType>
          <Description>A profile, named by the server.</Description>
          <DFFormat><node /></DFFormat>
          <Occurrence><ZeroOrMore /></Occurrence>
          <DFTitle>ProfileName</DFTitle>
          <DFType><DDFName /></DFType>
          <MSFT:DynamicNodeNaming>
            <MSFT:UniqueName>[a-zA-Z0-9]+</MSFT:UniqueName>
          </MSFT:DynamicNodeNaming>
        </DFProperties>
        <Node>
          <NodeName>Server</NodeName>
          <DFProperties>
            <AccessType><Add /><Get /><Replace /></AccessType>
            <DFFormat><chr /></DFFormat>
            <Occurrence><One /></Occurrence>
            <DFType><MIME>text/plain</MIME></DFType>
            <MSFT:AllowedValues ValueType="None">
              <MSFT:List Delimiter=";" />
            </MSFT:AllowedValues>
          </DFProperties>
        </Node>
      </Node>
    </Node>
    <Node>
      <NodeName>DoIt</NodeName>
      <DFProperties>
        <AccessType><Exec /></AccessType>
        <Description>Executes the thing.</Description>
        <DFFormat><null /></DFFormat>
        <Occurrence><One /></Occurrence>
        <DFType><MIME /></DFType>
        <MSFT:RebootBehavior>Automatic</MSFT:RebootBehavior>
      </DFProperties>
    </Node>
  </Node>
</MgmtTree>`

func TestParse(t *testing.T) {
	csps, err := Parse([]byte(fixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(csps) != 1 {
		t.Fatalf("got %d CSPs, want 1", len(csps))
	}
	csp := csps[0]
	if csp.Name != "Demo" || csp.Path != "./Device/Vendor/MSFT" {
		t.Fatalf("root = %q %q", csp.Name, csp.Path)
	}
	if csp.PolicyArea() {
		t.Errorf("PolicyArea() = true for standalone CSP")
	}
	if len(csp.Nodes) != 4 {
		t.Fatalf("got %d top nodes, want 4", len(csp.Nodes))
	}

	// Children are sorted by name.
	names := make([]string, 0, 4)
	for _, n := range csp.Nodes {
		names = append(names, n.Name)
	}
	want := []string{"AllowThing", "DoIt", "OldSetting", "Profiles"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("top-level order = %v, want %v", names, want)
	}

	allow := csp.Nodes[0]
	if allow.Path != "./Device/Vendor/MSFT/Demo/AllowThing" {
		t.Errorf("AllowThing path = %q", allow.Path)
	}
	if allow.Format != "int" || allow.Default != "1" || allow.Occurrence != "ZeroOrOne" {
		t.Errorf("AllowThing basics = %q %q %q", allow.Format, allow.Default, allow.Occurrence)
	}
	if got := allow.Access; !reflect.DeepEqual(got, []string{"Add", "Delete", "Get", "Replace"}) {
		t.Errorf("AllowThing access = %v", got)
	}
	if allow.GoType() != "int64" || !allow.Leaf() {
		t.Errorf("AllowThing GoType/Leaf = %q %v", allow.GoType(), allow.Leaf())
	}
	if allow.AllowedValues == nil || allow.AllowedValues.Type != "ENUM" || len(allow.AllowedValues.Enum) != 2 {
		t.Fatalf("AllowThing allowed values = %+v", allow.AllowedValues)
	}
	if e := allow.AllowedValues.Enum[0]; e.Value != "0" || e.Description != "Not allowed." {
		t.Errorf("enum[0] = %+v", e)
	}
	if allow.GpMapping == nil || allow.GpMapping.EnglishName != "L_AllowThing" {
		t.Errorf("GpMapping = %+v", allow.GpMapping)
	}

	doIt := csp.Nodes[1]
	if doIt.Format != "null" || doIt.GoType() != "" || !doIt.Leaf() {
		t.Errorf("DoIt = format %q gotype %q leaf %v", doIt.Format, doIt.GoType(), doIt.Leaf())
	}
	if !reflect.DeepEqual(doIt.Access, []string{"Exec"}) {
		t.Errorf("DoIt access = %v", doIt.Access)
	}
	if doIt.RebootBehavior != "Automatic" {
		t.Errorf("DoIt reboot = %q", doIt.RebootBehavior)
	}

	old := csp.Nodes[2]
	if old.DeprecatedOSBuild != "10.0.22621" {
		t.Errorf("OldSetting deprecated = %q", old.DeprecatedOSBuild)
	}

	profiles := csp.Nodes[3]
	if len(profiles.Children) != 1 {
		t.Fatalf("Profiles children = %d", len(profiles.Children))
	}
	dyn := profiles.Children[0]
	if !dyn.Dynamic() {
		t.Fatalf("dynamic node not detected: %+v", dyn)
	}
	if dyn.Title != "ProfileName" || dyn.DynamicNaming != "[a-zA-Z0-9]+" {
		t.Errorf("dynamic title/naming = %q %q", dyn.Title, dyn.DynamicNaming)
	}
	if dyn.Path != "./Device/Vendor/MSFT/Demo/Profiles/{ProfileName}" {
		t.Errorf("dynamic path = %q", dyn.Path)
	}
	if len(dyn.Children) != 1 || dyn.Children[0].Path != "./Device/Vendor/MSFT/Demo/Profiles/{ProfileName}/Server" {
		t.Errorf("dynamic child path = %+v", dyn.Children)
	}
	if av := dyn.Children[0].AllowedValues; av == nil || av.ListDelimiter != ";" {
		t.Errorf("list delimiter = %+v", av)
	}

	// Root applicability propagates onto the parsed root's nodes only via
	// each node's own tags; the root node's applicability is not modelled on
	// CSP. AllowThing has none.
	if allow.Applicability != nil {
		t.Errorf("AllowThing applicability = %+v", allow.Applicability)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse([]byte("<not-ddf/>")); err == nil {
		t.Fatal("expected error for non-DDF input")
	}
}

// multiRootFixture mirrors the 87 real DDF files (VPNv2, most ADMX areas…)
// that carry separate User- and Device-scoped trees as sibling root Nodes.
const multiRootFixture = `<?xml version="1.0" encoding="UTF-8"?>
<MgmtTree xmlns:MSFT="http://schemas.microsoft.com/MobileDevice/DM">
  <VerDTD>1.2</VerDTD>
  <Node>
    <NodeName>Demo</NodeName>
    <Path>./User/Vendor/MSFT</Path>
    <DFProperties>
      <AccessType><Get /></AccessType>
      <DFFormat><node /></DFFormat>
    </DFProperties>
    <Node>
      <NodeName>UserOnly</NodeName>
      <DFProperties>
        <AccessType><Get /></AccessType>
        <DFFormat><int /></DFFormat>
      </DFProperties>
    </Node>
  </Node>
  <Node>
    <NodeName>Demo</NodeName>
    <Path>./Device/Vendor/MSFT</Path>
    <DFProperties>
      <AccessType><Get /></AccessType>
      <DFFormat><node /></DFFormat>
    </DFProperties>
    <Node>
      <NodeName>DeviceOnly</NodeName>
      <DFProperties>
        <AccessType><Get /></AccessType>
        <DFFormat><int /></DFFormat>
      </DFProperties>
    </Node>
  </Node>
</MgmtTree>`

// TestParseMultipleRoots guards against the encoding/xml pitfall where
// sibling root Nodes decode into one struct, merging the User tree's
// children under the Device path with wrong URIs.
func TestParseMultipleRoots(t *testing.T) {
	csps, err := Parse([]byte(multiRootFixture))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(csps) != 2 {
		t.Fatalf("got %d CSPs, want 2", len(csps))
	}

	user, device := csps[0], csps[1]
	if user.Path != "./User/Vendor/MSFT" || device.Path != "./Device/Vendor/MSFT" {
		t.Fatalf("paths = %q, %q", user.Path, device.Path)
	}
	if len(user.Nodes) != 1 || user.Nodes[0].Name != "UserOnly" {
		t.Fatalf("user tree = %+v", user.Nodes)
	}
	if len(device.Nodes) != 1 || device.Nodes[0].Name != "DeviceOnly" {
		t.Fatalf("device tree = %+v", device.Nodes)
	}
	if user.Nodes[0].Path != "./User/Vendor/MSFT/Demo/UserOnly" {
		t.Errorf("user leaf path = %q", user.Nodes[0].Path)
	}
	if device.Nodes[0].Path != "./Device/Vendor/MSFT/Demo/DeviceOnly" {
		t.Errorf("device leaf path = %q", device.Nodes[0].Path)
	}
}
