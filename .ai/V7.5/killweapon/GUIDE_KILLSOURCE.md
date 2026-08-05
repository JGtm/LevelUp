# GUIDE KILLSOURCE — decoder la source du degat par kill, et la tester a fond

> Document autoportant. Il ne suppose aucune lecture prealable du journal de retro-ingenierie.
> Etat au 2026-07-28. Paquet : `apps/go-api/internal/games/halo_infinite/film/killsource`.
> **Nouveau le 2026-07-28** : §6bis, les impacts de projectile (codes 6 et 7) — ce qu ils
> donnent, ce qu ils ne donnent pas, et pourquoi le TIREUR n y est pas ; puis **§6bis.5, les deux
> INTERDITS DE METHODE** (la purete ne discrimine pas un alignement de bits ; le balayage bit a
> bit hors du parcours nominal fabrique des distributions credibles). Apres 7ter.89.
> Commande : `apps/go-api/cmd/killsource`.

---

## 1. CE QUE L OUTIL REND — DEUX VERITES, JAMAIS UNE

Pour chaque mort d un film Theater, l outil rend **deux reponses a la question << qu est-ce qui l a
tuee >>**, et elles ne se remplacent pas :

```
LE CREDIT DU JEU       qui recoit le kill au kill-feed. C est ce que le joueur voit a l ecran et
                       ce sur quoi ses statistiques officielles sont baties. Le jeu credite le
                       DERNIER joueur ayant inflige des degats.
LA SOURCE DU DEGAT     d ou vient le degat fatal, lu dans l etat de mort de la VICTIME. Ce n est
                       jamais l arme tenue, jamais le credit.
```

**Les deux sont vraies.** Elles repondent a des questions differentes. Quand elles divergent —
une roquette tiree trop pres d un mur, un baril lance trop pres, une chute — **la divergence est
elle-meme une information**, et l outil la signale au lieu de trancher. Sur les quatre films de
reference, 8 morts sont dans ce cas ; les 8 ont ete confirmees une par une en mode Theater.

Exemple, dans les mots de l utilisateur : *<< le kill feed attribue le kill a HizaroMne4262 alors
que c est MOI qui me tue en tirant une roquette sur un mur >>*.

**Il n existe volontairement aucune colonne << vrai tueur >> ni << arme corrigee >>.** Ce serait
presenter une verite comme l amendement de l autre.

**L ASSISTANT ET LES DEUX PARTS DE DEGATS NE SONT PAS UNE TROISIEME VERITE.** Ce sont des attributs
de la verite KILL-FEED : le jeu credite un tueur, et il declare a cote un assistant et deux
pourcentages de degats (celui du tueur, celui de l assistant). Ils viennent du KILL-EVENT, en tete
de paquet ; la source du degat, elle, vit dans le DEAD-STATE de la victime, dans la boucle de
records — **deux structures differentes du meme paquet**. Une donnee que l API ne fournit pas :
elle ne rend qu un TOTAL d assists par joueur et par match, jamais << qui a assiste quel kill >>.

---

## 2. DEMARRER EN TRENTE SECONDES

```bash
cd apps/go-api

# lister les films disponibles dans le cache
ls ../../data/cache/film_chunks

# la table des morts d un film
# Depuis apps/go-api. Le drapeau -cache est OBLIGATOIRE depuis ce worktree : il ne contient
# pas les films, ils vivent dans le depot principal.
go run ./cmd/killsource kills 000d5950 \
  -cache c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache
```

Un decodage prend **8 a 30 secondes** par film. Il ne touche ni la base, ni le reseau, ni les
fichiers du jeu : il lit des chunks sur le disque, et rien d autre.

Si les films ne sont plus en cache : `go run ./cmd/fetch_film_chunks/` les retelecharge.

---

## 3. LES QUATRE COMMANDES

| commande | ce qu elle fait |
|---|---|
| `kills <film>` | la table des morts, lisible : victime, source, nature, credit |
| `json <film>` | la meme chose, en JSON — schema stable, denominateurs nommes en clair |
| `sante <film>` | la metrique de sante, son verdict, ses alertes, et son point aveugle |
| `comparer <film> <film>` | les memes quantites sur deux films, cote a cote |

`<film>` est soit un identifiant court de 8 caracteres (resolu sous `-cache`), soit un chemin de
repertoire contenant des `chunk_NN.bin`.

**Options d affichage** (elles ne touchent JAMAIS au decodage — la configuration du decodeur est
gelee dans le paquet et n est pas pilotable depuis la ligne de commande). Elles se placent
**n importe ou** sur la ligne de commande, avant comme apres le nom du film :

```
-cache <dir>    racine du cache de films (defaut ../../data/cache depuis apps/go-api)
-limite N       n afficher que les N premieres morts
-tout           ajouter l identifiant brut de la source, la voie de lecture et le statut
```

**DEPUIS UN WORKTREE GIT**, `../../data/cache` pointe sur le worktree, qui ne contient pas les
films : ils vivent dans la copie de travail principale. Passer `-cache` explicitement :

```bash
go run ./cmd/killsource kills 000d5950 -cache <racine-du-depot-principal>/data/cache
```

Exemples :

```bash
go run ./cmd/killsource kills 9b191a7f -tout -limite 20
go run ./cmd/killsource sante 4f77afc1                    # un BTB : il doit sortir du domaine
go run ./cmd/killsource comparer 000d5950 fccc61cd
go run ./cmd/killsource json fccc61cd > fccc61cd.json
```

---

## 4. LIRE LA TABLE DES MORTS

```
temps  victime        SOURCE DU DEGAT FATAL   nature  CREDIT DU JEU
01:12  LORD PEINX13   M41 SPNKr               arme    aldusbroncus  <>
```

- **`<>`** = les deux verites divergent. La source appartenait a la victime, le jeu credite un
  autre joueur. **Les deux lignes sont justes.**
- **`Autres`** dans la colonne source = aucun nom propre publiable pour cette source. **La NATURE
  reste juste** (arme / melee / grenade / vehicule / objet explosif / environnement) : c est le
  nom qui manque, pas la lecture.
- **`(mort absente du kill-feed)`** a cote du credit = la victime est un **bot**. Le kill-feed du
  film est humain-seul (un bot n a pas de XUID), donc le kill y est mais pas la mort. La victime
  vient alors du roster de replication declare par le film lui-meme.
- **`(kill absent du kill-feed)`** = le **TUEUR** est un bot. C est l inverse exact : la mort est
  bien au kill-feed (la victime est humaine), c est le **kill** qui n y est pas, donc le nom du
  tueur vient du roster de replication. **Ces lignes s excluent par `origine = mort infligee par
  un bot`** si un consommateur ne veut que ce que le kill-feed nomme lui-meme.
- La qualification apres le point median (`· tir a la tete`, `· assassinat`) est le modificateur
  de degat. **Il ne distingue PAS grenade et melee ordinaire** : les deux sortent sans
  qualification, c est l identifiant de source qui les porte.

Avec `-tout`, deux colonnes s ajoutent :

- **`tag`** : l identifiant brut de la source. C est la quantite qui ne depend d **aucune** table
  de nommage — si le nom vous parait douteux, c est le tag qu il faut citer.
- **`lecture`** : `sequentielle` ou `balayage`. Voir §6.

---

## 5. LES DENOMINATEURS — POURQUOI IL Y EN A QUATRE

**Ne jamais ecrire << X % des morts >> sans dire lequel.** C est la regle la plus importante de ce
chantier, et elle a coute une section entiere de journal.

```
371 / 371  couples REELS                     100.0 %   <- LE DENOMINATEUR DE REFERENCE
371 / 372  couples reconstruits               99.7 %
371 / 375  morts du KILL-FEED                 98.9 %
380 / 380  morts de l API                    100.0 %   (etait 376 / 380 avant RE_LOG 7ter.79)
```

- **couples REELS** = ce que la reconstruction de kill-feed produit, **moins les couples
  FABRIQUES**. Un couple fabrique n est pas une mort manquee : c est une mort **qui n existe
  pas** — la vraie victime etait un bot, et la reconstruction avait pris la victime du voisin.
  Le decodeur a corrige notre propre appariement sur ce point, verifie en Theater.
- **couples reconstruits** = les memes, couples fabriques compris.
- **morts du KILL-FEED** = les evenements bruts du chunk HIGHLIGHT. Il est **humain seul** : il ne
  contient aucune mort de bot.
- **morts de l API** = la seule reference complete, bots compris. **Elle n est pas calculable hors
  ligne** : rien dans le film ne la rend. La commande ne l affiche donc pas ; elle est declaree
  comme constante documentee dans le golden cumule, et nulle part ailleurs. Les **5 morts DE bot**
  et les **4 morts INFLIGEES PAR un bot** s ajoutent **ici et nulle part ailleurs**.

La commande `kills` affiche les trois premiers et dit explicitement pourquoi le quatrieme manque.

**TROIS POPULATIONS DISJOINTES, ET LA COMPTABILITE SE FERME FILM PAR FILM** (RE_LOG 7ter.79) :

| population | le kill-feed porte | compteur |
|---|---|---|
| humain tue par un humain | le kill **et** la mort | `morts_couvertes` |
| **BOT** tue par un humain | le kill, pas la mort | `morts_de_bot_population_neuve` |
| humain tue par un **BOT** | la mort, pas le kill | `morts_infligees_par_un_bot_population_neuve` |

```
film        couverts  +  morts DE bot  +  morts PAR un bot  =  morts de l API
000d5950       93           0                 0                    93
9b191a7f       84           3                 3                    90
78919882       99           0                 0                    99
fccc61cd       95           2                 1                    98
CUMUL         371           5                 4                   380
```

**SEUL LE NUMERATEUR DU QUATRIEME A CHANGE** avec 7ter.79 (376 -> 380) ; le denominateur 380 et
les trois autres taux, numerateurs compris, sont inchanges.

---

## 6. CE QU IL FAUT CROIRE, ET CE DONT IL FAUT SE MEFIER

### CE QUI EST SOLIDE

| quoi | pourquoi |
|---|---|
| **l identifiant de source** (`tag`) | il ne depend d aucune table de nommage. 59 valeurs distinctes observees, 59/59 sont des entrees valides du groupe attendu, zero exception. Taux de faux positif du test : ~1 sur 9 000 000. |
| **la NATURE** (arme, melee, grenade, vehicule, objet explosif, environnement) | elle vient de la structure des archives du jeu, pas d une heuristique. Elle reste juste meme quand le nom sort en `Autres`. |
| **la divergence credit / source** | 8 cas sur la serie de reference, **confirmes 8 sur 8** en mode Theater. |
| **le fait qu une mort soit publiee** | le decodeur n invente pas de morts. Sur les paquets ou la cible est absente par construction, il accepte **1 candidat sur 208 913 712 bits** cumules sur les quatre films (0 / 1 / 0 / 0 par film). Mesure figee dans les golden, et elle reproduit au bit pres le controle negatif d origine du chantier. |
| **la couverture** | 371 sur 371 couples REELS sur les quatre films de reference, reproduit par une verification adversariale independante. |

### CE DONT IL FAUT SE MEFIER

| quoi | ce qu il faut savoir |
|---|---|
| **`Autres`** | 206 des 468 identifiants du catalogue ne remontent a aucune source nommee. Un encodage existe quelque part ; **ce n est pas la priorite**. Ne pas lire `Autres` comme << inconnu >> : la nature, elle, est connue. |
| **les 4 morts INFLIGEES PAR un bot** | **elles n ont AUCUNE ancre Theater, et elles n en auront jamais** : le kill-feed du jeu ne les attribue a aucun gamertag, donc on ne peut pas designer la ligne a verifier. Ce qui les tient : la voie MARCHE (98.2 %), la coherence tag/categorie (`0000d57f` Needler + `AttachedDamage` = supercombinaison), et le fait que leur nombre vaut EXACTEMENT les kills que l API donne au bot. Une seule d entre elles porte un tag sous `0x10000` (`0000d57f`), regime ou le test `jpt!` est 100 fois moins discriminant. |
| **la population des morts sans tueur** | elle est designee par le FEED : une mort qu aucun kill ne consomme. Un **suicide** ou un degat de monde y tomberait aussi. Sur les quatre films de reference il n y en a aucune ; **sur le BTB il y en a une, et c est precisement la que la population ne ferme pas** (7 orphelines pour 5 kills de bot + 1 suicide, 0 appariee, 0 publiee). |
| **un nom marque `nom sous reserve`** | la CLASSE est sure, le NOM PROPRE ne l est pas. Cas mesure : un marteau de campagne publie a la place du marteau multijoueur. Si le nom vous surprend, **citez le tag**. |
| **un nom marque `nom ambigu`** | l outil refuse de publier le nom et sort `Autres`. C est volontaire : le nom serait affirmatif et faux. |
| **la colonne `lecture`** | deux voies lisent le MEME champ, au meme bit quand elles repondent toutes les deux. La voie **sequentielle** apparie 98.2 % de ses candidats au couple exact du kill-feed, le **balayage** 78.4 %. **C est une ventilation du cout, pas deux precisions comparables** : la bijection indice → joueur est ajustee sur l union des deux, dont la sequentielle fournit ~91 % des candidats. |
| **le BTB** | **hors perimetre.** Le decodage y tient en agregat (**76.5 %**) mais la bijection y a une marge NULLE : au moins deux joueurs sont interchangeables, donc **aucune attribution ligne par ligne n y est publiable**. La commande le dit : *publication ligne par ligne REFUSEE*. ⚠ **RESERVE (7ter.74bis) : LE CHIFFRE BTB AFFICHE N EST PAS CELUI QUE CE PAQUET PRODUIT.** Il est extrait de l outil de RE `cmd/tmp_deadstate` (224 lignes publiees, 26.0 % d inexpliques, 3 hors-roster) ; le paquet, lui, rend **225 lignes, 26.5 % et 52 hors-roster** sur le meme film. L ecart est consigne comme **dette documentaire**, pas comme investigation — le BTB est hors perimetre par decision utilisateur, et les deux implementations sont EXACTEMENT d accord sur les quatre films de reference. |
| **le nom d une source de vehicule** | il nomme parfois LARGE (le chassis rend la faction et la classe, pas le modele), et **aucun critere ne le dit a l avance**. ⚠ **ET AUJOURD HUI IL N EN NOMME AUCUN** : les **89** entrees de classe VEHICULE du catalogue sont **toutes de statut `VALIDE` et toutes sans nom**, donc la regle d affichage `Name != "" && Publishable()` les envoie **100 % en `Autres`** (verifie deux fois : 7ter.90 (1) et 7ter.94 (7)(a)). **L information n est pas perdue** : le chassis est deja dans le champ `Detail` sous forme de nom de banque Wwise — 19 racines distinctes (17 chassis reels), et **9 tags seulement** ne citent aucune banque. **Les etiqueter est une table de 89 lignes, pas de la retro-ingenierie.** C est le seul livrable immediat du sujet vehicules. |
| **la peremption du catalogue** | risque **mineur** (les ajouts d armes sont rares : une ou deux par an au grand maximum). La detection existe et elle est testee — voir §7 — mais ce n est pas un entretien regulier. |

### LA REGLE QUI RESUME TOUT

**Une mesure porte sur ce qu elle a mesure.** Trois fois dans ce chantier, une mesure juste a ete
transformee en regle trop large, et les trois fois cela a coute des jours. Si un chiffre de ce
guide est cite ailleurs, il doit partir avec sa portee.

---

## 6bis. LES IMPACTS DE PROJECTILE (CODES 6 ET 7) — CE QU ILS DONNENT, ET CE QU ILS NE DONNENT PAS

> Ajoute le 2026-07-28. Sources : `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md` **7ter.86** (localisation),
> **7ter.88** (lot `pj.own`), **7ter.89** (verification adversariale `tv.ref`), **7ter.91**
> (decodage du corps, lot `c7b`) et **7ter.94** (verification adversariale `tp.ref`).
> **LA REPONSE COURTE, SI VOUS ETES VENU POUR << LES TOUCHES PAR ARME >> : NON.** Le corps de
> l evenement d impact est decode et ne porte aucun identifiant d arme ; les trois voies
> possibles sont fermees, chacune par une mesure. Detail en 6bis.4.
> **Cette section ne change RIEN au decodage de l arme du kill** : elle existe parce que la
> question << peut-on compter les touches d une roquette / d une aiguille ? >> revient
> regulierement, et la reponse a un cout d entree tres precis.

### 6bis.1 CE QUI EST ACQUIS

Le film porte les impacts de projectile, et ils ont un numero d evenement :

```
  code 6   impact sur la GEOMETRIE (rien touche)          80 886 sur 949 films
  code 7   impact sur une ENTITE                         129 390 sur 949 films

  dont en PREMIER evenement de leur paquet — LE SEUL SOUS-ENSEMBLE A POSITION CERTAINE :
  code 6    70 392        code 7    97 988       (949 films)
  reproduit A L UNITE par deux binaires ecrits separement (7ter.91 (2), 7ter.94 (1))
```

Le premier evenement d un paquet se lit au **bit 1** du payload (le bit 0 vaut 1 sur
28 726 935 paquets sur 28 726 935 : ce n est pas un bit de continuation), et **80.29 %** des
paquets type-0 portent une liste d evenements VIDE. Cette position **ne depend d aucune grammaire
de corps** — c est ce qui rend ces deux comptes plus solides que les totaux. Le desassemblage donne la mecanique : `FUN_142f1c44c` (impact de projectile) ->
`FUN_140de8cb0` -> `FUN_142eed4e8`, qui emet le **type 6** quand rien n est touche et le **type
7** quand une entite l est. **Le code 7 nomme DEUX entites : la CIBLE puis l OBJET PROJECTILE.**

Le volume est **LOCALISE** sur les armes a projectile, ce qui est le test qui compte :

```
  r(code 7, tirs de PROJECTILE)      +0.7675
  r(code 7, tirs a TRACE INSTANTANEE) -0.1929      <- le signe s inverse
  (code 6 + code 7) / tirs de projectile = 0.9831  <- un impact par tir, sans ajustement
  39 des 65 films SANS aucun tir de projectile portent EXACTEMENT zero code 7
```

### 6bis.2 CE QU ILS NE DONNENT PAS — ET C EST BLOQUANT

**L EVENEMENT NE NOMME JAMAIS LE TIREUR.** Il nomme la cible et le projectile. Sans tireur, pas
de ventilation par joueur, donc **aucun test d egalite exacte possible** : le lot reste a
`[MESURE]` et n ira pas plus haut par cette voie.

Et le tireur n est pas ailleurs dans le flux non plus, parce qu **IL N EXISTE AUCUN EVENEMENT DE
CREATION DE PROJECTILE**. Le critere est net : *un objet est cree exactement UNE fois, donc un
evenement de creation les COUVRE TOUS*. Aucun code ne le fait. **Consequence directe : la
ventilation des touches de projectile PAR JOUEUR n est pas atteignable par cette voie**, ni
aujourd hui ni en ameliorant le decodage des codes 6 et 7 — le tireur n est pas dans la donnee.

```
  couverture de l ensemble des projectiles, 150 films
     meilleur candidat (code 5)      0.2318   nulle 0.1601
     fire-event  (code 36)           0.0931   nulle 0.0711
  test sur une reference INDEPENDANTE (le candidat n en fait pas partie), 7ter.89 (3)
     meilleur candidat de tout le corpus  0.5046   nulle de MEME POOL 0.3600   rapport 1.40
     ce qu un nommage REEL exige                  1.0000                       rapport 11.37
```

**Le projectile n existe dans le flux d evenements qu a l INSTANT DE SA MORT.** Aucun evenement
de creation, et le balayage complet des ecarts de base ne montre aucun pic.

> ⚠ **NE PAS CITER << la somme 0.5408 + 0.4606 = 1.0014 prouve que l instrument voit ce qu il
> doit voir >>.** Cette somme est une IDENTITE ALGEBRIQUE (l ensemble de reference est l union
> des deux ensembles compares), reproduite a neuf decimales sur deux corpus — 7ter.89 (2). Le
> negatif tient, mais par le rapport reel/nulle ci-dessus.

### 6bis.3 LES DEUX AUTRES VOIES, ET LEUR PRIX

| voie | verdict | chiffre a citer |
|---|---|---|
| **La PARENTE (`object-parent-state-component`, i10) du projectile** | **A NE PAS OUVRIR SANS RAISON NEUVE.** i10 relie un objet a celui auquel il est ACCROCHE : pour un projectile c est l aiguille plantee ou la grenade collee, donc **la VICTIME**. Chercher le geste d un TUEUR dans un champ de la VICTIME est le patron d erreur deja paye trois fois par ce chantier. | l archetype 41 ne replique **aucun** proprietaire sur ses 22 composants ; `object-multiplayer-properties-component` (i9) est resolu et n est qu un selecteur d union de mode de jeu |
| **Le TEMPS (apparier l impact au tir qui le precede)** | **CHIFFREE ET INEXPLOITABLE.** C est le motif << same-clock >>, deja formellement invalide par ce chantier (un 94 % auto-confirmant, verite terrain 11/14 contre une baseline triviale de 10/14). | **plafond 0.41** d impacts a candidat UNIQUE, toutes fenetres et toutes restrictions confondues (150 films, 23 188 impacts). Sous 100 ms le nombre moyen de candidats est **inferieur a 1** ; au-dela l ambiguite croit plus vite que la couverture (1.98 a 1 s) |

### 6bis.4 CE QUI RESTE A PORTEE, ET CE QUI NE L EST PAS

**A PORTEE — la ventilation des touches RECUES par joueur.** Le fire-event (code 36) porte un
handle d entite dont la relation vers le tireur est une **FONCTION mesuree a 0.9805**
(14 896 / 15 192 handles, 150 films, contre 0.0652 apres permutation intra-film ; reproduit a
l unite par un second binaire en 7ter.89 (1)). Le code 7 nomme sa CIBLE dans le **meme pool** que
le code 36 nomme son objet. Tout ce qui, dans le film, nomme un objet de ce pool est donc
attribuable a un joueur.

**HORS DE PORTEE — la ventilation des touches DONNEES**, qui est la question de `shots_hit`. Elle
suppose de relier le projectile a son tireur, et c est exactement ce que 7ter.88 et 7ter.89
ferment.

> ⚠⚠ **LES DEUX << PIEGES >> QUI FIGURAIENT ICI SONT PERIMES DEPUIS LE 2026-07-28 SOIR. Ils
> disaient << la dedup est a rejouer >> et << les touches par arme sont a un decodage de
> distance >>. Le decodage A ETE FAIT (7ter.91, verifie independamment par 7ter.94) et les deux
> enonces sont remplaces par ce qui suit.** Ils sont retires, pas nuances : les laisser
> enverrait quelqu un refaire une semaine de travail deja faite.

**CE QU IL FAUT SAVOIR AVANT DE COMPTER QUOI QUE CE SOIT :**

1. **LE CORPS DU CODE 7 EST DECODE, ET LE TAG D ARME N Y EST JAMAIS TRANSMIS.** Le tag est bien
   lu par `FUN_14080d69c` — la MEME fonction que le tag `jpt!` du dead-state — mais derriere une
   porte que le flux n ouvre pas : le bit qui la commande vaut **1** sur **168 380** impacts a
   position certaine (97 988 codes 7 + 70 392 codes 6, 949 films), **zero exception**. Controle
   positif du meme instrument (reecriture d un tag connu, puis relecture) : **1.0000** dans les
   deux lots. **Aucun autre champ du corps n est un identifiant d arme** (`w16`, le seul assez
   large, porte 2 199 petits entiers). **La ventilation des touches PAR ARME est FERMEE par
   cette voie.**
2. **LA DEDUPLICATION PAR OBJET-SOURCE : L HYPOTHESE QUI LA MOTIVAIT EST REFUTEE.** << Une
   roquette qui blesse trois joueurs emet 3 codes 7 >> predit une queue de multi-occurrences sur
   le code 7. Mesure : la dedup retire **2.2 a 3.1 %** au code 7 et **4.9 a 5.9 % au CODE 6** —
   qui frappe la geometrie et ne peut structurellement pas se diffuser a plusieurs cibles ; et
   le rapport `code7/(code6+code7)` **MONTE** au lieu de descendre. Le multi-impact existe mais
   il est marginal. **Le sur-comptage vient d ailleurs, et sa cause reste inconnue.**
   (7ter.91 (6), reproduit par 7ter.94 (3).)
3. **LA LONGUEUR DU CORPS — LES << 6 BITS MANQUANTS >> SONT EXPLIQUES, MAIS CE N EST PAS UNE
   CONSTANTE.** Le corps du code 7 fait **118 bits** = `1+7+7+19+2+36+19+9+16+1+1`, chaque
   largeur lue dans un immediat du desassemblage — et le FILM le confirme independamment : apres
   un code 7, avancer de `L` bits et demander si la chaine reprend sur un en-tete valide rend
   **0.2029 a L = 118** et **0.0000 a neuf des dix voisins** (7ter.94 (5)). L ecart avec le
   code 6 vient de **deux lecteurs de position differents** (3 x 12 bits FIXES contre des
   largeurs lues dans une table installee au chargement), et il vaut **`36 - P(carte)`** :
   la largeur `P` du bloc de position du code 6 est **MAP-DEPENDANTE**, mesuree hors echantillon
   (25/31, fond 1/31), de **24** (Fortitude, High Ground, Threshold) a **43** (Chasm).
   **Consequence pratique : `evFixed[6] = 93` est une longueur MODALE, pas universelle** — c est
   pour cela que la marche s arrete plus souvent apres un code 6 qu apres un code 7.

### 6bis.5 DEUX INTERDITS DE METHODE, ET ILS SE PAIENT DANS CETTE SECTION

> Ajoute le 2026-07-28 apres 7ter.89. Ces deux points ne sont pas des reserves de confort : ce
> sont les deux facons connues de produire un chiffre credible et faux sur ce sujet precis.

**INTERDIT 1 — LA << PURETE >> NE DISCRIMINE PAS UN ALIGNEMENT DE BITS.** Quand on cherche a
quelle position se lit un champ (ici l indice de tireur du code 36), la tentation est de balayer
les positions voisines et de garder celle dont la relation est la plus << pure >> (un handle ->
un seul indice). **Cela ne marche pas** : la purete est plate sur quatre positions consecutives.

```
  purete du lien handle -> indice de tireur, 150 films
    +32   0.8422      <- elle ne CHUTE qu ici
    +33   0.9808
    +34   0.9805      <- la position retenue
    +35   0.9803
    +36   0.9801
```

Ce que la purete fait : elle **exclut un cadrage grossier** (+32). Ce qu elle ne fait pas :
choisir entre deux bits voisins. **Seule l EGALITE EXACTE contre une reference EXTERNE discrimine
un alignement** — et encore faut-il qu elle separe reellement. Ici elle ne suffit pas non plus :

```
  egalites exactes contre le roster de l API (nombre d indices de tireur du film == roster)
    offset   sans filtre        avec filtre << l evenement porte un handle >>
     +32       0 / 150                2 / 150
     +33       0 / 150                2 / 150
     +34      50 / 150               60 / 150
     +35      50 / 150               69 / 150   <- il EGALE, puis il BAT
     +36      46 / 150               60 / 150
```

**`+34` reste la bonne position — mais parce que DEUX instruments independants du depot la
designent** (7ter.26 (5) et `weaponv3.FirePi5SpanBefore`), **pas parce que le roster l aurait
confirmee**. Ne jamais citer ce tableau comme la confirmation de `+34` : il exclut `+32` et
`+33`, rien de plus. Ce qui separait `+34` de `+35` dans la premiere publication etait l ecart
absolu moyen — la statistique agregee que la methode du chantier interdit comme discriminant
(REGLE 1 / PATRON E). **Meme piege releve deux fois le meme jour** (7ter.87 puis 7ter.89 (4)).

**INTERDIT 2 — LE BALAYAGE BIT A BIT HORS DU CHEMIN NOMINAL EST MORT.** Lire un composant ou un
champ en balayant les positions au lieu de marcher la grammaire produit des resultats, et ces
resultats sont faux. Densite de faux positifs du predicat generique de record ECS : un en-tete
plausible sort **1 fois tous les 18 bits** ; **1 fois tous les 8 366 bits** avec toutes les
contraintes reunies. Et surtout **il FABRIQUE DES DISTRIBUTIONS CREDIBLES** : le scan cible sur
`9b191a7f` rend **292 candidats pour 87 kills**, dont **193/292 sont la MEME paire
(tueur=16, victime=5)** — un motif binaire repete qui ressemble a une population — et
l appariement au kill-feed autoritatif ne retrouve que **14/84 a <= 200 ms** (7ter.27 (6)).
Meme chose sur les vehicules : un scan pointe sur une bande de slots **VIDE** rend **17
echantillons** la ou la bande reelle en rend 5 (7ter.89 (8)). **C est un interdit, pas une
reserve** : ce qui est valide, c est le scan CONTRAINT a une bande declaree independamment.

---

## 6ter. LES FRAGS DE CLASSE VEHICULE — CE QUI EST PUBLIABLE, ET LA REGLE ANTI-INVENTION (7ter.96, corrige par 7ter.100)

**LE FAIT DE DEPART.** Le catalogue porte **89 tags de classe VEHICULE, tous de statut `VALIDE`,
et AUCUN n a de `Name`**. La regle d affichage etant `Name != "" && Publishable()`, ils tombent
tous en `Autres`. Le chassis, lui, est deja dans le champ `Detail`, sous la forme d une ou
plusieurs banques Wwise `sb_<chiffres>_<racine>`.

**MAIS LA RACINE DE BANQUE N EST PAS UNE IDENTITE DE CHASSIS.** Critere, et il ne contient **aucun
jugement de connaissance du jeu** : le `Detail` cite-t-il UNE SEULE racine de banque, plusieurs, ou
aucune ? Mesure sur les **14 tags de classe VEHICULE reellement observes** dans trois films recents
(rejouee a l unite par 7ter.100, **recomptee par 7ter.102**) :

```
  racine UNIQUE       8   veh_cv_ghost (f712c64a), veh_cv_wraith (deffdc6b, 7c4f8a28),
                          veh_un_rockethog (5a4450e4), tur_cv_plasmacannon (3c7560e0),
                          tur_un_machinegun (382cafaf),
                          veh_un_pelican (28907150), veh_un_falconlmgturret (77a61ef5)
  racines MULTIPLES   4   674a7d69, a859230a (gausscannon / falcongrenadelauncher),
                          003f582d (gausscannon / machinegun / rocketturret),
                          3b3b3d40 (autoturret banni / wasp)
  AUCUNE banque       2   00426796, d2ffec3f
```

**8 sur 14 = 57 %**, et **53 sur 89 = 59.6 %** sur le catalogue entier (repartition : 9 tags a zero
racine, 53 a une, 12 a deux, 13 a trois, 1 a quatre, 1 a six). **C EST LE CHIFFRE DE CATALOGUE QU IL
FAUT CITER** : il ne depend d aucun film ouvert.

> ⚠ **UN `6 / 14` A CIRCULE ICI JUSQU AU 2026-07-28 — IL EST RETIRE, ET LA RAISON EST UNE FAUTE DE
> METHODE.** Il additionnait << racine UNIQUE >>, qui se mesure, et << chassis PLAUSIBLE >>, qui
> est une croyance sur le jeu. La croyance etait fausse : **le Falcon existe en BTB** (correction
> de l utilisateur), et le film de la mesure, `4f77afc1`, **EST un BTB**. Le Pelican est reexamine
> pour la meme raison au lieu d etre reconduit. Statut des deux tags : **`[NON VERIFIE]`**, plus
> << implausible >>. ⚠⚠ **NE PAS SUR-CORRIGER** : que le Falcon existe **retire un argument
> CONTRE**, cela **n ajoute aucune preuve POUR** que le tag le designe correctement. Detail :
> 7ter.102.

**LA BANQUE EST-ELLE REUTILISEE ENTRE CHASSIS ?** L enonce fort — *le jeu reutilise la meme banque
sonore entre chassis* — est `[PLAUSIBLE]`, plus etaye par le Pelican ni par le Falcon. L enonce
faible est `[MESURE]` et il suffit a fonder la regle ci-dessous : *la racine ne determine pas le
chassis par cette chaine d extraction* — **dix `vehi` distincts citent de deux a six racines
distinctes** au catalogue, dont `0000d4ff` qui cite Scorpion, Chopper, Banshee, Wasp et tourelle
Shade dans le meme `Detail`.

**REGLE ANTI-INVENTION, NON NEGOCIABLE** : collecter TOUTES les banques du `Detail`, traduire,
dedupliquer les noms, et **publier la disjonction entiere quand elle reste a plusieurs — jamais
choisir**. Prendre la premiere occurrence fait sortir un nom affirmatif et FAUX (`003f582d`
sortait << Gauss >> alors que son `Detail` cite trois banques). Une racine absente de la table
d etiquettes se publie telle quelle, prefixee de `?`, pour qu un trou reste VISIBLE.
⚠ << implausible >> est un jugement de connaissance du jeu, **pas une mesure** — et il a ete porte
sur deux cartes de **CREATION COMMUNAUTAIRE** (`Flood Gulch`, `High Ground`), dont la palette
d objets est choisie par l auteur (7ter.100 (13)). Seule une confrontation Theater tranche.
⚠⚠ **ET IL A ETE PORTE CONTRE LE JEU REEL** (7ter.102) : le Falcon existe en BTB, et la mesure a
ete faite sur un film BTB. **NE JAMAIS DECLARER UN DECODAGE IMPLAUSIBLE SUR UNE CROYANCE
CONCERNANT LE JEU — la croyance se verifie, comme le reste.**

**OU EST EXACTEMENT LA CONFIANCE D UNE LIGNE — FORMULE CORRIGEE PAR 7ter.100 (12).**

```
  NE DEPEND PAS de la bijection : qu un dead-state DONNE porte tel tag `jpt!`, telle classe,
                                  telle categorie.
  EN DEPENDENT, sur une LIGNE PUBLIEE : les deux noms, ET le rattachement de l arme a l instant
                                  — donc la ligne entiere.
```

Raison : une ligne n existe que si `matchExact` apparie le dead-state a un evenement du feed, et
`matchExact` exige `e.victim == roster.nameOf(cd.victim)` **ET** `e.killer ==
roster.nameOf(cd.killer)`. La seule autre contrainte est `const tolMS = 2500` — une fenetre de
**5 s**, alors que le feed d un BTB porte **un kill toutes les 4.04 s** et que **17 des 55 frags
VEHICULE d un film reel ont un autre frag VEHICULE a moins de 2 500 ms**. Donc, quand
`BijectionMargin == 0` (cas BTB), **une ligne ne se lit pas** << a 12:03.392, quelqu un a tue
quelqu un au Wraith >> **mais** << dans les 5 s autour de 12:03.392, une mort a ete causee par un
Wraith >>. En Arena/Community a marge positive, les deux noms sont publiables — c est la porte
`LineByLinePublishable()`.

**LE CONTROLE POSITIF DU MEME INSTRUMENT, ET IL EXISTE.** `match_participants` porte
`grenade_kills` et `melee_kills`, alimentes par le MEME chemin (tag `jpt!` du dead-state) :

```
  film      couverture   GRENADE decodee/API   MELEE decodee/API   VEHICULE
  78919882   99 / 99          3 / 3                 8 / 12            7
  fccc61cd   98 / 98          3 / 3                 2 /  4            2
  4f77afc1  225 / 303         9 / 9                 9 / 12           55
```

**GRENADE : egalite EXACTE 3/3, 3/3, 9/9. MELEE : deficit systematique**, non explique, et il ne
vient PAS de la couverture. Ce controle valide la CLASSE la ou une reference existe ; il ne valide
**ni** la classe VEHICULE (`DriverAssists` est absent de `match_participants`), **ni** le nom du
chassis.

**ET UN SECOND CONTROLE, PLUS FORT, AJOUTE PAR 7ter.100 (11)** : en joignant
`shared.xuid_aliases`, l accord des frags par tueur avec l API est exact **NOM PAR NOM — 8/8 et
8/8** sur les deux films d arene (une permutation au hasard le reproduit avec probabilite
`2/8!`). En BTB : **4 egalites exactes sur 8, et le decode n est JAMAIS au-dessus de l API**.
Il valide `Feed.Killer`, pas la bijection indice -> joueur.

---

## 7. LIRE LA SANTE — ET SON POINT AVEUGLE

`go run ./cmd/killsource sante <film>` rend un **verdict de DOMAINE**, pas un jugement sur les
etiquettes :

```
NOMINAL              le film ressemble a ceux sur lesquels le decodeur a ete mesure
HORS DOMAINE MESURE  il n y ressemble plus. Il n est pas casse : ses lignes se ponderent.
ALERTE               une condition dure est franchie. Voir les messages.
```

Les seuils sortent de la distribution de **cinq** films, pas d une intuition : 7.0 / 9.4 / 11.6 /
17.8 % de candidats inexpliques sur les quatre films a 8 joueurs, 26.0 % sur le BTB. Le BTB est le
**controle positif** : il doit sortir du domaine, et il en sort par trois criteres.
⚠ **A LIRE AVEC LA RESERVE DE 7ter.74bis** : la ligne BTB (26.0 %, 3 hors-roster) est celle de
l outil de RE, pas celle de ce paquet (26.5 %, **52** hors-roster). Tant que l ecart n est pas
instruit, **ce controle positif ancre les seuils de l outil dont il est extrait, pas ceux du
paquet**. Le domaine 4v4, lui, est intact — et c est celui qui compte.

**LE POINT AVEUGLE, ET IL EST MESURE.** Le compteur principal — << identifiants hors catalogue vus
par la voie sequentielle >> — ne sonne **que si la voie sequentielle porte l identifiant
concerne**. Un identifiant dont la seule occurrence publiee vient du balayage passe inapercu :

```
ablation mesuree d un identifiant servi par le seul balayage :
   compteur principal = 0        -> aucune alerte
   couverture 99 -> 98           -> verdict << HORS DOMAINE MESURE >>, et rien de plus
```

**C est pourquoi le plancher de couverture vaut exactement 1.00.** Dans ce mode, il est le SEUL
filet. Ce n est pas une severite decorative a assouplir, c est le compteur de secours.

Portee juste du controle positif, a citer telle quelle : *20 ablations sur 20 declenchent
l alerte, **sur des identifiants que la voie sequentielle porte aussi*** — et non *20 sur 20 sur
un catalogue perime*.

Consequence sur la verite terrain : **5 des 30 ancres Theater sont servies par le seul balayage**
(4 sur `9b191a7f`, 1 sur `78919882`) et ne sont donc pas protegees.

---

## 8. LE CODE EXACT DU BRANCHEMENT

Le paquet a **une seule fonction publique**. Le cablage complet tient en ce qui suit — il n y a
rien d autre a ecrire :

```go
import "levelup/go-api/internal/games/halo_infinite/film/killsource"

// `chunks` : les chunks du film, deja telecharges. Le paquet ne les telecharge pas —
// c est deliberement la responsabilite de l appelant (le telechargement est le vrai cout,
// il est batchable, et il a deja son chemin store-first).
src := killsource.MemoryChunks(chunks)

// `nil` = la CONFIGURATION GELEE, celle qui a produit les chiffres publies.
res, err := killsource.Decode(ctx, matchID, src, nil)
if err != nil {
    // errors.Is(err, killsource.ErrNoKillFeed) — pas de kill-feed, rien de publiable
    // errors.Is(err, killsource.ErrNoChunk) / ErrNoPacket / ErrRegistry
    return err
}

// GARDE-FOU AVANT TOUTE PUBLICATION LIGNE PAR LIGNE.
if !res.LineByLinePublishable() {
    // marge de bijection nulle (BTB) ou sante en ALERTE : agregat seulement.
}

for _, k := range res.Kills {
    _ = k.TimeMS          // instant de la mort
    _ = k.Victim          // la victime
    _ = k.Feed.Killer     // VERITE KILL-FEED : le credit du jeu
    _ = k.Feed.Present    // false = mort de bot, absente du kill-feed
    _ = k.Source.Tag      // identifiant brut, independant de toute table de nommage
    _ = k.Source.Display  // etiquette, ou "Autres"
    _ = k.Source.Class    // nature — toujours renseignee, meme quand Display vaut "Autres"
    _ = k.Source.Status   // VALIDE / SOUS_RESERVE / AMBIGU / INCONNU
    _ = k.Diverges        // les deux verites ne designent pas le meme responsable
    _ = k.Read.Path       // killsource.PathWalk (98.2 %) ou PathScan (78.4 %) — a ponderer

    // L ASSISTANT — attribut de la VERITE KILL-FEED, pas une troisieme verite.
    _ = k.Assist.Known    // FAUX = ON NE SAIT PAS. Ce n est JAMAIS << pas d assistant >>.
    _ = k.Assist.Name     // vide + Known=true = << pas d assistant >>, MESURE
    _ = k.Assist.Rejected // "assistant==tueur" | "assistant==victime" | "hors-roster"
    _ = k.Assist.Index    // indice de replication brut, -1 si absent (ne depend d aucune bijection)
    _ = k.Assist.Extra    // garde-fou : assistants distincts EN SURPLUS sur cette mort

    // LES DEUX PARTS DE DEGATS, en pourcentage ENTIER. Trois etats, jamais deux.
    _ = k.KillerDamage    // DamageShare{Pct, Known} — part du TUEUR
    _ = k.AssistDamage    // part de l ASSISTANT — Known=false sans champ assistant
}

// Publication des compteurs (ADR 0009 : entiers, snake_case, aucun ratio).
for _, p := range res.Health.ExpvarPairs() {
    observability.AddInt(p.Name, p.Value)
}
```

Pour rejouer un film depuis le disque au lieu d une tranche en memoire :
`killsource.DirChunks(dir)`.

### CE QUE LE BRANCHEUR DOIT SAVOIR EN PLUS

1. **Un seul decodage a la fois par processus.** Les parametres de replication du lecteur de bits
   sont des globaux de paquet ; le paquet serialise donc les passes par un verrou et remet ces
   globaux a zero a chaque entree. **L appelant n a rien a faire**, mais il doit savoir que
   paralleliser deux films n accelere rien.
2. **Toute ecriture per-match passe par `internal/persist/BatchBuilder`** (regle anti-corruption
   ART, ADR 0019 / 0030). Aucune exception.
3. **La table cible est tranchee** : `shared.match_kill_events`, neuve, append-only, **une ligne
   par MORT**, lecture par la vue `match_kill_events_latest`. Voir §12. **Ne PAS ecrire dans
   `killer_victim_pairs`** : elle porte le CREDIT du kill-feed, pas la source du degat, et elle
   n a **aucune colonne d arme** — ses neuf colonnes sont `match_id`, `killer_xuid`,
   `killer_gamertag`, `victim_xuid`, `victim_gamertag`, `kill_count`, `time_ms`, `is_validated`,
   `created_at` (`internal/migration/steps_shared.go`). Elle reste en place le temps que ses
   lecteurs migrent ; son retrait est decrit dans `steps_shared_kill_events.go`.
   > *Ce point affirmait jusqu au 2026-07-27 : << Ne PAS ecrire dans
   > `killer_victim_pairs.weapon_id` : cette colonne existe mais n est pas dans la meme unite que
   > l identifiant rendu ici >>. **CETTE COLONNE N EXISTE PAS** — aucune migration ne l ajoute ;
   > seul un champ Go jamais ecrit (`persist.KillerVictimInsert.WeaponID`) portait ce nom. L enonce
   > a servi d argument pour ne pas trancher la table cible. Corrige le 2026-07-28.*
4. **Le paquet ne lit AUCUN fichier du jeu.** Le catalogue des identifiants et la table de nommage
   sont embarques par `go:embed` (63 Ko). La preuve a ete faite en renommant les racines du jeu
   sur le disque : sorties identiques au bit, meme empreinte. Le serveur de production n a pas
   Halo installe et n en a pas besoin.
5. **Aucun libelle FR/EN n est en dur dans le paquet** : il publie des identifiants stables
   (`ARME`, `SilentMelee`, `marche`). La traduction appartient a la couche d affichage — la
   commande en donne un exemple complet dans `cmd/killsource/libelles.go`.
6. **L assistant et les deux parts de degats tombent sous la MEME porte que le reste**
   (`LineByLinePublishable`) — aucune porte separee. Mesure qui le justifie : en BTB, 62
   assistants nommes pour 122 a l API (51 %), contre 17/17 et 29/29 en Arena ; la cause est la
   marge de bijection NULLE, pas un decodage plus mauvais (RE_LOG 7ter.76).
7. **RESERVE A PORTER AVEC TOUTE CITATION DES DEUX PARTS** : le chemin de donnees entre le
   kill-event du film et les champs `KillerPercentageDamageDone` / `AssistantPercentageDamageDone`
   du modele de recap **n est pas demontre** (quatre jambes convergentes, pas une chaine d appels
   prouvee — RE_LOG 7ter.75). Et ces deux nombres **ne sont pas bornes a 100** : 1,7 % des
   kill-events attaches vont jusqu a 228.

---

## 9. LES TESTS — CE QU ILS FIGENT ET COMMENT LES REJOUER

```bash
cd apps/go-api

# tests purs : ils tournent partout, sans fixture
go test ./internal/games/halo_infinite/film/killsource/

# non-regression sur films reels + comparaison aux sorties figees
KILLSOURCE_FIXTURES=../../data/cache/film_chunks \
  go test ./internal/games/halo_infinite/film/killsource/ -v

# regenerer les sorties figees apres un changement VOLONTAIRE
KILLSOURCE_FIXTURES=../../data/cache/film_chunks \
  go test ./internal/games/halo_infinite/film/killsource/ -run Golden -update
```

(Depuis un worktree, `KILLSOURCE_FIXTURES` doit pointer sur le `data/cache/film_chunks` de la
copie de travail principale — voir §3.)

**Les fixtures ne sont pas versionnees** : les quatre films de reference pesent 107 Mo (21 + 25 + 35 + 26). Ce sont les **sorties**
qui le sont, sous `internal/games/halo_infinite/film/killsource/testdata/*.golden`, et chaque
fichier porte sa propre recette de regeneration en en-tete.

Ce que les golden figent, et pourquoi chacun :

| ce qui est fige | pourquoi |
|---|---|
| la couverture **avec la phrase qui nomme son denominateur** | un nombre nu ne prouve rien : trois denominateurs coexistent hors ligne et ne donnent pas le meme taux. Un test dedie verifie que les phrases sont toujours la — un golden peut se degrader en nombres sans jamais casser. |
| **les 30 ancres Theater, tags compris** | le filet le plus important. Un compte de couverture peut rester juste pendant qu une etiquette devient fausse. |
| **la ligne `fccc61cd 01:25`** | la seule des 371 dont l etiquette change selon la configuration — et le changement est **invisible aux comptes** (la couverture reste a 100 % dans les deux cas). Un golden qui ne porterait que des nombres validerait la configuration fausse. |
| **les 8 morts a source-victime, tag ET credit** | c est leur COUPLE qui est la mesure : la doctrine dit que le jeu credite un autre joueur pendant que la source appartient a la victime. Garder le tag seul ne verifierait que la moitie. |
| **le controle negatif** | 1 candidat sur 208 913 712 bits de paquets ou la cible est absente par construction. Sans lui, un decodeur devenu permissif passerait tous les autres tests. La borne du test est LARGE (20 par film) : on veut attraper un decodeur devenu permissif, pas une variation d une unite. |

Les garde-fous ont ete verifies en les faisant echouer volontairement : corrompre une ancre dans
un golden fait tomber **trois** tests independants.

---

## 10. HORS PERIMETRE — ASSUME, PAS OUBLIE

- **Le BTB.** Decision utilisateur : le perimetre est le 4v4. Le code ne casse pas en BTB, il ne
  l optimise pas, et il refuse d y publier ligne par ligne.
- **Le nommage des sources sans nom.** Elles sortent `Autres`. Un encodage existe quelque part ;
  ce n est pas la priorite du lot.
- **La peremption du catalogue.** Risque mineur. La detection reste posee et testee ; elle n est
  pas presentee comme un entretien regulier.
- **Le branchement lui-meme.** Le paquet n a **aucun appelant** dans l application. C est
  volontaire : il devait pouvoir etre teste seul avant d etre cable. Les DEUX BOUTS du cablage
  sont pourtant deja ecrits, compiles et testes — le pont de telechargement
  (`internal/sync/killsource_bridge.go`) et l ecriture en base (§12) — il ne manque que le
  collecteur qui les relie. Le §8 et le §12 disent exactement ce qu il restera a ecrire.

---

## 11. OU CHERCHER LE DETAIL

| question | fichier |
|---|---|
| la doctrine, les deux verites, ce que le brancheur ecrit | `internal/games/halo_infinite/film/killsource/doc.go` |
| pourquoi la voie sequentielle est portee alors que le balayage suffirait | meme fichier, section << DECISION DE PERIMETRE >> |
| les seuils de sante, leur derivation, le point aveugle | `internal/analysis/filmdec/killhealth.go` |
| les chiffres, film par film | `internal/games/halo_infinite/film/killsource/testdata/*.golden` |
| la table, sa vue `_latest`, et le raisonnement de schema | `internal/migration/steps_shared_kill_events.go` |
| l ecriture en base (INSERT-only) et ce qu elle REFUSE | `internal/persist/kill_events_persister.go` |
| l assistant : ce qui est mesure, ce qui est refuse, et la reserve arithmetique | `internal/games/halo_infinite/film/killsource/assist.go` |
| les deux parts de degats : les quatre jambes, la reserve, le piege de la constante par film | meme paquet, `eventchain.go` (`killEventFields`) + RE_LOG **7ter.75** |
| pourquoi le << 31 assists contre 30 API >> ne doit plus etre cite, et ce qui le remplace | RE_LOG **7ter.76** |
| l index interrogeable du chantier de retro-ingenierie | `.ai/ETAT_DE_L_ART_KILLWEAPON.md` (§11 pour les deux parts) |
| le journal detaille (~12 800 lignes — **il s interroge, il ne se lit pas**) | `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md` |

---

## 9. QUELS CHUNKS TELECHARGER — et pourquoi la synchro actuelle ne suffit PAS telle quelle

Le manifeste `spectate` d un film declare **trois types de chunks**. Voici ce que le decodeur
utilise reellement, verifie dans le code :

| type | nom | le decodeur en fait quoi |
|---|---|---|
| 1 | `HEADER` | **rien**. Il ne l ouvre jamais. |
| 2 | `REPLICATION_DATA` | **indispensable, tous**. C est la que vivent les enregistrements de mort ET le paquet `BOT_METADATA`. |
| 3 | `HIGHLIGHT_EVENTS` | **indispensable, il n y en a qu un**. C est le kill-feed : sans lui, aucun couple tueur/victime. |

**LA SYNCHRO TELECHARGE DEJA TOUT LE NECESSAIRE — mais par DEUX APPELS SEPARES**, qui filtrent
chacun sur un seul type :

```
GetMatchFilm            -> map[int]filmChunkData   ChunkType == 2 UNIQUEMENT   (halo_client_film.go l.157)
GetHighlightEventsChunk -> []byte                  ChunkType == 3 UNIQUEMENT   (halo_client_film.go l.244)
```

Le decodeur, lui, veut **UNE seule sequence**. C est tout le role du pont
`internal/sync/killsource_bridge.go` (ecrit, compile, **sans aucun appelant**).

> ⚠ **MIS A JOUR LE 2026-08-01 (J4 session 1).** Le pont vit desormais dans
> `internal/sync/killcollector/bridge.go` (`killcollector.ChunkSourceForMatch`) — la racine de
> `internal/sync` est gelee a 80 fichiers par `TestSyncRootPackageFrozen`, le neuf va dans un
> sous-paquet. Et il n assemble PLUS les deux appels : il passe par
> `haloclient.GetFilmChunks`, qui rend les **trois** types en UNE lecture de manifeste — donc
> **l en-tete (type 1) aussi**, que ni `GetMatchFilm` ni `GetHighlightEventsChunk` ne prenaient
> (decouverte J0.1 : `cmd/fetch_film_chunks` avait le meme trou). Il a un appelant : le
> collecteur, `internal/sync/killcollector/collector.go`.

### Les trois pieges, tous verifies dans le code

1. **Le chunk HIGHLIGHT n est PAS dans la map de `GetMatchFilm`.** Un branchement qui n appellerait
   que celle-ci aurait 100 % des paquets de replication et **aucun kill-feed** — donc
   `ErrNoKillFeed` sur tous les matchs. C est l erreur la plus probable, et elle est silencieuse
   au telechargement.
2. **`GetHighlightEventsChunk` rend la VERSION MAJEURE du film en 2e valeur, pas son index.** Ce
   n est pas genant : le decodeur localise le HIGHLIGHT **par son contenu** (le chunk qui produit
   le plus d evenements `kill`), donc sa position dans la sequence est libre.
3. **Le cache disque local ne stocke QUE les chunks de replication.** Le HIGHLIGHT est
   re-telecharge a chaque appel. Sans consequence fonctionnelle, mais a savoir si on compte les
   requetes.
   > ⚠ **CORRIGE LE 2026-08-01 (J4 session 1) — CET ENONCE EST FAUX SUR LE CACHE ACTUEL.**
   > Croisement des 951 manifestes en cache avec les fichiers reellement presents sur disque :
   > les **949** films utilisables (2 repertoires vides) portent **l en-tete (type 1), TOUTES
   > les replications (type 2) ET le kill-feed (type 3)**. 949/949 sur les trois criteres.
   > Verifie ensuite fichier par fichier sur `000d5950` (type-3 = index 27), `4f77afc1`
   > (index 62) et `9b191a7f` (index 32). **Consequence : un backfill local est integralement
   > HORS LIGNE** — ni reseau, ni tokens, ni CDN, donc pas de risque d expiration en cours de
   > route. L enonce reste vrai de ce que le cache Python ECRIVAIT ; il est faux de ce que le
   > cache CONTIENT aujourd hui.

A noter : les deux appels refont chacun `fetchFilmManifest`, soit un aller-retour HTTP de plus.
Correct, pas optimal. Si le cout devenait genant, la voie propre serait d exposer une methode qui
rend les deux types en une passe — **pas** de mettre le manifeste en cache global.

### Le branchement, en entier

```go
src, found, err := sync.ChunkSourceForMatch(ctx, client, matchID)
if err != nil || !found {
    return err // film absent : cas NORMAL, tous les matchs n en ont pas
}
res, err := killsource.Decode(ctx, matchID, src, nil) // nil = configuration GELEE
if err != nil {
    return err
}
for _, k := range res.Kills {
    // k.Feed.Killer  : la verite KILL-FEED (qui recoit le credit)
    // k.Source.Tag   : la verite SOURCE    (d ou vient le degat fatal)
    // k.Diverges     : les deux ne designent pas le meme responsable
}
```

**L ecriture per-match passe OBLIGATOIREMENT par `internal/persist`** (regle anti-corruption ART,
ADR 0019/0030). La table cible est **tranchee et ecrite** : `shared.match_kill_events`. Tout le
detail au §12.

---

## 12. L ECRITURE EN BASE — `shared.match_kill_events`

**Ecrit, compile, teste, SANS AUCUN APPELANT** — comme le pont de telechargement. Le collecteur
qui relie les trois morceaux n existe pas encore : c est le decodeur qui se fait valider d abord.

### 12.1 Le cablage complet, du telechargement a l ecriture

```go
// (1) TELECHARGER — le pont assemble les chunks de replication ET le kill-feed en UNE sequence.
src, found, err := sync.ChunkSourceForMatch(ctx, client, matchID)
if err != nil || !found {
    return err // film absent : cas NORMAL, tous les matchs n en ont pas
}

// (2) DECODER — `nil` = la configuration GELEE.
res, err := killsource.Decode(ctx, matchID, src, nil)
if err != nil {
    return err
}

// (3) TRADUIRE — une ligne par mort. Le decodeur ne rend que des NOMS : la resolution
//     nom -> xuid est la responsabilite du COLLECTEUR, contre le roster du match. Un xuid vide
//     n est pas une erreur : c est le cas normal d une victime BOT.
pass := persist.KillSourceBatch{
    MatchID:     matchID,
    DecoderRev:  decoderRev,               // requis : dit QUELS matchs redecoder plus tard
    Publishable: res.LineByLinePublishable(),
    Deaths:      make([]persist.KillEventInsert, 0, len(res.Kills)),
}
for _, k := range res.Kills {
    pass.Deaths = append(pass.Deaths, persist.KillEventInsert{
        TimeMS:             k.TimeMS,
        VictimGamertag:     k.Victim,
        VictimXUID:         xuidOf(k.Victim),       // "" si bot / non resolu -> NULL
        FeedKillerGamertag: k.Feed.Killer,
        FeedKillerXUID:     xuidOf(k.Feed.Killer),
        FeedPresent:        k.Feed.Present,
        AssistGamertag:     k.Assist.Name,
        AssistXUID:         xuidOf(k.Assist.Name),
        AssistKnown:        k.Assist.Known,          // FAUX = ON NE SAIT PAS
        AssistRejected:     k.Assist.Rejected,
        AssistIndex:        indexOrNil(k.Assist.Index), // -1 => nil, JAMAIS -1 en base
        AssistExtra:        k.Assist.Extra,          // garde-fou, §12.4
        KillerDamagePct:    pctOrNil(k.KillerDamage), // nil quand Known == false
        AssistDamagePct:    assistPctOrNil(k),        // nil AUSSI si l assistant n est pas NOMME
        SourceTag:          k.Source.Tag,            // l identifiant BRUT, jamais l etiquette
        SourceCategory:     k.Source.Category.Name(),
        Diverges:           k.Diverges,
        ReadPath:           string(k.Read.Path),
        ReadOrigin:         string(k.Read.Origin),
    })
}

// (4) ECRIRE — INSERT purs, sur le writer RW shared (lease deja acquis par l appelant).
if err := persist.NewKillSourcePersister(sharedDB).PersistPass(ctx, pass); err != nil {
    return err
}
```

**LES TROIS TRADUCTIONS QUI NE SONT PAS DES COPIES DE CHAMP** — ce sont les seules du cablage, et
chacune protege une distinction que le schema affirme (**NULL veut dire « pas mesure », jamais
« zero »**) :

```go
// -1 = le champ assistant est ABSENT du kill-event. Ecrire -1 en base fabriquerait un indice.
func indexOrNil(i int) *int { if i < 0 { return nil }; return &i }

// Known == false = AUCUN kill-event attache a cette mort : on n a pas mesure.
func pctOrNil(d killsource.DamageShare) *uint8 {
    if !d.Known { return nil }
    v := uint8(d.Pct)   // AUCUN plafond a 100 : 1,7 % des lignes vont jusqu a 228
    return &v
}

// PIEGE : `AssistDamage.Known` peut valoir VRAI avec un `Assist.Name` VIDE — l assistant a ete
// REFUSE (`Rejected` non vide). Le champ etait present, la part est mesuree, c est son PORTEUR
// qu on refuse de nommer. Le persister REFUSE la ligne dans ce cas, et il a raison.
func assistPctOrNil(k killsource.Kill) *uint8 {
    if k.Assist.Name == "" { return nil }
    return pctOrNil(k.AssistDamage)
}
```

⚠ `uint8` plafonne a 255 pour un maximum MESURE a 228 : la marge existe mais elle est mince. Si une
valeur superieure apparaissait, c est le TYPE qu il faudrait elargir — pas la valeur qu il faudrait
plafonner (RE_LOG 7ter.75 (5)).

Variante batch : `builder.SetKillSource(&pass)` puis le chemin `BatchQueue` habituel — le
`CombinedPersister` ecrit le sous-batch dans la meme fenetre de lease que le reste du shared.
`SetKillSource` **remplace** (et n accumule pas) : deux passes concatenees produiraient une liste
de morts qui n a jamais existe.

### 12.2 Le schema, et les quatre decisions qui le tiennent

| decision | pourquoi |
|---|---|
| **une ligne par MORT**, jamais par paire | l agregat se retrouve par `GROUP BY`, l inverse est impossible. Et `killer_victim_pairs` etait DEJA un journal : `kill_count` vaut 1 sur ses 248 566 lignes, `SUM(kill_count)` y est un `COUNT(*)` deguise. |
| **le TAG brut**, pas l etiquette | 206 des 468 entrees du catalogue sortent `Autres` aujourd hui. Stocker `Autres` figerait a jamais ce qu on ignorait ce jour-la. Re-seeder 468 entrees coute des millisecondes ; redecoder 1 325 films coute 3 a 11 heures. **La regle : on ne stocke jamais une resolution qui peut S AMELIORER.** C est pour ca que `source_category` (enum de 10 valeurs GELEE par le format de film) est stockee, elle, sans risque. |
| **les DEUX VERITES en colonnes distinctes** + `diverges` + la PROVENANCE (`read_path`, `read_origin`, `publishable`) | aucune ne corrige l autre. Et une mesure porte sur ce qu elle a mesure : la portee s ecrit AVEC le resultat, pas dans un document a cote. |
| **append-only, lecture par la vue** | INSERT purs (ADR 0026/0030). ⚠ la vue ne retient pas « la derniere ligne par mort » mais **LA DERNIERE PASSE DE DECODAGE PAR MATCH** : l unite de production est le match entier, et un `_latest` ligne par ligne melangerait deux decodages (les morts qu une passe recente ne publie plus survivraient). |

### 12.3 Comment on LIT cette table

```sql
-- TOUJOURS la vue. Une lecture brute sert les lignes des passes precedentes.
SELECT * FROM match_kill_events_latest WHERE match_id = ?;

-- l agregat que faisait killer_victim_pairs, en juste :
SELECT feed_killer_xuid, victim_xuid, COUNT(*) AS kills   -- COUNT(*), pas SUM(kill_count)
FROM match_kill_events_latest
GROUP BY 1, 2;
```

La vue ajoute une colonne calculee, **`damage_pct_residual`** — un RESTE ARITHMETIQUE, et rien de
plus (`99 - tueur - assistant` avec assistant, `100 - tueur` en solo : le 99 est la signature du
double arrondi de deux parts complementaires). Elle est calculee **la et nulle part ailleurs** :
le CHOIX DU TOTAL depend du cas, donc chaque lecteur qui la recopierait aurait sa propre chance de
se tromper.

⚠ **Elle s appelait `unattributed_damage_pct` — « part que personne ne se voit crediter ». Ce nom a
ete retire : aucune mesure ne porte cette interpretation.** Le chemin de donnees entre le
kill-event du film et `KillerPercentageDamageDone` / `AssistantPercentageDamageDone` n est pas
demontre (quatre jambes convergentes, pas une chaine d appels), et rien ne dit qu il y ait eu un
« qui » derriere le reste. La colonne est gardee parce qu elle centralise le piege du total et
qu elle est le signal par lequel se manifesterait un troisieme contributeur — pas parce qu on sait
ce qu elle mesure. **Elle peut etre NEGATIVE et n est pas bornee** (1,7 % des kill-events portent
une part > 100, jusqu a 228 ; les borner cacherait la population qui contredit l interpretation).
Elle vaut **NULL** des qu un terme manque : part du tueur non lue, `assist_known = FALSE` (le total
applicable est inconnu), ou assistant NOMME dont la part n a pas ete lue.

⚠ **La verite SOURCE est FACULTATIVE en base** : `source_tag` / `source_category` / `diverges` sont
NULLABLES, NULL = « non mesure ». 949 films en cache pour 1 325 matchs porteurs de paires, et les
films Theater expirent cote serveur : exiger la source refuserait le journal de ~28 % des matchs,
c est-a-dire en ferait la CONDITION D EXISTENCE d une mort — la hierarchie que la doctrine des deux
verites a egalite interdit. Consequence : la table a **deux producteurs**, le decodeur de film (les
deux verites) et le chemin `highlight_events` (le credit seul, `read_path = 'highlight-events'`),
distingues par la provenance. ⚠ la vue retient LA DERNIERE PASSE : un producteur credit-seul qui
repasserait apres le decodeur effacerait la source de la lecture.

⚠ **L absence d assistant se lit dans `assist_gamertag` / `assist_known`, JAMAIS dans
`assist_damage_pct`** : sans assistant, ce bloc porte une constante par film qui ne signifie rien —
et elle vaut 20 sur certains films, un pourcentage parfaitement credible. Le persister refuse de
l ecrire, mais la regle de lecture reste a connaitre.

**Trois etats de l assistant, pas deux** : `assist_known = FALSE` (on ne sait pas) /
`assist_known = TRUE` + `assist_gamertag NULL` (pas d assistant, MESURE) / `assist_gamertag`
renseigne. Les confondre fabriquerait des faits « 0 assist » qui n ont jamais ete observes.

### 12.4 Le garde-fou a surveiller

```sql
SELECT SUM(assist_extra_count) FROM match_kill_events_latest;   -- doit valoir 0
```

Le schema PARIE qu une mort ne porte qu un assistant (colonnes simples, pas de LIST DuckDB dont le
cout se paierait a chaque lecture). Ce compteur est la surveillance de ce pari — **le jour ou il
bouge, c est le declencheur de migration** vers une table fille.

⚠ **SON DENOMINATEUR EST « LES KILL-EVENTS ATTACHES », PAS « LES MORTS »** — et la version
precedente de ce guide ecrivait « cardinalite 1 sur 100 % de la population », ce qui est faux. La
grammaire ne lit qu **UN** emplacement d assistant PAR KILL-EVENT : le seul surplus observable est
celui de **DEUX kill-events attaches a la meme mort** nommant des assistants differents.

```
sa nullite prouve  *aucune mort n a recu deux ENREGISTREMENTS nommant des assistants distincts*
elle NE prouve PAS *aucune mort ne porte deux assistants*
```

La seconde affirmation n est mesuree par personne, et **une reserve ARITHMETIQUE lui est opposee** :
sur `9b191a7f`, 22 assistants nommes + 6 morts non lues = **28 < 30 a l API** — soit une mort a deux
assistants, soit certains « pas d assistant » sont faux. Non tranche (RE_LOG 7ter.76 (4)(6)).

⚠ **CE COMPTEUR A LONGTEMPS ETE MUET** : aucun champ de ligne ne l alimentait, donc
`SUM(assist_extra_count)` valait zero **par construction**, jamais par mesure. Il est desormais
porte par `Kill.Assist.Extra`, et un test FABRIQUE le cas que le corpus n a jamais produit pour
prouver qu il PEUT se declencher (`TestAssistExtraNEstPasMuet`). **Un garde-fou muet est pire que
pas de garde-fou : il rassure.** La donnee etant derivee de chunks en cache, cette migration se
fera par un redecodage, et le cout d un `ALTER` a ete mesure, pas suppose : la vue suit les
colonnes ajoutees sans qu il faille la re-creer (test `ViewFollowsAddedColumn`).

### 12.5 Ce que le persister REFUSE, et pourquoi

Chaque refus protege une propriete que le schema affirme : victime vide, instant negatif, tag nul,
categorie vide, **portee vide** (`read_path` / `read_origin`), **assistant nomme alors que
`AssistKnown = false`** (cela detruirait les trois etats), **part d assistant sans assistant**
(le piege ci-dessus), `AssistExtra` negatif, `DecoderRev` vide. La validation passe **avant** la
transaction : un refus ne laisse aucune ligne derriere lui.

⚠ **CE QUI N EST PLUS REFUSE : une part de degats > 100.** Le plafond a existe, au motif qu une
valeur hors 0..100 « signalait une lecture au mauvais endroit ». C etait l INTERPRETATION qui
plafonnait la donnee : **1,7 % des kill-events attaches a de vraies morts nommees vont jusqu a
228**. Le cout du plafond etait dur — la validation passant avant la transaction, **une seule ligne
a 228 faisait echouer la passe ENTIERE d un match**. Il est retire ; la population > 100 reste
interrogeable et c est elle qui dira si la lecture « pourcentage » tient (RE_LOG 7ter.75 (5)).

Ce qui est en revanche **accepte, et doit l etre** : `victim_xuid` NULL, et `feed_present = false`
avec un `feed_killer_gamertag` RENSEIGNE. C est le cas normal d une mort de BOT — le KILL est au
kill-feed, c est la MORT qui n y est pas. `killer_victim_pairs` ne sait pas representer ce cas :
elle contient **0 ligne** de bot en prod, cote tueur comme victime.

⚠ **LE CAS SYMETRIQUE EXISTE DEPUIS RE_LOG 7ter.79 ET IL N EST PAS ENCORE CABLE** :
`read_origin = 'tueur-bot'` (`killsource.OriginBotKiller`), 4 lignes sur la serie de reference.
La MORT y est au kill-feed (`feed_present = TRUE`, `victim_xuid` renseigne) mais le **KILL** n y est
pas : `feed_killer_gamertag` porte le nom du bot lu dans BOT_METADATA, et **`feed_killer_xuid` doit
donc etre NULL** — un bot n a pas de XUID. **A verifier avant branchement** : le persister
accepte-t-il un tueur nomme sans XUID ? Cote decodeur la population est fermee et testee ; cote
base, ce point n a pas ete instruit par ce lot.

### 12.6 Le sort de `killer_victim_pairs`

**Elle reste.** Rien ne la supprime, rien ne la modifie. Son retrait suppose, dans cet ordre :
(1) un collecteur remplit `match_kill_events` ; (2) les 5 lecteurs « cumul » passent a `COUNT(*)`
sur la vue ; (3) les 2 lecteurs « journal » (match-view, penalite de depart LUSR v2) basculent — ils
y gagnent l assistant, la source du degat et les morts de bot ; (4) la sonde de presence
(`internal/sync/events_replay.go`) bascule ; (5) **alors seulement** un `DROP TABLE` dans une
migration dediee. Tant que (1) n est pas fait, tout retrait serait une perte seche.
