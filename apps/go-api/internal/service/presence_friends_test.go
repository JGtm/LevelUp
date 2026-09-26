// Package service — presence_friends_test.go : LE COMPTEUR « amis en jeu ».
//
// Ce que ces tests protègent, c'est une définition produit, pas une mécanique :
// « mes amis » = les joueurs inscrits que JE VOIS dans mon cercle (ADR 0029)
// SANS en être le propriétaire. Deux propriétés en découlent, et chacune a son
// test parce que chacune peut casser silencieusement :
//
//   - le compte est PERSONNEL : sur un état de watcher identique, deux
//     utilisateurs obtiennent deux valeurs (un compte calculé sur l'état global
//     du watcher, ou sur la seule liste des joueurs possédés, passerait les
//     autres tests mais pas celui-là) ;
//   - le compte s'arrête au CERCLE : un joueur d'un utilisateur étranger à mon
//     groupe n'y entre jamais — la propriété est prise au chokepoint authz, et
//     le dernier test le joue à travers le VRAI BootstrapService, seul endroit
//     où cette frontière est réellement décidée.
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// directOwnerOf : l'utilisateur possède EN PROPRE les xuids donnés, et eux
// seuls. Tient lieu de BootstrapService.DirectOwnerFor dans les tests unitaires
// (le test bout-en-bout, lui, prend la vraie implémentation).
func directOwnerOf(xuids ...string) DirectOwnerResolver {
	owned := make(map[string]bool, len(xuids))
	for _, x := range xuids {
		owned[x] = true
	}
	return func(*domain.SessionData) DirectOwnerFunc {
		return func(playerXUID string) bool { return owned[playerXUID] }
	}
}

// circle : les trois joueurs visibles du cercle A∪B. Les xuids comptent : c'est
// sur eux que se décide la propriété.
func circle() []domain.PlayerSummary {
	return []domain.PlayerSummary{
		{PlayerSlug: "p1", Gamertag: "P1", XUID: "111"},
		{PlayerSlug: "p2", Gamertag: "P2", XUID: "222"},
		{PlayerSlug: "p3", Gamertag: "P3", XUID: "333"},
	}
}

// LE test du lot : MÊME état de watcher, MÊME liste visible, deux utilisateurs —
// deux comptes. P1 est hors jeu, P2 et P3 en jeu.
func TestFriendsInGame_IsPersonalToEachUser(t *testing.T) {
	tracked := trackedSource(
		offline("P1"),
		inGame("P2", "halo_infinite", "Halo Infinite"),
		inGame("P3", "halo_5", "Halo 5"),
	)

	// A possède p1 : ses amis en jeu sont p2 et p3.
	svcA := NewPresenceService(ownedPlayers(circle()...), tracked).WithFriends(directOwnerOf("111"))
	if got := svcA.GetSnapshot(context.Background(), nil).FriendsInGame; got != 2 {
		t.Errorf("compte de A = %d, attendu 2 (p2 et p3 en jeu, p1 est le sien)", got)
	}

	// B possède p2 : p1 est bien un ami, mais il n'est pas en jeu — reste p3.
	svcB := NewPresenceService(ownedPlayers(circle()...), tracked).WithFriends(directOwnerOf("222"))
	if got := svcB.GetSnapshot(context.Background(), nil).FriendsInGame; got != 1 {
		t.Errorf("compte de B = %d, attendu 1 (p3 seul : p1 hors jeu, p2 est le sien)", got)
	}
}

// Propriétaire de TOUS les joueurs visibles : aucun ami, quel que soit le nombre
// de manettes allumées.
func TestFriendsInGame_OwnerOfEverything_CountsZero(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(circle()...),
		trackedSource(
			inGame("P1", "halo_infinite", "Halo Infinite"),
			inGame("P2", "halo_infinite", "Halo Infinite"),
			inGame("P3", "halo_infinite", "Halo Infinite"),
		),
	).WithFriends(directOwnerOf("111", "222", "333"))

	if got := svc.GetSnapshot(context.Background(), nil).FriendsInGame; got != 0 {
		t.Errorf("compte = %d, attendu 0 (tous les joueurs visibles sont les siens)", got)
	}
}

// Watcher éteint : le service ne sait rien de personne — compteur à zéro, et
// toujours une réponse (l'endpoint ne connaît pas l'échec).
func TestFriendsInGame_WatcherOff_CountsZero(t *testing.T) {
	for nom, source := range map[string]TrackedPresenceSource{
		"source absente": nil,
		"daemon arrêté":  trackedSource(),
		"aucun gamertag": trackedSource(TrackedPresence{Gamertag: ""}),
	} {
		t.Run(nom, func(t *testing.T) {
			svc := NewPresenceService(ownedPlayers(circle()...), source).
				WithFriends(directOwnerOf("111"))
			snap := svc.GetSnapshot(context.Background(), nil)
			if snap.FriendsInGame != 0 {
				t.Errorf("compte = %d, attendu 0", snap.FriendsInGame)
			}
			if snap.Players == nil {
				t.Error("players ne doit jamais être nil (tranche vide au contrat)")
			}
		})
	}
}

// La borne de fraîcheur s'applique au COMPTE comme à la manette : un ami dont le
// poll s'est tu depuis plus de presenceFreshnessWindow n'est plus « en jeu ».
// Sans ce test, le compteur pourrait servir un titre figé que la liste, elle,
// blanchit déjà — deux vérités dans la même réponse.
func TestFriendsInGame_StalePresence_IsNotCounted(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(circle()...),
		trackedSource(
			TrackedPresence{
				Gamertag:    "P2",
				TitleSlug:   "halo_infinite",
				TitleName:   "Halo Infinite",
				LastEventAt: time.Now().Add(-presenceFreshnessWindow - time.Second),
			},
			inGame("P3", "halo_infinite", "Halo Infinite"),
		),
	).WithFriends(directOwnerOf("111"))

	snap := svc.GetSnapshot(context.Background(), nil)
	if snap.FriendsInGame != 1 {
		t.Errorf("compte = %d, attendu 1 (P2 périmé ne compte pas, P3 oui)", snap.FriendsInGame)
	}
	for _, p := range snap.Players {
		if p.Gamertag == "P2" && p.InGame {
			t.Errorf("P2 servi en jeu malgré un poll muet: %+v", p)
		}
	}
}

// Sans prédicat de propriété (démo, boot partiel), le compteur reste à zéro :
// compter ses PROPRES joueurs comme des amis serait pire que ne rien afficher.
func TestFriendsInGame_NoOwnershipPredicate_CountsZero(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(circle()...),
		trackedSource(inGame("P2", "halo_infinite", "Halo Infinite")),
	)

	if got := svc.GetSnapshot(context.Background(), nil).FriendsInGame; got != 0 {
		t.Errorf("compte = %d, attendu 0 (aucun prédicat de propriété câblé)", got)
	}
}

// Un profil sans xuid n'est attribuable à personne : il n'est jamais compté (la
// propriété ne se déduit pas d'un gamertag).
func TestFriendsInGame_PlayerWithoutXUID_IsNotCounted(t *testing.T) {
	svc := NewPresenceService(
		ownedPlayers(domain.PlayerSummary{PlayerSlug: "p9", Gamertag: "P9"}),
		trackedSource(inGame("P9", "halo_infinite", "Halo Infinite")),
	).WithFriends(directOwnerOf("111"))

	if got := svc.GetSnapshot(context.Background(), nil).FriendsInGame; got != 0 {
		t.Errorf("compte = %d, attendu 0 (profil sans xuid)", got)
	}
}

// Session absente : même fail-closed que le reste de l'endpoint. La vraie
// implémentation de propriété (DirectOwnerFor) ne reconnaît alors aucun
// utilisateur, et la liste visible est vide — donc rien à compter.
func TestFriendsInGame_NoSession_CountsZero(t *testing.T) {
	snap := presenceCircleService(t).GetSnapshot(context.Background(), nil)
	if snap.FriendsInGame != 0 {
		t.Errorf("compte sans session = %d, attendu 0", snap.FriendsInGame)
	}
	if len(snap.Players) != 0 {
		t.Errorf("players sans session = %+v, attendu vide", snap.Players)
	}
}

// ─── BOUT EN BOUT : LA FRONTIÈRE DU CERCLE, PAR LE VRAI CHOKEPOINT ─────────────

// presenceCircleService monte un BootstrapService RÉEL (db_profiles temporaire,
// user store et groupes) et rend le PresenceService branché sur ses deux
// méthodes — OwnedPlayers (qui voit quoi) et DirectOwnerFor (qui possède quoi).
// Aucun fake d'autorisation : c'est justement la frontière qu'on teste.
//
// Le parc : alice (222) et bob (999) partagent un groupe ; carol (777) est
// étrangère aux deux. Les trois sont en jeu.
func presenceCircleService(t *testing.T) *PresenceService {
	t.Helper()

	profilesPath := filepath.Join(t.TempDir(), "db_profiles.json")
	profiles := `{"version":"3.0","profiles":{"halo_infinite":{
	  "alice":{"db_path":"a.duckdb","xuid":"222"},
	  "bob":{"db_path":"b.duckdb","xuid":"999"},
	  "carol":{"db_path":"c.duckdb","xuid":"777"}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{AuthMode: "password", DBProfilesPath: profilesPath}
	boot := NewBootstrapService(cfg, &mockBootRepo{}).
		WithUserLookup(fakeBootstrapLookup{
			byName: map[string]*domain.User{
				"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
				"bob":   {Username: "bob", Role: domain.RoleUser, XUID: "999"},
				"carol": {Username: "carol", Role: domain.RoleUser, XUID: "777"},
			},
		}).
		WithCoMemberResolver(func(xuid string) map[string]bool {
			if xuid == "222" || xuid == "999" {
				return map[string]bool{"222": true, "999": true}
			}
			return map[string]bool{xuid: true}
		})

	tracked := trackedSource(
		inGame("alice", "halo_infinite", "Halo Infinite"),
		inGame("bob", "halo_infinite", "Halo Infinite"),
		inGame("carol", "halo_infinite", "Halo Infinite"),
	)
	return NewPresenceService(boot.OwnedPlayers, tracked).WithFriends(boot.DirectOwnerFor)
}

// Trois utilisateurs, un seul état de watcher : chacun ne compte que SON cercle,
// et carol — étrangère au groupe d'alice et bob — n'entre dans aucun des deux
// comptes (ni eux dans le sien). C'est le cas que la règle « tous les autres
// joueurs inscrits » aurait raté.
func TestFriendsInGame_StrangerOutsideGroupIsNeverCounted(t *testing.T) {
	svc := presenceCircleService(t)

	cas := []struct {
		user  string
		count int
		// ami : le gamertag qui DOIT être compté (vide = personne).
		ami string
	}{
		{user: "alice", count: 1, ami: "bob"},
		{user: "bob", count: 1, ami: "alice"},
		{user: "carol", count: 0},
	}
	for _, c := range cas {
		t.Run(c.user, func(t *testing.T) {
			sess := &domain.SessionData{Username: strPtr(c.user)}
			snap := svc.GetSnapshot(context.Background(), sess)

			if snap.FriendsInGame != c.count {
				t.Errorf("compte de %s = %d, attendu %d", c.user, snap.FriendsInGame, c.count)
			}
			// Aucune identité hors cercle ne transite : la liste servie ne porte
			// que les joueurs visibles, comme avant le lot.
			for _, p := range snap.Players {
				if p.Gamertag != c.user && p.Gamertag != c.ami {
					t.Errorf("%s voit un joueur hors de son cercle: %+v", c.user, p)
				}
			}
		})
	}
}

// La propriété DIRECTE ne se confond pas avec la visibilité : un co-membre est
// visible sans être possédé, et un admin voit tout sans rien posséder de plus.
func TestDirectOwnerFor_DistinguishesOwnershipFromVisibility(t *testing.T) {
	svc := newOwnershipBootstrap("password")
	alice := &domain.SessionData{Username: strPtr("alice")} // xuid 222
	boss := &domain.SessionData{Username: strPtr("boss")}   // admin, xuid 111

	ownsForAlice := svc.DirectOwnerFor(alice)
	if !ownsForAlice("222") {
		t.Error("alice doit posséder son propre profil")
	}
	if ownsForAlice("999") {
		t.Error("un profil de co-membre est VISIBLE, pas possédé")
	}
	if ownsForAlice("") {
		t.Error("un profil sans xuid n'est possédé par personne")
	}
	if svc.DirectOwnerFor(boss)("222") {
		t.Error("le rôle admin donne l'accès, pas la propriété")
	}
	if svc.DirectOwnerFor(nil)("222") {
		t.Error("sans session, aucun profil n'est possédé")
	}
}

// La fabrique résout l'utilisateur UNE FOIS, pas une fois par joueur : sans cela
// chaque joueur en jeu coûtait une relecture du user store (users.json en
// production). Un lookup compteur le prouve sur un cercle de trois joueurs.
func TestDirectOwnerFor_ResolvesUserOncePerRequest(t *testing.T) {
	lookup := &countingLookup{inner: fakeBootstrapLookup{
		byName: map[string]*domain.User{
			"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
		},
	}}
	svc := NewBootstrapService(&config.AppConfig{AuthMode: "password"}, &mockBootRepo{}).
		WithUserLookup(lookup)

	ownsPlayer := svc.DirectOwnerFor(&domain.SessionData{Username: strPtr("alice")})
	for _, xuid := range []string{"111", "222", "333"} {
		ownsPlayer(xuid)
	}

	if lookup.gets != 1 {
		t.Errorf("user store interrogé %d fois, attendu 1 (identité invariante sur la requête)", lookup.gets)
	}
}

// countingLookup compte les résolutions d'utilisateur — la lecture qui, en
// production, ouvre et parse users.json.
type countingLookup struct {
	inner fakeBootstrapLookup
	gets  int
}

func (c *countingLookup) Get(username string) (*domain.User, error) {
	c.gets++
	return c.inner.Get(username)
}

func (c *countingLookup) GetByXUID(xuid string) (*domain.User, error) {
	c.gets++
	return c.inner.GetByXUID(xuid)
}

// ─── RÉGIME NON APPLIQUÉ (LEVELUP_AUTH_MODE=none, configuration par DÉFAUT) ────

// Sans enforcement, il n'existe AUCUN « possédé en propre » : rien n'est à
// retrancher du cercle visible. Le test verrouille la moitié « propriété » de la
// règle du 2026-08-25 ; le suivant verrouille sa conséquence visible.
func TestDirectOwnerFor_NotEnforced_OwnsNothing(t *testing.T) {
	svc := newOwnershipBootstrap("none")
	ownsPlayer := svc.DirectOwnerFor(&domain.SessionData{Username: strPtr("alice")}) // xuid 222

	if ownsPlayer("222") {
		t.Error("sans enforcement, aucun profil n'est possédé EN PROPRE — pas même le sien")
	}
	if ownsPlayer("999") {
		t.Error("sans enforcement, aucun profil n'est possédé EN PROPRE")
	}
}

// Instance mono-opérateur (le déploiement par défaut) : tous les joueurs
// visibles en jeu sont comptés. Sans cette règle la pastille resterait à zéro en
// permanence sur cette configuration — une fonctionnalité livrée éteinte.
func TestFriendsInGame_NotEnforced_CountsEveryVisiblePlayerInGame(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "db_profiles.json")
	profiles := `{"version":"3.0","profiles":{"halo_infinite":{
	  "alice":{"db_path":"a.duckdb","xuid":"222"},
	  "bob":{"db_path":"b.duckdb","xuid":"999"}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}

	// Ni AuthMode ni user store : authz.Enforced est faux, filterOwnedPlayers
	// rend le parc entier, et personne n'en est propriétaire.
	cfg := &config.AppConfig{AuthMode: "none", DBProfilesPath: profilesPath}
	boot := NewBootstrapService(cfg, &mockBootRepo{})
	svc := NewPresenceService(
		boot.OwnedPlayers,
		trackedSource(
			inGame("alice", "halo_infinite", "Halo Infinite"),
			inGame("bob", "halo_infinite", "Halo Infinite"),
		),
	).WithFriends(boot.DirectOwnerFor)

	snap := svc.GetSnapshot(context.Background(), nil)
	if snap.FriendsInGame != 2 {
		t.Errorf("compte = %d, attendu 2 (mode non appliqué : tous les visibles en jeu)", snap.FriendsInGame)
	}
	if len(snap.Players) != 2 {
		t.Errorf("players = %+v, attendu les 2 joueurs visibles", snap.Players)
	}
}
