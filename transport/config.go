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
	Bind           string `toml:"bind"`
	BasePort       int    `toml:"base_port"`
	RelayServer    string `toml:"relay_server"`
	DataDirectory  string `toml:"data_dir"`
	LogPath        string `toml:"log_path"`
	LogLevel       string `toml:"log_level"`
	LogColor       bool   `toml:"log_color"`
	MaxConnections int64  `toml:"max_connections"`
	MaxFrameSize   uint   `toml:"max_frame_size"`
}

var Config, configLoadError = loadConfig()

func loadConfig() (RuntimeConfig, error) {
	baseDirectory := executableDir()
	config := RuntimeConfig{
		Bind:           "::",
		BasePort:       21116,
		DataDirectory:  baseDirectory,
		LogPath:        filepath.Join(baseDirectory, "rdks.log"),
		LogLevel:       "info",
		LogColor:       true,
		MaxConnections: 4096,
		MaxFrameSize:   16 << 20,
	}
	path, required := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) && !required {
		return config, nil
	}
	if err != nil {
		return config, fmt.Errorf("read config %s: %w", path, err)
	}
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("parse config %s: %w", path, err)
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
	if Config.BasePort < 3 || Config.BasePort > 65532 {
		return fmt.Errorf("base_port must be between 3 and 65532")
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
	return os.MkdirAll(Config.DataDirectory, 0700)
}
