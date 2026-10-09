package snapshot

import (
	"slices"
	"testing"

	"levelup/go-api/internal/sync/matchflags"
)

// fullInfinite : un match Infinite COMPLET (toutes dérivations terminales,
// 2-team éligible, row LUSR présente, weapon-kills calculés).
func fullInfiniteFacts() matchReadinessFacts {
	return matchReadinessFacts{
		eventsLoaded:      true,
		backfillCompleted: matchflags.MBitWeaponKills,
		isRanked:          false,
		isFirefight:       false,
		durationSeconds:   600,
		humanTeamCount:    2,
		perfScoreSet:      true,
		dominanceSet:      true,
		psaCheckedSet:     true,
		citationsExist:    true,
		lusrRowExists:     true,
	}
}

var infiniteCaps = titleReadinessCaps{hasLUSR: true, hasWeaponKills: true, hasFirefight: true}

func TestIsMatchSnapshotReady(t *testing.T) {
	t.Run("complet 2-team Infinite → ready sans raison", func(t *testing.T) {
		ready, reasons := isMatchSnapshotReady(fullInfiniteFacts(), infiniteCaps, false)
		if !ready || len(reasons) != 0 {
			t.Fatalf("ready=%v reasons=%v, attendu ready sans raison", ready, reasons)
		}
	})

	t.Run("weapon film perdu → ready [weapons_absent]", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.backfillCompleted = matchflags.MBitFilmAbsent
		ready, reasons := isMatchSnapshotReady(f, infiniteCaps, false)
		if !ready || !slices.Contains(reasons, snapReasonWeaponsAbsent) {
			t.Fatalf("ready=%v reasons=%v, attendu ready+weapons_absent", ready, reasons)
		}
	})

	t.Run("FFA (3 teams) → ready [lusr_ineligible]", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.humanTeamCount = 3
		ready, reasons := isMatchSnapshotReady(f, infiniteCaps, false)
		if !ready || !slices.Contains(reasons, snapReasonLUSRIneligible) {
			t.Fatalf("ready=%v reasons=%v, attendu ready+lusr_ineligible", ready, reasons)
		}
	})

	t.Run("2-team éligible sans row LUSR (imbalance/DNF) → ready [lusr_skipped] NON bloquant", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.lusrRowExists = false
		ready, reasons := isMatchSnapshotReady(f, infiniteCaps, false)
		if !ready || !slices.Contains(reasons, snapReasonLUSRSkipped) {
			t.Fatalf("ready=%v reasons=%v, attendu ready+lusr_skipped (jamais bloqué)", ready, reasons)
		}
	})

	t.Run("perf manquant dans la grâce → NON ready", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.perfScoreSet = false
		ready, reasons := isMatchSnapshotReady(f, infiniteCaps, false)
		if ready || reasons != nil {
			t.Fatalf("ready=%v reasons=%v, attendu NON ready", ready, reasons)
		}
	})

	t.Run("perf manquant grâce dépassée → ready FORCÉ [forced, blocked_perf]", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.perfScoreSet = false
		ready, reasons := isMatchSnapshotReady(f, infiniteCaps, true)
		if !ready || !slices.Contains(reasons, snapReasonForced) || !slices.Contains(reasons, snapReasonBlockedPerf) {
			t.Fatalf("ready=%v reasons=%v, attendu ready forcé+blocked_perf", ready, reasons)
		}
	})

	t.Run("Infinite sans bit weapon (ni no-film) dans la grâce → NON ready (weapon EXIGÉ)", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.backfillCompleted = 0
		ready, _ := isMatchSnapshotReady(f, infiniteCaps, false)
		if ready {
			t.Fatal("Infinite doit EXIGER les weapon-kills (CapWeaponKills) → NON ready")
		}
	})

	t.Run("Halo 5 (pas CapWeaponKills) sans bit weapon → ready (weapon NON exigé)", func(t *testing.T) {
		h5Caps := titleReadinessCaps{hasLUSR: true, hasWeaponKills: false, hasFirefight: true}
		f := fullInfiniteFacts()
		f.backfillCompleted = 0 // pas de pipeline weapon côté Halo 5
		ready, reasons := isMatchSnapshotReady(f, h5Caps, false)
		if !ready || len(reasons) != 0 {
			t.Fatalf("ready=%v reasons=%v, attendu ready sans raison (weapon non exigé)", ready, reasons)
		}
	})

	t.Run("Firefight sans matchflags.MBitPVEStats dans la grâce → NON ready", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.isFirefight = true
		f.backfillCompleted = matchflags.MBitWeaponKills // pas de matchflags.MBitPVEStats
		ready, _ := isMatchSnapshotReady(f, infiniteCaps, false)
		if ready {
			t.Fatal("Firefight sans stats PvE doit être NON ready (dans la grâce)")
		}
	})

	t.Run("Firefight avec matchflags.MBitPVEStats → ready [lusr_ineligible] (firefight = LUSR inéligible)", func(t *testing.T) {
		f := fullInfiniteFacts()
		f.isFirefight = true
		f.backfillCompleted = matchflags.MBitWeaponKills | matchflags.MBitPVEStats
		ready, reasons := isMatchSnapshotReady(f, infiniteCaps, false)
		if !ready || !slices.Contains(reasons, snapReasonLUSRIneligible) {
			t.Fatalf("ready=%v reasons=%v, attendu ready+lusr_ineligible", ready, reasons)
		}
	})
}
