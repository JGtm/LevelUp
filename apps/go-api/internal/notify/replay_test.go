package notify

// replay_test.go — l'embed « rejeux 2D prêts » : parité FR/EN, rendu des lignes et
// no-op de gating (aucun réseau touché : le rendu est exercé par buildReplayEmbed).

import (
	"strings"
	"testing"
)

// replayKeys : les clés i18n du lot B. Toute clé ajoutée ici doit exister en FR ET en EN
// (règle CLAUDE.md n°1 : toute chaîne produit est bilingue).
var replayKeys = []string{
	"discord_replay_ready_title",
	"discord_replay_ready_desc_one",
	"discord_replay_ready_desc_many",
	"discord_replay_ready_more",
}

func TestReplayStrings_PariteFREN(t *testing.T) {
	for _, k := range replayKeys {
		entry, ok := discordStrings[k]
		if !ok {
			t.Errorf("clé %q absente de discordStrings", k)
			continue
		}
		for _, lang := range []string{"fr", "en"} {
			if strings.TrimSpace(entry[lang]) == "" {
				t.Errorf("clé %q : traduction %q vide", k, lang)
			}
		}
		if entry["fr"] == entry["en"] {
			t.Errorf("clé %q : FR et EN identiques (%q) — traduction oubliée ?", k, entry["fr"])
		}
	}
	// Pas d'anglicisme dans le texte FR : le projet dit « rejeu », jamais « replay ».
	for _, k := range replayKeys {
		if strings.Contains(strings.ToLower(discordStrings[k]["fr"]), "replay") {
			t.Errorf("clé %q : le texte FR contient « replay » — dire « rejeu »", k)
		}
	}
}

func TestBuildReplayEmbed_ListeLiensEtReste(t *testing.T) {
	cfg := NotifyConfig{Lang: "fr", NotifyReplay: true}
	items := []ReplayReadyItem{
		{MatchID: "aaaa1111", Label: "Aquarius", URL: "https://exemple.test/t/halo_infinite/players/JGtm/matches/aaaa1111-xxxx/replay"},
		{MatchID: "bbbb2222", Label: "Catalyst"},
		{MatchID: "cccc3333"},
	}
	e := buildReplayEmbed(cfg, items, 4)

	if !strings.Contains(e.Description, "7 rejeux") {
		t.Errorf("description = %q : le TOTAL (3 listés + 4 omis) doit être annoncé", e.Description)
	}
	if !strings.Contains(e.Description, "[`aaaa1111`](https://exemple.test/") {
		t.Errorf("description = %q : la ligne avec URL doit être un lien markdown", e.Description)
	}
	if !strings.Contains(e.Description, "`bbbb2222` — Catalyst") {
		t.Errorf("description = %q : sans URL, la ligne garde son libellé de carte", e.Description)
	}
	if !strings.Contains(e.Description, "`cccc3333`") || strings.Contains(e.Description, "`cccc3333` —") {
		t.Errorf("description = %q : sans URL ni libellé, la ligne montre l'identifiant seul", e.Description)
	}
	if !strings.Contains(e.Description, "et 4 autre") {
		t.Errorf("description = %q : le reste omis doit être annoncé", e.Description)
	}
	if e.Footer == nil || !strings.Contains(e.Footer.Text, "LevelUp") {
		t.Errorf("footer = %+v, attendu le footer title-aware", e.Footer)
	}
	if e.Color != colorBlurple {
		t.Errorf("couleur = %d, attendu colorBlurple", e.Color)
	}
}

func TestBuildReplayEmbed_SingulierEtAnglais(t *testing.T) {
	e := buildReplayEmbed(NotifyConfig{Lang: "en"}, []ReplayReadyItem{{MatchID: "aaaa1111"}}, 0)
	if !strings.Contains(e.Description, "**1 replay**") {
		t.Errorf("description EN = %q, attendu le singulier anglais", e.Description)
	}
	fr := buildReplayEmbed(NotifyConfig{Lang: "fr"}, []ReplayReadyItem{{MatchID: "aaaa1111"}}, 0)
	if !strings.Contains(fr.Description, "**1 rejeu**") {
		t.Errorf("description FR = %q, attendu le singulier français", fr.Description)
	}
}

// TestNotifyReplayBatch_NoOp — les trois portes qui rendent false SANS toucher le réseau :
// webhook absent, catégorie coupée, lot vide.
func TestNotifyReplayBatch_NoOp(t *testing.T) {
	items := []ReplayReadyItem{{MatchID: "aaaa1111"}}
	cases := []struct {
		nom     string
		cfg     NotifyConfig
		items   []ReplayReadyItem
		omitted int
	}{
		{"webhook absent", NotifyConfig{Lang: "fr", NotifyReplay: true}, items, 0},
		{"categorie coupee", NotifyConfig{Lang: "fr", WebhookURL: "https://discord.com/api/webhooks/x/y", NotifyReplay: false}, items, 0},
		{"lot vide", NotifyConfig{Lang: "fr", WebhookURL: "https://discord.com/api/webhooks/x/y", NotifyReplay: true}, nil, 0},
	}
	for _, c := range cases {
		if NotifyReplayBatch(c.cfg, c.items, c.omitted) {
			t.Errorf("%s : NotifyReplayBatch a rendu true, attendu false (no-op)", c.nom)
		}
	}
}

// TestLoadNotifyConfig_ReplayDefautActif — la catégorie est livrée ACTIVE : un réglage
// absent vaut true (CLAUDE.md n°11 — aucun flag qui laisse une feature éteinte).
func TestLoadNotifyConfig_ReplayDefautActif(t *testing.T) {
	cfg := notifyConfigFromMap("", map[string]any{
		"discord_notifications_enabled": true,
		"discord_webhook_url":           "https://discord.com/api/webhooks/1/2",
	})
	if !cfg.NotifyReplay {
		t.Error("NotifyReplay = false alors que le réglage est absent — le défaut doit être ACTIF")
	}
	off := notifyConfigFromMap("", map[string]any{
		"discord_notifications_enabled": true,
		"discord_webhook_url":           "https://discord.com/api/webhooks/1/2",
		"discord_notify_replay":         false,
	})
	if off.NotifyReplay {
		t.Error("NotifyReplay = true alors que le réglage vaut false — la coupure doit être respectée")
	}
}
