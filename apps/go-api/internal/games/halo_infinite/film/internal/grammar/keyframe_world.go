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

// kfRecherche porte la recherche d'ancres d'UN payload : ce que ses fenêtres partagent. La
// PREUVE (nil : l'élection sans réfutation d'avant le lot D-fix) et la mémoire des preuves déjà
// calculées vivent le temps du payload ; les candidats, le temps d'une fenêtre.
type kfRecherche struct {
	buf    []byte
	total  int
	maxWin int
	preuve *PreuveDImageCle
	// cands : les candidats de la DERNIÈRE fenêtre balayée, dans l'ordre des bits.
	cands []kfCandidat
	// prouves : [PreuveDImageCle.prouve] par position, pour tout le payload.
	prouves map[int]bool
}

// kfIssue est ce qu'une recherche d'ancre rend.
type kfIssue struct {
	// at : la position en bits de l'ancre retenue, -1 si aucune.
	at int
	// dec : la décision qui l'a retenue.
	dec kfDecision
	// fin : la fenêtre a rencontré la fin de table (2 048 identifiants sentinelles d'affilée).
	fin bool
	// traine : la traînée de sentinelles sur laquelle la fenêtre FINIT sans candidat —
	// [kfRecherche.glissante] reprend la fenêtre suivante à son début, pour qu'une fin de table à
	// cheval sur deux fenêtres se reconnaisse (constat F5 de la revue adverse du lot M3.1).
	traine int
	// refutes : les élus refusés parce qu'un record prouvé les contredit.
	refutes int
	// glissements : les fenêtres VIDES franchies pour atteindre l'ancre.
	glissements int
}

// suivante : la prochaine ancre après `from` dans une fenêtre de `maxWin` bits.
//
// TROIS DÉCISIONS, DANS CET ORDRE (lot M3.1, 2026-09-23) :
//
//  1. VOISIN : slot==prev+1 ET gen==1, n'importe où dans la fenêtre — retour immédiat, non
//     ambigu, inchangé.
//  2. RECALAGE : le PREMIER en-tête exact de bipède (gen==1, mot d'archétype == 35) quand il
//     précède l'ancre que l'élection retiendrait, ou qu'il EST cette ancre. Recensement exhaustif
//     des payloads d'image-clé de quatre films (dad793c7, 81c02726, a0c36016, b1f01a33) : 649
//     en-têtes de cette forme, ZÉRO faux — les 14 faux d'a0c36016 portent tous une génération 2
//     ou 3 et un mot de taille `n1` aberrant. Une ancre élue PLUS LOIN et de slot plus bas que ce
//     bipède est nécessairement fausse : la table est à slots croissants.
//  3. ÉLECTION : le repli ([kfCand.betterThan]) — DEPUIS LE LOT D-fix (2026-09-24), l'élu qu'un
//     record PROUVÉ contredit est refusé et l'élection reprend parmi les candidats restants
//     (keyframe_world_preuve.go). Le recalage se juge contre l'élu RETENU.
//
// LA RÈGLE « GÉNÉRATION 1 PUIS LE PLUS PROCHE » (V2 de la sonde P2) A ÉTÉ MESURÉE ET ÉCARTÉE :
// sur les quatre films elle rend les bipèdes, mais PERD les autres ancres par milliers
// (a0c36016 : 14 659 -> 11 890 ; b1f01a33 : 6 719 -> 5 524, trois bipèdes perdus), parce qu'une
// fausse ancre de génération 1 plus proche que le vrai record suivant se trouve presque partout
// hors de la bande des bipèdes. Le recalage ne change l'élection QUE devant un en-tête exact.
func (r *kfRecherche) suivante(from, prevSlot int) kfIssue {
	iss := kfIssue{at: -1}
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	exact := -1
	end := min(from+r.maxWin, r.total)
	r.cands = r.cands[:0]
	sentStreak := 0
	for q := from; q < end && q+64 <= r.total; q++ {
		id := kfReadBits(r.buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				iss.fin = true
				break
			}
			continue
		}
		sentStreak = 0
		s, ti, g, ok := kfAnchorFromID(r.buf, q, id, prevSlot, r.total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return kfIssue{at: q, dec: kfVoisin} // consécutif gen-1 : non ambigu
		}
		if exact < 0 && g == 1 && ti == BipedTypeIndex {
			exact = q
		}
		r.cands = append(r.cands, kfCandidat{gen: g, slot: s, bit: q, ti: ti})
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if iss.at < 0 || cand.betterThan(best) {
			iss.at, best = q, cand
		}
	}
	if iss.at >= 0 && r.preuve != nil {
		if elu, refutes := r.elire(prevSlot); elu >= 0 {
			iss.at, iss.refutes = elu, refutes
		}
	}
	switch {
	case exact >= 0 && (iss.at < 0 || exact <= iss.at):
		// `<=` : quand l'élection retient elle-même l'en-tête exact, c'est lui qui la justifie —
		// la décision se compte en recalage, et `Elections` ne compte que les choix du repli.
		iss.at, iss.dec = exact, kfRecalage
	case iss.at >= 0:
		iss.dec = kfElection // repli nomme `repli_ancre_d_image_cle_par_election`
	default:
		iss.traine = sentStreak
	}
	return iss
}

// glissante est [kfRecherche.suivante] dont une fenêtre SANS AUCUN candidat n'arrête plus la
// marche : la recherche GLISSE de fenêtre en fenêtre jusqu'au premier candidat, à la fin de table
// ou à la fin du payload.
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
func (r *kfRecherche) glissante(from, prevSlot int) kfIssue {
	vides := 0
	for f := from; f+64 <= r.total; {
		iss := r.suivante(f, prevSlot)
		if iss.at >= 0 {
			iss.glissements = vides
			return iss
		}
		if iss.fin {
			break
		}
		vides++
		f += r.maxWin - iss.traine
	}
	r.cands = r.cands[:0]
	return kfIssue{at: -1, dec: kfAucune}
}

// ecartes rend les candidats de la dernière fenêtre que l'ancre retenue à `at` (slot `slot`) rend
// INATTEIGNABLES : ceux qui la précèdent, et ceux qui la suivent avec un slot inférieur ou égal
// (la marche repart d'elle, à slots croissants).
func (r *kfRecherche) ecartes(at, slot int) []KeyframeRec {
	var out []KeyframeRec
	for _, c := range r.cands {
		if c.bit != at && (c.bit < at || c.slot <= slot) {
			out = append(out, KeyframeRec{Slot: c.slot, TI: c.ti, Gen: c.gen, Bit: c.bit})
		}
	}
	return out
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
	// Refutations : élus que le repli a REFUSÉS parce qu'un record prouvé par la grammaire du film
	// les contredisait (lot D-fix, keyframe_world_preuve.go) ; l'élection a repris sans eux.
	Refutations int
	// Glissements : fenêtres VIDES traversées sans arrêter la marche (cf. [kfRecherche.glissante]).
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
	s.Refutations += o.Refutations
	s.Glissements += o.Glissements
}

// compter note une recherche d'ancre.
func (s *KeyframeWalkStats) compter(iss kfIssue) {
	s.Glissements += iss.glissements
	s.Refutations += iss.refutes
	switch iss.dec {
	case kfVoisin:
		s.Voisins++
	case kfRecalage:
		s.Recalages++
	case kfElection:
		s.Elections++
	}
}

// kfScanFenetreBits est la PORTÉE DE L'ÉLECTION du balayeur. ELLE N'EXISTE PAS DANS LE JEU :
// `FUN_142e2bfd0` ne balaie rien, il enchaîne les entrées. C'est une invention du port.
//
// DEPUIS LE LOT M3.1 (2026-09-23) ELLE NE COUPE PLUS LA TABLE : une fenêtre sans candidat glisse
// ([kfRecherche.glissante]). Elle ne borne plus que la région où l'élection cherche son meilleur
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
