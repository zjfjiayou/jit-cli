package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"jit-cli/internal/config"
)

type Config struct {
	Server     string `json:"server"`
	DefaultApp string `json:"default_app,omitempty"`
}

type Summary struct {
	Name       string `json:"name"`
	Server     string `json:"server"`
	DefaultApp string `json:"default_app,omitempty"`
	HasToken   bool   `json:"has_token"`
	Current    bool   `json:"current"`
}

type SecretStore interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type Option func(*Manager)

type Manager struct {
	homeDir     string
	serviceName string
	secretStore SecretStore
}

func NewManager(homeDir string, opts ...Option) (*Manager, error) {
	if homeDir == "" {
		var err error
		homeDir, err = config.ResolveHomeDir()
		if err != nil {
			return nil, err
		}
	}
	if err := config.EnsureBaseDir(homeDir); err != nil {
		return nil, err
	}

	manager := &Manager{
		homeDir:     homeDir,
		serviceName: defaultServiceName,
		secretStore: defaultSecretStore{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(manager)
		}
	}
	return manager, nil
}

func WithSecretStore(store SecretStore) Option {
	return func(m *Manager) {
		if store != nil {
			m.secretStore = store
		}
	}
}

func WithServiceName(name string) Option {
	return func(m *Manager) {
		if strings.TrimSpace(name) != "" {
			m.serviceName = strings.TrimSpace(name)
		}
	}
}

func (m *Manager) SaveProfile(name string, cfg Config) error {
	var err error
	name, err = requireProfileName(name)
	if err != nil {
		return err
	}

	server, err := NormalizeServer(cfg.Server)
	if err != nil {
		return err
	}
	cfg.Server = server

	if err := validateDefaultApp(cfg.DefaultApp); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile config: %w", err)
	}
	data = append(data, '\n')
	return config.WriteProfileFile(m.ConfigPath(name), data)
}

func (m *Manager) LoadProfile(name string) (Config, error) {
	var cfg Config
	var err error
	name, err = requireProfileName(name)
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(m.ConfigPath(name))
	if err != nil {
		return cfg, fmt.Errorf("read profile config %q: %w", name, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse profile config %q: %w", name, err)
	}

	cfg.Server, err = NormalizeServer(cfg.Server)
	if err != nil {
		return cfg, fmt.Errorf("invalid server in profile %q: %w", name, err)
	}
	if err := validateDefaultApp(cfg.DefaultApp); err != nil {
		return cfg, fmt.Errorf("invalid default app in profile %q: %w", name, err)
	}
	return cfg, nil
}

func (m *Manager) DeleteProfile(name string) error {
	var err error
	name, err = requireProfileName(name)
	if err != nil {
		return err
	}
	if err := m.RemoveToken(name); err != nil {
		return err
	}
	if err := os.RemoveAll(m.ProfileDir(name)); err != nil {
		return fmt.Errorf("delete profile dir: %w", err)
	}
	return nil
}

func (m *Manager) ListProfiles() ([]string, error) {
	entries, err := os.ReadDir(config.ProfilesDir(m.homeDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profiles dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (m *Manager) Summaries(currentProfile string) ([]Summary, error) {
	names, err := m.ListProfiles()
	if err != nil {
		return nil, err
	}

	items := make([]Summary, 0, len(names))
	for _, name := range names {
		cfg, err := m.LoadProfile(name)
		if err != nil {
			return nil, err
		}
		token, err := m.LoadToken(name)
		if err != nil {
			return nil, err
		}
		items = append(items, Summary{
			Name:       name,
			Server:     cfg.Server,
			DefaultApp: cfg.DefaultApp,
			Current:    name == currentProfile,
			HasToken:   token != "",
		})
	}
	return items, nil
}

func (m *Manager) SaveToken(name, token string) error {
	var err error
	name, err = requireProfileName(name)
	if err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if err := ValidatePAT(token); err != nil {
		return err
	}
	if err := os.MkdirAll(m.ProfileDir(name), 0o700); err != nil {
		return fmt.Errorf("ensure profile dir: %w", err)
	}
	account := m.accountName(name)
	if m.secretStore != nil {
		if err := m.secretStore.Set(m.serviceName, account, token); err == nil {
			_ = os.Remove(m.CredentialsPath(name))
			return nil
		}
	}
	return config.WriteProfileFile(m.CredentialsPath(name), append([]byte(token), '\n'))
}

func (m *Manager) LoadToken(name string) (string, error) {
	var err error
	name, err = requireProfileName(name)
	if err != nil {
		return "", err
	}
	account := m.accountName(name)
	if m.secretStore != nil {
		token, err := m.secretStore.Get(m.serviceName, account)
		if cleaned := strings.TrimSpace(token); err == nil && cleaned != "" {
			return cleaned, nil
		}
	}
	data, err := os.ReadFile(m.CredentialsPath(name))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read credentials file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (m *Manager) RemoveToken(name string) error {
	var err error
	name, err = requireProfileName(name)
	if err != nil {
		return err
	}
	account := m.accountName(name)
	if m.secretStore != nil {
		if err := m.secretStore.Delete(m.serviceName, account); err != nil && !isNotFoundError(err) {
			return fmt.Errorf("delete keychain credential: %w", err)
		}
	}
	if err := os.Remove(m.CredentialsPath(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete credentials file: %w", err)
	}
	return nil
}

func (m *Manager) Exists(name string) bool {
	_, err := os.Stat(m.ConfigPath(name))
	return err == nil
}

func (m *Manager) ProfileDir(name string) string {
	return filepath.Join(config.ProfilesDir(m.homeDir), NormalizeProfileName(name))
}

func (m *Manager) ConfigPath(name string) string {
	return filepath.Join(m.ProfileDir(name), "config.json")
}

func (m *Manager) CredentialsPath(name string) string {
	return filepath.Join(m.ProfileDir(name), "credentials")
}

func (m *Manager) accountName(name string) string {
	return m.serviceName + ":" + NormalizeProfileName(name)
}

func requireProfileName(name string) (string, error) {
	name = NormalizeProfileName(name)
	if name == "" {
		return "", errors.New("profile name is required")
	}
	return name, nil
}

func validateDefaultApp(app string) error {
	if app == "" {
		return nil
	}
	_, _, err := ParseApp(app)
	return err
}

func NormalizeProfileName(name string) string {
	return strings.TrimSpace(name)
}

func ProfileNameFromServer(server string) (string, error) {
	normalized, err := NormalizeServer(server)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("parse server url: %w", err)
	}

	replacer := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	name := replacer.ReplaceAllString(parsed.Hostname(), "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "", errors.New("cannot derive profile name from server")
	}
	return name, nil
}

func NormalizeServer(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", errors.New("server is required")
	}

	parsed, err := url.Parse(server)
	if err != nil {
		return "", fmt.Errorf("parse server url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("server must include scheme and host, e.g. https://demo.jit.cn")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("server must not include query or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func ParseApp(app string) (string, string, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return "", "", errors.New("app is required")
	}
	parts := strings.Split(app, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("app must be in org/app format")
	}
	return parts[0], parts[1], nil
}

func ResolveORMApp(app string) (string, error) {
	org, _, err := ParseApp(app)
	if err != nil {
		return "", err
	}
	return org + "/JitORM", nil
}
