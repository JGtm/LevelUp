package anonymisegamertags

// registre_test.go — LE REGISTRE DES IDENTIFIANTS FICTIFS ET SON GARDE-RAIL.
//
// `testdata/IDENTITES.tsv` de la brique du decodeur (resolveur des briques) liste les xuids fictifs qui tiennent lieu d identifiants de
// joueurs dans les fixtures du decodeur, et `testdata/IDENTITES_APPS.tsv` (a cote de ce fichier) ceux
// des fichiers suivis sous `apps/`. Le garde-rail tourne a chaque `go test` : un xuid de la plage des
// xuids qui figure dans un fichier TEXTE de `libs/film` ou de `apps/` sans etre a l un des deux
// registres est un identifiant que personne n a rendu fictif — le plus souvent un xuid reel recopie
// d un film ou de la base dans un test. Les fichiers `.gz` sont lus decompresses.
//
// Il lit le decimal a seize chiffres, l hexadecimal a seize chiffres (avec ou sans `0x`) et l hexadecimal
// a octets inverses. Il ne voit pas les octets des films ni les fichiers du codec des faits : pour
// eux, c est la lecture du decodeur (table des joueurs, fil des morts) qui l etablit, a la
// regeneration d une fixture (cf. doc.go), et `binaires_couverts_test.go` liste, dans une liste
// fermee, quelle preuve couvre chaque famille de binaires.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/briques"
	"levelup/go-api/internal/identitefictive"
)

// libelleRegistre nomme le registre de la brique dans les messages ; cheminRegistre le situe sur le
// disque par le resolveur des briques.
const libelleRegistre = "brique du decodeur, testdata/IDENTITES.tsv"

func cheminRegistre() (string, error) {
	return briques.Chemin("film", "testdata", "IDENTITES.tsv")
}

// cheminRegistreApps : le registre des fichiers suivis sous `apps/`.
var cheminRegistreApps = filepath.Join("apps", "go-api", "cmd", "anonymise-gamertags", "testdata", "IDENTITES_APPS.tsv")

// dossiersNonSuivis : dossiers que git ne suit pas (dependances, sorties de build) et que le
// balayage de `apps/` ne lit pas.
var dossiersNonSuivis = map[string]bool{".git": true, "node_modules": true, "target": true, "dist": true}

// nombresHorsXuid : nombres de seize chiffres qui tombent dans la plage des xuids sans en etre, nommes
// un a un avec leur raison. Une ligne n exempte que le nombre exact. Les parties decimales d un flottant
// (`0.2857142857142857`) ne sont pas listees : le balayage les ignore, une suite de chiffres precedee de
// `<chiffre>.` n etant pas un entier.
var nombresHorsXuid = map[string]string{
	"00090000DEADBEEF": "motif de bits d'un test du resolveur de familles du decodeur (weaponv3/pi_resolver_bits_test.go), " +
		"lu par le balayage du module du decodeur : une constante, pas un identifiant",
}

// extensionsTexte : les fichiers que le garde-rail lit (les films, binaires, n en font pas partie).
var extensionsTexte = map[string]bool{
	".go": true, ".json": true, ".golden": true, ".txt": true, ".tsv": true, ".md": true,
	".mod": true, ".sum": true,
	// Fichiers propres au depot dedie d une brique publiee (ADR 0043), presents dans son dossier du cache
	// de modules : configuration de CI et de git, lus comme du texte.
	".yml": true, ".gitignore": true, ".gitattributes": true,
}

// textesSansExtension : fichiers sans extension lus comme du texte, par nom exact (la licence d un
// depot dedie). Tout autre fichier sans extension reste hors du texte, donc signale s il n est pas couvert.
var textesSansExtension = map[string]bool{"LICENSE": true}

// lireRegistre rend les xuids du registre, et refuse un registre incoherent : une ligne mal
// formee, un xuid hors plage ou en double, un registre vide.
func lireRegistre(chemin string) (map[uint64]bool, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin fixe, issu de la racine du depot
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[uint64]bool{}
	sc := bufio.NewScanner(f)
	for ligne := 1; sc.Scan(); ligne++ {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		champs := strings.Split(l, "\t")
		if len(champs) != 2 {
			return nil, fmt.Errorf("ligne %d : attendu decimal<TAB>hexadecimal, lu %q", ligne, l)
		}
		v, err := strconv.ParseUint(champs[0], 10, 64)
		if err != nil || champs[1] != "0x"+hexa(v, false) {
			return nil, fmt.Errorf("ligne %d : %q et %q ne decrivent pas le meme xuid", ligne, champs[0], champs[1])
		}
		if !identitefictive.DansLaPlage(v) {
			return nil, fmt.Errorf("ligne %d : %d hors de la plage des xuids", ligne, v)
		}
		if out[v] {
			return nil, fmt.Errorf("ligne %d : %d en double", ligne, v)
		}
		out[v] = true
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("registre vide")
	}
	return out, nil
}

// extensionsTexteApps : les fichiers que le garde-rail lit sous `apps/` (code, donnees, manifestes).
var extensionsTexteApps = map[string]bool{
	".go": true, ".json": true, ".golden": true, ".txt": true, ".tsv": true, ".md": true, ".mod": true,
	".sum": true, ".ts": true, ".tsx": true, ".js": true, ".mjs": true, ".toml": true, ".html": true,
	".snap": true, ".rs": true, ".sh": true, ".ps1": true, ".yaml": true, ".yml": true, ".xml": true,
	".css": true, ".py": true, ".svg": true, ".snapshot": true, ".out": true, ".lock": true,
	".gitignore": true, ".gitattributes": true, ".npmrc": true, ".nsh": true, ".tanstack": true, "": true,
}

// extensionDuTexte rend l extension qui dit ce qu un fichier contient : celle du fichier, ou celle sous le
// `.gz` pour un fichier compresse.
func extensionDuTexte(chemin string) string {
	if textesSansExtension[filepath.Base(chemin)] {
		return ".txt"
	}
	if ext := filepath.Ext(chemin); ext != ".gz" {
		return ext
	}
	return filepath.Ext(strings.TrimSuffix(chemin, ".gz"))
}

// lireTexte rend les octets que le garde-rail balaie pour `chemin`, ou nil si ce n est pas un texte :
// un fichier `.gz` est decompresse et son extension interne decide (`x.json.gz` est un JSON).
func lireTexte(chemin string, extensions map[string]bool) ([]byte, error) {
	if !extensions[extensionDuTexte(chemin)] {
		return nil, nil
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours
	if err != nil {
		return nil, err
	}
	if filepath.Ext(chemin) != ".gz" {
		return brut, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(brut))
	if err != nil {
		return nil, fmt.Errorf("%s : en-tete gzip : %w", chemin, err)
	}
	clair, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("%s : flux gzip : %w", chemin, err)
	}
	return clair, nil
}

// xuidsHorsRegistre parcourt les fichiers texte (`extensions`) sous `racine` et rend, par fichier, les
// xuids de la plage que le registre ne porte pas.
func xuidsHorsRegistre(racine string, registre map[uint64]bool, extensions map[string]bool) (map[string][]string, error) {
	hors := map[string][]string{}
	err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if dossiersNonSuivis[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		brut, err := lireTexte(p, extensions)
		if err != nil || brut == nil {
			return err
		}
		if l := xuidsHorsRegistreDuTexte(brut, registre); len(l) > 0 {
			hors[filepath.ToSlash(p)] = l
		}
		return nil
	})
	return hors, err
}

// xuidsHorsRegistreDuTexte rend les xuids de la plage, absents du registre, qu un texte porte sous
// les trois formes que l outil sait ecrire : decimal a seize chiffres, hexadecimal a seize chiffres
// (prefixe `0x` ou non, casse indifferente), hexadecimal a octets inverses.
func xuidsHorsRegistreDuTexte(brut []byte, registre map[uint64]bool) []string {
	var hors []string
	hors = appendSiHorsRegistre(hors, suitesDeSeize(brut, estChiffre, true), 10, false, registre)
	hexa := suitesDeSeize(brut, estChiffreHexa, false)
	hors = appendSiHorsRegistre(hors, hexa, 16, false, registre)
	return appendSiHorsRegistre(hors, hexa, 16, true, registre)
}

// suitesDeSeize rend les suites MAXIMALES de seize caracteres de l alphabet (`dans`), dans l ordre du
// texte ; une suite plus courte ou plus longue n est pas un xuid. Un seul passage, sans expression
// reguliere : le garde-rail lit des dizaines de milliers de fichiers. Avec `entiere`, une suite qui
// suit `<chiffre>.` est ecartee : c est la partie decimale d un flottant, pas un entier.
func suitesDeSeize(brut []byte, dans func(byte) bool, entiere bool) [][]byte {
	var out [][]byte
	for debut := 0; debut < len(brut); {
		if !dans(brut[debut]) {
			debut++
			continue
		}
		fin := debut + 1
		for fin < len(brut) && dans(brut[fin]) {
			fin++
		}
		partieDecimale := entiere && debut >= 2 && brut[debut-1] == '.' && estChiffre(brut[debut-2])
		if fin-debut == 16 && !partieDecimale {
			out = append(out, brut[debut:fin])
		}
		debut = fin
	}
	return out
}

// appendSiHorsRegistre ajoute a `hors` les suites de seize chiffres de la `base` dont la valeur
// (octets inverses si `inverse`) est dans la plage sans etre au registre.
func appendSiHorsRegistre(hors []string, suites [][]byte, base int, inverse bool, registre map[uint64]bool) []string {
	for _, m := range suites {
		if len(m) != 16 {
			continue
		}
		v, err := strconv.ParseUint(string(m), base, 64)
		if err != nil {
			continue
		}
		if inverse {
			v = bits.ReverseBytes64(v)
		}
		if identitefictive.DansLaPlage(v) && !registre[v] && nombresHorsXuid[string(m)] == "" {
			hors = append(hors, string(m))
		}
	}
	return hors
}

// lireRegistres rend l union des deux registres, et refuse un xuid inscrit dans les deux : chaque
// fictif n a qu une seule ligne.
func lireRegistres(racine string) (map[uint64]bool, error) {
	cheminFilm, err := cheminRegistre()
	if err != nil {
		return nil, fmt.Errorf("%s : %w", libelleRegistre, err)
	}
	film, err := lireRegistre(cheminFilm)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", libelleRegistre, err)
	}
	apps, err := lireRegistre(filepath.Join(racine, cheminRegistreApps))
	if err != nil {
		return nil, fmt.Errorf("%s : %w", filepath.ToSlash(cheminRegistreApps), err)
	}
	for v := range apps {
		if film[v] {
			return nil, fmt.Errorf("%d figure aux deux registres", v)
		}
		film[v] = true
	}
	return film, nil
}

// TestLesXuidsDesFixturesSontFictifs : aucun xuid de la plage, hors registres, dans les fichiers
// texte de `libs/film` ni dans ceux de `apps/` (les `.gz` lus decompresses).
func TestLesXuidsDesFixturesSontFictifs(t *testing.T) {
	registre, err := lireRegistres(racineDuDepot)
	if err != nil {
		t.Fatalf("registres des identifiants fictifs : %v", err)
	}
	hors := map[string][]string{}
	for _, cible := range []struct {
		dossier    string
		extensions map[string]bool
	}{{dossierDeLaBrique(t), extensionsTexte}, {filepath.Join(racineDuDepot, "apps"), extensionsTexteApps}} {
		h, err := xuidsHorsRegistre(cible.dossier, registre, cible.extensions)
		if err != nil {
			t.Fatal(err)
		}
		for f, l := range h {
			hors[f] = l
		}
	}
	if len(hors) == 0 {
		return
	}
	fichiers := make([]string, 0, len(hors))
	for f := range hors {
		fichiers = append(fichiers, f)
	}
	sort.Strings(fichiers)
	var sb strings.Builder
	for _, f := range fichiers {
		fmt.Fprintf(&sb, "\n  %s : %s", f, strings.Join(hors[f], ", "))
	}
	t.Fatalf("%d fichier(s) portent un xuid hors de %s et de %s : un identifiant reel, ou un fictif a ajouter "+
		"a l un des registres (utiliser de preference un xuid qui y figure deja), ou un nombre qui n est pas "+
		"un xuid a nommer dans nombresHorsXuid avec sa raison%s", len(hors), libelleRegistre,
		filepath.ToSlash(cheminRegistreApps), sb.String())
}

// TestLeGardeRailVoitUnXuidHorsRegistre : sur un dossier synthetique, un xuid absent du registre
// est signale sous chacune de ses formes (decimal, hexadecimal avec ou sans `0x` en minuscules ou
// majuscules, hexadecimal a octets inverses), un xuid du registre et un nombre qui n est pas dans la
// plage ne le sont pas, et une suite plus longue que seize chiffres n est pas un xuid.
func TestLeGardeRailVoitUnXuidHorsRegistre(t *testing.T) {
	dossier := t.TempDir()
	const dedans, dehors = "2550913680076637", "2533274790395904"
	v, _ := strconv.ParseUint(dedans, 10, 64)
	r, _ := strconv.ParseUint(dehors, 10, 64)
	contenu := "fictif " + dedans + "\nreel " + dehors + "\nautre 1234567890123456\ncolle 1" + dehors + "9\n" +
		fmt.Sprintf("fictif hexa 0x%016x\nreel hexa 0x%016x\nreel hexa majuscules %016X\n", v, r+1, r+2) +
		fmt.Sprintf("reel hexa sans prefixe %016x\nreel octets inverses %016x\n", r+3, bits.ReverseBytes64(r+4)) +
		fmt.Sprintf("fictif octets inverses %016x\nhors plage deadbeefdeadbeef\ncolle 0x1%016x\n", bits.ReverseBytes64(v), r+5)
	if err := os.WriteFile(filepath.Join(dossier, "x.go"), []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dossier, "x.bin"), []byte(dehors), 0o600); err != nil {
		t.Fatal(err)
	}
	hors, err := xuidsHorsRegistre(dossier, map[uint64]bool{v: true}, extensionsTexte)
	if err != nil || len(hors) != 1 {
		t.Fatalf("hors registre : %v (%v)", hors, err)
	}
	attendu := []string{dehors, fmt.Sprintf("%016x", r+1), fmt.Sprintf("%016X", r+2), fmt.Sprintf("%016x", r+3),
		fmt.Sprintf("%016x", bits.ReverseBytes64(r+4))}
	for _, l := range hors {
		sort.Strings(l)
		sort.Strings(attendu)
		if strings.Join(l, " ") != strings.Join(attendu, " ") {
			t.Fatalf("attendu %v, lu %v", attendu, l)
		}
	}
}

// TestLeGardeRailLitLesGzIgnoreLesFlottantsEtLesDossiersNonSuivis : un `.json.gz` est lu decompresse, une
// partie decimale de seize chiffres n est pas un xuid, un dossier `node_modules` n est pas balaye, et un
// nombre nomme dans la liste des non-xuids n est pas signale (lui seul).
func TestLeGardeRailLitLesGzIgnoreLesFlottantsEtLesDossiersNonSuivis(t *testing.T) {
	dossier := t.TempDir()
	const reel, flottant = "2533274790395904", "0.2533274790395905"
	base, _ := strconv.ParseUint(reel, 10, 64)
	nomme := strconv.FormatUint(base+2, 10) // construit : un litteral serait signale par le balayage de ce fichier
	var tampon bytes.Buffer
	zw := gzip.NewWriter(&tampon)
	if _, err := zw.Write([]byte(`{"xuid":"` + reel + `"}`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	ecrire(t, filepath.Join(dossier, "a.json.gz"), tampon.String())
	ecrire(t, filepath.Join(dossier, "b.ts"), "const v = "+flottant+"\n")
	ecrire(t, filepath.Join(dossier, "node_modules", "c.json"), reel)
	ecrire(t, filepath.Join(dossier, "d.png.gz"), tampon.String())
	ecrire(t, filepath.Join(dossier, "e.txt"), "nomme "+nomme+"\n")
	nombresHorsXuid[nomme] = "nombre de test"
	t.Cleanup(func() { delete(nombresHorsXuid, nomme) })
	hors, err := xuidsHorsRegistre(dossier, map[uint64]bool{}, extensionsTexteApps)
	if err != nil {
		t.Fatal(err)
	}
	if len(hors) != 1 || len(hors[filepath.ToSlash(filepath.Join(dossier, "a.json.gz"))]) != 1 {
		t.Fatalf("attendu un seul signalement, dans a.json.gz : %v", hors)
	}
}
