# Recoupement des gestes et postures (item C.4) et zoom / mêlée sans mort (item E.5)

Date : 2026-11-23. Lecture seule : aucun code modifié, aucun test, aucun décodage.

Références :
- Notre décodeur : tête de la branche `feat/v1.6.0` (commit `daac986`). Chemins relatifs à ce dépôt.
- Copie du contributeur : tête `a66b791` de son dépôt, dossier `third_party/film-decoder/`. Lue par l'API, en lecture seule. Chemins relatifs à cette copie.
- Inventaire de référence : `.ai/V8.3/annexes_decodeur_v160/inventaire_contributeur.md` du dépôt LevelUp.

Marquage : **vérifié** = relu dans le fichier ou vérifié par recherche sur le code. **déduit** = raisonné, non rejoué.

## Suppositions et manques

- Supposé : « la manche » désigne les mouvements et gestes du joueur pendant la manche. Le brief ne définit pas le terme.
- Manque : seuls les fichiers cités ici ont été lus dans la copie du contributeur. Le reste de sa copie reste au niveau de l'inventaire.
- Manque : aucun film n'a été rejoué. Les affirmations sur les builds de film restent **déduites**.

## Tableau 1 (item C.4) : recoupement des gestes et postures

| Sujet | Son champ (contributeur) | Notre champ (`feat/v1.6.0`) | Lequel reste | À vérifier |
|---|---|---|---|---|
| Accroupi et glissade | `stances[]` de genre `crouch` et `slide`, avec `ms0` / `ms1` en millisecondes. Lu dans la vue de contrôle, glissade inférée d'après le sprint et la vitesse au sol (seuil 2,45). `replay/postures_de_controle.go` l. 3-24, 52 : **vérifié** | `stances[]` de genre `crouch` et `slide`, lus dans les composants `i29` et `i62`. `replay/document_stances.go` l. 11 et 60-62 : **vérifié**. Le même commentaire dit que ces composants ne se lisent sur aucun film de version 37 et plus. | **À décider par le pilote.** Si le commentaire est juste, nos genres sont vides sur les films récents et seul le sien en fournit. Les deux écrivent le même nom de genre, `crouch`, avec une sémantique différente : il faut trancher le nom avant de fusionner. | Rejouer sur un film récent que `i29` et `i62` sont absents. Le contributeur affirme la même chose (`replay/postures_de_controle.go` l. 9-10) : deux sources, pas de preuve indépendante. |
| Saut | Impulsion du bouton de saut lue dans la vue de contrôle (`kind` `jump`, `replay/document_gestes.go` l. 23-26). Il combine ses sauts lus et dérivés (compteurs `jumpsDerivedKept`, `Replaced`, `Contradicted`, `replay/postures_de_controle.go` l. 109-111). | `jumpDerived` : intégrale de la vitesse verticale, reconnue à 0,85 m. `replay/document_stances.go` l. 12-15 : **vérifié**. Aucun saut lu dans la vue de contrôle. | **À décider par le pilote.** Le sien ajoute le saut lu. Son mécanisme de remplacement touche notre `jumpDerived`, qui est déjà publié et a un lecteur. | Le numéro du bit de saut diffère. Notre `internal/grammar/commandes_de_controle.go` l. 9-14 : le bit de poids 4 « paraît être » le saut, et ce champ n'est pas publié. Le sien : bit 2 pour le saut et bit 0 pour l'accroupi (`internal/grammar/boutons_de_controle.go` l. 3-4, lu). **Désaccord non tranché.** |
| Bouton d'accroupi | Bit 0 de l'entrée de contrôle, lu en changements (`internal/grammar/boutons_de_controle.go`, `types/grammar_boutons.go`). | Non publié : le champ de cinq bits est écarté (`internal/grammar/commandes_de_controle.go` l. 9-14). **Vérifié.** | **Le sien** (nous ne le publions pas). Dépend du numéro de bit ci-dessus. | Le numéro de bit (voir ligne « Saut »). |
| Commande de déplacement tenue | `replay/stick_de_deplacement.go` (schéma 114) : couple brut `sx`, `sy` à chaque changement. Valeur inconnue marquée à part. | `stances[]` de genre `moveCommand` : intervalles où la commande est tenue, couple au zéro exact à `0x1f` (`replay/document_stances_commandes.go` l. 3-16 ; `internal/grammar/commandes_de_controle.go` l. 6-10). **Vérifié.** | **Les deux sont possibles** : valeurs brutes au débit du film d'un côté, intervalles tenus de l'autre. Le choix de forme revient au pilote. | Que les deux lisent les mêmes bits et le même décalage (même source de couple). |
| Rechargement | `gestes[]` de genre `reload`, message `weapon_reload` (genre 38 de la vue A), avec milliseconde et `slot`. `replay/document_gestes.go` l. 31-35 : **vérifié**. | `reloads[]` : même message, index de joueur, copies dédoublonnées et trames arrêtées comptées. `replay/document_reloads.go` l. 24-30, 36-45 ; `internal/grammar/rechargements.go` l. 81 : **vérifié**. | **À décider par le pilote.** Même message source, deux formes. Le nôtre a le dédoublonnage et les compteurs ; le sien a la milliseconde et la vie. | Si son geste garde les copies du même message. Il ne les traite pas dans le fichier lu (**déduit**). |
| Lancer amorcé | `gestes[]` de genre `throwStart` (message genre 39). `replay/document_gestes.go` l. 31-35 : **vérifié**. | Aucun geste publié. Le lecteur de charge existe (`internal/grammar/vue_a_charges_execution.go` l. 84) mais ne publie rien. Recherche sur le code : **vérifié**. | **Le sien** (pas d'équivalent publié). | Ce que publient exactement nos projectiles lancés (`replay/grenades.go`) : **déduit** qu'ils ne décrivent pas l'amorçage. |
| Coup de corps à corps | `gestes[]` de genre `melee` (message genre 40), à l'instant où le coup part. `replay/document_gestes.go` l. 23-24 : **vérifié**. | Aucun geste publié. `chargeCorpsACorps` (`internal/grammar/vue_a_charges_armes.go` l. 85-93) lit le message sans le publier. **Vérifié.** | **Le sien** (pas d'équivalent publié). | Rien de bloquant. |
| Pistes à pleine cadence | Colonnes `fullRate` (millisecondes), dans la même copie. | Aucune occurrence de `FullRate` ni `fullRate` dans `replay/` hors tests. **Vérifié** par recherche. | **Le sien** (pas d'équivalent publié). Touche le schéma des pistes : à coordonner avec le pilote. | Le lot de pistes est hors périmètre de ce tableau. |

## Tableau 2 (item E.5) : zoom et coup de mêlée sans mort

### (a) Zoom (visée lunette)

- **Déjà publié** par notre décodeur, champ `s` des points de trajectoire. Définition : `replay/document_aim.go` l. 134-163 (palier de lunette, 0 = pas à la lunette, 1 et plus = à la lunette). **Vérifié.**
- Alimentation : `replay/tracks_publication.go` l. 404-407 (`pt.S = scoped(...)`). **Vérifié.** Source : bascules `unit_zoom` lues en tête de paquet, `internal/grammar/zoom_events.go` l. 178-186 (`ZoomStateAt`, maintien borné). **Vérifié.**
- Couverture : le fichier le dit lui-même. Seules les bascules en tête de paquet sont lues. Les sorties de lunette sont sous-comptées, et le maintien borné compense sans affirmer au-delà d'un délai. Le grossissement n'est pas publié : le film ne transmet que le palier. **Vérifié** (l. 135, 160-161).
- La copie du contributeur publie le même champ `s` par le même chemin (`replay/tracks_publication.go` l. 459 dans sa copie). **Vérifié.** Nous n'avons pas comparé le détail de son `zoom_events.go` (**déduit** qu'il est identique à notre version).

**Sort : déjà couvert** (champ `s`, couverture partielle : les sorties de lunette sont sous-comptées).

### (b) Coup de mêlée qui ne tue pas

- **Publié aujourd'hui, le coup lui-même : non.** Aucun geste de mêlée dans notre décodeur. **Vérifié** (voir tableau 1, ligne « Coup de corps à corps »).
- Le dégât porté par le coup : notre `damage[]` publie chaque dégât avec son étiquette d'effet (`effect`, tag `jpt!`), l'auteur et la victime. `replay/document_damage.go` l. 25-41. **Vérifié.** Un coup de mêlée sans mort y figure donc si son message de dégât existe (**déduit**).
- Mais la classe « mêlée » n'est pas publiée dans `damage[]`. La mêlée ordinaire sort avec la catégorie `None` ; c'est le tag qui la porte (`internal/facts/killsource/kill.go` l. 120 et 206-208). **Vérifié.** Un client devrait donc relier le tag à la mêlée, ce que rien ne fait aujourd'hui (**déduit**).
- Les kills de mêlée sont comptés par le tableau des scores (`meleeKills`, `replay/document_score.go` l. 145-150). **Vérifié.** C'est un compte, pas un instant.
- Le geste du contributeur (`melee`) donne l'instant du coup, qu'il touche ou non. Il ne dit pas si le coup a touché (`replay/document_gestes.go` l. 23-24). **Vérifié.**

**Sort : couvert par le contributeur** pour l'instant du coup. Le dégât d'un coup sans mort est déjà dans `damage[]`, mais sans étiquette « mêlée ».

## Ce qui reste à décider ou à vérifier

**À décider par le pilote :**
- Le nom de genre `crouch` et `slide` : son champ (lu dans la commande) ou notre champ (lu dans les composants, absent sur les films récents si le commentaire est juste).
- Le numéro du bit de saut et d'accroupi (désaccord entre notre commentaire et son code).
- La forme du déplacement tenu : intervalles, valeurs brutes, ou les deux.
- Le rechargement : garder notre forme (dédoublonnée), la sienne (milliseconde), ou les deux sous des noms distincts.

**À vérifier sur un film (hors de ce fichier, par le pilote ou une session dédiée) :**
- Absence de `i29` et `i62` sur un film récent.
- Bit de saut dans l'entrée de contrôle.
- Présence d'un message de dégât pour un coup de mêlée sans mort, et son tag.
