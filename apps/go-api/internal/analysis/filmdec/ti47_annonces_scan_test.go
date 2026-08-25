package filmdec

// ti47_annonces_scan_test.go — LE MOTEUR de l'instrument du lot « ti=47 i2 personal-ai-data »
// (plan `.ai/V7.5/replay2d/PLAN_TI47_ANNONCES_ZONE.md`). Les verdicts sont dans
// `ti47_annonces_test.go` ; ce fichier ne fait que produire la matiere.
//
// CE QU'IL FAIT. Le lot C a mesure que `ti=47 i2 personal-ai-data-component` est le canal le
// plus concentre du corpus (1 214x le plancher de densite, 77-81 % des records de son archetype
// en modes a zones, quasi absent en CTF) et qu'il n'est PAS PORTE : sa largeur de bits est
// inconnue, donc sa valeur n'a jamais ete lue. L'instrument fait deux choses, dans cet ordre :
//
//	phase 0  recenser la bande de ti=47 (reel contre fantome), et MESURER la largeur de i2 par
//	         le CHAINAGE — un record correctement dimensionne finit la ou le suivant commence.
//	phase 1  une fois la largeur etablie au gate 0 (variable `TI47_WIDTH`), lire la valeur et la
//	         confronter aux oracles hors film.
//
// POURQUOI LE CHAINAGE ET PAS LE BINAIRE. La recette du depot lit les largeurs dans le
// descripteur `+0x28` du binaire du jeu (Ghidra). Aucune instance Ghidra n'est disponible pour ce
// lot : la largeur ne peut donc pas etre LUE. Le chainage ne la devine pas non plus — il la
// MESURE sur les octets, et le lot C-bis a etabli le protocole (LOTCBIS_PHASE0 §3.2 : `i1` de
// ti=13 chaine a 96,6 % contre 2,1 % pour le fantome). Ici il sert a DECOUVRIR la largeur et non
// seulement a la confirmer : l'histogramme est publie sur tous les decalages, pas seulement sur
// celui qu'on esperait.
//
// FILTRE REEL/FANTOME OBLIGATOIRE (lecon i27 du registre, 18/08 : un classement fonde sur le seul
// recensement de masques ment). Trois garde-fous, tous mesures et publies :
//   - la bande est celle que les IMAGES-CLE montrent pour ti=47, sans comblement ;
//   - un FANTOME de meme cardinalite (slots jamais vus porter ti=47, meme voisinage numerique)
//     passe par le meme code, et c'est le RAPPORT qui parle ;
//   - une VIE CONFIRMEE (paire slot+generation vue au moins [ti47VieMin] fois) est exigee pour
//     toute mesure de valeur : un slot touche une fois est du bruit d'ancrage, pas une entite.
//
// MEMOIRE PLAFONNEE partout (la bombe RAM `NamedEventsFrom`/`incrementTimes` — OOM ~26 Go — est
// au registre). Chaque accumulation par composant porte sa borne et son compteur de debordement.
//
// LECTURE SEULE : aucun octet ecrit hors des TSV de sortie, aucune base, aucun artefact.
// SOUS GARDE D'ENVIRONNEMENT (TI47_FILM). UN SEUL FILM PAR PROCESSUS.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TI47_FILM=<repo>/data/cache/film_chunks/7344d24f \
//	  TI47_CACHE=<repo>/data/cache TI47_SHORT=7344d24f TI47_OBJTYPE=zone \
//	  go test ./internal/analysis/filmdec/ -run TestTI47Annonces -v -timeout 60m

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	ti47FilmEnv    = "TI47_FILM"    // repertoire des chunks du film (garde du lot)
	ti47CacheEnv   = "TI47_CACHE"   // racine du cache (manifeste : horloge du match)
	ti47ShortEnv   = "TI47_SHORT"   // identifiant court du film
	ti47ObjTypeEnv = "TI47_OBJTYPE" // famille d'objectif FOURNIE : zone | flag (jamais devinee)
	ti47ReplayEnv  = "TI47_REPLAY"  // artefact de rejeu (oracles zoneStates / colline)
	ti47TSVEnv     = "TI47_TSV"     // repertoire de sortie des TSV (facultatif)
	ti47WidthEnv   = "TI47_WIDTH"   // phase 1 : largeur de i2 retenue au gate 0, en bits
	ti47RunEnv     = "TI47_RUN"     // largeurs candidates a departager par longueur de chaine
	ti47ChampEnv   = "TI47_CHAMP"   // sous-champ analyse, « debut:fin » en positions de bit
)

// compPersonalAIData nomme le composant cible. Un NOM, jamais un numero : le lot 0 a mesure deux
// decoupages de registre differents sur le corpus, un « ti=47 i2 » cable serait faux au prochain
// patch.
const compPersonalAIData = "personal-ai-data-component"

// ti47MaxDecalage borne l'histogramme de chainage, en bits. Les largeurs totales des composants
// deja portes du corpus tiennent toutes sous 40 bits (table LOTCBIS_PHASE0 §2.3) ; 192 laisse
// quatre fois la marge et coute une lecture d'en-tete par bit.
const ti47MaxDecalage = 192

// ti47VieMin est le nombre de records qu'une paire (slot, generation) doit atteindre pour compter
// comme une VIE CONFIRMEE. Trois : deux records peuvent tomber par hasard sur un slot large, une
// entite scriptee du mode en produit des centaines.
const ti47VieMin = 3

// ti47MaxRun borne la longueur de chaine mesuree. Une largeur juste enchaine les records
// consecutifs du meme paquet ; une fausse largeur meurt au premier ou au deuxieme pas.
const ti47MaxRun = 24

// ti47MaxEcart borne l'histogramme des distances entre records consecutifs de la bande, en bits.
const ti47MaxEcart = 512

// Bornes d'accumulation (memoire plafonnee).
const (
	ti47MaxValeurs   = 200000 // valeurs distinctes retenues
	ti47MaxEmissions = 500000 // emissions horodatees retenues
)

// ti47Bande porte l'archetype ti=47 resolu dans le registre du film, sa bande et son fantome.
type ti47Bande struct {
	ti   int
	arch Archetype
	// iPersonal / iStatic / iDynamic : index des trois composants dans LA grammaire de ce film
	// (-1 quand absent). Lus dans le registre, jamais supposes.
	iPersonal, iStatic, iDynamic int
	band, ghost                  map[uint32]bool
	// tous : tous les slots vus en image-cle, quel que soit l'archetype. C'est la cible du test
	// de chainage — la condition que le vrai decodeur rencontre au bit suivant.
	tous map[uint32]bool
}

// ti47Emission est UNE valeur lue, situee dans le temps et rattachee a une vie.
type ti47Emission struct {
	val       uint64
	slot, gen uint32
	chunk     int
	tMS       int // horloge du MATCH, -1 si le manifeste manque
}

// ti47Stat compte ce qu'une bande a rendu.
type ti47Stat struct {
	records, horsGrammaire int
	parIndex               map[int]int
	parSlot                map[uint32]int
	parVie                 map[uint64]int
	// singletons[i] : records dont le masque annonce le SEUL index i.
	singletons map[int]int
}

func nouveauTi47Stat() ti47Stat {
	return ti47Stat{parIndex: map[int]int{}, parSlot: map[uint32]int{}, parVie: map[uint64]int{},
		singletons: map[int]int{}}
}

// ti47Moisson est la recolte d'une passe.
type ti47Moisson struct {
	b                    *ti47Bande
	reel, fantome        ti47Stat
	chainPersonal        ti47Chain // singleton {i2}, bande REELLE, cible = tous les slots
	chainPersonalBande   ti47Chain // singleton {i2}, cible = les SEULS slots de ti=47
	chainPersonalFantome ti47Chain // singleton {i2}, bande FANTOME (le niveau de hasard)
	chainDynamique       ti47Chain // singleton {i1}, temoin POSITIF (R(24) connu)
	chainDynamiqueBande  ti47Chain // singleton {i1}, cible = les SEULS slots de ti=47
	chainStatique        ti47Chain // singleton {i0}, second temoin
	// runs[W] : histogramme des longueurs de chaine pour la largeur candidate W (plafonne a
	// ti47MaxRun). Rempli seulement quand TI47_RUN nomme des candidates.
	runs map[int][]int
	// ecarts[i] : distances, en bits, entre le DEBUT d'un record singleton {i} de la bande et le
	// debut du record de bande SUIVANT dans le meme paquet. La taille d'un record singleton vaut
	// worldObjectHeaderBits + worldObjectIndexBits + largeur : ce histogramme lit donc la
	// largeur directement, sans hypothese sur ce qui suit. Plafonne a ti47MaxEcart.
	ecarts map[int][]int
	// ecartsHors[i] : distances au-dela du plafond, comptees mais non detaillees.
	ecartsHors map[int]int
	// suivants[i] : masque du record suivant, quand la distance vaut la largeur dominante.
	suivants map[int]map[int]int
	// valeurs et emissions : phase 1 seulement (largeur fournie). Plafonnees.
	valeurs                map[uint64]int
	valeursDebordees       int
	emissions              []ti47Emission
	emissionsDebordees     int
	luesTotal, luesRefusee int
	deltaPaquets, chunks   int
	// bits[k] : nombre d'emissions dont le k-ieme bit de la charge utile (0 = premier bit lu)
	// vaut 1. Un champ de 45 bits dont la moitie des positions ne bouge jamais n'est pas un
	// scalaire de 45 bits : c'est une structure, et ce profil la montre sans rien supposer.
	bits []int
	// largeur retenue, recopiee pour les rapports.
	largeur int
	// champLo / champHi bornent le SOUS-CHAMP analyse (positions de bit dans la charge utile,
	// fin exclue). Par defaut toute la largeur. Le profil de bits porte toujours sur la largeur
	// ENTIERE — c'est lui qui justifie le sous-champ, il ne peut donc pas en dependre.
	champLo, champHi int
}

// ti47Cle assemble la cle d'une vie : la PAIRE slot+generation, jamais le slot seul (le pool de
// slots reboucle).
func ti47Cle(slot, gen uint32) uint64 { return uint64(slot)<<8 | uint64(gen) }

// ti47Resout retrouve l'archetype de ti=47 DANS LE REGISTRE DU FILM par le nom de son composant
// cible, et tranche une eventuelle ambiguite par le recensement (l'archetype qui a des slots).
func ti47Resout(t *testing.T, reg *Registry, c probeCensus) *ti47Bande {
	t.Helper()
	var vus []int
	for _, a := range reg.Archetypes {
		if len(a.indicesOf(compPersonalAIData)) > 0 {
			vus = append(vus, a.Index)
		}
	}
	if len(vus) == 0 {
		t.Logf("ARCHETYPE : %q ABSENT du registre de ce film", compPersonalAIData)
		return nil
	}
	choisi := vus[0]
	for _, ti := range vus {
		if len(c.parTI[ti]) > len(c.parTI[choisi]) {
			choisi = ti
		}
	}
	if len(vus) > 1 {
		t.Logf("ARCHETYPE AMBIGU : %q porte par %v — ti=%d retenu (%d slots en image-cle)",
			compPersonalAIData, vus, choisi, len(c.parTI[choisi]))
	}
	a, _ := reg.Archetype(choisi)
	b := &ti47Bande{ti: choisi, arch: a, iPersonal: -1, iStatic: -1, iDynamic: -1, tous: c.tous}
	b.iPersonal = ti47Index(a, compPersonalAIData)
	b.iStatic = ti47Index(a, compSplashMessageStatic)
	b.iDynamic = ti47Index(a, compSplashMessageDynamic)
	b.band = map[uint32]bool{}
	for s := range c.parTI[choisi] {
		b.band[s] = true
	}
	b.ghost = probeFantome(c.parTI[choisi], c.tous, len(b.band))
	return b
}

// ti47Index rend le premier index du composant nomme, ou -1.
func ti47Index(a Archetype, name string) int {
	if idx := a.indicesOf(name); len(idx) > 0 {
		return idx[0]
	}
	return -1
}

// ti47Largeur lit la largeur imposee par l'environnement (phase 1), ou 0.
func ti47Largeur(t *testing.T) int {
	t.Helper()
	v := os.Getenv(ti47WidthEnv)
	if v == "" {
		return 0
	}
	w, err := strconv.Atoi(v)
	if err != nil || w <= 0 || w > ti47MaxDecalage {
		t.Fatalf("%s = %q : largeur invalide (1..%d attendus)", ti47WidthEnv, v, ti47MaxDecalage)
	}
	return w
}

// ti47Candidates lit les largeurs a departager par longueur de chaine (TI47_RUN), ou nil.
func ti47Candidates(t *testing.T) []int {
	t.Helper()
	v := os.Getenv(ti47RunEnv)
	if v == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(v, ",") {
		w, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || w <= 0 || w > ti47MaxDecalage {
			t.Fatalf("%s = %q : largeur candidate invalide", ti47RunEnv, part)
		}
		out = append(out, w)
	}
	return out
}

// ti47Champ lit le sous-champ a analyser (TI47_CHAMP="debut:fin"), ou toute la largeur.
//
// POURQUOI UN SOUS-CHAMP. Le profil de bits de la phase 1 a mesure que la charge utile de 45 bits
// porte un en-tete quasi constant (bit 0 a zero, bit 1 a un, dix-huit bits a zero) suivi de 25
// bits qui varient. Analyser les 45 bits comme un entier fait dominer le bit 1 : ses rares
// bascules pesent 2^43, donc tout « saut » mesure sur l'entier n'est que cela. Le sous-champ
// n'est pas un reglage de confort, c'est la condition pour que la mesure porte sur la donnee.
func ti47Champ(t *testing.T, width int) (lo, hi int) {
	t.Helper()
	v := os.Getenv(ti47ChampEnv)
	if v == "" {
		return 0, width
	}
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("%s = %q : format attendu « debut:fin »", ti47ChampEnv, v)
	}
	a, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	b, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || a < 0 || b > width || a >= b {
		t.Fatalf("%s = %q : bornes invalides (0 <= debut < fin <= %d)", ti47ChampEnv, v, width)
	}
	return a, b
}

// ti47Balaye fait LA passe sur les paquets delta du film.
func ti47Balaye(dir string, n int, b *ti47Bande, hor probeHorloge, width int, cands []int,
	champLo, champHi int) *ti47Moisson {
	m := &ti47Moisson{b: b, reel: nouveauTi47Stat(), fantome: nouveauTi47Stat(),
		valeurs: map[uint64]int{}, runs: map[int][]int{}, ecarts: map[int][]int{},
		ecartsHors: map[int]int{}, suivants: map[int]map[int]int{},
		bits: make([]int, maxI(width, 1)), largeur: width, champLo: champLo, champHi: champHi}
	for _, w := range cands {
		m.runs[w] = make([]int, ti47MaxRun+1)
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		m.chunks++
		base := uint64(0)
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			if base == 0 {
				base = pk.TimestampUS
			}
			m.deltaPaquets++
			m.paquet(pk.Payload(data), c, hor.matchMS(c, pk.TimestampUS, base), width)
		}
	}
	return m
}

// paquet balaye UN payload delta.
func (m *ti47Moisson) paquet(pay []byte, chunk, tMS, width int) {
	b := m.b
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits)
	precDebut, precIdx := -1, -1
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, b.band)
		if ok {
			m.ecart(precDebut, precIdx, p, rec)
			precDebut, precIdx = p, ti47Singleton(rec)
			m.recordReel(pay, rec, chunk, tMS, width)
			p = rec.After
			continue
		}
		if rec, ok = matchWorldObjectRecord(pay, p, b.ghost); ok {
			m.recordFantome(pay, rec)
			p = rec.After
		}
	}
}

// recordReel compte un record de la bande reelle et alimente les histogrammes de chainage.
func (m *ti47Moisson) recordReel(pay []byte, rec WorldObjectRecord, chunk, tMS, width int) {
	b := m.b
	st := &m.reel
	st.records++
	st.parSlot[rec.Slot]++
	st.parVie[ti47Cle(rec.Slot, rec.Gen)]++
	hors := false
	for _, i := range rec.Idx {
		st.parIndex[i]++
		if i >= len(b.arch.Components) {
			hors = true
		}
	}
	if hors {
		st.horsGrammaire++
		return // un record hors grammaire n'est pas un record : il ne mesure rien
	}
	if len(rec.Idx) == 1 {
		st.singletons[rec.Idx[0]]++
		switch rec.Idx[0] {
		case b.iPersonal:
			m.chainPersonal.mesure(pay, rec.After, b.tous)
			m.chainPersonalBande.mesure(pay, rec.After, b.band)
			m.mesureRuns(pay, rec.After)
		case b.iDynamic:
			m.chainDynamique.mesure(pay, rec.After, b.tous)
			m.chainDynamiqueBande.mesure(pay, rec.After, b.band)
		case b.iStatic:
			m.chainStatique.mesure(pay, rec.After, b.tous)
		}
	}
	if width > 0 {
		m.lit(pay, rec, chunk, tMS, width)
	}
}

// recordFantome compte un record de la bande temoin et mesure SON chainage : le niveau de hasard,
// obtenu par le meme code sur des slots qui n'ont jamais porte cet archetype.
func (m *ti47Moisson) recordFantome(pay []byte, rec WorldObjectRecord) {
	st := &m.fantome
	st.records++
	st.parSlot[rec.Slot]++
	st.parVie[ti47Cle(rec.Slot, rec.Gen)]++
	for _, i := range rec.Idx {
		st.parIndex[i]++
		if i >= len(m.b.arch.Components) {
			st.horsGrammaire++
			return
		}
	}
	if len(rec.Idx) == 1 {
		st.singletons[rec.Idx[0]]++
		if rec.Idx[0] == m.b.iPersonal {
			m.chainPersonalFantome.mesure(pay, rec.After, m.b.tous)
		}
	}
}

// lit consomme les composants annonces AVANT i2 avec le dispatch de PRODUCTION, puis lit la
// valeur de i2 sur `width` bits. Un composant non porte en amont fait REFUSER le record : au dela,
// le curseur n'est plus digne de confiance (meme position que `probeMarche`).
func (m *ti47Moisson) lit(pay []byte, rec WorldObjectRecord, chunk, tMS, width int) {
	b := m.b
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		if id == b.iPersonal {
			m.luesTotal++
			if at+width > total {
				m.luesRefusee++
				return
			}
			m.ajoute(PeekBits(pay, at, width), rec, chunk, tMS)
			return
		}
		name := b.arch.component(id)
		if name == "" {
			return
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		if _, _, ported := consumeByName(br, name, uint32(b.ti), b.arch.Level(id)); !ported {
			return
		}
		if at = br.BitPos(); at > total {
			return
		}
	}
}

// ajoute range une valeur lue, sous plafond. Le profil de bits porte sur la charge utile
// ENTIERE ; la valeur analysee est le sous-champ.
func (m *ti47Moisson) ajoute(v uint64, rec WorldObjectRecord, chunk, tMS int) {
	for k := 0; k < m.largeur; k++ {
		if v&(1<<uint(m.largeur-1-k)) != 0 {
			m.bits[k]++
		}
	}
	a := (v >> uint(m.largeur-m.champHi)) & ((1 << uint(m.champHi-m.champLo)) - 1)
	if _, vu := m.valeurs[a]; vu || len(m.valeurs) < ti47MaxValeurs {
		m.valeurs[a]++
	} else {
		m.valeursDebordees++
	}
	if len(m.emissions) < ti47MaxEmissions {
		m.emissions = append(m.emissions,
			ti47Emission{val: a, slot: rec.Slot, gen: rec.Gen, chunk: chunk, tMS: tMS})
		return
	}
	m.emissionsDebordees++
}

// viesConfirmees rend les paires (slot, generation) vues au moins ti47VieMin fois.
func (s ti47Stat) viesConfirmees() map[uint64]bool {
	out := map[uint64]bool{}
	for k, n := range s.parVie {
		if n >= ti47VieMin {
			out[k] = true
		}
	}
	return out
}
