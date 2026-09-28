package replaybuild

// pont_siege_recycle_test.go — LE SIEGE RECYCLE SUR UN FILM REEL (lot R1 du plan de suite d audit,
// constat C1 du rapport `.ai/V7.5/film_re/G_CORPUS_J11_2026-09-28.md`).
//
// `bcb6d393` (Cliffhanger, CTF, une manche) porte deux sieges statborg recycles : le slot 12 (un
// joueur part, un remplacant le reprend) et le slot 14 (trois occupants). Depuis le lot J8.5, le
// frag de 70 706 ms — celui du PREMIER occupant du slot 12, mort 16 ms plus tard sous son nom au fil
// des morts — etait publie au nom du remplacant, qui ne meurt pour la premiere fois qu'a 231 s.
//
// LE FIXTURE EST REEL ET FIGE : les enregistrements d entite du film (tels que la porte du statborg
// les rend), son fil des morts (xuid et instant seuls) et les faits du match. Aucun octet de film au
// test. Regeneration (la seule porte ; le fixture ne s edite jamais a la main) :
//
//	REPLAYBUILD_FILM_CACHE=<repo>/data/cache \
//	  go test ./internal/replaybuild/ -run TestPontSiegeRecycleRegenerer -update-pont-siege
//
// Les faits viennent du fixture d equivalence deja versionne
// (`film/replay/testdata/equivalence/bcb6d393.facts.json`, `levelup replay-facts-export`).

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/port"
)

var updatePontSiege = flag.Bool("update-pont-siege", false,
	"reecrire testdata/pont_bcb6d393.json.gz depuis le cache film (REPLAYBUILD_FILM_CACHE)")

const (
	pontSiegeMatch   = "bcb6d393-8f5e-40e3-806b-e3cb35bfc9b0"
	pontSiegeFixture = "testdata/pont_bcb6d393.json.gz"
	pontSiegeFaits   = "../games/halo_infinite/film/replay/testdata/equivalence/bcb6d393.facts.json"
)

// pontSiegeEntrees : ce que le fixture fige.
type pontSiegeEntrees struct {
	Records []types.StatRecord `json:"records"`
	Deaths  []types.Death      `json:"deaths"`
	Facts   port.MatchFacts    `json:"facts"`
}

// TestPontSiegeRecycleRegenerer est la SEULE porte d ecriture du fixture.
func TestPontSiegeRecycleRegenerer(t *testing.T) {
	cache := os.Getenv("REPLAYBUILD_FILM_CACHE")
	if !*updatePontSiege || cache == "" {
		t.Skip("regeneration : -update-pont-siege et REPLAYBUILD_FILM_CACHE requis")
	}
	ctx := context.Background()
	film, err := chargerLeFilmDeLaCuisson(ctx, pontSiegeMatch, filepath.Join(cache, "film_chunks", "bcb6d393"))
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	sb := statborgDuFilm(ctx, pontSiegeMatch, film)
	morts := lireMorts(film)
	if morts.err != nil {
		t.Fatalf("fil des morts : %v", morts.err)
	}
	in := pontSiegeEntrees{Records: sb.Records, Facts: lireFaitsDuFixture(t)}
	for _, d := range morts.list {
		in.Deaths = append(in.Deaths, types.Death{XUID: d.XUID, TimeMS: d.TimeMS}) // sans gamertag
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pontSiegeFixture, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s : %d enregistrements, %d morts, %d octets", pontSiegeFixture, len(in.Records),
		len(in.Deaths), buf.Len())
}

func lireFaitsDuFixture(t *testing.T) port.MatchFacts {
	t.Helper()
	raw, err := os.ReadFile(pontSiegeFaits)
	if err != nil {
		t.Fatalf("faits : %v", err)
	}
	var f port.MatchFacts
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("faits : %v", err)
	}
	return f
}

func chargerPontSiege(t *testing.T) pontSiegeEntrees {
	t.Helper()
	raw, err := os.ReadFile(pontSiegeFixture)
	if err != nil {
		t.Fatalf("fixture illisible : %v — regenerer (cf. l en-tete)", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	var in pontSiegeEntrees
	if err := json.NewDecoder(zr).Decode(&in); err != nil {
		t.Fatal(err)
	}
	return in
}

// TestSiegeRecycleBcb6d393 : les trois actions du constat C1 recoivent le joueur que le FILM place
// sur le siege a leur instant, et aucune action n est publiee pour un joueur absent du siege.
//
// ROUGE OBSERVE AVANT LE LOT : 70 706 ms -> 2535468064146356 (le remplacant, dont la premiere mort
// tombe a 231 179 ms).
func TestSiegeRecycleBcb6d393(t *testing.T) {
	in := chargerPontSiege(t)
	fs := assemblerFilmStats(context.Background(), pontSiegeMatch,
		replay.FilmStatborg{Records: in.Records, ChunkStartMS: map[int]int{0: 0}}, in.Facts, filmDeaths{list: in.Deaths})
	parInstant := map[int]map[int]string{}
	for _, e := range fs.objectives {
		if parInstant[e.TimeMS] == nil {
			parInstant[e.TimeMS] = map[int]string{}
		}
		parInstant[e.TimeMS][e.Slot] = e.XUID
	}
	for _, c := range []struct {
		t, slot  int
		want     string
		pourquoi string
	}{
		{70706, 12, "2535460750735339", "frag du premier occupant du slot 12 (il meurt a 70 732 ms au fil)"},
		{274479, 14, "2533274811363842", "premier frag du dernier occupant du slot 14 (siege repris a 232 705 ms)"},
		{347519, 14, "2533274811363842", "second frag du dernier occupant du slot 14"},
	} {
		if got := parInstant[c.t][c.slot]; got != c.want {
			t.Errorf("action de %d ms (slot %d) : %q, attendu %q — %s", c.t, c.slot, got, c.want, c.pourquoi)
		}
	}
	if fs.objectivesUnnamed != 0 {
		t.Errorf("%d action(s) d objectif sans auteur, attendu 0", fs.objectivesUnnamed)
	}
	// LES OCCUPATIONS PUBLIEES : deux sur le slot 12, trois sur le slot 14 (l occupant du milieu
	// meurt une seule fois : le pont s abstient sur son intervalle).
	id := fs.statborgIdentity
	for _, c := range []struct {
		slot int
		want []string
	}{
		{12, []string{"2535460750735339", "2535468064146356"}},
		{14, []string{"2533274876732804", "", "2533274811363842"}},
	} {
		occ := id.Occupations(0, c.slot)
		got := make([]string, 0, len(occ))
		for _, o := range occ {
			got = append(got, o.XUID)
		}
		if len(got) != len(c.want) {
			t.Errorf("slot %d : occupations %q, attendu %q", c.slot, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("slot %d : occupations %q, attendu %q", c.slot, got, c.want)
				break
			}
		}
	}
}
