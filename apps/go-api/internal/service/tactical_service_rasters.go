// Package service — tactical_service_rasters.go : LA LECTURE D'OCCUPATION (« ou je passe
// mon temps »), sommee depuis les sidecars de raster.
//
// Fichier separe de tactical_service.go (deja a 381 lignes) et de
// tactical_service_perimetre.go : celui-la dit QUELS matchs, celui-ci dit CE QU'ON Y
// MESURE quand la mesure ne vient pas de la base.
//
// ─── LA SEULE LECTURE DU CHANTIER QUI NE VIENT PAS DE SQL ──────────────────────
//
// Les trois questions de la phase 2 (morts, kills, gagne) se lisent sur
// `kill_positions_latest`. L'occupation, elle, se lit sur les PISTES du film — donc sur
// des sidecars deposes A LA CUISSON, un par match (cf.
// internal/sync/replayartifacts/raster.go). La base n'intervient ici que pour L'UNIVERS :
// quels matchs, et qui etait dans quelle equipe.
//
// AUCUN CACHE D'AGREGAT, AUCUNE INVALIDATION (decision produit du 2026-09-05). La page
// somme N fichiers immuables a chaque affichage. Un cache par filtre aurait autant
// d'entrees que de combinaisons de la barre L2, chacune a invalider au match suivant.
//
// ─── UN SIDECAR ABSENT EST UN MATCH NON MESURE, JAMAIS UN MATCH A ZERO ─────────
//
// C'est la meme regle que le drapeau `Mesure` du journal des morts (correction G2 du
// 2026-09-06), appliquee a l'autre substrat. Un match dont le film n'a jamais ete decode
// ne peut alimenter aucune cellule : le compter au denominateur ferait varier l'intensite
// avec la COUVERTURE DE FILM au lieu du jeu. Il compte dans `matchs_filtres`, pas dans
// `matchs_retenus`.
//
// ─── ELLE NE CUIT RIEN ──────────────────────────────────────────────────────────
//
// Un sidecar manquant n'est jamais fabrique a la volee : il le sera par le fil de l'eau au
// prochain cycle, ou par `levelup tactical-rasters --backfill`. Decoder un film sur la
// requete d'un visiteur est exactement ce que le ratchet
// archlint/no_cuisson_depuis_tactique_test.go rend impossible.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// ─── LE STORE : LA SEULE LECTURE DE FICHIER ────────────────────────────────────

// tacticalRasterStore lit les sidecars d'un titre depuis le disque, par PathResolver.
type tacticalRasterStore struct {
	repoRoot  string
	titleSlug string
}

// NewTacticalRasterStore construit le lecteur de sidecars d'un titre.
func NewTacticalRasterStore(repoRoot, titleSlug string) port.TacticalRasterStore {
	return &tacticalRasterStore{repoRoot: repoRoot, titleSlug: titleSlug}
}

// Charger lit le sidecar d'un match. ABSENT = (nil, nil) : ce n'est pas une erreur, c'est
// un match non mesure (cf. l'en-tete). Un fichier present mais illisible, LUI, est une
// erreur — il ne doit pas se confondre en silence avec une absence.
func (s *tacticalRasterStore) Charger(_ context.Context, matchID string) (*domain.TacticalRasterSidecar, error) {
	path := title.NewPathResolver(s.repoRoot).TacticalRasterPath(s.titleSlug, matchID)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture du sidecar de raster %s: %w", matchID, err)
	}
	var sc domain.TacticalRasterSidecar
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, fmt.Errorf("desérialisation du sidecar de raster %s: %w", matchID, err)
	}
	return &sc, nil
}

// ─── LA LECTURE ────────────────────────────────────────────────────────────────

// rasterOccupation orchestre la lecture « ou je passe mon temps » : sa porte, son
// univers, sa somme, son journal.
//
// ELLE NE PASSE PAS PAR `KillPositions` (contrairement aux trois autres questions) : ses
// valeurs ne viennent pas de la base, seulement son univers. Scanner les positions de kill
// de toute la carte pour en jeter le resultat aurait ete payer la lecture qu'on ne fait
// pas.
func (s *TacticalService) rasterOccupation(ctx context.Context, out *domain.TacticalRaster,
	scope domain.TacticalScope) error {
	if !s.caps.Has(games.CapFilmReplayArtifact) {
		s.logger.WarnContext(ctx, "tactique: occupation indisponible — le titre ne produit pas d'artefact de rejeu",
			"player", s.xuid, "map_id", out.MapID, "capability", string(games.CapFilmReplayArtifact))
		return games.ErrCapabilityNotSupported
	}
	debut := time.Now()
	univers, err := s.repo.Univers(ctx, requeteDuScope(s.xuid, out.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: univers de l'occupation en echec",
			"player", s.xuid, "map_id", out.MapID, "err", err)
		return err
	}
	if len(univers.Matchs) == 0 {
		// SENTINELLE NUE, meme raison que la lecture de placement : le message est publie
		// tel quel, et y citer la carte distinguerait ce 404 de celui d'un map_id refuse
		// par la validation du handler.
		s.logger.InfoContext(ctx, "tactique: carte sans match retenu (occupation)",
			"player", s.xuid, "map_id", out.MapID, "qui", out.Qui)
		return domain.ErrTacticalCarteInconnue
	}
	// MATCHS FILTRES = L'UNIVERS DE CETTE CARTE ; MatchsRetenus (pose par
	// lectureOccupation) en est le SOUS-ENSEMBLE MESURE — ceux dont le sidecar existe.
	out.MatchsFiltres = len(univers.Matchs)
	if err := s.lectureOccupation(ctx, out, univers, scope); err != nil {
		return err
	}
	// LE KPI D'ECHANGE EST CELUI DE LA CARTE, PAS CELUI DE LA QUESTION : il est servi sous
	// les quatre lectures, avec le meme perimetre (mon camp entier). La couverture
	// d'evenements, elle, rend 0 ici — l'occupation ne lit aucun journal des morts, et
	// `facesDeLaQuestion` le dit.
	s.lireLeJournal(ctx, out, scope)
	s.logger.InfoContext(ctx, "tactique: lecture d'occupation",
		"player", s.xuid, "map_id", out.MapID, "qui", out.Qui,
		"matchs_filtres", out.MatchsFiltres, "matchs_retenus", out.MatchsRetenus,
		"coequipiers", len(scope.Coequipiers), "cellules", len(out.Cellules),
		"duration", time.Since(debut))
	return nil
}

// lectureOccupation remplit `out` avec la somme des sidecars de l'univers.
//
// LE DENOMINATEUR EST LE NOMBRE DE MATCHS MESURES, c'est-a-dire ceux dont le sidecar est
// present ET exploitable. Il est passe a `RasteriseComptes` comme univers, exactement
// comme les points le sont pour les trois autres lectures : un match mesure dont la cible
// n'a produit aucune cellule est un zero LEGITIME et compte au denominateur.
func (s *TacticalService) lectureOccupation(ctx context.Context, out *domain.TacticalRaster,
	univers domain.TacticalUnivers, scope domain.TacticalScope) error {
	if s.rasters == nil {
		// DEFAUT DE CABLAGE, PAS UN ETAT DE TITRE : la capability est declaree, mais
		// aucun lecteur de sidecars n'a ete injecte. Il se dit fort, et la lecture
		// degrade proprement plutot que de rendre une carte vide qui se lirait
		// « il ne se passe rien ici ».
		s.logger.ErrorContext(ctx, "tactique: occupation demandee sans lecteur de sidecars cable",
			"player", s.xuid, "map_id", out.MapID)
		return games.ErrCapabilityNotSupported
	}
	dans := cible(univers.Equipes, out.Qui, s.xuid, scope.Coequipiers)
	mesures := make([]string, 0, len(univers.Matchs))
	comptes := make([]tactical.CompteCellule, 0, len(univers.Matchs)*32)
	ignores := 0
	for _, m := range univers.Matchs {
		sc, err := s.rasters.Charger(ctx, m.MatchID)
		if err != nil {
			// Sidecar present mais illisible : signale PUIS degrade en « non mesure ».
			// Une erreur avalee ferait passer un fichier corrompu pour une absence.
			s.logger.ErrorContext(ctx, "tactique: sidecar de raster illisible",
				"player", s.xuid, "map_id", out.MapID, "match_id", m.MatchID, "err", err)
			continue
		}
		if !s.sidecarExploitable(ctx, sc, m.MatchID) {
			continue
		}
		mesures = append(mesures, m.MatchID)
		// LES POINTS IGNORES VIENNENT DU FICHIER, ils ne se recalculent pas : la somme
		// part de comptes deja groupes par cellule, et un point ecarte n'a jamais eu de
		// cellule. Ils sont comptes PAR MATCH MESURE — un sidecar qu'on n'a pas retenu ne
		// doit pas alourdir la statistique d'un decodage qu'on n'a pas lu.
		ignores += sc.PointsIgnores
		comptes = append(comptes, comptesDuSidecar(sc, m.MatchID, dans)...)
	}
	out.MatchsRetenus = len(mesures)
	raster, err := tactical.RasteriseComptes(tactical.GrilleParDefaut(), mesures, comptes)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: somme des rasters en echec",
			"player", s.xuid, "map_id", out.MapID, "err", err)
		return fmt.Errorf("tactique: somme des rasters: %w", err)
	}
	remplirOccupation(out, raster, ignores)
	return nil
}

// sidecarExploitable dit si un sidecar peut entrer dans la somme, et DIT POURQUOI quand
// la reponse est non.
//
// LES DEUX VERIFICATIONS SONT DES UNITES, PAS DE LA PARANOIA. Sommer des rasters de pas
// differents melangerait deux resolutions de grille ; sommer des comptes echantillonnes a
// deux pas differents melangerait deux unites de temps sous le meme nom. Dans les deux
// cas le resultat serait faux SANS RIEN CASSER — le pire mode de defaillance. Un sidecar
// ecarte n'est pas une erreur de lecture : c'est un match non mesure, qui sera refait par
// `levelup tactical-rasters --backfill`.
func (s *TacticalService) sidecarExploitable(ctx context.Context,
	sc *domain.TacticalRasterSidecar, matchID string) bool {
	if sc == nil {
		// L'ABSENCE EST LE CAS NORMAL ET MAJORITAIRE (film jamais decode, ou expire) :
		// elle ne se journalise pas, elle se COMPTE — l'ecart entre matchs_filtres et
		// matchs_retenus la dit deja, en clair, au pied de la carte.
		return false
	}
	if sc.SchemaVersion != domain.TacticalRasterSchemaVersion ||
		sc.PasM != tactical.PasParDefautM ||
		sc.PasEchantillonMs != tactical.PasOccupationMs {
		s.logger.WarnContext(ctx, "tactique: sidecar de raster ecarte (format ou unites d'un autre temps)",
			"match_id", matchID, "schema_version", sc.SchemaVersion, "pas_m", sc.PasM,
			"pas_echantillon_ms", sc.PasEchantillonMs,
			"remede", "levelup tactical-rasters --backfill")
		return false
	}
	return true
}

// comptesDuSidecar rend les cellules du sidecar qui appartiennent a la cible.
//
// L'AXE « QUI » SE TRANCHE ICI, A LA LECTURE, et non a la cuisson : le sidecar est
// anonyme (xuid seul, aucune equipe), si bien qu'un changement de camp ou de composition
// ne perime aucun fichier.
func comptesDuSidecar(sc *domain.TacticalRasterSidecar, matchID string,
	dans predicatQui) []tactical.CompteCellule {
	out := make([]tactical.CompteCellule, 0, 64)
	for _, j := range sc.Joueurs {
		if !dans(matchID, j.XUID) {
			continue
		}
		for _, c := range j.Cellules {
			out = append(out, tactical.CompteCellule{
				Cellule:     tactical.Cellule{Col: c.Col, Lig: c.Lig},
				MatchID:     matchID,
				Occurrences: c.Echantillons,
			})
		}
	}
	return out
}

// remplirOccupation habille la reponse : cellules EN SECONDES, echelle, cadre.
//
// L'ORDRE COMPTE : l'echelle se calcule APRES la conversion, sur les valeurs qui seront
// peintes. Des quantiles calcules sur des comptes d'echantillons puis affiches en face de
// secondes decrirait une autre grandeur que celle qu'on regarde.
//
// LE COMPTE BRUT RESTE EN ECHANTILLONS : c'est la mesure, la seconde n'en est que l'unite
// de lecture (doctrine « jamais un taux seul » — la valeur est servie AVEC son brut).
func remplirOccupation(out *domain.TacticalRaster, raster *tactical.Raster, ignores int) {
	out.Cellules = tactical.EnSecondes(raster.Cellules(), tactical.PasOccupationMs)
	out.Echelle = tactical.Echelle(out.Cellules)
	out.PasM = raster.PasM()
	out.Bornes = raster.Bornes()
	// `raster.PointsIgnores()` vaudrait TOUJOURS 0 ici : `RasteriseComptes` ne recoit que
	// des cellules, jamais un point qui aurait pu etre ecarte. Le compte vient donc des
	// sidecars, qui l'ont mesure a la cuisson.
	out.PointsIgnores = ignores
}
