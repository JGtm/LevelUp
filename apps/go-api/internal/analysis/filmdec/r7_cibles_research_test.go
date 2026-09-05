package filmdec

// r7_cibles_research_test.go — lot R7 : LA QUESTION DE FOND. Le RÉPULSEUR (type 104
// `EquipmentKnockbackPlayer`) et le PROPULSEUR (types 42 `biped_dodge` et 43
// `initiate_mobility_action`) laissent-ils une trace dans le film, oui ou non ?
//
// R5 a etabli qu'ils n'ont AUCUNE tete de paquet sur 325 160 paquets delta. R6 a sonde la
// deuxieme position derriere 597 tetes fermees : rien non plus. Ce test marche la liste
// ENTIERE de chaque paquet et publie le resultat AVEC SES DENOMINATEURS : combien de listes
// marchees jusqu'au bout, combien d'evenements traverses, sur combien de films — car un
// negatif sans denominateur n'est pas un resultat.
//
// TROIS CATEGORIES DE CIBLES :
//   - REPULSEUR : 104 (effet applique au joueur pousse), 119 (requete), 105 (objet pousse) ;
//   - PROPULSEUR : 42 (esquive de bipede), 43 (action de mobilite) ;
//   - USAGE GENERIQUE : 30 (biped_equipment_activation), 93, 48, 31, 28, 51, 98, 115, 116.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0, balayage borne (R7_CHUNKS).
//
//	CGO_ENABLED=0 R7_ROOT=... R7_IDS=... R7_CAT=... R7_MAPS=... \
//	  go test ./internal/analysis/filmdec/ -run '^TestR7Cibles$' -count=1 -timeout 60m -v

import (
	"path/filepath"
	"sort"
	"testing"
)

// r7Cibles : les types dont le sort doit etre tranche, par famille.
var r7CiblesFamille = map[string][]int{
	"REPULSEUR":       {104, 119, 105},
	"PROPULSEUR":      {42, 43},
	"USAGE GENERIQUE": {30, 93, 48, 31, 28, 51, 98, 115, 116},
}

// r7TypesJustes : les types dont la LARGEUR DE CHARGE est validee par TestR7Largeur, c'est-a-
// dire mesuree sur des listes a UN SEUL evenement (cadrage certain) avec une profondeur de
// trame >= 70 % de la mediane. Y figurent aussi les types a charge 0 bit, dont la largeur ne
// peut pas etre fausse.
//
// N'Y FIGURENT PAS, et c'est le coeur du verdict de ce lot :
//   - 5 `projectile_detonate` : profondeur 0,308 contre une mediane de 1,793 — verdict FAUX ;
//   - 0 `damage_aftermath` : 1,161 — verdict DOUTEUX (grammaire de PRODUCTION, non modifiee) ;
//   - 23 `authority_ignored_predicted_position` : 1,173 — DOUTEUX ;
//   - tous les types trop rares pour etre juges (< 30 listes a un evenement).
var r7TypesJustes = map[int]bool{
	36: true, 15: true, 82: true, 21: true, 38: true, 76: true, 75: true, 9: true,
	6: true, 39: true, 1: true, 103: true, 7: true,
	// charges a 0 bit : largeur non falsifiable
	3: true, 4: true, 24: true, 25: true, 26: true, 33: true, 49: true, 54: true,
	57: true, 59: true, 92: true,
}

// r7ChainePropre dit si tous les evenements STRICTEMENT AVANT l'indice i ont une largeur de
// charge validee. C'est la condition pour que le cadrage de l'evenement i soit lui-meme sur.
func r7ChainePropre(evs []r7Ev, i int) bool {
	for k := 0; k < i; k++ {
		if !r7TypesJustes[evs[k].Typ] {
			return false
		}
	}
	return true
}

// r7EstCible dit si le type appartient a une famille cible.
func r7EstCible(typ int) bool {
	for _, l := range r7CiblesFamille {
		for _, c := range l {
			if c == typ {
				return true
			}
		}
	}
	return false
}

// r7CibleOcc : une occurrence d'un type cible trouvee dans une liste.
type r7CibleOcc struct {
	Film     string
	Chunk    int
	TsUS     uint64
	Typ      int
	Pos      int
	Ref0     uint64
	HasRef0  bool
	BitDebut int
	Pay      []byte
}

// TestR7Cibles chasse les types cibles dans la liste complete, avec denominateurs.
func TestR7Cibles(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	var (
		films                               int
		listes, listesCompletes, evenements int
		listesVides, paquetsDelta           int
		opaques                             = map[int]int{}
		trouvees                            []r7CibleOcc
		vusParType                          = map[int]int{}
		vusChainePropre                     = map[int]int{}
		predecesseurs                       = map[int]map[int]int{}
		tetes                               = map[int]int{}
	)
	for _, id := range ids {
		dir := filepath.Join(root, id)
		n := r7Chunks(dir)
		if n == 0 {
			continue
		}
		films++
		ctx := cartes[id]
		fListes, fCompletes, fEv, fCibles := 0, 0, 0, 0
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				paquetsDelta++
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 {
					listesVides++
					continue
				}
				listes++
				fListes++
				evs, stop, typ, _ := r7Marche(pay, ctx)
				evenements += len(evs)
				fEv += len(evs)
				if len(evs) > 0 {
					tetes[evs[0].Typ]++
				}
				if stop == r7StopFin {
					listesCompletes++
					fCompletes++
				} else if stop == r7StopOpaque {
					opaques[typ]++
				}
				for i, e := range evs {
					vusParType[e.Typ]++
					if r7ChainePropre(evs, i) {
						vusChainePropre[e.Typ]++
					}
					if !r7EstCible(e.Typ) {
						continue
					}
					if predecesseurs[e.Typ] == nil {
						predecesseurs[e.Typ] = map[int]int{}
					}
					if i == 0 {
						predecesseurs[e.Typ][-1]++ // en TETE : aucun predecesseur
					} else {
						predecesseurs[e.Typ][evs[i-1].Typ]++
					}
					fCibles++
					cp := append([]byte(nil), pay...)
					trouvees = append(trouvees, r7CibleOcc{Film: id, Chunk: c, TsUS: pk.TimestampUS,
						Typ: e.Typ, Pos: e.Pos, Ref0: e.Ref0, HasRef0: e.HasRef0,
						BitDebut: e.BitDebut, Pay: cp})
				}
			}
		}
		t.Logf("film %s : %d listes (%d marchees jusqu'au bout, %.1f %%) · %d evenements lus · %d cibles",
			id, fListes, fCompletes, 100*float64(fCompletes)/float64(max(1, fListes)), fEv, fCibles)
	}
	t.Logf("")
	t.Logf("=== DENOMINATEURS : %d films · %d paquets delta · %d listes vides · %d listes non vides ===",
		films, paquetsDelta, listesVides, listes)
	t.Logf("=== COUVERTURE : %d listes marchees INTEGRALEMENT (%.1f %%) · %d evenements traverses ===",
		listesCompletes, 100*float64(listesCompletes)/float64(max(1, listes)), evenements)
	if len(opaques) > 0 {
		t.Logf("=== RESTE OPAQUE (marche interrompue) : %s ===", r7Compte(opaques, 20))
	}
	t.Logf("=== PARC DES TYPES TRAVERSES : %s ===", r7Compte(vusParType, 40))
	t.Logf("")
	// EPREUVE DE REFUTATION : qui PRECEDE une cible dans la liste ? Si un seul type domine,
	// c'est SA grammaire qui est fausse et la « cible » n'est que la derive qui en resulte.
	// Un type cible reellement emis serait precede par la distribution ordinaire des types.
	for _, fam := range []string{"REPULSEUR", "PROPULSEUR", "USAGE GENERIQUE"} {
		for _, typ := range r7CiblesFamille[fam] {
			if predecesseurs[typ] == nil || sommeMap(predecesseurs[typ]) == 0 {
				continue
			}
			t.Logf("PREDECESSEURS de %d %s (%d occurrences) : %s", typ, r7Noms[typ],
				sommeMap(predecesseurs[typ]), r7Compte(predecesseurs[typ], 8))
		}
	}
	t.Logf("(pour memoire, distribution ordinaire des types traverses : %s)",
		r7Compte(vusParType, 8))
	t.Logf("")
	// LE TABLEAU DU VERDICT. Trois colonnes qui, ensemble, tranchent :
	//   - « vus » : occurrences brutes, toutes chaines confondues ;
	//   - « chaine propre » : occurrences dont TOUS les predecesseurs ont une largeur validee ;
	//   - « tetes » : occurrences en position 1, ou le cadrage est CERTAIN. C'est le juge :
	//     un type reellement emis doit y apparaitre a peu pres a la hauteur de sa part.
	totalEv := max(1, evenements)
	partTete := float64(listesCompletes) / float64(totalEv)
	t.Logf("part attendue d'un type en position 1 (si l'emission ne depend pas de la position) : %.1f %%",
		100*partTete)
	t.Logf("%-4s %-38s %7s %14s %7s %10s", "type", "nom", "vus", "chaine propre", "tetes", "attendu")
	for _, fam := range []string{"REPULSEUR", "PROPULSEUR", "USAGE GENERIQUE"} {
		t.Logf("-- %s --", fam)
		for _, typ := range r7CiblesFamille[fam] {
			t.Logf("%-4d %-38s %7d %14d %7d %10.1f", typ, r7Noms[typ], vusParType[typ],
				vusChainePropre[typ], tetes[typ], partTete*float64(vusParType[typ]))
		}
	}
	t.Logf("-- TEMOINS POSITIFS (types dont la largeur est validee) --")
	for _, typ := range []int{117, 21, 103, 100, 9, 38, 76} {
		t.Logf("%-4d %-38s %7d %14d %7d %10.1f", typ, r7Noms[typ], vusParType[typ],
			vusChainePropre[typ], tetes[typ], partTete*float64(vusParType[typ]))
	}
	if len(trouvees) == 0 {
		t.Logf("AUCUNE occurrence d'un type cible sur %d evenements traverses dans %d listes completes",
			evenements, listesCompletes)
		return
	}
	sort.Slice(trouvees, func(i, j int) bool {
		if trouvees[i].Typ != trouvees[j].Typ {
			return trouvees[i].Typ < trouvees[j].Typ
		}
		return trouvees[i].TsUS < trouvees[j].TsUS
	})
	t.Logf("=== %d OCCURRENCES DE TYPES CIBLES ===", len(trouvees))
	for i, o := range trouvees {
		if i >= 200 {
			t.Logf("  ... (+%d autres)", len(trouvees)-200)
			break
		}
		fin := o.BitDebut/8 + 12
		if fin > len(o.Pay) {
			fin = len(o.Pay)
		}
		t.Logf("  film %s chunk %d ts=%d : type %d %s en position %d · ref0=%d (presente=%v) · bit %d · octets % X",
			o.Film, o.Chunk, o.TsUS, o.Typ, r7Noms[o.Typ], o.Pos, o.Ref0, o.HasRef0, o.BitDebut,
			o.Pay[o.BitDebut/8:fin])
	}
}
