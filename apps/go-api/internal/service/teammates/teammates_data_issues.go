// Package teammates - teammates_data_issues.go : collecte des chargements
// best-effort dégradés d'une requête page Escouade.
//
// Avant (2026-08-02) chaque échec best-effort se contentait d'un slog.Warn puis
// poursuivait : la page affichait des nombres amputés sans que rien ne le dise,
// d'où des compteurs non reproductibles d'une requête à l'autre. Tout échec passe
// désormais par ce collecteur : journal structuré côté serveur (ERROR, DEBUG quand la
// requête a pris fin) ET remontée dans la réponse (domain.DataIssue) pour affichage
// côté UI.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// dataIssues accumule les dégradations d'une seule requête (usage séquentiel,
// aucune synchronisation nécessaire — GetPage n'a pas de goroutine concurrente).
type dataIssues struct {
	items []domain.DataIssue
}

// add logge l'échec (jamais avalé : ERROR, ou DEBUG quand la requête a pris fin — un
// client parti n'est pas une panne, lot perf L9-go) et le mémorise pour la réponse.
// detail identifie la ressource concernée (ex. gamertag) — jamais le message
// d'erreur brut, qui reste côté serveur.
func (d *dataIssues) add(ctx context.Context, code, detail string, err error) {
	slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError), "teammates.data_issue",
		"code", code,
		"detail", detail,
		"err", err,
	)
	if d == nil {
		return
	}
	d.items = append(d.items, domain.DataIssue{Code: code, Detail: detail})
}

// list retourne les dégradations collectées (nil si aucune → champ omis du JSON).
func (d *dataIssues) list() []domain.DataIssue {
	if d == nil || len(d.items) == 0 {
		return nil
	}
	return d.items
}
