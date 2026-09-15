package fallback

// registre.go — L'ASSEMBLAGE DU REGISTRE.
//
// # D'OÙ VIENNENT CES ENTRÉES
//
// Trois sources, toutes datées et vérifiées sur pièces au lot 1.9.0 (2026-09-14) :
//
//  1. la table (E) de `.ai/V7.5/AUDIT_HEURISTIQUES_DECODEUR_2026-09-13.md` — les 62 REPLIS
//     ANONYMES du décodeur, recensés par le lot 0.E. Ceux que les lots 1.0 à 1.8 ont convertis
//     ou supprimés depuis N'Y SONT PAS : le recensement du lot 1.9.0 les nomme un par un (§5
//     du plan) ;
//  2. les replis que les lots 0.D à 1.8 ont NOMMÉS en les posant (bijection hongroise du kill
//     feed, garde `contiguousRounds`, châssis de véhicule inconnu, `Registry.Truncated`...) ;
//  3. les lignes de la table (A) et de la table (C) de l'audit que la famille 1.9 désigne
//     nommément comme conversions (fenêtre temporelle des poses, chunk du pied par argmax,
//     `lifeGapUS`, fin de vie de véhicule).
//
// # CE QUI N'Y EST PAS ENCORE, ET POURQUOI
//
// Les 38 lignes de la table (C) de l'audit qui portent un NÉGATIF MESURÉ et ne sont visées par
// aucun item 1.9.x : ce sont des replis LÉGITIMES (le film n'écrit pas le fait, la mesure le
// prouve) déjà nommés et comptés par leur propre couverture — leur entrée au registre serait
// une recopie sans gain de contrôle, et elle diluerait la lecture du seul chiffre qui pilote
// (le nombre de replis à retirer). Consigné en §4 du plan.
//
// # L'ORDRE DE CE FICHIER NE COMPTE PAS
//
// [Table] trie par nom. Le découpage en SIX fichiers (cinq jusqu au lot 1.9.4, qui a
// scindé `registre_killsource.go` à 523 lignes) ne suit que la limite de 500 lignes du dépôt et
// le paquet des sites.

// registre est LA table. Elle ne se modifie jamais à l'exécution : c'est une donnée de code,
// au même titre qu'un catalogue versionné (D12).
var registre = concat(
	registreReplayEquipement,
	registreReplayIdentites,
	registreKillsource,
	registreKillsourceCarte,
	registreObjectifsEtConstruction,
	registreFilmdec,
)

// concat aplatit les tranches déclarées par fichier. Écrite à la main plutôt qu'avec `append`
// en chaîne : une entrée perdue dans un `append` mal parenthésé ne se verrait pas.
func concat(parts ...[]Repli) []Repli {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]Repli, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// dateAudit0E : date de l'audit qui a nommé les 62 replis anonymes pour la première fois. Elle
// sert de DatePose à tous les replis hérités (cf. [Repli.DatePose] : pose = inscription).
const dateAudit0E = "2026-09-13"

// dateVague2 : le jour de la vague 2 de la famille 1.9. Quatre replis y sont posés — un par lot
// de conversion — et `goconst` refuse à juste titre une quatrième occurrence du littéral.
//
// C'EST UNE DATE, PAS UN LOT : elle ne dit RIEN de la cible de retrait ni du critère, que chaque
// entrée porte à part. La constante partagée `lot194`, supprimée le 2026-09-15, avait justement
// le défaut inverse — elle nommait un LOT, et ses quatre citations ont divergé.
const dateVague2 = "2026-09-15"

// comptageFamille19 : la raison, écrite une fois, pour laquelle le compteur d'un repli hérité
// n'est pas câblé au lot 1.9.0.
//
// LE CÂBLAGE SUIT LA CONVERSION, ET CE N'EST PAS UN REPORT. Le compteur d'un repli doit
// atteindre la couverture du document, donc traverser la chaîne d'appel de la cuisson ; sur la
// grande majorité des sites cette chaîne passe par des fonctions pures déjà à cinq paramètres
// (limite du dépôt) que le pas 2 de M2 — « les lecteurs reçoivent le profil, famille par
// famille » — va de toute façon retoucher. Câbler maintenant, puis re-câbler au pas 2, serait
// de la dette payée deux fois ; câbler au moment où le lot de conversion ouvre déjà le fichier
// coûte une ligne. Le registre le dit entrée par entrée ([Repli.CibleComptage]) pour qu'un
// compte absent ne se lise jamais comme un compte nul.
const comptageFamille19 = "lot de conversion 1.9.x du fait, ou pas 2 de M2 (porteur du profil)"

// LA CONSTANTE `lot194` A ÉTÉ SUPPRIMÉE LE 2026-09-15, AVEC LE LOT QU'ELLE NOMMAIT. Quatre
// entrées la citaient comme cible de retrait ou de comptage ; le lot 1.9.4 est fait — la carte
// d'un film vient de son nom de match et non plus d'une signature de largeurs d'axe — et chacune
// des quatre porte désormais SA propre cible, distincte des trois autres. Une constante partagée
// par des entrées dont les cibles ont divergé mentirait sur ce qui reste à faire.
