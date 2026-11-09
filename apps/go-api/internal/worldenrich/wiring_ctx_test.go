package worldenrich

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
)

// L'arrêt du serveur interrompt la résolution des comptes : un contexte annulé rend la main avant
// le premier compte au lieu de dérouler tous les échanges Microsoft (environ une seconde chacun),
// que la phase scheduler de l'arrêt attendait sans les voir (ADR 0045, lot L12.6).
func TestBuildMultiHaloSource_ContexteAnnuleInterromptLaBoucle(t *testing.T) {
	dir := t.TempDir()
	profils := filepath.Join(dir, "db_profiles.json")
	const contenu = `{"version":"3.0","admin":"A","profiles":{"halo_infinite":{
"A":{"xuid":"1000000000000001","gamertag":"A"},
"B":{"xuid":"1000000000000002","gamertag":"B"}}}}`
	if err := os.WriteFile(profils, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{RepoRoot: dir, DBProfilesPath: profils}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := BuildMultiHaloSource(ctx, cfg, 5, true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, attendu context.Canceled (boucle interrompue)", err)
	}
}
