// Package presence — event_parser.go : parsing des payloads de présence Xbox RTA.
//
// Trois formats de payloads sont supportés par ParsePresencePayload :
//
//  1. Format XSAPI XblPresenceRecord (initial subscribe pour topics anciens) :
//     {
//     "xuid":"...","presenceState":"Online",
//     "presenceDetails":[{"titleid":"...","titleName":"...","isGame":true,
//     "isPrimary":true,"device":"PC","state":"Active"}]
//     }
//
//  2. Format string court pour events TitlePresenceChangeSubscription :
//     "Started:1144039928" / "Ended:1144039928"
//
//  3. Format /users/xuid(N)/titles/<TID> + nonce (observé prod 2026-05-25) :
//     {
//     "xuid":"...","state":"Online",
//     "devices":[{"type":"WindowsOneCore","titles":[{
//     "id":"2043073184","name":"Halo Infinite","placement":"Full",
//     "state":"Active","lastModified":"..."
//     }]}]
//     }
//     OU snapshot Offline :
//     {"xuid":"...","state":"Offline","lastSeen":{"deviceType":"...",
//     "titleId":"...","titleName":"...","timestamp":"..."}}
package presence

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// rtaPresencePayload est la structure JSON brute du payload RTA.
// Supporte les 3 formats : presenceState/presenceDetails (XSAPI), state/devices
// (topic /titles/<TID> + nonce), et lastSeen (snapshot Offline).
type rtaPresencePayload struct {
	XUID            string             `json:"xuid"`
	PresenceState   string             `json:"presenceState"` // format XSAPI
	State           string             `json:"state"`         // format /titles/<TID> + nonce
	PresenceDetails []rtaPresenceItem  `json:"presenceDetails"`
	Devices         []rtaPresenceDev   `json:"devices"`
	LastSeen        *rtaPresenceLastSe `json:"lastSeen"`
}

type rtaPresenceItem struct {
	TitleID   string `json:"titleid"`
	TitleName string `json:"titleName"`
	IsGame    bool   `json:"isGame"`
	IsPrimary bool   `json:"isPrimary"`
	Device    string `json:"device"`
	State     string `json:"state"`
}

// rtaPresenceDev correspond à un élément de `devices[]` dans le format
// /titles/<TID> + nonce.
type rtaPresenceDev struct {
	Type   string                `json:"type"` // ex. "WindowsOneCore", "Win32"
	Titles []rtaPresenceDevTitle `json:"titles"`
}

type rtaPresenceDevTitle struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Placement    string `json:"placement"` // "Full", "Background", "Fill", "Snapped"
	State        string `json:"state"`     // "Active", "Inactive"
	LastModified string `json:"lastModified"`
}

// rtaPresenceLastSe correspond au champ `lastSeen` quand state=Offline.
type rtaPresenceLastSe struct {
	DeviceType string `json:"deviceType"`
	TitleID    string `json:"titleId"`
	TitleName  string `json:"titleName"`
	Timestamp  string `json:"timestamp"`
}

// ParsePresencePayload parse un payload RTA en PresenceEvent.
// xuid est passé en paramètre pour fallback si absent du payload.
//
// Deux formats sont supportés (source : XSAPI Microsoft
// title_presence_change_subscription.cpp) :
//
//  1. Réponse initiale au subscribe : objet JSON XblPresenceRecord complet
//     (xuid, presenceState, presenceDetails[]).
//  2. Event push ultérieur sur /titles/<TID> : simple string JSON
//     "<state>:<titleId>" (ex. "Started:1144039928", "Ended:1144039928").
func ParsePresencePayload(raw json.RawMessage, fallbackXUID string) (PresenceEvent, error) {
	// Cas 2 : payload string "state:titleId"
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return PresenceEvent{}, fmt.Errorf("parse presence string: %w", err)
		}
		return parseTitleStateString(s, fallbackXUID), nil
	}

	// Cas 1 : payload objet XblPresenceRecord
	var p rtaPresencePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return PresenceEvent{}, fmt.Errorf("parse presence: %w", err)
	}

	xuid := p.XUID
	if xuid == "" {
		xuid = fallbackXUID
	}

	// Le champ state racine peut être soit `presenceState` (XSAPI) soit `state`
	// (/titles/<TID> + nonce). On prend le premier non-vide.
	presenceState := p.PresenceState
	if presenceState == "" {
		presenceState = p.State
	}

	event := PresenceEvent{
		XUID:          xuid,
		PresenceState: presenceState,
	}

	// Bloc lastSeen (format /titles/<TID> + nonce, snapshot Offline) :
	// extrait pour afficher "vu il y a 2h sur Halo Infinite" côté UI.
	if p.LastSeen != nil && p.LastSeen.Timestamp != "" {
		// Xbox renvoie le timestamp en ISO 8601 sans timezone explicite,
		// mais c'est en réalité de l'UTC (vérifié sur snapshots prod).
		// On parse en supposant UTC ; si Xbox change un jour, on adaptera.
		ts, err := parseXboxTimestamp(p.LastSeen.Timestamp)
		if err == nil {
			event.LastSeen = &LastSeenInfo{
				Timestamp:  ts,
				TitleID:    p.LastSeen.TitleID,
				TitleName:  p.LastSeen.TitleName,
				DeviceType: p.LastSeen.DeviceType,
			}
		}
	}

	// Format XSAPI : trouver le premier titre actif (isPrimary && isGame).
	for _, item := range p.PresenceDetails {
		if item.IsGame && item.IsPrimary {
			event.PresenceDetail = &PresenceDetail{
				TitleID:   item.TitleID,
				TitleName: item.TitleName,
				IsGame:    item.IsGame,
				IsPrimary: item.IsPrimary,
				Device:    item.Device,
				State:     item.State,
			}
			break
		}
	}
	// Fallback : premier item game si pas de primary.
	if event.PresenceDetail == nil {
		for _, item := range p.PresenceDetails {
			if item.IsGame {
				event.PresenceDetail = &PresenceDetail{
					TitleID:   item.TitleID,
					TitleName: item.TitleName,
					IsGame:    item.IsGame,
					IsPrimary: item.IsPrimary,
					Device:    item.Device,
					State:     item.State,
				}
				break
			}
		}
	}

	// Format /titles/<TID> + nonce : devices[].titles[]. Prendre le premier
	// titre Active ; à défaut le premier tout court. Pas de notion isGame/
	// isPrimary dans ce format — on infère IsGame=true (le topic est déjà
	// scopé sur un titleId qu'on a souscrit, donc c'est forcément un jeu).
	if event.PresenceDetail == nil {
		for _, dev := range p.Devices {
			for _, t := range dev.Titles {
				if t.State == "Active" {
					event.PresenceDetail = &PresenceDetail{
						TitleID:   t.ID,
						TitleName: t.Name,
						IsGame:    true,
						IsPrimary: t.Placement == "Full",
						Device:    dev.Type,
						State:     t.State,
					}
					break
				}
			}
			if event.PresenceDetail != nil {
				break
			}
		}
		// Fallback : premier titre quel que soit son state.
		if event.PresenceDetail == nil {
			for _, dev := range p.Devices {
				for _, t := range dev.Titles {
					event.PresenceDetail = &PresenceDetail{
						TitleID:   t.ID,
						TitleName: t.Name,
						IsGame:    true,
						IsPrimary: t.Placement == "Full",
						Device:    dev.Type,
						State:     t.State,
					}
					break
				}
				if event.PresenceDetail != nil {
					break
				}
			}
		}
	}

	return event, nil
}

// parseTitleStateString parse un payload event court "<state>:<titleId>"
// émis par les subscriptions TitlePresenceChangeSubscription (format XSAPI).
// Exemples : "Started:1144039928" → {State:"Started", TitleID:"1144039928"}.
func parseTitleStateString(s, fallbackXUID string) PresenceEvent {
	state, titleID, _ := strings.Cut(s, ":")
	event := PresenceEvent{
		XUID:          fallbackXUID,
		PresenceState: state,
	}
	if titleID != "" {
		event.PresenceDetail = &PresenceDetail{
			TitleID: titleID,
			IsGame:  true,
			State:   state,
		}
	}
	return event
}

// parseXboxTimestamp parse un timestamp Xbox au format ISO 8601 sans timezone
// (ex: "2026-05-25T20:00:36.8996648"). Interprété comme UTC.
//
// Tente plusieurs layouts pour gérer les variations Xbox :
//   - "2006-01-02T15:04:05.9999999" (fractions de seconde variables)
//   - "2006-01-02T15:04:05Z" (avec Z)
//   - RFC3339 strict
func parseXboxTimestamp(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02T15:04:05.9999999",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	// Si Xbox renvoie déjà un suffixe Z, le standard RFC3339 le gère.
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			// Si le layout n'a pas de timezone, on assume UTC.
			if t.Location() == time.UTC || layout == "2006-01-02T15:04:05.9999999" || layout == "2006-01-02T15:04:05" {
				return t.UTC(), nil
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse xbox timestamp %q: aucun layout ne matche", s)
}
