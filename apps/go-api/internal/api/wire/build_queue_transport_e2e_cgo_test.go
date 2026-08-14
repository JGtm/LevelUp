//go:build cgo

// Package api — build_queue_transport_e2e_cgo_test.go : LA PREUVE DU TRANSPORT.
//
// Tout est réel : les routes montées comme en production (MountBuildWorkerRoutes),
// un vrai serveur HTTP, une vraie base DuckDB, le vrai rangement d'artefact, et le
// vrai service de lecture au bout. Rien n'est simulé entre les deux bouts — c'est
// ce qui distingue ce test des contrats HTTP du package handlers, qui stubent le
// store.
//
// Ce qui est prouvé, dans l'ordre du plan :
//
//	mise en file → prise → ARTEFACT → compte rendu → l'artefact est lisible par le
//	service de rejeu, À L'OCTET IDENTIQUE à celui qui a été envoyé.
//
// Et les refus, qui comptent autant : job d'un autre ouvrier, artefact trop gros,
// JSON invalide, mauvais match, schéma périmé, succès annoncé sans artefact.
//
// Driver DuckDB requis (tag cgo).
package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/service"
)

const (
	transportToken   = "jeton-transport-de-test"
	transportWorker  = "ouvrier-transport"
	transportMatchID = "12341234-5678-4abc-9def-0123456789ab"
)

// transportStack monte le protocole ouvrier COMME EN PRODUCTION sur un registre
// réel (store monitoring DuckDB + dépôt temporaire).
func transportStack(t *testing.T) (*httptest.Server, *ServiceRegistry, string) {
	t.Helper()
	repoRoot := t.TempDir()
	st, err := ops.NewMonitoringStore(context.Background(), filepath.Join(t.TempDir(), "monitoring.duckdb"))
	if err != nil {
		t.Fatalf("NewMonitoringStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	reg := &ServiceRegistry{
		cfg:             &config.AppConfig{RepoRoot: repoRoot, BuildWorkerToken: transportToken},
		monitoringStore: st,
	}
	r := chi.NewRouter()
	r.Route("/internal", func(r chi.Router) { MountBuildWorkerRoutes(r, reg, nil) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, reg, repoRoot
}

// transportDoc fabrique un artefact VALIDE pour un match (une trajectoire suffit :
// ce qui est testé est le transport, pas le décodage).
func transportDoc(matchID string) replay.ReplayDocument {
	return replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       matchID,
		TitleSlug:     titlePkg.DefaultSlug,
		FrameCount:    2,
		Bounds:        replay.Bounds{MinX: 0, MaxX: 10, MinY: 0, MaxY: 10},
		Tracks: []replay.Track{{
			XUID:   "2533274000000001",
			Points: []replay.Point{{T: 0, X: 1, Y: 1}, {T: 1, X: 2, Y: 2}},
		}},
	}
}

// enqueueTransportJob met un job de rejeu en file, sans passer par la résolution
// de manifeste (qui exigerait des tokens Halo — hors sujet ici).
func enqueueTransportJob(t *testing.T, reg *ServiceRegistry, matchID string) domain.BuildQueueJob {
	t.Helper()
	job, created, err := reg.monitoringStore.EnqueueBuildJob(context.Background(), ops.EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: titlePkg.DefaultSlug,
		MatchID:   matchID,
		Payload:   &domain.BuildQueuePayload{MatchID: matchID, ShortID: titlePkg.FilmShortMatchID(matchID)},
	})
	if err != nil || !created {
		t.Fatalf("mise en file: err=%v created=%v", err, created)
	}
	return job
}

// postJSON envoie un appel du protocole et rend (status, corps).
func postJSON(t *testing.T, srv *httptest.Server, path string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("requête %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+transportToken)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("appel %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, payload
}

// claimTransportJob prend le job par le protocole HTTP.
func claimTransportJob(t *testing.T, srv *httptest.Server, workerID string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"worker_id": workerID})
	if code, payload := postJSON(t, srv, "/internal/build-queue/claim", body); code != http.StatusOK {
		t.Fatalf("claim: status %d body=%s", code, payload)
	}
}

// artifactPath compose l'URL de dépôt d'un artefact.
func artifactPath(jobID, workerID string) string {
	return fmt.Sprintf("/internal/build-queue/artifact?job_id=%s&worker_id=%s", jobID, workerID)
}

// TestTransport_DeBoutEnBout : le chemin complet, et l'octet-pour-octet au bout.
func TestTransport_DeBoutEnBout(t *testing.T) {
	srv, reg, repoRoot := transportStack(t)
	job := enqueueTransportJob(t, reg, transportMatchID)
	claimTransportJob(t, srv, transportWorker)

	// ── Le compte rendu de succès AVANT l'artefact est refusé ────────────────
	// « Le compte rendu ne ment jamais sur la présence du fichier » : sans
	// artefact, il n'y a rien à déclarer réussi.
	complete, _ := json.Marshal(map[string]any{
		"job_id": job.JobID, "worker_id": transportWorker, "succeeded": true,
	})
	if code, payload := postJSON(t, srv, "/internal/build-queue/complete", complete); code != http.StatusConflict {
		t.Fatalf("succès annoncé sans artefact : status %d (attendu 409) body=%s", code, payload)
	}

	// ── L'ouvrier pousse son artefact ────────────────────────────────────────
	sent, err := json.Marshal(transportDoc(transportMatchID))
	if err != nil {
		t.Fatalf("sérialisation de l'artefact: %v", err)
	}
	code, payload := postJSON(t, srv, artifactPath(job.JobID, transportWorker), sent)
	if code != http.StatusOK {
		t.Fatalf("dépôt d'artefact : status %d body=%s", code, payload)
	}
	var receipt domain.BuildArtifactReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		t.Fatalf("décodage de l'accusé: %v (body=%s)", err, payload)
	}
	if receipt.Bytes != len(sent) || receipt.SchemaVersion != replay.SchemaVersion {
		t.Fatalf("accusé = %+v, attendu %d octets / schéma %d", receipt, len(sent), replay.SchemaVersion)
	}

	// ── À L'OCTET IDENTIQUE, à la place canonique ────────────────────────────
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, transportMatchID)
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artefact absent du disque (%s): %v", path, err)
	}
	if !bytes.Equal(onDisk, sent) {
		t.Fatalf("artefact rangé ≠ artefact envoyé (%d octets vs %d)", len(onDisk), len(sent))
	}

	// ── Le service de rejeu le sert ──────────────────────────────────────────
	doc, err := service.NewReplayService(titlePkg.DefaultSlug, repoRoot, nil).GetReplay(context.Background(), transportMatchID)
	if err != nil {
		t.Fatalf("le service de rejeu ne lit pas l'artefact reçu: %v", err)
	}
	if doc.MatchID != transportMatchID || len(doc.Tracks) != 1 {
		t.Fatalf("document servi = match %q / %d trajectoires", doc.MatchID, len(doc.Tracks))
	}

	// ── Et MAINTENANT le compte rendu passe ──────────────────────────────────
	if code, payload := postJSON(t, srv, "/internal/build-queue/complete", complete); code != http.StatusOK {
		t.Fatalf("compte rendu après artefact : status %d body=%s", code, payload)
	}
	view, err := reg.monitoringStore.BuildQueueReport(context.Background(), 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if view.Counts.Succeeded != 1 {
		t.Fatalf("après rendu : %+v, attendu 1 fait", view.Counts)
	}
}

// TestTransport_Refus : tout ce qui ne doit RIEN écrire sur le disque.
func TestTransport_Refus(t *testing.T) {
	srv, reg, repoRoot := transportStack(t)
	job := enqueueTransportJob(t, reg, transportMatchID)
	claimTransportJob(t, srv, transportWorker)

	valide, _ := json.Marshal(transportDoc(transportMatchID))
	autreMatch, _ := json.Marshal(transportDoc("99999999-0000-4000-8000-000000000000"))
	perime := transportDoc(transportMatchID)
	perime.SchemaVersion = replay.SchemaVersion - 1
	schemaPerime, _ := json.Marshal(perime)
	sansTrace := transportDoc(transportMatchID)
	sansTrace.Tracks = nil
	sansTrajectoire, _ := json.Marshal(sansTrace)

	cases := []struct {
		nom    string
		path   string
		body   []byte
		attend int
	}{
		{"job d'un autre ouvrier", artifactPath(job.JobID, "ouvrier-etranger"), valide, http.StatusConflict},
		{"job inconnu", artifactPath("job-qui-n-existe-pas", transportWorker), valide, http.StatusConflict},
		{"identité absente", "/internal/build-queue/artifact", valide, http.StatusBadRequest},
		{"artefact trop gros", artifactPath(job.JobID, transportWorker),
			bytes.Repeat([]byte("a"), domain.MaxBuildArtifactBytes+1), http.StatusRequestEntityTooLarge},
		{"JSON invalide", artifactPath(job.JobID, transportWorker), []byte(`{ ceci n'est pas du json`), http.StatusBadRequest},
		{"mauvais match", artifactPath(job.JobID, transportWorker), autreMatch, http.StatusBadRequest},
		{"schéma périmé", artifactPath(job.JobID, transportWorker), schemaPerime, http.StatusBadRequest},
		{"sans trajectoire", artifactPath(job.JobID, transportWorker), sansTrajectoire, http.StatusBadRequest},
	}
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, transportMatchID)
	for _, c := range cases {
		code, payload := postJSON(t, srv, c.path, c.body)
		if code != c.attend {
			t.Errorf("%s : status %d (attendu %d) body=%s", c.nom, code, c.attend, payload)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s : un artefact a été écrit alors que le dépôt est refusé (%s)", c.nom, path)
		}
	}
}
