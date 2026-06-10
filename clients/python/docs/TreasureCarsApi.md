# forza_open_api_client.TreasureCarsApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_treasure_car**](TreasureCarsApi.md#get_treasure_car) | **GET** /v1/treasure-cars/{id} | Récupère une Treasure Car par identifiant.
[**list_treasure_cars**](TreasureCarsApi.md#list_treasure_cars) | **GET** /v1/treasure-cars | Liste les Treasure Cars (voitures liées aux postcards).


# **get_treasure_car**
> TreasureCar get_treasure_car(id)

Récupère une Treasure Car par identifiant.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.treasure_car import TreasureCar
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
    api_instance = forza_open_api_client.TreasureCarsApi(api_client)
    id = 'id_example' # str | Identifiant stable de la Treasure Car.

    try:
        # Récupère une Treasure Car par identifiant.
        api_response = api_instance.get_treasure_car(id)
        print("The response of TreasureCarsApi->get_treasure_car:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling TreasureCarsApi->get_treasure_car: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable de la Treasure Car. | 

### Return type

[**TreasureCar**](TreasureCar.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | La Treasure Car demandée. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_treasure_cars**
> TreasureCarList list_treasure_cars(game, region=region, car_id=car_id, page=page, page_size=page_size)

Liste les Treasure Cars (voitures liées aux postcards).

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.treasure_car_list import TreasureCarList
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
    api_instance = forza_open_api_client.TreasureCarsApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    region = 'region_example' # str | Filtre par région. (optional)
    car_id = 'car_id_example' # str | Filtre par voiture obtenue (lookup inverse « cette voiture est-elle une Treasure Car ? »). (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Liste les Treasure Cars (voitures liées aux postcards).
        api_response = api_instance.list_treasure_cars(game, region=region, car_id=car_id, page=page, page_size=page_size)
        print("The response of TreasureCarsApi->list_treasure_cars:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling TreasureCarsApi->list_treasure_cars: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **region** | **str**| Filtre par région. | [optional] 
 **car_id** | **str**| Filtre par voiture obtenue (lookup inverse « cette voiture est-elle une Treasure Car ? »). | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**TreasureCarList**](TreasureCarList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de Treasure Cars. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

