package teammates

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ─── DECOR ─────────────────────────────────────────────────────────────────────────────

func isolementRadar() map[string]int { return map[string]int{"Arena": 18, "BTB": 24} }

func isolementMatches(variant string, ids ...string) []domain.TacticalMatch {
	out := make([]domain.TacticalMatch, 0, len(ids))
	for _, id := range ids {
		out = append(out, domain.TacticalMatch{
			MatchID: id, Outcome: domain.OutcomeWin, Mesure: true, GameVariantName: variant,
		})
	}
	return out
}

func isolementRows(ids ...string) []domain.SquadMatchRow {
	start := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	out := make([]domain.SquadMatchRow, 0, len(ids))
	for i, id := range ids {
		out = append(out, domain.SquadMatchRow{
			MatchID: id, StartTime: start.Add(time.Duration(i) * time.Minute),
		})
	}
	return out
}

// mortContexte pose UNE mort localisee : `proche` nil = aucun coequipier visible (bande
// « hors de vue »).
func mortContexte(matchID, xuid string, timeMs int64, proche *float64, visibles, horsDeVue int) domain.MortContexte {
	return domain.MortContexte{
		MatchID: matchID, VictimXUID: xuid, TimeMs: timeMs,
		PlusProcheM: proche, Visibles: visibles, HorsDeVue: horsDeVue,
	}
}

func metres(v float64) *float64 { return &v }

// ─── LE NUAGE ──────────────────────────────────────────────────────────────────────────

// TestBuildSquadIsolementNuage_UnPointParMort couvre la jointure exacte
// (match_id, victim_xuid, time_ms) entre le contexte de mort et la riposte, et les trois
// cas limites du contrat :
//
//	m1 @ 10 000 ms   coequipier a 9 m (rayon 18) -> ratio 0,5 ; vengee a 3 000 ms.
//	m1 @ 40 000 ms   aucun coequipier visible    -> hors de vue, ratio absent ; jamais vengee.
//	m2 @ 20 000 ms   coequipier a 27 m (rayon 18)-> ratio 1,5 ; mort ABSENTE du journal des
//	                                                kills -> ni vengee ni delai.
func TestBuildSquadIsolementNuage_UnPointParMort(t *testing.T) {
	ids := []string{"m1", "m2"}
	univers := domain.TacticalUnivers{
		Matchs:  isolementMatches("Arena", ids...),
		Equipes: equipesDeuxContreDeux(ids...),
	}
	// Journal des kills : m1 @ 10 000 ms le joueur tombe sous adv1, puis Ami venge a
	// 13 000 ms (delai 3 000). m1 @ 40 000 ms : tombe sous adv2, jamais venge.
	events := []domain.KillEvent{
		{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 10_000},
		{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 13_000},
		{MatchID: "m1", KillerXUID: "x_adv2", VictimXUID: "x_main", TimeMs: 40_000},
	}
	morts := []domain.MortContexte{
		mortContexte("m1", "x_main", 10_000, metres(9), 1, 0),
		mortContexte("m1", "x_main", 40_000, nil, 0, 1),
		mortContexte("m2", "x_main", 20_000, metres(27), 1, 0),
	}

	repo := &mockTacticalRepo{
		lecture: domain.TacticalKillEvents{Univers: univers, Events: events},
		morts:   domain.TacticalMortsContexte{Morts: morts},
	}
	svc := &TeammatesService{
		titleSlug: "halo_infinite", gamertag: "main",
		tacticalRepo: repo, caps: capsFiables(), radarRange: isolementRadar(),
	}

	rows := isolementRows(ids...)
	got := svc.buildSquadEchange(context.Background(), rows, rows, "main", "x_main", nil)
	if got == nil || got.NuageIsolement == nil {
		t.Fatal("nuage isolement attendu, obtenu nil")
	}
	nuage := got.NuageIsolement
	if len(nuage.Morts) != 3 {
		t.Fatalf("morts publiees = %d, attendues 3 : %+v", len(nuage.Morts), nuage.Morts)
	}

	parInstant := map[int64]domain.SquadIsolementMort{}
	for _, m := range nuage.Morts {
		parInstant[m.TimeMs] = m
	}

	// Mort vengee, coequipier visible a 9 m sur un rayon de 18 -> ratio 0,5.
	a := parInstant[10_000]
	if a.DistanceRatio == nil || *a.DistanceRatio != 0.5 {
		t.Errorf("mort @10 000 : ratio = %v, attendu 0,5", a.DistanceRatio)
	}
	if a.HorsDeVue {
		t.Error("mort @10 000 : hors_de_vue attendu faux (coequipier visible)")
	}
	if !a.Vengee || a.DelaiMs == nil || *a.DelaiMs != 3_000 {
		t.Errorf("mort @10 000 : vengee=%v delai=%v, attendu vengee a 3 000 ms", a.Vengee, a.DelaiMs)
	}

	// Aucun coequipier visible : bande « hors de vue », aucune abscisse inventee.
	b := parInstant[40_000]
	if b.DistanceRatio != nil {
		t.Errorf("mort @40 000 : ratio = %v, attendu absent", *b.DistanceRatio)
	}
	if !b.HorsDeVue {
		t.Error("mort @40 000 : hors_de_vue attendu vrai")
	}
	if b.Vengee || b.DelaiMs != nil {
		t.Errorf("mort @40 000 : attendue jamais vengee, obtenu vengee=%v delai=%v", b.Vengee, b.DelaiMs)
	}

	// Mort ABSENTE du journal des kills : la jointure ne trouve rien -> non vengee.
	c := parInstant[20_000]
	if c.DistanceRatio == nil || *c.DistanceRatio != 1.5 {
		t.Errorf("mort @20 000 : ratio = %v, attendu 1,5", c.DistanceRatio)
	}
	if c.Vengee || c.DelaiMs != nil {
		t.Errorf("mort @20 000 (hors journal) : attendue non vengee, obtenu vengee=%v delai=%v",
			c.Vengee, c.DelaiMs)
	}

	// Un repere par joueur ayant des morts : medianes et volume.
	if len(nuage.Reperes) != 1 {
		t.Fatalf("reperes = %d, attendu 1 : %+v", len(nuage.Reperes), nuage.Reperes)
	}
	r := nuage.Reperes[0]
	if r.XUID != "x_main" || r.NbMorts != 3 {
		t.Errorf("repere = %s / %d morts, attendu x_main / 3", r.XUID, r.NbMorts)
	}
	// Ratios presents : 0,5 et 1,5 -> mediane 1,0.
	if r.MedianeDistanceRatio == nil || *r.MedianeDistanceRatio != 1.0 {
		t.Errorf("mediane ratio = %v, attendu 1,0", r.MedianeDistanceRatio)
	}
	// Une seule mort vengee (3 000 ms) -> mediane 3 000.
	if r.MedianeDelaiMs == nil || *r.MedianeDelaiMs != 3_000 {
		t.Errorf("mediane delai = %v, attendu 3 000 ms", r.MedianeDelaiMs)
	}
	// PartIsolee : 3 morts examinees, 2 isolees (la mort a 27 m est hors portee du radar
	// de 18 m, celle sans coequipier visible aussi).
	if r.PartIsolee.N != 3 || r.PartIsolee.Brut != 2 {
		t.Errorf("part isolee = %d/%d, attendu 2/3", r.PartIsolee.Brut, r.PartIsolee.N)
	}
}

// TestBuildSquadIsolementNuage_AucunCoequipierVisible : un roster dont TOUTES les morts
// sont hors de vue publie ses points, sans mediane d'abscisse (aucun ratio a mediane).
func TestBuildSquadIsolementNuage_AucunCoequipierVisible(t *testing.T) {
	univers := domain.TacticalUnivers{
		Matchs:  isolementMatches("Arena", "m1"),
		Equipes: equipesDeuxContreDeux("m1"),
	}
	repo := &mockTacticalRepo{
		lecture: domain.TacticalKillEvents{Univers: univers},
		morts: domain.TacticalMortsContexte{Morts: []domain.MortContexte{
			mortContexte("m1", "x_main", 5_000, nil, 0, 1),
			mortContexte("m1", "x_main", 9_000, nil, 0, 1),
		}},
	}
	svc := &TeammatesService{
		titleSlug: "halo_infinite", gamertag: "main",
		tacticalRepo: repo, caps: capsFiables(), radarRange: isolementRadar(),
	}

	rows := isolementRows("m1")
	got := svc.buildSquadEchange(context.Background(), rows, rows, "main", "x_main", nil)
	if got == nil || got.NuageIsolement == nil {
		t.Fatal("nuage isolement attendu, obtenu nil")
	}
	if len(got.NuageIsolement.Morts) != 2 {
		t.Fatalf("morts = %d, attendues 2", len(got.NuageIsolement.Morts))
	}
	r := got.NuageIsolement.Reperes[0]
	if r.MedianeDistanceRatio != nil {
		t.Errorf("mediane ratio = %v, attendue absente (aucun coequipier visible)", *r.MedianeDistanceRatio)
	}
	if r.MedianeDelaiMs != nil {
		t.Errorf("mediane delai = %v, attendue absente (aucune mort vengee)", *r.MedianeDelaiMs)
	}
}

// TestBuildSquadIsolementNuage_MatchSansRayon : une mort survenue sur un match dont la
// variante n'a pas de rayon n'entre PAS dans le nuage — aucune abscisse ne peut s'y lire.
func TestBuildSquadIsolementNuage_MatchSansRayon(t *testing.T) {
	matchs := append(isolementMatches("Arena", "m1"), isolementMatches("Husky", "m2")...)
	univers := domain.TacticalUnivers{Matchs: matchs, Equipes: equipesDeuxContreDeux("m1", "m2")}
	repo := &mockTacticalRepo{
		lecture: domain.TacticalKillEvents{Univers: univers},
		morts: domain.TacticalMortsContexte{Morts: []domain.MortContexte{
			mortContexte("m1", "x_main", 5_000, metres(9), 1, 0),
			mortContexte("m2", "x_main", 6_000, metres(9), 1, 0),
		}},
	}
	svc := &TeammatesService{
		titleSlug: "halo_infinite", gamertag: "main",
		tacticalRepo: repo, caps: capsFiables(), radarRange: isolementRadar(),
	}

	rows := isolementRows("m1", "m2")
	got := svc.buildSquadEchange(context.Background(), rows, rows, "main", "x_main", nil)
	if got == nil || got.NuageIsolement == nil {
		t.Fatal("nuage isolement attendu, obtenu nil")
	}
	if len(got.NuageIsolement.Morts) != 1 || got.NuageIsolement.Morts[0].MatchID != "m1" {
		t.Errorf("morts = %+v, attendue la seule mort de m1 (Husky n'a pas de rayon)",
			got.NuageIsolement.Morts)
	}
}

// TestBuildSquadIsolementNuage_SansTableDeRadar : sans injection (WithRadarRange jamais
// appele), le nuage est ABSENT — jamais un nuage vide qui se lirait comme une mesure a
// zero. Le reste de la section « echange » n'est pas affecte.
func TestBuildSquadIsolementNuage_SansTableDeRadar(t *testing.T) {
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{Univers: universDe("m1")}}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", nil)
	if got == nil {
		t.Fatal("section echange attendue, obtenu nil")
	}
	if got.NuageIsolement != nil {
		t.Error("nuage isolement attendu nil sans table de radar cablee")
	}
}
