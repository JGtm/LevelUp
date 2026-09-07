package main

// bake.go — CUIRE UN TEMOIN AVEC UN BINAIRE COMPILE (celui du HEAD, ou celui d'une base), sous
// le verrou de decodage PARTAGE.
//
// # POURQUOI UN SOUS-PROCESSUS, PAS UN IMPORT DIRECT DE internal/replaybuild
//
// La reference par defaut du gate est desormais une CUISSON A LA BASE (une revision Git
// anterieure au HEAD, cf. base.go) — pas le parc. Importer `internal/replaybuild` donnerait
// TOUJOURS le comportement du code AVEC LEQUEL CE GATE EST COMPILE (le HEAD), quelle que soit
// la revision qu'on croit cuire : c'est ce qui rendrait la comparaison HEAD-contre-base
// VACUANTE (les deux cotes cuiraient avec le meme code). La cuisson passe donc TOUJOURS par un
// binaire `cmd/replay-build` COMPILE DEPUIS LA REVISION VOULUE (base.go compile celui de la
// base ; le sourceRoot fournit celui du HEAD) et invoque en sous-processus — la MEME methode
// pour les deux cotes, symetrique par construction.
//
// # LE VERROU RESTE EXTERNE, LE SOUS-PROCESSUS A LE SIEN EN PLUS
//
// `cmd/replay-build` arme sa PROPRE sentinelle et son PROPRE verrou solo, mais sur
// `LEVELUP_REPO_ROOT=workRoot` — une racine de travail JETABLE et PROPRE A CETTE EXECUTION,
// que personne d'autre ne dispute. Ce verrou-la ne protege donc rien contre la concurrence
// MACHINE (un autre outil de cuisson, un autre run du gate) : c'est pourquoi ce fichier prend
// EN PLUS le verrou PARTAGE (`lockRoot` = CacheRootDir du parc, cf. roots.go) avant de lancer
// le sous-processus, et le relache apres — les deux coexistent sans conflit (chemins
// differents), et c'est le verrou externe qui porte la garantie reelle.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/replaybuild"
)

// outilNom : le nom que ce gate porte dans le journal des protections (verrou partage).
const outilNom = "replay-corpus-gate"

// attenteVerrouGate : combien de temps une cuisson attend son tour avant de renoncer — MEME
// REGIME que les trois autres enchaineurs de films de ce depot (cmd_backfill_replay_child.go,
// replay-equiv/child.go, replay-worker/job.go, tous a 10 min). Avant CORPUS-R1 C10,
// `filmproc.AcquireSolo` refusait IMMEDIATEMENT sur un decodage concurrent (worker, post-sync,
// backfill deja en cours) : un gate qui dure 13 a 25 min sortait alors en erreur pour une
// raison etrangere au diff sous revue — rouge FAUX, jamais vert faux, mais un bruit que
// l'attente bornee elimine.
const attenteVerrouGate = 10 * time.Minute

// resultatCuisson porte l'artefact produit et le temps qu'il a coute.
type resultatCuisson struct {
	ArtifactPath string
	Duree        time.Duration
}

// cuissonParams regroupe les chemins d'UNE cuisson — un struct plutot qu'une signature a plus
// de 5 parametres une fois `ctx` ajoute (CLAUDE.md n°5).
type cuissonParams struct {
	BinPath   string // replay-build compile a la revision voulue
	WorkRoot  string
	LockRoot  string
	TitleSlug string
}

// bakeTemoin cuit UN temoin dans `p.WorkRoot`, avec le binaire `p.BinPath`. Essaie chaque carte
// candidate de `facts.MapNames` DANS L'ORDRE (le plus fiable au moins fiable — meme regle que
// replaybuild.Builder.BuildMatch, que le CLI unitaire n'expose qu'a un seul --map a la fois)
// jusqu'a un succes. `ctx` borne l'ATTENTE du verrou partage (attenteVerrouGate) — annule (Ctrl-C
// sur un gate qui dure 13 a 25 min), il interrompt l'attente proprement au lieu de la laisser
// courir jusqu'a son terme.
func bakeTemoin(ctx context.Context, p cuissonParams, facts replaybuild.FactsFile) (resultatCuisson, error) {
	lock, err := filmproc.AcquireSoloWait(ctx, p.LockRoot, outilNom, facts.MatchID, attenteVerrouGate)
	if err != nil {
		return resultatCuisson{}, fmt.Errorf("verrou de decodage : %w", err)
	}
	defer lock.Release()
	filmproc.LowerOwnPriority(outilNom)

	factsPath, err := ecrireFaitsTemp(p.WorkRoot, facts)
	if err != nil {
		return resultatCuisson{}, fmt.Errorf("faits temporaires : %w", err)
	}
	if len(facts.MapNames) == 0 {
		return resultatCuisson{}, fmt.Errorf("aucune carte candidate pour %s", facts.MatchID)
	}

	debut := time.Now()
	var dernierErr error
	for _, mapName := range facts.MapNames {
		if err := cuireUneCarte(ctx, p, mapName, factsPath, facts.MatchID); err != nil {
			dernierErr = err
			continue
		}
		dernierErr = nil
		break
	}
	if dernierErr != nil {
		return resultatCuisson{}, fmt.Errorf("cuisson (cartes essayees %v) : %w", facts.MapNames, dernierErr)
	}

	return resultatCuisson{
		ArtifactPath: title.NewPathResolver(p.WorkRoot).ReplayArtifactPath(p.TitleSlug, facts.MatchID),
		Duree:        time.Since(debut),
	}, nil
}

// ecrireFaitsTemp depose les faits du match dans workRoot, forme attendue par
// `cmd/replay-build --facts` (replaybuild.ReadFactsFile).
func ecrireFaitsTemp(workRoot string, facts replaybuild.FactsFile) (string, error) {
	blob, err := json.Marshal(facts)
	if err != nil {
		return "", fmt.Errorf("serialisation des faits : %w", err)
	}
	path := filepath.Join(workRoot, "facts_"+title.FilmShortMatchID(facts.MatchID)+".json")
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		return "", fmt.Errorf("ecriture des faits (%s) : %w", path, err)
	}
	return path, nil
}

// cuireUneCarte invoque le binaire replay-build pour UNE carte candidate. Le dossier de chunks
// n'est jamais passe explicitement : `stageFilm` les a deja places au chemin que le binaire
// deduit par defaut depuis LEVELUP_REPO_ROOT (filmcache.ChunkDir), et c'est ce chemin qu'il
// faut lui laisser resoudre lui-meme — le repeter ici serait une deuxieme ecriture de la meme
// regle de disposition (filmcache est deja LE point unique, cf. son en-tete).
func cuireUneCarte(ctx context.Context, p cuissonParams, mapName, factsPath, matchID string) error {
	cmd := exec.CommandContext(ctx, p.BinPath, "--map", mapName, "--title", p.TitleSlug, "--facts", factsPath, matchID) //nolint:gosec // binaire et args construits par ce gate
	cmd.Env = append(os.Environ(), "LEVELUP_REPO_ROOT="+p.WorkRoot)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s --map %q : %w\n%s", filepath.Base(p.BinPath), mapName, err, stderr.String())
	}
	return nil
}
