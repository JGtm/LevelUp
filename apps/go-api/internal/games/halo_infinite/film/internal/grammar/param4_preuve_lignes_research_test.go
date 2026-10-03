//go:build research

package grammar

// param4_preuve_lignes_research_test.go — LOT 5.1.7-a : LA LIGNE QUI DISPARAIT, NOMMEE.
//
// # CE QU IL SERT A PROUVER
//
// L equivalence de rejeu fait bouger `abilityRanks` (103 -> 102) et `inventoryDeltas`
// (8914 -> 8915) sur `1c4c63c2`. UN COMPTE QUI DESCEND N EST UNE CORRECTION QUE SI L ON MONTRE
// LA LIGNE DISPARUE ET QU ON PROUVE QU ELLE ETAIT FAUSSE. Cet instrument rend les lignes, une
// par ligne de TSV, pour qu un `diff` entre deux cuissons (base et tete) nomme exactement
// laquelle apparait et laquelle disparait, avec son instant, son slot et ses valeurs.
//
// LES DEUX CRITERES DE REJET SONT DANS LE FLUX, PAS DANS UNE OPINION :
//
//	COMPTEUR  `Counter` est un `R(3)` : hors [0, 7] la lecture est mal placee (contrat ecrit
//	          chez `types.AbilityRank.Counter`).
//	RANG      `Rank` est un `R(6)` : hors [0, 63] il ne peut pas sortir du flux. Au-dela, le
//	          rang doit exister dans la PALETTE du match — un rang que la palette ne nomme pas
//	          est une lecture qui a rate son composant.
//
// LECTURE SEULE, N ASSERTE RIEN, UN SEUL FILM PAR INVOCATION, aucune base ouverte :
//
//	P4_FILM=<abs>/data/cache/film_chunks/1c4c63c2 P4_CARTE="<carte>" P4_OUT=<abs>/prefixe \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestParam4PreuveDesLignes$' -v -timeout 60m
//
// Il ecrit `<prefixe>.ranks.tsv` et `<prefixe>.inv.tsv`. Le meme fichier, joue sur l arbre de la
// REVISION DE BASE extrait par `git archive` hors depot, donne les deux cotes du diff.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestParam4PreuveDesLignes(t *testing.T) {
	dir, carte, sortie := os.Getenv("P4_FILM"), os.Getenv("P4_CARTE"), os.Getenv("P4_OUT")
	if dir == "" || carte == "" || sortie == "" {
		t.Skip("instrument de mesure : P4_FILM, P4_CARTE et P4_OUT requis")
	}
	fc := p4Contexte(t, dir, carte)
	p4EcrireRangs(t, fc, sortie+".ranks.tsv")
	p4EcrireInventaire(t, fc, sortie+".inv.tsv")
	p4EcrireMortsTi40(t, fc, sortie+".morts40.tsv")
}

// p4Contexte ouvre le contexte EXACTEMENT comme la cuisson : entree de catalogue de la carte
// pour le profil, largeurs d axe du catalogue sur le profil de balayage.
func p4Contexte(t *testing.T, dir, carte string) *FilmContext {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q : %v", carte, err)
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(bal)
	return fc
}

// p4EcrireRangs ecrit une ligne par identite de capacite, triee : le diff est alors stable.
func p4EcrireRangs(t *testing.T, fc *FilmContext, chemin string) {
	t.Helper()
	rangs, st, err := ScanAbilityRanks(fc)
	if err != nil {
		t.Fatalf("rangs de capacite : %v", err)
	}
	sort.SliceStable(rangs, func(i, j int) bool {
		if rangs[i].TimestampUS != rangs[j].TimestampUS {
			return rangs[i].TimestampUS < rangs[j].TimestampUS
		}
		return rangs[i].Slot < rangs[j].Slot
	})
	var b strings.Builder
	fmt.Fprintf(&b, "# rangs=%d records=%d avecI48=%d lues=%d nonLues=%d\n",
		len(rangs), st.Records, st.WithI48, st.Read, st.Unread)
	b.WriteString("tsUS\tslot\tchunk\tpaquet\tcompteur\trang\tcompteurHorsR3\trangHorsR6\n")
	for _, r := range rangs {
		fmt.Fprintf(&b, "%d\t%d\t%d\t%d\t%d\t%d\t%t\t%t\n", r.TimestampUS, r.Slot, r.Chunk,
			r.PacketIndex, r.Counter, r.Rank, r.Counter > 7, r.Rank < 0 || r.Rank > 63)
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", chemin, err)
	}
	t.Logf("RANGS : %d lignes -> %s (records=%d avecI48=%d lues=%d nonLues=%d)",
		len(rangs), chemin, st.Records, st.WithI48, st.Read, st.Unread)
}

// p4EcrireInventaire ecrit une ligne par lecture d inventaire, triee.
func p4EcrireInventaire(t *testing.T, fc *FilmContext, chemin string) {
	t.Helper()
	inv, st, err := ScanInventoryDeltas(fc)
	if err != nil {
		t.Fatalf("inventaire : %v", err)
	}
	sort.SliceStable(inv, func(i, j int) bool {
		if inv[i].TimestampUS != inv[j].TimestampUS {
			return inv[i].TimestampUS < inv[j].TimestampUS
		}
		return inv[i].Slot < inv[j].Slot
	})
	var b strings.Builder
	fmt.Fprintf(&b, "# lectures=%d records=%d\n", len(inv), st.Records)
	b.WriteString("tsUS\tslot\tchunk\tpaquet\tmasque\tsel\tselLu\tgrenades\tmunitions\n")
	for _, d := range inv {
		fmt.Fprintf(&b, "%d\t%d\t%d\t%d\t%d\t%d\t%t\t%v\t%d\n", d.TimestampUS, d.Slot, d.Chunk,
			d.PacketIndex, d.Mask, d.Sel, d.SelRead, d.Grenades, len(d.Ammo))
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", chemin, err)
	}
	t.Logf("INVENTAIRE : %d lignes -> %s (records=%d)", len(inv), chemin, st.Records)
}

// p4EcrireMortsTi40 ecrit une ligne par mort d entite `ti=40` LUE par la marche — c est la
// matiere de la preuve de contenu exigee sur `084a804d` : une mort qui disparait entre deux
// cuissons doit etre NOMMEE (instant, slot, generation) avant d etre appelee correction.
func p4EcrireMortsTi40(t *testing.T, fc *FilmContext, chemin string) {
	t.Helper()
	morts, st, err := ScanObjectDeaths(fc)
	if err != nil {
		t.Fatalf("morts ecrites : %v", err)
	}
	ti := uint32(VehicleTypeIndex)
	var b strings.Builder
	n := 0
	for _, m := range morts {
		if m.TypeIndex != ti {
			continue
		}
		n++
	}
	fmt.Fprintf(&b, "# mortsTi40=%d masqueDeclare=%d dontDesync=%d recordsAtteints=%d portes=%d\n",
		n, st.MaskDeclared[ti], st.MaskDeclaredDesync[ti], st.Records[ti], st.CleanRecords[ti])
	b.WriteString("tsUS\tslot\tgen\tqueueRompue\tenumA\tenumB\tv0c\tv0e\tv14\tv18\t" +
		"hasRef\tgidPresent\tglobalID\tsrcTag0\tsrcTag4c\thorsDomaine\n")
	for _, m := range morts {
		if m.TypeIndex != ti {
			continue
		}
		d := m.Dead
		fmt.Fprintf(&b, "%d\t%d\t%d\t%t\t%d\t%d\t%d\t%d\t%d\t%d\t%t\t%t\t%#x\t%#x\t%#x\t%s\n",
			m.TimestampUS, m.Slot, m.Gen, m.TailDesync, d.EnumA, d.EnumB, d.Val0c, d.Val0e,
			d.Val14, d.Val18, d.HasRef, d.GIDPresent, d.GlobalID, d.SrcTag0, d.SrcTag4c,
			p4HorsDomaine(d))
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", chemin, err)
	}
	t.Logf("MORTS ti=40 : %d lignes -> %s (masqueDeclare=%d dontDesync=%d records=%d portes=%d)",
		n, chemin, st.MaskDeclared[ti], st.MaskDeclaredDesync[ti], st.Records[ti], st.CleanRecords[ti])
}

// p4HorsDomaine nomme ce qui, dans une charge utile de dead-state, ne peut pas sortir du flux.
// Vide = rien a redire. Les criteres sont ceux du TYPE, ecrits avant la mesure :
//
//	poignee    `GlobalID`, `SrcTag0`, `SrcTag4c` : `0xFFFFFFFF` = ABSENT (contrat du champ) ;
//	           sinon une poignee d entite, qui n est jamais nulle.
//	enum       `EnumA` / `EnumB` sont des entiers signes de petite amplitude ; un negatif ou
//	           un au-dela de 64 dit que les bits ont ete pris ailleurs.
//	coherence  `GIDPresent` faux impose `GlobalID == 0xFFFFFFFF`, et reciproquement.
func p4HorsDomaine(d types.DeadState) string {
	var m []string
	nul := func(nom string, v uint32) {
		if v == 0 {
			m = append(m, nom+"=0")
		}
	}
	nul("globalID", d.GlobalID)
	nul("srcTag0", d.SrcTag0)
	nul("srcTag4c", d.SrcTag4c)
	if d.EnumA < 0 || d.EnumA > 64 {
		m = append(m, "enumA")
	}
	if d.EnumB < 0 || d.EnumB > 64 {
		m = append(m, "enumB")
	}
	if d.GIDPresent != (d.GlobalID != 0xFFFFFFFF) {
		m = append(m, "gidIncoherent")
	}
	if len(m) == 0 {
		return "-"
	}
	return strings.Join(m, ",")
}
