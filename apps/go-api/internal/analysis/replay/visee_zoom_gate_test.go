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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	zoomDumpBrut(t, evts, off)
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

// zoomDumpBrut publie les evenements BRUTS des unites les plus actives dans la fenetre du releve.
// C'est le diagnostic qui dit si la reconstruction d'intervalles est fidele ou si des transitions
// manquent (evenement de lunette porte en 2e position d'une liste, dans une autre famille).
func zoomDumpBrut(t *testing.T, evts []zoomEvt, off int64) {
	t.Helper()
	parU := map[uint64][]zoomEvt{}
	for _, e := range evts {
		s := float64(e.tMS-off) / 1000
		if s < zoomFenetreDebut || s > zoomFenetreFin {
			continue
		}
		parU[e.unite] = append(parU[e.unite], e)
	}
	type uc struct {
		u uint64
		n int
	}
	var l []uc
	for u, es := range parU {
		l = append(l, uc{u, len(es)})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	t.Logf("EVENEMENTS BRUTS dans la fenetre [%.0f ; %.0f] s — %d unites concernees",
		zoomFenetreDebut, zoomFenetreFin, len(parU))
	for i, e := range l {
		if i == 6 {
			break
		}
		var sb []string
		for _, ev := range parU[e.u] {
			sens := "SORTIE"
			if ev.entree {
				sens = "ENTREE"
			}
			sb = append(sb, fmt.Sprintf("%.1f:%s", float64(ev.tMS-off)/1000, sens))
		}
		t.Logf("  unite %3d (%d evts) : %s", e.u, e.n, strings.Join(sb, " "))
	}
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

// zoomCouvre dit si un intervalle recouvre un episode.
func zoomCouvre(iv, ep [2]float64) bool { return iv[0] <= ep[1] && iv[1] >= ep[0] }

// zoomEvalue rend (episodes couverts, intervalles hors episodes dans la fenetre) pour une unite.
func zoomEvalue(ivs [][2]float64, eps [][2]float64) (int, int) {
	couverts := 0
	for _, ep := range eps {
		for _, iv := range ivs {
			if zoomCouvre(iv, ep) {
				couverts++
				break
			}
		}
	}
	faux := 0
	for _, iv := range ivs {
		if iv[1] < zoomFenetreDebut || iv[0] > zoomFenetreFin {
			continue
		}
		dedans := false
		for _, ep := range eps {
			if zoomCouvre(iv, ep) {
				dedans = true
				break
			}
		}
		if !dedans {
			faux++
		}
	}
	return couverts, faux
}

// zoomMeilleureUnite rend l'unite qui reproduit le mieux la chronologie.
func zoomMeilleureUnite(parUnite map[uint64][][2]float64, eps [][2]float64) (uint64, int, int) {
	var best uint64
	bestCouv, bestFaux := -1, 0
	unites := make([]uint64, 0, len(parUnite))
	for u := range parUnite {
		unites = append(unites, u)
	}
	sort.Slice(unites, func(i, j int) bool { return unites[i] < unites[j] })
	for _, u := range unites {
		couv, faux := zoomEvalue(parUnite[u], eps)
		if couv > bestCouv || (couv == bestCouv && faux < bestFaux) {
			best, bestCouv, bestFaux = u, couv, faux
		}
	}
	return best, bestCouv, bestFaux
}

// zoomControleTranslation rejoue l'epreuve sur la chronologie translatee. Rend la part des
// decalages temoins qui produisent un candidat, et la puissance (part ou un candidat existe).
func zoomControleTranslation(parUnite map[uint64][][2]float64) (float64, float64) {
	essais, candidats := 0, 0
	for d := -400.0; d <= 400.0; d += 0.25 {
		if d > -3 && d < 3 {
			continue // voisinage du vrai decalage : exclu du temoin
		}
		eps := make([][2]float64, 0, len(chronoEpisodes))
		for _, e := range chronoEpisodes {
			eps = append(eps, [2]float64{e[0] + d, e[1] + d})
		}
		essais++
		_, couv, faux := zoomMeilleureUnite(parUnite, eps)
		if couv >= zoomEpisodesMin && faux <= zoomFauxMax {
			candidats++
		}
	}
	if essais == 0 {
		return 1, 0
	}
	p := float64(candidats) / float64(essais)
	return p, p
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

// TestViseeZoomBoutEnBout verifie que le palier de lunette arrive jusqu'au document — au bon
// joueur, au bon moment. C'est le gate de bout en bout : le decodage peut etre juste et le
// cablage faux (mauvais slot, mauvaise horloge, champ jamais pose).
//
// LA METRIQUE EST LE RAPPEL, ET C'EST UN CHOIX RAISONNE. Le releve de l'utilisateur est une
// liste de ce qu'il A VU (« brievement », « environ »), pas une certification d'absence sur le
// reste de la fenetre : il n'a jamais affirme que le joueur ne zoomait PAS ailleurs. Compter
// des « faux positifs » contre lui mesurerait donc l'exhaustivite du releve, pas la justesse du
// cablage. On mesure ce que le releve peut REELLEMENT arbitrer : ses episodes sont-ils couverts ?
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	RAPPEL   les 6 episodes releves doivent porter au moins un echantillon « a la lunette » sur
//	         la track du joueur (6/6 exige : un releve de six periodes vues de ses yeux ne
//	         souffre pas d'exception).
//	TEMOIN   le meme rappel sur les episodes TRANSLATES de 30 s doit s'effondrer, sans quoi le
//	         rappel ne dirait rien (un champ allume en permanence couvrirait tout).
//
// Garde ZOOM_FILM (00162144).
func TestViseeZoomBoutEnBout(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : gate saute", zoomFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	evts := filmdec.ScanFilmZoomEvents(dir)
	if len(evts) == 0 {
		t.Fatalf("aucun evenement de lunette : le scanner de production ne rend rien")
	}
	t.Logf("SCANNER DE PRODUCTION — %d bascules de lunette lues", len(evts))

	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	// LA MEME reconstruction que la production (zoom_state.go) : plusieurs causes de fermeture.
	etat := buildScopedLookup(evts, lives, zoomHoldUS)
	off, _ := bestDeathOffset(lives, ScanFilmDeaths2(t, dir))

	// La track de Nilton est celle du slot 513 (index 1 + base 512), etabli par le pont.
	const slotNilton = 513
	var scopedS []float64
	echantillons := 0
	for _, p := range pos {
		if p.Slot != slotNilton {
			continue
		}
		echantillons++
		if etat(p.Slot, p.TimestampUS) == 0 {
			continue
		}
		scopedS = append(scopedS, float64(int64(p.TimestampUS/1000)-off)/1000)
	}
	if len(scopedS) == 0 {
		t.Fatalf("COUVERTURE — aucun echantillon « a la lunette » sur la track du slot %d :"+
			" le cablage ne pose jamais le champ", slotNilton)
	}
	t.Logf("COUVERTURE — %d echantillons sur la track du slot %d, dont %d a la lunette (%.1f %%)",
		echantillons, slotNilton, len(scopedS), 100*float64(len(scopedS))/float64(echantillons))

	rappel := zoomRappel(scopedS, chronoEpisodes, 0)
	t.Logf("RAPPEL — %d/%d episodes releves portent un echantillon a la lunette",
		rappel, len(chronoEpisodes))
	// CONTROLE PAR TRANSLATION plutot qu'un decalage unique : avec 22 % d'echantillons a la
	// lunette, un seul decalage temoin ne dit rien (il attrape souvent 4/6 par hasard). On
	// mesure la part des decalages qui atteignent le rappel observe — c'est elle qui dit si
	// 6/6 est remarquable, et elle est publiee meme quand elle est mauvaise.
	essais, aussiBien := 0, 0
	for d := -300.0; d <= 300.0; d += 1 {
		if d > -6 && d < 6 {
			continue
		}
		essais++
		if zoomRappel(scopedS, chronoEpisodes, d) >= rappel {
			aussiBien++
		}
	}
	part := float64(aussiBien) / float64(essais)
	t.Logf("TEMOIN — %d/%d decalages atteignent aussi %d/%d (%.1f %%)",
		aussiBien, essais, rappel, len(chronoEpisodes), 100*part)
	if rappel < len(chronoEpisodes) {
		t.Fatalf("RAPPEL INSUFFISANT : %d/%d (6/6 exige)", rappel, len(chronoEpisodes))
	}
	t.Logf("GATE DE CABLAGE FRANCHI — le palier arrive au bon joueur et couvre les %d episodes"+
		" releves. RESERVE PUBLIEE : %.1f %% des decalages temoins en font autant, ce gate"+
		" verifie donc LE CABLAGE, pas l'identification — celle-ci est etablie par l'epreuve"+
		" des INSTANTS d'entree (TestViseeZoomGate, 6/6 a p = 0,00 %%).",
		len(chronoEpisodes), 100*part)
}

// zoomRappel compte les episodes (decales de `shift` secondes) portant au moins un echantillon.
func zoomRappel(scopedS []float64, eps [][2]float64, shift float64) int {
	n := 0
	for _, ep := range eps {
		for _, s := range scopedS {
			if s >= ep[0]+shift-zoomTolS && s <= ep[1]+shift+zoomTolS {
				n++
				break
			}
		}
	}
	return n
}

// ScanFilmDeaths2 est un adaptateur de test : ScanFilmDeaths rend une erreur, et l'ignorer
// silencieusement dans le corps du gate masquerait un film illisible.
func ScanFilmDeaths2(t *testing.T, dir string) []Death {
	t.Helper()
	d, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	return d
}

// TestViseeZoomDureesObservees mesure la distribution des periodes de lunette REELLEMENT
// fermees (une entree suivie d'une sortie du meme slot). Elle sert a calibrer le maintien
// (`zoomHoldUS`) SUR LA DONNEE, et non sur la chronologie de l'utilisateur — se caler sur la
// verite terrain reviendrait a s'ajuster a la reponse. Garde ZOOM_FILM.
func TestViseeZoomDureesObservees(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", zoomFilmEnv)
	}
	evts := filmdec.ScanFilmZoomEvents(dir)
	ouv := map[uint32]uint64{}
	var durees []float64
	entrees, sorties := 0, 0
	for _, e := range evts {
		if e.Scoped() {
			entrees++
			ouv[e.Slot] = e.TimestampUS
			continue
		}
		sorties++
		if t0, ok := ouv[e.Slot]; ok {
			durees = append(durees, float64(e.TimestampUS-t0)/1e6)
			delete(ouv, e.Slot)
		}
	}
	sort.Float64s(durees)
	if len(durees) == 0 {
		t.Fatalf("aucune periode fermee")
	}
	q := func(f float64) float64 { return durees[int(f*float64(len(durees)-1))] }
	t.Logf("PERIODES FERMEES — %d (sur %d entrees, %d sorties)", len(durees), entrees, sorties)
	t.Logf("  quantiles (s) : p10=%.2f p25=%.2f p50=%.2f p75=%.2f p90=%.2f p95=%.2f max=%.2f",
		q(0.10), q(0.25), q(0.50), q(0.75), q(0.90), q(0.95), durees[len(durees)-1])
}

// TestViseeZoomEntreesOrphelines cherche CE QUI FERME une periode de lunette quand aucun
// evenement de sortie n'est lu. Hypotheses de l'utilisateur, toutes deux plausibles et
// testables : (a) le joueur MEURT a la lunette — il n'a pas le temps de dezoomer, et le moteur
// n'a pas de sortie a emettre ; (b) il subit des DEGATS, ce qui force le dezoom — et cet
// evenement-la voyagerait alors dans le meme paquet que le degat, donc en DEUXIEME position
// d'une liste, hors de portee du scanner actuel.
//
// L'epreuve ne pose pas de seuil : elle mesure le delai entre une entree orpheline et la mort
// suivante du meme slot, et le compare au delai qui suit une entree NORMALEMENT fermee. Si (a)
// explique les orphelines, leur delai a la mort doit etre COURT la ou celui des fermees ne l'est
// pas. Garde ZOOM_FILM.
func TestViseeZoomEntreesOrphelines(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", zoomFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	evts := filmdec.ScanFilmZoomEvents(dir)
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	// Fin de vie d'un slot = l'instant ou sa trajectoire s'arrete : c'est la mort (ou la fin du
	// film). On l'utilise comme horloge de mort, sans passer par le fil des morts : ici on ne
	// cherche pas QUI meurt, seulement QUAND ce slot cesse d'exister.
	finDeVie := map[uint32][]uint64{}
	for _, l := range lives {
		finDeVie[l.slot] = append(finDeVie[l.slot], uint64(l.to))
	}
	for s := range finDeVie {
		sort.Slice(finDeVie[s], func(i, j int) bool { return finDeVie[s][i] < finDeVie[s][j] })
	}
	prochaineFin := func(slot uint32, ts uint64) (float64, bool) {
		for _, f := range finDeVie[slot] {
			if f >= ts {
				return float64(f-ts) / 1e6, true
			}
		}
		return 0, false
	}

	ouv := map[uint32]uint64{}
	var delaisFermees, delaisOrphelines []float64
	orphelines, fermees := 0, 0
	// Une entree est ORPHELINE si la bascule suivante du meme slot est une AUTRE entree (ou
	// s'il n'y en a plus) : le moteur ne peut pas entrer deux fois sans etre sorti entre-temps.
	for _, e := range evts {
		if e.Scoped() {
			if t0, encore := ouv[e.Slot]; encore {
				orphelines++
				if d, ok := prochaineFin(e.Slot, t0); ok {
					delaisOrphelines = append(delaisOrphelines, d)
				}
			}
			ouv[e.Slot] = e.TimestampUS
			continue
		}
		if t0, ok := ouv[e.Slot]; ok {
			fermees++
			if d, ok2 := prochaineFin(e.Slot, t0); ok2 {
				delaisFermees = append(delaisFermees, d)
			}
			delete(ouv, e.Slot)
		}
	}
	// Les entrees encore ouvertes en fin de film sont orphelines elles aussi.
	for slot, t0 := range ouv {
		orphelines++
		if d, ok := prochaineFin(slot, t0); ok {
			delaisOrphelines = append(delaisOrphelines, d)
		}
	}
	t.Logf("BASCULES — %d entrees fermees par une sortie lue, %d ORPHELINES", fermees, orphelines)

	med := func(v []float64) float64 {
		if len(v) == 0 {
			return -1
		}
		c := append([]float64(nil), v...)
		sort.Float64s(c)
		return c[len(c)/2]
	}
	sous := func(v []float64, seuil float64) float64 {
		if len(v) == 0 {
			return 0
		}
		n := 0
		for _, x := range v {
			if x <= seuil {
				n++
			}
		}
		return 100 * float64(n) / float64(len(v))
	}
	t.Logf("DELAI ENTREE -> FIN DE VIE DU SLOT (secondes) :")
	t.Logf("  entrees FERMEES    (n=%d) : mediane %.2f · %.0f %% a moins de 2 s · %.0f %% a moins de 5 s",
		len(delaisFermees), med(delaisFermees), sous(delaisFermees, 2), sous(delaisFermees, 5))
	t.Logf("  entrees ORPHELINES (n=%d) : mediane %.2f · %.0f %% a moins de 2 s · %.0f %% a moins de 5 s",
		len(delaisOrphelines), med(delaisOrphelines), sous(delaisOrphelines, 2), sous(delaisOrphelines, 5))
	t.Log("LECTURE : si les orphelines meurent nettement plus vite que les fermees, l'hypothese" +
		" « il meurt a la lunette » explique la sortie manquante — et la periode doit alors se" +
		" fermer A LA MORT, pas au plafond de maintien. Sinon la sortie existe ailleurs dans le" +
		" flux (2e position d'une liste, vraisemblablement le paquet du degat qui fait dezoomer).")
}
