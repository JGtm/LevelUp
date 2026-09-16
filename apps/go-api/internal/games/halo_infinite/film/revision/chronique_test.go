package revision_test

// chronique_test.go — LE LECTEUR DE CHRONIQUE, SUR DU SYNTHETIQUE ET SUR LE REEL.
//
// # POURQUOI LES DEUX
//
// Le synthetique prouve que les REFUS mordent (un trou, un doublon, une forme fausse) : on ne
// peut pas fabriquer ces cas dans les artefacts versionnes sans les casser. Le reel prouve que
// le lecteur lit la forme QUI EXISTE — un parseur valide sur ses propres fixtures et faux sur le
// fichier du depot est le defaut classique des lecteurs de goldens.
//
// # CE QUI N EST PAS VERIFIE SUR LE REEL, ET LA MESURE QUI LE DIT
//
// [revision.Chronique.VerifierRangs] n est PAS applique a la chronique de `grammar` : mesure du
// 2026-09-17 sur `grammar_rev.golden`, la serie du 2026-09-15 saute `.7` puis `.9` a `.11` — des
// rangs reserves par des lots paralleles dont la fusion n a pas eu lieu. Poser ici une
// continuite sur la chronique d une AUTRE couche rendrait ce paquet rouge a la prochaine fusion
// a rang provisoire, pour une decision qui ne lui appartient pas. C est la couche qui declare
// son plancher (`depuis`), au lot 2.6.1.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

func TestParserRevisionAccepteEtRefuse(t *testing.T) {
	for _, cas := range []struct {
		rev      string
		date     string
		n        int
		accepte  bool
		pourquoi string
	}{
		{rev: "grammar-2026-09-14", date: "2026-09-14", n: 1, accepte: true,
			pourquoi: "le premier lot du jour s ecrit sans suffixe"},
		{rev: "grammar-2026-09-15.27", date: "2026-09-15", n: 27, accepte: true,
			pourquoi: "le rang de journee separe deux lots du meme jour (D1 (1.2))"},
		{rev: "grammar-2026-09-14.1", accepte: false,
			pourquoi: "`.1` est une seconde ecriture du rang 1 : une position ne s ecrit pas de deux facons"},
		{rev: "grammar-2026-09-14.0", accepte: false, pourquoi: "rang de journee nul"},
		{rev: "facts-2026-09-14", accepte: false, pourquoi: "prefixe d une autre couche"},
		{rev: "grammar-2026-9-14", accepte: false, pourquoi: "date non ISO"},
		{rev: "grammar-2026-09-14.x", accepte: false, pourquoi: "rang non numerique"},
	} {
		rang, err := revision.ParserRevision("grammar", cas.rev)
		switch {
		case cas.accepte && err != nil:
			t.Errorf("%s refusee (%v) — %s", cas.rev, err, cas.pourquoi)
		case cas.accepte && (rang.Date != cas.date || rang.N != cas.n):
			t.Errorf("%s lue %v, attendu {%s %d}", cas.rev, rang, cas.date, cas.n)
		case !cas.accepte && err == nil:
			t.Errorf("%s acceptee — %s", cas.rev, cas.pourquoi)
		}
	}
}

func TestRangSuitEstLaContinuite(t *testing.T) {
	r := func(s string) revision.Rang {
		t.Helper()
		rang, err := revision.ParserRevision("grammar", s)
		if err != nil {
			t.Fatalf("%s : %v", s, err)
		}
		return rang
	}
	for _, cas := range []struct {
		precedent, suivant string
		suit               bool
	}{
		{"grammar-2026-09-14", "grammar-2026-09-14.2", true},
		{"grammar-2026-09-14.2", "grammar-2026-09-14.3", true},
		{"grammar-2026-09-14.3", "grammar-2026-09-15", true},
		{"grammar-2026-09-14.2", "grammar-2026-09-14.4", false},
		{"grammar-2026-09-14.2", "grammar-2026-09-14.2", false},
		{"grammar-2026-09-15", "grammar-2026-09-14.2", false},
		{"grammar-2026-09-14.3", "grammar-2026-09-15.2", false},
	} {
		if got := r(cas.suivant).Suit(r(cas.precedent)); got != cas.suit {
			t.Errorf("%s apres %s : suit=%v, attendu %v", cas.suivant, cas.precedent, got, cas.suit)
		}
	}
}

// ecrireChroniqueSynthetique pose un godoc et un golden dans un dossier temporaire.
func ecrireChroniqueSynthetique(t *testing.T, entrees []string, donnees []string) (godoc, golden string) {
	t.Helper()
	dir := t.TempDir()
	godoc = filepath.Join(dir, "rev.go")
	golden = filepath.Join(dir, "rev.golden")
	var b strings.Builder
	b.WriteString("package couche\n\n")
	for _, rev := range entrees {
		b.WriteString("// ENTREE `" + rev + "` (2026-09-17, lot de test) : une ligne de prose.\n")
	}
	b.WriteString("\nconst Rev = \"" + entrees[len(entrees)-1] + "\"\n")
	if err := os.WriteFile(godoc, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture du godoc : %v", err)
	}
	lignes := []string{"# GOLDEN DE TEST — la prose est conservee.", ""}
	lignes = append(lignes, donnees...)
	if err := os.WriteFile(golden, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("ecriture du golden : %v", err)
	}
	return godoc, golden
}

func TestChroniqueLitLesDeuxCotesEtRendLaCourante(t *testing.T) {
	godoc, golden := ecrireChroniqueSynthetique(t,
		[]string{"couche-2026-09-16", "couche-2026-09-17", "couche-2026-09-17.2"},
		[]string{"couche-2026-09-16\taaa", "couche-2026-09-17\tbbb", "couche-2026-09-17.2\tccc"})
	c, err := revision.LireChronique("couche", godoc, golden)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(c.Godoc) != 3 || len(c.Golden) != 3 {
		t.Fatalf("lu %d entrees de godoc et %d lignes de golden, attendu 3 et 3", len(c.Godoc), len(c.Golden))
	}
	if c.Godoc[0].Date != "2026-09-17" || c.Godoc[0].Lot != "lot de test" {
		t.Errorf("entete d entree mal lue : %+v", c.Godoc[0])
	}
	if courante := c.Courante(); courante.Revision != "couche-2026-09-17.2" || courante.Empreinte != "ccc" {
		t.Errorf("courante = %+v, attendu la DERNIERE ligne de donnees", courante)
	}
	if err := c.VerifierCouverture("couche-2026-09-17.2"); err != nil {
		t.Errorf("couverture refusee pour la revision courante : %v", err)
	}
	if err := c.VerifierRangs(""); err != nil {
		t.Errorf("rangs refuses sur une suite continue : %v", err)
	}
}

func TestChroniqueRefuseCeQuiDoitLEtre(t *testing.T) {
	// 1. La revision courante sans entree de godoc.
	godoc, golden := ecrireChroniqueSynthetique(t,
		[]string{"couche-2026-09-16"},
		[]string{"couche-2026-09-16\taaa", "couche-2026-09-17\tbbb"})
	c, err := revision.LireChronique("couche", godoc, golden)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if err := c.VerifierCouverture("couche-2026-09-17"); err == nil {
		t.Error("couverture acceptee pour une revision SANS entree de godoc (constat F5)")
	}
	// 2. Une courante qui n est pas en queue de golden : le golden n a pas ete regenere.
	if err := c.VerifierCouverture("couche-2026-09-16"); err == nil {
		t.Error("couverture acceptee pour une revision qui n est pas la derniere du golden")
	}
	// 3. Un TROU dans les rangs.
	godocTrou, goldenTrou := ecrireChroniqueSynthetique(t,
		[]string{"couche-2026-09-17", "couche-2026-09-17.4"},
		[]string{"couche-2026-09-17\taaa", "couche-2026-09-17.4\tbbb"})
	cTrou, err := revision.LireChronique("couche", godocTrou, goldenTrou)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if err := cTrou.VerifierRangs(""); err == nil {
		t.Error("rangs acceptes malgre un trou (.2 et .3 sautes)")
	}
	// 4. Le PLANCHER : la meme chronique, lue a partir du rang qui suit le trou, est acceptee.
	if err := cTrou.VerifierRangs("couche-2026-09-17.4"); err != nil {
		t.Errorf("le plancher ne neutralise pas un trou anterieur : %v", err)
	}
}

func TestChroniqueRefuseUnGoldenMalforme(t *testing.T) {
	dir := t.TempDir()
	godoc := filepath.Join(dir, "rev.go")
	if err := os.WriteFile(godoc, []byte("package couche\n"), 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	for nom, contenu := range map[string]string{
		"sans ligne de donnees": "# que de la prose\n",
		"sans tabulation":       "couche-2026-09-17 aaa\n",
		"empreinte vide":        "couche-2026-09-17\t\n",
	} {
		golden := filepath.Join(dir, "g.golden")
		if err := os.WriteFile(golden, []byte(contenu), 0o600); err != nil {
			t.Fatalf("ecriture : %v", err)
		}
		if _, err := revision.LireChronique("couche", godoc, golden); err == nil {
			t.Errorf("golden %s accepte", nom)
		}
	}
	if _, err := revision.LireChronique("couche", filepath.Join(dir, "absent.go"),
		filepath.Join(dir, "g.golden")); err == nil {
		t.Error("godoc absent accepte — un artefact versionne absent est une erreur")
	}
}

// TestChroniqueLitLesArtefactsReelsDeFilmdec : le lecteur lit la forme QUI EXISTE.
//
// Il ne fige aucune valeur : la revision courante est celle du golden, lue au moment du test.
// Ce que ce test prouve est que les deux formes reelles (le `// ENTREE` du godoc, la ligne de
// donnees du golden) sont comprises, et que la revision courante est couverte des deux cotes —
// ce que le gate de `grammar` exige deja par ses propres moyens.
func TestChroniqueLitLesArtefactsReelsDeFilmdec(t *testing.T) {
	api := racineAPI(t)
	dir := filepath.Join(api, "internal", "games", "halo_infinite", "film", "grammar")
	c, err := revision.LireChronique("grammar",
		// Depuis le lot 2.4 (D2 (2.4)), la chronique est ROTATIONNEE : les entrees courantes vivent dans
		// `grammar_rev_chronique.go`, les anciennes dans `grammar_rev_chronique_archive.go`.
		filepath.Join(dir, "grammar_rev_chronique.go"), filepath.Join(dir, "testdata", "grammar_rev.golden"))
	if err != nil {
		t.Fatalf("lecture de la chronique reelle : %v", err)
	}
	if len(c.Godoc) == 0 {
		t.Fatal("aucune entree de godoc lue dans grammar_rev_chronique.go — la forme `// ENTREE` a change")
	}
	courante := c.Courante()
	t.Logf("chronique reelle : %d entrees de godoc, revision courante %s (golden ligne %d)",
		len(c.Godoc), courante.Revision, courante.Ligne)
	if err := c.VerifierCouverture(courante.Revision); err != nil {
		t.Errorf("la revision courante de filmdec n est pas couverte : %v", err)
	}
	if _, err := revision.ParserRevision("grammar", courante.Revision); err != nil {
		t.Errorf("la revision courante de filmdec n a pas la forme attendue : %v", err)
	}
}
