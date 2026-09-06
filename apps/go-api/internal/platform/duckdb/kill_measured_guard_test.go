package duckdb

// kill_measured_guard_test.go — LA JOINTURE « MORT MESURÉE » NE S'ÉCRIT QU'À UN ENDROIT.
//
// POURQUOI CE GARDE-RAIL. La jointure `match_kill_events_latest × <positions>_latest` porte
// DEUX gardes qu'aucune relecture ne rattrape si elles manquent : `publishable` (une passe
// non publiable est juste en agrégat et FAUSSE ligne à ligne) et l'unanimité
// `HAVING count(DISTINCT source_tag) = 1` (accrocher une position à la mauvaise arme est
// indétectable à l'écran). Une copie qui en oublierait une continuerait de rendre des
// nombres plausibles. La règle n°6 du dépôt s'applique donc à la lettre : le lot 3 a
// centralisé la jointure dans kill_measured.go, migré le POC dessus, et pose ici l'interdit
// — sans quoi la dette re-croît (leçon chiffrée du dépôt : prédicat bot passé de 8 à
// 36 copies APRÈS centralisation).
//
// CE QU'IL COUVRE : les .go non-test de `platform/duckdb` et de son sous-paquet `halo5` —
// la couche de lecture, seul endroit d'où une copie pourrait venir.
//
// HORS MOTIF, ET DOCUMENTÉ ICI PLUTÔT QUE SUBI :
//
//   - `Q21bKillSources` (queries_match.go) ne joint AUCUNE table de positions : c'est le
//     kill-feed seul (arme et catégorie de la mort affichées dans la fiche de match). Elle
//     partage les gardes `publishable`/unanimité — d'où elles viennent, historiquement —
//     mais pas la jointure. Le motif ci-dessous vise la JOINTURE, elle n'est donc pas
//     concernée, et c'est voulu : la fusionner ferait dépendre le kill-feed de la présence
//     des positions, ce que Q21b existe précisément pour éviter.
//   - `halo5/halo5_match_events_source.go` porte un `LEFT JOIN kill_positions_latest` qui
//     ÉNUMÈRE les events d'un match pour la timeline Halo 5 (positions natives, une ligne
//     par event, aucune mesure de distance, aucune garde d'unanimité à poser puisque rien
//     n'est attribué à une arme). Ce n'est pas la même lecture ; l'exception est nominative
//     et datée (2026-09-06), pas un joker de répertoire.

import (
	"regexp"
	"strings"
	"testing"
)

// jointureMesuree matche une jointure sur l'une des deux tables de positions, quelle que
// soit sa forme (JOIN, LEFT JOIN, INNER JOIN — le mot-clé JOIN est toujours le dernier).
var jointureMesuree = regexp.MustCompile(`(?i)\bJOIN\s+kill_(positions|openings)_latest\b`)

// proprietaireJointure : le seul fichier qui a le droit de composer la jointure mesurée.
const proprietaireJointure = "kill_measured.go"

// exceptionsJointure : chemins (relatifs au paquet) autorisés à porter le motif, avec leur
// motif d'exception. Toute entrée nouvelle s'écrit ICI, datée et justifiée.
var exceptionsJointure = map[string]string{
	proprietaireJointure: "propriétaire de la jointure (lot 3, 2026-09-06)",
	"halo5/halo5_match_events_source.go": "énumération des events Halo 5 pour la timeline " +
		"(LEFT JOIN, positions natives, aucune mesure de distance) — 2026-09-06",
}

// TestJointureMesureeUneSeuleFois : aucun fichier hors exceptions ne joint une table de
// positions au kill-feed.
func TestJointureMesureeUneSeuleFois(t *testing.T) {
	for path, src := range sourcesDuPaquet(t) {
		if _, ok := exceptionsJointure[path]; ok {
			continue
		}
		if jointureMesuree.MatchString(src) {
			t.Errorf("%s compose sa propre jointure kills × positions — utiliser "+
				"measuredKillsQuery(...) de %s (règle n°6 : les gardes `publishable` et "+
				"d'unanimité ne se recopient pas)", path, proprietaireJointure)
		}
	}
}

// TestProprietaireJointurePorteLesDeuxTables : le garde-rail ci-dessus serait creux si
// l'helper cessait de nommer les deux tables (renommage, suppression, déplacement dans un
// autre fichier). On vérifie donc qu'il les porte toujours — un garde-rail qui ne garde
// plus rien est pire qu'aucun garde-rail.
func TestProprietaireJointurePorteLesDeuxTables(t *testing.T) {
	src, ok := sourcesDuPaquet(t)[proprietaireJointure]
	if !ok {
		t.Fatalf("%s introuvable — la jointure mesurée a changé de fichier sans son garde-rail",
			proprietaireJointure)
	}
	for _, table := range []string{"kill_positions_latest", "kill_openings_latest"} {
		if !strings.Contains(src, table) {
			t.Errorf("%s ne nomme plus %q : le garde-rail ne vérifie plus rien",
				proprietaireJointure, table)
		}
	}
	// Les deux gardes de la jointure, épinglées par leur texte : leur disparition serait
	// silencieuse (la requête continuerait de rendre des lignes).
	for _, garde := range []string{"e.publishable", "HAVING count(DISTINCT e.source_tag) = 1"} {
		if !strings.Contains(src, garde) {
			t.Errorf("%s a perdu la garde %q — une mesure fausse redevient publiable",
				proprietaireJointure, garde)
		}
	}
}
