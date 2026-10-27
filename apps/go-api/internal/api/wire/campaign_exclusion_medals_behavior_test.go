//go:build integration

// campaign_exclusion_medals_behavior_test.go — le set des médailles « déjà obtenues » du
// snapshot post-sync exclut la Campagne (E2.3 du plan .ai/archive/V7.5/PLAN_BACKLOG_2026-10-15.md, DB-26
// du backlog 2026-09-26) : une médaille décrochée seulement en Campagne n'est pas « déjà
// obtenue », sinon sa première obtention en PvP ne notifierait jamais « médaille inédite ».

package wire

import (
	"context"
	"testing"
)

func TestCampaignExclusion_LoadEarnedMedalIDs(t *testing.T) {
	for _, tc := range b4Titres {
		pdb := newB4Env(t, tc.slug)
		ctx := context.Background()
		// 100 : obtenue en arène ET en Campagne ; 200 : en Campagne seulement.
		for _, m := range []struct {
			match string
			medal int64
		}{{"arena1", 100}, {"camp1", 100}, {"camp1", 200}} {
			if _, err := pdb.Shared.Exec(ctx, `INSERT INTO medals_earned (match_id, xuid, medal_name_id, count)
				VALUES (?, ?, ?, 1)`, m.match, testXUID, m.medal); err != nil {
				t.Fatalf("%s : seed medals_earned : %v", tc.slug, err)
			}
		}
		got := loadEarnedMedalIDs(ctx, pdb.Shared.SQLDb(), testXUID, tc.slug)
		if got == nil {
			t.Fatalf("%s : set nil (invalide) sur une lecture saine", tc.slug)
		}
		want := b4Attendu(tc.masquee, 1, 2)
		if len(got) != want {
			t.Errorf("%s : %d médailles déjà obtenues %v, attendu %d (Campagne masquée : %v)", tc.slug, len(got), got, want, tc.masquee)
		}
		if _, ok := got[200]; ok == tc.masquee {
			t.Errorf("%s : médaille 200 (Campagne seule) présente=%v, masquée=%v", tc.slug, ok, tc.masquee)
		}
	}
}
