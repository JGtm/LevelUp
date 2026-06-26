// Package invariants — tests unitaires des helpers PURS (zéro DB). Les checks
// SQL eux-mêmes sont exercés par invariants_violation_test.go (tag integration).
package invariants

import "testing"

// TestReportFailures : Failures ne retient QUE les violations fail (les warn,
// dérives tolérées, ne doivent jamais faire échouer un gate). Préserve l'ordre.
func TestReportFailures(t *testing.T) {
	rep := Report{
		XUID: "x1",
		Violations: []Violation{
			{Key: "warn_a", Severity: SeverityWarn},
			{Key: "fail_a", Severity: SeverityFail},
			{Key: "warn_b", Severity: SeverityWarn},
			{Key: "fail_b", Severity: SeverityFail},
		},
	}
	got := rep.Failures()
	if len(got) != 2 {
		t.Fatalf("Failures() len = %d, want 2 (seuls les fail)", len(got))
	}
	if got[0].Key != "fail_a" || got[1].Key != "fail_b" {
		t.Errorf("Failures() = %v, ordre/contenu inattendu", got)
	}
}

// TestReportFailures_AllWarn : un rapport 100% warn ne produit aucune failure
// (slice vide non-nil) — le gate passe.
func TestReportFailures_AllWarn(t *testing.T) {
	rep := Report{Violations: []Violation{
		{Key: "w1", Severity: SeverityWarn},
		{Key: "w2", Severity: SeverityWarn},
	}}
	if got := rep.Failures(); len(got) != 0 {
		t.Fatalf("Failures(all warn) = %v, want vide", got)
	}
}

// TestReportFailures_Empty : un rapport SANS violation (cas dominant du gate —
// dataset sain) produit un résultat de longueur 0. C'est exactement la
// condition de passage du gate (`len(report.Failures()) > 0`, cf.
// invariants_gate_integration_test.go) ; on cadenasse qu'un rapport propre ne
// fabrique jamais de faux échec.
func TestReportFailures_Empty(t *testing.T) {
	if got := (Report{XUID: "clean"}).Failures(); len(got) != 0 {
		t.Errorf("Failures(rapport vide) = %v, want longueur 0 (gate passe)", got)
	}
}

// TestViolationString : le format de diagnostic inclut sévérité, clé, count,
// sample et description — la ligne lue dans les logs/dashboards.
func TestViolationString(t *testing.T) {
	v := Violation{
		Key:         "enrichment_missing",
		Severity:    SeverityFail,
		Count:       3,
		Sample:      []string{"m1", "m2"},
		Description: "matchs sans enrichment",
	}
	got := v.String()
	want := "[fail] enrichment_missing: count=3 sample=[m1 m2] — matchs sans enrichment"
	if got != want {
		t.Errorf("String() =\n  %q\nwant\n  %q", got, want)
	}
}

// TestViolationString_EmptySample : sans échantillon, la ligne rend `sample=[]`
// (et non `sample=<nil>` ni un panic) — une violation peut être construite à la
// main côté dashboard/sentinelle avant d'être loggée. Pin du format `%v` sur
// slice nil pour que le parseur de logs reste stable.
func TestViolationString_EmptySample(t *testing.T) {
	v := Violation{Key: "k", Severity: SeverityWarn, Count: 0}
	got := v.String()
	want := "[warn] k: count=0 sample=[] — "
	if got != want {
		t.Errorf("String(sample nil) =\n  %q\nwant\n  %q", got, want)
	}
}

// TestCapSample : le sample est borné à sampleCap (5) — protège logs/dashboards
// d'un dump massif. En deçà, la liste est rendue telle quelle.
func TestCapSample(t *testing.T) {
	if got := capSample([]string{"a", "b"}); len(got) != 2 {
		t.Errorf("capSample(2) len = %d, want 2", len(got))
	}
	exactly := []string{"a", "b", "c", "d", "e"}
	if got := capSample(exactly); len(got) != sampleCap {
		t.Errorf("capSample(=cap) len = %d, want %d", len(got), sampleCap)
	}
	over := []string{"a", "b", "c", "d", "e", "f", "g"}
	got := capSample(over)
	if len(got) != sampleCap {
		t.Fatalf("capSample(>cap) len = %d, want %d", len(got), sampleCap)
	}
	if got[0] != "a" || got[sampleCap-1] != "e" {
		t.Errorf("capSample(>cap) = %v, want les %d premiers", got, sampleCap)
	}
	if got := capSample(nil); got != nil {
		t.Errorf("capSample(nil) = %v, want nil", got)
	}
}
