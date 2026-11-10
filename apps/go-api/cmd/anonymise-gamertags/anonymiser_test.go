package anonymisegamertags

// anonymiser_test.go — LA PORTE : inventaire, ou reecriture sous GAMERTAGS_ECRIRE=1.

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/briques"
)

// racineDuDepot : ce paquet vit sous apps/go-api/cmd/anonymise-gamertags.
var racineDuDepot = filepath.Join("..", "..", "..", "..")

// bilanFichier : ce qu un fichier contenait.
type bilanFichier struct {
	chemin string
	forme  forme
	compte map[string]int
}

func (b bilanFichier) total() int {
	n := 0
	for _, v := range b.compte {
		n += v
	}
	return n
}

// estForme dit si une cle du compte designe une forme de xuid (sinon, un nom de joueur).
func estForme(cle string) bool { return strings.HasPrefix(cle, "xuid-") }

// TestAnonymiserFixtures : la porte. Sans table de noms ni liste de xuids, elle ne fait rien (CI
// comprise).
func TestAnonymiserFixtures(t *testing.T) {
	cheminTable := os.Getenv("GAMERTAGS_CORRESPONDANCE")
	cheminXuids := os.Getenv("XUIDS_ORIGINAUX")
	if cheminTable == "" && cheminXuids == "" {
		t.Skip("GAMERTAGS_CORRESPONDANCE et XUIDS_ORIGINAUX non definis : porte d anonymisation des fixtures fermee")
	}
	var ps []paire
	if cheminTable != "" {
		ps = lireFichier(t, cheminTable, lireCorrespondance)
	}
	var xs []paireXuid
	if cheminXuids != "" {
		xs = lireFichier(t, cheminXuids, lireXuidsOriginaux)
	}
	j := nouveauJeu(ps, xs)
	ecrire := os.Getenv("GAMERTAGS_ECRIRE") == "1"
	if sortie := os.Getenv("XUIDS_TABLE_SORTIE"); sortie != "" {
		ecrireTableXuids(t, sortie, xs)
	}
	fichiers, err := fichiersDesChemins(os.Getenv("GAMERTAGS_CHEMINS"))
	if err != nil {
		t.Fatal(err)
	}
	var bilans []bilanFichier
	for _, chemin := range fichiers {
		b, err := traiterFichier(chemin, j, ecrire)
		if err != nil {
			t.Fatalf("%s : %v", chemin, err)
		}
		if b.total() > 0 {
			bilans = append(bilans, b)
		}
	}
	t.Fatalf("%s", rendreBilan(bilans, len(fichiers), j, ecrire))
}

// ecrireTableXuids ecrit la table `original<TAB>fictif`, HORS DEPOT, que la regeneration des
// fixtures encodees par le codec des faits (`libs/film/replay`, entiers variables) relit : ces
// fixtures ne portent pas le xuid sous une forme que le motif de l outil retrouve.
func ecrireTableXuids(t *testing.T, chemin string, xs []paireXuid) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("# table original -> fictif, HORS DEPOT (XUIDS_TABLE_SORTIE de cmd/anonymise-gamertags)\n")
	for _, p := range xs {
		fmt.Fprintf(&sb, "%d\t%d\n", p.reel, p.fictif)
	}
	if err := os.WriteFile(chemin, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("table des xuids : %v", err)
	}
}

// lireFichier ouvre `chemin` et le lit avec `lire`, ou arrete le test en nommant le fichier.
func lireFichier[T any](t *testing.T, chemin string, lire func(r io.Reader) (T, error)) T {
	t.Helper()
	f, err := os.Open(chemin) //nolint:gosec // chemin fourni par l operateur
	if err != nil {
		t.Fatalf("%s : %v", chemin, err)
	}
	defer func() { _ = f.Close() }()
	v, err := lire(f)
	if err != nil {
		t.Fatalf("%s : %v", chemin, err)
	}
	return v
}

// fichiersDesChemins developpe les chemins (`;`), relatifs a la racine du depot ou absolus.
func fichiersDesChemins(liste string) ([]string, error) {
	var out []string
	for _, c := range strings.Split(liste, ";") {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !filepath.IsAbs(c) {
			c = filepath.Join(racineDuDepot, c)
		}
		err := filepath.WalkDir(c, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				out = append(out, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("parcours de %s : %w", c, err)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("GAMERTAGS_CHEMINS ne designe aucun fichier")
	}
	sort.Strings(out)
	return out, nil
}

// pour rend le jeu applicable a un fichier : les noms marques `donnees` ne se remplacent que dans
// les fichiers de donnees (sous un dossier `testdata`) ; partout ailleurs — code, `go.mod`, docs —
// ils sont aussi un chemin de module, un identifiant, de la prose. Le registre des xuids vus est
// partage.
func (j *jeu) pour(chemin string) *jeu {
	if sousTestdata(chemin) {
		return j
	}
	var noms []paire
	for _, p := range j.noms {
		if !p.donneesSeules {
			noms = append(noms, p)
		}
	}
	return &jeu{noms: noms, xuids: j.xuids, vus: j.vus}
}

// sousTestdata dit si le fichier vit sous un dossier `testdata`.
func sousTestdata(chemin string) bool {
	for _, element := range strings.Split(filepath.ToSlash(chemin), "/") {
		if element == "testdata" {
			return true
		}
	}
	return false
}

// traiterFichier inventorie un fichier, et le reecrit si demande et s il change.
func traiterFichier(chemin string, j *jeu, ecrire bool) (bilanFichier, error) {
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours demande
	if err != nil {
		return bilanFichier{}, err
	}
	sortie, compte, frm, err := anonymiserOctets(brut, j.pour(chemin))
	if err != nil {
		return bilanFichier{}, err
	}
	b := bilanFichier{chemin: chemin, forme: frm, compte: compte}
	if ecrire && b.total() > 0 {
		info, err := os.Stat(chemin)
		if err != nil {
			return bilanFichier{}, err
		}
		if err := os.WriteFile(chemin, sortie, info.Mode().Perm()); err != nil {
			return bilanFichier{}, err
		}
	}
	return b, nil
}

// rendreBilan : une ligne par fichier touche, le sous-total de chaque dossier, puis le total par
// forme. Un xuid de la liste que nul fichier ne portait est compte a part : c est lui qui explique
// un ecart avec une mesure faite par ailleurs.
func rendreBilan(bilans []bilanFichier, nFichiers int, j *jeu, ecrire bool) string {
	verbe := "INVENTAIRE (rien ecrit ; GAMERTAGS_ECRIRE=1 pour reecrire)"
	if ecrire {
		verbe = "REECRITURE"
	}
	noms := map[string]bool{}
	parForme := map[string]int{}
	parDossier := map[string]map[string]int{}
	var sb strings.Builder
	total := 0
	for _, b := range bilans {
		fmt.Fprintf(&sb, "\n  %-5s %6d occurrence(s)  %s", b.forme, b.total(), filepath.ToSlash(b.chemin))
		dossier := filepath.ToSlash(filepath.Dir(b.chemin))
		if parDossier[dossier] == nil {
			parDossier[dossier] = map[string]int{}
		}
		for cle, n := range b.compte {
			cible := cle
			if !estForme(cle) {
				noms[cle] = true
				cible = "noms"
			}
			parForme[cible] += n
			parDossier[dossier][cible] += n
		}
		total += b.total()
	}
	return fmt.Sprintf("%s : %d fichier(s) parcourus, %d contiennent des originaux ; %d occurrence(s) ; "+
		"%d xuid(s) d origine distinct(s) vus sur %d de la liste ; %d nom(s) distinct(s) sur %d dans la table%s%s%s",
		verbe, nFichiers, len(bilans), total, len(j.vus), len(j.xuids), len(noms), len(j.noms),
		"\n\nPar forme :"+sousTotaux(parForme), "\n\nPar dossier :"+sousTotauxDossiers(parDossier), sb.String())
}

func sousTotaux(m map[string]int) string {
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	var sb strings.Builder
	for _, k := range cles {
		fmt.Fprintf(&sb, "\n  %-28s %6d", k, m[k])
	}
	return sb.String()
}

func sousTotauxDossiers(m map[string]map[string]int) string {
	dossiers := make([]string, 0, len(m))
	for d := range m {
		dossiers = append(dossiers, d)
	}
	sort.Strings(dossiers)
	var sb strings.Builder
	for _, d := range dossiers {
		fmt.Fprintf(&sb, "\n  %s", d)
		for _, ligne := range strings.Split(strings.TrimPrefix(sousTotaux(m[d]), "\n"), "\n") {
			fmt.Fprintf(&sb, "\n    %s", ligne)
		}
	}
	return sb.String()
}

// dossierDeLaBrique rend le dossier de la brique du decodeur de film, tire de l'outil Go par le resolveur
// des briques : le registre des fictifs et les fichiers a balayer y sont, quel que soit l'endroit ou la
// brique est posee.
func dossierDeLaBrique(t *testing.T) string {
	t.Helper()
	dir, err := briques.Dir("film")
	if err != nil {
		t.Fatalf("dossier de la brique du decodeur : %v", err)
	}
	return dir
}
