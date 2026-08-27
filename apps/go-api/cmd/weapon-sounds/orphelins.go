package main

// orphelins.go — le mode `orphelins` : POURQUOI un `.wem` embarque n'est atteint par aucun
// evenement de sa banque.
//
// LA QUESTION EST PRODUITE PAR UNE MESURE, pas par une intuition. Le balayage structurel
// compte, par banque, les medias qu'aucun evenement n'atteint. Deux resultats obligent a
// regarder : la banque du translocateur (`dcfaa487`) laisse orphelins ses DEUX PLUS LONGS
// sons (6,77 s et 6,22 s) alors que l'utilisateur cherche justement un son long, et les
// QUATRE banques de bobine en laissent SEIZE CHACUNE — un compte identique quatre fois de
// suite ne se produit pas par accident, c'est une regle de structure.
//
// UN ORPHELIN N'EST PAS FORCEMENT INJOIGNABLE. Le parcours d'evenement applique deux
// FILTRES, chacun justifie par une mesure anterieure, et chacun capable de rendre un son
// invisible sans qu'il soit mort :
//
//	Switch  seuls les enfants de l'ETAT PAR DEFAUT sont retenus (les autres etats existent
//	        et le jeu les joue quand il impose l'etat correspondant) ;
//	Blend   seuls les enfants AUDIBLES au point de reference des courbes sont retenus.
//
// Ce mode remonte donc l'orphelin vers ses PARENTS BRUTS (listes d'enfants non filtrees) et
// dit, a chaque saut, si le lien a survecu au filtrage. Trois issues, et elles ne se
// confondent pas :
//
//	CHEMIN COUPE PAR UN FILTRE  le son existe, le jeu le joue sous condition — laquelle est
//	                            nommee (etat de commutation, courbe de fondu) ;
//	AUCUN PARENT                le son n'est reference par personne dans cette banque : il
//	                            est joue depuis une AUTRE banque, ou il est mort ;
//	PARENT SANS EVENEMENT       la chaine remonte mais aucun Event ne la declenche ici.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// sautOrphelin : un pas de la remontee, du son vers ses referents.
type sautOrphelin struct {
	Parent   uint32
	Type     byte
	Retenu   bool   // le lien parent -> enfant a survecu au filtrage du parcours
	Pourquoi string // la raison quand il ne l'a pas
}

// diagnostiquerOrphelins est le mode `orphelins`. Il accepte plusieurs banques (`-banks`)
// pour ne charger le module qu'une fois — la contrainte memoire l'impose autant que le temps.
func diagnostiquerOrphelins(cheminModule string, banques []uint32, wems []uint32) error {
	if len(banques) == 0 {
		return fmt.Errorf("le mode orphelins exige -banks (identifiants de banques, hexa)")
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	for _, gid := range banques {
		if err := orphelinsDUneBanque(m, gid, wems); err != nil {
			fmt.Printf("\nbanque %08x : %v\n", gid, err)
		}
	}
	return nil
}

// orphelinsDUneBanque diagnostique les medias inatteignables d'une seule banque.
func orphelinsDUneBanque(m *himodule.Module, gidBank uint32, wems []uint32) error {
	b, err := banqueParGid(m, gidBank)
	if err != nil {
		return err
	}
	connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }

	// Les listes d'enfants BRUTES, avant tout filtrage : c'est la seule vue qui permette
	// de distinguer « coupe par un filtre » de « reference par personne ».
	brut := map[uint32][]uint32{}
	for id, o := range b.Objets {
		if o.Type == typeSound || o.Type == typeAction || o.Type == typeEvent {
			continue
		}
		if enf := lireEnfants(o.Data, connu); len(enf) > 0 {
			brut[id] = enf
		}
	}
	parents := map[uint32][]uint32{}
	for p, enfants := range brut {
		for _, e := range enfants {
			parents[e] = append(parents[e], p)
		}
	}

	// Si aucun `.wem` n'est demande, on prend les orphelins de la banque.
	if len(wems) == 0 {
		wems = orphelinsDeBanque(b)
	}
	fmt.Printf("\nbanque %08x : %d objets, %d medias embarques, %d orphelin(s) examine(s)\n",
		gidBank, len(b.Objets), len(b.Embarques), len(wems))

	for _, w := range wems {
		fmt.Printf("\n=== .wem %d ===\n", w)
		son, ok := objetSonDe(b, w)
		if !ok {
			fmt.Println("  AUCUN objet Sound de cette banque ne declare ce media.")
			fmt.Println("  -> il est joue depuis une AUTRE banque, ou il est mort.")
			continue
		}
		fmt.Printf("  objet Sound %08x\n", son)
		remonterOrphelin(b, brut, parents, son, 0)
	}
	return nil
}

// remonterOrphelin affiche la remontee d'un noeud vers ses referents, en disant a chaque
// saut si le lien a survecu au filtrage du parcours d'evenement — et sinon, pourquoi.
func remonterOrphelin(b *bank, brut, parents map[uint32][]uint32, n uint32, niveau int) {
	if niveau > 6 {
		fmt.Printf("%s  (remontee arretee a 6 niveaux)\n", indent(niveau))
		return
	}
	ps := parents[n]
	if len(ps) == 0 {
		fmt.Printf("%s  AUCUN PARENT dans cette banque.\n", indent(niveau))
		fmt.Printf("%s  -> les Events de la banque ne peuvent pas l'atteindre.\n", indent(niveau))
		return
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
	for _, p := range ps {
		s := evaluerSaut(b, p, n)
		etat := "lien RETENU"
		if !s.Retenu {
			etat = "lien COUPE : " + s.Pourquoi
		}
		fmt.Printf("%s  <- %s %08x  (%s)\n", indent(niveau), nomType(s.Type), p, etat)
		if evts := eventsVers(b, p); len(evts) > 0 {
			for _, e := range evts {
				fmt.Printf("%s     declenche par l'event %08x\n", indent(niveau), e)
			}
			continue
		}
		remonterOrphelin(b, brut, parents, p, niveau+1)
	}
}

// evaluerSaut dit si le lien parent -> enfant survit au filtrage, et sinon pourquoi.
func evaluerSaut(b *bank, parent, enfant uint32) sautOrphelin {
	s := sautOrphelin{Parent: parent}
	if o, ok := b.Objets[parent]; ok {
		s.Type = o.Type
	}
	for _, e := range b.Enfants[parent] {
		if e == enfant {
			s.Retenu = true
			return s
		}
	}
	if c, ok := b.Switchs[parent]; ok {
		for _, pq := range c.Paquets {
			for _, e := range pq.Enfants {
				if e != enfant {
					continue
				}
				s.Pourquoi = fmt.Sprintf("etat de commutation %d, alors que le defaut est %d "+
					"(groupe %d) — le jeu le joue quand il impose cet etat",
					pq.Etat, c.EtatDefaut, c.Groupe)
				return s
			}
		}
		s.Pourquoi = fmt.Sprintf("Switch groupe %d : l'enfant n'appartient a aucun etat declare", c.Groupe)
		return s
	}
	if s.Type == typeBlend {
		s.Pourquoi = "Blend : enfant inaudible au point de reference de la courbe de fondu"
		if c, ok := b.Blends[parent]; ok {
			if rtpc, entree, plein, trouve := entreeDansCourbe(c, enfant); trouve {
				s.Pourquoi = fmt.Sprintf("Blend pilote par le parametre de jeu %d : "+
					"l'enfant entre a x = %.3f et domine a x = %.3f (le rendu evalue a x minimal)",
					rtpc, entree, plein)
			}
		}
		return s
	}
	s.Pourquoi = "non retenu par le parcours"
	return s
}

// entreeDansCourbe rend, pour un enfant d'un `Blend`, le parametre de jeu qui le pilote, le
// x ou il DEVIENT audible et le x ou il atteint son plein gain. C'est la description de la
// phase : jusqu'a `entree` le jeu ne le joue pas, apres `plein` il le joue entier.
func entreeDansCourbe(c conteneurBlend, enfant uint32) (rtpc uint32, entree, plein float32, ok bool) {
	for _, l := range c.Cies {
		for _, a := range l.Assocs {
			if a.Enfant != enfant || len(a.C) == 0 {
				continue
			}
			entree, plein = a.C[0].X, a.C[len(a.C)-1].X
			for i, p := range a.C {
				if p.Y > 0 {
					if i > 0 {
						entree = a.C[i-1].X
					} else {
						entree = p.X
					}
					break
				}
			}
			for _, p := range a.C {
				if p.Y >= 1 {
					plein = p.X
					break
				}
			}
			return l.RTPC, entree, plein, true
		}
	}
	return 0, 0, 0, false
}

// eventsVers rend les Events dont une action « jouer » vise directement ce noeud.
func eventsVers(b *bank, n uint32) []uint32 {
	var out []uint32
	for id, actions := range b.Events {
		for _, ia := range actions {
			if b.Actions[ia] == n {
				out = append(out, id)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// objetSonDe rend l'objet Sound qui declare ce media.
func objetSonDe(b *bank, wem uint32) (uint32, bool) {
	for id, w := range b.Sons {
		if w == wem {
			return id, true
		}
	}
	return 0, false
}

// orphelinsDeBanque rend les medias embarques qu'aucun evenement n'atteint.
func orphelinsDeBanque(b *bank) []uint32 {
	atteints := map[uint32]bool{}
	for id := range b.Events {
		for _, w := range b.wemsDeEvent(id) {
			atteints[w] = true
		}
	}
	var out []uint32
	for w := range b.Embarques {
		if !atteints[w] {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func indent(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "  "
	}
	return s
}
