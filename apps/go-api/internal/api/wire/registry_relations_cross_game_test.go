package wire

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// newTestRegistry construit un Registry isolé (jamais le DefaultRegistry global)
// à partir des descripteurs fournis, pour piloter l'énumération Active() de
// l'adapter cross-jeu sans dépendre de la config disque.
func newTestRegistry(descs ...*title.TitleDescriptor) *title.Registry {
	reg := title.NewRegistry()
	for _, d := range descs {
		reg.Register(d)
	}
	return reg
}

// TestBuildCrossGameCooccurrence_NilGuards : dégradation gracieuse. cfg nil, pdb
// nil ou xuid vide → l'adapter est inerte (nil), JAMAIS un panic, pour ne pas
// faire échouer /relations sur un joueur sans xuid résolu.
func TestBuildCrossGameCooccurrence_NilGuards(t *testing.T) {
	cases := []struct {
		name string
		reg  *ServiceRegistry
		pdb  *duckdb.PlayerDB
	}{
		{"cfg nil", &ServiceRegistry{cfg: nil}, &duckdb.PlayerDB{XUID: "x1"}},
		{"pdb nil", &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: "/tmp"}}, nil},
		{"xuid empty", &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: "/tmp"}}, &duckdb.PlayerDB{XUID: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.reg.buildCrossGameCooccurrence(tc.pdb); got != nil {
				t.Fatalf("buildCrossGameCooccurrence(%s) = %v, want nil", tc.name, got)
			}
		})
	}
}

// TestBuildCrossGameCooccurrence_Valid : avec une config + un xuid, l'adapter est
// construit et propage le slug courant + l'xuid (champs lus par CooccurrencesByXUID).
func TestBuildCrossGameCooccurrence_Valid(t *testing.T) {
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: t.TempDir(), UserTimezone: "Europe/Paris"}}
	pdb := &duckdb.PlayerDB{XUID: "2533274895653213", TitleSlug: "halo_infinite"}

	got := reg.buildCrossGameCooccurrence(pdb)
	if got == nil {
		t.Fatal("buildCrossGameCooccurrence = nil, want non-nil")
	}
	c, ok := got.(*crossGameCooccurrence)
	if !ok {
		t.Fatalf("type = %T, want *crossGameCooccurrence", got)
	}
	if c.currentSlug != "halo_infinite" {
		t.Errorf("currentSlug = %q, want halo_infinite", c.currentSlug)
	}
	if c.myXUID != "2533274895653213" {
		t.Errorf("myXUID = %q", c.myXUID)
	}
	if c.registry == nil || c.resolver == nil {
		t.Error("registry/resolver doivent être initialisés")
	}
}

// TestCooccurrencesByXUID_EmptyInput : aucun xuid candidat → map vide sans même
// énumérer les titres (court-circuit, zéro IO).
func TestCooccurrencesByXUID_EmptyInput(t *testing.T) {
	c := &crossGameCooccurrence{
		currentSlug: "halo_infinite",
		myXUID:      "me",
		registry:    newTestRegistry(),
	}
	out := c.CooccurrencesByXUID(context.Background(), nil)
	if len(out) != 0 {
		t.Fatalf("CooccurrencesByXUID(nil) = %v, want empty", out)
	}
	out = c.CooccurrencesByXUID(context.Background(), []string{})
	if len(out) != 0 {
		t.Fatalf("CooccurrencesByXUID([]) = %v, want empty", out)
	}
}

// TestCooccurrencesByXUID_ExcludesCurrentAndInternal : le titre courant ET les
// titres internes (fixtures de test) sont sautés AVANT tout accès DB. Quand les
// seuls titres actifs sont ces deux-là, countForTitle n'est jamais appelé → map
// vide. Garantit qu'un badge user-facing ne nommera jamais le titre courant ni un
// titre interne (ex. synthetic_title_b).
func TestCooccurrencesByXUID_ExcludesCurrentAndInternal(t *testing.T) {
	reg := newTestRegistry(
		&title.TitleDescriptor{Slug: "halo_infinite", Name: "Halo Infinite", Status: title.StatusActive},
		&title.TitleDescriptor{Slug: "synthetic_title_b", Name: "Synthetic B", Status: title.StatusActive, IsInternal: true},
	)
	c := &crossGameCooccurrence{
		repoRoot:    t.TempDir(),
		timezone:    "UTC",
		currentSlug: "halo_infinite",
		myXUID:      "me",
		registry:    reg,
		resolver:    title.NewPathResolver(t.TempDir(), reg),
	}
	out := c.CooccurrencesByXUID(context.Background(), []string{"opp1", "opp2"})
	if len(out) != 0 {
		t.Fatalf("CooccurrencesByXUID = %v, want empty (courant + interne exclus)", out)
	}
}

// TestCooccurrencesByXUID_SkipsTitleWithAbsentDB : un AUTRE titre actif dont le
// shared n'existe pas sur disque → countForTitle ouvre, échoue, et le titre est
// sauté (best-effort). Aucune erreur propagée, aucune entrée pour ce titre.
func TestCooccurrencesByXUID_SkipsTitleWithAbsentDB(t *testing.T) {
	repoRoot := t.TempDir()
	reg := newTestRegistry(
		&title.TitleDescriptor{Slug: "halo_infinite", Name: "Halo Infinite", Status: title.StatusActive},
		&title.TitleDescriptor{Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive},
	)
	c := &crossGameCooccurrence{
		repoRoot:    repoRoot,
		timezone:    "UTC",
		currentSlug: "halo_infinite",
		myXUID:      "me",
		registry:    reg,
		// resolver pointe vers un repoRoot vide → SharedDBPath(halo_5) absent.
		resolver: title.NewPathResolver(repoRoot, reg),
	}
	out := c.CooccurrencesByXUID(context.Background(), []string{"opp1"})
	if len(out) != 0 {
		t.Fatalf("CooccurrencesByXUID = %v, want empty (DB halo_5 absente → skip)", out)
	}
}
