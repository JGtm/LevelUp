package sync

import "testing"

// TestParseSeasonPlaylistServiceRecord valide le décodage de l'agrégat CoreStats du
// service record filtré (season+playlist) : sommation des médailles, TimePlayed en
// secondes, et (nil,nil) si zéro match (jamais jouée / record vide).
func TestParseSeasonPlaylistServiceRecord(t *testing.T) {
	body := []byte(`{
		"MatchesCompleted": 10, "Wins": 6, "Losses": 4, "TimePlayed": "PT1H",
		"CoreStats": {
			"Kills": 100, "Deaths": 80, "Assists": 20,
			"ShotsFired": 500, "ShotsHit": 250,
			"DamageDealt": 12000, "DamageTaken": 9000,
			"Medals": [{"Count": 3}, {"Count": 2}]
		}
	}`)
	r, err := parseSeasonPlaylistServiceRecord(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r == nil {
		t.Fatal("attendu un record non nil")
	}
	if r.MatchesCompleted != 10 || r.Wins != 6 || r.Losses != 4 {
		t.Errorf("compteurs = %d/%d/%d, want 10/6/4", r.MatchesCompleted, r.Wins, r.Losses)
	}
	if r.Kills != 100 || r.Deaths != 80 || r.Assists != 20 {
		t.Errorf("KDA bruts = %d/%d/%d, want 100/80/20", r.Kills, r.Deaths, r.Assists)
	}
	if r.MedalCount != 5 {
		t.Errorf("MedalCount = %d, want 5 (3+2)", r.MedalCount)
	}
	if r.TimePlayedSec != 3600 {
		t.Errorf("TimePlayedSec = %d, want 3600 (PT1H)", r.TimePlayedSec)
	}
	if r.ShotsHit != 250 || r.ShotsFired != 500 {
		t.Errorf("shots = %.0f/%.0f, want 250/500", r.ShotsHit, r.ShotsFired)
	}

	// Zéro match → (nil, nil) : le joueur est conservé sans stats.
	empty, err := parseSeasonPlaylistServiceRecord([]byte(`{"MatchesCompleted": 0}`))
	if err != nil || empty != nil {
		t.Errorf("0 match = (%v, %v), want (nil, nil)", empty, err)
	}
}
