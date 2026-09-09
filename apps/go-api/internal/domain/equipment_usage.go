// Package domain — equipment_usage.go : LE BLOC « SERVI OU GÂCHÉ » AU GRAIN
// PÉRIODE, publié avec la page Synthèse (étape E5) et la page Escouade (étape
// E6.1) du PLAN_EQUIPEMENT_GACHIS_2026-09-09.
//
// POURQUOI UN TYPE À PART ET NON SessionUsageBlock. Le bloc de la page Sessions
// est bâti pour une SESSION : il porte un point par match (bande de régularité),
// des cadences par dix minutes et deux parités d'effectif. Sur un scope de
// période — la Synthèse peut en compter des milliers — le point par match n'a
// aucun lecteur et pèserait à lui seul plus que tout le reste de la réponse. Ce
// bloc-ci ne publie donc QUE ce que les deux pages rendent : les trois issues par
// famille, la même chose par joueur suivi, et les comptes des deux donuts.
//
// L'AXE EST EN COMPTES, PAS EN POURCENTAGE (décision P9) : sur Solo et Escouade,
// la barre se lit en objets pris, sans trait de parité (il n'a pas de position sur
// un axe de comptes). Les SEULS pourcentages du bloc sont les trois taux d'issue
// hérités de SessionUsageOutcomes — dont les deux références qui EXCLUENT le
// sujet (décision P7).
//
// LE GO NE CALCULE AUCUNE PART DE DONUT (décisions P10/P11) : il publie les cinq
// comptes exclusifs (lobby, moi, mes amis, le reste de mon équipe, eux) et le
// front en fait des arcs et deux sous-totaux. Un pourcentage calculé ici serait
// une seconde vérité à côté de comptes déjà exacts.
package domain

// EquipmentUsageBlock — le bloc complet, attaché aux réponses des pages Synthèse
// et Escouade. Nil quand le scope n'a aucun match ; présent avec Available=false
// et une raison MACHINE (SessionUsageUnsupported / SessionUsageLoadFailed — mêmes
// constantes que le bloc de session, jamais un second vocabulaire) quand le titre
// ne déclare pas film.usage_summary ou que la lecture a échoué.
type EquipmentUsageBlock struct {
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// MatchesMeasured / MatchesTotal : « matchs mesurés N/M ». La couverture des
	// films n'est jamais totale et l'écran DOIT le dire ; 0/M est un état légitime.
	MatchesMeasured int `json:"matches_measured"`
	MatchesTotal    int `json:"matches_total"`
	// Families : une ligne par famille du bilan d'équipement, les grandeurs du
	// JOUEUR DE LA ROUTE, triées du plus pris au moins pris (décision P9). Le
	// répulseur n'y figure jamais (décision P4 : son usage n'est mesuré par aucun
	// canal — une ligne dirait « 0 utilisation » là où la vérité est « non mesuré »).
	Families []EquipmentUsageFamilyLine `json:"families,omitempty"`
	// TrackedPlayers : identités des coéquipiers suivis du scope, dans l'ordre
	// d'affichage (jetons squad-player-2..4 côté front). Vide hors contexte
	// escouade ou sans ami configuré présent sur le scope. Les lignes Players et
	// les ventilations ByFriend s'y alignent par XUID.
	TrackedPlayers []SessionUsageSquadPlayer `json:"tracked_players,omitempty"`
	// Players : une ligne par SUJET, TOUTES FAMILLES CONFONDUES — le joueur de la
	// route EN TÊTE, puis les coéquipiers suivis dans l'ordre de TrackedPlayers.
	// C'est la publication « par joueur suivi » de la page Escouade (étape E6.1).
	Players []EquipmentUsagePlayerLine `json:"players,omitempty"`
	// EquipmentParties / WeaponPadParties : les comptes des DEUX donuts (P10) —
	// objets d'équipement d'une part, prises de socle d'ARME de l'autre. Nil quand
	// aucun match mesuré du scope n'a de camp connu : sans camp, « le reste de mon
	// équipe » et « eux » n'existent pas, et un donut à zéro serait une affirmation.
	EquipmentParties *EquipmentUsageParties `json:"equipment_parties,omitempty"`
	WeaponPadParties *EquipmentUsageParties `json:"weapon_pad_parties,omitempty"`
}

// EquipmentUsageFamilyLine — UNE famille du bilan, vue du joueur de la route.
// Porte exactement les grandeurs de la page Sessions (familles, prises, trois
// issues, deux taux de référence) — la longueur de la barre est la somme des trois
// issues, son remplissage est leur répartition (décision P6).
type EquipmentUsageFamilyLine struct {
	// FamilyKey : la clé de famille du résumé (« wall », « powerup_camo »...), le
	// vocabulaire des POSES. Le libellé lisible vit côté front.
	FamilyKey string `json:"family_key"`
	SessionUsageOutcomes
}

// EquipmentUsagePlayerLine — UN sujet (moi ou un coéquipier suivi), toutes
// familles confondues. Les issues portent sur TOUT le scope mesuré (même règle que
// SessionUsageShares.PlayerTotal) ; les deux taux de référence excluent CE
// sujet-là, jamais seulement le joueur de la route (décision P7).
type EquipmentUsagePlayerLine struct {
	XUID string `json:"xuid"`
	SessionUsageOutcomes
	// PadPickups : prises de socle d'ARME du sujet sur le scope mesuré. Les armes
	// spéciales suivent la même grammaire que l'équipement (décision P5), mais leur
	// issue « avoir tiré » n'est PAS persistée au grain session (canal `shots`,
	// document seulement — .ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md §2 bis) :
	// ce compte est donc servi seul, sans issues, et le dire est le contrat.
	PadPickups float64 `json:"pad_pickups"`
}

// EquipmentUsageParties — les CINQ comptes d'un donut (décisions P10/P11). Les
// quatre parts sont EXCLUSIVES et leur somme vaut exactement LobbyTotal : c'est ce
// qui permet au front de lire l'emboîtement par contiguïté (moi -> mes amis -> le
// reste de mon équipe -> eux) et d'écrire les deux sous-totaux « mon escouade »
// (moi + mes amis) et « mon équipe » (+ le reste de mon équipe).
//
// SCOPE : les matchs mesurés à camp CONNU, numérateurs ET dénominateurs — la règle
// de scope de sessionusage.computeMetric, sans exception. Hors d'un match à camp
// connu on ne sait dire ni « mon camp » ni « eux », et mêler les scopes ferait un
// donut dont les parts ne feraient pas le tout. Ce scope est donc PLUS ÉTROIT que
// celui des lignes Families et Players : c'est voulu, et c'est la seule façon
// d'avoir des parts qui ferment.
type EquipmentUsageParties struct {
	LobbyTotal float64 `json:"lobby_total"`
	Player     float64 `json:"player"`
	// Friends : le cumul des coéquipiers suivis (0 quand il n'y en a aucun — un
	// compte, pas une absence de mesure : le lobby a bien été compté).
	Friends float64 `json:"friends"`
	// RestOfTeam : mon camp MOINS moi MOINS mes amis suivis.
	RestOfTeam float64 `json:"rest_of_team"`
	// Opponents : le lobby MOINS mon camp. L'adversaire reste AGRÉGÉ et non nommé.
	Opponents float64 `json:"opponents"`
	// ByFriend : la ventilation de Friends, alignée sur TrackedPlayers par XUID —
	// un arc par ami sur le donut de l'Escouade (jetons squad-player-2..4).
	ByFriend []EquipmentUsageFriendCount `json:"by_friend,omitempty"`
}

// EquipmentUsageFriendCount — le compte d'UN coéquipier suivi dans un donut.
type EquipmentUsageFriendCount struct {
	XUID  string  `json:"xuid"`
	Value float64 `json:"value"`
}
