package killcollector

// positions_decoupage_catalogue_test.go — LOT 1.9.2 : LE DÉCOUPAGE D'i0 DES POSITIONS VIENT DU
// CATALOGUE, ET FAUSSER LE CATALOGUE SE VOIT.
//
// # CE QUE CE FICHIER TIENT, ET CE QU'IL LAISSE À `filmdec`
//
// Ici : LE CÂBLAGE. Le collecteur tient l'entrée de carte du match (`resolveMapBounds`) ; ce
// qu'il en fait — les bornes ET le découpage d'i0 — est ce que ce fichier verrouille. La lecture
// des BITS sous un découpage faussé est tenue par `filmdec/i0_catalogue_mutation_test.go`, au
// plus près de la grammaire.
//
// # LE DÉFAUT QUE LE LOT FERME
//
// `optionsDeBalayageDesPositions` partait de `DefaultScanFilmOptions()` et ne posait QUE
// `WorldRange` : `Layout` restait nil, donc `filmdec.DetectI0LayoutOf` décidait du découpage en
// mesurant le film. Le découpage d'axe est une DONNÉE DE PROFIL (D-3 d'ADR 0034) : la seule
// source est le catalogue de carte, et le chemin de cuisson l'imposait déjà depuis le
// 2026-09-03. Deux producteurs du même fait, deux règles — c'est exactement ce que D13 interdit.
//
// # POURQUOI UNE ASSERTION SUR `Layout != nil`, ET PAS SEULEMENT SUR SA VALEUR
//
// Un `Layout` nil ne produit pas d'erreur : il rend la main à l'auto-détection, en silence. Le
// jour où une refonte le remettrait à nil, une comparaison de valeurs n'aurait rien à comparer —
// c'est la non-nullité qui dit « le catalogue décide », pas la valeur.

import (
	"path/filepath"
	"runtime"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// catalogueDeBornesVersionne charge le catalogue de bornes RÉEL du dépôt — DONNÉE DE RÉFÉRENCE
// VERSIONNÉE (`data/titles/halo_infinite/reference/map_quant_bounds.json`, commitée), pas une
// sortie de sync : elle est disponible même dans un worktree sans `data/` de travail. Chemin
// résolu par `PathResolver` (CLAUDE.md : jamais de `filepath.Join(..., "data", ...)` à la main).
func catalogueDeBornesVersionne(t *testing.T) *filmdec.MapQuantCatalog {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	pr := titlePkg.NewPathResolver(repoRoot)
	cat, err := filmdec.LoadMapQuantCatalog(pr.MapQuantBoundsPath(titlePkg.DefaultSlug))
	if err != nil {
		t.Skipf("catalogue de bornes indisponible (%v) — positions non testables sans lui", err)
	}
	return cat
}

// optionsPourEntree rejoue le câblage de production : le contexte de film sous la règle du
// catalogue, puis les réglages de balayage qu'il en tire. Le film est nil — `NewFilmContextForMap`
// l'accepte, et le découpage imposé ne dépend que de l'entrée de carte.
func optionsPourEntree(entry filmdec.MapQuantEntry) filmdec.ScanFilmOptions {
	return optionsDeBalayageDesPositions(filmdec.NewFilmContextForMap(nil, &entry, nil), entry)
}

// TestPositionsImposentLeDecoupageDuCatalogue : pour CHAQUE carte du catalogue versionné, les
// réglages de balayage portent le découpage de cette carte et ses bornes — jamais un découpage
// nul, qui rendrait la décision à l'auto-détection.
func TestPositionsImposentLeDecoupageDuCatalogue(t *testing.T) {
	cat := catalogueDeBornesVersionne(t)
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue vide : le test ne garderait rien")
	}
	for nom, entry := range cat.Maps {
		opt := optionsPourEntree(entry)
		if opt.Layout == nil {
			t.Errorf("%s : ScanFilmOptions.Layout est nil — l auto-detection deciderait du "+
				"decoupage d i0 alors que le catalogue le porte (D-3 d ADR 0034)", nom)
			continue
		}
		if want := entry.Layout(); *opt.Layout != want {
			t.Errorf("%s : decoupage %s, attendu celui du catalogue %s", nom, *opt.Layout, want)
		}
		if opt.WorldRange == nil || *opt.WorldRange != entry.Range() {
			t.Errorf("%s : bornes monde %v, attendu celles du catalogue %v", nom, opt.WorldRange, entry.Range())
		}
	}
}

// TestMutationDuCatalogueChangeLeDecoupageDesPositions — LA MUTATION DU CÂBLAGE. Une entrée dont
// `axisWidths` est décalée d'un bit, ou dont `regionIndexBits` change, doit donner un découpage
// DIFFÉRENT à l'entrée du balayage. Si les deux découpages restaient égaux, c'est que le
// catalogue ne décide pas — et l'auto-détection « rattraperait » la faute en silence.
//
// LE TÉMOIN EST LIVE FIRE, la seule carte dont la région jouée n'est pas la première du bloc
// structure-BSP (4 régions, arène en région 1, index sur DEUX bits) : c'est le seul endroit où
// catalogue et auto-détection divergent aujourd'hui (mesure du lot 1.9.2 : 15 films sur 17
// identiques, les 2 films Live Fire différents).
func TestMutationDuCatalogueChangeLeDecoupageDesPositions(t *testing.T) {
	cat := catalogueDeBornesVersionne(t)
	entry, err := cat.Lookup("Live Fire")
	if err != nil {
		t.Fatalf("entree Live Fire : %v", err)
	}
	reference := optionsPourEntree(entry)
	if reference.Layout == nil {
		t.Fatal("decoupage de reference nil : la mutation ne prouverait rien")
	}
	// LE DÉCOUPAGE ATTENDU EST FIGÉ ICI, et c'est ce qui fait rougir une faute DANS LE CATALOGUE
	// (et pas seulement dans le câblage) : sans cette ligne, décaler `axisWidths` d'un bit dans
	// `map_quant_bounds.json` laisserait ce test vert — mutation jouée le 2026-09-15. Jumeau au
	// plus près des bits : `filmdec/i0_catalogue_mutation_test.go`,
	// `decoupageDeReferenceLiveFire` ; les deux se mettent à jour ensemble le jour où le
	// découpage de cette carte change VOLONTAIREMENT.
	attendu := filmdec.I0Layout{GateBits: 6, AxisW: [3]uint{12, 12, 11}, Region: 1}
	if *reference.Layout != attendu {
		t.Fatalf("le catalogue donne %s a Live Fire, attendu %s (valeur du 2026-09-15). "+
			"Si le changement est voulu, reecrire cette reference et son jumeau `filmdec."+
			"decoupageDeReferenceLiveFire` dans le meme commit", *reference.Layout, attendu)
	}
	for _, cas := range []struct {
		nom   string
		muter func(*filmdec.MapQuantEntry)
	}{
		{"axisWidths X decale d un bit", func(m *filmdec.MapQuantEntry) { m.AxisWidths[0]++ }},
		{"axisWidths Z decale d un bit", func(m *filmdec.MapQuantEntry) { m.AxisWidths[2]-- }},
		{"regionIndexBits rabaissee a 1", func(m *filmdec.MapQuantEntry) { m.RegionIndexBits = 1 }},
		{"region attendue remise a 0", func(m *filmdec.MapQuantEntry) { m.Region = 0 }},
	} {
		mute := entry
		cas.muter(&mute)
		got := optionsPourEntree(mute)
		if got.Layout == nil {
			t.Errorf("%s : decoupage nil — le catalogue mute rend la main a l auto-detection", cas.nom)
			continue
		}
		if *got.Layout == *reference.Layout {
			t.Errorf("%s : decoupage INCHANGE (%s) — le balayage ne prend donc pas le sien au "+
				"catalogue", cas.nom, *got.Layout)
		}
	}
}
