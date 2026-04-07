package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvHome       = "JIT_CLI_HOME"
	defaultDir    = ".jit"
	defaultFormat = "json"
)

type GlobalConfig struct {
	CurrentProfile string `json:"current_profile,omitempty"`
	DefaultFormat  string `json:"default_format,omitempty"`
}

type Service struct {
	homeDir string
}

func ResolveHomeDir() (string, error) {
	if home := os.Getenv(EnvHome); home != "" {
		return filepath.Clean(home), nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(userHome, defaultDir), nil
}

func ResolveHome() (string, error) {
	return ResolveHomeDir()
}

func DefaultGlobalConfig() GlobalConfig {
	return normalizeGlobalConfig(GlobalConfig{})
}

func NewService(homeDir string) (*Service, error) {
	if homeDir == "" {
		var err error
		homeDir, err = ResolveHomeDir()
		if err != nil {
			return nil, err
		}
	}
	if err := EnsureBaseDir(homeDir); err != nil {
		return nil, err
	}
	return &Service{homeDir: homeDir}, nil
}

func (s *Service) HomeDir() string {
	return s.homeDir
}

func (s *Service) ConfigPath() string {
	return GlobalConfigPath(s.homeDir)
}

func (s *Service) Load() (GlobalConfig, error) {
	return Load(s.homeDir)
}

func (s *Service) Save(cfg GlobalConfig) error {
	return Save(s.homeDir, cfg)
}

func ProfilesDir(baseDir string) string {
	return filepath.Join(baseDir, "profiles")
}

func GlobalConfigPath(baseDir string) string {
	return filepath.Join(baseDir, "config.json")
}

func EnsureBaseDir(baseDir string) error {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	if err := os.MkdirAll(ProfilesDir(baseDir), 0o700); err != nil {
		return fmt.Errorf("ensure profiles dir: %w", err)
	}
	return nil
}

func Load(baseDir string) (GlobalConfig, error) {
	cfg := DefaultGlobalConfig()
	if err := EnsureBaseDir(baseDir); err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(GlobalConfigPath(baseDir))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read global config: %w", err)
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse global config: %w", err)
	}
	return normalizeGlobalConfig(cfg), nil
}

func Save(baseDir string, cfg GlobalConfig) error {
	if err := EnsureBaseDir(baseDir); err != nil {
		return err
	}
	cfg = normalizeGlobalConfig(cfg)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal global config: %w", err)
	}
	data = append(data, '\n')
	return WriteProfileFile(GlobalConfigPath(baseDir), data)
}

func WriteProfileFile(path string, data []byte) error {
	return writeFileAtomic(path, data, 0o600)
}

func normalizeGlobalConfig(cfg GlobalConfig) GlobalConfig {
	if cfg.DefaultFormat == "" {
		cfg.DefaultFormat = defaultFormat
	}
	return cfg
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("ensure dir for %s: %w", path, err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, mode); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file %s: %w", path, err)
	}
	return nil
}
