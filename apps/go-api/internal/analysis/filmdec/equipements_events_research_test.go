package filmdec

// equipements_events_research_test.go — lot R5 du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03 :
// quels equipements (hors translocateur, deja etabli par R1) ont un evenement d'usage dans le
// canal des evenements NOMMES du film, et avec quelle fiabilite mesuree ?
//
// CANAL MESURE : la liste d'evenements en tete de chaque paquet delta (modele M,
// NOTE_MODELE_EVENEMENTS_2026-08-30 ; grammaire GRAMMAIRE_EVENTS_FILM_2026-08-30) :
//
//	[1 bit config][ ( 1 [R(7) type] [3 refs gardees] [charge] )* 0 ][trame de records]
//
// SEULE LA TETE de liste est lisible sans la grammaire des charges de tous les types (reserve
// identique a R1 / zoom_events.go) : le recensement est O(1) par paquet, pas un decodage.
//
// VERITE TERRAIN (artefacts data/cache/replays/halo_infinite/<id8>.json, horloge FILM) :
//   - grappleLines[]           : tirs de grappin avec traction, dates (t0, slot) ;
//   - equipmentPlacements[]    : poses origin=deployed des equipements deployables
//     (mur, capteur, ecran, champ de reparation, threat_seeker), datees (t0, owner) ;
//   - equipmentEpisodes[]      : episodes camo / surbouclier, dates (t0, slot).
//
// PIEGE D'HORLOGE (rapport R1 par.0) : paquets en horloge MOTEUR, artefacts en horloge FILM.
// Conversion : ms_film = (ts_paquet - ts_premier_paquet_chunk1) / 1000 ;
// verite terrain : ms_film = originMs + t_frame * frameIntervalMs.
//
// REF0 : hypothese mesuree par R1 sur le type 117 (porte 1 bit, index 8 bits base 512,
// generation 2 bits, domaine 2). Les domaines des autres types ne sont pas sources de l'exe :
// l'hypothese w=9 est decodee en parallele et la fermeture se fait par le croisement.
//
// LECTURE SEULE, garde par EQUIP_EVENTS_ROOT/ARTS/IDS, skip par defaut, CGO_ENABLED=0.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 \
//	  EQUIP_EVENTS_ROOT=<repo>/data/cache/film_chunks \
//	  EQUIP_EVENTS_ARTS=<repo>/data/cache/replays/halo_infinite \
//	  EQUIP_EVENTS_IDS=000d5950,06dfe6d9,084a804d,4f77afc1,8a485699,1b2d9e08,a0c36016 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipementsEvenements$' -timeout 30m -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	equipEvRootEnv = "EQUIP_EVENTS_ROOT"
	equipEvArtsEnv = "EQUIP_EVENTS_ARTS"
	equipEvIDsEnv  = "EQUIP_EVENTS_IDS"
	// equipEvFenetreMS : rayon d'appariement evenement <-> usage mesure (R1 : le 117 tombe
	// 5-80 ms avant le saut ; la verite terrain est sur une grille de 100 ms et les poses
	// peuvent etre vues avec retard — fenetre large, le rapport donne les medianes).
	equipEvFenetreMS = 1200
	// equipEvFenetreSerreeMS : fenetre serree, comptee en plus pour juger la datation.
	equipEvFenetreSerreeMS = 300
)

// equipEvSuspects : liste FERMEE R5.1 — types de l'annexe A (GRAMMAIRE_EVENTS_FILM_2026-08-30)
// dont le nom exe parle d'equipement ou d'usage de capacite. Taille = tampon de reception.
var equipEvSuspects = map[int]string{
	28:  "biped_debug_teleport",                 // taille 12 — teleport (adjacent)
	30:  "biped_equipment_activation",           // taille 72 — LE candidat generique d'usage
	31:  "equipment_teleport_request",           // taille 8  — requete translocateur
	39:  "biped_throw_initiate",                 // taille 20 — lancer (grenades probables)
	42:  "biped_dodge",                          // taille 24 — esquive (propulseur ?)
	43:  "initiate_mobility_action",             // taille 164 — mobilite (propulseur ?)
	48:  "weapon_tether_request",                // taille 20 — cable (grappin ?)
	51:  "biped_throw_release",                  // taille 68 — lancer (grenades probables)
	93:  "activate_spartan_ability",             // taille 64 — activation de capacite
	98:  "Equipment",                            // taille 8  — 9 bits fixes
	100: "PowerUpApplied",                       // taille 12 — camo / surbouclier ?
	103: "EquipmentSpawnedObject",               // taille 0  — objet cree par un equipement
	104: "EquipmentKnockbackPlayer",             // taille 12 — repulseur (joueur pousse)
	105: "EquipmentObjectKnockedBack",           // taille 4  — repulseur (objet pousse)
	115: "synchronized_teleport",                // taille 44 — teleport (adjacent)
	116: "teleport_effects",                     // taille 56 — teleport (adjacent)
	117: "EquipmentTranslocatorTeleportEffects", // taille 28 — etabli par R1 (18/18)
	118: "repair_complete",                      // taille 4  — reparation
	119: "EquipmentKnockbackRequest",            // taille 276 — repulseur (requete)
}

// equipEvDeployables : familles d'equipementPlacements dont une pose origin=deployed est un
// USAGE d'equipement (les familles grenade_* sont des grenades, hors perimetre R5 ; les
// deployed de familles non deployables — grapple, thruster, repulsor — sont des attributions
// douteuses du canal placements et sont comptes a part, pas comme verite).
var equipEvDeployables = map[string]bool{
	"wall": true, "sensor": true, "shroud_screen": true,
	"repair_field": true, "threat_seeker": true,
}

// equipEvArtefact : sous-ensemble minimal de l'artefact du rejeu.
type equipEvArtefact struct {
	OriginMs        int64 `json:"originMs"`
	FrameIntervalMs int64 `json:"frameIntervalMs"`
	GrappleLines    []struct {
		Slot int   `json:"slot"`
		T0   int64 `json:"t0"`
	} `json:"grappleLines"`
	EquipmentPlacements []struct {
		T0     int64  `json:"t0"`
		Family string `json:"family"`
		Owner  int    `json:"owner"`
		Origin string `json:"origin"`
	} `json:"equipmentPlacements"`
	EquipmentEpisodes []struct {
		Slot int    `json:"slot"`
		Fam  string `json:"fam"`
		T0   int64  `json:"t0"`
		T1   int64  `json:"t1"`
	} `json:"equipmentEpisodes"`
}

// equipEvTruth : un usage mesure (verite terrain), en ms d'horloge FILM.
type equipEvTruth struct {
	slot  int
	tMs   int64
	canal string // grapple | deploy:<famille> | camo | overshield
}

// equipEvOcc : une tete de liste d'un type suspect.
type equipEvOcc struct {
	tsMS   int64 // ms d'horloge FILM
	typ    int
	chunk  int
	paquet int
	slot8  int    // ref0 sous w=8 base 512 ; -1 si porte 0
	slot9  int    // ref0 sous w=9 base 512 ; -1 si porte 0
	tete   []byte // premiers octets du payload, pour les hypotheses de chaine de refs
}

// equipEvCanauxTemps : croisement TEMPS SEUL (sans identification de slot) — pour les types
// dont la ref0 ne rend PAS le slot du porteur sous w=8/w=9 : le type date-t-il quand meme le
// geste ? prefixes de canal a apparier par type.
var equipEvCanauxTemps = map[int][]string{
	100: {"camo", "overshield"},
	103: {"deploy:", "place:"},
	105: {"deploy:", "camo", "overshield", "grapple"}, // pas de verite repulseur datee : info seulement
	118: {"deploy:repair_field"},
}

// TestEquipementsEvenements recense les tetes des types suspects sur chaque film et les
// croise avec la verite terrain de l'artefact. Precision et rappel PAR TYPE, denominateurs
// dits, negatifs dits.
func TestEquipementsEvenements(t *testing.T) {
	root := os.Getenv(equipEvRootEnv)
	arts := os.Getenv(equipEvArtsEnv)
	ids := strings.Split(strings.TrimSpace(os.Getenv(equipEvIDsEnv)), ",")
	if root == "" || arts == "" || len(ids) == 0 || ids[0] == "" {
		t.Skipf("instrument R5 : definir %s, %s et %s", equipEvRootEnv, equipEvArtsEnv, equipEvIDsEnv)
	}
	globalCensus := map[int]int{}
	globalTotal := 0
	// agregats parc : par type -> [apparies, total] et par canal x type -> [rappeles, total]
	precAgg := map[int][2]int{}
	rappelAgg := map[string]map[int][2]int{}
	tempsPrecAgg := map[int][2]int{}
	tempsRappelAgg := map[string]map[int][2]int{}
	sonde103Agg := map[string]int{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		t.Logf("")
		t.Logf("############ FILM %s ############", id)
		occs, census, total := equipEvRecense(t, filepath.Join(root, id))
		for typ, n := range census {
			globalCensus[typ] += n
		}
		globalTotal += total
		truths := equipEvVerite(t, filepath.Join(arts, id+".json"))
		equipEvCroise(t, occs, truths, precAgg, rappelAgg)
		equipEvTempsSeul(t, occs, truths, tempsPrecAgg, tempsRappelAgg)
		equipEvSonde103(t, occs, sonde103Agg)
	}
	t.Logf("")
	t.Logf("############ PARC (%d films, %d paquets delta) ############", len(ids), globalTotal)
	equipEvCensusGlobal(t, globalCensus, globalTotal)
	equipEvBilan(t, precAgg, rappelAgg)
	equipEvBilanTemps(t, tempsPrecAgg, tempsRappelAgg)
	t.Logf("== BILAN PARC — SONDE LISTE (tetes 103, hypothese refs 13+2 bits) : %v ==", sonde103Agg)
}

// equipEvRecense balaie les paquets delta d'un film : recensement O(1) des tetes de liste,
// occurrences detaillees pour les types suspects. Rend aussi l'origine d'horloge appliquee.
func equipEvRecense(t *testing.T, dir string) ([]equipEvOcc, map[int]int, int) {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	var origine uint64
	raw, err := ReadFilmChunk(dir, 1)
	if err != nil {
		t.Fatalf("chunk 1 illisible : %v", err)
	}
	if pks := WalkPackets(raw); len(pks) > 0 {
		origine = pks[0].TimestampUS
	} else {
		t.Fatalf("aucun paquet dans le chunk 1 de %s", dir)
	}
	census := map[int]int{}
	total := 0
	var occs []equipEvOcc
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			total++
			pay := pk.Payload(data)
			if pay[0]&0xC0 != 0xC0 { // bit config + bit continuation : liste non vide
				continue
			}
			typ := int(pay[0]&0x3F)<<1 | int(pay[1]>>7)
			census[typ]++
			if equipEvSuspects[typ] == "" {
				continue
			}
			s8, s9 := equipEvRef0(pay)
			tete := make([]byte, 0, 24)
			if n := len(pay); n > 24 {
				tete = append(tete, pay[:24]...)
			} else {
				tete = append(tete, pay...)
			}
			occs = append(occs, equipEvOcc{
				tsMS: (int64(pk.TimestampUS) - int64(origine)) / 1000,
				typ:  typ, chunk: c, paquet: pk.Index, slot8: s8, slot9: s9, tete: tete,
			})
		}
	}
	t.Logf("== %d chunks, %d paquets delta, %d occurrences de types suspects ==", n, total, len(occs))
	return occs, census, total
}

// equipEvRef0 decode la reference 0 sous les hypotheses w=8 et w=9 (base 512). -1 = porte 0.
func equipEvRef0(pay []byte) (int, int) {
	s8, s9 := -1, -1
	for _, w := range []uint{8, 9} {
		br := NewBitReader(pay)
		br.Skip(9) // config + continuation + R(7) type
		if br.Remaining() < int(w)+3 || !br.ReadBit() {
			continue
		}
		v := int(br.ReadBits(w)) + 512
		if w == 8 {
			s8 = v
		} else {
			s9 = v
		}
	}
	return s8, s9
}

// equipEvVerite charge la verite terrain de l'artefact, en ms d'horloge FILM.
func equipEvVerite(t *testing.T, path string) []equipEvTruth {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artefact illisible %s : %v", path, err)
	}
	var a equipEvArtefact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("artefact indecodable %s : %v", path, err)
	}
	if a.FrameIntervalMs == 0 {
		a.FrameIntervalMs = 100
	}
	frameMS := func(fr int64) int64 { return a.OriginMs + fr*a.FrameIntervalMs }
	var out []equipEvTruth
	for _, g := range a.GrappleLines {
		out = append(out, equipEvTruth{slot: g.Slot, tMs: frameMS(g.T0), canal: "grapple"})
	}
	douteux := map[string]int{}
	for _, p := range a.EquipmentPlacements {
		if p.Origin == "deployed" && !strings.HasPrefix(p.Family, "grenade") && equipEvDeployables[p.Family] {
			out = append(out, equipEvTruth{slot: p.Owner, tMs: frameMS(p.T0), canal: "deploy:" + p.Family})
			continue
		}
		// Toute autre pose (dropped / unknown / deployed hors perimetre, grenades comprises) :
		// canal place:* — PAS un usage d'equipement, mais un objet d'equipement APPARU (mort
		// du porteur, churn) — sert a departager les tetes 103 non appariees aux deploys.
		if p.Origin == "deployed" {
			douteux[p.Family]++
		}
		out = append(out, equipEvTruth{slot: p.Owner, tMs: frameMS(p.T0), canal: "place:" + p.Origin})
	}
	// Episodes camo/surbouclier : un usage peut se FRAGMENTER en plusieurs episodes (le camo
	// se rompt au tir puis revient). On publie les deux verites : chaque episode (canal nu),
	// et le DEBUT de groupe (canal suffixe _debut, ecart > 5 s entre episodes du meme slot)
	// comme approximation d'activation — dit comme tel, ce n'est pas une mesure d'activation.
	eps := append([]struct {
		Slot int    `json:"slot"`
		Fam  string `json:"fam"`
		T0   int64  `json:"t0"`
		T1   int64  `json:"t1"`
	}(nil), a.EquipmentEpisodes...)
	sort.Slice(eps, func(i, j int) bool {
		if eps[i].Fam != eps[j].Fam {
			return eps[i].Fam < eps[j].Fam
		}
		if eps[i].Slot != eps[j].Slot {
			return eps[i].Slot < eps[j].Slot
		}
		return eps[i].T0 < eps[j].T0
	})
	dernier := map[string]int64{}
	for _, e := range eps {
		tMs := frameMS(e.T0)
		out = append(out, equipEvTruth{slot: e.Slot, tMs: tMs, canal: e.Fam})
		cle := fmt.Sprintf("%s/%d", e.Fam, e.Slot)
		if prev, okPrev := dernier[cle]; !okPrev || tMs-prev > 5000 {
			out = append(out, equipEvTruth{slot: e.Slot, tMs: tMs, canal: e.Fam + "_debut"})
		}
		dernier[cle] = frameMS(e.T1)
	}
	parCanal := map[string]int{}
	for _, tr := range out {
		parCanal[tr.canal]++
	}
	t.Logf("== VERITE TERRAIN : %d usages %v ; deployed douteux ignores %v ==", len(out), parCanal, douteux)
	return out
}

// equipEvCroise apparie evenements et usages du meme slot, rapporte precision et rappel par
// type pour CE film, et verse les comptes dans les agregats parc.
func equipEvCroise(t *testing.T, occs []equipEvOcc, truths []equipEvTruth,
	precAgg map[int][2]int, rappelAgg map[string]map[int][2]int) {
	t.Helper()
	// Precision : chaque occurrence d'un type suspect trouve-t-elle un usage mesure du meme
	// slot dans la fenetre ? (w=8 d'abord ; w=9 compte a part pour la fermeture de largeur).
	types := equipEvTypesPresents(occs)
	for _, typ := range types {
		var tot, m8, m9, serres int
		var deltas []int64
		canaux := map[string]int{}
		for _, o := range occs {
			if o.typ != typ {
				continue
			}
			tot++
			if c, dt, ok := equipEvMatch(o.slot8, o.tsMS, truths); ok {
				m8++
				canaux[c]++
				deltas = append(deltas, dt)
				if dt >= -equipEvFenetreSerreeMS && dt <= equipEvFenetreSerreeMS {
					serres++
				}
				continue
			}
			if _, _, ok := equipEvMatch(o.slot9, o.tsMS, truths); ok {
				m9++
			}
		}
		agg := precAgg[typ]
		agg[0] += m8
		agg[1] += tot
		precAgg[typ] = agg
		t.Logf("  PRECISION type %3d %-36s : %d/%d apparies w8 (dont %d a +/-%d ms) ; +%d w9 seul ; canaux %v ; dt med %s",
			typ, equipEvSuspects[typ], m8, tot, serres, equipEvFenetreSerreeMS, m9, canaux, equipEvMedianeMS(deltas))
	}
	// Rappel : chaque usage mesure a-t-il un evenement de chaque type suspect (meme slot) ?
	canaux := map[string]bool{}
	for _, tr := range truths {
		canaux[tr.canal] = true
	}
	for canal := range canaux {
		if rappelAgg[canal] == nil {
			rappelAgg[canal] = map[int][2]int{}
		}
		for _, typ := range types {
			var tot, ok int
			for _, tr := range truths {
				if tr.canal != canal {
					continue
				}
				tot++
				if equipEvRappel(tr, typ, occs) {
					ok++
				}
			}
			agg := rappelAgg[canal][typ]
			agg[0] += ok
			agg[1] += tot
			rappelAgg[canal][typ] = agg
			if ok > 0 { // par film, ne detailler que les types qui rappellent au moins un usage
				t.Logf("  RAPPEL %-22s par type %3d %-36s : %d/%d", canal, typ, equipEvSuspects[typ], ok, tot)
			}
		}
	}
}

// equipEvMatch cherche l'usage du meme slot le plus proche en temps ; ok si |dt| <= fenetre.
// dt = t_evenement - t_usage (negatif : l'evenement precede la datation de l'usage).
func equipEvMatch(slot int, tsMS int64, truths []equipEvTruth) (string, int64, bool) {
	if slot < 0 {
		return "", 0, false
	}
	best, canal, found := int64(0), "", false
	for _, tr := range truths {
		if tr.slot != slot {
			continue
		}
		dt := tsMS - tr.tMs
		if !found || equipEvAbs(dt) < equipEvAbs(best) {
			best, canal, found = dt, tr.canal, true
		}
	}
	if !found || equipEvAbs(best) > equipEvFenetreMS {
		return "", 0, false
	}
	return canal, best, true
}

// equipEvRappel : existe-t-il une occurrence du type, meme slot (w=8), dans la fenetre ?
func equipEvRappel(tr equipEvTruth, typ int, occs []equipEvOcc) bool {
	for _, o := range occs {
		if o.typ == typ && o.slot8 == tr.slot && equipEvAbs(o.tsMS-tr.tMs) <= equipEvFenetreMS {
			return true
		}
	}
	return false
}

func equipEvAbs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func equipEvMedianeMS(deltas []int64) string {
	if len(deltas) == 0 {
		return "-"
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
	return fmt.Sprintf("%+d ms", deltas[len(deltas)/2])
}

func equipEvTypesPresents(occs []equipEvOcc) []int {
	seen := map[int]bool{}
	for _, o := range occs {
		seen[o.typ] = true
	}
	types := make([]int, 0, len(seen))
	for typ := range seen {
		types = append(types, typ)
	}
	sort.Ints(types)
	return types
}

// equipEvCensusGlobal : la distribution parc des types en tete de liste — le DENOMINATEUR.
func equipEvCensusGlobal(t *testing.T, census map[int]int, total int) {
	t.Helper()
	types := make([]int, 0, len(census))
	for typ := range census {
		types = append(types, typ)
	}
	sort.Ints(types)
	t.Logf("== CENSUS PARC : %d paquets delta, %d types en tete ==", total, len(types))
	for _, typ := range types {
		nom := equipEvSuspects[typ]
		if nom == "" {
			nom = "-"
		}
		t.Logf("  type %3d : %7d tetes  %s", typ, census[typ], nom)
	}
}
