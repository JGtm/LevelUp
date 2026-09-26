package profile

// identite.go — LA SECTION 2 DE `chunk_00`, COMME VALEUR.
//
// EXTRAITE DE `grammar/film_identity.go` AU LOT 2.5.b : le TYPE descend, le LECTEUR
// (`ReadFilmIdentity`, ses ancres, ses erreurs sentinelles de chunk tronque) reste en
// `grammar` et rend desormais un [FilmIdentity] de ce paquet.
//
// POURQUOI ELLE EST UNE VALEUR DE PROFIL : elle porte les DEUX cles que le film ecrit en clair
// — le nom de build et la version de format — et c est sur elles que la table par build et la
// table par format se lisent ([Resoudre]). Le champ [Profile.Identity] est de ce type ; le
// laisser en `grammar` aurait fait remonter `profile` vers `grammar`.

// FilmIdentity : la section 2 de `chunk_00`, lue champ par champ.
type FilmIdentity struct {
	// Version, Build, Flavor : les trois champs de 32 octets, en clair.
	Version string
	Build   string
	Flavor  string
	// BuildID, Changelist : les deux u32 de `0x0CB454` / `0x0CB458`, recopies par
	// `FUN_14299b674` depuis la structure d'infos de build de l'executable.
	BuildID    uint32
	Changelist uint32
	// MatchStartUnix : l'horodatage du match, `_time64()` au moment ou `chunk_00` est ecrit.
	// Mesure du 2026-09-12 contre `match_registry.start_time_utc` : +19 s, +29 s, +43 s sur
	// trois films (seuil de 120 s ecrit avant la mesure), et un seul decalage de bit sur
	// dix-sept rend une valeur plausible — celui que l'ecrivain predit.
	MatchStartUnix uint32
	// FormatVersion : la VERSION DE FORMAT de `chunk_00`, le u32 de `base+4`. C'est elle qui
	// commande la largeur du registre et celle de la table par type chez le LECTEUR du jeu
	// (`FUN_14299ab50`) — cf. `film_format_version.go`. Elle est renseignee ICI par commodite,
	// mais elle ne DEPEND PAS de cette section : [FilmFormatVersionFromHeader] la rend aussi
	// sur les cinq films du cache qui n'ont pas de section d'identification (format 20).
	FormatVersion int
	// TypeVersions : la table par type, les u32 qui precedent le champ de version. Le cardinal
	// suit la VERSION DE FORMAT (123 / 122 / 121 / 116 mesures pour les formats 27 / 25 / 24 /
	// 21). Leur semantique est desormais ETABLIE et non plus « appuyee » : `FUN_1428e1c64` les
	// lit indexees (`film+0xCB208 + i*4`, quatre sites d'instruction sur 13,6 M), son unique
	// appelant `FUN_141102ed0(i)` rend la version du TYPE i — celle du film en mode Theater, la
	// native `DAT_14474cd90` sinon — et QUINZE fonctions s'en servent pour brancher.
	//
	// CE LECTEUR NE LES INTERPRETE PAS, ET IL NE FAUT PAS LEUR DEMANDER LA GRAMMAIRE DU
	// DECODEUR : mesure du 2026-09-15 sur les sept bobines, la table est alignee PAR LE DEBUT
	// (l'index 18 vaut 2 sur les sept, comme la table native), 24 index varient d'une bobine a
	// l'autre, et AUCUN des neuf index que l'ecrivain interroge en clair (0x23, 0x24, 0x30,
	// 0x59, 0x5a, 0x5b, 0x5d, 0x61, 0x72) ne separe les films `8/3` des films `9/5`. Le
	// discriminant est [FilmIdentity.FormatVersion], pas cette table.
	TypeVersions []uint32
	// RegistryBlocks : le nombre de blocs du registre (49 ou 50 selon le build). C'est lui qui
	// ancre la fin du registre, donc le debut de la table par type.
	RegistryBlocks int
	// RegistryFingerprint : le FNV-1a 64 bits des entrees NOMMEES du registre — l identite de la
	// grammaire des composants du film. Zero quand le registre n a pas ete lu.
	//
	// ELLE ETAIT CALCULEE PUIS JETEE (decouverte D4 (3.2), 2026-09-16) : le lecteur de cette
	// section re-parse le registre entier pour trouver l ancre de la chaine de build, donc il
	// tient l empreinte sous la main. La rendre ne coute pas un cycle, et sans elle le decodeur
	// ne pouvait pas DIRE sous quelle grammaire de composants un artefact a ete cuit — seul un
	// avertissement de journal, dedupliqué par processus, en portait la trace.
	RegistryFingerprint uint64
	// RegistryNamedSlots : le nombre d entrees NOMMEES hachees — le denominateur de l empreinte.
	// Sans lui, deux empreintes differentes ne se distinguent pas d un registre tronque.
	RegistryNamedSlots int
	// BuildOffset : l'octet de la chaine de build dans le tampon inflate. Publie parce que
	// c'est l'ancre de toute la derivation, et qu'un rapport qui le porte se relit.
	BuildOffset int
	// BodyBit : le PREMIER BIT du corps (`FUN_1407ec560`), decalage d'un bit compris. C'est la
	// borne basse de tout balayage du corps — la table des joueurs en particulier.
	BodyBit int
	// ControleDeCorruption : LE BIT DE `base+0x0CB45C`, ET CE QU IL COMMANDE (lot 5.18.1).
	//
	// C est le booleen d un bit que la carte de `film_identity.go` portait sans nom depuis le
	// lot 1.5.1 — celui qui decale d un bit tout ce qui le suit. Sa VALEUR est lue chez
	// l ecrivain, et elle commande la grammaire des corps a composants :
	//
	//	ECRITURE  `FUN_14299b198` @14299b25b : `W(1)` de `film+0xCB45C` (`FUN_1406d49c4`),
	//	          apres le buildID et la changelist, avant les deux champs de nom.
	//	LECTURE   `FUN_14299ab50` @14299ac28 : `R(1)` (`FUN_1406cf008`) range en `film+0xCB45C`.
	//	REPORT    `FUN_1428e219c` @1428e2239 : `*(char *)(singleton + 0x1AE) = film[0xCB45C]`,
	//	          sous la garde `*film == 0x29` (la version MAJEURE du film = celle du build).
	//	USAGE     `FUN_14076cea8()` rend `DAT_144c23326` (= `DAT_144c23178 + 0x1AE`) en rejeu de
	//	          film, et `FUN_14076cb60` s en sert comme `extra` : apres CHAQUE composant
	//	          present, un `R(1)` de garde et, si ce bit vaut 1, un `R(32)` sentinelle
	//	          `0x0bcddcba` (« entity component corrupt »). Idem `FUN_142e2c690` sur le
	//	          chemin d etat complet.
	//
	// C est donc un drapeau de GRAMMAIRE, pas une metadonnee : leve, il coute au moins un bit
	// par composant present. Le depot le lisait a `false` par defaut (`GrammaireBalayage`) ;
	// depuis le lot 5.18.2 il vient d ICI, donc du film (ADR 0034 : le film est autoportant).
	ControleDeCorruption bool
}
