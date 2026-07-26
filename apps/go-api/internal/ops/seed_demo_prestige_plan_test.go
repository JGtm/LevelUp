package ops

import (
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/records"
)

// testDemoCorpus : agrégats représentatifs d'un corpus démo (≈ 60 matchs).
func testDemoCorpus() (demoCorpusStats, []demoMatchPP) {
	start := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	st := demoCorpusStats{
		Matches: 60, Kills: 780, Deaths: 640, Assists: 210,
		HeadshotKills: 240, Wins: 33,
		First: start, Last: start.Add(30 * 24 * time.Hour),
	}
	matches := make([]demoMatchPP, 0, st.Matches)
	for i := 0; i < st.Matches; i++ {
		matches = append(matches, demoMatchPP{
			MatchID: fmt.Sprintf("demo-match-%03d", i),
			Won:     i < st.Wins,
			At:      start.Add(time.Duration(i) * 12 * time.Hour),
		})
	}
	return st, matches
}

func testDemoPlan(t *testing.T) demoPrestigePlan {
	t.Helper()
	st, matches := testDemoCorpus()
	return buildDemoPrestigePlan("halo_infinite", st, matches)
}

// TestDemoPrestigePlan_NoRealIdentity : invariant anti-fuite — le plan ne porte
// QUE des identités démo (slug + xuid réservés), jamais un gamertag source.
func TestDemoPrestigePlan_NoRealIdentity(t *testing.T) {
	plan := testDemoPlan(t)
	if plan.UserID != DefaultDemoMainSlug {
		t.Errorf("UserID = %q, want %q (slug démo)", plan.UserID, DefaultDemoMainSlug)
	}
	if plan.XUID != DefaultDemoXUID {
		t.Errorf("XUID = %q, want %q (xuid démo)", plan.XUID, DefaultDemoXUID)
	}
	if plan.Gamertag != DefaultDemoMainGamertag {
		t.Errorf("Gamertag = %q, want %q", plan.Gamertag, DefaultDemoMainGamertag)
	}
}

// TestDemoPrestigePlan_CampaignStatusesInEnum — RÉGRESSION 2026-07-26. La
// campagne close était seedée « closed », hors de l'énum
// active|paused|completed|abandoned. La colonne étant un VARCHAR libre, l'INSERT
// passait ; c'est ListEnded (`status IN ('completed','abandoned')`) qui ne la
// voyait jamais. Le test verrouille le couple attendu : 1 active + 1 terminale.
func TestDemoPrestigePlan_CampaignStatusesInEnum(t *testing.T) {
	plan := testDemoPlan(t)
	if len(plan.Campaigns) != 2 {
		t.Fatalf("campagnes démo = %d, want 2 (1 en cours + 1 close)", len(plan.Campaigns))
	}
	if err := validateDemoCampaigns(plan.Campaigns); err != nil {
		t.Errorf("plan démo invalide: %v", err)
	}
	var active, ended int
	for _, c := range plan.Campaigns {
		switch c.Status {
		case campaign.StatusActive:
			active++
			if c.EndedAt != nil {
				t.Errorf("campagne %s active avec un ended_at", c.ID)
			}
		case campaign.StatusCompleted, campaign.StatusAbandoned:
			ended++
			if c.EndedAt == nil {
				t.Errorf("campagne %s terminale sans ended_at — ListEnded la trierait NULLS LAST", c.ID)
			}
		default:
			t.Errorf("campagne %s : statut %q ni actif ni terminal", c.ID, c.Status)
		}
	}
	if active != 1 || ended != 1 {
		t.Errorf("répartition = %d active / %d close, want 1 / 1", active, ended)
	}
}

// TestValidateDemoCampaigns_RejectsUnknownStatus — preuve de morsure : la garde
// doit refuser le statut EXACT qui a causé l'incident, sans quoi elle ne protège
// rien.
func TestValidateDemoCampaigns_RejectsUnknownStatus(t *testing.T) {
	err := validateDemoCampaigns([]demoPrestigeCampaign{
		{ID: "demo-campaign-kda", Status: campaign.CampaignStatus("closed")},
	})
	if err == nil {
		t.Fatal("statut « closed » accepté — la garde ne mord pas")
	}
}

// TestDemoPrestigePlan_ArcsAndObjectivesConsistent : chaque objectif référence un
// arc DÉCLARÉ (ou aucun) ; un arc complété n'a que des étapes terminales ; l'arc
// en cours a au moins une étape acquise ET une étape en cours (progression visible).
func TestDemoPrestigePlan_ArcsAndObjectivesConsistent(t *testing.T) {
	plan := testDemoPlan(t)
	known := map[string]*demoPrestigeArc{}
	for i := range plan.Arcs {
		known[plan.Arcs[i].ID] = &plan.Arcs[i]
	}
	perArc := map[string]map[prestige.ChallengeStatus]int{}
	for _, c := range plan.Challenges {
		if c.ArcID == "" {
			continue
		}
		if _, ok := known[c.ArcID]; !ok {
			t.Fatalf("objectif %s référence un arc inconnu %q", c.ID, c.ArcID)
		}
		if perArc[c.ArcID] == nil {
			perArc[c.ArcID] = map[prestige.ChallengeStatus]int{}
		}
		perArc[c.ArcID][c.Status]++
	}
	for id, a := range known {
		counts := perArc[id]
		if len(counts) == 0 {
			t.Errorf("arc %s sans aucun objectif", id)
			continue
		}
		if a.CompletedAt != nil {
			if n := counts[prestige.StatusActive]; n != 0 {
				t.Errorf("arc %s marqué complété mais %d objectif(s) encore actif(s)", id, n)
			}
			continue
		}
		if counts[prestige.StatusCompleted] == 0 || counts[prestige.StatusActive] == 0 {
			t.Errorf("arc %s en cours : %d acquis / %d actifs — la progression ne serait pas visible",
				id, counts[prestige.StatusCompleted], counts[prestige.StatusActive])
		}
	}
}

// TestDemoPrestigePlan_LifecycleCoverage : le jeu d'objectifs couvre TOUS les
// stades demandés (actif, complété, expiré, abandonné) et chaque statut terminal
// porte bien une date.
func TestDemoPrestigePlan_LifecycleCoverage(t *testing.T) {
	plan := testDemoPlan(t)
	seen := map[prestige.ChallengeStatus]int{}
	for _, c := range plan.Challenges {
		seen[c.Status]++
		if c.Status.IsTerminal() && c.TerminalAt == nil {
			t.Errorf("objectif %s en statut terminal %s sans date", c.ID, c.Status)
		}
		if !c.Status.IsTerminal() && c.TerminalAt != nil {
			t.Errorf("objectif %s actif mais porte une date terminale", c.ID)
		}
		if c.Target <= 0 {
			t.Errorf("objectif %s : cible %v (doit être strictement positive)", c.ID, c.Target)
		}
		if c.PP <= 0 {
			t.Errorf("objectif %s : récompense %d PP (doit être positive)", c.ID, c.PP)
		}
	}
	for _, st := range []prestige.ChallengeStatus{
		prestige.StatusActive, prestige.StatusCompleted,
		prestige.StatusExpired, prestige.StatusAbandoned,
	} {
		if seen[st] == 0 {
			t.Errorf("aucun objectif au statut %s — la démo ne montrerait pas ce stade", st)
		}
	}
}

// TestDemoPrestigePlan_TargetsDerivedFromCorpus : une cible d'objectif COMPLÉTÉ
// reste sous le réalisé du corpus, une cible d'objectif ACTIF passe au-dessus.
// Sans ça, la démo afficherait « objectif atteint » sur un compteur inférieur au
// palier, ou « en cours » sur un palier déjà dépassé.
func TestDemoPrestigePlan_TargetsDerivedFromCorpus(t *testing.T) {
	st, matches := testDemoCorpus()
	plan := buildDemoPrestigePlan("halo_infinite", st, matches)
	realized := map[string]float64{
		"matches_played": float64(st.Matches),
		"kills":          float64(st.Kills),
		"assists":        float64(st.Assists),
		"headshot_kills": float64(st.HeadshotKills),
		"wins":           float64(st.Wins),
	}
	for _, c := range plan.Challenges {
		r, ok := realized[c.Metric]
		if !ok {
			continue // métrique de moyenne (accuracy, kda) : pas de cumul comparable
		}
		switch c.Status {
		case prestige.StatusCompleted:
			if c.Target > r {
				t.Errorf("objectif complété %s : cible %v > réalisé %v", c.ID, c.Target, r)
			}
		case prestige.StatusActive:
			if c.Target <= r {
				t.Errorf("objectif actif %s : cible %v <= réalisé %v (il serait déjà atteint)",
					c.ID, c.Target, r)
			}
		}
	}
}

// TestDemoPrestigePlan_TotalPPMatchesEvents : le total de points de progression
// est la SOMME des événements qui l'ont produit, et le niveau en découle par le
// calcul canonique. C'est l'invariant que la démo doit rendre vérifiable à l'œil.
func TestDemoPrestigePlan_TotalPPMatchesEvents(t *testing.T) {
	plan := testDemoPlan(t)
	sum := 0
	for _, ev := range plan.Events {
		if ev.PP < 0 {
			t.Errorf("événement %s : PP négatif (%d)", ev.ID, ev.PP)
		}
		sum += ev.PP
	}
	if sum != plan.TotalPP {
		t.Errorf("TotalPP = %d, somme des événements = %d", plan.TotalPP, sum)
	}
	want := prestige.LevelFromPP(prestige.DefaultTuning(), plan.TotalPP).Index
	if plan.Level != want {
		t.Errorf("Level = %d, LevelFromPP(%d) = %d", plan.Level, plan.TotalPP, want)
	}
}

// TestDemoPrestigePlan_EventsCoverCorpusAndCompletions : un événement « match »
// par match du corpus (le total ne peut donc pas contredire l'historique affiché),
// un événement « objectif » par objectif complété, un bonus par arc complété.
func TestDemoPrestigePlan_EventsCoverCorpusAndCompletions(t *testing.T) {
	st, matches := testDemoCorpus()
	plan := buildDemoPrestigePlan("halo_infinite", st, matches)
	bySource := map[string]int{}
	ids := map[string]bool{}
	for _, ev := range plan.Events {
		if ids[ev.ID] {
			t.Errorf("identifiant d'événement dupliqué : %s", ev.ID)
		}
		ids[ev.ID] = true
		bySource[ev.SourceType]++
	}
	if bySource[prestige.SourceMatch] != len(matches) {
		t.Errorf("événements match = %d, want %d (1 par match du corpus)",
			bySource[prestige.SourceMatch], len(matches))
	}
	completed := 0
	for _, c := range plan.Challenges {
		if c.Status == prestige.StatusCompleted {
			completed++
		}
	}
	if bySource[prestige.SourceChallenge] != completed {
		t.Errorf("événements objectif = %d, want %d", bySource[prestige.SourceChallenge], completed)
	}
	completedArcs := 0
	for _, a := range plan.Arcs {
		if a.CompletedAt != nil {
			completedArcs++
		}
	}
	if bySource[prestige.SourceArc] != completedArcs {
		t.Errorf("bonus d'arc = %d, want %d", bySource[prestige.SourceArc], completedArcs)
	}
}

// TestDemoPrestigePlan_RecordsPlausibleAndProgressing : chaque PB est dans les
// bornes de vraisemblance du catalogue records ET domine la valeur précédente
// (une timeline qui régresse serait un non-sens à l'écran).
func TestDemoPrestigePlan_RecordsPlausibleAndProgressing(t *testing.T) {
	plan := testDemoPlan(t)
	if len(plan.Records) == 0 {
		t.Fatal("aucun record généré")
	}
	byMetric := map[records.TrackedMetric]map[records.RecordPeriod]float64{}
	for _, r := range plan.Records {
		if !records.IsPlausibleValue(r.Metric, r.Value) {
			t.Errorf("PB %s/%s : valeur %v hors bornes", r.Metric, r.Period, r.Value)
		}
		if r.Value <= r.PrevValue {
			t.Errorf("PB %s/%s : valeur %v <= précédente %v", r.Metric, r.Period, r.Value, r.PrevValue)
		}
		if !r.PrevAchievedAt.Before(r.AchievedAt) {
			t.Errorf("PB %s/%s : date précédente %v non antérieure à %v",
				r.Metric, r.Period, r.PrevAchievedAt, r.AchievedAt)
		}
		if byMetric[r.Metric] == nil {
			byMetric[r.Metric] = map[records.RecordPeriod]float64{}
		}
		byMetric[r.Metric][r.Period] = r.Value
	}
	for m, periods := range byMetric {
		all, hasAll := periods[records.RecordPeriodAllTime]
		recent, hasRecent := periods[records.RecordPeriod30d]
		if hasAll && hasRecent && all < recent {
			t.Errorf("métrique %s : record carrière %v < record 30 jours %v", m, all, recent)
		}
	}
}

// TestDemoPrestigePlan_DatesWithinCorpusWindow : aucune ligne du plan n'est datée
// avant le premier match du corpus démo.
func TestDemoPrestigePlan_DatesWithinCorpusWindow(t *testing.T) {
	st, matches := testDemoCorpus()
	plan := buildDemoPrestigePlan("halo_infinite", st, matches)
	check := func(label string, at time.Time) {
		if at.Before(st.First) {
			t.Errorf("%s daté %v, avant le début du corpus %v", label, at, st.First)
		}
	}
	for _, a := range plan.Arcs {
		check("arc "+a.ID, a.CreatedAt)
		if a.CompletedAt != nil {
			check("complétion arc "+a.ID, *a.CompletedAt)
		}
	}
	for _, c := range plan.Challenges {
		check("objectif "+c.ID, c.CreatedAt)
		if c.TerminalAt != nil {
			check("fin objectif "+c.ID, *c.TerminalAt)
		}
	}
	for _, s := range plan.Streaks {
		check("série "+s.ID, s.LastAt)
	}
	for _, ev := range plan.Events {
		check("événement "+ev.ID, ev.CreatedAt)
	}
}
