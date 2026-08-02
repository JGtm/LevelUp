// tmp_forgeshape — jetable, session « palette Forge : nommer et mesurer ».
//
// Question Q2(b) : le record d'objet du .mvar porte-t-il une ECHELLE ou des
// demi-extents ? Le decodeur mapvar ne lit aujourd'hui que les champs 2,3,4,5,7,8,10 ;
// le lecteur Bond, lui, decode l'arbre ENTIER. On inventorie donc tous les champs
// reellement presents, et on dump integralement les objets porteurs d'un role de zone.
//
//	fields  <mvar...>          histogramme (champ, type) sur tous les objets
//	obj     <mvar> <index>     arbre complet d'un objet
//	zones   <mvar>             arbre complet des objets a role d'objectif
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

var typeName = map[byte]string{
	2: "bool", 3: "u8", 4: "u16", 5: "u32", 6: "u64", 7: "float", 8: "double",
	9: "string", 10: "struct", 11: "list", 12: "set", 13: "map",
	14: "i8", 15: "i16", 16: "i32", 17: "i64", 18: "wstring",
}

func tn(t byte) string {
	if s, ok := typeName[t]; ok {
		return s
	}
	return fmt.Sprintf("t%d", t)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: tmp_forgeshape <fields|obj|zones> <mvar> [index]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "fields":
		cmdFields(expand(os.Args[2:]))
	case "obj":
		root, _ := load(os.Args[2])
		idx, err := strconv.Atoi(os.Args[3])
		must(err)
		objs, _ := root.Field(3)
		fmt.Printf("=== objet %d de %s ===\n", idx, os.Args[2])
		printValue(objs.Items[idx], 0)
	case "zones":
		cmdZones(os.Args[2])
	case "shapes":
		cmdShapes(expand(os.Args[2:]))
	case "coverage":
		cmdCoverage(expand(os.Args[2:]))
	case "props":
		cmdProps(expand(os.Args[3:]), os.Args[2])
	case "cratevar":
		cmdCrateVar(expand(os.Args[3:]), os.Args[2])
	case "cratedump":
		cmdCrateDump(os.Args[2], os.Args[3])
	case "crateobj":
		cmdCrateObj(os.Args[2], os.Args[3:])
	case "repname":
		cmdRepName(expand(os.Args[3:]), os.Args[2])
	case "spawners":
		cmdSpawners(expand(os.Args[3:]), os.Args[2])
	case "names":
		cmdNames(expand(os.Args[2:]))
	case "findhash":
		cmdFindHash(expand([]string{os.Args[2]}), os.Args[3:])
	case "upz":
		cmdUpZ(expand([]string{os.Args[2]}), os.Args[3:])
	case "types":
		cmdTypes(expand(os.Args[2:]))
	case "inventory":
		cmdInventory(expand(os.Args[2:]))
	}
}

func expand(args []string) []string {
	var out []string
	for _, a := range args {
		if st, err := os.Stat(a); err == nil && st.IsDir() {
			g, _ := filepath.Glob(filepath.Join(a, "*.mvar"))
			out = append(out, g...)
			continue
		}
		out = append(out, a)
	}
	return out
}

func load(path string) (mapvar.Value, *mapvar.Variant) {
	buf, err := os.ReadFile(path)
	must(err)
	root, err := mapvar.DecodeRoot(buf)
	must(err)
	v, err := mapvar.Parse(buf)
	must(err)
	return root, v
}

// cmdFields compte, sur tous les objets de tous les fichiers, chaque couple
// (identifiant de champ, type Bond). Un champ jamais lu par mapvar.Object et
// present en masse est un candidat « echelle ».
func cmdFields(paths []string) {
	type key struct {
		id  uint16
		typ byte
	}
	count := map[key]int{}
	files := map[key]int{}
	totalObjs, nfile := 0, 0
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			fmt.Printf("SKIP %s : %v\n", filepath.Base(p), err)
			continue
		}
		nfile++
		objs, ok := root.Field(3)
		if !ok {
			continue
		}
		seen := map[key]bool{}
		for _, o := range objs.Items {
			totalObjs++
			for id, v := range o.Fields {
				k := key{id, v.Type}
				count[k]++
				if !seen[k] {
					seen[k] = true
					files[k]++
				}
			}
		}
	}
	fmt.Printf("fichiers lus : %d, objets : %d\n", nfile, totalObjs)
	keys := make([]key, 0, len(count))
	for k := range count {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].id != keys[j].id {
			return keys[i].id < keys[j].id
		}
		return keys[i].typ < keys[j].typ
	})
	lu := map[uint16]string{2: "type_id", 3: "pos", 4: "up", 5: "forward", 7: "flags", 8: "sac", 10: "instance"}
	fmt.Printf("%-6s %-8s %-10s %-8s %s\n", "champ", "type", "occurrences", "fichiers", "lu par mapvar")
	for _, k := range keys {
		fmt.Printf("#%-5d %-8s %-10d %-8d %s\n", k.id, tn(k.typ), count[k], files[k], lu[k.id])
	}
}

// shapeOf lit le sac de forme d'un objet brut : #8 -> #0[0] -> #0[0].
// Rend le type de forme et les 8 emplacements numeriques BRUTS (0 si absent).
// Aucune conversion ici : la division par 65536 est une HYPOTHESE, elle se teste
// en aval sur les valeurs brutes.
func shapeOf(raw mapvar.Value) (int32, [9]int64, bool) {
	var slots [9]int64
	bag, ok := raw.Field(8)
	if !ok {
		return 0, slots, false
	}
	lst, ok := bag.Field(0)
	if !ok || len(lst.Items) == 0 {
		return 0, slots, false
	}
	inner, ok := lst.Items[0].Field(0)
	if !ok || len(inner.Items) == 0 {
		return 0, slots, false
	}
	sh := inner.Items[0]
	kind, ok := sh.Field(0)
	if !ok {
		return 0, slots, false
	}
	for i := uint16(1); i <= 8; i++ {
		if f, ok := sh.Field(i); ok {
			if v, ok2 := f.Field(0); ok2 {
				slots[i] = v.Int
			}
		}
	}
	return int32(kind.Int), slots, true
}

// cmdShapes sort un CSV de toutes les formes trouvees, plus les histogrammes de
// controle : type de forme, et divisibilite des valeurs brutes (test du pas 65536).
func cmdShapes(paths []string) {
	fmt.Println("fichier,obj_index,type_id,role,shape,s1,s2,s3,s4,s5,s6,s7,s8,pos_x,pos_y,pos_z,fwd_x,fwd_y,fwd_z,up_x,up_y,up_z,team,labels")
	byKind := map[int32]int{}
	filesWith := map[string]bool{}
	var raws []int64
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			continue
		}
		roleByIdx := map[int]string{}
		for _, ob := range v.Objectives() {
			roleByIdx[ob.ObjectIdx] = string(ob.Role)
		}
		objs, _ := root.Field(3)
		base := filepath.Base(p)
		for i, item := range objs.Items {
			kind, s, ok := shapeOf(item)
			if !ok {
				continue
			}
			byKind[kind]++
			filesWith[base] = true
			for j := 1; j <= 8; j++ {
				if s[j] != 0 {
					raws = append(raws, s[j])
				}
			}
			o := v.Objects[i]
			var lbl []string
			for _, h := range o.Labels {
				if n := mapvar.LabelName(h); n != "" {
					lbl = append(lbl, n)
				} else {
					lbl = append(lbl, strconv.Itoa(int(h)))
				}
			}
			fmt.Printf("%s,%d,%d,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%.4f,%.4f,%.4f,%.5f,%.5f,%.5f,%.5f,%.5f,%.5f,%d,%s\n",
				base, i, o.TypeID, roleByIdx[i], kind,
				s[1], s[2], s[3], s[4], s[5], s[6], s[7], s[8],
				o.Pos.X, o.Pos.Y, o.Pos.Z, o.Forward.X, o.Forward.Y, o.Forward.Z,
				o.Up.X, o.Up.Y, o.Up.Z, o.TeamIndex, strings.Join(lbl, "|"))
		}
	}
	fmt.Fprintf(os.Stderr, "fichiers porteurs de forme : %d\n", len(filesWith))
	kinds := make([]int, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, int(k))
	}
	sort.Ints(kinds)
	for _, k := range kinds {
		fmt.Fprintf(os.Stderr, "  forme %d : %d objets\n", k, byKind[int32(k)])
	}
	// Controle du pas : combien de valeurs brutes sont des multiples exacts de
	// 65536 (1 m entier), de 1024, etc. Un pas faux ne separera pas.
	for _, step := range []int64{65536, 16384, 4096, 1024, 256} {
		n := 0
		for _, r := range raws {
			if r%step == 0 {
				n++
			}
		}
		fmt.Fprintf(os.Stderr, "  multiples de %-6d : %d / %d (%.1f %%)\n",
			step, n, len(raws), 100*float64(n)/float64(len(raws)))
	}
}

// cmdCoverage mesure, carte par carte et role par role, combien d'objectifs
// portent une forme. Ce qui n'est pas resolu est PUBLIE, pas comble.
func cmdCoverage(paths []string) {
	type stat struct{ total, withShape int }
	byRole := map[string]*stat{}
	fmt.Println("carte,objectifs,avec_forme,sans_forme,roles_sans_forme")
	nmap, totAll, totShape, fullMaps := 0, 0, 0, 0
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			continue
		}
		objs, _ := root.Field(3)
		obs := v.Objectives()
		if len(obs) == 0 {
			continue
		}
		nmap++
		n, ns := 0, 0
		missing := map[string]int{}
		for _, ob := range obs {
			n++
			s := byRole[string(ob.Role)]
			if s == nil {
				s = &stat{}
				byRole[string(ob.Role)] = s
			}
			s.total++
			if _, _, ok := shapeOf(objs.Items[ob.ObjectIdx]); ok {
				ns++
				s.withShape++
			} else {
				missing[string(ob.Role)]++
			}
		}
		totAll += n
		totShape += ns
		if ns == n {
			fullMaps++
		}
		var miss []string
		for r, c := range missing {
			miss = append(miss, fmt.Sprintf("%s:%d", r, c))
		}
		sort.Strings(miss)
		fmt.Printf("%s,%d,%d,%d,%s\n", filepath.Base(p), n, ns, n-ns, strings.Join(miss, "|"))
	}
	fmt.Fprintf(os.Stderr, "cartes avec objectifs : %d (dont %d couvertes a 100 %%)\n", nmap, fullMaps)
	fmt.Fprintf(os.Stderr, "objectifs : %d, avec forme : %d (%.1f %%)\n",
		totAll, totShape, 100*float64(totShape)/float64(totAll))
	roles := make([]string, 0, len(byRole))
	for r := range byRole {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	for _, r := range roles {
		s := byRole[r]
		fmt.Fprintf(os.Stderr, "  %-20s %5d / %5d  (%.1f %%)\n", r, s.withShape, s.total,
			100*float64(s.withShape)/float64(s.total))
	}
}

func cmdZones(path string) {
	root, v := load(path)
	objs, _ := root.Field(3)
	objectives := v.Objectives()
	fmt.Printf("=== %s : %d objets, %d objectifs ===\n", filepath.Base(path), len(v.Objects), len(objectives))
	for _, ob := range objectives {
		fmt.Printf("\n--- role=%s type_id=%d team=%d pos=(%.3f,%.3f,%.3f) idx=%d ---\n",
			ob.Role, ob.TypeID, ob.TeamIndex, ob.Pos.X, ob.Pos.Y, ob.Pos.Z, ob.ObjectIdx)
		printValue(objs.Items[ob.ObjectIdx], 0)
	}
}

func printValue(v mapvar.Value, depth int) {
	pad := strings.Repeat("  ", depth)
	switch v.Type {
	case 10:
		ids := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			f := v.Fields[uint16(id)]
			fmt.Printf("%s#%d %s%s\n", pad, id, tn(f.Type), scalar(f))
			if f.Type == 10 || f.Type == 11 || f.Type == 12 {
				printValue(f, depth+1)
			}
		}
	case 11, 12:
		for i, it := range v.Items {
			if i >= 24 {
				fmt.Printf("%s... (%d items)\n", pad, len(v.Items))
				break
			}
			fmt.Printf("%s[%d] %s%s\n", pad, i, tn(it.Type), scalar(it))
			if it.Type == 10 || it.Type == 11 || it.Type == 12 {
				printValue(it, depth+1)
			}
		}
	}
}

func scalar(v mapvar.Value) string {
	switch v.Type {
	case 2, 3, 4, 5, 6:
		return fmt.Sprintf(" = %d", v.Uint)
	case 14, 15, 16, 17:
		return fmt.Sprintf(" = %d", v.Int)
	case 7, 8:
		return fmt.Sprintf(" = %g", v.Float)
	case 9, 18:
		return fmt.Sprintf(" = %q", v.Str)
	case 11, 12:
		return fmt.Sprintf(" (%d items)", len(v.Items))
	}
	return ""
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
