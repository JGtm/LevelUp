package prestige

import (
	"reflect"
	"testing"
)

func TestDecodeStringList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"vide → nil", "", nil},
		{"un élément", "kills", []string{"kills"}},
		{"plusieurs", "kills,deaths,wins", []string{"kills", "deaths", "wins"}},
		{"virgule finale → élément vide conservé", "a,", []string{"a", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeStringList(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
