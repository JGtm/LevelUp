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

import (
	"errors"
	"time"
)

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

	// Facts est CE QUE LA BASE SAIT DU MATCH, embarqué avec le travail parce que
	// l'ouvrier n'a AUCUNE base pour le résoudre lui-même — c'est sa propriété de
	// sécurité, pas une lacune à combler chez lui.
	//
	// POURQUOI IL VOYAGE ICI PLUTÔT QUE D'ÊTRE RE-DÉRIVÉ OU RATTRAPÉ APRÈS COUP.
	// Re-dériver est exclu (il faudrait donner une base à l'ouvrier) ; rattraper au
	// retour imposerait au VPS web de re-cuire l'artefact, ce que la règle « le VPS
	// web ne décode JAMAIS » interdit. Reste le transport, et il est bon marché :
	// MESURÉ à 713-756 octets sur trois matchs, contre ~20 Ko d'URL pré-signées que
	// ce même payload porte déjà — environ 3 % de plus.
	//
	// CE QUE COÛTE SON ABSENCE, MESURÉ le 2026-08-24 (témoin 7344d24f, Strongholds) :
	// actions d'objectif 246 -> 0, zones du mode 3 -> 0, joueurs de la courbe de
	// score 8 -> 0, identité des camps `b` -> `unresolved`. Un artefact ainsi appauvri
	// porte le bon `schemaVersion` : sans le prédicat de fraîcheur qui regarde les
	// faits, plus rien ne le re-cuirait jamais.
	//
	// NIL VEUT DIRE « RIEN DU TOUT », PAS « RIEN D'UTILE ». L'enfileur n'attache ces
	// faits que s'ils sont non vides au sens de `MatchFacts.Empty()` — c'est-à-dire dès
	// qu'UN SEUL champ est renseigné. Un pointeur non nil ne promet donc PAS des lignes
	// de match : un match présent au registre sans aucun participant voyage avec une
	// variante et des scores, et `Players` vide.
	//
	// Un consommateur qui a besoin des COMPTEURS DE JOUEUR doit donc tester
	// `len(Facts.Players)`, jamais la seule non-nullité du pointeur.
	//
	// Nil = l'enfileur n'a résolu aucun fait (match hors registre, base indisponible).
	// C'est LÉGITIME et journalisé : l'ouvrier construit alors un artefact valide mais
	// appauvri, exactement comme le CLI hors ligne.
	Facts *MatchFacts `json:"facts,omitempty"`
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

// MaxBuildArtifactBytes borne le corps du dépôt d'artefact
// (POST /internal/build-queue/artifact).
//
// POURQUOI UN PLAFOND, ET POURQUOI CELUI-CI. Le VPS web a un disque SOUS TENSION
// (plafond de cache 5 Go, zéro swap, incidents de gel documentés) : un corps non
// borné est une porte ouverte dessus, quel que soit le jeton présenté. Les
// artefacts mesurés pèsent ~2 Mo (2,19 · 1,64 · 2,62) ; 16 Mio laisse huit fois
// la marge du plus gros connu tout en restant FINI — assez large pour qu'un match
// hors norme passe, assez court pour qu'un ouvrier fou ne remplisse rien.
const MaxBuildArtifactBytes = 16 << 20

// ErrBuildArtifactInvalid : l'artefact déposé n'est pas celui qu'on attendait
// (illisible, mauvaise version de schéma, mauvais match, sans trajectoire). RIEN
// n'est écrit sur le disque dans ce cas — l'appelant HTTP en fait un 400.
//
// C'est un refus, pas une panne : un artefact construit par un décodeur d'une
// version antérieure DOIT être refusé, pas rangé (piste F, « ce qui peut faire
// échouer ce plan » §2).
var ErrBuildArtifactInvalid = errors.New("build queue: artefact refusé")

// BuildJobErrorCodeMemoryExceeded : ErrorCode explicite d'un BuildQueueJob mort par
// dépassement du plafond mémoire dur — un film-bombe isolé par la sentinelle de son propre
// processus ouvrier (cf. cmd/replay-worker/memlimit.go, même doctrine que le plafond de
// cmd/levelup/backfill_memlimit.go : soupçon mesuré sur 51101d1d, 7,9 Go en 2,6 s).
//
// DISTINCT DU CODE GÉNÉRIQUE "replay_build_failed" : un opérateur qui lit le tableau de bord
// admin doit voir IMMÉDIATEMENT qu'un film a été isolé pour sa RAM (film-bombe connu,
// décodage jamais fautif) plutôt que de rouvrir des journaux pour distinguer ce cas d'une
// vraie erreur de décodage.
const BuildJobErrorCodeMemoryExceeded = "memory_exceeded"

// BuildArtifactReceipt : accusé de dépôt d'un artefact. Rendu à l'ouvrier pour
// qu'il sache ce que le web a effectivement rangé — c'est sur cette réponse, et
// elle seule, qu'il s'autorise à rendre son compte rendu de succès.
type BuildArtifactReceipt struct {
	JobID         string `json:"job_id"`
	MatchID       string `json:"match_id"`
	Bytes         int    `json:"bytes"`
	SchemaVersion int    `json:"schema_version"`
}
