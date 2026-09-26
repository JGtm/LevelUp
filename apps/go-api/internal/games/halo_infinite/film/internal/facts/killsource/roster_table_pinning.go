package killsource

// roster_table_pinning.go — CE QUE LA TABLE DU FILM ET LE MOTIF DU XUID ONT EPINGLE, ET CE QUE LE
// KILL-FEED EN DIT : [FilmTablePinning] et [FilmTablePinning.AffectationUnique].
//
// SORTI DE `roster.go` PAR DEPLACEMENT PUR au lot J7.1 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM) :
// le fichier atteignait le plafond de 500 lignes (`archlint/film_file_size_test.go`) au moment ou
// le predicat du nom de remplissage s y ajoutait. Aucun octet de code n est change.

// FilmTablePinning : ce que la table du film a epingle, et ce que le kill-feed en dit.
//
// LES TROIS DERNIERS CHAMPS SONT UN CONTROLE, JAMAIS UNE DECISION (D14 b) : la table du film
// n est pas corrigee par les votes du kill-feed, elle est CONFRONTEE a eux. Une contradiction se
// compte et se lit ; elle ne se resout pas en silence.
type FilmTablePinning struct {
	// Refusal : la cause nommee quand la table n a pas ete lue. Vide = lue.
	Refusal FilmTableRefusal
	// Build : le build lu en clair (renseigne meme sur un refus pour build inconnu).
	Build string
	// Seats : les sieges OCCUPES et NOMMES que la table rend.
	Seats int
	// Pinned : les indices dont le joueur vient de la table — la part LUE de la bijection.
	Pinned int
	// AddedNames : les noms que la table ajoute au roster parce que le kill-feed ne les porte
	// pas. Mesure du 2026-09-14 : 6 sur 30 films, tous des joueurs qui n ont ni tue ni sont morts.
	AddedNames int
	// BotConflict : sieges refuses parce que BOT_METADATA epingle deja cet indice. Deux lectures
	// du film qui se contredisent : on garde la plus ancienne et la plus eprouvee (le bot), et on
	// COMPTE. Mesure du 2026-09-14 : 0 sur 30 films.
	BotConflict int
	// DuplicateName : sieges refuses parce que le meme nom est deja epingle a un autre indice.
	DuplicateName int
	// OutOfRange : sieges dont l indice sort de l espace des 5 bits (0..31). Impossible par
	// construction de la table de 32 ; compte pour que l impossible se voie s il arrive.
	OutOfRange int
	// Inferred : les indices laisses a l inference — LE REPLI, compte.
	Inferred int
	// FreeNames : les NOMS que l inference a encore a placer sur ces indices. Il n est PAS la
	// meme quantite que `Inferred` et il ne l a plus jamais ete depuis le lot 1.8 : la table du
	// film ajoute au roster les joueurs que le kill-feed ne nomme pas, donc il peut rester plus
	// de noms libres que d indices libres (cf. [roster.freeSlots]). Sans ce compte, « un seul
	// indice a inferer » se lisait a tort « une seule affectation possible » — alors que deux
	// noms pour un indice se tranchent par les votes, et qu un nom sans kill ni mort n en porte
	// aucun : le choix etait ARBITRAIRE (revue de jalon M1, lentille L4).
	FreeNames int
	// MotifPinned / MotifAgree / MotifContradict / MotifDuplicate : LE LIEN PAR LE MOTIF DU XUID
	// (lot 5.2b.1, `index_motif.go`), ventile comme la table l est au-dessus. `MotifPinned` est
	// ce qu il AJOUTE — les indices que ni BOT_METADATA ni la table de `chunk_00` n epinglent,
	// c est-a-dire les REMPLACANTS ; `MotifAgree` / `MotifContradict` sont le CONTROLE sur les
	// indices deja epingles (meme nom / autre nom), et une contradiction ne tranche rien : la
	// table de `chunk_00` garde la main, elle est la plus eprouvee. `MotifDuplicate` : le nom lu
	// est deja epingle ailleurs.
	MotifPinned, MotifAgree, MotifContradict, MotifDuplicate int
	// MotifReadings / MotifDisagreements / MotifAbsent : le COUT de cette lecture. `Readings` =
	// chunks de replication qui ont livre au moins un index ; `Disagreements` = xuids lus a deux
	// index differents (non publies) ; `Absent` = xuids dont le motif ne figure dans aucun chunk.
	MotifReadings, MotifDisagreements, MotifAbsent int
	// MotifTueursEcartes : TUEURS SANS MORT (cherches depuis le lot J7.3) dont la lecture se contredit
	// ou tombe sur un indice deja retenu : ecartes SEULS, jamais comptes en `MotifDisagreements` —
	// ils ne font pas tomber l epinglage par motif du film (revue du lot J7, `index_motif_tueurs.go`).
	MotifTueursEcartes int
	// Agree / Contradict / Silent : le CONTROLE des indices epingles par les votes du kill-feed.
	// `Agree` = les votes designent le meme joueur ; `Contradict` = ils en designent un autre,
	// strictement plus vote ; `Silent` = aucun vote sur cet indice (le joueur n a ni tue ni est
	// mort dans la fenetre d appariement).
	Agree, Contradict, Silent int
}

// AffectationUnique dit si l inference n avait QU UNE SEULE affectation possible a rendre.
//
// C EST LA QUESTION QUE `Inferred <= 1` CROYAIT POSER, ET QU IL NE POSAIT PAS. Un indice libre
// pour DEUX noms libres se tranche par les votes du kill-feed, et le cout d un nom qui n a ni
// tue ni ete tue vaut zero contre tous les indices : le hongrois rend alors un nom pris au
// hasard du departage ([permLess]), [refine] ne peut rien echanger (il lui faudrait deux indices
// libres) et [bijectionMargin] rend structurellement zero (sa double boucle ne tourne pas sur
// une seule case libre). La porte de publication ligne par ligne reposait donc sur ce seul
// booleen, et publiait la source du degat, le credit, l assistant et les deux parts de degats
// sur un occupant TIRE AU SORT.
//
// LES TROIS REGIMES, ET POURQUOI LE PREMIER RESTE VRAI. Aucun indice libre : la bijection est
// entierement LUE, il n y a rien a choisir — les noms libres qui restent ne portent aucun
// indice, ce qui est exact (cf. [hungarianStart]). Un indice libre pour au plus un nom libre :
// l affectation est forcee. Au-dela : au moins un choix, donc la marge de bijection reprend son
// office.
func (t FilmTablePinning) AffectationUnique() bool {
	switch t.Inferred {
	case 0:
		return true
	case 1:
		return t.FreeNames <= 1
	default:
		return false
	}
}
