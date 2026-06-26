# Halo 5 — DeathDisposition / KillerAgent / VictimAgent décodés

Sonde 2026-06-26 : 20 matchs Arena JGtm, **1909 morts** agrégées.

## DeathDisposition (dans les events Death)
| Valeur | n | Sens | Signature |
|---|---|---|---|
| `1` | 1860 | **Kill ennemi** (normal) | Killer & Victim = joueurs distincts |
| `0` | 30 | **Suicide** | `Killer.Gamertag == Victim.Gamertag` (même position monde, `IsWeapon:false`) |
| `2` | 19 | **Kill d'une entité non-joueur** | `Victim == null`, `VictimAgent == 0` (objet/IA/déployable) |

## KillerAgent / VictimAgent
- `1` = **joueur (Spartan)**. `0` = **non-joueur** (IA / objet / déployable).
- Sur Arena : KillerAgent toujours 1 ; VictimAgent = 1 (1890) ou 0 (19, cf. disposition 2).
- En **Warzone / Campagne** (PvE, non re-sondés ici car exclus du sync), KillerAgent
  prendrait 0 (une IA te tue) → ces champs sont LE discriminant joueur-vs-IA quand on
  ouvrira le PvE.

## Trahison (betrayal)
**PAS de code DeathDisposition dédié** dans l'échantillon (aucune trahison sur 20 matchs
Arena). → à inférer par **killer & victim de la MÊME équipe** (comparer `TeamId` via le
roster carnage / participants), pas via un code. Non confirmé sur données réelles à ce jour.

## Exploitation
- **Suicides** (disp 0) : fiables, ~1.6 % des morts ici → donut « discipline » + à
  dé-pondérer dans le rythme d'engagement (un suicide n'est pas de l'activité offensive).
- **Disposition 2 / Agent 0** : à exclure des stats joueur-vs-joueur (kills d'objets).
- **Trahisons** : nécessitent la jointure d'équipe ; prévoir le calcul, pas un champ direct.
