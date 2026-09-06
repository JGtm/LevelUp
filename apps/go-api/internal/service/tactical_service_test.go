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
	"fmt"
	"math"
	"testing"

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

	vuMaps domain.TacticalQuery
	vuPos  domain.TacticalQuery
	vuEv   domain.TacticalQuery
}

func (m *mockTacticalRepo) MapsPlayed(_ context.Context, q domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	m.vuMaps = q
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

// tsDemande assemble une demande de lecture. Le PERIMETRE (liste blanche) est vide
// par defaut : le mock du port sert un jeu pose a la main et ne le relit pas, mais il
// traverse le service et decide de `MatchsFiltres` — les deux tests qui l'affirment
// posent donc leurs identifiants explicitement.
func tsDemande(carte, question, qui string, coequipiers ...string) domain.TacticalRasterRequest {
	return domain.TacticalRasterRequest{
		MapID: carte, Question: question, Qui: qui,
		Scope: domain.TacticalScope{Coequipiers: coequipiers},
	}
}

// capsCompletes : profil Halo Infinite — positions capturees du film ET source du
// degat fatal.
func capsCompletes() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapFilmKillPositions: games.CapSupported,
		games.CapFilmKillSource:    games.CapSupported,
	}
}

// capsPositionsSeules : positions lisibles, journal des morts NON exploitable — le
// KPI d'echange doit alors etre silencieux, pas nul.
func capsPositionsSeules() games.CapabilityMap {
	return games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported}
}

// universUnMatch : un match MESURE (journal des morts lisible), une composition (moi +
// ami contre deux adversaires).
//
// `Mesure: true` depuis la correction G2 (2026-09-06) : seuls les matchs mesures entrent
// au denominateur de la lecture. Tous les matchs de reference de ces tests le sont — un
// match mesure SANS kill a moi reste au denominateur, et c'est justement ce que
// l'exemple de reference 12 V / 8 D verifie.
func universUnMatch(matchID string, outcome int) domain.TacticalUnivers {
	return domain.TacticalUnivers{
		Matchs: []domain.TacticalMatch{{MatchID: matchID, Outcome: outcome, Mesure: true}},
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

	morts, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster(morts): %v", err)
	}
	if len(morts.Cellules) != 1 || celluleEn(morts.Cellules, 10.0, 10.0) == nil {
		t.Fatalf("morts : attendu la SEULE cellule (10,10) — la ou JE tombe : %+v", morts.Cellules)
	}

	kills, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionKills, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster(kills): %v", err)
	}
	if len(kills.Cellules) != 1 || celluleEn(kills.Cellules, 2.0, 2.0) == nil {
		t.Fatalf("kills : attendu la SEULE cellule (2,2) — la ou JE tire : %+v", kills.Cellules)
	}
	// T8 : les lectures NON signees ont une echelle a un seul cote. Une echelle
	// symetrique y peindrait un demi-intervalle mort.
	for nom, lecture := range map[string]domain.TacticalRaster{"morts": morts, "kills": kills} {
		if lecture.Echelle.Symetrique {
			t.Errorf("%s : Echelle.Symetrique = true, want false (lecture non signee)", nom)
		}
		if lecture.MatchsVictoire != 0 || lecture.MatchsDefaite != 0 {
			t.Errorf("%s : les deux cotes n'ont aucun sens hors lecture signee : %d/%d",
				nom, lecture.MatchsVictoire, lecture.MatchsDefaite)
		}
	}

	gagne, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionGagne, domain.TacticalQuiMoi))
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

	esc, err := svc.Raster(context.Background(),
		tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiEscouade, tsAmi))
	if err != nil {
		t.Fatalf("Raster(escouade): %v", err)
	}
	if len(esc.Cellules) != 1 || celluleEn(esc.Cellules, 4.0, 4.0) == nil {
		t.Fatalf("escouade : attendu la seule cellule (4,4) — mon ami, moi EXCLU : %+v", esc.Cellules)
	}

	adv, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiAdversaires))
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

	var ids []string
	ajouter := func(id string, outcome int, avecPoint bool) {
		ids = append(ids, id)
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
	req := tsDemande(tsCarte, domain.TacticalQuestionGagne, domain.TacticalQuiMoi)
	req.Scope.MatchIDs = ids // les 20 matchs du filtre : c'est eux que MatchsFiltres compte
	got, err := svc.Raster(context.Background(), req)
	if err != nil {
		t.Fatalf("Raster(gagne): %v", err)
	}
	// Les 20 matchs sont MESURES (leur journal est lisible) ; 10 d'entre eux n'ont
	// simplement aucune position DE MOI. C'est bien le zero LEGITIME de la phase 1, a
	// ne pas confondre avec le match ILLISIBLE de la correction G2 : celui-la sort du
	// denominateur, celui-ci y reste.
	if got.MatchsRetenus != 20 || got.MatchsFiltres != 20 {
		t.Fatalf("MatchsRetenus/MatchsFiltres = %d/%d, want 20/20 (les 12 V et 8 D, muettes comprises)",
			got.MatchsRetenus, got.MatchsFiltres)
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
		t.Errorf("cotes de la CELLULE = %d V / %d D, want 6/4", c.MatchsVictoire, c.MatchsDefaite)
	}
	// T10 : les deux DENOMINATEURS de la lecture signee sont publies au niveau du
	// raster — ce ne sont pas MatchsRetenus, et c'est tout l'objet de leur presence.
	if got.MatchsVictoire != 12 || got.MatchsDefaite != 8 {
		t.Errorf("denominateurs du raster = %d V / %d D, want 12/8 (l'univers, pas la cellule)",
			got.MatchsVictoire, got.MatchsDefaite)
	}
}

// TestTacticalService_GagneFaceVictime : « ou je gagne » regarde LES DEUX faces de
// l'engagement. Ce corpus n'alimente QUE la face victime — je tombe toujours au
// meme endroit, et mes kills sont disperses hors plancher. La cellule doit etre
// peinte quand meme.
//
// (Inversion : `prendVictime := question == Morts` dans facesDeLaQuestion fait
// tomber ce test — la face victime disparaitrait de « gagne » sans que rien d'autre
// ne bouge.)
func TestTacticalService_GagneFaceVictime(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	ajouter := func(id string, outcome int, killX float64) {
		u := universUnMatch(id, outcome)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		// Je TOMBE toujours en (7,7) — la face victime, celle qui doit peindre.
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
			KillerX: 1.0, KillerY: 1.0, VictimX: 7.0, VictimY: 7.0,
		})
		// Et je TUE a un endroit DIFFERENT dans chaque match : aucune cellule de la
		// face tueur n'atteint le plancher de 3 matchs distincts.
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsMoi, VictimXUID: tsAdv,
			KillerX: killX, KillerY: 50.0, VictimX: 60.0, VictimY: 60.0,
		})
	}
	for i, id := range []string{"w1", "w2", "w3"} {
		ajouter(id, domain.OutcomeWin, 100.0+float64(i)*10)
	}
	for i, id := range []string{"l1", "l2", "l3"} {
		ajouter(id, domain.OutcomeLoss, 200.0+float64(i)*10)
	}

	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionGagne, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster(gagne): %v", err)
	}
	c := celluleEn(got.Cellules, 7.0, 7.0)
	if c == nil {
		t.Fatalf("la cellule ou JE TOMBE est absente de « ou je gagne » — la face victime "+
			"n'est pas projetee : %+v", got.Cellules)
	}
	if c.MatchsVictoire != 3 || c.MatchsDefaite != 3 {
		t.Errorf("cotes = %d V / %d D, want 3/3", c.MatchsVictoire, c.MatchsDefaite)
	}
	if math.Abs(c.Valeur) > 1e-9 {
		t.Errorf("Valeur = %v, want 0,00 (3/3 des deux cotes)", c.Valeur)
	}
}

// TestTacticalService_JoueurHorsComposition_AucunAxe : un joueur ABSENT de la
// composition du match n'a pas d'equipe connue — il n'appartient ni a mon escouade
// ni au camp adverse, et lui en deviner une serait une invention.
//
// (Inversion : retirer la garde `!jeSuisLa || !ilEstLa` de ciblePar fait tomber ce
// test — l'inconnu tomberait dans `adv` par defaut, l'equipe absente valant 0.)
func TestTacticalService_JoueurHorsComposition_AucunAxe(t *testing.T) {
	const inconnu = "xuid(9999)"
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id] // inconnu N'Y EST PAS
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: inconnu,
			KillerX: 1.0, KillerY: 1.0, VictimX: 12.0, VictimY: 12.0,
		})
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	for _, qui := range []string{domain.TacticalQuiEscouade, domain.TacticalQuiAdversaires} {
		got, err := svc.Raster(context.Background(),
			tsDemande(tsCarte, domain.TacticalQuestionMorts, qui, tsAmi))
		if err != nil {
			t.Fatalf("Raster(%s): %v", qui, err)
		}
		if len(got.Cellules) != 0 {
			t.Errorf("qui=%s : un joueur hors composition a ete range dans un axe : %+v", qui, got.Cellules)
		}
	}
}

// TestTacticalService_CouvertureDeLocalisation : le pied de carte doit pouvoir dire
// « N morts, M localisees ». Corpus : trois morts au journal, deux seulement avec
// une position mesuree.
func TestTacticalService_CouvertureDeLocalisation(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		// Le journal porte MA mort dans les trois matchs...
		repo.ev.Events = append(repo.ev.Events, domain.KillEvent{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi, TimeMs: 1000,
		})
	}
	repo.ev.Univers = repo.pos.Univers
	// ... mais seuls deux d'entre eux ont une position mesuree.
	for _, id := range []string{"m1", "m2"} {
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
			KillerX: 1.0, KillerY: 1.0, VictimX: 9.0, VictimY: 9.0,
		})
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if got.EvenementsJournal != 3 || got.EvenementsLocalises != 2 {
		t.Errorf("couverture = %d journal / %d localises, want 3/2",
			got.EvenementsJournal, got.EvenementsLocalises)
	}
	// La cellule reste SOUS le plancher de 3 matchs distincts : la carte est muette,
	// et c'est justement pour cela que la couverture doit etre publiee.
	if len(got.Cellules) != 0 {
		t.Errorf("cellules = %+v, want 0 (deux matchs distincts, plancher a trois)", got.Cellules)
	}
}

// TestTacticalService_UniversVide_CarteInconnue : aucune carte de ce nom sous ce
// filtre — 404, jamais une lecture vide qui se lirait comme « rien ne s'y passe ».
func TestTacticalService_UniversVide_CarteInconnue(t *testing.T) {
	svc := NewTacticalService(&mockTacticalRepo{}, capsCompletes(), tsMoi)
	_, err := svc.Raster(context.Background(), tsDemande("map_absente", domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if !errors.Is(err, domain.ErrTacticalCarteInconnue) {
		t.Errorf("err = %v, want ErrTacticalCarteInconnue", err)
	}
}

// TestTacticalService_VocabulaireRefuse : question ou axe hors vocabulaire — refus
// TYPE, avant toute lecture de base.
func TestTacticalService_VocabulaireRefuse(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	if _, err := svc.Raster(context.Background(), tsDemande(tsCarte, "temps", domain.TacticalQuiMoi)); !errors.Is(err, domain.ErrTacticalQuestionInconnue) {
		t.Errorf("question inconnue: err = %v, want ErrTacticalQuestionInconnue", err)
	}
	if _, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, "tout-le-monde")); !errors.Is(err, domain.ErrTacticalQuiInconnu) {
		t.Errorf("axe inconnu: err = %v, want ErrTacticalQuiInconnu", err)
	}
	if repo.vuPos.MapID != "" {
		t.Error("un refus de vocabulaire ne doit ouvrir aucune lecture de base")
	}
}

// TestTacticalService_MatchNonMesure_HorsDenominateur — CORRECTION G2 (2026-09-06).
//
// Un match RETENU PAR LE FILTRE dont le journal des morts n'a jamais ete lu (film
// jamais decode, ou EXPIRE cote serveur) ne peut alimenter aucune cellule. Le laisser
// au denominateur ferait varier l'intensite avec la COUVERTURE DE FILM au lieu du jeu :
// le meme joueur, le meme placement, deux filtres a couverture differente, deux
// intensites.
//
// Il reste publie a part — MatchsFiltres — pour que le pied de carte puisse dire
// « N mesures sur M » plutot que de faire disparaitre en silence ce que le joueur a
// joue.
func TestTacticalService_MatchNonMesure_HorsDenominateur(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	// Dix matchs retenus par le filtre ; TROIS seulement portent un journal lisible
	// (trois, c'est le plancher de rarete par cellule : en dessous, rien ne se peint).
	var ids []string
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("m%02d", i)
		ids = append(ids, id)
		u := universUnMatch(id, domain.OutcomeWin)
		u.Matchs[0].Mesure = i < 3
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		if i < 3 {
			repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
				MatchID: id, KillerXUID: tsMoi, VictimXUID: tsAdv,
				KillerX: 2.1, KillerY: 2.1, VictimX: 30.0, VictimY: 30.0,
			})
		}
	}

	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)
	req := tsDemande(tsCarte, domain.TacticalQuestionKills, domain.TacticalQuiMoi)
	req.Scope.MatchIDs = ids
	got, err := svc.Raster(context.Background(), req)
	if err != nil {
		t.Fatalf("Raster(kills): %v", err)
	}
	if got.MatchsFiltres != 10 {
		t.Errorf("MatchsFiltres = %d, want 10 (le filtre a bien retenu 10 matchs)", got.MatchsFiltres)
	}
	if got.MatchsRetenus != 3 {
		t.Errorf("MatchsRetenus = %d, want 3 (trois journaux lisibles)", got.MatchsRetenus)
	}
	c := celluleEn(got.Cellules, 2.1, 2.1)
	if c == nil {
		t.Fatalf("cellule (2.1, 2.1) absente : %+v", got.Cellules)
	}
	// 3 occurrences sur 3 matchs MESURES = 1,00. Sur les 10 matchs du filtre, on
	// lirait 0,30 — une intensite trois fois plus faible pour exactement le meme jeu.
	if c.Valeur != 1 {
		t.Errorf("valeur = %v, want 1 (3 occurrences / 3 matchs MESURES, pas / 10 filtres)", c.Valeur)
	}
}
