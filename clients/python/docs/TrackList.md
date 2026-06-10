# TrackList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Track]**](Track.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.track_list import TrackList

# TODO update the JSON string below
json = "{}"
# create an instance of TrackList from a JSON string
track_list_instance = TrackList.from_json(json)
# print the JSON string representation of the object
print(TrackList.to_json())

# convert the object into a dict
track_list_dict = track_list_instance.to_dict()
# create an instance of TrackList from a dict
track_list_from_dict = TrackList.from_dict(track_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


