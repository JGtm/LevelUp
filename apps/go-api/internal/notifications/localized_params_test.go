package notifications

import (
	"context"
	"encoding/json"
	"testing"

	"levelup/go-api/internal/i18n"
)

func decodeParams(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("params illisibles : %v", err)
	}
	return m
}

// Une notification de jalon déjà stockée (`title_fr`, `title_en`) sort avec un seul `title`, dans la
// langue de la requête, sans paire par langue ; les autres paramètres restent.
func TestLocalizeParams_JalonDejaStocke(t *testing.T) {
	stored := json.RawMessage(`{"milestone_id":"m1","title_en":"Sharpshooter","title_fr":"Tireur d'élite","threshold":50}`)
	for l, want := range map[i18n.Locale]string{i18n.LocaleFR: "Tireur d'élite", i18n.LocaleEN: "Sharpshooter"} {
		items := []Notification{{ID: 1, Params: append(json.RawMessage(nil), stored...)}}
		LocalizeParams(context.Background(), items, l)
		got := decodeParams(t, items[0].Params)
		if got[ParamTitle] != want {
			t.Errorf("langue %s : title = %v, attendu %q", l, got[ParamTitle], want)
		}
		if _, ok := got["title_fr"]; ok {
			t.Errorf("langue %s : la paire par langue ne doit pas sortir : %v", l, got)
		}
		if got["milestone_id"] != "m1" || got["threshold"] != 50.0 {
			t.Errorf("langue %s : les autres paramètres doivent rester : %v", l, got)
		}
	}
}

// Une langue absente de la ligne (écrite avant son arrivée) lit la langue de repli.
func TestLocalizeParams_LangueAbsenteDeLaLigne(t *testing.T) {
	items := []Notification{{Params: json.RawMessage(`{"medal_name_fr":"Tir parfait","medal_name_en":"Perfect"}`)}}
	LocalizeParams(context.Background(), items, i18n.Locale("zz"))
	if got := decodeParams(t, items[0].Params); got[ParamMedalName] != "Perfect" {
		t.Errorf("repli attendu sur la langue de repli : %v", got)
	}
}

func TestLocalizeParams_SansParamsOuIllisibles(t *testing.T) {
	items := []Notification{{ID: 1}, {ID: 2, Params: json.RawMessage(`{pas du json`)}}
	LocalizeParams(context.Background(), items, i18n.LocaleFR)
	if len(items[0].Params) != 0 {
		t.Errorf("sans params : inchangé, obtenu %q", items[0].Params)
	}
	if string(items[1].Params) != `{pas du json` {
		t.Errorf("params illisibles : servis bruts, obtenu %q", items[1].Params)
	}
}

// Le nom du rang de carrière (rank_name_<code>) se lit dans la langue de la requête ; une ligne écrite avant
// l'item 11.R, qui ne porte que `rank_name`, reste affichable telle quelle.
func TestLocalizeParams_NomDuRang(t *testing.T) {
	nouvelle := json.RawMessage(`{"rank":192,"rank_name_fr":"Colonel Platine II","rank_name_en":"Colonel Platinum II","previous":191}`)
	for l, want := range map[i18n.Locale]string{i18n.LocaleFR: "Colonel Platine II", i18n.LocaleEN: "Colonel Platinum II"} {
		items := []Notification{{Params: append(json.RawMessage(nil), nouvelle...)}}
		LocalizeParams(context.Background(), items, l)
		got := decodeParams(t, items[0].Params)
		if got[ParamRankName] != want {
			t.Errorf("langue %s : rank_name = %v, attendu %q", l, got[ParamRankName], want)
		}
		if _, ok := got["rank_name_fr"]; ok {
			t.Errorf("langue %s : la paire par langue ne doit pas sortir : %v", l, got)
		}
	}

	ancienne := json.RawMessage(`{"rank":190,"rank_name":"Colonel Platine II","previous":189}`)
	for _, l := range []i18n.Locale{i18n.LocaleFR, i18n.LocaleEN} {
		items := []Notification{{Params: append(json.RawMessage(nil), ancienne...)}}
		LocalizeParams(context.Background(), items, l)
		if got := decodeParams(t, items[0].Params); got[ParamRankName] != "Colonel Platine II" {
			t.Errorf("langue %s : une ligne ancienne (rank_name seul) doit rester affichée : %v", l, got)
		}
	}
}
