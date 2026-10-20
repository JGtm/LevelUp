# Note — Jointure geometrique pour attribuer les touches explosives (2026-09-01)

Instrument : `internal/analysis/filmdec/geo_explosifs_research_test.go`
(+ `geo_explosifs_helpers_test.go`, `geo_explosifs_measures_test.go`,
`geo_explosifs_finder_test.go`). Garde `LOT1_TRAME_FILM` (un film), borne 16 chunks,
verrou process, lecture seule. Base bipede DETECTEE par sweep (propre au film). Recherche
d'un film BTB : garde `LOT1_CORPUS`.

## Question

La jointure TEMPORELLE seule (« l'unique tir lourd de la fenetre de vol ») attribue les
touches explosives en arene mais AMBIGUE en BTB (plusieurs projectiles en vol, grande
carte, cf. `NOTE_TOUCHES_EXPLOSIVES`). On a mieux : la VISEE du tireur (i21), les POSITIONS
des deux bipedes, une VITESSE par arme. Le gagnant devait etre le tir qui MINIMISE
(alignement visee->victime) + (ecart de temps de vol), verifie contre la verite terrain du
dead-state (tueur EnumB, 97,6 %).

## Verdict, en un paragraphe

**Le pont geometrique visee+positions est VALIDE et attribue le tir DIRECT au degre pres ;
il n'attribue PAS l'explosif, parce que l'explosif est de l'ECLABOUSSURE (splash) et que la
victime d'un splash n'est PAS sur l'axe de visee du tireur.** L'instrument le mesure
proprement, sur trois cartes dont un BTB, et le chiffre. La verite terrain dead-state n'est
pas evaluable dans `filmdec` (rendement de morts trop faible ; l'extracteur robuste vit
dans le paquet `killsource`). Resultat honnete : la geometrie de visee est un excellent
attributeur du DIRECT et un bon FILTRE anti-coincidence, pas un attributeur de splash.

## M0 — la visee est validee sur trois cartes (le socle)

Sur les degats DIRECTS (`damage_aftermath` a ref1 = tireur BIPEDE, victime ref0 connue),
l'angle entre la visee i21 du tireur et la direction vers sa victime, a l'instant du degat :

| film | carte | n | mediane | < 5 deg | < 15 deg |
|---|---|---|---|---|---|
| 000d5950 (Fiesta) | cliffhanger [13,13,14] | 84 | **2,2 deg** | 79 | 84 |
| 00502e52 | bazaar [17,17,16] | 100 | **3,6 deg** | 59 | 100 |
| 4f77afc1 (BTB) | forge « flood gulch » [15,15,17] | 17 | **2,2 deg** | 16 | 17 |

La convention visee (`AimHeadingDeg`/`AimPitchDeg` = atan2(Y,X) + elevation), la
dequantification des positions, et la base bipede detectee sont donc CORRECTES — y compris
sur une grande carte BTB Forge, avec les bornes du canevas Forge partage (toutes les cartes
`fo##`, signature [15,15,17], partagent l'AABB [462,6 x 453,4 x 1188,5]). C'est le socle
qui rend le reste des mesures fiables : quand l'alignement est grand, ce n'est pas le
decodeur, c'est la physique.

## M2 — l'explosif est du splash : la victime est HORS de l'axe de visee

Meilleur alignement (le tir lourd le mieux aligne de la fenetre) par touche explosive :

| film | touches a candidat | < 5 deg | 5-30 deg | 30-60 deg | >= 60 deg | CONFIRMEES < 15 deg |
|---|---|---|---|---|---|---|
| 000d5950 | 34 | 2 | 0 | 18 | 13 | **2/34 (5,9 %)** |
| 00502e52 | 5 | 0 | 2 | 0 | 0 | **0/5** |
| 4f77afc1 | 0 | — | — | — | — | — |

La coincidence temporelle catch un tir lourd « au hasard » a ~57 deg (angle median entre
deux directions 3D). Le fait que le MEILLEUR alignement d'une touche explosive tombe a
45-90 deg pour ~94 % des touches signe une victime HORS AXE : le tireur visait un point
(direct-impact) et l'onde a touche un tiers a cote. Les 2 seules touches confirmees
(< 5 deg) sur Fiesta sont du **Stalker Rifle**, une arme a FAISCEAU quasi hitscan (donc un
« direct-impact » deguise), pas un projectile en cloche. En clair : sur ces fenetres,
**aucune touche de projectile explosif lobe n'est attribuable par l'alignement de visee**,
parce qu'elle est par nature hors axe.

C'est le prolongement — mesure — de la reserve ecrite d'avance (traqueurs/cloche) : au-dela
du Hydra en verrouillage et du Fuel Rod en cloche, c'est le SPLASH LUI-MEME (toutes armes
explosives confondues) qui defait l'alignement, parce que la victime d'eclaboussure n'est
pas la cible visee.

## M3 — verite terrain dead-state : NON evaluable dans `filmdec`

L'oracle (tueur EnumB du dead-state i11, 97,6 %) exige de recolter les MORTS. La recolte
par trame propre (`DecodeFrameRecords` + `Trace.Dead`) rend **5 morts** sur 16 chunks
d'arene et **0** sur le BTB — le decodeur de trame desynchronise en amont d'i11 sur la
grande majorite des frames de mort (limite connue, cf. `components_object.go` ; c'est
exactement ce que `projectile_owner` observait deja : 5 morts). Consequence : **0 touche
explosive fatale** appariable, donc l'accord geometrie<->dead-state N'EST PAS chiffrable
ici. L'extracteur robuste (93/93) vit dans `internal/games/halo_infinite/film/killsource`
(paquet hors perimetre de ce chantier). La table d'identite roster<->FilmIndex apprise des
morts s'est bien construite et est **injective** (3 roster mappes en arene), preuve que le
pont d'identite fonctionnerait — mais sans morts explosives, il n'a rien a valider.

Statut : `[!]` bloque par une dependance hors paquet (rendement de morts). A reprendre en
branchant `killsource` (ou en portant sa marche robuste dans `filmdec`).

## M4 — gain BTB : le scenario ne se presente pas sur ces films

Le gain geometrique se demontre sur des touches AMBIGUES (>= 2 tireurs lourds dans la
fenetre). Or :

- **Arene 000d5950 (Fiesta)** : 34 touches a candidat, ambiguite **1=34, 2=0, >=3=0** —
  malgre 131 tirs lourds, chaque touche n'a qu'UN tireur lourd distinct dans sa fenetre.
  Zero ambiguite : le temporel-unique suffirait (s'il attribuait juste, ce que M2 refute).
- **BTB 4f77afc1 (Forge, 16 FilmIndex distincts)** : 45 tirs lourds seulement, 19 touches
  non-bipede, et **0 touche coincidant un tir lourd** dans la fenetre — les armes lourdes y
  sont rares et ne recouvrent pas les touches non-bipede. Le film est bien BTB (16 tireurs)
  mais son activite explosive est trop maigre pour exercer la desambiguisation.

Donc la comparaison geometrie vs temporel sur AMBIGU n'a pas d'echantillon. Ce qui EST
mesure : la geometrie REJETTE les coincidences (94 % des appariements temporels sont a
> 30 deg, non alignes). Sa valeur demontree est un FILTRE anti-faux-positif, pas une
desambiguisation (faute d'ambiguite reelle sur ces films).

## Vitesse par arme (M1) : non calibrable ici

La vitesse se calibre sur les touches CONFIRMEES (tir aligne < 15 deg). Il n'y en a que 2
(Stalker, hitscan) : echantillon nul pour un projectile lobe. La vitesse par arme reste
donc NON etablie — non par manque de methode (dist/dt est correct, cf. M0) mais parce que
les touches directes de projectile explosif sont quasi absentes des fenetres examinees.

## Trouver un film BTB exploitable (finder, `LOT1_CORPUS`)

`TestGeoFindBTB` balaye le corpus et retient les films a >= 12 FilmIndex distincts sur une
carte a bornes exploitables. Les cartes BTB « dur » (fragmentation [17,17,15], deadlock
[15,15,15]) ont des signatures AMBIGUES ; les BTB a signature UNIQUE (highpower [18,19,17],
oasis [15,15,14], breaker [13,13,12]) sont rares dans le corpus. Le cas commode est le
**canevas FORGE [15,15,17]** : 59 cartes qui partagent les memes bornes, donc positions
correctes en forcant n'importe laquelle (`LOT1_SONDE_MAP="flood gulch"`). `4f77afc1` en est
un (16 tireurs).

## Reserves honnetes / pistes

1. **Splash = angle-mort de l'aim-join.** Attribuer une touche d'eclaboussure demanderait
   le POINT DE DETONATION (evenements projectile 0xC2 detonate / 0xC3 impact_effect) : la
   victime proche du point de detonation, le point de detonation sur la trajectoire du
   tireur. Le tireur du projectile reste non resolu (ref0 dom5, sens non tranche, cf.
   `lot1_projectiles` / `projectile_owner`). C'est LA piste pour l'explosif ; l'aim-join ne
   la remplace pas.
2. **Sources hors 0xD2.** Une part des touches non-bipede vient de GRENADES (events
   dedies, `grenade_events.go`) et de hasards de carte, pas de `action_weapon_fire` : aucun
   tir lourd 0xD2 ne les couvre, donc aucun candidat. Le denominateur « touches explosives »
   melange plusieurs mecanismes.
3. **Verite terrain bloquee** par le rendement de morts de la trame propre (M3) : rebrancher
   `killsource`.
4. **Direct valide, et c'est reutilisable.** Le pont visee+positions (M0, 2-4 deg sur trois
   cartes dont BTB) de-risque toute attribution par la visee : il marche. Il faut juste
   l'appliquer la ou la victime est SUR l'axe (direct-impact), pas au splash.

## Reproduire

```
export GOCACHE=".../.gocache-geo"
export LOT1_TRAME_FILM=".../data/cache/film_chunks/000d5950"      # arene Fiesta
go test ./internal/analysis/filmdec/ -run TestGeoExplosifs -count=1 -v

export LOT1_TRAME_FILM=".../data/cache/film_chunks/4f77afc1"      # BTB Forge
export LOT1_SONDE_MAP="flood gulch"
go test ./internal/analysis/filmdec/ -run TestGeoExplosifs -count=1 -v

export LOT1_CORPUS=".../data/cache/film_chunks"                   # trouver un BTB
go test ./internal/analysis/filmdec/ -run TestGeoFindBTB -count=1 -v
```
