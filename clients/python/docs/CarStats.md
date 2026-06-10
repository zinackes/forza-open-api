# CarStats

Statistiques de performance (échelle du jeu).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**speed** | **float** |  | [optional] 
**handling** | **float** |  | [optional] 
**acceleration** | **float** |  | [optional] 
**launch** | **float** |  | [optional] 
**braking** | **float** |  | [optional] 
**offroad** | **float** |  | [optional] 

## Example

```python
from forza_open_api_client.models.car_stats import CarStats

# TODO update the JSON string below
json = "{}"
# create an instance of CarStats from a JSON string
car_stats_instance = CarStats.from_json(json)
# print the JSON string representation of the object
print(CarStats.to_json())

# convert the object into a dict
car_stats_dict = car_stats_instance.to_dict()
# create an instance of CarStats from a dict
car_stats_from_dict = CarStats.from_dict(car_stats_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


