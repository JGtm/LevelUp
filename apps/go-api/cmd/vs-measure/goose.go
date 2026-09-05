package main

// goose.go — extensions du driver jetable vs-measure pour le lot GUNGOOSE (2026-09-02).
//
// Le lot Warthog a montre qu'une « permutation d'arme » du render_model du chassis peut n'etre
// qu'un SOCLE, la vraie arme vivant dans un objet-enfant `vehi` identifie par son `weap`. Le
// manifeste sons nomme le weap du Gungoose : `0x0042678e` (banque `veh_un_wargoose`). Il faut
// donc la recherche INVERSE : quel tag reference ce weap ? D'ou :
//
//	-refvers=0xHEX,...   : balaye les tags des groupes -refgroupes et liste ceux dont les octets
//	                       contiennent une des cibles (avec resolution de modele et noms ASCII).
//	-refgroupes=a,b,...  : groupes a balayer (defaut `vehi`) ; `*` = TOUS les groupes indexes.
//	-groupes             : inventaire (groupe -> nombre de tags indexes).
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/himap"
)

// catalogueGroupes : groupes de tags rencontres dans les modules Halo Infinite (liste issue des
// refs observees par refsParGroupe sur les lots vehicules ; sert d'univers au balayage `*`).
var catalogueGroupes = []string{
	"adlg", "aigl", "ant!", "bitm", "cddf", "cfxs", "char", "coll", "cusc", "drdf", "effe",
	"effg", "egfd", "foot", "forg", "fpal", "gegl", "gggl", "gmeg", "gptd", "grfr", "hlmt",
	"jpt!", "lens", "lgtd", "lsnd", "mdsv", "mode", "motl", "mulg", "obje", "pmcg", "proj",
	"rasg", "rtgo", "sadt", "scen", "sbnk", "siin", "snd!", "sngl", "sofd", "styl", "trak",
	"vehi", "weap", "wind", "wpdp", "xong", "bloc", "crea", "eqip", "item", "mach", "term",
	"ctrl", "phmo", "pmdf", "prj3", "scnr", "skel", "anim", "jmad", "mgee", "nclt", "unit",
}

// groupesIndexes rend, parmi le catalogue, les groupes reellement presents dans l'index.
func groupesIndexes(idx *himap.ModuleIndex) []string {
	var out []string
	for _, g := range catalogueGroupes {
		if len(idx.EntreesDuGroupe(g)) > 0 {
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return out
}

// inventaireGroupes imprime le nombre de tags indexes par groupe (tri par groupe).
func inventaireGroupes(idx *himap.ModuleIndex) {
	gs := groupesIndexes(idx)
	total := 0
	var parts []string
	for _, g := range gs {
		n := len(idx.EntreesDuGroupe(g))
		total += n
		parts = append(parts, fmt.Sprintf("%s=%d", g, n))
	}
	fmt.Printf("=== GROUPES (%d groupes du catalogue presents, %d tags) ===\n%s\n",
		len(gs), total, strings.Join(parts, " "))
}

// refsInverses balaye les tags des groupes demandes et imprime ceux dont les octets contiennent
// une des cibles. C'est la recherche « qui reference X » (le weap du Gungoose, par exemple).
func refsInverses(idx *himap.ModuleIndex, cibles []uint32, groupes []string) {
	set := map[uint32]bool{}
	for _, c := range cibles {
		set[c] = true
	}
	if len(groupes) == 1 && groupes[0] == "*" {
		groupes = groupesIndexes(idx)
	}
	fmt.Printf("=== REF-INVERSE vers %s dans les groupes %v ===\n", hexListe(cibles), groupes)
	nTot, nHit := 0, 0
	for _, g := range groupes {
		ids := idx.EntreesDuGroupe(g)
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			nTot++
			tag, err := idx.Extract(id)
			if err != nil {
				continue
			}
			touches := cherche(tag, set, id)
			if len(touches) == 0 {
				continue
			}
			nHit++
			imprimeHit(idx, g, id, tag, touches)
		}
	}
	fmt.Printf("=== REF-INVERSE : %d tags balayes, %d porteurs ===\n", nTot, nHit)
}

// cherche rend les cibles presentes dans les octets du tag (alignement 4, comme refsParGroupe).
func cherche(tag []byte, set map[uint32]bool, self uint32) []uint32 {
	vus := map[uint32]bool{}
	var out []uint32
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(tag[o]) | uint32(tag[o+1])<<8 | uint32(tag[o+2])<<16 | uint32(tag[o+3])<<24
		if h == self || vus[h] || !set[h] {
			continue
		}
		vus[h] = true
		out = append(out, h)
	}
	return out
}

// imprimeHit imprime un porteur de reference : groupe, id, modele resolu, refs, noms ASCII.
func imprimeHit(idx *himap.ModuleIndex, groupe string, id uint32, tag []byte, touches []uint32) {
	etat := "-"
	if mid, mg, ok := himap.RefModeleVehicule(context.Background(), idx, tag); ok {
		_, module, _ := idx.Lookup(mid)
		etat = fmt.Sprintf("%s %#08x@%s", mg, mid, moduleCourt(module))
	}
	noms := chainesASCII(tag, 4)
	fmt.Printf("PORTEUR %s %#08x -> cibles %s ; modele=%s ; refs{%s} ; noms=%s\n",
		groupe, id, hexListe(touches), etat, refsParGroupe(idx, tag, id),
		strings.Join(apercuNoms(nil, noms, 10), " | "))
}

// dumpRefsDe imprime, pour un tag, TOUTES ses refs sortantes vers les groupes demandes (avec
// leurs GlobalID) : c'est le complement de refsParGroupe, qui ne detaille que hlmt/mode/weap/vehi.
// Sert a suivre la chaine d'arme de vehicule `vehi -> sofd -> sofa -> uwfa -> weap` (R-VEHICULE).
func dumpRefsDe(idx *himap.ModuleIndex, id uint32, groupes []string) {
	tag, err := idx.Extract(id)
	if err != nil {
		fmt.Printf("REFS-DE %#08x : extraction KO (%v)\n", id, err)
		return
	}
	voulus := map[string]bool{}
	tous := len(groupes) == 1 && groupes[0] == "*"
	for _, g := range groupes {
		voulus[g] = true
	}
	parGroupe := map[string][]uint32{}
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(tag[o]) | uint32(tag[o+1])<<8 | uint32(tag[o+2])<<16 | uint32(tag[o+3])<<24
		if h == 0 || h == 0xffffffff || h == id || vus[h] {
			continue
		}
		g, _, ok := idx.Lookup(h)
		if !ok || (!tous && !voulus[g]) {
			continue
		}
		vus[h] = true
		parGroupe[g] = append(parGroupe[g], h)
	}
	var gs []string
	for g := range parGroupe {
		gs = append(gs, g)
	}
	sort.Strings(gs)
	fmt.Printf("=== REFS-DE %#08x (taille %d) ===\n", id, len(tag))
	for _, g := range gs {
		fmt.Printf("  %s : %s\n", g, hexListe(parGroupe[g]))
	}
}

// extraitTag ecrit les octets bruts d'un tag dans <out>/<id>.bin (pour un hexdump externe :
// les tags `sofa`/`sofd`/`vcdd` ne sont pas decodes par himap, on lit leurs floats a la main).
func extraitTag(idx *himap.ModuleIndex, id uint32, out string) {
	tag, err := idx.Extract(id)
	if err != nil {
		fmt.Printf("EXTRAIT %#08x : KO (%v)\n", id, err)
		return
	}
	chemin := filepath.Join(out, fmt.Sprintf("%08x.bin", id))
	if err := os.WriteFile(chemin, tag, 0o644); err != nil {
		fmt.Printf("EXTRAIT %#08x : ecriture KO (%v)\n", id, err)
		return
	}
	fmt.Printf("EXTRAIT %#08x -> %s (%d octets)\n", id, chemin, len(tag))
}

// axeHaut traduit le drapeau -axe (z|y|x) en himap.AxeHaut ; toute autre valeur = vue de dessus.
func axeHaut(s string) himap.AxeHaut {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y":
		return himap.HautY
	case "x":
		return himap.HautX
	default:
		return himap.HautZ
	}
}

func hexListe(ids []uint32) string {
	var s []string
	for _, id := range ids {
		s = append(s, fmt.Sprintf("%#08x", id))
	}
	return strings.Join(s, ",")
}
