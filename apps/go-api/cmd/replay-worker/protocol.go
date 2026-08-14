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
	"strings"
	"time"

	"levelup/go-api/internal/api/handlers"
)

// httpTimeout borne chaque appel de protocole. Généreux : le serveur prend un
// lease DuckDB pour servir une prise.
const httpTimeout = 30 * time.Second

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
