package filmdec

// faille_activation_events_research_test.go — R1.2 : quand un joueur ACTIVE le translocateur,
// un EVENEMENT du film tombe-t-il aux instants des ancres (pose et/ou saut) ?
//
// CANAL MESURE : la liste d'événements en tête de chaque paquet delta (modèle M,
// NOTE_MODELE_EVENEMENTS_2026-08-30) :
//
//	[1 bit config][ ( 1 [R(7) type] [3 refs gardées] [charge] )* 0 ][trame de records]
//
// Un paquet dont la liste ouvre sur un événement a un premier octet 0xC0|type>>1, le bit de
// poids faible du type étant le 1er bit du 2e octet. SEUL L'EVENEMENT DE TETE est lisible sans
// connaître la charge de tous les types (réserve identique à zoom_events.go) — le recensement
// ci-dessous compte donc les TETES de liste, et le dit.
//
// TYPES SUSPECTS (annexe A de GRAMMAIRE_EVENTS_FILM_2026-08-30) : 103 EquipmentSpawnedObject
// (charge 0 bit — « l'équipement a fait apparaître un objet », le candidat naturel de la
// pose), 98 Equipment, 100 PowerUpApplied, 104/105 EquipmentKnockback*, 115
// synchronized_teleport, 116 teleport_effects, 32 unit_teleported (85 % du flux : bruit de
// réplication, compté pour l'exclure), 108 NavpointRequest, 114 MusicMarker.
//
// LECTURE SEULE, gardé par FAILLE_FILM. Aucun décodage de trame (pas de verrou nécessaire),
// mais le recensement global lit l'octet de tête de TOUS les paquets : c'est une lecture O(1)
// par paquet, pas un balayage bit à bit.
//
// USAGE (depuis apps/go-api) — mêmes variables que TestFailleActivationEntites :
//
//	CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/1b2d9e08 \
//	  FAILLE_BOUNDS=-11.45,104.54,73.91,19.72,153.51,82.53 \
//	  FAILLE_ANCRES="A1:535:17.34,135.50:146862000-185262000:185262000;A2:560:18.34,120.19:328162000-351062000:351062000" \
//	  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEvenements$' -timeout 20m -v

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// failleTypesSuspects : les types d'événement à détailler occurrence par occurrence dans les
// fenêtres. Les noms viennent de l'annexe A (table corrigée, permutation lue dans l'exe).
var failleTypesSuspects = map[int]string{
	28:  "biped_debug_teleport",
	30:  "biped_equipment_activation",
	31:  "equipment_teleport_request",
	32:  "unit_teleported",
	33:  "vehicle_auto_turret_choose_target",
	98:  "Equipment",
	100: "PowerUpApplied",
	101: "LoadForgeObjectGroup",
	102: "NetworkedActionRequest",
	103: "EquipmentSpawnedObject",
	104: "EquipmentKnockbackPlayer",
	105: "EquipmentObjectKnockedBack",
	108: "NavpointRequest",
	114: "MusicMarker",
	115: "synchronized_teleport",
	116: "teleport_effects",
	117: "EquipmentTranslocatorTeleportEffects",
	119: "EquipmentKnockbackRequest",
}

// failleJumpFenetreUS : rayon autour de l'instant EXACT du saut pour le rapprochement fin.
const failleJumpFenetreUS = 500_000

// failleEvOcc est une occurrence d'événement de tête dans une fenêtre d'ancre.
type failleEvOcc struct {
	ts     uint64
	typ    int
	ancre  int
	chunk  int
	paquet int
	tete   []byte // premiers octets du payload (pour le décodage manuel des références)
}

// TestFailleActivationEvenements recense les événements de tête, film entier puis fenêtres.
func TestFailleActivationEvenements(t *testing.T) {
	dir, wr, ancres, origine := failleSetup(t)
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	global := map[int]int{}
	total, vides := 0, 0
	var tsMin, tsMax uint64
	var occs, t117 []failleEvOcc
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			if total == 0 || pk.TimestampUS < tsMin {
				tsMin = pk.TimestampUS
			}
			if pk.TimestampUS > tsMax {
				tsMax = pk.TimestampUS
			}
			total++
			pay := pk.Payload(data)
			if pay[0]&0xC0 != 0xC0 { // bit config + bit de continuation : liste non vide
				vides++
				continue
			}
			typ := int(pay[0]&0x3F)<<1 | int(pay[1]>>7)
			global[typ]++
			if typ == 117 {
				t117 = append(t117, failleEvOcc{ts: pk.TimestampUS, typ: typ, ancre: -1,
					chunk: c, paquet: pk.Index, tete: failleTete(pay)})
			}
			for _, ai := range failleFenetres(pk.TimestampUS, ancres) {
				occs = append(occs, failleEvOcc{ts: pk.TimestampUS, typ: typ, ancre: ai,
					chunk: c, paquet: pk.Index, tete: failleTete(pay)})
			}
		}
	}
	t.Logf("== HORLOGE PAQUETS (moteur) : min %d us · max %d us ==", tsMin, tsMax)
	failleRapportGlobal(t, global, total, vides)
	failleRapport117(t, t117, origine)
	failleRapportFenetres(t, occs, ancres, origine, wr)
}

// failleRapport117 liste TOUS les événements EquipmentTranslocatorTeleportEffects du film
// (têtes de liste), avec la référence d'unité décodée sous l'hypothèse mesurée sur les ancres
// (largeur 8, base 512 — fermeture 3/3 sur 1b2d9e08). C'est le rapport du RAPPEL : sur un
// film témoin, chaque spent/saut connu doit trouver ici son ou ses événements.
func failleRapport117(t *testing.T, t117 []failleEvOcc, origine uint64) {
	t.Helper()
	t.Logf("== TYPE 117 (EquipmentTranslocatorTeleportEffects) FILM ENTIER : %d têtes ==", len(t117))
	for _, o := range t117 {
		br := NewBitReader(o.tete)
		br.Skip(9)
		if !br.ReadBit() {
			t.Logf("  @%d ms : ref0 ABSENTE · tête % X", failleMS(o.ts, origine), o.tete)
			continue
		}
		idx := br.ReadBits(8)
		gen := br.ReadBits(2)
		t.Logf("  @%d ms : unité slot %d (gen %d) · chunk %d paquet %d",
			failleMS(o.ts, origine), idx+512, gen, o.chunk, o.paquet)
	}
}

// failleTete copie les premiers octets du payload (bornés) pour le rapport et les sondes.
func failleTete(pay []byte) []byte {
	n := 24
	if len(pay) < n {
		n = len(pay)
	}
	out := make([]byte, n)
	copy(out, pay[:n])
	return out
}

// failleRapportGlobal : le recensement film entier — le DENOMINATEUR de toute coïncidence.
func failleRapportGlobal(t *testing.T, global map[int]int, total, vides int) {
	t.Helper()
	types := make([]int, 0, len(global))
	for typ := range global {
		types = append(types, typ)
	}
	sort.Ints(types)
	t.Logf("== FILM ENTIER : %d paquets delta · %d listes vides · %d types en tête de liste ==",
		total, vides, len(types))
	for _, typ := range types {
		nom := failleTypesSuspects[typ]
		if nom == "" {
			nom = "-"
		}
		t.Logf("  type %3d (octet 0x%02X) : %6d têtes  %s", typ, 0xC0|typ>>1, global[typ], nom)
	}
}

// failleRapportFenetres détaille les occurrences fenêtrées : les types suspects un par un
// (avec distance temporelle au saut), les autres en volume.
func failleRapportFenetres(t *testing.T, occs []failleEvOcc, ancres []failleAncre, origine uint64, wr Vec3Range) {
	t.Helper()
	for ai, a := range ancres {
		volume := map[int]int{}
		for _, o := range occs {
			if o.ancre != ai {
				continue
			}
			volume[o.typ]++
			if failleTypesSuspects[o.typ] == "" {
				continue
			}
			dj := int64(o.ts) - int64(a.tJump)
			marque := ""
			if dj >= -int64(failleJumpFenetreUS) && dj <= int64(failleJumpFenetreUS) {
				marque = "  <== A +/- 500 ms DU SAUT"
			}
			t.Logf("  [%s] type %d %s @%d ms (saut%+d ms) chunk %d paquet %d tête % X%s",
				a.label, o.typ, failleTypesSuspects[o.typ], failleMS(o.ts, origine), dj/1000,
				o.chunk, o.paquet, o.tete, marque)
			if o.typ == 102 || o.typ == 103 || o.typ == 117 || o.typ == 30 || o.typ == 31 {
				failleRefsHypotheses(t, o, a)
				failleSondePosition(t, o, a, wr)
			}
		}
		types := make([]int, 0, len(volume))
		for typ := range volume {
			types = append(types, typ)
		}
		sort.Ints(types)
		var parts []string
		for _, typ := range types {
			parts = append(parts, fmt.Sprintf("%d:%d", typ, volume[typ]))
		}
		t.Logf("== [%s] FENETRE : %d têtes de liste · volume par type {%s} ==",
			a.label, lenOccs(occs, ai), joinParts(parts))
	}
}

func lenOccs(occs []failleEvOcc, ancre int) int {
	n := 0
	for _, o := range occs {
		if o.ancre == ancre {
			n++
		}
	}
	return n
}

func joinParts(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

// failleRefsHypotheses tente le décodage des trois références gardées d'un événement 102/103
// (charge 0 bit ou inconnue) sous une grille d'hypothèses de largeur d'index — les domaines de
// ces types ne sont pas encore sourcés de l'exe. Une hypothèse est RAPPORTEE quand la première
// référence désigne le slot du bipède de l'ancre (base 512, la fermeture mesurée du domaine 4
// dans zoom_events.go) ou un slot de la bande d'équipement (>= 1024). Un rapport n'est PAS une
// preuve : c'est une piste à re-sourcer côté exe (vtable+0x58 du descripteur).
func failleRefsHypotheses(t *testing.T, o failleEvOcc, a failleAncre) {
	t.Helper()
	widths := []uint{8, 9, 13}
	for _, w0 := range widths {
		br := NewBitReader(o.tete)
		br.Skip(9) // bit config + bit continuation + R(7) type
		if !br.ReadBit() {
			t.Logf("      refs : ref0 ABSENTE (porte 0) — aucune unité désignée en ref0")
			return
		}
		idx := br.ReadBits(w0)
		gen := br.ReadBits(2)
		cible := ""
		if uint32(idx)+512 == a.slot {
			cible = fmt.Sprintf("  <== ref0+512 == slot %d DE L'ANCRE", a.slot)
		}
		t.Logf("      hypothèse w0=%2d : ref0 idx=%d gen=%d (idx+512=%d)%s", w0, idx, gen, idx+512, cible)
	}
}

// failleSondePosition cherche, dans la charge utile d'un événement suspect, un vecteur
// quantifié 3 x 16 bits (le lecteur FUN_14076e494 des vecteurs d'événement, largeur 0x10,
// prouvé sur unit_teleported) dont la déquantification sur les bornes de la carte tombe à
// <= failleProcheM (2D) de l'ancre. Le décalage de départ est BALAYE (l'en-tête exact des
// références de ces types n'est pas encore sourcé de l'exe) : un rapport localise donc un
// CANDIDAT de position, pas une grammaire — le dire à cette hauteur, pas plus.
func failleSondePosition(t *testing.T, o failleEvOcc, a failleAncre, wr Vec3Range) {
	t.Helper()
	deq := func(q uint64, axe int) float64 {
		lo, hi := float64(wr[axe].Min), float64(wr[axe].Max)
		return lo + (float64(q)+0.5)*(hi-lo)/65536.0
	}
	max := len(o.tete)*8 - 48
	for p := 9; p <= max; p++ {
		br := NewBitReader(o.tete)
		br.SetBitPos(p)
		x := deq(br.ReadBits(16), 0)
		y := deq(br.ReadBits(16), 1)
		z := deq(br.ReadBits(16), 2)
		dx, dy := x-a.x, y-a.y
		if d := math.Sqrt(dx*dx + dy*dy); d <= failleProcheM {
			t.Logf("      sonde position : bit %d -> (%.2f,%.2f,%.2f) à %.2f m de l'ancre", p, x, y, z, d)
		}
	}
}
