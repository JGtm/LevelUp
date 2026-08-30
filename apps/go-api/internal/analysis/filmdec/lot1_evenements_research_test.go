package filmdec

// lot1_evenements_research_test.go — LOT 1 : LE MODELE « LISTE D'EVENEMENTS PUIS TRAME »,
// teste de bout en bout sur la famille 0xCA (unit_zoom).
//
// LE MODELE M (reconciliation des lots D et E) : un paquet delta =
//
//	[1 bit configuration][liste d'evenements : ( 1 [R(7) type][3 refs gardees][charge] )* 0]
//	[trame de records jusqu'a la fin]
//
// Il explique d'un coup : le k=2 de 0xA0/0x80/0x89 (bit 1 = 0 -> liste VIDE, trame au
// bit 2, prouve par l'oracle) ; l'arithmetique octet0 = 0xC0 | (type >> 1) qui tombe juste
// sur TOUTES les familles a bit 1 = 1 (0xD2 -> 36 action_weapon_fire, 0xC2 -> 5
// projectile_detonate, 0xC0 -> 0/1 damage_aftermath/..., 0xD3 -> 38/39 reload/throw,
// 0xE9 -> 82 PlayerGameEventSmall, 0xCA -> 20/21 incident/unit_zoom) ; et les « en-tetes de
// largeur variable » mesures (la liste d'evenements a une longueur propre au paquet).
//
// POURQUOI 0xCA : le type 21 (unit_zoom) a une charge FIXE de 2 bits (R(2), valeur-1 —
// grammaire PROUVEE au desassemblage, lot E) et ses domaines de reference sont lus dans
// l'exe (vtable+0x58 du descripteur 0x144724e80 : ref0 -> domaine 4 (R(9)), ref1 -> 8
// (R(13)), ref2 -> 7 (R(13))). L'evenement entier est donc decodable sans aucune largeur
// devinee — le test le plus pur du modele. ENJEU PRODUIT : si M tient, la conclusion
// « aucun evenement de zoom dans la bobine » (lot E, lecture decalee d'un bit) TOMBE, et la
// lunette revient dans le film (~400 000 paquets 0xCA sur le corpus).
//
// CRITERES ECRITS AVANT LA MESURE (temoin 000d5950 puis 00502e52) :
//
//	M1 — bit 8 des 0xCA : la repartition type 20 (incident) / type 21 (unit_zoom) est
//	     publiee ; le modele exige type < 123 par construction (toujours vrai ici).
//	M2 — pour les type 21 : la charge R(2) et l'index de ref0 sont publies ; attendu si
//	     l'evenement est un zoom : ref0 presente sur la quasi-totalite (l'unite qui zoome)
//	     et un PETIT nombre d'index distincts (une poignee d'unites par film).
//	M3 — apres [charge][continuation = 0], la TRAME doit decoder : part de paquets dont la
//	     trame se ferme proprement >= 50 % du taux de 0xA0 mesure par ailleurs (36 %), ET
//	     masques 1..7 >= 80 % sur les deltas lies aboutis. C'est le verdict du modele.
//	M4 — continuation = 1 (plusieurs evenements par liste) : compte publie, non decode
//	     (le 2e type est publie, on s'arrete la).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"fmt"
	"math/bits"
	"os"
	"sort"
	"testing"
)

// lot1RefDomWidths : largeur de l'index R(w) par domaine (table 0x1451f98d0, lot B1/E2).
var lot1RefDomWidths = map[int]int{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}

// lot1LireRef consomme une reference gardee du domaine dom ; rend (index, presente).
// Le domaine 1 porterait une sonde R(1) qui reduit la largeur a 9 — aucun des types testes
// ici ne l'utilise, la sonde n'est pas modelisee.
func lot1LireRef(br *BitReader, dom int) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(uint(lot1RefDomWidths[dom]))
	br.Skip(2) // generation R(2)
	return idx, true
}

func TestLot1EvenementZoom(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	var (
		paquets, type20, type21, autresTypes int
		ref0Abs, cont1                       int
		zoomVals                             = map[uint64]int{}
		ref0Idx                              = map[uint64]int{}
		deuxieme                             = map[uint64]int{}
		trameOK, trameKO                     int
		deltasLies, masquesOK                int
	)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		cfg2 := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg2)
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xCA {
				continue
			}
			paquets++
			br := NewBitReader(pay)
			br.Skip(1)         // bit de configuration
			if !br.ReadBit() { // continuation : un evenement suit
				continue // impossible pour 0xCA (bit 1 = 1 par construction)
			}
			typ := int(br.ReadBits(7))
			switch typ {
			case 20:
				type20++
				continue // charge variable (incident) : non decode ici
			case 21:
				type21++
			default:
				autresTypes++
				continue
			}
			// Trois references gardees, domaines lus dans l'exe : 4, 8, 7.
			idx0, ok0 := lot1LireRef(br, 4)
			_, _ = lot1LireRef(br, 8)
			_, _ = lot1LireRef(br, 7)
			if ok0 {
				ref0Idx[idx0]++
			} else {
				ref0Abs++
			}
			zoomVals[br.ReadBits(2)]++ // la charge : R(2), niveau de lunette + 1
			if br.ReadBit() {          // continuation
				cont1++
				deuxieme[br.ReadBits(7)]++
				continue // liste multiple : le 2e evenement n'est pas decode
			}
			// Fin de liste : la TRAME de records commence ici.
			w := NewWorld(reg)
			w.Restore(snap)
			recs, decErr := DecodeFrameRecords(br, w, DefaultFrameConfig())
			if decErr == nil {
				trameOK++
			} else {
				trameKO++
			}
			for i := range recs {
				r := &recs[i]
				nm := bits.OnesCount64(r.Trace.Mask)
				if r.Type == recDelta && r.DesyncAt == -1 && nm > 0 {
					deltasLies++
					if nm <= 7 {
						masquesOK++
					}
				}
			}
		}
	}
	t.Logf("== 0xCA sur %d paquets ==", paquets)
	t.Logf("M1 : type 21 (unit_zoom) x%d · type 20 (incident) x%d · autres x%d",
		type21, type20, autresTypes)
	t.Logf("M2 : ref0 (l'unite, domaine 4) : %d index distincts, absente x%d — distribution : %s",
		len(ref0Idx), ref0Abs, lot1TopU64(ref0Idx, 12))
	t.Logf("M2 : charge R(2) (niveau+1) : %s", lot1TopU64(zoomVals, 4))
	t.Logf("M4 : listes multiples (continuation=1) x%d — 2e type : %s", cont1, lot1TopU64(deuxieme, 6))
	t.Logf("M3 : trame apres l'evenement : fermee proprement %d / non %d (%.1f %%) · "+
		"deltas lies aboutis %d, masques 1..7 : %.1f %%",
		trameOK, trameKO, lot1Pct(trameOK, trameOK+trameKO), deltasLies, lot1Pct(masquesOK, deltasLies))
	okM3 := lot1Pct(trameOK, trameOK+trameKO) >= 18 && lot1Pct(masquesOK, deltasLies) >= 80 && deltasLies >= 30
	t.Logf("VERDICT M3 (fermeture >= 18 %%, masques >= 80 %%, n >= 30) : %s", lot1Verdict(okM3))
}

// lot1NomsTypes : noms des types d'evenements frequents (annexe A du lot E, extraits).
var lot1NomsTypes = map[int]string{
	0: "damage_aftermath", 1: "damage_section_response", 2: "restore_damage_section",
	4: "item_detonate_countdown", 5: "projectile_detonate", 6: "projectile_impact_effect",
	7: "projectile_object_impact_effect", 8: "biped_board_vehicle", 9: "biped_pickup",
	10: "weapon_effect", 11: "weapon_empty_click", 12: "biped_melee_clang",
	14: "PlayEffectOnObject", 15: "Script", 17: "Allegiance", 20: "incident",
	21: "unit_zoom", 22: "unit_exit_vehicle", 30: "biped_equipment_activation",
	32: "unit_teleported", 36: "action_weapon_fire", 37: "weapon_overheat",
	38: "weapon_reload", 39: "biped_throw_initiate", 40: "biped_melee_initiate",
	42: "biped_dodge", 43: "initiate_mobility_action", 44: "weapon_pickup",
	45: "weapon_put_away", 46: "weapon_drop", 47: "weapon_throw", 51: "biped_throw_release",
	52: "biped_melee_damage", 53: "unit_enter_vehicle", 56: "request_projectile_attach",
	58: "projectile_supercombine_request", 62: "player_loadout_request",
	66: "PlayerEmote", 67: "player_force_base_respawn", 74: "AISetMotorProgram",
	75: "AIDialog", 76: "Dialogue2D", 77: "DebugSendCameraPosition", 78: "ai_jump",
	82: "PlayerGameEventSmall", 83: "TeamGameEvent", 85: "PlayerKilledEvent",
	93: "activate_spartan_ability", 100: "PowerUpApplied", 102: "NetworkedActionRequest",
	103: "EquipmentSpawnedObject", 104: "EquipmentKnockbackPlayer",
	105: "EquipmentObjectKnockedBack", 106: "ObjectCollisionDamage", 114: "MusicMarker",
	115: "synchronized_teleport", 116: "teleport_effects", 120: "PlayerCalloutRequest",
}

// TestLot1TypesCorpus — LA TABLE DES EVENEMENTS DE TETE SUR LE CORPUS ENTIER. Sous le
// modele M, type = ((octet0 & 0x3F) << 1) | bit8 pour tout paquet a bit 1 = 1. Une passe
// sur tous les films de la garde LOT1_CORPUS (racine du cache) : distribution des types,
// avec noms. Recensement pur ; ~3 min sur 1 367 films.
func TestLot1TypesCorpus(t *testing.T) {
	racine := os.Getenv("LOT1_CORPUS")
	if racine == "" {
		t.Skipf("LOT1_CORPUS absent : instrument saute")
	}
	entries, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	types := map[int]int{}
	sansEvenement, films := 0, 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := racine + string(os.PathSeparator) + ent.Name()
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		films++
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 {
					sansEvenement++
					continue
				}
				types[int((pay[0]&0x3F)<<1)|int(pay[1]>>7)]++
			}
		}
	}
	t.Logf("== %d films · %d paquets sans evenement de tete (bit 1 = 0) ==", films, sansEvenement)
	type kv struct{ typ, n int }
	var s []kv
	for k, v := range types {
		s = append(s, kv{k, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].n > s[j].n })
	for _, e := range s {
		nom := lot1NomsTypes[e.typ]
		if nom == "" {
			nom = "?"
		}
		t.Logf("  type %3d %-36s : %9d", e.typ, nom, e.n)
	}
}

// TestLot1TirsEtCibles — LA PRECISION (demande utilisateur) : le type 36
// (action_weapon_fire, 2,5 M sur le corpus) porte un COMPTE DE CIBLES ; un evenement a
// zero cible est un tir/degat SANS cible touchee. Cet instrument decode, bit-exact,
// l'evenement de tete des paquets 0xD2 jusqu'aux deux comptes :
//
//	[config][cont][R(7)=36]
//	ref0 (ATTAQUANT, domaine 1) : R(1) ; si 1 : R(1) sonde ; R(sonde?9:13) ; R(2)
//	ref1 (dom 8) / ref2 (dom 7) : R(1) ; si 1 : R(13) ; R(2)
//	R(1) court · R(1) blocSup · R(7)+R(1) · [R(1);si 1:R(5)] · [R(1);si 0:R(2)] ·
//	[R(1);si 1:R(32)] · R(32) variant_name (ARME) · R(1) · R(1) ·
//	[si blocSup : R(1) + R(1) ; si ce dernier : horodatage NON RESOLU -> paquet ecarte]
//	si court : FIN (pas de cibles) ; sinon comptes :
//	R(1) toutVide | cibles=[R(1):1|R(4)] puis composantes=[R(1):0|[R(1):1|R(4)]]
//
// PUBLIE : parts de court/blocSup, DISTRIBUTION DES CIBLES (dont % zero cible), des
// composantes, les attaquants (ref0) et les armes (variant_name). Criteres avant mesure :
// T1 — type lu = 36 sur ~100 % des 0xD2 ; T2 — les comptes sont petits (cibles <= 8,
// composantes <= 12 pour l'essentiel) sinon le cadrage de la charge est faux ; T3 — ref0
// present sur la quasi-totalite (un tir a un tireur).
func TestLot1TirsEtCibles(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	var (
		paquets, mauvaisType, court, blocSup, ecartes int
		refAbs                                        int
		cibles                                        = map[uint64]int{}
		composantes                                   = map[uint64]int{}
		attaquants                                    = map[uint64]int{}
		armes                                         = map[uint64]int{}
	)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			paquets++
			br := NewBitReader(pay)
			br.Skip(2) // config + continuation
			if br.ReadBits(7) != 36 {
				mauvaisType++
				continue
			}
			// ref0 = l'attaquant (domaine 1, sonde)
			if br.ReadBit() {
				w := 13
				if br.ReadBit() {
					w = 9
				}
				attaquants[br.ReadBits(uint(w))]++
				br.Skip(2)
			} else {
				refAbs++
			}
			for range 2 { // ref1 dom 8, ref2 dom 7 : R(13) + R(2)
				if br.ReadBit() {
					br.Skip(15)
				}
			}
			estCourt := br.ReadBit()
			estBloc := br.ReadBit()
			br.Skip(8) // R(7)+R(1)
			if br.ReadBit() {
				br.Skip(5)
			}
			if !br.ReadBit() {
				br.Skip(2)
			}
			if br.ReadBit() {
				br.Skip(32)
			}
			armes[br.ReadBits(32)]++ // variant_name
			br.Skip(2)               // R(1) out[0x1d] + R(1) out[2]
			if estBloc {
				blocSup++
				br.Skip(1)
				if br.ReadBit() { // horodatage non resolu
					ecartes++
					continue
				}
			}
			if estCourt {
				court++
				continue
			}
			var nCibles, nComp uint64
			if !br.ReadBit() { // toutVide ?
				if br.ReadBit() {
					nCibles = 1
				} else {
					nCibles = br.ReadBits(4)
				}
				if !br.ReadBit() {
					if br.ReadBit() {
						nComp = 1
					} else {
						nComp = br.ReadBits(4)
					}
				}
			}
			cibles[nCibles]++
			composantes[nComp]++
		}
	}
	t.Logf("== 0xD2 : %d paquets · type != 36 : %d · ecartes (horodatage bloc) : %d ==",
		paquets, mauvaisType, ecartes)
	t.Logf("T3 : attaquant (ref0) : %d index distincts, absent x%d — %s",
		len(attaquants), refAbs, lot1TopU64(attaquants, 12))
	t.Logf("chemin court x%d (%.1f %%) · blocSup x%d", court, lot1Pct(court, paquets), blocSup)
	t.Logf("CIBLES par evenement (chemin long) : %s", lot1TopU64(cibles, 8))
	zero := cibles[0]
	total := 0
	for _, v := range cibles {
		total += v
	}
	t.Logf("  -> ZERO CIBLE (tir/degat sans cible touchee) : %d / %d (%.1f %%)",
		zero, total, lot1Pct(zero, total))
	t.Logf("composantes de degat : %s", lot1TopU64(composantes, 8))
	t.Logf("armes (variant_name) : %d distinctes — %s", len(armes), lot1TopU64(armes, 8))
}

// lot1TopU64 rend les k entrees les plus frequentes d'un histogramme a cle entiere.
func lot1TopU64(m map[uint64]int, k int) string {
	type kv struct {
		k uint64
		v int
	}
	var s []kv
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	if len(s) > k {
		s = s[:k]
	}
	out := ""
	for i, e := range s {
		if i > 0 {
			out += " · "
		}
		out += fmt.Sprintf("%d x%d", e.k, e.v)
	}
	return out
}
