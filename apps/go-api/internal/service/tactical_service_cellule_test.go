package service

// tactical_service_cellule_test.go — LE DETAIL D'UNE CELLULE (lien « voir dans le rejeu »,
// Tactique S.1, lot M1).
//
// Ce qui se verifie ici, et ce que chaque cas attrape :
//
//	le filtre de cellule    seuls les points qui tombent dans (col, lig) sortent ;
//	les trois faces         morts (victime), kills (tueur), gagne (les deux) ;
//	isole                   memes exclusions que rasterIsole (rayon, equipe a terre) ;
//	temps / routes          l'instant vient du sidecar (frame x FrameIntervalMs) ;
//	OWNERSHIP (ADR 0029)    un match d'un autre joueur n'apparait dans AUCUNE
//	                        contribution mais est COMPTE dans MatchsNonOuvrables ;
//	le tri                  date de match decroissante puis instant croissant.
import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// celluleDemande construit une demande de detail de cellule sur le perimetre du double.
func celluleDemande(repo *mockTacticalRepo, question, qui string, col, lig int, coequipiers ...string) domain.TacticalCelluleRequest {
	return domain.TacticalCelluleRequest{
		MapID: tsCarte, Question: question, Qui: qui, Col: col, Lig: lig,
		Scope: domain.TacticalScope{MatchIDs: tsPerimetreDu(repo), Coequipiers: coequipiers},
	}
}

// TestCellule_Ownership_MatchEtrangerCompteSansApparaitre : LE TEST DU LOT. m1 est ouvrable
// (a moi), m2 ne l'est pas (un autre joueur) — les deux ont une mort dans la MEME cellule.
// La contribution de m2 doit disparaitre des contributions ET etre comptee.
func TestCellule_Ownership_MatchEtrangerCompteSansApparaitre(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
	}
	// Je meurs en (2,1)/(2,2) dans les deux matchs — meme cellule (col=4, lig=4).
	repo.pos.Points = []domain.TacticalKillPosition{
		{MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi, KillerX: 9, KillerY: 9, VictimX: 2.1, VictimY: 2.1, TimeMs: 1000},
		{MatchID: "m2", KillerXUID: tsAdv, VictimXUID: tsMoi, KillerX: 9, KillerY: 9, VictimX: 2.2, VictimY: 2.2, TimeMs: 2000},
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	// m2 est ABSENT de la map d'ouvrabilite : c'est un match d'un autre joueur.
	repo.ouvrables = map[string]time.Time{"m1": base}

	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, 4, 4))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].MatchID != "m1" {
		t.Fatalf("contributions = %+v, want une seule (m1)", got.Contributions)
	}
	if got.MatchsNonOuvrables != 1 {
		t.Errorf("matchs_non_ouvrables = %d, want 1 (m2)", got.MatchsNonOuvrables)
	}
	if repo.vuOuvrXUID != tsMoi {
		t.Errorf("MatchsOuvrables appele avec xuid=%q, want %q", repo.vuOuvrXUID, tsMoi)
	}
}

// TestCellule_Morts_NeGardeQueLaCelluleDemandee : un point hors de la cellule demandee
// n'apparait pas, meme s'il appartient a un match ouvrable et a la bonne cible.
func TestCellule_Morts_NeGardeQueLaCelluleDemandee(t *testing.T) {
	repo := &mockTacticalRepo{}
	u := universUnMatch("m1", domain.OutcomeWin)
	repo.pos.Univers = u
	repo.pos.Points = []domain.TacticalKillPosition{
		{MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi, KillerX: 9, KillerY: 9, VictimX: 2.0, VictimY: 2.0, TimeMs: 1000},
		{MatchID: "m1", KillerXUID: tsAdv2, VictimXUID: tsMoi, KillerX: 20, KillerY: 20, VictimX: 20.0, VictimY: 20.0, TimeMs: 5000},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, 4, 4))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].InstantMs != 1000 {
		t.Fatalf("contributions = %+v, want une seule a l'instant 1000", got.Contributions)
	}
	if got.Contributions[0].XUID != tsMoi {
		t.Errorf("xuid = %q, want %q (la victime, question morts)", got.Contributions[0].XUID, tsMoi)
	}
}

// TestCellule_Kills_ProjectionTueur : « kills » attribue la contribution au TUEUR, a sa
// propre position — jamais celle de la victime.
func TestCellule_Kills_ProjectionTueur(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = universUnMatch("m1", domain.OutcomeWin)
	repo.pos.Points = []domain.TacticalKillPosition{
		{MatchID: "m1", KillerXUID: tsMoi, VictimXUID: tsAdv, KillerX: 2.0, KillerY: 2.0, VictimX: 20.0, VictimY: 20.0, TimeMs: 4200},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionKills, domain.TacticalQuiMoi, 4, 4))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].XUID != tsMoi || got.Contributions[0].InstantMs != 4200 {
		t.Fatalf("contributions = %+v, want moi a l'instant 4200 (position du tueur)", got.Contributions)
	}
}

// TestCellule_Gagne_LesDeuxFaces : « gagne » regarde l'engagement en entier — la victime
// ET le tueur, chacun dans sa propre cellule.
func TestCellule_Gagne_LesDeuxFaces(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = universUnMatch("m1", domain.OutcomeWin)
	repo.pos.Points = []domain.TacticalKillPosition{
		// je tue en (2,2), il meurt en (20,20) : deux cellules distinctes.
		{MatchID: "m1", KillerXUID: tsMoi, VictimXUID: tsAdv, KillerX: 2.0, KillerY: 2.0, VictimX: 20.0, VictimY: 20.0, TimeMs: 1000},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	gotTueur, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if err != nil {
		t.Fatalf("Cellule (cote tueur): %v", err)
	}
	if len(gotTueur.Contributions) != 1 || gotTueur.Contributions[0].XUID != tsMoi {
		t.Fatalf("cote tueur = %+v, want une contribution a moi", gotTueur.Contributions)
	}

	gotVictime, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiAdversaires, 40, 40))
	if err != nil {
		t.Fatalf("Cellule (cote victime): %v", err)
	}
	if len(gotVictime.Contributions) != 1 || gotVictime.Contributions[0].XUID != tsAdv {
		t.Fatalf("cote victime = %+v, want une contribution a l'adversaire", gotVictime.Contributions)
	}
}

// TestCellule_Isole : reprend le corpus de reference de l'isolement (mort a 19 m d'un
// coequipier, ISOLEE en Arene car le rayon vaut 18 m) et verifie que le detail de cellule
// applique EXACTEMENT la meme regle.
func TestCellule_Isole(t *testing.T) {
	univ := universVariantes(map[string]string{"m1": "Slayer:Arena"})
	repo := &mockTacticalRepo{
		univ: univ,
		morts: domain.TacticalMortsContexte{Univers: univ, Morts: []domain.MortContexte{
			mortContexteAvecInstant("m1", tsMoi, 4.0, 4.0, m(19.0), 1, 0, 7000),
		}},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).
		WithRadarRange(map[string]int{"Slayer:Arena": 18})

	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionIsole, domain.TacticalQuiMoi, 8, 8))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].XUID != tsMoi || got.Contributions[0].InstantMs != 7000 {
		t.Fatalf("contributions = %+v, want une mort isolee a moi, instant 7000", got.Contributions)
	}
}

// mortContexteAvecInstant : meme fixture que `mortContexte` (tactical_service_isolement_test.go),
// avec l'instant en plus — le detail de cellule en a besoin, la lecture agregee non.
func mortContexteAvecInstant(matchID, victime string, x, y float64, proche *float64,
	visibles, horsDeVue int, instantMs int64,
) domain.MortContexte {
	mc := mortContexte(matchID, victime, x, y, proche, visibles, horsDeVue)
	mc.TimeMs = instantMs
	return mc
}

// TestCellule_Temps : l'instant contributeur est la frame de PREMIERE ENTREE, convertie en
// millisecondes par `FrameIntervalMs`.
func TestCellule_Temps(t *testing.T) {
	univ := universTroisMatchs("m1")
	repo := &mockTacticalRepo{univ: univ}
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": {
			SchemaVersion: domain.TacticalRasterSchemaVersion, MatchID: "m1", ShortID: "m1",
			PasM: domain.TacticalRasterPasM, PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
			FrameIntervalMs: 100,
			Joueurs: []domain.TacticalRasterJoueur{{
				XUID: tsMoi,
				PremieresEntrees: []domain.TacticalRasterEntree{
					{Col: 4, Lig: 6, Frame: 30},
					{Col: 40, Lig: 40, Frame: 5}, // autre cellule : ne doit pas apparaitre
				},
			}},
		},
	}}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)

	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionTemps, domain.TacticalQuiMoi, 4, 6))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].XUID != tsMoi {
		t.Fatalf("contributions = %+v, want une seule a moi", got.Contributions)
	}
	if want := int64(30 * 100); got.Contributions[0].InstantMs != want {
		t.Errorf("instant_ms = %d, want %d (30 frames x 100 ms)", got.Contributions[0].InstantMs, want)
	}
}

// TestCellule_Routes : une route qui TRAVERSE la cellule demandee contribue avec son
// DEBUT DE VIE comme instant — pas l'instant du passage dans cette cellule precise.
func TestCellule_Routes(t *testing.T) {
	univ := universTroisMatchs("m1")
	repo := &mockTacticalRepo{univ: univ}
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": {
			SchemaVersion: domain.TacticalRasterSchemaVersion, MatchID: "m1", ShortID: "m1",
			PasM: domain.TacticalRasterPasM, PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
			FrameIntervalMs: 200,
			Joueurs: []domain.TacticalRasterJoueur{{
				XUID: tsMoi,
				Routes: []domain.TacticalRasterRoute{{
					DebutFrame: 10,
					Cases:      []domain.TacticalRasterCase{{Col: 4, Lig: 6}, {Col: 5, Lig: 6}},
				}},
			}},
		},
	}}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store)

	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionRoutes, domain.TacticalQuiMoi, 5, 6))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 1 {
		t.Fatalf("contributions = %+v, want une seule (la route traverse (5,6))", got.Contributions)
	}
	if want := int64(10 * 200); got.Contributions[0].InstantMs != want {
		t.Errorf("instant_ms = %d, want %d (le DEBUT de la route, pas le passage)", got.Contributions[0].InstantMs, want)
	}
}

// TestCellule_TrieeParDateDecroissantePuisInstant : deux contributions du meme match sont
// triees par instant ; deux matchs distincts sont tries par DATE decroissante d'abord.
func TestCellule_TrieeParDateDecroissantePuisInstant(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"tot", "tard"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
	}
	repo.pos.Points = []domain.TacticalKillPosition{
		{MatchID: "tard", KillerXUID: tsAdv, VictimXUID: tsMoi, VictimX: 2.0, VictimY: 2.0, TimeMs: 2000},
		{MatchID: "tot", KillerXUID: tsAdv, VictimXUID: tsMoi, VictimX: 2.1, VictimY: 2.1, TimeMs: 500},
		{MatchID: "tot", KillerXUID: tsAdv2, VictimXUID: tsMoi, VictimX: 2.2, VictimY: 2.2, TimeMs: 100},
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	repo.ouvrables = map[string]time.Time{
		"tot":  base,
		"tard": base.Add(24 * time.Hour),
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	got, err := svc.Cellule(context.Background(), celluleDemande(repo, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, 4, 4))
	if err != nil {
		t.Fatalf("Cellule: %v", err)
	}
	if len(got.Contributions) != 3 {
		t.Fatalf("contributions = %+v, want 3", got.Contributions)
	}
	wantOrdre := []string{"tard", "tot", "tot"}
	wantInstants := []int64{2000, 100, 500}
	for i, c := range got.Contributions {
		if c.MatchID != wantOrdre[i] || c.InstantMs != wantInstants[i] {
			t.Errorf("contribution[%d] = %+v, want match=%s instant=%d", i, c, wantOrdre[i], wantInstants[i])
		}
	}
}

// TestCellule_CarteInconnue : univers vide -> le meme refus que Raster.
func TestCellule_CarteInconnue(t *testing.T) {
	repo := &mockTacticalRepo{}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	_, err := svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
		MapID: tsCarte, Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"inconnu"}},
	})
	if err != domain.ErrTacticalCarteInconnue {
		t.Fatalf("err = %v, want ErrTacticalCarteInconnue", err)
	}
}

// TestCellule_QuestionInconnue : la validation du perimetre est partagee avec Raster
// (validerLecture) — un seul test suffit a prouver le branchement.
func TestCellule_QuestionInconnue(t *testing.T) {
	repo := &mockTacticalRepo{}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	_, err := svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
		MapID: tsCarte, Question: "sourire", Qui: domain.TacticalQuiMoi,
	})
	if err == nil {
		t.Fatal("attendu un refus pour une question hors vocabulaire")
	}
}

// TestCellule_CapabiliteAbsente : pas de positions lisibles -> 503 propre, comme Raster.
func TestCellule_CapabiliteAbsente(t *testing.T) {
	repo := &mockTacticalRepo{}
	svc := NewTacticalService(repo, nil, tsMoi)
	_, err := svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
		MapID: tsCarte, Question: domain.TacticalQuestionMorts, Qui: domain.TacticalQuiMoi,
	})
	if err == nil {
		t.Fatal("attendu ErrCapabilityNotSupported")
	}
}
