// Package service — openspartan_post_import_title_test.go : ratchet C.1 du lot
// finitions LUSR (2026-09-13).
//
// Le replay LUSR du post-import est title-aware (GetLUSRChainForTitle lit
// ctxkeys.TitleSlug) alors que les bases sont ouvertes sur opts.TitleSlug. Un ctx
// de requête portant un AUTRE titre (header X-LevelUp-Title, session sur le second
// titre) ferait écrire une chaîne étrangère dans la base du joueur — exactement la
// corruption h5_arena du 2026-06-26 (.ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md
// §5.1 T3). Le site n'a pas de test fonctionnel bon marché (il ouvre 4 bases et
// prend des leases) : ce ratchet lit la source et exige que l'appel reste stampé.
package service

import (
	"os"
	"strings"
	"testing"
)

func TestPostImportLUSRCallIsTitleStamped(t *testing.T) {
	const src = "openspartan_post_import_service.go"
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("lecture %s: %v", src, err)
	}
	const want = "s.recomputeLUSR(ctxkeys.WithTitleSlug(ctx, opts.TitleSlug)"
	if !strings.Contains(string(b), want) {
		t.Fatalf("%s : l'appel à recomputeLUSR doit porter le titre de la BASE ouverte "+
			"(attendu %q). Un ctx entrant d'un autre titre ferait écrire une chaîne LUSR "+
			"étrangère dans la player DB — corruption h5_arena du 2026-06-26.", src, want)
	}
}
