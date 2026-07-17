// Package windowscsp is a Go SDK for Windows Configuration Service
// Providers (CSPs), generated from Microsoft's DDF v2 metadata.
//
// Construct a Client around any transport implementing client.Client, then
// reach every CSP through the family registries:
//
//	c := windowscsp.NewClient(transport)
//	err := c.Policy.Camera.UpdateAllowCamera(ctx, 0)
//	err = c.CSP.Reboot.ExecRebootNow(ctx)
//
// Transports in this module, by workflow:
//
//   - syncml.Executor — execute against a live SyncML endpoint; SyncML is
//     rendered, delivered and parsed transparently, reads return values.
//   - syncml.Recorder — author a SyncML batch document offline; calls are
//     queued (never executed) and the document is the product. Write-only.
//   - clienttest.InMemory — fake CSP tree for tests.
//
// Device- or server-side executors implement client.Client out of tree.
//
// The generated surface (csp/, policy/, registry.go) is produced by
// cmd/gencsp from the committed snapshots under metadata/csp; do not edit
// generated files by hand.
package windowscsp
