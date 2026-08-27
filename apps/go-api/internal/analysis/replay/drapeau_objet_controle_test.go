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
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// objDrapeauSeuilVies / objDrapeauSeuilTemoin — les deux seuils du controle 3, ecrits avant la
// mesure. Le second est ce qui rend le premier interpretable.
const (
	objDrapeauSeuilVies   = 0.90
	objDrapeauSeuilTemoin = 0.20
)

// objDrapeauLacherFenetreUS — la fenetre de lacher du controle, en microsecondes. C'est LA
// CONSTANTE DE PRODUCTION (`flagFreeDropWindowMS`, flag_objects.go), convertie : l'instrument
// mesure sur l'horloge du FILM, la regle borne sur celle du MATCH, et les deux durees sont la
// meme. La redeclarer donnerait deux valeurs a un seul seuil.
const objDrapeauLacherFenetreUS = flagFreeDropWindowMS * 1000

// objDrapeauRef est un point de reference du controle : un socle (fixe, valable a tout instant)
// ou UN PORTAGE (une piste de porteur, valable pendant le portage et la seconde qui le suit).
type objDrapeauRef struct {
	// x, y : la position du socle. Renseignes seulement quand `porteur` est vide.
	x, y float32
	// porteur est la piste PUBLIEE du porteur, lue a l'instant demande ; t0US / t1US bornent
	// son portage sur l'horloge du FILM.
	porteur    []Track
	t0US, t1US uint64
}

// TestDrapeauObjetControle — controle 3 sur les trois films CTF du corpus.
func TestDrapeauObjetControle(t *testing.T) {
	root := objRequireRoot(t)
	cat := goldenCatalog(t)
	if len(cat.ObjectiveObjects) == 0 {
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
	vies := flagFreeLives(d.gw, cat.ObjectiveObjects)
	var m objDrapeauMesure
	m.viesN = len(vies)
	socle, porteur, residu := 0, 0, 0
	for i, l := range vies {
		x, y := l.First()
		s, p := objDrapeauPres(refs, x, y, l.T0US, step, d.originUS)
		socle, porteur = socle+objBool(s), porteur+objBool(p)
		m.viesOK += objBool(s || p)
		if !s && !p && objDrapeauResidu(vies, i) {
			residu++
		}
	}
	t.Logf("%s : %d vies libres du drapeau — %d nees a un socle, %d au dernier point d'un "+
		"porteur qui vient de finir, %d ni l'un ni l'autre (%.1f %% tenues)",
		id, m.viesN, socle, porteur, m.viesN-m.viesOK, 100*objPart(m.viesOK, m.viesN))
	// DIAGNOSTIC NON BLOQUANT (cf. objDrapeauResidu) : il NOMME le residu, il ne l'absout pas.
	t.Logf("%s : DIAGNOSTIC — sur les %d non expliquees, %d naissent LA OU L'OBJET REPOSAIT "+
		"DEJA (re-creation d'un drapeau au sol) ; %d restent sans explication",
		id, m.viesN-m.viesOK, residu, m.viesN-m.viesOK-residu)
	m.temOK, m.temN = objDrapeauTemoin(refs, d, step)
	t.Logf("%s : TEMOIN — %d creations ti=42 d'armes ordinaires, %d tenues (%.1f %%)",
		id, m.temN, m.temOK, 100*objPart(m.temOK, m.temN))
	return m
}

// objDrapeauRefs rassemble les points de reference du film : les socles `flag_spawn` du
// catalogue de carte, et LES PORTAGES publies — chacun avec la piste de son porteur.
//
// LA POSITION DU PORTEUR EST LUE SUR SA PISTE, PAS SUR LE SPAN `dropped` — et c'est ce qui rend
// le controle non circulaire. Le span `dropped` est precisement ce que la phase 2 REPOSITIONNE
// sur la piste libre : le confronter a la piste libre reviendrait a comparer une grandeur a
// elle-meme.
//
// LE PORTAGE EST UN INTERVALLE, PAS UN INSTANT — CORRECTION DE L'INSTRUMENT DU 2026-08-18, ET
// ELLE NE TOUCHE AUCUN SEUIL. La premiere ecriture ne retenait que la DERNIERE frame de chaque
// portage : elle mesurait donc « l'objet renait-il la ou un portage s'est acheve ? », ce qui rate
// par construction le LACHER VOLONTAIRE — un lacher que le film ne date PAR AUCUN EVENEMENT, qui
// survient AU MILIEU d'un portage publie, et qui est precisement ce que ce lot existe pour
// dater. Ainsi ecrit, le controle demandait a la mesure de retrouver un phenomene dont il avait
// exclu la moitie des occurrences. La regle du plan dit « la derniere position du porteur qui
// vient de finir » : un porteur qui lache VIENT DE FINIR de porter, meme si le film ne le dit
// pas. La reference est donc sa position A L'INSTANT DE LA NAISSANCE, pour tout portage qui
// couvre cet instant ou vient de s'achever.
//
// LE RAYON (1,5 m), LA FENETRE (1 s) ET LES DEUX SEUILS (90 % / 20 %) SONT INCHANGES, et le
// TEMOIN passe par la MEME correction : si elle etait trop genereuse, les armes ordinaires en
// profiteraient autant que le drapeau. C'est a cela que le temoin sert.
func objDrapeauRefs(t *testing.T, id string, d objDoc, step uint64) []objDrapeauRef {
	t.Helper()
	var out []objDrapeauRef
	for _, s := range objDrapeauSocles(t, id) {
		out = append(out, objDrapeauRef{x: float32(s.Center.X), y: float32(s.Center.Y)})
	}
	idx := tracksByXUID(d.doc.Tracks)
	for _, f := range d.doc.FlagCarries {
		for _, s := range f.Spans {
			if !flagStateCarrying(s.State) || s.XUID == nil {
				continue
			}
			out = append(out, objDrapeauRef{
				porteur: idx[*s.XUID],
				t0US:    d.originUS + uint64(s.T0)*step,
				t1US:    d.originUS + uint64(s.T1)*step,
			})
		}
	}
	return out
}

// objDrapeauPres dit si un point NAIT a un socle, et/ou aux pieds d'un porteur qui portait
// encore. Les deux reponses sont rendues separement : leur somme ne se lit pas comme leur
// disjonction, et le rapport publie les trois.
func objDrapeauPres(refs []objDrapeauRef, x, y float32, atUS uint64, step uint64,
	originUS uint64) (socle, porteur bool) {
	for _, r := range refs {
		if len(r.porteur) == 0 {
			if math.Hypot(float64(x-r.x), float64(y-r.y)) < originDropMaxDist {
				socle = true
			}
			continue
		}
		// Le portage doit COUVRIR l'instant, ou venir de s'achever dans la fenetre de lacher.
		if atUS+objDrapeauLacherFenetreUS < r.t0US || atUS > r.t1US+objDrapeauLacherFenetreUS {
			continue
		}
		if atUS < originUS {
			continue // la creation precede la premiere frame publiee : rien a comparer
		}
		p, ok := pointOfXUIDAt(r.porteur, int((atUS-originUS)/step))
		if !ok {
			continue
		}
		if math.Hypot(float64(x-p.X), float64(y-p.Y)) < originDropMaxDist {
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
func objDrapeauTemoin(refs []objDrapeauRef, d objDoc, step uint64) (ok, n int) {
	known := loadoutFamilies()
	for _, c := range d.gw.Creations {
		w, res := gwPadsIdentity(c)
		if !res || !known[w] {
			continue
		}
		n++
		s, p := objDrapeauPres(refs, c.X, c.Y, c.TimestampUS, step, d.originUS)
		ok += objBool(s || p)
	}
	return ok, n
}

// objBool rend 1 pour vrai, 0 pour faux — un compteur se lit mieux qu'un `if`.
func objBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

// objDrapeauSocles rend TOUS les points `flag_spawn` de la carte, LE SOCLE NEUTRE COMPRIS.
//
// POURQUOI PAS `objFlagSpawns` — SECONDE CORRECTION DE L'INSTRUMENT DU 2026-08-18, ET ELLE NE
// TOUCHE AUCUN SEUIL NON PLUS. Ce helper-la sert la REGLE DE PRODUCTION du calque des portages,
// qui ECARTE le socle neutre a bon droit : en CTF ordinaire il n'y a pas de drapeau neutre, et
// en publier un ferait un troisieme drapeau immobile pour l'eternite. Mais le controle ne
// publie rien : il demande « cette naissance est-elle a un `flag_spawn` de la carte ? », et
// `flag_spawn` designe le ROLE du catalogue — c'est le vocabulaire de l'acquis que le plan cite
// (« 41/16/18 a 0,0 m d'un `flag_spawn` », mesures faites sur TOUS les points du role). Reutiliser
// le filtre de production revenait a poser au controle une question que le plan ne pose pas.
func objDrapeauSocles(t *testing.T, id string) []PointObjective {
	t.Helper()
	repo, module := os.Getenv(objRepoEnv), objCTFModules[id]
	if repo == "" || module == "" {
		return nil
	}
	cat, err := LoadMapObjectives(filepath.Join(repo, "data", "titles", "halo_infinite",
		"reference", "map_objectives.json"))
	if err != nil {
		t.Logf("%s : catalogue d'objectifs illisible (%v) — controle sans socles", id, err)
		return nil
	}
	for _, e := range cat.Maps {
		if e.Module == module {
			return e.PointsOfRole(mapvar.RoleFlagSpawn)
		}
	}
	t.Logf("%s : module %q hors du catalogue d'objectifs — controle sans socles", id, module)
	return nil
}

// objDrapeauResidu MESURE CE QUE LE CONTROLE N'EXPLIQUE PAS, sans le compter comme tenu.
//
// NON BLOQUANT, ET C'EST DELIBERE : le verdict du controle 3 reste celui du plan (socle OU
// porteur, >= 90 %). Cette troisieme lecture ne s'y ajoute pas — elle NOMME le residu. La
// question qu'elle pose est la seule qui reste : une naissance qui n'est ni au socle ni aux
// pieds d'un porteur tombe-t-elle LA OU L'OBJET REPOSAIT DEJA ? Si oui, ce sont des
// re-creations d'un drapeau au sol (l'objet est detruit puis recree au meme endroit), et non
// des objets etrangers ; si non, la piste porte autre chose que le drapeau, et il faudra le
// dire.
//
// LA COMPARAISON SE FAIT AU DERNIER POINT DES VIES PRECEDENTES, dans le meme rayon et la meme
// fenetre que le reste du controle — aucun seuil neuf.
func objDrapeauResidu(vies []flagFreeLife, i int) bool {
	x, y := vies[i].First()
	for j := 0; j < i; j++ {
		if vies[i].T0US < vies[j].T1US ||
			vies[i].T0US-vies[j].T1US > objDrapeauLacherFenetreUS {
			continue
		}
		px, py := vies[j].Last()
		if math.Hypot(float64(x-px), float64(y-py)) < originDropMaxDist {
			return true
		}
	}
	return false
}
