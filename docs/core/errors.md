# Errors Module Documentation

## 1. Introduction to the Errors Module
The `errors` package provides a set of common application-wide error types and utility functions for creating and wrapping errors within the Acacia application. This module aims to standardize error handling and provide more context to errors.

## 2. Key Concepts

### 2.1. Common Application-Wide Errors
The package defines several predefined error variables for common scenarios:

*   `ErrNotFound`: Indicates that a requested resource could not be found.
*   `ErrInvalidInput`: Signifies that the provided input is invalid or malformed.
*   `ErrUnauthorized`: Denotes that the access attempt is not authenticated or lacks necessary credentials.
*   `ErrForbidden`: Indicates that the authenticated user does not have the necessary permissions to perform the action.
*   `ErrInternalServer`: Represents a generic server-side error that occurred unexpectedly.
*   `ErrAlreadyExists`: Indicates that a resource attempting to be created already exists.

### 2.2. New Function
`New(message string) error`
*   Creates a new error with the given message. This is a simple wrapper around Go's standard `errors.New` function.

### 2.3. Wrap Function
`Wrap(err error, message string) error`
*   Adds context to an existing error. If the original error `err` is `nil`, it returns `nil`. Otherwise, it returns a new error that includes the provided `message` and wraps the original error, allowing for error chaining and inspection using `errors.Is` and `errors.As`.

## 3. Usage Examples

### Creating a New Error
```go
package main

import (
	"acacia/core/errors"
	"fmt"
)

func main() {
	err := errors.New("something went wrong")
	fmt.Println(err)
	// Output: something went wrong
}
```

### Using Predefined Errors
```go
package main

import (
	"acacia/core/errors"
	"fmt"
)

func getUser(id string) error {
	if id == "nonexistent" {
		return errors.ErrNotFound
	}
	return nil
}

func main() {
	err := getUser("nonexistent")
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			fmt.Println("User not found error detected.")
		} else {
			fmt.Printf("An unexpected error occurred: %v\n", err)
		}
	}
}
```

### Wrapping an Existing Error
```go
package main

import (
	"acacia/core/errors"
	"fmt"
	"os"
)

func readFile(filename string) error {
	_, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to read file %s", filename))
	}
	return nil
}

func main() {
	err := readFile("nonexistent.txt")
	if err != nil {
		fmt.Println(err)
		// Example Output: failed to read file nonexistent.txt: open nonexistent.txt: no such file or directory

		// You can still check the underlying error
		if os.IsNotExist(errors.Unwrap(err)) {
			fmt.Println("Underlying error is 'file not exist'.")
		}
	}
}
