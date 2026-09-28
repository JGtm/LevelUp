package grammar

// bobines_memo_test.go — LES DECODAGES DE BOBINE QUE PLUSIEURS TESTS DU PAQUET REFONT A L IDENTIQUE,
// CALCULES UNE FOIS PAR PROCESSUS DE TEST (temps CI du paquet, 2026-09-28).
//
// # POURQUOI
//
// Le paquet depassait le `-timeout 900s` du job de couverture (commande CI : `-coverpkg=./...
// -covermode=atomic -tags=integration -p 1`) : mesure locale 1 334 s, tous tests en serie. La
// marche d image-cle des sept bobines par build coute ~90 s sous couverture, et plusieurs tests la
// refaisaient pour la MEME sortie : la fermeture par archetype ([KeyframeClosure]) pour le ratchet
// d image-cle ET la cible `ti=9`, la lecture des equipes ([ScanPlayerTeams]) pour le gel des comptes
// ET les entites `ti=9`.
//
// # CE QUI EST PARTAGE, ET CE QUI NE L EST PAS
//
// Seule la SORTIE d un point d entree de production sur une bobine INTACTE, chargee par le memo
// lui-meme (aucun test ne recoit le `*source.Film` ni le [FilmContext] du calcul : rien ne peut les
// muter). Chaque appelant recoit une COPIE de ce qui est mutable (cartes, tranches). Les assertions
// ne changent pas : elles portent sur la meme valeur, calculee par le meme appel.
//
// Le temoin EN SERIE de `TestDeuxFilmsEnParallele` n y passe PAS : il doit etre decode dans son test,
// hors de toute concurrence.
//
// # PAS DE `t.Parallel()` DANS CE PAQUET : MESURE, 2026-09-28
//
// Sous `-covermode=atomic`, chaque bloc execute incremente un compteur ATOMIQUE partage ; des tests
// paralleles qui marchent les memes lecteurs de bits se disputent les memes lignes de cache. Quatre
// tests de ce paquet : 36,8 s en serie, 74,7 s avec `-parallel 4` (GOMAXPROCS=4) — deux fois PLUS
// lent. Le memo n en reste pas moins sur sous concurrence ([sync.OnceValues] par cle).

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// memoDeBobine calcule, par cle, UNE valeur par processus de test.
type memoDeBobine[V any] struct {
	mu     sync.Mutex
	calcul map[string]func() (V, error)
}

// valeur rend la valeur de `cle`, calculee par `calcul` au premier appel seulement.
func (m *memoDeBobine[V]) valeur(cle string, calcul func() (V, error)) (V, error) {
	m.mu.Lock()
	f, ok := m.calcul[cle]
	if !ok {
		if m.calcul == nil {
			m.calcul = map[string]func() (V, error){}
		}
		f = sync.OnceValues(calcul)
		m.calcul[cle] = f
	}
	m.mu.Unlock()
	return f()
}

// fermeturesDeBobine : [KeyframeClosure] par repertoire de bobine (chemin nettoye).
var fermeturesDeBobine memoDeBobine[map[uint32]KeyframeClosureStat]

// fermetureMemo rend une COPIE de la fermeture par archetype de la bobine `dir`, decodee une fois.
func fermetureMemo(dir string) (map[uint32]KeyframeClosureStat, error) {
	stats, err := fermeturesDeBobine.valeur(filepath.Clean(dir), func() (map[uint32]KeyframeClosureStat, error) {
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			return nil, fmt.Errorf("LoadDir %s : %w", dir, err)
		}
		s, err := KeyframeClosure(NewFilmContext(film))
		if err != nil {
			return nil, fmt.Errorf("KeyframeClosure %s : %w", dir, err)
		}
		return s, nil
	})
	return maps.Clone(stats), err
}

// lectureDesEquipes : les trois sorties de [ScanPlayerTeams] sur une bobine intacte.
type lectureDesEquipes struct {
	teams map[int]int
	rep   TeamScanReport
	ents  PlayerEntityScan
}

// equipesDeBobine : [ScanPlayerTeams] par bobine par build (nom court).
var equipesDeBobine memoDeBobine[lectureDesEquipes]

// equipesMemo rend une COPIE de ce que [ScanPlayerTeams] lit sur la bobine par build `film`,
// decodee une fois. Meme chargement que [bobineFilm].
func equipesMemo(t *testing.T, film string) (map[int]int, TeamScanReport, PlayerEntityScan) {
	t.Helper()
	l, err := equipesDeBobine.valeur(film, func() (lectureDesEquipes, error) {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+film)
		f, err := source.LoadDir(dir, nil)
		if err != nil {
			return lectureDesEquipes{}, fmt.Errorf("bobine %s : %w — regenerer les bobines du lot 0.A.2", film, err)
		}
		teams, rep, ents := ScanPlayerTeams(NewFilmContext(f))
		return lectureDesEquipes{teams: teams, rep: rep, ents: ents}, nil
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
	ents := l.ents
	ents.KeyframesUS = slices.Clone(ents.KeyframesUS)
	ents.Entities = slices.Clone(ents.Entities)
	ents.Doutes = slices.Clone(ents.Doutes)
	return maps.Clone(l.teams), l.rep, ents
}
