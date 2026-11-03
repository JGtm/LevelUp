package transfer

import (
	"errors"
	"testing"
	"time"
)

func TestTokenStore_UsageUniqueEcheanceEtLimite(t *testing.T) {
	s := NewTokenStore()
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	g := Grant{Xuid: "2533274800000001", Gamertag: "SpartanB"}

	tok, exp, err := s.Issue(g, now)
	if err != nil || !exp.Equal(now.Add(TokenTTL)) || len(tok) < 32 {
		t.Fatalf("Issue : %q %v %v", tok, exp, err)
	}
	got, err := s.Redeem(tok, now.Add(time.Minute))
	if err != nil || got != g {
		t.Fatalf("Redeem : %+v %v", got, err)
	}
	if _, err := s.Redeem(tok, now.Add(2*time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("un jeton ne sert qu'une fois : %v", err)
	}

	// Limite : un export par heure et par joueur, le jeton reste alors valable.
	tok2, _, _ := s.Issue(g, now.Add(5*time.Minute))
	if _, err := s.Redeem(tok2, now.Add(6*time.Minute)); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("deuxième export dans l'heure : %v, attendu ErrRateLimited", err)
	}
	// Le jeton émis à +5 min expire à +20 min : l'heure de la limite passe après son échéance.
	if _, err := s.Redeem(tok2, now.Add(time.Hour+2*time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("jeton échu après l'heure : %v", err)
	}
	tok3, _, _ := s.Issue(g, now.Add(time.Hour+3*time.Minute))
	if _, err := s.Redeem(tok3, now.Add(time.Hour+4*time.Minute)); err != nil {
		t.Fatalf("après l'heure un jeton neuf passe : %v", err)
	}
}

func TestTokenStore_EcheanceEtJetonRemplace(t *testing.T) {
	s := NewTokenStore()
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	g := Grant{Xuid: "2533274800000001"}
	old, _, _ := s.Issue(g, now)
	fresh, _, _ := s.Issue(g, now.Add(time.Minute))
	if _, err := s.Redeem(old, now.Add(2*time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("un nouveau jeton invalide le précédent : %v", err)
	}
	if _, err := s.Redeem(fresh, now.Add(TokenTTL+2*time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("un jeton échu est refusé : %v", err)
	}
	if _, err := s.Redeem("inconnu", now); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("jeton inconnu : %v", err)
	}
}

func TestTokenStore_SessionLimiteUnParHeure(t *testing.T) {
	s := NewTokenStore()
	now := time.Now()
	if !s.AllowSession("1234567", now) || s.AllowSession("1234567", now.Add(time.Minute)) || !s.AllowSession("1234567", now.Add(61*time.Minute)) {
		t.Fatal("un export de session par heure et par joueur")
	}
}

func TestNormalizeXuid(t *testing.T) {
	for in, want := range map[string]string{"2533274800000001": "2533274800000001", "xuid(2533274800000001)": "2533274800000001"} {
		if got, err := NormalizeXuid(in); err != nil || got != want {
			t.Errorf("%q : %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "12", "abc", "1; DROP TABLE x", "25332748000000011111111"} {
		if _, err := NormalizeXuid(bad); !errors.Is(err, ErrInvalidXuid) {
			t.Errorf("%q devait être refusé : %v", bad, err)
		}
	}
}

func TestExportURL(t *testing.T) {
	ok := map[string]string{
		"https://levelup.example.com":  "https://levelup.example.com" + ExportPath,
		"https://levelup.example.com/": "https://levelup.example.com" + ExportPath,
		"http://127.0.0.1:8000":        "http://127.0.0.1:8000" + ExportPath,
		"https://example.com/app":      "https://example.com/app" + ExportPath,
	}
	for in, want := range ok {
		if got, err := exportURL(in); err != nil || got != want {
			t.Errorf("%q : %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "ftp://x", "http://example.com", "https://user:pw@example.com", "https://example.com?x=1", "pas une adresse"} {
		if _, err := exportURL(bad); !errors.Is(err, ErrBadURL) {
			t.Errorf("%q devait être refusée : %v", bad, err)
		}
	}
}
