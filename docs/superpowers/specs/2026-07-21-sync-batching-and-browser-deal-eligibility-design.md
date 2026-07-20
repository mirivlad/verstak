# Sync Batching and Browser Deal Eligibility Design

Date: 2026-07-21

## Goal

Make a paired desktop vault complete its first synchronization when the local
operation queue is larger than the server push limit, keep the paired account
state understandable without exposing a password, and prevent Browser Inbox
from assigning captures to Deals where its workspace tool is inactive.

## Confirmed Causes

- The current vault has 103 unpushed operations. The server is configured with
  `max_push_operations: 100`, while the desktop sends the complete queue in
  one request. Pairing succeeds, but the first push fails with
  `sync-server:too_many_operations` and HTTP 413.
- The Sync settings component explicitly clears both credential fields after
  pairing. A later settings write persists the cleared username.
- `PluginListWorkspaces` returns every semantic Deal and does not consult the
  Deal's definitive `workspaceTools` metadata.

## Boundaries

Synchronization ownership does not move into the desktop shell. The Sync
plugin continues to own scheduling, settings, labels, and user interaction.
The existing desktop synchronization transport performs batching because only
that layer owns pending operations and server push responses.

The server protocol and configured limits remain unchanged. Browser Inbox
continues to use the guarded read-only workspace API and receives no direct
filesystem access.

## Push Batching

The desktop sends pending operations in original queue order. A batch starts at
at most 100 operations, matching the public server default.

If the server rejects a multi-operation batch with the stable
`too_many_operations` code, the client halves that batch and retries without
marking anything as pushed. This repeats until a batch is accepted or a
single-operation batch fails. Other errors are returned immediately.

After each accepted batch, its accepted operation IDs are durably marked as
pushed before the next batch begins. Accepted IDs and conflicts are accumulated
for the final result. Therefore:

- a later failure does not resend already accepted work on the next run;
- the pending count reflects durable progress;
- operation order is preserved;
- `LastSyncAt` is still written only after every batch and the final pull
  complete successfully;
- server-side operation idempotency remains a final safety net.

Blob references are uploaded through the existing path before their operations
are pushed. No payload-size error is treated as an operation-count error.

## Paired Credential Presentation

The username is non-secret plugin configuration and remains visible after
pairing. A successful pairing persists the current username before the form can
be cleared or an automatic settings write can run.

The password is never persisted or read back. When sync status reports a stored
device token and the user has not typed a replacement, the password control is
empty internally and displays `••••••••` as a saved-password placeholder.
The placeholder must never be submitted as a password.

When a device token is stored:

- testing with no newly entered password verifies the paired device through
  the existing status request;
- entering a password makes credential testing use that new value;
- reconnecting with a newly entered password performs explicit pairing and
  replaces the locally stored device token;
- disconnect/reset removes the saved-password indication.

## Browser Inbox Deal Eligibility

`PluginListWorkspaces(pluginID)` returns semantic Deal nodes recursively, as
before, but includes a Deal only when its UUID-keyed workspace metadata lists
the requesting `pluginID` in `workspaceTools`.

This makes the API describe Deals where the calling workspace tool is active,
rather than exposing a list that the caller cannot safely use. Organizational
folders, missing metadata, malformed metadata, and Deals without the caller's
tool ID are excluded. Nested eligible Deals retain their full vault-relative
root path.

Browser Inbox keeps its existing assignment UI. Its option list becomes safe
because the host supplies only Deals containing `verstak.browser-inbox`.
Previously assigned captures remain readable even if the tool is later removed;
the inactive Deal simply cannot be selected for a new assignment.

## Error Handling

- A reduced batch is retried only for `too_many_operations`.
- A single operation rejected for that code is reported as a server
  configuration/protocol error rather than retried indefinitely.
- If a later batch fails, the user sees the existing synchronization error and
  the pending counter shows only remaining work.
- Missing or invalid workspace metadata fails closed: the Deal is not offered
  for Browser Inbox assignment.
- Technical error details remain in logs; localized UI messages remain
  non-technical.

## Verification

Tests will cover:

- 103 queued operations sent as 100 plus 3 and fully marked pushed;
- adaptive reduction against a server limit below 100;
- durable partial progress when a later batch fails;
- no adaptive retry for payload or authentication errors;
- successful sync timestamp only after all batches and the final pull;
- username persistence after pairing;
- saved-password placeholder without storing or submitting placeholder bytes;
- stored-token connection testing without a new password;
- recursive Deal listing that includes the caller's workspace tool;
- exclusion of folders, inactive Deals, missing metadata, and malformed
  metadata;
- Browser Inbox assignment options containing only eligible nested Deals.

Focused Go tests, plugin smoke tests, the real-server two-vault test, desktop
checks, and release packaging checks run before completion.

## Success Criteria

- The current 103-operation queue synchronizes against the server limit of 100.
- A successful pairing leaves the username visible and shows only a safe saved
  password indication.
- Browser Inbox never offers a new assignment to a Deal without its Browser
  workspace tool.
- No server limit is raised and no synchronization scheduling is added to the
  desktop shell.
