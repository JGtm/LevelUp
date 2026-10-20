//go:build cgo

// Package api — registry_build_queue_cgo_test.go : la MISE EN FILE de rejeu 2D, à l'unité.
//
// EnqueueReplayBuild est LA frontière de sécurité (résolution du manifeste sous tokens,
// dépôt des URL pré-signées dans le job). Ces tests l'exercent SANS réseau ni base de
// faits, par les deux seams nil-en-production (r.replayFilmResolver, r.replayJobFactsFn) :
// on prouve l'ASSEMBLAGE DU JOB (faits + URLs), les deux branches faits présents/absents,
// et la propagation des erreurs de frontière. Le store monitoring, lui, est RÉEL (DuckDB
// temporaire) — le job asserté est donc un vrai job persisté, relu via claim (seul chemin
// où le payload est servi). D'où le tag cgo.
//
// chunksFromResolver est testé en direct (frontière pure, résolveur mocké) : c'est le
// mapping référence-de-manifeste -> travail-de-job, plus les refus de dégradation propre.
package wire

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/port"
	syncpkg "levelup/go-api/internal/sync"
)

// stubFilmResolver mocke la SEULE opération Halo de la mise en file. gotID capture le
// match_id reçu, pour prouver ce qui a (ou n'a pas) atteint la frontière.
type stubFilmResolver struct {
	refs  []syncpkg.FilmChunkRef
	found bool
	err   error
	gotID string
}

func (s *stubFilmResolver) GetFilmChunkURLs(ctx context.Context, matchID string) ([]syncpkg.FilmChunkRef, bool, error) {
	s.gotID = matchID
	return s.refs, s.found, s.err
}

// buildQueueRegistry monte un registre avec un VRAI store monitoring (DuckDB temporaire)
// et sans autre dépendance : les seams de faits et de frontière sont posés par chaque test.
func buildQueueRegistry(t *testing.T) *ServiceRegistry {
	t.Helper()
	st, err := ops.NewMonitoringStore(context.Background(), filepath.Join(t.TempDir(), "monitoring.duckdb"))
	if err != nil {
		t.Fatalf("NewMonitoringStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &ServiceRegistry{
		cfg:             &config.AppConfig{RepoRoot: t.TempDir(), BuildWorkerToken: "tok-test"},
		monitoringStore: st,
	}
}

// claimPayload relit le payload d'un job en le prenant (seule réponse qui le sert).
func claimPayload(t *testing.T, reg *ServiceRegistry) *domain.BuildQueuePayload {
	t.Helper()
	claimed, err := reg.monitoringStore.ClaimBuildJob(context.Background(), "w-test", "host-test", "v-test")
	if err != nil {
		t.Fatalf("ClaimBuildJob: %v", err)
	}
	if claimed == nil || claimed.Payload == nil {
		t.Fatal("claim n'a pas rendu de job avec son payload")
	}
	return claimed.Payload
}

// TestEnqueueReplayBuild_FaitsPresents_JobPorteFaitsEtURLs : le cas nominal — un match dont
// la base porte les faits produit un job dont le payload embarque LES FAITS et LES URLs.
func TestEnqueueReplayBuild_FaitsPresents_JobPorteFaitsEtURLs(t *testing.T) {
	reg := buildQueueRegistry(t)
	const fullID = "abcd1234-5678-4abc-9def-0123456789ab"
	facts := port.MatchFacts{
		GameVariantName: "Slayer:Arena",
		TeamScores:      &[2]int{50, 43},
		Players: []domain.MatchPlayerFact{
			{XUID: "2533274800000001", Kills: 10, Deaths: 5, Assists: 2, TeamID: 0},
			{XUID: "2533274800000002", Kills: 8, Deaths: 9, Assists: 1, TeamID: 1},
		},
	}
	reg.replayJobFactsFn = func(ctx context.Context, titleSlug, matchID string) (string, []string, port.MatchFacts, error) {
		return fullID, []string{"Cliffhanger", "cliffhanger_ridgeline"}, facts, nil
	}
	resolver := &stubFilmResolver{found: true, refs: []syncpkg.FilmChunkRef{
		{Index: 0, ChunkType: 2, StartMS: 0, DurationMS: 1000, URL: "https://cdn/0"},
		{Index: 1, ChunkType: 2, StartMS: 1000, DurationMS: 1000, URL: "https://cdn/1"},
	}}
	reg.replayFilmResolver = resolver

	// On enfile par le PRÉFIXE court : le seam rend l'identité complète, comme le ferait la base.
	job, created, err := reg.EnqueueReplayBuild(context.Background(), titlePkg.DefaultSlug, "abcd1234")
	if err != nil {
		t.Fatalf("EnqueueReplayBuild: %v", err)
	}
	if !created {
		t.Fatal("created=false alors qu'aucun job préexistant")
	}
	if job.MatchID != fullID {
		t.Fatalf("job.MatchID = %q, veut le match_id COMPLET %q", job.MatchID, fullID)
	}
	if resolver.gotID != fullID {
		t.Fatalf("la frontière a résolu %q, veut le match_id complet %q", resolver.gotID, fullID)
	}

	p := claimPayload(t, reg)
	if p.MatchID != fullID || p.ShortID != titlePkg.FilmShortMatchID(fullID) {
		t.Fatalf("payload identité = (match=%q, short=%q)", p.MatchID, p.ShortID)
	}
	if len(p.MapNames) != 2 || p.MapNames[0] != "Cliffhanger" {
		t.Fatalf("payload.MapNames = %v, veut [Cliffhanger cliffhanger_ridgeline]", p.MapNames)
	}
	if len(p.Chunks) != 2 {
		t.Fatalf("payload.Chunks = %d, veut 2", len(p.Chunks))
	}
	want0 := domain.BuildQueueChunk{Index: 0, ChunkType: 2, StartMS: 0, DurationMS: 1000, URL: "https://cdn/0"}
	if p.Chunks[0] != want0 || p.Chunks[1].URL != "https://cdn/1" {
		t.Fatalf("payload.Chunks = %+v", p.Chunks)
	}
	// C'EST LE CRITÈRE : sans faits attachés, l'ouvrier sortirait un artefact appauvri.
	if p.Facts == nil {
		t.Fatal("payload.Facts nil alors que les faits sont NON vides")
	}
	if len(p.Facts.Players) != 2 || p.Facts.GameVariantName != "Slayer:Arena" || p.Facts.TeamScores == nil {
		t.Fatalf("payload.Facts = %+v", p.Facts)
	}
}

// TestEnqueueReplayBuild_FaitsAbsents_JobSansFaits : un film hors registre (faits vides) se
// met quand même en file — l'artefact sera VALIDE mais appauvri. NIL = rien du tout : le
// payload ne doit pas porter un objet MatchFacts vide.
func TestEnqueueReplayBuild_FaitsAbsents_JobSansFaits(t *testing.T) {
	reg := buildQueueRegistry(t)
	const fullID = "beef1234-5678-4abc-9def-0123456789ab"
	reg.replayJobFactsFn = func(ctx context.Context, titleSlug, matchID string) (string, []string, port.MatchFacts, error) {
		return fullID, []string{"Aquarius"}, port.MatchFacts{}, nil // Empty() == true
	}
	reg.replayFilmResolver = &stubFilmResolver{found: true, refs: []syncpkg.FilmChunkRef{
		{Index: 0, ChunkType: 2, URL: "https://cdn/only"},
	}}

	_, created, err := reg.EnqueueReplayBuild(context.Background(), titlePkg.DefaultSlug, "beef1234")
	if err != nil || !created {
		t.Fatalf("EnqueueReplayBuild: err=%v created=%v", err, created)
	}
	p := claimPayload(t, reg)
	if p.Facts != nil {
		t.Fatalf("payload.Facts = %+v, veut nil quand les faits sont vides (NIL = rien du tout, pas un objet vide)", p.Facts)
	}
	if len(p.Chunks) != 1 || p.Chunks[0].URL != "https://cdn/only" {
		t.Fatalf("payload.Chunks = %+v, veut le morceau unique", p.Chunks)
	}
}

// TestEnqueueReplayBuild_IdentiteInconnue_EchoueSansToucherLaFrontiere : une identité qui
// échoue est fatale à la mise en file, ET la frontière Halo ne doit JAMAIS être sollicitée
// (pas de résolution de manifeste pour un match qu'on ne sait pas nommer).
func TestEnqueueReplayBuild_IdentiteInconnue_EchoueSansToucherLaFrontiere(t *testing.T) {
	reg := buildQueueRegistry(t)
	sentinelle := errors.New("match inconnu du registre: zzz")
	reg.replayJobFactsFn = func(ctx context.Context, titleSlug, matchID string) (string, []string, port.MatchFacts, error) {
		return "", nil, port.MatchFacts{}, sentinelle
	}
	resolver := &stubFilmResolver{found: true, refs: []syncpkg.FilmChunkRef{{Index: 0, URL: "x"}}}
	reg.replayFilmResolver = resolver

	_, created, err := reg.EnqueueReplayBuild(context.Background(), titlePkg.DefaultSlug, "zzz")
	if err == nil {
		t.Fatal("EnqueueReplayBuild devrait échouer quand l'identité est inconnue")
	}
	if !errors.Is(err, sentinelle) {
		t.Fatalf("err = %v, veut la sentinelle d'identité", err)
	}
	if created {
		t.Fatal("created=true alors que la mise en file a échoué")
	}
	if resolver.gotID != "" {
		t.Fatalf("la frontière a été sollicitée (%q) alors que l'identité a échoué", resolver.gotID)
	}
}

// TestEnqueueReplayBuild_FilmExpire_Echoue : un film absent/expiré côté serveur est une
// ERREUR de mise en file (dite tout de suite), pas un job qui échouerait plus tard.
func TestEnqueueReplayBuild_FilmExpire_Echoue(t *testing.T) {
	reg := buildQueueRegistry(t)
	reg.replayJobFactsFn = func(ctx context.Context, titleSlug, matchID string) (string, []string, port.MatchFacts, error) {
		return "dead0001-5678-4abc-9def-0123456789ab", nil, port.MatchFacts{}, nil
	}
	reg.replayFilmResolver = &stubFilmResolver{found: false} // 404/410 : ~29 % du corpus

	_, created, err := reg.EnqueueReplayBuild(context.Background(), titlePkg.DefaultSlug, "dead0001")
	if err == nil || !strings.Contains(err.Error(), "film absent ou expiré") {
		t.Fatalf("err = %v, veut 'film absent ou expiré'", err)
	}
	if created {
		t.Fatal("created=true alors que le film est expiré")
	}
}

// ── Frontière pure : chunksFromResolver ──────────────────────────────────────────────

func TestChunksFromResolver_Succes_MappeLesReferences(t *testing.T) {
	stub := &stubFilmResolver{found: true, refs: []syncpkg.FilmChunkRef{
		{Index: 0, ChunkType: 2, StartMS: 0, DurationMS: 500, URL: "https://cdn/a"},
		{Index: 1, ChunkType: 6, StartMS: 500, DurationMS: 500, URL: "https://cdn/b"},
	}}
	out, err := chunksFromResolver(context.Background(), stub, "match-x")
	if err != nil {
		t.Fatalf("chunksFromResolver: %v", err)
	}
	if stub.gotID != "match-x" {
		t.Fatalf("le résolveur a reçu %q, veut match-x", stub.gotID)
	}
	if len(out) != 2 {
		t.Fatalf("out = %d morceaux, veut 2", len(out))
	}
	want := domain.BuildQueueChunk{Index: 1, ChunkType: 6, StartMS: 500, DurationMS: 500, URL: "https://cdn/b"}
	if out[1] != want {
		t.Fatalf("out[1] = %+v, veut %+v", out[1], want)
	}
}

func TestChunksFromResolver_FilmAbsent_DegradationPropre(t *testing.T) {
	// found=false : erreur de mise en file explicite, jamais de panic.
	_, err := chunksFromResolver(context.Background(), &stubFilmResolver{found: false}, "expire")
	if err == nil || !strings.Contains(err.Error(), "film absent ou expiré") {
		t.Fatalf("err = %v, veut 'film absent ou expiré'", err)
	}
}

func TestChunksFromResolver_ManifesteVide_Erreur(t *testing.T) {
	// found=true mais zéro référence : même refus (rien à construire).
	_, err := chunksFromResolver(context.Background(), &stubFilmResolver{found: true, refs: nil}, "vide")
	if err == nil || !strings.Contains(err.Error(), "rien à construire") {
		t.Fatalf("err = %v, veut 'rien à construire'", err)
	}
}

func TestChunksFromResolver_ErreurManifeste_Enveloppee(t *testing.T) {
	// Une panne réseau/manifeste remonte ENVELOPPÉE (jamais avalée), avec le préfixe de frontière.
	boom := errors.New("boom réseau")
	_, err := chunksFromResolver(context.Background(), &stubFilmResolver{err: boom}, "m")
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("err = %v, veut envelopper boom", err)
	}
	if !strings.Contains(err.Error(), "résolution du manifeste de film") {
		t.Fatalf("err = %v, veut le préfixe de frontière", err)
	}
}
