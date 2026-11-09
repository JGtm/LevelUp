# Plan de cloture v7.5 — ce qui reste, dans l'ordre

> Ecrit le 2026-08-14 a la demande de l'utilisateur (« fais un plan pour les points
> restants »). Il ORDONNE le reste et RENVOIE aux plans existants plutot que de les
> dupliquer. Execution sous `plan-execution` ; branche unique `feat/v75` (lots = commits,
> cloture = CI verte au niveau job, UN SEUL merge final = release + tag v7.5.0).

## Plans deja ecrits — a NE PAS reecrire

| sujet | plan | etat |
|---|---|---|
| Monitoring + jobs + **delocalisation** (ouvrier externe) | `.ai/PLAN_MASTER_FILM_KILLFEED_REJEU.md` §PISTE F, L1573-1770 (§1 ouvrier-role, §2 systeme de jobs, §2bis jobs+monitoring dans l'app, §4bis monitoring du travail distant) | CONCEPTION TRANCHEE avec l'utilisateur le 2026-08-02. La partie LOCALE est LIVREE (lot 6 : job `replay_build`, purge cron, fenetre de retention). Reste l'OUVRIER DISTANT = C4 ci-dessous |
| Capacites d'armure : les nommer toutes + montrer l'etat actif | `.ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md` (4 etapes, ecrit le 2026-07-31) | A ACTUALISER avant execution (il precede les mesures du 13-14/08) = A1 ci-dessous |
| Fonds de carte par map_id | `.ai/V7.5/cartes/PLAN_FONDS_PAR_MAP_ID.md` | Phases A/B/C livrees ; le RENDU est refuse au gate = B1 |
| Parite du rejeu (lots 1-7) | `.ai/V7.5/replay2d/PLAN_PARITE_REJEU_2D.md` | Livre, sauf 1.4 (=A1) et 6.5 (=C1) |
| Piste F fonds + kill feed (F1/F2/F3) | `.ai/V7.5/PLAN_PISTE_F_REJEU2D.md` | CLOS, tous items `[x]` |

## A0-bis — DECOUVERTE DU 2026-08-14 a corriger (mesure du superviseur)

- [ ] **Les zones Bastion ne sont PAS toutes neutres** — le lot 4 de la parite a ecrit la
      regle « `team_index` = -1 (neutre) pour TOUTES les zones Bastion » et le calque web
      les dessine donc toutes en encre neutre. MESURE sur les 158 zones du catalogue :
      **team 1 = 47, team 0 = 48, neutre = 63**. Soit **95 zones sur 158 (60 %) qui PORTENT
      une equipe** et sont rendues neutres a l'ecran. A verifier semantiquement (equipe
      proprietaire au depart ? camp de rattachement ?) AVANT de colorer — mais la regle
      ecrite est fausse, et le commentaire qui la porte aussi.
      Autres identifiants disponibles et non exploites sur ces zones : `object_index`
      (present partout, index stable), `instance_id` (non nul sur 30/158), `shape`
      (59 cylindres, 98 boites, 1 sans forme = Salvation, deja traitee), `labels`.
      Il n'existe toujours AUCUN nom ni lettre A/B/C (confirme).

## A — Ce qui attend l'utilisateur (aucun code)

- [ ] A0. **GATES EN ATTENTE**, en une passe : rejeu sur `606d9844` (synchro + morts
      neutres typees), `000d5950` (non-regression fiches/effets/callouts), `64e8adfa`
      (CTF : zones, pulses de capture), `696a9d7c` (Strongholds), un Slayer (aucune zone),
      + ECOUTE des sons (bouton « Son », eteint par defaut). Tant que ce gate n'est pas
      rendu, la parite du rejeu n'est pas prononcee.
- [ ] A1. **EFFET D'EQUIPEMENT ACTIF** — seul point fonctionnel demande et NON livre.
      L'investigation du lot 1.4 a conclu « non-decidable » sur un argument que le
      superviseur CONTESTE : elle exigeait que l'IDENTITE de la capacite voyage avec
      l'evenement d'usage, alors que l'identite est deja connue par ailleurs (capacite
      EQUIPEE lue a l'image-cle, affichee sur la fiche, 4 des 11 nommees). PISTE A MESURER :
      croiser « capacite equipee » (inventaire) x « episode d'usage date » (signal i54,
      67 episodes mesures sur le film temoin) — et lever le doute « usage d'equipement vs
      escalade » par la chute d'energie de capacite (i56), qui est deja la condition de
      reprise ecrite au registre. Le plan detaille existe : `PLAN_CAPACITES_ACTIVES.md`
      (etapes 1 a 4), a ACTUALISER avec les mesures du 13-14/08 avant de le lancer.
      DECISION UTILISATEUR REQUISE : on rouvre, ou on classe post-v7.5 ?

## B — Cartes (le plus gros reste, actuellement EN PAUSE sur decision utilisateur)

- [ ] B1. **33 fonds Forge refuses au gate** (« bouillie/patatoide ») : la cause est
      chiffree et un prototype de regle existe (`INVESTIGATION_BOUILLIE_FORGE_2026-08-13.md`,
      regle « arene par la reference », non-regression PROUVEE : 21/21 fonds valides
      identiques au bit). Reste : le GATE des 7 paires temoins
      (`Desktop/gate_cartes_v75/bouillie_proto/`), puis, s'il passe, le lot de production
      (annexe A du document + re-cuisson des 35 + registre). **BLOQUANT AVANT TAG** tant
      que ces fonds sont publies dans cet etat.
- [ ] B2. Vagabond et Catalyst « a revoir » (verdict utilisateur du 13/08, cas
      particuliers) — a reprendre ENSEMBLE, cf. la ligne dediee du registre.
- [ ] B3. Investigations : **Live Fire** (carte native sans tag `sbsp` — 71 matchs) et
      **Detachment / Argyle** (canevas inconnu au depot `.mvar`). Demande explicite de
      l'utilisateur (« ça me parait etrange, faut faire des investigations »).
- [ ] B4. Reliquat Forge < 9 matchs (~20 cartes) : la chaine s'applique telle quelle,
      commande de reprise ecrite au registre.

## C — Donnees et generation (aucune n'est urgente ; l'utilisateur a refuse les runs de masse)

- [ ] C1. **`backfill-killsource`** (item 6.5, `[!]`) : la couverture arme-du-kill est a
      0-5 % sur les matchs recents ; le kill feed et les effets de mort en dependent.
      Fenetre SERVEUR ARRETE. C'est LE run qui ameliore le plus le rendu a l'ecran.
- [ ] C2. **`backfill-replay`** (~928 films, ~8 h, ~2 Go) : le contrat d'artefact est
      stabilise (schema 5) depuis le lot 7, donc plus aucun risque de re-cuisson. Sans lui,
      seuls ~23 matchs ont un rejeu.
- [ ] C3. Rattrapage des films encore telechargeables (~989 matchs sans film) : les films
      EXPIRENT cote serveur (29,3 % deja perdus) — chaque jour d'attente en perd. Le pont
      disque du lot 6 les archive desormais au fil de l'eau, mais le retard, lui, ne se
      rattrape que par un run.
- [ ] C4. **Ouvrier distant + file durable + heartbeat** (delocalisation) : conception
      faite (master plan §1/§2/§2bis/§4bis), execution NON commencee. Acte POST-TAG : le
      rejeu est OFF en prod (garde local), donc personne ne produit d'artefact cote serveur
      tant que la feature n'est pas activee.

## D — Hygiene de cloture (registre) puis release

- [ ] D1. Items d'hygiene deja listes au registre : 7 libelles de mode non resolus,
      litteral `film_chunks`/`film_manifests` en dur dans ~7 CLI (factorisation +
      garde-rail grep), `loadGameVariant` qui rend `0, nil` sans tenter le chemin
      historique, interpretation canonique de `steaktacularMedalIDForTitle`.
- [ ] D2. Registre : re-statuer chaque ligne ouverte (83 lignes non barrees au 14/08) —
      sortir ce qui est fait, dater ce qui reste.
- [x] D3. Containment : `clockOffsets`. **CLOS le 2026-08-25** : mesure elargie -10s->-60s
      sur 8 films = le point etait DEJA resolu le 2026-08-14 (correction exacte `originMS`
      en production, pas un balayage) ; les 3 films a la borne trouvent un pic interieur,
      rattachement 57,2 % vs temoin plat 13,4 % (x4,3). Le bloquant du containment n'a
      jamais ete l'horloge : c'est le taux d'attribution (40,9 %, 64,1 % au mieux vs seuil
      80 %) et l'absence d'oracle de justesse a 95 %. Oracle de justesse (Vagabond) : reste
      la vraie porte de reprise, hors v7.5. Registre L13-14 statue.
- [ ] D4. `make gate-push` complet, puis **UN SEUL merge** `feat/v75` -> `main` (= deploiement
      prod automatique : PREVENIR l'utilisateur avant) et **tag v7.5.0**.
      Note deploiement : au premier demarrage prod, le snapshot est refuse -> lecture live
      jusqu'au premier cut ; les fonds de carte arrivent avec ce merge (ils sont versionnes).

## Ordre recommande (avis du superviseur)

1. **A0** (gates) — rien ne se prononce sans eux, et ils ne coutent que du temps utilisateur.
2. **C1** (`backfill-killsource`) — meilleur rapport rendu/effort, et il conditionne la
   qualite de ce qui vient d'etre livre.
3. **B1** (gate bouillie puis production) — le seul vrai bloquant technique avant tag.
4. **A1** (equipement actif) si l'utilisateur rouvre, sinon post-v7.5.
5. **C2/C3** (runs de masse) quand une fenetre serveur-arrete est possible.
6. **D1/D2/D3** puis **D4** (merge + tag).
7. **B3/B4** et **C4** : post-tag, sans regret.

## Questions ouvertes a trancher (utilisateur)

1. **A1** : rouvrir l'effet d'equipement actif avec le croisement propose, ou classer ?
2. **B** : les cartes restent-elles « en pause » (donc les 33 fonds refuses partent-ils du
   depot avant le tag, ou le tag attend-il leur correction) ?
3. **D3** : lance-t-on le calcul containment (~1 h) avant le tag ?
4. **C** : quelle fenetre pour les runs serveur-arrete ?
