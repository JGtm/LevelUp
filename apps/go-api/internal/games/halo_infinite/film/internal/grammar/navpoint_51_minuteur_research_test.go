//go:build research

package grammar

// navpoint_51_minuteur_research_test.go — LOT 5.1.2 : LE MINUTEUR MANUEL DU POINT DE NAVIGATION,
// LU SUR UN FILM ENTIER.
//
// # LE VERDICT, MESURE LE 2026-09-18 : LE FILM N ECRIT PAS CE COMPTE A REBOURS
//
// LA PREUVE DEMANDEE A ECHOUE, et c est un NEGATIF MESURE, pas une absence de mesure. Sur les
// deux CTF du corpus, par la voie IMAGE-CLE — la seule ou la position est garantie :
//
//	film        navpoints   lectures   i11 non nul   i12 non nul   instants PENDANT un lacher
//	bcb6d393       13          156          0             0         2 sur 17   (18 lectures)
//	fb1a1a72       14          432          0             0        14 sur 39   (154 lectures)
//
// 588 lectures sur 588 a ZERO, dont 172 situees A L INTERIEUR d un intervalle `dropped` publie
// par `flagCarries` (13 lachers / 56,8 s cumulees sur `bcb6d393` ; 33 lachers / 299,6 s sur
// `fb1a1a72`). Le critere ecrit AVANT la mesure — « `i12` non nul PENDANT un lacher, nul en
// dehors » — est donc FALSIFIE : `i12` est nul PARTOUT, lachers compris.
//
// L ORACLE DE POSITION ECARTE LA LECTURE FAUSSE, et c est pour cela qu il est dans l instrument.
// `i10` publie deux index de minuteur dont le domaine legal est etabli depuis le 2026-09-01 :
// 312 index sur 312 (`bcb6d393`) et 882 sur 882 (`fb1a1a72`) sont DANS le domaine, zero hors
// domaine. La marche est donc juste jusqu a `i10` ; `i11` et `i12`, lus 17 et 34 bits plus loin,
// le sont a la bonne place. Les zeros sont la VALEUR du film, pas un cadrage rate.
//
// LES SEULES VALEURS NON NULLES VIENNENT DE LA VOIE DELTA, ET CE SONT DU BRUIT D ANCRAGE : 41
// lectures au total, UNE SEULE chainee, et 27 portant des durees superieures a 1 000 s sur des
// matchs de 350 et 800 s (le plein des 17 bits vaut 6 553 s). Un balayage qui essaie chaque bit
// trouve des en-tetes fortuits ; c est la population que le chainage ecarte.
//
// CE QUE CE NEGATIF FERME. La note 3.7 § 8.2 designait `ti=12 i11`/`i12` comme « le chemin le
// moins cher vers un compte a rebours d objectif » apres avoir refute le bassin du moteur
// (`ti=11 i0` a `(-1, -1)` sur 446 records sur 446). Les DEUX voies nommees sont desormais des
// negatifs mesures. Ce qui reste est `ti=13`, la propriete reseau nommee par le script Lua du
// mode (note 3.7 § 9.3) — non instruite, et hors du perimetre de ce lot.
//
// # CE QUE CET INSTRUMENT MESURE, ET CE QU IL NE DECIDE PAS
//
// Le lot 5.1.1 a porte `ti=12` de `i1` a `i12`. `i11` et `i12` sont la duree INITIALE et la duree
// COURANTE d un minuteur manuel, `R(17)` au pas de 50 ms. LA QUESTION DU LOT 5.1.2 est posee
// AVANT la mesure, et elle est falsifiable : `i12` doit etre NON NUL pendant les intervalles
// `dropped` que `flagCarries` publie deja, et NUL en dehors. Cet instrument ne rend QUE les
// lectures datees ; la confrontation se fait avec le calque du drapeau, cuit a part
// (`replaybuild/navpoint_51_drapeau_research_test.go`), parce que le calque exige le catalogue
// versionne d objectifs de carte et la chaine complete de la cuisson.
//
// # POURQUOI LES LECTURES D AVANT LA DESYNCHRONISATION SONT GARDEES
//
// Une image-cle est un ETAT COMPLET : ses composants se suivent dans l ordre du registre, sans
// masque de presence. Le bloquant de `ti=12` est desormais `i13`, donc `i11` et `i12` — lus
// juste avant — sont consommes dans l ordre et leur position est juste. C est la regle de
// `NOTE_V13_DEADSTATE_VEHICULE` § 3 : `DesyncAt` est l index du PREMIER composant non porte,
// tout ce qui precede a ete consomme. L instrument REFUSE en revanche toute marche qui
// desynchronise AVANT `i12` : la lecture n y serait pas situee.
//
// # LE TEMPS EST CELUI DU MANIFESTE, comme ScanNavpointRadial
//
// Base = le PREMIER PAQUET DELTA du chunk, decalage = start_ms du manifeste. C est la meme
// horloge que `objectives.StatRecords`, donc que les evenements qui bornent `flagCarries` — sans
// quoi les deux tableaux ne seraient pas confrontables.
//
// UN SEUL FILM PAR INVOCATION (NAV51_FILM) : la machine ne decode qu un film a la fois.
//
//	NAV51_FILM_ROOT=<cache>/film_chunks NAV51_MANIFESTS=<cache>/film_manifests
//	NAV51_FILM=bcb6d393 NAV51_OUT=<tmp>/bcb6d393.navtimers.tsv
//	go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/
//	  -run Navpoint51MinuteurSurFilm -v -timeout 60m

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// nav51Lecture : une lecture datee du couple (duree initiale, duree courante) d un navpoint.
type nav51Lecture struct {
	TMS      int32
	Slot     uint32
	Voie     string // keyframe ou delta
	QInitial uint64
	QCourant uint64
	VuI11    bool
	VuI12    bool
	Ferme    bool // la marche est allee au bout (aucune desynchronisation)
}

// nav51Manifeste : la part du manifeste du cache film dont l horloge a besoin.
type nav51Manifeste struct {
	Chunks []struct {
		Index   int `json:"index"`
		Type    int `json:"chunk_type"`
		StartMS int `json:"start_ms"`
	} `json:"chunks"`
}

func TestNavpoint51MinuteurSurFilm(t *testing.T) {
	racine, manifests, court, sortie := nav51Garde(t)
	meta, clock := nav51Horloge(t, filepath.Join(manifests, court+".json"))
	film, err := source.LoadDir(filepath.Join(racine, court), meta)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", court, err)
	}
	fc := contexteDeBobine(film)
	w := &nav51Marche{prof: fc.ProfilDeBalayage()}
	if w.arch, w.reg, err = fc.filmArchetype(navpointRadialArchIndex); err != nil {
		t.Fatalf("archetype ti=12 de %s : %v", court, err)
	}
	w.iInitial, w.iCourant = nav51Index(w.arch)
	if w.iInitial < 0 || w.iCourant < 0 {
		t.Fatalf("%s : l archetype ti=12 ne declare pas les deux minuteurs manuels", court)
	}
	w.bande = bandeObserveeKeyframes(ScanWorldObjectKeyframes(fc.Film(), navpointRadialArchIndex))
	w.obs = w.installer()
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.chunk(data, pks, clock, c)
	}
	nav51Publier(t, court, w, sortie)
}

// nav51Garde lit les quatre variables d environnement. SKIP propre si elles manquent.
func nav51Garde(t *testing.T) (racine, manifests, court, sortie string) {
	t.Helper()
	racine, manifests = os.Getenv("NAV51_FILM_ROOT"), os.Getenv("NAV51_MANIFESTS")
	court, sortie = os.Getenv("NAV51_FILM"), os.Getenv("NAV51_OUT")
	if racine == "" || manifests == "" || court == "" || sortie == "" {
		t.Skip("NAV51_FILM_ROOT, NAV51_MANIFESTS, NAV51_FILM et NAV51_OUT requis")
	}
	if strings.ContainsAny(court, ",;") {
		t.Fatal("NAV51_FILM ne prend QU UN film : la machine ne decode qu un film a la fois")
	}
	return racine, manifests, court, sortie
}

// nav51Horloge lit le manifeste du cache et rend les metadonnees de chunk plus l horloge.
func nav51Horloge(t *testing.T, chemin string) ([]types.ChunkMeta, map[int]int) {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin fourni par l operateur
	if err != nil {
		t.Fatalf("manifeste %s : %v", chemin, err)
	}
	var m nav51Manifeste
	if err := json.Unmarshal(blob, &m); err != nil {
		t.Fatalf("manifeste %s : %v", chemin, err)
	}
	meta := make([]types.ChunkMeta, 0, len(m.Chunks))
	clock := make(map[int]int, len(m.Chunks))
	for _, c := range m.Chunks {
		meta = append(meta, types.ChunkMeta{Index: c.Index, ChunkType: c.Type, StartMS: c.StartMS})
		clock[c.Index] = c.StartMS
	}
	return meta, clock
}

// nav51Index rend les index de registre des deux composants de minuteur manuel.
func nav51Index(a Archetype) (int, int) {
	init, cur := -1, -1
	for i, n := range a.Components {
		switch n {
		case compNavpointManualTimerInitial:
			init = i
		case compNavpointManualTimerCurrent:
			cur = i
		}
	}
	return init, cur
}

// nav51Marche porte l etat d une marche et ce que le hook y depose.
type nav51Marche struct {
	obs                *Observation
	prof               ProfilDeBalayage
	arch               Archetype
	reg                *Registry
	iInitial, iCourant int
	bande              map[uint32]bool
	cur                nav51Lecture
	lectures           []nav51Lecture
	// Comptes : records vus, marches gardees, marches rejetees (desynchronisation AVANT i12),
	// marches fermees, et l histogramme des bloquants.
	keyRecords, keyGardes, keyRejetes, keyFermes int
	// ORACLE DE POSITION : i10 publie DEUX index de minuteur, dont le domaine legal est
	// etabli depuis le 2026-09-01 ({-1} U [0,63] U {65,66,67}). Une marche decalee d un bit
	// avant i11 les rendrait hors domaine : c est la contre-epreuve GRATUITE que les zeros
	// de i11 et i12 sont lus a la bonne place, et non a cote.
	i10Lus, i10Legaux, i10HorsDomaine       int
	deltaRecords, deltaGardes, deltaRejetes int
	bloquants                               map[int]int
}

func (w *nav51Marche) contexte() ContexteDeLecture {
	return ContexteDeLecture{Profil: w.prof, Obs: w.obs}
}

// installer pose le hook de ti=12 : il ne retient QUE les deux minuteurs manuels.
func (w *nav51Marche) installer() *Observation {
	w.bloquants = map[int]int{}
	obs := NouvelleObservation()
	obs.NavpointHook = func(f NavpointField, values []uint64) {
		if len(values) == 0 {
			return
		}
		switch f {
		case NavpointManualTimerInitial:
			w.cur.QInitial, w.cur.VuI11 = values[0], true
		case NavpointManualTimerCurrent:
			w.cur.QCourant, w.cur.VuI12 = values[0], true
		case NavpointRadialProgress:
		}
	}
	obs.ObjectiveHook = func(f ObjectiveField, values []uint64) {
		if f != ObjectiveFieldTimers || len(values) != 2 {
			return
		}
		for _, q := range values {
			w.i10Lus++
			if nav51IndexLegal(ObjectiveTimerValue(q)) {
				w.i10Legaux++
			} else {
				w.i10HorsDomaine++
			}
		}
	}
	return obs
}

// nav51IndexLegal applique le domaine legal d un index de minuteur, mesure le 2026-09-01 sur
// 1 149 lectures d image-cle : -1 (aucun minuteur), une fente du bassin [0, 63], ou l un des
// trois minuteurs reserves 65 (manche), 66 (mort subite), 67 (delai de grace).
func nav51IndexLegal(v int) bool {
	return v == -1 || (v >= 0 && v <= 63) || (v >= 65 && v <= 67)
}

// chunk balaye les paquets d UN chunk sur l horloge du manifeste.
func (w *nav51Marche) chunk(data []byte, pks []FilmPacket, clock map[int]int, c int) {
	base, ok := navpointRadialBaseChunk(pks)
	start, aStart := clock[c]
	if !ok || !aStart {
		return
	}
	for _, pk := range pks {
		ms := int32(start + int((int64(pk.TimestampUS)-int64(base))/1000))
		switch pk.Type {
		case PacketTypeKeyframe:
			w.imageCle(pk.Payload(data), ms)
		case PacketTypeDelta:
			w.delta(pk.Payload(data), ms)
		}
	}
}

// imageCle marche les records ti=12 d une image-cle sous le cadre d etat complet.
func (w *nav51Marche) imageCle(pay []byte, ms int32) {
	total := len(pay) * 8
	for _, b := range keyframeBornesToutes(pay) {
		if b.TI != navpointRadialArchIndex {
			continue
		}
		w.keyRecords++
		w.cur = nav51Lecture{TMS: ms, Slot: uint32(b.Slot), Voie: "keyframe"} //nolint:gosec // id de 30 bits
		tr := WalkKeyframeFullState(pay, b.Bit, w.reg, w.contexte())
		if tr.DesyncAt >= 0 {
			w.bloquants[tr.DesyncAt]++
		}
		// LA REGLE DE V13 : une desynchronisation APRES i12 laisse i11 et i12 a leur place ;
		// une desynchronisation AVANT les rend non situes, donc irrecevables.
		if tr.EndBit > total || (tr.DesyncAt >= 0 && tr.DesyncAt <= w.iCourant) {
			w.keyRejetes++
			continue
		}
		w.cur.Ferme = tr.DesyncAt < 0
		if w.cur.Ferme {
			w.keyFermes++
		}
		if w.cur.VuI11 || w.cur.VuI12 {
			w.keyGardes++
			w.lectures = append(w.lectures, w.cur)
		}
	}
}

// delta ancre les records ti=12 d un paquet delta et marche leur masque.
func (w *nav51Marche) delta(pay []byte, ms int32) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, w.bande)
		if !ok {
			continue
		}
		if !w.dansLeDomaine(rec.Idx) || !w.citeUnMinuteur(rec.Idx) {
			continue
		}
		w.deltaRecords++
		w.cur = nav51Lecture{TMS: ms, Slot: rec.Slot, Voie: "delta"}
		if w.marcherMasque(pay, rec) {
			w.deltaGardes++
			w.lectures = append(w.lectures, w.cur)
		} else {
			w.deltaRejetes++
		}
		p = rec.After
	}
}

func (w *nav51Marche) dansLeDomaine(idx []int) bool {
	for _, id := range idx {
		if id < 0 || id >= len(w.arch.Components) {
			return false
		}
	}
	return true
}

// citeUnMinuteur : le masque du record cite-t-il i11 ou i12 ?
func (w *nav51Marche) citeUnMinuteur(idx []int) bool {
	for _, id := range idx {
		if id == w.iInitial || id == w.iCourant {
			return true
		}
	}
	return false
}

// marcherMasque marche les composants du masque avec les desers de production et dit si les
// minuteurs ont ete atteints.
func (w *nav51Marche) marcherMasque(pay []byte, rec WorldObjectRecord) bool {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := w.arch.component(id)
		if name == "" || at > total {
			w.bloquants[id]++
			return false
		}
		br := LecteurSur(pay)
		br.PoserContexte(w.contexte())
		br.SetBitPos(at)
		_, _, porte := consumeByName(br, name, navpointRadialArchIndex, w.arch.Level(id))
		if !porte || br.BitPos() > total {
			w.bloquants[id]++
			return false
		}
		at = br.BitPos()
	}
	w.cur.Ferme = worldObjectHeaderAt(pay, at)
	return w.cur.VuI11 || w.cur.VuI12
}

// nav51Publier ecrit le TSV et resume la mesure au journal.
func nav51Publier(t *testing.T, court string, w *nav51Marche, sortie string) {
	t.Helper()
	sort.SliceStable(w.lectures, func(i, j int) bool { return w.lectures[i].TMS < w.lectures[j].TMS })
	var b strings.Builder
	b.WriteString("# minuteur manuel du navpoint (ti=12 i11 et i12) — film " + court + "\n")
	b.WriteString("tms\tslot\tvoie\tferme\tq_initial\tinitial_s\tq_courant\tcourant_s\n")
	nonNuls := 0
	for _, l := range w.lectures {
		if l.VuI12 && l.QCourant != 0 {
			nonNuls++
		}
		fmt.Fprintf(&b, "%d\t%d\t%s\t%t\t%d\t%.3f\t%d\t%.3f\n", l.TMS, l.Slot, l.Voie, l.Ferme,
			l.QInitial, NavpointManualTimerValue(l.QInitial), l.QCourant, NavpointManualTimerValue(l.QCourant))
	}
	if err := os.WriteFile(sortie, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", sortie, err)
	}
	t.Logf("FILM %s — bande de slots observes : %d", court, len(w.bande))
	t.Logf("FILM %s — image-cle : %d records, %d gardes, %d rejetes (desync avant i12), %d fermes",
		court, w.keyRecords, w.keyGardes, w.keyRejetes, w.keyFermes)
	t.Logf("FILM %s — delta : %d records citant un minuteur, %d gardes, %d rejetes",
		court, w.deltaRecords, w.deltaGardes, w.deltaRejetes)
	t.Logf("FILM %s — ORACLE DE POSITION i10 : %d index lus, %d legaux, %d HORS DOMAINE",
		court, w.i10Lus, w.i10Legaux, w.i10HorsDomaine)
	t.Logf("FILM %s — %d lectures ecrites dans %s, dont %d a duree courante NON NULLE",
		court, len(w.lectures), sortie, nonNuls)
	ids := make([]int, 0, len(w.bloquants))
	for id := range w.bloquants {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		nom := "(hors registre)"
		if id >= 0 && id < len(w.arch.Components) {
			nom = w.arch.Components[id]
		}
		t.Logf("FILM %s — bloquant i%d %s : %d", court, id, nom, w.bloquants[id])
	}
}
