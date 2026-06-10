# forza_open_api_client.MasteryApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_car_mastery**](MasteryApi.md#get_car_mastery) | **GET** /v1/cars/{id}/mastery | Arbre Car Mastery d&#39;une voiture (grille de perks 4×4).


# **get_car_mastery**
> List[CarMasteryPerk] get_car_mastery(id)

Arbre Car Mastery d'une voiture (grille de perks 4×4).

Perks de l'arbre Car Mastery FH6 de la voiture. Chaque perk occupe une case (row, col) de la grille 4×4, coûte des Skill Points (spCost), peut dépendre d'une autre (prereqPerkId) et certaines débloquent une voiture cachée (effectType car_unlock → unlockedCarId). Le jeu est déterminé par la voiture. Voiture inconnue ou arbre non sourcé → liste vide.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car_mastery_perk import CarMasteryPerk
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
    api_instance = forza_open_api_client.MasteryApi(api_client)
    id = 'id_example' # str | Identifiant stable de la voiture.

    try:
        # Arbre Car Mastery d'une voiture (grille de perks 4×4).
        api_response = api_instance.get_car_mastery(id)
        print("The response of MasteryApi->get_car_mastery:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MasteryApi->get_car_mastery: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable de la voiture. | 

### Return type

[**List[CarMasteryPerk]**](CarMasteryPerk.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les perks de l&#39;arbre Car Mastery de la voiture. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

