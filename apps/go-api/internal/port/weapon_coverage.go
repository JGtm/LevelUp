package port

// weapon_coverage.go — les lectures de la page d'administration « couverture de résolution d'arme » : quelle
// part des frags se résout à une arme nommée. Deux unités comptables, selon le titre : l'identifiant d'arme
// (journal d'événements) ou la source de dégât du film.

import "context"

// WeaponKillCount : les frags d'un identifiant d'arme (hors sentinelles 0, 1 et 2).
type WeaponKillCount struct {
	WeaponID int64
	Kills    int
}

// KillSourceCount : les frags d'une source de dégât du film.
type KillSourceCount struct {
	SourceTag uint32
	Kills     int
}

// WeaponCoverageRepo lit la couverture d'arme dans la base partagée du titre et dans metadata.
type WeaponCoverageRepo interface {
	// KillsByWeaponID rend les frags par identifiant d'arme distinct (hors sentinelles 0, 1 et 2).
	KillsByWeaponID(ctx context.Context) ([]WeaponKillCount, error)
	// KillsBySourceTag rend les frags par source de dégât mesurée, lus sur les résumés par match.
	KillsBySourceTag(ctx context.Context) ([]KillSourceCount, error)
	// HasMetadata dit si metadata est ouverte ; sinon rien ne se résout par le registre ni par les libellés.
	HasMetadata() bool
	// ClassifyWeaponIDs marque chaque identifiant comme présent au registre d'armes (inRegistry) et/ou dans
	// les libellés d'arme (inLabel) pour le titre ; un registre absent (ancien schéma) se replie sur les
	// libellés seuls.
	ClassifyWeaponIDs(ctx context.Context, titleSlug string, ids []int64) (inRegistry, inLabel map[int64]bool)
	// RegistryWeaponKeys rend les clés d'arme du registre du titre.
	RegistryWeaponKeys(ctx context.Context, titleSlug string) (map[string]bool, error)
}
