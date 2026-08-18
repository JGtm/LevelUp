package replay

// drapeau_objet_controle_test.go — LE CONTROLE 3 du plan
// `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md` : les vies LIBRES du drapeau naissent-elles la ou
// un drapeau nait ?
//
// LA QUESTION, ET POURQUOI ELLE SUFFIT. La chaine ne decode aucune etiquette « drapeau » sur la
// piste : elle prend les vies `ti=42` dont l'identite est celle que le manifeste declare, et
// affirme que ce sont les vies libres du drapeau. Cette affirmation se refute par la GEOMETRIE,
// et de deux facons seulement — un drapeau libre ne peut apparaitre qu'a DEUX endroits :
//
//	A SON SOCLE            il vient d'etre rendu, capture, ou le match commence ;
//	LA OU IL EST TOMBE     son porteur vient de finir — le drapeau tombe a ses pieds.
//
// Une vie libre qui ne naitrait NI a un socle NI aux pieds d'un porteur qui vient de finir n'est
// pas un drapeau, et le dire coute une distance.
//
// LES SEUILS SONT ECRITS AVANT LA MESURE (decision 3 du plan) : >= 90 % des vies libres, et le
// TEMOIN — la MEME regle appliquee aux creations `ti=42` d'ARMES ORDINAIRES du meme film — doit
// rester <= 20 %. Sans ce temoin, un rayon trop genereux ferait passer n'importe quel objet du
// monde pour un drapeau : les armes au sol apparaissent partout sur la carte, elles doivent donc
// echouer la ou le drapeau reussit.
//
// LE RAYON N'EST PAS UN SEUIL NEUF : c'est `originDropMaxDist` (1,5 m), celui de la regle du
// lacher, declare chez son proprietaire (equipment_placements.go) et jamais redeclare ici.
//
// GARDE : `OBJ_FILM` (racine du cache film) et `OBJ_REPO` (racine du depot), comme toute la
// phase 1 des objectifs vivants. Lecture seule, aucune base.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache OBJ_REPO=<depot> \
//	  go test ./internal/analysis/replay/ -run DrapeauObjetControle -v -timeout 60m

import (
	"math"
	"testing"
)

// objDrapeauSeuilVies / objDrapeauSeuilTemoin — les deux seuils du controle 3, ecrits avant la
// mesure. Le second est ce qui rend le premier interpretable.
const (
	objDrapeauSeuilVies   = 0.90
	objDrapeauSeuilTemoin = 0.20
)

// objDrapeauLacherFenetreUS — l'ecart maximal, en microsecondes, entre la fin d'un portage et la
// naissance de la vie libre qu'il explique : UNE SECONDE.
//
// POURQUOI UNE SECONDE, ET POURQUOI ECRIT MAINTENANT. Le lacher est un evenement PHYSIQUE (le
// porteur tombe, l'objet est recree la), pas une transition de compteur : l'axe du rejeu avance
// par pas de 100 ms et les deux horloges (match et film) sont calees a la frame pres. Une
// seconde laisse dix pas de marge sans jamais rattraper le portage PRECEDENT, qui dure des
// dizaines de secondes. La fenetre est SYMETRIQUE — on ne suppose pas l'ordre entre la fin
// publiee et la creation lue, on mesure leur voisinage.
const objDrapeauLacherFenetreUS = 1_000_000

// objDrapeauRef est un point de reference du controle : un socle, ou la derniere position d'un
// porteur qui vient de finir (celle-ci datee, celle-la non).
type objDrapeauRef struct {
	x, y  float32
	atUS  uint64
	borne bool // vrai : la reference est DATEE (fin de portage) ; faux : c'est un socle
}

// TestDrapeauObjetControle — controle 3 sur les trois films CTF du corpus.
func TestDrapeauObjetControle(t *testing.T) {
	root := objRequireRoot(t)
	cat := goldenCatalog(t)
	if len(cat.FlagObjects) == 0 {
		t.Fatal("le manifeste du titre ne declare aucun drapeau — le controle n'a rien a mesurer")
	}
	joues, cumOK, cumN, cumTemOK, cumTemN := 0, 0, 0, 0, 0
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		joues++
		r := objDrapeauControleFilm(t, root, id, src, cat)
		cumOK, cumN = cumOK+r.viesOK, cumN+r.viesN
		cumTemOK, cumTemN = cumTemOK+r.temOK, cumTemN+r.temN
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", objFilmEnv, root)
	}
	part, temoin := objPart(cumOK, cumN), objPart(cumTemOK, cumTemN)
	t.Logf("CONTROLE 3 — VIES LIBRES DU DRAPEAU : %d/%d = %.1f %% naissent a moins de %.1f m "+
		"d'un socle ou du dernier point d'un porteur qui vient de finir (seuil %.0f %%) -> %s",
		cumOK, cumN, 100*part, originDropMaxDist, 100*objDrapeauSeuilVies,
		objTenu(part >= objDrapeauSeuilVies))
	t.Logf("CONTROLE 3 — TEMOIN (creations ti=42 d'ARMES ordinaires, MEME regle) : %d/%d = "+
		"%.1f %% (seuil <= %.0f %%) -> %s", cumTemOK, cumTemN, 100*temoin,
		100*objDrapeauSeuilTemoin, objTenu(temoin <= objDrapeauSeuilTemoin))
}

// objDrapeauMesure porte le resultat d'un film : les vies libres et leur temoin.
type objDrapeauMesure struct {
	viesOK, viesN, temOK, temN int
}

// objDrapeauControleFilm mesure UN film : les vies libres, puis le temoin des armes ordinaires.
func objDrapeauControleFilm(t *testing.T, root, id string, src *objDiskFilm,
	cat LabelCatalog) objDrapeauMesure {
	t.Helper()
	b := objBridgeOf(t, root, id)
	d := objDocumentDe(t, root, id, b, src)
	step := uint64(d.doc.FrameIntervalMS) * 1000
	if step == 0 {
		t.Fatalf("%s : axe de temps sans echelle — les instants ne se comparent pas", id)
	}
	refs := objDrapeauRefs(t, id, d, step)
	vies := flagFreeLives(d.gw, cat.FlagObjects)
	var m objDrapeauMesure
	m.viesN = len(vies)
	socle, porteur := 0, 0
	for _, l := range vies {
		x, y := l.First()
		s, p := objDrapeauPres(refs, x, y, l.T0US)
		socle, porteur = socle+objBool(s), porteur+objBool(p)
		m.viesOK += objBool(s || p)
	}
	t.Logf("%s : %d vies libres du drapeau — %d nees a un socle, %d au dernier point d'un "+
		"porteur qui vient de finir, %d ni l'un ni l'autre (%.1f %% tenues)",
		id, m.viesN, socle, porteur, m.viesN-m.viesOK, 100*objPart(m.viesOK, m.viesN))
	m.temOK, m.temN = objDrapeauTemoin(refs, d)
	t.Logf("%s : TEMOIN — %d creations ti=42 d'armes ordinaires, %d tenues (%.1f %%)",
		id, m.temN, m.temOK, 100*objPart(m.temOK, m.temN))
	return m
}

// objDrapeauRefs rassemble les points de reference du film : les socles `flag_spawn` du
// catalogue de carte, et la DERNIERE POSITION PUBLIEE de chaque porteur a la fin de son portage.
//
// LA POSITION DU PORTEUR EST LUE SUR SA PISTE, PAS SUR LE SPAN `dropped` — et c'est ce qui rend
// le controle non circulaire. Le span `dropped` est precisement ce que la phase 2 REPOSITIONNE
// sur la piste libre : le confronter a la piste libre reviendrait a comparer une grandeur a
// elle-meme.
func objDrapeauRefs(t *testing.T, id string, d objDoc, step uint64) []objDrapeauRef {
	t.Helper()
	var out []objDrapeauRef
	for _, s := range objFlagSpawns(t, id) {
		out = append(out, objDrapeauRef{x: s.X, y: s.Y})
	}
	idx := tracksByXUID(d.doc.Tracks)
	for _, f := range d.doc.FlagCarries {
		for _, s := range f.Spans {
			if !flagStateCarrying(s.State) || s.XUID == nil {
				continue
			}
			p, ok := pointOfXUIDAt(idx[*s.XUID], s.T1)
			if !ok {
				continue
			}
			out = append(out, objDrapeauRef{
				x: p.X, y: p.Y, atUS: d.originUS + uint64(s.T1)*step, borne: true,
			})
		}
	}
	return out
}

// objDrapeauPres dit si un point NAIT a un socle, et/ou au dernier point d'un porteur qui vient
// de finir. Les deux reponses sont rendues separement : leur somme ne se lit pas comme leur
// disjonction, et le rapport publie les trois.
func objDrapeauPres(refs []objDrapeauRef, x, y float32, atUS uint64) (socle, porteur bool) {
	for _, r := range refs {
		if math.Hypot(float64(x-r.x), float64(y-r.y)) >= originDropMaxDist {
			continue
		}
		if !r.borne {
			socle = true
			continue
		}
		if objEcartUS(r.atUS, atUS) <= objDrapeauLacherFenetreUS {
			porteur = true
		}
	}
	return socle, porteur
}

// objDrapeauTemoin rejoue la MEME regle sur les creations `ti=42` d'ARMES ORDINAIRES du film.
//
// C'EST LE DENOMINATEUR DE LA CREDIBILITE : les armes au sol apparaissent aux quatre coins de la
// carte et n'ont aucune raison de naitre a un socle de drapeau. Si elles tenaient le meme taux
// que le drapeau, le controle ne mesurerait que la generosite du rayon.
func objDrapeauTemoin(refs []objDrapeauRef, d objDoc) (ok, n int) {
	known := loadoutFamilies()
	for _, c := range d.gw.Creations {
		w, res := gwPadsIdentity(c)
		if !res || !known[w] {
			continue
		}
		n++
		s, p := objDrapeauPres(refs, c.X, c.Y, c.TimestampUS)
		ok += objBool(s || p)
	}
	return ok, n
}

// objEcartUS rend l'ecart absolu entre deux instants.
func objEcartUS(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// objBool rend 1 pour vrai, 0 pour faux — un compteur se lit mieux qu'un `if`.
func objBool(b bool) int {
	if b {
		return 1
	}
	return 0
}
