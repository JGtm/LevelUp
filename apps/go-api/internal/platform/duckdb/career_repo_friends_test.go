//go:build integration

// Package duckdb — career_repo_friends_test.go : les xuids des amis en UNE lecture, sans la vue
// des noms (lot perf L9-go, 2026-09-23, revue adversariale D, P1). La référence est l'ANCIENNE
// lecture (ExplorerRepo.ResolveXUIDByGamertag, sur la VRAIE vue canonique) ; les écarts voulus
// sont écrits en clair.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/observability/timing"
)

// xuidSelonLaVue rejoue l'ancienne résolution d'un ami (une lecture de la vue par ami).
func xuidSelonLaVue(t *testing.T, pdb *PlayerDB, gamertag string) string {
	t.Helper()
	xuid, err := NewExplorerRepo(pdb, pdb.XUID).ResolveXUIDByGamertag(context.Background(), gamertag)
	if err != nil {
		return ""
	}
	return xuid
}

// TestCareerRepo_ResolveFriendXUIDs_MemesXuidsQueLaVue : chaque niveau que la vue et la lecture
// partagent — alias sans casse, participant de l'historique, alias vide qui laisse la main au
// participant, alias qui l'emporte — rend le xuid de l'ancienne lecture ; un bot, un inconnu : rien.
func TestCareerRepo_ResolveFriendXUIDs_MemesXuidsQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	execOnSharedDBs(t, pdb, context.Background(), `INSERT INTO shared.match_participants
		(match_id, xuid, gamertag, outcome, team_id) VALUES ('ma2', 'bid(2.0)', 'BotNomme', 2, 0)`)
	amis := []string{"nomalias", "NomPart", "NOMPARTALIASVIDE", "NomAliasTriple", "BotNomme", "Inconnu"}
	attendus := map[string]string{
		"nomalias":         "x_alias",
		"NomPart":          "x_part",
		"NOMPARTALIASVIDE": "x_alias_vide",
		"NomAliasTriple":   "x_triple",
	}
	ctx, tm := timing.WithTimings(context.Background())
	got, err := NewCareerRepo(pdb).ResolveFriendXUIDs(ctx, amis)
	if err != nil {
		t.Fatalf("ResolveFriendXUIDs : %v", err)
	}
	for _, ami := range amis {
		if got[ami] != attendus[ami] {
			t.Errorf("%s : xuid %q, want %q", ami, got[ami], attendus[ami])
		}
		if vue := xuidSelonLaVue(t, pdb, ami); got[ami] != vue {
			t.Errorf("%s : xuid %q, l'ancienne lecture (vue) rendait %q", ami, got[ami], vue)
		}
	}
	sections := map[string]int{}
	for _, s := range tm.Snapshot() {
		sections[s.Name] = s.Calls
	}
	if sections["career_friends"] != 1 {
		t.Errorf("sections = %v, want career_friends x1 (une seule lecture)", sections)
	}
}

// TestCareerRepo_ResolveFriendXUIDs_EcartsNommes : les écarts voulus avec l'ancienne lecture.
//   - un ANCIEN nom porté dans l'historique du joueur (x_triple y joue sous « NomPartTriple »,
//     son alias est « NomAliasTriple ») : la vue ne connaissait que le nom d'affichage, la
//     lecture reconnaît l'ami ;
//   - un nom porté seulement HORS de l'historique du joueur : la vue le trouvait, la lecture
//     non — un joueur jamais croisé ne peut pas figurer dans ses rencontres, l'exclure ne change
//     rien ;
//   - un nom porté par deux xuids : le niveau le plus fort, puis le plus petit xuid (la vue en
//     prenait un au hasard).
func TestCareerRepo_ResolveFriendXUIDs_EcartsNommes(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	ctx := context.Background()
	participant := `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, 2, 0)`
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry (match_id) VALUES ('mx')`)
	execOnSharedDBs(t, pdb, ctx, participant, "mx", "x_etranger", "NomAilleurs") // match sans le joueur
	execOnSharedDBs(t, pdb, ctx, participant, "ma2", "x_homonyme_b", "Homonyme")
	execOnSharedDBs(t, pdb, ctx, participant, "ma2", "x_homonyme_a", "Homonyme")
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES ('x_homonyme_z', 'homonyme')`)

	got, err := NewCareerRepo(pdb).ResolveFriendXUIDs(ctx, []string{"NomPartTriple", "NomAilleurs", "Homonyme"})
	if err != nil {
		t.Fatalf("ResolveFriendXUIDs : %v", err)
	}
	if got["NomPartTriple"] != "x_triple" || xuidSelonLaVue(t, pdb, "NomPartTriple") != "" {
		t.Errorf("ancien nom : lecture %q (want x_triple), vue %q (want rien)", got["NomPartTriple"], xuidSelonLaVue(t, pdb, "NomPartTriple"))
	}
	if got["NomAilleurs"] != "" || xuidSelonLaVue(t, pdb, "NomAilleurs") != "x_etranger" {
		t.Errorf("nom hors historique : lecture %q (want rien), vue %q (want x_etranger)", got["NomAilleurs"], xuidSelonLaVue(t, pdb, "NomAilleurs"))
	}
	if got["Homonyme"] != "x_homonyme_z" {
		t.Errorf("homonymes : %q, want x_homonyme_z (l'alias l'emporte sur les participants)", got["Homonyme"])
	}
}
