# forza_open_api_client.StoriesApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_stories**](StoriesApi.md#list_stories) | **GET** /v1/stories | Liste les Stories (missions narratives).


# **list_stories**
> StoryList list_stories(game, region=region, page=page, page_size=page_size)

Liste les Stories (missions narratives).

Stories FH6 : missions narratives de Discover Japan, qui rapportent des stamps au Collection Journal. Sources propres (wiki Fandom) ; champs non sourcés → omis.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.story_list import StoryList
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
    api_instance = forza_open_api_client.StoriesApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    region = 'region_example' # str | Filtre par région. (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Liste les Stories (missions narratives).
        api_response = api_instance.list_stories(game, region=region, page=page, page_size=page_size)
        print("The response of StoriesApi->list_stories:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling StoriesApi->list_stories: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **region** | **str**| Filtre par région. | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**StoryList**](StoryList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de Stories. |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

