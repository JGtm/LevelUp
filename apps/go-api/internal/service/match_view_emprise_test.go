package service

// match_view_emprise_test.go — L'EMPRISE SUR LA VUE MATCH : les joueurs de l'équipe dans l'ordre de la
// page (le joueur de la page, les profils suivis, le reste ; bots et partis sans fiche), un seul match,
// le fait « journal des morts publiable », l'« Isolement » de chaque joueur dans le même ordre, et
// chaque source qui dégrade seule (jamais une erreur de page).

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func vrai() *bool { v := true; return &v }

// tableauDeTest : l'équipe 0 de P (X non suivi avant Alpha suivi au tableau, un bot, un parti), et un
// adversaire. L'ordre est celui du tableau des scores.
func tableauDeTest() []domain.ScoreboardRaw {
	return []domain.ScoreboardRaw{
		{XUID: "E1", Gamertag: "Echo", TeamID: teamp(1)},
		{XUID: "X", Gamertag: "Xray", TeamID: teamp(0)},
		{XUID: "A", Gamertag: "Alpha", TeamID: teamp(0)},
		{XUID: "B", Gamertag: "Bot 7", TeamID: teamp(0), IsBot: true},
		{XUID: "L", Gamertag: "Lima", TeamID: teamp(0), LeftInProgress: vrai()},
		{XUID: "P", Gamertag: "Papa", TeamID: teamp(0)},
	}
}

func suivisDeTest() map[string]port.FriendMatchExtras {
	return map[string]port.FriendMatchExtras{"A": {}, "E1": {}}
}

func TestMatchCampPlayers_OrdreDeLaPage(t *testing.T) {
	got := matchCampPlayers(tableauDeTest(), "P", suivisDeTest())
	if fmt.Sprint(got) != "[{P Papa} {A Alpha} {X Xray}]" {
		t.Errorf("joueurs = %v, attendu Papa, Alpha (suivi), Xray ; ni bot, ni parti, ni adversaire", got)
	}
	if got := matchCampPlayers(tableauDeTest(), "absent", nil); got != nil {
		t.Errorf("joueur de la page absent : %v, attendu nil", got)
	}
	ffa := []domain.ScoreboardRaw{{XUID: "P", Gamertag: "Papa"}, {XUID: "Q", Gamertag: "Quebec"}}
	if got := matchCampPlayers(ffa, "P", nil); fmt.Sprint(got) != "[{P Papa}]" {
		t.Errorf("sans équipe : %v, attendu le joueur de la page seul", got)
	}
}

// donneesDeTest : le match m1 (film du résumé d'usage de test), le tableau ci-dessus, Q21d.
func donneesDeTest(publiables int) matchViewData {
	return matchViewData{scoreboard: tableauDeTest(), assistScope: domain.MatchAssistScopeRaw{MatchDeaths: 3, PublishableDeaths: publiables}}
}

func serviceMatchEmprise(usage *mockSessionUsageRepo, vies *viesCampFixes) *MatchViewService {
	svc := NewMatchViewService(nil, "P").WithEmpriseSheet(feuilleDeTest()).WithRadarRange(map[string]int{"Slayer:Arena": 18})
	if usage != nil {
		svc = svc.WithEmpriseUsageSummary(usage, "")
	}
	if vies != nil {
		svc = svc.WithCampLives(vies)
	}
	return svc
}

func TestMatchEmpriseFields_EquipeUnMatchEtJournal(t *testing.T) {
	vies := &viesCampFixes{parJoueur: map[string]domain.ViesLues{"A": viesDeA()}}
	f := serviceMatchEmprise(usageTestRepoMock(), vies).matchEmpriseFields(context.Background(), "m1", nil, donneesDeTest(3), suivisDeTest())
	e := f.Emprise
	if e == nil || e.MatchesTotal != 1 || len(e.Matches) != 1 || e.Matches[0].MatchID != "m1" || e.FilmUnavailable != "" {
		t.Fatalf("emprise = %+v, attendu le seul match m1, film lu", e)
	}
	if fmt.Sprint(e.Players) != "[{P Papa} {A Alpha} {X Xray}]" {
		t.Errorf("fiches = %v, attendu l'ordre de la page", e.Players)
	}
	if !e.KillJournalPublishable {
		t.Errorf("journal publiable attendu (3 morts publiables)")
	}
	l := f.LivesNearTeammate
	if l == nil || len(l.Players) != 3 || l.Players[0].XUID != "P" || l.Players[1].XUID != "A" || l.Players[2].XUID != "X" {
		t.Fatalf("vies = %+v, attendu P, A, X dans l'ordre des fiches", l)
	}
	if l.Players[1].Alone.Lives != 1 || l.Players[0].MatchesRead != 0 {
		t.Errorf("bilans = %+v, attendu A une vie seule, P sans vie lue (gardé, à zéro)", l.Players)
	}
	if fmt.Sprint(vies.xuids) != "[P A X]" || vies.appels != 1 {
		t.Errorf("lecture des vies : %d appel(s) pour %v, attendu une lecture de l'équipe", vies.appels, vies.xuids)
	}
}

func TestMatchEmpriseFields_JournalNonPubliable(t *testing.T) {
	f := serviceMatchEmprise(usageTestRepoMock(), nil).matchEmpriseFields(context.Background(), "m1", nil, donneesDeTest(0), nil)
	if f.Emprise == nil || f.Emprise.KillJournalPublishable {
		t.Errorf("emprise = %+v, attendu journal non publiable (0 mort publiable)", f.Emprise)
	}
	if f.LivesNearTeammate != nil {
		t.Errorf("vies sans lecteur câblé : %+v, attendu nil", f.LivesNearTeammate)
	}
}

func TestMatchEmpriseFields_Degradations(t *testing.T) {
	ctx := context.Background()
	// Halo 5 : aucun résumé d'usage câblé — film non supporté, la feuille seule.
	f := serviceMatchEmprise(nil, nil).matchEmpriseFields(ctx, "m1", nil, donneesDeTest(0), nil)
	if f.Emprise == nil || f.Emprise.FilmUnavailable != domain.EmpriseFilmUnsupported || f.Emprise.SheetUnavailable != "" {
		t.Errorf("sans film : %+v, attendu film_unsupported, feuille lue", f.Emprise)
	}
	// Lecture du film en échec, vies en échec : l'Emprise dit sa raison, les vies sont absentes.
	usage := usageTestRepoMock()
	usage.filmsErr = errors.New("panne")
	f = serviceMatchEmprise(usage, &viesCampFixes{err: errors.New("panne")}).matchEmpriseFields(ctx, "m1", nil, donneesDeTest(1), nil)
	if f.Emprise == nil || f.Emprise.FilmUnavailable != domain.EmpriseFilmLoadFailed || f.LivesNearTeammate != nil {
		t.Errorf("échecs : emprise %+v, vies %+v", f.Emprise, f.LivesNearTeammate)
	}
	// Joueur de la page absent du tableau : rien.
	d := donneesDeTest(1)
	d.scoreboard = d.scoreboard[:1]
	if f := serviceMatchEmprise(usageTestRepoMock(), nil).matchEmpriseFields(ctx, "m1", nil, d, nil); f.Emprise != nil || f.LivesNearTeammate != nil {
		t.Errorf("sans joueur de la page : %+v", f)
	}
}

// --- Outils de destruction ---

type categoriesFixes struct {
	rows []port.KillSourceCategoryRow
	err  error
}

func (c categoriesFixes) LoadKillSourceCategoryKills(context.Context, string, port.WeaponKillFilters) ([]port.KillSourceCategoryRow, error) {
	return c.rows, c.err
}

func moiAuTableau(kills, melee, grenade int) *domain.MatchScoreboardRow {
	return &domain.MatchScoreboardRow{XUID: "P", Gamertag: "Papa", IsMe: true, Kills: &kills, MeleeKills: &melee, GrenadeKills: &grenade}
}

func outilsParNom(tools *domain.SquadWeaponTools) map[string]int {
	out := map[string]int{}
	for _, l := range tools.Lines {
		name := l.Kind
		if l.Kind == domain.SquadToolKindWeapon {
			name = l.Label
		}
		out[name] = l.KillsByPlayer["Papa"]
	}
	return out
}

// Témoin du 22/09 (MESURES §3 B) : MK50 Sidekick 7, Mêlée 2, Grenade frag 1, VK78 Commando 1 ; les
// lignes d'un autre joueur sont ignorées.
func TestMatchWeaponTools_TemoinStarboard(t *testing.T) {
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "P", WeaponLabel: "MK50 Sidekick", Class: domain.FragClassSidearm, WeaponKey: "hinf_sidekick", Kills: 7, FromDamageSource: true},
		{XUID: "P", WeaponLabel: "VK78 Commando", Class: domain.FragClassShoulder, WeaponKey: "hinf_vk78", Kills: 1, FromDamageSource: true},
		{XUID: "P", WeaponLabel: "Grenade frag", Class: domain.FragClassGrenade, WeaponKey: "hinf_frag", Kills: 1, FromDamageSource: true},
		{XUID: "A", WeaponLabel: "MA40 AR", Class: domain.FragClassShoulder, WeaponKey: "hinf_ma40", Kills: 16, FromDamageSource: true},
	}
	tools := serviceMatchEmprise(nil, nil).matchWeaponTools(context.Background(), "m1", bulk, moiAuTableau(11, 2, 1))
	if tools == nil || fmt.Sprint(tools.Players) != "[Papa]" {
		t.Fatalf("outils = %+v, attendu un joueur", tools)
	}
	want := map[string]int{"MK50 Sidekick": 7, domain.SquadToolKindMelee: 2, "Grenade frag": 1, "VK78 Commando": 1}
	if got := outilsParNom(tools); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("outils = %v, attendu %v (ni MA40 d'un autre joueur, ni reliquat)", got, want)
	}
}

// Témoin du 24/07 (BTB, journal non publiable) : aucune ligne d'arme, dix frags « Non attribué ».
// Un objet explosif du film prend sa ligne et se retire de l'arme qu'il recouvre.
func TestMatchWeaponTools_ReliquatEtCategories(t *testing.T) {
	tools := serviceMatchEmprise(nil, nil).matchWeaponTools(context.Background(), "m1", nil, moiAuTableau(10, 0, 0))
	if got := outilsParNom(tools); fmt.Sprint(got) != fmt.Sprint(map[string]int{domain.SquadToolKindUnattributed: 10}) {
		t.Errorf("BTB : %v, attendu 10 non attribués", got)
	}
	bulk := []domain.BulkWeaponKillRaw{{XUID: "P", WeaponLabel: "Bobine", Class: domain.FragClassShoulder, WeaponKey: "hinf_coil", Kills: 2, FromDamageSource: true}}
	svc := serviceMatchEmprise(nil, nil).WithKillSourceCategories(categoriesFixes{rows: []port.KillSourceCategoryRow{
		{XUID: "P", Category: domain.KillSourceCategoryExplosiveObject, WeaponKey: "hinf_coil", Kills: 2},
	}})
	got := outilsParNom(svc.matchWeaponTools(context.Background(), "m1", bulk, moiAuTableau(2, 0, 0)))
	if got[domain.KillSourceCategoryExplosiveObject] != 2 || got["Bobine"] != 0 {
		t.Errorf("objet explosif : %v, attendu la ligne de catégorie seule", got)
	}
	// Catégories en échec : dégradation sans erreur, le reliquat garde ses frags.
	svc = serviceMatchEmprise(nil, nil).WithKillSourceCategories(categoriesFixes{err: errors.New("panne")})
	if tools := svc.matchWeaponTools(context.Background(), "m1", nil, moiAuTableau(1, 0, 0)); tools == nil {
		t.Errorf("catégories en échec : outils absents")
	}
	if tools := svc.matchWeaponTools(context.Background(), "m1", nil, nil); tools != nil {
		t.Errorf("sans ligne au tableau : %+v, attendu nil", tools)
	}
}
