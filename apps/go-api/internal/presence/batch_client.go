// Package presence — batch_client.go : présence de PLUSIEURS users en un appel.
//
// Pourquoi un batch et pas N fois GetPresence : la liste d'amis des Réglages
// (`friend_gamertags`) est interrogée À LA DEMANDE, sur une requête utilisateur.
// N appels séquentiels de ~200 ms chacun tiendraient la requête HTTP ouverte
// plusieurs secondes et multiplieraient le quota Xbox par N pour une information
// unique (« combien d'amis sont en jeu ? »). L'endpoint batch officiel rend les
// N présences en un aller-retour.
//
// Endpoint : POST https://userpresence.xboxlive.com/users/batch
// Même authentification que le poll unitaire (header XBL3.0 du client partagé,
// rafraîchi par UpdateAuth) et même `x-xbl-contract-version: 3`.
// Corps : {"users":["<xuid>",…],"level":"all"}.
//
// `level: "all"` est REQUIS pour notre usage : sans lui la réponse se limite au
// niveau « user » (state global) et ne porte pas `devices[].titles[]` — or c'est
// exactement le titre actif qui décide si un ami est « en jeu » sur un titre
// suivi. Le parsing réutilise ParsePresencePayload (event_parser.go), donc le
// format d'un élément est le même que celui du poll unitaire.
package presence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/platform/netguard"
)

const (
	// restPresenceBatchURL est l'URL REST de la présence de plusieurs users.
	restPresenceBatchURL = "https://userpresence.xboxlive.com/users/batch"

	// maxBatchPresenceUsers borne le nombre de xuids envoyés en un appel. La
	// liste d'amis vient d'un champ de Réglages saisi à la main : quelques
	// dizaines d'entrées au plus. La borne n'existe donc pas pour découper un
	// gros volume mais pour qu'une liste absurde (collage accidentel) n'envoie
	// jamais un corps démesuré à Xbox — au-delà, on tronque et on loggue.
	maxBatchPresenceUsers = 200

	// batchPresenceMaxBodyBytes borne la lecture de la réponse. 200 présences
	// « level=all » pèsent ~200 Ko ; 1 Mio laisse une marge large sans exposer
	// le process à une réponse anormale.
	batchPresenceMaxBodyBytes = 1 << 20

	// batchPresenceSurface nomme ce point de sortie pour le coupe-circuit démo.
	// Distinct du poll unitaire (`presence/rest_client.go`) : les deux sortent
	// vers le même hôte, mais pas par le même chemin ni pour le même
	// déclencheur — le journal doit pouvoir dire lequel aurait appelé.
	batchPresenceSurface = "xbox_presence.batch"
)

// batchPresenceRequest est le corps POST /users/batch.
type batchPresenceRequest struct {
	Users []string `json:"users"`
	Level string   `json:"level"`
}

// GetPresenceBatch interroge Xbox pour la présence de plusieurs users en un
// appel. Retourne un PresenceEvent par élément PARSABLE de la réponse — les
// éléments illisibles sont ignorés (log Debug) plutôt que de faire échouer tout
// le lot : une présence manquante coûte un ami non compté, une erreur coûterait
// le compteur entier.
//
// Les xuids vides sont écartés ; une liste vide après nettoyage retourne nil
// sans aucun appel réseau. En MODE DÉMO le lot ne part pas non plus (netguard) :
// même sortie vide, même absence d'erreur. Les erreurs HTTP sont rendues telles
// quelles (*HTTPError) pour que l'appelant discrimine 401 / 429 / 5xx comme sur
// le chemin unitaire.
//
// ⚠ Un ami dont la présence est MASQUÉE (privacy Xbox) n'apparaît pas dans la
// réponse, ou y apparaît sans titre : il n'est simplement pas compté. Ce n'est
// pas une erreur — cf. PresenceCounter côté service.
func (c *PresenceClient) GetPresenceBatch(ctx context.Context, xuids []string) ([]PresenceEvent, error) {
	users := make([]string, 0, len(xuids))
	for _, x := range xuids {
		if x != "" {
			users = append(users, x)
		}
	}
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) > maxBatchPresenceUsers {
		slog.WarnContext(ctx, "rest_presence: lot tronqué",
			"requested", len(users), "max", maxBatchPresenceUsers)
		users = users[:maxBatchPresenceUsers]
	}

	// Coupe-circuit démo. CE CHEMIN-CI EST ATTEIGNABLE EN DÉMO, contrairement au
	// poll unitaire : le poll appartient au daemon watcher (désactivé par les
	// app_settings de la démo, d'où son entrée d'allowlist), tandis que le lot
	// part d'une REQUÊTE UTILISATEUR — GET /api/v1/presence, que le shell tire
	// toutes les 30 s dès qu'un onglet est ouvert. Sans ce garde, une démo
	// tournant sur un poste porteur de tokens martèlerait Xbox pour les xuids
	// factices de la fixture.
	//
	// Dégradation : aucune présence connue, donc aucun ami compté — la même
	// sortie qu'une liste vide (cf. godoc), et NON une erreur. Une erreur ferait
	// journaliser un avertissement au compteur d'amis toutes les 45 s pour un
	// refus attendu, et déclencherait son backoff d'échec pour rien.
	if err := netguard.Check(ctx, batchPresenceSurface); err != nil {
		return nil, nil
	}

	body, err := json.Marshal(batchPresenceRequest{Users: users, Level: "all"})
	if err != nil {
		return nil, fmt.Errorf("rest presence batch: encode body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, restPresenceBatchURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rest presence batch: build req: %w", err)
	}
	req.Header.Set("Authorization", c.AuthHeader())
	req.Header.Set("x-xbl-contract-version", "3")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rest presence batch: do req: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		slog.DebugContext(ctx, "rest_presence: réponse batch non-OK",
			"users", len(users), "status", resp.StatusCode, "body", string(raw))
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, batchPresenceMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("rest presence batch: read body: %w", err)
	}

	// La réponse est un TABLEAU d'enregistrements de présence, chacun au format
	// déjà connu du poll unitaire (xuid + state + devices[].titles[]).
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("rest presence batch: parse: %w", err)
	}

	events := make([]PresenceEvent, 0, len(items))
	for i, item := range items {
		// Pas de fallback xuid : dans un lot, l'identité ne peut venir que du
		// payload — attribuer le xuid d'un autre ami serait pire que d'ignorer.
		ev, err := ParsePresencePayload(item, "")
		if err != nil {
			slog.DebugContext(ctx, "rest_presence: élément de lot illisible — ignoré",
				"index", i, "err", err)
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}
