package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

func ownedPlayers(players ...domain.PlayerSummary) OwnedPlayersFunc {
	return func(context.Context, *domain.SessionData) ([]domain.PlayerSummary, error) {
		return players, nil
	}
}

func trackedSource(list ...TrackedPresence) TrackedPresenceSource {
	return func() []TrackedPresence { return list }
}

// Cas nominal : un joueur en jeu porte son titre, l'autre non. Le player_slug
// vient de la configuration (le watcher ne connaît que le gamertag).
func TestPresenceSnapshot_MarksInGamePlayers(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(
			domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"},
			domain.PlayerSummary{PlayerSlug: "madina", Gamertag: "Madina"},
		),
		trackedSource(
			TrackedPresence{Gamertag: "JGtm", TitleSlug: "halo_infinite", TitleName: "Halo Infinite"},
			TrackedPresence{Gamertag: "Madina"},
		),
	)

	snap := svc.GetSnapshot(context.Background(), nil)
	if len(snap.Players) != 2 {
		t.Fatalf("players = %d, attendu 2", len(snap.Players))
	}
	if !snap.Players[0].InGame || snap.Players[0].TitleName != "Halo Infinite" {
		t.Errorf("JGtm devrait être en jeu sur Halo Infinite: %+v", snap.Players[0])
	}
	if snap.Players[1].InGame || snap.Players[1].TitleSlug != "" {
		t.Errorf("Madina ne devrait pas être en jeu: %+v", snap.Players[1])
	}
	if snap.Players[0].PlayerSlug != "jgtm" {
		t.Errorf("player_slug = %q, attendu jgtm", snap.Players[0].PlayerSlug)
	}
}

// LE cas du lot : un joueur configuré halo_5 qui lance Halo Infinite EST en
// jeu. C'est le titre détecté qui décide, jamais le titre configuré.
func TestPresenceSnapshot_PlayerOnAnotherTrackedTitleIsInGame(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm", TitleSlug: "halo_5"}),
		trackedSource(TrackedPresence{
			Gamertag: "JGtm", TitleSlug: "halo_infinite", TitleName: "Halo Infinite",
		}),
	)

	snap := svc.GetSnapshot(context.Background(), nil)
	if len(snap.Players) != 1 || !snap.Players[0].InGame {
		t.Fatalf("joueur halo_5 jouant à Infinite doit être en jeu: %+v", snap.Players)
	}
	if snap.Players[0].TitleSlug != "halo_infinite" {
		t.Errorf("title_slug = %q, attendu halo_infinite (le titre RÉEL)", snap.Players[0].TitleSlug)
	}
}

// Un même gamertag suivi sur deux titres a deux watchers : celui qui porte un
// titre courant gagne, c'est le seul qui sait où le joueur joue.
func TestPresenceSnapshot_TwoWatchersSameGamertag_TitleWins(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(
			TrackedPresence{Gamertag: "JGtm"},
			TrackedPresence{Gamertag: "JGtm", TitleSlug: "halo_5", TitleName: "Halo 5"},
		),
	)

	snap := svc.GetSnapshot(context.Background(), nil)
	if len(snap.Players) != 1 || !snap.Players[0].InGame || snap.Players[0].TitleSlug != "halo_5" {
		t.Fatalf("attendu en jeu sur halo_5: %+v", snap.Players)
	}
}

// Watcher désactivé (source nil) : liste vide, jamais nil côté JSON.
func TestPresenceSnapshot_NoWatcher_EmptyList(t *testing.T) {
	svc := NewPresenceService(ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}), nil)

	snap := svc.GetSnapshot(context.Background(), nil)
	if snap.Players == nil {
		t.Fatal("players ne doit jamais être nil (tranche vide au contrat)")
	}
	if len(snap.Players) != 0 || snap.FriendsInGame != 0 {
		t.Errorf("attendu réponse vide, reçu %+v", snap)
	}
}

// Daemon arrêté (source qui rend une tranche vide) : idem, liste vide.
func TestPresenceSnapshot_DaemonStopped_EmptyList(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(),
	)

	if got := len(svc.GetSnapshot(context.Background(), nil).Players); got != 0 {
		t.Errorf("players = %d, attendu 0", got)
	}
}

// Un joueur suivi par le watcher mais NON accessible à l'utilisateur (ADR 0029)
// n'apparaît pas : la liste est l'intersection des deux sources.
func TestPresenceSnapshot_OnlyOwnedPlayersAreListed(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(
			TrackedPresence{Gamertag: "JGtm", TitleSlug: "halo_infinite"},
			TrackedPresence{Gamertag: "Etranger", TitleSlug: "halo_infinite"},
		),
	)

	snap := svc.GetSnapshot(context.Background(), nil)
	if len(snap.Players) != 1 || snap.Players[0].Gamertag != "JGtm" {
		t.Fatalf("seul le joueur possédé doit apparaître: %+v", snap.Players)
	}
}

// Chargement des joueurs en échec : réponse vide, jamais d'erreur (le shell
// interroge cet endpoint toutes les 30 s).
func TestPresenceSnapshot_PlayersLoadError_Degrades(t *testing.T) {
	svc := NewPresenceService(
		func(context.Context, *domain.SessionData) ([]domain.PlayerSummary, error) {
			return nil, errors.New("db_profiles illisible")
		},
		trackedSource(TrackedPresence{Gamertag: "JGtm", TitleSlug: "halo_infinite"}),
	)

	if got := len(svc.GetSnapshot(context.Background(), nil).Players); got != 0 {
		t.Errorf("players = %d, attendu 0", got)
	}
}

// Le compteur d'amis est branché : sa valeur remonte dans la réponse.
func TestPresenceSnapshot_FriendsCountIncluded(t *testing.T) {
	counter := NewFriendPresenceCounter(
		func(context.Context) []string { return []string{"Ami"} },
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Ami": "111"}, nil
		},
		func(context.Context, []string) ([]FriendPresence, error) {
			return []FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil
		},
		twoTitleRegistry(),
	)
	svc := NewPresenceService(ownedPlayers(), trackedSource()).WithFriends(counter)

	if got := svc.GetSnapshot(context.Background(), nil).FriendsInGame; got != 1 {
		t.Errorf("friends_in_game = %d, attendu 1", got)
	}
}
