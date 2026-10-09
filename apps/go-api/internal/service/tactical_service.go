// Package service — TacticalService : l'onglet Tactique, lecture par CARTE.
//
// Orchestration (arch-rules) : combine UN port (port.TacticalRepository) et DEUX
// algos purs (analysis/tactical pour le rasterisage, analysis/coordination pour
// l'isolement). Aucun SQL, aucune ouverture de base, aucun appel a un autre service.
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
// ─── LA PORTE DES LECTURES DE BASE ─────────────────────────────────────────────
//
//	positions LISIBLES  ABSENTES -> ErrCapabilityNotSupported -> 503 propre. Sans
//	                    positions il n'y a pas de lecture de placement du tout.
//
// Elle se lit sur la CapabilityMap de l'adapter du titre du joueur — jamais une
// comparaison de slug (ratchet no_slug_comparison_test). Elle accepte DEUX PROVENANCES,
// et c'est le fond de la correction R1 (revue du 2026-09-06) : cf.
// positionsDeKillLisibles ci-dessous. Les lectures d'artefact ont leur propre porte
// (tactical_service_rasters.go).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	xuid string
	// rasters lit les sidecars d'occupation (phase 6). NIL = la lecture `temps` degrade
	// en 503 EN LE DISANT (cf. lectureOccupation) ; les trois autres questions, qui se
	// lisent sur la base, n'en dependent pas.
	rasters port.TacticalRasterStore
	// callouts nomme les grappes de reapparition. Nil = grappes MUETTES, jamais d'erreur.
	callouts port.TacticalCalloutsStore
	// radar : game_variant_name -> portee du radar en metres (`regulation.toml`). Une
	// variante absente n'a PAS de rayon : ses matchs sortent de la lecture « isole » et se
	// comptent a part (jamais un rayon de repli, qui rendrait une mesure d'apparence normale
	// sur une regle de jeu qu'on n'a pas etablie).
	radar map[string]int
	// retentionMois rend la fenetre de retention des artefacts de rejeu, en mois. MEME
	// SOURCE ET MEME CONVENTION que la purge et que la file (0 = ILLIMITEE).
	//
	// NIL VAUT 0, DONC ILLIMITEE — pas « pas de ventilation », qui reste toujours publiee.
	// Sans fenetre, aucun match n'est ecarte POUR SON AGE : seuls le marqueur de film perdu
	// et l'absence d'horodatage le rendent non cuisable. C'est la degradation sure — ne pas
	// connaitre la fenetre ne doit pas faire dire « jamais cuit » a un match cuisable.
	retentionMois func() int
	logger        *slog.Logger
	detail        sourcesDuDetail // les sources du détail d'une zone (tactical_service_cellule_enrichir.go)
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
	s.peindreLesVignettes(ctx, &page, scope)
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
		// LE FILTRE DE SPAWN TRAVERSE, et il faut le recopier explicitement : cette
		// reconstruction du scope existe pour NETTOYER la composition, et tout champ
		// qu'elle oublie est perdu en silence — la lecture repond alors sur l'univers
		// entier en ayant l'air d'avoir filtre.
		Spawn: req.Scope.Spawn,
	}
	if err := validerLecture(carte, question, qui, scope.Coequipiers); err != nil {
		return out, err
	}
	if s.repo == nil {
		return out, games.ErrCapabilityNotSupported
	}
	// LE FILTRE DE SPAWN S'APPLIQUE AVANT LE DISPATCH, ET C'EST TOUT L'OBJET DE P1-1 : il
	// restreint la LISTE BLANCHE DE MATCHS, donc il vaut pour les lectures SQL
	// (morts/kills/gagne/solde/isole) autant que pour les lectures d'artefact.
	// Applique dans la seule branche des sidecars, il rendait 200 sur l'univers ENTIER sous
	// un libelle de grappe.
	var dejaLus map[string]*domain.TacticalRasterSidecar
	if scope.Spawn != "" {
		per, err := s.perimetreDuSpawn(ctx, carte, scope)
		if err != nil {
			return out, err
		}
		out.Grappes = per.Grappes
		scope.MatchIDs = per.MatchIDs
		// LES SIDECARS SONT DEJA EN MAIN : la lecture qui suit porte sur un sous-ensemble
		// de cet univers, et les relire serait une seconde traversee du disque pour les
		// memes fichiers (revue P2).
		dejaLus = per.Sidecars
	}
	var err error
	switch {
	case lectureDArtefact(question):
		// L'OCCUPATION A SA PROPRE PORTE ET SON PROPRE SUBSTRAT (cf.
		// tactical_service_rasters.go) : elle ne lit pas `kill_positions` du tout, elle
		// somme des sidecars tires des PISTES du film.
		// L'ERREUR EST CAPTUREE AVANT LE RETOUR : `return out, f(&out)` laisserait
		// l'ordre d'evaluation des operandes decider si la reponse rendue est celle
		// d'avant ou d'apres le remplissage.
		err = s.rasterArtefact(ctx, &out, scope, dejaLus)
	case question == domain.TacticalQuestionIsole:
		// « ISOLE » LIT LA BASE COMME LES LECTURES DE PLACEMENT, mais sur DEUX tables de
		// plus : le contexte de chaque mort, ecrit au sync, et les positions pour le lieu.
		// Elle n'attend AUCUN artefact — la ventilation en attente / non cuisables ne la
		// concerne donc pas.
		// `rasterIsole` POSE LUI-MEME la section : elle sort de la lecture des morts qu'il
		// fait deja, et la redemander serait une seconde requete pour la meme table.
		err = s.rasterIsole(ctx, &out, scope)
	default:
		err = s.rasterDeKills(ctx, &out, scope)
	}
	if err == nil {
		out.Zones = s.zonesDuPlan(ctx, carte)
	}
	return out, err
}

// rasterDeKills sert les quatre lectures qui se lisent sur les POSITIONS MESUREES de
// `kill_positions` : ou je meurs, ou je tue, ou je gagne, le solde frags − morts.
//
// EXTRAITE DE `Raster` (constat C8 de la revue) : celle-ci depassait le seuil de 80 lignes
// au sens de `funlen`, et le ratchet de la CI ne pouvait pas le voir — la position d'une
// fonction n'est pas une ligne AJOUTEE, donc `--new-from-merge-base` ne la signale pas. La
// coupure suit la seule frontiere naturelle du service : ce qui vient de la BASE ici, ce
// qui vient des SIDECARS a cote.
func (s *TacticalService) rasterDeKills(ctx context.Context, out *domain.TacticalRaster,
	scope domain.TacticalScope) error {
	carte, question, qui := out.MapID, out.Question, out.Qui
	if !positionsDeKillLisibles(s.caps) {
		s.logger.WarnContext(ctx, "tactique: aucune position de kill lisible pour ce titre",
			"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "map_id", carte, "question", question)
		return games.ErrCapabilityNotSupported
	}
	debut := time.Now()

	lecture, err := s.repo.KillPositions(ctx, requeteDuScope(s.xuid, carte, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: lecture des positions en echec",
			"player", s.xuid, "map_id", carte, "question", question, "err", err)
		return err
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
		return domain.ErrTacticalCarteInconnue
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

	dans := cible(lecture.Univers.Equipes, qui, s.xuid, scope.Coequipiers)
	lue, err := rasteriserLaCible(mesure, lecture, question, dans)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: rasterisage en echec",
			"player", s.xuid, "map_id", carte, "question", question, "err", err)
		return fmt.Errorf("tactique: rasterisage: %w", err)
	}
	remplirRaster(out, lue.Raster, question)
	out.Voisines = s.voisinesDesPositions(ctx, out, mesure, lecture, dans)

	s.logger.InfoContext(ctx, "tactique: lecture de placement",
		"player", s.xuid, "titleSlug", ctxkeys.TitleSlug(ctx), "map_id", carte,
		"question", question, "qui", qui,
		"matchs_filtres", out.MatchsFiltres, "matchs_retenus", out.MatchsRetenus,
		"coequipiers", len(scope.Coequipiers),
		"pas_m", out.PasM, "densite_suffisante", lue.Suffisante, "pas_essayes", lue.Tentatives,
		"cellules", len(out.Cellules), "points_ignores", out.PointsIgnores,
		"voisines", len(out.Voisines), "duration", time.Since(debut))
	return nil
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
// des deux lectures qui en dependent — le rasterisage (projeter) et le detail d'une
// cellule (celluleDeKills) — pour qu'un « ou je gagne » qui cesserait de compter les
// morts ne puisse pas le faire d'un seul cote.
func facesDeLaQuestion(question string) (prendVictime, prendTueur bool) {
	if lectureDArtefact(question) {
		// L'OCCUPATION NE REGARDE AUCUNE FACE D'UNE MORT : elle se lit sur les pistes du
		// film, pas sur le journal. Sans ce cas, la question tomberait dans la branche par
		// defaut et regarderait les DEUX faces, comme « ou je gagne ».
		return false, false
	}
	if question == domain.TacticalQuestionIsole {
		// « ISOLE » NE REGARDE QUE LA FACE VICTIME, comme « ou je meurs » : elle mesure MES
		// morts survenues sans coequipier a portee. Sans ce cas, elle tomberait dans la
		// branche par defaut et regarderait aussi mes kills.
		return true, false
	}
	// « Ou je gagne » et le SOLDE regardent LES DEUX faces : le solde les rasterise
	// separement (rasteriserLaCible), mais son detail de cellule compte les frags ET les
	// morts de la cible.
	return question != domain.TacticalQuestionKills, question != domain.TacticalQuestionMorts
}

// ─── LA PORTE DES POSITIONS DE KILL ───────────────────────────────────────────

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

// lectureDArtefact dit si la question se lit sur les SIDECARS et non sur `kill_positions`.
//
// LES TROIS PARTAGENT TOUT : la meme porte (`film.replay_artifact`), le meme substrat, le
// meme denominateur et la meme couverture. Les enumerer a chaque test aurait fait diverger
// la liste au premier ajout — c'est exactement le defaut que `facesDeLaQuestion` a failli
// avoir, sa branche par defaut comptant les deux faces d'une mort pour une lecture qui n'en
// regarde aucune.
func lectureDArtefact(question string) bool {
	switch question {
	case domain.TacticalQuestionTemps, domain.TacticalQuestionRoutes:
		return true
	default:
		return false
	}
}
