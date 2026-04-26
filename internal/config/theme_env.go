package config

import (
	"os"
	"strings"

	ummtheme "github.com/difof/umm/internal/theme"
)

func applyThemeEnvOverride(cfg Config) Config {
	name := strings.TrimSpace(os.Getenv("UMM_THEME"))
	if name == "" {
		return cfg
	}

	configDir, err := ResolveConfigDir()
	if err != nil {
		return cfg
	}
	catalog, err := ummtheme.Discover(configDir)
	if err != nil {
		return cfg
	}
	if _, err := catalog.Resolve(name); err != nil {
		return cfg
	}

	cfg.Theme = name
	return cfg
}
