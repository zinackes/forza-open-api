# BarnFindList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[BarnFind]**](BarnFind.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.barn_find_list import BarnFindList

# TODO update the JSON string below
json = "{}"
# create an instance of BarnFindList from a JSON string
barn_find_list_instance = BarnFindList.from_json(json)
# print the JSON string representation of the object
print(BarnFindList.to_json())

# convert the object into a dict
barn_find_list_dict = barn_find_list_instance.to_dict()
# create an instance of BarnFindList from a dict
barn_find_list_from_dict = BarnFindList.from_dict(barn_find_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


