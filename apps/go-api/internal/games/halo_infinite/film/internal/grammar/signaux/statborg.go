package signaux

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// statborg.go — les ENREGISTREMENTS D'ENTITE du statborg, lus dans les paquets delta : le score de
// mode, le score personnel et les compteurs de combat, horodates a la ms. La marche ne les lit
// pas : ils sont LOCALISES par les contraintes mesurees de leur en-tete, a chaque position de bit
// du paquet, et ses deux replis se comptent dans [LectureDuStatborg].
//
// # Ce qu'est un enregistrement
//
// Chaque entite du match (2 equipes + 8 joueurs) est serialisee dans les paquets FRAME par
// un enregistrement portant une liste CREUSE de composants : l'enregistrement DIT quels
// composants il transporte, on n'a donc pas a parcourir les 58 de l'archetype. Un composant
// n'est reemis QUE lorsqu'il change — c'est ce qui rend la resolution temporelle bien plus
// fine que le tick de 5 s du footer.
//
// # La grammaire, MESUREE (jamais derivee du binaire seul)
//
//	[1 bit = 1 : record DELTA][13 bits identifiant][2 bits generation][1 bit selecteur d'etat
//	de base ; si 1 : 7 bits][liste de composants]
//
//	liste, gate = 0 (creuse) : [1 bit = 0][3 bits N][N x 6 bits index]
//	liste, gate = 1 (DENSE)  : [1 bit = 1][64 bits masque]
//
//	puis, par index : [5 bits MANCHE][5 bits MANCHE][valeur A][valeur B][2 drapeaux][conditionnelles]
//
// Chaque constante a ete lue sur 1 078 en-tetes et 2 708 lectures de composant issus d'une
// capture Cheat Engine, pas supposee (.ai/V7.5/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §15) :
//   - le bit qui precede l'identifiant vaut 1 dans 1 077/1 078 (c'est le code de record DELTA) ;
//   - les slots valent 6 et 8 (equipes) et 10..24 pairs (les 8 joueurs), soit
//     2 x (identifiant runtime - 0x40000000) ;
//   - la generation vaut 1 et le selecteur d'etat de base vaut 0 dans 1 077/1 078.
//
// # Ce que la description precedente disait de FAUX, et ce que cela coutait (2026-08-18)
//
// Elle decrivait « 14 bits de slot » puis « 2 bits constants a 10 » : ce sont en realite
// 13 bits d'identifiant, 2 bits de GENERATION et 1 bit de SELECTEUR D'ETAT DE BASE. La
// contrainte cachait donc deux hypotheses (generation = 1, selecteur = 0) — d'ou le slot
// toujours pair, qui vaut 2 x identifiant.
//
// Elle exigeait surtout les deux en-tetes de 5 bits du composant NULS. Or ces en-tetes portent
// le NUMERO DE MANCHE (0 = premiere manche), comme le getter natif le dit deja :
//
//	value = *(int32*)(world + slot*0x88 + equipe*0x1DF0 + 0x38 + manche*4)
//
// L'assertion « == 0 » etait donc un FILTRE DE MANCHE : tout match a plusieurs manches n'etait
// lu que jusqu'a la fin de la premiere. Mesure du 2026-08-18 (phase 0-ter du lot A) : en Oddball,
// le score d'equipe passe de 100/78 (une manche) a 200/121 (somme des manches) = l'oracle, sur
// 4 films sur 4 ; les frags passent de 48/88 a 385/391 (98,5 %). Un match peut avoir TROIS
// manches (`c88ec007`) : ne jamais cabler un nombre de manches.
//
// La forme DENSE de liste (gate = 1, masque de 64 bits) etait ignoree de meme : 33 records
// perdus sur `24dbb67d`. Le moteur y bascule quand un enregistrement change plus de sept
// composants — typiquement a la fin d'une manche.
//
// Controle de non-regression de ce relachement : sur 9 films sans manches multiples, les
// en-tetes non nuls representent 22 emissions sur 869 (2,5 %), toujours isolees et jamais
// groupees ; `530820e5` n'en a aucune.
//
// Piege consigne : les 2 bits qui PRECEDENT le bit de presence ne sont pas un champ de
// type. Ils sont statistiquement independants (22 co-occurrences pour 21,6 attendues sous
// independance) : c'est la queue de l'enregistrement precedent. Ne pas les contraindre.
//
// # L'horloge
//
// L'horodatage vient du `us` du paquet, recale par chunk sur le `start_ms` du manifeste
// ([LireLeStatborg]). Les deux concordent a moins de 4 ms sur 573 s de film, et le pied du film
// ([FooterEvents]) est sur la meme base : tout est superposable sans recalage. Prendre pour
// origine le premier paquet OU L'ON TROUVE QUELQUE CHOSE au lieu du manifeste decale toute
// la courbe (140 s mesures sur un CTF).

// Les bornes de format que la couche des faits lit : la contiguite des manches, la bande des slots
// de joueur, le plafond d'une lecture.
const (
	// StatborgMancheMax borne le numero de manche accepte. Huit manches est au-dela de tout format
	// observe (le maximum mesure est 3, sur `c88ec007`) et la borne conserve 2 bits de
	// contrainte sur chacun des deux en-tetes, soit 4 des 10 bits de filtre anti-faux-positifs
	// de la forme d'origine. Sans borne, l'ancrage laisserait passer 151 faux positifs par film.
	StatborgMancheMax = 7
	// StatborgSlotEquipeMax / StatborgSlotMax delimitent les slots d'entite avec [statSlotMin] :
	// 6 et 8 pour les deux equipes, 10 a 24 (pairs) pour les huit joueurs.
	StatborgSlotEquipeMax = 8
	StatborgSlotMax       = 24
	// StatborgEnregistrementsMax plafonne ce qu'un film peut rendre.
	//
	// POURQUOI CE PLAFOND EXISTE : ce balayage a rendu la machine de l'utilisateur inutilisable
	// deux fois en aout 2026, et un film du corpus (`1b1e380f`, Strongholds) a atteint 3,3 Go
	// avant d'etre tue par une surveillance externe le 2026-08-18. Un decodeur qui vit dans un
	// service ne peut pas dependre d'une surveillance externe.
	//
	// La valeur est QUATRE FOIS le maximum mesure sur le corpus de 22 films de la phase 0
	// (8 269 enregistrements sur `c88ec007`, un Oddball a trois manches — le film le plus
	// bavard). Un film qui la depasse est anormal : on rend ce qui a ete lu et on marque le
	// resultat TRONQUE plutot que de gonfler jusqu'au plafond memoire.
	StatborgEnregistrementsMax = 4 * 8269
)

const (
	// statIDBits est la largeur du champ de slot d'entite dans l'en-tete d'enregistrement.
	statIDBits = 14
	// statGenBits / statGenValue : les 2 bits qui suivent le slot, constants. Ils valent en
	// realite le second bit de la GENERATION et le SELECTEUR D'ETAT DE BASE (cf. en-tete) ;
	// la forme conservee ici est celle qui a ete calibree, et elle equivaut a exiger
	// generation = 1 et selecteur = 0.
	statGenBits  = 2
	statGenValue = 0b10
	// statHdrBits est la largeur de chacun des deux en-tetes d'un composant. Ils portent le
	// NUMERO DE MANCHE (0-based).
	statHdrBits = 5
	// statDenseMaskBits est la largeur du masque de la forme dense (gate = 1).
	statDenseMaskBits = 64
	// statMaxComp borne les index de composant de l'archetype (58 composants).
	statMaxComp = 58
	// statCompIndexBits est la largeur d'un index de composant dans la liste creuse.
	statCompIndexBits = 6
	// statMaxCompPerRecord : un enregistrement porte de 1 a 7 composants (compte sur
	// 3 bits ; mediane mesuree 2, jamais les 58 de l'archetype).
	statMaxCompPerRecord = 7
	// statSlotMin est le premier slot d'entite (cf. [StatborgSlotMax]).
	statSlotMin = 6
	// statTailBits : marge de fin de paquet sous laquelle on n'ancre plus.
	statTailBits = 64
	// statMaxCounter borne les CANAUX A et B d un enregistrement d entite. Ce n est pas un
	// reglage mais une contrainte de DOMAINE, au niveau de l ENREGISTREMENT et non du pas —
	// c est le filtre que le lot 6.7-B1 avait nomme sans le poser (decouverte D5) : rejeter le
	// record entier quand l un de ses canaux est hors domaine, comme le filtre du score de mode
	// (`objectives.modeScoreInDomain`) le fait deja pour ce score.
	//
	// LA MESURE QUI LE CALE (lot 6.11, item 3, sur 11 films du parc, tous modes) : la plus
	// grande valeur d un enregistrement dont AUCUN canal n atteint 2^20 est 102 934
	// (`4f77afc1`) et 32 518 partout ailleurs ; la population aberrante, elle, commence a
	// 2 415 919 104 (0x90000000) et monte a 4,1 milliards. Entre 102 934 et 1 048 576 il n y a
	// RIEN — un facteur dix de vide. Le seuil est donc dix fois au-dessus de la pire valeur
	// saine mesuree et exactement sous la plus petite valeur aberrante.
	//
	// CE QUE CE FILTRE FERME, ET IL A COUTE UNE PUBLICATION FAUSSE : sur `fb1a1a72`, un
	// enregistrement fortuit du slot 24 (t = 764 967, DOUZE composants, canaux a 2 415 919 104
	// et -30 456) portait `comp 22 A = 10`, publie en DIX prises de drapeau pour ZERO a
	// l oracle. Le pas de 10 passait sous la borne de deroulage (16) : la borne par pas ne
	// pouvait pas le voir, seul le niveau de l enregistrement le voit.
	//
	// A ET B SEULEMENT : ce sont les deux canaux mesures, et les seuls que les tables
	// d objectif lisent. C et D restent hors de la contrainte tant que personne ne les a
	// mesures — on ne borne pas ce qu on n a pas observe.
	statMaxCounter = 1 << 20
)

// scanFrameForRecords balaie un paquet FRAME et rend les enregistrements qu'il porte.
func scanFrameForRecords(pay []byte, tMS int) []types.StatRecord {
	return scanFrameAvecReplis(pay, tMS, nil)
}

// scanFrameAvecReplis est [scanFrameForRecords], qui compte ses deux replis dans `c` (nil : rien).
func scanFrameAvecReplis(pay []byte, tMS int, c *LectureDuStatborg) []types.StatRecord {
	var out []types.StatRecord
	var abandonnes, arretes int
	lim := len(pay)*8 - statTailBits
	for b := 1; b < lim; b++ {
		slot, idx, at, ok := matchRecordHeader(pay, b)
		if !ok {
			continue
		}
		comps, round, arrete := decodeComponentsAvecArret(pay, at, idx)
		if len(comps) == 0 || !statCountersInDomain(comps) {
			abandonnes++
			continue
		}
		if arrete {
			arretes++
		}
		out = append(out, types.StatRecord{TimeMS: tMS, Slot: slot, Round: round, Comps: comps})
	}
	if c != nil {
		c.EnregistrementsAbandonnes += abandonnes
		c.ComposantsArretes += arretes
	}
	return out
}

// statCountersInDomain dit que TOUS les canaux A et B d'un enregistrement tiennent dans le
// domaine des compteurs. Un seul canal hors domaine condamne l'enregistrement ENTIER : un
// ancrage fortuit ne produit pas une valeur fausse, il produit un record qui n'existe pas, et
// ses autres composants sont du bruit au meme titre (cf. [statMaxCounter]).
func statCountersInDomain(comps map[int]types.StatValue) bool {
	for _, v := range comps {
		if v.A > statMaxCounter || v.A < -statMaxCounter ||
			v.B > statMaxCounter || v.B < -statMaxCounter {
			return false
		}
	}
	return true
}

// matchRecordHeader teste l'en-tete d'enregistrement d'entite a la position b. Il rend le
// slot, la liste creuse d'index de composants et la position du premier composant.
func matchRecordHeader(pay []byte, b int) (slot int, idx []int, compAt int, ok bool) {
	if source.BitsTronques(pay, b-1, 1) != 1 {
		return 0, nil, 0, false
	}
	slot = int(source.BitsTronques(pay, b, statIDBits))
	if slot < statSlotMin || slot > StatborgSlotMax || slot%2 != 0 {
		return 0, nil, 0, false
	}
	if source.BitsTronques(pay, b+statIDBits, statGenBits) != statGenValue {
		return 0, nil, 0, false
	}
	m := b + statIDBits + statGenBits
	// Le moteur a DEUX formes de liste de composants (FUN_1406d7610) : creuse quand un
	// enregistrement change au plus sept composants, DENSE au-dela — typiquement a la fin
	// d'une manche, quand les 28 compteurs se figent et que les 28 suivants repartent.
	if source.BitsTronques(pay, m, 1) != 0 {
		idx, ok = denseComponentList(pay, m+1)
		return slot, idx, m + 1 + statDenseMaskBits, ok
	}
	n := int(source.BitsTronques(pay, m+1, 3))
	if n < 1 || n > statMaxCompPerRecord {
		return 0, nil, 0, false
	}
	// La liste creuse doit etre strictement croissante et bornee : c'est elle qui porte
	// l'essentiel de la contrainte dure.
	idx = make([]int, n)
	prev := -1
	for i := range n {
		idx[i] = int(source.BitsTronques(pay, m+4+statCompIndexBits*i, statCompIndexBits))
		if idx[i] >= statMaxComp || idx[i] <= prev {
			return 0, nil, 0, false
		}
		prev = idx[i]
	}
	return slot, idx, m + 4 + statCompIndexBits*n, true
}

// denseComponentList lit la forme DENSE : un masque de 64 bits dont le bit i designe le
// composant i. L'archetype n'ayant que [statMaxComp] composants, les bits de poids fort
// doivent etre nuls — c'est ce qui remplace ici la contrainte de la liste croissante.
func denseComponentList(pay []byte, p int) ([]int, bool) {
	if p+statDenseMaskBits > len(pay)*8 {
		return nil, false
	}
	mask := source.BitsTronques(pay, p, statDenseMaskBits)
	if mask == 0 || mask>>statMaxComp != 0 {
		return nil, false
	}
	idx := make([]int, 0, statMaxComp)
	for i := range statMaxComp {
		if mask>>uint(i)&1 == 1 {
			idx = append(idx, i)
		}
	}
	return idx, true
}

// decodeComponents lit les composants listes, dans l'ordre, et rend la MANCHE que porte
// l'enregistrement.
//
// Le PREMIER composant porte la contrainte qui separe un vrai enregistrement d'un ancrage
// fortuit : ses deux en-tetes de 5 bits doivent designer la MEME manche, bornee par
// [StatborgMancheMax]. Exiger la manche ZERO — ce que faisait la version d'avant le 2026-08-18 —
// revenait a jeter tout ce qui suit la premiere manche. Exiger l'egalite des deux en-tetes est
// ce qui recupere la contrainte perdue : sur les emissions reelles ils portent toujours la meme
// valeur, un couple depareille est un faux positif.
//
// Les composants suivants ne sont pas re-contraints — leurs largeurs sont chainees, une lecture
// qui derape s'arrete d'elle-meme.
func decodeComponents(pay []byte, at int, idx []int) (map[int]types.StatValue, int) {
	out, round, _ := decodeComponentsAvecArret(pay, at, idx)
	return out, round
}

// decodeComponentsAvecArret est [decodeComponents], plus `arrete` : un composant non decodable a
// ARRETE la boucle avant le dernier annonce — le verdict de `repli_composants_statborg_arretes`.
func decodeComponentsAvecArret(pay []byte, at int, idx []int) (map[int]types.StatValue, int, bool) {
	h1 := int(source.BitsTronques(pay, at, statHdrBits))
	h2 := int(source.BitsTronques(pay, at+statHdrBits, statHdrBits))
	if h1 != h2 || h1 > StatborgMancheMax {
		return nil, 0, false
	}
	out := make(map[int]types.StatValue, len(idx))
	q := at
	for _, i := range idx {
		v, w, ok := decodeStatComponent(pay, q)
		if !ok {
			return out, h1, true
		}
		out[i] = v
		q += w
	}
	return out, h1, false
}

// decodeStatComponent lit un composant et rend sa largeur consommee. Reproduit
// FUN_140C18794 : deux en-tetes de 5 bits, deux valeurs a longueur variable, deux drapeaux
// commandant chacun une valeur conditionnelle.
func decodeStatComponent(pay []byte, p int) (types.StatValue, int, bool) {
	q := p + 2*statHdrBits
	a, n1, ok := readStatVarWidth(pay, q)
	if !ok {
		return types.StatValue{}, 0, false
	}
	b, n2, ok := readStatVarWidth(pay, q+n1)
	if !ok {
		return types.StatValue{}, 0, false
	}
	q += n1 + n2
	if q+2 > len(pay)*8 {
		return types.StatValue{}, 0, false
	}
	flags := [2]uint64{source.BitsTronques(pay, q, 1), source.BitsTronques(pay, q+1, 1)}
	q += 2
	// LES DEUX CANAUX CONDITIONNELS SONT DESORMAIS GARDES (2026-08-31). Ils etaient lus pour
	// avancer le curseur, puis JETES — 56 emplacements que rien n'avait jamais regardes (cf.
	// l'en-tete de [types.StatValue]). Rien d'autre ne change : le curseur avance de la meme facon,
	// et aucun lecteur existant ne consulte C ou D.
	out := types.StatValue{A: a, B: b}
	for i, f := range flags {
		if f != 1 {
			continue
		}
		v, n, ok := readStatVarWidth(pay, q)
		if !ok {
			return types.StatValue{}, 0, false
		}
		if i == 0 {
			out.C, out.HasC = v, true
		} else {
			out.D, out.HasD = v, true
		}
		q += n
	}
	return out, q - p, true
}

// readStatVarWidth reproduit le lecteur a longueur variable FUN_140C18A1C : un selecteur
// de 2 bits donne la largeur (8 << selecteur), la valeur est signee.
func readStatVarWidth(pay []byte, p int) (int64, int, bool) {
	if p < 0 || p+2 > len(pay)*8 {
		return 0, 0, false
	}
	w := 8 << uint(source.BitsTronques(pay, p, 2))
	if w > 32 || p+2+w > len(pay)*8 {
		return 0, 0, false
	}
	v := source.BitsTronques(pay, p+2, w)
	iv := int64(v)
	if w < 32 && v&(1<<uint(w-1)) != 0 {
		iv = int64(v) - (1 << uint(w))
	}
	return iv, 2 + w, true
}
