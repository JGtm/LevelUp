package halo_5

// events.go — mapping PUR de la timeline d'events Halo 5 (/h5/matches/{id}/events)
// vers canonical.MatchEventTimeline. Le client (GetMatchEvents) vit dans client.go ;
// le wiring adapter (LoadMatchEvents) dans adapter_data.go. Ici : fonctions PURES,
// testables sur fixture JSON réelle sans réseau.
//
// NB temps : la TimeSinceStart Halo 5 est déjà relative au DÉBUT DU MATCH (pas de
// countdown à retrancher comme Infinite) → TimeMs ≥ 0, aucune correction T0.

import (
	"math"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// h5EventNameToType mappe l'EventName Halo 5 → type canonique. ok=false pour un
// EventName inconnu (ignoré par le mapper, robuste à un nouveau type d'event).
func h5EventNameToType(name string) (canonical.MatchEventType, bool) {
	switch name {
	case "Death":
		return canonical.MatchEventKill, true
	case "Medal":
		return canonical.MatchEventMedal, true
	case "Impulse":
		return canonical.MatchEventImpulse, true
	case "WeaponPickup":
		return canonical.MatchEventWeaponPickup, true
	case "WeaponDrop":
		return canonical.MatchEventWeaponDrop, true
	case "PlayerSpawn":
		return canonical.MatchEventSpawn, true
	case "RoundStart":
		return canonical.MatchEventRoundStart, true
	case "RoundEnd":
		return canonical.MatchEventRoundEnd, true
	}
	return "", false
}

// mapH5Events convertit la réponse /events en []canonical.MatchEvent, filtrée par
// opts.Types (Types vide = tout). Les events sans timestamp parsable sont ignorés.
func mapH5Events(resp *h5MatchEventsResponse, opts canonical.MatchEventOptions) []canonical.MatchEvent {
	if resp == nil {
		return nil
	}
	out := make([]canonical.MatchEvent, 0, len(resp.GameEvents))
	for i := range resp.GameEvents {
		e := &resp.GameEvents[i]
		et, ok := h5EventNameToType(e.EventName)
		if !ok || !opts.Wants(et) {
			continue
		}
		ms, ok := parseISO8601DurationMs(e.TimeSinceStart)
		if !ok {
			continue // sans instant, l'event est inexploitable dans une timeline
		}
		ev := canonical.MatchEvent{Type: et, TimeMs: ms}
		switch et {
		case canonical.MatchEventKill:
			ev.Killer = h5EventIdentity(e.Killer)
			ev.Victim = h5EventIdentity(e.Victim)
			ev.Kind = h5KillKind(e)
			ev.Headshot = e.IsHeadshot
			ev.Weapon = h5WeaponRef(e.KillerWeaponStockId)
			ev.KillerLoc = h5Vec3(e.KillerWorldLocation)
			ev.VictimLoc = h5Vec3(e.VictimWorldLocation)
		case canonical.MatchEventMedal:
			ev.Player = h5EventIdentity(e.Player)
			ev.RefID = h5IDString(e.MedalId)
		case canonical.MatchEventImpulse:
			ev.Player = h5EventIdentity(e.Player)
			ev.RefID = h5IDString(e.ImpulseId)
		case canonical.MatchEventWeaponPickup, canonical.MatchEventWeaponDrop:
			ev.Player = h5EventIdentity(e.Player)
			ev.Weapon = h5WeaponRef(e.WeaponStockId)
		case canonical.MatchEventSpawn:
			ev.Player = h5EventIdentity(e.Player)
		case canonical.MatchEventRoundStart, canonical.MatchEventRoundEnd:
			if e.RoundIndex != nil {
				r := *e.RoundIndex
				ev.Round = &r
			}
		}
		out = append(out, ev)
	}
	return out
}

// h5EventIdentity construit une PlayerIdentity gamertag-keyée (XUID vide en Halo 5).
// nil si l'acteur est absent (ex. kill d'environnement → Killer nil).
func h5EventIdentity(p *h5EventPlayer) *canonical.PlayerIdentity {
	if p == nil || strings.TrimSpace(p.Gamertag) == "" {
		return nil
	}
	return &canonical.PlayerIdentity{Gamertag: p.Gamertag}
}

// h5KillKind dérive la mécanique du kill des drapeaux Halo 5 (priorité : melee >
// groundpound > shoulderbash > weapon). Le headshot est porté séparément (Headshot).
func h5KillKind(e *h5GameEvent) canonical.KillKind {
	switch {
	case e.IsMelee:
		return canonical.KillKindMelee
	case e.IsGroundPound:
		return canonical.KillKindGroundPound
	case e.IsShoulderBash:
		return canonical.KillKindShoulderBash
	default:
		return canonical.KillKindWeapon
	}
}

// h5WeaponRef construit une AssetReference d'arme depuis un StockId. nil si 0
// (pas d'arme : ex. VictimStockId=0). Le label est résolu en aval (semantic adapter).
func h5WeaponRef(stockID int64) *canonical.AssetReference {
	if stockID == 0 {
		return nil
	}
	return &canonical.AssetReference{Kind: "weapon", ID: strconv.FormatInt(stockID, 10)}
}

// h5IDString convertit un identifiant catalogue (medal/impulse) en *string. nil si 0.
func h5IDString(id int64) *string {
	if id == 0 {
		return nil
	}
	s := strconv.FormatInt(id, 10)
	return &s
}

// h5Vec3 convertit une position monde Halo 5 en canonical.Vec3. nil si absente.
func h5Vec3(l *h5WorldLocation) *canonical.Vec3 {
	if l == nil {
		return nil
	}
	return &canonical.Vec3{X: l.X, Y: l.Y, Z: l.Z}
}

// parseISO8601DurationMs parse une durée ISO8601 (ex. "PT33.2154416S", "PT5M41.79S")
// en MILLISECONDES, en conservant la précision fractionnaire (vs
// parseISO8601DurationSeconds qui arrondit à la seconde). Réutilise la regex du
// package. Borne [0, 24h] (anti-corruption). ok=false si non parsable.
func parseISO8601DurationMs(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	m := iso8601DurationRe.FindStringSubmatch(s)
	if m == nil || (m[1] == "" && m[2] == "" && m[3] == "" && m[4] == "") {
		return 0, false
	}
	var total float64
	if m[1] != "" {
		if d, err := strconv.Atoi(m[1]); err == nil {
			total += float64(d) * 86400
		}
	}
	if m[2] != "" {
		if h, err := strconv.Atoi(m[2]); err == nil {
			total += float64(h) * 3600
		}
	}
	if m[3] != "" {
		if mn, err := strconv.Atoi(m[3]); err == nil {
			total += float64(mn) * 60
		}
	}
	if m[4] != "" {
		if sec, err := strconv.ParseFloat(m[4], 64); err == nil {
			total += sec
		}
	}
	if total < 0 || total > h5MaxDurationSeconds {
		return 0, false
	}
	return int(math.Round(total * 1000)), true
}
