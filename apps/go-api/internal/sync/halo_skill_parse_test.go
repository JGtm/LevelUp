// Package sync — halo_skill_parse_test.go : tests unitaires (sans IO) du parser
// public ParseMatchSkillResponseJSON, qui décode le payload skill Halo stocké
// verbatim par OpenSpartan dans PlayerMatchStats.ResponseBody (récupération CSR
// par-match hors ligne).
package sync

import "testing"

// skillBodyWithRankRecap est un payload skill Halo minimal (même forme que la
// table OpenSpartan PlayerMatchStats.ResponseBody) avec un RankRecap classé.
const skillBodyWithRankRecap = `{
  "Value": [
    {
      "Id": "xuid(2533274823110022)",
      "ResultCode": 0,
      "Result": {
        "TeamMmr": 1234.5,
        "RankRecap": {
          "PreMatchCsr":  {"Value": 1200, "Tier": "Gold", "SubTier": 1, "MeasurementMatchesRemaining": 0},
          "PostMatchCsr": {"Value": 1225, "Tier": "Gold", "SubTier": 2, "MeasurementMatchesRemaining": 0}
        }
      }
    }
  ]
}`

func TestParseMatchSkillResponseJSON_RankRecap(t *testing.T) {
	m, err := ParseMatchSkillResponseJSON([]byte(skillBodyWithRankRecap))
	if err != nil {
		t.Fatalf("ParseMatchSkillResponseJSON: %v", err)
	}
	sd, ok := m["2533274823110022"]
	if !ok {
		t.Fatalf("xuid manquant dans la map (clés: %v)", skillMapKeys(m))
	}
	if sd.TeamMMR == nil || *sd.TeamMMR != 1234.5 {
		t.Errorf("TeamMMR: want 1234.5, got %v", sd.TeamMMR)
	}
	if sd.PostMatchCSR == nil {
		t.Fatalf("PostMatchCSR nil — RankRecap non décodé")
	}
	if sd.PostMatchCSR.Value != 1225 || sd.PostMatchCSR.Tier != "Gold" || sd.PostMatchCSR.SubTier != 2 {
		t.Errorf("PostMatchCSR: want {1225 Gold 2}, got %+v", *sd.PostMatchCSR)
	}
	if sd.PreMatchCSR == nil || sd.PreMatchCSR.Value != 1200 {
		t.Errorf("PreMatchCSR: want Value=1200, got %+v", sd.PreMatchCSR)
	}
}

func TestParseMatchSkillResponseJSON_EmptyBody(t *testing.T) {
	m, err := ParseMatchSkillResponseJSON(nil)
	if err != nil {
		t.Fatalf("empty body should be no-op, got %v", err)
	}
	if len(m) != 0 {
		t.Errorf("want empty map, got %d entries", len(m))
	}
}

func TestParseMatchSkillResponseJSON_Malformed(t *testing.T) {
	if _, err := ParseMatchSkillResponseJSON([]byte("{ not json")); err == nil {
		t.Error("want error on malformed JSON, got nil")
	}
}

func skillMapKeys(m map[string]*MatchSkillData) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
