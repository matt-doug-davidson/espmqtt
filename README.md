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