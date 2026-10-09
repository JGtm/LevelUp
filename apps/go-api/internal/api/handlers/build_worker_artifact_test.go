// Package handlers — build_worker_artifact_test.go : contrat HTTP du dépôt
// d'artefact. La preuve de bout en bout (rangement réel, octet pour octet, refus
// par motif) vit au plus près du vrai câblage, dans internal/api/wire.
//
// Ce qui est vérifié ICI est la surface exposée sans session : la porte est
// fermée comme les autres routes du protocole, l'identité du job est exigée, et
// une instance sans rangement câblé répond 503 plutôt que d'avaler des octets
// qu'elle ne saurait pas ranger.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
)

// artifactReq construit un POST de dépôt d'artefact.
func artifactReq(query, token, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/internal/build-queue/artifact"+query, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// artifactHandler : protocole complet, avec un rangement qui répond err.
func artifactHandler(token string, err error) *BuildWorkerHandler {
	return okWorkerHandler(token).WithArtifactStore(
		func(_ context.Context, jobID, _ string, blob []byte) (domain.BuildArtifactReceipt, error) {
			if err != nil {
				return domain.BuildArtifactReceipt{}, err
			}
			return domain.BuildArtifactReceipt{JobID: jobID, Bytes: len(blob), SchemaVersion: 5}, nil
		})
}

const artifactQuery = "?job_id=job_test&worker_id=ouvrier-1"

// TestArtifact_PorteFermee : même porte que le reste du protocole — sans jeton
// configuré 503, avec un mauvais jeton 401, et dans les deux cas AVANT que le
// corps (2 Mo) ne soit seulement lu.
func TestArtifact_PorteFermee(t *testing.T) {
	body := `{"schemaVersion":5,"matchId":"m"}`
	rec := serveBuildWorker(artifactHandler("", nil), artifactReq(artifactQuery, "peu importe", body))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("sans jeton configuré : status = %d (attendu 503) body=%s", rec.Code, rec.Body.String())
	}
	rec = serveBuildWorker(artifactHandler("le-bon", nil), artifactReq(artifactQuery, "le-mauvais", body))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("mauvais jeton : status = %d (attendu 401) body=%s", rec.Code, rec.Body.String())
	}
}

// TestArtifact_RangementNonCable_503 : un protocole ouvert mais sans rangement
// n'accepte pas d'artefact — il le dit, il ne le jette pas en silence.
func TestArtifact_RangementNonCable_503(t *testing.T) {
	rec := serveBuildWorker(okWorkerHandler("jeton"), artifactReq(artifactQuery, "jeton", `{"schemaVersion":5}`))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d (attendu 503) body=%s", rec.Code, rec.Body.String())
	}
}

// TestArtifact_IdentiteRequise_400 : sans job_id/worker_id, on ne sait ni de quel
// travail il s'agit ni qui l'a fait — rien ne peut être rangé.
func TestArtifact_IdentiteRequise_400(t *testing.T) {
	h := artifactHandler("jeton", nil)
	for _, q := range []string{"", "?job_id=job_test", "?worker_id=ouvrier-1", "?job_id=%20&worker_id=w"} {
		rec := serveBuildWorker(h, artifactReq(q, "jeton", `{"schemaVersion":5}`))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("query %q : status = %d (attendu 400) body=%s", q, rec.Code, rec.Body.String())
		}
	}
}

// TestArtifact_MotifsDeRefus : chaque refus du rangement a son code HTTP, et un
// travail périmé (409) ne se confond pas avec un fichier invalide (400).
func TestArtifact_MotifsDeRefus(t *testing.T) {
	cases := map[string]struct {
		err    error
		attend int
	}{
		"job d'un autre ouvrier": {fmt.Errorf("bail: %w", ops.ErrBuildJobNotClaimed), http.StatusConflict},
		"artefact invalide":      {fmt.Errorf("schéma: %w", domain.ErrBuildArtifactInvalid), http.StatusBadRequest},
		"panne de rangement":     {fmt.Errorf("disque plein"), http.StatusInternalServerError},
	}
	for name, c := range cases {
		rec := serveBuildWorker(artifactHandler("jeton", c.err), artifactReq(artifactQuery, "jeton", `{"schemaVersion":5}`))
		if rec.Code != c.attend {
			t.Errorf("%s : status = %d (attendu %d) body=%s", name, rec.Code, c.attend, rec.Body.String())
		}
	}
}

// TestArtifact_Accepte_RendLAccuse : le cas nominal rend l'accusé de dépôt —
// c'est sur lui, et lui seul, que l'ouvrier s'autorise à rendre son compte rendu.
func TestArtifact_Accepte_RendLAccuse(t *testing.T) {
	body := `{"schemaVersion":5,"matchId":"m","tracks":[]}`
	rec := serveBuildWorker(artifactHandler("jeton", nil), artifactReq(artifactQuery, "jeton", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"job_id":"job_test"`) {
		t.Errorf("accusé sans job_id : %s", rec.Body.String())
	}
}
