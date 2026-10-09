//go:build research && campagne_overlay

package grammar

// r_veh_index_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L6 (2026-10-02) : les index
// de plage IMPOSSIBLES (index >= nombre de plages de la carte, `FUN_140be9a14` / `FUN_140770640`)
// lus dans la lecture FINALE des records d un paquet. Le releve de BIS_2 (`bis2NoterIndex`)
// compte toute lecture d index, essais de l inference compris ; ici on relit, composant par
// composant, les seuls records que la marche a retenus (StartBit de chaque composant), sous le
// contexte de production. Exige la surcouche (releve `bis2C3` de `lecteur_position.go`).
//
// Environnement : celui de `TestCampagneBis2PositionsProduction` (CAMPAGNE_CATALOGUE,
// CAMPAGNE_CARTES, CAMPAGNE_BORNES_<module> : le nombre de plages = 1 + bornes fournies).
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestRVehIndexImpossibles$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rvxEcouteur classe les paquets selon les index lus par leurs records retenus.
type rvxEcouteur struct {
	f      *cmFilm
	juge   *cmJuge
	plages int
	// paquets : fermes / fermes contredits / non fermes, avec et sans index impossible final.
	fermes, fermesImp, fermesImpContredits, nonFermesImp int
	parIndex                                             map[int]int
	parComposant                                         map[string]int
}

func (e *rvxEcouteur) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *rvxEcouteur) finDeFilm()                                     {}

func (e *rvxEcouteur) paquet(_ int, p *cmPaquet, _ *World) {
	impossible := false
	for _, r := range p.recs {
		arch, ok := e.f.reg.Archetype(int(r.TypeIndex))
		if !ok {
			continue
		}
		for _, cr := range r.Trace.Comps {
			if !cr.Ported {
				continue
			}
			bis2C3 = &bis2EtatC3{}
			br := LecteurSur(p.pay)
			br.poserCadre(e.f.cfg)
			br.SetBitPos(cr.StartBit)
			consumeByNameCapturing(br, cr.Name, r.TypeIndex, arch.Level(cr.Index))
			for _, ev := range bis2C3.evenements {
				e.parIndex[ev.idx]++
				if ev.idx >= e.plages {
					impossible = true
					e.parComposant[fmt.Sprintf("ti=%d %s", r.TypeIndex, strings.TrimSuffix(cr.Name, "-component"))]++
				}
			}
			bis2C3 = nil
		}
	}
	cle := [2]int{p.d.Chunk, p.d.Index}
	switch {
	case p.d.Fermee:
		e.fermes++
		if impossible {
			e.fermesImp++
			if e.juge.contreRef[cle] {
				e.fermesImpContredits++
			}
		}
	case impossible:
		e.nonFermesImp++
	}
}

// TestRVehIndexImpossibles : paquets dont la lecture finale porte un index de plage impossible.
func TestRVehIndexImpossibles(t *testing.T) {
	racine, sortie, films := b2Env(t)
	cat, err := profile.LoadMapQuantCatalog(os.Getenv("CAMPAGNE_CATALOGUE"))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	cartes := map[string]string{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_CARTES"), ";") {
		if id, carte, ok := strings.Cut(x, "="); ok {
			cartes[strings.TrimSpace(id)] = strings.TrimSpace(carte)
		}
	}
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tplages\tpaquets_fermes\tfermes_index_impossible_final\t" +
		"dont_contredits_par_le_juge\tnon_fermes_index_impossible_final\tlectures_finales_par_index\t" +
		"composants_index_impossible"}
	defer func() { bis2C3 = nil }()
	for _, id := range films {
		entry, err := cat.Lookup(cartes[id])
		if err != nil {
			t.Logf("%s : carte %q hors catalogue — film ignore", id, cartes[id])
			continue
		}
		garde := filmproc.Arm("campagne/r-veh-index", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		f, ok := b2pOuvrirProduction(t, racine, id, entry, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		blocs := cmLireBlocs(f)
		juge := cmNouveauJuge(f, blocs, nil)
		e := &rvxEcouteur{f: f, juge: juge, plages: 1 + len(b2pBornes(entry.Module)),
			parIndex: map[int]int{}, parComposant: map[string]int{}}
		cmMarcher(f, cmVariante{}, cmMux{juge, e})
		var idx, comps []string
		for k := -1; k < 4; k++ {
			if n := e.parIndex[k]; n > 0 {
				idx = append(idx, fmt.Sprintf("%d:%d", k, n))
			}
		}
		for c, n := range e.parComposant {
			comps = append(comps, fmt.Sprintf("%s=%d", c, n))
		}
		sort.Strings(comps)
		lignes = append(lignes, fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s", id, f.build, e.plages, e.fermes, e.fermesImp,
			e.fermesImpContredits, e.nonFermesImp, strings.Join(idx, ","), strings.Join(comps, ",")))
		t.Logf("%s %s : plages %d ; fermes %d dont %d avec index impossible final (%d contredits) ; pic %d Mio", id,
			f.build, e.plages, e.fermes, e.fermesImp, e.fermesImpContredits, garde.Peak()>>20)
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "r_veh_index_impossibles_final.tsv", lignes)
}
