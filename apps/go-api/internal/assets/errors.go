package assets

import "errors"

// ErrNotFound est retourné quand un asset n'existe pas (ni en local ni en upstream).
var ErrNotFound = errors.New("assets: not found")

// ErrUpstreamUnavailable est retourné quand la source distante est injoignable
// ou retourne une erreur non-404.
var ErrUpstreamUnavailable = errors.New("assets: upstream unavailable")

// ErrUnsupportedKind est retourné quand le Kind est inconnu ou non configuré.
var ErrUnsupportedKind = errors.New("assets: unsupported kind")

// ErrPersistFailed est retourné quand l'écriture FS échoue (espace disque…).
// C'est une erreur bloquante : l'asset ne peut pas être mis en cache.
var ErrPersistFailed = errors.New("assets: persist failed")
