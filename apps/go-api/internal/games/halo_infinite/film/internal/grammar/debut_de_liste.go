package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// debut_de_liste.go — LA LISTE D UN PAQUET A EVENEMENTS COMMENCE A SES RECORDS NEW DE TETE (lot
// M4b de la campagne « retours rejeu », 2026-09-25).
//
// # LE DEFAUT, MESURE
//
// Dans un paquet delta a liste d evenements, le localisateur de production ([LocaliserBoucleDeRecords],
// signature du slot 123) demarre la marche sur le premier record qu il sait ancrer. Or les
// CREATIONS d objets de l instant sont les premiers records de la liste — la naissance d un
// bipede (lot M3.2 : 40/41, 91/99 et 123/125 cas, `birth_loadouts.go`), les armes et
// l equipement laches a sa mort, les projectiles : le localisateur les SAUTE. Le monde ne lie
// donc jamais l objet ne en milieu de chunk qui disparait avant l image-cle suivante (la table
// anticipee du lot 5.23 ne le voit pas), et CHAQUE delta suivant de cet objet clot la vue B sur
// le rejet de son en-tete : la vue C — le tir continu — n est plus lue. Mesure sur `81c02726` :
// le bipede 521 (index 4, ne a la trame 495) rejette 438 paquets entre 495 et 575, puis les armes
// et l equipement qu il lache (1748, 1750) ceux de 575 a 658 — le trou de lecture 500-658 du gate
// G1, deux frags au Ghost sans rafale.
//
// # LA LECTURE, ET SA PREUVE
//
// Un CANDIDAT est une position qui porte un en-tete de record NEW (prefixe `0` puis `01`) dont le
// slot est dans la BANDE de l archetype annonce, lue dans les images-cles du film par la regle des
// objets du monde ([TableAnticipee.SlotDeLArchetype]). Un candidat n est qu un candidat. Il n est
// retenu que si la CHAINE des records qui en partent — chacun lu par son en-tete comme la boucle
// de records le lit : NEW traverse sans desynchronisation, DEL, delta qui se decode sur un slot que
// le monde connait, aucun masque que l ecrivain n ecrit pas ([pasDEssai]) — finit EXACTEMENT sur
// le debut que le localisateur a trouve : deux lectures independantes, l en-tete en tete et la
// signature du slot 123 en queue, qui s accordent au bit pres. Quand le localisateur ne trouve
// rien, la preuve est la FERMETURE du paquet par la marche complete qui part du candidat
// ([debutParFermetureRangee]) ; a defaut, le candidat d ou le paquet ferme au bit pres seulement
// est garde, et ce second rang N EST PAS une preuve : c est le repli nomme
// `repli_debut_de_liste_ferme_au_bit`, compte a part. Sinon le debut du localisateur est garde, et
// rien ne change d un bit. Par la chaine et au premier rang, aucun bit n est devine, les
// records sont LUS ; les listes ainsi etendues sont comptees
// ([types.MovementStateStats.EventPacketsNewRecordStart]).

// localiserLaListe rend le debut de la marche d un paquet a evenements dont la vue A n a pas decide
// ([debutDeLaVueBDeCuisson]) et COMMENT il a ete trouve : le premier record NEW de tete prouve par
// la chaine ([debutParChaine]) ou par la fermeture
// ([debutParFermetureRangee], aux deux rangs), sinon le debut du localisateur strict
// ([lecture.DebutParSignature]) ; -1 et [lecture.DebutNonLocalise] pour une liste non localisee.
// Tout debut autre que celui du localisateur est un record NEW que le localisateur sautait. Chaque
// comment est une recuperation que la structure de lecture marque (ADR 0037 IR-6).
func localiserLaListe(pay []byte, w *World, cfg FrameConfig) (int, lecture.DebutDeVueB) {
	debut, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignatureStricte)
	if debut < 0 {
		return debutParFermetureRangee(pay, candidatsDeTete(pay, len(pay)*8, w), w, cfg)
	}
	if d, parNeuf := debutParChaine(pay, debut, candidatsDeTete(pay, debut, w), w, cfg); parNeuf {
		return d, lecture.DebutParChaine
	}
	return debut, lecture.DebutParSignature
}

// candidatsDeTete rend, tries, les positions `p` (en-tete complet avant `fin`) qui portent un
// en-tete de record NEW dont le slot est dans la bande de l archetype annonce.
func candidatsDeTete(pay []byte, fin int, w *World) []int {
	var out []int
	for p := 0; p+woNewHeaderBits <= fin; p++ {
		slot, ti, ok := enteteNeufEn(pay, p)
		if ok && w.anticipee.SlotDeLArchetype(slot, ti) {
			out = append(out, p)
		}
	}
	return out
}

// enteteNeufEn lit, sans lecteur, l en-tete d un record NEW a `p` : prefixe `0` puis `01`, slot,
// generation, archetype borne par le registre des objets ([objectArchetypeCount]).
func enteteNeufEn(pay []byte, p int) (slot, ti uint32, ok bool) {
	if source.BitsTolerants(pay, p, 1) != 0 || source.BitsTolerants(pay, p+1, 2) != 1 {
		return 0, 0, false
	}
	ti = uint32(source.BitsTolerants(pay, p+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)) //nolint:gosec // 6 bits
	if ti >= objectArchetypeCount {
		return 0, 0, false
	}
	return LireHandle(pay, p+woNewTypeBits).Slot, ti, true
}

// motFacultatifDEnTete rend la largeur du mot facultatif qui precede le type de chaque record
// (`FUN_14080a9d4` : `HasExtraFields`), 0 sans lui.
func motFacultatifDEnTete(cfg FrameConfig) int {
	if cfg.HasExtraFields {
		return 32
	}
	return 0
}

// debutParChaine rend le premier candidat `p` < `debut` depuis lequel la chaine des records finit
// EXACTEMENT sur `debut` ; `false` : aucune chaine ne tient, le debut du localisateur est garde.
func debutParChaine(pay []byte, debut int, candidats []int, w *World, cfg FrameConfig) (int, bool) {
	extra := motFacultatifDEnTete(cfg)
	for _, p := range candidats {
		if p-extra < 0 || p >= debut {
			continue
		}
		if chaineJusqua(pay, p-extra, debut, extra, w, cfg) {
			return p - extra, true
		}
	}
	return debut, false
}

// plafondChaineDeTete borne la chaine d essai : ce n est pas une largeur de grammaire, c est la
// garde d une boucle hors ligne. Les chaines mesurees comptent de trois a quatorze records
// (`81c02726`, `8a485699`).
const plafondChaineDeTete = 64

// chaineJusqua marche les records qui partent de `pos`, chacun lu par son en-tete comme la boucle
// de records le lit ([pasDEssai]). Vrai si la chaine tombe EXACTEMENT sur `debut` ET si ses
// records, puis celui de `debut`, suivent la loi d ecriture de la vue B ([ordreDeLaVueB] : NEW*,
// DELTA*, DEL*, slots strictement croissants dans chaque groupe) : une chaine que l ecrivain ne
// peut pas ecrire ne prouve rien, meme si elle tombe au bit pres.
func chaineJusqua(pay []byte, pos, debut, extra int, w *World, cfg FrameConfig) bool {
	essai := cfg
	essai.Obs = nil // traversees d ESSAI : rien n est publie, la marche relira les records
	ordre := nouvelOrdreDeLaVueB()
	for n := 0; n < plafondChaineDeTete && pos < debut; n++ {
		if !suitLOrdreEn(&ordre, pay, pos, extra, essai) {
			return false
		}
		fin, ok := pasDEssai(pay, pos, extra, w, essai)
		if !ok || fin <= pos {
			return false
		}
		pos = fin
	}
	return pos == debut && suitLOrdreEn(&ordre, pay, debut, extra, essai)
}

// suitLOrdreEn lit, sans le traverser, l en-tete du record a `pos` (type, puis identifiant comme
// [pasDEssai] le lit) et le fait suivre par `ordre` ; faux pour un en-tete hors de l ordre, ou
// pour un terminateur (aucun genre).
func suitLOrdreEn(ordre *ordreDeLaVueB, pay []byte, pos, extra int, essai FrameConfig) bool {
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(pos + extra)
	ph, ok := phaseDeRecord(readRecordType(br))
	if !ok {
		return false
	}
	return ordre.suivre(ph, readRecordID(br, essai.IDLowBits, essai.IDBase)&0x3fffffff)
}

// pasDEssai lit UN record a `pos` et rend la position qui le suit : un NEW traverse sans
// desynchronisation, un DEL (en-tete et mot de 32 bits), un delta qui se decode sur un slot que le
// monde connait. Un terminateur ou un record illisible refusent le pas.
//
// UN RECORD DONT LE MASQUE CONTREDIT L ECRIVAIN REFUSE AUSSI LE PAS. `FUN_142e2da44`, le seul
// ecrivain du masque d un record NEW ou delta, ne pose aucun bit au-dela des composants de
// l archetype (`i < *(desc+0x4320)`), n ecrit dense qu un masque de plus de sept composants et
// ecrit croissants les index d un masque epars ([lireMasque], [EntityTrace.MasqueNonEcrit]). Un
// record lu dont le masque viole l une de ces regles n a pas ete ecrit la par le jeu : la chaine
// qui le traverse ne prouve rien, meme si elle tombe au bit pres sur le debut localise.
func pasDEssai(pay []byte, pos, extra int, w *World, essai FrameConfig) (int, bool) {
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(pos + extra)
	if br.Remaining() < woNewHeaderBits {
		return pos, false
	}
	switch readRecordType(br) {
	case recNew:
		readRecordID(br, essai.IDLowBits, essai.IDBase)
		tr := TraverseEntity(br, w.Reg, essai.NewDefaultStateBits)
		return tr.EndBit, tr.DesyncAt == -1 && tr.MasqueNonEcrit == InvariantAucun
	case recDel:
		readRecordID(br, essai.IDLowBits, essai.IDBase)
		br.Skip(32)
		return br.BitPos(), true
	case recDelta:
		rec, fin, ok := TryDeltaAt(pay, pos, w, essai)
		return fin, ok && rec.Trace.MasqueNonEcrit == InvariantAucun
	}
	return pos, false // terminateur : la liste finirait avant le debut localise
}

// debutParFermetureRangee rend, pour un paquet dont le localisateur ne trouve pas la liste, le
// PREMIER candidat d ou la marche complete (monde restaure apres chaque essai) FERME le paquet
// ([LectureVueC.Fermee] : au bit pres, sans regle de l ecrivain contredite) ; s il n y en a aucun,
// le premier d ou elle le ferme au bit pres ([LectureVueC.FermeeAuBit]) ; -1 sinon. Il rend aussi
// le RANG retenu : [lecture.DebutParFermeture], [lecture.DebutParFermetureAuBit], ou
// [lecture.DebutNonLocalise] avec -1.
//
// LE PREMIER RANG EST UNE PREUVE : le paquet ferme, aucune regle de l ecrivain contredite.
//
// LE SECOND RANG EST UN REPLI NOMME, `repli_debut_de_liste_ferme_au_bit` (lecture non portee : un
// tel debut porte un en-tete de record juste dont le corps est mal lu). Il garde la tete, pas la
// fermeture : le paquet lu depuis ce debut reste NON ferme (son verdict contredit l ecrivain, le
// tir continu y voit un trou), mais ses records sont lus et ses NEW lies, comme ceux de tout paquet
// lu et non ferme. Chaque liste prise a ce rang est comptee
// ([Observation.DebutsDeListeParRepliFermeAuBit]).
func debutParFermetureRangee(pay []byte, candidats []int, w *World, cfg FrameConfig) (int, lecture.DebutDeVueB) {
	extra := motFacultatifDEnTete(cfg)
	auBit := -1
	for _, p := range candidats {
		if p-extra < 0 {
			continue
		}
		l := lectureDEssai(pay, w, cfg, p-extra)
		if l.Fermee {
			return p - extra, lecture.DebutParFermeture
		}
		if l.FermeeAuBit && auBit < 0 {
			auBit = p - extra
		}
	}
	if auBit < 0 {
		return -1, lecture.DebutNonLocalise
	}
	cfg.Obs.compterDebutDeListeParRepliFermeAuBit()
	return auBit, lecture.DebutParFermetureAuBit
}

// lectureDEssai marche le paquet depuis `debut` sur le monde, puis le restaure, et rend le verdict
// de sa vue C.
func lectureDEssai(pay []byte, w *World, cfg FrameConfig, debut int) LectureVueC {
	essai := cfg
	obs := NouvelleObservation()
	var l LectureVueC
	obs.VueControleHook = func(x LectureVueC) { l = x }
	essai.Obs = obs
	snap := w.Snapshot()
	DecodeFrameViewsCurseur(pay, w, essai, MovementStateViews, debut)
	w.Restore(snap)
	return l
}
