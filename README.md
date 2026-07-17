# go-sdk-windowscsp

A Go SDK for **Windows Configuration Service Providers (CSPs)**, generated
from Microsoft's canonical [DDF v2](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf)
(Device Description Framework) metadata.

Every CSP and every Policy CSP area becomes a strongly-typed Go package with
**LCRUD** operations — `List`, `Create` (Add), `Get`, `Update` (Replace),
`Delete`, plus `Exec` — for every node the DDF declares, with OMA-URIs,
allowed-value constants and applicability metadata carried through from the
schema.

```
Microsoft DDF v2 zip ──fetchddf──▶ metadata/csp/*.json ──gencsp──▶ windowscsp/{csp,policy}/<area>/
   (download.microsoft.com)         (committed, reviewed)           (generated LCRUD packages)
```

## Usage

The SDK supports two distinct workflows; pick by whether a live endpoint
answers SyncML.

**Execute operations (SyncML fully transparent).** `syncml.Executor`
renders each typed call to SyncML, delivers it through your `SenderFunc`
(your OMA-DM session / MDM proxy), and parses the response back into typed
values and errors — callers never see SyncML, and reads return real values:

```go
c := windowscsp.NewClient(syncml.NewExecutor(mySender))

err := c.Policy.Camera.UpdateAllowCamera(ctx, camera.AllowCameraNotAllowed)
allow, err := c.Policy.Camera.GetAllowCamera(ctx) // typed result from the device
```

**Author a SyncML batch document (nothing executes).** With no endpoint,
`syncml.Recorder` queues the typed calls and the *document is the product*
— for Intune custom OMA-URI profiles or MDM delivery pipelines. A Recorder
is write-only (reads fail with `client.ErrDeferred`):

```go
rec := syncml.NewRecorder()
c := windowscsp.NewClient(rec)

_ = c.Policy.Camera.UpdateAllowCamera(ctx, camera.AllowCameraNotAllowed)
_ = c.CSP.VPNv2.CreateProfileName(ctx, "Corp")
_ = c.CSP.Reboot.ExecRebootNow(ctx)

doc := rec.Document() // the deliverable: a <SyncBody> batch for your MDM
```

- `windowscsp.NewClient(transport)` fans one transport out to every service:
  `c.CSP.<Name>` for the ~60 standalone CSPs, `c.Policy.<Area>` for the ~250
  Policy areas (including the `ADMX_*` backed ones).
- Dynamic (runtime-named) nodes become `string` parameters
  (`GetProfileNameServer(ctx, profileName)`), and their parents get `List`.
- Raw OMA-URIs are exported per package (`camera.URIAllowCamera`,
  `vpnv2.URIProfileNameServer("Corp")`) for Intune custom-OMA-URI scenarios.

Runnable, offline examples live under [`examples/<domain>/<example>`](examples/)
— e.g. `go run ./examples/vpnv2/manage_profiles`.

### Custom OMA-URIs

OMA-URIs are first-class in both directions
(see `examples/uris/custom_oma_uri`):

- **Generate a URI from the SDK** — every generated package exports its
  OMA-URIs: constants for static nodes (`camera.URIAllowCamera`) and builder
  functions for dynamic nodes (`vpnv2.URIProfileNameAlwaysOn("Corp")`).
  These are exactly what an Intune custom OMA-URI profile asks for
  (URI + data type + value).
- **Operate on a URI the SDK doesn't cover** — the generated services are a
  convenience layer, not a cage. Every transport implements
  `client.Client`, whose six verbs take plain URI strings, so OEM /
  third-party CSP nodes (or nodes newer than the pinned DDF release) work
  through the same transports:

  ```go
  rec.Replace(ctx, "./Device/Vendor/OEM/ContosoAgent/TelemetryLevel", client.Int(2))
  v, err := exec.Get(ctx, "./Device/Vendor/OEM/ContosoAgent/Status") // typed Value back
  ```

### Transports

Anything implementing the six-verb `client.Client` interface:

| Transport | Workflow | Purpose |
|---|---|---|
| `syncml.Executor` | execute | live SyncML exchange, fully transparent: calls rendered, responses parsed into typed results |
| `syncml.Recorder` | author | offline batch authoring: queue typed writes, emit the SyncML document (write-only) |
| `clienttest.InMemory` | test | fake CSP tree for unit tests |
| yours | execute | an MDM server session, or the local MDM WMI bridge (`root\cimv2\mdm\dmmap`) |

## The pipeline

| Stage | Command | Output |
|---|---|---|
| Acquire | `go run ./cmd/fetchddf` | `metadata/csp/*.json` + `PROVENANCE.json` |
| Generate | `go run ./cmd/gencsp` | `windowscsp/csp/`, `windowscsp/policy/`, `windowscsp/registry.go` |

`fetchddf` downloads the **pinned** DDF v2 release (currently *February
2026*), verifies its SHA-256 and writes one JSON snapshot per CSP. The
snapshots are committed; codegen is offline and deterministic from them.
`-zip` parses a local drop, `-discover` scrapes Microsoft Learn for a newer
one.

Generated packages separate concerns per file: `doc.go`,
`<pkg>_service.go` (service struct + constructor), `<pkg>_uris.go`
(OMA-URI constants/builders), `<pkg>_crud.go` (LCRUD methods),
`<pkg>_enums.go` (allowed-value constants). Every generated file starts with
the `DO NOT EDIT` marker, which also drives stale-file pruning.

## CI

- **CI** (`ci.yml`) — build + vet + tests on Ubuntu and Windows, plus a
  *regeneration determinism gate*: `gencsp` re-runs from the committed
  snapshots and the build fails if the output differs from the committed
  tree by a byte.
- **DDF update** (`ddf-update.yml`) — weekly (and on demand): discovers the
  current Microsoft drop, refreshes snapshots + generated code, and opens a
  PR when anything changed.

## Development

```sh
go test ./cmd/... ./internal/... ./windowscsp/...   # unit + golden + smoke tests
go run ./cmd/gencsp                                  # regenerate after template/builder changes
go test ./internal/codegen -run TestGolden -update   # refresh codegen golden files
```

Architecture notes live in [docs/implementation-plan.md](docs/implementation-plan.md).
The design borrows deliberately: DDF acquisition/snapshots from
`go-bindings-wmi`, the client/registry shape from `go-sdk-jamfpro-v2`, and
the template-firewalled, prune-and-diff codegen from `go-bindings-win32`.
