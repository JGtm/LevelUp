package main

// inventory.go — inventaire PROFOND du record d'objet .mvar.
//
// Pourquoi : `mapvar.parseObject` n'extrait que 6 champs du record (#2 #3 #4 #5 #7 #10)
// et `readGameplayBag` que 3 du sac (#1 #8 #9), alors que le lecteur Bond decode
// l'arbre ENTIER sans rien ignorer. Tout le reste est donc lu puis jete. On enumere
// ici ce qui est reellement present, chemin par chemin, avec la distribution des
// valeurs — de quoi reconnaitre une echelle, un delai de reapparition, un ordre
// d'apparition ou un drapeau « present au depart ».
//
// CONSIGNER, PAS CABLER : cette commande ne fait que mesurer.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

type nodeStat struct {
	path       string
	typ        byte
	count      int
	files      map[string]bool
	distinct   map[string]int
	minF, maxF float64
	minI, maxI int64
	numeric    bool
	first      bool
}

type inventory struct {
	nodes map[string]*nodeStat
	order []string
}

func (inv *inventory) note(path string, v mapvar.Value, file string) {
	key := path + "|" + tn(v.Type)
	n := inv.nodes[key]
	if n == nil {
		n = &nodeStat{path: path, typ: v.Type, files: map[string]bool{}, distinct: map[string]int{}, first: true}
		inv.nodes[key] = n
		inv.order = append(inv.order, key)
	}
	n.count++
	n.files[file] = true
	switch v.Type {
	case 2, 3, 4, 5, 6:
		inv.noteInt(n, int64(v.Uint))
	case 14, 15, 16, 17:
		inv.noteInt(n, v.Int)
	case 7, 8:
		n.numeric = true
		if n.first || v.Float < n.minF {
			n.minF = v.Float
		}
		if n.first || v.Float > n.maxF {
			n.maxF = v.Float
		}
		n.first = false
		if len(n.distinct) < 4096 {
			n.distinct[fmt.Sprintf("%g", v.Float)]++
		}
	case 9, 18:
		if len(n.distinct) < 4096 {
			n.distinct[v.Str]++
		}
	case 11, 12:
		if len(n.distinct) < 4096 {
			n.distinct[fmt.Sprintf("len=%d", len(v.Items))]++
		}
	}
}

func (inv *inventory) noteInt(n *nodeStat, x int64) {
	n.numeric = true
	if n.first || x < n.minI {
		n.minI = x
	}
	if n.first || x > n.maxI {
		n.maxI = x
	}
	n.first = false
	if len(n.distinct) < 4096 {
		n.distinct[fmt.Sprintf("%d", x)]++
	}
}

// walk parcourt un sous-arbre. Les listes sont agregees sous « [] » : on veut la
// forme du schema, pas un chemin par indice.
func (inv *inventory) walk(path string, v mapvar.Value, file string, depth int) {
	if depth > 6 {
		return
	}
	switch v.Type {
	case 10:
		ids := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			f := v.Fields[uint16(id)]
			p := fmt.Sprintf("%s/#%d", path, id)
			inv.note(p, f, file)
			inv.walk(p, f, file, depth+1)
		}
	case 11, 12:
		for _, it := range v.Items {
			p := path + "[]"
			inv.note(p, it, file)
			inv.walk(p, it, file, depth+1)
		}
	}
}

func cmdInventory(paths []string) {
	inv := &inventory{nodes: map[string]*nodeStat{}}
	nobj := 0
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			continue
		}
		objs, ok := root.Field(3)
		if !ok {
			continue
		}
		base := filepath.Base(p)
		for _, o := range objs.Items {
			nobj++
			inv.walk("", o, base, 0)
		}
	}
	// Champs deja extraits par le depot : record puis sac gameplay.
	lu := map[string]string{
		"/#2": "type_id (parseObject)", "/#3": "pos", "/#4": "up", "/#5": "forward",
		"/#7": "flags", "/#10": "instance_id",
		"/#8/#0[]/#1": "category (readGameplayBag)", "/#8/#0[]/#8": "team_index",
		"/#8/#0[]/#9": "labels",
	}
	sort.Slice(inv.order, func(i, j int) bool {
		a, b := inv.nodes[inv.order[i]], inv.nodes[inv.order[j]]
		if a.path != b.path {
			return a.path < b.path
		}
		return a.typ < b.typ
	})
	fmt.Printf("objets inventories : %d\n\n", nobj)
	fmt.Printf("%-26s %-8s %-9s %-6s %-7s %-26s %s\n",
		"chemin", "type", "occur.", "fich.", "valeurs", "etendue / echantillon", "extrait par mapvar")
	for _, k := range inv.order {
		n := inv.nodes[k]
		rng := ""
		switch {
		case n.typ == 7 || n.typ == 8:
			rng = fmt.Sprintf("%.4g .. %.4g", n.minF, n.maxF)
		case n.numeric:
			rng = fmt.Sprintf("%d .. %d", n.minI, n.maxI)
		default:
			rng = topValues(n.distinct, 3)
		}
		nd := fmt.Sprintf("%d", len(n.distinct))
		if len(n.distinct) >= 4096 {
			nd = ">4096"
		}
		fmt.Printf("%-26s %-8s %-9d %-6d %-7s %-26s %s\n",
			n.path, tn(n.typ), n.count, len(n.files), nd, rng, lu[n.path])
	}
	fmt.Println("\n--- valeurs les plus frequentes, champs scalaires non extraits ---")
	for _, k := range inv.order {
		n := inv.nodes[k]
		if lu[n.path] != "" || n.typ == 10 || n.typ == 11 || n.typ == 12 || len(n.distinct) == 0 {
			continue
		}
		fmt.Printf("%-26s %-8s %s\n", n.path, tn(n.typ), topValues(n.distinct, 8))
	}
}

func topValues(m map[string]int, k int) string {
	type kv struct {
		v string
		n int
	}
	all := make([]kv, 0, len(m))
	for v, n := range m {
		all = append(all, kv{v, n})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return all[i].v < all[j].v
	})
	if len(all) > k {
		all = all[:k]
	}
	parts := make([]string, 0, len(all))
	for _, e := range all {
		parts = append(parts, fmt.Sprintf("%s(%d)", e.v, e.n))
	}
	return strings.Join(parts, " ")
}

// cmdFindHash cherche des valeurs entieres N'IMPORTE OU dans l'arbre Bond de chaque
// variante de carte, et dit ou elles apparaissent (chemin de champ). Sert a savoir si un
// StringID de la palette est aussi pose sur l'objet dans la carte.
func cmdFindHash(paths []string, wanted []string) {
	want := map[int64]string{}
	for _, w := range wanted {
		if v, err := strconv.ParseInt(strings.TrimSpace(w), 10, 64); err == nil {
			want[v] = w
		}
	}
	hits := map[string]int{}
	files := map[string]map[string]bool{}
	var walk func(path string, v mapvar.Value, file string, depth int)
	walk = func(path string, v mapvar.Value, file string, depth int) {
		if depth > 8 {
			return
		}
		switch v.Type {
		case 14, 15, 16, 17:
			if w, ok := want[v.Int]; ok {
				k := w + " @ " + path
				hits[k]++
				if files[k] == nil {
					files[k] = map[string]bool{}
				}
				files[k][file] = true
			}
		case 2, 3, 4, 5, 6:
			if w, ok := want[int64(v.Uint)]; ok {
				k := w + " @ " + path
				hits[k]++
				if files[k] == nil {
					files[k] = map[string]bool{}
				}
				files[k][file] = true
			}
		case 10:
			ids := make([]int, 0, len(v.Fields))
			for id := range v.Fields {
				ids = append(ids, int(id))
			}
			sort.Ints(ids)
			for _, id := range ids {
				walk(fmt.Sprintf("%s/#%d", path, id), v.Fields[uint16(id)], file, depth+1)
			}
		case 11, 12:
			for _, it := range v.Items {
				walk(path+"[]", it, file, depth+1)
			}
		}
	}
	nobj := 0
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			continue
		}
		base := filepath.Base(p)
		walk("", root, base, 0)
		if objs, ok := root.Field(3); ok {
			nobj += len(objs.Items)
		}
	}
	fmt.Printf("objets balayes : %d, valeurs cherchees : %d\n", nobj, len(want))
	if len(hits) == 0 {
		fmt.Println("AUCUNE occurrence, nulle part dans l'arbre Bond des variantes.")
		return
	}
	ks := make([]string, 0, len(hits))
	for k := range hits {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Printf("  %-40s %d occurrences, %d cartes\n", k, hits[k], len(files[k]))
	}
}
