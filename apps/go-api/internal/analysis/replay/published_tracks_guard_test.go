package replay

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// GARDE-RAIL du helper `keepOfPublishedTracks` (lot 3.1) et de son jumeau par JOUEUR
// `publishedXUIDs` (lot des vies anonymes, 2026-09-06).
//
// La règle n°6 du dépôt : une factorisation SANS garde-rail re-diverge. La leçon est
// chiffrée (prédicat bot passé de 8 à 36 copies après centralisation). Ici la divergence
// serait INVISIBLE — un calque garderait des éléments qu'un autre écarte, sur la même
// fiche, sans qu'aucun compteur ne bouge.
//
// Ce test échoue si un second endroit du paquet reconstruit, À PARTIR DES PISTES, un
// ensemble de slots publiés ou un ensemble de joueurs publiés.

// slotSetPattern reconnaît la construction d'un ensemble de slots : `<map>[<x>.Slot] = true`.
var slotSetPattern = regexp.MustCompile(`\w+\[\w+\.Slot\]\s*=\s*true`)

// xuidSetPattern reconnaît la construction d'un ensemble de JOUEURS publiés À PARTIR DES
// PISTES : un `range tracks` suivi, dans les six lignes, de `<map>[<x>.XUID] = true`.
//
// POURQUOI CE SECOND MOTIF EXISTE (2026-09-06). Le garde-rail ne filtrait que le motif keyé
// par `.Slot`, qu'une map keyée par chaîne ne déclenche jamais. Les deux seuls filtres du
// paquet keyés par XUID — `objectives.go` et `neutral_deaths.go` — ont donc divergé des onze
// autres SANS QUE LE GARDE LES VOIE : ils cadençaient sur le seul nom LU, supprimant toutes
// les données d'un joueur dont aucune vie n'est nommée alors que le pont nomme son slot.
// C'est exactement la divergence INVISIBLE annoncée ci-dessus.
//
// POURQUOI L'ANCRE `range tracks` : sans elle le motif attrape `rosterFromDeaths`
// (player_index.go), qui indexe les xuids du FIL DES MORTS — un autre espace de clés
// (`uint64`), une autre question, et aucun rapport avec les pistes publiées. Un garde-rail
// qui crie sur du code juste finit désactivé.
var xuidSetPattern = regexp.MustCompile(`(?s)range tracks\b(?:[^\n]*\n){0,6}?[^\n]*\w+\[\w+\.XUID\]\s*=\s*true`)

// publishedSlotsOwner est le SEUL fichier de production autorisé à construire ces ensembles.
const publishedSlotsOwner = "published_tracks.go"

func TestUnSeulConstructeurDEnsembleDeSlotsPublies(t *testing.T) {
	// LE GARDE-RAIL DOIT POUVOIR ÉCHOUER (leçon J4.0 — un garde qui ne peut pas échouer ne
	// garde rien). Il se prouve ici sur un EXTRAIT SYNTHÉTIQUE portant les motifs interdits,
	// et non en exigeant que le propriétaire les porte : `publishedXUIDs` a le droit d'écrire
	// la règle autrement (il la factorise dans `xuidOfPublishedTrack`), et un garde adossé à
	// la forme d'écriture du propriétaire casserait à la première refactorisation légitime.
	temoin := []byte("func fauxAmi(tracks []Track) {\n\tpub := map[string]bool{}\n" +
		"\tslots := map[uint32]bool{}\n\tfor _, tr := range tracks {\n" +
		"\t\tpub[tr.XUID] = true\n\t\tslots[tr.Slot] = true\n\t}\n}\n")
	for _, cas := range []struct {
		nom    string
		motif  *regexp.Regexp
		remede string
	}{
		{"slots", slotSetPattern, "appeler keepOfPublishedTracks(...) au lieu de recopier la boucle"},
		{"joueurs", xuidSetPattern, "appeler publishedXUIDs(tracks, slotXUID) : un index bâti sur " +
			"le seul nom LU jette les vies dont le nommage a échoué et que le pont canonique nomme"},
	} {
		if !cas.motif.Match(temoin) {
			t.Fatalf("motif %s : le témoin synthétique ne le déclenche plus — "+
				"le garde-rail ne vérifie plus rien", cas.nom)
		}
		if offenders := fichiersPortantLeMotif(t, cas.motif); len(offenders) > 0 {
			t.Errorf("ensemble de %s publiés reconstruit hors de %s : %v — %s",
				cas.nom, publishedSlotsOwner, offenders, cas.remede)
		}
	}
}

// fichiersPortantLeMotif rend les fichiers de production du paquet où le motif apparaît hors
// de son propriétaire.
func fichiersPortantLeMotif(t *testing.T, motif *regexp.Regexp) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet: %v", err)
	}
	var offenders []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("lecture %s: %v", name, err)
		}
		if !motif.Match(raw) || name == publishedSlotsOwner {
			continue
		}
		offenders = append(offenders, name)
	}
	return offenders
}

// TestKeepOfPublishedTracks_ContratPreserve fige le contrat commun aux quatre calques :
// entrée vide -> nil, sortie vide -> nil (jamais un tableau vide), filtrage effectif.
func TestKeepOfPublishedTracks_ContratPreserve(t *testing.T) {
	tracks := []Track{{Slot: 512}}
	garde := func(s Shot, published map[uint32]bool) bool { return published[s.Slot] }

	if got := keepOfPublishedTracks(nil, tracks, garde); got != nil {
		t.Errorf("entrée vide : attendu nil, obtenu %v", got)
	}
	if got := keepOfPublishedTracks([]Shot{{Slot: 999}}, tracks, garde); got != nil {
		t.Errorf("tout filtré : attendu nil (et non un tableau vide), obtenu %v", got)
	}
	got := keepOfPublishedTracks([]Shot{{Slot: 512}, {Slot: 999}}, tracks, garde)
	if len(got) != 1 || got[0].Slot != 512 {
		t.Errorf("filtrage : attendu le seul slot publié, obtenu %v", got)
	}
}

// TestUnePisteCONTESTEEnEstJamaisIndexeeSousUnXUID — LE CONSTAT C2 de la revue VIES-R1
// (2026-09-07), reproduit sur la configuration REELLE du slot 734 de `084a804d`.
//
// LE DEFAUT. `xuidOfPublishedTrack` repliait sur `slotXUID[t.Slot]` SANS la garde d'ambiguite
// que le lot venait pourtant d'ajouter a ses deux jumeaux (`bridgeOfSlot`, `xuidAt`) pour cette
// raison precise. Sur un slot en collision, la passe de nommage REFUSE (`contested = 1`) et le
// helper partage servait quand meme le PREMIER occupant : `samplesByXUID` (zones) indexait les
// positions de la piste contestee sous ce joueur, si bien qu'une capture de zone pouvait etre
// geolocalisee sur la trajectoire d'un AUTRE — attribuee a la mauvaise zone, ou attribuee la ou
// elle aurait du sortir `NoPosition`. Meme exposition pour `tracksByXUID` (drapeau).
//
// LE CORRECTIF EST A LA SOURCE : les quatre lecteurs qui NOMMENT une piste recoivent
// `own.NamingBridge()`, d'ou les slots ambigus sont RETIRES. Le lecteur ne peut plus oublier la
// garde, puisqu'il n'a plus de quoi l'enfreindre.
//
// MUTATION : passer `own.SlotXUID` au lieu de `own.NamingBridge()` rougit — la piste contestee
// reprend le nom du premier occupant.
func TestUnePisteCONTESTEEnEstJamaisIndexeeSousUnXUID(t *testing.T) {
	// La configuration mesuree : A [5872..6981], la vie contestee [7123..7158], B [7457..7591].
	const a, b = uint64(2535430265968559), uint64(2535456423427614)
	own := OwnerReport{
		SlotXUID:      map[uint32]uint64{734: a}, // le pont garde le PREMIER occupant
		SlotAmbiguous: map[uint32]bool{734: true},
	}
	contestee := Track{Slot: 734, StartFrame: 7123, EndFrame: 7158,
		Points: []Point{{T: 7140, X: 1, Y: 0, Z: 0}}}

	if got := xuidOfPublishedTrack(contestee, own.NamingBridge()); got != "" {
		t.Errorf("piste contestee indexee sous %q — la passe de nommage l'a REFUSEE, "+
			"le helper ne doit pas la nommer non plus", got)
	}
	// Le chemin NEUF du lot : les zones. La piste contestee ne doit servir a personne.
	if s := samplesByXUID([]Track{contestee}, own.NamingBridge()); len(s) != 0 {
		t.Errorf("echantillons indexes %v — une capture de %d pourrait etre geolocalisee sur "+
			"la trajectoire d'un autre joueur", s, a)
	}
	// CONTRE-EPREUVE : sur un slot NON ambigu, le repli par le pont joue toujours.
	sain := OwnerReport{SlotXUID: map[uint32]uint64{900: b}}
	if got := xuidOfPublishedTrack(Track{Slot: 900}, sain.NamingBridge()); got != "2535456423427614" {
		t.Errorf("slot non ambigu : %q, attendu le repli par le pont", got)
	}
}
