# Reference

Facettes de référence pour amorcer les filtres d'un client. Les compteurs classes/drivetrains/bodyTypes/countries/categories sont scopés au `game` demandé ; games est global. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**game** | [**Game**](Game.md) |  | 
**classes** | [**List[RefCount]**](RefCount.md) | Classes PI et leurs compteurs (ordre PI, R en dernier ; count 0 inclus). | 
**drivetrains** | [**List[RefCount]**](RefCount.md) | Transmissions et leurs compteurs (count 0 inclus). | 
**body_types** | [**List[RefCount]**](RefCount.md) | Types de carrosserie présents (plus fréquents d&#39;abord). | 
**countries** | [**List[RefCount]**](RefCount.md) | Pays des constructeurs présents (plus fréquents d&#39;abord). | 
**categories** | [**List[RefCount]**](RefCount.md) | Catégories / divisions in-game présentes (plus fréquentes d&#39;abord). | 
**regions** | [**List[RefCount]**](RefCount.md) | Régions présentes sur les ressources carte (count &#x3D; tracés + événements + PR stunts de la région ; plus fréquentes d&#39;abord). Évite aux clients carte de hardcoder la liste des régions.  | 
**track_types** | [**List[RefCount]**](RefCount.md) | Types de tracé et leurs compteurs (ordre canonique, count 0 inclus). | 
**event_types** | [**List[RefCount]**](RefCount.md) | Types d&#39;événement et leurs compteurs (ordre canonique, count 0 inclus). | 
**pr_stunt_types** | [**List[RefCount]**](RefCount.md) | Types de PR Stunt et leurs compteurs (ordre canonique, count 0 inclus). | 
**obtain_methods** | [**List[RefCount]**](RefCount.md) | Méthodes d&#39;obtention (valeurs du paramètre obtain de /v1/cars) et leurs compteurs (ordre canonique, count 0 inclus). obtain_method étant multi-valeurs, une voiture compte dans chacune de ses méthodes.  | 
**games** | [**List[GameCount]**](GameCount.md) | Jeux disponibles et leurs volumes (global, indépendant du paramètre game). | 

## Example

```python
from forza_open_api_client.models.reference import Reference

# TODO update the JSON string below
json = "{}"
# create an instance of Reference from a JSON string
reference_instance = Reference.from_json(json)
# print the JSON string representation of the object
print(Reference.to_json())

# convert the object into a dict
reference_dict = reference_instance.to_dict()
# create an instance of Reference from a dict
reference_from_dict = Reference.from_dict(reference_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


