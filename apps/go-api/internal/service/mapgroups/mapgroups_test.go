package mapgroups

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
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
