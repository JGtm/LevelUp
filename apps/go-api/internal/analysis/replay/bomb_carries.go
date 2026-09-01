package replay

// bomb_carries.go — LE CÂBLAGE du calque du PORTEUR DE LA BOMBE d'Assaut (schéma 30) : ce
// que l'appelant fournit, d'où viennent les événements, et ce que l'assemblage en publie.
//
// Il vit à part de `build.go` pour la même raison que `bomb_armings.go` et
// `skull_carries.go` : l'assemblage garde UNE ligne par calque, le détail vit à côté de la
// donnée qu'il produit. La FORME publiée vit dans document_bomb_carries.go, avec la
// provenance des mesures (B1/B2/B3, 2026-09-01).
//
// # AUCUN BALAYAGE DE PLUS N'EST PAYÉ, et c'est le cœur du câblage
//
// La bombe voyage dans le canal des armes tenues, DÉJÀ balayé par `BuildFromFilm` pour
// `weaponChanges` (`opt.WeaponChanges`, brut, familles comprises). Ce calque FILTRE ce
// balayage sur la famille bombe — il ne relit pas le film. Le fil des morts (`opt.Deaths`)
// et le pont slot->xuid (`own.SlotXUID`) sont eux aussi déjà là.
//
// # L'HORLOGE : pourquoi `deathOffsetMS` et pas l'origine du chunk 1
//
// Les transitions sont datées en µs de l'horloge du FILM (`TimestampUS`) ; les morts qui
// FERMENT les périodes sont datées sur l'horloge du MATCH (le fil). La conversion est
// `matchMS = TimestampUS/1000 − deathOffsetMS` — le calage MESURÉ par le pont
// (`bestDeathOffset`, owners.go), c'est-à-dire exactement la grandeur qui met les deux
// sources sur le même axe. L'origine du chunk 1 (ScanFilmClockOrigin, employée par la
// mesure B2) en diffère de 16 à 81 ms sur les films témoins : ici la cohérence avec les
// morts prime, puisque ce sont elles qui bornent les périodes.
//
// # LA GARDE DE MODE : famille bomb, TOUTES variantes
//
// `opt.Bomb.CarryScanned` est posée par l'appelant (`replaybuild.isBombVariant`) sur toute
// variante de la famille bomb, One Bomb COMPRISE : le négatif One Bomb de v29 vise l'anneau
// d'armement, pas le composant d'arme tenue du bipède (cf. document_bomb_carries.go).

import (
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// bombHeldFamily est la FAMILLE de l'objet bombe dans le canal des armes tenues (moitié
// haute de l'identifiant 64 bits). B1 2026-09-01 : unique candidate hors catalogue d'armes
// des 9 films d'Assaut (prise et lâchée sur chacun, médiane 13 slots-vies) ; l'atlas HUD du
// jeu la confirme indépendamment (« ball | bomb », sprite contour-34).
const bombHeldFamily = uint32(0x3fee4fcf)

// bombHeldEventsOf filtre le canal des armes tenues sur la famille bombe et date chaque
// transition sur l'horloge du MATCH. PRISE = transition VERS la famille ; LÂCHER =
// transition DEPUIS — le protocole B2, à l'identique (les kinds du décodeur ne sont pas
// consultés : une ré-annonce de bombe n'existe pas, elle n'est jamais une arme de spawn).
func bombHeldEventsOf(changes []filmdec.HeldWeaponChange, deathOffsetMS int64) []HeldObjectEvent {
	var out []HeldObjectEvent
	for _, ch := range changes {
		matchMS := int(int64(ch.TimestampUS)/1000 - deathOffsetMS)
		if ch.Family == bombHeldFamily {
			out = append(out, HeldObjectEvent{TimeMS: matchMS, Slot: ch.Slot, Pickup: true})
		}
		if ch.Previous == bombHeldFamily && ch.Family != bombHeldFamily {
			out = append(out, HeldObjectEvent{TimeMS: matchMS, Slot: ch.Slot, Pickup: false})
		}
	}
	return out
}

// buildBombCarries projette la chronologie reconstruite sur l'axe de frames, sous le gate de
// présence des pistes publiées. Pur, testable sans film.
//
// `presence` a la même sémantique que pour le crâne (skullCarrierPresence) : un portage
// attribué à un joueur dont AUCUNE vie publiée ne couvre l'intervalle est écarté
// (`CarrierAbsent`) — le calque n'aurait aucune position où poser la bombe ; un portage qui
// déborde est ROGNÉ. Un porteur inconnu de `presence` n'est PAS vérifié.
func buildBombCarries(carry HeldObjectCarry, ctx matchClock,
	presence map[string][]presenceSpan) ([]BombCarry, *BombCarriesCoverage) {
	cov := &BombCarriesCoverage{BombFilm: true, Events: len(carry.Events)}
	cov.Periods = len(carry.Periods)
	out := make([]BombCarry, 0, len(carry.Periods))
	for _, p := range carry.Periods {
		if p.XUID == 0 {
			cov.NoBridge++
			continue
		}
		f0 := ctx.frameOfMatchMS(int64(p.DebutMS))
		if f0 < 0 || f0 >= ctx.frames {
			cov.OutOfWindow++
			continue
		}
		f1 := clampFrame(ctx.frameOfMatchMS(int64(p.FinMS)), ctx.frames)
		if f1 < f0 {
			f1 = f0
		}
		xuid := strconv.FormatUint(p.XUID, 10)
		// Gate de PRÉSENCE : le porteur doit être sur la carte pendant le portage (même
		// règle et mêmes helpers que le crâne).
		if spans := presence[xuid]; len(spans) > 0 {
			span, ok := bestOverlap(spans, f0, f1)
			if !ok {
				cov.CarrierAbsent++
				continue
			}
			if f0 < span.f0 {
				f0 = span.f0
			}
			if f1 > span.f1 {
				f1 = span.f1
			}
		}
		closed := !p.Ouverte
		out = append(out, BombCarry{XUID: xuid, T0: f0, T1: f1, Closed: closed})
		if closed {
			cov.Closed++
			if p.FinParMort {
				cov.ByDeath++
			}
		} else {
			cov.Open++
		}
	}
	cov.Carries = len(out)
	return out, cov
}

// attachBombCarries pose les périodes de portage de la bombe sur le document, avec leur
// couverture. Les événements viennent du balayage des armes tenues DÉJÀ fait
// (`opt.WeaponChanges`) ; le pont et son calage d'horloge viennent d'`own`, comme pour le
// drapeau. Sans pont (`own.SlotXUID` vide), rien n'est reconstruit : aucune période ne
// pourrait être nommée, et la couverture dit ce qui a été vu.
func attachBombCarries(doc *ReplayDocument, opt Options, own OwnerReport, clock replayClock) {
	if !opt.Bomb.CarryScanned {
		return
	}
	events := bombHeldEventsOf(opt.WeaponChanges, own.DeathOffsetMS)
	var carries []BombCarry
	var cov *BombCarriesCoverage
	if len(own.SlotXUID) == 0 {
		cov = &BombCarriesCoverage{BombFilm: true, Events: len(events)}
		slog.Warn("rejeu : portage de la bombe sans pont slot->xuid — aucune periode publiable",
			"match_id", doc.MatchID, "transitions", len(events))
	} else {
		carry := BuildHeldObjectCarry(events, own.SlotXUID, opt.Deaths)
		carries, cov = buildBombCarries(carry, matchClock{
			origin: clock.origin, step: clock.step, frames: clock.frames,
			deathOffsetMS: own.DeathOffsetMS,
		}, skullCarrierPresence(doc.Tracks))
	}
	doc.BombCarries = carries
	if doc.Coverage != nil {
		doc.Coverage.BombCarries = cov
	}
	slog.Info("rejeu : portage de la bombe d'Assaut",
		"transitions", cov.Events, "periodes", cov.Periods, "portages", cov.Carries,
		"fermes", cov.Closed, "ouverts", cov.Open, "parMort", cov.ByDeath,
		"sansPont", cov.NoBridge, "horsFenetre", cov.OutOfWindow,
		"porteurAbsent", cov.CarrierAbsent)
}
