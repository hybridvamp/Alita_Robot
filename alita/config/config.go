package config

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	// AllowedUpdates specifies which Telegram update types the bot should receive.
	AllowedUpdates []string
	// ValidLangCodes lists enabled locale codes for i18n.
	ValidLangCodes []string
	// BotToken is the Telegram bot token.
	BotToken string
	// DatabaseURI is the URI for the main database connection.
	DatabaseURI string
	// MainDbName is the name of the main database.
	MainDbName string
	// BotVersion is the current version of the bot.
	BotVersion string
	// ApiServer is the Telegram API server endpoint.
	ApiServer string
	// WorkingMode indicates the current working mode (e.g., "worker").
	WorkingMode = "worker"
	// Debug enables debug logging if true.
	Debug = false
	// DropPendingUpdates determines if pending updates should be dropped on startup.
	DropPendingUpdates = true
	// OwnerId is the Telegram user ID of the bot owner.
	OwnerId int64
	// MessageDump is the chat ID where startup/log messages are sent.
	MessageDump int64
	// RedisAddress is the address of the Redis server.
	RedisAddress string
	// RedisPassword is the password for the Redis server.
	RedisPassword string
	// RedisDB is the Redis database number to use.
	RedisDB int
	// Logger is the configured logger instance
	Logger *log.Logger
	// MongoDB connection pool settings
	MongoMaxPoolSize     uint64
	MongoMinPoolSize     uint64
	MongoMaxConnIdleTime time.Duration
	MongoMaxIdleTime     time.Duration
	// Resource monitoring thresholds
	HighGoroutineThreshold int
	HighMemoryThresholdMB  int
)

// defaultAllowedUpdates defines the complete set of Telegram update types the bot can subscribe to.
var defaultAllowedUpdates = []string{
	"message",
	"edited_message",
	"channel_post",
	"edited_channel_post",
	"inline_query",
	"chosen_inline_result",
	"callback_query",
	"shipping_query",
	"pre_checkout_query",
	"poll",
	"poll_answer",
	"my_chat_member",
	"chat_member",
	"chat_join_request",
}

// defaultValidLangCodes holds the language codes enabled when no explicit value is provided.
var defaultValidLangCodes = []string{"en"}

func parseUint64Env(key string, def uint64) uint64 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	u, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return def
	}
	return u
}

func parseDurationEnv(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}

// init initializes the config variables from environment variables and sets up logging.
// It loads .env files, parses environment variables, and applies defaults for unset values.
func init() {
	// Load environment variables from .env before we evaluate any settings
	godotenv.Load()

	// Determine debug mode early
	Debug = typeConvertor{str: os.Getenv("DEBUG")}.Bool()

	// Create and configure a structured logger instance (modern approach)
	Logger = log.New()

	if Debug {
		Logger.SetLevel(log.DebugLevel)
		Logger.SetReportCaller(true)
		Logger.SetFormatter(&log.JSONFormatter{
			DisableHTMLEscape: true,
			PrettyPrint:       true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				return f.Function, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	} else {
		Logger.SetLevel(log.InfoLevel)
		Logger.SetReportCaller(false)
		Logger.SetFormatter(&log.JSONFormatter{DisableHTMLEscape: true})
	}

	// Set global logger to use the configured instance (for backward compatibility)
	log.SetOutput(Logger.Out)
	log.SetLevel(Logger.Level)
	log.SetFormatter(Logger.Formatter)
	log.SetReportCaller(Logger.ReportCaller)

	// set necessary variables
	DatabaseURI = os.Getenv("DB_URI")
	MainDbName = os.Getenv("DB_NAME")
	DropPendingUpdates = typeConvertor{str: os.Getenv("DROP_PENDING_UPDATES")}.Bool()
	OwnerId = typeConvertor{str: os.Getenv("OWNER_ID")}.Int64()
	MessageDump = typeConvertor{str: os.Getenv("MESSAGE_DUMP")}.Int64()
	BotToken = os.Getenv("BOT_TOKEN")

	// Only perform these checks if we are not running tests
	if flag.Lookup("test.v") == nil {
		if BotToken == "" {
			log.Fatal("BOT_TOKEN environment variable is not set.")
		}
		if OwnerId == 0 {
			log.Fatal("OWNER_ID environment variable is not set or invalid.")
		}
		if DatabaseURI == "" {
			log.Fatal("DB_URI environment variable is not set.")
		}
	}

	AllowedUpdates = typeConvertor{str: os.Getenv("ALLOWED_UPDATES")}.StringArray()
	if len(AllowedUpdates) == 0 || (len(AllowedUpdates) == 1 && AllowedUpdates[0] == "") {
		AllowedUpdates = defaultAllowedUpdates
	}

	ValidLangCodes = typeConvertor{str: os.Getenv("ENABLED_LOCALES")}.StringArray()
	if len(ValidLangCodes) == 0 || (len(ValidLangCodes) == 1 && ValidLangCodes[0] == "") {
		ValidLangCodes = defaultValidLangCodes
	}

	ApiServer = os.Getenv("API_SERVER")
	// set as default api server if not set
	if ApiServer == "" {
		ApiServer = "https://api.telegram.org"
	}
	// set default db_name
	if MainDbName == "" {
		MainDbName = "Alita_Robot"
	}

	// redis config
	RedisAddress = os.Getenv("REDIS_ADDRESS")
	if os.Getenv("REDIS_ADDRESS") == "" {
		RedisAddress = "localhost:6379"
		log.Warn("REDIS_ADDRESS not set, defaulting to localhost:6379")
	}
	RedisPassword = os.Getenv("REDIS_PASSWORD")
	if os.Getenv("REDIS_PASSWORD") == "" {
		RedisPassword = ""
	}
	RedisDB = typeConvertor{str: os.Getenv("REDIS_DB")}.Int()
	if os.Getenv("REDIS_DB") == "" {
		RedisDB = 0
	}

	MongoMaxPoolSize = parseUint64Env("MONGO_MAX_POOL_SIZE", 100)
	MongoMinPoolSize = parseUint64Env("MONGO_MIN_POOL_SIZE", 10)
	MongoMaxConnIdleTime = parseDurationEnv("MONGO_MAX_CONN_IDLE_TIME", 30*time.Second)
	MongoMaxIdleTime = parseDurationEnv("MONGO_MAX_IDLE_TIME", 30*time.Second)

	// Resource monitoring thresholds
	HighGoroutineThreshold = typeConvertor{str: os.Getenv("HIGH_GOROUTINE_THRESHOLD")}.Int()
	if HighGoroutineThreshold == 0 {
		HighGoroutineThreshold = 1000
	}
	HighMemoryThresholdMB = typeConvertor{str: os.Getenv("HIGH_MEMORY_THRESHOLD_MB")}.Int()
	if HighMemoryThresholdMB == 0 {
		HighMemoryThresholdMB = 500
	}
}
