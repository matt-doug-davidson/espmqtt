# ESP MQTT
This activity is used to send data data to ESP MQTT broker

## Installation
### Flogo CLI
```bash
flogo install github.com/matt-doug-davdison/espmqtt
```

## Schema

### Settings

| Setting     | Type   | Required  | Description |
|:------------|:-------|:----------|:------------|
| host  | string      | True | The host running the MQTT broker|
| port | string | True | The MQTT port (typically 1883)|
| clientid | string | True | Unique ID for this MQTT client|
| mqttdebug | boolean | True | Enable (True) or Disable (False) |
| report | JSON String | False | A JSON string with key fo "report" and array of fields to be reported. If not configured, all fields will be reported. |
| path | JSON String| False | The entity path this ESP MQTT object supports. It is used to report RUNNING and NOT_RUNNING states for the path at startup and cleanup respectively. If not define, no state changes are reported.

Report Example:
```json
{
    "report": ["NO2", "O3"]
}
```
Path Exampe:
```json
{
    "path": "/Bucharest/Madison/Inner"
}
```

Only NO2 and O3 fields would be reported.

### Input
```json
{
    "input": [
      {
        "name": "connectorMsg",
        "type": "object",
        "description": "The message connectorMsg object"
      }
    ]
}
```
### Output

*Not applicable*

## Examples
```json
{
  "id": "espmqtt_3",
  "name": "10m ESP MQTT Interface",
  "description": "MQTT interface to ESP",
  "activity": {
    "ref": "#espmqtt",
    "input": {
      "connectorMsg": "=$activity[aeroqualaqy1_2].connectorMsg"
    },
    "settings": {
      "clientid": "Aeroqual10m",
      "host": "10.52.65.189",
      "mqttdebug": false,
      "port": "1883"
    }
  }
}
```

