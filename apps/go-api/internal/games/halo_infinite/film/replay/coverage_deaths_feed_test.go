package replay

// coverage_deaths_feed_test.go — LE DOCUMENT DIT SI LE FIL DES MORTS EST VIDE OU ILLISIBLE (lot
// M5.2 des retours rejeu, 2026-09-23).
//
// Avant ce lot, un fil illisible (film archive avant sa finalisation, `ab526724`) ne se lisait
// que dans les journaux de cuisson : le document publiait des compteurs a zero, indistinguables
// d un match sans mort. `coverage.bridge.deathsFeed` porte le VERDICT de la lecture.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestLectureDuFilDesMorts_TroisIssues : lu, vide, illisible — et un fil vide n est pas une panne.
func TestLectureDuFilDesMorts_TroisIssues(t *testing.T) {
	une := []Death{{XUID: 1, TimeMS: 1}}
	for _, cas := range []struct {
		nom    string
		deaths []Death
		err    error
		want   string
	}{
		{"au moins une mort", une, nil, DeathsFeedRead},
		{"morceau lu sans mort", nil, fmt.Errorf("chunk highlight (36) : %w", ErrFilDesMortsSansMort), DeathsFeedEmpty},
		{"aucun morceau des temps forts", nil, ErrFilSansTempsForts, DeathsFeedUnreadable},
		{"evenements illisibles", nil, errors.New("chunk highlight (33) : en-tete inconnu"), DeathsFeedUnreadable},
	} {
		if got := lectureDuFilDesMorts(cas.deaths, cas.err); got != cas.want {
			t.Errorf("%s : %q, attendu %q", cas.nom, got, cas.want)
		}
	}
}

// TestLectureDuFilDesMorts_SurLaBobine : le verdict sur de VRAIS octets — la bobine v40 lue avec
// son morceau des temps forts, puis typee sans lui (la signature d un film non finalise).
func TestLectureDuFilDesMorts_SurLaBobine(t *testing.T) {
	o := octetsBobineV40(t)
	for _, cas := range []struct {
		nom  string
		meta []types.ChunkMeta
		want string
	}{
		{"temps forts presents", []types.ChunkMeta{
			{Index: 0, ChunkType: 1}, {Index: 1, ChunkType: 2}, {Index: 2, ChunkType: filmcache.ChunkTypeTempsForts},
		}, DeathsFeedRead},
		{"film non finalise", []types.ChunkMeta{
			{Index: 0, ChunkType: 1}, {Index: 1, ChunkType: 2}, {Index: 2, ChunkType: 2},
		}, DeathsFeedUnreadable},
	} {
		film, err := source.Load(source.MemoryChunks(o), cas.meta)
		if err != nil {
			t.Fatalf("%s : chargement : %v", cas.nom, err)
		}
		deaths, err := ScanDeaths(film)
		if got := lectureDuFilDesMorts(deaths, err); got != cas.want {
			t.Errorf("%s : %q (err %v), attendu %q", cas.nom, got, err, cas.want)
		}
	}
}

// TestDocumentPublieLeVerdictDuFilDesMorts : le verdict du balayage est PUBLIE ; sans balayage, un
// fil non vide se dit `read` et un fil vide se TAIT (la cle est absente, jamais devinee).
func TestDocumentPublieLeVerdictDuFilDesMorts(t *testing.T) {
	pos := positionsPourOrigine()
	for _, cas := range []struct {
		nom  string
		opt  Options
		want string
	}{
		{"balaye, illisible", Options{DeathsFeed: DeathsFeedUnreadable}, DeathsFeedUnreadable},
		{"balaye, vide", Options{DeathsFeed: DeathsFeedEmpty}, DeathsFeedEmpty},
		{"sans balayage, fil non vide", Options{Deaths: []Death{{XUID: 1, TimeMS: 1}}}, DeathsFeedRead},
		{"sans balayage, fil vide", Options{}, ""},
	} {
		cas.opt.FilmClockOriginUS = 1_000_000
		doc := BuildFromPositions("m", "halo_infinite", pos, nil, cas.opt)
		if doc.Coverage == nil {
			t.Fatalf("%s : document sans couverture", cas.nom)
		}
		if got := doc.Coverage.Bridge.DeathsFeed; got != cas.want {
			t.Errorf("%s : deathsFeed = %q, attendu %q", cas.nom, got, cas.want)
		}
		blob, err := json.Marshal(doc.Coverage.Bridge)
		if err != nil {
			t.Fatal(err)
		}
		if present := strings.Contains(string(blob), `"deathsFeed"`); present != (cas.want != "") {
			t.Errorf("%s : cle deathsFeed presente = %v dans %s", cas.nom, present, blob)
		}
	}
}
