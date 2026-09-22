package grammar

import (
	"context"
	"log/slog"
)

// World tracks entity-id -> archetype (and the last resolved position) ACROSS FRAME records.
// A FRAME delta (type-3) carries NO typeIndex: it must resolve its archetype from the
// entity created earlier by a NEW record (or by the keyframe init). This mirrors the
// game's runtime datum store (FUN_1406cb5f0: per-id slot table, stride 0xa0, the
// archetype index living at slot+8). We materialize it as a simple per-slot map.
//
// Key = id & 0x3fffffff (the slot index). The full id also carries a 2-bit
// generation/namespace tag in bits 30-31, but the decoder keys ONLY on the slot index:
// a slot's identity is reset whenever a NEW record rebinds it (BindFull/BindSoft rewrite
// the whole slotState, PosValid=false). No separate generation guard is maintained — a
// well-formed stream only changes a slot's occupant via such a rebind.
//
// UN CACHE « ARME EN MAIN » A VECU ICI, ET IL A ETE RETIRE le 2026-08-18 (regle 0 code mort).
// `SetHeldWeapon` / `HeldWeapon` recopiaient la variante `weapon-state-type-info` (i43-i46) de
// chaque record propre dans le slot. Le cache n avait AUCUN appelant, et la mesure qui aurait pu
// lui en donner un l a REFUTE : sur trois films CTF, 68 284 lectures d arme tenue, dont 96 a
// 100 % valent la variante NULLE sur les slots que le pont bipede nomme, et zero occurrence du
// marqueur de portage du drapeau (plan `.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`,
// phase 0, item 0.2). Le canal du porteur publie par la phase 1 passe par les IMAGES-CLES.
type World struct {
	Reg   *Registry
	slots map[uint32]slotState
	// vueCourante : l index de la VUE de replication en cours de marche (0, 1 ou 2).
	//
	// LE JEU A TROIS TABLES D ENTITES, UNE PAR VUE, ET UNE ENTITE N APPARTIENT QU A UNE
	// (`FUN_142987460` : `param_1 + 0x228` donne trois objets de vue, et `FUN_1406cd128` teste
	// `vue[0x38][slot].eid == eid` avant de lire un corps de delta). Le monde hors ligne n a
	// qu une table ; cette variable, plus [slotState.Vue], en tiennent lieu — chaque liaison
	// retient DANS QUELLE VUE elle a ete posee, et [World.VuePossede] rend la garde.
	vueCourante int8
	// nsImageCle : le RANG DE VUE que les images-cles de ce film declarent, ou
	// `nsImageCleInconnu` avant la premiere liaison d image-cle. Voir
	// [World.vueDeLEspaceDeNoms].
	nsImageCle int8
	// anticipee / chunkCourant / anticipations : LE REPLI DU LOT 5.23, cf.
	// [World.LierParAnticipation]. `anticipee` nil = repli absent, et rien ne change.
	anticipee     *TableAnticipee
	chunkCourant  int
	anticipations map[uint32]int
	// anticipationDite : le journal du premier usage a-t-il ete ecrit pour ce monde ?
	anticipationDite bool
}

// nsImageCleInconnu : la valeur de [World.nsImageCle] avant toute liaison d image-cle.
const nsImageCleInconnu int8 = -1

type slotState struct {
	TypeIndex uint32
	FullID    uint32 // eid COMPLET (generation 2 bits comprise) tel que pose par le NEW / le keyframe
	Soft      bool   // inferred binding (chain inference), not a dump/NEW ground truth
	// GenAny : la GENERATION de cette liaison est inconnue (aucun keyframe ni NEW ne l'a
	// posee ; cf. BindWildcard). Le test strict de generation est alors neutralise pour ce
	// slot — sans quoi la liaison serait inutilisable, l'eid complet du flux ne pouvant pas
	// etre devine.
	GenAny bool
	// Pos / PosValid : dernière position ABSOLUE RÉSOLUE de l'entité (i0 object-position).
	// Le deser i0 (FUN_1406cfe44) réplique la position en DELTA (prev + delta) sur la majorité
	// des frames ; seuls les chemins ABSOLUS (keyframe/predFlag==1/fallback) posent une valeur
	// fraîche. On mémorise donc la position par slot pour accumuler les deltas (sinon un delta
	// isolé n'est pas une coordonnée). PosValid=false = pas encore de seed absolu (trou attendu
	// avant la 1re absolue d'un slot). Un rebind (BindFull/BindSoft) repart PosValid=false,
	// ce qui purge déjà la position d'un slot recyclé — pas de garde de génération séparée.
	Pos      [3]float32
	PosValid bool
	// Vue : l index de la vue de replication qui possede cette entite, ou -1 quand il est
	// INCONNU (liaison posee par une image-cle : le keyframe ne dit pas la vue). Une vue
	// inconnue ne rejette rien, exactement comme [slotState.GenAny] pour la generation.
	Vue int8
}

// vueInconnue : la valeur de [slotState.Vue] quand aucune vue n a pu etre attribuee.
const vueInconnue int8 = -1

// NewWorld creates an empty World bound to a parsed archetype registry.
func NewWorld(reg *Registry) *World {
	return &World{Reg: reg, slots: map[uint32]slotState{}, nsImageCle: nsImageCleInconnu,
		anticipations: map[uint32]int{}}
}

// --- LE REPLI DU LOT 5.23 : LA LIAISON PAR ANTICIPATION -------------------------------------
//
// CE QUE C EST, ET CE QUE CE N EST PAS. Une entite nee en MILIEU de chunk n est declaree ni par
// l image-cle de son chunk (0,0 % — 5.20.2, recherche exhaustive a toutes les positions de bit)
// ni par le bloc de type 1 de ce chunk (0 sur 23 325 rejets — 5.21.2), et le decodeur ne lit
// AUCUN record `NEW` la ou elle nait. **Le record de naissance reste NON LU** : ce repli ne le
// remplace pas, il le rend inutile pour la SUITE du flux. Il est NOMME, DATE (2026-09-22) et
// COMPTE — [Observation.LiaisonsParAnticipation], par archetype, a cote de `RejetsHorsDatum`.
//
// SUR QUELLE FOI. L image-cle du chunk SUIVANT declare 74,7 % des slots rejetes, avec leur
// archetype, sous la cle MEME que `FUN_1406caad8` compare (l eid entier — cf.
// `keyframe_anticipe.go`). Anticiper, c est lier un slot sur la foi d une image-cle ULTERIEURE.
//
// La liaison posee est celle de la table de datums ([World.BindDatum]) : `Soft`, `GenAny`, vue
// INCONNUE, sans position. Elle ne dit que ce qu elle sait — l archetype.

// PoserTableAnticipee installe (ou retire, avec nil) la table anticipee du film. Sans elle
// [World.LierParAnticipation] ne pose rien et la marche se comporte exactement comme avant.
func (w *World) PoserTableAnticipee(t *TableAnticipee) { w.anticipee = t }

// PoserChunkCourant annonce le chunk que la marche parcourt : la table anticipee ne rend qu une
// declaration STRICTEMENT POSTERIEURE a lui.
func (w *World) PoserChunkCourant(n int) { w.chunkCourant = n }

// LierParAnticipation lie le slot de l eid `id` a l archetype que la premiere image-cle
// POSTERIEURE au chunk courant lui donne, et rend cet archetype. `false` = la table est absente,
// le slot est deja lie, ou aucune image-cle ulterieure ne le declare.
func (w *World) LierParAnticipation(id uint32) (uint32, bool) {
	if w.anticipee == nil {
		return 0, false
	}
	slot := id & 0x3fffffff
	if _, lie := w.slots[slot]; lie {
		return 0, false
	}
	ti, declarant, ok := w.anticipee.ArchetypeApres(id, w.chunkCourant)
	if !ok {
		return 0, false
	}
	w.BindDatum(slot, ti)
	if w.anticipations == nil {
		w.anticipations = map[uint32]int{}
	}
	w.anticipations[ti]++
	if !w.anticipationDite {
		w.anticipationDite = true
		// Pas de `ctx` ici : le monde n en porte pas, et `registry_fingerprint.go` a le meme
		// besoin — meme geste, meme raison.
		slog.InfoContext(context.Background(),
			"liaison par anticipation : le repli du lot 5.23 est ACTIF sur ce film",
			"slot", slot, "archetype", ti, "chunk", w.chunkCourant, "declarant", declarant,
			"cles_de_la_table", w.anticipee.Entrees())
	}
	return ti, true
}

// AnticipationsParArchetype rend, par archetype, le nombre de liaisons que le repli a posees.
func (w *World) AnticipationsParArchetype() map[uint32]int { return w.anticipations }

// PoserVueCourante annonce au monde la vue de replication que la marche parcourt.
func (w *World) PoserVueCourante(v int) { w.vueCourante = int8(v) } //nolint:gosec // v vaut 0..2

// VuePossede rend la garde de TABLE DE VUE : la vue en cours possede-t-elle ce slot ?
//
// C EST LA TRANSCRIPTION DE `vue[0x38][slot].eid == eid` (FUN_1406cd128), avec la seule
// difference que le monde hors ligne ne peut pas TOUJOURS attribuer une vue : une liaison venue
// d une image-cle n en porte pas. Ces liaisons-la passent, comme les generations inconnues.
//
// **ET CETTE TRANSCRIPTION EST CELLE DE LA BRANCHE QUE LE JEU N EMPRUNTE PAS SUR UN FILM**
// (lot 5.15, 2026-09-22). `FUN_1406cd128` se scinde sur `DAT_14474cd78`, qui VAUT 1 dans
// l image : la branche vive (`!= 0`) passe par `FUN_1406cbaa0`, dont le prologue AGRANDIT
// `vue[0x38]` au lieu de rejeter, et dont la garde de delta porte sur la table de datums du
// decodeur partage (`*(vue+0x20) + 0x20`, pas de 200). Le seul ecrivain de `DAT_14474cd78 = 0`
// est `FUN_1428e24bc`, un aller-retour d etat qui pre-remplit la table puis restaure le global.
// Consequence MESUREE sur `bfecd02b` : 21 988 des 22 112 rejets portent sur un slot jamais lie,
// ZERO sur un slot lie ailleurs. Voir `.ai/V7.5/film_re/NOTE_5_15_RANG_1_FILM_DENSE_2026-09-22.md`
// §3 et le report 5.15.3 — le correctif est un MODELE (la table de datums par slot), pas une
// garde de plus.
//
// ELLE NE COMPARE PAS L EID COMPLET, ET C EST MESURE (lot 5.13.1). Le comparer — exiger que les
// deux bits de tete du delta valent ceux que la liaison porte — coute 21 records `ti=35` sur
// `bfecd02b` (114 458 -> 114 437), parce que les deux bits de tete du flux DELTA et ceux de
// l image-cle NE SONT PAS LE MEME CHAMP : voir [World.BindImageCle].
func (w *World) VuePossede(slot uint32) bool {
	s, ok := w.slots[slot&0x3fffffff]
	// UN SLOT NON LIE N EST POSSEDE PAR AUCUNE VUE, et c est l ecrivain qui le dit : le vecteur
	// de la vue est construit entree par entree (`FUN_1408f15c8`), une entree jamais posee porte
	// `eid = 0`, donc `entry.eid != eid` et la vue s arrete. C est ce qui remplace l inference
	// d archetype sur ce chemin — elle fabriquait un corps la ou le jeu ne lit rien.
	return ok && (s.Vue == vueInconnue || s.Vue == w.vueCourante)
}

// BindFull registers (or rebinds) the entity created by a NEW record: slot -> archetype.
// L'eid COMPLET (generation en bits 30-31) est memorise : le jeu teste `entry[+0x00] != eid` sur
// l'eid entier avant de lire le corps d'un delta (FUN_1406caad8 -> `return 3`), donc la generation
// est une contrainte de validite gratuite. Elle n'est appliquee que si SetStrictGeneration(true).
func (w *World) BindFull(id, typeIndex uint32) {
	w.slots[id&0x3fffffff] = slotState{TypeIndex: typeIndex, FullID: id, Vue: w.vueCourante}
}

// strictGeneration exige que l'eid COMPLET d'un delta (tag de generation inclus) corresponde a
// celui pose au binding, comme le fait `FUN_1406caad8` (`entry[0] != eid` -> return 3, corps NON
// lu, boucle abandonnee). Defaut false : les bindings issus des keyframes portent une generation
// qui peut avoir change depuis, et le mode strict doit rester un A/B mesurable.
//
// C EST UNE BASCULE DU PROFIL DE BALAYAGE DEPUIS LE LOT 2.3
// ([GrammaireBalayage.GenerationStricte]), plus une variable de paquet : elle arrive donc par le
// meme canal que les largeurs, et `killsource` la passe a la cuisson du rejeu comme le reste de
// ce qu il retient.

// GenerationMatches indique si l'eid complet `id` correspond a celui memorise pour son slot.
// Toujours vrai quand le mode strict est desactive ou quand aucune generation n'a ete memorisee.
func (w *World) GenerationMatches(id uint32, strict bool) bool {
	if !strict {
		return true
	}
	s, ok := w.slots[id&0x3fffffff]
	return ok && (s.GenAny || s.FullID == id)
}

// BindWildcard enregistre une liaison slot -> archetype dont la GENERATION est INCONNUE.
// Seul usage legitime : un slot qu'aucun keyframe et aucun record NEW propre ne declare,
// mais dont l'archetype est etabli par une contrainte structurelle externe (par exemple une
// entite creee ET detruite entre deux keyframes, dont la plage de slots et la fenetre de vie
// sont deduites de l'ordre d'allocation des keyframes). Le test strict de generation est
// neutralise pour ce slot, l'eid complet ne pouvant pas etre devine.
func (w *World) BindWildcard(slot, typeIndex uint32) {
	w.slots[slot&0x3fffffff] = slotState{
		TypeIndex: typeIndex, FullID: slot & 0x3fffffff, GenAny: true, Vue: 0,
	}
}

// vueDeLImageCle : LE RANG DE VUE que porte une liaison d image-cle dans la marche hors ligne.
//
// # LES DEUX BITS DE TETE D UN IDENTIFIANT D IMAGE-CLE SONT LE RANG DE LA VUE
//
// La liste de reference qu un paquet d image-cle transporte est celle d UNE vue : son ecrivain
// `FUN_142f2e174` est un slot de la vtable de VUE (`0x1436a87e0` + 0x10), il ne parcourt que la
// table de SA vue (`vue+0x38` a `vue+0x40`, bitmap `vue+0x58`), et il met les DEUX BITS DE TETE
// de chaque identifiant a `vue + 8` — `142f2e2ec MOV ECX, dword ptr [RDI + 0x8]` puis
// `142f2e304 SHL ECX, 0x1e`, et de meme sur ses deux autres sites de genre.
//
// ET `vue + 8` EST LE RANG DE LA VUE : c est le registraire `FUN_1409c9860(conteneur, rang, vue)`
// qui l ecrit, `*(int *)(param_3 + 1) = param_2`, au moment ou il range la vue dans le tableau
// que `FUN_142987460` parcourt. Ce n est donc NI une generation NI un identifiant d entite.
//
// MESURE (`TestImageCle513Vues`) : UN SEUL rang par paquet d image-cle, et il vaut 1 sur les
// 5 paquets de `dad793c7` (123 a 186 records) comme sur les 60 paquets de `bfecd02b` (424 a 482
// records). Le rang 1 est celui ou `FUN_141f855b4` enregistre la vue du GESTIONNAIRE D ENTITES
// (`0x1436a87e0`) — la seule des trois classes de vue dont la boucle de records est
// `FUN_1406cd128`, c est-a-dire la seule grammaire que la marche hors ligne porte.
//
// CORRESPONDANCE AVEC LE RANG DE LA MARCHE HORS LIGNE : la vue que la marche parcourt en PREMIER
// est celle qui rend les records — sur `dad793c7` ses 5 628 records, 5 628 sur 5 628 ; sur
// `bfecd02b`, 157 250 sur 157 554 (`TestVues513EspaceDeNoms`). C est donc la vue du gestionnaire
// d entites, celle que l image-cle enumere. LE DECALAGE ENTRE LES DEUX NUMEROTATIONS EST ETABLI
// DEPUIS LE LOT 5.14.1 : `FUN_141f855b4` enregistre le gestionnaire d entites au RANG 1 du film
// (`FUN_1409c9860(conteneur+8, 1, conteneur + 0x21b70)`), et le monde hors ligne le range en 0 —
// cette constante EST ce decalage, et `decodeFrameParRangs` marche la vue B sous elle. Le port
// retient neanmoins le PREMIER rang rencontre (cf. [World.vueDeLEspaceDeNoms]) : un second rang
// dans une image-cle serait une vue que rien ne situe, et les deux films temoins sont mono-rang.
//
// « Toutes les liaisons d image-cle vont en vue 0 » n est donc pas une limite du portage : c est
// ce que le film porte, et l image-cle le DIT au lieu qu on le suppose.
const vueDeLImageCle int8 = 0

// BindImageCle enregistre la liaison portee par un record de la table d IMAGE-CLE : elle LIT la
// vue dans les deux bits de tete de l identifiant, la ou `BindWildcard` les jetait.
//
// LES DEUX BITS DE TETE D UNE IMAGE-CLE ET CEUX D UN FLUX DELTA NE SONT PAS LE MEME CHAMP, et
// c est la lecon du lot 5.13.1 (le journal RE du lot G les nommait tous deux `gen`) :
//
//	image-cle   : `vue + 8`, le RANG de la vue (`FUN_142f2e174` / `FUN_1409c9860`) ;
//	flux delta  : l eid que la table de la vue porte (`FUN_142f30610` :
//	              `*(uint *)(slot * 0xa0 + 8 + vue[0x38])`), pose par `FUN_1408f1730` a
//	              `*(byte *)(datum + 1) << 0x1e | slot` — un champ du DATUM, par entite.
//
// D ou la forme de cette liaison : la VUE est connue, la GENERATION reste INCONNUE (`GenAny`).
// Confondre les deux — exiger que les deux bits de tete d un delta valent ceux de l image-cle —
// coute 21 records `ti=35` sur `bfecd02b` (114 458 -> 114 437) : mesure du lot, cf.
// [World.VuePossede].
//
// `ns` est le rang lu (`KeyframeRec.Gen`), et il SERT : il nomme la vue.
func (w *World) BindImageCle(ns, slot, typeIndex uint32) {
	w.slots[slot&0x3fffffff] = slotState{
		TypeIndex: typeIndex, FullID: slot & 0x3fffffff, GenAny: true,
		Vue: w.vueDeLEspaceDeNoms(ns),
	}
}

// vueDeLEspaceDeNoms rend le RANG DE VUE de la marche hors ligne que nomme un rang d image-cle.
//
// Le film enregistre UNE vue — celle du gestionnaire d entites — et la marche la parcourt en
// premier : le PREMIER rang rencontre est le sien, donc `vueDeLImageCle`. Un second rang serait
// une vue que rien ne situe dans l ordre de la marche ; sa liaison passe alors en vue INCONNUE,
// qui ne rejette rien, plutot que d etre attribuee d office a la vue de rang 0 comme
// `BindWildcard` le faisait. Sur les films temoins ce second cas ne se presente pas (un seul rang
// par image-cle), et c est pour cela que ce port ne deplace aucune mesure : il remplace une
// attribution d office par une LECTURE.
func (w *World) vueDeLEspaceDeNoms(ns uint32) int8 {
	rang := int8(ns & 3) //nolint:gosec // ns & 3 tient dans int8
	if w.nsImageCle == nsImageCleInconnu {
		w.nsImageCle = rang
		return vueDeLImageCle
	}
	if w.nsImageCle == rang {
		return vueDeLImageCle
	}
	return vueInconnue
}

// BindDatum enregistre une liaison venue de la TABLE DE DATUMS de l image-cle
// (`keyframe_datums.go`) : le SEUL fait qu elle porte est `slot -> archetype`.
//
// C est la forme minimale de ce que la branche vive de la boucle de records interroge
// (`FUN_1406cbaa0`, cas DELTA : garde `*(uint *)(slot * 200 + t) != eid`, archetype lu en
// `+0x04`). Elle est donc :
//
//	`Soft`   — ce n est pas une ancre de confirmation pour l inference de chaine : la table dit
//	           l archetype, pas la frontiere d un record ;
//	`GenAny` — la generation n est pas relue ici (l identifiant complet de l image-cle et les
//	           deux bits de tete d un delta ne sont pas le meme champ, lecon du 5.13.1) ;
//	`Vue` INCONNUE — la table de datums est celle du DECODEUR PARTAGE, pas d une vue.
//
// Elle ne pose AUCUNE position : `PosValid` reste faux, et le premier chemin absolu du slot
// l amorcera comme pour toute autre liaison.
func (w *World) BindDatum(slot, typeIndex uint32) {
	s := slot & 0x3fffffff
	w.slots[s] = slotState{
		TypeIndex: typeIndex, FullID: s, Soft: true, GenAny: true, Vue: vueInconnue,
	}
}

// BindSoft registers an INFERRED slot -> archetype binding (chain inference). Soft
// bindings decode subsequent deltas like hard ones, but are NOT confirmation anchors
// for further inference (a soft anchor could self-confirm a wrong chain).
func (w *World) BindSoft(id, typeIndex uint32) {
	w.slots[id&0x3fffffff] = slotState{
		TypeIndex: typeIndex, FullID: id, Soft: true, Vue: w.vueCourante,
	}
}

// HardBound reports whether slot carries a NON-inferred binding (world dump or clean
// NEW record) — the only bindings chain inference may confirm against.
func (w *World) HardBound(slot uint32) bool {
	s, ok := w.slots[slot]
	return ok && !s.Soft
}

// Unbind removes a slot (DEL record).
func (w *World) Unbind(slot uint32) { delete(w.slots, slot) }

// ArchetypeForSlot returns the archetype typeIndex bound to a slot, if any.
func (w *World) ArchetypeForSlot(slot uint32) (uint32, bool) {
	s, ok := w.slots[slot]
	return s.TypeIndex, ok
}

// SetPos mémorise la position ABSOLUE résolue d'un slot (seed absolu ou prev+delta). No-op si
// le slot n'est pas tracké (un delta sur un slot inconnu n'a pas d'entité à accumuler).
func (w *World) SetPos(slot uint32, v [3]float32) {
	if s, ok := w.slots[slot]; ok {
		s.Pos, s.PosValid = v, true
		w.slots[slot] = s
	}
}

// PosOf retourne la dernière position absolue résolue d'un slot (ok=false si aucun seed encore).
func (w *World) PosOf(slot uint32) ([3]float32, bool) {
	if s, ok := w.slots[slot]; ok && s.PosValid {
		return s.Pos, true
	}
	return [3]float32{}, false
}

// Bound reports how many slots are currently tracked (diagnostics).
func (w *World) Bound() int { return len(w.slots) }

// cloneSlots returns a shallow copy of the slot table (for per-frame rollback).
func (w *World) cloneSlots() map[uint32]slotState {
	m := make(map[uint32]slotState, len(w.slots))
	for k, v := range w.slots {
		m[k] = v
	}
	return m
}

// restoreSlots replaces the slot table (rollback after a desynced frame so a
// partial/garbage NEW/DEL does not corrupt a PERSISTENT World).
func (w *World) restoreSlots(m map[uint32]slotState) { w.slots = m }

// WorldSnapshot is an opaque copy of a World's slot table for rollback across a
// desynced frame (see World.Snapshot / World.Restore). Exported so decode tools can
// run a PERSISTENT World (NEW binds, DEL unbinds across frames, mirroring the game's
// runtime datum store) and roll back only the frames that desync.
type WorldSnapshot struct{ slots map[uint32]slotState }

// Snapshot captures the current slot table (shallow copy) for a later Restore.
func (w *World) Snapshot() WorldSnapshot { return WorldSnapshot{slots: w.cloneSlots()} }

// Restore rewinds the slot table to a prior Snapshot (e.g. to discard a desynced
// frame's partial/garbage NEW/DEL bindings without dropping earlier clean bindings).
func (w *World) Restore(s WorldSnapshot) { w.restoreSlots(s.slots) }
