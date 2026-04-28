// Tests purs (sans DB) pour newFriendsAdded — §6 plan Squad/Sessions overhaul.
//
// La fonction calcule le diff prev → next sur friend_gamertags pour identifier
// quels gamertags sont nouveaux et déclencher la notif friend_added.
// Couplé à friendGamertagsChanged (déjà testé) qui détermine si le diff existe.
package handlers

import (
	"reflect"
	"testing"
)

func TestNewFriendsAdded(t *testing.T) {
	tests := []struct {
		name string
		prev []string
		next []string
		want []string
	}{
		{
			name: "ajout simple",
			prev: []string{"Alpha"},
			next: []string{"Alpha", "Bravo"},
			want: []string{"Bravo"},
		},
		{
			name: "ajouts multiples",
			prev: []string{},
			next: []string{"Alpha", "Bravo", "Charlie"},
			want: []string{"Alpha", "Bravo", "Charlie"},
		},
		{
			name: "retrait pur (rien d'ajouté)",
			prev: []string{"Alpha", "Bravo"},
			next: []string{"Alpha"},
			want: nil,
		},
		{
			name: "rien changé (set égal)",
			prev: []string{"Alpha", "Bravo"},
			next: []string{"Alpha", "Bravo"},
			want: nil,
		},
		{
			name: "ré-ordonnancement (set égal, pas d'ajout)",
			prev: []string{"Alpha", "Bravo"},
			next: []string{"Bravo", "Alpha"},
			want: nil,
		},
		{
			name: "case-insensitive : pas considéré comme ajout",
			prev: []string{"Alpha"},
			next: []string{"ALPHA"},
			want: nil,
		},
		{
			name: "trim : pas considéré comme ajout",
			prev: []string{"Alpha"},
			next: []string{" Alpha "},
			want: nil,
		},
		{
			name: "ajout préserve la casse next (pas la casse normalisée)",
			prev: []string{"alpha"},
			next: []string{"alpha", "BRAVO"},
			want: []string{"BRAVO"},
		},
		{
			name: "swap (1 retiré + 1 ajouté)",
			prev: []string{"Alpha"},
			next: []string{"Bravo"},
			want: []string{"Bravo"},
		},
		{
			name: "from empty",
			prev: nil,
			next: []string{"Alpha"},
			want: []string{"Alpha"},
		},
		{
			name: "to empty",
			prev: []string{"Alpha"},
			next: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newFriendsAdded(tt.prev, tt.next)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newFriendsAdded(%v, %v) = %v, want %v", tt.prev, tt.next, got, tt.want)
			}
		})
	}
}
