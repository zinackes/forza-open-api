# Me

Identité et quota de la clé API présentée dans X-API-Key (endpoint /v1/me, authentification requise). remaining/resetAt reflètent l'état courant de la fenêtre glissante de rate-limit (cohérents avec les en-têtes X-RateLimit-* de la réponse) ; tous deux omis si le compteur (Redis) est indisponible. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Nom de la clé (libellé donné à la création). | 
**scopes** | **List[str]** | Permissions accordées à la clé (ex. read, submit-ugc). | 
**rate_limit** | **int** | Quota maximal de requêtes sur la fenêtre glissante. | 
**remaining** | **int** | Requêtes restantes sur la fenêtre courante (compteur sliding-window). Omis si le compteur (Redis) est indisponible.  | [optional] 
**reset_at** | **datetime** | Instant où un créneau de quota se libère (fin de la fenêtre courante). Omis si le compteur (Redis) est indisponible.  | [optional] 
**created_at** | **datetime** | Date de création de la clé. | 

## Example

```python
from forza_open_api_client.models.me import Me

# TODO update the JSON string below
json = "{}"
# create an instance of Me from a JSON string
me_instance = Me.from_json(json)
# print the JSON string representation of the object
print(Me.to_json())

# convert the object into a dict
me_dict = me_instance.to_dict()
# create an instance of Me from a dict
me_from_dict = Me.from_dict(me_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


