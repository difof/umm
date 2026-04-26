package config

import (
	"os"
	"path/filepath"

	"github.com/difof/errors"
)

func ResolveWritePath() (string, error) {
	configDir, err := ResolveConfigDir()
	if err != nil {
		return "", errors.Wrap(err)
	}

	return filepath.Join(configDir, "umm.yml"), nil
}

func ResolveConfigDir() (string, error) {
	base, err := configBaseDir()
	if err != nil {
		return "", errors.Wrap(err)
	}

	return filepath.Join(base, "umm"), nil
}

func FindUserPath() (string, bool, error) {
	paths, err := candidatePaths()
	if err != nil {
		return "", false, errors.Wrap(err)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, true, nil
		} else if !os.IsNotExist(err) {
			return "", false, errors.Wrap(err)
		}
	}

	path, err := ResolveWritePath()
	if err != nil {
		return "", false, errors.Wrap(err)
	}

	return path, false, nil
}

func candidatePaths() ([]string, error) {
	paths := []string{}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "umm", "umm.yml"))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err)
	}
	paths = append(paths, filepath.Join(home, ".config", "umm", "umm.yml"))

	return dedupe(paths), nil
}

func configBaseDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err)
	}

	return filepath.Join(home, ".config"), nil
}

func dedupe(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
