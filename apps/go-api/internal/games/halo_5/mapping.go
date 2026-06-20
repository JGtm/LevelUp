// Package halo_5 — mapping.go : projection PURE des DTO Halo 5 vers le canonique.
//
// Fonctions stateless (zero IO, testables sans reseau) — c'est ici que vivent les
// divergences Halo 5 vs Infinite documentees dans le handoff §0-ter :
//   - identite GAMERTAG-keyee (Player.Xuid toujours null) ;
//   - Outcome derive du Rank (1 = vainqueur), PAS de l'enum Result deviné ;
//   - MatchDuration ISO8601 "PT..." a parser ;
//   - MatchCompletedDate = FIN du match (StartedAtUTC = fin − duree) ;
//   - CSR natif DesignationId (palier majeur) + Tier (sous-palier).
package halo_5

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/classification"
)

// h5Identity construit l'identite canonique d'un joueur Halo 5. XUID vide
// (jamais fourni par l'API h5) ; indexation locale par gamertag normalise.
func h5Identity(gamertag string) canonical.PlayerIdentity {
	return canonical.PlayerIdentity{
		XUID:               "",
		Gamertag:           gamertag,
		GamertagNormalized: strings.ToLower(strings.TrimSpace(gamertag)),
		IsBot:              false,
	}
}

// iso8601DurationRe capture les composantes H/M/S d'une duree ISO8601 du type
// "PT5M41.7930011S" (heures/minutes/secondes ; jours rares en match h5, gérés).
var iso8601DurationRe = regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?$`)

// h5MaxDurationSeconds borne une durée de match plausible (24h). Une valeur au-delà
// est rejetée (donnée corrompue / overflow regex) → nil, pour ne pas polluer les
// stats ni produire un StartedAtUTC absurde via la soustraction fin−durée.
const h5MaxDurationSeconds = 24 * 3600

// parseISO8601DurationSeconds convertit "PT12M34.5S" en secondes (arrondi). Nil si
// vide, non-parsable, sans aucune composante ("PT"), ou hors borne plausible (le
// canonique distingue "indisponible" de "0").
func parseISO8601DurationSeconds(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	m := iso8601DurationRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	// Exiger au moins une composante (rejette "PT" / "P" qui matchent la regex
	// tout-optionnel et renverraient 0 au lieu de "indisponible").
	if m[1] == "" && m[2] == "" && m[3] == "" && m[4] == "" {
		return nil
	}
	var total float64
	if m[1] != "" { // jours
		if d, err := strconv.Atoi(m[1]); err == nil {
			total += float64(d) * 86400
		}
	}
	if m[2] != "" { // heures
		if h, err := strconv.Atoi(m[2]); err == nil {
			total += float64(h) * 3600
		}
	}
	if m[3] != "" { // minutes
		if mn, err := strconv.Atoi(m[3]); err == nil {
			total += float64(mn) * 60
		}
	}
	if m[4] != "" { // secondes (fractionnaires)
		if sec, err := strconv.ParseFloat(m[4], 64); err == nil {
			total += sec
		}
	}
	// Garde-fou overflow/corruption : hors [0, 24h] → indisponible.
	if total < 0 || total > h5MaxDurationSeconds {
		return nil
	}
	out := int(math.Round(total))
	return &out
}

// deriveOutcome derive l'issue d'un match pour le joueur a partir du Rang
// (1 = vainqueur). Choix data-grounded (sonde : JGtm Rank 1 = victoire) plutot
// que de deviner l'enum Result int.
//
//   - Jeu d'EQUIPE : on utilise le Rang D'EQUIPE (teamRank). En 4v4, Player.Rank
//     est un classement INDIVIDUEL au scoreboard (1er d'une equipe perdante = Rank 1)
//     -> l'utiliser donnerait un faux Win. Si le rang d'equipe est introuvable
//     (equipe absente de Teams), l'issue est INDETERMINEE -> OutcomeTie (degradation
//     documentee, rare).
//   - FFA (isTeamGame=false) : on utilise le rang INDIVIDUEL du joueur.
//
// Limitation Phase 1 (cf. review) : les vrais nuls ne sont pas distingues (pas de
// detection d'egalite de score). Les Ties derives des matchs sont donc systematiquement
// sous-comptes vs PlayerStats.Ties (issu du service record) — Phase 2.
func deriveOutcome(playerRank int, teamRank *int, isTeamGame bool) canonical.Outcome {
	if isTeamGame {
		if teamRank == nil {
			return canonical.OutcomeTie // indetermine : equipe absente de Teams
		}
		if *teamRank == 1 {
			return canonical.OutcomeWin
		}
		return canonical.OutcomeLoss
	}
	if playerRank == 1 {
		return canonical.OutcomeWin
	}
	return canonical.OutcomeLoss
}

// mapMatchSummaries projette une page d'historique h5 vers []MatchSummary, du
// point de vue du joueur requete (self = participant au gamertag donne).
//
// classifier (stratégie #1 set-membership, cf. package classification) résout le
// caractère classé/PvE depuis le HopperId. nil OU set vide → verdicts nil
// (INDETERMINE) : comportement conservateur byte-identique tant que la liste
// autoritative des HopperIds classés Halo 5 n'est pas publiée (handoff §4-5).
func mapMatchSummaries(resp *H5MatchesResponse, gamertag string, classifier classification.RankedClassifier) []canonical.MatchSummary {
	if resp == nil {
		return nil
	}
	out := make([]canonical.MatchSummary, 0, len(resp.Results))
	for i := range resp.Results {
		out = append(out, mapOneMatchSummary(&resp.Results[i], gamertag, classifier))
	}
	return out
}

func mapOneMatchSummary(r *H5MatchResult, gamertag string, classifier classification.RankedClassifier) canonical.MatchSummary {
	dur := parseISO8601DurationSeconds(r.MatchDuration)
	end := parseISODate(r.MatchCompletedDate.ISO8601Date)
	started := startedFromEnd(end, dur)

	self := selfPlayer(r, gamertag)
	// self introuvable (casse/renommage/pagination cote API gamertag-keyee) -> issue
	// INDETERMINEE (OutcomeTie par defaut, degradation documentee). Quand l'historique
	// sera cable (Phase 2), l'adapter loguera ce cas (les mappers restent purs ici).
	outcome := canonical.OutcomeTie
	if self != nil {
		outcome = deriveOutcome(self.Rank, teamRankFor(r, self.TeamId), r.IsTeamGame)
	}

	isRanked := classifyRanked(classifier, r.HopperId)
	isPvE := classifyPvE(classifier, r.HopperId)

	return canonical.MatchSummary{
		MatchID:         r.Id.MatchId,
		StartedAtUTC:    started,
		DurationSeconds: dur,
		MatchType:       h5MatchType(r.Id.GameMode, isRanked, isPvE),
		Playlist:        assetRef("playlist", r.HopperId),
		Map:             assetRef("map", r.MapId),
		GameVariant:     assetRef("game_variant", r.GameBaseVariantId),
		PairMode:        nil, // h5 n'a pas de pair_name (Phase 2)
		IsRanked:        isRanked,
		IsPvE:           isPvE,
		Outcome:         outcome,
		Teams:           mapTeams(r.Teams),
		T0Ms:            nil, // pas de countdown/real_start_time en h5
	}
}

// classifyRanked/classifyPvE nil-gardent l'interface classifier (nil = aucune
// stratégie câblée → verdict indéterminé). Gardent les mappers purs.
func classifyRanked(c classification.RankedClassifier, hopperID string) *bool {
	if c == nil {
		return nil
	}
	return c.IsRanked(hopperID)
}

func classifyPvE(c classification.RankedClassifier, hopperID string) *bool {
	if c == nil {
		return nil
	}
	return c.IsPvE(hopperID)
}

// selfPlayer retourne le participant correspondant au gamertag requete (compare
// insensible a la casse, h5 etant gamertag-keye), ou nil.
func selfPlayer(r *H5MatchResult, gamertag string) *H5MatchPlayer {
	norm := strings.ToLower(strings.TrimSpace(gamertag))
	for i := range r.Players {
		if strings.ToLower(strings.TrimSpace(r.Players[i].Player.Gamertag)) == norm {
			return &r.Players[i]
		}
	}
	return nil
}

// teamRankFor retourne le Rank de l'equipe d'id teamID, ou nil si absente.
func teamRankFor(r *H5MatchResult, teamID int) *int {
	for i := range r.Teams {
		if r.Teams[i].Id == teamID {
			rank := r.Teams[i].Rank
			return &rank
		}
	}
	return nil
}

func mapTeams(teams []H5Team) []canonical.TeamSnapshot {
	if len(teams) == 0 {
		return nil
	}
	out := make([]canonical.TeamSnapshot, 0, len(teams))
	for i := range teams {
		score := teams[i].Score
		out = append(out, canonical.TeamSnapshot{
			TeamID: teams[i].Id,
			Score:  &score,
			// MMR + ParticipantsXUIDs : indisponibles en h5 (xuid null).
		})
	}
	return out
}

// h5MatchType classe le mode h5. Priorité aux verdicts AUTORITATIFS du classifier
// (set-membership HopperId) quand ils existent :
//   - isPvE &true  -> firefight (Warzone FF) ;
//   - isRanked &true -> ranked ; isRanked &false -> social.
//
// Verdicts nil (pas de liste autoritative publiée) -> repli heuristique Phase 1 :
// GameMode 1 = Arena (PvP) -> social ; sinon unknown. Byte-identique au
// comportement avant classifier tant que les sets sont vides.
func h5MatchType(gameMode int, isRanked, isPvE *bool) canonical.MatchType {
	if isPvE != nil && *isPvE {
		return canonical.MatchTypeFirefight
	}
	if isRanked != nil {
		if *isRanked {
			return canonical.MatchTypeRanked
		}
		return canonical.MatchTypeSocial
	}
	switch gameMode {
	case 1:
		return canonical.MatchTypeSocial
	default:
		return canonical.MatchTypeUnknownMT
	}
}

func assetRef(kind, id string) *canonical.AssetReference {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	return &canonical.AssetReference{Kind: kind, ID: id}
}

// h5DateLayouts : la sonde montre toujours un suffixe 'Z' (RFC3339), mais par
// cohérence défensive avec le reste du codebase Halo (qui tolère plusieurs
// layouts), on accepte aussi le naïf sans offset plutôt que de renvoyer nil.
var h5DateLayouts = []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}

func parseISODate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range h5DateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			t = t.UTC()
			return &t
		}
	}
	return nil
}

// startedFromEnd derive l'heure de debut = fin − duree (h5 ne donne que la fin).
// Fallback : fin brute si la duree est inconnue ; zero si la fin est inconnue.
func startedFromEnd(end *time.Time, durSeconds *int) time.Time {
	if end == nil {
		return time.Time{}
	}
	if durSeconds == nil {
		return *end
	}
	return end.Add(-time.Duration(*durSeconds) * time.Second)
}
