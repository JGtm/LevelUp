# VERITE TERRAIN Oddball — d9781168 (Dredge, 27/12/2025), observee par l'utilisateur (JGtm) en Theater, 2026-08-28

Match 2 manches. Roster : Eq0 = joueur_766, JOUEUR_605, joueur_829, JOUEUR_198 ;
Eq1 = joueur_366, joueur_323, JGtm, JOUEUR_838.
Oracle API (time_as_skull_carrier s / grabs) : 116,1/1 · 105/2 · 51,1/0 · 49,1/3 · 37,1/2 ·
25,5/1 · 10,1/0 · 9,7/1 (xuid non apparie ici).

## MANCHE 1 — sequence des porteurs (temps du film, mm:ss)

- 0:48  JOUEUR_605 prend le crane
- ~1:01 SHROOM lache
- 1:05  joueur_829 le ramasse (lache par SHROOM) -> le porte jusqu'a 2:01 (~56 s)
- 2:01  joueur_829 MEURT DANS LE VIDE (frag attribue a JGtm) -> le crane NE tombe PAS sur
        place ; cooldown court ; il RESPAWN SUR SON SOCLE a 2:04
- 2:04  joueur_766 le reprend au socle (quasi immediat)
- 2:07  joueur_766 meurt (mort NORMALE) -> crane roule au sol
- 2:10  JOUEUR_838 le ramasse (~)
- 2:28  JOUEUR_838 fait un KILL AVEC LE CRANE (= kills_as_skull_carrier)
- 2:32  JOUEUR_838 meurt -> lache le crane
- 2:35  joueur_829 ramasse (~)
- 2:37  joueur_829 meurt -> crane tombe au sol
- 2:40  joueur_766 ramasse (~)
- 2:48  joueur_766 MEURT EN SAUTANT DANS LE VIDE avec le crane -> cooldown -> respawn socle 2:53
- 2:53  crane reapparait sur son socle
- 3:00  JOUEUR_838 ramasse
- 3:02  JOUEUR_838 le LANCE puis meurt -> crane au sol
- 3:09  joueur_366 ramasse
- ~3:35 joueur_366 lache
- ~3:39 SHROOM ramasse
- FIN manche 1

## MECANIQUES ETABLIES (causes d'erreur de reconstruction)

1. **Mort NORMALE du porteur -> crane tombe AU SOL SUR PLACE** (reprise locale, ce que la
   reconstruction suppose).
2. **Mort DANS LE VIDE / hors-zone -> crane NE tombe pas sur place : cooldown ~3-5 s puis
   RESPAWN SUR SON SOCLE** (reprise au spawn, PAS a la position de mort). Frequent sur Dredge
   (carte a trous). => la reconstruction proximite/traversee cherche le crane au mauvais
   endroit sur ces morts et attribue la reprise au mauvais joueur. CAUSE NOMMEE.
3. Le crane peut etre LANCE (throw) avant/pendant la mort -> il tombe a distance.

## SIGNAUX DE SCORE PERSONNEL (popups HUD vus par l'utilisateur — piste NATIVE majeure)

- **+10 « crane recupere »** a chaque PRISE (grab) : evenement date par joueur = l'analogue
  de flag_grabs, potentiellement plus complet que le compteur statborg skull_grabs.
- **+50 « controle de balle »** pendant qu'on PORTE : recompense de portage. HYPOTHESE
  utilisateur : si elle s'incremente periodiquement, le joueur qui la recoit EST le porteur a
  cet instant = SIGNAL DIRECT du porteur dans le temps.

## A TESTER (lot Go offline, d9781168 deja decode)

Notre score PERSONNEL par joueur par ms (objectiveevents/score.go) fait-il des SAUTS de +10
(prise) et +50 (controle de balle) qui reproduisent CETTE sequence ? Si oui -> canal natif du
porteur, on publie. Confronter chaque transition ci-dessus aux sauts de score decodes.
