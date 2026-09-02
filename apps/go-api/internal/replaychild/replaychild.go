// Package replaychild — LA CUISSON D'UN FILM DANS UN ENFANT BORNE, vue des deux cotes.
//
// # LE DECOUPAGE, ET IL EST LA RAISON D'ETRE DU PAQUET
//
//	le PARENT (serveur)  a deja lu les FAITS en base. Il les serialise, lance l'enfant, recupere
//	                     les OCTETS de l'artefact, et les RANGE lui-meme
//	                     (`replaybuild.StoreArtifact`) — donc le garde anti-regression et la
//	                     publication de l'evenement restent chez lui.
//	l'ENFANT             n'ouvre AUCUNE base. Il lit les faits d'un fichier, les chunks du film
//	                     au cache disque, decode, construit, ecrit les octets dans un fichier
//	                     temporaire, et meurt en rendant un CODE.
//
// POURQUOI L'ENFANT NE TOUCHE PAS LA BASE : le modele mono-processus DuckDB interdit un enfant
// lecteur pendant que le parent tient l'ecriture (ADR 0013/0016). Les faits VOYAGENT donc, et ils
// le peuvent : `port.MatchFacts` porte ses etiquettes JSON par construction, et pese au plus
// quelques kilo-octets (les lignes des joueurs, deux scores, deux chaines).
//
// POURQUOI LES OCTETS PASSENT PAR UN FICHIER ET NON PAR LE TUBE : le tube de `filmproc` relaie
// le JOURNAL de l'enfant, ligne a ligne, et sert deja au protocole du pic memoire. Y faire
// transiter plusieurs mega-octets de JSON melangerait la donnee et la trace, et obligerait a
// distinguer les deux au caractere pres. Un fichier temporaire, supprime par le parent, est
// plus simple et plus sur.
//
// # LE BINAIRE EST LE MEME DES DEUX COTES
//
// `filmproc.NewRunner` re-execute `os.Executable()` : le parent relance SON PROPRE binaire avec
// un drapeau. C'est ce qui evite d'exiger qu'un second binaire (`levelup`) soit present sur
// l'hote — contrainte qui aurait fait tomber toute la piste.
package replaychild

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// Flag : le drapeau qui fait d'un processus l'enfant de cuisson. Il n'est pas destine a la main
// de l'operateur — c'est un protocole interne, pas une commande.
const Flag = "-film-child"

// Request : ce que le parent demande a l'enfant.
type Request struct {
	MatchID   string          `json:"matchId"`
	TitleSlug string          `json:"titleSlug"`
	RepoRoot  string          `json:"repoRoot"`
	MapNames  []string        `json:"mapNames"`
	FilmDir   string          `json:"filmDir"`
	Facts     port.MatchFacts `json:"facts"`
	// OutPath : ou l'enfant depose les octets de l'artefact. C'est le PARENT qui le choisit et
	// qui le supprime — l'enfant n'a pas a nettoyer ce qu'il n'a pas cree.
	OutPath string `json:"outPath"`
}

// IsChild dit si cette ligne de commande fait de nous l'enfant de cuisson.
func IsChild(args []string) bool {
	for _, a := range args {
		if a == Flag {
			return true
		}
	}
	return false
}

// RunChild execute le cote ENFANT et rend le code de sortie du protocole.
//
// IL NE REND JAMAIS D'ERREUR : le code de sortie EST le canal de retour vers le parent. Une
// erreur rendue a `main` sortirait en 1, que le protocole reserve aux morts hors categorie.
//
// LA SENTINELLE EST ARMEE ICI. Elle mene a un arret du processus, ce qui n'est licite que parce
// que cet enfant NE TIENT AUCUN HANDLE D'ECRITURE : il ne connait pas la base, et l'artefact
// final est ecrit par le parent.
func RunChild(args []string) int {
	reqPath := valueOf(args, Flag)
	if reqPath == "" {
		fmt.Fprintf(os.Stderr, "enfant de cuisson : %s attend un chemin de requete\n", Flag)
		return filmproc.CodePreparation
	}
	var req Request
	raw, err := os.ReadFile(reqPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enfant de cuisson : requete illisible : %v\n", err)
		return filmproc.CodePreparation
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(os.Stderr, "enfant de cuisson : requete invalide : %v\n", err)
		return filmproc.CodePreparation
	}

	g := filmproc.Arm("post-sync", filmproc.DefaultLimitGiB, func(peak uint64) {
		// LE PIC PART AVANT LA MORT : `os.Exit` ne joue pas les differes.
		filmproc.EmitPeak(peak)
		fmt.Fprintf(os.Stderr, "enfant de cuisson : plafond memoire depasse (%d octets) — film abandonne\n", peak)
		os.Exit(filmproc.CodeMemory)
	})
	defer func() {
		g.Disarm()
		filmproc.EmitPeak(g.Peak())
	}()

	builder, err := replaybuild.NewBuilder(req.RepoRoot, req.TitleSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enfant de cuisson : builder indisponible : %v\n", err)
		return filmproc.CodePreparation
	}
	built, err := builder.BuildBytes(req.MatchID, req.MapNames, req.FilmDir, req.Facts)
	switch {
	case err == nil:
	case strings.Contains(err.Error(), replaybuild.ErrMapNotInCatalog.Error()):
		// ECHEC VOULU : carte hors catalogue (Forge). Le parent le journalise en debug.
		fmt.Fprintf(os.Stderr, "enfant de cuisson : carte hors catalogue (%v)\n", req.MapNames)
		return filmproc.CodeSkipped
	default:
		fmt.Fprintf(os.Stderr, "enfant de cuisson : construction en echec : %v\n", err)
		return filmproc.CodeFailed
	}
	if err := os.WriteFile(req.OutPath, built.Blob, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "enfant de cuisson : depot des octets impossible : %v\n", err)
		return filmproc.CodeFailed
	}
	fmt.Printf("  %s : %d tracks, %d octets (%s)\n",
		req.MatchID, built.Tracks, len(built.Blob), built.Module)
	return filmproc.CodeOK
}

// valueOf rend la valeur qui suit un drapeau, ou "" — un mini-lecteur d'arguments plutot que
// `flag`, parce que ce processus est AUSSI un serveur : installer un FlagSet complet ici
// entrerait en conflit avec le sien.
func valueOf(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(a, flag+"="); ok {
			return v
		}
	}
	return ""
}

// Result : ce que la cuisson d'UN film rend au parent — les octets ET LA MESURE.
//
// POURQUOI LA MESURE VOYAGE AVEC LES OCTETS (PLAN_CUISSON_PERF §3 D5). Le lanceur mesure deja
// les deux (`filmproc.Result.Dur` et `.Peak`, ce dernier par le protocole du tube), et `Spawn`
// les jetait : le log de succes du post-sync ne pouvait donc rien dire du temps ni de la memoire
// de la cuisson qu'il venait de payer. Les rendre ici est le SEUL chemin par lequel ces deux
// chiffres atteignent l'orchestrateur — un `slog` pose dans l'enfant finirait dans le tube du
// journal, pas dans la ligne de bilan du cycle.
type Result struct {
	// Blob est le document SERIALISE, tel que l'enfant l'a depose.
	Blob []byte
	// Dur est la duree de bout en bout de l'ENFANT (lancement compris), pas celle du seul
	// decodage : c'est ce que le cycle a reellement paye pour ce film.
	Dur time.Duration
	// Peak est le pic memoire de l'enfant en octets, tel qu'il s'est mesure lui-meme. ZERO
	// QUAND L'ENFANT EST MORT AVANT DE POUVOIR SE MESURER — une valeur nulle ne veut donc pas
	// dire « pas de memoire », elle veut dire « pas de mesure ».
	Peak uint64
}

// Spawn est le cote PARENT : il serialise la requete, lance l'enfant borne, et rend les OCTETS
// de l'artefact AVEC leur mesure. Il n'ecrit RIEN a la place canonique — c'est l'appelant qui
// range.
//
// TOUT CE QU'IL CREE, IL LE SUPPRIME : la requete et le depot sont des fichiers temporaires du
// parent. Un enfant tue en vol ne laisse donc rien derriere lui.
func Spawn(ctx context.Context, req Request) (Result, error) {
	dir, err := os.MkdirTemp("", "levelup-filmchild-")
	if err != nil {
		return Result{}, fmt.Errorf("repertoire temporaire: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	req.OutPath = filepath.Join(dir, "artifact.json")
	reqPath := filepath.Join(dir, "request.json")
	blob, err := json.Marshal(req)
	if err != nil {
		return Result{}, fmt.Errorf("serialisation de la requete: %w", err)
	}
	if err := os.WriteFile(reqPath, blob, 0o600); err != nil {
		return Result{}, fmt.Errorf("ecriture de la requete: %w", err)
	}

	runner, err := filmproc.NewRunner(req.RepoRoot, os.Stdout)
	if err != nil {
		return Result{}, err
	}
	res := runner.Run(ctx, []string{Flag, reqPath})
	switch res.Issue {
	case filmproc.IssueOK:
	case filmproc.IssueSkipped:
		return Result{}, fmt.Errorf("%w (candidats: %v)", replaybuild.ErrMapNotInCatalog, req.MapNames)
	case filmproc.IssueMemory:
		return Result{}, fmt.Errorf("cuisson abandonnee : plafond memoire depasse (pic %d octets)", res.Peak)
	default:
		// MORT SUBITE COMPRISE : un enfant tue par l'OS ne doit jamais passer pour un succes.
		return Result{}, fmt.Errorf("cuisson en echec (issue %s, code %d): %v", res.Issue, res.Code, res.Err)
	}
	out, err := os.ReadFile(req.OutPath)
	if err != nil {
		return Result{}, fmt.Errorf("octets de l'artefact illisibles: %w", err)
	}
	if len(out) == 0 {
		return Result{}, fmt.Errorf("l'enfant a rendu un artefact VIDE pour %s", req.MatchID)
	}
	return Result{Blob: out, Dur: res.Dur, Peak: res.Peak}, nil
}
