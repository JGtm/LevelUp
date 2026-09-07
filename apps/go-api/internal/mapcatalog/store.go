package mapcatalog

// store.go — ECRIRE LE CATALOGUE SANS POUVOIR ABIMER CE QU'IL PORTE DEJA.
//
// LE CONTEXTE QUI IMPOSE CE FICHIER. Jusqu'ici, `map_weapon_pads.json` n'etait ecrit que par une
// CLI, a la main, sur une machine ou rien d'autre ne tournait. Le rattrapage au fetch de films
// ecrit desormais A L'EXECUTION, pendant que le serveur LIT. Trois exigences en decoulent, et
// elles sont tenues par le code et non par la discipline de l'appelant :
//
//	DEUX FICHIERS       le catalogue VERSIONNE (`reference/map_weapon_pads.json`, suivi par git,
//	                    produit a la main par `cmd/mapopads-build` et relu en revue) et
//	                    l'OVERLAY NON VERSIONNE (`reference/generated/map_weapon_pads.json`,
//	                    ignore par git). LE RUNTIME N'ECRIT QUE L'OVERLAY : `AddOverlayEntry` ne
//	                    sait pas ecrire ailleurs. Correction du 2026-09-05 (constat A0) — avant,
//	                    le rattrapage ecrivait le fichier versionne, que `scripts/deploy.sh`
//	                    (`git reset --hard origin/main`) aurait efface a chaque deploiement.
//	                    La fusion se fait A LA LECTURE (`replay.LoadMapWeaponPadsMerged`), et
//	                    c'est le VERSIONNE qui prime.
//
//	ECRITURE ATOMIQUE   fichier temporaire A NOM UNIQUE puis `rename`. Un lecteur voit l'ancien
//	                    fichier ou le nouveau, jamais un fichier a moitie ecrit.
//
//	                    LE NOM UNIQUE N'EST PAS UN DETAIL, et la premiere version s'est trompee :
//	                    elle ecrivait dans `<chemin>.tmp`, un nom FIXE. Deux ecrivains
//	                    concurrents — la CLI lancee a la main pendant qu'un cycle de sync
//	                    rattrape une carte — ecrivaient alors dans le MEME fichier temporaire,
//	                    et le `rename` du plus rapide publiait un JSON tronque pour TOUS les
//	                    lecteurs. `os.CreateTemp` donne a chaque ecrivain le sien.
//	AJOUT SEUL          `AddOverlayEntry` ne peut PAS toucher une entree existante : il relit
//	                    l'overlay, REFUSE si la cle est deja la, et n'ecrit que dans le cas
//	                    contraire. Ce n'est pas une consigne, c'est la seule chose que la
//	                    fonction sache faire.
//
// POURQUOI L'AJOUT SEUL EST NON NEGOCIABLE ICI. Les entrees de socles alimentent des chemins
// LIVRES (datation des occupations, tableau de la page match, regles d'autres chantiers). Un
// rattrapage automatique qui reecrirait une entree existante changerait, sans revue, ce que
// l'application sert sur une carte deja jouee. Les cartes dont la source a DERIVE ne sont donc
// pas l'affaire de ce chemin : elles se traitent a la main par `mapopads-build --refresh-drifted`.
//
// LA PERTE DE MISE A JOUR EST GARDEE PAR UN VERROU CONSULTATIF, et il a fallu une revue pour
// le voir : l'ajout fait un LIRE-MODIFIER-ECRIRE. Deux ecrivains — la CLI lancee a la main
// pendant qu'un cycle de sync rattrape une carte — pouvaient lire le meme etat et publier
// chacun un fichier SANS la carte de l'autre. C'est exactement le trou que ce lot comble qui
// se rouvrait.
//
// LE VERROU EST UN FICHIER `.lock` cree en O_CREATE|O_EXCL, avec attente bornee et retrait en
// `defer`. Il est CONSULTATIF et assume comme tel : si un processus meurt sans le retirer, le
// suivant force le passage apres le delai et le journalise. Un verrou qui bloquerait pour
// toujours serait pire que le defaut qu'il corrige — un cycle de sync ne doit jamais rester
// pendu sur un catalogue de cartes.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/analysis/replay"
)

// ErrEntryExists dit que la carte est DEJA au catalogue : `AddEntry` ne fait rien.
//
// Ce n'est pas une erreur au sens d'un echec — c'est le cas nominal quand deux films de la
// meme carte arrivent dans le meme lot. L'appelant le compte, il ne s'en alarme pas.
var ErrEntryExists = fmt.Errorf("carte deja au catalogue")

// dureeAttenteVerrou / pasAttenteVerrou : l'attente bornee du verrou d'ecriture.
//
// Deux secondes suffisent tres largement : l'ecriture du catalogue est de l'ordre de la
// dizaine de millisecondes. Au-dela, on considere le verrou ORPHELIN — un processus mort ne
// doit pas geler le rattrapage de tous les suivants.
const (
	dureeAttenteVerrou = 2 * time.Second
	pasAttenteVerrou   = 25 * time.Millisecond
)

// prendreVerrou pose un verrou consultatif a cote du catalogue et rend sa fonction de retrait.
//
// # LE DOSSIER SE CREE AVANT LE VERROU (constat C3 de la revue A-R1, 2026-09-06)
//
// L'overlay d'un titre vit sous `reference/generated/`, un dossier IGNORE PAR GIT : il n'existe
// donc pas sur un checkout neuf ni sur une instance fraichement deployee — c'est l'etat NOMINAL
// du tout premier rattrapage. `os.OpenFile(..., O_CREATE|O_EXCL)` y echouait ENOENT, et la
// boucle ci-dessous ne distinguait pas cette erreur d'un verrou tenu : elle attendait
// `dureeAttenteVerrou` pour rien, journalisait un « verrou tenu trop longtemps » qui MENTAIT
// sur la cause, puis ecrivait SANS exclusion mutuelle. Mesure : 8 ecrivains concurrents sur un
// dossier absent ne conservaient qu'UNE carte sur 8, avec quatre `rename` en echec dur.
//
// # UN VERROU TENU EST UN `EEXIST`, ET RIEN D'AUTRE
//
// Toute autre erreur d'ouverture (droits, dossier disparu entre-temps) n'est pas une attente :
// la reessayer pendant deux secondes puis se tromper de diagnostic est pire que de le dire tout
// de suite. Ces cas passent en force IMMEDIATEMENT, avec un journal qui nomme l'erreur reelle.
//
// Il ne rend JAMAIS d'erreur : au pire il force le passage en le journalisant. Faire echouer
// une ecriture parce qu'un `.lock` traine serait transformer un garde-fou en panne.
func prendreVerrou(chemin string) func() {
	verrou := chemin + ".lock"
	dossier := filepath.Dir(chemin)
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		slog.Warn("mapcatalog: dossier du catalogue non creable — passage force, ecriture sans verrou",
			"dossier", dossier, "err", err,
			"consequence", "une ecriture concurrente pourrait perdre une entree")
		return func() {}
	}
	fin := time.Now().Add(dureeAttenteVerrou)
	for {
		f, err := os.OpenFile(verrou, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(verrou) }
		}
		if !errors.Is(err, fs.ErrExist) {
			slog.Warn("mapcatalog: verrou d'ecriture impossible a poser — passage force",
				"verrou", verrou, "err", err,
				"consequence", "une ecriture concurrente pourrait perdre une entree")
			return func() {}
		}
		if time.Now().After(fin) {
			slog.Warn("mapcatalog: verrou d'ecriture tenu trop longtemps — passage force",
				"verrou", verrou, "attente", dureeAttenteVerrou,
				"consequence", "une ecriture concurrente pourrait perdre une entree")
			return func() {}
		}
		time.Sleep(pasAttenteVerrou)
	}
}

// AddOverlayEntry ajoute UNE entree a l'OVERLAY NON VERSIONNE si sa cle n'y est pas encore.
//
// LE CHEMIN QU'ELLE RECOIT EST CELUI DE L'OVERLAY, JAMAIS CELUI DU CATALOGUE VERSIONNE — c'est
// la correction du 2026-09-05 (constat A0). L'ancienne `AddEntry` ecrivait dans
// `data/titles/{slug}/reference/map_weapon_pads.json`, un fichier SUIVI PAR GIT : en local un
// commit avalait +332 lignes de donnees de reference sans relecture, et en production
// `scripts/deploy.sh` (`git reset --hard origin/main`) aurait efface a chaque deploiement tout
// ce que le runtime avait rattrape. La fonction ne sait plus ecrire ailleurs que dans l'overlay,
// et le garde-rail `archlint/no_runtime_versioned_catalog_write_test.go` interdit qu'on lui
// repasse le chemin versionne.
//
// L'OVERLAY ABSENT EST LE CAS NOMINAL, et c'est la difference de fond avec l'ancienne fonction :
// le premier rattrapage d'un titre le cree (schema courant, `title_slug` renseigne, une carte).
// En creer un de zero ne perd RIEN — le catalogue versionne reste la base, l'overlay ne fait que
// s'y superposer a la lecture (`replay.LoadMapWeaponPadsMerged`). Un overlay CORROMPU, lui, fait
// echouer : l'ecraser en silence effacerait les cartes deja rattrapees.
//
// Rend `ErrEntryExists` quand la carte est deja dans l'overlay — y compris quand elle y est
// arrivee entre le moment ou l'appelant a constate son absence et celui-ci.
//
// LIRE-MODIFIER-ECRIRE SOUS VERROU : sans lui, deux ecrivains concurrents publiaient chacun un
// fichier sans la carte de l'autre.
func AddOverlayEntry(overlay, titleSlug, mapID string, entry replay.MapWeaponPadsEntry) error {
	defer prendreVerrou(overlay)()
	cat, err := chargerOuCreerOverlay(overlay, titleSlug)
	if err != nil {
		return err
	}
	if _, deja := cat.Maps[mapID]; deja {
		return ErrEntryExists
	}
	cat.Maps[mapID] = entry
	return WriteAtomic(cat, overlay)
}

// chargerOuCreerOverlay rend l'overlay existant, ou un overlay VIDE si le fichier n'existe pas
// encore. Toute autre erreur (JSON invalide, version de schema inconnue) remonte : on n'ecrase
// pas un overlay qu'on ne sait pas lire.
func chargerOuCreerOverlay(overlay, titleSlug string) (*replay.MapWeaponPadsCatalog, error) {
	cat, err := replay.LoadMapWeaponPads(overlay)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &replay.MapWeaponPadsCatalog{
			SchemaVersion: replay.MapWeaponPadsSchemaVersion,
			TitleSlug:     titleSlug,
			GeneratedAt:   time.Now().UTC(),
			Maps:          map[string]replay.MapWeaponPadsEntry{},
		}, nil
	case err != nil:
		return nil, fmt.Errorf("overlay du catalogue illisible : %w", err)
	}
	if cat.Maps == nil {
		cat.Maps = map[string]replay.MapWeaponPadsEntry{}
	}
	return cat, nil
}

// WriteAtomic ecrit le catalogue par fichier temporaire A NOM UNIQUE puis `rename`.
func WriteAtomic(cat *replay.MapWeaponPadsCatalog, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	// LE TEMPORAIRE NE DOIT JAMAIS SURVIVRE A UN ECHEC : sans ce nettoyage, chaque erreur
	// laisserait un fichier orphelin de plusieurs centaines de kilo-octets a cote du catalogue.
	defer func() { _ = os.Remove(tmp) }() // no-op apres un rename reussi
	if _, err := f.Write(buf); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Le catalogue est une reference du titre, lisible comme les autres.
	if err := os.Chmod(tmp, 0o644); err != nil { //nolint:gosec // reference publique du titre
		return err
	}
	return os.Rename(tmp, path)
}
