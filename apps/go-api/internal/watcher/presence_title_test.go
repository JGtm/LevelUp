// Package watcher — presence_title_test.go : le TITRE COURANT d'un joueur.
//
// Enjeu : `inGame` répond « le joueur joue-t-il au titre que CE watcher suit ? »
// et pilote la FSM. La question de l'UI est différente — « est-il en jeu, tout
// court ? ». Ces tests verrouillent la seconde, et surtout le piège d'ordre du
// handler : le titre doit être capté AVANT le test « titre du watcher », qui
// sort en inactif.
package watcher

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/presence"
)

// statusFor retourne l'état exposé du premier (et unique) joueur du daemon.
func statusFor(t *testing.T, d *Daemon) PlayerPresenceStatus {
	t.Helper()
	status := NewStateProvider(d).GetStatus()
	if len(status.Players) != 1 {
		t.Fatalf("players = %d, attendu 1", len(status.Players))
	}
	return status.Players[0]
}

// LE cas du lot : un watcher halo_5 voit son joueur lancer Halo Infinite. La
// FSM reste au repos (rien à syncer ici — comportement existant, préservé),
// mais le titre courant DOIT être exposé, sinon l'UI le dit hors jeu.
func TestCurrentTitle_OtherTrackedTitle_StillExposed(t *testing.T) {
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive, XboxTitleID: "219630713",
	})
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, reg, &mockDaemonSyncRunner{})
	pw := addTestPlayer(d, "JGtm", "111")
	pw.SetTitleSlug("halo_5") // titre CONFIGURÉ du joueur

	d.makePresenceHandler(context.Background(), pw)(haloPresenceEvent("111")) // il lance Infinite
	time.Sleep(20 * time.Millisecond)

	ps := statusFor(t, d)
	if ps.TitleSlug != "halo_infinite" || ps.TitleName != "Halo Infinite" {
		t.Errorf("titre courant = %q/%q, attendu halo_infinite/Halo Infinite", ps.TitleSlug, ps.TitleName)
	}
	if ps.InGame {
		t.Error("in_game (sémantique watcher) doit rester faux : ce watcher suit halo_5")
	}
	if pw.FSM().State() == StateWatching {
		t.Error("la FSM ne doit pas s'activer pour un autre titre (comportement existant)")
	}
}

// Titre suivi par CE watcher : titre courant exposé ET FSM active.
func TestCurrentTitle_SameTitle_ExposedAndInGame(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	pw := addTestPlayer(d, "JGtm", "111") // halo_infinite par défaut

	d.makePresenceHandler(context.Background(), pw)(haloPresenceEvent("111"))
	time.Sleep(20 * time.Millisecond)

	ps := statusFor(t, d)
	if ps.TitleSlug != "halo_infinite" {
		t.Errorf("title_slug = %q, attendu halo_infinite", ps.TitleSlug)
	}
	if !ps.InGame {
		t.Error("in_game devrait être vrai sur le titre du watcher")
	}
}

// Titre HORS registre (tableau de bord Xbox) : le joueur n'est en jeu sur aucun
// titre suivi — le titre courant précédent doit être effacé, pas laissé traîner.
func TestCurrentTitle_UntrackedTitle_Cleared(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	pw := addTestPlayer(d, "JGtm", "111")
	handler := d.makePresenceHandler(context.Background(), pw)

	handler(haloPresenceEvent("111")) // en jeu
	handler(presence.PresenceEvent{   // puis retour au tableau de bord
		XUID: "111", PresenceState: "Online",
		PresenceDetail: &presence.PresenceDetail{
			TitleID: "1022622766", TitleName: "Online", IsGame: false,
		},
	})
	time.Sleep(20 * time.Millisecond)

	if ps := statusFor(t, d); ps.TitleSlug != "" || ps.TitleName != "" {
		t.Errorf("titre courant = %q/%q, attendu vide", ps.TitleSlug, ps.TitleName)
	}
}

// Hors ligne (aucun PresenceDetail) : titre courant effacé lui aussi.
func TestCurrentTitle_Offline_Cleared(t *testing.T) {
	d := NewDaemon(DaemonConfig{RepoRoot: "/repo"}, title.NewRegistry(), &mockDaemonSyncRunner{})
	pw := addTestPlayer(d, "JGtm", "111")
	handler := d.makePresenceHandler(context.Background(), pw)

	handler(haloPresenceEvent("111"))
	handler(presence.PresenceEvent{XUID: "111", PresenceState: "Offline"})
	time.Sleep(20 * time.Millisecond)

	if ps := statusFor(t, d); ps.TitleSlug != "" {
		t.Errorf("title_slug = %q, attendu vide hors ligne", ps.TitleSlug)
	}
}
