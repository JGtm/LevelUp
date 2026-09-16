package archlint

// no_ad_hoc_source_fingerprint_test.go — LE GARDE-RAIL DE LA CENTRALISATION DES EMPREINTES DE
// SOURCES (lot 2.6.0, 2026-09-17).
//
// # POURQUOI IL EXISTE, ET POURQUOI EN MEME TEMPS QUE LE HELPER
//
// CLAUDE.md regle 6 : a la troisieme copie d un motif, on centralise ET on pose un garde-rail —
// « une factorisation sans garde-rail re-diverge » (lecon mesuree : un predicat bot passe de 8 a
// 36 copies APRES sa centralisation). Le lot 2.6.0 centralise le calcul d empreinte de sources
// dans `film/revision` parce que la decision V15 (11) en demande QUATRE, une par couche. Sans ce
// test, rien n empecherait la cinquieme d etre ecrite a la main — et une empreinte ecrite a la
// main est exactement ce qui a produit les deux cadres divergents d aujourd hui (chemin relatif
// a la racine d un cote, prefixe du nom de dossier de l autre).
//
// # CE QU IL DETECTE
//
// Un fichier qui reunit les TROIS marqueurs du motif : un hachage `sha256`, un parcours
// d arborescence (`filepath.Walk`/`WalkDir`) et le filtre des sources Go (`"_test.go"`). Les
// trois ensemble ne decrivent qu une chose : recalculer une empreinte de sources a la main.
//
// La conjonction est ce qui rend la mesure sure dans les deux sens. Mesure du 2026-09-17 sur
// `internal/` + `cmd/` : `internal/ops/media_store.go` hache par `sha256` en parcourant des
// fichiers, mais ne filtre aucune source Go — c est un hachage de MEDIAS, et il n est pas vise.
// Les commentaires sont retires avant la mesure (`stripComments`) : une explication qui cite
// `sha256` n est pas une reimplementation.
//
// # CE QU IL NE DETECTE PAS, ET C EST ASSUME
//
// Une copie ecrite avec un autre hachage (`sha1`, `fnv`) ou un parcours maison par `os.ReadDir`
// recursif passerait. Ce garde-rail attrape LA COPIE DU MOTIF EXISTANT — celui que les quatre
// couches auraient recopie — pas toute facon imaginable de hacher un arbre. Un ratchet qui
// pretendrait couvrir l espace entier des reecritures serait un ratchet a faux positifs, et un
// faux positif sur `archlint` se paie en amendement reflexe d allowlist.

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const (
	// paquetCanoniqueEmpreinte : LE mecanisme, relatif a `apps/go-api`. Tout ce qui vit dessous
	// est par definition hors de portee — c est l implementation.
	paquetCanoniqueEmpreinte = "internal/games/halo_infinite/film/revision"
	// plancherFichiersEmpreinte : LE PLANCHER CONTRE UN BALAYAGE MUET. 5 553 fichiers `.go`
	// mesures le 2026-09-17 dans `internal/` + `cmd/` (`find internal cmd -name '*.go' | wc -l`) ;
	// un balayage qui en rend nettement moins n a pas trouve l arborescence et ne garde plus rien.
	plancherFichiersEmpreinte = 4500
)

// marqueursEmpreinteDeSources : les trois marqueurs, TOUS requis.
var marqueursEmpreinteDeSources = []string{"sha256", "filepath.Walk", `"_test.go"`}

// empreinteAdHocToleree : un mecanisme ANTERIEUR a la centralisation, avec la date de sa mise en
// table, le lot qui le fait heriter et la forme de la reprise. Ce n est pas un blanc-seing :
// c est la liste de travail de 2.6.1, et le test refuse toute entree devenue sans objet
// (`TestAllowlistDesEmpreintesAdHocNEstPasPerimee`).
type empreinteAdHocToleree struct {
	fichier string
	pose    string
	lot     string
	reprise string
}

// empreintesAdHocTolerees — LES DEUX MECANISMES MESURES LE 2026-09-17 sur `26cf32399`.
//
// L equivalence de chacun avec `revision` est DEJA PROUVEE, au bit pres, par
// `film/revision/equivalence_test.go` : la reprise de 2.6.1 ne renumerote aucune revision.
var empreintesAdHocTolerees = []empreinteAdHocToleree{
	{
		fichier: "internal/games/halo_infinite/film/grammar/grammar_rev_fingerprint_test.go",
		pose:    "2026-09-17", lot: "2.6.1",
		reprise: "`grammar.Rev` passe par `revision.Empreinte` et `revision.Chronique` ; le cadre " +
			"herite (`revision.CadreHeriteGrammaire`) disparait avec lui, dans le commit qui " +
			"bascule la couche sur le chemin relatif a la racine",
	},
	{
		fichier: "internal/sync/killcollector/decoder_rev_fingerprint_test.go",
		pose:    "2026-09-17", lot: "2.6.1",
		reprise: "`facts.Rev` descend en `facts/` (note de preparation §3.1) et son gate passe " +
			"par `revision` ; le contrat courant rend deja l empreinte figee, aucun backlog " +
			"n est ouvert par la reprise",
	},
}

// TestAucuneEmpreinteDeSourcesAdHoc : personne ne recalcule une empreinte de sources hors de
// `film/revision`.
func TestAucuneEmpreinteDeSourcesAdHoc(t *testing.T) {
	porteurs, fichiers := balayerPorteursDEmpreinte(t)
	if fichiers < plancherFichiersEmpreinte {
		t.Fatalf("balayage muet : %d fichiers .go vus, plancher %d — l arborescence a bouge et "+
			"ce ratchet ne garde plus rien", fichiers, plancherFichiersEmpreinte)
	}
	tolerees := map[string]bool{}
	for _, e := range empreintesAdHocTolerees {
		tolerees[e.fichier] = true
	}
	var violations []string
	for _, rel := range porteurs {
		if !tolerees[rel] {
			violations = append(violations, rel)
		}
	}
	if len(violations) == 0 {
		return
	}
	t.Errorf("empreinte de sources recalculee hors de `%s` :\n  %s\n"+
		"Le mecanisme est CENTRAL depuis le lot 2.6.0 (CLAUDE.md regle 6) : `revision.Empreinte` "+
		"pour le calcul, `revision.Chronique` pour la chronique, `revision.Porte` pour la "+
		"regeneration, `revision.Messages` pour le message. Une copie de plus, c est un "+
		"cinquieme cadre de hachage — et deux cadres differents ne se comparent pas, donc une "+
		"renumerotation de revision pour rien. Ajouter une entree a `empreintesAdHocTolerees` "+
		"N EST PAS une reponse : cette table est datee et ne recense que les deux mecanismes "+
		"que le lot 2.6.1 doit faire heriter.",
		paquetCanoniqueEmpreinte, strings.Join(violations, "\n  "))
}

// TestAllowlistDesEmpreintesAdHocNEstPasPerimee : une entree qui ne decrit plus un porteur se
// RETIRE, dans le commit meme qui la resout.
func TestAllowlistDesEmpreintesAdHocNEstPasPerimee(t *testing.T) {
	porteurs, _ := balayerPorteursDEmpreinte(t)
	vivants := map[string]bool{}
	for _, rel := range porteurs {
		vivants[rel] = true
	}
	for _, e := range empreintesAdHocTolerees {
		if vivants[e.fichier] {
			continue
		}
		t.Errorf("`empreintesAdHocTolerees` cite %s (pose %s, lot %s), qui ne recalcule plus "+
			"d empreinte de sources : entree perimee, la retirer.", e.fichier, e.pose, e.lot)
	}
}

// TestDetecteurDEmpreinteMord : LA PREUVE QUE LE GARDE-RAIL MORD. Un ratchet qui ne detecte
// jamais rien est inutile (meme doctrine que `TestEmpreinteSourcesGoMord`).
func TestDetecteurDEmpreinteMord(t *testing.T) {
	for _, cas := range []struct {
		quoi   string
		source string
		porte  bool
	}{
		{quoi: "une copie du motif", porte: true, source: `package p
import ("crypto/sha256"; "path/filepath")
func f(racine string) { _ = sha256.New(); _ = filepath.WalkDir; _ = "_test.go" }`},
		{quoi: "un hachage de medias (pas de source Go filtree)", porte: false, source: `package p
import ("crypto/sha256"; "path/filepath")
func f() { _ = sha256.Sum256(nil); _ = filepath.WalkDir }`},
		{quoi: "un balayage de sources sans hachage", porte: false, source: `package p
import "path/filepath"
func f() { _ = filepath.WalkDir; _ = "_test.go" }`},
		{quoi: "un commentaire qui decrit le motif", porte: false, source: `package p
// Ce paquet n implemente rien : il decrit sha256, filepath.Walk et "_test.go".
func f() {}`},
	} {
		if got := porteUneEmpreinteDeSources(cas.source); got != cas.porte {
			t.Errorf("%s : detecte=%v, attendu %v", cas.quoi, got, cas.porte)
		}
	}
}

// porteUneEmpreinteDeSources dit si un fichier reunit les trois marqueurs, commentaires retires.
func porteUneEmpreinteDeSources(source string) bool {
	corps := stripComments(source)
	for _, m := range marqueursEmpreinteDeSources {
		if !strings.Contains(corps, m) {
			return false
		}
	}
	return true
}

// balayerPorteursDEmpreinte rend les fichiers porteurs (chemin relatif a `apps/go-api`, en
// slash, tries) et le nombre total de fichiers `.go` parcourus.
//
// Les `_test.go` sont PARCOURUS : les deux mecanismes d aujourd hui sont des tests, et c est la
// que la cinquieme copie naitrait.
func balayerPorteursDEmpreinte(t *testing.T) ([]string, int) {
	t.Helper()
	racine := apiRootDepuisIci(t)
	_, ceFichier, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	// CE fichier DECRIT le motif dans ses cas de test : il s exclut lui-meme. La comparaison se
	// fait sur le chemin RELATIF en slash — `runtime.Caller` rend un chemin a separateurs `/`
	// meme sous Windows, la ou `filepath.WalkDir` rend des `\`, et comparer les deux bruts ne
	// s exclut de rien (mesure : le ratchet se denoncait lui-meme).
	ceFichierRel, err := filepath.Rel(racine, ceFichier)
	if err != nil {
		t.Fatalf("chemin de ce fichier : %v", err)
	}
	ceFichierRel = filepath.ToSlash(ceFichierRel)
	var porteurs []string
	total := 0
	for _, sous := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(racine, sous), func(chemin string, d fs.DirEntry, errMarche error) error {
			if errMarche != nil {
				return errMarche
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			rel, errRel := filepath.Rel(racine, chemin)
			if errRel != nil {
				return errRel
			}
			rel = filepath.ToSlash(rel)
			total++
			if rel == ceFichierRel {
				return nil
			}
			if strings.HasPrefix(rel, paquetCanoniqueEmpreinte+"/") {
				return nil
			}
			blob, errLire := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis la racine du module
			if errLire != nil {
				return errLire
			}
			if porteUneEmpreinteDeSources(string(blob)) {
				porteurs = append(porteurs, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", sous, err)
		}
	}
	sort.Strings(porteurs)
	return porteurs, total
}
