package replay

// filmfacts_flux.go — LE FLUX D OCTETS DU CODEC DES FAITS DE FILM, ET RIEN D AUTRE.
//
// Extrait de `filmfacts_codec.go` le 2026-09-17 (lot 4.1.1-a) : celui-ci a franchi les 500
// lignes en gagnant la paire petit-boutiste ecrite a la main (cf. ci-dessous), et la table de
// `film_file_size_test.go` est DATEE ET FERMEE au 2026-09-16 — un fichier neuf ne s y inscrit
// pas, il se coupe.
//
// LA COUPE SUIT UNE FRONTIERE REELLE : ici le TRANSPORT (accumulateur, lecteur borne, varints,
// flottants, chaines, booleens, et le centimetre entier) ; dans `filmfacts_codec.go` les
// SOUS-CODECS PARTAGES entre sections (positions, pistes, objets du monde, images-cles,
// munitions). Le transport ne connait aucune forme du decodeur, et aucune forme ne connait le
// transport autrement que par `gwriter` / `greader`.

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ajouterPoidsFaibleDAbord / lirePoidsFaibleDAbord : les `n` octets de poids faible de `v`,
// petit-boutiste, ECRITS A LA MAIN ET PAS PAR `binary.LittleEndian`.
//
// # POURQUOI CETTE PAIRE EXISTE (lot 4.1.1-a, 2026-09-17)
//
// `TestAucuneLectureDOctetsBrutsHorsDeLaSource` (ADR 0034 D-2) interdit `binary.LittleEndian`
// dans les racines du decodeur, sur une regle de PERIMETRE qui est vraie et qu il faut garder :
// « dans ces racines, les seuls octets qu on lit sont ceux d un film ». Ce codec-ci ne lit AUCUN
// octet de film — il lit les octets de SON PROPRE conteneur, ecrits par [EncodeFilmFacts] — mais
// le ratchet ne peut pas faire la difference, et ELARGIR SA TOLERANCE aurait coute la regle pour
// tout le monde (son en-tete dit qu il n a plus d allowlist, et que rouvrir une tolerance est une
// DECISION ecrite, pas une ligne a remplir).
//
// LE FORMAT NE BOUGE PAS D UN OCTET : ces deux fonctions sont exactement
// `binary.LittleEndian.AppendUint32/64` et `Uint32/64`, d ou les huit fixtures inchangees
// (`git diff --stat -- testdata/` vide au commit de promotion).
func ajouterPoidsFaibleDAbord(b []byte, v uint64, n int) []byte {
	for i := 0; i < n; i++ {
		b = append(b, byte(v>>(8*uint(i))))
	}
	return b
}

// lirePoidsFaibleDAbord lit `n` octets petit-boutistes. L APPELANT A DEJA BORNE `b` : les deux
// sites verifient `r.off+n <= len(r.b)` et posent `r.err` sinon — la garde est la, pas ici.
func lirePoidsFaibleDAbord(b []byte, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v |= uint64(b[i]) << (8 * uint(i))
	}
	return v
}

// cmScale convertit une coordonnee en CENTIMETRES ENTIERS.
//
// CE N EST PAS UNE PERTE, ET C EST MESURABLE : toute coordonnee publiee par l assemblage passe
// par `round2` (arrondi au centieme), sans exception — traces, tirs, lancers, projectiles. Le
// centimetre entier est donc exactement la precision que la sortie porte. Coder un float32 brut
// couterait 12 octets par position pour une decimale que personne ne lit.
const cmScale = 100

// gwriter accumule un flux binaire. Les entiers sont en varint : les deltas d horodatage et de
// position tiennent sur un a deux octets, ce qui fait tout le poids du fixture.
type gwriter struct {
	b []byte
	// echec porte la PREMIERE erreur d encodage rencontree. Le codec est sans erreur par
	// construction sur tout ce qu il ecrit a la main ; seules les charges JSON (les morts
	// d objet) peuvent echouer, et un echec avale produirait un fichier de faits qui se relit
	// comme un fait FAUX. `EncodeFilmFactsFile` le remonte a l appelant.
	echec error
}

func (w *gwriter) u(v uint64)   { w.b = binary.AppendUvarint(w.b, v) }
func (w *gwriter) i(v int64)    { w.b = binary.AppendVarint(w.b, v) }
func (w *gwriter) byte8(v byte) { w.b = append(w.b, v) }
func (w *gwriter) f32(v float32) {
	w.b = ajouterPoidsFaibleDAbord(w.b, uint64(math.Float32bits(v)), 4)
}
func (w *gwriter) str(s string) {
	w.u(uint64(len(s)))
	w.b = append(w.b, s...)
}
func (w *gwriter) bool8(v bool) {
	if v {
		w.byte8(1)
		return
	}
	w.byte8(0)
}

// greader relit le flux. Toute incoherence est une ERREUR remontee, jamais une valeur nulle
// servie en silence.
type greader struct {
	b   []byte
	off int
	err error
}

func (r *greader) u() uint64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Uvarint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("uvarint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) i() int64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Varint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("varint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) byte8() byte {
	if r.err != nil {
		return 0
	}
	if r.off >= len(r.b) {
		r.err = fmt.Errorf("fin de flux prematuree a l offset %d", r.off)
		return 0
	}
	v := r.b[r.off]
	r.off++
	return v
}

func (r *greader) f32() float32 {
	if r.err != nil {
		return 0
	}
	if r.off+4 > len(r.b) {
		r.err = fmt.Errorf("float32 tronque a l offset %d", r.off)
		return 0
	}
	v := math.Float32frombits(uint32(lirePoidsFaibleDAbord(r.b[r.off:], 4)))
	r.off += 4
	return v
}

func (r *greader) str() string {
	n := int(r.u())
	if r.err != nil {
		return ""
	}
	if r.off+n > len(r.b) {
		r.err = fmt.Errorf("chaine tronquee a l offset %d", r.off)
		return ""
	}
	s := string(r.b[r.off : r.off+n])
	r.off += n
	return s
}

func (r *greader) bool8() bool { return r.byte8() == 1 }

func cmOf(v float32) int64 { return int64(math.Round(float64(v) * cmScale)) }

func fromCM(v int64) float32 { return float32(float64(v) / cmScale) }

// tranche rend les `n` prochains octets SANS copie, et pose l erreur si le flux est plus court.
//
// C est la primitive du CADRE A LONGUEUR PREFIXEE du fichier de faits : une section se lit comme
// une tranche, ce qui permet d en SAUTER une inconnue sans savoir la decoder.
func (r *greader) tranche(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.off+n > len(r.b) {
		r.err = fmt.Errorf("tranche de %d octet(s) a l offset %d : %d disponible(s)",
			n, r.off, len(r.b)-r.off)
		return nil
	}
	out := r.b[r.off : r.off+n]
	r.off += n
	return out
}

// compte lit un NOMBRE D ELEMENTS et REFUSE celui qui ne peut pas tenir dans ce qui reste.
//
// # POURQUOI CE GARDE-FOU EXISTE (2026-09-18, lot 4.1.3)
//
// Un `make([]T, 0, n)` sur un `n` lu dans un flux DESYNCHRONISE alloue des gigaoctets et fait
// PANIQUER le processus. Mesure du jour : un fichier de faits d une version anterieure, relu par
// le decodeur courant, a fait paniquer la cuisson dans `decodePositionSection` — au lieu de rendre
// une erreur que l appelant traite en redecodant le film (`lireLesFaitsFrais`, chemin « illisible
// malgre un en-tete frais »).
//
// UN FICHIER DE FAITS VIENT DU DISQUE, donc il peut etre perime, tronque ou corrompu : le
// decodeur doit rendre une ERREUR, jamais tomber. `coutMinimal` est le nombre d octets qu un
// element consomme AU MINIMUM (un varint vaut 1) : au-dela de ce que le flux porte encore, le
// compte est faux par construction.
func (r *greader) compte(coutMinimal int) int {
	n := int(r.u())
	if r.err != nil {
		return 0
	}
	if coutMinimal < 1 {
		coutMinimal = 1
	}
	if reste := len(r.b) - r.off; n < 0 || n > reste/coutMinimal {
		r.err = fmt.Errorf("compte de %d element(s) a l offset %d : %d octet(s) restants, "+
			"%d au minimum par element — flux desynchronise", n, r.off, reste, coutMinimal)
		return 0
	}
	return n
}
