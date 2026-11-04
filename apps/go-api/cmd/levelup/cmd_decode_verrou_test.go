package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"

	"github.com/JGtm/LevelUp/libs/film/filmproc"
)

// Un verrou tenu = refus immediat, detenteur nomme, code de sortie 2 ; rendu, la commande passe.
func TestPrendreLeVerrouDeDecodage_RefusImmediatPuisPasse(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	tenu, err := filmproc.AcquireSolo(cfg.PathResolver().CacheRootDir(), "film-regen", "un-film")
	if err != nil {
		t.Fatalf("verrou temoin : %v", err)
	}

	_, err = prendreLeVerrouDeDecodage(cfg, outilBackfillKillSource)
	var tenuErr *verrouDeDecodageTenu
	if !errors.As(err, &tenuErr) || !errors.Is(err, filmproc.ErrDecodeBusy) {
		t.Fatalf("erreur = %v, attendu un refus d'occupation (verrouDeDecodageTenu / ErrDecodeBusy)", err)
	}
	if !strings.Contains(err.Error(), "film-regen") || !strings.Contains(err.Error(), outilBackfillKillSource) {
		t.Errorf("le message doit nommer l'outil refuse et le detenteur : %v", err)
	}
	if code := sortirSur(err); code != CodeSortieVerrouTenu {
		t.Fatalf("code de sortie = %d, attendu %d", code, CodeSortieVerrouTenu)
	}
	if sortirSur(errors.New("autre")) != 1 {
		t.Error("une erreur ordinaire garde le code 1")
	}

	tenu.Release()
	release, err := prendreLeVerrouDeDecodage(cfg, outilBackfillKillSource)
	if err != nil {
		t.Fatalf("verrou libre : %v", err)
	}
	// Tant que la commande le tient, une seconde prise est refusee.
	if _, err := prendreLeVerrouDeDecodage(cfg, "autre"); !errors.Is(err, filmproc.ErrDecodeBusy) {
		t.Fatalf("seconde prise = %v, attendu un refus", err)
	}
	release()
	release2, err := prendreLeVerrouDeDecodage(cfg, "autre")
	if err != nil {
		t.Fatalf("verrou rendu : %v", err)
	}
	release2()
}

// Garde-rail : une commande de `levelup` qui construit le collecteur de decodage en processus
// (killcollector.NewKillSourceCollector) ou decode un film (decfilm.Decode, decfilm.Load, decfilm.LoadDir) n'existe
// que dans les fichiers de la passe `backfill-killsource`, et cette passe prend le verrou solo avant la base.
// Un nouveau decodeur en processus doit prendre le verrou et s'inscrire ici.
func TestDecodageEnProcessus_PrendLeVerrouSolo(t *testing.T) {
	decodeurs := []string{"NewKillSourceCollector(", "decfilm.Decode(", "decfilm.Load(", "decfilm.LoadDir("}
	permis := map[string]bool{
		"cmd_backfill_killsource_carte.go":  true,
		"cmd_backfill_killsource_online.go": true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range decodeurs {
			if usesInCode(string(src), d) && !permis[f] {
				t.Errorf("%s decode en processus (%s) sans etre inscrit : prendre prendreLeVerrouDeDecodage "+
					"(refus immediat, sortie 3) et l'inscrire dans ce test", f, d)
			}
		}
	}
	src, err := os.ReadFile("cmd_backfill_killsource.go")
	if err != nil {
		t.Fatal(err)
	}
	if !usesInCode(string(src), "verrouDeLaPasse(") {
		t.Error("runBackfillKillSource doit prendre le verrou solo de decodage (verrouDeLaPasse)")
	}
}

// usesInCode dit si motif apparait dans une ligne de code (hors commentaire de ligne).
func usesInCode(src, motif string) bool {
	for _, line := range strings.Split(src, "\n") {
		code, _, _ := strings.Cut(line, "//")
		if strings.Contains(code, motif) {
			return true
		}
	}
	return false
}

// --dry-run et --credit-only ne decodent rien : ils ne prennent pas le verrou (et ne sont donc pas refuses par un
// serveur qui cuit), la passe qui decode le prend.
func TestVerrouDeLaPasse_SeulLeDecodageLePrend(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	tenu, err := filmproc.AcquireSolo(cfg.PathResolver().CacheRootDir(), "film-regen", "un-film")
	if err != nil {
		t.Fatal(err)
	}
	defer tenu.Release()
	for _, o := range []killsourceOptions{{dryRun: true}, {creditOnly: true}} {
		relacher, err := verrouDeLaPasse(cfg, o)
		if err != nil {
			t.Fatalf("%+v : refuse alors qu'il ne decode pas : %v", o, err)
		}
		relacher()
	}
	if _, err := verrouDeLaPasse(cfg, killsourceOptions{}); !errors.Is(err, filmproc.ErrDecodeBusy) {
		t.Fatalf("la passe qui decode doit etre refusee : %v", err)
	}
	if _, err := verrouDeLaPasse(cfg, killsourceOptions{online: true}); !errors.Is(err, filmproc.ErrDecodeBusy) {
		t.Fatalf("la passe en ligne decode aussi : %v", err)
	}
}

// Revue V9 : 2 est le code de `flag.ExitOnError` (ligne de commande refusee, rendu par le paquet flag avant toute
// commande) ; « verrou tenu » doit s'en distinguer, comme de l'echec ordinaire (1) et de l'interruption (130).
func TestCodeSortieVerrouTenu_DistinctDesAutresCodes(t *testing.T) {
	const codeFlagExitOnError = 2
	if CodeSortieVerrouTenu != 3 {
		t.Fatalf("CodeSortieVerrouTenu = %d, documente a 3 (docs/COMMANDS.md)", CodeSortieVerrouTenu)
	}
	for _, autre := range []int{0, 1, codeFlagExitOnError, CodeSortieInterrompue} {
		if CodeSortieVerrouTenu == autre {
			t.Fatalf("CodeSortieVerrouTenu = %d se confond avec un autre code de sortie", autre)
		}
	}
}
