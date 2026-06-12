package observability

import "testing"

func TestPlayerAPICollector_AggregatesAndAvg(t *testing.T) {
	c := newPlayerAPICollector(8)
	c.record("match_history", "JGtm", 100, false)
	c.record("match_history", "JGtm", 300, true) // 1 erreur
	c.record("player_csrs", "JGtm", 50, false)

	snap := c.snapshot()
	if len(snap) != 2 {
		t.Fatalf("entries = %d (attendu 2)", len(snap))
	}
	// Trié erreurs desc → match_history (1 err) avant player_csrs (0).
	mh := snap[0]
	if mh.Call != "match_history" || mh.Player != "JGtm" {
		t.Fatalf("snap[0] = %s/%s (attendu match_history/JGtm)", mh.Call, mh.Player)
	}
	if mh.Count != 2 || mh.SumMs != 400 || mh.AvgMs != 200 || mh.MaxMs != 300 || mh.Errors != 1 {
		t.Errorf("mh = count %d sum %d avg %d max %d err %d (attendu 2/400/200/300/1)",
			mh.Count, mh.SumMs, mh.AvgMs, mh.MaxMs, mh.Errors)
	}
}

func TestPlayerAPICollector_IgnoresEmpty(t *testing.T) {
	c := newPlayerAPICollector(8)
	c.record("", "JGtm", 100, false)          // call vide
	c.record("match_history", "", 100, false) // player vide (match-level)
	if len(c.snapshot()) != 0 {
		t.Errorf("entries = %d (attendu 0 : appels non attribuables ignorés)", len(c.snapshot()))
	}
}

func TestPlayerAPICollector_EvictsLeastActive(t *testing.T) {
	c := newPlayerAPICollector(2)
	c.record("career_rank", "A", 10, false) // count 1
	c.record("career_rank", "B", 10, false) // count 1
	c.record("career_rank", "B", 10, false) // B count 2
	// 3e clé distincte → évince A (count le plus faible).
	c.record("career_rank", "C", 10, false)

	snap := c.snapshot()
	if len(snap) != 2 {
		t.Fatalf("entries = %d (attendu 2, cap)", len(snap))
	}
	for _, s := range snap {
		if s.Player == "A" {
			t.Errorf("A aurait dû être évincé (moins actif)")
		}
	}
}

func TestRecordPlayerAPICall_Singleton(t *testing.T) {
	ResetPlayerAPIStats()
	t.Cleanup(ResetPlayerAPIStats)
	RecordPlayerAPICall("playlist_csr", "Madina", 42, true)
	snap := PlayerAPIStats()
	if len(snap) != 1 || snap[0].Errors != 1 || snap[0].AvgMs != 42 {
		t.Fatalf("singleton snapshot inattendu : %+v", snap)
	}
}
