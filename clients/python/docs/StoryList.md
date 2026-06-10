# StoryList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Story]**](Story.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.story_list import StoryList

# TODO update the JSON string below
json = "{}"
# create an instance of StoryList from a JSON string
story_list_instance = StoryList.from_json(json)
# print the JSON string representation of the object
print(StoryList.to_json())

# convert the object into a dict
story_list_dict = story_list_instance.to_dict()
# create an instance of StoryList from a dict
story_list_from_dict = StoryList.from_dict(story_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


