package domain

// coordination_block.go — LE BLOC « COORDINATION » D'UN SCOPE DE MATCHS.
//
// Deux sujets, une seule mesure, trois surfaces (plan AJSUP, décisions D22 du 2026-09-21) :
//
//	RIPOSTE   « mes morts ont-elles été vengées » et « est-ce moi qui venge » — la fenêtre
//	          d'échange de analysis/coordination (5 s), rapportée aux morts de MON CAMP.
//	APPUI     « combien de mes frags m'ont été préparés » et « sur les appuis distribués
//	          dans mon camp, combien me sont revenus ».
//
//	Sessions     le bloc + une case par match (PerMatch) ;
//	Timeseries   le bloc + un point par SOIRÉE (Sessions) ;
//	Match view   un bloc À PART (MatchRiposteBlock) — un match, ce sont des COMPTES.
//
// ─── POURQUOI LE MÊME TYPE POUR SESSIONS ET TIMESERIES ────────────────────────────────
//
// Les deux pages posent la MÊME question à deux mailles. Deux types auraient donné deux
// définitions de « morts de mon camp » libres de diverger au premier ajustement — c'est le
// défaut que l'en-tête de analysis/coordination existe pour empêcher. La maille se lit
// dans le champ rempli : `PerMatch` (session) ou `Sessions` (soirées), jamais les deux.
//
// ─── TOUT TAUX VOYAGE EN Couverture ───────────────────────────────────────────────────
//
// Aucun quotient nu ici : chaque part est une `Couverture` (taux 0..1, compte brut,
// quantité par match, dénominateur, drapeau d'échantillon faible). C'est la règle de forme
// de analysis/coordination, et elle traverse le contrat HTTP telle quelle.
//
// ─── UN MATCH SANS FILM EST ABSENT, JAMAIS À ZÉRO ─────────────────────────────────────
//
// « Mesuré » veut dire : au moins une ligne publiable dans `match_kill_events_latest`. Un
// match non mesuré n'a pas un taux nul, il n'a pas de taux — il ne fournit ni numérateur
// ni dénominateur, et son absence se lit dans `MatchesMeasured / MatchesTotal`.

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
// couple (assistant, tueur crédité).
//
// `AssistXUID` VIDE est un ÉTAT MESURÉ, pas une absence de ligne : « publiable et
// assist_known, personne n'a assisté ». C'est ce qui permet au dénominateur « mes frags
// mesurés » d'exister sans jamais compter un frag dont l'assistance est INCONNUE — la
// doctrine des trois états de assist_pairs.go, rapportée à un scope de matchs.
type CoordinationAppuiRow struct {
	MatchID    string
	AssistXUID string
	KillerXUID string
	Nombre     int
}

// CoordinationEntree — tout ce que le calcul du bloc consomme. Une struct plutôt que six
// paramètres adjacents (seuil CLAUDE.md n 5), et un type de DOMAINE parce que
// analysis/coordination ne déclare aucun type exporté.
type CoordinationEntree struct {
	// MoiXUID : le joueur consulté. Vide = rien à mesurer (le bloc n'a pas de sujet).
	MoiXUID string
	Matchs  []CoordinationMatch
	Equipes EquipesParMatch
	Kills   []KillEvent
	Appuis  []CoordinationAppuiRow
}

// CoordinationRiposte — les deux grandeurs de riposte d'un scope.
type CoordinationRiposte struct {
	// JeSuisCouvert : part de MES morts vengeables qui ont été vengées dans la fenêtre.
	JeSuisCouvert Couverture `json:"je_suis_couvert"`
	// JeRiposte : part des morts vengeables DE MON CAMP que j'ai vengées moi-même. Le
	// dénominateur est bien les morts du CAMP, pas les miennes : sans cela un match où le
	// camp meurt peu gonflerait artificiellement ma part.
	JeRiposte Couverture `json:"je_riposte"`
	// TeamDeaths / TeamDeathsAvenged : les morts VENGEABLES de mon camp et celles qui ont
	// été vengées. Comptes bruts — le face-à-face et les infobulles les affichent tels quels.
	TeamDeaths        int `json:"team_deaths"`
	TeamDeathsAvenged int `json:"team_deaths_avenged"`
	// DelaiMedianMs : délai médian des ripostes portées à mon camp, DANS la fenêtre.
	// Absent quand aucune riposte n'est survenue — jamais un zéro qui se lirait « instantané ».
	DelaiMedianMs *int64 `json:"delai_median_ms,omitempty"`
	// HabituelPct : « je suis couvert » MESURÉ SUR LA PÉRIODE DE RÉFÉRENCE (les matchs du
	// filtre de la page, toutes sessions confondues), en pourcentage. C'est le repère de
	// la jauge « je suis couvert », qui n'a PAS de parité : être couvert ne se compare à
	// aucun 1/n — seulement à son propre habituel.
	//
	// Absent quand la référence est TAUTOLOGIQUE (elle se réduit au scope mesuré : le
	// repère tomberait alors exactement sur la valeur) ou non mesurée. Jamais un 0.
	HabituelPct *float64 `json:"habituel_pct,omitempty"`
	// ParityPct : la part ÉQUITABLE de `JeRiposte`, en pourcentage — 100/n pondéré par les
	// morts de camp de chaque match. Un scope qui mêle 4v4 et BTB n'a pas une parité unique,
	// et la moyenne des effectifs n'en donnerait pas la bonne. Absent quand AUCUN match
	// mesuré n'a d'effectif de camp connu (FFA).
	ParityPct *float64 `json:"parity_pct,omitempty"`
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
	// HabituelPct : « on me prépare » mesuré sur la MÊME période de référence que
	// CoordinationRiposte.HabituelPct, et avec la même règle d'absence. C'est le repère de
	// la jauge « on me prépare », que rien ne rapporte à une part équitable.
	HabituelPct *float64 `json:"habituel_pct,omitempty"`
	// ParityPct : 100/n pondéré par les appuis de camp de chaque match. Même règle
	// d'absence que CoordinationRiposte.ParityPct.
	ParityPct *float64 `json:"parity_pct,omitempty"`
}

// CoordinationMatchPoint — UNE case de la bande de régularité : les comptes bruts d'un
// match mesuré et les deux parts qui s'en déduisent.
//
// Une part NIL est un « non mesuré » : aucune mort vengeable de camp, aucun appui mesuré
// dans le camp. La case reste GRISE — un zéro s'y lirait comme une contre-performance.
type CoordinationMatchPoint struct {
	MatchID string `json:"match_id"`
	// TeamSize / ParityPct : l'effectif de mon camp et la parité 100/n de CE match
	// (réserve R1). Absents en FFA.
	TeamSize  *int     `json:"team_size,omitempty"`
	ParityPct *float64 `json:"parity_pct,omitempty"`

	TeamDeaths        int `json:"team_deaths"`
	TeamDeathsAvenged int `json:"team_deaths_avenged"`
	MyDeaths          int `json:"my_deaths"`
	MyDeathsAvenged   int `json:"my_deaths_avenged"`
	MyRipostes        int `json:"my_ripostes"`
	// RiposteSharePct : mes ripostes sur les morts vengeables de mon camp, en pourcentage.
	RiposteSharePct *float64 `json:"riposte_share_pct,omitempty"`
	// CoveredSharePct : mes morts vengées sur mes morts vengeables, en pourcentage.
	CoveredSharePct *float64 `json:"covered_share_pct,omitempty"`

	MyMeasuredKills int `json:"my_measured_kills"`
	MyAssistedKills int `json:"my_assisted_kills"`
	TeamAssists     int `json:"team_assists"`
	AssistsToMe     int `json:"assists_to_me"`
	// AssistShareOfTeamPct : les appuis qui me sont revenus sur ceux distribués dans mon
	// camp, en pourcentage — la grandeur que la bande du §6 peint face à la parité.
	AssistShareOfTeamPct *float64 `json:"assist_share_of_team_pct,omitempty"`
	// AssistedSharePct : mes frags appuyés sur mes frags mesurés, en pourcentage.
	AssistedSharePct *float64 `json:"assisted_share_pct,omitempty"`
}

// CoordinationSessionPoint — UN bâton de la frise temporelle : une SOIRÉE.
//
// POURQUOI LA SOIRÉE ET PAS LE MATCH. Le dénominateur de « je suis couvert », ce sont mes
// morts : 8 à 14 par match en arène. Une part sur 9 morts bouge de 11 points quand une
// seule mort change de côté — la frise par match dessinerait le bruit. La soirée cumule
// 20 à 35 morts, et c'est déjà la maille de la frise de l'Escouade.
type CoordinationSessionPoint struct {
	SessionLabel    string              `json:"session_label"`
	MatchesMeasured int                 `json:"matches_measured"`
	MatchesTotal    int                 `json:"matches_total"`
	Riposte         CoordinationRiposte `json:"riposte"`
	Appui           CoordinationAppui   `json:"appui"`
}

// CoordinationBlock — le bloc servi à Sessions (avec PerMatch) et à Timeseries (avec
// Sessions). `Available=false` porte TOUJOURS une raison machine.
type CoordinationBlock struct {
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// FenetreMs : la fenêtre d'échange, publiée pour que l'écran écrive « 5 s » sans la
	// recopier — une constante de règle du jeu recopiée côté web diverge au premier
	// ajustement (coordination.FenetreEchangeMs).
	FenetreMs int64 `json:"fenetre_ms"`
	// MatchesMeasured / MatchesTotal : la COUVERTURE, affichée avec le bloc et jamais en
	// note de bas de page. 0/M est un état légitime (Available=false, raison
	// `no_measured_match`).
	MatchesMeasured int `json:"matches_measured"`
	MatchesTotal    int `json:"matches_total"`

	Riposte CoordinationRiposte `json:"riposte"`
	Appui   CoordinationAppui   `json:"appui"`

	// PerMatch : une case par match MESURÉ, dans l'ordre du scope (page Sessions).
	PerMatch []CoordinationMatchPoint `json:"per_match,omitempty"`
	// Sessions : un point par soirée mesurée, chronologique (page Séries temporelles).
	Sessions []CoordinationSessionPoint `json:"sessions,omitempty"`
}

// ---------------------------------------------------------------------------
// Page MATCH — des comptes, jamais un taux
// ---------------------------------------------------------------------------

// MatchRiposteDeath — UNE mort du match et sa riposte.
//
// Le camp de la victime vient du scoreboard (`team_id`) : c'est lui qui range les joueurs
// en deux graphes. Nil = camp inconnu (FFA, joueur absent du tableau des scores).
type MatchRiposteDeath struct {
	VictimXUID     string `json:"victim_xuid,omitempty"`
	VictimGamertag string `json:"victim_gamertag,omitempty"`
	VictimTeamID   *int   `json:"victim_team_id,omitempty"`
	KillerXUID     string `json:"killer_xuid,omitempty"`
	TimeMs         int64  `json:"time_ms"`
	// Avenged / Avenger* / DelaiMs : la riposte, quand elle a eu lieu DANS la fenêtre.
	// `DelaiMs` absent quand la mort n'est pas vengée — jamais un 0 qui se lirait
	// « vengée instantanément ».
	Avenged         bool   `json:"avenged"`
	AvengerXUID     string `json:"avenger_xuid,omitempty"`
	AvengerGamertag string `json:"avenger_gamertag,omitempty"`
	DelaiMs         *int64 `json:"delai_ms,omitempty"`
	// Vengeable : un échange était POSSIBLE (tueur identifié, deux camps connus et
	// adverses). Une mort non vengeable n'est pas un échec de riposte : personne ne
	// pouvait la venger.
	Vengeable bool `json:"vengeable"`
}

// MatchRiposteePlayer — les deux comptes d'UN joueur du match : ses morts vengées par son
// camp (événement SUBI) et les ripostes qu'il a portées (événement PORTÉ).
//
// LES DEUX CÔTÉS NE S'ADDITIONNENT PAS, et c'est pourquoi ils voyagent en deux champs
// plutôt qu'en un solde : un joueur qui est beaucoup vengé et qui venge peu ne joue pas
// comme un joueur dont les deux comptes sont faibles.
type MatchRiposteePlayer struct {
	XUID     string `json:"xuid,omitempty"`
	Gamertag string `json:"gamertag,omitempty"`
	TeamID   *int   `json:"team_id,omitempty"`
	// DeathsAvenged : ses morts vengées par son camp. Ripostes : les ripostes qu'il a portées.
	DeathsAvenged int `json:"deaths_avenged"`
	Ripostes      int `json:"ripostes"`
}

// MatchRiposteBlock — le bloc « Riposte » de l'onglet Combat.
//
// AUCUN TAUX ICI, PAR DÉCISION (D21) : un taux sur 11 morts est du bruit affiché avec deux
// décimales. Ce sont des comptes exhaustifs du match, et la seule réserve est la couverture
// du film — sans film décodé, pas d'ordre des morts, donc pas de bloc (nil), jamais une
// section qui disparaît sans rien dire.
type MatchRiposteBlock struct {
	FenetreMs int64 `json:"fenetre_ms"`
	// MeasuredDeaths : les morts du match lues dans le journal. Zéro n'arrive pas — le
	// service n'émet alors aucun bloc.
	MeasuredDeaths int                   `json:"measured_deaths"`
	Deaths         []MatchRiposteDeath   `json:"deaths"`
	Players        []MatchRiposteePlayer `json:"players"`
}
