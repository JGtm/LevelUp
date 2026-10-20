package grammar

// film_player_table.go — LA TABLE DES JOUEURS DU FILM, LUE PAR LA PRODUCTION (lot 1.6.0).
//
// # « L'INDEX C'EST L'INDEX » (decision utilisateur du 2026-09-07)
//
// Le film porte sa PROPRE table d'identite : `chunk_00` ouvre sur le registre, puis sur une
// section d'identification (build, version, horodatage) et enfin sur trente-deux enregistrements
// de slot qui portent chacun un XUID et un gamertag (lot 1.5, `ReadFilmIdentity` /
// `ReadPlayerTable`). C'est le lien DIRECT `index <-> xuid <-> gamertag`, ecrit par le
// jeu ; ce fichier est son seul appelant de production.
//
// # CE QUE LA MESURE DU 2026-09-14 A APPRIS, ET QUI COMMANDE TOUTE LA SUITE
//
// Sur les huit builds du golden, confrontee a la table lue dans les chunks de replication
// (`player_index.go`) :
//
//	film     build       sieges  table d'index  accord  contradiction  le film seul  le controle seul
//	000d5950 HI_1_13_0     8          8            8          0             0                0
//	a521164d HI_1_4_1     24         27           23          0             1                4
//	60ae07c4 HI_1_8_0      8          8            8          0             0                0
//	11de8353 HI_1_9_0     24         27           23          0             1                4
//	111fa685 HI_1_10_0    24         25           24          0             0                1
//	e5adf7b2 HI_1_11_0    23         28           23          0             0                5
//	bcb6d393 HI_1_12_0     8         11            8          0             0                3
//	fb1a1a72 HI_1_13_0     8          8            8          0             0                0
//
// DEUX FAITS, ET AUCUN DES DEUX N'ETAIT ECRIT AVANT : (1) la ou les deux parlent du meme joueur
// elles DISENT LA MEME CHOSE — 125 accords, 0 contradiction, sur huit builds et six generations
// de jeu ; (2) la table du film est celle du DEBUT du film — un joueur qui rejoint en cours de
// partie n'y est pas (jusqu'a 5 sur un BTB), et inversement elle assoit un joueur que le
// balayage des chunks ne trouve pas (2 films sur 8).
//
// LA CONSEQUENCE EST QUE LA TABLE DU FILM NE REMPLACE PAS LA LECTURE DES CHUNKS, elle la
// PRECEDE : le registre d'identite pose d'abord les sieges du film (lien direct, voie
// `table_du_film`), puis COMPLETE par la lecture des chunks pour les joueurs dont la table est
// MUETTE — un repli nomme, declenche sur un diagnostic (« ce xuid n'a pas de siege ») et jamais
// sur un desaccord (D14 b). Remplacer l'une par l'autre perdrait des joueurs ; c'est exactement
// ce que le gate corpus interdit.
//
// # CE QUI SE REFUSE, ET COMMENT ON L'APPREND
//
// Un film sans section d'identification (5 au cache : `03af54c3`, `13b00e35`, `47d20b5d`,
// `50247b26`, `a349fea8`), un build absent de la table de profil, un `chunk_00` tronque, une
// table introuvable : chacun rend une cause NOMMEE ([FilmTableRefusal]), journalisee, et publiee
// dans la couverture. Aucun ne fait paniquer, aucun ne rend une table partielle en silence, et
// aucun n'est lu « au profil du build voisin » (D-4 d'ADR 0034).
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM, DU-3 = S1) :
// c est une LECTURE du film. Deplacement pur de la lecture ; la JOURNALISATION (qui porte le
// `match_id`) et le COMPTEUR expvar du build inconnu restent chez l appelant de production
// (`replay`, `lireLaTableDuFilm`) : cette couche nomme ses compteurs, le consommateur les cable
// (ADR 0034 D-4, meme patron que `UnknownBuildExpvarPairs`). La lecture rend donc la table ET
// l erreur typee qui a motive un refus. `FilmPlayerTable` n est pas en `film/types` : elle porte
// une methode ([FilmPlayerTable.Lue]), donc une regle (regle du paquet `types`, doc.go).

import (
	"errors"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// FilmTableRefusal nomme la cause pour laquelle la table du film n'a PAS ete lue. Liste FERMEE :
// un refus sans cause ne se corrige pas, il se contemple (doctrine D14 du chantier).
type FilmTableRefusal string

const (
	// FilmTableRead : la table a ete lue. C'est la valeur du cas nominal.
	FilmTableRead FilmTableRefusal = ""
	// FilmTableNoRegistry : le film ne porte pas son `chunk_00` (bobine partielle).
	FilmTableNoRegistry FilmTableRefusal = "sans_registre"
	// FilmTableNoSection : le `chunk_00` ne porte aucune section d'identification — 5 films du
	// cache, nommes dans `ErrNoFilmIdentity`.
	FilmTableNoSection FilmTableRefusal = "sans_section"
	// FilmTableUnknownBuild : le build du film n'est pas dans la table de profil. Le film est mis
	// de cote, JAMAIS lu au profil du build le plus proche (D-4), et un compteur expvar le dit.
	FilmTableUnknownBuild FilmTableRefusal = "build_inconnu"
	// FilmTableTruncated : le `chunk_00` s'arrete avant la fin de la section lue.
	FilmTableTruncated FilmTableRefusal = "tronque"
	// FilmTableNotFound : aucun depart ne ferme la table a 32 slots.
	FilmTableNotFound FilmTableRefusal = "table_introuvable"
)

// FilmPlayerSeat est un siege OCCUPE de la table du film, reduit a ce que l'assemblage consomme.
//
// LES NEUF CHAMPS COURTS ET LE JETON DE SESSION QUE `types.PlayerSlot` PUBLIE N'ENTRENT PAS :
// aucun consommateur ne les lit, et ce type voyage dans le fixture d'entrees fige — porter ce
// qu'on ne consomme pas est exactement ce que la doctrine du fixture interdit.
type FilmPlayerSeat struct {
	// FilmIndex est le RANG du siege dans la table de 32, vacants compris — l'index du tableau
	// que l'ecrivain parcourt. Il EST le `player_index` de production (mesure du 2026-09-12 :
	// constant sur 76 films sur 76) tant qu'aucun vacant n'est INTERCALE.
	FilmIndex int
	// XUID de l'occupant, en numerique.
	XUID uint64
	// Gamertag tel que LE FILM l'ecrit (UTF-16 dans l'enregistrement de slot).
	Gamertag string
}

// FilmPlayerTable est ce que la table du film donne, et ce que sa lecture a coute.
type FilmPlayerTable struct {
	// Seats sont les sieges OCCUPES, dans l'ordre du rang. Vide des que `Refusal` n'est pas
	// [FilmTableRead].
	Seats []FilmPlayerSeat
	// Build est le build lu en clair dans la section d'identification. Vide quand elle manque ;
	// RENSEIGNE meme sur un refus pour build inconnu, parce que c'est LUI qu'il faut nommer.
	Build string
	// Occupied / Vacant : la table lue. Leur somme vaut 32 sur une lecture nominale.
	Occupied int
	Vacant   int
	// InterleavedVacant dit qu'un siege vacant tombe AVANT le dernier occupe. C'est le SEUL cas
	// ou « rang absolu » et « index parmi les occupes » divergent, et la mesure du 2026-09-14 ne
	// tranche pas lequel des deux est le `player_index` : aucun des 13 films du cache concernes
	// n'a de document de rejeu. Le registre d'identite traite donc ce cas en CONTRADICTION
	// nommee — il n'affirme pas le lien direct (cf. identity_registry_film_table.go).
	InterleavedVacant bool
	// Refusal nomme la cause quand la table n'a pas ete lue. Vide = lue.
	Refusal FilmTableRefusal
}

// Lue dit si la table a ete lue ET si son lien direct est affirmable.
func (t FilmPlayerTable) Lue() bool {
	return t.Refusal == FilmTableRead && len(t.Seats) > 0 && !t.InterleavedVacant
}

// ScanFilmPlayerTable lit la table des joueurs du film. HORS LIGNE (elle lit `chunk_00`) ;
// appelee une fois par cuisson, dans l'etage de balayage.
//
// ELLE NE PANIQUE JAMAIS ET NE REND JAMAIS DE TABLE PARTIELLE : chaque cause d'echec est TYPEE
// ici, traduite en [FilmTableRefusal] NOMMEE ; l'erreur qui l'a motivee accompagne le refus
// (nil pour un film sans registre), pour que l'appelant la journalise et la publie dans la
// couverture de l'artefact. Un refus se compte ; il ne se tait pas.
func ScanFilmPlayerTable(film *source.Film) (FilmPlayerTable, error) {
	chunk0, ok := FilmRegistryChunk(film)
	if !ok {
		return FilmPlayerTable{Refusal: FilmTableNoRegistry}, nil
	}
	return lireTableDeChunk0(chunk0)
}

// lireTableDeChunk0 est le corps de [ScanFilmPlayerTable] sur les octets DEJA DECOMPRESSES du
// registre. Separe pour qu'un test puisse muter ces octets — le refus pour build inconnu, en
// particulier, ne se provoque pas autrement (aucun film du cache n'a de build hors profil).
func lireTableDeChunk0(chunk0 []byte) (FilmPlayerTable, error) {
	ident, err := ReadFilmIdentity(chunk0)
	if err != nil {
		return FilmPlayerTable{Refusal: causeIdentite(err)}, err
	}
	slots, rep, err := ReadPlayerTable(chunk0, ident)
	if err != nil {
		return FilmPlayerTable{Build: ident.Build, Refusal: causeTable(err)}, err
	}
	t := FilmPlayerTable{
		Seats: make([]FilmPlayerSeat, 0, len(slots)), Build: ident.Build,
		Occupied: rep.Occupied, Vacant: rep.Vacant, InterleavedVacant: rep.InterleavedVacant,
	}
	for _, s := range slots {
		t.Seats = append(t.Seats, FilmPlayerSeat{
			FilmIndex: s.FilmIndex, XUID: s.XUID, Gamertag: s.Gamertag})
	}
	return t, nil
}

// causeIdentite traduit l'erreur de [ReadFilmIdentity] en cause nommee.
func causeIdentite(err error) FilmTableRefusal {
	if errors.Is(err, ErrNoFilmIdentity) {
		return FilmTableNoSection
	}
	return FilmTableTruncated
}

// causeTable traduit l'erreur de [ReadPlayerTable] en cause nommee.
func causeTable(err error) FilmTableRefusal {
	switch {
	case errors.Is(err, profile.ErrUnknownBuild):
		return FilmTableUnknownBuild
	case errors.Is(err, ErrPlayerTableNotFound):
		return FilmTableNotFound
	default:
		return FilmTableTruncated
	}
}
