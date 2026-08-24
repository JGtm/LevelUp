// Package domain — match_assist_pairs.go : QUI EST L'ASSISTANT DE QUI.
//
// Le graphe des assistances lit la MÊME LIGNE de `match_kill_events_latest` que le kill
// feed, mais il en tire un agrégat au lieu d'une décoration : une paire
// (ASSISTANT -> TUEUR ASSISTÉ), comptée sur tout le match.
//
// ─── LES TROIS ÉTATS DE L'ASSISTANCE, ET CE QUE CE BLOC EN FAIT ───────────────────────
//
// La doctrine de la table (internal/migration/steps_shared_kill_events.go) tient en trois
// états qu'on ne confond jamais :
//
//	assist_known = FALSE                           ON NE SAIT PAS
//	assist_known = TRUE  + assist_gamertag NULL    MESURÉ : pas d'assistant
//	assist_known = TRUE  + assist_gamertag nommé   l'assistant
//
// Une liste de paires ne sait rendre que le troisième état. Les deux premiers se
// distinguent donc PAR UN COMPTEUR À CÔTÉ, pas par la longueur de la liste :
// [MatchAssistPairs.MeasuredDeaths] compte les morts du match dont l'assistance EST
// mesurée et publiable ligne à ligne. Sans lui, « aucune paire » serait ambigu et
// l'écran écrirait « aucune assistance » là où la mesure dit « on ne sait pas » — le
// mensonge exact que la table est construite pour empêcher.
//
// ─── ÉLIMINATIONS VOLÉES ──────────────────────────────────────────────────────────────
//
// `assist_damage_pct > killer_damage_pct` : l'assistant a infligé PLUS de dégâts que
// celui que le jeu crédite du kill. C'est une comparaison de deux quantités mesurées, pas
// un jugement : on ne réattribue rien, on compte.
//
// ⚠ CES DEUX PARTS NE SONT PAS BORNÉES À 100 (mesures jusqu'à 228) et le chemin de donnée
// qui les produit n'est pas démontré. On ne les plafonne NI ici NI à l'affichage — un
// plafond jetterait la population qui contredit l'interprétation. La comparaison, elle,
// reste valide quel que soit le plafond : elle porte sur l'ordre, pas sur l'échelle.
package domain

// MatchAssistPairRaw : une paire (assistant, tueur assisté) telle que Q21d la rend, avant
// résolution du gamertag du tueur.
//
// Le gamertag de l'ASSISTANT vient de la table (le film le nomme) ; celui du TUEUR n'y est
// pas lu — il se résout depuis le scoreboard, comme dans buildKillerVictimPairs.
type MatchAssistPairRaw struct {
	AssistXUID     string
	AssistGamertag string
	KillerXUID     string
	// AssistCount : nombre de morts sur lesquelles cet assistant a assisté ce tueur.
	AssistCount int
	// StolenCount : sous-ensemble de AssistCount où `assist_damage_pct > killer_damage_pct`.
	StolenCount int
}

// MatchAssistScopeRaw : la PORTÉE de la lecture d'assistance d'un match — les deux
// dénominateurs sans lesquels une liste de paires vide est illisible.
//
//	MatchDeaths     lignes du match dans `match_kill_events_latest`, toutes portées
//	                confondues. ZÉRO = le match n'est jamais passé au décodeur de film
//	                (ou le titre n'a pas de décodeur) : il n'y a rien à dire, et le
//	                service n'émet alors AUCUN bloc.
//	MeasuredDeaths  lignes `publishable AND assist_known` : les morts dont l'assistance
//	                est mesurée ET publiable ligne à ligne. ZÉRO avec MatchDeaths > 0 =
//	                « non mesuré pour ce match » — un état affiché, pas un vide.
type MatchAssistScopeRaw struct {
	MatchDeaths    int
	MeasuredDeaths int
}

// MatchAssistPair : une paire (ASSISTANT -> TUEUR ASSISTÉ) publiée pour un match.
//
// Le sens de lecture est celui de la décision produit : l'unité est l'ASSISTANT, et les
// tueurs qu'il a servis en sont les segments. C'est l'inverse de MatchKillerVictimPair,
// dont l'unité est le tueur — les deux graphes se lisent côte à côte sans se confondre.
//
// KillerGamertag peut être VIDE : le tueur est résolu depuis le scoreboard, et un tueur
// qui n'y figure pas garde son xuid sans nom inventé (même règle que buildKillerVictimPairs,
// sans le repli « afficher le xuid » — le front décide de l'affichage).
type MatchAssistPair struct {
	AssistXUID     string `json:"assist_xuid"`
	AssistGamertag string `json:"assist_gamertag"`
	KillerXUID     string `json:"killer_xuid"`
	KillerGamertag string `json:"killer_gamertag,omitempty"`
	// AssistCount : morts sur lesquelles cet assistant a assisté ce tueur.
	AssistCount int `json:"assist_count"`
	// StolenCount : sous-ensemble d'AssistCount où la part de dégâts de l'assistant
	// DÉPASSE celle du tueur crédité. « Éliminations volées » à l'écran — un décompte,
	// pas une réattribution : le crédit du jeu n'est jamais réécrit.
	StolenCount int `json:"stolen_count"`
}

// MatchAssistPairs : le bloc « assistances » de l'onglet Combat, avec la PORTÉE de sa
// mesure.
//
// TROIS ÉTATS À L'ÉCRAN, et il faut les trois champs pour les tenir :
//
//	bloc ABSENT (nil)                le match n'a aucune ligne de film — ou le titre n'a
//	                                 pas de décodeur. Rien à dire : l'UI ne rend rien.
//	MeasuredDeaths == 0              le film est là, l'assistance n'y est pas mesurée (ou
//	                                 la passe n'est pas publiable ligne à ligne).
//	                                 « Assistance non mesurée pour ce match ».
//	MeasuredDeaths > 0, Pairs vide   MESURÉ : personne n'a assisté personne.
//	                                 « Aucune assistance ».
//
// Écrire « aucune assistance » sur le deuxième cas serait fabriquer un fait jamais
// observé — la faute que la doctrine des trois états existe pour empêcher.
type MatchAssistPairs struct {
	// MeasuredDeaths : morts du match dont l'assistance est mesurée ET publiable ligne
	// à ligne. C'est le DÉNOMINATEUR de la couverture affichée.
	MeasuredDeaths int `json:"measured_deaths"`
	// Pairs : les paires nommées, triées par AssistCount décroissant.
	Pairs []MatchAssistPair `json:"pairs"`
}
