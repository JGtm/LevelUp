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

func isolementRows(session string, ids ...string) []domain.SquadMatchRow {
	lbl := session
	start := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	out := make([]domain.SquadMatchRow, 0, len(ids))
	for i, id := range ids {
		out = append(out, domain.SquadMatchRow{
			MatchID: id, StartTime: start.Add(time.Duration(i) * time.Minute),
			SessionLabel: &lbl,
		})
	}
	return out
}

// mortsAccompagnees rend n morts EXAMINEES et NON isolees : un coequipier visible a 5 m
// (sous les deux rayons de isolementRadar).
func mortsAccompagnees(matchIDs []string, xuid string, n int) []domain.MortContexte {
	proche := 5.0
	out := make([]domain.MortContexte, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.MortContexte{
			MatchID: matchIDs[i%len(matchIDs)], VictimXUID: xuid,
			Visibles: 1, HorsDeVue: 0, PlusProcheM: &proche,
		})
	}
	return out
}

// mortsIsolees rend n morts EXAMINEES et ISOLEES : un coequipier hors de vue (pas
// « equipe a terre »), aucun visible a portee.
func mortsIsolees(matchIDs []string, xuid string, n int) []domain.MortContexte {
	out := make([]domain.MortContexte, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.MortContexte{
			MatchID: matchIDs[i%len(matchIDs)], VictimXUID: xuid,
			Visibles: 0, HorsDeVue: 1, PlusProcheM: nil,
		})
	}
	return out
}

// mortsEquipeATerre rend n morts ECARTEES du denominateur : personne ne pouvait
// accompagner (Visibles + HorsDeVue == 0).
func mortsEquipeATerre(matchIDs []string, xuid string, n int) []domain.MortContexte {
	out := make([]domain.MortContexte, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.MortContexte{
			MatchID: matchIDs[i%len(matchIDs)], VictimXUID: xuid,
			Visibles: 0, HorsDeVue: 0, PlusProcheM: nil,
		})
	}
	return out
}

// ─── LE NUAGE ──────────────────────────────────────────────────────────────────────────

// TestBuildSquadIsolementNuage couvre les quatre regles d'exclusion du plancher de
// publication, sur cinq sessions distinctes :
//
//	S1  5 morts examinees (3 accompagnees + 2 isolees)      -> point publie, echantillon
//	                                                            faible (N < 30).
//	S2  4 morts examinees, SOUS le plancher de 5             -> aucun point.
//	S3  5 morts, TOUTES « equipe a terre »                   -> 0 examinee -> aucun point.
//	S4  4 morts mesurables + 1 mort SANS RAYON (variante Husky, absente de la table)
//	                                                          -> 4 examinees -> aucun point.
//	S5  30 morts examinees (18 accompagnees + 12 isolees)    -> point publie, PAS
//	                                                            d'echantillon faible.
func TestBuildSquadIsolementNuage(t *testing.T) {
	s1 := []string{"m1", "m2", "m3", "m4", "m5"}
	s2 := []string{"m6", "m7", "m8", "m9"}
	s3 := []string{"m10", "m11", "m12", "m13", "m14"}
	s4Arene := []string{"m15", "m16", "m17", "m18"}
	s4Husky := []string{"m19"}
	s5 := []string{"m20", "m21", "m22"}

	var matchs []domain.TacticalMatch
	matchs = append(matchs, isolementMatches("Arena", s1...)...)
	matchs = append(matchs, isolementMatches("Arena", s2...)...)
	matchs = append(matchs, isolementMatches("Arena", s3...)...)
	matchs = append(matchs, isolementMatches("Arena", s4Arene...)...)
	matchs = append(matchs, isolementMatches("Husky", s4Husky...)...)
	matchs = append(matchs, isolementMatches("Arena", s5...)...)

	var scopeRows []domain.SquadMatchRow
	scopeRows = append(scopeRows, isolementRows("S1", s1...)...)
	scopeRows = append(scopeRows, isolementRows("S2", s2...)...)
	scopeRows = append(scopeRows, isolementRows("S3", s3...)...)
	s4Tout := append(append([]string{}, s4Arene...), s4Husky...)
	scopeRows = append(scopeRows, isolementRows("S4", s4Tout...)...)
	scopeRows = append(scopeRows, isolementRows("S5", s5...)...)

	var morts []domain.MortContexte
	morts = append(morts, mortsAccompagnees(s1[:3], "x_main", 3)...)
	morts = append(morts, mortsIsolees(s1[3:], "x_main", 2)...)
	morts = append(morts, mortsAccompagnees(s2, "x_main", 4)...)
	morts = append(morts, mortsEquipeATerre(s3, "x_main", 5)...)
	morts = append(morts, mortsAccompagnees(s4Arene, "x_main", 4)...)
	morts = append(morts, mortsAccompagnees(s4Husky, "x_main", 1)...)
	morts = append(morts, mortsAccompagnees(s5, "x_main", 18)...)
	morts = append(morts, mortsIsolees(s5, "x_main", 12)...)

	repo := &mockTacticalRepo{
		lecture: domain.TacticalKillEvents{
			Univers: domain.TacticalUnivers{Matchs: matchs, Equipes: domain.EquipesParMatch{}},
		},
		morts: domain.TacticalMortsContexte{Morts: morts},
	}
	svc := &TeammatesService{
		titleSlug: "halo_infinite", gamertag: "main",
		tacticalRepo: repo, caps: capsFiables(), radarRange: isolementRadar(),
	}

	got := svc.buildSquadEchange(context.Background(), scopeRows, scopeRows, "main", "x_main", nil)
	if got == nil {
		t.Fatal("section echange attendue, obtenu nil")
	}
	if got.NuageIsolement == nil {
		t.Fatal("nuage isolement attendu, obtenu nil")
	}
	if got.NuageIsolement.PlancherMortsSession != domain.PlancherMortsSessionIsolement {
		t.Errorf("plancher_morts_session = %d, attendu %d",
			got.NuageIsolement.PlancherMortsSession, domain.PlancherMortsSessionIsolement)
	}

	parSession := map[string]domain.SquadIsolementPoint{}
	for _, p := range got.NuageIsolement.Points {
		parSession[p.SessionLabel] = p
	}
	if len(got.NuageIsolement.Points) != 2 {
		t.Fatalf("points = %d, attendus 2 (S1 et S5 seulement) : %+v",
			len(got.NuageIsolement.Points), got.NuageIsolement.Points)
	}

	p1, ok := parSession["S1"]
	if !ok {
		t.Fatal("point S1 attendu, absent")
	}
	if p1.MortsExaminees != 5 || p1.MortsIsolees != 2 {
		t.Errorf("S1 = %d examinees / %d isolees, attendu 5/2", p1.MortsExaminees, p1.MortsIsolees)
	}
	if !p1.PartIsolee.EchantillonFaible {
		t.Error("S1 (N=5) attendu EchantillonFaible=true")
	}

	if _, ok := parSession["S2"]; ok {
		t.Error("S2 (4 morts, sous le plancher de 5) ne devrait publier aucun point")
	}
	if _, ok := parSession["S3"]; ok {
		t.Error("S3 (toutes equipe a terre) ne devrait publier aucun point")
	}
	if _, ok := parSession["S4"]; ok {
		t.Error("S4 (4 morts mesurables + 1 sans rayon) ne devrait publier aucun point")
	}

	p5, ok := parSession["S5"]
	if !ok {
		t.Fatal("point S5 attendu, absent")
	}
	if p5.MortsExaminees != 30 || p5.MortsIsolees != 12 {
		t.Errorf("S5 = %d examinees / %d isolees, attendu 30/12", p5.MortsExaminees, p5.MortsIsolees)
	}
	if p5.PartIsolee.EchantillonFaible {
		t.Error("S5 (N=30) attendu EchantillonFaible=false")
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
