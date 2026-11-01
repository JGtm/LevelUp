// Package halo5 est le client des services Halo 5: Guardians de la brique « API Halo » : adresses
// des deux hôtes, transport HTTP à politique de reprise injectée, DTO JSON BRUTS (structs miroir du
// JSON reçu), octets des rendus PNG des profils.
//
// Recette du protocole (sonde live 2026-06-19, `cmd/probe-h5`) :
//
//   - hôte stats/CSR/carnage : `spartanstats` ; hôte profils/apparence : `haloplayer` ([Endpoints]) ;
//   - segment de jeu `h5` dans chaque chemin (`/h5/players/{gamertag}/matches`) ;
//   - `X-343-Authorization-Spartan` (posé à chaque requête, même vide) ; JAMAIS de `343-clearance` ;
//   - `User-Agent: cpprestsdk/2.4.0` (exigé côté 343) ; `?auth=st` ajouté à chaque URL ;
//   - identité = GAMERTAG BRUT dans le chemin (pas de `xuid(NNN)`) ;
//   - `Accept-Encoding` jamais posé (gzip transparent du client HTTP) ; les redirections (302 des
//     rendus PNG vers le CDN d'images) sont suivies par le [haloapi.Doer] reçu.
//
// Contrat :
//
//   - SANS ÉTAT : aucune variable de paquet mutable, aucun cache. Un [Client] est une valeur
//     immuable ; ses `With*` rendent une copie. Le jeton ([Auth]) est passé À CHAQUE appel.
//   - SANS POLITIQUE : adresses ([Endpoints]), politique de reprise ([haloapi.Retry], celle de
//     LevelUp : [RetryHalo5]) et limiteur ([haloapi.Limiter]) sont PASSÉS. Les statuts sont rendus
//     typés ([HTTPError]) ; l'appelant décide de ce qu'un 401 ou un 404 veut dire.
//   - AUCUN CLIENT HTTP CONSTRUIT : les requêtes JSON partent par un [haloapi.Doer], les requêtes
//     binaires par un second (chacun porte sa surface de garde chez l'appelant). Une erreur
//     [haloapi.ErrDefinitif] arrête toute reprise.
//   - IMPORTS : bibliothèque standard et `internal/haloapi` (`archlint`, table des briques internes).
//
// Quatre différences de FORME avec le client d'avant la brique (`games/halo_5/client.go`), sans
// effet hors du mode démo ni pour les fenêtres Retry-After en secondes :
//
//   - la [haloapi.Garde] du `Doer` est consultée AVANT le limiteur, une fois au début de chaque appel
//     (l'ancien client la consultait après le limiteur, à chaque essai : une sortie refusée
//     consommait un jeton du limiteur) ; le `Doer` gardé la reconsulte à chaque requête émise ;
//   - l'en-tête Retry-After est lu par [haloapi.ParseRetryAfter] (secondes ET date HTTP) puis
//     plafonné à 10 s par la politique ; l'ancien client ne lisait que les secondes ;
//   - un refus de la garde est marqué [haloapi.ErrDefinitif] (`errors.Is(err, netguard.ErrOffline)`
//     reste vrai, le message est inchangé, mais l'erreur n'est plus égale par `==` à la sentinelle) ;
//   - la garde est consultée avant la construction de l'URL : en mode démo, un identifiant qui ne
//     donne pas d'URL valide rend le refus de la garde, non l'erreur de construction.
//
// La conversion des DTO vers les types canoniques, le limiteur concret, le délai du client HTTP, la
// garde `netguard` et la lecture du jeton dans le contexte vivent dans LevelUp (`games/halo_5`).
package halo5
