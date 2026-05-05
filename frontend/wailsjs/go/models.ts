export namespace main {
	
	export class FrameEvent {
	    receivedAt: string;
	    deviceName: string;
	    transport: string;
	    port: string;
	    payload: string;
	    sentenceType: string;
	    decodedJson: string;
	    decodeError: string;
	    mode: string;
	    terminalLine: string;
	
	    static createFrom(source: any = {}) {
	        return new FrameEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.receivedAt = source["receivedAt"];
	        this.deviceName = source["deviceName"];
	        this.transport = source["transport"];
	        this.port = source["port"];
	        this.payload = source["payload"];
	        this.sentenceType = source["sentenceType"];
	        this.decodedJson = source["decodedJson"];
	        this.decodeError = source["decodeError"];
	        this.mode = source["mode"];
	        this.terminalLine = source["terminalLine"];
	    }
	}
	export class DevicePanelState {
	    name: string;
	    type: string;
	    transport: string;
	    port: string;
	    enabled: boolean;
	    status: string;
	    frameCount: number;
	    lastSeen: string;
	    lastSentenceType: string;
	    lastRawFrame: string;
	    decodedJson: string;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new DevicePanelState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.transport = source["transport"];
	        this.port = source["port"];
	        this.enabled = source["enabled"];
	        this.status = source["status"];
	        this.frameCount = source["frameCount"];
	        this.lastSeen = source["lastSeen"];
	        this.lastSentenceType = source["lastSentenceType"];
	        this.lastRawFrame = source["lastRawFrame"];
	        this.decodedJson = source["decodedJson"];
	        this.lastError = source["lastError"];
	    }
	}
	export class DeviceConfigView {
	    name: string;
	    type: string;
	    enabled: boolean;
	    transport: string;
	    port: string;
	    sentence: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceConfigView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	        this.transport = source["transport"];
	        this.port = source["port"];
	        this.sentence = source["sentence"];
	    }
	}
	export class MissionView {
	    name: string;
	    pi: string;
	    organization: string;
	
	    static createFrom(source: any = {}) {
	        return new MissionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pi = source["pi"];
	        this.organization = source["organization"];
	    }
	}
	export class ConfigView {
	    path: string;
	    raw: string;
	    mission: MissionView;
	    database: string;
	    debug: boolean;
	    echo: boolean;
	    deviceConfigs: DeviceConfigView[];
	
	    static createFrom(source: any = {}) {
	        return new ConfigView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.raw = source["raw"];
	        this.mission = this.convertValues(source["mission"], MissionView);
	        this.database = source["database"];
	        this.debug = source["debug"];
	        this.echo = source["echo"];
	        this.deviceConfigs = this.convertValues(source["deviceConfigs"], DeviceConfigView);
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
	export class AppState {
	    config: ConfigView;
	    devices: DevicePanelState[];
	    terminalFrames: FrameEvent[];
	    availableSerialPorts: string[];
	    running: boolean;
	    mode: string;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], ConfigView);
	        this.devices = this.convertValues(source["devices"], DevicePanelState);
	        this.terminalFrames = this.convertValues(source["terminalFrames"], FrameEvent);
	        this.availableSerialPorts = source["availableSerialPorts"];
	        this.running = source["running"];
	        this.mode = source["mode"];
	        this.lastError = source["lastError"];
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

