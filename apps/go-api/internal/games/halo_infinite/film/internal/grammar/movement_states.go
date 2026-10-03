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
// Il est donc un CANAL de la marche des trames ([CanalDesTrames], [Distribuer], `marche_trames.go`)
// — le port du frame-processeur `FUN_142987460` — avec les paquets a liste d evenements pleine
// localises par `marchLocateStrict`. Sur le meme film cette marche rend 97 447 records `ti=35` dont
// 3 desynchronises, 1 407 lectures d accroupi et 1 507 de glissade (lot 5.3.5). Le canal lit la
// structure de chaque trame et ce que son crochet recoit ; il ne pilote pas la marche.
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
// QUATRE etats LUS — `i29` accroupi, `i62` glissade, `i54` ESCALADE (`clamber`, nommee au lot
// 5.22.4 par neuf verdicts Theater sur neuf), `i57` SPRINT
// (l index de la fente de capacite active, lot 5.9.5) — et UN genre DERIVE, `jumpDerived`, qui
// n est pas lu mais integre depuis la vitesse verticale d `i1` (`movement_states_jump.go`,
// lot 5.9.4). Les noms disent la difference, et c est delibere. L en-tete de
// `types/grammar_mouvement.go` porte les chiffres.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
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

// velocityComponentName est l etiquette de registre d `i1` du bipede, la seule que le deserialiseur
// de la vitesse de translation lit et publie ([EtatVitesse], `dispatch_object.go`).
const velocityComponentName = "object-translational-velocity-dynamic-precision-component"

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

// ScanMovementStates decode les etats de mouvement d un film DEJA CHARGE. C est la PROJECTION de
// [ScanMarcheDesTrames] sur ses etats de mouvement : la marche est la meme, et le tir continu
// qu elle lit aussi est simplement laisse de cote (les instruments qui n en ont pas besoin).
func ScanMovementStates(fc *FilmContext) ([]types.MovementStateRead, types.MovementStateStats,
	error) {
	m, err := ScanMarcheDesTrames(fc)
	return m.MovementStates, m.MovementStateStats, err
}

// MarcheDesTrames est ce que LA marche du frame-processeur rend : les etats de mouvement du
// Spartan (vue B) et le TIR CONTINU (vue C, lot M4b). UNE marche, deux canaux : lire la vue C
// dans une seconde marche doublerait le cout du decodage le plus cher du film pour relire des
// paquets que celle-ci traverse deja.
type MarcheDesTrames struct {
	MovementStates      []types.MovementStateRead
	MovementStateStats  types.MovementStateStats
	ContinuousFire      []types.ContinuousFireBurst
	ContinuousFireStats types.ContinuousFireStats
	// LiaisonsParRepliDAnticipation : les liaisons que le repli `repli_liaison_par_anticipation`
	// ([World.LierParRepliDAnticipation]) a posees pendant la marche, tous archetypes confondus.
	// `replay` les verse au compteur de replis de la cuisson (lot J8.1, constat GA1-2). Ce compte
	// N EST PAS dans [types.MovementStateStats] : il ne se persiste pas avec les faits, il voyage
	// dans le rapport des replis que le fichier de faits porte deja.
	LiaisonsParRepliDAnticipation int
	// DebutsDeListeParRepliFermeAuBit : les listes dont le debut est pris au second rang de
	// [debutParFermetureRangee] (repli `repli_debut_de_liste_ferme_au_bit`). Comme le compte precedent,
	// il voyage dans le rapport des replis, pas dans [types.MovementStateStats].
	DebutsDeListeParRepliFermeAuBit int
}

// ScanMarcheDesTrames deroule la marche du frame-processeur sur un film DEJA CHARGE et rend ses
// deux canaux, distribues sur UNE marche ([Distribuer]) : les etats de mouvement et le tir continu.
//
// UN FILM SANS ETATS DE MOUVEMENT EST MARCHE QUAND MEME depuis le lot M4b : son archetype bipede
// ne declare aucun des composants d etat (`Absent`), mais sa vue de controle porte le tir continu.
// Les etats de mouvement y restent EXACTEMENT ce qu ils etaient — aucune lecture, `Absent` et
// `Scanned` poses, les compteurs de marche remis a zero.
func ScanMarcheDesTrames(fc *FilmContext) (MarcheDesTrames, error) {
	var m MarcheDesTrames
	st := &m.MovementStateStats
	st.MapWidths = fc.LargeursObjetDuMonde().AxisW
	if len(fc.ChunkNumbers()) == 0 {
		return m, ErrNoFilmChunk
	}
	reg, err := fc.Registry()
	if err != nil {
		return m, err
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return m, err
	}
	sc := nouveauCanalDesEtats(st, reg, arch)
	tir := nouveauCollecteurTirContinu(&m.ContinuousFireStats)
	if err := Distribuer(fc, sc, tir); err != nil {
		return m, err
	}
	m.LiaisonsParRepliDAnticipation = sc.liaisonsDuRepliDAnticipation()
	m.DebutsDeListeParRepliFermeAuBit = sc.obs.DebutsDeListeParRepliFermeAuBit
	m.ContinuousFire = tir.out
	m.ContinuousFireStats.Scanned = true
	if !sc.transmet {
		*st = types.MovementStateStats{MapWidths: st.MapWidths, Absent: true, Scanned: true}
		return m, nil
	}
	sc.deriverLesSauts()
	sc.publier()
	st.Scanned = true
	m.MovementStates = sc.out
	return m, nil
}

// nouveauCanalDesEtats prepare le canal des etats de mouvement d un film de registre `reg` et
// d archetype bipede `arch`.
//
// AUCUNE ERREUR quand les quatre etats manquent, et c est delibere : un film dont l archetype
// bipede ne declare aucun des quatre ne transmet pas les etats de mouvement. C est un fait MESURE,
// que `Absent` publie au lieu de le confondre avec un film ou personne ne s accroupit.
func nouveauCanalDesEtats(st *types.MovementStateStats, reg *Registry, arch Archetype) *movementStateScanner {
	transmet := componentIndexOfAny(arch, crouchComponentName, crouchComponentNameAlt) >= 0 ||
		componentIndexOfAny(arch, slideComponentName, slideComponentNameAlt) >= 0 ||
		componentIndexOfAny(arch, mobilityComponentName, mobilityComponentAlt) >= 0 ||
		componentIndexOfAny(arch, abilityComponentName, abilityComponentAlt) >= 0
	return &movementStateScanner{st: st, transmet: transmet, vues: map[movementStateKey]types.MovementStateRead{},
		physique: occurrencesDe(reg, compVehicleTypePhysics)}
}

// movementStateKey deduplique une lecture : le chemin d inference de la marche ([decodeInferLoop])
// re-parcourt un record quand une chaine de transitoires le demande, et le deserialiseur
// re-publie alors la MEME transition. Compter deux fois la meme gonflerait les denominateurs
// sans qu aucun compteur ne le dise.
type movementStateKey struct {
	slot uint32
	kind string
	ts   uint64
}

// movementStateScanner est le CANAL DES ETATS DE MOUVEMENT de la marche des trames
// ([CanalDesTrames]) : son crochet recoit ce que le deserialiseur publie, chaque trame ses comptes,
// lus dans la structure.
type movementStateScanner struct {
	st  *types.MovementStateStats
	out []types.MovementStateRead
	// transmet : l archetype bipede du film declare au moins un des quatre etats (accroupi,
	// glissade, mobilite, capacite active — `i57`, la fente de capacite active, donc le SPRINT, cf.
	// `sprintAbilitySlotRaw`). Faux : ni crochet ni interet, `Absent` publie.
	transmet bool
	vues     map[movementStateKey]types.MovementStateRead
	// m : ce que le canal voit de la marche — le paquet en cours, son en-tete pose avant sa marche
	// (le crochet le lit pendant), et la table d entites, en lecture seule.
	m *MarcheDistribuee
	// physique : les occurrences de `compVehicleTypePhysics` dans les archetypes du registre, que
	// [types.MovementStateStats.VehicleTypePhysicsByWriterLaw] compte par record.
	physique occurrencesDuComposant
	// vit porte les lectures de vitesse verticale par vie — la matiere du saut DERIVE
	// (`movement_states_jump.go`). Elle n est pas publiee telle quelle : seules les montees
	// reconnues a leur hauteur deviennent des transitions.
	vit map[uint32][]jumpVelSample
	// obs : l observation de la marche des trames, recue au bilan, NEW refuses soldes (constat
	// DFIX-R6, `keyframe_liaison.go`) : les comptes de replis de la marche s y lisent.
	obs *Observation
}

// Interets : sur le bipede, dans les trames, les quatre etats et la vitesse de translation, la
// matiere du saut derive — ce que le crochet garde. La posture et le controle d unite, que le meme
// deserialiseur publie, ne sont pas interpretes. Rien pour un film qui ne transmet pas les etats.
func (sc *movementStateScanner) Interets() []Interet {
	if !sc.transmet {
		return nil
	}
	noms := []string{crouchComponentName, crouchComponentNameAlt, slideComponentName, slideComponentNameAlt,
		mobilityComponentName, mobilityComponentAlt, abilityComponentName, abilityComponentAlt, velocityComponentName}
	out := make([]Interet, len(noms))
	for i, nom := range noms {
		out[i] = Interet{Phase: PhaseTrames, TI: BipedTypeIndex, Composant: nom}
	}
	return out
}

// Brancher pose la porte des etats de mouvement, pour un film qui les transmet.
func (sc *movementStateScanner) Brancher(obs *Observation, m *MarcheDistribuee) {
	sc.m = m
	if sc.transmet {
		obs.EtatMouvementHook = sc.recevoir
	}
}

// Trame compte UNE trame delta de la marche, dans la structure. Les paquets a liste d evenements
// PLEINE sont localises par la marche ([localiserLaListe]) ; ceux qu elle ne localise pas sont
// comptes et sautes, parce que sauter la liste bit-exactement demanderait la grammaire de charge de
// chaque type d evenement.
func (sc *movementStateScanner) Trame(p *lecture.Paquet) {
	if p.Debut != lecture.DebutEnTete {
		sc.st.EventPackets++
		if p.Debut == lecture.DebutNonLocalise {
			sc.st.EventPacketsUnlocated++
			return
		}
		sc.st.EventPacketsLocated++
		if p.Debut != lecture.DebutParSignature {
			sc.st.EventPacketsNewRecordStart++
		}
	}
	sc.st.Packets++
	for _, r := range p.Records {
		sc.st.VehicleTypePhysicsByWriterLaw += sc.physique.lu(p, r)
		if r.TI != BipedTypeIndex {
			continue
		}
		sc.st.Records++
		if r.Desync != lecture.SansDesynchronisation {
			sc.st.Desyncs++
		}
	}
}

// Clore recoit le bilan de la marche : ce que la liaison des images-cles a fait au monde, et le
// verdict des NEW refuses.
func (sc *movementStateScanner) Clore(b BilanDeMarche) {
	sc.obs = b.Obs
	sc.st.DatumBindings, sc.st.DatumAmbiguous = b.Liaisons.Datums, b.Liaisons.Ambigus
	sc.st.LiaisonsOubliees = b.Liaisons.Oubliees
	sc.st.NeufsContreUnVivant = b.Obs.NeufsContreUnVivant
	sc.st.NeufsRefusesLecturesFausses = b.Obs.NeufsRefusesLecturesFausses
	sc.st.NeufsRefusesCreationsPerdues = b.Obs.NeufsRefusesCreationsPerdues
	sc.st.NeufsRefusesIndecis = b.Obs.NeufsRefusesIndecis
}

// occurrencesDuComposant : par archetype du registre (son rang), le masque des index d iteration ou
// il declare un composant.
type occurrencesDuComposant map[int16]uint64

// occurrencesDe releve, dans le registre, les index ou chaque archetype declare le composant `nom`.
func occurrencesDe(reg *Registry, nom string) occurrencesDuComposant {
	out := occurrencesDuComposant{}
	for ti, a := range reg.Archetypes {
		for _, i := range a.indicesOf(nom) {
			if i < 64 {
				out[int16(ti)] |= 1 << uint(i) //nolint:gosec // rang d un archetype, sur 6 bits
			}
		}
	}
	return out
}

// lu rend 1 quand le record `r` du paquet `p` porte une occurrence du composant — traversee ou
// infranchissable : une lecture du composant au sens de la trace de la marche —, 0 sinon.
func (o occurrencesDuComposant) lu(p *lecture.Paquet, r lecture.Record) int {
	masque, ok := o[r.TI]
	if !ok {
		return 0
	}
	for _, c := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if masque>>c.Index&1 == 1 {
			return 1
		}
	}
	return 0
}

// archetypeDuSlot rend l archetype que la table d entites de la marche donne a un slot.
func (sc *movementStateScanner) archetypeDuSlot(slot uint32) (uint32, bool) {
	e, ok := sc.m.Entites.Entite(slot)
	return uint32(e.TI), ok
}

// recevoir capte UNE publication du deserialiseur.
//
// LE FILTRE EST ICI, ET IL RESTE NECESSAIRE : `i54` est un composant du bipede, et la lecture ne
// vaut que sur un slot LIE AU BIPEDE. Jusqu au lot J5.4, `decodeInferLoop` ne posait pas le slot
// de capture d un record NEW, qui publiait sous le slot du record precedent (D13 de la note 5.3,
// levee par GA1-3) ; le NEW publie desormais sous SON slot, mais ce slot n est lie qu APRES la
// traversee — la lecture d un NEW sur un slot neuf est donc ecartee ici, et `SlotUnbound` dit
// combien.
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
		lu = types.MovementStateRead{Kind: types.MovementClamber, On: v[0] != 0}
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
	if ti, ok := sc.archetypeDuSlot(slot); !ok || ti != BipedTypeIndex {
		sc.st.SlotUnbound++
		return
	}
	p := sc.m.Paquet
	lu.Slot, lu.Chunk, lu.PacketIndex, lu.TimestampUS = slot, p.Chunk, p.Index, p.TS
	k := movementStateKey{slot: slot, kind: lu.Kind, ts: p.TS}
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
	slices.SortStableFunc(out, func(a, b types.MovementStateRead) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Slot, b.Slot),
			cmp.Compare(a.Kind, b.Kind))
	})
}

// liaisonsDuRepliDAnticipation somme, tous archetypes confondus, les liaisons que le repli
// `repli_liaison_par_anticipation` a posees pendant la marche.
//
// UNE SEULE SOURCE DEPUIS LE LOT J8.7 (2026-09-27) : l observation de la marche
// ([Observation.LiaisonsParRepliDAnticipation]), incrementee au point de rejet. Le monde tenait un
// second compte du MEME fait (`World.anticipations`), retire : deux compteurs d un seul fait
// finissent par diverger sans que rien ne dise lequel publier.
func (sc *movementStateScanner) liaisonsDuRepliDAnticipation() int {
	if sc.obs == nil {
		return 0
	}
	n := 0
	for _, k := range sc.obs.LiaisonsParRepliDAnticipation {
		n += k
	}
	return n
}
