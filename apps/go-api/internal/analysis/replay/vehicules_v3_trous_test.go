package replay

// vehicules_v3_trous_test.go — ETAGES 2 et 3 de l'instrument V3 (lot V3, item A), separes de
// vehicules_v3_destruction_test.go pour tenir le seuil de 500 lignes par fichier. Meme test,
// memes gardes d'environnement (V3_DESTR_FILMS / V3_DESTR_ROOT) : ces etages sont appeles depuis
// v3dClasseVie et v3dRapport. Les gates 4 a 8 sont ecrits ci-dessous AVANT leur mesure.

import (
	"sort"
	"testing"
)

// ============================= ETAGE 2 — LE TROU D'EMBARQUEMENT =============================
//
// POURQUOI UN SECOND ETAGE, ET SON GATE ECRIT AVANT DE LE MESURER. L'etage 1 (vehicules_v3_destruction_test.go) exige
// que l'occupant soit encore A BORD a la FIN SERREE du vehicule. Sa mesure rend 0 occupant
// courant sur 37 vies a candidat (6 films) : le flux de position du VEHICULE se poursuit
// systematiquement 5 a 150 s APRES la fermeture du trou de son occupant. Autrement dit, la FIN
// SERREE du vehicule n'est PAS l'instant ou son conducteur le quitte — un vehicule continue de
// repliquer sa position une fois abandonne, et peut etre repris par un autre joueur (la vie
// slot=771 de 0d76e8f1 porte trois occupants successifs). Le postulat « fin de trajectoire =
// destruction » est donc REFUTE, et avec lui l'etage 1 tel qu'ecrit.
//
// L'ETAGE 2 pose la question dans l'autre sens, la seule que les primitives disponibles savent
// trancher : L'OCCUPANT MEURT-IL PENDANT QU'IL EST A BORD ? Le TROU du flux de position [gs, ge]
// est l'intervalle d'embarquement (V1a.4 : l'enfant attache ne replique plus ; V2b : la sortie
// ferme le trou a l'instant exact, 10/10). Une mort de CET occupant DANS son trou est donc une
// mort A BORD — l'evenement que la regle metier « si le vehicule est detruit, en general le
// joueur meurt aussi » designe.
//
// GATE 4 — MORT A BORD AU-DESSUS DU HASARD. La part des trous d'embarquement contenant une mort
//   de LEUR occupant doit depasser de plus de v3dEcartTemoinMin (10 pts) la meme part calculee
//   avec un occupant DECALE (rotation v3dTemoinRotation du roster) sur LE MEME intervalle. Le
//   temoin normalise exactement la duree du trou et la densite de morts : c'est le hasard, et
//   rien d'autre.
// GATE 5 — DATATION. Parmi les trous a mort a bord, la MEDIANE de |t_mort - fin serree du
//   vehicule| doit rester sous v3dPrecisionMedianeMaxMS (5 s) pour que la mort DATE la fin du
//   vehicule. Si elle echoue, la mort a bord reste un fait mesure (temps a bord, cause de fin de
//   trajet) mais ne DATE PAS la destruction, et il faut le dire.
// GATE 6 — LA SORTIE FERME LE TROU (recoupement V2b a l'echelle du corpus). Part des trous dont
//   un evenement de sortie de leur occupant tombe a moins de v3dSortieTolUS de la fermeture ;
//   V2b l'a mesuree a 10/10 sur un film. Ce n'est pas un gate bloquant : c'est le controle qui
//   dit si la lecture des trous est saine sur tout le corpus.

// v3dSortieTolUS : tolerance entre un evenement de sortie et la fermeture du trou de son occupant.
const v3dSortieTolUS = uint64(2_000_000)

// v3dTrouAgg agrege l'etage 2 : un enregistrement par (vie de vehicule, occupant candidat).
type v3dTrouAgg struct {
	trous, fermes             int
	avecSortie, sortieALaFin  int
	avecMort, avecMortTemoin  int
	sansMortNiSortie          int
	ecartMortFinSerree        []int64
	ecartMortFermeture        []int64
	dureesTrouMS              []int64
	ecartFermetureFinSerreeMS []int64
	// bord porte l'ETAGE 3 (la mort au bord du trou).
	bord v3dTrouAgg3
}

// v3dAnalyseTrous parcourt les couples (vie, occupant candidat) d'une vie et alimente l'etage 2.
func v3dAnalyseTrous(v v1cVie, ctx v3dCtx, finUS uint64, tg *v3dTrouAgg) {
	slots := make([]uint32, 0, len(v.Cand))
	for s := range v.Cand {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		v3dUnTrou(v.Cand[s], s, ctx, finUS, tg)
	}
}

// v3dUnTrou classe UN trou d'embarquement.
func v3dUnTrou(gs uint64, slot uint32, ctx v3dCtx, finUS uint64, tg *v3dTrouAgg) {
	tg.trous++
	ge, ferme := v3dFinDuTrou(ctx.ptracks[slot], gs)
	if !ferme {
		ge = ^uint64(0) // trou ouvert jusqu'a la fin du film : l'occupant ne re-emet jamais
	} else {
		tg.fermes++
		tg.dureesTrouMS = append(tg.dureesTrouMS, int64(ge-gs)/1000)
		tg.ecartFermetureFinSerreeMS = append(tg.ecartFermetureFinSerreeMS,
			int64(finUS/1000)-int64(ge/1000))
	}
	sortieUS, aSortie := v3dSortieDans(ctx.sorties[slot], gs, ge)
	if aSortie {
		tg.avecSortie++
		if ferme && attEcartUS(sortieUS, ge) <= v3dSortieTolUS {
			tg.sortieALaFin++
		}
	}
	xuid, connu := ctx.slotX[slot]
	if !connu {
		return
	}
	if ferme {
		v3dMortAuBord(ge, finUS, xuid, ctx, &tg.bord)
	}
	if v3dMortDans(ctx.morts[v3dTemoinXUID(ctx.roster, xuid)], gs, ge) > 0 {
		tg.avecMortTemoin++
	}
	m := v3dMortDans(ctx.morts[xuid], gs, ge)
	if m == 0 {
		if !aSortie {
			tg.sansMortNiSortie++
		}
		return
	}
	tg.avecMort++
	tg.ecartMortFinSerree = append(tg.ecartMortFinSerree, absI64(int64(m/1000)-int64(finUS/1000)))
	if ferme {
		tg.ecartMortFermeture = append(tg.ecartMortFermeture, int64(ge/1000)-int64(m/1000))
	}
}

// v3dMortDans rend la PREMIERE mort de la serie dans [debut, fin], 0 s'il n'y en a pas.
func v3dMortDans(morts []uint64, debut, fin uint64) uint64 {
	for _, m := range morts {
		if m >= debut && m <= fin {
			return m
		}
	}
	return 0
}

// v3dSortieDans rend la PREMIERE sortie de la serie dans [debut, fin].
func v3dSortieDans(ts []uint64, debut, fin uint64) (uint64, bool) {
	for _, t := range ts {
		if t >= debut && t <= fin {
			return t, true
		}
	}
	return 0, false
}

// v3dRapportTrous publie l'etage 2 et verdit les gates 4 a 6.
func v3dRapportTrous(t *testing.T, tg *v3dTrouAgg) {
	t.Helper()
	t.Logf("\n---------- ETAGE 2 — LE TROU D'EMBARQUEMENT (%d trous, %d fermes) ----------",
		tg.trous, tg.fermes)
	t.Logf("  duree du trou (temps a bord) : mediane %d ms · ecart fermeture -> fin serree du vehicule : mediane %d ms",
		v3dMediane(tg.dureesTrouMS), v3dMediane(tg.ecartFermetureFinSerreeMS))
	pm, pt := attPart(tg.avecMort, tg.trous), attPart(tg.avecMortTemoin, tg.trous)
	t.Logf("  MORT A BORD : %d/%d = %.1f %% · TEMOIN (occupant decale, MEME intervalle) %d = %.1f %% · ecart %+.1f pts",
		tg.avecMort, tg.trous, 100*pm, tg.avecMortTemoin, 100*pt, 100*(pm-pt))
	t.Logf("  GATE 4 (mort a bord - temoin > %.0f pts) : %s", 100*v3dEcartTemoinMin,
		v3dVerdictGate(pm-pt > v3dEcartTemoinMin))
	med := v3dMediane(tg.ecartMortFinSerree)
	t.Logf("  GATE 5 (mediane |t_mort - fin serree| <= %d ms) : mediane %d ms sur %d morts a bord — %s",
		v3dPrecisionMedianeMaxMS, med, len(tg.ecartMortFinSerree),
		v3dVerdictGate(len(tg.ecartMortFinSerree) > 0 && med <= v3dPrecisionMedianeMaxMS))
	t.Logf("  delai mort a bord -> fermeture du trou (respawn) : mediane %d ms sur %d",
		v3dMediane(tg.ecartMortFermeture), len(tg.ecartMortFermeture))
	t.Logf("  GATE 6 (controle V2b) : trous avec sortie de leur occupant %d/%d = %.1f %% · dont la sortie ferme le trou (+/-%d ms) %d = %.1f %%",
		tg.avecSortie, tg.trous, 100*attPart(tg.avecSortie, tg.trous), v3dSortieTolUS/1000,
		tg.sortieALaFin, 100*attPart(tg.sortieALaFin, tg.avecSortie))
	v3dRapportBord(t, &tg.bord, tg.fermes)
	t.Logf("  CLASSEMENT DES TROUS : mort a bord %d · sortie sans mort %d · ni mort ni sortie (despawn/abandon) %d",
		tg.avecMort, tg.avecSortie, tg.sansMortNiSortie)
}

// ========================== ETAGE 3 — LA SORTIE EST-ELLE UNE MORT ? ==========================
//
// CE QUE L'ETAGE 2 A MESURE, ET POURQUOI IL APPELLE UN TROISIEME. Sur 30 trous d'embarquement
// (4 films), l'occupant ne meurt JAMAIS a l'interieur de son trou (0 %), alors que le temoin a
// occupant decale y meurt 20 % du temps : la mort a bord est ANTI-correlee, pas absente au
// hasard. La cause est lisible dans le meme releve : 27 trous sur 30 (90 %) portent un evenement
// de SORTIE de leur occupant, et cette sortie ferme le trou a +/-2 s dans 27 cas sur 27 (100 %,
// le recoupement V2b reproduit a l'echelle du corpus). Autrement dit TOUT embarquement se termine
// par une sortie, y compris — hypothese de cet etage — quand l'occupant est TUE : le moteur
// l'ejecte, la sortie est emise, le flux de position reprend (cadavre puis respawn), et la mort
// tombe donc AU BORD du trou, pas DEDANS.
//
// GATE 7 — LA SORTIE PAR LA MORT, ecrit avant la mesure. La part des trous dont l'occupant meurt
//   a moins de v3dFenetreMS de la FERMETURE du trou doit depasser de plus de v3dEcartTemoinMin
//   (10 pts) la meme part pour un occupant DECALE (rotation v3dTemoinRotation, meme instant).
//   Le temoin mesure exactement le hasard « un joueur nomme meurt dans cette fenetre ».
// GATE 8 — CE QUE LA MORT DATE. Si le gate 7 passe, la mort date la SORTIE a la milliseconde. Il
//   reste a savoir si elle date la FIN DU VEHICULE : mediane de |t_mort - fin serree du vehicule|
//   sur les sorties par la mort, a comparer aux +/-20 s du recensement. Seuil
//   v3dPrecisionMedianeMaxMS (5 s).

// v3dTrouAgg3 agrege l'etage 3 : la mort au BORD du trou, par fenetre.
type v3dTrouAgg3 struct {
	mortFermeture, temoinFermeture map[int64]int
	ecartVehicule                  []int64
	ecartMortFermetureSigne        []int64
}

func newV3dTrouAgg3() v3dTrouAgg3 {
	return v3dTrouAgg3{mortFermeture: map[int64]int{}, temoinFermeture: map[int64]int{}}
}

// v3dMortAuBord impute l'etage 3 pour UN trou ferme : la mort de l'occupant tombe-t-elle au bord
// (fermeture) du trou, et que dit-elle de la fin du vehicule ?
func v3dMortAuBord(ge, finUS uint64, xuid uint64, ctx v3dCtx, tg *v3dTrouAgg3) {
	tem := v3dTemoinXUID(ctx.roster, xuid)
	for _, w := range v3dFenetresMS {
		if _, _, ok := v3dMortLaPlusProche(ctx.morts[xuid], ge, w); ok {
			tg.mortFermeture[w]++
		}
		if _, _, ok := v3dMortLaPlusProche(ctx.morts[tem], ge, w); ok {
			tg.temoinFermeture[w]++
		}
	}
	_, mortUS, ok := v3dMortLaPlusProche(ctx.morts[xuid], ge, v3dFenetreMS)
	if !ok {
		return
	}
	tg.ecartVehicule = append(tg.ecartVehicule, absI64(int64(mortUS/1000)-int64(finUS/1000)))
	tg.ecartMortFermetureSigne = append(tg.ecartMortFermetureSigne,
		int64(mortUS/1000)-int64(ge/1000))
}

// v3dRapportBord publie l'etage 3 et verdit les gates 7 et 8.
func v3dRapportBord(t *testing.T, tg *v3dTrouAgg3, trousFermes int) {
	t.Helper()
	t.Logf("\n---------- ETAGE 3 — LA SORTIE EST-ELLE UNE MORT ? (%d trous fermes) ----------", trousFermes)
	for _, w := range v3dFenetresMS {
		pr := attPart(tg.mortFermeture[w], trousFermes)
		pt := attPart(tg.temoinFermeture[w], trousFermes)
		marque := ""
		if w == v3dFenetreMS {
			marque = "  <= FENETRE DU GATE 7"
		}
		t.Logf("  mort a +/-%5d ms de la FERMETURE : reel %3d/%3d = %5.1f %% · TEMOIN %3d = %5.1f %% · ecart %+5.1f pts%s",
			w, tg.mortFermeture[w], trousFermes, 100*pr, tg.temoinFermeture[w], 100*pt,
			100*(pr-pt), marque)
	}
	pr := attPart(tg.mortFermeture[v3dFenetreMS], trousFermes)
	pt := attPart(tg.temoinFermeture[v3dFenetreMS], trousFermes)
	t.Logf("  GATE 7 (sortie par la mort - temoin > %.0f pts) : %+.1f pts — %s",
		100*v3dEcartTemoinMin, 100*(pr-pt), v3dVerdictGate(pr-pt > v3dEcartTemoinMin))
	t.Logf("  ecart SIGNE mort - fermeture du trou : mediane %d ms sur %d (positif = la mort suit la fermeture)",
		v3dMediane(tg.ecartMortFermetureSigne), len(tg.ecartMortFermetureSigne))
	med := v3dMediane(tg.ecartVehicule)
	t.Logf("  GATE 8 (mediane |t_mort - fin serree du VEHICULE| <= %d ms) : mediane %d ms sur %d — %s",
		v3dPrecisionMedianeMaxMS, med, len(tg.ecartVehicule),
		v3dVerdictGate(len(tg.ecartVehicule) > 0 && med <= v3dPrecisionMedianeMaxMS))
}

// ========================= ETAGE 4 — LE DEVENIR D'UN EMBARQUEMENT =========================
//
// POURQUOI CET ETAGE. La grammaire de l'evenement d'EMBARQUEMENT a ete corrigee le 2026-09-02
// par lecture de l'executable (rapport V3_EMBARQUEMENT) : ses trois references sont en domaines
// 2, 3, 7 — et non 1, 7, 7 — et son occupant tombe desormais a 100 % dans la bande bipede. La
// mesure cote `filmdec` montre que 77,3 % des embarquements OUVRENT un trou de position a
// l'instant exact (temoin 0 %), mais que 0 sur 17 de ces trous sont refermes par un evenement
// de SORTIE, quel qu'en soit l'occupant. Reste la seconde branche du gate, que seul le paquet
// `replay` peut mesurer (elle exige le pont slot -> xuid et le calage d'horloge du fil des
// morts) : l'occupant d'un embarquement MEURT-IL au lieu de sortir ?
//
// GATE B5, ecrit avant la mesure. La part des embarquements dont l'occupant meurt entre
// l'ouverture du trou et sa fermeture (+ v3dFenetreMS) doit depasser de plus de
// v3dEcartTemoinMin (10 pts) celle du TEMOIN a occupant decale (meme intervalle). ECHANTILLON
// ATTENDU FAIBLE : les embarquements sont rares (348 sur 949 films) et seules les cartes
// pourvues de bornes de quantification sont mesurables ici — le compte est publie avec le
// resultat, et un n trop faible se dit au lieu de se cacher.

// v3dBoardAgg agrege l'etage 4.
type v3dBoardAgg struct {
	boards, ouvrentTrou, fermes int
	avecSortie                  int
	avecMort, avecMortTemoin    int
	sansRien                    int
	dureeTrouMS                 []int64
}

// v3dAnalyseBoards classe chaque embarquement date d'un film.
func v3dAnalyseBoards(ctx v3dCtx, ag *v3dBoardAgg) {
	for _, b := range ctx.boards {
		ag.boards++
		gs, ok := v3dTrouOuvertA(ctx.ptracks[b.Slot], b.TsUS)
		if !ok {
			continue
		}
		ag.ouvrentTrou++
		ge, ferme := v3dFinDuTrou(ctx.ptracks[b.Slot], gs)
		if !ferme {
			ge = ^uint64(0)
		} else {
			ag.fermes++
			ag.dureeTrouMS = append(ag.dureeTrouMS, int64(ge-gs)/1000)
		}
		_, aSortie := v3dSortieDans(ctx.sorties[b.Slot], gs, ge)
		if aSortie {
			ag.avecSortie++
		}
		v3dDevenirOccupant(ctx, b.Slot, gs, ge, ferme, aSortie, ag)
	}
}

// v3dDevenirOccupant impute la branche MORT du gate B5 pour un embarquement.
func v3dDevenirOccupant(ctx v3dCtx, slot uint32, gs, ge uint64, ferme, aSortie bool,
	ag *v3dBoardAgg) {
	xuid, connu := ctx.slotX[slot]
	if !connu {
		return
	}
	fin := ge
	if ferme {
		fin = ge + uint64(v3dFenetreMS)*1000
	}
	if v3dMortDans(ctx.morts[v3dTemoinXUID(ctx.roster, xuid)], gs, fin) > 0 {
		ag.avecMortTemoin++
	}
	if v3dMortDans(ctx.morts[xuid], gs, fin) > 0 {
		ag.avecMort++
		return
	}
	if !aSortie {
		ag.sansRien++
	}
}

// v3dTrouOuvertA rend l'instant d'ouverture du trou de ce slot le plus proche de `at`, s'il
// tombe a moins de v3dFenetreMS. Le trou s'OUVRE au dernier echantillon avant le silence.
func v3dTrouOuvertA(tr slotTrack, at uint64) (uint64, bool) {
	for i := 1; i < len(tr.pts); i++ {
		gs := tr.pts[i-1].TimestampUS
		if tr.pts[i].TimestampUS-gs < uint64(v3dTrouMinMS)*1000 {
			continue
		}
		if absI64(int64(gs/1000)-int64(at/1000)) <= v3dFenetreMS {
			return gs, true
		}
	}
	return 0, false
}

// v3dRapportBoards publie l'etage 4 et verdit le gate B5.
func v3dRapportBoards(t *testing.T, ag *v3dAgg) {
	t.Helper()
	b := &ag.boards
	t.Logf("\n---------- ETAGE 4 — LE DEVENIR D'UN EMBARQUEMENT (%d embarquements dates) ----------",
		b.boards)
	if b.boards == 0 {
		t.Logf("  aucun embarquement dans ce corpus — rien a juger (ils sont rares : 348 sur 949 films)")
		return
	}
	t.Logf("  ouvrent un trou de position a l'instant : %d/%d = %.1f %% · trou referme %d · duree mediane %d ms",
		b.ouvrentTrou, b.boards, 100*attPart(b.ouvrentTrou, b.boards), b.fermes,
		v3dMediane(b.dureeTrouMS))
	pm, pt := attPart(b.avecMort, b.ouvrentTrou), attPart(b.avecMortTemoin, b.ouvrentTrou)
	t.Logf("  DEVENIR : sortie du MEME occupant %d · MORT de l'occupant %d (%.1f %%) · TEMOIN (occupant decale) %d (%.1f %%) · ni l'un ni l'autre %d",
		b.avecSortie, b.avecMort, 100*pm, b.avecMortTemoin, 100*pt, b.sansRien)
	t.Logf("  GATE B5 (mort de l'occupant - temoin > %.0f pts) : %+.1f pts sur n=%d — %s",
		100*v3dEcartTemoinMin, 100*(pm-pt), b.ouvrentTrou,
		v3dVerdictGate(pm-pt > v3dEcartTemoinMin))
}

// ---------------------- RAPPORT DE L'ETAGE 1 (deplace ici pour la taille de fichier) ----------
// v3dRapport publie le classement, les trois gates et les fenetres de report.
func v3dRapport(t *testing.T, ag *v3dAgg) {
	t.Helper()
	t.Logf("\n########## V3 — DESTRUCTION DATEE PAR LA MORT DU CONDUCTEUR (%d films) ##########", ag.films)
	t.Logf("  vies recensees %d · avec trajectoire %d · avec candidat occupant %d · occupant COURANT %d (ambigus %d)",
		ag.vies, ag.avecPiste, ag.avecCand, ag.avecOccupant, ag.ambigus)
	t.Logf("  CLASSEMENT : DESTRUCTION %d (%.1f %% des vies · %.1f %% des vies a occupant courant) · SORTIE %d · DESPAWN %d · INCONNUE %d",
		ag.destruction, 100*attPart(ag.destruction, ag.vies), 100*attPart(ag.destruction, ag.avecOccupant),
		ag.sortie, ag.despawn, ag.inconnue)
	for _, w := range v3dFenetresMS {
		pr, pt := attPart(ag.parFenetre[w], ag.avecOccupant), attPart(ag.temoin[w], ag.avecOccupant)
		marque := ""
		if w == v3dFenetreMS {
			marque = "  <= FENETRE DU GATE"
		}
		t.Logf("  fenetre +/-%5d ms : reel %3d/%3d = %5.1f %% · TEMOIN (occupant decale) %3d = %5.1f %% · ecart %+5.1f pts%s",
			w, ag.parFenetre[w], ag.avecOccupant, 100*pr, ag.temoin[w], 100*pt, 100*(pr-pt), marque)
	}
	v3dGates(t, ag)
	v3dRapportTrous(t, &ag.trous)
}

// v3dGates verdit les trois gates ecrits avant la mesure.
func v3dGates(t *testing.T, ag *v3dAgg) {
	t.Helper()
	reel := attPart(ag.parFenetre[v3dFenetreMS], ag.avecOccupant)
	tem := attPart(ag.temoin[v3dFenetreMS], ag.avecOccupant)
	g1 := reel-tem > v3dEcartTemoinMin
	med := v3dMediane(ag.ecarts)
	g2 := ag.destruction > 0 && med <= v3dPrecisionMedianeMaxMS
	coherentes := ag.dansTrou + ag.situesProches
	g3 := ag.destruction > 0 && attPart(coherentes, ag.destruction) >= v3dPartSpatialeMin
	t.Logf("  GATE 1 (reel - temoin > %.0f pts a +/-%d ms) : %+.1f pts — %s",
		100*v3dEcartTemoinMin, v3dFenetreMS, 100*(reel-tem), v3dVerdictGate(g1))
	t.Logf("  GATE 2 (mediane |t_mort - t_fin| <= %d ms) : mediane %d ms sur %d datations — %s",
		v3dPrecisionMedianeMaxMS, med, len(ag.ecarts), v3dVerdictGate(g2))
	t.Logf("  GATE 3 (coherence spatiale >= %.0f %%) : dans le trou %d + situes < %.0f m %d/%d = %.1f %% — %s",
		100*v3dPartSpatialeMin, ag.dansTrou, v3dRayonMortM, ag.situesProches, ag.situes,
		100*attPart(coherentes, ag.destruction), v3dVerdictGate(g3))
	t.Logf("  VERDICT GLOBAL : %s", v3dVerdictGate(g1 && g2 && g3))
}

func v3dVerdictGate(ok bool) string {
	if ok {
		return "PASSE"
	}
	return "ECHOUE"
}

// v3dMediane rend la mediane d'une serie d'ecarts (0 si vide).
func v3dMediane(xs []int64) int64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]int64(nil), xs...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}
