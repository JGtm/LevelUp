// replay_contract_test.go — LE CONTRAT DU DOCUMENT DE REJEU, CHAMP PAR CHAMP.
//
// CE QUE CE FICHIER FERME. L artefact de rejeu publie des dizaines de champs (le compte du
// jour et son historique : wantReplayDocumentFields). Le contrat OpenAPI n en a
// longtemps decrit que 6, et les types TypeScript etaient ECRITS A LA MAIN hors du fichier
// genere : trois verites parallelles, dont deux se corrigeaient a la main. Le cout de cette
// divergence a ete paye — l interface manuelle nommait `weapon` le champ que le contrat nomme
// `w`, et pendant toute la vie du rejeu 2D les huit familles d effet de tir sont restees
// inatteignables sans que rien ne le signale.
//
// LES TROIS SONT ALIGNEES DEPUIS LE 2026-07-31 (le document est tire du contrat genere, et le
// contrat est genere depuis les types Go). CE QUI MANQUAIT ETAIT LE TEST : sans lui, rien
// n empeche la divergence de revenir, et elle reviendra silencieusement.
//
// CE QUE CE TEST AJOUTE AU GOLDEN OPENAPI EXISTANT. `openapi_golden_test.go` compare le contrat
// regenere au fichier commite, octet pour octet — il attrape TOUTE derive, mais il exige CGO et
// il ne dit pas CE QUI a change. Celui-ci est CGO-free, il ne regarde que la famille du rejeu, et
// il nomme le champ qui manque. Les deux se completent : l un est exhaustif, l autre est lisible.
//
// LE VOLET TypeScript est teste ailleurs, et il le doit : la frontiere de nullabilite vit dans
// `apps/web/src/features/match-replay/replayNormalize.ts`, et c est la que se verifie qu aucun
// tableau du contrat ne lui echappe (cf. replayContract.test.ts).
//
// LES CHAMPS QUE PERSONNE NE LIT — INVENTAIRE CONSIGNE, NON TRAITE (leur sort est le lot 3.6 du
// plan de finalisation ; ce jalon-ci VERROUILLE, il ne supprime pas). Re-mesure du 2026-07-31,
// par grep sur `features/match-replay/` et la route du rejeu, en excluant la frontiere et les
// tests — DOUZE champs publies dont aucun consommateur :
//
//	schemaVersion      le client ne branche sur aucune version ; il lit ce qu il trouve
//	structureBounds    le cadrage se fait sur `bounds`, jamais sur l etendue de la structure
//	MapObject.typeId   aucun style par famille d objet Forge n est rendu
//	Surface.zb         la face INFERIEURE ; seule la superieure sert de sol
//	RosterEntry.filmIndex  le client joint par xuid, jamais par index — c est la regle
//	BridgeHealth.livesNamed / livesTotal / indexReadings   l ecran affiche les verdicts, pas
//	                   les compteurs qui les fondent
//	Inventory.cand     le nombre de lectures possibles du bloc de munitions
//	Projectile.rest    le seul champ qui CERTIFIE une fin de vol, et il n est pas dessine
//	Point.hp           publie a 0,6 % de couverture, et volontairement pas mis en barre
//	Track.name         cas a part : il n est JAMAIS ECRIT par le Go (le film ne porte aucun
//	                   gamertag sur les traces). Il n a donc pas de lecteur parce qu il n a
//	                   rien a lire — c est le seul des douze dont la suppression est acquise.
//
// AUCUN DE CES CHAMPS N EST SUPPRIME ICI. Trois familles s y melangent — jamais ecrit,
// deliberement publie sans etre affiche (hp, rest), et simplement pas encore consomme — et les
// trancher demande un arbitrage produit, pas un test.

package contracttest_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"levelup/go-api/internal/analysis/replay"
)

// replaySchemas apparie chaque type Go publie dans l artefact au schema qui doit le decrire.
//
// LA LISTE EST ECRITE, PAS DERIVEE : un parcours automatique des types atteignables depuis
// ReplayDocument aurait le defaut de suivre le code — si un type sortait du document, il
// sortirait aussi du test, et personne ne le verrait.
var replaySchemas = []struct {
	schema string
	value  any
}{
	{"ReplayDocument", replay.ReplayDocument{}},
	{"Track", replay.Track{}},
	{"Point", replay.Point{}},
	{"Shot", replay.Shot{}},
	{"Grenade", replay.Grenade{}},
	{"Projectile", replay.Projectile{}},
	{"Loadout", replay.Loadout{}},
	{"Inventory", replay.Inventory{}},
	{"AbilityRead", replay.AbilityRead{}},
	{"EquipmentEpisode", replay.EquipmentEpisode{}},
	{"EquipmentCoverage", replay.EquipmentCoverage{}},
	{"GrappleLine", replay.GrappleLine{}},
	{"GrappleCoverage", replay.GrappleCoverage{}},
	{"AmmoSlot", replay.AmmoSlot{}},
	{"Surface", replay.Surface{}},
	{"MapObject", replay.MapObject{}},
	{"Bounds", replay.Bounds{}},
	{"RosterEntry", replay.RosterEntry{}},
	{"ObjectiveAction", replay.ObjectiveAction{}},
	{"Coverage", replay.Coverage{}},
	{"LayerCoverage", replay.LayerCoverage{}},
	{"BridgeHealth", replay.BridgeHealth{}},
}

// wantReplayDocumentFields : le nombre de champs que l artefact publie. Ecrit ici pour que le
// chiffre du chantier soit verifiable et pas seulement affirme.
//
// CHRONIQUE DU COMPTE (un champ n entre au document que par cette ligne) :
//
//	22 -> 23  2026-08-05  `objectives`, le calque d actions d objectif, entre a l integration
//	                      de `feat/re-mode-score`.
//	23 -> 25  2026-08-13/14  DEUX champs, un par lot de la v7.5 :
//	                      - `killEffects` (lot 2) : la table qui donne leur famille de RENDU
//	                        aux effets de MORT. Les kills du feed portent un weapon_key resolu
//	                        cote base, jamais un identifiant d arme film — sans cette table le
//	                        client ne peut joindre aucun effet a un kill.
//	                      - `mapObjectives` (lot 4) : le calque STATIQUE des objectifs du mode
//	                        joue (zones, apparitions et livraisons de drapeau, socles), rempli
//	                        A LA REQUETE par le service et jamais ecrit dans l artefact —
//	                        l artefact ne connait ni sa carte ni son mode.
//	25 -> 27  2026-08-14  DEUX champs, un par item du lot 7 :
//	                      - `originMs` (7.2) : l ORIGINE de la frame 0 sur l horloge du fil.
//	                        L horodatage de paquet du film est une horloge MOTEUR (des milliers
//	                        de secondes depuis le demarrage du jeu) : sans cette origine publiee,
//	                        le client n a aucun zero commun avec les events de la Match View, et
//	                        le fil arrivait 3,6 s a 40 s apres le flash des fiches.
//	                      - `neutralDeaths` (7.1) : les morts que personne ne revendique
//	                        (suicide, environnement...), avec la NATURE du degat fatal — c est
//	                        elle qui choisit l icone du type de mort au fil.
//
//	27 -> 28  2026-08-14  `abilities` (plan PLAN_RANG_CAPACITE_I48, etape 1.2) : le RANG de
//	                      palette de la capacite d armure portee. UN CHAMP ENTRE, mais un
//	                      autre CHANGE DE SENS en meme temps et c est le vrai evenement :
//	                      `Inventory.a` portait `rang - 16` (l ancre du canal d image-cle se
//	                      termine par `010`, les bits de poids fort du rang, si bien que ce
//	                      canal ne voit que 16..23). Le champ `a` est RETIRE plutot que
//	                      reinterprete — republier une autre grandeur sous la meme cle aurait
//	                      laisse tout client non mis a jour lire un nombre qui ne veut plus dire
//	                      la meme chose. `abilityLabels` change de cle avec (rang, plus index).
//
//	28 -> 29  2026-08-16  `equipmentEpisodes` (plan PLAN_EQUIPEMENT_TI37, phase 1) : l etat
//	                      ACTIF du camouflage et du surbouclier, en episodes dates par vie
//	                      — les DEUX familles dont l etat est MESURE (i28 queue[1] binaire
//	                      0/4095 exclusif aux vies rang 8 ; i5 non clampe, regle q > 64, 0
//	                      faux positif sur ~150 000 mesures hors porteurs). Les autres
//	                      familles restent SANS etat : les deployables ne se datent pas par
//	                      les canaux mesures, la mobilite n a pas d instant d usage par i54.
//	                      `Coverage` gagne en meme temps son bloc `equipment` (vies
//	                      porteuses / vies publiees, par famille) : N episodes sans
//	                      denominateur se lirait comme une exhaustivite.
//
//	29 -> 30  2026-08-16  `grappleLines` (plan PLAN_GRAPPIN_LIGNE, phase 1) : les TRACTIONS
//	                      de grappin — fenetre datee par vie [t0, t1], du tir a l ARRIVEE
//	                      mesuree sur la trajectoire, et point d accroche en coordonnees
//	                      monde. Source : le corps tag==3 d i59, porte sur grammaire MESUREE
//	                      et prouve au gate 0 du plan (marche a l ecart zero sur 3 films,
//	                      ancre fixe a 0,05-0,07 u pres, distance joueur->ancre decroissante
//	                      contre temoins melanges effondres). `Coverage` gagne son bloc
//	                      `grapple` (tirs, accroches, tractions, rates, corps non
//	                      decodables) : N tractions sans ses rejets se lirait comme une
//	                      exhaustivite.
//
// Les six fois, ce test a ATTRAPE l ecart : une branche publiait le champ avant que le
// chiffre ne le dise. Contrat regenere (`make openapi-gen`), jamais ecrit a la main.
const wantReplayDocumentFields = 30

// TestReplayContractDescribesEveryPublishedField : AUCUN CHAMP PUBLIE SANS DESCRIPTION, ET
// AUCUNE DESCRIPTION SANS CHAMP.
//
// Les deux sens comptent, et pas pour la meme raison. Un champ publie que le contrat ignore est
// une donnee que le client ne peut pas typer — c est ce qui a produit l interface manuelle. Une
// propriete decrite que le Go ne publie plus est une promesse morte : un client la lit comme
// optionnelle et ne saura jamais qu elle ne viendra plus.
func TestReplayContractDescribesEveryPublishedField(t *testing.T) {
	schemas := loadReplaySchemas(t)
	for _, c := range replaySchemas {
		t.Run(c.schema, func(t *testing.T) {
			sch, ok := schemas[c.schema]
			if !ok {
				t.Fatalf("le contrat ne decrit AUCUN schema %q, alors que l artefact publie le "+
					"type Go correspondant", c.schema)
			}
			got := jsonFieldsOf(reflect.TypeOf(c.value))
			want := propertyNamesOf(sch)
			for _, f := range got {
				if !contains(want, f) {
					t.Errorf("champ %q publie par le Go et ABSENT du contrat — un client ne peut "+
						"pas le typer, et c est exactement ainsi qu une interface ecrite a la "+
						"main reapparait", f)
				}
			}
			for _, p := range want {
				if !contains(got, p) {
					t.Errorf("propriete %q decrite par le contrat et PLUS publiee par le Go — une "+
						"promesse morte que le client lira comme optionnelle", p)
				}
			}
		})
	}
}

// TestReplayDocumentFieldCountIsFrozen : le chiffre du chantier, verifie des deux cotes.
//
// LE NOM NE PORTE PLUS LE CHIFFRE (il a dit « TwentyTwo » jusqu au 2026-08-14 alors que le
// document en publiait 25) : un compte qui bouge se lit dans wantReplayDocumentFields et sa
// chronique, pas dans un identifiant que personne ne pense a renommer.
func TestReplayDocumentFieldCountIsFrozen(t *testing.T) {
	got := jsonFieldsOf(reflect.TypeOf(replay.ReplayDocument{}))
	if len(got) != wantReplayDocumentFields {
		t.Errorf("%d champ(s) publie(s) par l artefact, attendu %d : %v",
			len(got), wantReplayDocumentFields, got)
	}
	props := propertyNamesOf(loadReplaySchemas(t)["ReplayDocument"])
	if len(props) != wantReplayDocumentFields {
		t.Errorf("%d propriete(s) decrite(s) par le contrat, attendu %d", len(props),
			wantReplayDocumentFields)
	}
}

// TestReplayContractCarriesTupleArity : L ARITE D UN TUPLE SE DIT, ELLE NE SE DEVINE PAS.
//
// Le Go ecrit `Poly [][2]float32` et `P [][3]float32` : des sommets XY et des pas [dt, x, y].
// JSON Schema n a pas de type « tuple », mais il a `minItems` = `maxItems`, et c est exactement
// ce que cela veut dire. Sans cette paire, le contrat ne dit plus que « un tableau de nombres »,
// et la frontiere web qui retablit l arite le ferait sur une supposition au lieu d une lecture.
func TestReplayContractCarriesTupleArity(t *testing.T) {
	schemas := loadReplaySchemas(t)
	for _, c := range []struct {
		schema, prop string
		goType       reflect.Type
		goField      string
	}{
		{"Surface", "poly", reflect.TypeOf(replay.Surface{}), "Poly"},
		{"Projectile", "p", reflect.TypeOf(replay.Projectile{}), "P"},
	} {
		t.Run(c.schema+"."+c.prop, func(t *testing.T) {
			f, ok := c.goType.FieldByName(c.goField)
			if !ok {
				t.Fatalf("champ Go %s.%s introuvable", c.goType.Name(), c.goField)
			}
			// [][N]float32 : la longueur du tuple est celle du tableau interieur.
			wantLen := f.Type.Elem().Len()
			tuple := tupleSchemaOf(t, schemas[c.schema], c.prop)
			mn, mx := intAt(tuple, "minItems"), intAt(tuple, "maxItems")
			if mn != wantLen || mx != wantLen {
				t.Errorf("arite decrite minItems=%d maxItems=%d, attendu %d des deux cotes — le Go "+
					"ecrit un [%d]float32, et c est la SEULE facon de le dire en JSON Schema",
					mn, mx, wantLen, wantLen)
			}
		})
	}
}

// TestReplayContractDoesNotRequireOptionalFields : `omitempty` et `required` ne se contredisent pas.
//
// Un champ omis a la serialisation ne peut pas etre requis a la lecture. La contradiction ne
// casse rien au premier abord — elle casse le jour ou le champ vaut zero.
func TestReplayContractDoesNotRequireOptionalFields(t *testing.T) {
	schemas := loadReplaySchemas(t)
	for _, c := range replaySchemas {
		omit := omitemptyFieldsOf(reflect.TypeOf(c.value))
		for _, r := range requiredOf(schemas[c.schema]) {
			if contains(omit, r) {
				t.Errorf("%s : le contrat exige %q alors que le Go l omet quand il est vide — "+
					"la contradiction se revele le jour ou le champ vaut zero", c.schema, r)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Lecture du contrat et reflexion
// ---------------------------------------------------------------------------

// loadReplaySchemas lit components.schemas de api/openapi.yaml.
func loadReplaySchemas(t *testing.T) map[string]map[string]any {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "api", "openapi.yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fige, relatif au fichier de test
	if err != nil {
		t.Fatalf("lecture du contrat %s : %v", path, err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("contrat illisible : %v", err)
	}
	if len(doc.Components.Schemas) == 0 {
		t.Fatal("le contrat ne porte aucun schema : la lecture est fausse")
	}
	return doc.Components.Schemas
}

// jsonFieldsOf rend les noms JSON des champs serialises d une struct.
func jsonFieldsOf(rt reflect.Type) []string {
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		if name, ok := jsonNameOf(rt.Field(i)); ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// omitemptyFieldsOf rend les champs que le Go OMET quand ils sont vides.
func omitemptyFieldsOf(rt reflect.Type) []string {
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		name, ok := jsonNameOf(f)
		if !ok || !strings.Contains(f.Tag.Get("json"), ",omitempty") {
			continue
		}
		out = append(out, name)
	}
	return out
}

// jsonNameOf rend le nom JSON d un champ, et false s il n est pas serialise.
func jsonNameOf(f reflect.StructField) (string, bool) {
	if f.PkgPath != "" { // champ non exporte
		return "", false
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		name = f.Name
	}
	return name, true
}

func propertyNamesOf(schema map[string]any) []string {
	props, _ := schema["properties"].(map[string]any)
	out := make([]string, 0, len(props))
	for k := range props {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func requiredOf(schema map[string]any) []string {
	raw, _ := schema["required"].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// tupleSchemaOf descend `properties.<prop>.items` — l ELEMENT de la liste, c est-a-dire le
// tuple lui-meme. C est lui qui porte l arite ; le niveau en dessous ne decrit qu un nombre.
func tupleSchemaOf(t *testing.T, schema map[string]any, prop string) map[string]any {
	t.Helper()
	props, _ := schema["properties"].(map[string]any)
	p, _ := props[prop].(map[string]any)
	tuple, ok := p["items"].(map[string]any)
	if !ok {
		t.Fatalf("propriete %q : aucun schema d element — l arite ne peut plus s exprimer", prop)
	}
	if _, hasInner := tuple["items"]; !hasInner {
		t.Fatalf("propriete %q : l element n est plus un tableau — ce n est plus un tuple", prop)
	}
	return tuple
}

func intAt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return -1
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
