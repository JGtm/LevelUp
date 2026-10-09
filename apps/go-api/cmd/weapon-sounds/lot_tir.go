package main

// lot_tir.go — ETAPE 4, PASSE 2 : les sons de TIR de toutes les armes, en une ouverture
// du module `any/globals` (0,62 Go).
//
// PROCESSUS SEPARE DE LA PASSE 1 : celle-ci ouvre le module de 7,24 Go, celle-la celui de
// 0,62 Go. Les deux ne se croisent jamais en memoire, l'echange passe par le JSON.
//
// Chaine, pour chaque arme :
//
//	weap --[champ nomme « Weapon Fire Sound » de weap.xml]--> tags lsnd de tir
//	lsnd --[identifiants d'evenements portes]--> evenements
//	evenement --[rapport de la passe 1]--> .wem
//
// L'arme n'est PAS designee par un nom : un evenement appartient a une bank, et la bank a
// ete appariee a un `.pck` en passe 1. C'est l'evenement qui raccorde le `weap` au `.pck`.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"

	"levelup/go-api/internal/himodule"
)

type tirArme struct {
	Arme         string   `json:"arme"`
	Pck          string   `json:"pck"`
	WeapTags     []string `json:"weap_tags"`
	WeapRetenu   string   `json:"weap_retenu"`
	SonsDeTir    []string `json:"tags_son_de_tir"`
	EventsDeTir  []string `json:"events_de_tir"`
	WemsDeTir    []uint32 `json:"wems_de_tir"`
	WemsDuPck    int      `json:"wems_du_pck"`
	PartDuPckPct int      `json:"part_du_pck_pct"`
	// Cadence lue dans le tag (`barrels` -> `rounds per second`), en coups par minute.
	// Zero quand le tag n'en declare pas — on ne devine pas.
	CadenceMin int `json:"cadence_min_cpm"`
	CadenceMax int `json:"cadence_max_cpm"`
	// Modes : UN MODE DE TIR = UN TAG DE SON designe par « Weapon Fire Sound ». Le Ravageur
	// en a deux (tir normal et tir charge), le Sentinel Beam aussi (bref et continu). Les
	// fondre en un seul ensemble d'evenements rendait un coup qui sonnait a cote.
	Modes []modeDeTir `json:"modes"`
	// Variation : la fourchette du mode dominant. C'est le repli quand l'app ne sait pas de
	// quel mode vient le son qu'elle joue — le cas courant aujourd'hui, le rejeu 2D ne
	// distinguant pas les modes de tir.
	Variation *variationRendue `json:"variation,omitempty"`
}

type modeDeTir struct {
	TagSon string   `json:"tag_son"`
	Events []string `json:"events"`
	// Variation : fourchette de la couche dominante de ce mode, tous evenements confondus.
	// Absente quand aucun noeud du mode ne declare d'ecart : lecture pure.
	Variation *variationRendue `json:"variation,omitempty"`
}

// livrerTirLot resout les sons de tir de toutes les armes du rapport de passe 1.
func livrerTirLot(cheminModule, entree, sortie string) error {
	var passe1 rapportLot
	blob, err := os.ReadFile(entree)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(blob, &passe1); err != nil {
		return err
	}
	fmt.Printf("passe 1  : %d armes\n", len(passe1.Armes))

	// evenement -> index d'arme. C'est ce qui raccorde un `weap` a un `.pck`.
	armeDeEvent := map[uint32]int{}
	for i, a := range passe1.Armes {
		for _, e := range a.Evenements {
			armeDeEvent[e.IDEvent] = i
		}
	}
	fmt.Printf("events   : %d indexes\n", len(armeDeEvent))

	o, err := calculerOffsets()
	if err != nil {
		return err
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	groupes := make(map[uint32]string, 1<<17)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}

	sonParWeap, weapDuSon, cadences := sonsDeTirParArme(m, o, groupes)
	fmt.Printf("weap avec son de tir : %d | tags de son concernes : %d | cadences lues : %d\n",
		len(sonParWeap), len(weapDuSon), len(cadences))
	rapporterMemoire("weap analyses")

	// DEUX LECTURES, ET L'ORDRE COMPTE. Le lien DIRECT (le tag designe par le champ de tir)
	// est la preuve forte ; le suivi des relais est un REPLI pour les armes qu'il ne couvre
	// pas. Les mettre au meme rang a produit un faux : `bt_arczapper` se retrouvait rattache
	// au S7 Sniper parce qu'une chaine longue du sniper atteignait ses evenements.
	eventsDirect, eventsParTag := eventsDesSonsDetaille(m, weapDuSon, armeDeEvent)
	rapporterMemoire("liens directs analyses")

	elargi := etendreParDependances(m, weapDuSon, 4)
	fmt.Printf("tags de son apres suivi des relais : %d (etaient %d)\n", len(elargi), len(weapDuSon))
	eventsRelais := eventsDesSons(m, elargi, armeDeEvent)
	rapporterMemoire("relais analyses")

	rap := assembler(passe1, sonParWeap, eventsDirect, eventsRelais, armeDeEvent, cadences, eventsParTag)
	// Le nommage ne doit PAS dependre du tir : epee, marteau, tourelles et variantes PNJ
	// sont de vraies armes avec une icone. On complete par le lien `weap -> ... -> sbnk`.
	rap = completerRattachements(m, passe1, rap)
	rapporterMemoire("rattachements completes")
	sortieBlob, err := json.MarshalIndent(rap, "", " ")
	if err != nil {
		return err
	}
	fmt.Printf("\n%d armes avec sons de tir -> %s\n", len(rap), sortie)
	return os.WriteFile(sortie, sortieBlob, 0o644)
}

// sonsDeTirParArme rend, par `weap`, ses tags de son de tir, l'index inverse, et la cadence.
//
// La cadence n'est retenue que s'il y a EXACTEMENT un candidat plausible dans le tag :
// avec deux candidats, rien ne permettrait de trancher, et une cadence fausse est pire
// qu'une cadence absente (la rafale sonnerait faux sans qu'on sache pourquoi).
func sonsDeTirParArme(m *himodule.Module, o offsetsTir, groupes map[uint32]string) (
	map[uint32][]uint32, map[uint32][]uint32, map[uint32]cadenceArme) {
	parWeap := map[uint32][]uint32{}
	inverse := map[uint32][]uint32{}
	cadences := map[uint32]cadenceArme{}
	for _, f := range m.Files("weap") {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		gids := tirDansTag(data, o, groupes)
		if len(gids) == 0 {
			continue
		}
		parWeap[f.GlobalID] = gids
		for _, g := range gids {
			inverse[g] = append(inverse[g], f.GlobalID)
		}
		if c := lireCadences(data); len(c) == 1 {
			cadences[f.GlobalID] = c[0]
		}
	}
	return parWeap, inverse, cadences
}

// tirDansTag applique au tag la signature du champ « Weapon Fire Sound ».
func tirDansTag(data []byte, o offsetsTir, groupes map[uint32]string) []uint32 {
	t, err := ouvrirTagWeap(data)
	if err != nil {
		return nil
	}
	var tous []uint32
	for _, mode := range modesDansTag(t, data, groupes) {
		tous = append(tous, mode...)
	}
	sort.Slice(tous, func(i, j int) bool { return tous[i] < tous[j] })
	uniq := tous[:0]
	for i, g := range tous {
		if i == 0 || g != tous[i-1] {
			uniq = append(uniq, g)
		}
	}
	return uniq
}

// tailleElementTir : taille d'un element du tableau « Weapon Fire Sounds », en octets.
//
// MESUREE, pas postulee. Contre-epreuve sur deux armes : le pistolet a plasma porte un bloc
// de 192 octets dont le second sous-tableau « Variations » est au champ +108 = 12 + 96 ;
// le fusil d'assaut porte un bloc de 96 octets, donc UN element — dont les « Variations »
// comptent 3 references. C'est la distinction que je confondais : le fusil d'assaut a UN
// MODE A TROIS VARIANTES, et non trois modes. Compter les tags de son au lieu des elements
// du tableau annoncait des modes qui n'existent pas.
const tailleElementTir = 96

// modesDansTag rend, PAR MODE DE TIR, les tags de son designes par « Weapon Fire Sound ».
//
// Un mode = un ELEMENT du tableau « Weapon Fire Sounds ». Ses variantes sont les references
// de son sous-tableau « Variations », situe au champ 12 de l'element — donc a
// 12 + k*96 dans le bloc, pour le k-ieme mode.
func modesDansTag(t tagWeap, data []byte, groupes map[uint32]string) [][]uint32 {
	racine, err := t.blocRacine()
	if err != nil {
		return nil
	}
	var modes [][]uint32
	for _, bloc := range t.enfantsDe(racine) {
		_, taille := t.blocAbs(bloc)
		nb := taille / tailleElementTir
		if nb <= 0 {
			nb = 1
		}
		sous := t.enfantsDe(bloc)
		for k := 0; k < nb; k++ {
			blocVar, ok := sous[12+k*tailleElementTir]
			if !ok {
				continue
			}
			absV, tailleV := t.blocAbs(blocVar)
			if absV < 0 || tailleV < tailleRef {
				continue
			}
			if gids, tousSons := refsDeSons(data, absV, tailleV, groupes); tousSons && len(gids) > 0 {
				modes = append(modes, gids)
			}
		}
	}
	return modes
}

// groupesSonores : les classes de tags par lesquelles une chaine de son peut transiter.
// `stai` est un RELAIS : le Mutilator passe par `weap -> snd! -> stai -> snd! -> sbnk`,
// soit quatre niveaux, la ou la plupart des armes en font deux. S'arreter au tag designe
// par le champ de tir manquait donc toutes les armes a relais.
var groupesSonores = map[string]bool{"snd!": true, "lsnd": true, "stai": true}

// etendreParDependances suit, depuis les tags de son de tir, toute chaine qui reste dans
// les classes sonores. Rend l'ensemble ferme, relais compris.
func etendreParDependances(m *himodule.Module, depart map[uint32][]uint32, maxNiveaux int) map[uint32][]uint32 {
	// index gid -> groupe, lu dans la table d'entrees : gratuit, aucune decompression.
	groupes := make(map[uint32]string, 1<<17)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}
	parGid := make(map[uint32]himodule.File, 1<<15)
	for g := range groupesSonores {
		for _, f := range m.Files(g) {
			parGid[f.GlobalID] = f
		}
	}
	// Les associations s'ACCUMULENT : un tag atteint depuis deux armes appartient aux deux.
	// Ecraser (ou ignorer le deja-vu) donnait l'association au premier arrive — bug mesure :
	// `bt_arczapper` se retrouvait rattache au S7 Sniper via un relais partage, et le
	// Disrupteur disparaissait du rattachement.
	out := make(map[uint32]map[uint32]bool, len(depart))
	ajouter := func(gid uint32, armes []uint32) bool {
		s, existait := out[gid]
		if !existait {
			s = map[uint32]bool{}
			out[gid] = s
		}
		for _, a := range armes {
			s[a] = true
		}
		return !existait
	}
	frontiere := map[uint32][]uint32{}
	for k, v := range depart {
		ajouter(k, v)
		frontiere[k] = v
	}
	for niveau := 0; niveau < maxNiveaux && len(frontiere) > 0; niveau++ {
		suivante := map[uint32][]uint32{}
		for gid, armes := range frontiere {
			f, ok := parGid[gid]
			if !ok {
				continue
			}
			data, err := m.Extract(f)
			if err != nil {
				continue
			}
			for _, d := range dependances(data) {
				if !groupesSonores[d.Groupe] {
					continue
				}
				if ajouter(d.IDGlobal, armes) {
					suivante[d.IDGlobal] = armes
				}
			}
		}
		frontiere = suivante
	}

	// Un tag atteint depuis TROP d'armes est un point de passage commun, pas le son d'une
	// arme : le compter reviendrait a preter a chacune les evenements des autres.
	const maxArmesParTag = 3
	final := make(map[uint32][]uint32, len(out))
	ecartes := 0
	for gid, s := range out {
		if len(s) > maxArmesParTag {
			ecartes++
			continue
		}
		l := make([]uint32, 0, len(s))
		for a := range s {
			l = append(l, a)
		}
		sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
		final[gid] = l
	}
	if ecartes > 0 {
		fmt.Printf("tags ecartes comme points de passage communs : %d\n", ecartes)
	}
	return final
}

// eventsDesSons rend, par `weap`, les identifiants d'evenements portes par ses tags de son.
//
// Balayage par FENETRE de 4 octets avec consultation d'un ensemble, et non une recherche
// par evenement : sur ~1200 evenements candidats, chercher chacun separement serait mille
// fois plus lent que lire chaque position une fois.
func eventsDesSons(m *himodule.Module, weapDuSon map[uint32][]uint32, armeDeEvent map[uint32]int) map[uint32]map[uint32]bool {
	out, _ := eventsDesSonsDetaille(m, weapDuSon, armeDeEvent)
	return out
}

// eventsDesSonsDetaille rend aussi le detail PAR TAG DE SON — c'est ce qui permet de
// distinguer les modes de tir d'une meme arme.
func eventsDesSonsDetaille(m *himodule.Module, weapDuSon map[uint32][]uint32,
	armeDeEvent map[uint32]int) (map[uint32]map[uint32]bool, map[uint32]map[uint32]bool) {
	out := map[uint32]map[uint32]bool{}
	parTag := map[uint32]map[uint32]bool{}
	n := 0
	for _, f := range append(m.Files("lsnd"), m.Files("snd!")...) {
		armes, voulu := weapDuSon[f.GlobalID]
		if !voulu {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for o := 0; o+4 <= len(data); o++ {
			id := binary.LittleEndian.Uint32(data[o:])
			if _, ok := armeDeEvent[id]; !ok {
				continue
			}
			if parTag[f.GlobalID] == nil {
				parTag[f.GlobalID] = map[uint32]bool{}
			}
			parTag[f.GlobalID][id] = true
			for _, w := range armes {
				if out[w] == nil {
					out[w] = map[uint32]bool{}
				}
				out[w][id] = true
			}
		}
		n++
		if n%200 == 0 {
			runtime.GC()
		}
	}
	return out, parTag
}

// assembler croise les deux moities et rend le livrable par arme.
func assembler(p1 rapportLot, sonParWeap map[uint32][]uint32,
	eventsDirect, eventsRelais map[uint32]map[uint32]bool, armeDeEvent map[uint32]int,
	cadences map[uint32]cadenceArme, eventsParTag map[uint32]map[uint32]bool) []tirArme {
	// Un `weap` peut couvrir une arme ; plusieurs `weap` peuvent viser la meme (variantes).
	// On retient CELUI QUI COUVRE LE PLUS d'evenements de cette arme — decision assumee.
	indexer := func(src map[uint32]map[uint32]bool) map[int][]uint32 {
		out := map[int][]uint32{}
		for w := range src {
			vus := map[int]bool{}
			for e := range src[w] {
				if i, ok := armeDeEvent[e]; ok && !vus[i] {
					vus[i] = true
					out[i] = append(out[i], w)
				}
			}
		}
		return out
	}
	direct := indexer(eventsDirect)
	relais := indexer(eventsRelais)

	// Le repli ne sert QUE pour les armes qu'aucun lien direct ne couvre.
	parArme := direct
	eventsParWeap := eventsDirect
	parArmeRelais := map[int][]uint32{}
	for i, w := range relais {
		if _, deja := direct[i]; !deja {
			parArmeRelais[i] = w
		}
	}
	for i, w := range parArmeRelais {
		parArme[i] = w
	}
	var out []tirArme
	for i, weaps := range parArme {
		a := p1.Armes[i]
		// Les evenements de cette arme se lisent dans la source qui l'a fait entrer.
		if _, parRelais := parArmeRelais[i]; parRelais {
			eventsParWeap = eventsRelais
		} else {
			eventsParWeap = eventsDirect
		}
		// Tri DETERMINISTE : `sort.Slice` n'est pas stable, et plusieurs `weap` couvrent
		// souvent le meme nombre d'evenements. Sans depart au second critere, le tag retenu
		// changeait d'un run a l'autre — donc la cadence affichee aussi (mesure : le pistolet
		// a plasma passait de 405 a 1800 coups/min entre deux executions identiques).
		sort.Slice(weaps, func(x, y int) bool {
			nx := compteEvents(eventsParWeap[weaps[x]], armeDeEvent, i)
			ny := compteEvents(eventsParWeap[weaps[y]], armeDeEvent, i)
			if nx != ny {
				return nx > ny
			}
			return weaps[x] < weaps[y]
		})
		retenu := weaps[0]
		events := []uint32{}
		for e := range eventsParWeap[retenu] {
			if armeDeEvent[e] == i {
				events = append(events, e)
			}
		}
		sort.Slice(events, func(x, y int) bool { return events[x] < events[y] })
		wems := unionWemsLot(a, events)
		part := 0
		if a.WemsDuPck > 0 {
			part = len(wems) * 100 / a.WemsDuPck
		}
		t := tirArme{
			Arme: a.Arme, Pck: a.Pck, WeapRetenu: fmt.Sprintf("%08x", retenu),
			WemsDeTir: wems, WemsDuPck: a.WemsDuPck, PartDuPckPct: part,
		}
		if c, ok := cadences[retenu]; ok && c.Significative() {
			t.CadenceMin = int(c.Min*60 + 0.5)
			t.CadenceMax = int(c.Max*60 + 0.5)
		}
		for _, w := range weaps {
			t.WeapTags = append(t.WeapTags, fmt.Sprintf("%08x", w))
		}
		for _, g := range sonParWeap[retenu] {
			t.SonsDeTir = append(t.SonsDeTir, fmt.Sprintf("%08x", g))
		}
		for _, e := range events {
			t.EventsDeTir = append(t.EventsDeTir, fmt.Sprintf("%08x", e))
		}
		// UN MODE = UN TAG DE SON. Sans ce detail, les evenements des deux modes d'une
		// meme arme se melangent et le coup reconstitue peut venir du mauvais mode.
		t.Modes = modesDeArme(sonParWeap[retenu], eventsParTag, armeDeEvent, i, variationsDeArme(a))
		vars := make([]*variationRendue, 0, len(t.Modes))
		for _, m := range t.Modes {
			vars = append(vars, m.Variation)
		}
		t.Variation = variationDominante(vars)
		out = append(out, t)
	}
	sort.Slice(out, func(x, y int) bool { return out[x].Arme < out[y].Arme })
	return out
}

func compteEvents(set map[uint32]bool, armeDeEvent map[uint32]int, arme int) int {
	n := 0
	for e := range set {
		if armeDeEvent[e] == arme {
			n++
		}
	}
	return n
}

func unionWemsLot(a armeLot, events []uint32) []uint32 {
	garde := map[uint32]bool{}
	for _, e := range events {
		garde[e] = true
	}
	set := map[uint32]bool{}
	for _, ev := range a.Evenements {
		if !garde[ev.IDEvent] {
			continue
		}
		for _, w := range ev.Wems {
			set[w] = true
		}
	}
	out := make([]uint32, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
