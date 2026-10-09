package replay

// identity_registry_bridge_journal_test.go — LES DISCORDANCES DU PONT PAR MORTS SE PUBLIENT UNE
// FOIS PAR FILM : un compteur et une trace agregee au niveau Info (compte, concordances, slots),
// jamais une ligne Warn par vie. Le detail par vie reste au niveau Debug.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/observability"
)

// pontDeTest : quatre vies appariees. Deux discordances sur le slot 3, une concordance (slot 5),
// une vie nommee par deduction (slot 7) que la verification ne confronte pas.
func pontDeTest() ([]lifeSpan, []types.Death, []deathPair) {
	lives := []lifeSpan{
		{slot: 3, xuid: 10, nomPar: NomParCreation},
		{slot: 5, xuid: 20, nomPar: NomParCreation},
		{slot: 3, xuid: 10, nomPar: NomParCreationPropagee},
		{slot: 7, xuid: 30, nomPar: NomParMort},
	}
	deaths := []types.Death{{XUID: 11}, {XUID: 20}, {XUID: 12}, {XUID: 99}}
	pairs := []deathPair{{li: 0, di: 0}, {li: 1, di: 1}, {li: 2, di: 2}, {li: 3, di: 3}}
	return lives, deaths, pairs
}

func TestPontParMorts_DiscordancesPublieesUneFoisParFilm(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	lives, deaths, pairs := pontDeTest()
	v := verifierParLesMorts(context.Background(), lives, deaths, pairs, "m-test")
	if v.Discordant != 2 || v.Concordant != 1 {
		t.Fatalf("discordances %d / concordances %d, attendu 2 / 1", v.Discordant, v.Concordant)
	}
	if want := []uint32{3}; !reflect.DeepEqual(v.SlotsDiscordants, want) {
		t.Fatalf("slots discordants = %v, attendu %v (distincts)", v.SlotsDiscordants, want)
	}

	avant := observability.LoadCounter(compteurDiscordancesDuPont)
	IdentityRegistry{bridge: v}.publierDiscordancesDuPont(context.Background(), "m-test")
	if d := observability.LoadCounter(compteurDiscordancesDuPont) - avant; d != 2 {
		t.Fatalf("compteur %s : +%d, attendu +2", compteurDiscordancesDuPont, d)
	}

	var parVie, agregees int
	for _, ligne := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(ligne), &rec); err != nil {
			t.Fatalf("journal illisible %q : %v", ligne, err)
		}
		msg, _ := rec["msg"].(string)
		if !strings.Contains(msg, "pont par morts") {
			continue
		}
		switch {
		case strings.Contains(msg, "sur des vies"):
			agregees++
			if rec["level"] != "INFO" {
				t.Errorf("trace agregee au niveau %v, attendu INFO", rec["level"])
			}
			if rec["discordances"] != float64(2) {
				t.Errorf("trace agregee : discordances = %v, attendu 2", rec["discordances"])
			}
		default:
			parVie++
			if rec["level"] != "DEBUG" {
				t.Errorf("detail par vie au niveau %v, attendu DEBUG", rec["level"])
			}
		}
	}
	if parVie != 2 || agregees != 1 {
		t.Fatalf("%d lignes par vie et %d agregees, attendu 2 (Debug) et 1 (Info)", parVie, agregees)
	}
}

func TestPontParMorts_SansDiscordanceRienNePublie(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	avant := observability.LoadCounter(compteurDiscordancesDuPont)
	IdentityRegistry{bridge: bridgeVerification{Concordant: 4}}.publierDiscordancesDuPont(context.Background(), "m")
	if observability.LoadCounter(compteurDiscordancesDuPont) != avant || buf.Len() != 0 {
		t.Fatalf("un film sans discordance ne doit rien publier (journal %q)", buf.String())
	}
}
