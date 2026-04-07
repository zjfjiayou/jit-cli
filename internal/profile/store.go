package profile

import (
	"errors"
	"strings"

	keyring "github.com/zalando/go-keyring"
)

const defaultServiceName = "jit-cli"

type defaultSecretStore struct{}

func (defaultSecretStore) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (defaultSecretStore) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (defaultSecretStore) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, keyring.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found")
}
