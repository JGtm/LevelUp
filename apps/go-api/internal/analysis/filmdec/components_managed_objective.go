package filmdec

// components_managed_objective.go — L'ARCHETYPE DES OBJECTIFS GERES : ti=11.
//
// # CE QUE ti=11 EST, ET POURQUOI IL A ATTENDU
//
// C'est l'objectif tel que le HUD le suit : un minuteur, un type, un etat, une reference vers
// l'objet physique, et surtout une PROGRESSION avec son SEUIL — la jauge de capture. Il etait
// couvert 0 sur 34 par `consumeByName` : deux deserialiseurs existaient
// (`consumeObjectiveFormattedText`, i2 et i9) mais sans appelant, gardes depuis le 2026-08-01
// sous la condition de retrait « branchee ou supprimee quand ti=11 sera decode ». Ce fichier
// leve cette condition.
//
// # POURQUOI MAINTENANT : LE MIROIR
//
// Le chantier Assaut cherchait la jauge d'armement de la bombe et l'a cherchee dans `ti=13`
// (`managed-object-property-*`), ou vivent les jauges de Strongholds et de KOTH. Negatif net :
// 0 a 2 valeurs de tag 3 en Assaut contre 4 397 chez un temoin Strongholds. La cause est un
// MIROIR, pas une absence — les deux archetypes se partagent les modes :
//
//	Strongholds / KOTH   ti=13 riche (26 a 52 slots)   ti=11 vide (0 a 12 records, masques nuls)
//	Assaut               ti=13 pauvre (8 slots)        ti=11 riche (136 a 616 records)
//
// # LE DEVIS EST MESURE, PAS SUPPOSE (recensement du 2026-09-01, 13 films)
//
// La boucle de composants est SEQUENTIELLE : pour LIRE i12 il faut porter tout ce qui est
// PRESENT avant lui. `objectif_ti11_masque_test.go` a releve les masques sans decoder aucun
// corps — sur les 265 records qui portent la jauge, les composants qui la precedent sont
// i0 (100 %), i7 (14 %), i8 (13 %), i9 (8 %), i2 et i6 (5 %), i1, i3, i4 (4 %), i5 (2 %).
// Dix composants, pas trente-quatre. Le premier composant bloquant mesure etait i0 dans
// 1 009 records — la borne exacte que ce fichier fait sauter.
//
// # D'OU VIENT LA GRAMMAIRE
//
// Recette R7-d, la meme qu'au lot C-bis pour ti=13, et le desassemblage la donne entierement :
// chaine `.rdata` -> getter de nom (`LEA RAX,[chaine] ; RET`) -> descripteur de composant
// (10 pointeurs, le getter en `+0x28`) -> SERIALISEUR RESEAU en `+0x38`. Toutes les largeurs
// sont FIXES et lues dans le desassemblage ; aucune n'est inferee d'une mesure.
//
//	i0  timers                     FUN_142edbac8  2 x R(7)  (boucle sur 8 octets, pas de 4)
//	i1  color                      FUN_142edb548  4 x R(8)  (quantifie 0..255, FUN_142ed1a78)
//	i2  formatted-text             FUN_142edb5bc -> FUN_142c70d5c  (cf. components_batch3.go)
//	i3  object-reference           FUN_142edb6a4  R(32)
//	i4  interaction-filter         FUN_142edb5cc -> FUN_142c7023c  NON PORTE (cf. plus bas)
//	i5  type                       FUN_142edbb00  R(32)     (enumere « objective-type »)
//	i6  enabled                    FUN_142edb594  R(1)
//	i7  priority                   FUN_142edb820  R(8)
//	i8  message-type               FUN_142edb604  R(4)
//	i9  secondary-formatted-text   FUN_142edba00  meme lecteur que i2
//	i10 is-new-and-unseen          FUN_142edb5dc  R(1)
//	i11 is-only-one-item-unlocked  FUN_142edb5f0  R(1)
//	i12 PROGRESS                   FUN_142edb8c0  R(32)
//	i13 REQUIRED-PROGRESS          FUN_142edb960  R(32)
//	i14 state                      FUN_142edba10  R(3)      (FUN_1424d121c)
//	i15 parent-objective           FUN_142edb780  R(32)
//	i16..i31 sub-objective-entities FUN_142edba24 R(32) chacun
//	i32 outro-phase-duration       FUN_142edb740  R(8) quantifie [0, DAT_143cd84b8]
//	i33 forced-update              FUN_142edb5a8  R(1)
//
// # LE SEUL COMPOSANT LAISSE DEHORS, ET POURQUOI
//
// i4 `interaction-filter` : `FUN_142c7023c` lit R(4) de masque, R(1), puis pour chacun des
// quatre bits poses un R(4) de tag SUIVI D'UN APPEL VIRTUEL (`FUN_141e99630` case 1..5 :
// R(1) puis `(*vtable[8])(filtre)`). La largeur de cette queue depend de la sous-classe du
// filtre et n'est pas lisible sans enumerer ces sous-classes. Le porter a moitie
// DESYNCHRONISERAIT le record au lieu de l'arreter proprement, ce qui est strictement pire.
// COUT MESURE de ce trou : i4 est present dans 10 des 265 records porteurs de jauge (4 %).
// CONDITION DE RETRAIT : porte quand les sous-classes de filtre seront enumerees, ou supprime
// de cette liste si la mesure montre qu'il ne coute plus rien.
//
// # CE QUE CE FICHIER NE DIT PAS, ET C'EST IMPORTANT
//
// LES LARGEURS NE SONT PAS ENCORE VALIDEES SUR FILM. Le portage fait sauter le blocage du
// traverseur — 2 211 records marches jusqu'au bout contre 884 avant — mais « marcher jusqu'au
// bout » ne teste que la COUVERTURE du dispatch, jamais la JUSTESSE d'une largeur : un composant
// qui lit deux bits de trop laisse la marche aboutir, simplement decalee.
//
// Et la mesure du 2026-09-01 dit qu'une derive subsiste : la valeur de i12 est constante sur
// toute la duree d'un slot, identique dans trois matchs differents, et des slots consecutifs
// rendent des valeurs decalees d'un bit. Le detail et la marche a suivre sont dans
// `objective_scan.go`. NE PAS EXPLOITER i12/i13 EN PRODUCTION AVANT CET ORACLE.
//
// La SEMANTIQUE des valeurs, elle, n'est pas non plus etablie. Les 32 bits de i12 sont publies
// BRUTS ; `ObjectiveProgressFloat` porte la convention de relecture en flottant, qui reste une
// convention. Nommer un canal dans le decodeur figerait une interpretation.

import "math"

// ObjectiveTypeIndex est l'index d'archetype des objectifs geres.
const ObjectiveTypeIndex = 11

// Etiquettes de registre des composants portes par ce fichier. Ecrites EN ENTIER parce que le
// garde-rail G1 de la table ECS exige que le nom apparaisse dans le fichier que la table designe.
const (
	compObjectiveTimers                 = "managed-objective-timers-component"
	compObjectiveColor                  = "managed-objective-color-component"
	compObjectiveObjectReference        = "managed-objective-object-reference-component"
	compObjectiveType                   = "managed-objective-type-component"
	compObjectiveEnabled                = "managed-objective-enabled-component"
	compObjectivePriority               = "managed-objective-priority-component"
	compObjectiveMessageType            = "managed-objective-message-type-component"
	compObjectiveIsNewAndUnseen         = "managed-objective-is-new-and-unseen-component"
	compObjectiveIsOnlyOneItemUnlocked  = "managed-objective-is-only-one-item-unlocked-component"
	compObjectiveProgress               = "managed-objective-progress-component"
	compObjectiveRequiredProgress       = "managed-objective-required-progress-component"
	compObjectiveState                  = "managed-objective-state-component"
	compObjectiveParentObjective        = "managed-objective-parent-objective-component"
	compObjectiveSubObjectiveEntities   = "managed-objective-sub-objective-entities-component"
	compObjectiveOutroPhaseDuration     = "managed-objective-outro-phase-duration-component"
	compObjectiveForcedUpdate           = "managed-objective-forced-update-component"
	compObjectiveFormattedText          = "managed-objective-formatted-text-component"
	compObjectiveSecondaryFormattedText = "managed-objective-secondary-formatted-text-component"
)

// Les largeurs, telles que le desassemblage les donne. Une constante par role, jamais un
// litteral en ligne : c'est la table de largeurs de l'archetype, et l'unique endroit ou elle est
// ecrite.
const (
	objectiveTimerBits        = 7
	objectiveTimerCount       = 2
	objectiveColorChannelBits = 8
	objectiveColorChannels    = 4
	objectiveHandleBits       = 32
	objectiveTypeBits         = 32
	objectivePriorityBits     = 8
	objectiveMessageTypeBits  = 4
	objectiveProgressBits     = 32
	objectiveStateBits        = 3
	objectiveOutroBits        = 8
)

// ObjectiveField designe le champ publie de ti=11. Enumeration STABLE et NOMMEE, jamais un index
// de registre : le decoupage du registre CHANGE AVEC LE BUILD (mesure du lot 0), donc un index de
// composant ne designe pas la meme chose d'un film a l'autre.
type ObjectiveField int

// Les champs publies. TOUS ont un consommateur — les composants sans consommateur sont
// consommes en bits mais pas publies (meme regle qu'`i0` de ti=13, retire en revue R1 comme
// sortie morte).
const (
	ObjectiveFieldTimers           ObjectiveField = iota // i0  : les deux minuteurs
	ObjectiveFieldObjectReference                        // i3  : la reference vers l'objet
	ObjectiveFieldType                                   // i5  : le type d'objectif
	ObjectiveFieldProgress                               // i12 : LA JAUGE
	ObjectiveFieldRequiredProgress                       // i13 : LE SEUIL
	ObjectiveFieldState                                  // i14 : l'etat vivant
	ObjectiveFieldCount            = 6
)

// String rend l'etiquette de registre du champ.
func (f ObjectiveField) String() string {
	switch f {
	case ObjectiveFieldTimers:
		return compObjectiveTimers
	case ObjectiveFieldObjectReference:
		return compObjectiveObjectReference
	case ObjectiveFieldType:
		return compObjectiveType
	case ObjectiveFieldProgress:
		return compObjectiveProgress
	case ObjectiveFieldRequiredProgress:
		return compObjectiveRequiredProgress
	case ObjectiveFieldState:
		return compObjectiveState
	}
	return champInconnu
}

// objectiveHook, si non nil, recoit chaque lecture d'un champ publie de ti=11.
//
// PAS DE `present` ICI : aucun de ces composants n'a de porte de tete — leur presence est le bit
// de MASQUE, que l'appelant connait deja. Global de paquet : l'appelant detient
// `LockProcessDecode`.
var objectiveHook func(f ObjectiveField, values []uint64)

// SetObjectiveHook installe (ou retire, avec nil) la sonde des composants de ti=11.
//
// UN HOOK SEPARE de ceux de ti=10, ti=12 et ti=13, pour la meme raison qui les separait entre
// eux : les archetypes sont distincts et leurs slots disjoints.
func SetObjectiveHook(h func(f ObjectiveField, values []uint64)) { objectiveHook = h }

func publishObjective(f ObjectiveField, values ...uint64) {
	if objectiveHook != nil {
		objectiveHook(f, values)
	}
}

// consumeObjectiveTimers (i0) — FUN_142edbac8 : boucle sur huit octets par pas de quatre, donc
// DEUX appels a FUN_142ed15a0, qui ecrit `valeur + 1` sur SEPT bits. La valeur zero du flux
// signifie donc « pas de minuteur » (cf. ObjectiveTimerValue).
func consumeObjectiveTimers(br *BitReader) {
	vals := make([]uint64, 0, objectiveTimerCount)
	for i := 0; i < objectiveTimerCount; i++ {
		vals = append(vals, br.ReadBits(objectiveTimerBits))
	}
	publishObjective(ObjectiveFieldTimers, vals...)
}

// consumeObjectiveColor (i1) — FUN_142edb548 : quatre canaux quantifies sur huit bits
// (FUN_142ed1a78 -> FUN_140dc6248, 0x100 niveaux). Consomme sans publier : le camp d'un objectif
// se lit deja par des voies mesurees, et une couleur brute n'aurait pas de consommateur.
func consumeObjectiveColor(br *BitReader) {
	for i := 0; i < objectiveColorChannels; i++ {
		br.ReadBits(objectiveColorChannelBits)
	}
}

// consumeObjectiveObjectReference (i3) — FUN_142edb6a4 : R(32) plat. C'est la reference vers
// l'objet physique de l'objectif (le drapeau, le crane, la bombe) : la cle qui relierait la jauge
// a une entite du monde.
func consumeObjectiveObjectReference(br *BitReader) {
	publishObjective(ObjectiveFieldObjectReference, br.ReadBits(objectiveHandleBits))
}

// consumeObjectiveType (i5) — FUN_142edbb00 -> FUN_1407edaf4 : R(32) plat, un enumere que le jeu
// nomme « objective-type ».
func consumeObjectiveType(br *BitReader) {
	publishObjective(ObjectiveFieldType, br.ReadBits(objectiveTypeBits))
}

// consumeObjectiveBool porte les quatre booleens plats de l'archetype : i6 `enabled`,
// i10 `is-new-and-unseen`, i11 `is-only-one-item-unlocked`, i33 `forced-update` — tous
// FUN_1406d49c4, R(1). Un seul deserialiseur pour quatre composants : la grammaire est
// identique, et la dupliquer ne dirait rien de plus.
func consumeObjectiveBool(br *BitReader) { br.ReadBit() }

// consumeObjectivePriority (i7) — FUN_142edb820 : R(8) plat.
func consumeObjectivePriority(br *BitReader) { br.ReadBits(objectivePriorityBits) }

// consumeObjectiveMessageType (i8) — FUN_142edb604 : R(4) plat.
func consumeObjectiveMessageType(br *BitReader) { br.ReadBits(objectiveMessageTypeBits) }

// consumeObjectiveProgress (i12) — FUN_142edb8c0 : R(32) plat, sans porte. LA JAUGE.
func consumeObjectiveProgress(br *BitReader) {
	publishObjective(ObjectiveFieldProgress, br.ReadBits(objectiveProgressBits))
}

// consumeObjectiveRequiredProgress (i13) — FUN_142edb960 : R(32) plat. LE SEUIL. Avec i12 il
// donne la FRACTION de capture ; seul, il ne dit rien.
func consumeObjectiveRequiredProgress(br *BitReader) {
	publishObjective(ObjectiveFieldRequiredProgress, br.ReadBits(objectiveProgressBits))
}

// consumeObjectiveState (i14) — FUN_142edba10 -> FUN_1424d121c : R(3) plat. L'etat vivant de
// l'objectif (huit valeurs possibles) ; leur semantique n'est PAS etablie et n'est pas devinee
// ici.
func consumeObjectiveState(br *BitReader) {
	publishObjective(ObjectiveFieldState, br.ReadBits(objectiveStateBits))
}

// consumeObjectiveEntityRef porte i15 `parent-objective` (FUN_142edb780) et les seize instances
// de i16..i31 `sub-objective-entities` (FUN_142edba24) : R(32) plat dans les deux cas. Consomme
// sans publier — l'identite de zone que ces seize emplacements porteraient est un chantier a
// part, et publier seize handles bruts que personne ne lit serait une sortie morte.
func consumeObjectiveEntityRef(br *BitReader) { br.ReadBits(objectiveHandleBits) }

// consumeObjectiveOutroPhaseDuration (i32) — FUN_142edb740 -> FUN_1406d22c0 : R(8) quantifie sur
// [0, DAT_143cd84b8] avec les drapeaux (signe=0, inclusif=1). Consomme sans publier.
func consumeObjectiveOutroPhaseDuration(br *BitReader) { br.ReadBits(objectiveOutroBits) }

// ObjectiveTimerValue rend la valeur d'un minuteur : FUN_142ed15a0 ecrit `valeur + 1`, donc zero
// signifie « absent » et vaut -1.
func ObjectiveTimerValue(q uint64) int { return int(q) - 1 }

// ObjectiveProgressFloat relit les 32 bits de i12 / i13 comme un flottant simple precision.
//
// C'EST UNE CONVENTION, PAS UNE MESURE. Le serialiseur copie un `uint` de 32 bits sans le
// transformer ; la progression d'objectif est un flottant partout ailleurs dans le moteur, ce qui
// rend cette relecture probable — mais tant qu'une mesure sur film ne l'a pas confirmee,
// l'appelant qui veut le brut lit le brut.
func ObjectiveProgressFloat(q uint64) float32 { return math.Float32frombits(uint32(q)) }
