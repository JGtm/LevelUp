package prestige

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

// telemetry.go — émission d'événements de télémétrie pour le calage post-alpha.
//
// Référence : Annexe E du plan conceptuel.
// Ces événements sont agrégés par l'endpoint diag
// GET /api/v1/_diag/prestige/telemetry/{player_slug} (taux d'acceptation/complétion
// par origine — coach vs user vs pilot_mode, ADR 0020).
//
// Le helper expose des constructions typées pour chaque type d'événement
// (created/rejected/completed/...). Chaque événement recopie Challenge.Source
// (origine du défi). L'écriture est déléguée au TelemetryRepo.

// TelemetryEmitter encapsule un repo et un horloge pour des tests faciles.
type TelemetryEmitter struct {
	repo TelemetryRepo
	now  func() time.Time
}

// NewTelemetryEmitter crée un emitter prêt à l'emploi.
//
// nowFn est optionnel — si nil, time.Now().UTC() est utilisé.
func NewTelemetryEmitter(repo TelemetryRepo, nowFn func() time.Time) *TelemetryEmitter {
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &TelemetryEmitter{repo: repo, now: nowFn}
}

// EmitCreated trace la création d'un défi avec son palier calculé.
func (e *TelemetryEmitter) EmitCreated(ctx context.Context, c Challenge, stretch, baseline float64) {
	e.emit(ctx, PrestigeTelemetry{
		ID:            newTelemetryID(),
		UserID:        c.UserID,
		ChallengeID:   c.ID,
		EventType:     TelemetryCreated,
		Palier:        c.Tier,
		StretchRatio:  stretch,
		BaselineValue: baseline,
		Mode:          c.Mode,
		Cadence:       c.Cadence,
		EvalType:      c.EvalType,
		Source:        c.Source,
		CreatedAt:     e.now(),
	})
}

// EmitRejected trace un rejet de défi avec sa raison.
func (e *TelemetryEmitter) EmitRejected(ctx context.Context, userID string, c Challenge, stretch, baseline float64, reason RejectReason) {
	e.emit(ctx, PrestigeTelemetry{
		ID:            newTelemetryID(),
		UserID:        userID,
		ChallengeID:   c.ID,
		EventType:     TelemetryRejected + ":" + string(reason),
		StretchRatio:  stretch,
		BaselineValue: baseline,
		Mode:          c.Mode,
		Cadence:       c.Cadence,
		EvalType:      c.EvalType,
		Source:        c.Source,
		CreatedAt:     e.now(),
	})
}

// EmitTransition trace une transition de statut (committed/completed/expired/abandoned).
func (e *TelemetryEmitter) EmitTransition(ctx context.Context, c Challenge, eventType string) {
	tsc := 0
	if c.CommittedAt != nil {
		tsc = int(e.now().Sub(*c.CommittedAt).Seconds())
	}
	e.emit(ctx, PrestigeTelemetry{
		ID:                     newTelemetryID(),
		UserID:                 c.UserID,
		ChallengeID:            c.ID,
		EventType:              eventType,
		Palier:                 c.Tier,
		Mode:                   c.Mode,
		Cadence:                c.Cadence,
		EvalType:               c.EvalType,
		TimeSinceCreateSeconds: tsc,
		Source:                 c.Source,
		CreatedAt:              e.now(),
	})
}

// EmitPalierRecomputed trace un recalcul de palier (édition de cible mode libre).
func (e *TelemetryEmitter) EmitPalierRecomputed(ctx context.Context, c Challenge, oldTier Tier, stretch, baseline float64) {
	e.emit(ctx, PrestigeTelemetry{
		ID:            newTelemetryID(),
		UserID:        c.UserID,
		ChallengeID:   c.ID,
		EventType:     TelemetryPalierRecomputed,
		Palier:        c.Tier,
		StretchRatio:  stretch,
		BaselineValue: baseline,
		Mode:          c.Mode,
		Cadence:       c.Cadence,
		EvalType:      c.EvalType,
		Source:        c.Source,
		CreatedAt:     e.now(),
	})
}

// emit écrit l'événement et log l'erreur sans interrompre l'appelant.
//
// La télémétrie est best-effort : un échec d'écriture ne doit pas casser
// le flux principal (création de défi, sync, etc.).
func (e *TelemetryEmitter) emit(ctx context.Context, ev PrestigeTelemetry) {
	if e == nil || e.repo == nil {
		return
	}
	if err := e.repo.Emit(ctx, ev); err != nil {
		slog.WarnContext(ctx, "prestige: telemetry emit failed",
			"event_type", ev.EventType,
			"challenge_id", ev.ChallengeID,
			"err", err,
		)
	}
}

// newTelemetryID génère un identifiant aléatoire (8 octets hex = 16 chars).
func newTelemetryID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "tlm_" + hex.EncodeToString(buf[:])
}
