package main

// livraison_audio.go — les operations WAV du mode `livrer` : indexation des sources
// extraites, troncature passthrough (coups_lot.py:tronquer), lecture stricte 16 bits/48 kHz
// et melange par couches (coups_lot.py:lire/tirerCoup/ecrire).
//
// DEUX LECTEURS WAV DISTINCTS, VOLONTAIREMENT — ce fichier n'utilise PAS `lireWAV` /
// `ecrireWAV24` de wav_io.go (mode `rendu-event`, moteur 24 bits flottant avec pitch/gain
// de chemin HIRC complet, normalisation -1 dBFS) : ce moteur produirait un fichier DIFFERENT
// octet a octet. `livraisonTronquer` reproduit `wave.getparams()/setparams()` (passthrough
// sans decoder les echantillons, format d'origine preserve) ; `livraisonLire16` reproduit le
// filtre strict de `coups_lot.py:lire` (16 bits, 48 kHz, sinon rejet silencieux) ; l'un et
// l'autre sont le contrat EXACT que `_outils/livraison.py` attend.

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// livraisonTauxRequis : la frequence d'echantillonnage EXIGEE par coups_lot.py:lire (TAUX).
const livraisonTauxRequis = 48000

// livraisonDureeLivreeS : la troncature de livraison (coups_lot.py: DUREE_LIVREE_S), meme
// discipline de poids que le pack en place.
const livraisonDureeLivreeS = 1.2

// livraisonChunksWav lit les en-tetes RIFF/WAVE minimales et rend les octets bruts du chunk
// `data`, sans decoder un seul echantillon. Format PCM (1) exige : c'est le seul format que
// les deux sources de ce mode produisent (vgmstream, et coups_lot.py:ecrire lui-meme).
func livraisonChunksWav(chemin string) (canaux, taux, bits int, data []byte, err error) {
	b, err := os.ReadFile(chemin)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return 0, 0, 0, nil, fmt.Errorf("%s: pas un RIFF/WAVE", chemin)
	}
	var format int
	for off := 12; off+8 <= len(b); {
		id := string(b[off : off+4])
		taille := int(binary.LittleEndian.Uint32(b[off+4:]))
		corps := off + 8
		if taille < 0 || corps+taille > len(b) {
			taille = len(b) - corps
		}
		switch id {
		case "fmt ":
			if taille < 16 {
				return 0, 0, 0, nil, fmt.Errorf("%s: chunk fmt trop court", chemin)
			}
			format = int(binary.LittleEndian.Uint16(b[corps:]))
			canaux = int(binary.LittleEndian.Uint16(b[corps+2:]))
			taux = int(binary.LittleEndian.Uint32(b[corps+4:]))
			bits = int(binary.LittleEndian.Uint16(b[corps+14:]))
		case "data":
			data = b[corps : corps+taille]
		}
		off = corps + taille
		if taille%2 == 1 {
			off++
		}
	}
	if format != 1 {
		return 0, 0, 0, nil, fmt.Errorf("%s: format WAV %d non gere (PCM=1 attendu)", chemin, format)
	}
	if canaux <= 0 || bits <= 0 || data == nil {
		return 0, 0, 0, nil, fmt.Errorf("%s: en-tete WAV incomplet", chemin)
	}
	return canaux, taux, bits, data, nil
}

// livraisonEcrireWavBrut ecrit un WAV PCM canonique (44 octets d'en-tete) a partir d'octets
// deja au format voulu — aucun decodage, aucun melange.
func livraisonEcrireWavBrut(chemin string, canaux, taux, bits int, corps []byte) error {
	blocAlign := canaux * (bits / 8)
	entete := make([]byte, 44)
	copy(entete[0:], "RIFF")
	binary.LittleEndian.PutUint32(entete[4:], uint32(36+len(corps)))
	copy(entete[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(entete[16:], 16)
	binary.LittleEndian.PutUint16(entete[20:], 1) // PCM
	binary.LittleEndian.PutUint16(entete[22:], uint16(canaux))
	binary.LittleEndian.PutUint32(entete[24:], uint32(taux))
	binary.LittleEndian.PutUint32(entete[28:], uint32(taux*blocAlign))
	binary.LittleEndian.PutUint16(entete[32:], uint16(blocAlign))
	binary.LittleEndian.PutUint16(entete[34:], uint16(bits))
	copy(entete[36:], "data")
	binary.LittleEndian.PutUint32(entete[40:], uint32(len(corps)))
	return os.WriteFile(chemin, append(entete, corps...), 0o644)
}

// livraisonTronquer copie `src` vers `dst` en ne gardant que les `dureeS` premieres
// secondes, format d'origine preserve — port de coups_lot.py:tronquer
// (`wave.getparams()`/`setparams()`), qui ne decode AUCUN echantillon.
func livraisonTronquer(src, dst string, dureeS float64) error {
	canaux, taux, bits, data, err := livraisonChunksWav(src)
	if err != nil {
		return err
	}
	blocAlign := canaux * (bits / 8)
	if blocAlign <= 0 {
		return fmt.Errorf("livraison: %s: bloc audio invalide (canaux=%d bits=%d)", src, canaux, bits)
	}
	framesTotal := len(data) / blocAlign
	framesVoulues := int(float64(taux) * dureeS) // troncature vers zero, comme int() en Python
	if framesVoulues > framesTotal {
		framesVoulues = framesTotal
	}
	if framesVoulues < 0 {
		framesVoulues = 0
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return livraisonEcrireWavBrut(dst, canaux, taux, bits, data[:framesVoulues*blocAlign])
}

// livraisonIndexWemsRe : le patron de nom des `.wav` extraits (duree puis identifiant Wwise).
var livraisonIndexWemsRe = regexp.MustCompile(`^\d+\.\d+s_(\d+)\.wav$`)

// livraisonIndexWems rend, pour un dossier d'arme, l'identifiant Wwise -> chemin `.wav`, en
// cherchant aussi dans son dossier `_EMBARQUES` — port de coups_lot.py:indexWems. Premiere
// occurrence gagne (comme `dict.setdefault`) : le dossier NON embarque est cherche en premier.
func livraisonIndexWems(racine, dossier string) (map[uint32]string, error) {
	out := map[uint32]string{}
	for _, d := range [2]string{dossier, dossier + "_EMBARQUES"} {
		fichiers, err := filepath.Glob(filepath.Join(racine, d, "*.wav"))
		if err != nil {
			return nil, err
		}
		for _, p := range fichiers {
			m := livraisonIndexWemsRe.FindStringSubmatch(filepath.Base(p))
			if m == nil {
				continue
			}
			id64, err := strconv.ParseUint(m[1], 10, 32)
			if err != nil {
				continue
			}
			id := uint32(id64)
			if _, deja := out[id]; !deja {
				out[id] = p
			}
		}
	}
	return out, nil
}

// livraisonLire16 lit un WAV 16 bits / 48 kHz et rend l'entrelacement STEREO (mono duplique
// sur les deux canaux) — port de coups_lot.py:lire. Rejette (rend false) tout ce qui ne colle
// pas au format attendu, EXACTEMENT comme la version Python rend None : c'est un FILTRE, pas
// une erreur qui doit interrompre le rendu.
func livraisonLire16(chemin string) ([]int16, bool) {
	canaux, taux, bits, data, err := livraisonChunksWav(chemin)
	if err != nil || bits != 16 || taux != livraisonTauxRequis || len(data) == 0 {
		return nil, false
	}
	n := len(data) / 2
	e := make([]int16, n)
	for i := 0; i < n; i++ {
		e[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	switch canaux {
	case 1:
		st := make([]int16, len(e)*2)
		for i, v := range e {
			st[2*i], st[2*i+1] = v, v
		}
		return st, true
	case 2:
		return e, true
	default:
		return nil, false
	}
}

// livraisonPiste : une couche deja lue et son gain de chemin — port du triplet
// (pistes, gains, decalages) que coups_lot.py:tirerCoup rend en trois listes paralleles.
type livraisonPiste struct {
	Echantillons []int16 // entrelacement stereo, 16 bits
	GainDB       float64
	Decalage     int // en TRAMES ; toujours 0 pour livraison.py (pas de rafale ici)
}

// livraisonTirerCoup choisit UNE variante par couche (rng partage entre couches, comme
// Python) et lit son fichier — port de coups_lot.py:tirerCoup. Une couche sans candidat
// disponible, ou dont le fichier ne passe pas livraisonLire16, est simplement omise.
func livraisonTirerCoup(couches []livraisonLot1Couche, index map[uint32]string, rng *mt19937, decalage int) []livraisonPiste {
	var out []livraisonPiste
	for _, b := range couches {
		var dispo []uint32
		for _, w := range b.WemsCandidats {
			if _, ok := index[w]; ok {
				dispo = append(dispo, w)
			}
		}
		if len(dispo) == 0 {
			continue
		}
		w := dispo[rng.choice(len(dispo))]
		s, ok := livraisonLire16(index[w])
		if !ok {
			continue
		}
		out = append(out, livraisonPiste{
			Echantillons: s,
			GainDB:       b.GainsDB[strconv.FormatUint(uint64(w), 10)],
			Decalage:     decalage,
		})
	}
	return out
}

// livraisonRendreEvent rend l'evenement `idEvent` du dossier en melangeant une variante par
// couche (graine FIXE 20260816, un seul appel par execution du mode `livrer`) — port de
// livraison.py:rendreEvent.
func livraisonRendreEvent(dossier string, idEvent uint32, chemin, racine string, lot1 map[string]livraisonLot1Arme) error {
	a1, ok := lot1[dossier]
	if !ok {
		return fmt.Errorf("livraison: dossier %q absent de lot1.json", dossier)
	}
	var ev *livraisonLot1Event
	for i := range a1.Evenements {
		if a1.Evenements[i].IDEvent == idEvent {
			ev = &a1.Evenements[i]
			break
		}
	}
	if ev == nil {
		return fmt.Errorf("livraison: evenement %08x absent de %s dans lot1.json", idEvent, dossier)
	}
	index, err := livraisonIndexWems(racine, dossier)
	if err != nil {
		return err
	}
	pistes := livraisonTirerCoup(ev.Couches, index, newMT19937FromSeed(20260816), 0)
	if len(pistes) == 0 {
		return fmt.Errorf("livraison: rendu impossible %s %08x", dossier, idEvent)
	}
	return livraisonEcrireCoup(pistes, chemin)
}

// livraisonEcrireCoup somme les pistes (gain de chemin en dB) et normalise SEULEMENT si le
// pic depasse 30000 (jamais d'amplification) — port de coups_lot.py:ecrire.
func livraisonEcrireCoup(pistes []livraisonPiste, chemin string) error {
	if len(pistes) == 0 {
		return nil
	}
	longueurTrames := 0
	for _, p := range pistes {
		if fr := p.Decalage + len(p.Echantillons)/2; fr > longueurTrames {
			longueurTrames = fr
		}
	}
	longueurTrames += livraisonTauxRequis / 2 // demi-seconde de queue, comme TAUX // 2
	buf := make([]float64, longueurTrames*2)
	for _, p := range pistes {
		facteur := math.Pow(10, p.GainDB/20)
		b := p.Decalage * 2
		for k, v := range p.Echantillons {
			buf[b+k] += float64(v) * facteur
		}
	}
	crete := 0.0
	for _, v := range buf {
		if av := math.Abs(v); av > crete {
			crete = av
		}
	}
	if crete == 0 {
		crete = 1.0
	}
	att := 1.0
	if a := 30000.0 / crete; a < 1.0 {
		att = a
	}
	corps := make([]byte, len(buf)*2)
	for i, v := range buf {
		binary.LittleEndian.PutUint16(corps[i*2:], uint16(int16(v*att)))
	}
	return livraisonEcrireWavBrut(chemin, 2, livraisonTauxRequis, 16, corps)
}
