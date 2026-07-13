// Package ops — disk_watch_test.go : politique pure de notification disque.
package ops

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestShouldNotifyDisk(t *testing.T) {
	t0 := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)

	t.Run("ok stable : jamais de notification", func(t *testing.T) {
		st := DiskWatchState{}
		var notify bool
		for i := 0; i < 3; i++ {
			notify, st = ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(time.Duration(i)*time.Hour))
			if notify {
				t.Fatalf("itération %d : notification inattendue en ok stable", i)
			}
		}
	})

	t.Run("entrée en warn : notifie une fois puis silence", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusOK}
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusWarn, t0)
		if !notify {
			t.Fatal("transition ok→warn : notification attendue")
		}
		notify, _ = ShouldNotifyDisk(st, domain.FreshnessStatusWarn, t0.Add(time.Hour))
		if notify {
			t.Fatal("warn persistant < 24h : pas de re-notification attendue")
		}
	})

	t.Run("aggravation warn→critical : notifie", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusWarn, LastNotifiedAt: t0}
		notify, _ := ShouldNotifyDisk(st, domain.FreshnessStatusCritical, t0.Add(time.Minute))
		if !notify {
			t.Fatal("warn→critical : notification attendue")
		}
	})

	t.Run("rappel 24h en breach persistant", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusWarn, LastNotifiedAt: t0}
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusWarn, t0.Add(DiskRenotifyInterval))
		if !notify {
			t.Fatal("breach >= 24h : rappel attendu")
		}
		notify, _ = ShouldNotifyDisk(st, domain.FreshnessStatusWarn, t0.Add(DiskRenotifyInterval+time.Hour))
		if notify {
			t.Fatal("rappel déjà envoyé : silence attendu jusqu'au prochain intervalle")
		}
	})

	t.Run("recovery breach→ok : notifie le rétablissement", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusCritical, LastNotifiedAt: t0}
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(time.Hour))
		if !notify {
			t.Fatal("critical→ok : notification de recovery attendue")
		}
		notify, _ = ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(2*time.Hour))
		if notify {
			t.Fatal("ok après recovery : silence attendu")
		}
	})

	t.Run("unknown : jamais de notification, état préservé", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusWarn, LastNotifiedAt: t0}
		notify, next := ShouldNotifyDisk(st, domain.FreshnessStatusUnknown, t0.Add(time.Hour))
		if notify {
			t.Fatal("unknown : pas de notification attendue")
		}
		if next.LastStatus != domain.FreshnessStatusWarn {
			t.Fatalf("unknown ne doit pas écraser LastStatus (got %q)", next.LastStatus)
		}
		// Le breach warn reste actif : un retour en warn ne re-notifie pas (< 24h)...
		notify, _ = ShouldNotifyDisk(next, domain.FreshnessStatusWarn, t0.Add(2*time.Hour))
		if notify {
			t.Fatal("warn après unknown transitoire : pas une nouvelle transition")
		}
		// ... mais un retour en ok notifie bien le rétablissement.
		notify, _ = ShouldNotifyDisk(next, domain.FreshnessStatusOK, t0.Add(2*time.Hour))
		if !notify {
			t.Fatal("recovery après unknown transitoire : notification attendue")
		}
	})

	t.Run("boot direct en critical (LastStatus vide) : notifie", func(t *testing.T) {
		notify, _ := ShouldNotifyDisk(DiskWatchState{}, domain.FreshnessStatusCritical, t0)
		if !notify {
			t.Fatal("boot en critical : notification attendue")
		}
	})
}
