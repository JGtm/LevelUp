package replay

// zone_states_hill_gauge_test.go — LA JAUGE D UNE COLLINE (zone_states_hill_gauge.go), sur des
// enregistrements construits a la forme mesuree : une prise par le camp 0 (montee de 0,1 par frame,
// pousseur 0), une vidange (saut pres de 1 puis descente, pousseur neutre), une reprise par le camp
// 1. Le bloc tient par voisinage : designateur 40, proprietaire 41, pousseur 42, jauge 43.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// hillGaugeReads fabrique la jauge (slot 43) d une montee de `t0` a `t0+9` par pas de 0,1, suivie
// du retour a zero a `t0+10`.
func hillGaugeRise(t0 int) []grammar.ManagedPropertyRead {
	var out []grammar.ManagedPropertyRead
	for k := 0; k < 10; k++ {
		out = append(out, zoneReadAt(43, t0+k, grammar.ManagedPropertyTagQuant, gaugeQ(uint64(80+100*k))))
	}
	return append(out, zoneReadAt(43, t0+10, grammar.ManagedPropertyTagQuant, gaugeQ(0)))
}

// hillGaugeDrain fabrique une vidange : 0,98 a `t0` puis -0,1 par frame jusqu a zero.
func hillGaugeDrain(t0 int) []grammar.ManagedPropertyRead {
	var out []grammar.ManagedPropertyRead
	for k := 0; k < 10; k++ {
		out = append(out, zoneReadAt(43, t0+k, grammar.ManagedPropertyTagQuant, gaugeQ(uint64(980-100*k))))
	}
	return append(out, zoneReadAt(43, t0+10, grammar.ManagedPropertyTagQuant, gaugeQ(0)))
}

// TestCollineJaugePriseVidangeReprise — la serie monte, se vide et remonte ; les segments nomment
// le camp de chaque prise et marquent la vidange.
func TestCollineJaugePriseVidangeReprise(t *testing.T) {
	u32 := grammar.ManagedPropertyTagU32
	reads := []grammar.ManagedPropertyRead{
		zoneChainedReadAt(40, 400, grammar.ManagedPropertyTagStringID, 0x78F81557),
		zoneReadAt(41, 60, u32, zoneNeutralOwner), zoneReadAt(41, 110, u32, 0),
		zoneReadAt(41, 210, u32, zoneNeutralOwner), zoneReadAt(41, 230, u32, 1),
		zoneReadAt(42, 100, u32, 0), zoneReadAt(42, 110, u32, zoneNeutralOwner),
		zoneReadAt(42, 220, u32, 1), zoneReadAt(42, 230, u32, zoneNeutralOwner),
	}
	reads = append(reads, hillGaugeRise(100)...)
	reads = append(reads, hillGaugeDrain(200)...)
	reads = append(reads, hillGaugeRise(220)...)
	in := zoneTestInput(reads)
	in.Hill = true
	gardien := Track{XUID: "2533", Team: 0, Points: pointsIn(60, 399, 20.5)}
	autre := Track{XUID: "2535", Team: 1, Points: pointsIn(200, 399, 20.5)}
	states, cov := buildZoneStates(context.Background(), in, zoneTestCtx(nil, []Track{gardien, autre}))
	if len(states) != 1 {
		t.Fatalf("%d colline(s) publiee(s), attendu 1 : %+v", len(states), states)
	}
	st := states[0]
	if cov.GaugePoints == 0 || len(st.Gauge) == 0 {
		t.Fatal("aucun point de jauge publie sur la colline")
	}
	if top := maxGauge(st.Gauge, 100, 109); top < 0.95 {
		t.Errorf("sommet de la prise %.3f, attendu >= 0,95", top)
	}
	if v := gaugeAt(st.Gauge, 205); v <= 0 || v >= 0.98 {
		t.Errorf("jauge a mi-vidange %.3f, attendu dans ]0 ; 0,98[ (la vidange se publie)", v)
	}
	veut := []struct {
		t0       int
		team     int
		draining bool
	}{{100, 0, false}, {200, -1, true}, {220, 1, false}}
	if len(st.GaugeRamps) != len(veut) {
		t.Fatalf("%d segment(s), attendu %d : %+v", len(st.GaugeRamps), len(veut), st.GaugeRamps)
	}
	for i, w := range veut {
		r := st.GaugeRamps[i]
		team := -1
		if r.CapturingTeam != nil {
			team = *r.CapturingTeam
		}
		if r.T0 != w.t0 || team != w.team || r.Draining != w.draining {
			t.Errorf("segment %d : %+v (camp %d), attendu t0 %d camp %d vidange %v", i, r, team, w.t0, w.team, w.draining)
		}
	}
}

// maxGauge rend la plus haute valeur publiee dans [t0, t1].
func maxGauge(g []GaugePoint, t0, t1 int) float32 {
	var top float32
	for _, p := range g {
		if p.T >= t0 && p.T <= t1 && p.V > top {
			top = p.V
		}
	}
	return top
}

// gaugeAt rend la valeur de l escalier a la frame t.
func gaugeAt(g []GaugePoint, t int) float32 {
	var v float32
	for _, p := range g {
		if p.T > t {
			break
		}
		v = p.V
	}
	return v
}
