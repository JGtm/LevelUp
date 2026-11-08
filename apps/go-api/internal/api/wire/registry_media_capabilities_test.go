package wire

// registry_media_capabilities_test.go — le choix du trasher et du revealer du MediaService suit
// les capacités du déploiement, dans les deux sens.

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/port"
)

type trasherStub struct{}

func (trasherStub) TrashMediaFile(context.Context, string) (bool, error) { return true, nil }

type revealerStub struct{}

func (revealerStub) Reveal(context.Context, string) error { return nil }

func TestMediaTrasherFor(t *testing.T) {
	built := 0
	build := func() port.MediaFileTrasher { built++; return trasherStub{} }

	if got := mediaTrasherFor(config.DeploymentCapabilities{TrashMediaSources: true}, build); got == nil || built != 1 {
		t.Errorf("capacité vraie : trasher = %v, builds = %d, want un trasher construit une fois", got, built)
	}
	built = 0
	if got := mediaTrasherFor(config.DeploymentCapabilities{}, build); got != nil || built != 0 {
		t.Errorf("capacité fausse : trasher = %v, builds = %d, want nil sans construction", got, built)
	}
	// Les autres capacités du bureau ne suffisent pas.
	other := config.DeploymentCapabilities{RevealFilesInExplorer: true, KeepMediaSources: true}
	if got := mediaTrasherFor(other, build); got != nil {
		t.Errorf("sans TrashMediaSources : trasher = %v, want nil", got)
	}
}

func TestMediaRevealerFor(t *testing.T) {
	built := 0
	build := func() port.MediaFileRevealer { built++; return revealerStub{} }

	if got := mediaRevealerFor(config.DeploymentCapabilities{RevealFilesInExplorer: true}, build); got == nil || built != 1 {
		t.Errorf("capacité vraie : revealer = %v, builds = %d, want un revealer construit une fois", got, built)
	}
	built = 0
	if got := mediaRevealerFor(config.DeploymentCapabilities{}, build); got != nil || built != 0 {
		t.Errorf("capacité fausse : revealer = %v, builds = %d, want nil sans construction", got, built)
	}
	other := config.DeploymentCapabilities{TrashMediaSources: true, KeepMediaSources: true}
	if got := mediaRevealerFor(other, build); got != nil {
		t.Errorf("sans RevealFilesInExplorer : revealer = %v, want nil", got)
	}
}
