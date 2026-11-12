package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/port"
)

func TestBuildSyncScope_AllData(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{AllData: true, MaxMatches: 50})
	if !scope.AllData {
		t.Fatal("expected AllData")
	}
	if !scope.Medals || !scope.Events || !scope.Skill {
		t.Fatal("expected all types enabled")
	}
	if scope.MaxMatches != 50 {
		t.Fatalf("expected 50, got %d", scope.MaxMatches)
	}
}

func TestBuildSyncScope_NoFlags(t *testing.T) {
	if scope := buildSyncScope(domain.BackfillStartRequest{}); !scope.AllData {
		t.Fatal("expected AllData when no flags set")
	}
}

func TestBuildSyncScope_PartialFlags(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{Medals: true, Events: true})
	if scope.AllData {
		t.Fatal("expected partial, not AllData")
	}
	if !scope.Medals || !scope.Events {
		t.Fatal("expected medals+events")
	}
	if scope.Skill {
		t.Fatal("expected skill false")
	}
}

func TestBuildSyncScope_ForceRescan(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{Medals: true, ForceRescan: true})
	if !scope.ForceMedals {
		t.Fatal("expected ForceMedals")
	}
	if scope.ForceSkill {
		t.Fatal("expected ForceSkill false (skill not requested)")
	}
}

func TestBuildSyncScope_ForceRescanAllData(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{AllData: true, ForceRescan: true})
	if !scope.ForceMedals || !scope.ForceSkill || !scope.ForceEvents {
		t.Fatal("expected all force flags when AllData+ForceRescan")
	}
	if !scope.ForceAliases || !scope.ForceLUSR || !scope.ForcePerformanceScores {
		t.Fatal("expected ForceAliases/ForceLUSR/ForcePerformanceScores when AllData+ForceRescan")
	}
	if !scope.ForceEvents || !scope.ForcePersonalScores {
		t.Fatal("expected ForceEvents/ForcePersonalScores when AllData+ForceRescan")
	}
}

func TestBuildSyncScope_ForceRescanAliasesAndLUSR(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{Aliases: true, LUSR: true, ForceRescan: true})
	if !scope.ForceAliases {
		t.Fatal("expected ForceAliases when Aliases+ForceRescan")
	}
	if !scope.ForceLUSR {
		t.Fatal("expected ForceLUSR when LUSR+ForceRescan")
	}
	if scope.ForceMedals || scope.ForceSkill {
		t.Fatal("expected non-requested force flags to remain false")
	}
}

func TestBuildSyncScope_ForceRescanEventsAndPersonalScores(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{Events: true, PersonalScores: true, ForceRescan: true})
	if !scope.ForceEvents {
		t.Fatal("expected ForceEvents when Events+ForceRescan")
	}
	if !scope.ForcePersonalScores {
		t.Fatal("expected ForcePersonalScores when PersonalScores+ForceRescan")
	}
	if scope.ForceMedals || scope.ForceSkill {
		t.Fatal("expected non-requested force flags to remain false")
	}
}

// Coef-only : seul le flag EngagementCoefficients est posé, le reste reste off. AllData ne doit PAS
// s'activer (sinon l'utilisateur lancerait un backfill complet par surprise).
func TestBuildSyncScope_EngagementCoefficientsOnly(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{EngagementCoefficients: true})
	if scope.AllData {
		t.Fatal("AllData should NOT activate when only EngagementCoefficients is set")
	}
	if !scope.EngagementCoefficients {
		t.Fatal("EngagementCoefficients should be true")
	}
	if scope.EngagementScores {
		t.Fatal("EngagementScores should remain false (coef-only mode)")
	}
	if scope.Medals || scope.Events || scope.Skill {
		t.Fatal("Other flags should remain false in coef-only mode")
	}
}

// Quand EngagementScores=true, le recalcul des coefficients est implicite (en queue de
// RunBackfillEngagementScores) : le flag explicite reste à false pour éviter le double passage.
func TestBuildSyncScope_EngagementScoresImpliesCoefRecompute(t *testing.T) {
	scope := buildSyncScope(domain.BackfillStartRequest{EngagementScores: true})
	if !scope.EngagementScores {
		t.Fatal("EngagementScores should be true")
	}
	if scope.EngagementCoefficients {
		t.Fatal("EngagementCoefficients should remain false (recompute is implicit)")
	}
}

// Refus du rattrapage, sur des profils factices : aucun job n'est créé.
func TestStartBackfill_Refus(t *testing.T) {
	cases := []struct {
		name    string
		req     domain.BackfillStartRequest
		arrange func(f *syncJobsFixture)
		kind    port.SyncJobErrorKind
	}{
		{"joueur absent du titre demandé", domain.BackfillStartRequest{PlayerSlug: "jgtm", Title: "halo_5"}, nil,
			port.SyncErrPlayerNotFound},
		{"titre inconnu", domain.BackfillStartRequest{PlayerSlug: "jgtm", Title: "halo_9"}, nil,
			port.SyncErrPlayerNotFound},
		{"joueur suivi sur Halo 5 seulement, sans titre", domain.BackfillStartRequest{PlayerSlug: "h5only"}, nil,
			port.SyncErrPlayerNotFound},
		{"rattrapage déjà actif", domain.BackfillStartRequest{PlayerSlug: "jgtm"},
			func(f *syncJobsFixture) { f.jobs.Create(domain.JobTypeBackfill, "jgtm") },
			port.SyncErrBackfillAlreadyActive},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSyncJobsFixture(t)
			if c.arrange != nil {
				c.arrange(f)
			}
			before := f.jobs.count()
			_, err := f.svc.StartBackfill(context.Background(), sessionCaller(), c.req)
			requireSyncJobError(t, err, c.kind)
			if f.jobs.count() != before {
				t.Error("un refus ne doit créer aucun job")
			}
		})
	}
}

// Un titre absent vaut le titre par défaut AVANT la recherche du joueur : la recherche porte sur les
// profils de ce titre, jamais sur tous (un joueur suivi sur Halo 5 seulement est refusé).
func TestStartBackfill_TitreParDefautAvantLaRecherche(t *testing.T) {
	f := newSyncJobsFixture(t)
	_, _ = f.svc.StartBackfill(context.Background(), sessionCaller(), domain.BackfillStartRequest{PlayerSlug: "h5only"})
	if got := f.profiles.filters; len(got) != 1 || len(got[0]) != 1 || got[0][0] != titlePkg.DefaultSlug {
		t.Errorf("filtres de titre %v, attendu [[%s]]", got, titlePkg.DefaultSlug)
	}
}

// backfillTitleFixture : un db_profiles v3 où JGtm est suivi sur les deux titres et Solo sur Infinite
// seulement, dans une racine de dépôt temporaire ; le service lit la vraie configuration.
func backfillTitleFixture(t *testing.T) (*SyncJobService, *jobs.Store, string) {
	t.Helper()
	root := t.TempDir()
	profiles := `{"version":"3.0","admin":"JGtm","profiles":{
		"halo_infinite":{"JGtm":{"xuid":"1001"},"Solo":{"xuid":"1002"}},
		"halo_5":{"JGtm":{"xuid":"1001"},"H5Seul":{"xuid":"1003"}}}}`
	profilesPath := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	return NewSyncJobService(cfg, nil, store, nil), store, root
}

func startBackfillDryRun(t *testing.T, s *SyncJobService, store *jobs.Store, slug, title string) *domain.AsyncJobStatus {
	t.Helper()
	job, err := s.StartBackfill(context.Background(), sessionCaller(),
		domain.BackfillStartRequest{PlayerSlug: slug, Title: title, DryRun: true})
	if err != nil {
		t.Fatalf("StartBackfill: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if j := store.Get(job.JobID); j != nil && j.IsTerminal() {
			return job
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("le job %s n'est pas terminé", job.JobID)
	return nil
}

// Sous title=halo_5 le moteur travaille sur les bases de Halo 5 (la base joueur du titre apparaît),
// jamais sur celles du titre par défaut ; sans title, le titre par défaut.
func TestStartBackfill_TitreRoute(t *testing.T) {
	for _, c := range []struct {
		name, title, vers, pasVers string
	}{
		{"halo_5", "halo_5", "halo_5", titlePkg.DefaultSlug},
		{"sans titre", "", titlePkg.DefaultSlug, "halo_5"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, store, root := backfillTitleFixture(t)
			startBackfillDryRun(t, s, store, "JGtm", c.title)
			pr := titlePkg.NewPathResolver(root)
			if _, err := os.Stat(pr.PlayerDBPath(c.vers, "JGtm")); err != nil {
				t.Errorf("la base joueur de %s devait être ouverte par le moteur: %v", c.vers, err)
			}
			if _, err := os.Stat(pr.PlayerDBPath(c.pasVers, "JGtm")); !os.IsNotExist(err) {
				t.Errorf("la base joueur de %s ne devait pas être touchée (stat: %v)", c.pasVers, err)
			}
		})
	}
}

// Un joueur suivi sur Halo 5 seulement est refusé sans titre, sans qu'aucune base ne soit créée ; avec son
// titre il est accepté.
func TestStartBackfill_SansTitreRefuseUnJoueurSuiviSurHalo5Seulement(t *testing.T) {
	s, store, root := backfillTitleFixture(t)
	_, err := s.StartBackfill(context.Background(), sessionCaller(),
		domain.BackfillStartRequest{PlayerSlug: "H5Seul", DryRun: true})
	requireSyncJobError(t, err, port.SyncErrPlayerNotFound)
	pr := titlePkg.NewPathResolver(root)
	for _, slug := range []string{titlePkg.DefaultSlug, "halo_5"} {
		if _, statErr := os.Stat(pr.PlayerDBPath(slug, "H5Seul")); !os.IsNotExist(statErr) {
			t.Errorf("une base joueur de %s a été touchée pour un joueur refusé (stat: %v)", slug, statErr)
		}
	}
	startBackfillDryRun(t, s, store, "H5Seul", "halo_5")
}
