package halo_5

// events.go — mapping PUR de la timeline d'events Halo 5 (/h5/matches/{id}/events)
// vers canonical.MatchEventTimeline. Le client (GetMatchEvents) vit dans client.go ;
// le wiring adapter (LoadMatchEvents) dans adapter_data.go. Ici : fonctions PURES,
// testables sur fixture JSON réelle sans réseau.
//
// NB temps : la TimeSinceStart Halo 5 est déjà relative au DÉBUT DU MATCH (pas de
// countdown à retrancher comme Infinite) → TimeMs ≥ 0, aucune correction T0.

import (
	"strconv"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/halo/duration"
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
			ev.Assists = h5Assistants(e.Assistants)
		case canonical.MatchEventMedal:
			ev.Player = h5EventIdentity(e.Player)
			ev.RefID = h5IDString(e.MedalId)
		case canonical.MatchEventImpulse:
			ev.Player = h5EventIdentity(e.Player)
			ev.RefID = h5IDString(e.ImpulseId)
		case canonical.MatchEventWeaponPickup, canonical.MatchEventWeaponDrop:
			ev.Player = h5EventIdentity(e.Player)
			ev.Weapon = h5WeaponRef(e.WeaponStockId)
			// WeaponDrop porte les tirs de l'arme lâchée (précision par arme).
			// WeaponPickup ne les porte pas (toujours 0) → on ne les pose que sur drop.
			if et == canonical.MatchEventWeaponDrop {
				sf, sl := e.ShotsFired, e.ShotsLanded
				ev.ShotsFired = &sf
				ev.ShotsLanded = &sl
			}
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

// h5Assistants convertit la liste d'assistants Halo 5 en []PlayerIdentity
// (gamertag-keyé). Les entrées sans gamertag sont ignorées. nil si vide.
func h5Assistants(list []h5EventPlayer) []canonical.PlayerIdentity {
	if len(list) == 0 {
		return nil
	}
	out := make([]canonical.PlayerIdentity, 0, len(list))
	for _, p := range list {
		if strings.TrimSpace(p.Gamertag) == "" {
			continue
		}
		out = append(out, canonical.PlayerIdentity{Gamertag: p.Gamertag})
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
// parseISO8601DurationSeconds qui arrondit à la seconde). Borne [0, 24h]
// (anti-corruption). ok=false si non parsable. Délègue à la source UNIQUE
// (internal/platform/halo/duration).
func parseISO8601DurationMs(s string) (int, bool) {
	return duration.MillisBounded(s)
}
