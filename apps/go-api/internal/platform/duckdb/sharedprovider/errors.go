package sharedprovider

import "errors"

// ErrProviderClosed est retourné par Get et AcquireWriter quand le provider a
// été fermé (Close appelé, shutdown serveur en cours).
//
// Les handlers HTTP doivent mapper cette erreur en 503 Service Unavailable.
var ErrProviderClosed = errors.New("sharedprovider: provider is closed")

// ErrSwapTimeout est retourné par Get quand l'attente d'un retour en steady
// state RO (pendant un swap RW d'un sync) dépasse readyTimeout.
//
// Les handlers HTTP doivent mapper en 503 + header Retry-After.
// Introduit au commit 3, déclaré ici pour stabilité du contrat d'erreurs.
var ErrSwapTimeout = errors.New("sharedprovider: swap timeout — Get waited too long for RW→RO transition")

// ErrSwapFailed est retourné par Get quand le provider est dans l'état
// StateError suite à un échec de réouverture RO après un sync. Le retry
// interne du provider tente de récupérer ; en attendant les Get échouent.
//
// Introduit au commit 3, déclaré ici pour stabilité du contrat d'erreurs.
var ErrSwapFailed = errors.New("sharedprovider: swap to RO failed — provider in error state")

// ErrDrainTimeout est retourné par AcquireWriter quand la vidange des lecteurs en
// vol dépasse la borne PROPRE au provider (drainTimeout, 5 s par défaut) : un
// lecteur tenait encore le handle RO, le provider est revenu en RO et le writer
// n'a pas été pris. Erreur TRANSITOIRE : un appelant qui tient déjà ses données
// en mémoire peut retenter plus tard (cron du classement mondial, lot B1 du
// 2026-09-26). L'erreur enveloppe aussi la cause (context.DeadlineExceeded).
//
// Un contexte APPELANT annulé ou expiré pendant la vidange ne donne PAS cette
// erreur : c'est une fin demandée par l'appelant, pas un encombrement.
var ErrDrainTimeout = errors.New("sharedprovider: drain timeout — inflight readers still held the RO handle")
