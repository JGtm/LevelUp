package grammar

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
//  2. DÉBUT DE LA VUE B. Dans un paquet porteur d'une liste d'événements, la boucle de records ne
//     commence pas à l'amorce : elle commence à la fin de la vue A quand la lecture de la vue A en
//     décide, sinon au début que rend le localisateur unique ([DebutDeLaVueB], `localisateur.go`,
//     ordre [SignaturePuisLargeurLibre]).
//  3. HUIT VUES DE RÉPLICATION par paquet : une mort peut vivre dans une vue > 0.
//  4. SNAPSHOT / RESTORE : une marche qui a désynchronisé ne laisse aucune liaison derrière
//     elle (les deux politiques qui les conservaient ont été mesurées perdantes).
//
// CE FICHIER NE DÉCIDE RIEN DU SENS DES RECORDS : il rend des `FrameRecord`. La lecture d'un
// fait (une mort d'objet) vit dans `object_deaths.go`.
//
// LA BOUCLE DE RECORDS A UNE JUMELLE, `film/facts/killsource/walk.go` (`walkFrom`, qui ne pose
// pas le cadre du lecteur) ; le localisateur, lui, est commun aux deux marches.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// marchViews est le nombre de VUES de réplication déroulées par paquet. Huit, valeur de
// `killsource` : une mort peut vivre dans une vue > 0, et la mesure V13 a été conduite sous
// cette valeur.
const marchViews = 8

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
	slices.SortStableFunc(tl.events, func(a, b marchKeyframe) int { return cmp.Compare(a.timestampUS, b.timestampUS) })
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
func marchHasEvents(pay []byte) bool { return source.BitAt(pay, 1) != 0 }

// marchRecordsOf déroule la boucle de records d'UN paquet depuis le bit `start`, jusqu'à
// `marchViews` vues, et RESTAURE le monde : une marche désynchronisée ne laisse pas de liaison
// derrière elle. Les records déjà lus quand la chaîne casse sont rendus — c'est au lecteur de
// fait de trier, pas à la marche.
func marchRecordsOf(pay []byte, w *World, cfg FrameConfig, start int) []FrameRecord {
	snap := w.Snapshot()
	br := LecteurSur(pay)
	br.poserCadre(cfg)
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

// marchStartOf rend le bit de départ de la boucle de records d'un paquet PAR LE SEUL LOCALISATEUR,
// et dit si le paquet portait une liste d'événements et s'il a été localisé. `ok` faux = paquet à
// événements non localisé : aucun point de départ sûr, il se saute (jamais une marche au hasard).
//
// C'est la forme de la CALIBRATION DU CADRE ([trialFrameConfig]), qui juge chaque largeur
// d'identifiant par le taux de paquets que la signature localise : la signature dépend de cette
// largeur, la fin de la vue A n'en dépend pas — elle localiserait les mêmes paquets sous toutes les
// largeurs et ne départagerait rien. La marche des morts, elle, part de la fin de la vue A quand
// elle décide ([marchDebut]).
func marchStartOf(pay []byte, w *World, cfg FrameConfig) (start int, withEvents, ok bool) {
	start, withEvents, ok, _ = marchDebut(pay, w, cfg, VueADuFilm{})
	return start, withEvents, ok
}

// marchDebut rend le bit de depart de la boucle de records d un paquet sous la grammaire de vue A
// du film `v` : la fin de la vue A quand elle decide, sinon le localisateur ([DebutDeLaVueB]) ; plus
// le verdict du repli `repli_localisation_largeur_libre` : le paquet a ete localise par la seconde
// passe a LARGEUR LIBRE. [ScanMarchFacts] le compte au rapport du contexte (lot J8.7) ; la
// calibration du cadre, qui ESSAIE des largeurs, ne compte rien.
func marchDebut(pay []byte, w *World, cfg FrameConfig, v VueADuFilm) (start int, withEvents, ok, aLargeurLibre bool) {
	if !marchHasEvents(pay) {
		return cfg.PacketPreambleBits, false, true, false
	}
	s, aLargeurLibre := DebutDeLaVueB(pay, w, cfg, v)
	if s < 0 {
		return 0, true, false, false
	}
	return s, true, true, aLargeurLibre
}
