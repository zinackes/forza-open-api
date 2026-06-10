# CarComparison

Comparaison de 2 à 3 voitures, alignées dans l'ordre des ids demandés. Chaque entrée est l'objet Car complet (PI, classe, transmission, stats). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Car]**](Car.md) |  | 

## Example

```python
from forza_open_api_client.models.car_comparison import CarComparison

# TODO update the JSON string below
json = "{}"
# create an instance of CarComparison from a JSON string
car_comparison_instance = CarComparison.from_json(json)
# print the JSON string representation of the object
print(CarComparison.to_json())

# convert the object into a dict
car_comparison_dict = car_comparison_instance.to_dict()
# create an instance of CarComparison from a dict
car_comparison_from_dict = CarComparison.from_dict(car_comparison_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


