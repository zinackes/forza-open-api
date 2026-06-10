# TreasureCarList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[TreasureCar]**](TreasureCar.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.treasure_car_list import TreasureCarList

# TODO update the JSON string below
json = "{}"
# create an instance of TreasureCarList from a JSON string
treasure_car_list_instance = TreasureCarList.from_json(json)
# print the JSON string representation of the object
print(TreasureCarList.to_json())

# convert the object into a dict
treasure_car_list_dict = treasure_car_list_instance.to_dict()
# create an instance of TreasureCarList from a dict
treasure_car_list_from_dict = TreasureCarList.from_dict(treasure_car_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


