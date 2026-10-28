// Package archlint — no_accidental_goos_suffix_test.go : ratchet du 2026-09-06.
//
// # LE PIEGE, ET IL A COUTE UNE CI ROUGE
//
// Go applique une contrainte de compilation IMPLICITE au NOM du fichier : `x_windows.go` n'est
// compile que sur Windows, `x_linux.go` que sur Linux, `x_amd64.go` que sur amd64. Aucune ligne
// du fichier ne le dit ; rien ne le signale a la lecture.
//
// Le 2026-09-06, le lot des bornes de manche a nomme son fichier `round_windows.go` — « les
// FENETRES de manche ». Le paquet compilait et tous les tests passaient en local (Windows) ; sur
// le runner Linux de la CI le fichier n'existait tout simplement pas, et
// `go vet ./internal/analysis/...` rendait `undefined: ResolveRoundWindows`. Le poste de
// developpement ne pouvait PAS voir le defaut : c'est exactement la classe de faute qu'un ratchet
// doit attraper.
//
// # CE QUE CE TEST INTERDIT
//
// Un fichier `.go` du module dont le nom se termine par un suffixe GOOS ou GOARCH connu, sauf s'il
// est dans l'allowlist ci-dessous — c'est-a-dire s'il EST reellement propre a une plateforme.
//
// Un fichier legitime reste legitime : l'allowlist en porte cinq, tous des adaptateurs systeme
// Windows verifies. Un fichier qui n'a rien de specifique a une plateforme se renomme (le lot des
// bornes a rendu `round_windows.go` -> `round_bounds.go`).
package archlint

import (
	"path"
	"strings"
	"testing"
)

// goosArchSuffixes : les suffixes que le compilateur Go lit comme une contrainte de plateforme.
// Liste des GOOS et GOARCH de `go tool dist list`, tenue courte aux valeurs qu'un nom de fichier
// francais ou anglais du depot pourrait rencontrer par accident — un suffixe absent d'ici serait
// de toute facon attrape par la CI Linux, comme celui-ci l'a ete.
var goosArchSuffixes = []string{
	"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux",
	"nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos",
	"386", "amd64", "arm", "arm64", "loong64", "mips", "mips64", "mips64le", "mipsle",
	"ppc64", "ppc64le", "riscv64", "s390x", "sparc64", "wasm",
}

// goosSuffixAllowed : les fichiers REELLEMENT propres a une plateforme, verifies un a un le
// 2026-09-06. Chemin relatif a la racine du depot.
//
// Ajouter une entree exige que le fichier soit un ADAPTATEUR SYSTEME (appel d'API du systeme
// d'exploitation, ou type de donnee propre a une architecture) — pas un fichier metier dont le
// nom tombe par hasard sur un suffixe.
var goosSuffixAllowed = map[string]string{
	"apps/go-api/internal/filmproc/alive_windows.go": "vie d'un processus : OpenProcess/GetExitCodeProcess (Win32), " +
		"ajoute le 2026-10-17 (lecture du detenteur du verrou de decodage)",
	"apps/go-api/internal/filmproc/priority_windows.go":          "priorite de processus : SetPriorityClass (Win32)",
	"apps/go-api/internal/filmproc/selfpriority_windows.go":      "priorite du processus courant : Win32",
	"libs/gamefiles/himodule/projection_windows.go":              "projection memoire d'un module : CreateFileMapping (Win32)",
	"apps/go-api/internal/platform/diskfree/diskfree_windows.go": "espace disque : GetDiskFreeSpaceEx (Win32)",
	"apps/go-api/internal/platform/filelock/filelock_windows.go": "verrou de fichier entre processus : " +
		"LockFileEx (Win32), ajoute le 2026-09-26 (J2.6, DT-2)",
	"apps/go-api/internal/platform/session/store_purge_windows_test.go": "purge de sessions : comportement " +
		"de suppression propre a Windows",
}

// TestAucunSuffixeDePlateformeAccidentel balaie l'espace de travail (`apps/go-api` et `libs/**`) et
// refuse tout nom de fichier qui porte une contrainte de plateforme implicite sans etre dans
// l'allowlist.
func TestAucunSuffixeDePlateformeAccidentel(t *testing.T) {
	var violations []string
	for _, f := range workspaceGoSources(t, racinesEspaceDeTravail...) {
		if !porteUnSuffixeDePlateforme(path.Base(f.Rel)) {
			continue
		}
		if _, permis := goosSuffixAllowed[f.Rel]; !permis {
			violations = append(violations, f.Rel)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("ces fichiers portent une contrainte de compilation IMPLICITE par leur nom "+
			"(Go ne les compile que sur la plateforme du suffixe), et ne sont pas des adaptateurs "+
			"systeme : les RENOMMER, ou les inscrire dans `goosSuffixAllowed` avec leur "+
			"justification.\n  %s", strings.Join(violations, "\n  "))
	}
}

// porteUnSuffixeDePlateforme dit si un nom de fichier `.go` se termine par `_<goos>` ou
// `_<goarch>`, le suffixe `_test` retire au prealable — `x_windows_test.go` est aussi
// contraint que `x_windows.go`.
func porteUnSuffixeDePlateforme(nom string) bool {
	base := strings.TrimSuffix(nom, ".go")
	base = strings.TrimSuffix(base, "_test")
	for _, suf := range goosArchSuffixes {
		if strings.HasSuffix(base, "_"+suf) {
			return true
		}
	}
	return false
}
