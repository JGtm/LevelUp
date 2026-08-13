package himap

// SONDE DURABLE (2026-08-13) — LOT B : le SAUT `bloc`/`scen`/`mach` -> `hlmt` des objets
// Forge sans modele direct.
//
// AVANT le saut, sur Vagabond : 3 558 des 4 709 objets rendus (75,6 %) ; 1 113 sans modele —
// leurs `food` ne portent aucune ref `rtgo` inline et passent par une definition d'objet
// `bloc` (963 objets), `scen` (173) ou `mach` (9), qui reference le modele `hlmt`
// (sonde F1CouvertureRtgo, 2026-08-10). MESURE DU 2026-08-13 : ces `hlmt` ne referencent
// AUCUN `rtgo` — leur geometrie est un tag `mode` (render_model), lu par
// `NewRenderModelAsset` (offsets Reclaimer `RenderModelTag.cs`, cf. rtgo.go). Une partie des
// definitions vit dans les globals de la variante `any`, d'ou l'elargissement d'indexForge.
//
// CE QUE CETTE SONDE MESURE, ET POURQUOI ELLE EST DURABLE : elle joue le saut avec l'INDEX
// DE PRODUCTION (`indexForge`) et les FONCTIONS DE PRODUCTION (`modeleParSaut`,
// `ouvreAsset`) — ce qu'elle publie est ce que la cuisson fait, pas une replique. Elle
// ASSERTE la couverture mesuree : si l'index ou la chaine regressent, elle vire au rouge.

import (
	"context"
	"testing"

	"levelup/go-api/internal/himodule"
)

func TestSondeForgeSautBlocScenMach(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	carte := CartesForge[0]
	v, _ := chargeCarteForge(t, carte)
	idx, forge, err := indexForge(OptionsCuissonForge{
		RacineDeploy:        root,
		CheminModuleCanevas: CheminCanevasForge(carte),
	})
	if err != nil {
		t.Fatal(err)
	}
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}

	nParType := map[int32]int{}
	for _, o := range v.Objects {
		nParType[o.TypeID]++
	}

	ctx := context.Background()
	directsT, directsO := 0, 0
	sautT := map[string]int{}
	sautO := map[string]int{}
	mortsO, sansFoodT, sansFoodO := 0, 0, 0
	restantsT, restantsO := 0, 0
	var exemplesRestants []int32
	echecs := map[string]int{}          // premier maillon manquant -> nb de type_id
	modelesSaut := map[refModele]bool{} // modeles distincts atteints par le saut
	for id, n := range nParType {
		if _, mort := TypesVolumesDeMort[id]; mort {
			mortsO += n
			continue
		}
		f, ok := foodParID[uint32(id)]
		if !ok {
			sansFoodT++
			sansFoodO += n
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			t.Errorf("food %d : extraction : %v", id, err)
			continue
		}
		if refs := refsInlineDuGroupe(tag, idx, GroupeRtgo); len(refs) > 0 {
			directsT++
			directsO += n
			continue
		}
		if m, ok := modeleParSaut(ctx, idx, tag); ok {
			sautT[m.groupe]++
			sautO[m.groupe] += n
			modelesSaut[m] = true
			continue
		}
		restantsT++
		restantsO += n
		echecs[maillonManquant(idx, tag)]++
		if len(exemplesRestants) < 12 {
			exemplesRestants = append(exemplesRestants, id)
		}
	}
	t.Logf("vagabond : %d objets, %d type_id · index de production : %d tags", len(v.Objects), len(nParType), idx.Taille())
	t.Logf("rtgo DIRECT : %3d type_id · %4d objets", directsT, directsO)
	for _, g := range []string{GroupeRtgo, GroupeMode} {
		t.Logf("par SAUT -> %s : %3d type_id · %4d objets", g, sautT[g], sautO[g])
	}
	t.Logf("volumes de mort (exclus par construction) : %d objets", mortsO)
	t.Logf("sans tag food : %d type_id · %d objets", sansFoodT, sansFoodO)
	t.Logf("RESTANTS sans modele : %d type_id · %d objets · ex. %v", restantsT, restantsO, exemplesRestants)
	for maillon, n := range echecs {
		t.Logf("  maillon manquant %-40q : %d type_id", maillon, n)
	}

	// LA PREUVE DE DECODAGE : un modele resolu dont l'asset ne rend aucun maillage serait un
	// saut de papier. On ouvre chaque modele du saut par la fonction de production.
	ouverts, decodables := 0, 0
	for m := range modelesSaut {
		a := ouvreAsset(ctx, idx, m.id, m.groupe)
		if a == nil {
			continue
		}
		ouverts++
		for mi := 0; mi < a.MeshCount(); mi++ {
			if a.Mesh(mi) != nil {
				decodables++
				break
			}
		}
	}
	t.Logf("modeles du saut : %d distincts · %d ouverts · %d avec au moins un maillage decodable",
		len(modelesSaut), ouverts, decodables)

	// TEMOINS — caler sur la mesure du 2026-08-13 (installation Steam a jour) ; une
	// regression de l'index (chemins d'indexForge) ou d'un maillon de la chaine fait
	// retomber ces comptes, et c'est precisement ce qu'on veut voir.
	totalSautO := 0
	for _, n := range sautO {
		totalSautO += n
	}
	if totalSautO < 1000 {
		t.Errorf("le saut ne couvre que %d objets (>= 1000 attendus)", totalSautO)
	}
	if restantsO > 60 {
		t.Errorf("%d objets restent sans modele (<= 60 attendus)", restantsO)
	}
	if decodables < ouverts*8/10 || ouverts == 0 {
		t.Errorf("seuls %d/%d modeles du saut decodent un maillage (>= 80 %% attendus)", decodables, ouverts)
	}
}

// maillonManquant dit ou la chaine du saut casse pour un tag `food` donne — diagnostic, pour
// que la sonde dise POURQUOI et pas seulement combien.
func maillonManquant(idx *ModuleIndex, tagFood []byte) string {
	for _, groupe := range groupesSautForge {
		for _, hObjet := range refsInlineDuGroupe(tagFood, idx, groupe) {
			objet, err := idx.Extract(hObjet)
			if err != nil {
				return groupe + " illisible"
			}
			if len(refsInlineDuGroupe(objet, idx, GroupeHlmt)) == 0 {
				return groupe + " sans ref hlmt"
			}
			return groupe + " -> hlmt sans ref rtgo ni mode"
		}
	}
	return "aucune ref bloc/scen/mach dans le food"
}
