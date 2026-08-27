package replay

// oddball_fragmentation_d10_test.go — D10 / O1 : POURQUOI UN LONG PORTAGE SE FRAGMENTE.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER
// (`.ai/V7.5/replay2d/registre_film/D10_PROTOCOLE.md`, commit O0). Ce qui suit l'applique.
//
// # CE QUE CETTE CAMPAGNE N'EST PAS
//
// Elle ne re-mesure PAS le gate historique (ce serait O2, un second lot) et ne touche a
// AUCUN parametre de la chaine D9 : fenetre 8000 ms, rayon 1,5 m, ambiguite 1,0 m, cloture
// a la mort — tout est FIGE. Elle DECOMPOSE l'ecart deja etabli par D9 : le plus gros
// porteur API de chaque film ne recoit qu'environ la moitie de son temps. Ou passent ses
// secondes ? Le protocole nomme quatre causes D'AVANCE — (a) vol d'attribution par un
// tiers a portee, (b) meme joueur re-compte (fragmentation sans vol), (c) trou sans
// traversee, (d) autre — et ce fichier les compte.
//
// # L'AUTO-CONTROLE EST LA CONDITION DE VALIDITE
//
// La boucle ci-dessous REPRODUIT `d9Reconstruit` (memes primitives, memes constantes) en
// publiant le detail par trou. Si ses totaux par joueur s'ecartaient de la sortie de
// `d9Reconstruit` inchangee, la mesure du film serait INVALIDE — c'est un Errorf, pas un
// avertissement.
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM` + `ODDBALL_ORACLE` (oracle fige
// `D10_oracle_api_portage.json`), UN FILM PAR PROCESSUS, lecture seule, AUCUNE base.

import (
	"math"
	"os"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// d10Trou est UN trou rejoue par la chaine D9 figee, avec le detail que D9 ne publiait pas.
type d10Trou struct {
	// viePrec est l'INDEX de la vie libre qui precede ce trou (le trou separe la vie
	// viePrec de la vie viePrec+1). Il est stocke parce que la liste des trous SAUTE les
	// paires qui se chevauchent : son indice propre ne correspond plus a celui des vies —
	// le piege documente de D8.
	viePrec        int
	debutUS, finUS uint64
	// restX, restY : la derniere position emise par la vie precedente — le lieu de repos.
	restX, restY float32
	// xuid est l'attribution de la chaine (0 : aucune) ; atUS l'instant de la traversee
	// retenue ; finPortageUS la cloture (fin du trou ou mort du porteur).
	xuid               uint64
	atUS, finPortageUS uint64
	// retour : non attribue ET la vie suivante nait au socle.
	retour bool
}

// dureeAttribueeS rend les secondes que la chaine credite au porteur de ce trou.
func (tr d10Trou) dureeAttribueeS() float64 {
	if tr.xuid == 0 || tr.finPortageUS <= tr.atUS {
		return 0
	}
	return float64(tr.finPortageUS-tr.atUS) / 1e6
}

// TestOddballFragmentationD10 — LA MESURE O1. Un film par processus.
func TestOddballFragmentationD10(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", d4FilmEnv)
	}
	oracle, ok := d6Oracle(t)
	if !ok {
		return
	}
	g := filmproc.Arm("d10-fragmentation", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	e, ok := d8Charge(t, root, id)
	if !ok {
		return
	}
	deaths, err := ScanFilmDeaths(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}

	trous := d10Rejoue(e, deaths)
	ref, _ := d9Reconstruit(e, deaths, d9Options{fenetreMS: d9FenetreMS})
	d10AutoControle(t, id, trous, ref)
	d10PublieVies(t, id, e, trous)
	d10Ventile(t, id, e, trous, oracle)
	d10Distribution(t, id, e, trous)
}

// d10Rejoue reproduit la boucle de `d9Reconstruit` — memes primitives, memes constantes —
// en conservant le detail par trou. Toute divergence avec l'original est un defaut de CE
// fichier, et l'auto-controle la fait rougir.
func d10Rejoue(e d8Etat, deaths []Death) []d10Trou {
	var out []d10Trou
	for i := 0; i+1 < len(e.vies); i++ {
		fin := e.vies[i+1].T0US
		if fin <= e.vies[i].T1US {
			continue
		}
		x, y := e.vies[i].Last()
		tr := d10Trou{viePrec: i, debutUS: e.vies[i].T1US, finUS: fin, restX: x, restY: y}
		xuid, at, ok := d9PremierTraversant(e, tr.debutUS, fin, x, y, d9FenetreMS)
		if !ok {
			tr.retour = d6NaitAuSocle(e.vies[i+1], e.socles)
		} else {
			tr.xuid, tr.atUS = xuid, at
			tr.finPortageUS = d9FinPortage(xuid, at, fin, deaths, e.pont)
		}
		out = append(out, tr)
	}
	return out
}

// d10AutoControle confronte les totaux par joueur du rejeu D10 a la sortie de
// `d9Reconstruit` INCHANGEE. Un ecart au-dela du centieme de seconde invalide la mesure.
func d10AutoControle(t *testing.T, id string, trous []d10Trou, ref map[string]float64) {
	t.Helper()
	got := map[string]float64{}
	for _, tr := range trous {
		if d := tr.dureeAttribueeS(); d > 0 {
			got[strconv.FormatUint(tr.xuid, 10)] += d
		}
	}
	ecarts := 0
	for x, v := range ref {
		if math.Abs(got[x]-v) > 0.01 {
			ecarts++
			t.Errorf("AUTO-CONTROLE %s : joueur %s — D10 %.2f s contre d9Reconstruit %.2f s",
				id, x, got[x], v)
		}
	}
	for x := range got {
		if _, vu := ref[x]; !vu {
			ecarts++
			t.Errorf("AUTO-CONTROLE %s : joueur %s attribue par D10 mais pas par d9Reconstruit", id, x)
		}
	}
	if ecarts == 0 {
		t.Logf("AUTO-CONTROLE %s : OK — totaux par joueur identiques a d9Reconstruit "+
			"(%d joueur(s))", id, len(ref))
	}
}

// d10TrouApresVie rend le trou qui COMMENCE quand la vie j se tait, s'il existe.
func d10TrouApresVie(trous []d10Trou, j int) (d10Trou, bool) {
	for _, tr := range trous {
		if tr.viePrec == j {
			return tr, true
		}
	}
	return d10Trou{}, false
}

// d10Interieure applique la definition du protocole (§4) : les deux trous voisins existent
// et sont ATTRIBUES, et la vie ne nait pas au socle.
func d10Interieure(e d8Etat, trous []d10Trou, j int) bool {
	prec, okP := d10TrouApresVie(trous, j-1)
	suiv, okS := d10TrouApresVie(trous, j)
	return okP && okS && prec.xuid != 0 && suiv.xuid != 0 && !d6NaitAuSocle(e.vies[j], e.socles)
}

// d10DistJoueur rend la distance d'UN joueur nomme a un point, a un instant — la piste du
// joueur, tolerance d'echantillon `d6EcartMaxMS`, meme regle qu'ailleurs dans la chaine.
func d10DistJoueur(e d8Etat, xuid uint64, atUS uint64, x, y float32) float64 {
	best := math.MaxFloat64
	for slot, trk := range e.tracks {
		if e.pont.SlotXUID[slot] != xuid {
			continue
		}
		p, ecart := trk.at(atUS)
		if ecart > d6EcartMaxMS*1000 {
			continue
		}
		if d := math.Hypot(float64(p.X)-float64(x), float64(p.Y)-float64(y)); d < best {
			best = d
		}
	}
	return best
}

// d10PublieVies publie UNE ligne par vie libre : duree, naissance, porteur precedent,
// ramasseur, distance du porteur precedent a la re-prise, meme-joueur, interieure.
func d10PublieVies(t *testing.T, id string, e d8Etat, trous []d10Trou) {
	t.Helper()
	for j, vie := range e.vies {
		nx, ny := vie.First()
		prec, okP := d10TrouApresVie(trous, j-1)
		suiv, okS := d10TrouApresVie(trous, j)
		porteurPrec, ramasseur := uint64(0), uint64(0)
		if okP {
			porteurPrec = prec.xuid
		}
		if okS {
			ramasseur = suiv.xuid
		}
		dist := "-"
		if porteurPrec != 0 && okS && suiv.xuid != 0 {
			if d := d10DistJoueur(e, porteurPrec, suiv.atUS, suiv.restX, suiv.restY); d != math.MaxFloat64 {
				dist = strconv.FormatFloat(d, 'f', 2, 64) + " m"
			}
		}
		meme := "NON"
		if porteurPrec != 0 && ramasseur == porteurPrec {
			meme = "OUI"
		}
		marque := ""
		if d10Interieure(e, trous, j) {
			marque = " [INTERIEURE]"
		}
		t.Logf("VIE %2d %s : duree %6.2f s · naissance (%.1f, %.1f) · porteur precedent %s · "+
			"ramasseur %s · dist prec->re-prise %s · meme-joueur %s%s",
			j, id, float64(vie.T1US-vie.T0US)/1e6, nx, ny,
			d10Nom(porteurPrec), d10Nom(ramasseur), dist, meme, marque)
	}
}

// d10Nom rend les 4 derniers chiffres d'un xuid, ou « aucun ».
func d10Nom(xuid uint64) string {
	if xuid == 0 {
		return "aucun"
	}
	s := strconv.FormatUint(xuid, 10)
	if len(s) > 4 {
		s = s[len(s)-4:]
	}
	return s
}

// d10Ventile applique la ventilation du protocole (§4) aux secondes manquantes du plus
// gros porteur API, et publie le CONSTAT du seuil d'ouverture de O2 pour ce film.
func d10Ventile(t *testing.T, id string, e d8Etat, trous []d10Trou, oracle map[string]float64) {
	t.Helper()
	gros := d6Max(oracle)
	if gros == "" {
		t.Logf("NON EXPLOITABLE %s : oracle vide", id)
		return
	}
	p, err := strconv.ParseUint(gros, 10, 64)
	if err != nil {
		t.Fatalf("%s : xuid oracle illisible %q : %v", id, gros, err)
	}
	var rec float64
	for _, tr := range trous {
		if tr.xuid == p {
			rec += tr.dureeAttribueeS()
		}
	}
	manque := oracle[gros] - rec
	if manque < 0 {
		manque = 0
	}
	a, b, c := d10Causes(e, trous, p)
	d := manque - a - b - c
	if d < 0 {
		d = 0
	}
	pct := func(v float64) float64 {
		if manque <= 0 {
			return 0
		}
		return 100 * v / manque
	}
	t.Logf("VENTILATION %s : plus gros porteur API %s — API %.1f s, reconstruit %.1f s, "+
		"MANQUANT %.1f s", id, d10Nom(p), oracle[gros], rec, manque)
	t.Logf("VENTILATION %s :   (a) vol par un tiers a <= %.1f m : %.1f s (%.1f %%)",
		id, d6RayonRamassageM, a, pct(a))
	t.Logf("VENTILATION %s :   (b) meme joueur re-compte      : %.1f s (%.1f %%)", id, b, pct(b))
	t.Logf("VENTILATION %s :   (c) trou sans traversee        : %.1f s (%.1f %%)", id, c, pct(c))
	t.Logf("VENTILATION %s :   (d) autre / hors intervalle    : %.1f s (%.1f %%)", id, d, pct(d))
	if somme := a + b + c; somme > manque {
		t.Logf("VENTILATION %s : les causes (a)+(b)+(c) = %.1f s DEPASSENT le manquant de "+
			"%.1f s — l'excedent se dit, il ne se lisse pas (protocole §4)", id, somme, somme-manque)
	}
	seuil := pct(a+b) >= 50
	t.Logf("CONSTAT %s : (a)+(b) = %.1f %% des secondes manquantes — seuil d'ouverture de "+
		"O2 (>= 50 %%) %s sur ce film", id, pct(a+b), d8Oui(seuil))
}

// d10Causes compte les secondes des causes (a), (b), (c) pour le porteur P — definitions
// du protocole, recopiees : seuls les trous dont le PORTEUR PRECEDENT est P versent.
func d10Causes(e d8Etat, trous []d10Trou, p uint64) (a, b, c float64) {
	for _, tr := range trous {
		prec, okP := d10TrouApresVie(trous, tr.viePrec-1)
		if !okP || prec.xuid != p {
			continue
		}
		switch {
		case tr.xuid != 0 && tr.xuid != p:
			if d10DistJoueur(e, p, tr.atUS, tr.restX, tr.restY) <= d6RayonRamassageM {
				a += tr.dureeAttribueeS()
			}
		case tr.xuid == p:
			vie := e.vies[tr.viePrec]
			b += float64(vie.T1US-vie.T0US)/1e6 + float64(tr.atUS-tr.debutUS)/1e6
		case tr.xuid == 0 && !tr.retour:
			c += float64(tr.finUS-tr.debutUS) / 1e6
		}
	}
	return a, b, c
}

// d10Distribution publie la distribution des durees des vies libres INTERIEURES, et les
// durees brutes dans une ligne PARSABLE — l'agregation de corpus (et le N = q90 du plan)
// se calcule sur l'ensemble des films admis, hors de ce processus.
func d10Distribution(t *testing.T, id string, e d8Etat, trous []d10Trou) {
	t.Helper()
	var durees []float64
	for j := range e.vies {
		if d10Interieure(e, trous, j) {
			durees = append(durees, float64(e.vies[j].T1US-e.vies[j].T0US)/1e6)
		}
	}
	d8Quantiles(t, id, "duree des vies libres INTERIEURES (s)", durees)
	sort.Float64s(durees)
	ligne := "D10_INTERIEURES " + id + " :"
	for _, d := range durees {
		ligne += " " + strconv.FormatFloat(d, 'f', 2, 64)
	}
	t.Log(ligne)
}
