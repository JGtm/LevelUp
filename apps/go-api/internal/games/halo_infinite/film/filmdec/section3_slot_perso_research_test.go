package filmdec

// section3_slot_perso_research_test.go — L'HYPOTHESE DE LA PERSONNALISATION, MISE A L'EPREUVE.
//
// ## L'HYPOTHESE, TELLE QU'ELLE A ETE POSEE AVANT LA MESURE
//
// « L'enregistrement de slot porte la PERSONNALISATION du joueur (noyau et pieces d'armure,
// revetements, visiere, embleme, effets, apparences d'armes et de vehicules), et les deux
// classes de longueur mesurees par la phase 1 sont "avec" et "sans" ce bloc. » Appuis avances :
// rien de tel n'a ete vu ailleurs dans le film ; le Theater doit rendre les tenues de l'epoque
// des mois plus tard ; quelques milliers d'octets par joueur est l'ordre de grandeur d'une tenue
// complete en identifiants de tags.
//
// ## CE QUE LE DESASSEMBLAGE DIT, ET QUI N'EST PAS UNE ADJACENCE DE CHAINES
//
// `FUN_1407edea8` (l'ecrivain du sous-enregistrement) recopie `sub+0xcc0` en BRUT sur `0x39e0`
// bits, soit 1 852 octets. Or `FUN_140969c54` — le serialiseur DELTA de la MEME structure (memes
// offsets `+0xc10`, `+0xc12`, `+0xc14`, `+0xc34`..`+0xc38`, `+0xc48`, `+0xcb0`, `+0xcb8`, et le
// meme `FUN_1407ecd00` sur la base) — ne recopie pas ce `+0xcc0` en brut : il le passe CHAMP PAR
// CHAMP a `FUN_1407ec27c` (`LEA RDX,[RSI + 0xcc0] ; CALL 0x1407ec27c`, verifie au
// desassemblage). Et `FUN_1407ec27c` ecrit des champs NOMMES, les noms etant passes en clair a
// l'ecrivain de u32 `FUN_1407edaf4(writer, nom, valeur)` :
//
//	"variantName" [0x4f]   "styleName" [0x50]   "themeName" [0x1cc]   "coatingName" [0x1cd]
//	"actionPose" [0x1ce]   "model_region" / "model_permutation" (paires, des [2])
//
// et le pool de chaines qui les porte (`0x143686770`..`0x1436868a0`) aligne, dans l'ordre :
// `unarmed`, `variantName`, `styleName`, `regionOverrideName`, `themeName`, `coatingName`,
// `markerName`, `desired-representation`, `variant-name`, `queued-replay-mission`, `unknown`,
// `region`, `permutation`, `model_permutation`, `model_region`.
//
// La fermeture arithmetique ferme la question de l'identite des deux structures : le plus haut
// indice que `FUN_1407ec27c` touche est `param_2[0x1ce]` (« actionPose »), soit l'octet 0x738,
// et `0x738 + 4 = 0x73C = 1 852` — EXACTEMENT la largeur du bloc brut. Trois autres fermetures
// tombent sans ajustement :
//
//	FUN_1407ebf44(cust+0x14C) : 24 entrees de 0x24 o  -> 0x14C + 24 x 0x24 = 0x4AC
//	FUN_1407eda5c(cust+0x4AC) :  7 entrees de 0x58 o  -> 0x4AC +  7 x 0x58 = 0x714
//	                             (chacune : variant, style, theme/coating/marker, 8 x region+perm)
//	                                                     0x730 theme, 0x734 coating, 0x738 pose
//
// 24 attaches d'armure, 7 objets a 8 couples region/permutation, un theme et un revetement de
// tete, une pose. C'est la forme d'une tenue Halo Infinite, lue dans l'executable. LA STRUCTURE
// EST DONC PROUVEE ; reste a mesurer si le film la remplit.
//
// ## LES CONTROLES, TOUS ECRITS AVANT LA MESURE
//
//	P-ZER  LE BLOC EST-IL REMPLI ? Compte d'octets non nuls, zone par zone, sur tous les
//	       enregistrements. C'est le premier controle a passer : une structure prouvee mais vide
//	       ne porte aucune donnee, et toute conclusion tiree de ses champs serait un artefact.
//	P-STA  STABILITE PAR JOUEUR. Si une zone est la tenue d'un joueur, le MEME joueur dans deux
//	       films doit y avoir un contenu IDENTIQUE octet pour octet.
//	P-DIS  DISCRIMINATION ENTRE JOUEURS. Deux joueurs DIFFERENTS ne doivent pas partager le meme
//	       contenu. Une zone constante pour tout le monde ne porte pas de tenue : elle est vide
//	       ou par defaut. P-STA sans P-DIS ne prouve RIEN — une zone de zeros passe P-STA.
//	P-CHP  PLAUSIBILITE DES CHAMPS NOMMES du bloc de 1 852 octets, aux offsets lus dans l'exe.
//	P-TAG  FORME DES IDENTIFIANTS de la liste de mots de 32 bits : des identifiants de tags se
//	       reconnaissent a `0xFFFFFFFF` (absent) et a des valeurs hautes a forte entropie, PAS a
//	       une petite enumeration 0..10. Releve sans seuil.
//	P-CHA  La table des slots relue par CHAINAGE de la longueur predite (`s3sChaine`) doit rendre
//	       le meme roster que le balayage, SANS ses positions parasites.
//
// Garde `CHUNK00_FILMS`. Lecture seule, aucun code de production touche.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// Offsets DANS le bloc de 1 852 octets, lus chez `FUN_1407ec27c` et ses sous-serialiseurs.
const (
	s3pNbPaires   = 0x000 // [0] : nombre de couples model_region / model_permutation
	s3pTheme0     = 0x110 // FUN_1407ec50c : themeName / coatingName / markerName
	s3pVariant    = 0x13C // "variantName"  = param_2[0x4f]
	s3pStyle      = 0x140 // "styleName"    = param_2[0x50]
	s3pArmures    = 0x14C // 24 attaches de 0x24 octets (FUN_1407ebf44)
	s3pArmuresNb  = 24
	s3pArmurePas  = 0x24
	s3pObjets     = 0x4AC // 7 objets de 0x58 octets (FUN_1407eda5c)
	s3pObjetsNb   = 7
	s3pObjetPas   = 0x58
	s3pThemeTete  = 0x730 // "themeName"   = param_2[0x1cc]
	s3pCoatTete   = 0x734 // "coatingName" = param_2[0x1cd]
	s3pActionPose = 0x738 // "actionPose"  = param_2[0x1ce]
)

// s3pU32 lit un u32 petit-boutiste dans le bloc de personnalisation.
func s3pU32(b []byte, off int) uint32 {
	if off < 0 || off+4 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint32(b[off:])
}

// s3pEmpreinte rend les 8 premiers octets du SHA-256 d'une zone, en hexadecimal.
func s3pEmpreinte(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:8])
}

// s3pZones rend les zones de l'enregistrement, dans l'ordre ou l'ecrivain les produit. Le nom
// porte l'offset SOURCE, parce que c'est lui qui relie la mesure au desassemblage.
func s3pZones(e *s3sEnr) ([]string, [][]byte) {
	noms := []string{"masque sub+0x000", "liste octets sub+0x108", "liste mots sub+0x910",
		"bloc104 sub+0xc48", "bloc16 sub+0xc38", "PERSO sub+0xcc0", "queue sub+0x1400"}
	mots := make([]byte, 4*len(e.mots))
	for i, m := range e.mots {
		binary.LittleEndian.PutUint32(mots[i*4:], m)
	}
	return noms, [][]byte{e.masque, e.octets, mots, e.bloc104, e.bloc16, e.perso, e.bloc44}
}

// s3pNonNuls compte les octets non nuls d'une zone.
func s3pNonNuls(b []byte) int {
	n := 0
	for _, v := range b {
		if v != 0 {
			n++
		}
	}
	return n
}

// TestSection3SlotPersoRemplissage execute P-ZER : chaque zone de l'enregistrement est-elle
// REMPLIE ? Le controle passe avant tous les autres, parce qu'une zone vide invalide d'avance
// toute lecture de ses champs.
func TestSection3SlotPersoRemplissage(t *testing.T) {
	var noms []string
	var nn, taille, enrs []int
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		for _, e := range s3sChaine(d) {
			ns, zs := s3pZones(e)
			if noms == nil {
				noms = ns
				nn, taille, enrs = make([]int, len(ns)), make([]int, len(ns)), make([]int, len(ns))
			}
			for k, z := range zs {
				nn[k] += s3pNonNuls(z)
				taille[k] += len(z)
				if s3pNonNuls(z) > 0 {
					enrs[k]++
				}
			}
		}
	}
	for k, n := range noms {
		t.Logf("=== BILAN P-ZER === %-22s %7d octet(s) non nul(s) sur %7d ; zone non vide "+
			"dans %d enregistrement(s)", n, nn[k], taille[k], enrs[k])
	}
}

// TestSection3SlotPersoStabilite execute P-STA et P-DIS, zone par zone.
func TestSection3SlotPersoStabilite(t *testing.T) {
	var noms []string
	parJoueur := map[int]map[uint64]map[string]int{}
	parZone := map[int]map[string]map[uint64]bool{}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		for _, e := range s3sChaine(d) {
			ns, zs := s3pZones(e)
			noms = ns
			for k, z := range zs {
				emp := s3pEmpreinte(z)
				if parJoueur[k] == nil {
					parJoueur[k] = map[uint64]map[string]int{}
					parZone[k] = map[string]map[uint64]bool{}
				}
				if parJoueur[k][e.xuid] == nil {
					parJoueur[k][e.xuid] = map[string]int{}
				}
				parJoueur[k][e.xuid][emp]++
				if parZone[k][emp] == nil {
					parZone[k][emp] = map[uint64]bool{}
				}
				parZone[k][emp][e.xuid] = true
			}
		}
	}
	for k, n := range noms {
		stables, multi := s3pCompteStables(parJoueur[k])
		distinctes, partagees := s3pComptePartagees(parZone[k])
		t.Logf("=== BILAN P-STA/P-DIS === %-22s STABLE chez %d/%d joueurs vus dans plusieurs "+
			"films ; %d empreinte(s) distincte(s), dont %d partagee(s) par plusieurs joueurs",
			n, stables, multi, distinctes, partagees)
	}
}

// s3pCompteStables rend le nombre de joueurs vus dans au moins deux films dont la zone a une
// seule empreinte, et le nombre de tels joueurs.
func s3pCompteStables(par map[uint64]map[string]int) (stables, multi int) {
	for _, emps := range par {
		tot := 0
		for _, c := range emps {
			tot += c
		}
		if tot < 2 {
			continue
		}
		multi++
		if len(emps) == 1 {
			stables++
		}
	}
	return stables, multi
}

// s3pComptePartagees rend le nombre d'empreintes distinctes et celles portees par plusieurs
// xuids.
func s3pComptePartagees(par map[string]map[uint64]bool) (distinctes, partagees int) {
	for _, xs := range par {
		distinctes++
		if len(xs) > 1 {
			partagees++
		}
	}
	return distinctes, partagees
}

// TestSection3SlotPersoChamps execute P-CHP : releve des champs NOMMES du bloc de 1 852 octets.
func TestSection3SlotPersoChamps(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		es := s3sChaine(d)
		t.Logf("=== %s === %d enregistrement(s) chaines", filepath.Base(dir), len(es))
		for i, e := range es {
			b := e.perso
			t.Logf("  slot %2d %-16s variantName %08x styleName %08x theme/coating/marker "+
				"%08x/%08x/%08x theme(tete) %08x coating(tete) %08x actionPose %08x "+
				"paires %d ; %d/%d octets non nuls", i, e.gamertag, s3pU32(b, s3pVariant),
				s3pU32(b, s3pStyle), s3pU32(b, s3pTheme0), s3pU32(b, s3pTheme0+4),
				s3pU32(b, s3pTheme0+8), s3pU32(b, s3pThemeTete), s3pU32(b, s3pCoatTete),
				s3pU32(b, s3pActionPose), s3pU32(b, s3pNbPaires), s3pNonNuls(b), len(b))
			s3pArmure(t, b)
			s3pObjet(t, b)
		}
	}
}

// s3pArmure imprime les 24 attaches d'armure non vides.
func s3pArmure(t *testing.T, b []byte) {
	t.Helper()
	pleines := 0
	for k := 0; k < s3pArmuresNb; k++ {
		o := s3pArmures + k*s3pArmurePas
		v, s := s3pU32(b, o+4), s3pU32(b, o+8)
		th, co, ma := s3pU32(b, o+0xc), s3pU32(b, o+0x10), s3pU32(b, o+0x14)
		ro := s3pU32(b, o+0x20)
		if v == 0 && s == 0 && th == 0 && co == 0 && ma == 0 && ro == 0 {
			continue
		}
		pleines++
		t.Logf("      attache %2d variant %08x style %08x theme %08x coating %08x "+
			"marker %08x regionOverride %08x", k, v, s, th, co, ma, ro)
	}
	t.Logf("      -> %d/%d attaches d'armure non vides", pleines, s3pArmuresNb)
}

// s3pObjet imprime les 7 objets et leurs 8 couples region/permutation.
func s3pObjet(t *testing.T, b []byte) {
	t.Helper()
	for k := 0; k < s3pObjetsNb; k++ {
		o := s3pObjets + k*s3pObjetPas
		v, s := s3pU32(b, o+4), s3pU32(b, o+8)
		th, co, ma := s3pU32(b, o+0xc), s3pU32(b, o+0x10), s3pU32(b, o+0x14)
		var paires []string
		for j := 0; j < 8; j++ {
			r, p := s3pU32(b, o+0x18+j*8), s3pU32(b, o+0x1c+j*8)
			if r == 0 && p == 0 {
				continue
			}
			paires = append(paires, strconv.FormatUint(uint64(r), 16)+"/"+
				strconv.FormatUint(uint64(p), 16))
		}
		if v == 0 && s == 0 && th == 0 && co == 0 && ma == 0 && len(paires) == 0 {
			continue
		}
		t.Logf("      objet %d variant %08x style %08x theme %08x coating %08x marker %08x "+
			"region/perm %v", k, v, s, th, co, ma, paires)
	}
}

// TestSection3SlotChainage execute P-CHA : le chainage par longueur predite rend-il le meme
// roster que le balayage, sans ses positions parasites ?
func TestSection3SlotChainage(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		bal, ch := s3sEnrs(d), s3sChaine(d)
		ref := s3sOracle(dir)
		justes := 0
		var l []string
		for i, e := range ch {
			r, ok := ref[e.xuid]
			if ok && r.nom == e.gamertag {
				justes++
			}
			l = append(l, strconv.Itoa(i)+":"+e.gamertag)
		}
		t.Logf("=== %s === balayage %d position(s), chainage %d enregistrement(s), "+
			"%d nom(s) confirmes par l'oracle : %v", filepath.Base(dir), len(bal), len(ch),
			justes, l)
		if len(ref) > 0 && len(ch) != len(ref) {
			t.Logf("  ECART DE CARDINAL : oracle %d joueur(s), chainage %d", len(ref), len(ch))
		}
	}
}

// TestSection3SlotListes execute P-TAG et releve les trois listes prefixees — la SEULE partie
// variable de l'enregistrement, donc la source mecanique des deux classes de longueur mesurees
// par la phase 1.
func TestSection3SlotListes(t *testing.T) {
	distN, distM, distC := map[int]int{}, map[int]int{}, map[int]int{}
	vals := map[uint32]int{}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		for _, e := range s3sChaine(d) {
			distN[e.n]++
			distM[e.m]++
			distC[e.compteMasque]++
			for _, m := range e.mots {
				vals[m]++
			}
			if e.m > 0 {
				t.Logf("  %s %-16s masque %4d bits / %3d poses, N %4d o, M %3d mots ; "+
					"3 premiers mots %08x %08x %08x", filepath.Base(dir), e.gamertag,
					e.compteMasque, e.popMasque, e.n, e.m, e.mots[0], e.mots[1], e.mots[2])
			}
		}
	}
	t.Logf("  distribution de N : %v", s3pDist(distN))
	t.Logf("  distribution de M : %v", s3pDist(distM))
	t.Logf("  distribution du compte de masque : %v", s3pDist(distC))
	s3pRapportTags(t, vals)
}

// s3pRapportTags execute P-TAG : la forme des mots de 32 bits de la liste `sub+0x910`.
func s3pRapportTags(t *testing.T, vals map[uint32]int) {
	t.Helper()
	var total, absents, petits, hauts int
	for v, c := range vals {
		total += c
		switch {
		case v == 0xffffffff:
			absents += c
		case v < 256:
			petits += c
		default:
			hauts += c
		}
	}
	t.Logf("=== BILAN P-TAG === %d mot(s) de 32 bits lus, %d valeur(s) distincte(s) : "+
		"%d a 0xFFFFFFFF (absent), %d inferieurs a 256, %d hauts (forme d'identifiant de tag)",
		total, len(vals), absents, petits, hauts)
}

// s3pDist rend une distribution triee, lisible.
func s3pDist(m map[int]int) []string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		out = append(out, strconv.Itoa(k)+"x"+strconv.Itoa(m[k]))
	}
	return out
}

// TestSection3SlotPersoPrefixe execute P-PRE : QUELLE PART du contenu variable suit le JOUEUR ?
//
// P-STA a rendu 0/3 : aucune des zones variables n'est identique octet pour octet chez un meme
// joueur d'un match a l'autre. Mais « pas identique » n'est pas « sans rien de commun » —
// l'hypothese de l'utilisateur porte sur la tenue, qui ne change pas entre deux matchs
// rapproches, et elle survivrait a une zone qui melange tenue et donnees de match. Ce controle
// mesure donc le PREFIXE COMMUN, octet pour octet, des zones d'un meme joueur dans deux films.
//
// Critere ecrit avant la mesure : un prefixe commun LONG et une divergence nette ensuite
// designeraient une zone « tenue puis donnees de match » ; un prefixe commun court ou nul
// refuterait l'idee que la zone porte une tenue en clair au debut.
func TestSection3SlotPersoPrefixe(t *testing.T) {
	var noms []string
	vues := map[int]map[uint64][][]byte{}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		for _, e := range s3sChaine(d) {
			ns, zs := s3pZones(e)
			noms = ns
			for k, z := range zs {
				if vues[k] == nil {
					vues[k] = map[uint64][][]byte{}
				}
				vues[k][e.xuid] = append(vues[k][e.xuid], z)
			}
		}
	}
	for k, n := range noms {
		var lg []string
		total, mini := 0, -1
		for x, zs := range vues[k] {
			if len(zs) < 2 {
				continue
			}
			p := s3pPrefixe(zs)
			total++
			if mini < 0 || p < mini {
				mini = p
			}
			lg = append(lg, strconv.FormatUint(x, 10)+":"+strconv.Itoa(p)+"/"+
				strconv.Itoa(len(zs[0])))
		}
		sort.Strings(lg)
		t.Logf("=== BILAN P-PRE === %-22s prefixe commun le plus court %d octet(s) sur "+
			"%d joueur(s) vus dans plusieurs films : %v", n, mini, total, lg)
	}
}

// s3pPrefixe rend la longueur du plus long prefixe commun a toutes les occurrences d'une zone.
func s3pPrefixe(zs [][]byte) int {
	n := len(zs[0])
	for _, z := range zs[1:] {
		if len(z) < n {
			n = len(z)
		}
	}
	for i := 0; i < n; i++ {
		for _, z := range zs[1:] {
			if z[i] != zs[0][i] {
				return i
			}
		}
	}
	return n
}
