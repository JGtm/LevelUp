package userstore

import (
	"path/filepath"
	"testing"
)

func tempInviteStorePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "invites.json")
}

func TestGenerate_And_Validate(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))

	invite, err := s.Generate("admin", 7, "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if invite.Code == "" {
		t.Fatal("code vide")
	}
	if invite.CreatedBy != "admin" {
		t.Errorf("createdBy = %q, want admin", invite.CreatedBy)
	}
	if invite.ExpiresAt == "" {
		t.Error("expiresAt vide")
	}
	if len(invite.Code) != inviteCodeLen {
		t.Errorf("len(code) = %d, want %d", len(invite.Code), inviteCodeLen)
	}

	// Validate — doit réussir.
	if err := s.Validate(invite.Code); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestGenerate_WithGroupID(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))

	invite, err := s.Generate("alice", 7, "grp_abc")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if invite.GroupID != "grp_abc" {
		t.Errorf("GroupID = %q, want grp_abc", invite.GroupID)
	}

	// Get relit le GroupID persisté.
	got, err := s.Get(invite.Code)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.GroupID != "grp_abc" {
		t.Errorf("Get().GroupID = %q, want grp_abc", got.GroupID)
	}

	if _, err := s.Get("NOTEXIST"); err != ErrInviteNotFound {
		t.Errorf("Get(absent) = %v, want ErrInviteNotFound", err)
	}
}

func TestValidate_NotFound(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))
	err := s.Validate("NOTEXIST")
	if err != ErrInviteNotFound {
		t.Errorf("err = %v, want ErrInviteNotFound", err)
	}
}

func TestConsume_And_DoubleUse(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))
	invite, _ := s.Generate("admin", 7, "")

	if err := s.Consume(invite.Code, "bob"); err != nil {
		t.Fatalf("Consume: %v", err)
	}

	// Valider après consommation — doit échouer.
	err := s.Validate(invite.Code)
	if err != ErrInviteUsed {
		t.Errorf("validate after consume: err = %v, want ErrInviteUsed", err)
	}

	// Double consommation — doit échouer.
	err = s.Consume(invite.Code, "charlie")
	if err != ErrInviteUsed {
		t.Errorf("double consume: err = %v, want ErrInviteUsed", err)
	}
}

func TestConsume_NotFound(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))
	err := s.Consume("NOTEXIST", "user")
	if err != ErrInviteNotFound {
		t.Errorf("err = %v, want ErrInviteNotFound", err)
	}
}

func TestRevoke(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))
	invite, _ := s.Generate("admin", 7, "")

	if err := s.Revoke(invite.Code); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// Le code n'existe plus.
	err := s.Validate(invite.Code)
	if err != ErrInviteNotFound {
		t.Errorf("validate after revoke: err = %v, want ErrInviteNotFound", err)
	}

	// Revoke inexistant.
	err = s.Revoke("NOTEXIST")
	if err != ErrInviteNotFound {
		t.Errorf("revoke missing: err = %v, want ErrInviteNotFound", err)
	}
}

func TestList_Invites(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))
	_, _ = s.Generate("admin", 7, "")
	_, _ = s.Generate("admin", 14, "")

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
}

func TestGenerate_DefaultExpiry(t *testing.T) {
	s := NewInviteStore(tempInviteStorePath(t))
	// expiresInDays <= 0 → défaut 7 jours.
	invite, err := s.Generate("admin", 0, "")
	if err != nil {
		t.Fatalf("Generate(0): %v", err)
	}
	if invite.ExpiresAt == "" {
		t.Error("expiresAt vide avec défaut")
	}
}

func TestGenerateCode_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatalf("generateCode: %v", err)
		}
		if seen[code] {
			t.Fatalf("collision de code: %s", code)
		}
		seen[code] = true
	}
}

func TestGenerateCode_AlphabetOnly(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatalf("generateCode: %v", err)
		}
		for _, c := range code {
			found := false
			for _, a := range inviteAlphabet {
				if c == a {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("caractère inattendu %c dans code %s", c, code)
			}
		}
	}
}

func TestPersistence_Invites(t *testing.T) {
	path := tempInviteStorePath(t)
	s1 := NewInviteStore(path)
	invite, _ := s1.Generate("admin", 7, "")

	// Nouvel objet sur le même fichier.
	s2 := NewInviteStore(path)
	if err := s2.Validate(invite.Code); err != nil {
		t.Fatalf("persistence validate: %v", err)
	}
}
