//go:build research

package grammar

// mouvement_5_25_tableaux_research_test.go — LES TABLEAUX (c), (d) ET (e) DU LOT 5.25.
//
// Deplacement pur : la passe et les tableaux (a) et (b) vivent dans
// `mouvement_5_25_ticks_perdus_research_test.go`, sous le seuil de 500 lignes par fichier.
//
// (c) EST LE CHIFFRE QUI COMPTE, et sa methode tient en trois lignes :
//
//	LA VIE      le record de creation de bipede (`ScanBipedCreations`, lot E2) porte le slot, la
//	            generation ET l index de participant que `chunk_00` nomme.
//	LE TICK     une trame delta (~60 par seconde) ; un bipede qui BOUGE y est replique.
//	L ATTENDU   entre DEUX lectures consecutives d une meme vie dont au moins une porte une
//	            vitesse tenue non nulle, toutes les trames intermediaires sont des ticks que le
//	            calque AURAIT du voir. Leur somme est l attendu ; ce qui n y est pas lu est perdu.
//
// L ATTENDU EST ETALONNE, ET C EST CE QUI EMPECHE DE LE SUPPOSER. Sur les paires dont TOUTES les
// trames intermediaires sont FERMEES, la marche ne perd rien par abandon : le taux de trous qui
// y subsiste est le taux NATUREL de non-replication d un bipede en mouvement, et il borne le
// chiffre par le bas.

import (
	"fmt"
	"sort"
	"testing"
)

// t525Vie est UNE vie de corps de joueur : le couple (slot, generation) que la creation designe,
// son proprietaire, et les trames ou son corps a ete lu.
type t525Vie struct {
	slot, gen uint32
	index     int // index de participant absolu, -1 si la vie n a pas de record de creation
	joueur    string
	creation  int
	lectures  []int
	vitesses  []float64
	fenetres  [][2]int // les fenetres de MOUVEMENT, fusionnees
}

// t525Vies rattache les lectures de bipede de la passe aux vies, et les vies aux joueurs.
func t525Vies(t *testing.T, tc t516Temoin, p *t525Passe) []*t525Vie {
	t.Helper()
	creations, st, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations de bipede : %v", err)
	}
	noms := t525Noms(t, tc)
	t.Logf("(c) CREATIONS DE BIPEDE : %d ancres · %d acceptees · %d signatures autres · "+
		"%d joueurs nommes", st.Anchors, st.Accepted, st.SignatureMismatch, len(noms))
	rang := map[[2]int]int{}
	for i, tr := range p.tr {
		rang[[2]int{tr.chunk, tr.paquet}] = i
	}
	vies := t525Decouper(creations, rang, noms)
	t525Rattacher(p, vies)
	for _, v := range vies {
		v.fenetres = t525Fenetres(v)
	}
	return vies
}

// t525Noms lit la table des joueurs de `chunk_00` : index de participant -> gamertag.
func t525Noms(t *testing.T, tc t516Temoin) map[int]string {
	t.Helper()
	chunk0, ok := FilmRegistryChunk(tc.fc.Film())
	if !ok {
		return nil
	}
	ident, err := ReadFilmIdentity(chunk0)
	if err != nil {
		return nil
	}
	slots, _, err := ReadPlayerTable(chunk0, ident)
	if err != nil {
		return nil
	}
	out := map[int]string{}
	for _, s := range slots {
		out[s.FilmIndex] = s.Gamertag
	}
	return out
}

// t525Decouper cree une vie par record de creation, dans l ordre des trames.
func t525Decouper(creations []BipedCreation, rang map[[2]int]int,
	noms map[int]string) []*t525Vie {
	out := make([]*t525Vie, 0, len(creations))
	for _, c := range creations {
		i, ok := rang[[2]int{c.Chunk, c.PacketIndex}]
		if !ok {
			continue
		}
		v := &t525Vie{slot: c.Slot, gen: c.Generation, index: -1, creation: i,
			joueur: "non resolu"}
		if c.HasIndex {
			v.index = int(c.ParticipantIndex)
			if n, connu := noms[v.index]; connu {
				v.joueur = n
			} else {
				v.joueur = fmt.Sprintf("index %d (hors table)", v.index)
			}
		}
		out = append(out, v)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].creation < out[b].creation })
	return out
}

// t525Rattacher verse chaque lecture de bipede dans la vie ouverte du slot a cet instant. Une
// lecture qui precede toute creation du slot n a pas de vie : elle est comptee a part.
func t525Rattacher(p *t525Passe, vies []*t525Vie) {
	parSlot := map[uint32][]*t525Vie{}
	for _, v := range vies {
		parSlot[v.slot] = append(parSlot[v.slot], v)
	}
	curseur := map[uint32]int{}
	for i, tr := range p.tr {
		for k, s := range tr.bipedes {
			liste := parSlot[s]
			j := curseur[s]
			for j+1 < len(liste) && liste[j+1].creation <= i {
				j++
			}
			curseur[s] = j
			if len(liste) == 0 || liste[j].creation > i {
				continue // lecture anterieure a toute creation connue de ce slot
			}
			liste[j].lectures = append(liste[j].lectures, i)
			liste[j].vitesses = append(liste[j].vitesses, tr.vitesses[k])
		}
	}
}

// t525Fenetres rend les fenetres de MOUVEMENT d une vie, fusionnees : entre deux lectures
// consecutives dont au moins une porte une vitesse tenue au-dessus du seuil, toutes les trames
// sont des ticks attendus.
func t525Fenetres(v *t525Vie) [][2]int {
	var out [][2]int
	for i := 0; i+1 < len(v.lectures); i++ {
		if v.vitesses[i] <= t525SeuilVitesse && v.vitesses[i+1] <= t525SeuilVitesse {
			continue
		}
		a, b := v.lectures[i], v.lectures[i+1]
		if n := len(out); n > 0 && out[n-1][1] >= a {
			if out[n-1][1] < b {
				out[n-1][1] = b
			}
			continue
		}
		out = append(out, [2]int{a, b})
	}
	return out
}

// ---------------------------------------------------------------------------------------------
// (c) LES TICKS DE BIPEDE DE JOUEUR : ATTENDUS, LUS, PERDUS
// ---------------------------------------------------------------------------------------------

// t525Compte est le bilan de ticks d une vie ou d un joueur.
type t525Compte struct {
	vies, attendu, lus, perdus int
	// fermeSpan / fermeTrous : l ETALON — les memes trous, comptes sur les seules paires dont
	// toutes les trames intermediaires sont FERMEES. Le taux qui y subsiste est le taux NATUREL.
	fermeSpan, fermeTrous int
}

// t525TableauC publie le chiffre du lot : par joueur, puis au total.
func t525TableauC(t *testing.T, p *t525Passe, vies []*t525Vie) {
	t.Helper()
	parJoueur := map[string]*t525Compte{}
	tot := &t525Compte{}
	for _, v := range vies {
		c := parJoueur[v.joueur]
		if c == nil {
			c = &t525Compte{}
			parJoueur[v.joueur] = c
		}
		t525Compter(p, v, c)
		t525Compter(p, v, tot)
	}
	noms := make([]string, 0, len(parJoueur))
	for n := range parJoueur {
		noms = append(noms, n)
	}
	sort.Slice(noms, func(a, b int) bool {
		return parJoueur[noms[a]].perdus > parJoueur[noms[b]].perdus
	})
	t.Logf("(c) TICKS DE BIPEDE DE JOUEUR — attendus / lus / PERDUS, par joueur")
	t.Logf("    %-22s %5s %9s %9s %9s %7s", "joueur", "vies", "attendus", "lus", "PERDUS", "perte")
	for _, n := range noms {
		c := parJoueur[n]
		t.Logf("    %-22s %5d %9d %9d %9d %6.1f %%", n, c.vies, c.attendu, c.lus, c.perdus,
			m533bPart(c.perdus, c.attendu))
	}
	t.Logf("    %-22s %5d %9d %9d %9d %6.1f %%", "TOTAL", tot.vies, tot.attendu, tot.lus,
		tot.perdus, m533bPart(tot.perdus, tot.attendu))
	t.Logf("    ETALON (paires dont TOUTES les trames intermediaires sont FERMEES) : "+
		"%d trames de fenetre, %d trous -> taux NATUREL de non-replication %.2f %%",
		tot.fermeSpan, tot.fermeTrous, m533bPart(tot.fermeTrous, tot.fermeSpan))
	nat := m533bPart(tot.fermeTrous, tot.fermeSpan) / 100
	t.Logf("    PERTE NETTE DE L ABANDON (perte mesuree moins le taux naturel) : %.1f %% "+
		"soit %d ticks", m533bPart(tot.perdus, tot.attendu)-nat*100,
		tot.perdus-int(nat*float64(tot.attendu)))
}

// t525Compter cumule les ticks d une vie dans un bilan.
func t525Compter(p *t525Passe, v *t525Vie, c *t525Compte) {
	c.vies++
	dans := func(i int) bool {
		for _, f := range v.fenetres {
			if i >= f[0] && i <= f[1] {
				return true
			}
		}
		return false
	}
	for _, f := range v.fenetres {
		c.attendu += f[1] - f[0] + 1
	}
	for _, i := range v.lectures {
		if dans(i) {
			c.lus++
		}
	}
	c.perdus = c.attendu - c.lus
	t525Etalon(p, v, c)
}

// t525Etalon cumule l etalon : les trous des paires ENTIEREMENT fermees.
func t525Etalon(p *t525Passe, v *t525Vie, c *t525Compte) {
	for i := 0; i+1 < len(v.lectures); i++ {
		if v.vitesses[i] <= t525SeuilVitesse && v.vitesses[i+1] <= t525SeuilVitesse {
			continue
		}
		a, b := v.lectures[i], v.lectures[i+1]
		propre := true
		for j := a; j <= b && propre; j++ {
			propre = p.tr[j].classe == t525Fermee
		}
		if !propre {
			continue
		}
		c.fermeSpan += b - a
		c.fermeTrous += b - a - 1
	}
}

// t525Zones compte les zones ABANDONNEES contigues de plus de [t525ZoneMin] trames a l interieur
// d une fenetre de mouvement, sans une seule lecture : un intervalle d etat entier peut y tenir,
// et il serait perdu sans laisser de bord.
func t525Zones(p *t525Passe, v *t525Vie) int {
	lu := map[int]bool{}
	for _, i := range v.lectures {
		lu[i] = true
	}
	n, courant := 0, 0
	for _, f := range v.fenetres {
		courant = 0
		for i := f[0]; i <= f[1]; i++ {
			if lu[i] || p.tr[i].classe == t525Fermee {
				if courant > t525ZoneMin {
					n++
				}
				courant = 0
				continue
			}
			courant++
		}
		if courant > t525ZoneMin {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------------------------
// (d) LA BORNE DES TRANSITIONS D ETAT QUE LE CALQUE PEUT MANQUER
// ---------------------------------------------------------------------------------------------

// t525TableauD publie la borne haute des transitions manquees et la comparaison aux intervalles.
func t525TableauD(t *testing.T, p *t525Passe, vies []*t525Vie) {
	t.Helper()
	etatsParSlot := map[uint32][]int{}
	for _, e := range p.etats {
		etatsParSlot[e.slot] = append(etatsParSlot[e.slot], e.trame)
	}
	for _, l := range etatsParSlot {
		sort.Ints(l)
	}
	paires, trames := 0, map[int]bool{}
	for _, v := range vies {
		lu := map[int]bool{}
		for _, i := range v.lectures {
			lu[i] = true
		}
		for _, f := range v.fenetres {
			for i := f[0]; i <= f[1]; i++ {
				if lu[i] || p.tr[i].classe != t525Abandonnee {
					continue
				}
				if !t525EtatVoisin(etatsParSlot[v.slot], i) {
					continue
				}
				paires++
				trames[i] = true
			}
		}
	}
	t.Logf("(d) BORNE DES TRANSITIONS MANQUEES : %d couples (trame, vie) — un corps de joueur "+
		"en mouvement, NON LU dans une trame abandonnee, alors qu un changement d etat est lu "+
		"pour lui a +-%d trames · %d trames distinctes", paires, t525FenetreEtat, len(trames))
	t525Intervalles(t, p, vies)
}

// t525EtatVoisin dit si une lecture d etat de ce corps tombe dans la fenetre de la trame.
func t525EtatVoisin(trames []int, i int) bool {
	j := sort.SearchInts(trames, i-t525FenetreEtat)
	return j < len(trames) && trames[j] <= i+t525FenetreEtat
}

// t525Intervalles replie les lectures d etat en intervalles et dit combien ont un BORD dans une
// trame abandonnee (un bord peut etre DECALE, pas perdu) ; puis, par joueur, combien de zones
// abandonnees assez longues pour avaler un intervalle ENTIER.
func t525Intervalles(t *testing.T, p *t525Passe, vies []*t525Vie) {
	t.Helper()
	slotJoueur := map[uint32]string{}
	for _, v := range vies {
		slotJoueur[v.slot] = v.joueur
	}
	ouvert := map[[2]string]int{}
	parJoueur, bords := map[string]int{}, map[string]int{}
	total, bordAband := 0, 0
	for _, e := range p.etats {
		j := slotJoueur[e.slot]
		if j == "" {
			j = "non resolu"
		}
		cle := [2]string{fmt.Sprint(e.slot), e.genre}
		debut, ouvre := ouvert[cle]
		switch {
		case e.actif && !ouvre:
			ouvert[cle] = e.trame
		case !e.actif && ouvre:
			delete(ouvert, cle)
			total++
			parJoueur[j]++
			if p.tr[debut].classe == t525Abandonnee || p.tr[e.trame].classe == t525Abandonnee {
				bordAband++
				bords[j]++
			}
		}
	}
	t.Logf("    INTERVALLES D ETAT LUS (accroupi, glissade, escalade, sprint) : %d fermes · "+
		"%d (%.1f %%) ont au moins un bord dans une trame ABANDONNEE",
		total, bordAband, m533bPart(bordAband, total))
	t525ZonesParJoueur(t, p, vies, parJoueur, bords)
}

// t525ZonesParJoueur publie, par joueur, les intervalles lus, leurs bords en zone abandonnee et
// le nombre de zones abandonnees assez longues pour qu un intervalle entier y disparaisse.
func t525ZonesParJoueur(t *testing.T, p *t525Passe, vies []*t525Vie,
	parJoueur, bords map[string]int) {
	t.Helper()
	zones := map[string]int{}
	for _, v := range vies {
		zones[v.joueur] += t525Zones(p, v)
	}
	noms := make([]string, 0, len(zones))
	for n := range zones {
		noms = append(noms, n)
	}
	sort.Slice(noms, func(a, b int) bool { return zones[noms[a]] > zones[noms[b]] })
	t.Logf("    %-22s %11s %9s %14s", "joueur", "intervalles", "bords", fmt.Sprintf(
		"zones > %d trames", t525ZoneMin))
	for _, n := range noms {
		t.Logf("    %-22s %11d %9d %14d", n, parJoueur[n], bords[n], zones[n])
	}
}

// ---------------------------------------------------------------------------------------------
// (e) CE QUE SONT LES ENTITES REJETEES
// ---------------------------------------------------------------------------------------------

// t525Rejet cumule tout ce qu on sait d un eid rejete, sans lire un bit de plus.
type t525Rejet struct {
	n                int
	premier, dernier int
	tete             int
}

// t525TableauE publie ce que sont les entites que la table anticipee ne resout pas.
func t525TableauE(t *testing.T, p *t525Passe) {
	t.Helper()
	parID := map[uint32]*t525Rejet{}
	total := 0
	for i, tr := range p.tr {
		if !tr.rejet {
			continue
		}
		total++
		r := parID[tr.rejetID]
		if r == nil {
			r = &t525Rejet{premier: i, tete: tr.tete}
			parID[tr.rejetID] = r
		}
		r.n++
		r.dernier = i
	}
	t.Logf("(e) REJETS APRES LA TABLE ANTICIPEE : %d en-tetes rejetes · %d eid distincts",
		total, len(parID))
	t525ClasserRejets(t, p, parID)
	t525DureesRejets(t, p, parID)
	t525CroiserTetes(t, parID)
}

// t525ClasserRejets ventile les eid rejetes par tete et par ce que la table du film en dit.
func t525ClasserRejets(t *testing.T, p *t525Passe, parID map[uint32]*t525Rejet) {
	t.Helper()
	parTete, jamais, ailleurs := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	for id, r := range parID {
		tete := id >> 30
		parTete[tete] += r.n
		if _, _, ok := p.tab.ArchetypeApres(id, -1); ok {
			ailleurs[tete] += r.n
			continue
		}
		jamais[tete] += r.n
	}
	for tete := uint32(0); tete < 4; tete++ {
		if parTete[tete] == 0 {
			continue
		}
		t.Logf("      tete %d : %6d rejets · declaree par une image-cle du film %6d · "+
			"JAMAIS declaree %6d", tete, parTete[tete], ailleurs[tete], jamais[tete])
	}
}

// t525DureesRejets publie la duree de vie apparente d un eid rejete (premier -> dernier rejet)
// et le nombre de rejets par eid : une duree courte est un projectile, une grenade, un effet.
func t525DureesRejets(t *testing.T, p *t525Passe, parID map[uint32]*t525Rejet) {
	t.Helper()
	duree, compte := map[string]int{}, map[string]int{}
	for _, r := range parID {
		ms := float64(p.tr[r.dernier].ts-p.tr[r.premier].ts) / 1000
		duree[t525Seau(ms, []float64{0, 100, 500, 1000, 5000, 30000})]++
		compte[t525Seau(float64(r.n), []float64{1, 2, 5, 20, 100, 1000})]++
	}
	t.Logf("    DUREE entre le PREMIER et le DERNIER rejet d un meme eid :")
	for _, s := range t525Ordre(duree) {
		t.Logf("      %-14s %5d eid (%.1f %%)", s, duree[s], m533bPart(duree[s], len(parID)))
	}
	t.Logf("    NOMBRE DE REJETS par eid :")
	for _, s := range t525Ordre(compte) {
		t.Logf("      %-14s %5d eid (%.1f %%)", s, compte[s], m533bPart(compte[s], len(parID)))
	}
}

// t525CroiserTetes croise « premier rejet d un eid » et « evenement de tete du meme paquet ».
func t525CroiserTetes(t *testing.T, parID map[uint32]*t525Rejet) {
	t.Helper()
	croix := map[int]int{}
	for _, r := range parID {
		croix[r.tete]++
	}
	cles := make([]int, 0, len(croix))
	for k := range croix {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(a, b int) bool { return croix[cles[a]] > croix[cles[b]] })
	t.Logf("    EVENEMENT DE TETE du paquet du PREMIER rejet de chaque eid (-1 = aucun) :")
	for i, k := range cles {
		if i >= 12 {
			break
		}
		t.Logf("      type %3d : %5d eid (%.1f %%)", k, croix[k], m533bPart(croix[k], len(parID)))
	}
}

// t525Seau nomme le seau d une valeur dans une echelle croissante. Le nom PORTE le rang du seau
// (`3|...`) pour qu un tri alphabetique soit le tri des seaux : un histogramme dont l ordre
// depend du hasard d une map ne se relit pas.
func t525Seau(v float64, bornes []float64) string {
	for i := len(bornes) - 1; i >= 0; i-- {
		if v < bornes[i] {
			continue
		}
		if i == len(bornes)-1 {
			return fmt.Sprintf("%d|>= %g", i, bornes[i])
		}
		return fmt.Sprintf("%d|%g a %g", i, bornes[i], bornes[i+1])
	}
	return fmt.Sprintf("0|< %g", bornes[0])
}
