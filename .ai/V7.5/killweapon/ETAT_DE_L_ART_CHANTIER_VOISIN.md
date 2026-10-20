# ÉTAT DE L'ART DU CHANTIER VOISIN — ce que `filmdec-killweapon` a établi

> Écrit le 2026-07-27. Index de renvoi vers le worktree
> `.claude/worktrees/filmdec-killweapon`, **en lecture seule**.
>
> Raison d'être : deux chantiers travaillent sur le même format de film sans partager leur état.
> Ce document a été ouvert après avoir constaté qu'une question que je croyais ouverte — le
> nommage de l'arme d'une Gungoose — était résolue chez eux depuis des semaines, et qu'une
> structure sur laquelle je venais de buter (`sofd`) était déjà parcourue par leur code.
>
> **Règle qui en découle** : avant d'ouvrir une piste sur le format de film, chercher ici.

## LEURS DOCUMENTS, ET CE QU'ON Y TROUVE

Chemins relatifs à `.claude/worktrees/filmdec-killweapon/`.

| document | contenu | quand le lire |
|---|---|---|
| `.ai/ETAT_DE_L_ART_KILLWEAPON.md` | **L'index maître**, 663 lignes. Questions résolues, pistes mortes, patrons d'erreur, chiffres de référence | Toujours en premier |
| `.ai/V7.5/killweapon/GUIDE_KILLSOURCE.md` | Mode d'emploi du paquet `killsource` : branchement, dénominateurs, ce qu'il faut croire et ce dont il faut se méfier, écriture en base | Avant de consommer leurs sorties |
| `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md` | Le journal complet, > 7 100 lignes. **À interroger, pas à lire** | Pour retrouver le détail d'une section citée |
| `.ai/V7.5/film_re/RECETTE_DECODAGE_FILM_CHUNKS.md` | Format des morceaux, registre ECS, table des 64 noms de composants | Décodage bas niveau |
| `.ai/V7.5/film_re/RE_EXE_GHIDRA_FINDINGS.md` | Trouvailles dans l'exécutable | Avant d'ouvrir Ghidra |
| `.ai/PONT_SONORE_ARMES.md` | Nommage par les banques sonores Wwise | Nommage d'arme |
| `.ai/GRENADE_MELEE_DETECTION.md` | Détection des lancers et de la mêlée | Événements |
| `.ai/V7.5/film_re/KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md` | États par défaut des archétypes dans les images-clés | Lecture d'image-clé |
| `.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` | Géométrie de carte depuis les modules | Chantier structure |
| `.ai/V7.5/cartes/INVESTIGATION_MAP_ZONE_CALLOUT_NAMES.md` | Noms de zones de carte | Libellés de zone |

## CE QU'ILS ONT RÉSOLU ET QUE JE CROYAIS OUVERT

### Les armes de véhicule — la Gungoose est leur cas d'ancrage

`ETAT_DE_L_ART_KILLWEAPON.md` §2.5, journal 7ter.45 à 7ter.48.

**Règle R-VÉHICULE** : un `weap` est un armement de véhicule si un `vehi` le référence **ou** s'il
pend à `vcdd -> sofd -> sofa -> uwfa -> weap`. **62 `weap` sur 194** : 46 par un `vehi` direct,
16 par la chaîne `vcdd`. **L'ancre Gungoose passe exclusivement par la chaîne `vcdd`.**

Force : disjonction totale d'avec le catalogue des armes de joueur — **0 sur 62** portent un nom
du catalogue, attendu 10,2, `p = 1,1e-06`.

Tourelle contre fixe, établi par deux signaux indépendants (classe ASCII du `weap`, nomenclature
de la banque du châssis) : 22/23 et 17/22. **Ghost = fixe**, tourelle UNSC arrachée = tourelle.

### Les grenades — leur table est identique à la mienne

`ETAT_DE_L_ART_KILLWEAPON.md` §2.3, journal 7ter.37/44/47/47ter.

    entree 0 = FRAG · 1 = PLASMA · 2 = DYNAMO (choc) · 3 = POINTES (Spike)

**C'est exactement la table établie de mon côté** (`V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md` §1 et §8), par
deux chaînes qui n'ont rien en commun avec la leur : appariement de 35 lancers aux décréments de
compteur, et lecture de la table `grenade_types` du binaire.

**Trois chaînes indépendantes, une seule table.** Leur découpage repose sur l'**adressage** (table
`tagref`, période d'entrée `0xd0`, block de 832 octets = 4 × `0xd0`), pas sur l'ordre d'une liste —
distinction qu'ils ont payée cher et qui est reprise plus bas.

Détail utile : chaque entrée a sa banque sonore propre, mesurée sans filtre — `kineticunsc`,
`plasma` + `grn_cv_PLASMAGRENADE`, `shock` + `grn_un_LIGHTNINGGRENADE`, `kineticbanished`.

### Les objets explosifs

§2.4. **Règle R-DESTRUCT** : le modèle `hlmt` du `weap` déclare directement un `jpt!`. Critère
rare : 40 `hlmt` sur 6088, 12 `weap` sur 194. L'index d'entrée (période `0x1d8`) est le **type
d'énergie** du baril, dénominateur 7 et non 5.

Point de méthode qu'ils tirent de là : le déclencheur — lancé ou tiré dessus — **n'est pas dans le
tag**. Le même tag couvre un lancer et un tir. Le geste appartient au tueur, le dead-state à la
victime.

### Les assistances — livrées, avec une distinction à respecter

`V7.5/killweapon/GUIDE_KILLSOURCE.md`. Une seule fonction publique :

    src := killsource.MemoryChunks(chunks)
    res, err := killsource.Decode(ctx, matchID, src, nil)   // nil = configuration gelee

Champs d'assistance : `Assist.Known`, `Assist.Name`, `Assist.Index`, `Assist.Rejected`,
`Assist.Extra`, plus `KillerDamage` et `AssistDamage`.

**`Known = false` veut dire « on ne sait pas », JAMAIS « pas d'assistant ».** Vide avec
`Known = true` veut dire « pas d'assistant », et c'est une mesure. Confondre les deux est une
régression : ils ont explicitement payé pour les distinguer.

Couverture : assistants nommés à 17/17 et 29/29 en Arena, mais **51 %** en BTB (62 pour 122 à
l'API). L'assistant et les deux parts de dégâts tombent sous la même porte que le reste.

## LA CONNEXION QUE NI EUX NI MOI N'AVIONS FAITE — le groupe de tags `sofd`

Leur chaîne d'armement de véhicule traverse `sofd`. De mon côté, l'identité de capacité spartan se
résout par un parcours du bloc `sofd` (`V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md` §9 : `FUN_1407E7648`
compare `entrée+0x18` au handle de définition et rend le rang).

**Deux chantiers ont buté sur la même structure sans le savoir.** Si `sofd` est la table
d'équipement et d'armement d'un match, alors le rang de capacité et l'armement de véhicule se
lisent au même endroit — et le nommage des capacités, que j'ai déclaré fermé côté exécutable,
pourrait s'ouvrir par le chemin de tags qu'ils parcourent déjà.

**C'est la piste la plus prometteuse pour combler ma table de capacités** (4 index sur 11
capacités connues). Première question à leur poser : leur outillage sait-il énumérer un bloc
`sofd` ?

## LEURS PATRONS D'ERREUR — à confronter aux miens

Ils rangent huit erreurs sous trois patrons. `ETAT_DE_L_ART_KILLWEAPON.md` §4. Plusieurs
recoupent exactement ce que j'ai consigné dans `V7.5/film_re/METHODE_RETRO_INGENIERIE_FILM.md` — la
convergence de deux chantiers indépendants sur les mêmes pièges vaut d'être notée.

| leur patron | ce qu'il dit | mon équivalent |
|---|---|---|
| **A — circularité instrumentale** | Une justesse mesurée contre un oracle de même logique ne prouve rien. Toujours une **baseline triviale** et un oracle de logique différente | Ma règle 2 (deux chaînes indépendantes) et ma règle 4 (mesurer les faux positifs) |
| **B — conclure sans avoir énuméré** | « Le format ne porte pas X » alors que la mesure dit seulement « notre lecteur ne trouve pas X ». **L'erreur symétrique existe aussi** | Mon erreur A (prémisse jamais retestée) — même piège, ils l'ont pris dans les deux sens |
| **C — l'événement physique pas écrit** | Chercher dans le champ d'une victime une information qui appartient au geste d'un tueur | Pas d'équivalent chez moi. **À reprendre** |

**Trois de leurs règles que je n'avais pas, et qui valent d'être adoptées :**

1. **« Une mesure de concordance ne peut pas servir à la fois de score et de filtre. »** Un filtre
   de doctrine a masqué le cas qu'il devait attraper — une comparaison de chaînes a matché sur le
   mot « hammer » et compté concordante une ligne fausse.
2. **« Une liste de coupables lue sur une marche morte ne mesure pas ce qui casse, elle mesure où
   on meurt. »** Ne jamais interpréter une distribution de causes avant d'avoir fait tourner le
   décodeur à son régime nominal.
3. **« Avant d'ouvrir une piste, greper le journal. »** Une réponse déjà écrite a été re-cherchée
   2 400 lignes plus haut, par trois agents successifs et deux vérifications adverses. C'est
   l'utilisateur qui a dit « relis notre doc ». **C'est exactement l'erreur que j'ai commise
   aujourd'hui sur les véhicules.**

### Leur méta-patron le plus coûteux, et il me concerne directement

> **« Deux référentiels qui décrivent le même objet, jamais mis en regard. »**
> Une capture runtime nommait des décalages de structure ; la grammaire du fil parlait de
> positions de bit. **Les deux décrivaient la même paire de nombres**, et personne n'a superposé
> les deux repères. Ce n'était pas une donnée manquante, c'était un changement de repère jamais
> écrit. Coût annoncé : **six semaines**.

J'ai commis la version voisine de cette erreur : la capture décrivait un film, le POC un autre, et
je n'ai pas comparé les identifiants. **Écrire le changement de repère avant de conclure que l'une
des deux sources ne dit rien.**

### Leur protocole de correction de la vérité terrain

Le décodeur n'a le droit de corriger la vérité terrain que si les trois conditions sont réunies :

1. la correction vient d'un support **indépendant** de tout ce qui produit l'étiquette ;
2. ce support était **déjà concordant** sur d'autres points avant que la contradiction ne surgisse ;
3. la contradiction est **soulevée par nous** et posée comme question ouverte, jamais tranchée seul.

> Sinon, c'est le décodeur qui a tort. **Si la vérité terrain cesse de pouvoir falsifier, plus rien
> ne le peut.**

## LEURS CHIFFRES DE RÉFÉRENCE

Mesure du 2026-07-26, complétée le 2026-07-27.

- **371 couples sur 371** sur quatre films, décodage entièrement hors ligne, catalogue et nommage
  embarqués — preuve faite jeu renommé.
- 25 confrontations de vérité terrain, dont un cas où le décodeur a eu raison **contre** le
  kill-feed du jeu (5 sur 5), et un couple fabriqué dont la vraie victime était un bot.
- Le paramètre de calibration est réellement dépendant de la carte : quatre films, quatre couples
  distincts, zéro variable d'environnement.

## CE QU'ILS N'ONT PAS, ET QUE J'AI

Pour que l'échange aille dans les deux sens :

- La **carte mémoire du loadout joueur** : base `0x7F0`, pas `0x90`, quatre emplacements, avec les
  décalages de chargeur, réserve, jauge et surchauffe (`V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md` §7).
- La **grammaire en union d'`i30`/`i33`** et le fait que la jauge d'énergie est une fraction dans
  [0,1] sur 12 bits.
- La **table arme → chargeur/réserve** pour 16 armes.
- Le **sélecteur d'emplacement `i42`** et son sens, établi par oracle interne.
- La **localisation positionnelle** des lectures : 98,4 % de rappel, 0 faux positif sur 650 641
  positions.
- L'**archétype 40** comme archétype véhicule, avec le constat que position, vitesse, vie et
  bouclier d'un véhicule sont déjà décodables (`V7.5/film_re/VEHICULES_ARCHETYPE_40.md`).
