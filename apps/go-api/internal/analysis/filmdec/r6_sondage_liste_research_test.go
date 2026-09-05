package filmdec

// r6_sondage_liste_research_test.go — lot R6, question B (volet film) : les types
// d'equipement a ZERO tete sur 325 160 paquets (R5 : 28, 30, 31, 42, 43, 48, 51, 93, 98,
// 104, 115, 116, 119) apparaissent-ils DERRIERE d'autres evenements dans la liste ?
//
// PRINCIPE : la liste d'un paquet delta est [1 config][( 1 R(7) type refs charge )* 0].
// On ne peut MARCHER la liste qu'a travers les types dont la grammaire est FERMEE. Sont
// fermes (sources de l'exe par R6) :
//   - 103 EquipmentSpawnedObject : refs domaines {7,0,7} -> 13 bits, charge 0 bit ;
//   - 100 PowerUpApplied : refs {1,8,7} (sonde en domaine 1 : 9 ou 13 bits), charge
//     [R(1);si 1:R(32)] + R(32) + [R(1);si 1:R(32)] ;
//   - 117 EquipmentTranslocatorTeleportEffects : ref0 domaine 2 (8 bits), charge validee
//     18/18 par TestR6Layout117 — exige la carte du film (R6_MAPS) pour les largeurs ;
//   - les 13 types a charge 0 bit (annexe A : 3, 4, 23, 24, 25, 26, 33, 49, 54, 57, 59,
//     92, 103) : domaines non sources SAUF 103 -> acceptes seulement si les 3 portes de
//     refs valent 0 (sinon la marche s'arrete, comptee "opaque").
//
// La marche s'arrete au premier type non ferme (compte par type : c'est LE resultat).
// Chaque pas est auto-valide par construction : un desalignement rendrait des numeros de
// type uniformes, pas la distribution du census (argument R5 par.4.1).
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 \
//	  R6_ROOT=... R6_CAT=... (memes valeurs que TestR6Layout117) \
//	  R6_MAPS="1b2d9e08=944396dd-5661-4a16-b1d8-a6053f762c55,a0c36016=forest" \
//	  R6_IDS=000d5950,06dfe6d9,084a804d,4f77afc1,8a485699,bf2a9f05,d1dfbc02,1b2d9e08,a0c36016 \
//	  go test ./internal/analysis/filmdec/ -run '^TestR6SondageListe$' -timeout 20m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const r6MapsEnv = "R6_MAPS"

// r6Cibles : les types suspects d'equipement a zero tete (R5 par.2) — l'objet du sondage.
var r6Cibles = map[int]string{
	28: "biped_debug_teleport", 30: "biped_equipment_activation", 31: "equipment_teleport_request",
	42: "biped_dodge", 43: "initiate_mobility_action", 48: "weapon_tether_request",
	51: "biped_throw_release", 93: "activate_spartan_ability", 98: "Equipment",
	104: "EquipmentKnockbackPlayer", 115: "synchronized_teleport", 116: "teleport_effects",
	119: "EquipmentKnockbackRequest",
}

// r6ZeroBit : types a charge 0 bit (annexe A), domaines de refs NON sources (sauf 103).
var r6ZeroBit = map[int]bool{
	3: true, 4: true, 23: true, 24: true, 25: true, 26: true, 33: true,
	49: true, 54: true, 57: true, 59: true, 92: true,
}

// r6Stats agrege le sondage d'un parc.
type r6Stats struct {
	paquetsTeteFermee int         // paquets delta dont la tete est un type marchable
	finPropre         int         // marches terminees sur bit de continuation 0
	opaques           map[int]int // marches arretees sur un type non ferme (compte par type)
	refsNonSourcees   int         // marche arretee : refs d'un type 0-bit avec porte a 1
	horsBuffer        int         // payload copie trop court pour finir la marche
	profondeurMax     int
	parPosition       map[int]map[int]int // position (2, 3, ...) -> type -> compte
}

// TestR6SondageListe marche les listes d'evenements derriere les tetes fermees.
func TestR6SondageListe(t *testing.T) {
	root, ids := os.Getenv(r6RootEnv), os.Getenv(r6IDsEnv)
	catPath, maps := os.Getenv(r6CatEnv), os.Getenv(r6MapsEnv)
	if root == "" || ids == "" || catPath == "" {
		t.Skipf("instrument R6 : definir %s, %s et %s (+%s pour les films a tetes 117)",
			r6RootEnv, r6IDsEnv, r6CatEnv, r6MapsEnv)
	}
	cat := r6LireCatalogue(t, catPath)
	cartes := map[string]string{}
	for _, kv := range strings.Split(maps, ",") {
		if i := strings.IndexByte(kv, '='); i > 0 {
			cartes[strings.TrimSpace(kv[:i])] = strings.TrimSpace(kv[i+1:])
		}
	}
	agg := r6Stats{opaques: map[int]int{}, parPosition: map[int]map[int]int{}}
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		t.Logf("")
		t.Logf("############ FILM %s ############", id)
		var entry *r6CatEntry
		if nom := cartes[id]; nom != "" {
			if e, ok := cat[nom]; ok {
				entry = &e
				t.Logf("== carte %q (bits %v) : les tetes/suites 117 seront marchees ==", nom, e.AxisWidths)
			}
		}
		st := r6SondeFilm(t, filepath.Join(root, id), entry)
		r6RapportStats(t, "FILM", st)
		agg.paquetsTeteFermee += st.paquetsTeteFermee
		agg.finPropre += st.finPropre
		agg.refsNonSourcees += st.refsNonSourcees
		agg.horsBuffer += st.horsBuffer
		if st.profondeurMax > agg.profondeurMax {
			agg.profondeurMax = st.profondeurMax
		}
		for typ, n := range st.opaques {
			agg.opaques[typ] += n
		}
		for pos, m := range st.parPosition {
			if agg.parPosition[pos] == nil {
				agg.parPosition[pos] = map[int]int{}
			}
			for typ, n := range m {
				agg.parPosition[pos][typ] += n
			}
		}
	}
	t.Logf("")
	t.Logf("############ PARC ############")
	r6RapportStats(t, "PARC", agg)
	// Le verdict cible : occurrences des types suspects a n'importe quelle position >= 2.
	t.Logf("== CIBLES (types a zero tete R5) DERRIERE les tetes fermees ==")
	found := false
	for pos, m := range agg.parPosition {
		for typ, n := range m {
			if r6Cibles[typ] != "" {
				t.Logf("  position %d : type %d %s x%d", pos, typ, r6Cibles[typ], n)
				found = true
			}
		}
	}
	for typ, n := range agg.opaques {
		if r6Cibles[typ] != "" {
			t.Logf("  (opaque, position>=2) type %d %s x%d — VU derriere une tete, marche arretee la", typ, r6Cibles[typ], n)
			found = true
		}
	}
	if !found {
		t.Logf("  AUCUNE occurrence d'un type cible derriere les tetes marchees")
	}
}

// r6SondeFilm marche tous les paquets delta d'un film dont la tete est un type ferme.
func r6SondeFilm(t *testing.T, dir string, entry *r6CatEntry) r6Stats {
	t.Helper()
	st := r6Stats{opaques: map[int]int{}, parPosition: map[int]map[int]int{}}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
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
			if pay[0]&0xC0 != 0xC0 {
				continue
			}
			typ := int(pay[0]&0x3F)<<1 | int(pay[1]>>7)
			if !r6Marchable(typ, entry) {
				continue
			}
			st.paquetsTeteFermee++
			r6Marche(pay, typ, entry, &st)
		}
	}
	return st
}

// r6Marchable : la grammaire du type est-elle fermee (refs + charge) ?
func r6Marchable(typ int, entry *r6CatEntry) bool {
	if typ == 103 || typ == 100 || r6ZeroBit[typ] {
		return true
	}
	return typ == 117 && entry != nil
}

// r6Marche parcourt la liste d'evenements d'un paquet a partir de sa tete (deja identifiee).
// Chaque evenement suivant est compte par position ; la marche s'arrete sur un type non
// ferme (opaque), une ref non sourcee, la fin de liste, ou la fin du buffer copie.
func r6Marche(pay []byte, tete int, entry *r6CatEntry, st *r6Stats) {
	br := NewBitReader(pay)
	br.Skip(2) // bit config + bit de continuation de la tete
	typ := tete
	br.Skip(7) // R(7) de la tete
	for pos := 1; ; pos++ {
		if !r6SauteEvenement(br, typ, entry) {
			st.refsNonSourcees++
			return
		}
		if br.Remaining() < 8 {
			st.horsBuffer++
			return
		}
		if pos > st.profondeurMax {
			st.profondeurMax = pos
		}
		if !br.ReadBit() { // bit de continuation : 0 = fin de liste
			st.finPropre++
			return
		}
		typ = int(br.ReadBits(7))
		if st.parPosition[pos+1] == nil {
			st.parPosition[pos+1] = map[int]int{}
		}
		st.parPosition[pos+1][typ]++
		if !r6Marchable(typ, entry) {
			st.opaques[typ]++
			return
		}
	}
}

// r6SauteEvenement consomme les refs et la charge du type courant (grammaires fermees
// seulement). Rend false si une ref d'un type 0-bit non source porte 1 (indecodable).
func r6SauteEvenement(br *BitReader, typ int, entry *r6CatEntry) bool {
	saute13 := func() { br.Skip(13 + 2) } // index 13 bits + generation 2 bits
	switch {
	case typ == 103: // refs {7,0,7} — 13 bits chacune, pas de sonde
		for i := 0; i < 3; i++ {
			if br.ReadBit() {
				saute13()
			}
		}
		return true // charge 0 bit
	case typ == 100: // refs {1,8,7} — ref0 domaine 1 : sonde 9/13
		if br.ReadBit() {
			if br.ReadBit() { // sonde
				br.Skip(9 + 2)
			} else {
				br.Skip(13 + 2)
			}
		}
		for i := 0; i < 2; i++ {
			if br.ReadBit() {
				saute13()
			}
		}
		if br.ReadBit() { // [R(1) ; si 1 : R(32)]
			br.Skip(32)
		}
		br.Skip(32) // variant-name
		if br.ReadBit() {
			br.Skip(32)
		}
		return true
	case typ == 117: // ref0 domaine 2 ; ref1/ref2 non sourcees -> portes 0 exigees
		if br.ReadBit() {
			br.Skip(8 + 2)
		}
		if br.ReadBit() || br.ReadBit() {
			return false
		}
		if br.ReadBit() { // [R(1) ; si 1 : R(32)] — mot d'effet
			br.Skip(32)
		}
		for p := 0; p < 2; p++ { // deux positions, porte INVERSEE (cf. TestR6Layout117)
			if !br.ReadBit() {
				br.Skip(1) // index de region (1 bit mesure : une seule region)
				for i := 0; i < 3; i++ {
					br.Skip(int(entry.AxisWidths[i]))
				}
			} else {
				br.Skip(22 * 3)
			}
		}
		return true
	default: // types 0 bit, domaines non sources : portes 000 exigees
		for i := 0; i < 3; i++ {
			if br.ReadBit() {
				return false
			}
		}
		return true
	}
}

func r6RapportStats(t *testing.T, echelle string, st r6Stats) {
	t.Helper()
	t.Logf("== %s : %d paquets a tete fermee · %d fins propres · %d arrets refs-non-sourcees · %d hors buffer · profondeur max %d ==",
		echelle, st.paquetsTeteFermee, st.finPropre, st.refsNonSourcees, st.horsBuffer, st.profondeurMax)
	positions := make([]int, 0, len(st.parPosition))
	for pos := range st.parPosition {
		positions = append(positions, pos)
	}
	sort.Ints(positions)
	for _, pos := range positions {
		types := make([]int, 0, len(st.parPosition[pos]))
		for typ := range st.parPosition[pos] {
			types = append(types, typ)
		}
		sort.Ints(types)
		var parts []string
		for _, typ := range types {
			parts = append(parts, fmt.Sprintf("%d:%d", typ, st.parPosition[pos][typ]))
		}
		t.Logf("  position %d : {%s}", pos, strings.Join(parts, " "))
	}
	if len(st.opaques) > 0 {
		types := make([]int, 0, len(st.opaques))
		for typ := range st.opaques {
			types = append(types, typ)
		}
		sort.Ints(types)
		var parts []string
		for _, typ := range types {
			parts = append(parts, fmt.Sprintf("%d:%d", typ, st.opaques[typ]))
		}
		t.Logf("  arrets sur type non ferme : {%s}", strings.Join(parts, " "))
	}
}
