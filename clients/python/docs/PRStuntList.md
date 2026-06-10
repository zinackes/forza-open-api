# PRStuntList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[PRStunt]**](PRStunt.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.pr_stunt_list import PRStuntList

# TODO update the JSON string below
json = "{}"
# create an instance of PRStuntList from a JSON string
pr_stunt_list_instance = PRStuntList.from_json(json)
# print the JSON string representation of the object
print(PRStuntList.to_json())

# convert the object into a dict
pr_stunt_list_dict = pr_stunt_list_instance.to_dict()
# create an instance of PRStuntList from a dict
pr_stunt_list_from_dict = PRStuntList.from_dict(pr_stunt_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


