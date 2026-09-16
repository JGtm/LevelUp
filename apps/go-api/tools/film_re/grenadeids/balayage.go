//go:build research

package grenadeids

// balayage.go — LES TROIS PASSES DE LA QUESTION (1), EN UNE SEULE LECTURE DU FLUX DELTA.
//
// Le parcours est celui de `grammar.ScanGrenadeThrows` — un balayage d octets pur, bit a bit,
// sur les payloads des paquets de type delta — mais il lit TRENTE-DEUX bits a chaque position
// au lieu de vingt-quatre. Les vingt-quatre bits de poids fort de cette lecture SONT le
// marqueur (meme convention MSB-first, meme bourrage a zero au-dela du tampon), si bien qu une
// seule lecture sert la recherche du marqueur ET la recherche absolue d un identifiant.

import (
	"path/filepath"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// structureBits : la taille de la structure exploitee par la production — marqueur, identifiant,
// bourrage, index joueur. Recopiee de `grammar.grenadeThrowBits`, qui est prive.
const structureBits = 24 + 32 + 47 + 5

// SourceMarqueur dit de quel marqueur vient une occurrence.
type SourceMarqueur int

const (
	// MarqueurProduction : la constante 0x4C0C00 que le decodeur cherche aujourd hui.
	MarqueurProduction SourceMarqueur = iota
	// MarqueurRegistre : le marqueur derive de l archetype projectile LU DANS CE FILM.
	MarqueurRegistre

	// MarqueurImpair : le marqueur de production dont le DERNIER bit vaut 1 — c est-a-dire
	// l amorce de 23 bits suivie d un champ d identifiant dont le bit de poids fort est a 1.
	// Voir `.ai/V7.5/film_re/NOTE_3_3_IDENTIFIANTS_GRENADE_2026-09-16.md` §6 : sur les builds
	// anciens le champ commence un bit plus tot, si bien que le vingt-quatrieme bit filtre par
	// la production N EST PAS de l amorce mais le premier bit de l identifiant.
	MarqueurImpair
)

// Options regle les trois passes.
type Options struct {
	// Fenetre : demi-largeur, en bits, du balayage de decalage autour de la position de
	// production (`marqueur + 24`). Zero = pas de passe A.
	Fenetre int
	// FenetreIndex : demi-largeur du balayage de la position du champ d index. 0 = pas de
	// balayage.
	FenetreIndex int
	// Dump : identifiant dont on releve la suite de bits autour du marqueur ; 0 = aucun releve.
	Dump uint32
	// DumpMax : nombre maximal de tranches relevees par film.
	DumpMax int
	// Voisinage : distance maximale, en bits, pour rattacher une occurrence absolue au
	// marqueur le plus proche. Au-dela, l occurrence est comptee « hors marqueur ».
	Voisinage int
}

// Occurrence est un marqueur trouve dans le flux, et ce que la grammaire de production y lit.
type Occurrence struct {
	Source        SourceMarqueur
	Chunk, Paquet int
	BitPos        int
	TimestampUS   uint64
	// ID est la valeur de 32 bits a `BitPos+24` — ce que la production compare a la liste
	// blanche. Index est le champ de 5 bits a `BitPos+103`.
	ID    uint32
	Index int
	// TypeIndex est l archetype REELLEMENT en train de naitre, les six bits du record (le bit
	// de poids fort est juste avant le marqueur — cf. typeindex.go). TiIndetermine dit que ce
	// bit etait hors du payload : seul `TypeIndex mod 32` est alors connu.
	TypeIndex     int
	TiIndetermine bool
	// IndexAlt est le meme champ de 5 bits lu UN BIT PLUS TOT (+102). Si tout le record est
	// decale d un bit sur les builds anciens, c est LUI qui porte l index joueur plausible.
	IndexAlt int
	// IDAlt est la valeur de 32 bits lue a +23 — la position des builds anciens.
	IDAlt uint32
}

// Candidat est une valeur observee derriere un marqueur, et ce qu on en sait.
type Candidat struct {
	ID                 uint32
	N                  int
	Rang               int
	Connu              bool
	IndexMin, IndexMax int
	IndexHors8         int
	// AltMin / AltMax / AltHors8 : les memes bornes pour la lecture a +102.
	AltMin, AltMax int
	AltHors8       int
}

// Releve porte la mesure d UN film.
type Releve struct {
	// Marqueurs : le compte par source de marqueur.
	Marqueurs map[SourceMarqueur]int
	// ParDecalage[source][d][id] : passe A — combien de fois l identifiant actuel `id` est lu
	// a `marqueur + 24 + d`. Le decalage 0 est la lecture de production.
	ParDecalage map[SourceMarqueur]map[int]map[uint32]int
	// Absolus[id] : passe B — occurrences de l identifiant actuel a N IMPORTE QUELLE position
	// de bit du flux delta, marqueur ou pas.
	Absolus map[uint32]int
	// DistanceAuMarqueur[id][d] : passe B — pour chaque occurrence absolue, l ecart au
	// marqueur de production le plus proche, quand il est dans le voisinage.
	DistanceAuMarqueur map[uint32]map[int]int
	// HorsMarqueur[id] : passe B — occurrences absolues qu aucun marqueur ne borde.
	HorsMarqueur map[uint32]int
	// Famille : passe C — l histogramme SANS liste blanche de ce qui suit le marqueur de
	// production, trie par frequence decroissante.
	Famille []Candidat
	// FamilleRegistre : le meme histogramme pour le marqueur derive du registre, quand il
	// differe de celui de production.
	FamilleRegistre []Candidat
	// FamilleImpair : l histogramme derriere le marqueur IMPAIR (amorce de 23 bits + bit de
	// poids fort d un identifiant a 1). Sur ces occurrences, l identifiant se lit a +23.
	FamilleImpair []Candidat
	// ParTypeIndex[ti] : combien de marqueurs appartiennent REELLEMENT a l archetype ti. Sur le
	// marqueur de production, `41` et `9` s y separent (decouverte D2 (3.3r)).
	ParTypeIndex map[int]int
	// FamilleParTi[ti] : l histogramme de la passe C restreint aux naissances de ti — ce qui
	// suit une naissance de `managed-player` n a rien a faire dans la famille des grenades.
	FamilleParTi map[int][]Candidat
	// IndexParDecalage / LancersConfirmes : le balayage de la position du champ d index.
	IndexParDecalage map[int]*CompteIndex
	LancersConfirmes int
	// TiIndetermines : marqueurs en tete de payload, dont le sixieme bit d index manque.
	TiIndetermines int
	// Tranches : les suites de bits relevees autour des marqueurs portant `Options.Dump`.
	Tranches []Tranche
	// Occurrences : les marqueurs, conserves pour la passe D (appariement a i22).
	Occurrences []Occurrence
	// PaquetsDelta et OctetsDelta disent la matiere balayee.
	PaquetsDelta int
	OctetsDelta  int
}

// cheminFilm rend le repertoire d un film sous la racine donnee. La racine est un PARAMETRE :
// aucun nom de sous-dossier du cache n est ecrit dans cet instrument.
func cheminFilm(racine, id string) string { return filepath.Join(racine, id) }

// Balayer joue les passes A, B et C sur une bobine deja ouverte.
func Balayer(b *Bobine, opt Options) *Releve {
	r := &Releve{
		Marqueurs:          map[SourceMarqueur]int{},
		ParDecalage:        map[SourceMarqueur]map[int]map[uint32]int{},
		Absolus:            map[uint32]int{},
		DistanceAuMarqueur: map[uint32]map[int]int{},
		HorsMarqueur:       map[uint32]int{},
		ParTypeIndex:       map[int]int{},
		FamilleParTi:       map[int][]Candidat{},
	}
	f := familles{
		prod:     map[uint32]*Candidat{},
		registre: map[uint32]*Candidat{},
		impair:   map[uint32]*Candidat{},
		parTi:    map[int]map[uint32]*Candidat{},
	}
	for _, c := range grammar.FilmChunkNumbers(b.Film()) {
		chunk, pks, ok := grammar.FilmChunkAt(b.Film(), c)
		if !ok {
			continue
		}
		for _, p := range pks {
			if p.Type != grammar.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			r.PaquetsDelta++
			r.OctetsDelta += len(pay)
			balayerPayload(pay, b, c, p, r, opt, f)
		}
	}
	r.Famille = trierFamille(f.prod)
	r.FamilleRegistre = trierFamille(f.registre)
	r.FamilleImpair = trierFamille(f.impair)
	for ti, compte := range f.parTi {
		r.FamilleParTi[ti] = trierFamille(compte)
	}
	return r
}

// familles porte les histogrammes de la passe C, pour tenir la regle des cinq parametres : les
// deux par SOURCE de marqueur, et celui par ARCHETYPE reellement en train de naitre.
type familles struct {
	prod, registre, impair map[uint32]*Candidat
	parTi                  map[int]map[uint32]*Candidat
}

// noterTypeIndex compte le marqueur sous l archetype qui nait VRAIMENT derriere lui, et
// accumule l histogramme de la passe C restreint a cet archetype.
//
// Une occurrence dont le sixieme bit d index manque est comptee a part et n entre dans AUCUN
// histogramme par archetype : elle vaudrait `ti mod 32`, c est-a-dire le mauvais archetype une
// fois sur deux.
func noterTypeIndex(r *Releve, f familles, o Occurrence) {
	if o.TiIndetermine {
		r.TiIndetermines++
		return
	}
	r.ParTypeIndex[o.TypeIndex]++
	compte := f.parTi[o.TypeIndex]
	if compte == nil {
		compte = map[uint32]*Candidat{}
		f.parTi[o.TypeIndex] = compte
	}
	noterCandidat(compte, o)
}

// balayerPayload traite UN payload de paquet delta : une lecture de 32 bits par position, puis
// les deux post-traitements (decalages autour de chaque marqueur, rattachement des occurrences
// absolues).
func balayerPayload(pay []byte, b *Bobine, chunk int, p grammar.FilmPacket, r *Releve,
	opt Options, f familles) {
	marqueurs, absolus := lireLesDeuxSignaux(pay, b, chunk, p, r, f)
	for _, m := range marqueurs {
		compterDecalages(pay, m, opt.Fenetre, r)
		balayerIndexAuteur(pay, m, opt.FenetreIndex, r)
		if opt.Dump != 0 && m.ID == opt.Dump && len(r.Tranches) < opt.DumpMax {
			r.Tranches = append(r.Tranches, releverTranche(pay, m))
		}
	}
	rattacherAbsolus(marqueurs, absolus, opt.Voisinage, r)
}

// lireLesDeuxSignaux fait LA lecture : a chaque position de bit, une valeur de 32 bits dont les
// 24 bits de poids fort sont le marqueur candidat et les 32 bits entiers un identifiant
// candidat. Rend les marqueurs trouves et les positions des occurrences absolues.
func lireLesDeuxSignaux(pay []byte, b *Bobine, chunk int, p grammar.FilmPacket, r *Releve,
	f familles) ([]Occurrence, []occAbsolue) {
	bits := len(pay) * 8
	limMarqueur, limAbsolu := bits-structureBits, bits-32
	var marqueurs []Occurrence
	var absolus []occAbsolue
	for bp := 0; bp <= limAbsolu; bp++ {
		v := grammar.PeekBits(pay, bp, 32)
		if id := uint32(v); estIdentifiantActuel(id) {
			r.Absolus[id]++
			absolus = append(absolus, occAbsolue{bitPos: bp, id: id})
		}
		if bp > limMarqueur {
			continue
		}
		switch m := v >> 8; {
		case m == b.MarqueurProd:
			o := occurrenceAu(pay, bp, chunk, p, MarqueurProduction)
			r.Marqueurs[MarqueurProduction]++
			noterCandidat(f.prod, o)
			noterTypeIndex(r, f, o)
			marqueurs = append(marqueurs, o)
		case m == b.Marqueur:
			o := occurrenceAu(pay, bp, chunk, p, MarqueurRegistre)
			r.Marqueurs[MarqueurRegistre]++
			noterCandidat(f.registre, o)
			noterTypeIndex(r, f, o)
			marqueurs = append(marqueurs, o)
		case m == b.MarqueurProd|1:
			o := occurrenceAu(pay, bp, chunk, p, MarqueurImpair)
			r.Marqueurs[MarqueurImpair]++
			noterCandidat(f.impair, o)
			noterTypeIndex(r, f, o)
			marqueurs = append(marqueurs, o)
		}
	}
	r.Occurrences = append(r.Occurrences, marqueurs...)
	return marqueurs, absolus
}

// occAbsolue : une occurrence d un identifiant actuel trouvee hors de toute hypothese de
// marqueur.
type occAbsolue struct {
	bitPos int
	id     uint32
}

// occurrenceAu lit, a la position d un marqueur, ce que la grammaire de production y lit.
func occurrenceAu(pay []byte, bp, chunk int, p grammar.FilmPacket, src SourceMarqueur) Occurrence {
	ti, indetermine := typeIndexDuRecord(pay, bp)
	return Occurrence{
		Source:        src,
		Chunk:         chunk,
		Paquet:        p.Index,
		BitPos:        bp,
		TimestampUS:   p.TimestampUS,
		ID:            uint32(grammar.PeekBits(pay, bp+24, 32)),
		Index:         int(grammar.PeekBits(pay, bp+24+32+47, 5)),
		IndexAlt:      int(grammar.PeekBits(pay, bp+24+32+47-1, 5)),
		IDAlt:         uint32(grammar.PeekBits(pay, bp+23, 32)),
		TypeIndex:     ti,
		TiIndetermine: indetermine,
	}
}

// compterDecalages joue la PASSE A autour d UN marqueur : pour chaque decalage d de la fenetre,
// la valeur de 32 bits lue a `marqueur + 24 + d` est-elle un des quatre identifiants actuels ?
func compterDecalages(pay []byte, m Occurrence, fenetre int, r *Releve) {
	if fenetre <= 0 {
		return
	}
	parSource := r.ParDecalage[m.Source]
	if parSource == nil {
		parSource = map[int]map[uint32]int{}
		r.ParDecalage[m.Source] = parSource
	}
	for d := -fenetre; d <= fenetre; d++ {
		id := uint32(grammar.PeekBits(pay, m.BitPos+24+d, 32))
		if !estIdentifiantActuel(id) {
			continue
		}
		if parSource[d] == nil {
			parSource[d] = map[uint32]int{}
		}
		parSource[d][id]++
	}
}

// rattacherAbsolus joue la seconde moitie de la PASSE B : chaque occurrence absolue recoit son
// ecart au marqueur de PRODUCTION le plus proche, ou est comptee hors marqueur.
func rattacherAbsolus(marqueurs []Occurrence, absolus []occAbsolue, voisinage int, r *Releve) {
	var bases []int
	for _, m := range marqueurs {
		if m.Source == MarqueurProduction {
			bases = append(bases, m.BitPos+24)
		}
	}
	for _, a := range absolus {
		d, ok := ecartAuPlusProche(bases, a.bitPos, voisinage)
		if !ok {
			r.HorsMarqueur[a.id]++
			continue
		}
		if r.DistanceAuMarqueur[a.id] == nil {
			r.DistanceAuMarqueur[a.id] = map[int]int{}
		}
		r.DistanceAuMarqueur[a.id][d]++
	}
}

// ecartAuPlusProche rend `pos - base` pour la base la plus proche, et si elle est dans le
// voisinage. `bases` est croissante par construction (le balayage va dans le sens des bits).
func ecartAuPlusProche(bases []int, pos, voisinage int) (int, bool) {
	if len(bases) == 0 {
		return 0, false
	}
	i := sort.SearchInts(bases, pos)
	meilleur, trouve := 0, false
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= len(bases) {
			continue
		}
		d := pos - bases[j]
		if !trouve || abs(d) < abs(meilleur) {
			meilleur, trouve = d, true
		}
	}
	if !trouve || abs(meilleur) > voisinage {
		return 0, false
	}
	return meilleur, true
}

// abs rend la valeur absolue d un entier.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// estIdentifiantActuel dit si la valeur est un des quatre identifiants de la liste blanche.
func estIdentifiantActuel(id uint32) bool {
	_, ok := grammar.GrenadeRankOf(id)
	return ok
}

// noterCandidat accumule l histogramme de la PASSE C.
func noterCandidat(compte map[uint32]*Candidat, o Occurrence) {
	c := compte[o.ID]
	if c == nil {
		rang, connu := grammar.GrenadeRankOf(o.ID)
		c = &Candidat{ID: o.ID, Rang: rang, Connu: connu, IndexMin: o.Index, IndexMax: o.Index,
			AltMin: o.IndexAlt, AltMax: o.IndexAlt}
		compte[o.ID] = c
	}
	c.N++
	if o.Index < c.IndexMin {
		c.IndexMin = o.Index
	}
	if o.Index > c.IndexMax {
		c.IndexMax = o.Index
	}
	if o.Index > 7 {
		c.IndexHors8++
	}
	if o.IndexAlt < c.AltMin {
		c.AltMin = o.IndexAlt
	}
	if o.IndexAlt > c.AltMax {
		c.AltMax = o.IndexAlt
	}
	if o.IndexAlt > 7 {
		c.AltHors8++
	}
}

// trierFamille rend l histogramme trie par frequence decroissante, puis par identifiant.
func trierFamille(compte map[uint32]*Candidat) []Candidat {
	out := make([]Candidat, 0, len(compte))
	for _, v := range compte {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].ID < out[j].ID
	})
	return out
}
