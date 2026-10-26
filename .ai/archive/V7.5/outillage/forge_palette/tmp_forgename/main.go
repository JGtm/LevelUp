// tmp_forgename — jetable, session « palette Forge : nommer et mesurer ».
//
// Sondes hors ligne sur les modules de la palette Forge :
//
//	hdr     <module>            en-tete + table des chaines + histogramme de groupes
//	ascii   <module> <gid>      extrait un tag et imprime ses suites ASCII
//	entry   <module> <gid>      hexdump de l'entree fichier (0x58 o) du tag
//	list    <module> <groupe>   les gid d'un groupe (ex. fpal, foki, food)
//	dump    <module> <gid>      structure du tag : blocs, refs, racine
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

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: tmp_forgename <hdr|ascii|entry|list|dump|survey> <module|racine> [gid]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "survey":
		cmdSurvey(os.Args[2])
		return
	case "where":
		cmdWhere(os.Args[2], os.Args[3])
		return
	case "control":
		openIndexes(os.Args[2])
		cmdControl(os.Args[3])
		return
	case "crack":
		d := 3
		if len(os.Args) > 4 {
			d, _ = strconv.Atoi(os.Args[4])
		}
		cmdCrack(os.Args[2], os.Args[3], d)
		return
	case "groups":
		openIndexes(os.Args[2])
		cmdGroups(os.Args[3])
		return
	case "slots":
		openIndexes(os.Args[2])
		cmdSlots(os.Args[3])
		return
	case "scanu64":
		openIndexes(os.Args[2])
		cmdScanU64(os.Args[3:])
		return
	case "blk":
		openIndexes(os.Args[2])
		n, _ := strconv.Atoi(os.Args[3])
		for _, g := range os.Args[4:] {
			blk1Head(mustID(g), n)
		}
		return
	case "raw":
		openIndexes(os.Args[2])
		cmdRaw(mustID(os.Args[3]), os.Args[4])
		return
	case "refsof":
		openIndexes(os.Args[2])
		cmdRefsOf(os.Args[3:])
		return
	case "classify":
		openIndexes(os.Args[2])
		cmdClassify(os.Args[3])
		return
	case "hash":
		// Controle de l'hypothese « GlobalID = murmur3 du chemin de tag ».
		for _, s := range os.Args[2:] {
			fmt.Printf("%-60s %d\n", s, mapvar.LabelHash(s))
		}
		return
	}
	m, err := openMod(os.Args[2])
	must(err)
	switch os.Args[1] {
	case "hdr":
		cmdHdr(m)
	case "ascii":
		gid := mustID(os.Args[3])
		f, ok := findGID(m, gid)
		if !ok {
			fmt.Println("gid absent de ce module")
			return
		}
		data, err := m.extract(f)
		must(err)
		fmt.Printf("gid=%d group=%s len=%d\n", gid, f.Group, len(data))
		for _, s := range asciiRuns(data, 4) {
			fmt.Println("  ", s)
		}
	case "list":
		grp := os.Args[3]
		n := 0
		for i := 0; i < m.fileCount; i++ {
			if f := m.file(i); f.Group == grp {
				fmt.Printf("#%-6d gid=%-12d comp=%-9d uncomp=%d\n", f.Index, f.GlobalID, f.CompSize, f.UncompSize)
				n++
			}
		}
		fmt.Printf("total %s : %d\n", grp, n)
	case "dump":
		gid := mustID(os.Args[3])
		f, ok := findGID(m, gid)
		if !ok {
			fmt.Println("gid absent de ce module")
			return
		}
		data, err := m.extract(f)
		must(err)
		ti, ok := parseTag(data)
		fmt.Printf("gid=%d group=%s len=%d parsed=%v\n", gid, f.Group, len(data), ok)
		if !ok {
			return
		}
		fmt.Printf("blocks=%d structs=%d refs=%d dataRefs=%d deps=%d\n",
			ti.dataBlocks, ti.structs, ti.tagRefs, ti.dataRefs, ti.deps)
		for i := 0; i < ti.dataBlocks; i++ {
			abs, sz := ti.blockAbs(i)
			fmt.Printf("  blk%-3d abs=%-9d size=%d\n", i, abs, sz)
		}
		grpCount := map[string]int{}
		for _, r := range ti.refs() {
			grpCount[r.Group]++
		}
		fmt.Printf("  refs par groupe : %v\n", grpCount)
		rb := ti.rootBlock()
		abs, sz := ti.blockAbs(rb)
		fmt.Printf("  root=%d abs=%d size=%d\n", rb, abs, sz)
		hexdump(data, abs, min(sz, 512))
	case "entry":
		gid := mustID(os.Args[3])
		f, ok := findGID(m, gid)
		if !ok {
			fmt.Println("gid absent de ce module")
			return
		}
		e := m.ent[f.Index*modStride : (f.Index+1)*modStride]
		fmt.Printf("entree #%d group=%s\n", f.Index, f.Group)
		hexdump(e, 0, len(e))
		for o := 0; o+4 <= modStride; o += 4 {
			fmt.Printf("  +0x%02x u32=%d i32=%d\n", o, uint32(u32(e, o)), i32(e, o))
		}
	}
}

// cmdSurvey balaie tous les .module d'une racine : taille de la table des chaines
// (une seule non nulle suffirait a donner les noms de tag) et histogramme global
// des groupes, pour reperer un porteur de chaines lisibles.
func cmdSurvey(root string) {
	total := map[string]int{}
	withStrings := 0
	nmod := 0
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".module") {
			return nil
		}
		m, e := openMod(p)
		if e != nil {
			fmt.Printf("SKIP %s : %v\n", p, e)
			return nil
		}
		nmod++
		if len(m.strs) > 0 {
			withStrings++
			fmt.Printf("CHAINES %s : %d octets\n", p, len(m.strs))
		}
		for i := 0; i < m.fileCount; i++ {
			total[m.file(i).Group]++
		}
		_ = m.f.Close()
		return nil
	})
	must(err)
	fmt.Printf("modules: %d, dont avec table de chaines: %d\n", nmod, withStrings)
	keys := make([]string, 0, len(total))
	for k := range total {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return total[keys[i]] > total[keys[j]] })
	for _, k := range keys {
		fmt.Printf("  %-6q %d\n", k, total[k])
	}
}

// cmdWhere localise les tags d'un groupe dans tous les modules d'une racine.
func cmdWhere(root, grp string) {
	must(filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".module") {
			return nil
		}
		m, e := openMod(p)
		if e != nil {
			return nil
		}
		for i := 0; i < m.fileCount; i++ {
			if f := m.file(i); f.Group == grp {
				fmt.Printf("%s  #%d gid=%d uncomp=%d\n", p, f.Index, f.GlobalID, f.UncompSize)
			}
		}
		return m.f.Close()
	}))
}

func cmdHdr(m *modReader) {
	fmt.Printf("fichier      : %s\n", m.path)
	fmt.Printf("fileCount    : %d\n", m.fileCount)
	fmt.Printf("numRes       : %d\n", m.numRes)
	fmt.Printf("numBlocks    : %d\n", m.numBlocks)
	fmt.Printf("stringsSize  : %d  (en-tete +0x24)\n", len(m.strs))
	fmt.Printf("dataBase     : 0x%x\n", m.dataBase)
	fmt.Println("--- en-tete brut ---")
	hexdump(m.hdr, 0, modHdr)
	if len(m.strs) > 0 {
		fmt.Println("--- table des chaines (256 premiers octets) ---")
		n := len(m.strs)
		if n > 256 {
			n = 256
		}
		hexdump(m.strs, 0, n)
		fmt.Println("--- suites ASCII de la table ---")
		for i, s := range asciiRuns(m.strs, 3) {
			if i >= 40 {
				fmt.Printf("   ... (%d suites au total)\n", len(asciiRuns(m.strs, 3)))
				break
			}
			fmt.Println("  ", s)
		}
	}
	groups := map[string]int{}
	for i := 0; i < m.fileCount; i++ {
		groups[m.file(i).Group]++
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return groups[keys[i]] > groups[keys[j]] })
	fmt.Println("--- groupes ---")
	for _, k := range keys {
		fmt.Printf("  %-6q %d\n", k, groups[k])
	}
}

func findGID(m *modReader, gid int32) (modFile, bool) {
	var best modFile
	found := false
	for i := 0; i < m.fileCount; i++ {
		f := m.file(i)
		if f.GlobalID == gid && f.Group != "" && (!found || f.UncompSize > best.UncompSize) {
			best, found = f, true
		}
	}
	return best, found
}

// asciiRuns rend les suites imprimables d'au moins minLen caracteres.
func asciiRuns(b []byte, minLen int) []string {
	var out []string
	start := -1
	for i := 0; i <= len(b); i++ {
		printable := i < len(b) && (b[i] >= 0x20 && b[i] < 0x7f)
		switch {
		case printable && start < 0:
			start = i
		case !printable && start >= 0:
			if i-start >= minLen {
				out = append(out, fmt.Sprintf("@0x%x %q", start, string(b[start:i])))
			}
			start = -1
		}
	}
	return out
}

func hexdump(b []byte, off, n int) {
	if off+n > len(b) {
		n = len(b) - off
	}
	for i := 0; i < n; i += 16 {
		e := i + 16
		if e > n {
			e = n
		}
		line := b[off+i : off+e]
		var sb strings.Builder
		for _, c := range line {
			if c >= 0x20 && c < 0x7f {
				sb.WriteByte(c)
			} else {
				sb.WriteByte('.')
			}
		}
		fmt.Printf("  %04x: % 02x  %s\n", i, line, sb.String())
	}
}

func mustID(s string) int32 {
	v, err := strconv.ParseInt(s, 10, 64)
	must(err)
	return int32(v)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
