// Package service — TacticalService : l'onglet Tactique, lecture par CARTE.
//
// Orchestration (arch-rules) : combine UN port (port.TacticalRepository) et DEUX
// algos purs (analysis/tactical pour le rasterisage, analysis/coordination pour
// l'echange). Aucun SQL, aucune ouverture de base, aucun appel a un autre service.
//
// LE PERIMETRE ARRIVE RESOLU (phase 4 bis, 2026-09-06) : la page fait resoudre sa
// selection — periode OU sessions epinglees, contexte solo/escouade, cascade — par le
// endpoint de filtres (service.FilteredMatchIDs, base JOUEUR), et ce service recoit des
// match_id en LISTE BLANCHE. Il ne filtre donc plus rien : une seconde definition du
// perimetre donnerait deux comptes de matchs pour la meme question, et c'est elle qui
// laissait le filtre de session sans effet sur cet onglet (il vit dans la base joueur,
// que les requetes shared du lecteur ne joignent pas).
//
// ─── L'UNIVERS VIENT DU FILTRE, JAMAIS DES POINTS ──────────────────────────────
//
// C'est LA regle de ce chantier, et elle se voit dans chaque appel ci-dessous :
// `Rasterise` et `RasteriseAvecResultats` recoivent l'ensemble des matchs RETENUS
// en entree explicite, et les points par-dessus. Un match retenu sans aucune
// position mesuree compte au denominateur « par match ». Le deduire des points
// l'effacerait, et la lecture signee peindrait une zone gagnante la ou il n'y a
// que des matchs muets d'un cote (12 victoires dont 2 muettes : +0,10 lu au lieu
// de 0,00).
//
// ─── DEUX PORTES DE CAPABILITY, DEUX EFFETS DIFFERENTS ─────────────────────────
//
//	positions LISIBLES  ABSENTES -> ErrCapabilityNotSupported -> 503 propre. Sans
//	                    positions il n'y a pas de lecture de placement du tout.
//	journal FIABLE      ABSENT -> le KPI d'echange est SILENCIEUX (nil), la lecture
//	                    de placement reste servie. Publier un zero se lirait comme
//	                    une contre-performance ; ne rien publier dit ce qui est
//	                    vrai : ce titre ne sait pas mesurer ca.
//
// Les deux se lisent sur la CapabilityMap de l'adapter du titre du joueur — jamais
// une comparaison de slug (ratchet no_slug_comparison_test). Chacune accepte DEUX
// PROVENANCES, et c'est le fond de la correction R1 (revue du 2026-09-06) :
// cf. positionsDeKillLisibles et journalDesMortsFiable ci-dessous.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// TacticalService implemente port.TacticalService.
type TacticalService struct {
	repo port.TacticalRepository
	caps games.CapabilityMap
	// xuid est le joueur de la page : l'univers et l'axe « moi » sont les siens.
	xuid   string
	logger *slog.Logger
}

// NewTacticalService construit le service.
//
// `repo` nil (titre sans lecteur cable) -> toute lecture rend
// ErrCapabilityNotSupported. `caps` nil -> CapabilityMap.Has rend faux, donc meme
// degradation : une map non chargee ne vaut pas une capability presente.
func NewTacticalService(repo port.TacticalRepository, caps games.CapabilityMap, playerXUID string) *TacticalService {
	return &TacticalService{repo: repo, caps: caps, xuid: playerXUID, logger: slog.Default()}
}

// WithLogger injecte un logger (sinon slog.Default()). Chainable.
func (s *TacticalService) WithLogger(l *slog.Logger) *TacticalService {
	if l != nil {
		s.logger = l
	}
	return s
}

// MapsPlayed rend les cartes jouees dans le perimetre, avec leur verdict de
// lisibilite (plancher par carte).
func (s *TacticalService) MapsPlayed(ctx context.Context, scope domain.TacticalScope) (domain.TacticalMapsPage, error) {
	page := domain.TacticalMapsPage{PlancherMatchs: domain.PlancherMatchsParCarte}
	if s.repo == nil {
		return page, games.ErrCapabilityNotSupported
	}
	compo := compositionNettoyee(scope.Coequipiers)
	if err := domain.ValiderComposition(compo); err != nil {
		s.logger.WarnContext(ctx, "tactique: composition refusee",
			"player", s.xuid, "coequipiers", len(compo), "err", err)
		return page, err
	}
	debut := time.Now()
	rows, err := s.repo.MapsPlayed(ctx, requeteDuScope(s.xuid, "", scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: cartes jouees en echec", "player", s.xuid, "err", err)
		return page, err
	}
	page.Cartes = make([]domain.TacticalMapCard, 0, len(rows))
	for _, r := range rows {
		page.Cartes = append(page.Cartes, domain.TacticalMapCard{
			MapID: r.MapID, MapName: r.MapName, MapNameFR: r.MapNameFR,
			Matchs: r.Matchs, Victoires: r.Victoires, Defaites: r.Defaites,
			SousPlancher: r.Matchs < domain.PlancherMatchsParCarte,
		})
	}
	s.logger.InfoContext(ctx, "tactique: cartes jouees",
		"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "cartes", len(page.Cartes),
		"matchs_filtres", len(scope.MatchIDs), "coequipiers", len(scope.Coequipiers),
		"plancher_matchs", domain.PlancherMatchsParCarte, "duration", time.Since(debut))
	return page, nil
}

// Raster rend la lecture de placement d'une carte.
func (s *TacticalService) Raster(ctx context.Context, req domain.TacticalRasterRequest) (domain.TacticalRaster, error) {
	carte, question, qui := req.MapID, req.Question, req.Qui
	out := domain.TacticalRaster{MapID: carte, Question: question, Qui: qui}
	scope := domain.TacticalScope{
		MatchIDs:    req.Scope.MatchIDs,
		Coequipiers: compositionNettoyee(req.Scope.Coequipiers),
	}
	if err := validerLecture(carte, question, qui, scope.Coequipiers); err != nil {
		return out, err
	}
	if s.repo == nil || !positionsDeKillLisibles(s.caps) {
		s.logger.WarnContext(ctx, "tactique: aucune position de kill lisible pour ce titre",
			"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "map_id", carte, "question", question)
		return out, games.ErrCapabilityNotSupported
	}
	debut := time.Now()

	lecture, err := s.repo.KillPositions(ctx, requeteDuScope(s.xuid, carte, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: lecture des positions en echec",
			"player", s.xuid, "map_id", carte, "question", question, "err", err)
		return out, err
	}
	if len(lecture.Univers.Matchs) == 0 {
		// LA SENTINELLE NUE, ET LE DETAIL AU JOURNAL (revue R2, P1). Le message de cette
		// erreur est PUBLIE par le handler : y citer la carte demandee faisait differer le
		// corps d'une carte legitime inconnue de celui d'un map_id refuse par
		// `MapIDValide` (qui n'a rien a citer). La presence de l'identifiant suffisait
		// alors a dire a l'appelant laquelle des deux frontieres il avait heurtee — un
		// oracle par le libelle, apres celui qu'on venait de fermer par le code.
		s.logger.InfoContext(ctx, "tactique: carte sans match retenu",
			"player", s.xuid, "map_id", carte, "question", question, "qui", qui)
		return out, domain.ErrTacticalCarteInconnue
	}
	// MATCHS FILTRES = L'UNIVERS DE CETTE CARTE : les matchs du perimetre que le joueur
	// y a joues, mesures ou non. MatchsRetenus en est un SOUS-ENSEMBLE (les mesures),
	// de sorte que le pied de carte puisse dire « N sur M » sans mentir.
	out.MatchsFiltres = len(lecture.Univers.Matchs)

	// L'UNIVERS DE LA LECTURE, C'EST LES MATCHS MESURES (correction G2, 2026-09-06).
	// Un match du filtre dont le film n'a jamais ete decode ne peut alimenter aucune
	// cellule : le garder au denominateur ferait varier l'intensite avec la
	// couverture de film au lieu du jeu. Il reste publie a part, dans MatchsFiltres.
	mesure := universMesure(lecture.Univers)
	out.MatchsRetenus = len(mesure.Matchs)

	points := projeter(lecture, question, cible(lecture.Univers.Equipes, qui, s.xuid, scope.Coequipiers))
	out.EvenementsLocalises = len(points)
	raster, err := rasteriser(mesure, question, points)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: rasterisage en echec",
			"player", s.xuid, "map_id", carte, "question", question, "err", err)
		return out, fmt.Errorf("tactique: rasterisage: %w", err)
	}
	remplirRaster(&out, raster, question)
	s.lireLeJournal(ctx, &out, scope)

	s.logger.InfoContext(ctx, "tactique: lecture de placement",
		"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "map_id", carte,
		"question", question, "qui", qui,
		"matchs_filtres", out.MatchsFiltres, "matchs_retenus", out.MatchsRetenus,
		"coequipiers", len(scope.Coequipiers),
		"cellules", len(out.Cellules), "points_ignores", out.PointsIgnores,
		"evenements_journal", out.EvenementsJournal,
		"evenements_localises", out.EvenementsLocalises,
		"duration", time.Since(debut))
	return out, nil
}

// universMesure ne garde que les matchs dont le journal des morts est LISIBLE.
//
// Les EQUIPES sont conservees telles quelles : elles servent au predicat « qui »,
// qui interroge des matchs presents dans les points — donc mesures par construction
// (une position n'existe qu'accrochee a un kill-event publiable).
func universMesure(u domain.TacticalUnivers) domain.TacticalUnivers {
	out := domain.TacticalUnivers{
		Matchs:  make([]domain.TacticalMatch, 0, len(u.Matchs)),
		Equipes: u.Equipes,
	}
	for _, m := range u.Matchs {
		if m.Mesure {
			out.Matchs = append(out.Matchs, m)
		}
	}
	return out
}

// projeter transforme les morts mesurees en points a rasteriser, selon la question
// et l'axe « qui ».
//
//	morts  -> la position de la VICTIME, quand la victime est dans la cible ;
//	kills  -> la position du TUEUR, quand le tueur est dans la cible ;
//	gagne  -> LES DEUX (l'engagement a deux faces, cf. domain.TacticalQuestionGagne).
func projeter(lecture domain.TacticalPositions, question string, cible predicatQui) []domain.PositionSample {
	prendVictime, prendTueur := facesDeLaQuestion(question)

	points := make([]domain.PositionSample, 0, len(lecture.Points))
	for _, p := range lecture.Points {
		if prendVictime && cible(p.MatchID, p.VictimXUID) {
			points = append(points, domain.PositionSample{MatchID: p.MatchID, X: p.VictimX, Y: p.VictimY})
		}
		if prendTueur && cible(p.MatchID, p.KillerXUID) {
			points = append(points, domain.PositionSample{MatchID: p.MatchID, X: p.KillerX, Y: p.KillerY})
		}
	}
	return points
}

// facesDeLaQuestion dit quelles FACES d'une mort la question regarde. Source unique
// des deux lectures qui en dependent — le rasterisage (projeter) et le comptage de
// couverture (compterJournal) — pour qu'un « ou je gagne » qui cesserait de compter
// les morts ne puisse pas le faire d'un seul cote.
func facesDeLaQuestion(question string) (prendVictime, prendTueur bool) {
	return question != domain.TacticalQuestionKills, question != domain.TacticalQuestionMorts
}

// rasteriser choisit la forme de rasterisage qu'exige la question : SIGNEE pour
// « ou je gagne » (les resultats font partie de l'entree), simple sinon.
func rasteriser(univers domain.TacticalUnivers, question string, points []domain.PositionSample) (*tactical.Raster, error) {
	g := tactical.GrilleParDefaut()
	if question == domain.TacticalQuestionGagne {
		resultats := make(map[string]int, len(univers.Matchs))
		for _, m := range univers.Matchs {
			resultats[m.MatchID] = m.Outcome
		}
		return tactical.RasteriseAvecResultats(g, resultats, points)
	}
	ids := make([]string, 0, len(univers.Matchs))
	for _, m := range univers.Matchs {
		ids = append(ids, m.MatchID)
	}
	return tactical.Rasterise(g, ids, points)
}

// remplirRaster habille la reponse : cellules, echelle, cadre.
func remplirRaster(out *domain.TacticalRaster, raster *tactical.Raster, question string) {
	if question == domain.TacticalQuestionGagne {
		out.Cellules = raster.CellulesSignees()
		out.Echelle = tactical.EchelleSymetrique(out.Cellules)
		// Les DEUX denominateurs de la lecture signee, sur l'univers entier : ils ne
		// valent pas MatchsRetenus, et leur somme lui est en general inferieure (les
		// nuls et les resultats inconnus ne participent a aucun cote).
		out.MatchsVictoire = raster.NbMatchsResultat(domain.OutcomeWin)
		out.MatchsDefaite = raster.NbMatchsResultat(domain.OutcomeLoss)
	} else {
		out.Cellules = raster.Cellules()
		out.Echelle = tactical.Echelle(out.Cellules)
	}
	out.PasM = raster.PasM()
	out.Bornes = raster.Bornes()
	out.PointsIgnores = raster.PointsIgnores()
}

// lireLeJournal fait UN SEUL passage sur le journal des morts, et en tire DEUX
// choses de nature differente :
//
//  1. la COUVERTURE de la carte — combien d'evenements de la cible le journal
//     compte, face aux positions effectivement localisees. Elle est servie quelle
//     que soit la capability : c'est une propriete de la mesure, pas un KPI ;
//  2. le KPI d'ECHANGE, lui, seulement si le journal est FIABLE ligne a ligne
//     (journalDesMortsFiable). Sinon il reste nil — jamais un zero, qui se lirait
//     comme une contre-performance.
//
// Un echec de lecture est journalise puis degrade : la lecture de placement, elle,
// reste servie. Aucune erreur avalee.
func (s *TacticalService) lireLeJournal(ctx context.Context, out *domain.TacticalRaster, scope domain.TacticalScope) {
	lecture, err := s.repo.KillEvents(ctx, requeteDuScope(s.xuid, out.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: journal des morts en echec (couverture et echange non servis)",
			"player", s.xuid, "map_id", out.MapID, "err", err)
		return
	}
	out.EvenementsJournal = compterJournal(lecture, out.Question,
		cible(lecture.Univers.Equipes, out.Qui, s.xuid, scope.Coequipiers))
	if journalDesMortsFiable(s.caps) {
		out.Echange = s.mesurerEchange(ctx, out.MapID, lecture)
	}
}

// compterJournal compte les evenements de la cible dans le journal — le
// DENOMINATEUR de la couverture. Memes faces que le rasterisage
// (facesDeLaQuestion) : ce qui est compte ici est exactement ce qui aurait ete
// peint si toutes les positions existaient.
func compterJournal(lecture domain.TacticalKillEvents, question string, cible predicatQui) int {
	prendVictime, prendTueur := facesDeLaQuestion(question)
	n := 0
	for _, e := range lecture.Events {
		if prendVictime && cible(e.MatchID, e.VictimXUID) {
			n++
		}
		if prendTueur && cible(e.MatchID, e.KillerXUID) {
			n++
		}
	}
	return n
}

// mesurerEchange rend le taux de morts vengees DE MON EQUIPE sur cette carte.
func (s *TacticalService) mesurerEchange(ctx context.Context, carte string, lecture domain.TacticalKillEvents) *domain.Couverture {
	bilan := coordination.Echanges(lecture.Events, lecture.Univers.Equipes)
	monCamp := campDuMatch(lecture.Univers.Equipes, s.xuid)

	vengeables, vengees := 0, 0
	for _, m := range bilan.Morts {
		// MON CAMP ENTIER — mes coequipiers DU MATCH et moi (decision utilisateur du
		// 2026-09-06), et surtout PAS la composition choisie : le denominateur le moins
		// biaise ne doit pas retrecir parce qu'on a nomme deux coequipiers dans la barre
		// de filtres. `campDuMatch` m'exclut par construction, d'ou le test explicite.
		if m.VictimeXUID != s.xuid && !monCamp(m.MatchID, m.VictimeXUID) {
			continue
		}
		if !m.Vengeable {
			continue
		}
		vengeables++
		if m.Vengee {
			vengees++
		}
	}
	// Denominateur « par match » : les matchs MESURES, jamais tous les matchs du
	// filtre (correction G2) — le numerateur ne peut venir que d'eux.
	mesures := len(universMesure(lecture.Univers).Matchs)
	c := coordination.Mesurer(vengees, vengeables, mesures)
	s.logger.InfoContext(ctx, "tactique: echange sur cette carte",
		"player", s.xuid, "map_id", carte, "matchs_retenus", mesures,
		"morts_vengeables", c.N, "morts_vengees", c.Brut, "echantillon_faible", c.EchantillonFaible)
	return &c
}

// ─── LES DEUX PORTES DE LECTURE ────────────────────────────────────────────────

// positionsDeKillLisibles dit si la table `kill_positions` de ce titre est LISIBLE,
// quelle que soit la main qui l'a remplie (correction R1, revue du 2026-09-06).
//
// LE DEFAUT CORRIGE : la version precedente gatait sur `film.kill_positions` seule.
// Or cette cle GOUVERNE LA CAPTURE, pas la lecture — son propre commentaire le dit
// (`games/adapter.go`, doc de CapFilmKillPositions). Halo 5 ne la declare pas et
// n'a aucune raison de le faire : il n'a pas de decodeur de film, il remplit la
// MEME table NATIVEMENT depuis le carnage (`games/halo_5/ingest/positions.go`,
// `match.events.spatial = supported`). Un joueur Halo 5 recevait donc un 503 alors
// que la jointure aurait rendu toutes ses positions.
//
// LES DEUX PROVENANCES, donc :
//
//	film.kill_positions   la CAPTURE par decodage de film (Halo Infinite) ;
//	match.events.spatial  les positions NATIVES de l'API du titre (Halo 5).
//
// L'une ou l'autre suffit : ce qui est lu est la meme table, par la meme jointure.
//
// A TERME (consigne au §7 du plan) : une cle FINE de LECTURE — « positions de kill
// lisibles » — dirait cela d'un seul mot, au lieu d'un OU sur deux cles qui
// repondent chacune a une autre question. Elle releve du vocabulaire de
// capabilities du lot C de l'audit, pas de ce lot.
func positionsDeKillLisibles(caps games.CapabilityMap) bool {
	return caps.Has(games.CapFilmKillPositions) || caps.Has(games.CapMatchEventsSpatial)
}

// journalDesMortsFiable est le predicat PARTAGE games.JournalDesMortsFiable.
//
// Il a quitte ce fichier le 2026-09-06 (phase 3 du plan tactique) : la page Escouade
// applique EXACTEMENT la meme porte pour sa matrice d'echange et sa distribution de delais,
// et une seconde copie ici aurait donne deux verdicts au premier titre ajoute. Le corps, ses
// deux provenances et la raison du `supported` STRICT vivent desormais dans
// internal/games/kill_journal_gate.go.
func journalDesMortsFiable(caps games.CapabilityMap) bool {
	return games.JournalDesMortsFiable(caps)
}
