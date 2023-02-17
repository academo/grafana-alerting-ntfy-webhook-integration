package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	ntfyUrl       = flag.String("ntfy-url", "", "The ntfy url including the topic. e.g.: https://ntfy.sh/mytopic")
	allowInsecure = flag.Bool("allow-insecure", false, "Allow insecure connections to ntfy-url")
	port          = flag.Int("port", 8080, "The port to listen on")
)

var urlRe = regexp.MustCompile(`(https?://.*?)/(\w+)$`)
var topic string
var serverUrl string

func main() {
	flag.Parse()
	var err error

	err = validateFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *allowInsecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	matches := urlRe.FindStringSubmatch(*ntfyUrl)
	if len(matches) != 3 {
		fmt.Println("Error parsing ntfy-url")
		os.Exit(1)
	}
	serverUrl = matches[1]
	topic = matches[2]

	fmt.Println("ntfy-url:", *ntfyUrl)
	fmt.Println("topic:", topic)
	fmt.Println("serverUrl:", serverUrl)

	err = server()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func validateFlags() error {
	if *ntfyUrl == "" {
		return fmt.Errorf("ntfy-url is required")
	}
	if !strings.HasPrefix(*ntfyUrl, "http") {
		return fmt.Errorf("ntfy-url must start with http or https")
	}
	if !urlRe.MatchString(*ntfyUrl) {
		return fmt.Errorf("ntfy-url must follow the format https://ntfy.sh/<topic>. (you may use a custom ntfy server)")
	}
	return nil
}

// start a web server on port 8080 and output any json data to the console from a post request
func server() error {
	fmt.Println("Forwarding Grafana notifications to ntfy...", *ntfyUrl)
	http.HandleFunc("/", handleRequest)
	fmt.Println("Listening on port 8080...")
	err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *port), nil)
	if err != nil {
		return err
	}
	return nil
}

func handleRequest(response http.ResponseWriter, request *http.Request) {
	if request.Method == "POST" {
		// Read the request body
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(response, "Error reading request body", http.StatusBadRequest)
			return
		}

		// Parse the JSON payload
		var payload AlertsPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			http.Error(response, "Error parsing JSON payload", http.StatusBadRequest)
			return
		}

		notificationPayload := prepareNotification(payload)
		err = sendNotification(notificationPayload)
		if err != nil {
			http.Error(response, "Error sending notification", http.StatusInternalServerError)
			return
		}

		// Send response
		response.WriteHeader(http.StatusOK)
		fmt.Fprint(response, "Payload received successfully\n")
	} else {
		http.Error(response, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
}

func prepareNotification(alertPayload AlertsPayload) NtfyNotification {
	firstAlert := alertPayload.Alerts[0]
	actions := []NtfyAction{
		{
			Action: "view",
			Label:  "Open in Grafana",
			Url:    alertPayload.ExternalURL,
			Clear:  true,
		},
		{
			Action: "view",
			Label:  "Silence",
			Url:    firstAlert.SilenceURL,
			Clear:  false,
		},
	}

	// Prepare the payloa
	payload := NtfyNotification{
		Message: alertPayload.Message,
		Title:   alertPayload.Title,
		Topic:   topic,
		Actions: actions,
	}
	return payload
}

func sendNotification(payload NtfyNotification) error {
	// Marshal the payload
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	fmt.Println("Sending notification to ntfy...")
	fmt.Println(string(message))

	// Create a new request using http
	req, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(message))
	if err != nil {
		return err
	}

	// Set the content type to json
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	defer req.Body.Close()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy returned status code %d", resp.StatusCode)
	}

	fmt.Println("Notification sent to ntfy")

	return nil

}
