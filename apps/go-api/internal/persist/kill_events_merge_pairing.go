package persist

// kill_events_merge_pairing.go — QUI EST APPARIE A QUI, QUAND L INSTANT NE SUFFIT PLUS.
//
// ─── LE DEFAUT QUE CE FICHIER FERME ────────────────────────────────────────────────────────
//
// L appariement tenait dans une ligne : « l instant porte EXACTEMENT une mort de credit et
// EXACTEMENT une ligne de film ». Il reposait sur une propriete mesuree — `(match_id, time_ms)`
// est unique — dont le lot 2.9 a montre qu elle ne vaut QUE DANS CHAQUE COTE PRIS SEUL, jamais
// ENTRE les deux (mesure et temoin : en-tete de `kill_events_merge.go`). Deux morts a la meme
// milliseconde dont chaque cote ne voit que la sienne donnaient « une mort de chaque cote » : la
// clef les appariait, le controle d identite rendait l erreur, et la passe refusait LE FILM
// ENTIER — 249 films ecrits sur 250 dans la tranche 1 du backlog du 2026-09-17.
//
// ─── LA REGLE, ET SES QUATRE CAS ───────────────────────────────────────────────────────────
//
// LA VICTIME D ABORD, L INSTANT EN REPLI. La victime est ce que les deux cotes savent de la mort
// (98,9 % des lignes de film portent un `victim_xuid` — 109 750 sur 111 002 au 2026-09-16, et
// 100 % des lignes de credit) ; l instant, lui, ne distingue pas deux morts simultanees. Appliquee
// a un instant qui ne porte qu une mort de chaque cote avec la meme victime — le cas de
// l immense majorite — elle rend exactement ce que rendait la clef.
//
//	appariee                    la ligne de film et la mort de credit portent LA MEME victime.
//	                            Plusieurs morts au meme instant s apparient alors une a une.
//	orpheline d instant libre   aucune mort de credit a cet instant : conservee telle quelle,
//	                            c est la population des morts de bot (cf. `MergeCreditAndFilm`).
//	orpheline d instant partage LE CAS DU LOT 2.9 : une mort de credit est bien a cet instant,
//	                            mais sa victime N EST PAS celle de la ligne de film, et toutes
//	                            les morts de credit de l instant portent une victime resolue.
//	                            C est une AUTRE MORT, prouvee telle : elle se conserve et se
//	                            compte, elle ne se rend plus en erreur.
//	refusee                     tout le reste, et notamment TOUT RESIDU D UN INSTANT A
//	                            MULTIPLICITE (deux morts ou plus d un cote) que les deux passes
//	                            precedentes n ont pas tranche. Rien n est enrichi, rien n est
//	                            ajoute (la mort est peut-etre deja dans la base : l ajouter la
//	                            compterait deux fois), et l instant se COMPTE. Refuser plutot
//	                            que tirer au sort : cf. [appariement.repliSurLInstantClassique].
//
// CE QUI N EST JAMAIS DECLARE « AUTRE MORT » : une ligne de film sans `victim_xuid`, et une ligne
// de film dont l instant porte une mort de credit elle-meme sans `victim_xuid`. Sans les deux
// identites, « ce n est pas la meme mort » n est pas une mesure mais une supposition — et une
// supposition fausse ajouterait une mort qui n existe pas. Dans le doute, l instant est refuse.
//
// ─── POURQUOI CETTE FONCTION EST PURE, ET SEPAREE DE LA FUSION ─────────────────────────────
//
// Elle ne lit que les deux listes et ne rend qu un verdict par ligne de film : c est ce qui
// permet de la muter dans un test (inverser l appariement) et de voir rougir la fusion sans
// qu aucune base ne soit ouverte.

// verdictFilm : le sort d UNE ligne de film, decide instant par instant.
type verdictFilm uint8

const (
	// filmRefuse : ni enrichissement ni publication. L instant est compte ambigu.
	filmRefuse verdictFilm = iota
	// filmApparie : la ligne enrichit une mort de credit (cf. `appariement.filmPourCredit`).
	filmApparie
	// filmOrphelinInstantLibre : aucune mort de credit a cet instant.
	filmOrphelinInstantLibre
	// filmOrphelinInstantPartage : une mort de credit est a cet instant, mais la victime prouve
	// que ce n est pas la meme mort. C est la population que le lot 2.9 rend visible.
	filmOrphelinInstantPartage
)

// appariement : le resultat d un appariement de passe, et rien d autre — aucune mort n y est
// copiee ni modifiee.
type appariement struct {
	// filmPourCredit : indice de mort de credit -> indice de ligne de film qui l enrichit.
	filmPourCredit map[int]int
	// verdict : une entree par ligne de film, dans l ordre du film.
	verdict []verdictFilm
	// instantsAmbigus : instants ou au moins une ligne de film a ete REFUSEE. Un instant dont
	// toutes les lignes de film sont appariees ou prouvees autres n est PAS ambigu, meme s il
	// porte plusieurs morts : la question y a ete tranchee.
	instantsAmbigus int
}

// apparier : l appariement complet d une passe. Chaque instant est traite independamment, donc
// l ordre de parcours des instants n influe pas sur le resultat.
func apparier(base, film []KillEventInsert) appariement {
	ap := appariement{
		filmPourCredit: make(map[int]int, len(film)),
		verdict:        make([]verdictFilm, len(film)),
	}
	parInstantBase := indexerParInstant(base)
	for t, f := range indexerParInstant(film) {
		b := parInstantBase[t]
		if len(b) == 0 {
			for _, j := range f {
				ap.verdict[j] = filmOrphelinInstantLibre
			}
			continue
		}
		ap.apparierInstant(base, film, b, f)
	}
	return ap
}

// apparierInstant : les trois passes sur UN instant qui porte des morts des deux cotes.
//
// `b` et `f` sont les indices, dans `base` et `film`, des morts de cet instant.
func (ap *appariement) apparierInstant(base, film []KillEventInsert, b, f []int) {
	prisB := make([]bool, len(b))
	prisF := make([]bool, len(f))

	// 1. PAR LA VICTIME — la seule identite que les deux cotes portent pour la meme mort.
	for x, j := range f {
		y := seuleMortDeLaVictime(base, b, prisB, film[j].VictimXUID)
		if y < 0 {
			continue
		}
		ap.filmPourCredit[b[y]] = j
		ap.verdict[j] = filmApparie
		prisB[y], prisF[x] = true, true
	}

	// 2. LES LIGNES DE FILM QUE LA VICTIME PROUVE ETRE UNE AUTRE MORT.
	for x, j := range f {
		if prisF[x] || !autreMortProuvee(base, b, film[j]) {
			continue
		}
		ap.verdict[j] = filmOrphelinInstantPartage
		prisF[x] = true
	}

	// 3. LE REPLI SUR L INSTANT — ET SEULEMENT SUR L INSTANT CLASSIQUE.
	if ap.repliSurLInstantClassique(base, film, b, f, prisB, prisF) {
		return
	}
	// UNE LIGNE DE FILM REFUSEE COMPTE L INSTANT. Une mort de credit sans ligne de film en face,
	// elle, est le cas ordinaire des 25,6 % de morts que le film ne publie pas : elle ne compte
	// rien.
	if nbLibres(prisF) > 0 {
		ap.instantsAmbigus++
	}
}

// repliSurLInstantClassique : LE SEUL CAS OU L INSTANT SEUL APPARIE ENCORE — une mort de credit et
// une ligne de film, EN TOUT, a cet instant, dont l une au moins n a pas de victime resolue.
//
// ─── POURQUOI IL SE COMPTE SUR LE TOTAL DE L INSTANT, ET JAMAIS SUR LES RESIDUS ────────────
//
// La premiere version appariait « ce qu il reste des deux cotes quand il n en reste qu un de
// chaque » : un repli PAR ELIMINATION. Il apparie alors deux lignes qui n ont AUCUNE identite
// commune, et la revue adversariale du lot (2026-09-16, deux relecteurs, sondes rejouees) a
// montre qu il suffit d un instant a deux morts de credit pour qu il se trompe :
//
//	credit [A, B] + film [A, A]    A s apparie, le second A n est ni apparie ni « autre mort »
//	                               (A == A), et l elimination l apparie a B -> victime
//	                               divergente -> LE FILM ENTIER REFUSE, la regression meme que
//	                               ce lot ferme
//	credit [A, B] + film [A, ""]   l elimination donne a B l arme, les parts et l assistant
//	                               d une AUTRE mort — ecrits en base, servis par `_latest`,
//	                               et `AmbiguousInstants` tombe a 0 : RIEN ne le signale
//
// Des que l instant porte une multiplicite d un cote, un residu que les passes 1 et 2 n ont pas
// tranche est donc REFUSE : c est « refuser plutot que tirer au sort », applique a la lettre.
//
// LES DEUX VICTIMES RESOLUES NE PASSENT JAMAIS ICI, et le test le dit explicitement plutot que de
// s en remettre aux passes precedentes : egales, la passe 1 les a appariees ; differentes, la
// passe 2 en a fait une orpheline d instant partage (c est le temoin `9f9b19e5@63757`).
func (ap *appariement) repliSurLInstantClassique(
	base, film []KillEventInsert, b, f []int, prisB, prisF []bool,
) bool {
	if len(b) != 1 || len(f) != 1 || prisB[0] || prisF[0] {
		return false
	}
	if base[b[0]].VictimXUID != "" && film[f[0]].VictimXUID != "" {
		return false
	}
	ap.filmPourCredit[b[0]] = f[0]
	ap.verdict[f[0]] = filmApparie
	return true
}

// seuleMortDeLaVictime : l indice, DANS `b`, de la seule mort de credit libre dont la victime est
// `victime`. Rend -1 quand `victime` est vide, quand aucune ne correspond, ou quand PLUSIEURS
// correspondent — dans ce dernier cas choisir serait tirer au sort.
func seuleMortDeLaVictime(base []KillEventInsert, b []int, prisB []bool, victime string) int {
	if victime == "" {
		return -1
	}
	trouve := -1
	for y, i := range b {
		if prisB[y] || base[i].VictimXUID != victime {
			continue
		}
		if trouve >= 0 {
			return -1
		}
		trouve = y
	}
	return trouve
}

// autreMortProuvee : la ligne de film est-elle, A COUP SUR, une mort que la base ne porte pas ?
//
// Il y faut les DEUX identites : celle de la ligne de film, et celle de TOUTES les morts de
// credit de l instant. Une seule victime non resolue quelque part et la reponse est non — une
// mort ajoutee a tort est une mort qui n a pas eu lieu.
func autreMortProuvee(base []KillEventInsert, b []int, ligne KillEventInsert) bool {
	if ligne.VictimXUID == "" {
		return false
	}
	for _, i := range b {
		if base[i].VictimXUID == "" || base[i].VictimXUID == ligne.VictimXUID {
			return false
		}
	}
	return true
}

// nbLibres : combien d entrees ne sont pas prises. C est le compte qui dit « au moins une ligne de
// film est restee sans reponse », donc que l instant se compte ambigu.
func nbLibres(pris []bool) int {
	n := 0
	for _, p := range pris {
		if !p {
			n++
		}
	}
	return n
}

// indexerParInstant : `time_ms -> indices`, dans l ordre d entree. L instant reste le CADRE de
// l appariement — c est a l interieur d un instant que la victime tranche.
func indexerParInstant(deaths []KillEventInsert) map[int][]int {
	out := make(map[int][]int, len(deaths))
	for i := range deaths {
		out[deaths[i].TimeMS] = append(out[deaths[i].TimeMS], i)
	}
	return out
}
