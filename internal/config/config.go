package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	APIURL string `mapstructure:"api_url"`
	Token  string `mapstructure:"token"`
	Output string `mapstructure:"output"`
	Debug  bool   `mapstructure:"debug"`
	DryRun bool   `mapstructure:"dry_run"`
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("BROTNI")
	v.AutomaticEnv()
	if err := v.BindEnv("api_url", "BROTNI_API_URL"); err != nil {
		return nil, fmt.Errorf("binding api_url env: %w", err)
	}
	if err := v.BindEnv("token", "BROTNI_TOKEN"); err != nil {
		return nil, fmt.Errorf("binding token env: %w", err)
	}

	v.SetDefault("api_url", "https://api.brotni.io")
	v.SetDefault("output", "table")
	v.SetDefault("debug", false)
	v.SetDefault("dry_run", false)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
		}
		v.AddConfigPath(".")
		v.SetConfigName(".brotni")
		v.SetConfigType("yaml")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
