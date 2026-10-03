package main

// wemduree.go — duree APPROXIMATIVE d'un `.wem` lue dans son EN-TETE RIFF, SANS decodage
// audio (lot L6, S3 : le decodeur Vorbis/Opus Wwise n'est pas dans la chaine d'extraction —
// `.ai/V7.5/replay2d/PLAN_BALISE_MIX_WWISE.md` phase 5, `RECETTE_SONS_ARMES.md` §0).
//
// UN `.wem` EST UN CONTENEUR RIFF (magic `RIFF`/`WAVE`, sous-chunks `fmt `/`data`), meme
// quand le codec a l'interieur est le Vorbis maison de Wwise — c'est ce que les outils de la
// communaute lisent pour naviguer le fichier avant de decoder. Le sous-chunk `fmt ` porte
// `nAvgBytesPerSec`, renseigne par l'encodeur pour permettre l'estimation SANS decoder :
//
//	duree_s = taille(sous-chunk `data`) / nAvgBytesPerSec
//
// C'EST UNE ESTIMATION, PAS UNE MESURE : le debit Vorbis n'est pas rigoureusement constant.
// Suffisant pour un TRI par tranche de duree (0,3-2 s), pas pour un montage.
//
// VALIDE SUR DONNEES CONNUES avant tout usage en triage (mode `eqip-durees`, premiere
// execution) : les deux evenements de `b29ac6de` (0b2a938e, fb25cbdd) ont ete mesures par
// decodage REEL (vgmstream) dans `PLAN_BALISE_MIX_WWISE.md` phase 3.3 a 0,41-0,48 s — la
// comparaison contre cette fonction est le temoin, publiee au journal, pas postulee ici.

import "encoding/binary"

// chunksRIFF decoupe une suite de sous-chunks RIFF standard (magic 4o + taille u32 LE +
// charge utile, ALIGNEE SUR 2 OCTETS). Distincte de `chunks()` (bank.go) qui ne pave jamais :
// c'est le format `sbnk` maison, deja prouve sur toute la chaine d'extraction — le paver
// aurait pu decaler la lecture de toutes les banks. Le conteneur RIFF standard, lui, pave
// bien ses sous-chunks impairs (specification RIFF) ; une fonction separee evite de faire
// porter ce risque au format `sbnk` qui n'en a pas besoin.
func chunksRIFF(b []byte) map[string][]byte {
	out := map[string][]byte{}
	for off := 0; off+8 <= len(b); {
		magic := string(b[off : off+4])
		taille := int(binary.LittleEndian.Uint32(b[off+4:]))
		debut := off + 8
		if taille < 0 || debut+taille > len(b) {
			break
		}
		out[magic] = b[debut : debut+taille]
		off = debut + taille
		if off%2 == 1 && off < len(b) {
			off++ // alignement RIFF : chaque sous-chunk pave sur 2 octets si sa taille est impaire
		}
	}
	return out
}

// dureeApproxRIFF lit `RIFF`/`WAVE`/`fmt `/`data` et rend une duree en secondes.
//
// Rend ok=false, JAMAIS un nombre invente, si : les 12 premiers octets ne sont pas un
// conteneur RIFF/WAVE valide, le sous-chunk `fmt ` est absent ou trop court pour porter
// `nAvgBytesPerSec` (offset 8, 4 octets), ce champ est nul, le sous-chunk `data` est absent
// ou vide, ou la duree resultante sort d'une plage plausible (< 0 ou > 600 s) — meme discipline
// que `plausibleProp` (proprietes.go) : un layout qui derive echoue au controle plutot que de
// rendre un chiffre fantaisiste.
func dureeApproxRIFF(donnees []byte) (float64, bool) {
	if len(donnees) < 12 || string(donnees[0:4]) != "RIFF" || string(donnees[8:12]) != "WAVE" {
		return 0, false
	}
	ch := chunksRIFF(donnees[12:])
	fmtChunk, ok := ch["fmt "]
	if !ok || len(fmtChunk) < 16 {
		return 0, false
	}
	avgBytesPerSec := binary.LittleEndian.Uint32(fmtChunk[8:12])
	if avgBytesPerSec == 0 {
		return 0, false
	}
	data, ok := ch["data"]
	if !ok || len(data) == 0 {
		return 0, false
	}
	secondes := float64(len(data)) / float64(avgBytesPerSec)
	if secondes <= 0 || secondes > 600 {
		return 0, false
	}
	return secondes, true
}

// dureesEmbarquees calcule la duree approximative de chaque `.wem` EMBARQUE d'une banque
// (chunk `DATA` decoupe par l'index `DIDX`, `mediasEmbarques` de bank.go). Les `.wem` HORS de
// cette banque (index large, vivent dans un `.pck` externe) ne sont pas couverts : les
// atteindre exigerait de rouvrir un module different par arme, hors de portee du triage
// structurel — ils restent marques "duree inconnue" par l'appelant, jamais devines.
func dureesEmbarquees(emb map[uint32][2]uint32, data []byte) map[uint32]float64 {
	out := map[uint32]float64{}
	for id, e := range emb {
		debut, taille := int(e[0]), int(e[1])
		if debut < 0 || taille <= 0 || debut+taille > len(data) {
			continue
		}
		if d, ok := dureeApproxRIFF(data[debut : debut+taille]); ok {
			out[id] = d
		}
	}
	return out
}
