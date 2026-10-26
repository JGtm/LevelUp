# PROTOCOLE — Trancher la lunette par Cheat Engine (capture vivante)

> Prêt le 2026-08-30 au soir, à exécuter le 2026-08-31. Décidé par l'utilisateur après sept
> canaux mesurés négatifs (voir `.ai/thought_log.md`, entrées « Visee lunette » phases 1 à 10).
>
> **Pourquoi ce protocole met fin au débat.** La question « le film porte-t-il l'état de lunette ? »
> a résisté à toute la statistique parce qu'elle cherchait un signal dans un flux. Ici on ne cherche
> plus : on demande au jeu **quelle instruction écrit l'octet de zoom** pendant qu'il rejoue le
> film. La fonction qui écrit EST la réponse.

## Ce qu'on sait déjà (ne pas re-chercher)

| Élément | Valeur | Source |
|---|---|---|
| Niveau de zoom COURANT | `unite + 0x461`, `char` signé | phase 3 |
| Niveau de zoom DÉSIRÉ (prime sur le courant) | `unite + 0x462`, `char` signé | phase 3, confirmé lot E |
| Valeurs | `-1` (= `0xFF`) : pas de lunette · `0`, `1`, `2` : paliers | lot E (charge utile `R(2) − 1`) |
| Le grossissement de chaque palier | appartient à l'ARME, pas à l'unité | utilisateur |

**Les trois écrivains possibles, et ce que chacun signifierait :**

| Fonction | Ce qu'elle est | Verdict si c'est elle |
|---|---|---|
| `FUN_1406db688` | applique la COMMANDE joueur : `unite+0x462 = commande[6]` | **les commandes sont rejouées depuis le film** — on remonte alors au champ porteur |
| `FUN_14110ec20` | applicateur de l'événement `unit_zoom` | un événement existe malgré 41 M de paquets recensés sans lui — recensement à refaire |
| `FUN_1404a4ab8` | transition locale (courant → désiré), appelée par le tick d'unité | **reconstruit côté client** : la question est close, cap sur l'heuristique |
| autre | — | piste neuve, remonter la pile d'appels |

## Matériel

Film de référence : `00162144`. Chronologie relevée (horloge du feed) :
**Nilton410** à la lunette sur `{41 → 46,3}`, `{49 → 52}`, `{61 → 61,8}`, `{68 → 68,8}`,
`{71 → 73}`, `{85 → 86}` — et **Madina97294** sur `{45 → 46,3}`. Le frag Counter-snipe est à 0:46.

Le serveur MCP Cheat Engine est déjà configuré (voir la référence d'installation du dépôt).
Précédent d'usage dans ce chantier : la capture CE a servi à mesurer le coût en bits du composant
de position (« total i0 = 47 bits », cf. `filmdec/offline_aim.go`).

## Étape 1 — Trouver l'octet (5 min)

1. Lancer le jeu, ouvrir le film `00162144` dans Theater, se mettre **en première personne sur
   Nilton410**, placer la lecture **avant 0:41**.
2. Attacher Cheat Engine au processus.
3. Scan de valeur, type `Byte` : chercher `255` (= `-1`, pas de lunette).
4. Avancer la lecture dans un épisode zoomé (par exemple 0:42), puis **Next Scan** sur `0` ou `1`.
5. Revenir hors épisode, **Next Scan** sur `255`. Répéter deux ou trois fois sur vos épisodes :
   ils alternent six fois en cinquante secondes, c'est un filtre très discriminant.
6. Il doit rester une poignée d'adresses. **Contrôle d'identité** : l'adresse retenue doit avoir
   une voisine immédiate à `−1` octet qui suit le même motif (le couple courant/désiré).

**Piège à éviter** : ne pas confondre l'unité observée et l'unité d'un autre joueur. Si plusieurs
adresses survivent, c'est probablement bon signe — voir l'étape 3.

## Étape 2 — LA capture décisive (le point d'arrêt en écriture)

1. Clic droit sur l'adresse retenue → **« Find out what writes to this address »**.
2. Rejouer un épisode complet (par exemple 0:49 → 0:52), entrée **et** sortie de lunette.
3. Relever **toutes** les instructions qui écrivent, avec leur adresse.
4. Pour chacune, remonter à sa fonction et la comparer au tableau des trois écrivains ci-dessus.
5. **Capturer la pile d'appels** (Cheat Engine la propose sur le point d'arrêt) : c'est elle qui
   dit d'où vient la donnée — décodage de trame, application de commande, ou logique locale.

**Le seul livrable qui compte** : le nom de la fonction écrivante et sa pile d'appels. Tout le
reste en découle.

## Étape 3 — Le contrôle qui vaut la peine (10 min de plus)

Refaire l'étape 1 pour **un joueur distant** — idéalement Madina97294, dont on sait qu'elle zoome
sur `{45 → 46,3}` et pratiquement pas ailleurs.

- Si l'octet d'un joueur **distant** suit sa propre chronologie : l'état existe par joueur pendant
  le rejeu, donc il vient du film d'une manière ou d'une autre — et l'étape 2 dit laquelle.
- S'il reste à `−1` en permanence alors que l'observé varie : **le zoom n'existe que pour l'unité
  observée**, il est reconstruit à l'affichage, et la question est définitivement close.

Ce contrôle est le plus informatif des trois étapes ; ne pas le sauter.

## Étape 4 — Selon le verdict

- **Commande rejouée** → remonter du champ `commande[6]` vers la structure de commande, puis vers
  ce qui la remplit pendant le rejeu ; croiser avec la grammaire de trame
  (`PLAN_PERCER_TRAME_FILM_2026-08-30.md`).
- **Événement** → refaire le recensement des types avec la bonne numérotation, en cherchant
  précisément l'octet de cet événement.
- **Local** → clore la question au thought_log, et livrer le cône du rejeu par estimation assumée
  (arme à lunette tenue + cible dans l'axe), affichée comme telle, jamais comme une donnée du film.

## Règles

Tout ce qui est mesuré entre au `.ai/thought_log.md` : adresses relevées, instructions écrivantes,
pile d'appels, verdict. Une capture CE n'est pas reproductible plus tard — **ce qui n'est pas noté
est perdu**. Aucun code de production n'est touché par ce protocole.
