package service

// replay_demo_mask.go — MASQUE DES JOUEURS RÉELS dans le rejeu servi par la DÉMO.
//
// Décision D-1 (plan des recommandations du 2026-10-09) : les artefacts de rejeu ne sont PAS
// modifiés ; les noms se masquent à l'affichage. Le masque s'applique ici, au document SERVI,
// pour qu'aucune identité réelle ne sorte du serveur démo — ni à l'écran, ni dans la réponse
// HTTP. Les identités de remplacement sont celles du roster démo (index des rejeux figés,
// écrit par seed-demo) : le rejeu nomme donc les joueurs comme la vue match de la démo.
//
// LA RÈGLE, sur l'arbre JSON du document :
//  1. tout objet qui porte un `xuid` connu de l'index : ses champs `name` / `gamertag` sont
//     des noms RÉELS, relevés ;
//  2. toute chaîne égale à un xuid réel (ou `xuid(<réel>)`), à un nom relevé, et toute CLÉ
//     d'objet égale à un xuid réel, est remplacée par l'identité démo correspondante.
//
// Les bots n'ont pas de xuid : leurs noms (génériques) traversent.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
)

// demoNameKeys : les champs qui portent le nom d'un joueur à côté de son xuid.
var demoNameKeys = []string{"name", "gamertag"}

// maskDemoReplay applique le masque de l'index `indexPath` au document servi. Un index
// illisible REFUSE le rejeu (erreur) : servir le document sans masque publierait les noms
// réels.
func maskDemoReplay(ctx context.Context, indexPath string, doc replaydoc.ReplayDocument) (replaydoc.ReplayDocument, error) {
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return replaydoc.ReplayDocument{}, fmt.Errorf("index des rejeux démo: %w", err)
	}
	var index domain.DemoReplayIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return replaydoc.ReplayDocument{}, fmt.Errorf("index des rejeux démo illisible: %w", err)
	}
	masked, err := maskReplayIdentities(doc, index.Identities)
	if err != nil {
		slog.ErrorContext(ctx, "rejeu démo : masque des joueurs impossible — rejeu refusé", "err", err)
		return replaydoc.ReplayDocument{}, err
	}
	return masked, nil
}

// maskReplayIdentities rend le document avec les identités réelles remplacées (cf. en-tête).
func maskReplayIdentities(doc replaydoc.ReplayDocument, ids []domain.DemoReplayIdentity) (replaydoc.ReplayDocument, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return replaydoc.ReplayDocument{}, fmt.Errorf("sérialisation du rejeu: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var tree any
	if err := dec.Decode(&tree); err != nil {
		return replaydoc.ReplayDocument{}, fmt.Errorf("lecture du rejeu: %w", err)
	}
	repl := map[string]string{}
	byXUID := map[string]domain.DemoReplayIdentity{}
	for _, id := range ids {
		if id.XUID == "" {
			continue
		}
		byXUID[id.XUID] = id
		repl[id.XUID] = id.DemoXUID
		repl["xuid("+id.XUID+")"] = "xuid(" + id.DemoXUID + ")"
	}
	collectRealNames(tree, byXUID, repl)
	tree = replaceStrings(tree, repl)
	out, err := json.Marshal(tree)
	if err != nil {
		return replaydoc.ReplayDocument{}, fmt.Errorf("sérialisation du rejeu masqué: %w", err)
	}
	var masked replaydoc.ReplayDocument
	if err := json.Unmarshal(out, &masked); err != nil {
		return replaydoc.ReplayDocument{}, fmt.Errorf("relecture du rejeu masqué: %w", err)
	}
	return masked, nil
}

// collectRealNames relève les noms portés à côté d'un xuid connu (règle 1).
func collectRealNames(node any, byXUID map[string]domain.DemoReplayIdentity, repl map[string]string) {
	switch v := node.(type) {
	case map[string]any:
		if x, ok := v["xuid"].(string); ok {
			if id, known := byXUID[x]; known {
				for _, k := range demoNameKeys {
					if name, ok := v[k].(string); ok && name != "" {
						repl[name] = id.DemoGamertag
					}
				}
			}
		}
		for _, child := range v {
			collectRealNames(child, byXUID, repl)
		}
	case []any:
		for _, child := range v {
			collectRealNames(child, byXUID, repl)
		}
	}
}

// replaceStrings remplace, dans tout l'arbre, les chaînes et les clés connues (règle 2).
func replaceStrings(node any, repl map[string]string) any {
	switch v := node.(type) {
	case string:
		if r, ok := repl[v]; ok {
			return r
		}
		return v
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, child := range v {
			if r, ok := repl[k]; ok {
				k = r
			}
			out[k] = replaceStrings(child, repl)
		}
		return out
	case []any:
		for i, child := range v {
			v[i] = replaceStrings(child, repl)
		}
		return v
	default:
		return v
	}
}
