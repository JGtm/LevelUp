package replay

// ctf_retour_zone_research_test.go — INSTRUMENT DE RECHERCHE : LA ZONE DE RETOUR DU DRAPEAU.
//
// # LA QUESTION, ET POURQUOI ELLE SE POSE MAINTENANT
//
// Le jeu ne laisse PAS un drapeau au sol indefiniment : une zone entoure le drapeau tombe, les
// coequipiers du camp proprietaire qui s'y tiennent VIDENT sa jauge de retour plus vite, et sans
// personne la jauge se vide seule jusqu'a ce que le drapeau rentre. Le calque du drapeau vivant
// ne modelise rien de cela (`flag_carries_lives.go`, section « LE RETOUR AUTOMATIQUE N'EST PAS
// SIMULE ») : un `dropped` court jusqu'a la reprise, le `flag_returns` ou la fin du match, et le
// corpus porte des laches de 100 a 160 secondes qui n'ont jamais existe a l'ecran.
//
// La tentative anterieure a cherche le delai par un PROXY — l'ecart entre une fin de portage sans
// reprise et la prise suivante AU SOCLE — et l'a trouve trop disperse pour trancher. Le present
// instrument ne mesure pas la meme chose : il lit l'OBJET. Une vie libre du drapeau qui NAIT A UN
// SOCLE est le drapeau qui rentre, datee a la frame — c'est une observation, pas une inference.
//
// # LES DEUX CHAINES, ET POURQUOI LEUR ACCORD EST LA PREUVE
//
//	CREDITEE   `flag_returns` du statborg : un joueur a ete credite du retour. Elle date les
//	           retours PROVOQUES, jamais les retours par expiration (personne n'est credite).
//	OBJET      une creation `ti=42` du drapeau A MOINS DE [originDropMaxDist] d'un socle de la
//	           carte. Elle date TOUS les retours, provoques comme expires.
//
// Les deux lectures sont DISJOINTES (compteurs de statistique d'un cote, records de creation du
// monde de l'autre). Si elles tombent a la meme frame sur les retours credites, la chaine OBJET
// est licite pour dater ceux que le credit ne voit pas. Sinon elle ne l'est pas, et l'instrument
// le dit.
//
// # CE QU'IL MESURE ENSUITE
//
//	RAYON       pour chaque rayon candidat, la duree passee DANS la zone avant le retour. Si la
//	            zone existe et que la jauge se vide a taux constant, cette duree est CONSTANTE a
//	            occupation egale : le bon rayon est celui qui MINIMISE la dispersion.
//	LOI         la duree de sejour selon le NOMBRE de defenseurs presents — c'est l'hypothese du
//	            user (« plus on est, plus ca retourne vite »), a confirmer ou a refuter.
//	EXPIRATION  la duree des episodes ou AUCUN defenseur n'entre jamais : la minuterie nue.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne modifie ni le decodeur ni l'assemblage : il rejoue `objDocumentDe` et compte a cote.
// Garde d'environnement `OBJ_FILM` / `OBJ_REPO` comme toute la phase 0 — il se saute en CI.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache OBJ_REPO=<depot> \
//	  go test ./internal/analysis/replay/ -run CTFZoneRetour -v -timeout 90m

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// ctfzRayons — les rayons candidats, en metres monde. La borne haute (15 m) est volontairement
// au-dela du plausible : un rayon qui gagnerait la-haut dirait que la zone n'explique rien.
var ctfzRayons = []float32{0.5, 1, 1.3, 1.5, 2, 3, 5, 8, 15}

// ctfzRayonJeu — LE RAYON QUE LE JEU DECLARE, releve dans le pool de constantes de son propre
// script : `parcel_deliver_object.lua`, table CONFIG, `innerAreaMonitorRadius = 1.3`. Il entre
// ici comme HYPOTHESE A CONFRONTER, pas comme verite : c'est la mesure qui juge.
const ctfzRayonJeu float32 = 1.3

const (
	// ctfzTolAccordFrames — tolerance d'accord entre les deux chaines de retour, en frames.
	// Une seconde : les deux horloges sont calees a la frame pres, et une creation d'objet suit
	// le compteur de statistique d'un ou deux pas de replication.
	ctfzTolAccordFrames = 10
	// ctfzPeremptionFrames — au-dela de deux secondes sans echantillon, la position d'un joueur
	// n'est plus tenue pour connue. Compter un absent comme present gonflerait l'occupation.
	ctfzPeremptionFrames = 20
)

// ctfzPose est une position du drapeau au sol, valable a partir d'une frame et jusqu'a la
// suivante. Un drapeau tombe roule : sa position bouge pendant le lacher.
type ctfzPose struct {
	t0   int
	x, y float32
}

// ctfzEpisode est UN drapeau au sol, de son lacher a ce qui y met fin.
type ctfzEpisode struct {
	film   string
	flag   int
	team   int
	t0, t1 int        // bornes du lacher (spans `dropped` contigus fusionnes), en frames
	poses  []ctfzPose // la position du drapeau, pas a pas
	suite  string     // l'etat du span SUIVANT : ce qui a mis fin au lacher
	// credit / objet : la frame du retour selon chacune des deux chaines, -1 si muette.
	credit, objet int
	// occ[r][i] = nombre de defenseurs dans le rayon ctfzRayons[r] a la frame t0+i.
	occ [][]int
}

// retour rend la frame du retour QUI A CLOS CET EPISODE — la premiere des deux chaines qui parle,
// et SEULEMENT si elle parle a l'interieur du lacher (a deux frames pres de sa fin).
//
// LE FILTRE EST LA CORRECTION D'UNE PREMIERE VERSION QUI COMPTAIT FAUX : une naissance de l'objet
// au socle survenue LONGTEMPS APRES un lacher termine par une REPRISE etait prise pour le retour
// de ce lacher-la, et donnait des durees sans rapport.
func (e ctfzEpisode) retour() int {
	best := -1
	for _, f := range []int{e.credit, e.objet} {
		if f < e.t0 || f > e.t1+2 {
			continue
		}
		if best < 0 || f < best {
			best = f
		}
	}
	return best
}

// ctfzFilm porte ce qu'un film rend : ses episodes de lacher, et les DEUX chaines brutes.
type ctfzFilm struct {
	id string
	// socles dit si la carte est au catalogue d'objectifs. FAUX : la chaine OBJET ne peut PAS
	// parler (aucun socle a reconnaitre), et le film ne compte dans aucune mesure qui la
	// suppose — l'y compter mesurerait le catalogue de cartes, pas le drapeau.
	socles     bool
	eps        []ctfzEpisode
	credits    []int
	naissances []ctfzNaissance
}

// TestCTFZoneRetourRecherche — la mesure, sur les trois films CTF du corpus de la phase 0.
func TestCTFZoneRetourRecherche(t *testing.T) {
	root := objRequireRoot(t)
	cat := goldenCatalog(t)
	var films []ctfzFilm
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		films = append(films, ctfzEpisodesDuFilm(t, root, id, src, cat))
	}
	if len(films) == 0 {
		t.Skipf("aucun film CTF dans le cache (%s)", objFilmEnv)
	}
	var eps []ctfzEpisode
	for _, f := range films {
		if f.socles {
			eps = append(eps, f.eps...)
		}
	}
	ctfzRapportEpisodes(t, films)
	ctfzRapportAccord(t, films)
	ctfzRapportDistances(t, eps)
	ctfzRapportRayon(t, eps)
	ctfzRapportExpiration(t, eps)
}

// ctfzEpisodesDuFilm rend les episodes de lacher d'UN film, occupation comprise.
func ctfzEpisodesDuFilm(t *testing.T, root, id string, src *objDiskFilm, cat LabelCatalog) ctfzFilm {
	t.Helper()
	b := objBridgeOf(t, root, id)
	d := objDocumentDe(t, root, id, b, src)
	step := uint64(d.doc.FrameIntervalMS) * 1000
	if step == 0 {
		t.Fatalf("%s : axe de temps sans echelle", id)
	}
	spawns := objFlagSpawns(t, id)
	naissances := ctfzNaissancesAuSocle(flagFreeLives(d.gw, cat.ObjectiveObjects), spawns, d.originUS, step)
	pistes := ctfzPistesParEquipe(d.doc, ctfzEquipes(id))
	credits := ctfzRetoursCredites(src, b.OffsetMS, d.originUS, step)
	out := ctfzEpisodesDuDocument(d.doc, id, naissances, credits, pistes)
	t.Logf("%s : %d episodes de lacher, %d naissances de l'objet a un socle, %d retours credites, "+
		"%d socles d'equipe", id, len(out), len(naissances), len(credits), len(spawns))
	return ctfzFilm{id: id, socles: len(spawns) > 0, eps: out, credits: credits, naissances: naissances}
}

// ctfzRetoursCredites rend les frames des `flag_returns` du STATBORG.
//
// POURQUOI ON NE LES LIT PAS SUR LES SPANS `home` DU DOCUMENT, ET C'EST LA CORRECTION D'UNE
// CIRCULARITE : depuis que la production ramene aussi le drapeau sur la NAISSANCE DE L'OBJET, un
// `home` ne dit plus laquelle des deux chaines l'a produit. Mesurer leur accord sur les spans
// reviendrait a confronter la chaine objet a elle-meme. On repart donc des evenements bruts.
func ctfzRetoursCredites(src *objDiskFilm, offsetMS int64, originUS, step uint64) []int {
	var out []int
	evs := objectiveevents.NamedEventsFrom(objectiveevents.StatRecords(src), objectiveevents.ObjectiveTypeFlag)
	for _, e := range evs {
		if e.Stat != objectiveevents.StatFlagReturns {
			continue
		}
		filmUS := (int64(e.TimeMS) + offsetMS) * 1000
		if filmUS < int64(originUS) {
			continue
		}
		out = append(out, int((uint64(filmUS)-originUS)/step))
	}
	sort.Ints(out)
	return out
}

// ctfzEpisodesDuDocument parcourt les spans `dropped` publies et les instrumente.
//
// LES SPANS `dropped` CONTIGUS SONT UN SEUL LACHER. Le calque en ouvre un nouveau des que la
// POSITION change (le drapeau roule, ou une vie libre le repositionne) : les compter separement
// tronquerait la duree du lacher, qui est precisement ce qu'on mesure. La position, elle, reste
// suivie pas a pas — c'est `poses`.
func ctfzEpisodesDuDocument(doc ReplayDocument, id string, naissances []ctfzNaissance,
	credits []int, pistes map[int]map[string][]Point) []ctfzEpisode {
	var out []ctfzEpisode
	for f, fc := range doc.FlagCarries {
		cour := -1 // indice de l'episode en cours pour ce drapeau, -1 : aucun
		for i, s := range fc.Spans {
			if s.State != FlagStateDropped {
				cour = -1
				continue
			}
			if cour >= 0 && out[cour].t1+1 == s.T0 {
				out[cour].t1 = s.T1
				out[cour].poses = append(out[cour].poses, ctfzPose{t0: s.T0, x: s.X, y: s.Y})
			} else {
				out = append(out, ctfzEpisode{film: id, flag: f, team: fc.Team, t0: s.T0, t1: s.T1,
					credit: -1, objet: -1, poses: []ctfzPose{{t0: s.T0, x: s.X, y: s.Y}}})
				cour = len(out) - 1
			}
			out[cour].suite = "FIN_AXE"
			if i+1 < len(fc.Spans) {
				out[cour].suite = fc.Spans[i+1].State
			}
		}
	}
	for i := range out {
		out[i].objet = ctfzPremiereNaissance(naissances, out[i].flag, out[i].t0)
		out[i].credit = ctfzCreditDans(credits, out[i].t0, out[i].t1+2)
		out[i].occ = ctfzOccupation(out[i], pistes[out[i].team])
	}
	return out
}

// ctfzCreditDans rend le premier retour credite tombant dans une fenetre, ou -1.
func ctfzCreditDans(credits []int, t0, t1 int) int {
	for _, f := range credits {
		if f >= t0 && f <= t1 {
			return f
		}
	}
	return -1
}

// ctfzEquipes rend l'equipe de chaque joueur du film, depuis les lignes de match GELEES du
// corpus de la phase 0 (aucune base ouverte).
func ctfzEquipes(id string) map[string]int {
	out := map[string]int{}
	for _, p := range objCorpus[id].Players {
		out[p.XUID] = p.Team
	}
	return out
}

// ctfzPistesParEquipe regroupe les points publies par EQUIPE puis par joueur, tries.
func ctfzPistesParEquipe(doc ReplayDocument, teams map[string]int) map[int]map[string][]Point {
	out := map[int]map[string][]Point{}
	for _, tr := range doc.Tracks {
		tm, ok := teams[tr.XUID]
		if !ok {
			continue
		}
		if out[tm] == nil {
			out[tm] = map[string][]Point{}
		}
		out[tm][tr.XUID] = append(out[tm][tr.XUID], tr.Points...)
	}
	for _, par := range out {
		for x := range par {
			pts := par[x]
			sort.Slice(pts, func(i, j int) bool { return pts[i].T < pts[j].T })
			par[x] = pts
		}
	}
	return out
}

// ctfzNaissance est une creation de l'objet drapeau A UN SOCLE : le drapeau rentre.
type ctfzNaissance struct {
	flag  int // l'indice du socle, donc du drapeau
	frame int
}

// ctfzNaissancesAuSocle rend, triees, les naissances de vies libres a moins de
// [originDropMaxDist] d'un socle d'equipe — avec l'indice du socle.
func ctfzNaissancesAuSocle(vies []flagFreeLife, spawns []FlagSpawn, originUS, step uint64) []ctfzNaissance {
	var out []ctfzNaissance
	for _, l := range vies {
		x, y := l.First()
		for f, s := range spawns {
			if sqDist(s.X, s.Y, x, y) > originDropMaxDist*originDropMaxDist {
				continue
			}
			if l.T0US < originUS {
				break
			}
			out = append(out, ctfzNaissance{flag: f, frame: int((l.T0US - originUS) / step)})
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].frame < out[j].frame })
	return out
}

// ctfzPremiereNaissance rend la frame de la premiere naissance au socle du drapeau `flag`
// STRICTEMENT apres `apres`, ou -1.
func ctfzPremiereNaissance(ns []ctfzNaissance, flag, apres int) int {
	for _, n := range ns {
		if n.flag == flag && n.frame > apres {
			return n.frame
		}
	}
	return -1
}

// ctfzOccupation compte, rayon par rayon et frame par frame, les DEFENSEURS dans la zone.
func ctfzOccupation(e ctfzEpisode, pistes map[string][]Point) [][]int {
	n := e.t1 - e.t0 + 1
	occ := make([][]int, len(ctfzRayons))
	for r := range occ {
		occ[r] = make([]int, n)
	}
	for _, pts := range pistes {
		for i := 0; i < n; i++ {
			x, y, ok := ctfzPosA(pts, e.t0+i)
			if !ok {
				continue
			}
			fx, fy := e.poseA(e.t0 + i)
			d := float32(math.Sqrt(sqDist(x, y, fx, fy)))
			for r, ray := range ctfzRayons {
				if d <= ray {
					occ[r][i]++
				}
			}
		}
	}
	return occ
}

// poseA rend la position du drapeau a une frame : la derniere pose commencee.
func (e ctfzEpisode) poseA(f int) (float32, float32) {
	x, y := e.poses[0].x, e.poses[0].y
	for _, p := range e.poses {
		if p.t0 > f {
			break
		}
		x, y = p.x, p.y
	}
	return x, y
}

// ctfzPosA rend la position d'un joueur a une frame, par le dernier echantillon non PERIME.
func ctfzPosA(pts []Point, f int) (float32, float32, bool) {
	i := sort.Search(len(pts), func(i int) bool { return pts[i].T > f }) - 1
	if i < 0 || f-pts[i].T > ctfzPeremptionFrames {
		return 0, 0, false
	}
	return pts[i].X, pts[i].Y, true
}

// ctfzRapportEpisodes imprime la table brute — un episode par ligne.
func ctfzRapportEpisodes(t *testing.T, films []ctfzFilm) {
	t.Helper()
	n := 0
	for _, f := range films {
		n += len(f.eps)
	}
	t.Logf("EPISODES DE LACHER (%d) — film flag equipe [t0..t1] duree suite credit objet occ@1,3m", n)
	for _, f := range films {
		for _, e := range f.eps {
			t.Logf("  %s f%d eq%d [%d..%d] %.1fs %s credit=%s objet=%s occ=%d",
				e.film, e.flag, e.team, e.t0, e.t1, float64(e.t1-e.t0+1)/10, e.suite,
				ctfzFrameStr(e.credit), ctfzFrameStr(e.objet), ctfzOccMax(e, ctfzRayonJeu))
		}
	}
}

// ctfzFrameStr rend une frame ou un tiret quand la chaine est muette.
func ctfzFrameStr(f int) string {
	if f < 0 {
		return "-"
	}
	return fmt.Sprintf("%d", f)
}

// ctfzIndiceRayon rend l'indice du rayon demande dans [ctfzRayons], -1 s'il n'y est pas.
func ctfzIndiceRayon(ray float32) int {
	for i, r := range ctfzRayons {
		if r == ray {
			return i
		}
	}
	return -1
}

// ctfzOccMax rend l'occupation MAXIMALE d'un episode a un rayon donne.
func ctfzOccMax(e ctfzEpisode, ray float32) int {
	r := ctfzIndiceRayon(ray)
	if r < 0 {
		return 0
	}
	m := 0
	for _, v := range e.occ[r] {
		if v > m {
			m = v
		}
	}
	return m
}

// ctfzRapportAccord confronte les deux chaines. C'est LE CONTROLE qui autorise (ou non) a dater
// les retours automatiques par la chaine OBJET.
//
// IL SE COMPTE PAR EVENEMENT CREDITE, ET NON PAR EPISODE — la premiere version comptait faux.
// `flag_returns` NE NOMME PAS SON DRAPEAU (c'est la raison meme pour laquelle la production
// s'abstient quand deux drapeaux sont au sol) : rapporte aux episodes, un meme evenement se
// retrouve attribue AUX DEUX drapeaux tombes, et la moitie de ces attributions est fausse par
// construction. La question licite est donc « chaque retour credite a-t-il une naissance d'objet
// a moins d'une seconde ? », sans decider lequel des deux drapeaux il concerne.
//
// LES FILMS SANS SOCLE SONT HORS DENOMINATEUR, et ce n'est pas un tri de complaisance : sur une
// carte absente du catalogue d'objectifs, la chaine OBJET n'a AUCUN socle a reconnaitre et se
// tait par construction. L'y compter mesurerait la couverture du catalogue de cartes, pas
// l'accord des deux lectures.
func ctfzRapportAccord(t *testing.T, films []ctfzFilm) {
	t.Helper()
	credits, accords := 0, 0
	var ecarts []int
	for _, f := range films {
		if !f.socles {
			t.Logf("ACCORD — %s HORS DENOMINATEUR : carte sans socle au catalogue, la chaine "+
				"objet ne peut pas parler (%d retours credites ignores)", f.id, len(f.credits))
			continue
		}
		for _, c := range ctfzInstantsUniques(f.credits) {
			credits++
			d, ok := ctfzNaissanceLaPlusProche(f.naissances, c)
			if !ok {
				continue
			}
			ecarts = append(ecarts, d)
			if d < 0 {
				d = -d
			}
			if d <= ctfzTolAccordFrames {
				accords++
			}
		}
	}
	sort.Ints(ecarts)
	t.Logf("ACCORD DES DEUX CHAINES — %d retours credites DISTINCTS sur cartes a socles ; %d ont "+
		"une naissance d'objet a moins de %d frames (%.1f %%). Ecarts objet-credit : %s",
		credits, accords, ctfzTolAccordFrames, 100*objPart(accords, credits), ctfzQuantilesStr(ecarts))
}

// ctfzInstantsUniques rend les instants credites SANS doublon : deux emissions a la meme frame
// sont un seul retour.
func ctfzInstantsUniques(credits []int) []int {
	var out []int
	for i, c := range credits {
		if i > 0 && c == credits[i-1] {
			continue
		}
		out = append(out, c)
	}
	return out
}

// ctfzNaissanceLaPlusProche rend l'ecart signe a la naissance d'objet la plus proche d'un instant.
func ctfzNaissanceLaPlusProche(ns []ctfzNaissance, at int) (int, bool) {
	best, found := 0, false
	for _, n := range ns {
		d := n.frame - at
		if !found || abs(d) < abs(best) {
			best, found = d, true
		}
	}
	return best, found
}

// abs rend la valeur absolue d'un entier.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ctfzResetSecondes — la minuterie NUE, telle que le volet EXPIRATION la mesure (30 s au
// premier tour, sur trois episodes ou aucun defenseur n'entre jamais). Elle entre ici comme
// CONSTANTE DU MODELE : sans elle, la part de jauge que le temps seul consomme resterait
// inconnue et l'ajustement du taux par joueur n'aurait pas de denominateur.
const ctfzResetSecondes = 30.0

// ctfzRapportRayon AJUSTE LE MODELE DU JEU, rayon par rayon.
//
// LE MODELE VIENT DU JEU, PAS D'UNE INTUITION : son script nomme `CalculateReturnRateHarmonic`,
// donc la jauge se remplit au taux `1/Treset + b * H(n)`, ou `n` est le nombre de defenseurs dans
// la zone et `H` la serie harmonique (1 ; 1+1/2 ; 1+1/2+1/3 …). Un retour survient quand la
// jauge atteint 1 :
//
//	D/Treset + b * S = 1,  avec S = somme sur les frames de H(n) * 0,1 s
//
// `b` est donc CALCULABLE episode par episode — et il doit etre le MEME partout. Le bon rayon est
// celui qui rend `b` le plus constant (coefficient de variation minimal) ; `1/b` est le temps de
// retour d'un defenseur SEUL.
func ctfzRapportRayon(t *testing.T, eps []ctfzEpisode) {
	t.Helper()
	t.Logf("RAYON — ajustement de b dans D/%.0f + b*S = 1 (S = somme des H(n) dt). "+
		"Le meilleur rayon MINIMISE le cv de b ; 1/b est le retour a un seul defenseur.",
		ctfzResetSecondes)
	for r, ray := range ctfzRayons {
		var bs []float64
		sansOcc := 0
		for _, e := range eps {
			ret := e.retour()
			if ret < 0 {
				continue
			}
			s := ctfzIntegraleHarmonique(e, r, ret)
			if s <= 0 {
				sansOcc++
				continue
			}
			d := float64(ret-e.t0) / 10
			bs = append(bs, (1-d/ctfzResetSecondes)/s)
		}
		m, ec := ctfzMoyEcart(bs)
		inv := 0.0
		if m > 0 {
			inv = 1 / m
		}
		t.Logf("  r=%4.1f m | n=%2d b=%.4f ec=%.4f cv=%5.2f -> retour seul %5.2fs | "+
			"retours sans aucun defenseur dans la zone : %d", ray, len(bs), m, ec,
			ctfzCV(m, ec), inv, sansOcc)
	}
}

// ctfzIntegraleHarmonique rend S — la somme, sur les frames du lacher jusqu'au retour, de H(n)
// multiplie par le pas de temps. C'est ce que le modele du jeu appelle la contribution des
// joueurs a la jauge.
func ctfzIntegraleHarmonique(e ctfzEpisode, r, ret int) float64 {
	fin := ret - 1
	if fin > e.t1 {
		fin = e.t1
	}
	var s float64
	for i := 0; i <= fin-e.t0 && i < len(e.occ[r]); i++ {
		s += ctfzHarmonique(e.occ[r][i]) / 10
	}
	return s
}

// ctfzHarmonique rend H(n) = 1 + 1/2 + ... + 1/n, et 0 pour n <= 0.
func ctfzHarmonique(n int) float64 {
	var h float64
	for i := 1; i <= n; i++ {
		h += 1 / float64(i)
	}
	return h
}

// ctfzRapportDistances rend, sur les episodes RETOURNES, la distance du defenseur le PLUS PROCHE
// a l'instant du retour : le retour exige d'etre dans la zone, donc le maximum de cette
// distribution est une BORNE INFERIEURE du rayon.
func ctfzRapportDistances(t *testing.T, eps []ctfzEpisode) {
	t.Helper()
	var d []float64
	for _, e := range eps {
		ret := e.retour()
		if ret < 0 {
			continue
		}
		i := ret - 1 - e.t0
		if i < 0 || i >= len(e.occ[0]) {
			continue
		}
		d = append(d, ctfzPlusProche(e, i))
	}
	sort.Float64s(d)
	if len(d) == 0 {
		t.Logf("DISTANCE AU RETOUR — aucun episode retourne")
		return
	}
	t.Logf("DISTANCE AU RETOUR — %d episodes ; defenseur le plus proche : min=%.2f m med=%.2f m "+
		"p90=%.2f m max=%.2f m (borne INFERIEURE du rayon)", len(d), d[0], d[len(d)/2],
		d[(len(d)*9)/10], d[len(d)-1])
}

// ctfzPlusProche rend la plus petite distance a laquelle un defenseur se trouve, en lisant les
// paliers d'occupation deja calcules : le premier rayon occupe borne la distance.
func ctfzPlusProche(e ctfzEpisode, i int) float64 {
	for r, ray := range ctfzRayons {
		if e.occ[r][i] > 0 {
			return float64(ray)
		}
	}
	return math.Inf(1)
}

// ctfzRapportExpiration rend la duree des episodes ou AUCUN defenseur n'est jamais entre dans la
// zone la plus large : la minuterie nue, si elle existe.
func ctfzRapportExpiration(t *testing.T, eps []ctfzEpisode) {
	t.Helper()
	for _, ray := range []float32{ctfzRayonJeu, 3, 15} {
		var durees []float64
		for _, e := range eps {
			ret := e.retour()
			if ret < 0 || ctfzOccMax(e, ray) > 0 {
				continue
			}
			durees = append(durees, float64(ret-e.t0)/10)
		}
		sort.Float64s(durees)
		m, s := ctfzMoyEcart(durees)
		t.Logf("EXPIRATION — %d episodes retournes SANS qu'aucun defenseur n'entre a %.1f m : "+
			"moy=%.2fs ecart=%.2fs ; valeurs %v", len(durees), ray, m, s, ctfzArrondi(durees))
	}
}

// ctfzMoyEcart rend la moyenne et l'ecart-type d'un echantillon.
func ctfzMoyEcart(v []float64) (float64, float64) {
	if len(v) == 0 {
		return 0, 0
	}
	var s float64
	for _, x := range v {
		s += x
	}
	m := s / float64(len(v))
	var q float64
	for _, x := range v {
		q += (x - m) * (x - m)
	}
	return m, math.Sqrt(q / float64(len(v)))
}

// ctfzCV rend le coefficient de variation, 0 quand la moyenne est nulle.
func ctfzCV(m, s float64) float64 {
	if m == 0 {
		return 0
	}
	return s / m
}

// ctfzArrondi arrondit un echantillon au centieme, pour la lisibilite du journal.
func ctfzArrondi(v []float64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = math.Round(x*100) / 100
	}
	return out
}

// ctfzQuantilesStr rend min / median / max d'un echantillon trie.
func ctfzQuantilesStr(v []int) string {
	if len(v) == 0 {
		return "(aucun)"
	}
	return fmt.Sprintf("min=%d med=%d max=%d", v[0], v[len(v)/2], v[len(v)-1])
}
