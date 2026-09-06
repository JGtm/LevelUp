// Package service — tactical_service_test.go : tests TacticalService.
//
// Le port est double par un mock : ce qui se verifie ici est l'ORCHESTRATION
// (quel point part sur quel axe, quel univers sert de denominateur, quelle porte
// de capability produit quel effet), pas le SQL — celui-la a ses propres tests
// sur `:memory:` (platform/duckdb/tactical_repo_test.go).
package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

const (
	tsMoi  = "xuid(1)"
	tsAmi  = "xuid(2)"
	tsAdv  = "xuid(3)"
	tsAdv2 = "xuid(4)"

	tsCarte = "map_streets"
)

// mockTacticalRepo double port.TacticalRepository et retient ce qu'on lui demande.
type mockTacticalRepo struct {
	maps    []domain.TacticalMapRow
	pos     domain.TacticalPositions
	ev      domain.TacticalKillEvents
	errMaps error
	errPos  error
	errEv   error

	vuPos domain.TacticalQuery
	vuEv  domain.TacticalQuery
}

func (m *mockTacticalRepo) MapsPlayed(context.Context, domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	return m.maps, m.errMaps
}

func (m *mockTacticalRepo) KillPositions(_ context.Context, q domain.TacticalQuery) (domain.TacticalPositions, error) {
	m.vuPos = q
	return m.pos, m.errPos
}

func (m *mockTacticalRepo) KillEvents(_ context.Context, q domain.TacticalQuery) (domain.TacticalKillEvents, error) {
	m.vuEv = q
	return m.ev, m.errEv
}

// capsCompletes : le titre mesure les positions ET la source des morts.
func capsCompletes() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapFilmKillPositions: games.CapSupported,
		games.CapFilmKillSource:    games.CapSupported,
	}
}

// capsPositionsSeules : positions mesurees, source des morts NON — le KPI
// d'echange doit alors etre silencieux, pas nul.
func capsPositionsSeules() games.CapabilityMap {
	return games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported}
}

// universUnMatch : un match, une composition (moi + ami contre deux adversaires).
func universUnMatch(matchID string, outcome int) domain.TacticalUnivers {
	return domain.TacticalUnivers{
		Matchs: []domain.TacticalMatch{{MatchID: matchID, Outcome: outcome}},
		Equipes: domain.EquipesParMatch{matchID: {
			tsMoi: 0, tsAmi: 0, tsAdv: 1, tsAdv2: 1,
		}},
	}
}

// celluleEn retrouve la cellule couvrant (x, y) dans une lecture (grille 0,5 m).
func celluleEn(cellules []domain.CelluleTactique, x, y float64) *domain.CelluleTactique {
	col, lig := int(math.Floor(x/0.5)), int(math.Floor(y/0.5))
	for i := range cellules {
		if cellules[i].Col == col && cellules[i].Lig == lig {
			return &cellules[i]
		}
	}
	return nil
}

// ─── LES TROIS QUESTIONS ───────────────────────────────────────────────────────

// TestTacticalService_TroisQuestions_TroisProjections : la meme mort mesuree part
// sur la position de la VICTIME pour « morts », du TUEUR pour « kills », et sur
// LES DEUX pour « gagne » (l'engagement a deux faces).
//
// Le corpus : trois matchs (pour passer le plancher de 3 matchs distincts par
// cellule), chacun avec la meme mort — je tue un adversaire en (2,2), il tombe
// en (6,6) — plus une mort ou c'est moi qui tombe, en (10,10).
func TestTacticalService_TroisQuestions_TroisProjections(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		repo.pos.Points = append(repo.pos.Points,
			// je tue (position tueur (2,2), victime (6,6))
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsMoi, VictimXUID: tsAdv,
				KillerX: 2.0, KillerY: 2.0, VictimX: 6.0, VictimY: 6.0},
			// je meurs (position tueur (14,14), victime — moi — (10,10))
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
				KillerX: 14.0, KillerY: 14.0, VictimX: 10.0, VictimY: 10.0},
		)
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	morts, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster(morts): %v", err)
	}
	if len(morts.Cellules) != 1 || celluleEn(morts.Cellules, 10.0, 10.0) == nil {
		t.Fatalf("morts : attendu la SEULE cellule (10,10) — la ou JE tombe : %+v", morts.Cellules)
	}

	kills, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionKills, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster(kills): %v", err)
	}
	if len(kills.Cellules) != 1 || celluleEn(kills.Cellules, 2.0, 2.0) == nil {
		t.Fatalf("kills : attendu la SEULE cellule (2,2) — la ou JE tire : %+v", kills.Cellules)
	}

	gagne, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster(gagne): %v", err)
	}
	// Trois victoires, zero defaite : la lecture SIGNEE exige les deux cotes.
	if len(gagne.Cellules) != 0 {
		t.Errorf("gagne sans aucune defaite : plancher par cote non applique, %+v", gagne.Cellules)
	}
	if !gagne.Echelle.Symetrique {
		t.Error("gagne : l'echelle doit etre symetrique")
	}
}

// TestTacticalService_AxeQui : « escouade » = mes coequipiers DU MATCH, moi exclu ;
// « adv » = l'autre equipe.
func TestTacticalService_AxeQui(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		repo.pos.Points = append(repo.pos.Points,
			// mon ami tombe en (4,4) ; moi je tombe en (10,10) ; un adversaire en (20,20).
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: tsAmi,
				KillerX: 1.0, KillerY: 1.0, VictimX: 4.0, VictimY: 4.0},
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
				KillerX: 1.0, KillerY: 1.0, VictimX: 10.0, VictimY: 10.0},
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsMoi, VictimXUID: tsAdv2,
				KillerX: 1.0, KillerY: 1.0, VictimX: 20.0, VictimY: 20.0},
		)
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	esc, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiEscouade, nil)
	if err != nil {
		t.Fatalf("Raster(escouade): %v", err)
	}
	if len(esc.Cellules) != 1 || celluleEn(esc.Cellules, 4.0, 4.0) == nil {
		t.Fatalf("escouade : attendu la seule cellule (4,4) — mon ami, moi EXCLU : %+v", esc.Cellules)
	}

	adv, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiAdversaires, nil)
	if err != nil {
		t.Fatalf("Raster(adv): %v", err)
	}
	if len(adv.Cellules) != 1 || celluleEn(adv.Cellules, 20.0, 20.0) == nil {
		t.Fatalf("adv : attendu la seule cellule (20,20) : %+v", adv.Cellules)
	}
}

// ─── L'UNIVERS ─────────────────────────────────────────────────────────────────

// TestTacticalService_GagneCelluleNeutre : L'EXEMPLE DE REFERENCE du chantier.
//
// 12 victoires (dont 2 SANS aucune position) et 8 defaites. Une cellule vue dans
// 6 victoires et 4 defaites est NEUTRE : 6/12 - 4/8 = 0,00. Un denominateur qui
// oublierait les matchs muets lirait 6/10 - 4/8 = +0,10 et peindrait une zone
// gagnante qui n'existe pas.
func TestTacticalService_GagneCelluleNeutre(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}

	ajouter := func(id string, outcome int, avecPoint bool) {
		u := universUnMatch(id, outcome)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		if avecPoint {
			repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
				MatchID: id, KillerXUID: tsMoi, VictimXUID: tsAdv,
				KillerX: 2.1, KillerY: 2.1, VictimX: 30.0, VictimY: 30.0,
			})
		}
	}
	for i := 1; i <= 12; i++ {
		ajouter("w"+string(rune('a'+i-1)), domain.OutcomeWin, i <= 6) // 6 avec point, 6 muettes
	}
	for i := 1; i <= 8; i++ {
		ajouter("l"+string(rune('a'+i-1)), domain.OutcomeLoss, i <= 4) // 4 avec point, 4 muettes
	}

	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	got, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster(gagne): %v", err)
	}
	if got.MatchsRetenus != 20 {
		t.Fatalf("MatchsRetenus = %d, want 20 (les 12 V et 8 D, muettes comprises)", got.MatchsRetenus)
	}
	c := celluleEn(got.Cellules, 2.1, 2.1)
	if c == nil {
		t.Fatalf("cellule (2.1, 2.1) absente : %+v", got.Cellules)
	}
	if math.Abs(c.Valeur) > 1e-9 {
		t.Errorf("Valeur = %v, want 0,00 exactement — 6/12 - 4/8. "+
			"+0,10 signifierait que les matchs MUETS sont sortis du denominateur", c.Valeur)
	}
	if c.MatchsVictoire != 6 || c.MatchsDefaite != 4 {
		t.Errorf("cotes = %d V / %d D, want 6/4", c.MatchsVictoire, c.MatchsDefaite)
	}
}

// TestTacticalService_UniversVide_CarteInconnue : aucune carte de ce nom sous ce
// filtre — 404, jamais une lecture vide qui se lirait comme « rien ne s'y passe ».
func TestTacticalService_UniversVide_CarteInconnue(t *testing.T) {
	svc := NewTacticalService(&mockTacticalRepo{}, capsCompletes(), tsMoi)
	_, err := svc.Raster(context.Background(), "map_absente", domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil)
	if !errors.Is(err, domain.ErrTacticalCarteInconnue) {
		t.Errorf("err = %v, want ErrTacticalCarteInconnue", err)
	}
}

// TestTacticalService_FiltreTransmis : le filtre de l'Explorateur descend TEL QUEL
// au lecteur — le service n'en invente ni n'en retire aucun axe.
func TestTacticalService_FiltreTransmis(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	gagne := "win"
	spec := &domain.MatchFilterSpec{Outcome: &gagne, PlaylistNames: []string{"Ranked Arena"}}

	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	if _, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionKills, domain.TacticalQuiMoi, spec); err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if repo.vuPos.Filtre != spec || repo.vuPos.MapID != tsCarte || repo.vuPos.PlayerXUID != tsMoi {
		t.Errorf("demande transmise = %+v, want carte/joueur/filtre inchanges", repo.vuPos)
	}
	if repo.vuEv.Filtre != spec || repo.vuEv.MapID != tsCarte {
		t.Errorf("demande d'echange = %+v, want le MEME filtre que les positions", repo.vuEv)
	}
}

// ─── LE KPI D'ECHANGE ──────────────────────────────────────────────────────────

// TestTacticalService_Echange_EchantillonFaible : le KPI est une domain.Couverture
// (jamais un taux nu), et sous 30 morts vengeables il porte le drapeau
// d'echantillon faible — il se lit, il ne classe personne.
func TestTacticalService_Echange_EchantillonFaible(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	repo.ev = domain.TacticalKillEvents{Univers: universUnMatch("m1", domain.OutcomeWin)}
	// Deux morts de MON camp : la mienne (vengee par l'ami en 3 s) et celle de
	// l'ami (non vengee). Plus une mort adverse, qui ne doit PAS entrer au
	// denominateur de ma couverture.
	repo.ev.Events = []domain.KillEvent{
		{MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi, TimeMs: 1000},
		{MatchID: "m1", KillerXUID: tsAmi, VictimXUID: tsAdv, TimeMs: 4000},
		{MatchID: "m1", KillerXUID: tsAdv2, VictimXUID: tsAmi, TimeMs: 20000},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	got, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if got.Echange == nil {
		t.Fatal("Echange nil alors que film.kill_source est declaree")
	}
	if got.Echange.N != 2 || got.Echange.Brut != 1 {
		t.Errorf("couverture = %d/%d, want 1 vengee sur 2 vengeables DE MON CAMP "+
			"(la mort de l'adversaire n'y entre pas)", got.Echange.Brut, got.Echange.N)
	}
	if math.Abs(got.Echange.Taux-0.5) > 1e-9 {
		t.Errorf("Taux = %v, want 0,5 (unite 0..1, jamais un pourcentage)", got.Echange.Taux)
	}
	if !got.Echange.EchantillonFaible {
		t.Errorf("2 morts vengeables (< %d) : echantillon faible attendu", coordination.SeuilEchantillonFaible)
	}
	if got.Echange.ParMatch != 1.0 {
		t.Errorf("ParMatch = %v, want 1,0 (1 vengee sur 1 match retenu)", got.Echange.ParMatch)
	}
}

// TestTacticalService_SansKillSource_EchangeSilencieux : le titre mesure les
// positions mais pas la source des morts — la lecture de placement est servie, le
// KPI est ABSENT. Un zero se lirait comme une contre-performance.
func TestTacticalService_SansKillSource_EchangeSilencieux(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	got, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if got.Echange != nil {
		t.Errorf("Echange = %+v, want nil (film.kill_source absente)", got.Echange)
	}
	if repo.vuEv.MapID != "" {
		t.Error("le journal des morts ne devait meme pas etre demande")
	}
}

// TestTacticalService_EchangeEnEchec_LectureServie : une panne du journal des
// morts est journalisee et degrade le SEUL KPI — elle n'emporte pas la lecture de
// placement avec elle.
func TestTacticalService_EchangeEnEchec_LectureServie(t *testing.T) {
	repo := &mockTacticalRepo{
		pos:   domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)},
		errEv: errors.New("journal illisible"),
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	got, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil)
	if err != nil {
		t.Fatalf("Raster: %v, want la lecture servie malgre l'echec du KPI", err)
	}
	if got.Echange != nil {
		t.Errorf("Echange = %+v, want nil", got.Echange)
	}
	if got.MatchsRetenus != 1 {
		t.Errorf("MatchsRetenus = %d, want 1", got.MatchsRetenus)
	}
}

// ─── LES PORTES ────────────────────────────────────────────────────────────────

// TestTacticalService_SansKillPositions_Capability : sans positions mesurees il
// n'y a pas de lecture du tout — ErrCapabilityNotSupported (503 propre).
func TestTacticalService_SansKillPositions_Capability(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	for nom, caps := range map[string]games.CapabilityMap{
		"map vide":              {},
		"map nil":               nil,
		"kill_source seule":     {games.CapFilmKillSource: games.CapSupported},
		"positions non exposee": {games.CapFilmKillPositions: games.CapNotExposed},
	} {
		svc := NewTacticalService(repo, caps, tsMoi)
		_, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil)
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			t.Errorf("%s: err = %v, want ErrCapabilityNotSupported", nom, err)
		}
	}
}

// TestTacticalService_VocabulaireRefuse : question ou axe hors vocabulaire — refus
// TYPE, avant toute lecture de base.
func TestTacticalService_VocabulaireRefuse(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	if _, err := svc.Raster(context.Background(), tsCarte, "temps", domain.TacticalQuiMoi, nil); !errors.Is(err, domain.ErrTacticalQuestionInconnue) {
		t.Errorf("question inconnue: err = %v, want ErrTacticalQuestionInconnue", err)
	}
	if _, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, "tout-le-monde", nil); !errors.Is(err, domain.ErrTacticalQuiInconnu) {
		t.Errorf("axe inconnu: err = %v, want ErrTacticalQuiInconnu", err)
	}
	if repo.vuPos.MapID != "" {
		t.Error("un refus de vocabulaire ne doit ouvrir aucune lecture de base")
	}
}

// ─── L'ECRAN D'ENTREE ──────────────────────────────────────────────────────────

// TestTacticalService_MapsPlayed_Plancher : le drapeau « sous le plancher » est
// pose ICI (regle produit), pas dans la requete SQL.
func TestTacticalService_MapsPlayed_Plancher(t *testing.T) {
	repo := &mockTacticalRepo{maps: []domain.TacticalMapRow{
		{MapID: "a", MapName: "Aquarius", Matchs: domain.PlancherMatchsParCarte, Victoires: 6, Defaites: 4},
		{MapID: "b", MapName: "Bazaar", Matchs: domain.PlancherMatchsParCarte - 1, Victoires: 5, Defaites: 4},
	}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	page, err := svc.MapsPlayed(context.Background(), nil)
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if page.PlancherMatchs != domain.PlancherMatchsParCarte {
		t.Errorf("PlancherMatchs = %d, want %d", page.PlancherMatchs, domain.PlancherMatchsParCarte)
	}
	if len(page.Cartes) != 2 {
		t.Fatalf("cartes = %d, want 2", len(page.Cartes))
	}
	if page.Cartes[0].SousPlancher {
		t.Errorf("carte a %d matchs : le plancher est ATTEINT, pas franchi", page.Cartes[0].Matchs)
	}
	if !page.Cartes[1].SousPlancher {
		t.Errorf("carte a %d matchs : sous le plancher attendu", page.Cartes[1].Matchs)
	}
	if page.Cartes[0].Victoires != 6 || page.Cartes[0].Defaites != 4 {
		t.Errorf("V/D non transmis : %+v", page.Cartes[0])
	}
}

// TestTacticalService_SansLecteur : un titre sans lecteur cable degrade en
// capability absente, jamais en panique.
func TestTacticalService_SansLecteur(t *testing.T) {
	svc := NewTacticalService(nil, capsCompletes(), tsMoi)
	if _, err := svc.MapsPlayed(context.Background(), nil); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("MapsPlayed sans lecteur: err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := svc.Raster(context.Background(), tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, nil); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("Raster sans lecteur: err = %v, want ErrCapabilityNotSupported", err)
	}
}
