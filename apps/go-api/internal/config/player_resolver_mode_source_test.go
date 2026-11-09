package config

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// Le PlayerDB d'un titre reçoit la source du mode déclarée par le manifeste de CE titre, lue dans le
// registre partagé : un titre à « type de jeu » (Halo 5) la reçoit, Halo Infinite et un titre sans clé
// gardent la partie. Sans cette injection, PlayerMatchesRepo rend aucun mode pour Halo 5.
func TestBuildPoolConfig_ModeSourceDuManifeste(t *testing.T) {
	prev := title.DefaultRegistry()
	t.Cleanup(func() { title.SetDefaultRegistry(prev) })

	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{Slug: "halo_5", Name: "H5", Status: title.StatusActive, ModeSource: title.ModeSourceGameVariant})
	reg.Register(&title.TitleDescriptor{Slug: "sans_cle", Name: "X", Status: title.StatusActive})
	title.SetDefaultRegistry(reg)

	cfg := &AppConfig{RepoRoot: t.TempDir()}
	p := &domain.PlayerSummary{Gamertag: "GT", XUID: "1"}
	for slug, want := range map[string]title.ModeSource{
		"halo_5":          title.ModeSourceGameVariant,
		title.DefaultSlug: title.ModeSourcePair,
		"sans_cle":        title.ModeSourcePair,
		"inconnu":         title.ModeSourcePair,
	} {
		if got := buildPoolConfig(cfg, p, slug).ModeSource; got != want {
			t.Errorf("%s : ModeSource = %q, want %q", slug, got, want)
		}
	}
}
