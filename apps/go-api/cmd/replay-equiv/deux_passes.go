package main

// deux_passes.go — LE TEST S8 : LES DEUX BRANCHES DU MEME COMMIT, COMPAREES A L OCTET
// (item 4.1.3 du PLAN_DECODEUR_FILM, 2026-09-18).
//
// # CE QUE CE MODE EST, ET CE QU IL N EST PAS
//
// Il joue, pour chaque film, DEUX PASSES DU MEME BINAIRE : la premiere DECODE le film (et ecrit
// ses faits), la seconde REJOUE depuis ces faits. Puis il compare les deux fichiers de digests.
//
// CE N EST PAS « la tete contre le TSV fige ». Aucune reference n est lue, aucune n est ecrite :
// les deux cotes de la comparaison sont produits par la meme execution, a quelques secondes
// d intervalle. Il n y a donc RIEN a re-figer, et aucun `-update` reflexe n est possible — ce qui
// est le point : le mode ordinaire de ce harnais compare a un fige, et un fige se regenere par
// accident. Ici, une divergence est une divergence.
//
// # LE VERDICT PORTE SUR LA LIGNE `artifact`, ET SUR ELLE SEULE
//
// C est la decision du lot, et elle a une raison MESUREE : `killsource.Kill.paquet` est un champ
// NON EXPORTE (une coordonnee interne du decodeur), `digest.Of` hache les champs exportes OU NON,
// et le fichier de faits le perd deliberement a l aller-retour. L etape `killsource` du TSV
// DIVERGE donc par construction, sans qu aucun octet du document publie ne change.
//
// Les autres etapes ne sont pas pour autant du bruit : ce sont LES INSTRUMENTS DE LOCALISATION.
// Quand `artifact` diverge, elles disent OU. Quand `artifact` est identique, toute etape
// divergente se CLASSE — forme ou contenu, au type et au champ pres (protocole V14) — et se
// consigne au §5 du plan. Elle ne se regenere JAMAIS.
//
// # L ANTI-EQUIVALENCE-VACUANTE EST CHEZ L ENFANT
//
// La seconde passe doit VRAIMENT avoir relu les faits. Si la fraicheur echouait, elle
// redecoderait en silence et le harnais comparerait « film contre film » — deux passes identiques
// et un vert qui ne prouve rien. L enfant verifie donc l etape `filmFactsRejoue` et REFUSE quand
// la branche servie n est pas celle qu on lui a demandee (cf. `verifierLaBrancheServie`).
//
// # LA MACHINE RESTE SERIALISEE
//
// Un enfant par passe, jamais deux films dans un processus, verrou solo a attente bornee : le
// motif des quatre sinistres RAM ne change pas parce qu on double le nombre de passes.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/replaybuild"
)

// Les deux passes, par leur nom de drapeau. L enfant les recoit par `-passe`.
const (
	passeFilm  = "film"
	passeFaits = "faits"
)

// etapeArtefact : LA SEULE ligne qui porte le verdict.
const etapeArtefact = "artifact"

// bilanDeuxPasses compte ce qu a donne la passe S8.
//
// `divergents` compte les films dont l ARTEFACT differe — les seuls echecs. `classes` compte les
// films dont l artefact est identique mais qui portent au moins une etape divergente : ce sont des
// DECOUVERTES a consigner, pas des regressions.
type bilanDeuxPasses struct {
	identiques, divergents, classes, ecartes, echecs, infra int
}

// parentDeuxPasses joue les deux passes de chaque film et compare. Code non nul des qu un
// artefact diverge, qu un enfant echoue ou que le harnais n a pas pu poser la question.
func parentDeuxPasses(o options) int {
	films, err := listeDesFilms(o)
	if err != nil {
		fmt.Println("corpus illisible :", err)
		return 1
	}
	runner, err := filmproc.NewRunner(o.repoRoot, os.Stdout)
	if err != nil {
		fmt.Println("lanceur :", err)
		return 1
	}
	tmp, nettoyer, err := dossierDesDigests(o.outDir)
	if err != nil {
		fmt.Println("dossier des digests de l'enfant :", err)
		return 1
	}
	defer nettoyer()

	fmt.Printf("S8 — %d film(s), DEUX passes par film (decodage puis rejeu depuis les faits).\n"+
		"Le verdict porte sur la ligne `%s` et sur elle seule ; les autres etapes localisent.\n",
		len(films), etapeArtefact)
	var b bilanDeuxPasses
	for _, film := range films {
		jouerLesDeuxPasses(o, runner, tmp, film, &b)
	}
	fmt.Printf("\nBILAN S8 : %d identique(s) a l octet, %d ARTEFACT DIVERGENT, %d avec ecart(s) "+
		"classe(s), %d ecarte(s), %d echec(s), %d illisible(s) (harnais)\n",
		b.identiques, b.divergents, b.classes, b.ecartes, b.echecs, b.infra)
	if b.divergents+b.echecs+b.infra > 0 {
		return 1
	}
	return 0
}

// jouerLesDeuxPasses lance les deux enfants d un film et compare leurs digests.
func jouerLesDeuxPasses(o options, runner *filmproc.Runner, tmp, film string,
	b *bilanDeuxPasses,
) {
	fmt.Printf("===== %s =====\n", film)
	sorties := map[string]string{}
	var durees [2]time.Duration
	for i, nom := range []string{passeFilm, passeFaits} {
		sortie := filepath.Join(tmp, film+"."+nom+".tsv")
		res := runner.Run(context.Background(), argsEnfantPasse(o, film, nom, sortie))
		durees[i] = res.Dur
		fmt.Printf("  passe %-5s %-12s %9s  pic %5.2f Gio\n",
			nom, res.Issue, res.Dur.Round(time.Millisecond), gio(res.Peak))
		switch res.Issue {
		case filmproc.IssueSkipped:
			b.ecartes++
			fmt.Printf("  %s ECARTE (carte hors catalogue de bornes) — aucune comparaison\n", film)
			return
		case filmproc.IssueOK:
			sorties[nom] = sortie
		default:
			b.echecs++
			fmt.Printf("  %s ECHEC a la passe %s (code %d)", film, nom, res.Code)
			if res.Err != nil {
				fmt.Printf(" : %v", res.Err)
			}
			fmt.Println()
			return
		}
	}
	// LE GAIN SE MESURE, IL NE S ANNONCE PAS : les deux durees sont celles de ce film, sur cette
	// machine, dans cette passe.
	fmt.Printf("  duree : decodage %s, rejeu depuis les faits %s\n",
		durees[0].Round(time.Millisecond), durees[1].Round(time.Millisecond))
	comparerLesDeuxPasses(sorties[passeFilm], sorties[passeFaits], film, b)
}

// comparerLesDeuxPasses compare deux fichiers de digests du MEME film.
func comparerLesDeuxPasses(cheminFilm, cheminFaits, film string, b *bilanDeuxPasses) {
	parFilm, err := etapesParNom(cheminFilm)
	if err != nil {
		b.infra++
		fmt.Printf("  %s ILLISIBLE (passe %s) : %v\n", film, passeFilm, err)
		return
	}
	parFaits, err := etapesParNom(cheminFaits)
	if err != nil {
		b.infra++
		fmt.Printf("  %s ILLISIBLE (passe %s) : %v\n", film, passeFaits, err)
		return
	}
	artefactA, okA := parFilm[etapeArtefact]
	artefactB, okB := parFaits[etapeArtefact]
	if !okA || !okB {
		b.infra++
		fmt.Printf("  %s ILLISIBLE : l etape `%s` manque a l une des deux passes — le verdict "+
			"n a pas de support\n", film, etapeArtefact)
		return
	}
	ecarts := etapesDivergentes(parFilm, parFaits)
	if artefactA != artefactB {
		b.divergents++
		fmt.Printf("  %s ARTEFACT DIVERGENT :\n    decodage %s\n    faits    %s\n",
			film, artefactA, artefactB)
		if len(ecarts) > 0 {
			fmt.Printf("    etapes divergentes (elles LOCALISENT) : %s\n", strings.Join(ecarts, " "))
		}
		return
	}
	if len(ecarts) == 0 {
		b.identiques++
		fmt.Printf("  %s : les deux passes rendent le MEME artefact, et aucune etape ne diverge\n",
			film)
		return
	}
	b.classes++
	fmt.Printf("  %s : artefact IDENTIQUE a l octet ; %d etape(s) divergente(s) A CLASSER "+
		"(forme / contenu, protocole V14) et a consigner au §5 : %s\n",
		film, len(ecarts), strings.Join(ecarts, " "))
}

// etapesDivergentes rend, triees par l ordre attendu, les etapes dont le digest differe entre les
// deux passes — l etape de BRANCHE exclue, puisqu elle DOIT differer (c est son travail).
func etapesDivergentes(a, b map[string]string) []string {
	var out []string
	for _, etape := range etapesAttendues() {
		if etape == etapeDeBranche() {
			continue
		}
		va, oka := a[etape]
		vb, okb := b[etape]
		switch {
		case !oka && !okb:
			continue
		case oka != okb:
			out = append(out, etape+"(absente d une passe)")
		case va != vb:
			out = append(out, etape)
		}
	}
	return out
}

// etapeDeBranche : le nom de l etape qui NOMME la branche servie. Elle DOIT differer entre les
// deux passes — c est son travail — donc elle est exclue de la liste des divergences.
//
// Il vient du code de PRODUCTION (`replaybuild.EtapeRejeuDepuisLesFaits`) : une copie du nom ici
// se serait desynchronisee au premier renommage, et le harnais aurait alors compte la branche
// comme une divergence sur les vingt films.
func etapeDeBranche() string { return replaybuild.EtapeRejeuDepuisLesFaits }

// etapesParNom lit un TSV d enfant et rend `etape -> "compte sha"`.
func etapesParNom(chemin string) (map[string]string, error) {
	brut, err := lireLignes(chemin)
	if err != nil {
		return nil, err
	}
	_, lignes, err := detacherGrammaire(brut)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(lignes))
	for _, ligne := range lignes {
		champs := strings.SplitN(ligne, "\t", 3)
		if len(champs) != 3 {
			return nil, fmt.Errorf("ligne de digest malformee : %q", ligne)
		}
		out[champs[0]] = champs[1] + " " + champs[2]
	}
	return out, nil
}

// argsEnfantPasse construit la ligne de commande d un enfant de passe S8.
func argsEnfantPasse(o options, film, nom, sortie string) []string {
	return []string{
		"-child", "-film", film, "-out", sortie, "-passe", nom,
		"-repo-root", o.repoRoot, "-title", o.titleSlug,
		"-mem-gib", strconv.Itoa(o.memGiB),
	}
}
