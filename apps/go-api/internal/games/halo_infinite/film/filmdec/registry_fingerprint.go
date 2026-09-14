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
// ENTREES NOMMEES : les quatre octets du NIVEAU (u32 LE, entree+0x100) puis les octets du NOM.
// Les entrees de bourrage — nom vide — n'y entrent pas : ce sont elles qui portent le reste du
// bloc de 64 entrees et elles ne disent rien de la grammaire.
//
// LE DOMAINE A CHANGE AU LOT 1.2 (2026-09-14), ET C'EST LE POINT. Il hachait auparavant
// `kind | flags | nom` — deux champs qui n'existent pas : le « kind » etait la queue de
// bourrage du nom voisin et le « flags » le niveau du composant PRECEDENT. Une empreinte qui
// hache un decalage fige le decalage. Elle hache desormais exactement ce que le jeu lit.
//
// POURQUOI ELLE EST CALCULEE A LA LECTURE et pas recalculee a la demande : elle travaille sur
// les OCTETS du chunk, dans les memes bornes que le parse, et `Archetype` ne retient pas les
// octets. L'empreinte se calcule donc pendant l'unique passe qui les voit, et
// `RegistryFingerprint` la rend.
//
// CE QU'ELLE NE FAIT PAS. Elle ne refuse aucun film et ne change aucune largeur : c'est un
// signal, pas une porte.

import (
	"context"
	"hash"
	"hash/fnv"
	"log/slog"
	"sync"
)

// KnownRegistryFingerprint est l'empreinte du registre de REFERENCE — celui sur lequel toute
// la grammaire portee de ce paquet a ete etablie, et celui que decrit `testdata/ecs_table.tsv`
// (50 blocs, 49 archetypes porteurs, 1 067 slots non vides — le « 118 blocs » cite avant le
// lot 3 du plan « percer la trame » etait `len(fichier)/taille_bloc`, un artefact de division
// qui annexait les sections suivantes de chunk_00 au registre).
//
// RECALCULEE LE 2026-09-14 (lot 1.2) SUR LE CADRAGE DU JEU, et non recopiee : elle est la
// somme des entrees de `../killsource/testdata/minibobine_000d5950` sur le domaine
// `niveau | nom`. La valeur precedente (`0x61e492dd4de7fd4e`, lot 0 du 2026-08-17) hachait
// `kind | flags | nom`, et celle que citait l'ancien commentaire de `registry.go`
// (`0xa413610cd08e4355`) hachait « noms + flags » : trois domaines, trois valeurs — aucune
// n'est transposable dans une autre.
//
// CE QUE LA MESURE DU LOT 0 AVAIT TROUVE, ET QUI RESTE VRAI : le registre n'est PAS identique
// sur tous les films. `000d5950` et `64e8adfa` rendent la meme valeur (50 blocs / 1 067
// entrees), mais `06dfe6d9` en rend une autre sur **49 blocs et 1 031 entrees nommees** — 1
// bloc, 1 porteur et 36 composants de moins. La stabilite mesuree au lot table ECS
// (« bit-a-bit identique sur 000d5950, 00502e52, 07aa428d ») vaut DANS UN BUILD, pas entre
// builds. C'est exactement ce que cette empreinte est faite de dire, et l'alerte se declenche
// a bon droit sur ces films : leur grammaire n'est pas celle que la table decrit.
const KnownRegistryFingerprint uint64 = 0x36ca8c3d2a2f9b88

// registryFNV accumule l'empreinte pendant la passe de lecture des blocs. Il travaille sur le
// buffer INFLATE et sur les memes bornes que `parseRegistry` : une entree tronquee en fin de
// buffer est ignoree plutot que hachee a moitie.
type registryFNV struct {
	h     hash.Hash64
	slots int // entrees nommees hachees — le denominateur de l'alerte
}

// registryHasher construit l'accumulateur de l'empreinte.
func registryHasher() *registryFNV { return &registryFNV{h: fnv.New64a()} }

// addEntry ajoute `niveau | nom` d'une entree nommee. `off` est l'octet de l'ENTREE.
func (f *registryFNV) addEntry(data []byte, off int, name string) {
	lv := off + registryEntryLevelOffset
	if lv+4 > len(data) {
		return
	}
	_, _ = f.h.Write(data[lv : lv+4]) // u32 niveau LE, octets du fichier
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
