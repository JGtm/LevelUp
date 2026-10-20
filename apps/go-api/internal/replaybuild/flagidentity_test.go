package replaybuild

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// flagidentity_test.go — LA TABLE DES EQUIPES DE LA FEUILLE N'EST QU'UN CONTROLE.
//
// Les tests du pont d'identite du drapeau (`withFlagIdentity`, `pontParManche`) ont DEMENAGE le
// 2026-09-28 avec le code qu'ils testent : `film/replay/porteurs_drapeau_identite_test.go` (lot V1.4
// du plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`).

// TestScoreboardTeamsEstUnControle — LA FEUILLE DE MATCH NE POSE PLUS L EQUIPE DU PORTEUR (lot 1.7).
//
// L'invariant « un portage n'est JAMAIS pose sur le drapeau de l'equipe de son porteur » tirait
// son equipe de `FlagInput.TeamOf`, une table fournie par CE paquet depuis les lignes de match.
// Depuis le lot 1.7 (decision utilisateur V4), l equipe vient du FILM et cette table est
// un CONTROLE : elle alimente `coverage.teams.{accord, contradiction, silence}` par
// `Options.ScoreboardTeams`, et ne tranche que la contradiction d'un bot entre son entite et sa
// declaration (`replay/occupants_equipe_arbitree.go`). Ce test garde ce qu'elle doit contenir — une
// equipe INCONNUE (-1 en base) n'entre pas, sans quoi le controle compterait une contradiction
// contre une absence.
func TestScoreboardTeamsEstUnControle(t *testing.T) {
	facts := port.MatchFacts{Players: []domain.MatchPlayerFact{
		{XUID: "aaa", TeamID: 0},
		{XUID: "bbb", TeamID: 1},
		{XUID: "ccc", TeamID: -1}, // equipe absente de la base
	}}
	got := teamByXUID(facts)
	if len(got) != 2 || got["aaa"] != 0 || got["bbb"] != 1 {
		t.Fatalf("table de controle %v, attendu {aaa:0, bbb:1} — une equipe inconnue n'entre pas", got)
	}
	if _, ok := got["ccc"]; ok {
		t.Errorf("l'equipe -1 de « ccc » ne doit pas entrer : le controle compterait une " +
			"contradiction contre une absence")
	}
	if teamByXUID(port.MatchFacts{}) != nil {
		t.Errorf("sans lignes de match la table est nil : le controle se tait, et le document " +
			"est le meme a l'octet pres")
	}
}
