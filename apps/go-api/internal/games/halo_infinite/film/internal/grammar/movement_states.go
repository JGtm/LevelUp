package grammar

// movement_states.go — LE BALAYAGE DES ETATS DE MOUVEMENT (lot 5.3.6, 2026-09-21).
//
// # LE HOOK EST LA GRAMMAIRE
//
// Ce balayage ne relit AUCUN bit a cote du deserialiseur : il branche
// [Observation.EtatMouvementHook] et laisse le decodeur publier. C est la regle de
// `ScanAbilityImpulses`, `ScanFilmGrappleReads` et `ScanFilmAbilityRanks` — celle qui garantit
// que la valeur publiee est celle que la production decode, et non une seconde lecture qui
// pourrait deriver.
//
// # LA MARCHE EST CELLE DU FRAME-PROCESSEUR, ET C EST UNE DECISION MESUREE
//
// Elle n emploie PAS `walkDeltaBipedRecords`, la marche par CHERCHEUR D ANCRES des autres
// balayages de capacite. Mesure sur `bfecd02b` (lot 5.3.6, sonde de production) : sur
// 162 444 records reconnus par cette marche, les masques annoncent `i29` **ZERO** fois et `i62`
// **UNE** fois. C est le defaut meme que le lot 5.3.2 avait constate et nomme — un chercheur
// d ancres ne retient que les records qui RESSEMBLENT a un en-tete de bipede, c est-a-dire la
// population PAUVRE `{i0, i1, i21, i25}` (113 bits, l etalon Rosette), celle ou les etats de
// mouvement ne sont precisement pas.
//
// Elle emploie donc [DecodeFrameViews] — le port du frame-processeur `FUN_142987460` — sur
// [MovementStateViews] vues, avec les paquets a liste d evenements pleine localises par
// `marchLocateStrict`. Sur le meme film cette marche rend 97 447 records `ti=35` dont 3
// desynchronises, 1 407 lectures d accroupi et 1 507 de glissade (lot 5.3.5).
//
// # LES LARGEURS D AXE DE LA CARTE SONT UN PRE-REQUIS, ET L APPELANT LES POSE
//
// Le contexte doit porter le descripteur d axes de la carte du match — l installateur de
// `replay` (`replay/world_object_precision.go`), appele juste apres `NewFilmContextForMap`.
// Sans lui le
// chemin absolu d `i0` lit ses trois axes aux largeurs d UNE carte — `cliffhanger` — appliquees
// a toutes : cinq bits de trop par record sur `snowbound`, et tout ce qui suit est du bruit.
// Mesure de l ecart (lot 5.3.5) : records `ti=35` 31 530 contre 97 447, desyncs 38 contre 3.
// [types.MovementStateStats.MapWidths] publie le triplet employe pour que le document le dise.
//
// # CE QUI EST LU, ET CE QUI NE PEUT PAS L ETRE
//
// QUATRE etats LUS — `i29` accroupi, `i62` glissade, `i54` action de mobilite, `i57` SPRINT
// (l index de la fente de capacite active, lot 5.9.5) — et UN genre DERIVE, `jumpDerived`, qui
// n est pas lu mais integre depuis la vitesse verticale d `i1` (`movement_states_jump.go`,
// lot 5.9.4). Les noms disent la difference, et c est delibere. L en-tete de
// `types/grammar_mouvement.go` porte les chiffres.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Les etiquettes de registre des trois composants, avec leurs deux orthographes : les films
// portent l une OU l autre (avec ou sans `-component`), comme toute la famille.
// sprintAbilitySlotRaw est la valeur BRUTE d `i57` qui designe la fente du sprint.
//
// DEUX, ET PAS UN : le flux lit `R(2)` et l ecrivain pose `bloc+3 = valeur - 1`
// (`FUN_142f268c4`), donc la fente `1` — celle de `'sasp'`, nommee par `FUN_14319d1ec` — se lit
// `2` dans le flux. Le decalage vit ICI, en un seul point, plutot que dans chaque lecteur.
const sprintAbilitySlotRaw = 2

// abilityComponentName / abilityComponentAlt : l etiquette de registre d `i57`, deux orthographes.
const (
	abilityComponentName = "biped-spartan-ability-component"
	abilityComponentAlt  = "biped-spartan-ability"
)

const (
	crouchComponentName    = "unit-crouch-component"
	crouchComponentNameAlt = "unit-crouch"
	slideComponentName     = "biped-slide-component"
	slideComponentNameAlt  = "biped-slide"
	mobilityComponentName  = "biped-mobility-action-component"
	mobilityComponentAlt   = "biped-mobility-action"
)

// MovementStateViews est le nombre de vues de replication deroulees par paquet.
//
// TROIS, ET C EST CE QUE L ECRIVAIN DEROULE : `FUN_142987460` boucle exactement trois fois
// (`do { ... } while (uVar7 < 3)`), chaque vue portant sa propre fin de trame. Le depot emploie
// HUIT ailleurs (`marchViews`, `killsource.Options.Views`) ; la mesure du lot 5.3.3-b dit que 8
// lit au-dela de la trame — 7 000 records de plus, mais un etalon `i25` degrade de 0,8 point et
// 1 251 records `ti=35` de MOINS. Trois rend le meilleur etalon, et trois est ce que le code
// fait.
const MovementStateViews = 3

// movementStateSkipLeadBits est l amorce de paquet consommee avant le premier record d un
// paquet a liste d evenements VIDE : [drapeau de configuration][continuation = 0]. C est la
// valeur de [DefaultPacketPreambleBits], nommee ici pour que le site de lecture la dise.
const movementStateSkipLeadBits = DefaultPacketPreambleBits

// ScanFilmMovementStates est l ENVELOPPE D2, HORS PRODUCTION : elle charge le film et ouvre un
// contexte pour elle seule. ATTENTION — ce contexte n a PAS les largeurs d axe de la carte (cf.
// l en-tete) : cette enveloppe sert aux instruments, pas a la cuisson.
func ScanFilmMovementStates(dir string) ([]types.MovementStateRead, types.MovementStateStats,
	error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, types.MovementStateStats{}, err
	}
	return ScanMovementStates(contexteDeBobine(film))
}

// ScanMovementStates decode les etats de mouvement d un film DEJA CHARGE.
func ScanMovementStates(fc *FilmContext) ([]types.MovementStateRead, types.MovementStateStats,
	error) {
	var st types.MovementStateStats
	st.MapWidths = fc.LargeursObjetDuMonde().AxisW
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	reg, err := fc.Registry()
	if err != nil {
		return nil, st, err
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return nil, st, err
	}
	sc := &movementStateScanner{st: &st,
		crouch:   componentIndexOfAny(arch, crouchComponentName, crouchComponentNameAlt),
		slide:    componentIndexOfAny(arch, slideComponentName, slideComponentNameAlt),
		mobility: componentIndexOfAny(arch, mobilityComponentName, mobilityComponentAlt),
		ability:  componentIndexOfAny(arch, abilityComponentName, abilityComponentAlt),
		vues:     map[movementStateKey]types.MovementStateRead{},
	}
	if sc.crouch < 0 && sc.slide < 0 && sc.mobility < 0 && sc.ability < 0 {
		// AUCUNE ERREUR, et c est delibere : un film dont l archetype bipede ne declare aucun
		// des trois ne transmet pas les etats de mouvement. C est un fait MESURE, que `Absent`
		// publie au lieu de le confondre avec un film ou personne ne s accroupit.
		st.Absent, st.Scanned = true, true
		return nil, st, nil
	}
	cfg := fc.CadreDeBalayage()
	obs := NouvelleObservation()
	obs.EtatMouvementHook = sc.recevoir
	cfg.Obs = obs
	sc.monde = NewWorld(reg)
	// LE REPLI DU LOT 5.23 ENTRE EN PRODUCTION ICI. La table anticipee est construite en UNE
	// passe sur les images-cles de tous les chunks (1,8 s sur un film de 29 chunks, aucun
	// decodage de trame) ; au point de rejet, la marche y lit l archetype qu une image-cle
	// ULTERIEURE donne a un eid que le monde ne connait pas encore. Cf. `keyframe_anticipe.go`
	// et [World.LierParAnticipation] : le record de naissance n est toujours pas lu.
	sc.monde.PoserTableAnticipee(ConstruireTableAnticipee(fc))
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		// La table ne rend qu une declaration STRICTEMENT POSTERIEURE a ce chunk.
		sc.monde.PoserChunkCourant(c)
		sc.lierLeMonde(data, pks)
		for _, pk := range pks {
			sc.paquet(c, pk, data, cfg)
		}
	}
	sc.deriverLesSauts()
	sc.publier()
	st.Scanned = true
	return sc.out, st, nil
}

// movementStateKey deduplique une lecture : le chemin d inference de [DecodeFrameViews]
// re-parcourt un record quand une chaine de transitoires le demande, et le deserialiseur
// re-publie alors la MEME transition. Compter deux fois la meme gonflerait les denominateurs
// sans qu aucun compteur ne le dise.
type movementStateKey struct {
	slot uint32
	kind string
	ts   uint64
}

// movementStateScanner porte l etat du balayage.
type movementStateScanner struct {
	st                      *types.MovementStateStats
	out                     []types.MovementStateRead
	monde                   *World
	crouch, slide, mobility int
	// ability est l index d `i57` dans l archetype bipede — la fente de capacite active, donc
	// le SPRINT (cf. `sprintAbilitySlotRaw`). Il entre dans la garde d absence : un film dont
	// l archetype ne declare AUCUN des quatre ne transmet pas les etats de mouvement.
	ability            int
	vues               map[movementStateKey]types.MovementStateRead
	chunk, paquetIndex int
	ts                 uint64
	// vit porte les lectures de vitesse verticale par vie — la matiere du saut DERIVE
	// (`movement_states_jump.go`). Elle n est pas publiee telle quelle : seules les montees
	// reconnues a leur hauteur deviennent des transitions.
	vit map[uint32][]jumpVelSample
}

// lierLeMonde ajoute au monde les liaisons slot -> archetype portees par les images-cles du
// chunk. Sans elles le decodeur de trame ne sait pas quel archetype porte un slot, et chaque
// record delta est rejete avant toute lecture.
func (sc *movementStateScanner) lierLeMonde(data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			//nolint:gosec // slot, TI et Gen viennent du walker d image-cle, bornes par construction
			sc.monde.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
		}
	}
	// PUIS LA TABLE DE DATUMS, POUR LES SLOTS QUE LA CHAINE N A PAS ATTEINTS (lot 5.16.4).
	// La branche vive de la boucle de records lit l archetype d un delta dans la table de datums
	// du decodeur partage, et l image-cle est le DUMP de cette table (`keyframe_datums.go`). La
	// marche d ancres ci-dessus suit la CHAINE des records et se coupe ; la table, elle, se lit a
	// position libre. Les liaisons deja posees ne sont pas ecrasees.
	posees, ambigus := LierTableDeDatums(sc.monde, data, pks)
	sc.st.DatumBindings += posees
	sc.st.DatumAmbiguous += ambigus
}

// paquet decode UN paquet delta. Les paquets a liste d evenements PLEINE sont localises par la
// signature du depot (`marchLocateStrict` : un delta du slot 123, long de 35 bits, a composant
// unique — candidat unique et vrai sur 690 paquets sur 690) ; ceux qu elle ne localise pas sont
// comptes et sautes, parce que sauter la liste bit-exactement demanderait la grammaire de
// charge de chaque type d evenement.
func (sc *movementStateScanner) paquet(chunk int, pk FilmPacket, data []byte, cfg FrameConfig) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := movementStateSkipLeadBits
	if _, present := PacketHeadEventType(pay); present {
		sc.st.EventPackets++
		if debut = marchLocateStrict(pay, sc.monde, cfg); debut < 0 {
			sc.st.EventPacketsUnlocated++
			return
		}
		sc.st.EventPacketsLocated++
	}
	sc.st.Packets++
	sc.chunk, sc.paquetIndex, sc.ts = chunk, pk.Index, pk.TimestampUS
	recs, _ := DecodeFrameViews(pay, sc.monde, cfg, MovementStateViews, debut)
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		sc.st.Records++
		if r.DesyncAt >= 0 {
			sc.st.Desyncs++
		}
	}
}

// recevoir capte UNE publication du deserialiseur.
//
// LE FILTRE EST ICI, ET IL EST NECESSAIRE : `i54` est un composant du bipede, mais l attribution
// de slot du chemin d inference est PARTIELLE — `decodeInferLoop` ne pose pas le slot de capture
// pour un record NEW, si bien qu une lecture herite alors du slot du record precedent (D13 de la
// note 5.3). Ne garder que les slots LIES AU BIPEDE ecarte ces lectures au lieu de les attribuer
// a tort, et `SlotUnbound` dit combien.
func (sc *movementStateScanner) recevoir(comp EtatMouvementComposant, slot uint32, v []uint64) {
	var lu types.MovementStateRead
	switch comp {
	case EtatAccroupi:
		if len(v) < 2 {
			return
		}
		lu = types.MovementStateRead{Kind: types.MovementCrouch, On: v[0] != 0,
			Progress: uint32(v[1])} //nolint:gosec // R(10), borne a 1023
	case EtatGlissade:
		if len(v) < 1 {
			return
		}
		lu = types.MovementStateRead{Kind: types.MovementSlide, On: v[0] != 0}
	case EtatMobilite:
		if len(v) < 1 {
			return
		}
		lu = types.MovementStateRead{Kind: types.MovementMobility, On: v[0] != 0}
	case EtatCapaciteActive:
		if len(v) < 1 {
			return
		}
		// LA FENTE 1 EST LE SPRINT, et l image la nomme (cf. `types.MovementSprint`). Le flux
		// ecrit la fente DECALEE DE +1, donc le brut `2`. Toute autre valeur — y compris `0`,
		// « aucune fente active » — LEVE l etat : ce sont des transitions, et le plieur en fait
		// des intervalles.
		lu = types.MovementStateRead{Kind: types.MovementSprint, On: v[0] == sprintAbilitySlotRaw}
	case EtatVitesse:
		// PAS UN ETAT, LA MATIERE D UN ETAT : la vitesse verticale sert a DERIVER le saut
		// (`movement_states_jump.go`), qui publiera ses propres transitions apres la marche.
		sc.vitesse(slot, v)
		return
	case EtatPosture, EtatControleUnite:
		return // lus par le meme deserialiseur, hors du perimetre de ce calque
	}
	if ti, ok := sc.monde.ArchetypeForSlot(slot); !ok || ti != BipedTypeIndex {
		sc.st.SlotUnbound++
		return
	}
	lu.Slot, lu.Chunk, lu.PacketIndex, lu.TimestampUS = slot, sc.chunk, sc.paquetIndex, sc.ts
	k := movementStateKey{slot: slot, kind: lu.Kind, ts: sc.ts}
	if _, deja := sc.vues[k]; deja {
		sc.st.Duplicates++
		return
	}
	sc.vues[k] = lu
	sc.st.Read++
}

// publier vide la table de deduplication dans la sortie, TRIEE.
func (sc *movementStateScanner) publier() {
	sc.out = make([]types.MovementStateRead, 0, len(sc.vues))
	for _, r := range sc.vues {
		sc.out = append(sc.out, r)
	}
	sortMovementStates(sc.out)
}

// sortMovementStates ordonne les lectures sur (instant, slot, genre) — un ordre TOTAL. Un tri
// partiel laisserait l ordre des lectures d un meme paquet dependre du parcours, donc le
// document dependre d un detail de balayage.
func sortMovementStates(out []types.MovementStateRead) {
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.TimestampUS != b.TimestampUS {
			return a.TimestampUS < b.TimestampUS
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Kind < b.Kind
	})
}
