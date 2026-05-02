package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/jgrelet/geo-acq/config"
	"github.com/jgrelet/geo-acq/decoder"
	"github.com/jgrelet/geo-acq/devices"
	"github.com/jgrelet/geo-acq/simul"
	"github.com/jgrelet/geo-acq/storage"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventState    = "geoacq:state"
	eventFrame    = "geoacq:frame"
	terminalLimit = 200
)

type App struct {
	ctx context.Context

	mu             sync.RWMutex
	configPath     string
	configRaw      string
	cfg            config.Config
	deviceOrder    []string
	deviceStates   map[string]*DevicePanelState
	terminalFrames []FrameEvent
	serialPorts    []string
	running        bool
	mode           string
	lastError      string
	session        *acquisitionSession
}

type acquisitionSession struct {
	mode    string
	cancel  context.CancelFunc
	done    chan struct{}
	devices []*devices.Device
	store   *storage.SQLiteStore
}

type AppState struct {
	Config               ConfigView         `json:"config"`
	Devices              []DevicePanelState `json:"devices"`
	TerminalFrames       []FrameEvent       `json:"terminalFrames"`
	AvailableSerialPorts []string           `json:"availableSerialPorts"`
	Running              bool               `json:"running"`
	Mode                 string             `json:"mode"`
	LastError            string             `json:"lastError"`
}

type ConfigView struct {
	Path      string             `json:"path"`
	Raw       string             `json:"raw"`
	Mission   MissionView        `json:"mission"`
	Database  string             `json:"database"`
	Debug     bool               `json:"debug"`
	Echo      bool               `json:"echo"`
	DeviceCfg []DeviceConfigView `json:"deviceConfigs"`
}

type MissionView struct {
	Name         string `json:"name"`
	PI           string `json:"pi"`
	Organization string `json:"organization"`
}

type DeviceConfigView struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
	Transport string `json:"transport"`
	Port      string `json:"port"`
}

type DevicePanelState struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Transport        string `json:"transport"`
	Port             string `json:"port"`
	Enabled          bool   `json:"enabled"`
	Status           string `json:"status"`
	FrameCount       int    `json:"frameCount"`
	LastSeen         string `json:"lastSeen"`
	LastSentenceType string `json:"lastSentenceType"`
	LastRawFrame     string `json:"lastRawFrame"`
	DecodedJSON      string `json:"decodedJson"`
	LastError        string `json:"lastError"`
}

type FrameEvent struct {
	ReceivedAt   string `json:"receivedAt"`
	DeviceName   string `json:"deviceName"`
	Transport    string `json:"transport"`
	Port         string `json:"port"`
	Payload      string `json:"payload"`
	SentenceType string `json:"sentenceType"`
	DecodedJSON  string `json:"decodedJson"`
	DecodeError  string `json:"decodeError"`
	Mode         string `json:"mode"`
	TerminalLine string `json:"terminalLine"`
}

func NewApp() *App {
	return &App{
		mode:         "idle",
		deviceStates: make(map[string]*DevicePanelState),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_, _ = a.LoadConfig(config.DefaultFile())
}

func (a *App) GetState() AppState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	devicesSnapshot := make([]DevicePanelState, 0, len(a.deviceOrder))
	for _, name := range a.deviceOrder {
		state, ok := a.deviceStates[name]
		if !ok {
			continue
		}
		devicesSnapshot = append(devicesSnapshot, *state)
	}

	terminalSnapshot := append([]FrameEvent(nil), a.terminalFrames...)
	serialSnapshot := append([]string(nil), a.serialPorts...)

	return AppState{
		Config:               a.snapshotConfigLocked(),
		Devices:              devicesSnapshot,
		TerminalFrames:       terminalSnapshot,
		AvailableSerialPorts: serialSnapshot,
		Running:              a.running,
		Mode:                 a.mode,
		LastError:            a.lastError,
	}
}

func (a *App) LoadConfig(path string) (AppState, error) {
	if strings.TrimSpace(path) == "" {
		path = config.DefaultFile()
	}

	if a.isRunning() {
		return a.GetState(), fmt.Errorf("stop the current session before loading another config")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		a.setLastError(fmt.Sprintf("read config %s: %v", path, err))
		return a.GetState(), fmt.Errorf("read config %s: %w", path, err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		a.setLastError(err.Error())
		return a.GetState(), err
	}

	serialPorts := discoverSerialPorts()
	deviceOrder := sortedDeviceNames(cfg)
	deviceStates := buildDeviceStates(cfg, deviceOrder)

	a.mu.Lock()
	a.configPath = path
	a.configRaw = string(raw)
	a.cfg = cfg
	a.deviceOrder = deviceOrder
	a.deviceStates = deviceStates
	a.serialPorts = serialPorts
	a.mode = "idle"
	a.lastError = ""
	a.terminalFrames = nil
	a.mu.Unlock()

	a.emitState()
	return a.GetState(), nil
}

func (a *App) SaveConfig(raw string) (AppState, error) {
	if a.isRunning() {
		return a.GetState(), fmt.Errorf("stop the current session before saving the config")
	}

	var cfg config.Config
	if _, err := toml.Decode(raw, &cfg); err != nil {
		return a.GetState(), fmt.Errorf("invalid TOML: %w", err)
	}

	path := a.currentConfigPath()
	if path == "" {
		path = config.DefaultFile()
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		return a.GetState(), fmt.Errorf("write config %s: %w", path, err)
	}

	return a.LoadConfig(path)
}

func (a *App) SelectConfigFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}

	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Select geo-acq config",
		Filters: []wruntime.FileFilter{
			{
				DisplayName: "TOML config",
				Pattern:     "*.toml",
			},
		},
	})
}

func (a *App) RefreshSerialPorts() ([]string, error) {
	ports := discoverSerialPorts()
	a.mu.Lock()
	a.serialPorts = ports
	a.mu.Unlock()
	a.emitState()
	return append([]string(nil), ports...), nil
}

func (a *App) StartAcquisition() error {
	cfg, path, err := a.requireConfig()
	if err != nil {
		return err
	}
	if a.isRunning() {
		return fmt.Errorf("a session is already running")
	}

	deviceNames := enabledDeviceNames(cfg)
	if len(deviceNames) == 0 {
		return a.fail(fmt.Errorf("no enabled devices found in configuration"))
	}

	store, err := storage.OpenSQLite(cfg.Acq.File, cfg.Mission, path)
	if err != nil {
		return a.fail(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &acquisitionSession{
		mode:   "live",
		cancel: cancel,
		done:   make(chan struct{}),
		store:  store,
	}

	for _, name := range deviceNames {
		dev := devices.New(name, cfg)
		if err := dev.Connect(); err != nil {
			session.cancel()
			close(session.done)
			a.cleanupSession(session)
			return a.fail(fmt.Errorf("connect %s: %w", name, err))
		}
		session.devices = append(session.devices, dev)
		a.updateDeviceStatus(name, "connected", "")

		transport := cfg.Devices[name].Device
		port := dev.Port()
		go a.consumeDevice(ctx, session, name, transport, port, dev.Data)
	}

	a.mu.Lock()
	a.session = session
	a.running = true
	a.mode = "live"
	a.lastError = ""
	a.mu.Unlock()

	go a.waitForSession(ctx, session, "ready")
	a.emitState()
	return nil
}

func (a *App) StartDemo() error {
	cfg, path, err := a.requireConfig()
	if err != nil {
		return err
	}
	if a.isRunning() {
		return fmt.Errorf("a session is already running")
	}

	store, err := storage.OpenSQLite(cfg.Acq.File, cfg.Mission, path)
	if err != nil {
		return a.fail(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &acquisitionSession{
		mode:   "demo",
		cancel: cancel,
		done:   make(chan struct{}),
		store:  store,
	}

	a.mu.Lock()
	a.session = session
	a.running = true
	a.mode = "demo"
	a.lastError = ""
	for _, name := range a.deviceOrder {
		if state, ok := a.deviceStates[name]; ok {
			state.Status = "demo"
			state.LastError = ""
		}
	}
	a.mu.Unlock()

	if hasDevice(cfg, "gps") {
		go a.consumeSimulated(ctx, session, "gps", configuredTransport(cfg, "gps", "demo"), configuredPort(cfg, "gps", configuredTransport(cfg, "gps", "demo")), simul.NewGps(1, 5.4, 36.0))
	}
	if hasDevice(cfg, "echosounder") {
		go a.consumeSimulated(ctx, session, "echosounder", configuredTransport(cfg, "echosounder", "demo"), configuredPort(cfg, "echosounder", configuredTransport(cfg, "echosounder", "demo")), simul.NewEchoSounder(1500*time.Millisecond, 12.8))
	}

	go a.waitForSession(ctx, session, "ready")
	a.emitState()
	return nil
}

func (a *App) StopAcquisition() error {
	a.mu.Lock()
	session := a.session
	if session == nil {
		a.running = false
		a.mode = "idle"
		a.lastError = ""
		for _, name := range a.deviceOrder {
			if state, ok := a.deviceStates[name]; ok {
				state.Status = defaultStatus(state.Enabled)
			}
		}
		a.mu.Unlock()
		a.emitState()
		return nil
	}

	a.session = nil
	a.running = false
	a.mode = "idle"
	for _, name := range a.deviceOrder {
		if state, ok := a.deviceStates[name]; ok {
			state.Status = defaultStatus(state.Enabled)
		}
	}
	a.mu.Unlock()

	session.cancel()
	a.cleanupSession(session)
	a.emitState()
	return nil
}

func (a *App) consumeDevice(ctx context.Context, session *acquisitionSession, name string, transport string, port string, dataCh <-chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case sentence, ok := <-dataCh:
			if !ok {
				a.updateDeviceStatus(name, defaultStatus(isEnabled(a, name)), "")
				return
			}
			a.handleFrame(session, FrameEvent{
				ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano),
				DeviceName: name,
				Transport:  transport,
				Port:       port,
				Payload:    sentence,
				Mode:       session.mode,
			})
		}
	}
}

func (a *App) consumeSimulated(ctx context.Context, session *acquisitionSession, name string, transport string, port string, dataCh <-chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case sentence := <-dataCh:
			a.handleFrame(session, FrameEvent{
				ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano),
				DeviceName: name,
				Transport:  transport,
				Port:       port,
				Payload:    sentence,
				Mode:       session.mode,
			})
		}
	}
}

func (a *App) waitForSession(ctx context.Context, session *acquisitionSession, idleStatus string) {
	<-ctx.Done()
	close(session.done)
}

func (a *App) handleFrame(session *acquisitionSession, frame FrameEvent) {
	decoded, err := decoder.DecodeNMEA(frame.Payload)
	if err != nil {
		frame.SentenceType = "RAW"
		frame.DecodeError = err.Error()
	} else {
		frame.SentenceType = decoded.SentenceType
		frame.DecodedJSON = decoded.JSON
	}
	frame.TerminalLine = formatTerminalFrame(frame)

	if session != nil && session.store != nil {
		if err := session.store.SaveRawFrame(storage.RawFrame{
			ReceivedAt:   mustParseTime(frame.ReceivedAt),
			DeviceName:   frame.DeviceName,
			Transport:    frame.Transport,
			Payload:      frame.Payload,
			SentenceType: storedSentenceType(frame),
			DecodedJSON:  frame.DecodedJSON,
		}); err != nil {
			a.setLastError(fmt.Sprintf("insert raw frame: %v", err))
		}
	}

	a.mu.Lock()
	panel, ok := a.deviceStates[frame.DeviceName]
	if ok {
		panel.Status = "streaming"
		if frame.Mode == "demo" {
			panel.Status = "demo"
		}
		panel.FrameCount++
		panel.LastSeen = frame.ReceivedAt
		panel.LastSentenceType = frame.SentenceType
		panel.LastRawFrame = frame.Payload
		panel.DecodedJSON = frame.DecodedJSON
		panel.LastError = ""
	}
	a.terminalFrames = append(a.terminalFrames, frame)
	if len(a.terminalFrames) > terminalLimit {
		a.terminalFrames = append([]FrameEvent(nil), a.terminalFrames[len(a.terminalFrames)-terminalLimit:]...)
	}
	a.mu.Unlock()

	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, eventFrame, frame)
	}
}

func (a *App) fail(err error) error {
	a.setLastError(err.Error())
	a.emitState()
	return err
}

func (a *App) setLastError(msg string) {
	a.mu.Lock()
	a.lastError = msg
	a.mu.Unlock()
}

func (a *App) emitState() {
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, eventState, a.GetState())
}

func (a *App) cleanupSession(session *acquisitionSession) {
	if session == nil {
		return
	}
	for _, dev := range session.devices {
		_ = dev.Disconnect()
	}
	if session.store != nil {
		_ = session.store.Close()
	}
	select {
	case <-session.done:
	default:
	}
}

func (a *App) requireConfig() (config.Config, string, error) {
	path := a.currentConfigPath()
	if path == "" {
		path = config.DefaultFile()
	}
	a.mu.RLock()
	hasConfig := a.configPath != "" && len(a.deviceStates) > 0
	currentCfg := a.cfg
	a.mu.RUnlock()
	if hasConfig {
		return currentCfg, path, nil
	}
	if _, err := a.LoadConfig(path); err != nil {
		return config.Config{}, "", err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg, a.configPath, nil
}

func (a *App) currentConfigPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configPath
}

func (a *App) isRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

func (a *App) updateDeviceStatus(name string, status string, lastError string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if state, ok := a.deviceStates[name]; ok {
		state.Status = status
		state.LastError = lastError
	}
}

func (a *App) snapshotConfigLocked() ConfigView {
	devicesCfg := make([]DeviceConfigView, 0, len(a.deviceOrder))
	for _, name := range a.deviceOrder {
		deviceCfg, ok := a.cfg.Devices[name]
		if !ok {
			continue
		}
		devicesCfg = append(devicesCfg, DeviceConfigView{
			Name:      name,
			Type:      deviceCfg.Type,
			Enabled:   deviceCfg.Use,
			Transport: deviceCfg.Device,
			Port:      configuredPort(a.cfg, name, deviceCfg.Device),
		})
	}

	return ConfigView{
		Path: a.configPath,
		Raw:  a.configRaw,
		Mission: MissionView{
			Name:         a.cfg.Mission.Name,
			PI:           a.cfg.Mission.PI,
			Organization: a.cfg.Mission.Organization,
		},
		Database:  a.cfg.Acq.File,
		Debug:     a.cfg.Global.Debug,
		Echo:      a.cfg.Global.Echo,
		DeviceCfg: devicesCfg,
	}
}

func discoverSerialPorts() []string {
	ports, err := devices.SerialGetInfo()
	if err != nil {
		return nil
	}
	sort.Strings(ports)
	return ports
}

func buildDeviceStates(cfg config.Config, names []string) map[string]*DevicePanelState {
	out := make(map[string]*DevicePanelState, len(names))
	for _, name := range names {
		deviceCfg := cfg.Devices[name]
		out[name] = &DevicePanelState{
			Name:      name,
			Type:      deviceCfg.Type,
			Transport: deviceCfg.Device,
			Port:      configuredPort(cfg, name, deviceCfg.Device),
			Enabled:   deviceCfg.Use,
			Status:    defaultStatus(deviceCfg.Use),
		}
	}
	return out
}

func defaultStatus(enabled bool) string {
	if enabled {
		return "ready"
	}
	return "disabled"
}

func configuredPort(cfg config.Config, name string, transport string) string {
	switch transport {
	case "serial":
		return cfg.Serials[name].Port
	case "udp":
		if cfg.UDP[name].Host != "" {
			return netJoinHostPort(cfg.UDP[name].Host, cfg.UDP[name].Port)
		}
		return cfg.UDP[name].Port
	default:
		return ""
	}
}

func configuredTransport(cfg config.Config, name string, fallback string) string {
	if deviceCfg, ok := cfg.Devices[name]; ok && strings.TrimSpace(deviceCfg.Device) != "" {
		return deviceCfg.Device
	}
	return fallback
}

func sortedDeviceNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Devices))
	for name := range cfg.Devices {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func enabledDeviceNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Devices))
	for _, name := range sortedDeviceNames(cfg) {
		if cfg.Devices[name].Use {
			names = append(names, name)
		}
	}
	return names
}

func hasDevice(cfg config.Config, name string) bool {
	_, ok := cfg.Devices[name]
	return ok
}

func isEnabled(a *App, name string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	state, ok := a.deviceStates[name]
	return ok && state.Enabled
}

func formatTerminalFrame(frame FrameEvent) string {
	sentenceType := frame.SentenceType
	if sentenceType == "" {
		sentenceType = "RAW"
	}

	return fmt.Sprintf(
		"%s | %-12s | %-6s | %-8s | %-5s | %s",
		frame.ReceivedAt,
		frame.DeviceName,
		frame.Transport,
		frame.Port,
		sentenceType,
		frame.Payload,
	)
}

func storedSentenceType(frame FrameEvent) string {
	if frame.DecodeError != "" {
		return ""
	}
	return frame.SentenceType
}

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Now().UTC()
	}
	return parsed
}

func netJoinHostPort(host string, port string) string {
	if strings.TrimSpace(host) == "" {
		return port
	}
	return fmt.Sprintf("%s:%s", host, port)
}
