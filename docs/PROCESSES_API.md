# Local process API and action trust

The host supplies `api.processes` with capability `verstak/core/processes/v1`.
`process.spawn` is the existing process permission; `external.openURL` is a new,
separate permission for HTTP(S) handlers. Each Go entry point checks the loaded,
enabled plugin, current persisted disable state and declared capability/permission.
A declared storage flag or action trust record never grants these permissions.

```js
const request = {
  kind: 'process', executable: 'sshkeeper',
  args: ['connect', 'my-alias'], mode: 'detached'
};
// Call only in response to an explicit user gesture.
await api.processes.run(crypto.randomUUID(), request);
// Optional confirmation, when the workflow needs it:
if (await api.processes.confirm(request)) {
  await api.processes.run(crypto.randomUUID(), request);
}
```

This is an infrastructure example, not an SSHKeeper integration. No profile,
credential, key, session or tunnel management is implemented.

`run` checks current plugin access and never prompts. `confirm` is an optional
host-owned native dialog displaying program, individual quoted arguments and
working directory (or URL), with No as the default. It grants no permissions
and writes no request approval. Disable/reload/vault changes expire an open
confirmation. `approved`, `review` and `revoke` remain deprecated compatibility
methods for the old local request-review store; `run` no longer consults it.
No API lets a plugin grant itself process permissions.

Actions owns the decision to confirm a particular definition. Its form save is
an explicit local authorship boundary: after saving, the plugin records a hash
of authored execution fields plus confirmEveryRun. Public save commands,
sync/import writes and record events never call that form-only trust operation.
On launch Actions compares the current definition to that device-local hash.
An external new record or changed executable/argv/cwd/mode/interpreter/URL needs
confirmation; title, icon, description and ordering do not. The checkbox
confirmEveryRun requests confirmation every time, including trusted actions.
The record is read again after confirmation before its frozen request runs.

`storage.local.read/write(name, data)` is generic plugin/vault-scoped JSON
outside the vault, under the device config directory (LocalAppData on Windows).
It requires current storage.namespace access and cannot change process rights.
It is not a secret store. Actions saves only record IDs and SHA-256 hashes here.
This data is never a record operation, vault backup or import destination.
Changing device or vault path starts without local trust. Existing definitions
remain valid, default confirmEveryRun to false, and need one confirmation when
this device has no form-save/confirmation hash; old request grants and imported
approved/trusted fields are not promoted into action trust.

The actual sync boundary is `recordPluginRecordChanges` for local settings
writes versus `applyRemotePluginRecordOp` calling storage directly and then
publishing plugin.records.changed. No source is inferred from synced fields.
The current import service is confined to its vault target; it cannot populate
storage.local. No new import UI or generic provenance framework is introduced.

The process request has `executable`, a literal `args` array, an optional absolute
`cwd` (defaults to the user's home), `mode: 'wait' | 'detached'`, `timeoutMs`
(default 30,000, maximum 600,000) and `maxOutputBytes` (default 1 MiB, maximum
4 MiB across stdout and stderr). Executable names resolve through PATH; paths
must be absolute. No shell splitting, quoting, expansion, interpolation or
implicit interpreter is performed. An explicitly selected interpreter can run
its own language; optional confirmation shows it like any other executable.

Waited jobs return started/exitCode/stdout/stderr/timedOut/cancelled/outputLimited.
Output overflow terminates the job; it does not merely hide additional bytes.
Invalid UTF-8 is replaced with `?`. `cancel(operationId)` cancels only that
plugin's operation. At most four waited jobs per plugin, sixteen overall, can
run concurrently. Disable, plugin reload, vault change and shutdown cancel
waited jobs. Detached GUI processes return immediately (exitCode -1), have no
captured output and are reaped when they exit; they intentionally survive
navigation, plugin disable and host exit.

Linux/macOS use a separate process group for a waited job and kill that group.
Windows uses a suspended process assigned to a kill-on-close Job Object before
resuming its first thread. Failure to establish process control fails closed.
A malicious executable that deliberately escapes its process group on Unix is
outside the process-lifecycle guarantee. This is not an OS sandbox.

URL requests are `{ kind: 'url', url: 'https://example.org/' }` and use the
existing OS opening service (`xdg-open`, `open`, or `rundll32` with separate
arguments). Credentials in URLs and non-HTTP(S) schemes are rejected. Local
file associations are deliberately not exposed: an executable, shortcut,
`.desktop`, script or registered handler may execute code. Use an explicit
application + file argument instead.

The shared plugin frontend is trusted in-process code, **not an authenticated
caller or security boundary**. A malicious plugin in this context can interfere
with another plugin or impersonate its frontend identity. Backend permission/capability/lifecycle checks constrain API access;
a permission-bearing plugin is authorized to execute. Action trust does not
create isolation between mutually hostile frontend plugins. Strong isolation requires
a separate runtime/sandbox. Do not install untrusted plugins.

No process arguments, request bodies, output or native review contents are
written to application logs. Process errors use stable codes without echoing
sensitive arguments. Definitions themselves are ordinary synced plaintext;
never store passwords/tokens in argv or URLs.

## Other additive host contracts

- `ui.createIcon(name, size?)` creates an SVG from the existing built-in Lucide
  catalogue (lazy sprite for non-core icons), inheriting currentColor; unknown names use the circle glyph.
- `ui.pickIcon(current?)` opens the shared searchable icon modal, returning a name or null.
- `ui.openView(viewId)` opens one of the calling plugin's declared views.
- `workspaceItems[].alwaysAvailable` defaults to false. A utility may opt into
  every Deal independently of recipe tool restrictions. Disabling the plugin
  still removes its contributions. Core contains no Actions-specific ID or data.
