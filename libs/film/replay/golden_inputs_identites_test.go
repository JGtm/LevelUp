package replay

// golden_inputs_identites_test.go — LES XUIDS DES FIXTURES D'ENTREES : FICTIFS, ET REECRITS PAR LE CODEC.
//
// Les `testdata/inputs_<short8>.bin.gz` portent les identites du film (morts, table des index de
// joueur, table des sieges) en ENTIERS VARIABLES du codec des faits : le motif binaire d'un xuid ne
// s'y retrouve pas, et la table des index y est ECRITE TRIEE par xuid, si bien qu'une substitution
// en place laisserait un fichier que le codec ne reecrirait pas a l'identique
// (`TestGoldenInputsRoundTrip`). Une fixture se reecrit donc par DECODAGE, remplacement des xuids
// dans la structure, puis REENCODAGE.
//
// Deux tests : le garde-rail (toujours actif) lit chaque fixture et refuse un xuid hors du registre
// des identifiants fictifs `testdata/IDENTITES.tsv` ; la porte de reecriture (ouverte par
// REPLAY_IDENTITES_TABLE) applique la table `original<TAB>fictif` que produit
// `cmd/anonymise-gamertags` (XUIDS_TABLE_SORTIE, hors depot). Comme les autres portes de
// regeneration du paquet, elle ECHOUE TOUJOURS en nommant ce qu'elle a ecrit.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// tableIdentitesEnv porte le chemin de la table `original<TAB>fictif` (ouvre la porte de reecriture).
const tableIdentitesEnv = "REPLAY_IDENTITES_TABLE"

// registreIdentitesFictives lit les xuids du registre versionne (premiere colonne).
func registreIdentitesFictives(t *testing.T) map[uint64]bool {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "testdata", "IDENTITES.tsv")) //nolint:gosec // chemin fixe
	if err != nil {
		t.Fatalf("registre des identifiants fictifs : %v", err)
	}
	defer func() { _ = f.Close() }()
	out := map[uint64]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		champ, _, _ := strings.Cut(l, "\t")
		v, err := strconv.ParseUint(champ, 10, 64)
		if err != nil {
			t.Fatalf("registre : %q n'est pas un xuid", champ)
		}
		out[v] = true
	}
	if len(out) == 0 {
		t.Fatal("registre des identifiants fictifs vide")
	}
	return out
}

// xuidsDesEntrees rend tous les xuids qu'une fixture d'entrees porte.
func xuidsDesEntrees(g *FilmFacts) []uint64 {
	var out []uint64
	for _, d := range g.Deaths {
		out = append(out, d.XUID)
	}
	for x := range g.PlayerIndices.ByXUID {
		out = append(out, x)
	}
	for _, s := range g.FilmTable.Seats {
		out = append(out, s.XUID)
	}
	return out
}

// TestGoldenInputsPorteDesXuidsFictifs : aucune fixture d'entrees ne porte un xuid hors registre.
func TestGoldenInputsPorteDesXuidsFictifs(t *testing.T) {
	registre := registreIdentitesFictives(t)
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, _ := chargerGoldenBuild(t, b)
			xuids := xuidsDesEntrees(g)
			if len(xuids) == 0 {
				t.Skip("fixture sans identite")
			}
			for _, x := range xuids {
				if !registre[x] {
					t.Fatalf("%s : le xuid %d n'est pas au registre des identifiants fictifs "+
						"(%s=<table> go test -run GoldenInputsReecrire)", b.inputsPath(), x, tableIdentitesEnv)
				}
			}
		})
	}
}

// lireTableIdentites lit la table `original<TAB>fictif`.
func lireTableIdentites(t *testing.T, chemin string) map[uint64]uint64 {
	t.Helper()
	f, err := os.Open(chemin) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		t.Fatalf("table des identites : %v", err)
	}
	defer func() { _ = f.Close() }()
	out := map[uint64]uint64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		a, b, ok := strings.Cut(l, "\t")
		reel, errA := strconv.ParseUint(a, 10, 64)
		fictif, errB := strconv.ParseUint(b, 10, 64)
		if !ok || errA != nil || errB != nil {
			t.Fatalf("table des identites : ligne %q", l)
		}
		out[reel] = fictif
	}
	if len(out) == 0 {
		t.Fatal("table des identites vide")
	}
	return out
}

// remplacerXuidsDesEntrees applique la table aux morts, a la table des index et a celle des sieges ;
// un xuid que la table ne connait pas reste tel quel (le garde-rail le refusera ensuite). Rend le
// nombre de xuids remplaces.
func remplacerXuidsDesEntrees(g *FilmFacts, table map[uint64]uint64) int {
	n := 0
	chg := func(x uint64) uint64 {
		if f, ok := table[x]; ok {
			n++
			return f
		}
		return x
	}
	for i := range g.Deaths {
		g.Deaths[i].XUID = chg(g.Deaths[i].XUID)
	}
	index := make(map[uint64]int, len(g.PlayerIndices.ByXUID))
	for x, i := range g.PlayerIndices.ByXUID {
		index[chg(x)] = i
	}
	g.PlayerIndices.ByXUID = index
	for i := range g.FilmTable.Seats {
		g.FilmTable.Seats[i].XUID = chg(g.FilmTable.Seats[i].XUID)
	}
	return n
}

// TestGoldenInputsReecrireIdentites : LA PORTE DE REECRITURE des xuids des fixtures d'entrees.
func TestGoldenInputsReecrireIdentites(t *testing.T) {
	chemin := os.Getenv(tableIdentitesEnv)
	if chemin == "" {
		t.Skip("reecriture des xuids des fixtures d'entrees : " + tableIdentitesEnv + " non defini")
	}
	table := lireTableIdentites(t, chemin)
	var ecrits []string
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		n := remplacerXuidsDesEntrees(g, table)
		var buf bytes.Buffer
		zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			t.Fatalf("gzip : %v", err)
		}
		if _, err := zw.Write(EncodeFilmFacts(g)); err != nil {
			t.Fatalf("gzip : %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("gzip : %v", err)
		}
		if err := os.WriteFile(b.inputsPath(), buf.Bytes(), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", b.inputsPath(), err)
		}
		ecrits = append(ecrits, fmt.Sprintf("%s (%d xuid(s))", b.inputsPath(), n))
	}
	t.Fatalf("%d fixture(s) d'entrees reecrite(s) : %s ; relancer sans %s pour verifier",
		len(ecrits), strings.Join(ecrits, ", "), tableIdentitesEnv)
}
