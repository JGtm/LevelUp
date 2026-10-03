// Package domain — elevation_cloud.go : LE CONTRAT DU NUAGE « DÉNIVELÉ » (décision D25, T5).
//
// Un point par frag mesuré sur la fenêtre filtrée, des deux côtés (mes frags, mes morts),
// plus les quartiles de chaque côté et la couverture. Il REMPLACE la carte « Dénivelé » en
// barres empilées par arme : la question posée n'est pas « avec quelle arme » mais « d'où ».
//
// LE SIGNE VIENT DE GO, LE FRONT NE LE RECALCULE PAS. `DeltaZM` est déjà au point de vue du
// joueur (positif = j'étais au-dessus, pour un frag comme pour une mort) : la convention est
// posée par `analysis.signedElevation` et nulle part ailleurs.
package domain

// ElevationCloudBlock est le bloc publié — nil (champ omis) pour un titre sans positions par
// kill, un scope non décodé ou une lecture en échec ; jamais un bloc vide.
type ElevationCloudBlock struct {
	// Kills / Deaths : un point par frag mesuré, côté « je tue » et côté « je meurs ».
	// AUCUN PLAFOND, AUCUN ÉCHANTILLONNAGE (T5) : un échantillon donnerait deux dessins
	// différents pour la même fenêtre. Toujours sérialisés (`[]`, jamais `null`).
	Kills  []ElevationPoint `json:"kills"`
	Deaths []ElevationPoint `json:"deaths"`

	// KillsSummary / DeathsSummary : les quartiles du côté — le rectangle de masse
	// (p25→p75 en distance × p25→p75 en dénivelé) et le point médian du nuage.
	KillsSummary  ElevationSideSummary `json:"kills_summary"`
	DeathsSummary ElevationSideSummary `json:"deaths_summary"`

	// WeaponLabels : les noms d'affichage des clés d'arme portées par les points, FR et EN.
	//
	// UN DICTIONNAIRE ET PAS UN CHAMP PAR POINT : la fenêtre porte des milliers de points
	// pour quelques dizaines d'armes, et recopier deux libellés sur chacun multiplierait la
	// charge utile sans rien ajouter. Le front lit `weapon_key` dans cette table — il n'a
	// donc jamais à afficher une clé brute. Une clé absente (metadata muette) n'a pas
	// d'entrée : le front retombe alors sur la clé, jamais sur un nom inventé.
	WeaponLabels map[string]ElevationWeaponLabel `json:"weapon_labels,omitempty"`

	// MeasuredKills / TotalKills, MeasuredDeaths / TotalDeaths : la couverture, MÊME
	// source et même doctrine que `SynthesisWeaponRange` (totaux du scope canonique,
	// mesurés de la table de positions). L'écart se publie, il ne se cache pas.
	MeasuredKills  int `json:"measured_kills"`
	TotalKills     int `json:"total_kills"`
	MeasuredDeaths int `json:"measured_deaths"`
	TotalDeaths    int `json:"total_deaths"`
}

// ElevationPoint est UN frag mesuré du nuage.
type ElevationPoint struct {
	// DistanceM : abscisse, en mètres.
	DistanceM float64 `json:"distance_m"`
	// DeltaZM : ordonnée, en mètres, SIGNÉE du point de vue du joueur (cf. en-tête).
	DeltaZM float64 `json:"delta_z_m"`
	// MatchID / TimeMS : l'identité du frag — l'infobulle nomme le match, et l'instant
	// laisse la porte ouverte à un lien vers le rejeu.
	MatchID string `json:"match_id"`
	TimeMS  int64  `json:"time_ms"`
	// Weapon : la clé de REGISTRE de l'arme, à traduire par `WeaponLabels`.
	Weapon string `json:"weapon"`
}

// ElevationSideSummary : les quartiles d'un côté et son effectif.
//
// LES SIX QUANTILES SONT DES POINTEURS, PAS DES ZÉROS. Un côté sans mesure n'a pas de
// quartile ; publier `0` y ferait dessiner un halo collé à l'origine et une médiane à
// « 0,0 m », qui se lirait « à niveau, au contact » au lieu de « on ne sait pas ». `N` reste
// un entier : zéro mesure est un fait, pas une absence.
type ElevationSideSummary struct {
	DistanceP25 *float64 `json:"distance_p25,omitempty"`
	DistanceP50 *float64 `json:"distance_p50,omitempty"`
	DistanceP75 *float64 `json:"distance_p75,omitempty"`
	DeltaZP25   *float64 `json:"delta_z_p25,omitempty"`
	DeltaZP50   *float64 `json:"delta_z_p50,omitempty"`
	DeltaZP75   *float64 `json:"delta_z_p75,omitempty"`
	N           int      `json:"n"`
}

// ElevationWeaponLabel : le nom d'affichage d'une clé d'arme, dans les deux langues.
//
// DÉFINI ICI plutôt qu'emprunté à `port.WeaponLabel` : le contrat public ne dépend pas d'un
// port (le sens des flèches du dépôt), et le front sérialise déjà cette paire ailleurs sous
// les mêmes noms de champs.
type ElevationWeaponLabel struct {
	Label   string `json:"label,omitempty"`
	LabelEN string `json:"label_en,omitempty"`
}
