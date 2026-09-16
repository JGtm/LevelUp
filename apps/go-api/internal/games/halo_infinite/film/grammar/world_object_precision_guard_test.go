package grammar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// world_object_precision_guard_test.go — GARDE-RAIL des LECTEURS de `WorldObjectPrecision`.
//
// LA RÈGLE. Les largeurs world-object sont un CHAMP DU PROFIL DE BALAYAGE dont l'invariant est
// l'entrée `cliffhanger` du catalogue. Elles ne sont justes que si l'appelant a posé les
// largeurs de la carte du match (`replay.installWorldObjectPrecision`, appelé par
// `BuildFromFilm` sur le contexte du film). Un NOUVEAU lecteur posé hors de ce chemin lirait
// Cliffhanger en silence sur toutes les autres cartes — exactement le défaut corrigé le
// 2026-08-15, qui avait survécu des mois précisément parce que rien ne le signalait.
//
// DEPUIS LE LOT 2.3 CE N'EST PLUS UNE VARIABLE DE PAQUET : le profil voyage avec le contexte du
// film et avec le lecteur de bits. La garde ne change pas pour autant — la question qu'elle
// pose (« d'où ce lecteur tient-il les largeurs de la carte ? ») est la même, et sa réponse
// est désormais « de son profil », qui a lui aussi une provenance.
//
// LA GARDE : tout fichier de PRODUCTION qui mentionne les largeurs world-object doit figurer
// dans l'allowlist datée ci-dessous, avec la raison de sa présence. Ajouter une entrée est un
// acte délibéré, qui oblige à répondre « d'où ce lecteur tient-il les largeurs de la carte ? ».
//
// LES MARQUEURS SONT TROIS DEPUIS LE LOT 2.2.b, parce que la valeur a trois formes d'accès et
// qu'une garde qui n'en verrait qu'une laisserait passer les deux autres :
//
//	`orldObjectPrecision`    le champ du profil, son accesseur de lecteur
//	                         (`Lecteur.worldObjectPrecision`) et l'installateur ;
//	`largeursObjetDuMonde`   l'accès des trois lecteurs PAR DÉCALAGE D'OCTET (`projectiles.go`),
//	                         qui n'ont pas de lecteur de bits à porter le profil ;
//	`Movement.WorldObject`   la ligne de table du profil et sa godoc.
//
// Les fichiers `_test.go` sont hors périmètre : les instruments de mesure installent et
// restaurent leurs propres largeurs, c'est leur objet même.

// worldObjectPrecisionReaders — ALLOWLIST DATÉE (2026-08-15). Chemin relatif à apps/go-api.
var worldObjectPrecisionReaders = map[string]string{
	"internal/games/halo_infinite/film/grammar/grammar_rev_chronique.go": "CITATION dans " +
		"l'ENTRÉE DE CHRONIQUE du lot 2.3 (2026-09-17) : elle dit ce que le profil de balayage " +
		"a remplacé, donc elle nomme les largeurs world-object. Aucune lecture de la valeur — " +
		"ce fichier ne porte QUE la chronique de la révision de grammaire (sortie de " +
		"`grammar_rev.go` le 2026-09-18, lot 2.4.1, au seuil des 500 lignes)",
	"internal/games/halo_infinite/film/grammar/profil_balayage.go": "le PORTEUR (2026-09-17, " +
		"lot 2.3) : le champ `Mouvement.WorldObject` du profil de balayage, son accesseur et " +
		"les deux poses (brute, et depuis le découpage d'une carte). C'est ici que vivaient " +
		"la variable de paquet puis l'héritage de processus ; le profil se PASSE désormais",
	"internal/games/halo_infinite/film/grammar/film_context.go": "les trois accès du CONTEXTE " +
		"(2026-09-17, lot 2.3) : `LargeursObjetDuMonde`, ses deux poses, et le contexte des " +
		"enveloppes D2, qui pose le découpage LU DANS LE FILM. La cuisson, elle, prend celui " +
		"du CATALOGUE de la carte — et c'est écrit à chacun des deux endroits",
	"internal/games/halo_infinite/film/grammar/traverse.go": "la LARGEUR D'INDEX DE RÉGION lue par " +
		"la queue d'i60 (`consumeSimStateHandleTail`) — même contrat que le reste : les " +
		"largeurs viennent de l'appelant (BuildFromFilm / installWorldObjectPrecision). " +
		"Depuis le lot 2.7 (2026-09-16) c'est la SEULE mention restée dans ce fichier : la " +
		"déclaration, le setter et les deux lectures du chemin de traversée en sont sortis " +
		"par déplacement pur, vers `traverse_precision.go` et `dispatch_object.go`",
	"internal/games/halo_infinite/film/grammar/profile.go": "le CHAMP `Movement.WorldObject` et " +
		"son invariant (2026-09-17, lot 2.2.b) : l'entrée `cliffhanger` du catalogue, que " +
		"l'installateur de `replay` remplace par la carte du match. Aucune lecture de décodage ici",
	"internal/games/halo_infinite/film/grammar/profile_table.go": "la LIGNE DE TABLE de " +
		"`Movement.WorldObject` (2026-09-17, lot 2.2.b) : sa provenance, sa preuve et sa date, " +
		"comme toute valeur de profil. Elle DIT d'où viennent les largeurs — l'entrée de " +
		"catalogue de la carte — au lieu d'en lire une",
	"internal/games/halo_infinite/film/grammar/traverse_precision.go": "déclaration du global et son setter. " +
		"Vivait dans `traverse.go` jusqu'au lot 2.7 (2026-09-16), qui l'en a sorti par " +
		"déplacement pur — la scission des fichiers de plus de 500 lignes",
	"internal/games/halo_infinite/film/grammar/dispatch_object.go": "les deux lectures du chemin de " +
		"traversée (`object-position-component`). Vivaient dans le `switch` de `consumeByName`, " +
		"dans `traverse.go`, jusqu'au lot 2.7 (2026-09-16) qui a coupé ce switch en chaîne de " +
		"maillons par déplacement pur — même contrat d'installation qu'avant",
	"internal/games/halo_infinite/film/grammar/projectiles.go": "longueur du champ (`projPosBits`) et " +
		"déquantification (`decodeWorldObjectPos`) — le balayage des objets du monde",
	"internal/games/halo_infinite/film/grammar/components_movement.go": "CITATION en commentaire (2026-09-12, " +
		"lot B-bis) : le champ `Region` de `PrecisionDescriptor` documente qu'il ne vaut que " +
		"pour le descripteur world-object et pourquoi il vit DANS la structure (restauration " +
		"par valeur par l'installateur) — aucune lecture de la valeur ici",
	"internal/games/halo_infinite/film/grammar/position_capture.go": "repli d'`absAxisW`, INATTEIGNABLE en " +
		"l'état : il est gardé par `br.absoluteAxisW() > 0`, dont le défaut vaut 14 et dont le " +
		"seul écrivain (killsource/calibrate.go) balaie 6..26 — depuis le lot 2.2.a il le fait " +
		"par `FrameConfig.Mouvement` et par l'héritage de processus, plus par une variable de paquet",
	"internal/games/halo_infinite/film/grammar/keyframe_ground_weapons.go": "CITATION en commentaire " +
		"(parenté des archétypes d'objet du monde) — aucune lecture de la valeur",
	"internal/games/halo_infinite/film/grammar/components_biped_anchor.go": "le corps tag==3 d'i59 (ancre du " +
		"grappin, 2026-08-16) lit sa position absolue aux largeurs d'axe de la CARTE — mêmes " +
		"chemins d'installation que le reste : production via BuildFromFilm/" +
		"installWorldObjectPrecision, instruments via i59aSetup (installation + restauration)",
	"internal/games/halo_infinite/film/grammar/grapple_state.go": "CITATION en commentaire : le balayage de " +
		"production des événements de grappin documente que ses quanta de position sont aux " +
		"largeurs installées par l'appelant (le corps d'i59 les lit, cf. " +
		"components_biped_anchor.go) — aucune lecture de la valeur ici",
	"internal/games/halo_infinite/film/replay/world_object_precision.go": "l'INSTALLATEUR de production : " +
		"pose les largeurs de la carte du match et rend la restauration",
	"internal/games/halo_infinite/film/replay/build_from_film.go": "le BRANCHEMENT : `BuildFromFilm` appelle " +
		"`installWorldObjectPrecision` depuis `Options.MapQuant`, qui les pose sur le CONTEXTE " +
		"du film (lot 2.3 — plus aucun état de processus). " +
		"Le fichier s'appelait `build.go` jusqu'au lot 1 de PLAN_CUISSON_PERF (2026-09-02), qui " +
		"a sorti le decodage de l'assemblage par un deplacement pur",
}

// TestWorldObjectPrecisionReadersAreAllowlisted balaie les sources de production d'apps/go-api.
func TestWorldObjectPrecisionReadersAreAllowlisted(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..") // apps/go-api
	found := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		walkGoSources(t, root, dir, func(rel string, src []byte) {
			if mentionneLargeursObjetDuMonde(string(src)) {
				found[rel] = true
			}
		})
	}
	for rel := range found {
		if _, ok := worldObjectPrecisionReaders[rel]; !ok {
			t.Errorf("%s mentionne les largeurs world-object hors allowlist. Elles ne valent que "+
				"si l'appelant a installé les largeurs de la carte du match "+
				"(replay.installWorldObjectPrecision) ; sinon il rend celles de Cliffhanger, en "+
				"silence, sur toutes les autres cartes. Dire d'où ce lecteur tient ses largeurs, "+
				"puis l'ajouter à worldObjectPrecisionReaders avec sa raison.", rel)
		}
	}
	for rel := range worldObjectPrecisionReaders {
		if !found[rel] {
			t.Errorf("%s est dans l'allowlist mais ne mentionne plus les largeurs world-object : "+
				"retirer l'entrée (une allowlist périmée finit par autoriser n'importe quoi)", rel)
		}
	}
}

// marqueursLargeursObjetDuMonde : les trois formes d'accès (cf. l'en-tête).
var marqueursLargeursObjetDuMonde = []string{
	"orldObjectPrecision", "largeursObjetDuMonde", "Movement.WorldObject",
}

// mentionneLargeursObjetDuMonde dit si une source touche aux largeurs world-object.
func mentionneLargeursObjetDuMonde(src string) bool {
	for _, m := range marqueursLargeursObjetDuMonde {
		if strings.Contains(src, m) {
			return true
		}
	}
	return false
}

// walkGoSources applique fn à chaque .go de production (hors _test.go) sous root/dir, avec un
// chemin relatif à root en séparateurs slash.
func walkGoSources(t *testing.T, root, dir string, fn func(rel string, src []byte)) {
	t.Helper()
	base := filepath.Join(root, dir)
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fn(filepath.ToSlash(rel), src)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours des sources %s : %v", base, err)
	}
}
