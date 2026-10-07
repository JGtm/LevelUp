package replay

// journal_des_places_test.go — LE JOURNAL DES PLACES DIT UN DEFAUT DU ROSTER PUBLIE, ET DE LUI SEUL
// ([journaliserLesPlaces], [Options.documentInterne]).
//
//	JP-INTERNE  le document que `PortagesAuSync` assemble pour un Oddball ne lit pas les equipes du film
//	            (garde du drapeau seulement) : chaque entree presente y est sans equipe, et ce document,
//	            qui n'est pas publie, ne l'ecrit pas en ERREUR ;
//	JP-PUBLIE   les memes entrees republiees depuis les faits (`BuildFromFacts`, options d'une cuisson) :
//	            l'entree sans equipe reste une ERREUR, comptee dans `coverage.seats.sansEquipe`.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// extraitSansEquipe : le message d'ERREUR de [journaliserLesPlaces] pour une entree presente sans equipe.
const extraitSansEquipe = "SANS EQUIPE lue"

// joueursSansEquipe : le nombre de joueurs de [lecturesSansEquipe].
const joueursSansEquipe = 4

// lecturesSansEquipe : quatre joueurs presents tout le film (corps 100 a 103, index 0 a 3, xuids 1000 a
// 1003), nommes par le lien direct de creation et la table d'index ; ni table du film, ni entite ti=9,
// ni equipe lue — le roster d'un Oddball tel que le chemin des porteurs au sync l'assemble.
func lecturesSansEquipe() IdentityInput {
	in := IdentityInput{PlayerIndices: types.PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: joueursSansEquipe}}
	for i := range joueursSansEquipe {
		slot := uint32(100 + i) //nolint:gosec // index de test
		in.BipedCreations = append(in.BipedCreations, grammar.BipedCreation{Slot: slot, Generation: 1,
			HasIndex: true, ParticipantIndex: uint32(i)}) //nolint:gosec // index de test
		for ms := 0; ms < 10_000; ms += 100 {
			in.Positions = append(in.Positions, pos(slot, ms, float32(i*10+ms%7), float32(ms%13), 1))
		}
		in.PlayerIndices.ByXUID[uint64(1000+i)] = i
	}
	return in
}

// statborgVide : le lecteur du statborg d'un film sans enregistrement ni repli (aucun octet lu).
func statborgVide(*source.Film, string) ([]types.StatRecord, bool, objectives.ComptesDesReplis, []constat.Diagnostic) {
	return nil, false, objectives.ComptesDesReplis{}, nil
}

// journalDe joue `jouer` sous un journal JSON capture et rend ses enregistrements (niveau, message).
func journalDe(t *testing.T, jouer func()) [][2]string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(prev)
	jouer()
	var out [][2]string
	for ligne := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		var r struct{ Level, Msg string }
		if json.Unmarshal([]byte(ligne), &r) == nil {
			out = append(out, [2]string{r.Level, r.Msg})
		}
	}
	return out
}

func TestPortagesAuSync_LeDocumentInterneNeJournalisePasLesPlaces(t *testing.T) { // JP-INTERNE
	e := EntreePorteursAuSync{MatchID: "temoin", Variante: "Oddball:Arena", Identite: lecturesSansEquipe(),
		lireStatborg: statborgVide}
	// LE TEMOIN N'EST PAS VIDE : les memes entrees, sous les options que ce chemin pose, donnent un roster
	// dont chaque entree presente est sans equipe — ce que le journal des places ecrit en ERREUR.
	var temoin ReplayDocument
	journalDe(t, func() {
		temoin = BuildFromPositions(context.Background(), e.MatchID, "", e.Identite.Positions, nil, e.optionsDuRegistre(nil))
	})
	if s := temoin.Coverage.Seats; s == nil || s.SansEquipe != joueursSansEquipe {
		t.Fatalf("couverture des places du temoin %+v : attendu %d entrees presentes sans equipe", s, joueursSansEquipe)
	}
	var b BilanPortages
	journal := journalDe(t, func() { _, b = PortagesAuSync(context.Background(), e) })
	if !b.Gardes.Crane || b.Lectures.Equipes {
		t.Fatalf("gardes %+v, lectures %+v : attendu la garde du crane, sans lecture des equipes du film",
			b.Gardes, b.Lectures)
	}
	if aUnEnregistrement(journal, "ERROR", extraitSansEquipe) {
		t.Errorf("journal %v : le document des porteurs au sync n'est pas publie, ses entrees sans equipe ne "+
			"s'ecrivent pas en ERREUR", journal)
	}
}

func TestBuildFromFacts_LeDocumentPublieJournaliseLEntreeSansEquipe(t *testing.T) { // JP-PUBLIE
	lu := lecturesSansEquipe()
	faits := &FilmFactsFile{Facts: FilmFacts{Film: "temoin", FilmInputs: FilmInputs{Positions: lu.Positions,
		BipedCreations: lu.BipedCreations, PlayerIndices: lu.PlayerIndices}}}
	var doc ReplayDocument
	journal := journalDe(t, func() {
		doc = BuildFromFacts(context.Background(), "temoin", "halo_infinite", faits, Options{})
	})
	if s := doc.Coverage.Seats; s == nil || s.SansEquipe != joueursSansEquipe {
		t.Fatalf("couverture des places %+v : attendu %d entrees presentes sans equipe", s, joueursSansEquipe)
	}
	if !aUnEnregistrement(journal, "ERROR", extraitSansEquipe) {
		t.Errorf("journal %v : une entree du roster PUBLIE sans equipe s'ecrit en ERREUR", journal)
	}
}
