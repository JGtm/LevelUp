package duckdb

// kill_measured_guard_test.go — LA JOINTURE « MORT MESURÉE » NE S'ÉCRIT QU'À UN ENDROIT.
//
// POURQUOI CE GARDE-RAIL. La jointure `match_kill_events_latest × <positions>_latest` porte
// TROIS gardes qu'aucune relecture ne rattrape si elles manquent : `publishable` (une passe
// non publiable est juste en agrégat et FAUSSE ligne à ligne), l'unanimité
// `HAVING count(DISTINCT source_tag) = 1` (accrocher une position à la mauvaise arme est
// indétectable à l'écran) et l'unicité du frag `count(*) = 1` jugée AVANT tout filtre de côté
// (une ligne de positions ne dit pas laquelle de deux victimes simultanées elle place). Une
// copie qui en oublierait une continuerait de rendre des nombres plausibles. La règle n°6 du
// dépôt s'applique donc à la lettre : le lot 3 a centralisé la jointure dans kill_measured.go,
// migré le POC dessus, et pose ici l'interdit — sans quoi la dette re-croît (leçon chiffrée du
// dépôt : prédicat bot passé de 8 à 36 copies APRÈS centralisation).
//
// # CE QUI EST INTERDIT A CHANGÉ LE 2026-09-06 (revue adversariale, constat C5a)
//
// La première version cherchait le littéral `JOIN kill_positions_latest`. Elle ne captait NI
// une jointure écrite `FROM kill_positions_latest kp JOIN match_kill_events_latest e`, NI une
// composée par `fmt.Sprintf` (le nom de table vient d'une constante, le mot `JOIN` d'un
// template), NI une jointure posée sur les TABLES BRUTES `kill_positions` / `kill_openings`
// — laquelle serait pire que tout, puisqu'elle servirait les passes de décodage périmées
// (ADR 0026 : lecture par la vue `_latest` UNIQUEMENT).
//
// L'interdit porte donc désormais sur les NOMS eux-mêmes, tables brutes comprises :
// `kill_positions`, `kill_openings`, `kill_positions_latest`, `kill_openings_latest`. Un
// lecteur qui a besoin de nommer l'une des deux passe par les constantes du propriétaire
// (`positionsAtKill` / `positionsAtOpening`) — c'est ce que fait `KillDistanceRepo` pour son
// journal de dégradation.
//
// CE QU'IL COUVRE : les .go non-test de `platform/duckdb` et de son sous-paquet `halo5` — la
// couche de LECTURE, seul endroit d'où une copie pourrait venir. Les migrations (qui CRÉENT
// les tables) et les persisters (qui les ÉCRIVENT) vivent dans d'autres paquets et ont
// évidemment le droit de les nommer ; ce garde-rail ne les voit pas, et c'est voulu.
//
// LES COMMENTAIRES SONT RETIRÉS AVANT LA RECHERCHE : la doc d'un fichier a le droit de dire
// de quelle table il parle. C'est le CODE qui est contraint.
//
// HORS MOTIF, ET DOCUMENTÉ ICI PLUTÔT QUE SUBI :
//
//   - `Q21bKillSources` (queries_match.go) ne joint AUCUNE table de positions : c'est le
//     kill-feed seul (arme et catégorie de la mort affichées dans la fiche de match). Elle
//     partage les gardes `publishable`/unanimité — d'où elles viennent, historiquement —
//     mais ne nomme aucune table de positions, elle n'est donc pas concernée. La fusionner
//     ferait dépendre le kill-feed de la présence des positions, ce que Q21b existe
//     précisément pour éviter.
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

// tablesDePositions matche le NOM d'une des deux tables de positions ou de leurs vues —
// jamais un mot-clé SQL, dont aucune forme n'est fiable (cf. l'en-tête).
var tablesDePositions = regexp.MustCompile(`\bkill_(positions|openings)(_latest)?\b`)

// lignesDeCommentaire : une ligne dont le premier caractère non blanc ouvre un commentaire.
// Volontairement naïve et volontairement CONSERVATRICE : elle ne retire que des lignes
// ENTIÈRES, jamais un fragment — un `//` à l'intérieur d'une chaîne reste donc du code aux
// yeux du garde-rail, ce qui ne peut que le rendre plus strict, jamais plus laxiste.
var lignesDeCommentaire = regexp.MustCompile(`(?m)^[ \t]*//.*$`)

// codeSeul rend la source privée de ses lignes de commentaire.
func codeSeul(src string) string {
	return lignesDeCommentaire.ReplaceAllString(src, "")
}

// proprietaireJointure : le seul fichier qui a le droit de composer la jointure mesurée.
const proprietaireJointure = "kill_measured.go"

// exceptionsJointure : chemins (relatifs au paquet) autorisés à porter le motif, avec leur
// motif d'exception. Toute entrée nouvelle s'écrit ICI, datée et justifiée.
var exceptionsJointure = map[string]string{
	proprietaireJointure: "propriétaire de la jointure (lot 3, 2026-09-06)",
	"halo5/halo5_match_events_source.go": "énumération des events Halo 5 pour la timeline " +
		"(LEFT JOIN, positions natives, aucune mesure de distance) — 2026-09-06",
}

// TestJointureMesureeUneSeuleFois : aucun fichier hors exceptions ne NOMME une table de
// positions, ni sa vue, ni sa table brute.
func TestJointureMesureeUneSeuleFois(t *testing.T) {
	for path, src := range sourcesDuPaquet(t) {
		if _, ok := exceptionsJointure[path]; ok {
			continue
		}
		if tablesDePositions.MatchString(codeSeul(src)) {
			t.Errorf("%s nomme une table de positions dans son CODE — passer par "+
				"measuredKillsQuery(...) et les constantes positionsAt* de %s (règle n°6 : "+
				"les gardes `publishable`, d'unanimité et d'unicité du frag ne se recopient "+
				"pas ; et une lecture de la table BRUTE servirait des passes périmées)",
				path, proprietaireJointure)
		}
	}
}

// gardesDeLaJointure : ce que la requête DOIT porter, épinglé par son texte.
//
// LE PIN PORTE SUR LA CONSTANTE SQL ELLE-MÊME, PAS SUR LE FICHIER (revue adversariale du
// 2026-09-06, constat C5b) : la version précédente cherchait ces textes dans les OCTETS DU
// FICHIER, or l'en-tête de kill_measured.go cite la clause d'unanimité entre backticks. Le
// test était donc satisfait par un COMMENTAIRE : supprimer `HAVING count(DISTINCT
// e.source_tag) = 1` de la requête laissait le garde-rail vert. On lit désormais la valeur de
// `measuredKillsSQLTemplate` — le test vit dans le même paquet, il n'a rien à deviner.
var gardesDeLaJointure = []string{
	"e.publishable",
	"HAVING count(DISTINCT e.source_tag) = 1",
	"HAVING count(*) = 1 AND count(DISTINCT s.source_tag) = 1",
}

// TestProprietaireJointurePorteLesDeuxTables : le garde-rail ci-dessus serait creux si
// l'helper cessait de nommer les deux tables (renommage, suppression, déplacement dans un
// autre fichier) ou perdait l'une de ses gardes. Un garde-rail qui ne garde plus rien est
// pire qu'aucun garde-rail.
func TestProprietaireJointurePorteLesDeuxTables(t *testing.T) {
	if _, ok := sourcesDuPaquet(t)[proprietaireJointure]; !ok {
		t.Fatalf("%s introuvable — la jointure mesurée a changé de fichier sans son garde-rail",
			proprietaireJointure)
	}
	// Les deux tables sont nommées par les CONSTANTES, pas par le template : c'est là qu'il
	// faut les vérifier. Elles doivent rester les VUES `_latest` — une constante ramenée à la
	// table brute servirait les passes de décodage périmées (ADR 0026), sans rien casser.
	if string(positionsAtKill) != "kill_positions_latest" {
		t.Errorf("positionsAtKill = %q, attendu kill_positions_latest", string(positionsAtKill))
	}
	if string(positionsAtOpening) != "kill_openings_latest" {
		t.Errorf("positionsAtOpening = %q, attendu kill_openings_latest", string(positionsAtOpening))
	}
	// LA REQUÊTE, pas le fichier : un commentaire ne satisfait plus ce test.
	for _, garde := range gardesDeLaJointure {
		if !strings.Contains(measuredKillsSQLTemplate, garde) {
			t.Errorf("la requête de %s a perdu la garde %q — une mesure fausse redevient "+
				"publiable", proprietaireJointure, garde)
		}
	}
}
