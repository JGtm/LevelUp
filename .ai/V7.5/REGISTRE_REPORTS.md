# Registre des reports et re-explorations — pipeline v7.5 et apres

> Tenu par le superviseur, alimente a chaque cloture de lot (regle utilisateur du
> 2026-08-08). Un item entre ici quand il est REPORTE en connaissance de cause :
> bloquant contourne, feature remise, recherche ajournee. Chaque item porte sa
> condition de reprise — un registre sans condition de reprise est un cimetiere.

| Item | Origine | Pourquoi reporte | Condition de reprise |
|---|---|---|---|
| Recherche KOTH (semantique zones/possession, piste Ghidra) | lot 2, 2026-08-08 | decision user : hors v7.5 ; decodeur direct supprime (lot 3), variante B prouvee exacte une fois mais selection de variante aveugle au contexte | post-release v7.5 ; corpus preserve sous `film/testdata/corpus_objectifs/` (minibobine KOTH `0a247154`, oracle 4-2) |
| Peuplement live de la source evenementielle objectifs | lot 1 `[!]`, 2026-08-07 | chantier pipeline film, pas un correctif ; unique ecrivain actuel = `cmd/diag_weapons_v3 -write` | prealable a TOUT score over-time zone/hill/skull et au branchement complet du containment |
| Containment : elargir `clockOffsets` au-dela de -10 s | lot 4, 2026-08-08 | mesure tronquee — 3 films sur 8 piquent a la borne du balayage ; taux 28,6 % a -5 s vs temoin plat | ~1 h de calcul (corpus lourd : LIMIT + plafond RAM, machine libre) ; DECISION USER en attente |
| Containment : oracle de justesse absent | lot 4, 2026-08-08 | on mesure « une zone rattachee », jamais « la bonne » ; seul releve terrain connu = Vagabond, absente du catalogue | le lot CARTES (Catalyst/Vagabond) fournit l'oracle — le sequencer AVANT de re-statuer le containment |
| Alias Fragmentation Heavies dans `NormalizeMapName` | lot 4, 2026-08-08 | quick win hors perimetre du lot — les 3 zones existent au centimetre sur `fragmentation_map` | 1 ligne + test ; candidat au prochain lot qui touche les cartes |
| Highpower au catalogue sans zones Bastion | lot 4, 2026-08-08 | catalogue incomplet, hors perimetre | completer au lot cartes |
| Litteral `film_chunks`/`film_manifests` en dur dans ~7 CLI | lot 4 (dette anterieure) | hors perimetre du lot | factorisation + garde-rail grep (regle des <= 2 copies) ; lot hygiene de cloture v7.5 |
| `loadGameVariant` en echec renvoie 0, nil sans tenter le chemin historique | lot 1, 2026-08-07 | preexistant, hors perimetre du correctif P0 | lot hygiene de cloture v7.5 |
| `steaktacularMedalIDForTitle` vs repli marge de score : lectures divergentes du slug vide | lot 1, 2026-08-07 | deux interpretations du meme cas, aucune fausse isolement | trancher l'interpretation canonique au lot hygiene de cloture v7.5 |
| Rejeu 2D public (piste F) | decision #2, 2026-08-02 | conditionne par la comprehension du CTF (564 tirs perdus vs 44) | verdict de la voie B (recherche CTF en cours) puis DECISION USER |
| match_player_positions : table inutilisable en l'etat | lot 4, 2026-08-08 | 0 ligne, match-level sans xuid (delta-compression), team a -1 ; la vraie source = pipeline rejeu (BuildFromFilm -> Track par xuid, 100 ms) | si un besoin de positions persistees emerge : repartir du pipeline rejeu, PAS de cette table ; sinon la droper (candidat lot hygiene) |
