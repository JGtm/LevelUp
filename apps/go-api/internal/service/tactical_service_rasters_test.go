package service

// tactical_service_rasters_test.go — LA LECTURE D'OCCUPATION, SUR DES SIDECARS POSES A LA
// MAIN.
//
// Ce qui est eprouve ici, et chaque cas dit ce qu'il attrape :
//
//	la somme               deux sidecars du meme joueur sur la meme cellule s'additionnent,
//	                       et la valeur est en SECONDES PAR MATCH (echantillons x 0,25 s
//	                       / matchs mesures) ;
//	le denominateur        un sidecar ABSENT est un match NON MESURE : il compte dans
//	                       `matchs_filtres`, pas dans `matchs_retenus`. L'inverser ferait
//	                       varier l'intensite avec la couverture de film au lieu du jeu ;
//	l'axe « qui »          il se tranche a la LECTURE, sur un sidecar anonyme ;
//	la porte               un titre sans `film.replay_artifact` -> 503 propre, jamais une
//	                       carte vide qui se lirait « il ne se passe rien ici » ;
//	le cablage             sans lecteur de sidecars injecte -> 503 ET une ligne ERROR ;
//	les unites             un sidecar d'un autre pas ou d'un autre format est ECARTE, pas
//	                       somme — melanger deux unites donnerait un resultat faux SANS
//	                       rien casser, le pire mode de defaillance.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// mockRasterStore double port.TacticalRasterStore : une table match -> sidecar, et une
// table match -> erreur. Un match absent des deux rend (nil, nil) : le cas NORMAL d'un
// film jamais decode.
type mockRasterStore struct {
	sidecars map[string]*domain.TacticalRasterSidecar
	erreurs  map[string]error
	vus      []string
}

func (m *mockRasterStore) Charger(_ context.Context, matchID string) (*domain.TacticalRasterSidecar, error) {
	m.vus = append(m.vus, matchID)
	if err := m.erreurs[matchID]; err != nil {
		return nil, err
	}
	return m.sidecars[matchID], nil
}

// sidecarPose fabrique un sidecar valide : pour chaque joueur, une cellule (col, lig) a
// `echantillons`.
func sidecarPose(matchID string, joueurs ...domain.TacticalRasterJoueur) *domain.TacticalRasterSidecar {
	return &domain.TacticalRasterSidecar{
		SchemaVersion:         domain.TacticalRasterSchemaVersion,
		MatchID:               matchID,
		ShortID:               matchID,
		ArtifactSchemaVersion: 39,
		PasM:                  tactical.PasParDefautM,
		FrameIntervalMs:       100,
		PasEchantillonMs:      tactical.PasOccupationMs,
		Joueurs:               joueurs,
	}
}

func joueurEn(xuid string, col, lig, echantillons int) domain.TacticalRasterJoueur {
	return domain.TacticalRasterJoueur{
		XUID:     xuid,
		Cellules: []domain.TacticalRasterCellule{{Col: col, Lig: lig, Echantillons: echantillons}},
	}
}

// capsOccupation : le titre produit des artefacts de rejeu (Halo Infinite).
func capsOccupation() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapFilmReplayArtifact: games.CapSupported,
		games.CapFilmKillPositions:  games.CapSupported,
	}
}

// universTroisMatchs : les matchs nommes, sur la carte, moi + un ami contre deux
// adversaires. Tous MESURES au sens du journal des morts — c'est le sidecar, et lui seul,
// qui decide de la mesure d'une lecture d'occupation.
func universTroisMatchs(ids ...string) domain.TacticalUnivers {
	u := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range ids {
		u.Matchs = append(u.Matchs, domain.TacticalMatch{MatchID: id, Outcome: domain.OutcomeWin, Mesure: true})
		u.Equipes[id] = map[string]int{tsMoi: 0, tsAmi: 0, tsAdv: 1, tsAdv2: 1}
	}
	return u
}

// occupationSvc monte le service avec un univers pose et un magasin de sidecars.
func occupationSvc(univ domain.TacticalUnivers, store *mockRasterStore) (*TacticalService, *mockTacticalRepo) {
	// Le journal des morts est servi VIDE : `capsOccupation` ne declare pas
	// `film.kill_source`, donc le KPI d'echange reste silencieux — et la couverture
	// d'evenements doit rendre 0 parce que l'occupation ne regarde aucune face d'une
	// mort, pas parce que la lecture aurait echoue.
	repo := &mockTacticalRepo{univ: univ, ev: domain.TacticalKillEvents{
		Univers: univ,
		Events:  []domain.KillEvent{{MatchID: "m1", VictimXUID: tsMoi, KillerXUID: tsAdv}},
	}}
	return NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(store), repo
}

// TestOccupation_SommeEtDenominateur — LE CAS DE REFERENCE.
//
// Quatre matchs dans le perimetre, TROIS sidecars presents, tous sur la cellule (2,3) :
// m1 a 8 echantillons, m2 a 4, m3 a 6. Le quatrieme n'a pas de sidecar : son film n'a
// jamais ete decode.
//
//	matchs_filtres  4 (l'univers de la carte)
//	matchs_retenus  3 (les mesures)
//	brut            18 echantillons (8 + 4 + 6)
//	valeur          18 x 0,25 s / 3 matchs = 1,5 s par match
//
// Diviser par 4 rendrait 1,125 s : la meme occupation paraitrait plus faible parce qu'un
// film a expire. C'est exactement le defaut que la correction G2 a ferme sur l'autre
// substrat.
//
// TROIS matchs mesures et pas deux, parce que le plancher de rarete de l'agregat en exige
// trois DISTINCTS : c'est le meme plancher que les trois autres lectures, et il
// s'applique ICI, sur la somme — jamais sur le sidecar (cf. Raster.CellulesBrutes).
func TestOccupation_SommeEtDenominateur(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8)),
		"m2": sidecarPose("m2", joueurEn(tsMoi, 2, 3, 4)),
		"m3": sidecarPose("m3", joueurEn(tsMoi, 2, 3, 6)),
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3", "m4"), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3", "m4"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.MatchsFiltres != 4 {
		t.Fatalf("matchs_filtres = %d, attendu 4 (l'univers de la carte)", out.MatchsFiltres)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3 — un sidecar absent est un match NON MESURE",
			out.MatchsRetenus)
	}
	if len(out.Cellules) != 1 {
		t.Fatalf("cellules = %+v, attendu 1", out.Cellules)
	}
	c := out.Cellules[0]
	if c.Col != 2 || c.Lig != 3 {
		t.Fatalf("cellule = (%d,%d), attendu (2,3) — les indices sont ancres sur l'origine du monde", c.Col, c.Lig)
	}
	if c.Brut != 18 {
		t.Fatalf("brut = %v, attendu 18 echantillons (8 + 4 + 6)", c.Brut)
	}
	if c.Valeur != 1.5 {
		t.Fatalf("valeur = %v s, attendu 1,5 (18 x 0,25 s / 3 matchs mesures)", c.Valeur)
	}
	if c.Matchs != 3 {
		t.Fatalf("matchs distincts = %d, attendu 3", c.Matchs)
	}
	if out.PasM != tactical.PasParDefautM {
		t.Fatalf("pas_m = %v", out.PasM)
	}
	// L'occupation ne lit AUCUN journal des morts : sa couverture est l'ecart
	// matchs_retenus / matchs_filtres, pas un compte d'evenements.
	if out.EvenementsJournal != 0 || out.EvenementsLocalises != 0 {
		t.Fatalf("couverture d'evenements = %d/%d, attendu 0/0 pour une lecture d'occupation",
			out.EvenementsLocalises, out.EvenementsJournal)
	}
	// La question et l'axe sont republies tels quels.
	if out.Question != domain.TacticalQuestionTemps || out.Qui != domain.TacticalQuiMoi {
		t.Fatalf("question/axe = %q/%q", out.Question, out.Qui)
	}
}

// TestOccupation_PlancherDeRarete — le plancher appartient a l'AGREGAT : une cellule vue
// dans deux matchs seulement ne se peint pas (trois matchs distincts exiges). C'est ce qui
// rend le stockage par match legitime — le sidecar, lui, garde tout.
func TestOccupation_PlancherDeRarete(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8), joueurEn(tsMoi+"x", 9, 9, 8)),
		"m2": sidecarPose("m2", joueurEn(tsMoi, 2, 3, 8)),
		"m3": sidecarPose("m3", joueurEn(tsMoi, 5, 5, 8)),
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3"), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3", out.MatchsRetenus)
	}
	// (2,3) : 2 matchs ; (5,5) : 1 match — aucune n'atteint le plancher de 3.
	if len(out.Cellules) != 0 {
		t.Fatalf("cellules = %+v, attendu aucune sous le plancher de %d matchs distincts",
			out.Cellules, tactical.PlancherMatchsParCellule)
	}
}

// TestOccupation_AxeAdversaires — l'axe se tranche a la LECTURE, sur un sidecar anonyme :
// c'est ce qui permet a un changement de camp de ne perimer aucun fichier.
func TestOccupation_AxeAdversaires(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8), joueurEn(tsAdv, 7, 7, 4)),
		"m2": sidecarPose("m2", joueurEn(tsMoi, 2, 3, 8), joueurEn(tsAdv, 7, 7, 4)),
		"m3": sidecarPose("m3", joueurEn(tsMoi, 2, 3, 8), joueurEn(tsAdv2, 7, 7, 4)),
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3"), store)
	req := domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiAdversaires,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	}
	out, err := svc.Raster(context.Background(), req)
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if len(out.Cellules) != 1 {
		t.Fatalf("cellules = %+v, attendu la seule cellule des adversaires", out.Cellules)
	}
	if out.Cellules[0].Col != 7 || out.Cellules[0].Lig != 7 {
		t.Fatalf("cellule = (%d,%d), attendu (7,7) : celle des adversaires, pas la mienne",
			out.Cellules[0].Col, out.Cellules[0].Lig)
	}
	if out.Cellules[0].Brut != 12 {
		t.Fatalf("brut = %v, attendu 12 (3 x 4)", out.Cellules[0].Brut)
	}
}

// TestOccupation_SansCapability — un titre qui ne produit aucun artefact rend 503, jamais
// une carte vide : « aucune cellule » et « cette mesure n'existe pas ici » sont deux
// reponses differentes.
func TestOccupation_SansCapability(t *testing.T) {
	repo := &mockTacticalRepo{univ: universTroisMatchs("m1")}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi).
		WithRasterStore(&mockRasterStore{})
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1"}},
	})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, attendu ErrCapabilityNotSupported", err)
	}
	// AUCUNE LECTURE DE BASE N'A ETE DECLENCHEE : la porte passe avant l'univers.
	if repo.vuUniv.PlayerXUID != "" {
		t.Fatalf("l'univers a ete lu malgre la porte fermee : %+v", repo.vuUniv)
	}
}

// TestOccupation_SansLecteurCable — la capability est la, le cablage manque : 503, et ce
// n'est PAS silencieux (ligne ERROR). Un service qui rendrait une carte vide ferait
// passer un defaut de deploiement pour une absence de donnee.
func TestOccupation_SansLecteurCable(t *testing.T) {
	repo := &mockTacticalRepo{univ: universTroisMatchs("m1")}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi) // pas de WithRasterStore
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1"}},
	})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, attendu ErrCapabilityNotSupported", err)
	}
}

// TestOccupation_SidecarEcarte — un sidecar d'un autre format, d'un autre pas de grille ou
// d'un autre pas d'echantillonnage n'entre PAS dans la somme. Melanger deux unites de
// temps sous le meme nom donnerait un resultat faux sans rien casser.
func TestOccupation_SidecarEcarte(t *testing.T) {
	autreFormat := sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8))
	autreFormat.SchemaVersion = domain.TacticalRasterSchemaVersion + 1
	autrePas := sidecarPose("m2", joueurEn(tsMoi, 2, 3, 8))
	autrePas.PasEchantillonMs = 1000
	autreGrille := sidecarPose("m3", joueurEn(tsMoi, 2, 3, 8))
	autreGrille.PasM = 1.0
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": autreFormat, "m2": autrePas, "m3": autreGrille,
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3"), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.MatchsRetenus != 0 || len(out.Cellules) != 0 {
		t.Fatalf("retenus=%d cellules=%+v : un sidecar d'un autre temps a ete somme",
			out.MatchsRetenus, out.Cellules)
	}
	if out.MatchsFiltres != 3 {
		t.Fatalf("matchs_filtres = %d, attendu 3 : les matchs restent dans l'univers", out.MatchsFiltres)
	}
}

// TestOccupation_SidecarIllisible — un fichier present mais corrompu est SIGNALE puis
// degrade en « non mesure ». L'avaler le ferait passer pour une absence.
func TestOccupation_SidecarIllisible(t *testing.T) {
	store := &mockRasterStore{
		sidecars: map[string]*domain.TacticalRasterSidecar{"m2": sidecarPose("m2", joueurEn(tsMoi, 2, 3, 8))},
		erreurs:  map[string]error{"m1": errors.New("json casse")},
	}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2"), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2"}},
	})
	if err != nil {
		t.Fatalf("un sidecar corrompu ne doit pas casser la lecture : %v", err)
	}
	if out.MatchsRetenus != 1 || out.MatchsFiltres != 2 {
		t.Fatalf("retenus=%d filtres=%d, attendu 1 sur 2", out.MatchsRetenus, out.MatchsFiltres)
	}
}

// TestOccupation_CarteSansMatch — une carte que le filtre ne retient pas est un 404 typé,
// pas une lecture a zero cellule (qui se lirait « rien ne s'y passe »).
func TestOccupation_CarteSansMatch(t *testing.T) {
	svc, _ := occupationSvc(domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}, &mockRasterStore{})
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
	})
	if !errors.Is(err, domain.ErrTacticalCarteInconnue) {
		t.Fatalf("err = %v, attendu ErrTacticalCarteInconnue", err)
	}
}

// TestOccupation_EscouadeSansComposition — la regle de l'axe vaut pour les QUATRE
// questions : sans composition, l'axe escouade est refuse AVANT toute lecture.
func TestOccupation_EscouadeSansComposition(t *testing.T) {
	repo := &mockTacticalRepo{univ: universTroisMatchs("m1")}
	svc := NewTacticalService(repo, capsOccupation(), tsMoi).WithRasterStore(&mockRasterStore{})
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiEscouade,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1"}},
	})
	if !errors.Is(err, domain.ErrTacticalEscouadeSansComposition) {
		t.Fatalf("err = %v, attendu ErrTacticalEscouadeSansComposition", err)
	}
	if repo.vuUniv.PlayerXUID != "" {
		t.Fatalf("l'univers a ete lu malgre le refus d'axe : %+v", repo.vuUniv)
	}
}

// TestOccupation_NePasseJamaisParKillPositions — la lecture d'occupation ne demande a la
// base QUE son univers. Scanner `kill_positions` sur toute la carte pour en jeter le
// resultat serait payer une lecture qu'on ne fait pas.
func TestOccupation_NePasseJamaisParKillPositions(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8)),
	}}
	svc, repo := occupationSvc(universTroisMatchs("m1"), store)
	if _, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1"}},
	}); err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if repo.vuPos.PlayerXUID != "" {
		t.Fatalf("KillPositions a ete appele pour une lecture d'occupation : %+v", repo.vuPos)
	}
	if repo.vuUniv.MapID != "streets" || !repo.vuUniv.Matchs.Restreint() {
		t.Fatalf("l'univers n'a pas recu le perimetre : %+v", repo.vuUniv)
	}
	if len(store.vus) != 1 || store.vus[0] != "m1" {
		t.Fatalf("sidecars demandes = %v, attendu [m1]", store.vus)
	}
}

// TestOccupation_PointsIgnoresSommes — C9 : le compte des positions ecartees vient des
// SIDECARS, il ne se recalcule pas.
//
// La somme part de comptes deja groupes par cellule, et un point ecarte n'a jamais eu de
// cellule : `raster.PointsIgnores()` vaudrait toujours 0 ici. Sans le transport, un
// decodage qui derape se serait tu. Seuls les sidecars RETENUS comptent — un fichier qu'on
// n'a pas lu ne doit pas alourdir la statistique d'un decodage qu'on n'a pas vu.
func TestOccupation_PointsIgnoresSommes(t *testing.T) {
	m1 := sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8))
	m1.PointsIgnores = 3
	m2 := sidecarPose("m2", joueurEn(tsMoi, 2, 3, 4))
	m2.PointsIgnores = 5
	// Ecarte pour cause d'unites : ses points ignores ne doivent PAS entrer.
	m3 := sidecarPose("m3", joueurEn(tsMoi, 2, 3, 6))
	m3.PointsIgnores = 100
	m3.PasEchantillonMs = 1000
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": m1, "m2": m2, "m3": m3,
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3"), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.PointsIgnores != 8 {
		t.Fatalf("points_ignores = %d, attendu 8 (3 + 5 ; le sidecar ecarte n'y entre pas)",
			out.PointsIgnores)
	}
}

// TestOccupation_SidecarVideCompteAuDenominateur — C10 : un sidecar VIDE (`joueurs: []`)
// est un match MESURE a zero, pas un match non mesure. Il entre donc au denominateur et
// DILUE la valeur — c'est exactement ce qu'on veut : le joueur a joue ce match, il n'y a
// simplement rien passe de mesurable sur cette cible.
func TestOccupation_SidecarVideCompteAuDenominateur(t *testing.T) {
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8)),
		"m2": sidecarPose("m2", joueurEn(tsMoi, 2, 3, 8)),
		"m3": sidecarPose("m3", joueurEn(tsMoi, 2, 3, 8)),
		"m4": sidecarPose("m4"), // present, aucun joueur
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3", "m4"), store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3", "m4"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.MatchsRetenus != 4 {
		t.Fatalf("matchs_retenus = %d, attendu 4 : un sidecar VIDE est un match mesure a zero",
			out.MatchsRetenus)
	}
	if len(out.Cellules) != 1 {
		t.Fatalf("cellules = %+v, attendu 1", out.Cellules)
	}
	// 24 echantillons x 0,25 s / 4 matchs = 1,5 s. Avec 3 au denominateur ce serait 2,0.
	if out.Cellules[0].Valeur != 1.5 {
		t.Fatalf("valeur = %v s, attendu 1,5 (24 x 0,25 s / 4 matchs mesures)", out.Cellules[0].Valeur)
	}
}

// TestOccupation_EchangeServiSousTemps — C4 : la decision « le KPI d'echange est celui de
// la CARTE, pas de la question » n'etait prouvee nulle part — les huit cas d'occupation
// montaient un titre sans `film.kill_source`, si bien que supprimer l'appel au journal
// laissait la suite verte.
//
// Ce test tient les DEUX moities : l'echange EST servi sous `temps`, et la couverture
// d'evenements reste a zero — l'occupation ne regarde aucune face d'une mort.
func TestOccupation_EchangeServiSousTemps(t *testing.T) {
	univ := universTroisMatchs("m1", "m2", "m3")
	// Le journal des morts : deux morts de mon camp, une vengee dans la fenetre.
	events := []domain.KillEvent{
		{MatchID: "m1", VictimXUID: tsMoi, KillerXUID: tsAdv, TimeMs: 10_000},
		{MatchID: "m1", VictimXUID: tsAdv, KillerXUID: tsAmi, TimeMs: 12_000},
		{MatchID: "m2", VictimXUID: tsAmi, KillerXUID: tsAdv2, TimeMs: 30_000},
	}
	caps := games.CapabilityMap{
		games.CapFilmReplayArtifact: games.CapSupported,
		games.CapFilmKillPositions:  games.CapSupported,
		games.CapFilmKillSource:     games.CapSupported,
	}
	repo := &mockTacticalRepo{univ: univ, ev: domain.TacticalKillEvents{Univers: univ, Events: events}}
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueurEn(tsMoi, 2, 3, 8)),
		"m2": sidecarPose("m2", joueurEn(tsMoi, 2, 3, 8)),
		"m3": sidecarPose("m3", joueurEn(tsMoi, 2, 3, 8)),
	}}
	svc := NewTacticalService(repo, caps, tsMoi).WithRasterStore(store)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionTemps, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.Echange == nil {
		t.Fatal("l'echange n'est PAS servi sous `temps` : c'est le KPI de la CARTE, pas celui " +
			"de la question — supprimer l'appel au journal doit faire tomber ce test")
	}
	if out.Echange.N == 0 {
		t.Fatalf("echange = %+v, attendu au moins une mort vengeable", out.Echange)
	}
	// ET la couverture d'evenements reste a zero : l'occupation ne lit aucune face d'une
	// mort, son denominateur de couverture est l'ecart matchs_retenus / matchs_filtres.
	if out.EvenementsJournal != 0 {
		t.Fatalf("evenements_journal = %d, attendu 0 sous une lecture d'occupation",
			out.EvenementsJournal)
	}
}
