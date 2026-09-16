package grammar

import "fmt"

// FRAME-delta decoder (L3) — port of the type-0 packet record loop FUN_1406cd128 +
// dispatch FUN_1406cbaa0 + delta FUN_141f86b58. Grammar extracted via Ghidra
// (workflow frame-decoder-grammar). See .ai/V7.5/killweapon/PLAN_FILM_KILLFEED_V3.md / HANDOFF.
//
// Per-record header (CONFIRMED, the two traps a naive port falls into):
//   - record TYPE is a PREFIX CODE, not a flat R(2):
//       R(1) ; if 1 -> DELTA (1 bit) ; if 0 -> R(2) in {0=end,1=new,2=del,3=delta}.
//   - record ID is table-driven, not a fixed R(7):
//       low = R(idLowBits) + idBase ; tag = R(2) placed in bits 30-31.
//       id = (tag<<30) | (low & 0x3fffffff) ; slot = id & 0x3fffffff.
//
// Per-type body:
//   NEW   (1): [if hasExtraFields && R(1): R(8)] then R(6) typeIndex + default-state
//              + gate + mask + components  == TraverseEntity (default-state IS in stream).
//   DEL   (2): [if hasExtraFields && R(1): R(8)] then R(32).
//   DELTA (3): NO typeIndex, NO default-state (archetype + base state from RAM/World):
//              mask + present components only.
//
// An optional per-record R(32) prefix precedes every record iff hasExtraFields
// (cVar1 = FUN_14076cea8, a global config flag; hypothesised false for offline
// Theater films).

const (
	recEnd   = 0 // type 0: end of record loop
	recNew   = 1 // type 1: new entity (default-state in stream)
	recDel   = 2 // type 2: deletion
	recDelta = 3 // type 3: delta (common case; archetype + base state from World)
)

// FrameConfig holds the parameters that are NOT present in the payload and come from
// the game's runtime context. They must be calibrated empirically (sweep) — see
// open_risks in the spec. Defaults are the starting hypotheses.
type FrameConfig struct {
	HasExtraFields bool // cVar1: per-record R(32) prefix + R(8) NEW/DEL guards. Theater offline: likely false.
	// IDLowBits est la largeur du champ id bas. C'est une valeur de RUNTIME (FUN_1406d3140
	// -> FUN_1406d310c sur DAT_1451f98d0/d4, peuplee au chargement de carte) : elle DIFFERE
	// d'un film a l'autre. 11 sur 000d5950 (calibre) ; 21 bits d'en-tete = idLow 14 sur le
	// film de la capture live (mesure : cmd/tmp_hdrtruth). Le 13 par defaut n'est qu'une
	// hypothese de depart, PAS une constante du format.
	IDLowBits           int
	IDBase              uint32
	NewDefaultStateBits int // default-state width for a NEW record's archetype (runtime; brute-forced).
	// PacketPreambleBits est l'AMORCE DE PAQUET : les bits consommes sur le bitreader du
	// payload AVANT le premier record. Source (desassemblage) : le frame-processor
	// FUN_142987460 fait `DAT_144706104 = FUN_1406cf008(reader)` — un R(1) — puis boucle
	// sur ses 3 vues, chacune appelant vtable[0x40] = FUN_1406cd128 (la boucle de records).
	// Le reader est cree sur le payload EXACT du paquet (FUN_14298816c : memcpy de
	// *(param_2+4) octets, puis FUN_1406d5cc0(reader,3) qui repositionne byte-ptr au debut),
	// donc ce bit est bien le bit 0 du payload. N'est applique que si le lecteur est au
	// bit 0 (un appel en cours de flux n'est pas un debut de paquet).
	PacketPreambleBits int
	// Profil est le PROFIL DE BALAYAGE que les portes de balayage posent sur leur lecteur de
	// bits, EN TETE (lot 2.2.a pour le mouvement, elargi a tout le profil au lot 2.3). Il
	// porte ce que les variables de paquet portaient : descripteur de traversee, largeur
	// d'axe absolue, drapeaux de queue, cadre d'image-cle, decoupage MPP, `param_4` force.
	//
	// IL EST DANS `FrameConfig` POUR UNE RAISON PRECISE : `FrameConfig` est deja le seul objet
	// que le SEUL ecrivain de production de ces valeurs — la calibration de `killsource` —
	// passe a toutes ses portes. Les y mettre lui rend sa calibration sans qu'elle ecrive dans
	// le processus PENDANT son balayage. Son defaut ([DefaultFrameConfig]) est l'invariant de
	// [ProfilDeBalayageParDefaut].
	Profil ProfilDeBalayage
	// Obs est l OBSERVATEUR que les portes de balayage posent sur leur lecteur, EN TETE
	// (lot 2.2.f). `nil` — le cas de la production — n observe rien.
	//
	// C EST LA FORME « PASSE EN PARAMETRE » QUE L ITEM 2.2.f DEMANDE, et elle est disponible
	// des maintenant pour la famille de l inference de chaine : un instrument passe SON
	// observateur ici et lit ses compteurs sans jamais ecrire dans le processus.
	Obs *Observation
}

// contexte rend ce que ce cadre pose sur un lecteur : son profil et son observateur.
func (c FrameConfig) contexte() ContexteDeLecture {
	return ContexteDeLecture{Profil: c.Profil, Obs: c.Obs}
}

// DefaultPacketPreambleBits est l'amorce de paquet consommee avant le premier record.
//
// POURQUOI 2 ALORS QUE LE DESASSEMBLAGE N'EN MONTRE QU'UN. Le desassemblage etablit UN bit :
// FUN_142987460 fait `DAT_144706104 = FUN_1406cf008(reader)`, et FUN_1406cf008 est un R(1)
// (`*(p+0x2c) += 1`). Ce bit vaut 1 dans 100,00 % des 30 418 payloads de 000d5950 — signature
// d'un drapeau de configuration constant. Le SECOND bit, lui, n'est PAS localise dans le
// desassemblage : il est etabli par la MESURE, et cette distinction doit rester visible.
//
// Trois grammaires donnent le meme en-tete total de 20 bits sur le PREMIER record, et ne se
// separent qu'a partir du second : amorce 2 + idLow 11 · amorce 1 + idLow 12 · amorce 1 +
// prefixe long + idLow 10. Le temoin qui les departage est la FORME DU MASQUE, invariant par
// carte et par film : le compteur de composants tient sur 3 bits, donc un record epars porte
// au plus 7 composants, et la verite terrain (138 390 records bipedes captures sur le
// desassembleur reel) en compte 99,86 % dans cette plage, mode a 4.
//
//	part de masques 1..7 sur les DELTA de slots bindes :
//	  amorce 0, idLow 11 (avant)  : 14,65 %   (n = 33 154)   <- sous le niveau du hasard
//	  amorce 2, idLow 11          : 84,81 %   (n = 33 662)   <- retenu
//	  amorce 1, idLow 12          : 51,67 %   (n =  5 135)
//	  amorce 2, idLow 10          :  5,24 %
//	  niveau du hasard mesure     : 10,67 %   (critere evalue a des decalages arbitraires)
//	  verite terrain              : 99,86 %
//
// CE QUI RESTE FAUX, ET OU. Sur CE chemin — DecodeFrameRecords, dont la marche n'est PAS
// ancree sur la bande de slots bipedes des images-cles — i22 lit encore 92,46 % de comptes
// de grenades impossibles (le compteur doit valoir 4 ; il est uniforme sur 0..7) : il reste
// au moins une faute dans le CORPS des records parcourus ici.
//
// CE CHIFFRE NE VAUT QUE POUR CE CHEMIN, et il est ANTERIEUR aux correctifs de largeur d'i0
// (47 bits), d'i25/i26/i27 et de la polarite de porte d'i30/i33. Le chemin ANCRE
// (matchBipedHeader + walkRecordTo, cf. ability_rank.go et inventory_delta.go) mesure sur le
// meme film 000d5950 : compteur R(3) == 4 dans 120 lectures sur 120 (100,00 %) et valeurs
// R(8) dans {0, 1, 2} exclusivement (etude du 2026-08-24,
// .ai/V7.5/replay2d/FAISABILITE_SUIVI_DELTA_INVENTAIRE_2026-08-24.md §1.3). Ne pas lire les
// 92,46 % comme un verdict sur i22 : c'est un verdict sur la marche non ancree.
//
// Cette amorce est gardee parce qu'elle repose sur trois temoins independants d'i22 et
// concordants — le R(1) du desassemblage, le bit 0 a 100 %, et la forme des masques — et non
// parce qu'elle "ameliore" un chiffre.
const DefaultPacketPreambleBits = 2

// DefaultFrameConfig is the starting hypothesis for an offline Theater film.
//
// IDLowBits 13 reste une hypothese de depart a calibrer (c'est une valeur de RUNTIME : 11 sur
// 000d5950, 14 sur le film de la capture live). L'amorce, elle, est une propriete du FORMAT et
// non du runtime : elle a donc sa place ici en dur.
//
// LE MOUVEMENT VIENT DE L HERITAGE, PAS DE L INVARIANT, et la distinction porte (lot 2.2.a) :
// une porte de balayage qui reconstruit un cadre par defaut APRES la calibration de
// `killsource` heritait, avant ce lot, des largeurs calibrees par les variables de paquet du
// processus. Le cadre par defaut dit donc la meme chose qu elles disaient. Au repos, l heritage
// VAUT l invariant [profile.MouvementParDefaut].
func DefaultFrameConfig() FrameConfig {
	return FrameConfig{HasExtraFields: false, IDLowBits: 13, IDBase: 0, NewDefaultStateBits: 0,
		PacketPreambleBits: DefaultPacketPreambleBits, Profil: ProfilDeBalayageParDefaut()}
}

// FrameRecord is one decoded record of a type-0 FRAME packet.
type FrameRecord struct {
	Type      int
	ID        uint32
	Slot      uint32 // ID & 0x3fffffff
	TypeIndex uint32
	Trace     EntityTrace
	DesyncAt  int
	// HeaderBit : position bit (dans le payload) du PREMIER bit de l'en-tete du record.
	// Necessaire pour confronter la marche sequentielle a l'oracle Rosette, dont la colonne
	// recordHeaderBit donne la position VRAIE de chaque record capture : sans elle on ne peut
	// pas localiser le point de rupture de la marche.
	HeaderBit int
}

// readRecordType ports the record-type PREFIX CODE: R(1); set -> DELTA; else R(2).
func readRecordType(br *Lecteur) int {
	if br.ReadBit() {
		return recDelta
	}
	return int(br.ReadBits(2))
}

// readRecordID ports FUN_1406d3140(_,_,7,_): low = R(idLowBits)+idBase ; tag = R(2)
// in bits 30-31. The slot (id & 0x3fffffff) indexes the World; the tag is generation.
func readRecordID(br *Lecteur, idLowBits int, idBase uint32) uint32 {
	var low uint32
	if idLowBits > 0 {
		low = uint32(br.ReadBits(uint(idLowBits)))
	}
	low += idBase
	tag := uint32(br.ReadBits(2))
	return (tag << 30) | (low & 0x3fffffff)
}

// decodeDelta ports FUN_141f86b58: a delta carries NO typeIndex and NO default-state.
// The archetype is resolved from the World (entity created earlier); the base state
// lives in RAM. The bitstream content is the BASELINE selector + the component mask +
// the present components (the iterator FUN_14076cb60 == consumeMask + the shared loop).
//
// BASELINE (FUN_1406cdc04) : [R(1) ; si 1 -> R(7)] est lu ENTRE le tag d'id et le masque.
// Il manquait a ce port ; sa presence est prouvee contre l'oracle Rosette du film 000d5950
// (mode `oracle` de cmd/tmp_ecsschema) : eid + indices du masque + position de fin d'en-tete
// concordent 54760/54760 AVEC ce bit, et le bit vaut 0 sur 54760/54760 (jamais le R(7)).
func decodeDelta(br *Lecteur, w *World, slot uint32) EntityTrace {
	br.poserSlotDeCapture(slot) // cible d'accumulation i0 pour ce record (no-op sans World accumulateur)
	t := EntityTrace{DesyncAt: -1}
	if br.ReadBit() { // baseline selector
		br.Skip(7)
	}
	typeIndex, ok := w.ArchetypeForSlot(slot)
	if !ok {
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	t.TypeIndex = typeIndex
	arch, ok := w.Reg.Archetype(int(typeIndex))
	if !ok {
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	t.Mask = consumeMask(br) // FIRST and only header read of a delta (no R6/default-state/gate).
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// TryDeltaAt attempts to decode a single DELTA record starting at bit `bitpos` of buf,
// resolving the archetype from w. Returns the record, the bit position immediately after
// it, and whether it decoded cleanly (a recDelta header + a fully-traversed delta,
// DesyncAt==-1). Used by the resync scanner (scanForTargetDelta) to relocate the next
// target record after a desync WITHOUT binding the intervening unknown slots. A fresh
// reader is created so the caller's stream position is untouched.
func TryDeltaAt(buf []byte, bitpos int, w *World, cfg FrameConfig) (FrameRecord, int, bool) {
	br := LecteurSur(buf)
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3)
	br.Skip(bitpos)
	if br.Remaining() < 24 {
		return FrameRecord{}, bitpos, false
	}
	if cfg.HasExtraFields {
		br.Skip(32)
	}
	typ := readRecordType(br)
	if typ != recDelta {
		return FrameRecord{}, bitpos, false
	}
	id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
	slot := id & 0x3fffffff
	rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1}
	rec.Trace = decodeDelta(br, w, slot)
	rec.TypeIndex = rec.Trace.TypeIndex
	rec.DesyncAt = rec.Trace.DesyncAt
	return rec, br.BitPos(), rec.DesyncAt == -1
}

// DecodeFrameRecords decodes the type-0 FRAME record loop until type==0, maintaining
// the World (entity-id -> archetype, held-weapon cache). Returns the records decoded;
// on a component desync it returns the records so far plus an error (the bit position
// of the next record can no longer be trusted).
func DecodeFrameRecords(br *Lecteur, w *World, cfg FrameConfig) ([]FrameRecord, error) {
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3) : le lecteur vient de l'appelant
	var out []FrameRecord
	if cfg.PacketPreambleBits > 0 && br.BitPos() == 0 {
		br.Skip(cfg.PacketPreambleBits) // amorce de paquet (cf. FrameConfig.PacketPreambleBits)
	}
	for {
		hdrBit := br.BitPos()
		if cfg.HasExtraFields {
			br.Skip(32) // optional per-record R(32) prefix, discarded.
		}
		typ := readRecordType(br)
		if typ == recEnd {
			return out, nil // clean end of frame.
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		slot := id & 0x3fffffff
		br.poserSlotDeCapture(slot) // attribue les samples i0 (et l accumulation) au slot du record
		rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1, HeaderBit: hdrBit}

		switch typ {
		case recNew:
			if cfg.HasExtraFields && br.ReadBit() {
				br.Skip(8)
			}
			// NEW = R(6) typeIndex + default-state (in stream) + gate + mask + comps.
			rec.Trace = TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			rec.TypeIndex = rec.Trace.TypeIndex
			rec.DesyncAt = rec.Trace.DesyncAt
			// Bind only on a FULLY clean traversal. Binding on a desyncing NEW (using the
			// R6 typeIndex, which is read before the default-state) was tried and REGRESSED
			// clean frames 18060->6718: the decoder still produces false-cleans (unported
			// components skipped at approximate widths preserve a calibrated total), so a
			// NEW reached after a false-clean is bit-misaligned and its typeIndex binds the
			// slot to a WRONG archetype, overwriting correct world_dump bindings and
			// corrupting all of that slot's future DELTAs. Opportunistic binding is unsafe
			// until the default-state deser is bit-exact (handoff L3 "T3" wall).
			if rec.DesyncAt == -1 {
				w.BindFull(id, rec.TypeIndex)
			}
		case recDel:
			if cfg.HasExtraFields && br.ReadBit() {
				br.Skip(8)
			}
			br.Skip(32) // unconditional R(32) for DEL.
			w.Unbind(slot)
		case recDelta:
			// Rejet DUR sur l'eid COMPLET (generation comprise) quand le mode strict est actif :
			// FUN_1406caad8 compare entry[+0x00] a l'eid ENTIER et renvoie 3 sans lire le corps,
			// ce qui fait `break` a FUN_1406cd128. Un delta qui echoue ce test est donc la preuve
			// d'une lecture fausse, pas un record a decoder.
			if !w.GenerationMatches(id, cfg.Profil.Grammaire.GenerationStricte) {
				rec.Trace = EntityTrace{DesyncAt: 0, EndBit: br.BitPos()}
				rec.DesyncAt = 0
				break
			}
			rec.Trace = decodeDelta(br, w, slot)
			rec.TypeIndex = rec.Trace.TypeIndex
			rec.DesyncAt = rec.Trace.DesyncAt
		default:
			return out, fmt.Errorf("invalid record type %d at bit %d", typ, br.BitPos())
		}

		out = append(out, rec)
		if rec.DesyncAt != -1 {
			return out, fmt.Errorf("desync record slot=%d typeIdx=%d at component i%d (bit %d)",
				slot, rec.TypeIndex, rec.DesyncAt, br.BitPos())
		}
	}
}
