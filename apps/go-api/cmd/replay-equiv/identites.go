package main

// identites.go — LES FAITS FIGES PORTENT DES XUIDS FICTIFS, LE PARC EN PORTE DE REELS.
//
// Les `<short8>.facts.json` du dossier d'equivalence sont versionnes avec les references ; leurs
// xuids sont FICTIFS (`internal/identitefictive`, registre `libs/film/testdata/IDENTITES.tsv`).
// Les films qu'ils accompagnent, eux, sont ceux du parc : leurs octets portent les xuids REELS. Une
// cuisson qui mettrait des faits fictifs en face d'un film reel n'apparierait plus un seul joueur,
// et ses digests divergeraient de toute reference sans qu'aucun decodage n'ait change.
//
// Le harnais RETRADUIT donc les faits avant de cuire : chaque xuid fictif redevient l'original que
// la liste des originaux lui donne. Cette liste vit HORS du depot (elle annulerait l'objectif des
// fictifs) ; le format est celui de `cmd/anonymise-gamertags` (un xuid decimal par ligne), et elle
// se donne par `-originaux` ou `LEVELUP_IDENTITES_ORIGINALES`. La retraduction ne change aucun
// octet de film : les references restent celles de la cuisson des films reels.
//
// Sans liste, des faits fictifs sont REFUSES (message nommant le remede) plutot que cuits : le
// registre versionne dit si un xuid est fictif, la liste dit de qui il tient lieu.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/identitefictive"
	"levelup/go-api/internal/replaybuild"
)

// envIdentitesOriginales est la variable qui porte le chemin de la liste des xuids d'origine.
const envIdentitesOriginales = "LEVELUP_IDENTITES_ORIGINALES"

// retraducteur donne, pour un xuid fictif, l'original dont il tient lieu.
type retraducteur map[uint64]uint64

// lireRetraducteur lit la liste des originaux (un xuid decimal par ligne ; lignes vides et `#`
// ignorees) et rend la table inverse. Deux originaux de meme fictif sont refuses : l'inverse ne
// serait plus une fonction.
func lireRetraducteur(r io.Reader) (retraducteur, error) {
	inverse := retraducteur{}
	sc := bufio.NewScanner(r)
	for ligne := 1; sc.Scan(); ligne++ {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		reel, err := strconv.ParseUint(l, 10, 64)
		if err != nil || !identitefictive.DansLaPlage(reel) {
			return nil, fmt.Errorf("ligne %d : %q n'est pas un xuid de la plage", ligne, l)
		}
		fictif := identitefictive.Xuid(reel)
		if autre, dup := inverse[fictif]; dup && autre != reel {
			return nil, fmt.Errorf("ligne %d : %d et %d rendent le meme xuid fictif %d", ligne, reel, autre, fictif)
		}
		inverse[fictif] = reel
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("lecture de la liste des originaux : %w", err)
	}
	if len(inverse) == 0 {
		return nil, errors.New("liste des originaux vide")
	}
	return inverse, nil
}

// chargerRetraducteur lit la liste a `chemin`. Un chemin vide rend une table vide : rien ne se
// retraduit, et des faits fictifs seront refuses par [retraduireFaits].
func chargerRetraducteur(chemin string) (retraducteur, error) {
	if chemin == "" {
		return retraducteur{}, nil
	}
	f, err := os.Open(chemin) //nolint:gosec // chemin fourni par l'operateur (drapeau, environnement)
	if err != nil {
		return nil, fmt.Errorf("liste des originaux (%s) : %w", chemin, err)
	}
	defer func() { _ = f.Close() }()
	inv, err := lireRetraducteur(f)
	if err != nil {
		return nil, fmt.Errorf("liste des originaux (%s) : %w", chemin, err)
	}
	return inv, nil
}

// registreFictifs lit le registre versionne des xuids fictifs (`libs/film/testdata/IDENTITES.tsv`).
func registreFictifs(repoRoot string) (map[uint64]bool, error) {
	chemin := filepath.Join(repoRoot, "libs", "film", "testdata", "IDENTITES.tsv")
	f, err := os.Open(chemin) //nolint:gosec // chemin fixe sous la racine du depot
	if err != nil {
		return nil, fmt.Errorf("registre des identifiants fictifs : %w", err)
	}
	defer func() { _ = f.Close() }()
	out := map[uint64]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		champ, _, _ := strings.Cut(l, "\t")
		if v, err := strconv.ParseUint(champ, 10, 64); err == nil {
			out[v] = true
		}
	}
	return out, sc.Err()
}

// retraduireFaits remet les xuids REELS dans les faits. `r` vide : les faits doivent deja etre
// reels, et un xuid du registre des fictifs est refuse. `r` donnee : tout xuid fictif connu de la
// table est remplace, et un xuid fictif que la table ne connait pas est refuse — il fausserait la
// cuisson sans le dire.
func retraduireFaits(faits *replaybuild.FactsFile, r retraducteur, fictifs map[uint64]bool) (int, error) {
	n := 0
	for i := range faits.Players {
		xuid, err := strconv.ParseUint(faits.Players[i].XUID, 10, 64)
		if err != nil || !fictifs[xuid] {
			continue // un xuid illisible ou hors registre est laisse tel quel : ce sont des faits reels
		}
		reel, ok := r[xuid]
		if !ok {
			return n, fmt.Errorf("le xuid fictif %d des faits n'a pas d'original : donner la liste des originaux "+
				"(-originaux ou %s)", xuid, envIdentitesOriginales)
		}
		faits.Players[i].XUID = strconv.FormatUint(reel, 10)
		n++
	}
	return n, nil
}

// avecOriginaux ajoute `-originaux` a la ligne de commande d'un enfant quand le parent en porte un.
func avecOriginaux(o options, args []string) []string {
	if o.originaux == "" {
		return args
	}
	return append(args, "-originaux", o.originaux)
}

// retraduireLesFaits charge la liste des originaux et le registre des fictifs, puis retraduit les
// faits. Rend le nombre de xuids retraduits.
func retraduireLesFaits(faits *replaybuild.FactsFile, o options) (int, error) {
	inverse, err := chargerRetraducteur(o.originaux)
	if err != nil {
		return 0, err
	}
	fictifs, err := registreFictifs(o.repoRoot)
	if err != nil {
		return 0, err
	}
	return retraduireFaits(faits, inverse, fictifs)
}
