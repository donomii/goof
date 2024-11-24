// errors.go
package main

import (
    "fmt"
    "runtime"
    "strings"
    "path/filepath"
)

// ErrorWithContext wraps an error with additional context information
type ErrorWithContext struct {
    Err       error
    File      string
    Line      int
    Function  string
    Context   string
    Stack     string
}

func (e *ErrorWithContext) Error() string {
    var sb strings.Builder
    
    // Build the main error message with location
    sb.WriteString(fmt.Sprintf("\nError in %s [%s:%d]:\n", e.Function, filepath.Base(e.File), e.Line))
    
    // Add the original error
    sb.WriteString(fmt.Sprintf("→ %v\n", e.Err))
    
    // Add context if present
    if e.Context != "" {
        sb.WriteString(fmt.Sprintf("Context: %s\n", e.Context))
    }
    
    // Add the stack trace
    sb.WriteString("\nStack Trace:\n")
    sb.WriteString(e.Stack)
    
    return sb.String()
}

// NewError creates a new ErrorWithContext
func NewError(err error, context string) error {
    if err == nil {
        return nil
    }
    
    // Get the caller's info
    pc, file, line, _ := runtime.Caller(1)
    fn := runtime.FuncForPC(pc)
    
    // Capture stack trace
    var stack [32]uintptr
    n := runtime.Callers(2, stack[:])
    frames := runtime.CallersFrames(stack[:n])
    
    var stackBuilder strings.Builder
    for {
        frame, more := frames.Next()
        // Skip runtime frames
        if !strings.Contains(frame.File, "runtime/") {
            stackBuilder.WriteString(fmt.Sprintf("    %s\n        %s:%d\n", 
                frame.Function, 
                filepath.Base(frame.File), 
                frame.Line))
        }
        if !more {
            break
        }
    }
    
    return &ErrorWithContext{
        Err:      err,
        File:     file,
        Line:     line,
        Function: fn.Name(),
        Context:  context,
        Stack:    stackBuilder.String(),
    }
}

// NewErrorf creates a new ErrorWithContext with printf-style formatting
func NewErrorf(format string, args ...interface{}) error {
    // Get the caller's info
    pc, file, line, _ := runtime.Caller(1)
    fn := runtime.FuncForPC(pc)
    
    // Format the error message
    err := fmt.Errorf(format, args...)
    
    // Capture stack trace
    var stack [32]uintptr
    n := runtime.Callers(2, stack[:])
    frames := runtime.CallersFrames(stack[:n])
    
    var stackBuilder strings.Builder
    for {
        frame, more := frames.Next()
        // Skip runtime frames
        if !strings.Contains(frame.File, "runtime/") {
            stackBuilder.WriteString(fmt.Sprintf("    %s\n        %s:%d\n", 
                frame.Function, 
                filepath.Base(frame.File), 
                frame.Line))
        }
        if !more {
            break
        }
    }
    
    return &ErrorWithContext{
        Err:      err,
        File:     file,
        Line:     line,
        Function: fn.Name(),
        Context:  "",  // No additional context for direct formatted errors
        Stack:    stackBuilder.String(),
    }
}

// WrapError wraps an existing error with additional context
func WrapError(err error, context string) error {
    if err == nil {
        return nil
    }
    
    // If it's already an ErrorWithContext, just update the context
    if ewc, ok := err.(*ErrorWithContext); ok {
        if context != "" {
            ewc.Context = context + ": " + ewc.Context
        }
        return ewc
    }
    
    return NewError(err, context)
}

// WrapErrorf wraps an existing error with formatted context
func WrapErrorf(err error, format string, args ...interface{}) error {
    if err == nil {
        return nil
    }
    
    context := fmt.Sprintf(format, args...)
    return WrapError(err, context)
}

// LogError logs an error with full context information
func LogError(err error) {
    if err == nil {
        return
    }
    
    if verbose {
        fmt.Fprintf(os.Stderr, "%+v\n", err)
    }
}
