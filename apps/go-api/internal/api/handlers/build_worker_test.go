// Package handlers — build_worker_test.go : contrats HTTP du protocole ouvrier.
//
// Ce qui est vérifié ici est la SURFACE DE SÉCURITÉ, parce que c'est elle qui est
// exposée sans session : sans jeton configuré la porte est fermée (503), un
// mauvais jeton est refusé (401), et un résultat périmé est refusé (409) au lieu
// d'écraser le travail de l'ouvrier qui a repris le job.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
)

// serveBuildWorker monte h sous /internal (point de montage de server_apiv1.go).
func serveBuildWorker(h *BuildWorkerHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/internal", func(r chi.Router) { h.Mount(r) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// workerReq construit un POST authentifié du protocole ouvrier.
func workerReq(path, token, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/internal/build-queue/"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// okWorkerHandler : handler câblé sur des stubs qui répondent toujours oui.
func okWorkerHandler(token string) *BuildWorkerHandler {
	return NewBuildWorkerHandler(token,
		func(_ context.Context, workerID, _, _ string) (*domain.BuildQueueJob, error) {
			return &domain.BuildQueueJob{
				JobID: "job_test", JobType: string(domain.JobTypeReplayBuild),
				Status: domain.JobStatusRunning, WorkerID: workerID,
				MatchID: "aaaa-bbbb", Payload: &domain.BuildQueuePayload{
					MatchID: "aaaa-bbbb",
					Chunks:  []domain.BuildQueueChunk{{Index: 0, URL: "https://cdn.example/signed"}},
				},
			}, nil
		},
		func(_ context.Context, _ ops.CompleteBuildJobRequest) error { return nil },
		func(_ context.Context, _ ops.HeartbeatRequest) error { return nil },
	)
}

// validWorkerBodies : un corps conforme au schéma de chaque route. Le refus doit
// être prononcé par le JETON, pas par la validation de schéma — sinon le test ne
// prouverait rien sur la porte.
var validWorkerBodies = map[string]string{
	"claim":     `{"worker_id":"w1"}`,
	"complete":  `{"job_id":"j","worker_id":"w1","succeeded":true}`,
	"heartbeat": `{"worker_id":"w1"}`,
}

// TestBuildWorker_SansJetonConfigure_PorteFermee : une instance qui n'a pas
// configuré de jeton n'expose PAS le protocole. Le dépôt est public — une
// installation par défaut ne doit hériter d'aucune porte ouverte.
func TestBuildWorker_SansJetonConfigure_PorteFermee(t *testing.T) {
	h := okWorkerHandler("")
	for path, body := range validWorkerBodies {
		rec := serveBuildWorker(h, workerReq(path, "n-importe-quoi", body))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s : status = %d (attendu 503) body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

// TestBuildWorker_JetonInvalide : mauvais jeton (ou aucun) → 401 sur les trois
// routes, AVANT toute lecture du corps (un appelant non authentifié ne doit pas
// obtenir la description du schéma attendu).
func TestBuildWorker_JetonInvalide(t *testing.T) {
	h := okWorkerHandler("le-bon-jeton")
	for path, body := range validWorkerBodies {
		for name, token := range map[string]string{"mauvais": "le-mauvais-jeton", "absent": ""} {
			rec := serveBuildWorker(h, workerReq(path, token, body))
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s/%s : status = %d (attendu 401) body=%s", path, name, rec.Code, rec.Body.String())
			}
		}
		// Corps volontairement illisible : le verdict doit rester 401, preuve que
		// le jeton est vérifié avant que le corps ne soit seulement regardé.
		rec := serveBuildWorker(h, workerReq(path, "le-mauvais-jeton", `{pas du json`))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s/corps illisible : status = %d (attendu 401) body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

// TestBuildWorker_Claim_RendLeTravailResolu : la prise sert les URL pré-signées.
// C'est la propriété qui dispense l'ouvrier de tout secret Halo — si elle tombe,
// le découpage entier n'a plus de raison d'être.
func TestBuildWorker_Claim_RendLeTravailResolu(t *testing.T) {
	rec := serveBuildWorker(okWorkerHandler("jeton"), workerReq("claim", "jeton", `{"worker_id":"ouvrier-1"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var resp BuildQueueClaimResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("décodage: %v (body=%s)", err, rec.Body.String())
	}
	if resp.Job == nil {
		t.Fatal("aucun job rendu")
	}
	if resp.Job.Payload == nil || len(resp.Job.Payload.Chunks) == 0 || resp.Job.Payload.Chunks[0].URL == "" {
		t.Fatalf("URL pré-signées absentes de la prise : %+v", resp.Job.Payload)
	}
	if resp.LeaseSeconds != int(domain.BuildJobLeaseDuration.Seconds()) {
		t.Errorf("lease_seconds = %d, attendu %d", resp.LeaseSeconds, int(domain.BuildJobLeaseDuration.Seconds()))
	}
}

// TestBuildWorker_ClaimFileVide_200SansJob : file vide = 200 avec un corps sans
// job, PAS une erreur. Un ouvrier au repos est le cas nominal.
func TestBuildWorker_ClaimFileVide_200SansJob(t *testing.T) {
	h := NewBuildWorkerHandler("jeton",
		func(_ context.Context, _, _, _ string) (*domain.BuildQueueJob, error) { return nil, nil },
		func(_ context.Context, _ ops.CompleteBuildJobRequest) error { return nil },
		func(_ context.Context, _ ops.HeartbeatRequest) error { return nil })
	rec := serveBuildWorker(h, workerReq("claim", "jeton", `{"worker_id":"ouvrier-1"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var resp BuildQueueClaimResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("décodage: %v", err)
	}
	if resp.Job != nil {
		t.Fatalf("job rendu alors que la file est vide : %+v", resp.Job)
	}
}

// TestBuildWorker_ResultatPerime_409 : un ouvrier dont le bail a expiré et dont
// le job a été repris se voit refuser son rendu — sinon il écraserait le travail
// de son remplaçant. 409 et pas 400 : il n'a rien fait de mal, il est périmé.
func TestBuildWorker_ResultatPerime_409(t *testing.T) {
	h := NewBuildWorkerHandler("jeton",
		func(_ context.Context, _, _, _ string) (*domain.BuildQueueJob, error) { return nil, nil },
		func(_ context.Context, _ ops.CompleteBuildJobRequest) error {
			return fmt.Errorf("bail expiré: %w", ops.ErrBuildJobNotClaimed)
		},
		func(_ context.Context, _ ops.HeartbeatRequest) error { return nil })
	rec := serveBuildWorker(h, workerReq("complete", "jeton",
		`{"job_id":"job_test","worker_id":"ouvrier-mort","succeeded":true}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d (attendu 409) body=%s", rec.Code, rec.Body.String())
	}
}

// TestBuildWorker_ChampsRequis_400 : worker_id (et job_id pour /complete) requis.
func TestBuildWorker_ChampsRequis_400(t *testing.T) {
	h := okWorkerHandler("jeton")
	// worker_id/job_id VIDES (et non absents) : un champ absent est déjà rejeté en
	// 422 par le schéma Huma ; c'est la garde applicative qu'on vérifie ici.
	cases := map[string]struct{ path, body string }{
		"claim worker_id vide":     {"claim", `{"worker_id":"  "}`},
		"heartbeat worker_id vide": {"heartbeat", `{"worker_id":""}`},
		"complete job_id vide":     {"complete", `{"job_id":"","worker_id":"w1","succeeded":true}`},
	}
	for name, c := range cases {
		rec := serveBuildWorker(h, workerReq(c.path, "jeton", c.body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s : status = %d (attendu 400) body=%s", name, rec.Code, rec.Body.String())
		}
	}
}
