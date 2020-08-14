package espmqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	flogolog "github.com/project-flogo/core/support/log"
)

type Value struct {
	Field      string          `json:"field"`
	Amount     float64         `json:"amount"`
	Attributes json.RawMessage `json:"attributes"` // Avoid escaped "
}

type ESPSensorValuePayload struct {
	DateTime  string  `json:"datetime"`
	Values    []Value `json:"values"`
	MessageID string  `json:"messageId"`
}

type ESPSensorValueMessage struct {
	Topic   string
	Payload ESPSensorValuePayload
}

type ValueNoAttrib struct {
	Field  string  `json:"field"`
	Amount float64 `json:"amount"`
}

type ESPSensorValuePayloadNoAttrib struct {
	DateTime  string          `json:"datetime"`
	Values    []ValueNoAttrib `json:"values"`
	MessageID string          `json:"messageId"`
}

type ESPSensorValueMessageNoAttrib struct {
	Topic   string
	Payload ESPSensorValuePayloadNoAttrib
}

// Activity is used to create a custom activity. Add values here to retain them.
// Objects used by the time are defined here.
// Common structure
type ESPMqttClient struct {
	host               string
	port               string
	clientID           string
	client             mqtt.Client
	logger             flogolog.Logger
	report             []string
	connectedOnce      bool // Default false
	connectCallback    MqttCallback
	disconnectCallback MqttCallback
	esp                bool
	debug              bool // default true
}

type MqttCallback func()

func NewESPMqttClient(host string, port string, clientID string, esp bool, debug bool) *ESPMqttClient {
	client := &ESPMqttClient{
		host:     host,
		port:     port,
		clientID: clientID,
		esp:      esp,
		debug:    debug}
	client.Initialize()
	return client
}

func (c *ESPMqttClient) RegisterConnectionCallbacks(connect MqttCallback, disconnect MqttCallback) {
	c.connectCallback = connect
	c.disconnectCallback = disconnect
}

func (c *ESPMqttClient) Initialize() {
	fmt.Println("initialize")

	// onConnect defines the on connect handler which resets backoff variables.
	var onConnect mqtt.OnConnectHandler = func(client mqtt.Client) {
		fmt.Println("Client connected.")
		c.connectedOnce = true
		fmt.Println(c.connectedOnce)
		c.connectCallback()
	}

	// onDisconnect defines the connection lost handler for the mqtt client.
	var onDisconnect mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		fmt.Println("Client disconnected. Error: ", err.Error())
		c.connectedOnce = false
		fmt.Println(c.connectedOnce)
		c.disconnectCallback()
	}

	if c.debug {
		mqtt.DEBUG = log.New(os.Stdout, "", 0)
		mqtt.ERROR = log.New(os.Stdout, "", 0)
	}

	opts := mqtt.NewClientOptions()

	broker := "tcp://" + c.host + ":" + c.port
	//fmt.Println("broker ", broker)
	opts.AddBroker(broker)
	opts.SetClientID(c.clientID)
	//opts.SetConnectTimeout(25000 * time.Millisecond)
	opts.SetWriteTimeout(25 * time.Millisecond)
	opts.SetOnConnectHandler(onConnect)
	opts.SetConnectionLostHandler(onDisconnect)
	// Reconnect is used to recover connections without application intervention
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)

	// Create and connect a client using the above options.
	client := mqtt.NewClient(opts)
	c.client = client
	//fmt.Println(c.client)
}

func linearContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func binaryContains(a []string, x string) bool {
	i := sort.Search(len(a), func(i int) bool { return x <= a[i] })
	if i < len(a) && a[i] == x {
		return true
	}
	return false
}

func (c *ESPMqttClient) selectReportFields(original map[string]interface{}) map[string]interface{} {
	newPayload := map[string]interface{}{}
	// Copy command key/values
	newPayload["datetime"] = original["datetime"]
	newPayload["messageId"] = original["messageId"]
	if val, ok := original["status"]; ok {
		newPayload["status"] = val
	}
	if val, ok := original["description"]; ok {
		newPayload["description"] = val
	}
	if val, ok := original["values"]; ok {
		//values := original["values"].([]map[string]interface{})
		values := val.([]map[string]interface{})
		newValues := make([]map[string]interface{}, 0, 10)
		for _, v := range values {
			field := v["field"].(string)
			if linearContains(c.report, field) {
				newValue := map[string]interface{}{}
				newValue["field"] = field
				newValue["amount"] = v["amount"].(float64)
				// Append new value to values....
				newValues = append(newValues, newValue)
			}
		}
		if len(newValues) > 0 {
			newPayload["values"] = newValues
		}
	}

	return newPayload
}

func (c *ESPMqttClient) publishStatus(path string, status string, description string) {
	fmt.Println("PublishStatus")
	payload := make(map[string]string)
	payload["status"] = status
	payload["datetime"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	topicPrefix := ""
	if c.esp {
		payload["messageId"] = uuid.New().String()
		topicPrefix = "esp"
	} else {
		payload["commandId"] = uuid.New().String()
		topicPrefix = "sdw"
	}
	if len(description) > 0 {
		payload["description"] = description
	}
	// jsonData is a string
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshal status message faild. Cause:", err.Error())
		return
	}
	topic := topicPrefix + path
	fmt.Println("Calling publish topic: ", topic, "payload: ", string(jsonData))
	c.Publish(topic, jsonData)
}

// func (a *ESPMqttClient) publishAllStatus(status string, description string) {
// 	payload := make(map[string]string)
// 	payload["status"] = status
// 	payload["datetime"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
// 	payload["messageId"] = uuid.New().String()
// 	if len(description) > 0 {
// 		payload["description"] = description
// 	}
// 	// jsonData is a string
// 	jsonData, err := json.Marshal(payload)
// 	if err != nil {
// 		return
// 	}

// }

// func (a *ESPMqttClient) PublishAllRunning() {
// 	a.publishAllStatus("RUNNING", "")
// }
// func (a *ESPMqttClient) PublishAllNotRunning() {
// 	a.publishAllStatus("NOT_RUNNING", "")
// }

// PublishRunning publish RUNNING status for specified path
func (c *ESPMqttClient) PublishRunning(path string) {
	c.publishStatus(path+"[status]", "RUNNING", "")
}

// PublishNotRunning publishes NOT_RUNNING status for specified path
func (c *ESPMqttClient) PublishNotRunning(path string) {
	c.publishStatus(path+"[status]", "NOT_RUNNING", "")
}

// PublishError publishes ERROR status for specified path with description
func (c *ESPMqttClient) PublishError(path string, description string) {
	c.publishStatus(path+"[status]", "ERROR", description)
}

// Cleanup was expected to be called when the application stops.
func (c *ESPMqttClient) Cleanup() error {

	c.client.Disconnect(10)
	return nil
}

func (c *ESPMqttClient) Connect() error {
	fmt.Println("Acitvity:connect")
	fmt.Println(c)
	fmt.Println(c.client)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("Failed to connect client. Error: ", token.Error())
		return token.Error()
	}
	// Used to send running when connects. I don't know that is what
	// we want to do.
	return nil
}

func (c *ESPMqttClient) PublishESPValueMessage(svm *ESPSensorValueMessage) {
	//fmt.Println(svm.Topic)
	//fmt.Println(svm.Payload)
	svm.Payload.MessageID = uuid.New().String()
	pld, err := json.Marshal(svm.Payload)
	if err != nil {
		fmt.Println(err.Error())
	}
	//fmt.Println(string(pld))
	c.Publish("esp"+svm.Topic, pld)

}

func (c *ESPMqttClient) PublishESPValueMessageNoAttrib(svm *ESPSensorValueMessageNoAttrib) {
	//fmt.Println(svm.Topic)
	//fmt.Println(svm.Payload)
	svm.Payload.MessageID = uuid.New().String()
	pld, err := json.Marshal(svm.Payload)
	if err != nil {
		fmt.Println(err.Error())
	}
	//fmt.Println(string(pld))
	c.Publish("esp"+svm.Topic, pld)

}

// Publish is a wrapper for the publish call on the client object
func (c *ESPMqttClient) Publish(topic string, payload []byte) error {

	//fmt.Println("Publish:\nTopic: ", topic)
	//fmt.Println("payload:\n", string(payload))
	if !c.client.IsConnected() && !c.connectedOnce {
		c.Connect()
	}
	if c.client.IsConnected() {
		if token := c.client.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
			fmt.Println("Failed to publish payload to device state topic")
			return token.Error()
		}
	}
	return nil
}
