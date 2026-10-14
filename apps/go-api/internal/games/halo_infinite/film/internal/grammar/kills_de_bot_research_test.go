//go:build research

package grammar

// kills_de_bot_research_test.go — L INSTRUMENT DU LOT `assist-film` (2026-10-09) : LE FILM ECRIT-IL UN
// `PlayerKilledEvent` (genre 85) QUAND LA VICTIME EST UN BOT ?
//
// La recherche est EXHAUSTIVE et ne depend d aucune lecture de liste : a chaque bit de chaque trame
// delta, la tete d un message de kill (continuation a 1 puis genre 85, [estAncreDeKill]), ses
// presences ([evPresence]), ses champs lus sans queue ([lireLeKillDeLaChaine]) et la grammaire
// minimale ([killPlausible]). Elle rend donc tout ce que la vue A et le rattrapage pourraient lire,
// et davantage : aucun seuil de chaine n est applique, la longueur de la chaine derriere chaque tete
// ([evChainLen]) est seulement journalisee.
//
// Pour chaque mort donnee (`instant:victime:tueur`, indices de replication), elle compte les tetes
// qui ecrivent CE couple a moins de [toleranceInstant] de l instant, a n importe quel bit.
//
// LECTURE SEULE, gardee par une variable — sautee partout ailleurs, CI comprise :
//
//	KB_FILM=<repo>/data/cache/film_chunks/0a08d2f2 KB_MORTS=438625:8:1,510209:10:3,584132:3:11 \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestKillsDeBotDansLeFilm$' -v -count=1
//
// Les couples se lisent dans le journal de `killsource.TestBotsParInstant` (dead-states a indice de
// bot) ; une mort infligee PAR un bot y sert de temoin positif.

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// toleranceInstant : l ecart admis entre l instant du kill-feed et la trame d une tete, en
// millisecondes (4 a 6 ms mesures sur les morts infligees par un bot ; la marge couvre deux trames).
const toleranceInstant = 40

// mortCherchee : un couple attendu a un instant, et les tetes trouvees.
type mortCherchee struct {
	ms               int
	victime, tueur   int8
	tetes, enChaines int
}

func TestKillsDeBotDansLeFilm(t *testing.T) {
	dir := os.Getenv("KB_FILM")
	if dir == "" {
		t.Skip("KB_FILM absent : mesure sautee")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	morts := mortsCherchees(t, os.Getenv("KB_MORTS"))
	base := origineDuFilm(film)
	gate15 := trancherGate15(film)
	total := 0
	for _, p := range film.AllPackets() {
		if p.Type != int(PacketTypeDelta) {
			continue
		}
		ms := int((p.TS - base) / 1000)
		nb := len(p.Payload) * 8
		for x := 1; x+8 <= nb; x++ {
			if !estAncreDeKill(p.Payload, x) {
				continue
			}
			r := nouveauCurseurEv(p.Payload, x+LargeurGenreVueA)
			if !evPresence(r, GenreJoueurTue) {
				continue
			}
			k, fin := lireLeKillDeLaChaine(p.Payload, r.pos())
			if !killPlausible(k, fin) {
				continue
			}
			total++
			n := evChainLen(p.Payload, fin, gate15, maxChainProbe)
			for i := range morts {
				m := &morts[i]
				if d := ms - m.ms; d < -toleranceInstant || d > toleranceInstant ||
					k.Victime != m.victime || k.Tueur != m.tueur {
					continue
				}
				m.tetes++
				if n >= minChain {
					m.enChaines++
				}
				t.Logf("   %7d ms  chunk %2d trame %4d bit %6d  victime %2d tueur %2d assistant %2d  chaine %2d",
					ms, p.Chunk, p.Index, x-1, k.Victime, k.Tueur, k.Assistant, n)
			}
		}
	}
	t.Logf("tetes de genre 85 plausibles dans le film : %d (gate15 %v)", total, gate15)
	for _, m := range morts {
		t.Logf("mort %7d ms  victime %2d tueur %2d : %d tete(s) a ce couple, dont %d en chaine de %d",
			m.ms, m.victime, m.tueur, m.tetes, m.enChaines, minChain)
	}
}

// mortsCherchees lit `KB_MORTS` : `instant:victime:tueur`, separes par des virgules.
func mortsCherchees(t *testing.T, v string) []mortCherchee {
	t.Helper()
	var out []mortCherchee
	for _, s := range strings.Split(v, ",") {
		c := strings.Split(strings.TrimSpace(s), ":")
		if len(c) != 3 {
			continue
		}
		ms, e1 := strconv.Atoi(c[0])
		vi, e2 := strconv.Atoi(c[1])
		tu, e3 := strconv.Atoi(c[2])
		if e1 != nil || e2 != nil || e3 != nil {
			t.Fatalf("KB_MORTS : %q illisible", s)
		}
		out = append(out, mortCherchee{ms: ms, victime: int8(vi), tueur: int8(tu)}) //nolint:gosec // indices < 32
	}
	return out
}

// origineDuFilm : le plus petit horodatage des paquets de replication, l origine des instants de
// killsource.
func origineDuFilm(f *source.Film) uint64 {
	var base uint64
	vu := false
	for _, p := range f.AllPackets() {
		if p.Type == 0 && (!vu || p.TS < base) {
			base, vu = p.TS, true
		}
	}
	return base
}
