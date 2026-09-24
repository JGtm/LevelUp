package grammar

// keyframe_world.go — PONT réutilisable : reconstruction OFFLINE du World
// (mapping slot -> typeIndex) depuis la table keyframe type-2 d'un film, SANS Cheat
// Engine. Le walker porté ci-dessous est la version DURCIE validée par cmd/tmp_kfworldpos
// (249/250 entités + 8/8 bipeds slots 512-519 contre l'oracle CE), copiée fidèlement.
//
// En-tête RÉEL d'un record keyframe type-2 (RE FUN_141f86704) :
//   [id:32][field:26][ti:6] = 64 bits.
//   ti  = readBits(q+58, 6)  (== mot 32-bit à q+32 quand field26==0, cas du spawn)
//   gen = id>>30 ∈ {1,2,3}   (respawns mid-match ; gen==0 = handle null rejeté)
//   slot = id & 0x3FFFFFFF
//
// WorldFromKeyframe binde chaque record via World.BindFull(fullID, ti). Aucun input CE :
// seul le payload de frame type-2 du chunk (pay) est lu.

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

const (
	kfSent     = 0xFFFFFFFF // id sentinelle
	kfTableCap = 8192       // borne haute de slot
	// objectArchetypeCount (.exe DAT_144e61d88 COUNT=0x32) : ti valide < 50. Borne SÉMANTIQUE,
	// sourcée de l'exe — et le lot 3 (2026-08-30) a fermé la boucle par l'autre bout :
	// len(reg.Archetypes) vaut AUSSI 50 depuis que parseRegistry s'arrête à la fin structurelle
	// du registre (le « 118 » d'avant était la taille du fichier divisée par celle d'un bloc).
	// Constante Halo Infinite : un autre titre exigerait de re-sourcer ce COUNT depuis son .exe
	// (walker keyframe = RE Halo-Infinite-spécifique pour l'instant).
	kfArchMax = 50
	// keyframeRecordTIBit est la position, EN BITS DEPUIS LE DEBUT DU RECORD, du champ `ti`
	// de 6 bits : c'est le point exact ou TraverseEntity doit prendre la main (en-tete
	// [id:32][field:26][ti:6] documente en tete de fichier). Definition unique du paquet —
	// les marcheurs d'image-cle la reutilisent au lieu de re-ecrire le litteral 58.
	keyframeRecordTIBit = 58
)

// KeyframeRec est un record de la table keyframe reconstruit offline.
type KeyframeRec struct {
	Slot, TI, Gen, Bit int
}

// kfReadBits lit n bits big-endian à partir de la position bit pos (0-safe hors borne).
//
// Lecture par mot ([source.BitsAt]) sur le domaine ou elle coincide avec la boucle
// d'origine : `pos >= 0` et `0 <= n <= 64`. C'est la primitive la plus chaude de toute la
// cuisson (58 a 61 % du CPU au profil du 2026-09-02) : `kfScanNext` l'appelle pour CHAQUE
// position de bit du payload d'image-cle.
func kfReadBits(buf []byte, pos, n int) uint64 {
	if pos >= 0 && n >= 0 && n <= 64 {
		return source.BitsAt(buf, pos, uint(n))
	}
	return kfReadBitsLoop(buf, pos, n)
}

// kfReadBitsLoop est la lecture bit a bit d'origine, gardee pour les positions NEGATIVES
// (l'indexation y panique, comme avant) et les largeurs > 64 (seuls les 64 derniers bits
// lus sont rendus). Aucun appelant de production n'y passe : elle est la pour que la
// reecriture par mot ne CHANGE rien, pas pour servir.
func kfReadBitsLoop(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

// kfBitAt lit le bit unique à la position p (0 hors borne).
func kfBitAt(buf []byte, p int) uint64 {
	if idx := p >> 3; idx < len(buf) {
		return uint64(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// kfValidAnchor : en-tête de record valide ? gen∈{1,2,3}, prev<slot<cap. FILTRE FORT
// anti-faux-positif : le mot 32-bit à q+32 doit être < archMax (vrai pour TOUT record au
// spawn, car field26==0 -> ce mot == ti < 50). ti/field26 EXTRAITS de façon durcie.
// field26 n'est PAS rendu : il ne sert qu'à expliquer pourquoi le filtre fort tient
// (field26==0 au spawn -> le mot 32-bit vaut ti), et aucun appelant ne l'a jamais lu.
func kfValidAnchor(buf []byte, q, prevSlot, total int) (slot, ti, gen int, ok bool) {
	// La garde est ICI AUSSI parce qu'elle protege la LECTURE qui suit : `kfReadBits` a une
	// position negative panique (convention preservee, cf. `source.BitsAt`). [kfAnchorFromID] la
	// rejoue pour son autre appelant, qui lui a deja lu l'identifiant.
	if q < 0 || q+64 > total {
		return
	}
	return kfAnchorFromID(buf, q, kfReadBits(buf, q, 32), prevSlot, total)
}

// kfAnchorFromID est [kfValidAnchor] quand l'appelant a DEJA lu les 32 bits d'identifiant a
// la position q. `kfScanNext` les lit pour son test de sentinelle et les relisait ensuite :
// deux lectures de 32 bits par position de bit du payload, soit la moitie du temps de la
// primitive la plus chaude de la cuisson. Meme logique, memes gardes, meme resultat — `id`
// est une fonction pure de (buf, q).
func kfAnchorFromID(buf []byte, q int, id uint64, prevSlot, total int) (slot, ti, gen int, ok bool) {
	if q < 0 || q+64 > total {
		return
	}
	if id == kfSent {
		return
	}
	gen = int(id >> 30)
	if gen == 0 { // handle null / zone de données
		return
	}
	slot = int(id & 0x3FFFFFFF)
	if slot <= prevSlot || slot >= kfTableCap {
		return
	}
	if kfReadBits(buf, q+32, 32) >= kfArchMax { // filtre fort : field26==0 & ti<50
		return
	}
	ti = int(kfReadBits(buf, q+keyframeRecordTIBit, 6)) // extraction durcie (== mot 32-bit quand field26==0)
	ok = true
	return
}

// kfCand est une ancre candidate en cours d'évaluation. Les quatre champs sont exactement les
// quatre clés de tri de betterThan, dans l'ordre — les passer groupés évite la file de huit
// entiers nus où deux clés s'inversent sans que rien ne le signale.
type kfCand struct {
	consecutive int // 1 si slot == prevSlot+1
	gen         int
	slot        int
	bit         int // position en bits dans le payload
}

// betterThan : priorité de l'ÉLECTION — consécutif (slot==prev+1) d'abord, puis gen BAS
// (gen==1 réel bat un faux gen 2/3 au même slot ; un vrai gen≥2 mid-match reste choisi s'il
// est seul candidat du slot), puis slot bas, puis bit bas.
//
// C'EST UN REPLI NOMMÉ (`repli_ancre_d_image_cle_par_election`, lot M3.1 du 2026-09-23) : il
// ne décide que lorsqu'aucun voisin immédiat de génération 1 ne suit ET qu'aucun en-tête exact
// de bipède ne précède l'ancre qu'il élirait (cf. [kfScanNext]). « Slot bas » est la bonne
// intuition d'une table à slots croissants, et c'est aussi son défaut mesuré : une fausse ancre
// de génération 1 et de slot PLUS BAS, prise dans le corps du dernier record, bat le vrai
// record suivant situé plus près (sonde P2 : 81c02726 morceau 9, slot 256 `ti 0` élu à +62 618
// bits devant sept bipèdes ; a0c36016 morceau 2, slot 385 `ti 19` devant huit).
func (c kfCand) betterThan(best kfCand) bool {
	if c.consecutive != best.consecutive {
		return c.consecutive > best.consecutive
	}
	if c.gen != best.gen {
		return c.gen < best.gen
	}
	if c.slot != best.slot {
		return c.slot < best.slot
	}
	return c.bit < best.bit
}

// kfDecision dit COMMENT le balayeur a choisi l'ancre suivante. Chaque décision se compte
// ([KeyframeWalkStats]) : un balayeur qui décide sans le dire n'est pas une mesure.
type kfDecision uint8

const (
	// kfAucune : aucun candidat — la table est finie (sentinelles) ou le payload épuisé.
	kfAucune kfDecision = iota
	// kfVoisin : le voisin immédiat (slot+1, génération 1). C'est l'enchaînement de l'écrivain.
	kfVoisin
	// kfRecalage : l'en-tête EXACT d'un bipède (identifiant de génération 1, mot d'archétype de
	// 32 bits égal à [BipedTypeIndex]) précède l'ancre que l'élection retiendrait.
	kfRecalage
	// kfElection : le repli nommé (cf. [kfCand.betterThan]).
	kfElection
)

// kfScanNext : POSITION EN BITS de la prochaine ancre après `from` dans la fenêtre `maxWin`
// (-1 si aucune), la décision qui l'a choisie, `finDeTable` quand la fenêtre a rencontré la fin
// de table (2 048 identifiants sentinelles d'affilée), et `traine` : la longueur de la traînée de
// sentinelles sur laquelle la fenêtre FINIT sans candidat — [kfScanGlissant] reprend la fenêtre
// suivante à son début, pour qu'une fin de table à cheval sur deux fenêtres se reconnaisse
// (constat F5 de la revue adverse du lot M3.1, 2026-09-24).
//
// TROIS DÉCISIONS, DANS CET ORDRE (lot M3.1, 2026-09-23) :
//
//  1. VOISIN : slot==prev+1 ET gen==1, n'importe où dans la fenêtre — retour immédiat, non
//     ambigu, inchangé.
//  2. RECALAGE : le PREMIER en-tête exact de bipède (gen==1, mot d'archétype == 35) quand il
//     précède l'ancre que l'élection retiendrait, ou qu'il EST cette ancre. Recensement exhaustif des payloads d'image-clé
//     de quatre films (dad793c7, 81c02726, a0c36016, b1f01a33) : 649 en-têtes de cette forme,
//     ZÉRO faux — les 14 faux d'a0c36016 portent tous une génération 2 ou 3 et un mot de taille
//     `n1` aberrant. Une ancre élue PLUS LOIN et de slot plus bas que ce bipède est
//     nécessairement fausse : la table est à slots croissants.
//  3. ÉLECTION : le repli ([kfCand.betterThan]).
//
// LA RÈGLE « GÉNÉRATION 1 PUIS LE PLUS PROCHE » (V2 de la sonde P2) A ÉTÉ MESURÉE ET ÉCARTÉE :
// sur les quatre films elle rend les bipèdes, mais PERD les autres ancres par milliers
// (a0c36016 : 14 659 -> 11 890 ; b1f01a33 : 6 719 -> 5 524, trois bipèdes perdus), parce qu'une
// fausse ancre de génération 1 plus proche que le vrai record suivant se trouve presque partout
// hors de la bande des bipèdes. Le recalage ne change l'élection QUE devant un en-tête exact.
func kfScanNext(buf []byte, from, prevSlot, total, maxWin int) (at int, dec kfDecision, finDeTable bool, traine int) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	exact := -1
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q < end && q+64 <= total; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				finDeTable = true
				break
			}
			continue
		}
		sentStreak = 0
		s, ti, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q, kfVoisin, false, 0 // consécutif gen-1 : non ambigu
		}
		if exact < 0 && g == 1 && ti == BipedTypeIndex {
			exact = q
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	switch {
	case exact >= 0 && (at < 0 || exact <= at):
		// `<=` : quand l'élection retient elle-même l'en-tête exact, c'est lui qui la justifie —
		// la décision se compte en recalage, et `Elections` ne compte que les choix du repli.
		return exact, kfRecalage, finDeTable, 0
	case at >= 0:
		return at, kfElection, finDeTable, 0
	}
	return -1, kfAucune, finDeTable, sentStreak
}

// kfScanGlissant est [kfScanNext] dont une fenêtre SANS AUCUN candidat n'arrête plus la marche :
// la recherche GLISSE de fenêtre en fenêtre jusqu'au premier candidat, à la fin de table ou à la
// fin du payload. Il rend aussi le nombre de glissements.
//
// C'EST CE QUI COUPAIT LE SUFFIXE DE LA TABLE (mécanisme B de la sonde P2 : a0c36016 morceau 1,
// aucun candidat dans les 120 000 bits qui suivent le slot 122, le premier en-tête crédible à
// +125 270 bits — 120 ancres perdues). Le jeu n'a pas de fenêtre ; une fenêtre vide n'est pas
// une fin de table.
//
// UNE FENÊTRE QUI FINIT SUR DES SENTINELLES ne sait pas encore si c'est la fin de table : la
// suivante reprend au DÉBUT de cette traînée et la recompte en entier. Sans cette reprise, une fin
// de table coupée par la frontière de deux fenêtres (moins de 2 048 sentinelles de chaque côté)
// n'était jamais reconnue, et le glissement lisait des ancres au-delà de la table — ce que
// l'ancienne fenêtre, qui arrêtait la marche, ne pouvait pas faire. La reprise recule de moins de
// 2 048 bits sur une fenêtre de 120 000 : la recherche avance toujours.
//
// `glissements` ne compte que les fenêtres vides FRANCHIES pour atteindre une ancre : une
// recherche qui s'achève sur la fin de table ou du payload n'a rien franchi, elle rend 0.
func kfScanGlissant(buf []byte, from, prevSlot, total, maxWin int) (at int, dec kfDecision, glissements int) {
	vides := 0
	for f := from; f+64 <= total; {
		var fin bool
		var traine int
		at, dec, fin, traine = kfScanNext(buf, f, prevSlot, total, maxWin)
		if at >= 0 {
			return at, dec, vides
		}
		if fin {
			break
		}
		vides++
		f += maxWin - traine
	}
	return -1, kfAucune, 0
}

// KeyframeWalkStats compte ce que le balayeur d'image-clé a décidé, par payload ou cumulé sur
// un film ([KeyframeWalkStats.Ajouter]). Les champs se lisent ensemble : `Records` ancres, dont
// chacune (sauf la première) a été atteinte par un voisin, un saut de largeur, un recalage ou
// une élection.
type KeyframeWalkStats struct {
	// Payloads est le nombre de payloads d'image-clé marchés.
	Payloads int
	// Records est le nombre d'ancres rendues ; Bipedes celles d'archétype [BipedTypeIndex].
	Records, Bipedes int
	// Voisins : ancres atteintes par le voisin immédiat ; Sauts : par le saut de largeur.
	Voisins, Sauts int
	// Recalages : ancres atteintes par l'en-tête exact d'un bipède, devant une élection.
	Recalages int
	// Elections : ancres choisies par le REPLI nommé (`repli_ancre_d_image_cle_par_election`).
	Elections int
	// Glissements : fenêtres VIDES traversées sans arrêter la marche (cf. [kfScanGlissant]).
	Glissements int
}

// Ajouter cumule `o` dans `s`.
func (s *KeyframeWalkStats) Ajouter(o KeyframeWalkStats) {
	s.Payloads += o.Payloads
	s.Records += o.Records
	s.Bipedes += o.Bipedes
	s.Voisins += o.Voisins
	s.Sauts += o.Sauts
	s.Recalages += o.Recalages
	s.Elections += o.Elections
	s.Glissements += o.Glissements
}

// compter note une décision de balayage.
func (s *KeyframeWalkStats) compter(dec kfDecision, glissements int) {
	s.Glissements += glissements
	switch dec {
	case kfVoisin:
		s.Voisins++
	case kfRecalage:
		s.Recalages++
	case kfElection:
		s.Elections++
	}
}

// WalkKeyframeWorld porte la logique durcie de walkOffline (cmd/tmp_kfworldpos) : il parcourt
// la table keyframe type-2 (payload de frame `buf`) et retourne les records slot->typeIndex
// reconstruits. Depuis le lot M3.1 (2026-09-23) le choix de l'ancre suivante passe par
// [kfScanNext] (voisin, recalage, élection) et une fenêtre vide ne coupe plus la table.
func WalkKeyframeWorld(buf []byte) []KeyframeRec {
	return walkKeyframeWorldFenetre(buf, kfScanFenetreBits)
}

// WalkKeyframeWorldStats est [WalkKeyframeWorld] qui rend AUSSI ses décisions : c'est la forme
// que le balayage des armes portées emploie, pour publier la santé de la marche dans le document.
func WalkKeyframeWorldStats(buf []byte) ([]KeyframeRec, KeyframeWalkStats) {
	return walkKeyframeWorldStats(buf, kfScanFenetreBits)
}

// kfScanFenetreBits est la PORTÉE DE L'ÉLECTION du balayeur. ELLE N'EXISTE PAS DANS LE JEU :
// `FUN_142e2bfd0` ne balaie rien, il enchaîne les entrées. C'est une invention du port.
//
// DEPUIS LE LOT M3.1 (2026-09-23) ELLE NE COUPE PLUS LA TABLE : une fenêtre sans candidat glisse
// ([kfScanGlissant]). Elle ne borne plus que la région où l'élection cherche son meilleur
// candidat, et c'est une borne de COÛT, mesurée : sur quatre films (dad793c7, 81c02726,
// a0c36016, b1f01a33) la marche sans aucune fenêtre rend EXACTEMENT les mêmes ancres que la
// fenêtre glissante — 0 ajoutée, 0 perdue — pour un temps multiplié par 5 à 6 (a0c36016 :
// 0,98 s contre 4,74 s par passe, et la marche est rejouée par une quinzaine de balayages de
// cuisson). Le déraillement que le lot 5.20.1 avait mesuré sans fenêtre (dad793c7, chunks 2 à
// 5 : 187 -> 127 ancres) venait de l'élection « slot bas lointain » : le recalage le supprime
// (dad793c7 : 868 -> 902 ancres, 33 des 34 ajoutées confirmées par une autre image-clé).
//
// SON RETRAIT RESTE GAGÉ SUR LA MARCHE DÉTERMINISTE : `WalkKeyframeRecords` porte le cadre exact
// du jeu (108 bits d'en-tête, état complet sans masque) et n'a besoin d'aucune fenêtre. Le jour
// où la fermeture d'image-clé atteint 100 %, ce balayeur et cette constante disparaissent
// ensemble (critère mesurable : `KeyframeClosure` à 100 % sur les bobines par build ; suivi :
// `keyframe_closure.golden`).
const kfScanFenetreBits = 120000

// walkKeyframeWorldFenetre est [WalkKeyframeWorld] avec une FENÊTRE explicite : `maxWin <= 0` =
// une seule fenêtre, le payload entier. Le paramètre n'existe que pour MESURER ce que la fenêtre
// change (lots 5.20.1 et M3.1) ; la production passe par [WalkKeyframeWorld].
func walkKeyframeWorldFenetre(buf []byte, maxWin int) []KeyframeRec {
	recs, _ := walkKeyframeWorldStats(buf, maxWin)
	return recs
}

// walkKeyframeWorldStats est le corps du balayeur.
func walkKeyframeWorldStats(buf []byte, maxWin int) ([]KeyframeRec, KeyframeWalkStats) {
	st := KeyframeWalkStats{Payloads: 1}
	total := len(buf) * 8
	if maxWin <= 0 {
		maxWin = total
	}
	width := map[int]int{}
	seen := map[int]int{}
	var out []KeyframeRec
	pos := 1 // préfixe 1 bit
	prev := -1
	if _, _, _, ok := kfValidAnchor(buf, pos, prev, total); !ok {
		var dec kfDecision
		var g int
		pos, dec, g = kfScanGlissant(buf, pos, prev, total, maxWin)
		st.compter(dec, g)
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(buf, pos, prev, total)
		if !ok {
			break
		}
		startState := pos + 64
		nat := -1
		// fast-path saut-de-largeur : n'accepte QUE gen==1 (non ambigu). Les faux ancres
		// gen 2/3 de la zone d'état peuvent tomber pile sur startState+w ; on ne saute pas
		// dessus. Un vrai record gen≥2 (mid-match) est résolu par kfScanGlissant (fallback).
		if w, has := width[ti]; has {
			if _, _, jg, vok := kfValidAnchor(buf, startState+w, slot, total); vok && jg == 1 {
				nat = startState + w
				st.Sauts++
			}
		}
		if nat < 0 {
			var dec kfDecision
			var g int
			nat, dec, g = kfScanGlissant(buf, startState, slot, total, maxWin)
			st.compter(dec, g)
		}
		out = append(out, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos})
		if ti == BipedTypeIndex {
			st.Bipedes++
		}
		prev = slot
		if nat < 0 {
			break
		}
		kfApprendreLargeur(width, seen, ti, nat-startState)
		pos = nat
	}
	st.Records = len(out)
	return out, st
}

// kfApprendreLargeur retient la largeur d'un archétype quand deux records consécutifs de cet
// archétype l'ont donnée identique, et l'oublie dès qu'elle varie.
func kfApprendreLargeur(width, seen map[int]int, ti, w int) {
	if prevW, s := seen[ti]; s {
		if prevW == w {
			width[ti] = w
		} else {
			delete(width, ti)
		}
		return
	}
	seen[ti] = w
}

// WorldFromKeyframe construit un World OFFLINE en bindant chaque record keyframe reconstruit.
// pay = payload de frame type-2 du chunk (extrait comme cmd/tmp_kfworldpos le fait). Aucun
// input Cheat Engine : la source de vérité est uniquement le film.
func WorldFromKeyframe(reg *Registry, pay []byte) *World {
	w := NewWorld(reg)
	for _, r := range WalkKeyframeWorld(pay) {
		w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
	}
	return w
}
