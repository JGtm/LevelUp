# Mesure B.5 : lecture de 39 fichiers tactiques compresses, avec et sans cache (2026-11-17)

Methode. Banc Go temporaire dans le paquet `internal/service` (retire apres la mesure : aucun usage courant), sur
39 fichiers FABRIQUES par le banc (dix joueurs, 700 cellules et 700 premieres entrees chacun, 14 reapparitions avec
instantane, positions pseudo-aleatoires), ecrits par `replayartifact.Write`. Aucun fichier de `data/`. Un fichier :
808 368 octets bruts, 187 564 compresses (rapport 0,232 ; le parc reel est a 0,137 : la compression fabriquee est
plus defavorable). Poste de developpement partage avec d'autres sessions Go : les chiffres sont indicatifs.
Commande : `go test -v -run '^$' -bench BancTactique -benchtime 20x -count 2 ./internal/service/`.

Resultats (temps pour les 39 fichiers, deux passes de 20 iterations) :

| Lecture | Passe 1 | Passe 2 | Par fichier |
|---|---|---|---|
| Reapparitions, SANS cache (gunzip, decodage maigre, projection) | 313 ms | 331 ms | 8,0 a 8,5 ms |
| Reapparitions, AVEC cache (second passage : 39 `os.Stat`, projection) | 2,5 ms | 3,1 ms | 0,06 a 0,08 ms |
| Tactique, lecture complete (`Charger` : gunzip et decodage entier, sans cache) | 630 ms | 646 ms | 16 ms |
| Octets seuls (`replayartifact.ReadBytes`, gunzip) | 127 ms | 115 ms | 3,0 a 3,3 ms |

Un premier passage du meme banc, pendant une charge plus forte de la machine, donnait 760 ms sans cache et 17 ms
avec cache (meme ordre de rapport). Lecture : le cache ramene les reapparitions d'une carte de plus de 300 ms a
environ 3 ms des le second passage (rapport voisin de 100) ; la Tactique n'a pas de cache par fichier (ce plan n'en
prevoit pas) et paie le decodage entier a chaque lecture.
