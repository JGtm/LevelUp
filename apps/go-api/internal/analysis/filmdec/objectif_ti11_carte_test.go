package filmdec

// objectif_ti11_carte_test.go — LA CARTE DU CORPS DE 104 BITS, BIT PAR BIT.
//
// # POURQUOI CETTE MESURE, ET POURQUOI ELLE NE SUPPOSE RIEN
//
// Les records `ti=11` d'image-cle font 168 bits dans 93,8 % des cas, soit un corps de 104 bits
// (`objectif_ti11_longueur_test.go`). Un corps de taille FIXE ne se lit pas comme un flux pilote
// par un masque : il se lit comme une STRUCTURE, et une structure se cartographie sans decodeur.
//
// La question utile n'est pas « que vaut ce champ » mais « ce champ BOUGE-T-IL ». Une jauge de
// capture est, par definition, la seule chose du record qui change au fil du match pour un MEME
// objectif. Cet instrument mesure donc, pour chacun des 104 bits :
//
//	VARIE DANS LE SLOT   part des objectifs pour lesquels ce bit prend les deux valeurs au fil
//	                     des images-cles — LA SIGNATURE D'UN ETAT VIVANT ;
//	PART A UN           part des records ou le bit vaut 1, tous slots confondus — le profil
//	                     statique, qui separe le remplissage des champs reellement utilises.
//
// # CE QUE CHAQUE ISSUE VOUDRAIT DIRE, ecrit avant la mesure
//
//	UNE PLAGE CONTIGUE DE BITS VARIE      c'est le champ vivant : sa position et sa largeur sont
//	                                      lues directement, sans desassemblage ;
//	AUCUN BIT NE VARIE                    le corps d'image-cle est une description STATIQUE, la
//	                                      jauge n'y est pas, et le chantier doit repartir sur
//	                                      l'ancrage des paquets DELTA ;
//	TOUS LES BITS VARIENT                 les records regroupes ne sont pas le meme objet — le
//	                                      couple (slot, generation) ne suffit pas a identifier
//	                                      une vie, et il faut d'abord reparer ca.
//
// Les trois sont des resultats. Aucun n'est un echec.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Carte -v -timeout 40m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// ti11CorpsBits est la taille du corps d'un record `ti=11` d'image-cle : 168 bits mesures moins
// les 64 bits d'en-tete.
const ti11CorpsBits = 168 - keyframeHeaderBits

// ti11LongueurRecord est la longueur DOMINANTE mesuree (93,8 % des records). Les records qui ne
// la respectent pas sont ECARTES : lire 104 bits sur un record plus court empieterait sur le
// suivant, et une carte batie sur ce melange ne voudrait rien dire.
const ti11LongueurRecord = 168

// ti11Corps porte les 104 bits d'un corps, dans deux mots.
type ti11Corps struct{ hi, lo uint64 }

// bit rend le bit de rang b (0 = premier bit du corps).
func (c ti11Corps) bit(b int) uint64 {
	if b < 64 {
		return c.hi >> uint(63-b) & 1
	}
	return c.lo >> uint(63-(b-64)) & 1
}

// ti11Vie regroupe les observations d'UNE vie d'objectif (slot + generation, dans un film).
type ti11Vie struct {
	ou    ti11Corps // OU logique de toutes les observations
	et    ti11Corps // ET logique — un bit qui differe entre les deux a VARIE
	n     int
	film  string
	slot  int
	gen   int
	corps ti11Corps // la premiere observation, pour le profil statique
}

// TestObjectifTi11Carte cartographie le corps des records d'image-cle de `ti=11`.
func TestObjectifTi11Carte(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Carte", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — carte interrompue", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	vies := map[string]*ti11Vie{}
	records := 0
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				records += ti11CartePayload(pk.Payload(data), f.id, vies)
			}
		}
	}

	var varie, unSur [ti11CorpsBits]int
	suivies, obs := 0, 0
	for _, v := range vies {
		obs += v.n
		for b := 0; b < ti11CorpsBits; b++ {
			if v.corps.bit(b) == 1 {
				unSur[b]++
			}
		}
		if v.n < 2 {
			continue
		}
		suivies++
		for b := 0; b < ti11CorpsBits; b++ {
			if v.ou.bit(b) != v.et.bit(b) {
				varie[b]++
			}
		}
	}
	t.Logf("########## %d record(s) retenus, %d vie(s) d'objectif, %d suivie(s) sur au moins deux "+
		"images-cles, %d observation(s)", records, len(vies), suivies, obs)
	if suivies == 0 {
		t.Logf("AUCUNE vie observee deux fois : la carte ne peut rien dire du mouvement.")
		return
	}
	t.Logf("BITS QUI VARIENT AU FIL DU MATCH (part des %d vies suivies) :", suivies)
	t.Logf("   %s", ti11CarteLigne(varie[:], suivies))
	t.Logf("PROFIL STATIQUE — part des vies dont le bit vaut 1 a la premiere observation :")
	t.Logf("   %s", ti11CarteLigne(unSur[:], len(vies)))
	ti11CarteVerdict(t, varie[:], suivies)
}

// ti11CartePayload releve les corps des records `ti=11` de longueur dominante d'UN payload.
func ti11CartePayload(pay []byte, film string, vies map[string]*ti11Vie) int {
	bits := len(pay) * 8
	n := 0
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ti11ArchIndex {
			continue
		}
		// LA LONGUEUR EST EXIGEE, pas supposee : sans elle on lirait dans le record suivant.
		h, ok := readKeyframeHeader(pay, r.Bit+ti11LongueurRecord, bits)
		if !ok || h.Slot <= r.Slot || r.Bit+ti11LongueurRecord > bits {
			continue
		}
		n++
		c := ti11Corps{
			hi: kfReadBits(pay, r.Bit+keyframeHeaderBits, 64),
			lo: kfReadBits(pay, r.Bit+keyframeHeaderBits+64, ti11CorpsBits-64) << uint(128-ti11CorpsBits),
		}
		cle := fmt.Sprintf("%s/%d/%d", film, r.Slot, r.Gen)
		v := vies[cle]
		if v == nil {
			vies[cle] = &ti11Vie{ou: c, et: c, n: 1, film: film, slot: r.Slot, gen: r.Gen, corps: c}
			continue
		}
		v.ou.hi, v.ou.lo = v.ou.hi|c.hi, v.ou.lo|c.lo
		v.et.hi, v.et.lo = v.et.hi&c.hi, v.et.lo&c.lo
		v.n++
	}
	return n
}

// ti11CarteLigne rend une ligne de 104 pourcentages, groupes par octet pour rester lisible.
func ti11CarteLigne(v []int, total int) string {
	var sb strings.Builder
	for b := 0; b < len(v); b++ {
		if b > 0 && b%8 == 0 {
			sb.WriteString(" | ")
		}
		fmt.Fprintf(&sb, "%3.0f ", ti11Part(v[b], total))
	}
	return sb.String()
}

// ti11CarteVerdict applique les trois lectures ecrites avant la mesure.
func ti11CarteVerdict(t *testing.T, varie []int, suivies int) {
	t.Helper()
	const seuil = 5.0 // un bit qui bouge sur moins de 5 % des vies n'est pas un champ vivant
	var mobiles []int
	for b, n := range varie {
		if ti11Part(n, suivies) >= seuil {
			mobiles = append(mobiles, b)
		}
	}
	if len(mobiles) == 0 {
		t.Logf("VERDICT : AUCUN bit ne varie sur plus de %.0f %% des vies. Le corps d'image-cle de "+
			"ti=11 est une description STATIQUE — la jauge n'y est pas, et le chantier doit "+
			"repartir sur l'ancrage des paquets DELTA.", seuil)
		return
	}
	if len(mobiles) > ti11CorpsBits/2 {
		t.Logf("VERDICT : %d bits sur %d varient. C'est trop pour une structure : les records "+
			"regroupes ne sont probablement pas le meme objet, et le couple (slot, generation) ne "+
			"suffit pas a identifier une vie.", len(mobiles), ti11CorpsBits)
		return
	}
	sort.Ints(mobiles)
	t.Logf("VERDICT : %d bit(s) varient au fil du match — %v", len(mobiles), mobiles)
	deb, fin := mobiles[0], mobiles[0]
	var plages []string
	for _, b := range mobiles[1:] {
		if b == fin+1 {
			fin = b
			continue
		}
		plages = append(plages, fmt.Sprintf("[%d..%d] (%d bits)", deb, fin, fin-deb+1))
		deb, fin = b, b
	}
	plages = append(plages, fmt.Sprintf("[%d..%d] (%d bits)", deb, fin, fin-deb+1))
	t.Logf("   PLAGES CONTIGUES : %s — a rapprocher des largeurs portees de l'archetype.",
		strings.Join(plages, ", "))
}
