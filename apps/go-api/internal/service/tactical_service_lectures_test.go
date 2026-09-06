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
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// mockCallouts double le lecteur de zones nommees.
type mockCallouts struct{ zones []domain.ZoneNommee }

func (m *mockCallouts) ZonesDeLaCarte(context.Context, string) []domain.ZoneNommee {
	return m.zones
}

// universVariantes pose des matchs avec leur variante, moi + un ami contre deux adversaires.
func universVariantes(parMatch map[string]string) domain.TacticalUnivers {
	u := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range triees(parMatch) {
		u.Matchs = append(u.Matchs, domain.TacticalMatch{
			MatchID: id, GameVariantName: parMatch[id], Outcome: domain.OutcomeWin, Mesure: true,
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

// mortAvec pose une mort d'un joueur avec ses voisins vivants (xuid -> distance).
func mortAvec(frame int, x, y float64, voisins map[string]float64) domain.TacticalRasterMort {
	m := domain.TacticalRasterMort{Frame: frame, X: x, Y: y, Voisins: []domain.TacticalRasterVoisin{}}
	for _, xuid := range []string{tsAmi, tsAdv, tsAdv2} {
		if d, ok := voisins[xuid]; ok {
			m.Voisins = append(m.Voisins, domain.TacticalRasterVoisin{XUID: xuid, DistanceM: d})
		}
	}
	return m
}

// joueurMorts pose un joueur qui n'a que des morts.
func joueurMorts(xuid string, morts ...domain.TacticalRasterMort) domain.TacticalRasterJoueur {
	return domain.TacticalRasterJoueur{XUID: xuid, Morts: morts}
}

// svcIsole monte le service avec la table des rayons d'Arene et de BTB.
func svcIsole(univ domain.TacticalUnivers, store *mockRasterStore) *TacticalService {
	repo := &mockTacticalRepo{univ: univ}
	return NewTacticalService(repo, capsOccupation(), tsMoi).
		WithRasterStore(store).
		WithRadarRange(map[string]int{"Slayer:Arena": 18, "BTB:Slayer": 24})
}

func lireIsole(t *testing.T, svc *TacticalService, ids ...string) domain.TacticalRaster {
	t.Helper()
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionIsole, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids},
	})
	if err != nil {
		t.Fatalf("lecture isole: %v", err)
	}
	return out
}

// TestIsole_RayonDeLaVarianteDuMatch — LA MEME MORT, A 19 m D'UN COEQUIPIER, est ISOLEE en
// Arene (18 m) et NE L'EST PAS en BTB (24 m).
//
// Le rayon vient de la VARIANTE du match, resolue par `regulation.toml`. Un rayon unique
// applique a tout l'univers melangerait deux regles de jeu sous une seule mesure — et un
// filtre qui contient les deux formats est le cas normal.
func TestIsole_RayonDeLaVarianteDuMatch(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"arene": sidecarPose("arene", joueurMorts(tsMoi, mortAvec(10, 2, 3, map[string]float64{tsAmi: 19}))),
		"btb":   sidecarPose("btb", joueurMorts(tsMoi, mortAvec(10, 2, 3, map[string]float64{tsAmi: 19}))),
	}}
	svc := svcIsole(universVariantes(map[string]string{
		"arene": "Slayer:Arena", "btb": "BTB:Slayer",
	}), store)
	out := lireIsole(t, svc, "arene", "btb")
	if out.Isolement == nil {
		t.Fatal("la lecture isole ne publie aucune couverture")
	}
	if out.Isolement.N != 2 {
		t.Fatalf("denominateur = %d, attendu 2 morts examinees", out.Isolement.N)
	}
	if out.Isolement.Brut != 1 {
		t.Fatalf("morts isolees = %d, attendu 1 : 19 m depasse les 18 m de l'Arene mais pas "+
			"les 24 m du BTB", out.Isolement.Brut)
	}
	if out.MatchsSansRayon != 0 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 0", out.MatchsSansRayon)
	}
}

// TestIsole_UnAdversaireProcheNAccompagnePersonne — le service JOINT LES EQUIPES : un
// adversaire a 2 m n'entre pas au calcul, seul le coequipier a 40 m compte.
//
// C'est ce que le film ne peut pas faire : il ne porte aucun camp (`Track.Team` = -1).
func TestIsole_UnAdversaireProcheNAccompagnePersonne(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurMorts(tsMoi,
			mortAvec(10, 2, 3, map[string]float64{tsAmi: 40, tsAdv: 2, tsAdv2: 1}))),
	}}
	svc := svcIsole(universVariantes(map[string]string{"m1": "Slayer:Arena"}), store)
	out := lireIsole(t, svc, "m1")
	if out.Isolement.N != 1 || out.Isolement.Brut != 1 {
		t.Fatalf("couverture = %+v, attendu 1 mort isolee sur 1 : deux adversaires a 2 m et "+
			"1 m n'accompagnent personne", out.Isolement)
	}
}

// TestIsole_TousCoequipiersMorts_ExclusDuDenominateur — une mort dont AUCUN coequipier
// n'etait vivant sort du denominateur. Ici le seul voisin vivant est un adversaire.
func TestIsole_TousCoequipiersMorts_ExclusDuDenominateur(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurMorts(tsMoi,
			mortAvec(10, 2, 3, map[string]float64{tsAdv: 5}), // aucun coequipier vivant
			mortAvec(20, 2, 3, map[string]float64{tsAmi: 40}),
		)),
	}}
	svc := svcIsole(universVariantes(map[string]string{"m1": "Slayer:Arena"}), store)
	out := lireIsole(t, svc, "m1")
	if out.Isolement.N != 1 {
		t.Fatalf("denominateur = %d, attendu 1 : la mort sans coequipier VIVANT est exclue",
			out.Isolement.N)
	}
	if out.Isolement.Brut != 1 {
		t.Fatalf("morts isolees = %d, attendu 1", out.Isolement.Brut)
	}
}

// TestIsole_VarianteSansRayon — un match dont la variante n'est pas dans la table sort de
// la lecture ET SE COMPTE. Sans ce compte, une lecture amputee ressemblerait a une lecture
// complete.
func TestIsole_VarianteSansRayon(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"connu":   sidecarPose("connu", joueurMorts(tsMoi, mortAvec(10, 2, 3, map[string]float64{tsAmi: 40}))),
		"inconnu": sidecarPose("inconnu", joueurMorts(tsMoi, mortAvec(10, 2, 3, map[string]float64{tsAmi: 40}))),
	}}
	svc := svcIsole(universVariantes(map[string]string{
		"connu": "Slayer:Arena", "inconnu": "Husky Raid:CTF",
	}), store)
	out := lireIsole(t, svc, "connu", "inconnu")
	if out.MatchsSansRayon != 1 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 1", out.MatchsSansRayon)
	}
	if out.Isolement.N != 1 || out.Isolement.Brut != 1 {
		t.Fatalf("couverture = %+v, attendu 1 sur 1 : les morts du match sans rayon ne sont "+
			"NI examinees NI comptees isolees", out.Isolement)
	}
	// Les deux matchs restent dans l'univers : c'est la LECTURE qui les ecarte, pas le
	// filtre — et `matchs_retenus` doit continuer de dire ce que les sidecars couvrent.
	if out.MatchsFiltres != 2 || out.MatchsRetenus != 2 {
		t.Fatalf("filtres=%d retenus=%d, attendu 2 et 2", out.MatchsFiltres, out.MatchsRetenus)
	}
}

// TestIsole_SansTableDeRayon — un titre dont `regulation.toml` ne declare aucune portee ne
// rend AUCUNE lecture d'isolement, et le dit.
func TestIsole_SansTableDeRayon(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurMorts(tsMoi, mortAvec(10, 2, 3, map[string]float64{tsAmi: 40}))),
	}}
	repo := &mockTacticalRepo{univ: universVariantes(map[string]string{"m1": "Slayer:Arena"})}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionIsole, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.MatchsSansRayon != 1 || out.Isolement.N != 0 || len(out.Cellules) != 0 {
		t.Fatalf("sortie = %+v : sans table, aucune mort ne doit etre examinee", out)
	}
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
	svc := svcIsole(universVariantes(map[string]string{
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
	svc := svcIsole(univ, store).WithCalloutsStore(&mockCallouts{zones: []domain.ZoneNommee{
		{Nom: "Base rouge", X: 0, Y: 0},
		{Nom: "Base bleue", X: 51, Y: 51},
	}})
	out := lireTemps(t, svc, "", ids...)
	if len(out.Grappes) != 2 {
		t.Fatalf("grappes = %+v, attendu 2", out.Grappes)
	}
	noms := map[string]bool{out.Grappes[0].Nom: true, out.Grappes[1].Nom: true}
	if !noms["Base rouge"] || !noms["Base bleue"] {
		t.Fatalf("grappes nommees %v, attendu les deux bases", noms)
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
	svc := svcIsole(univ, store)
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
	svc := svcIsole(univ, store)
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
	// Chaque match porte en plus une mort isolee.
	for id, sc := range store.sidecars {
		sc.Joueurs[0].Morts = []domain.TacticalRasterMort{
			mortAvec(10, 2, 3, map[string]float64{tsAmi: 40}),
		}
		store.sidecars[id] = sc
	}
	svc := svcIsole(univ, store)
	complet := lireTemps(t, svc, "", ids...)

	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionIsole, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: complet.Grappes[0].ID},
	})
	if err != nil {
		t.Fatalf("lecture isole filtree: %v", err)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3 sous le filtre de spawn", out.MatchsRetenus)
	}
	if out.Isolement.N != 3 {
		t.Fatalf("denominateur = %d, attendu 3 : le filtre porte aussi sur les morts",
			out.Isolement.N)
	}
}

// TestLecturesDArtefact_MemePorte — les trois lectures partagent `film.replay_artifact`.
func TestLecturesDArtefact_MemePorte(t *testing.T) {
	store, univ, ids := grappesFixture()
	repo := &mockTacticalRepo{univ: univ}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi).WithRasterStore(store)
	for _, q := range []string{
		domain.TacticalQuestionTemps, domain.TacticalQuestionRoutes, domain.TacticalQuestionIsole,
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
