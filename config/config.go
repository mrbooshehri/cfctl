package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/viper"
)

type Config struct {
	Active string            `mapstructure:"active"`
	Tokens map[string]string `mapstructure:"tokens"`
}

// ActiveToken returns the token value for the currently active account.
func (c *Config) ActiveToken() string {
	if c.Tokens == nil || c.Active == "" {
		return ""
	}
	return c.Tokens[c.Active]
}

// AddToken adds or replaces a named token and marks it as active.
func (c *Config) AddToken(name, token string) {
	if c.Tokens == nil {
		c.Tokens = make(map[string]string)
	}
	c.Tokens[name] = token
	c.Active = name
}

// Names returns a sorted list of account names.
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Tokens))
	for n := range c.Tokens {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cfctl"), nil
}

func Load() (*Config, error) {
	d, err := dir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(d)
	viper.SetEnvPrefix("CFCTL")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Migrate legacy single-token config (token = "...").
	if len(cfg.Tokens) == 0 {
		if legacy := viper.GetString("token"); legacy != "" {
			cfg.Tokens = map[string]string{"default": legacy}
			cfg.Active = "default"
		}
	}

	// CFCTL_TOKEN env var overrides active token for this session.
	if envToken := os.Getenv("CFCTL_TOKEN"); envToken != "" {
		if cfg.Tokens == nil {
			cfg.Tokens = make(map[string]string)
		}
		cfg.Tokens["env"] = envToken
		cfg.Active = "env"
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}

	var content string
	content += fmt.Sprintf("active = %q\n", cfg.Active)
	if len(cfg.Tokens) > 0 {
		content += "\n[tokens]\n"
		for _, name := range cfg.Names() {
			content += fmt.Sprintf("  %q = %q\n", name, cfg.Tokens[name])
		}
	}

	path := filepath.Join(d, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}
