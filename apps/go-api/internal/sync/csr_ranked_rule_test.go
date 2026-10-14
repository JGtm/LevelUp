// Package sync — csr_ranked_rule_test.go : la règle unique de classement (D-10) sur deux charges
// skill complètes, en forme de réponse JSON de l'API :
//   - testdata/skill_match_social.json : RankRecap d'un match social, valeurs transcrites de la
//     charge réelle capturée le 2026-05-24 (jgtm_full_match/api_skill.json) : tout à zéro ;
//   - testdata/skill_match_ranked.json : réponse skill RÉELLE d'un Ranked Slayer (Nuzzles, 30/09/2026,
//     HTTP 200), copiée telle quelle : Onyx 1756 -> 1749, InitialMeasurementMatches = 5.
package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	rulePlayerXUID = "2533274823110022" // joueur de la charge sociale
	ruleRankedXUID = "2535472547643888" // joueur de la charge classée
)

func loadSkillFixture(t *testing.T, name string) map[string]*MatchSkillData {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("lecture fixture %s: %v", name, err)
	}
	m, err := ParseMatchSkillResponseJSON(body)
	if err != nil {
		t.Fatalf("ParseMatchSkillResponseJSON(%s): %v", name, err)
	}
	if len(m) != 1 {
		t.Fatalf("fixture %s : %d joueurs, want 1", name, len(m))
	}
	return m
}

func TestSkillShowsRankedMatch_Fixtures(t *testing.T) {
	ranked := loadSkillFixture(t, "skill_match_ranked.json")
	social := loadSkillFixture(t, "skill_match_social.json")

	if got := ranked[ruleRankedXUID].PostMatchCSR.InitialMeasurementMatches; got != 5 {
		t.Fatalf("InitialMeasurementMatches non décodé : %d, want 5", got)
	}
	if !SkillShowsRankedMatch(ranked[ruleRankedXUID]) || !AnySkillShowsRankedMatch(ranked) {
		t.Error("charge classée : doit être classée")
	}
	if social[rulePlayerXUID].PostMatchCSR == nil {
		t.Fatal("la charge sociale porte un RankRecap (à zéro) : PostMatchCSR attendu non nil")
	}
	if SkillShowsRankedMatch(social[rulePlayerXUID]) || AnySkillShowsRankedMatch(social) {
		t.Error("charge sociale : ne doit PAS être classée (RankRecap présent mais à zéro)")
	}
	if SkillShowsRankedMatch(nil) || AnySkillShowsRankedMatch(nil) {
		t.Error("absence de charge : jamais classé")
	}
}

// Le classement ne dépend ni du nom de la playlist ni de reg.IsRanked : une charge classée écrit
// son CSR même si le registre dit « faux » ; une charge sociale n'en écrit pas même si le
// registre dit « vrai ».
func TestRankedRule_DecidedBySkillOnly(t *testing.T) {
	ranked := loadSkillFixture(t, "skill_match_ranked.json")
	social := loadSkillFixture(t, "skill_match_social.json")
	st := time.Date(2026, 10, 7, 18, 0, 0, 0, time.UTC)
	regFalse := &MatchRegistryRow{MatchID: "m-ranked", StartTime: st, IsRanked: false}
	regTrue := &MatchRegistryRow{MatchID: "m-social", StartTime: st, IsRanked: true}

	row := ExtractCSRRowIfRanked(regFalse, ranked[ruleRankedXUID])
	if row == nil {
		t.Fatal("match classé, registre is_ranked=false : une ligne CSR est attendue")
	}
	if row.RatingValue == nil || *row.RatingValue != 1749 || row.Tier != "Onyx" || row.SubTier != 0 {
		t.Errorf("ligne CSR inattendue : %+v", row)
	}
	if row.RatingDelta == nil || *row.RatingDelta != -7 {
		t.Errorf("delta = %v, want -7", row.RatingDelta)
	}
	if shared := ExtractAllSharedCSRRows(regFalse, ranked); len(shared) != 1 {
		t.Errorf("shared : %d lignes, want 1", len(shared))
	}

	if ExtractCSRRowIfRanked(regTrue, social[rulePlayerXUID]) != nil {
		t.Error("match social, registre is_ranked=true : aucune ligne CSR attendue")
	}
	if shared := ExtractAllSharedCSRRows(regTrue, social); len(shared) != 0 {
		t.Errorf("shared social : %d lignes, want 0", len(shared))
	}
}

// Un lobby mêlant un joueur classé et un joueur au RankRecap vide : le match est classé (un
// seul suffit), seul le joueur classé reçoit une ligne.
func TestRankedRule_MixedLobby(t *testing.T) {
	ranked := loadSkillFixture(t, "skill_match_ranked.json")
	social := loadSkillFixture(t, "skill_match_social.json")
	lobby := map[string]*MatchSkillData{
		ruleRankedXUID: ranked[ruleRankedXUID],
		rulePlayerXUID: social[rulePlayerXUID],
	}
	if !AnySkillShowsRankedMatch(lobby) {
		t.Error("lobby mixte : classé")
	}
	reg := &MatchRegistryRow{MatchID: "m-mixed", StartTime: time.Now().UTC()}
	if rows := ExtractAllSharedCSRRows(reg, lobby); len(rows) != 1 || rows[0].XUID != ruleRankedXUID {
		t.Errorf("lignes = %+v, want 1 (joueur classé)", rows)
	}
}

// Le chemin de sync V2 (celui de la production) pose is_ranked et construit le CSR depuis le
// RankRecap, sur un payload stats SANS nom de playlist.
func TestBuildBatchFromRawForV2_RankedByRankRecap(t *testing.T) {
	const me = "0000000000000000"
	for _, tc := range []struct {
		name, fixture, xuid string
		wantRanked          bool
	}{
		{"classé", "skill_match_ranked.json", ruleRankedXUID, true},
		{"social", "skill_match_social.json", rulePlayerXUID, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			skill := loadSkillFixture(t, tc.fixture)[tc.xuid]
			batch, err := BuildBatchFromRawForV2WithMeta(t.Context(), nil, "halo_infinite", "Player0", me,
				"aabbccdd-0000-4000-8000-0000000000b1", makeMatchJSON("aabbccdd-0000-4000-8000-0000000000b1", 2),
				map[string]*MatchSkillData{me: skill}, nil, 0, false)
			if err != nil {
				t.Fatalf("BuildBatchFromRawForV2WithMeta: %v", err)
			}
			if batch.Shared.Match == nil || batch.Shared.Match.IsRanked != tc.wantRanked {
				t.Errorf("registre is_ranked = %v, want %v", batch.Shared.Match, tc.wantRanked)
			}
			if got := len(batch.Shared.MatchCSRs) > 0; got != tc.wantRanked {
				t.Errorf("CSR partagé écrit = %v, want %v", got, tc.wantRanked)
			}
			if got := batch.PlayerData.SkillRank != nil; got != tc.wantRanked {
				t.Errorf("CSR joueur écrit = %v, want %v", got, tc.wantRanked)
			}
		})
	}
}
