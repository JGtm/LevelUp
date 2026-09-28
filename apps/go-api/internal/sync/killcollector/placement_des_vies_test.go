package killcollector

// placement_des_vies_test.go — LA PROJECTION DU PLACEMENT DES VIES, SANS FIXTURE DE FILM (plan
// `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2). Meme patron que `isolation_facts_test.go` :
// ses gardes, ses traductions et son chemin d'erreur sont verifies partout, films absents.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// TestProjeterPlacementDesVies_PontNonPubliable_NeTenteAucuneEcriture — un pont refuse ne rend
// aucune ligne (meme refus que le contexte des morts), et la garde coupe AVANT le lease.
func TestProjeterPlacementDesVies_PontNonPubliable_NeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{acquireShared: panicWriterIso}
	avant := observability.LoadCounter(metricPlacementPontRefuse)
	mat := materiauDIsolement{registre: registreDeTest(positionsDUneVie(), 1)}
	c.projeterPlacementDesVies(context.Background(), "m1", mat,
		MatchIdentities{Equipes: map[string]int{"111": 0}}, persist.KillSourceBatch{})
	if got := observability.LoadCounter(metricPlacementPontRefuse) - avant; got != 1 {
		t.Fatalf("%s a bouge de %d, attendu 1", metricPlacementPontRefuse, got)
	}
}

// TestProjeterPlacementDesVies_EchecDEcriture_NEstPasBloquant — le contrat de l'etape : son
// echec ne coute ni les morts, ni les positions, ni les vies. Aucune panique, aucun retour :
// la sortie est le journal et le compteur.
func TestProjeterPlacementDesVies_EchecDEcriture_NEstPasBloquant(t *testing.T) {
	c := &KillSourceCollector{
		acquireShared: func(context.Context) (*sql.DB, func(), error) {
			return nil, nil, errors.New("lease indisponible")
		},
	}
	avant := observability.LoadCounter(metricPlacementWriteFail)
	c.projeterPlacementDesVies(context.Background(), "m1", materiauAvecUneVie(),
		MatchIdentities{Equipes: map[string]int{"111": 0}}, persist.KillSourceBatch{})
	if got := observability.LoadCounter(metricPlacementWriteFail) - avant; got != 1 {
		t.Fatalf("%s a bouge de %d, attendu 1", metricPlacementWriteFail, got)
	}
}

// TestPortagesDuMatch_HorsModeAPorteur_RienNEstLu — LA GARDE DE MODE, tenue par l'entree
// (`replay.GardesDeLaVariante`) : hors drapeau, crane, bombe et VIP, aucune lecture n'est payee.
// Le film et le contexte sont NIL : la moindre lecture paniquerait. Le bilan doit etre VIERGE
// hors des gardes (un assemblage sur un document vide le marquerait « sans calage »).
func TestPortagesDuMatch_HorsModeAPorteur_RienNEstLu(t *testing.T) {
	c := &KillSourceCollector{}
	for _, v := range []string{"Team Slayer:Arena", "Super Fiesta:Fiesta", "Arena:King of the Hill", ""} {
		gardes := replay.GardesDeLaVariante(v)
		if !gardes.Aucune() {
			t.Fatalf("%q : variante a porteur, le test la croit hors mode", v)
		}
		portages, bilan := c.portagesDuMatch(context.Background(), "m1", materiauDIsolement{},
			MatchIdentities{Variante: v})
		if portages != nil || bilan != (replay.BilanPortages{Gardes: gardes}) {
			t.Fatalf("%q : portages %v, bilan %+v — une lecture a ete payee hors mode a porteur",
				v, portages, bilan)
		}
	}
}

// TestToPlacementRows_TraduitSansRienInventer — la traduction pure, NULL compris, et le compte
// des vies non mesurees (mediane nulle).
func TestToPlacementRows_TraduitSansRienInventer(t *testing.T) {
	med, radar, hors := 12.5, 18.0, int64(300)
	rows, nonMesurees := toPlacementRows([]replay.PlacementVie{
		{XUID: 111, DebutMS: 0, FinMS: 9_900, DureeMS: 9_900, MesureMS: 8_000, MedianeM: &med,
			HorsRadarMS: &hors, RadarM: &radar, PorteurMS: 1_000, EquipeATerreMS: 500,
			NonSitueMS: 300, CoequipierNonSitueMS: 200, Frags: 3},
		{XUID: 222, DebutMS: 5, FinMS: 1_005, DureeMS: 1_000, MesureMS: 1_100},
	})
	if len(rows) != 2 || nonMesurees != 1 {
		t.Fatalf("%d lignes, %d non mesurees ; attendu 2 et 1", len(rows), nonMesurees)
	}
	a := rows[0]
	if a.XUID != "111" || a.StartMS != 0 || a.EndMS != 9_900 || a.DurationMS != 9_900 ||
		a.MeasuredMS != 8_000 || *a.MedianM != 12.5 || *a.BeyondMS != 300 || *a.RadarM != 18 ||
		a.CarrierMS != 1_000 || a.TeamDownMS != 500 || a.UnplacedMS != 300 ||
		a.TeammateUnplacedMS != 200 || a.Kills != 3 {
		t.Fatalf("ligne 111 = %+v", a)
	}
	if b := rows[1]; b.XUID != "222" || b.MedianM != nil || b.BeyondMS != nil || b.RadarM != nil {
		t.Fatalf("ligne 222 = %+v : les absences restent des absences", b)
	}
}

// TestJournalDuPlacement_PorteLaPubliabiliteEtLeTueurDuFil — le tueur est celui du FIL (jamais l assistant), la
// publiabilite celle de la passe fusionnee, et un xuid illisible devient « non resolu » (0).
func TestJournalDuPlacement_PorteLaPubliabiliteEtLeTueurDuFil(t *testing.T) {
	j := journalDuPlacement(persist.KillSourceBatch{Publishable: true, Deaths: []persist.KillEventInsert{
		{TimeMS: 1_000, FeedKillerXUID: "111", AssistXUID: "999", VictimXUID: "222"},
		{TimeMS: 2_000, FeedKillerXUID: "", VictimXUID: "bid(1.0)"},
	}})
	if len(j) != 2 {
		t.Fatalf("%d frags, attendu 2 (aucun ecart ici : les refus sont ceux du calcul pur)", len(j))
	}
	if j[0] != (replay.FragDuJournal{TueurXUID: 111, VictimeXUID: 222, TempsMS: 1_000, Publiable: true}) {
		t.Fatalf("frag 0 = %+v", j[0])
	}
	if j[1].TueurXUID != 0 || j[1].VictimeXUID != 0 || !j[1].Publiable {
		t.Fatalf("frag 1 = %+v", j[1])
	}
	if n := journalDuPlacement(persist.KillSourceBatch{Deaths: []persist.KillEventInsert{{}}}); n[0].Publiable {
		t.Fatal("une passe non publiable a produit un frag publiable")
	}
}

// TestPorteeDe — sans portee injectee ou variante inconnue : nil (la ligne s'ecrit sans
// portee) ; variante connue : la portee.
func TestPorteeDe(t *testing.T) {
	if (depsDuPlacement{}).porteeDe("Team Slayer:Arena") != nil {
		t.Fatal("sans portee injectee, une portee est sortie")
	}
	d := depsDuPlacement{portee: func(v string) (float64, bool) { return 18, v == "Team Slayer:Arena" }}
	if d.porteeDe("Super Fiesta:Fiesta") != nil {
		t.Fatal("variante inconnue : une portee est sortie")
	}
	if p := d.porteeDe("Team Slayer:Arena"); p == nil || *p != 18 {
		t.Fatalf("variante connue : portee %v, attendu 18", p)
	}
}

// TestSoclesDe_SansCatalogueNiCarte — l'absence degrade en « aucun socle », comme a la cuisson.
func TestSoclesDe_SansCatalogueNiCarte(t *testing.T) {
	if s := (depsDuPlacement{}).soclesDe(context.Background(), "m1", "une-carte"); s != nil {
		t.Fatalf("sans catalogue : %v", s)
	}
	d := depsDuPlacement{objectifs: &replay.MapObjectivesCatalog{}}
	if s := d.soclesDe(context.Background(), "m1", ""); s != nil {
		t.Fatalf("sans map_id : %v", s)
	}
	if s := d.soclesDe(context.Background(), "m1", "carte-inconnue"); s != nil {
		t.Fatalf("carte hors catalogue : %v", s)
	}
}

// TestEntreeDesPorteurs_PorteCeQueLaPasseALu — LA COUTURE vers `replay.PortagesAuSync`. Chaque
// champ transmis est pince : la variante (garde de mode), la feuille (pont par manche), les
// socles de la carte (par `map_id`), les libelles, et le materiau de la passe tel quel. La feuille
// ne sert au pont que sur certains films — son retrait laisserait verts les temoins reels
// (mutation constatee au lot V2 : sans elle, les films CTF et Oddball temoins rendent les memes
// portages).
func TestEntreeDesPorteurs_PorteCeQueLaPasseALu(t *testing.T) {
	socle := mapvar.Objective{Role: mapvar.RoleFlagSpawn, InstanceID: 11, TeamIndex: 1,
		Pos: mapvar.Vec3{X: 4, Y: 2}, Labels: []string{"ctf_include", "flag_spawn"}}
	d := depsDuPlacement{
		libelles: replay.LabelCatalog{Grenades: []replay.Label{{}}},
		objectifs: &replay.MapObjectivesCatalog{Maps: map[string]replay.MapObjectivesEntry{
			"carte-1": {MapID: "carte-1", Objectives: []mapvar.Objective{socle}},
		}},
	}
	mat := materiauDIsolement{film: &decfilm.Film{}, contexte: &decfilm.FilmContext{},
		carte: decfilm.MapQuantEntry{Module: "module-1"}, profil: &decfilm.ProfilDeBalayage{},
		identite: replay.IdentityInput{MatchID: "m1"}}
	ids := MatchIdentities{Variante: "CTF:Arena", CarteID: "carte-1",
		Feuille: []decfilm.PlayerLine{{XUID: "111", Kills: 3, Deaths: 1, Assists: 2}}}

	e := d.entreeDesPorteurs(context.Background(), "m1", mat, ids)

	if e.MatchID != "m1" || e.Film != mat.film || e.Contexte != mat.contexte || e.Carte.Module != "module-1" ||
		e.ProfilDeBalayage != mat.profil || e.Identite.MatchID != "m1" {
		t.Fatalf("materiau de la passe mal transmis : %+v", e)
	}
	if e.Variante != "CTF:Arena" {
		t.Fatalf("variante = %q : la garde de mode ne lirait pas la bonne famille", e.Variante)
	}
	if len(e.Lignes) != 1 || e.Lignes[0] != ids.Feuille[0] {
		t.Fatalf("feuille = %+v : le pont par manche perdrait son triplet", e.Lignes)
	}
	if len(e.Socles) != 1 || e.Socles[0] != (replay.FlagSpawn{Team: 1, X: 4, Y: 2}) {
		t.Fatalf("socles = %+v, attendu le socle d'equipe 1 de la carte", e.Socles)
	}
	if len(e.Libelles.Grenades) != 1 {
		t.Fatalf("libelles non transmis : %+v", e.Libelles)
	}
}
