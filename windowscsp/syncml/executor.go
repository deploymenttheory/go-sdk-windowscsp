package syncml

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
)

// SenderFunc delivers one SyncML request body to a device and returns the
// device's SyncML response. It is the only integration point an Executor
// needs: wrap your OMA-DM session, MDM proxy or test double here. The
// request is a <SyncBody> fragment; the response may be a <SyncBody>
// fragment or a full <SyncML> envelope — both are accepted.
type SenderFunc func(ctx context.Context, request string) (response string, err error)

// Executor is the live-exchange transport: a client.Client that makes
// SyncML fully transparent to callers. Every generated SDK call is rendered
// to SyncML, delivered through the SenderFunc, and the response is parsed
// back into typed results and errors — so reads (Get, List) return real
// values synchronously.
//
// Executor requires an endpoint that answers SyncML (an OMA-DM session, an
// MDM proxy, a simulator). To author a batch document offline, with nothing
// on the other end, use Recorder instead.
type Executor struct {
	send SenderFunc
}

// NewExecutor returns an Executor that exchanges SyncML via send.
func NewExecutor(send SenderFunc) *Executor {
	return &Executor{send: send}
}

// Get implements client.Client.
func (e *Executor) Get(ctx context.Context, uri string) (client.Value, error) {
	return e.roundTrip(ctx, Op{Verb: VerbGet, URI: uri}, true)
}

// List implements client.Client. OMA-DM enumerates an interior node's
// children as a "/"-separated list in the Get result.
func (e *Executor) List(ctx context.Context, uri string) ([]string, error) {
	v, err := e.roundTrip(ctx, Op{Verb: VerbGet, URI: uri}, true)
	if err != nil {
		return nil, err
	}
	var out []string
	for name := range strings.SplitSeq(v.Data, "/") {
		if name != "" {
			out = append(out, name)
		}
	}
	return out, nil
}

// Add implements client.Client.
func (e *Executor) Add(ctx context.Context, uri string, v client.Value) error {
	_, err := e.roundTrip(ctx, Op{Verb: VerbAdd, URI: uri, Value: v}, false)
	return err
}

// Replace implements client.Client.
func (e *Executor) Replace(ctx context.Context, uri string, v client.Value) error {
	_, err := e.roundTrip(ctx, Op{Verb: VerbReplace, URI: uri, Value: v}, false)
	return err
}

// Delete implements client.Client.
func (e *Executor) Delete(ctx context.Context, uri string) error {
	_, err := e.roundTrip(ctx, Op{Verb: VerbDelete, URI: uri}, false)
	return err
}

// Exec implements client.Client.
func (e *Executor) Exec(ctx context.Context, uri string, v client.Value) error {
	_, err := e.roundTrip(ctx, Op{Verb: VerbExec, URI: uri, Value: v}, false)
	return err
}

// roundTrip sends one operation (CmdID 1) and interprets the response
// status; when wantResult is set it also extracts the <Results> value.
func (e *Executor) roundTrip(ctx context.Context, op Op, wantResult bool) (client.Value, error) {
	resp, err := e.send(ctx, Document([]Op{op}))
	if err != nil {
		return client.Value{}, fmt.Errorf("syncml exchange for %s %s: %w", op.Verb, op.URI, err)
	}
	body, err := parseResponse(resp)
	if err != nil {
		return client.Value{}, fmt.Errorf("syncml response for %s %s: %w", op.Verb, op.URI, err)
	}

	code, ok := body.statusFor("1")
	if !ok {
		return client.Value{}, fmt.Errorf("syncml response for %s %s: no Status for CmdRef 1", op.Verb, op.URI)
	}
	if code < 200 || code > 299 {
		return client.Value{}, &client.StatusError{Code: code, URI: op.URI}
	}
	if !wantResult {
		return client.Value{}, nil
	}
	v, ok := body.resultFor("1")
	if !ok {
		return client.Value{}, fmt.Errorf("syncml response for Get %s: status %d but no Results item", op.URI, code)
	}
	return v, nil
}

// respBody is the subset of a response SyncBody the executor interprets.
type respBody struct {
	Status  []respStatus  `xml:"Status"`
	Results []respResults `xml:"Results"`
}

type respStatus struct {
	CmdRef string `xml:"CmdRef"`
	Cmd    string `xml:"Cmd"`
	Data   string `xml:"Data"`
}

type respResults struct {
	CmdRef string     `xml:"CmdRef"`
	Items  []respItem `xml:"Item"`
}

type respItem struct {
	Source struct {
		LocURI string `xml:"LocURI"`
	} `xml:"Source"`
	Meta struct {
		Format string `xml:"Format"`
	} `xml:"Meta"`
	Data string `xml:"Data"`
}

// parseResponse accepts a <SyncBody> fragment or a full <SyncML> envelope.
func parseResponse(doc string) (*respBody, error) {
	trimmed := strings.TrimSpace(doc)
	if trimmed == "" {
		return nil, fmt.Errorf("empty response")
	}
	if root, err := rootElement(trimmed); err != nil {
		return nil, err
	} else if root == "SyncML" {
		var full struct {
			Body respBody `xml:"SyncBody"`
		}
		if err := xml.Unmarshal([]byte(trimmed), &full); err != nil {
			return nil, err
		}
		return &full.Body, nil
	}
	var body respBody
	if err := xml.Unmarshal([]byte(trimmed), &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func rootElement(doc string) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("no root element: %w", err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start.Name.Local, nil
		}
	}
}

// statusFor returns the SyncML status code for a command, skipping the
// SyncHdr status (CmdRef 0).
func (b *respBody) statusFor(cmdRef string) (int, bool) {
	for _, s := range b.Status {
		if s.CmdRef != cmdRef || s.Cmd == "SyncHdr" {
			continue
		}
		code, err := strconv.Atoi(strings.TrimSpace(s.Data))
		if err != nil {
			continue
		}
		return code, true
	}
	return 0, false
}

// resultFor returns the returned Value for a command's <Results> item.
func (b *respBody) resultFor(cmdRef string) (client.Value, bool) {
	for _, r := range b.Results {
		if r.CmdRef != cmdRef || len(r.Items) == 0 {
			continue
		}
		item := r.Items[0]
		format := client.Format(strings.TrimSpace(item.Meta.Format))
		if format == "" {
			format = client.FormatChr
		}
		return client.Value{Format: format, Data: item.Data}, true
	}
	return client.Value{}, false
}
