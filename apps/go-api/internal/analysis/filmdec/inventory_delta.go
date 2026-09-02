package filmdec

// inventory_delta.go — L'INVENTAIRE DE GRENADES SUIVI DANS LES PAQUETS DELTA.
//
// CE QUE CE BALAYAGE LIT, ET POURQUOI IL EXISTE. L'inventaire du rejeu 2D est lu aux
// IMAGES-CLÉS (replay/inventory_decode.go), c'est-à-dire toutes les ~20 s : entre deux, la
// fiche du joueur affiche la dernière lecture connue, d'âge médian 8,4 s. Les paquets DELTA,
// eux, transmettent les composants d'inventaire AU CHANGEMENT — un ramassage, un lancer. Ce
// fichier les fait sortir du film pour RAFRAÎCHIR la même grandeur entre deux images-clés.
//
//	i22  `unit-grenade-counts-component`        les COMPTEURS, un par rang de grenade.
//	i47  `biped-desired-grenade-set-component`  le masque des types portés et le type
//	                                            SÉLECTIONNÉ (celui qui partira au lancer).
//
// LE CHEMIN EST CELUI D'i48 (ability_rank.go), à la lettre : ancre bit à bit
// `matchBipedHeader` sur la bande de slots bipèdes des images-clés, puis marche des
// composants du masque par les DÉSERS DE PRODUCTION (`walkRecordComponents` ->
// `consumeByName`). Aucun motif d'octets, aucune table de largeurs parallèle, aucune capture
// Cheat Engine : le désérialiseur PUBLIE ce qu'il lit, on ne relit jamais les bits à côté de
// lui — deux lecteurs du même champ divergeraient.
//
// LE TEST RÉFUTABLE EST DANS LA DONNÉE, pas dans un seuil choisi. `unit_weaponstate.go`
// l'énonce lui-même : la vérité terrain donne 35 bits FIXES pour i22 (3 + 4x8), donc le
// compteur R(3) vaut TOUJOURS 4, et « une lecture qui rend count != 4 est la signature d'un
// curseur mal placé ». Les valeurs, elles, sont bornées par le jeu (au plus 2 unités par
// type). Un curseur au hasard rendrait `count == 4` une fois sur huit et des octets uniformes
// sur 0..255. Les lectures qui violent ces bornes ne sont PAS publiées : lire du bruit vaut
// moins que ne rien lire. Elles sont COMPTÉES (`Implausible`), parce qu'un taux qui monte est
// le premier symptôme d'une dérive de largeur en amont.
//
// CE QUE CE CANAL NE DONNE PAS. L'IDENTITÉ DE L'ARME en main (i43/i44) : 14 et 9 annonces sur
// 171 851 records delta du film témoin — elle reste une lecture d'image-clé. Le suivi delta
// rafraîchit un inventaire dont l'arme est nommée ailleurs.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// L'appelant doit détenir LockProcessDecode (BuildFromFilm le fait) : les hooks installés
// sont des globaux de paquet.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// Les étiquettes de registre des deux composants. L'index d'itérateur est résolu PAR NOM dans
// le registre du film, jamais câblé : un index est un numéro de build (même règle que
// camoComponentName).
const (
	invDeltaGrenadeCountsName = "unit-grenade-counts-component"
	invDeltaGrenadeSetName    = "biped-desired-grenade-set-component"
	// invDeltaGrenadeSetAltName est l'étiquette courte que `consumeByName` accepte aussi ;
	// les deux existent selon les registres, et n'en chercher qu'une rendrait le balayage
	// muet sur les films qui portent l'autre.
	invDeltaGrenadeSetAltName = "biped-desired-grenade-set"
)

// invDeltaGrenadeSlots est le nombre de rangs de grenade du titre — la longueur des compteurs
// d'i22, et la largeur utile du masque d'i47. C'est AUSSI le `count` que le déser doit rendre
// (vérité terrain : 35 bits fixes = 3 + 4x8).
const invDeltaGrenadeSlots = 4

// invDeltaMaxPerType est le nombre maximal d'unités d'un même type que le jeu laisse porter.
// C'est une BORNE DE JEU, pas une borne de format : le champ est un R(8) qui pourrait coder
// 255. C'est précisément ce qui en fait un test réfutable.
const invDeltaMaxPerType = 2

// InventoryDeltaNoSel est la valeur de [InventoryDelta.Sel] quand i47 ne désigne AUCUN type.
// Même convention que le canal des images-clés (`KeyframeInventory.SelectedGrenadeRank`) :
// la grandeur publiée par les deux canaux est le RANG EN BASE 0, jamais le codage 1-base du
// flux. Publier deux grandeurs différentes sous le même nom est le défaut qui a coûté le
// chantier de la capacité d'armure (cf. replay/abilities.go).
const InventoryDeltaNoSel = -1

// InventoryDelta est UNE transmission d'inventaire de grenades, localisée dans le film.
type InventoryDelta struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires, donc
	// UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Grenades porte le compteur de chaque rang (i22). NIL = i22 n'était pas au masque de ce
	// record, ou sa lecture violait une borne : jamais « zéro grenade ».
	Grenades []uint32
	// Mask est le masque R(6) des types portés (i47), valide seulement si SelRead.
	Mask uint32
	// Sel est le rang SÉLECTIONNÉ en base 0, ou InventoryDeltaNoSel si i47 n'en désigne
	// aucun. Valide seulement si SelRead.
	Sel int
	// SelRead dit si i47 a été lu sur ce record. Faux = non transmis ou non atteint.
	SelRead bool
	// Ammo porte l'état de munitions des emplacements que CE record transmet — jamais les
	// quatre par défaut. Vide = aucun composant de munitions au masque.
	Ammo []InventoryDeltaAmmo
}

// ScanFilmInventoryDeltas décode les transmissions d'inventaire de grenades (i22 compteurs,
// i47 masque et sélection) dans les paquets delta du film de dir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe `grenadeCountsHook`
// et `grenadeSetHook`, qui sont des globaux de paquet. L'appelant doit détenir
// LockProcessDecode (BuildFromFilm le fait). Les hooks sont restaurés à la sortie.
//
// ScanFilmInventoryDeltas est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanInventoryDeltas].
func ScanFilmInventoryDeltas(dir string) ([]InventoryDelta, InventoryDeltaStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, InventoryDeltaStats{}, err
	}
	return ScanInventoryDeltas(film)
}

// ScanInventoryDeltas décode l'inventaire suivi dans les paquets delta d'un film DEJA CHARGE.
func ScanInventoryDeltas(film *filmsource.Film) ([]InventoryDelta, InventoryDeltaStats, error) {
	sc, err := newInvDeltaScanner(film)
	if err != nil {
		return nil, InventoryDeltaStats{}, err
	}
	restore := sc.installHooks()
	defer restore()

	for _, c := range sc.chunks {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			sc.scanPacket(c, pk, pk.Payload(data))
		}
	}
	sc.refuseAmmoIfContaminated()
	return sc.out, sc.st, nil
}

// invDeltaScanner porte l'état d'un balayage : la configuration résolue une fois (bande de
// slots, découpage i0, archétype, index des deux composants) et l'accumulateur.
type invDeltaScanner struct {
	chunks    []int
	slots     map[uint32]bool
	lay       I0Layout
	arch      Archetype
	minRecord int
	// role dit, pour un index de composant du masque, CE QU'IL EST pour l'inventaire — et,
	// pour les munitions, DE QUEL emplacement d'arme il parle. C'est la seule table câblée du
	// balayage, et elle est construite depuis les NOMS du registre du film, jamais depuis des
	// index en dur : un index est un numéro de build.
	role map[int]invDeltaRole
	// État de la lecture EN COURS (un record). Les hooks y déposent, `collect*` y puise.
	last22c    uint64
	last22v    []uint64
	got22      bool
	last47mask uint32
	last47sel  int
	got47      bool
	// ammo / rounds : ce que la marche a récolté par emplacement d'arme sur le record courant.
	ammo       [invDeltaWeaponSlots]invDeltaAmmoAcc
	rounds     [invDeltaWeaponSlots]uint32
	roundsRead [invDeltaWeaponSlots]bool
	// lastAmmo / lastRounds : le sas où le hook dépose, avant que `capture` ne sache à quel
	// emplacement l'attribuer. Le hook ne connaît pas l'emplacement — le déser est le même
	// pour les quatre.
	lastAmmo       invDeltaAmmoAcc
	lastRounds     uint32
	lastRoundsRead bool

	out []InventoryDelta
	st  InventoryDeltaStats
}

// invDeltaRole classe un index de composant du masque.
type invDeltaRole struct {
	kind invDeltaKind
	// weaponSlot est le rang de l'occurrence dans l'archétype (0..3), c'est-à-dire
	// l'emplacement d'arme. Sans objet pour les grenades.
	weaponSlot int
}

type invDeltaKind int

const (
	invRoleGrenadeCounts invDeltaKind = iota
	invRoleGrenadeSet
	invRoleAmmo
	invRoleRounds
)

// resetRecord vide l'état de lecture avant de marcher un nouveau record. Sans cela, la
// publication d'un record déborderait sur le suivant — un composant non transmis se lirait
// comme retransmis à l'identique.
func (sc *invDeltaScanner) resetRecord() {
	sc.got22, sc.got47 = false, false
	sc.ammo = [invDeltaWeaponSlots]invDeltaAmmoAcc{}
	sc.rounds, sc.roundsRead = [invDeltaWeaponSlots]uint32{}, [invDeltaWeaponSlots]bool{}
	sc.lastAmmo, sc.lastRoundsRead = invDeltaAmmoAcc{}, false
}

// newInvDeltaScanner résout tout ce qui ne dépend PAS du paquet — une fois pour le film.
func newInvDeltaScanner(film *filmsource.Film) (*invDeltaScanner, error) {
	chunks := FilmChunkNumbers(film)
	if len(chunks) == 0 {
		return nil, ErrNoFilmChunk
	}
	slots := bipedSlotBand(film, chunks)
	if len(slots) == 0 {
		return nil, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	lay, _, err := DetectI0LayoutOf(film)
	if err != nil {
		return nil, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := bipedArchetype(film)
	if err != nil {
		return nil, err
	}
	sc := &invDeltaScanner{
		chunks: chunks, slots: slots, lay: lay, arch: arch,
		role:      invDeltaRoles(arch),
		minRecord: bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits(),
	}
	if len(sc.role) == 0 {
		return nil, fmt.Errorf("aucun composant d'inventaire dans l'archétype biped du film")
	}
	return sc, nil
}

// invDeltaRoles construit la table des rôles depuis les NOMS du registre du film.
//
// L'ORDRE DES OCCURRENCES EST L'EMPLACEMENT D'ARME : `weapon-state-ammo` apparaît quatre fois
// dans l'archétype biped (i30/i33/i36/i39 sur les builds mesurés), et la k-ième occurrence
// décrit le k-ième emplacement. Le lire ainsi plutôt que de câbler 30 et 33 est ce qui rend le
// balayage indépendant du numéro de build.
func invDeltaRoles(arch Archetype) map[int]invDeltaRole {
	roles := map[int]invDeltaRole{}
	if i := archIndexOf(arch, invDeltaGrenadeCountsName); i >= 0 {
		roles[i] = invDeltaRole{kind: invRoleGrenadeCounts}
	}
	if i := archIndexOf(arch, invDeltaGrenadeSetName, invDeltaGrenadeSetAltName); i >= 0 {
		roles[i] = invDeltaRole{kind: invRoleGrenadeSet}
	}
	for k, id := range arch.indicesOf(invDeltaAmmoName) {
		if k >= invDeltaWeaponSlots {
			break
		}
		roles[id] = invDeltaRole{kind: invRoleAmmo, weaponSlot: k}
	}
	for k, id := range arch.indicesOf(invDeltaRoundsName) {
		if k >= invDeltaWeaponSlots {
			break
		}
		roles[id] = invDeltaRole{kind: invRoleRounds, weaponSlot: k}
	}
	return roles
}

// archIndexOf rend l'index d'itérateur du PREMIER nom présent dans l'archétype, ou -1.
func archIndexOf(arch Archetype, names ...string) int {
	for _, name := range names {
		if ids := arch.indicesOf(name); len(ids) > 0 {
			return ids[0]
		}
	}
	return -1
}

// installHooks branche les quatre sondes de déser et rend leur restauration.
func (sc *invDeltaScanner) installHooks() func() {
	prev22, prev47 := grenadeCountsHook, grenadeSetHook
	prevAmmo, prevRounds := weaponAmmoHook, weaponRoundsHook
	SetGrenadeCountsHook(func(c uint64, v []uint64) {
		sc.last22c, sc.last22v, sc.got22 = c, v, true
	})
	SetGrenadeSetHook(func(mask uint32, sel int) {
		sc.last47mask, sc.last47sel, sc.got47 = mask, sel, true
	})
	SetWeaponAmmoHook(func(hasMag bool, mag uint32, hasFrac bool, fracQ uint32) {
		sc.lastAmmo = invDeltaAmmoAcc{
			Read: true, HasMag: hasMag, Mag: mag, HasFrac: hasFrac, FracQ: fracQ,
		}
	})
	SetWeaponRoundsHook(func(rounds uint32) { sc.lastRounds, sc.lastRoundsRead = rounds, true })
	return func() {
		SetGrenadeCountsHook(prev22)
		SetGrenadeSetHook(prev47)
		SetWeaponAmmoHook(prevAmmo)
		SetWeaponRoundsHook(prevRounds)
	}
}

// scanPacket ancre et marche tous les records bipèdes d'un paquet delta.
func (sc *invDeltaScanner) scanPacket(chunk int, pk FilmPacket, pay []byte) {
	total := len(pay) * 8
	for p := 0; p+sc.minRecord <= total; {
		i0, slot, idx, ok := matchBipedHeader(pay, p, total, sc.slots, true, sc.lay)
		if !ok {
			p++
			continue
		}
		sc.st.Records++
		sc.readRecord(chunk, pk, pay, i0, total, slot, idx)
		p = i0 + sc.lay.TotalBits()
	}
}

// readRecord marche UNE SEULE FOIS le record et récolte au passage tous les composants
// d'inventaire que son masque annonce.
//
// UN SEUL PARCOURS POUR N CIBLES : appeler la marche une fois par composant relirait le
// record six fois (i22, i47, les deux chargeurs, les deux réserves) pour le même résultat.
// La marche s'arrête dès que tous les composants attendus ont été consommés — au-delà,
// dérouler la fin du record ne rapporterait rien.
func (sc *invDeltaScanner) readRecord(
	chunk int, pk FilmPacket, pay []byte, i0, total int, slot uint32, idx []int,
) {
	want := 0
	for _, id := range idx[1:] {
		if _, ok := sc.role[id]; ok {
			want++
		}
	}
	if want == 0 {
		return
	}
	sc.resetRecord()
	sc.countAnnounced(idx)
	seen := 0
	walkRecordComponents(pay, i0, total, idx, sc.lay, sc.arch, func(id int) bool {
		if r, ok := sc.role[id]; ok {
			sc.capture(r)
			seen++
		}
		return seen < want
	})
	rec := InventoryDelta{Slot: slot, Chunk: chunk, PacketIndex: pk.Index, TimestampUS: pk.TimestampUS}
	emit := sc.collectI22(&rec)
	emit = sc.collectI47(&rec) || emit
	emit = sc.collectAmmo(&rec) || emit
	if emit {
		sc.noteAccordI22I47(rec)
		sc.st.Emitted++
		sc.out = append(sc.out, rec)
	}
}

// countAnnounced incrémente les dénominateurs « le masque annonce ce composant ».
func (sc *invDeltaScanner) countAnnounced(idx []int) {
	for _, id := range idx[1:] {
		r, ok := sc.role[id]
		if !ok {
			continue
		}
		switch r.kind {
		case invRoleGrenadeCounts:
			sc.st.WithI22++
		case invRoleGrenadeSet:
			sc.st.WithI47++
		case invRoleAmmo:
			sc.st.WithAmmo++
		case invRoleRounds:
			sc.st.WithRounds++
		}
	}
}

// capture range la publication du déser qui vient d'être consommé, à la place que son RÔLE
// désigne. Le hook, lui, ne sait pas de quel emplacement il parle : c'est ici, et seulement
// ici, que l'index de composant devient un numéro d'emplacement d'arme.
// Les deux composants de grenade n'ont rien à ranger : leur hook les a déjà déposés dans
// l'unique emplacement qui les concerne.
func (sc *invDeltaScanner) capture(r invDeltaRole) {
	switch r.kind {
	case invRoleAmmo:
		sc.captureAmmo(r.weaponSlot)
	case invRoleRounds:
		sc.captureRounds(r.weaponSlot)
	}
}

// collectI22 statue la lecture des compteurs : lue ou non, plausible ou non.
func (sc *invDeltaScanner) collectI22(rec *InventoryDelta) bool {
	if !sc.got22 {
		sc.st.I22Unread++
		return false
	}
	sc.st.I22Read++
	if !invDeltaPlausible(sc.last22c, sc.last22v) {
		sc.st.Implausible++
		return false
	}
	rec.Grenades = make([]uint32, 0, len(sc.last22v))
	for _, v := range sc.last22v {
		rec.Grenades = append(rec.Grenades, uint32(v))
	}
	return true
}

// invDeltaPlausible applique LE test réfutable d'i22 : compteur == 4 (vérité terrain, 35 bits
// fixes) et chaque valeur dans les bornes du jeu. Il ne « nettoie » pas la donnée : il dit si
// le curseur était à sa place.
func invDeltaPlausible(count uint64, vals []uint64) bool {
	if count != invDeltaGrenadeSlots || len(vals) != invDeltaGrenadeSlots {
		return false
	}
	for _, v := range vals {
		if v > invDeltaMaxPerType {
			return false
		}
	}
	return true
}

// collectI47 statue la lecture du jeu de grenades, et convertit le codage 1-BASE du flux en
// RANG BASE 0 — la grandeur du canal des images-clés.
func (sc *invDeltaScanner) collectI47(rec *InventoryDelta) bool {
	if !sc.got47 {
		sc.st.I47Unread++
		return false
	}
	sc.st.I47Read++
	rec.Mask, rec.SelRead, rec.Sel = sc.last47mask, true, InventoryDeltaNoSel
	switch {
	case sc.last47mask == 0:
		// Aucune grenade portée : il n'y a rien à sélectionner, quoi que dise le champ.
		sc.st.MaskEmpty++
	case sc.last47sel == GrenadeSetNoSelection:
		sc.st.NoSelection++
	case sc.last47sel > invDeltaGrenadeSlots || sc.last47mask&(1<<uint(sc.last47sel-1)) == 0:
		// LE TEST RÉFUTABLE du handoff : une sélection non nulle appartient au masque NON VIDE
		// du même record. Quand elle n'y est pas, la lecture est publiée SANS sélection —
		// désigner un type que le porteur ne porte pas serait un faux, la taire n'en est pas un.
		sc.st.SelOutsideMask++
	default:
		rec.Sel = sc.last47sel - 1
	}
	return true
}

// noteAccordI22I47 confronte, sur un record qui porte les DEUX composants, le masque d'i47 au
// bitmap des compteurs d'i22. Cf. InventoryDeltaStats.Accord : c'est le contrôle croisé le
// plus fort du balayage, et il ne coûte rien.
func (sc *invDeltaScanner) noteAccordI22I47(rec InventoryDelta) {
	if rec.Grenades == nil || !rec.SelRead {
		return
	}
	var bitmap uint32
	for i, v := range rec.Grenades {
		if v > 0 {
			bitmap |= 1 << uint(i)
		}
	}
	sc.st.AccordChecked++
	if bitmap == rec.Mask {
		sc.st.Accord++
	}
}
