package filmdec

// object_deaths_march.go — LA MARCHE : dérouler séquentiellement les records ECS des paquets
// delta d'un film, dans un monde tenu à jour par les images-clés.
//
// POURQUOI ELLE EXISTE À CÔTÉ DES BALAYAGES ANCRÉS. Les balayages de ce paquet
// (`ScanBipedPositionsForBand`, `ScanEquipmentState`, ...) cherchent un record par son ANCRE :
// un masque qui s'ouvre sur un `i0` absolu. Ils n'atteignent donc JAMAIS un composant tardif —
// le dead-state est à `i11`, et la mesure V10 l'a chiffré : sur six films, l'ancre accepte
// 153 535 à 240 115 records `ti=35` et en rend UN SEUL portant `i11`, là où la marche en rend
// 47 à 66 (`.ai/V7.5/film_re/NOTE_V13_DEADSTATE_VEHICULE_2026-09-05.md` § 2).
//
// QUATRE MÉCANISMES, tous mesurés, aucun réglable :
//
//  1. TIMELINE CHRONOLOGIQUE. Le monde est amorcé par la PREMIÈRE déclaration de chaque slot
//     (une entité née entre deux images-clés n'est déclarée que par la suivante), puis les
//     images-clés s'appliquent DANS L'ORDRE DU TEMPS — sans quoi un slot recyclé en cours de
//     match serait décodé au mauvais archétype sur toute la première moitié du film.
//  2. LOCALISATEUR D'ÉVÉNEMENTS. Dans un paquet porteur d'une liste d'événements, la boucle de
//     records ne commence pas à l'amorce. Signature structurelle : le premier record est un
//     delta du slot 123 de 35 bits EXACTEMENT, précédé d'un bit nul. Repli à largeur libre
//     quand la signature stricte échoue (elle est sur-contrainte : 35 est le cas MODAL).
//  3. HUIT VUES DE RÉPLICATION par paquet : une mort peut vivre dans une vue > 0.
//  4. SNAPSHOT / RESTORE : une marche qui a désynchronisé ne laisse aucune liaison derrière
//     elle (les deux politiques qui les conservaient ont été mesurées perdantes).
//
// CE FICHIER NE DÉCIDE RIEN DU SENS DES RECORDS : il rend des `FrameRecord`. La lecture d'un
// fait (une mort d'objet) vit dans `object_deaths.go`.
//
// UNE SECONDE COPIE DE CETTE MARCHE EXISTE DANS LE DÉPÔT, `film/killsource/walk.go` — elle y
// porte son filtre de crédibilité de roster, ses golden et ses ancres Theater, et son
// changement casserait des empreintes gelées. Les deux se rejoignent au pas 4 de M2 (« une
// seule porte aux octets ») ; la découverte est consignée au plan § 4 (D1 (1.9.10)).

import "sort"

// marchViews est le nombre de VUES de réplication déroulées par paquet. Huit, valeur de
// `killsource` : une mort peut vivre dans une vue > 0, et la mesure V13 a été conduite sous
// cette valeur.
const marchViews = 8

// marchSignatureSlot / marchSignatureBits sont la SIGNATURE du premier record d'un paquet à
// événements : un delta du slot 123, long de 35 bits exactement, à composant unique. Candidat
// UNIQUE et VRAI sur 690 paquets sur 690 confrontés à une vérité de position indépendante.
const (
	marchSignatureSlot = uint32(123)
	marchSignatureBits = 35
)

// marchKeyframe est une image-clé décodée, avec son horodatage.
type marchKeyframe struct {
	timestampUS uint64
	recs        []KeyframeRec
}

// marchDelta est un paquet delta avec son horodatage et son payload (une vue sur les octets du
// film déjà résidents : la retenir ne coûte rien).
type marchDelta struct {
	timestampUS uint64
	payload     []byte
}

// marchTimeline est la suite chronologique des images-clés, plus le monde qu'elles lient.
type marchTimeline struct {
	events []marchKeyframe
	cursor int
	w      *World
}

// newMarchTimeline amorce le monde par la PREMIÈRE déclaration de chaque slot, sur des
// images-clés triées par instant.
func newMarchTimeline(reg *Registry, kfs []marchKeyframe) *marchTimeline {
	tl := &marchTimeline{w: NewWorld(reg), events: kfs}
	sort.SliceStable(tl.events, func(i, j int) bool {
		return tl.events[i].timestampUS < tl.events[j].timestampUS
	})
	seen := map[int]bool{}
	for _, e := range tl.events {
		for _, r := range e.recs {
			if seen[r.Slot] {
				continue
			}
			seen[r.Slot] = true
			tl.w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
	}
	return tl
}

// advanceTo applique toutes les images-clés d'horodatage <= `at`. LES APPELS DOIVENT CROÎTRE :
// c'est un curseur, pas une recherche.
func (tl *marchTimeline) advanceTo(at uint64) *World {
	for tl.cursor < len(tl.events) && tl.events[tl.cursor].timestampUS <= at {
		for _, r := range tl.events[tl.cursor].recs {
			tl.w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
		}
		tl.cursor++
	}
	return tl.w
}

// marchHasEvents dit si le paquet porte une liste d'événements (bit 1 du payload).
func marchHasEvents(pay []byte) bool { return kfBitAt(pay, 1) != 0 }

// marchSignature123 : un delta du slot de signature décode-t-il en `s`, finit-il exactement
// `marchSignatureBits` plus loin, avec un composant unique ?
func marchSignature123(pay []byte, s int, w *World, cfg FrameConfig) bool {
	rec, end, ok := TryDeltaAt(pay, s, w, cfg)
	return ok && rec.Slot == marchSignatureSlot && end == s+marchSignatureBits &&
		len(rec.Trace.Comps) == 1
}

// marchLocateStrict rend la première position `s >= 2`, précédée d'un bit nul, qui porte la
// signature stricte ; -1 si aucune.
func marchLocateStrict(pay []byte, w *World, cfg FrameConfig) int {
	nb := len(pay) * 8
	for s := 2; s+marchSignatureBits < nb; s++ {
		if kfBitAt(pay, s-1) != 0 {
			continue
		}
		if marchSignature123(pay, s, w, cfg) {
			return s
		}
	}
	return -1
}

// marchLocateFallback reprend la même condition à LARGEUR LIBRE. Il n'est essayé qu'après
// l'échec de la signature stricte : les paquets déjà localisés ne bougent pas d'un bit.
func marchLocateFallback(pay []byte, w *World, cfg FrameConfig) int {
	nb := len(pay) * 8
	for s := 2; s+16 < nb; s++ {
		if kfBitAt(pay, s-1) != 0 {
			continue
		}
		rec, _, ok := TryDeltaAt(pay, s, w, cfg)
		if !ok || rec.Slot != marchSignatureSlot ||
			!w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte) {
			continue
		}
		return s
	}
	return -1
}

// marchLocate rend le bit de départ de la boucle de records d'un paquet à événements, ou -1.
//
// LE CONTRÔLE DE GÉNÉRATION EST INDISPENSABLE DES DEUX CÔTÉS : sans lui, le localisateur
// désigne des positions où le slot de signature porte une AUTRE génération, et la marche y
// meurt aussitôt (mesure : 3 morts perdues sur un film, dont un double kill).
func marchLocate(pay []byte, w *World, cfg FrameConfig) int {
	if s := marchLocateStrict(pay, w, cfg); s >= 0 {
		if rec, _, ok := TryDeltaAt(pay, s, w, cfg); ok &&
			w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte) {
			return s
		}
	}
	return marchLocateFallback(pay, w, cfg)
}

// marchRecordsOf déroule la boucle de records d'UN paquet depuis le bit `start`, jusqu'à
// `marchViews` vues, et RESTAURE le monde : une marche désynchronisée ne laisse pas de liaison
// derrière elle. Les records déjà lus quand la chaîne casse sont rendus — c'est au lecteur de
// fait de trier, pas à la marche.
func marchRecordsOf(pay []byte, w *World, cfg FrameConfig, start int) []FrameRecord {
	snap := w.Snapshot()
	br := NewBitReader(pay)
	br.PoserProfil(cfg.Profil)
	br.Skip(start)
	var recs []FrameRecord
	for v := 0; v < marchViews && br.Remaining() >= 8; v++ {
		more, err := DecodeFrameRecords(br, w, cfg)
		recs = append(recs, more...)
		if err != nil {
			break
		}
	}
	w.Restore(snap)
	return recs
}

// marchStartOf rend le bit de départ de la boucle de records d'un paquet, et dit si le paquet
// portait une liste d'événements et s'il a été localisé. `ok` faux = paquet à événements non
// localisé : aucun point de départ sûr, il se saute (jamais une marche au hasard).
func marchStartOf(pay []byte, w *World, cfg FrameConfig) (start int, withEvents, ok bool) {
	if !marchHasEvents(pay) {
		return cfg.PacketPreambleBits, false, true
	}
	s := marchLocate(pay, w, cfg)
	if s < 0 {
		return 0, true, false
	}
	return s, true, true
}
