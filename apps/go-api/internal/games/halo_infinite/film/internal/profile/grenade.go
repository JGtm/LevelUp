package profile

// grenade.go — L AMORCE DU RECORD DE CREATION DE PROJECTILE, PAR CLE ECRITE DANS LE FILM.
//
// # CE QUE CETTE TABLE POSE, ET POURQUOI ELLE N EST PAS UNE LISTE BLANCHE
//
// Un lancer de grenade se lit dans un paquet delta comme la NAISSANCE d une entite de
// l archetype projectile : `[typeIndex : 6 bits][amorce de l etat par defaut][identifiant de tag
// : 32 bits]...[index du lanceur : 5 bits]`. Le « marqueur » historique `0x4C0C00` n est pas un
// motif de recherche : ce sont les CINQ bits bas du typeIndex suivis de l amorce constante de
// l etat par defaut (`grammar/projectiles.go`, point 2).
//
// Le volet RECHERCHE du lot 3.3 (note `.ai/V7.5/film_re/NOTE_3_3_IDENTIFIANTS_GRENADE_2026-09-16.md`,
// decouverte D3 (3.3r), 2026-09-17) a etabli que ni le nombre, ni l ordre, ni les valeurs des
// identifiants de grenade n ont change depuis la sortie du jeu : `GrenadeTypeIDsByRank` reste une
// constante du TITRE. Ce qui varie d un build a l autre est la GRAMMAIRE, et elle tient en trois
// nombres — la largeur de l amorce, sa valeur, et la position du champ d index de l auteur.
//
// # LES TROIS NOMBRES, MESURES SUR DIX FILMS (2026-09-17, lot 3.3.1)
//
//	cle                        amorce          identifiant  index auteur
//	build >= HI_1_12_0         24 bits 0x40C00 a +24        a +103
//	build HI_1_8_0..HI_1_11_0  23 bits 0x20600 a +23        a +100
//	build HI_1_4_1, majeure 33 23 bits 0x20600 a +23        a  +99
//	majeure 31                 23 bits 0x20400 a +23        a  +99
//
// Les positions sont comptees depuis le DEBUT du motif d amorce, c est-a-dire depuis le second
// des six bits de typeIndex — le repere historique de `grammar/grenade_events.go`.
//
// # LA MESURE, ET CE QUI LA REND REFUTABLE
//
// AMORCE : on part des occurrences de l identifiant (connu, constante du titre) et on lit CE QUI
// LES PRECEDE, largeur par largeur. Pour la bonne largeur la valeur lue est CONSTANTE — c est la
// definition d une amorce — et pour une largeur de plus elle se disperse. Sur les dix films la
// coupure est nette (137/137, 386/386, 95/95 ... puis 48/95 a la largeur suivante).
//
// INDEX AUTEUR : le critere « toutes les valeurs dans 0..7 » ne tranche PAS (30 decalages sur 81
// le satisfont sur `60ae07c4`) parce que les suites de zeros de l etat par defaut le satisfont
// aussi. Le critere fort est la DISPERSION rapportee a l effectif du match : la position juste
// rend autant de valeurs distinctes qu il y a de lanceurs, et un MAXIMUM egal au dernier index du
// roster. Mesure : `bf15f7ab` (arene, 8 joueurs) 8 distincts / max 7 ; `111fa685` et `084a804d`
// (BTB, 24 joueurs) 24 distincts / max 23 ; `60ae07c4` (arene) 8 / 7. Une suite de zeros rend UN
// distinct, et un decalage quelconque rend 32 distincts / max 31 : les trois signatures ne se
// confondent pas.

// AmorceGrenade porte la grammaire du record de creation de projectile pour UNE cle de film.
//
// Elle est une DONNEE, pas une mesure : rien ici ne se lit sur le film au moment du decodage —
// ce qui se lit sur le film, ce sont les CLES et le typeIndex de l archetype projectile.
type AmorceGrenade struct {
	// Bits est la largeur TOTALE du motif d amorce (cinq bits bas de typeIndex compris) : 23 ou
	// 24. C est aussi la position de l identifiant, en bits depuis le debut du motif.
	Bits int
	// EtatParDefaut est l amorce constante de l etat par defaut qui SUIT les cinq bits bas du
	// typeIndex. Sa largeur vaut `Bits - 5`.
	//
	// ELLE EST UNE DONNEE ET NON UNE TRONCATURE DE LA VALEUR DE REFERENCE : la majeure 31 porte
	// `0x20400` la ou toutes les autres cles anterieures a `HI_1_12_0` portent `0x20600`, a
	// largeur EGALE. Deriver l amorce ancienne de l amorce recente par un decalage d un bit
	// marcherait sur huit cles sur neuf et raterait la neuvieme.
	EtatParDefaut uint32
	// IndexAuteurBit est la position du champ d index du lanceur, en bits depuis le debut du
	// motif. Le champ fait [AmorceGrenadeIndexBits] bits.
	IndexAuteurBit int
	// Connue dit si la cle est dans la table. Faux = profil de reference applique, et le
	// declenchement est un repli NOMME (cf. [AmorceGrenadeDeReference]).
	Connue bool
}

// AmorceGrenadeIndexBits est la largeur du champ d index du lanceur. Cinq bits : un film porte
// jusqu a 24 joueurs indexes 0..23 (BTB), et le champ en couvre 32.
const AmorceGrenadeIndexBits = 5

// AmorceGrenadeTypeIndexBits est le nombre de bits de typeIndex portes EN TETE du motif : les
// cinq de poids faible, le sixieme vivant juste avant (cf. `grammar/grenade_events.go`).
//
// IL VAUT CINQ COMME [AmorceGrenadeIndexBits], ET CE N EST PAS LA MEME GRANDEUR : l un est la
// largeur d un champ d index de joueur, l autre le nombre de bits d archetype que le motif
// recouvre. Les nommer separement evite qu une correction de l un deplace l autre.
const AmorceGrenadeTypeIndexBits = 5

// StructureBits rend la longueur de la structure exploitee : le motif, l identifiant de 32 bits,
// le bourrage, et le champ d index. C est la borne du balayage.
func (a AmorceGrenade) StructureBits() int { return a.IndexAuteurBit + AmorceGrenadeIndexBits }

// MarqueurDe rend le motif d amorce a chercher pour un archetype de projectile donne : ses cinq
// bits bas, puis l amorce de l etat par defaut.
//
// LE TYPEINDEX SE LIT DANS LE REGISTRE DU FILM, il ne se cable pas (`grammar/projectiles.go` le
// dit de sa propre constante : « c est un index de build, pas une constante du format »). La
// mesure 0 du volet recherche l a trouve a 41 sur les SEPT builds du cache, resolu par les quatre
// noms de composant a chaque fois.
func (a AmorceGrenade) MarqueurDe(typeIndex int) uint64 {
	bas := uint64(typeIndex) & (1<<AmorceGrenadeTypeIndexBits - 1)
	return bas<<uint(a.Bits-AmorceGrenadeTypeIndexBits) | uint64(a.EtatParDefaut)
}

// LES QUATRE GRAMMAIRES MESUREES, nommees une fois. Neuf cles de film les partagent.
var (
	// amorceRecente : builds `HI_1_12_0` et `HI_1_13_0` — la grammaire de production depuis
	// toujours. Amorce de 19 bits apres les cinq du typeIndex, motif `0x4C0C00` pour ti=41.
	amorceRecente = AmorceGrenade{Bits: 24, EtatParDefaut: 0x40C00, IndexAuteurBit: 103, Connue: true}
	// amorceAncienne : builds `HI_1_8_0` a `HI_1_11_0`. Amorce de 18 bits, motif `0x260600`.
	amorceAncienne = AmorceGrenade{Bits: 23, EtatParDefaut: 0x20600, IndexAuteurBit: 100, Connue: true}
	// amorceMajeure33 : `HI_1_4_1` et les films de majeure 33 sans section d identification.
	// MEME amorce que [amorceAncienne], index de l auteur UN BIT plus tot.
	amorceMajeure33 = AmorceGrenade{Bits: 23, EtatParDefaut: 0x20600, IndexAuteurBit: 99, Connue: true}
	// amorceMajeure31 : la plus ancienne grammaire du cache. L amorce n a pas la meme VALEUR.
	amorceMajeure31 = AmorceGrenade{Bits: 23, EtatParDefaut: 0x20400, IndexAuteurBit: 99, Connue: true}
)

// AmorceGrenadeDeReference est la grammaire appliquee a une cle ABSENTE de la table.
//
// REPLI NOMME `repli_amorce_grenade_profil_de_reference` (registre `facts/fallback`, D14) : un
// build futur que ce depot ne connait pas encore se lit sous la grammaire du build de reference,
// qui est celle de tous les films recents du parc. Le refuser eteindrait les lancers au premier
// patch du jeu ; le nommer et le compter est ce qui empeche le repli de devenir invisible.
func AmorceGrenadeDeReference() AmorceGrenade {
	a := amorceRecente
	a.Connue = false
	return a
}

// AmorceGrenadePour rend la grammaire du record de creation de projectile pour les cles d un
// film : le BUILD de la section 2 quand il est ecrit, la VERSION MAJEURE de `chunk_00+0` sinon.
//
// UN COMMENTAIRE DE PROVENANCE PAR LIGNE, comme pour [PersonnalisationOctets] : une valeur sans
// preuve n est pas une valeur de profil (D-3 d ADR 0034).
//
// LA SECONDE CLE N EST PAS UN REPLI SUR LA PREMIERE : les cinq films du cache dont `chunk_00` ne
// porte aucune section d identification n ont PAS de build, et la version majeure est la seule
// cle qu ils ecrivent. C est la meme regle que la table des empreintes de registre du lot 3.2.1,
// qui porte neuf clefs pour sept builds.
func AmorceGrenadePour(build string, majeure int) (AmorceGrenade, bool) {
	if build != "" {
		return amorceParBuild(build)
	}
	return amorceParMajeure(majeure)
}

// amorceParBuild : la table keyee par le nom de build de la section 2.
func amorceParBuild(build string) (AmorceGrenade, bool) {
	switch build {
	case buildHI1131:
		// MESUREE sur `bf15f7ab` : 137 lancers, amorce constante sur 25 bits depuis le debut du
		// typeIndex (`0x14C0C00` = [41][0x40C00]) et dispersion de l index a +103 de 8 valeurs
		// distinctes, maximum 7 — les huit joueurs d un match d arene, sans exception.
		return amorceRecente, true
	case buildHI1120:
		// MESUREE sur `bcb6d393` : 55 lancers, soit EXACTEMENT le `grenades.available` de son
		// artefact cuit ; meme amorce constante `0x14C0C00` sur 55/55, index a +103 (54 sur 55
		// dans 0..7). Controle negatif : le motif impair `0x4C0C01` y compte 67 occurrences et
		// AUCUN identifiant reconnu — la grammaire a 24 bits est bien la bonne sur ce build.
		return amorceRecente, true
	case buildHI1110:
		// MESUREE sur `e5adf7b2` : 138 lancers la ou l artefact en publie zero ; amorce
		// constante `0xA60600` sur 138/138 et dispersion de l index a +100 de 23 valeurs
		// distinctes, maximum 23 — vingt-trois des vingt-quatre joueurs d un BTB.
		return amorceAncienne, true
	case buildHI1100:
		// MESUREE sur DEUX films du meme build, ce qui est le controle de reproductibilite :
		// `111fa685` 159 lancers et `084a804d` 289, amorce `0xA60600` sur 159/159 et 289/289,
		// index a +100 avec 24 valeurs distinctes et maximum 23 SUR LES DEUX.
		return amorceAncienne, true
	case buildHI190:
		// MESUREE sur `11de8353`, unique film de ce build au cache : 145 lancers, amorce
		// `0xA60600` sur 145/145, index a +100 avec 21 valeurs distinctes et maximum 22.
		// L index est celui de ses voisins, et le registre le dit : l empreinte de registre de
		// `HI_1_9_0` est celle de `HI_1_8_0` (D2 (3.2)), dont l index est mesure sans ambiguite.
		return amorceAncienne, true
	case buildHI180:
		// MESUREE sur `60ae07c4` : 310 lancers, amorce `0xA60600` sur 310/310, et l index a +100
		// rend 310/310 dans 0..7 avec 8 valeurs distinctes, maximum 7 — le film est un
		// Ranked:Oddball a huit joueurs, donc la mesure ferme sur l effectif exact.
		return amorceAncienne, true
	case buildHI141:
		// MESUREE sur `a521164d`, unique film de ce build au cache : 105 lancers, amorce
		// `0xA60600` sur 105/105, index a +99 avec 22 valeurs distinctes et maximum 23. LA
		// POSITION DE L INDEX EST UN BIT PLUS TOT QUE SUR `HI_1_8_0` : ce n est pas une
		// approximation, les deux decalages sont mesures separement et le voisin (+100) rend ici
		// 12 valeurs distinctes de maximum 11, la signature d une lecture decalee.
		return amorceMajeure33, true
	}
	return AmorceGrenade{}, false
}

// amorceParMajeure : la table des films dont `chunk_00` ne porte AUCUNE section d identification.
func amorceParMajeure(majeure int) (AmorceGrenade, bool) {
	switch majeure {
	case majeureSansSection33:
		// MESUREE sur `a349fea8` : 386 lancers la ou l artefact en publie zero ; amorce
		// `0xA60600` sur 386/386, index a +99 avec 23 valeurs distinctes et maximum 23. La
		// grammaire est celle de `HI_1_4_1`, ce que le registre disait deja : les deux clefs
		// partagent leur empreinte (D2 (3.2)).
		return amorceMajeure33, true
	case majeureSansSection31:
		// MESUREE sur `50247b26` : 95 lancers, et c est la cle qui a OBLIGE a mesurer la VALEUR
		// de l amorce au lieu de la deriver. Sous le motif `0x260600` ce film rend TREIZE
		// marqueurs sur 22 Mio de flux delta — alors que l identifiant de la grenade a
		// fragmentation y apparait 79 fois contre 0,17 attendue par hasard. La mesure de ce qui
		// PRECEDE l identifiant rend `0xA60400` sur 95/95, constante a 24 bits et dispersee a
		// 25 : l amorce vaut `0x20400`, pas `0x20600`. Index a +99, 21 valeurs distinctes,
		// maximum 23.
		return amorceMajeure31, true
	}
	return AmorceGrenade{}, false
}

// LES DEUX VERSIONS MAJEURES DES FILMS SANS SECTION D IDENTIFICATION, nommees une fois. Elles se
// lisent a `chunk_00+0` et sont la seule cle que ces cinq films du cache ecrivent.
const (
	// majeureSansSection33 : `a349fea8` et un second film du cache.
	majeureSansSection33 = 33
	// majeureSansSection31 : `03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`.
	majeureSansSection31 = 31
)
