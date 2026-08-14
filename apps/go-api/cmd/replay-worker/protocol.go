package main

// protocol.go — le CLIENT du protocole ouvrier : trois POST et rien d'autre.
//
// Ce fichier est la preuve exécutable de la frontière de sécurité : il ne lit
// aucun token Halo, n'ouvre aucune base, ne connaît aucun identifiant. Tout ce
// qu'il présente au serveur, c'est le jeton d'ouvrier ; tout ce qu'il reçoit,
// c'est un travail déjà résolu (des URL CDN pré-signées).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
)

// httpTimeout borne chaque appel de protocole. Généreux : le serveur prend un
// lease DuckDB pour servir une prise.
const httpTimeout = 30 * time.Second

// artifactUploadTimeout borne l'envoi de l'artefact (~2 Mo), le seul appel du
// protocole qui transporte du volume.
const artifactUploadTimeout = 2 * time.Minute

// protocolClient parle au serveur web. Sans état : l'état vit côté web.
type protocolClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newProtocolClient(baseURL, token string) *protocolClient {
	return &protocolClient{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: httpTimeout},
	}
}

// claim demande le prochain job. (nil, nil) = file vide, cas NOMINAL.
func (c *protocolClient) claim(ctx context.Context, id workerIdentity) (*handlers.BuildQueueClaimResponse, error) {
	var out handlers.BuildQueueClaimResponse
	err := c.post(ctx, "/build-queue/claim", handlers.BuildQueueClaimRequest{
		WorkerID: id.workerID, Hostname: id.hostname, Version: id.version,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// heartbeat envoie un signe de vie (et prolonge le bail du job en cours).
func (c *protocolClient) heartbeat(ctx context.Context, id workerIdentity, beat heartbeat) error {
	return c.post(ctx, "/build-queue/heartbeat", handlers.BuildQueueHeartbeatRequest{
		WorkerID: id.workerID, Hostname: id.hostname, Version: id.version,
		JobID: beat.jobID, Note: beat.note, JobsDone: beat.done, JobsFailed: beat.failed,
	}, nil)
}

// heartbeat : ce que l'ouvrier a à dire de lui à un instant donné.
type heartbeat struct {
	jobID, note  string
	done, failed int64
}

// complete rend le résultat d'un job pris.
func (c *protocolClient) complete(ctx context.Context, req handlers.BuildQueueCompleteRequest) error {
	return c.post(ctx, "/build-queue/complete", req, nil)
}

// sendArtifact POUSSE l'artefact construit vers le web. Le corps EST l'artefact
// (pas d'enveloppe : ré-encoder 2 Mo de JSON dans une chaîne JSON coûterait le
// double des deux côtés) ; l'identité du job voyage en paramètres d'URL.
//
// C'est le seul appel du protocole qui transporte du volume, et le seul dont
// l'échec doit empêcher le compte rendu : sans artefact chez le web, le travail
// n'a pas eu lieu.
func (c *protocolClient) sendArtifact(ctx context.Context, jobID, workerID string, blob []byte) (domain.BuildArtifactReceipt, error) {
	var out domain.BuildArtifactReceipt
	path := "/build-queue/artifact?job_id=" + url.QueryEscape(jobID) + "&worker_id=" + url.QueryEscape(workerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(blob))
	if err != nil {
		return out, fmt.Errorf("requête d'artefact : %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	// Un artefact pèse ~2 Mo : le délai du protocole (30 s) suffit sur un lien
	// correct, mais l'envoi mérite sa propre marge — un ouvrier derrière une
	// liaison lente ne doit pas perdre 50 s de décodage pour 2 s de transfert.
	client := &http.Client{Timeout: artifactUploadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return out, fmt.Errorf("envoi de l'artefact : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, fmt.Errorf("lecture de la réponse d'artefact : %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("envoi de l'artefact : HTTP %d — %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, fmt.Errorf("décodage de l'accusé d'artefact : %w", err)
	}
	return out, nil
}

// post envoie un corps JSON et décode la réponse. out nil = réponse ignorée.
func (c *protocolClient) post(ctx context.Context, path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("sérialisation de la requête %s : %w", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("requête %s : %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("appel %s : %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("lecture de la réponse %s : %w", path, err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("appel %s : HTTP %d — %s", path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("décodage de la réponse %s : %w", path, err)
	}
	return nil
}
