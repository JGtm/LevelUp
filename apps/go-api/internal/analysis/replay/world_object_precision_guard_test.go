package replay

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// world_object_precision_guard_test.go — GARDE-RAIL du correctif du 2026-08-15.
//
// CE QU'IL GARDE. `filmdec.WorldObjectPrecision` est un GLOBAL de paquet dont le défaut EST
// l'entrée `cliffhanger` du catalogue. Pendant des mois AUCUN chemin de production ne
// l'écrasait : toutes les autres cartes déquantifiaient leurs objets du monde aux largeurs de
// Cliffhanger, en silence. `BuildFromFilm` installe désormais les largeurs de l'entrée de
// catalogue DU MATCH et les restaure au retour.
//
// POURQUOI DEUX TESTS ET PAS UN. Un global oublié ne se voit pas : il faut garder les DEUX
// moitiés du correctif, et elles se cassent séparément.
//   - le MÉCANISME (installe puis restaure) : [TestInstallWorldObjectPrecision] ;
//   - le BRANCHEMENT (BuildFromFilm l'appelle, sous le verrou, depuis `opt.MapQuant`) :
//     [TestBuildFromFilmWiresWorldObjectPrecision], qui lit la source.
//
// Le branchement ne peut pas se vérifier en exécutant `BuildFromFilm` : le seul film versionné
// (`MiniFilmDir`) est une fenêtre de paquets DELTA sans image-clé de bipède, et le décodage
// s'arrête avant les objets du monde. Un test de source EST ici la seule garde toujours active
// — et c'est la forme que le dépôt emploie déjà pour ce genre d'invariant.

// TestInstallWorldObjectPrecision : le mécanisme. Installe les largeurs de l'entrée, les rend
// visibles pendant l'appel, et les restaure ensuite.
func TestInstallWorldObjectPrecision(t *testing.T) {
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })

	entry := filmdec.MapQuantEntry{Module: "ctf_bazaar", AxisWidths: [3]uint{17, 17, 16}}
	if entry.AxisWidths == prev.AxisW {
		t.Fatal("le cas de test doit différer du défaut de paquet, sinon il ne mesure rien")
	}
	restore := installWorldObjectPrecision(entry, "testdata")
	if got := filmdec.WorldObjectPrecision.AxisW; got != entry.AxisWidths {
		t.Fatalf("largeurs NON INSTALLÉES : %v, attendu %v (celles de la carte du match)",
			got, entry.AxisWidths)
	}
	restore()
	if filmdec.WorldObjectPrecision != prev {
		t.Fatalf("largeurs NON RESTAURÉES : %v, attendu %v — un global non rendu contamine le "+
			"film suivant décodé dans le même process",
			filmdec.WorldObjectPrecision.AxisW, prev.AxisW)
	}
}

// TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths : une entrée sans largeurs (catalogue
// antérieur au champ, entrée fabriquée à la main) garde le défaut. La dégradation est LOGGÉE
// par l'installateur — jamais silencieuse.
func TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths(t *testing.T) {
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })

	restore := installWorldObjectPrecision(filmdec.MapQuantEntry{Module: "sans_largeurs"}, "testdata")
	if filmdec.WorldObjectPrecision != prev {
		t.Fatalf("largeurs à zéro installées (%v) : le décodeur lirait des champs de 0 bit",
			filmdec.WorldObjectPrecision.AxisW)
	}
	restore()
	if filmdec.WorldObjectPrecision != prev {
		t.Fatalf("restauration fautive : %v, attendu %v",
			filmdec.WorldObjectPrecision.AxisW, prev.AxisW)
	}
}

// TestBuildFromFilmRefusesWithoutMapQuant : sans entrée de catalogue, aucun document. Bornes et
// largeurs venant du MÊME champ, l'état « bornes armées, largeurs oubliées » n'existe pas —
// c'est la raison d'être du champ unique `Options.MapQuant`.
func TestBuildFromFilmRefusesWithoutMapQuant(t *testing.T) {
	if _, err := BuildFromFilm("minifilm", "halo_infinite", MiniFilmDir, Options{}); err == nil {
		t.Fatal("BuildFromFilm a produit un document sans entrée de catalogue : les positions " +
			"ne seraient que des quanta déquantifiés au hasard")
	}
}

// TestBuildFromFilmWiresWorldObjectPrecision : le branchement. `BuildFromFilm` doit installer
// les largeurs DEPUIS `opt.MapQuant`, en DIFFÉRÉ (donc restaurer), et APRÈS avoir pris le verrou
// de décodage — le descripteur est un global, deux films décodés en parallèle se voleraient
// leurs largeurs.
func TestBuildFromFilmWiresWorldObjectPrecision(t *testing.T) {
	src, err := os.ReadFile("build.go")
	if err != nil {
		t.Fatal(err)
	}
	body, ok := funcBody(string(src), "func BuildFromFilm(")
	if !ok {
		t.Fatal("BuildFromFilm introuvable dans build.go : ce garde-rail ne garde plus rien")
	}
	install := regexp.MustCompile(`defer\s+installWorldObjectPrecision\(\*opt\.MapQuant\b`)
	if !install.MatchString(body) {
		t.Fatal("BuildFromFilm n'installe plus les largeurs d'axe depuis opt.MapQuant : les " +
			"objets du monde de TOUTES les cartes repassent en silence aux largeurs de " +
			"Cliffhanger (défaut de paquet). Mesuré le 2026-08-15 : la part d'échantillons de " +
			"projectile dans l'emprise des bipèdes tombe de ~99 % à 0,09-65 % hors Cliffhanger")
	}
	lock := strings.Index(body, "filmdec.LockProcessDecode()")
	if lock < 0 || lock > install.FindStringIndex(body)[0] {
		t.Fatal("l'installation des largeurs précède la prise du verrou de décodage : le " +
			"descripteur est un global de paquet, il ne s'écrit que sous le verrou")
	}
}

// funcBody rend le corps de la fonction dont la signature commence par head, du `{` ouvrant à
// l'accolade fermante de colonne 0 (convention gofmt).
func funcBody(src, head string) (string, bool) {
	i := strings.Index(src, head)
	if i < 0 {
		return "", false
	}
	rest := src[i:]
	if end := strings.Index(rest, "\n}\n"); end >= 0 {
		return rest[:end], true
	}
	return rest, true
}
