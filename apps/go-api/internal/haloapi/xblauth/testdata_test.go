package xblauth

// Valeurs partagées par les tests de la brique.

const testGamertagInternal = "TestPlayer"

// descripteurHaloTest : le descripteur Halo Infinite tel que `platform/auth` le construit depuis
// `title.DefaultHaloAuthDescriptor()` (parité tenue là-bas : `TestDescripteurXBL_PariteAvecTitre`).
var descripteurHaloTest = Descriptor{
	XSTSAudience:    "https://prod.xsts.halowaypoint.com/",
	SpartanAudience: "urn:343:s3:services",
	SpartanTokenURL: "https://settings.svc.halowaypoint.com/spartan-token",
	ClearanceURL:    "https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active",
}

// clientAzureTest : un client Azure de test (aucun secret réel), autorité « consumers ».
var clientAzureTest = ClientAzure{
	ID:       "client-id-de-test",
	Autorite: "https://login.microsoftonline.com/consumers",
	Scopes:   []string{"Xboxlive.signin", "Xboxlive.offline_access"},
}
