# forza_open_api_client.TracksApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_random_track**](TracksApi.md#get_random_track) | **GET** /v1/tracks/random | Renvoie un tracé aléatoire satisfaisant les filtres.
[**get_track**](TracksApi.md#get_track) | **GET** /v1/tracks/{id} | Récupère un tracé par identifiant.
[**list_tracks**](TracksApi.md#list_tracks) | **GET** /v1/tracks | Liste les tracés / circuits indexés.


# **get_random_track**
> Track get_random_track(game, type=type, region=region)

Renvoie un tracé aléatoire satisfaisant les filtres.

Tire un seul tracé au hasard parmi ceux qui satisfont les filtres (mêmes filtres optionnels que /v1/tracks, hors q et pagination). Pensé pour les bots Discord et défis communautaires (« course aléatoire du jour »). Réponse non cacheable (Cache-Control: no-store) : chaque appel re-tire. 404 si aucun tracé ne correspond.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.track import Track
from forza_open_api_client.models.track_type import TrackType
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
    api_instance = forza_open_api_client.TracksApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    type = forza_open_api_client.TrackType() # TrackType | Filtre par catégorie de tracé. (optional)
    region = 'region_example' # str | Filtre par région. (optional)

    try:
        # Renvoie un tracé aléatoire satisfaisant les filtres.
        api_response = api_instance.get_random_track(game, type=type, region=region)
        print("The response of TracksApi->get_random_track:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling TracksApi->get_random_track: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **type** | [**TrackType**](.md)| Filtre par catégorie de tracé. | [optional] 
 **region** | **str**| Filtre par région. | [optional] 

### Return type

[**Track**](Track.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Un tracé tiré au hasard parmi les correspondances. |  * Cache-Control - Toujours no-store (tirage aléatoire, jamais mis en cache). <br>  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_track**
> Track get_track(id)

Récupère un tracé par identifiant.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.track import Track
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
    api_instance = forza_open_api_client.TracksApi(api_client)
    id = 'id_example' # str | Identifiant stable du tracé.

    try:
        # Récupère un tracé par identifiant.
        api_response = api_instance.get_track(id)
        print("The response of TracksApi->get_track:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling TracksApi->get_track: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable du tracé. | 

### Return type

[**Track**](Track.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Le tracé demandé. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_tracks**
> TrackList list_tracks(game, type=type, region=region, q=q, updated_since=updated_since, page=page, page_size=page_size)

Liste les tracés / circuits indexés.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.track_list import TrackList
from forza_open_api_client.models.track_type import TrackType
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
    api_instance = forza_open_api_client.TracksApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    type = forza_open_api_client.TrackType() # TrackType | Filtre par catégorie de tracé. (optional)
    region = 'region_example' # str | Filtre par région. (optional)
    q = 'q_example' # str | Recherche plein texte sur le nom du tracé. (optional)
    updated_since = '2013-10-20T19:20:30+01:00' # datetime | Ne renvoie que les éléments modifiés après cet instant (synchro incrémentale : un client ne re-télécharge que le delta).  (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Liste les tracés / circuits indexés.
        api_response = api_instance.list_tracks(game, type=type, region=region, q=q, updated_since=updated_since, page=page, page_size=page_size)
        print("The response of TracksApi->list_tracks:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling TracksApi->list_tracks: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **type** | [**TrackType**](.md)| Filtre par catégorie de tracé. | [optional] 
 **region** | **str**| Filtre par région. | [optional] 
 **q** | **str**| Recherche plein texte sur le nom du tracé. | [optional] 
 **updated_since** | **datetime**| Ne renvoie que les éléments modifiés après cet instant (synchro incrémentale : un client ne re-télécharge que le delta).  | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**TrackList**](TrackList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de tracés. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

