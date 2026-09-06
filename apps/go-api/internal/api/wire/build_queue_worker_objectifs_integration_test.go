//go:build integration && cgo

// Package api — build_queue_worker_objectifs_integration_test.go : LES CALQUES D OBJECTIF DE LA
// CUISSON E2E, CONFRONTES A LA FEUILLE DE MATCH.
//
// EXTRAIT de `build_queue_worker_binary_integration_test.go` le 2026-09-06 (revue CTF-R2,
// constat 5) : ce fichier-la passait 562 lignes, au-dessus du seuil de 500 de CLAUDE.md, et sa
// dette etait gelee. Les assertions sont deplacees TELLES QUELLES ; seule la justification du
// « 7 joueurs pontes » est corrigee (constat 3).
package wire

import (
	"sort"
	"testing"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/port"
)

// assertCalquesDObjectif confronte les calques d'objectif à l'ORACLE INDÉPENDANT du fixture — la
// feuille de match de l'API, que le décodeur ne lit jamais.
//
// CE QU'IL ASSERTE, ET POURQUOI CHAQUE LIGNE EST JUSTIFIABLE :
//
//	les CAPTURES        `coverage.flagCarries.captures` compte les captures des DEUX camps ; son
//	                    oracle est donc la SOMME des scores d'équipe, pas le score du gagnant.
//	                    L'égalité avec `TeamScores[0]` seul ne tenait sur ce fixture que parce que
//	                    le perdant a marqué 0 — invariant arithmétiquement faux, corrigé le
//	                    2026-09-06 (revue CTF-R1, constat 4).
//	les COMPTEURS       chaque joueur que le pont d'identité nomme publie ses actions `kills` et
//	                    `assists` ; elles ne peuvent pas dépasser ce que la feuille de match lui
//	                    donne. C'est vrai QUELS QUE SOIENT les joueurs pontés, donc c'est un
//	                    invariant et non une épingle.
//	les VIES DE L'OBJET `objectLives` = 4 : épingle de caractérisation, mesurée identique sur le
//	                    film complet du cache aux schémas 20, 38 et 39. Elle n'est PAS la
//	                    « signature du registre ECS » que la première version de ce commentaire
//	                    annonçait : la revue CTF-R1 a montré qu'elle vaut 4 même avec le fixture
//	                    mal généré (la chaîne E2E pelait ses deux couches). Le seul garde-rail qui
//	                    attrape ce défaut-là est `film_fixture_integrite_cgo_test.go`.
//
//	les ACTIONS `flag`  la dérive INSTRUITE le 2026-09-06. Ce film porte exactement UNE prise et
//	                    UNE capture nommées, toutes deux au slot 22 (SweatyYeti75, 7 frags,
//	                    2 morts). Elles avaient disparu du document entre le schéma 20 et
//	                    aujourd'hui parce que `d173b1a8c` a basculé ce calque sur un pont
//	                    d'identité qui exige TROIS instants de mort — hors de portée d'un joueur
//	                    qui meurt deux fois. `CompletedByLines` les rend. C'est le seul film CTF
//	                    du dépôt qui porte des actions de drapeau nommées : les figer ici est la
//	                    seule protection contre une seconde disparition silencieuse.
//	le PONTAGE          7 des 8 joueurs de la feuille sont nommés, chacun avec EXACTEMENT ses
//	                    compteurs. LE 8e A POURTANT UN SLOT, et la première version de ce
//	                    commentaire disait le contraire (corrigé le 2026-09-06, revue CTF-R2,
//	                    constat 3) : 2535458702376288 occupe le slot 12, dont les compteurs de
//	                    film valent (5 frags, 0 mort, 60 assistances). Ses frags et ses morts
//	                    sont exactement ceux de sa feuille ; c'est le compteur d'assistances,
//	                    lu à 60 contre 0, qui fait refuser le triplet — et le pont par morts ne
//	                    peut évidemment pas voir un joueur qui ne meurt jamais. Un slot dont les
//	                    compteurs agrégés ne ressemblent à aucune ligne reste donc anonyme, et
//	                    c'est la prudence qui joue, pas une absence. Si un lot futur rend ce
//	                    slot nommable, cette attente passera à 8 : ce sera un PROGRÈS, pas une
//	                    régression.
func assertCalquesDObjectif(t *testing.T, doc replaydoc.ReplayDocument, fx filmFixture) {
	t.Helper()
	fc := doc.Coverage.FlagCarries
	if fc == nil {
		t.Fatal("artefact sans couverture du drapeau : le film est un CTF, le calque doit se prononcer")
	}
	if !fc.FlagFilm {
		t.Fatalf("film NON reconnu comme CTF (bursts=%d captures=%d steals=%d)", fc.Bursts, fc.Captures, fc.Steals)
	}
	if fx.Facts.TeamScores == nil {
		t.Fatal("fixture sans scores d'équipe : plus d'oracle pour les captures")
	}
	// ORACLE INDÉPENDANT : les captures des deux camps réunies valent la somme des scores.
	attendu := fx.Facts.TeamScores[0] + fx.Facts.TeamScores[1]
	if fc.Captures != attendu || fc.Bursts != attendu {
		t.Errorf("captures reconstruites = %d (bursts %d), la feuille de match en donne %d "+
			"(%d + %d) — le décodage du film et l'API doivent tomber d'accord",
			fc.Captures, fc.Bursts, attendu, fx.Facts.TeamScores[0], fx.Facts.TeamScores[1])
	}
	// ÉPINGLE DE CARACTÉRISATION (pas un oracle) : les vies libres de l'objet drapeau.
	if fc.ObjectLives != 4 {
		t.Errorf("vies libres de l'objet drapeau = %d, attendu 4 (mesure du 2026-09-06 sur le film "+
			"complet du cache, schémas 20, 38 et 39 confondus)", fc.ObjectLives)
	}
	// ORACLE INDÉPENDANT : aucun joueur ne peut publier plus d'actions que la feuille ne lui en
	// donne, et toute action publiée appartient à un joueur du match.
	feuille := map[string]port.MatchPlayerFact{}
	for _, p := range fx.Facts.Players {
		feuille[p.XUID] = p
	}
	parJoueur := map[string]map[string]int{}
	for _, a := range doc.Objectives {
		if _, connu := feuille[a.XUID]; !connu {
			t.Errorf("action d'objectif `%s` attribuée au xuid %s, absent de la feuille de match", a.Stat, a.XUID)
			continue
		}
		if parJoueur[a.XUID] == nil {
			parJoueur[a.XUID] = map[string]int{}
		}
		parJoueur[a.XUID][a.Stat]++
	}
	for xuid, actions := range parJoueur {
		p := feuille[xuid]
		// ÉGALITÉ, pas inégalité : sur ce film chaque joueur ponté publie EXACTEMENT sa ligne.
		if actions["kills"] != p.Kills || actions["assists"] != p.Assists {
			t.Errorf("joueur %s : %d frags / %d assistances publiés, la feuille de match dit %d / %d",
				xuid, actions["kills"], actions["assists"], p.Kills, p.Assists)
		}
	}
	// LE PONTAGE : 7 des 8 joueurs de la feuille, pas 5. Le 8e occupe le slot 12, dont les compteurs
	// agreges (5, 0, 60 assistances contre 0 a la feuille) ne ressemblent a aucune ligne.
	if len(parJoueur) != 7 {
		t.Errorf("joueurs pontés = %d, attendu 7 — un pont qui en perd deux perd avec eux leurs "+
			"actions d'objectif (régression `d173b1a8c`, corrigée le 2026-09-06). Pontés : %v",
			len(parJoueur), clesTriees(parJoueur))
	}
	// LES ACTIONS `flag` : la régression instruite. Elles vivent toutes deux sur le slot du
	// joueur qui meurt DEUX fois — celui que le pont par instants de mort ne peut pas nommer seul.
	const porteurDeDrapeau = "2535463878425995" // SweatyYeti75, 7 frags / 2 morts / 1 assistance
	for stat, attendu := range map[string]int{"flag_captures": 1, "flag_steals": 1} {
		if got := parJoueur[porteurDeDrapeau][stat]; got != attendu {
			t.Errorf("action `%s` du porteur de drapeau %s : %d, attendu %d — c'est exactement ce "+
				"que l'artefact du parc au schéma 20 publiait, et ce que le pont par morts seul perdait",
				stat, porteurDeDrapeau, got, attendu)
		}
	}
	t.Logf("calque objectifs : %d actions, %d joueurs pontés", len(doc.Objectives), len(parJoueur))
}

// clesTriees rend les clés d'une table, triées — pour un message d'échec reproductible.
func clesTriees[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
