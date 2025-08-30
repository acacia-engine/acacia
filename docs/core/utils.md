# Utils Module Documentation

## 1. Introduction to the Utils Module
The `utils` package provides a collection of generic utility functions that can be used across different parts of the Acacia application. These functions are designed to perform common, reusable operations that do not fit into more specific modules.

## 2. Key Concepts

### 2.1. `ReverseString` Function
`ReverseString(s string) string`
*   Reverses the characters of a given input string `s`. It handles Unicode characters correctly by operating on runes.

### 2.2. `IsValidEmail` Function
`IsValidEmail(email string) bool`
*   Checks if a given string `email` is a valid email address using a basic regular expression.

### 2.3. `ReadFileContent` Function
`ReadFileContent(filePath string) (string, error)`
*   Reads the content of a file at the specified `filePath` and returns it as a string. Returns an error if the file cannot be read.

## 3. Usage Examples

### Reversing a String
```go
package main

import (
	"acacia/core/utils"
	"fmt"
)

func main() {
	original := "Hello, World!"
	reversed := utils.ReverseString(original)
	fmt.Printf("Original: %s\n", original)
	fmt.Printf("Reversed: %s\n", reversed)
	// Output:
	// Original: Hello, World!
	// Reversed: !dlroW ,olleH

	unicodeString := "你好世界"
	reversedUnicode := utils.ReverseString(unicodeString)
	fmt.Printf("Original Unicode: %s\n", unicodeString)
	fmt.Printf("Reversed Unicode: %s\n", reversedUnicode)
	// Output:
	// Original Unicode: 你好世界
	// Reversed Unicode: 界世好你
}
```

### Validating an Email Address
```go
package main

import (
	"acacia/core/utils"
	"fmt"
)

func main() {
	email1 := "test@example.com"
	email2 := "invalid-email"

	fmt.Printf("'%s' is valid: %t\n", email1, utils.IsValidEmail(email1))
	fmt.Printf("'%s' is valid: %t\n", email2, utils.IsValidEmail(email2))
	// Output:
	// 'test@example.com' is valid: true
	// 'invalid-email' is valid: false
}
```

### Reading File Content
```go
package main

import (
	"acacia/core/utils"
	"fmt"
	"os"
)

func main() {
	// Create a dummy file for testing
	filePath := "test_file.txt"
	err := os.WriteFile(filePath, []byte("This is a test file content."), 0644)
	if err != nil {
		fmt.Printf("Error creating test file: %v\n", err)
		return
	}
	defer os.Remove(filePath) // Clean up the dummy file

	content, err := utils.ReadFileContent(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	fmt.Printf("File content:\n%s\n", content)
	// Output:
	// File content:
	// This is a test file content.

	// Test with a non-existent file
	_, err = utils.ReadFileContent("non_existent_file.txt")
	if err != nil {
		fmt.Printf("Error reading non-existent file (expected): %v\n", err)
	}
	// Output:
	// Error reading non-existent file (expected): failed to read file non_existent_file.txt: open non_existent_file.txt: no such file or directory
}
```
