package domain

// squad_isolement.go — LE NUAGE « POURQUOI LA VENGEANCE NE VIENT PAS » DE LA PAGE ESCOUADE
// (plan tactique, phase 7, item 7.7 ; contrat refondu le 2026-09-19, plan
// PLAN_AJUSTEMENTS_PRE_V75, decision 4).
//
// UN PETIT POINT PAR MORT, UN GROS POINT PAR JOUEUR. L'ancien contrat (un point par joueur
// et par session) ne rendait qu'un point par joueur sur une soiree — l'usage nominal de la
// page. Chaque mort du roster sur le perimetre porte donc ses deux coordonnees :
//
//   - X : la distance au coequipier VISIBLE le plus proche, RAPPORTEE A LA PORTEE DU RADAR
//     DU MATCH (`DistanceRatio` = PlusProcheM / rayon ; 1,0 = « a la portee du radar »).
//     Absente (nil) quand aucun coequipier n'est visible : le point va dans la bande
//     « hors de vue » de l'ecran, jamais a une distance inventee.
//   - Y : le delai avant vengeance (`DelaiMs`, riposte SANS borne de fenetre —
//     coordination.Ripostes). Une mort jamais vengee n'a pas de delai : bande haute.
//
// Le GROS point d'un joueur (SquadIsolementRepere) est la mediane X x la mediane Y de ses
// morts, sa taille dit combien de morts, et son infobulle porte encore la part isolee et le
// taux d'echange (la MEME mesure que la section « echange », restreinte a ce joueur).

// SquadIsolementMort est UN petit point du nuage : une mort d'un joueur du roster.
type SquadIsolementMort struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
	MatchID  string `json:"match_id"`
	// TimeMs : l'instant de la mort sur l'horloge du match — la cle de jointure entre le
	// contexte de mort (isolement) et la riposte, avec (MatchID, XUID).
	TimeMs int64 `json:"time_ms"`

	// DistanceRatio : distance au coequipier visible le plus proche, rapportee a la portee
	// du radar du match (1,0 = a la portee). Absent = aucun coequipier visible.
	DistanceRatio *float64 `json:"distance_ratio,omitempty"`
	// HorsDeVue : aucun coequipier visible a l'instant de la mort (DistanceRatio absent).
	HorsDeVue bool `json:"hors_de_vue"`

	// Vengee : un coequipier a abattu le tueur apres cette mort (sans borne de temps).
	Vengee bool `json:"vengee"`
	// DelaiMs : le delai avant cette vengeance. Absent quand la mort n'est pas vengee.
	DelaiMs *int64 `json:"delai_ms,omitempty"`
}

// SquadIsolementRepere est le GROS point d'un joueur : ses medianes, son volume, et les
// deux taux de son infobulle.
type SquadIsolementRepere struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`

	// NbMorts : le nombre de morts du joueur publiees dans `Morts` — la TAILLE du point.
	NbMorts int `json:"nb_morts"`
	// MedianeDistanceRatio : mediane de DistanceRatio sur les morts qui en portent un.
	// Absente si aucune mort du joueur n'a de coequipier visible.
	MedianeDistanceRatio *float64 `json:"mediane_distance_ratio,omitempty"`
	// MedianeDelaiMs : mediane du delai des morts VENGEES. Absente si aucune ne l'est.
	MedianeDelaiMs *int64 `json:"mediane_delai_ms,omitempty"`

	// PartIsolee : le taux d'isolement du joueur sur le perimetre (analysis/coordination
	// .Isolement), sous sa forme canonique.
	PartIsolee Couverture `json:"part_isolee"`
	// Couverture : le taux d'ECHANGE du joueur sur le perimetre (morts vengees dans la
	// fenetre sur morts vengeables) — la meme mesure que la section « echange »,
	// restreinte a ce seul joueur.
	Couverture Couverture `json:"couverture"`
}

// SquadNuageIsolement est la section « nuage » du bloc Echange de la page Escouade.
//
// Absente (nil) quand le titre n'a pas de table de portee de radar cablee, quand le journal
// d'isolement est en echec, ou quand aucune mort du roster n'est localisee sur un match a
// rayon connu : une OMISSION, jamais un nuage vide affiche comme une mesure a zero.
type SquadNuageIsolement struct {
	Morts   []SquadIsolementMort   `json:"morts"`
	Reperes []SquadIsolementRepere `json:"reperes"`

	// PlancherEchantillonFaible republie coordination.SeuilEchantillonFaible : le seuil
	// sous lequel PartIsolee.EchantillonFaible / Couverture.EchantillonFaible passe a
	// vrai.
	PlancherEchantillonFaible int `json:"plancher_echantillon_faible"`
}
