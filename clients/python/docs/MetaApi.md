# forza_open_api_client.MetaApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_meta**](MetaApi.md#get_meta) | **GET** /v1/meta | Métadonnées du service : jeux supportés, volumes et fraîcheur des données.


# **get_meta**
> Meta get_meta()

Métadonnées du service : jeux supportés, volumes et fraîcheur des données.

Métadonnées légères pour les consommateurs et le dogfooding : jeux supportés (fh6, fh5), nombre de voitures par jeu et derniers timestamps d'ingestion du catalogue et de la playlist. Les timestamps proviennent du journal d'ingestion (data_changes) : `catalogUpdatedAt` (ressource car) et `playlistUpdatedAt` (ressource series) sont absents si la ressource n'a jamais été ingérée pour le jeu (rien d'inventé). `dataVersion` porte la version de jeu de données publiée si l'opérateur l'a tamponnée. `generatedAt` = instant de calcul, pour estimer l'âge côté client. Réponse à cache court : la fraîcheur est l'objet même de l'endpoint.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.meta import Meta
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
    api_instance = forza_open_api_client.MetaApi(api_client)

    try:
        # Métadonnées du service : jeux supportés, volumes et fraîcheur des données.
        api_response = api_instance.get_meta()
        print("The response of MetaApi->get_meta:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MetaApi->get_meta: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**Meta**](Meta.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Métadonnées du service. |  * Cache-Control - Cache court + stale-while-revalidate (la playlist tourne ~hebdo). Revalidation conditionnelle via ETag / If-None-Match (304). <br>  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

