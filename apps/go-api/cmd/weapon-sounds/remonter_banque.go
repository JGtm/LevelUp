package main

// remonter_banque.go — mode `remonter-banque` : de QUELLES banques part un tag donne ?
// Autrement dit, la chaine LUE A L'ENVERS — banque -> `snd!` -> ... -> `vehi`.
//
// POURQUOI CE MODE. Le lot des sons de DESTRUCTION (2026-09-02) trouve sur le disque six
// banques d'explosion de vehicule (`sb_008_exp_vehicle_{small,med,large}_{unsc,covenant}`)
// dont la STRUCTURE est sans ambiguite (un evenement a 4 couches simultanees, one-shot).
// Reste la question qui decide de tout : QUEL vehicule joue QUELLE taille ? La descente
// `vehi -> lsnd/snd!/effe -> sbnk` (`vehicules_sons.go`) ne repond pas : elle ne franchit
// qu'UN niveau d'`effe`, et n'atteint que les deux effets partages par les 13 vehicules.
//
// Ce mode part donc de la banque et REMONTE : les tags de son qui en dependent, puis, par
// balayage inline (`GlobalID` en clair dans le corps d'un tag — meme lecture que
// `refsSonInline`), les tags qui les referencent, niveau par niveau. Il s'arrete des qu'un
// `vehi` est atteint et imprime le chemin.
//
// COUT. Un passage par niveau sur TOUS les tags du module (`any/globals`, 0,62 Go). Un
// filtre de 65 536 bits sur les 16 bits bas des cibles rejette la quasi-totalite des offsets
// avant tout acces a la table (meme dispositif que `pck_banques.go`).
//
// Usage : -mode remonter-banque -banks <gids hexa, virgules> [-limite <niveaux, defaut 4>]

import (
	"encoding/binary"
	"fmt"
	"sort"

	"levelup/go-api/internal/himodule"
)

// remonterDepuisBanques est le mode `remonter-banque`.
func remonterDepuisBanques(cheminModule string, banques map[uint32]bool, niveaux int) error {
	if len(banques) == 0 {
		return fmt.Errorf("le mode remonter-banque exige -banks (identifiants sbnk, hexa, virgules)")
	}
	if niveaux <= 0 {
		niveaux = 4
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	r := &remontee{parent: map[uint32]uint32{}, groupe: map[uint32]string{}, banques: banques}
	cible := r.sonsDesBanques(m)
	fmt.Printf("niveau 0 : %d tag(s) de son dependent des %d banque(s) visee(s)\n", len(cible), len(banques))
	if len(cible) == 0 {
		return nil
	}
	r.afficherNiveau(cible)

	tous := m.Files("")
	vus := map[uint32]bool{}
	for id := range cible {
		vus[id] = true
	}
	for n := 1; n <= niveaux && len(cible) > 0; n++ {
		// SENTINELLES. `00000000` et `ffffffff` sont les valeurs « pas de reference » des
		// tables de tags : les laisser dans la cible fait matcher 46 618 tags au niveau 2
		// (mesure du 2026-09-02) et noie la remontee. Elles sortent de la cible.
		delete(cible, 0)
		delete(cible, 0xFFFFFFFF)
		if len(cible) == 0 {
			break
		}
		suivant := r.niveauSuivant(m, tous, cible, vus)
		fmt.Printf("\nniveau %d : %d tag(s) referencent le niveau precedent\n", n, len(suivant))
		r.afficherNiveau(suivant)
		for id := range suivant {
			vus[id] = true
		}
		cible = suivant
	}
	return nil
}

// remontee porte l'etat de la remontee : le parent de chaque tag atteint, son groupe, et
// les banques de depart (elles ferment l'affichage du chemin).
type remontee struct {
	parent  map[uint32]uint32 // tag -> tag de niveau precedent
	groupe  map[uint32]string
	banques map[uint32]bool
}

// sonsDesBanques rend les tags `snd!`/`lsnd` qui DEPENDENT d'une des banques visees.
func (r *remontee) sonsDesBanques(m *himodule.Module) map[uint32]bool {
	cible := map[uint32]bool{}
	for _, g := range []string{"snd!", "lsnd"} {
		for _, f := range m.Files(g) {
			data, err := m.Extract(f)
			if err != nil {
				continue
			}
			for _, d := range dependances(data) {
				if d.Groupe == "sbnk" && r.banques[d.IDGlobal] {
					cible[f.GlobalID] = true
					r.groupe[f.GlobalID] = g
					r.parent[f.GlobalID] = d.IDGlobal
					break
				}
			}
		}
	}
	return cible
}

// niveauSuivant rend les tags, tous groupes confondus, dont le corps cite un tag du niveau
// courant. Filtre de 65 536 bits sur les 16 bits bas : sans lui le balayage ne finit pas.
func (r *remontee) niveauSuivant(m *himodule.Module, tous []himodule.File,
	cible, vus map[uint32]bool) map[uint32]bool {
	filtre := new([1 << 16]bool)
	for id := range cible {
		filtre[id&0xFFFF] = true
	}
	suivant := map[uint32]bool{}
	for _, f := range tous {
		if vus[f.GlobalID] {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for o := 0; o+4 <= len(data); o++ {
			v := binary.LittleEndian.Uint32(data[o:])
			if !filtre[v&0xFFFF] || !cible[v] {
				continue
			}
			suivant[f.GlobalID] = true
			r.groupe[f.GlobalID] = f.Group
			if _, deja := r.parent[f.GlobalID]; !deja {
				r.parent[f.GlobalID] = v
			}
			break
		}
	}
	return suivant
}

// afficherNiveau imprime le contenu d'un niveau, groupe par groupe, et detaille les `vehi`.
func (r *remontee) afficherNiveau(niveau map[uint32]bool) {
	compte := map[string]int{}
	var vehis []uint32
	for id := range niveau {
		g := r.groupe[id]
		compte[g]++
		if g == "vehi" {
			vehis = append(vehis, id)
		}
	}
	var groupes []string
	for g := range compte {
		groupes = append(groupes, g)
	}
	sort.Strings(groupes)
	for _, g := range groupes {
		fmt.Printf("    %-6s : %d\n", g, compte[g])
	}
	// Niveaux etroits : on liste les identifiants, c'est ce qui sert a lire la chaine.
	if len(niveau) <= 40 {
		var ids []uint32
		for id := range niveau {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			fmt.Printf("      %s %08x  <- %s\n", r.groupe[id], id, r.chemin(id))
		}
	}
	sort.Slice(vehis, func(i, j int) bool { return vehis[i] < vehis[j] })
	for _, v := range vehis {
		fmt.Printf("    >>> vehi %08x  <- %s\n", v, r.chemin(v))
	}
}

// chemin rend la remontee d'un tag jusqu'a la banque d'origine, sous forme lisible.
func (r *remontee) chemin(depart uint32) string {
	out := ""
	cur := depart
	for i := 0; i < 8; i++ {
		p, ok := r.parent[cur]
		if !ok {
			break
		}
		if out != "" {
			out += " <- "
		}
		out += fmt.Sprintf("%08x", p)
		if r.banques[p] {
			out += " (sbnk)"
			break
		}
		cur = p
	}
	return out
}
