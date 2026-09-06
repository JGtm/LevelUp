// Package service — TacticalService : l'onglet Tactique, lecture par CARTE.
//
// Orchestration (arch-rules) : combine UN port (port.TacticalRepository) et DEUX
// algos purs (analysis/tactical pour le rasterisage, analysis/coordination pour
// l'echange). Aucun SQL, aucune ouverture de base, aucun appel a un autre service
// — l'univers des matchs vient du lecteur, qui applique le filtre de
// l'Explorateur par le builder pur `analysis.BuildNeighborsWhereClause`.
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

// MapsPlayed rend les cartes jouees sous le filtre, avec leur verdict de
// lisibilite (plancher par carte).
func (s *TacticalService) MapsPlayed(ctx context.Context, filtre *domain.MatchFilterSpec) (domain.TacticalMapsPage, error) {
	page := domain.TacticalMapsPage{PlancherMatchs: domain.PlancherMatchsParCarte}
	if s.repo == nil {
		return page, games.ErrCapabilityNotSupported
	}
	debut := time.Now()
	rows, err := s.repo.MapsPlayed(ctx, domain.TacticalQuery{PlayerXUID: s.xuid, Filtre: filtre})
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
		"plancher_matchs", domain.PlancherMatchsParCarte, "duration", time.Since(debut))
	return page, nil
}

// Raster rend la lecture de placement d'une carte.
func (s *TacticalService) Raster(ctx context.Context, carte, question, qui string, filtre *domain.MatchFilterSpec) (domain.TacticalRaster, error) {
	out := domain.TacticalRaster{MapID: carte, Question: question, Qui: qui}
	if err := validerLecture(carte, question, qui); err != nil {
		return out, err
	}
	if s.repo == nil || !positionsDeKillLisibles(s.caps) {
		s.logger.WarnContext(ctx, "tactique: aucune position de kill lisible pour ce titre",
			"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "map_id", carte, "question", question)
		return out, games.ErrCapabilityNotSupported
	}
	debut := time.Now()

	lecture, err := s.repo.KillPositions(ctx, domain.TacticalQuery{PlayerXUID: s.xuid, MapID: carte, Filtre: filtre})
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: lecture des positions en echec",
			"player", s.xuid, "map_id", carte, "question", question, "err", err)
		return out, err
	}
	if len(lecture.Univers.Matchs) == 0 {
		return out, fmt.Errorf("%w (%q)", domain.ErrTacticalCarteInconnue, carte)
	}
	out.MatchsRetenus = len(lecture.Univers.Matchs)

	points := projeter(lecture, question, qui, s.xuid)
	raster, err := rasteriser(lecture.Univers, question, points)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: rasterisage en echec",
			"player", s.xuid, "map_id", carte, "question", question, "err", err)
		return out, fmt.Errorf("tactique: rasterisage: %w", err)
	}
	remplirRaster(&out, raster, question)
	out.Echange = s.echange(ctx, carte, filtre)

	s.logger.InfoContext(ctx, "tactique: lecture de placement",
		"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "map_id", carte,
		"question", question, "qui", qui, "matchs_retenus", out.MatchsRetenus,
		"cellules", len(out.Cellules), "points_ignores", out.PointsIgnores,
		"duration", time.Since(debut))
	return out, nil
}

// validerLecture refuse une demande hors vocabulaire AVANT toute lecture de base.
func validerLecture(carte, question, qui string) error {
	if carte == "" {
		return fmt.Errorf("%w (carte vide)", domain.ErrTacticalCarteInconnue)
	}
	switch question {
	case domain.TacticalQuestionMorts, domain.TacticalQuestionKills, domain.TacticalQuestionGagne:
	default:
		return fmt.Errorf("%w (%q)", domain.ErrTacticalQuestionInconnue, question)
	}
	switch qui {
	case domain.TacticalQuiMoi, domain.TacticalQuiEscouade, domain.TacticalQuiAdversaires:
		return nil
	default:
		return fmt.Errorf("%w (%q)", domain.ErrTacticalQuiInconnu, qui)
	}
}

// projeter transforme les morts mesurees en points a rasteriser, selon la question
// et l'axe « qui ».
//
//	morts  -> la position de la VICTIME, quand la victime est dans la cible ;
//	kills  -> la position du TUEUR, quand le tueur est dans la cible ;
//	gagne  -> LES DEUX (l'engagement a deux faces, cf. domain.TacticalQuestionGagne).
func projeter(lecture domain.TacticalPositions, question, qui, moi string) []domain.PositionSample {
	cible := ciblePar(lecture.Univers.Equipes, qui, moi)
	prendVictime := question != domain.TacticalQuestionKills
	prendTueur := question != domain.TacticalQuestionMorts

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

// ciblePar rend le predicat d'appartenance a l'axe demande, PAR MATCH.
//
// Une identite VIDE (bot, environnement) n'appartient a aucun axe : elle n'a pas
// d'equipe, et la ranger quelque part serait une invention. Un joueur absent de la
// composition du match non plus — son equipe est INCONNUE, pas devinable.
func ciblePar(equipes domain.EquipesParMatch, qui, moi string) func(matchID, xuid string) bool {
	return func(matchID, xuid string) bool {
		if xuid == "" {
			return false
		}
		if qui == domain.TacticalQuiMoi {
			return xuid == moi
		}
		duMatch := equipes[matchID]
		monEquipe, jeSuisLa := duMatch[moi]
		son, ilEstLa := duMatch[xuid]
		if !jeSuisLa || !ilEstLa {
			return false
		}
		if qui == domain.TacticalQuiEscouade {
			return son == monEquipe && xuid != moi
		}
		return son != monEquipe
	}
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
	} else {
		out.Cellules = raster.Cellules()
		out.Echelle = tactical.Echelle(out.Cellules)
	}
	out.PasM = raster.PasM()
	out.Bornes = raster.Bornes()
	out.PointsIgnores = raster.PointsIgnores()
}

// echange mesure le taux de morts vengees DE MON EQUIPE sur cette carte.
//
// nil (et non une Couverture a zero) quand le titre ne sait pas lire la source des
// morts, ou quand la lecture echoue : la lecture de placement, elle, reste servie.
// Un echec est journalise avant la degradation, jamais avale.
func (s *TacticalService) echange(ctx context.Context, carte string, filtre *domain.MatchFilterSpec) *domain.Couverture {
	if !journalDesMortsFiable(s.caps) {
		return nil
	}
	lecture, err := s.repo.KillEvents(ctx, domain.TacticalQuery{PlayerXUID: s.xuid, MapID: carte, Filtre: filtre})
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: journal des morts en echec (KPI d'echange non servi)",
			"player", s.xuid, "map_id", carte, "err", err)
		return nil
	}
	bilan := coordination.Echanges(lecture.Events, lecture.Univers.Equipes)
	monCamp := ciblePar(lecture.Univers.Equipes, domain.TacticalQuiEscouade, s.xuid)

	vengeables, vengees := 0, 0
	for _, m := range bilan.Morts {
		// Mon camp = mes coequipiers ET moi. `ciblePar(escouade)` m'exclut par
		// construction (c'est ce que « escouade » veut dire sur les rasters) : la
		// couverture, elle, porte sur l'equipe entiere.
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
	c := coordination.Mesurer(vengees, vengeables, len(lecture.Univers.Matchs))
	s.logger.InfoContext(ctx, "tactique: echange sur cette carte",
		"player", s.xuid, "map_id", carte, "matchs_retenus", len(lecture.Univers.Matchs),
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

// journalDesMortsFiable dit si `match_kill_events` de ce titre nomme le tueur de
// chaque mort de facon exploitable LIGNE A LIGNE — ce qu'exige l'echange (« qui a
// venge qui »), et rien de moins.
//
// LES DEUX PROVENANCES, et elles ne se lisent PAS de la meme facon :
//
//	film.kill_source              la source du degat fatal, decodee du film
//	                              (Halo Infinite : supported). `Has` suffit.
//	match.killfeed.per_kill       le kill-feed natif de l'API du titre. Exige ici
//	                              `supported` STRICTEMENT, pas `Has`.
//
// POURQUOI `supported` STRICTEMENT SUR LA SECONDE. `CapabilityMap.Has` accepte
// aussi `degraded`, et Halo Infinite declare justement `match.killfeed.per_kill =
// degraded` (kills simultanes possiblement omis, cf. capabilities.toml) — soit
// exactement le defaut qui fabriquerait de faux echanges : une mort omise dans la
// fenetre de 5 s se lit comme « non vengee ». Infinite passe deja par
// `film.kill_source` ; le exiger `supported` ici n'ote donc rien a personne, et
// protege le jour ou un titre ne declarerait QUE ce kill-feed la, en degrade.
// Halo 5 declare `supported` (mesure sur pieces, capabilities.toml du titre) et
// remplit `match_kill_events` par la reprise de `killer_victim_pairs`.
func journalDesMortsFiable(caps games.CapabilityMap) bool {
	return caps.Has(games.CapFilmKillSource) ||
		caps[games.CapMatchKillfeedPerKill] == games.CapSupported
}
