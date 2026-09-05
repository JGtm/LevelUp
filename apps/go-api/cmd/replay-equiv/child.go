package main

// child.go — L'ENFANT D'EQUIVALENCE : UN FILM, SES DIGESTS, PUIS IL MEURT.
//
// # LES PROTECTIONS, ARMEES AVANT LE MOINDRE DECODAGE
//
// Verrou solo D'ABORD (un refus doit couter zero decodage), puis priorite basse, puis la
// sentinelle memoire. Le verrou est celui a ATTENTE BORNEE : ce harnais est le gate de chaque
// lot, et un refus sec sur un simple chevauchement avec une cuisson en cours ferait echouer un
// lot entier. Aucune base n'est ouverte ici — la sentinelle a donc le droit de TUER le
// processus (cf. l'en-tete de `internal/filmproc`).
//
// # POURQUOI L'ENFANT ECRIT LE TSV LUI-MEME
//
// Le tube du lanceur FUSIONNE stdout et stderr de l'enfant et les relaie ligne a ligne : il
// porte un journal, pas un canal de donnees. Les digests passent donc par un FICHIER que le
// parent designe (`-out`), jamais par la sortie standard.
//
// Les lignes sont retenues en memoire et ecrites EN UNE FOIS, a la fin, et seulement en cas de
// succes : un fichier partiel laisse par un enfant mort serait compare par le parent comme s'il
// etait complet.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/digest"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/replaybuild"
)

// attenteVerrouMax : combien de temps un enfant attend son tour de decodage avant de refuser
// (PLAN_CUISSON_PERF §3 D7).
const attenteVerrouMax = 10 * time.Minute

// collecteur retient une ligne de digest par etape, DANS L'ORDRE D'APPEL de l'observateur, le
// COMPTE de chaque etape a part — c'est lui qui arme la garde anti-equivalence-vacuante — et la
// VALEUR des etapes qui en portent une lisible.
type collecteur struct {
	lignes  []string
	comptes map[string]int
	// etats porte la valeur des etapes qui rendent une CHAINE (`spawnPointsState` et ses
	// pareilles). Le compte, lui, ne dirait rien d'elles : `digest.Of` rend la LONGUEUR d'une
	// chaine, donc « 15 » la ou l'operateur attend « not_established ».
	etats map[string]string
}

// etape est l'observateur branche sur le constructeur : elle hache la sortie du balayage.
func (c *collecteur) etape(step string, v any) {
	compte, sum := digest.Of(v)
	c.lignes = append(c.lignes, fmt.Sprintf("%s\t%d\t%s", step, compte, sum))
	if c.comptes == nil {
		c.comptes = map[string]int{}
	}
	c.comptes[step] = compte
	if s, ok := v.(string); ok {
		if c.etats == nil {
			c.etats = map[string]string{}
		}
		c.etats[step] = s
	}
}

// enfantEquivalence cuit UN film et ecrit ses digests. Rend un code du protocole filmproc.
func enfantEquivalence(o options) int {
	cacheRoot := title.NewPathResolver(o.repoRoot).CacheRootDir()
	filmproc.LowerOwnPriority(outilNom)
	lock, err := filmproc.AcquireSoloWait(context.Background(), cacheRoot, outilNom, o.film, attenteVerrouMax)
	if err != nil {
		slog.Error("decodage refuse", "err", err, "film", o.film)
		return filmproc.CodePreparation
	}
	defer lock.Release()
	g := filmproc.Arm(outilNom, o.memGiB, func(peak uint64) {
		slog.Error("plafond memoire depasse — equivalence abandonnee",
			"pic_octets", peak, "pic_gio", gio(peak), "film", o.film)
		filmproc.EmitPeak(peak)
		lock.Release()
		os.Exit(filmproc.CodeMemory)
	})
	defer func() {
		g.Disarm()
		filmproc.EmitPeak(g.Peak())
	}()

	lignes, err := digestsDuFilm(o, cacheRoot)
	if err != nil {
		if errors.Is(err, replaybuild.ErrMapNotInCatalog) {
			slog.Warn("film ecarte — carte hors catalogue de bornes", "err", err, "film", o.film)
			return filmproc.CodeSkipped
		}
		slog.Error("equivalence impossible", "err", err, "film", o.film)
		return filmproc.CodeFailed
	}
	// LA LIGNE DE GRAMMAIRE OUVRE LE FICHIER : sans elle, un changement du RENDU de `digest`
	// se lirait chez le parent comme une regression du decodeur (cf. digest/grammar.go).
	if err := ecrireLignes(o.out, append([]string{digest.GrammarLine()}, lignes...)); err != nil {
		slog.Error("ecriture des digests", "err", err, "path", o.out, "film", o.film)
		return filmproc.CodeFailed
	}
	slog.Info("digests ecrits", "film", o.film, "etapes", len(lignes), "path", o.out)
	return filmproc.CodeOK
}

// digestsDuFilm cuit le film et rend une ligne `etape\tcompte\tsha` par etape observee.
func digestsDuFilm(o options, cacheRoot string) ([]string, error) {
	factsPath := filepath.Join(dossierEquivalence(o.repoRoot), o.film+".facts.json")
	faits, err := replaybuild.ReadFactsFile(factsPath)
	if err != nil {
		return nil, err
	}
	b, err := replaybuild.NewBuilder(o.repoRoot, o.titleSlug)
	if err != nil {
		return nil, fmt.Errorf("preparation du builder (titre %s) : %w", o.titleSlug, err)
	}
	var col collecteur
	b.WithObserver(col.etape)
	slog.Info("cuisson d'equivalence", "film", o.film, "match", faits.MatchID,
		"cartes", faits.MapNames, "joueurs", len(faits.Players), "variante", faits.GameVariantName)
	if _, err := b.BuildBytes(faits.MatchID, faits.MapNames,
		filmcache.ChunkDir(cacheRoot, o.film), faits.MatchFacts); err != nil {
		return nil, err
	}
	alerterCatalogueVide(o.film, faits, col.comptes, col.etats)
	return col.lignes, nil
}

// alerterCatalogueVide CRIE quand un digest est celui du VIDE alors que le film avait de quoi le
// remplir. C'EST LA DEFENSE DE FOND CONTRE L'EQUIVALENCE VACUANTE : `replaybuild` degrade
// gracieusement (catalogue d'objectifs absent, carte hors catalogue des socles) en rendant une
// tranche nulle, et `-update` figerait alors des digests de vide qui se compareront pour
// toujours « identiques » a d'autres digests de vide. Aucun ecart ne serait jamais visible.
//
// ELLE NE FAIT PAS ECHOUER L'ENFANT : un film sans zones est LEGITIME (Assassin, CTF, Oddball)
// et un catalogue partiel reste un cas nominal. Le WARN suffit — le pilote lit les journaux de
// la passe d'update, et le journal de l'enfant remonte dans celui du parent.
func alerterCatalogueVide(
	film string, faits replaybuild.FactsFile, comptes map[string]int, etats map[string]string,
) {
	if faits.MapID == "" {
		// Sans map_id, zones et socles sont court-circuites A LA SOURCE et le vide est attendu.
		return
	}
	if comptes["zones"] == 0 && modeAZones(faits.GameVariantName) {
		slog.Warn("zones vides malgre MapID : catalogue d'objectifs absent ?",
			"film", film, "map_id", faits.MapID, "variante", faits.GameVariantName,
			"consequence", "les digests de zoneStates figeraient du vide")
	}
	if comptes["spawnPoints"] == 0 {
		slog.Warn("points d'apparition vides malgre MapID : catalogue des socles absent ?",
			"film", film, "map_id", faits.MapID,
			// L'ETAT EST LA CHAINE RENDUE PAR L'ETAPE (`map_absent` / `not_established` / ...) :
			// c'est elle qui dit POURQUOI les socles manquent. Le compte, lui, ne donnerait que
			// la longueur de cette chaine.
			"etat", etats["spawnPointsState"],
			"consequence", "les ramassages figeraient sans origine")
	}
}

// modeAZones dit si la variante est de celles qui TIENNENT des zones (KOTH, Strongholds et
// famille). L'aiguillage est celui du depot — jamais une seconde table de variantes.
func modeAZones(variante string) bool {
	switch objectiveevents.ObjectiveTypeOf(variante) {
	case objectiveevents.ObjectiveTypeZone, objectiveevents.ObjectiveTypeHill:
		return true
	default:
		return false
	}
}

// ecrireLignes ecrit un fichier texte d'une ligne par entree.
func ecrireLignes(path string, lignes []string) error {
	if path == "" {
		return errors.New("aucun fichier de sortie (-out)")
	}
	contenu := strings.Join(lignes, "\n") + "\n"
	return os.WriteFile(path, []byte(contenu), 0o600)
}
