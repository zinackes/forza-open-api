# forza_open_api_client.ManufacturersApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_manufacturers**](ManufacturersApi.md#list_manufacturers) | **GET** /v1/manufacturers | Liste les constructeurs.


# **list_manufacturers**
> List[Manufacturer] list_manufacturers(game, country=country, q=q)

Liste les constructeurs.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.manufacturer import Manufacturer
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
    api_instance = forza_open_api_client.ManufacturersApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    country = 'country_example' # str | Filtre par pays du constructeur. Correspondance exacte. (optional)
    q = 'q_example' # str | Recherche plein texte sur le nom du constructeur. (optional)

    try:
        # Liste les constructeurs.
        api_response = api_instance.list_manufacturers(game, country=country, q=q)
        print("The response of ManufacturersApi->list_manufacturers:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ManufacturersApi->list_manufacturers: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **country** | **str**| Filtre par pays du constructeur. Correspondance exacte. | [optional] 
 **q** | **str**| Recherche plein texte sur le nom du constructeur. | [optional] 

### Return type

[**List[Manufacturer]**](Manufacturer.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Liste des constructeurs pour le jeu. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

