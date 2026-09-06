//go:build integration && cgo

// Package api — build_queue_worker_valeurs_integration_test.go : CE QUE VAUT LE DOCUMENT CUIT
// PAR L'OUVRIER, ET PAS SEULEMENT SA FORME.
//
// # LE TROU QUE CE FICHIER FERME (registre d'audit 2026-09-05, constat G1)
//
// La preuve voisine (`build_queue_worker_binary_integration_test.go`) fait décoder un VRAI film
// par le VRAI binaire de l'ouvrier, en CI. Elle ne vérifiait jusqu'ici que la FORME du document
// rangé : schéma courant, trajectoires non vides, courbe de score non vide. Une régression qui
// décale l'origine de l'horloge, qui divise par dix le nombre de trajectoires, qui perd un
// joueur du pont d'identité ou qui inverse un camp passait VERTE.
//
// # CE QUE L'ORACLE DE L'API PROUVE, ET CE QU'IL NE PROUVE PAS
//
// Le fixture porte son propre ORACLE : `facts.players[]` (frags, morts, assistances par xuid) et
// `facts.teamScores` viennent du SERVICE HALO, pas du film.
//
// UNE SEULE DES DEUX CONFRONTATIONS EST UN DIFFÉRENTIEL DE DEUX CHAÎNES INDÉPENDANTES, et la
// distinction a été payée par la revue adversariale du 2026-09-06 (F-R1-1) :
//
//   - LE ROSTER, OUI. `assertRoster` vérifie que le film ne NOMME que les joueurs auxquels
//     l'API donne au moins une mort. Les noms de vies sont lus du fil des morts du chunk
//     highlight (`analysis/replay/deaths_source.go` : « aucune base n'intervient ») ; ils ne
//     passent pas par `facts`. Les deux chaînes sont bien indépendantes, et l'égalité des deux
//     ensembles est un fait, pas une définition.
//   - LES COMPTEURS DE JOUEUR, NON. Le pont d'identité apparie un slot d'entité à un xuid par
//     ÉGALITÉ EXACTE du triplet frags/morts/assistances contre la ligne de l'API
//     (`objectiveevents/slotidentity.go` : `l.Kills == kills[slot] && l.Deaths == ... && ...`).
//     Tout joueur publié dans `ScoreTimeline.Players` porte donc, PAR CONSTRUCTION, le triplet
//     de l'API. Une régression du décodeur ne produit PAS un écart de valeur : elle fait
//     DISPARAÎTRE le joueur du calque, parce que son triplet ne désigne plus une ligne unique.
//     La première version de cet en-tête annonçait « 15 compteurs sur 15 exactement égaux à
//     ceux de l'API » comme une confrontation : c'était une propriété que le pont IMPOSE.
//
// D'OÙ LE VRAI DÉTECTEUR DE RÉGRESSION DU PONT : la LISTE FIGÉE des 5 joueurs appariés. Les 3
// joueurs restants ne sont pas publiés du tout (le pont refuse plutôt que de deviner) ; un 4e
// refus, comme un 6e appariement, fait rougir. Ce que la comparaison de valeurs garde en plus
// est décrit sur `assertCompteursJoueurs` : elle est INTERNE à la chaîne du film.
//
// # CE QUI EST FIGÉ COMME MESURE, ET POURQUOI C'EST LÉGITIME
//
// L'origine de l'horloge, le coup d'envoi, la grille (781 frames à 100 ms), le nombre de vies et
// leur répartition par joueur n'ont AUCUN oracle extérieur : ce sont des mesures du film. Elles
// sont recopiées à la main ci-dessous, telles que mesurées le 2026-09-05 sur ce fixture. Leur
// rôle n'est pas de prouver qu'elles sont justes — c'est de faire ÉCHOUER tout changement qui
// les déplace sans que personne ne l'ait voulu. Un changement voulu se solde par une mise à jour
// de ces constantes DANS LE MÊME COMMIT que le changement de décodeur.
package wire

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/port"
)

// oracleJoueur est une ligne de `facts.players[]` du fixture, RECOPIÉE À LA MAIN.
type oracleJoueur struct {
	XUID    string
	Kills   int
	Deaths  int
	Assists int
	TeamID  int
}

// oracleAPI : la feuille de match du SERVICE HALO, recopiée à la main de
// `testdata/film_e2e/c0a82e88/fixture.json` (`facts.players[]`).
//
// ELLE N'EST UN ORACLE INDÉPENDANT QUE POUR LE ROSTER ET LE SCORE DE CAMP. Les triplets
// frags/morts/assistances, eux, servent de CLÉ au pont d'identité (`slotidentity.go`) : un
// joueur publié les porte par construction — cf. l'en-tête et `assertCompteursJoueurs`.
//
// LA RECOPIE EST VOLONTAIREMENT REDONDANTE avec le fichier : `assertOracleFideleAuFixture`
// confronte les deux à chaque exécution, de sorte qu'une retouche du fixture ne puisse pas
// déplacer l'attendu en silence. Sans cette confrontation, lire l'oracle du fichier au moment du
// test rendrait l'assertion auto-validante côté oracle.
var oracleAPI = []oracleJoueur{
	{XUID: "2533274823110022", Kills: 2, Deaths: 3, Assists: 1, TeamID: 1},
	{XUID: "2533275001554469", Kills: 1, Deaths: 4, Assists: 2, TeamID: 1},
	{XUID: "2535429692041611", Kills: 1, Deaths: 3, Assists: 0, TeamID: 1},
	{XUID: "2535432531943478", Kills: 2, Deaths: 4, Assists: 0, TeamID: 1},
	{XUID: "2535458702376288", Kills: 5, Deaths: 0, Assists: 0, TeamID: 0},
	{XUID: "2535463878425995", Kills: 7, Deaths: 2, Assists: 1, TeamID: 0},
	{XUID: "2535465632069522", Kills: 0, Deaths: 1, Assists: 1, TeamID: 0},
	{XUID: "2535465779546251", Kills: 2, Deaths: 3, Assists: 1, TeamID: 0},
}

// oracleScoresCamps : `facts.teamScores` du fixture — 3 captures à 0.
var oracleScoresCamps = [2]int{3, 0}

// Mesures du film, relevées le 2026-09-05 sur ce fixture (cf. en-tête).
const (
	valeurFrameCount   = 781
	valeurIntervalleMS = 100
	valeurDureeMS      = 78100
	valeurNbTracks     = 22
	valeurNbAnonymes   = 2
	// valeurOriginMS : l'instant de la frame 0 sur l'horloge du fil des éliminations.
	// C'EST LA VALEUR QUE TOUT LE RECALAGE CLIENT SOUSTRAIT (`replayMs = event_time_ms +
	// t0_ms − originMs`) : la décaler d'une seule milliseconde décale tout le rejeu.
	valeurOriginMS int64 = 34870
	// valeurT0FilmMS : le coup d'envoi mesuré dans le film, sur la même horloge — 300 ms
	// (3 frames) après l'origine sur ce match.
	valeurT0FilmMS int64 = 35170
)

// valeurViesParXUID : combien de VIES NOMMÉES le film attribue à chaque joueur.
//
// CE N'EST PAS LE NOMBRE DE MORTS DE L'API, et il ne faut pas l'y ramener : une vie n'est nommée
// que par la mort qui la ferme (`lives.go`), donc une vie ouverte avant le début de la grille ou
// close par la fin de partie reste anonyme. Sur ce film : 4 joueurs sur 7 tombent sur le compte
// de morts de l'API, 3 non (2 en plus, 1 en moins). La mesure est figée telle quelle.
var valeurViesParXUID = map[string]int{
	"2533274823110022": 3,
	"2533275001554469": 4,
	"2535429692041611": 3,
	"2535432531943478": 2,
	"2535463878425995": 3,
	"2535465632069522": 2,
	"2535465779546251": 3,
}

// valeurJoueursApparies : les xuids que le pont d'identité rattache à un slot d'entité, donc les
// seuls dont le document publie des compteurs vivants.
var valeurJoueursApparies = []string{
	"2533275001554469",
	"2535429692041611",
	"2535432531943478",
	"2535463878425995",
	"2535465632069522",
}

// valeurCourbeCamp : la SEULE courbe d'équipe publiée sur ce film, frame par frame — les trois
// captures. Son dernier point vaut le score du vainqueur à l'API (3).
var valeurCourbeCamp = []replay.ScoreTick{{T: 195, V: 1}, {T: 485, V: 2}, {T: 706, V: 3}}

// valeurObjectifsParStat : le calque des actions d'objectif, par nom de statistique.
//
// LA MESURE CONTREDIT L'EN-TÊTE HISTORIQUE de la preuve voisine (« 92 actions d'objectif nommées
// (famille flag) », écrit le 2026-08-25 au schéma 37) : au schéma 39, ce film CTF rend 12 actions
// et AUCUNE de la famille drapeau. L'écart est consigné en découverte du lot F (registre
// d'audit : le calque des objectifs relève des lots A et E) ; il est figé ici pour que le
// prochain déplacement se voie.
var valeurObjectifsParStat = map[string]int{"kills": 8, "assists": 4}

// assertValeursDuDocument confronte le document cuit par l'ouvrier à l'oracle de l'API et aux
// mesures figées. Appelée par `assertArtefactLivreEtComplet` sur le document que le service de
// rejeu vient de relire — jamais sur une re-cuisson.
func assertValeursDuDocument(t *testing.T, doc replay.ReplayDocument, fx filmFixture) {
	t.Helper()
	assertOracleFideleAuFixture(t, fx)
	assertHorlogeEtGrille(t, doc)
	assertRoster(t, doc)
	assertCompteursJoueurs(t, doc)
	assertScoreCamp(t, doc)
	assertObjectifs(t, doc)
}

// assertOracleFideleAuFixture prouve que la table recopiée ci-dessus EST celle du fichier : sans
// cela, une retouche du fixture déplacerait l'attendu sans qu'aucun test ne le dise.
func assertOracleFideleAuFixture(t *testing.T, fx filmFixture) {
	t.Helper()
	if len(fx.Facts.Players) != len(oracleAPI) {
		t.Fatalf("fixture.json porte %d joueurs, l'oracle recopié en porte %d — recopie périmée",
			len(fx.Facts.Players), len(oracleAPI))
	}
	duFichier := map[string]port.MatchPlayerFact{}
	for _, p := range fx.Facts.Players {
		duFichier[p.XUID] = p
	}
	for _, o := range oracleAPI {
		p, ok := duFichier[o.XUID]
		if !ok {
			t.Fatalf("xuid %s recopié dans l'oracle mais absent de fixture.json", o.XUID)
		}
		if p.Kills != o.Kills || p.Deaths != o.Deaths || p.Assists != o.Assists || p.TeamID != o.TeamID {
			t.Fatalf("oracle recopié périmé pour %s : fixture.json dit %d/%d/%d camp %d, la recopie dit %d/%d/%d camp %d",
				o.XUID, p.Kills, p.Deaths, p.Assists, p.TeamID, o.Kills, o.Deaths, o.Assists, o.TeamID)
		}
	}
	if fx.Facts.TeamScores == nil || *fx.Facts.TeamScores != oracleScoresCamps {
		t.Fatalf("scores de camp du fixture %v ≠ oracle recopié %v", fx.Facts.TeamScores, oracleScoresCamps)
	}
}

// assertHorlogeEtGrille fige l'axe de temps du document : c'est lui que tout le client soustrait.
func assertHorlogeEtGrille(t *testing.T, doc replay.ReplayDocument) {
	t.Helper()
	if doc.MatchID != "c0a82e88-7b3b-419c-a984-13385af99259" {
		t.Errorf("matchId servi %q, attendu celui du fixture", doc.MatchID)
	}
	if doc.FrameCount != valeurFrameCount || doc.FrameIntervalMS != valeurIntervalleMS || doc.DurationMS != valeurDureeMS {
		t.Errorf("grille : %d frames à %d ms pour %d ms, attendu %d/%d/%d",
			doc.FrameCount, doc.FrameIntervalMS, doc.DurationMS, valeurFrameCount, valeurIntervalleMS, valeurDureeMS)
	}
	if doc.OriginMs == nil {
		t.Errorf("originMs ABSENT : l'origine n'est plus établie sur ce film (attendu %d ms)", valeurOriginMS)
	} else if *doc.OriginMs != valeurOriginMS {
		t.Errorf("originMs = %d ms, attendu %d ms — TOUT le recalage du fil des éliminations est décalé d'autant",
			*doc.OriginMs, valeurOriginMS)
	}
	if doc.T0FilmMs == nil {
		t.Errorf("t0FilmMs ABSENT : le coup d'envoi n'est plus mesuré (attendu %d ms)", valeurT0FilmMS)
	} else if *doc.T0FilmMs != valeurT0FilmMS {
		t.Errorf("t0FilmMs = %d ms, attendu %d ms", *doc.T0FilmMs, valeurT0FilmMS)
	}
}

// assertRoster fige les vies publiées ET les confronte à l'oracle : le film ne nomme QUE les
// joueurs que l'API donne morts au moins une fois — le seul joueur à 0 mort (2535458702376288)
// est aussi le seul absent du roster nommé. C'est un fait des deux chaînes, pas une tautologie.
func assertRoster(t *testing.T, doc replay.ReplayDocument) {
	t.Helper()
	if len(doc.Tracks) != valeurNbTracks {
		t.Errorf("%d trajectoires, attendu %d", len(doc.Tracks), valeurNbTracks)
	}
	vies := map[string]int{}
	anonymes := 0
	for _, tr := range doc.Tracks {
		if tr.XUID == "" {
			anonymes++
			continue
		}
		vies[tr.XUID]++
	}
	if anonymes != valeurNbAnonymes {
		t.Errorf("%d vies anonymes, attendu %d", anonymes, valeurNbAnonymes)
	}
	for xuid, attendu := range valeurViesParXUID {
		if vies[xuid] != attendu {
			t.Errorf("roster : %d vies nommées pour %s, attendu %d", vies[xuid], xuid, attendu)
		}
	}
	for xuid := range vies {
		if _, ok := valeurViesParXUID[xuid]; !ok {
			t.Errorf("roster : xuid %s nommé par le film mais absent des mesures figées", xuid)
		}
	}
	for _, o := range oracleAPI {
		_, nomme := vies[o.XUID]
		if (o.Deaths > 0) != nomme {
			t.Errorf("roster ↔ oracle : %s a %d mort(s) à l'API et %s du roster nommé du film",
				o.XUID, o.Deaths, map[bool]string{true: "fait partie", false: "est absent"}[nomme])
		}
	}
}

// assertCompteursJoueurs garde DEUX choses, et aucune des deux n'est un différentiel film ↔ API
// (cf. l'en-tête : le pont d'identité impose l'égalité des triplets).
//
//  1. LA LISTE DES JOUEURS APPARIÉS, figée. C'est le détecteur de régression du pont : un
//     décodage qui change les compteurs d'un joueur ne décale pas sa valeur, il le fait
//     DISPARAÎTRE du calque (son triplet ne désigne plus une ligne de match unique). Mesuré le
//     2026-09-06 : décaler d'un frag le film rend « 4 joueurs publiés, attendu 5 ».
//
//  2. LA COHÉRENCE INTERNE À LA CHAÎNE DU FILM entre les DEUX dérivations du même compteur :
//     la clé d'appariement est le NOMBRE D'INCRÉMENTS lus des enregistrements
//     (`objectiveevents.countsOf` → `len(incrementTimes(...))`), tandis que la valeur publiée
//     est la DERNIÈRE de la série posée sur la grille de frames (`replay.scoreTicksOf`, qui
//     écarte les émissions hors fenêtre et aplatit les paliers). Rien n'oblige ces deux
//     dérivations à coïncider : un point de score perdu par la fenêtre, une origine décalée ou
//     un palier mal filtré les sépare. Le triplet de l'API sert ici de VALEUR ATTENDUE COMMUNE
//     — pas d'oracle indépendant, mais la seule écriture qui les confronte l'une à l'autre.
//     Mesuré le 2026-09-05 : les deux dérivations coïncident sur les 15 compteurs.
func assertCompteursJoueurs(t *testing.T, doc replay.ReplayDocument) {
	t.Helper()
	if doc.ScoreTimeline == nil {
		t.Fatal("aucun calque de score : ni la liste des appariés ni les séries publiées n'existent")
	}
	publies := map[string]replay.PlayerScore{}
	for _, p := range doc.ScoreTimeline.Players {
		publies[p.XUID] = p
	}
	if len(publies) != len(valeurJoueursApparies) {
		t.Errorf("%d joueurs publiés, attendu %d (%v)", len(publies), len(valeurJoueursApparies), triXUIDs(publies))
	}
	for _, xuid := range valeurJoueursApparies {
		if _, ok := publies[xuid]; !ok {
			t.Errorf("joueur %s apparié le 2026-09-05 mais plus publié : le pont d'identité a régressé", xuid)
		}
	}
	for _, o := range oracleAPI {
		p, ok := publies[o.XUID]
		if !ok {
			continue // refus du pont : couvert par la liste figée ci-dessus
		}
		k, d, a := valeurFinale(p.Kills), valeurFinale(p.Deaths), valeurFinale(p.Assists)
		if k != o.Kills || d != o.Deaths || a != o.Assists {
			t.Errorf("SÉRIE PUBLIÉE ≠ CLÉ D'APPARIEMENT pour %s : la série finit à %d frags / %d morts / "+
				"%d assistances, alors que le pont l'a apparié sur %d / %d / %d incréments — un point de "+
				"score est perdu par la grille de frames, ou un palier est mal filtré",
				o.XUID, k, d, a, o.Kills, o.Deaths, o.Assists)
		}
	}
}

// assertScoreCamp confronte la courbe d'équipe au score final de l'API : sur ce film, un SEUL
// slot d'équipe est rattaché (`teamIdentity` = `unresolved`, propriété de ce Husky Raid), et sa
// courbe monte 1, 2, 3 — les trois captures que l'API donne au vainqueur.
func assertScoreCamp(t *testing.T, doc replay.ReplayDocument) {
	t.Helper()
	if doc.ScoreTimeline == nil {
		return // déjà signalé par assertCompteursJoueurs
	}
	if len(doc.ScoreTimeline.Teams) != 1 {
		t.Fatalf("%d courbes d'équipe, attendu 1 (le film ne rattache qu'un slot sur ce mode)",
			len(doc.ScoreTimeline.Teams))
	}
	camp := doc.ScoreTimeline.Teams[0]
	if camp.TeamID != nil {
		t.Errorf("la courbe porte un camp (%d) alors que l'identité est %q : le document ne doit pas deviner",
			*camp.TeamID, doc.Coverage.Score.TeamIdentity)
	}
	if len(camp.Total) != len(valeurCourbeCamp) {
		t.Fatalf("courbe d'équipe : %d points, attendu %d (%v)", len(camp.Total), len(valeurCourbeCamp), camp.Total)
	}
	for i, attendu := range valeurCourbeCamp {
		if camp.Total[i] != attendu {
			t.Errorf("courbe d'équipe, point %d : frame %d valeur %d, attendu frame %d valeur %d",
				i, camp.Total[i].T, camp.Total[i].V, attendu.T, attendu.V)
		}
	}
	final := camp.Total[len(camp.Total)-1].V
	if vainqueur := max(oracleScoresCamps[0], oracleScoresCamps[1]); final != vainqueur {
		t.Errorf("SCORE FINAL film ↔ API : la courbe finit à %d, l'API donne %d au vainqueur", final, vainqueur)
	}
}

// assertObjectifs fige le calque des actions d'objectif (cf. valeurObjectifsParStat).
func assertObjectifs(t *testing.T, doc replay.ReplayDocument) {
	t.Helper()
	parStat := map[string]int{}
	for _, o := range doc.Objectives {
		parStat[o.Stat]++
	}
	for stat, attendu := range valeurObjectifsParStat {
		if parStat[stat] != attendu {
			t.Errorf("actions d'objectif `%s` : %d, attendu %d", stat, parStat[stat], attendu)
		}
	}
	for stat, n := range parStat {
		if _, ok := valeurObjectifsParStat[stat]; !ok {
			t.Errorf("famille d'actions d'objectif `%s` (%d actions) apparue depuis la mesure du 2026-09-05", stat, n)
		}
	}
}

// valeurFinale rend le dernier total d'un compteur — ZÉRO quand la série est absente, ce qui est
// la façon dont le document représente un compteur resté à zéro.
func valeurFinale(s replay.ScoreSeries) int {
	if len(s.Total) == 0 {
		return 0
	}
	return s.Total[len(s.Total)-1].V
}

// triXUIDs rend les xuids publiés, triés — pour un message d'échec lisible.
func triXUIDs(m map[string]replay.PlayerScore) []string {
	out := make([]string, 0, len(m))
	for x := range m {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
