package killcollector

// porte_de_la_base.go — LE DROIT DE TOUCHER LA BASE PARTAGEE EST UN JETON UNIQUE (lot 5.24.2).
//
// # CE QU ELLE PROTEGE, ET POURQUOI ELLE EXISTE MAINTENANT
//
// Jusqu au lot 5.24 la passe des films etait une boucle : un seul goroutine decodait et ecrivait,
// donc l invariant « un seul writer » (ADR 0013) etait tenu par la forme du code. La mesure
// 5.24.1 dit que 93 a 99 % du temps d un film est du CPU hors base : le lot fait donc decoder
// plusieurs films en parallele. L invariant ne peut plus etre tenu par la forme — il lui faut
// une piece, et la voici.
//
// LA PIECE EST UN CANAL BORNE A UN JETON. Le prendre donne le droit EXCLUSIF de toucher la base
// partagee ; le rendre le cede. A tout instant, AU PLUS UN goroutine parle a la base — lectures
// comprises. La doctrine anti-corruption est donc intacte, mot pour mot : meme `BatchBuilder`,
// memes persisters INSERT-only, un seul ecrivain (ADR 0019/0026/0030).
//
// # POURQUOI PAS UN GOROUTINE ECRIVAIN A QUI ON ENVOIE DES MESSAGES
//
// Ce serait la forme canonique, et elle a ete ecartee sur piece. Le contrat d ecriture du
// collecteur est un LEASE : `acquireShared` rend `(db, release, err)` et l appelant ecrit ENTRE
// les deux, dans son propre code (`write`, `writeShots`, `writePositions`, `writeOpenings`,
// `writeIsolationFacts`, `writeHits`, `marquerFilm` — sept sites). Les transformer en messages
// exigerait de decouper `collect`, `collectPositions` et `collectHits` en « phase qui calcule »
// et « phase qui ecrit », c est-a-dire de reecrire ce que la passe FAIT — exactement ce que le
// perimetre de ce lot interdit. Le jeton donne le MEME invariant (un seul goroutine a la fois)
// sans toucher une ligne de ce qui est ecrit.
//
// # AUCUNE IMBRICATION, ET C EST VERIFIE
//
// Le jeton n est PAS reentrant : un goroutine qui le prendrait deux fois s attendrait lui-meme.
// Les sept sites d ecriture prennent des leases COURTS et DISJOINTS (aucun n en contient un
// autre — verifie sur pieces le 2026-09-22), et les deux lectures gardees (`IdentitiesForMatch`,
// `MapKeysForMatch`) sont appelees hors lease. Pour qu une imbrication future ne se presente pas
// comme un programme qui ne finit jamais, l attente est BORNEE : au-dela de
// `delaiDeLaPorte` elle rend une erreur qui NOMME la cause probable.
//
// # NIL = PASSE-PLAT
//
// Une porte nil ne garde rien : la passe en serie (`CollectMatches`) et les trois chemins de
// production qui ne parallelisent pas (post-sync du serveur, `--online`, tests) gardent
// exactement le comportement qu ils avaient.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
)

// delaiDeLaPorte : l attente maximale du jeton.
//
// CALIBRE SUR LA MESURE 5.24.1, PAS SUR L INTUITION : l ensemble des ecritures d un film coute
// 15 a 235 ms (le pire film du corpus inclus), et les lectures gardees 49 a 60 ms. Huit ouvriers
// en attente derriere le pire film attendraient donc moins de trois secondes. Cinq minutes
// valent plus de cent fois ce pire cas : ce delai ne peut expirer que sur une IMBRICATION (un
// goroutine qui s attend lui-meme), et c est precisement ce qu il est la pour nommer.
const delaiDeLaPorte = 5 * time.Minute

// PorteDeLaBase : le jeton unique qui donne le droit de toucher la base partagee.
type PorteDeLaBase struct {
	jeton chan struct{}
}

// NouvellePorteDeLaBase construit la porte, jeton disponible.
func NouvellePorteDeLaBase() *PorteDeLaBase {
	p := &PorteDeLaBase{jeton: make(chan struct{}, 1)}
	p.jeton <- struct{}{}
	return p
}

// prendre attend le jeton et rend la fonction qui le rend. Porte nil = passe-plat.
func (p *PorteDeLaBase) prendre(ctx context.Context) (func(), error) {
	if p == nil {
		return func() {}, nil
	}
	minuteur := time.NewTimer(delaiDeLaPorte)
	defer minuteur.Stop()
	select {
	case <-p.jeton:
		var rendu bool
		return func() {
			// IDEMPOTENTE : un `defer rendre()` double (ou un rendre() explicite suivi du
			// defer) rendrait sinon un second jeton et ouvrirait la porte a DEUX ecrivains.
			if rendu {
				return
			}
			rendu = true
			p.jeton <- struct{}{}
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-minuteur.C:
		return nil, fmt.Errorf("porte de la base: jeton non rendu au bout de %s — une ecriture "+
			"en imbrique probablement une autre (le jeton n est pas reentrant)", delaiDeLaPorte)
	}
}

// GarderLeWriter enveloppe la fonction de lease RW : le jeton est pris a l acquisition et rendu
// a la liberation. C est la piece qui tient ADR 0013 quand plusieurs ouvriers decodent.
func (p *PorteDeLaBase) GarderLeWriter(fn persist.SharedWriterFn) persist.SharedWriterFn {
	if p == nil || fn == nil {
		return fn
	}
	return func(ctx context.Context) (*sql.DB, func(), error) {
		rendre, err := p.prendre(ctx)
		if err != nil {
			return nil, nil, err
		}
		db, release, err := fn(ctx)
		if err != nil {
			rendre()
			return nil, nil, err
		}
		return db, func() { release(); rendre() }, nil
	}
}

// GarderLeRoster enveloppe la resolution d identites — une LECTURE, gardee elle aussi.
//
// POURQUOI GARDER UNE LECTURE : DuckDB sait servir des lecteurs pendant qu un ecrivain ecrit, et
// la garder n est donc pas une necessite du moteur. C est une necessite de DOCTRINE : l invariant
// que ce depot defend est « un seul goroutine parle a la base », pas « un seul ecrivain a la
// fois et des lecteurs quelque part ». La mesure 5.24.1 dit ce que cela coute — 49 a 60 ms par
// match, 0,3 a 4,6 % du temps d un film : le prix de l invariant le plus simple a verifier.
func (p *PorteDeLaBase) GarderLeRoster(r KillSourceRoster) KillSourceRoster {
	if p == nil || r == nil {
		return r
	}
	return rosterGarde{porte: p, sous: r}
}

// GarderLesCartes enveloppe la resolution de carte (`match_registry`), meme raison.
func (p *PorteDeLaBase) GarderLesCartes(m port.ReplayMapNameRepo) port.ReplayMapNameRepo {
	if p == nil || m == nil {
		return m
	}
	return cartesGardees{porte: p, sous: m}
}

type rosterGarde struct {
	porte *PorteDeLaBase
	sous  KillSourceRoster
}

func (r rosterGarde) IdentitiesForMatch(ctx context.Context, matchID string) (MatchIdentities, error) {
	rendre, err := r.porte.prendre(ctx)
	if err != nil {
		return MatchIdentities{}, fmt.Errorf("roster %s: %w", matchID, err)
	}
	defer rendre()
	return r.sous.IdentitiesForMatch(ctx, matchID)
}

type cartesGardees struct {
	porte *PorteDeLaBase
	sous  port.ReplayMapNameRepo
}

func (c cartesGardees) MapKeysForMatch(ctx context.Context, matchID string) (port.MatchMapKeys, error) {
	rendre, err := c.porte.prendre(ctx)
	if err != nil {
		return port.MatchMapKeys{}, fmt.Errorf("carte du match %s: %w", matchID, err)
	}
	defer rendre()
	return c.sous.MapKeysForMatch(ctx, matchID)
}

func (c cartesGardees) MapKeysForMap(ctx context.Context, mapID string) (port.MatchMapKeys, error) {
	rendre, err := c.porte.prendre(ctx)
	if err != nil {
		return port.MatchMapKeys{}, fmt.Errorf("carte %s: %w", mapID, err)
	}
	defer rendre()
	return c.sous.MapKeysForMap(ctx, mapID)
}
