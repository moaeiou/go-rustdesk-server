package transport

import (
	"bytes"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const configFileName = "config.toml"

type RuntimeConfig struct {
	Bind               string
	HTTPPort           int
	NATPort            int
	IDPort             int
	RelayPort          int
	WebSocketPort      int
	WebAPIPort         int
	RelayServer        string
	RelayServers       []string
	AlwaysUseRelay     bool
	RendezvousServers  []string
	ConfigSerial       int32
	UDPReceiveBuffer   int
	SingleBandwidth    int64
	TotalBandwidth     int64
	Blocklist          []string
	Blacklist          []string
	LimitSpeed         int64
	DowngradeThreshold float64
	DowngradeAfter     int64
	DataDirectory      string
	LogPath            string
	LogLevel           string
	LogColor           bool
	MaxLogSize         int64
	MaxLogFiles        int
	MaxConnections     int64
	MaxFrameSize       uint
	EnforceKey         bool
	RelayBufferSize    int
}

type configFile struct {
	Base  baseConfig  `toml:"base"`
	Log   logConfig   `toml:"log"`
	Extra extraConfig `toml:"extra"`
}

type baseConfig struct {
	Listen            string   `toml:"listen"`
	RelayServer       string   `toml:"relay_server"`
	RelayServers      []string `toml:"relay_servers"`
	RendezvousServers []string `toml:"rendezvous_servers"`
	ConfigSerial      int32    `toml:"config_serial"`
	HTTPPort          int      `toml:"http_port"`
	NATPort           int      `toml:"nat_port"`
	IDPort            int      `toml:"id_port"`
	RelayPort         int      `toml:"relay_port"`
	WebSocketPort     int      `toml:"websocket_port"`
	WebAPIPort        int      `toml:"web_api_port"`
	DataDirectory     string   `toml:"data_dir"`
}

type logConfig struct {
	Path     string `toml:"log_path"`
	Level    string `toml:"log_level"`
	Color    bool   `toml:"log_color"`
	MaxSize  int64  `toml:"max_size"`
	MaxFiles int    `toml:"max_files"`
}

type extraConfig struct {
	AlwaysUseRelay     bool     `toml:"always_use_relay"`
	UDPReceiveBuffer   int      `toml:"udp_receive_buffer"`
	SingleBandwidth    int64    `toml:"single_bandwidth"`
	TotalBandwidth     int64    `toml:"total_bandwidth"`
	Blocklist          []string `toml:"blocklist"`
	Blacklist          []string `toml:"blacklist"`
	LimitSpeed         int64    `toml:"limit_speed"`
	DowngradeThreshold float64  `toml:"downgrade_threshold"`
	DowngradeAfter     int64    `toml:"downgrade_start_check"`
	MaxConnections     int64    `toml:"max_connections"`
	MaxFrameSize       uint     `toml:"max_frame_size"`
	EnforceKey         bool     `toml:"enforce_key"`
	RelayBufferSize    int      `toml:"relay_buffer_size"`
}

var Config, configLoadError = loadConfig()

// LoadConfigFile replaces the startup configuration with the file selected by
// the run command. It must be called before any server goroutines are started.
func LoadConfigFile(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config %s: %w", path, err)
	}
	previous, existed := os.LookupEnv("RUSTDESK_CONFIG")
	if err := os.Setenv("RUSTDESK_CONFIG", absolute); err != nil {
		return fmt.Errorf("select config %s: %w", absolute, err)
	}
	loaded, loadErr := loadConfig()
	if existed {
		_ = os.Setenv("RUSTDESK_CONFIG", previous)
	} else {
		_ = os.Unsetenv("RUSTDESK_CONFIG")
	}
	if loadErr != nil {
		return loadErr
	}
	Config = loaded
	configLoadError = nil
	PortHTTPAPI = Config.HTTPPort
	PortNAT = Config.NATPort
	PortSignal = Config.IDPort
	PortRelay = Config.RelayPort
	PortWS = Config.WebSocketPort
	PortWebAPI = Config.WebAPIPort
	return nil
}

func loadConfig() (RuntimeConfig, error) {
	baseDirectory := executableDir()
	config := RuntimeConfig{
		Bind:       "::",
		LimitSpeed: 32, DowngradeThreshold: 0.66, DowngradeAfter: 1800,
		HTTPPort:        21114,
		NATPort:         21115,
		IDPort:          21116,
		RelayPort:       21117,
		WebSocketPort:   21118,
		WebAPIPort:      21119,
		DataDirectory:   baseDirectory,
		LogPath:         filepath.Join(baseDirectory, "rdks.log"),
		LogLevel:        "info",
		LogColor:        true,
		MaxLogSize:      10 << 20,
		MaxLogFiles:     4,
		MaxConnections:  4096,
		MaxFrameSize:    16 << 20,
		EnforceKey:      true,
		RelayBufferSize: 256 << 10,
	}
	path, required := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) && !required {
		return config, nil
	}
	if err != nil {
		return config, fmt.Errorf("read config %s: %w", path, err)
	}
	file := configFile{
		Base: baseConfig{
			Listen: config.Bind, RelayServer: config.RelayServer,
			HTTPPort: config.HTTPPort, NATPort: config.NATPort, IDPort: config.IDPort,
			RelayPort: config.RelayPort, WebSocketPort: config.WebSocketPort, WebAPIPort: config.WebAPIPort,
			DataDirectory: config.DataDirectory,
		},
		Log: logConfig{
			Path: config.LogPath, Level: config.LogLevel, Color: config.LogColor,
			MaxSize: config.MaxLogSize, MaxFiles: config.MaxLogFiles,
		},
		Extra: extraConfig{
			MaxConnections: config.MaxConnections, MaxFrameSize: config.MaxFrameSize,
			LimitSpeed: config.LimitSpeed, DowngradeThreshold: config.DowngradeThreshold, DowngradeAfter: config.DowngradeAfter,
			EnforceKey: config.EnforceKey, RelayBufferSize: config.RelayBufferSize,
		},
	}
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return config, fmt.Errorf("parse config %s: %w", path, err)
	}
	config = RuntimeConfig{
		Bind: file.Base.Listen, RelayServer: file.Base.RelayServer,
		RelayServers: file.Base.RelayServers, RendezvousServers: file.Base.RendezvousServers, ConfigSerial: file.Base.ConfigSerial,
		AlwaysUseRelay: file.Extra.AlwaysUseRelay, UDPReceiveBuffer: file.Extra.UDPReceiveBuffer,
		SingleBandwidth: file.Extra.SingleBandwidth, TotalBandwidth: file.Extra.TotalBandwidth, Blocklist: file.Extra.Blocklist,
		Blacklist: file.Extra.Blacklist, LimitSpeed: file.Extra.LimitSpeed, DowngradeThreshold: file.Extra.DowngradeThreshold, DowngradeAfter: file.Extra.DowngradeAfter,
		HTTPPort: file.Base.HTTPPort, NATPort: file.Base.NATPort, IDPort: file.Base.IDPort,
		RelayPort: file.Base.RelayPort, WebSocketPort: file.Base.WebSocketPort, WebAPIPort: file.Base.WebAPIPort,
		DataDirectory: file.Base.DataDirectory, LogPath: file.Log.Path,
		LogLevel: file.Log.Level, LogColor: file.Log.Color,
		MaxLogSize: file.Log.MaxSize, MaxLogFiles: file.Log.MaxFiles,
		MaxConnections: file.Extra.MaxConnections, MaxFrameSize: file.Extra.MaxFrameSize,
		EnforceKey: file.Extra.EnforceKey, RelayBufferSize: file.Extra.RelayBufferSize,
	}
	configDirectory := filepath.Dir(path)
	config.DataDirectory = resolveConfigPath(configDirectory, config.DataDirectory)
	config.LogPath = resolveConfigPath(configDirectory, config.LogPath)
	return config, nil
}

func configPath() (string, bool) {
	if path := os.Getenv("RUSTDESK_CONFIG"); path != "" {
		absolute, err := filepath.Abs(path)
		if err == nil {
			return absolute, true
		}
		return path, true
	}
	if path, err := filepath.Abs(configFileName); err == nil {
		return path, false
	}
	return configFileName, false
}

func resolveConfigPath(directory, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(directory, path)
}

func executableDir() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(executable)
}

func ListenAddress(port int) string { return net.JoinHostPort(Config.Bind, strconv.Itoa(port)) }
func DataDir() string               { return Config.DataDirectory }

func PrepareDataDir() error {
	if configLoadError != nil {
		return configLoadError
	}
	ports := map[string]int{
		"http_port": Config.HTTPPort, "nat_port": Config.NATPort,
		"id_port": Config.IDPort, "relay_port": Config.RelayPort,
		"websocket_port": Config.WebSocketPort, "web_api_port": Config.WebAPIPort,
	}
	usedPorts := make(map[int]string, len(ports))
	for name, port := range ports {
		if port < 1 || port > 65535 {
			return fmt.Errorf("%s must be between 1 and 65535", name)
		}
		if previous, exists := usedPorts[port]; exists {
			return fmt.Errorf("%s and %s cannot both use port %d", previous, name, port)
		}
		usedPorts[port] = name
	}
	if Config.ConfigSerial < 0 || Config.UDPReceiveBuffer < 0 || Config.SingleBandwidth < 0 || Config.TotalBandwidth < 0 || Config.SingleBandwidth > 1000000 || Config.TotalBandwidth > 1000000 {
		return fmt.Errorf("invalid config_serial, udp_receive_buffer or bandwidth (0..1000000 Mb/s)")
	}
	if len(Config.RelayServers) > 0 && Config.RelayServer != "" {
		return fmt.Errorf("use relay_server or relay_servers, not both")
	}
	for _, address := range Config.RelayServers {
		if strings.TrimSpace(address) == "" {
			return fmt.Errorf("relay_servers contains empty address")
		}
	}
	if Config.LimitSpeed < 0 || Config.LimitSpeed > 1000000 || Config.DowngradeAfter < 0 || Config.DowngradeThreshold < 0 || Config.DowngradeThreshold > 1 {
		return fmt.Errorf("invalid relay downgrade settings")
	}
	for _, entry := range append(append([]string(nil), Config.Blocklist...), Config.Blacklist...) {
		if _, err := netip.ParseAddr(entry); err != nil {
			if _, err := netip.ParsePrefix(entry); err != nil {
				return fmt.Errorf("invalid blocklist entry %q", entry)
			}
		}
	}
	if Config.Bind == "" {
		return fmt.Errorf("bind must not be empty")
	}
	if Config.DataDirectory == "" || Config.LogPath == "" {
		return fmt.Errorf("data_dir and log_path must not be empty")
	}
	if Config.MaxConnections < 1 || Config.MaxFrameSize < 1024 {
		return fmt.Errorf("invalid max_connections or max_frame_size")
	}
	if Config.RelayBufferSize != 0 && (Config.RelayBufferSize < 64<<10 || Config.RelayBufferSize > 16<<20) {
		return fmt.Errorf("relay_buffer_size must be 0 or between 65536 and 16777216 bytes")
	}
	if Config.LogLevel != "info" && Config.LogLevel != "warn" && Config.LogLevel != "error" {
		return fmt.Errorf("invalid log_level %q", Config.LogLevel)
	}
	if Config.MaxLogSize < 1024 {
		return fmt.Errorf("log.max_size must be at least 1024 bytes")
	}
	if Config.MaxLogFiles < 1 || Config.MaxLogFiles > 100 {
		return fmt.Errorf("log.max_files must be between 1 and 100")
	}
	return os.MkdirAll(Config.DataDirectory, 0700)
}
