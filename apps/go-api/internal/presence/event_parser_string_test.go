package presence

import (
	"encoding/json"
	"testing"
	"time"
)

// =============================================================================
// parseTitleStateString — format string court "<state>:<titleId>" émis par
// TitlePresenceChangeSubscription (format réel documenté, jamais testé).
// =============================================================================

// Exercé via ParsePresencePayload pour couvrir aussi la branche string-payload
// (raw[0] == '"') qui n'avait aucun test.
func TestParsePresencePayload_StringPayload(t *testing.T) {
	tests := []struct {
		name          string
		raw           string // payload JSON brut (string entre guillemets)
		fallbackXUID  string
		wantState     string
		wantTitleID   string
		wantDetailNil bool
	}{
		{
			name:         "Started avec titleID",
			raw:          `"Started:1144039928"`,
			fallbackXUID: "xuid-A",
			wantState:    "Started",
			wantTitleID:  "1144039928",
		},
		{
			name:         "Ended avec titleID",
			raw:          `"Ended:1144039928"`,
			fallbackXUID: "xuid-B",
			wantState:    "Ended",
			wantTitleID:  "1144039928",
		},
		{
			// titleID vide (pas de ':') → pas de PresenceDetail, state = chaîne entière.
			name:          "state seul sans separateur",
			raw:           `"Ended"`,
			fallbackXUID:  "xuid-C",
			wantState:     "Ended",
			wantDetailNil: true,
		},
		{
			// separateur present mais titleID vide après ':' → pas de PresenceDetail.
			name:          "separateur sans titleID",
			raw:           `"Started:"`,
			fallbackXUID:  "xuid-D",
			wantState:     "Started",
			wantDetailNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParsePresencePayload(json.RawMessage(tc.raw), tc.fallbackXUID)
			if err != nil {
				t.Fatalf("ParsePresencePayload() error = %v", err)
			}
			// XUID provient toujours du fallback : le format string ne le porte pas.
			if event.XUID != tc.fallbackXUID {
				t.Errorf("XUID = %q, want fallback %q", event.XUID, tc.fallbackXUID)
			}
			if event.PresenceState != tc.wantState {
				t.Errorf("PresenceState = %q, want %q", event.PresenceState, tc.wantState)
			}
			if tc.wantDetailNil {
				if event.PresenceDetail != nil {
					t.Errorf("PresenceDetail = %+v, want nil (titleID vide)", event.PresenceDetail)
				}
				return
			}
			if event.PresenceDetail == nil {
				t.Fatal("PresenceDetail nil — devrait être renseigné quand titleID présent")
			}
			if event.PresenceDetail.TitleID != tc.wantTitleID {
				t.Errorf("TitleID = %q, want %q", event.PresenceDetail.TitleID, tc.wantTitleID)
			}
			// Invariants du format court : IsGame=true (topic scopé sur un titleId
			// déjà souscrit) et State recopie le state du payload.
			if !event.PresenceDetail.IsGame {
				t.Error("IsGame should be true pour le format string court")
			}
			if event.PresenceDetail.State != tc.wantState {
				t.Errorf("PresenceDetail.State = %q, want %q", event.PresenceDetail.State, tc.wantState)
			}
		})
	}
}

// Branche d'erreur du json.Unmarshal sur le payload string : un raw qui commence
// par '"' mais n'est pas une string JSON valide doit remonter une erreur (pas
// silencieusement parser une string vide).
func TestParsePresencePayload_StringPayload_InvalidJSON(t *testing.T) {
	// Commence par '"' donc emprunte la branche string, mais guillemet non fermé.
	_, err := ParsePresencePayload(json.RawMessage(`"Started:123`), "x")
	if err == nil {
		t.Fatal("expected error for malformed string payload")
	}
}

// =============================================================================
// parseXboxTimestamp — layouts au-delà du premier + branche d'erreur.
// =============================================================================

func TestParseXboxTimestamp_Layouts(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		// validation légère : année attendue quand parse OK.
		wantYear int
	}{
		{
			// Layout 1 : fractions de seconde variables, sans timezone (assumé UTC).
			name:     "fraction sans timezone",
			input:    "2026-05-25T20:00:36.8996648",
			wantYear: 2026,
		},
		{
			// Layout 2 : pas de fraction, pas de Z (assumé UTC) — jamais couvert.
			name:     "sans fraction sans Z",
			input:    "2026-05-25T20:00:36",
			wantYear: 2026,
		},
		{
			// Layout 3 : RFC3339 strict avec Z.
			name:     "RFC3339 avec Z",
			input:    "2026-04-13T21:10:46Z",
			wantYear: 2026,
		},
		{
			// Layout 3/offset : RFC3339 avec décalage de fuseau → normalisé en UTC.
			name:     "RFC3339 avec offset",
			input:    "2026-04-13T23:10:46+02:00",
			wantYear: 2026,
		},
		{
			// Layout 4 : RFC3339Nano (fraction + Z).
			name:     "RFC3339Nano",
			input:    "2026-04-13T21:10:46.123456789Z",
			wantYear: 2026,
		},
		{
			// Branche d'erreur : aucun layout ne matche.
			name:    "chaine invalide",
			input:   "pas-une-date",
			wantErr: true,
		},
		{
			name:    "vide",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, err := parseXboxTimestamp(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %v", tc.input, ts)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseXboxTimestamp(%q) error = %v", tc.input, err)
			}
			if ts.Year() != tc.wantYear {
				t.Errorf("Year = %d, want %d", ts.Year(), tc.wantYear)
			}
			// Invariant : la valeur retournée est toujours normalisée en UTC.
			if ts.Location() != time.UTC {
				t.Errorf("Location = %v, want UTC", ts.Location())
			}
		})
	}
}

// Invariant de cohérence du fuseau : un input avec offset +02:00 doit donner
// la même instant que son équivalent en Z (-2h sur l'heure murale).
func TestParseXboxTimestamp_OffsetNormalizedToUTC(t *testing.T) {
	withOffset, err := parseXboxTimestamp("2026-04-13T23:10:46+02:00")
	if err != nil {
		t.Fatalf("parse offset error = %v", err)
	}
	asUTC, err := parseXboxTimestamp("2026-04-13T21:10:46Z")
	if err != nil {
		t.Fatalf("parse Z error = %v", err)
	}
	if !withOffset.Equal(asUTC) {
		t.Errorf("offset %v != UTC %v (même instant attendu)", withOffset, asUTC)
	}
}

// =============================================================================
// ParsePresencePayload — fallback "premier item game si pas de primary" pour le
// format devices[] (jamais exercé : aucun titre Active mais titres présents).
// =============================================================================

// Format devices[] sans aucun titre "Active" → fallback sur le premier titre
// quel que soit son state (branche lignes 194-211 non couverte).
func TestParsePresencePayload_DevicesFormat_NoActiveFallback(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","state":"Online","devices":[{"type":"Win32","titles":[{"id":"2043073184","name":"Halo Infinite","placement":"Background","state":"Inactive"}]}]}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.PresenceDetail == nil {
		t.Fatal("PresenceDetail nil — fallback devrait prendre le premier titre même Inactive")
	}
	if event.PresenceDetail.TitleID != "2043073184" {
		t.Errorf("TitleID = %q, want 2043073184", event.PresenceDetail.TitleID)
	}
	if event.PresenceDetail.State != "Inactive" {
		t.Errorf("State = %q, want Inactive", event.PresenceDetail.State)
	}
	// placement != Full → IsPrimary doit être false.
	if event.PresenceDetail.IsPrimary {
		t.Error("IsPrimary should be false pour placement Background")
	}
}

// Bloc lastSeen avec timestamp non-parsable : event.LastSeen doit rester nil
// (le parseur ne crée pas un LastSeen avec timestamp zéro). Couvre la branche
// where parseXboxTimestamp renvoie err à l'intérieur de ParsePresencePayload.
func TestParsePresencePayload_LastSeenBadTimestamp(t *testing.T) {
	raw := json.RawMessage(`{"xuid":"X","state":"Offline","lastSeen":{"titleName":"Halo Infinite","titleId":"2043073184","timestamp":"not-a-date"}}`)
	event, err := ParsePresencePayload(raw, "")
	if err != nil {
		t.Fatalf("ParsePresencePayload() error = %v", err)
	}
	if event.LastSeen != nil {
		t.Errorf("LastSeen = %+v, want nil (timestamp non parsable)", event.LastSeen)
	}
}
