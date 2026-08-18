package filmdec

// zone_census_test.go — INSTRUMENT DE RECENSEMENT des archetypes de ZONE et d'OBJECTIF du
// mode, phase 0 (items C.0.1 et C.0.3) du lot C de
// `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`.
//
// CE QU'IL MESURE, et pourquoi il ne peut pas se contenter de la traversee canonique.
// La traversee d'un record (`traverseComponentLoop`) s'ARRETE au premier composant PRESENT
// non porte. Or ti=10 (`managed-object`) n'a qu'un composant porte sur 30, et ti=12
// (`managed-navpoint`) qu'un sur 28 : des qu'un record annonce autre chose qu'i0, la
// traversee lache et la SUITE DU PAQUET est perdue. Compter les composants annonces par la
// voie canonique reviendrait donc a ne compter que les premiers records de chaque paquet.
//
// LA VOIE UTILISEE est celle des scanners ti=37/41/42 : ANCRAGE PAR BANDE DE SLOTS
// (`worldObjectSlotBand` + `matchWorldObjectRecord`, projectiles.go), qui reconnait un
// en-tete de record delta a n'importe quelle position de bit sans rien devoir aux records
// precedents. Le masque de presence est lu SANS decoder aucune valeur au-dela de l'en-tete :
// c'est un recensement d'ANNONCES, pas un portage.
//
// TROIS BANDES DE CONTROLE, parce qu'un balayage par ancrage doit prouver qu'il reconnait
// une STRUCTURE et non du bruit :
//
//	reelle          les slots vus porter l'archetype dans les tables d'image-cle ;
//	fantome pur     autant de slots JAMAIS vus porter aucun archetype -> attendu ~0 ;
//	fantome occupe  autant de slots vus porter un archetype HORS du perimetre -> temoin
//	                POSITIF : il doit rendre des records, sinon le balayeur ne voit rien.
//
// IL NE MODIFIE RIEN : aucun code de production, aucune ecriture en base, aucun champ
// d'artefact. Lecture seule du film, sortie en TSV sous `.ai/V7.5/replay2d/registre_film/lotC/`.
// SOUS GARDE D'ENVIRONNEMENT (ZONE_FILM), donc saute partout ailleurs, CI comprise.
//
// UN SEUL FILM PAR PROCESSUS (memoire de depot `reference_statrecords_corpus_sweep_ram_bomb`,
// deux plantages machine en aout 2026) : la garde prend UN chemin, jamais une liste.
//
// USAGE (depuis apps/go-api) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestZoneCensusLotC -v ./internal/analysis/filmdec/

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	// zcFilmEnv porte le repertoire des chunks d'UN film (chemin absolu).
	zcFilmEnv = "ZONE_FILM"
	// zcOutEnv permet de rediriger les TSV ; par defaut ils vont sous `.ai/` du depot.
	zcOutEnv = "ZONE_OUT"
)

// zcTargetTIs sont les archetypes recenses par l'item C.0.1, dans l'ordre du plan. ti=11 y
// est COMPTE (recensement) et rien de plus : la regle dure du plan interdit de le porter.
var zcTargetTIs = []int{10, 12, 23, 11, 13, 47, 4}

// zcWindowMS est la demi-fenetre autour d'un evenement d'objectif, telle que le gate 0 du
// lot C la fixe (+/- 3 s). Ecrite AVANT la mesure, jamais ajustee apres.
const zcWindowMS = 3000

// zcDir rend le repertoire du film, ou saute le test.
func zcDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(zcFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de recensement saute", zcFilmEnv)
	}
	return dir
}

// zcOutDir rend le repertoire de sortie des TSV et le cree au besoin.
func zcOutDir(t *testing.T) string {
	t.Helper()
	if v := os.Getenv(zcOutEnv); v != "" {
		mustMkdir(t, v)
		return v
	}
	root := zcModuleRoot(t)
	out := filepath.Join(root, "..", "..", ".ai", "V7.5", "replay2d", "registre_film", "lotC")
	mustMkdir(t, out)
	return out
}

// zcModuleRoot remonte jusqu'au go.mod (racine du module Go, apps/go-api).
func zcModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd : %v", err)
	}
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod introuvable au-dessus du repertoire de test")
	return ""
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creation de %s : %v", dir, err)
	}
}

// -------------------------------------------------------------------------------------
// RECENSEMENT DES TABLES D'IMAGE-CLE (C.0.1 partie 1)
// -------------------------------------------------------------------------------------

// zcCensus est le recensement des tables d'image-cle d'un film : combien de records par
// archetype, quels slots, et quels slots sont AMBIGUS (vus porter plusieurs archetypes).
type zcCensus struct {
	dir       string
	chunks    int
	keyframes int
	recordsTI map[int]int
	slotsTI   map[int]map[uint32]bool
	// slotTIs[slot] = l'ensemble des archetypes vus sur ce slot. Un slot a plusieurs
	// archetypes quand le pool de slots reboucle en cours de partie.
	slotTIs map[uint32]map[int]bool
}

// zcKeyframeCensus recense les records de toutes les tables d'image-cle du film. HORS LIGNE.
func zcKeyframeCensus(dir string) zcCensus {
	c := zcCensus{
		dir: dir, chunks: CountFilmChunks(dir),
		recordsTI: map[int]int{}, slotsTI: map[int]map[uint32]bool{},
		slotTIs: map[uint32]map[int]bool{},
	}
	for i := 1; i <= c.chunks; i++ {
		data, err := ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			c.keyframes++
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				c.recordsTI[r.TI]++
				if c.slotsTI[r.TI] == nil {
					c.slotsTI[r.TI] = map[uint32]bool{}
				}
				c.slotsTI[r.TI][uint32(r.Slot)] = true
				if c.slotTIs[uint32(r.Slot)] == nil {
					c.slotTIs[uint32(r.Slot)] = map[int]bool{}
				}
				c.slotTIs[uint32(r.Slot)][r.TI] = true
			}
		}
	}
	return c
}

// totalRecords rend le denominateur du recensement d'image-cle.
func (c zcCensus) totalRecords() int {
	n := 0
	for _, v := range c.recordsTI {
		n += v
	}
	return n
}

// topOtherTIs rend les n archetypes HORS perimetre les plus volumineux en image-cle — le
// plan demande les 3 premiers autres par volume.
func (c zcCensus) topOtherTIs(n int) []int {
	target := map[int]bool{}
	for _, ti := range zcTargetTIs {
		target[ti] = true
	}
	var others []int
	for ti := range c.recordsTI {
		if !target[ti] {
			others = append(others, ti)
		}
	}
	sort.Slice(others, func(i, j int) bool {
		if c.recordsTI[others[i]] != c.recordsTI[others[j]] {
			return c.recordsTI[others[i]] > c.recordsTI[others[j]]
		}
		return others[i] < others[j]
	})
	if len(others) > n {
		others = others[:n]
	}
	return others
}

// zcSortedSlots rend les slots d'un ensemble, tries — la stabilite des NUMEROS entre films
// de la meme carte ne se lit que sur une liste ordonnee.
func zcSortedSlots(s map[uint32]bool) []int {
	out := make([]int, 0, len(s))
	for k := range s {
		out = append(out, int(k))
	}
	sort.Ints(out)
	return out
}

// -------------------------------------------------------------------------------------
// BANDES DE SLOTS ET TEMOINS (C.0.1 partie 3)
// -------------------------------------------------------------------------------------

// Les CLASSES de slot du balayage. Les valeurs negatives sont les bandes de CONTROLE ; une
// valeur >= 0 est un archetype cible. Une seule table slot -> classe permet de balayer les
// quatre bandes en UNE passe : une position de bit ne rend qu'une valeur de slot, donc au
// plus une bande — balayer separement rendrait exactement les memes comptes quatre fois plus
// cher (la machine de l'utilisateur paie, D17).
const (
	// zcClassInconnu : slots absents de TOUTE table d'image-cle. CE N'EST PAS un temoin
	// vide : les entites de courte vie (projectiles) n'apparaissent presque jamais en
	// image-cle (`worldObjectSlotBand` documente 19 slots vus sur 580 vies decodees), donc
	// cette bande porte du signal REEL en plus du bruit. Elle mesure le plafond « bruit +
	// entites fugaces ».
	zcClassInconnu = -1
	// zcClassOccupe : slots vus porter un archetype HORS perimetre. Temoin POSITIF — il doit
	// rendre beaucoup de records, sinon le balayeur ne voit rien du tout.
	zcClassOccupe = -2
	// zcClassVide : slots du HAUT de l'espace de slots, jamais vus, au-dela de tout slot
	// observe. C'est le temoin le plus proche d'un vrai vide : ce qu'il rend est du BRUIT DE
	// RECONNAISSANCE pur (motifs de bits qui satisfont l'en-tete par hasard).
	zcClassVide = -3
)

// zcBands porte les bandes de l'ancrage : la bande reelle par archetype, la table
// slot -> classe qui les balaie toutes en une passe, et les trois bandes de controle.
type zcBands struct {
	// perTI[ti] = les slots OBSERVES de l'archetype, purges des slots ambigus.
	perTI map[int]map[uint32]bool
	// all est l'union de TOUTES les bandes (reelles et de controle) ; class attribue chaque
	// slot a son archetype (>= 0) ou a sa bande de controle (< 0).
	all   map[uint32]bool
	class map[uint32]int
	// cardinalite des bandes de controle, publiee avec les comptes.
	nInconnu, nOccupe, nVide int
	// union est le nombre de slots des bandes reelles (denominateur des temoins).
	union int
	// ambigus compte les slots ecartes parce que vus porter plusieurs archetypes.
	ambigus int
}

// zcBuildBands construit les bandes. LE CHOIX DE LA BANDE OBSERVEE (et non comblee) est
// celui du lot R4 et pour la meme raison : un objet de mode vit toute la partie, donc il
// est present a CHAQUE image-cle. Combler la plage ne recupererait aucune couverture et
// avalerait les slots voisins (`worldObjectSlotBand` existe pour les projectiles, qui
// vivent moins d'une seconde — l'exact contraire).
func zcBuildBands(c zcCensus) zcBands {
	b := zcBands{perTI: map[int]map[uint32]bool{}, all: map[uint32]bool{}, class: map[uint32]int{}}
	target := map[int]bool{}
	for _, ti := range zcTargetTIs {
		target[ti] = true
		b.perTI[ti] = map[uint32]bool{}
	}
	lo := uint32(kfTableCap)
	for slot, tis := range c.slotTIs {
		var inTarget []int
		for ti := range tis {
			if target[ti] {
				inTarget = append(inTarget, ti)
			}
		}
		if len(inTarget) == 0 {
			continue
		}
		if len(inTarget) > 1 || len(tis) > 1 {
			b.ambigus++ // un slot recycle ne peut pas etre attribue : il est ecarte
			continue
		}
		ti := inTarget[0]
		b.perTI[ti][slot] = true
		b.all[slot] = true
		b.class[slot] = ti
		b.union++
		if slot < lo {
			lo = slot
		}
	}
	b.nInconnu = b.addControl(c, zcClassInconnu, lo, b.union)
	b.nOccupe = b.addControl(c, zcClassOccupe, lo, b.union)
	b.nVide = b.addControl(c, zcClassVide, lo, b.union)
	return b
}

// addControl remplit une bande de controle de `size` slots et rend sa cardinalite reelle.
//
// LES DEUX PREMIERES sont tirees dans le VOISINAGE NUMERIQUE des slots reels (precaution du
// lot R4 : un balayage qui reconnait un slot sur 13 bits n'a pas la meme chance de tomber
// juste sur un petit numero que sur un grand, donc un temoin tire depuis 1 fausserait la
// comparaison). LA TROISIEME est au contraire tiree du HAUT de l'espace, au-dela de tout
// slot observe : c'est le prix a payer pour disposer d'un vide reel.
func (b *zcBands) addControl(c zcCensus, class int, lo uint32, size int) int {
	if size == 0 {
		return 0
	}
	n := 0
	add := func(s uint32) bool {
		if b.all[s] || !zcEligible(c, s, class) {
			return false
		}
		b.all[s], b.class[s] = true, class
		n++
		return n >= size
	}
	if class == zcClassVide {
		for s := uint32(kfTableCap - 1); s > 0 && n < size; s-- {
			if add(s) {
				break
			}
		}
		return n
	}
	for s := lo; s < kfTableCap && n < size; s++ {
		if add(s) {
			return n
		}
	}
	for s := uint32(1); s < lo && n < size; s++ {
		if add(s) {
			break
		}
	}
	return n
}

// zcEligible dit si un slot peut entrer dans une bande de controle donnee.
func zcEligible(c zcCensus, s uint32, class int) bool {
	tis, seen := c.slotTIs[s]
	switch class {
	case zcClassOccupe:
		return seen && !zcAnyTarget(tis)
	case zcClassInconnu, zcClassVide:
		return !seen
	}
	return false
}

// zcAnyTarget dit si l'ensemble d'archetypes observes sur un slot touche le perimetre.
func zcAnyTarget(tis map[int]bool) bool {
	for _, ti := range zcTargetTIs {
		if tis[ti] {
			return true
		}
	}
	return false
}

// zcMaxSlotSeen rend le plus grand slot vu en image-cle — il situe le temoin VIDE, qui est
// tire au-dessus de lui.
func zcMaxSlotSeen(c zcCensus) uint32 {
	var hi uint32
	for s := range c.slotTIs {
		if s > hi {
			hi = s
		}
	}
	return hi
}
