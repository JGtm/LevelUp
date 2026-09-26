// Package domain — relation_assists.go : les assistances ÉCHANGÉES entre le joueur et
// un autre joueur, sur leur historique commun.
//
// Deux surfaces lisent ce type : la page Relations (tableau, carte Binôme, carte Noyau
// dur) et l'historique des rencontres de la vue match. Même agrégat, même doctrine que
// assist_pairs.go (trois états de l'assistance, parts de dégâts non plafonnées), mais
// vu DEPUIS le joueur : ce que l'autre lui a donné, ce qu'il a donné à l'autre.
//
// ─── COUVERTURE ───────────────────────────────────────────────────────────────────────
//
// L'assistance n'est mesurée que sur les matchs dont le film a été décodé, et seulement
// sur les lignes `publishable AND assist_known` (une paire NOMME deux joueurs : lecture
// ligne à ligne). `MatchesMeasured` compte les matchs joués DANS LA MÊME ÉQUIPE portant
// au moins une telle ligne. Zéro match mesuré = pas d'objet `RelationAssists` du tout
// (champ nil) : l'écran affiche « — », jamais « 0 assistance ».
//
// ─── TRANCHES DE PART ─────────────────────────────────────────────────────────────────
//
// Chaque assistance porte la part de dégâts de l'assistant sur la victime
// (`assist_damage_pct`). Trois tranches, bornes publiées au front dans les infobulles :
//
//	Low   part < 25 %
//	Mid   25 % <= part <= 50 %
//	High  part > 50 %            (non plafonnée : les mesures vont jusqu'à 228)
//
// Une assistance sans part mesurée compte dans Total et dans AUCUNE tranche : la somme
// des tranches peut donc être inférieure au total, jamais supérieure.
package domain

// AssistTierLowMaxPct / AssistTierMidMaxPct : bornes des tranches de part (en %).
// Source unique côté Go ; le front répète les bornes dans ses libellés d'infobulle.
const (
	AssistTierLowMaxPct = 25
	AssistTierMidMaxPct = 50
)

// AssistTiers : un sens d'échange (reçues ou données), découpé par tranche de part.
type AssistTiers struct {
	Total int `json:"total"`
	Low   int `json:"low"`
	Mid   int `json:"mid"`
	High  int `json:"high"`
}

// RelationAssists : les assistances échangées avec un joueur, sur les matchs mesurés
// joués dans la même équipe.
type RelationAssists struct {
	// MatchesMeasured : matchs en même équipe dont l'assistance est mesurée. > 0 toujours
	// (un objet n'est publié que s'il y a au moins un match mesuré).
	MatchesMeasured int `json:"matches_measured"`
	// MyFrags / PartnerFrags : frags du joueur / de l'autre sur ces mêmes matchs mesurés.
	// Dénominateurs des parts « % de tes frags assistés par lui » et inverse.
	MyFrags      int `json:"my_frags"`
	PartnerFrags int `json:"partner_frags"`
	// Received : assistances de l'autre sur les frags du joueur.
	Received AssistTiers `json:"received"`
	// Given : assistances du joueur sur les frags de l'autre.
	Given AssistTiers `json:"given"`
}

// MatchAssistedFrags : sur UN match, la part des frags du joueur qui ont été assistés
// par un coéquipier — le sens « reçues » de RelationAssists, ramené à un seul match
// (tuile de match de l'Accueil). Mêmes lignes (`publishable AND assist_known`), mêmes
// tranches, mêmes bornes.
//
// FragsMeasured : frags du joueur portés par ces lignes sur le match (DÉNOMINATEUR de la
// part « N / M frags assistés »). Received : ceux dont `assist_xuid` est renseigné, par
// tranche de part ; une assistance sans part mesurée compte dans Total et dans aucune
// tranche. Zéro frag mesuré = pas d'objet (nil), même quand le joueur a 0 frag au match :
// « on ne sait pas » n'est pas « 0 ».
type MatchAssistedFrags struct {
	FragsMeasured int         `json:"frags_measured"`
	Received      AssistTiers `json:"received"`
}
