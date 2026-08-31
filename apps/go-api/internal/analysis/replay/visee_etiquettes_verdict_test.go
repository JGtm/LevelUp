package replay

// visee_etiquettes_verdict_test.go — LOT G : LA MESURE, LE CONTROLE, LE VERDICT.
//
// L'ORDRE DE PUBLICATION EST UNE DECISION DE METHODE, pas une mise en page, et c'est celui du
// lot F : d'abord les ETIQUETTES et leurs effectifs (sans quoi aucun score n'a de sens), puis la
// COUVERTURE de la marche (un negatif obtenu sur 30 % du record ne serait pas un negatif sur le
// record), puis seulement la mesure et son controle. Jamais un score sans son controle.
//
// LES SEUILS SONT CEUX DU LOT F, REPRIS ET NON REDEFINIS (`vfSeuilCand`, `vfSeuilSuivre`,
// `vfSeuilP`, `vfCompEchMin`, `vfCtrlEchMin`, `vfEchMinClasse`), et l'echelle de verdict est
// litteralement la fonction `vfVerdict`. C'est ce qui rend les deux lots comparables : entre F
// et G, seule la SOURCE DES ETIQUETTES change.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
)

// vgIdxSentinelle : index d archetype fictif de la colonne sentinelle. POSITIF a dessein :
// `vfPointe.comp` vaut -1 pour dire « aucune mesure », donc un index negatif ferait passer une
// sentinelle parfaite pour une absence de mesure.
const vgIdxSentinelle = 9999

// vgTopPublies : nombre de couples (composant, offset) publies au classement.
const vgTopPublies = 12

// TestViseeEtiquettes — LE CHAMP D'ETAT DE LUNETTE, CHERCHE AVEC DES MILLIERS D'ETIQUETTES.
func TestViseeEtiquettes(t *testing.T) {
	dir := os.Getenv(vgFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", vgFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	debut := time.Now()
	g, err := vgBatGrille(dir)
	if err != nil {
		t.Fatalf("etiquettes : %v", err)
	}
	vgPublieGrille(t, filepath.Base(dir), g, time.Since(debut))

	src := vfSourcePubliee(t, dir)
	cibles := g.cibles()
	deltas := vgDecalages(g)
	t.Logf("CONTROLE — translation CIRCULAIRE sur %.1f s : %d decalages temoins, pas %d ms,"+
		" voisinage +/- %d ms du decalage nul exclu",
		float64(g.dureeMS())/1000, len(deltas), vgPasDecalage(g), vgCtrlGardeMS)

	vgVoletDelta(t, dir, src, g, cibles, deltas)
	vgVoletKeyframe(t, dir, g, cibles, deltas)
}

// vgVoletDelta deroule G2 : collecte delta, couverture, domaines, verdicts.
func vgVoletDelta(t *testing.T, dir string, src vfSource, g *vgGrille,
	cibles map[uint32]bool, deltas []int64,
) {
	t.Logf("========== G2 — RECORDS DELTA ==========")
	debut := time.Now()
	recs, st := vgCollecte(dir, src, cibles)
	t.Logf("COUT — %d paquets delta deroules en %s, %d records retenus (%d paquets ecartes pour"+
		" desappariement hook/records)", st.paquets, time.Since(debut).Round(time.Second),
		len(recs), st.desappaires)
	vfPublieCouverture(t, st)
	if len(recs) == 0 {
		t.Fatalf("aucun record delta : la mesure n'a pas de matiere")
	}
	un, zero, exclu := vgClasses(recs, g)
	t.Logf("EFFECTIFS PAR CLASSE (records delta, decalage nul) — %d « zoome », %d « pas zoome »,"+
		" %d exclus par les marges (%.2f %% de la classe zoome dans le total classe)",
		un, zero, exclu, 100*attPart(un, un+zero))
	vgMesureVolet(t, "DELTA", recs, g, deltas)
}

// vgVoletKeyframe deroule G3 : collecte d'image-cle, sous-dimensionnement eventuel, verdicts.
func vgVoletKeyframe(t *testing.T, dir string, g *vgGrille, cibles map[uint32]bool, deltas []int64) {
	t.Logf("========== G3 — IMAGES-CLES ==========")
	brut, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		t.Logf("  registre (chunk_00) illisible : %v — volet image-cle impossible", err)
		return
	}
	reg, err := filmdec.ParseRegistryChunk(brut)
	if err != nil {
		t.Logf("  registre illisible : %v — volet image-cle impossible", err)
		return
	}
	debut := time.Now()
	recs, st := vgCollecteKF(dir, reg, cibles)
	vgPublieKF(t, st, time.Since(debut))
	if len(recs) == 0 {
		t.Logf("  SOUS-DIMENSIONNE — aucun record bipede d'image-cle sur les slots etiquetes :"+
			" ce volet ne mesure RIEN, et aucun negatif n'en est tirable (%d paquets, %d records"+
			" bipedes vus)", st.paquets, st.bipeds)
		return
	}
	un, zero, exclu := vgClasses(recs, g)
	t.Logf("EFFECTIFS PAR CLASSE (records d'image-cle, decalage nul) — %d « zoome », %d « pas"+
		" zoome », %d exclus par les marges", un, zero, exclu)
	if un < vfCtrlEchMin || zero < vfCtrlEchMin {
		t.Logf("  SOUS-DIMENSIONNE (S6) — moins de %d echantillons dans une classe : les"+
			" images-cles sont trop espacees sur ce film pour porter un verdict. C'est le"+
			" resultat, publie tel quel plutot que transforme en faux negatif.", vfCtrlEchMin)
		t.Logf("  CE QU'IL FAUDRAIT — a ce rendement (%d records zoomes par film), il en faut"+
			" ~%d films pour atteindre le seuil de recevabilite (%d) et ~%d pour le seuil de"+
			" candidature (%d). Le volet image-cle n'est donc PAS testable sur un film :"+
			" c'est une mesure de dimensionnement, pas un negatif.",
			un, vgFilmsRequis(un, vfCtrlEchMin), vfCtrlEchMin,
			vgFilmsRequis(un, vfEchMinClasse), vfEchMinClasse)
		return
	}
	vgMesureVolet(t, "IMAGE-CLE", recs, g, deltas)
}

// vgFilmsRequis rend le nombre de films qu'il faudrait, au rendement observe, pour atteindre un
// effectif de classe donne. Un sous-dimensionnement qui ne dit pas de combien il manque
// n'oriente aucune decision — celui-ci la chiffre.
func vgFilmsRequis(parFilm, vise int) int {
	if parFilm <= 0 {
		return -1
	}
	return (vise + parFilm - 1) / parFilm
}

// vgMesureVolet construit les colonnes d'un volet, deroule tous les domaines, et termine par la
// SENTINELLE — le controle de validite de la chaine entiere.
func vgMesureVolet(t *testing.T, volet string, recs []vfRecord, g *vgGrille, deltas []int64) {
	cols := vgBatColonnes(recs, vfCompEchMin)
	vgPublieColonnes(t, cols)
	if len(cols) == 0 {
		t.Logf("  aucun composant n'atteint %d records : mesure impossible (S6)", vfCompEchMin)
		return
	}
	vgPasseSentinelle(t, volet, cols, g, deltas)
	for _, d := range vgDomainesDe(cols) {
		sub := vgFiltre(cols, d)
		if len(sub) == 0 {
			continue
		}
		m, ctrl := vgMesureDomaine(fmt.Sprintf("%s · %s", volet, d.nom), sub, g, deltas)
		vgPublieMesure(t, m, ctrl, sub, g)
	}
}

// LA SENTINELLE — CE QUI SEPARE UN NEGATIF D'UNE PANNE SILENCIEUSE.
//
// Un negatif ne vaut que si l'instrument aurait su trouver. La puissance (S5) repond a cette
// question par extrapolation — quelle part des temoins atteint 1,0000 ? La sentinelle y repond
// PAR CONSTRUCTION : on fabrique un canal PARFAIT (un bit egal a l'etiquette, injecte dans une
// colonne au meme format que les autres) et on le fait passer par la CHAINE ENTIERE — meme
// transposition, meme masque, meme score, meme controle par translation.
//
// CE QU'ELLE ATTRAPE, ET QU'AUCUNE AUTRE MESURE N'ATTRAPE : un desalignement entre les records
// et les etiquettes (slots decales, horloges differentes, appariement rompu) ferait tomber tous
// les scores a 0,5 et produirait un negatif PARFAITEMENT convaincant et entierement faux. Si la
// sentinelle sort a 1,0000 avec p(max) nul, ce mode de panne est exclu.
func vgPasseSentinelle(t *testing.T, volet string, cols []vgColonne, g *vgGrille, deltas []int64) {
	t.Helper()
	sen := []vgColonne{vgSentinelle(cols, g)}
	m, ctrl := vgMesureDomaine(volet+" · SENTINELLE (canal parfait injecte)", sen, g, deltas)
	vgPublieMesure(t, m, ctrl, sen, g)
	if m.obs.comp < 0 || m.obs.score.score < 1 {
		t.Errorf("SENTINELLE — la chaine ne retrouve PAS un canal parfait (score %.4f) : tout"+
			" negatif de ce volet est suspect", m.obs.score.score)
	}
}

// vgSentinelle fabrique la colonne du canal parfait : un unique bit valant l'etiquette de
// l'echantillon, porte par le composant le plus fourni (donc au meme nombre d'echantillons et
// aux memes instants que la mesure reelle).
func vgSentinelle(cols []vgColonne, g *vgGrille) vgColonne {
	src := cols[0]
	for _, c := range cols {
		if c.n > src.n {
			src = c
		}
	}
	col := &ondeCol{temps: src.col.temps, mots: src.col.mots, nbits: 1}
	col.col = [][]uint64{make([]uint64, src.col.mots)}
	for i, ts := range col.temps {
		if g.classe(src.slots[i], ts, 0) == 1 {
			col.col[0][i/64] |= 1 << uint(i%64)
		}
	}
	return vgColonne{idx: vgIdxSentinelle, nom: "canal parfait injecte", col: col, slots: src.slots,
		n: src.n, offsets: 1, largMin: 1, largMax: 1}
}

// vgPasDecalage rend le pas du controle : la duree du film divisee par le nombre vise de
// decalages, avec un plancher — deux decalages plus proches que le plancher mesureraient deux
// fois le meme instant sans ajouter d'information.
func vgPasDecalage(g *vgGrille) int64 {
	pas := g.dureeMS() / vgCtrlDecalages
	if pas < vgCtrlPasMinMS {
		pas = vgCtrlPasMinMS
	}
	return pas
}

// vgDecalages rend les decalages temoins. La translation etant CIRCULAIRE, les decalages
// negatifs sont les positifs complementes : un seul balayage de 0 a la duree totale suffit, et
// le voisinage des deux extremites (qui est le meme point) est exclu.
func vgDecalages(g *vgGrille) []int64 {
	pas, total := vgPasDecalage(g), g.dureeMS()
	var out []int64
	for d := pas; d < total; d += pas {
		if d < vgCtrlGardeMS || total-d < vgCtrlGardeMS {
			continue
		}
		out = append(out, d)
	}
	return out
}

// vgCtrl porte le DIAGNOSTIC du controle : la taille des classes que les decalages temoins
// portent reellement.
//
// POURQUOI CE DIAGNOSTIC EXISTE, ET IL N'ETAIT PAS PREVU. La premiere execution a montre que le
// meilleur score MOYEN sous etiquettes translatees depasse le score observe (0,76 contre 0,65
// sur le domaine complet). La cause est mesurable et doit se lire : un slot n'a de records que
// pendant SES vies, donc translater ses periodes de lunette les envoie souvent la ou il n'existe
// pas — la classe « zoome » d'un decalage temoin est alors bien plus petite que celle du
// decalage nul, et un maximum sur petit echantillon monte plus haut. Consequence sur la lecture :
// p(max) est CONSERVATEUR (il refuse plus facilement), ce qui ne fragilise pas un negatif mais
// affaiblirait la detection d'un positif. Le publier est le minimum ; la sentinelle, elle,
// verifie qu'un positif franc passe malgre cela.
type vgCtrl struct {
	n1Obs           int
	n1Min, n1Med    int
	n1Max, nRetenus int
}

// vgMesureDomaine applique les etiquettes a un domaine et deroule le controle par translation.
func vgMesureDomaine(nom string, cols []vgColonne, g *vgGrille, deltas []int64) (vfMesure, vgCtrl) {
	m := vfMesure{nom: nom}
	m.obs = vgBalaye(cols, g, 0, vfCtrlEchMin)
	if m.obs.comp < 0 {
		return m, vgCtrl{}
	}
	var nMax, nComp, nPos, n100, n095 int
	var somme float64
	var tailles []int
	for _, d := range deltas {
		p := vgBalaye(cols, g, d, vfCtrlEchMin)
		if p.retenus == 0 || p.comp < 0 {
			m.ecartes++
			continue
		}
		m.retenus++
		somme += p.score.score
		tailles = append(tailles, p.n1)
		if p.score.score >= m.obs.score.score {
			nMax++
		}
		if p.score.score >= 1 {
			n100++
		}
		if p.score.score >= vfSeuilCand {
			n095++
		}
		sc, sp := vgScoresDuCandidat(cols, m.obs, g, d)
		if sc >= m.obs.score.score {
			nComp++
		}
		if sp >= m.obs.score.score {
			nPos++
		}
	}
	vgParts(&m, nMax, nComp, nPos, n100, n095, somme)
	return m, vgDiagCtrl(m.obs.n1, tailles)
}

// vgDiagCtrl resume les tailles de classe « zoome » portees par les decalages temoins.
func vgDiagCtrl(n1Obs int, tailles []int) vgCtrl {
	c := vgCtrl{n1Obs: n1Obs, nRetenus: len(tailles)}
	if len(tailles) == 0 {
		return c
	}
	sort.Ints(tailles)
	c.n1Min, c.n1Max = tailles[0], tailles[len(tailles)-1]
	c.n1Med = tailles[len(tailles)/2]
	return c
}

// vgParts convertit les compteurs du controle en parts publiables.
func vgParts(m *vfMesure, nMax, nComp, nPos, n100, n095 int, somme float64) {
	if m.retenus == 0 {
		return
	}
	d := float64(m.retenus)
	m.pMaxGlobal, m.pMaxComp, m.pPos = float64(nMax)/d, float64(nComp)/d, float64(nPos)/d
	m.puiss100, m.puiss095 = float64(n100)/d, float64(n095)/d
	m.moyTemoin = somme / d
}

// vgScoresDuCandidat rend, pour un decalage temoin, le meilleur score du COMPOSANT candidat et
// celui de sa POSITION exacte — les deux controles plus permissifs, publies pour comparaison.
func vgScoresDuCandidat(cols []vgColonne, obs vfPointe, g *vgGrille, delta int64) (float64, float64) {
	for i := range cols {
		if cols[i].idx != obs.comp {
			continue
		}
		m := vgMarque(&cols[i], g, delta)
		if m.n1 < vfCtrlEchMin || m.n0 < vfCtrlEchMin {
			return 0, 0
		}
		return vfMeilleur(cols[i].col, m).score, cols[i].col.ondeScorePos(obs.score.pos, m)
	}
	return 0, 0
}

// vgPublieGrille publie G1 : ce que les etiquettes contiennent, avant toute mesure.
func vgPublieGrille(t *testing.T, film string, g *vgGrille, cout time.Duration) {
	t.Helper()
	t.Logf("========== G1 — ETIQUETTES (film %s, %s) ==========", film, cout.Round(time.Second))
	t.Logf("EVENEMENTS — %d bascules unit_zoom lues : %d entrees, %d sorties",
		g.evts, g.entrees, g.sorties)
	t.Logf("GRILLE — %d cellules de %d ms sur [%d ; %d] ms de film (%.1f s) ; %d slots retenus"+
		" (au moins une periode ET les deux classes), %d periodes reconstruites",
		g.n, g.pas, g.t0, g.t0+g.dureeMS(), float64(g.dureeMS())/1000, len(g.slots), g.periodes)
	t.Logf("CELLULES PAR CLASSE — %d « zoome » (marge %d ms), %d « pas zoome » (marge %d ms),"+
		" %d exclues ; la classe zoome pese %.2f %% du materiau classe",
		g.cell1, vgMargeDedansMS, g.cell0, vgMargeDehorsMS, g.cellX,
		100*attPart(g.cell1, g.cell1+g.cell0))
	t.Logf("  slots etiquetes : %v", g.slots)
}

// vgPublieKF publie la couverture des deux marches d'image-cle.
func vgPublieKF(t *testing.T, st vgKFStat, cout time.Duration) {
	t.Helper()
	t.Logf("COUVERTURE DE LA MARCHE D'IMAGE-CLE (%s) — %d paquets d'image-cle ; balayeur de"+
		" production : %d records ancres (%d..%d par paquet), %d d'archetype bipede",
		cout.Round(time.Second), st.paquets, st.records, st.recMin, st.recMax, st.bipeds)
	t.Logf("  MARCHEUR DETERMINISTE (publie comme piece, pas utilise) — %d records enchaines,"+
		" %d bipedes ; causes d'arret : %v", st.detRecords, st.detBipeds, st.detArrets)
	t.Logf("  traversee des corps : %d records mesures, %d composants, %d desynchronises,"+
		" %d ECARTES pour debordement sur l'ancre suivante",
		st.retenus, st.comps, st.desync, st.deborde)
}

// vgPublieColonnes publie ce que la marche a rendu mesurable.
func vgPublieColonnes(t *testing.T, cols []vgColonne) {
	t.Helper()
	total := 0
	for _, c := range cols {
		total += c.offsets
	}
	t.Logf("DOMAINE MESURE — %d composants recevables (>= %d records atteints), %d couples"+
		" (composant, offset relatif) balayes :", len(cols), vfCompEchMin, total)
	for _, c := range cols {
		t.Logf("    i%-2d %-52s %7d records · largeur %d..%d bits · %d offsets balayes",
			c.idx, c.nom, c.n, c.largMin, c.largMax, c.offsets)
	}
}

// vgPublieMesure publie une variante : meilleur couple, classement, controle, puissance, verdict.
func vgPublieMesure(t *testing.T, m vfMesure, ctrl vgCtrl, cols []vgColonne, g *vgGrille) {
	t.Helper()
	t.Logf("=== DOMAINE %s", m.nom)
	if m.obs.comp < 0 {
		t.Logf("  AUCUNE mesure : pas un seul composant ne porte les deux classes au seuil de"+
			" recevabilite (%d echantillons).", vfCtrlEchMin)
		return
	}
	t.Logf("  MEILLEUR couple observe : i%d (%s) offset relatif %d, polarite %+d — exactitude"+
		" equilibree %.4f (%d echantillons zoome / %d pas zoome)",
		m.obs.comp, m.obs.nom, m.obs.score.pos, m.obs.score.polarite, m.obs.score.score,
		m.obs.n1, m.obs.n0)
	vgPublieClassement(t, cols, g)
	t.Logf("  CONTROLE PAR TRANSLATION — %d decalages retenus, %d ecartes faute d'echantillons"+
		" (S6) ; meilleur score moyen sous etiquettes translatees %.4f",
		m.retenus, m.ecartes, m.moyTemoin)
	t.Logf("    taille de la classe « zoome » — %d au decalage nul ; chez les temoins retenus"+
		" %d / %d / %d (min / mediane / max). Un temoin plus petit fait monter son maximum :"+
		" p(max) en est CONSERVATEUR, ce qui durcit un positif et jamais un negatif.",
		ctrl.n1Obs, ctrl.n1Min, ctrl.n1Med, ctrl.n1Max)
	t.Logf("    p(max GLOBAL) = %.2f %%   <- fait foi (S4)", 100*m.pMaxGlobal)
	t.Logf("    p(max composant) = %.2f %% · p(position) = %.2f %%   (publies pour comparaison"+
		" seulement)", 100*m.pMaxComp, 100*m.pPos)
	t.Logf("  PUISSANCE (S5) — %.2f %% des decalages temoins atteignent 1,0000 et %.2f %%"+
		" atteignent 0,95 : un canal PARFAIT %s par cet instrument",
		100*m.puiss100, 100*m.puiss095, vfPuissanceLecture(m.puiss100))
	vfVerdict(t, m)
}

// vgPublieClassement publie les meilleurs couples (composant, offset) a decalage nul.
func vgPublieClassement(t *testing.T, cols []vgColonne, g *vgGrille) {
	t.Helper()
	type ligne struct {
		idx, pos int
		nom      string
		score    float64
		n1, n0   int
	}
	var out []ligne
	for i := range cols {
		m := vgMarque(&cols[i], g, 0)
		if m.n1 < vfCtrlEchMin || m.n0 < vfCtrlEchMin {
			continue
		}
		for _, s := range vfClassement(cols[i].col, m) {
			out = append(out, ligne{cols[i].idx, s.pos, cols[i].nom, s.score, m.n1, m.n0})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	top := vgTopPublies
	if len(cols) == 1 {
		top = 3
	}
	t.Logf("  classement des %d meilleurs couples :", top)
	for i, l := range out {
		if i == top {
			break
		}
		t.Logf("    i%-2d offset %2d : %.4f  (%d/%d ech.) %s", l.idx, l.pos, l.score, l.n1, l.n0,
			l.nom)
	}
}
