package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain/title"
)

const (
	titleIDInfinite = "2043073184"
	titleIDHalo5    = "219630713"
	titleIDOther    = "2076696971" // jeu non suivi
)

// twoTitleRegistry : les deux titres suivis par LevelUp.
func twoTitleRegistry() *title.Registry {
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_infinite", Name: "Halo Infinite", Status: title.StatusActive, XboxTitleID: titleIDInfinite,
	})
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive, XboxTitleID: titleIDHalo5,
	})
	return reg
}

// counterFor assemble un compteur avec des sources en dur ; fetchCalls compte
// les allers-retours Xbox (le cache TTL en dépend).
func counterFor(t *testing.T, gamertags []string, resolved map[string]string,
	presences []FriendPresence, fetchErr error, fetchCalls *int,
) *FriendPresenceCounter {
	t.Helper()
	return NewFriendPresenceCounter(
		func(context.Context) []string { return gamertags },
		func(context.Context, []string) (map[string]string, error) { return resolved, nil },
		func(context.Context, []string) ([]FriendPresence, error) {
			if fetchCalls != nil {
				*fetchCalls++
			}
			return presences, fetchErr
		},
		twoTitleRegistry(),
	)
}

// Un ami est « en jeu » sur N'IMPORTE quel titre suivi — Halo 5 compte autant
// qu'Infinite. Un jeu hors registre ne compte pas.
func TestFriendsCount_AnyTrackedTitleCounts(t *testing.T) {
	c := counterFor(t,
		[]string{"Ami1", "Ami2", "Ami3"},
		map[string]string{"Ami1": "111", "Ami2": "222", "Ami3": "333"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "222", TitleID: titleIDHalo5},
			{XUID: "333", TitleID: titleIDOther},
		}, nil, nil)

	if got := c.Count(context.Background()); got != 2 {
		t.Errorf("Count = %d, attendu 2 (Infinite + Halo 5 ; l'autre jeu ne compte pas)", got)
	}
}

// Présence MASQUÉE (privacy) : l'ami arrive sans titre, ou n'arrive pas du tout.
// Dans les deux cas il n'est pas compté, et ce n'est PAS une erreur.
func TestFriendsCount_MaskedPresenceIgnored(t *testing.T) {
	c := counterFor(t,
		[]string{"Visible", "Masque", "Absent"},
		map[string]string{"Visible": "111", "Masque": "222", "Absent": "333"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "222", TitleID: ""}, // présence masquée : aucun titre rendu
			// "333" : absent de la réponse Xbox
		}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1 (masqué et absent ignorés)", got)
	}
}

// Un ami jamais croisé en match n'a pas de xuid connu : il est ignoré, sans
// empêcher le comptage des autres.
func TestFriendsCount_UnknownGamertagIgnored(t *testing.T) {
	c := counterFor(t,
		[]string{"Connu", "JamaisCroise"},
		map[string]string{"Connu": "111"},
		[]FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1", got)
	}
}

// Xbox indisponible : compteur à zéro, jamais d'erreur remontée — et l'échec
// n'est PAS mémorisé, sinon un incident d'une seconde gèlerait 45 s d'affichage.
func TestFriendsCount_FetchErrorReturnsZeroAndIsNotCached(t *testing.T) {
	calls := 0
	c := counterFor(t, []string{"Ami"}, map[string]string{"Ami": "111"},
		nil, errors.New("xbox 503"), &calls)

	if got := c.Count(context.Background()); got != 0 {
		t.Errorf("Count = %d, attendu 0", got)
	}
	if got := c.Count(context.Background()); got != 0 {
		t.Errorf("Count (2e appel) = %d, attendu 0", got)
	}
	if calls != 2 {
		t.Errorf("appels Xbox = %d, attendu 2 (un échec ne se met pas en cache)", calls)
	}
}

// Le cache évite un appel Xbox par poll du shell (30 s) : deux appels
// rapprochés ne touchent Xbox qu'une fois.
func TestFriendsCount_CachedWithinTTL(t *testing.T) {
	calls := 0
	c := counterFor(t, []string{"Ami"}, map[string]string{"Ami": "111"},
		[]FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil, &calls)

	first := c.Count(context.Background())
	second := c.Count(context.Background())
	if first != 1 || second != 1 {
		t.Errorf("Count = %d puis %d, attendu 1 et 1", first, second)
	}
	if calls != 1 {
		t.Errorf("appels Xbox = %d, attendu 1 (2e servi par le cache)", calls)
	}
}

// Changer la liste d'amis des Réglages doit invalider le cache : la clé EST la
// liste des xuids interrogés.
func TestFriendsCount_ListChangeInvalidatesCache(t *testing.T) {
	calls := 0
	gamertags := []string{"Ami1"}
	resolved := map[string]string{"Ami1": "111", "Ami2": "222"}
	c := NewFriendPresenceCounter(
		func(context.Context) []string { return gamertags },
		func(context.Context, []string) (map[string]string, error) { return resolved, nil },
		func(_ context.Context, xuids []string) ([]FriendPresence, error) {
			calls++
			out := make([]FriendPresence, 0, len(xuids))
			for _, x := range xuids {
				out = append(out, FriendPresence{XUID: x, TitleID: titleIDInfinite})
			}
			return out, nil
		},
		twoTitleRegistry(),
	)

	if got := c.Count(context.Background()); got != 1 {
		t.Fatalf("Count initial = %d, attendu 1", got)
	}
	gamertags = []string{"Ami1", "Ami2"}
	if got := c.Count(context.Background()); got != 2 {
		t.Errorf("Count après ajout = %d, attendu 2", got)
	}
	if calls != 2 {
		t.Errorf("appels Xbox = %d, attendu 2 (la liste a changé)", calls)
	}
}

// Deux gamertags pointant le même compte, ou un doublon de la réponse Xbox, ne
// comptent qu'une fois : un ami reste un ami.
func TestFriendsCount_DeduplicatesByXUID(t *testing.T) {
	c := counterFor(t,
		[]string{"Ami", "AmiRenomme"},
		map[string]string{"Ami": "111", "AmiRenomme": "111"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "111", TitleID: titleIDHalo5},
		}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1", got)
	}
}

// Une présence rendue par Xbox pour un xuid NON demandé n'entre pas au compte.
func TestFriendsCount_UnrequestedXUIDIgnored(t *testing.T) {
	c := counterFor(t,
		[]string{"Ami"},
		map[string]string{"Ami": "111"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "999", TitleID: titleIDInfinite},
		}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1", got)
	}
}

// Réglages sans aucun ami : zéro, sans toucher ni la base ni Xbox.
func TestFriendsCount_NoFriendsConfigured(t *testing.T) {
	calls := 0
	c := counterFor(t, nil, nil, nil, nil, &calls)
	if got := c.Count(context.Background()); got != 0 {
		t.Errorf("Count = %d, attendu 0", got)
	}
	if calls != 0 {
		t.Errorf("appels Xbox = %d, attendu 0", calls)
	}
}

// Dépendance manquante (démo, watcher off, shared indisponible) : pas de
// compteur du tout — le service rendra 0 sans jamais tenter d'appel.
func TestNewFriendPresenceCounter_NilWhenDependencyMissing(t *testing.T) {
	cases := map[string]*FriendPresenceCounter{
		"sans gamertags": NewFriendPresenceCounter(nil,
			func(context.Context, []string) (map[string]string, error) { return nil, nil },
			func(context.Context, []string) ([]FriendPresence, error) { return nil, nil },
			twoTitleRegistry()),
		"sans résolveur": NewFriendPresenceCounter(func(context.Context) []string { return nil },
			nil,
			func(context.Context, []string) ([]FriendPresence, error) { return nil, nil },
			twoTitleRegistry()),
		"sans fetch": NewFriendPresenceCounter(func(context.Context) []string { return nil },
			func(context.Context, []string) (map[string]string, error) { return nil, nil },
			nil, twoTitleRegistry()),
		"sans registre": NewFriendPresenceCounter(func(context.Context) []string { return nil },
			func(context.Context, []string) (map[string]string, error) { return nil, nil },
			func(context.Context, []string) ([]FriendPresence, error) { return nil, nil }, nil),
	}
	for name, c := range cases {
		if c != nil {
			t.Errorf("%s : compteur non nil", name)
		}
	}
}
