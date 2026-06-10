# ChangeList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Change]**](Change.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.change_list import ChangeList

# TODO update the JSON string below
json = "{}"
# create an instance of ChangeList from a JSON string
change_list_instance = ChangeList.from_json(json)
# print the JSON string representation of the object
print(ChangeList.to_json())

# convert the object into a dict
change_list_dict = change_list_instance.to_dict()
# create an instance of ChangeList from a dict
change_list_from_dict = ChangeList.from_dict(change_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


