package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func runPairing(envPath string) error {
	values, err := godotenv.Read(envPath)
	if err != nil {
		return fmt.Errorf("read environment file %q: %w", envPath, err)
	}
	apiURL := strings.TrimRight(strings.TrimSpace(values["API_URL"]), "/")
	if apiURL == "" {
		return fmt.Errorf("API_URL is required in %q", envPath)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("This CLI captures printable text typed while Chrome is active and active window titles.")
	fmt.Println("That can include sensitive text. Pair only with the device user's informed consent.")
	fmt.Print("Type yes to continue: ")
	consent, err := readLine(reader)
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if !strings.EqualFold(consent, "yes") {
		fmt.Println("Pairing cancelled.")
		return nil
	}

	fmt.Print("Enter the one-time pairing code from the parent dashboard: ")
	code, err := readLine(reader)
	if err != nil {
		return fmt.Errorf("read pairing code: %w", err)
	}
	if code == "" {
		return fmt.Errorf("pairing code is required")
	}

	deviceID, err := newDeviceID()
	if err != nil {
		return fmt.Errorf("create device id: %w", err)
	}
	payload, err := json.Marshal(map[string]string{
		"code":      code,
		"device_id": deviceID,
		"platform":  runtime.GOOS,
	})
	if err != nil {
		return fmt.Errorf("encode pairing request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, apiURL+"/api/v1/devices/pair", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create pairing request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send pairing request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("pairing rejected (%s); read error: %w", response.Status, readErr)
		}
		return fmt.Errorf("pairing rejected (%s): %s", response.Status, strings.TrimSpace(string(message)))
	}

	var child struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(response.Body).Decode(&child); err != nil {
		return fmt.Errorf("decode pairing response: %w", err)
	}
	if child.ID == "" {
		return fmt.Errorf("pairing response did not include a child id")
	}

	values["CHILD_ID"] = child.ID
	values["DEVICE_ID"] = deviceID
	updatedEnv, err := godotenv.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode updated environment: %w", err)
	}
	if err := os.WriteFile(envPath, []byte(updatedEnv+"\n"), 0600); err != nil {
		return fmt.Errorf("save paired device environment: %w", err)
	}
	if err := os.Chmod(envPath, 0600); err != nil {
		return fmt.Errorf("secure paired device environment: %w", err)
	}
	fmt.Printf("Paired this device with %s. Restart the CLI normally with --env %s to begin monitoring.\n", child.Name, envPath)
	return nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func newDeviceID() (string, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	return hex.EncodeToString(id), nil
}
