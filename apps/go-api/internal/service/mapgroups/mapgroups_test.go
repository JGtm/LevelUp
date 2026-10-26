package mapgroups

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

type lecteur struct {
	groups domain.GroupesDeCartes
	err    error
}

func (l lecteur) MapGroups(context.Context) (domain.GroupesDeCartes, error) { return l.groups, l.err }

func TestLoad_Degradation(t *testing.T) {
	if g := Load(context.Background(), nil); !g.Vide() {
		t.Error("sans lecteur : aucun groupe attendu")
	}
	if g := Load(context.Background(), lecteur{err: errors.New("base illisible")}); !g.Vide() {
		t.Error("lecture en échec : aucun groupe attendu (variantes laissées seules)")
	}
	groups := domain.NewGroupesDeCartes(map[string]string{"v": "b"})
	if g := Load(context.Background(), lecteur{groups: groups}); !g.Regroupe("v") {
		t.Error("lecture réussie : la table doit être rendue")
	}
}

func TestRefKeyEtRefLabel(t *testing.T) {
	groups := domain.NewGroupesDeCartes(map[string]string{"v": "b"}).
		AvecLibelles(map[string]map[string]string{"b": {"en": "Base", "fr": "Socle"}})
	v := &canonical.AssetReference{ID: "v", DefaultLabel: "Variante"}
	seule := &canonical.AssetReference{ID: "s", DefaultLabel: "Seule"}
	if RefKey(v, groups) != "b" || RefKey(seule, groups) != "s" || RefKey(nil, groups) != "" {
		t.Error("RefKey : la variante rejoint sa base, une carte seule garde son identifiant, nil rend vide")
	}
	if got := RefLabel(v, groups, "fr", "Variante"); got != "Socle" {
		t.Errorf("RefLabel(variante) = %q, attendu Socle", got)
	}
	if got := RefLabel(seule, groups, "fr", "Seule"); got != "Seule" {
		t.Errorf("RefLabel(seule) = %q, attendu le libellé propre", got)
	}
}
