//go:build research

package main

// verdict_oracle_research_test.go — LE VERDICT DE L'ETAPE 2 : ce que l'algorithme retrouve
// des positions que les guides pro decrivent, et ce qu'il colore a tort.
//
// # LA REGLE, ECRITE AVANT LA MESURE (plan, item 2.1 ; reserves du pilote, oracle §7)
//
//	RETROUVEE   une position calculee RETROUVE une zone de l'oracle si son polygone couvre
//	            >= 30 % de l'aire de la zone, OU si son barycentre tombe dans la zone.
//	RAPPEL      zones `forte` retrouvees / zones `forte`.
//	PRECISION   positions calculees qui retrouvent au moins une zone de l'oracle (`forte` ou
//	            `faible`) / positions calculees.
//	EXCLUES     les lignes de l'oracle a `zone_en = ?` (regle de lecture de sa section 3).
//	A PART      les zones `forte` dont la raison est « arme » SEULE : emplacements d'arme,
//	            pas forcement des lieux TENABLES (oracle §7). Rappel donne avec et sans.
//	TEMOIN      permutation circulaire des zones DANS CHAQUE CARTE : la position attendue en
//	            Z_i est reputee en Z_i+1 de la liste triee des zones nommees de la carte.
//	            Sans effondrement du rappel et de la precision, un GO ne vaut rien (item 2.4).
//	GO          >= 4 cartes a rappel >= 0,7 ET precision >= 0,6, ZERO contre-exemple colore,
//	            Live Fire exclue des preuves (contamination mesuree a l'etape 1), et un 0,5
//	            de rappel lu « indetermine » sur les cartes a moins de 3 zones `forte`.
//
// # CE QUE CE FICHIER NE FAIT PAS
//
// Il ne touche a AUCUN seuil du calcul. Le score, le plancher, la taille minimale d'une
// composante et le rayon du disque sont figes (`powerpos.ReglageV1`, MESURE_EMPIRIQUE du
// 2026-09-20) et le verdict porte sur eux tels quels. Un mauvais verdict est un RESULTAT, et
// l'issue prevue par le plan est alors l'etape 2bis (voie geometrique), jamais un reglage
// retouche apres coup.
//
// # INVOCATION
//
//	MAPPOWER_DATA_ROOT=<racine contenant data/> \
//	  go test -tags research ./cmd/mappower-build/ -run Verdict -v
//
// Sans la variable : SKIP (le depot de travail n'a pas de `data/`).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// seuilRecouvrement est la part de la zone de l'oracle qu'une position doit couvrir pour la
// retrouver par la surface (l'autre voie est le barycentre). Valeur du plan, item 2.1.
const seuilRecouvrement = 0.30

// Seuils du verdict, ecrits dans le plan avant toute mesure.
const (
	seuilRappel    = 0.70
	seuilPrecision = 0.60
	cartesExigees  = 4
	// fortesPourTrancher : en dessous de ce nombre de zones `forte`, le rappel est a gros
	// grain et un 0,5 se lit « indetermine » (relecture du pilote, oracle §7).
	fortesPourTrancher = 3
)

// carteContaminee est la carte dont les positions de kill sont decalees entre variantes
// (9,88 m, decouverte n°1 de l'etape 1) : ses chiffres sont rapportes, jamais comptes comme
// preuve, dans un sens comme dans l'autre.
const carteContaminee = "live fire"

// zoneAttendue est une zone de l'oracle, resolue (ou non) dans le vocabulaire du depot.
type zoneAttendue struct {
	NomEN     string
	Confiance string
	Raison    string
	ArmeSeule bool

	Resolue  bool
	surfaces []surface
	points   [][2]float64
	AireM2   float64

	Retrouvee bool
	Par       string
}

// positionJugee est une position calculee, avec son rattachement au vocabulaire.
type positionJugee struct {
	ID         string
	ScoreMoyen float64
	AireM2     float64
	forme      surface
	centre     [2]float64

	ZoneDominante   string
	PartDominante   float64
	AireDominanteM2 float64

	Appariee  bool
	ApparieeA string
}

// bilanCarte est le resultat d'une carte pour un jeu d'attentes (reel ou temoin).
type bilanCarte struct {
	Carte      string
	CarteCle   string
	NbFortes   int
	NbResolues int

	Rappel          float64
	RappelHorsArme  float64
	NbHorsArme      int
	Precision       float64
	FauxPositifs    []string
	FauxNegatifs    []string
	FortesNonResolu []string

	Positions []positionJugee
	Attendues []zoneAttendue

	ContreExemplesPurs    []string
	ContreExemplesPartage []string

	// Permutation : pour le TEMOIN seulement, le decalage applique aux zones `forte`
	// (« Z_i -> Z_i+1 »). Sans cette colonne, un temoin qui remonte se lit comme un bug
	// alors que c'est la mesure elle-meme — une carte dont les zones voisines se valent.
	Permutation []string
}

// TestVerdictOracle prononce le verdict de l'etape 2 et ECRIT le document de verdict.
func TestVerdictOracle(t *testing.T) {
	racineDonnees := cheminDonnees()
	if racineDonnees == "" {
		t.Skipf("%s absent : le verdict a besoin de data/ (depot principal)", verdictDataRootEnv)
	}
	racine := racineDepot(t)
	oracle, pieges := litOracle(t, filepath.Join(racine, verdictDossier, verdictOracleNom))
	doc := litPositions(t, filepath.Join(racine, verdictDossier, verdictMesures, "positions.json"))

	res := title.NewPathResolver(racineDonnees)
	cat, err := replay.LoadMapCallouts(res.MapCalloutsPath(verdictTitleSlug))
	if err != nil {
		t.Fatalf("catalogue de zones nommees illisible : %v", err)
	}

	reels := map[string]bilanCarte{}
	temoins := map[string]bilanCarte{}
	var diags []diagnosticZone
	var ordre []string
	for _, c := range doc.Cartes {
		cle, lignes := attentesDe(oracle, c)
		if cle == "" {
			continue
		}
		zones := zonesDeCarte(cat, c)
		if len(zones) == 0 {
			t.Logf("carte %s (%s) : aucune zone nommee au depot — hors verdict", c.Carte, cle)
			continue
		}
		_, piegesDeCarte := attentesDe(pieges, c)
		e := entreeCarte{Carte: c, Cle: cle, Zones: zones, Lignes: lignes, Pieges: piegesDeCarte}

		reels[c.Carte] = evalue(e, nil)
		temoins[c.Carte] = evalue(e, nomsTries(zones))
		diags = append(diags, diagnostique(
			filepath.Join(racine, verdictDossier, verdictMesures), e, reels[c.Carte])...)
		ordre = append(ordre, c.Carte)
	}
	if len(reels) == 0 {
		t.Fatal("aucune carte mesuree ne correspond a une carte de l'oracle")
	}
	sort.Strings(ordre)

	sortie := filepath.Join(racine, verdictDossier, verdictSortieNom)
	rapport := rendVerdict(verdictRendu{Doc: doc, Ordre: ordre, Reels: reels,
		Temoins: temoins, Diagnostics: diags})
	if err := os.WriteFile(sortie, []byte(rapport), 0o644); err != nil {
		t.Fatalf("ecriture du verdict (%s) : %v", sortie, err)
	}
	t.Logf("verdict ecrit : %s", sortie)
	for _, nom := range ordre {
		b := reels[nom]
		t.Logf("%-10s cle=%-40s fortes=%d/%d rappel=%.2f (temoin %.2f) precision=%.2f (temoin %.2f) positions=%d pieges_purs=%d",
			nom, b.CarteCle, b.NbResolues, b.NbFortes, b.Rappel, temoins[nom].Rappel,
			b.Precision, temoins[nom].Precision, len(b.Positions), len(b.ContreExemplesPurs))
	}
}

// attentesDe rend la cle de l'oracle qui designe la carte mesuree, et ses lignes.
func attentesDe(lignes []ligneOracle, c SortieCartePos) (string, []ligneOracle) {
	cles := map[string]bool{}
	for _, k := range clesDe(c) {
		cles[k] = true
	}
	cle := ""
	var out []ligneOracle
	for _, l := range lignes {
		if !cles[l.CarteCle] {
			continue
		}
		cle = l.CarteCle
		out = append(out, l)
	}
	return cle, out
}

// entreeCarte rassemble tout ce qu'une carte apporte au verdict (une struct et non six
// parametres — seuil du depot).
type entreeCarte struct {
	Carte  SortieCartePos
	Cle    string
	Zones  []replay.CalloutZone
	Lignes []ligneOracle
	Pieges []ligneOracle
}

// evalue calcule le bilan d'une carte. `permutation` non nil active le TEMOIN NEGATIF : la
// zone attendue de chaque ligne devient la SUIVANTE de la liste triee des zones de la carte.
func evalue(e entreeCarte, permutation []string) bilanCarte {
	b := bilanCarte{Carte: e.Carte.Carte, CarteCle: e.Cle}
	index := indexeZones(e.Zones)
	var remplace func(string) string
	if permutation != nil {
		remplace = func(nom string) string { return suivante(permutation, nom) }
	}
	b.Attendues = resoutAttentes(e.Lignes, index, remplace, &b)
	b.Positions = jugePositions(e.Carte, index)
	apparie(&b)
	compteContreExemples(&b, e.Pieges, index, e.Lignes)
	return b
}

// resoutAttentes transforme les lignes de l'oracle en zones attendues geolocalisees.
// `remplace` non nil active un TEMOIN NEGATIF : la zone attendue de chaque ligne devient
// celle que la fonction rend (v1 : la suivante de la liste triee ; v2 : la plus eloignee).
// Une fonction qui rend "" rend la ligne non exploitable.
func resoutAttentes(lignes []ligneOracle, index map[string]*zoneIndexee,
	remplace func(string) string, b *bilanCarte) []zoneAttendue {
	vues := map[string]bool{}
	var out []zoneAttendue
	for _, l := range lignes {
		nom := l.ZoneEN
		if nom == zoneInconnue || nom == "" {
			continue // regle de lecture de l'oracle : ligne non exploitable
		}
		if remplace != nil {
			nom = remplace(nom)
			if nom == "" {
				continue
			}
			if l.Confiance == confianceForte {
				b.Permutation = append(b.Permutation, l.ZoneEN+" -> "+nom)
			}
		}
		cle := l.Confiance + "|" + strings.ToLower(nom)
		if vues[cle] {
			continue
		}
		vues[cle] = true
		z := zoneAttendue{NomEN: nom, Confiance: l.Confiance, Raison: l.Raison,
			ArmeSeule: l.armeSeule()}
		if e := index[strings.ToLower(nom)]; e != nil && len(e.points) > 0 {
			z.Resolue, z.surfaces, z.points, z.AireM2 = true, e.surfaces, e.points, e.aireM2
		}
		if z.Confiance == confianceForte {
			b.NbFortes++
			if z.Resolue {
				b.NbResolues++
			} else {
				b.FortesNonResolu = append(b.FortesNonResolu, nom)
			}
		}
		out = append(out, z)
	}
	return out
}

// jugePositions echantillonne chaque position calculee et lui attribue sa zone nommee
// DOMINANTE par recouvrement (part de la position couverte par la zone).
func jugePositions(c SortieCartePos, index map[string]*zoneIndexee) []positionJugee {
	var out []positionJugee
	for _, p := range c.Positions {
		s := nouvelleSurface(p.Polygone)
		pts := s.echantillons()
		pj := positionJugee{ID: p.ID, ScoreMoyen: p.ScoreMoyen, AireM2: aireEchantillonnee(pts),
			forme: s, centre: [2]float64{p.CentreX, p.CentreY}}
		pj.ZoneDominante = "aucune"
		for nom, e := range index {
			part := partDansZone(pts, e)
			if part > pj.PartDominante {
				pj.ZoneDominante, pj.PartDominante, pj.AireDominanteM2 = nom, part, e.aireM2
			}
		}
		if pj.PartDominante == 0 {
			pj.ZoneDominante, pj.AireDominanteM2 = "aucune", 0
		}
		out = append(out, pj)
	}
	return out
}

// partDansZone rend la part des echantillons d'une position qui tombent dans la zone.
func partDansZone(pts [][2]float64, e *zoneIndexee) float64 {
	if len(pts) == 0 {
		return 0
	}
	dedans := 0
	for _, p := range pts {
		if e.contient(p[0], p[1]) {
			dedans++
		}
	}
	return float64(dedans) / float64(len(pts))
}

// apparie croise positions et attentes selon la regle du plan, puis calcule rappel et
// precision.
func apparie(b *bilanCarte) {
	for i := range b.Attendues {
		z := &b.Attendues[i]
		if !z.Resolue {
			continue
		}
		for j := range b.Positions {
			p := &b.Positions[j]
			if !retrouve(*p, *z) {
				continue
			}
			z.Retrouvee, z.Par = true, p.ID
			p.Appariee, p.ApparieeA = true, z.NomEN
			break
		}
	}
	// Une position peut retrouver une zone deja retrouvee par une autre : la boucle
	// ci-dessus s'arrete a la premiere, celle-ci complete l'appariement des positions.
	for j := range b.Positions {
		p := &b.Positions[j]
		if p.Appariee {
			continue
		}
		for _, z := range b.Attendues {
			if z.Resolue && retrouve(*p, z) {
				p.Appariee, p.ApparieeA = true, z.NomEN
				break
			}
		}
		if !p.Appariee {
			b.FauxPositifs = append(b.FauxPositifs, fmt.Sprintf("%s (zone dominante : %s)",
				p.ID, p.ZoneDominante))
		}
	}
	b.Rappel, b.FauxNegatifs = mesureRappel(b.Attendues, false)
	b.RappelHorsArme, _ = mesureRappel(b.Attendues, true)
	b.NbHorsArme = compteFortes(b.Attendues, true)
	apparies := 0
	for _, p := range b.Positions {
		if p.Appariee {
			apparies++
		}
	}
	if len(b.Positions) > 0 {
		b.Precision = float64(apparies) / float64(len(b.Positions))
	}
}

// retrouve applique la regle du plan : >= 30 % de la zone couverte, OU barycentre dedans.
func retrouve(p positionJugee, z zoneAttendue) bool {
	if partCouverte(z.points, p.forme) >= seuilRecouvrement {
		return true
	}
	for _, s := range z.surfaces {
		if s.contient(p.centre[0], p.centre[1]) {
			return true
		}
	}
	return false
}

// mesureRappel rend le rappel sur les zones `forte` RESOLUES, et les manquees.
func mesureRappel(attendues []zoneAttendue, horsArme bool) (float64, []string) {
	return mesureRappelSelon(attendues, func(z zoneAttendue) bool { return !(horsArme && z.ArmeSeule) })
}

// mesureRappelSelon rend le rappel sur les zones `forte` RESOLUES que `garde` retient, et
// les manquees. C'est la forme generale : la v1 en tire « hors arme seule », la v2 les
// groupes « hauteur / lignes de vue » et « arme / objectif ».
func mesureRappelSelon(attendues []zoneAttendue, garde func(zoneAttendue) bool) (float64, []string) {
	total, trouvees := 0, 0
	var manquees []string
	for _, z := range attendues {
		if z.Confiance != confianceForte || !z.Resolue || !garde(z) {
			continue
		}
		total++
		if z.Retrouvee {
			trouvees++
		} else {
			manquees = append(manquees, fmt.Sprintf("%s (%s)", z.NomEN, z.Raison))
		}
	}
	if total == 0 {
		return 0, manquees
	}
	return float64(trouvees) / float64(total), manquees
}

// compteFortes compte les zones `forte` resolues.
func compteFortes(attendues []zoneAttendue, horsArme bool) int {
	return compteFortesSelon(attendues, func(z zoneAttendue) bool { return !(horsArme && z.ArmeSeule) })
}

// compteFortesSelon compte les zones `forte` resolues que `garde` retient.
func compteFortesSelon(attendues []zoneAttendue, garde func(zoneAttendue) bool) int {
	n := 0
	for _, z := range attendues {
		if z.Confiance == confianceForte && z.Resolue && garde(z) {
			n++
		}
	}
	return n
}

// compteContreExemples marque les positions dont le BARYCENTRE tombe dans une zone que les
// guides deconseillent (oracle §3.1).
//
// DEUX COMPTES, ET LA DIFFERENCE EST DANS L'ORACLE LUI-MEME. Sa section 4 (« desaccords
// entre sources ») nomme des zones qui sont a la fois position et piege dans le MEME article
// : Streets `Main Street`, Bazaar `Market`, Recharge `Pit` (« on y va pour l'arme, on n'y
// reste pas »). Les compter contre l'algorithme reviendrait a lui demander de trancher un
// desaccord que l'oracle n'a pas tranche. Le compte qui engage le verdict est donc celui des
// pieges PURS — ceux que l'oracle ne decrit nulle part comme une position.
func compteContreExemples(b *bilanCarte, pieges []ligneOracle,
	index map[string]*zoneIndexee, positifs []ligneOracle) {
	aussiPositive := map[string]bool{}
	for _, l := range positifs {
		if l.ZoneEN != zoneInconnue && l.ZoneEN != "" {
			aussiPositive[strings.ToLower(l.ZoneEN)] = true
		}
	}
	for _, l := range pieges {
		if l.ZoneEN == zoneInconnue || l.ZoneEN == "" {
			continue
		}
		e := index[strings.ToLower(l.ZoneEN)]
		if e == nil {
			continue
		}
		for _, p := range b.Positions {
			if !e.contient(p.centre[0], p.centre[1]) {
				continue
			}
			marque := fmt.Sprintf("%s dans %s", p.ID, l.ZoneEN)
			if aussiPositive[strings.ToLower(l.ZoneEN)] {
				b.ContreExemplesPartage = append(b.ContreExemplesPartage, marque)
			} else {
				b.ContreExemplesPurs = append(b.ContreExemplesPurs, marque)
			}
		}
	}
	sort.Strings(b.ContreExemplesPurs)
	sort.Strings(b.ContreExemplesPartage)
}
