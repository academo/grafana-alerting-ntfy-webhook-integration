package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	ntfyUrl       = flag.String("ntfy-url", "", "The ntfy url including the topic. e.g.: https://ntfy.sh/mytopic")
	username      = flag.String("username", "", "The ntfy username")
	password      = flag.String("password", "", "The ntfy password")
	allowInsecure = flag.Bool("allow-insecure", false, "Allow insecure connections to ntfy-url")
	port          = flag.Int("port", 8080, "DEPRECATED. Use listenAddr. The port to listen on")
	listenAddr    = flag.String("addr", ":8080", "The address to listen on")
	debug         = flag.Bool("debug", false, "print extra debug information")
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

var urlRe = regexp.MustCompile(`(https?://.*?)/([-a-zA-Z0-9()@:%_\+.~#?&=]+)$`)

func main() {
	flag.Parse()
	var err error

	if *allowInsecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if *debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	matches := urlRe.FindStringSubmatch(*ntfyUrl)
	if len(matches) != 3 {
		slog.Error("Error parsing ntfy-url")

		os.Exit(1)
	}

	serverUrl = matches[1]
	topic = matches[2]

	// fallback for existing port option
	if *port != 8080 && *listenAddr == ":8080" {
		slog.Warn("Using deprecated -port flag. Please use -addr instead")
		*listenAddr = fmt.Sprintf(":%d", *port)
	}

	slog.Info(
		"starting server",
		"ntfy_url", *ntfyUrl,
		"topic", topic,
		"server_url", serverUrl,
		"listen_addr", *listenAddr,
	)

	err = server()
	if err != nil {
		slog.Error("server startup error", "error", err)
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

// start a web server on the configured address
func server() error {
	http.HandleFunc("/", handleRequest)

	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		return err
	}

	return nil
}

func handleRequest(response http.ResponseWriter, request *http.Request) {
	if request.Method == "POST" {
		fmt.Println(request.URL.RequestURI())
		matches := urlRe.FindStringSubmatch(request.URL.RequestURI())
		fmt.Println(matches)
		if len(matches) != 3 {
			slog.Error("Error parsing ntfy-url")
			return
		}

		var serverUrl string = matches[1]
		var topic string = matches[2]

		// Read the request body
		body, err := io.ReadAll(request.Body)
		if err != nil {
			slog.Error("Error reading request body", "err", err)
			http.Error(response, "Error reading request body", http.StatusBadRequest)

			return
		}

		// Parse the JSON payload
		var payload AlertsPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			slog.Error("Error parsing JSON payload", "err", err)
			http.Error(response, "Error parsing JSON payload", http.StatusBadRequest)

			return
		}

		notificationPayload := prepareNotification(payload, topic)
		err = sendNotification(notificationPayload, request.Header.Get("Authorization"), http.DefaultClient, serverUrl)
		if err != nil {
			slog.Error("Error sending notification", "err", err)
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

func prepareNotification(alertPayload AlertsPayload, topic string) NtfyNotification {
	// edge case with a non-alert
	if len(alertPayload.Alerts) == 0 {
		return NtfyNotification{
			Message: alertPayload.Message,
			Title:   alertPayload.Title,
			Topic:   topic,
		}
	}

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

	// Prepare the payload
	payload := NtfyNotification{
		Message: alertPayload.Message,
		Title:   alertPayload.Title,
		Topic:   topic,
		Actions: actions,
	}

	return payload
}

func sendNotification(payload NtfyNotification, authHeader string, client HttpClient, serverUrl string) error {
	// Marshal the payload
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	slog.Debug("Sending notification to ntfy...", "body", string(message))

	// Create a new request using http
	req, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(message))
	if err != nil {
		return err
	}

	// Set the content type to json
	req.Header.Set("Content-Type", "application/json")

	// Add auth if provided
	if *username != "" && *password != "" {
		req.SetBasicAuth(*username, *password)
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Send the request
	defer req.Body.Close()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy returned status code %d", resp.StatusCode)
	}

	slog.Debug("Notification sent to ntfy")

	return nil

}
