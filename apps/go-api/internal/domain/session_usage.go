// Package domain — session_usage.go : le bloc « usages d'équipement, socles et
// objectifs » de la page détail de session (chantier session-usage, S2 —
// .ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md §5/S2).
//
// DOCTRINE DU CONTRAT (§1 du handoff) : TOUT axe de comparaison est NORMALISÉ —
// parts en pourcentage, parités (100/effectif), cadences PAR MATCH mesuré. Les
// comptes bruts (player_total, team_total, lobby_total, pad_unnamed_total,
// occupations de bonus) ne survivent que comme DÉNOMINATEURS D'HONNÊTETÉ, à
// afficher en texte à côté d'un taux — jamais comme axe. La référence (l'équipe
// d'en face) n'apparaît nulle part : elle est le complément du dénominateur.
//
// DEUX DÉNOMINATEURS PARTOUT : la part de MON ÉQUIPE (parité = 100/effectif
// d'équipe) et la part du LOBBY (parité = 100/effectif du lobby). Les grenades
// sont produites en S1 mais NE FONT PAS partie du contrat (décision utilisateur
// 2026-09-04 : « ce ne sont pas des équipements »).
//
// CONTEXTE ESCOUADE : le scope des matchs est résolu EN AMONT par
// Filters.MatchContext (le bloc agrège les matchs de la session affichée, quel
// que soit le contexte) ; en contexte escouade, les coéquipiers suivis reçoivent
// EN PLUS une ligne par grandeur (squad) — la « piste du lobby découpée par
// joueur » de la maquette en a besoin.
package domain

// Raisons machine du bloc indisponible (jamais un libellé : l'i18n vit au front).
const (
	// SessionUsageUnsupported : le titre ne déclare pas la capability
	// film.usage_summary — réponse partielle propre, jamais un 500 (ADR 0011).
	SessionUsageUnsupported = "unsupported"
	// SessionUsageLoadFailed : la lecture des vues a échoué (best-effort dégradé,
	// l'erreur est loggée côté service).
	SessionUsageLoadFailed = "load_failed"
)

// SessionUsageShares — le triplet joueur / son camp / lobby d'UNE grandeur, avec
// ses parts croisées. Les totaux sont les dénominateurs d'honnêteté (texte) ; les
// parts sont les axes. Une part est nil quand son dénominateur est nul (0/0 n'est
// pas 0 %) ou quand le camp du joueur est inconnu sur tout le scope.
//
// RÈGLE DE SCOPE (ronde de correction S2) : les grandeurs relatives à l'ÉQUIPE
// (team_total, player_share_of_team_pct, team_share_of_lobby_pct) se calculent
// sur le SOUS-ENSEMBLE des matchs du scope à camp CONNU — numérateurs ET
// dénominateurs. Sans quoi une session mêlant matchs en équipe et FFA croiserait
// deux scopes et player_share_of_team_pct dépasserait 100 %. Sous-ensemble
// vide : team_total est nil — jamais un 0 inventé pour dire « inconnu ».
// player_total, lobby_total et les parts joueur/lobby restent sur TOUT le scope.
type SessionUsageShares struct {
	PlayerTotal float64 `json:"player_total"`
	// TeamTotal : somme du camp du joueur sur les matchs à camp connu du scope.
	// nil quand aucun match du scope n'a de camp connu (session entièrement FFA).
	TeamTotal  *float64 `json:"team_total,omitempty"`
	LobbyTotal float64  `json:"lobby_total"`
	// TeamShareOfLobbyPct : « mon camp / lobby » (§7 du handoff, 1re colonne).
	TeamShareOfLobbyPct *float64 `json:"team_share_of_lobby_pct,omitempty"`
	// PlayerShareOfTeamPct : « joueur / son équipe » (2e colonne).
	PlayerShareOfTeamPct *float64 `json:"player_share_of_team_pct,omitempty"`
	// PlayerShareOfLobbyPct : « joueur / lobby » (3e colonne).
	PlayerShareOfLobbyPct *float64 `json:"player_share_of_lobby_pct,omitempty"`
}

// SessionUsageSquadPlayer — l'identité d'UN coéquipier suivi du contexte
// escouade (l'ordre de la liste est l'ordre d'affichage — jetons
// squad-player-1..3 côté front). Le joueur de la route n'y figure pas : ses
// grandeurs sont les champs Player* des métriques.
type SessionUsageSquadPlayer struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
}

// SessionUsageSquadShare — la ligne d'UN coéquipier suivi sur UNE grandeur,
// mêmes conventions ET mêmes scopes que les champs Player* (total = dénominateur
// d'honnêteté sur tout le scope ; part d'équipe sur les matchs à camp connu ;
// cadence PAR MATCH MESURÉ ; nil quand le dénominateur est nul).
// Alignée sur SessionUsageBlock.SquadPlayers par XUID.
type SessionUsageSquadShare struct {
	XUID            string   `json:"xuid"`
	Total           float64  `json:"total"`
	ShareOfTeamPct  *float64 `json:"share_of_team_pct,omitempty"`
	ShareOfLobbyPct *float64 `json:"share_of_lobby_pct,omitempty"`
	PerMatch        *float64 `json:"per_match,omitempty"`
}

// SessionUsageMatchPoint — les parts d'UN match mesuré (une case de la bande de
// régularité ; l'étendue de la jauge double s'en dérive). Une part nil = non
// calculable sur ce match (dénominateur nul, camp inconnu).
type SessionUsageMatchPoint struct {
	MatchID               string   `json:"match_id"`
	PlayerShareOfTeamPct  *float64 `json:"player_share_of_team_pct,omitempty"`
	PlayerShareOfLobbyPct *float64 `json:"player_share_of_lobby_pct,omitempty"`
	TeamShareOfLobbyPct   *float64 `json:"team_share_of_lobby_pct,omitempty"`
	// TeamSize / PlayerTeam — L'EFFECTIF DE MON CAMP ET SON NUMÉRO, PAR MATCH
	// (réserve R1 du 2026-09-21). Sans eux, la jauge « ma part face à 1/n » de la
	// session n'a pas de parité juste : TeamSizeAvg est une MOYENNE de session, et
	// une soirée qui mêle 4v4 et BTB y perd la parité de chaque match.
	//
	// ABSENTS = CAMP INCONNU (FFA, participant manquant), jamais un 1 inventé : une
	// parité de 100 % fabriquée classerait le joueur au-dessus de son tour sur tous
	// les matchs sans camp. Même source que les parts d'équipe de la même ligne
	// (sessionusage.TeamContext) — l'un est nil exactement quand les autres le sont.
	// L'effectif compte les joueurs PRÉSENTS À LA FIN, bots inclus.
	TeamSize   *int `json:"team_size,omitempty"`
	PlayerTeam *int `json:"player_team,omitempty"`
}

// SessionUsageOutcomes — LES TROIS ISSUES d'un objet d'équipement pris, portées
// par les grandeurs "equipment_<famille>" et par elles seules (étape E3,
// PLAN_EQUIPEMENT_GACHIS_2026-09-09 ; décision P1). Les trois sont exclusives et
// leur somme vaut le total de la grandeur : c'est ce qui permet à la barre de
// porter DEUX lectures à la fois (sa longueur est ma part, son remplissage est
// l'issue — décision P6).
//
// « UTILISÉ » A DEUX DÉFINITIONS et le lecteur ne voit pas la différence
// (décision P2) : un équipement d'ACTIVATION sert quand il est activé (les deux
// bonus, par le canal des épisodes), un DÉPLOYABLE sert quand il est posé. Le
// contrat ne distingue pas les deux — c'est un détail de calcul, tranché côté Go.
//
// LES DEUX TAUX DE RÉFÉRENCE M'EXCLUENT (décision P7) : se comparer à une moyenne
// qui vous contient amortit le signal. « Le reste de mon équipe » est mon camp
// MOINS moi ; « eux » est le lobby MOINS mon camp. Les deux sont nil quand leur
// dénominateur est vide — jamais un 0 % inventé pour dire « personne n'en a pris ».
type SessionUsageOutcomes struct {
	// Used / Kept / Dropped : les trois issues du JOUEUR sur tout le scope mesuré.
	// Leur somme vaut SessionUsageShares.PlayerTotal de la même métrique.
	Used    float64 `json:"used"`
	Kept    float64 `json:"kept"`
	Dropped float64 `json:"dropped"`
	// Taken : les PRISES du canal `equipmentChanges` — le dénominateur d'honnêteté
	// (texte), jamais l'axe. Il DIFFÈRE de la somme des trois issues : pour un
	// déployable, une pose est une CHARGE et non un objet (un capteur pris une fois
	// et lancé quatre fois donne 4 poses pour 1 prise), et l'écart résiduel mesuré
	// est absorbé par le clamp du gardé. Zéro = le canal n'a nommé aucune prise de
	// cette famille sur le scope, ce qui n'annule pas les issues mesurées.
	Taken float64 `json:"taken"`
	// UsedRatePct : ma part utilisée de la barre — 100*used/(used+kept+dropped).
	// nil quand la barre est vide.
	UsedRatePct *float64 `json:"used_rate_pct,omitempty"`
	// TeammatesUsedRatePct : le même taux pour MON ÉQUIPE MOINS MOI.
	TeammatesUsedRatePct *float64 `json:"teammates_used_rate_pct,omitempty"`
	// OpponentsUsedRatePct : le même taux pour LE LOBBY MOINS MON ÉQUIPE.
	OpponentsUsedRatePct *float64 `json:"opponents_used_rate_pct,omitempty"`
}

// SessionUsageMetric — UNE grandeur agrégée sur les matchs MESURÉS de la session.
// Clés servies : "grapple_pulls", "camo_episodes", "overshield_episodes",
// "dropped_objects", "pad_pickups", "deployed_<famille>" par famille déployée
// observée (ensemble ouvert — manifeste du titre, ex. "deployed_wall"), et
// "equipment_<famille>" par famille du BILAN D'ÉQUIPEMENT (étape E3) — la seule
// dont la valeur est un compte d'OBJETS et non de gestes, et la seule qui porte
// Outcomes.
type SessionUsageMetric struct {
	Key string `json:"key"`
	SessionUsageShares
	// Outcomes : les trois issues, sur les grandeurs "equipment_<famille>"
	// UNIQUEMENT. Absent partout ailleurs — un titre dont le canal des ramassages
	// n'est pas mesuré ne publie RIEN ici, jamais des zéros.
	Outcomes *SessionUsageOutcomes `json:"outcomes,omitempty"`
	// Cadences PAR MATCH MESURÉ (décision utilisateur du 2026-09-13 : « Cadence
	// c'est par match, pas par minutes ») — total divisé par le NOMBRE de matchs
	// du scope, jamais par une durée. TeamPerMatch : scope camp connu (nombre de
	// matchs mesurés à camp connu) — nil quand ce scope est vide, jamais une
	// cadence inventée.
	PlayerPerMatch *float64 `json:"player_per_match,omitempty"`
	TeamPerMatch   *float64 `json:"team_per_match,omitempty"`
	LobbyPerMatch  *float64 `json:"lobby_per_match,omitempty"`
	// Matchs où la part du joueur dépasse LA PARITÉ DU MATCH (100/effectif de CE
	// match, pas la moyenne de session), contre chacun des deux dénominateurs.
	// MatchesAboveTeamParity : nil quand aucun match mesuré n'a de camp connu
	// (le compte serait invérifiable — même règle de scope que team_total).
	MatchesAboveTeamParity  *int `json:"matches_above_team_parity,omitempty"`
	MatchesAboveLobbyParity int  `json:"matches_above_lobby_parity"`
	// PerMatch : une entrée par match MESURÉ, dans l'ordre de la session.
	PerMatch []SessionUsageMatchPoint `json:"per_match,omitempty"`
	// Squad : une ligne par coéquipier suivi (contexte escouade uniquement),
	// alignée sur SessionUsageBlock.SquadPlayers.
	Squad []SessionUsageSquadShare `json:"squad,omitempty"`
}

// SessionUsagePadFamily — ventilation des prises de socle d'ARME nommées par clé
// de famille NORMALISÉE (replay.PadWeaponFamilyKey : huit hexa minuscules).
//
// FamilyLabel est le NOM de l'arme dans la langue de la requête, résolu au service
// contre le catalogue du titre (`session_page_usage_labels.go`, patron
// `replay_weapon_labels.go`). ABSENT quand le catalogue ne connaît pas la famille :
// le client affiche alors la clé, jamais un nom approchant — même règle que le
// catalogue du rejeu (« un nom approchant se lit comme une certitude »). La clé,
// elle, reste TOUJOURS servie : c'est elle qui identifie la ligne.
type SessionUsagePadFamily struct {
	FamilyKey   string `json:"family_key"`
	FamilyLabel string `json:"family_label,omitempty"`
	SessionUsageShares
}

// SessionUsagePowerup — occupations de socle de BONUS, ANONYMES par construction
// (un bonus s'identifie par un nom, jamais rattachable à un joueur — §4 du
// handoff). Grandeur de MATCH : ni part d'équipe ni part de joueur, seulement le
// total (texte) et sa cadence par match.
type SessionUsagePowerup struct {
	FamilyKey   string   `json:"family_key"` // nom canonique ("powerup_camo", ...)
	Occupations int      `json:"occupations"`
	PerMatch    *float64 `json:"per_match,omitempty"`
}

// SessionObjectiveRoleMetric — un rôle d'objectif agrégé. Role est une clé de
// narrative.ObjectiveRole ("take" | "defend" | "hold" — la classification des
// colonnes par rôle vit en SOURCE UNIQUE dans analysis/narrative,
// objective_roles.go). IsDuration : la grandeur est en secondes (rôle « tenir »)
// — les parts restent des pourcentages, seuls les totaux changent d'unité.
type SessionObjectiveRoleMetric struct {
	Role       string `json:"role"`
	IsDuration bool   `json:"is_duration,omitempty"`
	SessionUsageShares
	// Squad : une ligne par coéquipier suivi (contexte escouade uniquement),
	// alignée sur SessionUsageBlock.SquadPlayers. PerMatch reste nil : les rôles
	// d'objectif se publient en parts, jamais en cadence.
	Squad []SessionUsageSquadShare `json:"squad,omitempty"`
}

// SessionObjectiveFamilyBlock — le détail d'une famille de mode (clés
// narrative : "ctf", "zones_koth", "zones_strongholds", "oddball", "stockpile",
// "extraction", "vip"). Un rôle sans colonne pour la famille (extraction n'a pas
// de « tenir ») est absent de Roles.
type SessionObjectiveFamilyBlock struct {
	Family  string                       `json:"family"`
	Matches int                          `json:"matches"`
	Roles   []SessionObjectiveRoleMetric `json:"roles"`
}

// SessionObjectivesBlock — le bloc 3 : objectifs par rôle et par famille, lus de
// match_objective_stats_latest (les deux camps y sont déjà — rien n'est produit,
// tout est agrégé à la lecture). Son scope est l'ensemble des matchs de la
// session portant des stats objectifs — INDÉPENDANT de la couverture des films.
type SessionObjectivesBlock struct {
	MatchesWithObjectives int      `json:"matches_with_objectives"`
	TeamSizeAvg           float64  `json:"team_size_avg,omitempty"`
	LobbySizeAvg          float64  `json:"lobby_size_avg,omitempty"`
	TeamParityPct         *float64 `json:"team_parity_pct,omitempty"`
	LobbyParityPct        *float64 `json:"lobby_parity_pct,omitempty"`
	// Roles : les trois rôles agrégés TOUTES familles confondues (le tableau §7).
	Roles []SessionObjectiveRoleMetric `json:"roles"`
	// Families : le même découpage, par famille de mode.
	Families []SessionObjectiveFamilyBlock `json:"families,omitempty"`
	// FlagGrabsNet : les PRISES NETTES de drapeau du scope, publiées À PART des
	// rôles ci-dessus. Absent = aucune prise lue sur ce scope (mode sans
	// drapeau, ou aucun film lu).
	//
	// POURQUOI PAS DANS `Roles` : la grandeur entre bien dans « prendre » au
	// niveau de la table des rôles, mais elle n'est mesurée que sur les matchs
	// dont le film a été lu. La verser dans la somme du rôle ferait compter un
	// match sans film comme un match sans prise, et changerait le dénominateur
	// des parts selon la couverture du film. Elle porte donc SES dénominateurs.
	FlagGrabsNet *SessionFlagGrabsNetBlock `json:"flag_grabs_net,omitempty"`
}

// SessionFlagGrabsNetBlock — les prises de drapeau du scope, brutes et nettes.
//
// LE COMPTEUR OFFICIEL COMPTE LE JONGLAGE (lancer le drapeau devant soi pour
// courir plus vite, puis le reprendre). Les deux totaux sont publiés ensemble
// pour que l'écart se VOIE : c'est lui qui justifie la grandeur nette.
type SessionFlagGrabsNetBlock struct {
	// MatchesWithFlagFamily / MatchesMeasured : le dénominateur de couverture.
	// Un match À DRAPEAU sans film lu n'a AUCUNE prise ici — ce n'est pas un
	// match sans prise, c'est un match non mesuré, et l'écart entre les deux
	// nombres le dit.
	//
	// LE DÉNOMINATEUR EST LA FAMILLE DRAPEAU, PAS « les matchs à objectif » :
	// compter les matchs de zones ou de crâne attribuerait au film l'absence de
	// matchs qui n'ont simplement pas de drapeau.
	MatchesWithFlagFamily int `json:"matches_with_flag_family"`
	MatchesMeasured       int `json:"matches_measured"`
	// MatchesTeamKnown : ceux des matchs mesurés où le camp du joueur suivi est
	// connu — le SEUL périmètre sur lequel une part d'équipe a un sens.
	MatchesTeamKnown int `json:"matches_team_known"`
	// OpeningsTotal : les OUVERTURES DE PORTAGE comptées par l'oracle du film sur
	// ces matchs (une fois par match). C'est le dénominateur de `lobby_raw_total`
	// — les pistes ne portent que les prises que le pont a su nommer et situer.
	OpeningsTotal int `json:"openings_total"`
	// WindowSeconds : la fenêtre de jonglage appliquée. ZÉRO quand le scope en
	// mêle PLUSIEURS (parc partiellement re-projeté après un changement de
	// règle) : il n'y a alors pas UNE fenêtre, et en publier une mentirait sur
	// l'autre.
	WindowSeconds float64 `json:"window_seconds,omitempty"`
	// Totaux nets puis bruts du joueur et du lobby, sur TOUS les matchs mesurés.
	//
	// ⚠ CES DEUX-LÀ NE SE COMPARENT PAS À `TeamTotal` : l'équipe ne se compte que
	// sur les matchs à camp connu. Le couple comparable est
	// (PlayerTeamScopeTotal, TeamTotal) juste en dessous.
	PlayerTotal    int `json:"player_total"`
	LobbyTotal     int `json:"lobby_total"`
	PlayerRawTotal int `json:"player_raw_total"`
	LobbyRawTotal  int `json:"lobby_raw_total"`
	// PlayerTeamScopeTotal / TeamTotal : le couple COMPARABLE, tous deux
	// restreints aux matchs à camp connu (même règle que les métriques d'usage :
	// numérateur ET dénominateur sur le même périmètre, sinon la part dépasse
	// 100 % dès qu'un match du scope a un camp inconnu).
	PlayerTeamScopeTotal    int `json:"player_team_scope_total"`
	TeamTotal               int `json:"team_total"`
	PlayerTeamScopeRawTotal int `json:"player_team_scope_raw_total"`
	TeamRawTotal            int `json:"team_raw_total"`
	// PlayerShareOfTeamPct : part du joueur dans les prises nettes de son camp,
	// sur ce même périmètre. Absent quand le camp n'a pris aucun drapeau — pas
	// de part, jamais 0 %.
	PlayerShareOfTeamPct *float64 `json:"player_share_of_team_pct,omitempty"`
}

// SessionUsageBlock — le bloc complet servi avec la page détail de session
// (contexte Solo/Escouade déjà résolu en amont par Filters.MatchContext : le bloc
// agrège LES MATCHS DE LA SESSION AFFICHÉE, quel que soit le contexte).
type SessionUsageBlock struct {
	// Available : false = bloc non calculable (voir UnavailableReason) ; les
	// autres champs valent alors zéro sauf MatchesTotal.
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// MatchesMeasured / MatchesTotal : « matchs mesurés N/M » — la couverture des
	// films n'est jamais totale et l'écran DOIT le dire (§5/S2). Un match est
	// mesuré s'il a une ligne match_usage_films_latest. 0/M est un état légitime :
	// le bloc est vide mais présent.
	MatchesMeasured int `json:"matches_measured"`
	MatchesTotal    int `json:"matches_total"`
	// MeasuredDurationSeconds : durée jouée cumulée des matchs MESURÉS à durée
	// CONNUE — le dénominateur des cadences joueur/lobby par dix minutes. Un
	// match mesuré sans échelle de temps (artefact sans durée, aucun repli) n'y
	// entre pas : il est exclu des cadences (numérateur et dénominateur) mais
	// reste compté dans les totaux et les parts.
	MeasuredDurationSeconds float64 `json:"measured_duration_seconds,omitempty"`
	// Effectifs MOYENS sur les matchs mesurés (joueurs présents à la fin, bots
	// inclus) et les deux parités qui s'en déduisent (100/effectif).
	TeamSizeAvg    float64  `json:"team_size_avg,omitempty"`
	LobbySizeAvg   float64  `json:"lobby_size_avg,omitempty"`
	TeamParityPct  *float64 `json:"team_parity_pct,omitempty"`
	LobbyParityPct *float64 `json:"lobby_parity_pct,omitempty"`
	// PadUnnamedTotal : prises de socle d'ARME dont le ramasseur n'est pas nommé
	// par le film — dénominateur d'honnêteté de la note de pied (§7 : 82 sur le
	// témoin). Jamais réparties : elles n'appartiennent à personne.
	PadUnnamedTotal int `json:"pad_unnamed_total,omitempty"`
	// SquadPlayers : les coéquipiers suivis du contexte escouade (ordre
	// d'affichage ; vide en contexte solo ou sans coéquipier commun à toute la
	// session). Les lignes Squad des métriques s'y alignent par XUID.
	SquadPlayers []SessionUsageSquadPlayer `json:"squad_players,omitempty"`
	Metrics      []SessionUsageMetric      `json:"metrics,omitempty"`
	PadFamilies  []SessionUsagePadFamily   `json:"pad_families,omitempty"`
	// PadTiers : les memes prises, rangees par NIVEAU d'arme (base / terrain / puissance /
	// bonus / non classe). Autre table, autre perimetre mesure : cf. SessionUsagePadTiersBlock.
	PadTiers       *SessionUsagePadTiersBlock `json:"pad_tiers,omitempty"`
	PowerupPickups []SessionUsagePowerup      `json:"powerup_pickups,omitempty"`
	Objectives     *SessionObjectivesBlock    `json:"objectives,omitempty"`
}

// Les NIVEAUX D'ARME, tels qu'ils sont écrits en base ET servis au contrat.
//
// UNE SEULE SOURCE, ET ELLE EST ICI (correctif de revue, 2026-09-14). Ces valeurs étaient
// recopiées de `persist.PadTier*` sans rien pour les tenir ensemble : renommer une constante
// côté écriture — `PadTierGround` de « terrain » à « sol » — laissait toute la suite verte et
// faisait disparaître un niveau entier des trois pages, en silence. Le paquet `persist`
// réexporte désormais CES constantes, et un garde-rail confronte les deux listes.
//
// Ce sont des valeurs de DONNÉE, jamais des libellés : la traduction vit dans l'i18n du web.
const (
	PadTierBase         = "base"
	PadTierGround       = "terrain"
	PadTierPower        = "puissance"
	PadTierPowerup      = "bonus"
	PadTierUnclassified = "non_classe"
	// PadTierNoPickup — LE ZÉRO MESURÉ d'un joueur qui n'a pris aucun socle. Il ne figure pas
	// dans PadTierOrder : ce n'est pas un niveau de contrôle, c'est ce qui sépare « ce joueur
	// n'a rien pris » de « on n'a pas regardé ».
	PadTierNoPickup = "aucune_prise"
)

// PadTierOrder — L'ORDRE DE LECTURE DES NIVEAUX D'ARME, écrit une fois.
//
// Il n'est PAS trié par volume, et c'est délibéré : un classement dont l'ordre change d'une
// session à l'autre ne se compare pas d'un écran au suivant.
var PadTierOrder = []string{
	PadTierBase, PadTierGround, PadTierPower, PadTierPowerup, PadTierUnclassified,
}

// SessionUsagePadTierWeapon — le détail par ARME d'un niveau, servi au survol.
//
// PAS DE PART ICI, ET C'EST VOULU : une part par arme dans un niveau ajouterait trois
// dénominateurs pour une lecture que personne n'a demandée. Deux comptes suffisent à dire
// « sur les N prises de puissance du lobby, j'en ai fait n ».
type SessionUsagePadTierWeapon struct {
	FamilyKey string `json:"family_key"`
	// FamilyLabel : le nom de l'arme dans la langue de la requête, résolu au service contre le
	// catalogue du titre. ABSENT quand le catalogue ne connaît pas la famille — le client
	// affiche alors la clé, jamais un nom approchant (même règle que SessionUsagePadFamily).
	FamilyLabel   string  `json:"family_label,omitempty"`
	PlayerPickups float64 `json:"player_pickups"`
	LobbyPickups  float64 `json:"lobby_pickups"`
}

// SessionUsagePadTier — UN niveau d'arme et ses grandeurs.
type SessionUsagePadTier struct {
	// Tier : l'une des valeurs de PadTierOrder.
	Tier string `json:"tier"`
	SessionUsageShares
	// PlayerPerMatch : cadence du joueur PAR MATCH MESURÉ (même dénominateur que les autres
	// cadences du bloc — décision utilisateur du 2026-09-13).
	PlayerPerMatch *float64 `json:"player_per_match,omitempty"`
	// Weapons : le détail par arme, du plus pris au moins pris.
	Weapons []SessionUsagePadTierWeapon `json:"weapons,omitempty"`
}

// SessionUsagePadTiersBlock — LES PRISES DE SOCLE PAR NIVEAU D'ARME sur la session.
//
// LE BLOC EST OMIS, JAMAIS SERVI A ZERO, quand aucun match de la session n'a été projeté : un
// bloc vide se lirait « aucune prise » là où la vérité est « pas encore mesuré ».
//
// LES QUATRE COMPTEURS SONT DES DENOMINATEURS D'HONNETETE, et chacun dit une chose que les
// autres ne disent pas — le détail est en tête de `analysis/sessionusage/pad_tiers.go`.
type SessionUsagePadTiersBlock struct {
	// MatchesTotal : les matchs du SCOPE. Le denominateur d honnetete du bloc, et il est A LUI :
	// la carte affichait la couverture du RESUME D USAGE, qui porte sur un autre perimetre (une
	// passe distincte, sur d autres matchs). Constat de revue, 2026-09-14.
	MatchesTotal    int `json:"matches_total"`
	MatchesMeasured int `json:"matches_measured"`
	// MatchesWithPads : parmi eux, ceux dont le film a publié au moins un socle. L'écart avec
	// le précédent n'est pas une panne : un mode peut n'allumer aucun emplacement.
	MatchesWithPads int `json:"matches_with_pads"`
	// MatchesTiersEstablished : parmi eux, ceux dont la carte est dans la référence des
	// emplacements. En dessous, tout tombe en « non classé » — un défaut de référence, pas un
	// fait de jeu, et l'écran doit le dire.
	MatchesTiersEstablished int `json:"matches_tiers_established"`
	// MatchesRandomStarts : parmi eux, ceux dont le mode distribue des départs aléatoires. Le
	// niveau « base » n'y est pas publié.
	MatchesRandomStarts int `json:"matches_random_starts"`
	// LES TROIS PARITES DU BLOC, calculees SUR SON PERIMETRE (les matchs dont les niveaux ont
	// ete projetes) et non sur celui du resume d usage — meme raison que MatchesTotal : deux
	// perimetres, deux effectifs moyens, donc deux traits de parite. Nil quand aucun effectif
	// n est connu : un trait de parite invente est pire que pas de trait.
	TeamParityPct        *float64              `json:"team_parity_pct,omitempty"`
	LobbyParityPct       *float64              `json:"lobby_parity_pct,omitempty"`
	TeamOfLobbyParityPct *float64              `json:"team_of_lobby_parity_pct,omitempty"`
	Tiers                []SessionUsagePadTier `json:"tiers,omitempty"`
}
