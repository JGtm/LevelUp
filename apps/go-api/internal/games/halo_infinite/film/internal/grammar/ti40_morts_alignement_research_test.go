package grammar

// ti40_morts_alignement_research_test.go — QUELLE VALEUR DU PROFIL DEPLACE LES MORTS DE VEHICULE.
//
// # CE QU IL SERT A TRANCHER (D7 (3.4.2))
//
// Le corpus gate du lot 3.4.2, contre `492cb0923`, fait baisser `coverage.vehicles.deaths*` sur
// cinq temoins — `60ae07c4` 8 -> 0, `a521164d` 12 -> 6, `4f77afc1` 14 -> 11, `11de8353` 6 -> 5,
// `50247b26` 4 -> 3 — et DEUX LIGNES PUBLIEES disparaissent (`vehicles.tEnd/presents`,
// `vehicles/par-end/destroyed`). Le meme gate contre la fusion `151d0c6f9` sort 8 temoins sur 8
// a zero : la cause est dans le lot 3.4.1, pas dans le correctif du mot de poignee.
//
// Les morts de vehicule se lisent par `ScanObjectDeaths` : une MARCHE qui deroule TOUS les
// records d un paquet delta, tous archetypes confondus. Un record de bipede mal consomme
// desynchronise la suite du paquet — donc les largeurs du chemin absolu d i0, que le lot 3.4.1
// a changees, atteignent les morts de `ti=40` sans les toucher directement.
//
// TROIS GRANDEURS PEUVENT LE FAIRE, ET CET INSTRUMENT LES SEPARE :
//
//	AxisW de la carte    le triplet LU contre l uniforme `14/14/14` d avant le lot
//	Traversal.IndexW     le mot de poignee (1, 2, 3)
//	la branche idx == -1 `absAxisWFor` prend la table DEFAUT du build (22/22/22) quand la porte
//	                     est posee, la ou le lot precedent prenait l uniforme. L HISTOGRAMME DES
//	                     INDEX dit si cette branche est seulement empruntee sur ce film.
//
// LECTURE SEULE, N ASSERTE RIEN, garde par `TI40_FILM` :
//
//	TI40_FILM=<repo>/data/cache/film_chunks/60ae07c4 \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestTi40MortsAlignement$' -v

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestTi40MortsAlignement(t *testing.T) {
	dir := os.Getenv("TI40_FILM")
	if dir == "" {
		t.Skipf("instrument de mesure : TI40_FILM requis")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	nom := filepath.Base(dir)
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup(os.Getenv("TI40_CARTE"))
	if err != nil {
		t.Fatalf("carte %q : %v", os.Getenv("TI40_CARTE"), err)
	}
	lay := entree.Layout()
	detecte, _, err := DetectI0LayoutOf(film)
	if err != nil {
		t.Fatalf("decoupage i0 de %s : %v", nom, err)
	}
	t.Logf("%s : CATALOGUE %s (production) · auto-detecte %s", nom, lay, detecte)

	mesure := func(etiquette string, regler func(p *ProfilDeBalayage)) {
		fc := contexteDeBobine(film)
		p := fc.ProfilDeBalayage()
		p.PoserLargeursObjetDuMondeDepuisDecoupage(lay)
		if regler != nil {
			regler(&p)
		}
		fc.PoserProfilDeBalayage(p)
		morts, st, err := ScanObjectDeaths(fc)
		if err != nil {
			t.Fatalf("%s : %v", etiquette, err)
		}
		parType := map[uint32]int{}
		queues := 0
		for _, m := range morts {
			parType[m.TypeIndex]++
			if m.TailDesync {
				queues++
			}
		}
		abs := fc.Observation().prendreIndexAbsolus()
		wo := p.LargeursObjetDuMonde()
		t.Logf("%-28s axes=%v poigneeIW=%d | morts=%d ti40=%d queues=%d | paquets_a_events=%d "+
			"localises=%d idLow=%d cadre_par_defaut=%v | index_absolus %s",
			etiquette, wo.AxisW, p.Mouvement.Traversal.IndexW, len(morts), parType[40], queues,
			st.EventPackets, st.LocatedPackets, st.Config.IDLowBits, st.CadreParDefaut,
			histoIndex(abs))
	}

	mesure("APRES (carte, iw=1)", nil)
	mesure("AVANT (uniforme 14/14/14)", func(p *ProfilDeBalayage) {
		p.PoserLargeursObjetDuMonde(profile.PrecisionDescriptor{
			IndexW: p.LargeursObjetDuMonde().IndexW,
			AxisW:  [3]uint{14, 14, 14},
			Region: p.LargeursObjetDuMonde().Region,
		})
	})
	for r := uint32(0); r <= 5; r++ {
		v := r
		mesure("carte, param_4="+strconv.Itoa(int(v)), func(p *ProfilDeBalayage) {
			p.PoserParamEtat(v)
		})
	}
	for iw := uint(2); iw <= 3; iw++ {
		w := iw
		mesure("carte, iw="+strconv.Itoa(int(w)), func(p *ProfilDeBalayage) {
			p.Mouvement.Traversal.IndexW = w
		})
	}
}

// histoIndex rend l histogramme des index de plage absolus, trie, avec `-1` en tete : c est la
// PORTE POSEE (pas d index), la branche qui lit la table DEFAUT du build.
func histoIndex(h map[int]int) string {
	if len(h) == 0 {
		return "(aucune lecture absolue)"
	}
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strconv.Itoa(k) + ":" + strconv.Itoa(h[k]))
	}
	return b.String()
}
