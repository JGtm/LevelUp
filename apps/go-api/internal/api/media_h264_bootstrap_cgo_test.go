//go:build cgo

// media_h264_bootstrap_cgo_test.go — le verdict de la sonde média du démarrage atteint /bootstrap
// par le vrai routeur (câblage server_apiv1.go : sonde -> BootstrapService.WithMediaH264Missing).
// Le service seul est testé dans service/bootstrap_media_h264_test.go ; ici, le branchement.

package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/media"
	"levelup/go-api/internal/ops"
)

func bootstrapMediaH264Missing(t *testing.T, report ops.MediaToolingReport) bool {
	t.Helper()
	restore := api.SetMediaToolingInspector(func(context.Context) ops.MediaToolingReport { return report })
	defer restore()
	h := buildDesktopRouter(t)
	rec := (desktopReq{target: "/api/v1/bootstrap", cookie: enterDesktop(t, h)}).do(h)
	if rec.Code != http.StatusOK {
		t.Fatalf("/bootstrap : %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		MediaH264Missing bool `json:"media_h264_missing"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("décodage /bootstrap : %v", err)
	}
	return out.MediaH264Missing
}

func TestBootstrap_PubliePasDEncodeurH264(t *testing.T) {
	sans := ops.MediaToolingReport{FFmpegFound: true, CapabilitiesProbed: true, H264Err: media.ErrNoH264Encoder}
	if !bootstrapMediaH264Missing(t, sans) {
		t.Error("sonde sans encodeur H.264 : /bootstrap doit porter media_h264_missing=true")
	}
	avec := ops.MediaToolingReport{FFmpegFound: true, CapabilitiesProbed: true, H264Retained: "libx264"}
	if bootstrapMediaH264Missing(t, avec) {
		t.Error("sonde avec encodeur H.264 : media_h264_missing doit rester faux")
	}
}
