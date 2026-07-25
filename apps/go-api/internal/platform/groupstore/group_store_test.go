package groupstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

func newTestStore(t *testing.T) *GroupStore {
	t.Helper()
	return NewGroupStore(filepath.Join(t.TempDir(), "groups.json"))
}

func TestCreateAndGet(t *testing.T) {
	s := newTestStore(t)

	g, err := s.Create("Mon foyer", "xuid-owner", "Owner")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.ID == "" || g.Name != "Mon foyer" || g.OwnerXUID != "xuid-owner" {
		t.Fatalf("groupe inattendu: %+v", g)
	}
	if len(g.Members) != 1 || g.Members[0].Role != domain.GroupRoleOwner {
		t.Fatalf("le propriétaire doit être l'unique membre owner: %+v", g.Members)
	}

	got, err := s.Get(g.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != g.ID {
		t.Fatalf("Get a retourné un autre groupe: %s", got.ID)
	}
}

func TestCreateValidation(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Create("", "xuid", "GT"); err != ErrInvalidName {
		t.Fatalf("nom vide doit donner ErrInvalidName, got %v", err)
	}
	if _, err := s.Create("Foyer", "", "GT"); err != ErrMissingOwnerXUID {
		t.Fatalf("owner vide doit donner ErrMissingOwnerXUID, got %v", err)
	}
}

func TestAddRemoveMember(t *testing.T) {
	s := newTestStore(t)
	g, _ := s.Create("Foyer", "owner", "Owner")

	if err := s.AddMember(g.ID, "xuid-b", "Bob"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	// Idempotent : ré-ajout sans erreur, pas de doublon.
	if err := s.AddMember(g.ID, "xuid-b", "Bob"); err != nil {
		t.Fatalf("AddMember idempotent: %v", err)
	}
	got, _ := s.Get(g.ID)
	if len(got.Members) != 2 {
		t.Fatalf("attendu 2 membres, got %d", len(got.Members))
	}

	// Ré-ajout avec nouveau gamertag → mise à jour.
	if err := s.AddMember(g.ID, "xuid-b", "Bob2"); err != nil {
		t.Fatalf("AddMember rename: %v", err)
	}
	got, _ = s.Get(g.ID)
	if !got.HasMember("xuid-b") {
		t.Fatal("xuid-b doit rester membre")
	}

	// Retrait du propriétaire interdit.
	if err := s.RemoveMember(g.ID, "owner"); err != ErrCannotRemoveOwner {
		t.Fatalf("retrait owner doit échouer, got %v", err)
	}

	// Retrait d'un membre normal.
	if err := s.RemoveMember(g.ID, "xuid-b"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	got, _ = s.Get(g.ID)
	if got.HasMember("xuid-b") || len(got.Members) != 1 {
		t.Fatalf("xuid-b aurait dû être retiré: %+v", got.Members)
	}

	// Retrait d'un non-membre = no-op.
	if err := s.RemoveMember(g.ID, "inconnu"); err != nil {
		t.Fatalf("retrait non-membre doit être no-op, got %v", err)
	}
}

func TestListForXUIDAndCoMembers(t *testing.T) {
	s := newTestStore(t)
	g1, _ := s.Create("Famille", "alice", "Alice")
	_ = s.AddMember(g1.ID, "bob", "Bob")
	g2, _ := s.Create("Amis", "alice", "Alice")
	_ = s.AddMember(g2.ID, "carol", "Carol")
	// Groupe sans alice.
	g3, _ := s.Create("Externe", "dave", "Dave")
	_ = s.AddMember(g3.ID, "erin", "Erin")

	groups, err := s.ListForXUID("alice")
	if err != nil {
		t.Fatalf("ListForXUID: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("alice doit être dans 2 groupes, got %d", len(groups))
	}

	co, err := s.CoMemberXUIDs("alice")
	if err != nil {
		t.Fatalf("CoMemberXUIDs: %v", err)
	}
	for _, want := range []string{"alice", "bob", "carol"} {
		if !co[want] {
			t.Fatalf("co-membre %q manquant: %+v", want, co)
		}
	}
	if co["dave"] || co["erin"] {
		t.Fatalf("dave/erin ne doivent PAS être co-membres d'alice: %+v", co)
	}

	// User sans groupe → nil (authz retombe sur owner-only strict).
	if co, _ := s.CoMemberXUIDs("inconnu"); co != nil {
		t.Fatalf("user sans groupe doit donner nil, got %+v", co)
	}
}

func TestRenameAndDelete(t *testing.T) {
	s := newTestStore(t)
	g, _ := s.Create("Ancien", "owner", "Owner")

	if err := s.Rename(g.ID, "Nouveau"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	got, _ := s.Get(g.ID)
	if got.Name != "Nouveau" {
		t.Fatalf("rename non appliqué: %q", got.Name)
	}
	if err := s.Rename(g.ID, ""); err != ErrInvalidName {
		t.Fatalf("rename vide doit échouer, got %v", err)
	}

	if err := s.Delete(g.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(g.ID); err != ErrGroupNotFound {
		t.Fatalf("groupe supprimé doit être introuvable, got %v", err)
	}
	if err := s.Delete("inconnu"); err != ErrGroupNotFound {
		t.Fatalf("Delete inconnu doit donner ErrGroupNotFound, got %v", err)
	}
}

func TestMigrateDefault(t *testing.T) {
	s := newTestStore(t)

	members := []domain.GroupMember{
		{XUID: "bob", Gamertag: "Bob"},
		{XUID: "carol", Gamertag: "Carol"},
		{XUID: "", Gamertag: "SansXUID"},   // ignoré
		{XUID: "owner", Gamertag: "Owner"}, // déjà owner, ignoré
	}
	created, err := s.MigrateDefault("Mon foyer", "owner", "Owner", members)
	if err != nil {
		t.Fatalf("MigrateDefault: %v", err)
	}
	if !created {
		t.Fatal("première migration doit créer le groupe")
	}

	groups, _ := s.List()
	if len(groups) != 1 {
		t.Fatalf("attendu 1 groupe, got %d", len(groups))
	}
	g := groups[0]
	if g.Name != "Mon foyer" || !g.IsOwner("owner") {
		t.Fatalf("groupe par défaut inattendu: %+v", g)
	}
	// owner + bob + carol = 3 (SansXUID et doublon owner ignorés).
	if len(g.Members) != 3 {
		t.Fatalf("attendu 3 membres, got %d: %+v", len(g.Members), g.Members)
	}

	// Idempotent : deuxième appel ne recrée rien.
	created2, err := s.MigrateDefault("Autre", "owner", "Owner", nil)
	if err != nil {
		t.Fatalf("MigrateDefault 2: %v", err)
	}
	if created2 {
		t.Fatal("migration idempotente : ne doit pas recréer")
	}
	if groups, _ := s.List(); len(groups) != 1 {
		t.Fatalf("toujours 1 groupe attendu, got %d", len(groups))
	}
}

func TestMigrateDefaultRequiresOwner(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.MigrateDefault("Foyer", "", "", nil); err != ErrMissingOwnerXUID {
		t.Fatalf("owner vide doit donner ErrMissingOwnerXUID, got %v", err)
	}
}

// TestMembersNeverNilOnRead — invariant de contrat (V72-01 / H7) : `Group.members` est
// non nullable dans l'OpenAPI généré. Un fichier legacy sans `members` (ou avec `null`)
// doit être normalisé en slice VIDE à la lecture, sinon encoding/json émet `null` et le
// front reçoit un tableau absent là où le contrat promet un tableau.
func TestMembersNeverNilOnRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups.json")
	legacy := `{"version":"1.0","groups":{"grp_legacy":{"id":"grp_legacy","name":"Legacy",` +
		`"owner_xuid":"xuid-owner","members":null,"created_at":"2026-01-01T00:00:00Z",` +
		`"updated_at":"2026-01-01T00:00:00Z"}}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("écriture fixture: %v", err)
	}
	s := NewGroupStore(path)

	g, err := s.Get("grp_legacy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g.Members == nil {
		t.Fatal("Get: members nil — la normalisation nil → [] n'a pas eu lieu")
	}
	blob, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(blob), `"members":[]`) {
		t.Fatalf("sérialisation attendue avec \"members\":[], got %s", blob)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 || all[0].Members == nil {
		t.Fatalf("List: members nil — normalisation manquante: %+v", all)
	}
}
