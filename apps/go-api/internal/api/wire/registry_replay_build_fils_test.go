package wire

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/port"
)

// TestRequeteReplayAdmin_PorteLesFilsDeLInstallation : l'action d'administration et la file de regeneration du
// bureau (meme fonction) demandent le nombre de fils de la configuration (maillon « action admin »).
func TestRequeteReplayAdmin_PorteLesFilsDeLInstallation(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: "racine"}
	cfg.Capabilities.DecoderThreads = 4
	r := &ServiceRegistry{cfg: cfg}
	req := r.requeteReplayAdmin("m", "halo_infinite", []string{"carte"}, "chunks", port.MatchFacts{})
	if req.DecoderThreads != 4 {
		t.Errorf("fils de la requete = %d, attendu 4", req.DecoderThreads)
	}
	if req.RepoRoot != "racine" || req.MatchID != "m" || req.FilmDir != "chunks" {
		t.Errorf("requete mal composee : %+v", req)
	}
}
