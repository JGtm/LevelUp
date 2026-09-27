package squadformes

// objective_history_test.go — le rapport de force à l'objectif, soirée après soirée (lot L3,
// D6 / D7). Les parts du 07/09 sont celles de la maquette C3EW (relevé en lecture seule de la
// soirée réelle : sept matchs, quatre Bases puis trois Drapeau).

import (
	"fmt"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/narrative"
)

// matchRoles — [notre camp, lobby] pour Prendre, Défendre, Tenir (secondes).
type matchRoles struct {
	id                 string
	fam                narrative.ObjectiveFamily
	take, defend, hold [2]float64
}

// colonnes de la famille qui portent chaque rôle dans les lignes du témoin.
func roleColsOf(fam narrative.ObjectiveFamily) (take, defend, hold string) {
	if fam == narrative.FamilyCTF {
		return "flag_captures", "flag_returns", "time_as_flag_carrier_seconds"
	}
	return "zone_captures", "zone_secures", "time_in_zones_seconds"
}

// rowsOf — deux lignes par match : notre joueur (camp), l'adversaire (lobby − camp). Une
// grandeur facultative énorme est posée côté adversaire : elle ne doit peser dans aucun rôle.
func rowsOf(ms []matchRoles) ([]ObjectiveColumnRow, map[string]map[string]struct{}) {
	var rows []ObjectiveColumnRow
	camp := map[string]map[string]struct{}{}
	for _, m := range ms {
		tk, df, hd := roleColsOf(m.fam)
		rows = append(rows,
			ObjectiveColumnRow{MatchID: m.id, XUID: "moi", Family: m.fam, Values: map[string]float64{
				tk: m.take[0], df: m.defend[0], hd: m.hold[0],
			}},
			ObjectiveColumnRow{MatchID: m.id, XUID: "adv", Family: m.fam, Values: map[string]float64{
				tk: m.take[1] - m.take[0], df: m.defend[1] - m.defend[0], hd: m.hold[1] - m.hold[0],
				narrative.GrandeurFlagGrabsNet: 999,
			}},
		)
		camp[m.id] = map[string]struct{}{"moi": {}}
	}
	return rows, camp
}

// soiree0709 — les sept matchs du 07/09 (maquette C3EW, PERMATCH).
var soiree0709 = []matchRoles{
	{"b1", narrative.FamilyZonesStrongholds, [2]float64{29, 56}, [2]float64{12, 33}, [2]float64{285.7, 599.9}},
	{"b2", narrative.FamilyZonesStrongholds, [2]float64{14, 35}, [2]float64{3, 13}, [2]float64{140.8, 316.4}},
	{"b3", narrative.FamilyZonesStrongholds, [2]float64{29, 64}, [2]float64{8, 23}, [2]float64{401.3, 619.8}},
	{"b4", narrative.FamilyZonesStrongholds, [2]float64{21, 47}, [2]float64{7, 21}, [2]float64{240.0, 457.8}},
	{"d1", narrative.FamilyCTF, [2]float64{7, 20}, [2]float64{13, 38}, [2]float64{86.0, 209.3}},
	{"d2", narrative.FamilyCTF, [2]float64{3, 11}, [2]float64{7, 18}, [2]float64{32.7, 110.1}},
	{"d3", narrative.FamilyCTF, [2]float64{7, 24}, [2]float64{21, 37}, [2]float64{51.8, 196.0}},
}

func at(day, hour int) time.Time {
	return time.Date(2026, 9, day, hour, 0, 0, 0, time.UTC)
}

func pct(v *float64) float64 {
	if v == nil {
		return math.NaN()
	}
	return math.Round(*v*1000) / 10
}

// TestObjectiveHistory_Temoin0709 — les parts de la soirée affichée : Prendre 39,0 %, Défendre
// 36,8 %, Tenir 43,8 %, 1 victoire sur 7, 4 Bases et 3 Drapeau (L3.10).
func TestObjectiveHistory_Temoin0709(t *testing.T) {
	rows, camp := rowsOf(soiree0709)
	var current []HistoryMatch
	for i, m := range soiree0709 {
		current = append(current, HistoryMatch{MatchID: m.id, SessionLabel: "s0709", StartTime: at(7, 21).Add(time.Duration(i) * 10 * time.Minute), Won: m.id == "b3"})
	}
	got := BuildObjectiveHistory(HistoryInput{Current: current, Timeline: current, Rows: rows, Camp: camp})
	c := got.Current
	if pct(c.Take) != 39.0 || pct(c.Defend) != 36.8 || pct(c.Hold) != 43.8 {
		t.Fatalf("parts = %.1f / %.1f / %.1f, attendu 39,0 / 36,8 / 43,8", pct(c.Take), pct(c.Defend), pct(c.Hold))
	}
	if c.ObjectiveMatches != 7 || c.Wins != 1 || c.SessionLabel != "s0709" {
		t.Errorf("soirée = %d matchs, %d victoires, %q ; attendu 7, 1, s0709", c.ObjectiveMatches, c.Wins, c.SessionLabel)
	}
	if len(c.Families) != 2 || c.Families[0].Family != "zones_strongholds" || c.Families[0].Matches != 4 ||
		c.Families[1].Family != "ctf" || c.Families[1].Matches != 3 {
		t.Errorf("mélange de modes = %+v, attendu 4 Bases puis 3 Drapeau", c.Families)
	}
	if c.StartTime != "2026-09-07T21:00:00Z" {
		t.Errorf("début = %q", c.StartTime)
	}
	if len(got.Previous) != 0 {
		t.Errorf("la soirée affichée ne peut pas être sa propre soirée précédente : %+v", got.Previous)
	}
}

// soireeUniforme — une soirée de n matchs Drapeau à la même part, dans la session label.
func soireeUniforme(label string, day, n int, share float64) ([]HistoryMatch, []matchRoles) {
	var hm []HistoryMatch
	var mr []matchRoles
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s-%d", label, i)
		hm = append(hm, HistoryMatch{MatchID: id, SessionLabel: label, StartTime: at(day, 20).Add(time.Duration(i) * time.Minute), Won: i == 0})
		mr = append(mr, matchRoles{id, narrative.FamilyCTF, [2]float64{share * 10, 10}, [2]float64{share * 10, 10}, [2]float64{share * 100, 100}})
	}
	return hm, mr
}

// TestObjectiveHistory_SoireesPrecedentes — D6 : seules les soirées d'au moins trois matchs à
// objectif, jouées AVANT la soirée affichée, les dix plus récentes ; comptes des soirées sous
// le minimum.
func TestObjectiveHistory_SoireesPrecedentes(t *testing.T) {
	var timeline []HistoryMatch
	var all []matchRoles
	for d := 1; d <= 12; d++ { // douze soirées de 3 matchs, les 1er → 12 du mois
		hm, mr := soireeUniforme(fmt.Sprintf("s%02d", d), d, 3, 0.5)
		timeline, all = append(timeline, hm...), append(all, mr...)
	}
	court, mrc := soireeUniforme("s13", 13, 2, 0.5) // sous le minimum
	timeline, all = append(timeline, court...), append(all, mrc...)
	cur, mrcur := soireeUniforme("s20", 20, 4, 0.4) // la soirée affichée
	timeline, all = append(timeline, cur...), append(all, mrcur...)
	apres, mra := soireeUniforme("s25", 25, 3, 0.5) // jouée après : jamais « précédente »
	timeline, all = append(timeline, apres...), append(all, mra...)

	rows, camp := rowsOf(all)
	got := BuildObjectiveHistory(HistoryInput{Current: cur, Timeline: timeline, Rows: rows, Camp: camp})
	if len(got.Previous) != 10 {
		t.Fatalf("%d soirées précédentes, attendu 10", len(got.Previous))
	}
	if got.Previous[0].SessionLabel != "s03" || got.Previous[9].SessionLabel != "s12" {
		t.Errorf("soirées retenues %q → %q, attendu s03 → s12 (les dix plus récentes, chronologiques)",
			got.Previous[0].SessionLabel, got.Previous[9].SessionLabel)
	}
	if got.EveningsWithObjective != 15 || got.EveningsBelowMinimum != 1 {
		t.Errorf("soirées avec objectif = %d, sous le minimum = %d ; attendu 15 et 1",
			got.EveningsWithObjective, got.EveningsBelowMinimum)
	}
	if pct(got.Current.Take) != 40.0 || got.Current.ObjectiveMatches != 4 || got.MinObjectiveMatches != 3 {
		t.Errorf("soirée affichée = %+v", got.Current)
	}
}

// TestObjectiveHistory_ModeEcarteEtCampInconnu — un match marqué écarté (drapeau neutre) ou sans
// camp connu ne compte pas : la soirée tombe sous le minimum et n'a pas de point.
func TestObjectiveHistory_ModeEcarteEtCampInconnu(t *testing.T) {
	hm, mr := soireeUniforme("s01", 1, 4, 0.5)
	hm[0].Excluded = true
	rows, camp := rowsOf(mr)
	delete(camp, hm[1].MatchID)
	cur, mrc := soireeUniforme("s05", 5, 3, 0.5)
	rc, cc := rowsOf(mrc)
	for k, v := range cc {
		camp[k] = v
	}
	got := BuildObjectiveHistory(HistoryInput{Current: cur, Timeline: append(hm, cur...), Rows: append(rows, rc...), Camp: camp})
	if len(got.Previous) != 0 || got.EveningsBelowMinimum != 1 {
		t.Errorf("précédentes = %+v, sous le minimum = %d ; attendu aucune et 1", got.Previous, got.EveningsBelowMinimum)
	}
}

// TestObjectiveHistory_RoleSansMesure — un rôle que le lobby n'a pas touché ne pèse pas dans la
// moyenne (0/0 n'est pas 0 %) ; une soirée sans aucun rôle mesuré publie nil.
func TestObjectiveHistory_RoleSansMesure(t *testing.T) {
	ms := []matchRoles{
		{"m1", narrative.FamilyCTF, [2]float64{1, 2}, [2]float64{0, 0}, [2]float64{10, 20}},
		{"m2", narrative.FamilyCTF, [2]float64{0, 2}, [2]float64{3, 4}, [2]float64{0, 20}},
	}
	rows, camp := rowsOf(ms)
	cur := []HistoryMatch{{MatchID: "m1", StartTime: at(1, 20)}, {MatchID: "m2", StartTime: at(1, 21)}}
	got := BuildObjectiveHistory(HistoryInput{Current: cur, Rows: rows, Camp: camp}).Current
	if pct(got.Take) != 25.0 || pct(got.Defend) != 75.0 || pct(got.Hold) != 25.0 {
		t.Errorf("parts = %.1f / %.1f / %.1f, attendu 25 / 75 (m2 seul) / 25", pct(got.Take), pct(got.Defend), pct(got.Hold))
	}
	if got.SessionLabel != "" {
		t.Errorf("soirée sans session : libellé %q, attendu vide", got.SessionLabel)
	}
}

// TestObjectiveHistory_FiltrePartiel_LaSoireeAfficheeNEstJamaisPrecedente — constat R10 de la
// revue L6.1 : le filtre écarte le PREMIER match de la soirée affichée, qui commence donc avant le
// premier match du périmètre ; elle ne doit pas se retrouver parmi ses propres soirées précédentes.
func TestObjectiveHistory_FiltrePartiel_LaSoireeAfficheeNEstJamaisPrecedente(t *testing.T) {
	avant, mra := soireeUniforme("s10", 10, 3, 0.5)
	soir, mrs := soireeUniforme("s20", 20, 4, 0.4)
	rows, camp := rowsOf(append(mra, mrs...))
	timeline := append(append([]HistoryMatch(nil), avant...), soir...)
	got := BuildObjectiveHistory(HistoryInput{Current: soir[1:], Timeline: timeline, Rows: rows, Camp: camp})
	for _, e := range got.Previous {
		if e.SessionLabel == "s20" {
			t.Fatalf("la soirée affichée s20 figure parmi les soirées précédentes : %+v", got.Previous)
		}
	}
	if len(got.Previous) != 1 || got.Previous[0].SessionLabel != "s10" {
		t.Errorf("soirées précédentes = %+v, attendu la seule s10", got.Previous)
	}
}
