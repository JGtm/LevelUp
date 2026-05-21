// Package analysis — algorithme de découpage en sessions.
//
// Port Go de src/analysis/sessions.py (LevelUp-no-streamlit).
package analysis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// DefaultSessionGapMinutes est le delai par defaut entre deux sessions (2 heures).
const DefaultSessionGapMinutes = 120

const defaultCutoffHour = 8

// DefaultSessionOptions retourne les options par defaut pour le calcul des sessions.
func DefaultSessionOptions() domain.SessionComputeOptions {
	return domain.SessionComputeOptions{
		GapMinutes:          DefaultSessionGapMinutes,
		CutoffHour:          defaultCutoffHour,
		SplitOnRankedChange: false,
		TeamChangeMode:      domain.TeamChangeModeFriends,
		Mode:                domain.SessionModeContext,
	}
}

// ComputeSessions decoupe les matchs en sessions (mode gap-based simple).
func ComputeSessions(rows []domain.SessionMatchRow, gapMinutes int) []domain.SessionAssignment {
	if len(rows) == 0 {
		return nil
	}
	if gapMinutes <= 0 {
		gapMinutes = DefaultSessionGapMinutes
	}
	sorted := sessionSortedRows(rows)
	threshold := time.Duration(gapMinutes) * time.Minute
	out := make([]domain.SessionAssignment, len(sorted))
	sessionID := 0
	for i, r := range sorted {
		if i > 0 && sorted[i].StartTime.Sub(sorted[i-1].StartTime) > threshold {
			sessionID++
		}
		out[i] = domain.SessionAssignment{MatchID: r.MatchID, SessionID: sessionID}
	}
	return out
}

// ComputeSessionsWithContext decoupe les matchs en sessions (mode context).
func ComputeSessionsWithContext(rows []domain.SessionMatchRow, opts domain.SessionComputeOptions) []domain.SessionAssignment {
	if len(rows) == 0 {
		return nil
	}
	if opts.GapMinutes <= 0 {
		opts.GapMinutes = DefaultSessionGapMinutes
	}
	sorted := sessionSortedRows(rows)
	threshold := time.Duration(opts.GapMinutes) * time.Minute
	friendSet := buildFriendSet(opts.FriendsXUIDs)
	out := make([]domain.SessionAssignment, len(sorted))
	sessionID := 0
	for i, r := range sorted {
		if i > 0 {
			prev := sorted[i-1]
			gapBreak := sorted[i].StartTime.Sub(prev.StartTime) > threshold
			rankedBreak := opts.SplitOnRankedChange && (prev.IsRanked != r.IsRanked)
			var teammatesBreak bool
			switch opts.TeamChangeMode {
			case domain.TeamChangeModeIgnore:
				// ignorer la composition — pas de rupture sur coéquipiers
				teammatesBreak = false
			case domain.TeamChangeModeFriends:
				// rupture uniquement si un ami (FriendsXUIDs) change
				teammatesBreak = isTeammatesBreak(prev.TeammatesSig, r.TeammatesSig, friendSet)
			case domain.TeamChangeModeGroup:
				// rupture dès qu'un coéquipier quelconque change (ignorer
				// FriendsXUIDs). Comparaison directe — `isTeammatesBreak(nil)`
				// signifie maintenant "pas d'amis trackés → pas de break"
				// (sémantique inverse, cf. fix 2026-05-08).
				teammatesBreak = derefString(prev.TeammatesSig) != derefString(r.TeammatesSig)
			default:
				// rétrocompatibilité : comportement historique (group si pas d'amis, friends sinon)
				teammatesBreak = isTeammatesBreak(prev.TeammatesSig, r.TeammatesSig, friendSet)
			}
			if gapBreak || teammatesBreak || rankedBreak {
				sessionID++
			}
		}
		out[i] = domain.SessionAssignment{MatchID: r.MatchID, SessionID: sessionID}
	}
	return out
}

// BuildSessionGroups transforme des assignments en groupes avec labels.
func BuildSessionGroups(rows []domain.SessionMatchRow, assignments []domain.SessionAssignment) []domain.SessionGroup {
	if len(assignments) == 0 {
		return nil
	}
	timeByID := make(map[string]time.Time, len(rows))
	playedByID := make(map[string]int, len(rows))
	for _, r := range rows {
		timeByID[r.MatchID] = r.StartTime
		if r.TimePlayedSecs != nil {
			playedByID[r.MatchID] = *r.TimePlayedSecs
		}
	}
	type bucket struct {
		matchIDs    []string
		startTime   time.Time
		endTime     time.Time
		totalPlayed int
	}
	groupMap := make(map[int]*bucket)
	maxID := 0
	for _, a := range assignments {
		t := timeByID[a.MatchID]
		played := playedByID[a.MatchID]
		if b, ok := groupMap[a.SessionID]; !ok {
			groupMap[a.SessionID] = &bucket{matchIDs: []string{a.MatchID}, startTime: t, endTime: t, totalPlayed: played}
		} else {
			b.matchIDs = append(b.matchIDs, a.MatchID)
			b.totalPlayed += played
			if t.Before(b.startTime) {
				b.startTime = t
			}
			if t.After(b.endTime) {
				b.endTime = t
			}
		}
		if a.SessionID > maxID {
			maxID = a.SessionID
		}
	}
	groups := make([]domain.SessionGroup, 0, maxID+1)
	for sid := 0; sid <= maxID; sid++ {
		b, ok := groupMap[sid]
		if !ok {
			continue
		}
		span := int(b.endTime.Sub(b.startTime).Seconds())
		groups = append(groups, domain.SessionGroup{
			SessionID:          sid,
			SessionLabel:       buildSessionLabel(b.startTime, b.endTime, len(b.matchIDs)),
			MatchIDs:           b.matchIDs,
			DurationSeconds:    span,
			TotalPlayedSeconds: b.totalPlayed,
		})
	}
	return groups
}

// MergeSessionLabels injecte les labels dans les assignments (retourne une copie).
func MergeSessionLabels(assignments []domain.SessionAssignment, groups []domain.SessionGroup) []domain.SessionAssignment {
	labelByID := make(map[int]string, len(groups))
	for _, g := range groups {
		labelByID[g.SessionID] = g.SessionLabel
	}
	out := make([]domain.SessionAssignment, len(assignments))
	for i, a := range assignments {
		out[i] = domain.SessionAssignment{
			MatchID:      a.MatchID,
			SessionID:    a.SessionID,
			SessionLabel: labelByID[a.SessionID],
		}
	}
	return out
}

// IsSessionPotentiallyActive indique si la session est peut-etre encore active.
//
// Wrapper compat — utilise time.Now() en interne. Pour les tests deterministes,
// utiliser IsSessionPotentiallyActiveAt avec un clock.Fake (revue 2026-04-29
// P3.7 — gap test manquant IV).
func IsSessionPotentiallyActive(lastMatchTime time.Time, cutoffHour int) bool {
	return IsSessionPotentiallyActiveAt(time.Now(), lastMatchTime, cutoffHour)
}

// IsSessionPotentiallyActiveAt est la version testable avec horloge injectee.
// Permet de tester deterministe le cas "lastMatch hier avant cutoffHour" qui
// dependait auparavant de l'heure courante reelle (flaky).
func IsSessionPotentiallyActiveAt(now, lastMatchTime time.Time, cutoffHour int) bool {
	now = now.In(lastMatchTime.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	lmtDay := time.Date(lastMatchTime.Year(), lastMatchTime.Month(), lastMatchTime.Day(), 0, 0, 0, 0, lastMatchTime.Location())
	if lmtDay.Equal(today) {
		return true
	}
	if lmtDay.Equal(yesterday) {
		return now.Hour() < cutoffHour
	}
	return false
}

// GetBucketInfo retourne le type et le label d un bucket selon la plage de jours.
func GetBucketInfo(days float64) domain.BucketInfo {
	switch {
	case days < 1:
		return domain.BucketInfo{Type: domain.BucketMatch, Label: "partie"}
	case days <= 3:
		return domain.BucketInfo{Type: domain.BucketHour, Label: "heure"}
	case days <= 14:
		return domain.BucketInfo{Type: domain.BucketDay, Label: "jour"}
	case days <= 60:
		return domain.BucketInfo{Type: domain.BucketWeek, Label: "semaine"}
	default:
		return domain.BucketInfo{Type: domain.BucketMonth, Label: "mois"}
	}
}

func sessionSortedRows(rows []domain.SessionMatchRow) []domain.SessionMatchRow {
	sorted := make([]domain.SessionMatchRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})
	return sorted
}

func buildFriendSet(xuids []string) map[string]struct{} {
	if len(xuids) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(xuids))
	for _, x := range xuids {
		set[x] = struct{}{}
	}
	return set
}

func isTeammatesBreak(prevSig, currSig *string, friendSet map[string]struct{}) bool {
	prevStr := derefString(prevSig)
	currStr := derefString(currSig)
	if prevStr == currStr {
		return false
	}
	if friendSet == nil {
		// Bug fix 2026-05-08 : avant, friendSet=nil → "tout changement de
		// composition = rupture". Conséquence : un user sans amis configurés
		// avait `len(FriendsXUIDs)=0` → buildFriendSet retournait nil → mode
		// Friends dégradait silencieusement en mode Group → en matchmaking
		// solo, chaque match = nouvelle session (98% des sessions à 1 match
		// dans l'audit DB du 2026-05-08).
		//
		// Sémantique correcte : mode Friends sans amis trackés = aucun
		// changement de coéquipier ne doit casser la session (le critère gap
		// horaire reste seul actif). Pour l'ancien comportement "break sur
		// tout changement", le caller doit explicitement utiliser
		// TeamChangeModeGroup (cf. sessions.go:78-80, branche dédiée qui
		// passe friendSet=nil intentionnellement).
		return false
	}
	// mode "friends" : rupture uniquement si le sous-ensemble d'amis change
	prevFriends := filterFriends(parseXUIDs(prevStr), friendSet)
	currFriends := filterFriends(parseXUIDs(currStr), friendSet)
	if len(prevFriends) != len(currFriends) {
		return true
	}
	for _, xuid := range prevFriends {
		if !sliceContains(currFriends, xuid) {
			return true
		}
	}
	return false
}

func parseXUIDs(sig string) []string {
	if sig == "" {
		return nil
	}
	parts := strings.Split(sig, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func filterFriends(xuids []string, friendSet map[string]struct{}) []string {
	var out []string
	for _, x := range xuids {
		if _, ok := friendSet[x]; ok {
			out = append(out, x)
		}
	}
	return out
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func buildSessionLabel(start, end time.Time, count int) string {
	startStr := start.Format("15:04")
	endStr := end.Format("15:04")
	if startStr == endStr {
		return fmt.Sprintf("%s %s (%d)", start.Format("02/01/2006"), startStr, count)
	}
	return fmt.Sprintf("%s %s–%s (%d)", start.Format("02/01/2006"), startStr, endStr, count)
}
