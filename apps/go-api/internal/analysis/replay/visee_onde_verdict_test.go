package replay

// visee_onde_verdict_test.go — LOT C : LE CONTROLE PAR TRANSLATION ET LE VERDICT.
//
// Le moteur de mesure (onde, collecte, colonnes de bits, exactitude equilibree) et les SEUILS
// ECRITS AVANT MESURE sont dans `visee_onde_research_test.go`. Ce fichier ne fait que deux
// choses : appliquer le controle par translation, et publier.
//
// CE QUE LE CONTROLE PAR TRANSLATION REPOND. « La meilleure de 505 positions x 2 polarites
// atteint 0,93 » ne veut rien dire tant qu'on ne sait pas ce qu'atteint la meilleure de 505
// positions x 2 polarites CONTRE UNE ONDE QUI NE DECRIT RIEN. On translate donc l'onde entiere
// (creneaux, transitions, gardes ET fenetre d'analyse) de delta millisecondes et on rejoue la
// mesure complete. Le flux reel garde toute sa structure — sa repetitivite, ses rafales, ses
// silences — et c'est precisement ce qu'aucun tirage aleatoire ne saurait reproduire (regle 4 de
// METHODE_RETRO_INGENIERIE_FILM.md : mesurer les faux positifs sur le flux reel, jamais les
// calculer).
//
// DEUX PARTS SONT PUBLIEES, et la severe fait foi :
//   - p(max) : part des decalages ou le MEILLEUR score toutes positions confondues atteint le
//     score observe. Elle corrige d'elle-meme le balayage de ~1000 hypotheses ;
//   - p(pos) : part des decalages ou la seule position candidate atteint son score observe
//     (le controle du lot A, publie pour comparaison).
// Une troisieme part, p(max apparie), restreint le controle aux decalages dont la classe
// « zoome » compte au moins la moitie des echantillons observes : un decalage tombant en fin de
// film, ou les paquets se rarefient, n'est pas un controle honnete.
//
// SOUS GARDE (ONDE_FILM). Aucun code de production touche.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// ondeCtrl porte le resultat du controle par translation.
type ondeCtrl struct {
	deltas, apparies     int
	partMax, partApparie float64
	partPos              float64
	moyMax, maxMax       float64
	n1Min, n1Med, n1Max  int
	// Distribution des maxima de controle : c'est la PUISSANCE de l'instrument. La part des
	// decalages atteignant 1,0000 est la p(max) qu'obtiendrait un signal PARFAIT ; si elle
	// depassait 1 %, aucun signal, meme exact, ne pourrait etre declare significatif — et le
	// negatif publie ne vaudrait rien.
	part85, part95, part99, part100 float64
}

// ondeControle rejoue la mesure complete pour chaque translation de l'onde.
func ondeControle(c *ondeCol, o ondeCarree, obs ondeScore, n1Obs int) ondeCtrl {
	var ct ondeCtrl
	var n1s []int
	var somme float64
	var auMoinsMax, auMoinsApp, auMoinsPos int
	var n85, n95, n99, n100 int
	for delta := int64(-ondeCtrlAmplMS); delta <= ondeCtrlAmplMS; delta += ondeCtrlPasMS {
		if delta > -ondeCtrlGardeMS && delta < ondeCtrlGardeMS {
			continue
		}
		m := c.marque(o, delta)
		if m.n1 == 0 || m.n0 == 0 {
			continue
		}
		best := c.meilleur(m)
		ct.deltas++
		n1s = append(n1s, m.n1)
		somme += best.score
		if best.score > ct.maxMax {
			ct.maxMax = best.score
		}
		if best.score >= obs.score {
			auMoinsMax++
		}
		for seuil, compteur := range map[float64]*int{0.85: &n85, 0.95: &n95, 0.99: &n99, 1: &n100} {
			if best.score >= seuil {
				*compteur++
			}
		}
		if m.n1*2 >= n1Obs {
			ct.apparies++
			if best.score >= obs.score {
				auMoinsApp++
			}
		}
		if obs.pos >= 0 && c.ondeScorePos(obs.pos, m) >= obs.score {
			auMoinsPos++
		}
	}
	if ct.deltas == 0 {
		return ct
	}
	ct.moyMax = somme / float64(ct.deltas)
	ct.partMax = float64(auMoinsMax) / float64(ct.deltas)
	ct.partPos = float64(auMoinsPos) / float64(ct.deltas)
	if ct.apparies > 0 {
		ct.partApparie = float64(auMoinsApp) / float64(ct.apparies)
	}
	d := float64(ct.deltas)
	ct.part85, ct.part95 = float64(n85)/d, float64(n95)/d
	ct.part99, ct.part100 = float64(n99)/d, float64(n100)/d
	sort.Ints(n1s)
	ct.n1Min, ct.n1Med, ct.n1Max = n1s[0], n1s[len(n1s)/2], n1s[len(n1s)-1]
	return ct
}

// ondeVerdict rend le verdict aux seuils declares.
func ondeVerdict(s ondeScore, n1, n0 int, ct ondeCtrl) string {
	if n1 < ondeEchMin || n0 < ondeEchMin {
		if ct.partMax < 0.01 {
			return "SOUS-DIMENSIONNE mais le controle par translation tient (p(max) < 1 %)"
		}
		return "SOUS-DIMENSIONNE (S3) — et le controle par translation ne tient pas : NEGATIF"
	}
	switch {
	case s.score >= ondeSeuilCand && ct.partMax < 0.01:
		return "CANDIDAT (S1 + S4)"
	case s.score >= ondeSeuilCand:
		return "score S1 atteint mais CONTROLE PAR TRANSLATION ECHOUE : NEGATIF"
	case s.score >= ondeSeuilSuivre && ct.partMax < 0.01:
		return "A SUIVRE (S2 + S4)"
	default:
		return "NEGATIF"
	}
}

// ondePosition met en forme une position selon la variante (tete ou fin de payload).
func ondePosition(pos, nbits int, queue bool) string {
	if queue {
		return fmt.Sprintf("bit %d du buffer de fin (= %d bits avant la fin du payload)",
			pos, nbits-pos)
	}
	return fmt.Sprintf("bit %d", pos)
}

// ondeVariante execute une variante complete et publie tout : dimensionnement, classement,
// controle par translation, verdict.
func ondeVariante(t *testing.T, nom string, c *ondeCol, o ondeCarree, queue bool) ondeScore {
	t.Helper()
	debut := time.Now()
	m := c.marque(o, 0)
	if m.n1 == 0 || m.n0 == 0 {
		t.Logf("[%s / %s] AUCUN ECHANTILLON (classe 1 : %d, classe 0 : %d) — variante impossible",
			nom, o.nom, m.n1, m.n0)
		return ondeScore{pos: -1}
	}
	cl := c.classement(m)
	best := cl[0]
	ct := ondeControle(c, o, best, m.n1)
	t.Logf("[%s / %s] %d paquets collectes, dont %d dans la fenetre d'analyse : %d classe 1"+
		" (zoome), %d classe 0, %d exclus par les bandes de garde de +/-%d ms ; %d paquets hors"+
		" fenetre (reserve du controle par translation) ; %d positions de bit balayees",
		nom, o.nom, len(c.temps), m.n1+m.n0+m.nGarde, m.n1, m.n0, m.nGarde, o.garde, m.nHors,
		c.nbits-ondeBitMin)
	for i := 0; i < 8 && i < len(cl); i++ {
		t.Logf("    %d. %s polarite %+d : exactitude equilibree %.4f (%d vrais positifs,"+
			" %d faux positifs)", i+1, ondePosition(cl[i].pos, c.nbits, queue), cl[i].polarite,
			cl[i].score, cl[i].tp, cl[i].fp)
	}
	t.Logf("    CONTROLE PAR TRANSLATION — %d decalages valides (pas %d ms, amplitude +/-%d s,"+
		" garde +/-%d s) ; classe 1 par decalage : min %d, mediane %d, max %d", ct.deltas,
		ondeCtrlPasMS, ondeCtrlAmplMS/1000, ondeCtrlGardeMS/1000, ct.n1Min, ct.n1Med, ct.n1Max)
	t.Logf("    meilleur score de controle : moyenne %.4f, maximum %.4f (observe : %.4f)",
		ct.moyMax, ct.maxMax, best.score)
	t.Logf("    PUISSANCE — part des decalages de controle dont le meilleur score atteint"+
		" 0,85 : %.2f %% · 0,95 : %.2f %% · 0,99 : %.2f %% · 1,0000 : %.2f %% (cette derniere"+
		" est la p(max) qu'obtiendrait un signal PARFAIT)", 100*ct.part85, 100*ct.part95,
		100*ct.part99, 100*ct.part100)
	t.Logf("    p(max) = %.2f %% · p(max apparie en taille, %d decalages) = %.2f %% ·"+
		" p(pos %d) = %.2f %%", 100*ct.partMax, ct.apparies, 100*ct.partApparie, best.pos,
		100*ct.partPos)
	t.Logf("    VERDICT : %s   [%s]", ondeVerdict(best, m.n1, m.n0, ct),
		time.Since(debut).Round(time.Millisecond))
	return best
}

// ondeDimensionne publie ce que l'onde laisse comme matiere apres les bandes de garde.
func ondeDimensionne(t *testing.T, o ondeCarree) {
	t.Helper()
	var parts []string
	for _, e := range o.eps {
		parts = append(parts, fmt.Sprintf("[%d ; %d]", e[0], e[1]))
	}
	t.Logf("ONDE [%s] — %d episodes %s ; %d transitions ; garde +/-%d ms ; duree effectivement"+
		" classee « zoome » apres gardes : %d ms", o.nom, len(o.eps), strings.Join(parts, " "),
		len(o.trans), o.garde, o.dureeClasse1())
}

// ondeMesure0 publie le debit du film par tete d'evenement (mesure descriptive, sans seuil).
func ondeMesure0(t *testing.T, pk []ondePaquet, courts int) {
	t.Helper()
	debits := ondeDebitParTete(pk)
	type ligne struct{ tete, n int }
	var ls []ligne
	for tete, n := range debits {
		ls = append(ls, ligne{tete, n})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })
	var parts []string
	for i, l := range ls {
		if i == 12 {
			parts = append(parts, fmt.Sprintf("... %d autres tetes", len(ls)-12))
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", l.tete, l.n))
	}
	var duree int64 = 1
	if len(pk) > 1 {
		duree = (pk[len(pk)-1].tMS - pk[0].tMS) / 1000
	}
	t.Logf("MESURE 0 (descriptive) — %d paquets delta collectes sur %d s de film (%d/s), %d"+
		" tetes distinctes, %d paquets ecartes car payload < %d octets", len(pk), duree,
		int64(len(pk))/duree, len(ls), courts, ondeOctetMin)
	t.Logf("    paquets par tete d'evenement (decroissant) : %s", strings.Join(parts, " "))
}

// ondeVar decrit une variante : un sous-ensemble de tetes d'evenement, et le bout du payload
// ou les positions de bit sont prises.
type ondeVar struct {
	nom   string
	tetes map[int]bool
	queue bool
}

// ondePasse execute une variante sur les deux ondes de Nilton, puis — SEULEMENT si un candidat
// atteint le seuil « a suivre » — la contre-verification C4 sur l'onde de Madina.
func ondePasse(t *testing.T, v ondeVar, pk []ondePaquet, ondes []ondeCarree) {
	t.Helper()
	lot := ondeFiltre(pk, v.tetes)
	if len(lot) == 0 {
		t.Logf("[%s] aucun paquet : variante sautee", v.nom)
		return
	}
	c := ondeBatColonnes(lot, v.queue)
	defer func() { c.col = nil }()
	best := ondeVariante(t, v.nom, c, ondes[0], v.queue)
	ondeVariante(t, v.nom, c, ondes[1], v.queue)
	if best.pos < 0 || best.score < ondeSeuilSuivre {
		return
	}
	t.Logf("[%s] C4 — contre-verification sur l'onde de Madina (creneau unique) :", v.nom)
	for _, o := range ondes[2:] {
		ondeVariante(t, v.nom, c, o, v.queue)
	}
}

// TestViseeOndeCarree — LE LOT C. Cinq variantes sur 512 bits (debut de payload toutes tetes,
// debut tete 80, fin de payload toutes tetes, fin tete 80, debut tetes 96/97/116), puis deux
// variantes elargies a 1024 bits ; l'onde de Nilton en garde longue puis courte, et l'onde de
// Madina en contre-verification.
func TestViseeOndeCarree(t *testing.T) {
	dir := os.Getenv(ondeFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", ondeFilmEnv)
	}
	if filepath.Base(dir) != "00162144" {
		t.Fatalf("la chronologie relevee est celle de 00162144 ; film fourni : %s", filepath.Base(dir))
	}
	debut := time.Now()
	t0 := int64(ondeFeneDebutMS - ondeCtrlAmplMS - 1000)
	t1 := int64(ondeFeneFinMS + ondeCtrlAmplMS + 1000)
	pk, courts := ondeCollecte(dir, t0, t1, ondeOctetMin)
	if len(pk) == 0 {
		t.Fatalf("aucun paquet delta collecte dans %s", dir)
	}
	t.Logf("COLLECTE — %d paquets en %s ; fenetre d'analyse [%d ; %d] ms (feed 35 s -> 95 s,"+
		" decalage fige %d ms)", len(pk), time.Since(debut).Round(time.Millisecond),
		ondeFeneDebutMS, ondeFeneFinMS, ondeOffsetMS)
	ondeMesure0(t, pk, courts)

	ondes := []ondeCarree{
		ondeConstruit("Nilton410", chronoEpisodes, ondeGardeMS),
		ondeConstruit("Nilton410 garde 0,5 s", chronoEpisodes, ondeGardeBrefMS),
		ondeConstruit("Madina97294", [][2]float64{chronoEpisodeMadina}, ondeGardeMS),
		ondeConstruit("Madina97294 garde 0,5 s", [][2]float64{chronoEpisodeMadina}, ondeGardeBrefMS),
	}
	for _, o := range ondes {
		ondeDimensionne(t, o)
	}
	for _, v := range []ondeVar{
		{"C2 toutes tetes / debut de payload", nil, false},
		{"C2 tetes 80 / debut de payload", ondeTetes(80), false},
		{"C2bis toutes tetes / FIN de payload", nil, true},
		{"C2bis tetes 80 / FIN de payload", ondeTetes(80), true},
		{"C5 tetes 96+97+116 / debut de payload", ondeTetes(96, 97, 116), false},
	} {
		ondePasse(t, v, pk, ondes)
	}

	// ELARGISSEMENT DU DOMAINE. Le mandat borne les positions a 512 bits, mais les payloads
	// mesurent de 90 a 2575 octets : un negatif sur 64 octets de tete est un negatif etroit.
	// La mesure est donc rejouee sur 1024 bits. Le controle par translation etant un maximum
	// sur TOUTES les positions balayees, il se durcit de lui-meme quand le domaine s'elargit :
	// elargir ne peut pas fabriquer un faux positif, seulement en reveler l'absence plus loin.
	large, courtsL := ondeCollecte(dir, t0, t1, ondeOctetLarge)
	t.Logf("ELARGISSEMENT — %d paquets portent au moins %d octets (%d ecartes, soit %.1f %% du"+
		" flux) : le balayage passe a 1024 positions de bit", len(large), ondeOctetLarge,
		courtsL, 100*float64(courtsL)/float64(len(large)+courtsL))
	if len(large) > 0 {
		for _, v := range []ondeVar{
			{"C2 elargi 1024 bits toutes tetes / debut de payload", nil, false},
			{"C2bis elargi 1024 bits toutes tetes / FIN de payload", nil, true},
		} {
			ondePasse(t, v, large, ondes)
		}
	}
	t.Logf("LOT C — termine en %s", time.Since(debut).Round(time.Second))
}
