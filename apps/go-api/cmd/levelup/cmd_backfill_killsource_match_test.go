package main

// cmd_backfill_killsource_match_test.go — `--match` (plan Emprise vies, V5.2a) : la resolution des
// identifiants et le refus des combinaisons, sans base. La selection sur une vraie base migree est
// dans cmd_backfill_killsource_match_integration_test.go.

import (
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
)

var registreDeTest = []string{
	"aaaaaaaa-1111-0000-0000-000000000001",
	"bbbbbbbb-2222-0000-0000-000000000002",
	"abcd1234-0001-0000-0000-000000000003",
	"abcd1234-0002-0000-0000-000000000004",
}

func idsTries(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func TestResoudreMatchs_PrefixeUnivoqueEtIdentifiantComplet(t *testing.T) {
	// Un prefixe de 8 caracteres univoque, une casse differente, un identifiant complet, un doublon.
	got, err := resoudreMatchs(registreDeTest,
		"AAAAAAAA, bbbbbbbb-2222-0000-0000-000000000002,aaaaaaaa,,")
	if err != nil {
		t.Fatalf("resoudreMatchs: %v", err)
	}
	want := []string{registreDeTest[0], registreDeTest[1]}
	if !reflect.DeepEqual(idsTries(got), want) {
		t.Errorf("resolu = %v, attendu %v", idsTries(got), want)
	}
}

func TestResoudreMatchs_Refus(t *testing.T) {
	cas := []struct{ nom, brut, fragment string }{
		{"prefixe ambigu", "abcd1234", "AMBIGU"},
		{"prefixe inconnu", "deadbeef", "aucun match du registre"},
		{"trop court", "aaaaaaa", "au moins 8"},
		{"un bon et un ambigu", "aaaaaaaa,abcd1234", "AMBIGU"},
		{"un bon et un inconnu", "aaaaaaaa,deadbeef", "aucun match du registre"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := resoudreMatchs(registreDeTest, c.brut)
			if err == nil {
				t.Fatalf("aucune erreur, resolu = %v", idsTries(got))
			}
			if !strings.Contains(err.Error(), c.fragment) {
				t.Errorf("erreur = %q, attendu un message contenant %q", err.Error(), c.fragment)
			}
			if got != nil {
				t.Errorf("un refus ne doit rien resoudre, resolu = %v", idsTries(got))
			}
		})
	}
	// Un prefixe plus long leve l ambiguite.
	got, err := resoudreMatchs(registreDeTest, "abcd1234-0002")
	if err != nil || len(got) != 1 || !got[registreDeTest[3]] {
		t.Errorf("abcd1234-0002 : %v, %v — attendu le seul quatrieme match", idsTries(got), err)
	}
}

func TestValiderLesOptions_Match(t *testing.T) {
	cas := []struct {
		nom      string
		o        killsourceOptions
		fragment string
	}{
		{"hors ligne", killsourceOptions{workers: 1, match: "aaaaaaaa"}, ""},
		{"avec force et dry-run", killsourceOptions{workers: 1, match: "aaaaaaaa", force: true, dryRun: true}, ""},
		{"avec online", killsourceOptions{workers: 1, match: "aaaaaaaa", online: true, gamertag: "JGtm"}, "--match n a de sens que HORS LIGNE"},
		{"avec credit-only", killsourceOptions{workers: 1, match: "aaaaaaaa", creditOnly: true}, "--credit-only"},
		{"liste vide de virgules", killsourceOptions{workers: 1, match: " , ,"}, "aucun identifiant"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			err := validerLesOptions(c.o)
			if c.fragment == "" {
				if err != nil {
					t.Fatalf("invocation valide refusee : %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.fragment) {
				t.Errorf("erreur = %v, attendu un message contenant %q", err, c.fragment)
			}
		})
	}
}

// Le refus part a la porte de la commande : avant toute ouverture de base.
func TestBackfillKillSource_MatchRefuseAvecOnline(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	err := runBackfillKillSource(cfg, []string{"--online", "--gamertag", "JGtm", "--match", "aaaaaaaa", "--dry-run"})
	if err == nil || !strings.Contains(err.Error(), "--match n a de sens que HORS LIGNE") {
		t.Fatalf("erreur = %v, attendu le refus de --match avec --online", err)
	}
}

// capturerSortie rend ce que `f` ecrit sur la sortie standard.
func capturerSortie(t *testing.T, f func()) string {
	t.Helper()
	ancien := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = ancien
	b, _ := io.ReadAll(r)
	return string(b)
}

// Sous `--match` le plan liste CHAQUE film ; sans, le milieu d une longue liste est elide.
func TestAfficherPlan_ToutSousMatch(t *testing.T) {
	var c []filmCandidat
	for i := 0; i < 12; i++ {
		c = append(c, filmCandidat{matchID: "film" + string(rune('a'+i)), chunks: i + 1})
	}
	complet := capturerSortie(t, func() { afficherPlan(c, true) })
	for _, f := range c {
		if !strings.Contains(complet, f.matchID+" ") {
			t.Errorf("le plan complet n affiche pas %s", f.matchID)
		}
	}
	court := capturerSortie(t, func() { afficherPlan(c, false) })
	if strings.Contains(court, "filmg ") || !strings.Contains(court, "filma ") || !strings.Contains(court, "filml ") {
		t.Errorf("le plan par defaut doit garder les bouts et elider le milieu :\n%s", court)
	}
}
