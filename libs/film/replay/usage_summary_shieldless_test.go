package replay

import "testing"

func shieldlessTestDoc() *ReplayDocument {
	origin := int64(0)
	sh := func(v float32) *float32 { return &v }
	pts := func(n int, mesures map[int]float32) []Point {
		out := make([]Point, 0, n)
		for t := 0; t < n; t++ {
			p := Point{T: t}
			if v, ok := mesures[t]; ok {
				p.Sh = sh(v)
			}
			out = append(out, p)
		}
		return out
	}
	return &ReplayDocument{
		OriginMs: &origin, FrameIntervalMS: 100,
		Tracks: []Track{
			{Slot: 1, XUID: "zero", Points: pts(100, map[int]float32{5: 0.4, 10: 0})},
			{Slot: 2, XUID: "plein", Points: pts(100, nil)},
			{Slot: 3, XUID: "entame", Points: pts(100, map[int]float32{10: 0.3})},
			{Slot: 4, XUID: "ancien", Points: pts(100, map[int]float32{0: 0})},
			{Slot: 5, XUID: "echange", Points: pts(100, map[int]float32{10: 0})},
		},
	}
}

func TestAttachShieldlessKills(t *testing.T) {
	doc := shieldlessTestDoc()
	kills := []ShieldlessKillRef{
		{Killer: "zero", Victim: "v1", TimeMS: 2000},    // mesure a 0 vieille d 1 s, survit : compte
		{Killer: "plein", Victim: "v2", TimeMS: 2000},   // jamais touche : bouclier plein
		{Killer: "entame", Victim: "v3", TimeMS: 2000},  // bouclier entame, pas a 0
		{Killer: "ancien", Victim: "v4", TimeMS: 6000},  // zero vieux de 6 s : ecarte
		{Killer: "echange", Victim: "v5", TimeMS: 2000}, // meurt 500 ms apres : ne survit pas
		{Killer: "v6", Victim: "echange", TimeMS: 2500},
		{Killer: "zero", Victim: "zero", TimeMS: 9000}, // suicide : jamais un frag
	}
	s := UsageSummary{Players: []UsagePlayerSummary{{XUID: "entame"}, {XUID: "zero"}}}
	AttachShieldlessKills(&s, doc, ShieldlessKillsInput{Read: true, Kills: kills})
	got := map[string]int{}
	for _, p := range s.Players {
		if p.ShieldlessKills == nil {
			t.Fatalf("%s : compte absent alors que la mesure a eu lieu", p.XUID)
		}
		got[p.XUID] = *p.ShieldlessKills
	}
	if got["zero"] != 1 || got["entame"] != 0 || len(got) != 2 {
		t.Fatalf("comptes inattendus : %v", got)
	}
}

func TestAttachShieldlessKills_NonMesure(t *testing.T) {
	doc := shieldlessTestDoc()
	kills := []ShieldlessKillRef{{Killer: "zero", Victim: "v1", TimeMS: 2000}}
	s := UsageSummary{Players: []UsagePlayerSummary{{XUID: "zero"}}}
	AttachShieldlessKills(&s, doc, ShieldlessKillsInput{Read: false, Kills: kills})
	if s.Players[0].ShieldlessKills != nil {
		t.Fatal("fil des morts non lu : le compte doit rester absent")
	}
	doc.OriginMs = nil
	AttachShieldlessKills(&s, doc, ShieldlessKillsInput{Read: true, Kills: kills})
	if s.Players[0].ShieldlessKills != nil {
		t.Fatal("document sans origine : le compte doit rester absent")
	}
}

func TestAttachShieldlessKills_LigneNeuveTriee(t *testing.T) {
	doc := shieldlessTestDoc()
	s := UsageSummary{Players: []UsagePlayerSummary{{XUID: "a"}}}
	AttachShieldlessKills(&s, doc, ShieldlessKillsInput{Read: true, Kills: []ShieldlessKillRef{
		{Killer: "zero", Victim: "v1", TimeMS: 2000},
	}})
	if len(s.Players) != 2 || s.Players[0].XUID != "a" || s.Players[1].XUID != "zero" || *s.Players[1].ShieldlessKills != 1 {
		t.Fatalf("ligne du tueur attendue, triee : %+v", s.Players)
	}
}
