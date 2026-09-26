// Package notify — replay.go : l'embed « rejeux 2D prêts » (lot B v7.5, point 5 de
// l'encadré Notion « REPLAY 2D »).
//
// UN MESSAGE PAR LOT, JAMAIS PAR ARTEFACT : le groupement est décidé en amont
// (internal/replaynotify) ; ici on ne fait que rendre le lot reçu. Failsafe comme le
// reste du paquet — no-op si webhook absent ou catégorie coupée, jamais d'erreur
// propagée, jamais de panique qui remonte à la boucle appelante.
package notify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// replayListSeparator : préfixe de ligne de la liste des matchs (une seule écriture).
const replayListSeparator = "\n- "

// ReplayReadyItem : une ligne de la liste des rejeux prêts.
//
// URL et Label sont OPTIONNELS et le rendu dégrade proprement : sans URL la ligne montre
// l'identifiant du match en style code, sans Label elle ne montre que lui. C'est voulu —
// la résolution du lien dépend d'une base publique configurée et d'un joueur connu dans
// le match, deux choses qui peuvent manquer sans que le message doive être annulé.
type ReplayReadyItem struct {
	// MatchID : l'identifiant affiché (forme courte, telle que l'appelant l'a réduite).
	MatchID string
	// Label : libellé humain optionnel (nom de carte).
	Label string
	// URL : lien absolu optionnel vers la page de rejeu.
	URL string
}

// NotifyReplayBatch envoie UN message pour un lot de rejeux devenus disponibles.
// Retourne true si Discord a accepté le message.
//
// omitted = artefacts du lot qui ne sont PAS énumérés (plafond de liste) ; ils comptent
// dans le total annoncé et donnent la mention « et N autres ».
func NotifyReplayBatch(cfg NotifyConfig, items []ReplayReadyItem, omitted int) (sent bool) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			sent = false
			slog.ErrorContext(ctx, "discord_replay_panic", "op", "replay_ready",
				"recover", fmt.Sprintf("%v", r))
		}
	}()

	if cfg.WebhookURL == "" || !cfg.NotifyReplay {
		return false
	}
	total := len(items) + omitted
	if total <= 0 {
		return false
	}

	return SendWebhook(cfg.WebhookURL, WebhookPayload{
		Embeds: []Embed{buildReplayEmbed(cfg, items, omitted)},
	})
}

// buildReplayEmbed rend l'embed d'un lot. Séparé de l'envoi pour être exerçable sans
// réseau (même découpage que BuildSyncEmbedWithLabels / BuildCoachEmbed).
func buildReplayEmbed(cfg NotifyConfig, items []ReplayReadyItem, omitted int) Embed {
	total := len(items) + omitted
	descKey := "discord_replay_ready_desc_many"
	if total == 1 {
		descKey = "discord_replay_ready_desc_one"
	}
	var b strings.Builder
	b.WriteString(T(descKey, cfg.Lang, "count", total))
	for _, it := range items {
		b.WriteString(replayListSeparator)
		b.WriteString(replayLine(it))
	}
	if omitted > 0 {
		b.WriteString(replayListSeparator)
		b.WriteString(T("discord_replay_ready_more", cfg.Lang, "count", omitted))
	}
	return Embed{
		Title:       T("discord_replay_ready_title", cfg.Lang),
		Description: b.String(),
		Color:       colorBlurple,
		Footer:      &EmbedFooter{Text: discordFooterText(cfg.Labels)},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// replayLine rend une ligne : identifiant (lié si on a une URL) puis libellé si on en a un.
func replayLine(it ReplayReadyItem) string {
	id := "`" + it.MatchID + "`"
	if it.URL != "" {
		id = "[" + id + "](" + it.URL + ")"
	}
	if it.Label != "" {
		return id + " — " + it.Label
	}
	return id
}
