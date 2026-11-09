package coach_advisor

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/i18n"
)

func decodeReason(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("reason_params illisibles : %v (%q)", err, raw)
	}
	return m
}

// Une proposition de défi déjà stockée (`label_fr`, `label_en`) sort avec un seul `label`.
func TestResolveReasonParams_DefiDejaStocke(t *testing.T) {
	stored := `{"metric":"kda","signal_kind":"strength","label_en":"Improve KDA","label_fr":"Améliore le KDA"}`
	for l, want := range map[i18n.Locale]string{i18n.LocaleFR: "Améliore le KDA", i18n.LocaleEN: "Improve KDA"} {
		out, err := ResolveReasonParams(stored, l)
		if err != nil {
			t.Fatal(err)
		}
		got := decodeReason(t, out)
		if got[ReasonParamLabel] != want {
			t.Errorf("langue %s : label = %v, attendu %q", l, got[ReasonParamLabel], want)
		}
		if _, ok := got["label_fr"]; ok {
			t.Errorf("langue %s : la paire par langue ne doit pas sortir : %v", l, got)
		}
		if got["metric"] != "kda" {
			t.Errorf("langue %s : les autres paramètres doivent rester : %v", l, got)
		}
	}
}

// Un arc proposé porte un titre et une description, résolus ensemble.
func TestResolveReasonParams_Arc(t *testing.T) {
	stored := `{"radar_axis":"combat","step_count":3,"title_en":"Combat Mastery","title_fr":"Domination en combat",` +
		`"description_en":"A 3-step arc","description_fr":"Un arc en 3 étapes"}`
	out, err := ResolveReasonParams(stored, i18n.LocaleEN)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeReason(t, out)
	if got[ReasonParamTitle] != "Combat Mastery" || got[ReasonParamDescription] != "A 3-step arc" {
		t.Errorf("arc : %v", got)
	}
}

// Ce que le coach écrit aujourd'hui se relit : écriture (AddParams, toutes les langues) puis lecture.
func TestResolveReasonParams_RelitCeQueLeCoachEcrit(t *testing.T) {
	m := map[string]any{"metric": "kda"}
	i18n.Text{i18n.LocaleFR: "Frags", i18n.LocaleEN: "Kills"}.AddParams(m, ReasonParamLabel)
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ResolveReasonParams(string(raw), i18n.LocaleEN)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeReason(t, out); got[ReasonParamLabel] != "Kills" || len(got) != 2 {
		t.Errorf("relecture : %v", got)
	}
}

func TestResolveReasonParams_VideEtIllisible(t *testing.T) {
	if out, err := ResolveReasonParams("", i18n.LocaleFR); out != "" || err != nil {
		t.Errorf("vide : %q %v", out, err)
	}
	if _, err := ResolveReasonParams("{pas du json", i18n.LocaleFR); err == nil {
		t.Error("JSON illisible : erreur attendue")
	}
}
