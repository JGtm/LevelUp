package replay

// unnamed_lives.go — AUCUNE VIE PUBLIÉE NE RESTE SANS NOM.
//
// # LA DÉCISION PRODUIT (utilisateur, 2026-09-07)
//
// « Les vies anonymes n'existent pas : une vie est un humain ou un bot, point. » Une piste
// publiée sans identité n'est donc PAS une catégorie de donnée — c'est un DÉFAUT DE NOMMAGE du
// pont d'identité. Elle ne se publie pas comme un état « inconnu », elle ne se dessine pas en
// encre neutre : elle se RÉPARE à la source, et ce qui résiste se COMPTE et s'ALARME.
//
// # CE QUI NOMMAIT DÉJÀ, ET DANS QUEL ORDRE
//
//	1. le fil des MORTS   `nameTracksByLives` — une vie est nommée par la mort qui la TERMINE
//	                      (owners.go / lives.go). C'est la lecture, elle prime sur tout.
//	2. les FERMETURES     A (le corps disponible) et B (la réapparition) nomment la vie
//	                      qu'elles ont DÉSIGNÉE (closures.go, `nameClosedLives`).
//	3. le SIÈGE de bot    `nameBotTracks` — les slots que le pont attribue à l'index d'un bot,
//	                      sièges ambigus exclus (cf. identity.go).
//	4. les RELAIS         `attributeSuccessions` — le remplaçant, daté par l'instant de bascule
//	                      lu dans la base (successions.go).
//
// # CE QUE CE FICHIER AJOUTE, ET POURQUOI C'EST LE TEMPS QUI TRANCHE
//
// Après ces quatre passes il reste des vies sans nom, et la population est CONNUE : une vie
// qu'aucune mort ne termine. Dans la vie d'un joueur c'est donc, typiquement, sa DERNIÈRE —
// celle qui court de sa dernière mort au coup de sifflet (le raisonnement est déjà écrit dans
// `bodyExtendsShooter`, closures.go) — plus les vies antérieures au début réel du match. Le
// film les publie : il montre un joueur qui bouge, et c'est SON identité qui manque, pas sa
// présence.
//
// La réponse est l'OCCUPATION DU SLOT DANS LE TEMPS, jamais une identité de slot prise en bloc :
//
//	a. la vie nommée du MÊME slot la plus proche AVANT elle — un slot est réattribué à un
//	   instant, donc son occupant juste avant est le meilleur candidat, et c'est le TEMPS qui
//	   choisit, pas l'ordre d'itération ;
//	b. à défaut, la vie nommée du MÊME slot la plus proche APRÈS elle (cas d'une vie antérieure
//	   à la première mort du joueur) ;
//	c. à défaut, le PONT CANONIQUE par slot (`SlotXUID`) — il couvre les slots qu'AUCUNE vie
//	   nommée ne touche mais qu'une FERMETURE a attribués (`extendSlotXUID`).
//
// LA RÈGLE DE COLLISION EST RESPECTÉE PAR CONSTRUCTION : quand deux joueurs nommés se partagent
// un slot, (a) et (b) rendent celui qui l'occupait à cet instant-là — jamais « le premier », qui
// est ce que `SlotXUID` retient et ce que le constat P1-7 a fait corriger ailleurs.
//
// CE QUI RÉSISTE N'EST PAS DEVINÉ. Un slot dont aucune vie n'est nommée et que le pont ne nomme
// pas reste sans nom : il entre dans `Coverage.Bridge.UnnamedLives` et dans un `slog.Error` qui
// porte le match, le slot et les bornes. On ne publie pas une identité inventée ; on publie le
// fait qu'il en manque une.

import (
	"log/slog"
	"strconv"
)

// unnamedLivesReport est ce que la passe de nommage final rend — par CAUSE, jamais un total
// seul : les trois voies n'ont pas la même force de preuve, et le résidu n'est pas une voie.
type unnamedLivesReport struct {
	byPrevious, byNext, byBridge int
	// remaining : vies qu'aucune des trois voies n'a nommées. C'est le résidu publié.
	remaining int
	// contested : la part du résidu qui tombe sur une FRONTIÈRE entre deux occupants nommés
	// différents, que rien ne date. Comptée à part parce qu'elle n'appelle pas le même
	// chantier : ici l'identité n'est pas absente, elle est INDÉCIDABLE en l'état des pièces.
	contested int
	// deduced : les INDICES, dans `doc.Tracks`, des pistes que cette passe a nommées.
	//
	// POURQUOI ILS VOYAGENT. Leur identité est une DÉDUCTION, pas une lecture : elle établit
	// qu'un joueur était probablement là, jamais qu'un AUTRE n'y était pas. Les lecteurs qui se
	// servent des identités pour PROUVER UNE ABSENCE — le gate de présence des portages — ont
	// besoin de les distinguer, sans quoi le nommage final éteindrait leur abstention (mesure :
	// un portage et 101 frames perdus sur `d9781168`, cf. carrierPresenceOf).
	deduced map[int]bool
}

// total rend le nombre de vies que la passe a traitées.
func (r unnamedLivesReport) total() int {
	return r.byPrevious + r.byNext + r.byBridge + r.remaining
}

// nameRemainingLives nomme les pistes que les quatre passes précédentes ont laissées sans
// identité, PAR L'OCCUPATION DU SLOT DANS LE TEMPS. Rend le rapport de couverture.
//
// Les pistes de BOT ne sont pas touchées : `Track.Bot` EST une identité (un bot n'a pas de
// xuid — contrat de `Track.Bot`, document.go).
func nameRemainingLives(tracks []Track, reg IdentityRegistry,
	origin, step uint64) unnamedLivesReport {
	rep := unnamedLivesReport{deduced: map[int]bool{}}
	named := namedLivesBySlot(reg.Vies())
	for i := range tracks {
		if tracks[i].XUID != "" || tracks[i].Bot != "" {
			continue
		}
		from, to := trackSpanUS(tracks[i], origin, step)
		xuid, cause := slotOccupantAround(named[tracks[i].Slot], from, to,
			reg.PontDeSlot(tracks[i].Slot))
		switch cause {
		case occupantPrevious:
			tracks[i].XUID, rep.byPrevious = xuid, rep.byPrevious+1
			rep.deduced[i] = true
		case occupantNext:
			tracks[i].XUID, rep.byNext = xuid, rep.byNext+1
			rep.deduced[i] = true
		case occupantBridge:
			tracks[i].XUID, rep.byBridge = xuid, rep.byBridge+1
			rep.deduced[i] = true
		case occupantContested:
			rep.contested, rep.remaining = rep.contested+1, rep.remaining+1
		case occupantNone:
			rep.remaining++
		}
	}
	return rep
}

// occupantCause dit PAR QUELLE VOIE une vie a été nommée — ou qu'elle ne l'a pas été.
type occupantCause int

const (
	occupantNone occupantCause = iota
	occupantPrevious
	occupantNext
	occupantBridge
	// occupantContested : la vie tombe ENTRE deux vies nommées d'occupants DIFFÉRENTS, et rien
	// ne date la frontière. On refuse, et on compte.
	occupantContested
)

// slotOccupantAround rend le joueur qui occupait ce slot autour de [fromUS, toUS].
//
// L'ORDRE DES VOIES EST CELUI DE LA FORCE DE PREUVE, et il est décrit en tête de fichier : la
// vie nommée qui PRÉCÈDE, puis celle qui SUIT, puis le pont par slot. Chacune est bornée au MÊME
// slot : on ne traverse jamais la frontière d'un slot pour nommer une vie.
func slotOccupantAround(named []lifeSpan, fromUS, toUS int64,
	pont string) (string, occupantCause) {
	var prev, next *lifeSpan
	for i := range named {
		l := &named[i]
		switch {
		case l.to <= fromUS:
			if prev == nil || l.to > prev.to {
				prev = l
			}
		case l.from >= toUS:
			if next == nil || l.from < next.from {
				next = l
			}
		}
	}
	// UNE FRONTIÈRE ENTRE DEUX OCCUPANTS NE SE TRANCHE PAS AU HASARD (2026-09-07). Quand la vie
	// tombe ENTRE deux vies nommées de joueurs DIFFÉRENTS, « la précédente » n'est pas une
	// preuve : c'est un choix par l'ordre, exactement ce que le pont fait déjà et que ce fichier
	// existe pour corriger. Mesuré sur `084a804d` slot 734 — A `[5872..6981]`, la vie non
	// résolue `[7123..7158]`, B `[7457..7591]` : rien dans le film ne dit de quel côté de la
	// relève elle tombe. Ce qui pourrait la dater, ce serait une SUCCESSION (l'instant de
	// bascule lu dans la base) — mais `attributeSuccessions` a déjà couru, et il ne date que les
	// relèves de BOT ; une relève entre deux humains n'est datée par rien de disponible ici. On
	// refuse donc, et on compte.
	switch {
	case prev != nil && next != nil && prev.xuid != next.xuid:
		return "", occupantContested
	case prev != nil:
		return strconv.FormatUint(prev.xuid, 10), occupantPrevious
	case next != nil:
		return strconv.FormatUint(next.xuid, 10), occupantNext
	}
	if pont != "" {
		return pont, occupantBridge
	}
	return "", occupantNone
}

// namedLivesBySlot range les vies NOMMÉES par slot. Une vie sans nom n'y entre pas : elle est ce
// qu'on cherche à nommer, elle ne peut pas servir de preuve.
func namedLivesBySlot(lives []lifeSpan) map[uint32][]lifeSpan {
	out := map[uint32][]lifeSpan{}
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		out[l.slot] = append(out[l.slot], l)
	}
	return out
}

// trackSpanUS rend les bornes d'une piste sur l'horloge du film.
func trackSpanUS(t Track, origin, step uint64) (int64, int64) {
	return int64(origin) + int64(t.StartFrame)*int64(step),
		int64(origin) + int64(t.EndFrame)*int64(step)
}

// logUnnamedLives ALARME sur le résidu — jamais un silence, jamais un affichage.
//
// `slog.Error` et non `Warn` : depuis la décision du 2026-09-07, une vie publiée sans identité
// est un DÉFAUT, pas une donnée. Le message porte de quoi ouvrir l'enquête (le match, le nombre,
// puis le slot et les bornes de la première) sans noyer le journal — un film à fort défaut de
// nommage en produirait des dizaines.
func logUnnamedLives(matchID string, tracks []Track, rep unnamedLivesReport) {
	if rep.total() > 0 {
		slog.Info("rejeu : nommage final des vies restantes",
			"match_id", matchID, "traitees", rep.total(), "parViePrecedente", rep.byPrevious,
			"parVieSuivante", rep.byNext, "parPont", rep.byBridge, "residu", rep.remaining,
			"frontieresIndecidables", rep.contested)
	}
	if rep.remaining == 0 {
		return
	}
	slot, from, to := -1, 0, 0
	for _, t := range tracks {
		if t.XUID == "" && t.Bot == "" {
			slot, from, to = int(t.Slot), t.StartFrame, t.EndFrame
			break
		}
	}
	slog.Error("rejeu : des vies PUBLIEES restent sans identite — defaut de nommage du pont",
		"match_id", matchID, "vies", rep.remaining, "premierSlot", slot,
		"premiereFrameDebut", from, "premiereFrameFin", to)
}
