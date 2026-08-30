# Deal-only scope

Verstak has one runtime resource scope:

```text
DealScope { workspaceId: UUID }
```

`workspacetree.Service` is the canonical Deal registry. A path is a mutable
address; the UUID stored in the Deal marker and versioned metadata is its
identity. Notes, Files, Todo, Activity, Milestones, Git descriptors, and every
other provider resource use that UUID only. There is no supported
`Deal -> Project -> scoped resources` runtime model.

Project Meta is optional, namespaced configuration on one Deal. Its global
portfolio lists Deals with that capability and opens the chosen Deal by UUID.
Milestones and Git are independent reusable Deal plugins. Git synchronizes
repository descriptors only; credentials are Secret references, and checkout
trees including `.git` stay device-local and are excluded from ordinary Sync.

Templates are persisted plugin-owned recipes. Core validates generic recipe
limits and publishes canonical metadata, but has no hardcoded template catalog
or business-plugin mapping.

## Legacy migration

`deal-only-v1` is a one-time, backup-first transaction. It retains legacy
Project records only inside the immutable backup, flattens provider-owned
content to Deal UUID scope, transfers milestones where possible, and copies
Project Meta only when exactly one legacy Project maps to a Deal. The verified
ledger prevents the new runtime from consulting legacy Project data again.

No migration wizard, multi-Deal split, or compatibility layer is provided.
