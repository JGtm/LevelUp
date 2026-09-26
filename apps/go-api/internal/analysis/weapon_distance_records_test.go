// Package analysis — weapon_distance_records_test.go : le frag le plus lointain par arme.
//
// Ce que ces tests verrouillent :
//
//  1. le record est LE frag (clé conservée), pas seulement une distance ;
//  2. la médiane et l'effectif décrivent la même population que le record, sans seuil ;
//  3. l'ordre publié est le record croissant, départagé par la clé d'arme — indépendant de
//     l'ordre d'arrivée des lignes ;
//  4. le départage du record à distance égale est déterministe ;
//  5. le côté demandé filtre, une clé vide est ignorée, l'entrée n'est pas mutée.
package analysis

import (
	"math"
	"reflect"
	"testing"
)

const epsRecord = 1e-9

func recKill(weapon, match string, timeMS int64, dist float64) MeasuredKill {
	return MeasuredKill{
		MatchID: match, KillerXUID: "me", TimeMS: timeMS,
		WeaponKey: weapon, Side: SideKiller, DistanceM: dist,
	}
}

func TestWeaponDistanceRecords_Vide(t *testing.T) {
	if out := WeaponDistanceRecords(nil, SideKiller); len(out) != 0 {
		t.Fatalf("attendu aucune ligne, obtenu %+v", out)
	}
}

// Un seul frag : record == médiane == la distance, effectif 1. Aucun seuil ne l'écarte.
func TestWeaponDistanceRecords_UnFrag_EstUnRecord(t *testing.T) {
	out := WeaponDistanceRecords([]MeasuredKill{recKill("hinf_hydra", "m1", 42, 31.7)}, SideKiller)
	if len(out) != 1 {
		t.Fatalf("attendu 1 ligne, obtenu %+v", out)
	}
	r := out[0]
	if r.Measured != 1 || math.Abs(r.RecordM-31.7) > epsRecord || math.Abs(r.MedianM-31.7) > epsRecord {
		t.Errorf("ligne = %+v, attendu 1 mesure, record et médiane à 31,7 m", r)
	}
	if r.RecordMatchID != "m1" || r.RecordTimeMS != 42 || r.RecordKillerXUID != "me" {
		t.Errorf("clé du record = %s/%d/%s, attendu m1/42/me", r.RecordMatchID, r.RecordTimeMS, r.RecordKillerXUID)
	}
}

// Plusieurs armes : le record garde SA clé, la médiane est celle de type 7 sur les frags de
// l'arme, et les lignes sortent par record croissant quel que soit l'ordre d'entrée.
func TestWeaponDistanceRecords_PlusieursArmes_TriRecordCroissant(t *testing.T) {
	kills := []MeasuredKill{
		recKill("hinf_s7_sniper", "m2", 10, 28.3),
		recKill("hinf_br75", "m1", 5, 12.0),
		recKill("hinf_s7_sniper", "m3", 77, 96.4), // le record du sniper
		recKill("hinf_br75", "m4", 9, 52.7),       // le record du BR
		recKill("hinf_br75", "m1", 6, 14.0),
		recKill("hinf_sidekick", "m1", 1, 7.0),
	}
	out := WeaponDistanceRecords(kills, SideKiller)
	got := make([]string, 0, len(out))
	for _, r := range out {
		got = append(got, r.WeaponKey)
	}
	want := []string{"hinf_sidekick", "hinf_br75", "hinf_s7_sniper"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordre = %v, attendu %v (record croissant)", got, want)
	}
	br := out[1]
	if br.Measured != 3 || math.Abs(br.RecordM-52.7) > epsRecord || br.RecordMatchID != "m4" || br.RecordTimeMS != 9 {
		t.Errorf("BR = %+v, attendu 3 mesures, record 52,7 m en m4 à t=9", br)
	}
	// Médiane des trois distances 12, 14, 52,7 = 14 (type 7, rang central exact).
	if math.Abs(br.MedianM-14.0) > epsRecord {
		t.Errorf("médiane BR = %v, attendu 14", br.MedianM)
	}
	sn := out[2]
	if sn.RecordMatchID != "m3" || math.Abs(sn.RecordM-96.4) > epsRecord || sn.Measured != 2 {
		t.Errorf("sniper = %+v, attendu record 96,4 m en m3 sur 2 mesures", sn)
	}
}

// À record égal, la clé d'arme départage : l'ordre ne dépend pas de l'itération d'une map.
func TestWeaponDistanceRecords_RecordEgal_DepartageParCle(t *testing.T) {
	kills := []MeasuredKill{
		recKill("hinf_needler", "m1", 1, 19.4),
		recKill("hinf_disruptor", "m1", 2, 19.4),
	}
	for i := 0; i < 20; i++ {
		out := WeaponDistanceRecords(kills, SideKiller)
		if out[0].WeaponKey != "hinf_disruptor" || out[1].WeaponKey != "hinf_needler" {
			t.Fatalf("itération %d : ordre = %s, %s — attendu disruptor puis needler",
				i, out[0].WeaponKey, out[1].WeaponKey)
		}
	}
}

// Deux frags de la même arme à distance STRICTEMENT égale : le record est le plus petit dans
// la clé (match_id, time_ms), et il l'est dans les deux ordres d'entrée.
func TestWeaponDistanceRecords_DistanceEgale_RecordDeterministe(t *testing.T) {
	a := recKill("hinf_br75", "m2", 50, 40.0)
	b := recKill("hinf_br75", "m1", 90, 40.0)
	c := recKill("hinf_br75", "m1", 30, 40.0)
	for _, ordre := range [][]MeasuredKill{{a, b, c}, {c, b, a}, {b, a, c}} {
		r := WeaponDistanceRecords(ordre, SideKiller)[0]
		if r.RecordMatchID != "m1" || r.RecordTimeMS != 30 {
			t.Errorf("record = %s/%d, attendu m1/30 (le plus petit dans la clé)", r.RecordMatchID, r.RecordTimeMS)
		}
		if r.Measured != 3 {
			t.Errorf("effectif = %d, attendu 3", r.Measured)
		}
	}
}

// Le côté filtre : les morts (arme du tueur) ne font pas de record de frag, et inversement.
func TestWeaponDistanceRecords_CoteFiltre(t *testing.T) {
	victim := recKill("hinf_s7_sniper", "m1", 1, 118.3)
	victim.Side = SideVictim
	kills := []MeasuredKill{victim, recKill("hinf_br75", "m1", 2, 20.0)}

	out := WeaponDistanceRecords(kills, SideKiller)
	if len(out) != 1 || out[0].WeaponKey != "hinf_br75" {
		t.Errorf("côté tueur = %+v, attendu le seul BR", out)
	}
	out = WeaponDistanceRecords(kills, SideVictim)
	if len(out) != 1 || out[0].WeaponKey != "hinf_s7_sniper" || math.Abs(out[0].RecordM-118.3) > epsRecord {
		t.Errorf("côté victime = %+v, attendu le seul sniper à 118,3 m", out)
	}
}

// Une clé vide n'est pas une arme.
func TestWeaponDistanceRecords_CleVideIgnoree(t *testing.T) {
	kills := []MeasuredKill{recKill("", "m1", 1, 99), recKill("hinf_br75", "m1", 2, 20)}
	out := WeaponDistanceRecords(kills, SideKiller)
	if len(out) != 1 || out[0].WeaponKey != "hinf_br75" {
		t.Errorf("lignes = %+v, attendu le seul BR", out)
	}
}

// Pureté : l'entrée ressort identique.
func TestWeaponDistanceRecords_EntreeNonMutee(t *testing.T) {
	kills := []MeasuredKill{
		recKill("hinf_br75", "m1", 3, 30),
		recKill("hinf_br75", "m1", 1, 10),
		recKill("hinf_s7_sniper", "m1", 2, 20),
	}
	avant := append([]MeasuredKill(nil), kills...)
	_ = WeaponDistanceRecords(kills, SideKiller)
	if !reflect.DeepEqual(kills, avant) {
		t.Errorf("entrée mutée : %+v, attendu %+v", kills, avant)
	}
}
