// Package objectiveevents — extract.go : orchestration de l'extraction des
// events objectif d'un match vers []domain.ObjectiveEvent.
//
// Frontière PURE/IO : Extract() prend un `*filmsource.Film` DÉJÀ CHARGÉ (chunks décompressés,
// paquets découpés, métadonnées du manifeste portées par le film) et un Roster (xuid->team_id,
// résolu en amont depuis match_participants), et ne fait AUCUN accès DB ni FS lui-même —
// l'appelant charge le film une fois pour toute sa chaîne (`filmcache.LoadFilm`,
// `filmsource.LoadDir`). Le dispatch de mode se fait sur match_registry.game_variant_name
// (cf. PLAN §10).
package objectiveevents

import (
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/domain"
)

// Valeurs de domain.ObjectiveEvent.ObjectiveType (parent mode-agnostique).
const (
	ObjectiveTypeFlag  = "flag"  // CTF
	ObjectiveTypeZone  = "zone"  // Strongholds / Land Grab / Total Control
	ObjectiveTypeHill  = "hill"  // King of the Hill
	ObjectiveTypeSkull = "skull" // Oddball
	ObjectiveTypeVip   = "vip"   // VIP (statborg-only : le film ne porte pas le bit VIP)
	ObjectiveTypeBomb  = "bomb"  // Assaut (One Bomb / Neutral Bomb) — statborg-only
)

// Valeurs de domain.ObjectiveEvent.EventType (action).
const (
	EventTypeCapture     = "capture"      // CTF : drapeau capturé (burst tiers==6)
	EventTypeZoneCapture = "zone_capture" // Strongholds : interaction de zone
	EventTypeHillCapture = "hill_capture" // KOTH : interaction de colline
	EventTypeSkullCarry  = "skull_carry"  // Oddball : heartbeat de possession
)

// Valeurs de domain.ObjectiveEvent.Source (provenance du décodage).
const (
	SourceBurst = "burst" // CTF : FRAME re-transmettant la table 6-tiers
	SourceTh10  = "th10"  // event footer type_hint==10
)

// Valeurs de domain.ObjectiveEvent.Confidence (précision temporelle).
const (
	ConfidenceExact  = "exact"  // ms-précis (CTF burst)
	ConfidenceApprox = "approx" // ~5-20s (heartbeat / inflexion th10)
)

// Rôles de domain.ObjectiveEventPlayer.Role.
const (
	RoleScorer      = "scorer"      // l'acteur de la capture (max-t du cluster)
	RoleContributor = "contributor" // co-participant à l'interaction objectif
)

// captureClusterWindowMS = fenêtre de coïncidence entre un burst de capture CTF
// (ms via FRAME) et les events th=10 du footer : la capture reset les drapeaux
// -> cluster d'events th=10 simultanés. L'équipe = l'event de t MAX du cluster
// (ancre validée 53ce4390 : burst 656554 <-> th10 656558, ~4ms). On élargit à
// 2s pour absorber le décalage horloge FRAME/footer entre deux ancres.
const captureClusterWindowMS = 2000

// Roster résout xuid -> team_id (depuis match_participants). team_id canonique :
// le champ team du film étant non fiable (RESEARCH_THEATER_RE.md §M), l'équipe
// d'un event vient TOUJOURS du roster via le xuid de l'acteur.
type Roster interface {
	// TeamOf renvoie (team_id, true) si le xuid est un participant connu.
	TeamOf(xuid string) (int, bool)
}

// MapRoster est un Roster simple basé sur une map (pratique pour le CLI/test).
type MapRoster map[string]int

// TeamOf implémente Roster.
func (m MapRoster) TeamOf(xuid string) (int, bool) {
	t, ok := m[xuid]
	return t, ok
}

// Extract décode les events objectif d'un match. game_variant_name pilote le
// mode (CTF -> bursts ; Strongholds/KOTH/Oddball -> events th=10 du footer). Les
// modes non-objectif (Slayer, etc.) renvoient nil (no-op, pas d'erreur). team_id
// canonique via roster ; objective_id toujours NULL (zone/colline non récupérable).
//
// Renvoie les events ordonnés par time_ms avec un Seq dense 0..N-1.
func Extract(matchID, gameVariantName string, film *filmsource.Film, roster Roster) []domain.ObjectiveEvent {
	switch classifyObjectiveMode(gameVariantName) {
	case ObjectiveTypeFlag:
		return finalize(matchID, extractCTF(matchID, film, roster))
	case ObjectiveTypeZone:
		return finalize(matchID, extractFromTh10(matchID, film, roster, ObjectiveTypeZone, EventTypeZoneCapture))
	case ObjectiveTypeHill:
		return finalize(matchID, extractFromTh10(matchID, film, roster, ObjectiveTypeHill, EventTypeHillCapture))
	case ObjectiveTypeSkull:
		return finalize(matchID, extractFromTh10(matchID, film, roster, ObjectiveTypeSkull, EventTypeSkullCarry))
	default:
		return nil
	}
}

// ObjectiveTypeOf classe un game_variant_name vers une famille d'objectif
// exportée ("flag"|"zone"|"hill"|"skull") ou "" pour un mode non-objectif.
// Wrapper exporté de classifyObjectiveMode : permet au package sync de classer
// un match sans dupliquer le keyword-scan.
func ObjectiveTypeOf(gameVariantName string) string {
	return classifyObjectiveMode(gameVariantName)
}

// classifyObjectiveMode mappe un game_variant_name vers une famille d'objectif,
// ou "" pour un mode non-objectif. Robuste aux variations de formatage du nom
// ("CTF:Arena", "Arena:CTF", "Ranked:CTF", "Husky Raid:CTF", "Strongholds:Arena",
// "Arena:King of the Hill", "KOTH:Arena", "Oddball:Arena", "Ranked:Oddball"…) :
// scan par mots-clés sur le nom complet en minuscules.
func classifyObjectiveMode(gameVariantName string) string {
	n := strings.ToLower(gameVariantName)
	switch {
	case strings.Contains(n, "ctf") || strings.Contains(n, "flag"):
		return ObjectiveTypeFlag
	case strings.Contains(n, "stronghold") || strings.Contains(n, "land grab") ||
		strings.Contains(n, "total control"):
		return ObjectiveTypeZone
	case strings.Contains(n, "king of the hill") || strings.Contains(n, "koth"):
		return ObjectiveTypeHill
	case strings.Contains(n, "oddball"):
		return ObjectiveTypeSkull
	case strings.Contains(n, "assault") || strings.Contains(n, "bomb"):
		// Assaut : « Assault:One Bomb », « Assault:Neutral Bomb », « Assault:Neutral Bomb
		// Squad », « Husky Raid:Assault » — les 4 formes du registre. Le mot-clef `bomb`
		// double `assault` parce que le second manque a trois d'entre elles ; aucune variante
		// Halo non-Assaut ne porte l'un ou l'autre (releve `match_registry` du 2026-08-31).
		//
		// PLACE DANS LE SWITCH : apres le drapeau, qui gagne — une hypothetique variante
		// nommant les deux resterait un CTF, mode dont les emplacements sont mesures.
		return ObjectiveTypeBomb
	default:
		return ""
	}
}

// footerData renvoie le contenu DÉCOMPRESSÉ du footer (chunk de plus haut index,
// chunk_type 3), ou (nil,false). Le footer porte les events th=10. Si le footer
// n'est pas en cache, l'équipe par-event manque -> dégradation gracieuse : le film chargé ne
// porte QUE les chunks réellement présents, donc un pied manquant au cache n'a pas d'entrée.
func footerData(film *filmsource.Film) ([]byte, bool) {
	footerPos, footerIdx := -1, -1
	for _, c := range manifestChunks(film) {
		if c.meta.ChunkType == chunkTypePied && c.meta.Index > footerIdx {
			footerPos, footerIdx = c.pos, c.meta.Index
		}
	}
	if footerPos < 0 {
		return nil, false
	}
	return film.Chunk(footerPos), true
}

// extractCTF décode les captures CTF : pour chaque burst (tiers==6, ms via FRAME
// sur les chunks gameplay), l'équipe = l'event th=10 de t MAX dans le cluster
// coïncident du footer, mappé via roster. players=[{scorer xuid}].
func extractCTF(matchID string, film *filmsource.Film, roster Roster) []domain.ObjectiveEvent {
	bursts := collectCaptureBursts(film)
	footer, hasFooter := footerData(film)
	var th10 []th10Event
	if hasFooter {
		th10 = scanTh10Events(footer)
	}
	// Capacité EXACTE : un événement par burst, sans continue dans la boucle. Le nil
	// éventuel n'est pas perdu — finalize() ramène une tranche vide à nil.
	out := make([]domain.ObjectiveEvent, 0, len(bursts))
	for _, b := range bursts {
		ev := domain.ObjectiveEvent{
			MatchID:       matchID,
			TimeMS:        intPtr(b.matchMS),
			ObjectiveType: ObjectiveTypeFlag,
			EventType:     EventTypeCapture,
			Value:         intPtr(1), // +1 capture
			Source:        SourceBurst,
			Confidence:    ConfidenceExact,
			Details:       "{}",
		}
		if scorer, ok := captureScorer(th10, b.matchMS); ok {
			xuid := formatXUID(scorer.xuid)
			ev.Players = []domain.ObjectiveEventPlayer{{XUID: xuid, Role: RoleScorer}}
			if team, ok := roster.TeamOf(xuid); ok {
				ev.TeamID = intPtr(team)
			}
		}
		out = append(out, ev)
	}
	return out
}

// collectCaptureBursts parcourt tous les chunks gameplay (type 2) et concatène
// leurs bursts de capture, ordonnés par ms.
func collectCaptureBursts(film *filmsource.Film) []captureBurst {
	var out []captureBurst
	for _, c := range manifestChunks(film) {
		if c.meta.ChunkType != chunkTypeJeu {
			continue
		}
		out = append(out, scanCaptureBursts(framesOf(film, c.pos), c.meta.StartMS)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].matchMS < out[j].matchMS })
	return out
}

// captureScorer renvoie l'event th=10 de t MAX dans la fenêtre de coïncidence du
// burst (la capture reset les drapeaux -> cluster ; le dernier event = l'acteur
// de la capture). ok=false si aucun event coïncident (footer absent/partiel).
func captureScorer(th10 []th10Event, burstMS int) (th10Event, bool) {
	best := th10Event{t: -1}
	found := false
	for _, e := range th10 {
		if abs(e.t-burstMS) > captureClusterWindowMS {
			continue
		}
		if !found || e.t > best.t {
			best = e
			found = true
		}
	}
	return best, found
}

// extractFromTh10 décode Strongholds/KOTH/Oddball depuis les events th=10 du
// footer : un objective-event par event th=10 (zone_capture/hill_capture/
// skull_carry), team via roster (xuid de l'acteur), source=th10, confidence=
// approx (~5-20s). objective_id NULL. value laissée nil (score per-event non
// décodé ici ; le score-over-time est une couche séparée). Footer absent -> nil.
func extractFromTh10(
	matchID string, film *filmsource.Film, roster Roster, objType, evType string,
) []domain.ObjectiveEvent {
	footer, ok := footerData(film)
	if !ok {
		return nil
	}
	var out []domain.ObjectiveEvent
	for _, e := range scanTh10Events(footer) {
		xuid := formatXUID(e.xuid)
		ev := domain.ObjectiveEvent{
			MatchID:       matchID,
			TimeMS:        intPtr(e.t),
			ObjectiveType: objType,
			EventType:     evType,
			Source:        SourceTh10,
			Confidence:    ConfidenceApprox,
			Details:       "{}",
			Players:       []domain.ObjectiveEventPlayer{{XUID: xuid, Role: RoleScorer}},
		}
		if team, ok := roster.TeamOf(xuid); ok {
			ev.TeamID = intPtr(team)
		}
		out = append(out, ev)
	}
	return out
}

// finalize ordonne les events par time_ms (nil en tête) et assigne un Seq dense
// 0..N-1. Renvoie nil pour une liste vide (no-op côté repo WriteMatch).
func finalize(matchID string, events []domain.ObjectiveEvent) []domain.ObjectiveEvent {
	if len(events) == 0 {
		return nil
	}
	sort.SliceStable(events, func(i, j int) bool {
		return timeOrNeg(events[i].TimeMS) < timeOrNeg(events[j].TimeMS)
	})
	for i := range events {
		events[i].Seq = i
		events[i].MatchID = matchID
	}
	return events
}

// CaptureBurstTimes rend les instants (ms, horloge du match) des BURSTS DE CAPTURE de
// drapeau du film — le seul signal de mode CTF qui vive DANS le film.
//
// POURQUOI IL EST EXPORTE (2026-08-18, plan objectifs vivants, item 1.1). L'artefact de rejeu
// 2D est construit HORS LIGNE, a partir des seuls chunks : il ne connait ni la carte ni le
// `game_variant_name`, donc il ne peut pas demander a [ObjectiveTypeOf] de quel mode il s'agit.
// Or publier le portage du drapeau exige de savoir qu'on est en CTF : la table d'emplacements
// de statistiques du drapeau lue sur un film d'un AUTRE mode rendrait des « prises » qui n'en
// sont pas. Le burst repond a la question sans base : c'est l'evenement de score qui accompagne
// une capture de drapeau, detecte a 6 tiers distincts — 0 manque et 0 faux positif sur les
// matchs de verite terrain (cf. scanCaptureBursts).
//
// CE QU'IL NE DIT PAS : une partie CTF ou personne ne capture n'en produit aucun. Le rejeu
// publie alors un calque de drapeau VIDE, et sa couverture le dit.
func CaptureBurstTimes(film *filmsource.Film) []int {
	bursts := collectCaptureBursts(film)
	out := make([]int, 0, len(bursts))
	for _, b := range bursts {
		out = append(out, b.matchMS)
	}
	return out
}
