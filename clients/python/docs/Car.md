# Car


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**make** | **str** |  | 
**model** | **str** |  | [optional] 
**year** | **int** |  | [optional] 
**var_class** | [**CarClass**](CarClass.md) |  | 
**pi** | **int** |  | 
**drivetrain** | [**Drivetrain**](Drivetrain.md) |  | 
**stats** | [**CarStats**](CarStats.md) |  | [optional] 
**body_type** | **str** |  | [optional] 
**category** | **str** | Catégorie / division in-game (ex. \&quot;Modern Supercars\&quot;). Absente si non sourcée. | [optional] 
**rarity** | **str** |  | [optional] 
**value_cr** | **int** |  | [optional] 
**obtain_method** | **str** |  | [optional] 
**image_url** | **str** |  | [optional] 
**created_at** | **datetime** |  | [optional] 
**updated_at** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.car import Car

# TODO update the JSON string below
json = "{}"
# create an instance of Car from a JSON string
car_instance = Car.from_json(json)
# print the JSON string representation of the object
print(Car.to_json())

# convert the object into a dict
car_dict = car_instance.to_dict()
# create an instance of Car from a dict
car_from_dict = Car.from_dict(car_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


