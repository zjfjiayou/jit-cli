package appinfo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Save(path string, info *AppInfo) error {
	if info == nil {
		return fmt.Errorf("app info is nil")
	}

	payload := CachedAppInfo{
		FetchedAt: time.Now().UTC(),
		App:       *info,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal app info cache: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o600)
}

func Load(path string) (*CachedAppInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cached CachedAppInfo
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("parse app info cache: %w", err)
	}
	return &cached, nil
}

func Elements(info *AppInfo) []ElementDefine {
	if info == nil {
		return nil
	}

	seen := map[string]struct{}{}
	var out []ElementDefine
	var walk func(*AppInfo)
	walk = func(app *AppInfo) {
		if app == nil {
			return
		}

		keys := make([]string, 0, len(app.Elements))
		for key := range app.Elements {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			element := app.Elements[key]
			if IsPrivate(element.AccessModifier) {
				continue
			}

			fullName := strings.TrimSpace(element.FullName)
			if fullName == "" {
				fullName = key
				element.FullName = key
			}
			if _, exists := seen[fullName]; exists {
				continue
			}
			seen[fullName] = struct{}{}
			out = append(out, element)
		}

		for i := range app.ExtendApps {
			walk(&app.ExtendApps[i])
		}
	}
	walk(info)
	return out
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

func IsPrivate(accessModifier string) bool {
	return strings.EqualFold(strings.TrimSpace(accessModifier), "private")
}
