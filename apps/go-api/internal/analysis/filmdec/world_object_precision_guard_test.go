package filmdec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// world_object_precision_guard_test.go — GARDE-RAIL des LECTEURS de `WorldObjectPrecision`.
//
// LA RÈGLE. `WorldObjectPrecision` est un GLOBAL de paquet dont le défaut est l'entrée
// `cliffhanger` du catalogue. Il n'est juste que si l'appelant a installé les largeurs de la
// carte du match (`replay.installWorldObjectPrecision`, appelé par `BuildFromFilm` sous
// `LockProcessDecode`). Un NOUVEAU lecteur posé hors de ce chemin lirait Cliffhanger en
// silence sur toutes les autres cartes — exactement le défaut corrigé le 2026-08-15, qui avait
// survécu des mois précisément parce que rien ne le signalait.
//
// LA GARDE : tout fichier de PRODUCTION qui mentionne `WorldObjectPrecision` doit figurer dans
// l'allowlist datée ci-dessous, avec la raison de sa présence. Ajouter une entrée est un acte
// délibéré, qui oblige à répondre « d'où ce lecteur tient-il les largeurs de la carte ? ».
//
// Les fichiers `_test.go` sont hors périmètre : les instruments de mesure installent et
// restaurent leurs propres largeurs, c'est leur objet même.

// worldObjectPrecisionReaders — ALLOWLIST DATÉE (2026-08-15). Chemin relatif à apps/go-api.
var worldObjectPrecisionReaders = map[string]string{
	"internal/analysis/filmdec/traverse.go": "déclaration du global, son setter, et les deux " +
		"lectures du chemin de traversée (`object-position-component`)",
	"internal/analysis/filmdec/projectiles.go": "longueur du champ (`projPosBits`) et " +
		"déquantification (`decodeWorldObjectPos`) — le balayage des objets du monde",
	"internal/analysis/filmdec/position_capture.go": "repli d'`absAxisW`, INATTEIGNABLE en " +
		"l'état : il est gardé par `absoluteAxisW > 0`, dont le défaut vaut 14 et dont le seul " +
		"écrivain (killsource/calibrate.go) balaie 6..26",
	"internal/analysis/filmdec/keyframe_ground_weapons.go": "CITATION en commentaire " +
		"(parenté des archétypes d'objet du monde) — aucune lecture de la valeur",
	"internal/analysis/filmdec/components_biped_anchor.go": "le corps tag==3 d'i59 (ancre du " +
		"grappin, 2026-08-16) lit sa position absolue aux largeurs d'axe de la CARTE — mêmes " +
		"chemins d'installation que le reste : production via BuildFromFilm/" +
		"installWorldObjectPrecision, instruments via i59aSetup (installation + restauration)",
	"internal/analysis/replay/world_object_precision.go": "l'INSTALLATEUR de production : " +
		"pose les largeurs de la carte du match et rend la restauration",
	"internal/analysis/replay/build.go": "le BRANCHEMENT : `BuildFromFilm` appelle " +
		"`installWorldObjectPrecision` depuis `Options.MapQuant`, sous le verrou de décodage",
}

// TestWorldObjectPrecisionReadersAreAllowlisted balaie les sources de production d'apps/go-api.
func TestWorldObjectPrecisionReadersAreAllowlisted(t *testing.T) {
	root := filepath.Join("..", "..", "..") // apps/go-api
	found := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		walkGoSources(t, root, dir, func(rel string, src []byte) {
			if strings.Contains(string(src), "WorldObjectPrecision") {
				found[rel] = true
			}
		})
	}
	for rel := range found {
		if _, ok := worldObjectPrecisionReaders[rel]; !ok {
			t.Errorf("%s mentionne WorldObjectPrecision hors allowlist. Ce global ne vaut que "+
				"si l'appelant a installé les largeurs de la carte du match "+
				"(replay.installWorldObjectPrecision) ; sinon il rend celles de Cliffhanger, en "+
				"silence, sur toutes les autres cartes. Dire d'où ce lecteur tient ses largeurs, "+
				"puis l'ajouter à worldObjectPrecisionReaders avec sa raison.", rel)
		}
	}
	for rel := range worldObjectPrecisionReaders {
		if !found[rel] {
			t.Errorf("%s est dans l'allowlist mais ne mentionne plus WorldObjectPrecision : "+
				"retirer l'entrée (une allowlist périmée finit par autoriser n'importe quoi)", rel)
		}
	}
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
