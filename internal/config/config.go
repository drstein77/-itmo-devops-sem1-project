package config

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Options struct {
	runAddr     string
	logLevel    string
	dataBaseDSN string
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
	regStringVar(&o.runAddr, "a", getEnvOrDefault("RUN_ADDRESS", ":8080"), "address and port to run server")
	regStringVar(&o.logLevel, "l", getEnvOrDefault("LOG_LEVEL", "debug"), "log level")
	regStringVar(&o.dataBaseDSN, "d", getEnvOrDefault("DATABASE_URI", ""), "database connection string")

	// parse the arguments passed to the server into registered variables
	flag.Parse()
}

func (o *Options) RunAddr() string {
	return o.runAddr
}

func (o *Options) LogLevel() string {
	return o.logLevel
}

func (o *Options) DataBaseDSN() string {
	return o.dataBaseDSN
}

func regStringVar(p *string, name string, value string, usage string) {
	flag.StringVar(p, name, value, usage)
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
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	envPath := filepath.Join(cwd, "..", "..", ".env")

	// Load environment variables from the .env file
	err = godotenv.Load(envPath)
	if err != nil {
		log.Printf("No .env file found at %s, proceeding without it", envPath)
	} else {
		log.Printf(".env file loaded from %s", envPath)
	}
}
