package filmdec

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
// 179 lignes de la table portent une largeur ENTIERE en `bits_typ` — et toutes sont declarees
// « porte ». Le controle execute le deser de production (`consumeByName`) sur des tampons
// SYNTHETIQUES et lit le nombre de bits qu'il consomme. Zero fixture, zero variable
// d'environnement, zero octet de film : il tourne en CI comme le reste.
//
// TROIS MOTIFS, ET C'EST LA CLE. Beaucoup de composants sont GARDES : leur largeur depend des
// bits lus. Le controle mesure donc sur `0x00`, `0xFF` et `0xAA`, puis classe :
//
//   - LES TROIS MOTIFS S'ACCORDENT -> la largeur est FIXE, et elle DOIT egaler `bits_typ`.
//     C est la seule categorie ou un ecart est une faute. 114 lignes aujourd hui, dont 111
//     s accordent avec la table et 3 sont des ecarts connus, listes plus bas.
//   - LES MOTIFS DIVERGENT -> la largeur est gardee par le flux ; l'entier de la table est alors
//     une valeur NOMINALE, que ce controle ne peut pas confronter. 65 lignes aujourd'hui. Leur
//     compte est GELE : une ligne qui change de categorie est un signal, pas un silence.
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

// ecsLargeursFixes / ecsLargeursGardees : les comptes GELES des deux categories, mesures le
// 2026-09-06 sur les 179 lignes a largeur entiere.
//
// COMMENT LES FAIRE BOUGER. Vers le haut de `ecsLargeursFixes` (une largeur gardee devient fixe) :
// bienvenu, et c'est un progres de portage. Vers le bas : c'est qu'un deser a gagne une porte —
// a expliquer avant de reecrire le chiffre. La somme des deux, elle, vaut le nombre de lignes a
// largeur entiere de la table.
const (
	ecsLargeursFixes   = 114
	ecsLargeursGardees = 65
)

// ecsEcartAdmis decrit un ecart CONNU entre la largeur fixe mesuree et l'entier de la table.
type ecsEcartAdmis struct {
	TI, I         int
	Component     string
	Table, Mesure int
	Pourquoi      string
}

// ecsEcartsAdmis : LES TROIS ECARTS CONNUS, dates du 2026-09-06 (lot E, item E.7).
//
// AUCUN N'EST UN BOGUE DE DECODAGE, et aucun n'est silencieux : chacun est ici avec sa raison, et
// le controle echoue si un QUATRIEME apparait — ou si l'un de ceux-ci se resorbe sans qu'on
// retire sa ligne.
//
// RETRAIT : ces trois lignes tombent le jour ou la colonne `bits_typ` distingue une largeur FIXE
// d'une largeur NOMINALE (par exemple en prefixant les nominales). C'est une reecriture de la
// table, hors du perimetre du lot E, qui ne touche pas aux mesures de retro-ingenierie.
var ecsEcartsAdmis = []ecsEcartAdmis{
	{
		TI: 13, I: 1, Component: "managed-object-property-component", Table: 28, Mesure: 4,
		Pourquoi: "largeur gardee par le TAG, et la table le dit elle-meme dans ses notes : " +
			"« largeur totale 4/5/8/28/36 selon le tag ». Les trois motifs tombent sur des tags " +
			"a charge nulle (t0, t10, t15), d'ou 4 bits — le R(4) du tag seul. `bits_typ` fige " +
			"ici le cas t3, pas une largeur fixe.",
	},
	{
		TI: 35, I: 50, Component: "biped-map-editor-flag-component", Table: 1, Mesure: 8,
		Pourquoi: "C'EST LA TABLE QUI EST PERIMEE, pas le code. `consumeBipedMapEditorFlag` lit " +
			"un R(8) plat, « CONFIRMED bit-exact from the decompile » (FUN_142f02854, refill " +
			"8 bits unique). La correction de la colonne appartient a un lot qui revise la " +
			"table ; la signaler est le role de ce controle.",
	},
	{
		TI: 37, I: 14, Component: "object-dissolver-component", Table: 4, Mesure: 113,
		Pourquoi: "largeur gardee par la VALEUR lue, pas par un bit de porte : " +
			"`consumeObjectDissolver` lit R(4) puis, SI la valeur n'est pas 13, R(96)+R(12)+R(1). " +
			"Les trois motifs manquent la valeur 13, d'ou 113. La table fige le cas nominal " +
			"(« champ court », 4 bits), qui est le cas v==13.",
	},
}

// TestG4LargeursEntieresSuiventLeCode — LE CONTROLE.
func TestG4LargeursEntieresSuiventLeCode(t *testing.T) {
	rows := loadECSTable(t)
	release := LockProcessDecode()
	defer release()

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
					"en consomme maintenant %d — l'ecart a bouge, sa justification est a revoir :\n  %s",
					r.LineNo, r.TI, r.I, r.Component, e.Mesure, largeur, e.Pourquoi)
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
		t.Errorf("G4 : %d largeurs fixes et %d gardees, gelees a %d et %d (mesure du 2026-09-06).\n"+
			"Une ligne a change de categorie : un deser a gagne ou perdu une porte. L'expliquer, "+
			"puis reecrire les deux constantes avec la date.",
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
		br := NewBitReader(buf)
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
