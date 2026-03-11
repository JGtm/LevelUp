# Comment fonctionne le parser d'armes

> État : branche `analysis/weapon-parser-rewrite` — 2026-03-11

---

## Le problème à résoudre

Le jeu (via l'API Halo) donne le **nombre** de kills par type d'arme pour un match,
mais pas le détail kill par kill. Pour savoir "avec quoi JGtm a tué à t=3:24", la
seule source est le **film** du match — un fichier binaire téléchargeable depuis
les serveurs Xbox.

Le parser lit ce fichier binaire et relie chaque kill à son arme.

---

## Les deux sources d'information dans le film

Le film d'un match est découpé en **chunks** d'environ 19 secondes. Chaque chunk
contient deux types de données utiles :

### Section 1 — L'état des joueurs (Formula A)
Un snapshot périodique qui dit : "à cet instant, le joueur N tient telle arme".
C'est une photo d'état, pas un événement. Elle se met à jour quand le joueur
change d'arme ou ramasse une arme.

### Section 2 — Les tirs du POV (fire events)
Chaque fois que le **POV** (le joueur dont c'est le film) appuie sur la gâchette,
ça génère un event qui contient :
- L'instant du tir (en ms)
- L'arme utilisée au moment du tir

**Limitation fondamentale** : la Section 2 ne contient que les tirs du POV.
Les adversaires et coéquipiers ne tirent pas dans "son" film — ou du moins pas
de manière fiable et continue.

---

## Les deux chemins d'attribution

Pour chaque joueur dans un match, le pipeline choisit un chemin selon que c'est
le POV ou non.

### Chemin A — POV (Section 2, fire events)
**Pour qui** : le joueur qui a lancé le backfill (le "propriétaire" du film).

**Comment** :
1. On scanne tous les fire events du film pour `player_index = 1` (le POV)
2. Pour chaque kill à t=T, on cherche le dernier fire event dans `[T−5s, T]`
3. L'arme de ce fire event devient l'arme du kill

**Fiabilité** : élevée. Fenêtre de 5s pour capturer les armes à delayed damage
(Cindershot, Ravager, etc.). Un delta court = haute confiance, un delta long =
confiance réduite (peut-être un tir raté juste avant ?).

### Chemin B — Coéquipiers T1 (Section 1, snapshots)
**Pour qui** : tous les autres joueurs dont on retrouve le `player_index` dans le film.

**Comment** :
1. On construit une "timeline" : pour chaque chunk, quelle arme tenait chaque joueur
2. Pour chaque kill à t=T, on regarde le chunk qui couvre T
3. L'arme en snapshot pour ce joueur dans ce chunk = arme du kill

**Fiabilité** : moyenne. Si le joueur a changé d'arme au milieu du chunk (~19s),
on ne sait pas si le kill était avant ou après le changement → `confidence=medium`.

---

## Comment on identifie les armes (le mapping hex → nom)

Chaque arme dans le film est représentée par **8 octets** (un identifiant binaire).
La grande majorité des armes "standard" partagent le même suffixe (`42c9679f` pour
les 4 derniers octets). Les armes spéciales (variantes d'Energy Sword, Gravity Hammer)
ont des suffixes différents.

`WEAPON_ID_MAP` dans `_weapon_data.py` associe ces 8 octets au nom de l'arme.
Seules les armes **confirmées par investigation directe sur le film** sont dans ce
dictionnaire. Un hex inconnu reste inconnu — le kill est stocké avec `confidence=low`
et son ID numérique brut.

---

## Les sentinelles (melee, grenade, véhicule)

Ces kills n'ont pas de fire event et ne montrent pas d'arme dans la Section 1. Ils
sont détectés autrement : via les **médailles** obtenues par le joueur dans les
500ms autour du kill.

| Médaille présente | → Attribution |
|-------------------|---------------|
| Pummel, Back Smack, Ninja, Assassination, Pancake… | `weapon_id = 1` (melee) |
| Sticky Fingers, Grenadier, Boom!, Stick… | `weapon_id = 0` (grenade) |
| — (aucune médaille spécifique) | Chemin normal (fire event / snapshot) |

---

## La réconciliation API

L'API Halo donne par match le nombre total de kills arme / melee / grenade pour le
POV. En fin de traitement, on compare ce que le parser a trouvé avec ces agrégats et
on ajuste :

- **Trop de kills arme** → on rétrograde les moins sûrs de `high` à `medium`
- **Pas assez de kills melee/grenade** → on reclassifie les kills les plus incertains
  en melee/grenade pour combler le manque (Step 4b)
- **Pas assez de kills arme** → on promeut des `medium` en `high`

**Point de friction** : le Step 4b puise dans les kills les moins certains, et
les kills T1 à hex inconnu (`confidence=low`) sont les premiers candidats. Résultat :
des kills réels avec une arme inconnue peuvent être reclassifiés en grenade/melee
pour "équilibrer les comptes" avec l'API.

---

## Pourquoi il y a des weapon_id NULL en base

Un kill NULL signifie que le parser n'a trouvé **aucune info** sur l'arme :
- Chemin POV : aucun fire event dans les 5s avant le kill (vehicule, edge case)
- Chemin T1 : le film n'a encodé aucun snapshot pour ce joueur dans ce chunk

Le tableau ci-dessous montre la réalité en base (85 247 kills au total) :

| État | Kills | % |
|------|------:|--:|
| NULL — pas d'info | 54 313 | 63.7% |
| Hex inconnu (`conf=low`) | 15 377 | 18.0% |
| Arme identifiée (`conf=high`) | 10 631 | 12.5% |
| Melee / Grenade sentinelle | 4 447 | 5.2% |

Les NULL viennent quasi-exclusivement du chemin T1 pour les adversaires : le film
d'un joueur n'encode pas fiablement l'état arme des joueurs adverses.

---

## Résumé des limites actuelles

| Limite | Impact |
|--------|--------|
| Section 2 (fire events) uniquement pour le POV | Les coéquipiers et adverses ont une bien moins bonne couverture |
| Snapshot T1 sporadique pour les adverses | → 63% des kills NULL |
| 279 hex inconnus (T1 low confidence) | → 18% des kills "identifiés" mais sans nom |
| Step 4b reclassifie des hex inconnus en grenade/melee | → faux mélanges dans les stats grenade/arme du POV |
| `WEAPON_ID_MAP` à 36 armes confirmées | → tout hex hors map reste inconnu |
