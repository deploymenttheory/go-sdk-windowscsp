# Examples

Runnable examples, one directory per CSP domain, one subdirectory per
scenario (`examples/<domain>/<example>/main.go`):

```sh
go run ./examples/reboot/exec_reboot_now
```

Every example is self-contained and runs **offline** — but they demonstrate
two deliberately different workflows. Know which one you are looking at:

## Workflow 1 — execute operations (SyncML invisible)

There is a live endpoint that answers SyncML (an OMA-DM session, an MDM
proxy, a simulator). Use **`syncml.Executor`**: every typed SDK call is
rendered to SyncML, delivered through your `SenderFunc`, and the response
is parsed back into typed values and errors. Callers never see SyncML, and
reads (`Get`/`List`) return real values.

→ `transport/syncml_executor` is the canonical example of this workflow.

## Workflow 2 — author a SyncML batch document (nothing executes)

There is **no** endpoint; the *document itself is the product*. Use
**`syncml.Recorder`**: typed calls are queued, never executed, and
`rec.Document()` emits the `<SyncBody>` batch to paste into an Intune
custom OMA-URI profile or feed to an MDM delivery pipeline. A Recorder is
write-only — reads fail with `client.ErrDeferred` because nothing answers.

→ `camera/disable_camera`, `reboot/exec_reboot_now` and `defender/run_scan`
are authoring examples: they *print a document*, they do not change state.

## Test transport

`clienttest.InMemory` is a fake CSP tree used where an example wants to
read back what it wrote, without inventing a device.

| Example | Workflow | Shows |
|---|---|---|
| `transport/syncml_executor` | execute | Transparent SyncML: typed calls in, typed results out, wire format invisible |
| `transport/logging_transport` | execute | Writing your own `client.Client` transport |
| `camera/disable_camera` | author | Policy area + generated enum constants → SyncML document |
| `reboot/exec_reboot_now` | author | Exec on a node → SyncML document |
| `defender/run_scan` | author | Exec with a payload value → SyncML document |
| `reboot/schedule_daily_reboot` | test fake | Update + Get round-trip on leaf values |
| `vpnv2/manage_profiles` | test fake | Dynamic (runtime-named) nodes: Create/List/Update/Delete |
| `laps/reset_local_admin_password` | test fake | Configure policies, trigger an action, poll its status |
| `dmclient/inspect_enrollments` | test fake | Enumerating dynamic children and reading per-instance values |
| `uris/custom_oma_uri` | author | Generating OMA-URIs from the SDK (Intune custom OMA-URI rows) and operating on hand-written URIs the SDK doesn't cover |

To run against something real, implement `client.Client` (six OMA-DM verbs)
over your MDM session or the local MDM WMI bridge and pass it to
`windowscsp.NewClient` — `transport/logging_transport` shows the seam.
