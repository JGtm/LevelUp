package pool

import (
	"context"
	"log/slog"
)

// bootResolveConcurrency borne le nombre de comptes résolus en même temps par NewPool.
//
// Une résolution enchaîne cinq allers-retours réseau (refresh OAuth, XBL user, XSTS,
// Spartan, Clearance) : en série, la construction du pool attend la SOMME des comptes ;
// en parallèle, à peu près le plus lent. L'étape XBL user garde sa propre borne,
// process-wide (sémaphore de internal/platform/auth/halo_exchange.go), contre le 429
// « currentRequests » de user.auth.xboxlive.com.
const bootResolveConcurrency = 6

// resolvedSource : une source et les tokens obtenus pour elle.
type resolvedSource struct {
	src      CredentialSource
	resolved *ResolvedTokens
}

// bootOutcome : issue de la résolution d'une source. launched=false : source jamais
// lancée (issue vide, ni succès ni échec).
type bootOutcome struct {
	launched bool
	resolved *ResolvedTokens
	err      error
}

// bootDone : issue d'une source, avec son rang dans la liste triée.
type bootDone struct {
	idx     int
	outcome bootOutcome
}

// resolveBootSources rend les sources résolues de sorted, dans l'ordre de sorted, au plus
// maxSize (0 = sans plafond ; NewPool ramène un plafond négatif à 0). Une source en échec
// est journalisée et sautée ; elle ne consomme pas le plafond. Une source jamais lancée
// n'est jamais rendue.
//
// Le résultat est celui de la boucle en série « résoudre chaque source dans l'ordre
// jusqu'à maxSize succès » : mêmes sources tentées, mêmes slots, même ordre. Seule la durée
// change (cf. runBootResolutions).
func resolveBootSources(ctx context.Context, resolver Resolver, sorted []CredentialSource, maxSize int) []resolvedSource {
	outcomes := runBootResolutions(ctx, resolver, sorted, maxSize)
	out := make([]resolvedSource, 0, len(sorted))
	for i, src := range sorted {
		if maxSize > 0 && len(out) == maxSize {
			break
		}
		o := outcomes[i]
		// Les sources sont lancées dans l'ordre : la première jamais lancée clôt la liste
		// (plafond atteint par les précédentes).
		if !o.launched {
			break
		}
		if o.err != nil {
			slog.WarnContext(ctx, "pool: impossible de résoudre token au boot, skip slot",
				"gamertag", src.Gamertag, "err", o.err)
			continue
		}
		out = append(out, resolvedSource{src: src, resolved: o.resolved})
	}
	return out
}

// runBootResolutions résout les sources de sorted en parallèle et rend l'issue de chacune,
// au même rang. Une source jamais lancée garde une issue vide (launched=false).
//
// Contrat :
//   - les sources sont LANCÉES dans l'ordre de sorted ;
//   - la source i n'est lancée que si les sources 0..i-1 ne peuvent pas atteindre maxSize
//     succès, même en comptant comme réussies toutes celles encore en vol : l'ensemble des
//     sources tentées est exactement celui de la boucle en série, aucune résolution réseau
//     de plus (maxSize = 0 : toutes les sources sont tentées ; maxSize < 0 : aucune) ;
//   - deux sources qui portent le même refresh token sont résolues l'une APRÈS l'autre,
//     dans l'ordre de sorted, jamais en même temps : Microsoft fait tourner le jeton à
//     chaque usage, deux échanges concurrents du même jeton en perdraient la rotation ;
//   - au plus bootResolveConcurrency résolutions en vol ; la fonction ne rend la main
//     qu'une fois toutes les résolutions lancées terminées (aucune goroutine ne lui
//     survit).
//
// Le Resolver doit accepter des appels concurrents pour des comptes distincts.
func runBootResolutions(ctx context.Context, resolver Resolver, sorted []CredentialSource, maxSize int) []bootOutcome {
	outcomes := make([]bootOutcome, len(sorted))
	sameTokenBefore := previousSameRefreshToken(sorted)
	finished := make([]bool, len(sorted))
	results := make(chan bootDone)

	next, inFlight, succeeded := 0, 0, 0
	for {
		for next < len(sorted) && inFlight < bootResolveConcurrency &&
			(maxSize == 0 || succeeded+inFlight < maxSize) &&
			(sameTokenBefore[next] < 0 || finished[sameTokenBefore[next]]) {
			go func(i int) {
				resolved, err := resolver.Resolve(ctx, sorted[i])
				results <- bootDone{idx: i, outcome: bootOutcome{launched: true, resolved: resolved, err: err}}
			}(next)
			next++
			inFlight++
		}
		// Plus rien en vol : soit tout est lancé, soit le plafond est atteint (une source
		// bloquée par un jeton partagé attend une source EN VOL, donc inFlight > 0).
		if inFlight == 0 {
			return outcomes
		}
		done := <-results
		inFlight--
		finished[done.idx] = true
		outcomes[done.idx] = done.outcome
		if done.outcome.err == nil {
			succeeded++
		}
	}
}

// previousSameRefreshToken rend, pour chaque source, le rang de la dernière source
// précédente qui porte le même refresh token (-1 si aucune). Une source sans refresh token
// ne déclenche aucun échange OAuth : elle n'attend personne.
func previousSameRefreshToken(sorted []CredentialSource) []int {
	prev := make([]int, len(sorted))
	lastByToken := make(map[string]int, len(sorted))
	for i, src := range sorted {
		prev[i] = -1
		if src.RefreshToken == "" {
			continue
		}
		if j, ok := lastByToken[src.RefreshToken]; ok {
			prev[i] = j
		}
		lastByToken[src.RefreshToken] = i
	}
	return prev
}
