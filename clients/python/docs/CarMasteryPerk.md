# CarMasteryPerk

Perk de l'arbre Car Mastery FH6 : une case (row, col) de la grille 4×4 de la voiture, débloquée contre des Skill Points. Champs non sourcés → NULL (sourcing progressif : dataset forzagarage.com + wiki Fandom). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | 
**car_id** | **str** | Voiture propriétaire de l&#39;arbre (réf. /v1/cars). | 
**row** | **int** | Ligne dans la grille (1-4). | [optional] 
**col** | **int** | Colonne dans la grille (1-4). | [optional] 
**name** | **str** |  | [optional] 
**sp_cost** | **int** | Coût en Skill Points. | [optional] 
**effect_description** | **str** |  | [optional] 
**effect_type** | **str** | Type d&#39;effet (valeur libre). Courants : credits, xp_boost, wheelspin, super_wheelspin, skill_score, car_unlock.  | [optional] 
**effect_value** | **int** | Valeur de l&#39;effet (montant de crédits ou pourcentage de boost selon le type). | [optional] 
**prereq_perk_id** | **str** | Perk prérequise dans la grille. Absent si point d&#39;entrée. | [optional] 
**unlocked_car_id** | **str** | Voiture cachée débloquée par cette perk (effectType car_unlock ; réf. /v1/cars). Absent sinon.  | [optional] 
**source** | **str** | Source propre de la donnée. | [optional] 
**last_verified** | **datetime** |  | [optional] 

## Example

```python
from forza_open_api_client.models.car_mastery_perk import CarMasteryPerk

# TODO update the JSON string below
json = "{}"
# create an instance of CarMasteryPerk from a JSON string
car_mastery_perk_instance = CarMasteryPerk.from_json(json)
# print the JSON string representation of the object
print(CarMasteryPerk.to_json())

# convert the object into a dict
car_mastery_perk_dict = car_mastery_perk_instance.to_dict()
# create an instance of CarMasteryPerk from a dict
car_mastery_perk_from_dict = CarMasteryPerk.from_dict(car_mastery_perk_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


