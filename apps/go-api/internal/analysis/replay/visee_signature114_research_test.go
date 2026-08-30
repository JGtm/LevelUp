package replay

// visee_signature114_research_test.go — LOT A2/A3 : ATTRIBUTION DES PAQUETS 114 A UN JOUEUR PAR
// SIGNATURE DE BITS, PUIS SENS ENTREE/SORTIE DE LUNETTE.
//
// L'ETALON. La chronologie relevee a la main par l'utilisateur sur 00162144 (`chronoEpisodes`,
// partagee avec visee_chronologie_research_test.go — source unique) donne 12 transitions de
// Nilton410 entre 0:41 et 1:26 d'horloge du feed. Le decalage feed->film de ce film vaut
// +1 171 858 ms, etabli en phase 6 par appariement de 91 fins de vie (pont morts <-> pistes) ;
// il est fige ici en constante pour eviter un balayage de positions couteux, et le fichier le
// declare comme une DEPENDANCE, pas comme une mesure de cet instrument.
//
// POURQUOI UNE SEULE SIGNATURE SUFFIT SUR CETTE PLAGE : la vie de Nilton qui porte les douze
// transitions court de 1211,1 a 1337,4 s (film), soit tout l'intervalle etiquete [1212,9 ;
// 1257,9] s. Aucun respawn ne s'y produit : si un champ porte l'identite du bipede, il garde
// une valeur UNIQUE sur toute la plage. C'est ce qui rend le critere ci-dessous severe.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	A2. Une tranche de bits [d ; d+w) est CANDIDATE s'il existe UNE SEULE valeur v telle que
//	    (a) les paquets portant v couvrent >= 8 des 12 transitions a +/- 1,2 s (la precision du
//	        releve video), et
//	    (b) au plus 2 paquets portant v tombent HORS de toute fenetre dans la plage etiquetee
//	        elargie de 3 s de chaque cote.
//	    Le « une seule valeur » est la contrainte qui interdit le sur-ajustement : sans elle on
//	    couvrirait 12/12 en prenant l'ensemble des paquets des fenetres.
//	A2bis. CONTROLE DE TRIVIALITE : si le nombre de paquets 114 hors fenetres dans la plage est
//	    lui-meme <= 2, le critere (b) ne discrimine rien et la mesure est declaree NON
//	    CONCLUANTE.
//	A2ter. CONTROLE NEGATIF : le meme balayage tourne sur les tranches situees APRES la fin
//	    plausible de l'evenement (d >= 32, ou coule le reste du flux delta). Le nombre de
//	    tranches qui y passent le critere est le taux de faux positifs, publie en chiffres.
//	A3. Une tranche SEPARE le sens si, sur les paquets attribues a Nilton, les valeurs prises
//	    aux entrees et celles prises aux sorties sont disjointes, avec >= 4 entrees et >= 4
//	    sorties appariees — ET si la tranche n'est pas triviale : sa cardinalite sur les 125
//	    paquets du film doit rester <= 8 (une tranche large donne une valeur unique par paquet
//	    et « separe » toujours).
//
// SOUS GARDE (SIG114_FILM, qui doit pointer 00162144). Balayage de paquets purs.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 SIG114_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeSignature114 -v -timeout 30m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	sig114FilmEnv = "SIG114_FILM"
	// sig114OffsetMS : decalage feed -> film du SEUL film 00162144 (phase 6, 91 fins de vie
	// appariees). Dependance figee, pas une mesure de cet instrument.
	sig114OffsetMS  = 1171858
	sig114FenetreMS = 1200
	sig114MargeMS   = 3000
	sig114CouvMin   = 8
	sig114ImpurMax  = 2
	sig114DBruit    = 32 // debut de la zone de controle negatif
	sig114WMin      = 3
	sig114WMax      = 16
	sig114CardMax   = 8 // cardinalite maximale d'une tranche non triviale (A3)
)

// sig114Tranche decrit le resultat du critere A2 pour une tranche de bits.
type sig114Tranche struct {
	d, w      int
	valeur    uint32
	couv      int
	impur     int
	nPaquets  int
	nEnPlage  int
	cardTotal int
}

// sig114Fenetres rend les 12 instants de transition (film, ms) et les bornes de la plage.
func sig114Fenetres() (trans []int64, t0, t1 int64) {
	eps := chronoVersFilm(chronoEpisodes, sig114OffsetMS)
	trans = chronoTransitions(eps)
	return trans, eps[0][0] - sig114MargeMS, eps[len(eps)-1][1] + sig114MargeMS
}

// sig114Apparie rend l'index de la transition la plus proche et l'ecart, ou -1 hors fenetre.
func sig114Apparie(trans []int64, tMS int64) (int, int64) {
	best, bd := -1, int64(sig114FenetreMS)+1
	for i, tr := range trans {
		d := tMS - tr
		if d < 0 {
			d = -d
		}
		if d <= sig114FenetreMS && d < bd {
			best, bd = i, d
		}
	}
	return best, bd
}

// sig114Evalue applique le critere A2 a une tranche : rend la meilleure valeur unique.
func sig114Evalue(pk []env114Paquet, d, w int, trans []int64, t0, t1 int64) sig114Tranche {
	parValeur := map[uint32][]int64{}
	cardTotal := map[uint32]bool{}
	for _, p := range pk {
		if d+w > p.nBits {
			continue
		}
		v := filmdec.ReadBitsAtForDiag(p.pay, d, w)
		cardTotal[v] = true
		if p.tMS >= t0 && p.tMS <= t1 {
			parValeur[v] = append(parValeur[v], p.tMS)
		}
	}
	best := sig114Tranche{d: d, w: w, cardTotal: len(cardTotal)}
	for v, instants := range parValeur {
		couverts := map[int]bool{}
		impur := 0
		for _, tMS := range instants {
			if i, _ := sig114Apparie(trans, tMS); i >= 0 {
				couverts[i] = true
			} else {
				impur++
			}
		}
		if len(couverts) > best.couv || (len(couverts) == best.couv && impur < best.impur) {
			best.valeur, best.couv, best.impur = v, len(couverts), impur
			best.nEnPlage = len(instants)
		}
	}
	for _, p := range pk {
		if d+w <= p.nBits && filmdec.ReadBitsAtForDiag(p.pay, d, w) == best.valeur {
			best.nPaquets++
		}
	}
	return best
}

// sig114Trivialite — mesure A2bis : le critere d'impurete discrimine-t-il quelque chose ?
func sig114Trivialite(t *testing.T, pk []env114Paquet, trans []int64, t0, t1 int64) int {
	t.Helper()
	var enPlage, horsFenetre int
	for _, p := range pk {
		if p.tMS < t0 || p.tMS > t1 {
			continue
		}
		enPlage++
		if i, _ := sig114Apparie(trans, p.tMS); i < 0 {
			horsFenetre++
		}
	}
	t.Logf("A2bis. TRIVIALITE — %d paquets 114 dans la plage etiquetee [%d ; %d] ms, dont %d"+
		" HORS des 12 fenetres +/-%d ms", enPlage, t0, t1, horsFenetre, sig114FenetreMS)
	if horsFenetre <= sig114ImpurMax {
		t.Logf("   NON CONCLUANT : le critere d'impurete (<= %d) ne discrimine rien.", sig114ImpurMax)
	} else {
		t.Logf("   critere d'impurete EXPLOITABLE (%d paquets a rejeter au maximum).", horsFenetre)
	}
	return horsFenetre
}

// sig114Balaye — mesure A2 : toutes les tranches, avec separation signal / controle negatif.
func sig114Balaye(t *testing.T, pk []env114Paquet, nb int, trans []int64, t0, t1 int64) []sig114Tranche {
	t.Helper()
	var passants, faux []sig114Tranche
	for d := 7; d+sig114WMin <= nb; d++ {
		for w := sig114WMin; w <= sig114WMax && d+w <= nb; w++ {
			r := sig114Evalue(pk, d, w, trans, t0, t1)
			if r.couv < sig114CouvMin || r.impur > sig114ImpurMax {
				continue
			}
			if d >= sig114DBruit {
				faux = append(faux, r)
			} else {
				passants = append(passants, r)
			}
		}
	}
	t.Logf("A2. BALAYAGE — %d tranches CANDIDATES dans l'enveloppe (d < %d), %d dans la zone de"+
		" controle negatif (d >= %d)", len(passants), sig114DBruit, len(faux), sig114DBruit)
	sig114Front(t, pk, nb, trans, t0, t1)
	sort.Slice(passants, func(i, j int) bool {
		if passants[i].couv != passants[j].couv {
			return passants[i].couv > passants[j].couv
		}
		return passants[i].impur < passants[j].impur
	})
	for _, r := range passants {
		t.Logf("    bits [%d ; %d) valeur %d (0x%x) : %d/12 transitions, %d impurs, %d paquets"+
			" au film / %d dans la plage, cardinalite de la tranche %d",
			r.d, r.d+r.w, r.valeur, r.valeur, r.couv, r.impur, r.nPaquets, r.nEnPlage, r.cardTotal)
	}
	for _, r := range faux {
		t.Logf("    [controle negatif] bits [%d ; %d) valeur %d : %d/12, %d impurs, %d paquets",
			r.d, r.d+r.w, r.valeur, r.couv, r.impur, r.nPaquets)
	}
	return passants
}

// sig114Front — A QUEL POINT L'ATTRIBUTION ECHOUE-T-ELLE ? Un balayage qui ne rend « aucun
// candidat » ne dit pas s'il est passe a un cheveu ou tres loin. Cette mesure publie, pour
// chaque position de depart de l'enveloppe, la MEILLEURE couverture atteinte par une valeur
// unique, toutes largeurs et toutes impuretes confondues : c'est le plafond de ce qu'une
// signature a offset fixe peut faire.
func sig114Front(t *testing.T, pk []env114Paquet, nb int, trans []int64, t0, t1 int64) {
	t.Helper()
	var lignes []string
	meilleure := sig114Tranche{}
	for d := 7; d < sig114DBruit && d+sig114WMin <= nb; d++ {
		best := sig114Tranche{}
		for w := sig114WMin; w <= sig114WMax && d+w <= nb; w++ {
			r := sig114Evalue(pk, d, w, trans, t0, t1)
			if r.couv > best.couv || (r.couv == best.couv && r.impur < best.impur) {
				best = r
			}
		}
		lignes = append(lignes, fmt.Sprintf("%d:%d/%d", d, best.couv, best.impur))
		if best.couv > meilleure.couv {
			meilleure = best
		}
	}
	t.Logf("A2. FRONT — meilleure couverture par position de depart (position:couverture/impurs) :"+
		" %s", strings.Join(lignes, " "))
	t.Logf("    plafond de l'enveloppe : %d/12 transitions (bits [%d ; %d), valeur %d, %d impurs,"+
		" %d paquets dans la plage) — le critere exigeait %d/12 avec au plus %d impurs",
		meilleure.couv, meilleure.d, meilleure.d+meilleure.w, meilleure.valeur, meilleure.impur,
		meilleure.nEnPlage, sig114CouvMin, sig114ImpurMax)
}

// sig114Detaille imprime les paquets portant la signature retenue (attribution finale).
func sig114Detaille(t *testing.T, pk []env114Paquet, r sig114Tranche, trans []int64) []env114Paquet {
	t.Helper()
	var retenus []env114Paquet
	for _, p := range pk {
		if r.d+r.w <= p.nBits && filmdec.ReadBitsAtForDiag(p.pay, r.d, r.w) == r.valeur {
			retenus = append(retenus, p)
		}
	}
	t.Logf("A2. ATTRIBUTION — signature bits [%d ; %d) = %d : %d paquets sur tout le film",
		r.d, r.d+r.w, r.valeur, len(retenus))
	for _, p := range retenus {
		i, ecart := sig114Apparie(trans, p.tMS)
		sens, txt := "-", "hors fenetre"
		if i >= 0 {
			if i%2 == 0 {
				sens = "IN "
			} else {
				sens = "OUT"
			}
			txt = fmt.Sprintf("transition %2d (%s), ecart %4d ms", i, sens, ecart)
		}
		t.Logf("    t=%d ms · %s", p.tMS, txt)
	}
	return retenus
}

// sig114SensTranche teste le critere A3 sur une tranche pour un lot de paquets attribues.
func sig114SensTranche(pk, tous []env114Paquet, d, w int, trans []int64) (bool, string) {
	vIn, vOut := map[uint32]bool{}, map[uint32]bool{}
	var nIn, nOut int
	for _, p := range pk {
		if d+w > p.nBits {
			return false, ""
		}
		i, _ := sig114Apparie(trans, p.tMS)
		if i < 0 {
			continue
		}
		v := filmdec.ReadBitsAtForDiag(p.pay, d, w)
		if i%2 == 0 {
			vIn[v] = true
			nIn++
		} else {
			vOut[v] = true
			nOut++
		}
	}
	if nIn < 4 || nOut < 4 {
		return false, ""
	}
	for v := range vIn {
		if vOut[v] {
			return false, ""
		}
	}
	card := map[uint32]bool{}
	for _, p := range tous {
		if d+w <= p.nBits {
			card[filmdec.ReadBitsAtForDiag(p.pay, d, w)] = true
		}
	}
	if len(card) > sig114CardMax {
		return false, ""
	}
	return true, fmt.Sprintf("bits [%d ; %d) : IN=%s OUT=%s (%d entrees, %d sorties,"+
		" cardinalite film %d)", d, d+w, sig114Valeurs(vIn), sig114Valeurs(vOut), nIn, nOut, len(card))
}

func sig114Valeurs(m map[uint32]bool) string {
	var vs []int
	for v := range m {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var parts []string
	for _, v := range vs {
		parts = append(parts, fmt.Sprint(v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// sig114Sens — mesure A3 : quelle tranche separe les entrees des sorties ?
func sig114Sens(t *testing.T, retenus, tous []env114Paquet, nb int, trans []int64) {
	t.Helper()
	var trouves int
	for d := 7; d+sig114WMin <= nb; d++ {
		for w := sig114WMin; w <= sig114WMax && d+w <= nb; w++ {
			ok, txt := sig114SensTranche(retenus, tous, d, w, trans)
			if !ok {
				continue
			}
			trouves++
			zone := "enveloppe"
			if d >= sig114DBruit {
				zone = "CONTROLE NEGATIF"
			}
			t.Logf("A3. SEPARATION [%s] %s", zone, txt)
		}
	}
	if trouves == 0 {
		t.Log("A3. SEPARATION — aucune tranche non triviale ne separe les entrees des sorties" +
			" aux seuils declares.")
	}
}

// sig114Palette imprime les valeurs prises par une tranche sur tous les paquets (descriptif).
func sig114Palette(t *testing.T, pk []env114Paquet, d, w int) {
	t.Helper()
	comptes := map[uint32]int{}
	for _, p := range pk {
		if d+w <= p.nBits {
			comptes[filmdec.ReadBitsAtForDiag(p.pay, d, w)]++
		}
	}
	var vs []int
	for v := range comptes {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var parts []string
	for _, v := range vs {
		parts = append(parts, fmt.Sprintf("%d:%d", v, comptes[uint32(v)]))
	}
	t.Logf("    palette bits [%d ; %d) — %d valeurs : %s", d, d+w, len(vs), strings.Join(parts, " "))
}

// TestViseeSignature114 execute A2 puis A3.
func TestViseeSignature114(t *testing.T) {
	dir := os.Getenv(sig114FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", sig114FilmEnv)
	}
	if filepath.Base(dir) != "00162144" {
		t.Fatalf("la chronologie relevee est celle de 00162144 ; film fourni : %s", filepath.Base(dir))
	}
	pk := env114Collecte(dir)
	if len(pk) == 0 {
		t.Fatalf("aucun paquet 114 dans %s", dir)
	}
	nb := env114BitsCommuns(pk)
	if nb > env114MaxBits {
		nb = env114MaxBits
	}
	trans, t0, t1 := sig114Fenetres()
	t.Logf("ETALON — %d paquets 114 ; %d transitions ; decalage feed->film fige a %d ms",
		len(pk), len(trans), sig114OffsetMS)

	sig114Trivialite(t, pk, trans, t0, t1)
	t.Log("PALETTES DESCRIPTIVES des champs courts sortis de A1 :")
	sig114Palette(t, pk, 24, 6)
	sig114Palette(t, pk, 25, 6)
	sig114Palette(t, pk, 13, 7)

	passants := sig114Balaye(t, pk, nb, trans, t0, t1)
	if len(passants) == 0 {
		t.Log("A2. VERDICT — aucune tranche a offset fixe n'attribue les 12 transitions a une" +
			" signature unique aux seuils declares.")
		return
	}
	retenus := sig114Detaille(t, pk, passants[0], trans)
	sig114Sens(t, retenus, pk, nb, trans)
}
