# forza_open_api_client.PRStuntsApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_pr_stunt**](PRStuntsApi.md#get_pr_stunt) | **GET** /v1/pr-stunts/{id} | Récupère un PR Stunt par identifiant.
[**list_pr_stunts**](PRStuntsApi.md#list_pr_stunts) | **GET** /v1/pr-stunts | Liste les PR Stunts (speed trap, speed zone, drift zone, danger sign).


# **get_pr_stunt**
> PRStunt get_pr_stunt(id)

Récupère un PR Stunt par identifiant.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.pr_stunt import PRStunt
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
    api_instance = forza_open_api_client.PRStuntsApi(api_client)
    id = 'id_example' # str | Identifiant stable du PR Stunt.

    try:
        # Récupère un PR Stunt par identifiant.
        api_response = api_instance.get_pr_stunt(id)
        print("The response of PRStuntsApi->get_pr_stunt:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling PRStuntsApi->get_pr_stunt: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable du PR Stunt. | 

### Return type

[**PRStunt**](PRStunt.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Le PR Stunt demandé. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_pr_stunts**
> PRStuntList list_pr_stunts(game, type=type, region=region, q=q, page=page, page_size=page_size)

Liste les PR Stunts (speed trap, speed zone, drift zone, danger sign).

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.pr_stunt_list import PRStuntList
from forza_open_api_client.models.pr_stunt_type import PRStuntType
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
    api_instance = forza_open_api_client.PRStuntsApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    type = forza_open_api_client.PRStuntType() # PRStuntType | Filtre par type de PR Stunt. (optional)
    region = 'region_example' # str | Filtre par région. (optional)
    q = 'q_example' # str | Recherche plein texte sur le nom du PR Stunt. (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Liste les PR Stunts (speed trap, speed zone, drift zone, danger sign).
        api_response = api_instance.list_pr_stunts(game, type=type, region=region, q=q, page=page, page_size=page_size)
        print("The response of PRStuntsApi->list_pr_stunts:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling PRStuntsApi->list_pr_stunts: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **type** | [**PRStuntType**](.md)| Filtre par type de PR Stunt. | [optional] 
 **region** | **str**| Filtre par région. | [optional] 
 **q** | **str**| Recherche plein texte sur le nom du PR Stunt. | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**PRStuntList**](PRStuntList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de PR Stunts. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

