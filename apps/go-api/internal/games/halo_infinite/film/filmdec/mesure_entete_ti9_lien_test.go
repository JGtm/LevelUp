package filmdec

// mesure_entete_ti9_lien_test.go — MESURE M, VOLET 3 bis : L'ORDRE DES ENTITES ti=9 EST-IL
// L'ORDRE DES INDEX DE JOUEUR DU FILM ?
//
// LA QUESTION QUE LE FORK N'A PAS TRANCHEE. Huit designateurs d'equipe repartis 4-4 ne
// servent a rien sans identite. Le volet 3 a montre que l'etat par defaut de ti=9 ne porte
// aucun index (ses deux R(6) valent zero sur tous les records des trois films). Reste UNE
// piste, et elle ne demande aucune nouvelle retro-ingenierie : les huit entites ti=9
// occupent des slots CONSECUTIFS (de deux en deux, intercales avec ti=47), et les quatre
// premiers portent une valeur de designateur, les quatre derniers l'autre. Si l'ordre des
// slots etait l'ordre des INDEX DE JOUEUR du film — celui que `weaponv3.ResolveXuidToPI`
// lit deja, et qui est mesure juste sur 116 films sur 116 — alors le lien serait gratuit.
//
// LE TEST. La base dit quels xuids jouent dans quelle equipe. Le film dit quel index de
// joueur porte chaque xuid. Si l'ordre des slots ti=9 est l'ordre des index, alors les
// quatre joueurs d'UNE equipe doivent occuper un BLOC CONTIGU d'index — {0,1,2,3} ou
// {4,5,6,7} — et jamais un entrelacement. C'est une condition NECESSAIRE, pas suffisante :
// si elle tombe, la piste est morte ; si elle tient, elle reste a confirmer sur l'ordre
// exact (quelle equipe est le bloc bas).
//
// LANCEMENT (la verite vient de la base, passee en clair : le banc ne lit aucune DB) :
//
//	MESURE_TI9_ROOT=<...>/data/cache/film_chunks MESURE_TI9_IDS=a,b,c \
//	MESURE_TI9_ROSTER='a:<xuid>/<team>,...;b:...' \
//	  go test ./internal/analysis/filmdec/ -run MesureLienTI9 -v -timeout 60m

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/weaponv3"
)

const mesureTI9RosterEnv = "MESURE_TI9_ROSTER"

// mesureRoster est la verite externe d'un film : xuid -> equipe de la table des scores.
type mesureRoster map[uint64]int

// mesureLitRosters decoupe MESURE_TI9_ROSTER : `film:xuid/team,xuid/team;film:...`.
func mesureLitRosters(t *testing.T) map[string]mesureRoster {
	t.Helper()
	raw := os.Getenv(mesureTI9RosterEnv)
	if raw == "" {
		t.Skipf("%s absent : confrontation a la table des scores sautee", mesureTI9RosterEnv)
	}
	out := map[string]mesureRoster{}
	for _, bloc := range strings.Split(raw, ";") {
		film, liste, ok := strings.Cut(strings.TrimSpace(bloc), ":")
		if !ok {
			continue
		}
		r := mesureRoster{}
		for _, e := range strings.Split(liste, ",") {
			x, team, ok := strings.Cut(strings.TrimSpace(e), "/")
			if !ok {
				continue
			}
			xu, err1 := strconv.ParseUint(x, 10, 64)
			tm, err2 := strconv.Atoi(team)
			if err1 != nil || err2 != nil {
				t.Fatalf("%s : entree illisible %q", mesureTI9RosterEnv, e)
			}
			r[xu] = tm
		}
		out[film] = r
	}
	return out
}

// mesureIndexJoueurs lit l'index de joueur de chaque xuid du roster dans les chunks de
// REPLICATION du film, et EXIGE que tous les chunks lisibles concordent — la regle de
// `replay.ScanPlayerIndices` : deux lectures divergentes ne sont pas un desaccord a arbitrer,
// c'est le symptome d'une lecture fausse.
func mesureIndexJoueurs(f mesureFilm, roster mesureRoster) (map[uint64]int, int, int) {
	xuids := make([]uint64, 0, len(roster))
	for x := range roster {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(a, b int) bool { return xuids[a] < xuids[b] })
	table := map[uint64]int{}
	lectures, desaccords := 0, 0
	for c := 1; c <= f.chunks; c++ {
		chunk, err := ReadFilmChunk(f.dir, c)
		if err != nil {
			continue
		}
		got := weaponv3.ResolveXuidToPI(xuids, chunk)
		if len(got) != len(xuids) {
			continue // chunk sans enregistrement de joueur : ce n'est pas une lecture
		}
		lectures++
		for x, pi := range got {
			if prev, seen := table[x]; seen && prev != pi {
				desaccords++
				continue
			}
			table[x] = pi
		}
	}
	return table, lectures, desaccords
}

func TestMesureLienTI9(t *testing.T) {
	films := mesureOuvre(t)
	rosters := mesureLitRosters(t)
	defer LockProcessDecode()()

	t.Logf("")
	t.Logf("=== G. INDEX DE JOUEUR DU FILM CONFRONTES AUX EQUIPES DE LA BASE ===")
	for _, f := range films {
		roster := rosters[f.id]
		if len(roster) == 0 {
			t.Logf("%10s : aucun roster fourni", f.id)
			continue
		}
		table, lectures, desaccords := mesureIndexJoueurs(f, roster)
		t.Logf("%10s : %d lectures concordantes, %d desaccords, %d/%d xuids resolus",
			f.id, lectures, desaccords, len(table), len(roster))
		parEquipe := map[int][]int{}
		for x, pi := range table {
			parEquipe[roster[x]] = append(parEquipe[roster[x]], pi)
		}
		equipes := make([]int, 0, len(parEquipe))
		for e := range parEquipe {
			equipes = append(equipes, e)
		}
		sort.Ints(equipes)
		for _, e := range equipes {
			idx := parEquipe[e]
			sort.Ints(idx)
			t.Logf("%10s   equipe %d -> index de joueur %v  (bloc contigu : %v)",
				f.id, e, idx, mesureBlocContigu(idx))
		}
	}
	t.Logf("")
	t.Logf("Condition NECESSAIRE pour « rang de slot ti=9 = index de joueur » : chaque equipe")
	t.Logf("occupe un bloc contigu d'index. Un entrelacement suffit a tuer la piste.")
}

// mesureBlocContigu dit si des index tries se suivent sans trou.
func mesureBlocContigu(idx []int) bool {
	for i := 1; i < len(idx); i++ {
		if idx[i] != idx[i-1]+1 {
			return false
		}
	}
	return len(idx) > 0
}
