package notifications

import "encoding/json"

// coalesce.go — helpers purs pour la coalescence de notifications (EmitCoalesced).
// Extraction de l'acteur et du compteur depuis un EmitInput / une Notification
// persistée, tolérants aux payloads absents ou mal formés.

// actorNameOf extrait le nom d'acteur d'une entrée : Actor.Name en priorité,
// sinon le param `actor_name`. Retourne "" si aucun acteur (ex. sync_error) —
// l'appelant matche alors la catégorie seule.
func actorNameOf(actor *Actor, params map[string]any) string {
	if actor != nil && actor.Name != "" {
		return actor.Name
	}
	if params != nil {
		if v, ok := params["actor_name"].(string); ok {
			return v
		}
	}
	return ""
}

// actorNameOfNotif extrait le nom d'acteur d'une notification persistée :
// Actor.Name en priorité, sinon le param `actor_name` décodé du JSON.
func actorNameOfNotif(n *Notification) string {
	if n.Actor != nil && n.Actor.Name != "" {
		return n.Actor.Name
	}
	if s, ok := jsonStringField(n.Params, "actor_name"); ok {
		return s
	}
	return ""
}

// coalescedCountOf lit le `count` d'une notification candidate. Une notif sans
// count (ex. premier sync_error) compte pour 1 — la coalescence part de la base
// « il y en avait déjà 1 ».
func coalescedCountOf(n *Notification) int {
	if v, ok := jsonNumberField(n.Params, paramKeyCount); ok {
		return v
	}
	return 1
}

// inputCountOf lit le `count` d'un EmitInput (défaut 1 si absent).
func inputCountOf(in EmitInput) int {
	if in.Params != nil {
		switch v := in.Params[paramKeyCount].(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 1
}

// jsonStringField décode un champ string d'un payload JSON de notification.
func jsonStringField(raw json.RawMessage, key string) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", false
	}
	s, ok := m[key].(string)
	return s, ok
}

// jsonNumberField décode un champ numérique (JSON number = float64) d'un payload.
func jsonNumberField(raw json.RawMessage, key string) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return 0, false
	}
	if f, ok := m[key].(float64); ok {
		return int(f), true
	}
	return 0, false
}
