# forza_open_api_client.JournalApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_journal_tiers**](JournalApi.md#list_journal_tiers) | **GET** /v1/journal | Liste les paliers du Collection Journal (wristbands + stamps).


# **list_journal_tiers**
> List[JournalTier] list_journal_tiers(game, track=track)

Liste les paliers du Collection Journal (wristbands + stamps).

Paliers de progression du Collection Journal FH6 : 7 Wristbands (track horizon_festival, Yellow → Gold ; Gold débloque Legend Island + The Goliath) et 7 Stamps (track discover_japan, Visitor → Master Explorer ; poussent les Barn Finds). 17 voitures ne sont débloquables que via les rewardCarId de ces paliers. Remplace les Accolades de FH5. Ensemble borné (≤ 14 par jeu) → pas de pagination.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.game import Game
from forza_open_api_client.models.journal_tier import JournalTier
from forza_open_api_client.models.journal_track import JournalTrack
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
    api_instance = forza_open_api_client.JournalApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    track = forza_open_api_client.JournalTrack() # JournalTrack | Filtre par piste de progression. (optional)

    try:
        # Liste les paliers du Collection Journal (wristbands + stamps).
        api_response = api_instance.list_journal_tiers(game, track=track)
        print("The response of JournalApi->list_journal_tiers:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling JournalApi->list_journal_tiers: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **track** | [**JournalTrack**](.md)| Filtre par piste de progression. | [optional] 

### Return type

[**List[JournalTier]**](JournalTier.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les paliers du Collection Journal du jeu (par piste, puis niveau croissant). |  -  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

