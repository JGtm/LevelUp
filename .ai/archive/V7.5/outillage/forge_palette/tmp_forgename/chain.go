package main

// chain.go — resolution type_id -> objet de la palette Forge.
//
// Chaine (etablie par cmd/tmp_forgedim, session precedente, et REJOUEE ici comme
// controle positif) :
//
//	food (type_id = global tag id) -> bloc/scen/mach/eqip/weap/vehi/... -> hlmt -> mode
//	-> bloc « compression info » de 84 octets : flags u16 @0x00 puis 3 RealBounds
//	   (min,max) par axe a +0x04.
//
// Ce que cette session ajoute : le GROUPE de tag de l'objet est la seule
// classification disponible hors ligne (les modules ne portent AUCUN nom — table
// des chaines vide sur 44/44 modules de `any`). `eqip` = equipement/power-up,
// `weap` = arme, `vehi` = vehicule.

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type modIndex struct {
	name string
	m    *modReader
	byID map[int32][]modFile
}

var indexes []*modIndex

func openIndexes(root string) {
	var paths []string
	for _, sub := range []string{"any", "ds"} {
		_ = filepath.Walk(filepath.Join(root, sub), func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(p, ".module") {
				paths = append(paths, p)
			}
			return nil
		})
	}
	sort.Strings(paths)
	for _, p := range paths {
		m, err := openMod(p)
		if err != nil {
			continue
		}
		ix := &modIndex{name: shortName(p), m: m, byID: map[int32][]modFile{}}
		for i := 0; i < m.fileCount; i++ {
			f := m.file(i)
			ix.byID[f.GlobalID] = append(ix.byID[f.GlobalID], f)
		}
		indexes = append(indexes, ix)
	}
	fmt.Fprintf(os.Stderr, "modules indexes : %d\n", len(indexes))
}

func shortName(p string) string {
	p = filepath.ToSlash(p)
	parts := strings.Split(p, "/")
	if n := len(parts); n >= 3 {
		return strings.Join(parts[n-3:], "/")
	}
	return p
}

// find rend la copie la PLUS COMPLETE du tag (plus gros uncompSize) : les variantes
// any/ et ds/ d'un meme tag n'ont pas le meme contenu (geometrie strippee d'un cote).
func find(gid int32) (*modIndex, modFile, bool) {
	var bi *modIndex
	var bf modFile
	found := false
	for _, ix := range indexes {
		for _, f := range ix.byID[gid] {
			if f.Group == "" {
				continue
			}
			if !found || f.UncompSize > bf.UncompSize {
				bi, bf, found = ix, f, true
			}
		}
	}
	return bi, bf, found
}

func loadTag(gid int32) (modFile, tagInfo, []byte, bool) {
	ix, f, ok := find(gid)
	if !ok {
		return modFile{}, tagInfo{}, nil, false
	}
	data, err := ix.m.extract(f)
	if err != nil {
		return f, tagInfo{}, nil, false
	}
	ti, ok := parseTag(data)
	return f, ti, data, ok
}

func firstRef(ti tagInfo, groups ...string) (tagRef, bool) {
	for _, g := range groups {
		for _, r := range ti.refs() {
			if r.Group == g {
				return r, true
			}
		}
	}
	return tagRef{}, false
}

type resolved struct {
	typeID     int32
	objGID     int32
	objGroup   string
	modeGID    int32
	dx, dy, dz float64
	hasBox     bool
	note       string
}

var objGroups = []string{"bloc", "scen", "mach", "ctrl", "sinc", "term", "eqip", "weap", "vehi", "proj"}

func resolveType(id int32) resolved {
	res := resolved{typeID: id}
	fFile, fti, fdata, ok := loadTag(id)
	if fdata == nil || !ok {
		res.note = "food introuvable/illisible"
		return res
	}
	if fFile.Group != "food" {
		res.note = "groupe inattendu " + fFile.Group
	}
	objRef, ok := firstRef(fti, objGroups...)
	if !ok {
		if kr, kok := firstRef(fti, "foki"); kok {
			if _, kti, kdata, k2 := loadTag(kr.GlobalID); k2 && kdata != nil {
				if r2, ok2 := firstRef(kti, objGroups...); ok2 {
					objRef, ok = r2, true
					res.note = "via foki"
				}
			}
		}
	}
	if !ok {
		res.note = "aucune ref objet dans le food"
		return res
	}
	res.objGID, res.objGroup = objRef.GlobalID, objRef.Group
	_, oti, odata, ook := loadTag(objRef.GlobalID)
	if !ook || odata == nil {
		res.note = "objet illisible"
		return res
	}
	hRef, ok := firstRef(oti, "hlmt")
	if !ok {
		res.note = "pas de hlmt"
		return res
	}
	_, hti, hdata, hok := loadTag(hRef.GlobalID)
	if !hok || hdata == nil {
		res.note = "hlmt illisible"
		return res
	}
	mRef, ok := firstRef(hti, "mode")
	if !ok {
		res.note = "pas de mode"
		return res
	}
	res.modeGID = mRef.GlobalID
	_, mti, mdata, mok := loadTag(mRef.GlobalID)
	if !mok || mdata == nil {
		res.note = "mode illisible"
		return res
	}
	res.dx, res.dy, res.dz, res.hasBox = compressionDims(mti, mdata)
	if !res.hasBox {
		res.note = "pas de compression info"
	}
	return res
}

// compressionDims lit le(s) bloc(s) de 84 octets du tag `mode` et rend l'union
// des bornes de position. flags == 3 marque un vrai enregistrement.
func compressionDims(ti tagInfo, data []byte) (float64, float64, float64, bool) {
	var mn, mx [3]float32
	ok := false
	for i := 0; i < ti.dataBlocks; i++ {
		abs, sz := ti.blockAbs(i)
		if sz == 0 || sz%84 != 0 || abs+sz > len(data) {
			continue
		}
		for r := 0; r+84 <= sz; r += 84 {
			o := abs + r
			if u16(data, o) != 3 {
				continue
			}
			var lo, hi [3]float32
			bad := false
			for a := 0; a < 3; a++ {
				l, h := f32(data, o+4+a*8), f32(data, o+8+a*8)
				if h < l || h-l > 1e4 || l != l || h != h {
					bad = true
					break
				}
				lo[a], hi[a] = l, h
			}
			if bad {
				continue
			}
			if !ok {
				mn, mx, ok = lo, hi, true
				continue
			}
			for a := 0; a < 3; a++ {
				if lo[a] < mn[a] {
					mn[a] = lo[a]
				}
				if hi[a] > mx[a] {
					mx[a] = hi[a]
				}
			}
		}
	}
	return float64(mx[0] - mn[0]), float64(mx[1] - mn[1]), float64(mx[2] - mn[2]), ok
}

// cmdControl rejoue la chaine sur les type_id de forge_object_types.csv et compare
// groupe et dimensions ligne a ligne. C'est le CONTROLE POSITIF : la nouvelle voie
// doit rendre les 45 types nommes IDENTIQUES.
func cmdControl(csvPath string) {
	f, err := os.Open(csvPath)
	must(err)
	defer f.Close()
	sc := bufio.NewScanner(f)
	nOK, nGroupKO, nDimKO, nTotal := 0, 0, 0, 0
	for i := 0; sc.Scan(); i++ {
		if i == 0 {
			continue
		}
		fs := strings.Split(sc.Text(), ",")
		if len(fs) < 19 {
			continue
		}
		id, err := strconv.ParseInt(strings.TrimSpace(fs[0]), 10, 64)
		if err != nil {
			continue
		}
		nTotal++
		want := struct {
			grp        string
			dx, dy, dz float64
		}{fs[2], atof(fs[15]), atof(fs[16]), atof(fs[17])}
		got := resolveType(int32(id))
		gk := got.objGroup == want.grp
		dk := near(got.dx, want.dx) && near(got.dy, want.dy) && near(got.dz, want.dz)
		switch {
		case gk && dk:
			nOK++
		case !gk:
			nGroupKO++
			fmt.Printf("ECART groupe  type_id=%-12d attendu=%-5s obtenu=%-5s %s\n", id, want.grp, got.objGroup, got.note)
		default:
			nDimKO++
			fmt.Printf("ECART dims    type_id=%-12d attendu=%.4f/%.4f/%.4f obtenu=%.4f/%.4f/%.4f %s\n",
				id, want.dx, want.dy, want.dz, got.dx, got.dy, got.dz, got.note)
		}
	}
	fmt.Printf("\nCONTROLE POSITIF : %d/%d identiques (groupe ET dimensions) ; %d ecarts de groupe, %d de dimensions\n",
		nOK, nTotal, nGroupKO, nDimKO)
}

func near(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 5e-4
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// cmdClassify resout tous les type_id fournis (un par ligne) et rend le groupe de
// tag. Un type_id irresolu reste SANS classement — on n'approche jamais.
func cmdClassify(idFile string) {
	b, err := os.ReadFile(idFile)
	must(err)
	seen := map[int32]bool{}
	byGroup := map[string]int{}
	fmt.Println("type_id,obj_group,obj_gid,dx,dy,dz,note")
	for _, tok := range strings.Fields(string(b)) {
		v, err := strconv.ParseInt(strings.TrimSpace(tok), 10, 64)
		if err != nil || seen[int32(v)] {
			continue
		}
		seen[int32(v)] = true
		r := resolveType(int32(v))
		g := r.objGroup
		if g == "" {
			g = "(irresolu)"
		}
		byGroup[g]++
		fmt.Printf("%d,%s,%d,%.6f,%.6f,%.6f,%s\n", r.typeID, r.objGroup, r.objGID, r.dx, r.dy, r.dz, r.note)
	}
	gs := make([]string, 0, len(byGroup))
	for g := range byGroup {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return byGroup[gs[i]] > byGroup[gs[j]] })
	fmt.Fprintf(os.Stderr, "type_id distincts : %d\n", len(seen))
	for _, g := range gs {
		fmt.Fprintf(os.Stderr, "  %-11s %d\n", g, byGroup[g])
	}
}

// cmdRefsOf imprime TOUTES les references d'un tag `food`, pas seulement la
// premiere. `resolveType` retient la premiere d'une liste de groupes ORDONNEE :
// un food qui reference a la fois un `bloc` (le socle) et un `eqip` (le power-up)
// serait donc classe `bloc`. Cette commande existe pour tester cette faiblesse-la.
func cmdRefsOf(ids []string) {
	fmt.Println("type_id,groupe_food,refs (groupe:gid, toutes)")
	for _, tok := range ids {
		v, err := strconv.ParseInt(strings.TrimSpace(tok), 10, 64)
		if err != nil {
			continue
		}
		f, ti, data, ok := loadTag(int32(v))
		if data == nil || !ok {
			fmt.Printf("%d,(illisible),\n", v)
			continue
		}
		var parts []string
		for _, r := range ti.refs() {
			parts = append(parts, fmt.Sprintf("%s:%d", r.Group, r.GlobalID))
		}
		fmt.Printf("%d,%s,%s\n", v, f.Group, strings.Join(parts, " "))
	}
}

// cmdGroups rend, pour chaque type_id, l'ENSEMBLE des groupes d'objet atteignables
// a profondeur <= 2 (food -> refs, et food -> foki -> refs), sans ordre de priorite.
// C'est la version rigoureuse de `classify` : elle ne peut pas masquer un `eqip`
// derriere un `bloc`.
func cmdGroups(idFile string) {
	b, err := os.ReadFile(idFile)
	must(err)
	inSet := map[string]bool{}
	for _, g := range objGroups {
		inSet[g] = true
	}
	seen := map[int32]bool{}
	tally := map[string]int{}
	fmt.Println("type_id,groupes_atteignables")
	for _, tok := range strings.Fields(string(b)) {
		tok = strings.Split(tok, ",")[0]
		v, err := strconv.ParseInt(strings.TrimSpace(tok), 10, 64)
		if err != nil || seen[int32(v)] {
			continue
		}
		seen[int32(v)] = true
		found := map[string]bool{}
		f, ti, data, ok := loadTag(int32(v))
		if data != nil && ok {
			if inSet[f.Group] {
				found[f.Group] = true
			}
			for _, r := range ti.refs() {
				if inSet[r.Group] {
					found[r.Group] = true
				}
				if r.Group == "foki" {
					if _, kti, kdata, k2 := loadTag(r.GlobalID); k2 && kdata != nil {
						for _, r2 := range kti.refs() {
							if inSet[r2.Group] {
								found[r2.Group] = true
							}
						}
					}
				}
			}
		}
		gs := make([]string, 0, len(found))
		for g := range found {
			gs = append(gs, g)
			tally[g]++
		}
		sort.Strings(gs)
		if len(gs) == 0 {
			tally["(aucun)"]++
		}
		fmt.Printf("%d,%s\n", v, strings.Join(gs, "|"))
	}
	ks := make([]string, 0, len(tally))
	for k := range tally {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return tally[ks[i]] > tally[ks[j]] })
	fmt.Fprintf(os.Stderr, "types examines : %d\n", len(seen))
	for _, k := range ks {
		fmt.Fprintf(os.Stderr, "  %-11s atteignable pour %d types\n", k, tally[k])
	}
}

// cmdRaw ecrit le contenu decompresse d'un tag sur disque, pour un diff octet a octet.
func cmdRaw(gid int32, out string) {
	f, _, data, ok := loadTag(gid)
	if data == nil {
		fmt.Fprintf(os.Stderr, "gid %d introuvable\n", gid)
		return
	}
	must(os.WriteFile(out, data, 0o644))
	fmt.Printf("%d [%s] %d octets -> %s (parse=%v)\n", gid, f.Group, len(data), out, ok)
}

// cmdScanU64 cherche une valeur 64 bits a TOUS les offsets de l'entree fichier, dans
// tous les modules indexes. C'est la methode qui avait prouve Vagabond = fo08_wetland
// (le level_id trouve a +0x28, groupe `levl`, une seule occurrence sur 88 modules).
func cmdScanU64(hexes []string) {
	for _, h := range hexes {
		v, err := strconv.ParseUint(strings.TrimPrefix(h, "0x"), 16, 64)
		if err != nil {
			continue
		}
		fmt.Printf("=== 0x%016x ===\n", v)
		hits := 0
		for _, ix := range indexes {
			for off := 0; off+8 <= modStride; off += 4 {
				for i := 0; i < ix.m.fileCount; i++ {
					e := ix.m.ent[i*modStride:]
					if binary.LittleEndian.Uint64(e[off:]) == v {
						f := ix.m.file(i)
						fmt.Printf("  %s  entree #%d  +0x%02x  groupe=%q gid=%d taille=%d\n",
							ix.name, i, off, f.Group, f.GlobalID, f.UncompSize)
						hits++
					}
				}
			}
		}
		fmt.Printf("  -> %d occurrence(s)\n", hits)
	}
}

// blk1Head imprime les 16 premiers octets du bloc de donnees d'indice n, en passant
// par la table des blocs du tag (et non un offset devine).
func blk1Head(gid int32, n int) {
	_, ti, data, ok := loadTag(gid)
	if !ok || data == nil {
		fmt.Printf("%-13d illisible\n", gid)
		return
	}
	abs, sz := ti.blockAbs(n)
	if sz < 8 {
		fmt.Printf("%-13d bloc %d absent ou trop court\n", gid, n)
		return
	}
	fmt.Printf("%-13d blk%d abs=%-6d taille=%-5d  u64=0x%016x\n",
		gid, n, abs, sz, binary.LittleEndian.Uint64(data[abs:]))
}

// cmdSlots rend, pour chaque type_id d'un fichier, le mot de 32 bits en tete de son
// second bloc de donnees. ETABLI : ce mot est le murmur3 du nom snake_case de l'objet
// (0xACDA2C3C = murmur3("gravity_hammer"), 0x7A3BE607 = murmur3("needler")).
// Le second mot, non nul seulement sur les entrees « emplacement », est rendu aussi.
func cmdSlots(idFile string) {
	b, err := os.ReadFile(idFile)
	must(err)
	fmt.Println("type_id,mot0_hex,mot0_signe,mot1_hex,groupe")
	seen := map[int32]bool{}
	for _, tok := range strings.Fields(string(b)) {
		tok = strings.Split(tok, ",")[0]
		v, err := strconv.ParseInt(strings.TrimSpace(tok), 10, 64)
		if err != nil || seen[int32(v)] {
			continue
		}
		seen[int32(v)] = true
		f, ti, data, ok := loadTag(int32(v))
		if !ok || data == nil {
			continue
		}
		abs, sz := ti.blockAbs(1)
		if sz < 8 || abs+8 > len(data) {
			continue
		}
		w0 := binary.LittleEndian.Uint32(data[abs:])
		w1 := binary.LittleEndian.Uint32(data[abs+4:])
		fmt.Printf("%d,%08x,%d,%08x,%s\n", v, w0, int32(w0), w1, f.Group)
	}
}
