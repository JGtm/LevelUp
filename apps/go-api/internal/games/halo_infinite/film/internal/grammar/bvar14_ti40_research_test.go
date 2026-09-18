//go:build research

package grammar

// bvar14_ti40_research_test.go — LOT 5.1.7-b : LA PART DE RECORDS `ti=40` A `bVar14 == 1`.
//
// # LA CONDITION QUE `default_state_ti40.go` POSE
//
// Le deserialiseur de l etat par defaut de `ti=40` (`consumeDefaultStateTI40`, FUN_1410A5A74) a
// cinq feuilles. QUATRE sont etablies statiquement ; la CINQUIEME — le quaternion de la feuille 4,
// derriere la porte de flux `bVar14` — depend de globaux de configuration runtime, et le port la
// modelise ABSENTE (`vehicleMediaFrameBits = 0`). Le fichier ecrit la regle : « La part de records
// a `bVar14 == 1` se MESURE (oracle de position), elle ne se suppose pas. »
//
// C est cette mesure. Elle lit, pour chaque record `ti=40` d image-cle, l en-tete par entite, le
// premier mot de taille `n1`, puis les deux premieres feuilles (`V` et le bloc MPP) — et rend le
// BIT DE PORTE qui suit. Rien d autre n est consomme : la mesure s arrete au bit qu elle veut.
//
// LECTURE SEULE, UN FILM PAR INVOCATION, aucun paquet delta :
//
//	BV14_FILM=<abs>/film_chunks/4f77afc1 BV14_CARTE="Flood Gulch" \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestBVar14Ti40$' -v -timeout 60m

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestBVar14Ti40(t *testing.T) {
	dir, carte := os.Getenv("BV14_FILM"), os.Getenv("BV14_CARTE")
	if dir == "" || carte == "" {
		t.Skip("instrument de mesure : BV14_FILM et BV14_CARTE requis")
	}
	fc := bv14Contexte(t, dir, carte)
	if restore, err := InstallFilmFormatMPP(fc); err == nil {
		defer restore()
	}
	ctx := fc.ContexteDeLecture()
	var total, porteUn, n1Nul int
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			bv14UnPaquet(pk.Payload(data), ctx, &total, &porteUn, &n1Nul)
		}
	}
	bv14Ventile(t, fc)
	pc := 0.0
	if total > 0 {
		pc = 100 * float64(porteUn) / float64(total)
	}
	t.Logf("%s : records ti=40 d image-cle = %d | n1 <= 0 (aucun etat par defaut) = %d | "+
		"bVar14 == 1 = %d (%.2f %%)", filepath.Base(dir), total, n1Nul, porteUn, pc)
}

// bv14Contexte ouvre le contexte sous l entree de catalogue de la carte, comme la cuisson.
func bv14Contexte(t *testing.T, dir, carte string) *FilmContext {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q : %v", carte, err)
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(bal)
	return fc
}

// bv14UnPaquet lit la porte de chaque record `ti=40` d UN paquet d image-cle.
func bv14UnPaquet(pay []byte, ctx ContexteDeLecture, total, porteUn, n1Nul *int) {
	for _, b := range keyframeBornesToutes(pay) {
		if b.TI != VehicleTypeIndex {
			continue
		}
		*total++
		br := LecteurSur(pay)
		br.PoserContexte(ctx)
		br.SetBitPos(b.Bit + br.cadre().EnTeteBits)
		mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil, bornee a 32
		if int32(br.ReadBits(mot)) <= 0 {       //nolint:gosec // n1, comparaison SIGNEE
			*n1Nul++
			continue
		}
		consumeVersionPrefix(br)              // feuille 1
		consumeMultiplayerPropertiesBlock(br) // feuille 2
		if br.ReadBit() {                     // la porte bVar14 -> feuille 4
			*porteUn++
		}
	}
}

// bv14Ventile rend l histogramme de `DesyncAt` des records `ti=40` d image-cle, VENTILE par la
// valeur de la porte `bVar14`. C est l oracle qui dit si la feuille 4 est lue juste : si elle
// l est, les deux populations butent au MEME rang (le premier composant non porte) ; si elle est
// mal dimensionnee, la population `bVar14 == 1` part de travers et bute n importe ou.
func bv14Ventile(t *testing.T, fc *FilmContext) {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	ctx := fc.ContexteDeLecture()
	histo := map[bool]map[int]int{false: {}, true: {}}
	ferme := map[bool]int{}
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			bv14VentileUnPaquet(pk.Payload(data), reg, ctx, histo, ferme)
		}
	}
	for _, porte := range []bool{false, true} {
		t.Logf("bVar14 == %v : fermes=%d | desync par index — %s", porte, ferme[porte],
			ti40dHisto(histo[porte], map[int]string{}))
	}
}

// bv14VentileUnPaquet deroule les records `ti=40` d UN paquet.
func bv14VentileUnPaquet(pay []byte, reg *Registry, ctx ContexteDeLecture,
	histo map[bool]map[int]int, ferme map[bool]int) {
	for _, b := range keyframeBornesToutes(pay) {
		if b.TI != VehicleTypeIndex || b.Want < 0 {
			continue
		}
		br := LecteurSur(pay)
		br.PoserContexte(ctx)
		br.SetBitPos(b.Bit + br.cadre().EnTeteBits)
		mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil, bornee a 32
		if int32(br.ReadBits(mot)) <= 0 {       //nolint:gosec // n1, comparaison SIGNEE
			continue
		}
		consumeVersionPrefix(br)
		consumeMultiplayerPropertiesBlock(br)
		porte := br.ReadBit()
		tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
		histo[porte][tr.DesyncAt]++
		if tr.EndBit == b.Want {
			ferme[porte]++
		}
	}
}
