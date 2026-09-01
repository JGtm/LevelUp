package replay

// origine_elargi_research_test.go — LE TEST QUI TRANCHE : le seau ABSTENTION s'effondre-t-il
// quand on ajoute les points d'apparition d'ÉQUIPEMENT au catalogue ?
//
// ## CE QUI A ÉTÉ TROUVÉ DANS LES `.mvar` RE-TÉLÉCHARGÉS
//
// Les variantes de carte ont été récupérées de l'API UGC (`cmd/mapobj-build --save-mvar`,
// 3 appels) puis vidées objet par objet (`--dump-objects`). Confrontés aux 12 positions
// récurrentes publiées par le lot précédent, DEUX types hors liste blanche tombent PILE dessus,
// à 0,00-0,01 m — la même résolution que l'étalon de production (médiane 0,01 m) :
//
//	0xADEEE6D8  — 4 objets sur Catalyst, 5 sur Cliffhanger
//	0xE42158DF  — 4 objets sur Catalyst, 4 sur Cliffhanger
//
// Leur CARDINALITÉ est celle d'un socle (quelques unités par carte), pas d'un décor. Un
// troisième type, `0xA495FE83`, tombait à 0,51 m d'une position — mais il compte 95 à 100
// exemplaires par carte : c'est du décor, et sa proximité est fortuite. Il est ÉCARTÉ, et le
// dire est le point : trois chiffres suffisent à séparer un socle d'un pavé.
//
// ## CE QUE CE FICHIER FAIT, ET CE QU'IL NE FAIT PAS
//
// Il élargit le catalogue EN MÉMOIRE, pour la mesure seule. Il ne régénère PAS
// `map_weapon_pads.json`, il ne touche PAS `mapvar.PadFamilyOf`. Si le verdict est bon, la
// publication sera un lot séparé — c'est la consigne, et c'est aussi la prudence : élargir une
// liste blanche de production sur la foi d'une mesure à deux cartes serait prématuré.
//
// ## SEUILS ÉCRITS AVANT LA MESURE
//
//	X1 — sur la carte où l'étalon TIENT (Catalyst), le seau SOCLE doit AU MOINS DOUBLER et le
//	     seau ABSTENTION reculer d'autant. Sinon les deux types ne sont pas des points
//	     d'apparition.
//	X2 — les mêmes témoins spatiaux (+10 m en x, −7 m en y) doivent rester au plancher. Un
//	     catalogue élargi qui ferait aussi monter le témoin ne prouverait rien.
//	X3 — sur Cliffhanger (Super Fiesta), aucune promesse : ce mode n'allume aucun socle d'arme,
//	     et ses 7 grappes n'ont PAS trouvé d'objet de carte. Le résultat y est publié tel quel.
//
// Gardes PICKUP_FILM + PICKUP_MAP + ORIGINE_MAPID + ORIGINE_DUMP (le JSON de `--dump-objects`).

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// oriTypesEquipement : les deux types trouvés aux positions récurrentes. Publiés en hexadécimal
// comme le catalogue les écrit.
// LA LISTE N'EST PLUS ÉCRITE À LA MAIN : elle vient de la RECETTE (`ORIGINE_TYPES`, produit
// par `himap.TestOrigineRecetteBalayeLeCatalogueForge`). Le repli sur les deux identifiants
// mesurés reste, pour que l'instrument tourne sans la recette — mais c'est un repli, et il
// est marqué comme tel dans le journal du test.
var oriTypesEquipementRepli = map[int32]bool{
	-1376852264: true, // 0xADEEE6D8
	-467576609:  true, // 0xE42158DF
}

// oriTypesDeLaRecette charge la liste produite par la recette, ou rend le repli.
func oriTypesDeLaRecette(t *testing.T) (map[int32]bool, string) {
	path := os.Getenv("ORIGINE_TYPES")
	if path == "" {
		return oriTypesEquipementRepli, "REPLI (2 identifiants mesurés à la main)"
	}
	blob, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'opérateur de la mesure
	if err != nil {
		t.Fatalf("liste de types illisible : %v", err)
	}
	var ids []uint32
	if err := json.Unmarshal(blob, &ids); err != nil {
		t.Fatalf("liste de types : %v", err)
	}
	out := make(map[int32]bool, len(ids))
	for _, id := range ids {
		out[int32(id)] = true
	}
	return out, "RECETTE (" + fmt.Sprint(len(ids)) + " types du catalogue Forge)"
}

// oriObjetMvar est la forme du dump `--dump-objects`.
type oriObjetMvar struct {
	TypeID int32 `json:"type_id"`
	// UN TAG PAR CHAMP : `json:"x,y,z"` sur une declaration groupee donne le tag "x" aux TROIS,
	// et Y comme Z se seraient lus a ZERO en silence. `go vet` l a attrape.
	Pos struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"pos"`
}

// oriPadsElargis rend les emplacements du catalogue PLUS ceux des deux types d'équipement lus
// dans le dump de la variante.
func oriPadsElargis(t *testing.T, base []MapWeaponPadSpot) ([]MapWeaponPadSpot, int) {
	t.Helper()
	types, source := oriTypesDeLaRecette(t)
	t.Logf("SOURCE DES TYPES : %s", source)
	path := os.Getenv("ORIGINE_DUMP")
	if path == "" {
		t.Skip("ORIGINE_DUMP absent : instrument de mesure sauté")
	}
	blob, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'opérateur de la mesure
	if err != nil {
		t.Fatalf("dump d'objets illisible : %v", err)
	}
	var objs []oriObjetMvar
	if err := json.Unmarshal(blob, &objs); err != nil {
		t.Fatalf("dump d'objets : %v", err)
	}
	out := append([]MapWeaponPadSpot(nil), base...)
	ajouts := 0
	for _, o := range objs {
		if !types[o.TypeID] {
			continue
		}
		var p MapWeaponPadSpot
		p.Pos.X, p.Pos.Y, p.Pos.Z = o.Pos.X, o.Pos.Y, o.Pos.Z
		out = append(out, p)
		ajouts++
	}
	t.Logf("CATALOGUE ÉLARGI : %d emplacements d'origine + %d points d'équipement (%d objets au dump)",
		len(base), ajouts, len(objs))
	return out, ajouts
}

// oriClasse compte les trois seaux pour un jeu d'emplacements donné.
func oriClasse(pads []MapWeaponPadSpot, cre []filmdec.EquipmentCreation, ends []equipLife,
	dx, dy float64) oriBucket {
	var b oriBucket
	for _, c := range cre {
		switch {
		case oriNearest(pads, c.X, c.Y, c.Z, dx, dy) <= oriEps:
			b.socle++
		case oriNearDeath(ends, c.X, c.Y, c.Z, c.TimestampUS):
			b.sol++
		default:
			b.abstention++
		}
	}
	return b
}

func TestOrigineAvecCatalogueElargi(t *testing.T) {
	s := glResolve(t)
	base := oriPads(t)
	pads, ajouts := oriPadsElargis(t, base)
	_, pst := decodeFilmPlacements(s.dir, &s.wr)
	pu := decodeFilmPadScans(s.dir, &s.wr, pst.Calibration.Widths).Powerups
	if !pu.Scanned || len(pu.Creations) == 0 {
		t.Fatalf("voie des power-ups muette : scanned=%v creations=%d", pu.Scanned, len(pu.Creations))
	}
	ends := oriLifeEnds(oriFlat(s.pos))
	n := len(pu.Creations)

	avant := oriClasse(base, pu.Creations, ends, 0, 0)
	apres := oriClasse(pads, pu.Creations, ends, 0, 0)
	tx := oriClasse(pads, pu.Creations, ends, 10, 0)
	ty := oriClasse(pads, pu.Creations, ends, 0, -7)

	t.Logf("== ORIGINE ti=37 — CATALOGUE D'ORIGINE vs ÉLARGI · %s ==", s.dir)
	t.Logf("naissances : %d · points d'équipement ajoutés : %d", n, ajouts)
	t.Logf("            %-12s %-12s %-12s", "SOCLE", "SOL", "ABSTENTION")
	t.Logf("AVANT       %-12s %-12s %-12s",
		oriPct(avant.socle, n), oriPct(avant.sol, n), oriPct(avant.abstention, n))
	t.Logf("APRÈS       %-12s %-12s %-12s",
		oriPct(apres.socle, n), oriPct(apres.sol, n), oriPct(apres.abstention, n))
	t.Logf("TÉMOIN +10x %-12s %-12s %-12s",
		oriPct(tx.socle, n), oriPct(tx.sol, n), oriPct(tx.abstention, n))
	t.Logf("TÉMOIN -7y  %-12s %-12s %-12s",
		oriPct(ty.socle, n), oriPct(ty.sol, n), oriPct(ty.abstention, n))
	t.Logf("VERDICT X1 (le seau SOCLE au moins double) : %v — %d -> %d",
		apres.socle >= 2*avant.socle && apres.socle > avant.socle, avant.socle, apres.socle)
	t.Logf("VERDICT X2 (témoins au plancher) : %v — %d et %d contre %d",
		tx.socle*3 <= apres.socle && ty.socle*3 <= apres.socle, tx.socle, ty.socle, apres.socle)
}

// oriPct formate « n (x,y %) » pour les tableaux ci-dessus.
func oriPct(n, d int) string {
	return fmt.Sprintf("%d (%.1f %%)", n, pct100(n, d))
}
