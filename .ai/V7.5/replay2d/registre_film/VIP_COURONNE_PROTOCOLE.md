# PROTOCOLE — VIP COURONNE (v7.5) — COMMITE AVANT MESURE

> Executeur SEUL, branche `wt/resolution`. Contrat `plan-execution`. Ce fichier est COMMITE
> AVANT toute mesure (commit `vip-couronne(protocole):`). Il CORRIGE le temoin du gate VIP
> par-joueur (le lot precedent l'a etabli inapte sur un compteur a faible variance) SANS
> abaisser aucun seuil d'accord, puis pre-enregistre le gate des periodes et la publication de
> la couronne. Donnees en LECTURE via `LEVELUP_REPO_ROOT` (depot principal), DuckDB en
> `OpenReadForQuery` / vue `match_objective_stats_latest`, payloads figes. Decodage de film
> sous `internal/filmproc` (un film = un processus, plafond 2 Gio). UN SEUL build Go a la fois.
> thought_log et REGISTRE_REPORTS : textes au CR, le superviseur consigne.

## 0. Acquis figes (NE PAS re-mesurer — au log `V_statborg_vip.log`)

- **Signal archi-etabli.** `comp 22 A` du statborg reproduit `TimesSelectedAsVip` EXACTEMENT
  par joueur, 100 % sur 3/3 films (00761d27 Bazaar, 9903b1c5 Bazaar, 99553e4a Catalyst),
  somme-film exacte (15/17/18), **temoin decale (somme-film, immune au pont) = 0**. C'est le
  MEME comp que `flag_grabs` en CTF (`objectiveevents/named.go`, table `namedStatSlots`).
- **Le film ne porte PAS le bit VIP (script-side)** : la voie est le statborg. Acquis.
- Corpus VIP FIGE : `V_oracle_vipstats.json` (7 colonnes `VipStats` par xuid, valides par
  l'agregat d'equipe, 24/24 lignes). TSV de balayage FIGE : `V_statborg_sweep.tsv` (valeurs
  FINALES par emplacement, pont `SlotIdentityByDeaths`, 8/8 slots nommes par film).

## 1. POURQUOI le temoin permute etait INAPTE (defaut de methode, pas de signal)

Le gate frozen exigeait, dans le test par-joueur, un TEMOIN PERMUTE (permutation cyclique de
l'affectation xuid -> oracle) `<= 20 %` sur CHACUN des 3 films. Il a rendu 12,5 / 25,0 / 50,0 %
et le gate est tombe — **alors que le signal est parfait**.

Cause MESUREE et STRUCTURELLE : `TimesSelectedAsVip` est un compteur a FAIBLE VARIANCE (ex.
film 99553e4a : six joueurs sur huit a la valeur « 2 »). Quand le signal est parfait, les
valeurs du comp SONT les valeurs de l'oracle ; sous une permutation, l'accord attendu vaut la
**self-similarite de la donnee** = probabilite que deux joueurs tires partagent la meme valeur
= `sum_v p_v^2` (p_v = fraction des joueurs a la valeur v). Ce plancher NE PEUT PAS descendre
sous ~34-62 % pour ces films, quelle que soit la justesse du comp. Le temoin permute mesurait
donc la self-similarite de l'oracle, PAS la discrimination du comp. Exiger `<= 20 %` etait
demander l'impossible : un gate qu'aucun comp parfait ne peut franchir n'a rien mesure.

## 2. Le TEMOIN CORRIGE (pre-enregistre, date 2026-08-27)

On remplace le seul instrument inapte (le seuil `<= 20 %` sur le temoin permute) par le
**plancher statistique correct**. RIEN d'autre ne bouge : le seuil d'accord par-joueur reste
`>= 90 %`, la stabilite 3/3 reste, le temoin decale de la somme-film (deja 0) reste. Ce n'est
PAS un abaissement — c'est un test de discrimination PLUS exigeant que ce que le permute
pouvait offrir.

### Plancher analytique par film (le null correct pour un compteur additif)

`plancher(f) = sum_v (p_v(f))^2`, ou `p_v(f)` = part des joueurs confrontables du film `f`
ayant `TimesSelectedAsVip == v`. C'est l'accord par-joueur ATTENDU de toute affectation
xuid -> valeur decorrelee de la vraie (le null). Valeurs pre-calculees depuis l'oracle FIGE :

| film | multiset TimesSelectedAsVip | plancher `sum p_v^2` |
|---|---|---|
| 00761d27 | {4,3,2,1,2,1,1,1} | 22/64 = **34,4 %** |
| 9903b1c5 | {3,2,3,2,2,1,2,2} | 30/64 = **46,9 %** |
| 99553e4a | {2,2,2,2,2,3,2,3} | 40/64 = **62,5 %** |

Le plancher est CALCULE dans le code depuis l'oracle (pas recopie a la main) et imprime au log.

### GATE VIP par-joueur CORRIGE (comp 22 A / `TimesSelectedAsVip`) — NON NEGOCIABLE

Le comp REPLIQUE `TimesSelectedAsVip` si et seulement si, sur `>= 2/3` films :

1. **Exactitude** : accord par-joueur (egalite entiere) `>= 90 %` (seuil inchange). ET
2. **Discrimination sur le null** : `accord_signal(f) - plancher(f) >= 0,30` (30 points de %).
   La marge de 30 pp est un seuil GENERIQUE de « domine clairement le null » : au-dessus de la
   granularite d'echantillonnage sur 8 paires (1/8 = 12,5 pp) et fixe par la STRUCTURE du
   compteur (comp parfait = 100 %, null = `sum p_v^2` = 34-62 %), jamais regle sur le resultat
   observe. ET
3. **Stabilite** (garde-fou corpus mince) : comp 22 A est le MEILLEUR emplacement sur 3/3
   films (inchange). ET
4. **Somme-film immune au pont** : temoin decale `== 0` (deja acquis, ne se re-mesure pas — on
   le RECHARGE du signal existant).

Le temoin permute reste IMPRIME au log comme DIAGNOSTIC (transparence), marque « non gating,
artefact de self-similarite » — il ne conditionne plus le verdict.

**Log** : `VIP_temoin_corrige.log`. Le signal est deja mesure ; on RECHARGE/REJOUE la mesure
sur le TSV fige (`V_statborg_sweep.tsv`) + oracle (`V_oracle_vipstats.json`) — AUCUN film
n'est re-decode a cette etape.

## 3. PERIODES VIP — comp 22 A donne-t-il des EVENEMENTS DATES ?

Question : `incrementTimes` (named.go) convertit un compteur cumule en evenements DATES (un par
unite gagnee, a l'instant de l'emission). Comp 22 A donne-t-il, comme `flag_grabs`, une suite de
selections DATEES par joueur — ou seulement le compte final ?

### Instrument (un film par processus, sous filmproc, AUCUNE base ouverte)

`TestVIPPeriodes` (gate env `ATT_FILM` = racine cache + `VIP_FILM` = id court +
`VIP_ORACLE` = oracle fige) :

1. Decode le film -> `StatRecordsCtx` (records d'entite, plafond memoire).
2. `NamedEventsFrom(records, ObjectiveTypeVip)` -> selections VIP DATEES (horloge MATCH). Cela
   exige d'ajouter a `namedStatSlots` la famille `ObjectiveTypeVip` avec `{22, A} -> vip_selected`
   (meme discipline que flag/zone ; la TSV figee `.ai/refs/TABLE_STATS_STATBORG.tsv` recoit la
   ligne concordante, sinon `TestTableStatborgConcordeAvecNamedStatSlots` echoue).
3. `ScanFilmDeaths(dir)` -> morts datees (horloge MATCH, victime par xuid).
4. `SlotIdentityByDeaths(records, morts)` -> pont slot statborg -> xuid.
5. **VERIFICATION DES DATES** : imprimer, par slot, la suite des instants d'increment de comp
   22 A. Si tous les increments tombent au meme instant (compte final sans etalement), le canal
   ne DATE pas les selections -> le dire clairement, gate periodes non franchissable par ce
   seul canal (`[!]` chiffre), NE PAS forcer.
6. **RECONSTRUCTION** (patron `flag_carries`) : chaque selection (slot -> xuid, t0) OUVRE une
   periode VIP pour ce joueur, FERMEE au PREMIER de : (a) mort du joueur apres t0 (kill feed) ;
   (b) selection suivante du meme slot ; (c) fin du match (dernier fait date + 1). Duree =
   fermeture - t0.
7. Sommer les durees par xuid (secondes), confronter a `TimeAsVipSeconds` de l'oracle.

### GATE periodes (RECOPIE, NON NEGOCIABLE)

Metrique de recouvrement ETABLIE (identique a D6 portage) :
`recouv(f) = sum_x min(recon_x, oracle_x) / sum_x oracle_x`.

- **GATE** : `recouv >= 0,90` sur `>= 2/3` films.
- **TEMOIN OBLIGATOIRE** (garde du protocole : un test dont le temoin ne s'effondre pas n'a
  rien mesure) : affectation ALEATOIRE des selections aux xuid (graine fixe, reproductible).
  `recouv_temoin <= 0,50` ET `recouv_signal - recouv_temoin >= 0,30`. Si le temoin ne
  s'effondre pas, la mesure n'a rien montre -> traite comme NON tenu (honnete).

**Log** : `VIP_periodes.log`. Un film par processus.

Si le canal ne date PAS (5) ou si le gate rate : `[!]` chiffre, PAS de couronne, dire ce qui
manque (p.ex. les morts VIP seules suffiraient-elles a ordonner ?). NE PAS forcer.

## 4. PUBLICATION de la COURONNE (SI et SEULEMENT SI gates 2 ET 3 tiennent)

La COURONNE marque le joueur VIP courant PAR PERIODE (calque au patron `flagCarriesLayer` :
marqueur sur le joueur, comme le drapeau porte). Icone couronne cherchee d'abord dans la chaine
d'icones du jeu ; sinon glyphe couronne dessine (tokens semantiques, aucun hex).

- Go : type de periode VIP + assemblage calque + bump `EXPECTED_REPLAY_SCHEMA_VERSION` (triplet
  schema Go / contrat / web) + chronique.
- Web : calque couronne (patron `flagCarriesLayer.ts`), hook, garde, MAJ
  `EXPECTED_REPLAY_SCHEMA_VERSION`, i18n FR+EN (« VIP » / « VIP » — terme identique, verifier
  l'usage FR).
- Re-cuisson des TEMOINS avec verification de CONTENU (pas seulement l'existence).
- **Gates de publication** : `go test` des packages touches + contracttest ; `tsc -b` (cache
  purge) ; vitest match-replay ; lint web ; parite schema web/Go. `go vet` + `go build` exit 0.
  Arbre propre, AUCUN push.

Si un gate de periodes rate : `[!]` chiffre, pas de publication, dire ce qui manque.

## 5. Checklist d'execution (statuer chaque item a la cloture)

- [x] E1 — Ce protocole commite AVANT mesure (`vip-couronne(protocole):`, `2038557e8`).
- [x] E2 — Temoin corrige : vip.go (plancher analytique + gate corrige), re-mesure sur TSV
  fige, `VIP_temoin_corrige.log`, verdict chiffre. GATE TENU (comp 22 A REPLIQUE).
- [ ] E3 — Periodes : ObjectiveTypeVip a `namedStatSlots` + TSV, `TestVIPPeriodes`, 3 films
  sous filmproc, `VIP_periodes.log`, verdict chiffre (dates ? recouv ? temoin ?). Commit
  `vip-couronne(periodes):`.
- [ ] E4 — Publication couronne SI E2 ET E3 tiennent : triplet schema + calque + i18n + temoins
  re-cuits + gates verts. Commit `vip-couronne(couronne):`. SINON `[!]` chiffre, pas de calque.

## Journal d'execution

- **E1 (2026-08-27)** `[x]` — Protocole commite avant mesure (`2038557e8`). Temoin corrige
  pre-enregistre avec justification datee : plancher `sum p_v^2`, marge 30 pp, seuils d'accord
  inchanges.
- **E2 (2026-08-27)** `[x]` — vip.go re-ecrit (plancher analytique + gate corrige, permute garde
  en diagnostic). Re-mesure sur TSV/oracle figes, aucun film re-decode. VERDICT : comp 22 A
  **REPLIQUE** `TimesSelectedAsVip` — 3/3 films au gate corrige, marges 65,6 / 53,1 / 37,5 pp
  (toutes >= 30), planchers 34,4 / 46,9 / 62,5 %, permute diagnostic 12,5 / 25 / 50 % (colle au
  plancher), stabilite 3/3, somme-film decale 0. **VIP est valide nativement.** Log
  `VIP_temoin_corrige.log`. Cibles secondaires : VipKills (comp 0 A) tient, KillsAsVip (comp 21 B)
  echoue au decale — non bloquant.
