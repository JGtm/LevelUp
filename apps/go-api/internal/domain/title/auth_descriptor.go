// Package title — auth_descriptor.go : descripteur d'acquisition auth par titre
// (MT-02 / PMT-2). Value object portant les paramètres title-specific de la
// chaîne d'échange de tokens (XSTS audience, spartan audience, clearance URL,
// SISU app/title id, scopes OAuth, RP Xbox Live).
//
// Source de vérité title-agnostic : un 2e titre déclare ses propres valeurs dans
// config/titles/{slug}/auth.toml ; le défaut Halo (DefaultHaloAuthDescriptor) est
// câblé byte-pour-byte aux const ACTUELLES du package platform/auth → zéro
// changement de comportement tant que le seam consomme ce défaut.
package title

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ErrAuthNotConfigured est retourné quand un titre ne déclare pas de section
// [auth] (pas d'auth.toml). Le caller dégrade proprement (jamais de fallback
// silencieux vers les valeurs d'un autre titre).
var ErrAuthNotConfigured = errors.New("title: auth descriptor non configuré (auth.toml absent)")

// AuthDescriptor porte les paramètres d'acquisition auth d'un titre (MT-02).
type AuthDescriptor struct {
	// XSTSAudience — audience XSTS du titre (ex. Halo : https://prod.xsts.halowaypoint.com/).
	XSTSAudience string
	// SpartanAudience — audience du body spartan-token (ex. Halo : urn:343:s3:services).
	SpartanAudience string
	// SpartanTokenURL — endpoint spartan-token.
	SpartanTokenURL string
	// ClearanceURL — endpoint clearance (le segment titles/hi est Halo-specific).
	ClearanceURL string
	// SISUAppID / SISUTitleID — identifiants Xbox du flow SISU device-code.
	SISUAppID   string
	SISUTitleID string
	// XboxLiveRelyingParty — RP des tokens XSTS Xbox Live (RTA) ; partageable
	// cross-titre mais modélisé pour ne pas re-hardcoder.
	XboxLiveRelyingParty string
	// OAuthScopes — scopes OAuth Xbox (vraisemblablement constants cross-titre).
	OAuthScopes []string
}

// DefaultHaloAuthDescriptor retourne le descripteur Halo Infinite câblé aux
// valeurs ACTUELLES du package platform/auth (golden de parité MT-02). Toute
// dérive de ces littéraux doit être répercutée ici ET dans platform/auth.
func DefaultHaloAuthDescriptor() AuthDescriptor {
	return AuthDescriptor{
		XSTSAudience:         "https://prod.xsts.halowaypoint.com/",
		SpartanAudience:      "urn:343:s3:services",
		SpartanTokenURL:      "https://settings.svc.halowaypoint.com/spartan-token",
		ClearanceURL:         "https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active",
		SISUAppID:            "000000004c20a908",
		SISUTitleID:          "144209987",
		XboxLiveRelyingParty: "http://xboxlive.com",
		OAuthScopes:          []string{"Xboxlive.signin", "Xboxlive.offline_access"},
	}
}

// authTOML est la projection brute de la section [auth] de auth.toml.
type authTOML struct {
	Meta authMetaSection `toml:"meta"`
	Auth authSection     `toml:"auth"`
}

type authMetaSection struct {
	TitleSlug     string `toml:"title_slug"`
	SchemaVersion int    `toml:"schema_version"`
}

type authSection struct {
	XSTSAudience         string   `toml:"xsts_audience"`
	SpartanAudience      string   `toml:"spartan_audience"`
	SpartanTokenURL      string   `toml:"spartan_token_url"`
	ClearanceURL         string   `toml:"clearance_url"`
	SISUAppID            string   `toml:"sisu_app_id"`
	SISUTitleID          string   `toml:"sisu_title_id"`
	XboxLiveRelyingParty string   `toml:"xbox_live_relying_party"`
	OAuthScopes          []string `toml:"oauth_scopes"`
}

// LoadAuthDescriptor charge le descripteur auth d'un titre depuis
// config/titles/{slug}/auth.toml. Retourne ErrAuthNotConfigured si le fichier est
// absent (le caller décide alors d'utiliser DefaultHaloAuthDescriptor pour le
// titre par défaut, ou de refuser l'acquisition pour un titre sans auth).
func LoadAuthDescriptor(repoRoot, slug string) (*AuthDescriptor, error) {
	path := filepath.Join(repoRoot, "config", "titles", slug, "auth.toml")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrAuthNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadAuthDescriptorFromBytes(path, raw)
}

// LoadAuthDescriptorFromBytes parse et valide un payload auth.toml en mémoire.
func LoadAuthDescriptorFromBytes(path string, raw []byte) (*AuthDescriptor, error) {
	var doc authTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var errs []error
	if doc.Meta.TitleSlug == "" {
		errs = append(errs, fmt.Errorf("[meta].title_slug manquant"))
	}
	if doc.Meta.SchemaVersion <= 0 {
		errs = append(errs, fmt.Errorf("[meta].schema_version doit être > 0 (reçu %d)", doc.Meta.SchemaVersion))
	}

	desc := AuthDescriptor{
		XSTSAudience:         strings.TrimSpace(doc.Auth.XSTSAudience),
		SpartanAudience:      strings.TrimSpace(doc.Auth.SpartanAudience),
		SpartanTokenURL:      strings.TrimSpace(doc.Auth.SpartanTokenURL),
		ClearanceURL:         strings.TrimSpace(doc.Auth.ClearanceURL),
		SISUAppID:            strings.TrimSpace(doc.Auth.SISUAppID),
		SISUTitleID:          strings.TrimSpace(doc.Auth.SISUTitleID),
		XboxLiveRelyingParty: strings.TrimSpace(doc.Auth.XboxLiveRelyingParty),
		OAuthScopes:          doc.Auth.OAuthScopes,
	}
	errs = append(errs, desc.validate()...)

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}
	return &desc, nil
}

// validate vérifie que les champs requis sont présents et bien formés.
func (d AuthDescriptor) validate() []error {
	var errs []error
	required := map[string]string{
		"xsts_audience":     d.XSTSAudience,
		"spartan_audience":  d.SpartanAudience,
		"spartan_token_url": d.SpartanTokenURL,
		"clearance_url":     d.ClearanceURL,
		"sisu_app_id":       d.SISUAppID,
		"sisu_title_id":     d.SISUTitleID,
	}
	for key, val := range required {
		if val == "" {
			errs = append(errs, fmt.Errorf("[auth].%s manquant", key))
		}
	}
	for _, key := range []string{d.SpartanTokenURL, d.ClearanceURL} {
		if key != "" && !strings.HasPrefix(key, "https://") {
			errs = append(errs, fmt.Errorf("[auth] URL non-https: %q", key))
		}
	}
	if len(d.OAuthScopes) == 0 {
		errs = append(errs, fmt.Errorf("[auth].oauth_scopes vide"))
	}
	return errs
}
