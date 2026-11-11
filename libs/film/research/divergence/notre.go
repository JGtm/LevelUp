//go:build research

package divergence

// notre.go : NOTRE lecture, par les seuls points d entree de la grammaire. Deux marches du meme
// film, la meme que celle de production :
//
//   - `grammar.FrameClosureDetaillee` rend le verdict et la CAUSE d arret de chaque paquet, telle
//     que la carte de fermeture (A0.1) les classe ;
//   - `FilmContext.Trames` rend la structure de lecture de chaque paquet (`lecture.Paquet`) : vues,
//     records et composants, chacun avec son etendue en bits depuis le debut de la charge utile.
//
// Aucun bit n est relu ici : tout vient de ce que la marche a rangee. La seconde marche confirme la
// premiere (meme verdict de fermeture paquet par paquet) ; l ecart eventuel est compte et rendu.

import (
	"fmt"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// cleDePaquet situe un paquet : (chunk, rang dans le chunk, tous types confondus).
type cleDePaquet struct{ chunk, index int }

// verdictNous : ce que la carte de fermeture dit d un paquet.
type verdictNous struct {
	fermee bool
	cause  string
}

// lireLesVerdicts fait la premiere marche : le verdict de chaque paquet delta.
func lireLesVerdicts(fc *grammar.FilmContext) (map[cleDePaquet]verdictNous, error) {
	out := map[cleDePaquet]verdictNous{}
	_, err := grammar.FrameClosureDetaillee(fc, grammar.UsagesProduit{}, func(d grammar.PaquetDeCarte) {
		out[cleDePaquet{d.Chunk, d.Index}] = verdictNous{fermee: d.Fermee, cause: d.Cause}
	})
	if err != nil {
		return nil, fmt.Errorf("carte de fermeture detaillee : %w", err)
	}
	return out, nil
}
