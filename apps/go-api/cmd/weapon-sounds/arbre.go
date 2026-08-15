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

// couchesDeEvent rend une entree par ACTION, avec les `.wem` atteignables depuis elle.
func (b *bank) couchesDeEvent(id uint32) []brancheRendue {
	var out []brancheRendue
	for _, idAction := range b.Events[id] {
		cible, ok := b.Actions[idAction]
		if !ok {
			continue
		}
		vus := map[uint32]bool{}
		set := map[uint32]bool{}
		var descendre func(uint32)
		descendre = func(n uint32) {
			if vus[n] {
				return
			}
			vus[n] = true
			if w, estSon := b.Sons[n]; estSon {
				set[w] = true
			}
			for _, e := range b.Enfants[n] {
				descendre(e)
			}
		}
		descendre(cible)
		t := "inconnu"
		if o, ok := b.Objets[cible]; ok {
			t = nomType(o.Type)
		}
		out = append(out, brancheRendue{
			Cible: fmt.Sprintf("%08x", cible), TypeNoeud: t, Wems: trier(set),
		})
	}
	return out
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
