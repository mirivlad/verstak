# Reviewed local process API

The host supplies `api.processes` with capability `verstak/core/processes/v1`.
`process.spawn` is the existing process permission; `external.openURL` is a new,
separate permission for HTTP(S) handlers. Each Go entry point checks the loaded,
enabled plugin, current persisted disable state and declared capability/permission.
There is no unreviewed arbitrary-command endpoint.

```js
const request = {
  kind: 'process', executable: 'sshkeeper',
  args: ['connect', 'my-alias'], mode: 'detached'
};
// Call only in response to an explicit user gesture.
if (await api.processes.approved(request) || await api.processes.review(request)) {
  await api.processes.run(crypto.randomUUID(), request);
}
```

This is an infrastructure example, not an SSHKeeper integration. No profile,
credential, key, session or tunnel management is implemented.

`review` opens a host-owned native dialog, defaults to No, and records only
the exact normalized request after native confirmation. `run` never prompts and
rejects `approval_required` without a matching grant. `revoke` forgets a grant.
A definition with a different executable (including changed PATH resolution),
argv, cwd, launch mode or limits must be reviewed again. Titles and descriptions
are plugin metadata, not executable authority. Approvals bind the plugin and
local vault path. They persist outside the vault, in the host's device config (LocalAppData on Windows);
they never enter plugin settings, exports, backup restoration or Verstak sync.
Copying or tampering with the host's private config as the OS user is outside
this boundary. Approving a script path trusts the code stored at that path;
script/program bytes are not hashed or synchronized.

The process request has `executable`, a literal `args` array, an optional absolute
`cwd` (defaults to the user's home), `mode: 'wait' | 'detached'`, `timeoutMs`
(default 30,000, maximum 600,000) and `maxOutputBytes` (default 1 MiB, maximum
4 MiB across stdout and stderr). Executable names resolve through PATH; paths
must be absolute. No shell splitting, quoting, expansion, interpolation or
implicit interpreter is performed. An explicitly selected interpreter can run
its own language; it is shown in the review like any other executable.

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
reviewed application + file argument instead.

The shared plugin frontend is trusted in-process code, **not an authenticated
caller or security boundary**. A malicious plugin in this context can interfere
with another plugin or impersonate its frontend identity. Backend policy and
native per-request review constrain unreviewed execution; they do not create
isolation between mutually hostile frontend plugins. Strong isolation requires
a separate runtime/sandbox. Do not install untrusted plugins.

No process arguments, request bodies, output or native review contents are
written to application logs. Process errors use stable codes without echoing
sensitive arguments. Definitions themselves are ordinary synced plaintext;
never store passwords/tokens in argv or URLs.

## Other additive host contracts

- `ui.createIcon(name, size?)` creates an SVG from the existing built-in Lucide
  core vocabulary, inheriting currentColor; unknown names use the circle glyph.
- `ui.openView(viewId)` opens one of the calling plugin's declared views.
- `workspaceItems[].alwaysAvailable` defaults to false. A utility may opt into
  every Deal independently of recipe tool restrictions. Disabling the plugin
  still removes its contributions. Core contains no Actions-specific ID or data.
