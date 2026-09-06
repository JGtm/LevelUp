package domain

// tactical_raster.go — LE SIDECAR DE RASTER TACTIQUE : l'occupation d'UN match, calculee
// UNE FOIS a la cuisson et relue telle quelle par la page.
//
// # POURQUOI UN FICHIER PAR MATCH, ET AUCUN CACHE D'AGREGAT
//
// Decision produit du plan tactique (2026-09-05) : « un raster par match, calcule une
// fois a la cuisson ; la page en somme N ; aucun cache d'agregat, aucune invalidation ».
// Un cache par filtre aurait autant d'entrees que de combinaisons de la barre L2 — et
// chacune aurait fallu l'invalider a chaque nouveau match. Un fichier par match est
// IMMUABLE : il ne depend d'aucun filtre, il ne perime que si son artefact est re-cuit.
//
// # CE QUI REND LA SOMME POSSIBLE
//
// Les cellules sont adressees en indices ENTIERS ancres sur l'ORIGINE DU MONDE (cf.
// analysis/tactical.Grille). Deux rasters de matchs differents nomment donc la meme
// cellule pareil et se somment SANS RE-PROJECTION — c'est l'invariant sur lequel repose
// tout ce dispositif, et c'est pour lui que `pas_m` voyage dans le fichier : sommer deux
// rasters de pas differents melangerait deux resolutions.
//
// # CE QUI N'EST PAS DEDANS
//
// Aucun nom, aucune equipe, aucun resultat de match : le sidecar est ANONYME (xuid seul)
// et sans contexte. L'axe « qui » (moi / escouade / adversaires), l'univers des matchs
// retenus et le denominateur « par match » sont resolus a la LECTURE, par la base. Un
// sidecar qui porterait des equipes serait perime des qu'un joueur change de camp, et il
// faudrait le re-cuire pour une raison qui n'a rien a voir avec le film.

// TacticalRasterSchemaVersion est la version du FORMAT du sidecar. Un sidecar d'une autre
// version est ignore a la lecture et re-ecrit par le rattrapage.
//
// ELLE COUVRE AUSSI LE PAS D'ECHANTILLONNAGE : `echantillons` est un COMPTE, et sa
// traduction en secondes depend de ce pas. Changer analysis/tactical.PasOccupationMs sans
// incrementer cette version rendrait tous les sidecars du parc silencieusement faux d'un
// facteur — d'ou le champ PasEchantillonMs ci-dessous, que la lecture verifie.
// # LA LACUNE RESIDUELLE, ECRITE PLUTOT QUE TUE
//
// Le temps passe en VEHICULE est attribue par les episodes d'occupation du document
// (`replay.VehicleRide`), et cette primitive n'attribue que **15,6 a 21,1 % des vies de
// vehicule** (limite mesuree et publiee dans `analysis/replay/document_vehicles.go`, doc
// de `VehicleTrack.Rides`). Le reste du temps embarque n'est donc PAS mesure : il
// n'apparait ni dans les cellules, ni au denominateur. Une lecture « ou je passe mon
// temps » sous-estime le temps en vehicule, et c'est une propriete connue de la mesure —
// pas un defaut a chercher. Elle ne peut pas se corriger ici : elle se corrigerait en
// amont, dans la primitive d'attribution des episodes.
const TacticalRasterSchemaVersion = 2

// TacticalRasterSidecar est le fichier depose a cote de l'artefact
// (title.PathResolver.TacticalRasterPath).
type TacticalRasterSidecar struct {
	SchemaVersion int `json:"schema_version"`

	// MatchID est l'identifiant COMPLET, tel que l'artefact le porte ; ShortID est la
	// cle courte sous laquelle les deux fichiers sont ranges. Les deux sont ecrits parce
	// qu'ils ne se deduisent pas l'un de l'autre : le nom du fichier ne rend pas le
	// match_id complet, et le rattrapage part d'un simple listing de dossier.
	MatchID string `json:"match_id"`
	ShortID string `json:"short_id"`

	// ArtifactSchemaVersion est le schema de L'ARTEFACT PROJETE. C'est la cle de
	// fraicheur du rattrapage : un artefact re-cuit change de schema, et le sidecar qui
	// ne le suit plus doit etre refait. Elle dit aussi, en clair, de quelle qualite de
	// decodage vient la mesure.
	ArtifactSchemaVersion int `json:"artifact_schema_version"`

	// PasM est le pas de la grille en metres (0,5). Deux sidecars de pas differents ne
	// se somment pas — la lecture les refuse plutot que de melanger deux resolutions.
	PasM float64 `json:"pas_m"`

	// FrameIntervalMs est l'echelle de l'axe de temps de l'artefact, en millisecondes.
	// Publie pour que les frames des spawns et des premieres entrees restent
	// interpretables sans rouvrir l'artefact.
	FrameIntervalMs int `json:"frame_interval_ms"`

	// PasEchantillonMs est le pas de reechantillonnage (250 ms) qui a produit les
	// comptes. C'est L'UNITE de `echantillons` : sans lui, un compte de 8 ne dit pas
	// s'il vaut 2 s ou 4 s. Verifie a la lecture.
	PasEchantillonMs int `json:"pas_echantillon_ms"`

	// PointsIgnores est le nombre d'echantillons ECARTES faute de position finie (NaN /
	// Inf), comptes A LA CUISSON.
	//
	// IL VOYAGE DANS LE FICHIER PARCE QU'IL N'EXISTE PLUS APRES. La lecture somme des
	// comptes deja groupes PAR CELLULE, et un point ecarte n'a jamais eu de cellule : il
	// ne peut donc pas se retrouver dans l'agregat. Sans ce champ, la reponse aurait
	// publie 0 point ignore quoi qu'il arrive, et un decodage qui derape se serait tu.
	PointsIgnores int `json:"points_ignores"`

	// Joueurs est trie par xuid. VIDE mais PRESENT quand l'artefact n'a aucune piste
	// nommee : le fichier existe, donc le match est MESURE et il est mesure a zero — ce
	// qui ne se confond pas avec « pas de sidecar », qui veut dire non mesure.
	Joueurs []TacticalRasterJoueur `json:"joueurs"`
}

// TacticalRasterJoueur est ce qu'un joueur a produit sur ce match.
type TacticalRasterJoueur struct {
	XUID string `json:"xuid"`

	// Cellules : par cellule atteinte, le nombre d'echantillons — donc le temps passe,
	// en unites de PasEchantillonMs. Trie par colonne puis ligne.
	Cellules []TacticalRasterCellule `json:"cellules"`

	// Spawns : le premier point de chacune de ses vies, trie par frame. C'est la matiere
	// des grappes de reapparition et des routes de sortie de spawn (phase 7).
	Spawns []TacticalRasterSpawn `json:"spawns"`

	// PremieresEntrees : par cellule, la frame de la premiere fois. C'est l'INSTANT
	// CONTRIBUTEUR d'une lecture d'occupation — ce que le clic sur une cellule ouvre
	// dans le rejeu 2D.
	PremieresEntrees []TacticalRasterEntree `json:"premieres_entrees"`
}

// TacticalRasterCellule est le temps passe dans une cellule, compte en echantillons.
type TacticalRasterCellule struct {
	Col          int `json:"col"`
	Lig          int `json:"lig"`
	Echantillons int `json:"echantillons"`
}

// TacticalRasterSpawn est le premier point d'une vie.
type TacticalRasterSpawn struct {
	Frame int     `json:"frame"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// TacticalRasterEntree est la premiere entree dans une cellule.
type TacticalRasterEntree struct {
	Col   int `json:"col"`
	Lig   int `json:"lig"`
	Frame int `json:"frame"`
}

// TacticalQuestionTemps : OU JE PASSE MON TEMPS — la lecture d'OCCUPATION, sommee depuis
// les sidecars ci-dessus.
//
// POURQUOI ELLE VIT ICI ET NON A COTE DE SES TROIS SOEURS (domain/tactical.go). D'abord
// parce qu'elle n'a pas le meme SUBSTRAT : morts, kills et gagne se lisent tous sur
// `kill_positions_latest` — le seul substrat qui existe sans artefact —, celle-ci se lit
// sur les pistes du film. Ensuite parce que `domain/tactical.go` est deja a 504 lignes :
// la dette de seuil est GELEE par la baseline, on ne l'accroit pas (CLAUDE.md n 5). Le
// vocabulaire complet des questions se lit donc en deux endroits, et chacun dit d'ou sa
// mesure vient.
//
// L'UNITE EST LA SECONDE PAR MATCH : `CelluleTactique.Valeur` porte des secondes,
// `Brut` le compte d'echantillons qui les a produites.
const TacticalQuestionTemps = "temps"
