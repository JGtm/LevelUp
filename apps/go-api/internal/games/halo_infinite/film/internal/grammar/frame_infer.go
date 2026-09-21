package grammar

// frame_infer.go — L INFERENCE D ARCHETYPE SUR UN SLOT NON LIE, et la reprise validee :
// `DecodeFrameInfer`, sa boucle, `inferUnboundArchetype` et `validatedResync`.
//
// Sorti de `frame_records.go` par deplacement pur au lot 2.7 (scission des fichiers de plus
// de 500 lignes) : aucune ligne de logique n'a change. La frontiere est celle du chemin
// « decoder correctement » (deduire l archetype d un transitoire absent du liage) face au
// decodage nominal, qui n en a pas besoin.

// DecodeFrameInfer decodes a FRAME like DecodeFrameRecords but, on a delta for an
// UNBOUND slot (a transient entity absent from the binding dump), it INFERS the slot's
// archetype (inferUnboundArchetype) and skips its body accordingly — decoding the frame
// CORRECTLY past the transient instead of scanning/guessing. This is the offline
// "decode-correctly" alternative to the resync heuristic: no false positives, because it
// only advances when the inference is unambiguous AND confirmed by a clean bound
// successor. Falls back to a desync (returns) when inference is ambiguous/none.
func DecodeFrameInfer(buf []byte, w *World, cfg FrameConfig) ([]FrameRecord, int) {
	br := LecteurSur(buf)
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3)
	out, inferred, _ := decodeInferLoop(br, buf, w, cfg)
	return out, inferred
}

// --- LE PIED DE TRAME ET LA GARDE D EID DE LA VUE — TROU DE GRAMMAIRE NOMME, NON REFERME
// --- (lot 5.11.6, 2026-09-21). A LIRE AVANT DE TOUCHER CETTE BOUCLE.
//
// # CE QUE L ECRIVAIN FAIT, ET QUE CE PAQUET NE FAIT PAS
//
// `FUN_1406cd128` est LA boucle de records, et le frame-processeur `FUN_142987460` l appelle
// UNE FOIS PAR VUE (trois vues, `vtable[0x40]` ; `FUN_142987460` n ecrit RIEN apres elles —
// `FUN_1406d07b0` ne recoit pas le lecteur). Sur un record de type DELTA la boucle ne decode un
// corps QUE si l eid appartient a la table de CETTE vue :
//
//	lVar11 = (eid & 0x3fffffff) * 0xa0
//	if (vue[0x38][lVar11 + 8] == eid && vue[0x38][lVar11 + 2] == (short)type)
//	      FUN_141f86b58(...)   // le corps
//	else  uVar14 = 2           // REJET
//	if (uVar14 != 0) break     // LA VUE S ARRETE, sans avoir lu un seul bit de corps
//
// Une vue qui n a rien a rendre ecrit donc, et lit donc, exactement `1 + idLow + 2` bits : un
// prefixe, un `idLow`, un tag de generation. C EST LA GRAMMAIRE DU PIED DE TRAME.
//
// # CE QUE LE TROU COUTE, MESURE SUR `dad793c7` (lot 5.11.6)
//
// Ce paquet-ci ne porte PAS la garde — `DecodeFrameRecords` la porte depuis le lot 2.3
// (`frame_records.go`, `GenerationMatches`), cette boucle NON, et c est elle que
// `DecodeFrameViews` emprunte, donc toutes les marches du chantier. Sur un slot non lie elle
// appelle l inference d archetype au lieu de s arreter : l inference « trouve » un archetype
// pour l en-tete de la vue SUIVANTE et decode un corps qui n existe pas. Le dernier record rendu
// de chaque paquet de `dad793c7` etait ainsi un `ti=6` a masque vide, FABRIQUE a partir de zeros
// lus AU-DELA de la fin du payload.
//
//	la marche consomme 57 bits de PLUS que le paquet n en porte, sur 95,7 % des paquets
//	le pied vaut un mot de 32 bits, CONSTANT sur 5 197 des 5 221 paquets (99,54 %)
//	les 24 paquets qui s en ecartent sont les 23 paquets a liste d evenements... et le paquet
//	  du DECOLLAGE du saut, seul du film a porter le mot dominant modifie de deux bits
//
// # POURQUOI LA GARDE N EST PAS PORTEE ICI, ET CE QU IL FAUDRAIT POUR LA PORTER
//
// Le monde hors ligne ne porte PAS les trois tables de vues du jeu : `World` est une seule table
// slot -> archetype, reconstruite du film, et l inference d archetype existe pour rattraper les
// liaisons que les images-cles eparses ne donnent pas (lot 5.3.3-b). Deux transcriptions ont ete
// ESSAYEES ET MESUREES, et les deux perdent :
//
//	`GenerationMatches` pose ici comme dans `DecodeFrameRecords` : l A/B du mode strict coute
//	  12 248 records `ti=35` sur `bfecd02b` (97 343 -> 85 095) et ne referme pas le debordement,
//	  parce que la liaison du slot fautif vient d une image-cle (`BindWildcard`, `GenAny`) qui
//	  neutralise le test.
//	le rejet d un corps qui DEPASSE la fin du payload : `replay-equiv -films bcb6d393` rend
//	  `movementStates` 1 364 -> 1 254, une PERTE de 110 transitions.
//
// CE QU IL FAUT EST LE MODELE MANQUANT, ET IL EST NOMME : les TROIS TABLES D ENTITES PAR VUE
// (`param_1 + 0x228` donne trois objets de vue, chacun avec sa table `vue + 0x38`). Une entite
// appartient a UNE vue et une seule ; tant que le monde hors ligne n a qu une table, aucune
// transcription de la garde ne distingue « eid etranger a cette vue » de « slot non encore lie ».
// Le gate du trou est `TestMouvement5116Gate` : bits non lus par paquet dans [0 ; 7].
// --- fin de la note ---------------------------------------------------------------------------
// decodeInferLoop is the core of DecodeFrameInfer operating on a SUPPLIED Lecteur,
// so several replication "views" of one packet (frame-processor FUN_142987460 = a
// leading config bit then 3 view record-loops) can be decoded in sequence sharing one
// reader. Returns the records, the number of inferred transients, and hitEnd = whether
// it stopped on a clean end-of-records marker (true) vs a desync/EOF (false).
//
// EXEMPTION DE LONGUEUR (114 lignes, seuil 80 — CLAUDE.md regle 5, examinee au lot 2.7 le
// 2026-09-16). Raison STRUCTURELLE : c est une boucle de decodage a sortie multiple. Chaque
// branche du `switch` sur le type de record ou bien `continue` (le stall a recupere le
// curseur), ou bien rend les trois valeurs et arrete la trame ; la fermeture `stall` capture
// `out` et `br`. Sortir une branche dans une fonction demanderait de rendre a l appelante un
// SIGNAL de controle a re-interpreter — un aiguillage de plus la ou il y en a deja trois, et
// une reecriture de la logique, pas un deplacement. Le lot 2.7 est un lot de deplacement pur :
// il ne touche pas a cette boucle. Reexamen au lot 3.6 (ports de composants), qui la rouvre.
func decodeInferLoop(br *Lecteur, buf []byte, w *World, cfg FrameConfig) ([]FrameRecord, int, bool) {
	var out []FrameRecord
	inferred := 0
	frameLen := len(buf) * 8
	guard := 0
	// stall appends the desynced record and attempts a validated resync recovery
	// (inferResyncTargets set). Returns true = stop (unrecoverable desync), false =
	// recovered (br repositioned, keep looping).
	stall := func(startPos int, rec FrameRecord) bool {
		out = append(out, rec)
		if inferResyncTargets != nil {
			if next, ok := validatedResync(buf, startPos+1, w, cfg); ok {
				br.SetBitPos(next)
				return false
			}
		}
		return true
	}
	for br.BitPos() < frameLen {
		guard++
		if guard > 8192 {
			return out, inferred, false
		}
		startPos := br.BitPos()
		if cfg.HasExtraFields {
			br.Skip(32)
		}
		typ := readRecordType(br)
		if typ == recEnd {
			return out, inferred, true
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		slot := id & 0x3fffffff
		rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1}
		switch typ {
		case recNew:
			bodyStart := br.BitPos()
			rec.Trace = TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			rec.TypeIndex, rec.DesyncAt = rec.Trace.TypeIndex, rec.Trace.DesyncAt
			repaired := false
			if rec.DesyncAt != -1 && cfg.Profil.Grammaire.InferenceChaine && inferRepair {
				if t, end, ok := repairUnportedComponent(buf, bodyStart, recNew, slot, rec.Trace, w, cfg); ok {
					rec.Trace, rec.TypeIndex, rec.DesyncAt, repaired = t, t.TypeIndex, -1, true
					br.SetBitPos(end)
				}
			}
			switch {
			case rec.DesyncAt != -1:
				if stall(startPos, rec) {
					return out, inferred, false
				}
				continue
			case repaired:
				w.BindSoft(id, rec.TypeIndex)
			default:
				w.BindFull(id, rec.TypeIndex)
			}
		case recDel:
			br.Skip(32)
			w.Unbind(slot)
		case recDelta:
			if _, bound := w.ArchetypeForSlot(slot); bound {
				bodyStart := br.BitPos()
				rec.Trace = decodeDelta(br, w, slot)
				rec.TypeIndex, rec.DesyncAt = rec.Trace.TypeIndex, rec.Trace.DesyncAt
				if rec.DesyncAt != -1 && cfg.Profil.Grammaire.InferenceChaine && inferRepair {
					if t, end, ok := repairUnportedComponent(buf, bodyStart, recDelta, slot, rec.Trace, w, cfg); ok {
						rec.Trace, rec.DesyncAt = t, -1
						br.SetBitPos(end)
					}
				}
				if rec.DesyncAt != -1 {
					if stall(startPos, rec) {
						return out, inferred, false
					}
					continue
				}
			} else {
				var (
					ti   uint32
					end  int
					uniq bool
					ok   bool
				)
				if cfg.Profil.Grammaire.InferenceChaine {
					ti, end, uniq, ok = inferChainArchetype(buf, br.BitPos(), w, cfg)
				} else {
					ti, end, ok = inferUnboundArchetype(buf, br.BitPos(), w, cfg)
				}
				if !ok {
					rec.DesyncAt = 0
					if stall(startPos, rec) {
						return out, inferred, false
					}
					continue
				}
				if cfg.Profil.Grammaire.InferenceChaine && uniq {
					w.BindSoft(id, ti)
				}
				rec.TypeIndex = ti
				inferred++
				br.SetBitPos(end)
			}
		default:
			rec.DesyncAt = 0
			if stall(startPos, rec) {
				return out, inferred, false
			}
			continue
		}
		out = append(out, rec)
	}
	return out, inferred, false
}

// inferResyncTargets, when non-nil, enables validated-resync recovery in
// DecodeFrameInfer: on an unresolvable desync, scan forward for a landing where a
// clean delta on one of these slots decodes AND the chain walker confirms continued
// clean decoding (a hard-bound clean delta downstream or a flush end-of-frame). This
// reaches biped records stranded in a desync tail with far fewer false positives than
// raw DecodeFrameResync (which accepts any clean target delta). Nil = no recovery.
// Le reglage public `SetInferResyncTargets` a ete supprime le 2026-09-05 (lot E, item E.2) :
// aucun appelant. La table reste nil, c est-a-dire pas de recuperation par resync valide.
var inferResyncTargets map[uint32]bool

// validatedResync scans buf forward from bit `from` for the first landing where a
// clean delta on a target slot decodes AND the chain walker confirms that decoding
// continues cleanly from just after it (reaching a hard-bound clean delta or a flush
// end-of-frame within the walk budget). Returns the landing bit (the START of the
// confirming target record, so the caller re-decodes it WITH capture hooks live) and
// ok. The confirmation is the same structural test chain inference uses, so a
// coincidental target-slot delta that does not lead to sustained clean decode is
// rejected — the discriminator raw resync lacked.
func validatedResync(buf []byte, from int, w *World, cfg FrameConfig) (int, bool) {
	defer cfg.Obs.neutraliserCaptures()()

	frameLen := len(buf) * 8
	c := &chainCtx{buf: buf, frameLen: frameLen, w: w, cfg: cfg,
		budget: chainTrialBudget, minComps: chainConfirmMinComps, needConfirms: 1,
		overlay: map[uint32]uint32{}}
	for b := from; b < frameLen-24; b++ {
		rec, after, ok := TryDeltaAt(buf, b, w, cfg)
		if !ok || !inferResyncTargets[rec.Slot] || len(rec.Trace.Comps) < 1 {
			continue
		}
		// The landing itself is a clean hard-bound target delta; require the CONTINUATION
		// to confirm so a coincidental clean delta (followed by garbage) is rejected.
		if c.budget <= 0 {
			return 0, false
		}
		if c.confirmChainAt(after, chainMaxDepth-1, chainMaxRecords) {
			cfg.Obs.compterResyncValide()
			return b, true
		}
	}
	return 0, false
}

// inferUnboundArchetype tries every registered archetype for the unbound-slot delta at
// bit `bitpos`; a candidate qualifies if its body decodes cleanly AND the NEXT record is
// a clean delta on a bound slot (alignment confirmed). Returns the typeIndex + end bit
// iff EXACTLY ONE archetype qualifies (unambiguous), else ok=false. Hooks are suppressed
// during the trials (no spurious samples). This is the offline "decode-correctly" path
// for transient entities absent from the binding dump.
// inferRequireBoundSuccessor gates the strong confirmation (next record = clean bound
// delta). Default true (correct, but blocked by transient CHAINS). Set false to infer on
// unambiguity ALONE — reaches through chains; a wrong choice merely desyncs the frame
// (stops), it does NOT fabricate a biped position (unlike the resync scan). Le reglage public
// `SetInferStrict` a ete supprime le 2026-09-05 (lot E, item E.2) : aucun appelant. La
// confirmation forte reste active, comme en production.
// PROVENANCE : la confirmation forte est le comportement de production, et le seul mesure.
// Constante depuis le 2026-09-06 (lot E, item E.8).
const inferRequireBoundSuccessor = true

// C'ÉTAIT LA VARIABLE DE PAQUET `inferChain` JUSQU'AU LOT 2.3 : l'aiguillage vers le resolveur
// RECURSIF vit dans [GrammaireBalayage.InferenceChaine], que le cadre du balayage porte.

// inferRepair enables component-width inference (repairUnportedComponent) on records
// that desync on an un-ported component. Off by default: it is expensive and mostly
// rescues non-biped transients (its true value is the per-component width observations
// it accumulates for porting). Requires cfg.Profil.Grammaire.InferenceChaine. Le reglage public `SetInferRepair` a ete
// supprime le 2026-09-05 (lot E, item E.2) : aucun appelant. La reparation reste desactivee.
// JAMAIS ACTIVE : aucun chemin ne l a jamais mis a vrai. Constante depuis le 2026-09-06
// (lot E, item E.8).
const inferRepair = false

func inferUnboundArchetype(buf []byte, bitpos int, w *World, cfg FrameConfig) (uint32, int, bool) {
	// LES DEUX CROCHETS SONT NEUTRALISES POUR LA DUREE DES ESSAIS : ce qui suit est une lecture
	// SPECULATIVE (chaque archetype du registre est essaye sur les memes bits), et une lecture
	// speculative n est pas une lecture. Sans cette neutralisation, les tentatives abandonnees
	// deposent des valeurs a des positions que la traversee retenue ne lit jamais — elles
	// seraient attribuees a un composant au hasard.
	defer cfg.Obs.neutraliserCaptures()()

	var winTi uint32
	winEnd, matches := -1, 0
	for ti := range w.Reg.Archetypes {
		arch := w.Reg.Archetypes[ti]
		br := LecteurSur(buf)
		br.poserCadre(cfg)
		br.Skip(bitpos) // bitpos = delta body start (mask), already past type+id
		t := decodeDeltaWithArch(br, arch, uint32(ti))
		if t.DesyncAt != -1 {
			continue
		}
		if !inferRequireBoundSuccessor || boundDeltaCleanAt(buf, br.BitPos(), w, cfg) {
			matches++
			winTi, winEnd = uint32(ti), br.BitPos()
			if matches > 1 {
				return 0, bitpos, false // ambiguous
			}
		}
	}
	if matches == 1 {
		return winTi, winEnd, true
	}
	return 0, bitpos, false
}
