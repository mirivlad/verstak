# Deal Templates

Deal templates choose which dynamic plugin Deal tabs are visible when a new
top-level Deal is created. They do not enable or disable plugins for the
whole vault and do not affect global sidebar tools.

## Built-in templates

| Template | Deal tabs |
| --- | --- |
| General | Notes, Files, Journal, Activity, Browser Inbox |
| Project | Notes, Files, Todos, Journal, Activity, Browser Inbox |
| Writing | Notes, Files, Journal |
| Admin | Notes, Files, Secrets, Todos, Journal |
| Minimal | Notes, Files |

The create-Deal modal displays the selected template description and its
included plugin tabs before the folder is created.

## Metadata and compatibility

Creation stores a template snapshot in `.verstak/workspaces/` metadata. The
snapshot contains the template id, name, version, applied time, and an ordered
`workspaceTools` list of plugin IDs. Existing Deal metadata without
`workspaceTools` remains compatible: its Deal continues to show all globally
enabled Deal plugins rather than unexpectedly hiding tabs.

Templates are applied once. Editing the built-in catalog or creating another
Deal with a different template never changes an existing Deal snapshot.
There is no template editor or post-creation template switcher yet.

## Global tools and unavailable plugins

Template visibility applies only to `workspaceItems` in the selected Deal.
Global views and sidebar items, such as global Todos, Browser Inbox, and Trash,
remain available according to the normal plugin enablement state.

The Deal host intersects the snapshot with dynamically discovered, globally
enabled plugin contributions. If a template references a plugin that is missing or
disabled, that tab is simply absent; the other template tabs remain usable.
