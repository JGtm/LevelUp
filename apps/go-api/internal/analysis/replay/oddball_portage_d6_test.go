package replay

// oddball_portage_d6_test.go — D6 : LE PORTAGE DU CRANE PAR LA PROXIMITE.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_
// 2026-08.md`, section « D6-PORTAGE »). Ce qui suit l'applique.
//
// # POURQUOI CE N'EST PAS UN QUATRIEME PASSAGE DU MEME ORACLE
//
// D4 corrolait une SERIE (le score personnel) avec les trous, et l'a refutee. Ici deux choses
// changent de nature. Le CANAL : ce n'est plus une correlation temporelle mais une PROXIMITE
// GEOMETRIQUE — qui se tenait a l'endroit exact ou l'objet a cesse d'emettre. L'ORACLE : il sort
// du film. `time_as_skull_carrier_seconds` vient de l'API, par joueur ; aucune erreur de decodage
// ne peut le contaminer, ce qui est exactement ce qui manquait aux trois oracles precedents.
//
// # LA PRECISION EST CELLE DE L'OBJET, PAS CELLE DES IMAGES-CLES
//
// Le crane replique sa position IMAGE PAR IMAGE tant qu'il est libre : sa derniere position avant
// un trou est connue a l'image. C'est ce qui distingue cette piste de l'item 2.5 des socles, qui
// avait coule parce que la disparition d'un objet n'y etait bornee que par le recensement des
// images-cles, espace de ~20 s.
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM` + `ODDBALL_ORACLE` (fichier d'oracle fige, JSON),
// UN FILM PAR PROCESSUS, lecture seule, AUCUNE base ouverte par ce test.
//
//	$env:ATT_FILM="<cache>"; $env:ODDBALL_FILM="43716616"; $env:ODDBALL_ORACLE="<oracle.json>"
//	go test ./internal/analysis/replay/ -run OddballPortageProximite -v

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/filmproc"
)

const (
	// d6OracleEnv designe le fichier d'oracle FIGE : xuid -> secondes de portage, releve de
	// `match_objective_stats_latest`. Le test n'ouvre AUCUNE base — l'oracle arrive en entree,
	// meme contrat que `MatchFacts` pour le constructeur d'artefact.
	d6OracleEnv = "ODDBALL_ORACLE"
	// d6RayonRamassageM : distance maximale entre l'objet et son ramasseur. C'est
	// `originDropMaxDist`, deja au depot et deja valide des DEUX cotes (lachers a 0,63 m de
	// mediane, deploiements a 5,6-21,3 m) — un objet ramasse est aux pieds de qui le ramasse.
	d6RayonRamassageM = originDropMaxDist
	// d6AmbiguiteM : si le deuxieme plus proche est sous le seuil ET a moins de ceci du premier,
	// le porteur est null. Doctrine des occupations de socle, dont le xuid vaut TOUJOURS null.
	d6AmbiguiteM = 1.0
	// d6EcartMaxMS : ecart temporel tolere entre l'instant du trou et l'echantillon de position
	// le plus proche. Au-dela, on compare deux instants et non deux lieux.
	d6EcartMaxMS = 250
	// d6SocleM : distance au socle `oddball_spawn` en deca de laquelle une vie libre est un
	// RETOUR et non un lacher. Meme tolerance que la recette d'identite de D4.
	d6SocleM = 3.0
	// d6MortFenetreMS / d6MortRayonM : la fenetre et la distance dans lesquelles on cherche une
	// vie libre apres la mort d'un porteur (volet D6.2).
	d6MortFenetreMS = 3000
	d6MortRayonM    = 3.0
	// d6GraineTemoin fixe le tirage du temoin : deux executions rendent la MEME sortie.
	d6GraineTemoin = 20260828
	// d6SeuilRecouvrement / d6SeuilTemoin / d6PrincipalMin : le gate, recopie du protocole.
	d6SeuilRecouvrement = 0.80
	d6SeuilTemoin       = 0.50
)

// d6Trou est UN intervalle sans replication, avec ce que la reconstruction en a fait.
type d6Trou struct {
	debutUS, finUS uint64
	// classe vaut "porte", "ambigu", "retour" ou "inexplique".
	classe string
	// xuid est le porteur attribue ; zero quand il n'y en a pas.
	xuid uint64
	// distM est la distance au plus proche (MaxFloat64 : aucun bipede confrontable).
	distM float64
	// finPortageUS borne le portage : `finUS`, ou la mort du porteur si elle tombe avant.
	finPortageUS uint64
}

// dureeS rend la duree du PORTAGE attribue, en secondes.
func (t d6Trou) dureeS() float64 {
	if t.classe != "porte" || t.finPortageUS <= t.debutUS {
		return 0
	}
	return float64(t.finPortageUS-t.debutUS) / 1e6
}

// TestOddballPortageProximite — LA MESURE. Un film par processus.
func TestOddballPortageProximite(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", d4FilmEnv)
	}
	oracle, ok := d6Oracle(t)
	if !ok {
		return
	}
	g := filmproc.Arm("d6-portage", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	vies, socles, ok := d6ViesEtSocles(t, root, id)
	if !ok {
		return
	}
	dir := objChunkDir(root, id)
	wr, lay, ok := d6Bornes(t, root, id)
	if !ok {
		return
	}
	pont := objBridgeOf(t, root, id)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	pos, err := d6Positions(dir, wr, lay)
	if err != nil {
		t.Fatalf("%s : positions de bipede illisibles : %v", id, err)
	}
	tracks := indexBySlot(pos)
	t.Logf("%s : %d vie(s) libre(s), %d socle(s), %d slot(s) de bipede, %d slot(s) nomme(s), "+
		"%d mort(s), oracle sur %d joueur(s)",
		id, len(vies), len(socles), len(tracks), len(pont.SlotXUID), len(deaths), len(oracle))
	if len(pont.SlotXUID) < 2 {
		t.Logf("NON EXPLOITABLE %s : le pont nomme %d slot(s) — sans deux joueurs, la proximite "+
			"ne discrimine rien. NI POUR NI CONTRE.", id, len(pont.SlotXUID))
		return
	}

	trous := d6Reconstruit(vies, socles, tracks, pont, deaths, nil)
	d6PublieClasses(t, id, trous)
	d6PublieDistances(t, id, trous)
	d6PublieMort(t, id, vies, trous, tracks, pont)

	rec := d6ParJoueur(trous)
	rng := rand.New(rand.NewSource(d6GraineTemoin)) //nolint:gosec // temoin reproductible
	tem := d6ParJoueur(d6Reconstruit(vies, socles, tracks, pont, deaths, rng))
	d6Verdict(t, id, rec, tem, oracle)
}

// d6Oracle lit le fichier d'oracle fige : xuid decimal -> secondes de portage.
func d6Oracle(t *testing.T) (map[string]float64, bool) {
	t.Helper()
	path := os.Getenv(d6OracleEnv)
	if path == "" {
		t.Skipf("mesure non demandee : %s vide (oracle API fige)", d6OracleEnv)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur de la mesure
	if err != nil {
		t.Fatalf("oracle illisible (%s) : %v", path, err)
	}
	var all map[string]map[string]float64
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatalf("oracle invalide (%s) : %v", path, err)
	}
	o, ok := all[os.Getenv(d4FilmEnv)]
	if !ok || len(o) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun temps de portage API. NI POUR NI CONTRE.",
			os.Getenv(d4FilmEnv))
		return nil, false
	}
	return o, true
}

// d6ViesEtSocles rend les vies libres du crane et les socles `oddball_spawn` de la carte.
func d6ViesEtSocles(t *testing.T, root, id string) ([]flagFreeLife, []PointObjective, bool) {
	t.Helper()
	vies, ok := d4ViesLibres(t, root, id)
	if !ok {
		return nil, nil, false
	}
	if len(vies) < 2 {
		t.Logf("NON EXPLOITABLE %s : %d vie(s) libre(s) — sans deux vies il n'y a aucun trou. "+
			"NI POUR NI CONTRE.", id, len(vies))
		return nil, nil, false
	}
	sort.Slice(vies, func(i, j int) bool { return vies[i].T0US < vies[j].T0US })
	return vies, attMarqueurs(t, root, id, "oddball_spawn"), true
}

// d6Positions balaye les positions de bipede EN COORDONNEES MONDE.
//
// LES BORNES SONT OBLIGATOIRES, ET C'EST LA PIECE QU'IL NE FAUT PAS RATER. Les vies libres du
// crane sortent deja dequantifiees en metres (le balayage `ti=42` recoit `&wr`) ; balayer les
// bipedes en QUANTA — ce que fait le pont d'identite, qui n'a besoin que des slots et des
// instants — comparerait des metres a des quanta et rendrait des distances sans aucun sens, sans
// que rien ne le signale. La proximite exige LA MEME echelle des deux cotes.
func d6Positions(dir string, wr filmdec.Vec3Range, lay filmdec.I0Layout) ([]filmdec.BipedPosition, error) {
	release := filmdec.LockProcessDecode()
	defer release()
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &wr
	// Le découpage d'i0 vient du CATALOGUE quand il est complet (même doctrine que le
	// chemin world-object ; indispensable sur les cartes à plus de 2 régions — Live Fire,
	// lot C catalogues 2026-08-27). Un layout incomplet laisse l'auto-détection historique.
	if lay.Valid() {
		opt.Layout = &lay
	}
	return filmdec.ScanFilmBipedPositions(dir, opt)
}

// d6Bornes rend les bornes monde et le découpage d'i0 de la carte du film, sous verrou de
// processus.
func d6Bornes(t *testing.T, root, id string) (filmdec.Vec3Range, filmdec.I0Layout, bool) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	return attBornes(t, root, id)
}

// d6Reconstruit classe chaque trou et attribue son porteur.
//
// `rng` NON NIL bascule le TEMOIN : le porteur est tire au hasard parmi les joueurs nommes au
// lieu d'etre le plus proche. Meme chaine, meme cloture, meme metrique — c'est ce qui rend la
// comparaison honnete.
func d6Reconstruit(vies []flagFreeLife, socles []PointObjective, tracks map[uint32]slotTrack,
	pont objBridge, deaths []Death, rng *rand.Rand,
) []d6Trou {
	out := make([]d6Trou, 0, len(vies))
	for i := 0; i+1 < len(vies); i++ {
		fin, suivante := vies[i], vies[i+1]
		if suivante.T0US <= fin.T1US {
			continue
		}
		x, y := fin.Last()
		tr := d6Trou{debutUS: fin.T1US, finUS: suivante.T0US, distM: math.MaxFloat64}
		xuid, dist, second := d6PlusProche(tracks, pont, fin.T1US, x, y)
		tr.distM = dist
		switch {
		case xuid != 0 && dist <= d6RayonRamassageM && second-dist < d6AmbiguiteM:
			tr.classe = "ambigu"
		case xuid != 0 && dist <= d6RayonRamassageM:
			tr.classe, tr.xuid = "porte", xuid
		case d6NaitAuSocle(suivante, socles):
			tr.classe = "retour"
		default:
			tr.classe = "inexplique"
		}
		if tr.classe == "porte" && rng != nil {
			tr.xuid = d6TireAuHasard(pont, tr.xuid, rng)
		}
		tr.finPortageUS = d6FinPortage(tr, deaths, pont)
		out = append(out, tr)
	}
	return out
}

// d6PlusProche rend le xuid du bipede le plus proche du point, sa distance, et celle du DEUXIEME.
func d6PlusProche(tracks map[uint32]slotTrack, pont objBridge, atUS uint64, x, y float32,
) (uint64, float64, float64) {
	best, second, bestX := math.MaxFloat64, math.MaxFloat64, uint64(0)
	parXUID := map[uint64]float64{}
	for slot, tr := range tracks {
		xuid, nomme := pont.SlotXUID[slot]
		if !nomme {
			continue
		}
		p, ecart := tr.at(atUS)
		if ecart > d6EcartMaxMS*1000 {
			continue
		}
		d := math.Hypot(float64(p.X)-float64(x), float64(p.Y)-float64(y))
		// UN JOUEUR, PAS UNE VIE : le pool de slots reboucle a chaque reapparition, et deux
		// slots du meme joueur ne sont pas deux candidats.
		if cur, vu := parXUID[xuid]; !vu || d < cur {
			parXUID[xuid] = d
		}
	}
	for xuid, d := range parXUID {
		if d < best {
			best, second, bestX = d, best, xuid
			continue
		}
		if d < second {
			second = d
		}
	}
	return bestX, best, second
}

// d6NaitAuSocle dit si une vie libre nait a portee d'un socle `oddball_spawn`.
func d6NaitAuSocle(l flagFreeLife, socles []PointObjective) bool {
	if len(socles) == 0 {
		return false
	}
	x, y := l.First()
	for _, s := range socles {
		if math.Hypot(float64(x)-float64(s.Center.X), float64(y)-float64(s.Center.Y)) <= d6SocleM {
			return true
		}
	}
	return false
}

// d6TireAuHasard rend un joueur nomme AUTRE que celui donne — le temoin du protocole.
func d6TireAuHasard(pont objBridge, exclu uint64, rng *rand.Rand) uint64 {
	vus := map[uint64]bool{}
	for _, x := range pont.SlotXUID {
		if x != exclu {
			vus[x] = true
		}
	}
	if len(vus) == 0 {
		return exclu
	}
	cand := make([]uint64, 0, len(vus))
	for x := range vus {
		cand = append(cand, x)
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i] < cand[j] })
	return cand[rng.Intn(len(cand))]
}

// d6FinPortage borne le portage : la fin du trou, ou la MORT du porteur si elle tombe avant.
func d6FinPortage(tr d6Trou, deaths []Death, pont objBridge) uint64 {
	if tr.classe != "porte" {
		return tr.debutUS
	}
	fin := tr.finUS
	for _, d := range deaths {
		if d.XUID != tr.xuid {
			continue
		}
		at := uint64(d.TimeMS+pont.OffsetMS) * 1000
		if at > tr.debutUS && at < fin {
			fin = at
		}
	}
	return fin
}

// d6ParJoueur somme les durees de portage attribuees, par xuid decimal.
func d6ParJoueur(trous []d6Trou) map[string]float64 {
	out := map[string]float64{}
	for _, t := range trous {
		if t.classe != "porte" || t.xuid == 0 {
			continue
		}
		out[fmt.Sprintf("%d", t.xuid)] += t.dureeS()
	}
	return out
}
