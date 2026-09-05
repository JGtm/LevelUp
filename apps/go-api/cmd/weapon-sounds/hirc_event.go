package main

// hirc_event.go — mode `hirc-event` : le DUMP COMPLET d'un evenement, et son PLAN DE RENDU.
//
// POURQUOI CE MODE (lot V3E, 2026-09-02). Retour utilisateur, verbatim : « tu n'as rendu que
// les fichiers isoles, pas reconstitues avec leur format wwise et reglages de package ». Les
// modes existants (`eqip-arbre`, `arbre`) rendent bien la STRUCTURE d'un evenement, mais le
// gain qu'ils publient est le gain du chemin DESCENDANT seul (evenement -> conteneurs ->
// Sound), sans :
//
//	le chemin MONTANT (actor-mixer parents, via `DirectParentID`) ;
//	le BUS de sortie (`OverrideBusId`) et ses parents ;
//	le `MakeUpGain` (AkPropID 6), jamais lu par `proprietes.go` ;
//	les offsets de noeud (`InitialDelay`) au-dela du delai d'ACTION.
//
// Ce mode lit tout cela (`hirc_noeuds.go`), le publie noeud par noeud avec les OCTETS BRUTS
// de toute propriete non decodee, et en tire un PLAN DE RENDU exploitable par `rendu-event` :
// par couche, la liste des variantes avec leur gain de chemin COMPLET, leur decalage, leur
// hauteur et la fourchette RANGED que le moteur leur applique a chaque lecture.
//
// Usage :
//
//	ws -etroit -mode hirc-event -banks <gids hexa> [-events <gids hexa>] [-etats <ids dec>]
//	   -out plan.json
//
// `-events` vide = tous les evenements de la banque. `-etats` force des etats de conteneur
// `Switch` (regime moteur) ; absent, l'etat par defaut s'applique — et le lot V3C a MESURE
// que le defaut est le ralenti, pas la conduite.
//
// MEMOIRE : un module a la fois (7,24 Go pour `pc/globals`, 2,86 Go pour `pc/globals/common`).

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

// dumpeurV3E porte l'etat d'un dump : la banque parsee et la table des noeuds complets.
type dumpeurV3E struct {
	bk        *bank
	base      map[uint32]nodeBase
	etats     map[uint32]bool
	profil    map[string]int
	bus       map[string]float64
	inconnues *[]string
	hexBank   string
}

// noterInconnue consigne une propriete que la table `nomsAkProp` ne nomme pas, avec ses
// octets bruts. Le mandat du lot est explicite : un reglage non decodable se DIT.
func (d *dumpeurV3E) noterInconnue(id uint32, typ byte, p propBrute, suffixe string) {
	if _, connu := nomsAkProp[p.ID]; connu {
		return
	}
	*d.inconnues = append(*d.inconnues, fmt.Sprintf(
		"sbnk %s noeud %08x (%s) prop %d%s : octets [%s] = %g en flottant, %d en u32",
		d.hexBank, id, nomType(typ), p.ID, suffixe, p.Octets, p.Valeur, p.Bits))
}

// dumperEvenements est le mode `hirc-event`.
func dumperEvenements(cheminModule string, banques, events map[uint32]bool, etats map[uint32]bool, sortie string) error {
	if len(banques) == 0 {
		return fmt.Errorf("le mode hirc-event exige -banks (identifiants sbnk, hexa, virgules)")
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	rap := v3eRapport{Module: cheminModule, ProfilProps: map[string]int{}, Bus: map[string]float64{}}
	gids := make([]uint32, 0, len(banques))
	for g := range banques {
		gids = append(gids, g)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })

	for _, gid := range gids {
		d, err := nouveauDumpeur(m, gid, etats, rap.ProfilProps, rap.Bus, &rap.Inconnues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sbnk %08x : %v\n", gid, err)
			continue
		}
		ids := make([]uint32, 0, len(d.bk.Events))
		for id := range d.bk.Events {
			if len(events) == 0 || events[id] {
				ids = append(ids, id)
			}
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			for _, ev := range d.evenementParEtat(id) {
				rap.Events = append(rap.Events, ev)
				afficherEvenementV3E(ev)
			}
		}
	}
	afficherProfilProps(rap)
	if sortie == "" {
		return nil
	}
	blob, err := json.MarshalIndent(rap, "", " ")
	if err != nil {
		return err
	}
	fmt.Printf("\nplan ecrit : %s\n", sortie)
	return os.WriteFile(sortie, blob, 0o644)
}

// nouveauDumpeur ouvre une banque et lit le `NodeBaseParams` COMPLET de chacun de ses objets.
func nouveauDumpeur(m *himodule.Module, gid uint32, etats map[uint32]bool,
	profil map[string]int, bus map[string]float64, inconnues *[]string) (*dumpeurV3E, error) {
	_, brut, err := bankParIdentifiant(m, gid)
	if err != nil {
		return nil, err
	}
	ch := chunks(brut)
	emb := mediasEmbarques(ch)
	estWem := func(id uint32) bool {
		if _, ok := emb[id]; ok {
			return true
		}
		_, large := indexLarge[id]
		return large
	}
	bk, err := parserBank(brut, estWem)
	if err != nil {
		return nil, err
	}
	d := &dumpeurV3E{
		bk: bk, base: map[uint32]nodeBase{}, etats: etats, profil: profil, bus: bus,
		inconnues: inconnues, hexBank: fmt.Sprintf("%08x", gid),
	}
	for id, o := range bk.Objets {
		var nb nodeBase
		switch o.Type {
		case typeSound:
			nb = lireNodeBase(o.Data, decalageNodeBaseSound)
		case 8, 19: // Bus, AuxBus
			nb = lireBusHirc(o.Data)
			if nb.Lu {
				bus[fmt.Sprintf("%08x", id)] = nb.gainDeBus()
			}
		case typeAction, typeEvent:
			continue
		default:
			nb = lireNodeBase(o.Data, 0)
		}
		d.base[id] = nb
		for _, p := range nb.Props {
			profil[fmt.Sprintf("%d %s", p.ID, p.Nom)]++
			d.noterInconnue(id, o.Type, p, "")
		}
		for _, p := range nb.Ranged {
			profil[fmt.Sprintf("%d %s (RANGED)", p.ID, p.Nom)]++
			d.noterInconnue(id, o.Type, p, " RANGED")
		}
	}
	return d, nil
}

// amont remonte la hierarchie actor-mixer depuis un noeud et rend le gain cumule des
// PARENTS (le noeud lui-meme exclu), le bus effectif et le gain de la chaine de bus.
//
// Un parent absent de la banque n'est pas un zero : il est SIGNALE (`resolu` faux), parce
// qu'un gain inconnu suppose nul est exactement l'erreur que ce lot corrige.
func (d *dumpeurV3E) amont(n uint32) (gain float64, chaine string, busHex string, busResolu bool) {
	busResolu = true
	vus := map[uint32]bool{n: true}
	// Bus effectif : premier `OverrideBusId` non nul en remontant, noeud compris.
	cur := n
	for {
		nb, ok := d.base[cur]
		if !ok {
			break
		}
		if nb.BusOverride != 0 {
			busHex = fmt.Sprintf("%08x", nb.BusOverride)
			break
		}
		if nb.ParentDirect == 0 || vus[nb.ParentDirect] {
			break
		}
		cur = nb.ParentDirect
		vus[cur] = true
	}
	// Chaine de parents.
	vus = map[uint32]bool{n: true}
	cur = n
	for {
		nb, ok := d.base[cur]
		if !ok || nb.ParentDirect == 0 || vus[nb.ParentDirect] {
			break
		}
		p := nb.ParentDirect
		vus[p] = true
		pb, connu := d.base[p]
		if !connu {
			chaine += fmt.Sprintf(" <- %08x(HORS BANQUE)", p)
			busResolu = false
			break
		}
		gain += pb.gainPropre()
		chaine += fmt.Sprintf(" <- %08x(%s,%+.1fdB)", p, nomType(d.bk.Objets[p].Type), pb.gainPropre())
		cur = p
	}
	// Chaine de bus, si le bus vit dans cette banque.
	if busHex != "" {
		if g, ok := d.bus[busHex]; ok {
			gain += g
			chaine += fmt.Sprintf(" | bus %s %+.1fdB", busHex, g)
		} else {
			busResolu = false
			chaine += fmt.Sprintf(" | bus %s HORS BANQUE", busHex)
		}
	}
	return gain, chaine, busHex, busResolu
}

// evenementParEtat publie un evenement UNE FOIS PAR ETAT de regime quand il traverse un
// conteneur `Switch`, une seule fois sinon.
//
// POURQUOI. Le lot V3C a montre que l'etat par DEFAUT d'un moteur est le ralenti : rendre
// le defaut, c'est rendre le mauvais regime. Choisir l'etat « en conduite » demande de
// MESURER les medias de chaque etat, donc de tous les avoir. Les publier tous en une charge
// de module evite huit rechargements de 2,9 a 7,2 Go.
func (d *dumpeurV3E) evenementParEtat(id uint32) []v3eEvent {
	base := d.evenement(id)
	if len(base.Couches) == 0 {
		return nil
	}
	var etats []uint32
	for _, n := range base.Noeuds {
		for _, e := range n.SwitchEtats {
			if !contientEtat(etats, e.Etat) {
				etats = append(etats, e.Etat)
			}
		}
	}
	if len(etats) == 0 {
		return []v3eEvent{base}
	}
	sort.Slice(etats, func(i, j int) bool { return etats[i] < etats[j] })
	out := make([]v3eEvent, 0, len(etats))
	garde := d.etats
	for _, e := range etats {
		d.etats = map[uint32]bool{e: true}
		ev := d.evenement(id)
		ev.Etat = e
		if len(ev.Couches) > 0 {
			out = append(out, ev)
		}
	}
	d.etats = garde
	return out
}

func contientEtat(l []uint32, v uint32) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// evenement assemble le dump complet d'un evenement.
func (d *dumpeurV3E) evenement(id uint32) v3eEvent {
	ev := v3eEvent{Bank: d.hexBank, Event: fmt.Sprintf("%08x", id)}
	touches := map[uint32]bool{}
	for _, idAction := range d.bk.Events[id] {
		o := d.bk.Objets[idAction]
		var brut uint16
		if len(o.Data) >= 2 {
			brut = uint16(o.Data[0]) | uint16(o.Data[1])<<8
		}
		a := v3eAction{ID: fmt.Sprintf("%08x", idAction), Brut: brut,
			Type: nomActionV3E(brut), DelaiS: d.bk.DelaiAction[idAction]}
		cible, ok := d.bk.Actions[idAction]
		if !ok {
			a.Cible = "(non jouee)"
			ev.Actions = append(ev.Actions, a)
			continue
		}
		a.Cible = fmt.Sprintf("%08x", cible)
		ev.Actions = append(ev.Actions, a)

		gAmont, chaine, _, _ := d.amont(cible)
		depart := cheminV3E{Gain: gAmont, Delai: float64(d.bk.DelaiAction[idAction]),
			Texte: fmt.Sprintf("action %08x(+%.3fs)%s", idAction, d.bk.DelaiAction[idAction], chaine)}
		couches := d.descendre(cible, depart, map[uint32]bool{}, touches)
		for i := range couches {
			couches[i].GainAmont = arrondi3(gAmont)
		}
		ev.Couches = append(ev.Couches, couches...)
	}
	ids := make([]uint32, 0, len(touches))
	for n := range touches {
		ids = append(ids, n)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, n := range ids {
		ev.Noeuds = append(ev.Noeuds, d.noeud(n))
	}
	return ev
}

// descendre applique la semantique de `arbre.go` (prouvee au chantier armes) mais avec le
// gain de chemin COMPLET et les offsets de noeud.
func (d *dumpeurV3E) descendre(n uint32, etat cheminV3E, vus, touches map[uint32]bool) []v3eCouche {
	if vus[n] {
		return nil
	}
	vus[n] = true
	touches[n] = true
	o, connu := d.bk.Objets[n]
	if !connu {
		return nil
	}
	etat = etat.avecNoeud(d.base[n], fmt.Sprintf("%08x(%s,%+.1fdB)", n, nomType(o.Type), d.base[n].gainPropre()))
	if w, estSon := d.bk.Sons[n]; estSon {
		return []v3eCouche{d.couche(n, map[uint32]cheminV3E{w: etat}, map[uint32]uint32{w: n})}
	}
	enfants := d.enfantsDe(n)
	if len(enfants) == 0 {
		return nil
	}
	if o.Type == typeRandomSeq {
		pool := map[uint32]cheminV3E{}
		src := map[uint32]uint32{}
		d.collecter(n, etat, map[uint32]bool{n: true}, pool, src, touches)
		if len(pool) == 0 {
			return nil
		}
		return []v3eCouche{d.couche(n, pool, src)}
	}
	var out []v3eCouche
	for _, e := range enfants {
		suivant := etat
		if fondu, ok := d.bk.GainsFondu[n]; ok {
			suivant.Gain += fondu[e]
		}
		out = append(out, d.descendre(e, suivant, vus, touches)...)
	}
	return out
}

// enfantsDe rend les enfants a jouer : pour un `Switch`, ceux de l'etat FORCE s'il est
// declare (option `-etats`), sinon ceux retenus par `parserBank` (l'etat par defaut).
func (d *dumpeurV3E) enfantsDe(n uint32) []uint32 {
	if c, ok := d.bk.Switchs[n]; ok && len(d.etats) > 0 {
		for _, p := range c.Paquets {
			if d.etats[p.Etat] && len(p.Enfants) > 0 {
				return p.Enfants
			}
		}
	}
	return d.bk.Enfants[n]
}

// collecter rassemble les variantes d'un point de choix, chacune avec l'etat de SON chemin.
func (d *dumpeurV3E) collecter(n uint32, etat cheminV3E, vus map[uint32]bool,
	pool map[uint32]cheminV3E, src map[uint32]uint32, touches map[uint32]bool) {
	for _, e := range d.enfantsDe(n) {
		if vus[e] {
			continue
		}
		vus[e] = true
		touches[e] = true
		o := d.bk.Objets[e]
		suivant := etat.avecNoeud(d.base[e], fmt.Sprintf("%08x(%s,%+.1fdB)", e, nomType(o.Type), d.base[e].gainPropre()))
		if fondu, ok := d.bk.GainsFondu[n]; ok {
			suivant.Gain += fondu[e]
		}
		if w, estSon := d.bk.Sons[e]; estSon {
			if _, deja := pool[w]; !deja {
				pool[w] = suivant
				src[w] = e
			}
			continue
		}
		d.collecter(e, suivant, vus, pool, src, touches)
	}
}

// busDe rend le bus de sortie EFFECTIF d'un noeud : le premier `OverrideBusId` non nul en
// remontant depuis lui. Il se calcule au niveau de la COUCHE, pas au niveau de la cible de
// l'action : un conteneur de couche peut router vers un autre bus que son parent, et c'est
// une difference de chemin qui change le melange.
func (d *dumpeurV3E) busDe(n uint32) (string, bool) {
	vus := map[uint32]bool{}
	for cur := n; !vus[cur]; {
		vus[cur] = true
		nb, ok := d.base[cur]
		if !ok {
			break
		}
		if nb.BusOverride != 0 {
			hex := fmt.Sprintf("%08x", nb.BusOverride)
			_, resolu := d.bus[hex]
			return hex, resolu
		}
		if nb.ParentDirect == 0 {
			break
		}
		cur = nb.ParentDirect
	}
	return "", false
}

// couche transforme un point de choix en couche publiable.
func (d *dumpeurV3E) couche(cible uint32, pool map[uint32]cheminV3E, src map[uint32]uint32) v3eCouche {
	c := v3eCouche{Cible: fmt.Sprintf("%08x", cible), TypeNoeud: nomType(d.bk.Objets[cible].Type)}
	c.BusEffectif, c.BusResolu = d.busDe(cible)
	wems := make([]uint32, 0, len(pool))
	for w := range pool {
		wems = append(wems, w)
	}
	sort.Slice(wems, func(i, j int) bool { return wems[i] < wems[j] })
	var vmin, vmax, pmin, pmax float32
	for _, w := range wems {
		e := pool[w]
		c.Variantes = append(c.Variantes, v3eVariante{
			Wem: w, Noeud: fmt.Sprintf("%08x", src[w]),
			GainDB: arrondi3(e.Gain), DelaiS: arrondi3(e.Delai), PitchCts: arrondi3(e.Pitch),
		})
		vmin, vmax = min32(vmin, e.VolMin), max32(vmax, e.VolMax)
		pmin, pmax = min32(pmin, e.PitchMin), max32(pmax, e.PitchMax)
		c.Chemin = e.Texte
	}
	if vmin != 0 || vmax != 0 {
		c.RangedVolume = &[2]float32{vmin, vmax}
	}
	if pmin != 0 || pmax != 0 {
		c.RangedPitch = &[2]float32{pmin, pmax}
	}
	if bl := d.bk.boucleDeCouche(cible); bl.Lu && bl.Repetitions != 1 {
		r := bl.Repetitions
		c.Repetitions = &r
		c.ModeTransition, c.TransitionS = bl.Mode, bl.TransitionS
	}
	connu := func(id uint32) bool { _, ok := d.bk.Objets[id]; return ok }
	if mr := lireModeRanSeq(d.bk.Objets[cible].Data, connu); mr.Lu {
		c.Sequence, c.Continu = mr.Sequence, mr.Continu
	}
	return c
}

// noeud rend la vue publiable d'un noeud, table de `Switch` comprise.
func (d *dumpeurV3E) noeud(n uint32) v3eNoeud {
	o := d.bk.Objets[n]
	v := v3eNoeud{ID: fmt.Sprintf("%08x", n), Type: nomType(o.Type),
		Base: d.base[n], GainPropre: arrondi3(d.base[n].gainPropre()), Wem: d.bk.Sons[n]}
	for _, e := range d.bk.Enfants[n] {
		v.Enfants = append(v.Enfants, fmt.Sprintf("%08x", e))
	}
	if c, ok := d.bk.Switchs[n]; ok {
		v.SwitchGroupe, v.SwitchDefaut = c.Groupe, c.EtatDefaut
		for _, p := range c.Paquets {
			e := etatSwitch{Etat: p.Etat}
			for _, enf := range p.Enfants {
				e.Enfants = append(e.Enfants, fmt.Sprintf("%08x", enf))
				e.Wems = append(e.Wems, d.wemsSous(enf)...)
			}
			v.SwitchEtats = append(v.SwitchEtats, e)
		}
		sort.Slice(v.SwitchEtats, func(i, j int) bool { return v.SwitchEtats[i].Etat < v.SwitchEtats[j].Etat })
	}
	return v
}
