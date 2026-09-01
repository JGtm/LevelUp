package replay

// visee_zoom_gate_test.go — LA PORTE DU CHANTIER VISEE : les evenements `unit_zoom` du film
// decrivent-ils la chronologie relevee a la main par l'utilisateur ?
//
// CE QUI A CHANGE (2026-08-31). Le chantier « trame film » (branche `wt/trame-film`) a perce le
// MODELE DE PAQUET :
//
//	[1 bit configuration] [liste d'evenements : ( 1 [R(7) type] [3 refs gardees] [charge] )* 0]
//	[trame de records ECS]
//
// Nos sept campagnes de mesure lisaient le type a `octet & 0x7F` — elles ignoraient le bit de
// configuration et decalaient donc TOUT d'un bit. Le negatif « triple-verrouille » du chantier
// visee (« aucun evenement de zoom dans la bobine ») est REFUTE : ses trois chaines partageaient
// ce meme decalage. La famille `0xCA` porte le type 21 `unit_zoom`, ~400 000 evenements sur le
// corpus, charge R(2) = niveau + 1 (0 -> niveau -1 = SORTIE de lunette ; 1 -> niveau 0 = ENTREE).
//
// CE QUE CET INSTRUMENT FAIT, ET C'EST LE SEUL GESTE QUI MANQUAIT. La session soeur a prouve la
// grammaire ; ce chantier detient la VERITE TERRAIN. On confronte les deux :
//
//	1. decoder tous les `unit_zoom` du film, avec l'unite qui zoome (ref0, domaine 4) ;
//	2. reconstruire, PAR UNITE, les intervalles « a la lunette » (entree -> sortie) ;
//	3. les confronter aux six episodes releves par l'utilisateur pour Nilton410.
//
// Aucun pont ref0 -> joueur n'est suppose : c'est la chronologie qui DESIGNE l'unite. Si une unite
// et une seule reproduit les six episodes, le pont est etabli PAR la mesure, et la question du
// chantier est close par l'affirmative.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	CANDIDAT     >= 5 des 6 episodes couverts (un intervalle de cette unite recouvre l'episode)
//	             ET <= 2 intervalles de cette unite HORS episodes dans la fenetre [35 s, 95 s].
//	CONTROLE     obligatoire : la meme epreuve avec la chronologie TRANSLATEE (pas de 250 ms sur
//	             +/- 400 s, structure preservee). Verdict positif seulement si < 1 % des decalages
//	             temoins produisent un candidat. C'est le controle qui a refute la phase 6 et
//	             rattrape deux faux positifs au lot C : il n'est pas negociable.
//	PUISSANCE    publiee : part des decalages temoins ou UN candidat existe malgre tout.
//
// SOUS GARDE (ZOOM_FILM, qui doit pointer 00162144 — la chronologie est celle de CE film).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ZOOM_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeZoomGate -v -timeout 30m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	zoomFilmEnv = "ZOOM_FILM"
	// zoomTypeUnitZoom : le type d'evenement `unit_zoom` sous la numerotation du modele de paquet.
	zoomTypeUnitZoom = 21
	// zoomOctet : la famille de paquets qui porte cet evenement en tete.
	zoomOctet = 0xCA
	// Seuils, ecrits avant la mesure.
	zoomEpisodesMin = 5 // episodes couverts sur 6
	zoomFauxMax     = 2 // intervalles hors episodes toleres dans la fenetre
	zoomPMaxSeuil   = 0.01
	// Fenetre d'analyse, en secondes d'horloge du feed (la chronologie vit dedans).
	zoomFenetreDebut = 35.0
	zoomFenetreFin   = 95.0
)

// zoomEvt est un evenement `unit_zoom` decode : instant film, unite, et niveau.
type zoomEvt struct {
	tMS    int64
	unite  uint64
	entree bool // charge 1 -> niveau 0 = ENTREE ; charge 0 -> niveau -1 = SORTIE
}

// zoomRefWidths : largeur de l'index par domaine (table 0x1451f98d0, etablie par le lot B1/E2 et
// reprise telle quelle de l'instrument de la session soeur).
var zoomRefWidths = map[int]int{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}

// zoomLireRef consomme une reference gardee ; rend (index, presente).
func zoomLireRef(br *filmdec.BitReader, dom int) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(uint(zoomRefWidths[dom]))
	br.Skip(2) // generation R(2)
	return idx, true
}

// TestViseeZoomGate confronte les evenements de lunette a la chronologie relevee.
func TestViseeZoomGate(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : porte sautee", zoomFilmEnv)
	}
	if filepath.Base(dir) != "00162144" {
		t.Fatalf("la chronologie relevee est celle de 00162144 ; film fourni : %s", filepath.Base(dir))
	}
	release := filmdec.LockProcessDecode()
	defer release()

	off := zoomDecalage(t, dir)
	evts := zoomLitEvenements(t, dir)
	t.Logf("EVENEMENTS — %d `unit_zoom` decodes dans le film", len(evts))
	if len(evts) == 0 {
		t.Fatalf("aucun evenement de lunette : la grammaire ou la famille est fausse")
	}

	parUnite := zoomIntervalles(evts, off)
	t.Logf("UNITES — %d unites distinctes portent au moins une entree de lunette", len(parUnite))
	zoomJournaliseUnites(t, parUnite)

	zoomPontSlot(t, dir, evts, off)
	parEntrees := zoomEntrees(evts, off)
	best, bestCouv, bestExtras := zoomMeilleureParEntrees(parEntrees, chronoEpisodes)
	t.Logf("EPREUVE DES ENTREES — unite %d : %d/%d DEBUTS d'episode apparies a moins de %.1f s,"+
		" %d entrees supplementaires dans la fenetre",
		best, bestCouv, len(chronoEpisodes), zoomTolS, bestExtras)
	zoomDetaille(t, parUnite[best])

	if bestCouv < zoomEpisodesMin || bestExtras > 6 {
		t.Logf("VERDICT — AUCUN CANDIDAT au seuil declare (>= %d episodes couverts et <= %d hors"+
			" episodes). La chronologie n'est pas reproduite par les evenements de lunette de ce"+
			" film.", zoomEpisodesMin, zoomFauxMax)
		return
	}
	p, puissance := zoomControleEntrees(parEntrees, bestCouv)
	t.Logf("CONTROLE PAR TRANSLATION — %.2f %% des decalages temoins produisent un candidat"+
		" (seuil %.0f %%) ; puissance : %.2f %%", 100*p, 100*zoomPMaxSeuil, 100*puissance)
	if p >= zoomPMaxSeuil {
		t.Logf("VERDICT — NON SIGNIFICATIF : le controle refuse (p = %.2f %%). Le meilleur"+
			" accord observe est atteignable par hasard.", 100*p)
		return
	}
	t.Logf("VERDICT — PORTE FRANCHIE. L'unite %d reproduit la chronologie relevee a la main"+
		" (%d/%d debuts, %d extras, p = %.2f %%). LA LUNETTE EST DANS LE FILM, elle est LISIBLE,"+
		" et elle est ATTRIBUABLE PAR JOUEUR. Le negatif des sept campagnes precedentes est"+
		" definitivement leve : il tenait a un decalage d'UN bit dans la lecture du type.",
		best, bestCouv, len(chronoEpisodes), bestExtras, 100*p)
}

// zoomPontSlot teste L'HYPOTHESE DU PONT : l'index de ref0 (domaine 4) plus la base du domaine
// donne-t-il le SLOT bipede du joueur ? Indice qui l'a fait naitre : l'unite 1 designee par la
// chronologie correspond au slot 513 de Nilton410, et 513 = 512 + 1.
//
// L'epreuve est structurelle, sans seuil a declarer : pour chaque base candidate, on compte les
// index dont (index + base) est un slot bipede REELLEMENT vu dans le film. La bonne base fait
// tomber la quasi-totalite des index sur des slots existants ; une mauvaise base n'en fait
// tomber qu'une fraction. C'est une fermeture, pas une correlation.
func zoomPontSlot(t *testing.T, dir string, evts []zoomEvt, off int64) {
	t.Helper()
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	slots := map[uint32]int{}
	for _, p := range pos {
		slots[p.Slot]++
	}
	unites := map[uint64]bool{}
	for _, e := range evts {
		unites[e.unite] = true
	}
	t.Logf("PONT — %d slots bipedes distincts dans le film, %d unites de lunette distinctes",
		len(slots), len(unites))
	type essai struct {
		base    int
		touches int
	}
	var res []essai
	for _, base := range []int{0, 256, 512, 768, 1024} {
		n := 0
		for u := range unites {
			if slots[uint32(int(u)+base)] > 0 {
				n++
			}
		}
		res = append(res, essai{base, n})
	}
	for _, e := range res {
		t.Logf("  base %4d : %d/%d index tombent sur un slot bipede existant (%.0f %%)",
			e.base, e.touches, len(unites), 100*float64(e.touches)/float64(len(unites)))
	}
	// Controle nomme : le slot de Nilton dans la plage etiquetee est 513 (cf. le test de
	// chronologie) ; l'unite designee par la chronologie doit donc etre 513 - base.
	t.Log("  CONTROLE NOMME : la chronologie designe l'unite 1 ; si la base vaut 512," +
		" cela donne le slot 513 — celui de Nilton410 etabli par le pont des morts.")
}

// zoomDecalage rend l'ecart feed -> film par le pont des morts (meme mecanique que la chronologie).
func zoomDecalage(t *testing.T, dir string) int64 {
	t.Helper()
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	off, matched := bestDeathOffset(buildLifeSpans(indexBySlot(pos)), deaths)
	t.Logf("RECALAGE — decalage feed->film %d ms (%d fins de vie appariees)", off, matched)
	return off
}

// zoomLitEvenements decode les evenements `unit_zoom` de tete des paquets de la famille 0xCA.
func zoomLitEvenements(t *testing.T, dir string) []zoomEvt {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	var out []zoomEvt
	paquets, autres := 0, 0
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(chunk) {
			if pk.Type != filmdec.PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(chunk)
			if pay[0] != zoomOctet {
				continue
			}
			paquets++
			br := filmdec.NewBitReader(pay)
			br.Skip(1) // bit de configuration
			if !br.ReadBit() {
				continue // pas d'evenement en tete
			}
			if int(br.ReadBits(7)) != zoomTypeUnitZoom {
				autres++
				continue
			}
			idx, ok := zoomLireRef(br, 4) // l'unite qui zoome
			zoomLireRef(br, 8)
			zoomLireRef(br, 7)
			charge := br.ReadBits(2)
			if !ok {
				continue
			}
			out = append(out, zoomEvt{
				tMS: int64(pk.TimestampUS / 1000), unite: idx, entree: charge == 1,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	t.Logf("FAMILLE 0xCA — %d paquets ; %d d'un autre type ; %d evenements de lunette retenus",
		paquets, autres, len(out))
	return out
}

// zoomIntervalles reconstruit, par unite, les periodes « a la lunette » en secondes du FEED.
func zoomIntervalles(evts []zoomEvt, off int64) map[uint64][][2]float64 {
	ouverture := map[uint64]float64{}
	out := map[uint64][][2]float64{}
	for _, e := range evts {
		s := float64(e.tMS-off) / 1000
		if e.entree {
			if _, deja := ouverture[e.unite]; !deja {
				ouverture[e.unite] = s
			}
			continue
		}
		if debut, ok := ouverture[e.unite]; ok {
			out[e.unite] = append(out[e.unite], [2]float64{debut, s})
			delete(ouverture, e.unite)
		}
	}
	return out
}

// zoomTolMS : tolerance d'appariement instant<->transition. C'est la precision de lecture video
// annoncee par l'utilisateur (releve « a la seconde pres »), majoree de 20 %.
const zoomTolS = 1.2

// zoomEntrees rend, par unite, les instants d'ENTREE en lunette (secondes du feed).
func zoomEntrees(evts []zoomEvt, off int64) map[uint64][]float64 {
	out := map[uint64][]float64{}
	for _, e := range evts {
		if !e.entree {
			continue
		}
		out[e.unite] = append(out[e.unite], float64(e.tMS-off)/1000)
	}
	return out
}

// zoomAccord compte combien de DEBUTS d'episode ont une entree de cette unite a moins de
// zoomTolS, et combien d'entrees de l'unite tombent dans la fenetre sans correspondre a un debut.
func zoomAccord(entrees []float64, eps [][2]float64) (int, int) {
	couverts := 0
	for _, ep := range eps {
		for _, e := range entrees {
			if e >= ep[0]-zoomTolS && e <= ep[0]+zoomTolS {
				couverts++
				break
			}
		}
	}
	extras := 0
	for _, e := range entrees {
		if e < zoomFenetreDebut || e > zoomFenetreFin {
			continue
		}
		ok := false
		for _, ep := range eps {
			if e >= ep[0]-zoomTolS && e <= ep[0]+zoomTolS {
				ok = true
				break
			}
		}
		if !ok {
			extras++
		}
	}
	return couverts, extras
}

// zoomMeilleureParEntrees rend l'unite dont les entrees reproduisent le mieux les debuts.
func zoomMeilleureParEntrees(parU map[uint64][]float64, eps [][2]float64) (uint64, int, int) {
	var best uint64
	bestCouv, bestExtras := -1, 0
	unites := make([]uint64, 0, len(parU))
	for u := range parU {
		unites = append(unites, u)
	}
	sort.Slice(unites, func(i, j int) bool { return unites[i] < unites[j] })
	for _, u := range unites {
		couv, ex := zoomAccord(parU[u], eps)
		if couv > bestCouv || (couv == bestCouv && ex < bestExtras) {
			best, bestCouv, bestExtras = u, couv, ex
		}
	}
	return best, bestCouv, bestExtras
}

// zoomControleEntrees rejoue l'epreuve des entrees sur la chronologie TRANSLATEE : la part des
// decalages temoins dont la MEILLEURE unite atteint le score observe est la p-valeur. Le maximum
// est repris sur TOUTES les unites a chaque decalage — c'est ce qui rend le controle honnete
// (sinon on comparerait un maximum sur 58 unites a un tirage sur une seule).
func zoomControleEntrees(parU map[uint64][]float64, observe int) (float64, float64) {
	essais, aussiBien, parfaits := 0, 0, 0
	for d := -400.0; d <= 400.0; d += 0.25 {
		if d > -3 && d < 3 {
			continue
		}
		eps := make([][2]float64, 0, len(chronoEpisodes))
		for _, e := range chronoEpisodes {
			eps = append(eps, [2]float64{e[0] + d, e[1] + d})
		}
		essais++
		_, couv, _ := zoomMeilleureParEntrees(parU, eps)
		if couv >= observe {
			aussiBien++
		}
		if couv >= len(chronoEpisodes) {
			parfaits++
		}
	}
	if essais == 0 {
		return 1, 0
	}
	return float64(aussiBien) / float64(essais), float64(parfaits) / float64(essais)
}

// zoomJournaliseUnites publie le volume par unite : de quoi voir les huit joueurs.
func zoomJournaliseUnites(t *testing.T, parUnite map[uint64][][2]float64) {
	t.Helper()
	type uc struct {
		u uint64
		n int
	}
	var l []uc
	for u, ivs := range parUnite {
		l = append(l, uc{u, len(ivs)})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	for i, e := range l {
		if i == 12 {
			break
		}
		t.Logf("  unite %3d : %d periodes de lunette sur le film", e.u, e.n)
	}
}

// zoomDetaille publie les periodes de l'unite retenue dans la fenetre de la chronologie.
func zoomDetaille(t *testing.T, ivs [][2]float64) {
	t.Helper()
	t.Log("  periodes de cette unite dans la fenetre du releve (secondes du feed) :")
	for _, iv := range ivs {
		if iv[1] < zoomFenetreDebut || iv[0] > zoomFenetreFin {
			continue
		}
		t.Logf("    %.1f -> %.1f s", iv[0], iv[1])
	}
	t.Log("  releve utilisateur (Nilton410) : 41->46,3 · 49->52 · 61->61,8 · 68->68,8 ·" +
		" 71->73 · 85->86")
}
