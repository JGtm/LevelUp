package filmdec

// objectif_bombstate_test.go — L'ETAT DE LA BOMBE EST UNE PROPRIETE NOMMEE, ET SON NOM EST CALCULE.
//
// # CE QUE LE BINAIRE A RENDU (2026-09-01, Ghidra)
//
// Le jeu declare une enumeration `BombObjectState`, enregistree par reflexion avec CINQ membres
// (`FUN_14034a090` -> `FUN_14154c940(&DAT_144d6f640, "BombObjectState", ..., 5)`), et
// `FUN_14034a0d0` en donne les valeurs numeriques :
//
//	BombObjectState_None       = 0
//	BombObjectState_Unarmed    = 1
//	BombObjectState_Armed      = 2
//	BombObjectState_Disarming  = 3
//	BombObjectState_Contested  = 4
//
// **Il n'y a pas d'etat « Arming ».** Poser la bombe fait donc passer l'objet par `Contested`
// — l'etat « quelqu'un interagit » — avant `Armed`. L'INSTANT OU L'ETAT DEVIENT `Contested` EST
// L'INSTANT OU L'ARMEMENT COMMENCE, et c'est exactement ce que ce chantier cherche depuis le
// debut.
//
// Le binaire donne aussi, exposes au script du mode : `Device_GetHoldCompleteFraction` (la
// fraction d'avancement d'une interaction tenue), `Device_GetInteractionHoldTime`, et le champ de
// tag `chud_arming_meter_name` (la jauge d'armement du HUD).
//
// # POURQUOI LE CHANTIER AVAIT REGARDE AU BON ENDROIT AVEC LE MAUVAIS CRITERE
//
// `ti=13` (`managed-object-property-*`) est **un sac de proprietes NOMMEES** : `i0` porte le NOM
// (un identifiant de 32 bits), `i1` la valeur scalaire. La phase A7 y a cherche une PROGRESSION —
// une valeur qui CROIT — et n'a rien trouve, ce qui etait juste : un etat de bombe ne croit pas,
// il SAUTE (1 -> 4 -> 2). Et la valeur n'est pas du tag 3 (le flottant quantifie des jauges de
// zone) mais un ENUMERE.
//
// Les films d'Assaut portent exactement HUIT slots `ti=13`. Si l'etat de la bombe est replique
// par cette voie, **l'un de ces huit slots porte le nom `BombObjectState`**.
//
// # LE NOM EST CALCULABLE, DONC LA MESURE EST UNE EGALITE ET NON UNE CORRELATION
//
// `FUN_140748a74` est un **murmur3 x86_32, graine 0**, applique a la chaine NORMALISEE :
// majuscules en minuscules, `-` et espace en `_`, saut de ligne en `#`. Les constantes le
// confirment (0xcc9e2d51, 0x1b873593, rotation de 13, 0xe6546b64, puis le fmix32 0x85ebca6b /
// 0xc2b2ae35).
//
//	murmur3("bombobjectstate") = 0x19813E20  (427 900 448)
//
// **Chercher cette valeur exacte dans `ti=13 i0` n'est pas une correlation : c'est une egalite.**
// Elle tombe ou elle ne tombe pas, et les deux reponses sont des resultats.
//
// # CE QUI REND CETTE MESURE POSSIBLE SANS TOUCHER A LA PRODUCTION
//
// `SetProbeHook` publie deja `ProbeManagedObjectPropertyName` — le depot avait prevu ce besoin
// exact (« quatre composants portes dont personne ne sait a quoi ils servent »). Aucun chemin de
// production n'est modifie ici.
//
// # LES DEUX ISSUES, ecrites avant la mesure
//
//	L'IDENTIFIANT TOMBE      le slot qui le porte est l'objet bombe, sa valeur est l'etat, et le
//	                         passage a `Contested` (4) date le debut d'armement ;
//	L'IDENTIFIANT NE TOMBE PAS  l'etat de la bombe n'est pas replique comme propriete nommee.
//	                         C'est un negatif NET — pas « pas vu », mais « cherche par egalite
//	                         sur un nom calcule et absent ».
//
// TEMOIN : les memes identifiants sont releves sur KOTH, Strongholds et CTF. Un identifiant qui
// n'apparait QU'EN ASSAUT est un candidat meme si ce n'est pas celui-la.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifBombState -v -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// bsBombObjectState est murmur3("bombobjectstate"), l'identifiant attendu de la propriete.
const bsBombObjectState = 0x19813E20

// bsEtats nomme les cinq valeurs de l'enumeration, telles que `FUN_14034a0d0` les enregistre.
func bsEtats(v uint64) string {
	switch v {
	case 0:
		return "None"
	case 1:
		return "Unarmed"
	case 2:
		return "Armed"
	case 3:
		return "Disarming"
	case 4:
		return "Contested"
	}
	return fmt.Sprintf("hors-enum(%d)", v)
}

// bsLecture est une valeur de propriete lue, avec le nom du slot qui la porte.
type bsLecture struct {
	slot        uint32
	nom         uint64 // l'identifiant de 32 bits de la propriete (ti=13 i0)
	tag         int
	valeur      uint64
	aValeur     bool
	timestampUS uint64
	chaine      bool
}

// bsBilan porte l'etat du balayage d'un film.
type bsBilan struct {
	lectures []bsLecture
	noms     map[uint64]int // identifiant -> nombre de records ou il apparait
	records  int
	chaines  int
}

// TestObjectifBombState cherche la propriete nommee `BombObjectState` dans `ti=13`.
func TestObjectifBombState(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer LockProcessDecode()()
	g := filmproc.Arm("TestObjectifBombState", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — balayage interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	t.Logf("identifiant cherche : murmur3(\"bombobjectstate\") = 0x%08X (%d)",
		bsBombObjectState, bsBombObjectState)
	global := map[uint64]int{}
	trouve := 0
	for _, f := range ti11Corpus {
		b := bsFilm(t, cache, f.id)
		if b == nil {
			continue
		}
		for id, n := range b.noms {
			global[id] += n
		}
		chainage := ti11Part(b.chaines, b.records)
		t.Logf("%-9s %-26s %5d record(s), %4.1f %% chaines, %d nom(s) distinct(s) : %s",
			f.id, f.mode, b.records, chainage, len(b.noms), bsNoms(b.noms))
		if n, ok := b.noms[bsBombObjectState]; ok {
			trouve++
			t.Logf("           *** BombObjectState PRESENT dans %d record(s) ***", n)
			bsSerie(t, b)
		}
	}

	t.Logf("########## BILAN — %d film(s) portent l'identifiant BombObjectState", trouve)
	t.Logf("IDENTIFIANTS VUS, tous films confondus : %s", bsNoms(global))
	if trouve == 0 {
		t.Logf("NEGATIF NET : l'etat de la bombe n'est PAS replique comme propriete nommee de "+
			"ti=13. L'identifiant 0x%08X est CALCULE, pas devine — son absence est une reponse, "+
			"pas un manque de chance. Les identifiants ci-dessus restent a nommer : chacun est un "+
			"murmur3 d'un nom en clair du binaire, et le rapprochement est mecanique.",
			bsBombObjectState)
	}
}

// bsFilm balaie UN film et rend son bilan, ou nil s'il est illisible.
func bsFilm(t *testing.T, cache, id string) *bsBilan {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil
	}
	arch, ok := reg.Archetype(ManagedPropertyTypeIndex)
	if !ok {
		return nil
	}
	band := observedSlotBand(dir, n, ManagedPropertyTypeIndex)
	if len(band) == 0 {
		return nil
	}
	w := &bsWalk{arch: arch, b: &bsBilan{noms: map[uint64]int{}}}
	defer w.install()()
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			w.payload(pk.Payload(data), band, pk.TimestampUS)
		}
	}
	return w.b
}

// bsWalk porte l'etat que les deux hooks deposent pendant la marche d'un record.
type bsWalk struct {
	arch Archetype
	b    *bsBilan
	// nom est l'identifiant publie par `i0` pour le record en cours ; valeurs les lectures de
	// `i1`/`i2..i33` du meme record, en attente de se voir attribuer ce nom.
	nom     uint64
	aNom    bool
	valeurs []bsLecture
}

// install pose les DEUX hooks — le nom vient de la sonde generique, la valeur du hook de ti=13 —
// et rend leur restauration.
func (w *bsWalk) install() func() {
	prevProbe, prevProp := probeHook, managedPropertyHook
	SetProbeHook(func(ti uint32, comp ProbeComponent, values []uint64) {
		if comp != ProbeManagedObjectPropertyName || len(values) == 0 {
			return
		}
		w.nom, w.aNom = values[0], true
	})
	SetManagedPropertyHook(func(f ManagedPropertyField, values []uint64) {
		if len(values) == 0 {
			return
		}
		l := bsLecture{tag: int(values[0])}
		if len(values) > 1 {
			l.valeur, l.aValeur = values[1], true
		}
		w.valeurs = append(w.valeurs, l)
	})
	return func() {
		SetProbeHook(prevProbe)
		SetManagedPropertyHook(prevProp)
	}
}

// payload marche les records delta d'UN payload et range ce que les hooks publient.
func (w *bsWalk) payload(pay []byte, band map[uint32]bool, ts uint64) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !ti11IdxDansDomaine(rec.Idx, len(w.arch.Components)) {
			continue
		}
		w.nom, w.aNom, w.valeurs = 0, false, w.valeurs[:0]
		at, done := rec.After, true
		for _, id := range rec.Idx {
			name := w.arch.component(id)
			if name == "" || at > total {
				done = false
				break
			}
			br := NewBitReader(pay)
			br.SetBitPos(at)
			_, _, ported := consumeByName(br, name, uint32(ManagedPropertyTypeIndex), w.arch.Level(id))
			if !ported || br.BitPos() > total {
				done = false
				break
			}
			at = br.BitPos()
		}
		p = rec.After
		if !done {
			continue
		}
		w.b.records++
		chaine := worldObjectHeaderAt(pay, at)
		if chaine {
			w.b.chaines++
		}
		if !w.aNom {
			continue
		}
		w.b.noms[w.nom]++
		for _, l := range w.valeurs {
			l.slot, l.nom, l.timestampUS, l.chaine = rec.Slot, w.nom, ts, chaine
			w.b.lectures = append(w.b.lectures, l)
		}
	}
}

// bsNoms rend les identifiants vus, du plus frequent au moins frequent.
func bsNoms(m map[uint64]int) string {
	type l struct {
		id uint64
		n  int
	}
	ls := make([]l, 0, len(m))
	for id, n := range m {
		ls = append(ls, l{id, n})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].n != ls[j].n {
			return ls[i].n > ls[j].n
		}
		return ls[i].id < ls[j].id
	})
	out := ""
	for i, x := range ls {
		if i >= 12 {
			out += fmt.Sprintf(", … (+%d)", len(ls)-12)
			break
		}
		if out != "" {
			out += ", "
		}
		marque := ""
		if x.id == bsBombObjectState {
			marque = "=BombObjectState"
		}
		out += fmt.Sprintf("0x%08X%s x%d", x.id, marque, x.n)
	}
	if out == "" {
		return "(aucun)"
	}
	return out
}

// bsSerie imprime la suite datee des valeurs portees par l'identifiant cherche.
func bsSerie(t *testing.T, b *bsBilan) {
	t.Helper()
	var l []bsLecture
	for _, x := range b.lectures {
		if x.nom == bsBombObjectState {
			l = append(l, x)
		}
	}
	sort.SliceStable(l, func(i, j int) bool { return l[i].timestampUS < l[j].timestampUS })
	for i, x := range l {
		if i >= 40 {
			t.Logf("           … (+%d lectures)", len(l)-40)
			break
		}
		v := "(muet)"
		if x.aValeur {
			v = fmt.Sprintf("%d = %s", x.valeur, bsEtats(x.valeur))
		}
		t.Logf("           slot %-5d %8.1fs tag %-2d %-20s chaine=%v",
			x.slot, float64(x.timestampUS)/1e6, x.tag, v, x.chaine)
	}
}
