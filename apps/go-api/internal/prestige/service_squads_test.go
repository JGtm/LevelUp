package prestige

import (
	"context"
	"errors"
	"testing"
)

func TestService_CreateSquad_CreatesAndAddsMembers(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sq, err := svc.CreateSquad(context.Background(), CreateSquadRequest{
		Name:      "Trio",
		CreatedBy: "alice",
		Members: []SquadMember{
			{Xuid: "x1", UserID: "alice"},
			{Xuid: "x2", UserID: "bob"},
			{Xuid: "x3"}, // ami hors-app (pas de user_id)
		},
	})
	if err != nil {
		t.Fatalf("CreateSquad: %v", err)
	}
	if sq.ID == "" || sq.Name != "Trio" || sq.CreatedBy != "alice" {
		t.Errorf("squad mal formé: %+v", sq)
	}
	if len(sqRepo.created) != 1 {
		t.Errorf("created=%d, want 1", len(sqRepo.created))
	}
	if len(sqRepo.added) != 3 {
		t.Fatalf("added=%d, want 3", len(sqRepo.added))
	}
	for _, m := range sqRepo.added {
		if m.SquadID != sq.ID {
			t.Errorf("membre %q squadID=%q, want %q", m.Xuid, m.SquadID, sq.ID)
		}
	}
}

func TestService_CreateSquad_SkipsMembersWithoutXUID(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	_, err := svc.CreateSquad(context.Background(), CreateSquadRequest{
		Name:      "Duo",
		CreatedBy: "alice",
		Members: []SquadMember{
			{Xuid: "x1", UserID: "alice"},
			{Xuid: "", UserID: "ghost"}, // clé invalide → ignoré
		},
	})
	if err != nil {
		t.Fatalf("CreateSquad: %v", err)
	}
	if len(sqRepo.added) != 1 {
		t.Errorf("added=%d, want 1 (membre sans xuid ignoré)", len(sqRepo.added))
	}
}

func TestService_CreateSquad_RequiresNameAndCreator(t *testing.T) {
	svc, _, _, _, _, _ := buildFullService()
	if _, err := svc.CreateSquad(context.Background(), CreateSquadRequest{Name: "", CreatedBy: "alice"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("name vide: want ErrInvalidInput, got %v", err)
	}
	if _, err := svc.CreateSquad(context.Background(), CreateSquadRequest{Name: "X", CreatedBy: ""}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("created_by vide: want ErrInvalidInput, got %v", err)
	}
}

func TestService_AddSquadMember_RequiresMemberUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}} // alice = membre-user

	// outsider (non membre-user) rejeté
	if err := svc.AddSquadMember(context.Background(), "sq1", SquadMember{Xuid: "x9"}, "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("outsider doit être rejeté, got %v", err)
	}
	if len(sqRepo.added) != 0 {
		t.Errorf("aucun ajout attendu après rejet, got %d", len(sqRepo.added))
	}

	// alice (membre-user) autorisée
	if err := svc.AddSquadMember(context.Background(), "sq1", SquadMember{Xuid: "x9", UserID: "carol"}, "alice"); err != nil {
		t.Errorf("membre-user doit pouvoir ajouter, got %v", err)
	}
	if len(sqRepo.added) != 1 || sqRepo.added[0].Xuid != "x9" || sqRepo.added[0].SquadID != "sq1" {
		t.Errorf("ajout inattendu: %+v", sqRepo.added)
	}
}

func TestService_AddSquadMember_RequiresXUID(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}}
	if err := svc.AddSquadMember(context.Background(), "sq1", SquadMember{Xuid: ""}, "alice"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("xuid vide: want ErrInvalidInput, got %v", err)
	}
}

func TestService_RemoveSquadMember_RequiresMemberUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.members = []SquadMember{{Xuid: "x1", UserID: "alice"}}

	if err := svc.RemoveSquadMember(context.Background(), "sq1", "x1", "outsider"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("outsider doit être rejeté, got %v", err)
	}
	if err := svc.RemoveSquadMember(context.Background(), "sq1", "x1", "alice"); err != nil {
		t.Errorf("membre-user doit pouvoir retirer, got %v", err)
	}
	if len(sqRepo.removed) != 1 || sqRepo.removed[0] != "x1" {
		t.Errorf("retrait inattendu: %+v", sqRepo.removed)
	}
}

func TestService_ListSquadsForUser(t *testing.T) {
	svc, _, _, _, sqRepo, _ := buildFullService()
	sqRepo.squadsByUser = []Squad{{ID: "sq1", Name: "Trio"}}

	got, err := svc.ListSquadsForUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListSquadsForUser: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sq1" {
		t.Errorf("got=%+v, want [sq1]", got)
	}
	if _, err := svc.ListSquadsForUser(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("user vide: want ErrInvalidInput, got %v", err)
	}
}
