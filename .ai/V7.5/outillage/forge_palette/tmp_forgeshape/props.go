package main

// props.go — extraction des champs du record d'objet que `mapvar` lit puis jette.
//
// Candidats retenus par l'inventaire profond (cf. inventory.go) :
//
//	/#6                 float  echelle uniforme (absente quand elle vaut 1)
//	/#8/#0[]/#10        u16    index d'ordre (0..318)
//	/#8/#0[]/#3         bool   drapeau, toujours 1 quand present
//	/#8/#0[]/#11        u8     petit code (1..64, majoritairement 2)
//	/#8/#0[]/#15        bool   drapeau, toujours 1 quand present
//	/#8/#1[]/#4         u16    delai (1..240)  -- reapparition presumee
//	/#8/#1[]/#5         u16    delai (1..120)
//	/#8/#1[]/#13        bool   drapeau, toujours 1 quand present
//
// On ne cable rien : on sort un CSV et des croisements pour DIRE ce que c'est.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

type objProps struct {
	scale                        float64
	order, respawn4, delay5      int
	code11                       int
	b3, b13, b15, hasOrder, has4 bool
}

func readProps(raw mapvar.Value) objProps {
	var p objProps
	p.scale = 1
	if f, ok := raw.Field(6); ok {
		p.scale = f.Float
	}
	bag, ok := raw.Field(8)
	if !ok {
		return p
	}
	if lst, ok := bag.Field(0); ok && len(lst.Items) > 0 {
		gp := lst.Items[0]
		if v, ok := gp.Field(10); ok {
			p.order, p.hasOrder = int(v.Uint), true
		}
		if v, ok := gp.Field(11); ok {
			p.code11 = int(v.Uint)
		}
		if v, ok := gp.Field(3); ok {
			p.b3 = v.Uint != 0
		}
		if v, ok := gp.Field(15); ok {
			p.b15 = v.Uint != 0
		}
	}
	if lst, ok := bag.Field(1); ok && len(lst.Items) > 0 {
		mp := lst.Items[0]
		if v, ok := mp.Field(4); ok {
			p.respawn4, p.has4 = int(v.Uint), true
		}
		if v, ok := mp.Field(5); ok {
			p.delay5 = int(v.Uint)
		}
		if v, ok := mp.Field(13); ok {
			p.b13 = v.Uint != 0
		}
	}
	return p
}

// loadGroups lit forge_object_types.csv : type_id -> groupe de tag (bloc, weap, ...).
func loadGroups(path string) map[int32]string {
	out := map[int32]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 0; sc.Scan(); i++ {
		if i == 0 {
			continue
		}
		fs := strings.Split(sc.Text(), ",")
		if len(fs) < 3 {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(fs[0]), 10, 64)
		if err != nil {
			continue
		}
		out[int32(v)] = fs[2]
	}
	return out
}

func cmdProps(paths []string, groupCSV string) {
	groups := loadGroups(groupCSV)
	fmt.Println("fichier,obj_index,type_id,obj_group,role,scale,ordre,code11,respawn4,delay5,b3,b13,b15")
	// Croisements : pour chaque champ, la repartition par groupe de tag connu.
	byField := map[string]map[string]int{}
	note := func(field, grp string) {
		if byField[field] == nil {
			byField[field] = map[string]int{}
		}
		if grp == "" {
			grp = "(inconnu)"
		}
		byField[field][grp]++
	}
	orderPerFile := map[string][]int{}
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
			pr := readProps(item)
			if pr.scale == 1 && !pr.hasOrder && !pr.has4 && pr.delay5 == 0 &&
				!pr.b3 && !pr.b13 && !pr.b15 && pr.code11 == 0 {
				continue
			}
			o := v.Objects[i]
			g := groups[o.TypeID]
			if pr.scale != 1 {
				note("echelle #6", g)
			}
			if pr.hasOrder {
				note("ordre #8/0/10", g)
				orderPerFile[base] = append(orderPerFile[base], pr.order)
			}
			if pr.has4 {
				note("delai #8/1/4", g)
			}
			if pr.delay5 != 0 {
				note("delai #8/1/5", g)
			}
			if pr.b3 {
				note("bool #8/0/3", g)
			}
			if pr.b13 {
				note("bool #8/1/13", g)
			}
			if pr.b15 {
				note("bool #8/0/15", g)
			}
			fmt.Printf("%s,%d,%d,%s,%s,%.6f,%d,%d,%d,%d,%t,%t,%t\n",
				base, i, o.TypeID, g, roleByIdx[i], pr.scale, pr.order, pr.code11,
				pr.respawn4, pr.delay5, pr.b3, pr.b13, pr.b15)
		}
	}
	fields := make([]string, 0, len(byField))
	for k := range byField {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	fmt.Fprintln(os.Stderr, "--- repartition par groupe de tag (forge_object_types.csv, 45 types nommes) ---")
	for _, f := range fields {
		var parts []string
		gs := make([]string, 0, len(byField[f]))
		for g := range byField[f] {
			gs = append(gs, g)
		}
		sort.Slice(gs, func(i, j int) bool { return byField[f][gs[i]] > byField[f][gs[j]] })
		for _, g := range gs {
			parts = append(parts, fmt.Sprintf("%s:%d", g, byField[f][g]))
		}
		fmt.Fprintf(os.Stderr, "  %-16s %s\n", f, strings.Join(parts, " "))
	}
	// L'index d'ordre est-il une permutation par carte ? (test : doublons)
	fmt.Fprintln(os.Stderr, "--- index d'ordre : unicite par carte (5 cartes les plus fournies) ---")
	type fc struct {
		f string
		n int
	}
	var fl []fc
	for f, v := range orderPerFile {
		fl = append(fl, fc{f, len(v)})
	}
	sort.Slice(fl, func(i, j int) bool { return fl[i].n > fl[j].n })
	for i, e := range fl {
		if i >= 5 {
			break
		}
		seen := map[int]int{}
		for _, o := range orderPerFile[e.f] {
			seen[o]++
		}
		dup := 0
		for _, c := range seen {
			if c > 1 {
				dup++
			}
		}
		fmt.Fprintf(os.Stderr, "  %-44s n=%-5d distincts=%-5d valeurs repetees=%d\n",
			e.f, e.n, len(seen), dup)
	}
}

// cmdTypes rend les type_id distincts des variantes fournies, avec leur nombre
// d'instances — l'entree de la resolution de palette (cmd/tmp_forgename classify).
func cmdTypes(paths []string) {
	count := map[int32]int{}
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			continue
		}
		for _, o := range v.Objects {
			count[o.TypeID]++
		}
	}
	ids := make([]int32, 0, len(count))
	for id := range count {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return count[ids[i]] > count[ids[j]] })
	for _, id := range ids {
		fmt.Printf("%d,%d\n", id, count[id])
	}
	fmt.Fprintf(os.Stderr, "type_id distincts : %d\n", len(ids))
}

// cmdUpZ mesure l'orientation des objets d'un type : la composante verticale du
// vecteur Up. Un objet pose au sol a Up.z ~ 1 ; un RATELIER fixe au mur a Up.z ~ 0.
// C'est le discriminant donne par l'utilisateur : « des rateliers sur certains murs,
// ce sont des emplacements pour des armes de base — ne pas confondre ».
func cmdUpZ(paths []string, wanted []string) {
	want := map[int32]bool{}
	for _, w := range wanted {
		if n, err := strconv.ParseInt(w, 10, 64); err == nil {
			want[int32(n)] = true
		}
	}
	type acc struct{ sol, mur, autre, n int }
	by := map[int32]*acc{}
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			continue
		}
		for _, o := range v.Objects {
			if !want[o.TypeID] {
				continue
			}
			a := by[o.TypeID]
			if a == nil {
				a = &acc{}
				by[o.TypeID] = a
			}
			a.n++
			// Up absent du fichier = (0,0,0) ; Bond omet les defauts, l'orientation
			// canonique verticale s'ecrit donc souvent Up=(0,0,1) ou Up vide.
			z := o.Up.Z
			switch {
			case z > 0.9:
				a.sol++
			case z < 0.35 && z > -0.35:
				a.mur++
			default:
				a.autre++
			}
		}
	}
	ids := make([]int, 0, len(by))
	for id := range by {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	fmt.Printf("%-13s %-7s %-9s %-9s %-9s\n", "type_id", "n", "au sol", "au MUR", "incline")
	for _, id := range ids {
		a := by[int32(id)]
		fmt.Printf("%-13d %-7d %-9s %-9s %-9s\n", id, a.n,
			fmt.Sprintf("%.0f%%", 100*float64(a.sol)/float64(a.n)),
			fmt.Sprintf("%.0f%%", 100*float64(a.mur)/float64(a.n)),
			fmt.Sprintf("%.0f%%", 100*float64(a.autre)/float64(a.n)))
	}
}
