// Package observability — level.go : niveau de journal d'un échec selon la vie de la
// requête qui l'a subi.
package observability

import (
	"context"
	"errors"
	"log/slog"
)

// LevelUnlessCanceled rend slog.LevelDebug quand l'échec vient de la fin de la requête —
// son contexte a pris fin, ou l'erreur est une annulation —, sinon level. Un client parti
// (onglet fermé, navigation, requête remplacée par la suivante) n'est pas une anomalie du
// serveur : l'ERROR ou le WARN d'un chargement qu'il a abandonné noyait les vraies pannes
// (revue adversariale de la campagne perf, lot L9-go, 2026-09-23).
//
// Usage : slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError), msg, ...).
// SOURCE UNIQUE du prédicat (CLAUDE.md n°6) : level_test.go interdit sa copie en ligne
// ailleurs sous internal/.
func LevelUnlessCanceled(ctx context.Context, err error, level slog.Level) slog.Level {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) {
		return slog.LevelDebug
	}
	return level
}

// IsCancellation dit si err est une annulation ou un dépassement de contexte : la lecture a
// été coupée, la source n'a pas été jugée en panne.
func IsCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// LevelOnCancellation rend slog.LevelWarn quand err est une annulation ou un dépassement de
// contexte (IsCancellation), sinon level. Contrat de la vue match : une lecture coupée est un
// WARN, jamais une ERREUR de panne (source unique du prédicat, avec IsCancellation).
// Usage : slog.Log(ctx, observability.LevelOnCancellation(err, slog.LevelError), msg, ...).
func LevelOnCancellation(err error, level slog.Level) slog.Level {
	if IsCancellation(err) {
		return slog.LevelWarn
	}
	return level
}
