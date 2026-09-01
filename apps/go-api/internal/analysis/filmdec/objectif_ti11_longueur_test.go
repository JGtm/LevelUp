package filmdec

// objectif_ti11_longueur_test.go — LA LONGUEUR D'UN RECORD D'IMAGE-CLE, ET CE QU'ELLE TRANCHE.
//
// # POURQUOI CETTE MESURE PLUTOT QUE LA MARCHE DETERMINISTE
//
// L'oracle voisin a rendu un negatif net : meme les records `ti=11` a UN SEUL composant ne
// chainent qu'a 25 %, et ceux a plusieurs composants jamais. J'allais marcher la table avec
// `WalkKeyframeRecords` pour separer « largeur fausse » de « ancrage faux » — mais le depot
// portait deja la reponse, ecrite dans l'en-tete de `keyframe_record_walk.go` (lot R3/R5) :
//
//	« AUCUN decalage ne rend une seule marche bit-exacte. Le corps d'un record de la table
//	  d'image-cle n'est donc PAS le corps d'un record NEW, quel que soit l'endroit ou on le
//	  pose. [...] l'image-cle porterait un ETAT COMPLET — tous les composants de l'archetype,
//	  sans masque epars. »
//
// Cette lecture explique TOUT ce que l'oracle a mesure : si le corps n'a pas de masque epars,
// alors ce que je lisais comme « le masque » etait deja de la donnee de composant. D'ou les
// 14,9 % de masques hors domaine, d'ou des valeurs de i12 constantes par slot et identiques d'un
// match a l'autre, d'ou un chainage qui ne monte jamais.
//
// # L'EPREUVE, ET ELLE NE DEMANDE AUCUN DESERIALISEUR
//
// Si le record porte un ETAT COMPLET, sa longueur est FIXE par archetype (au variable pres des
// deux textes formates). Elle se mesure sans rien decoder : depuis l'ancre d'un record, chercher
// la premiere position qui porte un en-tete valide a slot croissant. Une distribution DOMINEE PAR
// UNE VALEUR confirme l'etat complet ; une distribution etalee le refute.
//
// # LE PREDICAT CHIFFRE, ecrit avant la mesure
//
// La somme des largeurs portees de `ti=11`, textes formates et `i4` exclus, vaut 745 bits
// (`ti11SommeLargeurs`). En ajoutant l'en-tete de 64 bits, les 6 bits de `typeIndex` et le
// prologue de version, un record d'etat complet doit tomber **entre 820 et 900 bits**. C'est le
// meme ordre de grandeur que la seule longueur deja mesuree dans le depot (`ti=38`, dominante
// 827 bits) — et c'est ce qui rend le predicat falsifiable plutot que complaisant.
//
// TEMOIN : `ti=38`, dont la dominante est connue. Un instrument qui ne la retrouve pas est faux,
// et son verdict sur `ti=11` ne vaut rien.
//
// # LE RESULTAT : LE PREDICAT EST REFUTE, ET IL LAISSE UN FAIT STRUCTUREL NEUF (2026-09-01)
//
// ZERO record ne tombe dans la fenetre 820..900. L'etat complet est donc REFUTE pour `ti=11` :
// la somme des largeurs portees (745 bits) excede a elle seule le corps mesure.
//
// Ce que la mesure trouve a la place vaut mieux que ce qu'elle cherchait :
//
//	ti=11   2 611 records, dominante 168 bits a 93,8 %, DIX longueurs distinctes en tout ;
//	ti=13   2 948 records, dominante 446 bits a 73,5 %, SEPT longueurs distinctes ;
//	ti=38  44 231 records, dominante 555 bits a 6,9 %, 266 longueurs — etale, pas concentre.
//
// **Les records d'image-cle de `ti=11` et `ti=13` sont de LONGUEUR FIXE PAR ARCHETYPE.** Aucun
// instrument du depot ne l'avait mesure, et cela change la nature du probleme : un record de
// taille fixe dont le contenu varie porte du REMPLISSAGE, et sa longueur ne peut donc plus servir
// d'oracle de largeur. (`ti=38` etale : soit sa longueur varie vraiment, soit la fenetre de
// recherche de 2 000 bits attrape de faux en-tetes sur des records longs — non tranche.)
//
// `TestObjectifTi11Residu` mesure ensuite l'ECART entre la marche de production et cette fin
// reelle, ventile par nombre de composants : +90, +70, +32, -6, -16 bits pour 0 a 4 composants.
// Il DECROIT avec le nombre de composants — la lecture par masque epars sur-consomme des qu'il y
// en a plusieurs, ce qui est coherent avec un corps a champs de position FIXE plutot qu'a flux
// pilote par un masque.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Longueur -v -timeout 40m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// ti11SommeLargeurs est la somme des largeurs portees de l'archetype, `i2`, `i9` et `i4` EXCLUS
// (les deux textes formates sont de longueur variable, `i4` n'est pas porte). Ecrite ici comme
// une CONSTANTE VERIFIEE par `TestObjectifTi11SommeLargeurs` plutot que recopiee a la main :
// 14 + 32 + 32 + 32 + 1 + 8 + 4 + 1 + 1 + 32 + 32 + 3 + 32 + 16x32 + 8 + 1.
const ti11SommeLargeurs = 745

// ti11LongueurMin / Max bornent le predicat ecrit avant la mesure : en-tete 64 + typeIndex 6 +
// prologue de version 1 a 9 + la somme des largeurs + les trois composants variables.
const (
	ti11LongueurMin = 820
	ti11LongueurMax = 900
)

// ti11FenetreMax borne la recherche du record suivant. Deux fois la longueur attendue : au-dela,
// ce n'est plus « le record suivant » mais une position quelconque du payload.
const ti11FenetreMax = 2000

// TestObjectifTi11SommeLargeurs verifie la constante contre les largeurs reellement portees.
//
// UN GARDE-RAIL, PAS UN CALCUL : si une largeur du fichier de portage change, ce test tombe et
// le predicat de longueur ci-dessus doit etre reecrit avec elle. Sans lui, la constante derive
// en silence et le predicat devient une opinion.
func TestObjectifTi11SommeLargeurs(t *testing.T) {
	somme := objectiveTimerBits*objectiveTimerCount + // i0
		objectiveColorChannelBits*objectiveColorChannels + // i1
		objectiveHandleBits + // i3
		objectiveTypeBits + // i5
		1 + // i6
		objectivePriorityBits + // i7
		objectiveMessageTypeBits + // i8
		1 + 1 + // i10, i11
		objectiveProgressBits*2 + // i12, i13
		objectiveStateBits + // i14
		objectiveHandleBits + // i15
		objectiveHandleBits*16 + // i16..i31
		objectiveOutroBits + // i32
		1 // i33
	if somme != ti11SommeLargeurs {
		t.Fatalf("ti11SommeLargeurs = %d, les largeurs portees en donnent %d — le predicat de "+
			"longueur (%d..%d bits) doit etre reecrit", ti11SommeLargeurs, somme,
			ti11LongueurMin, ti11LongueurMax)
	}
	t.Logf("somme des largeurs portees (i2, i4, i9 exclus) : %d bits ; un record d'etat complet "+
		"doit donc tomber entre %d et %d bits en-tete comprise", somme, ti11LongueurMin, ti11LongueurMax)
}

// TestObjectifTi11Longueur mesure la distribution des longueurs de record par archetype.
func TestObjectifTi11Longueur(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Longueur", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	// ti=38 est le TEMOIN (dominante connue : 827 bits) ; ti=11 la cible ; ti=13 le comparatif
	// d'un archetype dont les valeurs, elles, sont lues juste depuis le lot C-bis.
	cibles := []int{ti11ArchIndex, 38, ManagedPropertyTypeIndex}
	long := map[int]map[int]int{}
	for _, ti := range cibles {
		long[ti] = map[int]int{}
	}
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				ti11LongueursPayload(pk.Payload(data), cibles, long)
			}
		}
	}
	for _, ti := range cibles {
		t.Logf("########## ti=%d — %s", ti, ti11Dominante(long[ti]))
		t.Logf("   %s", ti11TopLongueurs(long[ti], 10))
	}
	ti11Verdict(t, long[ti11ArchIndex])
}

// ti11LongueursPayload releve, pour chaque record des archetypes cibles, la distance jusqu'a la
// premiere position portant un en-tete valide a slot STRICTEMENT croissant.
func ti11LongueursPayload(pay []byte, cibles []int, long map[int]map[int]int) {
	bits := len(pay) * 8
	vise := map[int]bool{}
	for _, ti := range cibles {
		vise[ti] = true
	}
	for _, r := range WalkKeyframeWorld(pay) {
		if !vise[r.TI] {
			continue
		}
		for d := keyframeHeaderBits; d <= ti11FenetreMax; d++ {
			h, ok := readKeyframeHeader(pay, r.Bit+d, bits)
			if ok && h.Slot > r.Slot {
				long[r.TI][d]++
				break
			}
		}
	}
}

// ti11Dominante rend la valeur la plus frequente et sa part.
func ti11Dominante(m map[int]int) string {
	total, meilleur, n := 0, 0, 0
	for d, k := range m {
		total += k
		if k > n || (k == n && d < meilleur) {
			meilleur, n = d, k
		}
	}
	if total == 0 {
		return "aucun record mesure"
	}
	return fmt.Sprintf("%d record(s), dominante %d bits (%.1f %%), %d longueur(s) distincte(s)",
		total, meilleur, ti11Part(n, total), len(m))
}

// ti11TopLongueurs rend les longueurs les plus frequentes.
func ti11TopLongueurs(m map[int]int, max int) string {
	type l struct{ d, n int }
	ls := make([]l, 0, len(m))
	total := 0
	for d, n := range m {
		ls = append(ls, l{d, n})
		total += n
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].n != ls[j].n {
			return ls[i].n > ls[j].n
		}
		return ls[i].d < ls[j].d
	})
	var sb strings.Builder
	for i, x := range ls {
		if i >= max {
			break
		}
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%d bits x%d (%.0f %%)", x.d, x.n, ti11Part(x.n, total))
	}
	if sb.Len() == 0 {
		return "(aucune)"
	}
	return sb.String()
}

// ti11Verdict applique le predicat ecrit avant la mesure a la distribution de `ti=11`.
func ti11Verdict(t *testing.T, m map[int]int) {
	t.Helper()
	total, dans := 0, 0
	for d, n := range m {
		total += n
		if d >= ti11LongueurMin && d <= ti11LongueurMax {
			dans += n
		}
	}
	if total == 0 {
		t.Logf("VERDICT : aucun record ti=11 mesure — l'instrument n'a rien vu.")
		return
	}
	t.Logf("VERDICT : %d record(s) ti=11, %.1f %% tombent dans la fenetre predite %d..%d bits.",
		total, ti11Part(dans, total), ti11LongueurMin, ti11LongueurMax)
	t.Logf("  Une dominante NETTE dans cette fenetre confirme l'etat complet et valide les " +
		"largeurs portees. Une distribution etalee, ou une dominante hors fenetre, dit que le " +
		"record n'est pas ce qu'on croit — et le chiffre exact devient la prochaine cible.")
}

// TestObjectifTi11Residu — L'ECART ENTRE CE QUE JE CONSOMME ET CE QUE LE RECORD MESURE.
//
// # CE QUE LA LONGUEUR A TRANCHE
//
// Un record `ti=11` fait **168 bits, 93,8 % du temps** (dix longueurs distinctes seulement sur
// 2 611 records). L'etat complet est donc REFUTE : la somme des largeurs portees vaut deja
// 745 bits a elle seule. Le record est COURT et le masque epars est bien la.
//
// # LA MESURE QUI CLOT, ET ELLE NE COUTE RIEN
//
// La marche de production s'arrete a une position connue (`EntityTrace.EndBit`) ; la longueur
// mesuree en donne une autre (`ancre + distance au prochain en-tete`). Leur ECART est le nombre de
// bits que je ne consomme pas — et sa FORME dit ou il manque :
//
//	un ecart CONSTANT               un bloc fixe manque au prologue ou en queue de record ;
//	un ecart PROPORTIONNEL au       un cout PAR COMPOSANT manque (le bit de garde du mode film,
//	nombre de composants presents   deja documente, en est le candidat nomme) ;
//	un ecart ETALE                  les largeurs elles-memes sont fausses.
//
// L'ecart est donc rendu DEUX FOIS : sa distribution brute, et sa moyenne ventilee par nombre de
// composants presents. C'est la seconde qui separe les trois lectures.
//
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Residu -v -timeout 40m
func TestObjectifTi11Residu(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Residu", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	ecarts := map[int]int{}
	parNb := map[int][]int{}
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		reg, err := ti11Registre(dir)
		if err != nil {
			continue
		}
		n := CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				ti11ResiduPayload(pk.Payload(data), reg, ecarts, parNb)
			}
		}
	}

	t.Logf("########## ECART brut — %s", ti11Dominante(ecarts))
	t.Logf("   %s", ti11TopLongueurs(ecarts, 12))
	nbs := make([]int, 0, len(parNb))
	for k := range parNb {
		nbs = append(nbs, k)
	}
	sort.Ints(nbs)
	t.Logf("########## ECART ventile par nombre de composants presents")
	for _, k := range nbs {
		t.Logf("   %d composant(s) : %d record(s), ecart median %d bits, plage [%d .. %d]",
			k, len(parNb[k]), ti11Median(parNb[k]), ti11Min(parNb[k]), ti11Max(parNb[k]))
	}
}

// ti11ResiduPayload releve, pour chaque record ti=11 dont la longueur est mesurable, l'ecart
// entre la fin de marche et la fin reelle du record.
func ti11ResiduPayload(pay []byte, reg *Registry, ecarts map[int]int, parNb map[int][]int) {
	bits := len(pay) * 8
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ti11ArchIndex {
			continue
		}
		fin := -1
		for d := keyframeHeaderBits; d <= ti11FenetreMax; d++ {
			h, ok := readKeyframeHeader(pay, r.Bit+d, bits)
			if ok && h.Slot > r.Slot {
				fin = r.Bit + d
				break
			}
		}
		if fin < 0 {
			continue
		}
		br := NewBitReader(pay)
		br.SetBitPos(r.Bit + keyframeRecordTIBit)
		tr := TraverseEntity(br, reg, 0)
		if tr.TypeIndex != ti11ArchIndex || tr.DesyncAt >= 0 || tr.Mask>>ti11Composants != 0 {
			continue
		}
		nb := 0
		for i := 0; i < ti11Composants; i++ {
			if tr.Mask>>uint(i)&1 == 1 {
				nb++
			}
		}
		e := fin - tr.EndBit
		ecarts[e]++
		parNb[nb] = append(parNb[nb], e)
	}
}

func ti11Median(xs []int) int {
	tri := append([]int(nil), xs...)
	sort.Ints(tri)
	return tri[len(tri)/2]
}

func ti11Min(xs []int) int {
	m := xs[0]
	for _, x := range xs {
		if x < m {
			m = x
		}
	}
	return m
}

func ti11Max(xs []int) int {
	m := xs[0]
	for _, x := range xs {
		if x > m {
			m = x
		}
	}
	return m
}
