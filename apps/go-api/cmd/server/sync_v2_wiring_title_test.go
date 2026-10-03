// Package main — sync_v2_wiring_title_test.go : gardes C.2 du lot finitions LUSR
// (2026-09-13), spécification G1/G2 du rapport
// .ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md §5.3.
//
// Contexte : la corruption LUSR h5_arena du 2026-06-26 vient d'une DOUBLE SOURCE
// DE TITRE dans le câblage V2 — handles DB depuis le titre du CYCLE, moteur depuis
// le titre du PROFIL. Les 4 joueurs déclarés sous deux titres dans db_profiles.json
// ont donc été synchronisés une seconde fois « en halo_5 » sur leurs bases
// halo_infinite, qui ont reçu 2 461 lignes match_skill_rank de chaîne h5_arena.
//
// Deux gardes ici :
//   - fail-loud runtime : un profil de titre étranger est refusé par la fabrique ;
//   - ratchet source : plus aucune ligne du câblage ne peut RÉINTRODUIRE le titre
//     du profil comme source (chemin, persister, moteur) — il ne sert qu'au garde.
package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	syncv2 "levelup/go-api/internal/sync/v2"
)

func TestBuildSyncEngineFactory_RefusesForeignTitleProfile(t *testing.T) {
	deps := SyncV2WiringDeps{
		Cfg:       &config.AppConfig{RepoRoot: t.TempDir()},
		TitleSlug: titlePkg.DefaultSlug,
	}
	factory := buildSyncEngineFactoryParityComplete(deps)

	cases := []struct {
		name         string
		profileTitle string
		wantErr      bool
	}{
		{"titre du profil == titre du cycle", titlePkg.DefaultSlug, false},
		{"titre du profil vide (défaut historique)", "", false},
		{"titre du profil étranger", "halo_5", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			engine, err := factory(context.Background(), syncv2.PlayerProfile{
				PlayerSlug: "JGtm", Gamertag: "JGtm", XUID: "2533274858283686",
				TitleSlug: c.profileTitle,
			})
			if !c.wantErr {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				if engine == nil {
					t.Fatal("engine nil alors que le profil est accepté")
				}
				return
			}
			if err == nil {
				t.Fatal("err nil : un profil d'un titre étranger doit être REFUSÉ " +
					"(sinon il écrit la sémantique de son titre dans les bases du cycle)")
			}
			if engine != nil {
				t.Error("engine non nil alors que le profil est refusé")
			}
			for _, want := range []string{"halo_5", titlePkg.DefaultSlug, "JGtm"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("l'erreur doit nommer %q pour être diagnosticable ; err = %v", want, err)
				}
			}
		})
	}
}

// TestSyncV2WiringHasSingleTitleSource — ratchet G2 : dans sync_v2_wiring.go, toute
// ligne qui lit le titre du PROFIL (p.TitleSlug) doit aussi porter le titre du CYCLE
// (deps.TitleSlug). Autrement dit : p.TitleSlug n'est plus qu'un opérande de
// COMPARAISON, jamais une source de chemin, de persister ou de moteur. Une seule
// source de titre = la classe de défaut du 2026-06-26 est structurellement fermée.
func TestSyncV2WiringHasSingleTitleSource(t *testing.T) {
	const src = "sync_v2_wiring.go"
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("lecture %s: %v", src, err)
	}
	for i, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, "p.TitleSlug") {
			continue
		}
		if !strings.Contains(line, "deps.TitleSlug") {
			t.Errorf("%s:%d — p.TitleSlug employé sans deps.TitleSlug sur la même ligne :\n\t%s\n"+
				"Le titre du cycle est la SEULE source (handles DB, persister, moteur) ; le titre "+
				"du profil ne sert qu'au garde d'égalité. Double source = corruption LUSR "+
				"h5_arena du 2026-06-26 (rapport .ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md §3).",
				src, i+1, strings.TrimSpace(line))
		}
	}
}
