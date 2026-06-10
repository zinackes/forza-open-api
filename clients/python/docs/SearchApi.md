# forza_open_api_client.SearchApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**search**](SearchApi.md#search) | **GET** /v1/search | Recherche globale multi-ressources (autocomplete).


# **search**
> SearchResultList search(game, q, kinds=kinds, limit=limit)

Recherche globale multi-ressources (autocomplete).

Recherche plein texte sur les ressources nommées (voitures, tracés, événements, PR stunts, constructeurs, packs DLC) en un seul appel. Pensé pour l'autocomplete d'un site ou d'un bot. Résultats bornés par limit (pas de pagination) ; kinds restreint les types cherchés.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.search_result_kind import SearchResultKind
from forza_open_api_client.models.search_result_list import SearchResultList
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
    api_instance = forza_open_api_client.SearchApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    q = 'q_example' # str | Terme de recherche (sous-chaîne, insensible à la casse).
    kinds = [forza_open_api_client.SearchResultKind()] # List[SearchResultKind] | Types de ressources à chercher (tous si absent). (optional)
    limit = 10 # int | Nombre maximal de résultats. (optional) (default to 10)

    try:
        # Recherche globale multi-ressources (autocomplete).
        api_response = api_instance.search(game, q, kinds=kinds, limit=limit)
        print("The response of SearchApi->search:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SearchApi->search: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **q** | **str**| Terme de recherche (sous-chaîne, insensible à la casse). | 
 **kinds** | [**List[SearchResultKind]**](SearchResultKind.md)| Types de ressources à chercher (tous si absent). | [optional] 
 **limit** | **int**| Nombre maximal de résultats. | [optional] [default to 10]

### Return type

[**SearchResultList**](SearchResultList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les résultats correspondants (groupés par type, nom croissant). |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

