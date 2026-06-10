# Manufacturer


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**game** | [**Game**](Game.md) |  | 
**name** | **str** |  | 
**country** | **str** |  | [optional] 
**car_count** | **int** | Nombre de voitures du constructeur pour le jeu (agrégat GROUP BY ; 0 si aucune). | 

## Example

```python
from forza_open_api_client.models.manufacturer import Manufacturer

# TODO update the JSON string below
json = "{}"
# create an instance of Manufacturer from a JSON string
manufacturer_instance = Manufacturer.from_json(json)
# print the JSON string representation of the object
print(Manufacturer.to_json())

# convert the object into a dict
manufacturer_dict = manufacturer_instance.to_dict()
# create an instance of Manufacturer from a dict
manufacturer_from_dict = Manufacturer.from_dict(manufacturer_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


