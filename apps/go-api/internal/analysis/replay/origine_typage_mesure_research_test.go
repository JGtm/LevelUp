package replay

// origine_typage_mesure_research_test.go — TYPER LES POINTS PAR LA MESURE, la chaine de tags
// ayant echoue.
//
// CE QUI A ECHOUE, ET IL FAUT LE LIRE AVANT DE REESSAYER. La voie « propre » aurait ete de
// nommer chaque `type_id` par les fichiers du jeu : type_id -> tag `food` -> `foki` -> `fosp`
// -> objet engendre. Elle s'arrete au `fosp` : ses references ne resolvent dans AUCUN module
// indexe (16 types sondes, 13 indetermines, le manifeste d'equipement du titre ne recoupe pas
// une seule reference). Mesure dans `himap.TestOrigineFospElucidation`.
//
// CE QUE FAIT CETTE SONDE A LA PLACE. Une naissance d'equipement (`ti=37`) porte SON IDENTITE
// (`AbilityID`) et SA POSITION. Le manifeste du titre nomme l'identite. Le catalogue de la
// carte donne le `type_id` du point le plus proche. On apparie les deux et on publie la table
// `type_id -> familles observees`. C'est une MESURE, pas une lecture : elle ne type que les
// points que le film ALLUME, et elle le dit.
//
// C'est aussi le recoupement demande au gate : les familles ici doivent etre celles des
// classes 2/3 du canal natif (grenades et equipement), jamais des armes.
//
// Gardes : PICKUP_FILM, PICKUP_MAP, ORIGINE_DUMP, ORIGINE_TYPES. Un film par process.

import (
	"math"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// oriTypageEps est le rayon d'appariement naissance -> point. MEME valeur que l'instrument
// d'origine (`oriEps`) : deux seuils differents sur la meme geometrie rendraient les deux
// mesures incomparables.
const oriTypageEps = oriEps

// oriPointType est un point catalogue AVEC son type_id — l'instrument d'origine, lui, n'a
// besoin que de la position et jette le type.
type oriPointType struct {
	x, y, z float64
	typeID  int32
}

// oriPointsAvecType relit le dump et rend les points retenus par la recette, type conserve.
func oriPointsAvecType(t *testing.T) []oriPointType {
	t.Helper()
	types, source := oriTypesDeLaRecette(t)
	t.Logf("SOURCE DES TYPES : %s", source)
	objs := oriChargeDumpObjets(t)
	var out []oriPointType
	for _, o := range objs {
		if !types[o.TypeID] {
			continue
		}
		out = append(out, oriPointType{o.Pos.X, o.Pos.Y, o.Pos.Z, o.TypeID})
	}
	return out
}

// oriPlusProcheType rend le type_id du point le plus proche et la distance.
func oriPlusProcheType(pts []oriPointType, x, y, z float32) (int32, float64) {
	best, bd := int32(0), 1e18
	for _, p := range pts {
		dx, dy, dz := p.x-float64(x), p.y-float64(y), p.z-float64(z)
		d := dx*dx + dy*dy + dz*dz
		if d < bd {
			bd, best = d, p.typeID
		}
	}
	if bd == 1e18 {
		return 0, bd
	}
	return best, math.Sqrt(bd)
}

// TestOrigineTypageParLaMesure publie la table type_id -> familles observees.
func TestOrigineTypageParLaMesure(t *testing.T) {
	s := glResolve(t)
	pts := oriPointsAvecType(t)
	if len(pts) == 0 {
		t.Skip("aucun point catalogue : dump ou liste de types absents")
	}
	cre, st, err := filmdec.ScanFilmEquipmentCreations(s.dir, &s.wr)
	if err != nil {
		t.Fatalf("balayage des naissances : %v", err)
	}
	familles := goldenCatalog(t).EquipmentFamilies
	if len(familles) == 0 {
		t.Fatal("manifeste d'equipement vide : le typage n'aurait aucun sens")
	}
	// type_id -> famille -> effectif
	table := map[int32]map[string]int{}
	var nommees, sansID, horsManifeste, horsPortee int
	idsVus := map[uint32]int{}
	for _, c := range cre {
		if !c.HasID {
			sansID++
			continue
		}
		idsVus[c.AbilityID]++
		fam, ok := familles[c.AbilityID]
		if !ok {
			horsManifeste++
			continue
		}
		nommees++
		ty, d := oriPlusProcheType(pts, c.X, c.Y, c.Z)
		if d > oriTypageEps {
			horsPortee++
			continue
		}
		if table[ty] == nil {
			table[ty] = map[string]int{}
		}
		table[ty][fam]++
	}
	t.Logf("== TYPAGE MESURE · %s · %d points catalogues ==", s.dir, len(pts))
	t.Logf("naissances : %d lues sur %d ancres (rejets : %d masque, %d position, %d debord) · "+
		"%d NOMMEES par le manifeste · %d anonymes · %d nommees mais hors portee (> %.2f m)",
		len(cre), st.Anchors, st.MaskBad, st.PosBad, st.Overflow,
		nommees, sansID, horsPortee, oriTypageEps)
	t.Logf("DIAGNOSTIC : %d sans identite transmise · %d avec identite mais HORS manifeste "+
		"(%d identifiants distincts)", sansID, horsManifeste, len(idsVus))
	if len(idsVus) > 0 {
		type kv struct {
			id uint32
			n  int
		}
		var l []kv
		for k, v := range idsVus {
			l = append(l, kv{k, v})
		}
		sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
		if len(l) > 12 {
			l = l[:12]
		}
		for _, e := range l {
			t.Logf("   identite 0x%08X vue %d fois", e.id, e.n)
		}
	}
	if len(table) == 0 {
		t.Log("VERDICT : aucune naissance nommee ne tombe sur un point catalogue — le typage " +
			"par la mesure est MUET sur ce film. Ce n'est pas une refutation du catalogue : " +
			"c'est que le mode n'allume pas les points d'equipement, ou que le film ne nomme " +
			"pas ses naissances.")
		return
	}
	keys := make([]int32, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return uint32(keys[i]) < uint32(keys[j]) })
	for _, k := range keys {
		fams := table[k]
		noms := make([]string, 0, len(fams))
		for f := range fams {
			noms = append(noms, f)
		}
		sort.Strings(noms)
		ligne := ""
		for i, n := range noms {
			if i > 0 {
				ligne += " "
			}
			ligne += n + ":" + itoa(fams[n])
		}
		t.Logf("0x%08X  %s", uint32(k), ligne)
	}
	t.Log("LECTURE : une famille `grenade_*` type le point en GRENADE, toute autre famille du " +
		"manifeste le type en EQUIPEMENT. Un point qui ne recoit que des familles d'armes " +
		"serait une REFUTATION du crible — aucune n'est attendue ici.")
}

// oriPosAuTemps rend la position du bipede `slot` la plus proche dans le TEMPS de `ts`, et
// l'ecart temporel. Sans position ou avec un ecart trop grand, l'appariement est refuse : un
// ramassage date par une position vieille de plusieurs secondes ne dit rien du lieu.
func oriPosAuTemps(pos map[uint32][]filmdec.BipedPosition, slot uint32, ts uint64,
) (filmdec.BipedPosition, uint64, bool) {
	l, ok := pos[slot]
	if !ok || len(l) == 0 {
		return filmdec.BipedPosition{}, 0, false
	}
	var best filmdec.BipedPosition
	bd := uint64(1) << 62
	for _, p := range l {
		d := p.TimestampUS - ts
		if p.TimestampUS < ts {
			d = ts - p.TimestampUS
		}
		if d < bd {
			bd, best = d, p
		}
	}
	return best, bd, true
}

// oriTypageEcartMax est l'ecart temporel maximal entre le ramassage et la position retenue.
// 100 ms : a la cadence des paquets de position, c'est l'ordre de la frame, et un joueur ne
// traverse pas un rayon d'un metre dans cet intervalle.
const oriTypageEcartMax = 100_000

// TestOrigineTypageParLesRamassages type les points par le CANAL NATIF plutot que par les
// naissances — les naissances de ce film ne portent pas d'identifiant de catalogue (39
// identites, toutes distinctes, une occurrence chacune : signature d'un identifiant
// d'instance). Le ramassage, lui, porte un `CatalogID` dont le depot a MESURE qu'il est un
// identifiant de catalogue, et c'est la table du manifeste qui a etabli les classes 2/3 au
// schema 31 : le recoupement demande au gate se joue donc ici, ou nulle part.
func TestOrigineTypageParLesRamassages(t *testing.T) {
	s := glResolve(t)
	pts := oriPointsAvecType(t)
	if len(pts) == 0 {
		t.Skip("aucun point catalogue")
	}
	pickups, st, err := filmdec.ScanFilmBipedPickups(s.dir)
	if err != nil {
		t.Fatalf("balayage des ramassages : %v", err)
	}
	familles := goldenCatalog(t).EquipmentFamilies
	table := map[int32]map[string]int{}
	var nonArme, nommes, sansPos, tropLoin, horsPortee int
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		nonArme++
		fam, ok := familles[p.CatalogID]
		if !ok {
			fam = "(hors manifeste)"
		} else {
			nommes++
		}
		bp, ecart, ok := oriPosAuTemps(s.pos, p.Slot, p.TimestampUS)
		if !ok {
			sansPos++
			continue
		}
		if ecart > oriTypageEcartMax {
			tropLoin++
			continue
		}
		ty, d := oriPlusProcheType(pts, bp.X, bp.Y, bp.Z)
		if d > oriTypageEps {
			horsPortee++
			continue
		}
		if table[ty] == nil {
			table[ty] = map[string]int{}
		}
		table[ty][fam]++
	}
	t.Logf("== TYPAGE PAR LES RAMASSAGES · %d points catalogues ==", len(pts))
	t.Logf("ramassages : %d lus (%d multi-evenements) · %d NON-ARME · %d nommes par le "+
		"manifeste · rejets : %d sans position, %d ecart > %d us, %d hors portee (> %.2f m)",
		len(pickups), st.MultiEvent, nonArme, nommes, sansPos, tropLoin, oriTypageEcartMax,
		horsPortee, oriTypageEps)
	if len(table) == 0 {
		t.Log("VERDICT : MUET — aucun ramassage non-arme ne se produit a moins d'un metre d'un " +
			"point catalogue. Le typage des points reste NON ETABLI.")
		return
	}
	keys := make([]int32, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return uint32(keys[i]) < uint32(keys[j]) })
	for _, k := range keys {
		fams := table[k]
		noms := make([]string, 0, len(fams))
		for f := range fams {
			noms = append(noms, f)
		}
		sort.Strings(noms)
		ligne := ""
		for i, n := range noms {
			if i > 0 {
				ligne += " "
			}
			ligne += n + ":" + itoa(fams[n])
		}
		marque := ""
		if _, ok := mapvarPadFamilyName(k); ok {
			marque = "   <<< SOCLE D'ARME PROUVE — un ramassage non-arme ici est un signal"
		}
		t.Logf("0x%08X  %s%s", uint32(k), ligne, marque)
	}
}

// mapvarPadFamilyName nomme les trois socles prouves. Table recopiee pour que le test ne
// depende pas d'un export que la production ne fait pas.
func mapvarPadFamilyName(ty int32) (string, bool) {
	switch uint32(ty) {
	case 0x5F379533:
		return "power", true
	case 0x6253CFC0:
		return "rack", true
	case 0x5E86D110:
		return "powerup", true
	}
	return "", false
}

// oriTypageMinObs / oriTypageMajorite — LES SEUILS, ECRITS AVANT LA MESURE DE CONTROLE.
// Un point n'est type que sur au moins 3 observations dont 80 % tombent du meme cote
// (grenade contre equipement). En dessous, il reste `spawner` : nature non etablie.
const (
	oriTypageMinObs   = 3
	oriTypageMajorite = 0.80
)

// TestOrigineControleArmesSurSocles est le CONTROLE SYMETRIQUE du typage : les ramassages
// d'ARME doivent tomber sur les socles d'armes PROUVES, pas sur les points d'equipement. Si
// les armes se repartissaient uniformement sur les 65 points, le catalogue elargi ne separerait
// rien et le typage precedent serait un artefact de densite.
func TestOrigineControleArmesSurSocles(t *testing.T) {
	s := glResolve(t)
	pts := oriPointsAvecType(t)
	if len(pts) == 0 {
		t.Skip("aucun point catalogue")
	}
	pickups, _, err := filmdec.ScanFilmBipedPickups(s.dir)
	if err != nil {
		t.Fatalf("balayage des ramassages : %v", err)
	}
	var armesSurSocle, armesSurAutre, armesHorsPortee int
	repart := map[int32]int{}
	for _, p := range pickups {
		if !filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		bp, ecart, ok := oriPosAuTemps(s.pos, p.Slot, p.TimestampUS)
		if !ok || ecart > oriTypageEcartMax {
			continue
		}
		ty, d := oriPlusProcheType(pts, bp.X, bp.Y, bp.Z)
		if d > oriTypageEps {
			armesHorsPortee++
			continue
		}
		repart[ty]++
		if _, estSocle := mapvarPadFamilyName(ty); estSocle {
			armesSurSocle++
		} else {
			armesSurAutre++
		}
	}
	total := armesSurSocle + armesSurAutre
	t.Logf("== CONTROLE : LES ARMES TOMBENT-ELLES SUR LES SOCLES D'ARMES ? ==")
	t.Logf("ramassages d'arme apparies : %d (%d hors portee)", total, armesHorsPortee)
	if total == 0 {
		t.Log("VERDICT : aucun ramassage d'arme apparie — controle MUET.")
		return
	}
	t.Logf("sur un SOCLE D'ARME prouve : %d (%.1f %%) · sur un autre point : %d (%.1f %%)",
		armesSurSocle, 100*float64(armesSurSocle)/float64(total),
		armesSurAutre, 100*float64(armesSurAutre)/float64(total))
	// Le temoin de densite : les socles d'armes ne sont qu'une PART des points catalogues.
	// Si les armes s'y concentrent bien au-dela de cette part, la separation est reelle.
	var nSocles int
	for _, p := range pts {
		if _, ok := mapvarPadFamilyName(p.typeID); ok {
			nSocles++
		}
	}
	part := 100 * float64(nSocles) / float64(len(pts))
	t.Logf("TEMOIN DE DENSITE : les socles d'armes sont %d des %d points catalogues (%.1f %%) — "+
		"c'est le taux qu'on obtiendrait si les armes tombaient AU HASARD sur les points",
		nSocles, len(pts), part)
	keys := make([]int32, 0, len(repart))
	for k := range repart {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return repart[keys[i]] > repart[keys[j]] })
	for _, k := range keys {
		nom, ok := mapvarPadFamilyName(k)
		if !ok {
			nom = "(point d'equipement)"
		}
		t.Logf("   0x%08X  %d armes   %s", uint32(k), repart[k], nom)
	}
}

// oriTypageVerdict — LE TYPAGE CONSOLIDE, et le seul qui vaille.
//
// POURQUOI LES DEUX SONDES PRECEDENTES NE SUFFISENT PAS. La premiere ne comptait que les
// ramassages NON-ARME : elle a type `0x0CD504B0` en grenade sur 3 observations, alors que le
// controle symetrique montre que ce meme point recoit 10 ARMES. Un point ne se juge pas sur une
// classe : il se juge sur la REPARTITION de tout ce qu'on y ramasse.
//
// LE TEMOIN EST LE TAUX DE BASE DU FILM. Si 33 % des ramassages du film sont non-arme, alors un
// point qui recoit 33 % de non-arme ne dit RIEN — c'est le hasard. Seul un point qui s'ecarte
// franchement du taux de base porte une information.
//
// CE QUI VALIDE LA METHODE : les trois socles PROUVES tombent exactement ou ils doivent. Le
// socle `power` rend 0 % de non-arme, le `rack` rend le taux de base, le `powerup` rend 100 %
// de `powerup_overshield`. Aucun de ces trois n'a servi a calibrer quoi que ce soit.
type oriTypageVerdict struct {
	armes, grenades, equipements int
}

func (v oriTypageVerdict) total() int { return v.armes + v.grenades + v.equipements }

// TestOrigineTypageConsolide publie la table finale type_id -> nature, avec son temoin.
func TestOrigineTypageConsolide(t *testing.T) {
	s := glResolve(t)
	pts := oriPointsAvecType(t)
	if len(pts) == 0 {
		t.Skip("aucun point catalogue")
	}
	pickups, _, err := filmdec.ScanFilmBipedPickups(s.dir)
	if err != nil {
		t.Fatalf("balayage des ramassages : %v", err)
	}
	familles := goldenCatalog(t).EquipmentFamilies
	par := map[int32]*oriTypageVerdict{}
	var totalArme, totalNonArme int
	for _, p := range pickups {
		arme := filmdec.BipedPickupIsWeaponClass(p.Class)
		if arme {
			totalArme++
		} else {
			totalNonArme++
		}
		bp, ecart, ok := oriPosAuTemps(s.pos, p.Slot, p.TimestampUS)
		if !ok || ecart > oriTypageEcartMax {
			continue
		}
		ty, d := oriPlusProcheType(pts, bp.X, bp.Y, bp.Z)
		if d > oriTypageEps {
			continue
		}
		if par[ty] == nil {
			par[ty] = &oriTypageVerdict{}
		}
		switch {
		case arme:
			par[ty].armes++
		case strings.HasPrefix(familles[p.CatalogID], "grenade_"):
			par[ty].grenades++
		default:
			par[ty].equipements++
		}
	}
	base := 100 * float64(totalNonArme) / float64(totalArme+totalNonArme)
	t.Logf("== TYPAGE CONSOLIDE · %d points catalogues ==", len(pts))
	t.Logf("TAUX DE BASE DU FILM : %d non-arme sur %d ramassages = %.1f %% — c'est ce qu'un "+
		"point rend s'il ne porte AUCUNE information", totalNonArme, totalArme+totalNonArme, base)
	t.Logf("%-12s %6s %6s %6s %8s   %s", "type_id", "armes", "gren.", "equip.", "non-arme", "verdict")
	keys := make([]int32, 0, len(par))
	for k := range par {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return par[keys[i]].total() > par[keys[j]].total() })
	for _, k := range keys {
		v := par[k]
		n := v.total()
		partNonArme := 100 * float64(v.grenades+v.equipements) / float64(n)
		verdict := "spawner (n trop faible ou pas de majorite)"
		switch {
		case n < oriTypageMinObs:
			verdict = "spawner (n < " + itoa(oriTypageMinObs) + ")"
		case partNonArme <= base:
			verdict = "PAS un point d'equipement (sous le taux de base)"
		case float64(v.grenades)/float64(n) >= oriTypageMajorite:
			verdict = "GRENADE"
		case float64(v.equipements)/float64(n) >= oriTypageMajorite:
			verdict = "EQUIPEMENT"
		}
		if nom, ok := mapvarPadFamilyName(k); ok {
			verdict += "   [socle PROUVE : " + nom + "]"
		}
		t.Logf("0x%08X %6d %6d %6d %7.1f %%   %s",
			uint32(k), v.armes, v.grenades, v.equipements, partNonArme, verdict)
	}
}
