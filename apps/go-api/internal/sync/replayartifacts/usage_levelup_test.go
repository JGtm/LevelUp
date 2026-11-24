package replayartifacts

// usage_levelup_test.go — la jointure des mesures de LevelUp aux lignes du résumé d'usage : ce qui
// est mesuré (zéro compris), ce qui reste NULL, les lignes ajoutées. Les règles de bris, de
// recharge et de vengeance se testent dans leurs paquets ; ici on teste ce que la jointure en fait.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func sh(v float32) *float32 { return &v }

// docBouclier : A casse puis recharge son bouclier, B le casse sans le recharger, C n'a aucune
// mesure (bouclier jamais bougé), D n'a aucune piste.
func docBouclier() *replay.ReplayDocument {
	return &replay.ReplayDocument{Tracks: []replay.Track{
		{XUID: "A", Points: []replay.Point{{T: 1, Sh: sh(1)}, {T: 2, Sh: sh(0)}, {T: 3, Sh: sh(1)}}},
		{XUID: "B", Points: []replay.Point{{T: 1, Sh: sh(1)}, {T: 2, Sh: sh(0)}}},
		{XUID: "C", Points: []replay.Point{{T: 1}}},
	}}
}

func valeur(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}

func TestAttacherMesuresLevelUp_BouclierMesureZeroCompris(t *testing.T) {
	s := &replay.UsageSummary{Players: []replay.UsagePlayerSummary{{XUID: "A"}, {XUID: "C"}, {XUID: "D"}}}
	lu := AttacherMesuresLevelUp(s, docBouclier(), EntreesMesuresUsage{})

	if lu.Rev != "" {
		t.Fatalf("Rev = %q, attendu vide : le fil des morts et les équipes ne sont pas lus", lu.Rev)
	}
	a, c, d := lu.Players["A"], lu.Players["C"], lu.Players["D"]
	if valeur(a.ShieldBreaks) != 1 || valeur(a.ShieldRecharges) != 1 {
		t.Errorf("A = %d bris / %d recharges, attendu 1 / 1", valeur(a.ShieldBreaks), valeur(a.ShieldRecharges))
	}
	if valeur(c.ShieldBreaks) != 0 || valeur(c.ShieldRecharges) != 0 {
		t.Errorf("C (piste sans mesure dans un match mesure) = %d / %d, attendu 0 / 0",
			valeur(c.ShieldBreaks), valeur(c.ShieldRecharges))
	}
	if d.ShieldBreaks != nil || d.ShieldRecharges != nil {
		t.Errorf("D (aucune piste) doit rester NULL, lu %v / %v", d.ShieldBreaks, d.ShieldRecharges)
	}
	if a.RevengeKills != nil {
		t.Errorf("sans fil des morts lu, les vengeances restent NULL, lu %v", *a.RevengeKills)
	}
	// B porte un bris mais n'avait pas de ligne : elle est ajoutee, avec ses mesures.
	b, ok := lu.Players["B"]
	if !ok || valeur(b.ShieldBreaks) != 1 || valeur(b.ShieldRecharges) != 0 {
		t.Fatalf("B = %+v (present %v), attendu une ligne ajoutee a 1 bris / 0 recharge", b, ok)
	}
	xuids := []string{}
	for _, p := range s.Players {
		xuids = append(xuids, p.XUID)
	}
	if len(xuids) != 4 || xuids[0] != "A" || xuids[1] != "B" || xuids[2] != "C" || xuids[3] != "D" {
		t.Errorf("lignes du resume = %v, attendu A B C D (triees)", xuids)
	}
}

func TestAttacherMesuresLevelUp_MatchSansMesureDeBouclier(t *testing.T) {
	doc := &replay.ReplayDocument{Tracks: []replay.Track{{XUID: "A", Points: []replay.Point{{T: 1}}}}}
	s := &replay.UsageSummary{Players: []replay.UsagePlayerSummary{{XUID: "A"}}}
	lu := AttacherMesuresLevelUp(s, doc, EntreesMesuresUsage{})
	if a := lu.Players["A"]; a.ShieldBreaks != nil || a.ShieldRecharges != nil {
		t.Errorf("un match sans aucune mesure Sh est non mesure (NULL), lu %v / %v", a.ShieldBreaks, a.ShieldRecharges)
	}
	if len(s.Players) != 1 {
		t.Errorf("aucune ligne ne doit etre ajoutee, %d lignes", len(s.Players))
	}
}

func TestAttacherMesuresLevelUp_Vengeances(t *testing.T) {
	doc := &replay.ReplayDocument{}
	equipes := map[string]int{"A": 0, "B": 1, "C": 0}
	frags := replay.ShieldlessKillsInput{Read: true, Kills: []replay.ShieldlessKillRef{
		{Killer: "B", Victim: "A", TimeMS: 1000}, // B tue A
		{Killer: "A", Victim: "B", TimeMS: 2000}, // A venge sa mort
		{Killer: "C", Victim: "A", TimeMS: 3000}, // trahison : ne se compte pas
	}}
	s := &replay.UsageSummary{Players: []replay.UsagePlayerSummary{{XUID: "A"}, {XUID: "B"}}}
	lu := AttacherMesuresLevelUp(s, doc, EntreesMesuresUsage{Frags: frags, Equipes: equipes})
	if valeur(lu.Players["A"].RevengeKills) != 1 {
		t.Errorf("A = %d vengeance, attendu 1", valeur(lu.Players["A"].RevengeKills))
	}
	if valeur(lu.Players["B"].RevengeKills) != 0 {
		t.Errorf("B = %d vengeance, attendu 0 (mesure, zero)", valeur(lu.Players["B"].RevengeKills))
	}

	// Fil des morts lu mais equipes absentes : les trahisons ne s'ecarteraient pas, NULL.
	lu = AttacherMesuresLevelUp(s, doc, EntreesMesuresUsage{Frags: frags})
	if lu.Players["A"].RevengeKills != nil {
		t.Errorf("sans equipes lues, les vengeances restent NULL")
	}
	// Equipes lues mais fil des morts non lu : NULL aussi.
	lu = AttacherMesuresLevelUp(s, doc, EntreesMesuresUsage{Equipes: equipes})
	if lu.Players["A"].RevengeKills != nil {
		t.Errorf("sans fil des morts lu, les vengeances restent NULL")
	}
}

func TestAttacherMesuresLevelUp_VengeurSansLigneRecoitUneLigne(t *testing.T) {
	frags := replay.ShieldlessKillsInput{Read: true, Kills: []replay.ShieldlessKillRef{
		{Killer: "B", Victim: "A", TimeMS: 1000},
		{Killer: "A", Victim: "B", TimeMS: 2000},
	}}
	s := &replay.UsageSummary{}
	lu := AttacherMesuresLevelUp(s, &replay.ReplayDocument{}, EntreesMesuresUsage{
		Frags: frags, Equipes: map[string]int{"A": 0, "B": 1}})
	if len(s.Players) != 1 || s.Players[0].XUID != "A" || valeur(lu.Players["A"].RevengeKills) != 1 {
		t.Fatalf("lignes = %+v, mesures = %+v : attendu une ligne A a 1 vengeance", s.Players, lu.Players)
	}
}

func TestAttacherMesuresLevelUp_EntreesNilesNeProduisentRien(t *testing.T) {
	if lu := AttacherMesuresLevelUp(nil, &replay.ReplayDocument{}, EntreesMesuresUsage{}); lu.Rev != "" || lu.Players != nil {
		t.Errorf("resume nil : %+v", lu)
	}
	if lu := AttacherMesuresLevelUp(&replay.UsageSummary{}, nil, EntreesMesuresUsage{}); lu.Rev != "" || lu.Players != nil {
		t.Errorf("document nil : %+v", lu)
	}
}

// La révision du calcul ne se pose que si le fil des morts ET les équipes ont été lus : sans eux les
// vengeances ne sont pas mesurées, et poser la révision ferait sauter le match au rattrapage, qui le
// croirait à jour pour toujours.
func TestAttacherMesuresLevelUp_RevisionSeulementSiLecturesCompletes(t *testing.T) {
	complet := EntreesMesuresUsage{
		Frags:   replay.ShieldlessKillsInput{Read: true},
		Equipes: map[string]int{"A": 0, "B": 1},
	}
	cas := []struct {
		nom string
		in  EntreesMesuresUsage
		rev string
	}{
		{"lectures complètes", complet, UsageLevelUpRev},
		{"fil des morts non lu", EntreesMesuresUsage{Equipes: complet.Equipes}, ""},
		{"équipes non lues", EntreesMesuresUsage{Frags: complet.Frags}, ""},
		{"rien de lu", EntreesMesuresUsage{}, ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			s := &replay.UsageSummary{Players: []replay.UsagePlayerSummary{{XUID: "A"}}}
			if lu := AttacherMesuresLevelUp(s, docBouclier(), c.in); lu.Rev != c.rev {
				t.Fatalf("Rev = %q, attendu %q", lu.Rev, c.rev)
			}
		})
	}
}
