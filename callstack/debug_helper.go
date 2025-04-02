package callstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StackFrame represents a single frame in the call stack
type StackFrame struct {
	File          string `json:"file"`
	LineNumber    int    `json:"lineNumber"`
	StructureName string `json:"structureName"`
	MethodName    string `json:"methodName"`
}

// StackTrace represents the entire call stack with metadata
type StackTrace struct {
	Sequence  []StackFrame `json:"sequence"`
	Timestamp string       `json:"timestamp"`
	TraceID   string       `json:"traceId"`
}

// GetCallStack returns a structured representation of the call stack
func GetCallStack() []StackFrame {
	// Skip this function and the calling function
	const skip = 2
	const depth = 20

	// Create a buffer for program counters
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])

	if n == 0 {
		return []StackFrame{}
	}

	// Get frame information
	frames := runtime.CallersFrames(pcs[:n])

	// Build the call stack
	var stackFrames []StackFrame
	for {
		frame, more := frames.Next()

		// Parse the function name to extract package, type (if any), and method
		funcNameParts := strings.Split(frame.Function, ".")

		var structureName, methodName string

		if len(funcNameParts) >= 2 {
			// Check if this is a method on a struct type
			methodName = funcNameParts[len(funcNameParts)-1]

			// Look for struct type pattern
			typeIndex := len(funcNameParts) - 2
			typePart := funcNameParts[typeIndex]

			// Check if it's a struct method (has a type component)
			if strings.Contains(typePart, ")") ||
				(typePart != "" && typePart[0] >= 'A' && typePart[0] <= 'Z') {
				// Clean up the type name (remove pointer symbol)
				typePart = strings.TrimPrefix(typePart, "(*")
				typePart = strings.TrimSuffix(typePart, ")")

				if typePart != "" {
					// It's a method on a struct
					structureName = typePart
				}
			} else {
				// Regular function
				structureName = "" // No structure for regular functions
			}
		} else if len(funcNameParts) == 1 {
			methodName = funcNameParts[0]
			structureName = ""
		} else {
			methodName = frame.Function
			structureName = ""
		}

		// Extract filename without full path
		fileName := frame.File
		if lastSlash := strings.LastIndexByte(fileName, '/'); lastSlash >= 0 {
			fileName = fileName[lastSlash+1:]
		}

		stackFrame := StackFrame{
			File:          fileName,
			LineNumber:    frame.Line,
			StructureName: structureName,
			MethodName:    methodName,
		}

		stackFrames = append(stackFrames, stackFrame)

		if !more {
			break
		}
	}

	// Reverse the call stack to show in chronological order
	for i, j := 0, len(stackFrames)-1; i < j; i, j = i+1, j-1 {
		stackFrames[i], stackFrames[j] = stackFrames[j], stackFrames[i]
	}

	return stackFrames
}

// FormatCallStack formats the call stack as a string with => separators
func FormatCallStack(frames []StackFrame) string {
	var parts []string

	for _, frame := range frames {
		var formattedName string
		if frame.StructureName != "" {
			formattedName = fmt.Sprintf("%s.%s(%s:%d)",
				frame.StructureName,
				frame.MethodName,
				frame.File,
				frame.LineNumber)
		} else {
			formattedName = fmt.Sprintf("%s(%s:%d)",
				frame.MethodName,
				frame.File,
				frame.LineNumber)
		}

		parts = append(parts, formattedName)
	}

	return strings.Join(parts, " => ")
}

// LogCallStack logs the call stack and posts it to the API
func LogCallStack() {
	// Get the call stack frames
	stackFrames := GetCallStack()

	// Format and log the call stack with => separators
	formattedStack := FormatCallStack(stackFrames)
	log.Println("Call stack:", formattedStack)

	// Create the full stack trace object for posting
	stackTrace := StackTrace{
		Sequence:  stackFrames,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		TraceID:   fmt.Sprintf("trace-%s", uuid.New().String()[:8]),
	}

	// Post to API in a way that silently continues on failure
	go func() {
		// Marshal to JSON
		jsonData, err := json.Marshal(stackTrace)
		if err != nil {
			return // Silently continue on error
		}

		// Post to the API
		client := &http.Client{
			Timeout: 5 * time.Second, // Add timeout to prevent hanging
		}

		req, err := http.NewRequest("POST",
			"http://localhost:3001/api/stackframes",
			bytes.NewBuffer(jsonData))
		if err != nil {
			return // Silently continue on error
		}

		req.Header.Set("Content-Type", "application/json")

		_, err = client.Do(req)
		if err != nil {
			return // Silently continue on error
		}

		// No need to check status code or close body - silently continue regardless
	}()
}
