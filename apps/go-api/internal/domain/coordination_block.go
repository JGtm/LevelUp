package domain

// coordination_block.go — LE BLOC « COORDINATION » D'UN SCOPE DE MATCHS : l'appui REÇU.
//
//	APPUI     « combien de mes frags m'ont été préparés » et « sur les appuis distribués
//	          dans mon camp, combien me sont revenus ».
//
//	Sessions     le bloc + une case par match (PerMatch) ;
//	Timeseries   le bloc + un point par SOIRÉE (Sessions) ;
//
// ─── POURQUOI LE MÊME TYPE POUR SESSIONS ET TIMESERIES ────────────────────────────────
//
// Les deux pages posent la MÊME question à deux mailles. Deux types auraient donné deux
// définitions de « appuis de mon camp » libres de diverger au premier ajustement. La maille se
// lit dans le champ rempli : `PerMatch` (session) ou `Sessions` (soirées), jamais les deux.
//
// ─── TOUT TAUX VOYAGE EN Couverture ───────────────────────────────────────────────────
//
// Aucun quotient nu ici : chaque part est une `Couverture` (taux 0..1, compte brut,
// quantité par match, dénominateur, drapeau d'échantillon faible). C'est la règle de forme
// de analysis/coordination, et elle traverse le contrat HTTP telle quelle.
//
// ─── UN MATCH SANS ASSISTANCE LUE EST ABSENT, JAMAIS À ZÉRO ───────────────────────────
//
// « Mesuré » veut dire : le journal des morts du match est lisible (au moins une ligne
// publiable dans `match_kill_events_latest`) ET son film porte l'assistance (au moins une
// ligne d'appui, `publishable AND assist_known`). Un match non mesuré n'a pas un taux nul,
// il n'a pas de taux — il ne fournit ni numérateur ni dénominateur.
//
// ─── RÈGLE DES BASES (domain/relation_assists.go) ─────────────────────────────────────
//
// « Frags appuyés » se rapporte aux frags OFFICIELS du joueur (feuille de match) sur les
// matchs mesurés, frags sur des bots compris ; un frag dont l'assistance n'est pas lue
// reste dans la base sans entrer au numérateur. Les appuis impliquant des BOTS (assistant,
// tueur ou victime) comptent comme les autres.
// CoordinationUnavailableReason — la raison MACHINE d'un bloc indisponible (même doctrine
// que SessionUsageUnavailableReason : l'écran traduit, le contrat ne rédige pas).
const (
	// CoordinationUnsupported : le titre ne nomme pas le tueur de chaque mort (capability
	// du journal des morts fermée), ou aucun lecteur n'est câblé.
	CoordinationUnsupported = "unsupported"
	// CoordinationLoadFailed : la lecture a échoué — loggée, puis dégradée. Le reste de
	// la page reste servi.
	CoordinationLoadFailed = "load_failed"
	// CoordinationNoMeasuredMatch : aucun match du scope ne porte de journal des morts
	// lisible. État NOMINAL d'un joueur dont les films ont expiré, pas une panne.
	CoordinationNoMeasuredMatch = "no_measured_match"
)

// CoordinationMatch — un match du scope, tel que le bloc a besoin de le connaître.
type CoordinationMatch struct {
	MatchID string
	// Mesure : le journal des morts de ce match est lisible (TacticalMatch.Mesure — la
	// MÊME définition, rendue par le MÊME lecteur).
	Mesure bool
	// TeamSize : effectif de MON camp sur ce match. Nil = camp inconnu (FFA) : le match ne
	// porte alors aucune parité, et jamais un 1 inventé (réserve R1).
	TeamSize *int
}

// CoordinationAppuiRow — les morts MESURÉES pour l'assistance d'un match, groupées par
// (assistant, tueur crédité).
//
// L'assistant est NOMMÉ quand le film écrit son xuid ou, à défaut, son gamertag (bot,
// joueur non résolu) : [CoordinationAppuiRow.Assiste]. Ni l'un ni l'autre est un ÉTAT
// MESURÉ, pas une absence de ligne : « publiable et assist_known, personne n'a assisté ».
//
// Un xuid VIDE (assistant, tueur) désigne un acteur sans xuid — un bot le plus souvent. Il
// n'a pas de camp dans la table des équipes : son camp se déduit de l'autre acteur de la
// ligne (une assistance est toujours portée par un coéquipier du tueur) ou, quand les deux
// sont sans xuid, de la victime (`VictimXUID`, renseignée pour ce seul cas).
type CoordinationAppuiRow struct {
	MatchID        string
	AssistXUID     string
	AssistGamertag string
	KillerXUID     string
	VictimXUID     string
	Nombre         int
}

// Assiste dit que le film nomme un assistant sur ces morts.
func (a CoordinationAppuiRow) Assiste() bool {
	return a.AssistXUID != "" || a.AssistGamertag != ""
}

// CoordinationEntree — tout ce que le calcul du bloc consomme. Une struct plutôt que six
// paramètres adjacents (seuil CLAUDE.md n 5), et un type de DOMAINE parce que
// analysis/coordination ne déclare aucun type exporté.
type CoordinationEntree struct {
	// MoiXUID : le joueur consulté. Vide = rien à mesurer (le bloc n'a pas de sujet).
	MoiXUID string
	Matchs  []CoordinationMatch
	Equipes EquipesParMatch
	Appuis  []CoordinationAppuiRow
	// FragsOfficiels : frags du joueur par match, tels que la feuille de match les compte.
	// Base de « frags appuyés » ; un match absent retombe sur les frags lus par le film.
	FragsOfficiels map[string]int
}

// CoordinationAppui — les deux grandeurs d'appui REÇU d'un scope.
//
// DEUX DÉNOMINATEURS DIFFÉRENTS, ET C'EST LE POINT. `OnMePrepare` se normalise par MES
// frags (« quelle proportion de mes frags m'a été préparée » — mon style) ;
// `MaPartDesAppuis` par les appuis DU CAMP (« sur ce que le camp a distribué, combien m'est
// revenu » — ma place dans l'escouade), face à la part équitable. Les mélanger — diviser
// les appuis reçus par mes frags puis comparer à 1/n — donnerait un nombre qui ne veut
// rien dire.
type CoordinationAppui struct {
	OnMePrepare     Couverture `json:"on_me_prepare"`
	MaPartDesAppuis Couverture `json:"ma_part_des_appuis"`
	// HabituelPct : « on me prépare » MESURÉ SUR LA PÉRIODE DE RÉFÉRENCE (les matchs du filtre de
	// la page, toutes sessions confondues), en pourcentage — le repère de la jauge « on me
	// prépare », que rien ne rapporte à une part équitable. Absent quand la référence est
	// TAUTOLOGIQUE (elle se réduit au scope mesuré : le repère tomberait sur la valeur) ou non
	// mesurée. Jamais un 0.
	HabituelPct *float64 `json:"habituel_pct,omitempty"`
	// ParityPct : la part ÉQUITABLE de `MaPartDesAppuis`, en pourcentage — 100/n pondéré par les
	// appuis de camp de chaque match. Un scope qui mêle 4v4 et BTB n'a pas une parité unique, et
	// la moyenne des effectifs n'en donnerait pas la bonne. Absent quand AUCUN match mesuré n'a
	// d'effectif de camp connu (FFA).
	ParityPct *float64 `json:"parity_pct,omitempty"`
}

// CoordinationMatchPoint — UNE case de la bande de régularité : les comptes bruts d'un
// match mesuré et les deux parts qui s'en déduisent.
//
// Une part NIL a un dénominateur vide : aucun frag, aucun appui dans le camp. La case reste
// GRISE — un zéro s'y lirait comme une contre-performance.
type CoordinationMatchPoint struct {
	MatchID string `json:"match_id"`
	// TeamSize / ParityPct : l'effectif de mon camp et la parité 100/n de CE match
	// (réserve R1). Absents en FFA.
	TeamSize  *int     `json:"team_size,omitempty"`
	ParityPct *float64 `json:"parity_pct,omitempty"`

	// MyKills : frags OFFICIELS du joueur sur le match (base de AssistedSharePct, jamais sous
	// les frags lus par le film).
	MyKills         int `json:"my_kills"`
	MyAssistedKills int `json:"my_assisted_kills"`
	TeamAssists     int `json:"team_assists"`
	AssistsToMe     int `json:"assists_to_me"`
	// AssistShareOfTeamPct : les appuis qui me sont revenus sur ceux distribués dans mon
	// camp, en pourcentage — la grandeur que la bande du §6 peint face à la parité.
	AssistShareOfTeamPct *float64 `json:"assist_share_of_team_pct,omitempty"`
	// AssistedSharePct : mes frags appuyés sur MyKills, en pourcentage.
	AssistedSharePct *float64 `json:"assisted_share_pct,omitempty"`
}

// CoordinationSessionPoint — UN bâton de la frise temporelle : une SOIRÉE.
//
// POURQUOI LA SOIRÉE ET PAS LE MATCH. Une part sur une dizaine de frags bouge de dix points
// quand un seul frag change de côté — la frise par match dessinerait le bruit. La soirée
// cumule assez d'événements, et c'est déjà la maille de la frise de l'Escouade.
type CoordinationSessionPoint struct {
	SessionLabel    string            `json:"session_label"`
	MatchesMeasured int               `json:"matches_measured"`
	MatchesTotal    int               `json:"matches_total"`
	Appui           CoordinationAppui `json:"appui"`
}

// CoordinationBlock — le bloc servi à Sessions (avec PerMatch) et à Timeseries (avec
// Sessions). `Available=false` porte TOUJOURS une raison machine.
type CoordinationBlock struct {
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// MatchesMeasured / MatchesTotal : la COUVERTURE, affichée avec le bloc et jamais en
	// note de bas de page. 0/M est un état légitime (Available=false, raison
	// `no_measured_match`).
	MatchesMeasured int `json:"matches_measured"`
	MatchesTotal    int `json:"matches_total"`

	Appui CoordinationAppui `json:"appui"`

	// PerMatch : une case par match MESURÉ, dans l'ordre du scope (page Sessions).
	PerMatch []CoordinationMatchPoint `json:"per_match,omitempty"`
	// Sessions : un point par soirée mesurée, chronologique (page Séries temporelles).
	Sessions []CoordinationSessionPoint `json:"sessions,omitempty"`
}
