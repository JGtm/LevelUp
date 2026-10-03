//go:build research && campagne_overlay

package grammar

// r_veh_ti40_variantes_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L4 (a) (2026-10-02),
// compagnon de `r_veh_ti40_research_test.go` : l A/B des manieres de lire un record `ti=40`
// d image-cle, sur les films de CAMPAGNE_FILMS. Mesure seulement.
//
// Pour chaque variante ([rvVariantes]) et chaque film : records bornes, records fermes, records a
// voisin consecutif (`slot suivant = slot + 1`, la seule frontiere qui ne peut pas sauter un
// record) et fermes parmi eux, records dont n2 lit la taille de l etat vehicule (0x8d8, l oracle
// de l etat par defaut), et records « sous » dont la fin tombe sur un en-tete de record valide
// (un record saute par le balayeur d ancres, pas une largeur fausse). Par classe de chassis.
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestRVehTi40Variantes$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rvVariantes : de la production a la lecture complete.
func rvVariantes() []rvVariante {
	return []rvVariante{
		{nom: "production", porte: "chassis"},
		{nom: "bis2-composants+porte-chassis", crochet: true, porte: "chassis"},
		{nom: "portee-etat", crochet: true, portee: 1, porte: "chassis"},
		{nom: "portee-record", crochet: true, portee: 2, porte: "chassis"},
		{nom: "i0-ecrivain", crochet: true, i0: true, porte: "chassis"},
		{nom: "portee-etat+i0-ecrivain", crochet: true, portee: 1, i0: true, porte: "chassis"},
		{nom: "portee-record+i0-ecrivain", crochet: true, portee: 2, i0: true, porte: "chassis"},
		{nom: "portee-record+i0-ecrivain+porte-posee", crochet: true, portee: 2, i0: true, porte: "posee"},
		{nom: "portee-record+i0-ecrivain+porte-levee", crochet: true, portee: 2, i0: true, porte: "levee"},
		{nom: "portee-record+i0-ecrivain-sans-composants", portee: 2, i0: true, porte: "chassis"},
		{nom: "portee-record+i0-ecrivain+etat-sans-liste", crochet: true, portee: 2, i0: true, porte: "chassis", etatSansListe: true},
		{nom: "portee-record+i0-ecrivain+etat-sans-liste+porte-posee", crochet: true, portee: 2, i0: true, porte: "posee", etatSansListe: true},
		{nom: "etat-sans-liste+portee-etat", crochet: true, portee: 1, porte: "chassis", etatSansListe: true},
		{nom: "temoin:portee-record+i0-ecrivain+boucle-decalee+1", crochet: true, portee: 2, i0: true, porte: "chassis", decale: 1},
		{nom: "temoin:portee-record+i0-ecrivain+boucle-decalee-1", crochet: true, portee: 2, i0: true, porte: "chassis", decale: -1},
		{nom: "temoin:mpp8/3+portee-record+i0-ecrivain+boucle-decalee+1", crochet: true, portee: 2, i0: true, porte: "chassis", mpp: rvMPP83, decale: 1},
		{nom: "mpp8/3+portee-record+i0-ecrivain+porte-posee", crochet: true, portee: 2, i0: true, porte: "posee", mpp: rvMPP83},
		{nom: "mpp8/3+portee-record+i0-ecrivain+porte-levee", crochet: true, portee: 2, i0: true, porte: "levee", mpp: rvMPP83},
		{nom: "mpp8/3+production", porte: "chassis", mpp: rvMPP83},
		{nom: "mpp8/3+portee-record+i0-ecrivain", crochet: true, portee: 2, i0: true, porte: "chassis", mpp: rvMPP83},
		{nom: "mpp8/3+portee-record+i0-ecrivain+etat-sans-liste", crochet: true, portee: 2, i0: true, porte: "chassis", etatSansListe: true, mpp: rvMPP83},
		{nom: "mpp8/3+portee-etat+etat-sans-liste", crochet: true, portee: 1, porte: "chassis", etatSansListe: true, mpp: rvMPP83},
	}
}

// rvCompte : les comptes d une (variante, classe).
type rvCompte struct {
	n, fermes, voisins, voisinsFermes, n2ok, sousSurAncre, sur, sous, desync int
	n2                                                                       map[uint64]int
}

// rvSurAncre : la fin `q` d un record « sous » tombe-t-elle sur un en-tete de record valide de
// slot compris entre le slot du record et celui de la frontiere ?
func rvSurAncre(pay []byte, q int, b keyframeBorne, slotSuivant int) bool {
	slot, _, _, ok := kfValidAnchor(pay, q, b.Slot, len(pay)*8)
	if !ok && q >= 0 && q+64 <= len(pay)*8 {
		// GENERATION 0 (D-51) : `kfValidAnchor` la refuse ; on l accepte ici avec le meme filtre fort
		// (mot de 32 bits suivant l identifiant < 50, champ 26 nul).
		id := sourceBits32(pay, q)
		slot, ok = int(id&0x3fffffff), id>>30 == 0 && sourceBits32(pay, q+32) < objectArchetypeCount
	}
	return ok && slot > b.Slot && slot < b.Slot+8192 && (slotSuivant < 0 || slot < slotSuivant)
}

// TestRVehTi40Variantes : l A/B des lectures, par film et par classe de chassis.
func TestRVehTi40Variantes(t *testing.T) {
	racine, sortie, films, physique, _ := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte, rvPortee, rvV = nil, false, false, rvVariante{} }()
	lignes := []string{"film\tbuild\tvariante\tclasse\trecords\tfermes\tvoisins\tvoisins_fermes\tn2_taille_vehicule\t" +
		"sous_sur_ancre\tsous\tsur\tdesync"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-variantes")
		if f == nil {
			continue
		}
		for _, v := range rvVariantes() {
			rvV, rvPortee = v, v.portee >= 1
			prec := f.fc.ProfilDeBalayage().MPP
			if v.mpp.Lead > 0 {
				f.fc.PoserMPP(v.mpp)
			}
			parClasse := map[string]*rvCompte{}
			rvRecords(f, physique, func(r *rvRecord) {
				for _, cl := range []string{r.classe, "tout"} {
					c := parClasse[cl]
					if c == nil {
						c = &rvCompte{n2: map[uint64]int{}}
						parClasse[cl] = c
					}
					c.n++
					if r.b.Voisin {
						c.voisins++
					}
					if r.pre.n2 == rvTailleVehicule {
						c.n2ok++
					}
					if r.pre.n1 != 0 {
						c.n2[r.pre.n2]++
					}
					switch {
					case r.issue == "fermee":
						c.fermes++
						if r.b.Voisin {
							c.voisinsFermes++
						}
					case r.issue == "sous":
						c.sous++
						if rvSurAncre(r.pay, r.tr.EndBit, r.b, -1) {
							c.sousSurAncre++
						}
					case r.issue == "sur":
						c.sur++
					default:
						c.desync++
					}
				}
			})
			f.fc.PoserMPP(prec)
			for cl, c := range parClasse {
				lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s", f.id, f.build, v.nom, cl,
					c.n, c.fermes, c.voisins, c.voisinsFermes, c.n2ok, c.sousSurAncre, c.sous, c.sur, c.desync, rvModal(c.n2)))
			}
			if c := parClasse["tout"]; c != nil {
				t.Logf("%s %-45s fermes %4d / %4d ; voisins %4d / %4d ; n2 %4d", id, v.nom, c.fermes, c.n,
					c.voisinsFermes, c.voisins, c.n2ok)
			}
		}
		fin()
	}
	b2Ecrire(t, sortie, "r_veh_ti40_variantes.tsv", lignes)
}

// rvModal rend « valeur<TAB>records » de la valeur la plus frequente (la plus petite a egalite).
func rvModal(m map[uint64]int) string {
	var best uint64
	bn := -1
	for v, n := range m {
		if n > bn || (n == bn && v < best) {
			best, bn = v, n
		}
	}
	if bn < 0 {
		return "-\t0"
	}
	return fmt.Sprintf("%d\t%d", best, bn)
}

// TestRVehTi40ChercheN2 : sur chaque record `ti=40` dont n1 > 0, cherche les mots de 32 bits
// compris entre 0x400 et 0x1000 (une taille d etat vehicule plausible) dans les 400 bits qui
// suivent la fin du bloc MPP ; rend, par film, les (valeur, decalage depuis la fin du bloc MPP,
// porte bVar14) les plus frequents. L oracle n2 SANS grammaire de l etat par defaut : la vraie
// taille est la valeur que presque tous les records portent.
func TestRVehTi40ChercheN2(t *testing.T) {
	racine, sortie, films, physique, _ := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte, rvPortee, rvV = nil, false, false, rvVariante{} }()
	rvV, rvPortee = rvVariante{nom: "cherche", porte: "chassis"}, false
	lignes := []string{"film\tbuild\tb14\tvaleur\tdecalage_apres_mpp\trecords\trecords_b14_du_film"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-cherche-n2")
		if f == nil {
			continue
		}
		type cle struct {
			b14 bool
			v   uint64
			d   int
		}
		compte, parB14 := map[cle]int{}, map[bool]int{}
		ctx := rvContexte(f)
		rvRecords(f, physique, func(r *rvRecord) {
			if r.pre.n1 == 0 {
				return
			}
			br := LecteurSur(r.pay)
			br.PoserContexte(ctx)
			br.SetBitPos(r.pre.dsDebut)
			consumeVersionPrefix(br)
			consumeMultiplayerPropertiesBlock(br)
			apres := br.BitPos()
			b14 := sourceBits32(r.pay, apres)>>31 == 1
			parB14[b14]++
			for d := 0; d < 400 && apres+d+32 <= len(r.pay)*8; d++ {
				if v := sourceBits32(r.pay, apres+d); v >= 0x400 && v <= 0x1000 {
					compte[cle{b14, v, d}]++
				}
			}
		})
		for k, n := range compte {
			if n*5 >= parB14[k.b14] { // au moins 20 % des records de la porte
				lignes = append(lignes, fmt.Sprintf("%s\t%s\t%t\t%#x\t%d\t%d\t%d", f.id, f.build, k.b14, k.v, k.d, n, parB14[k.b14]))
			}
		}
		fin()
	}
	b2Ecrire(t, sortie, "r_veh_ti40_cherche_n2.tsv", lignes)
}

// TestRVehTi40BitsEtat : les bits de l etat par defaut (V | MPP | reste) jusqu au premier mot de
// 32 bits qui vaut une taille d etat vehicule connue (0x8d8, 0x89c, 0x890), CAMPAGNE_BITS_N
// records par (chassis, porte bVar14).
func TestRVehTi40BitsEtat(t *testing.T) {
	racine, sortie, films, physique, _ := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte, rvPortee, rvV = nil, false, false, rvVariante{} }()
	rvV, rvPortee = rvVariante{nom: "bits", porte: "chassis"}, false
	n, _ := strconv.Atoi(os.Getenv("CAMPAGNE_BITS_N"))
	if n <= 0 {
		n = 2
	}
	tailles := map[uint64]bool{0x8d8: true, 0x89c: true, 0x890: true}
	lignes := []string{"film\tbuild\tchunk\tslot\tchassis\tn1\tbits_V|MPP|reste|n2"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-bits-etat")
		if f == nil {
			continue
		}
		vus := map[[2]uint64]int{}
		ctx := rvContexte(f)
		rvRecords(f, physique, func(r *rvRecord) {
			if r.pre.n1 == 0 {
				return
			}
			br := LecteurSur(r.pay)
			br.PoserContexte(ctx)
			br.SetBitPos(r.pre.dsDebut)
			consumeVersionPrefix(br)
			finV := br.BitPos()
			consumeMultiplayerPropertiesBlock(br)
			finMPP := br.BitPos()
			b14 := sourceBits32(r.pay, finMPP) >> 31
			k := [2]uint64{uint64(r.mpp), b14}
			if vus[k] >= n {
				return
			}
			vus[k]++
			fin := finMPP + 400
			for d := 0; d < 400; d++ {
				if tailles[sourceBits32(r.pay, finMPP+d)] {
					fin = finMPP + d
					break
				}
			}
			var sb strings.Builder
			for q := r.pre.dsDebut; q < fin+32 && q < len(r.pay)*8; q++ {
				if q == finV || q == finMPP || q == fin {
					sb.WriteByte('|')
				}
				sb.WriteByte(byte('0' + sourceBits32(r.pay, q)>>31))
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%04d\t%05d\t%08x\t%d\t%s", f.id, f.build, r.chunk, r.b.Slot,
				r.mpp, r.pre.n1, sb.String()))
		})
		fin()
	}
	b2Ecrire(t, sortie, "r_veh_ti40_bits_etat.tsv", lignes)
}

// rvMPP83 : le decoupage MPP que l oracle n2 designe sur les formats anciens (8/3, profil
// `MPPPourFormat`, section « les deux oracles se contredisent »), laisse INDETERMINE au depot.
var rvMPP83 = profile.MPPWidths{Lead: 8, Index: 3}
