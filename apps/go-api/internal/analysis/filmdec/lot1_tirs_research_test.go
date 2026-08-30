package filmdec

// lot1_tirs_research_test.go — LOT 1 : LES INSTRUMENTS DE RECENSEMENT CORPUS ET DE
// DECODAGE PAR TYPE (suite de lot1_evenements_research_test.go, scinde pour le seuil de
// 500 lignes). Table des types sur le corpus, decodage des tirs (type 36, la voie
// precision + le juge visee R(30)), entrees/sorties de vehicule.

import (
	"os"
	"sort"
	"testing"
)

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
		refAbs, portesNonVides, viseeOK, viseeKO      int
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
			// LE JUGE DU CADRAGE : dans le cas modal (0 composante, 0 cible, portes des
			// deux sous-lecteurs a 0), la VISEE R(30) est a 3 bits d'ici. Un vecteur
			// unitaire valide (DecodeAimVectorChecked) prouve la chaine ENTIERE.
			if nCibles != 0 || nComp != 0 {
				continue // les boucles exigent des sous-lecteurs non resolus
			}
			if br.ReadBit() { // b1 de FUN_1406cd5b8
				portesNonVides++
				continue
			}
			if br.ReadBit() { // b2
				portesNonVides++
				continue
			}
			if br.ReadBit() { // porte de FUN_1408eff64
				portesNonVides++
				continue
			}
			if v, ok := DecodeAimVectorChecked(uint32(br.ReadBits(30)), 30); ok {
				viseeOK++
				_ = v
			} else {
				viseeKO++
			}
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
	t.Logf("LE JUGE — visee R(30) au bout de la chaine (cas modal) : VALIDE %d · invalide %d · "+
		"portes non vides (sautes) %d", viseeOK, viseeKO, portesNonVides)
	t.Logf("VERDICT cadrage complet (>= 90 %% de vecteurs valides, n >= 50) : %s",
		lot1Verdict(viseeOK >= 50 && lot1Pct(viseeOK, viseeOK+viseeKO) >= 90))
}

// TestLot1Vehicules — ENTREES ET SORTIES DE VEHICULE (demande utilisateur). Octet de tete =
// 0xC0 | (type>>1) (config=1, continuation=1) ; bit8 = type&1 dans l'octet suivant. Types :
//
//	8  biped_board_vehicle   octet 0xC4 (bit8=0) — charge R(6) (le SIEGE), lecteur 0x142f168c0
//	53 unit_enter_vehicle    octet 0xDA (bit8=1) — MEME lecteur que 8 (charge R(6))
//	22 unit_exit_vehicle     octet 0xCB (bit8=0) — R(6) + R(1) + suite
//
// Reference domaine du type 8/53 (vtable+0x58 de 0x144724e20) : ref0 = domaine 1 (l'unite
// qui embarque/debarque, avec sonde), ref1/ref2 domaines a lire. On decode l'en-tete
// (attaquant + charge R(6) = siege) et on publie qui monte/descend, dans quel siege.
// Recensement, sans seuil. Passe corpus (garde LOT1_CORPUS) car les temoins arene ont peu
// de vehicules.
func TestLot1Vehicules(t *testing.T) {
	racine := os.Getenv("LOT1_CORPUS")
	if racine == "" {
		t.Skipf("LOT1_CORPUS absent : instrument saute")
	}
	entries, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	// octet de tete -> (nom, bit8 attendu)
	type vt struct {
		nom  string
		typ  int
		exit bool
	}
	vehic := map[byte]vt{0xC4: {"biped_board_vehicle", 8, false},
		0xDA: {"unit_enter_vehicle", 53, false}, 0xCB: {"unit_exit_vehicle", 22, true}}
	compte := map[string]int{}
	sieges := map[string]map[uint64]int{}
	unites := map[string]map[uint64]int{}
	filmsAvecVehic := map[string]int{}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := racine + string(os.PathSeparator) + ent.Name()
		n := CountFilmChunks(dir)
		vus := map[string]bool{}
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
				v, ok := vehic[pay[0]]
				if !ok {
					continue
				}
				br := NewBitReader(pay)
				br.Skip(2) // config + continuation
				if int(br.ReadBits(7)) != v.typ {
					continue
				}
				// ref0 = l'unite (domaine 1, sonde) ; ref1/ref2 (domaines 8/7 supposes)
				var unite uint64
				hasU := false
				if br.ReadBit() {
					w := 13
					if br.ReadBit() {
						w = 9
					}
					unite = br.ReadBits(uint(w))
					br.Skip(2)
					hasU = true
				}
				for range 2 {
					if br.ReadBit() {
						br.Skip(15)
					}
				}
				siege := br.ReadBits(6) // charge R(6) : le siege
				compte[v.nom]++
				vus[v.nom] = true
				if sieges[v.nom] == nil {
					sieges[v.nom], unites[v.nom] = map[uint64]int{}, map[uint64]int{}
				}
				sieges[v.nom][siege]++
				if hasU {
					unites[v.nom][unite]++
				}
			}
		}
		for nom := range vus {
			filmsAvecVehic[nom]++
		}
	}
	for _, nom := range []string{"biped_board_vehicle", "unit_enter_vehicle", "unit_exit_vehicle"} {
		t.Logf("  %-22s : %d evenements sur %d films · siege R(6) : %s · unites (ref0) : %d distinctes %s",
			nom, compte[nom], filmsAvecVehic[nom], lot1TopU64(sieges[nom], 8),
			len(unites[nom]), lot1TopU64(unites[nom], 6))
	}
}
