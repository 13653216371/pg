package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sigcn/pg/langs"
	"github.com/sigcn/pg/peermap/admin/types"
	"github.com/sigcn/pg/peermap/auth"
	"github.com/sigcn/pg/peermap/config"
	"github.com/sigcn/pg/peermap/oidc"
)

var (
	Version      string = "dev"
	ErrForbidden        = langs.Error{Code: 10000, Msg: "forbidden"}
)

type contextKey string

type AdministratorV1 struct {
	Config    config.Config
	Auth      *auth.Authenticator
	PeerStore types.PeerStore
	Grant     oidc.Grant

	mux      http.ServeMux
	initOnce sync.Once
}

func (a *AdministratorV1) init() {
	a.initOnce.Do(func() {
		a.mux.HandleFunc("GET /pg/apis/v1/admin/peers", a.handleQueryPeers)
		a.mux.HandleFunc("GET /pg/apis/v1/admin/psns.json", a.handleDownloadSecret)
		a.mux.HandleFunc("GET /pg/apis/v1/admin/server_info", a.handleQueryServerInfo)
	})
}

func (a *AdministratorV1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.init()
	token := r.Header.Get("X-Token")
	secret, err := a.Auth.ParseSecret(token)
	if err != nil {
		langs.Err(err).MarshalTo(w)
		return
	}
	if !secret.Admin {
		ErrForbidden.MarshalTo(w)
		return
	}
	if time.Until(time.Unix(secret.Deadline, 0)) <
		a.Config.SecretValidityPeriod-a.Config.SecretRotationPeriod {
		if newSecret, err := a.Grant(secret.Network, "PG_ADM"); err == nil {
			b, _ := json.Marshal(newSecret)
			w.Header().Add("X-Set-Token", string(b))
		}
	}
	a.mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey("secret"), secret)))
}

func (a *AdministratorV1) handleQueryPeers(w http.ResponseWriter, r *http.Request) {
	peers, err := a.PeerStore.Peers(r.Context().Value(contextKey("secret")).(auth.JSONSecret).Network)
	if err != nil {
		langs.Err(err).MarshalTo(w)
		return
	}
	langs.Data[any]{Data: peers}.MarshalTo(w)
}

func (a *AdministratorV1) handleDownloadSecret(w http.ResponseWriter, r *http.Request) {
	secret := r.Context().Value(contextKey("secret")).(auth.JSONSecret)
	secretJSON, err := a.Grant(secret.Network, "")
	if err != nil {
		langs.Err(err).MarshalTo(w)
		return
	}
	fileName := fmt.Sprintf("%s_psns.json", secret.Network)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	json.NewEncoder(w).Encode(secretJSON)
	slog.Info("Generate a secret", "network", secret.Network)
}

func (a *AdministratorV1) handleQueryServerInfo(w http.ResponseWriter, r *http.Request) {
	info, err := readBuildInfo()
	if err != nil {
		langs.Err(err).MarshalTo(w)
	}
	langs.Data[any]{Data: serverInfo{Version: Version, buildInfo: info}}.MarshalTo(w)
}

type buildInfo struct {
	GoVersion   string `json:"go_version"`
	VCSRevision string `json:"vcs_revision"`
	VCSTime     string `json:"vcs_time"`
}

func readBuildInfo() (buildInfo buildInfo, err error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		err = errors.ErrUnsupported
		return
	}
	buildInfo.GoVersion = info.GoVersion
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			buildInfo.VCSRevision = s.Value
			continue
		}
		if s.Key == "vcs.time" {
			buildInfo.VCSTime = s.Value
		}
	}
	return
}

type serverInfo struct {
	Version string `json:"version"`
	buildInfo
}
