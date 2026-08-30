package replay

// visee_obs114_research_test.go — LOT A2 (suite) : INSTRUMENT D'OBSERVATION DU TYPE 114, OUVERT
// APRES L'ECHEC DE L'ATTRIBUTION A OFFSET FIXE.
//
// CE QUI A ECHOUE, ET POURQUOI CE FICHIER EXISTE. Le balayage de
// visee_signature114_research_test.go rend ZERO tranche candidate dans l'enveloppe : aucune
// tranche de 3 a 16 bits, a aucune position de 7 a 104, ne rassemble 8 des 12 transitions de
// Nilton sous une valeur unique. Deux branches s'ouvrent, et la methode du depot interdit d'en
// choisir une sans mesure (erreur E, « resoudre une contradiction par une echappatoire ») :
//
//	B1. l'identite du bipede n'est PAS dans l'enveloppe de tete (elle vit plus loin, a offset
//	    variable, ou n'est pas rejouee du tout dans cet evenement) ;
//	B2. les paquets apparies aux transitions en phase 6 n'appartiennent PAS tous a Nilton — le
//	    « plus proche voisin » a pu ramasser le zoom d'un autre joueur, et la chronologie n'est
//	    alors couverte par aucune signature unique parce qu'elle ne le PEUT pas.
//
// TROIS MESURES D'OBSERVATION, sans verdict prealable :
//
//	O1. DUMP ALIGNE des paquets de la plage etiquetee : bits 0..48, ecart a la transition la
//	    plus proche, sens presume, et les champs sortis de A1.
//	O2. PAIRES JUMELLES — longueur du plus long prefixe commun a partir du bit 24, sur toutes
//	    les paires de paquets. SEUIL DECLARE : une paire est JUMELLE a >= 32 bits communs. Sous
//	    l'hypothese de bits independants, l'esperance sur 125 paquets vaut 125*124/2 * 2^-32,
//	    soit 2e-6 paire : toute jumelle observee est structurelle, pas fortuite. C'est le
//	    controle de l'observation de phase 6bis (« deux sorties bit-identiques de 24 a 71 »),
//	    qui n'avait jamais ete quantifiee.
//	O3. TEST D'HORLOGE — le champ variable de tete [13 ; 20) est-il un compteur de temps ? Pour
//	    chaque hypothese d'unite (milliseconde du film, tick 60 Hz), on mesure la distribution
//	    des residus (v - k*t) mod 2^w. SEUIL DECLARE : le champ est une HORLOGE si un residu
//	    unique rassemble >= 80 % des paquets ; il est DERIVE DU TEMPS mais decale si le plus
//	    gros residu depasse 25 % (contre 1/128 = 0,8 % attendu au hasard).
//
// SOUS GARDE (OBS114_FILM). Balayage de paquets purs.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 OBS114_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeObs114 -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	obs114FilmEnv  = "OBS114_FILM"
	obs114Jumelle  = 32  // bits communs a partir du bit 24 pour declarer une paire jumelle
	obs114HorlOK   = 0.8 // fraction d'un residu unique pour declarer une horloge
	obs114HorlIndc = 0.25
	// obs114Garde : demi-largeur, en ms, de la zone de decalages exclue du controle O4
	// (elle recouvrirait la vraie position de la chronologie).
	obs114Garde = 5000
)

// obs114Prefixe rend la longueur du plus long prefixe commun de deux paquets a partir de depuis.
func obs114Prefixe(a, b env114Paquet, depuis int) int {
	n := a.nBits
	if b.nBits < n {
		n = b.nBits
	}
	for i := depuis; i < n; i++ {
		if filmdec.ReadBitsAtForDiag(a.pay, i, 1) != filmdec.ReadBitsAtForDiag(b.pay, i, 1) {
			return i - depuis
		}
	}
	return n - depuis
}

// obs114Dump — mesure O1 : les paquets de la plage etiquetee, bit a bit.
func obs114Dump(t *testing.T, pk []env114Paquet, trans []int64, t0, t1 int64) []env114Paquet {
	t.Helper()
	var plage []env114Paquet
	for _, p := range pk {
		if p.tMS >= t0 && p.tMS <= t1 {
			plage = append(plage, p)
		}
	}
	t.Logf("O1. DUMP — %d paquets 114 dans la plage etiquetee ; colonnes : t, appariement,"+
		" champ [8;20), champ [13;20), champ [24;30), puis bits 0..48", len(plage))
	for _, p := range plage {
		i, ecart := obs114Apparie(trans, p.tMS)
		txt := "hors fenetre"
		if i >= 0 {
			sens := "IN "
			if i%2 == 1 {
				sens = "OUT"
			}
			txt = fmt.Sprintf("tr%02d %s %4dms", i, sens, ecart)
		}
		t.Logf("    t=%d %-20s c8=%4d c13=%3d c24=%2d  %s", p.tMS, txt,
			filmdec.ReadBitsAtForDiag(p.pay, 8, 12), filmdec.ReadBitsAtForDiag(p.pay, 13, 7),
			filmdec.ReadBitsAtForDiag(p.pay, 24, 6), obs114Bits(p, 0, 48))
	}
	return plage
}

// obs114Apparie reprend l'appariement du fichier signature (meme fenetre, meme convention).
func obs114Apparie(trans []int64, tMS int64) (int, int64) {
	return sig114Apparie(trans, tMS)
}

// obs114Bits rend une chaine de bits groupes par octet.
func obs114Bits(p env114Paquet, depuis, long int) string {
	var sb strings.Builder
	for i := 0; i < long && depuis+i < p.nBits; i++ {
		if i > 0 && (depuis+i)%8 == 0 {
			sb.WriteByte(' ')
		}
		sb.WriteByte('0' + byte(filmdec.ReadBitsAtForDiag(p.pay, depuis+i, 1)))
	}
	return sb.String()
}

// obs114Jumelles — mesure O2 : les paires de paquets au long prefixe commun.
func obs114Jumelles(t *testing.T, pk []env114Paquet, trans []int64) {
	t.Helper()
	hist := map[int]int{}
	type paire struct {
		i, j, n int
	}
	var fortes []paire
	for i := 0; i < len(pk); i++ {
		for j := i + 1; j < len(pk); j++ {
			n := obs114Prefixe(pk[i], pk[j], 24)
			hist[n/8]++
			if n >= obs114Jumelle {
				fortes = append(fortes, paire{i, j, n})
			}
		}
	}
	var cles []int
	for k := range hist {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	for _, k := range cles {
		parts = append(parts, fmt.Sprintf("%d-%do:%d", k*8, k*8+7, hist[k]))
	}
	t.Logf("O2. PAIRES — %d paires ; distribution de la longueur du prefixe commun depuis le"+
		" bit 24 (par tranche de 8 bits) : %s", len(pk)*(len(pk)-1)/2, strings.Join(parts, " "))
	t.Logf("    paires JUMELLES (>= %d bits communs) : %d", obs114Jumelle, len(fortes))
	sort.Slice(fortes, func(a, b int) bool { return fortes[a].n > fortes[b].n })
	for k, f := range fortes {
		if k == 25 {
			t.Logf("    ... %d autres paires jumelles non listees", len(fortes)-25)
			break
		}
		ai, _ := obs114Apparie(trans, pk[f.i].tMS)
		aj, _ := obs114Apparie(trans, pk[f.j].tMS)
		t.Logf("    %d bits communs : t=%d (tr%d) et t=%d (tr%d)", f.n, pk[f.i].tMS, ai,
			pk[f.j].tMS, aj)
	}
}

// obs114Horloge — mesure O3 : le champ [d ; d+w) suit-il le temps ?
func obs114Horloge(t *testing.T, pk []env114Paquet, d, w int) {
	t.Helper()
	mod := int64(1) << uint(w)
	unites := []struct {
		nom string
		k   func(tMS int64) int64
	}{
		{"milliseconde du film", func(tMS int64) int64 { return tMS }},
		{"tick 60 Hz", func(tMS int64) int64 { return tMS * 60 / 1000 }},
		{"tick 30 Hz", func(tMS int64) int64 { return tMS * 30 / 1000 }},
		{"centiseconde", func(tMS int64) int64 { return tMS / 10 }},
	}
	for _, u := range unites {
		res := map[int64]int{}
		for _, p := range pk {
			v := int64(filmdec.ReadBitsAtForDiag(p.pay, d, w))
			r := ((v-u.k(p.tMS))%mod + mod) % mod
			res[r]++
		}
		var best int64
		var bestN int
		for r, n := range res {
			if n > bestN {
				best, bestN = r, n
			}
		}
		frac := float64(bestN) / float64(len(pk))
		verdict := "aucun lien"
		switch {
		case frac >= obs114HorlOK:
			verdict = "HORLOGE"
		case frac >= obs114HorlIndc:
			verdict = "LIEN PARTIEL"
		}
		t.Logf("O3. HORLOGE [%d ; %d) en %s — residu dominant %d sur %d valeurs distinctes :"+
			" %d/%d paquets (%.1f %%, hasard %.1f %%) -> %s", d, d+w, u.nom, best, len(res),
			bestN, len(pk), 100*frac, 100/float64(mod), verdict)
	}
}

// obs114Couverture compte les transitions couvertes par au moins un paquet, chronologie
// decalee de delta millisecondes (delta = 0 : la position vraie).
func obs114Couverture(pk []env114Paquet, trans []int64, delta int64) int {
	couverts := 0
	for _, tr := range trans {
		cible := tr + delta
		for _, p := range pk {
			d := p.tMS - cible
			if d < 0 {
				d = -d
			}
			if d <= sig114FenetreMS {
				couverts++
				break
			}
		}
	}
	return couverts
}

// obs114Significativite — mesure O4 : L'ALIGNEMENT DU TYPE 114 SUR LA CHRONOLOGIE EST-IL PLUS
// QU'UNE COINCIDENCE ?
//
// La phase 6 a conclu de « 9 transitions sur 12 couvertes » que le type 114 portait le zoom,
// en comparant la densite des fenetres au DEBIT DE FOND du film entier. Ce controle est le
// mauvais : ce qui compte est le nombre de paquets 114 PRESENTS DANS LA PLAGE ETIQUETEE (17)
// et la part de cette plage que les 12 fenetres +/-1,2 s recouvrent deja. D'ou ce controle
// exact, qui ne tire rien au hasard : la chronologie entiere est TRANSLATEE de delta, ce qui
// preserve a la fois les ecarts entre transitions et la structure temporelle reelle des
// paquets. Les decalages a moins de obs114Garde ms sont exclus (ils recouvrent la vraie
// position).
//
// SEUIL ECRIT AVANT LA MESURE : l'alignement est declare SIGNIFICATIF si moins de 1 % des
// decalages de controle atteignent la couverture observee.
func obs114Significativite(t *testing.T, pk []env114Paquet, trans []int64) {
	t.Helper()
	obs := obs114Couverture(pk, trans, 0)
	var total, auMoins int
	var somme, max int
	for delta := int64(-400_000); delta <= 400_000; delta += 100 {
		if delta > -obs114Garde && delta < obs114Garde {
			continue
		}
		c := obs114Couverture(pk, trans, delta)
		total++
		somme += c
		if c > max {
			max = c
		}
		if c >= obs {
			auMoins++
		}
	}
	frac := float64(auMoins) / float64(total)
	verdict := "NON SIGNIFICATIF"
	if frac < 0.01 {
		verdict = "SIGNIFICATIF"
	}
	t.Logf("O4. SIGNIFICATIVITE — couverture observee %d/%d transitions ; sur %d decalages de"+
		" controle : moyenne %.1f, maximum %d, part atteignant %d ou plus : %.2f %% -> %s",
		obs, len(trans), total, float64(somme)/float64(total), max, obs, 100*frac, verdict)
}

// obs114Calage — mesure O4bis : la couverture en fonction du decalage, AUTOUR de zero. Si le
// recalage feed->film est juste, le maximum local tombe sur delta = 0 ; un maximum franc a
// quelques secondes de la dirait que le decalage fige est faux, et O4 mesurerait a cote.
func obs114Calage(t *testing.T, pk []env114Paquet, trans []int64) {
	t.Helper()
	var lignes []string
	meilleur, meilleurD := -1, int64(0)
	for delta := int64(-10_000); delta <= 10_000; delta += 500 {
		c := obs114Couverture(pk, trans, delta)
		lignes = append(lignes, fmt.Sprintf("%+ds:%d", delta/1000, c))
		if c > meilleur {
			meilleur, meilleurD = c, delta
		}
	}
	t.Logf("O4bis. CALAGE — couverture par decalage (pas de 500 ms) : %s",
		strings.Join(lignes, " "))
	t.Logf("    maximum local %d transitions a delta = %+d ms (0 = decalage fige)",
		meilleur, meilleurD)
}

// TestViseeObs114 execute les mesures d'observation.
func TestViseeObs114(t *testing.T) {
	dir := os.Getenv(obs114FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", obs114FilmEnv)
	}
	pk := env114Collecte(dir)
	if len(pk) == 0 {
		t.Fatalf("aucun paquet 114 dans %s", dir)
	}
	trans, t0, t1 := sig114Fenetres()
	obs114Dump(t, pk, trans, t0, t1)
	obs114Jumelles(t, pk, trans)
	obs114Horloge(t, pk, 13, 7)
	obs114Horloge(t, pk, 8, 12)
	obs114Horloge(t, pk, 9, 11)
	obs114Significativite(t, pk, trans)
	obs114Calage(t, pk, trans)
}
