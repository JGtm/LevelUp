//go:build research

package replay

// equipes_b1ad85eb_research_test.go — SONDE EN LECTURE SEULE (enquete « equipes b1ad85eb »,
// 2026-09-23). Relit les FAITS PERSISTES d'au plus trois films (jamais un film) et imprime ce que
// le film a dit de l'identite, de l'equipe et de la presence de chaque participant :
//   - la table de chunk_00, la table d'index des chunks, les bots declares (BOT_METADATA) ;
//   - le rapport de ScanPlayerTeams (divergences par ENTITE vs par INDEX) et la table publiee ;
//   - l'index de participant LU dans le record de creation de chaque corps, et la fenetre de
//     positions de chaque slot de bipede (publie ou non) ;
//   - la fenetre des tirs de chaque index de joueur.
// Aucune ecriture. Lancement :
//   go test -tags research -count=1 -run TestEquipesSondeFaits ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

const sondeRepo = "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration"

func sondeEntree(t *testing.T, id string, blob []byte) profile.MapQuantEntry {
	t.Helper()
	ent, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("%s : en-tete : %v", id, err)
	}
	t.Logf("%s : module=%q axisW=%v layoutDetecte=%v", id, ent.MapModule, ent.AxisW, ent.LayoutDetected)
	if ent.LayoutDetected {
		return profile.MapQuantEntry{Module: ent.MapModule}
	}
	cat, err := profile.LoadMapQuantCatalog(sondeRepo + "/data/titles/halo_infinite/reference/map_quant_bounds.json")
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, e := range cat.Maps {
		if e.Module == ent.MapModule && e.AxisWidths == ent.AxisW {
			return e
		}
	}
	return profile.MapQuantEntry{Module: ent.MapModule, AxisWidths: ent.AxisW}
}

func TestEquipesSondeFaits(t *testing.T) {
	ids := strings.Split(os.Getenv("SONDE_IDS"), ",")
	if len(ids) > 3 {
		t.Fatalf("au plus trois fichiers de faits")
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		blob, err := os.ReadFile(sondeRepo + "/data/cache/film_facts/halo_infinite/" + id + ".filmfacts.bin")
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		f, err := DecodeFilmFactsFile(blob, sondeEntree(t, id, blob))
		if err != nil {
			t.Fatalf("%s : faits : %v", id, err)
		}
		sondeRapport(t, id, f)
	}
}

func sondeRapport(t *testing.T, id string, f *FilmFactsFile) {
	in := f.Facts.FilmInputs
	var origin uint64
	for i, p := range in.Positions {
		if i == 0 || p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	const step = uint64(100_000)
	fr := func(ts uint64) int64 { return (int64(ts) - int64(origin)) / int64(step) }
	t.Logf("=== %s origine=%d us", id, origin)
	t.Logf("TABLE chunk_00 : lue=%v refus=%q occupes=%d vacants=%d intercale=%v build=%s",
		in.FilmTable.Lue(), in.FilmTable.Refusal, in.FilmTable.Occupied, in.FilmTable.Vacant,
		in.FilmTable.InterleavedVacant, in.FilmTable.Build)
	for _, s := range in.FilmTable.Seats {
		t.Logf("   siege %2d xuid=%d %q", s.FilmIndex, s.XUID, s.Gamertag)
	}
	idx := make([]string, 0, len(in.PlayerIndices.ByXUID))
	for x, i := range in.PlayerIndices.ByXUID {
		idx = append(idx, fmt.Sprintf("%d->%d", x, i))
	}
	sort.Strings(idx)
	t.Logf("TABLE D'INDEX (chunks) : lectures=%d desaccords=%d %v", in.PlayerIndices.Readings,
		in.PlayerIndices.Disagreements, idx)
	r := in.TeamScan
	t.Logf("EQUIPES ScanPlayerTeams : paquets=%d records=%d lus=%d nonAtteints=%d horsIndex=%d horsValeur=%d "+
		"ENTITES=%d divEntite=%d divIndex=%d indicesPublies=%d sansEquipe=%d composant=%q",
		r.Packets, r.Records, r.Read, r.Unreached, r.OutOfDomainIndex, r.OutOfDomainValue,
		r.Entities, r.EntityDivergences, r.IndexDivergences, r.Indices, r.NoTeam, r.Component)
	eq := make([]int, 0, len(in.PlayerTeams))
	for i := range in.PlayerTeams {
		eq = append(eq, i)
	}
	sort.Ints(eq)
	var sb strings.Builder
	for _, i := range eq {
		fmt.Fprintf(&sb, " %d:%d", i, in.PlayerTeams[i])
	}
	t.Logf("   table publiee index:designateur ->%s", sb.String())
	if f.Kills != nil {
		ro := f.Kills.Roster
		t.Logf("KILLSOURCE roster : humains=%d botsSuccedes=%d bots=%+v nonEpingles=%+v", ro.Humans,
			ro.BotsSuccedes, ro.Bots, ro.UnpinnedBots)
		for i, n := range ro.IndexToName {
			src := ""
			if i < len(ro.IndexSource) {
				src = fmt.Sprint(ro.IndexSource[i])
			}
			t.Logf("   indice %2d -> %q (%s)", i, n, src)
		}
	}
	// Creations : index LU par corps.
	type fen struct{ a, b, n int64 }
	pos := map[uint32]*fen{}
	for _, p := range in.Positions {
		w := pos[p.Slot]
		f0 := fr(p.TimestampUS)
		if w == nil {
			pos[p.Slot] = &fen{f0, f0, 1}
			continue
		}
		if f0 < w.a {
			w.a = f0
		}
		if f0 > w.b {
			w.b = f0
		}
		w.n++
	}
	crea := map[uint32][]string{}
	for _, c := range in.BipedCreations {
		s := fmt.Sprintf("f%d gen%d idx=%d(ok=%v)", fr(c.TimestampUS), c.Generation, c.ParticipantIndex, c.HasIndex)
		crea[c.Slot] = append(crea[c.Slot], s)
	}
	slots := make([]int, 0, len(pos))
	for s := range pos {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)
	t.Logf("CORPS (slot : positions [premiere..derniere] n ; creations)")
	for _, s := range slots {
		w := pos[uint32(s)]
		t.Logf("   slot %4d : [%5d..%5d] n=%5d ; %v", s, w.a, w.b, w.n, crea[uint32(s)])
	}
	for s, c := range crea {
		if pos[s] == nil {
			t.Logf("   slot %4d SANS POSITION : %v", s, c)
		}
	}
	// Tirs par index de joueur.
	tirs := map[int]*fen{}
	for _, e := range in.Fire {
		w := tirs[e.FilmIndex]
		f0 := fr(e.TimestampUS)
		if w == nil {
			tirs[e.FilmIndex] = &fen{f0, f0, 1}
			continue
		}
		if f0 < w.a {
			w.a = f0
		}
		if f0 > w.b {
			w.b = f0
		}
		w.n++
	}
	ks := make([]int, 0, len(tirs))
	for k := range tirs {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		w := tirs[k]
		t.Logf("TIRS index %2d : [%5d..%5d] n=%d", k, w.a, w.b, w.n)
	}
	// Vidage des tirs (index, frame) pour la jointure cote Node avec les vies publiees.
	if dir := os.Getenv("SONDE_DUMP"); dir != "" {
		var sbt strings.Builder
		for _, e := range in.Fire {
			fmt.Fprintf(&sbt, "%d\t%d\n", e.FilmIndex, fr(e.TimestampUS))
		}
		if err := os.WriteFile(dir+"/tirs_"+id+".tsv", []byte(sbt.String()), 0o644); err != nil {
			t.Fatalf("vidage : %v", err)
		}
	}
}
