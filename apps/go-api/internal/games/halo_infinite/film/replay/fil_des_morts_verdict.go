package replay

// fil_des_morts_verdict.go — LE VERDICT DE LA LECTURE DU FIL DES MORTS, ET SA CAUSE, SONT DES
// FAITS DU FILM (lot M8 des retours rejeu, 2026-09-24).
//
// # LA BRECHE QUE CE FICHIER FERME
//
// Le lot M5.2 a publie le verdict en `coverage.bridge.deathsFeed` (`read`, `empty`, `unreadable`),
// mais il le portait HORS des entrees du balayage : il ne voyageait pas dans les faits persistes.
// Deux consequences, et les deux faisaient diverger les deux branches de la cuisson :
//
//	LE DOCUMENT      rejoue depuis les faits d un film au fil vide ou illisible, il publiait la
//	                 cle ABSENTE la ou le decodage publiait `empty` ou `unreadable`.
//	L ASSEMBLAGE     `replaybuild` recevait des faits un fil sans erreur : les actions d objectif
//	                 s identifiaient sur un pont sans mort et les frags sous effet actif se
//	                 disaient lus, la ou le decodage refusait les deux.
//
// Le verdict ET la cause voyagent donc dans les faits (complement de la section 1, schema de faits
// 4 — la montee de la vague D, sans montee supplementaire), et la branche des faits reconstruit
// l erreur que la branche du film lit ([VerdictDuFilDesMorts.Erreur]).

import "errors"

// VerdictDuFilDesMorts est ce que la lecture du fil des morts a rendu : son verdict publie et,
// quand elle a echoue, le TEXTE de l erreur.
//
// LA CAUSE EST UN TEXTE, PAS UNE CHAINE D ERREURS : elle franchit un fichier. Les consommateurs de
// l erreur (`replaybuild`) ne font que la journaliser et brancher sur sa presence ; le texte suffit
// aux deux, et il est celui que le decodage aurait journalise.
type VerdictDuFilDesMorts struct {
	// Verdict : [DeathsFeedRead], [DeathsFeedEmpty] ou [DeathsFeedUnreadable] ; vide quand aucune
	// lecture n a eu lieu (entrees fabriquees sans film).
	Verdict string
	// Cause : le texte de l erreur de [ScanDeaths], vide quand la lecture a reussi.
	Cause string
}

// verdictDeLaLecture classe l issue de [ScanDeaths] et garde le texte de son erreur.
func verdictDeLaLecture(deaths []Death, err error) VerdictDuFilDesMorts {
	v := VerdictDuFilDesMorts{Verdict: lectureDuFilDesMorts(deaths, err)}
	if err != nil {
		v.Cause = err.Error()
	}
	return v
}

// Erreur rend l erreur de la lecture, RECONSTRUITE depuis le verdict : nil quand la lecture n en a
// pas rendu, sinon une erreur dont le texte est la cause et que [lectureDuFilDesMorts] reclasse au
// MEME verdict (`errors.Is(err, ErrFilDesMortsSansMort)` pour un fil vide).
func (v VerdictDuFilDesMorts) Erreur() error {
	if v.Cause == "" {
		return nil
	}
	return &erreurDuFilDesMorts{verdict: v.Verdict, cause: v.Cause}
}

// erreurDuFilDesMorts : l erreur de lecture relue depuis les faits.
type erreurDuFilDesMorts struct {
	verdict string
	cause   string
}

func (e *erreurDuFilDesMorts) Error() string { return e.cause }

// Is garde la seule distinction que les lecteurs du verdict font : un fil lu SANS MORT est une
// mesure, toute autre erreur une panne.
func (e *erreurDuFilDesMorts) Is(cible error) bool {
	return e.verdict == DeathsFeedEmpty && errors.Is(cible, ErrFilDesMortsSansMort)
}

// encodeVerdictDuFilDesMorts / decodeVerdictDuFilDesMorts : le verdict dans le COMPLEMENT de la
// section 1, a la suite des occupants du match.
func encodeVerdictDuFilDesMorts(w *gwriter, v VerdictDuFilDesMorts) {
	w.str(v.Verdict)
	w.str(v.Cause)
}

func decodeVerdictDuFilDesMorts(r *greader) VerdictDuFilDesMorts {
	return VerdictDuFilDesMorts{Verdict: r.str(), Cause: r.str()}
}
