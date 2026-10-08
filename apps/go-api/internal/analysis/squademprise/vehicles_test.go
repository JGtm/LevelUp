package squademprise

// vehicles_test.go — LA RESSOURCE « VEHICULES » DE L'EMPRISE (plan
// `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3). Une règle par test : prises par camp et
// par joueur (D2), temps à bord (D4), frags de classe véhicule par camp (D5), périmètre commun du
// rendement et rendement sur les frags appariés (D9), « non mesuré » contre zéro mesuré (D8),
// épisodes sans xuid en couverture (D10), nom des familles qualifiées, habitude.

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	warthog = "warthog"
	ghost   = "ghost"
	banshee = "banshee"
)

// ligne — une ligne de prise du match « m1 » (camp, joueur, famille, prises, ms à bord, frags appariés).
func ligne(match string, camp int, xuid, famille string, prises int, ms int64, frags int) VehicleRow {
	return VehicleRow{MatchID: match, Camp: camp, XUID: xuid, Family: famille, Takes: prises,
		AboardMS: ms, Episodes: prises, Frags: frags}
}

// passeMesuree — la passe d'un match mesuré dont les frags d'engin sont appariés.
func passeMesuree(match string, total, sansEpisode int) VehiclePass {
	return VehiclePass{MatchID: match, Measured: true, DocSchema: 71, FragsRead: true,
		FragsTotal: total, FragsUnmatched: sansEpisode}
}

func fragsDuCamp(match string, camp *int, n int) VehicleFragRow {
	return VehicleFragRow{MatchID: match, TeamID: camp, Frags: n}
}

// vehiculesUnMatch — m1, joueur au camp 0. Camp 0 : P (2 prises de Warthog), A (1 Warthog), R (1
// Ghost) ; camp 1 : E1 (2 Warthog), E2 (1 Banshee). 10 frags d'engin dont 4 tombent pendant un
// épisode (tous chez nous) ; les événements donnent 6 au camp 0, 3 au camp 1 et 1 à un tueur sans camp.
func vehiculesUnMatch() *VehicleRead {
	pass := passeMesuree("m1", 10, 6)
	pass.EpisodesRead, pass.EpisodesUnnamed, pass.EpisodesNoCamp = 9, 2, 1
	proche := ligne("m1", 0, "P", warthog, 2, 120_000, 3)
	proche.ProximityEpisodes = 1
	return &VehicleRead{
		Passes: []VehiclePass{pass},
		Rows: []VehicleRow{
			proche,
			ligne("m1", 0, "A", warthog, 1, 60_000, 1),
			ligne("m1", 0, "R", ghost, 1, 30_000, 0),
			ligne("m1", 1, "E1", warthog, 2, 90_000, 0),
			ligne("m1", 1, "E2", banshee, 1, 10_000, 0),
		},
		Frags:      []VehicleFragRow{fragsDuCamp("m1", equipe(0), 6), fragsDuCamp("m1", equipe(1), 3), fragsDuCamp("m1", nil, 1)},
		EventsRead: map[string]bool{"m1": true},
		PlayerTeam: map[string]int{"m1": 0},
	}
}

func avecVehicules(read *VehicleRead) Input {
	in := entreeUnMatch()
	in.Vehicles = read
	return in
}

func objetsDe(objs []domain.SquadEmpriseObject, res string) []domain.SquadEmpriseObject {
	var out []domain.SquadEmpriseObject
	for _, o := range objs {
		if o.Resource == res {
			out = append(out, o)
		}
	}
	return out
}

func productionDe(b domain.SquadEmpriseBlock, res string) *domain.SquadEmpriseProduction {
	for i := range b.Production {
		if b.Production[i].Resource == res {
			return &b.Production[i]
		}
	}
	return nil
}

// TestVehicules_PrisesParCampEtParJoueur — D2 et D4 : prises et temps à bord par camp, par famille et
// par joueur de l'escouade (le reste du camp à part), UNE fois même quand le film du match se lit
// (les objets du film et ceux des véhicules ne se versent pas deux fois).
func TestVehicules_PrisesParCampEtParJoueur(t *testing.T) {
	b := Build(avecVehicules(vehiculesUnMatch()))
	r := ressource(b, domain.EmpriseResourceVehicle)
	if r == nil || r.Taken != (domain.SquadEmpriseCount{Us: 4, Them: 3}) || r.MatchesMeasured != 1 {
		t.Fatalf("ressource véhicules = %+v, attendu 4 / 3 sur 1 match", r)
	}
	objs := objetsDe(b.Objects, domain.EmpriseResourceVehicle)
	if len(objs) != 3 || objs[0].Key != warthog || objs[1].Key != ghost || objs[2].Key != banshee {
		t.Fatalf("objets = %+v, attendu Warthog, Ghost, Banshee (prises de notre camp décroissantes)", objs)
	}
	w := objs[0]
	if w.Taken != (domain.SquadEmpriseCount{Us: 3, Them: 2}) || w.Aboard == nil ||
		*w.Aboard != (domain.SquadEmpriseCount{Us: 180_000, Them: 90_000}) {
		t.Errorf("Warthog = prises %+v, à bord %+v ; attendu 3 / 2 et 180 s / 90 s", w.Taken, w.Aboard)
	}
	moi, alpha, reste := w.Squad[0], w.Squad[1], w.Squad[2]
	if moi.XUID != "P" || moi.Taken != 2 || moi.AboardMS == nil || *moi.AboardMS != 120_000 ||
		alpha.XUID != "A" || alpha.Taken != 1 || *alpha.AboardMS != 60_000 ||
		reste.XUID != "" || reste.Taken != 0 || *reste.AboardMS != 0 {
		t.Errorf("parts du Warthog = %+v", w.Squad)
	}
	if g := objs[1]; g.Squad[2].XUID != "" || g.Squad[2].Taken != 1 || *g.Squad[2].AboardMS != 30_000 {
		t.Errorf("Ghost = %+v, attendu la prise du reste du camp", g.Squad)
	}
	if w.Squad[0].Kept != nil || w.Squad[0].Dropped != nil || w.PadsEmptied != nil {
		t.Errorf("issues de bonus publiées sur un véhicule : %+v", w)
	}
	// Les autres ressources restent intactes.
	if bonus := ressource(b, domain.EmpriseResourcePowerup); bonus == nil || bonus.Taken != (domain.SquadEmpriseCount{Us: 3, Them: 1}) {
		t.Errorf("bonus = %+v, attendu 3 / 1", bonus)
	}
}

// TestVehicules_CampDuJoueurDeLaPage — notre camp est celui du joueur de la page : les mêmes lignes
// lues depuis le camp 1 se renversent, prises, temps et frags.
func TestVehicules_CampDuJoueurDeLaPage(t *testing.T) {
	read := vehiculesUnMatch()
	read.PlayerTeam["m1"] = 1
	b := Build(avecVehicules(read))
	if r := ressource(b, domain.EmpriseResourceVehicle); r == nil || r.Taken != (domain.SquadEmpriseCount{Us: 3, Them: 4}) {
		t.Fatalf("prises vues du camp 1 = %+v, attendu 3 / 4", r)
	}
	if p := productionDe(b, domain.EmpriseResourceVehicle); p == nil ||
		p.Kills != (domain.SquadEmpriseCount{Us: 3, Them: 7}) {
		t.Errorf("frags vus du camp 1 = %+v, attendu 3 / 7 (le tueur sans camp chez l'adversaire)", p)
	}
}

// TestVehicules_FragsEtRendementSurLesFragsApparies — D5 et D9 : la barre épaisse garde tous les
// frags de classe véhicule du lobby (6 / 4, le tueur sans camp chez l'adversaire) ; le rendement ne
// compte que ceux tombés pendant un épisode (4 / 0) sur le temps à bord (3,5 min / 100 s).
func TestVehicules_FragsEtRendementSurLesFragsApparies(t *testing.T) {
	b := Build(avecVehicules(vehiculesUnMatch()))
	p := productionDe(b, domain.EmpriseResourceVehicle)
	if p == nil {
		t.Fatalf("production sans véhicules : %+v", b.Production)
	}
	if p.Kills != (domain.SquadEmpriseCount{Us: 6, Them: 4}) {
		t.Errorf("frags depuis un véhicule = %+v, attendu 6 / 4 (tous les frags, pas les seuls appariés)", p.Kills)
	}
	e := p.Exposure
	if e == nil || e.Kind != domain.EmpriseExposureAboardMS ||
		e.Value != (domain.SquadEmpriseCount{Us: 210_000, Them: 100_000}) ||
		e.Kills != (domain.SquadEmpriseCount{Us: 6, Them: 4}) || e.PairedKills == nil ||
		*e.PairedKills != (domain.SquadEmpriseCount{Us: 4, Them: 0}) {
		t.Fatalf("exposition = %+v, attendu temps à bord 210 s / 100 s, 6 / 4 frags de la population et 4 / 0 appariés", e)
	}
	if p.YieldUs == nil || math.Abs(*p.YieldUs-4/3.5) > 1e-9 {
		t.Errorf("rendement chez nous = %v, attendu 4 frags appariés / 3,5 min", p.YieldUs)
	}
	if p.YieldThem == nil || *p.YieldThem != 0 || p.RelativeGap != nil {
		t.Errorf("rendement chez eux = %v, écart = %v ; attendu 0 mesuré et pas d'écart relatif (diviseur nul)",
			p.YieldThem, p.RelativeGap)
	}
	cov := b.Vehicles
	if cov == nil || cov.FragsMatches != 1 || cov.FragsTotal != 10 || cov.FragsPaired != 4 ||
		cov.PairedShare == nil || math.Abs(*cov.PairedShare-0.4) > 1e-9 {
		t.Errorf("couverture des frags = %+v, attendu 4 appariés sur 10 (0,4)", cov)
	}
}

// TestVehicules_PerimetreCommunDuRendement — D9 : un match dont les frags n'ont pas été appariés, ou
// dont les événements de mort manquent, sort des frags, du temps à bord et du rendement (mêmes
// matchs pour le numérateur et le dénominateur) ; un match sans frag d'engin y reste (zéro mesuré).
func TestVehicules_PerimetreCommunDuRendement(t *testing.T) {
	in := avecVehicules(&VehicleRead{EventsRead: map[string]bool{}, PlayerTeam: map[string]int{}})
	in.Current = nil
	read := in.Vehicles
	ajouter := func(id string, pass VehiclePass, events bool, rows ...VehicleRow) {
		in.Current = append(in.Current, Match{MatchID: id, StartTime: t0.Add(time.Duration(len(in.Current)) * time.Hour)})
		pass.MatchID = id
		read.Passes = append(read.Passes, pass)
		read.PlayerTeam[id] = 0
		read.EventsRead[id] = events
		read.Rows = append(read.Rows, rows...)
	}
	// a : frags non appariés (raison dite) ; b : frags appariés mais événements absents ;
	// c : aucun frag d'engin ; d : 3 frags dont 1 sans épisode.
	sansFrags := VehiclePass{Measured: true, FragsRead: false, FragsReason: "no_kill_source"}
	ajouter("a", sansFrags, false, ligne("a", 0, "P", warthog, 1, 100_000, 0))
	ajouter("b", passeMesuree("b", 4, 0), false, ligne("b", 0, "P", warthog, 1, 100_000, 0))
	ajouter("c", passeMesuree("c", 0, 0), false, ligne("c", 0, "P", ghost, 1, 60_000, 0))
	ajouter("d", passeMesuree("d", 3, 1), true, ligne("d", 0, "P", warthog, 1, 120_000, 2))
	read.Frags = []VehicleFragRow{fragsDuCamp("d", equipe(0), 2), fragsDuCamp("d", equipe(1), 1)}

	b := Build(in)
	p := productionDe(b, domain.EmpriseResourceVehicle)
	if p == nil || p.Kills != (domain.SquadEmpriseCount{Us: 2, Them: 1}) ||
		p.Exposure.Value != (domain.SquadEmpriseCount{Us: 180_000}) ||
		p.Exposure.Kills != (domain.SquadEmpriseCount{Us: 2, Them: 1}) || *p.Exposure.PairedKills != (domain.SquadEmpriseCount{Us: 2}) {
		t.Fatalf("production = %+v / %+v, attendu les seuls matchs c et d : 180 s à bord, 2 frags appariés, 2 / 1 frags", p, p.Exposure)
	}
	if p.YieldUs == nil || math.Abs(*p.YieldUs-2/3.0) > 1e-9 {
		t.Errorf("rendement = %v, attendu 2 frags par 3 min", p.YieldUs)
	}
	if cov := b.Vehicles; cov.MatchesMeasured != 4 || cov.FragsMatches != 2 || cov.FragsTotal != 3 || cov.FragsPaired != 2 {
		t.Errorf("couverture = %+v, attendu 4 matchs mesurés dont 2 au périmètre du rendement, 2 frags appariés sur 3", cov)
	}
	// Les prises de a et b restent comptées : seul le rendement les écarte.
	if r := ressource(b, domain.EmpriseResourceVehicle); r == nil || r.Taken.Us != 4 {
		t.Errorf("prises = %+v, attendu les 4 prises des quatre matchs", r)
	}
}

// TestVehicules_NonMesureEtZeroMesure — D8 : un match sans passe, à occupation non lue ou au camp
// inconnu est « non mesuré » (et n'entre dans aucun compte) ; un match mesuré sans prise est un zéro
// mesuré — la ressource n'a pas d'entrée, l'état du match le dit.
func TestVehicules_NonMesureEtZeroMesure(t *testing.T) {
	in := entreeUnMatch()
	for i, id := range []string{"m2", "m3", "m4"} {
		in.Current = append(in.Current, Match{MatchID: id, StartTime: t0.Add(time.Duration(i+1) * time.Hour)})
	}
	in.Vehicles = &VehicleRead{
		Passes: []VehiclePass{
			passeMesuree("m1", 0, 0), // zéro mesuré
			{MatchID: "m3", Measured: false, Reason: "schema_before_67"},
			passeMesuree("m4", 0, 0), // camp du joueur inconnu
		},
		Rows:       []VehicleRow{ligne("m4", 0, "P", warthog, 5, 50_000, 0)},
		EventsRead: map[string]bool{},
		PlayerTeam: map[string]int{"m1": 0, "m3": 0},
	}
	b := Build(in)
	want := []struct{ etat, raison string }{
		{domain.EmpriseVehiclesMeasured, ""},
		{domain.EmpriseVehiclesNotMeasured, domain.EmpriseVehiclesNoPass},
		{domain.EmpriseVehiclesNotMeasured, "schema_before_67"},
		{domain.EmpriseVehiclesNotMeasured, domain.EmpriseVehiclesTeamUnknown},
	}
	for i, w := range want {
		if m := b.Matches[i]; m.Vehicles != w.etat || m.VehiclesReason != w.raison {
			t.Errorf("match %s : état %q raison %q, attendu %q %q", m.MatchID, m.Vehicles, m.VehiclesReason, w.etat, w.raison)
		}
	}
	if ressource(b, domain.EmpriseResourceVehicle) != nil || len(objetsDe(b.Objects, domain.EmpriseResourceVehicle)) != 0 {
		t.Errorf("ressource publiée alors que seuls un zéro mesuré et des matchs non mesurés existent : %+v", b.Resources)
	}
	if productionDe(b, domain.EmpriseResourceVehicle) != nil {
		t.Errorf("production véhicules publiée sans frag ni temps à bord : %+v", b.Production)
	}
	if cov := b.Vehicles; cov == nil || cov.MatchesMeasured != 1 || cov.MatchesNotMeasured != 3 {
		t.Errorf("couverture = %+v, attendu 1 mesuré, 3 non mesurés", cov)
	}
}

// TestVehicules_CouvertureDesEpisodes — D10 : les épisodes sans joueur nommé, sans camp et datés par
// proximité sont comptés en couverture, jamais en prise.
func TestVehicules_CouvertureDesEpisodes(t *testing.T) {
	cov := Build(avecVehicules(vehiculesUnMatch())).Vehicles
	if cov == nil || cov.EpisodesRead != 9 || cov.EpisodesUnnamed != 2 || cov.EpisodesNoCamp != 1 ||
		cov.ProximityEpisodes != 1 || cov.MatchesMeasured != 1 || cov.MatchesNotMeasured != 0 {
		t.Errorf("couverture = %+v, attendu 9 lus dont 2 sans nom, 1 sans camp, 1 par proximité", cov)
	}
}

// TestVehicules_IndependantDuFilm — la ressource vient de l'artefact : sans résumé d'usage, elle se
// publie (et seulement elle avec les frags aux armes spéciales de la feuille).
func TestVehicules_IndependantDuFilm(t *testing.T) {
	in := avecVehicules(vehiculesUnMatch())
	in.Film, in.FilmUnavailable = nil, domain.EmpriseFilmUnsupported
	b := Build(in)
	if r := ressource(b, domain.EmpriseResourceVehicle); r == nil || r.Taken != (domain.SquadEmpriseCount{Us: 4, Them: 3}) {
		t.Fatalf("véhicules sans film = %+v, attendu 4 / 3", r)
	}
	if len(b.Resources) != 1 || len(objetsDe(b.Objects, domain.EmpriseResourcePowerup)) != 0 {
		t.Errorf("grandeurs du film publiées sans film : %+v", b.Resources)
	}
	m := b.Matches[0]
	if m.HasFilm || m.Vehicles != domain.EmpriseVehiclesMeasured || len(m.Resources) != 1 ||
		m.Resources[0].Resource != domain.EmpriseResourceVehicle {
		t.Errorf("match = %+v, attendu sans film, véhicules mesurés, une seule ressource", m)
	}
	if productionDe(b, domain.EmpriseResourceVehicle) == nil {
		t.Errorf("production sans véhicules : %+v", b.Production)
	}
}

// TestVehicules_NonLueOuEnEchec — ressource non lue (titre sans la capability) : rien, ni état de
// match ni couverture ; lecture en échec : la couverture le dit, aucune donnée.
func TestVehicules_NonLueOuEnEchec(t *testing.T) {
	b := Build(entreeUnMatch())
	if b.Vehicles != nil || b.Matches[0].Vehicles != "" || ressource(b, domain.EmpriseResourceVehicle) != nil {
		t.Errorf("ressource non lue publiée : %+v / %q", b.Vehicles, b.Matches[0].Vehicles)
	}
	in := entreeUnMatch()
	in.VehiclesUnavailable = domain.EmpriseVehiclesLoadFailed
	b = Build(in)
	if b.Vehicles == nil || b.Vehicles.Unavailable != domain.EmpriseVehiclesLoadFailed || b.Vehicles.MatchesMeasured != 0 ||
		b.Matches[0].Vehicles != "" {
		t.Errorf("échec de lecture = %+v, attendu la raison et rien d'autre", b.Vehicles)
	}
}

// TestVehicules_NomDesFamillesQualifiees — seules les familles que le titre qualifie portent un nom ;
// les autres sont des noms propres que le client affiche depuis leur clé.
func TestVehicules_NomDesFamillesQualifiees(t *testing.T) {
	read := vehiculesUnMatch()
	read.Rows = append(read.Rows, ligne("m1", 0, "P", "tourelle_fixe", 1, 5_000, 0))
	in := avecVehicules(read)
	in.VehicleLabels = map[string]string{"tourelle_fixe": "Tourelle fixe"}
	for _, o := range objetsDe(Build(in).Objects, domain.EmpriseResourceVehicle) {
		want := ""
		if o.Key == "tourelle_fixe" {
			want = "Tourelle fixe"
		}
		if o.Label != want {
			t.Errorf("%s : libellé %q, attendu %q", o.Key, o.Label, want)
		}
	}
}

// TestVehicules_Habitude — une soirée précédente publie notre part des prises de véhicule, après
// celle des bonus.
func TestVehicules_Habitude(t *testing.T) {
	in := entreeUnMatch()
	in.Timeline = append(in.Timeline, in.Current...)
	soireeFilmee(&in, "a", t0.Add(-72*time.Hour), "Assassin", 3, 1)
	in.Vehicles = &VehicleRead{
		Passes:     []VehiclePass{passeMesuree("a-Assassin", 0, 0)},
		Rows:       []VehicleRow{ligne("a-Assassin", 0, "P", warthog, 3, 1000, 0), ligne("a-Assassin", 1, "E1", warthog, 1, 1000, 0)},
		EventsRead: map[string]bool{},
		PlayerTeam: map[string]int{"a-Assassin": 0},
	}
	h := Build(in).Habit
	if h == nil || len(h.Previous) != 1 {
		t.Fatalf("habitude = %+v", h)
	}
	part := h.Previous[0].Shares
	if len(part) != 2 || part[1].Resource != domain.EmpriseResourceVehicle ||
		part[1].Taken != (domain.SquadEmpriseCount{Us: 3, Them: 1}) || math.Abs(part[1].Share-0.75) > 1e-9 {
		t.Errorf("parts de la soirée précédente = %+v, attendu bonus puis véhicules 3 / 1", part)
	}
	for _, sh := range h.Current.Shares {
		if sh.Resource == domain.EmpriseResourceVehicle {
			t.Errorf("ce soir = %+v : part de véhicules publiée sans passe (m1 n'en a pas)", h.Current.Shares)
		}
	}
}

// TestVehicules_OrdreDesRessources — bonus, armes spéciales, véhicules, râteliers.
func TestVehicules_OrdreDesRessources(t *testing.T) {
	want := []string{domain.EmpriseResourcePowerup, domain.EmpriseResourcePowerWeapon,
		domain.EmpriseResourceVehicle, domain.EmpriseResourceRack}
	for i, r := range want {
		if resourceOrder[i] != r || resourceRank(r) != i {
			t.Fatalf("resourceOrder = %v, attendu %v", resourceOrder, want)
		}
	}
	b := Build(avecVehicules(vehiculesUnMatch()))
	var vus []string
	for _, r := range b.Matches[0].Resources {
		vus = append(vus, r.Resource)
	}
	if len(vus) != 4 || vus[2] != domain.EmpriseResourceVehicle || vus[3] != domain.EmpriseResourceRack {
		t.Errorf("ressources du match = %v, attendu bonus, armes, véhicules, râtelier", vus)
	}
}

// TestVehicules_TempsABordSansPrise — un second passager de la même vie prend 0 et passe du temps à
// bord : la famille reste publiée avec son temps, prises nulles (D2 : une prise par vie et par camp).
func TestVehicules_TempsABordSansPrise(t *testing.T) {
	read := vehiculesUnMatch()
	read.Rows = append(read.Rows, ligne("m1", 0, "A", "mongoose", 0, 20_000, 0))
	var mongoose *domain.SquadEmpriseObject
	objs := Build(avecVehicules(read)).Objects
	for i := range objs {
		if objs[i].Key == "mongoose" {
			mongoose = &objs[i]
		}
	}
	if mongoose == nil || mongoose.Taken != (domain.SquadEmpriseCount{}) || mongoose.Aboard == nil ||
		mongoose.Aboard.Us != 20_000 || mongoose.Squad[1].XUID != "A" || *mongoose.Squad[1].AboardMS != 20_000 {
		t.Errorf("Mongoose = %+v, attendu publié sans prise avec 20 s à bord d'Alpha", mongoose)
	}
}

// TestVehicules_ZeroFragSansPartApparie — revue L7.5, RV4 : sur un périmètre mesuré où AUCUN frag
// d'engin n'est compté, la part appariée reste ABSENTE (0 / 0 n'est pas un nombre) ; sinon elle
// vaudrait NaN et la page ne se sérialiserait plus.
func TestVehicules_ZeroFragSansPartApparie(t *testing.T) {
	read := vehiculesUnMatch()
	read.Passes = []VehiclePass{passeMesuree("m1", 0, 0)}
	read.Frags = nil
	b := Build(avecVehicules(read))
	cov := b.Vehicles
	if cov == nil || cov.MatchesMeasured != 1 || cov.FragsTotal != 0 {
		t.Fatalf("couverture = %+v, attendu un match mesuré sans frag d'engin", cov)
	}
	if cov.PairedShare != nil {
		t.Errorf("part appariée = %v à zéro frag, attendu absente", *cov.PairedShare)
	}
	if _, err := json.Marshal(b); err != nil {
		t.Errorf("le bloc ne se sérialise plus : %v", err)
	}
}
