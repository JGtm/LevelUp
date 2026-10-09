package grammar

// keyframe_world_marche.go — LA MARCHE D'UN PAYLOAD D'IMAGE-CLÉ, avec ou sans PREUVE (lot D-fix
// de la campagne « retours rejeu », 2026-09-24). Séparée de keyframe_world.go par la TAILLE (le
// seuil de 500 lignes du dépôt) : la recherche d'ancre (voisin, recalage, élection) y reste ; ce
// fichier porte le corps de la marche, ce qu'elle rend, et ses deux entrées — celle d'un FILM
// ([FilmContext.MarcheDImageCle], qui refuse l'élu qu'un record prouvé contredit) et celle des
// INSTRUMENTS ([WalkKeyframeWorld], sans preuve).

import (
	"slices"
	"sync"
)

// MarcheDePayload est ce que la marche d'ancres rend pour UN payload d'image-clé.
type MarcheDePayload struct {
	// Records : les ancres, dans l'ordre de la marche (bits et slots croissants).
	Records []KeyframeRec
	// Ecartes : les candidats que le REPLI (recalage, élection) a rendus inatteignables — ceux
	// qui précédaient l'ancre retenue, ou la suivaient avec un slot inférieur ou égal. Un vrai
	// record que le repli a perdu est l'un d'eux : la liaison des images-clés ne l'oublie pas
	// (`keyframe_liaison.go`). L'absence d'un JOUEUR, elle, ne se juge pas sur cette liste — la
	// marche perd aussi par d'autres chemins — mais sur l'en-tête exact cherché dans tout le payload
	// (`player_entities_entetes.go`, lot D-fix, 2026-09-24).
	Ecartes []KeyframeRec
	// Stats : les décisions de la marche.
	Stats KeyframeWalkStats
}

// MarcheDImageCle est la marche d'ancres d'UN film ([FilmContext.MarcheDImageCle]) : elle refuse
// l'élu qu'un record prouvé par la grammaire du film contredit. Sa valeur zéro marche SANS preuve.
type MarcheDImageCle struct {
	preuve *PreuveDImageCle
	// memoire : les marches deja faites de CE film ([FilmContext.MarcheDImageCle]) ; nil pour la
	// valeur zero, qui ne se souvient de rien.
	memoire *memoireDesMarches
}

// memoireDesMarches garde, pour UN film, la marche de chaque payload d image-cle deja marche.
//
// POURQUOI : une quinzaine de balayages d un meme film marchent les MEMES payloads (table
// anticipee, liaison au monde, generations vivantes, equipes, fermeture...) ; la marche est une
// fonction pure des octets du payload et de la preuve du film, tous deux fixes pour un contexte.
// Elle est donc faite UNE fois par payload et par contexte (2026-09-28, temps CI du paquet).
//
// LA CLE EST L IDENTITE DU PAYLOAD (adresse de son premier octet, longueur) : un payload est une
// sous-tranche du chunk du film ([source.Film]), jamais une copie, et la cle retient le tableau
// vivant — une adresse ne peut pas etre reutilisee par un autre tampon tant qu elle est gardee.
// Un tampon qui n est pas celui du film (payload tronque, fabrique) a une autre cle. Le memo vit
// et meurt avec son [FilmContext] : il ne traverse jamais deux films. Chaque lecture rend une
// COPIE (un appelant trie ses records en place, cf. [keyframeBornesDe]).
//
// PRECONDITION, deja celle du contexte : les octets du film ne changent pas sous un contexte qui
// les a lus (les tests qui reecrivent un film en memoire construisent un contexte neuf).
type memoireDesMarches struct {
	mu         sync.Mutex
	preuve     *PreuveDImageCle
	parPayload map[clePayload]MarcheDePayload
}

// clePayload : l identite d un payload (cf. [memoireDesMarches]).
type clePayload struct {
	debut *byte
	n     int
}

// copie rend une marche dont les tranches ne sont partagees avec personne.
func (mp MarcheDePayload) copie() MarcheDePayload {
	return MarcheDePayload{Records: slices.Clone(mp.Records), Ecartes: slices.Clone(mp.Ecartes), Stats: mp.Stats}
}

// Records rend les records d'un payload d'image-clé.
func (m MarcheDImageCle) Records(pay []byte) []KeyframeRec {
	return m.Marcher(pay).Records
}

// RecordsStats rend les records d'un payload d'image-clé et les décisions de la marche.
func (m MarcheDImageCle) RecordsStats(pay []byte) ([]KeyframeRec, KeyframeWalkStats) {
	mp := m.Marcher(pay)
	return mp.Records, mp.Stats
}

// Marcher rend tout ce que la marche d'un payload d'image-clé a lu et décidé.
func (m MarcheDImageCle) Marcher(pay []byte) MarcheDePayload {
	mem := m.memoire
	if len(pay) == 0 {
		mem = nil
	}
	var cle clePayload
	if mem != nil {
		cle = clePayload{debut: &pay[0], n: len(pay)}
		mem.mu.Lock()
		mp, ok := mem.parPayload[cle]
		mem.mu.Unlock()
		if ok {
			return mp.copie()
		}
	}
	mp := marcherLaTable(&kfRecherche{buf: pay, total: len(pay) * 8, maxWin: kfScanFenetreBits,
		preuve: m.preuve})
	if mem != nil {
		mem.mu.Lock()
		mem.parPayload[cle] = mp
		mem.mu.Unlock()
		return mp.copie()
	}
	return mp
}

// WalkKeyframeWorld porte la logique durcie de walkOffline (cmd/tmp_kfworldpos) : il parcourt
// la table keyframe type-2 (payload de frame `buf`) et retourne les records slot->typeIndex
// reconstruits. Depuis le lot M3.1 (2026-09-23) le choix de l'ancre suivante passe par
// [kfRecherche.suivante] (voisin, recalage, élection) et une fenêtre vide ne coupe plus la table.
//
// SANS PREUVE : c'est la marche des INSTRUMENTS. Un balayage de production marche par le
// contexte de son film ([FilmContext.MarcheDImageCle]), qui refuse l'élu contredit par un record
// prouvé (garde-rail `archlint/keyframe_walk_proof_test.go`).
func WalkKeyframeWorld(buf []byte) []KeyframeRec {
	return walkKeyframeWorldFenetre(buf, kfScanFenetreBits)
}

// WalkKeyframeWorldStats est [WalkKeyframeWorld] qui rend AUSSI ses décisions, SANS preuve.
func WalkKeyframeWorldStats(buf []byte) ([]KeyframeRec, KeyframeWalkStats) {
	return walkKeyframeWorldStats(buf, kfScanFenetreBits)
}

// walkKeyframeWorldFenetre est [WalkKeyframeWorld] avec une FENÊTRE explicite : `maxWin <= 0` =
// une seule fenêtre, le payload entier. Le paramètre n'existe que pour MESURER ce que la fenêtre
// change (lots 5.20.1 et M3.1) ; la production passe par [MarcheDImageCle].
func walkKeyframeWorldFenetre(buf []byte, maxWin int) []KeyframeRec {
	recs, _ := walkKeyframeWorldStats(buf, maxWin)
	return recs
}

// walkKeyframeWorldStats est [walkKeyframeWorldFenetre] qui rend aussi ses décisions (sans preuve).
func walkKeyframeWorldStats(buf []byte, maxWin int) ([]KeyframeRec, KeyframeWalkStats) {
	total := len(buf) * 8
	if maxWin <= 0 {
		maxWin = total
	}
	mp := marcherLaTable(&kfRecherche{buf: buf, total: total, maxWin: maxWin})
	return mp.Records, mp.Stats
}

// marcherLaTable est le corps du balayeur.
func marcherLaTable(r *kfRecherche) MarcheDePayload {
	mp := MarcheDePayload{Stats: KeyframeWalkStats{Payloads: 1}}
	width := map[int]int{}
	seen := map[int]int{}
	pos := 1 // préfixe 1 bit
	prev := -1
	// elue : l ancre `pos` a ete choisie par l election (le repli nomme) ; chaque record le porte.
	elue := false
	if _, _, _, ok := kfValidAnchor(r.buf, pos, prev, r.total); !ok {
		iss := r.glissante(pos, prev)
		mp.Stats.compter(iss)
		mp.noterEcartes(r, iss, prev)
		pos, elue = iss.at, iss.dec == kfElection
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(r.buf, pos, prev, r.total)
		if !ok {
			break
		}
		startState := pos + 64
		nat, suivanteElue := -1, false
		// fast-path saut-de-largeur : n'accepte QUE gen==1 (non ambigu). Les faux ancres
		// gen 2/3 de la zone d'état peuvent tomber pile sur startState+w ; on ne saute pas
		// dessus. Un vrai record gen≥2 (mid-match) est résolu par la recherche glissante.
		if w, has := width[ti]; has {
			if _, _, jg, vok := kfValidAnchor(r.buf, startState+w, slot, r.total); vok && jg == 1 {
				nat = startState + w
				mp.Stats.Sauts++
			}
		}
		if nat < 0 {
			iss := r.glissante(startState, slot)
			mp.Stats.compter(iss)
			mp.noterEcartes(r, iss, slot)
			nat, suivanteElue = iss.at, iss.dec == kfElection
		}
		mp.Records = append(mp.Records, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos, Elue: elue})
		if ti == BipedTypeIndex {
			mp.Stats.Bipedes++
		}
		prev = slot
		if nat < 0 {
			break
		}
		kfApprendreLargeur(width, seen, ti, nat-startState)
		pos, elue = nat, suivanteElue
	}
	mp.Stats.Records = len(mp.Records)
	return mp
}

// noterEcartes retient les candidats qu'une décision de REPLI (recalage, élection) a rendus
// inatteignables. Un voisin n'en écarte aucun qui puisse être un record : entre le record
// courant et son voisin de slot suivant, la table n'a pas de place.
func (mp *MarcheDePayload) noterEcartes(r *kfRecherche, iss kfIssue, prevSlot int) {
	if iss.at < 0 || (iss.dec != kfRecalage && iss.dec != kfElection) {
		return
	}
	slot, _, _, ok := kfValidAnchor(r.buf, iss.at, prevSlot, r.total)
	if !ok {
		return
	}
	mp.Ecartes = append(mp.Ecartes, r.ecartes(iss.at, slot)...)
}
