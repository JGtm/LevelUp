package replaybuild

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// TestWithGeometryDir_RepertoireSansPropImposeZeroProp — l'echappatoire `--geometry` ne doit
// jamais s'annuler toute seule : un repertoire impose qui ne porte aucun CSV de carte (ou aucun
// prop a emprise) impose ZERO prop, il ne rend pas la main a la resolution par carte. Sinon le
// journal dit « geometrie imposee, objets=0 » et l'artefact sort avec les props de `ridgeline`
// (revue de vague, 2026-09-12).
func TestWithGeometryDir_RepertoireSansPropImposeZeroProp(t *testing.T) {
	repoRoot := t.TempDir()
	const slug = "halo_infinite"
	typesDir := title.NewPathResolver(repoRoot).MapGeometryDir(slug, "")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Catalogue de types minimal et valide : l'en-tete seul suffit a `loadTypeExtents`.
	header := "type_id,instances,obj_group,obj_gid,hlmt_gid,mode_gid,coll_gid,phmo_gid,mode_module,min_x,min_y,min_z,max_x,max_y,max_z,dx,dy,dz,geom,note\n"
	if err := os.WriteFile(filepath.Join(typesDir, replay.ObjectTypesFile), []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}
	b := &Builder{repoRoot: repoRoot, titleSlug: slug, geometries: map[string][]replay.MapObject{
		"ridgeline": {{}}, // ce que la resolution par carte rendrait si l'imposition s'annulait
	}}
	b.WithGeometryDir(t.TempDir()) // impose : aucun map_objects.csv dedans

	if got := b.geometryFor("ridgeline"); got == nil || len(got) != 0 {
		t.Fatalf("un repertoire impose sans prop doit imposer zero prop, obtenu %v", got)
	}
}
