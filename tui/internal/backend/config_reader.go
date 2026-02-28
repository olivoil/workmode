package backend

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// tomlConfig mirrors the TOML config structure.
type tomlConfig struct {
	General struct {
		StateDir    string `toml:"state_dir"`
		MaxParallel int    `toml:"max_parallel"`
	} `toml:"general"`
	Trigger []tomlTrigger `toml:"trigger"`
}

// tomlTrigger mirrors a [[trigger]] entry in the TOML config.
type tomlTrigger struct {
	Name        string `toml:"name"`
	Type        string `toml:"type"`
	Permissions string `toml:"permissions"`
	Skill       string `toml:"skill"`
	Prompt      string `toml:"prompt"`
	WorkingDir  string `toml:"working_dir"`
	Cooldown    int    `toml:"cooldown"`
	Check       string `toml:"check"`
	Interval    string `toml:"interval"`
	Cron        string `toml:"cron"`
	Watch       string `toml:"watch"`
	Pattern     string `toml:"pattern"`
	Settle      int    `toml:"settle"`
	Retry       string `toml:"retry"`
	RetryMax    int    `toml:"retry_max"`
	RetryDelay  int    `toml:"retry_delay"`
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath(appName string) string {
	if p := os.Getenv("WORKMODE_CONFIG"); p != "" {
		return p
	}
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, appName, "config.toml")
}

// ReadConfigFile reads and parses the TOML config directly.
func ReadConfigFile(path string) (Config, error) {
	var tc tomlConfig
	if _, err := toml.DecodeFile(path, &tc); err != nil {
		return Config{}, err
	}

	cfg := Config{}
	cfg.General.StateDir = tc.General.StateDir
	cfg.General.MaxParallel = tc.General.MaxParallel

	for _, t := range tc.Trigger {
		cfg.Triggers = append(cfg.Triggers, Trigger{
			Name:        t.Name,
			Type:        t.Type,
			Permissions: t.Permissions,
			Skill:       t.Skill,
			Prompt:      t.Prompt,
			WorkingDir:  t.WorkingDir,
			Cooldown:    t.Cooldown,
			Check:       t.Check,
			Interval:    t.Interval,
			Cron:        t.Cron,
			Watch:       t.Watch,
			Pattern:     t.Pattern,
			Settle:      t.Settle,
			Retry:       t.Retry,
			RetryMax:    t.RetryMax,
			RetryDelay:  t.RetryDelay,
		})
	}
	return cfg, nil
}
