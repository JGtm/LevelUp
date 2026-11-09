package main

// cratevar.go — PISTE 1 du handoff du 2026-08-02 : le champ « Crate Variant »
// PAR OBJET, au chemin `/#3[]/#8/#1[]/#0[]/#0` (meme sac que le delai de
// reapparition `/#8/#1[]/#4`).
//
// La question, telle que le handoff la pose : trouver une carte ou DEUX
// instances du MEME `type_id` portent des variantes DIFFERENTES. Cela
// prouverait que la variante de caisse CHOISIT l'objet pose sur l'emplacement,
// et donnerait un couple (variante, objet observe) exploitable au Theatre.
//
// Deux commandes :
//
//	cratevar <groupes.csv> <mvar...>   agregation + detection des divergences
//	cratedump <mvar> <type_id>         arbre `#8/#1` du premier objet de ce type

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// crateVariant lit la surcharge de variante de caisse d'un objet.
// Renvoie (valeur, present) — Bond omet les defauts, l'absence est donc la
// regle et non une anomalie.
func crateVariant(raw mapvar.Value) (int32, bool) {
	bag, ok := raw.Field(8)
	if !ok {
		return 0, false
	}
	lst, ok := bag.Field(1)
	if !ok || len(lst.Items) == 0 {
		return 0, false
	}
	inner, ok := lst.Items[0].Field(0)
	if !ok || len(inner.Items) == 0 {
		return 0, false
	}
	v, ok := inner.Items[0].Field(0)
	if !ok {
		return 0, false
	}
	return int32(v.Int), true
}

type varKey struct {
	file   string
	typeID int32
}

func cmdCrateVar(paths []string, groupCSV string) {
	groups := loadGroups(groupCSV)

	// Par (carte, type) : l'ensemble des variantes observees et leur compte.
	perMapType := map[varKey]map[int32]int{}
	// Par type, toutes cartes confondues.
	perType := map[int32]map[int32]int{}
	mapsOfType := map[int32]map[string]bool{}
	totalWith, totalObjs := 0, 0

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
		objs, ok := root.Field(3)
		if !ok {
			continue
		}
		base := filepath.Base(p)
		for i, item := range objs.Items {
			totalObjs++
			cv, has := crateVariant(item)
			if !has {
				continue
			}
			totalWith++
			id := v.Objects[i].TypeID
			k := varKey{base, id}
			if perMapType[k] == nil {
				perMapType[k] = map[int32]int{}
			}
			perMapType[k][cv]++
			if perType[id] == nil {
				perType[id] = map[int32]int{}
				mapsOfType[id] = map[string]bool{}
			}
			perType[id][cv]++
			mapsOfType[id][base] = true
		}
	}

	fmt.Printf("objets balayes : %d ; porteurs d'une variante de caisse : %d\n\n",
		totalObjs, totalWith)

	// LE TEST DE LA PISTE 1 : meme carte, meme type, variantes differentes.
	fmt.Println("=== DIVERGENCES (meme carte, meme type_id, >= 2 variantes) ===")
	keys := make([]varKey, 0, len(perMapType))
	for k, set := range perMapType {
		if len(set) >= 2 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].file != keys[j].file {
			return keys[i].file < keys[j].file
		}
		return keys[i].typeID < keys[j].typeID
	})
	if len(keys) == 0 {
		fmt.Println("(aucune) — sur toutes les cartes balayees, un type_id donne")
		fmt.Println("porte toujours la MEME variante de caisse.")
	}
	for _, k := range keys {
		fmt.Printf("%-40s type=%-13d groupe=%-8s variantes=%s\n",
			k.file, k.typeID, groups[k.typeID], fmtSet(perMapType[k]))
	}

	// Vue par type : combien de variantes distinctes toutes cartes confondues.
	fmt.Println("\n=== PAR TYPE (toutes cartes) ===")
	ids := make([]int32, 0, len(perType))
	for id := range perType {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if len(perType[ids[i]]) != len(perType[ids[j]]) {
			return len(perType[ids[i]]) > len(perType[ids[j]])
		}
		return ids[i] < ids[j]
	})
	fmt.Printf("%-13s %-8s %-7s %-9s %s\n", "type_id", "groupe", "cartes", "variantes", "detail")
	for _, id := range ids {
		fmt.Printf("%-13d %-8s %-7d %-9d %s\n",
			id, groups[id], len(mapsOfType[id]), len(perType[id]), fmtSet(perType[id]))
	}
}

func fmtSet(m map[int32]int) string {
	vs := make([]int32, 0, len(m))
	for v := range m {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(i, j int) bool { return m[vs[i]] > m[vs[j]] })
	out := ""
	for i, v := range vs {
		if i >= 12 {
			out += fmt.Sprintf(" ...(%d au total)", len(vs))
			break
		}
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s(x%d)", named(v), m[v])
	}
	return out
}

// named rend « nom[hash] » quand le hachage est craque, sinon le hachage seul.
func named(h int32) string {
	if n, ok := variantNames[h]; ok {
		return fmt.Sprintf("%s[%d]", n, h)
	}
	return fmt.Sprintf("%d", h)
}

// variantNames — les StringID craques le 2026-08-02 par murmur3_x86_32(seed=0).
//
// `Crate Variant` (`/#8/#1[]/#0[]/#0`) et `Representation Name`
// (`/#8/#24[]/#1/#0`) puisent dans le MEME espace de nommage : les valeurs
// `banished`, `default`, `forge`, `apple`, `base_green`, `base_grime`
// apparaissent dans les deux. La table est donc unique.
//
// GARDE-FOU : une recherche exhaustive produit des collisions fortuites
// (esperance = essais x cibles / 2^32, imprimee a chaque passe). Ne sont
// retenues ici que les correspondances SEMANTIQUEMENT ancrees — soit par le
// domaine Halo, soit par l'usage mesure sur les 199 cartes. Les hits
// combinatoires du type `bombplants_umfgudaemu` sont rejetes, et `cvw`
// (3 lettres, aucune semantique) l'est aussi. Un hachage non liste reste
// SANS NOM : on ne devine pas.
var variantNames = map[int32]string{
	// --- Classes de degat Banished : les 4 entrees « emplacement », ensemble COMPLET ---
	-88833402:   "banished_kinetic",   // entree de palette 1486653438
	-1319554708: "banished_plasma",    // entree de palette 1882451900
	-224029073:  "banished_shock",     // entree de palette -1062552774
	-257275188:  "banished_hardlight", // entree de palette 801517767
	-439440667:  "banished_universal",
	-437051236:  "banished",
	// --- Styles / etats ---
	1120495519:  "default",
	528041935:   "forge",
	-105307848:  "forge_mp",
	1189159012:  "campaign",
	-2143518807: "damaged",
	140497641:   "forerunner",
	1189872317:  "emissive",
	-1105874398: "locked",
	-603392360:  "closed",
	-611385325:  "white",
	330814903:   "dirt",
	947971454:   "mossy",
	1501327410:  "glacier",
	-2031641008: "space",
	600307218:   "base_green",
	2010152917:  "base_yellow",
	-81686274:   "base_grime",
	314637593:   "weapon_only",
	-1138267470: "fo_ai_zone",
	1880549520:  "apple",
	2072964797:  "arrow",
	// --- Objets nommes : vehicules ---
	-266450505: "warthog",
	1063919886: "mongoose",
	-1284820930: "ghost",
	419783896:  "banshee",
	1730553442: "scorpion",
	-1087066335: "wasp",
	1977724336: "phantom",
	// --- Objets nommes : armes, grenades, tourelles ---
	313138863:   "assault_rifle",
	669296699:   "sniper_rifle",
	349319611:   "bandit",
	-976602164:  "commando_rifle",
	-188936615:  "frag_grenade",
	-1834087124: "plasma_grenade",
	1997547608:  "spike_grenade",
	1028518636:  "shade_turret",
	-262278708:  "plasma_turret",
	2072906399:  "unsc_turret",
	307451431:   "equipment",
	1353680800:  "skull",
	1192059526:  "skull_weapon", // deja dans mapvar/objectives.go — atteint par deux chemins
}

// cmdCrateObj detaille, sur une carte, chaque objet des types demandes :
// position, variante de caisse nommee, delai de reapparition, verticalite.
// C'est la table que l'on confronte au terrain.
func cmdCrateObj(path string, wanted []string) {
	want := map[int32]bool{}
	for _, w := range wanted {
		if n, err := strconv.ParseInt(w, 10, 64); err == nil {
			want[int32(n)] = true
		}
	}
	root, v := load(path)
	objs, _ := root.Field(3)
	fmt.Printf("=== %s ===\n", filepath.Base(path))
	fmt.Printf("%-6s %-13s %-9s %-9s %-9s %-20s %-20s %-8s %s\n",
		"idx", "type_id", "x", "y", "z", "variante #8/1", "style #8/24", "respawn", "up.z")
	for i, o := range v.Objects {
		if len(want) > 0 && !want[o.TypeID] {
			continue
		}
		cv, has := crateVariant(objs.Items[i])
		name := "(absente)"
		if has {
			if n, ok := variantNames[cv]; ok {
				name = n
			} else {
				name = fmt.Sprintf("%d", cv)
			}
		}
		respawn := ""
		if bag, ok := objs.Items[i].Field(8); ok {
			if lst, ok := bag.Field(1); ok && len(lst.Items) > 0 {
				if d, ok := lst.Items[0].Field(4); ok {
					respawn = fmt.Sprintf("%ds", d.Uint)
				}
			}
		}
		style := "(absent)"
		if rh, ok := representationName(objs.Items[i]); ok {
			if n, ok2 := variantNames[rh]; ok2 {
				style = n
			} else {
				style = fmt.Sprintf("%d", rh)
			}
		}
		fmt.Printf("%-6d %-13d %-9.2f %-9.2f %-9.2f %-20s %-20s %-8s %.3f\n",
			i, o.TypeID, o.Pos.X, o.Pos.Y, o.Pos.Z, name, style, respawn, o.Up.Z)
	}
}

// representationName lit le StringID de representation PAR OBJET, au chemin
// `/#3[]/#8/#24[]/#1/#0`. Le handoff du 2026-08-02 etablit que ses comptes
// correspondent exactement aux instances par type — c'est donc le meme
// espace de nommage que le `Representation Name` du tag `food`.
func representationName(raw mapvar.Value) (int32, bool) {
	bag, ok := raw.Field(8)
	if !ok {
		return 0, false
	}
	lst, ok := bag.Field(24)
	if !ok || len(lst.Items) == 0 {
		return 0, false
	}
	sub, ok := lst.Items[0].Field(1)
	if !ok {
		return 0, false
	}
	v, ok := sub.Field(0)
	if !ok {
		return 0, false
	}
	return int32(v.Int), true
}

// cmdRepName recense les StringID de representation par objet : leur nombre
// d'instances, le nombre de cartes, et les type_id qui les portent. La sortie
// « --cibles » n'imprime que les hachages, prete pour le craqueur murmur3.
func cmdRepName(paths []string, mode string) {
	count := map[int32]int{}
	maps := map[int32]map[string]bool{}
	types := map[int32]map[int32]bool{}
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
		objs, ok := root.Field(3)
		if !ok {
			continue
		}
		base := filepath.Base(p)
		for i, item := range objs.Items {
			h, has := representationName(item)
			if !has {
				continue
			}
			count[h]++
			if maps[h] == nil {
				maps[h] = map[string]bool{}
				types[h] = map[int32]bool{}
			}
			maps[h][base] = true
			types[h][v.Objects[i].TypeID] = true
		}
	}
	hs := make([]int32, 0, len(count))
	for h := range count {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return count[hs[i]] > count[hs[j]] })
	if mode == "--cibles" {
		for _, h := range hs {
			fmt.Println(h)
		}
		fmt.Fprintf(os.Stderr, "%d StringID de representation distincts\n", len(hs))
		return
	}
	fmt.Println("hash,nom_craque,instances,cartes,type_id_porteurs")
	for _, h := range hs {
		ts := make([]int, 0, len(types[h]))
		for t := range types[h] {
			ts = append(ts, int(t))
		}
		sort.Ints(ts)
		if len(ts) > 6 {
			ts = ts[:6]
		}
		parts := make([]string, 0, len(ts))
		for _, t := range ts {
			parts = append(parts, strconv.Itoa(t))
		}
		fmt.Printf("%d,%s,%d,%d,%s\n",
			h, variantNames[h], count[h], len(maps[h]), strings.Join(parts, " "))
	}
	fmt.Fprintf(os.Stderr, "%d StringID de representation distincts\n", len(hs))
}

// cmdNames rend la table des chaines lisibles (`root[10][1]`) de toutes les
// variantes : les noms que les AUTEURS de cartes ont donnes a leurs objets.
// Vivier de vocabulaire pour le craqueur murmur3, distinct des chaines du
// binaire et de celles de l'interface Forge.
func cmdNames(paths []string) {
	seen := map[string]bool{}
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			continue
		}
		for _, n := range v.Names {
			if n == "" {
				continue
			}
			seen[strings.ToLower(n)] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	for _, n := range out {
		fmt.Println(n)
	}
	fmt.Fprintf(os.Stderr, "%d chaines distinctes\n", len(out))
}

// cmdCrateDump sort l'arbre `#8/#1` du premier objet d'un type donne — la
// verification sur pieces du chemin lu par crateVariant.
func cmdCrateDump(path, typeArg string) {
	id64, err := strconv.ParseInt(typeArg, 10, 64)
	must(err)
	want := int32(id64)
	root, v := load(path)
	objs, _ := root.Field(3)
	for i, o := range v.Objects {
		if o.TypeID != want {
			continue
		}
		fmt.Printf("=== %s objet %d type=%d ===\n", filepath.Base(path), i, want)
		bag, ok := objs.Items[i].Field(8)
		if !ok {
			fmt.Println("(pas de sac #8)")
			return
		}
		sub, ok := bag.Field(1)
		if !ok {
			fmt.Println("(pas de #8/#1)")
			return
		}
		printValue(sub, 1)
		cv, has := crateVariant(objs.Items[i])
		fmt.Printf("crateVariant() -> %d (present=%t)\n", cv, has)
		return
	}
	fmt.Printf("type %d absent de %s\n", want, filepath.Base(path))
}
