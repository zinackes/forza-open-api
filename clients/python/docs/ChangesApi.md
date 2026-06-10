# forza_open_api_client.ChangesApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_changes**](ChangesApi.md#list_changes) | **GET** /v1/changes | Journal des changements de données.


# **list_changes**
> ChangeList list_changes(game, resource=resource, action=action, since=since, page=page, page_size=page_size)

Journal des changements de données.

Flux des changements du jeu de données (voiture ajoutée/modifiée, pack sorti, série publiée…), alimenté par les jobs d'ingestion. Donne aux clients un « what's new » et la base d'une synchro incrémentale. Plus récents d'abord.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.change_action import ChangeAction
from forza_open_api_client.models.change_list import ChangeList
from forza_open_api_client.models.game import Game
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
    api_instance = forza_open_api_client.ChangesApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    resource = 'resource_example' # str | Filtre par type de ressource (car, dlc_pack, series…). Valeur libre, non figée au contrat. (optional)
    action = forza_open_api_client.ChangeAction() # ChangeAction | Filtre par action. (optional)
    since = '2013-10-20T19:20:30+01:00' # datetime | Ne renvoie que les changements postérieurs à cet instant. (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Journal des changements de données.
        api_response = api_instance.list_changes(game, resource=resource, action=action, since=since, page=page, page_size=page_size)
        print("The response of ChangesApi->list_changes:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ChangesApi->list_changes: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **resource** | **str**| Filtre par type de ressource (car, dlc_pack, series…). Valeur libre, non figée au contrat. | [optional] 
 **action** | [**ChangeAction**](.md)| Filtre par action. | [optional] 
 **since** | **datetime**| Ne renvoie que les changements postérieurs à cet instant. | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**ChangeList**](ChangeList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de changements (plus récents d&#39;abord). |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

