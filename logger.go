package gongo

import "fmt"

// Logger interface for standard logger
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}

// Log a wrapper for logging
type Log struct {
	logger Logger
}

// Debugf wraps Debugf
func (c *Log) Debugf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Debugf(format, args...)
	}
}

// Infof wraps Infof
func (c *Log) Infof(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Infof(format, args...)
	}
}

// Warnf wraps Warnf
func (c *Log) Warnf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Warnf(format, args...)
	}
}

// Errorf wraps Errorf
func (c *Log) Errorf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Errorf(format, args...)
	}
}

// Debug wraps Debug
func (c *Log) Debug(args ...interface{}) {
	if c.logger != nil {
		c.logger.Debug(args...)
	}
}

// Info wraps Info
func (c *Log) Info(args ...interface{}) {
	if c.logger != nil {
		c.logger.Info(args...)
	}
}

// Warn wraps Warn
func (c *Log) Warn(args ...interface{}) {
	if c.logger != nil {
		c.logger.Warn(args...)
	}
}

// Error wraps Error
func (c *Log) Error(args ...interface{}) {
	if c.logger != nil {
		c.logger.Error(args...)
	}
}

// SimpleLogger implements a default logger to stdout
type SimpleLogger struct{}

// Debugf implements Debugf
func (c *SimpleLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf("DEBU: %s\n", format), args...)
}

// Infof implements Infof
func (c *SimpleLogger) Infof(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf("INFO: %s\n", format), args...)
}

// Warnf implements Warnf
func (c *SimpleLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf("WARN: %s\n", format), args...)
}

// Errorf implements Errorf
func (c *SimpleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf("ERRO: %s\n", format), args...)
}

// Debug implements Debug
func (c *SimpleLogger) Debug(args ...interface{}) {
	a := make([]interface{}, 0)
	a = append(a, "DEBU: ")
	a = append(a, args...)
	a = append(a, "\n")
	fmt.Print(a...)
}

// Info implements Info
func (c *SimpleLogger) Info(args ...interface{}) {
	a := make([]interface{}, 0)
	a = append(a, "INFO: ")
	a = append(a, args...)
	a = append(a, "\n")
	fmt.Print(a...)
}

// Warn implements Warn
func (c *SimpleLogger) Warn(args ...interface{}) {
	a := make([]interface{}, 0)
	a = append(a, "WARN: ")
	a = append(a, args...)
	a = append(a, "\n")
	fmt.Print(a...)
}

// Error implements Error
func (c *SimpleLogger) Error(args ...interface{}) {
	a := make([]interface{}, 0)
	a = append(a, "ERRO: ")
	a = append(a, args...)
	a = append(a, "\n")
	fmt.Print(a...)
}
