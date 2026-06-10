# TourList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Tour]**](Tour.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.tour_list import TourList

# TODO update the JSON string below
json = "{}"
# create an instance of TourList from a JSON string
tour_list_instance = TourList.from_json(json)
# print the JSON string representation of the object
print(TourList.to_json())

# convert the object into a dict
tour_list_dict = tour_list_instance.to_dict()
# create an instance of TourList from a dict
tour_list_from_dict = TourList.from_dict(tour_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


