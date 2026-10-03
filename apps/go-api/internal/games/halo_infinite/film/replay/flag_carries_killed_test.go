package replay

// flag_carries_killed_test.go — `flag_carriers_killed` NE FERME QUE LE PORTAGE D UN ADVERSAIRE DU
// TUEUR (constat RB1-1 de l audit du 2026-09-24, lot J9.1).
//
// LE DEFAUT : l evenement est credite au TUEUR et ne nomme pas la victime. La regle fermait
// l UNIQUE portage ouvert a cet instant en n excluant que le portage du tueur lui-meme — un
// coequipier du tueur, ou n importe qui quand le tueur n etait pas nomme, se voyait fermer son
// portage et effacer sa capture. Le drapeau etait alors publie AU SOL pendant qu il courait avec.
//
// LA REGLE : un porteur n est candidat que s il est d une equipe ADVERSE a celle du tueur ; un
// tueur non nomme, ou dont l equipe n est pas lue, ne ferme rien et se compte
// (`coverage.flagCarries.unjudgedCarrierKills`).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// flagKilledScan : « x » (equipe 0, slot 12) vole le drapeau de l equipe 1 a 1 000 ms et le
// CAPTURE a 5 000 ms ; un `flag_carriers_killed` credite au slot `tueur` tombe a 3 000 ms, au
// milieu du portage.
func flagKilledScan(tueur int, identity map[int]string, teams map[string]int) FlagCarryScan {
	return FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectives.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectives.StatFlagSteals},
			{TimeMS: 3000, Slot: tueur, Stat: objectives.StatFlagCarriersKilled},
			{TimeMS: 5000, Slot: 12, Stat: objectives.StatFlagCaptures},
		},
		Identity: objectives.FlatRoundIdentity(identity),
		Spawns:   flagInvariantSpawns(),
		TeamOf:   teams,
	}
}

// TestPorteurTueNeFermeQueLePortageDUnAdversaire — LE POINT DU LOT. Dans les trois cas le seul
// portage ouvert a 3 000 ms est celui de « x », et rien ne dit qu il est tombe : sa capture doit
// survivre, et le drapeau rentrer chez lui a la capture sans passer par le sol.
//
// MUTATION : retirer le filtre d equipe du candidat (flag_carries_killed.go) rougit le cas
// « coequipier » ; retirer l abstention sur tueur non nomme rougit le cas « tueur non nomme ».
func TestPorteurTueNeFermeQueLePortageDUnAdversaire(t *testing.T) {
	for _, cas := range []struct {
		nom      string
		tueur    int
		identity map[int]string
		teams    map[string]int
	}{
		{"coequipier du porteur", 16, map[int]string{12: "x", 16: "k"}, map[string]int{"x": 0, "k": 0}},
		{"tueur non nomme", 18, map[int]string{12: "x", 16: "k"}, map[string]int{"x": 0, "k": 1}},
		{"equipe du tueur non lue", 16, map[int]string{12: "x", 16: "k"}, map[string]int{"x": 0}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			tracks := []Track{flagTestTrack(10, "x", 0, 99, 2, 2)}
			got, cov := buildFlagCarries(flagKilledScan(cas.tueur, cas.identity, cas.teams),
				flagTestCtx(tracks, nil, 100))
			if cov == nil || cov.Carries != 1 {
				t.Fatalf("couverture %+v : un portage publie attendu", cov)
			}
			assertFlagStates(t, flagOfTeam(t, got, 1),
				[]string{FlagStateHome, FlagStateCarried, FlagStateHome})
			// Le coequipier n est pas un doute : la regle a JUGE (aucun adversaire ouvert). Les deux
			// autres cas n ont pas pu l etre, et le disent.
			if veut := map[bool]int{true: 0, false: 1}[cas.nom == "coequipier du porteur"]; cov.UnjudgedCarrierKills != veut {
				t.Errorf("%d evenements non juges, attendu %d", cov.UnjudgedCarrierKills, veut)
			}
		})
	}
}

// TestVerdictsDuPorteurTue — LA TABLE DES VERDICTS, sur [closeByCarrierKills] directement. Tous les
// portages courent de 1 000 a 9 000 ms ; l evenement tombe a 3 000 ms, credite au slot 30.
func TestVerdictsDuPorteurTue(t *testing.T) {
	for _, cas := range []struct {
		nom      string
		porteurs []string // xuids des portages ouverts
		teams    map[string]int
		tueur    string // "" : slot 30 non nomme
		ferme    int    // index du portage ferme, -1 aucun
		ambigus  int
		nonJuges int
	}{
		{"un adversaire", []string{"a"}, map[string]int{"a": 1, "k": 0}, "k", 0, 0, 0},
		{"un coequipier", []string{"a"}, map[string]int{"a": 0, "k": 0}, "k", -1, 0, 0},
		{"coequipier et adversaire", []string{"a", "b"}, map[string]int{"a": 0, "b": 1, "k": 0}, "k", 1, 0, 0},
		{"deux adversaires", []string{"a", "b"}, map[string]int{"a": 1, "b": 1, "k": 0}, "k", -1, 1, 0},
		{"adversaire et porteur d equipe non lue", []string{"a", "b"}, map[string]int{"a": 1, "k": 0}, "k", -1, 1, 0},
		{"seul un porteur d equipe non lue", []string{"a"}, map[string]int{"k": 0}, "k", -1, 0, 1},
		{"porteur a aucune equipe", []string{"a"}, map[string]int{"a": TeamNeutral, "k": 0}, "k", -1, 0, 1},
		{"tueur non nomme", []string{"a"}, map[string]int{"a": 1}, "", -1, 0, 1},
		{"equipe du tueur non lue", []string{"a"}, map[string]int{"a": 1}, "k", -1, 0, 1},
		{"le tueur est le seul porteur", []string{"k"}, map[string]int{"k": 0}, "k", -1, 0, 0},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			raws := make([]flagCarryRaw, 0, len(cas.porteurs))
			for _, x := range cas.porteurs {
				raws = append(raws, flagCarryRaw{xuid: x, t0: 1000, t1: 9000, closed: true,
					closedBy: flagCloserBound, flagIndex: -1})
			}
			identity := map[int]string{}
			if cas.tueur != "" {
				identity[30] = cas.tueur
			}
			scan := FlagCarryScan{
				Events: []objectives.NamedEvent{
					{TimeMS: 3000, Slot: 30, Stat: objectives.StatFlagCarriersKilled},
				},
				Identity: objectives.FlatRoundIdentity(identity), TeamOf: cas.teams,
			}
			ambigus, nonJuges := closeByCarrierKills(raws, scan)
			if ambigus != cas.ambigus || nonJuges != cas.nonJuges {
				t.Errorf("(%d ambigus, %d non juges), attendu (%d, %d)", ambigus, nonJuges,
					cas.ambigus, cas.nonJuges)
			}
			for i := range raws {
				veut := int64(9000)
				if i == cas.ferme {
					veut = 3000
				}
				if raws[i].t1 != veut {
					t.Errorf("portage %d (%s) borne a %d, attendu %d", i, raws[i].xuid, raws[i].t1, veut)
				}
			}
		})
	}
}
