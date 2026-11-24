package replaybuild

import (
	"context"
	"testing"

	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// TestAvecFils_LaValeurAtteintLOptionDuDecodeur : le nombre de fils de l'installation arrive dans
// `replay.Options.Vitesse`, l'unique porte de reglage de la vitesse du decodeur ; sans reglage le decodeur garde
// un fil (valeur zero).
func TestAvecFils_LaValeurAtteintLOptionDuDecodeur(t *testing.T) {
	options := func(b *Builder) int {
		b.geometryOverride = []replay.MapObject{}
		b.structures = map[string][]replay.Surface{}
		b.energieTried = true
		o := b.buildReplayOptions(context.Background(), decfilm.MapQuantEntry{}, port.MatchFacts{}, entreesCatalogue{}, &filmStats{})
		return o.Vitesse.Fils
	}
	if got := options(&Builder{}); got != 0 {
		t.Errorf("sans reglage : %d fils, attendu 0 (un fil)", got)
	}
	for _, n := range []int{1, 4} {
		if got := options((&Builder{}).AvecFils(n)); got != n {
			t.Errorf("AvecFils(%d) : %d fils dans les options du decodeur", n, got)
		}
	}
}
