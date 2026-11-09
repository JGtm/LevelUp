package service

import (
	"context"
	"testing"
)

// TestBootstrap_MediaH264Missing : /bootstrap publie le verdict de la sonde de l'outillage média (aucun
// encodeur H.264 utilisable) ; faux par défaut, vrai seulement quand le démarrage l'a posé.
func TestBootstrap_MediaH264Missing(t *testing.T) {
	cfg := newTestAppConfig(t)
	svc := NewBootstrapService(cfg, nil)

	resp, err := svc.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if resp.MediaH264Missing {
		t.Error("media_h264_missing doit valoir false tant que la sonde n'a rien constaté")
	}

	resp, err = svc.WithMediaH264Missing(true).Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !resp.MediaH264Missing {
		t.Error("media_h264_missing doit valoir true après WithMediaH264Missing(true)")
	}
}
