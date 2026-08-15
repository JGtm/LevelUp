package main

// arbre.go — la STRUCTURE d'un evenement de tir, pas seulement ses `.wem`.
//
// LA QUESTION A TRANCHER. Les modes precedents rendent, pour un evenement, l'ENSEMBLE des
// `.wem` atteignables — un sac plat. Or Wwise distingue deux natures de conteneur :
//
//	Random (type 5, mode aleatoire) : joue UN enfant tire au hasard  -> variantes
//	Blend  (type 9) / Random en mode SEQUENCE : joue TOUS les enfants -> couches simultanees
//
// Si le tir est un empilement de couches, aucun `.wem` isole ne peut sonner juste, et
// choisir « le meilleur exemple » n'a alors pas de sens : il faut les additionner.
// Ce mode affiche l'arbre avec le TYPE de chaque noeud pour repondre sur pieces.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

// nomsTypesHIRC : nomenclature Wwise des objets de la hierarchie.
var nomsTypesHIRC = map[byte]string{
	1: "Settings", 2: "Sound", 3: "Action", 4: "Event", 5: "RandomSequence",
	6: "Switch", 7: "ActorMixer", 8: "Bus", 9: "Blend", 10: "MusicSegment",
	11: "MusicTrack", 12: "MusicSwitch", 13: "MusicRanSeq", 14: "Attenuation",
	15: "DialogueEvent", 16: "MotionBus", 17: "MotionFX", 18: "Effect",
	19: "AuxBus", 20: "LFO", 21: "Envelope", 22: "AudioDevice", 23: "TimeMod",
}

func nomType(t byte) string {
	if n, ok := nomsTypesHIRC[t]; ok {
		return n
	}
	return fmt.Sprintf("type%d", t)
}

// brancheRendue : une action de l'evenement, donc UNE COUCHE du son joue.
//
// Les actions d'un evenement se declenchent EN PARALLELE : chacune choisit sa variante de
// son cote, et le resultat est leur somme. C'est la distinction que les modes precedents
// perdaient en aplatissant tout dans un seul ensemble de `.wem`.
type brancheRendue struct {
	Cible     string   `json:"cible"`
	TypeNoeud string   `json:"type_noeud"`
	Wems      []uint32 `json:"wems_candidats"`
	// Gains : volume en dB declare par l'objet Sound de chaque `.wem`. Le moteur applique
	// ces gains ; les ignorer faisait arriver au meme niveau une couche de renfort censee
	// rester 10 ou 20 dB en arriere-plan.
	Gains map[string]float32 `json:"gains_db,omitempty"`
}

type eventCouches struct {
	IDEvent  string          `json:"id_event"`
	Branches []brancheRendue `json:"branches"`
	Total    int             `json:"wems_total"`
}

type rapportCouches struct {
	Arme   string         `json:"arme"`
	Events []eventCouches `json:"events"`
}

// couchesDeEvent rend un POINT DE CHOIX par couche : un ensemble de `.wem` dont le moteur
// joue EXACTEMENT UN.
//
// LA DISTINCTION QUI MANQUAIT. Aplatir tout le sous-arbre d'une action dans un seul
// ensemble perd la nature des noeuds traverses, et les deux natures ne se jouent pas pareil :
//
//	RandomSequence -> joue UN enfant tire au sort  : ses `.wem` sont des VARIANTES
//	Blend, ActorMixer, Switch resolu -> jouent TOUS leurs enfants : ce sont des COUCHES
//
// Symptome mesure, signale par l'utilisateur sur le MA40 : « un tir est bien et le suivant
// etouffe ». Son evenement de 3e personne est un unique `Blend` de 64 `.wem` ; le rendu en
// tirait UN seul par coup, donc une piece du melange au lieu du melange. En descendant
// jusqu'aux points de choix, ce `Blend` rend une couche par enfant, et chaque coup les
// empile toutes — ce que fait le moteur.
//
// SIMPLIFICATION ASSUMEE : un `RandomSequence` en mode SEQUENCE joue ses enfants dans
// l'ordre plutot qu'un seul. Le mode n'est pas lu ; les pools observes sur les armes sont
// des variantes (22, 14, 8 sons pour un meme tir), donc « un seul » est le cas courant.
func (b *bank) couchesDeEvent(id uint32) []brancheRendue {
	var out []brancheRendue
	for _, idAction := range b.Events[id] {
		if cible, ok := b.Actions[idAction]; ok {
			out = append(out, b.pointsDeChoix(cible, map[uint32]bool{})...)
		}
	}
	return out
}

// pointsDeChoix descend jusqu'aux noeuds ou le moteur TRANCHE, et rend un ensemble par
// tranchage. Un noeud « tous ses enfants » se subdivise ; un noeud « un seul enfant » forme
// un ensemble avec tout ce qu'il porte.
func (b *bank) pointsDeChoix(n uint32, vus map[uint32]bool) []brancheRendue {
	if vus[n] {
		return nil
	}
	vus[n] = true
	if w, estSon := b.Sons[n]; estSon {
		return []brancheRendue{b.brancheDe(n, []uint32{w})}
	}
	o, connu := b.Objets[n]
	enfants := b.Enfants[n]
	if !connu || len(enfants) == 0 {
		return nil
	}
	if o.Type == typeRandomSeq {
		return []brancheRendue{b.brancheDe(n, b.wemsSous(n, map[uint32]bool{}))}
	}
	if o.Type == typeBlend {
		// UN `Blend` N'EMPILE PAS TROIS FOIS LE MEME SON. Mesure : ses enfants ont des
		// durees IDENTIQUES entre eux (fusil electrique 0,67/0,67/0,67 ; MA40
		// 0,08/0,08/0,08). Des couches d'un coup de feu differeraient — une attaque breve,
		// un corps, une queue longue. Trois elements de meme duree sont des ALTERNATIVES,
		// et l'inventaire dit lesquelles : 42 `Blend` sur 303 portent une automation par
		// parametre de jeu, soit un fondu de DISTANCE (proche / moyen / lointain).
		//
		// Les deux comportements precedents etaient donc faux tous les deux : tirer au
		// hasard dans l'ensemble des enfants changeait de distance a chaque coup (« un tir
		// est bien et le suivant etouffe »), et les empiler tous donnait un melange trop
		// epais (« aucun des reconstitues ne convient plus » sur le fusil electrique).
		// On fige UNE distance, toujours la meme — ce que la decision de l'utilisateur
		// autorise : le rejeu 2D n'a pas besoin de gerer la distance.
		return b.pointsDeChoix(distanceRetenue(enfants), vus)
	}
	var out []brancheRendue
	for _, e := range enfants {
		out = append(out, b.pointsDeChoix(e, vus)...)
	}
	return out
}

// distanceRetenue choisit l'enfant de `Blend` a rendre. Le choix doit etre STABLE d'une
// regeneration a l'autre — sinon les votes deja poses porteraient sur un son qui bouge —
// donc on prend le plus petit identifiant, et non le premier de la liste declaree, dont
// l'ordre depend de la lecture.
func distanceRetenue(enfants []uint32) uint32 {
	meilleur := enfants[0]
	for _, e := range enfants[1:] {
		if e < meilleur {
			meilleur = e
		}
	}
	return meilleur
}

// wemsSous rend tous les `.wem` d'un sous-arbre : c'est le contenu d'un point de choix.
func (b *bank) wemsSous(n uint32, vus map[uint32]bool) []uint32 {
	set := map[uint32]bool{}
	var descendre func(uint32)
	descendre = func(x uint32) {
		if vus[x] {
			return
		}
		vus[x] = true
		if w, ok := b.Sons[x]; ok {
			set[w] = true
		}
		for _, e := range b.Enfants[x] {
			descendre(e)
		}
	}
	descendre(n)
	return trier(set)
}

func (b *bank) brancheDe(cible uint32, wems []uint32) brancheRendue {
	t := "inconnu"
	if o, ok := b.Objets[cible]; ok {
		t = nomType(o.Type)
	}
	gains := map[string]float32{}
	for _, w := range wems {
		if g, ok := b.Gains[w]; ok {
			gains[fmt.Sprintf("%d", w)] = g
		}
	}
	return brancheRendue{
		Cible: fmt.Sprintf("%08x", cible), TypeNoeud: t, Wems: wems, Gains: gains,
	}
}

// arborescence affiche, pour chaque evenement d'une arme, la structure de sa hierarchie.
//
// La bank est designee par le `.pck` de l'arme, ou DIRECTEMENT par son identifiant quand
// aucun pack ne lui correspond — cas de la Carabine Vestige, dont tous les sons sont
// embarques dans la bank. Sans pack, la validation des `sourceID` repose sur l'index large
// et sur les medias embarques (que `parserBank` accepte d'office).
func arborescence(cheminModule, cheminPck string, profondeurMax int, gidBank uint32) error {
	var ids map[uint32]bool
	if cheminPck != "" {
		var err error
		if ids, err = idsPck(cheminPck); err != nil {
			return err
		}
	} else if gidBank == 0 {
		return fmt.Errorf("le mode arbre exige -pck ou -sbnk")
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	var f himodule.File
	var brut []byte
	score := 0
	if cheminPck != "" {
		if f, brut, score, err = trouverSbnk(m, ids); err != nil {
			return err
		}
	} else if f, brut, err = bankParIdentifiant(m, gidBank); err != nil {
		return err
	}
	b, err := parserBank(brut, validateurWem(ids))
	if err != nil {
		return err
	}
	nom := nomArme(cheminPck)
	if cheminPck == "" {
		nom = fmt.Sprintf("sbnk_%08x", gidBank)
	}
	fmt.Printf("arme     : %s\n", nom)
	fmt.Printf("sbnk     : %08x (score %d, %d media(s) embarque(s))\n",
		f.GlobalID, score, len(b.Embarques))
	fmt.Printf("hierarchie: %d objets, %d Events\n\n", len(b.Objets), len(b.Events))

	evs := make([]uint32, 0, len(b.Events))
	for id := range b.Events {
		evs = append(evs, id)
	}
	sort.Slice(evs, func(i, j int) bool {
		return len(b.wemsDeEvent(evs[i])) > len(b.wemsDeEvent(evs[j]))
	})
	rap := rapportCouches{Arme: nom}
	for _, id := range evs {
		w := b.wemsDeEvent(id)
		if len(w) == 0 {
			continue
		}
		couches := b.couchesDeEvent(id)
		fmt.Printf("=== event %08x : %d .wem au total, %d COUCHE(S) parallele(s) ===\n",
			id, len(w), len(couches))
		for _, c := range couches {
			fmt.Printf("  couche %s (%s) : %d variante(s) possible(s)\n",
				c.Cible, c.TypeNoeud, len(c.Wems))
		}
		for _, idAction := range b.Events[id] {
			if cible, ok := b.Actions[idAction]; ok {
				b.afficherNoeud(cible, 1, profondeurMax)
			}
		}
		fmt.Println()
		rap.Events = append(rap.Events, eventCouches{
			IDEvent: fmt.Sprintf("%08x", id), Branches: couches, Total: len(w),
		})
	}
	if sortieCouches != "" {
		blob, err := json.MarshalIndent(rap, "", " ")
		if err != nil {
			return err
		}
		fmt.Printf("couches ecrites : %s\n", sortieCouches)
		return os.WriteFile(sortieCouches, blob, 0o644)
	}
	return nil
}

// sortieCouches : fichier JSON des couches, renseigne par le drapeau -out du mode arbre.
var sortieCouches string

// indexLarge : identifiant `.wem` -> pack qui le contient, sur TOUS les packs du jeu.
// Renseigne au demarrage. Vide, on retombe sur la validation etroite d'origine.
var indexLarge map[uint32]string

// validateurWem rend le predicat de validation d'un `sourceID`.
//
// Large des que l'index est disponible : une couche partagee entre armes doit pouvoir se
// resoudre, sinon le coup reconstitue est incomplet sans qu'on sache pourquoi.
func validateurWem(idsDuPck map[uint32]bool) func(uint32) bool {
	if len(indexLarge) > 0 {
		return func(id uint32) bool { _, ok := indexLarge[id]; return ok }
	}
	return func(id uint32) bool { return idsDuPck[id] }
}

// afficherNoeud descend la hierarchie en indiquant le type et le nombre d'enfants.
func (b *bank) afficherNoeud(id uint32, niveau, max int) {
	indent := ""
	for i := 0; i < niveau; i++ {
		indent += "  "
	}
	o, connu := b.Objets[id]
	if !connu {
		fmt.Printf("%s- %08x (inconnu)\n", indent, id)
		return
	}
	enfants := b.Enfants[id]
	if wem, estSon := b.Sons[id]; estSon {
		fmt.Printf("%s- %-14s %08x  -> wem %d\n", indent, nomType(o.Type), id, wem)
		return
	}
	fmt.Printf("%s- %-14s %08x  %d enfant(s)\n", indent, nomType(o.Type), id, len(enfants))
	if niveau >= max {
		if len(enfants) > 0 {
			fmt.Printf("%s  ... (profondeur max atteinte)\n", indent)
		}
		return
	}
	for i, e := range enfants {
		if i >= 6 {
			fmt.Printf("%s  ... et %d autres enfants\n", indent, len(enfants)-6)
			return
		}
		b.afficherNoeud(e, niveau+1, max)
	}
}

// statsTypes compte les objets par type : vue d'ensemble avant de plonger dans l'arbre.
func (b *bank) statsTypes() map[string]int {
	out := map[string]int{}
	for _, o := range b.Objets {
		out[nomType(o.Type)]++
	}
	return out
}
