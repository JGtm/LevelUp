package squadformes

import (
	"fmt"
	"reflect"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

func team(v int) *int { return &v }

// matchOf — le match publié sous cet identifiant. Les tests ne raisonnent JAMAIS
// par rang : la liste publiée ne porte que les matchs à objectif, sa longueur n'est pas celle de
// la portée.
func matchOf(b domain.SquadFormesBlock, id string) *domain.SquadFormesMatch {
	for i := range b.Matches {
		if b.Matches[i].MatchID == id {
			return &b.Matches[i]
		}
	}
	return nil
}

// fixture — une soirée miniature : deux matchs, le premier sans film décodé.
// C'est la forme exacte que l'artefact 2ec1b8eb met en scène (un match non
// mesuré au milieu d'un scope mesuré).
func fixture() Input {
	return Input{
		PlayerXUID: "moi",
		SquadPlayers: []domain.SessionUsageSquadPlayer{
			{XUID: "moi", Gamertag: "JGtm"}, {XUID: "cop", Gamertag: "Madina97294"},
		},
		Metas: []MatchMeta{
			{MatchID: "m1", StartTime: "2026-07-31T19:22:00Z", ModeLabel: "Bastion", MapLabel: "Perilous"},
			{MatchID: "m2", StartTime: "2026-07-31T19:32:00Z", ModeLabel: "Assassin", MapLabel: "Curfew"},
		},
		Matches: []sessionusage.MatchInput{
			{MatchID: "m1", Measured: false, PlayerTeam: team(0), TeamSize: 4, LobbySize: 8},
			{MatchID: "m2", Measured: true, PlayerTeam: team(0), TeamSize: 4, LobbySize: 8,
				TeamOf: map[string]int{"moi": 0, "cop": 0, "adv": 1},
				Players: []sessionusage.PlayerRow{
					{MatchID: "m2", XUID: "moi", CamoEpisodes: 2, GrapplePulls: 1, PadPickups: 2,
						DeployedByFamily:   map[string]int{"wall": 3, "sensor": 9},
						PadPickupsByFamily: map[string]int{"71ab0a2c": 2}},
					{MatchID: "m2", XUID: "cop", OvershieldEpisodes: 1, DroppedObjects: 4,
						DroppedByFamily: map[string]int{"wall": 3, "sensor": 1}},
					{MatchID: "m2", XUID: "adv", PadPickups: 5,
						PadPickupsByFamily: map[string]int{"0a1992bc": 5}},
				}},
		},
	}

}

func TestBuild_ScopeEtCouverture(t *testing.T) {
	got := Build(fixture())
	if !got.Available || got.MatchesTotal != 2 || got.MatchesMeasured != 1 {
		t.Fatalf("couverture attendue 1/2 disponible, obtenu %+v", got)
	}
	if got.MainXUID != "moi" || len(got.Squad) != 2 {
		t.Fatalf("escouade attendue moi + 1, obtenu %+v", got.Squad)
	}
	// La PORTÉE fait deux matchs, la liste PUBLIÉE aucun : sans feuille d'objectif, un match
	// n'alimente aucune carte d'objectif, même filmé.
	if len(got.Matches) != 0 {
		t.Fatalf("aucun match à objectif dans la fixture, obtenu %+v", got.Matches)
	}
}

// Un match sans objectif n'est pas publié, mais reste COMPTÉ ; un match sans film AVEC objectif
// est publié, avec son identité d'affichage et son camp.
func TestBuild_MatchSansFilmAvecObjectif(t *testing.T) {
	got := Build(fixture())
	if matchOf(got, "m1") != nil {
		t.Fatal("un match sans objectif n'a rien à publier")
	}
	// Il reste COMPTÉ : c'est de ces deux nombres que l'écran tire « N matchs
	// sans film décodé sont hors de cette forme ».
	if got.MatchesTotal-got.MatchesMeasured != 1 {
		t.Fatalf("le match sans film doit rester compté, obtenu %d/%d",
			got.MatchesMeasured, got.MatchesTotal)
	}

	// Un match SANS FILM mais AVEC objectif, lui, est publié — sans lobby.
	in := fixture()
	in.Objectives = []ObjectiveColumnRow{{MatchID: "m1", XUID: "moi",
		Family: narrative.FamilyZonesStrongholds, Values: map[string]float64{"zone_secures": 3}}}
	withObj := Build(in)
	m1 := matchOf(withObj, "m1")
	if m1 == nil {
		t.Fatal("un match à objectif se publie même sans film")
	}
	if m1.ModeLabel != "Bastion" || m1.MapLabel != "Perilous" {
		t.Fatalf("l'identité d'affichage doit survivre à l'absence de film, obtenu %+v", m1)
	}
	if m1.PlayerTeam == nil || *m1.PlayerTeam != 0 {
		t.Fatalf("le camp du joueur reste publié sans film, obtenu %+v", m1.PlayerTeam)
	}
	if withObj.MatchesMeasured != 1 {
		t.Fatalf("le compte des matchs filmés ne dépend pas de l'objectif, obtenu %d", withObj.MatchesMeasured)
	}
}

// LES COLONNES SONT PILOTÉES PAR LA DONNÉE : une colonne que personne n'alimente
// sur le scope n'existe pas, et une colonne alimentée par un seul joueur existe
// pour tout le monde.
func TestBuild_ColonnesObjectifPiloteesParLaDonnee(t *testing.T) {
	in := fixture()
	in.Objectives = []ObjectiveColumnRow{
		{MatchID: "m2", XUID: "moi", Family: narrative.FamilyCTF,
			Values: map[string]float64{"flag_returns": 2, "flag_captures": 0, "time_as_flag_carrier_seconds": 0}},
		{MatchID: "m2", XUID: "adv", Family: narrative.FamilyCTF,
			Values: map[string]float64{"flag_returns": 0, "flag_captures": 0, "time_as_flag_carrier_seconds": 17.2}},
	}
	got := Build(in)
	obj := matchOf(got, "m2").Objective
	if obj == nil {
		t.Fatal("le match porte un objectif, le bloc doit exister")
	}
	if obj.Family != string(narrative.FamilyCTF) {
		t.Fatalf("famille attendue ctf, obtenu %s", obj.Family)
	}
	keys := map[string]domain.SquadFormesObjectiveColumn{}
	for _, c := range obj.Columns {
		keys[c.Key] = c
	}
	if _, ok := keys["flag_captures"]; ok {
		t.Fatalf("une colonne jamais alimentée ne doit pas exister, obtenu %+v", obj.Columns)
	}
	if c, ok := keys["flag_returns"]; !ok || c.Role != string(narrative.ObjectiveRoleDefend) || c.Duration {
		t.Fatalf("flag_returns attendue en rôle défendre, obtenu %+v", obj.Columns)
	}
	if c, ok := keys["time_as_flag_carrier_seconds"]; !ok || !c.Duration {
		t.Fatalf("le temps de portage est une durée, obtenu %+v", obj.Columns)
	}
	if len(obj.Players) != 2 {
		t.Fatalf("les deux camps attendus, obtenu %+v", obj.Players)
	}
	// LE CAMP EST RECOLLÉ DEPUIS LES PARTICIPANTS : sans lui, tout l'objectif se
	// lit comme adverse et le rapport de force affiche « 0 % » partout (défaut
	// mesuré sur données réelles le 2026-09-13).
	byXUID := map[string]*domain.SquadFormesObjectivePlayer{}
	for i := range obj.Players {
		byXUID[obj.Players[i].XUID] = &obj.Players[i]
	}
	if p := byXUID["moi"]; p == nil || p.TeamID == nil || *p.TeamID != 0 {
		t.Fatalf("le joueur de la page devait être rangé dans son camp, obtenu %+v", byXUID["moi"])
	}
	if p := byXUID["adv"]; p == nil || p.TeamID == nil || *p.TeamID != 1 {
		t.Fatalf("l'adversaire devait être rangé dans le sien, obtenu %+v", byXUID["adv"])
	}
	// L'ordre des colonnes suit les rôles (prendre, défendre, tenir), jamais une map.
	if obj.Columns[len(obj.Columns)-1].Role != string(narrative.ObjectiveRoleHold) {
		t.Fatalf("la durée ferme la marche, obtenu %+v", obj.Columns)
	}
}

// Un match sans objectif n'est pas publié : les six matchs Assassin d'une soirée ne sont pas un
// trou de mesure, et aucun match publié n'a de feuille vide.
func TestBuild_ModeSansObjectifNonPublie(t *testing.T) {
	got := Build(fixtureObjectif())
	for _, m := range got.Matches {
		if m.Objective == nil {
			t.Fatalf("match publié sans objectif : %s", m.MatchID)
		}
	}
}

func TestBuild_ScopeVide(t *testing.T) {
	got := Build(Input{PlayerXUID: "moi"})
	if !got.Available || got.MatchesTotal != 0 || len(got.Matches) != 0 {
		t.Fatalf("scope vide: bloc disponible à 0 match, obtenu %+v", got)
	}
}

// Une ligne d'objectif dont le camp est INCONNU des participants reste publiée
// SANS camp : elle compte dans le lobby et dans aucun des deux côtés. Inventer
// un camp ferait un rapport de force faux ; l'effacer perdrait le dénominateur.
func TestBuild_ObjectifCampInconnuResteSansCamp(t *testing.T) {
	in := fixture()
	in.Objectives = []ObjectiveColumnRow{
		{MatchID: "m2", XUID: "moi", Family: narrative.FamilyCTF,
			Values: map[string]float64{"flag_returns": 2}},
		{MatchID: "m2", XUID: "fantome", Family: narrative.FamilyCTF,
			Values: map[string]float64{"flag_returns": 5}},
	}
	got := Build(in)
	obj := matchOf(got, "m2").Objective
	if obj == nil {
		t.Fatal("le bloc objectif doit exister")
	}
	for _, p := range obj.Players {
		if p.XUID == "fantome" && p.TeamID != nil {
			t.Fatalf("un xuid absent des participants ne reçoit pas de camp, obtenu %+v", p)
		}
	}
}

// Mille matchs de plus qui ne portent NI film NI objectif ne doivent RIEN changer au bloc
// publié et tout changer aux seuls compteurs de portée : les cartes lisent le contenu, les pieds
// lisent les compteurs.
func TestBuild_MatchsVidesNeChangentQueLesCompteurs(t *testing.T) {
	base := Build(fixtureObjectif())

	in := fixtureObjectif()
	for i := 0; i < 1000; i++ {
		in.Metas = append(in.Metas, MatchMeta{
			MatchID:   fmt.Sprintf("vide-%03d", i),
			StartTime: "2025-01-01T00:00:00Z",
		})
	}
	got := Build(in)

	if got.MatchesTotal != base.MatchesTotal+1000 {
		t.Fatalf("la portée doit compter les mille matchs, obtenu %d", got.MatchesTotal)
	}
	if got.MatchesMeasured != base.MatchesMeasured {
		t.Fatalf("aucun d'eux n'est mesuré, obtenu %d", got.MatchesMeasured)
	}
	if !reflect.DeepEqual(got.Matches, base.Matches) {
		t.Fatalf("le contenu publié a changé :\n avant %+v\n après %+v", base.Matches, got.Matches)
	}

	if !reflect.DeepEqual(got.Squad, base.Squad) {
		t.Fatalf("l'escouade a changé : %+v vs %+v", base.Squad, got.Squad)
	}
}

// Le même invariant avec des matchs à OBJECTIF SEUL : eux se publient, et le
// reste du bloc ne bouge pas pour autant.
func TestBuild_MatchsAObjectifSeulSontPublies(t *testing.T) {
	in := fixture()
	in.Metas = append(in.Metas, MatchMeta{MatchID: "obj-seul", StartTime: "2026-07-31T18:10:00Z"})
	in.Objectives = []ObjectiveColumnRow{{MatchID: "obj-seul", XUID: "moi",
		Family: narrative.FamilyCTF, Values: map[string]float64{"flag_returns": 4}}}
	got := Build(in)

	m := matchOf(got, "obj-seul")
	if m == nil || m.Objective == nil {
		t.Fatalf("un match à objectif seul se publie, obtenu %+v", got.Matches)
	}

	if got.MatchesMeasured != 1 || got.MatchesTotal != 3 {
		t.Fatalf("compteurs attendus 1/3, obtenu %d/%d", got.MatchesMeasured, got.MatchesTotal)
	}
}
