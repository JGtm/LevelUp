// Tests pour friends_extras_loader.go — résolution des extras per-friend
// (perf score + skill rank) chargés depuis la player DB de chaque ami pour
// le panneau d'expander du scoreboard Match View.
//
// Stratégie : mock du FriendMatchExtrasRepo (interface) + opener spy pour
// vérifier que le bon (titleSlug, gamertag) est appelé par xuid.
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ---------------------------------------------------------------------------
// Mock FriendMatchExtrasRepo
// ---------------------------------------------------------------------------

type mockFriendRepo struct {
	enrichResult *domain.MatchEnrichmentRaw
	enrichErr    error
	rankResult   *domain.SkillRankRaw
	rankErr      error
	modelResult  *domain.PlayerAssistsModel
	modelErr     error
	enrichCalls  int
	rankCalls    int
}

func (m *mockFriendRepo) GetMatchEnrichment(_ context.Context, _ string) (*domain.MatchEnrichmentRaw, error) {
	m.enrichCalls++
	return m.enrichResult, m.enrichErr
}

func (m *mockFriendRepo) GetMatchSkillRank(_ context.Context, _ string) (*domain.SkillRankRaw, error) {
	m.rankCalls++
	return m.rankResult, m.rankErr
}

func (m *mockFriendRepo) GetPlayerAssistsModel(_ context.Context, _ string) (*domain.PlayerAssistsModel, error) {
	return m.modelResult, m.modelErr
}

// ---------------------------------------------------------------------------
// Tests loadOneFriendExtras
// ---------------------------------------------------------------------------

func TestLoadOneFriendExtras_PopulatesPerformanceScore(t *testing.T) {
	t.Parallel()
	score := 75.5
	repo := &mockFriendRepo{
		enrichResult: &domain.MatchEnrichmentRaw{PerformanceScore: &score},
	}
	res, _ := loadOneFriendExtras(context.Background(), repo, "match-1", "", "xuid-1", nil)
	if res == nil {
		t.Fatal("expected non-nil extras when enrichment present")
	}
	if res.PerformanceScore == nil || *res.PerformanceScore != 75.5 {
		t.Errorf("PerformanceScore want 75.5, got %v", res.PerformanceScore)
	}
	if res.SkillRank != nil {
		t.Errorf("SkillRank should be nil, got %+v", res.SkillRank)
	}
}

func TestLoadOneFriendExtras_PopulatesSkillRank(t *testing.T) {
	t.Parallel()
	tier := "Onyx 1500"
	val := 1500.0
	delta := 25.0
	repo := &mockFriendRepo{
		rankResult: &domain.SkillRankRaw{
			RatingType:  "CSR",
			TierLabel:   &tier,
			RatingValue: &val,
			RatingDelta: &delta,
		},
	}
	res, _ := loadOneFriendExtras(context.Background(), repo, "match-1", "", "xuid-1", nil)
	if res == nil {
		t.Fatal("expected non-nil extras when skill_rank present")
	}
	if res.SkillRank == nil {
		t.Fatal("expected SkillRank populated")
	}
	if res.SkillRank.RatingType != "CSR" {
		t.Errorf("RatingType want CSR, got %q", res.SkillRank.RatingType)
	}
	if res.SkillRank.TierLabel == nil || *res.SkillRank.TierLabel != "Onyx 1500" {
		t.Errorf("TierLabel want Onyx 1500, got %v", res.SkillRank.TierLabel)
	}
}

func TestLoadOneFriendExtras_BothSourcesEmpty_ReturnsNil(t *testing.T) {
	t.Parallel()
	// Enrichissement présent mais sans PerformanceScore + skill rank absent
	// → aucune donnée à afficher → nil pour skip la section côté front.
	repo := &mockFriendRepo{
		enrichResult: &domain.MatchEnrichmentRaw{}, // PerformanceScore nil
		rankResult:   nil,
	}
	res, _ := loadOneFriendExtras(context.Background(), repo, "match-1", "", "xuid-1", nil)
	if res != nil {
		t.Errorf("expected nil when no usable data, got %+v", res)
	}
}

func TestLoadOneFriendExtras_EnrichErrorDoesntBlockRank(t *testing.T) {
	t.Parallel()
	// L'enrichissement échoue (DB locked, table absente) — on doit quand
	// même tenter de lire le skill rank et retourner ce qu'on a.
	tier := "Diamond 3"
	repo := &mockFriendRepo{
		enrichErr:  errors.New("db locked"),
		rankResult: &domain.SkillRankRaw{RatingType: "CSR", TierLabel: &tier},
	}
	res, err := loadOneFriendExtras(context.Background(), repo, "match-1", "", "xuid-1", nil)
	if err == nil || !errors.Is(err, repo.enrichErr) {
		t.Errorf("la panne de l'enrichissement doit REMONTER avec les données lues, err = %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil extras (skill rank present)")
	}
	if res.SkillRank == nil || res.SkillRank.RatingType != "CSR" {
		t.Errorf("SkillRank should be populated despite enrich error, got %+v", res.SkillRank)
	}
	if res.PerformanceScore != nil {
		t.Errorf("PerformanceScore should be nil after enrich error, got %v", res.PerformanceScore)
	}
}

func TestLoadOneFriendExtras_BothErrors_ReturnsNil(t *testing.T) {
	t.Parallel()
	repo := &mockFriendRepo{
		enrichErr: errors.New("enrich db locked"),
		rankErr:   errors.New("rank db locked"),
	}
	res, err := loadOneFriendExtras(context.Background(), repo, "match-1", "", "xuid-1", nil)
	if !errors.Is(err, repo.enrichErr) || !errors.Is(err, repo.rankErr) {
		t.Errorf("les deux pannes doivent remonter, err = %v", err)
	}
	if res != nil {
		t.Errorf("expected nil when both sources error, got %+v", res)
	}
}

// ---------------------------------------------------------------------------
// Tests NewFriendsExtrasResolver
// ---------------------------------------------------------------------------

func TestNewFriendsExtrasResolver_LookupsFriendByXUID(t *testing.T) {
	t.Parallel()
	score := 60.0
	mockRepo := &mockFriendRepo{
		enrichResult: &domain.MatchEnrichmentRaw{PerformanceScore: &score},
	}
	openerCalls := 0
	opener := func(_ context.Context, titleSlug, gamertag string) (FriendMatchExtrasRepo, error) {
		openerCalls++
		if titleSlug != "halo_infinite" {
			t.Errorf("opener titleSlug want halo_infinite, got %q", titleSlug)
		}
		if gamertag != "Friend1" {
			t.Errorf("opener gamertag want Friend1, got %q", gamertag)
		}
		return mockRepo, nil
	}
	friends := map[string]FriendProfile{
		"friend-xuid-1": {XUID: "friend-xuid-1", Gamertag: "Friend1", TitleSlug: "halo_infinite"},
	}
	resolver := NewFriendsExtrasResolver(friends, opener, nil)

	out, _ := resolver(context.Background(), "match-1", "", []string{"friend-xuid-1"})

	if openerCalls != 1 {
		t.Errorf("opener calls want 1, got %d", openerCalls)
	}
	extras, ok := out["friend-xuid-1"]
	if !ok {
		t.Fatal("friend-xuid-1 missing from result")
	}
	if extras.PerformanceScore == nil || *extras.PerformanceScore != 60.0 {
		t.Errorf("PerformanceScore want 60.0, got %v", extras.PerformanceScore)
	}
}

func TestNewFriendsExtrasResolver_SkipsUnknownXUIDs(t *testing.T) {
	t.Parallel()
	openerCalls := 0
	opener := func(_ context.Context, _, _ string) (FriendMatchExtrasRepo, error) {
		openerCalls++
		return &mockFriendRepo{}, nil
	}
	friends := map[string]FriendProfile{
		"known-xuid": {XUID: "known-xuid", Gamertag: "Known", TitleSlug: "halo_infinite"},
	}
	resolver := NewFriendsExtrasResolver(friends, opener, nil)

	// Demande un xuid non configuré dans friends → opener jamais appelé.
	out, _ := resolver(context.Background(), "match-1", "", []string{"unknown-xuid", "another-unknown"})

	if openerCalls != 0 {
		t.Errorf("opener should not be called for unknown xuids, got %d calls", openerCalls)
	}
	if len(out) != 0 {
		t.Errorf("result should be empty for unknown xuids, got %+v", out)
	}
}

// Une player DB d'ami qui ne s'ouvre pas est une PANNE : elle remonte (le service la dit
// `section_unavailable`), elle ne passe pas pour un ami sans donnée.
func TestNewFriendsExtrasResolver_OpenerErrorRemonte(t *testing.T) {
	t.Parallel()
	opener := func(_ context.Context, _, _ string) (FriendMatchExtrasRepo, error) {
		return nil, errors.New("db corrupted")
	}
	friends := map[string]FriendProfile{
		"friend-1": {XUID: "friend-1", Gamertag: "Friend1", TitleSlug: "halo_infinite"},
	}
	resolver := NewFriendsExtrasResolver(friends, opener, nil)

	out, err := resolver(context.Background(), "match-1", "", []string{"friend-1"})

	if err == nil {
		t.Fatal("l'échec d'ouverture de la player DB d'un ami doit remonter en erreur")
	}
	if len(out) != 0 {
		t.Errorf("expected empty result, got %+v", out)
	}
}

// Un ami sans ligne sur ce match (absence : le repo rend nil, nil) n'est PAS une panne.
func TestNewFriendsExtrasResolver_AbsenceNEstPasUnePanne(t *testing.T) {
	t.Parallel()
	opener := func(_ context.Context, _, _ string) (FriendMatchExtrasRepo, error) {
		return &mockFriendRepo{enrichResult: &domain.MatchEnrichmentRaw{}}, nil
	}
	friends := map[string]FriendProfile{
		"friend-1": {XUID: "friend-1", Gamertag: "Friend1", TitleSlug: "halo_infinite"},
	}
	out, err := NewFriendsExtrasResolver(friends, opener, nil)(context.Background(), "match-1", "mode", []string{"friend-1"})
	if err != nil || len(out) != 0 {
		t.Fatalf("absence : attendu (vide, nil), obtenu (%+v, %v)", out, err)
	}
}

func TestNewFriendsExtrasResolver_PartialResults(t *testing.T) {
	t.Parallel()
	score := 45.0
	mockOK := &mockFriendRepo{enrichResult: &domain.MatchEnrichmentRaw{PerformanceScore: &score}}
	opener := func(_ context.Context, _, gamertag string) (FriendMatchExtrasRepo, error) {
		if gamertag == "Friend1" {
			return mockOK, nil
		}
		return nil, errors.New("not opened")
	}
	friends := map[string]FriendProfile{
		"xuid-1": {XUID: "xuid-1", Gamertag: "Friend1", TitleSlug: "halo_infinite"},
		"xuid-2": {XUID: "xuid-2", Gamertag: "Friend2", TitleSlug: "halo_infinite"},
	}
	resolver := NewFriendsExtrasResolver(friends, opener, nil)

	out, err := resolver(context.Background(), "match-1", "", []string{"xuid-1", "xuid-2"})

	if err == nil {
		t.Errorf("l'ouverture ratée de xuid-2 doit remonter, même avec un résultat partiel")
	}
	if len(out) != 1 {
		t.Errorf("want 1 result (Friend1 only), got %d", len(out))
	}
	if _, ok := out["xuid-1"]; !ok {
		t.Errorf("xuid-1 should be present (opener succeeded)")
	}
	if _, ok := out["xuid-2"]; ok {
		t.Errorf("xuid-2 should be absent (opener errored)")
	}
}

// Garantit que le type port.FriendsExtrasResolver est bien satisfait.
var _ port.FriendsExtrasResolver = NewFriendsExtrasResolver(nil, nil, nil)

// ---------------------------------------------------------------------------
// Ami sans player DB : une absence, pas une panne
// ---------------------------------------------------------------------------

// E5.9 : un profil suivi jamais synchronisé n'a pas de player DB (l'ouverture par le pool échoue
// quand son dossier manque). L'opener le dit par ErrFriendWithoutPlayerDB : le résolveur ne rend
// aucune erreur (donc aucune raison `section_unavailable`), ne journalise aucun ERROR, et lit
// quand même les autres amis.
func TestFriendsExtrasResolver_AmiSansPlayerDB_EstUneAbsence(t *testing.T) {
	logs := logsNiveau(t)
	score := 61.0
	lu := &mockFriendRepo{enrichResult: &domain.MatchEnrichmentRaw{PerformanceScore: &score}}
	opener := func(_ context.Context, _, gamertag string) (FriendMatchExtrasRepo, error) {
		if gamertag == "Jamais" {
			return nil, fmt.Errorf("ouverture de %s: %w", gamertag, ErrFriendWithoutPlayerDB)
		}
		return lu, nil
	}
	resolve := NewFriendsExtrasResolver(map[string]FriendProfile{
		"x-jamais": {XUID: "x-jamais", Gamertag: "Jamais", TitleSlug: "halo_infinite"},
		"x-sync":   {XUID: "x-sync", Gamertag: "Synchro", TitleSlug: "halo_infinite"},
	}, opener, nil)

	extras, err := resolve(context.Background(), "m1", "", []string{"x-jamais", "x-sync"})
	if err != nil {
		t.Fatalf("un ami sans player DB ne doit pas produire d'erreur, obtenu : %v", err)
	}
	if _, ok := extras["x-jamais"]; ok {
		t.Error("aucun extra attendu pour l'ami sans player DB")
	}
	if got := extras["x-sync"]; got.PerformanceScore == nil || *got.PerformanceScore != 61.0 {
		t.Errorf("l'ami synchronisé doit rester lu, obtenu %+v", got)
	}
	assertAucuneErreurJournalisee(t, logs)

	// Contrôle : toute autre panne d'ouverture reste une panne remontée.
	panne := NewFriendsExtrasResolver(map[string]FriendProfile{
		"x-jamais": {XUID: "x-jamais", Gamertag: "Jamais", TitleSlug: "halo_infinite"},
	}, func(context.Context, string, string) (FriendMatchExtrasRepo, error) {
		return nil, errors.New("database is locked")
	}, nil)
	if _, err := panne(context.Background(), "m1", "", []string{"x-jamais"}); err == nil {
		t.Error("une ouverture en échec autre que l'absence doit rester une panne remontée")
	}
}
