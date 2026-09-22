package killcollector

// collector_ouvriers.go — LA PASSE A N OUVRIERS DE DECODAGE (lot 5.24.2, 2026-09-22).
//
// # POURQUOI DES OUVRIERS, ET POURQUOI SEULEMENT MAINTENANT
//
// LA MESURE, PAS L INTUITION (5.24.1, `backfill_cout_integration_test.go`) : sur l echantillon
// du lot, **93 a 99 % du temps d un film est du CPU hors base**. Decodage 69,3 % et passes
// annexes 29,8 % (elles rebalayent les memes chunks) sur l echantillon large ; le chargement des
// chunks et la resolution de carte sont a 0,0 %, l ecriture a 0,6 %, le roster a 0,3 %. Une
// passe qui ne sature qu un coeur laisse donc les autres vides pendant quatre heures — et c est
// la seule chose que ce fichier change.
//
// ⚠ LE COMMENTAIRE QUI L INTERDISAIT ETAIT PERIME, ET SA CORRECTION EST DATEE. `CollectMatches`
// affirmait « les parametres de replication du decodeur sont des globaux de paquet ;
// `killsource.Decode` serialise deja par un verrou ». **Il n existe plus de verrou de paquet et
// plus de global mutable** : la cloture M3 du chantier decodeur (ADR 0034, 2026-09-17) a DEPENSE
// le profil — `ProfilDeBalayage` voyage desormais en argument jusqu a `calibrate` et `runWalk`,
// et le seul reglage global qui restait (`SetInferResyncTargets`) a ete supprime au lot E.2 du
// 2026-09-05 (sa table reste nil). Verifie sur pieces le 2026-09-22 : aucun `Set*` de paquet
// dans `grammar`, aucun `var` mutable de paquet dans `grammar` ni `killsource` hors tables
// constantes, `registryWarned` est une `sync.Map` et le compteur de replis porte son verrou.
// **ET LA PREUVE N EST PAS LE RAISONNEMENT** : `TestOuvriers_MemesLignesQuUnSeulOuvrier` ecrit
// les memes films avec 1 puis avec N ouvriers et compare les quatre tables ligne a ligne.
//
// # UN SEUL ECRIVAIN, ET IL EST NOMME
//
// Les ouvriers decodent ; la base ne se touche que jeton en main (`porte_de_la_base.go`). A tout
// instant AU PLUS UN goroutine parle a la base partagee — lectures comprises. ADR 0013 (un seul
// writer), ADR 0019/0026/0030 (INSERT-only, vues `_latest`, agregats d ecriture) sont intactes :
// ce fichier ne change NI ce qui est ecrit, NI par quel chemin.
//
// # LE PLAFOND MEMOIRE EST UNE MESURE, PAS UN REGLAGE
//
// Le pire film du corpus (`1c4c63c2`, 69 chunks, 92,2 Mio sur disque) culmine a **422 Mio de
// HeapInuse** (5.24.1). [PlafondMemoireOuvriers] refuse au demarrage un nombre d ouvriers dont
// le produit par ce pic depasserait 4 Gio — avec le chiffre, pour que le refus s explique.
//
// # L ORDRE DE DEPART RESTE CELUI DU COUT, L ORDRE D ARRIVEE NE COMPTE PAS
//
// Les films partent du moins cher au plus cher (la selection les trie, et c est ce qui fait
// qu une interruption laisse un resultat presque complet). Ils FINISSENT dans le desordre, et
// cela n a aucune consequence : chaque match est independant, chaque ecriture est append-only, et
// la reprise se decide en base (`decoder_rev`), jamais sur l ordre.

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/observability"
)

// OuvriersParDefaut : le nombre d ouvriers de decodage quand rien n est demande.
//
// TROIS, ET C EST MESURE (5.24.2, poste de developpement 2026-09-22) : trois ouvriers tiennent
// trois fois le pic d un film sous 1,3 Gio, loin du plafond, et le gain observe sur l echantillon
// est presque lineaire — le temps est du CPU pur. Au-dela, la machine est PARTAGEE (un serveur de
// developpement, un gate de corpus) et un quatrieme ouvrier prendrait le coeur de quelqu un
// d autre. Ce n est pas un drapeau qui laisse la feature a moitie eteinte (CLAUDE.md n 11) : les
// ouvriers sont ACTIFS par defaut, `--workers 1` rend la boucle d avant.
const OuvriersParDefaut = 3

// PicMemoireParFilm : le pic de tas mesure sur le PIRE film du corpus (5.24.1).
const PicMemoireParFilm = 422 << 20

// PlafondMemoireDeLaPasse : ce que la passe s autorise a tenir, tous ouvriers confondus.
const PlafondMemoireDeLaPasse = 4 << 30

// OuvriersMaximum : le nombre d ouvriers que le plafond memoire autorise.
func OuvriersMaximum() int { return PlafondMemoireDeLaPasse / PicMemoireParFilm }

// EvenementDeFilm : ce qu un film a donne, une fois fini.
type EvenementDeFilm struct {
	MatchID string
	Outcome KillSourceOutcome
	Morts   int
	Duree   time.Duration
	Err     error
}

// ObservateurDePasse : le suivi d une passe, film par film.
//
// IL EST ICI ET LA PRESENTATION EST AILLEURS. Le collecteur sait QUAND un film part et QUAND il
// arrive ; il ne sait pas ou s ecrit un fichier d etat ni comment se formate une ligne de
// progression — c est la commande qui le sait (`PathResolver`, ADR 0008). Les deux methodes
// peuvent etre appelees DEPUIS PLUSIEURS GOROUTINES : une mise en oeuvre porte son verrou.
type ObservateurDePasse interface {
	FilmDemarre(matchID string, debut time.Time)
	FilmFini(EvenementDeFilm)
}

// AvecObservateur branche le suivi. nil = aucun suivi (le defaut).
func (c *KillSourceCollector) AvecObservateur(o ObservateurDePasse) *KillSourceCollector {
	c.observateur = o
	return c
}

// CollectMatchesOuvriers : la passe multi-matchs, `ouvriers` decodages en parallele.
//
// `ouvriers <= 1` rend EXACTEMENT la boucle en serie ([CollectMatches]) — pas une passe a un
// ouvrier qui lui ressemblerait : la meme fonction, pour que le chemin de reference ne puisse pas
// deriver de son propre garde-fou.
//
// La synthese est totalisee par UN SEUL goroutine (celui-ci), sur le canal des resultats : aucun
// compteur partage, donc aucune course a compter.
func (c *KillSourceCollector) CollectMatchesOuvriers(
	ctx context.Context, matchIDs []string, ouvriers int,
) KillSourceSummary {
	if ouvriers <= 1 {
		return c.CollectMatches(ctx, matchIDs)
	}
	start := time.Now()
	sum := KillSourceSummary{Total: len(matchIDs)}
	slog.InfoContext(ctx, "killsource: passe a plusieurs ouvriers",
		"ouvriers", ouvriers, "total", len(matchIDs))

	travaux := make(chan string)
	resultats := make(chan EvenementDeFilm, ouvriers)

	// LE DISTRIBUTEUR porte les DEUX arrets — le budget de passe et l annulation de l appelant —
	// et il les porte SEUL : les verifier dans chaque ouvrier les ferait s arreter a des
	// endroits differents pour la meme raison, et le compte des « traites » cesserait d etre le
	// compte de ce qui a ete fait.
	go func() {
		defer close(travaux)
		for _, id := range matchIDs {
			if c.budget > 0 && time.Since(start) >= c.budget {
				observability.AddInt(metricBudget, 1)
				slog.InfoContext(ctx, "killsource: budget de passe epuise — le solde repart au cycle suivant",
					"budget", c.budget, "total", len(matchIDs))
				return
			}
			select {
			case travaux <- id:
			case <-ctx.Done():
				slog.InfoContext(ctx, "killsource: passe interrompue par l appelant",
					"total", len(matchIDs))
				return
			}
		}
	}()

	var groupe sync.WaitGroup
	for i := 0; i < ouvriers; i++ {
		groupe.Add(1)
		go func() {
			defer groupe.Done()
			c.ouvrier(ctx, travaux, resultats)
		}()
	}
	go func() {
		groupe.Wait()
		close(resultats)
	}()

	for ev := range resultats {
		comptabiliserFilm(&sum, ev)
	}

	sum.ElapsedTime = time.Since(start)
	slog.InfoContext(ctx, "killsource: passe terminee",
		"ouvriers", ouvriers,
		"total", sum.Total, "ecrits", sum.Written, "films_absents", sum.NoFilm,
		"sans_killfeed", sum.NoKillFeed, "abandons_delai", sum.Timeouts,
		"erreurs", sum.Errors, "capability_absente", sum.NotSupport,
		"ecartes_cle_inconnue", sum.UnknownKey,
		"duration", sum.ElapsedTime)
	return sum
}

// ouvrier : UN decodeur. Il prend des films jusqu a ce que le distributeur ferme la file, et
// rend un evenement PAR FILM — y compris pour un echec, que la synthese doit compter.
func (c *KillSourceCollector) ouvrier(ctx context.Context, travaux <-chan string, resultats chan<- EvenementDeFilm) {
	for id := range travaux {
		debut := time.Now()
		if c.observateur != nil {
			c.observateur.FilmDemarre(id, debut)
		}
		outcome, deaths, err := c.CollectMatch(ctx, id)
		ev := EvenementDeFilm{MatchID: id, Outcome: outcome, Morts: deaths,
			Duree: time.Since(debut), Err: err}
		if err == nil {
			// LES MARQUEURS DE REGISTRE, au meme endroit que dans la boucle en serie : seule
			// l issue d un film les autorise, une erreur n affirme rien (cf. registry_flags.go).
			c.marquerFilm(ctx, id, outcome, deaths)
		}
		if c.observateur != nil {
			c.observateur.FilmFini(ev)
		}
		resultats <- ev
	}
}

// comptabiliserFilm : l issue d UN film dans la synthese. Un seul chemin, un seul appelant.
func comptabiliserFilm(sum *KillSourceSummary, ev EvenementDeFilm) {
	if ev.Err != nil {
		sum.Errors++
		return
	}
	switch ev.Outcome {
	case OutcomeWritten:
		sum.Written++
		sum.Deaths += ev.Morts
	case OutcomeNoFilm:
		sum.NoFilm++
	case OutcomeNoKillFeed:
		sum.NoKillFeed++
	case OutcomeTimeout:
		sum.Timeouts++
	case OutcomeNotSupported:
		sum.NotSupport++
	case OutcomeUnknownKey:
		sum.UnknownKey++
	}
}
