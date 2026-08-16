# Plan — Le type d'une grenade qui tue, par le LANCER plutot que par le tag

> Ecrit le 2026-08-15. Branche `feat/v75`. Execution sous le contrat du skill
> `plan-execution`. Idee de l'utilisateur, retenue parce qu'elle est VALIDABLE sans rien
> affirmer : le depot possede deja un oracle.

## Le probleme, verifie sur pieces

`damagetag/data/labels.tsv` porte **17 lignes `GRENADE`** : 15 `VALIDE` et **2 `AMBIGU`** —
`31e8d17e` (entrees `gggl` 0+1 sur 4 : Fragmentation ou Plasma) et `88f1034c` (0+1+3 :
Fragmentation, Plasma ou Spike). Ce sont des **effets de degat generiques**, atteints depuis
plusieurs entrees de la liste `gggl` du jeu : les nommer reviendrait a retenir l'entree de
plus petit index, c'est-a-dire a choisir arbitrairement.

Le refus est structurel, pas un oubli : `killicon.go:167-172` — « Priorite NOM > GGGL >
CLASSE : la premiere qui repond gagne. **Une etiquette non publiable (statut AMBIGU, ou
INCONNU sans nom) n'obtient jamais de vignette.** » Les deux tags sont absents de
`rules.tsv`, et il n'existe aucune regle `CLASSE GRENADE` de repli. Le kill feed ne les
resout pas autrement : il lit la MEME table (`killicon.Lookup` = la map `byTag`).

Consequence en chaine : pas de nom -> pas de vignette -> **pas de son** depuis le lot
`f0d712103`, qui joint le son de l'explosion a la vignette.

## L'idee, et ce qui la rend recevable

Le type est peut-etre recuperable **ailleurs que dans le tag** : `doc.grenades` publie les
LANCERS avec leur type (rang 0..3) et leur AUTEUR — 1 117 lancers sur les 23 artefacts
locaux. Un kill a la grenade suit de quelques secondes un lancer du meme joueur.

Ce serait une INFERENCE, et le chantier interdit d'afficher une inference comme une mesure.
**Sauf qu'ici l'inference est testable** : les 15 tags `VALIDE` forment un ORACLE — on
connait leur type independamment. On mesure donc la jointure sur eux AVANT de decider si
elle vaut pour les deux ambigus.

## Objectif et critere de succes

Donner un type a une mort par grenade dont le tag ne le donne pas — **ou etablir que la
jointure n'est pas assez fiable et l'ecrire**. Dans les deux cas le livrable est un chiffre,
pas une opinion.

## Decisions tranchees AVANT execution

1. **Le seuil d'acceptation est fixe maintenant, pas apres avoir vu le resultat : 95 %
   d'accord sur l'oracle, et un temoin qui s'effondre.** Un seuil choisi apres coup n'est pas
   un seuil.
2. **Le temoin est obligatoire** : la meme jointure jouee (a) sur un lancer d'un AUTRE
   joueur choisi au hasard, et (b) sur la meme mort decalee de +10 s. Si le taux reel
   n'ecrase pas les deux temoins, la coincidence est celle de deux signaux frequents — meme
   discipline que les mesures `i54`/`i56`/`i57`, ou elle a deja tue deux pistes.
3. **La PROVENANCE voyage avec la valeur.** Un type infere n'est pas un type mesure : le
   document porte le champ ET son origine (`tag` ou `lancer`). Sans ce champ, personne ne
   pourra plus jamais distinguer les deux, et une revue future prendra l'inference pour une
   mesure.
4. **Aucun repli si le gate echoue** : la mort reste sans type, muette. On ne descend pas le
   seuil pour livrer quelque chose.

## Phases

### Phase 0 — QUANTIFIER (le denominateur manque, il n'a jamais ete compte)

- [x] 0.1 Sur le corpus d'artefacts (`data/cache/replays/halo_infinite`, garde
      `REPLAY_CORPUS`, non versionne) : combien de morts par grenade, et leur repartition
      par tag `jpt!`.
- [x] 0.2 **Quelle part porte l'un des 2 tags AMBIGU.** C'est ce chiffre qui dit si ce
      chantier vaut d'etre mene : marginal -> la ligne du registre suffit ; significatif ->
      la suite se justifie.
- [x] 0.3 Combien de morts par grenade n'ont AUCUN tag resolu (ni VALIDE ni AMBIGU).

**Gate 0** : les trois comptes, avec denominateurs. Si les AMBIGU pesent moins de 1 % des
morts par grenade, **le plan s'arrete** et le registre le consigne.

> **GATE 0 : ECHOUE (le plan s'arrete). Mesure du 2026-08-16.**
> Instruments : `internal/replaybuild/grenade_corpus_test.go` (plomberie),
> `grenade_join_corpus_test.go` (`TestPhase0_…`), `grenade_ambigu_sweep_test.go`
> (`TestPhase0Bis_…`, denominateur elargi). Gardes `REPLAY_CORPUS` / `FILM_SWEEP`, saut
> verifie sans elles.
>
> Corpus d'artefacts, 23 films, **868 morts decodees** :
>
> | item | mesure |
> |---|---|
> | 0.1 morts de classe GRENADE | **63 / 868 = 7,26 %** — 8 tags distincts, tous VALIDE |
> | 0.2 part des 2 tags AMBIGU | **0 / 63 = 0,00 %** (0,000 % de toutes les morts) |
> | 0.3 grenade sans tag resolu | **0** ; 0 tag absent du catalogue, 1 mort de statut INCONNU (classe non etablie) |
>
> Repartition des 63 : `da3b5ba4` 41 (rang 0), `59255106` 8 (rang 2), `0000d627` 6 (rang 1),
> `c8681ecf` 2 et `ee2d686d` 2 (rang 3), `00404748` 2 (rang 1), `000162eb` 1 (rang 0),
> `fca02b3c` 1 (rang 2).
>
> **UN NUMERATEUR NUL NE SE LIT PAS SEUL, et c'est ce qui a decide d'un second instrument.**
> Sur 63, la borne haute a 95 % (Wilson) vaut **5,75 %** : ce corpus ne tranche PAS un seuil de
> 1 %. Or le compte de la phase 0 n'a besoin QUE du film (le tag se lit dans le dead-state ;
> l'artefact ne sert qu'a la jointure de la phase 1), donc le denominateur peut s'elargir a tout
> le cache sans rien construire.
>
> **BALAYAGE ELARGI — 150 films du cache, 11 583 morts** (`TestPhase0Bis`, 45 min, 0 echec) :
>
> | mesure | valeur |
> |---|---|
> | morts de classe GRENADE | **696 / 11 583 = 6,01 %**, sur **13 tags distincts** |
> | tags AMBIGU | **1 / 696 = 0,14 %** — borne haute 95 % (Wilson) **0,81 %** |
> | detail | `88f1034c` : **1 occurrence**, film `13b00e35`. `31e8d17e` : **0** |
> | hors classe | 1 mort de tag absent du catalogue, 66 de statut INCONNU |
>
> **GATE 0 TRANCHE, ET AVEC UN VRAI NUMERATEUR.** Les deux tags ne sont pas un artefact de
> catalogue : `88f1034c` EXISTE dans les films, une fois sur 696 morts par grenade. Mais 0,14 %
> — borne haute 0,81 % — est **sous le seuil de 1 %** fixe avant execution. Le plan s'arrete.
>
> Ce resultat est plus fort que le zero du corpus d'artefacts : il ne dit pas « on n'a rien vu »
> (ce que `.ai/ETAT_DE_L_ART_KILLWEAPON.md` 8.3 disait deja, sans denominateur), il dit
> **combien**. Une occurrence sur 696 signifie qu'un joueur rencontre cette mort muette environ
> une fois tous les 700 morts par grenade — soit, au rythme du corpus (4,6 morts par grenade par
> film), une fois tous les ~150 matchs.

### Phase 1 — MESURER LA JOINTURE SUR L'ORACLE (aucune ligne de production modifiee)

> Phase menee MALGRE l'arret au gate 0, et pour une seule raison : le gate 0 dit que la cible
> est vide AUJOURD'HUI, pas que la methode serait bonne. Mesurer la jointure sur l'oracle
> repond a la question « et si les deux tags apparaissaient a une saison future ». Aucune ligne
> de production n'a ete touchee, et aucune phase 2/3 n'a ete ouverte.

- [x] 1.1 Instrument versionne, garde par variable d'environnement, saute en CI. Pour chaque
      mort dont le tag est `VALIDE` (donc de type connu) : chercher le dernier lancer du
      MEME tueur dans une fenetre anterieure, et comparer le rang du lancer au type du tag.
- [x] 1.2 Publier l'accord **en fonction de la fenetre** (0,5 / 1 / 2 / 3 / 5 / 10 s), pas a
      une seule valeur : c'est la courbe qui dit s'il existe une fenetre naturelle. Repere
      connu : pour une frag, la replication cesse ~1,4 s avant la meche (lot 2).
- [x] 1.3 Les deux temoins de la decision n°2, sur la meme population.
- [x] 1.4 Publier aussi la COUVERTURE : sur combien de morts par grenade un lancer est-il
      seulement trouve dans la fenetre. Un accord de 100 % sur 3 morts ne vaut rien.
- [x] 1.5 Cas confondants a COMPTER, pas a ecarter : un tueur qui a lance deux grenades de
      types DIFFERENTS dans la fenetre (la jointure est alors ambigue a son tour), et une
      mort sans aucun lancer anterieur du tueur.

**Gate 1** : accord >= 95 % sur l'oracle, temoins effondres, couverture publiee. Sinon,
**le plan s'arrete** et le negatif est ecrit au registre — il aura coute un instrument et
donne une reponse.

> **GATE 1 : ECHOUE. Mesure du 2026-08-16** (`TestPhase1_JointureDuLancerSurLOracle`,
> 23 films, 1 117 lancers publies sur 1 170 disponibles).
>
> Population : 63 morts de l'oracle, dont **5 sans pont** vers un index de film (le nom du
> proprietaire de la source n'est pas au roster de l'artefact) -> **58 mesurees** ; **1** n'a
> AUCUN lancer anterieur de son proprietaire, quelle que soit la fenetre. **0 mort non
> revendiquee par le kill-feed, 5 divergentes** (source appartenant a la victime alors que le
> jeu credite un autre joueur — la jointure y cherche donc le lancer de la VICTIME).
>
> Distribution des rangs attendus : rang 0 (frag) 42 = 66,7 %, rang 2 (dynamo) 9 = 14,3 %,
> rang 1 (plasma) 8 = 12,7 %, rang 3 (spike) 4 = 6,3 %. **PLANCHER : un predicteur constant
> « toujours frag » scorerait 66,67 % sans rien lire.** C'est ce chiffre qui rend le temoin
> indispensable.
>
> | fenetre | couverture | accord | confondus | accord(sain) | temoin (a) | temoin (b) |
> |---|---|---|---|---|---|---|
> | 0,5 s | 5/58 = 8,6 % | 100,00 % | 0 | 100,00 % | — (0 trouve) | 0,00 % (1) |
> | 1 s | 12/58 = 20,7 % | 91,67 % | 0 | 91,67 % | 75,00 % (4) | 0,00 % (1) |
> | 2 s | 50/58 = 86,2 % | **96,00 %** | 1 | 97,96 % | 60,00 % (5) | 0,00 % (2) |
> | 3 s | 55/58 = 94,8 % | **96,36 %** | 2 | 98,11 % | 80,00 % (5) | 0,00 % (3) |
> | 5 s | 56/58 = 96,6 % | **96,43 %** | 5 | 98,04 % | 75,00 % (12) | 0,00 % (3) |
> | 10 s | 56/58 = 96,6 % | **96,43 %** | 5 | 98,04 % | 78,95 % (19) | 42,86 % (7) |
>
> (entre parentheses : le nombre de morts ou le temoin a TROUVE un lancer — son propre
> denominateur.)
>
> **Il existe bien une fenetre naturelle** : la couverture saute de 20,7 % a 86,2 % entre 1 s
> et 2 s, puis plafonne a 96,6 % des 5 s. L'accord tient au-dessus du seuil de 95 % des 2 s.
>
> **MAIS LE GATE TOMBE SUR LE TEMOIN (a).** La decision n°2 exigeait que le taux reel ECRASE
> les deux temoins.
>
> - Temoin (b), mort decalee de +10 s : **effondre** — 0,00 % jusqu'a 5 s.
> - Temoin (a), autre joueur tire au hasard : **NON effondre** — 60 a 80 % d'accord. La cause
>   est le plancher : deux tiers des morts par grenade sont des frags, et deux tiers des
>   lancers aussi. Un lanceur QUELCONQUE tombe donc juste par frequence, pas par causalite.
>   Le temoin est de surcroit **sous-dimensionne** (4 a 19 morts seulement, parce qu'un autre
>   joueur a rarement lance dans la fenetre) : il ne pourrait pas demontrer un effondrement
>   meme s'il y en avait un. Dans les deux lectures, la condition « temoins effondres » n'est
>   PAS satisfaite.
>
> Ce que la jointure a bel et bien : la PRESENCE d'un lancer est tres specifique (86,2 % pour
> le vrai proprietaire a 2 s contre 8,6 % pour un autre joueur, soit 10x), et l'erreur passe de
> ~25 % (temoin) a 3,6 % (reel). Le signal existe. Ce qui manque, c'est la marge que le plan
> avait exigee AVANT de mesurer — et le seuil ne se rebaisse pas apres coup (decision n°4).
>
> **LES DEUX DESACCORDS SONT NOMMES**, parce qu'un taux sans ses exceptions n'est pas
> controlable (fenetre du plateau, 5 s) :
>
> - `01e1f945`, t = 194 610 ms, tag `da3b5ba4` : oracle rang 0 (frag), dernier lancer rang 3
>   (spike) — **2 rangs distincts dans la fenetre**, c'est un cas confondant ;
> - `696a9d7c`, t = 387 820 ms, tag `000162eb` : oracle rang 0 (frag), dernier lancer rang 3
>   (spike) — **1 seul rang dans la fenetre**, la jointure se trompe franchement.
>
> Les deux confondent une frag avec une spike, et dans les deux cas le lancer de spike est le
> plus recent. Aucune conclusion n'en est tiree : deux cas ne font pas un patron.

### Phase 2 — PUBLIER (ne s'ouvre que si le gate 1 passe)

> **NON OUVERTE.** Les gates 0 ET 1 echouent, chacun pour une raison independante de l'autre :
> la cible est vide, et la jointure n'a pas la marge exigee. Aucune ligne de production n'a ete
> ecrite. Decision n°4 du plan : « aucun repli si le gate echoue — la mort reste sans type,
> muette ; on ne descend pas le seuil pour livrer quelque chose. »

- [!] 2.1 La jointure vit dans `internal/analysis/replay/` — **non traite** : gate 1 echoue.
- [!] 2.2 Le document porte le type ET sa provenance — **non traite** : rien a publier.
- [!] 2.3 Une mort dont la jointure est ambigue reste SANS type — **non traite** ; la regle
      reste vraie et le compte des cas confondants est au journal de la phase 1 (1 a 5 selon
      la fenetre).
- [!] 2.4 `slog.DebugContext` sur le taux de jointure — **non traite** : aucune jointure au
      build.

**Gate 2** : `go test ./internal/analysis/replay/` vert, golden explique,
`golangci-lint --new-from-merge-base=origin/main` 0 issue. — **sans objet.**

### Phase 3 — EN TIRER LE SON ET L'ICONE

> **NON OUVERTE**, pour la meme raison. Le refus de vignette (donc de son) des deux tags
> AMBIGU reste le comportement voulu, et il reste sans consequence mesurable : aucune mort du
> corpus ne les porte.

- [!] 3.1 Le son en seconde entree de jointure — **non traite** : pas de type infere a sonner.
- [!] 3.2 L'icone d'un type infere — **non traite** ; l'arbitrage utilisateur devient sans
      objet tant qu'aucun type n'est infere.
- [!] 3.3 Strings UI FR/EN — **non traite** : aucune string ajoutee, aucun fichier web touche.

**Gate 3** : garde-rail `replaySoundAssets.guard.test.ts` vert, gates web verts, gate d'ecoute
utilisateur. — **sans objet.**

## Ce que la mesure NE dit PAS

1. Elle ne dit pas que les deux tags AMBIGU n'existent jamais — au contraire, le balayage large
   en a trouve UN (`88f1034c`, film `13b00e35`). Elle dit que leur poids est de 0,14 % (borne
   haute 0,81 %) sur 150 films. `31e8d17e` reste, lui, jamais observe : sur 696 morts par
   grenade sa borne haute vaut 0,55 %, ce qui ne prouve pas son inexistence.
2. Elle ne dit pas que la jointure par lancer est fausse : 96,4 % d'accord est un vrai signal
   (erreur 3,6 % contre ~25 % pour un lanceur quelconque). Elle dit que la marge exigee avant
   mesure n'est pas atteinte, et que le temoin (a) est trop peu peuple pour trancher.
3. Elle ne dit rien des morts par grenade dont le tag n'a PAS de classe : 1 mort de statut
   INCONNU sur 868 dans le corpus. Une grenade pourrait s'y cacher ; le compte est trop petit
   pour changer quoi que ce soit.
4. Elle ne dit rien du BTB ni des modes a plus de 8 joueurs pour la phase 1 : la mesure porte
   sur les artefacts existants, et l'attribution ligne par ligne y est deja conditionnee a la
   marge de bijection.
5. Les 5 morts « sans pont » (8 % de l'oracle) ne sont pas expliquees ici : le nom du
   proprietaire de la source n'a pas trouve d'entree au roster de l'artefact. Ce trou appartient
   au pont d'identite, pas a la jointure. **Le corpus compte AUSSI exactement 5 morts
   divergentes** ; l'instrument ne dit PAS s'il s'agit des memes cinq, et il ne faut pas le
   supposer — deux comptes egaux ne sont pas un appariement. Le verifier tient en un champ, et
   c'est le premier geste si la piste se rouvre.

## Regles dures

- Aucun type devine : sans jointure fiable, silence.
- La provenance voyage toujours avec la valeur.
- Zero fix hors perimetre.
- Aucune base DuckDB ouverte en ecriture.

## Statuts et cloture

`[x]` / `[~]` (reference) / `[!]` (justification). Aucune case vide. Entree datee au
`.ai/thought_log.md`. Reports au `.ai/V7.5/REGISTRE_REPORTS.md` avec condition de reprise.

### Cloture — 2026-08-16

**Plan CLOS sur un NEGATIF MESURE.** Les deux gates echouent independamment :

- **gate 0** : la cible est quasi vide — **1 mort sur 696** morts par grenade (150 films) porte
  un tag AMBIGU, soit **0,14 %**, borne haute 95 % **0,81 %**, sous le seuil de 1 % ;
- **gate 1** : la jointure atteint le seuil d'accord (96,4 % >= 95 %) mais le temoin (a) ne
  s'effondre pas (60-80 %), parce que la population est a 66,7 % des frags.

Aucune ligne de production n'a ete ecrite, aucun fichier web touche, aucune base ouverte.

Ce qui reste au depot : trois instruments gardes et rejouables
(`internal/replaybuild/grenade_corpus_test.go`, `grenade_join_corpus_test.go`,
`grenade_ambigu_sweep_test.go`). Ils repondront en une commande le jour ou la question se
reposera — c'est le seul livrable du chantier, et il etait prevu comme tel.

Rejouer :

```bash
cd apps/go-api
REPLAY_CORPUS=../../data/cache/replays/halo_infinite \
  go test ./internal/replaybuild/ -run 'TestPhase0_|TestPhase1' -v -timeout 60m
FILM_SWEEP=150 go test ./internal/replaybuild/ -run TestPhase0Bis -v -timeout 90m
```

Compter ~7 s par film pour le corpus d'artefacts (arena 8 joueurs) et ~18 s pour le cache
complet (les films BTB sont plus lourds) : le balayage de 150 films a pris 45 min.
