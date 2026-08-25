package main

// memlimit_test.go — le plafond mémoire par job, et le motif explicite qu'il produit.
//
// NOTE (même parti pris que cmd/levelup/backfill_memlimit_test.go) : ces tests n'arment
// jamais un vrai plafond avec les seuils par défaut, et aucun n'appelle
// reportMemoryExceeded (qui se termine par os.Exit) — l'armer pour de bon avec les seuils
// réels, ou appeler la fonction qui sort, donnerait à un test le pouvoir de tuer le binaire
// de test. On teste donc les pièces : le déclenchement déterministe via newMemoryGuard (seuil
// minuscule, réponse en quelques millisecondes) et le compte rendu HTTP via
// completeMemoryExceeded (qui n'appelle jamais os.Exit).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
)

func TestMemGuardMargeDure(t *testing.T) {
	// Le plafond dur se pose 25 % au-dessus du souple : sous cette marge, le GC a le droit
	// de travailler dur sans être abattu — c'est son rôle.
	if got := memGuardMargeDure(4 * memGuardOctetsParGiB); got != 5*memGuardOctetsParGiB {
		t.Fatalf("memGuardMargeDure(4 GiB) = %d, veut 5 GiB", got)
	}
	if got := memGuardMargeDure(0); got != 0 {
		t.Fatalf("memGuardMargeDure(0) = %d, veut 0 (désarmé reste désarmé)", got)
	}
}

// TestMemGuardEmpreinte_Mesure : la mesure doit rendre un chiffre plausible, sinon la
// sentinelle surveillerait le vide et ne couperait jamais.
func TestMemGuardEmpreinte_Mesure(t *testing.T) {
	v := memGuardEmpreinte()
	if v == 0 {
		t.Fatal("memGuardEmpreinte() = 0 — les compteurs runtime ne répondent pas")
	}
	if v < 128*1024 {
		t.Fatalf("memGuardEmpreinte() = %d octets — invraisemblablement bas", v)
	}
}

func TestMemoryGuard_NotePic(t *testing.T) {
	g := &memoryGuard{stop: make(chan struct{})}
	for _, v := range []uint64{10, 500, 42, 500, 3} {
		g.noterPic(v)
	}
	if got := g.pic.Load(); got != 500 {
		t.Fatalf("pic = %d, veut 500 (le maximum, jamais la dernière valeur)", got)
	}
}

// TestArmMemoryGuard_Desarme : giB <= 0 est l'échappatoire de l'opérateur (mesurer un
// film-bombe sans coupure). Elle doit désarmer la COUPURE : onExceeded ne doit jamais être
// invoqué, quel que soit le nombre d'échantillons pris.
func TestArmMemoryGuard_Desarme(t *testing.T) {
	g := armMemoryGuard(0, func(uint64) {
		t.Fatal("onExceeded appelé alors que le plafond est désarmé (giB<=0)")
	})
	defer g.disarm()
	if g.plafondDur != 0 {
		t.Fatalf("plafondDur = %d, veut 0 (giB<=0 désarme la coupure)", g.plafondDur)
	}
}

// TestNewMemoryGuard_DeclencheAuDelaDuPlafondDur : avec un plafond dur d'UN octet, la toute
// première mesure (empreinte réelle du processus de test, forcément > 1 octet) doit
// déclencher onExceeded. Déterministe : pas besoin d'allouer des gigaoctets pour tester la
// coupure, seulement de poser un seuil qu'un processus vivant franchit trivialement.
func TestNewMemoryGuard_DeclencheAuDelaDuPlafondDur(t *testing.T) {
	declenche := make(chan uint64, 1)
	g := newMemoryGuard(1, 5*time.Millisecond, func(peak uint64) {
		declenche <- peak
	})
	defer g.disarm()

	select {
	case peak := <-declenche:
		if peak == 0 {
			t.Fatal("le pic rendu au déclenchement est 0")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onExceeded jamais appelé malgré un plafond dur d'un octet")
	}
}

// TestNewMemoryGuard_DisarmAvantDeclenchementNAppelleJamais : un job qui se termine
// normalement AVANT tout dépassement ne doit jamais voir onExceeded s'exécuter après coup —
// sans quoi un job réussi pourrait être suivi d'un rapport d'échec fantôme.
func TestNewMemoryGuard_DisarmAvantDeclenchementNAppelleJamais(t *testing.T) {
	appele := false
	// Plafond minuscule mais période large : disarm() doit gagner la course contre le
	// premier tick, exactement comme processJob désarme dès que buildAndSend revient.
	g := newMemoryGuard(1, time.Hour, func(uint64) { appele = true })
	g.disarm()
	time.Sleep(20 * time.Millisecond)
	if appele {
		t.Fatal("onExceeded appelé après disarm()")
	}
}

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
		memLimitGiB: memGuardDefaultGiB, // plafond réel, jamais approché par cet échec instantané
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
