package grammar

// transloc_events.go — LA TÉLÉPORTATION DU TRANSLOCATEUR, LUE DANS LES ÉVÉNEMENTS DU FILM :
// QUAND, QUI, ET D'OÙ À OÙ.
//
// CE QUE C'EST. Chaque usage du translocateur émet un événement type 117
// `EquipmentTranslocatorTeleportEffects` (nom sourcé de l'exécutable, lecteur 0x140f04fb8)
// dans la liste d'événements en tête d'un paquet delta — le même canal que `unit_zoom`
// (zoom_events.go), qui est le patron de ce fichier. L'événement porte la référence du
// bipède qui saute, PUIS LES DEUX POSITIONS DU VA-ET-VIENT, et il précède la discontinuité de
// position de 5 à 80 ms.
//
// LA MESURE QUI FONDE LE FICHIER (rapport R1 §4, 2026-09-03, 5 films) : précision 18/18
// (chaque tête 117 correspond à une discontinuité vérifiée de 3,24 à 25,36 m du slot
// désigné), rappel 8/8 sur les téléportations connues — y compris un saut de 3,24 m
// invisible à tout seuil spatial. AUCUNE heuristique : l'événement EST la mesure.
//
// LA FORME DU RECORD (mesurée sur pièces, R1 §4.2) : premier octet 0xFA = bit de
// configuration (1) + bit de présence (1) + les 6 bits hauts du type ; le bit de poids
// faible du type est le premier bit du 2e octet (117 = 1110101). Puis ref0 = l'unité qui se
// téléporte : porte 1 bit, index 8 bits (domaine 2), génération 2 bits — l'index plus 512
// donne le SLOT du bipède, la même fermeture que le zoom (base mesurée, 18/18 slots
// plausibles). Les refs 1 et 2 sont absentes (portes à 0) sur toutes les observations.
//
// LA CHARGE, LUE DANS L'EXE ET VALIDÉE 18/18 (rapport R6 §1, décompilation FUN_140f04fb8 +
// FUN_14080d69c + FUN_14076e524 + FUN_140cc5128 ; écrivain FUN_142eec354 symétrique) :
//
//	[R(1) g0 ; si g0 : R(32) mot]   identifiant d'effet — mesuré constant 0xA1344FC2
//	position A : [R(1) porte ; si porte==0 : R(wr) index de région] R(bx) R(by) R(bz)
//	position B : idem
//
// A est le DÉPART du saut, B son ARRIVÉE — dans cet ordre sur les 18 événements des 5 films,
// à 0,00-0,26 m des discontinuités de piste correspondantes.
//
// DEUX PIÈGES, SOURCÉS ET RESPECTÉS ICI :
//
//  1. LA PORTE DE RÉGION EST INVERSÉE. Le bloc « lire l'index de région et prendre les bornes
//     de CETTE région » s'exécute quand le bit vaut ZÉRO ; le bit à UN sélectionne les bornes
//     par défaut du moteur (±20000, 22 bits par axe, DAT_143b8c6b8). C'est l'inverse de la
//     lecture naïve, et la validation film confirme ce sens.
//  2. LES BORNES SONT CELLES DU CATALOGUE (`map_quant_bounds.json`, profile.MapQuantEntry), JAMAIS le
//     champ `bounds` d'un artefact de rejeu, qui n'est qu'un cadrage d'affichage. Déquantifier
//     avec ce dernier fut l'erreur qui a fait manquer les positions à la sonde de R1.
//
// PAS DE BORNES -> PAS DE POSITION (règle map_bounds.go). Une carte absente du catalogue, une
// entrée sans largeurs d'axe, une région autre que celle que le catalogue décrit, une charge
// non conforme : la téléportation sort SANS positions, et l'appelant compte le cas. Jamais une
// position devinée — l'instant et le slot, eux, restent lus.
//
// RÉSERVE HONNÊTE, la même que zoom_events.go : seuls les événements EN TÊTE de paquet sont
// lus. Trois `spent` du corpus n'ont aucune tête 117 de leur slot (expiration sans usage, ou
// événement hors tête de liste — non départagé, R1 §4.3). Lire la liste entière est le
// chantier PLAN_PERCER_TRAME_FILM.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const (
	// translocFamilyByte : le premier octet des paquets dont l'événement de tête est le
	// type 117 — config(1) + présence(1) + les 6 bits hauts de 1110101.
	translocFamilyByte = 0xFA
	// translocEventType : le numéro de type de `EquipmentTranslocatorTeleportEffects` sous
	// la numérotation du dispatcher.
	translocEventType = 117
	// translocRefDomain : le DOMAINE de ref0. Sa largeur se lit dans `refDomWidth`
	// (event_list.go), la seule table des domaines du paquet. La copie locale de cette
	// largeur a disparu le 2026-09-05 (lot E, item E.3).
	translocRefDomain = 2
	// translocRefWidth : la même largeur, sous forme de constante, pour les instruments qui
	// calculent une longueur d'en-tête à la compilation. Le garde-rail
	// `event_preamble_guard_test.go` interdit qu'elle diverge de `refDomWidth(2)`.
	translocRefWidth = dom2RefWidth
	// translocGenBits : largeur du champ de génération de ref0.
	translocGenBits = 2
	// translocSlotBase : la base du domaine — même fermeture mesurée que zoomSlotBase.
	translocSlotBase = 512
	// translocEffectWordBits : largeur du mot gardé qui ouvre la charge (l'identifiant
	// d'effet, mesuré constant 0xA1344FC2 sur les 18 événements). Sa VALEUR n'est pas
	// contrôlée : c'est un identifiant de contenu, pas une signature de format — un autre
	// effet en changerait sans rien changer aux positions qui suivent.
	translocEffectWordBits = 32
	// translocMaxAxisBits : plafond de la loi du moteur sur une largeur d'axe
	// (min(26, ...), cf. profile.MapQuantEntry.AxisWidths). translocMaxRegionBits borne de même la
	// largeur de l'index de région. Au-delà, l'entrée de catalogue est refusée plutôt que
	// lue : une largeur aberrante ferait lire n'importe quoi.
	translocMaxAxisBits   = 26
	translocMaxRegionBits = 8
)

// ScanFilmTranslocatorTeleports lit les téléportations du translocateur d'un film, triées
// par instant.
//
// `entry` est l'entrée de catalogue de la carte du match — celle qui porte les bornes de
// déquantification. Nil (ou inutilisable) n'est pas une erreur : les événements sortent
// datés et attribués, mais sans positions.
//
// Lecture seule, sans état global de décodage : comme le zoom, ce scanner n'appelle aucun
// déserialiseur de trame et n'a pas besoin du verrou de paquet. Les chunks illisibles sont
// une couverture moindre, pas une erreur.
//
// ScanFilmTranslocatorTeleports est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film puis
// appelle [ScanTranslocatorTeleports]. La cuisson, elle, passe le film qu'elle a déjà chargé.
func ScanFilmTranslocatorTeleports(dir string, entry *profile.MapQuantEntry) []types.TranslocatorTeleport {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil // meme degradation silencieuse qu'un chunk illisible : couverture moindre
	}
	return ScanTranslocatorTeleports(NewFilmContext(film), entry)
}

// ScanTranslocatorTeleports lit les téléportations du translocateur d'un film DEJA CHARGE, dans la
// passe des têtes ([canalDesTeleportations]), triées par instant. Cf.
// [ScanFilmTranslocatorTeleports] pour la doctrine du balayage.
func ScanTranslocatorTeleports(fc *FilmContext, entry *profile.MapQuantEntry) []types.TranslocatorTeleport {
	c := &canalDesTeleportations{entry: entry}
	distribuerLesTetesSeules(fc, []Canal{c})
	return c.out
}

// canalDesTeleportations lit l'événement de translocation de tête des trames de la famille.
type canalDesTeleportations struct {
	entry *profile.MapQuantEntry
	out   []types.TranslocatorTeleport
}

func (*canalDesTeleportations) Interets() []Interet { return nil }

func (c *canalDesTeleportations) Tete(p *lecture.Paquet) {
	if len(p.Payload) < 2 || p.Payload[0] != translocFamilyByte {
		return
	}
	if ev, ok := decodeTranslocHead(p.Payload, p.TS, c.entry, teteDe(p)); ok {
		c.out = append(c.out, ev)
	}
}

func (c *canalDesTeleportations) Clore(BilanDeMarche) { trierTeleportations(c.out) }

// trierTeleportations range les teleportations dans un ordre TOTAL (lot J10.1, 2026-09-27, DT-9) :
// instant, puis slot — deux paquets de la famille peuvent porter le meme horodatage —, les ex
// aequo restants (meme instant, meme slot) dans l ORDRE DU FILM, seule cle qui les distingue.
func trierTeleportations(out []types.TranslocatorTeleport) {
	slices.SortStableFunc(out, func(a, b types.TranslocatorTeleport) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Slot, b.Slot))
	})
}

// decodeTranslocHead lit l'événement de tête d'un paquet de la famille, de tête `t`, et rend la
// téléportation. ok=false si la tête n'est pas un type 117 ou si l'unité n'est pas désignée
// (porte de ref0 à 0 — jamais observé, mais un slot non transmis ne se devine pas). La
// CHARGE, elle, ne conditionne rien : elle échoue en positions absentes, pas en événement
// perdu (l'instant et le slot sont déjà lus).
func decodeTranslocHead(pay []byte, tsUS uint64, entry *profile.MapQuantEntry, t teteDeTrame) (
	types.TranslocatorTeleport, bool,
) {
	if !t.liste {
		return types.TranslocatorTeleport{}, false // liste vide : pas d'événement en tête
	}
	if t.genre != translocEventType {
		return types.TranslocatorTeleport{}, false
	}
	br := LecteurSur(pay)
	br.SetBitPos(eventPayloadStartBit) // le corps suit la tête (event_list.go)
	if !br.ReadBit() {
		return types.TranslocatorTeleport{}, false // ref0 absente : pas d'unité désignée
	}
	idx := br.ReadBits(refDomWidth(translocRefDomain))
	br.Skip(translocGenBits)
	ev := types.TranslocatorTeleport{TimestampUS: tsUS, Slot: uint32(idx + translocSlotBase)}
	ev.From, ev.To, ev.HasPositions = decodeTranslocJump(br, entry)
	return ev, true
}

// decodeTranslocJump lit la CHARGE de l'événement après ref0 : les portes des refs 1-2, le
// mot d'effet gardé, puis les DEUX positions quantifiées. Rend (départ, arrivée, lues).
func decodeTranslocJump(br *Lecteur, entry *profile.MapQuantEntry) ([3]float32, [3]float32, bool) {
	var none [3]float32
	// LES DEUX PORTES SE LISENT, PAS UNE. Un `||` court-circuitait la seconde (SA4000) : sans
	// conséquence ici — le chemin sort aussitôt et le lecteur de bits est abandonné — mais
	// l'intention (deux portes, lues dans l'ordre du flux) appartient au code, pas au hasard
	// d'un court-circuit.
	ref1, ref2 := br.ReadBit(), br.ReadBit()
	if ref1 || ref2 {
		// Une ref 1 ou 2 présente décale tout ce qui suit d'une largeur non sourcée pour ce
		// type (domaines {2,0,0}, R6 §1.1) : la charge n'est plus lisible. Jamais observé.
		return none, none, false
	}
	if br.ReadBit() {
		br.Skip(translocEffectWordBits)
	}
	from, ok := readTranslocVec(br, entry)
	if !ok {
		return none, none, false
	}
	to, ok := readTranslocVec(br, entry)
	if !ok {
		return none, none, false
	}
	// PAS DE SECOND CONTRÔLE DE DÉBORDEMENT ICI, et c'est délibéré : `readTranslocVec` est la
	// SEULE garde, et elle refuse déjà tout vecteur dont les bits dépassent le tampon (le
	// Lecteur lit des zéros au-delà — padding de queue du moteur). Un `br.Remaining() < 0`
	// ajouté après coup serait INATTEIGNABLE, et sa présence masquerait la disparition de la
	// vraie garde (revue P1bis ronde 1, G4 : la redondance rendait les deux mutations vertes).
	return from, to, true
}

// readTranslocVec lit UNE position quantifiée de la charge et la déquantifie en coordonnées
// monde. PORTE INVERSÉE (cf. l'en-tête, piège n°1) : bit à 0 -> index de région puis bornes
// de la carte ; bit à 1 -> bornes par défaut du moteur.
//
// LA LECTURE EST CELLE DU PORTAGE UNIQUE (lot J6.3, 2026-09-27) : `FUN_140f04fb8` garde chaque
// position par `FUN_14076f91c` puis appelle `FUN_14076e524(0x10)` (CALLs 140f04ff0 et 140f05023,
// `MOV R9D,0x10` en 140f04fe5 et 140f05018). Ce fichier en tenait une seconde copie — largeur par
// défaut 22 et bornes ±20000 recopiées en constantes, sans la garde. Les tables viennent ici de
// l'ENTRÉE DE CATALOGUE du match ([tablesDeLEntree]), pas du profil du lecteur : le balayage du
// translocateur n'installe aucun profil, et c'est l'entrée qui porte les bornes.
func readTranslocVec(br *Lecteur, entry *profile.MapQuantEntry) ([3]float32, bool) {
	var out [3]float32
	pos, ok := lireE494Sur(br, niveauPosition, tablesDeLEntree(entry))
	if !ok || pos.brute {
		return out, false // pas de bornes -> pas de coordonnée monde (map_bounds.go) ; brut -> pas de quanta
	}
	rng := profile.QuantRangeParDefautDuBuild()
	if pos.idx >= 0 {
		if uint32(pos.idx) != entry.Region {
			// Une AUTRE région : ses quanta sont exprimés dans une autre AABB, et les
			// déquantifier avec ces bornes produirait une position fausse silencieuse —
			// exactement le refus que porte profile.I0Layout.Region sur le chemin du bipède.
			return out, false
		}
		rng = entry.Range()
	}
	lay := profile.I0Layout{AxisW: pos.w}
	for ax := range 3 {
		out[ax] = DequantBipedAxis(uint32(pos.q[ax]), ax, lay, rng)
	}
	return out, br.Remaining() >= 0
}

// tablesDeLEntree rend les tables de `FUN_14076e524` que porte une entrée de catalogue. Une
// entrée absente ou hors enveloppe ([translocEntryUsable]) ne sait pas lire un index : la lecture
// s'arrête alors sur la porte — la table DÉFAUT, elle, reste lisible sans carte.
func tablesDeLEntree(e *profile.MapQuantEntry) tablesDePosition {
	if !translocEntryUsable(e) {
		return tablesDePosition{}
	}
	return tablesDePosition{indexW: e.EffectiveRegionIndexBits(), axesCarte: e.AxisWidths,
		bornesCarte: bornesDeLaPlage(e.Range()), indexLisible: true}
}

// tablesDeLaRegionJouee rend les tables de `FUN_14076e524` que l entree de catalogue de la carte du
// match fait connaitre : celles de [tablesDeLEntree], restreintes a l index de sa region jouee. La
// grammaire de la vue A en fait une CONDITION de lecture (un autre index arrete la lecture,
// [lireE524Sur]) ; elle n accepte donc qu une region LUE dans le jeu.
//
// PROVENANCE DE [profile.MapQuantEntry.Region] (`cmd/mapquant-build`). Pour une carte dont le module
// porte ses tags sbsp, la region est la 0 de l ordre des regions de compression du bloc
// structure-BSP du tag de niveau (`himap.BSPQuantification`) : lue. Une region autre que 0 ne vient
// que de `regionExterneDeclarations` (le seul site du constructeur qui pose le champ) : une carte dont
// les regions vivent dans `ds/globals` (Live Fire, region 1 sur 4), ou la region JOUEE est tranchee
// par mesure (ancres d objectifs du catalogue contenues dans son AABB, index des records i0 de ses
// films), pas lue. Sous une telle entree, aucun index n est lisible : la lecture s arrete sur la porte
// de tout index, comme sans carte, plutot que de lire des largeurs sous une region choisie a la
// mesure. La table DEFAUT (porte a 1) reste lisible.
func tablesDeLaRegionJouee(e profile.MapQuantEntry) tablesDePosition {
	t := tablesDeLEntree(&e)
	t.regionSeule, t.region = true, e.Region
	if e.Region != regionLueDansLeTag {
		t.indexLisible = false
	}
	return t
}

// regionLueDansLeTag est la seule region jouee que le catalogue tient du tag de niveau (cf.
// [tablesDeLaRegionJouee]).
const regionLueDansLeTag = 0

// translocEntryUsable dit si l'entrée de catalogue permet une déquantification : bornes
// ordonnées et largeurs dans l'enveloppe de la loi du moteur. Une entrée hors enveloppe est
// REFUSÉE plutôt que lue — la largeur commande le nombre de bits consommés, une valeur
// aberrante ferait lire les bits du voisin.
func translocEntryUsable(e *profile.MapQuantEntry) bool {
	if e == nil || e.EffectiveRegionIndexBits() > translocMaxRegionBits {
		return false
	}
	for ax := range 3 {
		if e.AxisWidths[ax] == 0 || e.AxisWidths[ax] > translocMaxAxisBits {
			return false
		}
		if e.Max[ax] <= e.Min[ax] {
			return false
		}
	}
	return true
}
