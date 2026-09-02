package replay

// vehicules_v3_destruction_test.go — INSTRUMENT DE MESURE (lot V3, item A) : DATER la
// DESTRUCTION d'un vehicule par la MORT DE SON CONDUCTEUR. LECTURE SEULE, garde par
// V3_DESTR_FILMS / V3_DESTR_ROOT (sans env : le test saute proprement).
//
// POURQUOI CE SECOND ESSAI. Le lot V2 (item 4b) avait deja teste « une mort coincide avec la fin
// de vie d'un vehicule » et conclu AU HASARD (52 % contre 48 % au temoin) — mais il testait
// N'IMPORTE QUELLE mort, sans savoir QUI conduisait. Les morts sont denses (~95/match) : une
// fenetre de quelques secondes en capture la moitie par pur hasard. Depuis, l'OCCUPANT est
// resolu : V1a.4/V1c le nomment par le DEBUT DE TROU du flux de position pres d'un vehicule
// (« l'enfant attache ne replique plus »), et V2b l'a confirme a 100 % contre 0 % au temoin par
// la liste d'evenements. On refait donc la mesure en la RESTREIGNANT a l'occupant courant du
// vehicule : le denominateur du hasard passe de « toutes les morts du match » a « les morts d'UN
// joueur nomme », soit ~12 morts au lieu de ~95.
//
// ============================ LE GATE, ECRIT AVANT TOUTE MESURE ============================
//
// DEFINITION d'une DESTRUCTION DATEE. La vie de vehicule V (recensement des images-cles,
// cle (slot, gen), fenetre bornee [premiere image-cle .. premiere image-cle APRES la derniere])
// est dite DETRUITE ET DATEE a l'instant t_mort si les QUATRE conditions suivantes tiennent :
//
//	(1) FIN SERREE. V porte une trajectoire dans sa fenetre de recensement ; sa fin serree
//	    t_fin est le dernier echantillon de position du slot dans cette fenetre (~0,5 s de
//	    resolution, contre +/-20 s pour le recensement seul).
//	(2) OCCUPANT COURANT. Il existe un bipede O dont un TROU du flux de position s'OUVRE a
//	    moins de attBordRayonM (1,5 m) de V pendant la fenetre de vie (primitive V1a.4/V1c,
//	    rejouee telle quelle), dont le trou COUVRE t_fin (ouvert avant, non referme apres), et
//	    qui n'a AUCUN evenement de SORTIE (filmdec.ScanFilmVehicleEvents) entre l'ouverture du
//	    trou et t_fin. Autrement dit : O est encore a bord quand la vie de V s'arrete.
//	(3) MORT DE L'OCCUPANT. O meurt (fil des morts du film, cale sur l'horloge du film par le
//	    pont de production buildOwners : t = Death.TimeMS + DeathOffsetMS) a t_mort tel que
//	    |t_mort - t_fin| <= v3dFenetreMS.
//	(4) COHERENCE SPATIALE. La derniere position connue de V (celle de t_fin) est a moins de
//	    v3dRayonMortM de la position de O a t_mort — OU O n'a AUCUN echantillon de position a
//	    +/-v3dEchantillonGapUS de sa mort, ce qui est le signal ATTENDU d'un occupant EMBARQUE
//	    (V1a.4 : l'enfant attache ne replique plus ; V2 § 4.3 a mesure que les victimes
//	    SITUABLES sont des badauds a 17-25 m). La distance passe par dist3 (geometry.go) via
//	    l'adaptateur v2dDist — jamais une formule recopiee (garde-rail
//	    TestUneSeuleFormuleDeDistance3D).
//
// TROIS SEUILS A FRANCHIR, tous fixes ci-dessous en constantes nommees :
//
//	GATE 1 — AU-DESSUS DU HASARD. Sur les vies a occupant courant nomme, la part DETRUITE a
//	  la fenetre v3dFenetreMS doit depasser de plus de v3dEcartTemoinMin (10 pts) celle du
//	  TEMOIN A OCCUPANT DECALE : le meme test, mais avec les morts d'un AUTRE joueur (rotation
//	  deterministe de v3dTemoinRotation crans dans le roster trie). Le temoin garde donc la
//	  densite de morts d'un joueur reel et ne change QUE l'identite — c'est exactement
//	  l'hypothese « au hasard » du lot V2.
//	GATE 2 — PRECISION. La MEDIANE de |t_mort - t_fin| sur les destructions datees doit rester
//	  sous v3dPrecisionMedianeMaxMS (5 s), tres en dessous des +/-20 s du recensement : sans
//	  cela, « dater » n'apporte rien sur le bornage deja disponible.
//	GATE 3 — COHERENCE SPATIALE. Au moins v3dPartSpatialeMin (90 %) des destructions datees
//	  satisfont la condition (4) — occupant dans le trou, ou situe a moins de v3dRayonMortM.
//
// CLASSEMENT de TOUTES les vies (aucune vie sans statut) :
//
//	DESTRUCTION  (1)+(2)+(3) ; datee a t_mort.
//	SORTIE       occupant(s) candidat(s), mais tous ont quitte le vehicule avant t_fin (trou
//	             referme, ou evenement de sortie) : la fin de vie n'est pas une destruction.
//	DESPAWN      occupant courant identifie, mais aucune mort coincidente : vehicule abandonne.
//	INCONNUE     aucun candidat occupant, ou aucune trajectoire (V1c n'attribue que 15-21 %
//	             des vies : la non-attribution est publiee, pas cachee).
//
// ============================================================================================
//
// CE QUI EST REUTILISE, SANS UNE LIGNE RECOPIEE (regle des <= 2 copies) : v0Corpus/v0Bornes
// (corpus et bornes de carte), v1aBandeVehicule/v1aOptions/v1aPistes (nuage vehicule),
// v1cVies/v1cGapStartsNearVehicles/v1cAttribue (la primitive conducteur de V1c),
// indexBySlot/slotTrack.at (index temporel), v2dTightEnd (fin serree) et v2dDist (adaptateur
// dist3), ScanFilmDeaths/ScanFilmPlayerIndices/injectiveOrEmpty/buildOwners (fil des morts et
// CALAGE d'horloge prouve), filmdec.ScanFilmVehicleEvents (sorties datees a la ms).
//
// UN SEUL decodage filmdec par process : le verrou est pris par film.
//
//	CGO_ENABLED=0 V3_DESTR_ROOT=<depot>/data/cache \
//	  V3_DESTR_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  go test ./internal/analysis/replay/ -run '^TestV3DestructionDatee$' -v -timeout 180m

import (
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Gardes d'environnement de l'instrument.
const (
	v3dFilmsEnv = "V3_DESTR_FILMS"
	v3dRootEnv  = "V3_DESTR_ROOT"
)

// SEUILS DU GATE, ecrits AVANT toute mesure (cf. en-tete).
const (
	// v3dFenetreMS : ecart maximal |t_mort - t_fin| d'une destruction datee. 3 s = la
	// resolution de la fin serree (~0,5 s) + le pas d'echantillonnage + la tolerance du trou.
	v3dFenetreMS = int64(3000)
	// v3dPrecisionMedianeMaxMS : GATE 2. Sous ce seuil, la datation bat le bornage (+/-20 s).
	v3dPrecisionMedianeMaxMS = int64(5000)
	// v3dEcartTemoinMin : GATE 1. Ecart minimal reel - temoin, en points de part.
	v3dEcartTemoinMin = 0.10
	// v3dRayonMortM : GATE 3. Rayon de coherence spatiale entre la derniere position du
	// vehicule et la victime SITUEE. 12 m : au-dela d'une carrosserie et d'une ejection, mais
	// tres en deca des 17-25 m medians des badauds mesures au lot V2 (§ 4.3).
	v3dRayonMortM = 12.0
	// v3dPartSpatialeMin : GATE 3, part minimale des destructions spatialement coherentes.
	v3dPartSpatialeMin = 0.90
	// v3dEchantillonGapUS : ecart max entre la mort et un echantillon de position de la
	// victime pour la dire SITUEE. Au-dela, elle est « dans le trou » (embarquee).
	v3dEchantillonGapUS = uint64(2_000_000)
	// v3dTemoinRotation : rotation (en crans du roster trie) de l'occupant du TEMOIN.
	v3dTemoinRotation = 3
	// v3dTrouMinMS : duree minimale d'un TROU du flux de position (= attTrouMS, la primitive
	// V1a.4). Sert a l'etage 4 (le devenir d'un embarquement).
	v3dTrouMinMS = attTrouMS
)

// v3dFenetresMS : fenetres de report (celle du gate est v3dFenetreMS). Elles ne changent PAS le
// gate : elles montrent comment reel et temoin se separent quand la fenetre s'ouvre.
var v3dFenetresMS = []int64{1000, 3000, 10000, 20000}

// v3dMondeSeul ne garde que les echantillons PORTANT une position monde. C'est le meme filtre
// que celui de la primitive V1c (v1cGapStartsNearVehicles ne compte les trous que sur les points
// `HasWorld`) : sans lui, la FERMETURE d'un trou se lirait sur un point sans position et tout
// occupant paraitrait avoir quitte le vehicule aussitot.
func v3dMondeSeul(pos []filmdec.BipedPosition) []filmdec.BipedPosition {
	out := make([]filmdec.BipedPosition, 0, len(pos))
	for _, p := range pos {
		if p.HasWorld {
			out = append(out, p)
		}
	}
	return out
}

// v3dCtx porte tout ce qu'un film livre une fois decode.
type v3dCtx struct {
	ptracks map[uint32]slotTrack // positions JOUEUR par slot (monde seul)
	vtracks map[uint32]slotTrack // positions VEHICULE par slot (monde seul)
	times   []uint64             // instants de toutes les images-cles (bornage)
	slotX   map[uint32]uint64    // pont slot bipede -> xuid (production)
	morts   map[uint64][]uint64  // xuid -> instants de mort, horloge FILM (us), tries
	roster  []uint64             // xuids tries (temoin par rotation)
	sorties map[uint32][]uint64  // slot occupant -> instants de sortie (us)
	boards  []v3dBoard           // embarquements datés (occupant résolu depuis le 2026-09-02)
	offset  int64                // calage du fil des morts (ms)
}

// v3dAgg agrege un corpus.
type v3dAgg struct {
	films, vies, avecPiste, avecCand, avecOccupant, ambigus int
	destruction, sortie, despawn, inconnue                  int
	ecarts                                                  []int64
	dansTrou, situes, situesProches                         int
	parFenetre, temoin                                      map[int64]int
	// trous porte l'ETAGE 2 (le trou d'embarquement), agrege sur le meme corpus.
	trous v3dTrouAgg
	// boards porte l'ETAGE 4 (le devenir d'un EMBARQUEMENT daté).
	boards v3dBoardAgg
}

func newV3dAgg() *v3dAgg {
	ag := &v3dAgg{parFenetre: map[int64]int{}, temoin: map[int64]int{}}
	ag.trous.bord = newV3dTrouAgg3()
	return ag
}

// TestV3DestructionDatee mesure la datation de la destruction sur le corpus.
func TestV3DestructionDatee(t *testing.T) {
	root := os.Getenv(v3dRootEnv)
	if root == "" {
		t.Skipf("mesure non demandee : %s vide (racine du cache film)", v3dRootEnv)
	}
	ag := newV3dAgg()
	for _, f := range v3dCorpus(t) {
		v3dUnFilm(t, root, f, ag)
	}
	v3dRapport(t, ag)
	v3dRapportBoards(t, ag)
}

// v3dCorpus lit le corpus « short8:carte, ... » de l'environnement.
func v3dCorpus(t *testing.T) []v0Film {
	t.Helper()
	v := os.Getenv(v3dFilmsEnv)
	if v == "" {
		t.Skipf("mesure non demandee : %s vide (« short8:carte, ... »)", v3dFilmsEnv)
	}
	var out []v0Film
	for _, s := range strings.Split(v, ",") {
		id, carte, ok := strings.Cut(strings.TrimSpace(s), ":")
		if !ok {
			t.Fatalf("entree de corpus invalide %q : forme attendue « short8:carte »", s)
		}
		out = append(out, v0Film{ID: strings.TrimSpace(id), Carte: strings.TrimSpace(carte)})
	}
	return out
}

// v3dUnFilm decode UN film et classe toutes ses vies de vehicule.
func v3dUnFilm(t *testing.T, root string, f v0Film, ag *v3dAgg) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("V3d %s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V3d %s (%s) — bande ti=%d vide : rien a mesurer", f.ID, f.Carte, attVehiculeTI)
		return
	}
	// V3_DESTR_BRUT arme le FLUX BRUT du nuage vehicule (post-filtres MaxSpeedMPS /
	// IsolationGapMS desarmes, cf. v1aOptions) : il ne change aucune grammaire, il cesse
	// seulement d'ECARTER des echantillons. Le defaut reste le reglage de V1c, pour que les
	// comptes de candidats restent comparables au rapport V1.
	vehPos, err := filmdec.ScanFilmBipedPositionsForBand(dir, bande,
		v1aOptions(&wr, os.Getenv("V3_DESTR_BRUT") == ""))
	if err != nil {
		t.Logf("V3d %s : nuage vehicule : %v", f.ID, err)
		return
	}
	optBip := filmdec.DefaultScanFilmOptions()
	optBip.WorldRange = &wr
	bip, err := filmdec.ScanFilmBipedPositions(dir, optBip)
	if err != nil {
		t.Logf("V3d %s : nuage bipede : %v", f.ID, err)
		return
	}
	vies := v1cVies(dir)
	v1cAttribue(v1cGapStartsNearVehicles(bip, v1aPistes(vehPos)), vies)
	ctx := v3dContexte(t, dir, bip, vehPos)
	v3dAnalyseBoards(ctx, &ag.boards)
	ag.films++
	avant := *ag
	for i := range vies {
		v3dClasseVie(t, vies[i], ctx, ag)
	}
	t.Logf("V3d %s (%s) — offset %d ms · %d vies · DESTRUCTION %d · SORTIE %d · DESPAWN %d · INCONNUE %d",
		f.ID, f.Carte, ctx.offset, len(vies),
		ag.destruction-avant.destruction, ag.sortie-avant.sortie,
		ag.despawn-avant.despawn, ag.inconnue-avant.inconnue)
}

// v3dContexte assemble le contexte d'un film : pont slot->xuid + calage d'horloge (production),
// morts par joueur, sorties datees, index temporels.
func v3dContexte(t *testing.T, dir string, bip, vehPos []filmdec.BipedPosition) v3dCtx {
	t.Helper()
	tousBipedes := indexBySlot(bip)
	ctx := v3dCtx{
		ptracks: indexBySlot(v3dMondeSeul(bip)),
		vtracks: indexBySlot(v3dMondeSeul(vehPos)),
		times:   filmdec.ScanFilmWorldObjectKeyframes(dir, int(attVehiculeTI)).TimesUS,
		morts:   map[uint64][]uint64{},
		sorties: map[uint32][]uint64{},
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Logf("V3d %s : fil des morts illisible (%v) — aucune datation possible", shortOf(dir), err)
		return ctx
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Logf("V3d %s : index joueur illisible (%v) — pont sans identite", shortOf(dir), err)
	}
	table, _ := injectiveOrEmpty(idx)
	// Le pont de production se construit sur le flux COMPLET (c'est son entree habituelle) ;
	// seules les mesures de trou et de distance passent par le flux monde-seul.
	own := buildOwners(tousBipedes, deaths, table, nil)
	ctx.slotX, ctx.offset = own.SlotXUID, own.DeathOffsetMS
	for _, d := range deaths {
		if ms := d.TimeMS + own.DeathOffsetMS; ms > 0 {
			ctx.morts[d.XUID] = append(ctx.morts[d.XUID], uint64(ms)*1000)
		}
	}
	for x := range ctx.morts {
		sort.Slice(ctx.morts[x], func(i, j int) bool { return ctx.morts[x][i] < ctx.morts[x][j] })
		ctx.roster = append(ctx.roster, x)
	}
	sort.Slice(ctx.roster, func(i, j int) bool { return ctx.roster[i] < ctx.roster[j] })
	v3dLitSorties(t, dir, &ctx)
	return ctx
}

// v3dLitSorties remplit les sorties datees (liste d'evenements, occupant a la ms).
func v3dLitSorties(t *testing.T, dir string, ctx *v3dCtx) {
	t.Helper()
	evs, err := filmdec.ScanFilmVehicleEvents(dir)
	if err != nil {
		t.Logf("V3d %s : liste d'evenements illisible (%v) — sorties non lues", shortOf(dir), err)
		return
	}
	for _, e := range evs {
		if !e.OccupantPresent {
			continue
		}
		if e.Kind == filmdec.EventUnitExitVehicle {
			ctx.sorties[e.OccupantSlot] = append(ctx.sorties[e.OccupantSlot], e.TimestampUS)
			continue
		}
		ctx.boards = append(ctx.boards, v3dBoard{Slot: e.OccupantSlot, TsUS: e.TimestampUS})
	}
}

// v3dBoard est un EMBARQUEMENT daté, occupant résolu (grammaire corrigée le 2026-09-02).
type v3dBoard struct {
	Slot uint32
	TsUS uint64
}

// v3dDumpVie journalise le detail d'une vie a candidats (garde V3_DESTR_DUMP) : c'est le seul
// moyen de voir POURQUOI un candidat n'est pas l'occupant courant.
func v3dDumpVie(t *testing.T, v v1cVie, ctx v3dCtx, finUS uint64) {
	t.Helper()
	if os.Getenv("V3_DESTR_DUMP") == "" || len(v.Cand) == 0 {
		return
	}
	slots := make([]uint32, 0, len(v.Cand))
	for s := range v.Cand {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		debut := v.Cand[s]
		fin, ferme := v3dFinDuTrou(ctx.ptracks[s], debut)
		t.Logf("V3d DUMP vie slot=%d gen=%d [%.1f..%.1f s] fin_serree=%.1f s · cand=%d trou %.1f -> %.1f s (ferme=%t) sorties=%d",
			v.Key.Slot, v.Key.Gen, float64(v.T0)/1e6, float64(v.T1)/1e6, float64(finUS)/1e6,
			s, float64(debut)/1e6, float64(fin)/1e6, ferme, len(ctx.sorties[s]))
	}
}

// v3dClasseVie applique les quatre conditions du gate a UNE vie et l'impute a l'agregat.
func v3dClasseVie(t *testing.T, v v1cVie, ctx v3dCtx, ag *v3dAgg) {
	ag.vies++
	goneBy := v3dGoneBy(ctx.times, v.T1)
	finUS, finPos, aPos := v2dTightEnd(ctx.vtracks[v.Key.Slot], v.T0, goneBy)
	if finUS == 0 {
		ag.inconnue++
		return
	}
	ag.avecPiste++
	if len(v.Cand) == 0 {
		ag.inconnue++
		return
	}
	ag.avecCand++
	v3dDumpVie(t, v, ctx, finUS)
	v3dAnalyseTrous(v, ctx, finUS, &ag.trous)
	occ, ambigu, courant := v3dOccupantCourant(v, ctx, finUS)
	if !courant {
		ag.sortie++
		return
	}
	ag.avecOccupant++
	if ambigu {
		ag.ambigus++
	}
	xuid, connu := ctx.slotX[occ]
	if !connu {
		ag.despawn++
		return
	}
	v3dCompteFenetres(ctx, xuid, finUS, ag)
	ecart, mortUS, ok := v3dMortLaPlusProche(ctx.morts[xuid], finUS, v3dFenetreMS)
	if !ok {
		ag.despawn++
		return
	}
	ag.destruction++
	ag.ecarts = append(ag.ecarts, ecart)
	avant := ag.situesProches + ag.dansTrou
	v3dCompteSpatial(ctx, occ, mortUS, finPos, aPos, ag)
	t.Logf("V3d DESTRUCTION vie slot=%d gen=%d · occupant slot=%d · fin serree %.1f s · mort %.1f s · ecart %d ms · spatial %s",
		v.Key.Slot, v.Key.Gen, occ, float64(finUS)/1e6, float64(mortUS)/1e6, ecart,
		v3dVerdictGate(ag.situesProches+ag.dansTrou > avant))
}

// v3dCompteFenetres impute, pour chaque fenetre de report, le reel et le TEMOIN a occupant
// decale (rotation deterministe dans le roster : meme densite de morts, autre identite).
func v3dCompteFenetres(ctx v3dCtx, xuid, finUS uint64, ag *v3dAgg) {
	tem := v3dTemoinXUID(ctx.roster, xuid)
	for _, w := range v3dFenetresMS {
		if _, _, ok := v3dMortLaPlusProche(ctx.morts[xuid], finUS, w); ok {
			ag.parFenetre[w]++
		}
		if _, _, ok := v3dMortLaPlusProche(ctx.morts[tem], finUS, w); ok {
			ag.temoin[w]++
		}
	}
}

// v3dCompteSpatial applique la condition (4) : victime dans le trou, ou situee a moins de
// v3dRayonMortM de la derniere position connue du vehicule.
func v3dCompteSpatial(ctx v3dCtx, occ uint32, mortUS uint64, finPos [3]float64, aPos bool, ag *v3dAgg) {
	if !aPos {
		ag.dansTrou++ // pas de position vehicule : la condition ne se juge pas, comptee comme trou
		return
	}
	p, gap := ctx.ptracks[occ].at(mortUS)
	if gap > v3dEchantillonGapUS || !p.HasWorld {
		ag.dansTrou++
		return
	}
	ag.situes++
	if v2dDist([3]float64{float64(p.X), float64(p.Y), float64(p.Z)}, finPos) <= v3dRayonMortM {
		ag.situesProches++
	}
}

// v3dOccupantCourant rend le candidat encore A BORD a t_fin : trou ouvert avant t_fin, non
// referme avant, et aucune SORTIE de ce bipede entre l'ouverture et t_fin. Plusieurs candidats
// survivants : le plus recemment embarque l'emporte, et l'ambiguite est publiee.
func v3dOccupantCourant(v v1cVie, ctx v3dCtx, finUS uint64) (uint32, bool, bool) {
	best, bestDebut, n := uint32(0), uint64(0), 0
	for slot, debut := range v.Cand {
		if debut > finUS {
			continue
		}
		if fin, ok := v3dFinDuTrou(ctx.ptracks[slot], debut); ok && fin < finUS {
			continue
		}
		if v3dSortieEntre(ctx.sorties[slot], debut, finUS) {
			continue
		}
		n++
		if debut >= bestDebut {
			best, bestDebut = slot, debut
		}
	}
	return best, n >= 2, n >= 1
}

// v3dFinDuTrou rend l'instant du premier echantillon de position APRES `debut` : la fermeture
// du trou. ok=false quand le bipede ne re-emet jamais (trou ouvert jusqu'a la fin du film).
func v3dFinDuTrou(tr slotTrack, debut uint64) (uint64, bool) {
	i := sort.Search(len(tr.pts), func(k int) bool { return tr.pts[k].TimestampUS > debut })
	if i >= len(tr.pts) {
		return 0, false
	}
	return tr.pts[i].TimestampUS, true
}

// v3dSortieEntre dit si une sortie de ce bipede tombe dans ]debut, fin].
func v3dSortieEntre(ts []uint64, debut, fin uint64) bool {
	for _, t := range ts {
		if t > debut && t <= fin {
			return true
		}
	}
	return false
}

// v3dMortLaPlusProche rend l'ecart minimal (ms) et l'instant de la mort la plus proche de finUS
// dans la fenetre winMS.
func v3dMortLaPlusProche(morts []uint64, finUS uint64, winMS int64) (int64, uint64, bool) {
	best, bestEcart, ok := uint64(0), winMS+1, false
	for _, m := range morts {
		e := absI64(int64(m/1000) - int64(finUS/1000))
		if e <= winMS && e < bestEcart {
			bestEcart, best, ok = e, m, true
		}
	}
	return bestEcart, best, ok
}

// v3dTemoinXUID rend le xuid du TEMOIN : rotation deterministe de v3dTemoinRotation crans dans
// le roster trie. Il garde la densite de morts d'un joueur reel et ne change que l'identite.
func v3dTemoinXUID(roster []uint64, xuid uint64) uint64 {
	if len(roster) < 2 {
		return 0
	}
	i := sort.Search(len(roster), func(k int) bool { return roster[k] >= xuid })
	return roster[(i+v3dTemoinRotation)%len(roster)]
}

// v3dGoneBy rend la premiere image-cle APRES le dernier recensement : la borne HAUTE de la vie.
func v3dGoneBy(times []uint64, dernier uint64) uint64 {
	for _, ts := range times {
		if ts > dernier {
			return ts
		}
	}
	return dernier
}
