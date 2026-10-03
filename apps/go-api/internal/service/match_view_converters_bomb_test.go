package service

// match_view_converters_bomb_test.go — LE BLOC D'ASSAUT DU TABLEAU DES SCORES.
//
// CE QU'IL FERME. `buildScoreboardObjective` ne recopie QUE le bloc du mode joué, et le bloc
// d'Assaut est le SEUL qui ne vienne pas de `match_objective_stats_latest` : il est chargé par
// une seconde requête, gatée par une capability. Deux régressions sont possibles et muettes —
// (a) `HasObjective()` qui ne compterait pas le bloc bombe, et un match d'Assaut (qui n'a AUCUN
// bloc API, l'API 343 n'en publiant aucun pour ce mode) rendrait `nil` : la section entière
// disparaîtrait côté web sans qu'aucun test ne rougisse ; (b) un champ non recopié, qui
// s'afficherait « non mesuré » alors que la base le porte.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func mesure(v int) *int            { return &v }
func mesureSec(v float64) *float64 { return &v }

// TestBuildScoreboardObjective_BlocAssautSeul : un match d'Assaut n'a AUCUN bloc API. Le bloc
// doit exister quand même, et porter les cinq clés — dont celle qui n'est pas mesurée, à `nil`.
func TestBuildScoreboardObjective_BlocAssautSeul(t *testing.T) {
	raw := domain.ObjectiveRaw{
		BombDetonations:          mesure(2),
		BombArms:                 mesure(1),
		BombGrabs:                mesure(3),
		TimeAsBombCarrierSeconds: mesureSec(41.5),
		// BombCarriersKilled reste nil : la source n'est pas lue (report au registre).
	}
	out := buildScoreboardObjective(raw)
	if out == nil {
		t.Fatal("bloc nil pour un match d'Assaut — HasObjective() ne compte pas le bloc bombe, " +
			"et la section « Objectifs » disparaîtrait entièrement côté web")
	}
	if out.BombDetonations == nil || *out.BombDetonations != 2 ||
		out.BombArms == nil || *out.BombArms != 1 ||
		out.BombGrabs == nil || *out.BombGrabs != 3 ||
		out.TimeAsBombCarrierSeconds == nil || *out.TimeAsBombCarrierSeconds != 41.5 {
		t.Errorf("les quatre mesures ne sont pas recopiées : %+v", out)
	}
	// ABSENT N'EST PAS ZÉRO : le convertisseur ne comble rien.
	if out.BombCarriersKilled != nil {
		t.Errorf("bomb_carriers_killed = %d, attendu nil (source non lue)", *out.BombCarriersKilled)
	}
}

// TestBuildScoreboardObjective_SansAssautAucuneCleBombe : hors Assaut — ou sur un titre qui ne
// déclare pas `film.bomb_stats`, donc dont la seconde requête n'a jamais été payée — aucune des
// cinq clés n'est renseignée. Un zéro y serait un mensonge : le mode n'a pas de bombe.
func TestBuildScoreboardObjective_SansAssautAucuneCleBombe(t *testing.T) {
	out := buildScoreboardObjective(domain.ObjectiveRaw{
		FlagGrabs:    mesure(4),
		FlagCaptures: mesure(1),
	})
	if out == nil {
		t.Fatal("bloc nil pour un match CTF")
	}
	if out.BombDetonations != nil || out.BombArms != nil || out.BombGrabs != nil ||
		out.TimeAsBombCarrierSeconds != nil || out.BombCarriersKilled != nil {
		t.Errorf("un match CTF porte des clés d'Assaut : %+v", out)
	}
}

// TestObjectiveRaw_HasBombNeCompteQueLesSourcesLues : le discriminant. Une ligne totalement
// vide n'est pas un bloc d'Assaut — sans quoi TOUT match rendrait un bloc, et la section
// s'afficherait vide partout.
func TestObjectiveRaw_HasBombNeCompteQueLesSourcesLues(t *testing.T) {
	if (domain.ObjectiveRaw{}).HasBomb() {
		t.Error("une ObjectiveRaw vide se déclare bloc d'Assaut")
	}
	if !(domain.ObjectiveRaw{BombGrabs: mesure(0)}).HasBomb() {
		t.Error("un ramassage MESURÉ à zéro doit compter : c'est une mesure, pas une absence")
	}
	// `bomb_carriers_killed` seul ne discrimine pas — il est nul partout aujourd'hui, et le
	// jour où il ne le sera plus, les trois autres seront lues avec lui.
	if (domain.ObjectiveRaw{BombCarriersKilled: mesure(2)}).HasBomb() {
		t.Error("bomb_carriers_killed seul ne doit pas discriminer le bloc")
	}
}
