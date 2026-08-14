// Package domain — build_queue.go : contrat de la FILE DURABLE de construction et
// du PROTOCOLE OUVRIER (piste F §1/§2/§4bis du plan film/killfeed/rejeu).
//
// L'OUVRIER EST UN RÔLE, PAS UNE MACHINE. Le VPS web met en file et sert ; une
// machine quelconque (second VPS, poste de développement, ou rien du tout) tire
// le travail en HTTPS. Sans ouvrier, la file s'allonge et rien ne casse : la
// dégradation par absence d'artefact est déjà écrite (os.Stat côté service).
//
// CE QUE L'OUVRIER N'A PAS : aucun secret Halo, aucun accès base, aucun port
// entrant. Le web résout le manifeste (lui seul a les tokens) et dépose des URL
// CDN PRÉ-SIGNÉES dans le job — l'ouvrier n'est qu'un nœud de calcul. C'est cette
// propriété, et elle seule, qui rend sûr de le faire tourner n'importe où.
package domain

import "time"

// BuildQueueChunk : un morceau de film à télécharger, tel que le web l'a résolu.
// URL est PRÉ-SIGNÉE (CDN Azure, sans authentification) — c'est tout ce que
// l'ouvrier reçoit du monde Halo.
type BuildQueueChunk struct {
	Index      int    `json:"index"`
	ChunkType  int    `json:"chunk_type"`
	StartMS    int    `json:"start_ms"`
	DurationMS int    `json:"duration_ms"`
	URL        string `json:"url"`
}

// BuildQueuePayload : le travail complet d'un job, résolu côté web.
// MapNames est l'ordre de résolution de carte du rejeu (nom d'asset EN puis
// map_name brut, cf. replaybuild) — l'ouvrier ne sait pas interroger la base.
type BuildQueuePayload struct {
	MatchID   string            `json:"match_id"`
	ShortID   string            `json:"short_id"`
	TitleSlug string            `json:"title_slug"`
	MapNames  []string          `json:"map_names,omitempty"`
	Chunks    []BuildQueueChunk `json:"chunks,omitempty"`
}

// BuildQueueJob : l'état courant d'un job de la file (vue build_jobs_latest).
// Status réutilise JobStatus (queued/running/succeeded/failed/interrupted) — le
// contrat de job existait déjà, on ne l'invente pas.
type BuildQueueJob struct {
	JobID     string    `json:"job_id"`
	JobType   string    `json:"job_type"`
	TitleSlug string    `json:"title_slug,omitempty"`
	MatchID   string    `json:"match_id,omitempty"`
	Status    JobStatus `json:"status"`
	Priority  int       `json:"priority"`
	Attempt   int       `json:"attempt"`
	WorkerID  string    `json:"worker_id,omitempty"`

	LeaseExpiresAt string `json:"lease_expires_at,omitempty"` // RFC3339
	EnqueuedAt     string `json:"enqueued_at,omitempty"`      // RFC3339
	UpdatedAt      string `json:"updated_at,omitempty"`       // RFC3339

	ResultJSON   string `json:"result_json,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Payload n'est servi QU'À l'ouvrier qui prend le job (réponse de claim) :
	// l'admin n'a que faire de 20 Ko d'URL, et elles expirent.
	Payload *BuildQueuePayload `json:"payload,omitempty"`
}

// BuildQueueWorker : l'état courant d'un ouvrier (vue build_workers_latest).
// Online est calculé à la lecture (battement plus récent que le seuil) — jamais
// stocké : un état « en ligne » persisté serait faux dès la seconde suivante.
type BuildQueueWorker struct {
	WorkerID     string `json:"worker_id"`
	Hostname     string `json:"hostname,omitempty"`
	Version      string `json:"version,omitempty"`
	CurrentJobID string `json:"current_job_id,omitempty"`
	JobsDone     int64  `json:"jobs_done"`
	JobsFailed   int64  `json:"jobs_failed"`
	Note         string `json:"note,omitempty"`
	LastBeatAt   string `json:"last_beat_at,omitempty"` // RFC3339
	Online       bool   `json:"online"`
}

// BuildQueueCounts : la file d'un coup d'œil (badge admin sans parcourir la liste).
type BuildQueueCounts struct {
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// AdminBuildQueueResponse : réponse de GET /admin/monitoring/build-queue —
// le travail de construction ET les ouvriers, de bout en bout, dans l'app.
type AdminBuildQueueResponse struct {
	GeneratedAt string             `json:"generated_at"`
	Enabled     bool               `json:"enabled"` // un jeton d'ouvrier est configuré
	Counts      BuildQueueCounts   `json:"counts"`
	Jobs        []BuildQueueJob    `json:"jobs"`
	Workers     []BuildQueueWorker `json:"workers"`
}

// WorkerOfflineAfter : au-delà de ce silence, un ouvrier s'affiche hors ligne.
// Trois battements manqués à la cadence recommandée (30 s) — assez tolérant pour
// ne pas clignoter sur un décodage long, assez court pour être une information.
const WorkerOfflineAfter = 90 * time.Second

// BuildJobLeaseDuration : durée du bail posé à la prise d'un job. Au-delà, sans
// battement, le job retourne en file. Généreuse : un gros film se décode en ~50 s
// et l'ouvrier bat pendant ce temps ; le bail ne protège que contre la MORT.
const BuildJobLeaseDuration = 5 * time.Minute

// BuildJobMaxAttempts : au-delà, un job repris trop de fois part en `failed`
// plutôt que de tourner en boucle (un film corrompu ne se décodera jamais).
const BuildJobMaxAttempts = 3
