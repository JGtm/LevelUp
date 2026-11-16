package tactical

import (
	"reflect"
	"sort"
	"testing"
)

// matchsDeSpawns : deux matchs ; le joueur « moi » a, dans chacun, quatre vies : la premiere, une
// reapparition, un debut de manche et une piste sans drapeau.
func matchsDeSpawns() map[string][]JoueurLu {
	vies := func(dx float64) []SpawnLu {
		return []SpawnLu{
			{Frame: 0, X: dx, PremiereVie: true},
			{Frame: 1000, X: 50 + dx, Reapparition: true},
			{Frame: 2000, X: 100 + dx, DebutDeManche: true},
			{Frame: 3000, X: 150 + dx},
		}
	}
	return map[string][]JoueurLu{
		"m1": {{XUID: "moi", Equipe: 0, Spawns: vies(0)}, {XUID: "autre", Equipe: 1, Spawns: vies(1)}},
		"m2": {{XUID: "moi", Equipe: 0, Spawns: vies(2)}},
	}
}

func xsTries(points []PointSpawn) []float64 {
	out := make([]float64, 0, len(points))
	for _, p := range points {
		out = append(out, p.X)
	}
	sort.Float64s(out)
	return out
}

func TestSpawnsDeDepart_PremiereVieSeule(t *testing.T) {
	if got, want := xsTries(SpawnsDeDepart(matchsDeSpawns(), "moi")), []float64{0, 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("departs = %v, attendu %v (un par match, le joueur seul)", got, want)
	}
}

func TestApparitionsDuJoueur_ToutesLesVies(t *testing.T) {
	got := xsTries(ApparitionsDuJoueur(matchsDeSpawns(), "moi"))
	// Premiere vie, reapparition et debut de manche de chaque match ; la piste sans drapeau n'en est pas une.
	if want := []float64{0, 2, 50, 52, 100, 102}; !reflect.DeepEqual(got, want) {
		t.Errorf("apparitions = %v, attendu %v", got, want)
	}
}

func TestSpawnsSelonVies_ChoisitLaDefinition(t *testing.T) {
	m := matchsDeSpawns()
	if len(SpawnsSelonVies(m, "moi", true)) != 6 || len(SpawnsSelonVies(m, "moi", false)) != 2 {
		t.Error("`toutes` rend les apparitions de toutes les vies, sinon la premiere vie seule")
	}
	if got := SpawnsSelonVies(m, "absent", true); len(got) != 0 {
		t.Errorf("un joueur absent n'a aucun spawn : %v", got)
	}
}

func TestRoutesAuxVies_JointureParInstant(t *testing.T) {
	debuts := []int{0, 1000, 2000, 3000}
	if got, want := RoutesAuxVies(debuts, map[int]bool{1000: true, 3000: true}), []int{1, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("routes retenues = %v, attendu %v", got, want)
	}
	if got := RoutesAuxVies(debuts, nil); len(got) != 0 {
		t.Errorf("aucun instant retenu : aucune route, obtenu %v", got)
	}
}

// Ce qui est une reapparition adverse : une reapparition apres mort d'un joueur de l'AUTRE equipe. Ni un
// debut de manche, ni une premiere vie, ni un equipier, ni un joueur sans equipe.
func TestObservationsAdversesDuMatch_QuiCompte(t *testing.T) {
	instantane := []Vivant{{XUID: "moi", Equipe: 0, X: 2, Y: 2}, {Equipe: 1, X: 60, Y: 60}}
	joueurs := []JoueurLu{
		{XUID: "moi", Equipe: 0, Spawns: []SpawnLu{{Frame: 0, PremiereVie: true}}},
		{XUID: "ami", Equipe: 0, Spawns: []SpawnLu{{Frame: 1500, Reapparition: true, Instantane: instantane}}},
		{XUID: "adv", Equipe: 1, Spawns: []SpawnLu{
			{Frame: 0, PremiereVie: true},
			{Frame: 1000, X: 25, Y: 5, Z: 1, Reapparition: true, Instantane: instantane},
			{Frame: 2000, DebutDeManche: true, Instantane: instantane},
			{Frame: 3000},
		}},
		{XUID: "sans-equipe", Equipe: -1, Spawns: []SpawnLu{{Frame: 1700, Reapparition: true, Instantane: instantane}}},
	}
	obs, connue := ObservationsAdversesDuMatch("m1", joueurs, "moi")
	if !connue || len(obs) != 1 {
		t.Fatalf("(%d observations, equipe connue %v), attendu 1 (la seule reapparition adverse)", len(obs), connue)
	}
	if o := obs[0]; o.MatchID != "m1" || o.X != 25 || o.Y != 5 || o.Z != 1 || o.EquipeActive != 0 || len(o.Instantane) != 2 {
		t.Errorf("observation = %+v", o)
	}
	if _, connue := ObservationsAdversesDuMatch("m1", joueurs, "inconnu"); connue {
		t.Error("un joueur absent du match n'a pas d'equipe connue")
	}
	joueurs[0].Equipe = EquipeInconnue
	if obs, connue := ObservationsAdversesDuMatch("m1", joueurs, "moi"); connue || obs != nil {
		t.Errorf("equipe du joueur inconnue : le match ne sert pas la lecture, obtenu (%v, %v)", obs, connue)
	}
}
