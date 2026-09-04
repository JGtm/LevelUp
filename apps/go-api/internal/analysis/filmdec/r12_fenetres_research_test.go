package filmdec

// r12_fenetres_research_test.go — LA QUESTION POSEE DANS L'AUTRE SENS.
//
// R8, R9 et R11 ont juge chaque canal contre un plancher de bruit, faute d'ancre. R12 en a
// cinq : les instants d'usage du repulseur releves au Theater sur `215e7022`. On ne demande
// donc plus « ce canal porte-t-il le repulseur ? » mais **« qu'est-ce qui bouge dans le film
// a ces cinq instants, chez le PORTEUR, et pas chez les autres ? »** — la methode qui a fait
// tomber le translocateur (R1) et le propulseur (R8).
//
// DEUX RECENSEMENTS, ET LEURS CONTROLES.
//
//	TETES D'EVENEMENT  Le premier evenement d'une liste se lit sans aucune grammaire :
//	                   `[1 bit config][1 bit continuation][R(7) type]`. Son cadrage est
//	                   CERTAIN (R7 par. 4.3, epreuve D). On compte donc, par type, le taux de
//	                   tetes dans les fenetres d'usage contre le taux hors fenetre. Aucun
//	                   deser, aucune table de largeur, aucune borne de carte : ce recensement
//	                   ne peut pas desynchroniser. C'est ce qui permet d'instruire le type 14
//	                   `PlayEffectOnObject` — report n°1 de R9 — SANS porter sa grammaire.
//	MASQUE BIPEDE      Le taux d'annonce des 64 composants, ancre : porteur de repulseur DANS
//	                   la fenetre, porteur de repulseur HORS fenetre, autres slots DANS la
//	                   fenetre, tout le film. Le recensement ne lit AUCUN deser (en-tete et
//	                   masque seulement) : il ne souffre d'aucune desynchronisation. C'est le
//	                   recensement de R9 par. 7.5 rendu ANCRE — celui-la ne pouvait comparer
//	                   que des rangs PORTES, il ne pouvait pas voir un geste.
//
// TEMOIN POSITIF INTERNE (pre-inscrit, par. 1.3 n°4 du rapport) : le meme instrument, ancre
// sur les instants d'usage du GRAPPIN de ce meme film (donnes par le tag 3 d'i59, la
// signature etablie en production), doit faire ressortir quelque chose. Sans lui, aucun
// negatif n'est publie.
//
// PIEGE HERITE DE R8 : `WorldObjectPrecision` est un GLOBAL DE PAQUET, pose depuis le layout
// du film par `r12Prepare` et restaure en sortie.
//
// GARDES : `R12_FILMS`, `R12_IDS`, `R12_FENETRE_MS` (defaut 1500). Aucune ecriture, aucune
// DuckDB, `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R12_FILMS=<repo>/data/cache/film_chunks R12_IDS=215e7022 \
//	  go test ./internal/analysis/filmdec/ -run '^TestR12Fenetres$' -count=1 -timeout 60m -v

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"
)

// r12FenetreMS : demi-largeur des fenetres d'usage, en millisecondes.
func r12FenetreMS() int64 {
	if v := os.Getenv("R12_FENETRE_MS"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			return int64(k)
		}
	}
	return 1500
}

// r12Fen est une fenetre nommee autour d'un instant.
type r12Fen struct {
	nom string
	lo  int64
	hi  int64
}

func r12Fenetres(anc []r12Ancre, half int64) []r12Fen {
	out := make([]r12Fen, 0, len(anc))
	for _, a := range anc {
		out = append(out, r12Fen{a.nom, a.ms - half, a.ms + half})
	}
	return out
}

func r12In(fs []r12Fen, ms int64) (string, bool) {
	for _, f := range fs {
		if ms >= f.lo && ms <= f.hi {
			return f.nom, true
		}
	}
	return "", false
}

// --- RECENSEMENT 1 : LES TETES D'EVENEMENT ------------------------------------------------

// r12Tete lit le type du PREMIER evenement de la liste d'un paquet delta. Cadrage CERTAIN :
// rien ne precede, donc aucune derive ne peut s'y produire. Rend -1 pour une liste vide.
func r12Tete(pay []byte) int {
	br := NewBitReader(pay)
	if br.Remaining() < 9 {
		return -1
	}
	br.Skip(1) // bit de configuration
	if !br.ReadBit() {
		return -1 // liste vide
	}
	return int(br.ReadBits(7))
}

// r12TeteBilan compte les tetes par type, dedans et dehors.
type r12TeteBilan struct {
	dedans  map[int]int
	dehors  map[int]int
	nDedans int
	nDehors int
}

func newR12TeteBilan() *r12TeteBilan {
	return &r12TeteBilan{dedans: map[int]int{}, dehors: map[int]int{}}
}

func (b *r12TeteBilan) ajoute(typ int, in bool) {
	if in {
		b.nDedans++
		b.dedans[typ]++
		return
	}
	b.nDehors++
	b.dehors[typ]++
}

// rapport publie les types tries par ECART DE TAUX, avec les deux denominateurs. Un type
// absent hors fenetre a un facteur infini : il est publie comme tel, jamais efface.
func (b *r12TeteBilan) rapport(t *testing.T, titre string) {
	t.Helper()
	t.Logf("  %s — %d paquets dans les fenetres, %d hors", titre, b.nDedans, b.nDehors)
	if b.nDedans == 0 {
		return
	}
	types := map[int]bool{}
	for k := range b.dedans {
		types[k] = true
	}
	for k := range b.dehors {
		types[k] = true
	}
	keys := make([]int, 0, len(types))
	for k := range types {
		keys = append(keys, k)
	}
	rateIn := func(k int) float64 { return float64(b.dedans[k]) / float64(b.nDedans) }
	rateOut := func(k int) float64 {
		if b.nDehors == 0 {
			return 0
		}
		return float64(b.dehors[k]) / float64(b.nDehors)
	}
	sort.Slice(keys, func(i, j int) bool {
		return rateIn(keys[i])-rateOut(keys[i]) > rateIn(keys[j])-rateOut(keys[j])
	})
	t.Logf("    %-4s %-40s %8s %8s %9s %9s %8s", "type", "nom", "dedans", "dehors",
		"tauxDed", "tauxHor", "facteur")
	for _, k := range keys {
		if b.dedans[k] == 0 && b.dehors[k] == 0 {
			continue
		}
		fac := "inf"
		if rateOut(k) > 0 {
			fac = fmt.Sprintf("%.2f", rateIn(k)/rateOut(k))
		}
		nom := r7Noms[k]
		if k < 0 {
			nom = "(liste vide)"
		}
		t.Logf("    %-4d %-40s %8d %8d %9.4f %9.4f %8s",
			k, nom, b.dedans[k], b.dehors[k], rateIn(k), rateOut(k), fac)
	}
}

// --- RECENSEMENT 2 : LE MASQUE BIPEDE, ANCRE ----------------------------------------------

// r12Cell : une population de records, avec son compte d'annonces par composant.
type r12Cell struct {
	records  int
	annonces [64]int
}

func (c *r12Cell) taux(i int) float64 {
	if c.records == 0 {
		return 0
	}
	return float64(c.annonces[i]) / float64(c.records)
}

// r12RankTimeline donne, pour un slot et un instant, le rang porte le plus recemment ANNONCE
// par i48 pour ce slot, et l'AGE de cette annonce. i48 n'emet qu'environ une fois par vie :
// l'age est publie parce qu'un rang vieux de deux minutes ne nomme plus rien (la lecon du
// par. 6 de R11, ou six « baisses repulseur » etaient des accroches de grappin attribuees par
// un rang de 117 a 162 s d'age).
type r12RankTimeline map[uint32][]r12RankRead

func r12BuildTimeline(rs []r12RankRead) r12RankTimeline {
	tl := r12RankTimeline{}
	for _, r := range rs {
		tl[r.Slot] = append(tl[r.Slot], r)
	}
	for s := range tl {
		v := tl[s]
		sort.SliceStable(v, func(a, b int) bool { return v[a].MS < v[b].MS })
		tl[s] = v
	}
	return tl
}

func (tl r12RankTimeline) at(slot uint32, ms int64) (rank int, age int64, ok bool) {
	v := tl[slot]
	best := -1
	for i := range v {
		if v[i].MS <= ms {
			best = i
			continue
		}
		break
	}
	if best < 0 {
		return 0, 0, false
	}
	return v[best].Rank, ms - v[best].MS, true
}

// --- L'INSTRUMENT --------------------------------------------------------------------------

func TestR12Fenetres(t *testing.T) {
	for _, dir := range r12FilmDirs(t) {
		r12FenetresOneFilm(t, dir)
	}
}

func r12FenetresOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	rd := r12Collect(s)
	pal := r12ClassifyPalette(rd.Ranks)
	rep := pal.r12RankOf("Repulsor")
	gra := pal.r12RankOf("Grappleshot")
	tl := r12BuildTimeline(rd.Ranks)
	half := r12FenetreMS()

	// Les instants du GRAPPIN, pour le temoin positif interne : le tag 3 d'i59, signature
	// etablie en production. Ils ne viennent PAS du releve — ils viennent du film.
	var graAnc []r12Ancre
	for _, tg := range rd.Tags {
		if tg.Src == "i59" && tg.Tag == 3 {
			graAnc = append(graAnc, r12Ancre{fmt.Sprintf("G@%s", r12MMSS(tg.MS)), tg.MS})
		}
	}

	t.Logf("=== FILM %s — fenetres +/-%d ms ===", s.id, half)
	t.Logf("  rangs : repulseur=%d grappin=%d ; %d ancres d'usage repulseur (releve), "+
		"%d ancres d'usage grappin (i59 tag 3)", rep, gra, len(r12AncresUsage), len(graAnc))

	fRep := r12Fenetres(r12AncresUsage, half)
	fGra := r12Fenetres(graAnc, half)

	r12TetesUneCampagne(t, s, fRep, "TETES — fenetres d'usage du REPULSEUR (releve)")
	r12TetesUneCampagne(t, s, fGra, "TETES — fenetres d'usage du GRAPPIN (temoin positif)")

	r12MasqueAncre(t, s, tl, fRep, rep, "REPULSEUR (rang 6, releve)")
	r12MasqueAncre(t, s, tl, fGra, gra, "GRAPPIN (rang 4, temoin positif)")
}

// r12TetesUneCampagne balaie les paquets delta et recense les tetes dedans / dehors.
func r12TetesUneCampagne(t *testing.T, s r12Setup, fs []r12Fen, titre string) {
	t.Helper()
	b := newR12TeteBilan()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			_, in := r12In(fs, s.ms(pk.TimestampUS))
			b.ajoute(r12Tete(pk.Payload(data)), in)
		}
	}
	b.rapport(t, titre)
}

// r12MasqueAncre recense les annonces des 64 composants du bipede en quatre populations.
func r12MasqueAncre(t *testing.T, s r12Setup, tl r12RankTimeline, fs []r12Fen,
	rank int, titre string) {
	t.Helper()
	if rank < 0 || len(fs) == 0 {
		t.Logf("  MASQUE ANCRE — %s : rang inconnu ou aucune ancre, mesure sautee", titre)
		return
	}
	var porteurDedans, porteurDehors, autreDedans, tout r12Cell
	r12Walk(s, func(slot uint32, ms int64, ids []int) bool {
		_, in := r12In(fs, ms)
		r, age, ok := tl.at(slot, ms)
		// FRAICHEUR : un rang de plus de 60 s ne nomme plus le porteur (lecon du par. 6 de
		// R11). Au-dela, le record n'entre dans aucune population de porteur.
		porteur := ok && r == rank && age <= 60000
		add := func(c *r12Cell) {
			c.records++
			for _, i := range ids {
				if i >= 0 && i < 64 {
					c.annonces[i]++
				}
			}
		}
		add(&tout)
		switch {
		case porteur && in:
			add(&porteurDedans)
		case porteur:
			add(&porteurDehors)
		case in:
			add(&autreDedans)
		}
		return false // recensement pur : aucun deser n'est deroule
	})
	t.Logf("  MASQUE ANCRE — %s", titre)
	t.Logf("    records : porteurDANS=%d porteurHORS=%d autreDANS=%d tout=%d",
		porteurDedans.records, porteurDehors.records, autreDedans.records, tout.records)
	if porteurDedans.records == 0 {
		t.Logf("    AUCUN record du porteur dans les fenetres : rien a comparer")
		return
	}
	type row struct {
		i                       int
		nom                     string
		pd, ph, ad              float64
		nPD, nPH, nAD, nTout    int
		facteurPort, facteurQui float64
	}
	var rows []row
	for i := 0; i < 64; i++ {
		if tout.annonces[i] == 0 {
			continue
		}
		r := row{i: i, nom: s.arch.component(i),
			pd: porteurDedans.taux(i), ph: porteurDehors.taux(i), ad: autreDedans.taux(i),
			nPD: porteurDedans.annonces[i], nPH: porteurDehors.annonces[i],
			nAD: autreDedans.annonces[i], nTout: tout.annonces[i]}
		if r.ph > 0 {
			r.facteurPort = r.pd / r.ph
		}
		if r.ad > 0 {
			r.facteurQui = r.pd / r.ad
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].pd-rows[a].ph > rows[b].pd-rows[b].ph })
	t.Logf("    %-4s %-46s %6s %6s %6s %8s %8s %8s %8s",
		"comp", "nom", "nPD", "nPH", "nAD", "tauxPD", "tauxPH", "facPort", "facQui")
	for _, r := range rows {
		t.Logf("    i%-3d %-46s %6d %6d %6d %8.4f %8.4f %8.2f %8.2f",
			r.i, r.nom, r.nPD, r.nPH, r.nAD, r.pd, r.ph, r.facteurPort, r.facteurQui)
	}
}
