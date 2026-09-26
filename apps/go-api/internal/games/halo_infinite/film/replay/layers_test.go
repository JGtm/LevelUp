package replay

// layers_test.go — LA FERMETURE DE LA TABLE DES CALQUES (lot 4.2.1-a du
// PLAN_DECODEUR_FILM_2026-09-13).
//
// # CE QU IL GARDE, ET SUR LE MODELE DE QUI
//
// Le modele est `TestDocumentShapeCalquesALaRequeteRestentHorsCuisson`
// (`document_shape_test.go`) : une table de decisions, et un test qui refuse qu elle derive
// d avec le type qu elle decrit. Trois refus :
//
//	(1) un champ racine CUIT sans entree ni justification datee  -> la table a un trou
//	(2) une entree qui nomme un champ racine disparu             -> la table a un perime
//	(3) une valeur qui n est pas l une des SIX revisions         -> `layers` deviendrait du texte
//
// Le quatrieme refus est comportemental : `calquesProduits` doit suivre ses gardes DANS LES DEUX
// SENS (garde fermee = pas d entree, garde ouverte = entree), sans quoi l absence d une entree ne
// voudrait plus rien dire.
//
// PLANCHER ANTI-MUET : un balayage par reflexion qui ne trouverait plus la racine rendrait vert
// sur du vide. Le compte de champs a un plancher, et il echoue bruyamment.

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// plancherChampsRacine : 59 balises `json:` a la racine du document au 2026-09-17 (le meme
// compte que `wantReplayDocumentFields`, `apps/go-api/contracttest/replay_contract_test.go`). Un
// balayage qui en rend nettement moins n a pas trouve le type.
const plancherChampsRacine = 50

// baliseDatee : une justification de [calquesSansCouche] commence par sa date.
var baliseDatee = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} — .{20,}$`)

// balisesRacineCuites rend les balises JSON des champs racine que la CUISSON ecrit, c est-a-dire
// toutes sauf les calques resolus a la requete. Elle lit `calquesALaRequete`, la table que
// l empreinte de forme cuite emploie deja : deux listes des memes calques divergeraient.
func balisesRacineCuites(t *testing.T) map[string]bool {
	t.Helper()
	root := reflect.TypeOf(ReplayDocument{})
	if root.NumField() < plancherChampsRacine {
		t.Fatalf("balayage muet : %d champs a la racine du document, plancher %d — "+
			"le type n a pas ete trouve, ou il a fondu", root.NumField(), plancherChampsRacine)
	}
	out := make(map[string]bool, root.NumField())
	for i := 0; i < root.NumField(); i++ {
		f := root.Field(i)
		if f.PkgPath != "" {
			continue
		}
		balise := baliseJSON(f)
		if balise == "" || balise == "-" || calquesALaRequete[balise] {
			continue
		}
		out[balise] = true
	}
	return out
}

// TestCalquesCouvrentTousLesChampsRacineCuits : aucun champ cuit sans decision, aucun champ
// classe deux fois.
func TestCalquesCouvrentTousLesChampsRacineCuits(t *testing.T) {
	cuits := balisesRacineCuites(t)
	for balise := range cuits {
		_, attribue := couchesDesCalques[balise]
		_, exempte := calquesSansCouche[balise]
		switch {
		case attribue && exempte:
			t.Errorf("le champ racine %q est a la fois attribue a une couche et exempte — "+
				"les deux tables se contredisent, trancher", balise)
		case !attribue && !exempte:
			t.Errorf("le champ racine CUIT %q n a ni couche productrice ni justification : "+
				"ajouter une ligne a `couchesDesCalques` (la couche qui decode ce qu il publie) "+
				"ou a `calquesSansCouche` (avec sa date et sa raison)", balise)
		}
	}
}

// TestCalquesNeNommentQueDesChampsRacineExistants : une entree perimee se retire, elle ne se
// garde pas « au cas ou » — sans quoi `layers` publierait une cle qu aucun lecteur ne comble.
func TestCalquesNeNommentQueDesChampsRacineExistants(t *testing.T) {
	cuits := balisesRacineCuites(t)
	for balise := range couchesDesCalques {
		if !cuits[balise] {
			t.Errorf("`couchesDesCalques` nomme %q, qui n est pas un champ racine cuit du "+
				"document — retirer l entree (ou la deplacer si le calque est passe a la requete)", balise)
		}
	}
	for balise := range calquesSansCouche {
		if !cuits[balise] {
			t.Errorf("`calquesSansCouche` nomme %q, qui n est pas un champ racine cuit du "+
				"document — retirer l entree", balise)
		}
	}
}

// TestCalquesNePortentQueLesRevisionsConnues : LA FERMETURE DES VALEURS (six depuis le lot J3.3,
// ou `facts.Rev` s est scindee en `killsource.Rev` et `objectives.Rev`). Une chaine libre
// dans `layers` ferait de la revision d un calque du texte, que nul verdict ne saurait comparer.
func TestCalquesNePortentQueLesRevisionsConnues(t *testing.T) {
	connues := map[string]string{
		source.Rev:              "source",
		profile.Rev:             "profile",
		grammar.Rev:             "grammar",
		killsource.Rev:          "kill-feed",
		objectives.Rev:          "objectifs",
		revisionDeLaPublication: "publication",
	}
	if len(connues) != 6 {
		t.Fatalf("deux des six revisions portent la MEME valeur (%d valeurs distinctes) : "+
			"le verdict par couche ne saurait plus les distinguer", len(connues))
	}
	for balise, rev := range couchesDesCalques {
		if _, ok := connues[rev]; !ok {
			t.Errorf("le calque %q porte %q, qui n est aucune des six revisions connues "+
				"(source, profile, grammar, killsource, objectives, publication) — jamais de chaine libre", balise, rev)
		}
	}
}

// TestCalquesRevisionsGardentLeurPrefixe : les valeurs se lisent dans l artefact, et leur prefixe
// est ce qui dit la couche a un humain comme au badge.
func TestCalquesRevisionsGardentLeurPrefixe(t *testing.T) {
	for _, cas := range []struct{ rev, prefixe string }{
		{source.Rev, "source-"},
		{profile.Rev, "profile-"},
		{grammar.Rev, "grammar-"},
		{killsource.Rev, "killsource-"},
		{objectives.Rev, "objectives-"},
		{revisionDeLaPublication, "publication-"},
	} {
		if !strings.HasPrefix(cas.rev, cas.prefixe) {
			t.Errorf("la revision %q ne porte plus le prefixe %q : la table des calques et le "+
				"badge par couche le lisent", cas.rev, cas.prefixe)
		}
	}
}

// TestCalquesALaRequeteRestentHorsDeLaTable : les trois calques que la cuisson n ecrit jamais
// n ont ni couche ni exemption — ils ne sont pas cuits, il n y a rien a justifier.
func TestCalquesALaRequeteRestentHorsDeLaTable(t *testing.T) {
	if len(calquesALaRequete) == 0 {
		t.Fatal("`calquesALaRequete` est vide : le balayage des champs cuits ne garde plus rien")
	}
	for balise := range calquesALaRequete {
		if _, ok := couchesDesCalques[balise]; ok {
			t.Errorf("le calque a la requete %q est attribue a une couche : la cuisson ne "+
				"l ecrit pas, il n a pas de revision de production", balise)
		}
		if _, ok := calquesSansCouche[balise]; ok {
			t.Errorf("le calque a la requete %q est exempte : il n est pas cuit, il n y a rien "+
				"a justifier — retirer l entree", balise)
		}
	}
}

// TestCalquesSansCouchePortentUneJustificationDatee : une exemption est une DECISION, donc elle
// porte sa date. Une table vide serait le ratchet ; une entree sans date serait un trou poli.
func TestCalquesSansCouchePortentUneJustificationDatee(t *testing.T) {
	for balise, raison := range calquesSansCouche {
		if !baliseDatee.MatchString(raison) {
			t.Errorf("l exemption de %q ne porte pas de justification datee ("+
				"attendu « AAAA-MM-JJ — raison ») : %q", balise, raison)
		}
	}
}

// TestCalquesGardesNommentUnCalqueDeLaTable : une garde sur un calque inconnu ne garderait rien.
func TestCalquesGardesNommentUnCalqueDeLaTable(t *testing.T) {
	for balise := range gardesDeProduction {
		if _, ok := couchesDesCalques[balise]; !ok {
			t.Errorf("`gardesDeProduction` garde %q, qui n est pas dans `couchesDesCalques` : "+
				"la garde ne s appliquerait a rien", balise)
		}
	}
}

// optionsToutesGardesOuvertes rend les options minimales qui ouvrent LES SEIZE gardes.
func optionsToutesGardesOuvertes() Options {
	return Options{
		MapQuant:            &profile.MapQuantEntry{},
		Inventory:           []KeyframeInventory{},
		AbilityImpulseStats: types.AbilityImpulseStats{Scanned: true},
		AbilityChargeStats:  types.AbilityChargeStats{Scanned: true},
		Score:               &ScoreInput{},
		Flag:                FlagInput{Scanned: true},
		Vip:                 VipInput{Scanned: true},
		Skull:               SkullInput{Scanned: true},
		Bomb:                BombInput{Scanned: true, CarryScanned: true},
		Zone:                ZoneInput{Scanned: true},
		Vehicles:            VehicleScan{Scanned: true},
	}
}

// TestCalquesProduitsSuiventLesGardes : LA PREUVE DANS LES DEUX SENS. Gardes fermees, les calques
// gardes sont absents et les autres presents ; gardes ouvertes, la table entiere sort.
func TestCalquesProduitsSuiventLesGardes(t *testing.T) {
	if calquesProduits(nil, Options{}) != nil {
		t.Error("`calquesProduits` sur un document nil doit rendre nil : declarer des calques " +
			"pour un document qui n existe pas serait une affirmation")
	}
	// `FrameCount` NON NUL : sans lui le document serait « sans aucune piste », cas ou la table ne
	// declare que la publication (cf. `calquesProduits`) — ce que le test suivant couvre.
	ferme := calquesProduits(&ReplayDocument{FrameCount: 1}, Options{})
	for balise := range couchesDesCalques {
		_, gardee := gardesDeProduction[balise]
		_, produit := ferme[balise]
		switch {
		case gardee && produit:
			t.Errorf("garde FERMEE et le calque %q est quand meme declare produit : l absence "+
				"d entree ne voudrait plus rien dire", balise)
		case !gardee && !produit:
			t.Errorf("le calque %q n est pas garde et n est pourtant pas declare produit : sa "+
				"passe a tourne, son entree doit etre la meme vide", balise)
		}
	}
	origine := int64(42)
	ouvert := calquesProduits(&ReplayDocument{FrameCount: 1, OriginMs: &origine}, optionsToutesGardesOuvertes())
	if len(ouvert) != len(couchesDesCalques) {
		for balise := range couchesDesCalques {
			if _, ok := ouvert[balise]; !ok {
				t.Errorf("toutes gardes ouvertes, le calque %q manque : sa garde n est pas "+
					"ouverte par `optionsToutesGardesOuvertes` (ou elle lit autre chose)", balise)
			}
		}
	}
	for balise, rev := range ouvert {
		if rev != couchesDesCalques[balise] {
			t.Errorf("le calque %q est publie sous %q la ou la table dit %q", balise, rev,
				couchesDesCalques[balise])
		}
	}
}

// calquesGardesGeles — LA LISTE GELEE DES CALQUES GARDES, au 2026-09-17 (lot 4.2.1-a).
//
// POURQUOI UN GEL, ET DANS LES DEUX SENS. Une garde qui APPARAIT retire une entree de `layers`
// sur tout un pan du parc — un lecteur lirait « ce calque n a pas ete produit » sur des artefacts
// qui le portent ; une garde qui DISPARAIT fait affirmer « produit » sur des films ou la passe
// n a jamais tourne. Les deux sont des decisions, et aucune des deux ne se voit dans un diff de
// table sans ratchet.
//
// UNE SEULE CHAINE, TRIEE, ET C EST DELIBERE : une tranche de seize litteraux ferait de chaque
// nom de calque une troisieme copie du meme litteral dans le paquet (`goconst` mord a la
// quatrieme), et le remede — une constante par nom — rendrait les deux tables de `layers.go`
// illisibles ligne a ligne, ce qui est exactement ce qu elles existent pour etre.
const calquesGardesGeles = "abilityCharges abilityImpulses bombArmings bombCarries bombEvents " +
	"bombStats flagCarries flagReturnZone grappleLines inventory scoreTimeline skullCarries " +
	"t0FilmMs vehicles vipCrown zoneStates"

// TestCalquesGardesSontLaListeGelee : le jeu des calques gardes ne bouge pas en silence.
func TestCalquesGardesSontLaListeGelee(t *testing.T) {
	vues := make([]string, 0, len(gardesDeProduction))
	for balise := range gardesDeProduction {
		vues = append(vues, balise)
	}
	sort.Strings(vues)
	if got := strings.Join(vues, " "); got != calquesGardesGeles {
		t.Errorf("le jeu des calques gardes a change.\n  gele : %s\n  vu   : %s\n"+
			"Une garde qui apparait retire une entree de `layers` sur tout un pan du parc ; une "+
			"garde qui disparait fait affirmer « produit » sur des films ou la passe n a pas "+
			"tourne. Mettre la liste a jour DANS le commit qui change la garde, avec sa raison.",
			calquesGardesGeles, got)
	}
}

// champRacineParBalise rend le champ de la racine que porte cette balise JSON.
func champRacineParBalise(doc *ReplayDocument, balise string) (reflect.Value, bool) {
	v := reflect.ValueOf(doc).Elem()
	for i := 0; i < v.NumField(); i++ {
		if baliseJSON(v.Type().Field(i)) == balise {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// champEstVide : un calque non produit ne porte ni ligne ni objet.
func champEstVide(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Pointer:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

// TestCalquesGardesFermeesLaissentLeCalqueVide — LE LIEN ENTRE LA TABLE DES GARDES ET LA CUISSON.
//
// Les tests ci-dessus prouvent que `calquesProduits` honore sa table ; celui-ci prouve que la
// table dit la verite sur l ASSEMBLAGE : toutes gardes fermees, une CUISSON REELLE
// (`BuildFromPositions`, positions publiees) n ecrit aucun des calques declares gardes.
//
// CE QU IL NE PROUVE PAS, et c est ecrit : la reciproque. Un calque VIDE n est pas un calque non
// produit — c est precisement pourquoi `layers` existe.
func TestCalquesGardesFermeesLaissentLeCalqueVide(t *testing.T) {
	doc := BuildFromPositions("m", "halo_infinite", positionsPourVersion(), nil,
		Options{FrameIntervalMS: 100})
	if len(doc.Tracks) == 0 {
		t.Fatal("temoin anti-muet : la cuisson n a publie aucune piste, le document est vide et " +
			"ce test ne garderait plus rien")
	}
	for balise := range gardesDeProduction {
		champ, ok := champRacineParBalise(&doc, balise)
		if !ok {
			t.Fatalf("la garde de %q ne nomme aucun champ racine du document", balise)
		}
		if !champEstVide(champ) {
			t.Errorf("le calque %q est declare GARDE, et la cuisson l ecrit pourtant toutes "+
				"gardes fermees : sa passe tourne sans garde, retirer l entree de "+
				"`gardesDeProduction` (son absence de `layers` mentirait)", balise)
		}
	}
}

// TestCalquesProduitsSurUnDocumentSansPisteNeDeclarentQueLaPublication : un film sans aucune
// position publie un document TEL QUEL — aucune des treize passes de calque n a tourne. La table
// ne doit alors declarer que les quatre calques de publication, et surtout pas les trente-et-un
// calques non gardes : ce serait le mensonge exact que `layers` existe pour empecher.
func TestCalquesProduitsSurUnDocumentSansPisteNeDeclarentQueLaPublication(t *testing.T) {
	doc := BuildFromPositions("m", "halo_infinite", nil, nil, Options{FrameIntervalMS: 100})
	if doc.FrameCount != 0 {
		t.Fatalf("temoin : un document sans position doit garder FrameCount a zero, vu %d", doc.FrameCount)
	}
	if len(doc.Layers) == 0 {
		t.Fatal("`layers` ABSENT sur un document cuit au schema 62 : son absence voudrait dire " +
			"« artefact anterieur », et le document sans piste en est un de plus")
	}
	for nom, rev := range doc.Layers {
		if rev != revisionDeLaPublication {
			t.Errorf("le calque %q est declare produit sous %q sur un document SANS PISTE : "+
				"sa passe n a pas tourne", nom, rev)
		}
	}
	for nom, rev := range couchesDesCalques {
		if rev != revisionDeLaPublication {
			continue
		}
		if _, ok := doc.Layers[nom]; !ok {
			t.Errorf("le calque de publication %q manque : il est ecrit par `ouvrir`, donc "+
				"produit meme sans piste", nom)
		}
	}
}

// TestLayersEstPoseParLaCuisson : la table ATTEINT le document. Sans ce test, `layers` pourrait
// rester nil en production sans qu aucune fermeture ne le dise.
func TestLayersEstPoseParLaCuisson(t *testing.T) {
	doc := BuildFromPositions("m", "halo_infinite", positionsPourVersion(), nil,
		Options{FrameIntervalMS: 100})
	if len(doc.Layers) == 0 {
		t.Fatal("`layers` absent d une cuisson reelle : la passe `poserLesCalquesProduits` ne " +
			"pose rien, ou elle n est pas appelee")
	}
	if doc.Layers["tracks"] != grammar.Rev {
		t.Errorf("`layers[tracks]` vaut %q, attendu %q : la table n est pas celle qui est posee",
			doc.Layers["tracks"], grammar.Rev)
	}
	for _, requete := range []string{
		"mapObjectives", "mapWeaponPads", "weaponTiers", "vehicleLabels", "vehicleWeapons",
	} {
		if _, ok := doc.Layers[requete]; ok {
			t.Errorf("le calque a la requete %q est declare produit par la cuisson", requete)
		}
	}
}

// TestChaqueConsommateurDeFaitsDateUnCalqueNonGarde : LE VERDICT DE RECUISSON RESTE SUR APRES LA
// SCISSION DES FAITS (lot J3.3).
//
// Un calque ne porte qu UNE revision, et quelques-uns lisent les deux consommateurs de faits (les
// portages et les statistiques de la bombe). Le verdict (`replaybuild.Digest.decodageIntact`)
// reste sur tant que CHAQUE famille est declaree par au moins un calque que sa passe produit
// toujours : une montee de l une ou de l autre se voit alors sur tout document assemble.
func TestChaqueConsommateurDeFaitsDateUnCalqueNonGarde(t *testing.T) {
	for _, rev := range []string{killsource.Rev, objectives.Rev} {
		var nonGardes []string
		for balise, r := range couchesDesCalques {
			if _, garde := gardesDeProduction[balise]; r == rev && !garde {
				nonGardes = append(nonGardes, balise)
			}
		}
		if len(nonGardes) == 0 {
			t.Errorf("aucun calque NON GARDE ne porte %q : une montee de cette revision passerait "+
				"inapercue sur un document ou ses calques gardes n ont pas tourne", rev)
		}
	}
}
