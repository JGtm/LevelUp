//go:build research

package replay

// equipe_arrivee_bot_research_test.go — E8.4 (D-18) : D'OU VIENT L'EQUIPE LUE SUR LE CORPS D'UN BOT
// QUI ARRIVE SUR UN INDEX DEJA TENU AVANT LUI ?
//
// UN film, decode sous `filmproc.AcquireSolo` et le plafond memoire, rien d'ecrit, sur la carte du
// film (`NewFilmContextForMap` + `PoserLaCarteEtLeDecoupage`, comme la cuisson). L'instrument imprime,
// pour l'index de participant EAR_INDEX : les paquets BOT_METADATA qui le declarent (instant, bot,
// equipe ecrite), les entites `ti=9` de cet index (slot, equipe, premiere et derniere image-cle
// porteuse), la table d'equipe par index que le balayage publie, et les records de creation de bipede
// de cet index (instant, slot).
//
//	EAR_FILMS=<repo>/data/cache/film_chunks EAR_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  EAR_MODULE=fo08_wetland EAR_BUILD=HI_1_13_0 EAR_ID=3f048bf3 EAR_INDEX=8 EAR_VERROU=<repo>/data/cache \
//	  go test -tags research -count=1 -run '^TestEquipeALArriveeDUnBot$' -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestEquipeALArriveeDUnBot(t *testing.T) {
	films, catalogue, module, id, verrou, build := os.Getenv("EAR_FILMS"), os.Getenv("EAR_CATALOGUE"),
		os.Getenv("EAR_MODULE"), os.Getenv("EAR_ID"), os.Getenv("EAR_VERROU"), os.Getenv("EAR_BUILD")
	index, errIdx := strconv.Atoi(os.Getenv("EAR_INDEX"))
	if films == "" || catalogue == "" || module == "" || id == "" || verrou == "" || errIdx != nil {
		t.Skip("EAR_* absent : instrument saute")
	}
	lock, err := filmproc.AcquireSolo(verrou, "equipe-arrivee-bot", id)
	if err != nil {
		t.Fatalf("verrou : %v", err)
	}
	defer lock.Release()
	filmproc.LowerOwnPriority("equipe-arrivee-bot")
	garde := filmproc.Arm("equipe-arrivee-bot", 3, func(pic uint64) { panic(fmt.Sprintf("plafond memoire franchi : %d", pic)) })
	defer garde.Disarm()

	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	var entry profile.MapQuantEntry
	for _, m := range cat.Maps {
		if m.Module == module {
			entry = m
		}
	}
	film, err := source.LoadDir(filepath.Join(films, id), nil)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	fc := grammar.NewFilmContextForMap(film, &entry, nil)
	fc.PoserLaCarteEtLeDecoupage()

	for _, p := range grammar.PaquetsBotMetadata(film, grammar.LectureDesBotsPour(build)) {
		var decl []string
		for _, e := range p.Ecrites {
			decl = append(decl, fmt.Sprintf("%d:bid(%d.0) %s equipe=%d", e.Slot, e.BotID, e.Nom, e.Equipe))
		}
		if len(p.Ecrites) == 0 {
			for _, e := range p.Balayees {
				decl = append(decl, fmt.Sprintf("%d:bid(%d.0) %s (balayage seul)", e.Slot, e.BotID, e.Nom))
			}
		}
		t.Logf("BOT_METADATA %10d us nbBots=%d ferme=%v %v", p.Instant, p.NBots, p.Ferme, decl)
	}
	parIndex, rep, scan := grammar.ScanPlayerTeams(fc)
	t.Logf("table d'equipe par index : index %d -> %v (present=%v) ; divergences d'index=%d", index,
		parIndex[index], hasKey(parIndex, index), rep.IndexDivergences)
	for _, e := range scan.Entities {
		if e.Index != index {
			continue
		}
		de, _ := scan.KeyframeUS(e.FirstKF)
		a, _ := scan.KeyframeUS(e.LastKF)
		t.Logf("entite ti=9 slot=%d index=%d equipe=%d images-cles [%d..%d] = [%d..%d] us vue=%d instable=%v",
			e.Slot, e.Index, e.Team, e.FirstKF, e.LastKF, de, a, e.Seen, e.Unstable)
	}
	t.Logf("images-cles porteuses : %v", scan.KeyframesUS)
	creations, _, err := grammar.ScanBipedCreations(fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	for _, c := range creations {
		if c.HasIndex && int(c.ParticipantIndex) == index {
			t.Logf("creation de bipede slot=%d gen=%d index=%d a %d us", c.Slot, c.Generation, c.ParticipantIndex, c.TimestampUS)
		}
	}
}

func hasKey(m map[int]int, k int) bool {
	_, ok := m[k]
	return ok
}
