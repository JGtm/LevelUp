package api

import (
	"context"

	"levelup/go-api/internal/ops"
)

// SetMediaToolingInspector remplace la sonde de l'outillage média du démarrage et rend la
// restauration (cf. media_h264_bootstrap_cgo_test.go, seul à pouvoir construire le vrai routeur).
func SetMediaToolingInspector(f func(context.Context) ops.MediaToolingReport) (restore func()) {
	prev := inspectMediaTooling
	inspectMediaTooling = f
	return func() { inspectMediaTooling = prev }
}
