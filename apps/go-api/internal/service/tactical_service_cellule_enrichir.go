package service

// tactical_service_cellule_enrichir.go — CE QUE LE DÉTAIL D'UNE ZONE AJOUTE À SES CONTRIBUTIONS : le
// nom en jeu de la zone, puis, pour chaque contribution, ce que la mini-tuile « Rejeu » affiche —
// mode, score, arme ou catégorie de la source, placement d'une mort, présence du rejeu.
//
// UNE LECTURE PAR SOURCE ET PAR REQUÊTE, jamais par contribution : la lecture canonique des matchs
// de la carte, les noms de toutes les clés d'armes en un appel, les contextes de mort bornés aux
// matchs des contributions (ADR 0036 I2), la présence des rejeux en un listing. Chacune a sa
// section de durée (ADR 0036 I6).
//
// CHAQUE SOURCE EST BEST-EFFORT : absente (non câblée, titre sans la donnée) ou en échec, elle
// retire le champ qu'elle nourrit et le journal le dit en la nommant ; la liste des contributions
// est toujours servie.

import (
	"context"
	"errors"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// Noms des sources du détail, tels que le journal les cite.
const (
	sourceCanonique = "canonique"
	sourceArmes     = "armes"
	sourceContextes = "contextes"
	sourceRejeu     = "rejeu"
)

// sourcesDuDetail : les dépendances du détail d'une zone, injectées par les `With*` de
// tactical_service_cablage.go. Une source nil retire son champ.
type sourcesDuDetail struct {
	// matchs, slug, gamertag : la lecture canonique des matchs du joueur (mode, score).
	matchs   port.PlayerMatchesRepository
	slug     string
	gamertag string
	// roundsDecide : game_variant_name → le résultat se lit en manches (regulation.toml, ADR 0032).
	roundsDecide map[string]bool
	// classifieur : source de dégât du film → clé du registre d'armes ; nil = titre sans
	// classificateur, qui ne publie ni arme ni catégorie.
	classifieur port.KillSourceClassifier
	armes       port.WeaponLabelResolver
	rejeu       port.ReplayService
}

// nommerLaCellule rend le nom en jeu de la zone de la cellule (règle V6, tactical.NommerZone) : au
// centre de la cellule au pas demandé, avec la hauteur des événements retenus (aucune pour les
// lectures d'artefact, dont les sidecars ne portent pas de z). Carte hors catalogue, ou aucune zone
// qui la nomme : nil — jamais un nom de repli.
func (s *TacticalService) nommerLaCellule(ctx context.Context, req domain.TacticalCelluleRequest,
	retenues []contributionLue) *domain.TacticalZoneNom {
	zones := s.zonesDeLaCarte(ctx, req.MapID)
	if len(zones) == 0 {
		return nil
	}
	x, y := grilleDemandee(req.PasM).Centre(tactical.Cellule{Col: req.Col, Lig: req.Lig})
	zs := make([]float64, 0, len(retenues))
	for _, c := range retenues {
		if c.z != nil {
			zs = append(zs, *c.z)
		}
	}
	retenue, ok := tactical.NommerZone(x, y, zs, zonesPures(zones))
	if !ok {
		s.logger.DebugContext(ctx, "tactique: zone sans nom (detail de cellule)",
			"player", s.xuid, "map_id", req.MapID, "col", req.Col, "lig", req.Lig, "hauteurs", len(zs))
		return nil
	}
	s.logger.DebugContext(ctx, "tactique: zone nommee (detail de cellule)",
		"player", s.xuid, "map_id", req.MapID, "col", req.Col, "lig", req.Lig,
		"regle", retenue.Regle, "distance_m", retenue.DistanceM, "hauteurs", len(zs))
	return &domain.TacticalZoneNom{NomFR: retenue.Zone.NomFR, NomEN: retenue.Zone.NomEN}
}

// enrichir rend les contributions publiées, chacune munie de ce que ses sources savent. Sans
// contribution, aucune source n'est lue.
func (s *TacticalService) enrichir(ctx context.Context, req domain.TacticalCelluleRequest,
	univers domain.TacticalUnivers, retenues []contributionLue) []domain.TacticalContribution {
	out := make([]domain.TacticalContribution, 0, len(retenues))
	for _, c := range retenues {
		out = append(out, c.TacticalContribution)
	}
	if len(out) == 0 {
		return out
	}
	s.poserModeEtScore(ctx, req.MapID, out)
	s.poserArmes(ctx, req.MapID, retenues, out)
	s.poserPlacements(ctx, req.MapID, univers, out)
	s.poserRejeu(ctx, req.MapID, out)
	return out
}

// sourceDegradee journalise une source qui ne nourrit pas son champ : DEBUG quand elle manque (non
// câblée, ou titre sans la donnée), WARN quand elle échoue.
func (s *TacticalService) sourceDegradee(ctx context.Context, source, mapID string, err error) {
	if err == nil || errors.Is(err, games.ErrCapabilityNotSupported) {
		s.logger.DebugContext(ctx, "tactique: detail de zone, source "+source+" absente",
			"player", s.xuid, "map_id", mapID, "err", err)
		return
	}
	s.logger.WarnContext(ctx, "tactique: detail de zone, source "+source+" en echec",
		"player", s.xuid, "map_id", mapID, "err", err)
}

// poserModeEtScore lit les matchs canoniques de la carte (une lecture, déjà en cache, ADR 0036 I3)
// et pose le mode, dans la langue de la requête, et le score de chaque contribution.
func (s *TacticalService) poserModeEtScore(ctx context.Context, mapID string, out []domain.TacticalContribution) {
	if s.detail.matchs == nil {
		s.sourceDegradee(ctx, sourceCanonique, mapID, nil)
		return
	}
	filtres := port.PlayerMatchFilters{MapIDs: []string{mapID}}
	if err := filtres.Validate(); err != nil {
		s.sourceDegradee(ctx, sourceCanonique, mapID, err)
		return
	}
	stop := timing.FromContext(ctx).Section("tactical_cellule_canonique")
	rows, err := s.detail.matchs.LoadPlayerMatches(ctx, s.detail.slug, s.detail.gamertag, filtres)
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceCanonique, mapID, err)
		return
	}
	parMatch := make(map[string]canonical.PlayerMatchRow, len(rows))
	for _, r := range rows {
		parMatch[r.Summary.MatchID] = r
	}
	locale := ctxkeys.Locale(ctx)
	for i := range out {
		r, ok := parMatch[out[i].MatchID]
		if !ok {
			continue
		}
		out[i].ModeLabel = labelPourLocale(r.Summary.PairMode, locale)
		out[i].ScoreLabel, out[i].ScoreKind = scoreDuMatch(r, s.detail.roundsDecide)
	}
}

// scoreDuMatch rend le score d'un match, MON camp d'abord (camp lu sur `Self.TeamID`), en manches
// sur une variante qui se décide aux manches (analysis.ReadTeamScore, ADR 0032), ou rien quand
// aucun score n'est connu.
func scoreDuMatch(r canonical.PlayerMatchRow, roundsDecide map[string]bool) (label, kind string) {
	var mien, autre *canonical.TeamSnapshot
	for i := range r.Summary.Teams {
		switch r.Summary.Teams[i].TeamID {
		case 0:
			mien = &r.Summary.Teams[i]
		case 1:
			autre = &r.Summary.Teams[i]
		}
	}
	if mien == nil || autre == nil {
		return "", ""
	}
	if r.Self.TeamID != nil && *r.Self.TeamID == 1 {
		mien, autre = autre, mien
	}
	variante := ""
	if r.Summary.GameVariant != nil {
		variante = strings.TrimSpace(r.Summary.GameVariant.DefaultLabel)
	}
	d, ok := analysis.ReadTeamScore(analysis.TeamScoreInput{
		MyPoints: mien.Score, EnemyPoints: autre.Score,
		MyRoundsWon: mien.RoundsWon, EnemyRoundsWon: autre.RoundsWon,
		RoundsTotal:  r.Summary.RoundsTotal,
		RoundsDecide: roundsDecide[variante],
	})
	if !ok {
		return "", ""
	}
	return analysis.FormatTeamScoreLabel(d), string(d.Kind)
}

// poserArmes nomme l'arme de chaque engagement par le registre du titre (source → clé →
// noms FR et EN, un appel pour toutes les clés) ; à défaut, publie la catégorie brute de la
// source. Titre sans classificateur : ni arme ni catégorie.
func (s *TacticalService) poserArmes(ctx context.Context, mapID string, retenues []contributionLue,
	out []domain.TacticalContribution) {
	if s.detail.classifieur == nil {
		return
	}
	cles := make([]string, len(retenues))
	distinctes := make([]string, 0, len(retenues))
	vues := make(map[string]bool, len(retenues))
	for i, c := range retenues {
		if c.sourceTag == nil {
			continue
		}
		if k, ok := s.detail.classifieur.KillSourceRegistryKey(*c.sourceTag); ok {
			cles[i] = k
			if !vues[k] {
				vues[k] = true
				distinctes = append(distinctes, k)
			}
		}
	}
	noms := s.nomsDesArmes(ctx, mapID, distinctes)
	for i, c := range retenues {
		if l, ok := noms[cles[i]]; ok && cles[i] != "" {
			out[i].ArmeLabel, out[i].ArmeLabelEN = l.Label, l.LabelEN
			continue
		}
		out[i].CategorieSource = c.categorie
	}
}

// nomsDesArmes lit les noms des clés, en un appel.
func (s *TacticalService) nomsDesArmes(ctx context.Context, mapID string, cles []string) map[string]port.WeaponLabel {
	if len(cles) == 0 {
		return nil
	}
	if s.detail.armes == nil {
		s.sourceDegradee(ctx, sourceArmes, mapID, nil)
		return nil
	}
	stop := timing.FromContext(ctx).Section("tactical_cellule_armes")
	noms, err := s.detail.armes.ResolveWeaponLabels(ctx, cles)
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceArmes, mapID, err)
		return nil
	}
	return noms
}

// poserPlacements pose le badge de placement de chaque MORT (D7) : la ligne de contexte la plus
// proche à ± TolerancePlacementMs, comparée à la portée du radar du match. Les contextes sont lus
// une fois, bornés aux matchs des morts retenues. Un frag, une entrée, une réapparition : aucun
// badge.
func (s *TacticalService) poserPlacements(ctx context.Context, mapID string, univers domain.TacticalUnivers,
	out []domain.TacticalContribution) {
	ids := matchsDesMorts(out)
	if len(ids) == 0 {
		return
	}
	stop := timing.FromContext(ctx).Section("tactical_cellule_contextes")
	contextes, err := s.repo.ContextesDeMort(ctx, domain.TacticalQuery{
		PlayerXUID: s.xuid, MapID: mapID, Matchs: domain.RestreindreAux(ids),
	})
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceContextes, mapID, err)
		return
	}
	rayons, _ := s.rayonsParMatch(univers.Matchs)
	for i := range out {
		if out[i].Face != domain.TacticalFaceMort {
			continue
		}
		rayon, aUnRayon := rayons[out[i].MatchID]
		c := tactical.ContexteLePlusProche(contextes, out[i].MatchID, out[i].XUID, out[i].InstantMs)
		out[i].Placement = tactical.PlacementDeLaMort(c, rayon, aUnRayon)
	}
}

// matchsDesMorts rend les matchs distincts des contributions de face « mort », triés.
func matchsDesMorts(out []domain.TacticalContribution) []string {
	vus := make(map[string]bool, len(out))
	ids := make([]string, 0, len(out))
	for _, c := range out {
		if c.Face == domain.TacticalFaceMort && !vus[c.MatchID] {
			vus[c.MatchID] = true
			ids = append(ids, c.MatchID)
		}
	}
	sort.Strings(ids)
	return ids
}

// poserRejeu dit, pour chaque contribution, si l'artefact de rejeu 2D de son match existe — un
// listing par requête (port.ReplayService.AvailableSet).
func (s *TacticalService) poserRejeu(ctx context.Context, mapID string, out []domain.TacticalContribution) {
	if s.detail.rejeu == nil {
		s.sourceDegradee(ctx, sourceRejeu, mapID, nil)
		return
	}
	stop := timing.FromContext(ctx).Section("tactical_cellule_rejeu")
	presents, err := s.detail.rejeu.AvailableSet(ctx)
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceRejeu, mapID, err)
		return
	}
	for i := range out {
		out[i].ReplayAvailable = presents.Has(out[i].MatchID)
	}
}
