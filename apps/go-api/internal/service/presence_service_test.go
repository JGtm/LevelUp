package service

import (
	"context"
	"errors"
	"testing"
	"time"

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

// inGame : un joueur vu sur un titre, avec un témoin de vivacité FRAIS. Le témoin
// est explicite dans tous les fixtures parce qu'il décide : un titre dont le poll
// s'est tu depuis plus de presenceFreshnessWindow n'est plus servi.
func inGame(gamertag, slug, name string) TrackedPresence {
	return TrackedPresence{
		Gamertag:    gamertag,
		TitleSlug:   slug,
		TitleName:   name,
		LastEventAt: time.Now(),
	}
}

// offline : un joueur suivi mais sur aucun titre. Le témoin reste renseigné — le
// poll vit, il ne rapporte simplement aucun jeu.
func offline(gamertag string) TrackedPresence {
	return TrackedPresence{Gamertag: gamertag, LastEventAt: time.Now()}
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
			inGame("JGtm", "halo_infinite", "Halo Infinite"),
			offline("Madina"),
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
		trackedSource(inGame("JGtm", "halo_infinite", "Halo Infinite")),
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
//
// LES DEUX ORDRES SONT JOUÉS, et c'est tout l'intérêt du test. Avec le seul ordre
// « sans titre d'abord », la dernière écriture gagne naturellement : retirer le
// garde de trackedByGamertag laisserait le test vert (constat de revue F5). C'est
// l'ordre INVERSE qui l'éprouve — il exige que l'entrée déjà titrée résiste à
// celle qui ne l'est pas.
func TestPresenceSnapshot_TwoWatchersSameGamertag_TitleWins(t *testing.T) {
	titre := inGame("JGtm", "halo_5", "Halo 5")
	sansTitre := offline("JGtm")

	cas := map[string][]TrackedPresence{
		"sans titre en premier": {sansTitre, titre},
		"titre en premier":      {titre, sansTitre},
	}
	for nom, list := range cas {
		t.Run(nom, func(t *testing.T) {
			svc := NewPresenceService(
				ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
				trackedSource(list...),
			)
			snap := svc.GetSnapshot(context.Background(), nil)
			if len(snap.Players) != 1 || !snap.Players[0].InGame || snap.Players[0].TitleSlug != "halo_5" {
				t.Fatalf("attendu en jeu sur halo_5: %+v", snap.Players)
			}
		})
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
			inGame("JGtm", "halo_infinite", "Halo Infinite"),
			inGame("Etranger", "halo_infinite", "Halo Infinite"),
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
		trackedSource(inGame("JGtm", "halo_infinite", "Halo Infinite")),
	)

	if got := len(svc.GetSnapshot(context.Background(), nil).Players); got != 0 {
		t.Errorf("players = %d, attendu 0", got)
	}
}

// ─── FRAÎCHEUR DU TITRE SERVI (F9) ──────────────────────────────────────────────

// Le titre courant n'est jamais effacé par un minuteur côté watcher : si le poll
// se tait, la dernière valeur resterait servie indéfiniment. Au-delà de la fenêtre
// de fraîcheur, le service la blanchit.
func TestPresenceSnapshot_StaleTitleIsNotServed(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(TrackedPresence{
			Gamertag:    "JGtm",
			TitleSlug:   "halo_infinite",
			TitleName:   "Halo Infinite",
			LastEventAt: time.Now().Add(-presenceFreshnessWindow - time.Second),
		}),
	)

	snap := svc.GetSnapshot(context.Background(), nil)
	if len(snap.Players) != 1 {
		t.Fatalf("players = %d, attendu 1 (le joueur reste listé)", len(snap.Players))
	}
	p := snap.Players[0]
	if p.InGame || p.TitleSlug != "" || p.TitleName != "" {
		t.Errorf("titre périmé encore servi: %+v", p)
	}
}

// Juste sous la fenêtre : rien ne change. La borne ne doit pas effacer un joueur
// dont le poll a simplement pris un backoff réseau (30 s).
func TestPresenceSnapshot_RecentTitleIsServed(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(TrackedPresence{
			Gamertag:    "JGtm",
			TitleSlug:   "halo_infinite",
			TitleName:   "Halo Infinite",
			LastEventAt: time.Now().Add(-presenceFreshnessWindow + 10*time.Second),
		}),
	)

	if p := svc.GetSnapshot(context.Background(), nil).Players[0]; !p.InGame {
		t.Errorf("titre encore frais non servi: %+v", p)
	}
}

// Un titre PÉRIMÉ ne doit pas gagner l'arbitrage contre un watcher frais qui dit
// « hors jeu » : le blanchiment se fait à l'ingestion, avant la préséance.
func TestPresenceSnapshot_StaleTitleDoesNotWinOverFreshOffline(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(
			offline("JGtm"),
			TrackedPresence{
				Gamertag:    "JGtm",
				TitleSlug:   "halo_5",
				TitleName:   "Halo 5",
				LastEventAt: time.Now().Add(-2 * presenceFreshnessWindow),
			},
		),
	)

	if p := svc.GetSnapshot(context.Background(), nil).Players[0]; p.InGame || p.TitleSlug != "" {
		t.Errorf("un titre périmé a gagné l'arbitrage: %+v", p)
	}
}

// Aucun event jamais reçu (témoin à zéro) : aucun titre servi.
func TestPresenceSnapshot_NoEventEver_NoTitle(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(TrackedPresence{Gamertag: "JGtm", TitleSlug: "halo_infinite"}),
	)

	if p := svc.GetSnapshot(context.Background(), nil).Players[0]; p.InGame || p.TitleSlug != "" {
		t.Errorf("titre servi sans aucun event: %+v", p)
	}
}

// ─── LE COMPTEUR D'AMIS DANS LA RÉPONSE ─────────────────────────────────────────

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

// F7 — une source d'amis qui BLOQUE ne doit pas tenir la réponse ouverte : le
// comptage sort du budget, la pastille tombe à zéro, et la liste des joueurs
// (la vraie raison d'être de l'endpoint) est servie intacte.
func TestPresenceSnapshot_BlockingFriendsSource_DoesNotStallResponse(t *testing.T) {
	debloque := make(chan struct{})
	defer close(debloque)

	counter := NewFriendPresenceCounter(
		func(context.Context) []string { return []string{"Ami"} },
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Ami": "111"}, nil
		},
		func(ctx context.Context, _ []string) ([]FriendPresence, error) {
			// Bloque jusqu'à l'annulation du contexte — le comportement d'un appel
			// Xbox parti dans un incident.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-debloque:
				return nil, nil
			}
		},
		twoTitleRegistry(),
	)
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "jgtm", Gamertag: "JGtm"}),
		trackedSource(inGame("JGtm", "halo_infinite", "Halo Infinite")),
	).WithFriends(counter)
	svc.friendsBudget = 30 * time.Millisecond

	debut := time.Now()
	snap := svc.GetSnapshot(context.Background(), nil)
	ecoule := time.Since(debut)

	if ecoule > time.Second {
		t.Fatalf("réponse rendue en %s : le comptage d'amis a tenu la requête ouverte", ecoule)
	}
	if snap.FriendsInGame != 0 {
		t.Errorf("friends_in_game = %d, attendu 0 (hors budget)", snap.FriendsInGame)
	}
	if len(snap.Players) != 1 || !snap.Players[0].InGame {
		t.Errorf("les joueurs doivent être servis intacts: %+v", snap.Players)
	}
}
