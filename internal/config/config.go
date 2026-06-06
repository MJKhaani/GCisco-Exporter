package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr         string         `yaml:"listen_addr"`
	CollectionInterval string         `yaml:"collection_interval"`
	Workers            int            `yaml:"workers"`
	Timeout            string         `yaml:"timeout"`
	CacheTTL           string         `yaml:"cache_ttl"`
	Devices            []Device       `yaml:"devices"`
	Credentials        []Credential   `yaml:"credentials"`
	DebugCapture       DebugCapture   `yaml:"debug_capture"`
	Debug              bool           `yaml:"debug"`
	LegacyCiphers      bool           `yaml:"legacy_ciphers"`
	MetricSelection    MetricSelection `yaml:"metric_selection"`
}

type MetricSelection struct {
	HardwareHealth   bool `yaml:"hardware_health"`
	Interfaces       bool `yaml:"interfaces"`
	Layer2Stability  bool `yaml:"layer2_stability"`
	EtherChannel     bool `yaml:"etherchannel"`
	Layer3Tables     bool `yaml:"layer3_tables"`
	SystemMaintenance bool `yaml:"system_maintenance"`
	CapacityTrends   bool `yaml:"capacity_trends"`
}

type Device struct {
	Name              string   `yaml:"name"`
	Host              string   `yaml:"host"`
	Port              int      `yaml:"port"`
	Type              string   `yaml:"type"`
	CredentialName    string   `yaml:"credential"`
	CollectionMethods []string `yaml:"collection_methods"`
	ProcessLimit      int      `yaml:"process_limit"`
}

type Credential struct {
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"key_file"`
}

type DebugCapture struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	setDefaults(&cfg)
	return &cfg, nil
}

func (c *Config) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

func setDefaults(cfg *Config) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":9427"
	}
	if cfg.CollectionInterval == "" {
		cfg.CollectionInterval = "60s"
	}
	if cfg.Workers == 0 {
		cfg.Workers = 50
	}
	if cfg.Timeout == "" {
		cfg.Timeout = "60s"
	}
	if cfg.CacheTTL == "" {
		cfg.CacheTTL = "120s"
	}
	if cfg.DebugCapture.Path == "" {
		cfg.DebugCapture.Path = "./captures"
	}
	for i := range cfg.Devices {
		if cfg.Devices[i].Port == 0 {
			cfg.Devices[i].Port = 22
		}
		if cfg.Devices[i].Type == "" {
			cfg.Devices[i].Type = "auto"
		}
		if len(cfg.Devices[i].CollectionMethods) == 0 {
			cfg.Devices[i].CollectionMethods = []string{"json", "restconf", "netconf", "ssh"}
		}
		if cfg.Devices[i].ProcessLimit == 0 {
			cfg.Devices[i].ProcessLimit = 10
		}
	}
	if !cfg.MetricSelection.HardwareHealth && !cfg.MetricSelection.Interfaces && !cfg.MetricSelection.Layer2Stability && !cfg.MetricSelection.EtherChannel && !cfg.MetricSelection.Layer3Tables && !cfg.MetricSelection.SystemMaintenance && !cfg.MetricSelection.CapacityTrends {
		cfg.MetricSelection.HardwareHealth = true
		cfg.MetricSelection.Interfaces = true
		cfg.MetricSelection.Layer2Stability = true
		cfg.MetricSelection.EtherChannel = true
		cfg.MetricSelection.Layer3Tables = true
		cfg.MetricSelection.SystemMaintenance = true
		cfg.MetricSelection.CapacityTrends = true
	}
}
