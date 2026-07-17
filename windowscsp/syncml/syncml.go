// Package syncml renders CSP operations as OMA-DM SyncML, the wire format
// MDM servers (and Intune custom OMA-URI profiles) use to talk to Windows
// devices.
//
// # Choosing a transport
//
// The package supports two distinct workflows. Pick by answering one
// question: is there a live endpoint that answers SyncML?
//
//   - YES — use [Executor]. It implements client.Client over a live
//     request/response exchange: every generated SDK call is rendered to
//     SyncML, delivered through the [SenderFunc] you supply, and the
//     device's response is parsed back into typed values and errors. SyncML
//     is fully transparent to callers; Get and List return real values.
//
//   - NO — use [Recorder]. It is an offline *document authoring* tool: the
//     typed SDK calls are queued, never executed, and the product is the
//     SyncML batch document itself (Recorder.Document) — ready to paste
//     into an Intune custom OMA-URI profile or feed to an MDM delivery
//     pipeline. Because nothing answers, reads (Get/List) cannot return
//     data and fail with client.ErrDeferred; a Recorder is write-only.
//
// [Document] is the low-level shared renderer: it turns a batch of Ops into
// a <SyncBody> fragment directly, without going through the typed services.
package syncml

import (
	"context"
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
)

// Verb is a SyncML command name.
type Verb string

const (
	VerbGet     Verb = "Get"
	VerbAdd     Verb = "Add"
	VerbReplace Verb = "Replace"
	VerbDelete  Verb = "Delete"
	VerbExec    Verb = "Exec"
	VerbAtomic  Verb = "Atomic"
)

// Op is one SyncML command against an OMA-URI.
type Op struct {
	Verb  Verb
	URI   string
	Value client.Value
}

// Document renders ops as a SyncBody XML fragment with sequential CmdIDs,
// ending with <Final/>. The fragment is what an OMA-DM session embeds in its
// SyncML envelope.
func Document(ops []Op) string {
	var b strings.Builder
	b.WriteString("<SyncBody>\n")
	for i, op := range ops {
		writeOp(&b, i+1, op)
	}
	b.WriteString("  <Final/>\n</SyncBody>\n")
	return b.String()
}

func writeOp(b *strings.Builder, cmdID int, op Op) {
	fmt.Fprintf(b, "  <%s>\n    <CmdID>%d</CmdID>\n    <Item>\n      <Target>\n        <LocURI>%s</LocURI>\n      </Target>\n", op.Verb, cmdID, escape(op.URI))
	if hasPayloadMeta(op) {
		fmt.Fprintf(b, "      <Meta>\n        <Format xmlns=\"syncml:metinf\">%s</Format>\n        <Type xmlns=\"syncml:metinf\">text/plain</Type>\n      </Meta>\n", op.Value.Format)
	}
	if hasData(op) {
		fmt.Fprintf(b, "      <Data>%s</Data>\n", escape(op.Value.Data))
	}
	fmt.Fprintf(b, "    </Item>\n  </%s>\n", op.Verb)
}

// hasPayloadMeta reports whether the op carries a Meta/Format block: any
// value-bearing format, plus explicit node-format Adds (container creation).
func hasPayloadMeta(op Op) bool {
	switch op.Verb {
	case VerbGet, VerbDelete:
		return false
	}
	switch op.Value.Format {
	case "", client.FormatNull:
		return false
	}
	return true
}

func hasData(op Op) bool {
	return hasPayloadMeta(op) && op.Value.Format != client.FormatNode
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// Recorder is the offline document-authoring transport: it implements
// client.Client by queueing operations instead of executing them, so the
// strongly-typed generated services can be used to build a SyncML batch
// without a device. The end product is Recorder.Document — the operations
// themselves never run until that document is delivered by an MDM.
//
// A Recorder is write-only. Reads (Get, List) are queued as Get commands
// for the eventual recipient but return client.ErrDeferred / no results,
// because nothing is connected to answer them. If you have a live SyncML
// endpoint and want reads to work, use Executor instead.
type Recorder struct {
	ops []Op
}

// NewRecorder returns an empty Recorder. See the package documentation for
// when to choose Recorder (offline batch authoring) over Executor (live,
// transparent SyncML exchange).
func NewRecorder() *Recorder { return &Recorder{} }

// Ops returns the queued operations in call order.
func (r *Recorder) Ops() []Op { return r.ops }

// Document renders the queued operations; see Document.
func (r *Recorder) Document() string { return Document(r.ops) }

// Get implements client.Client by queueing a Get command. The returned
// error is client.ErrDeferred: results arrive only when the batch runs on a
// device.
func (r *Recorder) Get(_ context.Context, uri string) (client.Value, error) {
	r.ops = append(r.ops, Op{Verb: VerbGet, URI: uri})
	return client.Value{}, client.ErrDeferred
}

// List implements client.Client by queueing a Get command on the interior
// node; the child list is deferred.
func (r *Recorder) List(_ context.Context, uri string) ([]string, error) {
	r.ops = append(r.ops, Op{Verb: VerbGet, URI: uri})
	return nil, client.ErrDeferred
}

// Add implements client.Client.
func (r *Recorder) Add(_ context.Context, uri string, v client.Value) error {
	r.ops = append(r.ops, Op{Verb: VerbAdd, URI: uri, Value: v})
	return nil
}

// Replace implements client.Client.
func (r *Recorder) Replace(_ context.Context, uri string, v client.Value) error {
	r.ops = append(r.ops, Op{Verb: VerbReplace, URI: uri, Value: v})
	return nil
}

// Delete implements client.Client.
func (r *Recorder) Delete(_ context.Context, uri string) error {
	r.ops = append(r.ops, Op{Verb: VerbDelete, URI: uri})
	return nil
}

// Exec implements client.Client.
func (r *Recorder) Exec(_ context.Context, uri string, v client.Value) error {
	r.ops = append(r.ops, Op{Verb: VerbExec, URI: uri, Value: v})
	return nil
}
