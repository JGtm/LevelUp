package replay

// visee_typage_trame_research_test.go — LOTS D3 et D4 : CE QUE LE PREMIER OCTET D'UNE TRAME EST
// VRAIMENT, CONFRONTE A LA TABLE DE TYPES QUE LE FILM DECLARE.
//
// L'ETAT DU DOSSIER A L'OUVERTURE. Deux cadrages du premier octet coexistent dans le depot sans
// avoir jamais ete reconcilies (phase 7, « taxonomie NON TRANCHEE ») :
//   cadrage A — `type = octet >> 1`, le bit de poids faible etant une « variante » ;
//               c'est ce que lit `filmdec/fire_events.go`, et la chaine killsource marche 59/59 ;
//   cadrage B — `type = octet & 0x7F`, le bit de poids FORT etant le bit de continuation lu par
//               `FUN_14076a1c4` (l'ecrivain pose `acc = type | 0x80`, note NOTE_EMETTEUR B2b).
//
// CE QUE LE LOT D APPORTE COMME ARBITRE : le film DECLARE lui-meme sa table par type, juste
// apres le registre de composants (lot D1 : 123 entrees u32 avant la chaine de build). Un
// cadrage qui produit un type >= 123 sur du flux reel est donc refute PAR LE FILM.
//
// CRITERES ECRITS AVANT MESURE :
//   S1 — un cadrage est REFUTE si la part des paquets dont il implique un type >= 123 depasse
//        0,1 % du flux reel. En deca : NON CONCLUANT (un paquet tronque suffit a en fabriquer un).
//   S2 — le premier octet est un CHAMP D'ENUMERATION (et non de la donnee libre) si le nombre de
//        valeurs distinctes reste inferieur a 32 sur un corpus d'au moins 100 films. Au-dela de
//        64 valeurs distinctes : c'est de la donnee, et les deux cadrages tombent ensemble.
//   S3 — les paquets a premier octet minoritaire sont des EVENEMENTS greffes sur la trame de tick
//        si leur horodatage coincide avec celui d'un paquet majoritaire dans plus de 90 % des cas ;
//        ils sont des trames autonomes si la coincidence reste sous 10 %. Entre les deux :
//        NON CONCLUANT.
//   S4 — L'ARGUMENT DE LONGUEUR, et c'est lui qui tranche sans statistique. L'en-tete minimal
//        d'un evenement fait 11 bits sous les deux cadrages : 7 bits de type + 3 portes de
//        reference, plus le bit de continuation (cadrage B) ou le bit de variante (cadrage A).
//        11 bits ne tiennent pas dans un octet. Donc TOUT paquet delta d'UN SEUL octet refute,
//        pour ce paquet, l'idee que son premier octet soit un en-tete d'evenement. Le cadrage
//        « un paquet delta = un evenement » n'est retenu que si AUCUN paquet delta du corpus ne
//        fait moins de 2 octets. Il est REFUTE des qu'il en existe, et le compte est publie.
//
// Gardes TYPAGE_FILMS (liste `;`) et TYPAGE_CORPUS (racine du cache, avec TYPAGE_MAX).
// Aucun code de production touche.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// typesDeclaresParLeFilm est le nombre d'entrees de la table par type que porte chunk_00, mesure
// au lot D1 (`filmdec.TestD1TableParType`) : 123 valeurs u32 non nulles qui s'arretent pile a la
// chaine d'identification du build. C'est aussi, a l'octet pres, la borne du dispatcher relevee
// dans l'exe au lot B (`CMP R15,0x7b ; JNC`) — deux chaines sans etape commune.
const typesDeclaresParLeFilm = 123

// statOctet0 accumule ce qu'on sait d'une valeur de premier octet.
type statOctet0 struct {
	paquets              int
	tailleMin, tailleMax int
	tailleTotale         int
	memeInstantQuAutre   int
}

// TestD4CadrageOctet0 recense le premier octet des paquets delta et confronte les deux cadrages
// a la table de types declaree par le film.
func TestD4CadrageOctet0(t *testing.T) {
	dirs := typageDirs(t)
	stats := map[byte]*statOctet0{}
	total := 0
	for _, dir := range dirs {
		total += recenserFilm(dir, stats)
	}
	if total == 0 {
		t.Skipf("aucun paquet delta lu")
	}
	t.Logf("%d films, %d paquets delta", len(dirs), total)
	publierOctets(t, stats, total)
	verdictCadrages(t, stats, total, len(dirs))
	verdictS3(t, stats, total)
	verdictS4(t, stats, total)
}

// typageDirs rend la liste des repertoires de film a mesurer.
func typageDirs(t *testing.T) []string {
	t.Helper()
	if v := os.Getenv("TYPAGE_FILMS"); v != "" {
		var out []string
		for _, p := range strings.Split(v, ";") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	racine := os.Getenv("TYPAGE_CORPUS")
	if racine == "" {
		t.Skipf("ni TYPAGE_FILMS ni TYPAGE_CORPUS : instrument saute")
	}
	maxN := 0
	if v := os.Getenv("TYPAGE_MAX"); v != "" {
		maxN, _ = strconv.Atoi(v)
	}
	entries, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, filepath.Join(racine, e.Name()))
		if maxN > 0 && len(out) >= maxN {
			break
		}
	}
	return out
}

// recenserFilm accumule les statistiques de premier octet d'un film et rend le nombre de paquets
// delta lus.
func recenserFilm(dir string, stats map[byte]*statOctet0) int {
	n := filmdec.CountFilmChunks(dir)
	lus := 0
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		paquets := filmdec.WalkPackets(chunk)
		instants := map[uint64]int{}
		for _, p := range paquets {
			if p.Type == filmdec.PacketTypeDelta && p.Size >= 1 {
				instants[p.TimestampUS]++
			}
		}
		for _, p := range paquets {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			b := p.Payload(chunk)[0]
			s := stats[b]
			if s == nil {
				s = &statOctet0{tailleMin: p.Size}
				stats[b] = s
			}
			s.paquets++
			s.tailleTotale += p.Size
			if p.Size < s.tailleMin {
				s.tailleMin = p.Size
			}
			if p.Size > s.tailleMax {
				s.tailleMax = p.Size
			}
			if instants[p.TimestampUS] > 1 {
				s.memeInstantQuAutre++
			}
			lus++
		}
	}
	return lus
}

// publierOctets publie le recensement trie par frequence, avec les deux lectures de type.
func publierOctets(t *testing.T, stats map[byte]*statOctet0, total int) {
	t.Helper()
	var octets []byte
	for o := range stats {
		octets = append(octets, o)
	}
	sort.Slice(octets, func(i, j int) bool {
		return stats[octets[i]].paquets > stats[octets[j]].paquets
	})
	t.Logf("%d valeurs distinctes de premier octet", len(octets))
	t.Logf("  octet | paquets | part | taille (min/moy/max) | partage son instant | A:o>>1 | B:o&0x7F | grammaire de trame")
	for i, o := range octets {
		if i == 40 {
			t.Logf("  ... (%d autres valeurs)", len(octets)-40)
			break
		}
		s := stats[o]
		t.Logf("  0x%02X | %8d | %5.2f %% | %5d/%6d/%6d | %5.1f %% | %3d | %3d | %s",
			o, s.paquets, 100*float64(s.paquets)/float64(total),
			s.tailleMin, s.tailleTotale/s.paquets, s.tailleMax,
			100*float64(s.memeInstantQuAutre)/float64(s.paquets),
			o>>1, o&0x7f, lectureTrame(o))
	}
}

// verdictCadrages applique S1 et S2 : quel cadrage sort de la plage de types declaree.
func verdictCadrages(t *testing.T, stats map[byte]*statOctet0, total, films int) {
	t.Helper()
	horsA, horsB, contNul := 0, 0, 0
	var coupablesA, coupablesB []string
	for o, s := range stats {
		if int(o>>1) >= typesDeclaresParLeFilm {
			horsA += s.paquets
			coupablesA = append(coupablesA, fmt.Sprintf("0x%02X->%d", o, o>>1))
		}
		if int(o&0x7f) >= typesDeclaresParLeFilm {
			horsB += s.paquets
			coupablesB = append(coupablesB, fmt.Sprintf("0x%02X->%d", o, o&0x7f))
		}
		if o&0x80 == 0 {
			contNul += s.paquets
		}
	}
	sort.Strings(coupablesA)
	sort.Strings(coupablesB)
	t.Logf("S1 — cadrage A (o>>1) : %d paquets hors plage (%.4f %%) %s",
		horsA, 100*float64(horsA)/float64(total), strings.Join(coupablesA, " "))
	t.Logf("S1 — cadrage B (o&0x7F) : %d paquets hors plage (%.4f %%) %s",
		horsB, 100*float64(horsB)/float64(total), strings.Join(coupablesB, " "))
	t.Logf("S1 — verdict A : %s ; verdict B : %s",
		verdictS1(horsA, total), verdictS1(horsB, total))
	t.Logf("BIT DE POIDS FORT NUL (cadrage B : « aucun evenement dans cette trame ») : %d paquets"+
		" (%.4f %%)", contNul, 100*float64(contNul)/float64(total))
	verdict := "NON CONCLUANT (corpus trop petit)"
	if films >= 100 {
		switch {
		case len(stats) < 32:
			verdict = "CHAMP D'ENUMERATION"
		case len(stats) > 64:
			verdict = "DONNEE LIBRE — les deux cadrages tombent"
		default:
			verdict = "NON CONCLUANT (entre 32 et 64 valeurs)"
		}
	}
	t.Logf("S2 — %d valeurs distinctes sur %d films : %s", len(stats), films, verdict)
}

// verdictS3 applique le critere de coincidence : un evenement greffe sur une trame de tick
// partage l'horodatage de cette trame ; une trame autonome ne le partage avec personne.
func verdictS3(t *testing.T, stats map[byte]*statOctet0, total int) {
	t.Helper()
	partages := 0
	for _, s := range stats {
		partages += s.memeInstantQuAutre
	}
	part := 100 * float64(partages) / float64(total)
	verdict := "NON CONCLUANT (entre 10 et 90 %)"
	switch {
	case part > 90:
		verdict = "EVENEMENTS GREFFES sur une trame de tick"
	case part < 10:
		verdict = "TRAMES AUTONOMES — un paquet delta = un instant, il ne se greffe sur rien"
	}
	t.Logf("S3 — %d paquets partagent leur horodatage avec un autre paquet delta du meme chunk"+
		" (%.4f %%) : %s", partages, part, verdict)
}

// verdictS4 applique l'argument de longueur : un paquet delta d'un seul octet ne peut pas porter
// l'en-tete minimal d'un evenement (11 bits).
func verdictS4(t *testing.T, stats map[byte]*statOctet0, total int) {
	t.Helper()
	court := 0
	var coupables []string
	for o, s := range stats {
		if s.tailleMin < 2 {
			court += s.paquets
			coupables = append(coupables,
				fmt.Sprintf("0x%02X (min %d o, %d paquets)", o, s.tailleMin, s.paquets))
		}
	}
	sort.Strings(coupables)
	if court == 0 {
		t.Logf("S4 — aucun paquet delta de moins de 2 octets : l'argument de longueur ne refute rien")
		return
	}
	t.Logf("S4 — %d valeurs de premier octet apparaissent sur des paquets de moins de 2 octets : %s",
		len(coupables), strings.Join(coupables, " · "))
	t.Logf("S4 — VERDICT : « un paquet delta = un evenement dont le premier octet porte le type »"+
		" est REFUTE. Un octet ne contient pas les 11 bits minimaux d'un en-tete d'evenement,"+
		" et la lecture parcimonieuse d'un paquet 0x80 d'un octet est une TRAME VIDE"+
		" (amorce puis boucle de records close aussitot), ce que le decodeur de production lit deja."+
		" Part du flux concernee : %.4f %%", 100*float64(court)/float64(total))
}

// lectureTrame rend ce que la grammaire de trame du decodeur de production (amorce de 2 bits puis
// boucle de records) lit dans le premier octet : le type du PREMIER record de la trame.
func lectureTrame(o byte) string {
	bit := func(i uint) byte { return (o >> (7 - i)) & 1 }
	if bit(2) == 1 {
		return "DELTA"
	}
	switch bit(3)<<1 | bit(4) {
	case 0:
		return "FIN (vue vide)"
	case 1:
		return "NEW"
	case 2:
		return "DEL"
	default:
		return "DELTA(long)"
	}
}

// verdictS1 applique le seuil de refutation ecrit avant la mesure.
func verdictS1(hors, total int) string {
	if float64(hors) > 0.001*float64(total) {
		return "REFUTE (plus de 0,1 % du flux hors de la plage declaree par le film)"
	}
	if hors == 0 {
		return "non refute (aucun paquet hors plage)"
	}
	return "NON CONCLUANT (hors plage, mais sous 0,1 %)"
}

// TestD3CadenceDesTrames est le CONTROLE POSITIF du modele de trame, et la reponse directe a la
// question des « 125 paquets 114 » : si chaque paquet delta est une trame de tick, alors les
// paquets se succedent a cadence reguliere et les paquets a premier octet minoritaire occupent des
// places ORDINAIRES dans cette cadence. S'ils etaient des evenements, ils s'intercaleraient entre
// deux ticks et creeraient des ecarts hors cadence.
//
// CRITERE ECRIT AVANT LA MESURE : le modele de trame est confirme si l'ecart modal entre deux
// paquets delta consecutifs represente plus de 80 % des ecarts, ET si la part des paquets `0xE5`
// (les « 114 ») dont l'ecart amont vaut l'ecart modal ne s'ecarte pas de plus de 10 points de
// celle de l'ensemble. Une sur-representation des ecarts nuls ou tres courts chez `0xE5`
// signalerait au contraire un evenement intercale.
func TestD3CadenceDesTrames(t *testing.T) {
	dirs := typageDirs(t)
	ecarts := map[uint64]int{}
	ecartsE5 := map[uint64]int{}
	for _, dir := range dirs {
		n := filmdec.CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			var prec uint64
			premier := true
			for _, p := range filmdec.WalkPackets(chunk) {
				if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
					continue
				}
				if !premier && p.TimestampUS >= prec {
					d := p.TimestampUS - prec
					ecarts[d]++
					if chunk[p.Start] == 0xE5 {
						ecartsE5[d]++
					}
				}
				prec, premier = p.TimestampUS, false
			}
		}
	}
	modal, partTous := ecartModal(ecarts)
	_, partE5 := ecartModal(ecartsE5)
	t.Logf("ecart modal entre deux paquets delta consecutifs : %d us, %.2f %% des ecarts"+
		" (%d ecarts mesures)", modal, 100*partTous, sommeEcarts(ecarts))
	t.Logf("meme ecart chez les paquets 0xE5 (les « 114 ») : %.2f %% (%d ecarts)",
		100*partE5, sommeEcarts(ecartsE5))
	t.Logf("VERDICT : %s", verdictCadence(partTous, partE5))
}

// ecartModal rend l'ecart le plus frequent et sa part.
func ecartModal(m map[uint64]int) (uint64, float64) {
	total, meilleur, n := 0, uint64(0), 0
	for d, c := range m {
		total += c
		if c > n {
			n, meilleur = c, d
		}
	}
	if total == 0 {
		return 0, 0
	}
	return meilleur, float64(n) / float64(total)
}

// sommeEcarts rend le nombre d'ecarts mesures.
func sommeEcarts(m map[uint64]int) int {
	n := 0
	for _, c := range m {
		n += c
	}
	return n
}

// verdictCadence applique le critere ecrit avant la mesure.
func verdictCadence(partTous, partE5 float64) string {
	if partTous <= 0.80 {
		return "NON CONCLUANT — la cadence des paquets n'est pas assez reguliere pour servir de temoin"
	}
	if diff := partTous - partE5; diff > 0.10 || diff < -0.10 {
		return "LES 0xE5 SONT HORS CADENCE — ils s'intercalent, la lecture « evenement » reprend du poids"
	}
	return "LES 0xE5 SONT DANS LA CADENCE ORDINAIRE — ce sont des trames de tick comme les autres," +
		" pas des evenements intercales"
}

// TestD4BitBasEstUnBitDIdentifiant teste la lecture que la grammaire de trame impose au bit de
// poids faible du premier octet — celui dont le lot D4 devait etablir la semantique.
//
// SOUS LA GRAMMAIRE DE TRAME, ce bit (bit 7 du payload) tombe DANS l'identifiant du premier
// record : amorce 2 bits, `R(1)` de type 1 bit, `R(2)` de type 2 bits, donc l'identifiant commence
// au bit 5 et court sur `IDLowBits` bits (11 sur le film calibre). Le bit 7 en est le troisieme.
//
// CRITERE ECRIT AVANT LA MESURE. Si ce champ de 11 bits est un identifiant d'entite, il prend un
// nombre RESTREINT de valeurs — l'ordre de grandeur des entites repliquees d'un film, quelques
// centaines — et non les 2 048 valeurs d'un champ de bruit. Verdict : moins de 25 % des valeurs
// possibles occupees = IDENTIFIANT ; plus de 75 % = BRUIT ; entre les deux = NON CONCLUANT. Le
// meme champ est mesure sur l'ensemble des paquets et separement sur les paquets `0xD2` et `0xD3`,
// dont le lot D4 a montre qu'ils ne rendent pas la meme chose au meme offset.
func TestD4BitBasEstUnBitDIdentifiant(t *testing.T) {
	const debutID, largeurID = 5, 11
	dirs := typageDirs(t)
	vues := map[string]map[uint32]int{"tous": {}, "0xD2": {}, "0xD3": {}}
	totaux := map[string]int{}
	for _, dir := range dirs {
		n := filmdec.CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range filmdec.WalkPackets(chunk) {
				if p.Type != filmdec.PacketTypeDelta || p.Size < 4 {
					continue
				}
				pay := p.Payload(chunk)
				v := filmdec.ReadBitsAtForDiag(pay, debutID, largeurID)
				vues["tous"][v]++
				totaux["tous"]++
				if cle := fmt.Sprintf("0x%02X", pay[0]); vues[cle] != nil {
					vues[cle][v]++
					totaux[cle]++
				}
			}
		}
	}
	possibles := 1 << largeurID
	for _, cle := range []string{"tous", "0xD2", "0xD3"} {
		occ := len(vues[cle])
		part := 100 * float64(occ) / float64(possibles)
		verdict := "NON CONCLUANT"
		if part < 25 {
			verdict = "IDENTIFIANT (champ restreint)"
		} else if part > 75 {
			verdict = "BRUIT"
		}
		t.Logf("champ de %d bits au bit %d, paquets %s : %d valeurs distinctes sur %d possibles"+
			" (%.1f %%), %d paquets -> %s",
			largeurID, debutID, cle, occ, possibles, part, totaux[cle], verdict)
	}
}

// TestD3TypeZoomDansLeFilm repond a la question du lot D3 : un type « zoom » peut-il seulement
// s'exprimer dans un film ? Le test ne cherche pas un nom : il cherche les OCTETS qui coderaient
// le type 126 sous chacun des deux cadrages, et confronte le resultat a la table du film.
func TestD3TypeZoomDansLeFilm(t *testing.T) {
	dirs := typageDirs(t)
	const typeZoom = 126
	cibles := map[byte]string{
		byte(typeZoom << 1):         "cadrage A, variante 0",
		byte(typeZoom<<1) | 1:       "cadrage A, variante 1",
		byte(0x80 | typeZoom):       "cadrage B, continuation 1",
		byte(typeZoom) & byte(0x7f): "cadrage B, continuation 0",
	}
	trouves := map[byte]int{}
	total := 0
	for _, dir := range dirs {
		stats := map[byte]*statOctet0{}
		total += recenserFilm(dir, stats)
		for o, s := range stats {
			if _, ok := cibles[o]; ok {
				trouves[o] += s.paquets
			}
		}
	}
	t.Logf("%d films, %d paquets delta", len(dirs), total)
	for o, quoi := range cibles {
		t.Logf("  octet 0x%02X (%s) : %d paquets", o, quoi, trouves[o])
	}
	t.Logf("TABLE DU FILM : %d types declares (0..%d). Le type 126 est HORS de cette plage sous"+
		" les deux cadrages — il ne peut pas etre exprime par un film de ce build.",
		typesDeclaresParLeFilm, typesDeclaresParLeFilm-1)
}
