package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fakeSecretStore struct {
	data      map[string]string
	setErr    error
	getErr    error
	deleteErr error
}

func (f *fakeSecretStore) Get(service, user string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.data[service+"/"+user], nil
}

func (f *fakeSecretStore) Set(service, user, password string) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.data == nil {
		f.data = map[string]string{}
	}
	f.data[service+"/"+user] = password
	return nil
}

func (f *fakeSecretStore) Delete(service, user string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.data, service+"/"+user)
	return nil
}

func TestProfileCRUD(t *testing.T) {
	home := t.TempDir()
	mgr, err := NewManager(home, WithSecretStore(&fakeSecretStore{}))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	name := "demo"
	want := Config{
		Server:     "https://demo.jit.cn",
		DefaultApp: "wanyun/JitAi",
	}

	if err := mgr.SaveProfile(name, want); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	got, err := mgr.LoadProfile(name)
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadProfile() = %#v, want %#v", got, want)
	}

	list, err := mgr.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(list) != 1 || list[0] != "demo" {
		t.Fatalf("ListProfiles() = %#v, want [demo]", list)
	}

	if err := mgr.DeleteProfile(name); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	if _, err := os.Stat(mgr.ProfileDir(name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile dir should be removed, stat err = %v", err)
	}
}

func TestParsePAT(t *testing.T) {
	parts, err := ParsePAT("jit_pat_abc123_def456")
	if err != nil {
		t.Fatalf("ParsePAT(valid) error = %v", err)
	}
	if parts.TokenID != "abc123" || parts.Secret != "def456" {
		t.Fatalf("ParsePAT(valid) = %#v", parts)
	}

	invalid := []string{
		"",
		"jit_pat_only_one_part",
		"jit_pat__secret",
		"jit_pat_token_",
		"pat_xxx_yyy",
	}
	for _, token := range invalid {
		if err := ValidatePAT(token); err == nil {
			t.Fatalf("ValidatePAT(%q) should fail", token)
		}
	}
}

func TestTokenFallbackToCredentialFile(t *testing.T) {
	home := t.TempDir()
	store := &fakeSecretStore{
		setErr: errors.New("keychain unavailable"),
		getErr: errors.New("keychain unavailable"),
	}
	mgr, err := NewManager(home, WithSecretStore(store))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	token := "jit_pat_tokenid_secretid"
	if err := mgr.SaveToken("demo", token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	credPath := mgr.CredentialsPath("demo")
	content, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("ReadFile(credentials) error = %v", err)
	}
	if string(content) != token+"\n" {
		t.Fatalf("credential content = %q, want %q", string(content), token+"\n")
	}

	info, err := os.Stat(credPath)
	if err != nil {
		t.Fatalf("Stat(credentials) error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("credential file perm = %o, want 600", got)
	}

	got, err := mgr.LoadToken("demo")
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if got != token {
		t.Fatalf("LoadToken() = %q, want %q", got, token)
	}
}

func TestLoadTokenPrefersKeychain(t *testing.T) {
	home := t.TempDir()
	store := &fakeSecretStore{data: map[string]string{}}
	mgr, err := NewManager(home, WithSecretStore(store))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(mgr.CredentialsPath("demo")), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(mgr.CredentialsPath("demo"), []byte("jit_pat_file_token_secret\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store.data["jit-cli/jit-cli:demo"] = "jit_pat_keychain_token_secret"
	got, err := mgr.LoadToken("demo")
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if got != "jit_pat_keychain_token_secret" {
		t.Fatalf("LoadToken() = %q, want keychain token", got)
	}
}
