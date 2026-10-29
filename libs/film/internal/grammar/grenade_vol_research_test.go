//go:build research

package grammar

// grenade_vol_research_test.go — INSTRUMENT : les lancers de grenade, la vie de leur projectile et
// les messages de detonation de la vue A, comptes sur des films reels.
//
// LA QUESTION (controle n° 24 des controles v8, 2026-10-21). Au rejeu, des lancers de grenade sans
// explosion. Le rejeu pose l explosion au dernier point de la piste du projectile liee au lancer ;
// un lancer sans piste liee n explose pas. Ce banc compte, par film et par minute : les lancers, les
// lancers qu une naissance de piste ti=41 LUE PAR LA MARCHE suit a moins de 200 ms (toutes bandes
// confondues, puis dans la seule bande des images-cles), et les messages `projectile_detonate`
// (genre 5) et `projectile_impact_effect` (genre 6) de la vue A.
//
// USAGE (lecture seule, aucun film = saut propre) :
//
//	GRENVOL_FILMS=<racine data/cache> GRENVOL_IDS='d6918972=origin;d9781168=dredge' \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar -run TestGrenadeVol -v -timeout 1800s

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

// grenVolNaissance est le premier record d une vie de projectile lue par la marche.
type grenVolNaissance struct {
	ts         uint64
	slot, gen  uint32
	dansBande  bool
	dernierTS  uint64
	nbEchantil int
}

func TestGrenadeVol(t *testing.T) {
	racine, liste := os.Getenv("GRENVOL_FILMS"), os.Getenv("GRENVOL_IDS")
	if racine == "" || liste == "" {
		t.Skip("GRENVOL_FILMS ou GRENVOL_IDS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		origine, err := ScanClockOrigin(film)
		if err != nil {
			t.Fatalf("%s : origine : %v", id, err)
		}
		grenVolUnFilm(t, id, aj14fContexte(film, &entree), origine, entree.Range())
	}
}

func grenVolUnFilm(t *testing.T, id string, fc *FilmContext, origine uint64, wr profile.Vec3Range) {
	t.Helper()
	lancers, err := ScanGrenadeThrows(fc)
	if err != nil {
		t.Fatalf("%s : lancers : %v", id, err)
	}
	// LA PISTE DE PRODUCTION : celle que la cuisson publie et a laquelle le rejeu lie l explosion.
	pistes, err := ScanProjectiles(fc, &wr)
	if err != nil {
		t.Fatalf("%s : pistes : %v", id, err)
	}
	lies := 0
	for _, g := range lancers {
		for _, p := range pistes {
			if len(p.Pts) > 0 && p.Pts[0].TimestampUS+50_000 >= g.TimestampUS && p.Pts[0].TimestampUS <= g.TimestampUS+200_000 {
				lies++
				break
			}
		}
	}
	t.Logf("=== %s : PRODUCTION ScanProjectiles : %d pistes ; %d lancers sur %d suivis d une naissance de piste a 200 ms",
		id, len(pistes), lies, len(lancers))
	bande := worldObjectSlotBand(fc, ProjectileTypeIndex)
	objets, _, err := fc.objetsDuMonde()
	if err != nil {
		t.Fatalf("%s : marche : %v", id, err)
	}
	naissances := grenVolNaissances(objets, bande, ProjectileTypeIndex)
	horsBande := 0
	for _, n := range naissances {
		if !n.dansBande {
			horsBande++
		}
	}
	t.Logf("=== %s : %d lancers ; bande ti=41 des images-cles : %d slots ; vies ti=41 lues par la marche : %d (hors bande : %d)",
		id, len(lancers), len(bande), len(naissances), horsBande)
	for _, ti := range []int{EquipmentTypeIndex, GroundWeaponTypeIndex} {
		b := worldObjectSlotBand(fc, ti)
		vies := grenVolNaissances(objets, b, ti)
		hors, echHors := 0, 0
		for _, n := range vies {
			if !n.dansBande {
				hors++
				echHors += n.nbEchantil
			}
		}
		t.Logf("    ti=%d : bande %d slots ; vies lues par la marche %d, hors bande %d (%d echantillons)",
			ti, len(b), len(vies), hors, echHors)
	}

	detos, impacts := grenVolMessages(fc)
	t.Logf("    vue A : %d projectile_detonate (genre 5), %d projectile_impact_effect (genre 6)", len(detos), len(impacts))

	type minute struct{ lancers, lieTous, lieBande, deto int }
	parMinute := map[int]*minute{}
	at := func(ts uint64) *minute {
		m := int((ts - origine) / 60_000_000)
		if parMinute[m] == nil {
			parMinute[m] = &minute{}
		}
		return parMinute[m]
	}
	for _, g := range lancers {
		m := at(g.TimestampUS)
		m.lancers++
		tous, bandeSeule := false, false
		for _, n := range naissances {
			if n.ts+50_000 < g.TimestampUS || n.ts > g.TimestampUS+200_000 {
				continue
			}
			tous = true
			if n.dansBande {
				bandeSeule = true
			}
		}
		if tous {
			m.lieTous++
		}
		if bandeSeule {
			m.lieBande++
		}
		for _, d := range detos {
			if d >= g.TimestampUS && d <= g.TimestampUS+5_000_000 {
				m.deto++
				break
			}
		}
	}
	cles := make([]int, 0, len(parMinute))
	for k := range parMinute {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	tot := minute{}
	for _, k := range cles {
		m := parMinute[k]
		t.Logf("    minute %2d : lancers %3d · naissance lue (toutes bandes) %3d · dans la bande %3d · une detonation dans les 5 s %3d",
			k, m.lancers, m.lieTous, m.lieBande, m.deto)
		tot.lancers += m.lancers
		tot.lieTous += m.lieTous
		tot.lieBande += m.lieBande
		tot.deto += m.deto
	}
	t.Logf("    TOTAL : lancers %d · naissance lue %d · dans la bande %d · detonation dans les 5 s %d",
		tot.lancers, tot.lieTous, tot.lieBande, tot.deto)
}

// grenVolNaissances rend les vies ti=41 que la marche a lues : une vie est une suite de records du
// meme (slot, generation) sans trou de plus d une seconde.
func grenVolNaissances(objets *objetsDuMondeLus, bande map[uint32]bool, ti int) []grenVolNaissance {
	type cle struct{ slot, gen uint32 }
	enCours := map[cle]*grenVolNaissance{}
	var out []*grenVolNaissance
	for k, p := range objets.pistes.paquets {
		for _, r := range objets.pistes.recordsDu(k) {
			if int(r.ti) != ti {
				continue
			}
			c := cle{r.slot, uint32(r.gen)}
			ts := p.paquet.TimestampUS
			if v := enCours[c]; v != nil && ts <= v.dernierTS+1_000_000 {
				v.dernierTS = ts
				v.nbEchantil++
				continue
			}
			v := &grenVolNaissance{ts: ts, slot: r.slot, gen: uint32(r.gen), dansBande: bande[r.slot], dernierTS: ts, nbEchantil: 1}
			enCours[c] = v
			out = append(out, v)
		}
	}
	res := make([]grenVolNaissance, len(out))
	for i, v := range out {
		res[i] = *v
	}
	return res
}

// grenVolMessages rend les instants des messages de genre 5 et 6 de la vue A du film.
func grenVolMessages(fc *FilmContext) (detos, impacts []uint64) {
	bal, g := fc.profilDeLaVueA(), fc.grammaireDeLaVueA()
	for _, num := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			flux := lireLaVueA(pk.Payload(data), 1, bal, g)
			lus := len(flux.Genres)
			if !flux.Porte && lus > 0 {
				lus-- // le dernier genre est celui qui a arrete la lecture
			}
			for _, genre := range flux.Genres[:lus] {
				switch genre {
				case 5:
					detos = append(detos, pk.TimestampUS)
				case 6:
					impacts = append(impacts, pk.TimestampUS)
				}
			}
		}
	}
	return detos, impacts
}
