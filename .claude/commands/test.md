Lance la suite de tests du projet LevelUp et résume les résultats.

Exécute cette commande :
```
.venv/Scripts/python.exe -m pytest -q --ignore=tests/integration
```

Puis :
- Si tous les tests passent : confirme brièvement avec le nombre de tests et le temps d'exécution
- Si des tests échouent : liste chaque test échoué avec le message d'erreur, et indique si l'échec est pré-existant (connu) ou potentiellement lié aux modifications récentes
- Ne propose pas de corrections sauf si l'utilisateur le demande explicitement
