package transport

import (
	"bytes"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const configFileName = "config.toml"

type RuntimeConfig struct {
	Bind           string
	HTTPPort       int
	NATPort        int
	IDPort         int
	RelayPort      int
	WebSocketPort  int
	WebAPIPort     int
	RelayServer    string
	DataDirectory  string
	LogPath        string
	LogLevel       string
	LogColor       bool
	MaxLogSize     int64
	MaxLogFiles    int
	MaxConnections int64
	MaxFrameSize   uint
	EnforceKey     bool
}

type configFile struct {
	Base  baseConfig  `toml:"base"`
	Log   logConfig   `toml:"log"`
	Extra extraConfig `toml:"extra"`
}

type baseConfig struct {
	Listen        string `toml:"listen"`
	RelayServer   string `toml:"relay_server"`
	HTTPPort      int    `toml:"http_port"`
	NATPort       int    `toml:"nat_port"`
	IDPort        int    `toml:"id_port"`
	RelayPort     int    `toml:"relay_port"`
	WebSocketPort int    `toml:"websocket_port"`
	WebAPIPort    int    `toml:"web_api_port"`
	DataDirectory string `toml:"data_dir"`
}

type logConfig struct {
	Path     string `toml:"log_path"`
	Level    string `toml:"log_level"`
	Color    bool   `toml:"log_color"`
	MaxSize  int64  `toml:"max_size"`
	MaxFiles int    `toml:"max_files"`
}

type extraConfig struct {
	MaxConnections int64 `toml:"max_connections"`
	MaxFrameSize   uint  `toml:"max_frame_size"`
	EnforceKey     bool  `toml:"enforce_key"`
}

var Config, configLoadError = loadConfig()

func loadConfig() (RuntimeConfig, error) {
	baseDirectory := executableDir()
	config := RuntimeConfig{
		Bind:           "::",
		HTTPPort:       21114,
		NATPort:        21115,
		IDPort:         21116,
		RelayPort:      21117,
		WebSocketPort:  21118,
		WebAPIPort:     21119,
		DataDirectory:  baseDirectory,
		LogPath:        filepath.Join(baseDirectory, "rdks.log"),
		LogLevel:       "info",
		LogColor:       true,
		MaxLogSize:     10 << 20,
		MaxLogFiles:    4,
		MaxConnections: 4096,
		MaxFrameSize:   16 << 20,
		EnforceKey:     true,
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
		Extra: extraConfig{MaxConnections: config.MaxConnections, MaxFrameSize: config.MaxFrameSize, EnforceKey: config.EnforceKey},
	}
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return config, fmt.Errorf("parse config %s: %w", path, err)
	}
	config = RuntimeConfig{
		Bind: file.Base.Listen, RelayServer: file.Base.RelayServer,
		HTTPPort: file.Base.HTTPPort, NATPort: file.Base.NATPort, IDPort: file.Base.IDPort,
		RelayPort: file.Base.RelayPort, WebSocketPort: file.Base.WebSocketPort, WebAPIPort: file.Base.WebAPIPort,
		DataDirectory: file.Base.DataDirectory, LogPath: file.Log.Path,
		LogLevel: file.Log.Level, LogColor: file.Log.Color,
		MaxLogSize: file.Log.MaxSize, MaxLogFiles: file.Log.MaxFiles,
		MaxConnections: file.Extra.MaxConnections, MaxFrameSize: file.Extra.MaxFrameSize,
		EnforceKey: file.Extra.EnforceKey,
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
	if Config.Bind == "" {
		return fmt.Errorf("bind must not be empty")
	}
	if Config.DataDirectory == "" || Config.LogPath == "" {
		return fmt.Errorf("data_dir and log_path must not be empty")
	}
	if Config.MaxConnections < 1 || Config.MaxFrameSize < 1024 {
		return fmt.Errorf("invalid max_connections or max_frame_size")
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
