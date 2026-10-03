//go:build cgo

// Package ops — build_queue_cgo_test.go : la file durable tient ses quatre
// promesses (mettre en file, prendre atomiquement, rendre un résultat, reprendre
// après la mort d'un ouvrier). Driver DuckDB requis.
package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// enqueueTestJob met un job de test en file et rend son identifiant.
func enqueueTestJob(t *testing.T, st *MonitoringStore, matchID string) domain.BuildQueueJob {
	t.Helper()
	job, created, err := st.EnqueueBuildJob(context.Background(), EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: "halo_infinite",
		MatchID:   matchID,
		Payload: &domain.BuildQueuePayload{
			MatchID: matchID, ShortID: matchID[:8], TitleSlug: "halo_infinite",
			MapNames: []string{"Cliffhanger"},
			Chunks: []domain.BuildQueueChunk{
				{Index: 0, ChunkType: 1, URL: "https://cdn.example/pre-signed/header.bin"},
				{Index: 1, ChunkType: 2, URL: "https://cdn.example/pre-signed/rep.bin"},
			},
		},
	})
	if err != nil {
		t.Fatalf("EnqueueBuildJob: %v", err)
	}
	if !created {
		t.Fatalf("EnqueueBuildJob: job attendu créé pour %s", matchID)
	}
	return job
}

// TestBuildQueue_EnqueueClaimComplete parcourt le cycle nominal et vérifie que
// le travail RÉSOLU (URL pré-signées) n'est servi qu'à la prise — c'est la seule
// réponse où l'ouvrier en a besoin.
func TestBuildQueue_EnqueueClaimComplete(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	job := enqueueTestJob(t, st, "aaaaaaaa-1111-2222-3333-444444444444")

	claimed, err := st.ClaimBuildJob(ctx, "ouvrier-1", "poste", "test/1")
	if err != nil {
		t.Fatalf("ClaimBuildJob: %v", err)
	}
	if claimed == nil {
		t.Fatal("ClaimBuildJob: aucun job pris alors que la file en contient un")
	}
	if claimed.JobID != job.JobID {
		t.Fatalf("job pris = %s, attendu %s", claimed.JobID, job.JobID)
	}
	if claimed.Status != domain.JobStatusRunning || claimed.WorkerID != "ouvrier-1" {
		t.Fatalf("prise mal enregistrée : statut=%s worker=%s", claimed.Status, claimed.WorkerID)
	}
	if claimed.Payload == nil || len(claimed.Payload.Chunks) != 2 {
		t.Fatalf("le travail résolu doit accompagner la prise, obtenu %+v", claimed.Payload)
	}
	if claimed.Payload.Chunks[0].URL == "" {
		t.Fatal("URL CDN pré-signée absente du travail — l'ouvrier n'aurait rien à télécharger")
	}

	if err := st.CompleteBuildJob(ctx, CompleteBuildJobRequest{
		JobID: job.JobID, WorkerID: "ouvrier-1", Succeeded: true, ResultJSON: `{"tracks":8}`,
	}); err != nil {
		t.Fatalf("CompleteBuildJob: %v", err)
	}

	report, err := st.BuildQueueReport(ctx, 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if report.Counts.Succeeded != 1 || report.Counts.Queued != 0 || report.Counts.Running != 0 {
		t.Fatalf("compteurs inattendus : %+v", report.Counts)
	}
	if len(report.Jobs) != 1 || report.Jobs[0].ResultJSON != `{"tracks":8}` {
		t.Fatalf("résultat non restitué : %+v", report.Jobs)
	}
	if report.Jobs[0].Payload != nil {
		t.Fatal("la vue admin ne doit PAS porter les URL pré-signées (volume + expiration)")
	}
	if len(report.Workers) != 1 || report.Workers[0].WorkerID != "ouvrier-1" || !report.Workers[0].Online {
		t.Fatalf("l'ouvrier doit apparaître vivant : %+v", report.Workers)
	}
}

// TestBuildQueue_LesFaitsSurviventALaFile — LES FAITS DU MATCH ARRIVENT INTACTS CHEZ L'OUVRIER.
//
// POURQUOI CE TEST EXISTE. L'ouvrier n'a aucune base : les faits du match ne lui parviennent que
// par le job, et le job les range en TEXTE dans `payload_json` avant d'être relu à la prise. Le
// trajet traverse donc deux sérialisations et une base — trois occasions de perdre un champ en
// silence. Ce qu'on perdrait, c'est MESURÉ (témoin 7344d24f, 2026-08-24) : actions d'objectif
// 246 -> 0, zones du mode 3 -> 0, joueurs de la courbe de score 8 -> 0, identité des camps
// `b` -> `unresolved`.
//
// Le `TeamScores` est un POINTEUR de tableau : c'est le champ qui casse le plus discrètement
// (un nil rendu à la place de [3,0] ne fait échouer aucune compilation), d'où sa vérification
// explicite.
func TestBuildQueue_LesFaitsSurviventALaFile(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	const matchID = "dddddddd-1111-2222-3333-444444444444"

	scores := [2]int{3, 0}
	facts := &domain.MatchFacts{
		Players: []domain.MatchPlayerFact{
			{XUID: "2533274819954312", Kills: 12, Deaths: 7, Assists: 3, TeamID: 0},
			{XUID: "2535469190789936", Kills: 9, Deaths: 11, Assists: 5, TeamID: 1},
		},
		TeamScores:      &scores,
		GameVariantName: "CTF:Arena",
		MapID:           "e859cf75-9b8a-429a-91be-2376681c8537",
	}
	if _, _, err := st.EnqueueBuildJob(ctx, EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: "halo_infinite",
		MatchID:   matchID,
		Payload: &domain.BuildQueuePayload{
			MatchID: matchID, ShortID: matchID[:8], TitleSlug: "halo_infinite",
			MapNames: []string{"Catalyst"},
			Chunks:   []domain.BuildQueueChunk{{Index: 0, ChunkType: 1, URL: "https://cdn.example/h.bin"}},
			Facts:    facts,
		},
	}); err != nil {
		t.Fatalf("EnqueueBuildJob: %v", err)
	}

	claimed, err := st.ClaimBuildJob(ctx, "ouvrier-1", "poste", "test/1")
	if err != nil || claimed == nil {
		t.Fatalf("ClaimBuildJob: %v (job=%v)", err, claimed)
	}
	got := claimed.Payload.Facts
	if got == nil {
		t.Fatal("les faits n'ont pas survécu à la file — l'ouvrier construirait un artefact appauvri")
	}
	if got.GameVariantName != "CTF:Arena" {
		t.Errorf("variante = %q, attendu \"CTF:Arena\" (sans elle, aucune action d'objectif n'est nommable)",
			got.GameVariantName)
	}
	if got.MapID != facts.MapID {
		t.Errorf("mapId = %q, attendu %q (sans lui : ni zones de mode, ni socles de drapeau)",
			got.MapID, facts.MapID)
	}
	if got.TeamScores == nil || *got.TeamScores != scores {
		t.Errorf("scores des camps = %v, attendu %v", got.TeamScores, scores)
	}
	if len(got.Players) != 2 {
		t.Fatalf("lignes de match = %d, attendu 2 (le triplet est la CLÉ d'appariement des slots)",
			len(got.Players))
	}
	if got.Players[0] != facts.Players[0] || got.Players[1] != facts.Players[1] {
		t.Errorf("triplet d'appariement altéré : %+v, attendu %+v", got.Players, facts.Players)
	}
}

// TestBuildQueue_SansFaitsResteUnJobValide — un match hors registre se met en file quand même.
//
// C'est le cas RÉEL d'un film du cache dont le match n'a jamais été synchronisé : il doit
// produire un artefact appauvri, pas bloquer la file. Le pointeur nil est la forme normale.
func TestBuildQueue_SansFaitsResteUnJobValide(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	enqueueTestJob(t, st, "eeeeeeee-1111-2222-3333-444444444444")

	claimed, err := st.ClaimBuildJob(ctx, "ouvrier-1", "", "")
	if err != nil || claimed == nil {
		t.Fatalf("ClaimBuildJob: %v (job=%v)", err, claimed)
	}
	if claimed.Payload == nil {
		t.Fatal("un job sans faits doit garder son travail résolu")
	}
	if claimed.Payload.Facts != nil {
		t.Errorf("faits inventés alors qu'aucun n'a été fourni : %+v", claimed.Payload.Facts)
	}
}

// TestBuildQueue_ClaimNeSertJamaisDeuxFoisLeMemeJob : deux ouvriers, un seul job.
// C'est l'invariant qui justifie le verrou de la section critique.
func TestBuildQueue_ClaimNeSertJamaisDeuxFoisLeMemeJob(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	enqueueTestJob(t, st, "bbbbbbbb-1111-2222-3333-444444444444")

	first, err := st.ClaimBuildJob(ctx, "ouvrier-1", "", "")
	if err != nil || first == nil {
		t.Fatalf("première prise: %v (job=%v)", err, first)
	}
	second, err := st.ClaimBuildJob(ctx, "ouvrier-2", "", "")
	if err != nil {
		t.Fatalf("seconde prise: %v", err)
	}
	if second != nil {
		t.Fatalf("le même job a été servi deux fois (à %s puis %s)", first.WorkerID, second.WorkerID)
	}
}

// TestBuildQueue_MiseEnFileIdempotente : re-demander un match déjà en file ne
// crée pas un second job (sinon deux ouvriers décoderaient le même film).
func TestBuildQueue_MiseEnFileIdempotente(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	first := enqueueTestJob(t, st, "cccccccc-1111-2222-3333-444444444444")

	again, created, err := st.EnqueueBuildJob(ctx, EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: "halo_infinite",
		MatchID:   "cccccccc-1111-2222-3333-444444444444",
	})
	if err != nil {
		t.Fatalf("seconde mise en file: %v", err)
	}
	if created {
		t.Fatal("un doublon a été créé alors que le match était déjà en file")
	}
	if again.JobID != first.JobID {
		t.Fatalf("job rendu = %s, attendu le job existant %s", again.JobID, first.JobID)
	}
}

// TestBuildQueue_OuvrierDisparu_JobRetourneEnFile : le cœur de la promesse. Un
// job dont le bail a expiré revient en file, et le rendu tardif de l'ouvrier
// disparu est REFUSÉ (sinon il écraserait le travail de son remplaçant).
func TestBuildQueue_OuvrierDisparu_JobRetourneEnFile(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	job := enqueueTestJob(t, st, "dddddddd-1111-2222-3333-444444444444")

	claimed, err := st.ClaimBuildJob(ctx, "ouvrier-mourant", "", "")
	if err != nil || claimed == nil {
		t.Fatalf("prise: %v (job=%v)", err, claimed)
	}

	// L'ouvrier meurt : on force l'expiration de son bail (append d'un événement
	// `running` au bail périmé — jamais un UPDATE, la table est append-only).
	expireLeaseForTest(t, st, job.JobID, "ouvrier-mourant")

	n, err := st.ReclaimExpiredBuildJobs(ctx)
	if err != nil {
		t.Fatalf("ReclaimExpiredBuildJobs: %v", err)
	}
	if n != 1 {
		t.Fatalf("jobs repris = %d, attendu 1", n)
	}

	// Le remplaçant récupère le travail, avec son travail résolu intact.
	next, err := st.ClaimBuildJob(ctx, "ouvrier-remplacant", "", "")
	if err != nil || next == nil {
		t.Fatalf("reprise par le remplaçant: %v (job=%v)", err, next)
	}
	if next.JobID != job.JobID {
		t.Fatalf("job repris = %s, attendu %s", next.JobID, job.JobID)
	}
	if next.Attempt != 1 {
		t.Fatalf("tentative = %d, attendu 1 (le compteur doit avancer)", next.Attempt)
	}
	if next.Payload == nil || len(next.Payload.Chunks) != 2 {
		t.Fatal("le travail résolu doit survivre à la reprise")
	}

	// Le mort revient et prétend rendre son résultat : refus.
	err = st.CompleteBuildJob(ctx, CompleteBuildJobRequest{
		JobID: job.JobID, WorkerID: "ouvrier-mourant", Succeeded: true,
	})
	if !errors.Is(err, ErrBuildJobNotClaimed) {
		t.Fatalf("rendu périmé accepté (err=%v) — il aurait écrasé le travail du remplaçant", err)
	}
}

// TestBuildQueue_OuvrierHorsLigne : un ouvrier muet depuis plus longtemps que le
// seuil s'affiche hors ligne, sans disparaître du tableau.
func TestBuildQueue_OuvrierHorsLigne(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	if err := st.HeartbeatBuildWorker(ctx, HeartbeatRequest{WorkerID: "ouvrier-muet", Hostname: "poste", Version: "test/1"}); err != nil {
		t.Fatalf("HeartbeatBuildWorker: %v", err)
	}
	backdateWorkerBeat(t, st, "ouvrier-muet", time.Now().UTC().Add(-2*domain.WorkerOfflineAfter))

	report, err := st.BuildQueueReport(ctx, 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if len(report.Workers) != 1 {
		t.Fatalf("ouvriers = %d, attendu 1 (un ouvrier mort reste visible)", len(report.Workers))
	}
	if report.Workers[0].Online {
		t.Fatal("un ouvrier silencieux depuis 2× le seuil doit s'afficher hors ligne")
	}
}

// expireLeaseForTest simule la mort d'un ouvrier : on relit le job puis on
// append un événement `running` dont le bail est DÉJÀ passé, par le writer
// normal (append-only, comme la production — jamais un UPDATE).
//
// La relecture est explicite plutôt qu'un INSERT ... SELECT : l'inférence de type
// des paramètres d'un INSERT ... SELECT décale les horodatages du fuseau local
// dans ce driver, ce qui rendrait le test faux d'une heure sans rien dire.
func expireLeaseForTest(t *testing.T, st *MonitoringStore, jobID, workerID string) {
	t.Helper()
	ctx := context.Background()
	var jobType, titleSlug, matchID, payload string
	if err := st.db.QueryRow(ctx,
		`SELECT job_type, COALESCE(title_slug, ''), COALESCE(match_id, ''), COALESCE(payload_json, '')
		 FROM build_jobs_latest WHERE job_id = ?`, jobID,
	).Scan(&jobType, &titleSlug, &matchID, &payload); err != nil {
		t.Fatalf("relecture du job %s: %v", jobID, err)
	}
	w, err := st.acquire()
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer w.Release()
	past := time.Now().UTC().Add(-time.Hour)
	if _, err := w.ExecContext(ctx,
		`INSERT INTO build_job_events
			(job_id, job_type, title_slug, match_id, status, priority, attempt,
			 worker_id, lease_expires_at, payload_json, enqueued_at)
		 VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?)`,
		jobID, jobType, titleSlug, matchID, string(domain.JobStatusRunning),
		workerID, past, payload, past); err != nil {
		t.Fatalf("expiration du bail: %v", err)
	}
}

// backdateWorkerBeat vieillit le dernier battement d'un ouvrier (append d'un
// événement antidaté — la vue prend le beat_at le plus récent, donc on écrit
// l'unique battement voulu sur une base fraîche).
func backdateWorkerBeat(t *testing.T, st *MonitoringStore, workerID string, at time.Time) {
	t.Helper()
	w, err := st.acquire()
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer w.Release()
	if _, err := w.ExecContext(context.Background(),
		`DELETE FROM build_worker_events WHERE worker_id = ?`, workerID); err != nil {
		t.Fatalf("purge des battements: %v", err)
	}
	if _, err := w.ExecContext(context.Background(),
		`INSERT INTO build_worker_events (worker_id, hostname, version, beat_at)
		 VALUES (?, ?, ?, ?)`, workerID, "poste", "test/1", at); err != nil {
		t.Fatalf("battement antidaté: %v", err)
	}
}
