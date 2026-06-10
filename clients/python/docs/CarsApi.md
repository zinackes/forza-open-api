# forza_open_api_client.CarsApi

All URIs are relative to *https://api.forza-open-api.org*

Method | HTTP request | Description
------------- | ------------- | -------------
[**compare_cars**](CarsApi.md#compare_cars) | **GET** /v1/cars/compare | Compare 2 à 3 voitures côte à côte.
[**get_car**](CarsApi.md#get_car) | **GET** /v1/cars/{id} | Récupère une voiture par identifiant.
[**get_car_obtain**](CarsApi.md#get_car_obtain) | **GET** /v1/cars/{id}/obtain | Vue agrégée « comment obtenir cette voiture ».
[**get_random_car**](CarsApi.md#get_random_car) | **GET** /v1/cars/random | Renvoie une voiture aléatoire satisfaisant les filtres.
[**list_cars**](CarsApi.md#list_cars) | **GET** /v1/cars | Liste les voitures du catalogue.


# **compare_cars**
> CarComparison compare_cars(ids)

Compare 2 à 3 voitures côte à côte.

Renvoie les voitures demandées (2 à 3, via `ids`) dans l'ordre de la requête, pour un affichage côte à côte (overlays, bots Discord) : PI, classe, transmission et stats (vitesse/accélération/handling/freinage) sont alignés. Chaque entrée est l'objet Car complet. Comparaison stricte : 400 si moins de 2 ou plus de 3 ids ; 404 si un id est inconnu (contrairement au filtre `ids` de /v1/cars, aucun id n'est ignoré). L'id étant la clé stable globale, pas de paramètre `game`.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car_comparison import CarComparison
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
    api_instance = forza_open_api_client.CarsApi(api_client)
    ids = ['ids_example'] # List[str] | Identifiants des voitures à comparer (CSV, ex. ids=fh6-mazda-rx7-1997,fh6-toyota-supra-1998), 2 à 3 valeurs. L'ordre est conservé dans la réponse.

    try:
        # Compare 2 à 3 voitures côte à côte.
        api_response = api_instance.compare_cars(ids)
        print("The response of CarsApi->compare_cars:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CarsApi->compare_cars: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ids** | [**List[str]**](str.md)| Identifiants des voitures à comparer (CSV, ex. ids&#x3D;fh6-mazda-rx7-1997,fh6-toyota-supra-1998), 2 à 3 valeurs. L&#39;ordre est conservé dans la réponse. | 

### Return type

[**CarComparison**](CarComparison.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les voitures demandées, alignées dans l&#39;ordre des ids. |  * Cache-Control - Toujours no-store (tirage aléatoire, jamais mis en cache). <br>  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_car**
> Car get_car(id)

Récupère une voiture par identifiant.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car import Car
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
    api_instance = forza_open_api_client.CarsApi(api_client)
    id = 'id_example' # str | Identifiant stable de la voiture.

    try:
        # Récupère une voiture par identifiant.
        api_response = api_instance.get_car(id)
        print("The response of CarsApi->get_car:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CarsApi->get_car: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable de la voiture. | 

### Return type

[**Car**](Car.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | La voiture demandée. |  * Cache-Control - Toujours no-store (tirage aléatoire, jamais mis en cache). <br>  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_car_obtain**
> CarObtain get_car_obtain(id)

Vue agrégée « comment obtenir cette voiture ».

Agrège toutes les voies d'obtention connues d'une voiture : méthode du catalogue (autoshow, wheelspin…) + prix, packs DLC qui la contiennent, Barn Find, Treasure Car, paliers du Collection Journal qui la récompensent et perks Car Mastery (car_unlock) qui la débloquent. Sources absentes → listes vides / champs omis (rien d'inventé). 404 si la voiture est inconnue.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car_obtain import CarObtain
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
    api_instance = forza_open_api_client.CarsApi(api_client)
    id = 'id_example' # str | Identifiant stable de la voiture.

    try:
        # Vue agrégée « comment obtenir cette voiture ».
        api_response = api_instance.get_car_obtain(id)
        print("The response of CarsApi->get_car_obtain:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CarsApi->get_car_obtain: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Identifiant stable de la voiture. | 

### Return type

[**CarObtain**](CarObtain.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Les voies d&#39;obtention de la voiture. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_random_car**
> Car get_random_car(game, make=make, var_class=var_class, pi_min=pi_min, pi_max=pi_max, drivetrain=drivetrain, category=category)

Renvoie une voiture aléatoire satisfaisant les filtres.

Tire une seule voiture au hasard parmi celles qui satisfont les filtres (mêmes filtres optionnels que /v1/cars, hors q/dlc/ids/obtain/ updated_since et pagination). Pensé pour les bots Discord ("bagnole random du jour"), défis communautaires et easter-eggs sur la landing. Réponse non cacheable (Cache-Control: no-store) : chaque appel re-tire. 404 si aucune voiture ne correspond.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car import Car
from forza_open_api_client.models.car_class import CarClass
from forza_open_api_client.models.drivetrain import Drivetrain
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
    api_instance = forza_open_api_client.CarsApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    make = 'make_example' # str | Filtre par constructeur (ex. \"Ford\"). (optional)
    var_class = forza_open_api_client.CarClass() # CarClass | Filtre par classe PI. (optional)
    pi_min = 56 # int | Performance Index minimum (inclus). (optional)
    pi_max = 56 # int | Performance Index maximum (inclus). (optional)
    drivetrain = forza_open_api_client.Drivetrain() # Drivetrain | Filtre par transmission. (optional)
    category = 'category_example' # str | Filtre par catégorie / division in-game (ex. \"Modern Supercars\"). Correspondance exacte. (optional)

    try:
        # Renvoie une voiture aléatoire satisfaisant les filtres.
        api_response = api_instance.get_random_car(game, make=make, var_class=var_class, pi_min=pi_min, pi_max=pi_max, drivetrain=drivetrain, category=category)
        print("The response of CarsApi->get_random_car:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CarsApi->get_random_car: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **make** | **str**| Filtre par constructeur (ex. \&quot;Ford\&quot;). | [optional] 
 **var_class** | [**CarClass**](.md)| Filtre par classe PI. | [optional] 
 **pi_min** | **int**| Performance Index minimum (inclus). | [optional] 
 **pi_max** | **int**| Performance Index maximum (inclus). | [optional] 
 **drivetrain** | [**Drivetrain**](.md)| Filtre par transmission. | [optional] 
 **category** | **str**| Filtre par catégorie / division in-game (ex. \&quot;Modern Supercars\&quot;). Correspondance exacte. | [optional] 

### Return type

[**Car**](Car.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Une voiture tirée au hasard parmi les correspondances. |  * Cache-Control - Cache court + stale-while-revalidate (la rotation tourne ~hebdo). Revalidation conditionnelle via ETag / If-None-Match (304). <br>  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**404** | Ressource introuvable. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_cars**
> CarList list_cars(game, ids=ids, make=make, var_class=var_class, pi_min=pi_min, pi_max=pi_max, drivetrain=drivetrain, category=category, q=q, dlc=dlc, obtain=obtain, updated_since=updated_since, sort=sort, page=page, page_size=page_size)

Liste les voitures du catalogue.

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import forza_open_api_client
from forza_open_api_client.models.car_class import CarClass
from forza_open_api_client.models.car_list import CarList
from forza_open_api_client.models.drivetrain import Drivetrain
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
    api_instance = forza_open_api_client.CarsApi(api_client)
    game = forza_open_api_client.Game() # Game | Jeu cible (obligatoire sur les ressources multi-jeux).
    ids = ['ids_example'] # List[str] | Restreint le résultat à une liste d'identifiants (CSV, ex. ids=audi-r8,ford-gt), bornée à 100. Les ids inconnus ou d'un autre jeu sont ignorés (pas d'erreur) ; total et pagination portent sur les correspondances. (optional)
    make = 'make_example' # str | Filtre par constructeur (ex. \"Ford\"). (optional)
    var_class = forza_open_api_client.CarClass() # CarClass | Filtre par classe PI. (optional)
    pi_min = 56 # int | Performance Index minimum (inclus). (optional)
    pi_max = 56 # int | Performance Index maximum (inclus). (optional)
    drivetrain = forza_open_api_client.Drivetrain() # Drivetrain | Filtre par transmission. (optional)
    category = 'category_example' # str | Filtre par catégorie / division in-game (ex. \"Modern Supercars\"). Correspondance exacte. (optional)
    q = 'q_example' # str | Recherche plein texte sur name/model. (optional)
    dlc = 'dlc_example' # str | Filtre par pack DLC (identifiant d'un dlc_packs) ; liste les voitures du pack. (optional)
    obtain = 'obtain_example' # str | Filtre par méthode d'obtention. obtain_method est multi-valeurs (ex. \"Autoshow, Wheelspin\") : la correspondance se fait par token — obtain=wheelspin renvoie toute voiture dont l'une des méthodes est Wheelspin. Correspondances : autoshow → Autoshow ; wheelspin → Wheelspin ; wristband → Wristband reward / Yellow Wristband ; barn_find → Barn Find ; treasure → Treasure Car ; car_mastery → Car Mastery ; journal → Collection Journal ; car_pass → Car Pass ; hard_to_find → Hard to Find ; aftermarket → Aftermarket Car ; prologue → Complete the Prologue ; loyalty → Loyalty Reward ; preorder → Pre-order ; promotional → Promotional ; vip → VIP Membership ; welcome_pack → Welcome Pack ; unobtainable → Unobtainable. Les voitures de packs DLC se filtrent via `dlc`. Compteurs par valeur : facette `obtainMethods` de /v1/reference. (optional)
    updated_since = '2013-10-20T19:20:30+01:00' # datetime | Ne renvoie que les éléments modifiés après cet instant (synchro incrémentale : un client ne re-télécharge que le delta).  (optional)
    sort = 'sort_example' # str | Tri du résultat : pi, name, year ou value (valeur en crédits). Préfixe \"-\" pour l'ordre décroissant (ex. sort=-pi). Défaut : pi croissant. Tri stable (départage par name puis id) ; les valeurs absentes (year/value NULL) sont renvoyées en dernier. (optional)
    page = 1 # int | Numéro de page (1-based). (optional) (default to 1)
    page_size = 50 # int | Taille de page. (optional) (default to 50)

    try:
        # Liste les voitures du catalogue.
        api_response = api_instance.list_cars(game, ids=ids, make=make, var_class=var_class, pi_min=pi_min, pi_max=pi_max, drivetrain=drivetrain, category=category, q=q, dlc=dlc, obtain=obtain, updated_since=updated_since, sort=sort, page=page, page_size=page_size)
        print("The response of CarsApi->list_cars:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling CarsApi->list_cars: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **game** | [**Game**](.md)| Jeu cible (obligatoire sur les ressources multi-jeux). | 
 **ids** | [**List[str]**](str.md)| Restreint le résultat à une liste d&#39;identifiants (CSV, ex. ids&#x3D;audi-r8,ford-gt), bornée à 100. Les ids inconnus ou d&#39;un autre jeu sont ignorés (pas d&#39;erreur) ; total et pagination portent sur les correspondances. | [optional] 
 **make** | **str**| Filtre par constructeur (ex. \&quot;Ford\&quot;). | [optional] 
 **var_class** | [**CarClass**](.md)| Filtre par classe PI. | [optional] 
 **pi_min** | **int**| Performance Index minimum (inclus). | [optional] 
 **pi_max** | **int**| Performance Index maximum (inclus). | [optional] 
 **drivetrain** | [**Drivetrain**](.md)| Filtre par transmission. | [optional] 
 **category** | **str**| Filtre par catégorie / division in-game (ex. \&quot;Modern Supercars\&quot;). Correspondance exacte. | [optional] 
 **q** | **str**| Recherche plein texte sur name/model. | [optional] 
 **dlc** | **str**| Filtre par pack DLC (identifiant d&#39;un dlc_packs) ; liste les voitures du pack. | [optional] 
 **obtain** | **str**| Filtre par méthode d&#39;obtention. obtain_method est multi-valeurs (ex. \&quot;Autoshow, Wheelspin\&quot;) : la correspondance se fait par token — obtain&#x3D;wheelspin renvoie toute voiture dont l&#39;une des méthodes est Wheelspin. Correspondances : autoshow → Autoshow ; wheelspin → Wheelspin ; wristband → Wristband reward / Yellow Wristband ; barn_find → Barn Find ; treasure → Treasure Car ; car_mastery → Car Mastery ; journal → Collection Journal ; car_pass → Car Pass ; hard_to_find → Hard to Find ; aftermarket → Aftermarket Car ; prologue → Complete the Prologue ; loyalty → Loyalty Reward ; preorder → Pre-order ; promotional → Promotional ; vip → VIP Membership ; welcome_pack → Welcome Pack ; unobtainable → Unobtainable. Les voitures de packs DLC se filtrent via &#x60;dlc&#x60;. Compteurs par valeur : facette &#x60;obtainMethods&#x60; de /v1/reference. | [optional] 
 **updated_since** | **datetime**| Ne renvoie que les éléments modifiés après cet instant (synchro incrémentale : un client ne re-télécharge que le delta).  | [optional] 
 **sort** | **str**| Tri du résultat : pi, name, year ou value (valeur en crédits). Préfixe \&quot;-\&quot; pour l&#39;ordre décroissant (ex. sort&#x3D;-pi). Défaut : pi croissant. Tri stable (départage par name puis id) ; les valeurs absentes (year/value NULL) sont renvoyées en dernier. | [optional] 
 **page** | **int**| Numéro de page (1-based). | [optional] [default to 1]
 **page_size** | **int**| Taille de page. | [optional] [default to 50]

### Return type

[**CarList**](CarList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Page de voitures. |  * Cache-Control - Cache court + stale-while-revalidate (la rotation tourne ~hebdo). Revalidation conditionnelle via ETag / If-None-Match (304). <br>  |
**400** | Requête invalide. |  -  |
**401** | Clé API absente ou invalide. |  -  |
**429** | Quota de rate-limit dépassé. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

