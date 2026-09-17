package grammar

// ability_impulses.go — LES IMPULSIONS DE CAPACITÉ, lues dans les paquets delta.
//
// CE QUE CE BALAYAGE LIT, ET D'OÙ VIENT LA RÈGLE. Le tag externe des composants bipède
// `biped-spartan-ability` (i57, prédit) et `biped-spartan-ability-non-predicted-state`
// (i59) est un `R(2)` : il a QUATRE valeurs, et la production n'en exploitait qu'UNE — le
// corps `tag == 3` d'i59, qui porte le GRAPPIN (grapple_state.go, 2026-08-16). Le lot R8
// du 2026-09-03 a mesuré la deuxième : **le corps `tag == 1` date une IMPULSION du
// PROPULSEUR** (rapport RAPPORT_R8_USAGE_REPULSEUR_PROPULSEUR_2026-09-03.md §8).
//
// CE QUI LE PROUVE, EN TROIS CHIFFRES (R8 §8.8, quatre films de famille A, rang attribué
// DANS LA MÊME VIE) : 0,361 impulsion par vie de propulseur (22 sur 61) contre 0,011 par
// vie de répulseur (1 sur 90 — et le répulseur est PLUS porté), et **0,000 sur 132 vies de
// grappin**, qui a son propre tag. L'oracle physique indépendant (pic de vitesse
// horizontale du porteur) rend 6,2 à 8,8 m/s à ces instants contre 2,9 à 3,6 pour un
// instant tiré au hasard dans la même vie. Vérité terrain utilisateur du 2026-09-03
// (Theater, film `1cd3848a`) : 5 usages relevés, 5 impulsions rendues, écart ≤ 1 s.
//
// CE QUE CE CANAL NE PORTE PAS. Le RÉPULSEUR n'y est pas — négatif MESURÉ, pas supposé
// (R8 §8.7 et rapport R9 : ses trois portes sont fermées). Un appelant ne doit donc jamais
// lire « impulsion » comme « usage d'un équipement quelconque ».
//
// L'IDENTITÉ NE VIENT PAS DU COMPOSANT. Le `sub` (R(2) interne d'i57) a été essayé comme
// discriminant puis RÉFUTÉ par le corpus : sur `1cd3848a`, ses quatre valeurs tombent
// TOUTES majoritairement sur le rang du propulseur (R8 §8.5). L'identité vient du canal
// i48 (`ScanFilmAbilityRanks`), rang lu dans la MÊME VIE et ANTÉRIEUREMENT — et cette
// jointure vit chez l'assembleur (replay/document_ability_impulses.go), qui seul connaît
// les vies. Ce fichier ne rend que QUI et QUAND.
//
// AUCUNE GRAMMAIRE NOUVELLE N'EST PORTÉE ICI : les deux désérialiseurs publient déjà le
// tag depuis le 2026-08-16 (`observateur.SpartanAbilityHook`, `observateur.AbilityNonPredictedHook`). Ce balayage
// est le patron exact de `ScanFilmGrappleReads` — même composant, tag 1 au lieu de 3.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// sont des globaux de paquet.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// abilityImpulseTag : la valeur du tag externe qui date une impulsion. Le 3 est le grappin
// (grapple_state.go), le 0 et le 2 sont l'état de repos (1 572 et 1 565 lectures sur
// `00ba2e1c`, tous rangs mélangés, pic au niveau du témoin aléatoire — R8 §8.2).
const abilityImpulseTag = 1

// abilityPredictedName / abilityPredictedNameAlt : les deux étiquettes de registre d'i57 —
// les films portent l'une OU l'autre (avec ou sans le suffixe `-component`, même dualité
// que `grappleComponentName` pour i59). L'index d'itérateur est résolu PAR NOM dans le
// registre du film, jamais câblé.
const (
	abilityPredictedName    = "biped-spartan-ability-component"
	abilityPredictedNameAlt = "biped-spartan-ability"
)

// ScanFilmAbilityImpulses décode les impulsions de capacité (corps tag==1 d'i57 et d'i59)
// dans les paquets delta du film de dir. Les lectures sortent TRIÉES par instant, puis par
// slot — un ordre total, pour que deux exécutions rendent le même artefact.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe
// `observateur.SpartanAbilityHook` et `observateur.AbilityNonPredictedHook`, qui sont des globaux de paquet.
// restaurés à la sortie, y compris en cas d'erreur.
//
// ScanFilmAbilityImpulses est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film, ouvre un
// contexte pour elle seule, puis appelle [ScanAbilityImpulses]. La cuisson, elle, passe le
// contexte qu'elle partage entre tous ses balayages.
func ScanFilmAbilityImpulses(dir string) ([]types.AbilityImpulse, types.AbilityImpulseStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, types.AbilityImpulseStats{}, err
	}
	return ScanAbilityImpulses(contexteDeBobine(film))
}

// ScanAbilityImpulses décode les impulsions de capacité d'un film DEJA CHARGE. Cf.
// [ScanFilmAbilityImpulses] pour la doctrine du balayage.
func ScanAbilityImpulses(fc *FilmContext) ([]types.AbilityImpulse, types.AbilityImpulseStats, error) {
	var st types.AbilityImpulseStats
	s, err := resolveAbilityScan(fc)
	if err != nil {
		return nil, st, err
	}
	sc := &abilityImpulseScanner{st: &st, gram: s.gram,
		i57idx: componentIndexOfAny(s.gram.arch, abilityPredictedName, abilityPredictedNameAlt),
		i59idx: componentIndexOfAny(s.gram.arch, grappleComponentName, grappleComponentNameAlt),
	}
	if sc.i57idx < 0 && sc.i59idx < 0 {
		// AUCUNE ERREUR ICI, et c'est délibéré : `ScanFilmGrappleReads` refuse un film sans
		// i59 parce que le grappin EST i59 ; ce canal-ci a deux portes et se contente d'une.
		// Les deux absentes, le film ne transmet pas la capacité — un fait mesuré, que
		// `Absent` publie au lieu de le confondre avec un film sans propulseur.
		st.Absent, st.Scanned = true, true
		return nil, st, nil
	}

	// Le hook est LA grammaire : c'est le déserialiseur lui-même qui publie, on ne relit pas
	// les bits à côté de lui (même règle que ScanFilmGrappleReads et ScanFilmAbilityRanks).
	obs := NouvelleObservation()
	obs.SpartanAbilityHook = func(tag, _, _ uint64, _ bool) { sc.tag57, sc.got57 = tag, true }
	obs.AbilityNonPredictedHook = func(s AbilityNonPredictedState) {
		sc.tag59, sc.got59 = uint64(s.Tag), true
	}
	sc.gram.obs = obs

	walkDeltaBipedRecords(s.fc, s.chunks, s.slots, s.gram.lay, func(r deltaBipedRecord) {
		st.Records++
		sc.account(r.Payload, r.I0, r.Total, r.Mask, r.Slot, r.Chunk, r.Packet)
	})
	sortAbilityImpulses(sc.out)
	st.Scanned = true
	return sc.out, st, nil
}

// componentIndexOfAny résout l'index d'itérateur d'un composant par ses étiquettes
// possibles, ou -1. Les films portent l'une OU l'autre (avec ou sans `-component`).
func componentIndexOfAny(arch Archetype, names ...string) int {
	for _, n := range names {
		if ids := arch.indicesOf(n); len(ids) > 0 {
			return ids[0]
		}
	}
	return -1
}

// abilityImpulseScanner porte l'état du balayage : compteurs, capture des deux hooks, et
// sortie.
type abilityImpulseScanner struct {
	st             *types.AbilityImpulseStats
	out            []types.AbilityImpulse
	gram           grammaireRecord
	i57idx, i59idx int
	tag57, tag59   uint64
	got57, got59   bool
}

// account marche UN record et impute ses lectures aux compteurs. LES DEUX COMPOSANTS SE
// LISENT DANS LE MÊME RECORD quand le masque les annonce tous deux : la marche s'arrête au
// PLUS LOINTAIN des deux, et le hook de l'autre a déjà parlé en chemin.
func (sc *abilityImpulseScanner) account(pay []byte, i0, total int, idx []int,
	slot uint32, chunk int, pk FilmPacket) {
	has57 := sc.i57idx >= 0 && maskHas(idx, sc.i57idx)
	has59 := sc.i59idx >= 0 && maskHas(idx, sc.i59idx)
	if !has57 && !has59 {
		return
	}
	if has57 {
		sc.st.WithI57++
	}
	if has59 {
		sc.st.WithI59++
	}
	sc.got57, sc.got59 = false, false
	target := sc.i57idx
	if !has57 || (has59 && sc.i59idx > target) {
		target = sc.i59idx
	}
	walkRecordTo(pay, i0, total, idx, sc.gram, target)
	sc.emit(has57, has59, slot, chunk, pk)
}

// emit impute les DEUX lectures du record — celle du composant prédit et celle de son jumeau —
// puis compte ce que la marche n'a pas atteint.
//
// LES DEUX CONTRIBUENT, ET IL FAUT QUE LES DEUX CONTRIBUENT : i57 et i59 sont co-transmis, et
// n'en publier qu'un ferait tomber `coverage.abilityImpulses.reads` de moitié dans le document
// servi (86 -> 43 sur le film de référence) sans qu'aucun compteur ne le dise. Extrait d'
// `account` pour être testable sans film : le balayage, lui, ne se juge que sur pièces.
func (sc *abilityImpulseScanner) emit(has57, has59 bool, slot uint32, chunk int, pk FilmPacket) {
	sc.publish(has57 && sc.got57, sc.tag57, true, slot, chunk, pk)
	sc.publish(has59 && sc.got59, sc.tag59, false, slot, chunk, pk)
	sc.imputeUnread(has57, has59)
}

// publish compte une lecture aboutie et l'émet si son tag date une impulsion.
func (sc *abilityImpulseScanner) publish(got bool, tag uint64, predicted bool,
	slot uint32, chunk int, pk FilmPacket) {
	if !got {
		return
	}
	sc.st.Read++
	if tag != abilityImpulseTag {
		return
	}
	sc.st.Tag1++
	sc.out = append(sc.out, types.AbilityImpulse{
		Slot: slot, Chunk: chunk, PacketIndex: pk.Index,
		TimestampUS: pk.TimestampUS, Predicted: predicted,
	})
}

// imputeUnread compte les composants ANNONCÉS par le masque que la marche n'a pas atteints
// — la mesure de ce que ce balayage ne voit pas. Un composant annoncé et non lu n'est pas
// une absence d'impulsion : c'est une lecture perdue, et le dénominateur doit le dire.
func (sc *abilityImpulseScanner) imputeUnread(has57, has59 bool) {
	if has57 && !sc.got57 {
		sc.st.Unread++
	}
	if has59 && !sc.got59 {
		sc.st.Unread++
	}
}

// sortAbilityImpulses ordonne les lectures sur (instant, slot, composant) — un ordre TOTAL.
// Un tri partiel laisserait l'ordre des lectures d'un même paquet dépendre du parcours,
// donc l'artefact dépendre de rien de mesurable.
func sortAbilityImpulses(out []types.AbilityImpulse) {
	if len(out) < 2 {
		return
	}
	lessImpulse := func(a, b types.AbilityImpulse) bool {
		if a.TimestampUS != b.TimestampUS {
			return a.TimestampUS < b.TimestampUS
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Predicted && !b.Predicted
	}
	// Tri par insertion : la sortie est DÉJÀ presque triée (le film se marche dans l'ordre
	// des chunks et des paquets), et le canal est rare — 0 à 160 lectures par film mesurées
	// sur 22 films (R8 §8.5-8.6). Une dépendance de tri pour cela n'en vaut pas la peine.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && lessImpulse(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}
