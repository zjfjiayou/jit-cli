package appinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadAndElements(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appinfo.json")
	info := &AppInfo{
		AppID: "whwy/mmm",
		Elements: map[string]ElementDefine{
			"services.PublicSvc": {
				FullName: "services.PublicSvc",
				Title:    "Public",
				Type:     "services.Meta",
			},
			"services.PrivateSvc": {
				FullName:       "services.PrivateSvc",
				Title:          "Private",
				Type:           "services.Meta",
				AccessModifier: "private",
			},
		},
		ExtendApps: []AppInfo{{
			AppID: "whwy/base",
			Elements: map[string]ElementDefine{
				"services.PublicSvc": {
					FullName: "services.PublicSvc",
					Title:    "Duplicate",
					Type:     "services.Meta",
				},
				"models.BaseModel": {
					FullName: "models.BaseModel",
					Title:    "Base Model",
					Type:     "models.NormalType",
					FieldList: []map[string]any{{
						"name": "id",
					}},
				},
			},
		}},
	}

	if err := Save(path, info); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	cached, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cached.App.AppID != "whwy/mmm" {
		t.Fatalf("AppID = %q, want whwy/mmm", cached.App.AppID)
	}

	elements := Elements(&cached.App)
	if len(elements) != 2 {
		t.Fatalf("len(Elements) = %d, want 2", len(elements))
	}
	if elements[0].FullName != "services.PublicSvc" || elements[1].FullName != "models.BaseModel" {
		t.Fatalf("unexpected flattened elements: %#v", elements)
	}
}
