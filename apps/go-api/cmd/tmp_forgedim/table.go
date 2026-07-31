package main

// Construction de la table type_id -> objet -> dimensions.
//
// Chaîne : food (type_id = global tag id) -> bloc/scen/mach -> hlmt -> mode
// -> bloc "compression info" (0x54 = 84 o) : flags u16 @0x00 puis 3 RealBounds
// (min,max) par axe à +0x04 (X, Y, Z), puis les bornes de texcoord.

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type bbox struct {
	min, max [3]float32
	ok       bool
}

func (b bbox) dims() [3]float32 {
	return [3]float32{b.max[0] - b.min[0], b.max[1] - b.min[1], b.max[2] - b.min[2]}
}

// compressionBounds cherche dans un tag `mode` le(s) bloc(s) de 84 octets et en
// extrait les bornes de position. Renvoie l'union si plusieurs.
func compressionBounds(ti tagInfo, data []byte) bbox {
	var out bbox
	for i := 0; i < ti.dataBlocks; i++ {
		abs, sz := ti.blockAbs(i)
		if sz == 0 || sz%84 != 0 || abs+sz > len(data) {
			continue
		}
		for r := 0; r+84 <= sz; r += 84 {
			o := abs + r
			if u16(data, o) != 3 {
				continue // flags == 3 marque un vrai enregistrement "compression info"
			}
			var mn, mx [3]float32
			bad := false
			for a := 0; a < 3; a++ {
				lo, hi := f32(data, o+4+a*8), f32(data, o+8+a*8)
				if hi < lo || hi-lo > 1e4 || lo != lo || hi != hi {
					bad = true
					break
				}
				mn[a], mx[a] = lo, hi
			}
			if bad {
				continue
			}
			if !out.ok {
				out = bbox{min: mn, max: mx, ok: true}
				continue
			}
			for a := 0; a < 3; a++ {
				if mn[a] < out.min[a] {
					out.min[a] = mn[a]
				}
				if mx[a] > out.max[a] {
					out.max[a] = mx[a]
				}
			}
		}
	}
	return out
}

// NOTE : la piste `phmo` (Havok) a été testée comme repli pour les modèles de
// rendu vides. Le triplet de la diagonale +0x64/+0x74/+0x84 du bloc de 192 octets
// N'EST PAS un demi-extent (valeurs incohérentes : 529 pour un objet de 0,8 ;
// 22 pour un objet de 0,56) — c'est très probablement un tenseur d'inertie.
// Piste abandonnée : aucune dimension n'est inventée pour ces objets.

type resolved struct {
	typeID              int32
	objGID              int32
	objGroup            string
	hlmtGID, modeGID    int32
	collGID, phmoGID    int32
	box                 bbox
	count               int
	modeModule, objMod  string
	note                string
	renderBoxFromCollGT bool
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
	_ = fti
	objGroups := []string{"bloc", "scen", "mach", "ctrl", "sinc", "term", "eqip", "weap", "vehi", "proj"}
	objRef, ok := firstRef(fti, objGroups...)
	if !ok {
		// Certains food pointent une "forge object kind" (foki) qui porte les variantes.
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
	ix, _, _ := find(objRef.GlobalID)
	if ix != nil {
		res.objMod = ix.name
	}
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
	res.hlmtGID = hRef.GlobalID
	_, hti, hdata, hok := loadTag(hRef.GlobalID)
	if !hok || hdata == nil {
		res.note = "hlmt illisible"
		return res
	}
	if r, ok := firstRef(hti, "coll"); ok {
		res.collGID = r.GlobalID
	}
	if r, ok := firstRef(hti, "phmo"); ok {
		res.phmoGID = r.GlobalID
	}
	mRef, ok := firstRef(hti, "mode")
	if !ok {
		res.note = "pas de mode"
		return res
	}
	res.modeGID = mRef.GlobalID
	if ix, _, _ := find(mRef.GlobalID); ix != nil {
		res.modeModule = ix.name
	}
	_, mti, mdata, mok := loadTag(mRef.GlobalID)
	if !mok || mdata == nil {
		res.note = "mode illisible"
		return res
	}
	res.box = compressionBounds(mti, mdata)
	if !res.box.ok {
		res.note = "pas de compression info"
	}
	return res
}

// countsFromMap lit map_objects.csv et compte les instances par type_id.
func countsFromMap(path string) (map[int32]int, []int32) {
	b, err := os.ReadFile(path)
	must(err)
	counts := map[int32]int{}
	var order []int32
	for i, line := range strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, ",")
		if len(f) < 2 {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(f[1]), 10, 64)
		if err != nil {
			continue
		}
		if counts[int32(v)] == 0 {
			order = append(order, int32(v))
		}
		counts[int32(v)]++
	}
	sort.SliceStable(order, func(i, j int) bool { return counts[order[i]] > counts[order[j]] })
	return counts, order
}

func runTable(mapCSV, outCSV string) {
	counts, ids := countsFromMap(mapCSV)
	var rows []resolved
	for _, id := range ids {
		r := resolveType(id)
		r.count = counts[id]
		rows = append(rows, r)
		d := r.box.dims()
		fmt.Printf("%-12d x%-3d %-5s obj=%-11d mode=%-11d bbox %.3f x %.3f x %.3f  [%s] %s\n",
			r.typeID, r.count, r.objGroup, r.objGID, r.modeGID, d[0], d[1], d[2], geomFlag(d), r.note)
	}
	var sb strings.Builder
	sb.WriteString("type_id,instances,obj_group,obj_gid,hlmt_gid,mode_gid,coll_gid,phmo_gid,mode_module," +
		"min_x,min_y,min_z,max_x,max_y,max_z,dx,dy,dz,geom,note\n")
	for _, r := range rows {
		d := r.box.dims()
		sb.WriteString(fmt.Sprintf("%d,%d,%s,%d,%d,%d,%d,%d,%s,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%s,%s\n",
			r.typeID, r.count, r.objGroup, r.objGID, r.hlmtGID, r.modeGID, r.collGID, r.phmoGID, r.modeModule,
			r.box.min[0], r.box.min[1], r.box.min[2], r.box.max[0], r.box.max[1], r.box.max[2],
			d[0], d[1], d[2], geomFlag(d), r.note))
	}
	must(os.WriteFile(outCSV, []byte(sb.String()), 0o644))
	fmt.Printf("écrit %s (%d lignes)\n", outCSV, len(rows))
}

// geomFlag distingue un modèle de rendu réel d'un modèle vide (volume invisible :
// point d'apparition, zone de mort, blocage) dont les bornes valent ±0,0005.
func geomFlag(d [3]float32) string {
	if d[0] < 0.005 && d[1] < 0.005 && d[2] < 0.005 {
		return "modele_vide"
	}
	return "ok"
}
