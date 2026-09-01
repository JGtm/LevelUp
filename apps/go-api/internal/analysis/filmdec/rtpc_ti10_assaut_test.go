package filmdec

// rtpc_ti10_assaut_test.go — LE PARAMETRE QUI PILOTE LE SON, INTERROGE SUR L'ASSAUT.
//
// # LA QUESTION
//
// `managed-object-rtpc-component` (ti=10, instances i26 a i29) porte un parametre temps reel
// Wwise : un IDENTIFIANT de 32 bits qui nomme le canal, puis sa VALEUR sur 22 bits. Un RTPC est
// litteralement ce qui pilote un son. L'utilisateur a etabli sur pieces qu'un SON accompagne le
// compte a rebours de la bombe d'Assaut — le meme que l'apparition d'une nouvelle colline en
// KOTH. Le composant est PORTE depuis le lot C et n'a jamais ete interroge sur un film d'Assaut.
//
// # LE CRITERE, ECRIT AVANT LA MESURE — ET CE QUE CHAQUE ISSUE VOUDRAIT DIRE
//
// GATE 0 — PRESENCE. L'archetype ti=10 doit exister au registre des NEUF films d'Assaut, y
// recenser des vies aux images-cles, et rendre une bande de slots observes non vide.
//
//	nul sur les neuf films  -> NEGATIF NET. Les objets scriptes du mode n'existent pas en
//	                           Assaut, la piste RTPC se ferme, et le lot s'arrete la.
//	non nul                 -> on continue.
//
// GATE 1 — VALIDITE DE L'INSTRUMENT, PAR LE TEMOIN, ET IL PASSE AVANT L'ASSAUT. Sur KOTH
// (`7f1bbf06`), Strongholds (`696a9d7c`, `2ce58582`) et CTF (`cde26226`), le lot C a mesure que
// `managed-object-rtpc-component` porte 17 a 53 % des records ti=10.
//
//	le temoin rend des lectures  -> l'instrument voit ; son verdict sur l'Assaut compte.
//	le temoin ne rend RIEN       -> L'INSTRUMENT EST CASSE. Son silence sur l'Assaut ne vaut
//	                                alors rien, et c'est ce qu'il faut dire, pas un negatif.
//
// GATE 2 — INVENTAIRE ET COMPARAISON DES IDENTIFIANTS. L'identifiant est CONSTANT par composant :
// c'est lui, pas l'index de registre, qui nomme le canal. Trois issues, toutes informatives :
//
//	(a) un identifiant present en ASSAUT et ABSENT des quatre temoins -> canal de mode ; sa
//	    rampe est le candidat direct pour la jauge d'armement, on lui applique le gate 4 ;
//	(b) les memes identifiants partout (dont `0x06854540` / `0x7CBF0066`, mesures en
//	    Strongholds au lot C) -> RTPC generiques d'objet gere, NEGATIF PROPRE sur la piste ;
//	(c) aucun identifiant lu en Assaut -> NEGATIF NET, la piste se ferme.
//
// LE FILTRE DE ROBUSTESSE EST FIGE ICI, AVANT LA MESURE, et il vient d'une lecon du lot C : sur
// ti=13, 811 identifiants rares s'etaient reveles etre du bruit d'ancrage (22,6 % de records
// hors grammaire). N'est donc RETENU qu'un identifiant porte par AU MOINS DEUX SLOTS et vu sur
// AU MOINS DEUX FILMS. Les autres sont publies, mais dans la colonne « rares ».
//
// GATE 3 — EGALITE AVEC LES IDENTIFIANTS ATTENDUS. La reconnaissance Ghidra a rendu la fabrique
// d'identifiants (FNV-1 32 bits sur le nom minuscule, `FUN_140871f08`) et onze noms d'evenements
// d'armement de la banque `sb_004_mod_mp_assault`. Ils sont testes PAR EGALITE.
//
//	une egalite  -> le canal est NOMME, pas devine. C'est infiniment plus fort qu'une
//	                correlation, et cela clot la question du sens.
//	aucune       -> NE PROUVE RIEN. Ce sont des identifiants d'EVENEMENTS Wwise, pas de RTPC ;
//	                ils n'ont aucune raison mecanique d'apparaitre dans ce composant. On les
//	                teste parce que c'est gratuit et qu'une prise vaudrait tout le lot.
//
// GATE 4 — LE CRITERE DU CHANTIER, contre les 28 explosions datees de l'oracle.
//
//	COUVERTURE  au moins une MONTEE close avant CHACUNE des 28 explosions ;
//	CONSTANCE   dispersion des delais <= 20 % de la mediane. Mesure figee : (p75 - p25) /
//	            mediane, l'ecart interquartile rapporte a la mediane (`ti12DispersionMax`) ;
//	SENS        delai positif, sous 120 000 ms (`ti12SensMaxMS`).
//
// LES TROIS ENSEMBLE -> CANDIDAT : la fin de montee date le moment ou le parametre se fige, donc
// l'armement. UN SEUL QUI MANQUE -> NEGATIF SUR CE CANAL, publie tel quel.
//
// # LA DEFINITION D'UNE MONTEE, FIGEE AVANT LA MESURE
//
// MONTEE = suite maximale d'echantillons CONSECUTIFS d'un meme couple (slot, identifiant),
// ordonnee dans le temps, non decroissante, d'au moins `ti12MonteeMinEch` echantillons et
// d'amplitude au moins egale au SEIZIEME de l'etendue observee de ce couple, avec un plancher
// d'un quantum. Sa FIN est l'instant de son dernier echantillon.
//
// POURQUOI UN SEUIL RELATIF ET NON ABSOLU, contrairement a ti=12. La plage d'un RTPC est
// [-10000, +10000] sur 22 bits, mais le SENS de la valeur est libre : le script peut y ecrire
// une fraction de 0 a 1, un pourcentage de 0 a 100, ou des secondes. Un seuil absolu en quanta
// choisirait donc l'echelle a la place de la donnee. Le seizieme de l'etendue est une REGLE,
// fixee d'avance, qui s'adapte a l'echelle sans etre ajustee sur le resultat.
//
// # L'HORLOGE
//
// Les lectures sont datees sur l'horloge du MANIFESTE (`start_ms` par chunk, `filmcache`), la
// MEME base que `objectiveevents.StatRecords` — donc que l'oracle. Les delais absolus sont alors
// lisibles, et publies. LE VERDICT RESTE CELUI DE LA DISPERSION : un decalage constant de
// conversion ne la ferait pas bouger.
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
//	go test ./internal/analysis/filmdec/ -run RtpcTi10 -v -timeout 120m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// ti10MonteeDiviseur fixe le seuil d'amplitude d'une montee : le SEIZIEME de l'etendue observee
// du couple (slot, identifiant). Fige avant la mesure — voir l'en-tete.
const ti10MonteeDiviseur = 16

// ti10Cle identifie une SERIE : un canal RTPC sur un objet gere. Le couple, jamais l'un des
// deux seuls — un meme identifiant vit sur plusieurs objets, et un meme objet porte jusqu'a
// quatre canaux.
type ti10Cle struct {
	slot uint32
	id   uint32
}

// ti10Ech est un echantillon date d'un canal RTPC (quantum de 22 bits).
type ti10Ech struct {
	tMS int32
	q   uint32
}

// ti10Montee est une montee au sens du gate 4.
type ti10Montee struct {
	cle            ti10Cle
	debutMS, finMS int32
	bas, haut      uint32
	n              int
}

// ti10IDStat resume ce qu'UN identifiant a rendu sur UN film.
type ti10IDStat struct {
	lectures, chainees int
	// sansValeur compte les lectures a identifiant nul (emplacement libere) — il n'y en a que
	// pour l'identifiant 0, par construction du lecteur.
	sansValeur int
	slots      map[uint32]bool
	comps      map[int16]bool
	qMin, qMax uint32
	distinct   map[uint32]bool
	montees    int
}

func nouveauTi10IDStat() *ti10IDStat {
	return &ti10IDStat{slots: map[uint32]bool{}, comps: map[int16]bool{},
		distinct: map[uint32]bool{}, qMin: ^uint32(0)}
}

// ti10FilmBilan porte ce qu'un film a rendu, une fois ses lectures digerees et relachees.
type ti10FilmBilan struct {
	id, mode           string
	sc                 *ti10Scan
	lectures, chainees int
	nuls               int
	seriesN            int
	ids                map[uint32]*ti10IDStat
	montees            []ti10Montee // toutes, triees par finMS
	monteesChainees    []ti10Montee
	instants           map[uint32][]int32 // par identifiant, tries
	tousInstants       []int32
	extraits           []string
	duree              time.Duration
}

// TestRtpcTi10AssautIdentifiants interroge ti=10 i26..i29 sur les neuf films d'Assaut et les
// quatre temoins.
func TestRtpcTi10AssautIdentifiants(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestRtpcTi10AssautIdentifiants", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()
	release := LockProcessDecode()
	defer release()

	bilans := make([]*ti10FilmBilan, 0, len(ti11Corpus))
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		b := ti10MesurerFilm(t, dir, f.id, f.mode)
		if b == nil {
			continue
		}
		ti10JournalFilm(t, b)
		bilans = append(bilans, b)
	}
	ti10Gate0(t, bilans)
	ti10Gate1(t, bilans)
	inv := ti10Gate2(t, bilans)
	ti10Gate3(t, inv)
	ti10Gate4(t, bilans, inv)
}

// ti10MesurerFilm balaye UN film et digere ses lectures. Les lectures brutes sont relachees
// avant de rendre : seul le bilan survit d'un film au suivant (garde memoire).
func ti10MesurerFilm(t *testing.T, dir, id, mode string) *ti10FilmBilan {
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
	sc, err := ti10ScanFilm(dir, clk)
	if err != nil {
		t.Logf("%-9s %-26s balayage impossible (%v) — saute", id, mode, err)
		return nil
	}
	b := &ti10FilmBilan{id: id, mode: mode, sc: sc, lectures: len(sc.Reads),
		duree: time.Since(debut)}
	b.digerer(sc.Reads)
	sc.Reads = nil
	return b
}

// digerer range les lectures en series par couple (slot, identifiant), en tire les statistiques
// par identifiant, les montees et les instants. Les lectures brutes ne survivent pas a l'appel.
func (b *ti10FilmBilan) digerer(reads []ti10Read) {
	b.ids = map[uint32]*ti10IDStat{}
	series, seriesCh := map[ti10Cle][]ti10Ech{}, map[ti10Cle][]ti10Ech{}
	inst := map[uint32]map[int32]bool{}
	tous := map[int32]bool{}
	for _, r := range reads {
		st, ok := b.ids[r.ID]
		if !ok {
			st = nouveauTi10IDStat()
			b.ids[r.ID] = st
		}
		st.lectures++
		st.slots[r.Slot] = true
		st.comps[r.Comp] = true
		if r.Chained {
			st.chainees++
			b.chainees++
		}
		tous[r.TMS] = true
		if inst[r.ID] == nil {
			inst[r.ID] = map[int32]bool{}
		}
		inst[r.ID][r.TMS] = true
		if !r.HasQ {
			st.sansValeur++
			b.nuls++
			continue
		}
		ti10Borner(st, r.Q)
		cle := ti10Cle{r.Slot, r.ID}
		series[cle] = append(series[cle], ti10Ech{r.TMS, r.Q})
		if r.Chained {
			seriesCh[cle] = append(seriesCh[cle], ti10Ech{r.TMS, r.Q})
		}
	}
	b.seriesN = len(series)
	b.montees = ti10Digerer(series)
	b.monteesChainees = ti10Digerer(seriesCh)
	for _, m := range b.montees {
		b.ids[m.cle.id].montees++
	}
	b.instants, b.tousInstants = ti10Instants(inst, tous)
	b.extraits = ti10Extraits(series)
}

// ti10Borner tient les bornes et le compte de valeurs distinctes d'un identifiant. Le compte de
// distinctes est PLAFONNE : un canal bruite en produirait des millions, et seule la question
// « fige ou pas » se joue ici.
func ti10Borner(st *ti10IDStat, q uint32) {
	if q < st.qMin {
		st.qMin = q
	}
	if q > st.qMax {
		st.qMax = q
	}
	if len(st.distinct) < 4096 {
		st.distinct[q] = true
	}
}

// ti10Instants rend les instants de lecture par identifiant, tries, et l'union triee.
func ti10Instants(par map[uint32]map[int32]bool, tous map[int32]bool) (map[uint32][]int32, []int32) {
	out := make(map[uint32][]int32, len(par))
	for id, m := range par {
		s := make([]int32, 0, len(m))
		for t := range m {
			s = append(s, t)
		}
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		out[id] = s
	}
	u := make([]int32, 0, len(tous))
	for t := range tous {
		u = append(u, t)
	}
	sort.Slice(u, func(i, j int) bool { return u[i] < u[j] })
	return out, u
}

// ti10Digerer trie chaque serie et en extrait les montees, rendues TRIEES par instant de fin.
func ti10Digerer(series map[ti10Cle][]ti10Ech) []ti10Montee {
	var montees []ti10Montee
	for cle, s := range series {
		sort.Slice(s, func(i, j int) bool { return s[i].tMS < s[j].tMS })
		montees = append(montees, ti10Montees(cle, s)...)
	}
	// TRI DETERMINISTE : deux montees peuvent finir a la MEME milliseconde (un objet gere porte
	// jusqu'a quatre canaux, et les objets vont par paires, un par camp). Departager par slot
	// puis par identifiant rend l'ordre stable d'une passe a l'autre.
	sort.Slice(montees, func(i, j int) bool {
		if montees[i].finMS != montees[j].finMS {
			return montees[i].finMS < montees[j].finMS
		}
		if montees[i].cle.slot != montees[j].cle.slot {
			return montees[i].cle.slot < montees[j].cle.slot
		}
		return montees[i].cle.id < montees[j].cle.id
	})
	return montees
}

// ti10Montees decoupe une serie triee en montees au sens du gate 4.
func ti10Montees(cle ti10Cle, s []ti10Ech) []ti10Montee {
	seuil := ti10SeuilAmplitude(s)
	var out []ti10Montee
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1].q >= s[j].q {
			j++
		}
		n := j - i + 1
		if n >= ti12MonteeMinEch && s[j].q-s[i].q >= seuil {
			out = append(out, ti10Montee{cle, s[i].tMS, s[j].tMS, s[i].q, s[j].q, n})
		}
		if j == i {
			i++
			continue
		}
		i = j
	}
	return out
}

// ti10SeuilAmplitude rend le seizieme de l'etendue de la serie, avec un plancher d'un quantum.
func ti10SeuilAmplitude(s []ti10Ech) uint32 {
	if len(s) == 0 {
		return 1
	}
	lo, hi := s[0].q, s[0].q
	for _, e := range s {
		if e.q < lo {
			lo = e.q
		}
		if e.q > hi {
			hi = e.q
		}
	}
	if seuil := (hi - lo) / ti10MonteeDiviseur; seuil > 1 {
		return seuil
	}
	return 1
}
