package persist

import "testing"

func passeMainmiseValide() LeadShareBatch {
	return LeadShareBatch{
		MatchID: "m1", Measured: true, DocSchema: 71,
		MeasuredMS: 5000, TieMS: 3000, LeadChanges: 2,
		Rows: []LeadShareRow{{Camp: 0, LeadMS: 1500}, {Camp: 1, LeadMS: 500}},
	}
}

func TestValidateLeadShareBatch_PasseValide(t *testing.T) {
	if err := ValidateLeadShareBatch(passeMainmiseValide()); err != nil {
		t.Fatalf("une passe cohérente est refusée : %v", err)
	}
	refusee := LeadShareBatch{MatchID: "m1", Reason: "mode_events", DocSchema: 71}
	if err := ValidateLeadShareBatch(refusee); err != nil {
		t.Fatalf("une passe non mesurée avec sa raison est refusée : %v", err)
	}
}

func TestValidateLeadShareBatch_Refus(t *testing.T) {
	cas := []struct {
		nom    string
		mutate func(*LeadShareBatch)
	}{
		{"match sans identifiant", func(b *LeadShareBatch) { b.MatchID = "" }},
		{"non mesurée sans raison", func(b *LeadShareBatch) { *b = LeadShareBatch{MatchID: "m1"} }},
		{"non mesurée avec des lignes", func(b *LeadShareBatch) {
			*b = LeadShareBatch{MatchID: "m1", Reason: "mode_events", Rows: []LeadShareRow{{Camp: 0}}}
		}},
		{"non mesurée avec une durée", func(b *LeadShareBatch) {
			*b = LeadShareBatch{MatchID: "m1", Reason: "mode_events", MeasuredMS: 10}
		}},
		{"mesurée avec une raison", func(b *LeadShareBatch) { b.Reason = "mode_events" }},
		{"mesurée sans durée", func(b *LeadShareBatch) { b.MeasuredMS = 0 }},
		{"mesurée sans camp", func(b *LeadShareBatch) { b.Rows = nil }},
		{"camp hors 0..8", func(b *LeadShareBatch) { b.Rows[0].Camp = 9 }},
		{"temps négatif", func(b *LeadShareBatch) { b.Rows[0].LeadMS = -1 }},
		{"doublon de camp", func(b *LeadShareBatch) { b.Rows[1].Camp = 0 }},
		{"somme des temps fausse", func(b *LeadShareBatch) { b.TieMS = 2999 }},
		{"retournements négatifs", func(b *LeadShareBatch) { b.LeadChanges = -1 }},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := passeMainmiseValide()
			c.mutate(&b)
			if err := ValidateLeadShareBatch(b); err == nil {
				t.Errorf("la passe %+v devait être refusée", b)
			}
		})
	}
}
