package main

// parent.go — LE PARENT D'EQUIVALENCE : IL PLANIFIE, IL COMPARE, IL NE DECODE RIEN.
//
// Un enfant par film, sequentiellement, jamais deux films dans un processus (c'est le motif des
// quatre sinistres RAM). Le parent lit le corpus, lance, attend, compare, et nomme la PREMIERE
// etape qui differe — c'est ce qui transforme « quelque chose a change » en « le balayage des
// projectiles a change ».

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/digest"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/replaybuild"
)

// bilan compte ce qu'a donne la passe.
//
// `infra` EST DISTINCT DE `differents`, ET IL LE FAUT : un fichier de digests illisible (celui
// de l'enfant, ou la reference pas encore figee) n'est pas un decodeur qui a change d'avis,
// c'est le harnais qui n'a pas pu poser la question. Les confondre ferait lire « 3 films
// different » a une passe ou trois references manquaient — et enverrait chercher une regression
// qui n'existe pas. Les deux restent fatals pour le code de sortie.
type bilan struct {
	identiques, differents, ecartes, echecs, infra int
}

// errInfra marque une panne de HARNAIS et non un ecart de digest (cf. bilan.infra). Il
// s'enveloppe autour des erreurs de lecture de fichier, jamais autour d'une comparaison.
var errInfra = errors.New("harnais")

// passe porte ce qui NE CHANGE PAS d'un film a l'autre pendant une passe : le dossier des
// references, la liste des etapes attendues, et le regime (comparer ou figer). Les regrouper
// evite de les faire voyager un a un a travers trois signatures.
type passe struct {
	dir       string
	attendues []string
	update    bool
}

// parentEquivalence lance un enfant par film et compare les digests. Code de sortie non nul des
// qu'un film differe, echoue ou meurt.
func parentEquivalence(o options) int {
	films, err := listeDesFilms(o)
	if err != nil {
		fmt.Println("corpus illisible :", err)
		return 1
	}
	runner, err := filmproc.NewRunner(o.repoRoot, os.Stdout)
	if err != nil {
		fmt.Println("lanceur :", err)
		return 1
	}
	tmp, err := os.MkdirTemp("", "replay-equiv")
	if err != nil {
		fmt.Println("dossier temporaire :", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	p := passe{dir: dossierEquivalence(o.repoRoot), attendues: etapesAttendues(), update: o.update}
	fmt.Printf("%d film(s), %d etape(s) attendues par film, reference %s\n",
		len(films), len(p.attendues), p.dir)
	var b bilan
	for _, film := range films {
		sortie := filepath.Join(tmp, film+".tsv")
		res := runner.Run(context.Background(), argsEnfant(o, film, sortie))
		fmt.Printf("%-9s %-12s %9s  pic %5.2f Gio\n",
			film, res.Issue, res.Dur.Round(time.Millisecond), gio(res.Peak))
		compterIssue(&b, p, res, film, sortie)
	}
	fmt.Printf("\nBILAN : %d identique(s), %d different(s), %d ecarte(s), %d echec(s), "+
		"%d illisible(s) (harnais)\n",
		b.identiques, b.differents, b.ecartes, b.echecs, b.infra)
	if b.differents+b.echecs+b.infra > 0 {
		return 1
	}
	return 0
}

// compterIssue traduit l'issue d'un enfant en ligne de bilan, et compare quand il a reussi.
//
// UN FILM ECARTE N'EST PAS FATAL : une carte hors catalogue de bornes est un refus VOULU du
// constructeur, pas une regression du decodeur. Il est rapporte et compte a part.
func compterIssue(b *bilan, p passe, res filmproc.Result, film, sortie string) {
	switch res.Issue {
	case filmproc.IssueSkipped:
		b.ecartes++
		fmt.Printf("  %s ECARTE (carte hors catalogue de bornes) — aucune comparaison\n", film)
		return
	case filmproc.IssueOK:
	default:
		b.echecs++
		fmt.Printf("  %s ECHEC (code %d)", film, res.Code)
		if res.Err != nil {
			fmt.Printf(" : %v", res.Err)
		}
		fmt.Println()
		return
	}
	switch err := traiterFilm(p, sortie, film); {
	case err == nil:
	case errors.Is(err, errInfra):
		b.infra++
		fmt.Printf("  %s ILLISIBLE : %v\n", film, err)
		return
	default:
		b.differents++
		fmt.Printf("  %s : %v\n", film, err)
		return
	}
	b.identiques++
	if p.update {
		fmt.Printf("  %s : digests FIGES\n", film)
		return
	}
	fmt.Printf("  %s : identique\n", film)
}

// argsEnfant construit la ligne de commande d'un enfant.
//
// ELLE PORTAIT UN `extra []string` pour les drapeaux propres a un mode : le seul consommateur
// etait `-walkers`, retire au lot 6. Un parametre qu'un seul appelant renseigne a `nil` ne
// documente plus rien — il se supprime avec le mode.
func argsEnfant(o options, film, sortie string) []string {
	return []string{
		"-child", "-film", film, "-out", sortie,
		"-repo-root", o.repoRoot, "-title", o.titleSlug,
		"-mem-gib", strconv.Itoa(o.memGiB),
	}
}

// etapesAttendues : toutes les etapes de BuildBytes et de BuildFromFilm, DANS L'ORDRE.
//
// Elle est construite depuis les listes EXPORTEES par le code de production : un balayage
// ajoute la-bas devient une etape attendue ici, sans que personne ait a y penser.
func etapesAttendues() []string {
	return slices.Concat(
		replaybuild.BuildBytesStepsBefore,
		replay.BuildFromFilmSteps,
		replaybuild.BuildBytesStepsAfter,
	)
}

// traiterFilm verifie la grammaire et les etapes, puis compare (ou fige) les digests d'un film.
//
// Les lectures de fichier et la LIGNE DE GRAMMAIRE rendent `errInfra` : elles disent que la
// question n'a pas pu etre posee, pas que la reponse a change (cf. bilan.infra).
func traiterFilm(p passe, sortie, film string) error {
	brut, err := lireLignes(sortie)
	if err != nil {
		return fmt.Errorf("%w : digests de l'enfant illisibles : %v", errInfra, err)
	}
	// L'enfant est LE MEME BINAIRE que le parent (le lanceur reexecute os.Executable) : sa
	// version ne peut donc pas differer de la notre, seule la PRESENCE du marqueur se verifie.
	_, obtenu, err := detacherGrammaire(brut)
	if err != nil {
		return fmt.Errorf("%w : digests de l'enfant : %v", errInfra, err)
	}
	if err := verifierEtapes(obtenu, p.attendues); err != nil {
		return err
	}
	ref := filepath.Join(p.dir, film+".tsv")
	if p.update {
		contenu := strings.Join(append([]string{digest.GrammarLine()}, obtenu...), "\n") + "\n"
		if err := os.WriteFile(ref, []byte(contenu), 0o600); err != nil {
			return fmt.Errorf("%w : ecriture de la reference %s : %v", errInfra, ref, err)
		}
		return nil
	}
	brutRef, err := lireLignes(ref)
	if err != nil {
		return fmt.Errorf("%w : digests de reference illisibles (%s) : %v — figer d'abord avec -update",
			errInfra, ref, err)
	}
	attendu, err := verifierGrammaireReference(brutRef, ref)
	if err != nil {
		return err
	}
	return comparer(attendu, obtenu)
}

// detacherGrammaire retire la ligne de grammaire en tete et rend la version qu'elle porte.
func detacherGrammaire(lignes []string) (int, []string, error) {
	if len(lignes) == 0 {
		return 0, nil, errors.New("fichier vide")
	}
	version, ok := digest.ParseGrammarLine(lignes[0])
	if !ok {
		return 0, nil, fmt.Errorf("premiere ligne %q : ligne de grammaire attendue (%q)",
			lignes[0], digest.GrammarLine())
	}
	return version, lignes[1:], nil
}

// verifierGrammaireReference exige que la reference ait ete figee sous LA GRAMMAIRE COURANTE, et
// rend ses lignes de digest.
//
// C'EST UNE PANNE D'INFRASTRUCTURE, PAS UN ECART. Une reference figee sous une autre grammaire
// n'a rien a dire du decodeur : ses empreintes sont celles d'un AUTRE rendu de la meme valeur.
// Les compter en « different » enverrait chercher une regression qui n'existe pas — c'est
// exactement ce qui est arrive le 2026-09-02 (six TSV sur neuf restes sous la v1).
func verifierGrammaireReference(brut []string, ref string) ([]string, error) {
	version, lignes, err := detacherGrammaire(brut)
	if err != nil {
		return nil, fmt.Errorf("%w : reference %s : %v — re-figer par -update", errInfra, ref, err)
	}
	if version != digest.GrammarVersion {
		return nil, fmt.Errorf(
			"%w : references figees sous la grammaire %d, harnais en %d : re-figer par -update (%s)",
			errInfra, version, digest.GrammarVersion, ref)
	}
	return lignes, nil
}

// verifierEtapes exige que le fichier porte TOUTES les etapes, dans l'ordre et sans surplus.
//
// C'est ce qui empeche une equivalence VACUANTE : un fichier ampute de la moitie de ses
// balayages se comparerait « identique » a un autre fichier ampute.
func verifierEtapes(lignes, attendues []string) error {
	noms := make([]string, 0, len(lignes))
	for _, l := range lignes {
		noms = append(noms, champ(l, 0))
	}
	if slices.Equal(noms, attendues) {
		return nil
	}
	for i, veut := range attendues {
		if i >= len(noms) {
			return fmt.Errorf("etape MANQUANTE : %q (le fichier s'arrete apres %d etapes)", veut, len(noms))
		}
		if noms[i] != veut {
			return fmt.Errorf("etape %d : attendue %q, obtenue %q", i+1, veut, noms[i])
		}
	}
	return fmt.Errorf("%d etape(s) en TROP apres %q", len(noms)-len(attendues), attendues[len(attendues)-1])
}

// comparer nomme la PREMIERE ligne qui differe.
func comparer(attendu, obtenu []string) error {
	for i := range max(len(attendu), len(obtenu)) {
		switch {
		case i >= len(obtenu):
			return fmt.Errorf("ECART : l'etape %q de la reference n'a pas ete produite", champ(attendu[i], 0))
		case i >= len(attendu):
			return fmt.Errorf("ECART : etape %q produite en trop (absente de la reference)", champ(obtenu[i], 0))
		case attendu[i] != obtenu[i]:
			return fmt.Errorf("ECART a l'etape %q : attendu compte=%s sha=%s, obtenu compte=%s sha=%s",
				champ(obtenu[i], 0), champ(attendu[i], 1), champ(attendu[i], 2),
				champ(obtenu[i], 1), champ(obtenu[i], 2))
		}
	}
	return nil
}

// champ rend la n-ieme colonne d'une ligne TSV, vide si elle manque.
func champ(ligne string, n int) string {
	cols := strings.Split(ligne, "\t")
	if n >= len(cols) {
		return ""
	}
	return cols[n]
}

// listeDesFilms rend les films a traiter : `-films` s'il est pose, le corpus sinon.
func listeDesFilms(o options) ([]string, error) {
	if films := filmsDemandes(o.films); len(films) > 0 {
		return films, nil
	}
	return lireCorpus(o.corpus)
}

// lireCorpus lit un fichier de corpus : premier champ avant « | », lignes `#` ignorees.
func lireCorpus(path string) ([]string, error) {
	lignes, err := lireLignes(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range lignes {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, strings.TrimSpace(strings.Split(l, "|")[0]))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("corpus vide : %s", path)
	}
	return out, nil
}

// lireLignes lit un fichier texte et rend ses lignes non vides de fin.
func lireLignes(path string) ([]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit par le parent ou fourni par l'operateur
	if err != nil {
		return nil, err
	}
	lignes := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	for len(lignes) > 0 && strings.TrimSpace(lignes[len(lignes)-1]) == "" {
		lignes = lignes[:len(lignes)-1]
	}
	return lignes, nil
}
