import "./style.css";
import "./app.css";

import { EventsOn } from "../wailsjs/runtime/runtime.js";
import {
  GetState,
  LoadConfig,
  RefreshSerialPorts,
  SaveConfig,
  StartAcquisition,
  StartDemo,
  StopAcquisition,
} from "../wailsjs/go/main/App.js";

const state = {
  snapshot: null,
  activeDevice: "",
};

document.querySelector("#app").innerHTML = `
  <div class="shell">
    <header class="hero">
      <div>
        <p class="eyebrow">geo-acq / prototype Wails</p>
        <h1>Acquisition cockpit</h1>
        <p class="subtitle">Configuration, panels par device et vue terminal brute dans une seule interface.</p>
      </div>
      <div class="hero-status">
        <span id="run-badge" class="badge badge-idle">idle</span>
        <span id="mode-badge" class="subtle-pill">mode: idle</span>
      </div>
    </header>

    <section class="toolbar">
      <label class="path-input">
        <span>Config</span>
        <input id="config-path" type="text" placeholder="windows.toml" />
      </label>
      <div class="toolbar-actions">
        <button id="reload-config" class="btn btn-secondary">Reload config</button>
        <button id="save-config" class="btn btn-secondary">Save config</button>
        <button id="refresh-ports" class="btn btn-secondary">Refresh ports</button>
        <button id="start-live" class="btn btn-primary">Start live</button>
        <button id="start-demo" class="btn btn-highlight">Start demo</button>
        <button id="stop-session" class="btn btn-danger">Stop</button>
      </div>
    </section>

    <section id="error-banner" class="error-banner hidden"></section>

    <main class="grid">
      <section class="card config-card">
        <div class="card-head">
          <h2>Current config</h2>
          <p id="mission-summary" class="muted">No config loaded</p>
        </div>
        <div class="config-metadata" id="config-meta"></div>
        <textarea id="config-editor" spellcheck="false"></textarea>
      </section>

      <section class="card devices-card">
        <div class="card-head">
          <h2>Device panels</h2>
          <p class="muted">Etat courant et dernière trame décodée</p>
        </div>
        <div id="devices-grid" class="devices-grid"></div>
      </section>

      <section class="card terminal-card">
        <div class="card-head">
          <h2>Terminal raw frames</h2>
          <p class="muted">Dernières trames reçues, utiles pour le diagnostic terrain</p>
        </div>
        <div id="terminal-view" class="terminal-view"></div>
      </section>

      <section class="card side-card">
        <div class="card-head">
          <h2>Environment</h2>
          <p class="muted">Ports visibles et synthèse runtime</p>
        </div>
        <div class="side-section">
          <h3>Detected serial ports</h3>
          <div id="serial-ports" class="chip-list"></div>
        </div>
        <div class="side-section">
          <h3>Session notes</h3>
          <div id="session-summary" class="summary-block"></div>
        </div>
      </section>
    </main>
  </div>
`;

const elements = {
  configPath: document.getElementById("config-path"),
  configEditor: document.getElementById("config-editor"),
  missionSummary: document.getElementById("mission-summary"),
  configMeta: document.getElementById("config-meta"),
  devicesGrid: document.getElementById("devices-grid"),
  terminalView: document.getElementById("terminal-view"),
  serialPorts: document.getElementById("serial-ports"),
  sessionSummary: document.getElementById("session-summary"),
  runBadge: document.getElementById("run-badge"),
  modeBadge: document.getElementById("mode-badge"),
  errorBanner: document.getElementById("error-banner"),
};

document.getElementById("reload-config").addEventListener("click", async () => {
  await safely(async () => {
    const path = elements.configPath.value.trim();
    const snapshot = await LoadConfig(path);
    applyState(snapshot);
  });
});

document.getElementById("save-config").addEventListener("click", async () => {
  await safely(async () => {
    const snapshot = await SaveConfig(elements.configEditor.value);
    applyState(snapshot);
  });
});

document.getElementById("refresh-ports").addEventListener("click", async () => {
  await safely(async () => {
    await RefreshSerialPorts();
    const snapshot = await GetState();
    applyState(snapshot);
  });
});

document.getElementById("start-live").addEventListener("click", async () => {
  await safely(async () => {
    await StartAcquisition();
    applyState(await GetState());
  });
});

document.getElementById("start-demo").addEventListener("click", async () => {
  await safely(async () => {
    await StartDemo();
    applyState(await GetState());
  });
});

document.getElementById("stop-session").addEventListener("click", async () => {
  await safely(async () => {
    await StopAcquisition();
    applyState(await GetState());
  });
});

EventsOn("geoacq:state", (payload) => {
  if (payload) {
    applyState(payload);
  }
});

EventsOn("geoacq:frame", (frame) => {
  if (!state.snapshot || !frame) {
    return;
  }
  const snapshot = structuredClone(state.snapshot);
  snapshot.terminalFrames = [...(snapshot.terminalFrames || []), frame].slice(-200);

  snapshot.devices = (snapshot.devices || []).map((device) => {
    if (device.name !== frame.deviceName) {
      return device;
    }
    return {
      ...device,
      status: frame.mode === "demo" ? "demo" : "streaming",
      frameCount: (device.frameCount || 0) + 1,
      lastSeen: frame.receivedAt,
      lastSentenceType: frame.sentenceType,
      lastRawFrame: frame.payload,
      decodedJson: prettyFrameJSON(frame.decodedJson),
      lastError: "",
    };
  });

  state.snapshot = snapshot;
  render();
});

bootstrap();

async function bootstrap() {
  await safely(async () => {
    const snapshot = await GetState();
    applyState(snapshot);
  });
}

function applyState(snapshot) {
  state.snapshot = snapshot;
  if (!state.activeDevice && snapshot?.devices?.length) {
    state.activeDevice = snapshot.devices[0].name;
  }
  render();
}

function render() {
  const snapshot = state.snapshot;
  if (!snapshot) {
    return;
  }

  elements.configPath.value = snapshot.config?.path || "";
  elements.configEditor.value = snapshot.config?.raw || "";
  elements.missionSummary.textContent = formatMission(snapshot.config);
  elements.configMeta.innerHTML = `
    <div class="meta-pair"><span>Database</span><strong>${escapeHTML(snapshot.config?.database || "n/a")}</strong></div>
    <div class="meta-pair"><span>Debug</span><strong>${snapshot.config?.debug ? "on" : "off"}</strong></div>
    <div class="meta-pair"><span>Echo</span><strong>${snapshot.config?.echo ? "on" : "off"}</strong></div>
    <div class="meta-pair"><span>Devices</span><strong>${(snapshot.devices || []).length}</strong></div>
  `;

  elements.runBadge.textContent = snapshot.running ? "running" : "idle";
  elements.runBadge.className = `badge ${snapshot.running ? "badge-running" : "badge-idle"}`;
  elements.modeBadge.textContent = `mode: ${snapshot.mode || "idle"}`;

  elements.errorBanner.textContent = snapshot.lastError || "";
  elements.errorBanner.classList.toggle("hidden", !snapshot.lastError);

  elements.devicesGrid.innerHTML = (snapshot.devices || [])
    .map((device) => {
      const decodedBlock = device.decodedJson
        ? `<pre>${escapeHTML(prettyFrameJSON(device.decodedJson))}</pre>`
        : `<p class="muted">No decoded payload yet.</p>`;
      return `
        <article class="device-panel">
          <div class="device-top">
            <div>
              <h3>${escapeHTML(device.name)}</h3>
              <p class="muted">${escapeHTML(device.transport)} / ${escapeHTML(device.port || "n/a")}</p>
            </div>
            <span class="status-pill status-${escapeHTML(device.status || "ready")}">${escapeHTML(device.status || "ready")}</span>
          </div>
          <div class="device-stats">
            <div><span>Type</span><strong>${escapeHTML(device.type || "n/a")}</strong></div>
            <div><span>Enabled</span><strong>${device.enabled ? "yes" : "no"}</strong></div>
            <div><span>Frames</span><strong>${device.frameCount || 0}</strong></div>
            <div><span>Last sentence</span><strong>${escapeHTML(device.lastSentenceType || "n/a")}</strong></div>
            <div><span>Last seen</span><strong>${escapeHTML(device.lastSeen || "n/a")}</strong></div>
          </div>
          <div class="device-block">
            <h4>Last raw frame</h4>
            <pre>${escapeHTML(device.lastRawFrame || "Waiting for data...")}</pre>
          </div>
          <div class="device-block">
            <h4>Decoded payload</h4>
            ${decodedBlock}
          </div>
          ${device.lastError ? `<p class="device-error">${escapeHTML(device.lastError)}</p>` : ""}
        </article>
      `;
    })
    .join("");

  const terminalLines = (snapshot.terminalFrames || [])
    .slice()
    .reverse()
    .map((frame) => `<div>${escapeHTML(frame.terminalLine || frame.payload || "")}</div>`)
    .join("");
  elements.terminalView.innerHTML = terminalLines || `<div class="muted">No frames yet.</div>`;

  elements.serialPorts.innerHTML = (snapshot.availableSerialPorts || [])
    .map((port) => `<span class="chip">${escapeHTML(port)}</span>`)
    .join("") || `<span class="muted">No serial ports detected.</span>`;

  elements.sessionSummary.innerHTML = `
    <p><strong>Config path:</strong> ${escapeHTML(snapshot.config?.path || "n/a")}</p>
    <p><strong>Session mode:</strong> ${escapeHTML(snapshot.mode || "idle")}</p>
    <p><strong>Stored frames:</strong> ${(snapshot.terminalFrames || []).length}</p>
    <p><strong>Mission:</strong> ${escapeHTML(snapshot.config?.mission?.name || "n/a")}</p>
  `;
}

async function safely(action) {
  try {
    await action();
  } catch (error) {
    const message = error?.message || String(error);
    elements.errorBanner.textContent = message;
    elements.errorBanner.classList.remove("hidden");
  }
}

function formatMission(config) {
  if (!config?.mission?.name) {
    return "No mission metadata";
  }
  const parts = [config.mission.name, config.mission.pi, config.mission.organization].filter(Boolean);
  return parts.join(" / ");
}

function prettyFrameJSON(value) {
  if (!value) {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}
