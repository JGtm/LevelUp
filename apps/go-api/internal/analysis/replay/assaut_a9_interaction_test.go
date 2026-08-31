package replay

// assaut_a9_interaction_test.go — L'ARMEMENT EST UNE INTERACTION TENUE : ou le film la porte-t-il ?
//
// # La derniere piste, et d'ou elle vient
//
// Le script du mode dit que poser la bombe est une INTERACTION TENUE :
// `primitive_carriable_arming_base` porte `Device_GetInteractionHoldTime`,
// `deactivationBaseInteractTimeSec`, et la machine `GotoArming` / `GotoArmed` / `GotoDisarming`.
// La JAUGE, elle, est CALCULEE par le client (`armProgressFunction` est une FONCTION, et le
// canal de jauge du film — `ti=13` tag 3 — ne porte rien en Assaut : 0 a 2 valeurs contre 4 397
// chez un temoin Strongholds).
//
// Il ne manque donc qu'une chose : L'INSTANT OU L'INTERACTION COMMENCE. Trois canaux ont ete
// fouilles et temoines sans le trouver (composants du statborg, pied de film, `ti=13`). Restent
// les COMPOSANTS ECS eux-memes — et le film porte son propre DICTIONNAIRE : le registre
// (`chunk_00`) nomme chaque archetype et chacun de ses composants, en clair.
//
// Cet instrument imprime ce dictionnaire et y cherche le vocabulaire de l'interaction. C'est la
// question posee au film dans SA langue, avant toute mesure de bits.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautA9 -v -timeout 20m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// a9Motifs : le vocabulaire cherche dans les noms de composant du registre.
var a9Motifs = []string{
	"interact", "device", "arm", "plant", "bomb", "hold", "progress",
	"carriable", "carry", "timer", "countdown", "detonat", "defus", "disarm",
}

// TestAssautA9Dictionnaire imprime les composants du registre ECS dont le nom porte l'un des
// motifs, archetype par archetype.
func TestAssautA9Dictionnaire(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	dir := filepath.Join(cache, "film_chunks", "9f57c612")
	raw, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre invalide : %v", err)
	}
	total, retenus := 0, 0
	for ti := 0; ; ti++ {
		arch, ok := reg.Archetype(ti)
		if !ok {
			break
		}
		var lignes []string
		for i, nom := range arch.Components {
			total++
			bas := strings.ToLower(nom)
			for _, m := range a9Motifs {
				if strings.Contains(bas, m) {
					lignes = append(lignes, fmt.Sprintf("        i%d %s", i, nom))
					retenus++
					break
				}
			}
		}
		if len(lignes) == 0 {
			continue
		}
		t.Logf("    ti=%d — %d composant(s), dont :", ti, len(arch.Components))
		for _, l := range lignes {
			t.Log(l)
		}
	}
	t.Logf("REGISTRE : %d composants au total, %d portent le vocabulaire de l'interaction", total, retenus)
}

// TestAssautA9Archetypes imprime la taille de chaque archetype — la carte du registre.
func TestAssautA9Archetypes(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	raw, err := filmdec.ReadFilmChunk(filepath.Join(cache, "film_chunks", "9f57c612"), 0)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre invalide : %v", err)
	}
	type l struct {
		ti, n   int
		premier string
	}
	var ls []l
	for ti := 0; ; ti++ {
		arch, ok := reg.Archetype(ti)
		if !ok {
			break
		}
		if len(arch.Components) == 0 {
			continue
		}
		ls = append(ls, l{ti, len(arch.Components), arch.Components[0]})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].ti < ls[j].ti })
	for _, x := range ls {
		t.Logf("    ti=%-3d %3d composant(s)  i0=%s", x.ti, x.n, x.premier)
	}
}

// TestAssautA9Ti11Present — `ti=11` VIT-IL DANS LES FILMS D'ASSAUT ?
//
// Le registre nomme, a l'archetype 11, exactement ce qu'il faut :
//
//	i0   managed-objective-timers-component
//	i4   managed-objective-interaction-filter-component
//	i12  managed-objective-progress-component            <- LA JAUGE
//	i13  managed-objective-required-progress-component   <- LE SEUIL
//
// Avant d'engager le portage des deserialiseurs (`ti=11` est couvert 0/34 par le dispatch —
// `components_batch3.go` n'en porte que deux, sans appelant), il faut savoir si l'archetype
// EMET quelque chose en Assaut. Ce recensement ne decode aucun corps de record : il ne lit que
// les EN-TETES d'image-cle, ou le `typeIndex` tient sur 6 bits. C'est donc gratuit, et c'est le
// bon ordre — on ne porte pas treize deserialiseurs pour un archetype absent.
//
// TEMOINS : Strongholds et KOTH, ou la progression d'objectif EXISTE a l'ecran.
func TestAssautA9Ti11Present(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA9Ti11Present")()
	ligne := func(id, mode string) {
		dir := filepath.Join(cache, "film_chunks", id)
		k11 := filmdec.ScanFilmWorldObjectKeyframes(dir, 11)
		k13 := filmdec.ScanFilmWorldObjectKeyframes(dir, 13)
		k42 := filmdec.ScanFilmWorldObjectKeyframes(dir, 42)
		t.Logf("%-9s %-24s ti=11 : %3d slot(s), %4d vie(s)   |   ti=13 : %3d slot(s)   |   ti=42 : %4d slot(s)",
			id, mode, len(k11.Band), len(k11.SeenUS), len(k13.Band), len(k42.Band))
	}
	ligne("2ce58582", "Ranked:Strongholds")
	ligne("696a9d7c", "Strongholds:Arena")
	ligne("7f1bbf06", "KOTH:Arena")
	ligne("cde26226", "CTF:Arena")
	for _, id := range []string{"9f57c612", "c75f33b8", "df8fcbef", "34bb3bc8", "1c01e34f"} {
		ligne(id, "Assaut")
	}
}

// TestAssautA9GrammaireObjectifs — LA GRAMMAIRE COMPLETE DES ARCHETYPES D'OBJECTIF.
//
// Le registre ECS du film nomme chaque composant en clair. Cet instrument imprime, pour les
// archetypes qui portent la donnee d'objectif, la LISTE ORDONNEE de leurs composants — l'index
// `i` est le bit de masque, donc la cle par laquelle un deserialiseur se branche.
//
// Sert deux choses : le portage des deserialiseurs de `ti=11`, et la page de reference.
func TestAssautA9GrammaireObjectifs(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	raw, err := filmdec.ReadFilmChunk(filepath.Join(cache, "film_chunks", "9f57c612"), 0)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre invalide : %v", err)
	}
	for _, ti := range []int{10, 11, 12, 13} {
		arch, ok := reg.Archetype(ti)
		if !ok {
			continue
		}
		t.Logf("########## ti=%d — %d composants", ti, len(arch.Components))
		for i, nom := range arch.Components {
			t.Logf("i%-3d %s", i, nom)
		}
	}
}
