// Package middleware — demo_read_only.go : la démo publique est en LECTURE SEULE.
//
// CONTRAT. En mode démo, toute requête qui ÉCRIT reçoit 403 `demo_mode_forbidden`, avant
// d'atteindre un handler. « Écrit » se décide par [IsMutatingRequest], la classification
// unique du dépôt (verbe + sous-chemin joueur) : GET/HEAD/OPTIONS et les POST de lecture
// produit (pages, filtres, tactique, série d'engagement) passent. Deux exemptions
// s'ajoutent, nommées dans [demoExtraReadPosts] et [DemoReadOnly] :
//
//   - `POST /session/context` : l'état de navigation du VISITEUR (titre, joueur affiché),
//     écrit dans le magasin de sessions de la démo (`<démo>/runtime/sessions`). Sans lui, un
//     visiteur ne peut pas changer de joueur ni de titre ;
//   - le protocole ouvrier (préfixe passé par l'appelant) : authentifié par un jeton Bearer
//     dédié, sans cookie, il n'est pas une surface de visiteur. La démo n'a pas de jeton
//     (docker-compose : aucun env_file sur le service démo), donc il y répond 503
//     `build_queue_disabled`.
//
// POURQUOI UN MIDDLEWARE ET PAS UN REFUS PAR ROUTE. `RequireAdmin` et `RequireAuth` sont
// transparents en démo : toute route d'écriture était atteignable par un visiteur anonyme
// tant qu'un handler n'avait pas pensé à la refuser. La garde générale inverse le défaut :
// une route d'écriture ajoutée demain est refusée en démo sans que personne n'ait à y
// penser. Le ratchet `internal/api/demo_read_only_ratchet_test.go` tient le câblage et la
// liste des écritures que la garde laisse passer.
//
// Le littéral du code d'erreur n'existe qu'ici (garde-rail
// internal/archlint/no_demo_forbidden_literal_test.go).
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// DemoModeForbiddenCode est le code d'erreur d'une écriture refusée en mode démo.
const DemoModeForbiddenCode = "demo_mode_forbidden"

// demoExtraReadPosts : POST hors lectures produit que la démo laisse passer, par chemin
// absolu exact. Chaque entrée porte sa raison dans l'en-tête du fichier.
var demoExtraReadPosts = map[string]bool{
	"/api/v1/session/context": true,
}

// DemoReadOnly retourne la garde « démo en lecture seule ». Hors démo, elle est l'identité.
// exemptPrefixes : préfixes absolus authentifiés hors session (protocole ouvrier), traités
// comme par [CSRF] (frontière de segment, chemin nettoyé avant comparaison).
func DemoReadOnly(demoMode bool, exemptPrefixes ...string) func(http.Handler) http.Handler {
	exempt := normalizeExemptPrefixes(exemptPrefixes)
	return func(next http.Handler) http.Handler {
		if !demoMode {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if DemoAllows(r) || (!AmbiguousPath(r) && isExemptPath(r.URL.Path, exempt)) {
				next.ServeHTTP(w, r)
				return
			}
			slog.InfoContext(r.Context(), "demo: écriture refusée (démo en lecture seule)",
				"method", r.Method, "path", r.URL.Path)
			WriteDemoForbidden(w)
		})
	}
}

// DemoAllows dit si la démo laisse passer la requête : une lecture au sens de
// [IsMutatingRequest], ou un POST de [demoExtraReadPosts] dont le chemin se lit comme il se
// route (cf. AmbiguousPath). Exportée pour que le ratchet des routes raisonne sur la MÊME
// définition que la garde.
func DemoAllows(r *http.Request) bool {
	if !IsMutatingRequest(r) {
		return true
	}
	return r.Method == http.MethodPost && !AmbiguousPath(r) && demoExtraReadPosts[r.URL.Path]
}

// WriteDemoForbidden écrit le refus 403 `demo_mode_forbidden`, dans la forme ApiError
// (code, message, retryable) de toutes les erreurs de l'API.
func WriteDemoForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		errKeyCode:      DemoModeForbiddenCode,
		errKeyMessage:   "This action is disabled in demo mode.",
		errKeyRetryable: false,
	})
}
