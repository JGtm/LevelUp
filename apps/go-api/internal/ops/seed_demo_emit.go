// Package ops — seed_demo_emit.go : génération d'un manifeste démo depuis la
// sélection dynamique courante (mode `seed-demo --emit-manifest`).
package ops

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// EmitDemoManifest exécute la sélection dynamique actuelle UNE fois et sérialise le
// corpus résultant en manifeste figé (à curer puis committer). Ne seede AUCUNE DB.
// Le compte source (SourceLabel) est enregistré pour traçabilité ; son xuid ne l'est pas
// (le manifeste est versionné).
func EmitDemoManifest(ctx context.Context, opts SeedDemoOptions, outPath string) (*DemoManifest, error) {
	if err := validateSeedDemoOpts(&opts); err != nil {
		return nil, fmt.Errorf("emit-manifest: %w", err)
	}
	dc := selectDynamicCorpus(ctx, opts)
	m := newEmittedManifest(opts, dc, time.Now())
	if len(m.CorpusMatchIDs()) == 0 {
		return nil, fmt.Errorf("emit-manifest: corpus vide pour xuid=%s (la sélection dynamique n'a rien retourné)", opts.SourceXUID)
	}
	if err := writeDemoManifest(outPath, m); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "seed-demo: manifeste émis",
		"path", outPath, "solo", len(dc.solo), "squad", len(dc.squad), "ranked", len(dc.ranked), "media", len(dc.media))
	return m, nil
}

// newEmittedManifest assemble le manifeste émis pour une sélection. Il ne porte aucun
// identifiant de joueur : le fichier est committé, et aucun lecteur n'en a l'usage.
func newEmittedManifest(opts SeedDemoOptions, dc dynamicCorpus, now time.Time) *DemoManifest {
	return &DemoManifest{
		Version:        demoManifestVersion,
		TitleSlug:      opts.TitleSlug,
		SourceGamertag: opts.SourceLabel,
		GeneratedAt:    now.UTC().Format(time.RFC3339),
		GeneratedBy:    "levelup seed-demo --emit-manifest",
		Notes:          "Sélection dynamique figée — à CURER (matchs solo/squad/ranked, ancres médias) puis committer. Le roster anonymisé et l'association des médias sont dérivés du corpus.",
		Corpus: DemoManifestCorpus{
			SoloMatchIDs:   orEmpty(dc.solo),
			SquadMatchIDs:  orEmpty(dc.squad),
			RankedMatchIDs: orEmpty(dc.ranked),
			MediaMatchIDs:  orEmpty(dc.media),
		},
	}
}

// orEmpty garantit un slice non-nil (sérialisé `[]` et non `null` → manifeste lisible).
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
