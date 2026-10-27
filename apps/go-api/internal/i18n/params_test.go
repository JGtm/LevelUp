package i18n

import (
	"encoding/json"
	"testing"
)

func TestAddParams_UneCleParLangue(t *testing.T) {
	m := map[string]any{}
	Text{LocaleFR: "Jalon", LocaleEN: "Milestone"}.AddParams(m, "title")
	if m["title_fr"] != "Jalon" || m["title_en"] != "Milestone" {
		t.Errorf("clés écrites : %v", m)
	}
	if len(m) != len(Locales()) {
		t.Errorf("une clé par langue attendue : %v", m)
	}
}

func TestResolveParams_LigneDejaStockee(t *testing.T) {
	// Format écrit avant l'arrivée d'une troisième langue : `title_fr`, `title_en` seulement.
	stored := map[string]any{"milestone_id": "m1", "title_fr": "Jalon", "title_en": "Milestone", "threshold": 3.0}
	for l, want := range map[Locale]string{LocaleFR: "Jalon", LocaleEN: "Milestone", Locale("zz"): "Milestone"} {
		got := ResolveParams(stored, []string{"title"}, l)
		if got["title"] != want {
			t.Errorf("langue %q : title = %v, attendu %q", l, got["title"], want)
		}
		if _, ok := got["title_fr"]; ok {
			t.Errorf("langue %q : la paire par langue ne doit pas sortir : %v", l, got)
		}
		if got["milestone_id"] != "m1" || got["threshold"] != 3.0 {
			t.Errorf("langue %q : les autres paramètres doivent rester : %v", l, got)
		}
	}
	if _, ok := stored["title_fr"]; !ok {
		t.Error("l'entrée ne doit pas être modifiée")
	}
}

func TestResolveParams_RepliSurUneLangueRenseignee(t *testing.T) {
	stored := map[string]any{"title_fr": "Seul", "title_en": ""}
	if got := ResolveParams(stored, []string{"title"}, LocaleEN); got["title"] != "Seul" {
		t.Errorf("nom présent dans une seule langue : %v", got)
	}
}

func TestResolveParams_BaseAbsenteLaisseTelle(t *testing.T) {
	stored := map[string]any{"autre": 1.0}
	got := ResolveParams(stored, []string{"title"}, LocaleFR)
	if len(got) != 1 || got["autre"] != 1.0 {
		t.Errorf("base absente : %v", got)
	}
	if ResolveParams(nil, []string{"title"}, LocaleFR) != nil {
		t.Error("nil attendu pour une entrée nil")
	}
}

func TestResolveParamsJSON(t *testing.T) {
	out, err := ResolveParamsJSON(`{"label_fr":"Frags","label_en":"Kills","metric":"kda"}`, []string{"label"}, LocaleEN)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	if m["label"] != "Kills" || m["metric"] != "kda" || len(m) != 2 {
		t.Errorf("résultat : %v", m)
	}
	if out, err := ResolveParamsJSON("", []string{"label"}, LocaleEN); out != "" || err != nil {
		t.Errorf("JSON vide : %q %v", out, err)
	}
	if _, err := ResolveParamsJSON("{pas du json", []string{"label"}, LocaleEN); err == nil {
		t.Error("JSON illisible : erreur attendue")
	}
}
