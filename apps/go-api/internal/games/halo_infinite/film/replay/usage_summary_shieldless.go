package replay

import (
	"cmp"
	"slices"
)

// usage_summary_shieldless.go — LES FRAGS SANS BOUCLIER, jointure PURE du document de rejeu et du
// fil des morts du match (citation « Presque, ça ne compte pas » : tuer un Spartan bouclier à 0 %
// et survivre).
//
// # L'ÉTAT DU BOUCLIER À L'INSTANT DU FRAG
//
// Le film ne réplique le bouclier (`Point.Sh`) que lorsqu'il CHANGE : la dernière mesure de la vie
// du tueur avant le frag est l'état à l'instant. Une vie sans aucune mesure avant le frag n'a
// jamais vu son bouclier bouger depuis l'apparition : il est plein, le frag n'est pas sans
// bouclier. Une mesure à 0 plus ancienne que [ShieldlessFreshnessMS] n'est pas retenue : la
// recharge ne se lit pas de façon fiable, un zéro ancien a pu être effacé sans réplication.
//
// # SURVIVRE
//
// Le tueur ne meurt pas dans les [ShieldlessSurvivalMS] qui suivent le frag (fil des morts, même
// match). Un échange de frags simultanés ne compte donc pas.
//
// # CE QUE L'APPELANT FOURNIT
//
// Les frags sont lus en base (`match_kill_events_latest`) par l'appelant : ce paquet ne lit ni
// film ni base. Les instants sont sur l'horloge « début du film » (`killsource.Kill.TimeMS`),
// recalés ici sur l'origine du document — la même conversion que la jointure des épisodes
// d'équipement ([frameOfFilmMS]).

const (
	// ShieldlessFreshnessMS : âge maximal de la mesure de bouclier à 0 retenue pour un frag.
	ShieldlessFreshnessMS = 5000
	// ShieldlessSurvivalMS : le tueur doit rester en vie pendant cette fenêtre après le frag.
	ShieldlessSurvivalMS = 1000
)

// ShieldlessKillRef est un frag du fil des morts : tueur et victime en xuid décimal (vide = bot ou
// identité non résolue), instant sur l'horloge du film.
type ShieldlessKillRef struct {
	Killer string
	Victim string
	TimeMS int
}

// ShieldlessKillsInput porte les frags d'un match ET si la lecture a eu lieu. Read faux = non mesuré
// (aucune ligne du fil des morts pour ce match, ou lecture impossible) : les lignes gardent un
// compte ABSENT, jamais un zéro.
type ShieldlessKillsInput struct {
	Read  bool
	Kills []ShieldlessKillRef
}

// AttachShieldlessKills pose `ShieldlessKills` sur les lignes du résumé : un compte (zéro compris)
// pour chaque ligne quand la mesure est possible, et une ligne neuve pour un tueur qui n'en avait
// pas. Sans lecture des frags ou sans origine de document, rien n'est posé.
func AttachShieldlessKills(s *UsageSummary, doc *ReplayDocument, in ShieldlessKillsInput) {
	if s == nil || doc == nil || !in.Read || doc.OriginMs == nil || doc.FrameIntervalMS <= 0 {
		return
	}
	counts := countShieldlessKills(doc, in.Kills)
	seen := make(map[string]bool, len(s.Players))
	for i := range s.Players {
		p := &s.Players[i]
		n := counts[p.XUID]
		p.ShieldlessKills = &n
		seen[p.XUID] = true
	}
	added := false
	for xuid, n := range counts {
		if !seen[xuid] && n > 0 {
			v := n
			s.Players = append(s.Players, UsagePlayerSummary{XUID: xuid, ShieldlessKills: &v})
			added = true
		}
	}
	if added {
		slices.SortFunc(s.Players, func(a, b UsagePlayerSummary) int { return cmp.Compare(a.XUID, b.XUID) })
	}
}

// countShieldlessKills compte, par tueur, les frags bouclier à 0 suivis de sa survie.
func countShieldlessKills(doc *ReplayDocument, kills []ShieldlessKillRef) map[string]int {
	lives := make(map[string][]*Track)
	for i := range doc.Tracks {
		if tr := &doc.Tracks[i]; tr.XUID != "" {
			lives[tr.XUID] = append(lives[tr.XUID], tr)
		}
	}
	deaths := make(map[string][]int)
	for _, k := range kills {
		if k.Victim != "" {
			deaths[k.Victim] = append(deaths[k.Victim], k.TimeMS)
		}
	}
	origin, step := *doc.OriginMs, doc.FrameIntervalMS
	out := make(map[string]int)
	for _, k := range kills {
		if k.Killer == "" || k.Killer == k.Victim {
			continue
		}
		frame := frameOfFilmMS(k.TimeMS, origin, step)
		if !shieldZeroAt(lifeAt(lives[k.Killer], frame), frame, step) {
			continue
		}
		if diesWithin(deaths[k.Killer], k.TimeMS, ShieldlessSurvivalMS) {
			continue
		}
		out[k.Killer]++
	}
	return out
}

// lifeAt rend la vie qui couvre `frame`, nil sinon.
func lifeAt(lives []*Track, frame int) *Track {
	for _, tr := range lives {
		if frame >= trackStartFrame(tr) && frame <= trackEndFrame(tr) {
			return tr
		}
	}
	return nil
}

// shieldZeroAt dit si la dernière mesure de bouclier de la vie, au plus tard à `frame`, vaut 0 et
// date de moins de [ShieldlessFreshnessMS].
func shieldZeroAt(tr *Track, frame, step int) bool {
	if tr == nil {
		return false
	}
	var last *Point
	for j := range tr.Points {
		p := &tr.Points[j]
		if p.T > frame {
			break
		}
		if p.Sh != nil {
			last = p
		}
	}
	return last != nil && *last.Sh == 0 && (frame-last.T)*step < ShieldlessFreshnessMS
}

func diesWithin(deathsMS []int, fromMS, windowMS int) bool {
	for _, ms := range deathsMS {
		if ms >= fromMS && ms <= fromMS+windowMS {
			return true
		}
	}
	return false
}

func trackStartFrame(tr *Track) int {
	if tr.StartFrame > 0 || len(tr.Points) == 0 {
		return tr.StartFrame
	}
	return tr.Points[0].T
}

func trackEndFrame(tr *Track) int {
	if tr.EndFrame > 0 || len(tr.Points) == 0 {
		return tr.EndFrame
	}
	return tr.Points[len(tr.Points)-1].T
}
