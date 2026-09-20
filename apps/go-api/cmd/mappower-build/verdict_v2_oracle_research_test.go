//go:build research

package main

// verdict_v2_oracle_research_test.go — LE VERDICT V2 (item 2bis.D, premiere moitie,
// 2026-09-20) : ce qu'un fichier de positions retrouve des positions fortes de l'ORACLE V2,
// et ce qu'il colore a tort.
//
// # LES REGLES, ECRITES AVANT LA MESURE (plan D11 ; relecture du pilote, oracle v2 §10)
//
//	RETROUVEE   une position RETROUVE une zone si son polygone couvre >= 30 % de l'aire de la
//	            zone, OU si son barycentre tombe dans la zone (regle v1, inchangee).
//	RAPPEL      zones `forte` retrouvees / zones `forte` RESOLUES au vocabulaire du depot.
//	PRECISION   positions qui TOUCHENT une zone `forte` / positions calculees. « Toucher » est
//	            la meme relation que « retrouver » (30 % de la zone ou barycentre dedans) :
//	            une seule relation pour les deux mesures. Les zones `faible` NE COMPTENT PLUS
//	            (decouverte 9 du plan : elles pavent les cartes).
//	A PART      le rappel sur les fortes dont la raison porte « hauteur » ou « lignes de vue »,
//	            et celui sur les fortes de raison « arme » / « objectif » SANS hauteur ni vue
//	            (oracle v2 §10 : les positions attestees sont presque toutes du premier groupe,
//	            c'est le signal que la voie geometrique mesure directement).
//	EXCLUES     les lignes de l'oracle a `zone_en = ?` (regle de lecture de sa section 6).
//	PIEGES      une position dont le barycentre tombe dans une zone de la section 7 ; piege
//	            PUR si l'oracle ne decrit la zone nulle part comme une position, PARTAGE
//	            sinon (Long Hall, Market, Pit : les deux a la fois). Le GO porte sur les PURS.
//	TEMOIN      GEOGRAPHIQUE : chaque zone attendue est remplacee par la zone nommee de la
//	            carte dont le centroide est LE PLUS ELOIGNE. Rappel et precision doivent
//	            s'effondrer, sinon l'appariement ne mesure que la densite des zones.
//	CALIBRAGE   Recharge, Aquarius, Streets : les cartes sur lesquelles le reglage v2 a ete
//	            choisi. Rapportees, JAMAIS comptees au verdict.
//	VALIDATION  Live Fire, Bazaar, Forbidden, Empyrean, Solitude — et toute autre carte
//	            mesuree ou l'oracle v2 porte une forte resolue ; sans forte : « hors oracle ».
//	INDETERMINE sur une carte a 2 fortes, un rappel de 0,50 se lit « indetermine ».
//	GO          >= 4 cartes de VALIDATION a rappel >= 0,7 ET precision >= 0,6, ZERO piege PUR
//	            colore sur les cartes de validation.
//
// # CE QUE CE FICHIER NE FAIT PAS
//
// Il ne touche a AUCUN reglage (`ReglageV1`, `ReglageV2`) ni a aucune formule : il JUGE. Le
// fichier de positions est un parametre, pour juger de la meme facon l'empirique v2, la
// geometrie (`positions_geo.json`) et leur fusion, a regles egales.
//
// # INVOCATION
//
//	MAPPOWER_DATA_ROOT=<racine contenant data/> \
//	  [MAPPOWER_POSITIONS=<fichier positions.json, defaut mesures_v2_2026-09-20/positions_v2.json>] \
//	  [MAPPOWER_VERDICT_SORTIE=<nom du document, defaut VERDICT_V2_2026-09-20.md>] \
//	  go test -tags research ./cmd/mappower-build/ -run VerdictV2 -v
//
// Sans MAPPOWER_DATA_ROOT : SKIP (le depot de travail n'a pas de `data/`).

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// Chemins du verdict v2.
const (
	verdictV2PositionsEnv = "MAPPOWER_POSITIONS"
	verdictV2SortieEnv    = "MAPPOWER_VERDICT_SORTIE"
	verdictV2Mesures      = "mesures_v2_2026-09-20"
	verdictV2PositionsNom = "positions_v2.json"
	verdictV2OracleNom    = "ORACLE_PRO_V2_2026-09-20.md"
	verdictV2SortieNom    = "VERDICT_V2_2026-09-20.md"
	// fortesIndeterminees : nombre de fortes d'une carte ou un rappel de 0,50 se lit
	// « indetermine » (oracle v2 §10 : les cartes a 2 fortes).
	fortesIndeterminees = 2
	// seuilCouloirCellules : au-dela, une position est un « couloir » a juger (le
	// document de mesure v2 en signale deux, de 85 et 88 cellules).
	seuilCouloirCellules = 60
)

// marqueursOracleV2 : les sections de l'oracle v2 (`ORACLE_PRO_V2_2026-09-20.md`).
var marqueursOracleV2 = marqueursOracle{
	DebutPositions: "## 6. TABLE FINALE", DebutPieges: "## 7. Table des contre-exemples", Fin: "## 8.",
}

// Roles des cartes au verdict v2.
const (
	roleCalibrage  = "calibrage"
	roleValidation = "validation"
	roleHorsOracle = "hors oracle"
)

// cartesCalibrage : le reglage v2 a ete choisi sur leurs distributions (2bis.B).
var cartesCalibrage = map[string]bool{"recharge": true, "aquarius": true, "streets": true}

// cartesValidation : nommees par le pilote (oracle v2 §10) ; toute autre carte a forte
// resolue s'y ajoute d'elle-meme.
var cartesValidation = map[string]bool{
	"live fire": true, "bazaar": true, "forbidden": true, "empyrean": true, "solitude": true,
}

// bilanV2 est le bilan v1 augmente de ce que le verdict v2 rapporte en plus.
type bilanV2 struct {
	bilanCarte
	Role string

	// PrecisionForte : positions qui touchent une forte / positions.
	PrecisionForte float64
	NbToucheForte  int

	RappelHauteurVue      float64
	NbHauteurVue          int
	FauxNegatifsHauteur   []string
	RappelArmeObjectif    float64
	NbArmeObjectif        int
	FauxNegatifsArmeObj   []string
	NbFortesAutreRaison   int
	FortesAutreRaisonNoms []string

	Couloirs []couloir
}

// couloir est une position assez grande pour qu'on se demande si c'est une salle.
type couloir struct {
	ID         string
	NbCellules int
	AireM2     float64
	// Zones : « zone : part de la position / part de la zone », par part decroissante.
	Zones   []string
	Lecture string
}

// TestVerdictV2 juge un fichier de positions contre l'oracle v2 et ECRIT le document.
func TestVerdictV2(t *testing.T) {
	racineDonnees := cheminDonnees()
	if racineDonnees == "" {
		t.Skipf("%s absent : le verdict a besoin de data/ (depot principal)", verdictDataRootEnv)
	}
	racine := racineDepot(t)
	dossier := filepath.Join(racine, verdictDossier)
	oracle, pieges := litOracleEntre(t, filepath.Join(dossier, verdictV2OracleNom), marqueursOracleV2)
	pieges = eclateZones(pieges)
	t.Logf("oracle v2 : %d lignes de positions, %d contre-exemples (apres eclatement)", len(oracle), len(pieges))

	cheminPositions := cheminPositionsJuge(racine)
	doc := litPositions(t, cheminPositions)

	cat, err := replay.LoadMapCallouts(title.NewPathResolver(racineDonnees).MapCalloutsPath(verdictTitleSlug))
	if err != nil {
		t.Fatalf("catalogue de zones nommees illisible : %v", err)
	}
	src := sourcesV2{Oracle: oracle, Pieges: pieges, Catalogue: cat}
	rendu := verdictV2Rendu{CheminPositions: cheminPositions, Doc: doc,
		Lignee: ligneeDuFichier(cheminPositions), ReglageBrut: reglageBrut(cheminPositions)}
	rendu.Ordre, rendu.Reels, rendu.Temoins = jugeDocument(t, src, doc)
	rendu.Lignees = jugeLignees(t, racine, src)
	rendu.Diagnostics, rendu.Fidelites = diagnostiqueV2(t, filepath.Dir(cheminPositions), src, doc, rendu.Reels)
	if filepath.Base(cheminPositions) == positionsGeoNom {
		rendu.DiagnosticsGeo = diagnostiqueGeo(t, filepath.Dir(cheminPositions), src, doc, rendu.Reels)
	}
	peinsVerdicts(t, racineDonnees, doc, rendu.Reels)

	sortie := filepath.Join(dossier, nomSortie())
	if err := os.WriteFile(sortie, []byte(rendVerdictV2(rendu)), 0o644); err != nil {
		t.Fatalf("ecriture du verdict v2 (%s) : %v", sortie, err)
	}
	t.Logf("verdict v2 ecrit : %s", sortie)
	for _, nom := range rendu.Ordre {
		b, tm := rendu.Reels[nom], rendu.Temoins[nom]
		t.Logf("%-10s %-11s fortes=%d/%d rappel=%.2f (temoin %.2f) precision=%.2f (temoin %.2f) HV=%.2f(%d) AO=%.2f(%d) positions=%d pieges_purs=%d",
			nom, b.Role, b.NbResolues, b.NbFortes, b.Rappel, tm.Rappel, b.PrecisionForte, tm.PrecisionForte,
			b.RappelHauteurVue, b.NbHauteurVue, b.RappelArmeObjectif, b.NbArmeObjectif,
			len(b.Positions), len(b.ContreExemplesPurs))
	}
}

// sourcesV2 rassemble ce que le jugement lit (une struct, pas quatre parametres).
type sourcesV2 struct {
	Oracle    []ligneOracle
	Pieges    []ligneOracle
	Catalogue *replay.MapCalloutsCatalog
}

// jugeDocument juge toutes les cartes d'un fichier de positions, en reel et en temoin.
func jugeDocument(t *testing.T, src sourcesV2, doc SortiePositions) ([]string, map[string]bilanV2, map[string]bilanV2) {
	t.Helper()
	reels, temoins := map[string]bilanV2{}, map[string]bilanV2{}
	var ordre []string
	for _, c := range doc.Cartes {
		cle, lignes := attentesDe(src.Oracle, c)
		zones := zonesDeCarte(src.Catalogue, c)
		if cle == "" || len(zones) == 0 {
			t.Logf("carte %s : cle oracle %q, %d zones nommees — hors oracle", c.Carte, cle, len(zones))
			b := bilanV2{bilanCarte: bilanCarte{Carte: c.Carte, CarteCle: cle}, Role: roleHorsOracle}
			b.Positions = jugePositions(c, indexeZones(zones))
			reels[c.Carte], temoins[c.Carte] = b, b
			ordre = append(ordre, c.Carte)
			continue
		}
		_, piegesDeCarte := attentesDe(src.Pieges, c)
		e := entreeCarte{Carte: c, Cle: cle, Zones: zones, Lignes: lignes, Pieges: piegesDeCarte}
		reels[c.Carte] = evalueV2(e, false)
		temoins[c.Carte] = evalueV2(e, true)
		ordre = append(ordre, c.Carte)
	}
	if len(reels) == 0 {
		t.Fatal("aucune carte mesuree dans le fichier de positions")
	}
	sort.Strings(ordre)
	return ordre, reels, temoins
}

// cheminPositionsJuge rend le fichier de positions a juger (variable d'environnement,
// relative a la racine du depot, ou la mesure empirique v2 par defaut).
func cheminPositionsJuge(racine string) string {
	if v := strings.TrimSpace(os.Getenv(verdictV2PositionsEnv)); v != "" {
		if filepath.IsAbs(v) {
			return v
		}
		return filepath.Join(racine, v)
	}
	return filepath.Join(racine, verdictDossier, verdictV2Mesures, verdictV2PositionsNom)
}

// nomSortie rend le nom du document ecrit.
func nomSortie() string {
	if v := strings.TrimSpace(os.Getenv(verdictV2SortieEnv)); v != "" {
		return v
	}
	return verdictV2SortieNom
}

// eclateZones separe les contre-exemples qui nomment PLUSIEURS zones en une ligne
// (« East Tower / West Tower », « Blue Courtyard / Yellow Courtyard ») et ramene a `?` les
// lignes que l'oracle n'a pas su rattacher (« ? (aucune zone dans le depot) »).
func eclateZones(lignes []ligneOracle) []ligneOracle {
	var out []ligneOracle
	for _, l := range lignes {
		if strings.HasPrefix(l.ZoneEN, zoneInconnue) {
			l.ZoneEN = zoneInconnue
			out = append(out, l)
			continue
		}
		for _, nom := range strings.Split(l.ZoneEN, " / ") {
			m := l
			m.ZoneEN = strings.TrimSpace(nom)
			out = append(out, m)
		}
	}
	return out
}

// evalueV2 calcule le bilan v2 d'une carte. `temoin` active le remplacement geographique.
func evalueV2(e entreeCarte, temoin bool) bilanV2 {
	b := bilanV2{bilanCarte: bilanCarte{Carte: e.Carte.Carte, CarteCle: e.Cle}}
	index := indexeZones(e.Zones)
	var remplace func(string) string
	var distances map[string]float64
	if temoin {
		remplace, distances = remplacementLePlusLoin(index, affichagesDe(e.Zones))
	}
	b.Attendues = resoutAttentes(e.Lignes, index, remplace, &b.bilanCarte)
	if temoin {
		b.Permutation = annoteDistances(b.Permutation, distances)
	}
	b.Positions = jugePositions(e.Carte, index)
	apparieV2(&b)
	compteContreExemples(&b.bilanCarte, e.Pieges, index, e.Lignes)
	b.Couloirs = couloirsDe(e.Carte, index)
	b.Role = roleDe(b)
	return b
}

// roleDe classe la carte : calibrage, validation (nommee par le pilote ou porteuse d'une
// forte resolue), hors oracle sinon.
func roleDe(b bilanV2) string {
	switch {
	case cartesCalibrage[b.Carte]:
		return roleCalibrage
	case cartesValidation[b.Carte] || b.NbResolues > 0:
		return roleValidation
	default:
		return roleHorsOracle
	}
}

// affichagesDe rend le libelle EN d'origine de chaque zone, par cle minuscule.
func affichagesDe(zones []replay.CalloutZone) map[string]string {
	out := map[string]string{}
	for _, z := range zones {
		nom := strings.TrimSpace(z.EN)
		if nom != "" {
			out[strings.ToLower(nom)] = nom
		}
	}
	return out
}

// apparieV2 croise positions et attentes : rappel sur les fortes (total et par groupe de
// raison), precision sur les fortes seules.
func apparieV2(b *bilanV2) {
	for i := range b.Attendues {
		z := &b.Attendues[i]
		if !z.Resolue {
			continue
		}
		for _, p := range b.Positions {
			if retrouve(p, *z) {
				z.Retrouvee, z.Par = true, p.ID
				break
			}
		}
	}
	for j := range b.Positions {
		p := &b.Positions[j]
		for _, z := range b.Attendues {
			if z.Resolue && z.Confiance == confianceForte && retrouve(*p, z) {
				p.Appariee, p.ApparieeA = true, z.NomEN
				break
			}
		}
		if p.Appariee {
			b.NbToucheForte++
			continue
		}
		b.FauxPositifs = append(b.FauxPositifs, fmt.Sprintf("%s (zone dominante : %s, %.0f %%)",
			p.ID, p.ZoneDominante, p.PartDominante*100))
	}
	tout := func(zoneAttendue) bool { return true }
	b.Rappel, b.FauxNegatifs = mesureRappelSelon(b.Attendues, tout)
	b.RappelHauteurVue, b.FauxNegatifsHauteur = mesureRappelSelon(b.Attendues, estHauteurOuVue)
	b.NbHauteurVue = compteFortesSelon(b.Attendues, estHauteurOuVue)
	b.RappelArmeObjectif, b.FauxNegatifsArmeObj = mesureRappelSelon(b.Attendues, estArmeOuObjectifSeuls)
	b.NbArmeObjectif = compteFortesSelon(b.Attendues, estArmeOuObjectifSeuls)
	autre := func(z zoneAttendue) bool { return !estHauteurOuVue(z) && !estArmeOuObjectifSeuls(z) }
	b.NbFortesAutreRaison = compteFortesSelon(b.Attendues, autre)
	for _, z := range b.Attendues {
		if z.Confiance == confianceForte && z.Resolue && autre(z) {
			b.FortesAutreRaisonNoms = append(b.FortesAutreRaisonNoms, z.NomEN+" ("+z.Raison+")")
		}
	}
	if len(b.Positions) > 0 {
		b.PrecisionForte = float64(b.NbToucheForte) / float64(len(b.Positions))
	}
}

// estHauteurOuVue : la raison de l'oracle porte « hauteur » ou « lignes de vue ».
func estHauteurOuVue(z zoneAttendue) bool {
	r := strings.ToLower(z.Raison)
	return strings.Contains(r, "hauteur") || strings.Contains(r, "lignes de vue")
}

// estArmeOuObjectifSeuls : « arme » ou « objectif » SANS hauteur ni lignes de vue.
func estArmeOuObjectifSeuls(z zoneAttendue) bool {
	if estHauteurOuVue(z) {
		return false
	}
	r := strings.ToLower(z.Raison)
	return strings.Contains(r, "arme") || strings.Contains(r, "objectif")
}

// remplacementLePlusLoin rend la fonction du TEMOIN GEOGRAPHIQUE : a chaque zone, la zone
// nommee de la carte dont le centroide (des echantillons de l'union du libelle) est le
// plus eloigne. Deterministe : egalite tranchee par le nom. Rend "" pour une zone absente
// du catalogue (la ligne devient non exploitable, comme en reel). La distance entre les
// deux centroides est notee, par zone d'origine, dans la table rendue en second.
func remplacementLePlusLoin(index map[string]*zoneIndexee, affichage map[string]string) (func(string) string, map[string]float64) {
	noms := make([]string, 0, len(index))
	centres := map[string][2]float64{}
	for nom, e := range index {
		noms = append(noms, nom)
		centres[nom] = centroide(e.points)
	}
	sort.Strings(noms)
	distances := map[string]float64{}
	return func(nom string) string {
		cle := strings.ToLower(nom)
		c, ok := centres[cle]
		if !ok {
			return ""
		}
		meilleur, dmax := "", -1.0
		for _, autre := range noms {
			if autre == cle {
				continue
			}
			if d := math.Hypot(centres[autre][0]-c[0], centres[autre][1]-c[1]); d > dmax {
				meilleur, dmax = autre, d
			}
		}
		if meilleur == "" {
			return ""
		}
		distances[cle] = dmax
		return affichage[meilleur]
	}, distances
}

// annoteDistances complete chaque « Zone -> Remplacante » du temoin par la distance entre
// les deux centroides.
func annoteDistances(permutation []string, distances map[string]float64) []string {
	out := make([]string, 0, len(permutation))
	for _, p := range permutation {
		origine := strings.ToLower(strings.TrimSpace(strings.SplitN(p, " -> ", 2)[0]))
		out = append(out, fmt.Sprintf("%s (%.0f m)", p, distances[origine]))
	}
	return out
}

// centroide rend la moyenne d'un nuage de points (zero si vide).
func centroide(pts [][2]float64) [2]float64 {
	if len(pts) == 0 {
		return [2]float64{}
	}
	var sx, sy float64
	for _, p := range pts {
		sx, sy = sx+p[0], sy+p[1]
	}
	return [2]float64{sx / float64(len(pts)), sy / float64(len(pts))}
}

// couloirsDe juge les positions de plus de `seuilCouloirCellules` cellules : quelles zones
// nommees elles traversent, et si elles ressemblent a une SALLE plutot qu'a un lieu tenu.
func couloirsDe(c SortieCartePos, index map[string]*zoneIndexee) []couloir {
	var out []couloir
	for _, p := range c.Positions {
		if p.NbCellules < seuilCouloirCellules {
			continue
		}
		pts := nouvelleSurface(p.Polygone).echantillons()
		out = append(out, jugeCouloir(p, pts, index))
	}
	return out
}

// jugeCouloir mesure les parts d'une grande position dans chaque zone, et la LIT :
// « salle entiere » si sa zone dominante est couverte a plus de la moitie, « couloir
// traversant » si au moins deux zones portent chacune un quart ou plus de la position (elle
// enjambe une frontiere), « position large » sinon (une partie d'une seule zone : ce que D1
// appelle une position).
func jugeCouloir(p SortiePosition, pts [][2]float64, index map[string]*zoneIndexee) couloir {
	type part struct {
		nom             string
		dansPos, deZone float64
	}
	var parts []part
	for nom, e := range index {
		if dedans := partDansZone(pts, e); dedans >= 0.10 {
			parts = append(parts, part{nom, dedans, partCouverte(e.points, nouvelleSurface(p.Polygone))})
		}
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].dansPos > parts[j].dansPos })
	k := couloir{ID: p.ID, NbCellules: p.NbCellules, AireM2: aireEchantillonnee(pts)}
	traversees := 0
	for _, x := range parts {
		k.Zones = append(k.Zones, fmt.Sprintf("%s : %.0f %% de la position / %.0f %% de la zone",
			x.nom, x.dansPos*100, x.deZone*100))
		if x.dansPos >= 0.25 {
			traversees++
		}
	}
	switch {
	case len(parts) > 0 && parts[0].deZone >= 0.5:
		k.Lecture = fmt.Sprintf("SALLE ENTIERE : la zone dominante (%s) est couverte a %.0f %%",
			parts[0].nom, parts[0].deZone*100)
	case traversees >= 2:
		k.Lecture = fmt.Sprintf("COULOIR TRAVERSANT : a cheval sur %d zones a 25 %% ou plus", traversees)
	default:
		k.Lecture = "POSITION LARGE : une partie d'une seule zone, pas la zone"
	}
	return k
}
