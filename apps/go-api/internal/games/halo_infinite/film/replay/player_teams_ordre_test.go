package replay

// player_teams_ordre_test.go — LE XUID D'ABORD, LE PONT SLOT ENSUITE
// (revue de jalon M1, lentille L6 : ce que les tests ne couvrent pas).
//
// # LE TROU QUE CE FICHIER FERME, ET IL A ETE MESURE
//
// L'en-tete de [teamPublication.poserSurLesTraces] dit « L'ORDRE EST FIXE ET IL N'EST PAS
// ARBITRAIRE : le xuid d'abord (le lien direct du lot 1.6), le pont slot -> index ensuite (la
// seule voie d'un bot) ». Mutation jouee le 2026-09-15 sur la base `34fa53da5` — les deux voies
// PERMUTEES — et rien ne rougissait : `replay`, `replaybuild`, `replayview` et `replaydoc` ne
// rendaient que les deux goldens deja rouges par construction pendant la vague
// (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted`).
//
// La raison tient a la donnee : sur les bobines versionnees, les deux voies s'accordent partout —
// un ordre ne se teste que la ou les deux sources DIVERGENT, et cela ne s'observe pas, cela se
// fabrique.
//
// # POURQUOI UN SIEGE RECYCLE, ET NON UN CAS D'ECOLE
//
// `IndexParSlot` rend le PREMIER occupant d'un siege de bipede
// (`OwnerReport.Owner`) : quand un joueur quitte et qu'un autre reprend le siege, le pont continue
// de nommer le premier. Une vie du SECOND occupant porte donc son propre xuid et un slot dont le
// pont donne l'equipe de quelqu'un d'autre. Le xuid est le lien DIRECT, le pont est une
// DEDUCTION : c'est le lien direct qui doit gagner, sans quoi le rejeu attribuerait au remplacant
// l'equipe du partant.

import "testing"

// publicationDeuxVoiesDivergentes : un decor ou les deux voies ne disent PAS la meme chose.
//
//	index 0 -> equipe 0     et le xuid 111 porte cet index
//	index 1 -> equipe 1     et le pont donne l'index 1 pour le slot 200
//
// Une vie qui porte A LA FOIS le xuid 111 et le slot 200 est donc nommee « equipe 0 » par le lien
// direct et « equipe 1 » par le pont. Le slot n'est PAS ambigu — il a une entree et une seule —
// pour que le temoin porte l'ORDRE et rien d'autre.
func publicationDeuxVoiesDivergentes() teamPublication {
	return teamPublication{
		byIndex: map[int]int{0: 0, 1: 1},
		byXUID:  map[uint64]int{111: 0},
		bySlot:  map[uint32]int{200: 1},
	}
}

// TestLEquipeDuXUIDPrimeSurLePontDuSlot — LE TEMOIN.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : permuter les deux voies dans
// [teamPublication.poserSurLesTraces] (le pont du slot essaye en premier). La vie du remplacant
// prend alors l'equipe du premier occupant du siege. Jouee et restauree par nom le 2026-09-15
// (sorties collees au §5 du plan).
func TestLEquipeDuXUIDPrimeSurLePontDuSlot(t *testing.T) {
	tracks := []Track{{Slot: 200, XUID: "111", Team: -1}}

	total, nommees := publicationDeuxVoiesDivergentes().poserSurLesTraces(tracks)

	if tracks[0].Team != 0 {
		t.Errorf("equipe posee = %d, attendue 0 (celle du XUID 111) : le pont du siege 200 dit 1, "+
			"mais il ne nomme que son PREMIER occupant — le lien direct prime", tracks[0].Team)
	}
	if total != 1 || nommees != 1 {
		t.Errorf("comptes = %d vie(s) / %d nommee(s), attendu 1 / 1", total, nommees)
	}
}

// TestLePontDuSlotNommeLaVieSansXUID : l'autre moitie de l'ordre, et elle est indispensable.
//
// Sans elle, un lecteur qui supprimerait purement le pont passerait le temoin precedent. Une vie
// de BOT n'a pas de xuid — le pont est sa SEULE voie, et il doit continuer de servir.
func TestLePontDuSlotNommeLaVieSansXUID(t *testing.T) {
	tracks := []Track{{Slot: 200, Team: -1}}

	total, nommees := publicationDeuxVoiesDivergentes().poserSurLesTraces(tracks)

	if tracks[0].Team != 1 {
		t.Errorf("equipe posee = %d, attendue 1 (celle du pont slot 200 -> index 1) : une vie sans "+
			"xuid n'a que cette voie", tracks[0].Team)
	}
	if total != 1 || nommees != 1 {
		t.Errorf("comptes = %d vie(s) / %d nommee(s), attendu 1 / 1", total, nommees)
	}
}

// TestUneVieQueNiLUneNiLAutreVoieNeNommeGardeMoinsUn : le silence se compte et ne s'invente pas.
// C'est la troisieme branche de la boucle, et elle porte la sentinelle publiee (`-1`).
func TestUneVieQueNiLUneNiLAutreVoieNeNommeGardeMoinsUn(t *testing.T) {
	tracks := []Track{{Slot: 999, XUID: "222", Team: -1}}

	total, nommees := publicationDeuxVoiesDivergentes().poserSurLesTraces(tracks)

	if tracks[0].Team != -1 {
		t.Errorf("equipe posee = %d, attendue -1 : ni le xuid 222 ni le siege 999 ne sont nommes",
			tracks[0].Team)
	}
	if total != 1 || nommees != 0 {
		t.Errorf("comptes = %d vie(s) / %d nommee(s), attendu 1 / 0 : une vie non nommee se COMPTE",
			total, nommees)
	}
}
