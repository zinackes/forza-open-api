# SearchResultList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[SearchResult]**](SearchResult.md) |  | 

## Example

```python
from forza_open_api_client.models.search_result_list import SearchResultList

# TODO update the JSON string below
json = "{}"
# create an instance of SearchResultList from a JSON string
search_result_list_instance = SearchResultList.from_json(json)
# print the JSON string representation of the object
print(SearchResultList.to_json())

# convert the object into a dict
search_result_list_dict = search_result_list_instance.to_dict()
# create an instance of SearchResultList from a dict
search_result_list_from_dict = SearchResultList.from_dict(search_result_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


