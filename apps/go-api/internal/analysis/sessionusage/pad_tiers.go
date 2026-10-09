package sessionusage

// pad_tiers.go — LA LIGNE DES PRISES DE SOCLE PAR NIVEAU D'ARME (`match_pad_pickups_by_tier_latest`),
// lue par l'Emprise pour séparer armes spéciales et armes de râtelier (analysis/squademprise).
//
// Le niveau vient d'une AUTRE table que le résumé d'usage, alimentée par une passe distincte qui a
// besoin de la carte du match : un match absent de cette table n'est pas un match sans prise, c'est
// un match non classé.

// PadTierRow — une ligne (match, joueur, niveau, arme) de
// `match_pad_pickups_by_tier_latest`.
type PadTierRow struct {
	MatchID string
	XUID    string
	// Tier : le vocabulaire ECRIT EN BASE (`persist.PadTier*`). Ce paquet ne le traduit pas :
	// il le transporte jusqu'au contrat, et c'est l'i18n du web qui le nomme.
	Tier string
	// WeaponFamily : la clé normalisée de la famille d'arme. VIDE sur une ligne
	// `aucune_prise` — le zéro mesuré d'un joueur qui n'a pris aucun socle.
	WeaponFamily string
	Pickups      int
	// PadsConfirmed / PadsTotal / RandomStarts : valeurs de MATCH, identiques sur toutes les
	// lignes du match (cf. persist.PadTiersBatch).
	PadsConfirmed int
	PadsTotal     int
	RandomStarts  bool
}
