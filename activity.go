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
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
	flogolog "github.com/project-flogo/core/support/log"
)

// Activity is used to create a custom activity. Add values here to retain them.
// Objects used by the time are defined here.
// Common structure
type Activity struct {
	settings *Settings // Defind in metadata.go in this package
	client   mqtt.Client
	logger   flogolog.Logger
	report   []string
	paths    []string
}

// Metadata returns the activity's metadata
// Common function
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// The init function is executed after the package is imported. This function
// runs before any other in the package.
func init() {
	//_ = activity.Register(&Activity{})
	_ = activity.Register(&Activity{}, New)
	connectedOnce = false
}

// Used when the init function is called. The settings, Input and Output
// structures are optional depends application. These structures are
// defined in the metadata.go file in this package.
var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

var connectedOnce bool

// New Looks to be used when the Activity structure contains fields that need to be
// configured using the InitContext information.
// New does this
func New(ctx activity.InitContext) (activity.Activity, error) {
	logger := ctx.Logger()
	logger.Info("espmqtt:New enter")
	s := &Settings{}
	fmt.Println("setting, s:\n", s)
	err := metadata.MapToStruct(ctx.Settings(), s, true)
	if err != nil {
		logger.Error("Failed to convert settings")
		return nil, err
	}
	host := s.Host
	port := s.Port
	mqttDebug := s.MqttDebug
	clientID := s.ClientId

	// Report array: if empty report everything. If not empty only report those in the array.
	var result map[string]interface{}
	fmt.Println("s.Report:\n", s.Report)
	json.Unmarshal([]byte(s.Report), &result)
	fmt.Println("result:\n", result)
	// Only the size required.
	reportArray := make([]string, 0)
	for _, mapper := range result {
		fmt.Println("mapper:\n", mapper)
		mapper1 := mapper.(map[string]interface{})
		fmt.Println("mapper1:\n", mapper1)
		array := mapper1["report"].([]interface{})
		//array := mapper.([]interface{}) // Convert to a slice
		fmt.Println("array:\n", array)
		for _, x := range array {
			// Type assert to string and add to slice
			reportArray = append(reportArray, x.(string))
		}
		sort.Strings(reportArray)
	}
	fmt.Println("reportArray:\n", reportArray)

	var pathResult map[string]interface{}
	fmt.Println(s.Paths)
	json.Unmarshal([]byte(s.Paths), &pathResult)
	fmt.Println("pathResult:\n", pathResult)
	// Only the size required.
	pathArray := make([]string, 0)
	for _, mapper := range pathResult {
		fmt.Println("mapper:\n", mapper)
		mapper1 := mapper.(map[string]interface{})
		fmt.Println("mapper1:\n", mapper1)
		array := mapper1["path"].([]interface{})
		fmt.Println("array:\n", array)
		for _, x := range array {
			// Type assert to string and add to slice
			pathArray = append(pathArray, x.(string))
		}
		sort.Strings(pathArray)
	}
	// Do MQTT stuff here

	// onConnect defines the on connect handler which resets backoff variables.
	var onConnect mqtt.OnConnectHandler = func(client mqtt.Client) {
		flogolog.RootLogger().Warn("Client connected.")
		connectedOnce = true
	}
	// onDisconnect defines the connection lost handler for the mqtt client.
	var onDisconnect mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		flogolog.RootLogger().Warn("Client disconnected. Error: ", err.Error())
		connectedOnce = false
	}

	if mqttDebug {
		mqtt.DEBUG = log.New(os.Stdout, "", 0)
		mqtt.ERROR = log.New(os.Stdout, "", 0)
	}

	opts := mqtt.NewClientOptions()

	broker := "tcp://" + host + ":" + port
	logger.Info("broker ", broker)
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	//opts.SetConnectTimeout(25000 * time.Millisecond)
	opts.SetWriteTimeout(25 * time.Millisecond)
	opts.SetOnConnectHandler(onConnect)
	opts.SetConnectionLostHandler(onDisconnect)
	// Reconnect is used to recover connections without application intervention
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)

	// Create and connect a client using the above options.
	client := mqtt.NewClient(opts)
	// Try the initial connection

	// Create the activity with settings as defaut. Set any other field in
	//the activity here as well
	act := &Activity{settings: s, client: client, logger: logger, report: reportArray, paths: pathArray}
	act.connect()

	logger.Info("espmqtt:New exit")
	return act, nil
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

func (a *Activity) selectReportFields(original map[string]interface{}) map[string]interface{} {
	newPayload := map[string]interface{}{}
	// Copy command key/values
	newPayload["datetime"] = original["datetime"]
	newPayload["messageId"] = original["messageId"]
	values := original["values"].([]map[string]interface{})
	newValues := make([]map[string]interface{}, 0, 10)
	for _, v := range values {
		field := v["field"].(string)
		if linearContains(a.report, field) {
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

	return newPayload
}

// Eval evaluates the activity
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	logger := ctx.Logger()
	logger.Info("espmqtt:Eval enter")

	input := &Input{}
	err = ctx.GetInputObject(input)
	if err != nil {
		logger.Error("Failed to input object")
		return false, err
	}
	payload := input.ConnectorMsg["data"].(map[string]interface{})
	payload["messageId"] = uuid.New().String()

	// Select fields to be reported if defined. Otherwise, send the original
	if len(a.report) > 0 {
		payload = a.selectReportFields(payload)
	}

	// jsonData is a string
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Failed json marshalling", err.Error())
		return false, err
	}
	logger.Info("JsonData: ", string([]byte(jsonData)))

	a.Publish("esp/"+input.ConnectorMsg["entity"].(string), jsonData)

	logger.Info("espmqtt:Eval exit")

	return true, nil
}

func (a *Activity) publishStatus(path string, status string, description string) {
	payload := make(map[string]string)
	payload["status"] = status
	payload["datetime"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	payload["messageId"] = uuid.New().String()
	if len(description) > 0 {
		payload["description"] = description
	}
	// jsonData is a string
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	topic := "esp" + path
	a.Publish(topic, jsonData)
}

func (a *Activity) publishAllStatus(status string, description string) {
	payload := make(map[string]string)
	payload["status"] = status
	payload["datetime"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	payload["messageId"] = uuid.New().String()
	if len(description) > 0 {
		payload["description"] = description
	}
	// jsonData is a string
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for _, path := range a.paths {
		topic := "esp" + path
		a.Publish(topic, jsonData)
	}
}

func (a *Activity) PublishAllRunning() {
	a.publishAllStatus("RUNNING", "")
}
func (a *Activity) PublishAllNotRunning() {
	a.publishAllStatus("NOT_RUNNING", "")
}

func (a *Activity) PublishError(path string, description string) {
	a.publishStatus(path, "ERROR", description)
}

// Cleanup was expected to be called when the application stops.
func (a *Activity) Cleanup() error {

	a.PublishAllNotRunning()

	flogolog.RootLogger().Infof("cleaning up espmqtt activity")

	a.client.Disconnect(10)
	return nil
}

func (a *Activity) connect() error {
	a.logger.Info("Acitvity:connect")
	if token := a.client.Connect(); token.Wait() && token.Error() != nil {
		a.logger.Error("Failed to connect client. Error: ", token.Error())
		return token.Error()
	}
	a.PublishAllRunning()
	return nil
}

// Publish is a wrapper for the publish call on the client object
func (a *Activity) Publish(topic string, payload []byte) error {
	if !a.client.IsConnected() && !connectedOnce {
		a.connect()
	}
	if a.client.IsConnected() {
		if token := a.client.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
			a.logger.Error("Failed to publish payload to device state topic")
			return token.Error()
		}
	}
	return nil
}
