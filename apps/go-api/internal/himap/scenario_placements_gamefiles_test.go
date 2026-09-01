package himap

// scenario_placements_gamefiles_test.go — LA CARTE DES BLOCS DU SCENARIO `levl`.
//
// # Pourquoi cet inventaire, et ce qu'il debloque
//
// Deux chantiers du 2026-08-31 butent sur la MEME porte : les VEHICULES des cartes officielles
// (mesure : 121 cartes, leurs `.mvar` ne portent que des tourelles — Fragmentation en a UNE) et
// les SITES D'AMORCAGE de l'Assaut (`defender_bombsite` / `attacker_bombsite`, nommes par le
// script du mode, absents des 224 `.mvar` du corpus). Les deux vivent dans le SCENARIO de la
// carte, pas dans sa variante.
//
// Le scenario est le tag `levl`, et il est deja lu en production — les zones nommees en sortent
// (`callouts.go` : noms a root+0x91C, volumes a root+0x3BC). Ce qui manque n'est pas l'acces,
// c'est la CARTE : quels autres blocs le root porte, combien d'elements chacun, et lesquels
// ressemblent a des placements d'objets.
//
// # Ce que l'inventaire rend, et pourquoi c'est suffisant pour decider
//
// La navigation de struct-table (`meilleurTagInfo`, `liensBlocs`, `compteChamp`) donne, pour
// chaque bloc enfant du root : son OFFSET DE CHAMP, son NOMBRE d'elements et sa TAILLE de
// record. Un bloc de placements se reconnait a trois traits, sans rien supposer du format :
//
//	COMPTE     de quelques unites a quelques centaines — un scenario ne pose pas 10 000 objets ;
//	STRIDE     assez large pour porter au moins une position (3 flottants) et une reference ;
//	CONTENU    des flottants dans l'emprise de la carte, que le dump hexa laisse voir.
//
// L'inventaire est imprime pour DEUX cartes de nature differente — une carte BTB a vehicules
// (Fragmentation) et une carte d'Assaut (Rat's Nest) — pour que la comparaison designe les
// blocs qui changent avec le contenu, et non ceux qui sont la partout.
//
// REGIME : garde `HALO_DEPLOY` (racine `deploy` du jeu). Le depot `ds/` est celui sur lequel la
// table des callouts a ete etablie : le build serveur dedie porte le scenario complet et pese
// 4,6 Mo — aucun risque memoire, contrairement aux modules `pc/` (586 Mo de tags anonymes).
//
//	$env:HALO_DEPLOY="D:/SteamLibrary/steamapps/common/Halo Infinite/deploy"
//	go test ./internal/himap/ -run ScenarioBlocs -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

// scCartes : les modules sondes, et ce qu'on attend d'y trouver.
var scCartes = []struct{ dossier, module, pourquoi string }{
	{"btb_fragmentation", "btb_fragmentation-rtx-new.module", "BTB a vehicules (Warthog, Wasp, Scorpion)"},
	{"rats_nest", "rats_nest-rtx-new.module", "carte d'Assaut (Neutral Bomb Squad)"},
	{"ctf_aquarius", "ctf_aquarius-rtx-new.module", "arene sans vehicule — TEMOIN"},
}

// TestScenarioBlocs imprime la carte des blocs du root du scenario.
func TestScenarioBlocs(t *testing.T) {
	deploy := os.Getenv("HALO_DEPLOY")
	if deploy == "" {
		t.Skip("mesure non demandee : HALO_DEPLOY requis")
	}
	for _, c := range scCartes {
		chemin := filepath.Join(deploy, "ds", "levels", "multi", c.dossier, c.module)
		if _, err := os.Stat(chemin); err != nil {
			t.Logf("=== %s : module absent (%s)", c.dossier, chemin)
			continue
		}
		t.Logf("=== %s — %s", c.dossier, c.pourquoi)
		scInventaire(t, chemin)
	}
}

// scInventaire imprime, pour un module, les blocs enfants du root du tag `levl`.
func scInventaire(t *testing.T, chemin string) {
	t.Helper()
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Logf("    ouverture impossible : %v", err)
		return
	}
	levls := m.Files("levl")
	if len(levls) != 1 {
		t.Logf("    %d tag(s) levl (attendu 1)", len(levls))
		return
	}
	tag, err := m.Extract(levls[0])
	if err != nil {
		t.Logf("    extraction impossible : %v", err)
		return
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		t.Logf("    en-tete illisible : %v", err)
		return
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		t.Logf("    root introuvable : %v", err)
		return
	}
	_, rootSize := ti.blockAbs(root)
	liens := liensBlocs(ti)
	type ligne struct {
		off, n, taille, cible int
	}
	var ls []ligne
	for _, l := range liens {
		if l.owner != root {
			continue
		}
		n := compteChamp(ti, l)
		if n <= 0 {
			continue
		}
		_, tailleCible := ti.blockAbs(l.target)
		stride := 0
		if n > 0 {
			stride = tailleCible / n
		}
		ls = append(ls, ligne{off: l.fieldOff, n: n, taille: stride, cible: l.target})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].off < ls[j].off })
	t.Logf("    tag levl : %d octets, root de %d octets, %d bloc(s) enfant non vide", len(tag), rootSize, len(ls))
	for _, l := range ls {
		marque := ""
		switch l.off {
		case levlChampNames:
			marque = "  <- zones nommees (connu)"
		case levlChampVolumes:
			marque = "  <- volumes (connu)"
		}
		t.Logf("      champ 0x%04x : %5d element(s) de %4d octets  (bloc %d)%s",
			l.off, l.n, l.taille, l.cible, marque)
	}
}

// TestScenarioBlocDump imprime les premiers octets d'un bloc designe, pour lire sa grammaire.
//
//	$env:HALO_DEPLOY="..." ; $env:SC_CARTE="btb_fragmentation" ; $env:SC_CHAMP="0x1234"
//	go test ./internal/himap/ -run ScenarioBlocDump -v
func TestScenarioBlocDump(t *testing.T) {
	deploy, carte, champ := os.Getenv("HALO_DEPLOY"), os.Getenv("SC_CARTE"), os.Getenv("SC_CHAMP")
	if deploy == "" || carte == "" || champ == "" {
		t.Skip("mesure non demandee : HALO_DEPLOY, SC_CARTE et SC_CHAMP requis")
	}
	var off int
	if _, err := fmt.Sscanf(champ, "0x%x", &off); err != nil {
		t.Fatalf("SC_CHAMP doit etre 0xNNNN : %v", err)
	}
	chemin := filepath.Join(deploy, "ds", "levels", "multi", carte, carte+"-rtx-new.module")
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatalf("ouverture : %v", err)
	}
	tag, err := m.Extract(m.Files("levl")[0])
	if err != nil {
		t.Fatalf("extraction : %v", err)
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	root, _ := ti.rootBlockIndex()
	for _, l := range liensBlocs(ti) {
		if l.owner != root || l.fieldOff != off {
			continue
		}
		n := compteChamp(ti, l)
		abs, taille := ti.blockAbs(l.target)
		stride := 0
		if n > 0 {
			stride = taille / n
		}
		t.Logf("champ 0x%04x : %d element(s) de %d octets", off, n, stride)
		for i := 0; i < n && i < 4; i++ {
			t.Logf("  --- element %d", i)
			deb := abs + i*stride
			for j := 0; j < stride; j += 16 {
				fin := j + 16
				if fin > stride {
					fin = stride
				}
				t.Logf("    %04x  % x", j, ti.tag[deb+j:deb+fin])
			}
		}
		return
	}
	t.Fatalf("aucun bloc enfant du root au champ 0x%04x", off)
}

// TestScenarioPositions — LE DETECTEUR DE PLACEMENTS, sans rien supposer du format.
//
// Un bloc de placements porte forcement une POSITION MONDE. On ne cherche donc pas un nom de
// champ ni une signature : on cherche, dans chaque bloc et a chaque offset aligne, un triplet de
// flottants qui tombe DANS L'EMPRISE DU BSP pour la grande majorite des records. L'emprise vient
// du module lui-meme (`ReadModuleBSPBounds`), pas d'un catalogue — la sonde est autonome.
//
// LE CRITERE, ecrit avant la mesure : un offset est retenu si au moins 70 %% des records y
// portent un triplet dans l'emprise ELARGIE de 10 %%, et si le bloc compte au moins 3 records.
// Un faux positif est possible (des flottants quelconques peuvent tomber dans la boite) ; c'est
// pour ca que la sortie imprime AUSSI l'etendue des valeurs — un vrai champ de position couvre
// la carte, un artefact se serre sur quelques valeurs.
func TestScenarioPositions(t *testing.T) {
	deploy := os.Getenv("HALO_DEPLOY")
	if deploy == "" {
		t.Skip("mesure non demandee : HALO_DEPLOY requis")
	}
	for _, c := range scCartes {
		chemin := filepath.Join(deploy, "ds", "levels", "multi", c.dossier, c.module)
		if _, err := os.Stat(chemin); err != nil {
			continue
		}
		t.Logf("=== %s — %s", c.dossier, c.pourquoi)
		scPositions(t, chemin)
	}
}

func scPositions(t *testing.T, chemin string) {
	t.Helper()
	bsps, err := ReadModuleBSPBounds(chemin)
	if err != nil || len(bsps) == 0 {
		t.Logf("    emprise BSP illisible : %v", err)
		return
	}
	b := bsps[0].Bounds
	mx, my, mz := b.Max[0]-b.Min[0], b.Max[1]-b.Min[1], b.Max[2]-b.Min[2]
	lo := [3]float64{b.Min[0] - 0.1*mx, b.Min[1] - 0.1*my, b.Min[2] - 0.1*mz}
	hi := [3]float64{b.Max[0] + 0.1*mx, b.Max[1] + 0.1*my, b.Max[2] + 0.1*mz}
	t.Logf("    emprise BSP : [%.0f %.0f %.0f] .. [%.0f %.0f %.0f]",
		b.Min[0], b.Min[1], b.Min[2], b.Max[0], b.Max[1], b.Max[2])

	m, err := himodule.Open(chemin)
	if err != nil {
		return
	}
	tag, err := m.Extract(m.Files("levl")[0])
	if err != nil {
		return
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		return
	}
	for _, l := range liensBlocs(ti) {
		if l.owner != root {
			continue
		}
		n := compteChamp(ti, l)
		if n < 3 {
			continue
		}
		abs, taille := ti.blockAbs(l.target)
		stride := taille / n
		if stride < 12 {
			continue
		}
		for off := 0; off+12 <= stride; off += 4 {
			dedans, minv, maxv := 0, [3]float64{}, [3]float64{}
			premier := true
			for i := 0; i < n; i++ {
				p := abs + i*stride + off
				var v [3]float64
				ok := true
				for k := 0; k < 3; k++ {
					f := float64(f32(ti.tag, p+4*k))
					if f != f || f < lo[k] || f > hi[k] {
						ok = false
						break
					}
					v[k] = f
				}
				if !ok {
					continue
				}
				dedans++
				for k := 0; k < 3; k++ {
					if premier || v[k] < minv[k] {
						minv[k] = v[k]
					}
					if premier || v[k] > maxv[k] {
						maxv[k] = v[k]
					}
				}
				premier = false
			}
			if 100*dedans < 70*n {
				continue
			}
			// L'ETENDUE distingue un vrai champ de position d'un artefact : un placement
			// couvre la carte, un artefact se serre.
			couv := 0.0
			for k := 0; k < 3; k++ {
				if e := hi[k] - lo[k]; e > 0 {
					couv += (maxv[k] - minv[k]) / e
				}
			}
			if couv/3 < 0.20 {
				continue
			}
			t.Logf("      champ 0x%04x off +0x%02x : %d/%d record(s) dans l'emprise, "+
				"etendue [%.0f..%.0f  %.0f..%.0f  %.0f..%.0f] couverture %.0f %%",
				l.fieldOff, off, dedans, n, minv[0], maxv[0], minv[1], maxv[1],
				minv[2], maxv[2], 100*couv/3)
		}
	}
}

// TestScenarioRefsGroupes — CE QUE CHAQUE BLOC DE PLACEMENT REFERENCE, par son fourCC.
//
// # La grammaire d'un record de placement, lue le 2026-08-31 sur `btb_fragmentation`
//
//	+0x00  8 o   identifiant unique du placement
//	+0x0c  3f    POSITION monde
//	+0x18  3f    ORIENTATION
//	+0x60  u32   GlobalID du tag reference
//	+0x68  u32   GlobalID (second)
//	+0x6c  4 o   fourCC du GROUPE de tag, ECRIT A L'ENVERS ("snel" = `lens`)
//
// Le fourCC inverse est la cle : il DIT ce que le bloc pose, sans qu'on ait a le deviner. Ce
// balayage cherche donc, dans chaque bloc du root et a chaque offset aligne, une suite de
// quatre octets imprimables qui, RENVERSEE, nomme un groupe de tag PRESENT dans le module. Le
// groupe majoritaire d'un bloc est ce que ce bloc place.
//
// C'est ainsi qu'on trouvera `vehi` (les vehicules des cartes officielles) sans supposer
// l'ordre historique des blocs de scenario Halo.
func TestScenarioRefsGroupes(t *testing.T) {
	deploy := os.Getenv("HALO_DEPLOY")
	if deploy == "" {
		t.Skip("mesure non demandee : HALO_DEPLOY requis")
	}
	for _, c := range scCartes {
		chemin := filepath.Join(deploy, "ds", "levels", "multi", c.dossier, c.module)
		if _, err := os.Stat(chemin); err != nil {
			continue
		}
		t.Logf("=== %s — %s", c.dossier, c.pourquoi)
		scRefs(t, chemin)
	}
}

func scRefs(t *testing.T, chemin string) {
	t.Helper()
	m, err := himodule.Open(chemin)
	if err != nil {
		return
	}
	// Les groupes REELLEMENT presents dans ce module : le dictionnaire du renversement.
	presents := map[string]bool{}
	for _, f := range m.Files("") {
		if f.Group != "" {
			presents[f.Group] = true
		}
	}
	tag, err := m.Extract(m.Files("levl")[0])
	if err != nil {
		return
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		return
	}
	type res struct {
		off    int
		n      int
		stride int
		par    map[string]int
	}
	var out []res
	for _, l := range liensBlocs(ti) {
		if l.owner != root {
			continue
		}
		n := compteChamp(ti, l)
		if n < 1 {
			continue
		}
		abs, taille := ti.blockAbs(l.target)
		stride := taille / n
		if stride < 8 {
			continue
		}
		par := map[string]int{}
		for i := 0; i < n; i++ {
			deb := abs + i*stride
			for o := 0; o+4 <= stride; o += 4 {
				g := scFourCCInverse(ti.tag[deb+o : deb+o+4])
				if g != "" && (presents[g] || scGroupePlausible(g)) {
					par[g]++
				}
			}
		}
		if len(par) == 0 {
			continue
		}
		out = append(out, res{off: l.fieldOff, n: n, stride: stride, par: par})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].off < out[j].off })
	for _, r := range out {
		groupes := make([]string, 0, len(r.par))
		for g := range r.par {
			groupes = append(groupes, g)
		}
		sort.Slice(groupes, func(i, j int) bool { return r.par[groupes[i]] > r.par[groupes[j]] })
		var s []string
		for i, g := range groupes {
			if i >= 5 {
				break
			}
			s = append(s, fmt.Sprintf("%q x%d", g, r.par[g]))
		}
		t.Logf("      champ 0x%04x : %5d record(s) de %4d o  ->  %v", r.off, r.n, r.stride, s)
	}
}

// scFourCCInverse rend le groupe de tag qu'une suite de 4 octets nomme A L'ENVERS, ou "".
func scFourCCInverse(b []byte) string {
	for _, c := range b {
		if c < 0x20 || c >= 0x7f {
			return ""
		}
	}
	return string([]byte{b[3], b[2], b[1], b[0]})
}

// scGroupePlausible dit si un fourCC ressemble a un groupe de tag Halo : quatre caracteres
// minuscules, chiffres, `!`, `*` ou espace final. Le filtre « present dans CE module » ne suffit
// pas — un scenario reference des tags qui vivent dans les modules PARTAGES (`vehi`, `weap`), pas
// dans le sien : le module `ds/` de Fragmentation ne declare que 7 groupes.
func scGroupePlausible(g string) bool {
	lettres := 0
	for i, c := range []byte(g) {
		switch {
		case c >= 'a' && c <= 'z':
			lettres++
		case c >= '0' && c <= '9', c == '!', c == '*':
		case c == ' ' && i == 3:
		default:
			return false
		}
	}
	return lettres >= 3
}
