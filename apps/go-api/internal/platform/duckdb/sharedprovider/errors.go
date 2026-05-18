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
