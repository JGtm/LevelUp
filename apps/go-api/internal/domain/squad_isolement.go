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
//   - Y : le delai avant que le tueur tombe (`DelaiMs`). LA LECTURE SANS BORNE
//     (coordination.Ripostes) N'EST QU'UNE MATIERE DE DESSIN, JAMAIS UNE RIPOSTE
//     (decision utilisateur du 2026-09-22) : la REGLE DES 5 s vaut partout, et le point
//     porte son etat, pas le client. Une mort jamais suivie n'a pas de delai : bande haute.
//
// TROIS ETATS EXCLUSIFS PAR POINT, et c'est le serveur qui tranche :
//
//	Vengee                  le tueur est tombe sous un coequipier DANS la fenetre
//	                        (DelaiMs <= SquadNuageIsolement.FenetreMs, borne comprise) —
//	                        la MEME riposte que la carte « Riposte » et le taux d'echange ;
//	HorsFenetre             le tueur est tombe APRES la fenetre mais AVANT le plafond
//	                        (DelaiMs <= SquadNuageIsolement.PlafondMs, borne comprise) : le
//	                        delai est publie (le nuage continue de MONTRER ces morts) mais
//	                        ce n'est PAS une riposte — Vengee reste faux ;
//	ni l'un ni l'autre      aucune riposte connue, OU une chute du tueur au-dela du plafond
//	                        (decision utilisateur du 2026-09-22 : passe 60 s le tueur est
//	                        mort de sa propre vie, le point ne dit plus rien) : pas de
//	                        delai, bande « jamais ripostee ».
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

	// Vengee : un coequipier a abattu le tueur DANS LA FENETRE DE RIPOSTE
	// (SquadNuageIsolement.FenetreMs, borne comprise). C'est la MEME definition que le
	// taux de la carte « Riposte » et que le bloc Riposte du match : jamais une riposte
	// sans borne.
	Vengee bool `json:"vengee"`
	// HorsFenetre : un coequipier a abattu le tueur APRES la fenetre mais AVANT le plafond
	// (SquadNuageIsolement.PlafondMs, borne comprise). Le delai est publie pour que le
	// nuage montre encore ces morts, et `Vengee` reste FAUX : ce n'est pas une riposte.
	// Exclusif de `Vengee`. Au-dela du plafond, ce champ retombe a faux : la chute du
	// tueur n'a plus de lien avec la mort initiale.
	HorsFenetre bool `json:"hors_fenetre"`
	// DelaiMs : le delai avant que le tueur tombe. Present quand `Vengee` OU `HorsFenetre`
	// est vrai ; absent quand aucune riposte n'est connue, et absent AUSSI quand la chute
	// du tueur depasse le plafond — publier ce delai laisserait croire a un lien de cause
	// a effet que le jeu ne porte plus.
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
	// MedianeDelaiMs : mediane du delai des SEULES morts VENGEES — celles dont le tueur
	// est tombe DANS la fenetre. Les morts `HorsFenetre` en sont exclues (leur delai
	// tirerait la mediane vers une population que le taux de riposte ne compte pas).
	// Absente si aucune mort du joueur n'est vengee.
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

	// FenetreMs republie coordination.FenetreEchangeMs : la fenetre de riposte, bornes
	// comprises. Publiee pour que le client TRACE le repere de fenetre sans coder 5 000 en
	// dur — une regle du jeu ne se recopie pas de l'autre cote du contrat.
	FenetreMs int64 `json:"fenetre_ms"`

	// PlafondMs republie coordination.PlafondRiposteTardiveMs : le delai au-dela duquel
	// aucun point ne porte plus d'etat de riposte (ni `Vengee`, ni `HorsFenetre`, ni
	// `DelaiMs`). C'est AUSSI le haut de la zone mesuree de l'axe des delais : le client
	// ne code jamais 60 000 en dur, il lit cette valeur.
	PlafondMs int64 `json:"plafond_ms"`

	// PlancherEchantillonFaible republie coordination.SeuilEchantillonFaible : le seuil
	// sous lequel PartIsolee.EchantillonFaible / Couverture.EchantillonFaible passe a
	// vrai.
	PlancherEchantillonFaible int `json:"plancher_echantillon_faible"`
}
