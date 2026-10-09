package main

// rendu_event.go — mode `rendu-event` : rendre UN EVENEMENT ENTIER, tel que le jeu le joue.
//
// CE QUE CE MODE CORRIGE (lot V3E, 2026-09-02). Reproche utilisateur, verbatim : « tu n'as
// rendu que les fichiers isoles, pas reconstitues avec leur format wwise et reglages de
// package ». Un fichier livre doit etre un EVENEMENT COMPLET :
//
//	toutes les couches simultanees sommees, chacune a son gain de chemin HIRC COMPLET
//	   (parents actor-mixer + bus + makeup gain + gain de noeud), pas un gain relatif ;
//	les offsets de couche appliques (`AkPropID_InitialDelay`, 59) ;
//	la hauteur declaree appliquee (`AkPropID_Pitch`, 2) ;
//	un TIRAGE de variante par couche, independant d'une couche a l'autre ;
//	une SEULE normalisation finale a -1 dBFS, aucune egalisation, aucune compression.
//
// Les couches isolees restent produites, mais dans un sous-dossier `stems/` : ce ne sont
// pas des livrables d'ecoute, ce sont des pieces de verification.
//
// Usage :
//
//	ws -etroit -mode rendu-event -json plan.json -events <gid hexa> \
//	   -wav <dossier de .wav decodes> -dest <dossier de sortie> -nom explosion \
//	   -tirages 3 [-graine 1] [-duree 8] [-out mesures.json]
//
// `-duree` non nul : chaque couche est BOUCLEE (repetee bout a bout, comme un corps de
// boucle Wwise) jusqu'a la duree demandee avant la somme — c'est le rendu des moteurs.
// Sans `-duree`, l'evenement est joue une fois (one-shot : explosions).

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
)

// mesureRendu : ce qu'un fichier rendu vaut, pour le rapport et le manifeste.
type mesureRendu struct {
	Fichier   string  `json:"fichier"`
	DureeS    float64 `json:"duree_s"`
	CreteDB   float64 `json:"crete_dbfs"`
	RMSDB     float64 `json:"rms_dbfs"`
	Canaux    int     `json:"canaux"`
	Variantes []int   `json:"variantes,omitempty"`
}

// rapportRendu : la sortie JSON du mode.
type rapportRendu struct {
	Event      string        `json:"event"`
	Bank       string        `json:"bank"`
	Nom        string        `json:"nom"`
	GainNormDB float64       `json:"gain_normalisation_db"`
	Couches    []coucheRendu `json:"couches"`
	Fichiers   []mesureRendu `json:"fichiers"`
}

// coucheRendu : ce que le rendu a applique a une couche, en clair.
type coucheRendu struct {
	Cible        string  `json:"cible"`
	GainDB       float64 `json:"gain_db"`
	DelaiS       float64 `json:"delai_s"`
	PitchCts     float64 `json:"pitch_cents"`
	NbVariantes  int     `json:"nb_variantes"`
	Bus          string  `json:"bus,omitempty"`
	RangedVolume *string `json:"ranged_volume_db,omitempty"`
	RangedPitch  *string `json:"ranged_pitch_cents,omitempty"`
	// CentreVolDB / CentrePitchCts : la valeur CENTRALE des fourchettes RANGED, celle que
	// le rendu applique reellement (nulle pour une fourchette symetrique).
	CentreVolDB    float64  `json:"centre_ranged_volume_db,omitempty"`
	CentrePitchCts float64  `json:"centre_ranged_pitch_cents,omitempty"`
	Wems           []uint32 `json:"wems"`
}

// optionsRendu regroupe les reglages du mode (la fonction depasserait 5 parametres).
type optionsRendu struct {
	Plan, Dossier, Dest, Nom, Sortie string
	Events                           map[uint32]bool
	Tirages                          int
	Graine                           int64
	DureeS                           float64
	// Etat : etat de regime a rendre pour un evenement a `Switch` (0 = la variante du plan
	// sans etat force). Le plan publie un evenement par etat ; ce champ dit lequel jouer.
	Etat uint32
}

// rendreEvenements est le mode `rendu-event`.
func rendreEvenements(o optionsRendu) error {
	if o.Plan == "" || o.Dossier == "" || o.Dest == "" {
		return fmt.Errorf("le mode rendu-event exige -json (plan), -wav (dossier des .wav) et -dest")
	}
	blob, err := os.ReadFile(o.Plan)
	if err != nil {
		return err
	}
	var plan v3eRapport
	if err := json.Unmarshal(blob, &plan); err != nil {
		return err
	}
	if o.Tirages <= 0 {
		o.Tirages = 3
	}
	if manque := listerWavManquants(plan, o.Dossier, o.Events, o.Etat); len(manque) > 0 {
		return fmt.Errorf("%d media(s) du plan sans .wav decode dans %s : %v", len(manque), o.Dossier, manque)
	}
	var rapports []rapportRendu
	for _, ev := range plan.Events {
		var gid uint32
		if _, e := fmt.Sscanf(ev.Event, "%x", &gid); e != nil {
			continue
		}
		if len(o.Events) > 0 && !o.Events[gid] {
			continue
		}
		if ev.Etat != o.Etat {
			continue
		}
		r, err := rendreUnEvenement(ev, o)
		if err != nil {
			return err
		}
		rapports = append(rapports, r)
	}
	if len(rapports) == 0 {
		return fmt.Errorf("aucun evenement du plan ne correspond a -events")
	}
	if o.Sortie == "" {
		return nil
	}
	j, err := json.MarshalIndent(rapports, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(o.Sortie, j, 0o644)
}

// rendreUnEvenement rend un evenement : N tirages complets, plus les stems du tirage 1.
func rendreUnEvenement(ev v3eEvent, o optionsRendu) (rapportRendu, error) {
	r := rapportRendu{Event: ev.Event, Bank: ev.Bank, Nom: o.Nom}
	for _, c := range ev.Couches {
		cr := coucheRendu{Cible: c.Cible, NbVariantes: len(c.Variantes), Bus: c.BusEffectif}
		if len(c.Variantes) > 0 {
			cr.GainDB, cr.DelaiS, cr.PitchCts = c.Variantes[0].GainDB, c.Variantes[0].DelaiS, c.Variantes[0].PitchCts
		}
		for _, v := range c.Variantes {
			cr.Wems = append(cr.Wems, v.Wem)
		}
		if c.RangedVolume != nil {
			s := fmt.Sprintf("%+.2f .. %+.2f", c.RangedVolume[0], c.RangedVolume[1])
			cr.RangedVolume = &s
		}
		if c.RangedPitch != nil {
			s := fmt.Sprintf("%+.0f .. %+.0f", c.RangedPitch[0], c.RangedPitch[1])
			cr.RangedPitch = &s
		}
		cr.CentreVolDB, cr.CentrePitchCts = centreRanged(c)
		r.Couches = append(r.Couches, cr)
	}
	if err := os.MkdirAll(o.Dest, 0o755); err != nil {
		return r, err
	}
	fmt.Printf("\n== rendu %s (sbnk %s / event %s), %d couche(s), %d tirage(s)\n",
		o.Nom, ev.Bank, ev.Event, len(ev.Couches), o.Tirages)

	// Les tirages sont TOUS melanges d'abord, puis une SEULE et MEME normalisation les
	// ramene ensemble : le tirage le plus fort touche -1 dBFS, les autres gardent leur
	// ecart reel. Normaliser chaque tirage separement effacerait cet ecart ; le calibrer
	// sur le premier seul en laissait passer au-dessus de 0 dBFS (mesure : +0,13 dBFS).
	melanges := make([]*audio, o.Tirages)
	choix := make([][]int, o.Tirages)
	var pistes1 []*audio
	pire := math.Inf(-1)
	for t := 0; t < o.Tirages; t++ {
		choix[t] = tirage(ev, o.Graine, t)
		mel, pistes, err := melangerCouches(ev, choix[t], o)
		if err != nil {
			return r, err
		}
		if t == 0 {
			pistes1 = pistes
		}
		if c := crete(mel); c > pire {
			pire = c
		}
		melanges[t] = mel
	}
	gainNorm := -1.0 - pire
	r.GainNormDB = arrondi3(gainNorm)
	if err := ecrireStems(pistes1, ev, gainNorm, o); err != nil {
		return r, err
	}
	for t, mel := range melanges {
		appliquerGain(mel, gainNorm)
		nom := o.Nom + ".wav"
		if t > 0 {
			nom = fmt.Sprintf("%s_v%d.wav", o.Nom, t+1)
		}
		if o.DureeS > 0 {
			fondreBords(mel, 10)
		}
		if err := ecrireWAV24(filepath.Join(o.Dest, nom), mel); err != nil {
			return r, err
		}
		m := mesurer(nom, mel, choix[t])
		r.Fichiers = append(r.Fichiers, m)
		fmt.Printf("   %-34s %5.2f s  crete %+6.2f dBFS  RMS %7.2f dBFS  variantes %v\n",
			nom, m.DureeS, m.CreteDB, m.RMSDB, choix[t])
	}
	return r, nil
}

// tirage choisit une variante par couche. Le tirage 0 prend TOUJOURS la variante d'indice 0
// — c'est celle qu'avait rendue rev10, ce qui rend la comparaison avant/apres directe. Les
// tirages suivants sont pseudo-aleatoires et INDEPENDANTS d'une couche a l'autre, comme le
// moteur, qui tire une variante par conteneur `RandomSequence`.
func tirage(ev v3eEvent, graine int64, t int) []int {
	out := make([]int, len(ev.Couches))
	if t == 0 {
		return out
	}
	for i, c := range ev.Couches {
		if len(c.Variantes) <= 1 {
			continue
		}
		src := rand.New(rand.NewSource(graine*1_000_003 + int64(t)*7919 + int64(i)))
		out[i] = src.Intn(len(c.Variantes))
	}
	return out
}

// melangerCouches somme les couches d'un tirage. Rend le melange et les pistes isolees.
func melangerCouches(ev v3eEvent, choix []int, o optionsRendu) (*audio, []*audio, error) {
	pistes := make([]*audio, len(ev.Couches))
	taux, longueur := 48000, 0
	for i, c := range ev.Couches {
		if len(c.Variantes) == 0 {
			continue
		}
		v := c.Variantes[choix[i]]
		a, err := chargerVariante(o.Dossier, v, c)
		if err != nil {
			return nil, nil, err
		}
		taux = a.Taux
		if o.DureeS > 0 {
			a, err = tenirLaDuree(a, c, choix[i], o)
			if err != nil {
				return nil, nil, err
			}
		}
		if v.DelaiS > 0 {
			a = decaler(a, int(v.DelaiS*float64(a.Taux)))
		}
		pistes[i] = a
		if n := a.nEch(); n > longueur {
			longueur = n
		}
	}
	mel := &audio{Taux: taux, Canaux: 2, Ech: [][]float64{make([]float64, longueur), make([]float64, longueur)}}
	for _, p := range pistes {
		if p == nil {
			continue
		}
		for c := 0; c < 2; c++ {
			for i, v := range p.Ech[c] {
				mel.Ech[c][i] += v
			}
		}
	}
	return mel, pistes, nil
}

// chargerVariante lit un `.wem` decode et lui applique CE QUE LE PAQUET DECLARE : la
// hauteur, puis le gain de chemin complet. La montee en stereo se fait entre les deux —
// avant le gain, pour que la mesure du stem soit celle du melange.
func chargerVariante(dossier string, v v3eVariante, c v3eCouche) (*audio, error) {
	a, err := lireWAV(filepath.Join(dossier, fmt.Sprintf("%d.wav", v.Wem)))
	if err != nil {
		return nil, err
	}
	dVol, dPitch := centreRanged(c)
	if p := v.PitchCts + dPitch; p != 0 {
		a = reechantillonner(a, math.Pow(2, p/1200))
	}
	a = versStereo(a)
	appliquerGain(a, v.GainDB+dVol)
	return a, nil
}

// centreRanged rend la VALEUR CENTRALE des fourchettes RANGED de la couche.
//
// Le paquet RANGED donne deux composantes par propriete. Qu'il s'agisse d'OFFSETS SIGNES
// autour du nominal est MESURE, pas postule : les couches de contact du Ghost declarent
// (-80, +80) et (-85, +80) en hauteur — une paire symetrique ne peut etre qu'un couple
// min/max signe. La valeur centrale est donc (min + max) / 2, nulle pour une fourchette
// symetrique (la majorite) et non nulle pour les rares fourchettes decentrees, comme le
// (-3, 0) dB du contact Ghost ou le (0, +800) cents du lit sub.
func centreRanged(c v3eCouche) (volDB, pitchCts float64) {
	if c.RangedVolume != nil {
		volDB = (float64(c.RangedVolume[0]) + float64(c.RangedVolume[1])) / 2
	}
	if c.RangedPitch != nil {
		pitchCts = (float64(c.RangedPitch[0]) + float64(c.RangedPitch[1])) / 2
	}
	return volDB, pitchCts
}

// tenirLaDuree remplit la duree demandee POUR UNE COUCHE EN BOUCLE.
//
// LE PIEGE QUE CETTE FONCTION FERME. Un conteneur `RandomSequence` declare en boucle
// infinie ne rejoue pas le meme media : il RETIRE une variante a chaque iteration, et le
// mode de transition dit COMMENT les lectures se suivent. Les moteurs covenant/bannis sont
// en `AkTransitionMode` 5 (CADENCE) : une lecture demarre toutes les 0,125 a 0,25 s et elles
// SE CHEVAUCHENT. Le Ghost rend le piege visible — ses grains font 0,12 a 0,68 s et sont
// redeclenches toutes les 0,13 s, soit quatre a cinq grains superposes en permanence. Les
// mettre bout a bout (et a plus forte raison en repeter un seul) ne donne pas un souffle
// mais un hachage periodique.
//
// Une couche sans boucle declaree, ou a une seule variante, retombe sur la repetition
// simple — c'est le cas des corps de boucle longs (Scorpion, Wraith, Warthog : 4 a 8 s).
func tenirLaDuree(premier *audio, c v3eCouche, indice int, o optionsRendu) (*audio, error) {
	cible := int(o.DureeS * float64(premier.Taux))
	continu := c.Continu && c.Repetitions != nil && *c.Repetitions == 0
	if !continu || len(c.Variantes) <= 1 {
		return boucler(premier, cible), nil
	}
	pas := int(float64(c.TransitionS) * float64(premier.Taux))
	if c.ModeTransition != transitionCadence || pas <= 0 {
		pas = premier.nEch() // bout a bout : la lecture suivante commence a la fin
	}
	out := &audio{Taux: premier.Taux, Canaux: 2, Ech: [][]float64{
		make([]float64, cible), make([]float64, cible)}}
	src := rand.New(rand.NewSource(o.Graine*7919 + int64(indice)*104729 + 17))
	for d, n := 0, 0; d < cible; d, n = d+pas, n+1 {
		piece := premier
		if n > 0 {
			var err error
			if piece, err = chargerVariante(o.Dossier, c.Variantes[src.Intn(len(c.Variantes))], c); err != nil {
				return nil, err
			}
		}
		poser(out, piece, d)
	}
	return out, nil
}

// poser additionne un signal dans un melange a partir de l'echantillon `d`, sans deborder.
func poser(dest, src *audio, d int) {
	n := dest.nEch()
	for c := 0; c < dest.Canaux; c++ {
		canal := src.Ech[0]
		if c < len(src.Ech) {
			canal = src.Ech[c]
		}
		for i, v := range canal {
			if d+i >= n {
				break
			}
			dest.Ech[c][d+i] += v
		}
	}
}

// couper tronque un signal a `n` echantillons.
func couper(a *audio, n int) *audio {
	if a.nEch() <= n {
		return a
	}
	out := &audio{Taux: a.Taux, Canaux: a.Canaux, Ech: make([][]float64, a.Canaux)}
	for c := range a.Ech {
		out.Ech[c] = a.Ech[c][:n]
	}
	return out
}

// ecrireStems ecrit les couches isolees dans `stems/`, au MEME gain de normalisation que le
// melange : elles se superposent donc exactement au fichier principal.
func ecrireStems(pistes []*audio, ev v3eEvent, gainNorm float64, o optionsRendu) error {
	dossier := filepath.Join(o.Dest, "stems")
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		return err
	}
	for i, p := range pistes {
		if p == nil {
			continue
		}
		cp := &audio{Taux: p.Taux, Canaux: p.Canaux, Ech: make([][]float64, p.Canaux)}
		for c := range p.Ech {
			cp.Ech[c] = append([]float64(nil), p.Ech[c]...)
		}
		appliquerGain(cp, gainNorm)
		if o.DureeS > 0 {
			fondreBords(cp, 10)
		}
		nom := fmt.Sprintf("%s_couche%d_%s.wav", o.Nom, i+1, ev.Couches[i].Cible)
		if err := ecrireWAV24(filepath.Join(dossier, nom), cp); err != nil {
			return err
		}
	}
	return nil
}

// boucler repete un signal bout a bout jusqu'a `n` echantillons. C'est ce que fait un
// conteneur declare en boucle : les `.wem` du jeu SONT des corps de boucle.
func boucler(a *audio, n int) *audio {
	src := a.nEch()
	if src == 0 || n <= src {
		return a
	}
	out := &audio{Taux: a.Taux, Canaux: a.Canaux, Ech: make([][]float64, a.Canaux)}
	for c := range a.Ech {
		out.Ech[c] = make([]float64, n)
		for i := 0; i < n; i++ {
			out.Ech[c][i] = a.Ech[c][i%src]
		}
	}
	return out
}

// decaler insere `n` echantillons de silence en tete — l'offset de couche du paquet Wwise.
func decaler(a *audio, n int) *audio {
	if n <= 0 {
		return a
	}
	out := &audio{Taux: a.Taux, Canaux: a.Canaux, Ech: make([][]float64, a.Canaux)}
	for c := range a.Ech {
		out.Ech[c] = append(make([]float64, n), a.Ech[c]...)
	}
	return out
}

// mesurer rend les grandeurs d'un fichier rendu.
func mesurer(nom string, a *audio, choix []int) mesureRendu {
	return mesureRendu{
		Fichier: nom, DureeS: arrondi3(float64(a.nEch()) / float64(a.Taux)),
		CreteDB: arrondi3(crete(a)), RMSDB: arrondi3(rms(a)),
		Canaux: a.Canaux, Variantes: append([]int(nil), choix...),
	}
}

// listerWavManquants dit quels `.wem` du plan n'ont pas de `.wav` decode : un rendu qui
// tombe sur un fichier absent doit le DIRE avant de commencer, pas echouer au milieu.
func listerWavManquants(plan v3eRapport, dossier string, events map[uint32]bool, etat uint32) []uint32 {
	manque := map[uint32]bool{}
	for _, ev := range plan.Events {
		var gid uint32
		if _, e := fmt.Sscanf(ev.Event, "%x", &gid); e != nil {
			continue
		}
		if len(events) > 0 && !events[gid] {
			continue
		}
		if ev.Etat != etat {
			continue
		}
		for _, c := range ev.Couches {
			for _, v := range c.Variantes {
				if _, err := os.Stat(filepath.Join(dossier, fmt.Sprintf("%d.wav", v.Wem))); err != nil {
					manque[v.Wem] = true
				}
			}
		}
	}
	out := make([]uint32, 0, len(manque))
	for w := range manque {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
