//go:build research

package grammar

// ri27d0_images_cles_research_test.go — LA MESURE DE 2.7.d0, volet des images-cles (plan de l etape 2
// de la representation intermediaire) : pour chaque record bipede (ti=35) des images-cles, ce que la
// grammaire lit de son etat complet — compteurs de grenades (i22), munitions (i30 a i41), emplacement
// desire (i42), identite des armes (i43 a i46), jeu de grenades (i47), capacite (i48), relus a
// l etendue de leur occurrence — contre ce que les fenetres de bits de la production lisent dans
// l emprise du meme record : armes portees (`familiesByRecordRecs`), inventaire
// (`keyframeInventoriesDe`), marque de portage (`carrierMarkViews`). Aucun fichier de production
// n est touche. La marche mesuree est celle de la production a la tete : l instrument n a qu un mode.
//
// Sortie (RI27C_OUT) : `images_cles.tsv`
//
//	S  film  classe  compte                 une ligne par classe et par film
//	D  film  classe  ts  slot  detail       des desaccords (au plus 30 par classe et par film)
//	F  film  composant  decalage  compte    ou tombent les familles que la fenetre trouve
//
// plus les lignes des complements de 2.7.d1 (A, T, M, MX, Y, R, MB et les agregats par format :
// `ri27d1_instrument_research_test.go`). RI27D1_RECORDS=<id,...> ecrit le dump R de ces films.
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> [RI27C_CARTES=<id=Carte;...>] \
//	  [RI27D1_RECORDS=<id,...>] go test -tags=research -count=1 -run '^TestRI27d0ImagesCles$' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27d0Gram est ce que la grammaire lit d un record bipede d image-cle.
type ri27d0Gram struct {
	// atteint : les index de composant traverses (interpretes) du record.
	atteint map[int]bool
	compte  uint64
	gren    []uint64
	magLu   [4]bool
	mag     [4]int // -1 : chargeur absent (porte a 1)
	resLu   [4]bool
	res     [4]int
	sel     int // i42, -1 non lu
	armeLue [4]bool
	idHigh  [4]uint32
	idLow   [4]uint32
	gsLu    bool
	gsMask  uint32
	gsSel   int
	rangLu  bool
	rang    int // -1 : aucun rang (porte a 1)
	// relu : lectures a l etendue dont la consommation depasse l occurrence (anomalie).
	debord int
}

// ri27d0Canal confronte, record par record, la grammaire et les fenetres.
type ri27d0Canal struct {
	fc      *FilmContext
	arch    Archetype
	court   string
	known   map[uint32]bool
	noms    map[uint32]string
	roles   map[int]string // index de composant -> role (« i22 », « am0 », « rs0 », « i42 », « ti0 »…)
	classes map[string]int
	ech     map[string]int
	lignes  []string
	fams    map[string]int
	// preuve : la preuve du record en cours, pour ventiler les classes.
	preuve string
	// x : les complements de 2.7.d1 (ri27d1_instrument_research_test.go).
	x *ri27d1Ext
}

func (c *ri27d0Canal) Interets() []Interet {
	vus := map[string]bool{}
	var out []Interet
	for id := range c.roles {
		n := c.arch.component(id)
		if !vus[n] {
			vus[n] = true
			out = append(out, Interet{Phase: PhaseImagesCles, TI: keyframeBipedTI, Composant: n})
		}
	}
	return out
}

func (*ri27d0Canal) Clore(BilanDeMarche) {}

// ri27d0Roles nomme les composants que la mesure relit.
func ri27d0Roles(arch Archetype) map[int]string {
	roles := map[int]string{}
	if i := archIndexOf(arch, invDeltaGrenadeCountsName); i >= 0 {
		roles[i] = "i22"
	}
	for k, id := range arch.indicesOf(invDeltaAmmoName) {
		roles[id] = fmt.Sprintf("am%d", k)
	}
	for k, id := range arch.indicesOf(invDeltaRoundsName) {
		roles[id] = fmt.Sprintf("rs%d", k)
	}
	for _, id := range arch.indicesOf("weapon-state-overheated") {
		roles[id] = "oh"
	}
	if i := archIndexOf(arch, "biped-desired-weapon-set"); i >= 0 {
		roles[i] = "i42"
	}
	for id, k := range weaponEmplacements(arch) {
		roles[id] = fmt.Sprintf("ti%d", k)
	}
	if i := archIndexOf(arch, invDeltaGrenadeSetName, invDeltaGrenadeSetAltName); i >= 0 {
		roles[i] = "i47"
	}
	if i := archIndexOf(arch, compBipedAbilitySet, compBipedAbilitySetAlt); i >= 0 {
		roles[i] = "i48"
	}
	return roles
}

// lire relit les composants interpretes du record `r` a l etendue de leur occurrence.
func (c *ri27d0Canal) lire(p *lecture.Paquet, r *lecture.Record, ctx ContexteDeLecture) ri27d0Gram {
	g := ri27d0Gram{atteint: map[int]bool{}, sel: -1, gsSel: -1, rang: -1}
	for k := range g.mag {
		g.mag[k], g.res[k] = -1, -1
	}
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		role, ok := c.roles[int(co.Index)]
		if !ok || co.Etat != lecture.EtatInterprete {
			continue
		}
		g.atteint[int(co.Index)] = true
		obs := &Observation{}
		var k int
		switch {
		case role == "i22":
			obs.GrenadeCountsHook = func(n uint64, v []uint64) { g.compte, g.gren = n, append([]uint64(nil), v...) }
		case strings.HasPrefix(role, "am"):
			fmt.Sscanf(role, "am%d", &k) //nolint:errcheck // role construit ici
			g.magLu[k] = true
			obs.WeaponAmmoHook = func(has bool, m uint32, _ bool, _ uint32) {
				if has {
					g.mag[k] = int(m)
				}
			}
		case strings.HasPrefix(role, "rs"):
			fmt.Sscanf(role, "rs%d", &k) //nolint:errcheck // role construit ici
			g.resLu[k] = true
			obs.WeaponRoundsHook = func(v uint32) { g.res[k] = int(v) }
		case role == "i42":
			obs.DesiredWeaponSetHook = func(s uint32) { g.sel = int(s) }
		case strings.HasPrefix(role, "ti"):
			fmt.Sscanf(role, "ti%d", &k) //nolint:errcheck // role construit ici
			g.armeLue[k] = true
			obs.HeldWeaponHook = func(h, l uint32) { g.idHigh[k], g.idLow[k] = h, l }
		case role == "i47":
			g.gsLu = true
			obs.GrenadeSetHook = func(m uint32, s int) { g.gsMask, g.gsSel = m, s }
		case role == "i48":
			g.rangLu = true
			obs.AbilitySetHook = func(_ uint64, rk int, _ int) { g.rang = rk }
		}
		br := sousLaPortee(LecteurSur(p.Payload)) // la marche d etat complet l a lu sous la portee
		br.PoserContexte(ctx)
		br.etatComplet = true
		br.obs = obs
		br.SetBitPos(int(co.Debut))
		_, _, _, porte := consumeByNameCapturing(br, c.arch.component(int(co.Index)), uint32(keyframeBipedTI), //nolint:gosec // archetype constant
			c.arch.Level(int(co.Index)))
		if !porte || br.BitPos()-int(co.Debut) > int(co.Bits) {
			g.debord++
		}
	}
	return g
}

// classer compte une classe et garde un echantillon.
func (c *ri27d0Canal) classer(classe string, p *lecture.Paquet, slot uint32, detail string) {
	c.classes[classe]++
	c.classes[classe+"|"+c.preuve]++
	c.x.parAdm[c.x.adm+"\t"+classe]++
	cle := classe + "|" + c.preuve
	if detail == "" || c.ech[cle] >= 30 {
		return
	}
	c.ech[cle]++
	c.lignes = append(c.lignes, fmt.Sprintf("D	%s	%s	%d	%d	%s", c.court, cle, p.TS, slot, detail))
}

// ImageCle confronte les deux lectures de chaque record bipede du paquet.
func (c *ri27d0Canal) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	ctx := c.fc.ContexteDeLecture()
	fen := map[uint32][]uint32{} // debut du record -> familles de la fenetre
	for _, rf := range familiesByRecordRecs(p.Payload, p.Records, c.known, keyframeBipedTI) {
		fen[rf.Rec.Debut] = rf.Families
	}
	invs := map[uint32]types.KeyframeInventory{}
	for _, inv := range keyframeInventoriesDe(p.Payload, invRecordSpansDe(p.Payload, p.Records), c.known, DefaultGrenadeMax) {
		invs[inv.Slot] = inv
	}
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != keyframeBipedTI {
			continue
		}
		c.classes["records_bipedes"]++
		if r.Desync == lecture.CorpsNonParcouru {
			c.classes["records_corps_non_parcouru"]++
			continue
		}
		fin := len(p.Payload) * 8
		if i+1 < len(p.Records) {
			fin = int(p.Records[i+1].Debut)
		}
		g := c.lire(p, r, ctx)
		if g.debord > 0 {
			c.classes["relecture_qui_deborde"] += g.debord
		}
		c.classes[fmt.Sprintf("desync_%d", r.Desync)]++
		c.preuve = fmt.Sprintf("preuve_%d", r.Preuve)
		c.classes[c.preuve]++
		c.x.admettre(c, r, &g)
		c.x.temoinDeHasard(c, p, i, ctx)
		c.comparerLesArmes(p, r, &g, fen[r.Debut])
		c.x.armesParEmplacement(c, &g, fen[r.Debut])
		inv, okInv := invs[r.Vie.Slot]
		c.comparerLInventaire(p, r, &g, inv, okInv)
		c.situerLesFenetres(p, r, fin)
		c.x.situerMarquesEtEnPlus(c, p, r, &g, fen[r.Debut], fin)
		c.x.dumpRecord(c, p, r, &g, fen[r.Debut])
	}
}

// nomsDe rend les noms canoniques (tries, sans doublon) d une suite de familles connues.
func (c *ri27d0Canal) nomsDe(fs []uint32) []string {
	vus := map[string]bool{}
	var out []string
	for _, f := range fs {
		if n, ok := c.noms[f]; ok && !vus[n] {
			vus[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func (c *ri27d0Canal) comparerLesArmes(p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram, fen []uint32) {
	var gf []uint32
	atteinte := false
	for k := range 4 {
		if !g.armeLue[k] {
			continue
		}
		atteinte = true
		for _, f := range []uint32{g.idHigh[k], g.idLow[k]} {
			if c.known[f] {
				gf = append(gf, f)
			}
		}
	}
	gn, fn := c.nomsDe(gf), c.nomsDe(fen)
	detail := fmt.Sprintf("grammaire=%v fenetre=%v ids=%08x/%08x %08x/%08x desync=%d", gn, fn,
		g.idHigh[0], g.idLow[0], g.idHigh[1], g.idLow[1], r.Desync)
	switch {
	case !atteinte && len(fn) == 0:
		c.classer("armes:non_atteintes_fenetre_vide", p, r.Vie.Slot, "")
	case !atteinte:
		c.classer("armes:non_atteintes_fenetre_lit", p, r.Vie.Slot, detail)
	case slices.Equal(gn, fn) && len(gn) == 0:
		c.classer("armes:aucune_des_deux", p, r.Vie.Slot, "")
	case slices.Equal(gn, fn):
		c.classer("armes:egales", p, r.Vie.Slot, "")
	case inclus(fn, gn):
		c.classer("armes:grammaire_en_plus", p, r.Vie.Slot, detail)
	case inclus(gn, fn):
		c.classer("armes:fenetre_en_plus", p, r.Vie.Slot, detail)
	default:
		c.classer("armes:differentes", p, r.Vie.Slot, detail)
	}
	if atteinte && !slices.Equal(c.idsTries(gf), c.idsTries(fen)) {
		c.classes["armes:identifiants_differents"]++
	}
}

func (c *ri27d0Canal) idsTries(fs []uint32) []uint32 {
	out := slices.Clone(fs)
	slices.Sort(out)
	return slices.Compact(out)
}

// inclus dit si a est inclus dans b (deux suites triees sans doublon).
func inclus(a, b []string) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}

func (c *ri27d0Canal) comparerLInventaire(p *lecture.Paquet, r *lecture.Record, g *ri27d0Gram,
	inv types.KeyframeInventory, okInv bool) {
	slot := r.Vie.Slot
	if !okInv {
		inv = types.KeyframeInventory{AbilityRank: -1, DrawnSlot: -1, SelectedGrenadeRank: -1}
	}
	// capacite : la fenetre ne voit que les rangs 16 a 23 ; un rang de la grammaire hors de ce
	// domaine n est pas comparable et se compte a part (correction 2.7.d1 (c)).
	if g.rangLu && g.rang >= 0 && (g.rang < 16 || g.rang > 23) {
		c.classer("capacite:hors_domaine_16_23", p, slot, "")
	} else {
		c.comparer("capacite", p, slot, g.rangLu, g.rang >= 0, g.rang, inv.AbilityRank >= 0, inv.AbilityRank)
	}
	// grenades
	gv := -1
	if g.compte == 4 && len(g.gren) == 4 {
		gv = int(g.gren[0]<<24 | g.gren[1]<<16 | g.gren[2]<<8 | g.gren[3])
	} else if g.atteint[archIndexOf(c.arch, invDeltaGrenadeCountsName)] {
		c.classer("grenades:compte_grammaire_different_de_4", p, slot, fmt.Sprintf("compte=%d", g.compte))
	}
	fv := -1
	if inv.GrenadesRead {
		fv = int(inv.Grenades[0]<<24 | inv.Grenades[1]<<16 | inv.Grenades[2]<<8 | inv.Grenades[3])
	}
	c.comparer("grenades", p, slot, gv >= 0 || g.atteint[archIndexOf(c.arch, invDeltaGrenadeCountsName)], gv >= 0, gv, fv >= 0, fv)
	// munitions des deux premiers emplacements
	for k := range 2 {
		fm, fr := -1, -1
		if inv.AmmoRead && inv.Ammo[k].Mag != nil {
			fm = int(*inv.Ammo[k].Mag)
		}
		if inv.AmmoRead && inv.Ammo[k].Res != nil {
			fr = int(*inv.Ammo[k].Res)
		}
		c.comparer(fmt.Sprintf("chargeur%d", k), p, slot, g.magLu[k], g.mag[k] >= 0, g.mag[k], fm >= 0, fm)
		c.comparer(fmt.Sprintf("reserve%d", k), p, slot, g.resLu[k], g.res[k] >= 0, g.res[k], fr >= 0, fr)
	}
	// degaine : NON COMPARABLE (correction 2.7.d1 (b)) — le crochet d i42 ne publie que le R(3)
	// de tete (param[0]), la fenetre lit DrawnSlot ; ce ne sont pas les memes champs.
	c.classer("degaine:non_comparable", p, slot, "")
	// grenade_selectionnee : i47 est code en base 1 (0 = aucune selection, GrenadeSetNoSelection),
	// le rang de la fenetre en base 0 (correction 2.7.d1 (a)).
	switch {
	case g.gsLu && g.gsSel == GrenadeSetNoSelection && inv.SelectedGrenadeRank >= 0:
		c.classer("grenade_selectionnee:grammaire_sans_selection_fenetre_lit", p, slot,
			fmt.Sprintf("fenetre=%d", inv.SelectedGrenadeRank))
	case g.gsLu && g.gsSel == GrenadeSetNoSelection:
		c.classer("grenade_selectionnee:grammaire_sans_selection_fenetre_vide", p, slot, "")
	default:
		c.comparer("grenade_selectionnee", p, slot, g.gsLu, g.gsLu, g.gsSel-1, inv.SelectedGrenadeRank >= 0,
			inv.SelectedGrenadeRank)
	}
}

// comparer classe une grandeur : `atteinte` = l occurrence est traversee, `lue`/`gv` = sa valeur,
// `flue`/`fv` = celle de la fenetre.
func (c *ri27d0Canal) comparer(nom string, p *lecture.Paquet, slot uint32, atteinte, lue bool, gv int, flue bool, fv int) {
	detail := fmt.Sprintf("grammaire=%d fenetre=%d", gv, fv)
	switch {
	case !atteinte && !flue:
		c.classer(nom+":non_atteinte_fenetre_vide", p, slot, "")
	case !atteinte:
		c.classer(nom+":non_atteinte_fenetre_lit", p, slot, detail)
	case !lue && !flue:
		c.classer(nom+":aucune_des_deux", p, slot, "")
	case lue && flue && gv == fv:
		c.classer(nom+":egales", p, slot, "")
	case lue && !flue:
		c.classer(nom+":grammaire_seule", p, slot, "")
	case !lue && flue:
		c.classer(nom+":fenetre_seule", p, slot, detail)
	default:
		c.classer(nom+":differentes", p, slot, detail)
	}
}

// situerLesFenetres range chaque marque de portage et chaque famille trouvee par fenetre dans
// le composant qui la porte.
func (c *ri27d0Canal) situerLesFenetres(p *lecture.Paquet, r *lecture.Record, fin int) {
	comps := p.Comps[r.Comps[0]:r.Comps[1]]
	situer := func(b int) (string, int) {
		for _, co := range comps {
			if co.Etat == lecture.EtatInfranchissable {
				return fmt.Sprintf("au_dela_de_i%d_infranchissable", co.Index), 0
			}
			if b >= int(co.Debut) && b < int(co.Debut+co.Bits) {
				return fmt.Sprintf("i%d", co.Index), b - int(co.Debut)
			}
		}
		if len(comps) > 0 && b < int(comps[0].Debut) {
			return "avant_le_premier_composant", 0
		}
		return "apres_la_traversee", 0
	}
	var w uint32
	premier := -1
	for b := int(r.Debut); b < fin; b++ {
		w = w<<1 | uint32(source.BitAt(p.Payload, b))
		if b-int(r.Debut) < 31 {
			continue
		}
		at := b - 31
		if c.known[w] {
			o, d := situer(at)
			c.fams[fmt.Sprintf("%s\t%d", o, d)]++
			if premier < 0 {
				premier = at
			}
		}
	}
	// LES LARGEURS PAR COMPOSANT, ventilees par la preuve du record : un composant dont la largeur
	// varie dans les records non prouves et pas dans les records fermes designe la lecture fautive.
	for _, co := range comps {
		if co.Index <= 22 {
			c.fams[fmt.Sprintf("largeur_i%02d_%s\t%d", co.Index, c.preuve, co.Bits)]++
		}
	}
	// L ECART A LA PREMIERE FAMILLE : ou la traversee place i43 contre ou la fenetre trouve la
	// premiere famille (porte d un bit avant) ; et l ecart de la fin de la traversee au record suivant.
	if len(comps) > 0 {
		der := comps[len(comps)-1]
		c.fams[fmt.Sprintf("ecart_fin\t%d", int(der.Debut+der.Bits)-fin)]++
	}
	for _, co := range comps {
		if c.roles[int(co.Index)] == "ti0" && premier >= 0 {
			c.fams[fmt.Sprintf("ecart_i43\t%d", int(co.Debut)-(premier-1))]++
		}
	}
}

func TestRI27d0ImagesCles(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	noms := weaponv3.KnownWeaponHigh32Copie()
	known := make(map[uint32]bool, len(noms))
	for f := range noms {
		known[f] = true
	}
	var lignes []string
	agr := map[string]map[string]int{}
	dumps := map[string]bool{}
	for _, f := range strings.Split(os.Getenv("RI27D1_RECORDS"), ",") {
		dumps[f] = true
	}
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		arch, err := fc.bipedArchetype()
		if err != nil {
			t.Fatalf("%s : archetype : %v", court, err)
		}
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre : %v", court, err)
		}
		vf, okf := FilmFormatVersion(fc.film)
		fm := fmt.Sprintf("f%d", vf)
		if !okf {
			fm = "f_inconnu"
		}
		c := &ri27d0Canal{fc: fc, arch: arch, court: court, known: known, noms: noms, roles: ri27d0Roles(arch),
			classes: map[string]int{}, ech: map[string]int{}, fams: map[string]int{}, x: nouvelleExt(reg, dumps[court])}
		if err := Distribuer(fc, c); err != nil {
			t.Fatalf("%s : distribution : %v", court, err)
		}
		lignes = append(lignes, ri27d0Trier(court, "S", c.classes)...)
		lignes = append(lignes, c.x.ri27d1Lignes(court, fm, c.classes, agr)...)
		t.Logf("FORMAT %s %s", court, fm)
		lignes = append(lignes, ri27d0Trier(court, "F", c.fams)...)
		lignes = append(lignes, c.lignes...)
		t.Logf("%s : %d records bipedes, armes egales %d, non atteintes %d+%d", court, c.classes["records_bipedes"],
			c.classes["armes:egales"], c.classes["armes:non_atteintes_fenetre_vide"], c.classes["armes:non_atteintes_fenetre_lit"])
		runtime.GC()
	}
	for _, genre := range []string{"SF", "AF", "TF", "MF", "YF"} {
		lignes = append(lignes, ri27d0Trier("*", genre, agr[genre])...)
	}
	ri27cEcrire(t, filepath.Join(sortie, "images_cles.tsv"), lignes)
}

// ri27d0Trier rend les comptes d une table, une ligne par cle, dans l ordre des cles.
func ri27d0Trier(court, genre string, m map[string]int) []string {
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	out := make([]string, 0, len(cles))
	for _, k := range cles {
		out = append(out, fmt.Sprintf("%s\t%s\t%s\t%d", genre, court, k, m[k]))
	}
	return out
}
