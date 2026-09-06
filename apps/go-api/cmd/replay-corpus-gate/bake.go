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

// resultatCuisson porte l'artefact produit et le temps qu'il a coute.
type resultatCuisson struct {
	ArtifactPath string
	Duree        time.Duration
}

// bakeTemoin cuit UN temoin dans `workRoot`, avec le binaire `binPath` (replay-build compile a
// la revision voulue). Essaie chaque carte candidate de `facts.MapNames` DANS L'ORDRE (le plus
// fiable au moins fiable — meme regle que replaybuild.Builder.BuildMatch, que le CLI unitaire
// n'expose qu'a un seul --map a la fois) jusqu'a un succes.
func bakeTemoin(binPath, workRoot, lockRoot, titleSlug string, facts replaybuild.FactsFile) (resultatCuisson, error) {
	lock, err := filmproc.AcquireSolo(lockRoot, outilNom, facts.MatchID)
	if err != nil {
		return resultatCuisson{}, fmt.Errorf("verrou de decodage : %w", err)
	}
	defer lock.Release()
	filmproc.LowerOwnPriority(outilNom)

	factsPath, err := ecrireFaitsTemp(workRoot, facts)
	if err != nil {
		return resultatCuisson{}, fmt.Errorf("faits temporaires : %w", err)
	}
	if len(facts.MapNames) == 0 {
		return resultatCuisson{}, fmt.Errorf("aucune carte candidate pour %s", facts.MatchID)
	}

	debut := time.Now()
	var dernierErr error
	for _, mapName := range facts.MapNames {
		if err := cuireUneCarte(binPath, workRoot, titleSlug, mapName, factsPath, facts.MatchID); err != nil {
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
		ArtifactPath: title.NewPathResolver(workRoot).ReplayArtifactPath(titleSlug, facts.MatchID),
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
func cuireUneCarte(binPath, workRoot, titleSlug, mapName, factsPath, matchID string) error {
	cmd := exec.Command(binPath, "--map", mapName, "--title", titleSlug, "--facts", factsPath, matchID) //nolint:gosec // binaire et args construits par ce gate
	cmd.Env = append(os.Environ(), "LEVELUP_REPO_ROOT="+workRoot)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s --map %q : %w\n%s", filepath.Base(binPath), mapName, err, stderr.String())
	}
	return nil
}
