# BarnFind

Barn Find FH6 : épave cachée à localiser dans une zone de recherche, puis à restaurer. Déblocage progressif via les stamps Discover Japan (prerequisiteStampLevel 1-7 : Visitor, Sightseer, Traveller, Pathfinder, Navigator, Adventurer, Master Explorer). Mécanique distincte des Treasure Cars. Coordonnées NULL si non sourcées (datasets communautaires). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**game** | [**Game**](Game.md) |  | 
**car_id** | **str** | Identifiant de la voiture obtenue (réf. /v1/cars). | 
**region** | **str** | Région de la carte (dépend du jeu ; FH6 &#x3D; 7 régions / 74 districts). Valeur libre : l&#39;ensemble n&#39;est pas figé au contrat. Absente → champ omis.  | [optional] 
**search_zone_center_lat** | **float** | Latitude du centre de la zone de recherche. | [optional] 
**search_zone_center_lng** | **float** | Longitude du centre de la zone de recherche. | [optional] 
**search_zone_radius_m** | **int** | Rayon de la zone de recherche (mètres). | [optional] 
**prerequisite_stamp_level** | **int** | Niveau de stamp Discover Japan requis pour débloquer ce Barn Find (1&#x3D;Visitor, 2&#x3D;Sightseer, 3&#x3D;Traveller, 4&#x3D;Pathfinder, 5&#x3D;Navigator, 6&#x3D;Adventurer, 7&#x3D;Master Explorer).  | [optional] 
**restoration_time_h** | **int** | Durée de restauration (heures de jeu). | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.barn_find import BarnFind

# TODO update the JSON string below
json = "{}"
# create an instance of BarnFind from a JSON string
barn_find_instance = BarnFind.from_json(json)
# print the JSON string representation of the object
print(BarnFind.to_json())

# convert the object into a dict
barn_find_dict = barn_find_instance.to_dict()
# create an instance of BarnFind from a dict
barn_find_from_dict = BarnFind.from_dict(barn_find_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


