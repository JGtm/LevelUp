# PROTOCOLE — TEST DE CADRE ti=11 (COMMITE AVANT MESURE)

> Ecrit le 2026-08-27, branche `wt/ti11-cadre`, base `f350eb01d`. Ce document FIGE le corpus,
> les seuils et les temoins AVANT toute mesure (contrat du chantier : protocole commite avant
> mesure). Les seuils sont recopies du plan `.ai/V7.5/PLAN_TI11_TEST_CADRE_2026-08-27.md` SANS
> modification. Aucune mesure ne modifie ce fichier ; les resultats vont aux logs figes
> (`TI11_T1_cadre.log`, `TI11_T2_porteur.log`) et au compte-rendu.

## 0. Objet et instrument

Question unique de T1 : le CADRE d'image-cle (en-tete par entite + bloc d'etat par defaut +
mots de controle du mode film) se reproduit-il sur le corps type-2 du film pour l'archetype
ti=11 (managed-objective) ? Les feuilles ti=11 cablees sont TRIVIALES et resolues au Ghidra
(docs PISTE_A / PISTE_B), donc un echec localise le mur dans le CADRE, pas dans les feuilles.

Instrument : `WalkKeyframeFullState` (`keyframe_fullstate_loop.go`, la boucle d'etat complet du
jeu portee de `FUN_142e2bfd0` / `FUN_142e2c690`), transpose du harnais biped
`keyframe_biped_fullstate_test.go`. L'oracle de frontiere de record est `WalkKeyframeWorld`
(en-tete 64 bits `[id:32][field:26][ti:6]`, valide 249/250 vs Cheat Engine) : une marche juste
atterrit BIT-EXACT sur le debut du record ti=11 suivant. Complement de frontieres :
`.ai/V7.5/dumps/kf_capture_sample.txt` (400 frontieres, sparse pour les objectifs, secondaire).

Le test est SOUS GARDE D'ENVIRONNEMENT (`TI11_ROOT`), jamais en CI ; un seul decodage filmdec
par process (verrou `LockProcessDecode`). Il ne PUBLIE RIEN (schema/contrat/calque intacts en
T1) : il rend des taux et leurs denominateurs.

## 1. Feuilles ti=11 cablees en T1.1 (perimetre ferme du plan)

Cablees dans `consumeByName` (grammaires des docs, patron ti=10 i1 / ti=13) + hook nomme par
famille :

| i | composant | grammaire | bits |
|---|---|---|---|
| i0 | managed-objective-timers-component | 2x R(7) | 14 |
| i1 | managed-objective-color-component | 4x R(8) RGBA | 32 |
| i3 | managed-objective-object-reference-component | R(32) GlobalID (LE PORTEUR) | 32 |
| i5 | managed-objective-type-component | R(32) | 32 |
| i12 | managed-objective-progress-component | R(32) | 32 |
| i13 | managed-objective-required-progress-component | R(32) | 32 |
| i14 | managed-objective-state-component | R(3) | 3 |
| i15 | managed-objective-parent-objective-component | R(32) | 32 |
| i16..i31 | managed-objective-sub-objective-entities-component | 16x R(32) GlobalID | 512 |

NON cablees (hors perimetre du plan, grammaire non resolue en EXE) : i2 / i9 formatted-text
(deser `consumeObjectiveFormattedText` ecrit mais non branche), i4 interaction-filter, i6
enabled, i7 priority, i8 message-type, i10 is-new-and-unseen, i11 is-only-one-item-unlocked,
i32 outro-phase-duration (8 bits QUANTIFIES, non trivial), i33 forced-update. Ces composants
restent `ported=false` : l'instrument les NEUTRALISE a 0 bit (mecanisme `SetUnportedStubWidth`,
herite du harnais biped) pour isoler le cadre — exactement le meme dispositif que la variante
v4 du biped.

## 2. Corpus admis (verifie present dans `data/cache/film_chunks`)

Familles DISTINCTES portant des objectifs ti=11 (le gate T1 exige >= 2 films de familles
distinctes) :

| famille | films (prefixe 8 hex) |
|---|---|
| Oddball (corpus 5) | 24dbb67d, 43716616, 92f18088, d9781168, c88ec007 |
| CTF | 64e8adfa (Catalyst), 530820e5, 53ce4390 |
| Strongholds | 696a9d7c, 7344d24f, 10ed320d |
| KOTH | 01e1f945 (Catalyst), 606d9844, 8076f97f, 0a247154 |

Corpus de MESURE T1 (cadre, multi-familles), un a deux films representatifs par famille, mesure
et rapporte PAR FILM :
`24dbb67d` (Oddball), `64e8adfa` (CTF), `696a9d7c` (Strongholds), `01e1f945` (KOTH), etendu aux
autres films des familles pour le denominateur. Corpus PORTEUR T2.4 : les 5 Oddball.
Bastion (T2.3) : `7344d24f` / `696a9d7c` (assets Bastion/Strongholds). Total Control : corpus D1
(hors cache verifie ici ; a confirmer au moment de T2.3 si atteint).

Selection par `TI11_FILMS` (liste separee par des virgules) ; defaut = corpus de mesure T1
ci-dessus. Un film par chargement, mesure puis relache (borne memoire).

## 3. Temoins (figes)

1. TEMOIN « record NEW » : `WalkKeyframeBody{DefaultState:true, Gate:true, Mask:true}` — la
   lecture masque-gardee historiquement REFUTEE (en-tete 64 bits, etat par defaut, porte R(1),
   masque de presence variable). Mesure sous le MEME regime de stub que la variante d'etat
   complet, pour que la longueur consommee se compare a quelque chose.
2. TEMOIN « cadre faux » : la meme boucle d'etat complet mais avec une variable de cadre
   deliberement fausse (en-tete 64 au lieu de 108 quand le gagnant est 108, ou l'inverse) —
   controle que le taux ne vient pas d'une degenerescence (records ti=11 si courts que toute
   lecture atterrit).

## 4. SEUILS — recopies du plan, NON MODIFIABLES

### GATE T1 (le gate maitre, joue en premier ; conditionne T2)

> Atterrissage bit-exact (direct + chaine) des records ti=11 **>= 85 %** sur **>= 2 films de
> familles distinctes**, TEMOIN « record NEW / faux » **<= 20 %**. Denominateur = records ti=11
> BORNES par un voisin de l'oracle. Log fige `TI11_T1_cadre.log`.
>
> - SI RATE : le cadre ne reproduit PAS. STOP mesure. Le mur est le format type-2 (localise
>   DEFINITIVEMENT, feuilles triviales donc echec non imputable aux feuilles). Verdict [!]
>   chiffre, condition de reprise = levier type-0 `FUN_142e35a58`. NE RIEN PUBLIER. Fin du lot
>   cote natif ; la resolution bascule sur la voie statborg (lot separe).
> - SI PASSE : enchainer T2.

### GATES T2 (seulement si T1 passe)

- **T2.1 B1 auto-coherence** : les GlobalID lus (i3, i16-31) sont des entites valides et stables
  dans le temps **>= 90 %** (sous-ensemble des GlobalID ti=11 du meme film ; temoin = 16 valeurs
  32b au hasard <= 1 %).
- **T2.2 B2 owner** : i1 (couleur -> camp par clustering RGBA <= 3) vs proprietaire deja publie
  (`zone_states` 93 %, `hillStates` 88-89 %). Accord global **>= 90 %** ET **>= 85 % par equipe
  tenante** prise SEPAREMENT (juge du risque POV).
- **T2.3 B3 zones** : sur un Bastion (3 zones), i16-31 apparie les 3 formes (cardinal = **3**,
  appariement **>= 90 %**, temoin decale 12 m <= 20 %). Sur Total Control, cardinal <= 16, 3
  actives par manche, attribution **>= 80 %**, temoin decale <= 20 %.
- **T2.4 PORTEUR (livrable phare)** : i3 object-reference -> l'objet porte ; confronter au gate
  historique Oddball `time_as_skull_carrier_seconds` (vue `match_objective_stats_latest`)
  **>= 80 %** par joueur ET porteur principal sur **>= 3/4 (5)** films. Log `TI11_T2_porteur.log`.
- **T2.5 PUBLICATION** (si tous les gates T2 tiennent) : etat d'objectif natif — porteur dans
  `objectiveObjects`, owner de zone/colline, identite de sous-zone ; triplet schema Go/contrat/
  web, chronique, i18n FR+EN, re-cuisson des temoins avec verification de CONTENU.

## 5. Gates techniques du lot

Protocole commite (ce fichier, un seul commit). Logs figes. `go vet` / `go build` exit 0 sur les
paquets touches. Arbre propre. Pas de push. Si publication (T2.5) : gates web+contrat verts.
