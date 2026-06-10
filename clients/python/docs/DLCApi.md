# forza_open_api_client.DLCApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_dlc_pack**](DLCApi.md#get_dlc_pack) | **GET** /v1/dlc-packs/{id} | Récupère un pack DLC par identifiant.
[**list_dlc_packs**](DLCApi.md#list_dlc_packs) | **GET** /v1/dlc-packs | Liste les packs DLC / extensions d&#39;un jeu.


# **get_dlc_pack**
> DlcPack get_dlc_pack(id)

Récupère un pack DLC par identifiant.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.dlc_pack import DlcPack
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
    api_instance = forza_open_api_client.DLCApi(api_client)
    id = 'id_example' # str | Identifiant stable du pack.

    try:
        # Récupère un pack DLC par identifiant.
        api_response = api_instance.get_dlc_pack(id)
        print("The response of DLCApi->get_dlc_pack:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DLCApi->get_dlc_pack: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable du pack. | 

### Return type

[**DlcPack**](DlcPack.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Le pack demandé. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_dlc_packs**
> List[DlcPack] list_dlc_packs(game, kind=kind)

Liste les packs DLC / extensions d'un jeu.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.dlc_kind import DlcKind
from forza_open_api_client.models.dlc_pack import DlcPack
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
    api_instance = forza_open_api_client.DLCApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    kind = forza_open_api_client.DlcKind() # DlcKind | Filtre par type de pack. (optional)

    try:
        # Liste les packs DLC / extensions d'un jeu.
        api_response = api_instance.list_dlc_packs(game, kind=kind)
        print("The response of DLCApi->list_dlc_packs:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DLCApi->list_dlc_packs: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **kind** | [**DlcKind**](.md)| Filtre par type de pack. | [optional] 

### Return type

[**List[DlcPack]**](DlcPack.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les packs DLC connus pour le jeu (plus récents d&#39;abord). |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

