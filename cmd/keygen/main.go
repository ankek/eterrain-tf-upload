package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 12 // Higher cost = more secure but slower
)

// OrgConfig represents an organization's configuration with API keys
type OrgConfig struct {
	OrgID   uuid.UUID
	APIKeys []string
}

func main() {
	inputFile := "./init-config.cfg"
	outputFile := "./auth.cfg"

	if len(os.Args) > 1 {
		inputFile = os.Args[1]
	}
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	log.Printf("Reading organizations from: %s", inputFile)
	log.Printf("Generating hashed API keys to: %s", outputFile)

	// Read the input configuration
	orgs, err := readInitConfig(inputFile)
	if err != nil {
		log.Fatalf("Failed to read init config: %v", err)
	}

	log.Printf("Found %d organization(s)", len(orgs))

	// Generate auth config with hashed keys
	if err := generateAuthConfig(orgs, outputFile); err != nil {
		log.Fatalf("Failed to generate auth config: %v", err)
	}

	log.Printf("Successfully generated %s with hashed API keys", outputFile)
	log.Println("All API keys have been hashed using bcrypt with salt")
}

// readInitConfig reads the init-config.cfg file
func readInitConfig(filePath string) ([]OrgConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var orgs []OrgConfig
	var currentOrg *OrgConfig

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if line is an org ID header [UUID]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			orgIDStr := strings.TrimSpace(line[1 : len(line)-1])
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid UUID on line %d: %s", lineNum, orgIDStr)
			}

			// Save previous org if exists
			if currentOrg != nil {
				orgs = append(orgs, *currentOrg)
			}

			// Start new org
			currentOrg = &OrgConfig{
				OrgID:   orgID,
				APIKeys: []string{},
			}
			continue
		}

		// If we have a current org, this line is an API key
		if currentOrg != nil {
			apiKey := line
			if apiKey != "" {
				currentOrg.APIKeys = append(currentOrg.APIKeys, apiKey)
			}
		} else {
			return nil, fmt.Errorf("API key on line %d appears before any org ID declaration", lineNum)
		}
	}

	// Save the last org
	if currentOrg != nil {
		orgs = append(orgs, *currentOrg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return orgs, nil
}

// generateAuthConfig generates the auth.cfg file with hashed API keys
func generateAuthConfig(orgs []OrgConfig, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	fmt.Fprintf(writer, "# Authentication configuration file\n")
	fmt.Fprintf(writer, "# Generated automatically - DO NOT EDIT MANUALLY\n")
	fmt.Fprintf(writer, "# Format: [OrgID]\n")
	fmt.Fprintf(writer, "# followed by bcrypt-hashed API keys (one per line)\n\n")

	for i, org := range orgs {
		if i > 0 {
			fmt.Fprintf(writer, "\n")
		}

		// Write org ID header
		fmt.Fprintf(writer, "[%s]\n", org.OrgID.String())

		// Hash and write each API key
		for _, apiKey := range org.APIKeys {
			hashedKey, err := hashAPIKey(apiKey)
			if err != nil {
				return fmt.Errorf("failed to hash API key for org %s: %w", org.OrgID, err)
			}
			fmt.Fprintf(writer, "%s\n", hashedKey)
			log.Printf("Hashed API key for org %s: %s -> %s...", org.OrgID, apiKey, hashedKey[:20])
		}
	}

	return nil
}

// hashAPIKey hashes an API key using bcrypt
func hashAPIKey(apiKey string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedBytes), nil
}

// generateRandomAPIKey generates a cryptographically secure random API key
func generateRandomAPIKey() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
