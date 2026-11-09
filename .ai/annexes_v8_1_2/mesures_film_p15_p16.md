# Mesures du rejeu pour la v8.1.2 (annexe du plan, étape 4), 2026-11-09

Lecture seule, sans cuisson. Faits au schéma 19 : 1 440 films sur 1 447. Liste de frags : copie à froid de la sauvegarde
`data/backups/avant_recuisson_98_2026-10-16/shared_matches_v2.duckdb`, donc du 16/10 : à recapturer comme témoin avant le code.
Sondes conservées dans `sondes/` (`.txt`, à recompiler par `go test -overlay`, jamais à versionner comme tests).

## `5da6fd30` (6 vies sans nom)

Les paquets de type 12 (BOT_METADATA, `grammar.PaquetsBotMetadata`) portent `slot 8 bid 59 "343 Hollis"`, `slot 9 bid 19 "343 Brew Dog"`,
`slot 10 bid 51 "343 The Referee"`, équipe 0. Le tableau de l'API porte `bid(59.0)`, `bid(19.0)`, `bid(51.0)`. Les créations (type 35) donnent les slots
520, 521, 522 aux index 8, 9, 10. Le film n'a aucun chunk HIGHLIGHT (match à 0 frag) : `killsource.Decode` rend `ErrNoKillFeed`, `replaybuild/kills.go decodeKillSource`
renvoie `nil`, `BotIdentities(nil)` donne `Bots = nil`. Injecter ces trois bots dans `BuildIdentityRegistry` nomme les 6 vies par la règle `tableau_api`, `sansIdentite=0`.
Parc : 1 seul film a `Kills == nil` dans ses faits. La phrase du backlog « bots que BOT_METADATA ne déclare jamais » est fausse.

## `7aa132d1` (2 vies, slots 36 et 292)

379 créations de bipèdes, toutes dans les slots 512..767. Slots 36 et 292 : 4 positions chacun, aucune création, aucune position de véhicule, un objet qui parcourt ~20 m en 1,6 s.
Parc (1 440 films) : 0 création hors [512, 768) ; positions sans création dans 2 films seulement, `7aa132d1` (36, 292, 361) et `a12f267b` (325, déjà écartée en aval).

## `72c48587` (7 vies)

Ni artefact, ni fait, ni chunk, ni manifeste sur le poste ; le match existe dans la base.

## Châssis `b857fb95`

455 vies, 117 films, 453 sans échantillon de position, 238 nées à t=0, renaissance aux mêmes points fixes, 265 avec occupant. Exemple `0862dce4` : spawns répétés à (-132,-61,47) et (-120,-60,44) ;
`00ba2e1c` : deux spawns miroir (-51, 0.3, 101) et (51, -0.3, 101). `libs/film/damagetag/data/labels.tsv` classe `vehi b857fb95` avec les armes `sb_010_tur_un_{gausscannon,machinegun,rocketturret}`, classe `turret`.
Profil identique à `3a8060e2` (`tourelle_fixe`, validé à l'œil par le user le 2026-09-24). Parent sans échantillon de 213 occupations.

## Second corps de `4f77afc1`

`TestLieuUniqueDesMorts` (`-tags research`, depuis `libs/film`, `L1_FAITS=<dossier des faits> L1_BORNES=<map_quant_bounds.json>`) :
- registre de cuisson (défaut) : `300 morts ; localisees avant=238 apres=285 ; PERDUES=0` ;
- `REG_COLLECTEUR=1` : `PERTE 4f77afc1 : 1/300 (ambigu=1)`, sièges 547 et 565 à 83 m.

Cause : dans le registre du collecteur (sans table du film, sans tirs, sans horloge), la seconde vie du siège 565 n'est pas nommée ; `identity_registry_pont.go OwnerReport.xuidNumAt` ignore les vies couvrantes non nommées et retombe sur `SlotXUID`.

Parc, `REG_COLLECTEUR=1`, 143 882 morts : PERDUES = 52, dont 24 ambiguïté de corps (14 films, 11 sur `3abf0258`), 12 à bord sans véhicule, 16 débarqué. GAGNÉES = 3 145.

## Entame (`kill_openings`)

88 films, 8 144 kills, 16 288 côtés, 1 049 sans position (626 tueurs, 423 victimes). Voies : échantillon hors vie 245 (refus correct), tenue 266 (168 même vie, 98 autre vie), siège 415, à bord sans véhicule 78, muette 45. Gain plafonné à 168 à 583 côtés.

## Parent non véhicule

`5676a9ba`, parent 806 : création au châssis `dd7f9102` (tourelle montée de Warthog, déjà dans `vehicle_turrets.go`), 0 position, 3 occupants au siège 0.
Parc : 6 843 parents d'occupation, 1 714 sans échantillon : `dd7f9102` 468, sans création 320, `bcfb852f` 241, `b857fb95` 213, `001b33fc` 115, `3a8060e2` 102, `0000d500` 72, `1a043c29` 42, `000df0c4` 40, `f4c45d71` 31.
Explique 28 des 52 pertes (12 à bord sans véhicule, 16 débarqué).
