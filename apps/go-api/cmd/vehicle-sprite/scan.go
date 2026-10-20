package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/himap"
)

// cmdScan enumere les tags `vehi`, resout leur chaine vers le render_model, et dumpe les
// chaines ASCII lisibles du tag (pour identifier le vehicule sans son nom de fichier, strippe
// dans les modules release).
func cmdScan(args []string) error {
	fs := flag.NewFlagSet("inventaire", flag.ExitOnError)
	mods := fs.String("modules", "", "modules a ouvrir (basenames, virgule)")
	variant := fs.String("variant", "any", "variante deploy: any|pc|ds")
	nStr := fs.Int("strings", 12, "chaines ASCII a montrer par vehi")
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
	ids := idx.EntreesDuGroupe(himap.GroupeVehi)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	fmt.Printf("%d tags vehi indexes (%d entrees au total)\n\n", len(ids), idx.Taille())

	ctx := context.Background()
	resolus := 0
	for _, id := range ids {
		tag, err := idx.Extract(id)
		if err != nil {
			fmt.Printf("vehi %#08x : extraction KO: %v\n", id, err)
			continue
		}
		mid, grp, ok := himap.RefModeleVehicule(ctx, idx, tag)
		etat := "SANS MODELE"
		if ok {
			etat = fmt.Sprintf("%s %#08x", grp, mid)
			resolus++
		}
		noms := chainesASCII(tag, 4)
		fmt.Printf("vehi %#08x  taille=%-8d  modele=%-16s  noms=%s\n",
			id, len(tag), etat, apercu(noms, *nStr))
	}
	fmt.Printf("\n%d/%d vehi resolus jusqu'au modele\n", resolus, len(ids))
	return nil
}

// chainesASCII rend les suites de caracteres imprimables d'au moins min octets, dedupliquees,
// dans l'ordre d'apparition — de quoi reperer un nom interne (chassis, wheel, warthog...).
func chainesASCII(b []byte, min int) []string {
	var out []string
	vus := map[string]bool{}
	cur := make([]byte, 0, 64)
	flush := func() {
		if len(cur) >= min {
			s := string(cur)
			if !vus[s] {
				vus[s] = true
				out = append(out, s)
			}
		}
		cur = cur[:0]
	}
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			cur = append(cur, c)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// apercu rend au plus n chaines, en privilegiant celles qui ressemblent a un nom d'objet
// (lettres, pas un chemin de shader ni un GUID).
func apercu(noms []string, n int) string {
	sort.SliceStable(noms, func(i, j int) bool { return scoreNom(noms[i]) > scoreNom(noms[j]) })
	if len(noms) > n {
		noms = noms[:n]
	}
	return strings.Join(noms, " | ")
}

// scoreNom favorise les chaines courtes en lettres (un nom interne) sur les longues chaines
// techniques (chemins, hash).
func scoreNom(s string) int {
	lettres := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			lettres++
		}
	}
	score := lettres
	if strings.ContainsAny(s, "/\\.:") {
		score -= 10
	}
	if len(s) > 32 {
		score -= 5
	}
	return score
}
