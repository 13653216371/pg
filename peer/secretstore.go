package peer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type SecretStore interface {
	NetworkSecret() (NetworkSecret, error)
	UpdateNetworkSecret(NetworkSecret) error
}

type NetworkSecret struct {
	Secret  string    `json:"secret"`
	Network string    `json:"network"`
	Expire  time.Time `json:"expire"`
}

func (s NetworkSecret) Expired() bool {
	return time.Until(s.Expire) <= 0
}

func (s *NetworkSecret) NetworkSecret() (NetworkSecret, error) {
	return *s, nil
}

func (s *NetworkSecret) UpdateNetworkSecret(secret NetworkSecret) error {
	s.Secret = secret.Secret
	s.Network = secret.Network
	s.Expire = secret.Expire
	return nil
}

type FileSecretStore struct {
	StoreFilePath string
}

func (s *FileSecretStore) NetworkSecret() (NetworkSecret, error) {
	f, err := os.Open(s.StoreFilePath)
	if err != nil {
		return NetworkSecret{}, fmt.Errorf("file secret store(%s) open failed: %s", s.StoreFilePath, err)
	}
	defer f.Close()
	var secret NetworkSecret
	if err = json.NewDecoder(f).Decode(&secret); err != nil {
		return secret, fmt.Errorf("file secret store(%s) decode failed: %w", s.StoreFilePath, err)
	}
	return secret, nil
}

func (s *FileSecretStore) UpdateNetworkSecret(secret NetworkSecret) error {
	f, err := os.Create(s.StoreFilePath)
	if err != nil {
		return fmt.Errorf("update network secret failed: %w", err)
	}
	if err := json.NewEncoder(f).Encode(secret); err != nil {
		return fmt.Errorf("save network secret failed: %w", err)
	}
	return f.Close()
}