import "./style.css";
import "./app.css";

import { EventsOn } from "../wailsjs/runtime/runtime.js";
import {
  GetState,
  LoadConfig,
  RefreshSerialPorts,
  SaveConfig,
  SelectConfigFile,
  StartAcquisition,
  StartDemo,
  StopAcquisition,
} from "../wailsjs/go/main/App.js";

const state = {
  snapshot: null,
  editorOpen: false,
  selectedSource: "all",
  activeTab: "devices",
};

document.querySelector("#app").innerHTML = `
  <div class="shell">
    <section class="toolbar startup-banner">
      <div class="monitoring-bar">
        <p class="eyebrow">Monitoring</p>
        <div class="tabs-bar monitoring-tabs">
          <button class="tab-btn active" data-tab="config">Configuration</button>
          <button class="tab-btn" data-tab="devices">Device panels</button>
          <button class="tab-btn" data-tab="terminal">Terminal raw frames</button>
          <button class="tab-btn" data-tab="inputs">Available inputs</button>
        </div>
      </div>
      <div class="hero-actions">
        <div class="hero-status">
          <span id="run-badge" class="badge badge-idle">idle</span>
          <span id="mode-badge" class="subtle-pill">mode: idle</span>
        </div>
        <div class="toolbar-actions">
          <button id="start-live" class="btn btn-primary">Start</button>
          <button id="start-demo" class="btn btn-highlight">Start demo</button>
          <button id="stop-session" class="btn btn-danger">Stop</button>
        </div>
      </div>
    </section>

    <section id="error-banner" class="error-banner hidden"></section>

    <section class="tabs-card">
      <main class="tabs-content">
        <section class="tab-panel" id="tab-config">
          <section class="card inner-card">
            <div class="card-head">
              <h2>Configuration</h2>
              <p id="mission-summary" class="muted">No config loaded</p>
            </div>
            <section class="toolbar config-toolbar">
              <label class="path-input">
                <span>Config file</span>
                <input id="config-path" type="text" placeholder="windows.toml" />
              </label>
              <div class="toolbar-actions">
                <button id="browse-config" class="btn btn-secondary">Choose file</button>
                <button id="load-config" class="btn btn-secondary">Load config</button>
                <button id="edit-config" class="btn btn-secondary">Edit config</button>
                <button id="refresh-ports" class="btn btn-secondary">Refresh ports</button>
              </div>
            </section>
            <div class="config-metadata" id="config-meta"></div>
            <div id="session-summary" class="summary-block summary-inline"></div>
          </section>
        </section>

        <section class="tab-panel" id="tab-devices">
          <section class="card inner-card">
            <div id="devices-grid" class="devices-grid"></div>
          </section>
        </section>

        <section class="tab-panel" id="tab-terminal">
          <section class="card inner-card">
            <div class="card-head terminal-head">
              <div>
                <h2>Terminal raw frames</h2>
                <p class="muted">Latest received frames.</p>
              </div>
              <label class="terminal-filter">
                <span>Source</span>
                <select id="terminal-source"></select>
              </label>
            </div>
            <div id="terminal-view" class="terminal-view"></div>
          </section>
        </section>

        <section class="tab-panel" id="tab-inputs">
          <section class="card inner-card">
            <div class="card-head">
              <h2>Available inputs</h2>
              <p class="muted">Detected serial ports and sources exposed by the configuration.</p>
            </div>
            <div id="serial-ports" class="chip-list chip-list-wide"></div>
          </section>
        </section>
      </main>
    </section>

    <div id="editor-overlay" class="editor-overlay hidden">
      <div class="editor-shell">
        <div class="editor-toolbar">
          <div>
            <p class="eyebrow">Edit config</p>
            <h2>TOML editor</h2>
          </div>
          <div class="toolbar-actions">
            <button id="editor-cancel" class="btn btn-secondary">Cancel</button>
            <button id="editor-save" class="btn btn-primary">Validate config</button>
          </div>
        </div>
        <textarea id="config-editor" spellcheck="false"></textarea>
      </div>
    </div>
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
  editorOverlay: document.getElementById("editor-overlay"),
  terminalSource: document.getElementById("terminal-source"),
  tabButtons: Array.from(document.querySelectorAll(".tab-btn")),
  tabPanels: {
    config: document.getElementById("tab-config"),
    devices: document.getElementById("tab-devices"),
    terminal: document.getElementById("tab-terminal"),
    inputs: document.getElementById("tab-inputs"),
  },
};

document.getElementById("browse-config").addEventListener("click", async () => {
  await safely(async () => {
    const selected = await SelectConfigFile();
    if (selected) {
      elements.configPath.value = selected;
    }
  });
});

document.getElementById("load-config").addEventListener("click", async () => {
  await safely(async () => {
    const path = elements.configPath.value.trim();
    const snapshot = await LoadConfig(path);
    applyState(snapshot);
  });
});

document.getElementById("edit-config").addEventListener("click", () => {
  state.editorOpen = true;
  render();
});

document.getElementById("editor-cancel").addEventListener("click", () => {
  state.editorOpen = false;
  render();
});

document.getElementById("editor-save").addEventListener("click", async () => {
  await safely(async () => {
    const snapshot = await SaveConfig(elements.configEditor.value);
    state.editorOpen = false;
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
    state.activeTab = "devices";
    applyState(await GetState());
  });
});

document.getElementById("start-demo").addEventListener("click", async () => {
  await safely(async () => {
    await StartDemo();
    state.activeTab = "devices";
    applyState(await GetState());
  });
});

document.getElementById("stop-session").addEventListener("click", async () => {
  await safely(async () => {
    await StopAcquisition();
    applyState(await GetState());
  });
});

elements.terminalSource.addEventListener("change", (event) => {
  state.selectedSource = event.target.value;
  render();
});

for (const button of elements.tabButtons) {
  button.addEventListener("click", () => {
    state.activeTab = button.dataset.tab;
    render();
  });
}

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

    const shouldUseDecodedValues = matchesConfiguredSentence(snapshot, device.name, frame.sentenceType);
    const configuredSentences = configuredSentenceDisplayForDevice(snapshot, device.name);
    return {
      ...device,
      status: frame.mode === "demo" ? "demo" : "streaming",
      frameCount: (device.frameCount || 0) + 1,
      lastSeen: shouldUseDecodedValues ? frame.receivedAt : device.lastSeen,
      lastSentenceType: shouldUseDecodedValues ? configuredSentences : device.lastSentenceType,
      decodedJson: shouldUseDecodedValues
        ? mergeDecodedValues(device.decodedJson, frame.decodedJson, configuredSentences)
        : device.decodedJson,
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
  elements.editorOverlay.classList.toggle("hidden", !state.editorOpen);
  document.body.classList.toggle("editor-open", state.editorOpen);

  for (const button of elements.tabButtons) {
    button.classList.toggle("active", button.dataset.tab === state.activeTab);
  }
  for (const [tabName, panel] of Object.entries(elements.tabPanels)) {
    panel.classList.toggle("active", tabName === state.activeTab);
  }

  elements.devicesGrid.innerHTML = (snapshot.devices || [])
    .map((device) => {
      const configuredSentences = configuredSentenceDisplayForDevice(snapshot, device.name);
      const decodedBlock = device.decodedJson
        ? renderDecodedFields(device.decodedJson)
        : `<p class="muted">No decoded values yet.</p>`;
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
            <div><span>Sentences</span><strong>${escapeHTML(configuredSentences || device.lastSentenceType || "n/a")}</strong></div>
          </div>
          <div class="device-block">
            <h4>Decoded values</h4>
            ${decodedBlock}
          </div>
          ${device.lastError ? `<p class="device-error">${escapeHTML(device.lastError)}</p>` : ""}
        </article>
      `;
    })
    .join("");

  const sourceOptions = buildSourceOptions(snapshot);
  if (!sourceOptions.some((option) => option.value === state.selectedSource)) {
    state.selectedSource = "all";
  }
  elements.terminalSource.innerHTML = sourceOptions
    .map((option) => `<option value="${escapeHTML(option.value)}"${option.value === state.selectedSource ? " selected" : ""}>${escapeHTML(option.label)}</option>`)
    .join("");

  const terminalLines = (snapshot.terminalFrames || [])
    .filter((frame) => matchesSourceFilter(frame, state.selectedSource))
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

function renderDecodedFields(value) {
  if (!value) {
    return `<p class="muted">No decoded values yet.</p>`;
  }

  try {
    const parsed = JSON.parse(value);
    const entries = Object.entries(parsed).filter(([key, entryValue]) => {
      if (key === "sentence_type") {
        return false;
      }
      if (key === "time_utc" && parsed.datetime_utc) {
        return false;
      }
      return entryValue !== null && entryValue !== "";
    });
    if (entries.length === 0) {
      return `<p class="muted">No decoded values yet.</p>`;
    }

    return `
      <dl class="decoded-list">
        ${entries
          .map(([key, entryValue]) => {
            const formattedValue = formatDecodedValue(key, entryValue);
            return `<div class="decoded-entry"><dt>${escapeHTML(formatDecodedLabel(key))}</dt><dd>${escapeHTML(formattedValue)}</dd></div>`;
          })
          .join("")}
      </dl>
    `;
  } catch {
    return `<pre>${escapeHTML(value)}</pre>`;
  }
}

function formatDecodedLabel(key) {
  return key
    .replace(/^sentence_type$/i, "sentence")
    .replaceAll("_", " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

function formatDecodedValue(key, value) {
  if ((key === "latitude" || key === "longitude") && typeof value === "number") {
    return formatDMS(value, key === "latitude");
  }

  if (Array.isArray(value)) {
    return value.join(", ");
  }

  if (typeof value === "object" && value !== null) {
    return JSON.stringify(value);
  }

  return String(value);
}

function formatDMS(decimal, isLatitude) {
  const absolute = Math.abs(decimal);
  const degrees = Math.floor(absolute);
  const minutesFloat = (absolute - degrees) * 60;
  const minutes = Math.floor(minutesFloat);
  const seconds = (minutesFloat - minutes) * 60;
  const hemisphere = isLatitude
    ? decimal >= 0
      ? "N"
      : "S"
    : decimal >= 0
      ? "E"
      : "W";

  return `${degrees}°${minutes}′${seconds.toFixed(1)}″ ${hemisphere}`;
}

function buildSourceOptions(snapshot) {
  const options = [{ value: "all", label: "All sources" }];
  const seen = new Set(["all"]);

  for (const port of snapshot.availableSerialPorts || []) {
    const value = `serial:${port}`;
    if (!seen.has(value)) {
      seen.add(value);
      options.push({ value, label: `Serial ${port}` });
    }
  }

  for (const device of snapshot.devices || []) {
    const value = `${device.transport}:${device.port}`;
    if (device.port && !seen.has(value)) {
      seen.add(value);
      options.push({ value, label: `${String(device.transport || "").toUpperCase()} ${device.port}` });
    }
  }

  for (const frame of snapshot.terminalFrames || []) {
    const value = `${frame.transport}:${frame.port}`;
    if (frame.port && !seen.has(value)) {
      seen.add(value);
      options.push({ value, label: `${String(frame.transport || "").toUpperCase()} ${frame.port}` });
    }
  }

  return options;
}

function matchesSourceFilter(frame, filterValue) {
  if (filterValue === "all") {
    return true;
  }
  return `${frame.transport}:${frame.port}` === filterValue;
}

function matchesConfiguredSentence(snapshot, deviceName, sentenceType) {
  const deviceConfig = snapshot.config?.deviceConfigs?.find((device) => device.name === deviceName);
  const expected = configuredSentenceIDs(deviceConfig?.sentence || "");
  if (expected.length === 0) {
    return true;
  }

  const actual = normalizeSentenceID(sentenceType || "");
  if (!actual) {
    return false;
  }

  return expected.includes(actual);
}

function configuredSentenceDisplayForDevice(snapshot, deviceName) {
  const deviceConfig = snapshot.config?.deviceConfigs?.find((device) => device.name === deviceName);
  const ids = configuredSentenceIDs(deviceConfig?.sentence || "");
  if (ids.length === 0) {
    return "";
  }

  return ids.map((id) => `GP${id}`).join(",");
}

function mergeDecodedValues(existingJson, incomingJson, sentenceDisplay) {
  const incoming = parseDecodedObject(incomingJson);
  if (!incoming) {
    return existingJson;
  }

  const merged = parseDecodedObject(existingJson) || {};
  for (const [key, value] of Object.entries(incoming)) {
    if (key === "sentence_type") {
      continue;
    }
    merged[key] = value;
  }

  if (sentenceDisplay) {
    merged.sentence_type = sentenceDisplay;
  } else if (incoming.sentence_type) {
    merged.sentence_type = incoming.sentence_type;
  }

  return JSON.stringify(merged);
}

function parseDecodedObject(value) {
  if (!value) {
    return null;
  }

  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === "object" ? parsed : null;
  } catch {
    return null;
  }
}

function configuredSentenceIDs(value) {
  const seen = new Set();
  const configured = [];

  for (const part of String(value || "").split(",")) {
    const normalized = normalizeSentenceID(part);
    if (!normalized || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    configured.push(normalized);
  }

  return configured;
}

function normalizeSentenceID(value) {
  const normalized = String(value || "").trim().toUpperCase();
  if (!normalized) {
    return "";
  }

  return normalized.length > 3 ? normalized.slice(-3) : normalized;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}
