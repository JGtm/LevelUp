package main

// cmd_backfill_killsource_status.go — `levelup backfill-killsource --status` (lot 5.24.3).
//
// # CE QU ELLE EST, ET CE QU ELLE N OUVRE PAS
//
// Elle LIT le fichier d etat et l affiche. Elle n ouvre AUCUNE base, ne joue AUCUNE migration,
// ne prend AUCUN verrou — et ce n est pas une economie, c est la seule facon que ca marche : la
// passe qu on interroge tient le shared en ECRITURE pendant des heures (un seul writer,
// ADR 0013), donc une sous-commande d etat qui l ouvrirait echouerait exactement quand on en a
// besoin.
//
// ELLE AFFICHE UNE FOIS ET SORT. Pas de boucle, pas de rafraichissement : c est a
// l utilisateur de retaper, ou d entourer l appel de `watch` s il veut une boucle. Une commande
// qui ne rend pas la main est une commande qu on ne peut pas mettre dans un script.
//
// ⚠ ELLE NE DIT PAS CE QUI RESTE A FAIRE EN BASE. Le fichier est un AFFICHAGE : la verite de la
// reprise est `decoder_rev`, relue par la passe a chaque demarrage. Un fichier absent, vieux ou
// supprime ne change rien a ce que la prochaine passe fera.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// afficherEtatDeLaPasse lit le fichier d etat du titre et l ecrit en clair.
func afficherEtatDeLaPasse(chemin string) error {
	raw, err := os.ReadFile(chemin)
	if os.IsNotExist(err) {
		fmt.Printf("aucun etat de passe pour ce titre (%s)\n", chemin)
		fmt.Println("  — soit aucune passe n a tourne depuis l arrivee de --status,")
		fmt.Println("  — soit le fichier a ete supprime : cela ne change RIEN a ce que la")
		fmt.Println("    prochaine passe fera (la reprise se decide sur `decoder_rev`, en base).")
		return nil
	}
	if err != nil {
		return fmt.Errorf("lecture de l etat (%s): %w", chemin, err)
	}
	var e etatDeLaPasse
	if err := json.Unmarshal(raw, &e); err != nil {
		return fmt.Errorf("etat illisible (%s) — fichier tronque ou d une autre version: %w", chemin, err)
	}
	fmt.Print(rendreEtat(e, time.Now()))
	return nil
}

// rendreEtat : le texte. Separe de la lecture pour etre teste sans fichier.
func rendreEtat(e etatDeLaPasse, maintenant time.Time) string {
	var b strings.Builder
	age := maintenant.Sub(e.MiseAJourA)
	fmt.Fprintf(&b, "%s [%s] — phase %s, PID %d\n", e.Commande, e.Titre, e.Phase, e.PID)
	fmt.Fprintf(&b, "  demarree     %s (il y a %s)\n",
		e.DemarreeA.Format(time.RFC3339), time.Duration(e.EcouleeS*float64(time.Second)).Round(time.Second))
	fmt.Fprintf(&b, "  mise a jour  %s (il y a %s)%s\n",
		e.MiseAJourA.Format(time.RFC3339), age.Round(time.Second), avertissementDAge(e, age))
	fmt.Fprintf(&b, "  revisions    morts %s | isolement %s%s\n",
		e.RevisionMorts, e.RevisionIsol, siForce(e.Force))
	if e.Interrompue != "" {
		fmt.Fprintf(&b, "  INTERROMPUE  %s — relancer LA MEME commande reprend au dernier etat\n", e.Interrompue)
	}
	b.WriteString(rendreFilms(e))
	if e.Credit.AExaminer > 0 || e.Phase == phaseCredit {
		fmt.Fprintf(&b, "  credit       %d / %d matchs examines%s\n",
			e.Credit.Examines, e.Credit.AExaminer, resteSi(e.Credit.RestantS))
	}
	return b.String()
}

// rendreFilms : le bloc de la passe des films.
func rendreFilms(e etatDeLaPasse) string {
	f := e.Films
	var b strings.Builder
	fmt.Fprintf(&b, "  films        %d / %d traites — %d chunks / %d\n",
		f.Traites, f.AFaire, f.ChunksFaits, f.ChunksAFaire)
	fmt.Fprintf(&b, "               %d ecrits (%d morts), %d sans film, %d sans kill-feed, "+
		"%d cle inconnue, %d abandons sur delai, %d erreurs\n",
		f.Ecrits, f.Morts, f.SansFilm, f.SansKillFeed, f.CleInconnue, f.AbandonsDelai, f.Erreurs)
	fmt.Fprintf(&b, "               %d deja a jour au demarrage (sautes : c est la REPRISE, et "+
		"elle se decide en base)\n", e.RepriseDe)
	fmt.Fprintf(&b, "               %d ouvrier(s), %.2f films/min, %.3f s/chunk mesure%s\n",
		f.Ouvriers, f.DebitParMin, f.CoutParChunkS, resteSi(f.RestantS))
	if f.FinEstimeeA != nil {
		fmt.Fprintf(&b, "               fin estimee vers %s\n", f.FinEstimeeA.Format(time.RFC3339))
	}
	if f.DernierFini != nil {
		fmt.Fprintf(&b, "  dernier fini %s (%d chunks) en %.2f s — %s\n",
			f.DernierFini.MatchID, f.DernierFini.Chunks, f.DernierFini.DureeS, f.DernierFini.Resultat)
	}
	for _, c := range f.EnCours {
		fmt.Fprintf(&b, "  EN COURS     %s (%d chunks) depuis %.1f s\n", c.MatchID, c.Chunks, c.DepuisS)
	}
	return b.String()
}

// seuilDEtatPerime : au-dela, l etat ne decrit probablement plus une passe vivante.
//
// DIX MINUTES, ET LA BORNE EST CELLE DU PIRE FILM CONNU : le plus gros du corpus coute ~52 s
// (mesure 5.24.1) et la limite par match du collecteur est de 45 min. Un etat plus vieux que dix
// minutes n est donc ni « entre deux films » ni « sur un gros » : soit la passe est morte, soit
// elle est sur le cas pathologique que la limite par match existe pour borner. Dans les deux cas
// il faut le DIRE, pas laisser lire un chiffre perime pour un chiffre courant.
const seuilDEtatPerime = 10 * time.Minute

// avertissementDAge : le fichier decrit-il encore une passe vivante ?
func avertissementDAge(e etatDeLaPasse, age time.Duration) string {
	if e.Phase == phaseTerminee || age < seuilDEtatPerime {
		return ""
	}
	return fmt.Sprintf("  ⚠ AUCUNE MISE A JOUR DEPUIS %s — la passe est probablement morte, "+
		"ou sur un film pathologique (limite par match : 45 min)", age.Round(time.Second))
}

func siForce(force bool) string {
	if force {
		return " | --force (tout est redecode)"
	}
	return ""
}

func resteSi(s float64) string {
	if s <= 0 {
		return ""
	}
	return fmt.Sprintf(" — reste ~%s", time.Duration(s*float64(time.Second)).Round(time.Second))
}
