package groupstore

import (
	"fmt"

	"levelup/go-api/internal/domain"
)

// MigrateDefault crée un groupe par défaut (typiquement "Mon foyer") depuis l'ancienne
// liste globale `friend_gamertags`, pour préserver la continuité d'accès lors du passage
// au modèle multi-groupes. Idempotent : no-op si des groupes existent déjà (retourne
// created=false). Le caller résout les gamertags → xuid (profils connus) et passe les
// membres résolus via `extraMembers` (le propriétaire est ajouté automatiquement).
//
// Pattern : cf. auth.MigrateLegacyTokens — la résolution legacy est faite par le caller,
// le store ne dépend ni des settings ni de db_profiles.json.
func (s *GroupStore) MigrateDefault(name, ownerXUID, ownerGamertag string, extraMembers []domain.GroupMember) (bool, error) {
	if ownerXUID == "" {
		return false, ErrMissingOwnerXUID
	}

	// Garde idempotente : si au moins un groupe existe, la migration a déjà eu lieu.
	existing, err := s.List()
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		return false, nil
	}

	g, err := s.Create(name, ownerXUID, ownerGamertag)
	if err != nil {
		return false, fmt.Errorf("création groupe par défaut : %w", err)
	}

	for _, m := range extraMembers {
		if m.XUID == "" || m.XUID == ownerXUID {
			continue // owner déjà ajouté par Create ; xuid vide ignoré
		}
		if err := s.AddMember(g.ID, m.XUID, m.Gamertag); err != nil {
			return true, fmt.Errorf("ajout membre %q au groupe par défaut : %w", m.Gamertag, err)
		}
	}
	return true, nil
}
