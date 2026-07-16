// Package service — match_view_canonical.go : voie CANONIQUE de la Match View.
//
// buildMatchViewFromCanonical projette un canonical.MatchDetail (servi LIVE par
// un DataAdapter, ex. Halo 5 via la carnage) vers domain.MatchViewResponse. C'est
// le pendant de la voie repo (buildMatchViewFromData) quand le titre n'a PAS de
// substrat DuckDB pour le match (live-only). Le routage repo-first / adapter-
// fallback vit dans GetMatchView (match_view_service.go).
//
// Garanties :
//   - Aucune panique sur champs nil (Participants vides, Skill nil, Map nil) :
//     tout dégrade en chaînes/pointeurs vides.
//   - IsPartial = true systématiquement : la donnée live ne porte ni narratif de
//     combat, ni citations, ni médias, ni précision/dégâts subis natifs. Le front
//     affiche le bandeau « sync incomplet » existant et laisse les onglets riches
//     vides (le kill-feed h5 est servi par l'endpoint events séparé, supported).
//   - Le participant SELF est matché par GAMERTAG (ctxkeys.ViewerGamertag), PAS
//     par xuid : la carnage h5 est gamertag-keyée (Identity.XUID vide).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// PartialReasons émis par la voie canonique : sections indisponibles en live.
const (
	partialReasonCombatNarrative = "combat_narrative_unavailable"
	partialReasonCitations       = "citations_unavailable"
	partialReasonMedia           = "media_unavailable"
	partialReasonAccuracyDamage  = "accuracy_damage_taken_native_unavailable"
)

// assetNameResolver est une capability OPTIONNELLE du metadataRepo : résout les
// noms localisés d'assets (map/playlist/game_variant) depuis asset_translations,
// par lot. Seul l'adapter DuckDB réel (*duckdb.MetadataRepo) l'implémente ; un
// mock de test peut ne pas le faire → l'enrichissement est alors gracieusement
// ignoré (les refs gardent leur DefaultLabel). Même pattern que participantChecker.
type assetNameResolver interface {
	ResolveAssetNamesBulk(
		ctx context.Context, assetType string, assetIDs, preferredLangs []string,
	) (map[string]string, error)
}

// teamNameResolver est une capability OPTIONNELLE d'un TitleAssetURLAdapter : résout un
// team_id en libellé d'équipe localisé (Halo 5 : « Rouge »/« Red » depuis team_colors).
// Seul l'adapter H5 l'implémente ; HINF ne l'implémente pas → team_name reste vide et
// le front retombe sur sa résolution existante (Eagle/Cobra). Même pattern optionnel
// (type-assertion) que assetNameResolver / participantChecker : la capability n'élargit
// PAS l'interface games.TitleAssetURLAdapter (HINF n'a rien à stubber).
type teamNameResolver interface {
	TeamName(teamID int, locale string) string
}

// applyTeamNames renseigne row.TeamName sur chaque ligne de scoreboard quand l'adapter
// d'assets du titre expose teamNameResolver (Halo 5). No-op si l'adapter ne l'implémente
// pas (HINF), est nil, ou renvoie "" (team_colors vide) → dégradation gracieuse (le
// front garde son libellé d'équipe existant). Title-agnostic : aucune comparaison de
// slug, la capability seule décide. Appelée sur les DEUX voies (canonique live + repo
// persisté) car un match H5 peut emprunter l'une ou l'autre selon le substrat DuckDB.
func (s *MatchViewService) applyTeamNames(ctx context.Context, rows []domain.MatchScoreboardRow) {
	resolver, ok := s.assetURL.(teamNameResolver)
	if !ok {
		return
	}
	locale := ctxkeys.Locale(ctx)
	for i := range rows {
		id, ok := teamSideToID(rows[i].TeamSide)
		if !ok {
			continue
		}
		if name := resolver.TeamName(id, locale); name != "" {
			rows[i].TeamName = name
		}
	}
}

// teamSideToID parse le team_side DTO "t{N}" en son entier N. (0, false) si nil ou
// format inattendu (le backend émet toujours fmt.Sprintf("t%d", teamID)).
func teamSideToID(teamSide *string) (int, bool) {
	if teamSide == nil {
		return 0, false
	}
	s := *teamSide
	if len(s) < 2 || s[0] != 't' {
		return 0, false
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil {
		return 0, false
	}
	return n, true
}

// canonicalAssetLangs : ordre de préférence des langues d'asset_translations.
// FR-first (le projet est FR-first), EN ensuite. Aligné sur
// duckdb.PreferredLangsForLocale("fr") — répété ici pour éviter un import
// platform/duckdb depuis le service (frontière hexagonale).
var (
	canonicalAssetLangsFR = []string{"fr-FR", "fr", "en-US", "en"}
	canonicalAssetLangsEN = []string{"en-US", "en", "fr-FR", "fr"}
)

// enrichCanonicalDetailTranslations hydrate Labels["fr"]/Labels["en"] des
// AssetReference Map/Playlist/GameVariant du detail LIVE depuis asset_translations
// (metadataRepo). Sans cette passe, un detail servi live (Halo 5, pas de DB match)
// porte des refs brutes (ID seul, Labels=nil) → le header match view affiche un
// mode/carte/playlist vide. C'est le pendant LIVE de
// HomeRepo.EnrichCanonicalAssetTranslations (voie home/DB).
//
// No-op si le metadataRepo n'expose pas la capability (mock) ou si la table
// n'est pas peuplée (un autre agent fournit la donnée) : les refs gardent alors
// leur DefaultLabel. NO-OP fonctionnel sur Infinite : ce chemin n'est emprunté
// que par les titres sans substrat DuckDB (live-only).
func (s *MatchViewService) enrichCanonicalDetailTranslations(ctx context.Context, detail *canonical.MatchDetail) {
	resolver, ok := s.metadataRepo.(assetNameResolver)
	if !ok || detail == nil {
		return
	}
	enrichCanonicalAssetRefTranslations(ctx, resolver, "map", detail.Map)
	enrichCanonicalAssetRefTranslations(ctx, resolver, "playlist", detail.Playlist)
	enrichCanonicalAssetRefTranslations(ctx, resolver, "game_variant", detail.GameVariant)
}

// enrichCanonicalAssetRefTranslations résout FR + EN pour une ref unique et
// pose les labels in-place. N'écrase pas un label déjà présent et non vide.
func enrichCanonicalAssetRefTranslations(
	ctx context.Context, resolver assetNameResolver, assetType string, ref *canonical.AssetReference,
) {
	if ref == nil {
		return
	}
	id := strings.TrimSpace(ref.ID)
	if id == "" {
		return
	}
	ids := []string{id}
	frNames, err := resolver.ResolveAssetNamesBulk(ctx, assetType, ids, canonicalAssetLangsFR)
	if err != nil {
		slog.WarnContext(ctx, "match_view: résolution FR asset live échouée",
			"asset_type", assetType, "asset_id", id, "err", err)
	}
	enNames, err := resolver.ResolveAssetNamesBulk(ctx, assetType, ids, canonicalAssetLangsEN)
	if err != nil {
		slog.WarnContext(ctx, "match_view: résolution EN asset live échouée",
			"asset_type", assetType, "asset_id", id, "err", err)
	}
	setCanonicalLabelIfEmpty(ref, "fr", frNames[id])
	setCanonicalLabelIfEmpty(ref, "en", enNames[id])
}

// setCanonicalLabelIfEmpty pose Labels[locale]=name si name non vide ET que le
// label courant est vide (n'écrase jamais une valeur déjà résolue).
func setCanonicalLabelIfEmpty(ref *canonical.AssetReference, locale, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if ref.Labels == nil {
		ref.Labels = map[string]string{}
	}
	if strings.TrimSpace(ref.Labels[locale]) == "" {
		ref.Labels[locale] = name
	}
}

// buildMatchViewFromCanonical assemble la réponse Match View depuis un
// canonical.MatchDetail servi par un DataAdapter (voie LIVE). Header + Rank +
// Summary (self) + Team (scoreboard complet) sont remplis ; Combat / Media /
// Citations restent vides (servis ailleurs ou indisponibles live) et la réponse
// est marquée IsPartial.
func (s *MatchViewService) buildMatchViewFromCanonical(ctx context.Context, detail *canonical.MatchDetail) domain.MatchViewResponse {
	if detail == nil { // défense : le caller ne route ici que sur detail non-nil
		return domain.MatchViewResponse{IsPartial: true}
	}
	self := canonicalSelfParticipant(ctx, detail.Participants)
	comms := s.loadCanonicalCommendations(ctx, detail)
	citationsTab, citationsUnavailable := buildCanonicalCitationsTab(comms)
	teamTab := buildCanonicalTeamTab(detail, self)
	s.applyTeamNames(ctx, teamTab.Scoreboard) // Halo 5 : « Rouge/Bleu » si team_colors seedé
	return domain.MatchViewResponse{
		Header:         s.buildCanonicalHeader(detail, self),
		Rank:           buildCanonicalRank(detail.Skill),
		SummaryTab:     buildCanonicalSummaryTab(self),
		CombatTab:      domain.MatchCombatTab{},
		TeamTab:        teamTab,
		MediaTab:       domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{}},
		CitationsTab:   citationsTab,
		IsPartial:      true,
		PartialReasons: canonicalPartialReasons(citationsUnavailable),
	}
}

// canonicalSelfParticipant retrouve le participant du viewer (gamertag du ctx,
// comparaison insensible à la casse). Retourne nil si le ctx n'a pas de viewer ou
// si aucun participant ne correspond — header/summary self dégradent alors, mais
// le scoreboard reste complet.
func canonicalSelfParticipant(ctx context.Context, participants []canonical.MatchParticipant) *canonical.MatchParticipant {
	gt := strings.TrimSpace(ctxkeys.ViewerGamertag(ctx))
	if gt == "" {
		return nil
	}
	norm := strings.ToLower(gt)
	for i := range participants {
		if strings.ToLower(strings.TrimSpace(participants[i].Identity.Gamertag)) == norm {
			return &participants[i]
		}
	}
	return nil
}

// canonicalPartialReasons liste les sections indisponibles sur la voie live.
// citationsUnavailable=false quand des commendations natives sont présentes (Halo 5
// AXE B) → la section Citations n'est plus signalée comme manquante.
func canonicalPartialReasons(citationsUnavailable bool) []string {
	reasons := []string{partialReasonCombatNarrative}
	if citationsUnavailable {
		reasons = append(reasons, partialReasonCitations)
	}
	reasons = append(reasons, partialReasonMedia, partialReasonAccuracyDamage)
	return reasons
}

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

// buildCanonicalHeader remplit l'en-tête depuis le détail canonique + le self.
// PerfDisplay/Dominance restent neutres (pas de perf score en live).
func (s *MatchViewService) buildCanonicalHeader(detail *canonical.MatchDetail, self *canonical.MatchParticipant) domain.MatchViewHeader {
	h := domain.MatchViewHeader{
		MatchID:      detail.MatchID,
		OutcomeLabel: "-",
		OutcomeColor: mvHexOutcomeUnknown,
		PerfDisplay:  "-",
	}
	if !detail.StartedAtUTC.IsZero() {
		t := detail.StartedAtUTC
		h.StartTime = &t
		h.StartTimeLabel = formatDateFRLong(t)
	}
	mapLabel, mapID := assetLabelAndID(detail.Map)
	h.MapUI, h.MapID = mapLabel, mapID
	h.ModeUI = canonicalModeUI(detail)
	if playlistLabel, _ := assetLabelAndID(detail.Playlist); playlistLabel != "" {
		h.PlaylistLabel = playlistLabel
	}
	if detail.IsRanked != nil {
		h.IsRanked = *detail.IsRanked
	}
	h.PlayableDurationSeconds = canonicalGameplayDuration(detail)
	s.applyCanonicalMapImage(&h, detail.Map)
	applyCanonicalHeaderOutcome(&h, detail, self)
	return h
}

// canonicalModeUI privilégie la variante de jeu (GameVariant) puis la playlist.
func canonicalModeUI(detail *canonical.MatchDetail) string {
	if label, _ := assetLabelAndID(detail.GameVariant); label != "" {
		return label
	}
	label, _ := assetLabelAndID(detail.Playlist)
	return label
}

// canonicalGameplayDuration dérive la durée jouable (EndedAtUTC − StartedAtUTC)
// quand les deux sont connus. Nil sinon (durée non fabriquée).
func canonicalGameplayDuration(detail *canonical.MatchDetail) *int64 {
	if detail.EndedAtUTC == nil || detail.StartedAtUTC.IsZero() {
		return nil
	}
	secs := int64(detail.EndedAtUTC.Sub(detail.StartedAtUTC).Seconds())
	if secs < 0 {
		return nil
	}
	return &secs
}

// applyCanonicalMapImage résout MapImageURL via l'AssetURLAdapter. L'adapter h5
// résout par GUID (Map.ID), HINF par nom — on passe l'ID s'il existe, sinon le
// label. Dégradation gracieuse : assetURL nil ou résolution vide → champ absent.
func (s *MatchViewService) applyCanonicalMapImage(h *domain.MatchViewHeader, m *canonical.AssetReference) {
	if s.assetURL == nil || m == nil {
		return
	}
	key := strings.TrimSpace(m.ID)
	if key == "" {
		key = strings.TrimSpace(m.DefaultLabel)
	}
	if key == "" {
		return
	}
	if url := s.assetURL.MapImageURL(key); url != "" {
		h.MapImageURL = &url
	}
}

// applyCanonicalHeaderOutcome remplit Outcome* + ScoreLabel depuis le self + les
// équipes. ScoreLabel = "X - Y" avec le score de l'équipe du self en premier.
func applyCanonicalHeaderOutcome(h *domain.MatchViewHeader, detail *canonical.MatchDetail, self *canonical.MatchParticipant) {
	if self != nil {
		if code := outcomeCodeFromCanonical(self.Outcome); code != domain.OutcomeUnknown {
			c := code
			h.OutcomeCode = &c
			h.OutcomeLabel = outcomeLabel(code)
			h.OutcomeColor = outcomeColor(code)
			h.OutcomeColorToken = outcomeColorToken(code)
		}
	}
	h.ScoreLabel = canonicalScoreLabel(detail.Teams, self)
}

// canonicalScoreLabel construit "X - Y" depuis les TeamSnapshot. L'équipe du self
// (self.TeamID) est affichée en premier quand identifiable. Vide si < 2 équipes
// scorées (FFA / scores absents) — pas de label fabriqué.
func canonicalScoreLabel(teams []canonical.TeamSnapshot, self *canonical.MatchParticipant) string {
	scored := make([]canonical.TeamSnapshot, 0, len(teams))
	for _, t := range teams {
		if t.Score != nil {
			scored = append(scored, t)
		}
	}
	if len(scored) < 2 {
		return ""
	}
	myScore, otherScore, haveMine := 0, 0, false
	var others []int
	for _, t := range scored {
		if self != nil && self.TeamID != nil && t.TeamID == *self.TeamID && !haveMine {
			myScore, haveMine = *t.Score, true
			continue
		}
		others = append(others, *t.Score)
	}
	if haveMine && len(others) > 0 {
		otherScore = others[0]
		return fmt.Sprintf("%d - %d", myScore, otherScore)
	}
	// Self non identifiable dans les équipes : ordre brut des 2 premières scorées.
	return fmt.Sprintf("%d - %d", *scored[0].Score, *scored[1].Score)
}

// ---------------------------------------------------------------------------
// Rank
// ---------------------------------------------------------------------------

// buildCanonicalRank projette MatchSkillSnapshot vers le bloc Rank. Halo 5 ne
// charge pas le snapshot CSR par match (phase ultérieure) → Skill nil → bloc
// "none" vide (le front masque la pastille rang).
func buildCanonicalRank(skill *canonical.MatchSkillSnapshot) domain.MatchViewRank {
	if skill == nil {
		return domain.MatchViewRank{RatingType: "none"}
	}
	// Un MatchSkillSnapshot ne porte que les MMR/expected (pas de tier/value
	// affichable) ; on n'expose donc qu'un type neutre. La valeur affichable
	// (CSR pré/post) viendra d'un SkillSnapshot dédié en phase ultérieure.
	return domain.MatchViewRank{RatingType: "none"}
}

// ---------------------------------------------------------------------------
// Summary Tab (self)
// ---------------------------------------------------------------------------

// buildCanonicalSummaryTab remplit les KPIs perso depuis le participant self.
// Self nil (viewer absent / introuvable) → tab dégradé (KPIs vides, outcome "-").
func buildCanonicalSummaryTab(self *canonical.MatchParticipant) domain.MatchSummaryTab {
	tab := domain.MatchSummaryTab{
		KPIs:           domain.MatchSummaryKpis{},
		PersonalResult: domain.MatchPersonalResult{OutcomeLabel: "-", OutcomeColor: mvHexOutcomeUnknown},
		Medals:         []domain.MatchMedal{},
		Citations:      []domain.MatchCitationSnippet{},
		ExpectedStats:  domain.MatchExpectedStats{},
	}
	if self == nil {
		return tab
	}
	tab.KPIs = domain.MatchSummaryKpis{
		Kills:           self.Kills,
		Deaths:          self.Deaths,
		Assists:         self.Assists,
		KDA:             self.KDA,
		DamageDealt:     intPtrToFloatPtr(self.DamageDealt),
		AverageLife:     formatLifeSeconds(self.AvgLifeSeconds),
		HeadshotKills:   self.HeadshotKills,
		MaxKillingSpree: self.MaxKillingSpree,
		PerfectKills:    self.PerfectKills,
		Accuracy:        self.Accuracy,
		PersonalScore:   self.PersonalScore,
	}
	if code := outcomeCodeFromCanonical(self.Outcome); code != domain.OutcomeUnknown {
		score := 0
		if self.Score != nil {
			score = *self.Score
		}
		tab.PersonalResult = domain.MatchPersonalResult{
			OutcomeLabel:      outcomeLabel(code),
			OutcomeColor:      outcomeColor(code),
			OutcomeColorToken: outcomeColorToken(code),
			Score:             &score,
			RankInTeam:        self.RankInMatch,
		}
	}
	return tab
}

// ---------------------------------------------------------------------------
// Team Tab (scoreboard complet)
// ---------------------------------------------------------------------------

// buildCanonicalTeamTab projette tous les participants en lignes de scoreboard
// (mêmes champs JSON que buildTeamTabFull pour la cohérence de forme côté front).
// is_me est marqué sur le participant self (matché par gamertag). Roster/Nemesis/
// Encounters restent vides (non calculables sur la donnée live).
func buildCanonicalTeamTab(detail *canonical.MatchDetail, self *canonical.MatchParticipant) domain.MatchTeamTab {
	rows := make([]domain.MatchScoreboardRow, 0, len(detail.Participants))
	for i := range detail.Participants {
		p := &detail.Participants[i]
		rows = append(rows, canonicalScoreboardRow(p, self))
	}
	return domain.MatchTeamTab{
		Roster:     []domain.MatchRosterRow{},
		Scoreboard: rows,
		Nemesis:    []domain.MatchNemesisRow{},
		Encounters: []domain.MatchEncounterRow{},
	}
}

// canonicalScoreboardRow projette un participant canonique en ligne de scoreboard.
// Mêmes champs que buildTeamTabFull. Outcome label dégrade en "-" si inconnu.
func canonicalScoreboardRow(p *canonical.MatchParticipant, self *canonical.MatchParticipant) domain.MatchScoreboardRow {
	row := domain.MatchScoreboardRow{
		XUID:               p.Identity.XUID,
		Gamertag:           p.Identity.Gamertag,
		IsMe:               self != nil && p == self,
		IsBot:              p.IsBot != nil && *p.IsBot,
		Rank:               p.RankInMatch,
		Score:              p.PersonalScore,
		Kills:              p.Kills,
		Deaths:             p.Deaths,
		Assists:            p.Assists,
		KDA:                p.KDA,
		Accuracy:           p.Accuracy,
		DamageDealt:        intPtrToFloatPtr(p.DamageDealt),
		DamageTaken:        intPtrToFloatPtr(p.DamageTaken),
		ShotsFired:         p.ShotsFired,
		ShotsHit:           p.ShotsHit,
		AvgLifeSeconds:     p.AvgLifeSeconds,
		HeadshotKills:      p.HeadshotKills,
		MaxKillingSpree:    p.MaxKillingSpree,
		PerfectKills:       p.PerfectKills,
		GrenadeKills:       p.GrenadeKills,
		MeleeKills:         p.MeleeKills,
		PowerWeaponKills:   p.PowerWeaponKills,
		AssassinationKills: p.AssassinationKills,
		GroundPoundKills:   p.GroundPoundKills,
		ShoulderBashKills:  p.ShoulderBashKills,
		OutcomeLabel:       canonicalOutcomeLabel(p.Outcome),
	}
	if p.TeamID != nil {
		team := fmt.Sprintf("t%d", *p.TeamID)
		row.TeamSide = &team
	}
	return row
}

// canonicalOutcomeLabel : label FR de l'outcome canonique, "-" si inconnu.
func canonicalOutcomeLabel(o canonical.Outcome) string {
	if code := outcomeCodeFromCanonical(o); code != domain.OutcomeUnknown {
		return outcomeLabel(code)
	}
	return "-"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// outcomeCodeFromCanonical convertit canonical.Outcome (string) vers le code int
// Halo (domain.Outcome*). OutcomeUnknown (0) si non reconnu.
func outcomeCodeFromCanonical(o canonical.Outcome) int {
	switch o {
	case canonical.OutcomeWin:
		return domain.OutcomeWin
	case canonical.OutcomeLoss:
		return domain.OutcomeLoss
	case canonical.OutcomeTie:
		return domain.OutcomeDraw
	case canonical.OutcomeDNF:
		return domain.OutcomeDNF
	}
	return domain.OutcomeUnknown
}

// assetLabelAndID retourne (label FR si dispo sinon DefaultLabel, ID) d'une ref.
// ("", "") si ref nil.
func assetLabelAndID(ref *canonical.AssetReference) (string, string) {
	if ref == nil {
		return "", ""
	}
	label := ref.DefaultLabel
	if v, ok := ref.Labels["fr"]; ok && v != "" {
		label = v
	}
	return label, ref.ID
}

// intPtrToFloatPtr convertit un *int en *float64 (les champs scoreboard de dégâts
// sont des *float64 côté domain). Nil → nil.
func intPtrToFloatPtr(v *int) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v)
	return &f
}
