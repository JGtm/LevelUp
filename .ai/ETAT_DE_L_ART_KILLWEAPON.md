# ETAT DE L ART — arme / categorie par kill (films Halo Infinite, 100% offline)

> INDEX INTERROGEABLE du chantier. Etat au 2026-07-28, apres 7ter.102.
> Ce fichier n est PAS une source : la source est `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md`.
>
> **AVANT DE LANCER UN LOT, LIRE §3bis — LA METHODE** (les onze regles qui ont produit les
> resultats, avec le lot qui illustre chacune). Elle se lit avec §4, qui dit l inverse : les
> patrons d erreur a ne pas refaire. Motifs de grep : `LA METHODE` · `METHODE §3bis` · `REGLE 1`.
>
> **LIVRABLE PRODUIT LE PLUS RECENT, ET IL N EST PAS DANS CET INDEX** : la precision d un joueur
> se publie **sans aucune reference externe** (taux de remplissage `porteurs / records`, MAE
> **0.0266**, `r = +0.8204`). Ce qui fait foi pour l USAGE est `.ai/GUIDE_WEAPON_SHOTS.md`
> **§3quater** ; ici, la mesure est en **§21.3** et sa portee en **§22**.

> ## RESERVATION DE NUMERO DE SECTION — A LIRE AVANT D ECRIRE DANS LE RE_LOG (2026-07-28)
>
> **DERNIERE POSEE : `7ter.102` (lot `vf`, correction Falcon / Pelican + recompte + livrable vehicules). PROCHAINE LIBRE : `7ter.103`.**
> `7ter.90` (`vk`, vehicules), `7ter.91` (`c7b`), `7ter.92` (`co.pi`), `7ter.93` (`co.re`),
> `7ter.94` (`tp.ref`, verification adversariale de 7ter.90 et 7ter.91), `7ter.95` (`co.ref`,
> verification adversariale de 7ter.92 et 7ter.93) et `7ter.96` (`vl`, livrable utilisateur
> vehicules) sont ECRITES ; `7ter.98` (`rc.unite`, arbitrage tir / touche), `7ter.99` (`rc.perm`,
> permutation d indice) et `7ter.101` (`rc.ref`, verification adversariale de 7ter.98 et 7ter.99)
> sont ECRITES ; `7ter.97` (`vc`), `7ter.100` (`vc.ref`) et `7ter.102` (`vf`) sont ECRITES.
> **PIEGE MESURE LE 2026-07-28 PAR LE LOT `vl`, ET IL FAUT LE CONNAITRE** : reserver son numero
> ne suffit pas. Trois lots ont pose leur en-tete APRES `7ter.96` pendant que `vl` mesurait, et
> le corps de `7ter.96`, ajoute par `>>` en fin de fichier, s est retrouve **sous l en-tete de
> `7ter.99`**. **Un corps ne s ecrit pas en fin de fichier : il s INSERE apres SON PROPRE
> en-tete** (relire le numero de ligne de son en-tete juste avant d ecrire, et verifier apres
> coup `sed -n '<ligne>,+15p'`).
> Verification, toujours, avant d ecrire : `grep -n "^### 7ter" .ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md | tail -5`.

## 1. COMMENT SE SERVIR DE CE FICHIER

1. **Avant d ouvrir une piste, CHERCHER ICI** (Ctrl-F sur le sujet, puis §6 pour le motif de grep).
2. Chaque entree porte le **numero de section du RE_LOG qui fait foi** : y aller lire le detail.
3. Le RE_LOG reste la SOURCE ; ce fichier est l INDEX. En cas d ecart, le RE_LOG gagne.
4. Regle nee de 7ter.59bis : une reponse deja ecrite a ete re-cherchee par trois agents successifs
   et deux verifications adversariales. **Ce fichier existe pour que cela ne se reproduise pas.**
5. Statuts : `THEATER` = confirme par l utilisateur en mode Theater · `MESURE` = chiffre reproduit ·
   `MESURE+VERIFIE` = rejoue par un agent adversarial independant.
6. **DEPUIS 7ter.80, UN VOCABULAIRE DE STATUT PLUS STRICT s applique aux sections neuves** :
   `[ETABLI]` = mesure + controle negatif discriminant + reproduction independante (second agent
   ou second instrument) · `[MESURE]` = mesure + controle negatif, non reproduit par un tiers ·
   `[PLAUSIBLE]` = faisceau convergent, il manque un controle ou une reproduction — **ne jamais
   l ecrire comme un fait** · `[NON VERIFIE]` = avance sans mesure, ou dont la mesure d origine
   s est revelee irreproductible. Plus deux mentions : `[HORS ECHANTILLON]` (evalue sur des films
   qui n ont pas servi a formuler le resultat) et `[REFUTE]` (teste, et FAUX).
   **Un statut trop genereux est pire que pas de statut, parce qu il sera cru.** Dans le doute,
   descendre d un cran.
7. **UN `[ETABLI]` DOIT NOMMER SON REPRODUCTEUR** (regle posee par `jr.v` le 2026-07-27, en
   verifiant 7ter.80/81) : ecrire QUI a reproduit et AVEC QUOI (quel lot, quel binaire, quel
   corpus, quel partage). Un statut qui ne le dit pas n est pas auditable et se lit comme un
   `[MESURE]`. Corollaire : quand un paragraphe couvre deux enonces de force inegale, il porte
   DEUX statuts, pas le plus haut des deux.
8. **DEUX SECTIONS SE LISENT AVANT DE COMMENCER, PAS APRES** : **§3bis (LA METHODE)** — les dix
   regles qui ont produit les resultats — et **§4 (LES PATRONS D ERREUR A..F)** — ce qu il ne faut
   pas refaire. La premiere dit quoi faire, la seconde quoi eviter ; chaque regle de §3bis nomme le
   patron dont elle est l antidote.
9. **UN TITRE DE SECTION DU RE_LOG PEUT ETRE FAUX ALORS QUE SON CORPS EST JUSTE.** Le journal est
   append-only : quand une section ulterieure refute ou corrige un titre, un **RENVOI EN TETE** est
   pose juste sous ce titre (motif de grep : `RENVOI EN TETE`). **Dix en portent un a ce jour** :
   7ter.80, 7ter.81, 7ter.83 (portee), 7ter.84 (**titre REFUTE**), 7ter.85 (titre corrige),
   7ter.88 (**titre partiellement REFUTE** : son << controle positif passe >> est une identite
   algebrique, 7ter.89 (2)), 7ter.90 (**TROIS renvois** : << rang 1 sur 1608 >> REFUTE par
   7ter.94 (7) ; portee de deux films, 7ter.97 ; et 7ter.100, qui RETABLIT son negatif sur ses
   deux films et jette son << controle positif a 1.00000 >>), 7ter.91 (**son point (7) est trop
   pessimiste et son tableau de largeurs est REFUTE**, 7ter.94 (6)), 7ter.96 (**sa formule de
   confiance est trop genereuse d un cran**, 7ter.100 (12)) et 7ter.97 (**trois de ses controles
   n ont aucun degre de liberte et la portee de son titre est fausse**, 7ter.100). Lire le renvoi
   avant le corps.
10. **LE NUMERO DE SECTION SE RESERVE EN ECRIVANT L EN-TETE AVANT DE MESURER.** Regle jumelle de
    la 9 : le journal est append-only, donc deux lots paralleles qui mesurent d abord et ecrivent
    ensuite visent le MEME numero et se telescopent. La sequence obligatoire est :
    `grep -n "^### 7ter" .ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md | tail -5` -> **ecrire immediatement
    `### 7ter.<N> <titre provisoire> (EN COURS)`** -> mesurer -> completer le corps.
    Un en-tete pose est une reservation ; un numero annonce dans un brief n en est pas une.
    **C est arrive DEUX FOIS** : `7ter.82` le 2026-07-27 (deux lots, le second est parti en
    `7ter.83`) et `7ter.89` le 2026-07-28 (le lot vehicules a pris le numero qu un lot encore en
    cours devait ecrire — cf. le bandeau RESERVATION DE NUMERO en tete de fichier).
    **Si un doublon est ne malgre tout** : renumeroter la **SECONDE** occurrence (jamais la
    premiere, deja citee ailleurs), puis corriger toutes les references croisees
    (`grep -rn "7ter\.<N>" .ai/`) et la ligne << derniere posee / prochaine libre >> du §6.

---

## 2. TABLE DES QUESTIONS RESOLUES

### 2.1 LOCALISATION DE LA SOURCE (ou est l arme dans le film)

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Ou est l arme du kill dans le film ? | `tag+0x00` du composant **dead-state i11 de la VICTIME** (lu par `FUN_14080d69c`) EST l identifiant de la source de degat fatale. | 7ter.35 (9/9) · THEATER |
| Le kill-event porte-t-il l arme (queue DamageSource) ? | Non : la queue existe dans l .exe, elle est **ABSENTE des films**, anciens comme recents — 0/133 au vrai curseur sur 5 films. | 7ter.23 · MESURE |
| Grammaire du dead-state i11 ? | `FUN_140c1dce0` [R1 Mort] -> `FUN_140c1dd44` ; cas nominal : VICTIME bits 11..15, TUEUR 17..21, CATEGORIE (R4) 22..25 ; largeur totale 191 bits. | 7ter.27 (bloc DETAILLE) · MESURE |
| Grammaire d un record ECS delta ? | `[R1=1][R13 slot][R2 tag][R1 baseline][R1 flag masque][R3 count][count x R6]` ; 2e forme = masque `R64`, en-tete complet **82 bits**. 54 760/54 760. | 7ter.26(2) + 7ter.28(2)(3) · MESURE+VERIFIE |
| Ou commence la boucle de records dans un paquet A EVENTS ? | Signature structurelle : tout paquet type-0 commence par un `DELTA slot=123 ti=4` de **35 bits exactement** ; candidat UNIQUE et VRAI sur 690/690. Ce n est pas une heuristique. | 7ter.34 (bloc DETAILLE) · MESURE |
| Ou sont les morts (population) ? | **93/93 dans les paquets type-0 A EVENTS**, calage -10 ms, pic unique large d un paquet, taux de base 11.5%, controles negatifs 9.7-15.1%. | 7ter.33 · MESURE |
| `slot -> archetype` est-il derivable ? | Non, c est un **tableau persistant** : `entries = *(World+0x20)`, stride 200, `entry+0x00` = eid COMPLET, `entry+0x04` = index d archetype. Ecrit par `FUN_1408f2150`, un seul ecrivain. | 7ter.33(3) + 7ter.54 AXE1 · MESURE+VERIFIE |
| Un DELTA sur slot non lie peut-il exister dans un film valide ? | Non : le jeu fait `return 3` et **ABANDONNE le paquet** sans lire le corps. Rejet dur legitime, pas une heuristique. | 7ter.33(3) + 7ter.54(2b) · MESURE+VERIFIE |
| Le schema ECS est-il dans le film ? | Oui, `chunk_00` : archetype 35 = biped, **64 composants NOMMES**, `i11 = object-dead-state-component`. Identique entre films (49 concordants, 0 divergent). Zero CE. | 7ter.28 · MESURE |
| Comment resoudre un deserialiseur de composant 100% statiquement ? | nom (`chunk_00`) -> chaine unique `.rdata` -> unique xref = `getName()` -> motif d octets -> descripteur ; **deser = `*(ptrGetName+0x20)`**, et si c est le thunk `FUN_14076ce9c` alors **`+0x28`**. **ATTENTION, DEUX FORMULES COEXISTENT DANS LE JOURNAL** : 7ter.29 (bloc DETAILLE) ecrit `bloc+0x40` avec le thunk en `bloc+0x38` — autre BASE, autre formule. C est celle-ci, en base `ptrGetName`, qui fait foi. | 7ter.31, 7ter.33(4), handoff §7 · MESURE |
| La categorie 4 bits discrimine-t-elle grenade et melee ? | **NON.** Grenade frag et melee ordinaire sortent toutes deux `cat=0 None`. `SilentMelee` = assassinat ; `AttachedDamage` = projectile FIXE sur la cible (grenade collee **ET** supercombinaison Needler). C est le TAG qui porte melee et grenade. | 7ter.37(4) + 7ter.33bis · THEATER |
| L assist est-il decodable ? | Oui, mais **le chiffre de 7ter.24ter est MORT** (voir §6.2 et §11) : la lecture nettoyee donne **22 nommes pour 30 API** sur `9b191a7f`, et la validation qui tient est le **MULTISET par joueur** (deux films, 8 joueurs, rendu A L IDENTIQUE : 17/17 et 29/29). Grammaire REELLE : `victime(E5) tueur(E5) [% TUEUR] R1 assistant(E5) [% ASSISTANT]`. | 7ter.76 (remplace 7ter.24ter) · MESURE |
| Que sont les deux blocs R32 du kill-event ? | Des **PARTS DE DEGATS EN POURCENTAGE ENTIER** — la 1re du TUEUR, la 2e de l ASSISTANT. Quatre jambes convergentes (adjacence `+0x228`/`+0x22c`, type entier, somme == 99 sur 22 367 / 31 204, collision CE sur la constante 149). **RESERVE : le chemin de donnees n est PAS demontre** (setter generique partage par 9 modeles d UI). | 7ter.75 (3) · MESURE |
| Dans quel ORDRE sont les deux premiers champs du kill-event ? | **VICTIME puis TUEUR** — mesure sur le fil, facteur 6 a 13 : 79/101 contre 6/101, 83/100 contre 10/100, 81/105 contre 14/105, 89/106 contre 10/106. Le rejet historique << assistant == tueur >> portait en fait sur `assistant == victime`. | 7ter.75 (2) · MESURE |
| Le taux d attachement des kill-events valide-t-il quelque chose ? | **NON, il est CIRCULAIRE** : `quadScore` fixe la bijection en maximisant le meme critere de couple. Ce n est PAS une mesure independante de l espace d indices — c est un **SELECTEUR DE POPULATION**, et rien d autre. | 7ter.76 (2) · MESURE |
| **Ou sont les touches des armes a PROJECTILE ?** | **DANS LE FLUX D EVENTS, ET ELLES ONT UN NUMERO : les codes 6 et 7.** 7ter.84 concluait l inverse ; 7ter.86 le REFUTE avec le meme corpus et un binaire independant : **80 886 code 6 et 129 390 code 7 sur 949 films**, dont 80 % en PREMIER evenement — position independante de toute grammaire de corps. Ils etaient deja dans l histogramme de 7ter.84 (3), aux rangs 14 et 11, sous une coupure d affichage a 9 lignes. Desassemblage : `FUN_142f1c44c` (impact de projectile) -> `FUN_140de8cb0` -> `FUN_142eed4e8`, qui emet le **type 6** quand rien n est touche (charge 0x40) et le **type 7** quand une entite l est (charge 0x4c, 2 entites : la CIBLE puis le PROJECTILE). **Localise** : `r(code7, tirs projectile) = +0.7675` contre `r(code7, tirs hitscan) = -0.1929` ; `(code6+code7) / tirs de projectile = 0.9831`. **Pas `[ETABLI]`** : le TIREUR est absent de l evenement, donc aucun test d egalite exacte par joueur. | 7ter.86 (2)(3)(4) · `[MESURE]` |
| Les touches de projectile dans la REPLICATION (archetype 41) | **Piste PARALLELE, ni fermee ni prioritaire** : l **archetype 41** du registre `chunk_00` EST un objet-projectile (22 composants, identique sur 6 films, reellement instancie), portant `projectile-at-rest-state` (champ de **19 bits jamais lu**), `projectile-tether-state`, `object-parent-state-component` et le meme `object-dead-state-component` i11 que le coup fatal. **Les 4 deserialiseurs sont dans `filmdec/traverse.go` depuis le 2026-06-13**, ecrits comme CONSOMMATEURS de bits. Bloqueur nomme : `object-position-component` de ti=41 (i0, `FUN_14076e29c`) n est pas bit-exact (45 / 60 bits), et i0 precede i18. **Les codes 6/7 sont la voie COURTE ; celle-ci est la voie longue.** | 7ter.84 (5)(6)(7) · `[MESURE]` existence / `[PLAUSIBLE]` comme porteur |
| Le record de degat (code 36) est-il exploitable ? | Il est le **1er event de son paquet** (2874/2874 sur 9b191a7f) ; famille+variante a `pos+42` (ou `+37`) ; attaquant = `R(5)@pos+34`, joueur = /2. Mais il n est **jamais dans le paquet du kill-event**. (⚠ le `/2` est **REFUTE** par 7ter.81 (7).) | 7ter.26(5)(ARBITRAGE) · MESURE |
| **Que portent les 3 emplacements de PRESENCE d un evenement ?** | Des **HANDLES D ENTITE** : `FUN_1406d3140` finit par `*param_4 = uVar11 << 0x1e \| uVar8` — 30 bits d indice (`base(cfgIdx)` + valeur quantifiee) et 2 bits de generation. Ce que `FUN_141fd8460` serialise est le champ **`objet + 0x114`**, celui-la meme qui sert de porte de deduplication dans `FUN_142f1c44c` : c est donc un **SLOT DE REPLICATION**. Le depot les LISAIT SANS LES DECODER (`evPresence` n avance que le curseur). Signature mesuree, non reglee : presence 98-99 % sur l emplacement 0 de tous les codes frequents, et la **generation separe les populations** — 1 dans 99.9 % des cas pour les objets longs, etalee sur 4 valeurs pour les objets recycles. | 7ter.88 (1) · `[MESURE]` |
| **Le << marqueur de fire-event >> du depot, c est quoi exactement ?** | **L EN-TETE D UN EVENEMENT DE CODE 36.** `0b10100100110` se lit `[1 continuation][R(7)=0100100=36][1 emplacement 0 present][1 court-circuit -> ligne SPECIALE 4]`. Mesure concordante sans avoir ete construite pour : la ligne effective du code 36 emplacement 0 est **4** dans 43 411 cas contre **1** dans 23 (25 films). Deux instruments du depot decrivaient le meme objet sans jamais se rencontrer. | 7ter.88 (1) · `[MESURE]` |
| **Le fire-event permet-il d attribuer un OBJET a un JOUEUR ?** | **OUI, et c est une FONCTION** : le handle de son emplacement 0 rend un unique indice de tireur pour **14 896 / 15 192 = 0.9805** des handles (150 films, 319 883 occurrences), contre **0.0652** apres permutation INTRA-film. **Reproduit a l unite** par un second binaire (7ter.89 (1), lot `tv.ref` : memes 14 896/15 192, meme table 1:14896 2:258 3:31 4:6 5:1). **Cet objet n est PAS le joueur** : handle -> une seule famille d arme 0.6632 contre 0.0502 regroupe par joueur ; 10 handles par joueur et par match, generation 1 a 99.94 %. | 7ter.88 (2)(3) · `[ETABLI]` pour la fonction (reproducteur `tv.ref`) ; << c est l arme tenue >> `[PLAUSIBLE]` |
| **Le roster de l API confirme-t-il la position `+34` du champ de tireur ?** | **NON — il exclut un cadrage grossier, il ne choisit pas entre deux bits voisins.** Le tableau de 7ter.88 donne **+35 a 50/150 comme +34** ; sous le filtre << l evenement porte un handle >>, +35 le **BAT** (69/150 contre 60/150). Ce qui les separe est l ecart absolu moyen — la statistique agregee que la REGLE 1 interdit comme discriminant. **`+34` reste la bonne position, mais parce que DEUX instruments du depot la designent** (7ter.26 (5) et `weaponv3.FirePi5SpanBefore`). ⚠ La PURETE est aveugle sur **+33..+36** (0.9808/0.9805/0.9803/0.9801) mais **PAS partout** : elle chute a **0.8422 a +32**. | 7ter.89 (4) · `[CORRIGE]` |
| **Pourquoi le film rend-il 9 indices de tireur distincts pour un roster de 8 ?** | **RESOLU — artefact d instrument, pas un participant de trop.** Le compte de 7ter.88 porte sur TOUS les code 36, y compris ceux sans aucun emplacement de presence. Filtrer sur << l evenement porte un handle >> deplace les trois chiffres ensemble : mediane **9 -> 8**, ecart absolu **1.953 -> 1.600**, egalites exactes **50/150 -> 60/150**. Ni bot, ni joueur parti, ni indice reserve : du bruit de marche. | 7ter.89 (5) · `[MESURE]` |
| **Peut-on relier le PROJECTILE a son TIREUR ?** | **NON PAR LE FLUX D EVENTS.** Critere : un objet est cree UNE fois, donc un evenement de creation **COUVRE TOUS** les projectiles. Les codes 7 (emplacement 1) et 6 (emplacement 0) couvrent **0.5408** et **0.4606** ; **aucun troisieme code** : meilleur suivant 0.2318 (permute 0.1601), fire-event **0.0931** (permute 0.0711) ; balayage COMPLET des ecarts de base **sans aucun pic**. ⚠ **CORRECTION 7ter.89 (2)(3) — NE PLUS CITER << la somme 1.0014 prouve que l instrument sait reconnaitre un code qui nomme des projectiles >>** : l ensemble de reference EST l union des deux ensembles compares, donc la somme vaut `1 + recouvrement` **IDENTIQUEMENT** (reproduit a 9 decimales sur 2 corpus). Le negatif TIENT quand meme, par le controle qui manquait : couverture d une reference **INDEPENDANTE** (code 6 seul) avec une nulle de **MEME POOL** — meilleur candidat **0.5046** pour une nulle a **0.3600** (rapport 1.40), quand un nommage reel exige 1.0. Le projectile n existe dans le flux **qu a l instant de sa mort**. | 7ter.88 (4)(5) **corrige par 7ter.89 (2)(3)** · `[MESURE]` |
| **Que vaut la voie du TEMPS (apparier l impact au tir qui precede) ?** | **Plafond 0.41 d impacts a candidat UNIQUE**, toutes fenetres et toutes restrictions confondues (150 films, 23 188 impacts). Sous 100 ms le nombre moyen de candidats est **inferieur a 1** (la majorite n a aucun tir dans la fenetre) ; au-dela l ambiguite croit plus vite que la couverture (1.98 candidats a 1 s). C est le motif **same-clock** deja invalide (7ter.19). **Aucun test d egalite exacte ne peut reposer la-dessus.** | 7ter.88 (7) · `[MESURE]` |

### 2.2 NOMMAGE

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Qu est-ce que `tag+0x00` exactement ? | Un **tag de groupe `jpt!` (damage_effect)** des archives `.module` (entree 0x58 o, global tag id a `+0x28`, groupe fourCC a `+0x14`). **59/59** valeurs distinctes sur 4 films, zero exception, dont 22 inedites hors echantillon. | 7ter.36(7) -> 7ter.42(5) -> 7ter.43(5) · MESURE+VERIFIE |
| Taux de faux positif du test `jpt!` ? | ~**1 sur 9 000 000** pour les valeurs >= 0x10000 (468 `jpt!` sur 705 064 entrees = 0.066% ; 2.7 M tirages uniformes -> 0 en `jpt!`). RESERVE : sous 0x10000 le taux monte a 1.15%. | 7ter.38(3) · MESURE |
| Comment passer de `jpt!` au NOM de l arme ? | Graphe de dependances des `.module` (en-tete `ucsh` + table 0x18 o/entree) : remontee en LARGEUR jusqu au premier niveau contenant un `weap`, en acceptant les ENFANTS `weap` de l ancetre (motif `wcfg`), restreinte aux groupes qui DECRIVENT une arme. | 7ter.39(2) · MESURE+VERIFIE (7ter.41) |
| Les cinq chaines de nommage ? | `weap->proj->jpt!` (standard) · `weap->wcfg->sofd->sofa->weab->proj->jpt!` (variante) · `jpt!` reference par N `weap` (melee) · `vehi->weap` OU `vcdd->sofd->sofa->uwfa->weap` (vehicule) · `hlmt` du `weap` declare directement un `jpt!` (objet explosif). | 7ter.39 + 7ter.45(2) · MESURE+VERIFIE |
| Un tag = une arme ? | **Non** : la relation est MANY-TO-ONE **par EFFET**. Le Disruptor porte quatre tags distincts (standard, variante Fiesta, deux effets). L etiquette utile est `arme (effet)`. | 7ter.44(3) + 7ter.45(6) · THEATER |
| Le meme nom d arme a-t-il le meme tag d un film a l autre ? | Seulement si la VARIANTE ne change pas. Disruptor `eea85c26` (Super Fiesta) vs `7e6a7461` (standard). La MELEE `daa03c35` est stable cross-film (pas de variante). | 7ter.37(4) + 7ter.37bis · THEATER |
| Les noms LITTERAUX existent-ils offline ? | **OUI**, par les banques Wwise `Content/Sound/win/SFX/*.pck` : un `sbnk` porte un `BKHD` = **FNV-1 32 bits** du nom de banque en minuscules. 1339 `.pck`, **0 collision** ; 1953 `sbnk` a BKHD, 1156 resolus (59.2%). | 7ter.48(1) · MESURE+VERIFIE (7ter.49, 7ter.50) |
| Le pont sonore est-il precis ? | Pas toujours : il nomme parfois **LARGE** (le chassis de la Gungoose rend faction+classe, jamais le modele) et **aucun critere ne le dit A L AVANCE**. Controle catalogue : 14/35, baseline triviale 0/35, permutation 1/35. | 7ter.48(1) + 7ter.49(1) · MESURE+VERIFIE |
| Peut-on faire confiance a une etiquette de banque « PRECISE » ? | Pas seule : `69fd30b9` est publie vers `escharumhammer` (marteau Bannis de CAMPAGNE) alors que `gravityhammer.pck` existe. **Poser un drapeau de desaccord quand banque et etiquette divergent de famille.** | 7ter.50(3)5 · MESURE |
| Les `.module` stockent-elles des chemins de tag ? | Non : `stringsSize` = 0, le `strtab` est **DECLARE mais NON LIVRE** (`headerSize` = somme des tables MOINS `strTabSize`) ; seul ASCII utile d un `weap` = une CLASSE (`turret`, `fixed`, `pball`, `rifle`, `pistol`...). | 7ter.45(1), mecanisme explique 7ter.48(1) · MESURE |
| Le catalogue `WeaponIDToName` couvre-t-il tout ? | Non : **32 des 194 `weap`**, etendu a 67/194 par la partition selon le modele `hlmt`. La moitie basse de l id 64 bits du catalogue (`42c9679f`...) **N EST PAS un global tag id** (0 occurrence sur 705 064 entrees). | 7ter.39(2)(6) · MESURE+VERIFIE |
| Injectivite reelle de `jpt! -> weap` ? | 55.7% des `jpt!` resolus designent UNE arme ; 23% en designent 2 (arme de base + variante, meme nom apres partition `hlmt`) ; 161 n atteignent aucun `weap` ; 14 tags a >= 12 `weap` = effets PARTAGES (melee). | 7ter.39(5) · MESURE+VERIFIE |

### 2.3 GRENADES

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Combien de grenades multijoueur, et lesquelles ? | **QUATRE**, toutes fermees : entree 0 = FRAG · 1 = PLASMA · 2 = DYNAMO (choc) · 3 = POINTES (Spike, Bannis). | 7ter.37, 7ter.44, 7ter.47 Q3/Q4, 7ter.47ter · THEATER (3) + MESURE (1) |
| Sur quoi repose le decoupage des 8 rangs `gggl` en 4 entrees ? | Sur l **ADRESSAGE** (table `tagref`), pas sur l ordre de la liste de dependances : les 8 references `eqip` alternent entre deux ecarts (`0x38`/`0x98`), periode d entree **0xd0**, 2 couples par entree ; le block fait 832 o = 4 x 0xd0 exactement. | 7ter.48(2), verifie 7ter.49(1) · MESURE+VERIFIE |
| Chaque entree a-t-elle sa banque ? | Oui, **disjointes, mesurees SANS filtre** : 0 `kineticunsc` · 1 `plasma` + `grn_cv_PLASMAGRENADE` · 2 `shock` + `grn_un_LIGHTNINGGRENADE` · 3 `kineticbanished`. Seule banque partagee : `watershallow` (generique). | 7ter.48(2) + 7ter.49(1) · MESURE+VERIFIE |
| `gggl` ou `weap` : qui gagne quand les deux repondent ? | La **precedence `gggl`** est VALIDEE : `0000d627` est une GRENADE, pas un Plasma Pistol. Le lien vers le Plasma Pistol (`wcfg 042a4629`) est un artefact generique. Cas ouvert depuis 7ter.41(6)/P3 : **CLOS**. | 7ter.47(4) · THEATER |
| Pourquoi les films ne montrent-ils que des rangs PAIRS ? | Les 4 rangs pairs ont leur `eqip` reference par les **bot globals** (`botg`, arsenal multijoueur), les 4 impairs jamais (ils pendent a des `hlmt` de bipedes de campagne). 4/4 et 4/4. | 7ter.45(4) · MESURE |

### 2.4 OBJETS EXPLOSIFS

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Comment nommer un objet explosif de decor ? | **R-DESTRUCT** : le modele `hlmt` du `weap` declare DIRECTEMENT un `jpt!`. Critere rare et brut : 40 `hlmt` sur 6088, 12 `weap` sur 194. | 7ter.45(2), 7ter.46 · THEATER (2 temoignages) |
| Que signifie l index d entree du tableau `hlmt` (periode 0x1d8) ? | Le **TYPE D ENERGIE** du baril : un seul modele d objet, plusieurs variantes d energie ; la flaveur de banque le DECRIT. Etiquette : `OBJET EXPLOSIF PORTE (explosion <flaveur>, etat de degat n/7)`. **Denominateur 7, pas 5.** | 7ter.57 (cloture) ; mecanique 7ter.49(3) · THEATER |
| `0d203522` et `5e389b5d` : deux objets ou un ? | Le **MEME objet** (un baril), deux TYPES D ENERGIE (`kineticunsc` / `plasma`). | 7ter.51(1) + 7ter.57 · THEATER |
| Le declencheur (lance / tire dessus) est-il dans le tag ? | **NON, definitivement.** Le MEME tag `0d203522` couvre un LANCER (78919882 06:19, medaille Kong) et un TIR sur baril porte (fccc61cd 03:16). Le geste appartient au TUEUR ; le dead-state est celui de la VICTIME. | 7ter.52bis + 7ter.53(R2) + 7ter.56 · THEATER |

### 2.5 VEHICULES

> **VUE D ENSEMBLE ET FILE DE TRAVAIL DU SUJET : §18 (`VEHICULES — ETAT ET PISTES`).** Les deux
> tableaux ci-dessous restent la source de detail question par question.

> **DEUX SUJETS SANS RAPPORT PORTENT LE MEME MOT, ET LES CONFONDRE COUTE UNE JOURNEE.**
> (a) **NOMMER l arme d un vehicule** dans les archives `.module` — resolu, R-VEHICULE, ci-dessous.
> (b) **LE VEHICULE COMME ENTITE REPLIQUEE dans le film** (archetype, slots, positions, et le lien
> joueur-vehicule) — c est le bloc 2.5bis, et le lien joueur-vehicule **N EST PAS ETABLI**.

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Comment nommer une arme de vehicule ? | **R-VEHICULE** : un `weap` est un armement de vehicule si un `vehi` le reference OU s il pend a `vcdd -> sofd -> sofa -> uwfa -> weap`. 62 `weap` sur 194. | 7ter.45(2) · THEATER (3 temoignages) |
| Quelle est la force de la regle ? | Disjonction TOTALE d avec le catalogue tenu en main : **0/62** `weap` de vehicule portent un nom du catalogue (attendu 10.2), **p = 1.1e-06**. `uwfa` : 16 entrees dans tout le jeu, 16/16 referencees par un `sofa`, 0 atteignable depuis un `wcfg`. | 7ter.46(2)(5c) · MESURE+VERIFIE |
| Vraie decomposition des 62 ? | **46 par un `vehi` direct, 16 par la chaine `vcdd`** (et non « 61/1 » comme ecrit en 7ter.45). L ancre Gungoose passe EXCLUSIVEMENT par la chaine `vcdd`. | 7ter.46(4) · MESURE+VERIFIE |
| `turret` / `fixed` : vraie distinction ? | **OUI**, par deux signaux sans rapport : la classe ASCII du `weap` et la nomenclature de la banque du chassis (`_tur_` / `_veh_`). turret 22/23 en `_tur_`, fixed 17/22 en `_veh_`. Confirme : Ghost (fixed) contre tourelle UNSC arrachee (turret). | 7ter.47(2) + 7ter.48 AXE3 · THEATER |
| La partition par chassis est-elle trop fine ? | Non — la question a ete posee explicitement en Q2 : le materiel de Q1 et Q2 est bien DIFFERENT. | 7ter.47(2) · THEATER |

### 2.5bis LE VEHICULE COMME ENTITE REPLIQUEE (film) — cartographie faite, CHAINON MANQUANT

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Quel archetype est le vehicule ? | **`ti=40`**, trouve PAR SON NOM (composants `vehicle-*`), jamais par un indice code en dur. **48 composants**, dont **14** prefixes `vehicle-` (et non 15 : le 15e nom contenant `vehicle` est `player-vehicle-entrance-ban-component`, porte par `ti=5`, le JOUEUR). Identique sur 5 films, 0 divergent. | lot `vh` **corrige par** 7ter.89 (7) · `[MESURE+VERIFIE]` |
| Le vehicule partage-t-il l epine dorsale du bipede #35 ? | **PAS a la lettre** : le plus long prefixe commun fait **2** composants, pas 18. `i2` et `i3` sont les variantes **`-dynamic-precision-`** chez le vehicule (`object-forward-and-up-*`, `object-angular-velocity-*`) la ou #35 porte les variantes simples. #35 = 64 composants, #40 = 48. **MAIS `i4..i17` sont identiques, dont `i10 object-parent-state` et `i11 object-dead-state`**, et surtout les **30 composants `i0..i29` ont TOUS un deserialiseur porte** dans `traverse.go` ; le premier absent est `i30 vehicle-auto-turret-triggers-component`. La conclusion << la desync ne peut pas arriver avant i30 >> tient — par la couverture des desers, pas par une identite de liste. | lot `vh` **corrige par** 7ter.89 (7) · `[CORRIGE]` |
| Ou vivent les slots de vehicule ? | Bande **`[768, 1023]`**, jamais depassee. Recensement, 5 films : **0 · 0 · 19 [768..786] · 21 [768..788] · 256 [768..1023]** — reproduit A L UNITE par un second binaire. ⚠ Le **256** du BTB est une **SATURATION** de bande (256 valeurs, 256 declarees) : sur ce film le filtre de slot ne filtre plus rien. ⚠ Les bandes **grandissent avec le roster** (`ti=35` va jusqu a **764** sur le BTB, `ti=40` demarre a 768) : coder `[768,1023]` en dur est un **pari sur la taille du roster** (meme piege que 7ter.52 (0)(i) pour la plage bipede). | lot `vh` + 7ter.89 (6) · `[MESURE+VERIFIE]` (reproducteur `tv.ref`, mode `vcen`) |
| **Les vehicules sont-ils tracables en POSITION ?** | ⚠ **DEUX AFFIRMATIONS ONT ETE CONFONDUES, ET SEULE LA PREMIERE EST MESUREE (7ter.90 (3)).** Le balayage TROUVE des enregistrements dans la bande declaree — c est ce que dit la ligne ci-dessous, et elle tient. Mais leurs **COORDONNEES ne sont des coordonnees que sur `000d5950`** : `filmdec` fige `bipedAxisWidths = [13,13,14]` et `QuantRangeCEBiped`, propres a la carte de ce film. Test de continuite (pas entre trames voisines contre deux trames du meme slot au hasard) : `000d5950` **0.042 contre 9.472** (1/225) ; `fccc61cd` **15.850 contre 64.208** ; `78919882` **38.404 contre 62.216** ; `9b191a7f` **21.348 contre 63.279** — vitesses implicites de 1 000 a 2 500 unites/s dans une boite de 113. **Calibration retrouvee** en 7ter.90 (4) : (13,13) sur le film temoin, (18,16) sur `fccc61cd`, (15,15) sur `78919882`, (17,17) sur `9b191a7f` — et **reproduite sur la bande VEHICULE seule**, population disjointe. ⚠ **LE << RANG 1 SUR 1608 >> EST REFUTE (7ter.94 (7)(c))** : l outil du lot imprime `rang 6/1608` et les onze premiers sont tous des `(13,13,w2)` ; `w2` etant libre, l espace effectif est de **144 couples `(w0,w1)`**. Ecrire << premiere FAMILLE sur 144 >>. Sur `78919882`, `(15,16)` egale `(15,15)` : **`w1` n y est pas determine**. L exactitude METRIQUE reste `[NON VERIFIE]` (aucune verite terrain hors `000d5950`). | 7ter.90 (3)(4) **corrige par** 7ter.94 (7) · `[MESURE]` |
| Les vehicules sont-ils tracables en POSITION ? (l enonce d origine, qui porte sur les ENREGISTREMENTS) | **OUI, et le controle negatif qui manquait le confirme.** `filmdec.ScanBipedRecords` pointe sur `[768,1023]` (options par defaut, zero modification de `filmdec`) contre une bande **VIDE de meme largeur** du meme film : **23 946 contre 12** et **5 566 contre 45** ; et **99.87 % / 97.3 %** des echantillons tombent sur un slot que les **keyframes declarent independamment `ti=40`**. ⚠ Sur un film SANS vehicule, la bande vehicule rend **5** et la bande vide **17** : les << 5 et 8 echantillons >> des films sans vehicule sont **SOUS le plancher de bruit**. | 7ter.89 (8) · `[MESURE]` |
| Existe-t-il un composant << siege / occupant / passager >> ? | **NON — enumeration exhaustive des 325 noms** : `mount`, `rider`, `occup`, `passenger`, `enter`, `exit`, `attach`, `driver`, `gunner`, `embark` rendent **ZERO** ; `seat` n en rend que trois, tous des reglages (`player-desired-respawn-seat`, `vehicle-seats-override-pitch/yaw`). **`object-parent-state-component` (i10) est le SEUL mecanisme du schema** capable d exprimer << ce bipede est dans ce vehicule >>. Porte par 9 archetypes : 35..43. | lot `vh` + 7ter.89 (9) · `[MESURE+VERIFIE]` |
| **LE LIEN JOUEUR-VEHICULE EST-IL ETABLI ?** | **NON, ET IL N EST MEME PAS SOUTENU.** Sur un film a 19 vehicules : **9** enregistrements de bipede a parente ACCROCHEE sur tout le film, **ZERO** parent designant un slot reellement lie a `ti=40`, et le **controle decale d un bit rend EXACTEMENT la meme valeur que le reel** (1/9 des deux cotes) — alors qu un bit d ecart doit detruire le champ (REGLE 3, 7ter.82 (1)). La generation du parent est etalee sur les 4 valeurs, **`gen0` inclus**, que l outil declare impossible. Et la marche ne ferme proprement que **5.9 a 9.0 %** des paquets : PATRON A / symptome E9. Le controle positif de l instrument (<< arme tenue >>) **echoue** : 21 variantes distinctes pour 21 observations. | 7ter.89 (9) · `[REFUTE]` comme etabli |
| Peut-on faire confiance a la grammaire de `i10` telle qu elle est ecrite ? | **PAS ENCORE.** `[porte 1][sonde 1][slot 13][generation 2][marqueur 16]...` est CONFORME au port Go, mais les **13 bits viennent de `quantStatDefaultWidth` (`filmdec/entity_quant.go`), la configuration PAR DEFAUT** (`DAT_144706100 = 0x1fff`), pas d une mesure sur `ti=40`. Or `FUN_1406d3140` prend sa largeur de la table du POOL — celle dont la colonne `base` n a jamais ete capturee (7ter.88 (1)). Pool different = largeur differente = tout le composant decale. | 7ter.89 (7) · `[NON VERIFIE]` |

### 2.6 COUVERTURE ET CALIBRATION

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Un enregistrement de degat (code 36 / `0xD2`) est-il un TIR ou une TOUCHE ? | **CELA DEPEND DE L ARSENAL — c est la correction de 7ter.101.** Sur arme a trace instantanee : **un TIR** (Tactical `records/shots_fired` **0.9300**, `records/shots_hit` **3.4655** pour 1/p = 3.601 ; egalite `|d|<=5` **45** contre **14.3** au fond permute, **0/200** ; et `records/tirs` reste **PLAT** 0.9305 -> 0.9457 quand la precision DOUBLE). Sur arsenal a projectiles : **une TOUCHE** (Fiesta `records == shots_hit` **174** contre **134.7** permutees, max 163, **0/200** ; BTB 43 contre 29.1, 0/200). **LA TOUCHE EST DANS LES DEUX CAS LE PORTEUR** (tableau A non vide) : `porteurs/shots_hit` 0.8509, et le taux de remplissage `porteurs/records` **EST** la precision du joueur (0.4267 contre 0.4462, erreur mediane 0.027, r = +0.82 ; par quartile 0.1724 / 0.2222 / 0.2727 / 0.3220 contre 0.1819 / 0.2465 / 0.2906 / 0.3558). | 7ter.98 amende par **7ter.101** · ETABLI |
| Le << type 105 >> du chantier replay 2D est-il notre code 36 ? | **OUI, a l unite** : `519 = 519` sur `000d5950`, `1 799 630 = 1 799 630` sur 927 films. Sa variante LONGUE (`0xD2`) est exactement notre code 36 ; sa variante COURTE (`0xD3`) est un autre code (38/39) et **n est pas** la touche. | 7ter.98 (1) · ETABLI |
| Le << plafond a 87 % des tirs qui ont touche >> tient-il ? | **NON comme propriete du FORMAT** : c est le **p05** d une distribution dont la mediane vaut **1.7260** record par touche, mesure sur un film **Super Fiesta** classe **26 / 927**. **MAIS OUI POUR SA FAMILLE DE MODE** (7ter.101) : en Fiesta le record suit les TOUCHES (`rec/touche` agrege **1.1046**, egalite `|d|<=5` 174 contre 134.7 permutees, 0/200) — le chantier voisin travaille en Fiesta, sa lecture y est la bonne. | 7ter.98 (2)(3) amende par **7ter.101** · CORRIGE |
| Combien vaut le compte de records ? Et celui des porteurs ? | **DEUX DEFINITIONS, ET IL FAUT DIRE LAQUELLE.** Corpus 927 films : **1 799 630 BRUTS** (porte attaquant ouverte OU fermee) contre **1 774 183 INDEXES** — 1.41 %. Sur les 150 films de 7ter.86 (1) : porteurs **TOUS 98 685** contre porteurs **INDEXES 97 556** — c est **la cause de l ecart de 1.14 %** laisse ouvert par 7ter.98 (7), et **ce n est PAS la borne `s34`** (aucun effet entre 16 et 64). | 7ter.101 (6) · ETABLI |
| Le decodeur est-il portable d un film a l autre ? | Oui via `cv_autocalib` : **quatre films, quatre couples (axisW, indexW) DISTINCTS**, zero variable d environnement. Le parametre est reellement map-dependant. | 7ter.40(6) + 7ter.42(2) · MESURE+VERIFIE (7ter.43) |
| Que designe `axisW` physiquement ? | *(ici << precision >> = **QUANTIFICATION**, pas le tir — pour la precision de tir voir §5.7, §14, §17 ; cf. §6.1 piege 5)* Le **NIVEAU L de precision** du composant position dans la config de replication de la MAP, avec `W = 6 + L`. Le balayage 6..26 cherche dans le bon espace. Le niveau du registre `chunk_00` vaut **0** sur les 4 films : ce n est PAS lui. | 7ter.54 AXE3 · MESURE |
| Les tables de precision dumpees remplacent-elles le balayage ? | **NON.** Decodees et validees **7/7** par la forme fermee `W = min(26, bitLen(ceil(span/(2*step))))` — mais leur contenu est installe PAR MAP : injectees, elles effondrent 9b191a7f (77 -> 11) et 78919882 (91 -> 41). | 7ter.54 AXE3 (3d) · MESURE |
| `recordStateParam` : comment le calibrer ? | Par un critere de **CROISSANCE des slots** (`calib2` en est totalement aveugle : score identique pour rsp 0..5). Vaut 3 / 4 / 2 / 4 selon le film. PIEGE : le garde-fou de platitude annonce « PLAT » **tout en tranchant**, et ce choix pese ~8 morts. | 7ter.40(5) + 7ter.42(2) · MESURE+VERIFIE |
| Les cinq causes qui ont porte le gate (a) de 77/93 a 89/93 ? | (1) balayage de keyframe restreint a `ti=35` · (2) fenetre de vie derivee des SEULS keyframes (7/7, p ~ 4e-11) · (3) repli de largeur libre du localisateur · (4) test de generation sur `TryDeltaAt` · (5) `recordStateParam != 0`. | 7ter.40 · MESURE+VERIFIE (7ter.41) |
| Le monde est-il un etat unique ? | Non, c est une **TIMELINE** : keyframes appliques dans l ordre du temps + pre-chargement par la PREMIERE declaration de chaque slot. 59 -> 79 morts atteintes. | 7ter.36(1)(2) · MESURE |
| Le bloqueur dominant est-il une liaison manquante ? | **Non, c est un symptome de LARGEUR.** 25.6% de ces desyncs portent un slot qui RECULE ; les slots demandes (26, 27, 0, 2, 2048, 6807-6809) ne sont pas des slots de biped, et sur 462 records NEW ti35 atteints proprement, **2** seulement ont un slot dans la plage. | 7ter.40(9) + 7ter.54 AXE2 (2d) · MESURE+VERIFIE |
| La plage de credibilite `[512,610]` est-elle bonne ? | Codee en dur dans `reportEvDeads`. Derivee des keyframes elle vaut `[512,610]/[512,609]/[512,614]/[512,615]` et rend **+4 morts (334/372)**, gate (b) INCHANGE. Devient OBLIGATOIRE des que le roster n est plus a 8. | 7ter.52(0)(i) · MESURE |

### 2.7 ROSTER ET BIJECTION

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Le roster est-il derivable offline ? | Oui : les xuids du kill-feed apparaissent dans le paquet **type-8** dans l ORDRE EXACT des slots, bit-shiftes en LE. Valide 8/8. | §8.2 · MESURE |
| La bijection indice -> joueur est-elle unique ? | **Oui a 8 joueurs.** Enumeration exhaustive des 8! ; medianes 1-2. Le rang 2 (~51-60) vaut EXACTEMENT le **plancher mecanique** = les couples justes n impliquant aucun des deux joueurs echanges — donc la transposition recupere 0 des couples contestes. | 7ter.43(4) · MESURE+VERIFIE |
| La reserve « une transposition preserve la quasi-totalite des couples » est-elle juste ? | **NON, elle est fausse sur ses deux moities** (les joueurs echanges se tuent ; la transposition perd 27-37% des couples). Reserve a SUPPRIMER, pas a nuancer — a 8 joueurs. | 7ter.43(4) · MESURE+VERIFIE |
| Le gate (b) est-il une precision ? | Non : c est le **MAXIMUM d un objectif ajuste**. Le seul chiffre lisible est le PLANCHER de sur-ajustement (horloge cassee hors tolerance) : **9.6% a 8 joueurs, 7.9% en BTB**. | 7ter.53(3) · MESURE+VERIFIE |
| Piege de parseur de roster ? | **UN GAMERTAG CONTIENT DES ESPACES** (« Zeus Herd »). Decouper la ligne `bijection indice -> joueur` sur les blancs fait tomber l appariement de **89 a 47 sur 99, sans aucun message d erreur**. Decouper sur les marqueurs `<chiffre>=`. | 7ter.42(10) · MESURE |
| L indice d attaquant du flux de degat est-il l ordre du roster de la base ? | **NON, et nos mesures ne s en servent pas.** L ancrage des mesures par joueur est **film-seul** (`resolveXuidPIStrict` : motif xuid cherche au bit pres, 5 bits precedents, indices deux a deux distincts) : 935/949 films a solution unique, 94.1 % des participants. Il reproduit **8/8** la table publiee par le chantier voisin sur `000d5950`. Piege a connaitre : l ancrage NAIF (`weaponv3.ResolveBest`, 1re occurrence) rend l indice **0 pour les 8 joueurs** de ce film. | 7ter.98 (6) · ETABLI |
| A-t-on VERIFIE que l appariement change quelque chose, sur la quantite PUBLIEE ? | **OUI, et il change tout** (7ter.101). Population publiee (roster <= 16, sans fantome, 579 films, n = **4 607**), quantite `porteurs/records` contre la precision API : ancrage **film-seul r = 0.7740** (MAE 0.0802) · **ordre de la base r = 0.5731** (MAE 0.1141) · **nulle permutee r = 0.5730** (MAE 0.1142), **0/200**. **L ordre de la base EST la nulle au quatrieme chiffre.** Les AGREGATS, eux, sont invariants (0.3798 contre 0.3799) : ils ne prouvaient rien. Reproduit le controle deja publie par 7ter.81 (5) (0.791 contre 0.582, 0/200). | 7ter.101 (5) · ETABLI |

### 2.8 MEDAILLES

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Le film porte-t-il des medailles horodatees ? | **Oui** : chunk HIGHLIGHT, **151 events `medal`** sur 4 films, avec XUID et gamertag — deja decodes par `analysis.ParseHighlightEvents` (`EventTypeMedal`, champ `MedalType`). Il n y avait rien a craquer. | 7ter.52 (B1) · MESURE+VERIFIE |
| `MedalType` (1 octet) -> `medal_name_id` ? | Etabli **32/32 UNIQUE** par le VECTEUR de comptes par (film, joueur) ; hasard max 8-9. Exact aussi **par occurrence : 151/151** (les 26 « manquantes » cote API sont la pseudo-medaille `Avenger` fabriquee par le projet). | 7ter.52 (B2) + 7ter.53(5) · MESURE+VERIFIE |
| Une medaille recupere-t-elle une mort non atteinte ? | **NON** : elle QUALIFIE une mort, elle ne la DECODE pas. Le paquet reste bloque. | 7ter.52 (B5) · MESURE+VERIFIE |
| Que valent les medailles comme oracle ? | **Oracle offline de CATEGORIE, 19/19 sans contradiction** (Back Smack + Sneak King -> melee, Stick -> projectile fixe, Snipe -> headshot, Chain Reaction -> projectile chaine). **JAMAIS un oracle de declencheur.** | 7ter.52 (B5) + 7ter.52bis · MESURE+VERIFIE |
| Que vaut l ATTRIBUTION par medaille ? | 8 joueurs : 12/42 morts manquantes portent une medaille, **3 contraignent, 0 en source exacte**. BTB : 54/117, 20 contraignent, **13 en source exacte**. HAUTE PRECISION, FAIBLE RAPPEL. | 7ter.54 AXE4 · MESURE |
| La classe de contrainte d une medaille est-elle portable entre films ? | **NON** : le type 108 est SOURCE EXACTE sur le BTB (`7654f890` x9) et seulement CATEGORIE SEULE a 8 joueurs. C est l ARSENAL du mode qui porte la contrainte -> apprendre PAR FILM, jamais une table globale. | 7ter.54 AXE4 (D12) · MESURE |
| L ordre de la table de medailles de l en-tete = l ordre de `MedalType` ? | **NON** (`Shot Caller` 5e entree = type 140 ; `Kong` 114e = 101). Et aucun couple (decalage -288..+288 bits, largeur 7..16) ne rend `MedalType` : meilleur accord **2/32**. Le maillon 100% offline passera par un ordre present dans l .exe. | 7ter.52 (B4) + 7ter.53(6) · MESURE+VERIFIE |

### 2.9 BOTS

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Le chunk HIGHLIGHT porte-t-il les bots ? | **NON, il est HUMAIN SEUL** — et c est mesure SANS aucune ancre XUID : l enumeration exhaustive des end-markers rend exactement les memes **194 et 247** events que le parseur ; les chaines UTF-16LE se ferment au bit pres (194 gamertags pour 194 events). Pas la place d un 9e nom. | 7ter.59 (mesure 3) · MESURE |
| Ou est le bot, alors ? | Dans la voie **REPLICATION** : le dead-state i11 identifie un participant par son **INDICE ABSOLU** (`comp+0x04` victime, `comp+0x08` tueur, `R(1) present ; si 0 -> R(5)`), espace **0..31** — il porte structurellement le 9e participant. | 7ter.42(7c), re-verifie 7ter.43(6), ferme 7ter.59 · MESURE+VERIFIE |
| Que manquait-il, alors ? | **Un GATE d affichage** : `cmd/tmp_deadstate/evrun.go` l.174-177 fige `EnumA <= 7 && EnumB <= 7`. L indice 8 est CORRECTEMENT DECODE PUIS JETE. Mesure : `9b191a7f` 3 kills/3 morts, `fccc61cd` 1/2 — exactement l API. | 7ter.59 · MESURE |
| Faut-il corriger `highlight_event_parser.go` ? | **NON.** Relacher `minXUID`/`maxXUID` n ajouterait RIEN (l enumeration sans ancre rend deja le meme total). Fichier laisse INTACT, comptes reproduits. | 7ter.59 + 7ter.59bis · MESURE |
| Une mort INFLIGEE PAR un bot est-elle decodable ? | **OUI, ET ELLE EST DESIGNEE PAR LE KILL-FEED LUI-MEME** : une MORT que la reconstruction de couples ne consomme jamais = un tueur non humain. Cardinal exact des kills API du `bid(...)` (3 et 1), **VIDE sur les deux films sans bot**. Publiee sous `OriginBotKiller`, 4 lignes, toutes servies par la MARCHE. | 7ter.79 · MESURE |
| Que devient la comptabilite des morts ? | Elle **FERME** : `couverts + morts DE bot + morts PAR un bot == morts de l API`, film par film (93/90/99/98) et en cumul (371 + 5 + 4 = **380/380 = 100.0 %**). Morts publiees par joueur = `deaths` de l API, **ecart ZERO sur 34 participants**. | 7ter.79 · MESURE |
| La symetrie du kill-feed humain-seul etait-elle connue ? | **OUI, DEPUIS 7ter.59**, chiffree (<< trois orphelines de chaque cote >>). Seule la moitie << KILL orphelin -> mort DE bot >> etait devenue une population dans le code ; l autre est restee une phrase pendant tout le chantier. | 7ter.79 (9) · LECON |
| Peut-on obtenir le NOM d un bot offline ? | **OUI, RESOLU.** Le paquet **`BOT_METADATA` = PacketType 12** (pas un ChunkType : il est A L INTERIEUR d un chunk de replication decompresse, il etait sur le disque depuis le debut). Il porte `nbBots`, le **SLOT**, l identifiant `bid(N.0)` et le NOM — le tout en **BIG-ENDIAN**, nom en **UTF-16BE**. Critere pre-enregistre atteint : `343 Aloysius`/bid(39.0) sur 9b191a7f et `343 PardonMy`/bid(7.0) sur fccc61cd, **les deux declarant `slot=8`**. L identification << indice 8 = le bot >> est donc LUE, plus deduite d une coincidence de K/D. | 7ter.62 · MESURE |
| Levier « bots » pour l objectif 99% ? | **CETTE LIGNE EST PERIMEE.** Elle disait « NUL voire NEGATIF : cela grossit le DENOMINATEUR sans grossir le numerateur ». C est faux dans les deux sens : les morts de bot NE SONT PAS au denominateur des couples (7ter.66), et depuis 7ter.79 les deux populations de bot portent le taux des morts de l API de 98.9 % a **100.0 %** sans toucher aux trois autres denominateurs. | 7ter.58 + 7ter.59, **CORRIGEE par 7ter.66 puis 7ter.79** |

### 2.10 BTB (test de charge)

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Le BTB, c est 24 joueurs ? | Non : **24 slots pour 36 participants dont 8 BOTS** ; 300 morts au feed, 25 joueurs distincts, 303 kills / 304 morts a l API (donc 1 suicide). | 7ter.52 (C1) · MESURE+VERIFIE |
| Pourquoi le BTB s effondrait-il a 4.4% ? | **Bug de NOTRE calibration** : `cv_autocalib` avance le monde a la FIN du film (`tl.advanceTo(^uint64(0))`) puis echantillonne des paquets du DEBUT. En BTB les slots de biped sont RECYCLES AVEC UNE NOUVELLE GENERATION -> 387/400 paquets meurent au 1er record. Corrige : **4.4% -> 60.2%**. | 7ter.52 SONDE C · MESURE+VERIFIE (7ter.53) |
| Le correctif regresse-t-il sur les 4 films ? | **NON** : le meme balayage sur monde chronologique choisit la MEME calibration partout (14/1, 17/2, 16/2, 17/1). **Il reste A POSER** dans `cv_autocalib.go`. | 7ter.53(2) · MESURE+VERIFIE |
| Le 60.2% du BTB est-il exploitable ? | **Comme AGREGAT oui, ligne par ligne NON** : la marge de la bijection est NULLE (une transposition preserve 177), au moins deux joueurs interchangeables. Aucune attribution individuelle du BTB n est publiable. | 7ter.52 (C3)(C7) + 7ter.53(3) · MESURE+VERIFIE |
| Le gate (a) du BTB est-il dans la meme unite qu a 8 joueurs ? | **NON** : le filtre `EnumA, EnumB < nPlayers` sur un `R(5)` (0..31) est **~9.8x plus laxiste** a 25 joueurs ((25/32)^2 = 61% contre 6.25%) et la plage biped passe de ~92 a 257 slots. | 7ter.53(3) · MESURE |
| Les vehicules produisent-ils une classe de bloqueur neuve ? | **OUI** : `vehicle-type-physics` (1016) et `vehicle-emp-timer` (363), 2e et 4e coupables, absents des 4 films a 8 joueurs. (La 1re mesure disait le contraire — artefact de marche morte, cf. §4 E9.) | 7ter.52 (C7)(D9) · MESURE+VERIFIE |
| Pieges d outillage sur un BTB ? | `loadPackets`, `loadKillFeed` et `cmd/tmp_hl` **bornent tous a `chunk 41`** ; un BTB a 63 chunks et son HIGHLIGHT est le **n62** -> kill-feed purement INTROUVABLE. Et les parametres de replication de `filmdec` sont des **GLOBAUX de package** : enchainer deux films dans le meme process contamine la calibration du second. | 7ter.52 (0)(iii)(ii) · MESURE |

---

## 3. TABLE DES PISTES MORTES — NE PAS REOUVRIR

| PISTE | POURQUOI ELLE EST MORTE (avec le chiffre) | SECTION |
|---|---|---|
| Queue « arme » du kill-event (DamageSource / DamageEffect / modifier) | Existe dans l .exe, **ABSENTE des films** : 0/133 au vrai curseur, 5 films, recents comme anciens. | 7ter.23, 7ter.24bis |
| `damageClass()` / les « 115 records non-firearm » | Artefact de decalage de bits : `0x42c9679f<<3 = 0x164B3CF8` (faux grenade), `<<5 = 0x592CF3E0` (faux melee). Les armes non-firearm ont **0 occurrence** dans tout le film. | 7ter.26(6) |
| « Region pre-liste » avant la boucle d events | N existe pas : `FUN_14076a1c4` ne lit RIEN avant sa boucle (desassemblage). Layout = `[1 bit][EVENTS][RECORDS ECS]`. | 7ter.27 |
| Scan de motif comme localisateur d un record ECS | En-tete « plausible » 1 fois tous les 18 bits ; 1/8366 avec toutes les contraintes ; 193/292 candidats = la MEME paire (motif binaire repete). Il faut la marche sequentielle. | 7ter.27(6) |
| Attribution same-clock (dernier record de degat avant le kill) | Le « 94% » etait mesure contre un oracle de MEME LOGIQUE. Verite terrain : **11/14 = 78.6% contre une baseline triviale de 10/14 = 71.4%** -> NON VALIDEE. | 7ter.19, 7ter.26(7) |
| Held-weapon / fire-events | Rejete ~10x par l utilisateur (doctrine) + plafond mesure 9-13/93 : le film n enregistre pas chaque tir. | §9 |
| Solveur exhaustif de slot non lie (50 archetypes, « fermeture unique ») | « Resout » **1169 slots repartis sur tout [0,8191]** = des artefacts. A/B : 79/93 -> **48/93**. Bascule `DS_SOLVE=1`. | 7ter.36(5) |
| Combler la plage biped (lier les 9 slots jamais declares) | **+8 dechets**, 79 -> 78 morts. La bonne forme est la FENETRE DE VIE derivee des seuls keyframes (7/7, p ~ 4e-11). Bascule `DS_BIPEDFILL=1`. | 7ter.36(5) puis 7ter.40(2) |
| Conserver les liaisons NEW (paquet propre, ou prefixe propre) | **330 -> 328 -> 315 / 372** ; BTB 177 -> 177 -> **160**. Les DEUX politiques sont mortes. `DS_KEEPBIND=1`. | 7ter.54 AXE2 (2c) |
| Lire le record NEW qui cree les slots bloquants dans un paquet a events | Seuls **1.1%** des paquets se marchent sans desync ; **0 NEW atteint** sur ces slots, sur tout le film. Mode `cvnew`. | 7ter.40(0) |
| Resoudre la largeur du corps runtime-gate de i54 par contrainte | Histogramme des largeurs gagnantes **DIFFUS**, aucun pic (126 x146, 150 x110, 69 x78...). A porter STATIQUEMENT depuis Ghidra. Mode `cvmob`. | 7ter.40(9) |
| Table du jeu `DAT_14494a908` comme resolveur effet -> arme | Pointeur RUNTIME : **64 octets NULS** lus dans l .exe. Voie morte statiquement. | 7ter.36(7), verifie 7ter.38(1) |
| Medailles dans les archives `.module` | 0 groupe de medailles sur 312 recenses ; 0 occurrence de `kong` dans `Content/__cms__`. (Mais VIVANTE cote FILM, cf. §2.8.) | 7ter.48(3) |
| Audio Wwise jouable | Codebooks Vorbis **EXTERNES** (le `fmt ` etendu finit par des constantes, deux jeux, un par nombre de canaux) ; 7 essais ffmpeg, **7 echecs, aucun fichier produit** ; ni vgmstream ni ww2ogg presents. Le mapping vaut comme NOMMAGE, pas comme son. | 7ter.50(1) |
| « periode = somme des ecarts distincts » comme regle de format | Appliquee a 460 tableaux : compatible dans **67.6%** des cas seulement (149 echecs). Juste sur `gggl` — et corroboree la par deux mesures independantes — mais **a ne jamais citer comme un resultat de format**. | 7ter.49(2)c |
| Dead-state `+0x10` (GID) = l arme | CE : `+0x10` **CONSTANT** (116963283), ne distingue pas le modele. | §8.6, §9 |
| Position-linking (record de degat -> victime par position monde) | pos impact **0/519** decodable, pos victime 22/93, match **0/93**. | §9 |
| Warp 2 horloges | Plafond d HORLOGE : 96% Fiesta / **58% Team Slayer**, residu 1-2 s irreductible. | §9, §10 |
| « Signature d arsenal par mode de jeu » (controle semantique) | Ne generalise pas : une carte FORGE est aussi dispersee qu un Fiesta (78919882 : 24 armes sur 89 morts, top-2 = 25.8%). **Controle RETIRE de la liste.** | 7ter.41(4) refute par 7ter.42(8) |
| « Les morts gagnees se concentrent en fin de match » | Propre a `9b191a7f`. Sur les deux films recents c est l INVERSE (les MANQUES se concentrent a la fin : 28.6% et 40.0% sur 08-10 min). | 7ter.41(6) puis 7ter.43(8) |
| « Les auto-kills expliquent des morts manquantes » | Taux de base **0/372**, 0 parmi les 42, Fisher p = 1.0 ; corrobore par l API (`kills == deaths` sur les 4 matchs). **RESERVE (7ter.53 R4)** : le detecteur de feed n a AUCUN controle positif — sur le BTB, ou l API prouve 1 suicide, il rend 0. **Le verdict ne tient que sur la jambe API.** | 7ter.52 SONDE A |
| « Les couples reconstruits par voisinage echouent plus » | **FAUX** : 5/64 = 7.8% contre 42/372 = 11.3%, p = 0.886. Ils echouent MOINS. | 7ter.52 (A2) |
| « Une medaille Kong horodatee recupere une mort non atteinte » | **FAUX** : elle QUALIFIE, elle ne DECODE pas. Les 2 Kong tombees sur une mort non atteinte la laissent non atteinte. | 7ter.52 (B5) |
| « Kong NOMME deux des cinq tags `??` » | **FAUX, l etat cite etait perime** : `0d203522` et `5e389b5d` sont etiquetes depuis 7ter.45(3) et confirmes en Theater en 7ter.44 puis 7ter.47 Q5. Un des quatre points d appui etait deja au journal. | 7ter.53 (R1) |
| **Verifier les DEUX PARTS DE DEGATS en Theater** | **IMPOSSIBLE, structurellement, pour deux raisons independantes** : l ecran qui les affiche est le **RECAP DE MORT EN PREMIERE PERSONNE**, que le Theatre NE REJOUE PAS ; et il n existe que **DEUX chaines `PercentageDamageDone`** dans tout l executable, **sans aucun libelle localise** — il n y a rien a lire a l ecran, meme en jouant. Ce resultat ne recevra jamais de verite terrain de la forme habituelle. | 7ter.75 (4) |
| << 31 assists decodes contre 30 API >> (7ter.24ter) | Le 31 comptait les faux positifs d un pipeline a 99 kills pour 87 morts reelles ; nettoye, c est **22 pour 30**. Et l ancre par joueur portait sur un AUTRE joueur. **Ne plus citer ce chiffre.** | 7ter.76 (1) |
| Axe LANCE / DETRUIT de R-EFFET | Le MEME tag couvre les deux gestes. Refute par mesure offline croisee (7ter.52bis), verrouille par `gvkong` (exactement UN kill candidat par Kong, 6/6), puis ferme en Theater. | 7ter.52bis, 7ter.53(R2), 7ter.56 |
| Compteur de MUNITIONS a portee du fire-event | **N EXISTE PAS** : 3 681 candidats balayes (offsets -64..+352, largeurs 4..12), meilleur P(pas = -1) = **0.1305** = le niveau du bruit. Ce qui est la, c est un compteur de **TIR croissant** (cf. §14). | 7ter.80 (11) · `[MESURE]` |
| Le champ `HitLikely` de `weapon_scanner.go` | **MORT, et il le reste** : 75-79 % de touches annoncees contre une precision reelle mediane de **0.446**. ⚠ **MAIS LA QUESTION N EST PLUS FERMEE** : un AUTRE bit (`eventStart+106`) bat un taux constant — reproduit par un second instrument, et il bat aussi le taux sur l EGALITE EXACTE (82 contre 29, McNemar z = 5.08). **Portee : mode standard, armes a trace instantanee ; en Fiesta il est BATTU** (§14.2). | 7ter.80 (8)(9) + 7ter.82 · `[ETABLI]` |
| **« MA40 0.928 · Sidekick 0.925 · BR75 0.981 · Bandit 1.012 »** | **NE PLUS CITER — IRREPRODUCTIBLE.** N existait que comme assertion dans `GUIDE_WEAPON_SHOTS` §3 point 3 : aucune section du RE_LOG, aucun outil, aucune methode. Deux origines testees (indice 4 bits **REFUTE** ; arme dominante sur denominateur total **PARTIEL**). Mesure courante : MA40 **0.971** · Sidekick **1.004** · BR75 **1.007** · Bandit **1.007**. | 7ter.80 (6) · `[NON VERIFIE]` |
| « l indice attaquant du flux `0xD2` = `bitsAt(pl, 36, 5) >> 1` » (note de memoire projet) | **FAUX** : **53 376 valeurs paires contre 56 799 impaires**. Il n y a pas de division par deux — le champ est l indice **DIRECT**. Si le `>> 1` avait ete vrai, aucune valeur impaire n existerait. | 7ter.81 (7) · `[REFUTE]` |
| « la deduplication du scan de tirs perd des events » | **FAUX, par comptage exhaustif** : sur **7 343 955** marqueurs, **ZERO** event supprime par la deduplication de proximite d octet. | 7ter.80 (3) · `[MESURE]` |
| « la cadence explique le deficit des armes automatiques » (lecture naive d `acurtis`) | **FAUX telle quelle** : le BR75 a l ecart median **le plus court** entre enregistrements (60 ms contre 80 ms MA40, 190 ms Sidekick) **et le deficit le plus petit**. Cadence et deficit ne sont pas ordonnes pareil. | 7ter.81 (9) · `[REFUTE]` |
| **Chercher les compteurs de TIRS / de TOUCHES dans les STATLINES du film** | **MORTE — balayage EXHAUSTIF** (toutes les positions du bloc, largeurs **4 a 32**, puis **4 096 bits** de charge utile), 4 films, 23 lignes de joueur appariees : `shots_fired` **4/23**, `shots_hit` **4/23**, precision **7/23** — contre un **taux de fond permute de 3/23 · 4/23 · 7/23**, soit le MEME NIVEAU. **L instrument n est pas aveugle** : le meme balayage rend `kills` **23/23** et `score` **23/23** (permutes 9/23 et 4/23). Argument de fermeture : le **dernier keyframe EST l etat final** — un compteur present y serait aussi exact que les kills. | 7ter.83 (1) · `[MESURE]` |
| **Chercher une statistique de tir PAR ARME dans les composants REPLIQUES** | **MORTE — ENUMERATION EXHAUSTIVE, pas une recherche infructueuse** : **325 noms de composants**, **118 archetypes** du registre `chunk_00`, **ZERO** composant portant une statistique de tir. Cote arme il n existe que : identite **par emplacement**, **munitions**, **inventaire de chargeurs**, **surchauffe** — l ETAT de l arme, jamais son HISTORIQUE. | 7ter.83 (2) · `[MESURE]` |
| **La touche de projectile dans le flux code-36, sous une AUTRE moitie basse d identifiant** | **MORTE, et cette fois SANS ANCRE** : les **48** moities basses sont enumerees (**150 films** : 95.34 % = `42c9679f`, l ARME). Les enregistrements NON-ARME sont **17 167 sur 250 films dont 3 203 porteurs** — pour **~15 300 touches de projectile attendues** (extrapolation des ~58 000 / 949 films de 7ter.81 (8)), soit un facteur **4.8** trop petit, et 82 % d entre eux sans aucun degat. | 7ter.84 (1) · `[MESURE]` |
| **<< L enregistrement porteur d un projectile existe, mais DIFFERE a l instant de l impact >>** | **`[REFUTE]`, et le controle avait un degre de liberte** : ecart median a l enregistrement d ARME anterieur du MEME attaquant = **33.0 s** (porteurs seuls 24.8 s), **controle attaquant PERMUTE = 15.0 s** — le reel est DEUX FOIS PLUS LACHE que le hasard. Aucun couplage a l echelle d un temps de vol. | 7ter.84 (2) · `[REFUTE]` |
| **Un AUTRE code d event porte le flux de touches de projectile** | ⚠ **RESSUSCITEE — c est le cas, et ce sont les codes 6 et 7 (7ter.86 (3)). NE PAS RECITER CETTE LIGNE COMME UNE PISTE MORTE.** Ce que 7ter.84 (3) a reellement mesure, et qui tient : aucun autre code ne porte de **constante 32 bits dense** (balayage sans ancre, Misra-Gries + comptage exact, 12 codes, controle positif code 36 bit 76 -> `42c9679f` part 0.8993). **Mais << pas de constante dense >> n est pas << pas de flux >>** : les codes 6/7 ne portent pas de constante d arme, ils portent un tag, un vec3 et des drapeaux. La conclusion generale tiree de ce balayage etait trop large. | 7ter.84 (3) + **7ter.86 (3)** · `[MESURE]` |
| **`f5c335df` / `e7232c0b` = la source chainee du Disruptor, appariee 1:1 au tir** | **`[REFUTE]` par changement de corpus** : l egalite 4 049 = 4 049 sur 150 films ne survit pas — 250 films donnent **8 517 contre 6 501**. Coincidence. **La lecon : changer la taille du corpus AVANT de conclure sur une egalite de comptes.** ⚠ **CORRECTION 7ter.86 (6)** : cette famille n est PAS << inconnue du catalogue >>. `f5c335dfe7232c0f` = **MA5K Avenger** dans `weapon_data.go`. Les deux instruments cherchent le nom par `famille<<32 \| 42c9679f` et sont donc AVEUGLES a toute arme dont la moitie basse differe — recherche ancree, exactement le defaut reproche a 7ter.81 (8). Le MA5K est une arme a trace instantanee : son taux de porteur en bande hitscan n appelle aucune hypothese de projectile. Ecart non explique : 0.3238 mesure en 7ter.86 contre 0.3447 annonce en 7ter.84. | 7ter.84 (4) + 7ter.86 (6) · `[REFUTE]` / `[CORRIGE]` |
| Marcher vers les events NON PREMIERS via le `endBit` du deser code-36 | **CAUSE TROUVEE, VOIE REMPLACEE.** L echec vient de l instrument : **le port du code-36 s arrete au tableau A**, alors que `FUN_14080c1f8` continue avec au moins 8 groupes de champs (boucle `+0xf8`, `FUN_1406cd5b8`, `FUN_1408eff64`, `R(0x1e)`, 2 x `R(6)` dequantifie, `R(6)`, `R(32)` optionnel, bloc 16 bits). **Ce qui marche a la place** : le walker generique `evStep` du depot (`killsource/eventchain.go`) — 9.6 M d evenements deroules sur 949 films. ⚠ **Et le critere d alignement de 7ter.84 (9) etait le mauvais** : la part de terminaisons propres est MAXIMALE hors alignement (bit 2 : 0.8643 contre 0.5724 au bit 1), parce qu un depart faux lit un bit de continuation a 0 et termine avec ZERO evenement. Le discriminant est le **nombre d evenements lus** (1.624 contre 0.184) et le compte des codes 6/7 (facteur 30 a 300). | 7ter.84 (9) + **7ter.86 (3)(7)** · `[MESURE]` |
| **Le `rho` de Spearman film/API PAR ARME comme preuve de l ordre des armes** | **`[REFUTE]` comme preuve** : une nulle qui permute la precision API entre joueurs du MEME film le reproduit ou le bat — **200/200** a purete >= 50 % (hitscan), 189/200 a >= 95 %. Sur 5 armes il ne prend que quelques valeurs. Le chiffre qui survit est la **MAE entre armes**, et seulement contre la nulle P0. | 7ter.85 (3) · `[REFUTE]` |
| **Publier la precision par arme calculee sur le CORPUS ENTIER** | **MORTE — elle INVERSE l ordre de deux armes.** Naif : MA40 **0.4485** contre Sidekick **0.3701** (MA40 +8 pts). Reference API : MA40 **0.4196** contre Sidekick **0.4491** (Sidekick +3 pts). Ecart absolu moyen **0.0361** contre **0.0144** pour le chiffre restreint aux joueurs a arme dominante >= 80 % : **2.5 x plus faux**. Cause : le taux par arme depend de la POPULATION DE TIREURS (Sidekick 0.370 -> 0.484 selon la seule purete). | 7ter.85 (6) · `[MESURE]` |
| **Chercher dans le FLUX D EVENTS l evenement qui CREE le projectile (donc son tireur)** | **MORTE.** Critere : un objet est cree UNE fois, donc un evenement de creation **couvre TOUS** les projectiles. **Aucun troisieme code** : meilleur suivant 0.2318 (permute 0.1601), fire-event 0.0931 (permute 0.0711) ; balayage COMPLET des ecarts de base : **aucun pic**, rapport reel/permute 1.006 a 1.064. ⚠ **LA JUSTIFICATION A CHANGE (7ter.89 (2)(3))** : le << controle positif >> `0.5408 + 0.4606 = 1.0014` **est une IDENTITE** (la reference est l union des deux ensembles compares : la somme vaut `1 + recouvrement` quoi qu il arrive, reproduit a 9 decimales sur 2 corpus). Ce qui rend le negatif exploitable est le controle pose en 7ter.89 : couverture d une reference **INDEPENDANTE** avec nulle de **MEME POOL** — meilleur candidat **0.5046** contre nulle **0.3600**, rapport **1.40**, quand un nommage reel exige 1.0. **Le projectile n existe dans le flux qu a l instant de sa mort.** | 7ter.88 (4)(5) **corrige par** 7ter.89 (2)(3) · `[MESURE]` |
| **Relier un JOUEUR a un VEHICULE par la parente `i10` (le seul mecanisme du schema)** | **NON REOUVRABLE EN L ETAT — la population n existe pas.** `i10` est bien le seul candidat (enumeration exhaustive des 325 noms : `mount`/`rider`/`occup`/`passenger`/`enter`/`exit`/`attach`/`driver`/`gunner`/`embark` = **ZERO**), mais sur un film a 19 vehicules la lecture rend **9** bipedes a parente accrochee, **0** parent lie a un slot `ti=40`, et un **controle decale d un bit IDENTIQUE au reel** (1/9 contre 1/9). Marche a **5.9-9.0 %** de fermeture propre : la distribution est lue sur une marche morte (PATRON A / E9). **Prealable non negociable avant toute reouverture : faire remonter le taux de marche propre**, sinon la population restera a un chiffre. | 7ter.89 (9) · `[REFUTE]` comme etabli |
| **Chercher le TIREUR dans la PARENTE (`object-parent-state-component`, i10) du projectile** | **A NE PAS OUVRIR SANS RAISON NEUVE — c est semantiquement la VICTIME.** i10 relie un objet a celui auquel il est ACCROCHE : pour un projectile c est l aiguille plantee ou la grenade collee (`AttachedDamage`), donc la cible. Chercher le geste d un TUEUR dans un champ de la VICTIME est le **PATRON C / symptome E7**, deja paye trois fois. Et l archetype 41 ne replique **aucun** proprietaire sur ses 22 composants ; le seul candidat credible, `object-multiplayer-properties-component` (i9), a ete resolu (`FUN_140f53308` -> `FUN_1407d4c94`) : `R(1)` de porte puis `R(5)` de **selecteur d union** de mode de jeu. Le proprietaire existe cote SERVEUR (`FUN_1404969f0(projComp + 0xe0)`) — REGLE 6. | 7ter.88 (6) · `[RAISONNEMENT]` |

| **Ventiler les touches PAR ARME avec le TAG du corps des codes 6 / 7** | **MORTE — le tag n est JAMAIS transmis.** Le corps est decode champ par champ (grammaire lue dans les immediats de `FUN_142f1c6cc`, somme = **118 = `evFixed[7]` a l unite**) ; le bit qui commande la lecture du tag vaut **1 sur 168 380 observations de profondeur 0** (97 988 codes 7 + 70 392 codes 6), **zero exception** — et la profondeur 0 est la seule position qui ne depende d aucune longueur de corps. Controle positif du meme instrument : **129 381 / 129 384 = 1.0000** (corps reels reecrits avec un tag `jpt!` connu, tous relus et reconnus). Alignement prouve par un champ non postule : `H(f1)` = **0.4745** a decalage 0, croissance monotone des deux cotes. **Le test decisif rend un ZERO EXACT** : aucune arme ne bouge d un centieme (Needler 0.0070 -> 0.0070, MA40 0.4292 -> 0.4292). Aucun autre champ du corps n est un identifiant d arme — `w16`, le seul assez large, porte des petits entiers (2 199 valeurs, 0/19/1/61). **REPRODUIT A L UNITE PAR UN BINAIRE INDEPENDANT** (7ter.94 (1)(2), `cmd/tmp_refc7`) : memes **97 988** et **70 392**, meme **zero** sur 168 380, controle positif **70 950/70 950 = 1.0000**, `H(f1)` = **0.1925** a decalage 0 (minimum plus profond, population de profondeur 0 pure). | 7ter.91 (2)(3)(4) · `[ETABLI]` (reproducteur `tp.ref`, binaire `tmp_refc7`, 949 films) |
| **<< La diffusion reseau explique le sur-comptage des codes 7 >>** (7ter.86 (5)(b)) | **`[REFUTE]` par sa propre prediction.** Elle predit que plusieurs codes 7 partagent le handle d objet-projectile. Mesure sur 400 films : la dedup retire **3.13 %** au code 7 mais **5.89 %** au **code 6** — qui est l impact sur la GEOMETRIE et ne peut structurellement pas se diffuser a plusieurs cibles. Et le rapport **MONTE** : `code7/(code6+code7)` passe de **0.6180 a 0.6248** au lieu de descendre vers la bande 0.24-0.39. Le multi-impact existe (1 640 handles a 2+) mais il est marginal. **Le sur-comptage vient d ailleurs, et sa cause n est pas connue.** **DIRECTION REPRODUITE** par un binaire independant sur la population de profondeur 0 : dedup **0.0217** au code 7 contre **0.0486** au code 6, rapport 0.5852 -> 0.5919 (7ter.94 (3)). | 7ter.91 (6) · `[REFUTE]` (reproducteur `tp.ref`) |
| **Mesurer la MAP-DEPENDANCE d une largeur de champ par balayage + hors echantillon, AVEC UNE ENTROPIE COMME DISCRIMINANT** | **MORTE — mais SEULEMENT dans cette version, et la nuance est vitale : voir §3.1, la piste a ete RESSUSCITEE avec un autre discriminant.** Le balayage par carte de la largeur du bloc de position du code 6, valide hors echantillon par hachage SHA-256 dans les deux sens, rend **11/12 = 0.9167** contre un fond de 1/15 — ce qui a l air d une preuve. Le controle de localisation le tue : la MEME procedure appliquee au **code 7**, dont la largeur est lue dans un `MOV R9D,0xc` et vaut **36 partout par construction**, rend **15/18 = 0.8333** avec `argmin = 42` sur 12 cartes. **Raison** : changer une largeur de SAUT ne detruit pas la lecture, elle la DEPLACE — un champ deplace peut tomber dans une zone de bits calmes et voir son entropie BAISSER sans aucun alignement. Troisieme costume du piege de 7ter.86 (3) et 7ter.88 (2). ⚠ **NE PAS LIRE CETTE LIGNE COMME << LA MAP-DEPENDANCE N EST PAS MESURABLE >>** : c est l ENTROPIE qui est morte, pas le balayage. | 7ter.91 (7) · `[REFUTE]` comme instrument · **ressuscite** 7ter.94 (6) |

### 3.1 REFUTATIONS DE REFUTATIONS — ce qui a ete RESSUSCITE

| CE QUI AVAIT ETE DECLARE MORT | CE QUI EST VRAI AUJOURD HUI | SECTIONS |
|---|---|---|
| « Le modele `type = rang/2` de `gggl` est FAUX » (deux reponses Theater le contredisaient) | **VIVANT.** La reponse Theater de 78919882 02:35 etait ERRONEE (grenade a POINTES, pas plasma) ; l utilisateur l a corrigee sur signalement du decodeur. Le modele n etait pas faux, il etait **MAL FONDE** — il est desormais etabli par l ADRESSAGE. | 7ter.47(4) **ANNULEE** par 7ter.47ter ; refonde par 7ter.48(2) |
| « L index d etat de degat n est pas le TYPE DU DEGAT RECU » | **C EST LA BONNE LECTURE.** La refutation etait invalide : la question confondait le degat qui a DETRUIT LE BARIL et celui qui a TUE LA VICTIME. La refutation a ete RETIREE, puis la lecture etablie. **Une refutation retiree a temps a sauve la bonne reponse.** | 7ter.51(2) **RETIREE** par 7ter.51bis ; etablie par 7ter.57 |
| « Les noms litteraux n existent nulle part — limite dure, 4 supports mesures, 4 echecs » | **PREMATURE.** Le 5e support (banques Wwise `.pck`) existe et le pont `sbnk -> nom` est arithmetique. Les 4 supports examines alors sont bien morts ; la conclusion generale ne l etait pas. | 7ter.45(1) leve par 7ter.48(1) / 7ter.47ter |
| « Le kill-feed du film est structurellement incomplet sur les bots » | **DEUX MOITIES.** Le chunk HIGHLIGHT est bien humain-seul (7ter.52 avait raison sur le FAIT, tort sur le MOT) ; mais l information EST dans le film (7ter.58 avait raison sur le FOND, tort sur le LIEU) : elle est dans la voie REPLICATION, deja decodee depuis 7ter.42. | 7ter.52(A3) -> 7ter.58 -> 7ter.59 / 7ter.59bis |
| « Le compte des refutations de l index d etat de degat est de QUATRE » | **TROIS** : porte/detruit (7ter.47), quel objet (7ter.51), declencheur (7ter.56). La 4e (7ter.51bis) n etait pas une refutation mais le RETRAIT d une refutation invalide. | 7ter.57 (consequence 3) |
| « Les vehicules du BTB ne produisent pas de bloqueur neuf » (top 15 vide) | **FAUX** : artefact de marche morte. Sur la passe CALIBREE, `vehicle-type-physics` et `vehicle-emp-timer` sont 2e et 4e coupables. | 7ter.52 (C5) corrige par (C7)/(D9) |
| « La map-dependance de la longueur du code 6 n est pas mesurable — mon instrument pour la mesurer est mort » | **L INSTRUMENT ETAIT MORT, LA QUESTION NON.** Ce qui etait mort, c est le **DISCRIMINANT** (une ENTROPIE : un champ deplace peut tomber dans une zone calme et voir son entropie baisser). Le meme balayage de largeur avec un discriminant **STRUCTUREL** — la chaine reprend-elle sur un en-tete d evenement valide ? — **PASSE** son controle de localisation (**26/36** sur la valeur certaine 118, contre 15/18 d une valeur fausse) et rend la map-dependance hors echantillon a **25/31** (fond 1/31), avec **3/3** sur les paires `X` / `X Heavies`. **Avant de declarer un instrument mort, demander si c est le balayage ou le discriminant qui l est.** | 7ter.91 (7) **ressuscite** par 7ter.94 (5)(6) |

---

## 3bis. LA METHODE — LES ONZE REGLES QUI ONT PRODUIT LES RESULTATS (a lire AVANT de lancer un lot)

> **A QUOI SERT CETTE SECTION, ET POURQUOI ELLE EST SEPAREE DE §4.** §4 dit ce qu il ne faut PAS
> refaire (les patrons d erreur A..F). **Celle-ci dit ce qu il FAUT faire.** Les deux se lisent
> ensemble et ne se dupliquent pas : chaque regle porte le patron dont elle est l antidote. Les
> chiffres vieilliront, la methode non — elle est ecrite le 2026-07-28 a partir de ce qui a
> reellement tranche dans les lots `mu` / `gt` / `sl` / `rt` / `pj` / `pv` / `ap.ref` / `pv.ref`,
> et chaque regle cite le lot qui l illustre.
> **Motifs de grep** : `LA METHODE` · `REGLE 1` .. `REGLE 11` · `METHODE §3bis` · `EGALITE EXACTE` ·
> `GAIN LOCALISE` · `DEGRES DE LIBERTE` · `RECHERCHE ANCREE` · `RESULTAT NEGATIF`.

**REGLE 1 — LE CRITERE EST L EGALITE EXACTE, JAMAIS UNE STATISTIQUE AGREGEE.** Une mediane, une
erreur moyenne, une correlation, un `R2` sont reproduits par n importe quel facteur d echelle. Le
discriminant est le **NOMBRE D OBSERVATIONS QUI EGALENT LA REFERENCE A L UNITE**, parce qu un taux
ne peut pas produire le bon ENTIER pour la bonne observation.
*Illustration (7ter.80 (10), antidote du PATRON E)* : le compteur de tir fermait un deficit sur la
MEDIANE (0.9700 -> 1.0000) — un simple taux constant faisait aussi bien (0.9700 -> 1.0020). Sur
l egalite exacte, l echelle rend **3** egalites, le compteur **13** ; et sur la bande mixte
l echelle **DEGRADE 89 -> 53** la ou le compteur **ameliore 89 -> 124**.
*Confirmations* : 7ter.82 (2) — le drapeau `+106` bat le taux constant **82 contre 29** (McNemar
z = 5.08) ; 7ter.83 (1) — le balayage exhaustif des statlines rend `kills` **23/23** mais
`shots_fired` **4/23** contre un fond permute de **3/23**.
**COROLLAIRE (PATRON D)** : avant d imposer ce critere, mesurer qu il est ATTEIGNABLE — 12.0 % des
joueurs ont un decodage de tirs exact (7ter.82 (2)). Un critere que rien ne peut passer n est pas
un critere.

**REGLE 2 — LE GAIN DOIT ETRE LOCALISE.** Une correction qui ameliore TOUT uniformement est un
facteur d echelle deguise. Une vraie cause ameliore **exactement la population concernee** et ne
touche pas les autres.
*Illustration (7ter.86 (4))* : `r(code7, tirs de projectile) = +0.7675` contre
`r(code7, tirs hitscan) = -0.1929` — **le signe s inverse**. Ajustement `code7 = a x P + b x H` :
`a / b = 23`. Et le controle le plus dur n est pas un taux, c est une **egalite a zero** : **39 des
65 films sans aucun tir de projectile portent EXACTEMENT zero code 7**.
*Reserve a ecrire AVEC le resultat, pas apres* : le kill-event (code 85), qui n a aucune raison
d etre un impact, penche lui aussi vers le projectile (+0.6207 contre +0.0951) — il existe donc un
**effet de composition de film**, et seul l ecart ABSOLU tranche (0.442 code-7 par tir de
projectile contre 0.086 kill-event).

**REGLE 3 — LE CONTROLE NEGATIF DOIT AVOIR DE VRAIS DEGRES DE LIBERTE, ET SA VALEUR ALTERNATIVE
S ECRIT D AVANCE.** Un controle qui ne pouvait pas echouer ne prouve rien (PATRON D).
*Illustrations* : la **lecture decalee d un bit** — un bit d ecart detruit le champ dans les DEUX
instruments (7ter.82 (1)) ; la **permutation intra-film** — le reel a **0/200** tirages aussi bons
(7ter.82 (4)) ; **983 000 lectures a position aleatoire** pour chiffrer le bruit de fond d une
constante arbitraire (7ter.75 (3), jambe 4) ; le **controle d alignement** de 7ter.86 (3), ou cinq
departs de bit concurrents sont mesures cote a cote.
**PIEGE JUMEAU (7ter.86 (3))** : un controle peut avoir des degres de liberte et rester le MAUVAIS
discriminant — la << part de marches terminees proprement >> est MAXIMALE hors alignement (0.8643
au bit 2 contre 0.5724 au bit 1), parce qu un depart faux termine proprement avec ZERO evenement.
Le discriminant etait le VOLUME lu.
**TROISIEME FORME, ET C EST CELLE QUI COUTE LE PLUS CHER (7ter.89 (2))** : le controle POSITIF qui
ne peut pas ECHOUER. *Avant de publier un controle positif, ecrire l expression algebrique de la
quantite mesuree et verifier que **LA REFERENCE NE CONTIENT PAS LE CANDIDAT**.* Si le
denominateur est construit a partir des ensembles compares, le resultat est une identite :
`|A|/|AuB| + |B|/|AuB| = 1 + |AnB|/|AuB|`, reproduit a **neuf decimales** — la << partition
mesuree >> de 7ter.88 (4) etait la definition d une union. Le test qui a de vrais degres de
liberte est la couverture d une reference **INDEPENDANTE**, avec une nulle de **MEME POOL**
(un pool de 256 valeurs collisionne a 0.36 tout seul).
**QUATRIEME OCCURRENCE, MESUREE LE 2026-07-28 (7ter.100 (2)(5)), ET ELLE DONNE LA RECETTE** : le
<< controle positif du detecteur de co-mouvement a **1.00000** >>, publie par 7ter.90 puis repris
par 7ter.97, est le meme piege — le passager synthetique y est la piste vehicule TRANSLATEE de
`dmax/2`, donc `distance <= dmax` et `ecart de deplacement = 0` sont vrais **par construction**
(`match == close == avail`, ce que la sortie imprimait telle quelle). Il rend 1.00000 jusqu au
decoupage que la section elle-meme declare absurde. **RECETTE, EN TROIS TEMPS** : (i) ecrire
l expression algebrique du positif AVANT de le publier ; (ii) le rejouer dans un regime ou il
DOIT echouer (ici : un decoupage faux, une bande de bruit) ; (iii) lui donner un degre de liberte
reel — le meme passager REQUANTIFIE sur la grille du film rend **0.955**, pas 1.000, et c est ce
chiffre-la qui mesure le detecteur. **MEME LOT, MEME FAUTE SUR UN CONTROLE D UNICITE** : << 10
bipedes apparies a UN SEUL vehicule chacun >> passe **84 % du temps** sous une attribution au
hasard (`C(10,2)/253 = 0.178`) — *avant de publier un controle, calculer ce qu il rend sous sa
propre nulle*.

**REGLE 4 — QUAND UN ACCORD EST DEJA ETABLI A UN NIVEAU, IL NE PROUVE RIEN AU NIVEAU AU-DESSUS.**
*Illustration decisive (7ter.85 (3), PATRON E troisieme costume)* : l accord par **ARME** contre
l API n etait pas une validation neuve, parce que l accord par **JOUEUR** etait deja etabli
(7ter.81 (5)) — n importe quel regroupement de joueurs, **y compris AU HASARD**, le reproduit :
**197 tirages sur 200** a purete >= 50 %. La correlation de rang (`rho` de Spearman) tombe comme
preuve : la nulle la reproduit **200/200**. Ce qui survit est le test qui **SORT du regroupement** :
le contraste **INTRA-JOUEUR** (deux armes, meme joueur, meme match) — etendue 0.3428 contre une
nulle a 0.0392, **0/200**.
**REGLE A APPLIQUER** : la nulle a battre n est pas << aucun lien >>, c est << le meme lien fin,
regroupe au hasard >>. Et cette nulle doit **publier la part d etiquettes qu elle laisse en
place** : celle du lot `pv` en figeait **80.2 %** (7ter.87 (4)), donc elle ne pouvait rien decider.

**REGLE 5 — HORS ECHANTILLON, AVEC UNE REGLE DE PARTAGE INDEPENDANTE DE L ORDRE, ET LE RESULTAT
DOIT SURVIVRE A L INVERSION DES DEUX MOITIES.** Un partage par rang lexicographique ou par ordre de
lecture depend du corpus ; un **hachage** n en depend pas.
*Illustrations* : 7ter.82 (2) — partage par bit de poids faible du **SHA-256 du `match_id`**, puis
inversion A <-> B : **86 contre 23, z = 6.27**, calage 1.0037 ; 7ter.87 (10) — biais appris sur une
moitie et applique a l autre **sans reajustement**, erreur **0.0070 dans LES DEUX SENS**.
**A ECRIRE AVEC** : une correction apprise sur l API n est plus un produit offline, **c est une
calibration** (7ter.87 (10)).

**REGLE 6 — LE DESASSEMBLAGE DIT CE QUE LE JEU COMPTE ; LE FILM DIT CE QU IL ENREGISTRE. NE JAMAIS
CONFONDRE LES DEUX.** Une chaine d appels fermee dans l executable ne prouve **aucune** presence
dans le flux.
*Illustration* : 7ter.81 (1)(3) etablit dans l `.exe` que `ShotsFired` / `ShotsLanded` sont pousses
vers la replication sous les identifiants de stat **0x0D** et **0x0E** — et 7ter.83 (1) mesure
qu ils sont **ABSENTS des statlines du film** (balayage exhaustif, **4/23** contre **3/23** au
hasard, quand `kills` sort a **23/23** par le meme balayage). 7ter.81 (0) avait ecrit la separation
AVANT de mesurer : c est ce qui a evite de publier un pont inexistant.
*Reciproque, tout aussi vraie* : le desassemblage NOMME des cibles a chercher (les codes 6 et 7) —
ne pas aller les COMPTER a coute une conclusion inversee (regle 9).

**REGLE 7 — UN NOM DE FONCTION NE PROUVE RIEN.** Verifier que le code fait ce que son nom annonce,
et qu un << lecteur >> n est pas un **stub 0-bit**.
*Illustrations* : 7ter.32 (1) — **21 des 50 archetypes** portent le stub `FUN_1408d8220`
(`return 1`, zero bit de corps), et **13 des 123 codes d evenement** sont ce meme stub : leur
porter un deserialiseur etait un travail vide, le decodeur avait deja raison pour eux.
7ter.84 (5) — les quatre deserialiseurs de l archetype 41 sont dans `filmdec/traverse.go` depuis le
**2026-06-13**, mais ecrits comme **CONSOMMATEURS DE BITS** : `consumeObjectParentState` marche la
grammaire au bit pres et **JETTE toutes les valeurs**. Un nom qui promet un etat, un corps qui n en
extrait rien.
*Bonne pratique* : 7ter.86 (2) rouvre CHAQUE fonction citee par l axe precedent et statue
`CONFORME` ligne par ligne — c est ce qui a permis de renforcer un controle de comptage en
**indexage direct** d une table de 123 entrees.

**REGLE 8 — UNE RECHERCHE ANCREE EST AVEUGLE A CE QUI N EST PAS ANCRE.** Avant d ecrire << X n est
pas la >>, **ENUMERER sans presupposer aucune valeur** (antidote du PATRON B, symptome E3).
*Illustration* : une << famille d arme inconnue >> (`f5c335df`) etait le **MA5K Avenger**, DEJA au
catalogue du depot — la recherche du nom exigeait une moitie basse fixe (`famille << 32 | 42c9679f`)
et etait donc structurellement aveugle a toute arme dont la moitie basse differe. **Le meme defaut
a ete releve TROIS fois** : 7ter.84 (1) le reproche a 7ter.81 (8), 7ter.86 (6) le trouve dans
7ter.84 lui-meme, et 7ter.87 (5) le reproduit avec un troisieme instrument (desaccord **0/24**,
non explique).

**REGLE 9 — LIRE SA PROPRE SORTIE EN ENTIER.** Un `top-N` est un choix de PRESENTATION ; il ne doit
jamais porter une affirmation d ABSENCE (PATRON F).
*Illustration* : la refutation de 7ter.84 etait **dans la sortie de 7ter.84**, sous une coupure
d affichage a **neuf lignes** — les codes 7 et 6 etaient aux rangs **11 et 14** de son propre
histogramme. Meme calcul, autre binaire : les neuf memes nombres a l unite, **puis les deux qui
manquaient** (7ter.86 (3)). *Second exemple, hors RE* : un archivage de 29 documents affichait
<< 29 D, 1 A >> — le desequilibre etait dans la sortie, et personne ne l a lu.
**REGLE A APPLIQUER** : quand une valeur ATTENDUE est nommee ailleurs, l interroger explicitement
(`m[6]`, `m[7]`) et **imprimer le zero s il y en a un**, plutot que de lire un classement tronque.

**REGLE 10 — UN RESULTAT NEGATIF BIEN MESURE EST UN LIVRABLE.** Il vaut les semaines qu il fait
economiser, et il s ecrit **avec ses chiffres** pour que personne ne recommence.
*Illustrations* : 7ter.83 — << les compteurs ne sont ni dans les statlines ni dans le registre
ECS >>, par balayage exhaustif (largeurs 4 a 32, puis 4 096 bits de charge utile) et enumeration
des **325 composants / 118 archetypes**, taux de fond permute publie a cote ; 7ter.80 (11) — pas de
compteur de MUNITIONS a portee du fire-event, **3 681 candidats**, meilleur `P(pas = -1) = 0.1305`
= le bruit ; 7ter.87 (11) — un tableau entier << MESURE COMME FAUX >>.
**FORME IMPOSEE** : un negatif se publie avec (i) le protocole exhaustif, (ii) le **controle
positif du meme instrument** (sinon on mesure sa propre cecite), (iii) le taux de fond, (iv) sa
**PORTEE** — c est faute de portee ecrite dans le titre que 7ter.83 a du recevoir un renvoi.

**REGLE 11 — UNE CORRELATION SEULE NE SEPARE PAS DEUX HYPOTHESES. CHAQUE HYPOTHESE PREDIT UNE
NULLE *ET* UNE FORTE : C EST LE COUPLE QUI COMPTE, ET SOUVENT C EST LE NIVEAU QUI DECIDE.**
*(Ajoutee le 2026-07-28, mesuree par 7ter.98 puis 7ter.101 — antidote du PATRON E, complement de la
regle 1.)*

Le cas est net et il aurait piege n importe qui. Question : un enregistrement de degat code-36
compte-t-il les **TIRS** ou les **TOUCHES** ? Sur le corpus entier, les deux correlations avec la
precision du joueur valent :

```
  r(records / tirs   , precision)  = +0.1308
  r(records / touches, precision)  = +0.0166      <- la plus PLATE
```

**La lecture naive — << celle qui est plate designe l unite >> — aurait conclu TOUCHE, et elle
aurait eu tort.** Les deux hypotheses predisent chacune une nulle ET une forte : si le record est
un tir, `records/tirs` doit etre plat et `records/touches` doit varier comme `1/p` ; si c est une
touche, l inverse. Une seule des deux colonnes ne dit rien.

**LA PARADE, ET ELLE NE COUTE RIEN : CALCULER LES BORNES ALGEBRIQUES SUR LES DONNEES ELLES-MEMES,
PAS LES SUPPOSER.** Faites sur la population a arme unique hitscan (Tactical, n = 138) :

```
  si record = c x TIRS      les deux correlations valent  ( 0.0000 , -0.7517 )
  si record = c x TOUCHES   elles valent                  (+1.0000 ,  0.0000 )
  MESURE                                                  (+0.2182 , -0.1539 )   <- NI L UNE NI L AUTRE
```

**Aucune des deux hypotheses n est compatible avec le couple observe : la correlation ne tranche
pas, et il faut le CONSTATER au lieu de choisir.** Ce qui a tranche, c est **LE NIVEAU** :
`records/tirs` **0.9300** (prediction si TIR : ~1) contre `records/touches` **3.4655**
(prediction si TIR : `1/p` = **3.601**) — les deux predictions du TIR tombent ensemble, celle de
la TOUCHE est fausse d un facteur 3.3. *Si un record etait une touche, un joueur de Tactical
aurait 93 % de precision.*

**ET LE COMPLEMENT QUI ACHEVE, C EST LA LOCALISATION** (regle 2) : par quartile de precision, quand
la precision DOUBLE, `records/tirs` reste **PLAT** (0.9305 -> 0.9457) et le taux de remplissage
`porteurs/records` la **SUIT** (0.1724 -> 0.3220). Aucun facteur d echelle ne produit une colonne
plate et une colonne qui suit **en meme temps**.

**COROLLAIRE DE PORTEE, ET IL A COUTE UNE SECTION** (7ter.101 contre 7ter.98) : ce niveau a ete
mesure sur un arsenal a **trace instantanee**, et il **s inverse** sur un arsenal a projectiles
(Fiesta : `records == shots_hit` **174** contre **134.7** au fond permute, 0/200). **Ecrire la
population avec le chiffre, toujours** — c est la faute que 7ter.98 reprochait au chantier voisin
et qu il a commise sur son propre resultat trois paragraphes plus loin.

---

## 4. LES ERREURS DE METHODE RECURRENTES, ET LEUR PATRON

> Les huit erreurs ci-dessous se rangent sous **trois patrons**. Les reconnaitre coute moins cher
> que de les refaire : chacune a coute entre une demi-journee et quatre sections de journal.
>
> **SECTION JUMELLE : §3bis — LA METHODE.** Celle-ci dit quoi NE PAS refaire, §3bis dit quoi FAIRE.
> Correspondances : PATRON D -> regle 3 · PATRON D-bis -> regle 7 (verifier, ne pas croire) ·
> PATRON E -> regles 1 et 4 · PATRON F -> regle 9 · PATRON B / symptome E3 -> regle 8 ·
> PATRON A -> regles 2 et 5 · **la correlation qui ne separe pas deux hypotheses -> regle 11**.
> **Ne pas dupliquer un contenu d une section a l autre : ajouter le renvoi.**

### PATRON A — « L INSTRUMENT PARTAGE UNE PIECE AVEC LA REPONSE » (circularite instrumentale)

| # | SYMPTOME | SECTION | REGLE A APPLIQUER |
|---|---|---|---|
| E1 | Une accuracy mesuree contre un oracle de **MEME LOGIQUE** : le « 94% per-paire » du same-clock etait valide contre un oracle live appliquant la meme regle temporelle. Confronte au Theater : 11/14 = 78.6% contre une baseline triviale de 71.4%. | 7ter.12 -> 7ter.19, 7ter.26(7) | **Tout chiffre d accuracy se compare a une BASELINE TRIVIALE** et a un oracle de logique DIFFERENTE. Sans les deux, il ne prouve rien. |
| E2 | Une **population mesuree qui NE CONTIENT PAS la cible** : 7ter.30 a 7ter.32 mesuraient sur les 26 910 paquets SANS event ; les 93 morts sont 93/93 dans les 3 508 paquets A EVENTS. Meme erreur un cran plus haut sur l oracle Rosette (que des deltas de biped SPARSE, 0 NEW, 0 masque portant i11). | 7ter.33(1), 7ter.32 | **Avant d optimiser une metrique, PROUVER que la cible est dans la population** — par un calage avec pic net et controle negatif. |
| E5 | Un **filtre de doctrine qui MASQUE le cas qu il devait attraper** : `69fd30b9` publie « PRECIS » vers `escharumhammer` (marteau Bannis de campagne) alors que `gravityhammer.pck` existe ; la ligne n est pas dans le residu douteux parce que la comparaison de chaines a matche sur le mot « hammer » et l a comptee CONCORDANTE. | 7ter.50(3)5 | **Une mesure de concordance ne peut pas servir a la fois de SCORE et de FILTRE de ce qu on montre a l utilisateur.** Deux criteres distincts, toujours. |
| E9 | Une **distribution de causes lue sur un decodeur qui meurt immediatement** : sur la passe BTB non calibree, aucun composant de vehicule dans le top 15 -> « hypothese non soutenue ». Sur la passe calibree, ils sont 2e et 4e. Le piege etait deja nomme en 7ter.30 (« le 74% designait un symptome, pas la cause ») et il s est reproduit. | 7ter.52 (D9) | **Une liste de coupables lue sur une marche morte ne mesure pas ce qui casse, elle mesure OU on meurt.** Ne jamais interpreter une distribution de causes avant d avoir fait tourner le decodeur a son regime nominal. |

### PATRON B — « CONCLURE SUR LA NATURE DE L OBJET SANS AVOIR ENUMERE »

| # | SYMPTOME | SECTION | REGLE A APPLIQUER |
|---|---|---|---|
| E10 | Une **SYMETRIE MESUREE dont une seule moitie est passee dans le code**. 7ter.59 chiffrait « kills du bot -> une MORT SANS KILL en face ; morts du bot -> un KILL SANS MORT en face, trois orphelines de CHAQUE cote ». La moitie « kill orphelin » est devenue une population (`orphK` -> morts DE bot) ; la moitie « mort orpheline » est restee une phrase. 7ter.77 a re-decouvert les memes trois morts **par l autre bout** et conclu « impubliables », sans revenir a la moitie manquante. | 7ter.59 -> 7ter.77 -> 7ter.79 | **Quand une mesure etablit une SYMETRIE, ecrire les DEUX moities dans le code ou aucune.** Une moitie exploitee et une moitie en prose, c est une piste morte qui a l air traitee. |

| # | SYMPTOME | SECTION | REGLE A APPLIQUER |
|---|---|---|---|
| E3 | **« Le format ne porte pas X » alors que la mesure ne dit que « notre lecteur ne trouve pas X ».** Deux occurrences : (a) les noms litteraux declares « limite dure, quatre supports, quatre echecs », puis trouves dans les banques Wwise ; (b) « le kill-feed du film est structurellement incomplet sur les bots », alors que le jeu les affiche. **L erreur SYMETRIQUE existe aussi** : 7ter.58 a declare « defaut de notre lecteur » ce qui etait une propriete reelle du format, et a lance la chasse a une ancre inexistante. | 7ter.45(1) / 7ter.48(1) · 7ter.52(A3) / 7ter.58 / 7ter.59 | **Avant d ecrire « le format porte X » OU « ne porte pas X », ENUMERER sans ancre.** L enumeration exhaustive des end-markers du chunk HIGHLIGHT coute 40 lignes de Go et deux minutes, et elle a tranche definitivement. |
| E8 | Une **reponse deja ecrite dans le journal, re-cherchee faute de l avoir grepe** : la question des bots a ete rouverte alors que la reponse etait en 7ter.42(7c), re-verifiee en 7ter.43(6) — **2 400 lignes plus haut**. Trois agents successifs et deux verifications adversariales sont passes a cote ; c est l utilisateur qui a dit « relis notre doc ». | 7ter.59bis | **Avant d ouvrir une piste, GREPER le journal sur le sujet.** Le RE_LOG depasse 7 100 lignes : il n est plus lisible en entier, il doit etre INTERROGE. C est la raison d etre de ce fichier-ci. |

### PATRON C — « L EVENEMENT PHYSIQUE N A PAS ETE ECRIT AVANT LA QUESTION »

| # | SYMPTOME | SECTION | REGLE A APPLIQUER |
|---|---|---|---|
| E7 | **Chercher dans le champ d une VICTIME une information qui appartient au GESTE d un TUEUR.** Trois refutations successives de l index d etat de degat, toutes avec le meme vice : « porte / detruit », « quel objet », « le declencheur ». Le declencheur n est pas une propriete de la SOURCE, c est une propriete de l ACTION. | 7ter.47(5), 7ter.51(1), 7ter.56 | **REGLE-MERE (nee de 7ter.51bis) : AVANT de poser une question, ecrire QUEL EVENEMENT PHYSIQUE le champ decode est cense decrire, et verifier que la question porte sur CE meme evenement.** Une chaine causale (arme -> objet -> victime) a plusieurs maillons ; se tromper de maillon suffit a detruire l information. |
| E6 | Une **condition de falsification mal ecrite qui aurait declare FAUSSE une lecture juste** : P1 disait « FALSIFIEE SI le tueur est a pied avec une arme tenue en main » ; la reponse fut une **tourelle UNSC arrachee et portee a la main** — une classification correcte aurait ete declaree fausse. La verification adversariale l avait reecrite a temps. | 7ter.46(8), sauvetage constate 7ter.47(3) | **Enumerer les cas limites AVANT de poser la question.** Toute prediction doit ecrire trois blocs : ce qui FALSIFIE, ce qui NE FALSIFIE PAS, ce qui est NON DECIDABLE (a consigner comme tel, surtout pas comme une confirmation). |
| E4 | Une **conclusion sur un tag publiee SANS recouper la verite terrain deja au journal** : 7ter.52(B5) conclut que les tags Kong etablissent l axe LANCE/DETRUIT, alors que 7ter.44 consignait deja « il portait un baril, j ai TIRE dessus » sur l un de ces deux tags. | 7ter.52bis | **Avant de publier une conclusion sur un tag, GREPER ce tag dans le RE_LOG** et verifier qu aucune verite terrain ne la contredit. (Meme famille que E8 : un signal fort interprete sans confronter aux mesures existantes.) |

### META-PATRON — « CHERCHER LE SENS DANS UN INDEX »

Quatre lectures successives de la MEME donnee ont echoue parce qu elles cherchaient une semantique
dans un **ORDRE** (rang dans une liste de dependances, index d entree de tableau). Un ordre de
deduplication ne porte aucun sens. **Ce qui en porte, c est l ADRESSAGE** : la table `tagref` dit,
pour chaque dependance, le block et l OFFSET DE CHAMP ou le jeu va la lire — et les deux modeles
refutes se sont re-derives immediatement la-dessus. (7ter.48(0))

### META-PATRON 2 — « DEUX REFERENTIELS QUI DECRIVENT LE MEME OBJET, JAMAIS MIS EN REGARD »

**LA LIGNE D INDEX QUI AURAIT EVITE SIX SEMAINES.** La capture Cheat Engine du **2026-06-10**
nommait des **DECALAGES DE STRUCTURE** (`KillerPercentageDamageDone` a `+0x228`,
`AssistantPercentageDamageDone` a `+0x22c`). La grammaire du kill-event, elle, parlait de
**POSITIONS DE FIL** (le 1er et le 2e bloc `R32` apres les references d entite, publies comme
« inexpliques »). **Les deux descriptions portaient sur LA MEME PAIRE DE NOMBRES**, et personne n a
superpose les deux referentiels. Ce n etait pas une donnee manquante : c etait un CHANGEMENT DE
REPERE jamais ecrit. (7ter.75 (7))

**REGLE A APPLIQUER** : quand une capture runtime et une grammaire de fil decrivent le meme objet,
ECRIRE LE CHANGEMENT DE REPERE (offset de structure <-> position de bit) AVANT de conclure que l un
des deux ne dit rien.
### PATRON D — « UN CONTROLE SANS AUCUN DEGRE DE LIBERTE » (2026-07-27, 7ter.81 (10))

Deux lots successifs ont publie comme **PREUVES** deux controles **qui ne pouvaient rien rendre
d autre** : une « fermeture arithmetique a zero parametre libre » dont le quotient n est meme pas
entier (`0x694 / 0xA8 = 10.024`), et une multiplicite par arme dont la valeur **1.000 est imposee
par le format du paquet** (un enregistrement est le 1er event de SON paquet, et un paquet porte UN
horodatage). Les deux ont ete acceptes parce qu ils **pointaient dans le sens du resultat**.

**REGLE A APPLIQUER : avant de citer un controle, ECRIRE CE QU IL AURAIT RENDU SI L HYPOTHESE
ETAIT FAUSSE. S il n existe pas de valeur alternative, ce n est pas un controle, c est une
reformulation.** (Famille du PATRON A, mais l instrument ne partage pas une piece avec la reponse :
il n a simplement aucun degre de liberte.)

**LE PATRON D A UN JUMEAU SYMETRIQUE, ET IL EST PLUS SOURNOIS : LE CONTROLE POSITIF QUI NE PEUT
PAS REUSSIR** (2026-07-28, 7ter.88 (5)). Le lot `pj.own` a construit un controle positif —
<< le code 6 nomme un projectile, ses handles doivent donc se retrouver dans ceux du code 7 >> —
et l a balaye sur les 9 lignes de configuration : **zero partout**. Il a d abord conclu que
l instrument etait aveugle, ce qui rendait son propre negatif inexploitable. **Le controle etait
faux** : `FUN_142eed4e8` emet le type 6 **OU** le type 7, jamais les deux, donc les deux
ensembles sont **disjoints par construction** et le controle ne pouvait rendre que zero. Reecrit
dans le bon sens — la **COUVERTURE** de l ensemble des projectiles, denominateur inverse — il
rend `0.5408 + 0.4606 = 1.0014` et passe.
**REGLE A APPLIQUER : un controle POSITIF s ecrit avec la meme exigence qu un controle negatif —
ce qu il rendrait si l hypothese etait VRAIE, et pourquoi il pourrait le rendre.** Et, dans une
comparaison d ensembles, **ecrire lequel des deux est le DENOMINATEUR de la question posee** :
sous le mauvais denominateur, le meilleur candidat du lot sortait a 0.3643 ; sous le bon, a
0.0189.

**ET LE REMPLACANT ETAIT UN PATRON D DANS L AUTRE SENS — TROISIEME OCCURRENCE, 2026-07-28,
7ter.89 (2).** Le controle de remplacement de `pj.own` — la **COUVERTURE** — a pour ensemble de
reference l **UNION** des deux ensembles qu il compare (`projSet()` dans `tmp_pjown/link.go`).
Donc, avec A = code 7 empl. 1 et B = code 6 empl. 0 :

```
  |A| / |A u B|  +  |B| / |A u B|  =  1 + |A n B| / |A u B|
```

La somme publiee, `0.5408 + 0.4606 = 1.0014` avec << 0.14 % de recouvrement >>, est **cette
identite** — vraie quel que soit le corpus, quelle que soit la grammaire, et **meme si aucun des
deux codes ne nommait un projectile**. Reproduite a **neuf decimales sur deux corpus** (150 films :
ecart a 1 = 0.000034 = le recouvrement ; 25 films : 0.000000 = 0.000000). Le negatif tenait, mais
pas pour la raison ecrite : ce qui le porte est la **couverture d une reference INDEPENDANTE**
(le candidat n en fait pas partie) **avec une nulle de MEME POOL** — 0.5046 contre 0.3600.

**REGLE A APPLIQUER, EN UNE QUESTION : avant de publier un controle positif, ECRIRE L EXPRESSION
ALGEBRIQUE de la quantite mesuree et chercher si LA REFERENCE CONTIENT LE CANDIDAT.** Si le
denominateur est construit a partir des ensembles compares, la reponse est une identite. Trois
lignes de calcul. Ce que cela aurait economise : la phrase qui rendait un negatif exploitable.

**QUATRIEME OCCURRENCE, 2026-07-28, 7ter.100 (2) — ET C EST LE MOTIF LE PLUS RECURRENT DE LA
JOURNEE : UN CONTROLE POSITIF QUI NE PEUT PAS ECHOUER NE PROUVE RIEN.** Le detecteur de
co-mouvement vehicule-bipede (7ter.90, puis 7ter.97) publie un controle positif a **1.00000** —
un << passager synthetique >> retrouve a chaque fois. Ce passager **EST la piste du vehicule
translatee de `dmax/2`** :

```
  dist = dmax/2 <= dmax        TOUJOURS
  e    = 0      <= mmax * lv   TOUJOURS
  => match == close == avail   PAR CONSTRUCTION   (25 666/25 666 · 2 848/2 848 · 18 644/18 644)
```

**Preuve empirique de l identite : il rend 1.00000 AUSSI au decoupage `(14,15)`, que le lot
lui-meme declare absurde.** Le positif qui PEUT echouer — un passager **requantifie sur 13 bits**,
donc porteur d une erreur reelle — vaut **0.955**, et c est le seul chiffre citable.

**LES QUATRE OCCURRENCES, POUR QU ON RECONNAISSE LA FORME** :

```
  1. fermeture arithmetique a zero parametre libre      7ter.81 (10)  quotient meme pas entier
  2. controle positif qui ne peut pas REUSSIR           7ter.88 (5)   ensembles disjoints par
                                                                      construction -> zero partout
  3. couverture dont la reference EST l union           7ter.89 (2)   somme = 1 + recouvrement,
                                                                      reproduite a 9 decimales
  4. passager synthetique translate de dmax/2           7ter.100 (2)  1.00000, y compris au
                                                                      decoupage declare absurde
```

**Deux compagnons du meme lot, qui ne sont pas des identites mais n ont pas plus de puissance** :
le controle d **UNICITE** (`P(zero collision) = 0.84` a 10 couples sur 253 pistes — il passe
presque toujours) et le controle d **ALIGNEMENT**, qui est **structurellement asymetrique**
(`sh = 52-(w0+w1+w2)` : `w0+1` n ajoute qu un bit de poids faible, `w0-1` promeut un bit d un axe
au rang de poids fort d un autre — il ne peut echouer que d un cote, et `(16,16)`, non teste,
rend 50 contre 0/0/0).

**REGLE A APPLIQUER, ET ELLE TIENT EN UNE PHRASE : ECRIRE LA VALEUR QUE LE CONTROLE POSITIF
RENDRAIT SI L HYPOTHESE ETAIT FAUSSE, ET LA MESURER — pas seulement l imaginer.** Le test le moins
cher, et il a suffi les quatre fois : **rejouer le controle positif dans une configuration que le
lot lui-meme declare absurde.** S il passe encore, ce n est pas un controle.

### PATRON D-bis — « J AI JUGE LE DECODAGE CONTRE UNE CROYANCE SUR LE JEU, AU LIEU DE LA MESURER » (2026-07-28, 7ter.96 (3), corrige par 7ter.102)

Le lot `vl` a publie que la racine de banque Wwise ne nomme un chassis << UNIQUE et PLAUSIBLE >>
que **6 fois sur 14**, et en a tire que la banque est reutilisee entre chassis. Deux tags etaient
comptes comme faux au motif que *<< ni le Pelican ni le Falcon ne sont pilotables en multijoueur
Halo Infinite >>*. **Cette phrase n est pas une mesure : c est une croyance sur le jeu, produite
sans instrument** — et elle est fausse. **Le Falcon existe en BTB**, et la mesure avait ete faite
sur `4f77afc1`, **un film BTB** : trouver le tag la etait exactement ce qu il fallait attendre.
Le compte remonte a **8/14 (57 %)**, et a **53/89 (59.6 %)** sur le catalogue entier.

C est une variante du PATRON A ou **la piece partagee entre l instrument et la reponse est
l opinion de l analyste** — plus difficile a voir que les autres, parce qu elle ne figure dans
aucune ligne de code et qu elle se presente comme du bon sens.

**REGLE A APPLIQUER : NE JAMAIS DECLARER UN DECODAGE IMPLAUSIBLE SUR UNE CROYANCE CONCERNANT LE
JEU.** La croyance se verifie comme le reste — par le catalogue embarque, par le mode et la carte
du film, par une confrontation Theater. **Un jugement non instrumente n entre pas dans un COMPTE
publie** ; au mieux il se note a cote, avec son statut.
**ET SA SYMETRIQUE, PARCE QU ELLE EST AUSSI FACILE A RATER** : quand la croyance tombe, elle
**retire un argument CONTRE — elle n ajoute pas une preuve POUR**. Que le Falcon existe ne prouve
pas que le tag le designe ; la confrontation Theater reste due.

### PATRON E — « LA STATISTIQUE AGREGEE QUE N IMPORTE QUEL FACTEUR D ECHELLE REPRODUIT » (7ter.80 (10))

Un lot a failli publier la fermeture d un deficit de decodage sur une **MEDIANE** (0.9700 -> 1.0000).
Un simple **taux constant** appris ailleurs faisait aussi bien (0.9700 -> 1.0020). La mediane ne
prouvait donc rien. Le discriminant est le **nombre de sujets dont le decodage egale la reference
A L UNITE** : un taux ne peut pas produire le bon ENTIER pour le bon joueur (echelle 3 contre
compteur 13 sur la bande a forte proportion d automatique ; l echelle **DEGRADE** 89 -> 53 la ou le
compteur ameliore 89 -> 124).

**REGLE A APPLIQUER : quand un correctif est propose pour fermer un ecart AGREGE, mesurer d abord
ce que ferait un simple facteur d echelle sur la meme population. S il fait aussi bien, la mesure
n a rien etabli.** A faire AVANT de conclure, pas apres.

**TROISIEME COSTUME DU MEME PATRON — « LE RESULTAT FIN QUI REND VRAI LE RESULTAT GROSSIER »
(7ter.85 (3))** : il n y avait cette fois aucun facteur d echelle. Le lot `pv` validait la
precision PAR ARME en comparant, arme par arme, le taux du film a la precision API des joueurs de
cette arme — et l accord etait bon. **Mais 7ter.81 (5) avait deja etabli l accord JOUEUR PAR
JOUEUR** ; des lors **tout regroupement de joueurs, y compris AU HASARD, fait coincider les deux
taux du groupe**. Mesure : la nulle qui regroupe les memes joueurs sous des etiquettes d arme
tirees au sort fait aussi bien (**8/200** a purete >= 80 %, **197/200** a >= 50 %).
**REGLE : quand on valide une quantite AGREGEE alors qu une validation FINE existe deja sur les
memes objets, la nulle a battre n est pas << aucun lien >> — c est << le meme lien fin, regroupe au
hasard >>.** Le seul test qui a echappe au piege est celui qui SORT du regroupement (contraste
INTRA-joueur, 7ter.85 (4)).

### PATRON F — « LA REPONSE ETAIT DANS MA SORTIE, SOUS LA COUPURE D AFFICHAGE » (2026-07-28, 7ter.86 (3))

Le lot `pj` a titre << la touche de projectile n est dans AUCUN flux d events >> apres avoir
enumere, SANS ANCRE, les codes du premier evenement de chaque paquet. **Son enumeration contenait
la reponse** : les codes 7 et 6, aux rangs **11 et 14** — et son mode d affichage en imprimait
**neuf**. Le meme calcul, refait a l identique par un autre binaire, rend les neuf memes nombres a
l unite, puis les deux qui manquaient.

Deux causes se sont additionnees, et aucune n est une erreur de mesure :

1. **UN `topMap(m, 9)` A DECIDE D UNE CONCLUSION.** Un `top-N` est un choix de PRESENTATION ; il ne
   doit jamais porter une affirmation d ABSENCE. **REGLE : une conclusion negative (<< X n est pas
   dans le flux >>) ne se tire jamais d une sortie tronquee. On interroge la structure sur la
   valeur ATTENDUE — ici `m[6]` et `m[7]` — et on imprime le zero s il y en a un.**
2. **LA PREDICTION NOMMEE N A PAS ETE CONFRONTEE.** Un axe de desassemblage avait deja designe les
   codes **6 et 7**, nommement, dans le meme chantier. **REGLE : quand un autre axe fournit un
   NOMBRE precis a chercher, on le cherche EXPLICITEMENT avant de conclure a l absence — c est
   deux lignes de code, et c est la difference entre un lot juste et un lot inverse.**

**COROLLAIRE SUR LES CONTROLES D ALIGNEMENT** (meme section) : la << part de marches terminees
proprement >> n est PAS un discriminant d alignement, elle est MAXIMALE hors alignement (0.8643 au
bit 2 contre 0.5724 au bit 1) parce qu un depart faux termine avec ZERO evenement. Le discriminant
est le **volume lu**. C est ce qui a fait lire a 7ter.84 (9) une concentration << plus forte hors
alignement >> comme un echec du modele.

### PROTOCOLE DE CORRECTION DE LA REFERENCE — les trois conditions, non negociables

Le decodeur a corrige la verite terrain **une fois** (7ter.47ter). Ce n est acceptable QUE si :
1. la correction vient d un support **INDEPENDANT** de tout ce qui produit l etiquette ;
2. ce support etait **deja concordant** sur d autres points AVANT que la contradiction ne surgisse ;
3. la contradiction est **SOULEVEE PAR NOUS** et posee comme question ouverte, jamais tranchee seul.

Sinon, **c est le decodeur qui a tort**. Si la verite terrain cesse de pouvoir falsifier, plus rien ne le peut.

---

## 5. LES CHIFFRES DE REFERENCE — LIGNE DE BASE DU 2026-07-26 (etat courant : §10 et §12)

### 5.1 Gates par film — configuration par defaut, ZERO variable d environnement

```
                                 000d5950     9b191a7f     78919882     fccc61cd    | 4f77afc1 BTB
mode                             Fiesta       standard     Forge        Fiesta      | BTB:CTF
carte                            Cliffhanger  -            High Ground  Launch Site | Flood Gulch
morts au kill-feed                    93           87           99           96     |    300
couples reconstruits                  93           85           99           95     |    294
(a) dead-states bipedes credibles     89           77           91           85     |    191    ECHEC partout
(b) couples justes             86/89 96.6%  73/77 94.8%  89/91 97.8%  82/85 96.5%   | 177/191 92.7%
    2e bijection / mediane          52 / 2       51 / 1       56 / 1       60 / 1   | marge 0 (!)
couverture (morts retrouvees)  86/93 92.5%  73/85 85.9%  89/99 89.9%  82/95 86.3%   | 177/294 60.2%
diagnostic mort par mort       91/93 97.8%  77/87 88.5%  91/99 91.9%  86/96 89.6%   | 229/300 76.3%
paquets a events localises          97.6%        96.7%        95.8%        96.8%    |   89.3%
calibration AUTO axisW/indexW        14/1         17/2         16/2         17/1    |   16/2 (apres correctif)
recordStateParam                        3            4            2            4    |      4

CUMUL 4 films   couverture     330 / 372 = 88.7%     couples justes  330 / 342 = 96.5%
PLANCHER de sur-ajustement (horloge cassee, hors tolerance) :  9.6% a 8 joueurs  |  7.9% en BTB
```
> ⚠ **CE BLOC EST LA MESURE DU 2026-07-26 MATIN. IL N EST PLUS L ETAT COURANT — CORRIGE LE
> 2026-07-28.** Les chiffres de reference d aujourd hui sont au **§10** (table hybride de 7ter.72,
> verifiee 7ter.73) : **371/371 couples REELS = 100.0 %**, **380/380 morts de l API = 100.0 %**
> (§12, 7ter.79), gate (b) ventile **98.2 % pour la marche** et **78.4 % pour le rattrapage du
> scan**. Le << CUMUL 330 / 372 = 88.7 % >> ci-dessus et le << gate (a) n est franchi sur aucun
> film >> ne decrivent plus le decodeur : le SCAN DIRECT (7ter.60) rend le gate (a) **FACULTATIF**
> pour la question de l arme, puis 7ter.63 / 7ter.66 / 7ter.70 / 7ter.79 ont ferme la couverture.
> **Ce bloc est conserve comme LIGNE DE BASE** — c est lui qui donne l amplitude de la percee, et
> les colonnes << carte >>, << recordStateParam >> et << 2e bijection / mediane >> n existent
> nulle part ailleurs.
>
> *(L annonce << le decodage de la source et son nommage sont etablis ; la COUVERTURE ne l est
> pas >> etait juste au 2026-07-26. Elle est fausse aujourd hui sur les quatre films a 8 joueurs ;
> elle reste vraie pour le BTB — §2.10.)*
Variante mesuree, ACTIVEE depuis 7ter.60 AXE A : plage de credibilite derivee des keyframes.

### 5.2 Bilan de verite terrain — 34 confrontations (etait 25 au 2026-07-26)

```
4 ancres (7ter.33bis) + 6 predictions scellees (7ter.35 x5, 7ter.37bis x1)
+ 5 lignes douteuses (7ter.44) + 5 questions de classe (7ter.47)
+ 1 correction du decodeur SUR la reference (7ter.47ter) + 2 objets (7ter.51)
+ 1 declencheur (7ter.56) + 1 type d energie (7ter.57)                          = 25
+ 5 morts auto-infligees (7ter.63) + 4 dernieres (7ter.66)                      = 34

etiquettes d arme PUBLIEES et FAUSSES : 0
abstentions justes : 2      ambiguites levees : 2
reponses de REFERENCE corrigees par le decodeur : 3 — (1) une reponse de l UTILISATEUR
   (grenade a pointes lue << plasma >>, 7ter.47ter) · (2) le KILL-FEED DU JEU (8 morts
   auto-infligees, confirmees 8/8, 7ter.63) · (3) NOTRE PROPRE APPARIEMENT (un couple fabrique
   de toutes pieces, 7ter.66). Chaque fois sous les 3 conditions du §4.
predictions d INTERPRETATION fausses : 3 (sous-type de grenade, 2 lectures de l index d etat)
```
**MAINTENIR LA DISTINCTION** : ce que le decodeur PUBLIE n a jamais ete infirme. Ce qui a ete infirme,
ce sont des MODELES DE SECOND NIVEAU poses par-dessus des etiquettes justes.

### 5.3 Espace des tags

```
tags distincts observes, 4 films            59   |   dont entrees `jpt!` : 59 / 59, zero exception
                                                 |   22 inedites sur les 2 films hors echantillon
60e valeur (`d7df0000`)                     dechet de decodage connu, absent des `jpt!`, DISPARU depuis rsp=4
etiquetes                                   59 / 60
faux positif du test `jpt!`                 ~1 / 9 000 000 (>= 0x10000)  |  1.15% sous 0x10000
inventaire du jeu                           468 `jpt!`  |  194 `weap`  |  67 `weap` nommes  |  catalogue 32/35
controle negatif sur les 468 `jpt!`         ?? 206 | ARME 114 | VEHICULE 89 | OBJET 19 | GRENADE 17 | MELEE 14 | DEGAT GLOBAL 9
tirage au hasard de 60 `jpt!` (200 tirages) 33.5 etiquetes en moyenne (max 43) contre 59/60 reels
```

### 5.4 Pont sonore

```
1339 `.pck`, 0 collision de hachage FNV-1  |  2107 `sbnk` distincts, 1953 a BKHD (154 sans = 7.3%)
1156 resolus = 59.2% des BKHD              |  controle catalogue 14/35, baseline 0/35, permutation 1/35
regle REELLEMENT publiee (96 tags)         25 / 15 / 56 = 62.5%, dont 6 concordances par un simple mot
                                           de classe -> 19 sur nom propre
validite PAR CLASSE  GRENADE VALIDE · VEHICULE VALIDE · ARME VALIDE SOUS RESERVE (cas 69fd30b9)
                     OBJET EXPLOSIF : **VALIDE depuis 7ter.57** (la reserve de 7ter.51(4) est LEVEE)
```

### 5.5 Baselines triviales — a poser AVANT de compter les points

```
9b191a7f, 14 kills de JGtm    « toujours Fuel Rod SPNKr »   10/14 = 71.4%   (7/10 = 70% sur les atteints)
78919882, table entiere       « toujours M41 SPNKr »        13/89 = 14.6%
fccc61cd, table entiere       « toujours Needler »          12/82 = 14.6%
attribution par medaille      « toujours le tag le plus frequent »  11.4% (8 joueurs) / 20.6% (BTB)
```

### 5.6 Corroborations independantes par l API (aucune n entre dans le decodage)

```
kills == deaths           93/93 · 90/90 · 99/99 · 98/98      BTB : 303/304 (1 suicide), 36 participants dont 8 bots
bots                      bid(39.0) 3k/3m sur 9b191a7f  |  bid(7.0) 1k/2m sur fccc61cd  -> indice 8 du dead-state, exact
medailles                 151 events `medal` dans le film (44/20/34/53), 32/32 par TYPE, 151/151 par OCCURRENCE
```
### 5.7 LA PRECISION DE REFERENCE (`shots_hit / shots_fired` de l API) — 3 129 joueurs, mode standard

```
q10 0.321  ·  q25 0.382  ·  MEDIANE 0.446  ·  q75 0.501  ·  q90 0.547
moyenne 0.4394  ·  ecart absolu moyen 0.0707, soit 16.1 % en RELATIF
```

**C est la baseline que tout compte de touches doit battre**, et c est aussi une CORRECTION : la
bande **27-45 %**, citee par un lot precedent comme une invraisemblance, **est l ordre de grandeur
NORMAL** de la precision reelle. Ce qui etait faux dans `HitLikely`, c est le **75-79 %**, pas le
27-45 %. (7ter.80 (7))

---

## 6. LES MOTIFS DE RECHERCHE

> Fichier a greper : `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md` (15 814 lignes au 2026-07-28, **en croissance**).

| SUJET | MOTIF |
|---|---|
| Plan du fichier / toutes les sections | `^### 7ter` — **posees au 2026-07-28 : jusqu a `7ter.102`** (`.89` tv.ref · `.90` vk · `.91` c7b · `.92` co.pi · `.93` co.re · `.94` tp.ref · `.95` co.ref · `.96` vl · `.97` vc · `.98` rc.unite · `.99` rc.perm · `.100` vc.ref · `.101` rc.ref · **`.102` vf**), plusieurs **RESERVEES** avec un en-tete pose avant mesure. **NE JAMAIS FAIRE CONFIANCE AU NUMERO D UN BRIEF** : trois lots se sont vu annoncer `7ter.89` le meme jour. Greper `^### 7ter`, prendre le premier REELLEMENT libre, puis **poser l en-tete AVANT de mesurer** (§1 regle 10). |
| **LE TIREUR N EST PAS DANS LE FLUX D EVENTS — et ce que les emplacements de PRESENCE portent** | `pj.own` · `tmp_pjown` · `emplacement de presence` · `handle d entite` · `FUN_1406d3140` · `+0x114` · `COUVERTURE` · `0.5408` · `1.0014` · `0.9805` · `ligne SPECIALE 4` · `taux de collision` -> 7ter.88 |
| **TIREUR — verification adversariale, et le controle positif qui etait une IDENTITE** | `tv.ref` · `tmp_tvref` · `IDENTITE ALGEBRIQUE` · `LA REFERENCE CONTIENT LE CANDIDAT` · `reference INDEPENDANTE` · `MEME POOL` · `0.5046` · `0.3600` · `rapport 1.40` · `neuf decimales` · `+35` · `69/150` · `mediane a 9` · `0.8422` -> 7ter.89 (1)(2)(3)(4)(5) |
| **VEHICULE (entite repliquee du film, PAS le nommage d arme)** | `VEHICULES — ETAT ET PISTES` -> **§18 de ce fichier, la page du sujet** · puis `ti=40` · `archetype 40` · `bande vehicule` · `[768,1023]` · `[768..1023]` · `vcen` · `vpos` · `bande VIDE` · `SATURATION` · `dynamic-precision` · `vehicle-seats-override` · `tmp_vehcensus` · `tmp_vehparent` · `lien joueur-vehicule` · `9 observations` · `controle decale d un bit` -> 7ter.89 (6)(7)(8)(9) et §2.5bis. ⚠ **Ne PAS greper `vcdd` / `uwfa` / `R-VEHICULE` pour ce sujet** : ceux-la sont le NOMMAGE d une arme de vehicule dans les `.module` (7ter.45/46/47), un tout autre chantier. |
| **LES DEUX INDICES DE JOUEUR DU PIPELINE (le CONSOMMATEUR, pas le decodeur)** | `co.pi` · `cmd/copi` · `getXuidToPI` · `resolveFilmPlayerIndices` · `piUnresolved` · `ordre de la base` · `22.268` · `22.658` · `76.384` · `116 films sur 116` · `nulle a zero` · `CommonWeaponSuffix` · `0.0799` · `1 512` -> 7ter.92 et **§20 de ce fichier** |
| **INDICE DU TUEUR — VERIFICATION ADVERSARIALE, et le test de LOCALISATION qui manquait** | `co.ref` · `cmd/tmp_coref` · `TEST DE LOCALISATION` · `dbPI == filmPI` · `1 736 = 1 736` · `22.820` · `21.304` · `77.040` · `237 / 239` · `z = 92.788` · `SANS ORACLE` · `11.769` · `11.137` · `IDENTITE ALGEBRIQUE` · `consumeWeaponStateAmmo` · `UBIGINT` -> 7ter.95 et **§20 / §20.3 de ce fichier** |
| **LES PATRONS D ERREUR — ce qu il ne faut PAS refaire (jumelle de §3bis)** | `PATRON A` · `PATRON B` · `PATRON C` · **`PATRON D`** (controle sans degre de liberte — **quatre occurrences**, motifs `IDENTITE ALGEBRIQUE` · `dmax/2` · `0.955` · `decoupage declare absurde`) · **`PATRON D-bis`** (juger sur une croyance sur le jeu — motifs `le falcon existe en btb` · `croyance` · `retire un ARGUMENT CONTRE`) · `PATRON E` · `PATRON F` · `META-PATRON` -> **ce fichier, §4** |
| **LA METHODE — les onze regles a lire AVANT un lot** | `LA METHODE` · `METHODE §3bis` · `REGLE 1` .. `REGLE 11` · `UNE CORRELATION SEULE NE SEPARE PAS` · `BORNES ALGEBRIQUES` · `EGALITE EXACTE` · `GAIN LOCALISE` · `DEGRES DE LIBERTE` · `RECHERCHE ANCREE` · `RESULTAT NEGATIF` -> **ce fichier, §3bis** (jumelle de §4) |
| **`precision` : DEUX SENS, ne jamais greper le mot nu** | precision de **TIR** : `precision de reference` · `0.446` · `arme dominante` · `purete` · `partBit` -> §5.7, §14, §15, §17 · precision de **QUANTIFICATION** : `axisW` · `calib2` · `cv_autocalib` -> 7ter.54 AXE3 (cf. §6.1 piege 5) |
| **TITRES DE SECTION REFUTES OU CORRIGES (a lire avant le corps)** | `RENVOI EN TETE` dans le RE_LOG -> 7ter.80, 7ter.81, 7ter.83, **7ter.84 (titre REFUTE)**, 7ter.85 |
| **PRECISION PAR ARME : ce qui est publiable et ce qui ne l est pas** | `arme dominante` · `purete` · `tmp_pvarme` · `film(dom)` · `contraste intra-joueur` · `P0 / P1 / P2` · `INVERSION` · `0.4485 contre 0.3701` -> 7ter.85 |
| **LA VENTILATION PAR ARME DEGRADE — verification adversariale de 7ter.85** | `tmp_pvref` · `film(tout)` · `gain de la VENTILATION` · `etiquettes FIGEES` · `80.2 %` · `nulle P2 globale` · `175/200` · `contamination de la REFERENCE` · `ordonnee a purete=1` · `1303 / 1303` · `MA5K Avenger` -> 7ter.87 |
| **OU SONT LES TOUCHES DE PROJECTILE (elimination sans ancre + localisation)** | `archetype 41` · `projectile-at-rest-state` · `projectile-tether-state` · `object-parent-state-component` · `moitie basse` · `48 variantes` · `tmp_pjhit` · `33.0 s` · `1 002.7 unites` -> 7ter.84 |
| **LES CODES 6 ET 7 = L IMPACT DE PROJECTILE — LA REPONSE, et elle REFUTE le titre de 7ter.84** | `code 6` · `code 7` · `FUN_142eed4e8` · `FUN_142f1c44c` · `FUN_1410f03b4` · `FUN_142f1c6cc` · `evFixed` · `tmp_apref` · `marche complete` · `0.9831` · `+0.7675` · `zero tir de projectile` · `MA5K Avenger` -> 7ter.86 |
| **LES COMPTEURS AGREGES de tirs et de touches ne sont PAS dans le film** (resultat negatif, portee stricte) | ⚠ **NE PAS LIRE << le film n enregistre ni les tirs ni les touches >>** — le titre de 7ter.83 est plus large que sa mesure, et un renvoi le dit en tete de section (2026-07-28). Ce qui est mesure : l absence des **COMPTEURS** (a) dans les **STATLINES** du film et (b) dans le **registre ECS** `chunk_00`. Le film porte bien, par ailleurs : un **compteur de TIR de 7 bits a `eventStart+22`** (7ter.80), un **drapeau de touche a `+106`** pour les armes a trace instantanee (7ter.80/7ter.82), et les **impacts de projectile codes 6 et 7**, 80 886 + 129 390 sur 949 films (7ter.86). Motifs : `statline` · `taux de fond` · `4 / 23` · `23 / 23` · `sl.read` · `sl.ref` · `tmp_slread` · `tmp_slref` · `dernier keyframe EST l etat final` -> 7ter.83 (1) |
| **Enumeration du registre ECS (325 composants, 118 archetypes)** | `325 noms` · `118 archetypes` · `ENUMERATION SANS ANCRE` · `surchauffe` · `inventaire de chargeurs` -> 7ter.83 (2) |
| **Score / kills / morts lisibles A L UNITE dans les statlines** | `172` + `126` + `208` · `OFFSET` · `LARGEUR` · `decodage offline GRATUIT` · `TEMOIN D ANCRAGE` -> 7ter.83 (5) |
| **Compteur de TIR du fire-event / tirs perdus des automatiques** | `eventStart+22` · `FireCounter` · `pas=+1` · `0.9738` · `R3` · `facteur d echelle` -> 7ter.80 |
| **LES DEUX BRANCHES NON DECODEES, FERMEES : marqueurs rejetes + jauge de charge** | `co.re` · `tmp_co_shot` · `tmp_co_gauge` · `enregistrement compact` · `bande R3` · `120-250 ms` · `5.839 contre 6.368` · `appariement a effectif egal` · `offset degenere` · `profil de bits` · `0.0039 contre 0.2602` · `largeur 22` · `4 lectures par joueur et par match` -> **7ter.93**. ⚠ **NE PAS CITER `3 537 / 3 537` COMME UN CONTROLE** : c est une identite algebrique, refute par 7ter.95 (5)(c) — renvoi en tete de 7ter.93. |
| **Compte de TOUCHES / precision** | `+106` · `HitLikely` · `drapeau de touche` · `precision de reference` · `0.446` -> 7ter.80 (7)(8)(9), 7ter.81 (5) |
| **Reproduction du drapeau / egalite exacte / McNemar / portee Fiesta** | `rt.rep` · `tmp_rtrep` · `EGALITE EXACTE` · `McNemar` · `SHA-256` · `Mangler` · `Disruptor` -> 7ter.82 |
| **Ce que le JEU compte (regles lues dans l .exe)** | `ShotsFired` · `ShotsLanded` · `RoundsCorrected` · `FUN_1408df45c` · `FUN_1408df6a4` · `objet-projectile` -> 7ter.81 (1) |
| **Pont vers la replication (stats 0x0D / 0x0E / 0x0F)** | `0x0D` · `statline` · `FUN_142bad97c` · `FUN_142bb2d80` · `0x1DF0` -> 7ter.81 (3) |
| **Flux de degat par coup (code 36 / marqueur 0xd2)** | `code-36` · `porteurs` · `ancrage` · `fantome` · `roster <= 16` -> 7ter.81 (5)(7)(8) |
| **LE RECORD ET LE PORTEUR — les deux objets a nommer avant de lire un chiffre** | `record` · `porteur` · `tableau A` · `0xD2` · `0xD3` · `type 105` · `BRUTS` · `INDEXES` · `1 799 630` · `643 233` -> 7ter.98 et **7ter.101**, index §21 / §22 |
| **LA PRECISION D UN JOUEUR SANS AUCUNE REFERENCE EXTERNE (le livrable produit)** | `taux de remplissage` · `porteurs / records` · `0.4267` · `0.4462` · `0.0266` · `+0.8204` · `port/rec` · `SUIT` · `PLAT` -> 7ter.98 (index §21.3) et 7ter.101 (index §22.4) ; **guide qui fait foi pour l usage** : `.ai/GUIDE_WEAPON_SHOTS.md` **§3quater** (avec ses reserves : `19/34` · `deficit de 7 %` · `seul le TAUX`) |
| **L UNITE DU RECORD DEPEND DE L ARSENAL (portee, a ne jamais omettre)** | `UNITE DEPEND DE L ARSENAL` · `rc.ref` · `FIESTA 174` · `Tactical 45` · `fond permute intra-film` · `CONDITION D INDICE` -> 7ter.101, index §22 |
| **Indice joueur du fire-event (5 bits) / ventilation des tirs** | `playerIndexBitOffset` · `bit emprunte` · `getXuidToPI` · `match_weapon_shots` · `EvaluateShotsGate` -> 7ter.78, et `.ai/GUIDE_WEAPON_SHOTS.md` |
| **Morts infligees PAR un bot (population neuve)** | `OriginBotKiller` · `orphD` · `tueur-bot` · `380 / 380` -> 7ter.79 |
| **Parts de degats / les deux blocs R32** | `PercentageDamageDone` · `+0x228` · `killerPct` · `DamageShare` · `damage_pct_residual` -> 7ter.75 |
| **Assistant : ce qui est mort et ce qui le remplace** | `multiset` · `7ter.24ter` · `AssistExtra` · `quadScore` -> 7ter.76 |
| **Corps des codes 6 / 7, tag d impact, longueur du corps** | `evFixed[7] = 118` · `b0 == 1` · `168 380` · `H(f1)` · `tmp_c7b` · `tmp_refc7` · `reprise de chaine` · `0.2029` · `36 - P(carte)` · `X Heavies` -> 7ter.91 et 7ter.94, index §19 |
| Verifications adversariales (font foi sur les chiffres) | `VERIFICATION ADVERSARIALE` -> 7ter.38, .41, .43, .46, .49, .50, .53, .86 (`ap.ref`), .87 (`pv.ref`), .89 (`tv.ref`), .94 (`tp.ref`) |
| Verite terrain Theater | `THEATER` · `REPONSE THEATER` · `verite terrain` |
| Bilan cumule des confrontations | `BILAN TERRAIN` · `confrontations` |
| **Un tag precis (OBLIGATOIRE avant toute conclusion)** | son hexa 8 chiffres, ex. `daa03c35`, `0d203522`, `0000d627` |
| Ce qui a ete refute dans une session | `MESUREES COMME FAUSSES` (bloc `(D)` de fin de section) · `REFUTE` · `**FAUX**` |
| Gates et couverture | `gate (a)` · `gate (b)` · `credibles` · `couverture` |
| Calibration | `axisW` · `calib2` · `cv_autocalib` · `recordStateParam` · `PLAT` |
| Bascules A/B et modes | `DS_` `GP_` `GH_` `GV_` · `ev.exe` · `evtl` · nom de mode (`cvgap`, `gpbtb`, `ghbind`...) |
| Baselines et honnetete | `BASELINE TRIVIALE` · `A DIRE FRANCHEMENT` · `HONNETETE` · `RESERVE` |
| Bots | `bot` · `bid(` · `indice 8` · `BOT_METADATA` · `minXUID` |
| Medailles | `medaille` · `Kong` · `MedalType` · `gpmed` |
| Grenades | `gggl` · `entree 0` · `kineticbanished` · `botg` |
| Objets explosifs | `95b23ee5` · `0x1d8` · `pball` · `OBJET EXPLOSIF` |
| Vehicules — NOMMAGE de l arme (`.module`) | `vcdd` · `uwfa` · `R-VEHICULE` · `turret` (pour le vehicule comme ENTITE du film, voir la ligne `ti=40` plus haut) |
| Pont sonore | `sbnk` · `BKHD` · `.pck` · `FNV-1` · `flaveur` |
| Adresses et desassemblage | `FUN_14` |
| Commandes reproductibles | `COMMANDES` (bloc de fin de chaque section) |
| Lecons de methode | `LECON` · `REGLE A APPLIQUER` · `PIEGE` |
| Chiffres perimes / corrections | `CORRECTION` · `PERIME` · `ANNULEE` · `RETIREE` |

### 6.1 SIX PIEGES DE LECTURE DU FICHIER — a connaitre avant de greper

1. **LE FICHIER N EST PAS EN ORDRE NUMERIQUE.** Ordre physique de la fin :
   `... 7ter.48 · 7ter.47ter · 7ter.49 · 7ter.51 · 7ter.51bis · 7ter.50 · 7ter.52 · 7ter.52bis ·
   7ter.53 · 7ter.56 · 7ter.57 · 7ter.58 · 7ter.59 · 7ter.54 · 7ter.59bis`, puis la serie du
   2026-07-26/27 : `7ter.70 · 7ter.72 · 7ter.71 · 7ter.73 · 7ter.74 · 7ter.74bis · 7ter.75 ·
   7ter.76` (**la VERIFICATION d une section precede parfois physiquement la section verifiee**).
   **Ne jamais supposer qu une section est posterieure a une autre parce que son numero est plus grand.**
2. **SEPT numeros portent DEUX en-tetes** : `7ter.25 .26 .27 .29 .31 .32 .34` (PAS .33 — la seconde
   occurrence est `7ter.33bis`, section distincte, comme `7ter.28bis`). La regle << resume AVANT
   `## 12. REGLES` (l.2121), detaille APRES >> ne vaut que pour `.26 .27 .29 .32 .34` : les deux blocs
   de `.25` (1036/1208) et `.31` (1553/1656) sont TOUS DEUX AVANT. Le bloc DETAILLE fait foi.
   la premiere moitie du fichier, un bloc DETAILLE **apres la ligne `## 12. REGLES`**. Le DETAILLE fait
   foi sur les chiffres.
3. **Les sections `## 8` a `## 12`** (FAITS RE STABLES · IMPASSES HISTORIQUES · HISTORIQUE DES METHODES ·
   OUTILLAGE · REGLES) sont **AU MILIEU** du fichier, entre 7ter.2 et le bloc detaille de 7ter.26.
4. **Il n existe pas de `7ter.55`**, ni de `7ter.47bis`. `7ter.42-pre` existe (predictions scellees).
5. **`precision` A DEUX SENS DANS CE CHANTIER, ET UN GREP NU MELANGE LES DEUX.** (a) La **precision
   de QUANTIFICATION** d un composant de position (`axisW`, niveau `L`, `W = 6 + L`) — greper
   `axisW` · `calib2` · `cv_autocalib`. (b) La **precision de TIR** (`shots_hit / shots_fired`) —
   greper `precision de reference` · `0.446` · `arme dominante` · `purete` · `partBit` · §5.7,
   §14, §15, §17. Meme piege sur `tir` (tir d arme / tirage de nulle : greper `tirages >= reel`
   pour le second) et sur `touche` (drapeau `+106` = **7ter.80/82** ; evenement d impact codes 6/7
   = **7ter.86**).
6. **LA SERIE `7ter.80` A `7ter.88` EST, ELLE, EN ORDRE NUMERIQUE** a la fin du fichier (contrairement
   au piege 1, qui porte sur les series anterieures) — et **cinq de ces sections portent un
   `RENVOI EN TETE`** pose sous leur titre : le titre de **7ter.84 est REFUTE** (7ter.86), celui de
   7ter.85 est corrige (7ter.87), ceux de 7ter.80 / 7ter.81 / 7ter.83 sont limites en portee.
   **Ne jamais citer une de ces sections sans avoir lu son renvoi.**

### 6.2 CONTRADICTIONS INTERNES CONNUES — la plus RECENTE et celle de VERIFICATION font foi

| POINT | ETAT PERIME | ETAT COURANT |
|---|---|---|
| Table d occurrences de 7ter.39(4) | union = 38 tags, `d7df0000` present | **37**, `d7df0000` DISPARU (7ter.41 C1) |
| Test `jpt!` | 34/35 (7ter.38), puis 37/37 (7ter.41 C2), puis 48/48 (7ter.42) | **59/59 cumule sur 4 films** |
| Decomposition de R-VEHICULE | « 61 par `vehi`, 1 par `vcdd` » (7ter.45) | **46 `vehi` direct / 16 chaine `vcdd`** (7ter.46(4)) |
| Etat de degat, denominateur | `n/5` (7ter.48, encore dans des livrables) | **`n/7`** (7ter.49(2)b, arrete 7ter.57) |
| Nombre de `sbnk` | 4214, « 2261 sans BKHD » (7ter.48) | **2107 distincts, 154 sans BKHD (7.3%)** (7ter.49(2)a) |
| Arithmetique de 7ter.39(5) et 7ter.40(8) | « 15..65 : 11 » · « 19/19 ancres » | **13** · **20/20** (7ter.41 C6) |
| Sante du lecteur `.module` | « 0 <= ecart < 4096 » (7ter.39(0)) | **36 archives sur 132 ont un ecart NEGATIF** ; sans consequence sur l archive porteuse des armes (7ter.41 C5) |
| Carte de 78919882 | « Altitude » (7ter.42-pre) | **High Ground**, carte FORGE (7ter.42(0)) |
| « Correction Flood Gulch / Ravin Parasite » | presentee comme une correction (7ter.52 C0) | c est une **TRADUCTION** (`Flood` = `Parasite` en FR), pas une correction (7ter.53 R6ii) |
| Bots et kill-feed | « le kill-feed du film est structurellement incomplet » | **l index HIGHLIGHT est un index de JOUEURS** ; le bot est dans la voie REPLICATION (7ter.59) |
| **Validation des assistants** | « 31 decodes contre 30 API, RaiiZeNBack 3 = 3 » (7ter.24ter) — **RECOPIE DANS LE BRIEF D UN CHANTIER ET UTILISE COMME REFERENCE PENDANT DES SEMAINES** | **22 nommes pour 30** sur `9b191a7f` (le 31 comptait des faux positifs), l ancre par joueur portait sur un **AUTRE joueur** (xuid resolu ailleurs par la bijection courante) ; la validation qui tient est le **MULTISET par joueur**, 17/17 et 29/29 sur deux films (7ter.76) |
| **Grammaire du kill-event** | `killer(E5) victim(E5) R32 R1 assist(E5) R32` (7ter.24ter), les deux R32 « inexpliques » | `victime(E5) tueur(E5) [% TUEUR] R1 assistant(E5) [% ASSISTANT]` — ordre INVERSE et les deux R32 sont des **parts de degats en pourcentage entier** (7ter.75) |
| **Cardinalite de l assistant** | « cardinalite MESUREE A 1 sur 100 % de la population » (la formule venait de `internal/migration/steps_shared_kill_events.go` ; CE FICHIER A ETE CORRIGE le 2026-07-28 et porte desormais le bon denominateur) | denominateur reel = **LES KILL-EVENTS ATTACHES** : la nullite du compteur prouve *aucune mort n a recu deux ENREGISTREMENTS nommant des assistants distincts*, PAS *aucune mort ne porte deux assistants* (7ter.76 (6)) |
| **La touche de projectile dans le flux d events** | « **PAS dans un flux d events** — trois eliminations sans ancre » (titre et resume de **7ter.84**, 2026-07-28) | **FAUX. Les codes 6 et 7 y sont** : 80 886 et 129 390 sur 949 films, dont 80 % en PREMIER evenement. Ils figuraient deja dans l histogramme de 7ter.84 (3) aux rangs 14 et 11, **sous une coupure d affichage a 9 lignes**. Meme corpus, meme population de paquets a l unite, binaire independant (**7ter.86 (3)**) |
### 6.2bis CONTRADICTIONS NON TRANCHEES — deux mesures vivantes qui se contredisent (2026-07-27)

> A la difference de §6.2, **aucune des deux n est resolue**. Les ecrire ici evite qu un lot
> futur en cite une moitie comme un fait. Traitement prescrit dans les deux cas : **§13.1 —
> comparer les POPULATIONS avant de decider qui a tort.**

| POINT | MESURE A | MESURE B | CE QUI SURVIT AUX DEUX |
|---|---|---|---|
| **Deficit du Mk51 Sidekick** | **AUCUN deficit** : ratio 1.004 (mode standard, 392 films, ratio PAR JOUEUR) — et le Sidekick n est pas une arme automatique (183.8 ms) — 7ter.80 (6) | **DEFICIT** : `k_fire` = **0.935**, colle au MA40 (0.936), et `k_enr` 0.795 contre 0.842 pour le BR75 (949 films tous modes, population « arme dominante >= 85 % des DEUX comptes », AGREGAT de tirs) — 7ter.81 (9) | **Le MA40 a un deficit, plus grand que celui du BR75** — `[ETABLI]`, deux lots, deux estimateurs. Le cas Sidekick est **CONTESTE** : ne le publier ni comme present ni comme absent. |
| ~~**Un bit de touche existe-t-il dans le fire-event ?**~~ **TRANCHEE PAR 7ter.82 (7)** | **NON** : 656 positions balayees, aucune ne bat un taux constant — mais sur **8 joueurs** | **OUI pour les armes a trace instantanee** : bit `eventStart+106`, 0.1070 contre 0.1900, sur **1 556 joueurs** — 7ter.80 (8), **reproduit** 7ter.82 | **LES DEUX AVAIENT RAISON, ET LA RECONCILIATION EST FAITE** : refait a 8 joueurs, un balayage aveugle ne DESIGNE `+106` que sur 42.3 % des films (au-dela du rang 30 sur 25/196) alors que le drapeau y bat deja le taux sur 172/196 films. **656 candidats pour 8 observations : le vainqueur d un film ne se reproduit pas.** Deficit d effectif, pas contradiction. Le champ `HitLikely` reste mort dans les deux lectures. |

---

## 7. CE QUE CET INDEX NE COUVRE PAS

- **Le detail des mesures** : chaque entree est un pointeur, pas une preuve. Les commandes, les
  A/B, les controles negatifs et les p-values restent dans le RE_LOG.
- **Les sections 7ter.1 a 7ter.25** (juillet, ere same-clock / walker d events) ne sont indexees
  que par leurs verdicts survivants ; leur detail historique n a pas ete depouille ligne a ligne.
- **Les documents satellites** : `HANDOFF_KILLWEAPON.md` (point d entree), `PONT_SONORE_ARMES.md`
  (bandeau NON CONSOMMABLE, 15 corrections non appliquees), `HANDOFF_PONT_SONORE.md`,
  `V7.5/replay2d/HANDOFF_ALL_PLAYERS_TRAJECTORIES.md`. Ils peuvent porter des chiffres PERIMES par le §5.
- **Les deux guides de branchement** : `.ai/V7.5/killweapon/GUIDE_KILLSOURCE.md` (arme du KILL, tag `jpt!`
  32 bits, table `match_kill_events`) et `.ai/GUIDE_WEAPON_SHOTS.md` (ventilation des TIRS,
  identifiant filmshell 64 bits, table `match_weapon_shots`). **DEUX ESPACES D IDENTIFIANTS
  DISTINCTS QUI NE SE JOIGNENT JAMAIS** — les confondre rend zero ligne, silencieusement.
- **L etat du code** : ce fichier indexe des CONCLUSIONS, pas des fichiers Go. Deux correctifs
  mesures ne sont **toujours pas poses** (monde chronologique dans `cv_autocalib.go` ; plage de
  credibilite derivee dans `reportEvDeads`), et le gate `EnumA <= 7` d `evrun.go` est toujours la.
- **Le RE_LOG est en cours d ecriture par plusieurs agents** : les sections indexees vont jusqu a
  **7ter.89** (§9 a §18). Verifier la fin du fichier avant de considerer cet index comme
  complet — `7ter.78` EXISTE (§13) mais `7ter.55` et `7ter.47bis` N EXISTENT PAS.
  ⚠ **DEUX FOIS DEJA, DEUX LOTS ONT VISE LE MEME NUMERO** : `7ter.82` le 2026-07-27 (`rt.rep` l a
  pris, le lot des statlines est parti en **`7ter.83`**) et `7ter.89` le 2026-07-28 (le lot
  vehicules l a pris alors qu un lot encore en cours avait recu la meme consigne — voir le bandeau
  RESERVATION DE NUMERO en tete de fichier). **Avant d ecrire une section, `grep "^### 7ter"`,
  prendre le premier numero REELLEMENT libre, et POSER L EN-TETE AVANT DE MESURER** (§1 regle 10) —
  ne jamais faire confiance au numero annonce dans un brief.
- **Ce que cet index ne remplace PAS pour le branchement** : `.ai/V7.5/killweapon/GUIDE_KILLSOURCE.md` fait foi sur
  l API du paquet `killsource`, le schema `shared.match_kill_events` et le chemin de retrait de
  `killer_victim_pairs`.

---

## 8. COMPLEMENTS APRES CRITIQUE DE COMPLETUDE (2026-07-26)

> Un agent de completude a relu le journal en entier contre cet index. Verdict : **<< non, pas
> encore >>** — deux trous structurels. Ils sont bouches ci-dessous.

### 8.1 LE CHANTIER 7ter.54 ETAIT SOUS-INDEXE — et pour une raison instructive

`7ter.54` porte le **plus grand numero de la serie** mais est **avant-derniere PHYSIQUEMENT** dans
le fichier : exactement le piege de lecture que le paragraphe 6.1 documente. Ses verdicts porteurs :

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| L arborescence de dispatch POV/bot/proxy explique-t-elle le `DELTA slot NON LIE` ? | **NON.** `FUN_1406caad8` a exactement UN appelant. Le flag de propriete (`param_7`) bascule vers une file d etats en ombre, mais la table d entites n a **qu un ecrivain**, appele sur `param_7 == 0`. La propriete ne touche pas la liaison slot -> archetype. | 7ter.54 AXE1 · MESURE+VERIFIE |
| Les slots migrent-ils au respawn, et est-ce la cause du blocage ? | Le mecanisme est **confirme** (le client RECOPIE la generation du flux, il ne la calcule jamais ; un respawn est toujours annonce par un record NEW, et il n en existe **aucun autre**). Mais **ce n est PAS la cause** : les slots ou la marche meurt sont 1420 a 6962 DISTINCTS par film, domines par 26, 27, 0, 2, 2048 — **aucun dans la plage bipede**. C est du **BOURRAGE lu comme un slot**. | 7ter.54 AXE2 · MESURE+VERIFIE |
| Les tables de precision dumpees remplacent-elles le balayage de calibration ? | **NON.** Decodees et validees 7/7 par forme fermee non circulaire, mais le dump est un **instantane installe PAR MAP**, pas une specification portable : l injecter fait s effondrer 3 films sur 4. Ce que l axe apporte : le parametre manquant est **NOMME** (niveau L de precision de la position, W = 6+L) et le modele a largeur UNIQUE est retro-justifie. | 7ter.54 AXE3 · MESURE+VERIFIE |
| Le chemin de delta du JEU tolere-t-il un slot non lie ? | **NON — il abandonne le paquet entier.** Notre `DELTA slot NON LIE` et le test strict de generation sont donc des **PORTS FIDELES**, pas des faiblesses. | 7ter.54 AXE2 · MESURE |
| Ou sourcer le niveau L hors du film ? | Les **archives `.module` de la map** — meme corpus que la geometrie 2D. Ni le film ni l .exe (les deux tables lisent zero en statique). | 7ter.54 (F)1 · MESURE |

### 8.2 LE PROTOCOLE DE VERITE TERRAIN — c est la recette qui a produit le 25/0

Il n etait nulle part dans cet index, alors que c est **la ressource la plus rare du chantier** :
```
1. DECODER TOUT, puis classer chaque kill par CONFIANCE INTERNE (sans verite terrain) :
     certain  = tag deja ferme en Theater, OU une seule lecture possible ET categorie coherente
     douteux  = deux lectures concurrentes, OU categorie contredisant l arme, OU decodage marginal
     inconnu  = le tag ne remonte a aucune source
2. NE SOUMETTRE QUE douteux + inconnu — une POIGNEE de lignes, JAMAIS une table.
3. Toujours avec DATE + CARTE + MODE, et sur des matchs RECENTS : l historique Theater n est pas
   date, un match de 4 mois est en pratique inverifiable. `go run ./cmd/tmp_datemap recent`.
4. Toujours designer LA ligne qui discrimine le plus, pour qu il puisse n en regarder qu une.
5. Ecrire les CONDITIONS DE FALSIFICATION avant de poser la question : ce qui falsifie, ce qui NE
   falsifie PAS, ce qui est NON DECIDABLE. Une condition mal ecrite detruit de l information.
```

### 8.3 DEUX LIGNES DU PONT SONORE SONT FAUSSES, ET C EST MESURE

`grenadeEntryOf` prend le **minimum** des entrees atteintes. Or les `jpt!` generiques traversent les
couples `gggl` (`88f1034c` pend aux rangs 0,1,2,3,6,7 ; `31e8d17e` et `d21ac495` aux rangs 0-3).
Donc `88f1034c` et `31e8d17e` sortent un nom de banque **affirmatif et faux**. Aucun tag observe sur
les films n est touche, mais un consommateur publierait l erreur sans le savoir.
Sections : 7ter.45 (9)2, 7ter.46 (7), 7ter.49 (2)d, 7ter.50 (3)1.

### 8.4 `internal/himodule` NE SAIT PAS LIRE `globals-rtx-new.module`

L archive qui contient les armes. Trois causes, **toutes corrigees dans un lecteur LOCAL**
(`cmd/tmp_tagname/hmod.go`) et **jamais remontees dans `internal/himodule`** :
```
1. `dataOffset` est un entier 48 BITS + 16 bits de flags, pas un u32 (8 764 entrees > 4 Go)
2. flag bit0 = UseHd1 -> la donnee est dans le compagnon `.module_hd1` (737 entrees)
3. ordre des tables = [entrees 0x58][2 u32 sentinelles 00000000 ffffffff][resources u32][blocs 20o]
   -> ce sont ces 8 octets de sentinelles qui bloquaient tout
```
Section 7ter.39 (0). **Sans cette entree, un verrou technique complet se re-craque a la main.**

### 8.5 LE GATE (c) ET SA CORRECTION DE CIRCULARITE

Valeur : **10/14** sur les kills Theater de JGtm, **inchangee depuis 7ter.37** malgre +21 morts de
couverture (les 4 kills manquants echouent tous pour cause structurelle amont).
**LE << 10/10 VICTIMES >> N EST PAS UNE PREUVE INDEPENDANTE** : la bijection indice -> joueur est
ajustee sur le kill-feed, donc identifier les victimes en decoule mecaniquement. Le gate (c)
n apporte **qu une seule quantite neuve : le TAG D ARME**. Sections 7ter.37 (3), 7ter.38 (4),
7ter.41 (7). C est le chiffre le plus tentant du chantier : le re-annoncer comme preuve, c est
refaire le patron A.

### 8.6 LA REFERENCE ELLE-MEME A UNE LIGNE CONTESTEE

Le kill-feed attribue **15** kills a JGtm sur `9b191a7f`, la liste Theater en donne **14**. L extra
est `08:31`, qui decode en auto-kill (victime == tueur) mais porte le tag Fuel Rod. Non tranche.
Section 7ter.53 (6). A savoir avant de compter un score sur ce film.

---

## 9. LA PERCEE DU 2026-07-26 SOIR — scan direct, morts auto-infligees, couple fantome

> Trois resultats qui changent l etat du chantier. Sections `7ter.60` a `7ter.66`.

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Faut-il marcher les records depuis le debut du paquet ? | **NON.** Le SCAN DIRECT balaie toutes les positions de bit et ne retient que ce qui passe 4 tests (victime, tueur, categorie, et le tag dans `jpt!`). Cela **contourne entierement** le probleme des largeurs : calibration, keyframes et localisateur slot-123 deviennent FACULTATIFS pour la question de l arme. **88.7 % -> 97.6 %.** | 7ter.60 · MESURE+VERIFIE |
| Le scan invente-t-il des morts ? | **NON.** Controle negatif : **1 candidat sur 208 913 712 bits** de paquets sans mort. Concordance avec la marche : **346/346 AU MEME BIT**. Leurre de 468 ids AU HASARD : **0 candidat**. Dans les paquets sans event les tests residuels se comportent EXACTEMENT comme le hasard (1 observe, 1.54 attendu) ; dans les paquets a events ils passent **85 fois plus souvent**. | 7ter.61 · MESURE+VERIFIE |
| Le scan est-il meilleur que la marche ? | **NON, il est COMPLEMENTAIRE.** Il gagne 29 morts mais **DEGRADE** le gate (b) : 96.3 % contre 98.2 % a armes egales, et 12.7 % d enregistrements inexplicables contre 3.5 %. **L architecture de production est HYBRIDE** : la marche d abord (precise), le scan en RATTRAPAGE. Les deux concordent 346/346, leur union est sans conflit. | 7ter.61 R1 et C2 · MESURE+VERIFIE |
| Pourquoi le decodeur dit-il parfois << victime == tueur >> alors que le feed credite un autre joueur ? | Parce que **les deux repondent a des questions differentes** : le feed donne le CREDIT, le dead-state donne la SOURCE DU DEGAT. Roquette tiree trop pres, baril lance trop pres, chute : la source appartient a la victime. **Verifie 8/8 en Theater.** | 7ter.63, 7ter.66 · THEATER |
| Le filtre `v != k` est-il legitime ? | **NON tel quel.** Il reposait sur << `victime == tueur` = signature de dechet >> (7ter.38), vrai mais INCOMPLET. Deux phenomenes partagent la signature. Discriminant MESURE : les faux se repetent AU MEME BIT avec un tag < 0x10000 ; les vrais sont a des bits varies avec des tags forts et NOMMES. | 7ter.63 · THEATER |
| La mort sans aucun candidat, c est quoi ? | **Ce n est pas une mort.** C est un COUPLE FABRIQUE par notre heuristique : la vraie victime est un BOT, dont la mort ne produit aucun event dans le chunk HIGHLIGHT (humain-seul), donc `feedPairs` prend la victime du voisin. Le dead-state du bot EXISTE (indice absolu 8) et il est jete par un `< nPlayers` code en dur. | 7ter.66 · THEATER |
| Combien de fois le decodeur a-t-il corrige la reference ? | **TROIS.** Contre une reponse de l utilisateur (grenade a pointes, 7ter.47ter), contre le KILL-FEED DU JEU (8 morts auto-infligees, 7ter.63), et contre NOTRE PROPRE appariement (7ter.66). Les trois conditions de 7ter.47ter etaient reunies a chaque fois. | 7ter.47ter, .63, .66 · THEATER |
| Quel denominateur citer ? | **TROIS coexistent et il faut les nommer** : 372 couples reconstruits (contient au moins un couple FABRIQUE) · 375 morts du feed (aucune mort de bot) · **380 morts de l API (la seule complete)**. Ne jamais ecrire << X % des morts >> sans dire lequel. **Le NUMERATEUR du 4e est passe de 376 a 380 en 7ter.79** ; les trois autres n ont pas bouge. | 7ter.66, amende 7ter.79 · MESURE |
| Le decodage est-il PUR offline ? | **PAS ENCORE.** Le catalogue de 468 ids `jpt!` se construit en lisant les EN-TETES des archives du jeu ; le VPS de prod n a pas Halo. **Directive utilisateur : embarquer les en-tetes.** Remede trivial (~2 Ko versionnes) + table `tag -> nom` precalculee. **A PROUVER, pas a affirmer** : rendre les fichiers du jeu inaccessibles et montrer que les chiffres ne bougent pas. | handoff 0f · A FAIRE |

### 9.1 CE QUI RESTE, APRES CETTE SERIE

```
8 morts auto-infligees   recuperables par la levee du filtre `v != k` + les 2 discriminants
1 couple fantome         a RETIRER du denominateur (ce n est pas une mort)
=> plafond sur les couples REELS : 372/372
NOUVELLE QUESTION OUVERTE : la couverture des MORTS DE BOTS, jamais mesuree parce qu elles
n ont jamais ete dans le denominateur. Le correctif est connu et 100 % offline : deriver
nPlayers du paquet BOT_METADATA (nbBots + slot, 7ter.62) au lieu du 8 code en dur.
```
**CETTE QUESTION EST CLOSE, DANS SES DEUX MOITIES** : morts DE bot en 7ter.70 (5 lignes), morts
PAR un bot en **7ter.79** (4 lignes). Les morts de l API sont a **380/380**.

### 9.2 LA LECON DE METHODE DE CETTE SERIE

**Les trois corrections de reference ont la meme forme** : une mesure juste a ete transformee en
regle trop large.
```
7ter.38   << `v == k` est une signature de dechet >>   -> devenu << donc tout `v == k` est du dechet >>
7ter.45   << ces 4 supports ne portent pas de noms >>  -> devenu << le format ne porte pas de noms >>
7ter.52   << le chunk HIGHLIGHT n a pas de bots >>     -> devenu << le film n a pas de bots >>
```
**REGLE : une mesure porte sur ce qu elle a mesure. Ecrire la portee AVEC le resultat, toujours.**

---

## 10. L HYBRIDE ET LA METRIQUE DE SANTE (2026-07-27, section 7ter.72)

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| L hybride << marche d abord, scan en rattrapage >> fait-il remonter le gate (b) global ? | **NON — +0.00 point sur les QUATRE films.** Le gate (b) est une propriete de l ENSEMBLE des enregistrements consultes, pas de l ORDRE de lecture : les 346 enregistrements partages sont les MEMES objets au MEME bit. Hypothese MESUREE COMME FAUSSE. | 7ter.72 (2) · MESURE |
| Que gagne-t-on alors ? | Le cout est **LOCALISE ET ETIQUETE** : gate (b) **98.2 % (334/340) pour la marche** contre **78.4 % (29/37) pour le rattrapage du scan**. Chaque ligne publiee porte son ORIGINE (classe + voie). Le 78.4 % n avait jamais ete isole. | 7ter.72 (2) · MESURE |
| La couverture bouge-t-elle ? | Non : **371/371 couples REELS**, identique a 7ter.70, et **30/30 ancres Theater CONFORMES** tags et categories compris (25 des 30 ont au moins une ligne de la MARCHE dans leur seconde ; 5 ancres de `9b191a7f` sont servies par le SCAN SEUL). | 7ter.72 (5) · MESURE |
| Les deux voies se contredisent-elles jamais ? | **DESACCORD = 0** sur les 334 instants ou les deux repondent ; **0 mort a plus d un candidat en concurrence**. L hybride PREFERE, il n arbitre pas — et le mecanisme d arbitrage n a jamais eu a servir. | 7ter.72 (1) · MESURE |
| Le vrai benefice de l hybride, alors ? | **LA ROBUSTESSE A UN CATALOGUE PERIME.** La marche n a AUCUNE porte T4. Ablation d un tag reel, 20 essais : scan-d-abord perd **224 lignes**, l hybride **20**. Facteur **11.2**. | 7ter.72 (3) · MESURE |
| Le filtre `TAG >= 0x10000` est-il retenu ? | **OUI**, par defaut (`fcSelfOK`) — 7ter.70 ecrit l inverse. Il achete de la JUSTESSE, pas de la couverture. **Sous l hybride il ne change plus AUCUNE etiquette** (0/4 films) : la marche publie `acd1cff4` directement a fccc61cd 01:25. | 7ter.72 (0)(C) · MESURE |
| Le bloc `fcself` de 7ter.70 est-il general ? | **NON, un seul film.** Cout du test de tag par film : 1/92 · **47/82** · 19/97 · 2/92. Le BR75 `0000b29c` vit sous 0x10000. Le seuil de multiplicite est **INERTE sur 2 films sur 4**. | 7ter.72 (0)(A)(B) · MESURE |
| Comment detecter un catalogue `jpt!` PERIME ? | Par la **MARCHE** : un de ses enregistrements apparie au COUPLE EXACT dont le tag est absent du catalogue. **Bruit NUL** (la marche n a pas de porte T4). Distribution 0/0/0/0/0 ; controle positif **20 ablations sur 20**, dont 8 SANS perte de couverture. | 7ter.72 (4b) · MESURE |
| La sonde a T4 relache sert-elle d alerte ? | **NON** : rapport signal/hasard **1.10 a 1.72**. Utilisable seulement restreinte aux morts NON COUVERTES, ou elle vaut zero par construction a 100 % de couverture. Publiee, **exclue des alertes**. Piege : sans exclusion du couple FANTOME elle rend 6 sur `9b191a7f`. | 7ter.72 (4c) · MESURE |
| Le compteur << hors roster >> est-il fiable brut ? | **NON** : 2 faux sur `78919882` (`tag=ffffffff`, `v=-1`) — du dechet. Il exige la porte du catalogue. Durci : 0/0/0/0 a 8 joueurs, et **3 VRAIS sur le BTB** (indices 28/29/30 pour 36 participants API) — premiere detection positive. | 7ter.72 (4d) · MESURE |
| Les seuils de la metrique de sante ? | Tires de la distribution de **CINQ films** : inexpliques 7.0 / 9.4 / 11.6 / 17.8 % a 8 joueurs, **27.0 % en BTB**. HORS DOMAINE > 18.0 % ou couverture < 100 % ; ALERTE > 36.0 % ou l un des deux compteurs a distribution nulle. Le BTB est le CONTROLE POSITIF DE DOMAINE et il sort par TROIS criteres. | 7ter.72 (4a) · MESURE |
| Ou vit la metrique ? | `internal/analysis/filmdec/killhealth.go` (type PUR, 0 dependance interne, 6 tests verts) + publication `expvar` cablee par l appelant (ADR 0009, namespace `levelup`, **compteurs ENTIERS, aucun ratio**). | 7ter.72 (4) · MESURE |
| Le BTB sous l hybride ? | **224/293 = 76.5 %** (contre 177/294 = 60.2 % en 7ter.52). **AGREGAT SEUL** : la marge de bijection a 25 joueurs est NULLE (7ter.53 (4)) et n a PAS ete re-verifiee. | 7ter.72 (7)3 · MESURE, RESERVE |

**STATUT** : 7ter.72 n a **PAS** recu de verification adversariale.

---

## 11. LES DEUX PARTS DE DEGATS, ET LA MORT DE `7ter.24ter` (2026-07-27, sections 7ter.75 et 7ter.76)

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Les deux blocs R32 du kill-event sont-ils encore << inexpliques >> ? | **NON.** Ce sont des **PARTS DE DEGATS EN POURCENTAGE ENTIER** : la premiere pour le TUEUR, la seconde pour l ASSISTANT. | 7ter.75 (3) · MESURE |
| Sur quoi cela repose-t-il ? | **Quatre jambes convergentes** : (1) adjacence et ordre dans le modele de recap (`KillerPercentageDamageDone` `+0x228`, `AssistantPercentageDamageDone` `+0x22c`) ; (2) type ENTIER confirme au desassemblage (tag 1 contre 2 pour les flottants voisins) ; (3) **somme == 99 sur 22 367 des 31 204 kills assistes de 892 films** (2e valeur 89 a 1.78 %, RIEN a 100) ; (4) collision avec une capture Cheat Engine du 2026-06-10 sur la constante arbitraire **149** (3 films sur 886, absente de 983 000 lectures a position aleatoire). | 7ter.75 (3) · MESURE |
| Quelle RESERVE porter avec ces deux nombres ? | Le **CHEMIN DE DONNEES** entre le kill-event du film et ces deux champs du modele de recap **N EST PAS DEMONTRE** : l ecrivain de `+0x228` est un setter GENERIQUE partage par **9 modeles d UI**. Quatre jambes convergentes, PAS une chaine d appels prouvee. | 7ter.75 (3) · RESERVE |
| Peut-on trancher en Theater ? | **NON, c est IMPOSSIBLE** : l ecran concerne est le recap de mort en PREMIERE PERSONNE, que le Theatre ne rejoue pas ; et il n existe que **deux chaines `PercentageDamageDone`** dans tout l executable, sans libelle localise. | 7ter.75 (4) · MESURE |
| Faut-il plafonner a 100 ? | **NON.** **1,7 % des kill-events attaches a de vraies morts nommees** portent une valeur superieure, **jusqu a 228**. Un plafond jetterait de la donnee reelle pour proteger une interpretation — et une seule ligne a 228 faisait echouer la passe ENTIERE d un match. | 7ter.75 (5) · MESURE |
| Le second bloc se lit-il toujours ? | **NON** : sans champ assistant il porte une **CONSTANTE PAR FILM** (149, 70, 20, 197) — et **20 est un pourcentage credible**. Population : **74 %** des lignes attachees du corpus large (442/595), **77 %** sur les quatre films de reference (286/370). L absence d assistant se lit dans `assist_gamertag`/`assist_known`, JAMAIS dans `assist_damage_pct`. | 7ter.75 (6) · MESURE |
| Quel est l ORDRE des deux premiers champs ? | **VICTIME puis TUEUR** (facteur 6 a 13 sur les 4 films). Consequence : le rejet historique << assistant == tueur >> portait sur `assistant == victime`. | 7ter.75 (2) · MESURE |
| Le << 31 assists contre 30 API >> de 7ter.24ter tient-il ? | **NON, MORT SUR SES DEUX CHIFFRES.** Le 31 comptait les faux positifs d un pipeline a 99 kills pour 87 morts reelles (nettoye : **22 pour 30**), et l ancre par joueur portait sur un **AUTRE joueur** que celui qu elle nomme. | 7ter.76 (1) · MESURE |
| Qu est-ce qui valide les assistants, alors ? | **LE MULTISET PAR JOUEUR** — comptes tries et **PRIVES DES NOMS**, donc **INVARIANT PAR LA BIJECTION**, et l assistant n entre pas dans l objectif optimise (`quadScore` ne voit que le couple). Deux films rendent le multiset de l API **A L IDENTIQUE sur 8 joueurs** : `000d5950` 6,3,2,2,1,1,1,1 (17/17) et `78919882` 7,5,4,3,3,3,2,2 (29/29). Fige par `TestAssistMultisetParJoueur`. | 7ter.76 (3) · MESURE |
| Le taux d attachement est-il une mesure independante ? | **NON, CIRCULAIRE** — `quadScore` maximise le meme critere. Il reste un **SELECTEUR DE POPULATION**, non circulaire pour ce seul usage. | 7ter.76 (2) · MESURE |
| Reste-t-il une anomalie ? | **OUI, ECRITE ET NON RESOLUE** : sur `9b191a7f`, 22 nommes + 6 morts non lues = **28 < 30 a l API**. Deux issues : une mort a deux assistants, ou certains << pas d assistant >> sont faux. Localisee : les deux films entierement attaches reproduisent l API **a l unite** (17/17, 29/29). **INSTRUITE ET AGGRAVEE en 7ter.77 : le plafond honnete tombe a 27.** **ETAT COURANT (7ter.79) : 24 nommes + 3 morts non lues = 27 < 30, ecart 3 — le plafond ne bouge pas, mais il n est plus atteint par des lignes qu on renonce a publier, et l ecart se localise sur trois joueurs.** | 7ter.76 (4), amende 7ter.79 (4) · RESERVE |
| Les morts a MULTI-ATTACHEMENT portent-elles un desaccord sur l assistant ? | **NON — zero desaccord sur les 4 films.** `Multi` vaut 1 sur `9b191a7f` et 1 sur `fccc61cd`, et dans les DEUX cas c est **le meme enregistrement lu deux fois**, a 15 bits d ecart dans le meme paquet, champs identiques. Preferer le hit porteur ([pickAssistHit]) est donc un **NO-OP sur le corpus** : c est une DECISION, pas une mesure — d ou le compteur `AssistFieldDisagree`, dont le zero est MESURE. | 7ter.77 (2) · MESURE |
| D ou viennent les 2 assistances manquantes de `9b191a7f` ? | **Pas de (a) ni de (b) : d une TROISIEME issue.** Les **3 kills DU BOT** portent un kill-event credible (chaine 12), **2 y nomment un assistant** — et aucun ne s attache : le chunk HIGHLIGHT est humain-seul, donc une mort infligee **PAR** un bot ne produit aucune paire de feed. Assistances LUES, non publiables. Le plafond devient 22 + 2 + 3 = **27 < 30** : l ecart passe de 2 a **3**. | 7ter.77 (4)(5) · MESURE |
| Le mecanisme de code qui pouvait nourrir l issue (b) est-il ferme ? | **OUI, et seulement lui.** `attachAssists` prenait `hits[0]` sans preferer un porteur : un desaccord aurait fabrique un << pas d assistant >>. Mesure : 0 desaccord sur le corpus. **PORTEE STRICTE** : c est LE MECANISME qui tombe, pas l issue (b) — un << pas d assistant >> faux par lecture ratee du bit de porte `E5` reste non mesure. | 7ter.77 (3) · MESURE |
| Le garde-fou << un seul assistant >> peut-il se declencher ? | **OUI, DESORMAIS.** `SELECT SUM(assist_extra_count)` valait zero PAR CONSTRUCTION (aucun champ de ligne ne l alimentait). `Assist.Extra` pose, et un test FABRIQUE le cas que le corpus n a jamais produit (`TestAssistExtraNEstPasMuet`). | 7ter.76 (5) · MESURE |
| Faut-il une porte de publication separee pour l assistant ? | **NON** : meme porte que le reste (`LineByLinePublishable`). Justification mesuree : **62 assistants nommes pour 122 a l API en BTB (51 %)** contre **17/17 et 29/29** en Arena — cause = marge de bijection NULLE, pas un decodage plus mauvais. | 7ter.76 (7) · MESURE |

---

## 12. LES MORTS INFLIGEES PAR UN BOT — LA SECONDE POPULATION NEUVE (2026-07-27, section 7ter.79)

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Comment IDENTIFIER la population, sans circularite ? | **PAR LE KILL-FEED SEUL.** `feed.split()` isolait les KILLS orphelins (mort DE bot) ; la moitie symetrique — les **MORTS orphelines**, qu aucun kill ne consomme — n avait jamais ete isolee. Une mort sans kill en face a un **tueur non humain**. Le dead-state ne sert qu ensuite, a decoder ; il ne sert JAMAIS a designer. | 7ter.79 (1) · MESURE |
| Qu est-ce qui rend la mesure FALSIFIABLE ? | **SA MOITIE NEGATIVE.** Sur les deux films SANS bot la population est **VIDE** (0 sur 93 et 99 morts au feed) ; sur les deux films AVEC bot son cardinal vaut **exactement** les kills que l API donne au `bid(...)` (3 et 1). Trois quantites independantes, elles coincident 4 films sur 4. | 7ter.79 (1) · MESURE |
| Que valent les 4 lignes ? | Tueur (le bot), victime, instant, **source du degat**, categorie, assistant et deux parts. **Toutes servies par la MARCHE** (voie a 98.2 %), chaine 12, decalage 4-6 ms entre feed et paquet. | 7ter.79 (1)(5) · MESURE |
| Quel denominateur bouge ? | **UN SEUL, ET C EST SON NUMERATEUR** : morts de l API **376/380 = 98.9 % -> 380/380 = 100.0 %**. Les trois autres (couples REELS, couples reconstruits, morts du kill-feed) sont INCHANGES, numerateur compris. `Coverage.Covered` ne compte PAS la population neuve. | 7ter.79 (3) · MESURE |
| Le controle qui decide ? | **Morts publiees par joueur == `deaths` de l API : ecart ZERO sur 34 participants**, bots compris, sur les quatre films. Et `couverts + morts DE bot + morts PAR un bot == morts de l API` film par film. | 7ter.79 (4) · MESURE |
| Les assistances progressent-elles ? | `9b191a7f` **22 -> 24 pour 30** (les deux assistances que 7ter.77 declarait lues-non-publiables). `fccc61cd` **INCHANGE a 16/18** : sa mort par bot ne nomme aucun assistant. **Les deux multisets des films exacts (17/17, 29/29) NE BOUGENT PAS D UNE UNITE.** | 7ter.79 (4) · MESURE |
| Y a-t-il une verite terrain ? | **NON, ET IL N Y EN AURA JAMAIS** : le kill-feed du jeu n attribue ces morts a aucun gamertag, donc on ne peut pas designer la ligne a verifier en Theatre. Ce qui remplace l ancre : la coherence **tag/categorie** — `0000d57f` (Needler) sort `AttachedDamage` = supercombinaison, `ec919da5` (Needler) sort `None` — et la concordance avec les categories relevees en 7ter.59 par un AUTRE outil. | 7ter.79 (5)(8) · MESURE |
| Le BTB ? | **IL NE FERME PAS, et c est ecrit** : 7 morts orphelines pour 5 kills de bot + 1 suicide, 6 des 8 bots hors de la regle d epinglage, **0 appariee, 0 publiee**. Le risque << un suicide tombe dans la meme population >> n est pas theorique : il est OBSERVE la, et seulement la. | 7ter.79 (6)(8) · MESURE |
| Ce qui n a pas bouge ? | 30 ancres Theater · 8 morts a source-victime · 371/371 couples REELS · controle negatif · DESACCORD 0 · marges de bijection. Cote sante : `UnexplainedBotIdx` 3 -> 0 et 2 -> 1, **seuils deliberement NON re-derives**. | 7ter.79 (7) · MESURE |

**STATUT** : 7ter.79 n a **PAS** recu de verification adversariale. La population ne fait que
**4 lignes**, et le controle de comptabilite la valide **en agregat par film** — il etablit que le
compte ferme, PAS que la bonne mort a recu la bonne source. Elle n a en outre **aucune ancre
Theater possible** (§12, ligne << Y a-t-il une verite terrain ? >>).

---

## 13. L INDICE JOUEUR DU FIRE-EVENT ET LA VENTILATION DES TIRS (2026-07-27, section 7ter.78)

> Chantier `tw`. Guide dedie : `.ai/GUIDE_WEAPON_SHOTS.md` (fait foi sur le schema, l API du
> persister et la portee de la donnee).

> **DEUX LIGNES DE CETTE SECTION SONT PERIMEES (2026-07-28, 7ter.92 / §20).** (a) La ligne
> << `getXuidToPI` NON REMPLACE — hors perimetre >> de §13.2 : **il EST remplace**, le pipeline
> lit desormais l indice DANS LE FILM. (b) La reponse << Alors la largeur juste ne sert a
> rien ? >>, qui s appuyait sur le **+31,2 % de kills attribues** : cette quantite reste ce que
> 7ter.78 disait deja d elle — **elle ne prouve rien**. Le signe est desormais etabli par une
> REFERENCE INDEPENDANTE (`killsource`) : **22.268 %** d accord pour l ordre de la base,
> **22.658 %** pour une permutation au hasard, **76.384 %** pour l indice du film, 116 films sur
> 116. **Ne plus citer le +31,2 %.** Tout le reste de la section tient.

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Quelle est la largeur du champ d indice du joueur dans le fire-event ? | **5 bits, a `eventStart+31`** (le nibble haut de `b5` PLUS le bit qui le precede) — et non 4. Preuve **100 % offline** sur 949 films, population exacte de `ScanFireEventsB5` : sous 17 joueurs le bit emprunte vaut 1 **une fois sur 1 672 653** ; au-dela il vaut 1 sur **31,7 %** des 542 992 events, la lecture 4 bits SATURE a 15 (couverture 0,615 des indices du roster) quand la 5 bits atteint 26 (couverture 0,917). L image ne sort JAMAIS de `[0, roster-1]`. | 7ter.78 (1) · MESURE |
| Que coutait l ancienne lecture ? | **Tout participant d indice >= 16 decodait ZERO** — 323 observations, 90 722 tirs API, 106 films. | 7ter.78 · MESURE |
| Le correctif ameliore-t-il l attribution d arme en production ? | **NON, et c est le resultat qu il ne faut pas embellir.** A 8 joueurs : **72 077 kills, ZERO difference**, 0/823 films touches. Au-dela de 16 : 1 757 armes gagnees pour 1 082 perdues, et le nombre de kills attribues par fire-event **BAISSE** (6 126 -> 6 121). Cause : `getXuidToPI` derive l indice du tueur de l **ORDRE DB**, pas du film. | 7ter.78 (2) · MESURE |
| Alors la largeur juste ne sert a rien ? | Si — **mais seulement avec un indice de tueur juste**. Rejoue avec la resolution bit-level du xuid : **4 065 gagnees / 376 perdues, +31,2 % de kills attribues par fire-event** (10,8 gains pour 1 perte, contre 1,6 avec l ordre DB). ⚠ **CE CHIFFRE N EST PAS UNE PREUVE** : mesure sur des kills, sans verite terrain, et son objectif est du meme type que la quantite optimisee. | 7ter.78 (2) · MESURE, RESERVE |
| `weaponv3` est-il affecte ? | **NON, par construction** : avec son defaut `FireRelax3 = true` il n appelle JAMAIS `analysis.ScanFireEventsB5`. Fige par `TestScanFiresUS_NEmprunteJamaisLaV2ParDefaut`. Il porte deja le meme layout 5 bits (`FirePi5SpanBefore`), actif au-dela de 16 joueurs. Aucun appelant de production. | 7ter.78 (2) · MESURE |
| `weaponv3` mesurait l INVERSE (« SpanBefore regresse les petits matchs »). Qui a tort ? | **PERSONNE — deux POPULATIONS differentes**, mesure sur les memes 60 films : marqueur 11 bits (strict) roster <= 16 -> bit emprunte a 1 sur **0** des 96 333 events, 0 hors roster, max 7 ; marqueur 3 bits + canon (le defaut de `weaponv3`) -> **246 (0,25 %)**, 612 hors roster (0,63 %), **max 31**. Le bit n est du bruit que dans la population RELACHEE. | 7ter.78 (3) · MESURE |
| Ou vit la ventilation des tirs par arme ? | `shared.match_weapon_shots` — grain `match x joueur x arme`, **UNE SEULE colonne de mesure** (`shots_decoded`), append-only, vue `match_weapon_shots_latest` par passe de decodage. Persister INSERT-only + cablage `BatchBuilder`. **AUCUN APPELANT** : personne ne la remplit. | 7ter.78 (4) · MESURE |
| Que refuse cette table de porter ? | **Les touches** (le bit `HitLikely` annonce 75-79 % quand la precision reelle est 27-45 % — ⚠ **CORRIGE en 7ter.80 (7)** : 27-45 % est la bande NORMALE, mediane 0.446, q10 0.321 / q90 0.547 ; c est le 75-79 % qui est faux, pas le 27-45 %. Et un AUTRE bit, `eventStart+106`, porte bien une information de touche pour les armes a tir instantane : §14), **les coefficients par arme** (surajustement : deux fois pire que k = 1), **le nom de l arme** (resolu a la lecture), et **le total du joueur** — `shots_fired` le donne deja et mieux. **LA SEULE INFORMATION NOUVELLE EST LA VENTILATION.** | 7ter.78 (4) · MESURE |
| Ou vit la porte de publication ? | **Dans le persister, en une seule copie** (`EvaluateShotsGate`). Le collecteur fournit la REFERENCE API, pas le verdict : un `publishable = true` sans reference est impossible a ecrire. Le lecteur ne recalcule jamais rien. | 7ter.78 (4) · MESURE |
| `publishable = FALSE` veut-il dire la meme chose que dans `match_kill_events` ? | **NON, ET C EST UN PIEGE.** La-bas : juste en agregat, faux ligne par ligne. **ICI : NE PAS UTILISER** — une ventilation hors tolerance est fausse en agregat aussi, c est le TOTAL qui est faux. | 7ter.78 (4) · doctrine |
| Que laisse passer la porte ? | Tous modes melanges : **55,0 %** des joueurs a <= 16 (ratio mediane **0,968** — la loi k = 1 lue directement) et **20,1 %** au-dela (mediane **0,557** : le grand format sous-decode de moitie en son milieu tout en portant une queue de sur-attribution). ⚠ **PORTEE** : le 84,0 % etabli porte sur le mode standard SEUL. | 7ter.78 (4) · MESURE |
| Un joueur dont rien n est decode est-il visible ? | **NON, limite structurelle assumee** : le verdict vit sur les LIGNES du joueur, donc zero arme = zero ligne = aucun verdict. Le cas « l API annonce des tirs, le decodeur n en trouve aucun » est SILENCIEUX — le compter exige de croiser avec `match_participants`. | 7ter.78 (5) · MESURE |
| Quel defaut reel un test a-t-il trouve ? | Le persister serialisait l identifiant d arme en `int64` : **plus de la moitie du catalogue filmshell a son bit de poids fort a 1** (Fuel Rod SPNKr, Needler, Sidekick, Commando, les deux grenades...) et l INSERT echouait. **Invisible sur un BR75 ou un MA40** — les deux fixtures « evidentes ». Pattern canonique du depot : `ubigintArg` (chaine decimale + `CAST AS UBIGINT`). | 7ter.78 (6) · MESURE |
| Combien coute la passe ? | **Passe de tirs seule** : 1,65 s CPU/film (mediane 1,42, p90 2,49), dont 66 % de lecture/decompression ; **~0,5 s** si les chunks sont deja decompresses, **~1,45 s** sinon -> **+2 % a +18 %** du decodage de source. Volume : **~52 lignes/film**, ~3 Mo pour 1 000 matchs. ⚠ Le 1,90 s median du harnais de mesure du lot mesure L INSTRUMENT (deux correlations + resolution bit-level), pas la passe. | 7ter.78 (7) · MESURE |

### 13.1 LA LECON DE METHODE DE CE LOT

**DEUX MESURES QUI SE CONTREDISENT SUR LE MEME OBJET SE RECONCILIENT D ABORD EN COMPARANT LEURS
POPULATIONS.** `weaponv3` disait « le bit emprunte est du bruit sur les petits matchs », le lot
`tw` disait « il vaut 1 une fois sur 1 672 653 ». Les deux etaient vraies : l une porte sur le
marqueur 3 bits relache, l autre sur le marqueur 11 bits strict. L ecart n etait pas dans le
champ mesure, il etait dans le **marqueur qui selectionne les evenements**.

Corollaire operationnel : le patron A (« l instrument partage une piece avec la reponse ») a un
frere — **deux instruments differents qui mesurent deux choses differentes sous le meme nom**.

### 13.2 CE QUE CE LOT N A PAS FAIT

```
AUCUN APPELANT              la table et son persister compilent et sont testes, rien ne les remplit
`getXuidToPI` NON REMPLACE  PERIME — il l EST depuis 7ter.92 (§20), et le gain a desormais une
                            reference independante (l oracle killsource), pas juste un taux
aucune colonne de touches   REFUS mesure, pas oubli
`analysis.ScanFireEvents`   (mono-joueur) est MORT — seul son propre test l appelle.
                            Decouverte notee, NON traitee (zero fix opportuniste hors perimetre)
verite terrain grand lobby  NON MESUREE — le BTB n a pas de marge de bijection (7ter.53), donc
                            aucune attribution individuelle n y est publiable de toute facon
```

---

## 14. LE COMPTEUR DE TIR ET LE DRAPEAU DE TOUCHE (2026-07-27, sections 7ter.80 et 7ter.82)

> Chantier `mu` (`cmd/tmp_mun`) puis `mu.ref` (`cmd/tmp_muref`, **binaire separe**, `GOCACHE` /
> `GOTMPDIR` dedies) — c est ce second lot qui reproduit, et le gate de reproduction est joue AVANT
> toute conclusion. **Portee de TOUS les chiffres de cette section** : 392 films de
> **mode STANDARD**, 980 163 fire-events, 3 129 joueurs x match. Rien n y porte sur le Fiesta, le
> BTB ni le grand format.
>
> ⚠ **LE DRAPEAU DE TOUCHE A ETE REPRIS EN 7ter.82** (label `rt.rep`, `cmd/tmp_rtrep`, instrument
> ecrit de ZERO, partage SHA-256 au lieu de la parite de rang). **Son statut a bouge dans les deux
> sens** : il monte a `[ETABLI]` pour le mode standard et les armes a trace instantanee (l egalite
> exacte bat un taux constant, McNemar z = 5.08 a 6.27), et il DESCEND sur deux points — l enonce
> « il echoue exactement sur les projectiles » est `[REFUTE]` (Mangler et Disruptor le falsifient)
> et **en FIESTA il est BATTU par un taux constant** (z = -2.83). Les lignes de §14.2 font foi.

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Le fire-event porte-t-il un compteur de tirs ? | **OUI — 7 bits a `eventStart+22..+28`**, valeurs 0..127, cyclique modulo 128, **par JOUEUR** (pas par arme). P(pas = +1) = **0.9738** sur 977 033 paires ; 0.9481 par (joueur, arme). Controle d alignement : **0.5037** a -1 bit, **0.0023** a +1 bit. | 7ter.80 (1) · `[ETABLI]` |
| Le champ `FireCounter` de la production lit-il ce compteur ? | **NON — c est une lecture MASSACREE du bon champ.** Il lit `+24..+31` : il **rogne les 2 bits de tete et avale 3 bits etrangers**. Il avance de +1 **zero fois** (0.0000). `FireSeq` (+8) : 0.0002. **Le bon champ etait a deux bits de distance depuis le portage.** | 7ter.80 (1) · `[ETABLI]` |
| Comment sait-on que les sauts du compteur sont de VRAIS tirs ? | **Second instrument, independant** : au pas = 2 l ecart de temps **double exactement** — MA40 83.4 -> **166.8 ms** (rapport **2.00**, n = 17 113, dispersion 0.01), Needler idem. Et les **REFUS** tombent ou il faut : MA40 pas = 3 -> rapport **0.80**, Sidekick pas = 2 -> **0.82** (donc pas des tirs manquants). ⚠ **STATUT EN DEUX MORCEAUX (`jr.v`)** : la CONJONCTION « ces sauts sont de vrais tirs » est `[ETABLI]` (deux instruments sans piece commune) ; les CHIFFRES d horloge eux-memes sortent d **un seul binaire** et ne sont que `[MESURE]`. | 7ter.80 (2) · `[ETABLI]` conjonction / `[MESURE]` chiffres |
| Peut-on recuperer les tirs perdus des armes automatiques ? | **OUI**, en comptant un tir de plus quand le compteur saute de +2 ET que l ecart de temps vaut 2x la cadence. Hors echantillon : joueurs a >= 60 % d arme automatique **0.974 -> 1.000** en mediane, et **3 a 7 fois plus de joueurs egalent l API A L UNITE** (90-100 % : 2 -> 13 ; 60-90 % : 12 -> 49), **sans toucher** ceux qui n en tirent aucune (75/239 -> 75/239). **Portee : mode standard, MA40 AR et Needler** — les deux seules armes a cycler a 83.4 ms. | 7ter.80 (4) · `[ETABLI]` `[HORS ECHANTILLON]` |
| Comment sait-on que ce n est pas un simple ajustement ? | **LE TEST QUI SEPARE UNE CAUSE D UN FACTEUR D ECHELLE.** Un taux constant reproduit la MEDIANE (0.9700 -> 1.0020) — **la mediane ne prouve donc RIEN**. Sur l EGALITE EXACTE : l echelle rend **3** la ou le compteur rend **13**, n apporte **aucun** gain sur la bande 60-90 % (11 contre 12 de base) et **DEGRADE** la bande 0-60 % (89 -> **53**) la ou le compteur ameliore (89 -> **124**). | 7ter.80 (4)(10) · `[ETABLI]` |
| Ou est le tir manquant ? | **NON TRANCHE, et c est a redire d un cran.** Les instruments etablissent qu un tir A EU LIEU et qu il **n est pas dans NOTRE DECODAGE** — pas qu il n est pas dans le film. Elimine : il n est pas decode sous un autre indice de joueur (0.0319 contre 0.0286 attendus par hasard, exces 0.33 %). ⚠ **LA PISTE DES MARQUEURS REJETES EST DESORMAIS FERMEE** — cf. ligne suivante. | 7ter.80 (5) · `[REFUTE]` |
| L enregistrement de tir COMPACT se cache-t-il dans les marqueurs 11 bits REJETES ? | **NON — `[REFUTE]`, en volume ET en contenu.** L exces de densite (0.2409 -> 0.2527/kbit) **se reproduit** mais il est **entierement un effet de LONGUEUR** : apres appariement a effectif egal il tombe de **+3.9447** a **+0.0829** (bin 512) puis **-0.0388** (bin 128), et dans la bande d ecart de temps **120-250 ms** — la population R3, **89.4 %** des paires sautees — il y a **0.529 motif de MOINS** au pas 2 (5.839 contre 6.368). En contenu : la cible `delta=+1` est en **deficit** (z = -4.01) ; les 161 offsets balayes ne rendent aucun exces qui survive a la stratification ; le profil de bits des rejetes sautes est **indiscernable** des autres (**0.0039** d ecart, quand le controle positif du meme instrument rend **0.2602**). | **7ter.93 (1)-(7)** · `[MESURE]` |
| Un compteur de MUNITIONS existe-t-il pres du fire-event ? | **NON.** 3 681 candidats balayes (offsets -64..+352, largeurs 4..12), meilleur P(pas = -1) = **0.1305** = le bruit. **PISTE MORTE.** | 7ter.80 (11) · `[MESURE]` |
| Quelle est la precision REELLE d un joueur ? | q10 **0.321** · q25 0.382 · **MEDIANE 0.446** · q75 0.501 · q90 **0.547** ; moyenne 0.4394, ecart absolu moyen 16.1 % en relatif (3 129 joueurs). ⚠ **CORRECTION** : la bande **27-45 %**, presentee ailleurs comme une invraisemblance, **est l ordre de grandeur NORMAL**. | 7ter.80 (7) · `[MESURE]` |
| Un compte de touches est-il atteignable ? | **OUI, POUR LES ARMES A TRACE INSTANTANEE, EN MODE STANDARD — bit a `eventStart+106`, polarite 0.** Balayage aveugle de 176 champs d un bit x 2 polarites. Erreur relative sur 1 556 joueurs : taux constant **0.1900** · melange d armes **0.2048** · **le bit 0.1070**. Alignement +105 -> 0.9155, +107 -> 0.4833. **Controle INTRA-ARME** : MA40 0.2083 -> **0.0919**, BR75 0.3296 -> **0.0966**, Sidekick 0.1345 -> **0.0947**, Bandit 0.1320 -> **0.0676**. **REPRODUIT PAR UN SECOND INSTRUMENT** (7ter.82, cf. §14.2). | 7ter.80 (8) + 7ter.82 · `[ETABLI]` `[HORS ECHANTILLON]` |
| Ce bit est-il un compteur de touches universel ? | **NON — `[REFUTE]`.** Il vaut **ZERO** sur les armes a projectile lent : Needler **0.0075**, Pulse Carbine **0.0075**, Bulldog **0.0000**, roquettes <= 0.012. ⚠ **MAIS LA FRONTIERE N EST PAS hitscan/projectile** : 7ter.82 la falsifie — le **Mangler (0.3231)** et le **Disruptor (0.3314)**, deux armes a projectile, se lisent comme des hitscan. L explication par le degat resolu a l IMPACT (§15) reste `[PLAUSIBLE]`, elle n est pas testee. | 7ter.80 (9) + 7ter.82 (5) · `[REFUTE]` |
| Le champ `HitLikely` est-il ressuscite ? | **NON, il reste mort** (75-79 % annonces contre 0.446 reels) — ce n est pas le meme champ. **RESERVE LEVEE PAR 7ter.82 (7)** : le lot anterieur qui balayait **656** positions sur **8 joueurs** n avait pas tort, il etait AVEUGLE — refait a cette taille, un balayage ne DESIGNE `+106` que sur **42.3 %** des films (rang > 30 sur 25/196), alors que le drapeau y bat deja le taux constant sur **172/196** films. 656 candidats pour 8 observations : le vainqueur d un film ne se reproduit pas sur le suivant. | 7ter.80 (8)(9) + 7ter.82 (7) · `[MESURE]` |

### 14.2 LA REPRODUCTION DU DRAPEAU DE TOUCHE (2026-07-27, section 7ter.82, label `rt.rep`)

> Instrument `cmd/tmp_rtrep`, **ecrit de zero, zero fichier partage** avec `tmp_mun` / `tmp_muref` ;
> partage hors echantillon par **SHA-256 du `match_id`** (et non la parite du rang) ; temoin de
> fidelite du localisateur joue AVANT toute mesure (2 663 events, **0 ecart** contre la boucle naive
> de `ScanFireEventsB5`). Corpus : 392 films standard, 3 102 observations, A = 1 555 / B = 1 547.

| QUESTION | REPONSE | STATUT |
|---|---|---|
| Les chiffres du lot d origine ressortent-ils ? | **OUI.** taux constant 0.1900 -> **0.1855** · bit +106 0.1070 -> **0.1122** · intra-arme MA40 0.0919 -> **0.1007**, BR75 0.0966 -> **0.0853**, Sidekick 0.0947 -> **0.0809**, Bandit 0.0676 -> **0.0381**. Alignement : un bit de decalage detruit le champ dans les deux mesures. | `[ETABLI]` |
| **LE CRITERE DU PATRON E** : le drapeau bat-il un taux constant sur l EGALITE EXACTE ? | **OUI.** A denominateur commun (`shots_fired` de l API pour les deux, seul le TAUX change) : **82 exacts contre 29** sur 1 547 (McNemar 81 contre 28, **z = 5.08**) ; hitscan >= 90 % **79 contre 23** (z = 5.60) ; sur les joueurs dont le decodage de TIRS est deja parfait, **20/162 = 12.35 % contre 5/162** (z = 3.00). Survit a l inversion A<->B : **86 contre 23, z = 6.27**. | `[MESURE]` |
| Le critere est-il atteignable (patron D) ? | **OUI** : le compte de TIRS decode egale `shots_fired` a l unite pour **12.0 %** des joueurs. Ce n est pas un critere vide. | `[MESURE]` |
| Faut-il un facteur d echelle ? | **NON** : le calage appris sur A vaut **0.9990** (1.0037 en sens inverse). **La part du bit EST le taux de touche.** | `[ETABLI]` |
| `+106` est-il singulier, ou trente positions font-elles aussi bien ? | **Singulier : 1er sur 297 candidats** sur l egalite exacte (60 contre 34 au 2e) ; seuls **4/297** depassent la baseline. ⚠ **RESERVE** : `+111` pol 1 (errRel 0.1449) et `+124` pol 1 (0.1360) portent aussi de l information — ce n est peut-etre pas un bit isole mais le meilleur bit d un CHAMP. **Non investigue.** | `[MESURE]` |
| Le bit porte-t-il une information PAR JOUEUR ? | **OUI, trois fois.** Permutation intra-film : **0/200** tirages aussi bons. Correlation part-du-bit / precision REELLE **a arme fixee** : BR75 **r = 0.941** (n = 320), MA40 0.875, Bandit Evo 0.978, Sidekick 0.773 ; toutes armes 0.781, hitscan 0.871. Et la part moyenne egale la precision moyenne sans ajustement (BR75 0.4014 / 0.4034). | `[MESURE]` |
| Depend-il d autre chose que la touche ? | **NON** : part par quart de match 0.4374 / 0.4384 / 0.4353 / 0.4317 ; correlation avec le volume de tir **r = 0.101**. | `[MESURE]` |
| **QUELLE EST SA PORTEE ?** | **LE MODE STANDARD, ET RIEN D AUTRE.** Rejoue sur **272 films de FIESTA** : le drapeau est **BATTU** par un taux constant (3 exacts contre 15, **z = -2.83**). Cause mesuree : le **LOCALISATEUR de tirs** s effondre sur ce mode (1 038/1 121 joueurs manquent >= 6 tirs, egalite exacte sur les TIRS **1.1 %** contre 12.0 %, echelle 4.87 contre 1.02). **Ce n est pas le drapeau qui echoue, c est la population d events.** | `[MESURE]` |
| L echec Fiesta s explique-t-il par les projectiles ? | **NON.** Un predicteur qui ne lit le drapeau que sur les events HITSCAN et rend le reste a un taux constant ne recupere **RIEN** en Fiesta (errRel 0.4076 contre 0.4296, McNemar **z = 0.00**) — alors qu il AMELIORE en standard (91 exacts contre 29, z = 5.66). Controle avec degre de liberte, qui rend zero. | `[MESURE]` |

**LA PREDICTION QUI EST TOMBEE** — ecrite avant la mesure, et falsifiee : *le drapeau marche
exactement sur les armes a trace instantanee et vaut ~0 sur les armes a projectile, sans exception.*
La premiere moitie tient (**9 armes hitscan sur 9** entre 0.37 et 0.53) ; la seconde est FAUSSE
(**Mangler 0.3231, Disruptor 0.3314**, et trois intermediaires : Skewer 0.1288, Heatwave 0.1168,
Plasma Pistol 0.0897). Le taux de touche par arme, estime par moindres carres, confirme que le
drapeau est aveugle au Needler (0.0075 pour un taux reel de 0.3881), a la Pulse Carbine, au Bulldog
(0.0000 / 0.3792), aux roquettes. **Ne pas repeindre la cible** : l hypothese de remplacement (le
degat resolu dans le MEME TICK que le tir) est `[PLAUSIBLE]` et n a pas ete testee — il faudrait une
vitesse de projectile ou une distance de tir.

### 14.1 LE CHIFFRE A NE PLUS CITER, ET POURQUOI

**« MA40 0.928 · Sidekick 0.925 · BR75 0.981 · Bandit 1.012 (n = 275/80/313/55) »** est
**IRREPRODUCTIBLE** : ces valeurs n existaient que comme **assertion** dans
`.ai/GUIDE_WEAPON_SHOTS.md` §3 point 3 — aucune section du RE_LOG, aucun outil, aucune methode.
Elles ont declenche une chasse entiere. Deux origines testees : indice joueur sur 4 bits
(**REFUTE**, ratios 1.94 a 2.15) ; numerateur restreint a l arme dominante avec denominateur total
(**PARTIEL** : reproduit MA40 a 0.924, pas les trois autres).

Mesure courante, mode standard, ratio par joueur : **MA40 0.971 · Sidekick 1.004 · BR75 1.007 ·
Bandit Evo 1.007**. Le **Sidekick n est meme pas une arme automatique** (183.8 ms, sauts a rapport
d horloge 0.82).

> ⚠ **CONTRADICTION OUVERTE ENTRE §14 ET §15 — NE PAS L EFFACER.** Sur une AUTRE population (949
> films tous modes, joueurs a arme dominante >= 85 %, agregat de tirs et non ratio par joueur), le
> lot `gt.film` mesure `k_fire = 0.935` pour le **Sidekick** contre 0.977 pour le BR75 — soit le
> deficit que §14 ne retrouve pas. **Populations et estimateurs differents ; la reconciliation
> N EST PAS FAITE.** Survit aux deux : **le MA40 a un deficit, plus grand que celui du BR75**
> (`[ETABLI]`). **CONTESTE** : l existence d un deficit sur le Sidekick — a ne publier ni comme
> present ni comme absent. Traitement : §13.1 (comparer les POPULATIONS d abord).

---

## 15. CE QUE LE JEU COMPTE COMME UN TIR ET COMME UNE TOUCHE (2026-07-27, section 7ter.81)

> Chantiers `gt.dis` (desassemblage) et `gt.film` (confrontation au film), **verifies par `gt.ref`**
> (20 fonctions re-decompilees, six modes re-executes sur 949 films, cinq controles neufs).
>
> **LA DISTINCTION QUI COMMANDE TOUS LES STATUTS** : la chaine d appels est fermee **DANS
> L EXECUTABLE**. Que ces compteurs soient **PRESENTS DANS LE FILM** n est **PAS** mesure.
>
> **PORTEE DES MESURES DE FILM DE CETTE SECTION** : **949 films, TOUS MODES** (Fiesta et BTB
> inclus), population propre `roster <= 16 sans fantome` = **4 422** observations, dont **2 551**
> a part de tirs a projectile < 5 %. **Ce n est PAS le corpus de §14** (392 films de mode
> STANDARD) : aucun chiffre des deux sections ne se compare sans repasser par sa population — c est
> l origine de la contradiction Sidekick. ⚠ **RESERVE SUR LE DESASSEMBLAGE** : les deux agents ont
> lu **le meme binaire avec le meme decompilateur** ; seule la relecture en ASSEMBLEUR et le
> controle croise d offsets sortent de ce referentiel. (7ter.81 (0)(0bis)(1))

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Que compte le jeu comme UN TIR ? | **UN OBJET-PROJECTILE CREE.** `FUN_1408df45c` fait `*(comp + 8 + slot*0xa8) += 1` ; **un seul appelant** (`FUN_1408e1bdc`), lui-meme atteint par **exactement trois sites, tous des creations d objet-projectile**. Ce n est PAS un appui sur la detente : **une salve de fusil a pompe compte autant de tirs que de plombs**. | 7ter.81 (1a) · `[ETABLI]` dans l .exe |
| Que compte le jeu comme UNE TOUCHE ? | **LA PREMIERE APPLICATION DE DEGAT D UN OBJET-SOURCE, POUR TOUTE SA VIE.** `FUN_1408df6a4`, **deux appelants seulement** : `FUN_1404d8600` (application du degat, groupe **`jpt!`**) et `FUN_142f1c44c` (**impact de projectile**, groupe `proj`). Les deux portent la meme **porte de deduplication** (`FUN_140f052cc`, liste `+0x50068`, cardinal `+0x50060`), et l identifiant n en sort **qu a la DESTRUCTION de l objet** (`FUN_140701364`). **Une roquette qui blesse trois joueurs = UNE touche ; bouclier puis sante = UNE touche ; un degat continu = UNE touche pour toute sa vie.** | 7ter.81 (1b) · `[ETABLI]` dans l .exe |
| Les grenades sont-elles comptees ? | **OUI, dans `ShotsFired`, mais dans LEUR PROPRE case** : `FUN_14067009c` fait `*(comp + 0x200 + g*0xa8) += 1`, et `0x200 = 8 + 3*0xA8` = `entries[3+g]`. Espace d index ferme : entrees **0..3** = les 4 emplacements d arme (`unit+0x764`), entrees **4..9** = les 6 grenades (`FUN_1405ccfb8` : `idx < 6 -> idx+1`). Borne juste : `comp+0x698 - comp+0x08 = 0x690 = 10 x 0xA8`. | 7ter.81 (1c) · `[ETABLI]` dans l .exe |
| La melee est-elle comptee ? | **NON** : aucun chemin de melee n atteint `FUN_1408e1bdc`, et la porte des touches exige un objet-source de type projectile. Ses stats vivent dans des champs Bond separes. ⚠ **C est une ABSENCE, corroboree par une absence dans le film** (1.463 % du flux sans identifiant d arme, aucune population melee) : **jamais teste POSITIVEMENT**. | 7ter.81 (1d) · `[MESURE]` par la negative |
| Qu est-ce que `RoundsCorrected` ? | **Un terme de RECONCILIATION PAR LES MUNITIONS** : `FUN_1410af034` ajoute `(short)(dMag + dReserve)` a `entry+0x08`, appele par tick avec le delta de munitions mesure avant/apres. **Le jeu sait lui-meme que son compteur de tirs perd des coups, et il le rattrape.** ⚠ Si l API additionne `TotalShotsFired` **et** cette correction, une part du deficit des automatiques serait **du cote de l API** — NON TESTE. | 7ter.81 (1f) · `[MESURE]` (seul maillon non re-decompile) |
| Les compteurs sont-ils pousses vers la replication ? | **OUI, dans l executable** : `ShotsFired++ -> FUN_142bb2d80 -> stat 0x0D` · `ShotsLanded++ -> FUN_142bb2e28 -> stat 0x0E` · `FUN_142bb7074` **recalcule 0x0F (la precision) a partir de 0x0E et 0x0D**. Chemin : `FUN_142b63d98(engine+0xDF77C, ...)` -> `FUN_142b995b8` -> `FUN_142bad97c` : `*(world + statIdx*0x88 + lineIdx*0x1DF0 + 0x38 + round*4) += delta` — **le layout de statline dont le deserialiseur (`FUN_140c18794`) et l apply (`FUN_140807ebc`) sont deja connus**. DEUX lignes par tir : joueur et equipe. | 7ter.81 (3) · `[ETABLI]` dans l .exe |
| Cette statline est-elle DANS LE FILM ? | ⚠ **QUESTION TRANCHEE PAR 7ter.83 — ET LA REPONSE EST NON POUR LES TROIS GRANDEURS VOULUES.** La statline **est** dans le film (score, kills et morts s y lisent A L UNITE, §16), mais `0x0D` (tirs), `0x0E` (touches) et `0x0F` (precision) **n y voyagent pas** : balayage exhaustif, 4/23 · 4/23 · 7/23 contre un taux de fond de 3/23 · 4/23 · 7/23. La reserve GRENADE / DEBORDEMENT etait la bonne intuition — le resultat est plus radical qu elle. | 7ter.81 (3)(11) puis **7ter.83 (1)** · `[MESURE]` |
| Un enregistrement code-36 (`0xd2`) est-il une touche ? | **NON — c est un enregistrement de TIR**, et c est son tableau A (les applications de degat) qui dit s il a porte. `enregistrements / tirs decodes` : mediane **0.861**, et **correlation NULLE avec la precision du joueur** (`r = -0.002`, n = 6197 ; -0.055 au passage precedent). | 7ter.81 (5) · `[MESURE]` |
| Que peut-on publier a partir de ce flux ? | **UN ESTIMATEUR DE PRECISION PAR JOUEUR, PAS UN COMPTE.** ⚠ **STATUT EN DEUX MORCEAUX (`jr.v`)** : (a) **« sur la population quasi 100 % hitscan le taux du film egale EN MOYENNE celui de l API » = `[ETABLI]`** — reproduit par DEUX agents et DEUX binaires : `gt.film` n = 2 494, ecart **+0.0077** ; `gt.ref` n = 2 551, ecart **+0.008** ; (b) **le pouvoir DISCRIMINANT par joueur (MAE 0.0329, mediane 0.0226, `r = 0.863`, pente 0.839) = `[MESURE]`** — il sort du **SEUL** binaire de `gt.ref`, avec des controles negatifs forts mais **jamais rejoues**. Hors de cette population l estimateur sous-estime jusqu a **-0.20**. | 7ter.81 (5) · `[ETABLI]` moyenne / `[MESURE]` discrimination |
| Et en compte absolu ? | **NON PUBLIABLE** : `porteurs / shots_hit` mediane **0.65** (p25 0.26, p75 0.84). Le film ne capte que deux tiers des enregistrements. | 7ter.81 (5) · `[REFUTE]` comme compte |
| Le film bat-il vraiment un taux constant ? | **Brut, d un facteur 1.21 seulement** — moins que le facteur 2 qui avait suffi a condamner `HitLikely`. Calibre : 1.83, et le gain survit hors echantillon (A->B et B->A). **Ce sont trois controles neufs qui tranchent, pas le facteur** : permutation intra-film **0 tirage sur 200** au moins aussi bon que le reel ; correlation des ecarts **a la moyenne du match r = 0.626** ; et le predicteur de MELANGE D ARMES s effondre a **r = 0.065** la ou le film fait **0.863** — **le film mesure les touches du joueur, pas quelles armes il a portees**. | 7ter.81 (6) · `[MESURE]` `[HORS ECHANTILLON]` |
| L ancrage xuid -> indice est-il sain ? | **OUI, mesure** : 89.5 % des 10 159 joueurs portent exactement DEUX valeurs de 5 bits ; **936 films sur 949 ont exactement UN joueur a singleton `{0}`** ; les 11 films ecartes sont EXPLIQUES. Controle de position : vrai champ dans `[0, roster-1]` a **0.9997** (1 828 368 lectures) contre **0.4287 / 0.2287 / 0.2858** decale de +2 / +7 / +13 bits. | 7ter.81 (7) · `[MESURE]` / `[ETABLI]` (position) |
| Quelle est la limite dure de cet ancrage ? | **LE CONTROLE NEGATIF `shots_fired = 0` ECHOUE** : 248 participants a 0 tir API, dont **81 seulement (32.7 %) muets** ; fuites jusqu a **724 enregistrements** pour un joueur qui n a pas tire. Cause : des **participants fantomes** (inscrits n ayant pas joue, sur **312 films sur 949**) volent l indice d un vrai joueur. Apres nettoyage : 8 zeros exacts sur 13 — **62 %, pas 100 %**. **Cette reserve accompagne tout livrable.** | 7ter.81 (7) · `[MESURE]` |
| Ou est la touche de PROJECTILE ? | **TROUVEE : ce sont les evenements de CODE 6 (impact sur la geometrie) et CODE 7 (impact sur une entite)** — 80 886 et 129 390 sur 949 films, dont **80 % en PREMIER evenement de leur paquet**, et le volume est LOCALISE sur les armes a projectile (`r(code7, tirs projectile) = +0.7675` contre `-0.1929` en hitscan ; 39 des 65 films sans tir de projectile portent EXACTEMENT zero code 7). ⚠ **CE QUI EST ECRIT CI-DESSOUS ETAIT L ETAT AU 2026-07-27 ET DEUX DE SES CONCLUSIONS SONT FAUSSES** : *(a)* << le premier event des paquets type-0 est ELIMINE >> — non, la reponse y etait ; *(b)* << A CHERCHER aux events NON PREMIERS de la boucle >> — mauvaise direction, elle a coute un lot entier. Ce qui reste vrai : **PAS dans le flux code-36** (Bulldog **0 porteur sur 6 833**, Mutilator 0/1 984, Needler 0.007, Cindershot 0.003 ; 1.463 % sans attaquant, Needler 4 porteurs sur 1 345), et le volume attendu ~**58 000** touches sur 949 films. La cause de l erreur : un `topMap(m, 9)` a tronque l histogramme au-dessus des rangs 11 et 14 ou dormaient les codes 7 et 6 (PATRON F, §4). | 7ter.86 (3)(4) `[MESURE]` — corrige 7ter.81 (8) et le titre de 7ter.84 |
| Le deficit des automatiques est-il explique ? | **NON**, mais **deux causes sont MORTES** : ce n est pas la semantique de comptage (**site d increment unique**, `[ETABLI]`) et ce n est pas le marqueur de fire-event (**le deficit se reproduit sur un SECOND compteur lu par une autre grammaire** : `k_enr` MA40 0.801 contre BR75 0.842, **meme contraste +0.041** que `k_fire`). **Candidat nomme, non teste** : la branche de **DEBORDEMENT** `FUN_1408df4f4` incremente `ShotsFired` **sans appeler le pont de replication** — un tir compte par le jeu et jamais replique. | 7ter.81 (9) · `[MESURE]` / `[NON VERIFIE]` |

### 15.1 CE QUI EST TOMBE DANS CES DEUX LOTS — sept enonces, dont les deux presentes comme les preuves

```
0x694 / 0xA8 = 10 EXACTEMENT ..... FAUX (= 10.024). Conclusion vraie, PREUVE fausse.
                                   La borne juste : comp+0x698 - comp+0x08 = 0x690 = 10 x 0xA8
la dedup du desassemblage a
tranche dans le film ............. FAUX : elle est INERTE (118 ecarts sur 655 172), et la
                                   multiplicite 1.000 est IMPOSEE PAR LE FORMAT -> controle vide
0.3648 contre 0.3664, zero
parametre ajuste ................. AGREGAT PONDERE. Joueur par joueur l estimateur est BIAISE
                                   (pente 0.568). L enonce honnete est celui des hitscan (0.839)
rapport uniforme ~0.80 ........... FAUX : 0.609 a 0.853 sur 24 armes (6 avaient ete montrees)
FUN_142bb2d80(_, participant) .... le 2e argument N EST PAS le participant
bloc B disjoint du bloc A ........ FAUX : la chaine ShotsFired est partagee par les deux familles
liste des 10 noms du bloc A ...... FAUSSE sur 4 entrees (PlayerSeqId / GameplaySeqId n y sont pas)
```

**Lecon (PATRON D, §4)** : ces deux controles ont ete acceptes parce qu ils pointaient dans le sens
du resultat, alors qu **aucun des deux ne pouvait rendre autre chose**.

---

## 16. LES STATLINES DU FILM — LE RESULTAT NEGATIF QUI FERME LA VOIE DIRECTE, ET LE GAIN LATERAL (2026-07-27, section 7ter.83)

> Chantiers `sl.read` (`cmd/tmp_slread`) puis `sl.ref` (`cmd/tmp_slref`, durcissement de la
> distribution nulle + enumeration ECS). **PORTEE DE TOUS LES CHIFFRES DE CETTE SECTION** : les
> **4 films de reference**, **23 lignes de joueur appariees** a `match_participants`. Ce n est ni le
> corpus de §14 (392 films standard) ni celui de §15 (949 films tous modes) : aucun chiffre ne se
> compare a ceux de ces sections sans repasser par sa population.
>
> **CETTE SECTION REPOND A LA QUESTION QUE §15 LAISSAIT `[NON VERIFIE]`** (« cette statline est-elle
> DANS LE FILM ? »). La reponse est en deux temps, et c est le premier qui compte : **la statline
> est bien dans le film, mais les tirs et les touches n y sont pas.**

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| **Les compteurs de TIRS et de TOUCHES sont-ils dans les statlines du film ?** | **NON. PISTE MORTE, NE PAS ROUVRIR.** Balayage **EXHAUSTIF** (toutes les positions du bloc, largeurs **4 a 32**, puis **4 096 bits** de charge utile), critere = **EGALITE EXACTE A L UNITE** avec l API : `shots_fired` **4/23**, `shots_hit` **4/23**, precision **7/23**. **Taux de fond par appariement PERMUTE : 3/23 · 4/23 · 7/23** — le MEME NIVEAU. | 7ter.83 (1) · `[MESURE]` |
| Comment sait-on que l instrument n est pas simplement aveugle ? | **PAR SES CONTROLES POSITIFS, tires du MEME balayage** : `kills` **23/23** et `score` **23/23**, nettement separes de leur propre taux de fond (**9/23** et **4/23**). Le balayage sait chercher ; il ne trouve pas les tirs. | 7ter.83 (1) · `[MESURE]` |
| L argument qui ferme la porte ? | **LE DERNIER KEYFRAME EST L ETAT FINAL.** C est ce qui rend `kills` et `score` exacts a l unite ; un compteur de tirs present dans cette structure y serait donc **aussi exact**, il ne peut pas y etre « approximatif ». ⚠ **C est un RAISONNEMENT sur le format, pas une mesure** — cite comme tel. | 7ter.83 (1) · `[RAISONNEMENT]` |
| Le controle negatif est-il assez fort ? | **PAS DANS SA FORME D ORIGINE** : `sl.read` comparait le meilleur balayage a **UNE seule** permutation. Un maximum sur des milliers de candidats se compare a une **DISTRIBUTION** nulle. Durci par `sl.ref` (mode `sweep`, R permutations, moyenne / max / p-value) : **verdict inchange**. **Ne plus citer la forme naive seule.** | 7ter.83 (1) · `[MESURE]` (patron D) |
| **Un composant REPLIQUE porte-t-il une statistique de tir par arme ?** | **NON — ET C EST UNE ENUMERATION EXHAUSTIVE, pas une recherche infructueuse** (patron E3). Registre `chunk_00` enumere sans ancre : **118 archetypes**, **325 noms de composants**, **ZERO** portant une statistique de tir. | 7ter.83 (2) · `[MESURE]` |
| Que porte la replication cote ARME, alors ? | **L ETAT, jamais l HISTORIQUE** : identite de l arme **par emplacement**, **munitions**, **inventaire de chargeurs**, **surchauffe**. Coherent avec §15 : le jeu compte les tirs dans un composant de **statistiques du joueur** (`*(comp + 8 + slot*0xa8) += 1`), qui n est pas dans le jeu de composants repliques. | 7ter.83 (2) · `[MESURE]` |
| **Consequence de perimetre** | Les **DEUX HEURISTIQUES SONT LES SEULES VOIES** vers les tirs et les touches : compteur de tir 7 bits `eventStart+22` (§14, `[ETABLI]`) et drapeau de touche `eventStart+106` (§14.2, `[ETABLI]` en mode standard sur les armes a trace instantanee). Ce n est pas un repli, c est un perimetre **ferme par deux mesures**. **A dire franchement** : ce sont des estimateurs sur des evenements, pas des compteurs lus — 12.0 % des joueurs seulement egalent l API a l unite sur les tirs. | 7ter.83 (3) · decision |

### 16.1 LE GAIN LATERAL — TROIS GRANDEURS LISIBLES A L UNITE, SANS API

**Ce n est PAS ce que le lot cherchait** : c est le sous-produit du controle positif ci-dessus.

```
grandeur     OFFSET     LARGEUR     EGALITE EXACTE     appariement permute
score          172         12           23 / 23              4 / 23
kills          126          8           23 / 23              9 / 23
morts          208          8           23 / 23              -
```

Trois raisons d y croire : **memes positions sur les 4 films** (aucune re-derivation) ; **HORS
ECHANTILLON** — derivees sur **2** films, verifiees **sans reajustement** sur les **2** autres ;
**controle d alignement discriminant** — **0/23 des qu on decale d un seul bit**.

**A QUOI CA SERT** : du decodage **offline gratuit**, sans aucun appel d API.
1. **Controle de sante d une passe** — confronter score et kills lus dans le film a ce que le
   decodeur de source publie ; un desaccord signale une passe malade **sans reference externe**.
2. **Temoin d ancrage** — l appariement `ligne de statline -> joueur` est une **seconde jambe**,
   independante de la bijection `indice -> joueur` du kill-feed (que le §8.5 rappelle circulaire).
3. **Corpus sans API** — sur un film dont le match n est pas en base, ces trois entiers existent.

**RESERVES** : `morts` n a pas de controle permute publie ; aucune verite terrain Theater (tout est
mesure contre l API) ; **NON MESURE** hors des 4 films de reference (Fiesta, BTB, grand format) et
a travers les versions du jeu. **RIEN N EST BRANCHE** : aucun code de production ne lit ces offsets.

### 16.2 LA LECON DE METHODE DE CE LOT

**UN RESULTAT NEGATIF N EST PUBLIABLE QUE S IL EMBARQUE SON CONTROLE POSITIF.** « Nous n avons pas
trouve X » ne vaut rien ; « le meme instrument, sur le meme bloc, avec le meme critere, trouve Y et
Z a 23/23, et ne trouve pas X au-dessus du taux de fond » ferme la question. C est la forme
operationnelle de **E3** (§4) : *avant d ecrire « le format ne porte pas X », ENUMERER sans ancre* —
et ici, l enumeration a ete faite **deux fois par deux voies** (le fil, puis le registre ECS).

**Corollaire de tenue de journal** : deux agents ont vise `7ter.82` le meme jour. **Le numero libre
se lit dans le fichier (`grep "^### 7ter"`), jamais dans un brief.**

---

## 17. LA PRECISION PAR ARME — CE QUI EST PUBLIABLE, ET POURQUOI L ACCORD NE PROUVE PAS CE QU IL SEMBLE PROUVER (2026-07-28, section 7ter.85)

> Chantier `pv` (`cmd/tmp_pvarme`). **PORTEE** : 949 films caches, **571 retenus** (roster <= 16,
> SANS participant fantome, ancrage unique), **4 422 observations joueur x match**, TOUS MODES —
> c est la population de §15, pas celle de §14. Le decodeur est **RECOPIE de `tmp_gtref`** : aucun
> chiffre de ce lot n a de reproduction par un instrument independant, statut plafonne a `[MESURE]`.
>
> ⚠ **CE TABLEAU A ETE VERIFIE PAR `pv.ref` (7ter.87, `cmd/tmp_pvref`) : TOUS SES CHIFFRES SE
> REPRODUISENT A L UNITE, ET DEUX DE SES CONCLUSIONS SONT CORRIGEES.** Lire **§17.3 AVANT** de
> citer une ligne de ce tableau — en particulier la ligne << biais reproductibles >> et la ligne
> << l accord valide-t-il ? >>, dont l argument (la nulle `P2`) est mesure comme non concluant.

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| Existe-t-il une reference API par arme ? | **NON, et c est structurel** : `shared.match_participants` donne `shots_fired` / `shots_hit` par JOUEUR ET PAR MATCH. Une precision par arme ne se valide **jamais** directement. La seule voie : les joueurs dont les tirs decodes portent essentiellement UNE arme. | 7ter.85 (1) · `[MESURE]` |
| L accord film/API se resserre-t-il avec la purete ? | **OUI** : ecart agrege **+0.0104** (>= 50 %) -> **+0.0103** (>= 80 %) -> **+0.0077** (>= 90 %) -> **+0.0057** (>= 95 %) -> **-0.0074** (= 100 %). Temoin interne : a 100 % `film(dom)` et `film(tout)` sont egaux par construction, et ils le sont (0.3955). | 7ter.85 (1) · `[MESURE]` |
| Le tableau par arme discrimine-t-il ? | **OUI, et le controle positif est net** : ecart **0.011 a 0.077** sur les hitscan contre **0.19 a 0.25** sur Needler / Pulse Carbine / Plasma Pistol — facteur **7 a 20**. Pente de l ajustement a >= 80 % : **1.000**, decalage -0.008 (niveau ET etendue reproduits). | 7ter.85 (2) · `[MESURE]` |
| Ou est le controle positif, exactement ? | **AU CRAN >= 50 % DE PURETE, ET NULLE PART AILLEURS.** La population a arme dominante ne contient presque **aucune** arme a projectile : le Needler y a **122 observations a >= 50 %, 2 a >= 80 %, 0 a >= 90 %**. Propriete du jeu (les armes lourdes se ramassent), pas choix de confort. | 7ter.85 (1) · `[MESURE]` |
| **L accord du tableau par arme valide-t-il la precision par arme ?** | **NON, ET C EST LE RESULTAT CENTRAL DU LOT.** Nulles a 200 tirages, statistique = MAE entre armes, a >= 80 % : contre un fond **SANS AUCUN LIEN** (P0) le reel gagne **0/200** — mais contre une nulle qui garde la structure de match (P1) c est **100/200**, et contre un regroupement des memes joueurs sous des etiquettes d arme AU HASARD (P2) **8/200** (et **197/200** a >= 50 %). **La justesse est HERITEE du resultat par joueur de 7ter.81 (5) : tout regroupement de joueurs fait coincider les deux taux.** ⚠ **LA CONCLUSION TIENT, SON ARGUMENT NON (7ter.87 (4))** : la nulle `P2` est INTRA-FILM et **FIGE 80.2 % des etiquettes a >= 80 %** (92.9 % a >= 95 %) — c est une quasi-identite, ses `8/200` et `27/200` sont reproduits a l unite et ne prouvent rien. Sous une nulle `P2` **GLOBALE**, le hasard fait **MIEUX** que le tableau reel (**175/200**). Et **la MAE confrontee a une permutation d etiquette ne decide dans AUCUN sens** : la nulle detruit la variance entre groupes, donc sa MAE est petite par construction. | 7ter.85 (3) + **7ter.87 (4)** · `[MESURE]` |
| Le `rho` de Spearman film/API par arme prouve-t-il l ordre ? | **NON — `[REFUTE]` comme preuve.** La nulle P1 le reproduit ou le bat : **200/200** a >= 50 % hitscan, 189/200 a >= 95 %. Sur 5 armes il ne prend que quelques valeurs. **Ne plus le citer.** | 7ter.85 (3) · `[REFUTE]` |
| Le film porte-t-il malgre tout un signal PAR ARME ? | **OUI, et le test sort du regroupement** : le CONTRASTE INTRA-JOUEUR (deux armes chez le MEME joueur dans le MEME match) a une etendue de **0.3428** contre une nulle a **0.0392 [0.0119..0.1032]**, **0/200**. 8 armes sur 12 hors enveloppe. Apparies : MA40 - Sidekick **+0.1083** (77 %, z = +16.98, n = 982) · MA40 - Needler **+0.3918** (95 %, z = +12.48). | 7ter.85 (4) · `[MESURE]` |
| Ce signal EST-il une precision ? | **NON, et le Needler le prouve** : son contraste intra-joueur vaut **-0.2085** pour une precision reelle de l ordre de **0.24 a 0.39**. **Le contraste mesure la VISIBILITE de l arme par le decodeur autant que sa precision**, et rien dans le contraste ne dit laquelle des deux on lit (`rho` avec l API **+0.357**, avec le tableau du film **+0.810**). | 7ter.85 (4) · `[MESURE]` |
| Que rend la decomposition par moindres carres (second instrument) ? | Elle fait entrer les armes a projectile : Needler **film 0.0066 contre API 0.2382**, Pulse Carbine **0.0083 contre 0.1533**, Disruptor **0.1867 contre 0.3404**. **RESERVE** : la nulle par permutation est SERREE et PROCHE du reel (nulle du Needler **0.2750**, pas 0.39) — le coefficient n est que faiblement une propriete de l ARME. Et son gain hors echantillon (**21 %** sur une constante) **N EST PAS NEUF** : c est le predicteur de MELANGE D ARMES de 7ter.81 (6), deja mesure a 18 %. | 7ter.85 (5) · `[MESURE]` |
| **Quel chiffre publier ?** | **PAS CELUI DU CORPUS ENTIER.** Trois chiffres coexistent ; ecart absolu moyen a la reference : corpus entier **0.0361** · dominante >= 50 % **0.0209** · pure >= 80 % **0.0144**. **Le chiffre naif est 2.5 x plus faux.** | 7ter.85 (6) · `[MESURE]` |
| **Le prix de l erreur, en clair** | **UNE INVERSION D ORDRE.** Chiffre naif : MA40 **0.4485** contre Sidekick **0.3701** (MA40 +8 points). Reference API : MA40 **0.4196** contre Sidekick **0.4491** (Sidekick +3 points). **Signe faux, ecart exagere de 11 points.** Cause : le Sidekick est une arme de SECOURS, son taux film monte avec sa part (0.370 -> 0.408 -> 0.444 -> **0.484**). **Le taux par arme depend de la population de tireurs.** | 7ter.85 (6) · `[MESURE]` |
| Tient-il hors echantillon ? | **PAS ROBUSTEMENT.** Partage des films par SHA-256 : A -> B ecart absolu moyen **0.0118** contre une constante a 0.0345 (**facteur 2.9**) ; B -> A **0.0240** contre 0.0249 (**facteur 1.0**). Les 4 armes communes s etalent de 0.42 a 0.51 ; le S7 Sniper, seule arme qui elargirait la plage, ne survit pas au partage. | 7ter.85 (6) · `[MESURE]` `[HORS ECHANTILLON]` |
| Quelle est la LISTE publiable ? | **MA40 AR** (443 joueurs, 264 films) et **BR75** (282 joueurs, 149 films) a **+-0.03**. **Sidekick** (49 joueurs mais 12 seulement a >= 95 %) et **Bandit Evo** (54 joueurs, **13 films**) : justes, base mince, a assortir de leur effectif. **S7 Sniper : NON** (11 observations, **2 films**). **Tout le reste : NON PUBLIABLE.** ⚠ **AJOUT 7ter.87 (2)(3)** : ne **JAMAIS publier un ORDRE** entre MA40, BR75 et Sidekick — leur etendue de reference vaut **0.0295** et l erreur du film **0.0136**, soit la moitie de ce qu il y a a distinguer ; et **la restriction a l arme DEGRADE l estimation** (le taux des memes joueurs sur TOUTES leurs armes est plus proche de la reference, a tous les crans). | 7ter.85 (6)(9) + **7ter.87 (2)(3)** · `[MESURE]` |
| Biais reproductibles a connaitre ? | Le film **SURESTIME** le MA40 de **+0.0249 [+0.020..+0.029]** et **SOUS-ESTIME** le BR75 de **-0.0108 [-0.017..-0.003]** (bootstrap 200 tirages sur les films). ⚠ **CORRECTION 7ter.87 (7)(8)** : les valeurs centrales se reproduisent, mais **<< a tous les crans >> est FAUX**. Le biais du BR75 **CONTIENT ZERO a >= 95 %** (-0.0107 [-0.0210..+0.0002]) et sa regression sur (1 - purete) rend une ordonnee **-0.0023 [-0.0103..+0.0073] : c est une CONTAMINATION DE LA REFERENCE, pas une propriete du film**. Le biais du MA40, lui, N EST PAS une contamination (pente dans la nulle, ordonnee +0.0265 excluant zero) — mais il DECROIT avec la purete jusqu a **+0.0082 [+0.0004..+0.0165]** a purete 100 %, hors de l intervalle publie. | 7ter.85 (2) + **7ter.87 (7)(8)** · `[MESURE]` / `[CORRIGE]` |
| Cela clot-il la contradiction Sidekick de §14.1 / §15 ? | **NON, mais cela l eclaire** : une TROISIEME quantite du Sidekick — son taux de TOUCHE — varie de **0.370 a 0.484** selon la seule purete. **La sensibilite a la population est etablie ; la reconciliation du deficit de TIRS ne l est pas.** | 7ter.85 (6) · `[MESURE]` |

### 17.1 CE QUE L UTILISATEUR PROPOSAIT, ET POURQUOI SA FORME NE MARCHE PAS

*<< Un kill au sniper avec tir a la tete prouve une precision de 100 %. >>* Un kill prouve qu **UN**
tir a touche ; une precision est un **TAUX**, et un kill ne porte **aucun denominateur**. Pire : le
kill est un evenement **selectionne par sa reussite** — n echantillonner que des kills rend 100 %
pour toute arme, toujours, y compris pour le Needler dont ce lot mesure que le film ne voit jamais
les touches. **L intuition sous-jacente est JUSTE** (chercher un cas ou l arme est unique) ; c est
sa forme qui ne l est pas : il faut prendre un **MATCH ENTIER** a arme quasi unique, pas un KILL.

### 17.2 LA LECON DE METHODE DE CE LOT — PATRON E, TROISIEME COSTUME

Les deux occurrences precedentes du patron E portaient sur une **statistique agregee** qu un facteur
d echelle reproduisait. Ici la forme est plus retorse : **un resultat DEJA ETABLI a un grain FIN
(par joueur) rend automatiquement vrai le meme resultat a un grain GROSSIER (par arme), quel que
soit le regroupement.** L accord par arme n etait donc pas faux — il etait **VIDE**, et seule une
nulle qui regroupe les memes joueurs AU HASARD (P2) pouvait le montrer.

**REGLE A APPLIQUER : quand on valide une quantite AGREGEE alors qu une validation FINE existe deja
sur les memes objets, la nulle a battre n est pas << aucun lien >> — c est << le meme lien fin,
regroupe au hasard >>.** Sans cette nulle, on republie un resultat ancien sous un nom neuf.

**Corollaire retenu** : le test qui a echappe au piege est celui qui **sort du regroupement** — le
contraste INTRA-joueur. Il a rendu un signal massif (0/200)... et a immediatement montre, par le
Needler, que ce signal n est pas la quantite voulue. **Deux verdicts, deux statuts, dans le meme
paragraphe.**

### 17.3 VERIFICATION ADVERSARIALE PAR UN SECOND BINAIRE (2026-07-28, section 7ter.87, label `pv.ref`)

> Outil : `cmd/tmp_pvref` (modes `gate` `grid` `scale` `ventil` `contam` `stab` `pub` `circ`).
> **SECONDE TRANSCRIPTION, PAS SECOND CRAQUAGE** : la grammaire code-36 et le principe d ancrage
> viennent de 7ter.81 ; sont neufs le lecteur de bits, la marche des paquets, le collecteur et
> **toutes** les analyses. Le statut reste donc plafonne a `[MESURE]` — un `[ETABLI]` exige un
> craquage independant, qui n existe toujours pas.

| QUESTION DE REFUTATION | REPONSE MESUREE | STATUT |
|---|---|---|
| Le tableau par arme se reproduit-il ? | **OUI, 21 armes sur 21 a +-0.0006**, comptes d enregistrements identiques a l unite, et la POPULATION aussi : 571 films retenus, 310 fantomes, 63 roster > 16, 2 ancrages non uniques, 4 422 observations. **0 record tronque sur 1 855 515 paquets** (le lot pv ne testait pas ce cas). | 7ter.87 (1) · `[MESURE]` |
| **La ventilation par arme apporte-t-elle quelque chose ?** | **NON — ELLE DEGRADE.** Remplacer le taux restreint a l arme par le taux des **MEMES joueurs sur TOUTES leurs armes** (`film(tout)`) AMELIORE la prediction a **tous** les crans : gain de la ventilation **-0.0262** (>= 50 %), **-0.0157** (>= 60 %), **-0.0102** (>= 70 %), **-0.0030** (>= 80 %), **-0.0017** (>= 90 %), **-0.0005** (>= 95 %), **0.0000** a 100 % (ou les deux sont egaux par construction). **Ce qui porte l information, c est la SELECTION des joueurs (+0.0339 sur une constante), pas l arme.** La moitie du biais du MA40 est fabriquee par la restriction : `film(tout)` = 0.4316 contre `film(dom)` = 0.4445 pour une reference a 0.4196. | 7ter.87 (2) · `[MESURE]` |
| Le film bat-il une constante quand on retire les armes a base etroite ? | **NON ROBUSTEMENT.** En ne gardant que les armes portees par **>= 30 films** (donc sans Bandit Evo ni S7 Sniper), la constante GAGNE a trois crans sur six (>= 50 %, >= 60 %, >= 70 %) et le film ne gagne qu a >= 80 % (0.0136 contre 0.0155). **Le << gain sur la constante >> de 7ter.85 est porte par 13 films (Bandit Evo) et 2 films (S7 Sniper).** | 7ter.87 (3) · `[MESURE]` |
| La nulle `P2` de 7ter.85 vaut-elle quelque chose ? | **NON A HAUTE PURETE.** Elle permute l etiquette INTRA-FILM ; part d etiquettes INCHANGEES : 66.7 % (>= 50 %), **80.2 % (>= 80 %)**, **92.9 % (>= 95 %)**, 97.1 % (100 %). Ses `8/200` et `27/200` sont **reproduits a l unite** et sont l artefact d une quasi-identite. Nulle `P2` GLOBALE : le hasard fait MIEUX que le reel, **175/200** a >= 80 %. **Et la MAE est une statistique degeneree sous permutation d etiquette** — a ne plus employer dans aucun sens. | 7ter.87 (4) · `[MESURE]` / `[REFUTE]` |
| **La population est-elle circulaire ?** | **NON.** Controle par un SECOND FLUX du film — les **fire-events** (marqueur 11 bits, arme sur **64 bits** a `eventStart+40`) contre le flux de degat (**famille 32 bits**, position variable dans un record type 36). Accord sur l arme dominante : **1 303 / 1 303 = 1.0000 a >= 80 %**, 0.9963 a >= 50 %, 0.9613 global. **PORTEE ETROITE** : ce controle valide l IDENTIFICATION DE L ARME, **PAS l ancrage** — les deux flux partagent la bijection xuid -> indice, un ancrage faux passerait inapercu. | 7ter.87 (5) · `[MESURE]` |
| Le seuil de purete a-t-il ete choisi pour le resultat ? | **NON** : sur les armes hitscan la MAE descend en pente douce (0.0439 a 0.40 -> 0.0126 a 0.95) et **le meilleur cran est 0.95, pas 0.80**. **MAIS** la chute spectaculaire de la MAE globale entre 0.70 (0.0515) et 0.75 (0.0160) **n est pas un gain de justesse** : ce sont les armes a projectile qui quittent le tableau. **Le seuil qui purifie ELIMINE aussi les armes que le film ne sait pas lire** — toute MAE citee a >= 80 % doit porter la mention << sur les seules armes que le film voit >>. | 7ter.87 (6) · `[MESURE]` |
| D ou vient le biais par arme (reserve n.8 de 7ter.85) ? | **CONTAMINATION DE LA REFERENCE pour le BR75 et le Sidekick, PAS pour le MA40.** Regression du residu sur (1 - purete) : BR75 pente **-0.0550** (hors nulle), ordonnee a purete=1 **-0.0023 [-0.0103..+0.0073] CONTIENT ZERO** ; Sidekick pente **-0.1537** (hors nulle), ordonnee **+0.0029** contient zero ; MA40 pente **+0.0032 DANS la nulle**, ordonnee **+0.0265** excluant zero. **TENSION NON TRANCHEE** : la regression extrapole +0.0265 pour le MA40 la ou le cran 100 % MESURE +0.0082 (36 obs). | 7ter.87 (8) · `[MESURE]` / `[NON VERIFIE]` |
| Le taux par arme est-il une propriete de l arme ? | **NON, il depend de la population de tireurs** — sur les memes populations, l amplitude du taux FILM a travers les crans vaut **2.58 x** celle de la reference pour le Sidekick et **2.14 x** pour le MA40. Nuance : ce n est PAS uniforme (BR75 1.11 x). | 7ter.87 (9) · `[MESURE]` |
| Le biais par arme se transporte-t-il ? | **OUI.** Partage des films par SHA-256, biais appris sur une moitie applique a l autre : erreur **0.0070 dans LES DEUX SENS** a >= 80 % (film brut 0.0126 et 0.0150, constante 0.0249 et 0.0345). **Mais** les 4 armes incluent le Bandit Evo (13 films) qui porte l etendue, et **une correction apprise sur l API n est plus un produit offline : c est une calibration.** | 7ter.87 (10) · `[HORS ECHANTILLON]` |
| Reste-t-il un resultat de 7ter.85 non verifie ? | **OUI, et c est le plus important** : le **CONTRASTE INTRA-JOUEUR** (7ter.85 (4)) n a PAS ete rejoue. C est le seul resultat du lot pv qui sorte du regroupement de joueurs — donc le seul que 7ter.87 n a ni confirme ni refute. | 7ter.87 (13) · `[NON VERIFIE]` |

**LA LIMITE STRUCTURELLE, A REECRIRE A CHAQUE FOIS** : `shared.match_participants` ne donne
`shots_fired` / `shots_hit` que **par joueur et par match, jamais par arme**. Une precision par
arme ne peut donc etre validee **QUE** sur les joueurs a arme quasi unique — c est-a-dire sur une
population qui n est pas celle des utilisateurs de l arme, et qui **exclut par construction les
armes qu on ramasse** (Needler : 122 observations a >= 50 %, **2** a >= 80 %, **0** a >= 90 %).

**POURQUOI LES ARMES A PROJECTILE SONT INUTILISABLES, ET CE N EST PAS UN DEFAUT DE MESURE** : un
enregistrement code-36 porte le degat resolu **dans le meme tick que le tir**. Un projectile part,
et son impact arrive plus tard — `ShotsLanded++` a alors pour appelant l IMPACT DE PROJECTILE
(groupe `proj`), qui **n ecrit pas dans l enregistrement du tir** (7ter.81 (1)). Le film ne voit
donc **jamais** ces touches : Needler **0.007** contre une reference a **0.24-0.39**, Bulldog
**0.000**, Cindershot 0.003, Hydra 0.004, SPNKr 0.004 — **faux d un facteur 30 a 60**. Les touches
de projectile existent ailleurs dans le film : ce sont les **codes 6 et 7** (7ter.86), et elles
n ont **pas encore de tireur**.

**LISTE PUBLIABLE APRES VERIFICATION** (population : joueurs dont l arme domine **>= 80 %** de
leurs tirs decodes, roster <= 16, tous modes, 949 films caches, aucune verite terrain Theater) :

```
MA40 AR         PUBLIABLE   +-0.03   443 joueurs / 264 films   biais film +0.0249 (reel, non explique)
BR75            PUBLIABLE   +-0.03   282 joueurs / 149 films   biais film -0.0108 (CONTAMINATION)
Mk51 Sidekick   AVEC EFFECTIF        49 joueurs / 133 films    biais indiscernable de zero
Bandit Evo      AVEC EFFECTIF        54 joueurs / 13 FILMS     base etroite
S7 Sniper       NON                  11 obs / 2 FILMS          IC bootstrap sur 2 unites = IC qui ment
toutes armes a projectile   NON      faux d un facteur 30 a 60
ORDRE entre MA40 / BR75 / Sidekick   NON   etendue 0.0295 pour une erreur 0.0136
taux calcule sur le CORPUS ENTIER    NON   2.5 x plus faux, et il INVERSE MA40 / Sidekick
```

### 17.4 LA LECON DE METHODE DE `pv.ref` — LA NULLE QUI NE PERMUTE RIEN

Le lot `pv` avait ecrit lui-meme l avertissement — *<< a haute purete l intra-film ne permute quasi
rien (2.3 observations par film) >>* — **et ne l avait applique qu a `P0`**, alors qu il vaut
identiquement pour `P2`, la nulle sur laquelle reposait sa conclusion centrale.

**REGLE A APPLIQUER : une nulle par permutation doit publier LA PART DE SES ETIQUETTES QU ELLE
LAISSE EN PLACE.** Sans ce chiffre, un `8/200` peut aussi bien signifier << le reel se distingue >>
que << la nulle est le reel >>. Ici : **80.2 % d identite**.

**Seconde regle, jumelle** : **verifier que la STATISTIQUE choisie peut bouger sous la nulle.** Une
MAE entre groupes confrontee a une permutation d etiquettes est degeneree — la nulle detruit la
variance entre groupes, donc sa MAE est petite quoi qu il arrive. La statistique qui garde un sens
est le **gain sur une constante**, parce qu une nulle sans variance entre groupes rend la constante
parfaite et le gain nul.

---

## 18. VEHICULES — ETAT ET PISTES (2026-07-28)

> **POURQUOI CETTE SECTION EXISTE** : l index parlait des vehicules en une vingtaine d endroits,
> repartis entre §2.5 (le NOMMAGE d une arme de vehicule dans les `.module`) et §2.5bis (le
> vehicule comme ENTITE REPLIQUEE du film), sans page qui dise ou en est le sujet. Les deux
> tableaux restent la source de detail ; ceci est la vue d ensemble et la file de travail.
> Sections du RE_LOG : **7ter.45/46/47/48** (nommage), **7ter.89 (6)(7)(8)(9)** et
> **7ter.90** (film).
> Le lot `vh` qui a defriche l axe film **n a jamais ete publie au journal** : tout ce qui vient
> de lui et que 7ter.89 n a pas rejoue est marque `[NON VERIFIE]` ci-dessous, sans exception.

> ## MISE A JOUR LA PLUS RECENTE — 2026-07-28 — 7ter.102 (lot `vf`). DEUX ENONCES DE CETTE PAGE ETAIENT FAUX, ET IL FAUT LES CORRIGER AVANT DE LIRE LE RESTE.
>
> **1. << LE RATTACHEMENT PAR CONTENANCE EST IMPOSSIBLE, LA PISTE DE L UTILISATEUR EST MORTE >> —
> FAUX, ET C EST L INVERSE.** Ce qui est mesure par 7ter.97 (6), c est que **le film ne transporte
> pas la GEOMETRIE** : sur 325 noms de composants et 24 radicaux, seul `object-scale-component`
> (`i12`) sort sur `ti=40`, et il n est qu un facteur d echelle. Mais le film transporte **la
> POSITION (`i0`)**, **l ORIENTATION (`i2 object-forward-and-up-dynamic-precision-component`)** et
> **l ECHELLE (`i12`)** ; ce qui manque, ce sont les **dimensions statiques du chassis**, qui vivent
> dans les `.module` — **que le depot lit deja** (`internal/himodule/module.go`,
> `internal/ooz/ooz.go`, la chaine de la geometrie 2D des cartes). **LA PISTE EST VIVANTE, ELLE EST
> UNE JOINTURE, ET ELLE NE DEPEND D AUCUN SEUIL** — ce qui la rend structurellement meilleure que la
> correlation de co-mouvement, dont tout le dossier montre qu elle vit et meurt par son seuil.
> **A LIRE COMME LA PISTE 1 DU SUJET** (§18.3).
>
> **2. << LES FRAGS AU VEHICULE SONT LIVRABLES SANS RETRO-INGENIERIE, IL NE MANQUE QU UNE TABLE DE
> 89 LIGNES >> — TROP OPTIMISTE, ET LE CHIFFRE QUI LE CORRIGEAIT ETAIT LUI AUSSI FAUX.** 7ter.96 (3)
> avait ramene la piste 0 a **6 tags nommables sur 14** ; ce compte additionnait deux criteres de
> nature differente — << racine de banque UNIQUE >>, qui se mesure, et << chassis PLAUSIBLE >>, qui
> est **une croyance sur le jeu**. **L utilisateur a corrige la croyance : le Falcon existe en BTB**,
> et `4f77afc1`, le film de la mesure, **EST un BTB** (`BTB:CTF / Flood Gulch`). Le Pelican est
> reexamine pour la meme raison, pas reconduit.
> **CHIFFRES A CITER DESORMAIS : `8 / 14` (57 %)** sur les tags observes, **`53 / 89` (59.6 %)** sur
> le catalogue entier — repartition 9 / 53 / 12 / 13 / 1 / 1 pour 0 / 1 / 2 / 3 / 4 / 6 racines.
> Statut des deux tags : **`[NON VERIFIE]`**, plus << implausible >>.
> ⚠ **NE PAS SUR-CORRIGER** : que le Falcon existe **retire un argument CONTRE**, cela **n ajoute
> aucune preuve POUR** que le tag le designe correctement. La confrontation Theater des trois
> instants (`02:19.035` et `08:29.356` Pelican, `03:40.856` Falcon, film `4f77afc1`) reste due.
>
> **3. << LA BANQUE WWISE EST REUTILISEE ENTRE CHASSIS >> PERD SES DEUX MEILLEURS APPUIS.** Sur la
> population des films elle tombe a **`[PLAUSIBLE]`**, portee par les seuls 4 tags a racines
> multiples. Elle est re-etayee par une mesure de catalogue **qui ne contient aucune croyance** :
> **dix `vehi` distincts citent de DEUX a SIX racines de banque distinctes** (`0000d4ff` cite
> Scorpion, Chopper, Banshee, Wasp et tourelle Shade dans le meme `Detail`). Mais cette mesure
> n autorise que la **forme faible** — *la racine ne determine pas le chassis PAR CETTE CHAINE
> D EXTRACTION*, statut `[MESURE]` — et non la forme forte sur le jeu, qui reste `[PLAUSIBLE]`.
> **Pour la table d etiquettes, la consequence est la meme : publier la disjonction, jamais choisir.**
>
> **4. LA FAUTE DE METHODE A UNE LIGNE A ELLE, ET ELLE EST NEUVE** (§4, apres le PATRON D) : *ne
> jamais declarer un decodage implausible sur une croyance concernant le jeu — la croyance se
> verifie, comme le reste.*
>
> **5. LA LISTE DE FRAGS AU VEHICULE EST RENDUE, LIGNE PAR LIGNE** (7ter.102 (5)) : `78919882`,
> `Community:Slayer / High Ground / 2026-07-23 21:14`, **99 morts sur 99**, marge de bijection
> **35**, zero alerte, **7 frags de classe VEHICULE**, multiset egal a l API a l unite —
> **publiable ligne par ligne**. `4f77afc1` n est cite que pour le VOLUME (**55 sur 225**) : marge
> **NULLE**, donc c est **la LIGNE ENTIERE** qui depend de la bijection, pas seulement ses deux noms.

> ## MISE A JOUR — 2026-07-28 — 7ter.100 (lot `vc.ref`, verification adversariale de 7ter.97 ET de 7ter.96). A LIRE AVANT LE RESTE DE §18, Y COMPRIS AVANT LE BLOC 7ter.97.
>
> **0. TOUT SE REJOUE A L UNITE.** 85 / 0 / 0 / 0 sur `4f77afc1`, le tableau d alignement de
> 7ter.97 (5bis), `(13,13)` rang 1/144, les 48 composants de `ti=40`, l enumeration des
> 24 radicaux, les 14 tags VEHICULE et le **6/14**, les 7 lignes de `78919882`, GRENADE
> **3/3, 3/3, 9/9**, MELEE **8/12, 2/4, 9/12**. **Aucun chiffre publie n est faux.** Ce sont trois
> CONTROLES et deux PORTEES qui ne tiennent pas.
>
> **1. LE SIGNAL DE CO-MOUVEMENT EST REEL, ET IL SORT RENFORCE DE DEUX ALTERNATIVES QUI TOMBENT.**
> Ce ne sont ni des doublons (**0/85 a distance nulle, 0/85 a 52 bits bruts egaux**) ni une bande
> mal etiquetee (**10/10 slots reellement declares `ti=35`**). Mieux : la distance est un **OFFSET
> RIGIDE** — `veh=968/bip=701` tient `0.002983..0.003833` sur 24 trames — ce qui absorbe au
> passage le confondant << bipede TUE a cote du vehicule >> de 7ter.90 (8), qu aucun lot n avait
> ecarte : un mort ne conserve pas un offset constant d un vehicule qui s eloigne.
>
> **2. MAIS SA PORTEE EST D UN FILM, PAS DE QUATRE — ET C EST UNE FAUTE DE FORME, PAS DE CALCUL.**
> Le tableau (4) de 7ter.97, qui porte << hors de ses deux films, le negatif ne se reproduit
> pas >>, est mesure au **seuil ABSOLU `0.002/s`** — exactement la forme que le (5bis) de la meme
> section disqualifie, et pour la meme raison : changer l echelle change ce que le seuil veut
> dire, et **le decoupage change d un film a l autre** (de `(13,13)` a `(18,16)`). Rejoue au
> **seuil en PERCENTILE**, la seule forme defendable :
>
> ```
>   film       slots  cal BIP / VEH        roulant   REEL   nulles     serie
>   4f77afc1    256   (15,15) / (15,15)     11 240     34   0 / 0 / 0    20
>   036c102a     51   (15,15) / (15,16)      2 857      8   0 / 0 / 0     5
>   00ba2e1c     52   (16,14) / (15,15)      2 512      0   0 / 0 / 0     0    <- bandes DIVERGENTES
>   03af54c3     50   (14,13) / (14,13)      2 013      0   0 / 0 / 0     0    <- NEGATIF PROPRE
>   78919882     19   (16,14) / (15,15)      2 362      0   0 / 0 / 0     0    <- bandes DIVERGENTES
>   fccc61cd     21   (18,16) / (18,16)        520      0   0 / 0 / 0     0    <- 1.6 attendu : muet
> ```
>
> Au taux de `4f77afc1` (`34 / 11 240 = 0.00302` par trame roulante), l attendu vaut 8.6 sur
> `036c102a` (**observe 8, ACCORD**), 6.1 sur `03af54c3` (**observe 0**). **A ECRIRE DESORMAIS** :
> *le co-mouvement existe et se separe de sa nulle sur DEUX films ; il est absent sur un troisieme
> de calibration pourtant concordante ; il n est pas testable sur trois autres.* Statut :
> **`[MESURE] SUR DEUX FILMS`**, pas plus. Et **le negatif de 7ter.90 se reproduit sur ses deux
> films** a cette forme (0 et 0).
>
> **3. TROIS CONTROLES DE 7ter.97 N ONT AUCUN DEGRE DE LIBERTE.**
> (a) **LE CONTROLE POSITIF EST UNE IDENTITE ALGEBRIQUE** — quatrieme PATRON D du chantier. Le
> passager synthetique est la piste vehicule translatee de `dmax/2` : `dist = dmax/2 <= dmax` et
> `e = 0 <= mmax*lv` TOUJOURS, donc `match == close == avail` par construction (d ou les
> `25 666/25 666`, `2 848/2 848`, `18 644/18 644`). **Mesure : il rend 1.00000 AUSSI au decoupage
> `(14,15)` que 7ter.97 declare absurde.** Le vrai positif, un passager requantifie sur 13 bits,
> vaut **0.955** — c est ce chiffre qu il faut publier.
> (b) **L UNICITE** (<< 10 bipedes apparies a un seul vehicule >>) : sur 253 pistes bipedes,
> `P(zero collision) = 0.84` a 10 couples et **0.977** a 4. Elle mesure le VOLUME d appariements,
> pas leur qualite.
> (c) **LA BANDE LEURRE** rend 42 enregistrements sur 35 pistes, donc `dispo = 0` : aucune nulle
> n y est possible.
>
> **4. LE CONTROLE D ALIGNEMENT EST STRUCTURELLEMENT ASYMETRIQUE.** `sh = 52-(w0+w1+w2)` : `w0+1`
> n ajoute qu un bit de POIDS FAIBLE, `w0-1` promeut un bit d un axe au rang de POIDS FORT d un
> autre. Il ne peut donc echouer que vers `(14,15)` et `(15,14)`. **`(16,16)`, non teste par
> 7ter.97, rend 50 contre 0/0/0, serie 21** — plus que le decoupage calibre. Ni `w0` ni `w1` ne
> sont determines a un bit pres, et le balayage le disait deja : `(15,16)=0.0027` EGALE
> `(15,15)=0.0027`. Ce controle separe << geometrie a peu pres juste >> de << axes melanges >> ;
> **il ne pose pas l alignement au bit pres**.
>
> **5. LE DETECTEUR EST AVEUGLE A UN PASSAGER HORS PHASE.** `comove` exige que le bipede ait emis
> sur **les DEUX trames** de chaque pas de vehicule score. Un passager parfait qui n emet qu une
> trame sur deux rend `dispo = 0`. **Tout zero de cet instrument est un zero d occupation OU de
> cadence** — cela vaut pour 7ter.90 comme pour les quatre zeros ci-dessus.
>
> **6. DUREE, ET ELLE CONFIRME << INSTANT >>.** `4f77afc1` : 35 499 trames sur 1 203.7 s = **29.5
> trames/s**, donc les 22 trames consecutives valent **0.75 s**. La CO-LOCALISATION SEULE (sans
> condition de vitesse) ne depasse jamais **5.34 s continues**, 2.5 a 8.2 s cumulees par couple.
> << Trajet de passager >> est **exclu par la duree** ; << transition d embarquement >> reste
> `[NON VERIFIE]` mais rien ne le contredit.
>
> **7. SUR `4f77afc1` LES DEUX BANDES SONT SATUREES** : `ti=40` declare 256 slots sur 256
> (`[768..1023]`) **et `ti=35` en declare 253 sur 253** (`[512..764]`). Consequence : la
> verification croisee de 7ter.89 (8) (<< 99.87 % tombent sur un slot declare INDEPENDAMMENT
> `ti=40`, deux sources sans piece commune >>) **ne porte aucune information sur ce film** — dans
> une bande pleine, l accord est automatique. C est le film de la demonstration la plus forte, et
> celui ou l etiquetage est le moins verifiable.
>
> **8. LA PISTE DE L UTILISATEUR N EST PAS MORTE — SEULEMENT DEPLACEE.** L enumeration est juste,
> mais << LE RATTACHEMENT PAR CONTENANCE EST IMPOSSIBLE >> deborde la mesure. Ce qui est mesure :
> **le film ne transporte pas la GEOMETRIE**. Or il transporte **la POSITION (`i0`),
> l ORIENTATION (`i2 object-forward-and-up-dynamic-precision-component`) et l ECHELLE
> (`i12 object-scale-component`)** ; seules manquent les dimensions statiques du chassis, qui
> vivent dans les `.module` — **que le depot lit deja** (`internal/himodule/module.go`,
> `internal/ooz/ooz.go`, la chaine de la geometrie 2D des cartes). **La contenance devient une
> piste de JOINTURE, et elle ne depend d aucun seuil.**
>
> **9. COTE 7ter.96 (LIVRABLE) — DEUX CORRECTIONS ET UN GAIN.**
> (a) **LA FORMULE DE CONFIANCE EST TROP GENEREUSE D UN CRAN.** << l INSTANT, le TAG, la CLASSE,
> la CATEGORIE ne dependent pas de la bijection >> vaut pour un dead-state PRIS ISOLEMENT, pas
> pour une LIGNE : une ligne n existe que par `matchExact`, qui exige l egalite des DEUX noms, et
> la seule autre contrainte est `tolMS = 2500` — une fenetre de **5 s** quand le feed BTB porte
> **un kill toutes les 4.04 s** et que **17 des 55 frags VEHICULE ont un autre frag VEHICULE a
> moins de 2 500 ms**. **En marge nulle, c est la LIGNE ENTIERE qui depend de la bijection.**
> (b) << Pelican et Falcon n existent pas en multijoueur >> est juge **sans dire que les deux
> films sont sur des cartes de CREATION COMMUNAUTAIRE** (`Flood Gulch`, `High Ground`), dont la
> palette d objets est choisie par l auteur. Presomption de faussete affaiblie ; Theater arbitre.
> ⚠ **7ter.102 VA PLUS LOIN : LA PRESOMPTION EST RETIREE, PAS AFFAIBLIE.** Le Falcon existe en
> BTB (correction de l utilisateur) et `4f77afc1` **EST un BTB** ; le jugement portait donc contre
> le jeu reel. Les deux tags passent a `[NON VERIFIE]` et **rentrent dans le compte** : `8 / 14`.
> (c) **LE GAIN, ET IL ETAIT DANS LA MEME TABLE (REGLE 9)** : `shared.xuid_aliases` resout **les
> huit** gamertags de chaque film d arene, pas seulement JGtm. L accord API est exact **NOM PAR
> NOM, 8/8 et 8/8** (`p = 2/8!` par film) — un controle du kill-feed avec de vrais degres de
> liberte, la ou le multiset n en avait aucun. En BTB : **4 egalites exactes sur 8, et le decode
> n est JAMAIS au-dessus de l API**.
>
> **10. LE RECENSEMENT LAISSE OUVERT PAR 7ter.97 (1) EST FAIT, ET IL CONFIRME LE LOT `vc` EN
> L ELARGISSANT** : sur **150 films** (et non 20), **53 portent au moins un slot `ti=40`, soit
> 35 %**, avec des charges de **256, 230, 200, 199, 180, 172, 166, 148, 128, 117, 114...** Les
> deux films de 7ter.90 (**21** et **19**) sont bien **le bas absolu de la distribution**, et le
> corpus disponible pour un test hors echantillon est large. `tmp_vcaud census 150`.

> ## MISE A JOUR DU 2026-07-28, NUIT — 7ter.97 (lot `vc`). LIRE D ABORD LE BLOC 7ter.100 CI-DESSUS : trois de ses controles et deux de ses portees y sont corriges.
>
> **1. LE << REFUTEE COMME FAISABLE >> DE 7ter.90 EST UNE PORTEE DE DEUX FILMS, ET IL FAUT
> CESSER DE L ECRIRE SANS ELLE.** Les deux reserves du brief tombent — le corpus PORTAIT bien
> des vehicules (21 et 19 slots `ti=40`, verifie), et ses coordonnees etaient bien CALIBREES
> (`comove.go` lit `filmW[f]` et saute le film s il n y est pas). **Mais ce sont les deux films
> les moins charges du cache.** Le cache porte des films a **50 / 51 / 52** slots, et
> `4f77afc1` en porte **256** — deja au recensement de §18.1 — et **il n a jamais ete mesure**,
> parce qu il n a pas d entree dans `filmW`. Hors echantillon, le negatif ne tient pas : sur
> **quatre films neufs**, au critere IDENTIQUE, le reel passe **au-dessus de ses trois nulles**.
>
> **2. ET LE TEST QUI SEPARE LE PASSAGER DU PIETON QUI LONGE REND UN ZERO EXACT SUR LA NULLE.**
> Le critere de 7ter.90 (`vehicule roulant >= 0.002/s`) ne distingue pas un passager d un pieton
> qui court le long d un vehicule lent — le fond `BIP<->BIP` le prouve. Porte a **`0.012/s`**,
> un regime que les pietons de `4f77afc1` n atteignent quasiment pas (leurs occasions tombent de
> **366 064 a 6 471**, leurs appariements de **1 425 a 2**, serie maximale **1**) :
> **REEL 85 appariements, 10 couples, chacun des 10 bipedes apparie a UN SEUL vehicule, series
> de 22 et 20 trames consecutives — NULLES +101 / +307 / +1009 : `0` / `0` / `0`**, alors
> qu elles ont PLUS de trames << proches >> que le reel ; **bande LEURRE : 0** ; **controle
> positif : 1.00000**.
>
> **2bis. ET LE CONTROLE D ALIGNEMENT PASSE, UNE FOIS POSE A ECHELLE CONSTANTE** — c est ce qui
> fait monter le statut a **`[MESURE]`**. Un seuil ABSOLU rend ce controle ininterpretable
> (changer `(w0,w1)` change ce que `0.012/s` veut dire : les trames << roulantes >> triplent).
> La forme valide est un **percentile de la distribution des vitesses DU DECOUPAGE TESTE** —
> alors `roulant = 11 240` dans les quatre lignes, et seule la GEOMETRIE change :
> `(15,15)` calibre **34 contre 0/0/0**, serie 20 · `(14,15)` **205 contre 186**, serie 3 ·
> `(15,14)` **454 contre 412**, serie 2. Un bit d ecart et le reel EGALE sa nulle a 10 % pres.
> Troisieme controle gratuit dans la meme sortie : au mauvais decoupage la vitesse du 90e
> percentile vaut **21.3 et 15.3 unites/s dans une boite de cote 1** — vingt fois la carte par
> seconde ; au bon, **0.0164**. ⚠ `(16,15)` porte le signal lui aussi : **`w0` n est pas
> determine a un bit pres**, seul `w1` l est.
> **CE QUI EMPECHE `[ETABLI]`** : aucune reproduction par un tiers. **ET CE QUE CE N EST PAS** :
> 22 trames consecutives ne sont pas un trajet — le modele qui rend compte de tout est une
> **TRANSITION D EMBARQUEMENT**, `[NON VERIFIE]`, a tester en croisant `killsource` (le
> confondant << bipede tue a cote du vehicule >> que 7ter.90 (8) n avait pas absorbe).
>
> **3. LA PISTE << BOITE ENGLOBANTE / RATTACHEMENT PAR CONTENANCE >> EST MORTE, ET C EST MESURE.**
> ⚠ **TITRE PERIME — LIRE LE POINT 1 DU BLOC 7ter.102 EN TETE DE §18.** Ce qui est mort, c est
> << lire la boite englobante DANS LE FILM >>. **Le rattachement par contenance, lui, est VIVANT** :
> il devient une JOINTURE film (`i0` position, `i2` orientation, `i12` echelle) x `.module`
> (dimensions statiques), et il ne depend d aucun seuil. Ne pas citer ce titre sans la correction.
> Enumeration sans ancre des **325** noms du registre, **24 radicaux** ecrits d avance
> (`bound extent radius hull box dimension size volume collision sphere scale width height
> length aabb obb bbox capsule shape geometry mesh aabox diameter girth`) : **sept noms sortent
> en tout, et un seul est porte par `ti=40` — `object-scale-component` (i12), un FACTEUR
> D ECHELLE.** Zero `bound` / `extent` / `radius` / `hull` / `aabb` / `obb` / `capsule` /
> `collision` sur l archetype vehicule. **Le film ne transporte pas la geometrie des objets** ;
> elle est dans les `.module`. Controle positif de l enumeration : les trois noms `seat` de
> 7ter.89 (9) sont retrouves a l identique ; controle negatif : trois radicaux absurdes rendent 0.
>
> **4. TROIS ACQUIS DE PLUS, GRATUITS.** (a) `vehicle-weapon-set-component` est bien a **`i38`** :
> le `[NON VERIFIE]` de la piste 3 de §18.3 est **LEVE**. (b) La calibration survit au changement
> d instrument : `(13,13)` **rang 1 sur 144** sur le film temoin, par un balayage ecrit
> independamment. (c) **Le registre ne porte PAS la calibration par carte** : `L(i0) = 0` sur
> **947 films et ~95 cartes**, deux signatures seulement et elles ne se separent pas par carte —
> donc `W = 6+L` ne decrit pas ces largeurs, et elles restent une quantite EMPIRIQUE.
> ⚠ Reserve neuve : sur `78919882`, bipede `(16,14)` et vehicule `(15,15)` **DIVERGENT** — le
> couple n y est pas determine et les deux bandes n y sont pas garanties dans le meme repere.

> ## MISE A JOUR DU 2026-07-28 SOIR — 7ter.90 (lot `vk`). TROIS CHOSES CHANGENT ICI.
>
> **1. LE CHIFFRAGE DES FRAGS AU VEHICULE EST FAIT, ET IL N Y AVAIT RIEN A DECODER.** Le
> catalogue porte **89 tags de classe VEHICULE, tous de statut `VALIDE`, AUCUN avec un nom** :
> la regle d affichage etant `Name != "" && Publishable()`, **100 % tombent en `Autres`**. Le
> chassis est deja dans le champ `Detail` (17 chassis reels : `veh_cv_ghost`, `veh_cv_banshee`,
> `veh_un_scorpion`, `veh_un_wasp`, `tur_un_gausscannon`, ...), **9 tags sur 89 sans banque**.
> **Integrer les frags au vehicule est une table d etiquettes sur 89 lignes, pas de la RE.**
> ⚠ **CETTE PHRASE EST TROP OPTIMISTE, ET ELLE A ETE CORRIGEE DEUX FOIS** (7ter.96 (3), puis
> **7ter.102**). La racine de banque du `Detail` ne cite **UNE SEULE** racine que sur **53 des 89
> tags (59.6 %)** — 9 n en citent aucune, 27 en citent 2 a 6. Ce ne sont donc pas 89 lignes a
> recopier : ce sont **53 lignes recopiables**, **27 disjonctions a publier entieres** (jamais
> choisir) et **9 tags a nommer par une autre voie** (la chaine `jpt! -> proj -> weap` de
> `.ai/PONT_SONORE_ARMES.md`, non instruite). ⚠⚠ Et le `6 / 14` publie entre-temps est lui aussi
> **RETIRE** : il melait une mesure et une croyance sur le jeu — cf. bloc 7ter.102 en tete de §18.
> Comptes : **18.4 %** des morts publiees en BTB (102/555, 13 films) contre **0.75 %** hors BTB
> (32/4 293, 52 films) — facteur **25**. La classe n est PAS << BTB seulement >> : elle sort sur
> six films non-BTB (cartes communautaires a vehicules, et jusque sur de l Arena), zero sur les
> 46 autres. ⚠ Le denominateur BTB est CHOISI par le decodeur : onze de ces
> treize films publient 1 a 39 morts pour ~300 a l API (cause nommee 7ter.52 SONDE C, correctif
> **toujours a poser** dans `cv_autocalib.go`). ⚠⚠ Ce chiffre hors BTB a valu **0 sur 1 327**
> puis **14 sur 2 557** avant de se figer : **un zero lu sur une passe inachevee n est pas un
> zero** (meme famille que le PATRON F).
>
> **2. L OCCUPATION PAR CORRELATION DE POSITION ET DE VITESSE EST `[REFUTEE]` COMME FAISABLE.**
> Pas faute de volume — **5 428 et 23 932 observations** de position de vehicule, 11 slots vus,
> mediane de 69 s et 162 s par slot. Le co-mouvement (vehicule roulant + distance + meme
> deplacement) rend **19/15 735 et 49/102 045**, soit **0.00121 et 0.00048** ; la nulle par
> DECALAGE DE TRAMES rend **jusqu a 20 et 81 appariements**, donc le reel tombe DEDANS ; et le
> fond **pieton-pieton vaut 0.01019 et 0.00348**, soit 8 et 7 fois PLUS. Le controle positif du
> meme detecteur (passager synthetique) passe a **1.00000**. **Le passager n est pas dans le flux
> de position absolue.** La bande LEURRE rend **0 intervalle** partout. `DriverAssists`
> **n existe pas dans `match_participants`** : la reference externe faible annoncee est
> indisponible offline.
>
> **3. ET LE SOUS-PRODUIT VAUT PLUS QUE LE LOT : LES POSITIONS DE `filmdec` SONT FAUSSES PARTOUT
> SAUF SUR `000d5950`.** Voir la premiere ligne de §2.5bis. La calibration par carte est
> retrouvee par un balayage dont le controle positif designe la reponse connue **a la premiere
> FAMILLE `(w0,w1)`**. **A poser dans `filmdec` avant tout travail de position hors `000d5950`**
> — cela concerne le replay 2D et les trajectoires, pas seulement les vehicules.

> ## MISE A JOUR DU 2026-07-28, SOIR TARD — 7ter.94 (lot `tp.ref`, verification adversariale).
>
> **CE QUI TIENT, REJOUE A L UNITE** : le catalogue (89 tags VEHICULE, 0 nom, 0 statut non
> VALIDE, 9 sans banque) ; le negatif d occupation par co-mouvement (19/15 735 = 0.00121 et
> 49/102 045 = 0.00048, nulles a 20 et 81, controle positif **1.00000**, fond pieton-pieton
> 0.01019 et 0.00348) ; la calibration `(18,16)` / `(15,15)` sur la bande vehicule.
>
> **CE QUI EST REFUTE — UN CHIFFRE, PAS LA CONCLUSION** : << `(13,13)` **rang 1 sur 1608** >>.
> `vkcal sweep 000d5950`, rejoue tel quel, imprime `REFERENCE (13,13,14) : rMoy=0.0036
> (rang 6/1608)`. Cinq candidats font strictement mieux, **et les onze premiers sont tous des
> `(13,13,w2)`** : `w2` ne change que la troncature de l axe 2 (lisible dans `norm()`), donc
> **l espace d hypotheses effectif est de 144 couples, pas 1608 triplets**. La confiance publiee
> etait sur-evaluee d un facteur 11. `(13,13)` reste le meilleur couple. **PATRON F / REGLE 9 :
> la bonne valeur etait imprimee par l outil du lot, sur la ligne prevue pour elle.**
>
> **DEUX RESERVES AJOUTEES** : sur `78919882`, `(15,16)` egale `(15,15)` a 0.0032 — **`w1` n y
> est pas determine** ; et l exactitude **METRIQUE** des positions reste `[NON VERIFIE]`, ce qui
> est mesure etant une CONTINUITE, sans aucune verite terrain hors `000d5950`.
>
> **SUR LA << DUREE MINIMALE >> QUE LE BRIEF EXIGEAIT** : elle manque bien a 7ter.90, mais son
> absence ne peut pas sauver un negatif — l exiger ne peut que DIMINUER le nombre
> d appariements. La sortie donne d ailleurs la reponse : la plus longue association
> co-mouvante observee vaut **20 trames** (`78919882`) et **10 trames** (`fccc61cd`), sur des
> pistes de 69 s a 439 s. **Aucune association soutenue n existe.**

> ## MISE A JOUR DU 2026-07-28, LIVRABLE UTILISATEUR — 7ter.96 (lot `vl`). QUATRE CHOSES.
>
> **1. LA << PISTE 0 >> DE §18.3 EST PLUS CHERE QUE `[89 LIGNES DE TABLE]`, ET C EST MESURE.**
> Sur les **14 tags de classe VEHICULE reellement observes** dans trois films, la racine de
> banque Wwise du champ `Detail` ne nomme un chassis a la fois **UNIQUE et PLAUSIBLE que 6 fois
> sur 14 (43 %)** : 4 tags citent plusieurs chassis (`674a7d69`, `a859230a`, `003f582d`,
> `3b3b3d40`), 2 n en citent aucun (`00426796`, `d2ffec3f`), et **2 en nomment un qui n existe
> pas en multijoueur Halo Infinite** — `28907150` (Pelican, deux lignes) et `77a61ef5` (Falcon),
> tous deux en `credit-concordant/marche`, la voie a 98.2 %. **La banque est REUTILISEE entre
> chassis** : c est une piste de nommage, pas une identite. Regle a poser avec la table :
> **publier la disjonction, jamais choisir** — la premiere version de `tmp_vklist` prenait la
> premiere occurrence et sortait un << Gauss >> affirmatif et faux sur `003f582d`.
> ⚠⚠ **LE `6 / 14` ET LES DEUX << N EXISTE PAS EN MULTIJOUEUR >> SONT RETIRES — 7ter.102, ET C EST
> UNE FAUTE DE METHODE, PAS UN CALCUL FAUX.** Le compte additionnait une mesure (<< racine
> UNIQUE >>) et une croyance sur le jeu (<< chassis PLAUSIBLE >>). **Le Falcon existe en BTB**, et
> `4f77afc1` **EST un BTB**. **LIRE `8 / 14` (57 %) et, mieux, `53 / 89` (59.6 %) au catalogue.**
> Statut des deux tags : `[NON VERIFIE]`. La conclusion << la banque est REUTILISEE entre
> chassis >> tombe a `[PLAUSIBLE]` sur les films — elle n est re-etayee que par une mesure de
> catalogue (**dix `vehi` citent 2 a 6 racines distinctes**), qui n autorise que la forme faible.
> **Cela retire un argument CONTRE, cela n ajoute aucune preuve POUR** : la confrontation Theater
> des trois instants reste due. Bloc 7ter.102 en tete de §18.
>
> **2. LE CORPUS BTB DE 7ter.90 ETAIT ALPHABETIQUE, DONC ANCIEN, ET SA RESERVE NE TIENT PAS SUR
> UN FILM RECENT.** `tmp_vkfrag` trie par nom de dossier : les 13 BTB comptes datent de 2023 a
> 2025. Le **seul BTB de 2026** du cache, `4f77afc1` (BTB:CTF / Flood Gulch, 2026-07-24), publie
> **225 morts pour 303 kills API = 74.3 %**, configuration gelee — pas 1 a 39. **Ce qui reste
> vrai du blocage BTB, c est la BIJECTION** (`marge 0`, alerte de sante, donc
> `LineByLinePublishable == false`), pas la couverture. Il porte **55 frags de classe VEHICULE
> (24.4 %)**.
>
> **3. UNE REFERENCE EXTERNE EXISTE POUR LE MEME INSTRUMENT, ET 7ter.90 L AVAIT MANQUEE.**
> `match_participants` porte `grenade_kills` et `melee_kills`. **GRENADE : egalite EXACTE
> 3/3, 3/3, 9/9** sur les trois films (REGLE 1). **MELEE : deficit systematique 8/12, 2/4, 9/12**,
> inexplique — et il ne vient PAS de la couverture, `78919882` publiant 99 morts sur 99.
> ⚠ Ce controle valide la CLASSE la ou une reference existe ; il ne valide **ni** la classe
> VEHICULE (aucune reference), **ni** le nom du chassis.
>
> **4. PRECISION SUR §18.1 (c) — L ARME NE DEPEND PAS DE LA BIJECTION, SON RATTACHEMENT A UNE
> LIGNE NOMMEE, SI.** `killsource/match.go::matchExact` compare
> `e.victim == roster.nameOf(cd.victim)` **ET** `e.killer == roster.nameOf(cd.killer)`. Formule a
> employer, **CORRIGEE PAR 7ter.100 (12) — la version ci-dessous etait trop genereuse d un cran** :
> ne dependent pas de la bijection **le tag `jpt!`, la classe et la categorie D UN DEAD-STATE PRIS
> ISOLEMENT** ; **en dependent, sur une LIGNE PUBLIEE, les deux noms ET le rattachement de l arme
> a l instant — donc la ligne entiere**, parce qu une ligne n existe que par `matchExact` et que
> la seule autre contrainte est `tolMS = 2500` (fenetre de 5 s) alors que le feed BTB porte un
> kill toutes les **4.04 s** et que **17 des 55 frags VEHICULE ont un voisin VEHICULE dans la
> fenetre**. En BTB une ligne se lit donc << dans les 5 s autour de T, une mort a ete causee par
> un Wraith >>, jamais avec des noms et jamais a l instant exact.

### 18.1 CE QUI EST ACQUIS

**(a) `[MESURE]` — LES VEHICULES SONT DANS LE FILM, ET INDIVIDUELLEMENT IDENTIFIES.**
Archetype **`ti=40`**, trouve PAR SON NOM (composants `vehicle-*` du registre `chunk_00`), jamais
par un indice code en dur : 48 composants, dont 14 prefixes `vehicle-`, identique sur 5 films
(5 concordants, 0 divergent). Bande de slots dediee **`[768, 1023]`**, jamais depassee. Et la
presence est **LOCALISEE EXACTEMENT sur les films qui en portent** :

```
  000d5950 (Arena)     ti=40 : AUCUN slot
  9b191a7f (Arena)     ti=40 : AUCUN slot
  78919882             ti=40 :  19  [768..786]
  fccc61cd             ti=40 :  21  [768..788]
  4f77afc1 (BTB)       ti=40 : 256  [768..1023]
```

**REPRODUCTEUR : AUCUN.** Le recensement `0/0/19/21/256` a bien ete rejoue a l unite par le lot
`tv.ref` (7ter.89 (6)) — mais avec **un seul agent et un seul binaire**, sur la meme session et
les memes cinq films. Il n existe ni second corpus, ni verite terrain Theater sur ce sujet.
**Le statut ne montera pas a `[ETABLI]` sans une reproduction reellement independante**, et il ne
faut pas l ecrire comme tel en attendant (regle 7 du §1 : un `[ETABLI]` NOMME son reproducteur).

⚠ **DEUX PIEGES DANS CE MEME TABLEAU** (7ter.89 (6)) : le **256** du BTB est une **SATURATION**
de bande (256 valeurs possibles, 256 declarees) — sur ce film le filtre de slot ne filtre plus
rien ; et les bandes **grandissent avec le roster** (`ti=35` monte jusqu a 764 sur le BTB, `ti=40`
demarre a 768, trois slots d ecart). **Coder `[768,1023]` en dur est un pari sur la taille du
roster**, exactement le piege deja consigne en 7ter.52 (0)(i) pour la plage bipede.

⚠⚠ **TROISIEME PIEGE, MESURE PAR 7ter.100 (4) : SUR `4f77afc1` LA BANDE BIPEDE EST SATUREE
AUSSI** — `ti=35` y declare **253 slots sur les 253** de `[512..764]`. Les deux bandes sont donc
pleines, et **la verification croisee de 7ter.89 (8)** (<< 99.87 % des echantillons tombent sur un
slot que les keyframes declarent INDEPENDAMMENT `ti=40` — deux sources sans piece commune >>)
**n y porte AUCUNE information** : dans une bande pleine, l accord est automatique. C est le film
sur lequel les demonstrations les plus fortes du sujet ont ete faites, et celui ou l etiquetage
d un slot est le moins verifiable.

**(b) `MESURE+VERIFIE` (7ter.46) — LE NOMMAGE DES ARMES DE VEHICULE EST ACQUIS.**
Regle **R-VEHICULE** : un `weap` est un armement de vehicule s il est reference par un `vehi`, ou
s il pend a la chaine `vcdd -> sofd -> sofa -> uwfa -> weap`. **62 `weap` sur 194** sous la racine
vehicule (46 par un `vehi` direct, 16 par la chaine `vcdd`). Force de la regle : disjonction
TOTALE d avec le catalogue tenu en main (**0/62**, attendu 10.2, **p = 1.1e-06**) ; `uwfa` :
16 entrees dans tout le jeu, 16/16 referencees par un `sofa`, 0 atteignable depuis un `wcfg`.
Distinction `turret` / `fixed` confirmee en Theater. **La moitie NOMMAGE est acquise.**

**(c) LES KILLS A L ARME DE VEHICULE SONT DEJA LISIBLES — par la source du degat fatal.**
Le decodeur de production ne lit pas << dans quoi etait le tueur >>, il lit le **tag `jpt!` du
dead-state i11 de la VICTIME**, c est-a-dire la SOURCE DU DEGAT. Les tags des armes de vehicule
sont au catalogue par (b). Donc la question **<< quand frague-t-il AVEC un canon de vehicule >>
est en grande partie deja repondue par le decodeur existant** — cf. `.ai/V7.5/killweapon/GUIDE_KILLSOURCE.md`.
**Ce qui manque n est pas << avec quoi >>, c est << quand est-il DEDANS >>** : le temps passe a
bord, le role (pilote / tourelle / passager), les morts subies en vehicule.

### 18.2 CE QUI BLOQUE — ET C EST UN SEUL VERROU : LA MARCHE

Tout le reste (`i10` l accrochage, l armement) se lit par la **MARCHE SEQUENTIELLE** des
composants d un enregistrement : pour atteindre le composant `iN`, il faut avoir consomme
exactement le bon nombre de bits sur `i0..iN-1`. Un seul composant approximatif decale tout ce
qui suit. Aujourd hui, sur les films a vehicules, **la marche ne ferme proprement que 5.9 % a
9.0 % des paquets** (2 309 / 38 936 et 2 423 / 26 863, 7ter.89 (9)) — c est pour cela que le lien
joueur-vehicule ne rend que **9 observations sur un film entier**. Toute distribution lue
la-dessus mesure OU on meurt, pas ce qu on cherche (PATRON A / symptome E9).

⚠ **NE PAS ATTRIBUER AU VEHICULE LE CHIFFRE << i0 : 45 bits lus sur 60 >>.** Il circule comme le
bloqueur du vehicule ; il ne l est pas. Ce chiffre porte sur **`object-position-component`
(`FUN_14076e29c`) de l archetype `ti=41`, le PROJECTILE** (7ter.84 (7), index §2.1). L archetype
`ti=40` porte a `i0` un AUTRE composant — `object-position-dynamic-precision-component`
(`FUN_1406cfe44`), le meme que le bipede `ti=35` — et **les 30 composants `i0..i29` de `ti=40`
ont TOUS un deserialiseur porte** dans `filmdec/traverse.go` ; le premier absent est
**`i30 vehicle-auto-turret-triggers-component`** (verifie un par un, 7ter.89 (7)). Les deux
composants sont voisins de nom et de role : les confondre envoie desassembler la mauvaise
fonction. **Ce qui reste vrai des deux cotes** : `i0` precede tout le reste, donc un `i0` inexact
ferme l acces a tout l archetype.

**RESERVE STRUCTURELLE SUR `i10`, ET ELLE N EST PAS LEVEE** (7ter.89 (7)) : la grammaire ecrite
`[porte 1][sonde 1][slot 13][generation 2][marqueur 16]...` est conforme au port Go, mais les
**13 bits viennent de `quantStatDefaultWidth` (`filmdec/entity_quant.go`), la configuration PAR
DEFAUT** (`DAT_144706100 = 0x1fff`), pas d une mesure sur `ti=40`. Or `FUN_1406d3140` prend sa
largeur de la table du POOL, celle dont la colonne `base` n a jamais ete capturee (7ter.88 (1)).
**Pool different = largeur differente = tout le composant decale.** C est une hypothese, pas une
lecture — et c est un candidat serieux comme cause du 5.9 %.

### 18.3 LES PISTES, PAR ORDRE DE COUT CROISSANT

> **PISTE 0, AJOUTEE PAR 7ter.90 ET MOINS CHERE QUE TOUTES LES AUTRES : LES 89 ETIQUETTES DE
> CLASSE VEHICULE.** Aucune retro-ingenierie, aucun film a ouvrir : le chassis est deja dans le
> champ `Detail` du catalogue embarque. C est la seule piste du sujet qui livre quelque chose a
> l utilisateur aujourd hui.
> ⚠ **ELLE N EST PAS << UNE TABLE DE 89 LIGNES >>, ET LE CHIFFRE QUI LE DISAIT A ETE CORRIGE DEUX
> FOIS.** 7ter.96 (3) publiait `6/14` ; ce compte melait une mesure et une croyance sur le jeu, et
> **7ter.102 le recompte a `8/14` (57 %) sur les tags observes et `53/89` (59.6 %) sur le catalogue
> entier**. Reste a traiter, tag par tag : **27 tags a racines multiples** (publier la disjonction,
> jamais choisir) et **9 tags sans aucune banque** — pour ceux-la, la voie est la chaine
> `jpt! -> proj -> weap` deja documentee dans `.ai/PONT_SONORE_ARMES.md` (cas `00426796`, la
> Gungoose), et elle n a pas ete instruite.

> **PISTE 1, ET C EST LA MEILLEURE DU SUJET PARCE QU ELLE NE DEPEND D AUCUN SEUIL : LE
> RATTACHEMENT PAR CONTENANCE, EN JOINTURE.** Elle avait ete declaree morte a tort — la mesure de
> 7ter.97 (6) dit seulement que **le film ne transporte pas la GEOMETRIE**, pas que la contenance
> est hors de portee.
>
> ```
>   DU FILM              i0   position                                  (deja lue)
>                        i2   object-forward-and-up-dynamic-precision    (orientation)
>                        i12  object-scale-component                     (echelle)
>   DES `.module`        les dimensions statiques du chassis
>   LECTEURS DEJA LA     internal/himodule/module.go · internal/ooz/ooz.go
>                        (la chaine de la geometrie 2D des cartes les lit deja)
> ```
>
> **POURQUOI ELLE EST STRUCTURELLEMENT SUPERIEURE A LA CORRELATION** : un test de contenance est un
> predicat geometrique — un point est dans une boite ou il n y est pas. Tout le dossier du
> co-mouvement (§18.4, 7ter.97, 7ter.100) montre au contraire un signal qui **vit et meurt par son
> seuil**, jusqu a rendre 34 ou 0 selon que le seuil est absolu ou en percentile.
> **PREALABLES, ET ILS SONT REELS** : (i) la calibration par carte de `i0` doit etre posee dans
> `filmdec` (piste 0bis) — sans elle les positions ne sont des coordonnees que sur `000d5950` ;
> (ii) `i2` et `i12` ne sont pas encore LUS, seulement enumeres — ils sont sur le parcours
> sequentiel de `ti=40`, donc soumis au verrou de la MARCHE (§18.2) ; (iii) la correspondance
> `chassis observe -> entree .module` n a jamais ete etablie. **Statut : `[PLAUSIBLE]` comme
> mecanisme, aucune mesure a ce jour.**
>
> **PISTE 0bis : POSER LA CALIBRATION PAR CARTE DE `i0` DANS `filmdec`.** Elle deborde le sujet
> vehicules (replay 2D, trajectoires) et elle est prete : 7ter.90 (4).
>
> ⚠ **CETTE LIGNE A ETE PERIMEE DEUX FOIS — LIRE LE BLOC 7ter.100 EN TETE DE §18.** Elle disait
> d abord << NE PAS ROUVRIR LA CORRELATION DE POSITION ET DE VITESSE POUR L OCCUPATION >> ;
> 7ter.97 l a rouverte au statut `[MESURE]` ; **7ter.100 en reduit la portee a DEUX FILMS et
> jette trois de ses controles.**
> **ETAT COURANT, ET IL EST PRECIS.** Le negatif de 7ter.90 vaut sur `fccc61cd` et `78919882`, et
> **il s y reproduit** au seuil en percentile (0 et 0). Le signal, lui, existe et se separe de sa
> nulle sur **`4f77afc1` (34 contre 0/0/0, serie 20) et `036c102a` (8 contre 0/0/0, serie 5)**,
> et il est **ABSENT sur `03af54c3`** dont la calibration est pourtant concordante (attendu 6.1,
> observe 0). Les deux derniers films ne sont pas testables (bandes bipede et vehicule
> DIVERGENTES). **Statut : `[MESURE] SUR DEUX FILMS`.** Ce qui l a fait monter de `[PLAUSIBLE]`
> n est PAS le controle positif (identite algebrique, 7ter.100 (2)) ni l unicite (`P = 0.84` sous
> la nulle, 7ter.100 (5)) : c est la nulle par decalage a **zero exact**, la constance de
> l offset (0.00298..0.00383 sur 24 trames) et l elimination mesuree du doublon (0/85 bits
> egaux).
> **CE QUI RESTE AVANT `[ETABLI]`, DANS CET ORDRE** : (i) **expliquer `03af54c3`** — tant qu un
> film propre rend zero, le signal n est pas une propriete du jeu ; (ii) une reproduction par un
> tiers, avec un controle positif QUI PEUT ECHOUER (le passager requantifie, 0.955) ; (iii) etendre
> aux films riches du recensement (53 films sur 150 portent des vehicules, jusqu a 256 slots),
> en n en gardant que ceux dont les deux bandes CONCORDENT ; (iv) ecarter le confondant
> << bipede tue a cote du vehicule >> en croisant `killsource` — largement absorbe par la
> constance de l offset, mais non mesure.

**1. FAIRE REMONTER LE TAUX DE MARCHE PROPRE SUR LES FILMS A VEHICULES.** C est le bloqueur
unique : tant qu il vaut 5.9 %, aucune population n existe et rien n est testable. Deux entrees,
dans cet ordre : (i) capturer la **colonne `base` de la table de configuration du POOL** lue par
`FUN_1406d3140`, qui fixe la largeur reelle du handle et leve la reserve de §18.2 ;
(ii) desassembler le deserialiseur du premier composant qui desynchronise, a la methode qui a
deja produit les autres. **Prealable non negociable a toute reouverture de la piste 2.**

**2. LIRE `i10` (L ACCROCHAGE) SUR LES VEHICULES, une fois (1) fait.** L enumeration rend cette
hypothese **OBLIGATOIRE, pas seulement plausible** : sur les **325** noms de composants du
registre (et non 326 — corrige par 7ter.89 (7)), les radicaux `mount`, `rider`, `occup`,
`passenger`, `enter`, `exit`, `attach`, `driver`, `gunner`, `embark` rendent **ZERO** ; `seat`
n en rend que trois, tous des reglages (`player-desired-respawn-seat`,
`vehicle-seats-override-pitch`, `vehicle-seats-override-yaw`). **AUCUN COMPOSANT DE SIEGE
N EXISTE.** `object-parent-state-component` (i10) est donc le **seul porteur possible** du lien
joueur -> vehicule dans ce schema. Statut : `[PLAUSIBLE]` comme mecanisme, **non refute** — ce que
7ter.89 (9) refute, c est << le lien est ETABLI >>, pas << i10 est le candidat >>.

**3. TROUVER LE DESERIALISEUR DU COMPOSANT D ARMEMENT DU VEHICULE, QUI N EN A PAS.**
Le vehicule **n a pas de `weapon-state-type-info`** (le bipede en porte **quatre**, a `i43..i46`,
et c est par la que se lit son arme tenue) : son armement passe donc par un composant propre.
Ghidra, meme methode que celle qui a nomme les parts de degats et resolu i9 : nom (`chunk_00`) ->
chaine unique `.rdata` -> unique xref = `getName()` -> descripteur -> `deser = *(ptrGetName+0x20)`
(et `+0x28` si c est le thunk `FUN_14076ce9c`) — §2.1 et 7ter.31.
✅ **`[MESURE]` DEPUIS 7ter.97 (7)(b) — LE `[NON VERIFIE]` EST LEVE.** La liste a ete re-enumeree
par un instrument independant : `ti=40` porte **48** composants, `i0..i29` **tous** avec un
deserialiseur, le premier absent est `i30 vehicle-auto-turret-triggers-component` (7ter.89 (7)
confirme a l unite), et **`vehicle-weapon-set-component` est bien a `i38`** — le nom et l indice
du lot `vh` sont exacts. **La piste peut demarrer sur cette cible sans re-verification.**

### 18.4 CE QUI EST MORT, AVEC SON CHIFFRE

- ⚠ **LIGNE CORRIGEE DEUX FOIS (7ter.97 puis 7ter.100) : LIRE `[REFUTEE] SUR DEUX FILMS`, JAMAIS
  `[REFUTEE] COMME FAISABLE`.** Le negatif ci-dessous est reproduit **a l unite** par un
  troisieme instrument sur `fccc61cd` — il n est pas discute LA, et **7ter.100 le retablit meme
  sur les deux films de 7ter.90** (seuil en percentile : `fccc61cd` **0**, `78919882` **0**).
  Ce qui subsiste hors d eux : un co-mouvement bref, separe de sa nulle sur **`4f77afc1`
  (34 contre 0/0/0)** et **`036c102a` (8 contre 0/0/0)**, ABSENT sur `03af54c3` a calibration
  concordante. ⚠ Le << quatre films neufs, reel au-dessus de ses trois nulles >> de 7ter.97 est
  mesure au seuil ABSOLU, forme que 7ter.97 disqualifie lui-meme : **ne pas le citer**.
  Detail : bloc 7ter.100 en tete de §18.
- **L OCCUPATION PAR CORRELATION DE POSITION ET DE VITESSE — `[REFUTEE]` SUR `fccc61cd` ET
  `78919882`, ET LE NEGATIF Y EST COMPLET** (7ter.90 (5)(6)(7)). Volume : 5 428 et 23 932 observations de vehicule,
  11 slots vus, 15 735 et 102 045 occasions de comparaison — **le test n est pas affame**.
  Proximite seule : elle ne discrimine pas (le controle **pieton-pieton** la reproduit,
  0.0335 contre 0.0674 et 0.0198 contre 0.0295) — PATRON A. Co-mouvement : **19 et 49
  appariements**, taux **0.00121 et 0.00048**, quand la nulle par decalage de trames monte a
  **20 et 81** et que le fond pieton-pieton vaut **0.01019 et 0.00348** (8 et 7 fois plus).
  ⚠ **<< Controle positif du meme detecteur : 1.00000 >> (2 848/2 848 et 18 644/18 644) EST UNE
  IDENTITE ALGEBRIQUE** — 7ter.100 (2) : le passager synthetique est la piste vehicule translatee
  de `dmax/2`, donc `match == close == avail` par construction, y compris a un decoupage absurde.
  **Ne plus le citer comme argument** ; le positif qui peut echouer (passager requantifie) vaut
  **0.955**. Bande LEURRE : **0 intervalle** partout — mais elle ne lit que **42 enregistrements
  sur 35 pistes**, donc `dispo = 0` : elle ne pouvait pas rendre autre chose (7ter.100 (5)).
  **CE QUE CELA VEUT DIRE** : le passager n emet plus sa position absolue — ce qui est la
  prediction du modele PARENTE, et donc un argument INDIRECT de plus pour `i10`. La prediction
  DIRECTE du modele (un TROU dans la piste du passager pendant que le vehicule roule) est
  `[PLAUSIBLE]` et **les deux films se contredisent** : `78919882` rend 9/29 contre une nulle a
  1/20-1/16-0/30, `fccc61cd` rend **0/11** contre 0/18-0/9-0/8. Neuf observations, et un
  confondant que la nulle n absorbe pas (un joueur TUE pres du vehicule disparait aussi).
- **LES TROIS CONTROLES QUI NE SOUTIENNENT PLUS RIEN — a retirer de tout brief sur ce sujet**
  (7ter.100 (2)(4)(5), rappel groupe ici parce qu ils sont cites separement ailleurs) :
  **(a) LE CONTROLE POSITIF EST UNE IDENTITE ALGEBRIQUE** — le passager synthetique est la piste
  vehicule translatee de `dmax/2`, donc `dist = dmax/2 <= dmax` et `e = 0 <= mmax*lv` TOUJOURS :
  `match == close == avail` par construction. Il rend **1.00000 y compris au decoupage `(14,15)`
  que 7ter.97 declare absurde**. Le positif qui PEUT echouer — passager requantifie sur 13 bits —
  vaut **0.955** : c est celui-la qu on cite. **QUATRIEME PATRON D du chantier** (§4).
  **(b) LE CONTROLE D UNICITE N A AUCUNE PUISSANCE** — << 10 bipedes apparies a un seul vehicule
  chacun >> : sur 253 pistes bipedes, `P(zero collision) = 0.84` a 10 couples et **0.977** a 4. Il
  mesure le VOLUME d appariements, pas leur qualite, et il passe **plus souvent qu il n echoue
  sous sa propre nulle**.
  **(c) LE CONTROLE D ALIGNEMENT EST STRUCTURELLEMENT ASYMETRIQUE** — `sh = 52-(w0+w1+w2)` : `w0+1`
  n ajoute qu un bit de POIDS FAIBLE, `w0-1` promeut un bit d un axe au rang de POIDS FORT d un
  autre. Il ne peut donc echouer que vers `(14,15)` et `(15,14)`. Et **`(16,16)`, NON TESTE par
  7ter.97, rend 50 contre 0/0/0 (serie 21) — plus que le decoupage calibre**. **Ni `w0` ni `w1` ne
  sont determines a un bit pres** ; ce controle separe << geometrie a peu pres juste >> de
  << axes melanges >>, rien de plus.
  **(d) ET LA DUREE N EST PAS CE QU ELLE PARAIT** : `4f77afc1` tourne a **29.5 trames/s**, donc les
  << 22 trames consecutives >> valent **0.75 s** — pas un trajet, au mieux une transition
  d embarquement (`[NON VERIFIE]`).
- **`DriverAssists` COMME REFERENCE EXTERNE — INDISPONIBLE OFFLINE.** Le champ est dans
  `internal/openspartan/models.go` et **absent de `match_participants`** (41 colonnes verifiees) :
  il est jete a l ecriture. Aucun croisement possible sans appel reseau. (7ter.90 (7))
- **LE BALAYAGE BIT A BIT PAR POSITION (scan de motif, `TryDeltaAt`) COMME LOCALISATEUR HORS DU
  PARCOURS NOMINAL — MORT, ET C EST UN INTERDIT, PAS UNE RESERVE.** C etait le chemin naturel
  pour lire un composant sans reussir la marche. Densite de faux positifs du predicat generique :
  un en-tete plausible sort **1 fois tous les 18 bits** ; **1 fois tous les 8 366 bits** avec
  TOUTES les contraintes reunies (i11 atteignable + victime et tueur presents + Mort==1). Et
  surtout **IL FABRIQUE DES DISTRIBUTIONS CREDIBLES** : scan cible sur `9b191a7f`, 5 940 hits ->
  **292 candidats pour 87 kills**, dont **193/292 sont la MEME paire (tueur=16, victime=5)** — un
  motif binaire repete — et les 201 << categorie=2 >> sont exactement cet artefact ; appariement
  au kill-feed autoritatif : **14/84 a <= 200 ms**. Cote vehicules, le meme scanner pointe sur une
  bande **VIDE** de 256 slots d un film sans vehicule rend **17 echantillons**, contre 5 sur la
  bande vehicule du meme film : le plancher de bruit est AU-DESSUS du signal.
  Sources : 7ter.27 (6) et index §3 ; controle de bande vide 7ter.89 (8).
  ⚠ Un chiffre rond de **<< ~99.85 % de faux positifs >>** circule dans les briefs : il n est
  **dans aucune section du RE_LOG**. Citer les chiffres ci-dessus, qui sont sources.
  **CE QUI RESTE VALIDE, EN REVANCHE : le scan CONTRAINT a une bande declaree.** Sur les deux
  films a vehicules, **23 946 contre 12** et **5 566 contre 45** echantillons face a une bande
  vide de meme largeur, et **99.87 % / 97.3 %** des echantillons tombent sur un slot que les
  keyframes declarent independamment `ti=40` — deux sources sans piece commune (7ter.89 (8)).
  La tracabilite EN POSITION est reelle ; c est le scan **hors bande declaree** qui est mort.
- `[RETIRE]` — **<< le passager emet sa position absolue >>.** Cette conclusion a ete tiree sur
  des coordonnees invalides. Elle est **retiree, et non nuancee** : rien n est etabli dans un sens
  ni dans l autre, et le sujet est a reprendre a zero si quelqu un y revient.
- `[REFUTE]` comme etabli — **le lien joueur-vehicule par `i10`, en l etat**. Sur un film a
  19 vehicules : **9** enregistrements de bipede a parente accrochee sur tout le film, **ZERO**
  parent designant un slot reellement lie a `ti=40`, **controle decale d un bit IDENTIQUE au
  reel** (1/9 des deux cotes, alors qu un bit d ecart doit detruire le champ — REGLE 3),
  generation etalee sur les 4 valeurs **`gen0` inclus** que l outil declare impossible, et le
  controle positif de l instrument (<< arme tenue >>) qui **echoue** : 21 variantes distinctes
  pour 21 observations (7ter.89 (9)). **Ce n est pas la piste qui est morte, c est la mesure.**

### 18.5 LES MOTIFS DE GREP DE CE SUJET

| SOUS-SUJET | MOTIF (fichier : `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md`, sauf mention) |
|---|---|
| **Cette page** | `VEHICULES — ETAT ET PISTES` -> index §18 |
| **Le chiffrage des frags au vehicule, et les 89 etiquettes manquantes** | `89 tags de classe VEHICULE` · `tmp_vkcat` · `tmp_vkfrag` · `sb_010_veh_` · `sb_010_tur_` · `facteur 33` -> 7ter.90 (1) ; **et la nuance mesuree** : `6 des 14 tags` · `banque REUTILISEE` · `Pelican` · `Falcon` · `publier la disjonction` · `tmp_vklist` -> 7ter.96 (3). ⚠ Le motif `6 des 14 tags` mene a un chiffre **RETIRE** : lire `8 / 14` et `53 / 89`. Motifs de la correction : `racine UNIQUE` · `8 sur 14` · `53 / 89` · `racines de banque DISTINCTES` · `dix vehi` · `croyance sur le jeu` -> **7ter.102 (1)(2)(3)(4)** |
| **La correction Falcon / Pelican, et la faute de methode qui va avec** | `le falcon existe en btb` · `NE JAMAIS DECLARER UN DECODAGE IMPLAUSIBLE` · `retire un ARGUMENT CONTRE` · `[NON VERIFIE]` · `02:19.035` · `03:40.856` -> 7ter.102 (1)(2), patron du §4 |
| **La liste ligne par ligne de `78919882` (le livrable rendu)** | `sans nom` · `tourelle UNSC` · `f712c64a` · `003f582d` · `marge de bijection 35` · `99 / 99` -> 7ter.102 (5) |
| **La liste de frags au vehicule rendue a l utilisateur (matchs recents)** | `4f77afc1` · `78919882` · `Flood Gulch` · `High Ground` · `74.3 %` · `multiset decode` -> 7ter.96 (1)(5) |
| **La reference API des classes GRENADE et MELEE (controle positif du meme instrument)** | `grenade_kills` · `melee_kills` · `3/3, 3/3, 9/9` · `deficit MELEE` -> 7ter.96 (2) |
| **L occupation par correlation (REFUTEE SUR DEUX FILMS SEULEMENT)** | `co-mouvement` · `passager synthetique` · `1.00000` · `decalage de trames` · `BIP<->BIP` · `bande LEURRE` · `tmp_vkcorr` · `tmp_vkcal` -> 7ter.90 (5)(6)(7) ; **et la correction de portee** : `AUDIT DU CORPUS` · `tmp_vcmv` · `VC_VMIN` · `0.012` · `85 appariements` · `ZERO EXACT` · `4f77afc1` · `10 couples` -> 7ter.97 (4)(5) |
| **Le corpus disponible, et pourquoi deux films ne suffisaient pas** | `tmp_vcaud` · `census` · `50 / 51 / 52` · `256 slots` · `filmW` -> 7ter.97 (1) |
| **LA BOITE ENGLOBANTE — ABSENTE DU FILM, MAIS LA CONTENANCE N EST PAS MORTE** | `boite englobante` · `contenance` · `24 radicaux` · `object-scale-component` · `aabb` · `hull` · `capsule` -> 7ter.97 (6) ; **et la correction de portee** : `une piste **jointure**` · `rattachement par contenance` · `himodule` · `ooz` · `object-forward-and-up` -> 7ter.100 (10). ⚠ Le titre `EST MORTE` de 7ter.97 (3) est **PERIME** : ce qui est mort, c est lire la boite DANS LE FILM. Motifs de la piste vivante : `PISTE 1` · `EN JOINTURE` · `ne depend d aucun seuil` · `dimensions statiques du chassis` -> index §18.3 et 7ter.102 (bloc de tete de §18, point 1) |
| **LES TROIS CONTROLES DE `vc` QUI N ONT AUCUN DEGRE DE LIBERTE** | `IDENTITE ALGEBRIQUE` · `dmax/2` · `match == close == avail` · `0.955` · `passager requantifie` · `P(zero collision)` · `effet de volume` · `42 enregistrements sur 35 pistes` -> 7ter.100 (2)(5) |
| **LE SEUIL EN PERCENTILE SUR SIX FILMS — LA PORTEE REELLE DU SIGNAL** | `VC_PCT` · `seuil en PERCENTILE` · `34 / 8 / 0 / 0 / 0 / 0` · `036c102a` · `03af54c3` · `NEGATIF PROPRE` · `0.00302` · `bandes DIVERGENTES` -> 7ter.100 (7) |
| **CE QUI ELIMINE LE DOUBLON ET LE PIETON QUI LONGE (l offset rigide)** | `OFFSET RIGIDE` · `0/85` · `52 bits bruts` · `distance EXACTEMENT nulle` · `tmp_vcref` · `vcref pairs` -> 7ter.100 (3) |
| **LES DEUX BANDES SATUREES, ET LA VERIFICATION CROISEE QUI DEVIENT VIDE** | `253 slots sur les 253` · `512..764` · `Les deux bandes sont pleines` -> 7ter.100 (4) et §18.1 (a) |
| **LA DUREE REELLE : 22 TRAMES = 0.75 s** | `29.5 trames/s` · `CO-LOCALISATION SEULE` · `5.34 s` · `vcref pairs` -> 7ter.100 (9) |
| **L AVEUGLEMENT DU DETECTEUR A UN PASSAGER HORS PHASE** | `hors phase` · `dispo = 0` · `DEUX trames consecutives` · `zero d occupation OU de cadence` -> 7ter.100 (8) |
| **L ACCORD API NOM PAR NOM (controle du kill-feed avec de vrais degres de liberte)** | `NOM PAR NOM` · `8/8` · `xuid_aliases` · `2/8!` · `JAMAIS AU-DESSUS de l API` -> 7ter.100 (11) |
| **OU EST EXACTEMENT LA CONFIANCE D UNE LIGNE (formule CORRIGEE)** | `tolMS = 2500` · `4.04 s` · `17 des 55` · `la ligne entiere` · `matchExact` -> 7ter.100 (12), corrige 7ter.96 (4) |
| **LE RECENSEMENT DES FILMS A VEHICULES, 150 FILMS** | `53 portent` · `census 150` · `256, 230, 200` · `bas absolu de la distribution` -> 7ter.100 (bloc index, point 10) |
| **Le registre ne porte PAS la calibration par carte** | `L(i0) = 0` · `947 films` · `2 SIGNATURES` · `Archetype.Flags` · `W = 6+L` · `tmp_vcaud flags` -> 7ter.97 (2)(b) |
| **`vehicle-weapon-set-component` est bien a `i38`** | `i38` · `48 composants` · `premier absent i30` -> 7ter.97 (7)(b) |
| **LA CALIBRATION PAR CARTE DE `i0` (le sous-produit qui deborde le sujet)** | `bipedAxisWidths` · `QuantRangeCEBiped` · `(18,16)` · `(15,15)` · `(17,17)` · `continuite` · `sweepveh` -> 7ter.90 (3)(4) **et sa correction** 7ter.94 (7), et 7ter.54 AXE3. ⚠ Le motif `rang 1 sur 1608` mene a un chiffre **REFUTE** : l outil imprime `rang 6/1608`, l espace effectif est de **144 couples**. Motifs de la correction : `rang 6/1608` · `144 couples` · `premiere FAMILLE` |
| **Le TROU de trajectoire (prediction du modele parente)** | `tmp_vkcal hole` · `candidats passagers` · `9 disparaissent` · `0/11` -> 7ter.90 (8) |
| **Le vehicule comme entite du film** | `ti=40` · `archetype 40` · `[768..1023]` · `bande VIDE` · `SATURATION` · `vcen` · `vpos` · `tmp_vehparent` · `lien joueur-vehicule` · `9 observations` · `controle decale d un bit` -> 7ter.89 (6)(7)(8)(9) et index §2.5bis. ⚠ `tmp_vehcensus` **ne rend RIEN dans le RE_LOG** : cet outil appartient au lot `vh`, jamais publie. |
| **Le NOMMAGE d une arme de vehicule (sujet SANS RAPPORT)** | `R-VEHICULE` · `vcdd` · `uwfa` · `sofa` · `62 weap` · `_tur_` · `_veh_` -> 7ter.45/46/47/48 et index §2.5 |
| **Le composant d accrochage** | `object-parent-state-component` · `quantStatDefaultWidth` · `DAT_144706100` · `FUN_1406d3140` -> 7ter.89 (7)(9) et 7ter.88 (1) |
| **Ne PAS confondre les deux `i0`** | `FUN_14076e29c` (= `ti=41`, le PROJECTILE, 45/60) · `FUN_1406cfe44` (= `ti=40` et `ti=35`, dynamic-precision) -> 7ter.84 (7) et 7ter.89 (7) |
| **Le scan de motif comme localisateur (MORT)** | `1/8366` · `292 candidats` · `193/292` · `TryDeltaAt` dans le RE_LOG -> 7ter.27 (6) ; la ligne de synthese `Scan de motif comme localisateur` est dans **cet index, §3** (elle n existe pas telle quelle dans le journal). |
| **L absence de composant de siege** | `mount` · `rider` · `occup` · `passenger` · `gunner` · `embark` · `player-desired-respawn-seat` · `vehicle-seats-override` -> 7ter.89 (9) |

---

## 19. LE CORPS DES CODES 6 ET 7 — CE QU IL CONTIENT VRAIMENT, ET POURQUOI LA PRECISION PAR ARME NE SORTIRA PAS DE LA (2026-07-28, sections 7ter.91 et 7ter.94)

> **A LIRE AVANT DE ROUVRIR << LES TOUCHES PAR ARME >>.** L axe etait la demande explicite de
> l utilisateur : << Pour les touches par armes, qui permettrait d avoir la precision pour toutes
> les armes, je suis interesse. >> La reponse est **non par cette voie**, et elle est mesuree, pas
> raisonnee. Motifs de grep : `7ter.91` · `7ter.94` · `c7b` · `tp.ref` · `evFixed[7] = 118` ·
> `b0 == 1` · `168 380` · `H(f1)` · `tmp_c7b` · `tmp_refc7` · `reprise de chaine` ·
> `0.2029` · `36 - P(carte)`.
>
> **VERIFICATION ADVERSARIALE FAITE (7ter.94, lot `tp.ref`, binaire `cmd/tmp_refc7`).** Le
> negatif TIENT, reproduit **a l unite** par un binaire ecrit separement : memes 97 988 et
> 70 392, meme **zero** sur 168 380, controle positif 70 950/70 950. **Ce qui a change** : la
> longueur `118` est desormais mesuree DANS LE FILM (§19.1) et la map-dependance du code 6, que
> 7ter.91 (7) declarait non mesurable, est **MESUREE** hors echantillon (§19.1, point 2).

### 19.1 CE QUI EST ACQUIS — LA GRAMMAIRE DU CORPS, ET LA RECONCILIATION DES DEUX LONGUEURS

Le brief posait un premier mystere : les longueurs `evFixed[6] = 93` et `evFixed[7] = 118` ne se
deduisent pas l une de l autre — ecart 25 pour 19 bits de champs supplementaires, **il manque
6 bits**. **RESOLU** : les deux codes n emploient PAS le meme lecteur de position.

```
  code 7  (FUN_142f1c6cc)   b0 R(1) | [tag R(1)+R(32) | var R(32)] | f1 R(7) | f2 R(7) |
                            n1 R(19) | sel R(2) | pos 3 x R(12) FIXES | n2 R(19) |
                            mat R(9) | w16 R(16) | b1 R(1) | b2 R(1)
                            SOMME (branche sans tag) = 1+7+7+19+2+36+19+9+16+1+1 = 118
  code 6  (FUN_1410f03b4)   MEME suite, moins sel, w16 et b1 (= -19 bits), et surtout avec
                            FUN_14076e524 au lieu de FUN_140c1e924 : R(1), puis un index de
                            `DAT_144632be0` bits, puis 3 axes dont les LARGEURS SONT LUES DANS
                            UNE TABLE (`DAT_1445ccbe0`) installee au chargement.

  118 - 93 = 25 = 2 (sel) + 16 (w16) + 1 (b1) + 6 (ECART DE LECTEUR DE POSITION)
```

Chaque largeur est un immediat du desassemblage (`MOV EBX,0x7` en 142f1c746, `LEA R14D,[RBX+0xc]`
en 142f1c77f, `LEA R11D,[R14-0x11]` en 142f1c7a2, `MOV R9D,0xc` en 142f1c85c, `MOV R11D,0x9` en
142f1c889, `CMP EAX,0x10` en 142f1c94a). **Aucune n a ete ajustee** : `evFixed[7]` a ete mesuree en
7ter.25 contre l oracle du dispatcher sans connaissance du corps, et les onze largeurs sortent du
binaire sans connaissance de `evFixed`. Les deux se rencontrent **a l unite**.

⚠⚠ **CORRIGE PAR 7ter.94 (5)(6), 2026-07-28 — TROIS CHOSES CHANGENT ICI.**

1. **LA LONGUEUR `118` EST DESORMAIS MESUREE DANS LE FILM**, et pas seulement par une somme
   d immediats confrontee a `evFixed[7]`. Le test : apres un code 6/7 de profondeur 0, avancer
   de `L` bits et demander si le flux **REPREND sur un en-tete de code 6 ou 7** (hasard
   0.015625). **`L = 118` rend 0.2029 ; NEUF des dix voisins immediats rendent 0.0000.** C est
   la troisieme jambe, et la seule qui interroge le film.
2. **LA LARGEUR DU BLOC DU CODE 6 EST MESUREE, ET ELLE EST MAP-DEPENDANTE** — le test de
   7ter.91 (7) etait mort, la question ne l etait pas. Meme instrument (une POSITION, pas une
   largeur de saut), hors echantillon par SHA-256, **25/31** d egalite exacte (fond 1/31), avec
   son controle de localisation sur le code 7 a **26/36** de la valeur certaine 118. Largeurs
   `P` par carte : Bazaar 40 · Behemoth 33 · Catalyst 35 · Chasm 43 · Cliffhanger 30 ·
   Command 30 · Detachment 30 · Empyrean 30 · Forest 29 · Fortitude 24 · Fragmentation 26 ·
   High Ground 24 · Illusion 36 · Kaiketsu 30 · Obituary 24 · Prism 32 · Recharge 41 ·
   Refuge 30 · Smallhalla 24 · Streets 26 · The Pit 30 · Threshold 24. Corroboration non
   construite pour ca : les trois paires `X` / `X Heavies` (populations de films disjointes)
   rendent la meme largeur, **3/3**.
3. **LES << 6 BITS MANQUANTS >> NE SONT PAS UNE CONSTANTE.** Le MECANISME — deux lecteurs de
   position differents — tient. L ecart vaut **`36 - P(carte)`**, de **-7** (Chasm) a **+12**
   (Fortitude, High Ground, Threshold), et l egalite `118 - 93 = 25 = 2+16+1+6` n est valable
   que sur les cartes ou `P = 30` (sept des vingt-deux tranchees). **`evFixed[6] = 93` est bien
   une longueur MODALE — et la classe modale est maintenant nommee.**

⚠ **LES LARGEURS PAR CARTE PUBLIEES EN 7ter.91 (7) SONT REFUTEES** (3 concordances sur 11) :
elles viennent du balayage d entropie que la section declare elle-meme invalide. **Ne pas les
citer** ; la liste qui fait foi est celle du point 2 ci-dessus.

### 19.2 LE FAIT QUI FERME L AXE

**Le tag n est jamais transmis.** Le bit `b0` qui commande sa lecture vaut **1** dans
**97 988 codes 7 et 70 392 codes 6 de profondeur 0** — le premier evenement de son paquet, seule
position qui ne depende d aucune longueur de corps — **sans une seule exception sur 168 380
observations**. Les 3 857 cas `b0 == 0` sont tous a profondeur >= 1 (derives de marche), et
**aucun** de leurs 2 274 tags n appartient au catalogue `jpt!` ni au catalogue de familles d arme,
enumere sans presumer la moitie basse.

Les deux controles qui rendent ce negatif publiable :

| CONTROLE | VALEUR ALTERNATIVE ECRITE D AVANCE | RESULTAT |
|---|---|---|
| **POSITIF, meme instrument** — reecrire un tag `jpt!` connu dans le corps REEL, puis relire | un instrument aveugle rend 0.0000 | **129 381 / 129 384 = 1.0000** |
| **ALIGNEMENT** — relire le corps a -3..+3 bits, entropie par champ | si rien ne bouge, la mesure ne vaut rien | `H(f1)` = **0.4745** a 0 ; +1 1.4714, +2 2.1974, +3 2.8252, -1 6.9619 — **minimum net, croissance monotone** |

**LE TEST DECISIF DU BRIEF REND UN ZERO EXACT** : aucune arme ne bouge d un centieme, ni
projectile ni hitscan (Needler 0.0070 -> 0.0070 · Bulldog 0.0000 -> 0.0000 · MA40 0.4292 ->
0.4292 · BR75 0.3988 -> 0.3988). Ce n est pas un gain uniforme — c est un zero. Et le
denominateur par arme se reproduit **a la 4e decimale** contre 7ter.86 (1), donc la chaine de
mesure n a pas echoue par panne.

### 19.3 LES TROIS VOIES SONT FERMEES, CHACUNE PAR UNE MESURE

| CE QU IL FAUDRAIT | ETAT | SECTION |
|---|---|---|
| l **ARME**, dans l evenement d impact | **FERMEE** — tag jamais transmis, et aucun autre champ du corps n est un identifiant d arme | 7ter.91 (2)(5) |
| le **TIREUR**, dans l evenement | **FERMEE** — `FUN_142eed4e8` transmet la CIBLE et l OBJET PROJECTILE, jamais le tireur | 7ter.86 (5)(a) |
| un **PONT** projectile -> tir | **FERMEE** — aucun autre evenement ne nomme l objet-projectile (fire-event 0.0931 contre 0.0711 permute) | 7ter.88 (4), corrigee 7ter.89 (3) |

**SUR LA REFERENCE EXTERNE** : le brief demandait de chercher si le total par joueur (toutes
armes) pouvait remplacer la population << arme dominante >= 80 % >>, trop pauvre en armes a
projectile (Needler : 122 observations a >= 50 %, **2 a >= 80 %**, 0 a >= 90 %). **La reponse est
non, et pour une raison structurelle et non statistique** : un total PAR JOUEUR suppose de
rattacher chaque impact a un joueur, et le code 7 ne nomme que la CIBLE. La seule granularite
atteignable reste **le FILM**. Le probleme n est pas la taille de l echantillon de reference,
c est l absence de la variable a ventiler.

### 19.4 CE QUE LE CORPS PORTE A LA PLACE — ET LA SEULE PORTE LAISSEE ENTROUVERTE

```
  f1  7 bits, DEQUANTIFIE EN FLOTTANT (`MOVSS [RSI+0xc],XMM0`)   127 : 0.9574   H = 0.4745
  f2  7 bits, flottant                                            0 : 0.3458 · 127 : 0.1837
  sel 2 bits, indexe une table de reperes                          1:72 990 · 0:54 033
  mat 9 bits, stocke `- 1` (materiau / region)                    32 : 0.4537 · 131 : 0.3085
  w16 16 bits — 2 199 valeurs distinctes, PETITS ENTIERS (0/19/1/61) : PAS un identifiant d arme
  b1  1 dans 0.8132        b2  0 dans 0.8184
```

⚠ **PISTE NOTEE, NON MESUREE** : `f1` et `f2` sont **deux scalaires flottants par impact**, dans
le seul evenement du film qui marque une application de degat sur une entite, alors que 7ter.75 a
etabli que le kill-event porte deux **parts de degats en pourcentage entier**. C est la forme d un
**flux de degat PAR TOUCHE**. Rien n a ete mesure la-dessus. ⚠ **Et la reserve va avec la piste** :
la CIBLE etant nommee alors que le tireur ne l est pas, un tel flux compterait les degats
**RECUS** — qui n ont **aucune reference API**. Voir aussi la note de memoire projet
`project_damage_stream_per_hit_opportunity`.

### 19.5 LES DEUX LECONS DE METHODE DE CE LOT

1. **UN BALAYAGE DE LARGEUR N EST PAS UN BALAYAGE DE POSITION, ET SEUL LE SECOND EST UN CONTROLE.**
   Decaler le POINT DE DEPART du corps deplace tous les champs a la fois et detruit l information :
   c est un vrai controle negatif. Changer la LARGEUR D UN SAUT ne detruit rien, elle deplace un
   champ — qui peut tomber dans une zone de bits calmes et voir son entropie **baisser**. Mon test
   de map-dependance, pourtant valide hors echantillon dans les deux sens (11/12 contre un fond de
   1/15), est mort de cela : le meme protocole applique au code 7, dont la largeur est **certaine**,
   reproduit **15/18** un argmin **faux**.
   ⚠ **COMPLETE PAR 7ter.94 (5)(6) — LA LECON EST PLUS FINE QUE << UN BALAYAGE DE LARGEUR EST
   MORT >>.** Ce qui etait mort, c est le **DISCRIMINANT** (une statistique du CONTENU, ici une
   entropie), pas le balayage. En changeant de discriminant pour une propriete **STRUCTURELLE**
   — la chaine reprend-elle sur un en-tete d evenement valide ? — le meme balayage de largeur
   **passe** son controle de localisation (**26/36** de la valeur certaine 118 contre 15/18 d une
   valeur fausse) et rend la map-dependance hors echantillon a **25/31**. **REGLE A RETENIR :
   avant de declarer un instrument mort, demander si c est le balayage ou le discriminant qui
   l est. Un discriminant structurel se trompe rarement ; un discriminant statistique se laisse
   deplacer.**
2. **UN NEGATIF SE PUBLIE AVEC LE CONTROLE POSITIF DU MEME INSTRUMENT, ET LA REECRITURE EST LA
   FORME LA MOINS CHERE.** Injecter la valeur cherchee dans la donnee reelle, a la position
   predite, puis relire avec le meme code : 4 lignes, aucun degre de liberte, et cela transforme
   << je ne trouve pas >> en << il n y en a pas >>.

---

## 20. LES DEUX INDICES DE JOUEUR DU PIPELINE DE PRODUCTION (2026-07-28, sections 7ter.92, 7ter.95 et 7ter.99)

> ⚠ **TROISIEME ROUTE, SANS ORACLE ET SANS ARME (7ter.99, label `rc.perm`, 2026-07-28) — ET LA
> REPONSE A UNE ALERTE VENUE DU WORKTREE VOISIN.** Le chantier replay 2D a publie que
> << l index d attaquant du record n est pas l index de joueur du roster >> et que cela
> << detruit toute confrontation film contre base >>. **NOS MESURES PAR JOUEUR SONT IMMUNISEES, ET
> C EST MESURE** : sur son film temoin `000d5950`, sa bijection est reproduite **8 / 8** par la
> FONCTION LIVREE (`weaponv3.ResolveXuidToPIAllStrings` sur les chunks de type 2) **et** par
> l ancrage strict des lots `gt.film` / `gt.ref` / `pv` ; l ordre de la base rend **1 / 8**. Sur
> 949 films les deux routes s accordent **9 956 / 9 956** (tables entieres **936 / 936**), quand
> l ordre de la base rend **0.0903** contre une nulle a **0.0939** — indiscernable du hasard, sur
> une quantite qui n a rien a voir avec celle de `co.pi`. **TROIS AJOUTS.**
> *(ajout 1)* **L ARTEFACT DE `ResolveBest` N EST PAS DANS LA FONCTION, IL EST DANS L ENTREE.** La
> mise en garde de `cmd/tmp_gtref/pi.go` (<< les 8 xuids rendent tous 0 >>) est vraie et mal
> attribuee : le coupable est le **chunk 0, de type 1**, que ce binaire de RE lit et que la
> production ne voit JAMAIS (`GetMatchFilm` ne remonte que `filmChunkTypeReplicationData`). Meme
> fonction sur les chunks de type 2 : **8 / 8, bijection**. Meme fonction de production sur TOUS
> les chunks : **1 seul indice distinct, 0 / 8**.
> *(ajout 2)* **LE TEST DECISIF DE L APPARIEMENT** : meme decodage, seule la table change — la
> correlation par joueur du taux de touche tombe de **0.6521 a 0.3808** (nulle 0.3993) si on
> applique l ordre de la base. Population mobile isolee : **0.6579 contre 0.3662**.
> *(ajout 3)* **MAIS LES DEUX AGREGATS DE 7ter.86 SONT INVARIANTS PAR PERMUTATION** (0.3649 du bras
> publie contre 0.3653 de la NULLE) : ils somment sur tous les indices. **Le `0.3648 contre 0.3664`
> ne prouvait rien sur l appariement, dans aucun sens.**
> ⚠ **UNE DIFFERENCE DE POLITIQUE, MESUREE ET NON CORRIGEE** : sur 947 films, la fonction de
> production produit **5 COLLISIONS** (deux joueurs sur le meme indice) et 0 resolution
> incomplete ; l ancrage strict produit **0 collision** et **11 films ECARTES**. Les 5 collisions
> sont TOUTES parmi ces 11 films — **0 / 936** ailleurs. `piUnresolved` couvre l ABSENCE d indice,
> pas la COLLISION. Aucune de nos mesures par joueur ne passe par la fonction de production, donc
> aucune n en souffre aujourd hui.
> ⚠ **CE QUI RESTE** : `main` porte TOUJOURS `getXuidToPI` (`ORDER BY team_id, rank`) — le
> correctif de 7ter.92 n existe que sur `feat/filmdec-killweapon`.

> ⚠ **VERIFIE PAR UN SECOND AGENT (7ter.95, label `co.ref`, 2026-07-28) — CE QUI CHANGE ICI.**
> **LA CONCLUSION PASSE A `[ETABLI]`** : instrument ecrit de zero (`cmd/tmp_coref/`), bras FILM
> branche sur la FONCTION LIVREE et non sur une reimplementation, corpus plus grand (**239 films,
> 16 411 kills**) et **films a resolution INCOMPLETE INCLUS** — ordre de la base **22.820 %**,
> nulle **21.304 %**, film **77.040 %**, `z = 92.788`, le film gagne **237 / 239 films**.
> **TROIS AJOUTS ET DEUX CORRECTIONS.**
> *(ajout 1)* **LE TEST DE LOCALISATION PAR LE TUEUR, que 7ter.92 n avait pas fait** : sur les
> kills dont le tueur avait DEJA le bon indice par l ordre de la base, les deux bras rendent
> **exactement le meme entier** — 1 736 / 2 284, 76.007 % des deux cotes, et memes sous-totaux par
> voie ; sur les autres, 14.221 % -> 77.207 %. **Zero mouvement sur la population inerte : ce
> n est pas un facteur d echelle.**
> *(ajout 2)* **UNE ROUTE SANS ORACLE** : l ordre de la base designe le bon joueur **11.769 %** du
> temps (815 / 6 925) contre **11.137 %** attendus au pur hasard — et **3.556 % contre 3.762 %**
> au-dela de 16 joueurs. 949 films, aucun oracle, aucune arme.
> *(ajout 3)* **LA RESERVE DE §20.2 EST FERMEE** : sur la population entiere, sentinelle comprise,
> le bras livre rend **77.040 %**, soit MIEUX que sur les seuls films complets (76.304 %). La
> perte de couverture ne coute rien au taux.
> *(correction a)* **LA TABLE DE 7ter.92 (2) NE SE REJOUE PAS AVEC LE CODE LIVRE** : `copi oracle`
> ne restreint plus aux films complets. Le chiffre est bon (sous-total `co.ref` : 76.304 % contre
> 76.384 % publie), le chemin a disparu.
> *(correction b)* **<< LA NULLE FAIT MIEUX QUE LA PRODUCTION >> EST `[REFUTE]` COMME FAIT** : le
> sens s inverse au second instrument. Ecrire *indiscernable*, jamais *pire que*.

> Lot `co.pi`, outil `apps/go-api/cmd/copi/`. **Il s agit du CONSOMMATEUR, pas du decodeur** : ce
> lot ne craque rien, il mesure ce que le pipeline `sync/backfill_weapons.go` fait de deux champs
> deja connus. C est la premiere fois qu une correction de ce chantier est jugee par une
> REFERENCE INDEPENDANTE (`killsource`) et non par un taux d attribution.

| QUESTION | REPONSE | SECTION + STATUT |
|---|---|---|
| D ou venait l indice du TUEUR dans le pipeline de production ? | De l **ORDRE DE LA BASE** (`ORDER BY team_id, rank`) — une permutation arbitraire du roster, sans rapport avec le film. C est ce qui rendait INERTE le correctif de largeur de 7ter.78. | 7ter.92 (0) · `[MESURE]` |
| Cet indice valait-il quelque chose ? | **NON, ET IL FAIT MOINS BIEN QU UN TIRAGE AU SORT.** Accord d egalite exacte avec l oracle `killsource`, 116 films 4v4, 8 236 kills, memes denominateurs : ordre de la base **22.268 %**, permutation AU HASARD **22.658 %**, indice lu dans le film **76.384 %**. McNemar 4 517 contre 60 (`z = 65.88`), et le film gagne sur **116 films sur 116**. | 7ter.92 (2) · `[MESURE]` |
| Le gain est-il localise ? | **OUI, exactement la ou l indice sert.** `CorrelateKillsGlobal` ne l utilise qu au filtre `ev.PlayerIndex == killerPI` : la voie fire-event passe de 45.0 % a 97.5 %, la voie `formula_a` (aucun filtre) n ameliore pas (5.0 % -> 2.1 %). ⚠ Les taux PAR VOIE ne sont pas comparables terme a terme — la population de chaque voie change d un bras a l autre. Le comparable est le taux global et McNemar. | 7ter.92 (3) · `[MESURE]` |
| Que devient un participant que le film ne nomme pas ? | Il recoit une **sentinelle -1**, hors de l image du champ (5 bits), donc **aucune arme** plutot que celle d un autre. Portee chiffree : en 4v4, **56.87 %** des films nomment tous leurs participants et **7.531 %** des participants n ont aucun indice ; au-dela de 16, 86.29 % et 1.941 %. | 7ter.92 (4) · `[MESURE]` |
| Le champ d indice de `ScanFormulaA` (3 bits) est-il tronque lui aussi ? | **LA QUESTION EST MAL POSEE.** Le premier bit lu vaut 1 dans **99.4 a 99.8 %** des resultats, toutes tailles de lobby (291 films, 67 589 resultats) : ce n est pas un bit d indice, et le critere de bornes qui avait tranche pour le fire-event est **VIDE** ici, dans les deux sens. | 7ter.92 (5) · `[MESURE]` |
| Alors que faut-il corriger sur `ScanFormulaA` ? | **PAS LA LARGEUR : CE QU IL ACCEPTE.** Sur 67 589 resultats, **54 (0.08 %)** portent un identifiant du catalogue d armes ; 389 identifiants distincts contre 25 pour les fire-events, intersection 0.06 %. La cause se lit dans la boucle : les 8 octets sont valides contre `WeaponBytesMap` pour tous les suffixes SAUF `CommonWeaponSuffix`. Cout deja en base : **1 512 des 4 264** lignes armees de la voie `formula_a` portent un identifiant hors catalogue (35.46 %), contre **5 sur 89 284** pour la voie `fire_event`. | 7ter.92 (7) · `[MESURE]` |
| Un autre depart ou une autre largeur sauverait-il ce champ ? | **NON.** Confronte au meme oracle (donc avec controle positif : 97.5 % sur la voie fire-event dans le meme run), aucun candidat de la couche brute ne produit **une seule** egalite, et tous ceux de la couche decalee tiennent dans leur propre permutation. | 7ter.92 (8) · `[MESURE]` |
| Combien coute la lecture de l indice dans le film ? | **Mediane 5 ms, p90 85 ms, max 160 ms** par match pour 22.9 Mo balayes en moyenne (60 films). L arret anticipe des que tout le roster est trouve explique la mediane. Negligeable devant le telechargement du film. | 7ter.92 (0)(9) · `[MESURE]` |

### 20.1 LA LECON DE METHODE DE CE LOT

**UN NEGATIF DONT LA NULLE VAUT ZERO EST UN NEGATIF SUR L INSTRUMENT, PAS SUR LA QUESTION.** Le
premier controle du volet B (confronter `ScanFormulaA` au fire-event, egalite exacte de l id
d arme) rendait **0 / 1 187** pour tous les candidats — de quoi ecrire << le champ n est pas
l indice >>. **Sa nulle rendait zero elle aussi (0.046 %)**, parce que les deux instruments ne
partagent pas leur vocabulaire d armes. Le controle ne pouvait pas reussir ; il mesurait sa propre
cecite. La regle 10 (§3bis) exigeait deja un controle positif du meme instrument : **une nulle a
zero en est le symptome le plus lisible, et se traite comme une alarme, jamais comme un succes.**

### 20.2 CE QUE CE LOT N A PAS FAIT

```
validation de catalogue     le vrai defaut de `ScanFormulaA` est nomme et chiffre, PAS corrige :
sur CommonWeaponSuffix      le corriger deplace le 4v4, donc il exige sa propre passe d oracle
au-dela de 16 joueurs       AUCUN signe mesure — l oracle n y est pas publiable ligne par ligne
                            (7ter.53). Le mecanisme est le meme, l affirmation ne l est pas.
la voie `formula_a`         reste branchee telle quelle : elle sert 19 505 kills et n en nomme
                            correctement que 2 752 (14.1 %). Constat, pas correctif.
films a resolution          FERME PAR 7ter.95 : le bras LIVRE sur toute la population 4v4
INCOMPLETE                  (sentinelle comprise) rend 77.040 %, MIEUX que sur les seuls films
                            complets (76.304 %). La perte de couverture ne coute rien au taux.
AUCUN RE-BACKFILL           `weapon_kills` porte encore les attributions de l ordre de la base :
                            89 290 lignes `fire_event` dont l indice du tueur venait d une
                            permutation arbitraire. TOUJOURS VRAI apres 7ter.95 — rien n a ete
                            ecrit en base. **C EST LE SEUL RESTE A FAIRE POUR L UTILISATEUR.**
```

### 20.3 CE QUE LA VERIFICATION A TROUVE AILLEURS (7ter.95)

```
`co.re` point (8)     le controle de grammaire << 3 537 / 3 537 >> est une IDENTITE ALGEBRIQUE :
                      l ecart au StartBit suivant EST le nombre de bits consommes par
                      `consumeWeaponStateAmmo`, que `decodeAmmo` reimplemente. [REFUTE] comme
                      controle. La CORRECTION de RECETTE_LOADOUT §4bis (largeur 22 majoritaire,
                      << et >> et non << ou >>) tient : observation directe de deux bits, plus le
                      deserialiseur de production (deux `if` successifs).
piege d instrument    `weapon_kills.weapon_id` est un UBIGINT : un `sql.NullInt64` fait ECHOUER
                      le Scan sur les identifiants > MaxInt64, et un `continue` sur cette erreur
                      fabrique un total faux de 36 % SANS aucun message. Scanner en `uint64`.
```

---

## 21. UN ENREGISTREMENT DE DEGAT EST UN TIR, SON TABLEAU A EST LA TOUCHE (2026-07-28, section 7ter.98)

> **RENVOI EN TETE (2026-07-28, §22, section 7ter.101, label `rc.ref`) — LE TITRE DE CETTE
> SECTION EST TROP GENERAL.** L unite du record **depend de l arsenal** : c est le TIR sur arme a
> trace instantanee (Tactical : 45 egalites `record == shots_fired` contre 14.3 permutees, 0/200)
> et la **TOUCHE** en **FIESTA** (174 egalites `record == shots_hit` contre 134.7 permutees, max
> 163, 0/200) et en BTB (43 contre 29.1, 0/200). **Le film temoin du chantier voisin est un
> Fiesta : sur sa famille de mode, sa lecture est la bonne**, et le `[REFUTE]` porte plus bas
> contre son << 87 % des touches >> vaut comme enonce de FORMAT, pas comme verdict sur sa mesure.
> Deux corrections de comptage vont avec (§22.6) : l ecart de 1.14 % de §21.4 n est **pas** la
> borne `s34` mais la **condition d indice** (98 685 porteurs TOUS contre 97 556 INDEXES), et les
> records du corpus valent **1 799 630 BRUTS contre 1 774 183 INDEXES**. Tout le reste de cette
> section se reproduit **a l unite** par un quatrieme binaire.

> **A QUOI SERT CETTE SECTION.** Le chantier voisin (replay 2D, worktree `filmdec-continuation`)
> a publie le 2026-07-28 que le film << ne porte pas les tirs mais les records de degat >>, avec
> un plafond a << environ 87 % des tirs QUI ONT TOUCHE >>. Ce chantier-ci dit depuis 7ter.81 qu un
> enregistrement code-36 est un TIR. **Les deux ont chacun une moitie de la reponse**, et le
> desaccord venait de deux fautes cumulees : un DENOMINATEUR (touches au lieu de tirs) et un
> CORPUS (un seul film, et c est un Fiesta au 5e percentile).
> **Motifs de grep** : `rc.unite` · `7ter.98` · `taux de remplissage` · `tableau A` · `type 105`.

### 21.1 LES DEUX VOCABULAIRES DESIGNENT LE MEME ENREGISTREMENT — `[ETABLI]`

```
  voisin : pay[0] >> 1 == 105                        (7 bits de POIDS FORT de l octet 0)
  ici    : bits(pl,1,1)==1 && bits(pl,2,7)==36       (bit de continuation, puis bits 2..8)
```

`0xD2` satisfait les DEUX. Mesure sur `000d5950` : `832` type-105 dont `519` longs (0xD2) et
`313` courts (0xD3) ; `519` code-36, **tous** de premier octet `0xD2`, **zero** ailleurs. Sur
**927 films** : `1 799 630 = 1 799 630`. La variante COURTE `0xD3` est un autre code (38/39) que
ce chantier n a jamais compte — et elle **n est pas** la touche (rapport aux touches 0.18 en
Tactical, 0.46 en Fiesta).

### 21.2 LE TABLEAU A QUATRE CASES, ET LA PORTEE DU 0,87

```
  film 000d5950 (Super Fiesta:Slayer)   / tirs API   / touches API      corpus 927 films (mediane)
  records code-36            519          0.2329       0.8723           0.7495   /   1.7260
  porteurs (tableau A plein)  97          0.0435       0.1630           0.2857   /   0.7045
```

`000d5950` est **sous le p05 des deux distributions** et son rang est **26 / 927**. La mediane
`records / touches` vaut **1.7260** : il y a 73 % de records EN TROP par rapport aux touches, donc
un record ne peut pas etre une touche. **Le 0,87 du voisin est exact sur son film et faux comme
regle.**

### 21.3 LE TEST QUI SEPARE, ET POURQUOI LA CORRELATION NE SUFFIT PAS

Les deux hypotheses predisent chacune **UNE correlation nulle ET UNE forte** : il faut lire le
COUPLE, pas la seule qui est plate. Bornes calculees sur les donnees (Tactical, n = 138) :
`record = c x TIRS -> (0.0000, -0.7517)` ; `record = c x TOUCHES -> (+1.0000, 0.0000)` ; mesure
`(+0.2182, -0.1539)` — **aucune des deux**, la correlation ne tranche pas.

**C EST LE NIVEAU QUI TRANCHE, SUR UNE POPULATION A ARME UNIQUE HITSCAN** (Tactical Slayer,
BR75 seul, n = 138) :

```
  precision API du joueur          med 0.2777
  records / shots_fired            med 0.9300      (prediction si TIR : ~1)
  records / shots_hit              med 3.4655      (prediction si TIR : 1/p = 3.601)
                                                   (prediction si TOUCHE : ~1)
  porteurs / shots_hit             med 0.8509      r(port/touches, precision) = +0.0383
```

**Si un record etait une touche, un joueur de Tactical aurait 93 % de precision.** Egalite exacte :
`records == shots_fired` 3/138, `records == shots_hit` **0/138** ; a `|d| <= 5`, **45 contre 1**.

**LA FORME LA PLUS DIRECTE** (population hitscan, records >= 20, n = 2 185) : le **taux de
remplissage** `porteurs / records` vaut **0.4267** contre une precision API de **0.4462**, erreur
absolue mediane par joueur **0.0266**, `r = +0.8204`.

> **C EST LE LIVRABLE PRODUIT DU DOSSIER, ET IL A ATTERRI DANS UN GUIDE LE 2026-07-28** :
> `.ai/GUIDE_WEAPON_SHOTS.md` **§3quater**. Cette forme **ne demande NI l API NI un denominateur
> choisi** — c est ce qui la separe de tout le reste de ce chantier. Le guide y porte aussi ses
> deux reserves (controle zero **19/34**, deficit de **7 %** sur les records et **15 %** sur les
> porteurs) et **tranche explicitement** ce qu elle change pour la precision PAR ARME : **rien**
> (§3quater.4 — flux different, grain different, question differente).

### 21.4 LES CONTROLES, ET LES LIMITES A NE PAS PERDRE

```
FOND PERMUTE (200 tirages intra-film, hitscan n = 2 225) — le tableau se lit EN CROIX :
  records == shots_fired |d|<=5     reel 109   permute 78.5  (max 98)    -> 0/200
  records == shots_hit   |d|<=5     reel  12   permute 52.8              -> SOUS le fond
  porteurs == shots_hit  |d|<=5     reel 251   permute 182.0 (max 210)   -> 0/200
  porteurs == shots_fired |d|<=5    reel   3   permute 25.7              -> SOUS le fond
HORS ECHANTILLON (SHA-256 du prefixe, puis inversion) : med(port/touches) 0.8128 / 0.8153
CONTROLE ZERO : shots_hit == 0 -> zero porteur seulement 19/34 (0.5588), films sans fantome
```

**LIMITES OBLIGATOIRES** : deficit de 7 % sur les records et de 15 % sur les porteurs, **non
expliques** ; `porteurs / shots_hit` **n est pas publiable comme un COMPTE** de touches, seul le
TAUX l est ; ecart non explique de 1.14 % entre mon compte de porteurs (97 556) et celui de
7ter.86 (1) (98 685) sur les memes 150 films, alors que les records se reproduisent a l unite.

### 21.5 LA BIJECTION INDICE -> JOUEUR N EST PAS EN CAUSE — `[ETABLI]`

`cmd/tmp_rcunite gt 000d5950` reproduit **8/8** la table publiee par le voisin
(`0 whiteknight2519 · 1 JAVIERLOLITO540 · 2 JGtm · 3 LORD PEINX13 · 4 IKE ILYA ·
5 Akatsuki fire17 · 6 aldusbroncus · 7 VitaminA1688`). Notre ancrage (`resolveXuidPIStrict`,
ecrit par `gt.ref`) est **film-seul** : motif xuid cherche au bit pres, 5 bits precedents, indices
deux a deux distincts. **Aucun ordre de roster n y entre**, donc aucune de nos mesures par joueur
n est affectee par la permutation. Corollaire mesure et utile : l ancrage NAIF
(`weaponv3.ResolveBest`, premiere occurrence) rend l indice **0 pour les 8 joueurs** de ce film —
c est exactement le piege que le voisin decrit, et c est pourquoi cet ancrage-la n est plus
utilise pour les mesures. Ce qui reste a auditer (sites de PRODUCTION construisant
`xuid -> indice` depuis un ordre de base) est l objet du lot `rc.perm` (7ter.99).

---

## 22. L UNITE DU RECORD DEPEND DE L ARSENAL — LA VERIFICATION ADVERSARIALE DE §21 (2026-07-28, section 7ter.101)

> **A LIRE AVEC §21, ET AVANT DE CITER SON TITRE.** §21 conclut << un enregistrement de degat est
> un TIR >>. C est vrai **sur arsenal a trace instantanee**, et **faux en Fiesta et en BTB**, ou le
> meme enregistrement suit les TOUCHES. §21 a donc commis sur son propre resultat la faute de
> portee qu il reproche au chantier voisin.
> **Motifs de grep** : `UNITE DEPEND DE L ARSENAL` · `rc.ref` · `7ter.101` · `FIESTA 174` ·
> `CONDITION D INDICE`.

### 22.1 LE DOCUMENT DU CHANTIER VOISIN, ET COMMENT LE CITER

Le chantier **replay 2D** vit dans le worktree `.claude/worktrees/filmdec-continuation`
(branche `feat/filmdec-continuation`). Son document de reference est
**`.ai/SUIVI_REPLAY_2D.md`**, section *<< CE QUE LE FILM PORTE VRAIMENT — a ne jamais
surinterpreter >>*, **datee du 2026-07-28**. Elle publie, sur le film `000d5950` :
`tirs partis (API) 2 228` · `tirs qui touchent (API) 595` · `records longs dans le film 519` ·
`rapport record / touche 0,87`. Sa regle d hygiene, adoptee ici, est dans
**`.ai/V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md` §0** : *tout document de rejeu porte l identifiant du film de
CHAQUE bloc.* **CE WORKTREE EST EN LECTURE SEULE POUR NOUS** — on le cite, on n y ecrit rien.
Reciproquement, ce qui suit est ce qu il peut citer de notre cote : `7ter.98` (`rc.unite`),
`7ter.99` (`rc.perm`) et `7ter.101` (`rc.ref`) de `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md`.

### 22.2 LE CONFLIT ETAIT LEXICAL SUR L OBJET — `[ETABLI]`

`0xD2` est un objet unique et les deux chantiers le comptent. Reproduit par un **quatrieme
binaire** (`cmd/tmp_rcref`), et la premiere jambe **n emploie AUCUNE grammaire** : elle compte des
octets.

```
film 000d5950 : type-0 30 371 | pay[0]>>1==105 832 | 0xD2 519 | 0xD3 313 | code 36 519 (tous 0xD2)
                porteurs 97 | API 2 228 tirs / 595 touches
corpus 927 films : code-36 BRUTS 1 799 630 | porteurs 643 233 | tirs 3 268 658 | touches 1 195 460
```

La variante COURTE `0xD3` (313 sur `000d5950`) est un **autre code**, jamais compte par ce
chantier, et **elle n est toujours pas decodee**.

### 22.3 CE QUI A CHANGE : L UNITE N EST PAS INVARIANTE — `[REFUTE]` contre §21 comme enonce general

Fond permute **intra-film**, 200 tirages, `|d| <= 5`, **par famille de mode** :

```
famille (obs)          egalite                  reel   permute moy   max   tirages >= reel
Tactical  (138)        records == shots_fired     45       14.3       24        0 / 200   <- TIR
                       records == shots_hit        1        2.4        7      188 / 200
FIESTA   (1846)        records == shots_fired      8       25.7       38      200 / 200
                       records == shots_hit      174      134.7      163        0 / 200   <- TOUCHE
BTB       (583)        records == shots_hit       43       29.1       41        0 / 200   <- TOUCHE
Arena    (2087)        porteurs == shots_hit     168      137.5      167        0 / 200
```

**Le film temoin du chantier voisin est un `Super Fiesta:Slayer on Cliffhanger`** : sur SA famille
de mode, sa lecture << le film porte les records de degat, pas les tirs >> est **la bonne**.
Agregats de famille : Fiesta `rec/tir` **0.3147**, `rec/touche` **1.1046** ; Tactical `rec/tir`
**0.9457**, `rec/touche` **3.4007**.

**MECANISME, ET IL RESTE UNE HYPOTHESE NON MESUREE** : un record par **instance de source de
degat** — creee au depart du coup pour une trace instantanee, a l impact pour un projectile. Le
test qui la separerait d une coincidence de composition (ventiler les records **par arme** dans un
meme film Fiesta) **n a pas ete fait**.

### 22.4 CE QUI EST RENFORCE — LE TEST DE LOCALISATION QUE §21 N AVAIT PAS FAIT

Sur arsenal hitscan, par quartile de precision API :

```
Tactical (n=138)   precision  0.1819  0.2465  0.2906  0.3558   <- DOUBLE
                   rec/tirs   0.9305  0.9306  0.9239  0.9457   <- PLAT      => record = TIR
                   port/rec   0.1724  0.2222  0.2727  0.3220   <- SUIT      => porteur = TOUCHE
Arena (n=2087)     precision  0.3495  0.4248  0.4771  0.5407
                   rec/tirs   0.8400  0.8453  0.8504  0.8543
                   port/rec   0.3396  0.4092  0.4630  0.5227
```

Aucun facteur d echelle ne produit une colonne PLATE et une colonne qui SUIT en meme temps.

### 22.5 NOS MESURES PAR JOUEUR NE SONT PAS INVALIDEES — `[ETABLI]`, sur la vraie quantite publiee

Population **publiee** (roster <= 16, films sans participant fantome, ancrage strict, 579 films,
**n = 4 607**), quantite `porteurs / records` contre `shots_hit / shots_fired` :

```
  ancrage FILM-SEUL (le notre)   r 0.7740   MAE 0.0802   agregat film 0.3798
  ORDRE DE LA BASE               r 0.5731   MAE 0.1141   agregat film 0.3799
  NULLE permutee (200 tirages)   r 0.5730 [0.5586 .. 0.5878]   MAE 0.1142   -> 0/200
  hors echantillon (SHA-256)     moitie A 0.7773 (n=2380) | moitie B 0.7701 (n=2227)
```

**L ordre de la base EST la nulle permutee, au quatrieme chiffre.** Notre appariement porte le
signal, le sien n en porte aucun. Et **les agregats sont invariants par permutation** (0.3798
contre 0.3799) : le couple d agregats ne prouvait rien, ni avant ni apres (deja demontre par
7ter.99 (5)(b)).

### 22.6 DEUX CORRECTIONS DE COMPTAGE A CONNAITRE AVANT DE CITER UN CHIFFRE

1. **L ecart de 1.14 % laisse ouvert par §21.4 est RESOLU, et sa cause supposee est FAUSSE.** Ce
   n est pas la borne `s34` (**aucun effet entre 16 et 64** : 97 556 identique aux quatre bornes) :
   c est la **CONDITION D INDICE**. Sur les 150 films de 7ter.86 (1), porteurs **TOUS** = **98 685**
   (le chiffre de 7ter.86, a l unite), porteurs **INDEXES** = **97 556**.
2. **Le meme piege sur les records du corpus** : **1 799 630 BRUTS** (porte attaquant ouverte OU
   fermee) contre **1 774 183 INDEXES**, soit **1.41 %**. §21 publie les BRUTS sans le dire, et sur
   `000d5950` les deux coincident (zero porte fermee) — le cas ou l ambiguite ne se voit pas.
   **Ecrire la definition a cote du chiffre.**

### 22.7 CE QUI RESTE VRAI, ET LA REPONSE COURTE AUX TROIS QUESTIONS

```
UN RECORD EST-IL UN TIR OU UNE TOUCHE ?   NI L UN NI L AUTRE UNIVERSELLEMENT : le TIR sur arsenal
                                          hitscan, la TOUCHE sur arsenal a projectiles. La touche
                                          est portee, dans les deux cas, par le sous-ensemble
                                          << tableau A non vide >> (le PORTEUR).
LA PRECISION PAR ARME DEVIENT-ELLE        NON. Ce qui est disponible reste un TAUX (le remplissage
DISPONIBLE ?                              du tableau A), pas un COMPTE, et seulement sur les armes
                                          a tir instantane. Rien ne change a §17 / 7ter.85 / 87.
                                          -> tranche noir sur blanc dans le guide qui sera lu :
                                          `.ai/GUIDE_WEAPON_SHOTS.md` §3quater.4 (la liste des
                                          quatre armes, l interdit du corpus entier et le piege
                                          MA40 / Sidekick restent TOUS en vigueur).
LA PRECISION D UN JOUEUR EST-ELLE         OUI, et c est le livrable : taux de remplissage, MAE
PUBLIABLE SANS L API ?                    0.0266, r +0.8204. Guide : §3quater.
NOS MESURES PAR JOUEUR SONT-ELLES         NON (22.5). L ancrage est film-seul et il bat sa nulle
INVALIDEES ?                              a 0/200 sur la quantite publiee.
```
