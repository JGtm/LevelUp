package filmdec

// registry_fingerprint.go — L'EMPREINTE DU REGISTRE ECS, ET L'ALERTE QUAND ELLE CHANGE.
//
// LE PROBLEME QU'ELLE RESOUT. Toute la grammaire de ce decodeur est indexee par le registre
// du film (`chunk_00`) : les noms de composants routent le dispatch, leur ORDRE est l'index de
// bit du masque de presence. Le registre est bit-a-bit IDENTIQUE sur tous les films mesures a
// ce jour — trois films, trois cartes differentes (lot table ECS, 2026-08-18). Cette stabilite
// est une propriete du BUILD DU JEU, pas du format : une mise a jour de Halo Infinite peut
// reordonner, ajouter ou renommer des composants, et rien dans le decodeur ne le dirait. Les
// symptomes seraient des desalignements silencieux, attribues a une mauvaise grammaire de
// composant pendant des jours.
//
// CE QUE L'EMPREINTE EST. Un FNV-1a 64 bits sur la concatenation, DANS L'ORDRE DU CHUNK, des
// slots NON VIDES : les quatre octets `kind` (u32 LE, slot+0), les quatre octets `flags`
// (u32 LE, slot+4), puis les octets du NOM. Les slots de bourrage — nom vide — n'y entrent
// pas : ce sont eux qui portent le reste du bloc de 64 slots et ils ne disent rien de la
// grammaire.
//
// POURQUOI ELLE EST CALCULEE A LA LECTURE et pas recalculee a la demande : `kind` n'est PAS
// retenu par `parseRegistry` (aucun deser ne le lit — R7-e a etabli que le premier `u32` d'un
// slot est la queue de bourrage du nom precedent, pas un champ). Le garder dans `Archetype`
// serait un champ sans lecteur ; l'empreinte se calcule donc pendant l'unique passe qui voit
// les octets, et `RegistryFingerprint` la rend.
//
// CE QU'ELLE NE FAIT PAS. Elle ne refuse aucun film et ne change aucune largeur. Le decalage
// `Flags[k]` / `Flags[k+1]` (le jeu lit le niveau un cran plus loin, mesure au lot R7-e,
// inerte a ce jour) n'est PAS corrige ici : l'empreinte fige le binaire TEL QU'IL EST LU. La
// corriger sans temoin deplacerait un bug silencieux — decision consignee au registre des
// reports.

import (
	"context"
	"hash"
	"hash/fnv"
	"log/slog"
	"sync"
)

// KnownRegistryFingerprint est l'empreinte du registre de REFERENCE — celui sur lequel toute
// la grammaire portee de ce paquet a ete etablie, et celui que decrit `testdata/ecs_table.tsv`
// (118 blocs, 49 archetypes porteurs, 1 067 slots non vides).
//
// RECALCULEE LE 2026-08-17 (lot 0, item 0.3) et NON RECOPIEE. Elle ne vaut pas le
// `0xa413610cd08e4355` cite par le commentaire de `registry.go`, et l'ecart est explique :
// cette valeur-la est un FNV sur « noms + flags » (deux champs), celle-ci sur
// `kind | flags | nom` (trois champs, `kind` inclus). Deux domaines, deux valeurs — la
// documentee n'etait pas transposable.
//
// CE QUE LA MESURE A TROUVE DU PREMIER COUP, ET QUI CHANGE UNE CROYANCE DU DEPOT : le registre
// N'EST PAS identique sur tous les films. `000d5950` et `64e8adfa` rendent bien la meme valeur
// (118 blocs / 1 067 slots), mais `06dfe6d9` rend `0x5827362c37d2adb3` sur **116 blocs et
// 1 031 slots non vides** — 2 archetypes et 36 composants de moins. La stabilite mesuree au
// lot table ECS (« bit-a-bit identique sur 000d5950, 00502e52, 07aa428d ») vaut DANS UN BUILD,
// pas entre builds. C'est exactement ce que cette empreinte est faite de dire, et l'alerte se
// declenche a bon droit sur ce film : sa grammaire n'est pas celle que la table decrit.
const KnownRegistryFingerprint uint64 = 0x61e492dd4de7fd4e

// registryFNV accumule l'empreinte pendant la passe de lecture des blocs. Il travaille sur le
// buffer INFLATE et sur les memes bornes que `parseRegistry` : un slot tronque en fin de
// buffer est ignore plutot que hache a moitie.
type registryFNV struct {
	h     hash.Hash64
	slots int // slots non vides haches — le denominateur de l'alerte
}

// registryHasher construit l'accumulateur de l'empreinte.
func registryHasher() *registryFNV { return &registryFNV{h: fnv.New64a()} }

// addSlot ajoute `kind | flags | nom` d'un slot non vide.
func (f *registryFNV) addSlot(data []byte, off int, name string) {
	if off+8 > len(data) {
		return
	}
	_, _ = f.h.Write(data[off : off+8]) // kind (u32 LE) | flags (u32 LE), octets du fichier
	_, _ = f.h.Write([]byte(name))
	f.slots++
}

func (f *registryFNV) sum() uint64 { return f.h.Sum64() }

// RegistryFingerprint rend l'empreinte du registre lu. Zero pour un registre nil.
func RegistryFingerprint(reg *Registry) uint64 {
	if reg == nil {
		return 0
	}
	return reg.fingerprint
}

// registryWarned : les empreintes inconnues DEJA signalees dans ce process.
//
// LA DEDUPLICATION EST NECESSAIRE, ET ELLE SE FAIT SUR L'EMPREINTE. Le registre est re-parse
// par QUATRE chemins de production a chaque film (mesure item 0.3) : sans deduplication,
// une grammaire inconnue produirait quatre lignes identiques par film et des milliers sur une
// re-cuisson de corpus. Deduper sur l'empreinte donne exactement « une alerte par grammaire
// inconnue rencontree », c'est-a-dire une par film tant que les films d'un meme build
// partagent leur registre — la propriete mesuree.
var registryWarned sync.Map

// warnUnknownRegistry signale UNE FOIS par empreinte qu'un registre ne correspond pas au
// binaire de reference. Le film reste decode : l'empreinte est un signal, pas une porte.
//
// `context.Background()` : `ParseRegistryChunk` est une fonction de lecture d'octets sans
// `ctx`, appelee depuis quatre chemins dont deux n'en ont pas non plus. Faire remonter un
// `ctx` jusqu'ici changerait la signature publique et celle de ses appelants pour une ligne
// de journal.
func warnUnknownRegistry(fp uint64, blocks, slots int) {
	if fp == KnownRegistryFingerprint {
		return
	}
	if _, seen := registryWarned.LoadOrStore(fp, true); seen {
		return
	}
	slog.WarnContext(context.Background(),
		"empreinte du registre ECS du film INCONNUE — grammaire des composants suspecte "+
			"(mise a jour du jeu ?) ; le film reste decode",
		"empreinte", fp, "empreinte_connue", KnownRegistryFingerprint,
		"blocs", blocks, "slots_non_vides", slots)
}
