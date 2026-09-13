package service

// tactical_service_isolement.go — LA LECTURE « OU JE MEURS ISOLE ».
//
// # ELLE NE DEDUIT PLUS RIEN, ELLE COMPARE
//
// Le fait est etabli AU SYNC (`match_death_context`, lot 7C) : chaque mort du journal y porte
// combien de coequipiers pouvaient accompagner et a quelle distance etait le plus proche de ceux
// qu'on VOYAIT. Ce fichier joint les equipes, ne garde que les morts de la CIBLE, resout le
// rayon de la variante DU MATCH, et laisse `coordination.Isolement` trancher.
//
// La version precedente deduisait la vitalite des coequipiers a la lecture, sur la chronologie
// d'un artefact de rejeu. Elle demandait au film ce qu'il ne sait pas dire (`replay/owners.go`
// nomme aussi une vie par FERMETURE DE SLOT) et faisait dependre un fait de base du calendrier
// de cuisson. Les deux defauts disparaissent avec elle.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// rasterIsole sert « ou je meurs isole ».
//
// ELLE PASSE PAR LA MEME PORTE QUE LES LECTURES DE PLACEMENT (`film.kill_positions`) : ses
// morts viennent de `kill_positions` pour le lieu et de `match_death_context` pour le
// voisinage, et les deux naissent de la meme passe du collecteur.
func (s *TacticalService) rasterIsole(ctx context.Context, out *domain.TacticalRaster,
	scope domain.TacticalScope,
) error {
	if !positionsDeKillLisibles(s.caps) {
		s.logger.WarnContext(ctx, "tactique: aucune position de kill lisible pour ce titre",
			"player", s.xuid, "map_id", out.MapID, "question", out.Question)
		return games.ErrCapabilityNotSupported
	}
	debut := time.Now()

	lecture, err := s.repo.MortsAvecContexte(ctx, requeteDuScope(s.xuid, out.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: lecture d'isolement en echec",
			"player", s.xuid, "map_id", out.MapID, "err", err)
		return err
	}
	if len(lecture.Univers.Matchs) == 0 {
		// SENTINELLE NUE, meme raison que les autres lectures : le message est publie tel
		// quel, et y citer la carte distinguerait ce 404 de celui d'un map_id refuse par la
		// validation du handler.
		s.logger.InfoContext(ctx, "tactique: carte sans match retenu (isolement)",
			"player", s.xuid, "map_id", out.MapID, "qui", out.Qui)
		return domain.ErrTacticalCarteInconnue
	}
	out.MatchsFiltres = len(lecture.Univers.Matchs)

	rayons, sansRayon := s.rayonsParMatch(lecture.Univers.Matchs)
	dans := cible(lecture.Univers.Equipes, out.Qui, s.xuid, scope.Coequipiers)
	morts := mortsDeLaCible(lecture, dans)
	bilan := coordination.Isolement(morts, rayons, len(rayons))
	bilan.MatchsSansRayon = sansRayon
	// LA SECTION « COORDINATION » SORT DE LA MEME LECTURE, jamais d'une seconde requete :
	// c'est la meme table de morts, deja en main.
	out.Coordination = construireCoordination(morts, rayons)

	// L'UNIVERS DE CETTE LECTURE EST CELUI DES MATCHS AYANT UN RAYON. Rasteriser sur tous
	// les matchs mesures diviserait les cellules par des matchs qu'on a refuse de lire —
	// c'est le defaut deja corrige trois fois sous « correction G2 ».
	out.MatchsRetenus = len(rayons)
	out.MatchsSansRayon = bilan.MatchsSansRayon
	out.MortsEquipeATerre = bilan.EquipeATerre
	cov := bilan.Couverture
	out.Isolement = &cov

	// LES MORTS SE REPROJETTENT A CHAQUE PAS ESSAYE, elles ne se regroupent pas : leurs
	// positions sont en main (contrairement aux sidecars, deja agreges), et projeter la
	// position exacte est plus juste que regrouper une cellule fine.
	lue, err := rasteriserComptes(matchsAyantUnRayon(rayons), func(g tactical.Grille) []tactical.CompteCellule {
		return comptesDesMortsIsolees(g, bilan.Isolees)
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: rasterisation des morts isolees en echec",
			"player", s.xuid, "map_id", out.MapID, "err", err)
		return fmt.Errorf("tactique: rasterisation des morts isolees: %w", err)
	}
	// `ignores` VAUT ZERO ET C'EST STRUCTUREL : cette lecture ne lit aucun sidecar, donc
	// aucun point n'a pu etre ecarte a la cuisson.
	remplirDepuisSidecars(out, lue.Raster, 0)
	s.lireLeJournal(ctx, out, scope)

	s.logger.InfoContext(ctx, "tactique: lecture d'isolement",
		"player", s.xuid, "map_id", out.MapID, "qui", out.Qui,
		"matchs_filtres", out.MatchsFiltres, "matchs_retenus", out.MatchsRetenus,
		"matchs_sans_rayon", out.MatchsSansRayon, "morts_equipe_a_terre", out.MortsEquipeATerre,
		"isolees", bilan.Couverture.Brut, "examinees", bilan.Couverture.N,
		"duration", time.Since(debut))
	return nil
}

// mortsDeLaCible ne garde que les morts des joueurs de l'axe « qui ».
//
// LA JOINTURE DES EQUIPES SE FAIT ICI, ET C'EST TOUT CE QUE LA LECTURE AJOUTE AU FAIT : le
// collecteur a compte les coequipiers de CHAQUE victime, sans savoir laquelle interesse la page.
func mortsDeLaCible(lecture domain.TacticalMortsContexte, dans predicatQui) []domain.MortAExaminer {
	out := make([]domain.MortAExaminer, 0, len(lecture.Morts))
	for i := range lecture.Morts {
		m := &lecture.Morts[i]
		if !dans(m.MatchID, m.VictimXUID) {
			continue
		}
		out = append(out, domain.MortAExaminer{
			MatchID: m.MatchID, X: m.X, Y: m.Y,
			PlusProcheM: m.PlusProcheM, Visibles: m.Visibles, HorsDeVue: m.HorsDeVue,
		})
	}
	return out
}

// comptesDesMortsIsolees rend une occurrence par mort isolee, dans sa cellule.
func comptesDesMortsIsolees(g tactical.Grille, isolees []domain.MortAExaminer) []tactical.CompteCellule {
	out := make([]tactical.CompteCellule, 0, len(isolees))
	for _, m := range isolees {
		c, ok := g.Cellule(m.X, m.Y)
		if !ok {
			continue
		}
		out = append(out, tactical.CompteCellule{Cellule: c, MatchID: m.MatchID, Occurrences: 1})
	}
	return out
}

// matchsAyantUnRayon rend l'univers de rasterisation : les matchs dont la variante a une portee.
func matchsAyantUnRayon(rayons map[string]float64) []string {
	out := make([]string, 0, len(rayons))
	for id := range rayons {
		out = append(out, id)
	}
	return out
}

// rayonsParMatch resout la portee du radar de chaque match MESURE, par sa variante, et compte
// ceux qui n'en ont pas.
//
// UN MATCH DONT LA VARIANTE N'EST PAS DANS LA TABLE N'ENTRE PAS DANS LA TABLE DE SORTIE, et il
// sort donc de l'UNIVERS de la lecture — pas seulement de ses numerateurs. Le laisser au
// denominateur diviserait la mesure par des matchs qu'on avait refuse de lire : deux matchs dont
// un Husky Raid rendraient 0,5 mort isolee par match au lieu de 1.
//
// LE NOM EST NETTOYE, ET CE N'EST PAS DE LA COURTOISIE : il vient de
// `match_registry.game_variant_name`, donc de ce que l'API a envoye, et des variantes y arrivent
// avec un blanc de tete ou de queue. Une cle non nettoyee manque la table, et le match sort
// SILENCIEUSEMENT de la lecture — un defaut de donnee deguise en trou de referentiel, qui envoie
// chercher la panne au mauvais endroit.
func (s *TacticalService) rayonsParMatch(matchs []domain.TacticalMatch) (map[string]float64, int) {
	out := make(map[string]float64, len(matchs))
	sans := 0
	for _, m := range matchs {
		if !m.Mesure {
			continue
		}
		metres, ok := s.radar[strings.TrimSpace(m.GameVariantName)]
		if !ok || metres <= 0 {
			sans++
			continue
		}
		out[m.MatchID] = float64(metres)
	}
	return out, sans
}

// construireCoordination assemble la section « Coordination d'equipe » : la FORME de la
// distance a l'equipier (mediane, distribution) et les denominateurs que la note nomme.
//
// LES RAYONS SORTENT DISTINCTS ET TRIES, jamais moyennes : un filtre qui melange Arene
// (18 m) et BTB (24 m) melange DEUX REGLES DU JEU, et la moyenne des deux n'est la regle
// d'aucun match. Le web pose alors deux seuils sur l'histogramme.
func construireCoordination(morts []domain.MortAExaminer,
	rayons map[string]float64,
) *domain.TacticalCoordination {
	d := coordination.Distances(morts, rayons)
	return &domain.TacticalCoordination{
		DistanceMedianeM:       d.Mediane,
		Distribution:           d.Distribution,
		NDistances:             d.N,
		MortsSansDistance:      d.MortsSansDistance,
		RayonsM:                rayonsDistincts(rayons),
		MatchsMesures:          len(rayons),
		FenetreEchangeSecondes: int(coordination.FenetreEchangeMs / 1000),
	}
}

// rayonsDistincts rend les portees DISTINCTES des matchs lus, triees croissant.
func rayonsDistincts(rayons map[string]float64) []float64 {
	vus := make(map[float64]bool, 2)
	out := make([]float64, 0, 2)
	for _, r := range rayons {
		if vus[r] {
			continue
		}
		vus[r] = true
		out = append(out, r)
	}
	sort.Float64s(out)
	return out
}

// mesurerCoordination sert la section « Coordination d'equipe » pour les lectures QUI NE
// SONT PAS « ou je meurs isole ».
//
// POURQUOI UNE LECTURE DE PLUS, ET POURQUOI ELLE EST LEGITIME : la section est affichee
// sur TOUTES les questions (maquette 034b1915) — savoir ou l'on meurt et savoir si l'on y
// meurt seul sont deux reponses a la meme question de placement. C'est exactement le
// regime deja en place pour le KPI d'ECHANGE (`lireLeJournal`), qui lit lui aussi le
// journal des morts a chaque raster.
//
// LE TAUX D'ISOLEMENT EST POSE ICI AUSSI : il ne depend pas de la question, seulement des
// morts du joueur sur la carte. Le reserver a la question « isole » obligeait a changer de
// question pour lire un chiffre qui ne change pas.
//
// UN ECHEC EST JOURNALISE PUIS DEGRADE : la lecture de placement reste servie, la section
// reste silencieuse. Aucune erreur avalee.
func (s *TacticalService) mesurerCoordination(ctx context.Context, out *domain.TacticalRaster,
	scope domain.TacticalScope,
) {
	if !positionsDeKillLisibles(s.caps) {
		return
	}
	lecture, err := s.repo.MortsAvecContexte(ctx, requeteDuScope(s.xuid, out.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: coordination non servie (lecture des morts en echec)",
			"player", s.xuid, "map_id", out.MapID, "question", out.Question, "err", err)
		return
	}
	if len(lecture.Univers.Matchs) == 0 {
		return
	}
	rayons, sansRayon := s.rayonsParMatch(lecture.Univers.Matchs)
	dans := cible(lecture.Univers.Equipes, out.Qui, s.xuid, scope.Coequipiers)
	morts := mortsDeLaCible(lecture, dans)
	bilan := coordination.Isolement(morts, rayons, len(rayons))
	cov := bilan.Couverture
	out.Isolement = &cov
	out.MatchsSansRayon = sansRayon
	out.MortsEquipeATerre = bilan.EquipeATerre
	out.Coordination = construireCoordination(morts, rayons)
}
