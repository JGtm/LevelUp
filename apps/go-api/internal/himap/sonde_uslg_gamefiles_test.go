//go:build gamefiles

package himap

// SONDE (2026-09-02) — OU VIT LE TEXTE JOUEUR D'UNE ZONE NOMMEE, ET POURQUOI IL MANQUE
// ENCORE POUR LES DEUX TIERS DES ZONES FORGE.
//
// LE CONTEXTE. Le catalogue de callouts joint ses libelles par `callouts_i18n.csv`, une
// extraction `uslg` faite UNE fois par la recherche. Ce CSV ne couvre que le vocabulaire des
// 22 cartes NATIVES (463 string_id). Les cartes Forge en emploient d'autres : sur la rotation
// du 2026-08-27, 66 des 266 string_id de lieu employes y trouvent un texte, soit un quart.
// Les 200 restants demandent de rejouer l'extraction — d'ou cette sonde.
//
// CE QUE LA SONDE ETABLIT, ET QUI FERME UNE PISTE :
//
//	1. le tag `locs` NE PORTE AUCUN TEXTE. Sa struct-table est mesuree : root block de
//	   64 octets, UN seul bloc enfant de 778 entrees de 4 octets (778 x 4 + 0x120 = 3 400 =
//	   la taille du tag). C'est un VOCABULAIRE de string_id, rien d'autre. Chercher les
//	   libelles la-dedans est sans objet.
//	2. le groupe `uslg` (488 tags, 577 Ko decompresses dans globals-rtx-new.module) ne porte
//	   pas non plus de texte DANS LE TAG : chaque tag uslg tient en 1 184 octets, un bloc de
//	   18 entrees de 20 octets — les 18 LANGUES, numerotees 0..17.
//	3. LE TEXTE EST DANS LE BLOB DE RESSOURCES du tag uslg (520 Ko pour le plus gros), et il
//	   est en ASCII lisible : « Pick up Blind Skull », « Out of Ammo », « Cindershot »... La
//	   table d'index vit vers 0x150 du blob, par paires de mots de 32 bits croissants.
//
// SUITE DONNEE, LE MEME JOUR : la table d'index est DECODEE et le chantier est clos. Le
// blob n'est pas une soupe d'octets mais 18 sous-fichiers `ucsh` CONCATENES (un par langue,
// dans l'ordre du bloc des 18) ; dans chacun, la table d'index est le TagBlock du champ 0
// de la racine — N paires { u32 string_id, u32 offset } — et le texte est le bloc de la
// premiere reference de donnee. Le decodeur de production vit dans uslg.go (format
// documente au champ pres) ; il rend 810 noms de lieu en 18 langues et reproduit les 463
// string_id de callouts_i18n.csv au caractere pres, EN et FR.
//
// Cette sonde reste au depot comme MESURE : c'est elle qui etablit ce que le tag ne porte
// PAS (le point 1 ferme la piste `locs`, le point 2 celle du texte dans le tag). Elle ne
// conclut pas au-dela : elle mesure et journalise. Aucun octet n'est ecrit.

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

// TestSondeGroupesDeTagsGlobals — quels groupes de tags porte le module globals, et lequel
// pese assez pour etre une table de chaines ? (mesure : `uslg`, 488 tags.)
func TestSondeGroupesDeTagsGlobals(t *testing.T) {
	p := moduleGlobals(t)
	m, err := himodule.Open(p)
	if err != nil {
		t.Fatalf("ouvrir %s : %v", p, err)
	}
	type stat struct {
		n     int
		bytes int64
	}
	par := map[string]*stat{}
	for _, f := range m.Files("") {
		s := par[f.Group]
		if s == nil {
			s = &stat{}
			par[f.Group] = s
		}
		s.n++
		s.bytes += int64(f.UncompSize)
	}
	groupes := make([]string, 0, len(par))
	for g := range par {
		groupes = append(groupes, g)
	}
	sort.Slice(groupes, func(i, j int) bool { return par[groupes[i]].bytes > par[groupes[j]].bytes })
	t.Logf("%s : %d groupes de tags", filepath.Base(p), len(groupes))
	for i, g := range groupes {
		if i >= 40 {
			t.Logf("  ... (%d groupes de plus)", len(groupes)-40)
			break
		}
		t.Logf("  %-6q %5d tags, %12d octets decompresses", g, par[g].n, par[g].bytes)
	}
}

// TestSondeUslgTexteDansLesRessources — LA MESURE DECISIVE : le tag uslg ne porte que ses
// 18 langues ; le texte est dans son blob de ressources.
func TestSondeUslgTexteDansLesRessources(t *testing.T) {
	m, err := himodule.Open(moduleGlobals(t))
	if err != nil {
		t.Fatalf("globals : %v", err)
	}
	fichiers := m.Files("uslg")
	t.Logf("%d tags uslg", len(fichiers))
	if len(fichiers) == 0 {
		t.Skip("aucun uslg")
	}
	sort.Slice(fichiers, func(i, j int) bool { return fichiers[i].UncompSize > fichiers[j].UncompSize })
	f := fichiers[0]
	brut, err := m.Extract(f)
	if err != nil {
		t.Fatalf("extraire uslg #%d : %v", f.Index, err)
	}
	t.Logf("uslg #%d : tag de %d octets", f.Index, len(brut))
	ti, terr := meilleurTagInfo(brut)
	if terr != nil {
		t.Fatalf("struct-table du tag uslg : %v", terr)
	}
	for _, l := range liensBlocs(ti) {
		_, taille := ti.blockAbs(l.target)
		n := compteChamp(ti, l)
		t.Logf("  bloc %d : %d octets, %d entrees (stride %d) — les langues", l.target, taille, n, taille/maxi(n, 1))
	}
	blob, berr := m.ResourceBlob(f)
	if berr != nil {
		t.Fatalf("blob de ressources : %v", berr)
	}
	t.Logf("  blob de ressources : %d octets", len(blob))
	lisibles := chainesLisibles(blob, 6, 12)
	if len(lisibles) == 0 {
		t.Fatal("aucune chaine lisible dans le blob : la piste du texte en ressource tombe")
	}
	for _, s := range lisibles {
		t.Logf("    txt %q", s)
	}
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
