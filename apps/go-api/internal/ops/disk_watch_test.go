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

	t.Run("recovery breach→ok : confirmée après 2 ticks (débounce)", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusCritical, LastNotifiedAt: t0}
		// 1er ok : amélioration en attente de confirmation, pas encore de notif.
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(time.Hour))
		if notify {
			t.Fatal("critical→ok (1er tick) : recovery pas encore confirmée (débounce)")
		}
		// 2e ok consécutif : rétablissement confirmé → notif.
		notify, st = ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(2*time.Hour))
		if !notify {
			t.Fatal("critical→ok (2e tick consécutif) : notification de recovery attendue")
		}
		notify, _ = ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(3*time.Hour))
		if notify {
			t.Fatal("ok après recovery confirmée : silence attendu")
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
		// ... et un retour en ok se confirme sur 2 ticks (débounce) avant de notifier.
		_, afterOK := ShouldNotifyDisk(next, domain.FreshnessStatusOK, t0.Add(2*time.Hour))
		notify, _ = ShouldNotifyDisk(afterOK, domain.FreshnessStatusOK, t0.Add(3*time.Hour))
		if !notify {
			t.Fatal("recovery (2 ticks ok) après unknown transitoire : notification attendue")
		}
	})

	t.Run("boot direct en critical (LastStatus vide) : notifie", func(t *testing.T) {
		notify, _ := ShouldNotifyDisk(DiskWatchState{}, domain.FreshnessStatusCritical, t0)
		if !notify {
			t.Fatal("boot en critical : notification attendue")
		}
	})

	t.Run("oscillation ok↔warn (jitter 1 tick) : pas de spam", func(t *testing.T) {
		// Volume à ~80 % : warn confirmé une fois, puis dips ok isolés absorbés
		// (jamais 2 ok consécutifs → aucune recovery, LastStatus reste warn → aucun
		// re-warn). Zéro notification supplémentaire malgré l'alternance à chaque tick.
		st := DiskWatchState{LastStatus: domain.FreshnessStatusOK}
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusWarn, t0)
		if !notify {
			t.Fatal("1re entrée en warn : notif attendue")
		}
		for i := 1; i <= 10; i++ {
			status := domain.FreshnessStatusOK
			if i%2 == 0 {
				status = domain.FreshnessStatusWarn
			}
			notify, st = ShouldNotifyDisk(st, status, t0.Add(time.Duration(i)*15*time.Minute))
			if notify {
				t.Fatalf("tick %d (%s) : oscillation ok↔warn ne doit pas notifier", i, status)
			}
		}
	})

	t.Run("oscillation warn↔critical (jitter 1 tick) : une seule alerte critique", func(t *testing.T) {
		st := DiskWatchState{LastStatus: domain.FreshnessStatusWarn, LastNotifiedAt: t0}
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusCritical, t0.Add(time.Minute))
		if !notify {
			t.Fatal("warn→critical : notif attendue")
		}
		// Dips critical→warn isolés : débouncés, LastStatus reste critical → pas de
		// ré-aggravation, aucune notif supplémentaire.
		for i := 1; i <= 8; i++ {
			status := domain.FreshnessStatusWarn
			if i%2 == 0 {
				status = domain.FreshnessStatusCritical
			}
			notify, st = ShouldNotifyDisk(st, status, t0.Add(time.Duration(i)*15*time.Minute))
			if notify {
				t.Fatalf("tick %d (%s) : oscillation warn↔critical ne doit pas re-notifier", i, status)
			}
		}
	})

	t.Run("amélioration interrompue : débounce réinitialisé", func(t *testing.T) {
		// critical, ok (pending=1), critical (reset), ok (pending=1) : jamais 2 ok
		// consécutifs → aucune recovery notifiée.
		st := DiskWatchState{LastStatus: domain.FreshnessStatusCritical, LastNotifiedAt: t0}
		notify, st := ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(time.Hour))
		if notify {
			t.Fatal("1er ok : pas encore confirmé")
		}
		notify, st = ShouldNotifyDisk(st, domain.FreshnessStatusCritical, t0.Add(2*time.Hour))
		if notify {
			t.Fatal("retour critical (== dernier confirmé) : pas de notif")
		}
		notify, _ = ShouldNotifyDisk(st, domain.FreshnessStatusOK, t0.Add(3*time.Hour))
		if notify {
			t.Fatal("ok après reset : compteur repart à 1, pas de recovery")
		}
	})
}
