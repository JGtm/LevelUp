//go:build research

package grammar

// mouvement_5_3_3c_capacites_research_test.go — `i59` ET `i57` : CE QUI DESYNCHRONISE, COMPTE
// (lot 5.3.3-c).
//
// # POURQUOI CET INSTRUMENT AVANT TOUT PORT
//
// Les deux composants sont declares `partiel` dans `ecs_table.tsv`, et chacun porte une desync
// PROPRE — une branche que le depot refuse de deviner. Avant d en porter une seule, il faut
// savoir LAQUELLE coute des records, et combien :
//
//	i59  le corps `Tag==3` rend `ported=false` sur `Zero3 != 0` ou `Inner` hors {1,2}.
//	     L ECRIVAIN (`FUN_142f25e90`) dispatche sur SEPT valeurs internes (0 a 6) : le port en
//	     modelise DEUX. Combien de lectures tombent sur les cinq autres ?
//	i57  la branche `tag==3` rend `ported=false` des que son premier bit vaut 1
//	     (`FUN_142f262d4` : le chemin suivant est garde par `p[2] & 1` et `p[2] & 0x10`, deux
//	     octets d ETAT RUNTIME). Combien de lectures ont ce premier bit a 1 ?
//
// Les deux hooks de l observateur publient DEJA tout ce qu il faut, sans changer un bit.
//
// Rejouable :
//
//	MOUV533C_FILM=<dir> MOUV533C_CARTE=snowbound MOUV533C_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement533CCapacites$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m533cStat : ce que les deux hooks et la ventilation des desyncs rapportent.
type m533cStat struct {
	// i59 : par tag externe, puis par valeur interne et par Zero3 pour le corps tag==3.
	i59Tags              map[uint32]int
	i59Inner             map[int]int
	i59Zero3             map[uint32]int
	i59Corps, i59CorpsOK int
	// i57 : par tag, et l issue de la branche 3.
	i57Tags map[uint64]int
	// fautifs : composant fautif des records ti=35 desynchronises.
	fautifs          map[int]int
	ti35, ti35Desync int
}

func TestMouvement533CCapacites(t *testing.T) {
	dir := os.Getenv("MOUV533C_FILM")
	if dir == "" {
		t.Skip("MOUV533C_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m533cContexte(t, film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	st := m533cStat{i59Tags: map[uint32]int{}, i59Inner: map[int]int{},
		i59Zero3: map[uint32]int{}, i57Tags: map[uint64]int{}, fautifs: map[int]int{}}
	cfg := fc.CadreDeBalayage()
	cfg.Obs = m533cObservateur(&st)
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, m533cVues(), debut)
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				st.ti35++
				if r.DesyncAt >= 0 {
					st.ti35Desync++
					st.fautifs[r.DesyncAt]++
				}
			}
		}
	}
	m533cRendre(t, st)
}

// m533cVues : le nombre de vues, TROIS — ce que le frame-processor deroule (cf. 5.3.3-b).
// `MOUV533C_VUES` le surcharge.
func m533cVues() int {
	if v := os.Getenv("MOUV533C_VUES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

// m533cContexte ouvre le contexte du film sous la carte.
func m533cContexte(t *testing.T, film *source.Film) *FilmContext {
	t.Helper()
	nom := os.Getenv("MOUV533C_CARTE")
	if nom == "" {
		return NewFilmContext(film)
	}
	chemin := os.Getenv("MOUV533C_BORNES")
	if chemin == "" {
		t.Fatalf("MOUV533C_BORNES attendu avec MOUV533C_CARTE")
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q : %v", nom, err)
	}
	t.Logf("CARTE : %s (axes %v)", nom, entry.AxisWidths)
	return NewFilmContextForMap(film, &entry, nil)
}

// m533cObservateur branche les deux hooks. IL NE CHANGE AUCUN BIT : les deux publient ce que le
// deser de production a deja lu (cf. l en-tete d `observateur.go`).
func m533cObservateur(st *m533cStat) *Observation {
	return &Observation{
		AbilityNonPredictedHook: func(s AbilityNonPredictedState) {
			st.i59Tags[s.Tag]++
			if !s.BodyWalked {
				return
			}
			st.i59Corps++
			if s.BodyOK {
				st.i59CorpsOK++
			}
			st.i59Inner[s.Inner]++
			st.i59Zero3[s.Zero3]++
		},
		SpartanAbilityHook: func(tag, _, _ uint64, _ bool) { st.i57Tags[tag]++ },
	}
}

// m533cRendre publie les trois tableaux.
func m533cRendre(t *testing.T, st m533cStat) {
	t.Helper()
	t.Logf("RECORDS ti=35 : %d, dont %d desynchronises (%.2f %%)", st.ti35, st.ti35Desync,
		m533bPart(st.ti35Desync, st.ti35))
	t.Logf("  COMPOSANT FAUTIF : %s", m533cTable(st.fautifs))
	t.Logf("i59 : tags externes %s", m533cTableU32(st.i59Tags))
	t.Logf("  corps tag==3 parcourus %d, dont %d complets (%.1f %%)", st.i59Corps, st.i59CorpsOK,
		m533bPart(st.i59CorpsOK, st.i59Corps))
	t.Logf("  VALEUR INTERNE (l ecrivain en dispatche SEPT, 0 a 6 ; le port en porte DEUX) : %s",
		m533cTable(st.i59Inner))
	t.Logf("  Zero3 (desync quand != 0) : %s", m533cTableU32(st.i59Zero3))
	t.Logf("i57 : tags %s (le tag 3 est la seule branche a desync)", m533cTableU64(st.i57Tags))
}

// m533cTable rend une table clef->compte, triee par compte decroissant.
func m533cTable(m map[int]int) string {
	type p struct{ k, n int }
	ps := make([]p, 0, len(m))
	for k, n := range m {
		ps = append(ps, p{k, n})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].n > ps[j].n })
	var out []string
	for _, x := range ps {
		out = append(out, fmt.Sprintf("%d:%d", x.k, x.n))
	}
	if len(out) == 0 {
		return "(aucune lecture)"
	}
	return strings.Join(out, " ")
}

// m533cTableU32 / m533cTableU64 : la meme table sur des clefs non signees.
func m533cTableU32(m map[uint32]int) string {
	c := make(map[int]int, len(m))
	for k, n := range m {
		c[int(k)] = n
	}
	return m533cTable(c)
}

func m533cTableU64(m map[uint64]int) string {
	c := make(map[int]int, len(m))
	for k, n := range m {
		c[int(k)] = n //nolint:gosec // clefs de tag, 2 bits
	}
	return m533cTable(c)
}
