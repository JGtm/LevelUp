package service

// replay_spawn_forecast.go — LA PREVISION DE REAPPARITION DU REJEU 2D : pour chaque mort d'un match
// suivie d'une reapparition, la probabilite de chaque zone nommee, une fois par seconde du compte a
// rebours, puis le point reel.
//
// LE CALCUL EST PUR (analysis/tactical, PrevoirLesReapparitions) ; ce fichier le nourrit et en
// habille le resultat :
//
//	match -> carte, variante, categorie (registre partage)   port.ReplaySpawnRegistry
//	variante, categorie -> famille de modes                  spawn_families.toml (mappings)
//	famille -> poids du modele                               reference/spawn_model.json
//	carte -> reapparitions des AUTRES matchs                 sidecars de raster (replay_spawn_candidats.go)
//	carte -> zones nommees                                   cascade des callouts (zonesPourIdentites)
//	match -> pistes des joueurs et debuts de compte          artefact de rejeu
//
// UNE PREVISION NE VOIT PAS LA REPONSE : ni le sidecar du match prevu n'est lu, ni ses
// reapparitions ne comptent dans les candidats ou dans les frequences.
//
// AUCUN FILTRE PAR MODE : la famille de modes (spawn_families.toml) ne choisit que les poids (repli
// « tous ») et les matchs de la carte qui fournissent les candidats ; ce sont les seuils de donnees de
// la carte et de la famille (tactical.MatchsMinCartePrevision, ReapparitionsMinCartePrevision) qui
// decident si une prevision est servie.
//
// QUAND AUCUNE PREVISION N'A LIEU D'ETRE, LA REPONSE LE DIT : un modele absent, une carte trop pauvre
// sont servis en statut `unavailable` avec leur raison et un libelle dans la langue de la requete. Seuls
// l'absence de capability du titre (`film.spawn_forecast`) et l'absence de rejeu sont des refus.
//
// EN DEMO, les xuid sont masques par l'index des rejeux figes, comme le document de rejeu.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/replayartifacts"
)

// fichierFamillesReapparition est le nom du fichier de familles dans le dossier des mappings d'un
// titre.
const fichierFamillesReapparition = "spawn_families.toml"

// WithReplayCapabilities donne au service les capabilities du titre : sans `film.spawn_forecast`, la
// prevision de reapparition se refuse (games.ErrCapabilityNotSupported). Nil = aucune capability.
func WithReplayCapabilities(caps games.CapabilityMap) ReplayOption {
	return func(s *replayService) { s.caps = caps }
}

// famillesDeReapparition : la table des familles de modes de chaque titre, decodee une fois par
// processus (donnee versionnee, qui ne change qu'au deploiement).
var famillesDeReapparition = nouveauCacheParChemin(func(chemin string) (*mappings.SpawnFamilySet, error) {
	return mappings.LoadSpawnFamiliesFromFile(chemin)
})

// cadrePrevision est ce que la prevision d'un match determine avant de calculer : son identite
// dans le registre, sa famille de modes et les poids qui la servent.
type cadrePrevision struct {
	registre port.ReplaySpawnRegistry
	familles *mappings.SpawnFamilySet
	match    port.SpawnMatch
	famille  string
	modele   string
	// fichierPoids : le fichier de poids lu ; poids : le jeu retenu pour la famille (le jeu `modele`).
	fichierPoids domain.PoidsReapparition
	poids        [tactical.NbVariables]float64
}

// SpawnForecast sert la prevision de reapparition d'un match (cf. port.ReplayService).
func (s *replayService) SpawnForecast(ctx context.Context, matchID string) (replaydoc.SpawnForecast, error) {
	debut := time.Now()
	if !s.caps.Has(games.CapFilmSpawnForecast) {
		return replaydoc.SpawnForecast{}, games.ErrCapabilityNotSupported
	}
	if !s.IsAvailable(ctx, matchID) {
		return replaydoc.SpawnForecast{}, port.ErrReplayNotAvailable
	}
	cadre, raison, err := s.cadreDePrevision(ctx, matchID)
	if err != nil {
		return replaydoc.SpawnForecast{}, err
	}
	if raison != "" {
		return s.previsionIndisponible(ctx, raison, cadre), nil
	}
	calc, err := s.calculerLaPrevision(ctx, matchID, cadre)
	if err != nil {
		return replaydoc.SpawnForecast{}, err
	}
	out, err := s.habillerLaPrevision(ctx, calc, cadre)
	if err != nil {
		return replaydoc.SpawnForecast{}, err
	}
	slog.InfoContext(ctx, "rejeu 2D : prevision de reapparition",
		"match_id", matchID, "map_id", cadre.match.MapID, "famille", cadre.famille, "modele", cadre.modele,
		"statut", out.Status, "raison", out.Reason, "matchs_carte", out.MatchesOnMap, "candidats", out.Candidates,
		"morts", len(out.Deaths), "morts_ecartees", out.DiscardedDeaths,
		"sidecars_lus", calc.bilan.Lus, "sidecars_absents", calc.bilan.Absents, "sidecars_perimes", calc.bilan.Perimes,
		"sidecars_illisibles", calc.bilan.Illisibles, "titleSlug", s.titleSlug, "duration", time.Since(debut))
	return out, nil
}

// cadreDePrevision resout le registre, la famille et les poids du match. `raison` non vide : la
// prevision n'a pas lieu (le cadre porte ce qui est deja connu, pour la reponse). Une erreur n'est
// rendue que pour une panne de lecture du registre.
func (s *replayService) cadreDePrevision(ctx context.Context, matchID string) (cadrePrevision, string, error) {
	var c cadrePrevision
	reg, ok := s.maps.(port.ReplaySpawnRegistry)
	if !ok {
		slog.WarnContext(ctx, "prevision de reapparition : registre des matchs non cable",
			"match_id", matchID, "titleSlug", s.titleSlug)
		return c, replaydoc.SpawnForecastReasonMapUnknown, nil
	}
	c.registre = reg
	m, err := reg.SpawnMatchOf(ctx, matchID)
	if errors.Is(err, port.ErrMatchMapUnknown) {
		return c, replaydoc.SpawnForecastReasonMapUnknown, nil
	}
	if err != nil {
		return c, "", fmt.Errorf("prevision de reapparition, registre du match %s: %w", matchID, err)
	}
	c.match = m
	chemins := title.NewPathResolver(s.repoRoot)
	// Aucun mode n'est ecarte : sans table (titre sans fichier, ou fichier illisible), le match n'a pas
	// de famille et se sert avec les poids « tous » ; seuls les seuils de donnees decident.
	c.familles, err = famillesDeReapparition.lire(filepath.Join(chemins.TitleMappingsDir(s.titleSlug), fichierFamillesReapparition))
	if err != nil {
		c.familles = nil
		if !errors.Is(err, fs.ErrNotExist) {
			slog.ErrorContext(ctx, "prevision de reapparition : familles de modes illisibles, poids communs",
				"err", err, "titleSlug", s.titleSlug)
		}
	}
	c.famille = c.familles.FamilyOf(m.Variant, m.Category)
	c.fichierPoids, ok = ChargerPoidsReapparition(ctx, s.titleSlug, chemins.SpawnModelPath(s.titleSlug))
	if !ok {
		return c, replaydoc.SpawnForecastReasonNoModel, nil
	}
	c.modele, c.poids, ok = tactical.PoidsPourFamille(c.fichierPoids, c.famille)
	if !ok {
		return c, replaydoc.SpawnForecastReasonNoModel, nil
	}
	return c, "", nil
}

// previsionCalculee est le resultat pur de la prevision, avec ce qui l'a nourri et dont la reponse a
// besoin : les zones de la carte (pour les nommer) et le bilan de la lecture des sidecars.
type previsionCalculee struct {
	res   tactical.PrevisionsDuMatch
	zones []domain.ZoneNommee
	bilan bilanCandidats
}

// calculerLaPrevision nourrit le calcul pur : les reapparitions des autres matchs de la carte, les
// zones de la carte, les pistes du match.
func (s *replayService) calculerLaPrevision(ctx context.Context, matchID string,
	c cadrePrevision) (previsionCalculee, error) {
	matchs, err := c.registre.SpawnMatchesOnMap(ctx, c.match.MapID)
	if err != nil {
		return previsionCalculee{}, fmt.Errorf("prevision de reapparition, matchs de la carte %s: %w", c.match.MapID, err)
	}
	evts, bilan := s.reapparitionsDesAutresMatchs(ctx, matchs, c.familles, c.famille, c.match.MatchID)
	doc, err := replayartifacts.LireArtefactRange(s.artifactPath(matchID))
	if errors.Is(err, os.ErrNotExist) {
		return previsionCalculee{}, port.ErrReplayNotAvailable
	}
	if err != nil {
		return previsionCalculee{}, fmt.Errorf("prevision de reapparition, artefact %s: %w", matchID, err)
	}
	zones := s.zonesDeLaCartePrevue(ctx, matchID)
	res := tactical.PrevoirLesReapparitions(tactical.EntreePrevision{
		Match: replayartifacts.EntreeOccupationDeLArtefact(doc), MatchID: c.match.MatchID,
		Evenements: evts, Poids: c.poids, Zones: zonesPures(zones),
	})
	return previsionCalculee{res: res, zones: zones, bilan: bilan}, nil
}

// zonesDeLaCartePrevue rend les zones nommees de la carte du match (aucune : carte hors catalogue).
func (s *replayService) zonesDeLaCartePrevue(ctx context.Context, matchID string) []domain.ZoneNommee {
	keys := s.matchMapKeys(ctx, matchID)
	entry, ok := zonesPourIdentites(ctx, s.repoRoot, s.titleSlug, keys)
	if !ok {
		return nil
	}
	return zonesNommees(entry.Zones)
}

// raisonDuStatut traduit le statut du calcul pur en raison de la reponse.
func raisonDuStatut(statut string) string {
	switch statut {
	case tactical.PrevisionCartePauvre:
		return replaydoc.SpawnForecastReasonNotEnough
	case tactical.PrevisionSansZone:
		return replaydoc.SpawnForecastReasonNoZone
	case tactical.PrevisionAucuneReapparition:
		return replaydoc.SpawnForecastReasonNoRespawn
	}
	return ""
}

// previsionIndisponible sert la reponse d'une prevision qui n'a pas lieu, avec son libelle.
func (s *replayService) previsionIndisponible(ctx context.Context, raison string, c cadrePrevision) replaydoc.SpawnForecast {
	return replaydoc.SpawnForecast{
		Status: replaydoc.SpawnForecastStatusUnavailable, Reason: raison, Message: messageDeRaison(ctx, raison),
		Family: c.famille, Model: c.modele, SampleStepMs: tactical.PasPrevisionMs,
		Zones: []replaydoc.SpawnForecastZone{}, Deaths: []replaydoc.SpawnForecastDeath{},
	}
}

// messageDeRaison rend le libelle d'une raison, dans la langue de la requete.
func messageDeRaison(ctx context.Context, raison string) string {
	loc := i18n.FromContext(ctx)
	switch raison {
	case replaydoc.SpawnForecastReasonNoModel:
		return i18n.Msg(loc, "replay.forecast.no_model", nil)
	case replaydoc.SpawnForecastReasonMapUnknown:
		return i18n.Msg(loc, "replay.forecast.map_unknown", nil)
	case replaydoc.SpawnForecastReasonNotEnough:
		return i18n.Msg(loc, "replay.forecast.not_enough_matches", nil)
	case replaydoc.SpawnForecastReasonNoZone:
		return i18n.Msg(loc, "replay.forecast.no_zone", nil)
	case replaydoc.SpawnForecastReasonNoRespawn:
		return i18n.Msg(loc, "replay.forecast.no_respawn", nil)
	}
	return ""
}

// habillerLaPrevision projette le resultat pur sur le corps servi : libelles de zone dans la langue
// de la requete, coordonnees au centimetre, identites masquees en demo.
func (s *replayService) habillerLaPrevision(ctx context.Context, calc previsionCalculee,
	c cadrePrevision) (replaydoc.SpawnForecast, error) {
	res := calc.res
	if raison := raisonDuStatut(res.Statut); raison != "" {
		out := s.previsionIndisponible(ctx, raison, c)
		out.MatchesOnMap, out.Candidates, out.DiscardedDeaths = res.MatchsCarte, res.Candidats, res.MortsEcartees
		return out, nil
	}
	out := replaydoc.SpawnForecast{
		Status: replaydoc.SpawnForecastStatusReady, Family: c.famille, Model: c.modele, ModelDate: c.fichierPoids.Date,
		MatchesOnMap: res.MatchsCarte, Candidates: res.Candidats, DiscardedDeaths: res.MortsEcartees,
		SampleStepMs: tactical.PasPrevisionMs, Deaths: make([]replaydoc.SpawnForecastDeath, 0, len(res.Morts)),
	}
	for _, m := range res.Morts {
		out.Deaths = append(out.Deaths, mortServie(m))
	}
	out.Zones = zonesDeLaPrevision(res, calc.zones, i18n.FromContext(ctx))
	if s.demoIndexPath != "" {
		if err := masquerPrevisionDemo(s.demoIndexPath, &out); err != nil {
			slog.ErrorContext(ctx, "rejeu demo : masque des joueurs impossible, prevision refusee", "err", err)
			return replaydoc.SpawnForecast{}, err
		}
	}
	return out, nil
}

// mortServie projette la prevision d'une mort.
func mortServie(m tactical.PrevisionDeMort) replaydoc.SpawnForecastDeath {
	d := replaydoc.SpawnForecastDeath{
		XUID: m.XUID, DeathFrame: m.Mort.Frame, RespawnFrame: m.FrameReapparition,
		Death:   pointServi(m.Mort.X, m.Mort.Y, m.Mort.Z),
		Actual:  pointServi(m.Reel.X, m.Reel.Y, m.Reel.Z),
		Samples: make([]replaydoc.SpawnForecastSample, 0, len(m.Echantillons)),
	}
	if m.ZoneReelle >= 0 {
		z := m.ZoneReelle
		d.ActualZone = &z
	}
	for _, e := range m.Echantillons {
		smp := replaydoc.SpawnForecastSample{
			Frame: e.Frame, OutsideZone: e.HorsZone, Zones: make([]replaydoc.SpawnForecastZoneChance, 0, len(e.Zones)),
		}
		for _, z := range e.Zones {
			smp.Zones = append(smp.Zones, replaydoc.SpawnForecastZoneChance{Zone: z.Zone, P: z.P})
		}
		d.Samples = append(d.Samples, smp)
	}
	return d
}

func pointServi(x, y, z float64) replaydoc.SpawnForecastPoint {
	cm := func(v float64) float64 { return math.Round(v*100) / 100 }
	return replaydoc.SpawnForecastPoint{X: cm(x), Y: cm(y), Z: cm(z)}
}

// zonesDeLaPrevision nomme, dans la langue de la requete, les zones que les previsions et les
// points reels designent. Une zone designee que le catalogue ne nomme plus n'est pas servie.
func zonesDeLaPrevision(res tactical.PrevisionsDuMatch, zones []domain.ZoneNommee, loc i18n.Locale) []replaydoc.SpawnForecastZone {
	vues := map[int]bool{}
	for _, m := range res.Morts {
		if m.ZoneReelle >= 0 {
			vues[m.ZoneReelle] = true
		}
		for _, e := range m.Echantillons {
			for _, z := range e.Zones {
				vues[z.Zone] = true
			}
		}
	}
	out := make([]replaydoc.SpawnForecastZone, 0, len(vues))
	for _, z := range zones {
		if vues[z.VolumeIndex] {
			out = append(out, replaydoc.SpawnForecastZone{VolumeIndex: z.VolumeIndex, Name: z.Nom.Pick(loc)})
			delete(vues, z.VolumeIndex)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].VolumeIndex < out[b].VolumeIndex })
	return out
}

// masquerPrevisionDemo remplace les xuid reels par ceux de la demo (index des rejeux figes).
func masquerPrevisionDemo(indexPath string, f *replaydoc.SpawnForecast) error {
	index, err := lireIndexDemo(indexPath)
	if err != nil {
		return err
	}
	demo := make(map[string]string, len(index.Identities))
	for _, id := range index.Identities {
		if id.XUID != "" {
			demo[id.XUID] = id.DemoXUID
		}
	}
	for i := range f.Deaths {
		if r, ok := demo[f.Deaths[i].XUID]; ok {
			f.Deaths[i].XUID = r
		}
	}
	return nil
}
