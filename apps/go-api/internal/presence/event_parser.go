// Package presence — event_parser.go : parsing des payloads de présence Xbox RTA.
//
// Payload RTA attendu :
//
//	{
//	  "xuid": "1234567890123456",
//	  "presenceState": "Online",
//	  "presenceDetails": [{
//	    "titleid": "1144039928",
//	    "titleName": "Halo Infinite",
//	    "isGame": true,
//	    "isPrimary": true,
//	    "device": "PC",
//	    "state": "Active"
//	  }]
//	}
package presence

import (
	"encoding/json"
	"fmt"
)

// rtaPresencePayload est la structure JSON brute du payload RTA.
type rtaPresencePayload struct {
	XUID            string            `json:"xuid"`
	PresenceState   string            `json:"presenceState"`
	PresenceDetails []rtaPresenceItem `json:"presenceDetails"`
}

type rtaPresenceItem struct {
	TitleID   string `json:"titleid"`
	TitleName string `json:"titleName"`
	IsGame    bool   `json:"isGame"`
	IsPrimary bool   `json:"isPrimary"`
	Device    string `json:"device"`
	State     string `json:"state"`
}

// ParsePresencePayload parse un payload RTA en PresenceEvent.
// xuid est passé en paramètre pour fallback si absent du payload.
func ParsePresencePayload(raw json.RawMessage, fallbackXUID string) (PresenceEvent, error) {
	var p rtaPresencePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return PresenceEvent{}, fmt.Errorf("parse presence: %w", err)
	}

	xuid := p.XUID
	if xuid == "" {
		xuid = fallbackXUID
	}

	event := PresenceEvent{
		XUID:          xuid,
		PresenceState: p.PresenceState,
	}

	// Trouver le premier titre actif (isPrimary && isGame)
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
	// Fallback : premier item game si pas de primary
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

	return event, nil
}
