// Package middleware — contract_validate.go : validation de contrat en dev mode.
//
// Sprint 40 T1 : valide que les réponses Go respectent les conventions du contrat :
//   - Content-Type: application/json sur toutes les routes /api/v1/
//   - Corps JSON valide (parseble)
//   - Réponses d'erreur au format standard {code, message, retryable}
//
// Activé uniquement quand LEVELUP_CONTRACT_VALIDATE=1 (dev / CI).
// En production, le middleware est un no-op transparent.
//
// Aucune dépendance externe — stdlib uniquement.
package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// contractEnabled indique si la validation est active (lu une seule fois au démarrage).
var contractEnabled = os.Getenv("LEVELUP_CONTRACT_VALIDATE") == "1"

// ContractValidate est un middleware qui valide les réponses en dev mode.
// En dehors de LEVELUP_CONTRACT_VALIDATE=1, il est transparent (zero overhead).
func ContractValidate(next http.Handler) http.Handler {
	if !contractEnabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capturer la réponse dans un buffer pour l'inspecter après.
		cw := &contractResponseWriter{
			ResponseWriter: w,
			buf:            &bytes.Buffer{},
		}
		next.ServeHTTP(cw, r)

		// Valider uniquement les routes API v1.
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			return
		}

		ctype := cw.Header().Get("Content-Type")
		if !strings.HasPrefix(ctype, "application/json") {
			slog.Warn("contract: Content-Type non-JSON",
				"path", r.URL.Path,
				"method", r.Method,
				"status", cw.status,
				"content_type", ctype,
			)
		}

		body := cw.buf.Bytes()
		if len(body) == 0 {
			return
		}

		var parsed any
		if err := json.Unmarshal(body, &parsed); err != nil {
			slog.Warn("contract: corps JSON invalide",
				"path", r.URL.Path,
				"method", r.Method,
				"status", cw.status,
				"err", err,
			)
			return
		}

		// Pour les erreurs (4xx/5xx), vérifier le format standard.
		if cw.status >= 400 {
			validateErrorShape(parsed, r, cw.status)
		}
	})
}

// validateErrorShape vérifie que la réponse d'erreur respecte {code, message, retryable}.
func validateErrorShape(parsed any, r *http.Request, status int) {
	obj, ok := parsed.(map[string]any)
	if !ok {
		slog.Warn("contract: erreur non-objet JSON",
			"path", r.URL.Path,
			"method", r.Method,
			"status", status,
		)
		return
	}
	missing := []string{}
	for _, field := range []string{errKeyCode, errKeyMessage, errKeyRetryable} {
		if _, found := obj[field]; !found {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		slog.Warn("contract: champs manquants dans réponse d'erreur",
			"path", r.URL.Path,
			"method", r.Method,
			"status", status,
			"missing_fields", strings.Join(missing, ","),
		)
	}
}

// contractResponseWriter capture le corps de réponse pour inspection.
type contractResponseWriter struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	status int
	wrote  bool
}

func (cw *contractResponseWriter) WriteHeader(code int) {
	if !cw.wrote {
		cw.status = code
		cw.wrote = true
	}
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *contractResponseWriter) Write(b []byte) (int, error) {
	if !cw.wrote {
		cw.status = http.StatusOK
		cw.wrote = true
	}
	cw.buf.Write(b) //nolint:errcheck
	return cw.ResponseWriter.Write(b)
}
