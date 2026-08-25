// Package auth — multi_user_token_store.go : persistance multi-user des tokens RTA.
//
// Layout : data/auth/watcher_tokens/{xuid}.json — 1 fichier par utilisateur.
// Décision D4 (cf. SPRINT_XBOX_SSO §0bis / thought_log 2026-05-18) :
//   - Source unique des tokens RTA d'un user (pas de duplication dans sync_meta).
//   - Write-to-temp + os.Rename atomique → zéro contention sur writes parallèles.
//   - Permissions 0600 fichiers / 0700 répertoire.
//   - Au boot : LoadAll() scanne le dossier et reconstruit le map RAM.
//
// Depuis ADR 0023 Phase 5 (2026-08-25), c'est la SEULE source de credentials
// auth du projet : aucun chemin de code ne lit plus sync_meta.oauth_refresh_token,
// sync_meta.msal_token_cache, la variable d'environnement de refresh token, ni le store
// mono-user data/auth/watcher_tokens.json comme credential. Le TokenStore
// mono-user (token_store.go) ne sert plus qu'à l'état propre du watcher RTA
// (access_token + XSTS), jamais de refresh token d'un autre joueur.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UserTokens regroupe les tokens persistés pour un utilisateur Xbox SSO.
//
// Le `OAuthRefreshToken` est le refresh_token OAuth v2 brut Microsoft (SSO web,
// cmd/token-capture device flow, cmd/token-import). SEULE credential de refresh
// depuis le retrait de MSAL (2026-07-15) et de ses caches (ADR 0023 Phase 5).
// Rotaté par Microsoft à chaque usage (cf. ADR 0023) — toute mise à jour passe
// par UpdateOAuthRefreshToken pour préserver l'atomicité.
//
// NB : les entrées prod écrites avant Phase 5 peuvent encore contenir une clé
// JSON `msal_cache_json` — elle est simplement ignorée au décodage et disparaît
// à la première réécriture du fichier (rotation, ~50 min).
type UserTokens struct {
	XUID              string    `json:"xuid"`
	Gamertag          string    `json:"gamertag"`
	XSTSToken         string    `json:"xsts_token"`
	XSTSUserHash      string    `json:"xsts_user_hash"`
	XSTSExpiresAt     time.Time `json:"xsts_expires_at"`
	AccessToken       string    `json:"access_token,omitempty"`
	OAuthExpiresAt    time.Time `json:"oauth_expires_at,omitempty"`
	OAuthRefreshToken string    `json:"oauth_refresh_token,omitempty"`
	// TokenClientFamily : famille du CLIENT OAuth qui a émis le token (provenance,
	// AU4/F12). Détermine le préfixe RpsTicket de l'échange XBL user-token
	// (TokenFamilyAzure → "d=", TokenFamilyXboxNative → "t="). Apprise et posée à
	// l'acquisition/refresh (ExchangeRefreshTokenWithRotation sait quel client a
	// répondu). Vide = provenance inconnue (entrées antérieures à F12) → l'échange
	// retombe sur le retry aveugle d=→t= (migration douce).
	TokenClientFamily string    `json:"token_client_family,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// ReauthRequired : true quand le refresh silencieux a définitivement échoué
	// (refresh_token Microsoft révoqué/expiré) → l'utilisateur DOIT se reconnecter
	// via le SSO Xbox pour re-semer un cache/RT valide. Le front l'expose en
	// bannière (cf. /bootstrap.reauth_required). Remis à false dès qu'un refresh
	// réussit ou après une ré-authentification interactive (PR-B).
	ReauthRequired   bool      `json:"reauth_required,omitempty"`
	ReauthDetectedAt time.Time `json:"reauth_detected_at,omitempty"`

	// LastAuthError* : dernier échec OAuth permanent observé par le resolver
	// (classe "config" ou "revoked", cf. auth.AuthErrorClass). Effacés au
	// premier refresh réussi. Affichés par le dashboard admin « Santé des
	// tokens ». Le message ne contient JAMAIS de token/secret.
	LastAuthErrorClass string    `json:"last_auth_error_class,omitempty"`
	LastAuthError      string    `json:"last_auth_error,omitempty"`
	LastAuthErrorAt    time.Time `json:"last_auth_error_at,omitempty"`
}

// Familles de client OAuth (provenance du token, AU4/F12). Déterminent le préfixe
// RpsTicket de l'échange XBL user-token.
const (
	// TokenFamilyAzure : app Azure (MSAL, SSO web, refresh v2) → RpsTicket "d=".
	TokenFamilyAzure = "azure"
	// TokenFamilyXboxNative : client Xbox natif (SISU device-flow, refresh MSA) → "t=".
	TokenFamilyXboxNative = "xbox_native"
)

// IsXSTSValid retourne true si le XSTS est encore valide (avec marge).
func (u *UserTokens) IsXSTSValid(margin time.Duration) bool {
	if u.XSTSToken == "" {
		return false
	}
	return time.Now().Add(margin).Before(u.XSTSExpiresAt)
}

// AuthHeader retourne l'header XBL3.0 pour l'authentification RTA.
// Retourne "" si les champs nécessaires sont vides.
func (u *UserTokens) AuthHeader() string {
	if u.XSTSToken == "" || u.XSTSUserHash == "" {
		return ""
	}
	return fmt.Sprintf("XBL3.0 x=%s;%s", u.XSTSUserHash, u.XSTSToken)
}

// MultiUserTokenStore persiste les UserTokens d'un ensemble d'utilisateurs.
// Thread-safe. Chaque fichier `{xuid}.json` est écrit/lu de manière atomique.
type MultiUserTokenStore struct {
	dir string
	mu  sync.RWMutex
}

// ErrUserTokensNotFound est retourné par Load quand aucun fichier n'existe pour le xuid.
var ErrUserTokensNotFound = errors.New("multi_user_token_store: user tokens not found")

// NewMultiUserTokenStore crée un MultiUserTokenStore pointant vers le dossier donné.
// Le répertoire est créé avec perms 0700 si absent.
func NewMultiUserTokenStore(dir string) *MultiUserTokenStore {
	return &MultiUserTokenStore{dir: dir}
}

// Dir retourne le chemin du répertoire de stockage.
func (s *MultiUserTokenStore) Dir() string { return s.dir }

// xuidIsSafe vérifie qu'un XUID est utilisable comme nom de fichier sans risque.
// Les XUID Xbox Live sont des entiers décimaux ; on accepte aussi `-` (rare).
// Refuse tout caractère de path traversal (`.`, `/`, `\`).
func xuidIsSafe(xuid string) bool {
	if xuid == "" {
		return false
	}
	for _, r := range xuid {
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// pathFor retourne le chemin du fichier d'un XUID donné.
// Retourne "" si le XUID n'est pas safe (validation anti path-traversal).
func (s *MultiUserTokenStore) pathFor(xuid string) string {
	if !xuidIsSafe(xuid) {
		return ""
	}
	return filepath.Join(s.dir, xuid+".json")
}

// Upsert écrit ou remplace les tokens d'un utilisateur.
// Crée le répertoire si absent. Garantit CreatedAt à la première écriture
// et touche UpdatedAt à chaque écriture.
func (s *MultiUserTokenStore) Upsert(tokens *UserTokens) error {
	if tokens == nil {
		return fmt.Errorf("multi_user_token_store: tokens nil")
	}
	if !xuidIsSafe(tokens.XUID) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", tokens.XUID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.upsertLocked(tokens); err != nil {
		return err
	}

	slog.Debug("multi_user_token_store: tokens persistés",
		"xuid", tokens.XUID, "gamertag", tokens.Gamertag,
		"xsts_valid", tokens.IsXSTSValid(0))
	return nil
}

// Load lit les tokens d'un utilisateur par XUID.
// Retourne ErrUserTokensNotFound si aucun fichier n'existe pour ce xuid.
func (s *MultiUserTokenStore) Load(xuid string) (*UserTokens, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadLocked(xuid)
}

// loadLocked est la version interne sans verrouillage (callers doivent tenir le mutex).
func (s *MultiUserTokenStore) loadLocked(xuid string) (*UserTokens, error) {
	path := s.pathFor(xuid)
	if path == "" {
		return nil, fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrUserTokensNotFound
		}
		return nil, fmt.Errorf("multi_user_token_store: lecture %s: %w", path, err)
	}
	var t UserTokens
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("multi_user_token_store: décodage %s: %w", path, err)
	}
	return &t, nil
}

// LoadAll scanne le répertoire et retourne tous les tokens persistés, indexés par XUID.
// Ignore silencieusement les fichiers mal formés (log warning), retourne map vide si
// le répertoire n'existe pas.
func (s *MultiUserTokenStore) LoadAll() (map[string]*UserTokens, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*UserTokens{}, nil
		}
		return nil, fmt.Errorf("multi_user_token_store: scan %s: %w", s.dir, err)
	}

	result := make(map[string]*UserTokens)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		xuid := strings.TrimSuffix(name, ".json")
		if !xuidIsSafe(xuid) {
			slog.Warn("multi_user_token_store: fichier au nom invalide, ignoré", "name", name)
			continue
		}
		t, err := s.loadLocked(xuid)
		if err != nil {
			slog.Warn("multi_user_token_store: lecture échouée, fichier ignoré", "xuid", xuid, "err", err)
			continue
		}
		result[xuid] = t
	}
	return result, nil
}

// Remove supprime les tokens d'un utilisateur (révocation).
// No-op si le fichier n'existe pas.
func (s *MultiUserTokenStore) Remove(xuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.pathFor(xuid)
	if path == "" {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("multi_user_token_store: remove %s: %w", path, err)
	}
	return nil
}

// UpdateOAuthRefreshToken met à jour le champ OAuthRefreshToken pour un xuid en
// préservant tous les autres champs (XSTS, gamertag, CreatedAt).
// Appelée par le callback onRotated du Pool/Resolver à chaque rotation Microsoft.
//
// Crée l'entrée si elle n'existe pas (utile pour cmd/token-capture sur un
// joueur jamais authentifié auparavant). Dans ce cas, gamertag est laissé vide
// — l'appelant doit le compléter via Upsert si nécessaire.
//
// Read-modify-write atomique : prend le verrou, lit, modifie, écrit via Upsert.
// Idempotent : appeler avec le même rt ne corrompt pas.
func (s *MultiUserTokenStore) UpdateOAuthRefreshToken(xuid, refreshToken string) error {
	if !xuidIsSafe(xuid) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	if refreshToken == "" {
		return fmt.Errorf("multi_user_token_store: refresh_token vide pour xuid=%q", xuid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked(xuid)
	if err != nil && !errors.Is(err, ErrUserTokensNotFound) {
		return fmt.Errorf("multi_user_token_store: lecture pour update: %w", err)
	}
	if existing == nil {
		existing = &UserTokens{XUID: xuid}
	}
	existing.OAuthRefreshToken = refreshToken

	return s.upsertLocked(existing)
}

// MarkReauthRequired positionne le flag de ré-authentification requise pour un
// xuid (refresh_token mort). Read-modify-write atomique préservant les autres
// champs. Crée l'entrée si absente. Idempotent : ReauthDetectedAt conserve la
// première détection ; le gamertag n'est complété que s'il était vide.
//
// Retourne newlyMarked=true uniquement lors de la TRANSITION false→true (permet
// au caller de ne notifier qu'une fois, cf. ping Discord PR-B).
func (s *MultiUserTokenStore) MarkReauthRequired(xuid, gamertag string) (newlyMarked bool, err error) {
	if !xuidIsSafe(xuid) {
		return false, fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, lerr := s.loadLocked(xuid)
	if lerr != nil && !errors.Is(lerr, ErrUserTokensNotFound) {
		return false, fmt.Errorf("multi_user_token_store: lecture pour mark reauth: %w", lerr)
	}
	if existing == nil {
		existing = &UserTokens{XUID: xuid}
	}
	if existing.ReauthRequired {
		return false, nil // déjà marqué — préserve ReauthDetectedAt + évite un write inutile
	}
	existing.ReauthRequired = true
	existing.ReauthDetectedAt = time.Now().UTC()
	if existing.Gamertag == "" {
		existing.Gamertag = gamertag
	}
	if werr := s.upsertLocked(existing); werr != nil {
		return false, werr
	}
	return true, nil
}

// ClearReauthRequired remet le flag à false (refresh réussi ou ré-auth interactive).
// No-op si l'entrée est absente ou déjà non marquée.
func (s *MultiUserTokenStore) ClearReauthRequired(xuid string) error {
	if !xuidIsSafe(xuid) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked(xuid)
	if err != nil {
		if errors.Is(err, ErrUserTokensNotFound) {
			return nil
		}
		return fmt.Errorf("multi_user_token_store: lecture pour clear reauth: %w", err)
	}
	if !existing.ReauthRequired {
		return nil
	}
	existing.ReauthRequired = false
	existing.ReauthDetectedAt = time.Time{}
	return s.upsertLocked(existing)
}

// RecordAuthError persiste le dernier échec OAuth permanent d'un xuid
// (read-modify-write atomique, préserve les autres champs, crée l'entrée si
// absente). Le gamertag n'est complété que s'il était vide.
func (s *MultiUserTokenStore) RecordAuthError(xuid, gamertag, class, msg string) error {
	if !xuidIsSafe(xuid) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked(xuid)
	if err != nil && !errors.Is(err, ErrUserTokensNotFound) {
		return fmt.Errorf("multi_user_token_store: lecture pour record auth error: %w", err)
	}
	if existing == nil {
		existing = &UserTokens{XUID: xuid}
	}
	existing.LastAuthErrorClass = class
	existing.LastAuthError = msg
	existing.LastAuthErrorAt = time.Now().UTC()
	if existing.Gamertag == "" {
		existing.Gamertag = gamertag
	}
	return s.upsertLocked(existing)
}

// ClearAuthError efface le dernier échec OAuth mémorisé (refresh réussi).
// No-op si l'entrée est absente ou déjà vierge.
func (s *MultiUserTokenStore) ClearAuthError(xuid string) error {
	if !xuidIsSafe(xuid) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked(xuid)
	if err != nil {
		if errors.Is(err, ErrUserTokensNotFound) {
			return nil
		}
		return fmt.Errorf("multi_user_token_store: lecture pour clear auth error: %w", err)
	}
	if existing.LastAuthErrorClass == "" && existing.LastAuthError == "" {
		return nil
	}
	existing.LastAuthErrorClass = ""
	existing.LastAuthError = ""
	existing.LastAuthErrorAt = time.Time{}
	return s.upsertLocked(existing)
}

// IsReauthRequired retourne true si l'entrée du xuid existe et est marquée
// reauth_required. Lecture sans effet de bord ; false si absente/illisible.
func (s *MultiUserTokenStore) IsReauthRequired(xuid string) bool {
	t, err := s.Load(xuid)
	return err == nil && t != nil && t.ReauthRequired
}

// LoadByGamertag scanne le répertoire et retourne le premier UserTokens dont
// le Gamertag matche (case-insensitive). Utilisé par cmd/token-capture et
// cmd/token-import qui ne connaissent pas le xuid avant l'authentification.
//
// Retourne ErrUserTokensNotFound si aucun match ET que tous les fichiers ont pu
// être lus — c'est-à-dire « ce gamertag n'a jamais été authentifié ». Si aucun
// match mais qu'au moins un fichier était ILLISIBLE (I/O, JSON corrompu),
// retourne une erreur DISTINCTE enveloppant la cause : sans cette distinction, un
// store corrompu se présentait aux appelants comme une simple absence de token,
// et le remède affiché (« lancer token-capture ») ne répare pas un fichier
// corrompu (revue adversariale r2).
//
// Si plusieurs entrées matchent (cas pathologique : doublon gamertag pour deux
// xuid différents), retourne la première trouvée et log un warning.
func (s *MultiUserTokenStore) LoadByGamertag(gamertag string) (*UserTokens, error) {
	if gamertag == "" {
		return nil, fmt.Errorf("multi_user_token_store: gamertag vide")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrUserTokensNotFound
		}
		return nil, fmt.Errorf("multi_user_token_store: scan %s: %w", s.dir, err)
	}

	target := strings.ToLower(strings.TrimSpace(gamertag))
	var match *UserTokens
	matchCount := 0
	var unreadableErr error
	unreadable := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		xuid := strings.TrimSuffix(name, ".json")
		if !xuidIsSafe(xuid) {
			continue
		}
		t, err := s.loadLocked(xuid)
		if err != nil {
			// Fichier illisible : le scan CONTINUE (le gamertag cherché vit
			// peut-être dans un fichier sain), mais l'anomalie ne se perd plus
			// dans un `continue` nu. Warn et non Error : à ce niveau on ne sait
			// pas encore si la corruption bloque quoi que ce soit — c'est
			// l'appelant réellement privé d'auth qui escalade en Error (cf.
			// watcher_refresh.lookupRefreshToken). Même politique que LoadAll.
			// Récurrent tant que le fichier est corrompu : c'est voulu, l'anomalie
			// demande une action humaine.
			slog.Warn("multi_user_token_store: fichier de tokens illisible, ignoré du scan",
				"file", name, "dir", s.dir, "err", err)
			unreadableErr = err
			unreadable++
			continue
		}
		if strings.ToLower(t.Gamertag) == target {
			matchCount++
			if match == nil {
				match = t
			}
		}
	}

	if matchCount == 0 {
		if unreadable > 0 {
			// PAS ErrUserTokensNotFound : « introuvable » et « illisible » appellent
			// des remèdes opposés (authentifier vs réparer/supprimer le fichier).
			return nil, fmt.Errorf(
				"multi_user_token_store: aucun match pour %q et %d fichier(s) illisible(s) dans %s: %w",
				gamertag, unreadable, s.dir, unreadableErr)
		}
		return nil, ErrUserTokensNotFound
	}
	if matchCount > 1 {
		slog.Warn("multi_user_token_store: plusieurs entrées matchent le gamertag",
			"gamertag", gamertag, "count", matchCount, "selected_xuid", match.XUID,
			"hint", "doublons à nettoyer manuellement")
	}
	return match, nil
}

// upsertLocked est la version interne d'Upsert sans verrouillage (le caller
// tient déjà le mutex). Factorisé pour permettre les UpdateXxx atomiques.
func (s *MultiUserTokenStore) upsertLocked(tokens *UserTokens) error {
	if tokens == nil {
		return fmt.Errorf("multi_user_token_store: tokens nil")
	}
	if !xuidIsSafe(tokens.XUID) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", tokens.XUID)
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("multi_user_token_store: mkdir %s: %w", s.dir, err)
	}

	path := s.pathFor(tokens.XUID)
	if path == "" {
		return fmt.Errorf("multi_user_token_store: path résolu vide pour xuid=%q", tokens.XUID)
	}

	if existing, err := s.loadLocked(tokens.XUID); err == nil {
		if !existing.CreatedAt.IsZero() {
			tokens.CreatedAt = existing.CreatedAt
		} else if tokens.CreatedAt.IsZero() {
			tokens.CreatedAt = time.Now().UTC()
		}
		// Merge-preserve du credential COÛTEUX à ré-obtenir : un Upsert PARTIEL
		// (mirror, link/AddPlayer… qui ne poussent que XSTS/access) ne doit JAMAIS
		// effacer le refresh_token déjà persisté. Aucun appelant ne le vide
		// volontairement via Upsert (clear = Delete du fichier). Incident
		// 2026-06-13/14 : RT e1cb35ab frais écrasé à vide par le mirror PUIS par le
		// link → migration boot refill un RT mort (39829f7a) → AADSTS70000 en boucle.
		if tokens.OAuthRefreshToken == "" && existing.OAuthRefreshToken != "" {
			tokens.OAuthRefreshToken = existing.OAuthRefreshToken
		}
	} else if tokens.CreatedAt.IsZero() {
		tokens.CreatedAt = time.Now().UTC()
	}
	tokens.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("multi_user_token_store: marshal: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("multi_user_token_store: écriture tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("multi_user_token_store: rename atomique: %w", err)
	}

	return nil
}
