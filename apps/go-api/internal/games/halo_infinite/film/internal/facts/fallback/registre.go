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
// [Table] trie par nom. Le découpage en fichiers (cinq jusqu au lot 1.9.4, qui a scindé
// `registre_killsource.go` à 523 lignes ; `registre_replay_positions.go` né au lot M1 des retours
// du rejeu, `registre_replay_places.go` au lot M2.3, 2026-09-23, `registre_replay_objectifs.go`
// au lot J5.5 du plan de suite d'audit, 2026-09-27, `registre_filmdec_marche.go` et
// `registre_killsource_collecteur.go` au lot J8.7, le même jour) ne suit que la limite de 500
// lignes du dépôt et le paquet des sites. Un fichier de plus s'ajoute à [Tranches], et à rien
// d'autre.

// Tranche est une famille du registre : son nom de lecture et les entrées qu'elle porte.
//
// ELLE EST EXPORTÉE PARCE QUE LE TEST QUI LA VÉRIFIE DOIT LIRE LA MÊME LISTE QUE L'ASSEMBLAGE.
// Constat de la revue de jalon M1 (2026-09-15) : le test des familles en énumérait CINQ à la
// main quand l'assemblage en concaténait SIX — `registreKillsourceCarte`, ajoutée au lot 1.9.4,
// n'était vérifiée par rien, et la septième ne l'aurait pas été davantage. Une liste recopiée à
// côté de celle qui décide finit toujours par diverger ; il n'y en a donc plus qu'une.
type Tranche struct {
	// Nom : le nom de lecture de la famille, tel que le rapport l'affiche.
	Nom string
	// Replis : les entrées déclarées par son fichier.
	Replis []Repli
}

// Tranches rend les familles du registre, dans l'ordre des fichiers. C'est LA source : [registre]
// en découle, et le test des familles la parcourt.
func Tranches() []Tranche {
	return []Tranche{
		{"replay/equipement", registreReplayEquipement},
		{"replay/identites", registreReplayIdentites},
		{"replay/places", registreReplayPlaces},
		{"killsource", registreKillsource},
		{"killsource/carte", registreKillsourceCarte},
		{"killsource/calibration", registreKillsourceCalibration},
		{"killsource/collecteur", registreKillsourceCollecteur},
		{"objectifs et construction", registreObjectifsEtConstruction},
		{"filmdec", registreFilmdec},
		{"filmdec/marche", registreFilmdecMarche},
		{"replay/positions", registreReplayPositions},
		{"replay/vehicules", registreReplayVehicules},
		{"replay/objectifs", registreReplayObjectifs},
	}
}

// registre est LA table. Elle ne se modifie jamais à l'exécution : c'est une donnée de code,
// au même titre qu'un catalogue versionné (D12).
var registre = concat(Tranches())

// concat aplatit les tranches déclarées par fichier. Écrite à la main plutôt qu'avec `append`
// en chaîne : une entrée perdue dans un `append` mal parenthésé ne se verrait pas.
func concat(parts []Tranche) []Repli {
	n := 0
	for _, p := range parts {
		n += len(p.Replis)
	}
	out := make([]Repli, 0, n)
	for _, p := range parts {
		out = append(out, p.Replis...)
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

// dateM3 : le jour des lots de M3 qui posent des replis (3.3.1 et 3.4.1). QUATRE entrees y sont
// nees — deux par lot — et `goconst` refuse a juste titre une quatrieme occurrence du litteral.
//
// C EST UNE DATE, PAS UN LOT, exactement comme [dateVague2] : elle ne dit RIEN de la cible de
// retrait ni du critere, que chaque entree porte a part.
const dateM3 = "2026-09-17"

// LA CONSTANTE `lot194` A ÉTÉ SUPPRIMÉE LE 2026-09-15, AVEC LE LOT QU'ELLE NOMMAIT. Quatre
// entrées la citaient comme cible de retrait ou de comptage ; le lot 1.9.4 est fait — la carte
// d'un film vient de son nom de match et non plus d'une signature de largeurs d'axe — et chacune
// des quatre porte désormais SA propre cible, distincte des trois autres. Une constante partagée
// par des entrées dont les cibles ont divergé mentirait sur ce qui reste à faire.

// LES CONSTANTES `comptageFamille19` ET `comptageCollecteur` ONT ÉTÉ SUPPRIMÉES LE 2026-09-27 (lot
// J8.7) avec le dernier compteur qu'elles différaient : les replis hérités se comptent en données
// (`grammar`, `killsource`, `objectives`) ou au site (`replaybuild`, `sync/killcollector`).

// retraitRegle4 : LA CIBLE DE RETRAIT DE LA REGLE 4 DE D-10, ecrite une fois (lot J8.7, 2026-09-27,
// decision 7 du superviseur). Elle remplace, dans chaque [Repli.CibleRetrait], le lot ou le jalon d un
// plan CLOS (PLAN_DECODEUR_FILM, cloture du 2026-09-18 ; campagne des retours du rejeu) que la cible
// nommait : la condition de lecture qui retirerait le repli reste ecrite devant elle, et a defaut le
// compte decide — un repli nul au corpus gate de J11 sort au jalon suivant. Le garde-rail
// `archlint/no_stale_fallback_target_test.go` refuse toute cible qui nommerait de nouveau un lot clos.
const retraitRegle4 = "retrait au jalon suivant si le compte est nul au corpus gate de J11 (regle 4 de D-10, 2026-09-27)"

// fichierDeVersement : LA TABLE de `replay` qui verse au compteur de la cuisson les comptes de replis
// que `grammar`, `profile`, `killsource` et `objectives` rendent EN DONNEES (lot J8.7, 2026-09-27).
const fichierDeVersement = "internal/games/halo_infinite/film/replay/versement_des_replis.go"

// siteDeVersement rend le site d un repli verse par cette table : son ancre est la LIGNE de la table,
// `{fallback.NomX,`. La direction (C) d `archlint` relit les noms de la table comme des declenchements,
// et exige ce site ; retirer la ligne fait rougir la direction (E) (`fallback_versement_test.go`).
func siteDeVersement(constante string) Site {
	return Site{Fichier: fichierDeVersement, Ancre: "{fallback." + constante + ","}
}

// date0927 : le jour des jalons J5 et J8 du plan de suite d audit, ou plusieurs replis sont nes ;
// `goconst` refuse a juste titre une cinquieme occurrence du litteral. C EST UNE DATE, PAS UN LOT,
// comme [dateVague2] et [dateM3].
const date0927 = "2026-09-27"

// date1007 : le jour ou naissent le rattrapage des kills de la representation intermediaire (lot
// 2.7.c4) et les replis de la lecture des zones par le NOM de propriete `ti=13` (proprietaire,
// pousseur, designateur et proprietaire de colline) ; `goconst` refuse a juste titre une quatrieme
// occurrence du litteral. C EST UNE DATE, PAS UN LOT, comme [dateVague2], [dateM3] et [date0927].
const date1007 = "2026-10-07"
