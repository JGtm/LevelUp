package mapvar

// CHASSE AUX NOMS DE LIEUX DANS LE .mvar (2026-08-27).
//
// Question posee : le jeu affiche des callouts sur TOUTES les cartes, Forge comprises,
// alors que notre catalogue n'en porte que pour les 22 cartes natives (tag levl du module).
// Hypothese a tester : la variante de carte (.mvar) porterait elle-meme les noms de lieux,
// soit en clair (table root[10][1]), soit sous forme de StringId dans le sac de proprietes
// des objets.
//
// Ce fichier ne conclut pas tout seul : il MESURE (recensement exhaustif des champs) et
// laisse la trace des chiffres dans les logs.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func cheminDump(nom string) string {
	return filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar", nom)
}

func chargeVariante(t *testing.T, nom string) []byte {
	t.Helper()
	brut, err := os.ReadFile(cheminDump(nom))
	if err != nil {
		t.Skipf("variante %s absente : %v", nom, err)
	}
	return brut
}

// ---------------------------------------------------------------------------
// 1. LA TABLE DE CHAINES root[10][1]
// ---------------------------------------------------------------------------

func TestNomsLisiblesDeLaVariante(t *testing.T) {
	for _, nom := range []string{"isolation_map.mvar", "aquarius_map.mvar", "streets_map.mvar", "cliffhanger_map.mvar"} {
		brut, err := os.ReadFile(cheminDump(nom))
		if err != nil {
			t.Logf("%s : absente", nom)
			continue
		}
		v, err := Parse(brut)
		if err != nil {
			t.Errorf("%s : %v", nom, err)
			continue
		}
		t.Logf("=== %s : %d objets, %d chaines dans root[10][1] ===", nom, len(v.Objects), len(v.Names))
		distinct := map[string]int{}
		for _, s := range v.Names {
			distinct[s]++
		}
		t.Logf("  %d chaines distinctes", len(distinct))
		for i, s := range v.Names {
			if i >= 60 {
				t.Logf("  ... (%d de plus)", len(v.Names)-60)
				break
			}
			t.Logf("  [%3d] %q", i, s)
		}
	}
}

// TestArbreRacineVarianteInventaire — que contient la racine, champ par champ ?
func TestArbreRacineVarianteInventaire(t *testing.T) {
	brut := chargeVariante(t, "isolation_map.mvar")
	root, err := DecodeRoot(brut)
	if err != nil {
		t.Fatalf("racine illisible : %v", err)
	}
	ids := make([]int, 0, len(root.Fields))
	for id := range root.Fields {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		f := root.Fields[uint16(id)]
		t.Logf("root[%d] : %s", id, resume(f))
	}
}

func resume(v Value) string {
	switch v.Type {
	case btStruct:
		ids := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		return fmt.Sprintf("struct{%d champs: %v}", len(v.Fields), ids)
	case btList, btSet:
		var kinds string
		if len(v.Items) > 0 {
			kinds = fmt.Sprintf(" elem=%s", nomType(v.Items[0].Type))
		}
		return fmt.Sprintf("liste[%d]%s", len(v.Items), kinds)
	case btMap:
		return fmt.Sprintf("map[%d]", len(v.Pairs))
	case btString, btWString:
		return fmt.Sprintf("chaine %q", v.Str)
	case btFloat, btDouble:
		return fmt.Sprintf("flottant %g", v.Float)
	case btBool, btUint8, btUint16, btUint32, btUint64:
		return fmt.Sprintf("uint %d (0x%X)", v.Uint, v.Uint)
	default:
		return fmt.Sprintf("int %d (0x%X)", v.Int, uint32(v.Int))
	}
}

func nomType(t byte) string {
	switch t {
	case btBool:
		return "bool"
	case btUint8:
		return "u8"
	case btUint16:
		return "u16"
	case btUint32:
		return "u32"
	case btUint64:
		return "u64"
	case btFloat:
		return "f32"
	case btDouble:
		return "f64"
	case btString:
		return "str"
	case btWString:
		return "wstr"
	case btStruct:
		return "struct"
	case btList:
		return "list"
	case btSet:
		return "set"
	case btMap:
		return "map"
	case btInt8:
		return "i8"
	case btInt16:
		return "i16"
	case btInt32:
		return "i32"
	case btInt64:
		return "i64"
	}
	return fmt.Sprintf("t%d", t)
}

// ---------------------------------------------------------------------------
// 2. RECENSEMENT EXHAUSTIF DES CHAMPS D'OBJET
// ---------------------------------------------------------------------------

// stat decrit un chemin de champ rencontre dans l'arbre d'un objet.
type stat struct {
	chemin    string
	typ       byte
	nObjets   int // objets distincts portant ce chemin
	nOcc      int // occurrences totales (listes comprises)
	valeurs   map[int64]int
	flottants []float64
	chaines   map[string]int
	// positions des objets par valeur : pour mesurer l'etalement spatial
	posParVal map[int64][]Vec3
}

const maxValeursSuivies = 4000

type recensement struct {
	stats map[string]*stat
	ordre []string
}

func (r *recensement) get(chemin string, typ byte) *stat {
	cle := chemin + "|" + nomType(typ)
	s, ok := r.stats[cle]
	if !ok {
		s = &stat{chemin: chemin, typ: typ, valeurs: map[int64]int{}, chaines: map[string]int{}, posParVal: map[int64][]Vec3{}}
		r.stats[cle] = s
		r.ordre = append(r.ordre, cle)
	}
	return s
}

// marche parcourt recursivement une valeur et alimente le recensement.
// vus evite de compter deux fois le meme chemin pour un meme objet.
func (r *recensement) marche(chemin string, v Value, pos Vec3, vus map[string]bool) {
	switch v.Type {
	case btStruct:
		ids := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			r.marche(fmt.Sprintf("%s/%d", chemin, id), v.Fields[uint16(id)], pos, vus)
		}
	case btList, btSet:
		s := r.get(chemin+"#len", btInt32)
		r.compte(s, int64(len(v.Items)), pos, vus, chemin+"#len")
		for _, it := range v.Items {
			r.marche(chemin+"[]", it, pos, vus)
		}
	case btMap:
		for _, kv := range v.Pairs {
			r.marche(chemin+"{k}", kv.Key, pos, vus)
			r.marche(chemin+"{v}", kv.Val, pos, vus)
		}
	case btString, btWString:
		s := r.get(chemin, v.Type)
		s.nOcc++
		if !vus[chemin] {
			vus[chemin] = true
			s.nObjets++
		}
		if len(s.chaines) < maxValeursSuivies {
			s.chaines[v.Str]++
		}
	case btFloat, btDouble:
		s := r.get(chemin, v.Type)
		s.nOcc++
		if !vus[chemin] {
			vus[chemin] = true
			s.nObjets++
		}
		if len(s.flottants) < 200000 {
			s.flottants = append(s.flottants, v.Float)
		}
	case btBool, btUint8, btUint16, btUint32, btUint64:
		s := r.get(chemin, v.Type)
		r.compte(s, int64(v.Uint), pos, vus, chemin)
	default:
		s := r.get(chemin, v.Type)
		r.compte(s, v.Int, pos, vus, chemin)
	}
}

func (r *recensement) compte(s *stat, val int64, pos Vec3, vus map[string]bool, chemin string) {
	s.nOcc++
	if !vus[chemin] {
		vus[chemin] = true
		s.nObjets++
	}
	if len(s.valeurs) < maxValeursSuivies {
		s.valeurs[val]++
	}
	if len(s.posParVal) < maxValeursSuivies {
		if len(s.posParVal[val]) < 4000 {
			s.posParVal[val] = append(s.posParVal[val], pos)
		}
	}
}

// champsLus : ce que parseObject/readGameplayBag consomment aujourd'hui.
var champsLus = map[string]bool{
	"/2/0":           true, // type_id
	"/3/0":           true, // pos x
	"/3/1":           true,
	"/3/2":           true,
	"/4/0":           true, // up
	"/4/1":           true,
	"/4/2":           true,
	"/5/0":           true, // forward
	"/5/1":           true,
	"/5/2":           true,
	"/7":             true, // flags
	"/10/0":          true, // instance id
	"/8/0[]/1":       true, // categorie
	"/8/0[]/8":       true, // team index
	"/8/0[]/9[]/0":   true, // labels
	"/8/0[]/0/0[]/0": true, // shape family (cf. shape.go)
	"/8/0[]/0/0[]/5": true,
	"/8/0[]/0/0[]/6": true,
	"/8/0[]/0/0[]/7": true,
	"/8/0[]/0/0[]/8": true,
}

func TestBalayageChampsObjetsIsolation(t *testing.T) {
	balaie(t, "isolation_map.mvar")
}

func TestBalayageChampsObjetsNative(t *testing.T) {
	balaie(t, "aquarius_map.mvar")
}

func balaie(t *testing.T, nom string) {
	brut := chargeVariante(t, nom)
	root, err := DecodeRoot(brut)
	if err != nil {
		t.Fatalf("racine illisible : %v", err)
	}
	objs, ok := root.Field(3)
	if !ok {
		t.Fatal("root[3] absent")
	}
	r := &recensement{stats: map[string]*stat{}}
	for _, o := range objs.Items {
		pos := readVec(o, 3)
		r.marche("", o, pos, map[string]bool{})
	}
	n := len(objs.Items)
	t.Logf("=== %s : %d objets, %d chemins de champ distincts ===", nom, n, len(r.ordre))

	sort.Slice(r.ordre, func(i, j int) bool {
		a, b := r.stats[r.ordre[i]], r.stats[r.ordre[j]]
		if a.nObjets != b.nObjets {
			return a.nObjets > b.nObjets
		}
		return a.chemin < b.chemin
	})
	t.Logf("%-28s %-6s %8s %8s %9s  %s", "CHEMIN", "TYPE", "OBJETS", "OCC", "DISTINCT", "LU? / apercu")
	for _, cle := range r.ordre {
		s := r.stats[cle]
		nd := len(s.valeurs) + len(s.chaines)
		if s.typ == btFloat || s.typ == btDouble {
			nd = nDistinctsFlottants(s.flottants)
		}
		lu := " "
		if champsLus[s.chemin] {
			lu = "LU"
		}
		t.Logf("%-28s %-6s %8d %8d %9d  %-3s %s", s.chemin, nomType(s.typ), s.nObjets, s.nOcc, nd, lu, apercu(s))
	}
}

func nDistinctsFlottants(f []float64) int {
	m := map[float64]bool{}
	for _, x := range f {
		m[x] = true
	}
	return len(m)
}

func apercu(s *stat) string {
	if len(s.chaines) > 0 {
		keys := make([]string, 0, len(s.chaines))
		for k := range s.chaines {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > 6 {
			keys = keys[:6]
		}
		return strings.Join(keys, " | ")
	}
	if len(s.valeurs) > 0 {
		keys := make([]int64, 0, len(s.valeurs))
		for k := range s.valeurs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return s.valeurs[keys[i]] > s.valeurs[keys[j]] })
		var b []string
		for i, k := range keys {
			if i >= 8 {
				break
			}
			b = append(b, fmt.Sprintf("%d x%d", k, s.valeurs[k]))
		}
		return strings.Join(b, ", ")
	}
	if len(s.flottants) > 0 {
		c := append([]float64(nil), s.flottants...)
		sort.Float64s(c)
		return fmt.Sprintf("min=%.3f p50=%.3f max=%.3f", c[0], c[len(c)/2], c[len(c)-1])
	}
	return ""
}

// ---------------------------------------------------------------------------
// 3. CANDIDATS « NOM DE LIEU » : peu de valeurs, etalees dans toute la carte
// ---------------------------------------------------------------------------

func TestCandidatsNomDeLieu(t *testing.T) {
	for _, nom := range []string{"isolation_map.mvar", "aquarius_map.mvar"} {
		brut, err := os.ReadFile(cheminDump(nom))
		if err != nil {
			continue
		}
		root, err := DecodeRoot(brut)
		if err != nil {
			t.Errorf("%s: %v", nom, err)
			continue
		}
		objs, _ := root.Field(3)
		r := &recensement{stats: map[string]*stat{}}
		etendue := etendueCarte(objs.Items)
		for _, o := range objs.Items {
			pos := readVec(o, 3)
			r.marche("", o, pos, map[string]bool{})
		}
		t.Logf("=== %s : %d objets, etendue XY = %.0f x %.0f m ===", nom, len(objs.Items), etendue.X, etendue.Y)
		type cand struct {
			s       *stat
			nd      int
			couvre  float64
			medEtal float64
		}
		var cands []cand
		for _, cle := range r.ordre {
			s := r.stats[cle]
			if s.typ == btFloat || s.typ == btDouble || s.typ == btString || s.typ == btWString {
				continue
			}
			nd := len(s.valeurs)
			if nd < 3 || nd > 200 {
				continue
			}
			// etalement : pour chaque valeur ayant >= 3 objets, rayon moyen autour du centroide
			var etals []float64
			nGroupes := 0
			for v, ps := range s.posParVal {
				_ = v
				if len(ps) < 3 {
					continue
				}
				nGroupes++
				etals = append(etals, rayonMoyen(ps))
			}
			if nGroupes == 0 {
				continue
			}
			sort.Float64s(etals)
			cands = append(cands, cand{s: s, nd: nd,
				couvre:  float64(s.nObjets) / float64(len(objs.Items)),
				medEtal: etals[len(etals)/2]})
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].medEtal < cands[j].medEtal })
		t.Logf("%-28s %-5s %7s %8s %9s  %s", "CHEMIN", "TYPE", "NDIST", "COUVRE", "RAYON_MED", "LU?")
		for _, c := range cands {
			lu := ""
			if champsLus[c.s.chemin] {
				lu = "LU"
			}
			t.Logf("%-28s %-5s %7d %7.0f%% %8.1f m  %s", c.s.chemin, nomType(c.s.typ), c.nd, c.couvre*100, c.medEtal, lu)
		}
	}
}

func etendueCarte(items []Value) Vec3 {
	var min, max Vec3
	first := true
	for _, o := range items {
		p := readVec(o, 3)
		if first {
			min, max, first = p, p, false
			continue
		}
		min.X, max.X = math.Min(min.X, p.X), math.Max(max.X, p.X)
		min.Y, max.Y = math.Min(min.Y, p.Y), math.Max(max.Y, p.Y)
		min.Z, max.Z = math.Min(min.Z, p.Z), math.Max(max.Z, p.Z)
	}
	return Vec3{max.X - min.X, max.Y - min.Y, max.Z - min.Z}
}

func rayonMoyen(ps []Vec3) float64 {
	var cx, cy float64
	for _, p := range ps {
		cx += p.X
		cy += p.Y
	}
	cx /= float64(len(ps))
	cy /= float64(len(ps))
	var s float64
	for _, p := range ps {
		s += math.Hypot(p.X-cx, p.Y-cy)
	}
	return s / float64(len(ps))
}

// ---------------------------------------------------------------------------
// 4. LES AUTRES BRANCHES DE LA RACINE (4, 6, 8, 10, 11, 13)
// ---------------------------------------------------------------------------

func TestInventaireBranchesRacine(t *testing.T) {
	for _, nom := range []string{"isolation_map.mvar", "aquarius_map.mvar"} {
		brut, err := os.ReadFile(cheminDump(nom))
		if err != nil {
			continue
		}
		root, err := DecodeRoot(brut)
		if err != nil {
			t.Errorf("%s: %v", nom, err)
			continue
		}
		t.Logf("################ %s ################", nom)
		for _, id := range []uint16{1, 4, 6, 7, 8, 10, 11, 13} {
			f, ok := root.Field(id)
			if !ok {
				continue
			}
			t.Logf("---- root[%d] : %s ----", id, resume(f))
			dumpProfond(t, fmt.Sprintf("root[%d]", id), f, 0, 4)
		}
	}
}

// dumpProfond imprime une valeur en profondeur, en agregeant les listes.
func dumpProfond(t *testing.T, chemin string, v Value, prof, maxProf int) {
	if prof > maxProf {
		return
	}
	pad := strings.Repeat("  ", prof+1)
	switch v.Type {
	case btStruct:
		ids := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			f := v.Fields[uint16(id)]
			t.Logf("%s%s/%d = %s", pad, chemin, id, resume(f))
			if f.Type == btStruct || f.Type == btList || f.Type == btSet || f.Type == btMap {
				dumpProfond(t, fmt.Sprintf("%s/%d", chemin, id), f, prof+1, maxProf)
			}
		}
	case btList, btSet:
		// agrege : profil des champs sur tous les items
		champs := map[int][]Value{}
		for _, it := range v.Items {
			if it.Type != btStruct {
				continue
			}
			for id, f := range it.Fields {
				champs[int(id)] = append(champs[int(id)], f)
			}
		}
		if len(champs) == 0 {
			// liste de scalaires : echantillon
			var b []string
			for i, it := range v.Items {
				if i >= 12 {
					break
				}
				b = append(b, resume(it))
			}
			t.Logf("%s%s[] scalaires : %s", pad, chemin, strings.Join(b, ", "))
			return
		}
		ids := make([]int, 0, len(champs))
		for id := range champs {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			vals := champs[id]
			t.Logf("%s%s[]/%d : present %d/%d, %s", pad, chemin, id, len(vals), len(v.Items), profilValeurs(vals))
			if vals[0].Type == btStruct || vals[0].Type == btList || vals[0].Type == btMap {
				dumpProfond(t, fmt.Sprintf("%s[]/%d", chemin, id), vals[0], prof+1, maxProf)
			}
		}
	case btMap:
		t.Logf("%s%s : map de %d paires", pad, chemin, len(v.Pairs))
		for i, kv := range v.Pairs {
			if i >= 8 {
				t.Logf("%s  ... (%d de plus)", pad, len(v.Pairs)-8)
				break
			}
			t.Logf("%s  %s -> %s", pad, resume(kv.Key), resume(kv.Val))
		}
		if len(v.Pairs) > 0 && (v.Pairs[0].Val.Type == btStruct || v.Pairs[0].Val.Type == btList) {
			dumpProfond(t, chemin+"{v}", v.Pairs[0].Val, prof+1, maxProf)
		}
	}
}

func profilValeurs(vals []Value) string {
	typ := vals[0].Type
	switch typ {
	case btString, btWString:
		set := map[string]bool{}
		for _, v := range vals {
			set[v.Str] = true
		}
		keys := make([]string, 0, len(set))
		for k := range set {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > 8 {
			keys = keys[:8]
		}
		return fmt.Sprintf("str %d distinctes : %s", len(set), strings.Join(keys, " | "))
	case btFloat, btDouble:
		var f []float64
		for _, v := range vals {
			f = append(f, v.Float)
		}
		sort.Float64s(f)
		return fmt.Sprintf("f %d distincts min=%.3f p50=%.3f max=%.3f", nDistinctsFlottants(f), f[0], f[len(f)/2], f[len(f)-1])
	case btStruct:
		return fmt.Sprintf("struct, ex. %s", resume(vals[0]))
	case btList, btSet:
		n := 0
		for _, v := range vals {
			n += len(v.Items)
		}
		return fmt.Sprintf("liste, total %d items", n)
	default:
		set := map[int64]int{}
		for _, v := range vals {
			k := v.Int
			if typ == btBool || typ == btUint8 || typ == btUint16 || typ == btUint32 || typ == btUint64 {
				k = int64(v.Uint)
			}
			set[k]++
		}
		keys := make([]int64, 0, len(set))
		for k := range set {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return set[keys[i]] > set[keys[j]] })
		var b []string
		for i, k := range keys {
			if i >= 8 {
				break
			}
			b = append(b, fmt.Sprintf("%d x%d", k, set[k]))
		}
		return fmt.Sprintf("%s %d distincts : %s", nomType(typ), len(set), strings.Join(b, ", "))
	}
}
