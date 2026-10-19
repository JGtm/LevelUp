# Fiche des contrôles avant la release v8.0.0

Étape 3 du plan `.ai/PLAN_RELEASE_V8_2026-10-17.md`. Source : encadré « Contrôles visuels » et
« Contrôles techniques » de la page Notion « Backlog LevelUp ». Serveur local relancé le 19/10 sur
les bases recuites (rejeux depuis le 01/09/2025). Ouvrir les liens avec `make dev` lancé (vite :5173).

Pour chaque ligne : **OK**, **KO** (à corriger avant la release) ou **v8.1** (à reporter). Note ta
réponse en face, je m'occupe du reste.

Abréviations : « vue » = vue du match, « rejeu » = rejeu 2D de ce match.

## Matchs de référence

| Repère | Match | Carte, mode, date | Liens |
|---|---|---|---|
| A | `7344d24f` | Vagabond, Bases, 05/02 | [vue](http://localhost:5173/t/halo_infinite/players/Chocoboflor/matches/7344d24f-0154-4949-80ad-e2b781c122f1) · [rejeu](http://localhost:5173/t/halo_infinite/players/Chocoboflor/matches/7344d24f-0154-4949-80ad-e2b781c122f1/replay) |
| B | `bfecd02b` | Snowbound, Assassin, 01/09 | [vue](http://localhost:5173/t/halo_infinite/players/JGtm/matches/bfecd02b-9798-4407-aa55-05244f4c1fa0) · [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/bfecd02b-9798-4407-aa55-05244f4c1fa0/replay) |
| C | `51d3ab9f` | Launch Site, Super Fiesta, 21/01 | [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/51d3ab9f-0c70-4c49-ae6d-06a1ed6c5765/replay) |
| D | `92c950ee` | Catalyst classé, Drapeau, 08/09 | [rejeu](http://localhost:5173/t/halo_infinite/players/Nuzzles/matches/92c950ee-9aad-4471-a076-98882738f869/replay) |
| E | `d6918972` | Origin, Drapeau, 22/09 (le plus récent sur Origin en Drapeau) | [vue](http://localhost:5173/t/halo_infinite/players/JGtm/matches/d6918972-471b-4523-88cd-7b6fe24e4662) · [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/d6918972-471b-4523-88cd-7b6fe24e4662/replay) |
| F | `f2966f08`, `7b0d89c4` | Behemoth, Assassin, 01/09, les deux défaites | [rejeu F1](http://localhost:5173/t/halo_infinite/players/Chocoboflor/matches/f2966f08-f15a-4d9c-aa14-938d733173b7/replay) · [rejeu F2](http://localhost:5173/t/halo_infinite/players/Chocoboflor/matches/7b0d89c4-a068-4ea2-82b8-8f13ba0311d6/replay) |
| G | `b1ad85eb` | Domicile, Assassin, 07/09 | [vue](http://localhost:5173/t/halo_infinite/players/Madina97294/matches/b1ad85eb-8113-4639-a024-952c9d8214ee) · [rejeu](http://localhost:5173/t/halo_infinite/players/Madina97294/matches/b1ad85eb-8113-4639-a024-952c9d8214ee/replay) |
| H | `696a9d7c` | Vagabond, Bases, 17/07 | [rejeu](http://localhost:5173/t/halo_infinite/players/Madina97294/matches/696a9d7c-009f-4750-8202-f1819947aa4e/replay) |
| I | `af13e2b2` | Origin, Bases, 26/10/2025 | [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/af13e2b2-6a78-4430-8eed-6f3156148086/replay) |
| J | `396cfc92` | Illusion, Bases, 07/09 | [vue](http://localhost:5173/t/halo_infinite/players/XxDaemonGamerxX/matches/396cfc92-6f9e-403b-a46f-20bc73b38c11) · [rejeu](http://localhost:5173/t/halo_infinite/players/XxDaemonGamerxX/matches/396cfc92-6f9e-403b-a46f-20bc73b38c11/replay) |
| K | `000d5950` | Cliffhanger, Super Fiesta, 08/03 | [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/000d5950-83d9-423f-ab55-d068a7237b9f/replay) |
| L | `4898d586` | Behemoth, 4v4 à véhicules (rejeu de la démo) | [vue](http://localhost:5173/t/halo_infinite/players/JGtm/matches/4898d586-a097-4de7-9c8e-d85b01c5f884) · [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/4898d586-a097-4de7-9c8e-d85b01c5f884/replay) |
| M | `5676a9ba` | Insolence, BTB Contrôle total, 24/07 (soirée BTB) | [vue](http://localhost:5173/t/halo_infinite/players/JGtm/matches/5676a9ba-bafa-4fbb-bd3e-b957ae2ee8e0) · [rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/matches/5676a9ba-bafa-4fbb-bd3e-b957ae2ee8e0/replay) |

Introuvables dans le parc (ni au registre, ni en rejeu) : `6aadf3d5` (n° 5) et `1b08c056`
(n° 12). Pour ces deux contrôles, prends n'importe quel rejeu d'Oddball / un rejeu avec un
translocateur.

Pages : [Explorer](http://localhost:5173/t/halo_infinite/players/JGtm/explorer) ·
[Escouade](http://localhost:5173/t/halo_infinite/players/JGtm/squad) ·
[Emprise](http://localhost:5173/t/halo_infinite/players/JGtm/squad/emprise) ·
[Tactique](http://localhost:5173/t/halo_infinite/players/JGtm/ascension/tactique) ·
[Sessions](http://localhost:5173/t/halo_infinite/players/JGtm/stats/sessions)

## Contrôles visuels (encadré Notion, numérotation d'origine)

| N° | Quoi regarder | Où | Réponse |
|---|---|---|---|
| 1 | Notification Discord des rejeux : déjà traitée le 26/08 | — | OK (rien à voir) |
| 2 | Sons des bases ; 1re capture de Chocoboflor visible ; capture d'une base neutre remplie à la couleur de l'équipe qui capture | A | |
| 3 | Power-up du milieu de carte affiché, avec son compteur | D | |
| 4 | Kill au répulseur : son branché | un rejeu où un répulseur tue (à repérer) | |
| 5 | Sons Oddball / crâne branchés | un rejeu d'Oddball | |
| 6 | Modes à manches : manches gagnées/perdues affichées plutôt que les points | vue d'un match à manches (ex. E) | |
| 7 | Plusieurs manches : indicateur de manche, message et son « manche terminée » | E | |
| 8 | Fiches ultra compactes en BTB | M | |
| 9 | Véhicules visibles (Ghost) | B, L | |
| 10 | Médias calés sur la frise (mort de Chocoboflor par brail à 2:15) | B | |
| 11 | Les 7 destructions (7:19 et 7:35) | C | |
| 12 | Effet du translocateur / faille posée sur fiche et canevas | un rejeu avec translocateur | |
| 13 | Icônes d'arme dans le kill-feed | G, J | |
| 14 | Joueurs dans la bonne équipe | E | |
| 15 | Joueur en Banshee affiché (pas « en attente de réapparition ») | F1 ou F2 | |
| 16 | Page Tactique présente dans le menu L1 | menu | |
| 17 | Effets UI et son des tirs de véhicule ; cône de visée en véhicule | L | |
| 18 | Tuiles de la page Tactique : icône de l'arme, pas son nom | Tactique | |
| 19 | Blizzard : label « Tourelles automatiques bannies » retiré, tourelles en gris | (aucun match Blizzard au parc : à voir sur la page des cartes si elle existe) | |
| 20 | Tirs de véhicule depuis l'avant du véhicule | L | |
| 21 | Drapeau au sol : cercle de retour affiché | D, E | |
| 22 | Barre verticale de lecture élargie sur les frises | n'importe quel rejeu | |
| 23 | Cercle autour du drapeau au sol ; pas de tir en portant le drapeau | E | |
| 24 | Explosions de grenades cohérentes avec les lancers | E | |
| 25 | Vue match : graphe des assistances rempli, tableau des assistances retiré | vue B, vue E | |
| 26 | Bases neutres en blanc/gris avec contour (plus de bleu) | A, I | |
| 27 | Page Tactique : la carte ouverte affiche ses données | Tactique | |
| 28 | Effets UI du translocateur sur fiche et canevas | comme 12 | |
| 29 | Orientation des véhicules vs visée ; effets de tir ; taille des véhicules (+20 %) | L | |
| 30 | Sons des bases calés sur les événements | A | |
| 31 | Remplissage à la couleur de l'équipe qui capture (neutre ou adverse) | A, H | |
| 32 | `b1ad85eb` : 2 équipes (pas 3) ; armes dans le kill-feed | G | |
| 33 | Explorer : colonne rejeu, filtre `?replay=with`, largeurs | [Explorer avec rejeu](http://localhost:5173/t/halo_infinite/players/JGtm/explorer?replay=with) | |
| 34 | Vue match : les 3 onglets ; vieux lien `?tab=details` | [vue B ?tab=details](http://localhost:5173/t/halo_infinite/players/JGtm/matches/bfecd02b-9798-4407-aa55-05244f4c1fa0?tab=details) | |
| 35 | Onglet Joueurs, graphe Assistances (sans film : rien ; BTB avec film : « non disponibles ») | vue M | |
| 36 | Compteur d'amis servi à tout authentifié : laissé tel quel par défaut | — | décision : tel quel |
| 37 | Lettres des zones : Vagabond ouest=C, milieu=A, est=B (A et H identiques) ; Origin zones 0,1,2 = A,B,C | A, H, I | |
| 38 | Ligne « Temps mort » en pied de fiche | n'importe quel rejeu | |
| 39 | Infobulle « Écart cumulé au FDA attendu » + intensité ; pourquoi une image | Escouade | |
| 40 | « Frag mort » : (i) qui explique le bonus | vue B | |
| 41 | Noms des modes tous traduits dans la liste de l'Explorer | Explorer | |
| 42 | (i) sur Rendement et Résistance dans la vue | vue B | |

## Contrôles techniques

| N° | Quoi | Où | Réponse |
|---|---|---|---|
| T1 | Nouveautés derrière des capacités | audit du 19/10 (lecture du code, tests non rejoués) | **5 OK, 3 à trancher** : (a) le bouton « rejeu » de la vue match n'est caché que par l'absence de fichier, pas par la capacité `replay` (mineur, invisible pour Halo 5) ; (b) citations : aucune garde côté web, l'URL directe `/career/citations` d'un joueur Halo 5 interroge l'API des citations Infinite, et le menu bascule Citations/Commendations sur le slug (préexistant, juin) ; (c) **assistances attendues : contrairement à ce qu'on pensait, la vue match d'un joueur Halo 5 LES AFFICHE** (badge « estimé localement »), alors que Séries temporelles, Sessions et Escouade les masquent (capacité `expected_stats`). Rejeu, Tactique, Emprise, « Analyse en cours », zones nommées, mise en commun des jetons : OK. Réponse attendue pour (a), (b), (c) : corriger avant la release, ou v8.1. |
| T2 | Autosnap : la page Escouade s'ouvre sur la dernière session en escouade | Escouade | |
| T3 | « Composition exacte » dans la L2 de l'Escouade : cochée par défaut ? | Escouade | |
| T4 | Drapeau : score qui suit, bases neutres dessinées, pas d'équipe « sans équipe », grenades cohérentes, armes au kill-feed | J (le match cité), D | |
| T5 | Démo : un rejeu par mode + véhicules | fait le 19/10 (9 modes + Behemoth) | OK |
| T6 | `000d5950` : JGtm ne disparaît plus (pas de réapparition > 20 s alors qu'il joue) | K | |
| T7 | « Argyle n'a pas de bornes de déquantification au catalogue » : explication | — | Réponse : le film code les positions des joueurs sur un nombre fixe de bits, relativement à une boîte propre à chaque carte ; sans les bornes de cette boîte (`map_quant_bounds.json`), on ne peut pas reconvertir les positions en mètres, donc pas de rejeu sur la carte. C'était vrai à l'époque ; **Argyle est désormais au catalogue** (vérifié le 19/10), ses rejeux se cuisent. Les 9 films non cuisables de la recuisson sont dans ce cas : des cartes Forge retirées de Waypoint, sans bornes connues. |
