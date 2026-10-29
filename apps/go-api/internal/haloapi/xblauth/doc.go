// Package xblauth est la chaîne d'authentification Xbox Live de la brique « API Halo » (plan D3) :
// jeton d'accès Microsoft -> User Token XBL -> XSTS -> Spartan -> Clearance, XSTS Xbox Live pour
// RTA, rafraîchissement et code d'autorisation OAuth v2, device code du client Xbox natif,
// résolution gamertag -> xuid (profil Xbox, PeopleHub).
//
// Contrat :
//
//   - SANS ÉTAT : aucune variable de paquet mutable, aucun cache, aucun compteur. Ce qui borne la
//     concurrence (créneau XBL), ce qui retient la provenance d'un jeton et ce qui compte les issues
//     appartient à l'appelant : la brique reçoit des fonctions ([Chaine]) et rend des issues typées
//     ([RefreshOutcome]).
//   - SANS POLITIQUE : la brique ne choisit ni le compte, ni le moment, ni le magasin. Elle ne lit
//     ni l'environnement, ni un fichier, ni `sync_meta` ; le client Azure ([ClientAzure]) et les
//     paramètres du titre ([Descriptor]) lui sont PASSÉS.
//   - AUCUN CLIENT HTTP CONSTRUIT : chaque appel reçoit un [haloapi.Doer]. La garde `netguard` et le
//     délai sont portés par le `Doer` que fournit `platform/auth` ; une erreur
//     [haloapi.ErrDefinitif] traverse sans être réécrite.
//   - IMPORTS : bibliothèque standard et `internal/haloapi` seulement (`archlint`, table des
//     briques internes).
//   - JOURNAL : `slog` avec le `ctx` reçu ; l'identifiant d'événement éventuel est posé par
//     l'appelant dans ce `ctx`. Aucun jeton ni secret n'est journalisé.
//
// L'exploitation (magasin des jetons, mise en commun, boucles, compteurs expvar, lecture des
// identifiants Azure) reste dans `platform/auth`, qui garde ses noms exportés et délègue ici.
package xblauth
