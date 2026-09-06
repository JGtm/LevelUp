package main

// memlimit_test.go — le motif explicite que produit un dépassement du plafond mémoire par
// job.
//
// La sentinelle elle-même (armement, échantillonnage, plafond dur, Disarm) est
// internal/filmproc.Arm depuis le lot v2 G.1 (2026-09-05) — ses propres tests vivent dans
// internal/filmproc (filmproc_test.go) et ne sont pas dupliqués ici. ATTENTION : aucun test
// ci-dessous n'appelle reportMemoryExceeded (qui se termine par os.Exit) — l'appeler pour de
// bon donnerait à un test le pouvoir de tuer le binaire de test. On teste donc les pièces
// LOCALES à ce paquet : le motif explicite (memoryExceededRequest/memoryExceededMessage) et
// le compte rendu HTTP via completeMemoryExceeded (qui n'appelle jamais os.Exit).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/filmproc"
)

func TestMemoryExceededMessage_ContientMatchIDPicEtMention(t *testing.T) {
	const matchID = "51101d1d-aca8-4c95-a431-2012114b87be"
	peak := 7*uint64(memGuardOctetsParGiB) + memGuardOctetsParGiB/2 // ~7.5 GiB
	msg := memoryExceededMessage(matchID, peak)
	for _, want := range []string{matchID, "isolé", "poursuivie", "GiB"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q ne contient pas %q", msg, want)
		}
	}
}

// TestMemoryExceededMessage_PicInconnu : la sentinelle peut être tuée avant son premier
// échantillon (job mort trop vite) — le message doit le dire plutôt que d'afficher "0".
func TestMemoryExceededMessage_PicInconnu(t *testing.T) {
	msg := memoryExceededMessage("abc123", 0)
	if !strings.Contains(msg, "pic inconnu") {
		t.Fatalf("message %q devrait signaler un pic inconnu quand peakBytes=0", msg)
	}
}

func TestFormatMemGuardBytes(t *testing.T) {
	if got := formatMemGuardBytes(2 * memGuardOctetsParGiB); got != "2.00 GiB" {
		t.Fatalf("formatMemGuardBytes(2 GiB) = %q", got)
	}
	if got := formatMemGuardBytes(512 * 1024 * 1024); got != "512 MiB" {
		t.Fatalf("formatMemGuardBytes(512 MiB) = %q", got)
	}
}

// TestMemoryExceededRequest_MotifExplicite est le cœur du lot : le compte rendu d'un
// dépassement mémoire doit porter un ErrorCode DISTINCT du motif générique, et son message
// doit nommer le match. C'est cette structure, pas un "failed" anonyme, que l'admin doit
// pouvoir lire dans le tableau de bord build-queue.
func TestMemoryExceededRequest_MotifExplicite(t *testing.T) {
	job := &domain.BuildQueueJob{JobID: "job-1", MatchID: "51101d1d"}
	req := memoryExceededRequest("worker-1", job, 7*uint64(memGuardOctetsParGiB))

	if req.Succeeded {
		t.Fatal("Succeeded doit être false pour un job isolé par la sentinelle")
	}
	if req.JobID != "job-1" || req.WorkerID != "worker-1" {
		t.Fatalf("identité du compte rendu = (%q, %q), veut (job-1, worker-1)", req.JobID, req.WorkerID)
	}
	if req.ErrorCode != domain.BuildJobErrorCodeMemoryExceeded {
		t.Fatalf("ErrorCode = %q, veut %q", req.ErrorCode, domain.BuildJobErrorCodeMemoryExceeded)
	}
	if req.ErrorCode == genericBuildFailedErrorCode {
		t.Fatal("le motif mémoire ne doit jamais dégénérer vers le motif générique")
	}
	if !strings.Contains(req.ErrorMessage, "51101d1d") {
		t.Fatalf("ErrorMessage = %q, doit contenir le match_id", req.ErrorMessage)
	}
}

// TestWorker_CompleteMemoryExceeded_MotifExplicite vérifie le chemin RÉEL (HTTP compris),
// jusqu'à la limite d'os.Exit : completeMemoryExceeded doit poster au serveur un
// error_code=memory_exceeded, jamais le motif générique.
func TestWorker_CompleteMemoryExceeded_MotifExplicite(t *testing.T) {
	var recu handlers.BuildQueueCompleteRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/build-queue/complete" {
			t.Fatalf("route inattendue: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&recu); err != nil {
			t.Fatalf("décodage du corps posté: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := &worker{
		identity: workerIdentity{workerID: "worker-test"},
		client:   newProtocolClient(srv.URL, "test-token"),
	}
	job := &domain.BuildQueueJob{JobID: "job-42", MatchID: "51101d1d-aca8-4c95-a431-2012114b87be"}
	w.completeMemoryExceeded(context.Background(), job, 7*uint64(memGuardOctetsParGiB))

	if recu.ErrorCode != domain.BuildJobErrorCodeMemoryExceeded {
		t.Fatalf("error_code posté = %q, veut %q", recu.ErrorCode, domain.BuildJobErrorCodeMemoryExceeded)
	}
	if recu.Succeeded {
		t.Fatal("succeeded posté = true, veut false")
	}
	if !strings.Contains(recu.ErrorMessage, job.MatchID) {
		t.Fatalf("error_message posté = %q, doit contenir le match_id", recu.ErrorMessage)
	}
}

// TestProcessJob_EchecOrdinaire_GardeSonMotif : un job SANS travail résolu échoue tout de
// suite dans buildAndSend, sans jamais approcher le plafond mémoire (état par défaut, tests
// rapides et synchrones). Le motif posté au serveur doit rester le motif GÉNÉRIQUE — la
// preuve que le nouveau motif mémoire ne déborde pas sur les échecs ordinaires.
func TestProcessJob_EchecOrdinaire_GardeSonMotif(t *testing.T) {
	var recu handlers.BuildQueueCompleteRequest
	var vu bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/build-queue/complete":
			vu = true
			if err := json.NewDecoder(r.Body).Decode(&recu); err != nil {
				t.Fatalf("décodage du corps posté: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		case "/build-queue/heartbeat":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("route inattendue: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	w := &worker{
		identity:    workerIdentity{workerID: "worker-test"},
		client:      newProtocolClient(srv.URL, "test-token"),
		memLimitGiB: filmproc.DefaultLimitGiB, // plafond réel, jamais approché par cet échec instantané
	}
	// Payload nil : buildAndSend échoue IMMÉDIATEMENT ("job sans travail résolu"), avant
	// tout téléchargement ou décodage — un échec ordinaire, réel, pas simulé.
	job := &domain.BuildQueueJob{JobID: "job-7", MatchID: "m-ordinaire", Payload: nil}
	w.processJob(context.Background(), job)

	if !vu {
		t.Fatal("aucun compte rendu posté au serveur")
	}
	if recu.ErrorCode != genericBuildFailedErrorCode {
		t.Fatalf("error_code posté = %q, veut le motif générique %q", recu.ErrorCode, genericBuildFailedErrorCode)
	}
	if recu.ErrorCode == domain.BuildJobErrorCodeMemoryExceeded {
		t.Fatal("un échec ordinaire ne doit jamais porter le motif mémoire")
	}
}
