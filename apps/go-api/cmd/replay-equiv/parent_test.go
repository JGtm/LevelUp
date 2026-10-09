package main

// parent_test.go — LA LECTURE DU CORPUS ET LES DEUX VERDICTS DU PARENT.
//
// Le parent ne decode rien : tout ce qu'il fait de sa propre autorite est de lire une liste et
// de nommer un ecart. Les deux se testent sans film.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/digest"
)

func TestLireCorpusIgnoreCommentairesEtColonnes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CORPUS.txt")
	contenu := "# un commentaire | avec des barres\n" +
		"\n" +
		"000d5950 | Cliffhanger | Slayer:Arena Super Fiesta | 28 | 20.2 | temoin\n" +
		"01e1f945 | Catalyst | KOTH:Arena | 30 | 22.5 | zones KOTH\n"
	if err := os.WriteFile(path, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	films, err := lireCorpus(path)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(films) != 2 || films[0] != "000d5950" || films[1] != "01e1f945" {
		t.Fatalf("films attendus [000d5950 01e1f945], obtenus %v", films)
	}
}

func TestLireCorpusRefuseUnFichierSansFilm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CORPUS.txt")
	if err := os.WriteFile(path, []byte("# rien que des commentaires\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := lireCorpus(path); err == nil {
		t.Fatal("un corpus sans film doit echouer : une passe vide passerait pour un succes")
	}
}

// ligne fabrique une ligne de digest.
func ligne(etape string, compte int, sha string) string {
	return etape + "\t" + string(rune('0'+compte)) + "\t" + sha
}

func TestVerifierEtapesNommeCeQuiManque(t *testing.T) {
	attendues := []string{"score", "fire", "artifact"}
	cas := []struct {
		nom     string
		lignes  []string
		fragile string
	}{
		{"etape manquante", []string{ligne("score", 1, "a"), ligne("fire", 2, "b")}, "MANQUANTE"},
		{"ordre inverse", []string{ligne("fire", 2, "b"), ligne("score", 1, "a"), ligne("artifact", 3, "c")}, "attendue"},
		{"etape en trop", []string{ligne("score", 1, "a"), ligne("fire", 2, "b"),
			ligne("artifact", 3, "c"), ligne("zzz", 4, "d")}, "TROP"},
	}
	for _, c := range cas {
		err := verifierEtapes(c.lignes, attendues)
		if err == nil {
			t.Errorf("%s : aucune erreur rendue", c.nom)
			continue
		}
		if !strings.Contains(err.Error(), c.fragile) {
			t.Errorf("%s : message %q, attendu contenant %q", c.nom, err, c.fragile)
		}
	}
	bon := []string{ligne("score", 1, "a"), ligne("fire", 2, "b"), ligne("artifact", 3, "c")}
	if err := verifierEtapes(bon, attendues); err != nil {
		t.Errorf("une liste complete et ordonnee doit passer : %v", err)
	}
}

// TestComparerNommeTOUTESLesEtapesQuiDiffERENT — D2 (pilote), 2026-09-17 : le harnais rendait
// la PREMIERE divergence et s'arretait la, ce qui imposait un decodage complet du film (une a
// trois minutes) PAR etape divergente. Les deux fichiers sont en memoire : la liste complete ne
// coute rien, et elle dit du premier coup si la divergence est locale ou generale.
//
// LA MUTATION QUI LE FAIT ROUGIR : remettre un `return` dans la boucle de `comparer` a la place
// de l'accumulation dans `ecarts`.
func TestComparerNommeTOUTESLesEtapesQuiDiffERENT(t *testing.T) {
	ref := []string{ligne("score", 1, "aaa"), ligne("fire", 2, "bbb"), ligne("artifact", 3, "ccc")}
	obtenu := []string{ligne("score", 1, "aaa"), ligne("fire", 2, "XXX"), ligne("artifact", 3, "YYY")}
	err := comparer(ref, obtenu)
	if err == nil {
		t.Fatal("aucun ecart rendu alors que deux etapes different")
	}
	msg := err.Error()
	// Le message est le PRODUIT de ce harnais : `go test -v` l'affiche tel quel, pour que la
	// forme rendue se relise sans decoder un film.
	t.Logf("message rendu :\n%s", msg)
	for _, veut := range []string{"fire", "artifact", "2 etape(s) sur 3"} {
		if !strings.Contains(msg, veut) {
			t.Errorf("le message doit contenir %q — toutes les etapes divergentes, avec leur compte :\n%s",
				veut, msg)
		}
	}
	if strings.Contains(msg, "score") {
		t.Errorf("une etape IDENTIQUE ne doit pas etre listee :\n%s", msg)
	}
	// Le compte ET le sha des deux cotes, par etape : sans eux, « fire differe » n'apprend rien.
	for _, veut := range []string{"compte=2", "sha=bbb", "sha=XXX", "sha=ccc", "sha=YYY"} {
		if !strings.Contains(msg, veut) {
			t.Errorf("le message doit porter %q (compte et sha attendus/obtenus par etape) :\n%s", veut, msg)
		}
	}
	if err := comparer(ref, ref); err != nil {
		t.Errorf("deux listes identiques ne doivent rien rendre : %v", err)
	}
}

// TestComparerNommeLesEtapesManquantesEtEnTrop — les deux asymetries, elles aussi listees
// toutes, et jamais confondues l'une avec l'autre.
func TestComparerNommeLesEtapesManquantesEtEnTrop(t *testing.T) {
	ref := []string{ligne("score", 1, "aaa"), ligne("fire", 2, "bbb"), ligne("artifact", 3, "ccc")}
	court := []string{ligne("score", 1, "aaa")}
	err := comparer(ref, court)
	if err == nil {
		t.Fatal("deux etapes de la reference non produites : un ecart est attendu")
	}
	for _, veut := range []string{"fire", "artifact", "n'a pas ete produite", "2 etape(s) sur 3"} {
		if !strings.Contains(err.Error(), veut) {
			t.Errorf("le message doit contenir %q :\n%v", veut, err)
		}
	}
	err = comparer(court, ref)
	if err == nil {
		t.Fatal("deux etapes produites en trop : un ecart est attendu")
	}
	if !strings.Contains(err.Error(), "produite en trop") {
		t.Errorf("une etape en trop doit etre nommee comme telle, pas comme manquante :\n%v", err)
	}
}

// TestDossierDesDigestsConserveCeQuOnLuiDemande — `-out-dir` : les TSV des enfants survivent a
// la passe, pour instruire une divergence sans re-decoder le film.
func TestDossierDesDigestsConserveCeQuOnLuiDemande(t *testing.T) {
	demande := filepath.Join(t.TempDir(), "digests", "lot28")
	dir, nettoyer, err := dossierDesDigests(demande)
	if err != nil {
		t.Fatalf("creation du dossier demande : %v", err)
	}
	if dir != demande {
		t.Errorf("dossier = %q, %q attendu", dir, demande)
	}
	temoin := filepath.Join(dir, "000d5950.tsv")
	if err := os.WriteFile(temoin, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nettoyer()
	if _, err := os.Stat(temoin); err != nil {
		t.Fatalf("le TSV de l'enfant devait etre CONSERVE sous --out-dir : %v", err)
	}
}

// TestDossierDesDigestsEffaceLeTemporaire — sans `-out-dir`, rien ne traine : le regime normal
// reste un dossier jetable.
func TestDossierDesDigestsEffaceLeTemporaire(t *testing.T) {
	dir, nettoyer, err := dossierDesDigests("")
	if err != nil {
		t.Fatalf("creation du dossier temporaire : %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "000d5950.tsv"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nettoyer()
	if _, err := os.Stat(dir); err == nil {
		t.Fatalf("le dossier temporaire %s devait etre efface", dir)
	}
}

func TestEtapesAttenduesPorteLesTroisListes(t *testing.T) {
	etapes := etapesAttendues()
	if len(etapes) < 40 {
		t.Fatalf("trop peu d'etapes attendues (%d) : les listes exportees ne sont pas concatenees", len(etapes))
	}
	if etapes[0] != "score" {
		t.Errorf("premiere etape attendue \"score\", obtenue %q", etapes[0])
	}
	if last := etapes[len(etapes)-1]; last != "artifact" {
		t.Errorf("derniere etape attendue \"artifact\", obtenue %q", last)
	}
}

func TestArgsEnfantPorteLaRacineEtLeMode(t *testing.T) {
	o := options{repoRoot: "/depot", titleSlug: "halo_infinite", memGiB: 3}
	args := argsEnfant(o, "000d5950", "/tmp/x.tsv")
	joint := strings.Join(args, " ")
	for _, veut := range []string{"-child", "-film 000d5950", "-out /tmp/x.tsv",
		"-repo-root /depot", "-title halo_infinite", "-mem-gib 3"} {
		if !strings.Contains(joint, veut) {
			t.Errorf("args de l'enfant sans %q : %s", veut, joint)
		}
	}
}

// ecrireTSV pose un fichier de digests avec l'entete demande.
func ecrireTSV(t *testing.T, path, entete string, lignes ...string) {
	t.Helper()
	contenu := strings.Join(append([]string{entete}, lignes...), "\n") + "\n"
	if err := os.WriteFile(path, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestTraiterFilmClasseLaGrammaireEnINFRA — LE MARQUEUR DE VERSION (N-B) : une reference figee
// sous une AUTRE grammaire n'est pas un decodeur qui a change d'avis, c'est le harnais qui ne
// peut pas poser la question. Elle doit rendre `errInfra` et nommer les deux versions.
func TestTraiterFilmClasseLaGrammaireEnINFRA(t *testing.T) {
	dir := t.TempDir()
	sortie := filepath.Join(dir, "enfant.tsv")
	p := passe{dir: dir, attendues: []string{"score"}}
	digestScore := ligne("score", 1, "aaa")
	ecrireTSV(t, sortie, digest.GrammarLine(), digestScore)

	cas := []struct {
		nom      string
		entete   string
		fragiles []string
	}{
		{"grammaire anterieure", "# digest-grammar: 1", []string{"grammaire 1", "-update"}},
		{"aucun marqueur", "score\t9\tzzz", []string{"ligne de grammaire attendue", "-update"}},
	}
	for _, c := range cas {
		ref := filepath.Join(dir, c.nom+".tsv")
		ecrireTSV(t, ref, c.entete, digestScore)
		err := traiterFilm(p, sortie, c.nom)
		if !errors.Is(err, errInfra) {
			t.Errorf("%s : erreur %v — attendue une panne d'INFRA, jamais un ECART", c.nom, err)
			continue
		}
		for _, fragile := range c.fragiles {
			if !strings.Contains(err.Error(), fragile) {
				t.Errorf("%s : message %q, attendu contenant %q", c.nom, err, fragile)
			}
		}
	}

	// Sous la grammaire courante, la comparaison reprend son cours.
	ok := filepath.Join(dir, "courante.tsv")
	ecrireTSV(t, ok, digest.GrammarLine(), digestScore)
	if err := traiterFilm(p, sortie, "courante"); err != nil {
		t.Errorf("reference figee sous la grammaire courante : %v", err)
	}
}

// TestTraiterFilmExigeLeMarqueurDeLEnfant : un fichier d'enfant sans ligne de grammaire est
// illisible pour le harnais — jamais un ecart d'etape.
func TestTraiterFilmExigeLeMarqueurDeLEnfant(t *testing.T) {
	dir := t.TempDir()
	sortie := filepath.Join(dir, "enfant.tsv")
	if err := os.WriteFile(sortie, []byte(ligne("score", 1, "aaa")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := traiterFilm(passe{dir: dir, attendues: []string{"score"}}, sortie, "sans")
	if !errors.Is(err, errInfra) {
		t.Fatalf("erreur %v — attendue une panne d'INFRA", err)
	}
}

// TestTraiterFilmFigeLeMarqueur : `-update` ecrit la reference AVEC sa ligne de grammaire, sans
// quoi la passe suivante la refuserait.
func TestTraiterFilmFigeLeMarqueur(t *testing.T) {
	dir := t.TempDir()
	sortie := filepath.Join(dir, "enfant.tsv")
	ecrireTSV(t, sortie, digest.GrammarLine(), ligne("score", 1, "aaa"))
	p := passe{dir: dir, attendues: []string{"score"}, update: true}
	if err := traiterFilm(p, sortie, "fige"); err != nil {
		t.Fatalf("figeage : %v", err)
	}
	lignes, err := lireLignes(filepath.Join(dir, "fige.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lignes) != 2 || lignes[0] != digest.GrammarLine() {
		t.Fatalf("reference figee = %q, attendue ouverte par %q", lignes, digest.GrammarLine())
	}
	p.update = false
	if err := traiterFilm(p, sortie, "fige"); err != nil {
		t.Errorf("la reference qu'on vient de figer doit se comparer : %v", err)
	}
}
