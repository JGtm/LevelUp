package main

// decide_test.go — la règle d'écriture, éprouvée sur les témoins RÉELS du corpus mesuré le
// 2026-08-29 (`.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`). Aucun réseau, aucune base.

import "testing"

// payload fabrique un GetMatchStats minimal à deux camps.
func payload(w0, l0, t0, w1, l1, t1 int) map[string]any {
	team := func(id, w, l, tie int) map[string]any {
		return map[string]any{
			"TeamId": float64(id),
			"Stats": map[string]any{
				"CoreStats": map[string]any{
					"RoundsWon":  float64(w),
					"RoundsLost": float64(l),
					"RoundsTied": float64(tie),
				},
			},
		}
	}
	return map[string]any{"Teams": []any{team(0, w0, l0, t0), team(1, w1, l1, t1)}}
}

func ptr(v int) *int { return &v }

func TestDecide_LigneNulleEstAEcrire(t *testing.T) {
	// Témoin 293a763e (Arena:Oddball) : 2 manches à 1 sur 3 jouées.
	d := Decide(payload(2, 1, 0, 1, 2, 0), RegistryRounds{})
	if d.Verdict != VerdictFix || !d.Writable() {
		t.Fatalf("verdict = %q, want %q", d.Verdict, VerdictFix)
	}
	if d.NewTeam0Won != 2 || d.NewTeam1Won != 1 || d.NewTotal != 3 {
		t.Errorf("décision = %d-%d sur %d, want 2-1 sur 3", d.NewTeam0Won, d.NewTeam1Won, d.NewTotal)
	}
}

func TestDecide_DejaAJourNEcritPas(t *testing.T) {
	cur := RegistryRounds{Team0Won: ptr(2), Team1Won: ptr(1), Total: ptr(3)}
	d := Decide(payload(2, 1, 0, 1, 2, 0), cur)
	if d.Verdict != VerdictIdentical || d.Writable() {
		t.Errorf("verdict = %q, want %q non écrivable", d.Verdict, VerdictIdentical)
	}
}

func TestDecide_ZeroNEstPasUnNull(t *testing.T) {
	// Un camp à 0 manche gagnée est une MESURE ; la ligne NULL doit tout de même être
	// écrite, et une ligne déjà à 0 ne doit pas être réécrite.
	d := Decide(payload(0, 2, 0, 2, 0, 0), RegistryRounds{Team0Won: ptr(0), Team1Won: ptr(2), Total: ptr(2)})
	if d.Verdict != VerdictIdentical {
		t.Errorf("verdict = %q, want %q (0 en base == 0 de l'API)", d.Verdict, VerdictIdentical)
	}
	d2 := Decide(payload(0, 2, 0, 2, 0, 0), RegistryRounds{})
	if d2.Verdict != VerdictFix || d2.NewTeam0Won != 0 || d2.NewTotal != 2 {
		t.Errorf("ligne NULL : verdict = %q, %d-%d sur %d ; want a_ecrire 0-2 sur 2",
			d2.Verdict, d2.NewTeam0Won, d2.NewTeam1Won, d2.NewTotal)
	}
}

func TestDecide_MancheNulleComptee(t *testing.T) {
	// Témoin adb93fb7 : 1 manche chacun + 1 nulle → 3 jouées, à égalité.
	d := Decide(payload(1, 1, 1, 1, 1, 1), RegistryRounds{})
	if d.Verdict != VerdictFix || d.NewTotal != 3 || d.NewTeam0Won != 1 || d.NewTeam1Won != 1 {
		t.Errorf("décision = %q %d-%d sur %d, want a_ecrire 1-1 sur 3",
			d.Verdict, d.NewTeam0Won, d.NewTeam1Won, d.NewTotal)
	}
}

func TestDecide_AbandonTotalAuMax(t *testing.T) {
	// Témoin 27a69918 : un camp non crédité. Le total reste 1, jamais 0.
	d := Decide(payload(0, 0, 0, 1, 0, 0), RegistryRounds{})
	if d.NewTotal != 1 {
		t.Errorf("total = %d, want 1 (max des deux camps)", d.NewTotal)
	}
}

func TestDecide_SansCampsEstSaute(t *testing.T) {
	for name, js := range map[string]map[string]any{
		"payload vide": {},
		"un seul camp": {"Teams": []any{map[string]any{"TeamId": float64(0), "Stats": map[string]any{"CoreStats": map[string]any{"RoundsWon": float64(1)}}}}},
		"sans CoreStats": {"Teams": []any{
			map[string]any{"TeamId": float64(0)},
			map[string]any{"TeamId": float64(1)},
		}},
	} {
		if d := Decide(js, RegistryRounds{}); d.Verdict != VerdictSkipNoTeams || d.Writable() {
			t.Errorf("%s : verdict = %q, want %q", name, d.Verdict, VerdictSkipNoTeams)
		}
	}
}

func TestDecide_TotalIncoherentEstSaute(t *testing.T) {
	// 2 manches gagnées de chaque côté mais un seul total de 2 : payload incohérent.
	// Fabriqué à la main (RoundsLost/Tied à 0 des deux côtés) — l'écrire ferait mentir
	// la règle d'affichage.
	js := payload(2, 0, 0, 2, 0, 0)
	d := Decide(js, RegistryRounds{})
	if d.Verdict != VerdictSkipImplausible || d.Writable() {
		t.Errorf("verdict = %q (%s), want %q", d.Verdict, d.Reason, VerdictSkipImplausible)
	}
}

func TestDecide_ValeurNegativeEstSautee(t *testing.T) {
	d := Decide(payload(-1, 0, 0, 1, 0, 0), RegistryRounds{})
	if d.Verdict != VerdictSkipImplausible || d.Writable() {
		t.Errorf("verdict = %q, want %q", d.Verdict, VerdictSkipImplausible)
	}
}

func TestFormatRounds_DistingueNullEtZero(t *testing.T) {
	if got := formatRounds(RegistryRounds{}); got != "NULL-NULL sur NULL" {
		t.Errorf("formatRounds(vide) = %q", got)
	}
	if got := formatRounds(RegistryRounds{Team0Won: ptr(0), Team1Won: ptr(2), Total: ptr(2)}); got != "0-2 sur 2" {
		t.Errorf("formatRounds(0-2/2) = %q", got)
	}
}
