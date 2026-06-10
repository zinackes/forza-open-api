# SearchResult

Résultat de la recherche globale. id permet le lookup détaillé sur la ressource correspondante (pour manufacturer, id = nom du constructeur). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**kind** | [**SearchResultKind**](SearchResultKind.md) |  | 
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**detail** | **str** | Contexte court d&#39;affichage (ex. \&quot;Ford · A 800\&quot; pour une voiture, \&quot;circuit\&quot; pour un tracé). Absent si rien à ajouter.  | [optional] 

## Example

```python
from forza_open_api_client.models.search_result import SearchResult

# TODO update the JSON string below
json = "{}"
# create an instance of SearchResult from a JSON string
search_result_instance = SearchResult.from_json(json)
# print the JSON string representation of the object
print(SearchResult.to_json())

# convert the object into a dict
search_result_dict = search_result_instance.to_dict()
# create an instance of SearchResult from a dict
search_result_from_dict = SearchResult.from_dict(search_result_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


