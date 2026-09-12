package replay

// flag_secures_sweep_test.go — OU EST `flag_secures` DANS LE STATBORG ? Le balayage, et son
// gate STRICT.
//
// # LE FAIT (audit du 2026-09-10, cause C5)
//
// L'API publie `flag_secures` par joueur. Le rejeu n'en publie AUCUNE : l'emplacement n'est dans
// aucune table (`objectiveevents/named.go`, famille `ObjectiveTypeFlag`). 264 actions a l'oracle
// sur les 13 films CTF d'arene du parc, 0 publiee, jamais nommee.
//
// # LE PROTOCOLE EST CELUI QUI A NOMME `flag_grabs`, ET IL N'EST PAS NEGOCIE ICI
//
// `named.go:139-155` le decrit : on balaye les composants, on demande a chacun la valeur d'un
// joueur NOMME (pont slot -> xuid par le triplet frags/morts/assistances) et on la compare a
// l'oracle. Le discriminant est l'EXACTITUDE PAR JOUEUR, jamais la coincidence de somme — c'est
// ce qui a demasque `comp 22 A`, nomme `flag_taken` par sa somme et qui valait `flag_grabs`.
//
// # LE GATE EST STRICT, ET IL EST ECRIT AVANT LA MESURE
//
// Un emplacement n'est nommable que s'il reproduit `flag_secures` joueur par joueur sur **au
// moins 10 films** et **sans AUCUN desaccord**. En dessous, ON NE PUBLIE RIEN : une statistique
// fausse a l'ecran coute plus qu'une statistique absente, et la table de `named.go` porte deja
// la trace des cles « nommees par coincidence de comptes » qui sont tombees au controle.
//
// # LE GATE EST PASSE LE 2026-09-11, ET CE FICHIER EST DEVENU LA GARDE
//
// `comp 23 B` est le SEUL emplacement exact du balayage, et il l'est sur **11 films sur 11
// exploitables, 79 slots apparies, 0 desaccord**, pour 169 securisations non nulles. Il est
// donc entre dans `namedStatSlots` (famille drapeau) le meme jour. Ce fichier n'est plus
// seulement l'instrument de la decision : il ECHOUE desormais si `comp 23 B` cesse de
// reproduire `flag_secures` sur un film du corpus.
//
// LES DEUX FILMS ECARTES ne temoignent NI POUR NI CONTRE, et il faut le dire : `7fce3219` ne
// rend AUCUN pont d'identite (CTF multi-manche), et `fb1a1a72` fait tomber le TEMOIN POSITIF —
// `comp 22 A` y publie 10 prises de drapeau pour 0 a l'oracle sur un joueur.
//
// REGIME : garde `ZONE_FILM` (repertoire de chunks), UN FILM PAR PROCESSUS, lecture seule,
// AUCUNE base ouverte.
//
//	$env:ZONE_FILM="<cache>/film_chunks/64e8adfa"
//	go test ./internal/games/halo_infinite/film/replay/ -run FlagSecuresSweep -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// fsJoueur est la ligne d'oracle d'un joueur : ce que l'API dit de ses securisations, le
// TEMOIN POSITIF (`flag_grabs`, deja nomme `comp 22 A`) et le triplet du pont d'identite.
type fsJoueur struct {
	xuid                   string
	secures, grabs         int
	kills, deaths, assists int
}

// fsOracle — RELEVE DERIVE DES EXPORTS COMMITES de la vague 6
// (`oracle_vague6_objective_stats.tsv` joint a `oracle_vague6_participants.tsv`), gele ici : le
// paquet `replay` n'ouvre aucune DuckDB, meme convention que `e1bOracle` et `p2aCorpus`.
//
// LES 13 FILMS SONT LES CTF D'ARENE DU PARC QUI ONT DES CHUNKS. Les trois CTF de BTB
// (`4f77afc1` 36 lignes, `879a4dba` 26, `1c4c63c2` 24) sont EXCLUS et ce n'est pas un choix de
// confort : au-dela de huit sieges le statborg n'a plus de place pour dire de qui il parle
// (`objectiveevents/rosterfit.go`), et leurs series ne correspondent a personne de facon
// reproductible. Les y inclure fabriquerait des desaccords qui ne diraient rien du composant.
var fsOracle = map[string][]fsJoueur{
	// 16ea3668 — CTF:Arena sur Aquarius : 7 securisation(s) a l oracle.
	"16ea3668": {
		{"2533274823110022", 0, 0, 12, 11, 3},
		{"2533274858283686", 2, 6, 8, 9, 2},
		{"2533275034585819", 1, 3, 11, 12, 7},
		{"2535417044536883", 0, 0, 13, 5, 1},
		{"2535447614023932", 1, 6, 10, 8, 6},
		{"2535466949313010", 2, 2, 11, 11, 2},
		{"2535469190789936", 0, 0, 5, 11, 6},
		{"2598952284978039", 1, 1, 4, 7, 4},
	},
	// 4ecdf3e7 — CTF:Arena Neutral Flag sur High Ground : 11 securisation(s) a l oracle.
	"4ecdf3e7": {
		{"2533274823110022", 1, 1, 1, 4, 1},
		{"2533274858283686", 6, 9, 7, 3, 3},
		{"2533274877168586", 1, 1, 1, 2, 2},
		{"2533274913257456", 0, 0, 1, 3, 0},
		{"2533274924091420", 0, 0, 2, 6, 3},
		{"2535409159085195", 0, 0, 6, 6, 2},
		{"2535426953464929", 2, 10, 7, 2, 1},
		{"2535434009919524", 0, 0, 3, 3, 1},
		{"2535469190789936", 1, 1, 5, 4, 3},
		{"bid(29.0)", 0, 0, 0, 0, 0},
	},
	// 58864b3c — CTF:Arena sur Domicile : 13 securisation(s) a l oracle.
	"58864b3c": {
		{"2533274822170595", 0, 0, 4, 2, 1},
		{"2533274823110022", 1, 1, 5, 4, 1},
		{"2533274829350000", 4, 4, 3, 4, 2},
		{"2533274833178266", 0, 0, 1, 3, 0},
		{"2533274858283686", 4, 4, 3, 4, 2},
		{"2535450759128461", 0, 0, 4, 3, 0},
		{"2535452521259564", 3, 3, 5, 2, 2},
		{"2535469190789936", 1, 2, 2, 5, 0},
	},
	// 64e8adfa — CTF:Arena sur Catalyst : 46 securisation(s) a l oracle.
	"64e8adfa": {
		{"2533274792763167", 5, 3, 18, 12, 5},
		{"2533274808613055", 2, 2, 15, 15, 9},
		{"2533274823110022", 5, 5, 20, 16, 1},
		{"2535413221816250", 2, 4, 10, 21, 7},
		{"2535449464686885", 7, 21, 9, 19, 4},
		{"2535449963449748", 4, 5, 17, 12, 10},
		{"2535456378021162", 15, 21, 24, 9, 4},
		{"2535465820713037", 6, 13, 8, 19, 3},
	},
	// 7fce3219 — CTF:Arena sur Takamanohara : 15 securisation(s) a l oracle.
	"7fce3219": {
		{"2533274823110022", 2, 2, 13, 25, 8},
		{"2533274858283686", 4, 6, 23, 20, 11},
		{"2533274943116584", 0, 0, 27, 17, 9},
		{"2533274955626571", 1, 24, 25, 21, 12},
		{"2535415254450151", 4, 9, 10, 20, 8},
		{"2535458465148583", 1, 1, 25, 24, 5},
		{"2535469190789936", 1, 15, 20, 21, 7},
		{"2654577101798078", 2, 13, 26, 21, 10},
	},
	// 8bc6074f — CTF:Arena sur Origin : 20 securisation(s) a l oracle.
	"8bc6074f": {
		{"2533274823110022", 4, 4, 7, 9, 3},
		{"2533274829350000", 6, 15, 6, 10, 5},
		{"2533274833178266", 1, 1, 3, 9, 6},
		{"2533274858283686", 1, 4, 15, 13, 2},
		{"2535449383340628", 4, 7, 13, 5, 6},
		{"2535450759128461", 0, 0, 9, 10, 3},
		{"2535452521259564", 3, 2, 14, 12, 6},
		{"2535461109438273", 0, 0, 0, 2, 0},
		{"2535469190789936", 1, 9, 14, 11, 7},
		{"bid(20.0)", 0, 0, 0, 0, 0},
	},
	// a0c36016 — CTF:Arena sur Forest : 16 securisation(s) a l oracle.
	"a0c36016": {
		{"2533274823110022", 2, 1, 9, 15, 6},
		{"2533274858283686", 2, 3, 22, 18, 4},
		{"2533274858911298", 4, 2, 10, 18, 4},
		{"2535425652517893", 1, 4, 16, 19, 9},
		{"2535427927026623", 1, 6, 26, 11, 12},
		{"2535448372866366", 1, 1, 17, 18, 9},
		{"2535456861208728", 5, 7, 22, 17, 8},
		{"2535469190789936", 0, 8, 15, 21, 9},
	},
	// b8a44fe8 — CTF:Arena sur Forest : 43 securisation(s) a l oracle.
	"b8a44fe8": {
		{"2533274823110022", 2, 2, 25, 14, 5},
		{"2533274858283686", 14, 25, 12, 9, 4},
		{"2533274858911298", 3, 1, 9, 18, 6},
		{"2533274897257135", 2, 2, 9, 18, 5},
		{"2533275027826442", 1, 0, 21, 12, 7},
		{"2535442462807197", 15, 13, 17, 13, 5},
		{"2535455799302553", 6, 8, 20, 20, 2},
		{"2535469190789936", 0, 0, 11, 20, 10},
	},
	// bc60b4d9 — CTF:Arena sur Illusion : 12 securisation(s) a l oracle.
	"bc60b4d9": {
		{"2533274823110022", 2, 2, 10, 11, 2},
		{"2533274858283686", 0, 0, 11, 10, 8},
		{"2533274897645479", 3, 3, 13, 9, 1},
		{"2533274924336100", 2, 2, 5, 12, 6},
		{"2533274943116584", 0, 1, 9, 10, 4},
		{"2535458465148583", 0, 0, 11, 10, 4},
		{"2535469190789936", 0, 1, 9, 11, 7},
		{"2654577101798078", 5, 10, 13, 8, 3},
	},
	// bf5ced1b — CTF:Arena sur Illusion : 5 securisation(s) a l oracle.
	"bf5ced1b": {
		{"2533274823110022", 0, 0, 0, 4, 3},
		{"2533274858283686", 0, 0, 3, 5, 0},
		{"2535412865870758", 2, 2, 3, 2, 2},
		{"2535415855124749", 0, 0, 2, 4, 0},
		{"2535434888128184", 2, 2, 8, 0, 1},
		{"2535459651975585", 1, 5, 2, 4, 4},
		{"2535469190789936", 0, 0, 3, 5, 0},
		{"2535472614974408", 0, 0, 5, 2, 1},
		{"bid(30.0)", 0, 0, 0, 0, 0},
	},
	// cde26226 — CTF:Arena sur Critical Dewpoint : 38 securisation(s) a l oracle.
	"cde26226": {
		{"2533274817658307", 2, 2, 14, 15, 7},
		{"2533274818074729", 9, 3, 10, 19, 12},
		{"2533274823110022", 1, 0, 21, 23, 6},
		{"2533274849050722", 4, 2, 22, 24, 8},
		{"2533274858283686", 7, 7, 26, 21, 15},
		{"2535415492484355", 5, 12, 24, 14, 10},
		{"2535468122262494", 8, 35, 28, 22, 3},
		{"2535469190789936", 2, 10, 18, 25, 7},
	},
	// f8efc5ca — CTF:Arena sur Absolution : 21 securisation(s) a l oracle.
	"f8efc5ca": {
		{"2533274804730586", 0, 0, 7, 10, 6},
		{"2533274811578968", 3, 3, 10, 9, 5},
		{"2533274823110022", 7, 2, 12, 11, 4},
		{"2533274833178266", 0, 0, 4, 11, 1},
		{"2533274858283686", 2, 4, 14, 12, 8},
		{"2535458336606528", 2, 7, 17, 8, 3},
		{"2535467632752562", 5, 5, 10, 10, 2},
		{"2535469190789936", 2, 1, 7, 12, 8},
	},
	// fb1a1a72 — CTF:Arena sur Banished Narrows : 17 securisation(s) a l oracle.
	"fb1a1a72": {
		{"2533274795950878", 1, 1, 19, 19, 2},
		{"2533274823110022", 0, 0, 18, 17, 8},
		{"2535408981717353", 0, 0, 9, 19, 5},
		{"2535417580715145", 2, 6, 16, 17, 4},
		{"2535425179403277", 3, 4, 16, 17, 9},
		{"2535450323793545", 0, 0, 14, 19, 5},
		{"2535469190789936", 7, 9, 24, 15, 3},
		{"2535473461033821", 4, 1, 24, 18, 11},
	},
}

// fsMaxComp borne le balayage, comme E1-bis : l'archetype ne porte pas des centaines de
// composants, et le plus haut jamais nomme est 24 (`flag_steals`).
const fsMaxComp = 64

// fsCandidat est le verdict d'UN emplacement sur UN film.
type fsCandidat struct {
	nom string
	// exacts / desaccords portent sur les seuls slots APPARIES a un joueur nomme.
	exacts, desaccords int
	// pire est le plus grand ecart absolu observe, pour ordonner les quasi-candidats.
	pire int
	// detail nomme les joueurs en desaccord. Il sert au TEMOIN : quand c'est LUI qui tombe,
	// le film est ecarte, et il faut pouvoir dire de QUI vient le desaccord sans relancer.
	detail []string
}

// fsVerdict confronte UN emplacement a une cible par joueur, et rend son verdict.
//
// PURE ET SANS `testing` : c'est ce qui permet de la rejouer sur le TEMOIN POSITIF
// (`flag_grabs`, emplacement deja nomme) dans le meme processus, et donc de prouver que le
// balayage sait reconnaitre une reponse juste quand il en voit une.
//
// Un slot SANS EMISSION est la lecture « zero » du compteur, pas une absence de mesure.
func fsVerdict(
	recs []objectiveevents.StatRecord, identity map[int]string,
	c objectiveevents.StatComponent, cible map[string]int,
) fsCandidat {
	series := objectiveevents.SeriesTotal(recs, c, false)
	v := fsCandidat{nom: fmt.Sprintf("comp %d %s%s", c.Comp, fsSide(c.SideB), fsStrict(c.Strict))}
	for slot, xuid := range identity {
		publie := 0
		if pts := series[slot]; len(pts) > 0 {
			publie = int(pts[len(pts)-1].Value)
		}
		ecart := publie - cible[xuid]
		if ecart == 0 {
			v.exacts++
			continue
		}
		v.desaccords++
		v.detail = append(v.detail,
			fmt.Sprintf("slot %d (%s) publie %d, oracle %d", slot, xuid, publie, cible[xuid]))
		if ecart < 0 {
			ecart = -ecart
		}
		if ecart > v.pire {
			v.pire = ecart
		}
	}
	sort.Strings(v.detail)
	return v
}

func fsSide(b bool) string {
	if b {
		return "B"
	}
	return "A"
}

func fsStrict(s bool) string {
	if s {
		return " strict"
	}
	return ""
}

// fsIdentity construit le pont slot -> xuid par le triplet, et la cible par xuid.
func fsIdentity(recs []objectiveevents.StatRecord, oracle []fsJoueur,
	cible func(fsJoueur) int,
) (map[int]string, map[string]int) {
	lines := make([]objectiveevents.PlayerLine, 0, len(oracle))
	par := map[string]int{}
	for _, j := range oracle {
		par[j.xuid] = cible(j)
		if j.kills < 0 {
			continue // bot sans ligne de match : aucun pont possible, et c'est dit
		}
		lines = append(lines, objectiveevents.PlayerLine{
			XUID: j.xuid, Kills: j.kills, Deaths: j.deaths, Assists: j.assists,
		})
	}
	return objectiveevents.SlotIdentityFrom(recs, lines), par
}

// TestFlagSecuresSweep — LE BALAYAGE. Un film par processus, via `ZONE_FILM`.
//
// Il ECHOUE si le TEMOIN POSITIF ne se retrouve pas : un balayage qui ne sait plus reconnaitre
// `flag_grabs` a `comp 22 A` ne prouve rien de son silence sur `flag_secures`. C'est la
// contre-epreuve qui donne un sens au negatif.
func TestFlagSecuresSweep(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	oracle, ok := fsOracle[short]
	if !ok {
		t.Skipf("film %s hors corpus `flag_secures` (aucun oracle gele pour lui)", short)
	}
	recs := objectiveevents.StatRecords(p2aBobine(t, dir))
	if len(recs) == 0 {
		t.Fatalf("%s : aucun enregistrement de statistiques — rien a balayer", short)
	}

	// TEMOIN POSITIF : le balayage doit retrouver `flag_grabs` a `comp 22 A`, sans desaccord.
	idGrabs, cibleGrabs := fsIdentity(recs, oracle, func(j fsJoueur) int { return j.grabs })
	temoin := fsVerdict(recs, idGrabs,
		objectiveevents.StatComponent{Comp: 22, SideB: false}, cibleGrabs)
	t.Logf("FLAGSEC-TEMOIN\t%s\t%s (flag_grabs) : %d exact(s), %d desaccord(s)",
		short, temoin.nom, temoin.exacts, temoin.desaccords)
	// UN PONT VIDE N'EST PAS UN TEMOIN QUI TOMBE, et les deux ne doivent pas se confondre :
	// sans slot apparie, le film ne dit RIEN — ni pour ni contre — et l'exclure du corpus est
	// la seule lecture honnete. C'est le cas de `7fce3219` (CTF multi-manche) : le pont par le
	// triplet n'y rend aucun slot, et un « 0 exact, 0 desaccord » se lirait sinon comme un
	// accord parfait.
	if temoin.exacts == 0 && temoin.desaccords == 0 {
		t.Skipf("%s : AUCUN slot apparie (pont d'identite vide) — film hors corpus, il ne "+
			"temoigne ni pour ni contre", short)
	}
	if temoin.desaccords > 0 {
		for _, d := range temoin.detail {
			t.Logf("FLAGSEC-TEMOIN-KO\t%s\t%s", short, d)
		}
		t.Fatalf("%s : le TEMOIN POSITIF `flag_grabs` n'est plus reproduit par `comp 22 A` "+
			"(%d exact(s), %d desaccord(s)) — le balayage ne prouve rien dans cet etat",
			short, temoin.exacts, temoin.desaccords)
	}

	identity, cible := fsIdentity(recs, oracle, func(j fsJoueur) int { return j.secures })
	attendu := 0
	for _, j := range oracle {
		attendu += j.secures
	}
	t.Logf("FLAGSEC\t%s\t%d slot(s) apparie(s), %d securisation(s) a l'oracle sur %d ligne(s)",
		short, len(identity), attendu, len(oracle))
	fsNonVide(t, short, identity, cible)

	var exacts, proches []fsCandidat
	for comp := 0; comp <= fsMaxComp; comp++ {
		for _, sideB := range []bool{false, true} {
			for _, strict := range []bool{false, true} {
				c := objectiveevents.StatComponent{Comp: comp, SideB: sideB, Strict: strict}
				v := fsVerdict(recs, identity, c, cible)
				switch {
				case v.desaccords == 0:
					exacts = append(exacts, v)
				case v.desaccords <= fsProcheMax:
					proches = append(proches, v)
				}
			}
		}
	}
	sort.Slice(proches, func(i, j int) bool { return proches[i].desaccords < proches[j].desaccords })
	for _, v := range exacts {
		t.Logf("FLAGSEC-EXACT\t%s\t%s\t%d exact(s)\t0 desaccord", short, v.nom, v.exacts)
	}
	for _, v := range proches {
		t.Logf("FLAGSEC-PROCHE\t%s\t%s\t%d exact(s)\t%d desaccord(s)\tpire ecart %d",
			short, v.nom, v.exacts, v.desaccords, v.pire)
	}
	if len(exacts) == 0 {
		t.Logf("FLAGSEC-AUCUN\t%s\taucun emplacement ne reproduit `flag_secures` sans desaccord", short)
	}

	// LA GARDE — depuis que `comp 23 B` est nomme, un desaccord est une REGRESSION, pas un
	// resultat de balayage. Le film-temoin, lui, n'est pas fige : c'est le corpus entier.
	garde := fsVerdict(recs, identity, fsSecuresSlot, cible)
	for _, d := range garde.detail {
		t.Errorf("FLAGSEC-GARDE-KO\t%s\t%s", short, d)
	}
	if garde.desaccords > 0 {
		t.Errorf("FLAGSEC-GARDE\t%s\t`comp 23 B` ne reproduit plus `flag_secures` : %d exact(s), "+
			"%d desaccord(s), pire ecart %d", short, garde.exacts, garde.desaccords, garde.pire)
	}
}

// fsSecuresSlot est l'emplacement NOMME de `flag_secures` — la meme cle que
// `namedStatSlots[ObjectiveTypeFlag][{23, sideB}]`, ecrite ici une fois pour la garde.
var fsSecuresSlot = objectiveevents.StatComponent{Comp: 23, SideB: true}

// fsProcheMax borne la liste des QUASI-candidats journalises. Sans borne, le balayage
// recracherait ses 260 emplacements a chaque film et la liste ne se lirait plus. Deux
// desaccords, c'est un emplacement qui colle sur 6 joueurs sur 8 : assez proche pour meriter
// d'etre nomme dans le rapport, assez loin pour ne jamais etre publie.
const fsProcheMax = 2

// fsNonVide — LE CONTROLE DE NON-VACUITE, et il est indispensable au gate.
//
// « Zero desaccord » ne vaut RIEN si la cible est nulle partout : un composant muet serait alors
// declare exact sur tous les films. Ce releve dit, pour les seuls slots APPARIES, combien de
// joueurs ont une securisation a l'oracle et combien elles font. C'est ce chiffre qui rend le
// verdict lisible : un accord sur cinq joueurs a valeur non nulle est une preuve, un accord sur
// zero n'en est pas une.
func fsNonVide(t *testing.T, short string, identity map[int]string, cible map[string]int) {
	t.Helper()
	nonNuls, somme := 0, 0
	for _, xuid := range identity {
		if v := cible[xuid]; v > 0 {
			nonNuls++
			somme += v
		}
	}
	t.Logf("FLAGSEC-CIBLE\t%s\t%d joueur(s) apparie(s) a securisation NON NULLE, somme %d",
		short, nonNuls, somme)
}
