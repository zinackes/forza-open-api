# forza_open_api_client.ForzathonShopApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_forzathon_shop**](ForzathonShopApi.md#get_forzathon_shop) | **GET** /v1/forzathon-shop | Rotation courante du Forzathon Shop d&#39;un jeu.
[**list_forzathon_shop_history**](ForzathonShopApi.md#list_forzathon_shop_history) | **GET** /v1/forzathon-shop/history | Historique des rotations du Forzathon Shop.


# **get_forzathon_shop**
> List[ForzathonShopItem] get_forzathon_shop(game)

Rotation courante du Forzathon Shop d'un jeu.

Les objets de la rotation hebdomadaire en cours (la plus récente connue) du Forzathon Shop, achetables contre des Forza Points. Chaque objet porte sa semaine (weekStart/weekEnd). Tableau vide si aucune rotation connue.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.forzathon_shop_item import ForzathonShopItem
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
    api_instance = forza_open_api_client.ForzathonShopApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).

    try:
        # Rotation courante du Forzathon Shop d'un jeu.
        api_response = api_instance.get_forzathon_shop(game)
        print("The response of ForzathonShopApi->get_forzathon_shop:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ForzathonShopApi->get_forzathon_shop: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 

### Return type

[**List[ForzathonShopItem]**](ForzathonShopItem.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les objets de la rotation courante. |  * Cache-Control - Cache court + stale-while-revalidate (la rotation tourne ~hebdo). Revalidation conditionnelle via ETag / If-None-Match (304). <br>  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_forzathon_shop_history**
> ForzathonShopList list_forzathon_shop_history(game, page=page, page_size=page_size)

Historique des rotations du Forzathon Shop.

Les objets de toutes les rotations connues du Forzathon Shop pour le jeu, les plus récentes d'abord (paginé). Regroupables par weekStart côté client pour reconstituer chaque rotation hebdomadaire.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.forzathon_shop_list import ForzathonShopList
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
    api_instance = forza_open_api_client.ForzathonShopApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Historique des rotations du Forzathon Shop.
        api_response = api_instance.list_forzathon_shop_history(game, page=page, page_size=page_size)
        print("The response of ForzathonShopApi->list_forzathon_shop_history:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ForzathonShopApi->list_forzathon_shop_history: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**ForzathonShopList**](ForzathonShopList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page d&#39;objets de Forzathon Shop (rotations récentes d&#39;abord). |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

