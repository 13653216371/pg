package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/sigcn/pg/langs"
	"github.com/sigcn/pg/secure/aescbc"
)

var (
	ErrInvalidToken = langs.Error{Code: 9000, Msg: "invalid token"}
	ErrTokenExpired = langs.Error{Code: 9001, Msg: "token expired"}
)

type JSONSecret struct {
	Network   string   `json:"n"`
	Admin     bool     `json:"adm,omitzero"`
	Alias     string   `json:"n1,omitzero"`
	Neighbors []string `json:"ns,omitempty"`
	Deadline  int64    `json:"t"`
}

type Net struct {
	ID        string
	Alias     string
	Neighbors []string
}

type Authenticator struct {
	key []byte
}

func NewAuthenticator(key string) *Authenticator {
	sum := sha256.Sum256([]byte(key))
	return &Authenticator{key: sum[:]}
}

func (auth *Authenticator) GenerateSecret(n Net, validDuration time.Duration) (string, error) {
	return auth.GenerateSecretAdmin(false, n, validDuration)
}

func (auth *Authenticator) GenerateSecretAdmin(adm bool, n Net, validDuration time.Duration) (string, error) {
	b, _ := json.Marshal(JSONSecret{
		Network:   n.ID,
		Admin:     adm,
		Alias:     n.Alias,
		Neighbors: n.Neighbors,
		Deadline:  time.Now().Add(validDuration).Unix(),
	})
	chiperData, err := aescbc.Encrypt(auth.key, b)
	return base64.URLEncoding.EncodeToString(chiperData), err
}

func (auth *Authenticator) ParseSecret(networkIDChiper string) (JSONSecret, error) {
	chiperData, err := base64.URLEncoding.DecodeString(networkIDChiper)
	if err != nil {
		return JSONSecret{}, ErrInvalidToken
	}
	plainData, err := aescbc.Decrypt(auth.key, chiperData)
	if err != nil {
		return JSONSecret{}, ErrInvalidToken
	}

	var token JSONSecret
	err = json.Unmarshal(plainData, &token)
	if err != nil {
		return JSONSecret{}, ErrInvalidToken
	}

	if time.Until(time.Unix(token.Deadline, 0)) <= 0 {
		return token, ErrTokenExpired
	}
	return token, nil
}
