// Package auth — pool_sharing.go : consentement d'un joueur à la mise en commun de son jeton.
//
// Le pool de jetons (internal/platform/auth/pool) fait tourner les requêtes de synchro
// sur TOUS les jetons partagés pour répartir la charge sous la limite de débit de
// Microsoft. Un jeton n'y entre que si son propriétaire l'a décidé : la décision vit
// dans le fichier de jetons du joueur (UserTokens.PoolSharing), à côté du credential
// qu'elle gouverne. Aucune décision enregistrée vaut refus.
package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// Bases d'une décision, conservées pour pouvoir la justifier.
const (
	// PoolSharingBasisUserChoice : le joueur a choisi lui-même (écran de connexion ou
	// réglage).
	PoolSharingBasisUserChoice = "user_choice"
	// PoolSharingBasisGrandfathered : accord explicite recueilli hors application avant
	// l'existence du réglage, enregistré par l'opérateur (levelup pool-sharing
	// grandfather).
	PoolSharingBasisGrandfathered = "grandfathered_explicit"
)

// PoolSharing est la décision d'un joueur sur la mise en commun de son jeton.
type PoolSharing struct {
	Granted   bool      `json:"granted"`
	DecidedAt time.Time `json:"decided_at"`
	Basis     string    `json:"basis"`
}

// SharesWithPool indique si le jeton peut entrer dans le round-robin public :
// seulement sur un consentement enregistré et non retiré.
func (u *UserTokens) SharesWithPool() bool {
	return u != nil && u.PoolSharing != nil && u.PoolSharing.Granted
}

// SetPoolSharing enregistre la décision d'un joueur (accord ou retrait). L'entrée doit
// exister : une décision ne crée jamais un fichier de jetons. Read-modify-write
// atomique ; les autres champs sont préservés.
func (s *MultiUserTokenStore) SetPoolSharing(xuid string, granted bool, basis string) error {
	if !xuidIsSafe(xuid) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	if basis == "" {
		return fmt.Errorf("multi_user_token_store: base de décision vide pour xuid=%q", xuid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked(xuid)
	if err != nil {
		return err
	}
	existing.PoolSharing = &PoolSharing{Granted: granted, DecidedAt: time.Now().UTC(), Basis: basis}
	return s.upsertLocked(existing)
}

// GrandfatherPoolSharing enregistre l'accord de tous les joueurs qui ont un jeton et
// aucune décision. Une décision existante (accord comme refus) n'est jamais écrasée.
// Idempotent. Retourne le nombre de joueurs marqués.
func (s *MultiUserTokenStore) GrandfatherPoolSharing(basis string) (int, error) {
	all, err := s.LoadAll()
	if err != nil {
		return 0, err
	}
	marked := 0
	for xuid := range all {
		done, err := s.grandfatherOne(xuid, basis)
		if err != nil {
			return marked, fmt.Errorf("multi_user_token_store: grandfather xuid=%s: %w", xuid, err)
		}
		if done {
			slog.Info("multi_user_token_store: accord pool enregistré", "xuid", xuid, "basis", basis)
			marked++
		}
	}
	return marked, nil
}

// grandfatherOne relit l'entrée SOUS le verrou avant d'écrire : un joueur qui décide
// pendant que l'opérateur lance la commande garde sa décision.
func (s *MultiUserTokenStore) grandfatherOne(xuid, basis string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked(xuid)
	if err != nil {
		if errors.Is(err, ErrUserTokensNotFound) {
			return false, nil
		}
		return false, err
	}
	if existing.OAuthRefreshToken == "" || existing.PoolSharing != nil {
		return false, nil
	}
	existing.PoolSharing = &PoolSharing{Granted: true, DecidedAt: time.Now().UTC(), Basis: basis}
	return true, s.upsertLocked(existing)
}

// PoolSharingState résout l'état de la décision d'un xuid. « unavailable » quand le joueur
// n'a pas de jeton (entrée absente, illisible ou sans refresh token).
func (s *MultiUserTokenStore) PoolSharingState(xuid string) string {
	t, err := s.Load(xuid)
	if err != nil && !errors.Is(err, ErrUserTokensNotFound) {
		slog.Warn("multi_user_token_store: état de mise en commun illisible", "xuid", xuid, "err", err)
	}
	if err != nil || t == nil || t.OAuthRefreshToken == "" {
		return domain.PoolSharingUnavailable
	}
	switch {
	case t.PoolSharing == nil:
		return domain.PoolSharingUndecided
	case t.PoolSharing.Granted:
		return domain.PoolSharingGranted
	default:
		return domain.PoolSharingDeclined
	}
}
