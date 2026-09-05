package filmdec

// zoom_events.go — L'ÉTAT DE LUNETTE (ADS) LU DANS LE FILM.
//
// # CE QUE C'EST, ET POURQUOI C'EST ARRIVÉ SI TARD
//
// Un paquet delta ne porte pas qu'une trame d'état : il commence par un bit de configuration,
// puis une LISTE D'ÉVÉNEMENTS, et seulement ensuite les enregistrements d'entités :
//
//	[1 bit config] [ ( 1 [R(7) type] [3 références gardées] [charge] )* 0 ] [trame de records]
//
// Sept campagnes de mesure ont conclu « aucun événement de zoom dans la bobine » parce
// qu'elles lisaient le type d'événement à `payload[0] & 0x7F` — elles ignoraient le bit de
// configuration et décalaient donc TOUT d'un bit. Leurs « chaînes indépendantes » partageaient
// cette erreur. Le type 21 `unit_zoom` existe bel et bien : ~400 000 occurrences sur le corpus
// de 1 367 films, portées en tête des paquets dont le premier octet vaut 0xCA.
//
// # LA GRAMMAIRE
//
// Après le type viennent trois références gardées (domaines 4, 8 et 7 pour cet événement),
// chacune sous la forme `R(1) présence [ R(largeur) index + R(2) génération ]`, la largeur
// venant de la table des domaines. Puis la charge : `R(2)`, qui vaut le NIVEAU DE LUNETTE + 1
// — 0 pour « plus de lunette », 1 pour le premier palier. Le grossissement, lui, appartient à
// l'ARME et ne voyage pas ici : le film ne transmet que le palier.
//
// # LE PONT VERS LE JOUEUR — MESURÉ, PAS SUPPOSÉ
//
// La première référence désigne l'unité qui zoome, par un index de domaine 4. Cet index plus
// 512 donne le SLOT du bipède, celui-là même que porte [BipedPosition.Slot] :
//
//	base 512 : 63 index sur 64 tombent sur un slot bipède réellement vu dans le film (98 %)
//	bases 0, 256, 768, 1024 : 0 sur 64 (0 %)
//
// Ce n'est pas une corrélation, c'est une fermeture : une mauvaise base ne place presque aucun
// index sur un slot existant. Contrôle nommé : sur le film 00162144, l'index 1 donne le slot
// 513, celui que le pont des morts attribue à Nilton410 — et ses six entrées en lunette
// reproduisent la chronologie relevée à la main dans Theater (6/6 à moins de 1,2 s, contrôle
// par translation à 0,00 % sur ~3 200 décalages témoins).
//
// # RÉSERVE HONNÊTE
//
// Seuls les événements portés EN TÊTE d'un paquet 0xCA sont lus ici. Une liste peut en contenir
// plusieurs, et un `unit_zoom` en deuxième position échappe donc à ce scanner : les SORTIES de
// lunette sont sous-comptées (sur le film témoin, 10 entrées pour 5 sorties dans la fenêtre
// mesurée). Les ENTRÉES, elles, sont fidèles — c'est sur elles que la validation porte. Lire la
// liste entière exige de connaître la longueur de charge de chaque type d'événement ; c'est le
// prochain palier, et il est suivi au plan `PLAN_PERCER_TRAME_FILM_2026-08-30.md`.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

const (
	// zoomFamilyByte : le premier octet des paquets dont l'événement de tête est unit_zoom.
	zoomFamilyByte = 0xCA
	// zoomEventType : le numéro de type de `unit_zoom` sous la numérotation du dispatcher.
	zoomEventType = 21
	// zoomSlotBase : la base du domaine 4, MESURÉE (98 % contre 0 % pour les autres bases).
	zoomSlotBase = 512
)

// zoomRefDomains : les domaines des trois références de `unit_zoom`, dans l'ordre de lecture.
var zoomRefDomains = [3]int{4, 8, 7}

// zoomRefWidth : largeur de l'index par domaine (table des domaines de l'exécutable).
var zoomRefWidth = map[int]uint{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}

// ZoomEvent est UNE bascule de lunette : quand, pour quel bipède, et vers quel palier.
type ZoomEvent struct {
	// TimestampUS : l'horodatage du paquet porteur, en microsecondes (horloge du film) — la
	// même que [BipedPosition.TimestampUS].
	TimestampUS uint64
	// Slot : le slot du bipède qui zoome, directement comparable à [BipedPosition.Slot].
	Slot uint32
	// Level : le palier de lunette APRÈS la bascule. 0 = plus de lunette (sortie) ; 1 et
	// au-delà = à la lunette. Le grossissement dépend de l'arme et n'est pas transmis.
	//
	// LES PALIERS SUPÉRIEURS EXISTENT, et c'est une mesure, pas une déduction : sur quatre
	// films, la charge prend la valeur 2 à cinq reprises sur ~1 000 bascules (0 %, 0 %, 0,6 %
	// et 1,0 % selon le film). Ce sont les armes à plusieurs crans — le fusil de précision
	// zoome deux fois. La rareté est attendue : peu d'armes en ont, et peu de joueurs vont
	// au second cran. Un décodage lu au mauvais endroit aurait réparti les quatre valeurs à
	// peu près également ; la distribution observée est écrasée sur {0, 1}, avec une queue.
	Level int
}

// Scoped dit si cet événement met le joueur À la lunette (par opposition à l'en sortir).
func (z ZoomEvent) Scoped() bool { return z.Level > 0 }

// ScanFilmZoomEvents lit les bascules de lunette d'un film, triées par instant.
//
// Lecture seule, sans état global de décodage : ce scanner n'a pas besoin du verrou de paquet
// (il n'appelle aucun décodeur de trame). Les paquets illisibles sont ignorés en silence — un
// chunk absent n'est pas une erreur pour ce scanner, c'est une couverture moindre.
//
// ScanFilmZoomEvents est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [ScanZoomEvents].
func ScanFilmZoomEvents(dir string) []ZoomEvent {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil // meme degradation silencieuse qu'un chunk illisible : couverture moindre
	}
	return ScanZoomEvents(film)
}

// ScanZoomEvents lit les bascules de lunette d'un film DEJA CHARGE, triées par instant.
func ScanZoomEvents(film *filmsource.Film) []ZoomEvent {
	var out []ZoomEvent
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(chunk)
			if pay[0] != zoomFamilyByte {
				continue
			}
			if ev, ok := decodeZoomHead(pay, pk.TimestampUS); ok {
				out = append(out, ev)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// decodeZoomHead lit l'événement de tête d'un paquet de la famille et rend sa bascule.
// ok=false si l'en-tête n'est pas un `unit_zoom` ou si l'unité n'est pas désignée.
func decodeZoomHead(pay []byte, tsUS uint64) (ZoomEvent, bool) {
	br := NewBitReader(pay)
	br.Skip(1) // bit de configuration
	if !br.ReadBit() {
		return ZoomEvent{}, false // liste vide : pas d'événement en tête
	}
	if int(br.ReadBits(7)) != zoomEventType {
		return ZoomEvent{}, false
	}
	idx, ok := readZoomRef(br, zoomRefDomains[0]) // l'unité qui zoome
	readZoomRef(br, zoomRefDomains[1])
	readZoomRef(br, zoomRefDomains[2])
	level := int(br.ReadBits(2))
	if !ok {
		return ZoomEvent{}, false
	}
	return ZoomEvent{
		TimestampUS: tsUS,
		Slot:        uint32(idx + zoomSlotBase),
		Level:       level,
	}, true
}

// readZoomRef consomme une référence gardée et rend (index, présente).
func readZoomRef(br *BitReader, dom int) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(zoomRefWidth[dom])
	br.Skip(2) // génération
	return idx, true
}

// ZoomStateAt reconstruit l'état de lunette par slot et rend une fonction de consultation :
// pour un slot et un instant, le palier en vigueur (0 = pas à la lunette).
//
// LA RÈGLE DE MAINTIEN, ET ELLE EST DÉLIBÉRÉMENT PRUDENTE : un palier reste en vigueur jusqu'à
// la bascule suivante DU MÊME SLOT, mais pas au-delà de holdUS. Les sorties de lunette sont
// sous-comptées (cf. la réserve en tête de fichier) ; sans plafond, une entrée non refermée
// laisserait un joueur « à la lunette » pendant tout le reste du match. Le plafond transforme
// une lacune de mesure en absence d'affirmation, ce qui est le bon défaut.
func ZoomStateAt(evts []ZoomEvent, holdUS uint64) func(slot uint32, tsUS uint64) int {
	parSlot := map[uint32][]ZoomEvent{}
	for _, e := range evts {
		parSlot[e.Slot] = append(parSlot[e.Slot], e)
	}
	return func(slot uint32, tsUS uint64) int {
		liste := parSlot[slot]
		i := sort.Search(len(liste), func(i int) bool { return liste[i].TimestampUS > tsUS })
		if i == 0 {
			return 0
		}
		last := liste[i-1]
		if last.Level == 0 || tsUS-last.TimestampUS > holdUS {
			return 0
		}
		return last.Level
	}
}
