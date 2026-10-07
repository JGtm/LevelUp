package grammar

// ecs_widths_guard_test.go — LE CONTROLE G4 DE LA TABLE ECS : ses largeurs ENTIERES, confrontees
// a ce que le deser de production consomme reellement. Sorti de `ecs_table_guard_test.go` pour
// tenir sous le seuil de 500 lignes du depot ; il partage son lecteur de table (`loadECSTable`)
// et sa constante de chemin, donc il n'y a toujours qu'UN lecteur du TSV.

import (
	"strconv"
	"testing"
)

// --- G4 : LES LARGEURS ENTIERES DE LA TABLE, CONFRONTEES AU CODE -----------------------------
//
// (lot E, item E.7 du PLAN_V2_REJEU_FILM, 2026-09-06.)
//
// # CE QUE CE CONTROLE MESURE, ET COMMENT
//
// Les lignes de la table qui portent une largeur ENTIERE en `bits_typ` (leur nombre :
// `ecsLargeursFixes` + `ecsLargeursGardees`) sont toutes declarees « porte ». Le controle execute
// le deser de production (`consumeByName`) sur des tampons SYNTHETIQUES et lit le nombre de bits
// qu il consomme. Zero fixture, zero variable d environnement, zero octet de film : il tourne en
// CI comme le reste.
//
// TROIS MOTIFS, ET C'EST LA CLE. Beaucoup de composants sont GARDES : leur largeur depend des
// bits lus. Le controle mesure donc sur `0x00`, `0xFF` et `0xAA`, puis classe :
//
//   - LES TROIS MOTIFS S'ACCORDENT -> la largeur est FIXE, et elle DOIT egaler `bits_typ`.
//     C est la seule categorie ou un ecart est une faute. `ecsLargeursFixes` lignes, qui
//     s accordent toutes avec la table hors des ecarts connus (`ecsEcartsAdmis`).
//   - LES MOTIFS DIVERGENT -> la largeur est gardee par le flux ; l'entier de la table est alors
//     une valeur NOMINALE, que ce controle ne peut pas confronter. `ecsLargeursGardees`
//     lignes. Leur compte est GELE : une ligne qui change de categorie est un signal, pas un
//     silence.
//
// # POURQUOI DES TAMPONS SYNTHETIQUES ET PAS UN FILM
//
// Parce que la question posee est « ce deser consomme-t-il la largeur que la table annonce », pas
// « que vaut ce champ sur ce film ». Un film ne visiterait qu'une combinaison de portes parmi
// beaucoup, et il faudrait le versionner. Les trois motifs couvrent les deux polarites de chaque
// porte et une alternance — assez pour separer le fixe du garde, qui est tout ce que ce controle
// pretend faire.

// ecsBitsPatterns : les trois motifs de tampon. `0x00` et `0xFF` couvrent les deux polarites de
// toute porte d'un bit ; `0xAA` ajoute l'alternance, qui separe une porte d'un champ de valeur.
var ecsBitsPatterns = []byte{0x00, 0xFF, 0xAA}

// ecsProbeBytes : la taille du tampon de mesure. Large : le deser le plus lourd de la table
// consomme 826 bits sur un tampon de uns (`object-region-state-component`), et un tampon trop
// court ferait lire des zeros de bourrage — donc une largeur fausse, et un vert faux.
const ecsProbeBytes = 512

// ecsLargeursFixes / ecsLargeursGardees : les comptes GELES des deux categories. Leur somme est le
// nombre de lignes de la table qui portent un ENTIER en `bits_typ` ; chacune y entre quand son
// composant est porte avec une largeur entiere, et tombe dans la categorie que les trois motifs
// lui donnent.
//
// D OU VIENNENT LES DEUX VALEURS : les lecteurs portes, par archetype (index des lignes).
//
//	FIXES (155)
//	ti=0  (9)  i2 i5 i6 i7 i8 i9 i11 i16 i17     ti=12 (8)  i0 i1 i7 i8 i10 i11 i12 i14
//	ti=1  (1)  i11                                ti=13 (2)  i0 i1 (ecart admis)
//	ti=2  (4)  i3 i11 i16 i17                     ti=14 (1)  i2
//	ti=3  (1)  i1                                 ti=16 (1)  i1
//	ti=4  (1)  i0                                 ti=20 (1)  i1
//	ti=5  (21) i0..i6 i8 i9 i11 i13..i19 i21      ti=30 (1)  i2
//	           i23 i25 i26                        ti=34 (8)  i0 i1 i3 i4 i5 i6 i8 i16
//	ti=6  (2)  i56 i57                            ti=35 (10) i4 i31 i34 i35 i37 i38 i40 i41 i50 i51
//	ti=9  (9)  i0..i8                             ti=37 (10) i4 i11 i14 (ecart admis) i18 i20 i24..i27 i30
//	ti=10 (4)  i0 i1 i24 i25                      ti=38 (1)  i19
//	ti=11 (31) i0 i1 i3 i5..i8 i10..i33           ti=40 (9)  i30 i31 i32 i36 i37 i39 i41 i42 i45
//	                                              ti=41 (2)  i19 i21
//	                                              ti=42 (2)  i18 i20
//	                                              ti=43 (15) i18 i19 i20 i22 i23 i25..i28 i30
//	                                                         i32 i33 i38..i40
//	                                              ti=47 (1)  i1
//
//	GARDEES (66) : ti=13 32, ti=35 21, ti=37 6, et une ligne chacun pour ti=2, 9, 38, 40, 41, 42, 43.
//
// COMMENT LES FAIRE BOUGER. Vers le haut de `ecsLargeursFixes` (une ligne neuve a largeur fixe, ou
// une largeur gardee devenue fixe) : bienvenu, c est un progres de portage. Vers le bas : un deser
// a gagne une porte — a expliquer avant de reecrire le chiffre. La liste ci-dessus se reecrit avec
// les deux constantes. L historique date des deux comptes vit dans les comptes rendus des lots qui
// les ont fait bouger (le dernier releve : `LOT_L4a.md` §14 de la campagne de grammaire).
const (
	ecsLargeursFixes   = 155
	ecsLargeursGardees = 66
)

// ecsEcartAdmis decrit un ecart CONNU entre la largeur fixe mesuree et l'entier de la table.
type ecsEcartAdmis struct {
	TI, I         int
	Component     string
	Table, Mesure int
	Pourquoi      string
}

// ecsEcartsAdmis : LES ECARTS CONNUS, dates.
//
// AUCUN N'EST UN BOGUE DE DECODAGE, et aucun n'est silencieux : chacun est ici avec sa raison, et
// le controle echoue si un ecart de plus apparait — ou si l'un de ceux-ci se resorbe sans qu'on
// retire sa ligne, que la resorption vienne du CODE (`largeur != e.Mesure`) ou de la TABLE
// (`e.Table != r.BitsTyp`). LES DEUX SENS sont controles depuis le 2026-09-06 : le second manquait,
// et c'etait exactement la manoeuvre de l'item E.9 (D-3 de la revue E-R2).
//
// CHRONIQUE. 2026-09-06 (item E.7) : TROIS ecarts. 2026-09-06 (item E.9, decision utilisateur 10) :
// **DEUX** — `ti=35 i=50 biped-map-editor-flag-component` en sort, parce que la table a ete
// corrigee (`bits_typ` 1 -> 8) et que le code et la table s'accordent desormais. C'etait le seul
// des trois dont la mesure donnait UNE largeur : les deux qui restent sont des largeurs GARDEES,
// dont aucun entier unique ne peut rendre compte.
//
// RETRAIT DES DEUX DERNIERES : le jour ou la colonne `bits_typ` distingue une largeur FIXE d'une
// largeur NOMINALE (par exemple en prefixant les nominales). C'est une reecriture de la table,
// hors du perimetre de l'item E.9, qui corrige entree par entree sur mesure.
var ecsEcartsAdmis = []ecsEcartAdmis{
	{
		TI: 13, I: 1, Component: "managed-object-property-component", Table: 28, Mesure: 4,
		Pourquoi: "largeur gardee par le TAG, et la table le dit elle-meme dans ses notes : " +
			"« largeur totale 4/5/8/28/36 selon le tag ». Les trois motifs tombent sur des tags " +
			"a charge nulle (t0, t10, t15), d'ou 4 bits — le R(4) du tag seul. `bits_typ` fige " +
			"ici le cas t3, pas une largeur fixe. Mesure tenue par le controle G5.",
	},
	{
		TI: 37, I: 14, Component: "object-dissolver-component", Table: 4, Mesure: 113,
		Pourquoi: "largeur gardee par la VALEUR lue, pas par un bit de porte : " +
			"`consumeObjectDissolver` lit R(4) puis, SI la valeur n'est pas 13, R(96)+R(12)+R(1). " +
			"Les trois motifs manquent la valeur 13, d'ou 113. La table fige le cas nominal " +
			"(« champ court », 4 bits), qui est le cas v==13 — et sa colonne `notes` le dit " +
			"depuis le 2026-09-06 (item E.9), avec la mesure que tient le controle G5.",
	},
}

// TestG4LargeursEntieresSuiventLeCode — LE CONTROLE.
func TestG4LargeursEntieresSuiventLeCode(t *testing.T) {
	rows := loadECSTable(t)

	admis := map[string]ecsEcartAdmis{}
	for _, e := range ecsEcartsAdmis {
		admis[ecsCle(e.TI, e.I, e.Component)] = e
	}

	var fixes, gardees int
	vus := map[string]bool{}
	for _, r := range rows {
		if r.BitsTyp < 0 {
			continue
		}
		largeur, fixe := ecsLargeurConsommee(r)
		if !fixe {
			gardees++
			continue
		}
		fixes++
		cle := ecsCle(r.TI, r.I, r.Component)
		if e, ok := admis[cle]; ok {
			vus[cle] = true
			if largeur != e.Mesure {
				t.Errorf("G4 : ligne %d (ti=%d i=%d %s) est un ecart ADMIS a %d bits, mais le code "+
					"en consomme maintenant %d — l'ecart a bouge par le CODE, sa justification est a revoir :\n  %s",
					r.LineNo, r.TI, r.I, r.Component, e.Mesure, largeur, e.Pourquoi)
			}
			// L'ECART PEUT AUSSI SE RESORBER PAR LA TABLE, et c'est la manoeuvre que l'item E.9
			// vient d'executer sur `ti=35 i=50` : corriger `bits_typ` a la valeur mesuree rend
			// l'entree d'allowlist sans objet, et son champ `Table` devient un mensonge. Sans ce
			// controle, rien n'obligeait a retirer la ligne (D-3 de la revue E-R2, 2026-09-06).
			if e.Table != r.BitsTyp {
				t.Errorf("G4 : ligne %d (ti=%d i=%d %s) est un ecart ADMIS declare contre une table a "+
					"%d bits, mais la table annonce maintenant %d.\n"+
					"Si elle a ete CORRIGEE a la valeur mesuree (%d), l'ecart n'existe plus : RETIRER "+
					"sa ligne d'`ecsEcartsAdmis`, une allowlist sans objet est du code mort.\n"+
					"Si elle a ete changee pour autre chose, mettre a jour le champ `Table` avec la "+
					"raison datee.",
					r.LineNo, r.TI, r.I, r.Component, e.Table, r.BitsTyp, largeur)
			}
			continue
		}
		if largeur != r.BitsTyp {
			t.Errorf("G4 : ligne %d (ti=%d i=%d %s) : la table annonce %d bits, le deser de "+
				"production en consomme %d (largeur FIXE : les trois motifs de tampon s'accordent).\n"+
				"Soit le portage a change, soit la table est perimee — dans les deux cas cela se "+
				"tranche, jamais ne se tait. Un ecart connu et justifie entre dans `ecsEcartsAdmis`.",
				r.LineNo, r.TI, r.I, r.Component, r.BitsTyp, largeur)
		}
	}

	for cle, e := range admis {
		if !vus[cle] {
			t.Errorf("G4 : l'ecart admis %s (ti=%d i=%d) n'existe plus ou n'est plus a largeur "+
				"fixe : retirer sa ligne d'`ecsEcartsAdmis`, une allowlist sans cible est du code mort",
				e.Component, e.TI, e.I)
		}
	}
	if fixes != ecsLargeursFixes || gardees != ecsLargeursGardees {
		t.Errorf("G4 : %d largeurs fixes et %d gardees, gelees a %d et %d "+
			"(ecsLargeursFixes, ecsLargeursGardees).\n"+
			"Une ligne a change de categorie ou est entree dans le controle. L'expliquer, puis "+
			"reecrire les deux constantes et leur liste par archetype ; l'historique date va au "+
			"compte rendu du lot.",
			fixes, gardees, ecsLargeursFixes, ecsLargeursGardees)
	}
}

// ecsLargeurConsommee execute le deser de production sur les trois motifs et rend la largeur
// consommee, plus `fixe` : vrai quand les trois motifs s'accordent.
func ecsLargeurConsommee(r ecsRow) (int, bool) {
	var premiere int
	for k, motif := range ecsBitsPatterns {
		buf := make([]byte, ecsProbeBytes)
		for i := range buf {
			buf[i] = motif
		}
		br := LecteurSur(buf)
		if _, _, ported := consumeByName(br, r.Component, uint32(r.TI), r.Level); !ported {
			return 0, false // non porte : aucune largeur a confronter
		}
		if k == 0 {
			premiere = br.BitPos()
			continue
		}
		if br.BitPos() != premiere {
			return 0, false
		}
	}
	return premiere, true
}

// ecsCle identifie une ligne de la table.
func ecsCle(ti, i int, component string) string {
	return itoaECS(ti) + "|" + itoaECS(i) + "|" + component
}

// itoaECS evite d'elargir la portee de strconv a un seul usage de plus.
func itoaECS(n int) string { return strconv.Itoa(n) }

// --- G5 : LES MESURES CITEES PAR LA TABLE SONT TENUES PAR UN TEST ----------------------------
//
// (item E.9 du PLAN_V2_REJEU_FILM, decision utilisateur 10 du 2026-09-06.)
//
// # POURQUOI CE CONTROLE EXISTE
//
// La revision de `ecs_table.tsv` demandee par la decision 10 exige que « chaque entree corrigee
// soit adossee a une mesure ». Une note dans une colonne de prose n'est pas une mesure : c'est une
// affirmation, et le lot E a deja montre ce que devient une affirmation que personne ne rejoue
// (« 25 familles sur 30 », « un rendu %+v »). Les lignes de la table dont la colonne `notes` CITE
// une largeur mesuree la voient donc figee ici, motif par motif.
//
// # CE QU'IL MESURE, EXACTEMENT
//
// La largeur consommee par `consumeByName` sur CHACUN des trois motifs de tampon, separement —
// la ou G4 ne retient que l'accord ou le desaccord des trois. C'est ce detail qui donne son sens
// a une note comme « 45 ou 60 selon le motif » : sans lui, on ne saurait pas laquelle des deux
// branches chaque motif visite.

// ecsMesureCitee fige les trois largeurs d'une ligne dont la table cite la mesure.
type ecsMesureCitee struct {
	TI, I     int
	Component string
	// Largeurs sont les bits consommes sur `ecsBitsPatterns`, dans l'ordre : 0x00, 0xFF, 0xAA.
	Largeurs [3]int
	Pourquoi string
}

// ecsMesuresCitees : les quatre lignes dont la colonne `notes` porte une largeur mesuree.
//
// TROIS Y SONT PARCE QUE LEUR `bits_typ` NE PEUT PAS DIRE LA VERITE A LUI SEUL (la largeur depend
// du flux), et une parce que sa correction du 2026-09-06 s'appuie sur cette mesure.
var ecsMesuresCitees = []ecsMesureCitee{
	{
		TI: 13, I: 1, Component: "managed-object-property-component", Largeurs: [3]int{4, 4, 4},
		Pourquoi: "largeur gardee par le TAG : 4/5/8/28/36 selon lui. Les trois motifs tombent " +
			"sur des tags a charge nulle (t0, t10, t15), d'ou le R(4) du tag seul.",
	},
	{
		TI: 35, I: 50, Component: "biped-map-editor-flag-component", Largeurs: [3]int{8, 8, 8},
		Pourquoi: "R(8) plat, sans porte : `consumeBipedMapEditorFlag` (FUN_142f02854), " +
			"« CONFIRMED bit-exact from the decompile ». C'est la mesure sur laquelle la colonne " +
			"`bits_typ` a ete corrigee de 1 a 8 le 2026-09-06.",
	},
	{
		TI: 37, I: 14, Component: "object-dissolver-component", Largeurs: [3]int{113, 113, 113},
		Pourquoi: "largeur gardee par la VALEUR lue : `consumeObjectDissolver` lit R(4) puis, SI " +
			"cette valeur n'est pas 13, R(96)+R(12)+R(1). Les trois motifs manquent la valeur 13.",
	},
	{
		TI: 43, I: 0, Component: "object-position-component", Largeurs: [3]int{45, 60, 60},
		Pourquoi: "largeur gardee par le bit precHigh de tete, ET propre a la CARTE : le chemin " +
			"dominant vaut 3 de porte + AxisW + 2 de queue (45 bits avec le defaut Cliffhanger " +
			"13/13/14, cf. `WorldObjectPrecision`), l'autre branche en consomme davantage.",
	},
}

// TestG5MesuresCiteesParLaTable — les largeurs que la table cite en prose sont celles que le
// deser consomme, motif par motif.
func TestG5MesuresCiteesParLaTable(t *testing.T) {
	rows := loadECSTable(t)

	index := map[string]ecsRow{}
	for _, r := range rows {
		index[ecsCle(r.TI, r.I, r.Component)] = r
	}
	for _, m := range ecsMesuresCitees {
		cle := ecsCle(m.TI, m.I, m.Component)
		r, ok := index[cle]
		if !ok {
			t.Errorf("G5 : la ligne citee ti=%d i=%d %s n'existe plus dans la table : retirer son "+
				"entree, une mesure sans cible est du code mort", m.TI, m.I, m.Component)
			continue
		}
		got := ecsLargeursParMotif(r)
		if got != m.Largeurs {
			t.Errorf("G5 : ti=%d i=%d %s consomme %v bits sur les motifs %v, fige a %v "+
				"(mesure du 2026-09-06).\nLa colonne `notes` de la table cite ces largeurs : les "+
				"deux se corrigent ensemble, jamais l'une sans l'autre.\n  %s",
				m.TI, m.I, m.Component, got, ecsBitsPatterns, m.Largeurs, m.Pourquoi)
		}
	}
}

// ecsLargeursParMotif rend les bits consommes par le deser de production sur chacun des trois
// motifs, sans les confondre. Une largeur vaut -1 quand le composant n'est pas porte.
func ecsLargeursParMotif(r ecsRow) [3]int {
	var out [3]int
	for k, motif := range ecsBitsPatterns {
		buf := make([]byte, ecsProbeBytes)
		for i := range buf {
			buf[i] = motif
		}
		br := LecteurSur(buf)
		if _, _, ported := consumeByName(br, r.Component, uint32(r.TI), r.Level); !ported {
			out[k] = -1
			continue
		}
		out[k] = br.BitPos()
	}
	return out
}
