package main

// protocol_test.go — le CLIENT du protocole ouvrier, appel par appel, contre un faux serveur
// de file (httptest). On prouve : la prise (job rendu / file vide / jeton refusé), le
// battement (corps posté exact + en-tête d'autorisation), et le dépôt d'artefact (corps =
// artefact brut, identité en query, accusé décodé, refus et réponse illisible mappés).
//
// complete/post sont déjà couverts par memlimit_test.go (completeMemoryExceeded, processJob) —
// on ne les re-teste pas ici.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
)

func TestProtocolClaim_RendLeJob(t *testing.T) {
	job := &domain.BuildQueueJob{JobID: "job-1", MatchID: "m-1", Status: domain.JobStatusRunning}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/build-queue/claim" {
			t.Fatalf("route inattendue: %s", r.URL.Path)
		}
		var req handlers.BuildQueueClaimRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("décodage de la requête de claim: %v", err)
		}
		if req.WorkerID != "w-1" || req.Hostname != "host" || req.Version != "ver" {
			t.Fatalf("identité d'ouvrier postée = %+v", req)
		}
		_ = json.NewEncoder(w).Encode(handlers.BuildQueueClaimResponse{Job: job, LeaseSeconds: 300})
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "tok")
	resp, err := c.claim(context.Background(), workerIdentity{workerID: "w-1", hostname: "host", version: "ver"})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if resp.Job == nil || resp.Job.JobID != "job-1" {
		t.Fatalf("resp.Job = %+v, veut job-1", resp.Job)
	}
}

func TestProtocolClaim_FileVide_NEstPasUneErreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(handlers.BuildQueueClaimResponse{}) // Job nil = file vide
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "tok")
	resp, err := c.claim(context.Background(), workerIdentity{workerID: "w"})
	if err != nil {
		t.Fatalf("une file vide ne doit pas être une erreur, a: %v", err)
	}
	if resp.Job != nil {
		t.Fatalf("file vide devrait rendre Job nil, a %+v", resp.Job)
	}
}

func TestProtocolClaim_JetonRefuse_Erreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "jeton refusé", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "mauvais")
	if _, err := c.claim(context.Background(), workerIdentity{workerID: "w"}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, veut la mention 401", err)
	}
}

func TestProtocolHeartbeat_PosteLeCorpsExact(t *testing.T) {
	var got handlers.BuildQueueHeartbeatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/build-queue/heartbeat" {
			t.Fatalf("route inattendue: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer tok" {
			t.Fatalf("Authorization = %q, veut 'Bearer tok'", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("décodage du battement: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "tok")
	err := c.heartbeat(context.Background(),
		workerIdentity{workerID: "w-1", hostname: "host", version: "ver"},
		heartbeat{jobID: "job-9", note: "décodage en cours", done: 3, failed: 1})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if got.WorkerID != "w-1" || got.JobID != "job-9" || got.Note != "décodage en cours" ||
		got.JobsDone != 3 || got.JobsFailed != 1 {
		t.Fatalf("corps du battement posté = %+v", got)
	}
}

func TestProtocolSendArtifact_Succes_CorpsEstLArtefact(t *testing.T) {
	blob := []byte(`{"schemaVersion":1,"matchId":"m-1"}`)
	var gotBody []byte
	var gotJobID, gotWorkerID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/build-queue/artifact" {
			t.Fatalf("route inattendue: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatal("en-tête d'autorisation absent")
		}
		gotJobID = r.URL.Query().Get("job_id")
		gotWorkerID = r.URL.Query().Get("worker_id")
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(domain.BuildArtifactReceipt{
			JobID: "job-1", MatchID: "m-1", Bytes: len(blob), SchemaVersion: 1,
		})
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "tok")
	receipt, err := c.sendArtifact(context.Background(), "job-1", "w-1", blob)
	if err != nil {
		t.Fatalf("sendArtifact: %v", err)
	}
	if receipt.Bytes != len(blob) || receipt.SchemaVersion != 1 {
		t.Fatalf("accusé = %+v", receipt)
	}
	if !bytes.Equal(gotBody, blob) {
		t.Fatal("le corps reçu par le serveur n'est pas l'artefact brut envoyé")
	}
	if gotJobID != "job-1" || gotWorkerID != "w-1" {
		t.Fatalf("identité en query = (job=%q, worker=%q)", gotJobID, gotWorkerID)
	}
}

func TestProtocolSendArtifact_Refus_HTTPMappe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "schéma périmé", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "tok")
	if _, err := c.sendArtifact(context.Background(), "job-1", "w-1", []byte("{}")); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v, veut la mention 400", err)
	}
}

func TestProtocolSendArtifact_ReponseIllisible_Erreur(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{pas du json")) // 200 mais accusé illisible
	}))
	defer srv.Close()

	c := newProtocolClient(srv.URL, "tok")
	if _, err := c.sendArtifact(context.Background(), "job-1", "w-1", []byte("{}")); err == nil || !strings.Contains(err.Error(), "décodage de l'accusé") {
		t.Fatalf("err = %v, veut 'décodage de l'accusé'", err)
	}
}
