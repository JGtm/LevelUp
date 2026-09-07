package service

// tactical_service_lectures_test.go — LES ROUTES, L'ISOLEMENT, LES GRAPPES ET LE FILTRE DE
// SPAWN, sur des sidecars poses a la main.
//
// L'algorithme d'isolement a ses propres tests (analysis/coordination) : la borne du rayon,
// les deux exclusions, la forme canonique du taux. CE QUI SE VERIFIE ICI est ce que seul le
// service peut faire — joindre les EQUIPES (le film ne les porte pas) et resoudre le rayon
// de la VARIANTE de chaque match.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// mockCallouts double le lecteur de zones nommees.
type mockCallouts struct{ zones []domain.ZoneNommee }

func (m *mockCallouts) ZonesDeLaCarte(context.Context, string) []domain.ZoneNommee {
	return m.zones
}

// universVariantes pose des matchs mesures et ELIGIBLES, moi + un ami contre deux
// adversaires. La cle de la map nomme le match ; sa valeur ne sert plus (le nom de variante
// est parti avec la lecture d'isolement), elle documente la fixture.
func universVariantes(parMatch map[string]string) domain.TacticalUnivers {
	u := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range triees(parMatch) {
		u.Matchs = append(u.Matchs, domain.TacticalMatch{
			MatchID: id, Outcome: domain.OutcomeWin, Mesure: true, EligibleALaCuisson: true,
		})
		u.Equipes[id] = map[string]int{tsMoi: 0, tsAmi: 0, tsAdv: 1, tsAdv2: 1}
	}
	return u
}

func triees(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// svcArtefact monte le service avec un lecteur de sidecars et le journal du repo.
func svcArtefact(univ domain.TacticalUnivers, store *mockRasterStore,
	morts ...domain.KillEvent) *TacticalService {
	repo := &mockTacticalRepo{
		univ: univ,
		ev:   domain.TacticalKillEvents{Univers: univ, Events: morts},
	}
	return NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)
}

// ─── ROUTES ────────────────────────────────────────────────────────────────────

// sidecarRoutes pose un joueur dont chaque vie a un chemin.
func sidecarRoutes(matchID string, chemins ...[]domain.TacticalRasterCase) *domain.TacticalRasterSidecar {
	j := domain.TacticalRasterJoueur{XUID: tsMoi}
	for i, ch := range chemins {
		j.Routes = append(j.Routes, domain.TacticalRasterRoute{DebutFrame: i * 100, Cases: ch})
	}
	sc := sidecarPose(matchID)
	sc.Joueurs = []domain.TacticalRasterJoueur{j}
	return sc
}

// TestRoutes_ComptentDesPassages — une cellule pese autant qu'on la TRAVERSE, jamais autant
// qu'on y reste.
//
// C'est ce qui distingue « par ou je sors » de « ou je passe mon temps ». La cellule (0,0)
// est traversee par les trois matchs ; la cellule (5,5), par un seul, tombe sous le plancher.
func TestRoutes_ComptentDesPassages(t *testing.T) {
	commun := []domain.TacticalRasterCase{{Col: 0, Lig: 0}, {Col: 1, Lig: 0}}
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarRoutes("m1", commun, commun), // deux vies : deux passages
		"m2": sidecarRoutes("m2", commun),
		"m3": sidecarRoutes("m3", commun, []domain.TacticalRasterCase{{Col: 5, Lig: 5}}),
	}}
	svc := svcArtefact(universVariantes(map[string]string{
		"m1": "Slayer:Arena", "m2": "Slayer:Arena", "m3": "Slayer:Arena",
	}), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	})
	if err != nil {
		t.Fatalf("lecture routes: %v", err)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3", out.MatchsRetenus)
	}
	c := celluleEn(out.Cellules, 0.25, 0.25)
	if c == nil {
		t.Fatalf("cellule (0,0) absente : %+v", out.Cellules)
	}
	// m1 la traverse DEUX fois (deux vies), m2 et m3 une fois : 4 passages sur 3 matchs.
	if c.Brut != 4 {
		t.Fatalf("passages = %v, attendu 4 (2 + 1 + 1)", c.Brut)
	}
	if c.Matchs != 3 {
		t.Fatalf("matchs distincts = %d, attendu 3", c.Matchs)
	}
	// LES ROUTES NE SONT PAS DES SECONDES : la valeur reste un compte par match, jamais
	// convertie par le pas d'echantillonnage de l'occupation.
	if c.Valeur != 4.0/3.0 {
		t.Fatalf("valeur = %v, attendu 4/3 passages par match (jamais des secondes)", c.Valeur)
	}
	if celluleEn(out.Cellules, 2.75, 2.75) != nil {
		t.Fatalf("la cellule vue dans un seul match a ete peinte : %+v", out.Cellules)
	}
}

// ─── GRAPPES ET FILTRE DE SPAWN ────────────────────────────────────────────────

// sidecarSpawn pose un joueur dont la premiere vie part de (x, y).
func sidecarSpawn(matchID string, x, y float64) *domain.TacticalRasterSidecar {
	sc := sidecarPose(matchID)
	sc.Joueurs = []domain.TacticalRasterJoueur{{
		XUID: tsMoi,
		Spawns: []domain.TacticalRasterSpawn{
			{Frame: 0, X: x, Y: y, PremiereVie: true},
			// Une reapparition AILLEURS, qui ne doit jamais entrer dans une grappe.
			{Frame: 500, X: 90 + x, Y: 90 + y},
		},
		Cellules: []domain.TacticalRasterCellule{{Col: int(x * 2), Lig: int(y * 2), Echantillons: 8}},
	}}
	return sc
}

func lireTemps(t *testing.T, svc *TacticalService, spawn string, ids ...string) domain.TacticalRaster {
	t.Helper()
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: spawn},
	})
	if err != nil {
		t.Fatalf("lecture temps (spawn=%q): %v", spawn, err)
	}
	return out
}

// grappesFixture : trois matchs partis d'une base, trois d'une autre.
func grappesFixture() (*mockRasterStore, domain.TacticalUnivers, []string) {
	sc := map[string]*domain.TacticalRasterSidecar{}
	variantes := map[string]string{}
	ids := []string{}
	for i, id := range []string{"a1", "a2", "a3"} {
		sc[id] = sidecarSpawn(id, 0.25+float64(i)*0.4, 0.25)
		variantes[id] = "Slayer:Arena"
		ids = append(ids, id)
	}
	for i, id := range []string{"b1", "b2", "b3"} {
		sc[id] = sidecarSpawn(id, 50.25+float64(i)*0.4, 50.25)
		variantes[id] = "Slayer:Arena"
		ids = append(ids, id)
	}
	return &mockRasterStore{sidecars: sc}, universVariantes(variantes), ids
}

// TestGrappes_ServiesAvecLaLecture — deux bases, deux grappes, nommees par les callouts.
func TestGrappes_ServiesAvecLaLecture(t *testing.T) {
	store, univ, ids := grappesFixture()
	svc := svcArtefact(univ, store).WithCalloutsStore(&mockCallouts{zones: []domain.ZoneNommee{
		{NomFR: "Base rouge", NomEN: "Red base", X: 0, Y: 0},
		{NomFR: "Base bleue", NomEN: "Blue base", X: 51, Y: 51},
	}})
	out := lireTemps(t, svc, "", ids...)
	if len(out.Grappes) != 2 {
		t.Fatalf("grappes = %+v, attendu 2", out.Grappes)
	}
	noms := map[string]bool{out.Grappes[0].NomFR: true, out.Grappes[1].NomFR: true}
	if !noms["Base rouge"] || !noms["Base bleue"] {
		t.Fatalf("grappes nommees %v, attendu les deux bases", noms)
	}
	// LES DEUX LANGUES VOYAGENT : un nom de lieu vient du catalogue du JEU, et le client ne
	// peut pas le traduire.
	for _, g := range out.Grappes {
		if g.NomEN == "" {
			t.Fatalf("grappe %s sans nom EN : %+v", g.ID, g)
		}
	}
	for _, g := range out.Grappes {
		if g.Matchs != 3 {
			t.Fatalf("grappe %s vue dans %d matchs, attendu 3", g.ID, g.Matchs)
		}
	}
}

// TestFiltreSpawn_RestreintLUnivers — LE FILTRE PORTE SUR L'UNIVERS, PAS SUR LES POINTS.
//
// Garder au denominateur les matchs partis de l'autre base ferait repondre « je passe peu
// de temps ici » a une carte ou l'on n'a simplement pas commence.
func TestFiltreSpawn_RestreintLUnivers(t *testing.T) {
	store, univ, ids := grappesFixture()
	svc := svcArtefact(univ, store)
	complet := lireTemps(t, svc, "", ids...)
	if complet.MatchsFiltres != 6 || complet.MatchsRetenus != 6 {
		t.Fatalf("sans filtre : filtres=%d retenus=%d, attendu 6 et 6",
			complet.MatchsFiltres, complet.MatchsRetenus)
	}
	if len(complet.Grappes) != 2 {
		t.Fatalf("grappes = %+v, attendu 2", complet.Grappes)
	}

	restreint := lireTemps(t, svc, complet.Grappes[0].ID, ids...)
	if restreint.MatchsFiltres != 3 || restreint.MatchsRetenus != 3 {
		t.Fatalf("avec filtre : filtres=%d retenus=%d, attendu 3 et 3 — le filtre doit "+
			"restreindre l'UNIVERS, pas seulement les points peints",
			restreint.MatchsFiltres, restreint.MatchsRetenus)
	}
	// LES GRAPPES RESTENT TOUTES LES DEUX : la page les propose, et une liste qui se
	// reduirait a la selection enfermerait l'utilisateur dedans.
	if len(restreint.Grappes) != 2 {
		t.Fatalf("grappes sous filtre = %+v, attendu les 2 (la liste ne se reduit pas)",
			restreint.Grappes)
	}
}

// TestFiltreSpawn_GrappeInconnue — une grappe absente de l'univers courant est un 404 type,
// jamais une lecture silencieusement non filtree.
func TestFiltreSpawn_GrappeInconnue(t *testing.T) {
	store, univ, ids := grappesFixture()
	svc := svcArtefact(univ, store)
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: "s+999999+999999"},
	})
	if err == nil {
		t.Fatal("grappe inconnue : attendu un refus, pas une lecture non filtree")
	}
	if err.Error() != domain.ErrTacticalSpawnInconnu.Error() {
		t.Fatalf("err = %v, attendu ErrTacticalSpawnInconnu", err)
	}
}

// TestFiltreSpawn_SappliqueAuxAutresLectures — le filtre est un filtre d'univers : il vaut
// pour « isole » comme pour « temps ».
func TestFiltreSpawn_SappliqueAuxAutresLectures(t *testing.T) {
	store, univ, ids := grappesFixture()
	// Chaque match porte en plus une route, pour que la seconde lecture d'artefact ait de
	// quoi repondre sous le filtre.
	for id, sc := range store.sidecars {
		sc.Joueurs[0].Routes = []domain.TacticalRasterRoute{{
			DebutFrame: 0,
			Cases:      []domain.TacticalRasterCase{{Col: 4, Lig: 6}, {Col: 5, Lig: 6}},
		}}
		store.sidecars[id] = sc
	}
	svc := svcArtefact(univ, store)
	complet := lireTemps(t, svc, "", ids...)

	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: complet.Grappes[0].ID},
	})
	if err != nil {
		t.Fatalf("lecture routes filtree: %v", err)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3 sous le filtre de spawn", out.MatchsRetenus)
	}
	if out.MatchsFiltres != 3 {
		t.Fatalf("matchs_filtres = %d, attendu 3 : le filtre de spawn est un filtre d'UNIVERS",
			out.MatchsFiltres)
	}
}

// TestLecturesDArtefact_MemePorte — les deux lectures partagent `film.replay_artifact`.
func TestLecturesDArtefact_MemePorte(t *testing.T) {
	store, univ, ids := grappesFixture()
	repo := &mockTacticalRepo{univ: univ}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi).WithRasterStore(store)
	for _, q := range []string{
		domain.TacticalQuestionTemps, domain.TacticalQuestionRoutes,
	} {
		_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
			MapID: "streets", Question: q, Qui: domain.TacticalQuiMoi,
			Scope: domain.TacticalScope{MatchIDs: ids},
		})
		if err == nil || err.Error() != games.ErrCapabilityNotSupported.Error() {
			t.Fatalf("question %q sans film.replay_artifact : err = %v, attendu 503", q, err)
		}
	}
}

// ─── LE FILTRE DE SPAWN VAUT POUR TOUTES LES LECTURES (revue P1-1) ─────────────

// posEtEvents pose des positions de kill et un journal des morts sur les mêmes matchs, pour
// que les lectures SQL et le KPI d'échange aient de quoi mesurer.
func posEtEvents(univ domain.TacticalUnivers) (domain.TacticalPositions, domain.TacticalKillEvents) {
	pos := domain.TacticalPositions{Univers: univ}
	ev := domain.TacticalKillEvents{Univers: univ}
	for _, m := range univ.Matchs {
		pos.Points = append(pos.Points, domain.TacticalKillPosition{
			MatchID: m.MatchID, KillerXUID: tsAdv, VictimXUID: tsMoi,
			KillerX: 9, KillerY: 9, VictimX: 2.25, VictimY: 3.25,
		})
		ev.Events = append(ev.Events,
			domain.KillEvent{MatchID: m.MatchID, VictimXUID: tsMoi, KillerXUID: tsAdv, TimeMs: 10_000},
			domain.KillEvent{MatchID: m.MatchID, VictimXUID: tsAdv, KillerXUID: tsAmi, TimeMs: 12_000},
		)
	}
	return pos, ev
}

// TestFiltreSpawn_SappliqueAuxLecturesSQL — LE DEFAUT P1-1.
//
// `{"question":"morts","spawn":"s+..."}` rendait 200 sur l'univers ENTIER sous un libelle
// de grappe : la restriction ne vivait que dans la branche des sidecars. Elle porte
// desormais sur la LISTE BLANCHE, donc sur tout ce qui en descend.
func TestFiltreSpawn_SappliqueAuxLecturesSQL(t *testing.T) {
	store, univ, ids := grappesFixture()
	pos, ev := posEtEvents(univ)
	caps := games.CapabilityMap{
		games.CapFilmReplayArtifact: games.CapSupported,
		games.CapFilmKillPositions:  games.CapSupported,
		games.CapFilmKillSource:     games.CapSupported,
	}
	repo := &mockTacticalRepo{univ: univ, pos: pos, ev: ev}
	svc := NewTacticalService(repo, caps, tsMoi).WithRasterStore(store)

	complet, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids},
	})
	if err != nil {
		t.Fatalf("lecture morts: %v", err)
	}
	if complet.MatchsFiltres != 6 {
		t.Fatalf("sans filtre : matchs_filtres = %d, attendu 6", complet.MatchsFiltres)
	}
	// Les grappes ne sont PAS calculees hors filtre sur une lecture SQL : elles couteraient
	// un chargement de sidecars que la lecture n'a pas besoin de faire.
	grappes := grappesDeLUnivers(store.sidecars, tsMoi, nil)
	if len(grappes) != 2 {
		t.Fatalf("grappes = %+v, attendu 2", grappes)
	}

	restreint, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: grappes[0].ID},
	})
	if err != nil {
		t.Fatalf("lecture morts filtree: %v", err)
	}
	if restreint.MatchsFiltres != 3 {
		t.Fatalf("matchs_filtres = %d, attendu 3 : le filtre de spawn doit valoir pour les "+
			"lectures SQL aussi", restreint.MatchsFiltres)
	}
	// LES GRAPPES SONT SERVIES AVEC, et elles restent les DEUX : la liste que la page
	// propose ne se reduit pas a la selection courante.
	if len(restreint.Grappes) != 2 {
		t.Fatalf("grappes = %+v, attendu 2 sous filtre", restreint.Grappes)
	}
	// LE KPI D'ECHANGE SUIT : son denominateur est l'univers RESTREINT.
	if restreint.Echange == nil {
		t.Fatal("l'echange n'est pas servi")
	}
	if complet.Echange.N != 6 || restreint.Echange.N != 3 {
		t.Fatalf("morts vengeables : %d sans filtre, %d avec — attendu 6 puis 3 (le KPI "+
			"recevait le scope NON restreint)", complet.Echange.N, restreint.Echange.N)
	}
}

// TestFiltreSpawn_InconnuSurUneLectureSQL — 404 typé, pas une lecture non filtrée.
func TestFiltreSpawn_InconnuSurUneLectureSQL(t *testing.T) {
	store, univ, ids := grappesFixture()
	pos, ev := posEtEvents(univ)
	repo := &mockTacticalRepo{univ: univ, pos: pos, ev: ev}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: "s+999999+999999+c01"},
	})
	if err == nil || err.Error() != domain.ErrTacticalSpawnInconnu.Error() {
		t.Fatalf("err = %v, attendu ErrTacticalSpawnInconnu — jamais une lecture non filtree", err)
	}
}

// TestFiltreSpawn_SansLecteurDArtefact — un titre qui ne produit pas d'artefact ne peut PAS
// honorer le filtre : 503, jamais un silence qui servirait l'univers entier.
func TestFiltreSpawn_SansLecteurDArtefact(t *testing.T) {
	_, univ, ids := grappesFixture()
	pos, ev := posEtEvents(univ)
	repo := &mockTacticalRepo{univ: univ, pos: pos, ev: ev}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: "s+00008+00003+c02"},
	})
	if err == nil || err.Error() != games.ErrCapabilityNotSupported.Error() {
		t.Fatalf("err = %v, attendu ErrCapabilityNotSupported", err)
	}
}

// mapsEnErreur double le lecteur d'identites de carte, toujours en echec.
type mapsEnErreur struct{}

func (mapsEnErreur) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, errors.New("shared indisponible")
}
func (mapsEnErreur) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, errors.New("shared indisponible")
}

// TestCallouts_ErreurNonAvalee — une PANNE de lecture ne doit pas se confondre avec une
// carte hors catalogue (revue P1-4).
//
// Les deux donnent des grappes muettes a l'ecran ; seul le journal les distingue. Le test
// tient ce qui est verifiable sans lire les logs : la degradation est PROPRE (aucune
// erreur remontee, la lecture reste servie) et les grappes sortent sans nom.
func TestCallouts_ErreurNonAvalee(t *testing.T) {
	store := NewTacticalCalloutsStore(t.TempDir(), "halo_infinite", mapsEnErreur{})
	if zones := store.ZonesDeLaCarte(context.Background(), "streets"); zones != nil {
		t.Fatalf("zones = %+v, attendu aucune : la lecture des identites a echoue", zones)
	}
	// Et un magasin sans lecteur de cartes degrade pareil, sans paniquer.
	muet := NewTacticalCalloutsStore(t.TempDir(), "halo_infinite", nil)
	if zones := muet.ZonesDeLaCarte(context.Background(), "streets"); zones != nil {
		t.Fatalf("zones = %+v, attendu aucune", zones)
	}
}

// TestZonesNommees_LesDeuxLanguesEtLeurRepli — LE NOM D'UN LIEU N'EST PAS UNE CHAINE
// D'INTERFACE (revue P2).
//
// Il vient du catalogue du JEU, et le client ne peut pas le traduire : n'en servir qu'une
// langue figerait la moitie des joueurs sur l'autre. Le repli est PAR LANGUE — chacune
// retombe sur le nom de CONCEPTION quand son libelle manque (le lexique FR ne couvre pas
// encore tout le vocabulaire Forge).
func TestZonesNommees_LesDeuxLanguesEtLeurRepli(t *testing.T) {
	out := zonesNommees([]replay.CalloutZone{
		{Name: "design_a", FR: "Base rouge", EN: "Red base", X: 1, Y: 2},
		{Name: "design_b", EN: "Ramp", X: 3, Y: 4},  // FR manquant
		{Name: "design_c", FR: "Rampe", X: 5, Y: 6}, // EN manquant
		{Name: "design_d", X: 7, Y: 8},              // les deux manquants
		{X: 9, Y: 10},                               // muette des trois cotes
	})
	if len(out) != 4 {
		t.Fatalf("zones = %+v, attendu 4 : la zone muette des TROIS cotes est ecartee", out)
	}
	if out[0].NomFR != "Base rouge" || out[0].NomEN != "Red base" {
		t.Fatalf("zone complete = %+v", out[0])
	}
	if out[1].NomFR != "design_b" || out[1].NomEN != "Ramp" {
		t.Fatalf("FR manquant = %+v, attendu le repli sur le nom de conception POUR LE FR SEUL",
			out[1])
	}
	if out[2].NomFR != "Rampe" || out[2].NomEN != "design_c" {
		t.Fatalf("EN manquant = %+v, attendu le repli pour l'EN SEUL", out[2])
	}
	if out[3].NomFR != "design_d" || out[3].NomEN != "design_d" {
		t.Fatalf("les deux manquants = %+v, attendu le nom de conception des deux cotes", out[3])
	}
}
