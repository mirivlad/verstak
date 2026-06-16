export namespace api {
	
	export class ContributionSummary {
	    views: contribution.ContributionView[];
	    commands: contribution.ContributionCommand[];
	    settingsPanels: contribution.ContributionSettingsPanel[];
	    sidebarItems: contribution.ContributionSidebarItem[];
	    fileActions: contribution.ContributionAction[];
	    noteActions: contribution.ContributionAction[];
	    searchProviders: contribution.ContributionSearchProvider[];
	
	    static createFrom(source: any = {}) {
	        return new ContributionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.views = this.convertValues(source["views"], contribution.ContributionView);
	        this.commands = this.convertValues(source["commands"], contribution.ContributionCommand);
	        this.settingsPanels = this.convertValues(source["settingsPanels"], contribution.ContributionSettingsPanel);
	        this.sidebarItems = this.convertValues(source["sidebarItems"], contribution.ContributionSidebarItem);
	        this.fileActions = this.convertValues(source["fileActions"], contribution.ContributionAction);
	        this.noteActions = this.convertValues(source["noteActions"], contribution.ContributionAction);
	        this.searchProviders = this.convertValues(source["searchProviders"], contribution.ContributionSearchProvider);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace capability {
	
	export class Entry {
	    name: string;
	    description?: string;
	    pluginId: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.pluginId = source["pluginId"];
	        this.status = source["status"];
	    }
	}

}

export namespace contribution {
	
	export class ContributionAction {
	    pluginId: string;
	    item: plugin.ContributionAction;
	
	    static createFrom(source: any = {}) {
	        return new ContributionAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.item = this.convertValues(source["item"], plugin.ContributionAction);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContributionCommand {
	    pluginId: string;
	    item: plugin.ContributionCommand;
	
	    static createFrom(source: any = {}) {
	        return new ContributionCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.item = this.convertValues(source["item"], plugin.ContributionCommand);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContributionSearchProvider {
	    pluginId: string;
	    item: plugin.ContributionSearchProvider;
	
	    static createFrom(source: any = {}) {
	        return new ContributionSearchProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.item = this.convertValues(source["item"], plugin.ContributionSearchProvider);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContributionSettingsPanel {
	    pluginId: string;
	    item: plugin.ContributionSettingsPanel;
	
	    static createFrom(source: any = {}) {
	        return new ContributionSettingsPanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.item = this.convertValues(source["item"], plugin.ContributionSettingsPanel);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContributionSidebarItem {
	    pluginId: string;
	    item: plugin.ContributionSidebarItem;
	
	    static createFrom(source: any = {}) {
	        return new ContributionSidebarItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.item = this.convertValues(source["item"], plugin.ContributionSidebarItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContributionView {
	    pluginId: string;
	    item: plugin.ContributionView;
	
	    static createFrom(source: any = {}) {
	        return new ContributionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.item = this.convertValues(source["item"], plugin.ContributionView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace permissions {
	
	export class Entry {
	    name: string;
	    description: string;
	    dangerous: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.dangerous = source["dangerous"];
	    }
	}

}

export namespace plugin {
	
	export class HealthCheckConfig {
	    type?: string;
	    timeout?: number;
	
	    static createFrom(source: any = {}) {
	        return new HealthCheckConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.timeout = source["timeout"];
	    }
	}
	export class BackendConfig {
	    type: string;
	    entry: Record<string, string>;
	    healthCheck?: HealthCheckConfig;
	
	    static createFrom(source: any = {}) {
	        return new BackendConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.entry = source["entry"];
	        this.healthCheck = this.convertValues(source["healthCheck"], HealthCheckConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ContributionAction {
	    id: string;
	    label: string;
	    icon?: string;
	    capability?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.capability = source["capability"];
	        this.handler = source["handler"];
	    }
	}
	export class ContributionActivityProvider {
	    id: string;
	    events?: string[];
	    handler: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionActivityProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.events = source["events"];
	        this.handler = source["handler"];
	    }
	}
	export class ContributionCommand {
	    id: string;
	    title: string;
	    keybinding?: string;
	    icon?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.keybinding = source["keybinding"];
	        this.icon = source["icon"];
	        this.handler = source["handler"];
	    }
	}
	export class ContributionContextMenuEntry {
	    id: string;
	    label: string;
	    context: string;
	    group?: string;
	    capability?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionContextMenuEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.context = source["context"];
	        this.group = source["group"];
	        this.capability = source["capability"];
	        this.handler = source["handler"];
	    }
	}
	export class ContributionSearchProvider {
	    id: string;
	    label: string;
	    handler: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionSearchProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.handler = source["handler"];
	    }
	}
	export class ContributionSettingsPanel {
	    id: string;
	    title: string;
	    component: string;
	    icon?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionSettingsPanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.component = source["component"];
	        this.icon = source["icon"];
	    }
	}
	export class ContributionSidebarItem {
	    id: string;
	    title: string;
	    icon?: string;
	    view: string;
	    position?: number;
	
	    static createFrom(source: any = {}) {
	        return new ContributionSidebarItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.view = source["view"];
	        this.position = source["position"];
	    }
	}
	export class ContributionStatusBarItem {
	    id: string;
	    label: string;
	    position?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionStatusBarItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.position = source["position"];
	        this.handler = source["handler"];
	    }
	}
	export class ContributionView {
	    id: string;
	    title: string;
	    icon?: string;
	    component: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.component = source["component"];
	    }
	}
	export class Contributions {
	    views?: ContributionView[];
	    commands?: ContributionCommand[];
	    settingsPanels?: ContributionSettingsPanel[];
	    sidebarItems?: ContributionSidebarItem[];
	    fileActions?: ContributionAction[];
	    noteActions?: ContributionAction[];
	    contextMenuEntries?: ContributionContextMenuEntry[];
	    searchProviders?: ContributionSearchProvider[];
	    activityProviders?: ContributionActivityProvider[];
	    statusBarItems?: ContributionStatusBarItem[];
	
	    static createFrom(source: any = {}) {
	        return new Contributions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.views = this.convertValues(source["views"], ContributionView);
	        this.commands = this.convertValues(source["commands"], ContributionCommand);
	        this.settingsPanels = this.convertValues(source["settingsPanels"], ContributionSettingsPanel);
	        this.sidebarItems = this.convertValues(source["sidebarItems"], ContributionSidebarItem);
	        this.fileActions = this.convertValues(source["fileActions"], ContributionAction);
	        this.noteActions = this.convertValues(source["noteActions"], ContributionAction);
	        this.contextMenuEntries = this.convertValues(source["contextMenuEntries"], ContributionContextMenuEntry);
	        this.searchProviders = this.convertValues(source["searchProviders"], ContributionSearchProvider);
	        this.activityProviders = this.convertValues(source["activityProviders"], ContributionActivityProvider);
	        this.statusBarItems = this.convertValues(source["statusBarItems"], ContributionStatusBarItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FrontendConfig {
	    entry: string;
	    style?: string;
	
	    static createFrom(source: any = {}) {
	        return new FrontendConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.style = source["style"];
	    }
	}
	
	export class SyncConfig {
	    namespaces?: string[];
	    participate?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SyncConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.namespaces = source["namespaces"];
	        this.participate = source["participate"];
	    }
	}
	export class MigrationConfig {
	    path?: string;
	
	    static createFrom(source: any = {}) {
	        return new MigrationConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class Manifest {
	    schemaVersion: number;
	    id: string;
	    name: string;
	    version: string;
	    apiVersion: string;
	    description?: string;
	    source?: string;
	    icon?: string;
	    provides: string[];
	    requires?: string[];
	    optionalRequires?: string[];
	    permissions: string[];
	    frontend?: FrontendConfig;
	    backend?: BackendConfig;
	    migrations?: MigrationConfig;
	    contributes?: Contributions;
	    sync?: SyncConfig;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.apiVersion = source["apiVersion"];
	        this.description = source["description"];
	        this.source = source["source"];
	        this.icon = source["icon"];
	        this.provides = source["provides"];
	        this.requires = source["requires"];
	        this.optionalRequires = source["optionalRequires"];
	        this.permissions = source["permissions"];
	        this.frontend = this.convertValues(source["frontend"], FrontendConfig);
	        this.backend = this.convertValues(source["backend"], BackendConfig);
	        this.migrations = this.convertValues(source["migrations"], MigrationConfig);
	        this.contributes = this.convertValues(source["contributes"], Contributions);
	        this.sync = this.convertValues(source["sync"], SyncConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Plugin {
	    manifest: Manifest;
	    status: string;
	    error?: string;
	    enabled: boolean;
	    rootPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Plugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifest = this.convertValues(source["manifest"], Manifest);
	        this.status = source["status"];
	        this.error = source["error"];
	        this.enabled = source["enabled"];
	        this.rootPath = source["rootPath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

