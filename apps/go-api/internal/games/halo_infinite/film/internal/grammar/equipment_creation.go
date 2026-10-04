package grammar

// equipment_creation.go — LE RECORD DE CRÉATION d'un objet d'équipement (ti=37).
//
// CE QUE CE BALAYAGE FAIT SORTIR, ET POURQUOI IL N'EXISTAIT PAS. Les quatre composants delta
// de ti=37 (`deployed`, `activated`, `creator`, `energy` — cf. equipment_state.go) ne portent
// AUCUNE identité : mesuré le 2026-08-15, aucun d'eux ne distingue un mur d'un capteur ni d'un
// bonus au sol. Mais le DEFAULT-STATE de l'archétype, lui, en porte deux — et son déserialiseur
// les consommait pour rester aligné, exactement le défaut d'i48 (2026-08-14) et des quatre
// champs delta (2026-08-16) :
//
//	FUN_1407f105c = V ; deser ti36 ; ECS_ReadEntityRefIndex5 [R(1) ; si 0 -> R(5)] ;
//	                R(1) porte ; si 1 -> R(32) « ability-enabled-id »
//
// LA RÈGLE QUI GOUVERNE, la même qu'ailleurs : c'est le DÉSERIALISEUR qui publie
// (consumeDefaultStateTI37, default_state_arch.go), jamais un second lecteur posé à côté de lui.
//
// OÙ CE RECORD SE TROUVE. Un record de CRÉATION (type NEW) vit dans les paquets DELTA, à côté
// des records delta ordinaires, et son en-tête diffère du leur :
//
//	DELTA   [1] [slot:13] [gen:2] [baseline:1] [masque…]          (cf. matchWorldObjectRecord)
//	NEW     [0] [type:2 = 1] [slot:13] [gen:2] [ti:6] [default-state] [porte:1] [masque…]
//
// C'est le R(6) `typeIndex` qui rend ce balayage sélectif là où celui des deltas ne l'est pas :
// un record NEW DIT son archétype, on ne le déduit pas d'une bande de slots.
//
// CE QU'IL NE DIT PAS. Ni ce que valent les deux champs, ni s'ils nomment quoi que ce soit :
// ce fichier les fait SORTIR, l'instrument de mesure (equipment_creation_test.go) les juge.
// VERDICT DE LA MESURE, à lire avant de bâtir sur ces deux champs : leur porte est FERMÉE sur
// 503 records de création sur 503 (2026-08-17). L'identité de l'objet est dans le bloc
// `object-multiplayer-properties` du MÊME record — son mot de 32 bits inconditionnel est le
// GlobalID du tag `eqip` de l'objet (11 valeurs observées sur 11 résolues dans les 105 tags
// `eqip` du jeu). C'est ce champ-là que le lecteur veut, et il voyage dans MPPVal[MPPWord32].
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// EquipmentCreationField désigne l'un des deux champs que le default-state de ti=37 lisait et jetait ; l'ordre est celui du flux.
type EquipmentCreationField int

// Les deux champs, et EquipmentCreationFieldCount qui les compte.
const (
	// EquipCreationRef est la référence d'entité sur 5 bits (ECS_ReadEntityRefIndex5,
	// FUN_1407f2058). Le nom du déserialiseur dit « référence d'entité » ; il ne dit PAS que
	// c'est un créateur, et rien ici ne le suppose.
	EquipCreationRef EquipmentCreationField = iota
	// EquipCreationAbilityID est l'identifiant 32 bits nommé « ability-enabled-id » par le
	// registre de champs de FUN_14080dec4.
	EquipCreationAbilityID
	EquipmentCreationFieldCount = 2
)

// String rend l'étiquette du champ — celle du déserialiseur, la seule façon honnête de le nommer.
func (f EquipmentCreationField) String() string {
	switch f {
	case EquipCreationRef:
		return "entity-ref-index5"
	case EquipCreationAbilityID:
		return "ability-enabled-id"
	}
	return fmt.Sprintf("champ inconnu (%d)", int(f))
}

func (o *Observation) publishEquipmentCreation(f EquipmentCreationField, value uint64, present bool) {
	if o == nil || o.EquipmentCreationHook == nil {
		return
	}
	o.EquipmentCreationHook(f, value, present)
}

// Découpage de l'en-tête d'un record de CRÉATION (type NEW) d'objet du monde dans un paquet
// delta. Chaque largeur vient d'un lecteur porté et vérifié : readRecordType (R(1) ; si 0 ->
// R(2)), readRecordID (R(IDLowBits=13) puis R(2) de génération, cf. DefaultFrameConfig) et
// TraverseEntity (R(6) typeIndex).
const (
	woNewTypeBits   = 3              // R(1)=0 « pas un delta » + R(2)=1 « recNew »
	woNewSlotBits   = handleSlotBits // FrameConfig.IDLowBits — le handle se lit par LireHandle
	woNewGenBits    = handleGenBits
	woNewTIBits     = 6
	woNewHeaderBits = woNewTypeBits + woNewSlotBits + woNewGenBits + woNewTIBits
)

// ScanFilmEquipmentCreations décode les records de création des objets d'équipement du film de
// dir, sur la bande de slots de ti=37 lue dans les images-clés.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe
// `observateur.EquipmentCreationHook`, qui est un global de paquet. Le hook est restauré à la sortie.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
//
// ScanFilmEquipmentCreations est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanEquipmentCreations].
func ScanFilmEquipmentCreations(dir string, wr *profile.Vec3Range) ([]types.EquipmentCreation, types.EquipmentCreationStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, types.EquipmentCreationStats{}, err
	}
	return ScanEquipmentCreations(contexteDeBobine(film), wr)
}

// ScanEquipmentCreations décode les records de création d'équipement d'un film DEJA CHARGE.
func ScanEquipmentCreations(fc *FilmContext, wr *profile.Vec3Range) ([]types.EquipmentCreation, types.EquipmentCreationStats, error) {
	var st types.EquipmentCreationStats
	if len(fc.ChunkNumbers()) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	band := worldObjectSlotBand(fc, EquipmentTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archétype ti=%d dans les keyframes du film",
			EquipmentTypeIndex)
	}
	return ScanEquipmentCreationsForBand(fc, wr, band)
}

// ScanFilmEquipmentCreationsForBand balaye une BANDE DE SLOTS donnée. La bande est un paramètre
// pour que le témoin de contrôle (une bande FANTÔME de même cardinalité, faite de slots jamais
// vus porter cet archétype) passe par le MÊME code que la mesure — sans quoi le contrôle ne
// contrôlerait pas le décodeur mais une variante de lui (règle établie par
// WorldObjectPositionsForBand).
//
// ScanFilmEquipmentCreationsForBand est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanEquipmentCreationsForBand].
func ScanFilmEquipmentCreationsForBand(
	dir string, wr *profile.Vec3Range, band map[uint32]bool,
) ([]types.EquipmentCreation, types.EquipmentCreationStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, types.EquipmentCreationStats{}, err
	}
	return ScanEquipmentCreationsForBand(contexteDeBobine(film), wr, band)
}

// ScanEquipmentCreationsForBand balaye une bande de slots donnée dans un film DEJA CHARGE.
func ScanEquipmentCreationsForBand(
	fc *FilmContext, wr *profile.Vec3Range, band map[uint32]bool,
) ([]types.EquipmentCreation, types.EquipmentCreationStats, error) {
	var st types.EquipmentCreationStats
	if wr == nil {
		return nil, st, fmt.Errorf("bornes monde absentes : sans elles le décodeur ne rend que des quanta")
	}
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	st.Slots = len(band)
	w, err := fc.marcheDeCreation(EquipmentTypeIndex, wr, band)
	if err != nil {
		return nil, st, err
	}
	cre, st := fc.creationsRelevees(w)
	return cre, st, nil
}

// installCreationHooks branche les DEUX sondes du record de création — le default-state de
// ti=37 et le bloc `object-multiplayer-properties` qu'il contient — sur un même tampon, et rend
// la fonction qui restaure les sondes précédentes.
//
// Les deux vont ENSEMBLE et se posent d'un seul geste : le balayage lit un record entier, et
// n'installer que l'une des deux rendrait un record à moitié observé sans que rien ne le dise.
func installCreationHooks(cur *equipCreationRead) *Observation {
	obs := NouvelleObservation()
	obs.EquipmentCreationHook = func(f EquipmentCreationField, v uint64, present bool) {
		cur.present[f], cur.val[f] = present, v
	}
	obs.MppHook = func(f MPPField, v uint64, present bool) {
		cur.mppPresent[f], cur.mppVal[f] = present, v
	}
	return obs
}

// equipCreationRead porte ce que les hooks ont publié pour UN record.
type equipCreationRead struct {
	present    [EquipmentCreationFieldCount]bool
	val        [EquipmentCreationFieldCount]uint64
	mppPresent [MPPFieldCount]bool
	mppVal     [MPPFieldCount]uint64
}

// equipCreationWalk porte ce que la marche d'un record doit connaître (règle des 5 paramètres).
//
// ELLE EST PARAMÉTRÉE PAR L'ARCHÉTYPE, et il le faut : la marche d'un record de création est la
// MÊME pour tous les objets du monde (en-tête NEW, default-state, porte, masque, i0) — seuls
// changent le `typeIndex` que l'en-tête doit porter et le déserialiseur du default-state. Les
// ARMES AU SOL (`ti=42`, ground_weapon_creation.go) empruntent donc ce code au lieu d'en
// recopier une seconde version qui re-divergerait au premier correctif.
type equipCreationWalk struct {
	// obs / prof : l OBSERVATEUR et le PROFIL que cette marche pose sur chaque lecteur qu elle
	// construit (lot 2.3, cf. [equipCreationWalk.contexte]). Le profil vient du contexte du
	// film, ou d un candidat quand la calibration MPP balaie.
	obs   *Observation
	prof  ProfilDeBalayage
	comps int
	wr    *profile.Vec3Range
	band  map[uint32]bool
	cur   *equipCreationRead
	// ti est le typeIndex exigé de l'en-tête NEW ; zéro vaut `EquipmentTypeIndex`.
	ti uint32
	// deser est le déserialiseur du default-state de cet archétype ; nil vaut celui de `ti=37`.
	deser func(*Lecteur)
	// posDecode décode et VALIDE le composant i0 à l'offset donné (gate de sélectivité). nil vaut
	// le chemin OBJET DU MONDE (decodeWorldObjectPos, porte 2 + IndexW bits). L'archétype VÉHICULE (`ti=40`)
	// porte un i0 en PRÉCISION-DYNAMIQUE (porte 5 bits, biped) : il passe ici decodeBipedI0Pos
	// (vehicle_creation.go). Sans ce paramètre le gate lit i0 avec la mauvaise grammaire et n'est
	// plus sélectif — mesuré : le témoin fantôme rendait alors PLUS de records que la vraie bande.
	posDecode func([]byte, int) ([3]float32, bool)
	// posBits est la largeur du composant i0, pour avancer le curseur après un record accepté.
	// Zéro vaut projPosBits() (chemin objet du monde) ; `ti=40` passe lay.TotalBits().
	posBits int
	// ammoArch, non nil, demande la lecture des MUNITIONS du record (composant i20) en rejouant
	// la boucle de composants de production sur l'archétype donné. SEULE l'arme au sol la
	// demande : c'est le seul archétype de cette marche qui porte `weapon-ammo-component`, et le
	// seul dont la sémantique des champs soit prouvée (ground_weapon_ammo.go). La lecture se
	// fait avec son PROPRE curseur : elle ne change aucun bit du chemin existant.
	ammoArch *Archetype
}

// archetype et defaultState rendent les réglages effectifs de la marche (défauts `ti=37`).
func (w equipCreationWalk) archetype() uint32 {
	if w.ti == 0 {
		return EquipmentTypeIndex
	}
	return w.ti
}

// contexte rend ce que cette marche pose sur ses lecteurs.
func (w equipCreationWalk) contexte() ContexteDeLecture {
	return ContexteDeLecture{Profil: w.prof, Obs: w.obs}
}
func (w equipCreationWalk) defaultState() func(*Lecteur) {
	if w.deser == nil {
		return consumeDefaultStateTI37
	}
	return w.deser
}

// decodePos VALIDE et décode i0 à l'offset at : le décodeur paramétré (dyn.-préc. pour `ti=40`)
// s'il est fourni, sinon le chemin objet du monde (porte 2 + IndexW bits).
func (w equipCreationWalk) decodePos(pay []byte, at int) ([3]float32, bool) {
	if w.posDecode != nil {
		return w.posDecode(pay, at)
	}
	return decodeWorldObjectPos(pay, at, w.wr, w.prof.LargeursObjetDuMonde())
}

// posAdvance rend la largeur d'i0 pour avancer le curseur après un record accepté.
func (w equipCreationWalk) posAdvance() int {
	if w.posBits > 0 {
		return w.posBits
	}
	return projPosBits(w.prof.LargeursObjetDuMonde())
}

// creationA lit le record de création dont l'en-tête est reconnu à la position p (la vie `h`),
// dans le paquet `lieu`, et compte : une ancre, puis, quand son corps se déroule, un record
// accepté. La marche des créations (`creations_du_monde.go`) avance son curseur après un record
// accepté.
func (w equipCreationWalk) creationA(pay []byte, p int, h types.LifeKey, lieu lieuDuPaquet,
	st *types.EquipmentCreationStats) (types.EquipmentCreation, bool) {
	st.Anchors++
	cre, ok := w.readCreation(pay, p, len(pay)*8, st)
	if !ok {
		return cre, false
	}
	cre.Slot, cre.Gen, cre.BitPos = h.Slot, h.Gen, p
	cre.Chunk, cre.PacketIndex, cre.TimestampUS = lieu.chunk, lieu.pk.Index, lieu.pk.TimestampUS
	st.Accepted++
	st.MaskSparse, st.MaskFull = st.MaskSparse+b2i(!cre.MaskFull), st.MaskFull+b2i(cre.MaskFull)
	st.NoI0 += b2i(!cre.MaskHasI0)
	if cre.HasRef {
		st.WithRef++
	}
	if cre.HasID {
		st.WithID++
	}
	if cre.HasAmmo {
		st.WithAmmo++
	}
	return cre, true
}

// matchEquipmentNewHeader reconnaît un en-tête de record de CRÉATION d'objet d'ÉQUIPEMENT.
func matchEquipmentNewHeader(pay []byte, p int, band map[uint32]bool) (slot, gen uint32, ok bool) {
	return matchWorldObjectNewHeader(pay, p, band, EquipmentTypeIndex)
}

// matchWorldObjectNewHeader reconnaît un en-tête de record de CRÉATION d'objet du monde de
// l'archétype `ti` à la position de bit p. PUR (aucune I/O).
//
// Quatre contraintes, dont trois sont des CONSTANTES du format : le préfixe de type
// (`0` puis `01`), le typeIndex R(6) == ti, et l'appartenance du slot à la bande de l'archétype.
// La génération est libre : ses quatre valeurs sont légitimes (même règle que les deltas
// d'objet du monde, cf. matchWorldObjectRecord).
func matchWorldObjectNewHeader(
	pay []byte, p int, band map[uint32]bool, ti uint32,
) (slot, gen uint32, ok bool) {
	return matchWorldObjectNewHeaderIn(pay, p, func(s uint32) bool { return band[s] }, ti)
}

// matchWorldObjectNewHeaderIn est la MÊME reconnaissance, la bande passée en PRÉDICAT.
//
// POURQUOI CETTE FORME EXISTE (lot E2, 2026-09-08). Le balayage des créations de BIPÈDE
// (biped_creation.go) tient sa bande en [SlotBand] DENSE — un tableau indexé, parce qu'il
// interroge la bande une fois par bit candidat du payload, soit des dizaines de millions de fois
// par film (cf. slot_band_dense.go). Convertir cette bande en `map[uint32]bool` pour appeler la
// forme ci-dessus annulerait exactement le gain que le type dense existe pour obtenir. Le
// prédicat est donc le point de passage COMMUN, et il n'y a toujours qu'UNE reconnaissance
// d'en-tête NEW dans le paquet — pas une seconde copie qui divergerait au premier correctif.
func matchWorldObjectNewHeaderIn(
	pay []byte, p int, dansLaBande func(uint32) bool, ti uint32,
) (slot, gen uint32, ok bool) {
	if lu, ok := archetypeDeLEnTeteNEW(pay, p); !ok || lu != ti {
		return 0, 0, false
	}
	h := LireHandle(pay, p+woNewTypeBits)
	if !dansLaBande(h.Slot) {
		return 0, 0, false
	}
	return h.Slot, h.Gen, true
}

// archetypeDeLEnTeteNEW lit le DÉBUT d'un en-tête de création à la position p — le préfixe de type
// (`0` puis `01`) — et rend le typeIndex R(6) qu'il annonce : la première étape de
// [matchWorldObjectNewHeaderIn], qui ne dépend d'aucun archétype ni d'aucune bande.
func archetypeDeLEnTeteNEW(pay []byte, p int) (uint32, bool) {
	if source.BitsTolerants(pay, p, 1) != 0 { // un record DELTA ouvre sur 1
		return 0, false
	}
	if source.BitsTolerants(pay, p+1, 2) != 1 { // type de record : 1 = NEW
		return 0, false
	}
	return uint32(source.BitsTolerants(pay, p+woNewTypeBits+woNewSlotBits+woNewGenBits, woNewTIBits)), true
}

// readCreation déroule le corps d'un record de création : le default-state (qui publie les deux
// champs par le hook), la porte du record NEW, le masque de présence, puis le composant i0.
//
// CHAQUE ÉCHEC EST COMPTÉ À PART. Un en-tête reconnu dont le corps ne se déroule pas est très
// probablement un faux positif du balayage bit à bit ; le dire par un compteur, plutôt que par
// un rejet silencieux, est ce qui permet de juger la sélectivité de l'ancre.
func (w equipCreationWalk) readCreation(
	pay []byte, p, total int, st *types.EquipmentCreationStats,
) (types.EquipmentCreation, bool) {
	var cre types.EquipmentCreation
	*w.cur = equipCreationRead{}
	br := LecteurSur(pay)
	br.PoserContexte(w.contexte())
	br.PoserObservation(w.obs)
	start := p + woNewHeaderBits
	br.SetBitPos(start)
	w.defaultState()(br)
	if br.BitPos() > total {
		st.Overflow++
		return cre, false
	}
	cre.DefaultStateBits = br.BitPos() - start
	br.ReadBit() // porte has-components du record NEW (R(1) avant le masque, cf. TraverseEntity)
	idx, full, ok := readMaskIndices(br, w.comps)
	if !ok || br.BitPos() > total {
		st.MaskBad++
		return cre, false
	}
	compStart := br.BitPos()
	v, ok := w.decodePos(pay, compStart)
	if !ok {
		st.PosBad++
		return cre, false
	}
	if w.ammoArch != nil {
		cre.Ammo, cre.HasAmmo = readGroundWeaponAmmo(pay, compStart, idx, *w.ammoArch, w.contexte())
	}
	cre.HasRef, cre.Ref = w.cur.present[EquipCreationRef], uint32(w.cur.val[EquipCreationRef])
	cre.HasID, cre.AbilityID = w.cur.present[EquipCreationAbilityID], uint32(w.cur.val[EquipCreationAbilityID])
	cre.MPPPresent, cre.MPPVal = w.cur.mppPresent, w.cur.mppVal
	cre.X, cre.Y, cre.Z = v[0], v[1], v[2]
	cre.Mask, cre.MaskFull, cre.MaskHasI0 = idx, full, idx[0] == 0
	cre.AfterBit = br.BitPos() + w.posAdvance()
	return cre, true
}

// readMaskIndices lit le masque de présence (FUN_1406d7610) et rend ses index, plus la branche
// empruntée. La contrainte commune aux deux branches — TOUT index annoncé doit exister dans
// l'archétype — est ce qui fait la sélectivité, et elle est bien plus sévère sur la branche
// pleine (33 bits de poids fort à zéro) que sur l'éparse.
//
// LA BRANCHE PLEINE EST ACCEPTÉE, et il le faut : un record de CRÉATION annonce l'état complet
// de l'entité, donc souvent plus de sept composants — au-delà, le compte de 3 bits de la
// branche éparse ne suffit plus et le moteur bascule sur le masque explicite. La refuser
// perdait la majorité des créations (mesuré : 58 records retenus contre 351 une fois acceptée).
func readMaskIndices(br *Lecteur, comps int) (idx []int, full, ok bool) {
	if br.ReadBit() { // gate==1 : masque plein explicite R(64)
		// Le composant i est le bit i du mot rendu par ReadBits(64) — la convention de
		// consumeMask, dont la branche éparse pose `mask |= 1 << idx` et que
		// traverseComponentLoop interroge par `Mask & (1 << i)`.
		m := br.ReadBits(64)
		for i := range 64 {
			if m&(uint64(1)<<uint(i)) == 0 {
				continue
			}
			if i >= comps {
				return nil, true, false
			}
			idx = append(idx, i)
		}
		return idx, true, len(idx) > 0
	}
	cnt := int(br.ReadBits(3))
	if cnt < 1 || cnt > worldObjectMaxMaskCnt {
		return nil, false, false
	}
	idx = make([]int, cnt)
	prev := -1
	for i := range cnt {
		v := int(br.ReadBits(6))
		if v <= prev || v >= comps {
			return nil, false, false
		}
		idx[i], prev = v, v
	}
	return idx, false, true
}
