// Package domain — relation_assists.go : les assistances ÉCHANGÉES entre le joueur et
// un autre joueur, sur leur historique commun, et les frags assistés d'un seul match.
//
// Deux surfaces lisent RelationAssists : la page Relations (tableau, carte Binôme, carte
// Noyau dur, encart de l'Explorer) et l'historique des rencontres de la vue match. La
// tuile de match de l'Accueil lit MatchAssistedFrags, le sens « reçues » ramené à un match.
//
// ─── LA RÈGLE DES BASES (une seule, pour toutes les surfaces d'assistance) ────────────
//
//   - PORTÉE : les matchs dont le film PORTE l'assistance — au moins une ligne
//     `publishable AND assist_known` dans le match. Un match sans film, ou dont le film ne
//     lit aucune assistance, n'apporte ni numérateur ni base : il est hors de la mesure,
//     et rien ne l'annonce à l'écran.
//   - BASE : les frags OFFICIELS du joueur (feuille de match, `match_participants.kills`)
//     sur ces matchs. Frags sur des bots compris.
//   - NUMÉRATEUR : les frags dont l'assistant est NOMMÉ par le film, bot compris. Un frag
//     de la base dont l'assistance n'est pas lue n'est simplement pas compté au numérateur :
//     aucun compte « sans information » n'est publié.
//
// Un bot ne devient jamais une LIGNE de relation (ce n'est pas un coéquipier nommé), mais
// ses victimes et ses frags restent dans les bases.
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

// RelationAssists : les assistances échangées avec un joueur, sur les matchs joués dans la
// même équipe dont le film porte l'assistance (règle des bases, en-tête). Absent (nil)
// quand aucun de ces matchs n'existe : l'écran affiche « — », jamais « 0 assistance ».
type RelationAssists struct {
	// MyFrags / PartnerFrags : frags OFFICIELS du joueur / de l'autre sur ces matchs.
	// Bases des parts « frags du joueur assistés par lui » et inverse.
	MyFrags      int `json:"my_frags"`
	PartnerFrags int `json:"partner_frags"`
	// Received : assistances de l'autre sur les frags du joueur.
	Received AssistTiers `json:"received"`
	// Given : assistances du joueur sur les frags de l'autre.
	Given AssistTiers `json:"given"`
}

// MatchAssistedFrags : sur UN match, les frags du joueur qu'un coéquipier a assistés —
// le sens « reçues » de RelationAssists ramené à un seul match (tuile de match de
// l'Accueil). Même règle des bases, mêmes tranches, mêmes bornes.
//
// Received : frags du joueur dont l'assistant est nommé, par tranche. FragsOfficial : la
// base, posée par WithOfficialFrags. Nil (pas d'objet) quand le film du match ne porte pas
// l'assistance ou que le joueur n'y a aucun frag lu.
type MatchAssistedFrags struct {
	// FragsFilm : frags du joueur lus par le film sur ce match. Plancher de la base
	// (WithOfficialFrags), pas publié.
	FragsFilm     int         `json:"-"`
	FragsOfficial int         `json:"frags_official"`
	Received      AssistTiers `json:"received"`
}

// WithOfficialFrags pose la base officielle (`kills`, frags du match tels que la tuile les
// affiche). La base ne descend jamais sous FragsFilm : un compte officiel absent (nil) ou
// inférieur au film ne rapporterait pas les assistés à une base plus petite que les frags
// qui les portent.
func (a MatchAssistedFrags) WithOfficialFrags(kills *int) MatchAssistedFrags {
	a.FragsOfficial = BaseFragsOfficiels(kills, a.FragsFilm)
	return a
}

// BaseFragsOfficiels : la base d'une part de frags assistés sur un match — les frags
// officiels, jamais sous les frags lus par le film (`lusAuFilm`). Partagée par la tuile de
// match et le bloc Coordination (Sessions, Séries temporelles).
func BaseFragsOfficiels(officiels *int, lusAuFilm int) int {
	if officiels != nil && *officiels > lusAuFilm {
		return *officiels
	}
	return lusAuFilm
}
