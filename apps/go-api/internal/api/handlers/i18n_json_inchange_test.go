package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/progression/coach_advisor"
)

// Preuve de l'étape 5 du plan i18n (lot B) : les données internes sont des i18n.Text, le JSON de
// l'API ne bouge pas. Ici les propositions du coach (pont vers Prestige : `reason_key_en` /
// `reason_key_fr`, omitempty) et le catalogue de saisons des field-mappings (clé `label`).

func decodeObject(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal : %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal : %v", err)
	}
	return out
}

func TestContratLangue_PropositionDuCoach(t *testing.T) {
	p := coach_advisor.Proposal{
		ID: "p1", Kind: coach_advisor.ProposalKindArc,
		ReasonKey: i18n.FromColumns("coach.proposal.arc.combat.en", "coach.proposal.arc.combat.fr"),
		CreatedAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC),
	}
	// Un seul champ `reason_key`, dans la langue demandée : plus de `reason_key_en` / `reason_key_fr`.
	fr := decodeObject(t, toProposalDTO(context.Background(), p, i18n.LocaleFR))
	en := decodeObject(t, toProposalDTO(context.Background(), p, i18n.LocaleEN))
	if fr["reason_key"] != "coach.proposal.arc.combat.fr" || en["reason_key"] != "coach.proposal.arc.combat.en" {
		t.Errorf("clé de motif : fr=%v en=%v", fr["reason_key"], en["reason_key"])
	}
	for _, k := range []string{"reason_key_en", "reason_key_fr"} {
		if _, ok := fr[k]; ok {
			t.Errorf("%s ne fait plus partie du contrat : %v", k, fr)
		}
	}

	// Sans motif : la clé est omise (omitempty).
	bare := decodeObject(t, toProposalDTO(context.Background(), coach_advisor.Proposal{ID: "p2"}, i18n.LocaleFR))
	if _, ok := bare["reason_key"]; ok {
		t.Errorf("reason_key doit être omis sans motif : %v", bare)
	}
}

func TestJSONInchange_LibelleDeSaison(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	catalog := []SeasonCatalogEntry{
		{ID: "s1", Label: i18n.FromColumns("Heroes of Reach", "Héros de Reach"), Start: start},
		{ID: "s2", Label: i18n.FromColumns("Shadows", "Shadows"), Start: start},
	}
	fr := decodeObject(t, projectCatalogToBucket(catalog, "fr"))
	en := decodeObject(t, projectCatalogToBucket(catalog, "en"))
	label := func(bucket map[string]any, id string) any {
		return bucket[id].(map[string]any)["label"]
	}
	if label(fr, "s1") != "Héros de Reach" || label(en, "s1") != "Heroes of Reach" {
		t.Errorf("s1 : fr=%v en=%v", label(fr, "s1"), label(en, "s1"))
	}
	if label(fr, "s2") != "Shadows" || label(en, "s2") != "Shadows" {
		t.Errorf("s2 : fr=%v en=%v", label(fr, "s2"), label(en, "s2"))
	}
}
