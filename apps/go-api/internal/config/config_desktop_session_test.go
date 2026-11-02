package config

// config_desktop_session_test.go — le secret de session et l'origine du mode desktop (ADR 0045
// D-3 et D-7, lot L13.1) : généré au premier lancement, repris ensuite, jamais le défaut public ;
// Validate accepte desktop sous LEVELUP_ENV=production et refuse un secret public ; l'origine
// autorisée dérive du port lié.

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"levelup/go-api/internal/authz"
)

var hexSecretRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// chargerDesktop charge le profil desktop sous une racine de données vide et rend la config.
func chargerDesktop(t *testing.T, data string) *AppConfig {
	t.Helper()
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	t.Setenv("LEVELUP_DATA_DIR", data)
	t.Setenv("LEVELUP_ENV", "production")
	t.Setenv("LEVELUP_DEMO_MODE", "")
	t.Setenv("LEVELUP_AUTH_MODE", "")
	t.Setenv("LEVELUP_API_HOST", "")
	t.Setenv("LEVELUP_SESSION_SECRET", "")
	cfg, err := LoadServer(LaunchModeDesktop)
	if err != nil {
		t.Fatalf("LoadServer(desktop) : %v", err)
	}
	return cfg
}

// Premier lancement : le secret est tiré, écrit sous <DataRoot>/auth/session_secret et ce n'est pas
// le défaut public. Second lancement : le même secret, les cookies posés restent valides.
func TestDesktop_SecretDeSession_GenereAuPremierLancementPuisRepris(t *testing.T) {
	data := t.TempDir()
	cfg := chargerDesktop(t, data)
	if cfg.SessionSecret == DefaultSessionSecret || !hexSecretRE.MatchString(cfg.SessionSecret) {
		t.Fatalf("secret = %q, attendu 64 caractères hexadécimaux, jamais le défaut public", cfg.SessionSecret)
	}
	path := cfg.PathResolver().SessionSecretPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fichier du secret absent : %v", err)
	}
	if got := strings.TrimSpace(string(raw)); got != cfg.SessionSecret {
		t.Errorf("fichier = %q, config = %q : ils doivent coïncider", got, cfg.SessionSecret)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Errorf("mode du fichier = %v, attendu 0600", info.Mode().Perm())
		}
	}
	if again := chargerDesktop(t, data); again.SessionSecret != cfg.SessionSecret {
		t.Errorf("second lancement : secret %q, attendu le même que le premier", again.SessionSecret)
	}
	if other := chargerDesktop(t, t.TempDir()); other.SessionSecret == cfg.SessionSecret {
		t.Error("deux racines de données vides ont tiré le même secret")
	}
}

// LEVELUP_SESSION_SECRET ne remplace pas le secret du fichier en desktop.
func TestDesktop_SecretDeSession_LEnvironnementEstIgnore(t *testing.T) {
	data := t.TempDir()
	t.Setenv("LEVELUP_SESSION_SECRET", strings.Repeat("v", 40)) // valeur de forme, pas un secret : le test prouve qu'elle est ignorée
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	t.Setenv("LEVELUP_DATA_DIR", data)
	t.Setenv("LEVELUP_DEMO_MODE", "") // la CI pose LEVELUP_DEMO_MODE=true : incompatible avec le profil desktop
	t.Setenv("LEVELUP_AUTH_MODE", "")
	t.Setenv("LEVELUP_API_HOST", "")
	cfg, err := LoadServer(LaunchModeDesktop)
	if err != nil {
		t.Fatalf("LoadServer(desktop) : %v", err)
	}
	if !hexSecretRE.MatchString(cfg.SessionSecret) {
		t.Errorf("secret = %q : la variable d'environnement a remplacé le secret du fichier", cfg.SessionSecret)
	}
}

// Un fichier trop court ou égal au défaut public est remplacé ; un fichier illisible est une erreur.
func TestDesktop_SecretDeSession_FichierInutilisable(t *testing.T) {
	for _, contenu := range []string{"", "court", DefaultSessionSecret} {
		data := t.TempDir()
		if err := os.MkdirAll(data+"/auth", 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(data+"/auth/session_secret", []byte(contenu), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := chargerDesktop(t, data)
		if !hexSecretRE.MatchString(cfg.SessionSecret) {
			t.Errorf("contenu %q : secret = %q, attendu un secret neuf de 64 caractères", contenu, cfg.SessionSecret)
		}
	}
	// Un dossier à la place du fichier : lecture impossible, le chargement échoue (rien n'est régénéré).
	data := t.TempDir()
	if err := os.MkdirAll(data+"/auth/session_secret", 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	t.Setenv("LEVELUP_DATA_DIR", data)
	t.Setenv("LEVELUP_DEMO_MODE", "")
	t.Setenv("LEVELUP_AUTH_MODE", "")
	t.Setenv("LEVELUP_API_HOST", "")
	if _, err := LoadServer(LaunchModeDesktop); err == nil {
		t.Error("LoadServer(desktop) a réussi alors que le secret de session est illisible")
	}
}

// 13.1 : desktop passe Validate sous LEVELUP_ENV=production (le refus de production est celui du
// profil hosted) et authz.Enforced y reste faux (mono-utilisateur, ADR 0045 D-7).
func TestDesktop_ValidateAccepteLaProductionEtLEnforcementResteFaux(t *testing.T) {
	cfg := chargerDesktop(t, t.TempDir())
	if !cfg.IsProduction() {
		t.Fatal("le test suppose LEVELUP_ENV=production")
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate sous desktop + production : %v", err)
	}
	if authz.Enforced(cfg.DemoMode, cfg.AuthMode) {
		t.Error("authz.Enforced vrai sous desktop : le profil est mono-utilisateur (ADR 0045 D-7)")
	}
}

// Validate refuse le secret public par défaut et un secret trop court sous desktop, comme en production.
func TestDesktop_ValidateRefuseUnSecretPublicOuTropCourt(t *testing.T) {
	cfg := chargerDesktop(t, t.TempDir())
	cfg.SessionSecret = DefaultSessionSecret
	if err := cfg.Validate(); err == nil {
		t.Error("Validate accepte le secret de session public par défaut en desktop")
	}
	cfg.SessionSecret = "court"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate accepte un secret de session trop court en desktop")
	}
}

// L'origine autorisée est celle du port lié ; LEVELUP_CORS_ORIGINS ne l'élargit pas.
func TestDesktop_OrigineEtAutoriteDerivantDuPortLie(t *testing.T) {
	t.Setenv("LEVELUP_CORS_ORIGINS", "https://autre.example")
	cfg := chargerDesktop(t, t.TempDir())
	cfg.SetBoundPort(49321)
	if got, want := cfg.DesktopAuthority(), "127.0.0.1:49321"; got != want {
		t.Errorf("DesktopAuthority() = %q, attendu %q", got, want)
	}
	if got, want := cfg.DesktopOrigin(), "http://127.0.0.1:49321"; got != want {
		t.Errorf("DesktopOrigin() = %q, attendu %q", got, want)
	}
	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "http://127.0.0.1:49321" {
		t.Errorf("CORSOrigins = %v, attendu la seule origine du port lié", cfg.CORSOrigins)
	}
}

// Hors desktop, SetBoundPort ne touche pas aux origines configurées.
func TestSetBoundPort_HorsDesktopLaisseLesOriginesIntactes(t *testing.T) {
	cfg, err := chargerServeur(t, LaunchModeDefault, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	avant := append([]string(nil), cfg.CORSOrigins...)
	cfg.SetBoundPort(9999)
	if cfg.APIPort != 9999 || strings.Join(cfg.CORSOrigins, ",") != strings.Join(avant, ",") {
		t.Errorf("port = %d, origines = %v (avant %v)", cfg.APIPort, cfg.CORSOrigins, avant)
	}
}

// Sous desktop, l'origine unique du port lié n'est pas un défaut oublié : aucun avertissement de
// production (secret, CORS) ne vise une configuration sûre par construction.
func TestDesktop_AucunAvertissementDeSecuriteSurUneConfigurationSure(t *testing.T) {
	cfg := chargerDesktop(t, t.TempDir())
	cfg.SetBoundPort(49555)
	if w := cfg.SecurityWarnings(); len(w) != 0 {
		t.Errorf("SecurityWarnings() = %v, attendu aucun avertissement sous desktop", w)
	}
}
