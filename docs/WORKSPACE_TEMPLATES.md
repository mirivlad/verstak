# Deal templates

Templates are persisted creation recipes owned by the `verstak.templates`
plugin. They create a Deal; they do not define a second container type and
never govern that Deal after creation.

## Scope and ownership

- A Deal UUID is the only runtime scope for all resources.
- Core accepts a complete recipe snapshot: enabled tools, initial files and
  folders, tool configuration, and historical provenance.
- Core does not contain a template catalog or interpret plugin-specific recipe
  fields. Provider plugins own their data and any folders they need.
- `createdFromTemplate` is an immutable history snapshot. Editing or deleting
  a template cannot mutate existing Deals.

The Templates plugin includes persisted seed recipes: General, Project,
Writing, Admin, and Minimal. Users can create, edit, and delete their own
recipes. A recipe that names a missing Deal plugin is rejected before creation;
the host never silently substitutes another tool set.

## Project recipe

The Project seed assembles independent Deal plugins: Project Meta, Git, Todo,
Milestones, Notes, Files, Activity, Journal, and Secrets. Project Meta is
metadata on that same Deal, not a nested Project scope. A Deal may have zero or
more Git repository descriptors and milestones independently of Project Meta.

## Migration behavior

Opening a legacy vault triggers the one-shot `deal-only-v1` migration only
when legacy Project-scoped data is found. It first writes an immutable backup
and then flattens provider records to their Deal UUID, migrates one
unambiguous Project record into Project Meta, and copies milestones to the
Milestones provider. Multiple legacy Projects in one Deal are retained only in
the backup; no extra Deals are created and no metadata winner is guessed.

After the verified ledger is written, normal runtime does not read legacy
Project records again. See
[`DEAL_ONLY_SCOPE.md`](DEAL_ONLY_SCOPE.md) for the canonical architecture and
migration invariants.
