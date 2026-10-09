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
