package filmdec

// objectif_ti11_minuteurs_test.go — ti=11 i0 `managed-objective-timers-component`, LES DEUX
// VALEURS DE SEPT BITS, INTERROGEES POUR LA PREMIERE FOIS.
//
// # CE QUE LE CANAL EST, ET POURQUOI IL VIENT MAINTENANT
//
// `consumeObjectiveTimers` lit DEUX fois sept bits (FUN_142edbac8 -> FUN_142ed15a0, qui ecrit
// `valeur + 1`) ; `ObjectiveTimerValue(q) = int(q) - 1`, donc le zero du flux vaut « pas de
// minuteur ». Le composant est PRESENT DANS 100 % des records porteurs de la jauge (recensement
// des masques du 2026-09-01) et il n'a jamais ete interprete. Il est aussi le PREMIER composant
// lu apres l'en-tete : aucune largeur fausse d'un autre composant ne peut le decaler.
//
// # LES DEUX LECTURES QUI S'AFFRONTENT, ECRITES AVANT LA MESURE
//
//	A. UNE DUREE. Sept bits de compte a rebours. Alors la valeur DECROIT au fil du temps, et
//	   l'instant ou une descente COMMENCE date le debut de l'armement de la bombe.
//	B. UN INDEX DE MINUTEUR. La reconnaissance Ghidra du 2026-08-31 lit FUN_1410d9088 comme sept
//	   bits BRUTS moins un — sans min, sans max, sans dequantification (le lecteur de reel
//	   quantifie de ce protocole, FUN_1406d84b4, n'est PAS appele). Le domaine legal est alors
//	   {-1} U [0, 63] (les fentes du bassin replique, masque R(64)) U {65, 66, 67} (les trois
//	   minuteurs reserves : manche, mort subite, periode de grace). UN INDEX NE DECROIT PAS.
//
// Les deux lectures sont exclusives et cet instrument les separe : si A tient, il y a des
// DESCENTES ; si B tient, il n'y en a aucune et le taux de LEGALITE frise 100 %.
//
// # GATE 0 — LEGALITE : L'ORACLE D'ANCRAGE QUE `objective_scan.go` RECLAME, ET IL EST GRATUIT
//
// Le domaine legal ne couvre que 68 des 128 valeurs encodables sur sept bits.
//
//	taux proche de 100 %  -> l'ancrage et la largeur sont justes ; i0 devient l'oracle d'ancrage
//	                         de ti=11, et la lecture B est confirmee du meme coup.
//	taux proche de 53 %   -> les sept bits sont UNIFORMES : la fenetre est mal posee, et aucun
//	                         chiffre de ti=11 ne vaut, ni ici ni ailleurs. C'est alors le
//	                         resultat du lot, et il ferme le canal.
//	taux intermediaire    -> melange de vrais records et de faux ancrages ; la voie CHAINEE
//	                         tranche (elle doit alors etre nettement plus legale que les autres).
//
// # GATE 1 — TEMOIN, ET IL PASSE AVANT TOUT VERDICT SUR L'ASSAUT
//
// Les quatre films non-Assaut de `ti11Corpus` (CTF `cde26226`, KOTH `7f1bbf06`, deux
// Strongholds) au meme instrument. Le temoin CTF porte 186 records ti=11 au recensement.
//
//	le temoin rend des lectures            -> l'instrument voit ; son verdict sur l'Assaut compte.
//	le temoin ne rend rien                 -> instrument casse : son silence ne vaut rien.
//	le temoin a la MEME forme que l'Assaut -> le canal est generique, il ne parle pas de la bombe.
//
// # GATE 2 — SEMANTIQUE (mission, point 1)
//
// Distribution des deux valeurs, VOIE PAR VOIE, avec les classes du domaine, l'ordre des deux
// valeurs (l'une est-elle toujours >= l'autre ?) et le nombre de slots dont la valeur BOUGE.
//
//	seulement -1 et/ou 65 (chrono de manche) -> NEGATIF NET : i0 ne parle pas de la bombe.
//	un index de bassin (0..63) sur un objectif d'Assaut -> le compte a rebours est dans le film.
//
// # GATE 3 — LE CRITERE DU CHANTIER (mission, points 2 et 3)
//
// DESCENTE, definie AVANT la mesure : suite maximale d'echantillons CONSECUTIFS d'un meme slot
// et d'un meme champ, ordonnee dans le temps, NON CROISSANTE, d'au moins `mntDescenteMinEch`
// echantillons et d'amplitude au moins `mntDescenteMinAmpl` quanta. Une valeur ABSENTE (-1)
// n'est pas une valeur : elle COUPE la suite, elle ne la prolonge pas. Le DEPART d'une descente
// est l'instant de son premier echantillon — le candidat « debut d'armement ».
//
//	COUVERTURE  au moins une descente commencee avant CHACUNE des 28 explosions d'`a5Explosions` ;
//	CONSTANCE   dispersion des delais <= 20 % de la mediane, mesuree (p75 - p25) / mediane ;
//	SENS        delai positif, sous 120 000 ms.
//
// LES TROIS ENSEMBLE -> CANDIDAT. UN SEUL QUI MANQUE -> NEGATIF SUR CE CANAL, publie tel quel.
//
// # L'HORLOGE, ET CE QUE CET INSTRUMENT EN MESURE EN PASSANT
//
// La doctrine dit que l'ecart entre l'horloge MOTEUR (les canaux ECS) et celle du MANIFESTE (les
// explosions) est inconnu mais CONSTANT, et de ne juger que la dispersion. Cet instrument date
// ses lectures sur l'horloge du MANIFESTE (recette de `navpoint_ti12_scan_test.go` : base = le
// premier paquet delta du chunk, `start_ms` lu au manifeste) et PUBLIE l'ecart chunk par chunk :
// s'il est constant, la doctrine est verifiee ; s'il ne l'est pas, elle est a corriger.
//
// LES CHIFFRES DE LA PASSE UNIQUE (2026-09-01) SONT DANS L EN-TETE DE
// `objectif_ti11_minuteurs_verdict_test.go`, a cote des portes qui les produisent.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee, UN SEUL
// decodage a la fois (`LockProcessDecode`).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Minuteurs -v -timeout 90m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// Le domaine legal d'un index de minuteur, tel que FUN_140fe957c le rend.
const (
	mntAbsent      = -1 // le zero du flux : « pas de minuteur »
	mntBassinMax   = 63 // derniere fente du bassin replique (masque R(64) de ti=0 i15)
	mntManche      = 65 // Engine_GetRoundTimer()
	mntMortSubite  = 66 // Engine_GetOvertimeTimer()
	mntGrace       = 67 // Engine_GetGracePeriodTimer()
	mntValeursSept = 128
	mntLegales     = 68 // {-1} U [0,63] U {65,66,67}
)

// Les seuils du gate 3, figes avant la mesure.
const (
	mntDescenteMinEch  = 3
	mntDescenteMinAmpl = 4
	mntSensMaxMS       = 120_000
	mntDispersionMax   = 0.20
)

// Les trois seaux de lecture. Decides avant la mesure, et TOUS publies : la voie chainee est le
// seul temoin de fiabilite par lecture, et la publier a cote des deux autres empeche de choisir
// le seau le plus flatteur apres coup.
const (
	mntVoieCle = iota
	mntVoieDelta
	mntVoieDeltaChainee
	mntVoies
)

var mntNomVoie = [mntVoies]string{"IMAGE-CLE", "DELTA", "DELTA CHAINEE"}

// mntEch est UNE lecture de i0 : les deux valeurs decodees, datees sur l'horloge du MANIFESTE.
type mntEch struct {
	tMS int32
	v   [2]int
}

// mntDescente est une descente au sens du gate 3.
type mntDescente struct {
	slot           uint32
	champ          int
	debutMS, finMS int32
	haut, bas      int
	n              int
}

// mntVoieBilan agrege UNE voie d'UN film.
type mntVoieBilan struct {
	lectures       int
	legaux         [2]int
	legalPaire     int
	histo          [2]map[int]int
	series         map[uint32][]mntEch
	slots          int
	slotsVariables [2]int
	ordre          [3]int // v0 > v1, v0 == v1, v0 < v1
	descentes      []mntDescente
	instants       []int32
	extraits       []string
}

func mntNouvelleVoie() *mntVoieBilan {
	return &mntVoieBilan{histo: [2]map[int]int{{}, {}}, series: map[uint32][]mntEch{}}
}

// mntBilan porte ce qu'un film a rendu, une fois ses lectures digerees et relachees.
type mntBilan struct {
	id, mode    string
	voies       [mntVoies]*mntVoieBilan
	sc          ObjectiveScan
	sansHorloge int
	sansSecond  int
	ecarts      []int // ecart moteur -> manifeste, chunk par chunk (ms)
	ambigus     int
	duree       time.Duration
}

// mntHorloge est la table « horodatage moteur -> milliseconde de manifeste » d'un film.
type mntHorloge struct {
	ms      map[uint64]int32
	ecarts  []int
	ambigus int
}

// mntLegal dit si une valeur decodee appartient au domaine d'un index de minuteur.
func mntLegal(v int) bool {
	return v == mntAbsent || (v >= 0 && v <= mntBassinMax) ||
		v == mntManche || v == mntMortSubite || v == mntGrace
}

// mntChargerHorloge construit la table des horodatages du film, chunk par chunk.
//
// MEME RECETTE QUE `ScanFilmNavpointRadial` : la base est le PREMIER PAQUET DELTA du chunk, et `start_ms`
// vient du manifeste — c'est ce qui met les lectures sur la MEME base que l'oracle des
// explosions. L'ecart par chunk est conserve : c'est lui qui dit si l'horloge moteur est affine
// a celle du manifeste, comme la doctrine du chantier le suppose.
func mntChargerHorloge(dir string) (mntHorloge, bool) {
	h := mntHorloge{ms: map[uint64]int32{}}
	root, court := filepath.Dir(filepath.Dir(dir)), filepath.Base(dir)
	src, ok, err := filmcache.Open(root, court)
	if err != nil || !ok {
		return h, false
	}
	debut := map[int]int{}
	for _, m := range src.Meta() {
		debut[m.Index] = m.StartMS
	}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		pks := WalkPackets(data)
		base, aBase := navpointRadialBaseChunk(pks)
		start, aStart := debut[c]
		if !aBase || !aStart {
			continue
		}
		h.ecarts = append(h.ecarts, start-int(base/1000))
		for _, pk := range pks {
			ms := int32(start + int((int64(pk.TimestampUS)-int64(base))/1000))
			if vu, deja := h.ms[pk.TimestampUS]; deja && vu != ms {
				h.ambigus++
				continue
			}
			h.ms[pk.TimestampUS] = ms
		}
	}
	return h, len(h.ms) > 0
}

// TestObjectifTi11Minuteurs interroge ti=11 i0 sur les neuf films d'Assaut et les quatre temoins.
func TestObjectifTi11Minuteurs(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Minuteurs", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()
	release := LockProcessDecode()
	defer release()

	bilans := make([]*mntBilan, 0, len(ti11Corpus))
	for _, f := range ti11Corpus {
		b := mntMesurerFilm(t, cache, f.id, f.mode)
		if b == nil {
			continue
		}
		mntJournalFilm(t, b)
		bilans = append(bilans, b)
	}
	mntGate0(t, bilans)
	mntGate1(t, bilans)
	mntGate2(t, bilans)
	mntGate3(t, bilans)
}

// mntMesurerFilm balaye UN film et digere ses lectures. Les lectures brutes sont relachees avant
// de rendre : seul le bilan survit d'un film au suivant (garde memoire).
func mntMesurerFilm(t *testing.T, cache, id, mode string) *mntBilan {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	if CountFilmChunks(dir) == 0 {
		t.Logf("%-9s %-26s AUCUN CHUNK — film absent du cache, saute", id, mode)
		return nil
	}
	h, ok := mntChargerHorloge(dir)
	if !ok {
		t.Logf("%-9s %-26s MANIFESTE ABSENT — sans horloge la lecture n'est pas confrontable",
			id, mode)
		return nil
	}
	depart := time.Now()
	sc, err := ScanFilmObjectives(dir)
	if err != nil {
		t.Logf("%-9s %-26s balayage impossible (%v) — saute", id, mode, err)
		return nil
	}
	b := &mntBilan{id: id, mode: mode, ecarts: h.ecarts, ambigus: h.ambigus}
	for i := range b.voies {
		b.voies[i] = mntNouvelleVoie()
	}
	mntDigerer(b, sc.Reads, h)
	for _, v := range b.voies {
		v.finaliser()
	}
	sc.Reads = nil
	b.sc, b.duree = sc, time.Since(depart)
	return b
}

// mntDigerer range les lectures de i0 dans leurs seaux.
func mntDigerer(b *mntBilan, reads []ObjectiveRead, h mntHorloge) {
	for _, r := range reads {
		if r.Field != ObjectiveFieldTimers {
			continue
		}
		if !r.HasB {
			b.sansSecond++
		}
		ms, ok := h.ms[r.TimestampUS]
		if !ok {
			b.sansHorloge++
			continue
		}
		e := mntEch{tMS: ms, v: [2]int{ObjectiveTimerValue(r.Value), mntSecond(r)}}
		if r.FromKeyframe {
			b.voies[mntVoieCle].ajouter(r.Slot, e)
			continue
		}
		b.voies[mntVoieDelta].ajouter(r.Slot, e)
		if r.Chained {
			b.voies[mntVoieDeltaChainee].ajouter(r.Slot, e)
		}
	}
}

// mntSecond rend la seconde valeur, ou l'absence si le deserialiseur n'en a publie qu'une (ce
// qui ne doit pas arriver : `consumeObjectiveTimers` en publie toujours deux).
func mntSecond(r ObjectiveRead) int {
	if !r.HasB {
		return mntAbsent
	}
	return ObjectiveTimerValue(r.ValueB)
}

// ajouter range UNE lecture dans une voie.
func (v *mntVoieBilan) ajouter(slot uint32, e mntEch) {
	v.lectures++
	for k := 0; k < 2; k++ {
		v.histo[k][e.v[k]]++
		if mntLegal(e.v[k]) {
			v.legaux[k]++
		}
	}
	if mntLegal(e.v[0]) && mntLegal(e.v[1]) {
		v.legalPaire++
	}
	switch {
	case e.v[0] > e.v[1]:
		v.ordre[0]++
	case e.v[0] == e.v[1]:
		v.ordre[1]++
	default:
		v.ordre[2]++
	}
	v.series[slot] = append(v.series[slot], e)
}

// finaliser trie les series, en extrait les descentes et les extraits en clair, puis RELACHE les
// series : un corpus de treize films ne doit pas garder ses echantillons en memoire.
func (v *mntVoieBilan) finaliser() {
	v.slots = len(v.series)
	vus := map[int32]bool{}
	for slot, s := range v.series {
		sort.Slice(s, func(i, j int) bool { return s[i].tMS < s[j].tMS })
		for _, e := range s {
			vus[e.tMS] = true
		}
		for k := 0; k < 2; k++ {
			if mntVarie(s, k) {
				v.slotsVariables[k]++
			}
			v.descentes = append(v.descentes, mntDescentes(slot, k, s)...)
		}
	}
	sort.Slice(v.descentes, func(i, j int) bool {
		if v.descentes[i].debutMS != v.descentes[j].debutMS {
			return v.descentes[i].debutMS < v.descentes[j].debutMS
		}
		return v.descentes[i].slot < v.descentes[j].slot
	})
	v.instants = make([]int32, 0, len(vus))
	for t := range vus {
		v.instants = append(v.instants, t)
	}
	sort.Slice(v.instants, func(i, j int) bool { return v.instants[i] < v.instants[j] })
	v.extraits = mntExtraits(v.series)
	v.series = nil
}

// mntVarie dit si le champ k d'une serie n'est PAS constant. Un index de minuteur ne bouge que
// lorsque le mode rebranche l'objectif ; une duree bouge a chaque paquet.
func mntVarie(s []mntEch, k int) bool {
	for i := 1; i < len(s); i++ {
		if s[i].v[k] != s[0].v[k] {
			return true
		}
	}
	return false
}

// mntDescentes decoupe une serie triee en descentes au sens du gate 3.
func mntDescentes(slot uint32, champ int, s []mntEch) []mntDescente {
	var out []mntDescente
	for i := 0; i < len(s); {
		if s[i].v[champ] < 0 {
			i++
			continue
		}
		j := i
		for j+1 < len(s) && s[j+1].v[champ] >= 0 && s[j+1].v[champ] <= s[j].v[champ] {
			j++
		}
		if j-i+1 >= mntDescenteMinEch && s[i].v[champ]-s[j].v[champ] >= mntDescenteMinAmpl {
			out = append(out, mntDescente{slot: slot, champ: champ, debutMS: s[i].tMS,
				finMS: s[j].tMS, haut: s[i].v[champ], bas: s[j].v[champ], n: j - i + 1})
		}
		if j == i {
			i++
			continue
		}
		i = j
	}
	return out
}

// mntExtraits rend EN CLAIR les series des trois slots les plus fournis. Une duree se reconnait a
// l'oeil ; un index constant aussi.
func mntExtraits(series map[uint32][]mntEch) []string {
	type l struct {
		slot uint32
		n    int
	}
	ls := make([]l, 0, len(series))
	for s, v := range series {
		ls = append(ls, l{s, len(v)})
	}
	sort.Slice(ls, func(a, c int) bool {
		if ls[a].n != ls[c].n {
			return ls[a].n > ls[c].n
		}
		return ls[a].slot < ls[c].slot
	})
	out := make([]string, 0, 3)
	for i, x := range ls {
		if i == 3 {
			break
		}
		out = append(out, fmt.Sprintf("SERIE slot %d (%d ech.) : %s",
			x.slot, x.n, mntEnClair(series[x.slot])))
	}
	return out
}

// mntEnClair rend au plus 16 echantillons d'une serie, « t_s=(v0,v1) ».
func mntEnClair(s []mntEch) string {
	pas := 1
	if len(s) > 16 {
		pas = len(s) / 16
	}
	out, n := make([]string, 0, 17), 0
	for i := 0; i < len(s) && n < 16; i += pas {
		out = append(out, fmt.Sprintf("%.1fs=(%d,%d)",
			float64(s[i].tMS)/1000, s[i].v[0], s[i].v[1]))
		n++
	}
	if len(s) > 16 {
		out = append(out, "...")
	}
	return strings.Join(out, " ")
}
