package domain

// AdminWeaponCoverage — couverture de la résolution d'arme pour un titre
// (monitoring admin). Pour les weapon_id distincts vus dans v_weapon_kills (hors
// sentinels 0/1/2), combien sont résolus par le REGISTRE (weapon_key), par
// weapon_labels SEUL, ou NON résolus (affichés en décimal). Détecte en continu
// les armes non mappées (nouvelles variantes / events).
type AdminWeaponCoverage struct {
	TitleSlug        string                    `json:"title_slug"`
	GeneratedAt      string                    `json:"generated_at"`
	DistinctWeapons  int                       `json:"distinct_weapons"`
	ResolvedRegistry int                       `json:"resolved_registry"`
	ResolvedLabel    int                       `json:"resolved_label"`
	Unresolved       int                       `json:"unresolved"`
	CoveragePercent  float64                   `json:"coverage_percent"` // (registry+label)/distinct
	RegistryPercent  float64                   `json:"registry_percent"` // registry/distinct
	TopUnresolved    []AdminWeaponCoverageItem `json:"top_unresolved"`   // non résolus, kills desc
}

// AdminWeaponCoverageItem — un weapon_id non résolu + son volume de kills.
// WeaponID est une string (et non un int64) : les ids filmshell HINF dépassent
// 2^53 et perdraient en précision en tant que number JSON côté JS. Champ string
// réel (pas un tag `,string`) pour que le schéma OpenAPI auto-dérivé soit fidèle.
type AdminWeaponCoverageItem struct {
	WeaponID string `json:"weapon_id"`
	Kills    int    `json:"kills"`
}
