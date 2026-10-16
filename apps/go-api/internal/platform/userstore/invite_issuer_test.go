package userstore

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
)

// stubIssuers : comptes émetteurs connus par nom (insensible à la casse, comme Store).
type stubIssuers map[string]bool

func (s stubIssuers) Get(username string) (*domain.User, error) {
	if s[slugify(username)] {
		return &domain.User{Username: username}, nil
	}
	return nil, ErrUserNotFound
}

func issuersOf(names ...string) stubIssuers {
	s := stubIssuers{}
	for _, n := range names {
		s[slugify(n)] = true
	}
	return s
}

func TestInvite_IssuerGoneInvalidatesInvite(t *testing.T) {
	issuers := issuersOf("boss")
	s := NewInviteStore(tempInviteStorePath(t), issuers)
	inv, err := s.Generate("boss", 7, "")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	delete(issuers, "boss")

	if err := s.Validate(inv.Code); !errors.Is(err, ErrInviteIssuerGone) {
		t.Errorf("Validate = %v, attendu ErrInviteIssuerGone", err)
	}
	if _, err := s.Consume(inv.Code, "newcomer"); !errors.Is(err, ErrInviteIssuerGone) {
		t.Errorf("Consume = %v, attendu ErrInviteIssuerGone", err)
	}
	list, _ := s.List()
	if len(list) != 1 || list[0].Valid {
		t.Errorf("List : l'invitation d'un émetteur disparu doit être marquée invalide : %+v", list)
	}
}

func TestInvite_NilIssuersMeansNoValidInvite(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t), nil)
	inv, _ := s.Generate("admin", 7, "")
	if err := s.Validate(inv.Code); !errors.Is(err, ErrInviteIssuerGone) {
		t.Errorf("Validate sans annuaire = %v, attendu ErrInviteIssuerGone", err)
	}
}

func TestInvite_ConsumeIsSingleUseUnderConcurrency(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t), issuersOf("admin"))
	inv, _ := s.Generate("admin", 7, "")
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Consume(inv.Code, "user"+string(rune('a'+i))); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("consommations réussies = %d, attendu 1", wins.Load())
	}
}

func TestInvite_ReleaseRestoresOnlyForSameUser(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t), issuersOf("admin"))
	inv, _ := s.Generate("admin", 7, "")
	if _, err := s.Consume(inv.Code, "bob"); err != nil {
		t.Fatalf("Consume : %v", err)
	}
	if err := s.Release(inv.Code, "mallory"); err != nil {
		t.Fatalf("Release (autre) : %v", err)
	}
	if err := s.Validate(inv.Code); !errors.Is(err, ErrInviteUsed) {
		t.Errorf("un autre que le consommateur a rendu le code : %v", err)
	}
	if err := s.Release(inv.Code, "bob"); err != nil {
		t.Fatalf("Release : %v", err)
	}
	if err := s.Validate(inv.Code); err != nil {
		t.Errorf("code non rendu : %v", err)
	}
}
