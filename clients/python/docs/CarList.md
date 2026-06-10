# CarList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Car]**](Car.md) |  | 
**total** | **int** |  | 
**page** | **int** |  | 
**page_size** | **int** |  | 

## Example

```python
from forza_open_api_client.models.car_list import CarList

# TODO update the JSON string below
json = "{}"
# create an instance of CarList from a JSON string
car_list_instance = CarList.from_json(json)
# print the JSON string representation of the object
print(CarList.to_json())

# convert the object into a dict
car_list_dict = car_list_instance.to_dict()
# create an instance of CarList from a dict
car_list_from_dict = CarList.from_dict(car_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


