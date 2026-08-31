package filmdec

// zoom_events_test.go — QUELLE PART DES EVENEMENTS DE LUNETTE LE SCANNER RATE-T-IL ?
//
// Le scanner de production (`ScanFilmZoomEvents`) ne lit que l'evenement de TETE de chaque
// paquet. Une liste peut en porter plusieurs : un dezoom PROVOQUE (degat recu, changement
// d'arme) voyagerait dans le paquet de sa cause, donc en 2e position, et echapperait a la
// lecture. C'est l'explication designee pour les deux tiers des entrees orphelines.
//
// CE QUE CET INSTRUMENT MESURE, ET IL N'A PAS BESOIN DE LA GRAMMAIRE DE TOUS LES TYPES : le bit
// de CONTINUATION qui suit chaque evenement est lisible sans savoir decoder les charges des
// autres types, DES LORS que l'evenement de tete est un `unit_zoom` (charge R(2), longueur
// connue). On mesure donc, sur la seule famille qu'on sait traverser, la frequence des listes
// MULTIPLES — et le type du deuxieme evenement quand il y en a un.
//
// C'est un DIMENSIONNEMENT, pas un verdict : il dit si le chantier « marcher la liste entiere »
// vaut la peine, et ce qu'il faudra savoir decoder pour y arriver.
//
// Garde ZOOM_EVT_FILM (un film ; ne jamais balayer le corpus — bombe RAM connue du depot).

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestZoomListesMultiples(t *testing.T) {
	dir := os.Getenv("ZOOM_EVT_FILM")
	if dir == "" {
		t.Skipf("ZOOM_EVT_FILM absent : mesure sautee")
	}
	paquets, avecSuite := 0, 0
	seconds := map[int]int{}
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(chunk) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(chunk)
			if pay[0] != zoomFamilyByte {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(1)
			if !br.ReadBit() {
				continue
			}
			if int(br.ReadBits(7)) != zoomEventType {
				continue
			}
			paquets++
			readZoomRef(br, zoomRefDomains[0])
			readZoomRef(br, zoomRefDomains[1])
			readZoomRef(br, zoomRefDomains[2])
			br.Skip(2) // la charge : R(2)
			if !br.ReadBit() {
				continue // fin de liste : l'evenement de tete etait seul
			}
			avecSuite++
			seconds[int(br.ReadBits(7))]++
		}
	}
	if paquets == 0 {
		t.Fatalf("aucun paquet de lunette dans %s", dir)
	}
	t.Logf("LISTES — %d paquets a evenement de lunette en tete ; %d portent un SECOND evenement"+
		" (%.1f %%)", paquets, avecSuite, 100*float64(avecSuite)/float64(paquets))
	type tc struct{ t, n int }
	var l []tc
	for ty, n := range seconds {
		l = append(l, tc{ty, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	for i, e := range l {
		if i == 10 {
			break
		}
		t.Logf("  2e evenement de type %3d : %d fois", e.t, e.n)
	}
	t.Log("LECTURE : une part elevee de listes multiples dit que le scanner de tete est aveugle" +
		" a une fraction du flux, et le palmares des seconds types dit CE QU'IL FAUDRA savoir" +
		" decoder pour marcher la liste entiere.")
}

// TestZoomStructureMultiFilms verifie que le decodage TIENT AILLEURS que sur le film qui a servi
// a l'etablir. Tout le chantier visee a ete valide sur 00162144 ; avant qu'un backfill ne cuise
// le parc, il faut savoir si la grammaire et le pont sont GENERAUX ou si l'on a ajuste un cas.
//
// Ce sont des CONTROLES DE STRUCTURE, pas de verite terrain (elle n'existe que sur 00162144) :
// chacun echoue par construction si le decodage derape, sans avoir besoin de savoir qui zoome.
//
//	C1 PURETE DE FAMILLE     tous les paquets 0xCA doivent porter le type 21 en tete. Un
//	                         decalage de lecture ferait apparaitre des types arbitraires.
//	C2 PALIERS               la charge R(2) donne le palier. La session soeur n'avait observe
//	                         que {0, 1} sur deux films et en avait deduit « entree / sortie ».
//	                         MESURE SUR QUATRE FILMS : les valeurs 2 et 3 EXISTENT, rares
//	                         (5 sur ~1 000). Ce ne sont pas des erreurs de lecture — ce sont les
//	                         PALIERS SUPERIEURS de lunette, ceux des armes qui en ont plusieurs
//	                         (le fusil de precision zoome a deux crans). Le controle verifie
//	                         donc la FORME de la distribution : dominee par {0, 1}, avec une
//	                         queue rare sur {2, 3} — un champ lu au mauvais endroit serait
//	                         reparti a peu pres uniformement sur les quatre valeurs.
//	C3 PONT VERS LE SLOT     index + 512 doit tomber sur un slot bipede EXISTANT dans une large
//	                         majorite des cas. C'est la fermeture qui a etabli la base : elle
//	                         doit se reproduire, sinon la base est un artefact du film temoin.
//	C4 EQUILIBRE             entrees et sorties doivent etre du meme ordre. Un desequilibre
//	                         massif dirait que l'un des deux sens n'est pas lu.
//
// Garde ZOOM_FILMS (repertoires separes par « ; »). JAMAIS le corpus entier — bombe RAM connue.
func TestZoomStructureMultiFilms(t *testing.T) {
	liste := os.Getenv("ZOOM_FILMS")
	if liste == "" {
		t.Skipf("ZOOM_FILMS absent : controle saute")
	}
	dirs := strings.Split(liste, ";")
	if len(dirs) < 2 {
		t.Fatalf("il faut au moins DEUX films : un seul ne dit rien de la generalite")
	}
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		nom := filepath.Base(dir)
		paquets, horsType := 0, 0
		entrees, sorties := 0, 0
		paliers := map[uint64]int{}
		indexVus := map[uint64]bool{}
		for c := 1; c <= CountFilmChunks(dir); c++ {
			chunk, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(chunk) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(chunk)
				if pay[0] != zoomFamilyByte {
					continue
				}
				paquets++
				br := NewBitReader(pay)
				br.Skip(1)
				if !br.ReadBit() {
					horsType++
					continue
				}
				if int(br.ReadBits(7)) != zoomEventType {
					horsType++
					continue
				}
				idx, ok := readZoomRef(br, zoomRefDomains[0])
				readZoomRef(br, zoomRefDomains[1])
				readZoomRef(br, zoomRefDomains[2])
				charge := br.ReadBits(2)
				paliers[charge]++
				if charge > 0 {
					entrees++
				} else {
					sorties++
				}
				if ok {
					indexVus[idx] = true
				}
			}
		}
		if paquets == 0 {
			t.Errorf("%s : AUCUN paquet de la famille — le film n'a pas de lunette, ou la"+
				" famille change de numero d'un build a l'autre", nom)
			continue
		}
		// Slots bipedes reellement vus, pour le controle du pont.
		scan := DefaultScanFilmOptions()
		scan.QuantaOnly = true
		pos, err := ScanFilmBipedPositions(dir, scan)
		if err != nil {
			t.Errorf("%s : positions illisibles : %v", nom, err)
			continue
		}
		slots := map[uint32]bool{}
		for _, p := range pos {
			slots[p.Slot] = true
		}
		touches := 0
		for idx := range indexVus {
			if slots[uint32(int(idx)+zoomSlotBase)] {
				touches++
			}
		}
		partPont := 100 * float64(touches) / float64(len(indexVus))
		superieurs := paliers[2] + paliers[3]
		partSup := 100 * float64(superieurs) / float64(paquets)
		t.Logf("%s — %d paquets · C1 hors-type %d · C2 paliers {0:%d 1:%d 2:%d 3:%d}"+
			" (superieurs %.1f %%) · C3 pont %d/%d (%.0f %%) · C4 %d entrees / %d sorties",
			nom, paquets, horsType, paliers[0], paliers[1], paliers[2], paliers[3], partSup,
			touches, len(indexVus), partPont, entrees, sorties)
		if horsType > 0 {
			t.Errorf("%s : C1 ECHOUE — %d paquets de la famille ne portent pas le type attendu",
				nom, horsType)
		}
		// C2 : la distribution doit etre DOMINEE par les deux premiers paliers. Une lecture au
		// mauvais endroit disperserait a peu pres uniformement (25 % par valeur).
		if partSup > 10 {
			t.Errorf("%s : C2 ECHOUE — %.1f %% de paliers superieurs (seuil 10 %%) : la charge"+
				" est vraisemblablement lue au mauvais endroit", nom, partSup)
		}
		if partPont < 90 {
			t.Errorf("%s : C3 ECHOUE — seulement %.0f %% des index tombent sur un slot bipede"+
				" (seuil 90 %%) : la base du domaine n'est pas generale", nom, partPont)
		}
		if sorties*2 < entrees {
			t.Errorf("%s : C4 ECHOUE — %d sorties pour %d entrees : un sens n'est pas lu",
				nom, sorties, entrees)
		}
	}
}
