package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// vip_crown.go — LA REGLE : de quoi est faite une periode de port de la COURONNE VIP.
//
// # Le principe, en une phrase
//
// On ne decode PAS la couronne : on lit QUI a ete DESIGNE VIP, et QUAND. Le statborg date a la
// milliseconde chaque SELECTION (`comp 22 A` = `TimesSelectedAsVip`, resolu au gate corrige
// `VIP_temoin_corrige.log` : 100 % par joueur x3, temoin decale 0). Chaque selection OUVRE une
// periode de port pour ce joueur ; elle se FERME a la mort du VIP (il perd la couronne) ou a la
// selection suivante du meme slot. La couronne est a la position du joueur qui la porte — le
// client la dessine sur sa piste deja publiee, comme le drapeau porte.
//
// # Pourquoi ce canal, et ce qu'il a coute d'etablir
//
// Le film ne porte PAS le bit VIP (script-side) : la voie est le statborg. La reconstruction a
// ete MESUREE (`TestVIPPeriodes`, 3 films) : les periodes somment, par joueur, a `TimeAsVip` de
// l'API au SUB-SECONDE (24/24 joueurs a +0,2-0,3 s ; recouv 100 % 3/3), contre un temoin
// d'attribution aleatoire effondre (exactitude 8/8 contre 0-1/8). La surcote de ~0,2 s est
// l'ecart entre l'instant de la mort et la fin du statut VIP — le biais joue CONTRE la duree
// affirmee, jamais en sa faveur.
//
// # Ce qui n'est PAS decide ici, et c'est delibere
//
// Le MODE. `comp 22 A` vaut `flag_grabs` en CTF et `vip_selected` en VIP — le meme emplacement,
// le sens change avec le mode. La garde de mode est chez l'APPELANT (`replaybuild`, qui connait
// `game_variant_name`), exactement comme la colline de KOTH : ce paquet consomme un `VipInput`
// et ne devine aucun mode. Un film non-VIP ne fournit pas de `VipInput.Scanned`, et la couronne
// reste vide.

// VipInput est CE QUE L'APPELANT FOURNIT du VIP. Entree de DONNEES, comme `Flag` et `Score`.
//
// LA GARDE DE MODE EST ICI, chez l'appelant : `comp 22 A` vaut `flag_grabs` en CTF, donc seul
// un appelant qui SAIT que le match est VIP (par `game_variant_name`) doit poser `Scanned`. Ce
// paquet ne devine aucun mode — `Scanned` faux = ni couronne ni couverture.
type VipInput struct {
	// Scanned dit que l'appelant a RECONNU un film VIP et fournit de quoi lire.
	Scanned bool
	// Records sont les enregistrements d'entite du film — les MEMES que la courbe de score et le
	// drapeau : ils portent les selections VIP (`comp 22 A`) et les progressions du compteur de
	// morts qui identifient les slots. Aucun fait de match n'entre : le VIP se nomme par les
	// instants de mort, et le calque est donc publiable hors ligne.
	Records []objectiveevents.StatRecord
}

// VipCrownScan porte ce que le film rend du VIP. Les lectures voyagent ensemble, et `Scanned`
// dit qu'elles ont abouti : une liste vide sans lui serait indistinguable d'un film non-VIP.
type VipCrownScan struct {
	Scanned bool
	// Events sont les evenements nommes de la table VIP (`vip_selected`), PAR SLOT statborg.
	Events []objectiveevents.NamedEvent
	// Identity est le pont slot statborg -> xuid, PAR MANCHE (le slot est reattribue d'une
	// manche a l'autre). Une selection est nommee par l'identite de SA manche, choisie sur
	// l'instant de la selection. Sur un film mono-manche c'est le pont plat, a l'octet pres.
	Identity objectiveevents.RoundIdentity
	// Deaths est le fil des morts du film (horloge du MATCH, comme les evenements nommes).
	Deaths []Death
}

// L'AXE DE TEMPS est le `matchClock` partage (match_clock.go) : la conversion match <-> frames
// etait la meme que celle du drapeau et du crane, elle n'est plus ecrite qu'une fois.

// vipRawPeriod est une periode de port, en horloge du MATCH, avant mise en frames.
type vipRawPeriod struct {
	slot       int
	xuid       string
	t0MS, t1MS int64
	closed     bool
	cause      string // "mort", "selection", "fin"
}

// vipReconstructPeriods borne chaque selection au PREMIER de : la MORT du VIP, la selection
// suivante du meme slot, la fin (`endMS`). C'est le coeur PUR, partage par la mesure
// (`TestVIPPeriodes`) et par le build — une seule regle, une seule source.
func vipReconstructPeriods(events []objectiveevents.NamedEvent, identity objectiveevents.RoundIdentity,
	deaths []Death, endMS int64) []vipRawPeriod {
	sels := vipSelectionOpenings(events, identity)
	byXUID := deathTimesByXUID(deaths)
	next := vipNextSelectionOfSlot(sels)
	out := make([]vipRawPeriod, 0, len(sels))
	for i, s := range sels {
		t1, cause := endMS, "fin"
		if d, ok := firstAfter(byXUID[s.xuid], s.t0MS); ok && d < t1 {
			t1, cause = d, "mort"
		}
		if n, ok := next[i]; ok && n < t1 {
			t1, cause = n, "selection"
		}
		s.t1MS, s.closed, s.cause = t1, cause != "fin", cause
		out = append(out, s)
	}
	return out
}

// vipSelectionOpenings rend les selections VIP nommees, triees par instant puis slot.
func vipSelectionOpenings(events []objectiveevents.NamedEvent, identity objectiveevents.RoundIdentity) []vipRawPeriod {
	out := make([]vipRawPeriod, 0, len(events))
	for _, e := range events {
		if e.Stat != objectiveevents.StatVipSelected {
			continue
		}
		out = append(out, vipRawPeriod{slot: e.Slot, xuid: identity.At(e.Slot, e.TimeMS), t0MS: int64(e.TimeMS)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].t0MS != out[j].t0MS {
			return out[i].t0MS < out[j].t0MS
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// vipNextSelectionOfSlot rend, par index de selection, l'instant de la selection SUIVANTE du
// meme slot — une selection ferme la precedente (on ne peut etre redesigne sans l'avoir perdue).
func vipNextSelectionOfSlot(sels []vipRawPeriod) map[int]int64 {
	last := map[int]int{}
	out := map[int]int64{}
	for i, s := range sels {
		if prev, ok := last[s.slot]; ok {
			out[prev] = s.t0MS
		}
		last[s.slot] = i
	}
	return out
}

// vipMatchEndMS rend une borne STRICTEMENT posterieure a tout fait date (selection ou mort).
func vipMatchEndMS(events []objectiveevents.NamedEvent, deaths []Death) int64 {
	var end int64
	for _, e := range events {
		if int64(e.TimeMS) > end {
			end = int64(e.TimeMS)
		}
	}
	for _, d := range deaths {
		if d.TimeMS > end {
			end = d.TimeMS
		}
	}
	return end + 1
}

// buildVipCrown rend les periodes de port de la couronne en FRAMES et la couverture du calque.
//
// Rend (nil, nil) quand rien n'a ete balaye (film non-VIP), et (nil, couverture) quand le film
// est VIP mais qu'aucune periode ne sort — la couverture dit alors POURQUOI.
func buildVipCrown(scan VipCrownScan, ctx matchClock) ([]VipPeriod, *VipCrownCoverage) {
	if !scan.Scanned {
		return nil, nil
	}
	cov := &VipCrownCoverage{VipFilm: true}
	endMS := vipMatchEndMS(scan.Events, scan.Deaths)
	if e := ctx.matchMSOfFrame(ctx.frames - 1); e > endMS {
		endMS = e
	}
	raws := vipReconstructPeriods(scan.Events, scan.Identity, scan.Deaths, endMS)
	cov.Selections = len(raws)
	out := make([]VipPeriod, 0, len(raws))
	for _, r := range raws {
		if r.xuid == "" {
			cov.NoBridge++
			continue
		}
		f0 := ctx.frameOfMatchMS(r.t0MS)
		if f0 < 0 || f0 >= ctx.frames {
			cov.OutOfWindow++
			continue
		}
		f1 := clampFrame(ctx.frameOfMatchMS(r.t1MS), ctx.frames)
		if f1 < f0 {
			f1 = f0
		}
		out = append(out, VipPeriod{XUID: r.xuid, T0: f0, T1: f1, Closed: r.closed})
		cov.tally(r)
	}
	cov.Periods = len(out)
	return out, cov
}

// attachVipCrown pose les periodes de la couronne sur le document, avec leur couverture.
//
// LE PONT D'IDENTITE (slot statborg -> xuid) SE FAIT ICI, comme pour le drapeau, par les seuls
// INSTANTS DE MORT — aucune base — et PAR MANCHE (le slot est reattribue d'une manche a l'autre ;
// une selection est nommee par l'identite de sa manche). `own.DeathOffsetMS` cale l'horloge du
// fil des morts (et des evenements nommes, meme horloge) sur l'axe des frames.
func attachVipCrown(doc *ReplayDocument, opt Options, own OwnerReport, clock replayClock) {
	in := opt.Vip
	if !in.Scanned {
		return
	}
	scan := VipCrownScan{
		Scanned:  true,
		Events:   objectiveevents.NamedEventsFrom(in.Records, objectiveevents.ObjectiveTypeVip),
		Identity: objectiveevents.ResolveRoundIdentity(in.Records, deathInstantsOf(opt.Deaths)),
		Deaths:   opt.Deaths,
	}
	periods, cov := buildVipCrown(scan, matchClock{
		origin: clock.origin, step: clock.step, frames: clock.frames,
		deathOffsetMS: own.DeathOffsetMS,
	})
	doc.VipCrown = periods
	if doc.Coverage != nil {
		doc.Coverage.VipCrown = cov
	}
}
