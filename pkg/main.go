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
)

var (
	allowInsecure = flag.Bool("allow-insecure", false, "Allow insecure connections to ntfy-url")
	listenAddr    = flag.String("addr", ":8080", "The address to listen on")
	debug         = flag.Bool("debug", false, "print extra debug information")
)

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

	slog.Info(
		"starting server",
		"listen_addr", *listenAddr,
	)

	err = server()
	if err != nil {
		slog.Error("server startup error", err)

		os.Exit(1)
	}
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
	    matches := urlRe.FindStringSubmatch(request.URL.RequestURI())
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
		err = sendNotification(notificationPayload, request.Header.Get("Authorization"), serverUrl)
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

func sendNotification(payload NtfyNotification, authHeader string, serverUrl string) error {
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

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

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

	slog.Debug("Notification sent to ntfy")

	return nil

}
