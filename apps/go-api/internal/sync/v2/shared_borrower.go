// Package v2 — shared_borrower.go : la base partagée servie à la découverte (ensemble connu),
// face aux bascules RO↔RW du provider B-swap (ADR 0016).
package v2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// errNoCachedShared : mode legacy (aucun provider), la connexion partagée n'est pas ouverte.
var errNoCachedShared = errors.New("aucune connexion partagée ouverte (mode legacy)")

// errSharedSwapWait : la base partagée est restée sans connexion lisible au-delà de la borne
// d'attente (bascule qui ne se termine pas, réouverture RO en échec).
var errSharedSwapWait = errors.New("base partagée sans connexion lisible au-delà de la borne d'attente")

// sharedBorrowWait borne l'attente d'une connexion lisible pendant une bascule. Elle couvre la
// vidange des lecteurs (5 s au provider) et la réouverture RO qui suit la libération d'un
// écrivain (quelques ms) ; elle ne couvre PAS la tenue de l'écriture, servie par la connexion
// de l'écrivain (cf. SharedBorrower).
const sharedBorrowWait = 15 * time.Second

// sharedBorrowStep : pas de l'attente — relecture de l'état du provider et de la connexion en cache.
const sharedBorrowStep = 50 * time.Millisecond

// SharedSwapSource : le provider B-swap d'une base partagée (`sharedprovider.Provider`), réduit
// à ce que la découverte lit — l'emprunt RO et l'état de la bascule.
type SharedSwapSource interface {
	Get(ctx context.Context) (*sql.DB, func(), error)
	State() sharedprovider.State
}

// SharedBorrower rend l'emprunt de la base partagée servi au KnownLoader.
//
// src nil (kill-switch LEVELUP_USE_SHARED_PROVIDER=0, aucune bascule dans le process) : la
// connexion en cache `cached()` avec un release sans effet ; nil → erreur.
//
// src non nil, selon l'état du provider :
//   - RO : emprunt `src.Get` — le provider compte un lecteur en vol jusqu'au release, une
//     bascule demandée pendant la lecture attend sa fin au lieu de fermer la connexion ;
//   - RW (un écrivain tient la base, parfois des minutes : sync V1 du suivi en direct, attente
//     du film, récupération d'orphelins) : la connexion RW en cache `cached()`, une lecture
//     marche sur un handle RW. Attendre la fin de l'écriture arrêterait tout le cycle ;
//   - vidange, réouverture, erreur, ou RW sans connexion en cache encore (la RW s'ouvre juste
//     après la fermeture de la RO) : attente par pas de sharedBorrowStep, bornée par
//     sharedBorrowWait ; la borne dépassée rend une erreur (base réellement illisible) ;
//   - provider fermé, contexte fini : erreur.
//
// La connexion RW en cache n'est pas empruntée : sa libération par l'écrivain peut la fermer
// sous la lecture. KnownLoader relit alors une fois (cf. knownLoaderV2.LoadKnown).
func SharedBorrower(src SharedSwapSource, cached func() *sql.DB) SharedDBAcquirer {
	return sharedBorrower(src, cached, sharedBorrowWait, sharedBorrowStep)
}

// sharedBorrower : SharedBorrower à borne et pas d'attente explicites.
func sharedBorrower(src SharedSwapSource, cached func() *sql.DB, wait, step time.Duration) SharedDBAcquirer {
	if src == nil {
		return func(context.Context) (*sql.DB, func(), error) {
			db := cached()
			if db == nil {
				return nil, nil, errNoCachedShared
			}
			return db, func() {}, nil
		}
	}
	return func(ctx context.Context) (*sql.DB, func(), error) {
		deadline := time.Now().Add(wait)
		for {
			switch src.State() {
			case sharedprovider.StateClosed:
				return nil, nil, sharedprovider.ErrProviderClosed
			case sharedprovider.StateRO:
				// Bascule commencée entre l'état lu et l'emprunt : Get attend au plus un pas
				// (ErrSwapTimeout) ou rend l'échec de réouverture (ErrSwapFailed) ; on reprend
				// la boucle. Provider fermé, contexte fini : arrêt.
				db, release, err := src.Get(sharedprovider.WithSwapWaitBudget(ctx, step))
				if err == nil {
					return db, release, nil
				}
				if errors.Is(err, sharedprovider.ErrProviderClosed) || ctx.Err() != nil {
					return nil, nil, err
				}
			case sharedprovider.StateRW:
				if db := cached(); db != nil {
					return db, func() {}, nil
				}
			}
			if !time.Now().Before(deadline) {
				return nil, nil, fmt.Errorf("%w (borne %s, état %s)", errSharedSwapWait, wait, src.State())
			}
			if err := sleepCtx(ctx, step); err != nil {
				return nil, nil, err
			}
		}
	}
}

// sleepCtx attend d, interrompu par la fin du contexte (rendue).
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
