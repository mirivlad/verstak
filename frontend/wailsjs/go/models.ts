export namespace api {
	
	export class FlatContextMenuEntry {
	    pluginId: string;
	    id: string;
	    label: string;
	    context: string;
	    group?: string;
	    capability?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatContextMenuEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.label = source["label"];
	        this.context = source["context"];
	        this.group = source["group"];
	        this.capability = source["capability"];
	        this.handler = source["handler"];
	    }
	}
	export class FlatAction {
	    pluginId: string;
	    id: string;
	    label: string;
	    icon?: string;
	    capability?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.capability = source["capability"];
	        this.handler = source["handler"];
	    }
	}
	export class FlatWorkspaceItem {
	    pluginId: string;
	    id: string;
	    title: string;
	    icon?: string;
	    component: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatWorkspaceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.component = source["component"];
	    }
	}
	export class FlatOpenProviderSupport {
	    kind: string;
	    mime?: string[];
	    extensions?: string[];
	    contexts?: string[];
	    modes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new FlatOpenProviderSupport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.mime = source["mime"];
	        this.extensions = source["extensions"];
	        this.contexts = source["contexts"];
	        this.modes = source["modes"];
	    }
	}
	export class FlatOpenProvider {
	    pluginId: string;
	    id: string;
	    title: string;
	    priority?: number;
	    component: string;
	    supports: FlatOpenProviderSupport[];
	
	    static createFrom(source: any = {}) {
	        return new FlatOpenProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.priority = source["priority"];
	        this.component = source["component"];
	        this.supports = this.convertValues(source["supports"], FlatOpenProviderSupport);
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
	export class FlatStatusBarItem {
	    pluginId: string;
	    id: string;
	    label: string;
	    position?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatStatusBarItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.label = source["label"];
	        this.position = source["position"];
	        this.handler = source["handler"];
	    }
	}
	export class FlatSidebarItem {
	    pluginId: string;
	    id: string;
	    title: string;
	    icon?: string;
	    view: string;
	    position?: number;
	
	    static createFrom(source: any = {}) {
	        return new FlatSidebarItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.view = source["view"];
	        this.position = source["position"];
	    }
	}
	export class FlatSettingsPanel {
	    pluginId: string;
	    id: string;
	    title: string;
	    icon?: string;
	    component: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatSettingsPanel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.component = source["component"];
	    }
	}
	export class FlatSearchProvider {
	    pluginId: string;
	    id: string;
	    label: string;
	    handler: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatSearchProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.label = source["label"];
	        this.handler = source["handler"];
	    }
	}
	export class FlatCommand {
	    pluginId: string;
	    id: string;
	    title: string;
	    icon?: string;
	    handler?: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.handler = source["handler"];
	    }
	}
	export class FlatView {
	    pluginId: string;
	    id: string;
	    title: string;
	    icon?: string;
	    component: string;
	
	    static createFrom(source: any = {}) {
	        return new FlatView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.component = source["component"];
	    }
	}
	export class ContributionSummary {
	    views: FlatView[];
	    commands: FlatCommand[];
	    searchProviders: FlatSearchProvider[];
	    settingsPanels: FlatSettingsPanel[];
	    sidebarItems: FlatSidebarItem[];
	    statusBarItems: FlatStatusBarItem[];
	    openProviders: FlatOpenProvider[];
	    workspaceItems: FlatWorkspaceItem[];
	    fileActions: FlatAction[];
	    noteActions: FlatAction[];
	    contextMenuEntries: FlatContextMenuEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ContributionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.views = this.convertValues(source["views"], FlatView);
	        this.commands = this.convertValues(source["commands"], FlatCommand);
	        this.searchProviders = this.convertValues(source["searchProviders"], FlatSearchProvider);
	        this.settingsPanels = this.convertValues(source["settingsPanels"], FlatSettingsPanel);
	        this.sidebarItems = this.convertValues(source["sidebarItems"], FlatSidebarItem);
	        this.statusBarItems = this.convertValues(source["statusBarItems"], FlatStatusBarItem);
	        this.openProviders = this.convertValues(source["openProviders"], FlatOpenProvider);
	        this.workspaceItems = this.convertValues(source["workspaceItems"], FlatWorkspaceItem);
	        this.fileActions = this.convertValues(source["fileActions"], FlatAction);
	        this.noteActions = this.convertValues(source["noteActions"], FlatAction);
	        this.contextMenuEntries = this.convertValues(source["contextMenuEntries"], FlatContextMenuEntry);
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
	
	
	
	
	
	
	
	
	
	
	
	export class SyncStatusDTO {
	    configured: boolean;
	    serverUrl: string;
	    deviceId: string;
	    deviceName: string;
	    connected: boolean;
	    revoked: boolean;
	    tokenStored: boolean;
	    unpushedOps: number;
	    lastSyncAt: string;
	    syncInterval: number;
	    lastError: string;
	    statusLabel: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncStatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.serverUrl = source["serverUrl"];
	        this.deviceId = source["deviceId"];
	        this.deviceName = source["deviceName"];
	        this.connected = source["connected"];
	        this.revoked = source["revoked"];
	        this.tokenStored = source["tokenStored"];
	        this.unpushedOps = source["unpushedOps"];
	        this.lastSyncAt = source["lastSyncAt"];
	        this.syncInterval = source["syncInterval"];
	        this.lastError = source["lastError"];
	        this.statusLabel = source["statusLabel"];
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

export namespace files {
	
	export class FileBytes {
	    relativePath: string;
	    size: number;
	    mimeHint: string;
	    dataBase64: string;
	
	    static createFrom(source: any = {}) {
	        return new FileBytes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relativePath = source["relativePath"];
	        this.size = source["size"];
	        this.mimeHint = source["mimeHint"];
	        this.dataBase64 = source["dataBase64"];
	    }
	}
	export class FileEntry {
	    name: string;
	    relativePath: string;
	    type: string;
	    size: number;
	    modifiedAt: string;
	    extension: string;
	    isHidden: boolean;
	    isReserved: boolean;
	    canRead: boolean;
	    canWrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relativePath = source["relativePath"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.modifiedAt = source["modifiedAt"];
	        this.extension = source["extension"];
	        this.isHidden = source["isHidden"];
	        this.isReserved = source["isReserved"];
	        this.canRead = source["canRead"];
	        this.canWrite = source["canWrite"];
	    }
	}
	export class FileMetadata {
	    relativePath: string;
	    type: string;
	    size: number;
	    modifiedAt: string;
	    createdAt?: string;
	    extension: string;
	    mimeHint: string;
	    isText: boolean;
	    isHidden: boolean;
	    isReserved: boolean;
	    canRead: boolean;
	    canWrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relativePath = source["relativePath"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.modifiedAt = source["modifiedAt"];
	        this.createdAt = source["createdAt"];
	        this.extension = source["extension"];
	        this.mimeHint = source["mimeHint"];
	        this.isText = source["isText"];
	        this.isHidden = source["isHidden"];
	        this.isReserved = source["isReserved"];
	        this.canRead = source["canRead"];
	        this.canWrite = source["canWrite"];
	    }
	}
	export class MoveOptions {
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MoveOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overwrite = source["overwrite"];
	    }
	}
	export class RestoreOptions {
	    targetPath?: string;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RestoreOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetPath = source["targetPath"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class TrashEntry {
	    originalPath: string;
	    trashPath: string;
	    trashId: string;
	    deletedAt: string;
	    originalType: string;
	    basename: string;
	
	    static createFrom(source: any = {}) {
	        return new TrashEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.trashPath = source["trashPath"];
	        this.trashId = source["trashId"];
	        this.deletedAt = source["deletedAt"];
	        this.originalType = source["originalType"];
	        this.basename = source["basename"];
	    }
	}
	export class TrashResult {
	    originalPath: string;
	    trashPath: string;
	    trashId: string;
	    deletedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new TrashResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.trashPath = source["trashPath"];
	        this.trashId = source["trashId"];
	        this.deletedAt = source["deletedAt"];
	    }
	}
	export class WriteOptions {
	    createIfMissing: boolean;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WriteOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.createIfMissing = source["createIfMissing"];
	        this.overwrite = source["overwrite"];
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
	export class OpenProviderSupport {
	    kind: string;
	    mime?: string[];
	    extensions?: string[];
	    contexts?: string[];
	    modes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new OpenProviderSupport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.mime = source["mime"];
	        this.extensions = source["extensions"];
	        this.contexts = source["contexts"];
	        this.modes = source["modes"];
	    }
	}
	export class ContributionOpenProvider {
	    id: string;
	    title: string;
	    priority?: number;
	    component: string;
	    supports: OpenProviderSupport[];
	
	    static createFrom(source: any = {}) {
	        return new ContributionOpenProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.priority = source["priority"];
	        this.component = source["component"];
	        this.supports = this.convertValues(source["supports"], OpenProviderSupport);
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
	export class ContributionWorkspaceItem {
	    id: string;
	    title: string;
	    icon?: string;
	    component: string;
	
	    static createFrom(source: any = {}) {
	        return new ContributionWorkspaceItem(source);
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
	    openProviders?: ContributionOpenProvider[];
	    workspaceItems?: ContributionWorkspaceItem[];
	
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
	        this.openProviders = this.convertValues(source["openProviders"], ContributionOpenProvider);
	        this.workspaceItems = this.convertValues(source["workspaceItems"], ContributionWorkspaceItem);
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

export namespace workbench {
	
	export class OpenResourceContext {
	    sourcePluginId?: string;
	    sourceView?: string;
	    isInsideNotesFolder?: boolean;
	    notesScopePath?: string;
	    notesMode?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OpenResourceContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourcePluginId = source["sourcePluginId"];
	        this.sourceView = source["sourceView"];
	        this.isInsideNotesFolder = source["isInsideNotesFolder"];
	        this.notesScopePath = source["notesScopePath"];
	        this.notesMode = source["notesMode"];
	    }
	}
	export class OpenResourceRequest {
	    kind: string;
	    path: string;
	    mode?: string;
	    mime?: string;
	    extension?: string;
	    context?: OpenResourceContext;
	
	    static createFrom(source: any = {}) {
	        return new OpenResourceRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.mode = source["mode"];
	        this.mime = source["mime"];
	        this.extension = source["extension"];
	        this.context = this.convertValues(source["context"], OpenResourceContext);
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
	export class OpenResourceResult {
	    status: string;
	    providerId?: string;
	    providerPluginId?: string;
	    providerComponent?: string;
	    request: OpenResourceRequest;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenResourceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.providerId = source["providerId"];
	        this.providerPluginId = source["providerPluginId"];
	        this.providerComponent = source["providerComponent"];
	        this.request = this.convertValues(source["request"], OpenResourceRequest);
	        this.message = source["message"];
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
	export class OpenedResource {
	    id: string;
	    providerId: string;
	    providerPluginId: string;
	    providerComponent: string;
	    request: OpenResourceRequest;
	    openedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenedResource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.providerId = source["providerId"];
	        this.providerPluginId = source["providerPluginId"];
	        this.providerComponent = source["providerComponent"];
	        this.request = this.convertValues(source["request"], OpenResourceRequest);
	        this.openedAt = source["openedAt"];
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
	export class Preferences {
	    defaultTextEditorProvider?: string;
	    defaultMarkdownEditorProvider?: string;
	    defaultNotesMarkdownEditorProvider?: string;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultTextEditorProvider = source["defaultTextEditorProvider"];
	        this.defaultMarkdownEditorProvider = source["defaultMarkdownEditorProvider"];
	        this.defaultNotesMarkdownEditorProvider = source["defaultNotesMarkdownEditorProvider"];
	    }
	}

}

export namespace workspace {
	
	export class TemplateSnapshot {
	    templateId: string;
	    templateName: string;
	    templateVersion: number;
	    appliedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templateId = source["templateId"];
	        this.templateName = source["templateName"];
	        this.templateVersion = source["templateVersion"];
	        this.appliedAt = source["appliedAt"];
	    }
	}
	export class Metadata {
	    workspaceName: string;
	    createdFromTemplate?: TemplateSnapshot;
	    features?: Record<string, boolean>;
	    folders?: Record<string, string>;
	    updatedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Metadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspaceName = source["workspaceName"];
	        this.createdFromTemplate = this.convertValues(source["createdFromTemplate"], TemplateSnapshot);
	        this.features = source["features"];
	        this.folders = source["folders"];
	        this.updatedAt = source["updatedAt"];
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
	export class MetadataPatch {
	    features?: Record<string, boolean>;
	    folders?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new MetadataPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.features = source["features"];
	        this.folders = source["folders"];
	    }
	}
	
	export class TrashResult {
	    originalPath: string;
	    trashPath: string;
	    trashId: string;
	    deletedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new TrashResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.trashPath = source["trashPath"];
	        this.trashId = source["trashId"];
	        this.deletedAt = source["deletedAt"];
	    }
	}
	export class Workspace {
	    name: string;
	    rootPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Workspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rootPath = source["rootPath"];
	    }
	}

}

