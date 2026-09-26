package grammar

// frame_closure.go — LA CARTE DE FERMETURE DES TRAMES DELTA (lot J4.0 du plan de suite de
// l audit du decodeur, decision DT-12, 2026-09-26). UN INSTRUMENT : ni la cuisson ni le
// collecteur ne l appellent, et aucune sortie de production ne change.
//
// # CE QU ELLE PROLONGE
//
// [KeyframeClosure] dit, archetype par archetype, combien de records d IMAGE-CLE la grammaire
// traverse jusqu au premier bit du record suivant. [FrameClosure] pose la meme question aux
// trames DELTA — celles qui portent le match instant par instant — par VUE et par ARCHETYPE, avec
// le meme contrat : fermes, total, bloquant le PLUS FREQUENT, departage par nom.
//
// # CE QUE « FERME » VEUT DIRE DANS UNE TRAME DELTA
//
// Une trame delta se lit EN SEQUENCE (bit de configuration, vue A des messages, vue B des
// entites, vue C de controle) : aucun balayeur d ancres ne donne la frontiere d un record
// independamment de la marche. Le seul oracle independant est la FIN DU PAQUET — la vue C, dernier
// rang, doit finir sur son terminateur avec un reste de 0 a 7 bits nuls ([vueCFermee]). Un paquet
// qui satisfait cet oracle est FERME AU BIT PRES, et TOUS les records qu il porte le sont avec lui :
// une largeur fausse n importe ou devant decalerait la fin. Un record n est donc ferme que si son
// paquet l est ; un paquet qui ne ferme pas ne prouve aucun de ses records, et son bloquant est la
// premiere cause qui l a arrete.
//
// # CE QU ELLE NE FAIT PAS
//
// Elle ne lit AUCUN bit a cote des marcheurs : elle rejoue la marche de production
// ([ScanMarcheDesTrames] — liaison des images-cles, table anticipee, localisation des listes
// d evenements, [DecodeFrameViewsCurseur]) et CLASSE ce que ces marcheurs rendent (records,
// rangs, verdict de la vue C publie au crochet). Elle ne tient aucun etat de paquet (ADR 0034
// D-5) : tout vit dans la mesure, construite par appel. Elle ne lit aucun fichier : l usage
// produit des composants lui est PASSE ([UsagesProduit]), l appelant le tire de `ecs_table.tsv`.
//
// # LA MESURE QUI DECIDE : LES RECORDS UTILES
//
// Le declencheur du chantier de representation intermediaire (spec du 2026-09-25, §9) n est pas
// « 95 % des paquets » : c est la fermeture des records qui PORTENT une donnee que le produit lit
// — un composant a usage produit (colonne `product_use`) annonce par le masque du record — et des
// entrees de la vue de controle. Les premiers se comptent A PART ([FrameUtileStat]). Les
// secondes ne se comptent qu une fois lues, donc que dans une vue C fermee : leur denominateur
// est le PAQUET (une vue C par paquet), pas l entree — une vue C qui ne ferme pas ne dit pas
// combien d entrees elle portait.

import (
	"errors"
	"fmt"
	"strings"
)

// VueDeTrame nomme l un des trois rangs d une trame delta (`FUN_142987460`).
type VueDeTrame int

// Les trois vues, dans l ordre de la marche.
const (
	// VueMessages : rang 0, la vue A (`frame_vue_messages.go`).
	VueMessages VueDeTrame = iota
	// VueEntites : rang 1, la vue B, celle des records d entite.
	VueEntites
	// VueControle : rang 2, la vue C (`frame_vue_controle.go`).
	VueControle
	// NombreDeVues est le nombre de rangs d une trame delta.
	NombreDeVues = 3
)

// ComposantUtile identifie un composant a usage produit : l archetype, et le NOM du composant sous
// son orthographe canonique ([CleComposant]).
type ComposantUtile struct {
	TI  int
	Nom string
}

// UsagesProduit est l ensemble des composants dont le produit lit la valeur — la colonne
// `product_use` de `ecs_table.tsv`, que l APPELANT lit et passe ici.
type UsagesProduit map[ComposantUtile]bool

// suffixeComposant est le suffixe que certains builds ecrivent et d autres non (`biped-slide` /
// `biped-slide-component`) : la cle l ignore, comme les dispatchs acceptent les deux.
const suffixeComposant = "-component"

// CleComposant rend la cle d un composant, orthographe canonique (sans le suffixe `-component`).
func CleComposant(ti int, nom string) ComposantUtile {
	return ComposantUtile{TI: ti, Nom: strings.TrimSuffix(nom, suffixeComposant)}
}

// FrameViewStat est la fermeture d UNE vue sur un film, comptee en paquets.
type FrameViewStat struct {
	// Atteints : paquets ou la marche a atteint la vue.
	Atteints int
	// Terminees : paquets ou la vue s est lue jusqu a son terminateur.
	Terminees int
	// Fermes : paquets ou la vue s est terminee ET ou le paquet se ferme au bit pres.
	Fermes int
	// Arrets : par cause, les paquets ou la vue atteinte ne s est pas terminee (vues A et B) ou
	// ne ferme pas le paquet (vue C).
	Arrets map[string]int
}

// FrameArchetypeStat est la fermeture des records d UN archetype de la vue B sur un film.
type FrameArchetypeStat struct {
	// Neufs / NeufsFermes : records NEW lus, et ceux dont le paquet se ferme.
	Neufs, NeufsFermes int
	// Deltas / DeltasFermes : records DELTA lus, et ceux dont le paquet se ferme.
	Deltas, DeltasFermes int
	// Utiles / UtilesFermes : ceux de ces records qui portent un composant a usage produit.
	Utiles, UtilesFermes int
	// Blocking nomme la cause LA PLUS FREQUENTE parmi les records non fermes — leur propre
	// composant sans lecteur, sinon ce qui a arrete leur paquet. A egalite, le nom le plus petit.
	Blocking string
}

// FrameUtileStat est la fermeture de ce que le produit lit : le declencheur de la spec, §9.
type FrameUtileStat struct {
	// Records / RecordsFermes : records de la vue B dont le masque annonce un composant utile.
	Records, RecordsFermes int
	// EntreesDeControleFermees : entrees `kind 0` lues dans les vues C FERMEES.
	EntreesDeControleFermees int
}

// FrameBlockerStat est ce qu UNE cause d arret coute : les paquets qu elle arrete, et les records
// utiles LUS dans ces paquets (borne superieure de ce que son port fermerait : un autre bloquant
// peut suivre, et les records d apres l arret ne sont pas lus du tout).
type FrameBlockerStat struct {
	// TI, Index et Composant nomment le composant sans lecteur ; -1 / -1 / "" pour une cause qui
	// n est pas un composant (vue, liste d evenements, slot non lie).
	TI, Index int
	Composant string
	// Paquets : paquets dont c est la PREMIERE cause d arret.
	Paquets int
	// UtilesEnJeu : records utiles lus et non fermes dans ces paquets.
	UtilesEnJeu int
}

// FrameClosureReport est la carte de fermeture des trames delta d UN film.
type FrameClosureReport struct {
	// Paquets : paquets delta marches (listes d evenements non localisees comprises).
	Paquets int
	// PaquetsFermes : paquets fermes au bit pres (oracle de la vue C).
	PaquetsFermes int
	// ListesNonLocalisees : paquets a liste d evenements dont le debut n a pas ete trouve — aucune
	// vue n est lue.
	ListesNonLocalisees int
	// Vues : la fermeture de chaque vue, par [VueDeTrame].
	Vues [NombreDeVues]FrameViewStat
	// Archetypes : par archetype ; la cle -1 regroupe les records dont l archetype n a pas ete
	// resolu (delta d un slot non lie, inference en echec).
	Archetypes map[int]FrameArchetypeStat
	// Utiles : les records et entrees que le produit lit.
	Utiles FrameUtileStat
	// Bloquants : par cause, ce qu elle arrete.
	Bloquants map[string]FrameBlockerStat
}

// ArchetypeNonResolu est la cle de [FrameClosureReport.Archetypes] des records sans archetype.
const ArchetypeNonResolu = -1

// FrameClosure mesure la carte de fermeture des trames delta d un film, sous la marche de
// production ([ScanMarcheDesTrames]) et le profil que porte `fc`.
//
// LA MARCHE EST CELLE DE PRODUCTION, PAS UNE COPIE DE SES LECTURES : meme monde (liaisons des
// images-cles chunk par chunk, table anticipee), meme localisation des listes d evenements
// ([debutDeLaListe]), meme marcheur ([DecodeFrameViewsCurseur], trois vues). Seul le PILOTAGE —
// la boucle chunk -> paquet — est recopie de `movementStateScanner.marcher`, parce que celui-ci
// publie des etats de mouvement et non ce que la carte classe ; le brancher ici aurait touche
// une marche de production pour un instrument.
func FrameClosure(fc *FilmContext, utiles UsagesProduit) (FrameClosureReport, error) {
	if fc == nil {
		return FrameClosureReport{}, fmt.Errorf("filmdec: contexte de film nil — aucune fermeture a mesurer")
	}
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return FrameClosureReport{}, ErrNoFilmChunk
	}
	reg, err := fc.Registry()
	if err != nil {
		return FrameClosureReport{}, fmt.Errorf("filmdec: registre illisible, les trames n ont pas de grammaire: %w", err)
	}
	cfg := fc.CadreDeBalayage()
	if !cfg.Profil.Grammaire.ClassesDeVue {
		// La carte mesure la marche PAR CLASSES DE VUE, celle de production : un cadre qui ne la
		// porte pas ne publie pas le verdict de la vue C, et la mesure n aurait pas d oracle.
		return FrameClosureReport{}, errors.New("filmdec: la carte de fermeture exige la marche " +
			"par classes de vue (GrammaireBalayage.ClassesDeVue)")
	}
	m := nouvelleMesureDesTrames(reg, utiles, cfg)
	monde := NewWorld(reg)
	monde.PoserTableAnticipee(ConstruireTableAnticipee(fc))
	marche := fc.MarcheDImageCle()
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		monde.PoserChunkCourant(c)
		lierLeChunkAuMonde(monde, marche, data, pks, m.cfg.Obs)
		for _, pk := range pks {
			m.paquet(pk, data, monde)
		}
	}
	return m.rapport(), nil
}

// mesureDesTrames porte ce que la carte accumule d un paquet a l autre, pour UN film.
type mesureDesTrames struct {
	reg    *Registry
	utiles UsagesProduit
	// cfg est le cadre de la marche ; son observateur ne porte QUE le crochet de la vue C.
	cfg FrameConfig
	// vueC est le verdict que la marche a publie pour le paquet en cours.
	vueC LectureVueC
	rep  FrameClosureReport
	// bloquantsParTI : par archetype, combien de records non fermes chaque cause a arretes.
	bloquantsParTI map[int]map[string]int
}

// nouvelleMesureDesTrames ouvre une mesure sous le cadre `cfg`, dont elle remplace l observateur
// par le sien : un observateur qui ne recoit QUE le verdict de la vue C.
func nouvelleMesureDesTrames(reg *Registry, utiles UsagesProduit, cfg FrameConfig) *mesureDesTrames {
	m := &mesureDesTrames{reg: reg, utiles: utiles, bloquantsParTI: map[int]map[string]int{},
		rep: FrameClosureReport{Archetypes: map[int]FrameArchetypeStat{},
			Bloquants: map[string]FrameBlockerStat{}}}
	obs := NouvelleObservation()
	obs.VueControleHook = func(l LectureVueC) { m.vueC = l }
	cfg.Obs = obs
	m.cfg = cfg
	return m
}

// paquet marche UN paquet du film, comme `movementStateScanner.paquet` : les paquets delta
// seulement, et ceux a liste d evenements depuis le debut que [debutDeLaListe] leur trouve.
func (m *mesureDesTrames) paquet(pk FilmPacket, data []byte, w *World) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := movementStateSkipLeadBits
	if _, present := PacketHeadEventType(pay); present {
		debut, _ = debutDeLaListe(pay, w, m.cfg)
		if debut < 0 {
			m.listeNonLocalisee()
			return
		}
	}
	m.marcherPaquet(pay, w, debut)
}

// marcherPaquet marche UN payload delta depuis `debut` et le classe. La vue A n est lue que si la
// marche part de la TETE du paquet — la condition meme de `decodeFrameParRangs`.
func (m *mesureDesTrames) marcherPaquet(pay []byte, w *World, debut int) {
	m.vueC = LectureVueC{}
	recs, rangs, _ := DecodeFrameViewsCurseur(pay, w, m.cfg, MovementStateViews, debut)
	enTete := debut == m.cfg.PacketPreambleBits && m.cfg.PacketPreambleBits >= 1
	m.classer(paquetMarche{enTete: enTete, recs: recs, rangs: rangs, vueC: m.vueC})
}

// rapport rend la carte accumulee, bloquant de chaque archetype compris.
func (m *mesureDesTrames) rapport() FrameClosureReport {
	for ti, parCause := range m.bloquantsParTI {
		s := m.rep.Archetypes[ti]
		s.Blocking = composantLePlusBloquant(parCause)
		m.rep.Archetypes[ti] = s
	}
	return m.rep
}

// BloquantPrincipal rend la cause qui arrete le PLUS de paquets du film — le premier port a faire.
// A egalite, le nom le plus petit (la regle de [KeyframeClosure]). Vide quand tout ferme.
func (r FrameClosureReport) BloquantPrincipal() string {
	parCause := make(map[string]int, len(r.Bloquants))
	for nom, b := range r.Bloquants {
		parCause[nom] = b.Paquets
	}
	return composantLePlusBloquant(parCause)
}
