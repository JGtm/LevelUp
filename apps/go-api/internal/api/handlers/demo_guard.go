package handlers

import (
	"net/http"

	"levelup/go-api/internal/api/humacore"
)

// demoModeForbiddenCode est le code d'erreur d'une action refusée en mode démo. Source
// unique : le garde-rail internal/archlint/no_demo_forbidden_literal_test.go interdit ce
// littéral hors de ce fichier.
const demoModeForbiddenCode = "demo_mode_forbidden"

// refuseInDemo rend le refus 403 demo_mode_forbidden d'une action de mutation en mode
// démo, nil hors démo. En démo, RequireAdmin est transparent (middleware/require_admin.go) :
// sans ce refus, n'importe quel visiteur de la démo publique atteint la route.
//
// Appelants (backlog 2026-09-26) : sauvegarde (lot B5.6), création de profil, purge des
// données d'un titre et abonnements du watcher (lot B-C1). Le contrat OpenAPI de ces
// opérations est tenu par la réponse d'erreur `default` (ApiError), comme pour B5.6.
//
// action est un libellé technique anglais (garde-rail des libellés FR en dur).
func refuseInDemo(demoMode bool, action string) error {
	if !demoMode {
		return nil
	}
	return humacore.NewError(http.StatusForbidden, demoModeForbiddenCode, action+" is disabled in demo mode")
}
