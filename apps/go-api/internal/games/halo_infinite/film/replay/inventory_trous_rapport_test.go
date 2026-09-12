package replay

// inventory_trous_rapport_test.go — L'AGREGATION ET LA MISE EN FORME de la mesure des trous
// d'inventaire. Le diagnostic bit a bit vit dans `inventory_trous_mesure_test.go` ; ici on
// compte, on fusionne et on publie. Aucun bit n'est lu dans ce fichier.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// invTrousCompte agrege les diagnostics d'un film.
type invTrousCompte struct {
	film                  string
	keyframes             int
	keyframesSansBiped    int
	chunksIllisibles      int
	records               int
	slotsAbsents          int
	cat                   map[string]int
	manque                map[string]int
	fichesVides           int
	ammoMulti             int
	genSauveR1            int // R1 echoue ET la lecture generalisee rend UN rang unique
	genAccordI48          int // ... et ce rang est confirme par i48
	genDesaccordI48       int
	r1OKAccordI48         int
	r1OKDesaccordI48      int
	motifAuDelaDe60       int
	ancreMultipleMemeRang int
	// SONDES DE CAUSE — elles ne classent rien, elles departagent les causes.
	sansAncreAvecArme  int // b1 alors que le record porte une arme et des munitions
	sansAncreSansArme  int // b1 sans arme du tout : entite sans equipement a cet instant
	sansAncreMotifSeul int // b1 alors que le motif 20 bits existe ailleurs dans le record
	sansAncrePrefSeul  int // b1 alors que le prefixe 17 bits existe ailleurs
	sansAncreH1        int // b1 alors qu'une ancre a distance de Hamming 1 existe
	r2ZeroPossible     int // R2 tombe alors qu'un i22 de somme NULLE existe apres l'ancre
	videsAvecArme      int // fiche publiee vide alors que le bipede porte une arme
	largeurAvecAncre   int
	nAvecAncre         int
	largeurSansAncre   int
	nSansAncre         int
	h1Rang             map[int]int // histogramme du rang du bit qui differe
	h1MotifUnique      int         // b1 : l'ancre H1 est suivie d'un motif rendant UN rang
	h1AccordI48        int
	h1DesaccordI48     int
	parKeyframe        map[int]int // histogramme du nombre de records de bipede par image-cle
	// LOCALISATION PAR L'ORACLE i48 : parmi toutes les occurrences du prefixe 17 bits d'un
	// record ou R1 tombe, y en a-t-il une dont les 6 bits suivants donnent le rang que i48
	// annonce pour ce slot ? Et ces occurrences tombent-elles a un offset CONSTANT de
	// l'ancre-variante ? Le temoin est le meme test avec un rang decale de 1 : il donne le
	// niveau du hasard.
	prefOracleOK     int
	prefOracleTemoin int
	prefOracleTotal  int
	prefOffRang      map[int]int
}

func invTrousNewCompte(film string) *invTrousCompte {
	return &invTrousCompte{film: film, cat: map[string]int{}, manque: map[string]int{},
		h1Rang: map[int]int{}, parKeyframe: map[int]int{},
		prefOffRang: map[int]int{}}
}

// TestInventaireTrousMesure est le point d'entree. Il ne se lance que si un corpus est designe.

// compter agrege UN diagnostic.
func (c *invTrousCompte) compter(d *invTrousDiag) {
	c.records++
	c.cat[d.categorie]++
	c.manque[d.champsManque]++
	if !d.gren && d.ammoSols == 0 {
		c.fichesVides++
		if d.fam {
			c.videsAvecArme++
		}
	}
	if d.ancres == 0 {
		c.nSansAncre++
		c.largeurSansAncre += d.bits
		if d.fam && d.ammoSols > 0 {
			c.sansAncreAvecArme++
		}
		if !d.fam {
			c.sansAncreSansArme++
		}
		if d.motifSeul > 0 {
			c.sansAncreMotifSeul++
		}
		if d.prefSeul > 0 {
			c.sansAncrePrefSeul++
		}
		if len(d.h1) > 0 {
			c.sansAncreH1++
		}
		for _, h := range d.h1 {
			c.h1Rang[h.rang]++
		}
		if u := invTrousUnique(d.h1Gen); len(u) == 1 {
			c.h1MotifUnique++
			if d.rangI48 >= 0 {
				if u[0] == d.rangI48 {
					c.h1AccordI48++
				} else {
					c.h1DesaccordI48++
				}
			}
		}
	} else {
		c.nAvecAncre++
		c.largeurAvecAncre += d.bits
	}
	if d.hits != 1 && d.rangI48 >= 0 {
		c.prefOracleTotal++
		ok, temoin, off, aOff := invTrousOracle(*d)
		if ok {
			c.prefOracleOK++
		}
		if temoin {
			c.prefOracleTemoin++
		}
		if aOff {
			c.prefOffRang[off]++
		}
	}
	if d.categorie == "d_R2" && d.i22Zero {
		c.r2ZeroPossible++
	}
	if d.ammoSols > 1 {
		c.ammoMulti++
	}
	if d.motifLarge >= 0 && d.hits == 0 {
		c.motifAuDelaDe60++
	}
	uniq := invTrousUnique(d.genRangs)
	if d.hits != 1 && len(uniq) == 1 {
		c.genSauveR1++
		if d.rangI48 >= 0 {
			if uniq[0] == d.rangI48 {
				c.genAccordI48++
			} else {
				c.genDesaccordI48++
			}
		}
	}
	if d.hits == 1 && d.rangI48 >= 0 {
		if d.rangR1 == d.rangI48 {
			c.r1OKAccordI48++
		} else {
			c.r1OKDesaccordI48++
		}
	}
	if d.hits > 1 && len(uniq) == 1 {
		c.ancreMultipleMemeRang++
	}
}

func (c *invTrousCompte) fusion(o *invTrousCompte) {
	c.keyframes += o.keyframes
	c.keyframesSansBiped += o.keyframesSansBiped
	c.chunksIllisibles += o.chunksIllisibles
	c.records += o.records
	c.slotsAbsents += o.slotsAbsents
	c.fichesVides += o.fichesVides
	c.ammoMulti += o.ammoMulti
	c.genSauveR1 += o.genSauveR1
	c.genAccordI48 += o.genAccordI48
	c.genDesaccordI48 += o.genDesaccordI48
	c.r1OKAccordI48 += o.r1OKAccordI48
	c.r1OKDesaccordI48 += o.r1OKDesaccordI48
	c.motifAuDelaDe60 += o.motifAuDelaDe60
	c.ancreMultipleMemeRang += o.ancreMultipleMemeRang
	c.sansAncreAvecArme += o.sansAncreAvecArme
	c.sansAncreSansArme += o.sansAncreSansArme
	c.sansAncreMotifSeul += o.sansAncreMotifSeul
	c.sansAncrePrefSeul += o.sansAncrePrefSeul
	c.sansAncreH1 += o.sansAncreH1
	c.r2ZeroPossible += o.r2ZeroPossible
	c.videsAvecArme += o.videsAvecArme
	c.largeurAvecAncre += o.largeurAvecAncre
	c.nAvecAncre += o.nAvecAncre
	c.largeurSansAncre += o.largeurSansAncre
	c.nSansAncre += o.nSansAncre
	c.h1MotifUnique += o.h1MotifUnique
	c.h1AccordI48 += o.h1AccordI48
	c.h1DesaccordI48 += o.h1DesaccordI48
	for k, v := range o.h1Rang {
		c.h1Rang[k] += v
	}
	for k, v := range o.parKeyframe {
		c.parKeyframe[k] += v
	}
	c.prefOracleOK += o.prefOracleOK
	c.prefOracleTemoin += o.prefOracleTemoin
	c.prefOracleTotal += o.prefOracleTotal
	for k, v := range o.prefOffRang {
		c.prefOffRang[k] += v
	}
	for k, v := range o.cat {
		c.cat[k] += v
	}
	for k, v := range o.manque {
		c.manque[k] += v
	}
}

func (c *invTrousCompte) log(t *testing.T) {
	t.Helper()
	t.Logf("")
	t.Logf("=== %s : %d images-cles (%d sans aucun bipede), %d records de bipede, %d slots absents"+
		" [chunks illisibles : %d]",
		c.film, c.keyframes, c.keyframesSansBiped, c.records, c.slotsAbsents, c.chunksIllisibles)
	for _, k := range invTrousCats {
		if c.cat[k] > 0 || k == "h_complet" {
			t.Logf("    %-18s %6d  %5.1f %%", k, c.cat[k], invTrousPct(c.cat[k], c.records))
		}
	}
	t.Logf("    fiches VIDES (ni grenade ni munition) : %d (%.1f %%)",
		c.fichesVides, invTrousPct(c.fichesVides, c.records))
	t.Logf("    munitions a plusieurs parses : %d (%.1f %%)",
		c.ammoMulti, invTrousPct(c.ammoMulti, c.records))
	t.Logf("    lecture generalisee : R1 tombe et rang unique = %d ; accord i48 = %d ; desaccord = %d",
		c.genSauveR1, c.genAccordI48, c.genDesaccordI48)
	t.Logf("    controle R1 strict vs i48 : accord %d, desaccord %d",
		c.r1OKAccordI48, c.r1OKDesaccordI48)
	t.Logf("    motif strict trouve entre 60 et 400 bits alors que R1 tombe : %d", c.motifAuDelaDe60)
	t.Logf("    ancre multiple mais rang generalise UNIQUE : %d", c.ancreMultipleMemeRang)
	t.Logf("    SANS ANCRE (%d) : avec arme+munitions %d · sans aucune arme %d · "+
		"motif seul present %d · prefixe seul present %d · ancre a Hamming 1 %d",
		c.nSansAncre, c.sansAncreAvecArme, c.sansAncreSansArme,
		c.sansAncreMotifSeul, c.sansAncrePrefSeul, c.sansAncreH1)
	t.Logf("    records de bipede par image-cle : %s", invTrousHisto(c.parKeyframe))
	t.Logf("    ancre H1 : rang du bit qui differe %s", invTrousHisto(c.h1Rang))
	t.Logf("    ancre H1 suivie d'un motif rendant UN rang : %d ; accord i48 %d, desaccord %d",
		c.h1MotifUnique, c.h1AccordI48, c.h1DesaccordI48)
	t.Logf("    oracle i48 sur les records ou R1 tombe : %d/%d portent un prefixe suivi du BON rang "+
		"(temoin rang+1 : %d) ; offsets les plus frequents %s",
		c.prefOracleOK, c.prefOracleTotal, c.prefOracleTemoin, invTrousTop(c.prefOffRang, 6))
	t.Logf("    R2 tombe alors qu'un i22 de somme NULLE existe : %d / %d", c.r2ZeroPossible, c.cat["d_R2"])
	t.Logf("    fiche vide alors que le bipede PORTE une arme : %d", c.videsAvecArme)
	t.Logf("    largeur moyenne de record : avec ancre %.0f bits (n=%d) · sans ancre %.0f bits (n=%d)",
		invTrousMoy(c.largeurAvecAncre, c.nAvecAncre), c.nAvecAncre,
		invTrousMoy(c.largeurSansAncre, c.nSansAncre), c.nSansAncre)
	for _, kv := range invTrousTri(c.manque) {
		t.Logf("    champs manquants %-32s %6d", kv.k, kv.v)
	}
}

func (c *invTrousCompte) tsv(sb *strings.Builder) {
	fmt.Fprintf(sb, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		c.film, c.keyframes, c.keyframesSansBiped, c.records, c.slotsAbsents,
		c.cat["b1_ancreAbsente"], c.cat["b2_motifAbsent"], c.cat["c_ancreMultiple"],
		c.cat["d_R2"], c.cat["e_R3"], c.cat["f_R4"], c.cat["g_partiel"], c.cat["h_complet"],
		c.fichesVides, c.ammoMulti, c.genSauveR1, c.genAccordI48, c.genDesaccordI48,
		c.motifAuDelaDe60)
}

var invTrousCats = []string{
	"b1_ancreAbsente", "b2_motifAbsent", "c_ancreMultiple",
	"d_R2", "e_R3", "f_R4", "g_partiel", "h_complet",
}

type invTrousKV struct {
	k string
	v int
}

func invTrousTri(m map[string]int) []invTrousKV {
	out := make([]invTrousKV, 0, len(m))
	for k, v := range m {
		out = append(out, invTrousKV{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].v > out[j].v })
	return out
}

// invTrousOffsets rend, pour chaque occurrence du motif, sa position RELATIVE a la premiere
// ancre-variante (Hamming 1). Positif = apres l'ancre. C'est ce qui dit si la fenetre de 60
// bits est trop courte, ou si le motif vit ailleurs.
func invTrousOffsets(d invTrousDiag) []int {
	if len(d.h1) == 0 {
		return nil
	}
	out := make([]int, 0, len(d.motifPos))
	for _, p := range d.motifPos {
		out = append(out, p-d.h1[0].fin-1)
	}
	return out
}

// invTrousRel rend des positions absolues ramenees au debut du record.
func invTrousRel(pos []int, from int) []int {
	out := make([]int, 0, len(pos))
	for _, p := range pos {
		out = append(out, p-from)
	}
	return out
}

// invTrousRelH1 rend les positions des ancres-variantes, ramenees au debut du record.
func invTrousRelH1(d invTrousDiag) []int {
	out := make([]int, 0, len(d.h1))
	for _, h := range d.h1 {
		out = append(out, h.fin-27-d.from)
	}
	return out
}

// invTrousTop rend les n entrees les plus frequentes d'un histogramme entier.
func invTrousTop(m map[int]int, n int) string {
	type kv struct{ k, v int }
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	if len(all) > n {
		all = all[:n]
	}
	var sb strings.Builder
	for _, e := range all {
		fmt.Fprintf(&sb, "%+d:%d ", e.k, e.v)
	}
	return sb.String()
}

func invTrousHisto(m map[int]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "b%d=%d ", k, m[k])
	}
	return sb.String()
}

func invTrousMoy(somme, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(somme) / float64(n)
}

func invTrousPct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

// invTrousCreuser publie quelques exemples de la categorie dominante, au bit pres.
func invTrousCreuser(t *testing.T, c *invTrousCompte, diags []invTrousDiag, dig int) {
	t.Helper()
	if dig <= 0 {
		return
	}
	echecs := map[string]int{}
	for _, k := range invTrousCats {
		if k != "h_complet" && c.cat[k] > 0 {
			echecs[k] = c.cat[k]
		}
	}
	tri := invTrousTri(echecs)
	if len(tri) > 2 {
		tri = tri[:2]
	}
	for _, kv := range tri {
		invTrousCreuserCat(t, diags, kv.k, dig)
	}
}

// invTrousCreuserCat publie quelques exemples d'UNE categorie, au bit pres.
func invTrousCreuserCat(t *testing.T, diags []invTrousDiag, dom string, dig int) {
	t.Helper()
	t.Logf("    --- %d exemples de la categorie %s ---", dig, dom)
	n := 0
	for i := range diags {
		if diags[i].categorie != dom || n >= dig {
			continue
		}
		d := diags[i]
		t.Logf("      chunk %2d pkt %4d slot %-5d largeur %6d bits | ancres %d (H1 %d) hits %d rangR1 %3d | "+
			"gen %v | H1gen %v | motif ailleurs %d prefixe %d | motif>60 %4d | i48 %3d | fam %v ammoSols %d degaine %d | i22zero %v",
			d.chunk, d.pkt, d.slot, d.bits, d.ancres, len(d.h1), d.hits, d.rangR1,
			invTrousUnique(d.genRangs), invTrousUnique(d.h1Gen), d.motifSeul, d.prefSeul, d.motifLarge, d.rangI48,
			d.fam, d.ammoSols, d.drawn, d.i22Zero)
		t.Logf("        ancre a +%d du debut de record · motif a %v · ancre-variante a %v",
			d.ancrePos, invTrousRel(d.motifPos, d.from), invTrousRelH1(d))
		t.Logf("        offsets du motif par rapport a l'ancre-variante : %v", invTrousOffsets(d))
		n++
	}
}

// invTrousEcrire depose le TSV si INV_OUT est donne. Rien n'est ecrit par defaut.
func invTrousEcrire(t *testing.T, tsv string) {
	t.Helper()
	out := strings.TrimSpace(os.Getenv(invTrousOutEnv))
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("dossier de sortie %s : %v", out, err)
	}
	p := filepath.Join(out, "inventaire_trous.tsv")
	if err := os.WriteFile(p, []byte(tsv), 0o644); err != nil {
		t.Fatalf("ecriture %s : %v", p, err)
	}
	t.Logf("TSV ecrit : %s", p)
}

// invTrousClasser pose la categorie exclusive et la liste des champs manquants.
func invTrousClasser(d *invTrousDiag) {
	switch {
	case d.ancres == 0:
		d.categorie = "b1_ancreAbsente"
	case d.hits == 0:
		d.categorie = "b2_motifAbsent"
	case d.hits > 1:
		d.categorie = "c_ancreMultiple"
	case !d.gren:
		d.categorie = "d_R2"
	case !d.fam:
		d.categorie = "e_R3"
	case d.ammoSols == 0:
		d.categorie = "f_R4"
	default:
		d.categorie = "h_complet"
	}
	var manque []string
	if d.rangR1 < 0 {
		manque = append(manque, "capacite")
	}
	if !d.gren {
		manque = append(manque, "grenades")
	}
	if d.ammoSols == 0 {
		manque = append(manque, "munitions")
	}
	if d.drawn < 0 {
		manque = append(manque, "degaine")
	}
	if len(manque) == 0 {
		d.champsManque = "-"
		return
	}
	d.champsManque = strings.Join(manque, "+")
	if d.categorie == "h_complet" {
		d.categorie = "g_partiel"
	}
}

// absents a celle-ci. C'est la seule definition de « vivant » qui ne suppose rien.
func invTrousSlotsAbsents(sets []map[uint32]bool) int {
	if len(sets) < 3 {
		return 0
	}
	avant := map[uint32]bool{}
	total := 0
	for i := 1; i < len(sets)-1; i++ {
		for s := range sets[i-1] {
			avant[s] = true
		}
		for s := range avant {
			if sets[i][s] {
				continue
			}
			apres := false
			for j := i + 1; j < len(sets) && !apres; j++ {
				apres = sets[j][s]
			}
			if apres {
				total++
			}
		}
	}
	return total
}
