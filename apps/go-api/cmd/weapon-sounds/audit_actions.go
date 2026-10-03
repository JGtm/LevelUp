package main

// audit_actions.go — le mode `audit-actions` : CE QUE PORTE UNE EVENT ACTION.
//
// LE SIXIEME OUBLI DE FORMAT, ET IL EST DE LA MEME FAMILLE QUE LES CINQ AUTRES. Le parseur
// lit d'un objet `Action` (type 3) exactement deux choses : son TYPE (pour ne garder que les
// « jouer ») et sa CIBLE. Tout ce qui suit la cible dans la charge utile n'est jamais lu.
//
// Or c'est la que Wwise range le DELAI DE L'ACTION. Consequence directe et mesurable :
// `couchesDeEvent` (arbre.go) rend UNE COUCHE PAR ACTION et le rendu les somme A t = 0.
// Si un evenement enchaine « jouer A » puis « jouer B dans 2 s », nous rendons les deux
// superposes. Le symptome existe et il est date : sur l'evenement `71cb04b8` (avant
// apparition d'une nouvelle zone, KOTH), l'utilisateur entend « un tres court son au debut
// qui me parait EN TROP » — deux couches sommees a t = 0.
//
// LE RELEVE DE DELAIS DEJA FAIT NE DIT RIEN DE CELUI-CI : « 0 sur 275 couches » a ete mesure
// sur les NOEUDS (`AkPropID` porte par le Sound ou le conteneur), pas sur les ACTIONS.
//
// LAYOUT VISE (`CAkAction`, Wwise 2019+), depuis le debut de la charge utile :
//
//	+0  u16 ulActionType
//	+2  u32 idExt              (la cible, deja lue par `lireCibleAction`)
//	+6  u8  idExt_4            (bIsBus)
//	+7  AkPropBundle           u8 n | n x u8 idProp | n x f32
//	    AkPropBundle RANGED    u8 n | n x u8 idProp | n x 2 f32
//	    parametres specifiques au type d'action
//
// LE CONTROLE EST ECRIT AVANT LA MESURE, ET IL EST REFUTABLE PAR CONSTRUCTION. Pour une
// action « jouer », les parametres specifiques sont EXACTEMENT `u8 byBitVector | u32 bankID`,
// soit 5 octets. Le decodage est donc juste si, et seulement si, le nombre d'octets RESTANTS
// apres les deux paquets vaut 5. Ce n'est pas une plausibilite floue : c'est une egalite.
//
// TEMOIN NEGATIF, obligatoire (lecon 4 de `RECETTE_SONS_ARMES`) : la meme lecture est tentee
// aux offsets 6 et 8, qui sont faux d'un octet. Si le taux de « reste = 5 » y est comparable
// a celui de l'offset 7, la mesure ne prouve rien et il faut le dire.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

// offsetPropsAction : l'offset THEORIQUE du premier paquet de proprietes d'une action.
const offsetPropsAction = 7

// resteAttenduPlay : octets specifiques d'une action « jouer » apres les deux paquets
// (`u8 byBitVector | u32 bankID`).
const resteAttenduPlay = 5

// actionLue : ce qu'une Event Action porte reellement, une fois sa charge utile decodee.
type actionLue struct {
	ID     string                `json:"action"`
	Type   uint16                `json:"type"`
	Cible  string                `json:"cible"`
	Props  map[string]float32    `json:"props,omitempty"`
	Ranged map[string][2]float32 `json:"ranged,omitempty"`
	Reste  int                   `json:"reste_octets"`
	Lu     bool                  `json:"lu"`
}

// evenementActions : un evenement et la LISTE ORDONNEE de ses actions.
type evenementActions struct {
	Event   string      `json:"event"`
	Actions []actionLue `json:"actions"`
}

// banqueActions : le detail d'une banque ciblee.
type banqueActions struct {
	Bank       string             `json:"bank"`
	Evenements []evenementActions `json:"evenements"`
}

// plageProp : l'etendue des valeurs observees pour un identifiant de propriete.
type plageProp struct {
	N        int
	Min, Max float32
	NonNul   int
}

func (p *plageProp) noter(v float32) {
	if p.N == 0 || v < p.Min {
		p.Min = v
	}
	if p.N == 0 || v > p.Max {
		p.Max = v
	}
	p.N++
	if v != 0 {
		p.NonNul++
	}
}

// statsActions : les compteurs de l'audit, regroupes pour ne pas passer huit entiers.
type statsActions struct {
	Reste       map[int]map[int]int
	Props       map[byte]*plageProp
	RangedIDs   map[byte]int
	Dist        map[int]int
	Actions     int
	Play        int
	Evenements  int
	MultiPlay   int
	AvecPropVal int
}

// auditActions balaie les banques d'un module et statue sur le contenu des Event Actions.
// `cibles` vide = aucun detail JSON, seules les statistiques globales sont produites.
func auditActions(cheminModule string, cibles map[uint32]bool, sortie string) error {
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	offsets := []int{6, offsetPropsAction, 8}
	st := &statsActions{
		Reste:     map[int]map[int]int{},
		Props:     map[byte]*plageProp{},
		RangedIDs: map[byte]int{},
		Dist:      map[int]int{},
	}
	for _, o := range offsets {
		st.Reste[o] = map[int]int{}
	}
	var detail []banqueActions

	for _, f := range m.Files("sbnk") {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		debut := indexBKHD(data)
		if debut < 0 {
			continue
		}
		b, err := parserBank(data[debut:], func(uint32) bool { return false })
		if err != nil {
			continue
		}
		balayerActions(b, offsets, st)
		if d, ok := detailBanque(b, f.GlobalID, cibles, st); ok {
			detail = append(detail, d)
		}
	}

	imprimerAuditActions(offsets, st)

	if sortie == "" {
		return nil
	}
	blob, err := json.MarshalIndent(detail, "", " ")
	if err != nil {
		return err
	}
	fmt.Printf("\ndetail ecrit : %s (%d banque(s))\n", sortie, len(detail))
	return os.WriteFile(sortie, blob, 0o644)
}

// balayerActions applique le controle d'offset et releve les proprietes d'une banque.
func balayerActions(b *bank, offsets []int, st *statsActions) {
	for _, o := range b.Objets {
		if o.Type != typeAction || len(o.Data) < 2 {
			continue
		}
		st.Actions++
		if byte(binary.LittleEndian.Uint16(o.Data)>>8) != actionPlay {
			continue
		}
		st.Play++
		for _, off := range offsets {
			a := lireActionA(o.Data, off)
			if !a.Lu {
				st.Reste[off][-1]++
				continue
			}
			st.Reste[off][a.Reste]++
		}
		a := lireActionA(o.Data, offsetPropsAction)
		if !a.Lu || a.Reste != resteAttenduPlay {
			continue
		}
		if len(a.Props) > 0 {
			st.AvecPropVal++
		}
		for nom, v := range a.Props {
			id := byte(atoiSafe(nom))
			if st.Props[id] == nil {
				st.Props[id] = &plageProp{}
			}
			st.Props[id].noter(v)
		}
		for nom := range a.Ranged {
			st.RangedIDs[byte(atoiSafe(nom))]++
		}
	}
}

// detailBanque compte les actions « jouer » par evenement et, si la banque est ciblee,
// rend le detail ordonne de ses evenements.
func detailBanque(b *bank, gid uint32, cibles map[uint32]bool, st *statsActions) (banqueActions, bool) {
	connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }
	out := banqueActions{Bank: fmt.Sprintf("%08x", gid)}
	vise := cibles[gid]
	for id, actions := range b.Events {
		st.Evenements++
		n := 0
		for _, ia := range actions {
			if _, ok := b.Actions[ia]; ok {
				n++
			}
		}
		st.Dist[n]++
		if n >= 2 {
			st.MultiPlay++
		}
		if !vise {
			continue
		}
		ev := evenementActions{Event: fmt.Sprintf("%08x", id)}
		for _, ia := range actions {
			o, ok := b.Objets[ia]
			if !ok {
				continue
			}
			a := lireActionA(o.Data, offsetPropsAction)
			a.ID = fmt.Sprintf("%08x", ia)
			if len(o.Data) >= 2 {
				a.Type = binary.LittleEndian.Uint16(o.Data)
			}
			if c, _, ok := lireCibleAction(o.Data, connu); ok {
				a.Cible = fmt.Sprintf("%08x", c)
			}
			ev.Actions = append(ev.Actions, a)
		}
		out.Evenements = append(out.Evenements, ev)
	}
	if !vise {
		return banqueActions{}, false
	}
	sort.Slice(out.Evenements, func(i, j int) bool {
		return out.Evenements[i].Event < out.Evenements[j].Event
	})
	return out, true
}

// imprimerAuditActions rend la sortie DANS L'ORDRE IMPOSE : le controle d'offset et son
// temoin negatif D'ABORD, les resultats ensuite. Une lecture dont le temoin negatif tient
// aussi bien que la these ne doit rien conclure, et le lecteur doit le voir avant le reste.
func imprimerAuditActions(offsets []int, st *statsActions) {
	fmt.Println()
	fmt.Println("=== 1. CONTROLE D'OFFSET, AVEC SON TEMOIN NEGATIF (avant tout resultat) ===")
	fmt.Printf("  actions balayees : %d, dont \"jouer\" : %d\n", st.Actions, st.Play)
	fmt.Println("  critere : apres les deux paquets, une action \"jouer\" doit laisser")
	fmt.Printf("            EXACTEMENT %d octets (u8 byBitVector | u32 bankID).\n", resteAttenduPlay)
	for _, off := range offsets {
		bon := st.Reste[off][resteAttenduPlay]
		etiquette := "TEMOIN NEGATIF (faux d'un octet)"
		if off == offsetPropsAction {
			etiquette = "THESE"
		}
		fmt.Printf("  offset %d  %-32s reste=%d : %d (%.2f %%)   illisible : %d\n",
			off, etiquette, resteAttenduPlay, bon,
			100*float64(bon)/float64(max(st.Play, 1)), st.Reste[off][-1])
	}

	fmt.Println()
	fmt.Println("=== 2. COMBIEN D'ACTIONS \"JOUER\" PAR EVENEMENT ===")
	fmt.Printf("  evenements balayes : %d\n", st.Evenements)
	cles := make([]int, 0, len(st.Dist))
	for k := range st.Dist {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	for _, k := range cles {
		fmt.Printf("  %2d action(s) : %6d (%.2f %%)\n", k, st.Dist[k],
			100*float64(st.Dist[k])/float64(max(st.Evenements, 1)))
	}
	fmt.Printf("  >= 2 actions : %d (%.2f %%) — CE SONT EUX QUE LE RENDU SOMME A t = 0\n",
		st.MultiPlay, 100*float64(st.MultiPlay)/float64(max(st.Evenements, 1)))

	fmt.Println()
	fmt.Println("=== 3. LES PROPRIETES PORTEES PAR LES ACTIONS \"JOUER\" ===")
	fmt.Printf("  actions au decodage valide portant au moins une propriete : %d\n", st.AvecPropVal)
	ids := make([]int, 0, len(st.Props))
	for k := range st.Props {
		ids = append(ids, int(k))
	}
	sort.Ints(ids)
	if len(ids) == 0 {
		fmt.Println("  (aucune) — AUCUNE action ne porte de propriete : pas de delai d'action.")
	}
	for _, id := range ids {
		p := st.Props[byte(id)]
		fmt.Printf("  idProp %3d : %6d occurrence(s), %6d non nulle(s), etendue [%g ; %g]\n",
			id, p.N, p.NonNul, p.Min, p.Max)
	}
	if len(st.RangedIDs) > 0 {
		fmt.Print("  paquet RANGED des actions, identifiants vus :")
		rids := make([]int, 0, len(st.RangedIDs))
		for k := range st.RangedIDs {
			rids = append(rids, int(k))
		}
		sort.Ints(rids)
		for _, id := range rids {
			fmt.Printf(" %d(x%d)", id, st.RangedIDs[byte(id)])
		}
		fmt.Println()
	}
}

// lireActionA decode une Event Action en supposant son paquet de proprietes a `off`.
//
// Lecture BRUTE : aucun filtre semantique n'est applique aux valeurs (le filtre
// `plausibleProp` est calibre sur les NOEUDS ; l'appliquer ici rejetterait par avance ce
// que l'audit cherche a decouvrir). Seules la borne du compteur et la finitude sont exigees.
func lireActionA(d []byte, off int) actionLue {
	var out actionLue
	simple, fin, ok := lirePaquetBrut(d, off, 1)
	if !ok {
		return out
	}
	large, fin2, ok := lirePaquetBrut(d, fin, 2)
	if !ok {
		return out
	}
	out.Lu = true
	out.Reste = len(d) - fin2
	if len(simple) > 0 {
		out.Props = map[string]float32{}
		for id, v := range simple {
			out.Props[fmt.Sprint(id)] = v[0]
		}
	}
	if len(large) > 0 {
		out.Ranged = map[string][2]float32{}
		for id, v := range large {
			out.Ranged[fmt.Sprint(id)] = [2]float32{v[0], v[1]}
		}
	}
	return out
}

// lirePaquetBrut lit un AkPropBundle sans filtre semantique.
func lirePaquetBrut(d []byte, off, largeur int) (map[byte][]float32, int, bool) {
	if off < 0 || off >= len(d) || largeur < 1 {
		return nil, off, false
	}
	n := int(d[off])
	if n > 16 {
		return nil, off, false
	}
	debutIDs := off + 1
	debutVals := debutIDs + n
	fin := debutVals + n*4*largeur
	if fin > len(d) {
		return nil, off, false
	}
	out := make(map[byte][]float32, n)
	for i := 0; i < n; i++ {
		id := d[debutIDs+i]
		comp := make([]float32, largeur)
		for c := 0; c < largeur; c++ {
			v := math.Float32frombits(binary.LittleEndian.Uint32(d[debutVals+(i*largeur+c)*4:]))
			f := float64(v)
			if math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > 1e9 {
				return nil, off, false
			}
			comp[c] = v
		}
		out[id] = comp
	}
	return out, fin, true
}

// atoiSafe convertit une cle numerique de map JSON, 0 si elle ne l'est pas.
func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
