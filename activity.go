package espmqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	}
	fmt.Println("reportArray:\n", reportArray)
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
	act := &Activity{settings: s, client: client, logger: logger, report: reportArray}
	act.connect()

	logger.Info("espmqtt:New exit")
	return act, nil
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

// Cleanup was expected to be called when the application stops.
func (a *Activity) Cleanup() error {

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
