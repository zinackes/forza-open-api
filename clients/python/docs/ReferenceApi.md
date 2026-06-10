# forza_open_api_client.ReferenceApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_reference**](ReferenceApi.md#get_reference) | **GET** /v1/reference | Énumérations et compteurs (facettes) pour les filtres clients.


# **get_reference**
> Reference get_reference(game)

Énumérations et compteurs (facettes) pour les filtres clients.

Facettes agrégées pour construire les filtres d'un client en un seul appel : classes PI (incluant R en FH6), transmissions, types de carrosserie, pays des constructeurs et catégories (divisions in-game) — comptées pour le `game` demandé. La liste `games` est globale (volumes par jeu, indépendante du paramètre game) pour amorcer un sélecteur de jeu. Réponse fortement cacheable, invalidée par les jobs d'ingestion.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.reference import Reference
from forza_open_api_client.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://api.forza-open-api.org
# See configuration.py for a list of all supported configuration parameters.
configuration = forza_open_api_client.Configuration(
    host = "https://api.forza-open-api.org"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Enter a context with an instance of the API client
with forza_open_api_client.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = forza_open_api_client.ReferenceApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).

    try:
        # Énumérations et compteurs (facettes) pour les filtres clients.
        api_response = api_instance.get_reference(game)
        print("The response of ReferenceApi->get_reference:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ReferenceApi->get_reference: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 

### Return type

[**Reference**](Reference.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Facettes de référence pour le jeu. |  * Cache-Control - Cache court + stale-while-revalidate (la playlist tourne ~hebdo). Revalidation conditionnelle via ETag / If-None-Match (304). <br>  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

