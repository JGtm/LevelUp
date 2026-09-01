package objectiveevents

// assaut_pied_ancre_test.go — LE PIED DE FILM EN ASSAUT : le negatif tenait-il a L'ANCRE ?
//
// # LE DOUTE, ET IL EST LEGITIME (utilisateur, 2026-09-01)
//
// Oddball, KOTH et Strongholds sortent TOUS LES TROIS du pied de film (`extractFromTh10`) : la
// possession du crane — un objet PORTE, l'analogue exact de la bombe — y est. Il serait tres
// etrange que l'Assaut n'y laisse rien. Or la sonde precedente (`TestAssautPiedDeFilm`) a rendu
// « quasi rien » … en levant le filtre `th==10` mais en GARDANT L'ANCRE : elle n'acceptait un
// bloc que si le XUID etait suivi des octets `0x2d|0x25` puis `0xc0`. Si l'Assaut prefixe ses
// evenements AUTREMENT, ce negatif est un artefact d'ancre, pas une absence.
//
// # CE QUE CETTE SONDE CHANGE — une seule chose a la fois
//
// L'ancre devient LE XUID SEUL (valeur 64 bits vraisemblable, bornes du scanner de production),
// sans aucune contrainte sur les octets qui suivent. Puis elle RELEVE au lieu d'exiger :
//
//	1. l'HISTOGRAMME DES DEUX OCTETS qui suivent chaque XUID — en Oddball on doit y voir
//	   dominer `[0x2d|0x25, 0xc0]` (le temoin de l'instrument) ; en Assaut, si un autre couple
//	   domine, c'est le prefixe propre au mode, et le negatif precedent tombe ;
//	2. pour chaque candidat, le BLOC par sa geometrie de production (end-marker `[00 00 2e e0]`
//	   en avant, bloc de 60 octets en arriere) et l'histogramme du `th` (octet 47) — SANS
//	   filtre de prefixe ;
//	3. la taille du pied, film par film — un pied de 100 octets et un pied de 100 ko ne
//	   racontent pas la meme histoire.
//
// # LES ISSUES, ecrites avant la mesure
//
//	L'ODDBALL MONTRE [0x2d|0x25,0xc0] ET L'ASSAUT UN AUTRE COUPLE DOMINANT
//	    -> le negatif precedent etait un artefact d'ancre ; la nouvelle ancre est lue dans
//	       l'histogramme et la sonde `th` se rejoue avec elle.
//	L'ASSAUT NE MONTRE PRESQUE AUCUN XUID DANS SON PIED (contre des centaines en Oddball)
//	    -> le pied d'Assaut ne porte pas d'evenements par joueur : le negatif etait REEL,
//	       et il se lit desormais sur une ancre insoupconnable.
//	LES PIEDS D'ASSAUT SONT ABSENTS OU MINUSCULES
//	    -> `footerData` ne trouve pas le bon chunk en Assaut, et TOUT est a reprendre.
//
// TEMOIN : `43716616` (Oddball, protocole D4) — l'instrument DOIT y voir l'ancre de production.
// S'il ne la voit pas, il est casse et son verdict sur l'Assaut ne vaut rien.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedAncre -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// paCorpus : les neuf films d'Assaut, le temoin Oddball, et trois temoins de modes a pied connu.
var paCorpus = []struct{ id, mode string }{
	{"43716616", "TEMOIN Oddball (D4)"},
	{"7f1bbf06", "TEMOIN KOTH arene"},
	{"696a9d7c", "TEMOIN Strongholds arene"},
	{"cde26226", "TEMOIN CTF arene"},
	{"9f57c612", "Assaut"},
	{"c75f33b8", "Assaut"},
	{"df8fcbef", "Assaut"},
	{"34bb3bc8", "Assaut"},
	{"1c01e34f", "Assaut"},
	{"ce083875", "Assaut"},
	{"35b75a31", "Assaut"},
	{"69b16f5d", "Assaut"},
	{"3d58eb37", "Assaut"},
}

// paBilan agrege le releve d'UN pied de film.
type paBilan struct {
	taille   int
	xuids    int
	couples  map[[2]byte]int // les deux octets qui suivent le XUID
	thParTag map[int]int     // th du bloc trouve par la geometrie de production, sans prefixe
	ancres   int             // candidats qui passent l'ancre DE PRODUCTION (reference)
}

// TestAssautPiedAncre releve l'ancre reelle des evenements du pied, mode par mode.
func TestAssautPiedAncre(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedAncre", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — releve interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	for _, f := range paCorpus {
		src, ok := afOuvrir(t, cache, f.id)
		if !ok {
			t.Logf("%-9s %-24s FILM ABSENT du cache", f.id, f.mode)
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			t.Logf("%-9s %-24s AUCUN PIED lisible (chunk_type 3 absent ?)", f.id, f.mode)
			continue
		}
		b := paReleve(footer)
		t.Logf("%-9s %-24s pied %6d o, %4d XUID(s), ancre production %3d, couples suivants : %s",
			f.id, f.mode, b.taille, b.xuids, b.ancres, paCouples(b.couples, 6))
		if len(b.thParTag) > 0 {
			t.Logf("           th des blocs (geometrie de production, SANS prefixe) : %s",
				paTh(b.thParTag))
		}
	}
}

// paReleve balaie UN pied : XUID seuls, puis histogrammes.
func paReleve(data []byte) paBilan {
	total := len(data) * 8
	b := paBilan{taille: len(data), couples: map[[2]byte]int{}, thParTag: map[int]int{}}
	seen := map[int]bool{}
	for pos := 0; pos+64+16 <= total; pos++ {
		x := readU64LEAtBit(data, pos)
		if x <= minXUID || x >= maxXUID {
			continue
		}
		// DEDUPLICATION par position de fin de XUID : les 64 lectures decalees d'un meme
		// champ rendent des valeurs differentes, donc la borne suffit a separer les vrais
		// champs ; mais un meme champ ne doit compter qu'une fois.
		if seen[pos] {
			continue
		}
		seen[pos] = true
		b.xuids++
		p1 := readByteAtBit(data, pos+64)
		p2 := readByteAtBit(data, pos+64+8)
		b.couples[[2]byte{p1, p2}]++
		if (p1 == 0x2d || p1 == 0x25) && p2 == 0xc0 {
			b.ancres++
		}
		if th, ok := paBlocTh(data, pos, total); ok {
			b.thParTag[th]++
		}
	}
	return b
}

// paBlocTh rejoue la geometrie de production d'un bloc (end-marker en avant, 60 octets en
// arriere, th a l'octet 47) SANS aucune contrainte de prefixe.
func paBlocTh(data []byte, xstart, total int) (int, bool) {
	win := xstart + 20000
	if win > total {
		win = total
	}
	for p := xstart; p <= win-32; p++ {
		if readByteAtBit(data, p) == 0 && readByteAtBit(data, p+8) == 0 &&
			readByteAtBit(data, p+16) == 0x2e && readByteAtBit(data, p+24) == 0xe0 {
			ebs := p - 60*8
			if ebs < xstart {
				return 0, false
			}
			return int(readByteAtBit(data, ebs+47*8)), true
		}
	}
	return 0, false
}

// paCouples rend les couples d'octets les plus frequents.
func paCouples(m map[[2]byte]int, max int) string {
	type l struct {
		c [2]byte
		n int
	}
	ls := make([]l, 0, len(m))
	for c, n := range m {
		ls = append(ls, l{c, n})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })
	out := ""
	for i, x := range ls {
		if i >= max {
			out += fmt.Sprintf(", … (+%d)", len(ls)-max)
			break
		}
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("[%02X %02X] x%d", x.c[0], x.c[1], x.n)
	}
	if out == "" {
		return "(aucun)"
	}
	return out
}

// paTh rend l'histogramme des th, trie par valeur.
func paTh(m map[int]int) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := ""
	for _, k := range ks {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("th=%d x%d", k, m[k])
	}
	return out
}

// TestAssautPiedFamilles — LES FAMILLES DU PIED, dedoublonnees, contre l'oracle des explosions.
//
// # CE QUE LE RELEVE D'ANCRE A ETABLI, ET CE QU'IL LAISSAIT FAUX
//
// Le pied d'Assaut n'est PAS vide : il porte des MILLIERS de blocs a la geometrie de production
// (end-marker + bloc de 60 octets), sous `th` 20, 50, 100 et 150 — seul `th=10` y est absent.
// L'affirmation « pied quasi absent en Assaut » etait donc fausse comme generalisation : elle ne
// valait que pour th=10.
//
// Mais le premier releve comptait par XUID CANDIDAT, et plusieurs candidats precedent le meme
// bloc : les comptes etaient gonfles et l'attribution d'acteur incertaine. Celui-ci DEDOUBLONNE
// PAR MARQUEUR DE FIN (la position du bloc, pas celle du candidat) : un bloc = un compte, son
// `t`, son slot d'acteur (octet 36).
//
// # LA QUESTION POSEE A CHAQUE FAMILLE — les criteres du chantier, inchanges
//
// Pour chaque valeur de `th`, sur les neuf films d'Assaut : la famille couvre-t-elle CHAQUE
// explosion avec un delai constant ? (COUVERTURE 28/28, CONSTANCE <= 20 %, SENS positif < 120 s.)
// Une pose de bombe precede chaque explosion d'exactement la meche : si une famille du pied la
// porte, elle tient ces criteres.
//
// TEMOIN : l'Oddball et le KOTH passent au meme instrument — leurs familles th=10 doivent y
// exister, et leurs autres familles ne doivent PAS couvrir les explosions d'Assaut (elles n'en
// ont pas) : le verdict ne porte que sur l'Assaut.
//
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedFamilles -v -timeout 30m
func TestAssautPiedFamilles(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedFamilles", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — releve interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	// LES TEMOINS D'ABORD : les comptes par famille d'un mode CONNU calibrent la lecture des
	// comptes d'Assaut. L'Oddball du protocole D4 porte des evenements de crane a la douzaine.
	for _, w := range []struct{ id, mode string }{
		{"43716616", "TEMOIN Oddball"}, {"7f1bbf06", "TEMOIN KOTH"},
		{"696a9d7c", "TEMOIN Strongholds"}, {"cde26226", "TEMOIN CTF"},
	} {
		src, ok := afOuvrir(t, cache, w.id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		blocs := paBlocs(footer)
		parTh := map[int][]int{}
		for _, b := range blocs {
			parTh[b.th] = append(parTh[b.th], b.t)
		}
		t.Logf("%-9s %-18s %5d bloc(s) dedoublonnes : %s", w.id, w.mode, len(blocs), paFamilles(parTh))
	}

	films := make([]string, 0, len(afExplosions))
	for id := range afExplosions {
		films = append(films, id)
	}
	sort.Strings(films)

	delais := map[int][]float64{}
	couverts := map[int]int{}
	total := 0
	for _, id := range films {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		blocs := paBlocs(footer)
		parTh := map[int][]int{}
		for _, b := range blocs {
			parTh[b.th] = append(parTh[b.th], b.t)
		}
		t.Logf("%-9s %5d bloc(s) dedoublonnes : %s", id, len(blocs), paFamilles(parTh))
		exps := afExplosions[id]
		total += len(exps)
		for th, ts := range parTh {
			for _, ms := range exps {
				meilleur := -1
				for _, p := range ts {
					d := ms - p
					if d > 0 && d <= afMecheMaxMS && (meilleur < 0 || d < meilleur) {
						meilleur = d
					}
				}
				if meilleur >= 0 {
					couverts[th]++
					delais[th] = append(delais[th], float64(meilleur))
				}
			}
		}
	}

	t.Logf("########## %d explosion(s) — familles contre les trois criteres :", total)
	ths := make([]int, 0, len(couverts))
	for th := range couverts {
		ths = append(ths, th)
	}
	sort.Ints(ths)
	for _, th := range ths {
		med, cv := afMedianeEtCV(delais[th])
		verdict := ""
		if couverts[th] == total && cv <= afCVMax {
			verdict = "   *** TIENT LES TROIS CRITERES ***"
		}
		t.Logf("  th=%-4d : %2d/%d couvertes, delai median %6.1f s, dispersion %5.1f %%%s",
			th, couverts[th], total, med/1000, cv*100, verdict)
	}
}

// paBloc est UN bloc du pied, dedoublonne par la position de son marqueur de fin.
type paBloc struct {
	th, t, slot int
}

// paBlocs balaie le pied par MARQUEURS DE FIN, pas par candidats XUID, AU BIT : le pied est un
// flux bit-packe (la production balaie au bit, `decodeTh10Block`), et une premiere version
// alignee sur l'octet sous-comptait d'un facteur ~8. Dedoublonnage par la position de bit du
// marqueur : un bloc = un compte.
//
// FAUX MARQUEURS : 32 bits exiges, soit ~2e-3 faux positif attendu par mebioctet de bruit ; les
// gardes de domaine sur `th` et `t` ecartent le reste.
func paBlocs(data []byte) []paBloc {
	total := len(data) * 8
	var out []paBloc
	for p := 60 * 8; p+32 <= total; p++ {
		if readByteAtBit(data, p) != 0 || readByteAtBit(data, p+8) != 0 ||
			readByteAtBit(data, p+16) != 0x2e || readByteAtBit(data, p+24) != 0xe0 {
			continue
		}
		ebs := p - 60*8
		th := int(readByteAtBit(data, ebs+47*8))
		if th == 0 || th > 250 {
			continue
		}
		t := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
			int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
		if t < 0 || t > 4*3600*1000 {
			continue
		}
		out = append(out, paBloc{th: th, t: t, slot: int(readByteAtBit(data, ebs+36*8))})
	}
	return out
}

// paFamilles rend le compte par famille, trie par th.
func paFamilles(m map[int][]int) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := ""
	for _, k := range ks {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("th=%d x%d", k, len(m[k]))
	}
	return out
}

// TestAssautPiedExplosion — LES BLOCS AUTOUR DES EXPLOSIONS, EN ENTIER.
//
// # POURQUOI VIDER LES BLOCS
//
// Les familles th=20 et th=50 couvrent 28/28 explosions avec un delai median de 0,5 a 1 s : la
// RECOMPENSE de l'explosion est dedans — mais noyee, car ces familles sont generiques (presentes
// dans tous les modes, dispersion enorme). Le bloc fait 60 octets et la production n'en lit que
// QUATRE (slot@36, th@47, t@48-51, team@55). L'octet de SOUS-TYPE — celui qui distingue
// « explosion » de « frag » — est forcement dans les 56 autres.
//
// Cet instrument imprime les blocs des fenetres [-3 s, +3 s] autour de chaque explosion, en
// entier, pour DEUX films. La lecture se fait a l'oeil : l'octet qui prend la meme valeur dans
// toutes les fenetres et une autre ailleurs est le sous-type.
//
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedExplosion -v -timeout 30m
func TestAssautPiedExplosion(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedExplosion", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — releve interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	for _, id := range []string{"9f57c612", "35b75a31"} {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		blocs := paBlocsOctets(footer)
		t.Logf("########## %s — %d bloc(s), fenetres de +-3 s autour des explosions :", id, len(blocs))
		for _, ms := range afExplosions[id] {
			t.Logf("  EXPLOSION a %d ms :", ms)
			for _, b := range blocs {
				if b.t < ms-3000 || b.t > ms+3000 {
					continue
				}
				t.Logf("    t=%7d (%+5d)  %s", b.t, b.t-ms, paHex(b.oct[:]))
			}
		}
	}
}

// paBlocOctets est un bloc AVEC ses 60 octets.
type paBlocOctets struct {
	t   int
	oct [60]byte
}

// paBlocsOctets rend les blocs dedoublonnes du pied avec leur contenu entier.
func paBlocsOctets(data []byte) []paBlocOctets {
	total := len(data) * 8
	var out []paBlocOctets
	for p := 60 * 8; p+32 <= total; p++ {
		if readByteAtBit(data, p) != 0 || readByteAtBit(data, p+8) != 0 ||
			readByteAtBit(data, p+16) != 0x2e || readByteAtBit(data, p+24) != 0xe0 {
			continue
		}
		ebs := p - 60*8
		th := int(readByteAtBit(data, ebs+47*8))
		if th == 0 || th > 250 {
			continue
		}
		t := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
			int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
		if t < 0 || t > 4*3600*1000 {
			continue
		}
		var b paBlocOctets
		b.t = t
		for i := 0; i < 60; i++ {
			b.oct[i] = readByteAtBit(data, ebs+i*8)
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// paHex rend 60 octets en hexadecimal groupe par 4, pour la lecture a l'oeil.
func paHex(b []byte) string {
	out := ""
	for i, x := range b {
		if i > 0 && i%4 == 0 {
			out += " "
		}
		out += fmt.Sprintf("%02x", x)
	}
	return out
}

// TestAssautPiedPose — LA POSE, cherchee comme recompense a delai constant avant l'explosion.
//
// # CE QUE LE VIDAGE DES BLOCS A ETABLI
//
// Le bloc du pied porte le GAMERTAG DE L'ACTEUR EN CLAIR (UTF-16LE), la VALEUR de la recompense
// en grand-boutien aux octets 44-47 (+20, +50, +100 — l'octet 47 que la production lit comme
// « type » est le petit octet de cette valeur), et l'instant aux octets 48-51. Le pied est donc
// LE FLUX DES RECOMPENSES DE SCORE PERSONNEL, et les evenements `th=10` des autres modes ne sont
// que les recompenses de +10 (tic de colline, de zone, de crane).
//
// # LA QUESTION
//
// Une POSE DE BOMBE est recompensee, et elle precede chaque explosion d'exactement la meche.
// Cet instrument imprime, par explosion, les blocs de la fenetre [-45 s, +2 s] en une ligne
// compacte (delai, valeur, gamertag) : la recompense de pose se lit comme la colonne dont le
// delai est CONSTANT d'une explosion a l'autre.
//
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedPose -v -timeout 30m
func TestAssautPiedPose(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedPose", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — releve interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	films := make([]string, 0, len(afExplosions))
	for id := range afExplosions {
		films = append(films, id)
	}
	sort.Strings(films)
	for _, id := range films {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		blocs := paBlocsOctets(footer)
		t.Logf("########## %s", id)
		for _, ms := range afExplosions[id] {
			t.Logf("  EXPLOSION a %d ms :", ms)
			for _, b := range blocs {
				if b.t < ms-45000 || b.t > ms+2000 {
					continue
				}
				t.Logf("    %+7d ms  +%-3d  %s", b.t-ms, paValeur(b.oct[:]), paTag(b.oct[:]))
			}
		}
	}
}

// paValeur lit la valeur de la recompense (u32 BE aux octets 44-47).
func paValeur(oct []byte) int {
	return int(oct[44])<<24 | int(oct[45])<<16 | int(oct[46])<<8 | int(oct[47])
}

// paTag extrait le gamertag UTF-16LE du bloc : la premiere suite de paires [ascii, 00] d'au
// moins trois caracteres.
func paTag(oct []byte) string {
	best := ""
	for i := 0; i+6 <= len(oct); i += 2 {
		s := ""
		for j := i; j+1 < len(oct); j += 2 {
			c := oct[j]
			if oct[j+1] != 0 || c < 0x20 || c > 0x7e {
				break
			}
			s += string(rune(c))
		}
		if len(s) > len(best) {
			best = s
		}
	}
	if best == "" {
		return "(sans tag)"
	}
	return best
}
