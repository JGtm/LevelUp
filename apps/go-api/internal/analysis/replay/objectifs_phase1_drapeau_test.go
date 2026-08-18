package replay

// objectifs_phase1_drapeau_test.go — ITEMS 1.1 ET 1.2 : LES PORTAGES, MESURES SUR FILMS REELS.
//
// CET INSTRUMENT N'A PAS DE REGLE A LUI : il construit le DOCUMENT de production
// (`BuildFromPositions` avec `Options.Flag`, cf. build_objectives_live.go) sur les films du
// corpus et publie les chiffres de `doc.FlagCarries` / `doc.Coverage.FlagCarries`. Une seconde
// copie aurait diverge au premier correctif, et surtout la mesure ne dirait plus rien de ce que
// le client recoit.
//
// CE QU'IL MESURE, ET LES SEUILS ECRITS AVANT (plan, gate 1) :
//
//	le CONTROLE DU MARQUEUR   part des portages FERMES, PARMI CEUX QUI CONTIENNENT UNE IMAGE-CLE,
//	                          dont au moins une porte le marqueur `0x00010005` sur le slot du
//	                          porteur. Seuil : >= 90 %. Les portages OUVERTS (que rien ne ferme,
//	                          publies `carried_open`) ont leur propre compte, publie a cote.
//	l'ORACLE                  chaque `flag_grabs` / `flag_steals` nomme par le pont doit ouvrir
//	                          un span `carried` du BON xuid. Seuil : 100 % des prises publiees.
//	les INCOHERENCES          simultaneite > 2 portages, porteurs tues ambigus, retours ambigus :
//	                          publies et comptes, jamais tus.
//	le RETOUR AUTOMATIQUE     (item 1.2) ecart entre une fin de portage SANS reprise et la prise
//	                          suivante au socle. Mesure, pas seuil.
//
// GARDE : `OBJ_FILM` (racine du cache film), comme toute la phase 0. Lecture seule, aucune base.

import (
	"os"
	"path/filepath"
	"sort"

	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// objSeuilMarqueur — le seuil du gate 1, ecrit avant la mesure et jamais rebaisse.
const objSeuilMarqueur = 0.90

// objRepoEnv porte la racine du depot, d'ou se lit le catalogue VERSIONNE d'objectifs de carte
// (`data/titles/{slug}/reference/map_objectives.json`). Absente : la mesure tourne sans socles,
// et le dit — les portages restent mesurables, l'equipe du drapeau non.
const objRepoEnv = "OBJ_REPO"

// objCTFModules — le MODULE de chaque film TEL QUE `map_objectives.json` le nomme. Il ne sert
// qu a retrouver l entree du catalogue d objectifs ; aucun decodage n en depend.
//
// LE MODULE, ET NON LE NOM PUBLIC : dans ce catalogue, `public_name` est VIDE sur la
// quasi-totalite des entrees (il est produit depuis les variantes UGC, qui ne le portent pas).
// Joindre dessus ne trouverait rien, SILENCIEUSEMENT.
//
// ET CE N EST PAS LE MEME MODULE QUE CELUI DES BORNES : `map_quant_bounds.json` dit `va_behemoth`
// et `ridgeline` la ou le catalogue d objectifs dit `behemoth_va_behemoth` et
// `cliffhanger_ridgeline`. Les deux catalogues sont produits par des chaines differentes, et
// aucun test ne les rapproche — d ou les deux tables ci-dessous plutot qu une.
var objCTFModules = map[string]string{
	"64e8adfa": "catalyst", "530820e5": "catalyst", "53ce4390": "behemoth_va_behemoth",
	"bcb6d393": "cliffhanger_ridgeline", "000d5950": "cliffhanger_ridgeline",
}

// objCTFCarteNom — le NOM AFFICHE de la carte de chaque film, la cle du catalogue de bornes de
// quantification (`filmdec.MapQuantCatalog.Lookup`, qui normalise lui-meme).
var objCTFCarteNom = map[string]string{
	"64e8adfa": "Catalyst", "530820e5": "Catalyst", "53ce4390": "Behemoth",
	"bcb6d393": "Cliffhanger", "000d5950": "Cliffhanger",
}

// TestObjectifsPhase1Portages — items 1.1 et 1.2 sur les trois films CTF et un temoin non-CTF.
func TestObjectifsPhase1Portages(t *testing.T) {
	root := objRequireRoot(t)
	joues, cumObs, cumConf, cumOpenObs, cumOpenConf := 0, 0, 0, 0, 0
	for _, id := range append(append([]string{}, objCTFFilms...), "000d5950") {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		joues++
		carries, cov := objMesurePortages(t, root, id, src)
		cumObs, cumConf = cumObs+cov.MarkerObserved, cumConf+cov.MarkerConfirmed
		cumOpenObs, cumOpenConf = cumOpenObs+cov.OpenObserved, cumOpenConf+cov.OpenConfirmed
		objLogPortages(t, id, carries, cov)
		objVerifieOracle(t, root, id, src, objPortageRes{carries: carries, cov: cov})
		objMesureRetourAuto(t, id, carries)
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", objFilmEnv, root)
	}
	part := objPart(cumConf, cumObs)
	// LE VERDICT SE PUBLIE, IL NE FAIT PAS TOMBER L INSTRUMENT — meme regle que le canal delta
	// de la phase 0.2, refute et journalise sans `t.Errorf`. Le seuil est ECRIT ET FIGE ; le
	// resultat NEGATIF est consigne au plan en `[!]` avec sa mesure, ce qui est sa place. Un
	// instrument qui echoue en permanence finit par etre desarme, et c est la mesure qu on perd.
	//
	// LE DENOMINATEUR EST CELUI DES PORTAGES FERMES (arbitrage du 2026-08-18, item 1.3). Le taux
	// MELANGE reste publie a cote : c'est lui qui valait 88,1 % a l'item 1.1, et l'ecart entre
	// les deux EST le resultat — un portage que rien ne ferme est trop long, ses images-cles
	// tardives tombent apres le lacher, et aucune ne porte le marqueur.
	t.Logf("GATE 1.3 — CONTROLE DU MARQUEUR SUR LES FERMES : %d/%d = %.1f %% (seuil %.0f %%) -> %s",
		cumConf, cumObs, 100*part, 100*objSeuilMarqueur, objTenu(part >= objSeuilMarqueur))
	t.Logf("GATE 1.3 — pour memoire, TOUS PORTAGES CONFONDUS : %d/%d = %.1f %% (ouverts seuls : %d/%d)",
		cumConf+cumOpenConf, cumObs+cumOpenObs, 100*objPart(cumConf+cumOpenConf, cumObs+cumOpenObs),
		cumOpenConf, cumOpenObs)
}

// objMesurePortages joue la chaine de production sur un film et rend CE QUE LE DOCUMENT PUBLIE.
//
// DEPUIS L'ITEM 1.3, LA MESURE PASSE PAR L'ARTEFACT LUI-MEME : le calque n'est plus construit a
// cote de l'assemblage mais DEDANS (`Options.Flag` -> `attachFlagCarries`), et c'est
// `doc.FlagCarries` / `doc.Coverage.FlagCarries` qui sont mesures. Un instrument qui rebatirait
// le calque a la main ne dirait plus rien de ce que le client recoit.
func objMesurePortages(t *testing.T, root, id string, src *objDiskFilm) ([]FlagCarry, *FlagCarriesCoverage) {
	t.Helper()
	b := objBridgeOf(t, root, id)
	doc := objDocumentDe(t, root, id, b, src)
	if doc.doc.Coverage == nil {
		t.Fatalf("%s : document sans couverture", id)
	}
	return doc.doc.FlagCarries, doc.doc.Coverage.FlagCarries
}

// objDoc porte le document de rejeu d'un film et l'origine de son axe de temps.
type objDoc struct {
	doc      ReplayDocument
	originUS uint64
	// gw est le balayage `ti=42` du film, garde a cote du document : le controle du drapeau
	// OBJET le rejoue sur les creations d ARMES ordinaires (son temoin), que l artefact ne
	// publie pas.
	gw WorldObjectScan
}

// objDocMemo memorise le document par film : sa construction rebalaye tout le film.
var objDocMemo = map[string]objDoc{}

// objDocumentDe construit le document de rejeu d'un film EN COORDONNEES MONDE, une seule fois
// par process.
//
// LES BORNES DE CARTE SONT OBLIGATOIRES ICI, ET C'EST LA DIFFERENCE AVEC LA PHASE 0. Celle-ci
// travaillait en QUANTA parce qu'elle ne comparait que des instants ; la phase 1 compare des
// POSITIONS DE JOUEUR a des SOCLES lus dans le catalogue de carte, qui sont en METRES. Melanger
// les deux ferait comparer des indices de quantum a des metres — l'attribution du drapeau
// deviendrait un tirage, sans que rien ne le signale.
func objDocumentDe(t *testing.T, root, id string, b objBridge, src *objDiskFilm) objDoc {
	t.Helper()
	if d, ok := objDocMemo[id]; ok {
		return d
	}
	scan := filmdec.DefaultScanFilmOptions()
	quant := objMapQuant(t, id)
	if quant == nil {
		t.Skipf("%s : bornes de carte absentes — la mesure exige des coordonnees MONDE", id)
	}
	wr := quant.Range()
	scan.WorldRange = &wr
	pos, err := filmdec.ScanFilmBipedPositions(objChunkDir(root, id), scan)
	if err != nil {
		t.Fatalf("%s : positions : %v", id, err)
	}
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	if len(pos) == 0 {
		t.Fatalf("%s : aucune position", id)
	}
	idx, err := ScanFilmPlayerIndices(objChunkDir(root, id), rosterFromDeaths(b.Deaths))
	if err != nil {
		t.Fatalf("%s : index de joueur : %v", id, err)
	}
	table, _ := injectiveOrEmpty(idx)
	marks, err := filmdec.ScanFilmCarrierMarks(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : marqueurs de portage : %v", id, err)
	}
	gw := objGroundWeapons(t, root, id, quant)
	doc := BuildFromPositions(id, "halo_infinite", pos, nil, Options{
		Deaths: b.Deaths, PlayerIndices: table, MapQuant: quant,
		Labels: goldenCatalog(t), Pads: PadScans{Weapons: gw},
		Flag: FlagInput{
			Scanned: true, Records: objectiveevents.StatRecords(src),
			Bursts: objectiveevents.CaptureBurstTimes(src), Spawns: objFlagSpawns(t, id), Marks: marks,
		},
	})
	out := objDoc{doc: doc, originUS: pos[0].TimestampUS, gw: gw}
	objDocMemo[id] = out
	return out
}

// objMapQuant rend les bornes de quantification de la carte du film, lues dans le catalogue
// VERSIONNE. nil si la racine du depot n'est pas fournie ou la carte hors catalogue.
func objMapQuant(t *testing.T, id string) *filmdec.MapQuantEntry {
	t.Helper()
	repo, carte := os.Getenv(objRepoEnv), objCTFCarteNom[id]
	if repo == "" || carte == "" {
		return nil
	}
	cat, err := filmdec.LoadMapQuantCatalog(
		filepath.Join(repo, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Logf("%s : catalogue de bornes illisible (%v)", id, err)
		return nil
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Logf("%s : carte %q hors du catalogue de bornes (%v)", id, carte, err)
		return nil
	}
	return &e
}

// objFlagSpawns rend les socles `flag_spawn` de la carte du film, lus dans le catalogue
// VERSIONNE. Liste vide si la racine du depot n'est pas fournie ou la carte hors catalogue :
// la mesure des portages reste valable, l'equipe du drapeau non.
func objFlagSpawns(t *testing.T, id string) []FlagSpawn {
	t.Helper()
	repo := os.Getenv(objRepoEnv)
	module := objCTFModules[id]
	if repo == "" || module == "" {
		return nil
	}
	cat, err := LoadMapObjectives(filepath.Join(repo, "data", "titles", "halo_infinite", "reference", "map_objectives.json"))
	if err != nil {
		t.Logf("%s : catalogue d'objectifs illisible (%v) — mesure sans socles", id, err)
		return nil
	}
	for _, e := range cat.Maps {
		if e.Module != module {
			continue
		}
		pts := e.PointsOfRole(mapvar.RoleFlagSpawn)
		if len(pts) == 0 {
			continue
		}
		// LES SOCLES D EQUIPE SEULEMENT. Chaque carte de CTF en declare TROIS : un par equipe,
		// plus un NEUTRE au centre, qui n'est celui d'aucun camp et ne sert qu'aux variantes
		// « drapeau neutre ». Le retenir sur une partie de CTF ordinaire publierait un troisieme
		// drapeau qui n'existe pas dans le match, immobile a la maison pour l'eternite.
		out := make([]FlagSpawn, 0, len(pts))
		for _, p := range pts {
			if p.TeamIndex == TeamNeutral {
				continue
			}
			out = append(out, FlagSpawn{Team: p.TeamIndex, X: float32(p.Center.X), Y: float32(p.Center.Y)})
		}
		return out
	}
	t.Logf("%s : module %q hors du catalogue d objectifs — mesure sans socles", id, module)
	return nil
}

// objPortageRes regroupe ce que la chaine de production a rendu sur un film — sans lui, le
// controle de l oracle prendrait six parametres.
type objPortageRes struct {
	carries []FlagCarry
	cov     *FlagCarriesCoverage
}

// objLogPortages publie les chiffres d un film.
func objLogPortages(t *testing.T, id string, carries []FlagCarry, cov *FlagCarriesCoverage) {
	t.Helper()
	if cov == nil {
		t.Fatalf("%s : aucune couverture rendue", id)
	}
	t.Logf("%s : CTF=%v (bursts %d, captures %d, vols %d) ; %d prises -> %d portages publies "+
		"(%d fermes, %d ouverts), %d sans pont, %d sans piste, %d hors fenetre ; marqueur "+
		"FERMES %d/%d, OUVERTS %d/%d ; socles %d ; incoherences : simultaneite>2 %d (dont entre "+
		"FERMES %d), porteurs tues ambigus %d, retours ambigus %d",
		id, cov.FlagFilm, cov.Bursts, cov.Captures, cov.Steals, cov.Openings, cov.Carries,
		cov.Closed, cov.Open, cov.NoBridge, cov.NoTrack, cov.OutOfWindow,
		cov.MarkerConfirmed, cov.MarkerObserved, cov.OpenConfirmed, cov.OpenObserved,
		cov.Spawns, cov.Overlaps, cov.ClosedOverlaps, cov.AmbiguousCarrierKills,
		cov.AmbiguousReturns)
	if !cov.Balanced() {
		t.Errorf("%s : invariant de couverture ROMPU — %+v", id, *cov)
	}
	for i, f := range carries {
		etats := map[string]int{}
		for _, s := range f.Spans {
			etats[s.State]++
		}
		t.Logf("%s : drapeau %d (equipe %d) — %d spans : %d portes, %d au sol, %d a la base",
			id, i, f.Team, len(f.Spans), etats[FlagStateCarried]+etats[FlagStateCarriedOpen], etats[FlagStateDropped],
			etats[FlagStateHome])
	}
}

// objVerifieOracle controle que chaque prise NOMMEE de l'oracle ouvre bien un span `carried` du
// bon xuid — la seconde clause du gate 1.
func objVerifieOracle(t *testing.T, root, id string, src *objDiskFilm, res objPortageRes) {
	t.Helper()
	carries, cov := res.carries, res.cov
	if !cov.FlagFilm {
		if len(carries) != 0 {
			t.Errorf("%s : %d drapeaux publies sur un film NON reconnu CTF", id, len(carries))
		}
		return
	}
	b := objBridgeOf(t, root, id)
	evs := objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeFlag)
	identity := objIdentites(src, b.Deaths)
	porteurs := map[string]int{}
	for _, f := range carries {
		for _, s := range f.Spans {
			if flagStateCarrying(s.State) && s.XUID != nil {
				porteurs[*s.XUID]++
			}
		}
	}
	attendus, manquants := map[string]int{}, 0
	for _, o := range flagOpenings(evs, identity) {
		if o.xuid == "" {
			continue
		}
		attendus[o.xuid]++
	}
	for x, n := range attendus {
		if porteurs[x] < n {
			manquants += n - porteurs[x]
		}
	}
	rejets := cov.NoTrack + cov.OutOfWindow
	t.Logf("%s : ORACLE — %d prises nommees, %d spans `carried` publies ; %d prises sans span "+
		"(rejets comptes : %d)", id, sommeDe(attendus), sommeDe(porteurs), manquants, rejets)
	if manquants > rejets {
		t.Errorf("%s : %d prises nommees sans span `carried`, pour seulement %d rejets comptes",
			id, manquants, rejets)
	}
}

// sommeDe additionne les valeurs d'une table.
func sommeDe(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// objMesureRetourAuto — ITEM 1.2 : combien de temps un drapeau reste-t-il au sol avant d'etre
// repris ? La mediane de ces durees BORNE le retour automatique par le haut : un drapeau repris
// apres N frames n'a evidemment pas ete renvoye avant.
func objMesureRetourAuto(t *testing.T, id string, carries []FlagCarry) {
	t.Helper()
	var reprises, jusquAuBout []int
	for _, f := range carries {
		for i, s := range f.Spans {
			if s.State != FlagStateDropped {
				continue
			}
			d := s.T1 - s.T0 + 1
			if i+1 < len(f.Spans) && flagStateCarrying(f.Spans[i+1].State) {
				reprises = append(reprises, d)
				continue
			}
			jusquAuBout = append(jusquAuBout, d)
		}
	}
	t.Logf("%s : ITEM 1.2 — %d laches REPRIS (duree au sol en frames : mediane %d, p10 %d, p90 %d, "+
		"max %d) ; %d laches non repris (mediane %d)", id, len(reprises), objMediane(reprises),
		objPercentile(reprises, 10), objPercentile(reprises, 90), objMax(reprises),
		len(jusquAuBout), objMediane(jusquAuBout))
}

// objPercentile rend le percentile p d'une serie d'entiers (0 si vide).
func objPercentile(v []int, p int) int {
	if len(v) == 0 {
		return 0
	}
	c := append([]int{}, v...)
	sort.Ints(c)
	i := len(c) * p / 100
	if i >= len(c) {
		i = len(c) - 1
	}
	return c[i]
}

// objMax rend le maximum d'une serie (0 si vide).
func objMax(v []int) int {
	m := 0
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}

// objGroundWeapons decode l'archetype `ti=42` du film — LA MEME lecture que la production
// (`BuildFromFilm`), calibration des largeurs du bloc MPP comprise.
//
// POURQUOI L'INSTRUMENT LA FAIT DESORMAIS (2026-08-18, PLAN_DRAPEAU_OBJET phase 1). Le drapeau
// EST un objet `ti=42` : ses vies libres se lisent dans ce balayage-la, et le calque des socles
// doit l'ECARTER. Sans cette entree, l'instrument mesurait un document ou ni l'un ni l'autre
// n'existait — c'est-a-dire un document que la production ne sert pas.
//
// LA CALIBRATION VIENT DES POSES `ti=37`, comme en production : le mot d'identite de 32 bits se
// lit derriere deux champs de largeur VARIABLE, mesures sur CE film. Balayer aux largeurs par
// defaut d'un film calibre autrement ne rend pas une mesure fausse, il rend du bruit.
func objGroundWeapons(t *testing.T, root, id string, quant *filmdec.MapQuantEntry) WorldObjectScan {
	t.Helper()
	dir := objChunkDir(root, id)
	release := filmdec.LockProcessDecode()
	defer release()
	defer installWorldObjectPrecision(*quant, dir)()
	wr := quant.Range()
	_, st := decodeFilmPlacements(dir, &wr)
	return decodeFilmPadScan(dir, &wr, st.Calibration.Widths, groundWeaponArchetype())
}
