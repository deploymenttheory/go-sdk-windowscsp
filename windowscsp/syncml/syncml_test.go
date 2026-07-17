package syncml

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
)

func TestDocument(t *testing.T) {
	ops := []Op{
		{Verb: VerbReplace, URI: "./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera", Value: client.Int(0)},
		{Verb: VerbExec, URI: "./Device/Vendor/MSFT/Reboot/RebootNow", Value: client.Null()},
		{Verb: VerbDelete, URI: "./Device/Vendor/MSFT/VPNv2/P1"},
		{Verb: VerbAdd, URI: "./Device/Vendor/MSFT/VPNv2/P2", Value: client.Node()},
	}
	doc := Document(ops)

	for _, want := range []string{
		"<SyncBody>",
		"<CmdID>1</CmdID>",
		"<CmdID>4</CmdID>",
		"<LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI>",
		`<Format xmlns="syncml:metinf">int</Format>`,
		"<Data>0</Data>",
		`<Format xmlns="syncml:metinf">node</Format>`,
		"<Final/>",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("document missing %q:\n%s", want, doc)
		}
	}
	// Exec on a null-format node and Delete carry no Meta/Data.
	if strings.Count(doc, "<Meta>") != 2 {
		t.Errorf("expected 2 Meta blocks (Replace + node Add):\n%s", doc)
	}
	// A node-format Add has Meta but no Data payload.
	if strings.Count(doc, "<Data>") != 1 {
		t.Errorf("expected 1 Data block:\n%s", doc)
	}
}

func TestDocumentEscapes(t *testing.T) {
	doc := Document([]Op{{Verb: VerbReplace, URI: "./x", Value: client.XML(`<a b="1">&</a>`)}})
	if !strings.Contains(doc, "&lt;a b=&quot;1&quot;&gt;&amp;&lt;/a&gt;") {
		t.Errorf("payload not escaped:\n%s", doc)
	}
}

func TestRecorder(t *testing.T) {
	r := NewRecorder()
	ctx := context.Background()

	if err := r.Replace(ctx, "./a", client.Int(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(ctx, "./b"); !errors.Is(err, client.ErrDeferred) {
		t.Fatalf("Get error = %v, want ErrDeferred", err)
	}
	if err := r.Exec(ctx, "./c", client.Null()); err != nil {
		t.Fatal(err)
	}

	ops := r.Ops()
	if len(ops) != 3 || ops[0].Verb != VerbReplace || ops[1].Verb != VerbGet || ops[2].Verb != VerbExec {
		t.Fatalf("ops = %+v", ops)
	}
	if !strings.Contains(r.Document(), "<CmdID>3</CmdID>") {
		t.Error("Document() should render all queued ops")
	}
}
