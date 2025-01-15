package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Options struct {
	flagRunAddr, flagLogLevel, flagDataBaseDSN string
}

func NewOptions() *Options {
	return new(Options)
}

// ParseFlags handles command line arguments
// and stores their values in the corresponding variables.
func (o *Options) ParseFlags() {
	// Load environment variables from the .env file
	loadEnvFile()

	// Override variable values with values from command line flags
	regStringVar(&o.flagRunAddr, "a", getEnvOrDefault("RUN_ADDRESS", ":8080"), "address and port to run server")
	regStringVar(&o.flagDataBaseDSN, "d", getEnvOrDefault("DATABASE_URI", ""), "")
	regStringVar(&o.flagLogLevel, "l", getEnvOrDefault("LOG_LEVEL", "debug"), "log level")

	// parse the arguments passed to the server into registered variables
	flag.Parse()
}

func (o *Options) RunAddr() string {
	return o.flagRunAddr
}

func (o *Options) LogLevel() string {
	return o.flagLogLevel
}

func (o *Options) DataBaseDSN() string {
	return o.flagDataBaseDSN
}

func regStringVar(p *string, name string, value string, usage string) {
	if flag.Lookup(name) == nil {
		flag.StringVar(p, name, value, usage)
	}
}

// getEnvOrDefault reads an environment variable or returns a default value if the variable is not set or is empty.
func getEnvOrDefault(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile() {
	// Determine the path to the .env file relative to the current working directory
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDir := filepath.Dir(execPath)
	envPath := filepath.Join(execDir, "..", "..", ".env")

	// Load environment variables from the .env file
	err = godotenv.Load(envPath)
	if err != nil {
		log.Printf("No .env file found at %s, proceeding without it", envPath)
	} else {
		log.Printf(".env file loaded from %s", envPath)
	}
}
