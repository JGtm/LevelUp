package killicon

import (
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

// TestNeutralVignettesExistentSurDisque ferme le meme trou que pour les regles : le pont ne
// peut pas designer un fichier absent, sinon la CI est verte et l ecran est vide.
func TestNeutralVignettesExistentSurDisque(t *testing.T) {
	spr := NeutralSprites()
	if len(spr) == 0 {
		t.Fatal("aucune vignette de type de mort : le test ne prouve rien")
	}
	for kind, stem := range spr {
		if !strings.HasPrefix(stem, atlasPrefix) {
			t.Errorf("type %q -> %q : hors atlas %q", kind, stem, atlasPrefix)
		}
		if _, err := os.Stat(spritePath(stem)); err != nil {
			t.Errorf("type %q -> %q : PNG absent (%v)", kind, stem, err)
		}
	}
}

// TestNeutralDeathClasseVersType verrouille LA decision de neutral.go : c est la NATURE du
// degat qui choisit, et une nature non etablie ne choisit RIEN.
func TestNeutralDeathClasseVersType(t *testing.T) {
	// Les deux temoins REELS de la mesure du 2026-08-14, cites par leur tag : `00403594` est
	// le degat global (chute/environnement) porte par 4 des 5 morts mesurees, `d468178e` le
	// M41 SPNKr d une mort auto-infligee.
	cas := []struct {
		nom  string
		tag  uint32
		kind string
	}{
		{"degat global (chute, environnement)", 0x00403594, NeutralKindEnvironment},
		{"sa propre roquette", 0xd468178e, NeutralKindSuicide},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			lab, known := damagetag.Lookup(c.tag)
			if !known {
				t.Fatalf("temoin %08x absent du catalogue : la mesure ne peut plus etre rejouee", c.tag)
			}
			kind, icon, ok := NeutralDeath(c.tag)
			if !ok {
				t.Fatalf("temoin %08x (classe %s) : aucun type rendu", c.tag, lab.Class)
			}
			if kind != c.kind {
				t.Errorf("temoin %08x (classe %s) : type %q, attendu %q", c.tag, lab.Class, kind, c.kind)
			}
			if icon.Sprite != NeutralSprites()[c.kind] {
				t.Errorf("temoin %08x : vignette %q, attendu %q", c.tag, icon.Sprite,
					NeutralSprites()[c.kind])
			}
		})
	}
}

// TestNeutralDeathRefuseLInconnu : un tag hors catalogue ne rend AUCUN type. C est la regle
// dure du lot — jamais l icone d une autre mort — et elle doit tomber en rouge si elle bouge.
func TestNeutralDeathRefuseLInconnu(t *testing.T) {
	// 0 n est pas un `jpt!` du catalogue (le scan le rejette par construction, cf. damagetag).
	if _, _, ok := NeutralDeath(0); ok {
		t.Error("tag hors catalogue : un type a ete rendu, aucun n est admissible")
	}
	// Et toute entree de classe INCONNU du catalogue reel doit etre refusee de la meme facon.
	refusees := 0
	for _, l := range damagetag.Labels() {
		if l.Class != damagetag.ClassInconnu {
			continue
		}
		refusees++
		if _, _, ok := NeutralDeath(l.Tag); ok {
			t.Errorf("tag %08x de classe INCONNU : un type a ete rendu", l.Tag)
		}
	}
	t.Logf("entrees de classe INCONNU refusees : %d", refusees)
}
