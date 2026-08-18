package objectiveevents

// flagfilm.go — LE FILM EST-IL UNE PARTIE DE CTF ? La reponse par le SEUL film.
//
// # Pourquoi cette question se pose, alors que la base y repond
//
// [ObjectiveTypeOf] classe un match par son `game_variant_name`, qui vient du registre. Mais
// l'artefact de rejeu 2D est construit HORS LIGNE, a partir des seuls chunks du film : il ne
// connait ni la carte, ni le mode, ni le moindre champ de la base (cf. `analysis/replay`,
// en-tete de document.go). Publier le portage du drapeau exige pourtant de savoir qu'on est en
// CTF — la table d'emplacements de statistiques du drapeau ([namedStatSlots]), appliquee au film
// d'un AUTRE mode, rend des « prises » qui n'en sont pas : sur un film Oddball elle compte 1 470
// prises et 994 vols, sur un film de Bastion 27 prises.
//
// # La regle, et les quatre comptes qui la portent
//
// Trois signaux, tous lus dans le film, et leur ACCORD :
//
//	bursts   > 0            l'evenement de score a 6 tiers d'une capture de drapeau
//	captures > 0            `comp 21 A` de la table drapeau
//	captures <= bursts      les deux chaines comptent LA MEME CHOSE
//	steals   > 0            `comp 24 A` — on ne vole un drapeau qu'en CTF
//
// L'INEGALITE EST DANS CE SENS-LA POUR UNE RAISON MESUREE : un film que le Theater rend TRONQUE
// arrete ses compteurs avant la fin, donc `captures` peut etre INFERIEUR au nombre de bursts
// (`64e8adfa` : 4 contre 5), jamais superieur. Exiger l'egalite rejetterait les films tronques —
// exactement ceux que le pont par instants de mort vient de rendre exploitables.
//
// # Ce que la regle rend sur un corpus de mode CONNU (2026-08-18, plan objectifs vivants,
// item 1.1 ; seuil ecrit avant mesure : ZERO faux positif)
//
//	mode      films   bursts / captures / steals                        verdict
//	flag        6     3-5 / 3-5 / 4-17                                  6 retenus
//	skull       1     2 / 6 / 994      -> captures > bursts             ecarte
//	zone        2     0 / 12-16 / 0    -> aucun burst                   ecartes
//	hill        4     0-4 / 0-3 / 0-55 -> aucun burst, ou 0 capture     ecartes
//	slayer      2     0-2 / 0 / 0      -> 0 capture                     ecartes
//
// 15 films, 15 verdicts justes. La regle est gelee par [TestFlagFilmVerdictSurCorpusMesure],
// qui rejoue ces quinze lignes SANS film.
//
// CE QU'ELLE NE DIT PAS : une partie de CTF ou personne ne capture ne produit aucun burst et
// sera ECARTEE. Le rejeu publie alors un calque de drapeau vide, et sa couverture le dit — un
// silence, jamais un calque invente.

// FlagFilmSignals porte les comptes qui fondent le verdict. Ils se publient AVEC lui : un
// « ce film n'est pas du CTF » sans ses comptes ne se verifie pas.
type FlagFilmSignals struct {
	// Bursts : nombre de bursts de capture (cf. [CaptureBurstTimes]).
	Bursts int
	// Captures / Steals / Grabs : les comptes de la table DRAPEAU appliquee au film, quel que
	// soit son vrai mode. Hors CTF ils n'ont aucun sens — c'est tout l'objet du verdict.
	Captures, Steals, Grabs int
}

// FlagFilmSignalsFrom compte les signaux a partir de deux lectures deja faites : les instants
// de burst ([CaptureBurstTimes]) et les evenements nommes de la table DRAPEAU
// ([NamedEvents] avec [ObjectiveTypeFlag]).
//
// Prendre les lectures en ENTREE plutot que la source evite de rebalayer le film : l'appelant
// qui publie le portage a besoin des deux de toute facon.
func FlagFilmSignalsFrom(bursts []int, evs []NamedEvent) FlagFilmSignals {
	s := FlagFilmSignals{Bursts: len(bursts)}
	for _, e := range evs {
		switch e.Stat {
		case StatFlagCaptures:
			s.Captures++
		case StatFlagSteals:
			s.Steals++
		case StatFlagGrabs:
			s.Grabs++
		}
	}
	return s
}

// IsFlagFilm applique la regle mesuree (cf. en-tete de fichier).
func (s FlagFilmSignals) IsFlagFilm() bool {
	return s.Bursts > 0 && s.Captures > 0 && s.Captures <= s.Bursts && s.Steals > 0
}
