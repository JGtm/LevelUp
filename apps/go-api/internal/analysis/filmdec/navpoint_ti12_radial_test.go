package filmdec

// navpoint_ti12_radial_test.go — LE DISQUE QUE LE JOUEUR VOIT, INTERROGE SUR L'ASSAUT.
//
// # LA QUESTION
//
// `managed-navpoint-radial-progress` (ti=12 i14) est le disque qui se remplit autour d'un
// marqueur d'objectif a l'ecran. Il est PORTE depuis le lot C (R(8), lecteur `FUN_140fc8d14`,
// publie par `SetNavpointHook`) et il n'a jamais ete interroge sur un film d'ASSAUT. Si la jauge
// d'armement de la bombe est peinte sur le marqueur du site, elle passe par ce canal.
//
// # LE CRITERE, ECRIT AVANT LA MESURE — ET CE QUE CHAQUE ISSUE VOUDRAIT DIRE
//
// GATE 0 — PRESENCE. L'archetype ti=12 doit exister au registre des NEUF films d'Assaut, y
// recenser des vies aux images-cles, et rendre une bande de slots observes non vide.
//
//	nul sur les neuf films  -> NEGATIF NET. Le disque radial n'existe pas en Assaut, la piste
//	                           se ferme, et le lot s'arrete la.
//	non nul                 -> on continue.
//
// GATE 1 — VALIDITE DE L'INSTRUMENT, PAR LE TEMOIN, ET IL PASSE AVANT L'ASSAUT. Sur KOTH
// (`7f1bbf06`) et Strongholds (`696a9d7c`, `2ce58582`), la progression radiale EXISTE A L'ECRAN
// et le lot C l'a mesuree a 74,7 % et 93,1 % des records ti=12 des deux Strongholds.
//
//	le temoin rend des lectures  -> l'instrument voit ; son verdict sur l'Assaut compte.
//	le temoin ne rend RIEN       -> L'INSTRUMENT EST CASSE. Son silence sur l'Assaut ne vaut
//	                               alors rien, et c'est ce qu'il faut dire, pas un negatif.
//
// GATE 2 — FORME. Une jauge MONTE. Definitions figees AVANT la mesure :
//
//	MONTEE  = suite maximale d'echantillons CONSECUTIFS d'un meme slot, ordonnee dans le temps,
//	          non decroissante, d'au moins ti12MonteeMinEch echantillons et d'amplitude au moins
//	          ti12MonteeMinAmpl quanta. Sa FIN est l'instant de son dernier echantillon.
//
//	aucune montee, ou valeurs constantes / uniformes sur 8 bits -> le canal est allume mais ne
//	                                                              porte pas de progression.
//	des montees structurees                                     -> gate 3.
//
// GATE 3 — LE CRITERE DU CHANTIER, contre les 28 explosions datees d'`a5Explosions`.
//
//	COUVERTURE  au moins une MONTEE close avant CHACUNE des 28 explosions ;
//	CONSTANCE   dispersion des delais <= 20 % de la mediane. Mesure figee : (p75 - p25) /
//	            mediane, l'ecart interquartile rapporte a la mediane ;
//	SENS        delai positif, sous 120 000 ms.
//
// LES TROIS ENSEMBLE -> CANDIDAT : la fin de montee date le moment ou la jauge se fige, donc
// l'armement. UN SEUL QUI MANQUE -> NEGATIF SUR CE CANAL, publie tel quel.
//
// # L'HORLOGE
//
// La doctrine du chantier dit de ne juger que la DISPERSION, l'ecart moteur/manifeste etant
// inconnu et constant. Cet instrument fait mieux quand il le peut : il date les lectures sur
// l'horloge du MANIFESTE (`start_ms` par chunk, `filmcache`), la MEME base que
// `objectiveevents.StatRecords` — donc que l'oracle. Les delais absolus sont alors lisibles et
// sont publies. LE VERDICT RESTE CELUI DE LA DISPERSION : si la conversion d'horloge etait
// fausse d'un decalage constant, la dispersion, elle, ne bougerait pas.
//
// # LA PARTITION DU CORPUS, ET POURQUOI ELLE N EST PAS UN REPENTIR
//
// Le corpus se coupe en deux DEPUIS LA VEILLE : trois films One Bomb (multi-manches) et six
// films Neutral Bomb, decoupage ecrit dans `.ai/V7.5/ETAT_ASSAUT_2026-08-31.md` §1.b et §1.e.
// Le critere est applique aux deux moities EN PLUS du corpus entier, jamais a sa place — le
// verdict primaire reste celui des 28 explosions.
//
// # LE SECOND JEU DE CHIFFRES : LES LECTURES CHAINEES
//
// Tout est publie DEUX FOIS — sur toutes les lectures, et sur les seules lectures CHAINEES (le
// record se termine sur un en-tete valide). Decide avant la mesure, pour qu'aucun des deux ne
// soit choisi apres coup.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee, UN SEUL
// decodage a la fois (`LockProcessDecode`).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run NavpointTi12Radial -v -timeout 90m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// Les seuils de la definition d'une MONTEE, figes avant la mesure. L'amplitude est en quanta de
// la plage R(8) : 16 quanta = 1/16 de la course du disque.
const (
	ti12MonteeMinEch  = 3
	ti12MonteeMinAmpl = 16
	// ti12SensMaxMS borne le delai admissible entre une montee et l'explosion qu'elle annonce.
	ti12SensMaxMS = 120_000
	// ti12DispersionMax est le plafond de (p75 - p25) / mediane.
	ti12DispersionMax = 0.20
)

// ti12Ech est un echantillon date de la progression radiale.
type ti12Ech struct {
	tMS int32
	q   uint8
}

// ti12Montee est une montee au sens du gate 2.
type ti12Montee struct {
	slot           uint32
	debutMS, finMS int32
	bas, haut      uint8
	n              int
}

// ti12FilmBilan porte ce qu'un film a rendu, une fois ses lectures digerees et relachees.
type ti12FilmBilan struct {
	id, mode                          string
	sc                                *ti12Scan
	lectures, chainees                int
	slotsPorteurs                     int
	montees, monteesChainees          []ti12Montee
	histo                             [8]int // histogramme des quanta, par huitieme de plage
	instants, instantsChaines         []int32
	extraits                          []string
	valeursDistinctes, slotsVariables int
	duree                             time.Duration
}

// ti12Horloge rend l'horloge du film (start_ms par chunk, lu au manifeste) et dit si elle
// existe. Variante NON FATALE de `zcLoadClock` : un manifeste manquant fait sauter le film, pas
// la mesure entiere.
func ti12Horloge(dir string) (zcClock, bool) {
	root, short := filepath.Dir(filepath.Dir(dir)), filepath.Base(dir)
	src, ok, err := filmcache.Open(root, short)
	if err != nil || !ok {
		return zcClock{}, false
	}
	clk := zcClock{startMS: map[int]int{}}
	for _, m := range src.Chunks() {
		clk.startMS[m.Index] = m.StartMS
	}
	return clk, len(clk.startMS) > 0
}

// TestNavpointTi12RadialAssaut interroge ti=12 i14 sur les neuf films d'Assaut et les quatre
// temoins.
func TestNavpointTi12RadialAssaut(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestNavpointTi12RadialAssaut", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()
	release := LockProcessDecode()
	defer release()

	bilans := make([]*ti12FilmBilan, 0, len(ti11Corpus))
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		b := ti12MesurerFilm(t, dir, f.id, f.mode)
		if b == nil {
			continue
		}
		ti12JournalFilm(t, b)
		bilans = append(bilans, b)
	}
	ti12Gate0(t, bilans)
	ti12Gate1(t, bilans)
	ti12Gate2(t, bilans)
	ti12Gate3(t, bilans)
}

// ti12MesurerFilm balaye UN film et digere ses lectures. Les lectures brutes sont relachees
// avant de rendre : seul le bilan survit d'un film au suivant (garde memoire).
func ti12MesurerFilm(t *testing.T, dir, id, mode string) *ti12FilmBilan {
	t.Helper()
	if CountFilmChunks(dir) == 0 {
		t.Logf("%-9s %-26s AUCUN CHUNK — film absent du cache, saute", id, mode)
		return nil
	}
	clk, ok := ti12Horloge(dir)
	if !ok {
		t.Logf("%-9s %-26s MANIFESTE ABSENT — sans horloge la lecture n'est pas confrontable, saute",
			id, mode)
		return nil
	}
	debut := time.Now()
	sc, err := ti12ScanFilm(dir, clk)
	if err != nil {
		t.Logf("%-9s %-26s balayage impossible (%v) — saute", id, mode, err)
		return nil
	}
	b := &ti12FilmBilan{id: id, mode: mode, sc: sc, lectures: len(sc.Reads),
		duree: time.Since(debut)}
	series, seriesCh := map[uint32][]ti12Ech{}, map[uint32][]ti12Ech{}
	for _, r := range sc.Reads {
		series[r.Slot] = append(series[r.Slot], ti12Ech{r.TMS, r.Q})
		b.histo[r.Q>>5]++
		if r.Chained {
			b.chainees++
			seriesCh[r.Slot] = append(seriesCh[r.Slot], ti12Ech{r.TMS, r.Q})
		}
	}
	sc.Reads = nil
	b.slotsPorteurs = len(series)
	b.montees, b.instants = ti12Digerer(series)
	b.monteesChainees, b.instantsChaines = ti12Digerer(seriesCh)
	b.valeursDistinctes, b.slotsVariables = ti12Variabilite(series)
	b.extraits = ti12Extraits(series)
	return b
}

// ti12Digerer trie chaque serie, en extrait les montees, et rend l'ensemble TRIE ET DEDOUBLE des
// instants de lecture du film — le denominateur de la couverture faible (« une lecture avant
// l'explosion », a defaut d'une montee).
func ti12Digerer(series map[uint32][]ti12Ech) ([]ti12Montee, []int32) {
	var montees []ti12Montee
	vus := map[int32]bool{}
	for slot, s := range series {
		sort.Slice(s, func(i, j int) bool { return s[i].tMS < s[j].tMS })
		for _, e := range s {
			vus[e.tMS] = true
		}
		montees = append(montees, ti12Montees(slot, s)...)
	}
	// TRI DETERMINISTE : deux montees peuvent finir a la MEME milliseconde (les navpoints vont
	// par paires, une par camp, et portent le meme anneau). Departager par slot rend l'ordre
	// stable d'une passe a l'autre ; sans cela, seul le NOM du slot rapporte changeait, jamais
	// un delai — ceux-ci se calculent sur `finMS`, qui est la cle de tri.
	sort.Slice(montees, func(i, j int) bool {
		if montees[i].finMS != montees[j].finMS {
			return montees[i].finMS < montees[j].finMS
		}
		return montees[i].slot < montees[j].slot
	})
	inst := make([]int32, 0, len(vus))
	for t := range vus {
		inst = append(inst, t)
	}
	sort.Slice(inst, func(i, j int) bool { return inst[i] < inst[j] })
	return montees, inst
}

// ti12Montees decoupe une serie triee en montees au sens du gate 2.
func ti12Montees(slot uint32, s []ti12Ech) []ti12Montee {
	var out []ti12Montee
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1].q >= s[j].q {
			j++
		}
		n := j - i + 1
		if n >= ti12MonteeMinEch && int(s[j].q)-int(s[i].q) >= ti12MonteeMinAmpl {
			out = append(out, ti12Montee{slot, s[i].tMS, s[j].tMS, s[i].q, s[j].q, n})
		}
		if j == i {
			i++
			continue
		}
		i = j
	}
	return out
}

// ti12Variabilite rend le nombre de valeurs distinctes toutes series confondues, et le nombre de
// slots dont la valeur n'est PAS constante. Une jauge varie ; un canal fige ne dit rien.
func ti12Variabilite(series map[uint32][]ti12Ech) (int, int) {
	vals, variables := map[uint8]bool{}, 0
	for _, s := range series {
		if len(s) == 0 {
			continue
		}
		first, bouge := s[0].q, false
		for _, e := range s {
			vals[e.q] = true
			if e.q != first {
				bouge = true
			}
		}
		if bouge {
			variables++
		}
	}
	return len(vals), variables
}
