# forza_open_api_client.UpgradesApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_car_upgrades**](UpgradesApi.md#list_car_upgrades) | **GET** /v1/cars/{id}/upgrades | Liste les upgrades disponibles pour une voiture.
[**list_upgrade_parts**](UpgradesApi.md#list_upgrade_parts) | **GET** /v1/upgrade-parts | Catalogue global des pièces d&#39;upgrade.


# **list_car_upgrades**
> CarUpgradeList list_car_upgrades(id, category=category, page=page, page_size=page_size)

Liste les upgrades disponibles pour une voiture.

Pièces d'upgrade montables sur la voiture, avec leurs contraintes d'installation (prérequis, groupe exclusif). Le jeu est déterminé par la voiture (pas de paramètre game). Voiture inconnue ou sans upgrade sourcé → page vide.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car_upgrade_list import CarUpgradeList
from forza_open_api_client.models.upgrade_part_category import UpgradePartCategory
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
    api_instance = forza_open_api_client.UpgradesApi(api_client)
    id = 'id_example' # str | Identifiant stable de la voiture.
    category = forza_open_api_client.UpgradePartCategory() # UpgradePartCategory | Filtre par catégorie de pièce. (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Liste les upgrades disponibles pour une voiture.
        api_response = api_instance.list_car_upgrades(id, category=category, page=page, page_size=page_size)
        print("The response of UpgradesApi->list_car_upgrades:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling UpgradesApi->list_car_upgrades: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable de la voiture. | 
 **category** | [**UpgradePartCategory**](.md)| Filtre par catégorie de pièce. | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**CarUpgradeList**](CarUpgradeList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page d&#39;upgrades disponibles pour la voiture. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_upgrade_parts**
> UpgradePartList list_upgrade_parts(game, category=category, page=page, page_size=page_size)

Catalogue global des pièces d'upgrade.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.upgrade_part_category import UpgradePartCategory
from forza_open_api_client.models.upgrade_part_list import UpgradePartList
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
    api_instance = forza_open_api_client.UpgradesApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    category = forza_open_api_client.UpgradePartCategory() # UpgradePartCategory | Filtre par catégorie de pièce. (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Catalogue global des pièces d'upgrade.
        api_response = api_instance.list_upgrade_parts(game, category=category, page=page, page_size=page_size)
        print("The response of UpgradesApi->list_upgrade_parts:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling UpgradesApi->list_upgrade_parts: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **category** | [**UpgradePartCategory**](.md)| Filtre par catégorie de pièce. | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**UpgradePartList**](UpgradePartList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de pièces d&#39;upgrade. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

