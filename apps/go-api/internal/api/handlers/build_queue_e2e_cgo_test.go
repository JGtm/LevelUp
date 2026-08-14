//go:build cgo

// Package handlers — build_queue_e2e_cgo_test.go : LA PREUVE DE BOUT EN BOUT.
//
// Un vrai serveur HTTP, une vraie base DuckDB, les vraies routes : on met un job
// en file, un « ouvrier » le prend par le protocole, rend son résultat, et
// l'admin voit l'ensemble par GET /admin/monitoring/build-queue. Aucune pièce
// n'est simulée entre les deux bouts — c'est ce qui distingue ce test des tests
// de contrat HTTP voisins, qui eux stubent le store.
//
// Driver DuckDB requis (tag cgo).
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
)

const e2eWorkerToken = "jeton-de-test-de-bout-en-bout"

// e2eStack monte les routes ouvrier ET la vue admin sur un vrai serveur HTTP,
// toutes deux branchées sur le MÊME store — c'est l'invariant du plan : l'état
// vit côté web, donc l'admin voit le travail distant sans interroger l'ouvrier.
func e2eStack(t *testing.T) (*httptest.Server, *ops.MonitoringStore) {
	t.Helper()
	st, err := ops.NewMonitoringStore(context.Background(), filepath.Join(t.TempDir(), "monitoring.duckdb"))
	if err != nil {
		t.Fatalf("NewMonitoringStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	workerH := NewBuildWorkerHandler(e2eWorkerToken,
		st.ClaimBuildJob,
		st.CompleteBuildJob,
		st.HeartbeatBuildWorker)
	adminH := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).
		WithBuildQueue(func(ctx context.Context, limit int) (domain.AdminBuildQueueResponse, error) {
			resp, rerr := st.BuildQueueReport(ctx, limit)
			resp.Enabled = true
			return resp, rerr
		})

	r := chi.NewRouter()
	r.Route("/internal", func(r chi.Router) { workerH.Mount(r) })
	r.Route("/admin", func(r chi.Router) { adminH.Mount(r) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, st
}

// e2ePost envoie un appel du protocole ouvrier et décode la réponse.
func e2ePost(t *testing.T, srv *httptest.Server, path, token string, body, out any) int {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("requête %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("appel %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, _ := io.ReadAll(resp.Body)
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(payload, out); err != nil {
			t.Fatalf("décodage %s: %v (body=%s)", path, err, payload)
		}
	}
	return resp.StatusCode
}

// e2eAdminView lit la vue admin de la file.
func e2eAdminView(t *testing.T, srv *httptest.Server) domain.AdminBuildQueueResponse {
	t.Helper()
	resp, err := srv.Client().Get(srv.URL + "/admin/monitoring/build-queue")
	if err != nil {
		t.Fatalf("GET build-queue: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET build-queue: status %d body=%s", resp.StatusCode, body)
	}
	var out domain.AdminBuildQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("décodage vue admin: %v", err)
	}
	return out
}

// TestBuildQueue_DeBoutEnBout : mise en file → prise par l'ouvrier → battement →
// rendu du résultat, chaque étape constatée dans la vue admin.
func TestBuildQueue_DeBoutEnBout(t *testing.T) {
	ctx := context.Background()
	srv, st := e2eStack(t)

	// ── 1. Le web met en file, travail RÉSOLU compris (URL pré-signées) ──────
	const matchID = "11111111-2222-3333-4444-555555555555"
	job, created, err := st.EnqueueBuildJob(ctx, ops.EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: "halo_infinite",
		MatchID:   matchID,
		Payload: &domain.BuildQueuePayload{
			MatchID: matchID, ShortID: "11111111", TitleSlug: "halo_infinite",
			MapNames: []string{"Cliffhanger"},
			Chunks: []domain.BuildQueueChunk{
				{Index: 0, ChunkType: 1, URL: "https://cdn.example/signed/header.bin"},
				{Index: 1, ChunkType: 2, URL: "https://cdn.example/signed/rep01.bin"},
			},
		},
	})
	if err != nil || !created {
		t.Fatalf("mise en file: err=%v created=%v", err, created)
	}

	view := e2eAdminView(t, srv)
	if view.Counts.Queued != 1 {
		t.Fatalf("après mise en file : queued=%d, attendu 1 (%+v)", view.Counts.Queued, view.Counts)
	}

	// ── 2. L'ouvrier prend le job par le protocole HTTP ──────────────────────
	var claim BuildQueueClaimResponse
	if code := e2ePost(t, srv, "/internal/build-queue/claim", e2eWorkerToken,
		BuildQueueClaimRequest{WorkerID: "ouvrier-e2e", Hostname: "poste-test", Version: "test/1"},
		&claim); code != http.StatusOK {
		t.Fatalf("claim: status %d", code)
	}
	if claim.Job == nil || claim.Job.JobID != job.JobID {
		t.Fatalf("claim: job rendu %+v, attendu %s", claim.Job, job.JobID)
	}
	if claim.Job.Payload == nil || len(claim.Job.Payload.Chunks) != 2 {
		t.Fatal("claim: l'ouvrier doit recevoir les URL pré-signées, sinon il n'a rien à télécharger")
	}
	if claim.Job.Payload.Chunks[0].URL == "" {
		t.Fatal("claim: URL pré-signée vide")
	}

	view = e2eAdminView(t, srv)
	if view.Counts.Running != 1 || view.Counts.Queued != 0 {
		t.Fatalf("après prise : %+v, attendu 1 en cours / 0 en attente", view.Counts)
	}
	if len(view.Jobs) == 0 || view.Jobs[0].WorkerID != "ouvrier-e2e" {
		t.Fatalf("la vue admin doit nommer l'ouvrier qui traite : %+v", view.Jobs)
	}
	if view.Jobs[0].Payload != nil {
		t.Fatal("la vue admin ne doit pas porter les URL pré-signées")
	}

	// ── 3. L'ouvrier bat pendant son travail ─────────────────────────────────
	if code := e2ePost(t, srv, "/internal/build-queue/heartbeat", e2eWorkerToken,
		BuildQueueHeartbeatRequest{
			WorkerID: "ouvrier-e2e", JobID: job.JobID, Note: "décodage en cours", JobsDone: 0,
		}, nil); code != http.StatusOK {
		t.Fatalf("heartbeat: status %d", code)
	}

	view = e2eAdminView(t, srv)
	if len(view.Workers) != 1 || !view.Workers[0].Online {
		t.Fatalf("l'ouvrier doit apparaître vivant : %+v", view.Workers)
	}
	if view.Workers[0].CurrentJobID != job.JobID {
		t.Fatalf("travail courant de l'ouvrier = %q, attendu %s", view.Workers[0].CurrentJobID, job.JobID)
	}

	// ── 4. L'ouvrier rend son résultat ───────────────────────────────────────
	if code := e2ePost(t, srv, "/internal/build-queue/complete", e2eWorkerToken,
		BuildQueueCompleteRequest{
			JobID: job.JobID, WorkerID: "ouvrier-e2e", Succeeded: true,
			ResultJSON: `{"tracks":8,"bytes":2100000}`,
		}, nil); code != http.StatusOK {
		t.Fatalf("complete: status %d", code)
	}

	view = e2eAdminView(t, srv)
	if view.Counts.Succeeded != 1 || view.Counts.Running != 0 {
		t.Fatalf("après rendu : %+v, attendu 1 fait / 0 en cours", view.Counts)
	}
	if len(view.Jobs) == 0 || view.Jobs[0].ResultJSON != `{"tracks":8,"bytes":2100000}` {
		t.Fatalf("le résultat doit être visible côté admin : %+v", view.Jobs)
	}
}

// TestBuildQueue_DeBoutEnBout_SurvitAuRedemarrage : la file est DURABLE. On ferme
// le store (le serveur meurt), on le rouvre sur le même fichier, et le job en
// attente est toujours là — c'est précisément ce que le ring mémoire du JobStore
// ne sait pas faire.
func TestBuildQueue_DeBoutEnBout_SurvitAuRedemarrage(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "monitoring.duckdb")

	st, err := ops.NewMonitoringStore(ctx, path)
	if err != nil {
		t.Fatalf("ouverture: %v", err)
	}
	const matchID = "99999999-8888-7777-6666-555555555555"
	job, _, err := st.EnqueueBuildJob(ctx, ops.EnqueueBuildJobRequest{
		JobType: string(domain.JobTypeReplayBuild), TitleSlug: "halo_infinite", MatchID: matchID,
		Payload: &domain.BuildQueuePayload{MatchID: matchID, ShortID: "99999999"},
	})
	if err != nil {
		t.Fatalf("mise en file: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("fermeture: %v", err)
	}

	reopened, err := ops.NewMonitoringStore(ctx, path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	view, err := reopened.BuildQueueReport(ctx, 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if view.Counts.Queued != 1 {
		t.Fatalf("après redémarrage : queued=%d, attendu 1 — la file a perdu son travail", view.Counts.Queued)
	}
	claimed, err := reopened.ClaimBuildJob(ctx, "ouvrier-apres-reboot", "", "")
	if err != nil || claimed == nil {
		t.Fatalf("prise après redémarrage: err=%v job=%v", err, claimed)
	}
	if claimed.JobID != job.JobID {
		t.Fatalf("job repris = %s, attendu %s", claimed.JobID, job.JobID)
	}
}
