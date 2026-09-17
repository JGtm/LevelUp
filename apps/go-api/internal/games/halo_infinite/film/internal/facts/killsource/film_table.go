package killsource

// film_table.go — LE LIEN DIRECT `index de joueur <-> gamertag`, ECRIT PAR LE JEU.
//
// # « L INDEX C EST L INDEX » (decision utilisateur du 2026-09-07, lot 1.8 du 2026-09-14)
//
// Le dead-state designe un participant par un INDICE ABSOLU sur 5 bits, jamais par un nom. Jusqu
// ici ce paquet reconstituait le lien par INFERENCE : une matrice de votes du kill-feed resolue
// par l algorithme hongrois, puis une montee locale sur le couple (bijection.go). C etait la
// seule voie possible tant que le film etait cense ne porter aucune identite cote replication —
// affirmation ecrite noir sur blanc dans `killcollector/roster.go` et DEMENTIE par le lot 1.5 :
// `chunk_00` porte trente-deux enregistrements de slot, chacun avec son XUID et son gamertag.
//
// LA TABLE DU FILM DEVIENT DONC LA SOURCE, ET L INFERENCE LE REPLI (doctrine D13 / D-10 d ADR
// 0034 : la grammaire prime, l heuristique ne survit qu en repli NOMME et COMPTE).
//
// # CE QUE LA MESURE DU 2026-09-14 DIT, SUR 30 FILMS ET 8 BUILDS
//
// Sonde jetable (supprimee apres la mesure), table de `chunk_00` confrontee a la bijection
// inferee, index par index : **314 accords sur 322 sieges lus**, et les HUIT ecarts se rangent
// en deux familles, aucune n etant une contradiction entre deux lectures fiables :
//
//	SIX  le gamertag du siege est ABSENT du kill-feed (`FlukiestGolf` 111fa685 i10,
//	     `MarshallG6443` e5adf7b2 i13, `manistoff` a521164d i18, `Iskra 20252993` 11de8353 i23,
//	     `probablybxllets` 1c5c10cc i22, `Alpha122092` 23ffd885 i4). Le kill-feed ne nomme que
//	     les joueurs qui TUENT ou qui MEURENT : l inference n avait aucun moyen de placer ces
//	     six-la, et elle a mis un AUTRE joueur sur leur indice. Ce n est pas un desaccord, c est
//	     un trou de l inference que la lecture comble.
//	DEUX le gamertag EST au feed (`CR951802` 23ffd885 i2, `SerdarTsn` b1bcbe24 i13) — et les deux
//	     tombent sur un film dont la MARGE DE BIJECTION VAUT ZERO, c est-a-dire ou l inference
//	     DIT ELLE-MEME que deux joueurs sont interchangeables et ou les lignes ne sont pas
//	     publiables. Aucun ecart sur un film ou l inference se declare fiable.
//
// DEUXIEME MESURE, QUI FERME UNE QUESTION OUVERTE DU LOT 1.5 : les treize films du cache a slot
// vacant INTERCALE rendent 119 accords sur 123, et les quatre ecarts sont ceux des deux familles
// ci-dessus. C est le RANG ABSOLU (vacants compris) que le dead-state emploie, pas l index parmi
// les occupes — sur ces treize films les deux lectures divergent, et c est le rang absolu qui
// s accorde. Decouverte D1 (1.8).
//
// # CE QUI SE REFUSE, ET COMMENT ON L APPREND
//
// Un film sans `chunk_00` (bobine partielle), sans section d identification (5 au cache :
// `03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`), un build absent de la table de
// profil, un tampon tronque, une table introuvable : chacun rend une cause NOMMEE, et le decodeur
// retombe alors sur l inference ENTIERE, comptee comme telle. Aucun ne panique, aucun ne rend une
// table partielle en silence, aucun n est lu « au profil du build voisin » (D-4 d ADR 0034).
//
// # COPIE JUMELLE, ASSUMEE ET CONSIGNEE
//
// `film/replay/film_player_table.go` (lot 1.6.0) fait la meme traduction erreur -> cause nommee
// pour l assembleur du rejeu. C est la DEUXIEME copie et la derniere tolerable (CLAUDE.md regle
// 6) : une troisieme impose la centralisation chez `grammar`. Consigne en §4 du plan.

import (
	"errors"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// FilmTableRefusal nomme la cause pour laquelle la table du film n a PAS ete lue. Liste FERMEE :
// un refus sans cause ne se corrige pas, il se contemple.
type FilmTableRefusal string

const (
	// FilmTableRead : la table a ete lue. Valeur du cas nominal.
	FilmTableRead FilmTableRefusal = ""
	// FilmTableNoRegistry : le film ne porte pas son `chunk_00`.
	FilmTableNoRegistry FilmTableRefusal = "sans_registre"
	// FilmTableNoSection : le `chunk_00` ne porte aucune section d identification.
	FilmTableNoSection FilmTableRefusal = "sans_section"
	// FilmTableUnknownBuild : le build du film n est pas dans la table de profil. Le film est mis
	// de cote, JAMAIS lu au profil du build le plus proche (D-4).
	FilmTableUnknownBuild FilmTableRefusal = "build_inconnu"
	// FilmTableTruncated : le `chunk_00` s arrete avant la fin de la section lue.
	FilmTableTruncated FilmTableRefusal = "tronque"
	// FilmTableNotFound : aucun depart ne ferme la table a 32 slots.
	FilmTableNotFound FilmTableRefusal = "table_introuvable"
)

// FilmTable est ce que la table des joueurs du film donne au decodeur, et ce que sa lecture a
// coute. Publiee dans [Result] pour que le consommateur sache d ou vient chaque identite.
type FilmTable struct {
	// Seats : `index de joueur -> gamertag`, sieges OCCUPES seulement. Vide sur un refus.
	Seats map[int]string
	// Build : le build lu en clair. RENSEIGNE meme sur un refus pour build inconnu — c est lui
	// qu il faut nommer.
	Build string
	// Occupied / Vacant : la table lue. Leur somme vaut 32 sur une lecture nominale.
	Occupied, Vacant int
	// InterleavedVacant : un siege vacant tombe AVANT le dernier occupe. Mesure du lot 1.8 :
	// le rang absolu s accorde quand meme avec l indice du dead-state (119/123 sur les 13 films
	// du cache concernes), donc ce n est PAS un motif de refus ici — c est une donnee.
	InterleavedVacant bool
	// Refusal : la cause nommee quand la table n a pas ete lue. Vide = lue.
	Refusal FilmTableRefusal
}

// Lue dit si la table a ete lue et porte au moins un siege.
func (t FilmTable) Lue() bool { return t.Refusal == FilmTableRead && len(t.Seats) > 0 }

// readFilmTable lit la table des joueurs depuis le PREMIER chunk du film — le meme que
// `newTimeline` lit comme registre ECS (cf. l en-tete de chunks.go sur « le chunk 0 est le
// premier de la source »).
//
// ELLE NE REND JAMAIS D ERREUR : chaque cause d echec est TYPEE chez `grammar` et traduite ici en
// cause NOMMEE, portee par le resultat et journalisee par l appelant. Un refus se compte.
func readFilmTable(f *film) FilmTable {
	if f == nil || f.src.NumChunks() == 0 || len(f.src.Chunk(0)) == 0 {
		return FilmTable{Refusal: FilmTableNoRegistry}
	}
	registre := f.src.Chunk(0)
	ident, err := grammar.ReadFilmIdentity(registre)
	if err != nil {
		return FilmTable{Refusal: causeIdentite(err)}
	}
	slots, rep, err := grammar.ReadPlayerTable(registre, ident)
	if err != nil {
		return FilmTable{Build: ident.Build, Refusal: causeTable(err)}
	}
	t := FilmTable{
		Seats: make(map[int]string, len(slots)), Build: ident.Build,
		Occupied: rep.Occupied, Vacant: rep.Vacant, InterleavedVacant: rep.InterleavedVacant,
	}
	for _, s := range slots {
		if s.Gamertag == "" {
			continue // un siege sans nom n identifie personne : il ne peut rien epingler
		}
		t.Seats[s.FilmIndex] = s.Gamertag
	}
	return t
}

// causeIdentite traduit l erreur de [grammar.ReadFilmIdentity] en cause nommee.
func causeIdentite(err error) FilmTableRefusal {
	if errors.Is(err, grammar.ErrNoFilmIdentity) {
		return FilmTableNoSection
	}
	return FilmTableTruncated
}

// causeTable traduit l erreur de [grammar.ReadPlayerTable] en cause nommee.
func causeTable(err error) FilmTableRefusal {
	switch {
	case errors.Is(err, profile.ErrUnknownBuild):
		return FilmTableUnknownBuild
	case errors.Is(err, grammar.ErrPlayerTableNotFound):
		return FilmTableNotFound
	default:
		return FilmTableTruncated
	}
}
