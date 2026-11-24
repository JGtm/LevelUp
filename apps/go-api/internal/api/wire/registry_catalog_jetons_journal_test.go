package wire

// registry_catalog_jetons_journal_test.go — le refus de jeton de lecture publique se journalise UNE fois par
// épisode : l'avertissement part quand aucun compte ne peut servir, les refus suivants se disent au niveau
// débogage, et le retour d'un jeton clôt l'épisode (le refus suivant avertit de nouveau).

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

type tampon struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (t *tampon) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.b.Write(p)
}

func (t *tampon) lignes() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(t.b.String()), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func capturerJournalSurveillance(t *testing.T) *tampon {
	t.Helper()
	tp := &tampon{}
	ancien := monitoringLog
	monitoringLog = slog.New(slog.NewTextHandler(tp, &slog.HandlerOptions{Level: slog.LevelDebug}))
	t.Cleanup(func() { monitoringLog = ancien })
	return tp
}

func compter(lignes []string, niveau, fragment string) int {
	n := 0
	for _, l := range lignes {
		if strings.Contains(l, "level="+niveau) && strings.Contains(l, fragment) {
			n++
		}
	}
	return n
}

func TestHaloTokensForDrain_RefusJournaliseUneFoisParEpisode(t *testing.T) {
	tp := capturerJournalSurveillance(t)
	src := &stubPoolSource{err: errors.New("aucun slot sain")}
	reg := &ServiceRegistry{publicReadTokenSrc: src}

	for i := 0; i < 5; i++ {
		if _, err := reg.haloTokensForDrain(context.Background()); !errors.Is(err, ErrNoPublicReadToken) {
			t.Fatalf("prise %d : %v, attendu ErrNoPublicReadToken", i, err)
		}
	}
	l := tp.lignes()
	if n := compter(l, "WARN", "jeton de lecture publique"); n != 1 {
		t.Errorf("%d avertissements pour 5 refus d'un même épisode, attendu 1 :\n%s", n, strings.Join(l, "\n"))
	}
	if n := compter(l, "DEBUG", "jeton de lecture publique"); n != 4 {
		t.Errorf("%d refus au niveau débogage, attendu 4 :\n%s", n, strings.Join(l, "\n"))
	}

	// Un compte consentant revient : l'épisode se clôt, une ligne le dit.
	src.err, src.tokens, src.gamertag = nil, freshTokens("sp-pool"), "Ouvert"
	if _, err := reg.haloTokensForDrain(context.Background()); err != nil {
		t.Fatalf("prise avec un compte consentant : %v", err)
	}
	if n := compter(tp.lignes(), "INFO", "de nouveau disponible"); n != 1 {
		t.Errorf("%d ligne(s) de retour du jeton, attendu 1", n)
	}
	// Une prise réussie de plus ne dit rien de plus.
	if _, err := reg.haloTokensForDrain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := compter(tp.lignes(), "INFO", "de nouveau disponible"); n != 1 {
		t.Errorf("%d ligne(s) de retour du jeton après deux prises, attendu 1", n)
	}

	// Nouveau refus : nouvel épisode, nouvel avertissement.
	src.err, src.tokens = errors.New("aucun slot sain"), nil
	if _, err := reg.haloTokensForDrain(context.Background()); !errors.Is(err, ErrNoPublicReadToken) {
		t.Fatal(err)
	}
	if n := compter(tp.lignes(), "WARN", "jeton de lecture publique"); n != 2 {
		t.Errorf("%d avertissements au total, attendu 2 (un par épisode)", n)
	}
}

func TestHaloTokensForDrain_SourceNonCableeJournaliseUneFois(t *testing.T) {
	tp := capturerJournalSurveillance(t)
	reg := &ServiceRegistry{}
	for i := 0; i < 3; i++ {
		_, _ = reg.haloTokensForDrain(context.Background())
	}
	if n := compter(tp.lignes(), "WARN", "jeton de lecture publique"); n != 1 {
		t.Errorf("%d avertissements pour une source non câblée, attendu 1", n)
	}
}

func TestHaloTokensForDrain_JetonVideJournaliseUneFois(t *testing.T) {
	tp := capturerJournalSurveillance(t)
	reg := &ServiceRegistry{publicReadTokenSrc: &stubPoolSource{}}
	for i := 0; i < 3; i++ {
		_, _ = reg.haloTokensForDrain(context.Background())
	}
	if n := compter(tp.lignes(), "WARN", "jeton de lecture publique"); n != 1 {
		t.Errorf("%d avertissements pour un jeton vide, attendu 1", n)
	}
}
