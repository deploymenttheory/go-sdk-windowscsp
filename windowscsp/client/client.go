// Package client defines the transport seam every generated CSP service
// depends on: six OMA-DM verbs against OMA-URIs, carrying typed Values.
//
// The SDK ships two implementations — clienttest.InMemory (a fake CSP tree
// for tests) and syncml.Recorder (batches write operations into an OMA-DM
// SyncML document). Real executors (an MDM server session, the local MDM
// WMI bridge in root\cimv2\mdm\dmmap) implement this interface out of tree.
package client

import "context"

// Client executes OMA-DM operations against a CSP tree.
//
// The generated services are a typed convenience layer over this interface;
// the interface itself accepts any OMA-URI string. To address nodes the SDK
// does not cover (OEM / third-party CSPs, or nodes newer than the pinned
// DDF release), call a transport's verbs directly with a hand-written URI
// and a typed Value — see examples/uris/custom_oma_uri.
type Client interface {
	// Get reads a leaf node's value.
	Get(ctx context.Context, uri string) (Value, error)
	// List enumerates the child node names of an interior node.
	List(ctx context.Context, uri string) ([]string, error)
	// Add creates a node. Interior (container) nodes are created with a
	// node-format Value; leaves carry their initial value.
	Add(ctx context.Context, uri string, value Value) error
	// Replace overwrites a leaf node's value.
	Replace(ctx context.Context, uri string, value Value) error
	// Delete removes a node (and, for interior nodes, its subtree).
	Delete(ctx context.Context, uri string) error
	// Exec triggers a node that supports execution. Nodes with the DDF
	// "null" format take an empty (FormatNull) value.
	Exec(ctx context.Context, uri string, value Value) error
}
