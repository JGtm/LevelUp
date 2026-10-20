package grammar

// handle.go — LA LECTURE DU HANDLE D UN RECORD, ET ELLE EST UNIQUE (lot J5.1 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DT-7).
//
// # CE QU EST LE HANDLE
//
// L en-tete de tout record de la trame designe son entite par un HANDLE : `R(13)` de slot puis
// `R(2)` de generation (`readRecordID`, port de FUN_1406d3140 : « the tag is generation »). Le
// slot indexe le monde ; la generation distingue les vies successives d un meme slot, parce que
// le pool de slots reboucle. La paire est [types.LifeKey].
//
//	record DELTA  [1 prefixe = 1][13 slot][2 generation] ...   le handle commence au bit p+1
//	record NEW    [1 = 0][2 type = 1][13 slot][2 generation]   le handle commence au bit p+3
//
// # POURQUOI UN SEUL LECTEUR
//
// Le handle etait relu A LA MAIN par cinq balayages (`matchBipedHeaderRaw`, `matchAimOnlyRecord`,
// `scanEquipRecoveryPacket`, `matchWorldObjectRecord`, `matchWorldObjectNewHeaderIn`), plus
// `enteteNeufEn` et l instrument `cmd_fermeture` — sous le litteral `p+14, 2)` ou sous des
// constantes nommees (`woNewSlotBits`, `decalageDuTag`). Les trois copies du filtre « tag == 1 »
// du bipede (constat GB-1 de l audit du 2026-09-24) en sont nees : chaque copie decidait pour son
// compte de ce qu est une generation acceptable. Le handle se lit desormais ICI ; ce que chaque
// balayage FAIT de la generation reste chez lui.
//
// Garde-rail : `archlint/film_handle_lecture_test.go` rougit sur toute lecture de 13 bits, et
// sur toute lecture de 2 bits placee a la generation d un handle, hors de ce fichier.
//
// # LA CONVENTION DE BORD
//
// [source.BitsTolerants] : zero au-dela du tampon. Les balayages bipedes bornent eux-memes leur
// curseur avant de lire l en-tete (le handle y est toujours dans le tampon, et la lecture y est
// donc bit a bit celle de `BitsStricts`) ; les reconnaissances d en-tete d objet du monde, elles,
// sont appelees a TOUTE position d un paquet par les sondes et doivent rendre un refus, jamais une
// panique — c est la convention qu elles portaient deja.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Largeurs du handle, et place du handle dans un en-tete de record DELTA.
const (
	// handleSlotBits : le slot (`IDLowBits` du cadre par defaut).
	handleSlotBits = 13
	// handleGenBits : la generation — les deux bits de tete de l identifiant de 32 bits.
	handleGenBits = 2
	// prefixeDeltaBits : le prefixe `1` d un record DELTA, qui precede son handle.
	prefixeDeltaBits = 1
)

// LireHandle lit le handle (slot, generation) qui COMMENCE au bit `at` de `pay`. Zero au-dela du
// tampon (cf. l en-tete).
func LireHandle(pay []byte, at int) types.LifeKey {
	return types.LifeKey{
		Slot: uint32(source.BitsTolerants(pay, at, handleSlotBits)),               //nolint:gosec // 13 bits
		Gen:  uint32(source.BitsTolerants(pay, at+handleSlotBits, handleGenBits)), //nolint:gosec // 2 bits
	}
}

// LireHandleDelta lit le handle d un record DELTA dont l en-tete commence au bit `p`.
func LireHandleDelta(pay []byte, p int) types.LifeKey { return LireHandle(pay, p+prefixeDeltaBits) }
