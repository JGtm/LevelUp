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
	"internal/games/halo_infinite/film/filmdec/build_profile.go": "CITATION en commentaire " +
		"(2026-09-15, lot 1.9.1 bis pas 3) : `InstallFilmFormatMPP` renvoie au contrat de " +
		"`replay.installWorldObjectPrecision` pour dire que son appelant doit detenir " +
		"LockProcessDecode — le profil par build installe les largeurs MPP, jamais celles " +
		"des axes ; aucune lecture de la valeur ici",
	"internal/games/halo_infinite/film/filmdec/traverse.go": "la LARGEUR D'INDEX DE RÉGION lue par " +
		"la queue d'i60 (`consumeSimStateHandleTail`) — même contrat que le reste : les " +
		"largeurs viennent de l'appelant (BuildFromFilm / installWorldObjectPrecision). " +
		"Depuis le lot 2.7 (2026-09-16) c'est la SEULE mention restée dans ce fichier : la " +
		"déclaration, le setter et les deux lectures du chemin de traversée en sont sortis " +
		"par déplacement pur, vers `traverse_precision.go` et `dispatch_object.go`",
	"internal/games/halo_infinite/film/filmdec/traverse_precision.go": "déclaration du global et son setter. " +
		"Vivait dans `traverse.go` jusqu'au lot 2.7 (2026-09-16), qui l'en a sorti par " +
		"déplacement pur — la scission des fichiers de plus de 500 lignes",
	"internal/games/halo_infinite/film/filmdec/dispatch_object.go": "les deux lectures du chemin de " +
		"traversée (`object-position-component`). Vivaient dans le `switch` de `consumeByName`, " +
		"dans `traverse.go`, jusqu'au lot 2.7 (2026-09-16) qui a coupé ce switch en chaîne de " +
		"maillons par déplacement pur — même contrat d'installation qu'avant",
	"internal/games/halo_infinite/film/filmdec/projectiles.go": "longueur du champ (`projPosBits`) et " +
		"déquantification (`decodeWorldObjectPos`) — le balayage des objets du monde",
	"internal/games/halo_infinite/film/filmdec/components_movement.go": "CITATION en commentaire (2026-09-12, " +
		"lot B-bis) : le champ `Region` de `PrecisionDescriptor` documente qu'il ne vaut que " +
		"pour le descripteur world-object et pourquoi il vit DANS la structure (restauration " +
		"par valeur par l'installateur) — aucune lecture de la valeur ici",
	"internal/games/halo_infinite/film/filmdec/position_capture.go": "repli d'`absAxisW`, INATTEIGNABLE en " +
		"l'état : il est gardé par `absoluteAxisW > 0`, dont le défaut vaut 14 et dont le seul " +
		"écrivain (killsource/calibrate.go) balaie 6..26",
	"internal/games/halo_infinite/film/filmdec/keyframe_ground_weapons.go": "CITATION en commentaire " +
		"(parenté des archétypes d'objet du monde) — aucune lecture de la valeur",
	"internal/games/halo_infinite/film/filmdec/components_biped_anchor.go": "le corps tag==3 d'i59 (ancre du " +
		"grappin, 2026-08-16) lit sa position absolue aux largeurs d'axe de la CARTE — mêmes " +
		"chemins d'installation que le reste : production via BuildFromFilm/" +
		"installWorldObjectPrecision, instruments via i59aSetup (installation + restauration)",
	"internal/games/halo_infinite/film/filmdec/grapple_state.go": "CITATION en commentaire : le balayage de " +
		"production des événements de grappin documente que ses quanta de position sont aux " +
		"largeurs installées par l'appelant (le corps d'i59 les lit, cf. " +
		"components_biped_anchor.go) — aucune lecture de la valeur ici",
	"internal/games/halo_infinite/film/replay/world_object_precision.go": "l'INSTALLATEUR de production : " +
		"pose les largeurs de la carte du match et rend la restauration",
	"internal/games/halo_infinite/film/replay/build_from_film.go": "le BRANCHEMENT : `BuildFromFilm` appelle " +
		"`installWorldObjectPrecision` depuis `Options.MapQuant`, sous le verrou de décodage. " +
		"Le fichier s'appelait `build.go` jusqu'au lot 1 de PLAN_CUISSON_PERF (2026-09-02), qui " +
		"a sorti le decodage de l'assemblage par un deplacement pur",
	"internal/games/halo_infinite/film/replay/build_ground_weapons.go": "2026-08-17 (correctif de revue du " +
		"lot des socles) : le décodage `ti=42` est SOUS les largeurs que `BuildFromFilm` a " +
		"installées juste avant — il ne les pose pas, il en dépend, et son commentaire le dit " +
		"en citant l'installateur. Il pose en revanche les largeurs du bloc MPP " +
		"(`gwInstallMPPWidths`), qui sont un AUTRE global et qu'il restaure lui-même",
}

// TestWorldObjectPrecisionReadersAreAllowlisted balaie les sources de production d'apps/go-api.
func TestWorldObjectPrecisionReadersAreAllowlisted(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..") // apps/go-api
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
