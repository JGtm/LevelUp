package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// cmdDiag est un outil de reconnaissance : il dumpe, pour un tag donne (vehi, bloc, hlmt,
// mode...), toutes les references de tag inline resolues (offset, id, groupe) et les StringId
// voisins. C'est le levier pour comprendre le bloc « child objects » / « attachments » d'un
// tag d'objet : chaque entree y couple un StringId de marqueur et un tagref d'objet enfant.
func cmdDiag(args []string) error {
	fs := flag.NewFlagSet("diag", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir")
	variant := fs.String("variant", "any", "variante deploy")
	idHex := fs.String("id", "", "GlobalID(s) du tag a diagnostiquer (hex, virgule)")
	group := fs.String("group", "", "ne montrer que les refs de ce groupe (ex: vehi,bloc,hlmt,mode)")
	roots := fs.Bool("roots", false, "dumper les tag-blocks racine (offset, cible, taille, count)")
	hexRoot := fs.Int("hexroot", -1, "hex-dumper le bloc du champ racine a cet offset (mesure de stride)")
	_ = fs.Parse(args)

	chemins, err := cheminsModules(*variant, listeModules(*mods))
	if err != nil {
		return err
	}
	fmt.Printf("ouverture de %d modules...\n", len(chemins))
	idx, err := himap.NewModuleIndex(chemins...)
	if err != nil {
		return err
	}
	wantG := map[string]bool{}
	for _, s := range strings.Split(*group, ",") {
		if s = strings.TrimSpace(s); s != "" {
			wantG[s] = true
		}
	}
	for _, s := range strings.Split(*idHex, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		v, err := strconv.ParseUint(s, 0, 32)
		if err != nil {
			return fmt.Errorf("id %q illisible: %w", s, err)
		}
		id := uint32(v)
		tag, err := idx.Extract(id)
		if err != nil {
			fmt.Printf("extraction %#08x: %v\n", id, err)
			continue
		}
		g, _, _ := idx.Lookup(id)
		fmt.Printf("=== tag %#08x groupe=%s taille=%d ===\n", id, g, len(tag))
		if mid, mg, ok := himap.RefModeleVehicule(context.Background(), idx, tag); ok {
			fmt.Printf("  RefModeleVehicule -> %s %#08x\n", mg, mid)
		} else if g == himap.GroupeVehi {
			fmt.Printf("  RefModeleVehicule -> AUCUN\n")
		}
		if *roots {
			dumpRoots(tag)
		}
		if *hexRoot >= 0 {
			dumpHexRoot(tag, *hexRoot)
		}
		if !*roots && *hexRoot < 0 {
			dumpRefsAvecOffset(idx, tag, id, wantG)
		}
	}
	return nil
}

// dumpRoots imprime les tag-blocks racine d'un tag (offset, cible, taille, count) — pour
// localiser un bloc par son empreinte (ex. le bloc marker groups d'un mode).
func dumpRoots(tag []byte) {
	infos, err := himap.RootBlocksOfTag(tag)
	if err != nil {
		fmt.Printf("  roots: %v\n", err)
		return
	}
	for _, b := range infos {
		fmt.Printf("  root off=%-5d target=%-4d size=%-8d count=%d\n", b.FieldOffset, b.Target, b.BlockSize, b.Count)
	}
}

// dumpHexRoot hex-dumpe le debut du data-block d'un champ racine, avec une lecture flottante,
// pour mesurer le pas d'un enregistrement (les translations de marqueur sortent en floats
// plausibles ~[-3,+3] la ou l'offset est juste).
func dumpHexRoot(tag []byte, off int) {
	b, err := himap.RawRootBlock(tag, off)
	if err != nil {
		fmt.Printf("  hexroot %d: %v\n", off, err)
		return
	}
	n := len(b)
	if n > 512 {
		n = 512
	}
	fmt.Printf("  bloc off=%d taille=%d, %d premiers octets:\n", off, len(b), n)
	for o := 0; o+4 <= n; o += 4 {
		u := u32le(b, o)
		f := f32le(b, o)
		fmt.Printf("    +%-4d  u32=%-11d  hex=%08x  f32=%+.4f\n", o, u, u, f)
	}
}

func f32le(b []byte, o int) float64 {
	return float64(math.Float32frombits(u32le(b, o)))
}

// refAOffset : une reference de tag resolue, avec son offset dans le tag.
type refAOffset struct {
	off int
	id  uint32
	grp string
}

// dumpRefsAvecOffset scanne le tag par pas de 4 et imprime toute valeur qui resout comme
// GlobalID d'un tag connu, avec son offset et son groupe. Le voisinage (StringId a +/-4) est
// imprime pour reperer un couple {marqueur, enfant}.
func dumpRefsAvecOffset(idx *himap.ModuleIndex, tag []byte, self uint32, wantG map[string]bool) {
	var refs []refAOffset
	for o := 0; o+4 <= len(tag); o += 4 {
		h := u32le(tag, o)
		if h == 0 || h == 0xffffffff {
			continue
		}
		g, _, ok := idx.Lookup(h)
		if !ok {
			continue
		}
		if len(wantG) > 0 && !wantG[g] {
			continue
		}
		refs = append(refs, refAOffset{off: o, id: h, grp: g})
	}
	// Comptage par groupe.
	parGroupe := map[string]int{}
	for _, r := range refs {
		parGroupe[r.grp]++
	}
	var groupes []string
	for g := range parGroupe {
		groupes = append(groupes, g)
	}
	sort.Strings(groupes)
	fmt.Printf("refs par groupe: ")
	for _, g := range groupes {
		fmt.Printf("%s=%d ", g, parGroupe[g])
	}
	fmt.Println()
	for _, r := range refs {
		marque := ""
		if r.id == self {
			marque = " (SELF)"
		}
		var prev, next uint32
		if r.off-4 >= 0 {
			prev = u32le(tag, r.off-4)
		}
		if r.off+8 <= len(tag) {
			next = u32le(tag, r.off+4)
		}
		fmt.Printf("  off=%6d (0x%05x) %s %#08x  prev=%#010x next=%#010x%s\n",
			r.off, r.off, r.grp, r.id, prev, next, marque)
	}
}

func u32le(b []byte, o int) uint32 {
	return uint32(b[o]) | uint32(b[o+1])<<8 | uint32(b[o+2])<<16 | uint32(b[o+3])<<24
}
