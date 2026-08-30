package replay

// visee_composant_verdict_test.go — LOT F : LA MESURE, LE CONTROLE, LE VERDICT.
//
// L'ORDRE DE PUBLICATION EST UNE DECISION DE METHODE, pas une mise en page. La COUVERTURE de la
// marche sort AVANT tout score : un negatif obtenu sur 30 % du record ne serait pas un negatif
// sur le record, et le lecteur doit savoir sur quelle fraction du materiau porte la conclusion
// avant d'en lire une. Vient ensuite la mesure, puis le controle par translation — jamais
// l'inverse : un score sans son controle n'est pas un resultat, c'est un chiffre.
//
// LES SEUILS SONT DANS L'EN-TETE DE `visee_composant_research_test.go`, ecrits avant la premiere
// execution. Ce fichier ne fait que les appliquer.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
)

// vfCalibChunks : nombre de chunks deroules par largeur candidate pendant la calibration.
const vfCalibChunks = 3

// vfTopPublies : nombre de couples (composant, offset) publies au classement.
const vfTopPublies = 12

// vfMesure porte tout ce qu'une variante d'onde produit.
type vfMesure struct {
	nom       string
	onde      ondeCarree
	obs       vfPointe
	dureeZoom int64
	// Les trois parts du controle. p(max GLOBAL) fait foi (S4).
	pMaxGlobal, pMaxComp, pPos float64
	// Puissance (S5) : parts des decalages temoins atteignant 1,0000 et 0,95.
	puiss100, puiss095 float64
	// Decalages temoins retenus / ecartes faute d'echantillons (S6).
	retenus, ecartes int
	// moyenne des meilleurs scores sous onde translatee : la mesure de l'autocorrelation.
	moyTemoin float64
}

// TestViseeComposantOffsetVariable — LE DERNIER CANAL.
func TestViseeComposantOffsetVariable(t *testing.T) {
	dir := os.Getenv(vfFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", vfFilmEnv)
	}
	if filepath.Base(dir) != "00162144" {
		t.Fatalf("la chronologie relevee est celle de 00162144 ; film fourni : %s", filepath.Base(dir))
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pont := vfPontPublie(t, dir, chronoGT)
	src := vfSourcePubliee(t, dir)
	vfSequentielPublie(t, dir, pont.slots)

	debut := time.Now()
	recs, st, err := vfCollecte(dir, src, pont, 0)
	if err != nil {
		t.Fatalf("collecte : %v", err)
	}
	t.Logf("COUT — %d paquets delta deroules en %s, %d records de %s retenus (%d ecartes hors"+
		" vie nommee, %d paquets ecartes pour desappariement hook/records)",
		st.paquets, time.Since(debut).Round(time.Second), len(recs), chronoGT,
		st.horsVie, st.desappaires)
	vfPublieCouverture(t, st)
	if len(recs) == 0 {
		t.Fatalf("aucun record cible : la mesure n'a pas de matiere (voir la couverture ci-dessus)")
	}

	cols := vfBatColonnes(recs, vfCompEchMin)
	vfPublieColonnes(t, cols)
	if len(cols) == 0 {
		t.Fatalf("aucun composant n'atteint %d records : mesure impossible (S6)", vfCompEchMin)
	}
	for _, d := range vfDomainesDe(cols) {
		sub := vfFiltre(cols, d)
		if len(sub) == 0 {
			t.Logf("=== DOMAINE %s — aucun composant recevable, domaine saute.", d.nom)
			continue
		}
		for _, garde := range []int64{ondeGardeMS, ondeGardeBrefMS} {
			m := vfMesureVariante(fmt.Sprintf("%s · garde %d ms", d.nom, garde), sub, garde)
			vfPublieFenetre(t, recs, m.onde)
			vfPublieMesure(t, m, sub)
		}
	}
}

// vfPontPublie construit le pont et publie ce qui permet d'en juger.
func vfPontPublie(t *testing.T, dir, gt string) vfPont {
	t.Helper()
	p, err := vfBatPont(dir, gt)
	if err != nil {
		t.Fatalf("pont : %v", err)
	}
	t.Logf("PONT — %s xuid=%d · %d positions bipedes · decalage feed->film mesure %d ms"+
		" (%d fins de vie appariees) · %d vies NOMMEES, slots %v",
		gt, p.xuid, p.positions, p.offMS, p.apparies, len(p.vies), vfSlotsTries(p.slots))
	for _, l := range p.vies {
		t.Logf("    vie nommee : slot %d, film [%.1f ; %.1f] s", l.slot,
			float64(l.from)/1e6, float64(l.to)/1e6)
	}
	t.Logf("  MORTS de %s sur l'horloge du film (ms) : %v", gt, p.morts)
	t.Logf("  RATTACHEMENT DES FRAGMENTS ANONYMES (double critere mort + reapparition," +
		" unicite exigee) — fragments recouvrant la fenetre d'analyse :")
	for _, l := range p.anonymes {
		vfLogRattachement(t, p, l)
	}
	if p.offMS != ondeOffsetMS {
		t.Logf("  ATTENTION — le decalage mesure (%d ms) differe de la dependance figee du dossier"+
			" (%d ms), ecart %d ms. La chronologie est exprimee dans la seconde : c'est ELLE qui"+
			" sert, et cet ecart est publie pour que le lecteur le pese.",
			p.offMS, ondeOffsetMS, p.offMS-ondeOffsetMS)
	}
	return p
}

// vfLogRattachement publie le sort d'UN fragment anonyme : rattache a qui, avec quelles marges,
// ou laisse anonyme et pourquoi.
func vfLogRattachement(t *testing.T, p vfPont, l lifeSpan) {
	t.Helper()
	for _, r := range p.rattaches {
		if r.slot != l.slot || r.from != l.from {
			continue
		}
		if r.candidats == 1 {
			t.Logf("    slot %d [%.1f ; %.1f] s -> xuid %d (%d candidat ; mort a %+d ms de la fin"+
				" du fragment, reapparition %+d ms apres)", l.slot, float64(l.from)/1e6,
				float64(l.to)/1e6, r.xuid, r.candidats, r.ecartMort, r.ecartRespawn)
			return
		}
		t.Logf("    slot %d [%.1f ; %.1f] s -> NON RATTACHE (%d candidats : l'unicite n'est pas"+
			" satisfaite, le fragment reste anonyme)", l.slot, float64(l.from)/1e6,
			float64(l.to)/1e6, r.candidats)
		return
	}
}

// vfSlotsTries rend les slots dans l'ordre, pour une publication stable.
func vfSlotsTries(m map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// vfSourcePubliee lit le decoupage d'i0 et l'archetype bipede, et publie de quoi juger.
func vfSourcePubliee(t *testing.T, dir string) vfSource {
	t.Helper()
	s, err := vfOuvre(dir)
	if err != nil {
		t.Fatalf("ouverture du film : %v", err)
	}
	t.Logf("SOURCE — decoupage d'i0 %s · registre %d blocs · archetype bipede (ti=%d) %d composants",
		s.lay.String(), s.blocs, filmdec.BipedTypeIndex, len(s.arch.Components))
	return s
}

// vfSequentielPublie republie le rendement du chemin NON ANCRE. Ce n'est plus un choix, c'est
// une piece : le lot F a commence par ce chemin et l'a mesure vide.
func vfSequentielPublie(t *testing.T, dir string, cibles map[uint32]bool) {
	t.Helper()
	t.Logf("CHEMIN SEQUENTIEL (ecarte, republie comme piece) — %d premiers chunks :", vfCalibChunks)
	for _, l := range vfSequentiel(dir, cibles, vfCalibChunks) {
		t.Logf("    IDLowBits=%2d : %7d records bipedes · %6d sur les slots cibles · %6d traverses"+
			" en entier", l.idLowBits, l.bipeds, l.cibles, l.entiers)
	}
}

// vfPublieCouverture publie la reponse au mandat F5 — AVANT tout score.
func vfPublieCouverture(t *testing.T, st vfStat) {
	t.Helper()
	t.Logf("COUVERTURE DE LA MARCHE (F5) — %d records bipedes ancres sur les slots cibles, %d"+
		" marches ; %d traverses EN ENTIER (%.2f %%), %d arretes avant la fin du masque",
		st.bipeds, st.cibles, st.entiers, 100*attPart(st.entiers, st.cibles), st.desync)
	t.Logf("  composants : %d annonces par les masques, %d consommes (%.2f %%) ; part moyenne"+
		" consommee par record %.2f %% ; rang moyen du dernier composant consomme i%.1f",
		st.annonces, st.consommes, 100*attPart(st.consommes, st.annonces),
		100*vfMoyenne(st.partSomme, st.partN), vfMoyenne(st.rangSomme, st.rangN))
	type ligne struct {
		idx, n int
	}
	var arrets []ligne
	for i, n := range st.arret {
		arrets = append(arrets, ligne{i, n})
	}
	sort.Slice(arrets, func(i, j int) bool { return arrets[i].n > arrets[j].n })
	for i, a := range arrets {
		if i == 8 {
			break
		}
		if a.idx < 0 {
			t.Logf("    traversee complete : %d records", a.n)
			continue
		}
		t.Logf("    arret sur i%-2d (%s) : %d records", a.idx, st.noms[a.idx], a.n)
	}
}

// vfPublieFenetre publie ce que la fenetre d'analyse contient, AVANT tout score.
func vfPublieFenetre(t *testing.T, recs []vfRecord, o ondeCarree) {
	t.Helper()
	dans, un, zero, garde, t0, t1 := vfFenetre(recs, o)
	t.Logf("FENETRE — records du joueur sur [%d ; %d] ms de film ; %d tombent dans la fenetre"+
		" d'analyse [%d ; %d] : %d classes « zoome », %d « pas zoome », %d en bande de garde",
		t0, t1, dans, int64(ondeFeneDebutMS), int64(ondeFeneFinMS), un, zero, garde)
}

// vfMoyenne rend une moyenne, 0 quand l'effectif est nul.
func vfMoyenne(somme float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return somme / float64(n)
}

// vfPublieColonnes publie ce que la marche a effectivement rendu mesurable.
func vfPublieColonnes(t *testing.T, cols []vfColonne) {
	t.Helper()
	total := 0
	for _, c := range cols {
		total += c.offsets
	}
	t.Logf("DOMAINE MESURE — %d composants recevables (>= %d records atteints), %d couples"+
		" (composant, offset relatif) balayes :", len(cols), vfCompEchMin, total)
	for _, c := range cols {
		t.Logf("    i%-2d %-52s %6d records · largeur %d..%d bits · %d offsets balayes",
			c.idx, c.nom, c.n, c.largMin, c.largMax, c.offsets)
	}
}

// vfMesureVariante applique l'onde a toutes les colonnes et deroule le controle par translation.
func vfMesureVariante(nom string, cols []vfColonne, garde int64) vfMesure {
	m := vfMesure{nom: nom, onde: ondeConstruit(chronoGT, chronoEpisodes, garde)}
	m.dureeZoom = m.onde.dureeClasse1()
	// LE MEME FILTRE DE RECEVABILITE A DECALAGE NUL ET AUX DECALAGES TEMOINS : sans cela le
	// domaine d'hypotheses observe ne serait pas celui que le controle explore, et p(max)
	// comparerait deux choses differentes.
	m.obs = vfBalaye(cols, m.onde, 0, vfCtrlEchMin)
	if m.obs.comp < 0 {
		return m
	}
	var nMax, nComp, nPos, n100, n095 int
	var somme float64
	for d := -int64(ondeCtrlAmplMS); d <= ondeCtrlAmplMS; d += ondeCtrlPasMS {
		if d > -ondeCtrlGardeMS && d < ondeCtrlGardeMS {
			continue
		}
		p := vfBalaye(cols, m.onde, d, vfCtrlEchMin)
		if p.retenus == 0 || p.comp < 0 {
			m.ecartes++
			continue
		}
		m.retenus++
		somme += p.score.score
		if p.score.score >= m.obs.score.score {
			nMax++
		}
		if p.score.score >= 1 {
			n100++
		}
		if p.score.score >= 0.95 {
			n095++
		}
		sc, sp := vfScoresDuCandidat(cols, m.obs, m.onde, d)
		if sc >= m.obs.score.score {
			nComp++
		}
		if sp >= m.obs.score.score {
			nPos++
		}
	}
	if m.retenus > 0 {
		m.pMaxGlobal = float64(nMax) / float64(m.retenus)
		m.pMaxComp = float64(nComp) / float64(m.retenus)
		m.pPos = float64(nPos) / float64(m.retenus)
		m.puiss100 = float64(n100) / float64(m.retenus)
		m.puiss095 = float64(n095) / float64(m.retenus)
		m.moyTemoin = somme / float64(m.retenus)
	}
	return m
}

// vfScoresDuCandidat rend, pour un decalage temoin, le meilleur score du COMPOSANT candidat et
// celui de sa POSITION exacte — les deux controles plus permissifs, publies pour comparaison.
func vfScoresDuCandidat(cols []vfColonne, obs vfPointe, o ondeCarree, delta int64) (float64, float64) {
	for _, c := range cols {
		if c.idx != obs.comp {
			continue
		}
		m := c.col.marque(o, delta)
		if m.n1 < vfCtrlEchMin || m.n0 < vfCtrlEchMin {
			return 0, 0
		}
		return vfMeilleur(c.col, m).score, c.col.ondeScorePos(obs.score.pos, m)
	}
	return 0, 0
}

// vfPublieMesure publie une variante : classement, controle, puissance, verdict.
func vfPublieMesure(t *testing.T, m vfMesure, cols []vfColonne) {
	t.Helper()
	t.Logf("=== VARIANTE %s — %d ms effectivement classes « zoome » sur la fenetre", m.nom, m.dureeZoom)
	if m.obs.comp < 0 {
		t.Logf("  AUCUNE mesure : pas un seul composant ne porte les deux classes sur cette variante.")
		return
	}
	t.Logf("  MEILLEUR couple observe : i%d (%s) offset relatif %d, polarite %+d — exactitude"+
		" equilibree %.4f (%d echantillons zoome / %d pas zoome)",
		m.obs.comp, m.obs.nom, m.obs.score.pos, m.obs.score.polarite, m.obs.score.score,
		m.obs.n1, m.obs.n0)
	vfPublieClassement(t, cols, m.onde)
	t.Logf("  CONTROLE PAR TRANSLATION — %d decalages retenus, %d ecartes faute d'echantillons"+
		" (S6) ; meilleur score moyen sous onde translatee %.4f",
		m.retenus, m.ecartes, m.moyTemoin)
	t.Logf("    p(max GLOBAL) = %.2f %%   <- fait foi (S4)", 100*m.pMaxGlobal)
	t.Logf("    p(max composant) = %.2f %% · p(position) = %.2f %%   (publies pour comparaison"+
		" seulement : le lot C a rattrape un faux positif a p(position) = 0,19 %%)",
		100*m.pMaxComp, 100*m.pPos)
	t.Logf("  PUISSANCE (S5) — %.2f %% des decalages temoins atteignent 1,0000 et %.2f %%"+
		" atteignent 0,95 : un canal PARFAIT %s par cet instrument",
		100*m.puiss100, 100*m.puiss095, vfPuissanceLecture(m.puiss100))
	vfVerdict(t, m)
}

// vfPuissanceLecture met la puissance en mots, sans la maquiller.
func vfPuissanceLecture(p100 float64) string {
	if p100 < vfSeuilP {
		return "SERAIT DETECTE"
	}
	return "NE SERAIT PAS distingue du hasard — le negatif qui suit est un aveuglement, pas un resultat"
}

// vfPublieClassement publie les meilleurs couples (composant, offset) a decalage nul.
func vfPublieClassement(t *testing.T, cols []vfColonne, o ondeCarree) {
	t.Helper()
	type ligne struct {
		idx, pos int
		nom      string
		score    float64
		n1, n0   int
	}
	var out []ligne
	for _, c := range cols {
		m := c.col.marque(o, 0)
		if m.n1 < vfCtrlEchMin || m.n0 < vfCtrlEchMin {
			continue // meme recevabilite qu'au controle : sinon on publierait des scores degeneres
		}
		for _, s := range vfClassement(c.col, m) {
			out = append(out, ligne{c.idx, s.pos, c.nom, s.score, m.n1, m.n0})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	top := vfTopPublies
	if vfMonoComposant(cols) {
		top = 3 // un domaine mono-composant n'a pas besoin d'un classement long
	}
	t.Logf("  classement des %d meilleurs couples :", top)
	for i, l := range out {
		if i == top {
			break
		}
		t.Logf("    i%-2d offset %2d : %.4f  (%d/%d ech.) %s", l.idx, l.pos, l.score, l.n1, l.n0, l.nom)
	}
}

// vfVerdict applique les seuils declares, dans l'ordre ou ils ont ete ecrits.
func vfVerdict(t *testing.T, m vfMesure) {
	t.Helper()
	sous := m.obs.n1 < vfEchMinClasse || m.obs.n0 < vfEchMinClasse
	if sous {
		t.Logf("  S3 — SOUS-DIMENSIONNE (%d / %d echantillons, seuil %d par classe) : le score"+
			" n'est pas publiable comme candidat, le verdict repose sur le seul controle.",
			m.obs.n1, m.obs.n0, vfEchMinClasse)
	}
	switch {
	case m.puiss100 >= vfSeuilP:
		// LA PUISSANCE PASSE AVANT LE VERDICT. Si un canal PARFAIT ne se distinguerait pas du
		// hasard sur ce domaine, alors « rien trouve » ne veut rien dire : l'instrument est
		// aveugle ici, et le dire est le seul resultat honnete.
		t.Logf("  VERDICT : NON CONCLUANT — la puissance manque (%.2f %% des decalages temoins"+
			" atteignent 1,0000, seuil %.0f %%). Le score observe %.4f et p(max global) %.2f %%"+
			" sont publies, mais AUCUN negatif ne peut etre tire de ce domaine.",
			100*m.puiss100, 100*vfSeuilP, m.obs.score.score, 100*m.pMaxGlobal)
	case m.pMaxGlobal >= vfSeuilP:
		t.Logf("  VERDICT : NEGATIF (adosse a une puissance mesuree) — %.2f %% des decalages"+
			" temoins font aussi bien que le meilleur couple observe (seuil %.0f %%), et un canal"+
			" parfait SERAIT detecte ici. Aucun bit a offset relatif fixe dans la charge utile de"+
			" ce domaine ne porte l'etat de lunette.", 100*m.pMaxGlobal, 100*vfSeuilP)
	case m.obs.score.score >= vfSeuilCand && !sous:
		t.Logf("  VERDICT : CANDIDAT (S1) — i%d offset %d, %.4f, p(max global) %.2f %%."+
			" Contre-verification exigee (F4).", m.obs.comp, m.obs.score.pos,
			m.obs.score.score, 100*m.pMaxGlobal)
	case m.obs.score.score >= vfSeuilSuivre:
		t.Logf("  VERDICT : A SUIVRE (S2) — i%d offset %d, %.4f, p(max global) %.2f %%.",
			m.obs.comp, m.obs.score.pos, m.obs.score.score, 100*m.pMaxGlobal)
	default:
		t.Logf("  VERDICT : NEGATIF — le controle passe (p(max global) %.2f %%) mais le score"+
			" observe %.4f reste sous le seuil « a suivre » %.2f.",
			100*m.pMaxGlobal, m.obs.score.score, vfSeuilSuivre)
	}
}

// vfEchMinClasse : le volet « echantillons par classe » de S1/S2, meme valeur qu'au lot C.
const vfEchMinClasse = ondeEchMin
