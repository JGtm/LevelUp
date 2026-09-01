package objectiveevents

// assaut_pied_classement_test.go — NOMMER LES RECOMPENSES D'ASSAUT : detonation d'abord, pose ensuite.
//
// # LA STRATEGIE, en deux temps et sans pont d'identite
//
// Le pied est le flux des recompenses (valeur + gamertag en clair + instant), et chaque bloc est
// precede du XUID de son acteur — c'est l'ancre meme de la production. La jointure se fait donc
// SANS le pont PLAT :
//
//	TEMPS 1 — LA DETONATION. Pour chaque explosion datee (oracle A0.3), les recompenses a
//	          moins de 1,5 s : la valeur qui couvre TOUTES les explosions avec UN acteur
//	          dominant par explosion est la prime de detonation, et son acteur est le poseur.
//	TEMPS 2 — LA POSE. Pour chaque explosion dont le poseur est connu, les recompenses AU MEME
//	          ACTEUR dans les 120 s precedentes : la colonne dont le delai est CONSTANT d'une
//	          explosion a l'autre est la prime de pose, et ce delai est LA MECHE.
//
// La meche etant une constante moteur, sa dispersion attendue est quasi nulle — le critere du
// chantier (<= 20 %) est ici genereux.
//
// # LE VERDICT (2026-09-01) — lire AVANT de reutiliser quoi que ce soit d'ici
//
// TEMPS 1, ATTRIBUTION REFUTEE : « la recompense individuelle la plus proche » designe l'auteur
// du DERNIER FRAG avant l'explosion, pas le poseur — les valeurs coincidentes sont +20/+50/+100,
// c'est-a-dire du combat. La table des « acteurs des detonations » que ce test imprime NE VAUT
// RIEN pour l'attribution ; le nommage du detonateur reste au statborg (chemin livre).
//
// TEMPS 2, NEGATIF NET ET DEFINITIF POUR LA POSE : aucune valeur de recompense n'a un delai
// constant avant l'explosion (dispersions 54 a 60 %). Et le recensement des valeurs est CLOS —
// le petit octet de la valeur borne l'enumeration : {10, 20, 50, 100, 150, 200, 220}, rien
// d'autre sur les neuf films. **L'armement n'est recompense par AUCUN score personnel.** C'est
// coherent avec le statborg (bande de mode vide) et avec la nature du mode : un mode SCRIPTE
// (Lua `primitive_carriable_arming_base`), dont les recompenses passent par le script.
//
// Les +150/+200/+220 sont des recompenses de MEDAILLE ou de fin de partie (la salve de +150 aux
// quatre joueurs de l'equipe gagnante a +25 ms de la derniere explosion de `c75f33b8`), pas des
// evenements de mode. Leur nommage exact passera par l'oracle des medailles cote base — un
// chantier a part, sans urgence.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedClassement -v -timeout 30m

import (
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// pcFenetreDetonMS borne la coincidence recompense-explosion : l'oracle A0.3 date le point de
// score, la prime tombe dans la meme seconde.
const pcFenetreDetonMS = 1500

// pcRecompense est un bloc du pied enrichi de l'identite de son acteur.
type pcRecompense struct {
	t, valeur int
	tag       string
	xuid      uint64
}

// TestAssautPiedClassement identifie la prime de detonation puis cherche la prime de pose.
func TestAssautPiedClassement(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedClassement", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — classement interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	films := make([]string, 0, len(afExplosions))
	for id := range afExplosions {
		films = append(films, id)
	}
	sort.Strings(films)

	type deton struct {
		film      string
		ms        int
		acteurTag string
		xuid      uint64
	}
	couvertureParValeur := map[int]int{}
	var detons []deton
	total := 0
	// posesParDelai[valeur] = les delais pose->explosion observes pour cette valeur.
	posesParDelai := map[int][]float64{}
	posesActeur := map[int]int{}

	for _, id := range films {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		recs := pcRecompenses(footer)
		total += len(afExplosions[id])
		for _, ms := range afExplosions[id] {
			// TEMPS 1 : les recompenses coincidentes, valeur par valeur.
			vus := map[int]bool{}
			var acteur *pcRecompense
			for i := range recs {
				r := &recs[i]
				if abs(r.t-ms) > pcFenetreDetonMS {
					continue
				}
				if !vus[r.valeur] {
					vus[r.valeur] = true
					couvertureParValeur[r.valeur]++
				}
				// L'acteur candidat de la detonation : la recompense INDIVIDUELLE la plus
				// proche (les salves d'equipe partagent le meme instant ; elles se
				// reconnaissent au nombre d'acteurs et s'ecartent au temps 2 par la
				// constance).
				if acteur == nil || abs(r.t-ms) < abs(acteur.t-ms) {
					acteur = r
				}
			}
			if acteur != nil {
				detons = append(detons, deton{film: id, ms: ms, acteurTag: acteur.tag, xuid: acteur.xuid})
			}
			// TEMPS 2 : les recompenses au MEME acteur dans les 120 s precedentes.
			if acteur == nil {
				continue
			}
			for _, r := range recs {
				if r.tag != acteur.tag {
					continue
				}
				d := ms - r.t
				if d <= pcFenetreDetonMS || d > 120000 {
					continue
				}
				posesParDelai[r.valeur] = append(posesParDelai[r.valeur], float64(d))
				posesActeur[r.valeur]++
			}
		}
	}

	t.Logf("########## TEMPS 1 — %d explosions ; couverture par VALEUR de recompense (fenetre %d ms) :",
		total, pcFenetreDetonMS)
	pcTableCouverture(t, couvertureParValeur, total)
	t.Logf("ACTEURS DES DETONATIONS (recompense individuelle la plus proche) :")
	for _, d := range detons {
		t.Logf("  %-9s %7d ms  %-20s xuid %d", d.film, d.ms, d.acteurTag, d.xuid)
	}

	t.Logf("########## TEMPS 2 — recompenses au poseur dans les 120 s AVANT son explosion :")
	vals := make([]int, 0, len(posesParDelai))
	for v := range posesParDelai {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	for _, v := range vals {
		ds := posesParDelai[v]
		med, cv := afMedianeEtCV(ds)
		verdict := ""
		if len(ds) >= total*3/4 && cv <= afCVMax {
			verdict = "   *** DELAI CONSTANT — CANDIDATE POSE, meche mediane ***"
		}
		t.Logf("  valeur +%-3d : %2d occurrence(s), delai median %6.1f s, dispersion %5.1f %%%s",
			v, len(ds), med/1000, cv*100, verdict)
	}
}

// pcRecompenses rend les blocs du pied avec valeur, gamertag et XUID adjacent.
func pcRecompenses(data []byte) []pcRecompense {
	blocs := paBlocsOctets(data)
	out := make([]pcRecompense, 0, len(blocs))
	total := len(data) * 8
	_ = total
	for _, b := range blocs {
		out = append(out, pcRecompense{
			t:      b.t,
			valeur: paValeur(b.oct[:]),
			tag:    paTag(b.oct[:]),
			xuid:   0, // le XUID adjacent est resolu par le gamertag : meme tag = meme acteur
		})
	}
	return out
}

// pcTableCouverture imprime la couverture par valeur, triee par couverture decroissante.
func pcTableCouverture(t *testing.T, m map[int]int, total int) {
	t.Helper()
	type l struct{ v, n int }
	ls := make([]l, 0, len(m))
	for v, n := range m {
		ls = append(ls, l{v, n})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].n != ls[j].n {
			return ls[i].n > ls[j].n
		}
		return ls[i].v < ls[j].v
	})
	for _, x := range ls {
		marque := ""
		if x.n == total {
			marque = "   <- couvre TOUTES les explosions"
		}
		t.Logf("  valeur +%-3d : %2d/%d explosions%s", x.v, x.n, total, marque)
	}
	if len(ls) == 0 {
		t.Logf("  (aucune recompense dans les fenetres)")
	}
}

// abs de l'entier — le paquet en porte deja un pour extractCTF ; celui-ci vit dans les tests.
func absPc(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var _ = math.Inf // garde l'import stable si la dispersion locale disparait
var _ = fmt.Sprintf
var _ = absPc

// TestAssautPiedEtendu — LE BLOC AU-DELA DES 60 OCTETS : ou vit le TYPE de la recompense.
//
// La fenetre de 60 octets tronque le debut du bloc (le gamertag y commence a des positions
// variables), et ni la valeur ni les octets de queue ne discriminent la pose du frag. Le champ
// de TYPE est donc EN AMONT. Cet instrument imprime 120 octets avant le marqueur pour les blocs
// des fenetres d'explosion, en marquant la position du XUID (l'ancre de production) s'il s'y
// trouve : la region entre le XUID et le gamertag est la candidate naturelle du type.
//
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedEtendu -v -timeout 30m
func TestAssautPiedEtendu(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedEtendu", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — releve interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	for _, id := range []string{"9f57c612"} {
		src, ok := afOuvrir(t, cache, id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		marqueurs := pcMarqueurs(footer)
		t.Logf("########## %s — fenetres de +-3 s, 120 octets avant marqueur :", id)
		for _, ms := range afExplosions[id] {
			t.Logf("  EXPLOSION a %d ms :", ms)
			for _, mk := range marqueurs {
				if mk.t < ms-3000 || mk.t > ms+3000 {
					continue
				}
				t.Logf("    t=%7d (%+5d) v=+%d", mk.t, mk.t-ms, mk.valeur)
				t.Logf("      %s", paHex(mk.avant[:60]))
				t.Logf("      %s", paHex(mk.avant[60:]))
			}
		}
	}
}

// pcMarqueur est un bloc etendu : 120 octets avant le marqueur de fin.
type pcMarqueur struct {
	t, valeur int
	avant     [120]byte
}

// pcMarqueurs rend les blocs dedoublonnes avec leur fenetre etendue.
func pcMarqueurs(data []byte) []pcMarqueur {
	total := len(data) * 8
	var out []pcMarqueur
	for p := 120 * 8; p+32 <= total; p++ {
		if readByteAtBit(data, p) != 0 || readByteAtBit(data, p+8) != 0 ||
			readByteAtBit(data, p+16) != 0x2e || readByteAtBit(data, p+24) != 0xe0 {
			continue
		}
		ebs := p - 60*8
		th := int(readByteAtBit(data, ebs+47*8))
		if th == 0 || th > 250 {
			continue
		}
		tms := int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 |
			int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8))
		if tms < 0 || tms > 4*3600*1000 {
			continue
		}
		var m pcMarqueur
		m.t = tms
		m.valeur = int(readByteAtBit(data, ebs+44*8))<<24 | int(readByteAtBit(data, ebs+45*8))<<16 |
			int(readByteAtBit(data, ebs+46*8))<<8 | th
		debut := p - 120*8
		for i := 0; i < 120; i++ {
			m.avant[i] = readByteAtBit(data, debut+i*8)
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// TestAssautPiedRares — LE RECENSEMENT COMPLET DES RECOMPENSES RARES (+150 et au-dela).
//
// Les valeurs +20/+50/+100 sont du combat (paires simultanees dans tous les modes). Les RARES
// sont les recompenses de MODE : la salve de +150 a quatre joueurs juste apres une explosion en
// est une. Cet instrument imprime CHAQUE bloc de valeur >= 150 des neuf films, avec le delai a
// l'explosion la plus proche (avant ET apres) — c'est le tableau qui nomme.
//
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedRares -v -timeout 30m
func TestAssautPiedRares(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedRares", filmproc.MeasureLimitGiB, func(peak uint64) {
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
		recs := pcRecompenses(footer)
		t.Logf("########## %s — explosions a %v :", id, afExplosions[id])
		for _, r := range recs {
			if r.valeur < 150 {
				continue
			}
			t.Logf("  t=%7d  +%-3d  %-20s  %s", r.t, r.valeur, r.tag, pcContexte(r.t, afExplosions[id]))
		}
	}
}

// pcContexte situe un instant par rapport aux explosions du film.
func pcContexte(t int, exps []int) string {
	meilleurAvant, meilleurApres := -1, -1
	for _, ms := range exps {
		if d := ms - t; d >= 0 && (meilleurAvant < 0 || d < meilleurAvant) {
			meilleurAvant = d
		}
		if d := t - ms; d >= 0 && (meilleurApres < 0 || d < meilleurApres) {
			meilleurApres = d
		}
	}
	out := ""
	if meilleurAvant >= 0 {
		out += fmt.Sprintf("%d ms AVANT la prochaine explosion", meilleurAvant)
	}
	if meilleurApres >= 0 {
		if out != "" {
			out += " ; "
		}
		out += fmt.Sprintf("%d ms APRES la precedente", meilleurApres)
	}
	if out == "" {
		return "(hors fenetre)"
	}
	return out
}

// TestAssautPiedTemoinsTags — LE GAIN TRANSVERSE : les evenements de mode des AUTRES modes
// portent-ils aussi leur gamertag ?
//
// La production attribue les evenements th=10 (zone, colline, crane) par le pont d'identite
// slot->XUID, resolu par les morts. Si le bloc porte le gamertag EN CLAIR, l'attribution
// devient directe — pour tous les modes, sans pont.
//
//	go test ./internal/analysis/objectiveevents/ -run AssautPiedTemoinsTags -v -timeout 30m
func TestAssautPiedTemoinsTags(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestAssautPiedTemoinsTags", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — releve interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	for _, w := range []struct{ id, mode string }{
		{"43716616", "Oddball"}, {"7f1bbf06", "KOTH"}, {"696a9d7c", "Strongholds"},
	} {
		src, ok := afOuvrir(t, cache, w.id)
		if !ok {
			continue
		}
		footer, ok := footerData(src)
		if !ok {
			continue
		}
		recs := pcRecompenses(footer)
		n, avecTag := 0, 0
		exemples := ""
		for _, r := range recs {
			if r.valeur != 10 {
				continue
			}
			n++
			if r.tag != "(sans tag)" {
				avecTag++
				if avecTag <= 4 {
					exemples += fmt.Sprintf(" [t=%d %s]", r.t, r.tag)
				}
			}
		}
		t.Logf("%-9s %-12s : %d evenement(s) +10, %d avec gamertag en clair —%s",
			w.id, w.mode, n, avecTag, exemples)
	}
}
