//go:build research

package grenadeids

// bobine.go — OUVRIR UN FILM ET LUI DEMANDER SON IDENTITE, SON REGISTRE ET SON MARQUEUR.
//
// Tout ce qui est lu ici vient du film lui-meme : la version majeure de l en-tete de
// `chunk_00`, le build en clair de sa section d identification, et le registre d archetypes.
// Rien n est suppose du build : l archetype projectile se resout PAR LE NOM de ses composants.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// amorceEtatParDefaut : les 19 bits de poids faible du « marqueur » de production, c est-a-dire
// l amorce constante de l etat par defaut qui suit les cinq bits bas du typeIndex dans un record
// de creation d entite. `0x4C0C00 & 0x7FFFF` == `0x40C00`.
const amorceEtatParDefaut uint64 = 0x40C00

// marqueurDeProduction : la constante 24 bits que `grammar.ScanGrenadeThrows` cherche. Recopiee
// ici parce qu elle est privee dans `grammar` ; l instrument VERIFIE qu elle est bien
// `MarqueurDe(grammar.ProjectileTypeIndex)`, il ne lui fait pas confiance sur parole.
const marqueurDeProduction uint64 = 0x4C0C00

// composantsProjectile : les noms qui identifient l archetype projectile dans un registre, quel
// que soit son rang. Source : `grammar/projectiles.go`, en-tete, point 1.
var composantsProjectile = []string{
	"projectile-at-rest-state",
	"projectile-tether-state",
	"projectile-command_tick",
	"projectile-deceleration-disabled-state",
}

// Bobine est un film ouvert, avec ce que son chunk_00 dit de lui.
type Bobine struct {
	// ID est le nom du repertoire du film (les huit caracteres du match).
	ID string
	// Version est la version majeure de format, u32 LE des quatre premiers octets de chunk_00.
	Version int
	// Build est la chaine `HI_1_x_y` de la section d identification ; vide sur les films
	// anterieurs a cette section.
	Build string
	// TiProjectile est l index d archetype dont les composants portent les noms de projectile,
	// resolu DANS CE FILM ; -1 quand aucun bloc ne les porte.
	TiProjectile int
	// NomsProjectileVus est le nombre de noms de `composantsProjectile` trouves dans le bloc
	// retenu — il dit a quel point la resolution est franche.
	NomsProjectileVus int
	// Marqueur est le marqueur derive de TiProjectile ; MarqueurProd est celui de production.
	Marqueur     uint64
	MarqueurProd uint64
	// Archetypes est le nombre de blocs du registre de ce film.
	Archetypes int
	// Ambiguites : les autres archetypes de CE registre qui produiraient le meme marqueur.
	Ambiguites []int

	film *source.Film
}

// Film rend le film charge.
func (b *Bobine) Film() *source.Film { return b.film }

// MarqueurDe rend le marqueur de 24 bits d un archetype : ses cinq bits bas, puis l amorce.
func MarqueurDe(ti int) uint64 {
	return (uint64(ti)&0x1F)<<19 | amorceEtatParDefaut
}

// VerifierMarqueurDeProduction controle que le « marqueur » de production est bien la valeur que
// [MarqueurDe] derive de l archetype projectile du build courant. C est l invariant sur lequel
// toute la lecture « le marqueur est une donnee de build » repose : s il tombe, la derivation est
// fausse et la mesure ne vaut rien.
func VerifierMarqueurDeProduction() error {
	if got := MarqueurDe(grammar.ProjectileTypeIndex); got != marqueurDeProduction {
		return fmt.Errorf("derivation du marqueur fausse : MarqueurDe(%d) = 0x%06X, attendu 0x%06X",
			grammar.ProjectileTypeIndex, got, marqueurDeProduction)
	}
	return nil
}

// AmbiguitesDuMarqueur rend les autres index d archetype qui produiraient LE MEME marqueur que
// `ti`, dans un registre de `blocs` blocs. Le marqueur ne porte que CINQ bits de l index : sur un
// registre de cinquante archetypes, `ti` et `ti-32` sont indiscernables.
func AmbiguitesDuMarqueur(ti, blocs int) []int {
	var out []int
	for autre := ti % 32; autre < blocs; autre += 32 {
		if autre != ti {
			out = append(out, autre)
		}
	}
	return out
}

// Ouvrir charge le film de `racine/id` et lit son identite, son registre et son marqueur.
//
// LECTURE SEULE : rien n est ecrit, ni sous la racine ni ailleurs.
func Ouvrir(racine, id string) (*Bobine, error) {
	film, err := source.LoadDir(cheminFilm(racine, id), nil)
	if err != nil {
		return nil, fmt.Errorf("film illisible : %w", err)
	}
	b := &Bobine{ID: id, film: film, TiProjectile: -1, MarqueurProd: marqueurDeProduction}
	if v, lue := grammar.FilmMajorVersion(film); lue {
		b.Version = v
	}
	raw, ok := grammar.FilmRegistryChunk(film)
	if !ok {
		b.Marqueur = marqueurDeProduction
		return b, nil
	}
	if ident, errID := grammar.ReadFilmIdentity(raw); errID == nil {
		b.Build = ident.Build
	}
	reg, errReg := grammar.ParseRegistryChunk(raw)
	if errReg != nil || reg == nil {
		b.Marqueur = marqueurDeProduction
		return b, nil
	}
	b.Archetypes = len(reg.Archetypes)
	b.TiProjectile, b.NomsProjectileVus = tiProjectileParNom(reg)
	if b.TiProjectile < 0 {
		b.Marqueur = marqueurDeProduction
		return b, nil
	}
	b.Marqueur = MarqueurDe(b.TiProjectile)
	b.Ambiguites = AmbiguitesDuMarqueur(b.TiProjectile, b.Archetypes)
	return b, nil
}

// tiProjectileParNom rend l index d archetype qui porte le plus de noms de projectile, et ce
// compte. Un bloc qui n en porte aucun ne peut pas gagner : le second retour vaut alors zero et
// le premier -1.
func tiProjectileParNom(reg *grammar.Registry) (int, int) {
	meilleur, vus := -1, 0
	for _, a := range reg.Archetypes {
		n := 0
		for _, nom := range composantsProjectile {
			if contientComposant(a, nom) {
				n++
			}
		}
		if n > vus {
			meilleur, vus = a.Index, n
		}
	}
	return meilleur, vus
}

// contientComposant dit si l archetype porte ce nom de composant.
func contientComposant(a grammar.Archetype, nom string) bool {
	for _, c := range a.Components {
		if c == nom {
			return true
		}
	}
	return false
}
