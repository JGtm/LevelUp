package grammar

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// Décodage OFFLINE des ÉVÉNEMENTS DE TIR du film : le record `action_weapon_fire`, TYPE 36 de la
// liste d'événements (numérotation trame, `event_list.go`).
//
// CE QU'EST CE RECORD (RE Ghidra, répartiteur `FUN_14080a9d4`, désérialiseur `FUN_14080c1f8`) :
// un COUP TIRÉ, pas un dégât. LE DÉGÂT EST UN RECORD DISTINCT — `damage_aftermath` (type 0), avec
// son propre responsable et sa victime. Un tir n'est donc PAS forcément une touche : la touche se
// reconstruit en APPARIANT un tir à un `damage_aftermath` du même attaquant dans une fenêtre
// (méthode PAR LE TIR, `NOTE_ATTRIBUTION_ARME_TIR_2026-08-31`, `weapon_hits.go`).
//
// # LE RECORD EST LU PAR SA GRAMMAIRE, PLUS À DES OFFSETS FIXES (lot M4b, 2026-09-24)
//
// Jusqu'à ce lot, la tête du record était lue à des OFFSETS FIXES (attaquant aux bits 36..40,
// arme aux bits 44..107), derrière un filtre sur l'octet de tête 0xD2. Ces offsets ne valent que
// pour la disposition CANONIQUE — référence 0 présente avec sa sonde, références 1 et 2 absentes,
// indice de tireur présent, arme haute présente — et le filtre laissait passer le type 37
// (`weapon_overheat`, même octet de tête). Mesures du rapport `RAPPORT_tirs_vehicules.md` (§6) :
// 1 205 tirs publiés sur 24 documents avec un identifiant d'arme DÉCALÉ (garde du tireur fermée :
// cinq bits de moins), et les 39 joueurs de BTB d'index ≥ 16 sans aucun tir publié (l'indice lu
// sur 4 bits). La grammaire, relue chez l'écrivain et déjà portée pour la visée
// (`fire_aim_modal.go`), est désormais LA lecture :
//
//	[config R(1)][continuation R(1)][type R(7) = 36]
//	ref0 domaine 1 (garde, sonde ; R(9) si sonde, sinon R(13) ; R(2) génération)   L'UNITÉ TIREUSE
//	ref1 domaine 8 · ref2 domaine 7 (garde ; R(13) + R(2))
//	a  estCourt R(1)       b  estBloc R(1)
//	c  NUMÉRO DE TIR R(7) puis R(1) — `n mod 256 = R(7) | R(1) << 7` (sonde P1, S1 bis)
//	d  R(1) ; si 0 : R(5)  L'INDICE DE TIREUR (polarité inversée) — la PLACE, sur cinq bits
//	e  R(1) ; si 0 : R(2)
//	f  R(1) ; si 1 : R(32) l'arme, moitié haute (famille)
//	g  R(32)               l'arme, moitié basse (variante)
//	i, j R(1), R(1)        puis les comptes et la visée (`fire_aim_modal.go`)
//
// La RÉFÉRENCE 0 est l'UNITÉ qui tire : le bipède à pied, le VÉHICULE ou la PIÈCE MONTÉE pour une
// arme de véhicule à coup (sonde P1 : `772` pour le mortier du Wraith sur `8a485699`). Son slot est
// `0x200 + index` — la base des catégories 1 et 4 de `FUN_1406d3140` (`FUN_140d10bb0`), la même que
// celle du parent d'`object-parent-state` ([parentHandleBase]).
//
// NON RÉSOLU, à ne pas prétendre : la VICTIME n'est pas décodée (elle vit dans la liste des cibles).
// Un record 36 dit qui tire, avec quoi, quand, depuis quelle unité, et vers où — pas qui est touché.
// Les records 36 qui ne sont PAS en tête de liste ne sont pas lus : sauter les événements qui les
// précèdent demanderait la grammaire de charge de chacun (limite nommée au rapport du lot).

// TypeTirArme est le type d'événement `action_weapon_fire` dans la liste d'un paquet delta.
const TypeTirArme = 36

// FireAimBits est la largeur du champ de visée du record (0x1E au site d'appel `FUN_1406D8288`
// dans `FUN_14080C1F8`).
const FireAimBits uint = 30

// Largeurs de la tête du record, lues chez l'écrivain (`FUN_141fcf670` et suivants).
const (
	largeurNumeroDeTir = 7  // c : R(7), puis un R(1) de poids fort
	largeurTireur      = 5  // d : l'indice de tireur
	largeurChampE      = 2  // e
	largeurMotArme     = 32 // f, g : les deux moitiés de l'identifiant d'arme
	// bitPoidsFortNumero : le R(1) qui suit les sept bits du numéro de tir en est le HUITIÈME,
	// de poids fort (P1 : les valeurs brutes enchaînent … c252 c254 c1 c3 … au passage de 128).
	bitPoidsFortNumero = 7
)

// FireEvent est un événement de tir décodé (tête du record type 36).
type FireEvent struct {
	// Chunk / PacketIndex localisent l'event dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet, en microsecondes — MÊME horloge que
	// BipedPosition.TimestampUS, donc directement croisable avec les positions.
	TimestampUS uint64
	// FilmIndex est l'index de joueur du TIREUR tel que le film l'écrit — le champ `d`, sur ses
	// CINQ bits : la PLACE du tireur (sonde P4, `sieges_tirs.go`), jamais une identité. -1 quand
	// la garde du champ est fermée ([FireEvent.HasShooter] faux) : le tir ne nomme alors que son
	// unité (référence 0).
	//
	// L'IDENTITÉ D'UN JOUEUR EST SON XUID. Toute jointure passe par lui ; cet index ne sert qu'à
	// regrouper les événements d'un même tireur À L'INTÉRIEUR d'un film. Il s'appelait
	// `PlayerIndex`, et avait été confondu avec le tri alphabétique du roster.
	//
	// C'EST L'INDICE QU'ÉCRIT `shared.match_weapon_shots` : il est aligné sur
	// `weaponscan.FireEvent.FilmIndex5` (event_start + 31, cinq bits), le DÉNOMINATEUR de la
	// précision (`TestWeaponIndexNumDenomEquivalence`). L'ancien champ à quatre bits (bits 36..39)
	// saturait à 15 au-delà de seize joueurs : il a disparu avec les offsets fixes (lot M4b).
	FilmIndex int
	// HasShooter dit que le record porte l'indice de tireur (garde du champ `d` ouverte).
	HasShooter bool
	// FireNumber est le NUMÉRO DE TIR du joueur, modulo 256 (`FUN_14202f3a0` l'incrémente à
	// CHAQUE tir, émis ou non : ses sauts comptent les tirs continus non écrits, sonde P1-S3).
	FireNumber uint8
	// Unit est la RÉFÉRENCE 0 : l'unité qui tire (bipède, véhicule ou pièce montée).
	Unit UnitRef
	// Short et Bloc sont les deux drapeaux de tête (`estCourt`, `estBloc`).
	Short, Bloc bool
	// WeaponID est l'identifiant global 64 bits de l'arme : clé directe de
	// metadata.weapon_labels.weapon_id et de filmshell.WeaponIDToName. Sa moitié haute vaut 0
	// quand la garde du champ `f` est fermée.
	WeaponID uint64
	// HasAim indique que la visée a pu être lue (record modal, `fire_aim_modal.go`).
	HasAim bool
	// Aim est la direction UNITAIRE monde du tir (cubemap 30 bits déquantifié).
	Aim [3]float32
}

// UnitRef est la référence d'unité d'un record : sa présence, son slot (`0x200 + index`), sa
// génération, et la sonde qui a fixé la largeur de l'index.
type UnitRef struct {
	Present bool
	Slot    uint32
	Gen     uint32
	Probe   bool
}

// ScanFilmFireEvents décode tous les événements de tir des chunks du film de dir.
//
// ScanFilmFireEvents est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [ScanFireEvents].
func ScanFilmFireEvents(dir string) ([]FireEvent, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanFireEvents(film)
}

// ScanFireEvents décode les événements de tir d'un film DEJA CHARGE : le record type 36 EN TÊTE
// de la liste d'un paquet delta. Les chunks illisibles sont ignorés (le film peut être partiel).
func ScanFireEvents(film *source.Film) ([]FireEvent, error) {
	var out []FireEvent
	read := 0
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		read++
		for _, p := range pks {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			e, ok := decodeFireEvent(p.Payload(chunk))
			if !ok {
				continue // autre type d'événement de tête, ou record tronqué
			}
			e.Chunk, e.PacketIndex, e.TimestampUS = c, p.Index, p.TimestampUS
			out = append(out, e)
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	return out, nil
}

// enteteTir36 est la tête du record lue par la grammaire, jusqu'aux drapeaux `i`, `j` inclus.
type enteteTir36 struct {
	unite           guardedRef
	court, bloc     bool
	numero          uint8
	tireur          int
	armeHaute       uint32
	armeBasse       uint32
	apresDrapeaux   int // le bit qui suit `i` et `j`
	longueurPayload int
}

// lireEnteteTir36 lit la tête d'un record `action_weapon_fire` en tête de liste. Rend ok=false
// quand le paquet ne porte pas ce type en tête, ou que la tête déborde du payload.
func lireEnteteTir36(pay []byte) (enteteTir36, bool) {
	h := enteteTir36{tireur: -1, longueurPayload: len(pay) * 8}
	if len(pay) < 2 {
		return h, false
	}
	br := LecteurSur(pay)
	tete := readPacketHead(br)
	if !tete.More || tete.Type != TypeTirArme {
		return h, false
	}
	h.unite = lireRefDomaine1(br)
	for _, dom := range []int{8, 7} { // ref1 domaine 8, ref2 domaine 7 : gardees, sautees
		if br.ReadBit() {
			br.ReadBits(refDomWidth(dom) + largeurGenerationRef)
		}
	}
	h.court, h.bloc = br.ReadBit(), br.ReadBit()
	bas := br.ReadBits(largeurNumeroDeTir)
	h.numero = uint8(bas | br.ReadBits(1)<<bitPoidsFortNumero) //nolint:gosec // huit bits
	if !br.ReadBit() {                                         // d : polarité inversée
		h.tireur = int(br.ReadBits(largeurTireur)) //nolint:gosec // cinq bits
	}
	if !br.ReadBit() { // e : polarité inversée
		br.ReadBits(largeurChampE)
	}
	if br.ReadBit() { // f : l'arme, moitié haute
		h.armeHaute = uint32(br.ReadBits(largeurMotArme)) //nolint:gosec // 32 bits
	}
	h.armeBasse = uint32(br.ReadBits(largeurMotArme)) //nolint:gosec // 32 bits
	br.ReadBits(2)                                    // i, j
	h.apresDrapeaux = br.BitPos()
	return h, h.apresDrapeaux <= h.longueurPayload
}

// largeurGenerationRef : les deux bits de génération qui suivent l'index d'une référence gardée.
const largeurGenerationRef = 2

// lireRefDomaine1 lit une référence gardée de DOMAINE 1 sur le lecteur — garde R(1) ; si 1 :
// sonde R(1), R(9) si sonde sinon R(13), R(2) génération — la même grammaire que [readDom1Ref],
// mais par le LECTEUR, qui rend des zéros au-delà du payload au lieu de paniquer.
func lireRefDomaine1(br *Lecteur) guardedRef {
	var r guardedRef
	if r.Present = br.ReadBit(); !r.Present {
		return r
	}
	largeur := uint(dom7RefWidth)
	if br.ReadBit() {
		r.Sonde, largeur = 1, varWidthBits(varWidthProbeSlot)
	}
	r.Index = uint32(br.ReadBits(largeur))            //nolint:gosec // au plus 13 bits
	r.Gen = uint32(br.ReadBits(largeurGenerationRef)) //nolint:gosec // deux bits
	r.EndBit = br.BitPos()
	return r
}

// decodeFireEvent lit la tête du record type 36 d'un payload de paquet. Rend ok=false, sans rien
// rendre, si le payload ne porte pas ce record en tête ou si la tête déborde.
//
// LA GARDE N'EST PAS DÉFENSIVE « au cas où » : un film tronqué par un téléchargement partiel
// porte des paquets coupés, et ce décodeur tourne aussi dans un collecteur de fond du process de
// sync (`killcollector`), où une panique coûte le process entier. Le lecteur rend des zéros
// au-delà du payload ; la tête n'est acceptée que si elle y TIENT.
func decodeFireEvent(pay []byte) (FireEvent, bool) {
	h, ok := lireEnteteTir36(pay)
	if !ok {
		return FireEvent{}, false
	}
	e := FireEvent{FilmIndex: h.tireur, HasShooter: h.tireur >= 0, FireNumber: h.numero,
		Short: h.court, Bloc: h.bloc,
		WeaponID: uint64(h.armeHaute)<<32 | uint64(h.armeBasse)}
	if h.unite.Present {
		e.Unit = UnitRef{Present: true, Slot: parentHandleBase + h.unite.Index, Gen: h.unite.Gen,
			Probe: h.unite.Sonde == 1}
	}
	if aimBit, okAim := modalAimBitFrom(pay, h); okAim {
		readAimAt(pay, &e, aimBit)
	}
	return e, true
}

// TireurDuTir rend l'indice de tireur (cinq bits) du record type 36 en tête d'un payload, ou -1
// (autre type en tête, garde du tireur fermée, tête tronquée).
//
// EXPORTÉ POUR LA MESURE, et pour une raison précise : les instruments qui confrontent l'indice du
// NUMÉRATEUR de la précision (celui-ci) à celui du DÉNOMINATEUR (`weaponscan.FireEvent.FilmIndex5`)
// ou aux vies du rejeu doivent lire LA grammaire de production, pas en recopier les champs — une
// seconde copie d'un offset de bit est exactement ce que ce lot a retiré.
func TireurDuTir(pay []byte) int {
	e, ok := decodeFireEvent(pay)
	if !ok || !e.HasShooter {
		return -1
	}
	return e.FilmIndex
}

// AimHeadingDeg renvoie le cap de visée du tir dans le plan XY, en degrés dans [0,360[,
// même origine et même sens que atan2(Y, X) des positions déquantifiées.
func (e FireEvent) AimHeadingDeg() (float64, bool) {
	if !e.HasAim {
		return 0, false
	}
	h := math.Atan2(float64(e.Aim[1]), float64(e.Aim[0])) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h, true
}
