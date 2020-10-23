package util

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
)

func expandVariable(name string) (value string) {
	return os.Getenv(name)
}

func parseValue(token string) (value string) {
	var builder strings.Builder

	valueStartIndex := 1
	quoteType := token[0]
	if quoteType != '\'' && quoteType != '"' {
		quoteType = ' '
		valueStartIndex = 0
	}

	escapeNext := false
	varNext := false

	var varName strings.Builder

	for _, r := range token[valueStartIndex:] {
		char := byte(r)

		// Break if ends
		if char == quoteType && !escapeNext {
			if varNext {
				builder.WriteString(expandVariable(varName.String()))
			}

			break
		}

		// Processing a variable
		if varNext {
			// Next variable starts here
			if char == '$' && !escapeNext {
				builder.WriteString(expandVariable(varName.String()))
				varName.Reset()
				continue
			}

			// Variable ended
			if !regexp.MustCompile(`^[a-zA-Z_]$`).MatchString(string(char)) {
				builder.WriteString(expandVariable(varName.String()) + string(char))
				varName.Reset()
				varNext = false
				continue
			}

			varName.WriteString(string(char))

			continue
		}

		// Variable start
		if char == '$' && !escapeNext {
			varNext = true
			varName.Reset()
			continue
		}

		builder.WriteString(string(char))

		// Start escaping
		if char == '\\' && !escapeNext {
			escapeNext = true
			continue
		}

		escapeNext = false
	}

	value = builder.String()

	// Expand ~ to home directory
	if strings.HasPrefix(value, "~/") {
		value = filepath.Join(xdg.Home, strings.TrimPrefix(value, "~/"))
	}

	return
}

func LoadEnvFile(path string) (err error) {
	// Parse env file
	file, err := os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line[0] == '#' {
			continue
		}

		entry := strings.SplitN(line, "=", 2)
		keyTokens := strings.Split(entry[0], " ")

		key := keyTokens[len(keyTokens)-1:][0]
		value := parseValue(entry[1])

		os.Setenv(key, value)
	}

	return scanner.Err()
}
