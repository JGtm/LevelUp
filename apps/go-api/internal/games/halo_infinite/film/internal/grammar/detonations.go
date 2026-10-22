package grammar

// detonations.go — LES DETONATIONS/IMPACTS DE PROJECTILE D UN FILM : l instant, la position
// dequantifiee et la reference de projectile de chaque message `projectile_detonate` (genre 5) ou
// `projectile_impact_effect` (genre 6) de la vue A (vue_a_detonations.go, lot detonations du
// 2026-10-22).
//
// LECTURE SEULE, sans seuil : ce balayage rend ce que la vue A ecrit. Une liste arretee avant son
// message de detonation ne le rend pas ; une reference ou une position non lues sortent comptees
// (ProjectileDetonationStats), jamais devinees.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ScanProjectileDetonations lit les detonations/impacts de projectile d un film DEJA CHARGE, dans
// la passe des tetes, tries par instant.
func ScanProjectileDetonations(fc *FilmContext) ([]types.ProjectileDetonation, types.ProjectileDetonationStats) {
	c := &canalDesDetonations{}
	distribuerLesTetesSeules(fc, []Canal{c})
	return c.out, c.st
}

// canalDesDetonations recueille les messages de detonation/impact de la vue A de chaque trame.
type canalDesDetonations struct {
	out []types.ProjectileDetonation
	st  types.ProjectileDetonationStats
}

func (*canalDesDetonations) Interets() []Interet { return nil }

func (c *canalDesDetonations) Tete(p *lecture.Paquet) {
	for _, m := range p.VueA.Detonations {
		c.st.Read++
		d := types.ProjectileDetonation{TimestampUS: p.TS, Genre: m.Genre}
		if aLeDrapeauDetonation(m, lecture.DetonationRef0) {
			d.Projectile = types.LifeKey{Slot: uint32(m.Ref0), Gen: uint32(m.Gen0)}
			d.HasProjectile = true
			c.st.WithProjectile++
		}
		if aLeDrapeauDetonation(m, lecture.DetonationPositionLue) {
			d.X, d.Y, d.Z, d.HasPosition = m.X, m.Y, m.Z, true
			c.st.WithPosition++
		}
		c.out = append(c.out, d)
	}
}

func (c *canalDesDetonations) Clore(BilanDeMarche) { trierLesDetonations(c.out) }

// trierLesDetonations range les detonations dans un ordre TOTAL : instant, puis l ordre du film
// pour les ex aequo restants (meme principe que trierLesPoussees, poussees.go).
func trierLesDetonations(out []types.ProjectileDetonation) {
	slices.SortStableFunc(out, func(a, b types.ProjectileDetonation) int {
		return cmp.Compare(a.TimestampUS, b.TimestampUS)
	})
}
