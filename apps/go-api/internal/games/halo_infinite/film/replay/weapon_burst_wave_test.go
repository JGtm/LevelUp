package replay

// weapon_burst_wave_test.go — LA PRIMITIVE DE SIGNAL de l'instrument de rafale : lire un `.wav`,
// en tirer une enveloppe, y compter les transitoires d'attaque.
//
// EXTRAIT DE `weapon_burst_research_test.go` pour la meme raison que
// `pads_proximity_geometry_test.go` l'a ete de son instrument : le fichier depassait le seuil de
// 500 lignes du depot (CLAUDE.md n°5). La coupure est franche — ce fichier ne connait NI artefact,
// NI arme, NI seuil de decision : il ne sait que redresser un signal et compter des departs. Les
// DEUX INSTRUMENTS (cadence des fire events, contenu des assets) vivent ensemble a cote.
//
// AUCUN CODE DE PRODUCTION : lecture seule, aucun asset n'est ecrit ni regenere.

import (
	"encoding/binary"
	"fmt"
	"math"
)

// LE LISSAGE de l'enveloppe, en millisecondes. Une forme d'onde redressee est un herisson : sans
// lissage, chaque alternance compte pour un pic.
//
// CALIBRE, PAS CHOISI : la premiere version lissait sur 5 ms (court devant l'attaque d'un coup de
// feu, ~20 ms) et l'ancre du fusil de precision rendait 8 puis 5 transitoires au lieu de 1. La
// cause est une echelle de temps oubliee — la QUEUE d'un coup de feu est riche en graves, et une
// composante a 50 Hz module l'enveloppe avec une periode de 20 ms qu'une fenetre de 5 ms laisse
// intacte. 20 ms efface cette ondulation tout en restant sous la separation de deux coups (40 ms),
// donc sans jamais fondre deux departs en un.
const wbrSmoothMS = 20.0

// LE SEUIL D'ATTAQUE, en part du maximum de l'enveloppe.
const wbrOnsetRatio = 0.35

// LA SEPARATION MINIMALE entre deux transitoires, en millisecondes. Sous 40 ms, deux depassements
// sont la MEME attaque (rebond de l'enveloppe), pas deux coups : aucune arme du jeu ne tire a plus
// de 25 coups par seconde.
const wbrOnsetGapMS = 40.0

// LE FACTEUR D'ATTAQUE — ce qui fait d'un pic un COUP et non un rebond de queue.
//
// CALIBRAGE ANCRE SUR LE FUSIL DE PRECISION (`hinf_s7_sniper`) : son asset est un COUP UNIQUE par
// construction (arme a verrou), il doit donc rendre exactement 1 transitoire, et l'instrument
// l'ASSERTE. Une premiere version comptait tout FRANCHISSEMENT du seuil, avec retour au repos a
// 50 % de celui-ci pour hysteresis : elle rendait 8 transitoires sur ce meme fichier. La cause
// mesuree est la QUEUE DE REVERBERATION — elle oscille durablement entre 17 % et 35 % du maximum,
// et chaque remontee comptait pour un coup. Le niveau seul ne distingue donc pas une attaque
// d'une resonance.
//
// LE CRITERE RETENU EST UNE MONTEE, pas un niveau : un pic ne compte que s'il domine son
// voisinage (maximum local sur +/- la separation) ET s'il vaut au moins ce facteur fois le creux
// qui le precede immediatement. Une attaque de coup de feu monte de plus de 20 dB ; une queue qui
// decroit ne remonte jamais d'un facteur 4 (12 dB). C'est ce qui rend l'ancre a 1.
const wbrAttackFactor = 4.0

// wbrWave est le contenu decode d'un `.wav` PCM 16 bits, ramene a un canal.
type wbrWave struct {
	rate     int
	channels int
	samples  []float64
}

func (w wbrWave) durationS() float64 {
	if w.rate <= 0 {
		return 0
	}
	return float64(len(w.samples)) / float64(w.rate)
}

// wbrDecodeWave lit un RIFF/WAVE PCM 16 bits et rend le mixage mono de ses canaux.
//
// LE PARCOURS EST PAR CHUNKS, pas a offset fixe : un `.wav` peut porter des chunks intermediaires
// (`LIST`, `fact`) selon l'outil qui l'a ecrit, et lire `data` a l'octet 44 rendrait du bruit sur
// ceux-la. Un format autre que PCM 16 bits est une ERREUR DITE, jamais un decodage approximatif —
// un tableau de transitoires calcule sur des octets mal interpretes serait credible et faux.
func wbrDecodeWave(raw []byte) (wbrWave, error) {
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return wbrWave{}, fmt.Errorf("en-tete RIFF/WAVE absent")
	}
	var w wbrWave
	var bits int
	var data []byte
	for off := 12; off+8 <= len(raw); {
		id := string(raw[off : off+4])
		size := int(binary.LittleEndian.Uint32(raw[off+4 : off+8]))
		body := off + 8
		if size < 0 || body+size > len(raw) {
			size = len(raw) - body
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return wbrWave{}, fmt.Errorf("chunk fmt tronque (%d octets)", size)
			}
			if f := binary.LittleEndian.Uint16(raw[body:]); f != 1 {
				return wbrWave{}, fmt.Errorf("format audio %d (PCM=1 attendu)", f)
			}
			w.channels = int(binary.LittleEndian.Uint16(raw[body+2:]))
			w.rate = int(binary.LittleEndian.Uint32(raw[body+4:]))
			bits = int(binary.LittleEndian.Uint16(raw[body+14:]))
		case "data":
			data = raw[body : body+size]
		}
		// Les chunks RIFF sont alignes sur deux octets : une taille impaire porte un octet de
		// bourrage que le parcours doit enjamber, sinon tout ce qui suit est decale.
		off = body + size + size%2
	}
	if bits != 16 {
		return wbrWave{}, fmt.Errorf("%d bits par echantillon (16 attendus)", bits)
	}
	if w.channels <= 0 || w.rate <= 0 || len(data) == 0 {
		return wbrWave{}, fmt.Errorf("chunk fmt ou data absent")
	}
	w.samples = wbrMixMono(data, w.channels)
	return w, nil
}

// wbrMixMono ramene des echantillons entrelaces 16 bits signes a un canal, normalises -1..1.
func wbrMixMono(data []byte, channels int) []float64 {
	n := len(data) / (2 * channels)
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		somme := 0.0
		for c := 0; c < channels; c++ {
			v := int16(binary.LittleEndian.Uint16(data[2*(i*channels+c):])) //nolint:gosec // PCM 16 bits signe
			somme += float64(v) / 32768
		}
		out[i] = somme / float64(channels)
	}
	return out
}

// wbrEnvelope redresse puis lisse : moyenne glissante de la valeur absolue sur wbrSmoothMS.
func wbrEnvelope(w wbrWave) []float64 {
	win := int(wbrSmoothMS * float64(w.rate) / 1000)
	if win < 1 {
		win = 1
	}
	out := make([]float64, len(w.samples))
	somme := 0.0
	for i, v := range w.samples {
		somme += math.Abs(v)
		if i >= win {
			somme -= math.Abs(w.samples[i-win])
		}
		large := win
		if i+1 < win {
			large = i + 1
		}
		out[i] = somme / float64(large)
	}
	return out
}

// wbrOnsets compte les TRANSITOIRES D'ATTAQUE d'un fichier.
//
// LA REGLE, en trois conditions et dans cet ordre :
//  1. le pic depasse le seuil et DOMINE son voisinage (maximum local sur +/- la separation) —
//     sans quoi les quelques echantillons qui suivent une attaque comptent chacun pour un depart ;
//  2. le PREMIER pic retenu compte toujours : le debut d'un fichier de son EST une attaque, et
//     rien ne le precede a quoi le comparer. Une version anterieure l'exigeait plus fort qu'un
//     creux inexistant et rendait ZERO transitoire sur trois assets (epee, pistolet a plasma,
//     carabine a impulsion) — un fichier qui sonne et qui compte zero coup est une anomalie
//     d'instrument, pas une propriete de l'asset ;
//  3. les SUIVANTS doivent monter d'au moins wbrAttackFactor depuis le CREUX QUI LES SEPARE du
//     transitoire precedent. Entre deux coups, l'enveloppe redescend ; dans la queue d'un seul
//     coup, elle ondule sans jamais retomber d'un facteur 4. C'est cette condition qui distingue
//     une salve d'une resonance — et, sur un son TENU (rayon de Sentinelle), elle rend 1 : un
//     faisceau n'a qu'un depart, ce qui est physiquement juste.
func wbrOnsets(env []float64, rate int, ratio float64) int {
	maxi := 0.0
	for _, v := range env {
		if v > maxi {
			maxi = v
		}
	}
	if maxi <= 0 {
		return 0
	}
	haut := ratio * maxi
	sep := int(wbrOnsetGapMS * float64(rate) / 1000)
	if sep < 1 {
		sep = 1
	}
	n, dernier := 0, -1
	for i, v := range env {
		if v < haut || !wbrIsLocalMax(env, i, sep) {
			continue
		}
		if dernier < 0 {
			n, dernier = 1, i
			continue
		}
		if i-dernier < sep || v < wbrAttackFactor*wbrMinBetween(env, dernier, i) {
			continue
		}
		n, dernier = n+1, i
	}
	return n
}

// wbrIsLocalMax dit si `i` domine sa fenetre.
func wbrIsLocalMax(env []float64, i, sep int) bool {
	lo, hi := i-sep, i+sep
	if lo < 0 {
		lo = 0
	}
	if hi > len(env)-1 {
		hi = len(env) - 1
	}
	for j := lo; j <= hi; j++ {
		if env[j] > env[i] {
			return false
		}
	}
	return true
}

// wbrMinBetween rend le creux entre deux instants — la vallee qui separe deux pics candidats.
func wbrMinBetween(env []float64, from, to int) float64 {
	mini := math.Inf(1)
	for j := from; j <= to && j < len(env); j++ {
		if env[j] < mini {
			mini = env[j]
		}
	}
	if math.IsInf(mini, 1) {
		return 0
	}
	return mini
}
