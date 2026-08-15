package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// mappingsReels recopie, sous une racine de test, les DEUX tables du titre dont dépend la
// résolution des clés d'arme. Recopier les vraies plutôt qu'en inventer : un test qui écrit
// ses propres mappings vérifie son fixture, pas le catalogue livré.
func mappingsReels(t *testing.T, root, titleSlug string) {
	t.Helper()
	src := title.NewPathResolver(depotRacine(t)).TitleMappingsDir(titleSlug)
	dst := title.NewPathResolver(root).TitleMappingsDir(titleSlug)
	for _, name := range []string{"weapon_names.toml", "replay_labels.toml"} {
		body, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Fatalf("lecture du mapping réel %s : %v", name, err)
		}
		ecrire(t, filepath.Join(dst, name), string(body))
	}
}

// depotRacine remonte jusqu'au répertoire qui porte config/titles (le dépôt lui-même).
func depotRacine(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("racine du dépôt introuvable")
	return ""
}

// artefactArmes pose un artefact minimal dont la table d'armes porte les trois cas qui
// comptent : une famille seule, l'identifiant global 64 bits de la même arme, et une
// famille absente du registre.
func artefactArmes(t *testing.T, root, titleSlug, matchID string) {
	t.Helper()
	doc := `{"schemaVersion":6,"matchId":"` + matchID + `","titleSlug":"` + titleSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"weaponLabels":{` +
		`"0x48C19D2D":{"en":"MA40 AR","fr":"MA40 AR","fx":"ballistic"},` +
		`"0x48C19D2D42C9679F":{"en":"MA40 AR","fr":"MA40 AR","fx":"ballistic"},` +
		`"0xDEADBEEF":{"en":"?","fr":"?"}}}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(titleSlug, matchID), doc)
}

// TestWeaponLabels_PoseLaTeinteDeLaDecharge : la teinte suit la clé, et elle vient de la
// table du TITRE — jamais d'une couleur écrite côté serveur.
func TestWeaponLabels_PoseLaTeinteDeLaDecharge(t *testing.T) {
	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	artefactArmes(t, root, title.DefaultSlug, "match-armes")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-armes")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if got := doc.WeaponLabels["0x48C19D2D42C9679F"].Tint; got != "kinetic" {
		t.Errorf("teinte du MA40 = %q, attendu kinetic", got)
	}
	if got := doc.WeaponLabels["0xDEADBEEF"].Tint; got != "" {
		t.Errorf("teinte %q posée sur une famille hors registre — la teinte neutre est la "+
			"seule réponse juste", got)
	}
}

// TestWeaponKeys_PoseLaCleDuRegistre : la clé canonique arrive sur les DEUX formes
// d'identifiant, parce que le document emploie les deux (famille pour un loadout,
// identifiant global pour un tir) et que la banque de sons du client n'en connaît qu'une.
func TestWeaponKeys_PoseLaCleDuRegistre(t *testing.T) {
	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	artefactArmes(t, root, title.DefaultSlug, "match-armes")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-armes")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	for _, id := range []string{"0x48C19D2D", "0x48C19D2D42C9679F"} {
		if got := doc.WeaponLabels[id].Key; got != "hinf_ma40_ar" {
			t.Errorf("clé de %s = %q, attendu hinf_ma40_ar — sans elle le tir reste muet", id, got)
		}
	}
}

// TestWeaponKeys_ArmeHorsRegistreResteMuette : aucune clé empruntée à une famille voisine.
// C'est la même règle que les libellés et les effets — un son approchant se prend pour une
// certitude, exactement comme un mot faux.
func TestWeaponKeys_ArmeHorsRegistreResteMuette(t *testing.T) {
	root := t.TempDir()
	mappingsReels(t, root, title.DefaultSlug)
	artefactArmes(t, root, title.DefaultSlug, "match-armes")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-armes")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if got := doc.WeaponLabels["0xDEADBEEF"].Key; got != "" {
		t.Errorf("clé %q posée sur une famille hors registre — le silence est la seule "+
			"réponse juste", got)
	}
}

// TestWeaponKeys_TitreSansCatalogue : un titre sans mappings sert le rejeu ENTIER, sans
// clés. L'absence de catalogue n'est pas une erreur de lecture du document.
func TestWeaponKeys_TitreSansCatalogue(t *testing.T) {
	root := t.TempDir()
	artefactArmes(t, root, title.DefaultSlug, "match-armes")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-armes")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(doc.WeaponLabels) != 3 {
		t.Fatalf("%d libellés d'arme servis, attendu 3 — le document doit rester entier",
			len(doc.WeaponLabels))
	}
	for id, lbl := range doc.WeaponLabels {
		if lbl.Key != "" {
			t.Errorf("clé %q posée sur %s sans catalogue de titre", lbl.Key, id)
		}
	}
}
