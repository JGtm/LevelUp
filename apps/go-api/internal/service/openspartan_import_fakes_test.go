package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// ─── faux magasin partagé ─────────────────────────────────────────────────────

type fakeSharedStore struct {
	matches    []port.OpenSpartanMatchWrite
	highlights int
	aliases    []string
	persistErr error
	hlErr      error
	aliasErr   error
}

func (f *fakeSharedStore) PersistMatch(_ context.Context, w port.OpenSpartanMatchWrite) error {
	if f.persistErr != nil {
		return f.persistErr
	}
	f.matches = append(f.matches, w)
	return nil
}

func (f *fakeSharedStore) InsertHighlightEvents(_ context.Context, _ string, events []highlightevent.HighlightEvent) (int, error) {
	if f.hlErr != nil {
		return 0, f.hlErr
	}
	f.highlights += len(events)
	return len(events), nil
}

func (f *fakeSharedStore) UpsertOwnerAlias(_ context.Context, xuid, _ string) error {
	if f.aliasErr != nil {
		return f.aliasErr
	}
	f.aliases = append(f.aliases, xuid)
	return nil
}

type fakeSharedAcquirer struct {
	store    *fakeSharedStore
	err      error
	acquired int
	released int
}

func (f *fakeSharedAcquirer) AcquireShared(context.Context) (port.OpenSpartanSharedStore, func(), error) {
	f.acquired++
	if f.err != nil {
		return nil, nil, f.err
	}
	return f.store, func() { f.released++ }, nil
}

func TestOpenSpartanImport_EcritParLePortEtLibereLeMagasin(t *testing.T) {
	dir := t.TempDir()
	path := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	acq := &fakeSharedAcquirer{store: &fakeSharedStore{}}

	res, err := NewOpenSpartanImportService(acq).Import(context.Background(), testOwnerXUID, path,
		port.OpenSpartanImportOptions{StashDir: filepath.Join(dir, "stash")})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if acq.acquired != 1 || acq.released != 1 {
		t.Errorf("magasin pris %d fois, libéré %d fois, attendu 1 et 1", acq.acquired, acq.released)
	}
	if len(acq.store.matches) != 1 || res.InsertedMatches != 1 {
		t.Fatalf("matchs écrits = %d (résultat %d), attendu 1", len(acq.store.matches), res.InsertedMatches)
	}
	w := acq.store.matches[0]
	if len(w.Participants) == 0 || w.Registry.MatchID == "" {
		t.Errorf("l'écriture du match est vide : %+v", w)
	}
	if acq.store.highlights != 1 || len(acq.store.aliases) != 1 || acq.store.aliases[0] != testOwnerXUID {
		t.Errorf("événements = %d, alias de l'importeur = %v, attendu 1 et [%s]", acq.store.highlights, acq.store.aliases, testOwnerXUID)
	}
}

func TestOpenSpartanImport_UneErreurDEcritureEstComptee(t *testing.T) {
	dir := t.TempDir()
	path := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	acq := &fakeSharedAcquirer{store: &fakeSharedStore{
		persistErr: errors.New("écriture du match refusée"),
		aliasErr:   errors.New("alias refusé"),
	}}

	res, err := NewOpenSpartanImportService(acq).Import(context.Background(), testOwnerXUID, path,
		port.OpenSpartanImportOptions{StashDir: filepath.Join(dir, "stash")})
	if err != nil {
		t.Fatalf("Import doit rendre ses erreurs par étape, pas échouer : %v", err)
	}
	stages := map[string]string{}
	for _, e := range res.Errors {
		stages[e.Stage] = e.Err
	}
	if stages["persist_match"] != "écriture du match refusée" {
		t.Errorf("erreur d'écriture du match absente ou altérée : %v", res.Errors)
	}
	if stages["upsert_alias"] != "alias refusé" {
		t.Errorf("erreur d'alias absente ou altérée : %v", res.Errors)
	}
	if res.InsertedMatches != 0 {
		t.Errorf("InsertedMatches = %d, attendu 0", res.InsertedMatches)
	}
}

func TestOpenSpartanImport_LaPriseDuMagasinQuiEchoueArreteL_Import(t *testing.T) {
	dir := t.TempDir()
	path := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	boom := errors.New("writer indisponible")
	acq := &fakeSharedAcquirer{err: boom}

	_, err := NewOpenSpartanImportService(acq).Import(context.Background(), testOwnerXUID, path, port.OpenSpartanImportOptions{StashDir: dir})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, attendu l'échec de la prise du magasin", err)
	}
}

func TestOpenSpartanImport_LaSimulationNePrendAucunMagasin(t *testing.T) {
	dir := t.TempDir()
	path := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	acq := &fakeSharedAcquirer{store: &fakeSharedStore{}}

	res, err := NewOpenSpartanImportService(acq).Import(context.Background(), testOwnerXUID, path,
		port.OpenSpartanImportOptions{DryRun: true, StashDir: dir})
	if err != nil {
		t.Fatalf("Import (simulation) : %v", err)
	}
	if acq.acquired != 0 || len(acq.store.matches) != 0 {
		t.Errorf("simulation : magasin pris %d fois, %d match(s) écrit(s), attendu 0 et 0", acq.acquired, len(acq.store.matches))
	}
	if res.InsertedMatches != 1 {
		t.Errorf("la simulation doit compter le match : %d", res.InsertedMatches)
	}
}

// ─── fausse session d'après-import ────────────────────────────────────────────

type fakePostImportSession struct {
	prime       []port.OpenSpartanEnrichmentFailure
	primeErr    error
	primeCalls  [][]string
	csrErr      error
	lusrErr     error
	perfErr     error
	citationErr error
	closed      bool
}

func (f *fakePostImportSession) Close() error { f.closed = true; return nil }
func (f *fakePostImportSession) PrimeEnrichment(_ context.Context, ids []string) ([]port.OpenSpartanEnrichmentFailure, error) {
	f.primeCalls = append(f.primeCalls, ids)
	return f.prime, f.primeErr
}
func (f *fakePostImportSession) ProjectCSR(context.Context, string) (int, error) {
	return 4, f.csrErr
}
func (f *fakePostImportSession) RecomputeLUSR(context.Context, string) (int, error) {
	return 5, f.lusrErr
}
func (f *fakePostImportSession) ComputePerfScores(context.Context, string, bool) (int, error) {
	return 6, f.perfErr
}
func (f *fakePostImportSession) BackfillCitations(context.Context, string, []string) error {
	return f.citationErr
}

type fakePostImportStore struct {
	session *fakePostImportSession
	err     error
	paths   port.OpenSpartanPostImportPaths
}

func (f *fakePostImportStore) Open(_ context.Context, paths port.OpenSpartanPostImportPaths) (port.OpenSpartanPostImportSession, error) {
	f.paths = paths
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

func newFakePostImport() (*OpenSpartanPostImportService, *fakePostImportSession) {
	session := &fakePostImportSession{}
	cfg := &config.AppConfig{RepoRoot: "unused"}
	return NewOpenSpartanPostImportService(cfg, &fakePostImportStore{session: session}), session
}

func TestPostImportFake_OuvreLaBaseDuTitreEtLaFerme(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	store := &fakePostImportStore{session: &fakePostImportSession{}}
	svc := NewOpenSpartanPostImportService(cfg, store)

	// Les étapes de la session sont fausses ; le recalcul des sessions, lui, lit les bases du titre (absentes
	// ici) et range son échec dans le résultat : on ne regarde que le reste.
	res, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, []string{"m1"},
		port.OpenSpartanPostImportOptions{TitleSlug: "halo_infinite"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !store.session.closed {
		t.Error("la session doit être fermée à la fin du Run")
	}
	if store.paths.Player != config.PlayerDBPath(cfg, "halo_infinite", postImportTestGamertag) ||
		store.paths.Shared != config.SharedDBPath(cfg, "halo_infinite") ||
		store.paths.Metadata == "" || store.paths.PVE == "" {
		t.Errorf("chemins passés à la session : %+v", store.paths)
	}
	if res.CSRProjected != 4 || res.LUSRRecomputed != 5 || res.PerfScoresTouched != 6 || !res.CitationsBackfilled {
		t.Errorf("résultats des étapes non repris : %+v", res)
	}
	if len(store.session.primeCalls) != 1 || len(store.session.primeCalls[0]) != 1 {
		t.Errorf("amorçage appelé avec %v, attendu une fois avec [m1]", store.session.primeCalls)
	}
}

func TestPostImportFake_UneBaseQuiNeS_OuvrePasEstFatale(t *testing.T) {
	boom := errors.New("base du joueur illisible")
	svc := NewOpenSpartanPostImportService(&config.AppConfig{RepoRoot: t.TempDir()}, &fakePostImportStore{err: boom})
	_, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, nil, port.OpenSpartanPostImportOptions{})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, attendu l'échec de l'ouverture", err)
	}
}

func TestPostImportFake_AmorcageSansArretALaPremiereErreur(t *testing.T) {
	svc, session := newFakePostImport()
	session.prime = []port.OpenSpartanEnrichmentFailure{
		{MatchID: "m2", Err: errors.New("insertion refusée m2")},
		{MatchID: "m4", Err: errors.New("insertion refusée m4")},
	}
	before := observability.LoadCounterT("halo_infinite", counterPrimeEnrichmentFailed)
	var result port.OpenSpartanPostImportResult

	svc.ensureEnrichmentRows(context.Background(), session, "halo_infinite", []string{"m1", "m2", "m3", "m4"}, &result)

	if len(result.Errors) != 2 {
		t.Fatalf("erreurs = %v, attendu une par match en échec", result.Errors)
	}
	for i, id := range []string{"m2", "m4"} {
		e := result.Errors[i]
		if e.Stage != stagePrimeEnrichment || e.MatchID != id || e.Err != "insertion refusée "+id {
			t.Errorf("erreur %d inattendue : %+v", i, e)
		}
	}
	if d := observability.LoadCounterT("halo_infinite", counterPrimeEnrichmentFailed) - before; d != 2 {
		t.Errorf("compteur : +%d, attendu +2", d)
	}
}

func TestPostImportFake_AmorcageLectureImpossible(t *testing.T) {
	svc, session := newFakePostImport()
	session.primeErr = errors.New("lecture impossible")
	var result port.OpenSpartanPostImportResult

	svc.ensureEnrichmentRows(context.Background(), session, "halo_infinite", []string{"m1"}, &result)

	if len(result.Errors) != 1 || result.Errors[0].Stage != stagePrimeEnrichment || result.Errors[0].MatchID != "" ||
		result.Errors[0].Err != "lecture impossible" {
		t.Errorf("erreurs = %+v", result.Errors)
	}
}

func TestPostImportFake_AmorcageSansMatchNAppellePasLaSession(t *testing.T) {
	svc, session := newFakePostImport()
	var result port.OpenSpartanPostImportResult
	svc.ensureEnrichmentRows(context.Background(), session, "halo_infinite", nil, &result)
	if len(session.primeCalls) != 0 || len(result.Errors) != 0 {
		t.Errorf("appels = %v, erreurs = %v", session.primeCalls, result.Errors)
	}
}

// Les étapes rangent leur échec sous leur nom, ou sous celui que la session impose (prise d'une base) ; le
// texte de l'erreur est celui de la cause.
func TestPostImportFake_LesEchecsDEtapeGardentLeurNom(t *testing.T) {
	svc, session := newFakePostImport()
	session.csrErr = &port.OpenSpartanStageError{Stage: "csr_acquire", Err: errors.New("writer pris")}
	session.lusrErr = errors.New("rejeu impossible")
	session.perfErr = &port.OpenSpartanStageError{Stage: "perf_scores_acquire", Err: errors.New("writer pris")}
	session.citationErr = &port.OpenSpartanStageError{Stage: "open_metadata", Err: errors.New("metadata tenue")}
	var result port.OpenSpartanPostImportResult
	ctx := context.Background()

	svc.recomputeCSR(ctx, session, "x", &result)
	svc.recomputeLUSR(ctx, session, "x", &result)
	svc.recomputePerfScores(ctx, session, "x", false, &result)
	svc.recomputeCitations(ctx, session, "x", []string{"m1"}, &result)

	want := []port.OpenSpartanPostImportError{
		{Stage: "csr_acquire", Err: "writer pris"},
		{Stage: "lusr", Err: "rejeu impossible"},
		{Stage: "perf_scores_acquire", Err: "writer pris"},
		{Stage: "open_metadata", Err: "metadata tenue"},
	}
	if len(result.Errors) != len(want) {
		t.Fatalf("erreurs = %+v", result.Errors)
	}
	for i, w := range want {
		if result.Errors[i] != w {
			t.Errorf("erreur %d = %+v, attendu %+v", i, result.Errors[i], w)
		}
	}
	if result.CSRProjected != 0 || result.LUSRRecomputed != 0 || result.PerfScoresTouched != 0 || result.CitationsBackfilled {
		t.Errorf("un échec ne doit rien compter comme fait : %+v", result)
	}
}

func TestPostImportFake_CitationsSansMatchSontSautees(t *testing.T) {
	svc, session := newFakePostImport()
	session.citationErr = errors.New("ne doit pas être appelé")
	var result port.OpenSpartanPostImportResult
	svc.recomputeCitations(context.Background(), session, "x", nil, &result)
	if len(result.Errors) != 0 || result.CitationsBackfilled {
		t.Errorf("rien à faire sans match : %+v", result)
	}
}
