package grammar

// birth_loadouts.go — LA DOTATION DE NAISSANCE : les armes qu'un bipède porte À SA CRÉATION, lues
// dans le record NEW `ti=35` du paquet delta qui le crée (lot M3.2 de la campagne « retours
// rejeu », 2026-09-23 ; sonde P3, `.ai/V7.5/retours_rejeu_2026-09-23/SONDE_P3_armes_naissance.md`).
//
// # LE CANAL
//
// Le record NEW d'une naissance annonce à son masque les emplacements `weapon-state-type-info` :
// famille = moitié HAUTE, variante = moitié basse ; le premier emplacement porte l'arme 1, le
// second l'arme 2. Sonde P3 : un record NEW lu pour 45/45, 106/107 et 142/142 vies ; accord avec
// le premier relevé d'image-clé de la vie 98,8 % sur le film à dotation tirée au sort, contre
// 0 % au témoin.
//
// # LES DEUX RÉPARATIONS QUI L'OUVRENT
//
// (a) ATTEINDRE LE RECORD. Dans un paquet à liste d'événements, le record NEW de naissance est
// le PREMIER de la trame (la liste finit sur son en-tête dans 40/41, 91/99 et 123/125 cas), et le
// localisateur de production (`marchLocate`, signature du slot 123) démarre plus loin et le
// saute. Démarrer la marche à la fin de la liste demanderait la grammaire de charge de CHAQUE
// type d'événement (cf. `movementStateScanner.paquet`) : ce lecteur prend donc le record comme
// ANCRE, par la signature que [ScanBipedCreations] reconnaît déjà en production (43 bits
// déterminés) — l'alternative que la sonde P3 nomme.
//
// (b) LE TRAVERSER. L'état par défaut du bipède lit désormais le R(32) de sa dernière feuille
// (`default_state.go`) : sans lui, toute la boucle de composants partait 32 bits trop tôt.
//
// # LA FERMETURE, ET RIEN SANS ELLE
//
// Une ancre par signature peut être fortuite, et une traversée peut aller au bout en lisant du
// bruit. La dotation n'est rendue que si le record se FERME : traversé sans désynchronisation,
// dans le payload, et SUIVI d'un record confirmé — un delta qui se décode proprement sur un slot
// que le monde connaît, ou un record NEW dont le monde ou une image-clé ULTÉRIEURE confirme
// l'archétype. MESURE (voie film, lecture corrigée) : 48/48, 104/106 et 139/142 naissances
// fermées sur un film Arena, Super Fiesta et CTF ; au témoin (fin du record décalée de ±1, ±3,
// ±7 bits), 3 fermetures sur 1 776. Un record qui ne se ferme pas ne rend RIEN : aucune lecture
// de repli (une lecture « par catalogue » n'existe qu'en instrument de mesure) — le refus est
// COMPTÉ par cause dans [types.BirthLoadoutStats].
//
// HORS LIGNE (paquets delta porteurs de naissances) — jamais depuis un chemin de requête.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// weaponEmplacements rend le RANG de chaque composant `weapon-state-type-info` de l'archétype
// (index de composant -> 0, 1, 2...), dans l'ordre des composants. Les index viennent des NOMS
// du registre du film, jamais de constantes : un index de composant est un numéro de build.
func weaponEmplacements(arch Archetype) map[int]int {
	out := map[int]int{}
	for id := 0; id < archetypeBlockSlots; id++ {
		if arch.component(id) == compWeaponStateTypeInfo {
			out[id] = len(out)
		}
	}
	return out
}

// ScanBirthLoadouts lit la dotation de naissance de chaque création de bipède déjà reconnue par
// [ScanBipedCreations] (les créations sont passées : le balayage de production les a déjà).
func ScanBirthLoadouts(
	fc *FilmContext, creations []BipedCreation,
) ([]types.BirthLoadout, types.BirthLoadoutStats, error) {
	st := types.BirthLoadoutStats{Creations: len(creations)}
	if len(creations) == 0 {
		return nil, st, nil
	}
	reg, err := fc.Registry()
	if err != nil {
		return nil, st, err
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return nil, st, err
	}
	emp := weaponEmplacements(arch)
	if len(emp) == 0 {
		return nil, st, fmt.Errorf("aucun %s dans l'archétype biped du film", compWeaponStateTypeInfo)
	}
	s := newBirthScan(fc, reg, emp, &st)
	parPaquet := map[birthPacketKey][]BipedCreation{}
	for _, c := range creations {
		k := birthPacketKey{c.Chunk, c.PacketIndex}
		parPaquet[k] = append(parPaquet[k], c)
	}
	var out []types.BirthLoadout
	for _, num := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		s.lierLeChunk(num, data, pks)
		for _, pk := range pks {
			nes := parPaquet[birthPacketKey{num, pk.Index}]
			if len(nes) == 0 || pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			// LES CORPS DU PAQUET SONT LIÉS AVANT LA LECTURE : un record NEW de naissance est
			// souvent suivi d'un AUTRE (plusieurs réapparitions dans le même paquet), et la
			// signature de 43 bits qui l'a reconnu est une preuve indépendante de son archétype.
			for _, c := range nes {
				s.monde.BindFull(c.Generation<<30|c.Slot, BipedTypeIndex)
			}
			for _, c := range nes {
				if bl, ok := s.lire(pay, c); ok {
					out = append(out, bl)
				}
			}
		}
	}
	return out, st, nil
}

// birthPacketKey localise le paquet d'une création.
type birthPacketKey struct{ chunk, packet int }

// heldWeaponRead est UNE lecture d'emplacement d'arme rendue par le déserialiseur.
type heldWeaponRead struct{ high, low uint32 }

// birthScan porte ce que la lecture d'un film doit connaître (règle des 5 paramètres).
type birthScan struct {
	fc  *FilmContext
	cfg FrameConfig
	emp map[int]int
	// marche : la marche d image-cle DU FILM (preuve comprise, lot D-fix).
	marche MarcheDImageCle
	// monde porte les liaisons slot -> archétype du chunk courant : c'est lui qui dit qu'un delta
	// qui suit une naissance tombe sur un slot connu.
	monde *World
	// table est la table anticipée du film, construite à la PREMIÈRE question (un film dont
	// aucune naissance n'est suivie d'un record NEW inconnu du monde ne la paie pas).
	table    *TableAnticipee
	tableLue bool
	chunk    int
	obs      *Observation
	lus      []heldWeaponRead
	st       *types.BirthLoadoutStats
}

func newBirthScan(fc *FilmContext, reg *Registry, emp map[int]int, st *types.BirthLoadoutStats) *birthScan {
	s := &birthScan{fc: fc, cfg: fc.CadreDeBalayage(), emp: emp, monde: NewWorld(reg), st: st,
		marche: fc.MarcheDImageCle()}
	s.obs = NouvelleObservation()
	s.obs.HeldWeaponHook = func(h, l uint32) { s.lus = append(s.lus, heldWeaponRead{h, l}) }
	return s
}

// lierLeChunk pose sur le monde ce que les images-clés du chunk déclarent, puis la table de
// datums — la liaison de la marche de production (`movementStateScanner.lierLeMonde`), SANS son
// oubli des slots que l image-clé ne porte plus (lot D-fix, `keyframe_liaison.go`) : ce balayage
// ne décode aucun delta sous l archétype du monde (le record NEW porte le sien), il n y lit que la
// CONFIRMATION d une fermeture, et l oubli y retirait une fermeture sur 104 (`b1f01a33`) sans
// qu aucune lecture ne change de sens.
func (s *birthScan) lierLeChunk(num int, data []byte, pks []FilmPacket) {
	s.chunk = num
	s.monde.PoserChunkCourant(num)
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range s.marche.Records(pk.Payload(data)) {
			//nolint:gosec // slot, TI et Gen viennent du walker d image-cle, bornes par construction
			s.monde.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
		}
	}
	LierTableDeDatums(s.monde, data, pks)
}

// lire traverse le record NEW d'une création et rend sa dotation quand le record se ferme.
func (s *birthScan) lire(pay []byte, c BipedCreation) (types.BirthLoadout, bool) {
	var bl types.BirthLoadout
	br := LecteurSur(pay)
	br.poserCadre(s.cfg)
	br.PoserObservation(s.obs)
	br.SetBitPos(c.BitPos + woNewTypeBits + woNewSlotBits + woNewGenBits)
	s.lus = s.lus[:0]
	tr := TraverseEntity(br, s.monde.Reg, s.cfg.NewDefaultStateBits)
	switch {
	case tr.DesyncAt != -1:
		s.st.Desync++
		return bl, false
	case tr.EndBit > len(pay)*8:
		s.st.Overflow++
		return bl, false
	case !s.fermeSur(pay, tr.EndBit):
		s.st.Unconfirmed++
		return bl, false
	}
	bl = types.BirthLoadout{TimestampUS: c.TimestampUS, Chunk: c.Chunk, PacketIndex: c.PacketIndex,
		Slot: c.Slot, Generation: c.Generation}
	lu := 0
	for _, cr := range tr.Comps {
		k, arme := s.emp[cr.Index]
		if !arme || lu >= len(s.lus) {
			continue
		}
		r := s.lus[lu]
		lu++
		bl.Weapons = append(bl.Weapons, types.BirthWeapon{Emplacement: k, Family: r.high, Low: r.low})
	}
	if len(bl.Weapons) == 0 {
		s.st.NoWeaponComponent++
		return bl, false
	}
	s.st.Read++
	return bl, true
}

// fermeSur dit si le record qui commence à `p` CONFIRME la fin de la traversée : un delta qui se
// décode proprement sur un slot que le monde connaît, ou un record NEW dont le monde ou une
// image-clé ULTÉRIEURE donne le même archétype. Tout autre record — terminateur compris, que
// des bits nuls imitent — ne confirme rien.
func (s *birthScan) fermeSur(pay []byte, p int) bool {
	br := LecteurSur(pay)
	br.poserCadre(s.cfg)
	br.SetBitPos(p)
	if br.Remaining() < 24 {
		return false
	}
	if s.cfg.HasExtraFields {
		br.Skip(32)
	}
	switch readRecordType(br) {
	case recDelta:
		if _, _, ok := TryDeltaAt(pay, p, s.monde, s.cfg); ok {
			s.st.ClosedByDelta++
			return true
		}
	case recNew:
		id := readRecordID(br, s.cfg.IDLowBits, s.cfg.IDBase)
		ti := uint32(PeekBits(pay, br.BitPos(), 6)) //nolint:gosec // 6 bits
		if lie, ok := s.monde.ArchetypeForSlot(id & 0x3fffffff); ok && lie == ti {
			s.st.ClosedByBoundNew++
			return true
		}
		if ta, _, ok := s.tableAnticipee().ArchetypeApres(id, s.chunk); ok && ta == ti {
			s.st.ClosedByAnticipatedNew++
			return true
		}
	}
	return false
}

// tableAnticipee construit la table anticipée du film à la première question.
func (s *birthScan) tableAnticipee() *TableAnticipee {
	if !s.tableLue {
		s.table, s.tableLue = ConstruireTableAnticipee(s.fc), true
	}
	return s.table
}
