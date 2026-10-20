//go:build research && campagne_overlay

package grammar

// r_comp_l3_kf_research_test.go — CHANTIER « comp », R-L3 : les records d IMAGE-CLE du moteur
// (`ti` qui declarent `managed-engine-timers-component` en `i15`). Une image-cle est un etat
// COMPLET et le record est BORNE (frontiere = premier bit du record suivant) : avec les six
// grammaires du moteur (r_comp_l3_research_test.go), la FERMETURE du record est l oracle de
// justesse que la note 3.7 (§9.5 point 3) reclamait.
//
// Trois mesures par film :
//  1. fermeture des records du moteur sous chaque variante, et classe du masque du bassin ;
//  2. RECHERCHE DE DECALAGE : pour chaque record NON ferme sous `moteur`, on ajoute `d` bits
//     (d dans [-16, 16], d != 0) a l ENTREE du composant `k` (k dans [0, n)) ; un couple (k, d)
//     qui ferme beaucoup de records d un build, et aucun d un autre, designe le composant dont la
//     largeur differe sur ce build. Les couples sont publies avec leur compte de fermetures ;
//  3. le meme balayage sur les records FERMES de HI_1_13_0 sert de temoin de hasard (combien de
//     couples ferment par accident un record deja ferme).
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -run '^TestRCompL3ImagesCles$' ...

import (
	"fmt"
	"sort"
	"testing"
)

// rl3DecalageMax borne la recherche de decalage.
const rl3DecalageMax = 16

// rl3MarcherMoteur marche un record d image-cle du moteur sous une variante, avec un decalage
// `d` a l entree du composant `k` (k < 0 : aucun). Rend l issue et le masque du bassin.
func rl3MarcherMoteur(pay []byte, b keyframeBorne, reg *Registry, ctx ContexteDeLecture,
	v rl3Variante, k, d int) (string, string) {
	arch, _ := reg.Archetype(b.TI)
	base := rl3Crochet(v)
	var classe string
	bis2Intercepteur = func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
		if k >= 0 && k < len(arch.Components) && name == arch.Components[k] && int(typeIndex) == b.TI {
			br.SetBitPos(br.BitPos() + d)
		}
		n0 := len(rl3Masques)
		pris, porte := base(br, name, typeIndex, level)
		if len(rl3Masques) > n0 {
			classe = rl3Masques[len(rl3Masques)-1]
			rl3Masques = rl3Masques[:n0]
		}
		if !pris && k >= 0 && name == arch.Components[k] && int(typeIndex) == b.TI {
			_, _, porte = consumeByName(br, name, typeIndex, level)
			return true, porte
		}
		return pris, porte
	}
	tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
	bis2Intercepteur = nil
	switch {
	case tr.DesyncAt >= 0:
		return "arret " + nomComposantBloquant(reg, b.TI, tr.DesyncAt), classe
	case tr.EndBit == b.Want:
		return "ferme", classe
	case tr.EndBit < b.Want:
		return "sous la frontiere", classe
	}
	return "au-dela de la frontiere", classe
}

// TestRCompL3ImagesCles : fermeture des records du moteur et recherche de decalage.
func TestRCompL3ImagesCles(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tti\tvariante\tissue\tclasse_bassin\trecords"}
	decal := []string{"film\tbuild\tti\tpopulation\tcomposant\tdecalage\tfermes\trecords"}
	defer func() { bis2Intercepteur = nil; rl3Masques = nil }()
	for _, id := range films {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		restore, errMPP := InstallFilmFormatMPP(f.fc)
		if errMPP != nil {
			t.Logf("%s : MPP %v", id, errMPP)
		}
		ctx := f.fc.ContexteDeLecture()
		marche := f.fc.MarcheDImageCle()
		tis := reap37ArchetypesMoteur(f.reg)
		type cle struct {
			ti                    int
			variante, issue, clas string
		}
		comptes := map[cle]int{}
		type cleD struct {
			ti, k, d int
			pop      string
		}
		fermesD := map[cleD]int{}
		pop := map[[2]string]int{}
		for _, num := range f.fc.ChunkNumbers() {
			data, pks, okc := f.fc.ChunkAt(num)
			if !okc {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				pay := pk.Payload(data)
				for _, b := range seulesBornees(keyframeBornesDe(marche.Records(pay))) {
					if !reap37Contient(tis, uint32(b.TI)) { //nolint:gosec // TI < 50
						continue
					}
					var issueMoteur string
					for _, v := range rl3AvecHypotheses() {
						issue, classe := rl3MarcherMoteur(pay, b, f.reg, ctx, v, -1, 0)
						comptes[cle{b.TI, v.nom, issue, classe}]++
						if v.nom == "moteur" {
							issueMoteur = issue
						}
					}
					p := "non ferme sous moteur"
					if issueMoteur == "ferme" {
						p = "ferme sous moteur (temoin de hasard)"
					}
					pop[[2]string{fmt.Sprint(b.TI), p}]++
					arch, _ := f.reg.Archetype(b.TI)
					for k := range arch.Components {
						for d := -rl3DecalageMax; d <= rl3DecalageMax; d++ {
							if d == 0 {
								continue
							}
							if issue, _ := rl3MarcherMoteur(pay, b, f.reg, ctx, rl3Variantes[2], k, d); issue == "ferme" {
								fermesD[cleD{b.TI, k, d, p}]++
							}
						}
					}
				}
			}
		}
		if restore != nil {
			restore()
		}
		for c, n := range comptes {
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%d\t%s\t%s\t%s\t%d", id, f.build, c.ti, c.variante, c.issue, c.clas, n))
		}
		ks := make([]cleD, 0, len(fermesD))
		for c := range fermesD {
			ks = append(ks, c)
		}
		sort.Slice(ks, func(a, b int) bool { return fermesD[ks[a]] > fermesD[ks[b]] })
		for _, c := range ks {
			arch, _ := f.reg.Archetype(c.ti)
			decal = append(decal, fmt.Sprintf("%s\t%s\t%d\t%s\ti%d %s\t%d\t%d\t%d", id, f.build, c.ti, c.pop, c.k,
				arch.Components[c.k], c.d, fermesD[c], pop[[2]string{fmt.Sprint(c.ti), c.pop}]))
		}
		t.Logf("%s %s : %v archetypes moteur, %d couples (composant, decalage) fermants", id, f.build, tis, len(ks))
	}
	b2Ecrire(t, sortie, "r_comp_l3_images_cles.tsv", lignes)
	b2Ecrire(t, sortie, "r_comp_l3_decalages.tsv", decal)
}

// rl3AvecHypotheses : les variantes, plus `moteur` sous chaque hypothese de format.
func rl3AvecHypotheses() []rl3Variante {
	out := append([]rl3Variante(nil), rl3Variantes...)
	for _, h := range rl3Hypotheses {
		out = append(out, rl3Variante{nom: "moteur + " + h, bassin: true, moteur: true, ancien: h})
	}
	return out
}
