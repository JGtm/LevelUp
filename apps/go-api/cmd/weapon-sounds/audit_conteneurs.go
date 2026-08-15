package main

// audit_conteneurs.go — le volet « conteneurs » de l'inventaire.
//
// Le mode `audit` mesurait les chunks, les types d'objets, les types d'action et la charge
// utile des `Sound`. Il ne mesurait RIEN sur les conteneurs, et c'est la que se cachait le
// quatrieme oubli du chantier. Ce volet repond a une seule question, type par type :
// **combien d'octets de la charge utile le parseur laisse-t-il derriere lui ?**
//
// Un conteneur dont on ignore la moitie de la charge utile est un defaut en attente. Le
// chiffre le rend visible AVANT qu'un rendu faux ne le revele.

import (
	"fmt"
	"sort"
)

type statConteneur struct {
	N            int
	SommeTaille  int
	MaxTaille    int
	AvecEnfants  int
	SommeRestant int // octets apres la liste d'enfants, donc jamais lus
	MaxRestant   int
	SommeAvant   int // octets avant la liste d'enfants, donc jamais lus non plus
}

// statsConteneurs accumule l'inventaire par type d'objet conteneur.
type statsConteneurs struct {
	parType map[byte]*statConteneur
	// Volet propre au `Switch` : ce que le nouveau decodeur arrive a lire.
	swTotal, swLus, swAvecDefaut int
	swSommePaquets               int
	swMaxPaquets                 int
	swEtatsSansEnfant            int
	// Volets `RandomSequence` et `Blend` : statuer ce qui reste non lu (gate 12a).
	rsLus, rsUniformes int
	blLus, blAvecRTPC  int
	blEchantillons     []string
}

func nouvellesStatsConteneurs() *statsConteneurs {
	return &statsConteneurs{parType: map[byte]*statConteneur{}}
}

// ajouter mesure un objet conteneur. `connu` valide l'appartenance a la bank courante.
func (s *statsConteneurs) ajouter(o objetHIRC, connu func(uint32) bool) {
	st := s.parType[o.Type]
	if st == nil {
		st = &statConteneur{}
		s.parType[o.Type] = st
	}
	st.N++
	st.SommeTaille += len(o.Data)
	if len(o.Data) > st.MaxTaille {
		st.MaxTaille = len(o.Data)
	}
	off, n := positionEnfants(o.Data, connu)
	if off < 0 {
		return
	}
	st.AvecEnfants++
	st.SommeAvant += off
	restant := len(o.Data) - (off + 4 + 4*n)
	st.SommeRestant += restant
	if restant > st.MaxRestant {
		st.MaxRestant = restant
	}
	switch o.Type {
	case typeRandomSeq:
		if p := lirePoidsAleatoire(o.Data, connu); p.Lu {
			s.rsLus++
			if p.Uniforme() {
				s.rsUniformes++
			}
		}
		return
	case typeBlend:
		if c := lireBlend(o.Data, connu); c.Lu {
			s.blLus++
			if c.PiloteParRTPC() {
				s.blAvecRTPC++
			}
		} else if len(s.blEchantillons) < 3 {
			// Un decodeur qui echoue doit montrer ce sur quoi il echoue, sinon on corrige
			// a l'aveugle. On garde les octets qui suivent la liste d'enfants.
			off, n := positionEnfants(o.Data, connu)
			if off >= 0 {
				s.blEchantillons = append(s.blEchantillons,
					fmt.Sprintf("%d enfants, %d octets apres : % x",
						n, len(o.Data)-(off+4+4*n), o.Data[off+4+4*n:]))
			}
		}
		return
	case typeSwitch:
	default:
		return
	}
	s.swTotal++
	c := lireSwitch(o.Data, connu)
	if !c.Lu {
		return
	}
	s.swLus++
	if c.EtatDefaut != 0 {
		s.swAvecDefaut++
	}
	s.swSommePaquets += len(c.Paquets)
	if len(c.Paquets) > s.swMaxPaquets {
		s.swMaxPaquets = len(c.Paquets)
	}
	for _, p := range c.Paquets {
		if len(p.Enfants) == 0 {
			s.swEtatsSansEnfant++
		}
	}
}

func (s *statsConteneurs) afficher() {
	fmt.Println("\n=== CONTENEURS : ce que le parseur laisse derriere lui ===")
	fmt.Println("  (« avant » et « apres » = octets de charge utile jamais lus)")
	fmt.Printf("  %-16s %7s %8s %9s %9s %9s\n",
		"type", "n", "taille", "avec enf.", "avant", "apres")
	tps := make([]byte, 0, len(s.parType))
	for t := range s.parType {
		tps = append(tps, t)
	}
	sort.Slice(tps, func(i, j int) bool { return s.parType[tps[i]].N > s.parType[tps[j]].N })
	for _, t := range tps {
		st := s.parType[t]
		fmt.Printf("  %-16s %7d %8.0f %8d%% %9.0f %9.0f  (max %d)\n",
			nomType(t), st.N, moy(st.SommeTaille, st.N),
			100*st.AvecEnfants/max(st.N, 1),
			moy(st.SommeAvant, st.AvecEnfants), moy(st.SommeRestant, st.AvecEnfants),
			st.MaxRestant)
	}

	if s.swTotal == 0 {
		return
	}
	fmt.Printf("\n=== CONTENEURS Switch : etat du nouveau decodeur ===\n")
	fmt.Printf("  conteneurs Switch avec liste d'enfants : %d\n", s.swTotal)
	fmt.Printf("  table etat -> enfants decodee et validee : %d (%.0f %%)\n",
		s.swLus, 100*float64(s.swLus)/float64(max(s.swTotal, 1)))
	fmt.Printf("  dont un etat par defaut recoupe par la table : %d\n", s.swAvecDefaut)
	fmt.Printf("  etats par conteneur : moyenne %.1f, maximum %d\n",
		moy(s.swSommePaquets, s.swLus), s.swMaxPaquets)
	fmt.Printf("  etats declares SANS aucun enfant : %d\n", s.swEtatsSansEnfant)
	fmt.Printf("  => un Switch dont aucun etat ne correspond ne joue RIEN ; melanger ses\n")
	fmt.Printf("     enfants dans un seul lot aleatoire est donc doublement faux\n")

	fmt.Printf("\n=== RandomSequence : la table de poids ===\n")
	fmt.Printf("  table lue et validee : %d\n", s.rsLus)
	fmt.Printf("  dont poids tous egaux : %d (%.0f %%)\n",
		s.rsUniformes, 100*float64(s.rsUniformes)/float64(max(s.rsLus, 1)))
	fmt.Printf("  => si la quasi-totalite est uniforme, tirer au hasard sans poids est exact\n")

	fmt.Printf("\n=== Blend : automation des couches ===\n")
	fmt.Printf("  table des couches lue et validee : %d\n", s.blLus)
	fmt.Printf("  dont au moins une couche pilotee par un parametre de jeu : %d\n", s.blAvecRTPC)
	fmt.Printf("  => si ce nombre est non nul, « le Blend joue toutes ses couches » est FAUX\n")
	for _, e := range s.blEchantillons {
		fmt.Printf("  non decode : %s\n", e)
	}
}

func moy(somme, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(somme) / float64(n)
}
