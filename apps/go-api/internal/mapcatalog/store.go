package mapcatalog

// store.go — ECRIRE LE CATALOGUE SANS POUVOIR ABIMER CE QU'IL PORTE DEJA.
//
// LE CONTEXTE QUI IMPOSE CE FICHIER. Jusqu'ici, `map_weapon_pads.json` n'etait ecrit que par une
// CLI, a la main, sur une machine ou rien d'autre ne tournait. Le rattrapage au fetch de films
// l'ecrit desormais A L'EXECUTION, pendant que le serveur LIT le meme fichier. Deux exigences
// en decoulent, et elles sont tenues par le code et non par la discipline de l'appelant :
//
//	ECRITURE ATOMIQUE   fichier temporaire puis `rename`. Un lecteur voit l'ancien fichier ou le
//	                    nouveau, jamais un fichier a moitie ecrit. C'est le motif que la CLI
//	                    utilisait deja (`writeCatalog`) ; il est repris tel quel plutot
//	                    qu'invente.
//	AJOUT SEUL          `AddEntry` ne peut PAS toucher une entree existante : il relit le
//	                    catalogue, REFUSE si la cle est deja la, et n'ecrit que dans le cas
//	                    contraire. Ce n'est pas une consigne, c'est la seule chose que la
//	                    fonction sache faire.
//
// POURQUOI L'AJOUT SEUL EST NON NEGOCIABLE ICI. Les entrees de socles alimentent des chemins
// LIVRES (datation des occupations, tableau de la page match, regles d'autres chantiers). Un
// rattrapage automatique qui reecrirait une entree existante changerait, sans revue, ce que
// l'application sert sur une carte deja jouee. Les cartes dont la source a DERIVE ne sont donc
// pas l'affaire de ce chemin : elles se traitent a la main par `mapopads-build --refresh-drifted`.
//
// LA RELECTURE JUSTE AVANT L'ECRITURE n'est pas un verrou entre processus — rien ici ne
// pretend en etre un. Elle reduit la fenetre, et l'ajout-seul fait que le pire cas est une
// entree perdue (re-tentee au prochain film de la meme carte), jamais une entree ecrasee.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay"
)

// ErrEntryExists dit que la carte est DEJA au catalogue : `AddEntry` ne fait rien.
//
// Ce n'est pas une erreur au sens d'un echec — c'est le cas nominal quand deux films de la
// meme carte arrivent dans le meme lot. L'appelant le compte, il ne s'en alarme pas.
var ErrEntryExists = fmt.Errorf("carte deja au catalogue")

// AddEntry ajoute UNE entree au catalogue si et seulement si sa cle n'y est pas encore.
//
// Rend `ErrEntryExists` quand la carte y est deja — y compris quand elle y est arrivee entre le
// moment ou l'appelant a constate son absence et celui-ci.
func AddEntry(path, mapID string, entry replay.MapWeaponPadsEntry) error {
	cat, err := replay.LoadMapWeaponPads(path)
	if err != nil {
		return fmt.Errorf("catalogue illisible : %w", err)
	}
	if cat.Maps == nil {
		cat.Maps = map[string]replay.MapWeaponPadsEntry{}
	}
	if _, deja := cat.Maps[mapID]; deja {
		return ErrEntryExists
	}
	cat.Maps[mapID] = entry
	return WriteAtomic(cat, path)
}

// WriteAtomic ecrit le catalogue par fichier temporaire puis `rename`.
func WriteAtomic(cat *replay.MapWeaponPadsCatalog, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil { //nolint:gosec // catalogue lisible par tous, comme les autres references du titre
		return err
	}
	return os.Rename(tmp, path)
}
