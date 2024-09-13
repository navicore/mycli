package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/spf13/cobra"
)

var filePath string

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Read a JSON file and query it using a JSONPath expression",
	Args:  cobra.MaximumNArgs(1), // Accept at most one argument
	Run: func(cmd *cobra.Command, args []string) {
		if filePath == "" {
			fmt.Println("Please specify a file using the -f or --file flag.")
			return
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			return
		}

		var jsonData interface{}
		if err := json.Unmarshal(data, &jsonData); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			return
		}

		if len(args) == 0 {
			// No JSONPath provided, print the entire JSON data
			prettyPrintJSON(jsonData)
		} else {
			jsonPath := args[0]
			// Strip surrounding double quotes if present
			jsonPath = strings.Trim(jsonPath, "\"")
			// Use JSONPath to query the data
			result, err := queryJSONPath(jsonData, jsonPath)
			if err != nil {
				fmt.Printf("Error querying JSONPath: %v\n", err)
				return
			}
			prettyPrintJSON(result)
		}
	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	// Define the -f or --file flag
	readCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the JSON file")
	readCmd.MarkFlagRequired("file")

	// Enable file path completion for the --file flag
	readCmd.RegisterFlagCompletionFunc("file", fileCompletion)

	// Register the dynamic JSONPath completion function
	readCmd.ValidArgsFunction = jsonPathCompletion
}

// fileCompletion provides file path completion for the --file flag
func fileCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

// queryJSONPath queries the JSON data using the provided JSONPath expression
func queryJSONPath(jsonData interface{}, jsonPath string) (interface{}, error) {
	result, err := jsonpath.Get(jsonPath, jsonData)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// prettyPrintJSON formats and prints JSON data
func prettyPrintJSON(data interface{}) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		return
	}
	fmt.Println(string(bytes))
}

// jsonPathCompletion provides dynamic JSONPath suggestions based on the JSON file
func jsonPathCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Debugging output
	fmt.Fprintf(os.Stderr, "DEBUG: jsonPathCompletion called with toComplete='%s'\n", toComplete)

	// Handle inputs starting with a double quote
	isQuoted := false
	if strings.HasPrefix(toComplete, "\"") {
		isQuoted = true
		toComplete = strings.TrimPrefix(toComplete, "\"")
	}

	// Handle inputs containing double quotes (partial completion within quotes)
	if strings.Contains(toComplete, "\"") {
		toComplete = strings.ReplaceAll(toComplete, "\"", "")
		isQuoted = true
	}

	// Get the file path from the --file flag
	filePath, err := cmd.Flags().GetString("file")
	if err != nil || filePath == "" {
		// Cannot provide completions without the file
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Read the JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Error reading file, cannot provide completions
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Unmarshal JSON into interface{}
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		// Error parsing JSON
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Generate suggestions based on the JSON data
	suggestions := generateJSONPathSuggestions(jsonData, toComplete)

	// If the input was quoted, add the starting double quote back to suggestions
	if isQuoted {
		for i, s := range suggestions {
			suggestions[i] = "\"" + s
		}
	}

	// Use appropriate shell completion directives
	return suggestions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

// generateJSONPathSuggestions generates suggestions based on the JSON data and current input
func generateJSONPathSuggestions(jsonData interface{}, toComplete string) []string {
	// Remove leading '$' and '.' from toComplete
	path := strings.TrimPrefix(toComplete, "$")
	path = strings.TrimPrefix(path, ".")

	// Split the path by '.'
	tokens := strings.Split(path, ".")

	// Start from the root of the JSON data
	currentData := jsonData

	// Traverse the JSON data according to the tokens, except the last incomplete token
	for i := 0; i < len(tokens)-1; i++ {
		token := tokens[i]
		if token == "" {
			continue
		}

		// Handle array indices and wildcards
		if strings.Contains(token, "[") {
			token, indexPart := splitArrayToken(token)
			currentData = traverseToArrayElement(currentData, token, indexPart)
			if currentData == nil {
				return nil
			}
			continue
		}

		switch data := currentData.(type) {
		case map[string]interface{}:
			if val, exists := data[token]; exists {
				currentData = val
			} else {
				// Key does not exist
				return nil
			}
		default:
			// Not traversable
			return nil
		}
	}

	// Handle the last incomplete token
	incompleteToken := tokens[len(tokens)-1]

	// Handle array indices and wildcards for the incomplete token
	if strings.Contains(incompleteToken, "[") {
		token, indexPart := splitArrayToken(incompleteToken)
		suggestions := suggestArrayIndices(currentData, token, indexPart, toComplete)
		return suggestions
	}

	suggestions := []string{}

	switch data := currentData.(type) {
	case map[string]interface{}:
		for key := range data {
			if strings.HasPrefix(key, incompleteToken) {
				suggestion := fmt.Sprintf("%s%s", toComplete, key[len(incompleteToken):])
				suggestions = append(suggestions, suggestion)
			}
		}
	case []interface{}:
		// Suggest array indices or '*'
		if strings.HasPrefix("*", incompleteToken) {
			suggestion := fmt.Sprintf("%s%s", toComplete, "*"[len(incompleteToken):])
			suggestions = append(suggestions, suggestion)
		}
		// Suggest numeric indices
		for i := range data {
			indexStr := fmt.Sprintf("%d", i)
			if strings.HasPrefix(indexStr, incompleteToken) {
				suggestion := fmt.Sprintf("%s%s", toComplete, indexStr[len(incompleteToken):])
				suggestions = append(suggestions, suggestion)
			}
		}
	default:
		// Cannot suggest further
	}

	return suggestions
}

// suggestArrayIndices suggests indices and wildcards for arrays
func suggestArrayIndices(currentData interface{}, token string, indexPart string, toComplete string) []string {
	suggestions := []string{}

	// Handle the object key before the array index
	if token != "" {
		switch data := currentData.(type) {
		case map[string]interface{}:
			if val, exists := data[token]; exists {
				currentData = val
			} else {
				// Key does not exist
				return nil
			}
		default:
			// Not traversable
			return nil
		}
	}

	// Handle the array index or wildcard
	if indexPart != "" {
		indexPart = strings.TrimLeft(indexPart, "[")
		incompleteIndex := indexPart
		switch data := currentData.(type) {
		case []interface{}:
			// Suggest indices and wildcard
			for i := range data {
				indexStr := fmt.Sprintf("%d", i)
				if strings.HasPrefix(indexStr, incompleteIndex) {
					suggestion := fmt.Sprintf("%s%s", toComplete, indexStr[len(incompleteIndex):])
					suggestions = append(suggestions, suggestion)
				}
			}
			if strings.HasPrefix("*", incompleteIndex) {
				suggestion := fmt.Sprintf("%s%s", toComplete, "*"[len(incompleteIndex):])
				suggestions = append(suggestions, suggestion)
			}
		default:
			return nil
		}
	}

	return suggestions
}

// splitArrayToken splits a token with array notation, e.g., "book[0]"
func splitArrayToken(token string) (string, string) {
	idx := strings.Index(token, "[")
	if idx == -1 {
		return token, ""
	}
	return token[:idx], token[idx:]
}

// traverseToArrayElement navigates to the specified array element
func traverseToArrayElement(currentData interface{}, token string, indexPart string) interface{} {
	// Handle the object key before the array index
	if token != "" {
		switch data := currentData.(type) {
		case map[string]interface{}:
			if val, exists := data[token]; exists {
				currentData = val
			} else {
				// Key does not exist
				return nil
			}
		default:
			// Not traversable
			return nil
		}
	}

	// Handle the array index or wildcard
	if indexPart != "" {
		indexPart = strings.Trim(indexPart, "[]")
		switch data := currentData.(type) {
		case []interface{}:
			if indexPart == "*" {
				// Wildcard, keep currentData as is
				return currentData
			} else {
				index, err := strconv.Atoi(indexPart)
				if err != nil || index < 0 || index >= len(data) {
					return nil
				}
				currentData = data[index]
			}
		default:
			return nil
		}
	}

	return currentData
}
