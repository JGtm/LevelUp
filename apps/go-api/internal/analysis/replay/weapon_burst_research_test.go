package replay

// weapon_burst_research_test.go — L'INVENTAIRE FERME DES ARMES AUTOMATIQUES, MESURE AVANT
// D'IMPLANTER LA RAFALE (item C-1 du plan .ai/V7.5/replay2d/PLAN_RETOURS_REJEU_2026-08-27.md).
//
// LE RETOUR UTILISATEUR A EPROUVER (2026-08-27, mot pour mot) : « L'arme MA40 est une auto donc
// les sons joues doivent chacun tirer 3 balles, un fire event pour cette arme c'est une rafale de
// 3 balles (comportement normalement par defaut pour les armes automatiques) ». La decision D7 du
// plan tranche l'IMPLANTATION (rafale a la lecture, pas d'asset re-cuit) ; ce qu'elle ne tranche
// PAS, c'est l'ECART entre deux balles de la rafale. Cet instrument le mesure au lieu de le poser.
//
// DEUX QUESTIONS, DEUX INSTRUMENTS, ET IL FAUT LES DEUX POUR CONCLURE :
//
//  1. LA CADENCE DES FIRE EVENTS DU FILM, par famille d'arme : l'ecart entre deux tirs
//     consecutifs du MEME tireur avec la MEME arme, plafonne a 1,5 s pour ne garder que
//     l'intra-rafale (au-dela, ce n'est plus une rafale, c'est du temps sans tirer). Si un fire
//     event du film vaut trois balles a l'oreille, l'ecart entre deux BALLES est cette mediane
//     divisee par trois — c'est la seule facon d'obtenir un nombre qui vienne de la donnee.
//  2. LE CONTENU DES ASSETS DEJA LIVRES : combien de transitoires d'attaque porte chaque `.wav`.
//     C'est ce qui distingue « l'asset est deja une salve » (candidats : BR75, carabine a
//     impulsion) de « l'asset est un seul coup ». Programmer trois departs d'un fichier qui
//     contient deja trois coups en jouerait neuf : la mesure ferme ce piege AVANT le code.
//
// CE QUE CET INSTRUMENT NE DECIDE PAS : quelles armes recoivent la rafale. Le perimetre initial
// est `hinf_ma40_ar` SEUL (D7) et l'extension appartient au GATE D'ECOUTE de l'utilisateur
// (« les votes priment sur tout critere », RECETTE_SONS_ARMES §5). Une mesure ne vote pas.
//
// POURQUOI UN SECOND INSTRUMENT DE CADENCE, alors que `replaylabels/cadence_corpus_test.go`
// (env `REPLAY_CORPUS`) en publie deja un. Il repond a une AUTRE question — « quelles armes
// tirent en faisceau tenu », d'ou son plafond a 1 s et sa colonne « <= 1 image » — et il ne
// publie ni p10 ni analyse d'asset. Les deux mesures se CONTROLENT l'une l'autre sur la ligne
// MA40 (chiffres compares au rapport du lot C) : c'est la DEUXIEME copie, pas la troisieme, et
// la regle des trois copies (CLAUDE.md n°6) reste tenue. Une troisieme demanderait un helper
// partage.
//
// LECTURE SEULE, AUCUN CODE DE PRODUCTION : l'instrument relit les artefacts JSON deja cuits et
// les `.wav` deja livres. Il ne cuit rien, ne regenere aucun asset, n'ecrit nulle part.
//
// USAGE (depuis apps/go-api) :
//
//	WEAPON_CADENCE_CORPUS=<repo>/data/cache/replays/halo_infinite \
//	  go test ./internal/analysis/replay/ -run '^TestWeaponBurstResearch$' -v -count=1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/weapons"
)

// wbrCorpusEnv designe le REPERTOIRE des artefacts JSON a relire (lecture seule).
const wbrCorpusEnv = "WEAPON_CADENCE_CORPUS"

// LE PLAFOND INTRA-RAFALE, en millisecondes, ecrit avant la mesure. Au-dela, deux tirs ne sont
// plus une rafale : c'est une reprise apres visee, couvert ou rechargement, et les melanger
// tirerait la mediane vers le comportement de jeu au lieu de la cadence de l'arme.
const wbrMaxGapMS = 1500.0

// LES BALLES PAR FIRE EVENT postulees par le retour utilisateur pour une automatique. Le nombre
// est le SIEN ; l'instrument ne fait qu'en deduire l'ecart correspondant.
const wbrBallesParEvent = 3

func TestWeaponBurstResearch(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv(wbrCorpusEnv))
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", wbrCorpusEnv)
	}
	docs := wbrLoad(t, dir)
	if len(docs) == 0 {
		t.Fatalf("aucun artefact porteur de tirs sous %s", dir)
	}
	registre := weapons.FilmshellWeaponKeysByFamily()
	par := wbrMeasureCadence(docs, registre)
	wbrReportCadence(t, docs, par)
	wbrReportSalves(t, docs, registre)
	wbrReportAssets(t, wbrSoundsDir(t))
}

// ---------------------------------------------------------------------------------------------
// INSTRUMENT 1 — LA CADENCE DES FIRE EVENTS PAR FAMILLE D'ARME
// ---------------------------------------------------------------------------------------------

// wbrDoc est un artefact relu, reduit a ce que la cadence lui prend.
type wbrDoc struct {
	// frameMs est la duree REELLE d'une image, telle que l'artefact la publie
	// (`durationMs / frameCount`) : c'est elle qui convertit un ecart d'images en millisecondes.
	frameMs float64
	shots   []Shot
}

// wbrSerie identifie une SUITE DE TIRS : un tireur, une arme. Deux tirs de la meme famille par
// deux tireurs differents ne sont pas consecutifs — les melanger fabriquerait des ecarts qui
// n'ont jamais existe.
type wbrSerie struct{ slot, fam uint32 }

// wbrCadenceStat accumule la mesure d'UNE famille d'arme sur tout le corpus.
type wbrCadenceStat struct {
	fam uint32
	// key est le weapon_key du registre, ou "" pour une famille que le registre ne nomme pas
	// (tourelles, armes de PNJ) : elles restent dans la table sous leur hexadecimal plutot que
	// d'etre ecartees — decider avant de mesurer est exactement ce qu'on refuse ici.
	key  string
	tirs int
	// gaps : les ecarts INTRA-RAFALE retenus (<= wbrMaxGapMS), en millisecondes.
	gaps []float64
	// nuls : parmi `gaps`, ceux a 0 ms — deux degats dans la MEME image. Ce n'est pas une
	// cadence : une arme qui touche deux cibles d'un coup en produit autant, et l'image du film
	// dure 100 ms. Compte a part parce que les inclure ou non CHANGE la mediane.
	nuls int
	// paires : toutes les paires consecutives, plafond compris — le denominateur.
	paires int
}

func wbrLoad(t *testing.T, dir string) []wbrDoc {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("corpus illisible : %v", err)
	}
	sort.Strings(paths)
	out := make([]wbrDoc, 0, len(paths))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s illisible : %v", filepath.Base(p), err)
		}
		var doc ReplayDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s non deserialisable : %v", filepath.Base(p), err)
		}
		if len(doc.Shots) == 0 || doc.FrameCount <= 1 || doc.DurationMS <= 0 {
			continue
		}
		out = append(out, wbrDoc{
			frameMs: float64(doc.DurationMS) / float64(doc.FrameCount),
			shots:   doc.Shots,
		})
	}
	return out
}

// wbrMeasureCadence rejoue la mesure sur tout le corpus : groupement par (film, tireur, famille),
// tri par image, ecarts consecutifs convertis en millisecondes.
func wbrMeasureCadence(docs []wbrDoc, registre map[uint32]string) map[uint32]*wbrCadenceStat {
	par := make(map[uint32]*wbrCadenceStat, 32)
	for _, d := range docs {
		suites := make(map[wbrSerie][]int, 64)
		for _, s := range d.shots {
			fam, ok := FamilyOfWeaponID(s.Weapon)
			if !ok {
				continue
			}
			st := par[fam]
			if st == nil {
				st = &wbrCadenceStat{fam: fam, key: registre[fam]}
				par[fam] = st
			}
			st.tirs++
			k := wbrSerie{slot: s.Slot, fam: fam}
			suites[k] = append(suites[k], s.T)
		}
		for k, frames := range suites {
			sort.Ints(frames)
			st := par[k.fam]
			for i := 1; i < len(frames); i++ {
				st.paires++
				gap := float64(frames[i]-frames[i-1]) * d.frameMs
				if gap > wbrMaxGapMS {
					continue
				}
				st.gaps = append(st.gaps, gap)
				if gap == 0 {
					st.nuls++
				}
			}
		}
	}
	return par
}

// wbrReportCadence publie la table par famille, puis la LIGNE QUI COMMANDE (MA40) et l'ecart de
// rafale qu'elle propose.
func wbrReportCadence(t *testing.T, docs []wbrDoc, par map[uint32]*wbrCadenceStat) {
	t.Helper()
	var tirs int
	for _, d := range docs {
		tirs += len(d.shots)
	}
	t.Logf("CORPUS — %d films porteurs de tirs · %d tirs publies · plafond intra-rafale %.0f ms",
		len(docs), tirs, wbrMaxGapMS)
	lignes := make([]*wbrCadenceStat, 0, len(par))
	for _, st := range par {
		lignes = append(lignes, st)
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].tirs > lignes[j].tirs })
	t.Logf("%-24s %-10s %7s %7s %7s %8s %8s %8s %7s",
		"weapon_key", "famille", "tirs", "paires", "n<=1,5s", "p10(ms)", "med(ms)", "p90(ms)", "nuls%")
	for _, st := range lignes {
		nom := st.key
		if nom == "" {
			nom = "(hors registre)"
		}
		partNulle := 0.0
		if len(st.gaps) > 0 {
			partNulle = 100 * float64(st.nuls) / float64(len(st.gaps))
		}
		t.Logf("%-24s %-10s %7d %7d %7d %8.0f %8.0f %8.0f %6.1f%%",
			nom, fmt.Sprintf("%08X", st.fam), st.tirs, st.paires, len(st.gaps),
			wbrQuantile(st.gaps, 0.10), wbrQuantile(st.gaps, 0.50), wbrQuantile(st.gaps, 0.90),
			partNulle)
	}
	wbrReportMA40(t, par)
}

// wbrReportMA40 publie l'ECART DE RAFALE PROPOSE pour la seule arme du perimetre initial.
//
// DEUX LECTURES SONT PUBLIEES, et c'est deliberé : la mediane BRUTE (tous les ecarts retenus) et
// la mediane HORS ECARTS NULS. Les ecarts nuls sont deux degats dans la meme image de film, pas
// deux balles espacees ; les garder tire la mediane vers le bas d'une quantite qui n'est pas une
// cadence. Publier les deux laisse le choix VISIBLE au lieu de le cacher dans un filtre.
func wbrReportMA40(t *testing.T, par map[uint32]*wbrCadenceStat) {
	t.Helper()
	var ma40 *wbrCadenceStat
	for _, st := range par {
		if st.key == "hinf_ma40_ar" {
			ma40 = st
		}
	}
	if ma40 == nil || len(ma40.gaps) == 0 {
		t.Logf("MA40 — AUCUN tir de `hinf_ma40_ar` dans le corpus : l'ecart de rafale ne peut pas "+
			"etre mesure ici (familles vues : %d)", len(par))
		return
	}
	sansNuls := make([]float64, 0, len(ma40.gaps))
	for _, g := range ma40.gaps {
		if g > 0 {
			sansNuls = append(sansNuls, g)
		}
	}
	brute := wbrQuantile(ma40.gaps, 0.50)
	nette := wbrQuantile(sansNuls, 0.50)
	t.Logf("MA40 — %d tirs · %d ecarts intra-rafale · mediane BRUTE %.0f ms · mediane HORS NULS "+
		"%.0f ms (%d ecarts nuls ecartes)", ma40.tirs, len(ma40.gaps), brute, nette, ma40.nuls)
	t.Logf("MA40 — ECART DE RAFALE PROPOSE = mediane / %d : brute %.1f ms · hors nuls %.1f ms "+
		"(la table `weaponBurstSpecs.ts` retient la seconde, arrondie)",
		wbrBallesParEvent, brute/wbrBallesParEvent, nette/wbrBallesParEvent)
}

// ---------------------------------------------------------------------------------------------
// INSTRUMENT 1 bis — LA CADENCE PAR SALVE, QUI ECHAPPE A LA QUANTIFICATION DU FILM
// ---------------------------------------------------------------------------------------------
//
// POURQUOI LA TABLE PRECEDENTE NE SUFFIT PAS, ET C'EST LA MESURE ELLE-MEME QUI LE DIT : le film
// date ses evenements A L'IMAGE, et une image dure 100 ms. Un ecart entre deux tirs consecutifs ne
// peut donc valoir que 0, 100, 200... ms — la mediane MA40 tombe pile sur 100 ms, avec 14 % de
// zeros. Une grandeur dont tous les quantiles sont des multiples du pas d'echantillonnage est
// CENSUREE : elle ne resout rien sous 100 ms, et diviser une telle mediane par trois donnerait un
// chiffre de precision imaginaire.
//
// CE QUI RESOUT LA CENSURE : une SALVE — une suite de tirs du meme tireur avec la meme arme,
// separes de moins de wbrSalveBreakMS. Son intervalle MOYEN vaut (dernier - premier) / (n - 1) :
// les erreurs d'arrondi des bornes se divisent par le nombre d'intervalles, si bien qu'une salve
// de dix tirs resout la cadence a 10 ms pres. C'est la cadence VRAIE de l'arme, celle qu'un
// joueur entend, et c'est elle qui doit commander l'ecart de la rafale.

// LA RUPTURE DE SALVE, en millisecondes : au-dela, le tireur a cesse de tirer (visee, couvert,
// rechargement). Volontairement plus serre que le plafond intra-rafale — une salve est une tenue
// de detente continue, pas un episode de combat.
const wbrSalveBreakMS = 500.0

// LA LONGUEUR MINIMALE d'une salve retenue, en tirs. En dessous, l'intervalle moyen est celui d'un
// ou deux ecarts bruts : la censure reviendrait par la fenetre.
const wbrSalveMin = 4

// wbrSalveStat accumule les intervalles moyens de salve d'UNE arme (par weapon_key : deux familles
// qui resolvent vers la meme arme — Bandit et Bandit Evo — sont la MEME arme a l'oreille).
type wbrSalveStat struct {
	key       string
	salves    int
	tirs      int
	intervals []float64
}

// wbrReportSalves publie la cadence par salve, arme par arme, et l'ecart de rafale qu'elle dicte.
func wbrReportSalves(t *testing.T, docs []wbrDoc, registre map[uint32]string) {
	t.Helper()
	par := wbrMeasureSalves(docs, registre)
	lignes := make([]*wbrSalveStat, 0, len(par))
	for _, st := range par {
		lignes = append(lignes, st)
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].tirs > lignes[j].tirs })
	t.Logf("CADENCE PAR SALVE — rupture %.0f ms · salves d'au moins %d tirs · intervalle MOYEN par "+
		"salve, non censure par l'image de 100 ms", wbrSalveBreakMS, wbrSalveMin)
	t.Logf("%-24s %8s %8s %9s %9s %9s %11s",
		"weapon_key", "salves", "tirs", "p10(ms)", "med(ms)", "p90(ms)", "med/3(ms)")
	for _, st := range lignes {
		med := wbrQuantile(st.intervals, 0.50)
		t.Logf("%-24s %8d %8d %9.1f %9.1f %9.1f %11.1f",
			st.key, st.salves, st.tirs,
			wbrQuantile(st.intervals, 0.10), med, wbrQuantile(st.intervals, 0.90),
			med/wbrBallesParEvent)
	}
}

// wbrMeasureSalves decoupe chaque suite (film, tireur, arme) en salves et retient l'intervalle
// moyen de chacune.
func wbrMeasureSalves(docs []wbrDoc, registre map[uint32]string) map[string]*wbrSalveStat {
	par := make(map[string]*wbrSalveStat, 32)
	for _, d := range docs {
		suites := make(map[wbrSerie][]int, 64)
		for _, s := range d.shots {
			fam, ok := FamilyOfWeaponID(s.Weapon)
			if !ok {
				continue
			}
			suites[wbrSerie{slot: s.Slot, fam: fam}] = append(suites[wbrSerie{slot: s.Slot, fam: fam}], s.T)
		}
		for k, frames := range suites {
			key := registre[k.fam]
			if key == "" {
				key = fmt.Sprintf("(hors registre %08X)", k.fam)
			}
			st := par[key]
			if st == nil {
				st = &wbrSalveStat{key: key}
				par[key] = st
			}
			sort.Ints(frames)
			for _, salve := range wbrCutSalves(frames, d.frameMs) {
				st.salves++
				st.tirs += len(salve)
				span := float64(salve[len(salve)-1]-salve[0]) * d.frameMs
				st.intervals = append(st.intervals, span/float64(len(salve)-1))
			}
		}
	}
	return par
}

// wbrCutSalves decoupe une suite d'images triee en salves d'au moins wbrSalveMin tirs.
func wbrCutSalves(frames []int, frameMs float64) [][]int {
	var out [][]int
	debut := 0
	for i := 1; i <= len(frames); i++ {
		rupture := i == len(frames)
		if !rupture {
			rupture = float64(frames[i]-frames[i-1])*frameMs > wbrSalveBreakMS
		}
		if !rupture {
			continue
		}
		// UNE SALVE DE DUREE NULLE (tous les tirs dans la meme image) n'a pas d'intervalle a
		// offrir : la retenir diviserait un zero par son nombre d'ecarts et publierait 0 ms.
		if i-debut >= wbrSalveMin && frames[i-1] > frames[debut] {
			out = append(out, frames[debut:i])
		}
		debut = i
	}
	return out
}

// wbrQuantile rend le quantile d'un echantillon (interpolation par index, comme les instruments
// voisins). -1 pour un echantillon vide : un « pas de donnee » doit se voir dans la table.
func wbrQuantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return -1
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	i := int(q * float64(len(c)-1))
	return c[i]
}

// ---------------------------------------------------------------------------------------------
// INSTRUMENT 2 — LE CONTENU DES ASSETS LIVRES (combien de coups porte un fichier)
// ---------------------------------------------------------------------------------------------

// LES SEUILS DE CONTROLE : ils ne decident rien, ils disent si le compte tient au seuil retenu ou
// s'il bascule au premier reglage — un resultat qui change du simple au double entre 25 % et 50 %
// n'est pas un fait, c'est un artefact de mesure.
var wbrControlRatios = []float64{0.25, 0.35, 0.50}

// LA PRIMITIVE DE SIGNAL VIT A COTE (`weapon_burst_wave_test.go`) : decodage RIFF/WAVE,
// enveloppe, comptage des transitoires, et les quatre reglages du detecteur. Elle ne connait ni
// arme ni artefact — meme coupure que la geometrie de l'instrument des socles.

// wbrSoundsDir rend le dossier des sons livres, en remontant jusqu'a la racine du depot.
func wbrSoundsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("repertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "static", "sounds", "halo_infinite")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("static/sounds/halo_infinite introuvable depuis le repertoire de test")
	return ""
}

// wbrReportAssets publie, pour chaque son d'arme livre, sa duree et son nombre de transitoires —
// puis CONTROLE que l'ensemble des fichiers correspond aux armes du registre.
func wbrReportAssets(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "hinf_*.wav"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("aucun son d'arme sous %s (err %v)", dir, err)
	}
	sort.Strings(paths)
	t.Logf("ASSETS — %d sons d'arme livres · lissage %.0f ms · seuil retenu %.0f %% du max · "+
		"separation %.0f ms · facteur de montee x%.0f",
		len(paths), wbrSmoothMS, wbrOnsetRatio*100, wbrOnsetGapMS, wbrAttackFactor)
	t.Logf("%-24s %8s %6s %7s %9s %9s %9s", "stem", "duree(s)", "canaux", "Hz", "n@25%", "n@35%", "n@50%")
	ancre := -1
	for _, p := range paths {
		stem := strings.TrimSuffix(filepath.Base(p), ".wav")
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s illisible : %v", stem, err)
		}
		w, err := wbrDecodeWave(raw)
		if err != nil {
			t.Fatalf("%s non decodable : %v", stem, err)
		}
		env := wbrEnvelope(w)
		comptes := make([]int, 0, len(wbrControlRatios))
		for _, r := range wbrControlRatios {
			comptes = append(comptes, wbrOnsets(env, w.rate, r))
		}
		if stem == "hinf_s7_sniper" {
			ancre = wbrOnsets(env, w.rate, wbrOnsetRatio)
		}
		t.Logf("%-24s %8.3f %6d %7d %9d %9d %9d",
			stem, w.durationS(), w.channels, w.rate, comptes[0], comptes[1], comptes[2])
	}
	// L'ANCRE DE CALIBRAGE : le fusil de precision est un coup unique. Si elle casse, le compte
	// de toutes les autres lignes est suspect — mieux vaut un test rouge qu'un tableau credible.
	if ancre != 1 {
		t.Errorf("ANCRE DE CALIBRAGE ROMPUE — `hinf_s7_sniper` rend %d transitoire(s) au seuil "+
			"%.0f %%, 1 attendu (arme a verrou, un coup par fichier)", ancre, wbrOnsetRatio*100)
	}
	wbrControlStems(t, paths)
}

// wbrControlStems verifie que les sons livres et les armes du registre se recouvrent : un stem
// sans arme serait un fichier orphelin, une arme sans stem une arme muette. Les GRENADES du
// registre sont attendues sans stem d'arme — elles ont leurs propres tables cote web.
func wbrControlStems(t *testing.T, paths []string) {
	t.Helper()
	livres := make(map[string]bool, len(paths))
	for _, p := range paths {
		livres[strings.TrimSuffix(filepath.Base(p), ".wav")] = true
	}
	registre := make(map[string]bool, 32)
	for _, k := range weapons.FilmshellWeaponKeysByFamily() {
		registre[k] = true
	}
	var sansStem, sansArme []string
	for k := range registre {
		if !livres[k] {
			sansStem = append(sansStem, k)
		}
	}
	for s := range livres {
		if !registre[s] {
			sansArme = append(sansArme, s)
		}
	}
	sort.Strings(sansStem)
	sort.Strings(sansArme)
	t.Logf("CONTROLE — %d sons livres · %d clefs d'arme au registre filmshell", len(livres), len(registre))
	t.Logf("CONTROLE — clefs du registre SANS son d'arme (grenades attendues) : %v", sansStem)
	t.Logf("CONTROLE — sons d'arme SANS clef au registre (aucun attendu) : %v", sansArme)
}
