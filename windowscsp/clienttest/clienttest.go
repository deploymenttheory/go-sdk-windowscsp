// Package clienttest provides an in-memory client.Client for testing code
// built on the generated CSP services: a fake OMA-DM tree keyed by OMA-URI.
package clienttest

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
)

// InMemory is a thread-safe fake CSP tree. The zero value is not usable;
// call New.
type InMemory struct {
	mu    sync.Mutex
	nodes map[string]client.Value
	// Executed records every Exec call, in order.
	Executed []ExecCall
}

// ExecCall is one recorded Exec operation.
type ExecCall struct {
	URI   string
	Value client.Value
}

// New returns an empty in-memory CSP tree.
func New() *InMemory {
	return &InMemory{nodes: map[string]client.Value{}}
}

// Seed pre-populates a node without going through Add semantics.
func (m *InMemory) Seed(uri string, v client.Value) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes[norm(uri)] = v
}

// Get implements client.Client.
func (m *InMemory) Get(_ context.Context, uri string) (client.Value, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.nodes[norm(uri)]
	if !ok {
		return client.Value{}, client.ErrNotFound
	}
	return v, nil
}

// List implements client.Client: it returns the distinct first path segments
// found directly beneath uri.
func (m *InMemory) List(_ context.Context, uri string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := norm(uri) + "/"
	seen := map[string]bool{}
	for path := range m.nodes {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			rest = rest[:i]
		}
		seen[rest] = true
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// Add implements client.Client.
func (m *InMemory) Add(_ context.Context, uri string, v client.Value) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes[norm(uri)] = v
	return nil
}

// Replace implements client.Client.
func (m *InMemory) Replace(_ context.Context, uri string, v client.Value) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes[norm(uri)] = v
	return nil
}

// Delete implements client.Client: removes the node and its subtree.
func (m *InMemory) Delete(_ context.Context, uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := norm(uri)
	found := false
	for path := range m.nodes {
		if path == target || strings.HasPrefix(path, target+"/") {
			delete(m.nodes, path)
			found = true
		}
	}
	if !found {
		return client.ErrNotFound
	}
	return nil
}

// Exec implements client.Client: it records the call.
func (m *InMemory) Exec(_ context.Context, uri string, v client.Value) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Executed = append(m.Executed, ExecCall{URI: norm(uri), Value: v})
	return nil
}

func norm(uri string) string { return strings.TrimRight(uri, "/") }
