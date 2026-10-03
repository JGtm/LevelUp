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
// par rang : la liste publiée ne porte que les matchs qui ont quelque chose à
// dire (film ou feuille d'objectif), sa longueur n'est pas celle de la portée.
func matchOf(b domain.SquadFormesBlock, id string) *domain.SquadFormesMatch {
	for i := range b.Matches {
		if b.Matches[i].MatchID == id {
			return &b.Matches[i]
		}
	}
	return nil
}

// registryWeapon — une arme telle que le service la résout (nom du catalogue du
// titre, dimensions du registre canonique).
func registryWeapon(label, key, class, role string) WeaponInfo {
	return WeaponInfo{Label: label, WeaponKey: key, Class: class, Role: role}
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
		Films: map[string]sessionusage.FilmRow{"m2": {MatchID: "m2", DurationMS: 564000}},
		Pads: map[string]FilmPads{"m2": {MatchID: "m2", PadNamed: 7, PadUnnamed: 3,
			WeaponPads: []WeaponPad{{Weapon: "71ab0a2c", Occupations: 4, Named: 2}}}},
		WallFamilyKey: "wall",
		Gamertags:     map[string]string{"moi": "JGtm", "cop": "Madina97294", "adv": "Bob5499"},
		Weapons: map[string]WeaponInfo{
			"71ab0a2c": registryWeapon("SPNKr", "hinf_m41", "heavy", "power"),
			"0a1992bc": registryWeapon("S7", "hinf_s7", "heavy", "sniper"),
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
	// La PORTÉE fait deux matchs, la liste PUBLIÉE n'en porte qu'un : le match
	// sans film ni objectif n'alimente aucune carte.
	if len(got.Matches) != 1 || got.Matches[0].MatchID != "m2" {
		t.Fatalf("seul le match mesuré est publié, obtenu %+v", got.Matches)
	}
}

// Un match sans film n'a AUCUNE ligne de lobby : les formes doivent pouvoir le
// rendre en « non mesuré », jamais en zéros.
func TestBuild_MatchNonMesureSansLobby(t *testing.T) {
	got := Build(fixture())
	if matchOf(got, "m1") != nil {
		t.Fatal("un match sans film NI objectif n'a rien à publier")
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
	if m1.Measured || len(m1.Lobby) != 0 {
		t.Fatalf("m1 devait rester non mesuré et sans lobby, obtenu %+v", m1)
	}
	if m1.ModeLabel != "Bastion" || m1.MapLabel != "Perilous" {
		t.Fatalf("l'identité d'affichage doit survivre à l'absence de film, obtenu %+v", m1)
	}
	if m1.TeamSize != 4 || m1.LobbySize != 8 {
		t.Fatalf("les effectifs du match restent publiés, obtenu %+v", m1)
	}
}

func TestBuild_LobbyDesDeuxCamps(t *testing.T) {
	got := Build(fixture())
	m2 := *matchOf(got, "m2")
	if len(m2.Lobby) != 3 {
		t.Fatalf("les deux camps attendus (3 joueurs), obtenu %d", len(m2.Lobby))
	}
	var me *domain.SquadFormesLobbyPlayer
	for i := range m2.Lobby {
		if m2.Lobby[i].XUID == "moi" {
			me = &m2.Lobby[i]
		}
	}
	if me == nil {
		t.Fatal("le joueur de la page doit avoir sa ligne")
	}
	if me.Camo != 2 || me.Grapple != 1 || me.Wall != 3 || me.PadPickups != 2 {
		t.Fatalf("gestes attendus camo=2 grappin=1 mur=3 socles=2, obtenu %+v", me)
	}
	if me.Gamertag != "JGtm" || me.TeamID == nil || *me.TeamID != 0 {
		t.Fatalf("identité attendue JGtm camp 0, obtenu %+v", me)
	}
	if m2.DurationSeconds != 564 {
		t.Fatalf("durée mesurée attendue 564 s, obtenu %v", m2.DurationSeconds)
	}
	if m2.PadNamed != 7 || m2.PadUnnamed != 3 || len(m2.WeaponPads) != 1 {
		t.Fatalf("grain match des socles attendu 7/3 et 1 socle, obtenu %+v", m2)
	}
}

// Seul le MUR est un geste d'équipement ici : les autres familles déployées
// (capteur...) ne sont pas des axes de l'artefact et ne doivent pas fuir dans
// le compte des murs.
func TestBuild_MurSeulementLaFamilleMur(t *testing.T) {
	got := Build(fixture())
	for _, p := range matchOf(got, "m2").Lobby {
		if p.XUID == "moi" && p.Wall != 3 {
			t.Fatalf("mur attendu 3 (jamais 12 avec le capteur), obtenu %d", p.Wall)
		}
	}
}

func TestBuild_ArmesNommeesEtRangees(t *testing.T) {
	got := Build(fixture())
	if len(got.Weapons) != 2 {
		t.Fatalf("2 armes rencontrées attendues, obtenu %+v", got.Weapons)
	}
	// Tri par clé : contrat stable, jamais l'ordre d'une map.
	if got.Weapons[0].Key != "0a1992bc" || got.Weapons[1].Key != "71ab0a2c" {
		t.Fatalf("tri par clé attendu, obtenu %+v", got.Weapons)
	}
	for _, w := range got.Weapons {
		if w.Class != domain.SquadFormesWeaponHeavy {
			t.Fatalf("les deux armes du registre sont lourdes, obtenu %+v", w)
		}
	}
}

func TestWeaponClassOf(t *testing.T) {
	cases := []struct {
		name string
		in   WeaponInfo
		want string
	}{
		{"lourde par la classe", WeaponInfo{Class: "heavy", Role: "power"}, domain.SquadFormesWeaponHeavy},
		{"fusil de précision: lourde", WeaponInfo{Class: "heavy", Role: "sniper"}, domain.SquadFormesWeaponHeavy},
		{"précision hors classe lourde", WeaponInfo{Class: "shoulder", Role: "precision"}, domain.SquadFormesWeaponPrecision},
		{"automatique: autre", WeaponInfo{Class: "shoulder", Role: "automatic"}, domain.SquadFormesWeaponOther},
		{"hors registre: autre", WeaponInfo{}, domain.SquadFormesWeaponOther},
	}
	for _, c := range cases {
		if got := WeaponClassOf(c.in); got != c.want {
			t.Errorf("%s: attendu %s, obtenu %s", c.name, c.want, got)
		}
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

// Un match sans objectif n'a pas de bloc : les six matchs Assassin d'une soirée
// ne sont pas un trou de mesure.
func TestBuild_ModeSansObjectifSansBloc(t *testing.T) {
	got := Build(fixture())
	for _, m := range got.Matches {
		if m.Objective != nil {
			t.Fatalf("aucun objectif dans la fixture, obtenu %+v sur %s", m.Objective, m.MatchID)
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

// LA NON-RÉGRESSION DE L'ALLÈGEMENT (2026-09-13). Mille matchs de plus qui ne
// portent NI film NI objectif ne doivent RIEN changer au bloc publié — ni une
// ligne de lobby, ni une arme, ni un socle, ni une colonne d'objectif — et tout
// changer aux seuls compteurs de portée. C'est exactement la propriété qui rend
// l'allègement invisible à l'écran : les cartes lisent le contenu, les pieds de
// forme lisent les compteurs.
func TestBuild_MatchsVidesNeChangentQueLesCompteurs(t *testing.T) {
	base := Build(fixture())

	in := fixture()
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
	if !reflect.DeepEqual(got.Weapons, base.Weapons) {
		t.Fatalf("les armes ont changé : %+v vs %+v", base.Weapons, got.Weapons)
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
	if m.Measured || len(m.Lobby) != 0 {
		t.Fatalf("il n'a ni film ni lobby, obtenu %+v", m)
	}
	if got.MatchesMeasured != 1 || got.MatchesTotal != 3 {
		t.Fatalf("compteurs attendus 1/3, obtenu %d/%d", got.MatchesMeasured, got.MatchesTotal)
	}
}

// La VENTILATION DES LÂCHERS descend telle quelle jusqu'au bloc (D9, lot G du
// 2026-09-21). Elle existait depuis le décodeur et s'arrêtait ici, sur le seul
// scalaire `Dropped` : une colonne d'écran qui mélange un mur, un capteur et un
// grappin lâchés à la mort ne se compare à rien.
func TestBuild_LachersVentilesParFamille(t *testing.T) {
	m := matchOf(Build(fixture()), "m2")
	if m == nil {
		t.Fatal("le match mesuré doit être publié")
	}
	var cop *domain.SquadFormesLobbyPlayer
	for i := range m.Lobby {
		if m.Lobby[i].XUID == "cop" {
			cop = &m.Lobby[i]
		}
	}
	if cop == nil {
		t.Fatal("la ligne de lobby de cop doit exister")
	}
	// Le TOTAL reste publié : aucun lecteur de `dropped` n'est cassé.
	if cop.Dropped != 4 {
		t.Fatalf("total des lâchers attendu 4, obtenu %d", cop.Dropped)
	}
	if got := cop.DroppedByFamily["wall"]; got != 3 {
		t.Fatalf("lâchers de mur attendus 3, obtenu %d", got)
	}
	if got := cop.DroppedByFamily["sensor"]; got != 1 {
		t.Fatalf("lâchers de capteur attendus 1, obtenu %d", got)
	}
	// ABSENTE PLUTÔT QUE NULLE : un joueur sans ventilation (ligne écrite avant
	// la colonne) ne porte pas une carte de zéros.
	for i := range m.Lobby {
		if m.Lobby[i].XUID == "moi" && m.Lobby[i].DroppedByFamily != nil {
			t.Fatalf("sans lâcher mesuré, la ventilation doit être absente, obtenu %+v",
				m.Lobby[i].DroppedByFamily)
		}
	}
}
