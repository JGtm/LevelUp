package replay

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
	"levelup/go-api/internal/testutil"
)

// flag_spawn_neutral_label_test.go — LE LABEL TRANCHE LA NEUTRALITE D'UN SOCLE DE DRAPEAU,
// PAS LE `team_index` (D9, corrige le 2026-09-13).
//
// CE QUI ETAIT FAUX. Le socle CENTRAL d'Illusion (`9e821f5e`, `ctf_illusion.mvar`,
// object_index 201, au point (0, 0)) porte `ctf_neutral_include` ET `team_index = 0`. Tout
// l'aval triait la neutralite sur le seul `team_index` : ce socle devenait donc un TROISIEME
// drapeau d'equipe 0, fige au milieu de la carte, que `flag_neutral.go` ne pouvait pas
// ecarter — la plus grande zone aveugle du parc (rapport 6.11, decouverte D1, mesuree sur
// `bc60b4d9`).
//
// CE QUE CES TESTS FIGENT. (1) `IsCTFNeutral` lit le label et rien d'autre ; (2) la
// projection `PointsOfRole` le reporte dans `PointObjective.Neutral`, puisqu'elle laisse
// tomber `Labels` et que sans lui l'information n'existerait plus en aval ; (3) le calque
// statique publie ce socle en NEUTRE malgre son `team_index`, et ne touche PAS aux socles
// d'equipe voisins.
//
// LE CAS D'ILLUSION EST REPRODUIT A L'IDENTIQUE (memes positions, memes team_index, memes
// labels que le catalogue versionne) : ce test echoue si la regle repasse au `team_index`.

// illusionFlagSpawns reproduit les TROIS socles de drapeau d'Illusion tels que le catalogue
// versionne les porte (recense le 2026-09-13).
func illusionFlagSpawns() []mapvar.Objective {
	spawn := func(idx int, instance int32, y float64, team int, labels ...string) mapvar.Objective {
		return mapvar.Objective{
			Role:       mapvar.RoleFlagSpawn,
			TypeID:     -1430101324,
			InstanceID: instance,
			ObjectIdx:  idx,
			TeamIndex:  team,
			Pos:        mapvar.Vec3{X: 0, Y: y, Z: 0.8},
			Forward:    mapvar.Vec3{X: 1},
			Labels:     labels,
		}
	}
	return []mapvar.Objective{
		spawn(198, -1996868974, 14.8, 1, "ctf_include", "flag_spawn"),
		spawn(199, 1764626852, -14.8, 0, "ctf_include", "flag_spawn"),
		// LE SOCLE FAUTIF : central, neutre au label, equipe 0 au team_index.
		spawn(201, 1092073439, 0, 0, "ctf_neutral_include", "flag_spawn"),
	}
}

func TestIsCTFNeutral_LitLeLabelPasLEquipe(t *testing.T) {
	socles := illusionFlagSpawns()
	for _, c := range []struct {
		nom  string
		obj  mapvar.Objective
		want bool
	}{
		{"socle equipe 1", socles[0], false},
		{"socle equipe 0", socles[1], false},
		{"socle central (team_index 0, label neutre)", socles[2], true},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.obj.IsCTFNeutral(); got != c.want {
				t.Fatalf("IsCTFNeutral() = %v, attendu %v (labels %v, team_index %d)",
					got, c.want, c.obj.Labels, c.obj.TeamIndex)
			}
		})
	}
}

func TestPointsOfRole_ReporteLaNeutraliteDuLabel(t *testing.T) {
	e := MapObjectivesEntry{Objectives: illusionFlagSpawns()}
	points := e.PointsOfRole(mapvar.RoleFlagSpawn)
	if len(points) != 3 {
		t.Fatalf("%d socle(s) ponctuel(s), attendu 3", len(points))
	}
	neutres := 0
	for _, p := range points {
		if !p.Neutral {
			continue
		}
		neutres++
		// Le seul socle neutre d'Illusion est le CENTRAL. Si la projection reportait la
		// neutralite depuis le team_index, ce serait l'un des deux socles d'equipe.
		if p.Center.X != 0 || p.Center.Y != 0 {
			t.Errorf("socle neutre au point (%v, %v), attendu le centre (0, 0)", p.Center.X, p.Center.Y)
		}
		if p.TeamIndex != 0 {
			t.Errorf("team_index du socle neutre = %d, attendu 0 — la fixture ne reproduit "+
				"plus le cas d'Illusion, qui est TOUT l'objet de ce test", p.TeamIndex)
		}
	}
	if neutres != 1 {
		t.Fatalf("%d socle(s) neutre(s), attendu exactement 1", neutres)
	}
}

func TestBuildMapObjectives_SocleCentralNeutreMalgreSonTeamIndex(t *testing.T) {
	e := MapObjectivesEntry{
		MapID:      "9e821f5e-042f-407c-97f3-de165b1cdb26",
		PublicName: "Illusion",
		Objectives: illusionFlagSpawns(),
	}
	// Spec du CTF : le rôle n'est PAS forcé neutre — c'est exactement le cas qui
	// découvre le défaut, puisque seul le label peut alors trancher.
	mo := BuildMapObjectives(e, []ObjectiveRoleSpec{{Role: mapvar.RoleFlagSpawn}})
	if mo == nil {
		t.Fatal("BuildMapObjectives rend nil — les socles de drapeau ne sont plus publies ?")
	}

	var central *ObjectiveMarkerDTO
	equipes := map[int]int{}
	for i := range mo.Markers {
		m := &mo.Markers[i]
		if m.Role != string(mapvar.RoleFlagSpawn) {
			continue
		}
		equipes[m.Team]++
		if m.X == 0 && m.Y == 0 {
			central = m
		}
	}
	if central == nil {
		t.Fatal("le socle central (0, 0) n'est pas publie")
	}
	if central.Team != TeamNeutral {
		t.Errorf("socle central publie en equipe %d, attendu %d (neutre) — la neutralite est "+
			"repassee au team_index du fichier de carte, qui vaut 0 sur ce socle",
			central.Team, TeamNeutral)
	}
	// Les deux socles d'equipe gardent leur camp : la correction ne neutralise pas tout.
	if equipes[0] != 1 || equipes[1] != 1 {
		t.Errorf("socles d'equipe publies : %d en equipe 0 et %d en equipe 1, attendu 1 et 1",
			equipes[0], equipes[1])
	}
}

// ----------------------------------------------------------------------------------------
// D-B2 — LE DEFAUT SYMETRIQUE : « SANS EQUIPE » N'EST PAS « NEUTRE » (corrige le 2026-09-13)
// ----------------------------------------------------------------------------------------
//
// CE QUI ETAIT FAUX. D9 (ci-dessus) a corrige le socle qui se declarait d'une equipe alors
// qu'il etait neutre. L'inverse existe aussi : HUIT socles du catalogue portent
// `team_index = -1` — la valeur « aucun camp » — sans porter le label neutre. Ce sont des
// socles D'EQUIPE dont le fichier de carte ne dit pas l'equipe. `flag_neutral.go` triant son
// panier sur `Team == TeamNeutral`, ils y tombaient et pouvaient faire basculer un film en
// variante « drapeau neutre » a tort.
//
// CE QUE CE RECENSEMENT FIGE. Le panier neutre passe de 71 socles (63 vrais + 8 faux) a 63,
// et le compte des VRAIS socles neutres ne bouge pas : la correction ne retire rien de
// legitime, elle ne retire que ce qui n'aurait jamais du entrer.

// dbb2SoclesNeutresAuLabel : le nombre de socles `flag_spawn` du catalogue versionne qui
// portent le label `ctf_neutral_include`. C'est le panier neutre APRES correction, et le
// meme compte qu'AVANT pour les socles reellement neutres (recense le 2026-09-13).
const dbb2SoclesNeutresAuLabel = 63

// dbb2SoclesSansEquipeSansLabel : les socles a `team_index = -1` SANS label neutre — les
// huit sortis du panier. Cliffside, Highpower Heavies, Solitude, Solitude - Ranked, plus
// quatre entrees d'un meme map_id sans `public_name` (`1042b738`).
const dbb2SoclesSansEquipeSansLabel = 8

// TestCatalogueRecensementDesSoclesNeutres — LE RECENSEMENT, sur le catalogue VERSIONNE.
//
// Il echouera si le catalogue est regenere avec d'autres cartes : c'est voulu. Le compte est
// la mesure qui fonde la correction ; s'il bouge, la correction se re-instruit plutot qu'elle
// ne se suppose.
func TestCatalogueRecensementDesSoclesNeutres(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot: %v", err)
	}
	cat, err := LoadMapObjectives(filepath.Join(root,
		"data", "titles", "halo_infinite", "reference", "map_objectives.json"))
	if err != nil {
		t.Fatalf("catalogue versionne d'objectifs: %v", err)
	}
	var auLabel, sansEquipeSansLabel int
	for _, e := range cat.Maps {
		for _, p := range e.PointsOfRole(mapvar.RoleFlagSpawn) {
			switch {
			case p.Neutral:
				auLabel++
			case p.TeamIndex == TeamNeutral:
				sansEquipeSansLabel++
			}
		}
	}
	if auLabel != dbb2SoclesNeutresAuLabel {
		t.Errorf("%d socles neutres au label, attendu %d", auLabel, dbb2SoclesNeutresAuLabel)
	}
	if sansEquipeSansLabel != dbb2SoclesSansEquipeSansLabel {
		t.Errorf("%d socles a team_index = -1 sans label neutre, attendu %d",
			sansEquipeSansLabel, dbb2SoclesSansEquipeSansLabel)
	}
	// LE PANIER D'AVANT, dit explicitement : c'est ce que le tri sur l'equipe ramassait.
	if avant := auLabel + sansEquipeSansLabel; avant != dbb2SoclesNeutresAuLabel+dbb2SoclesSansEquipeSansLabel {
		t.Errorf("panier neutre trie sur l'equipe = %d socles", avant)
	}
}
