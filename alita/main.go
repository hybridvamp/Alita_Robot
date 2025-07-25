package alita

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/divideprojects/Alita_Robot/alita/config"
	log "github.com/sirupsen/logrus"

	"github.com/divideprojects/Alita_Robot/alita/db"
	"github.com/divideprojects/Alita_Robot/alita/modules"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/divideprojects/Alita_Robot/alita/utils/string_handling"
)

// ResourceMonitor periodically logs system resource usage such as goroutine count and memory usage.
// It warns if thresholds are exceeded, helping to detect leaks or abnormal resource consumption.
func ResourceMonitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		numGoroutines := runtime.NumGoroutine()

		// Log metrics
		log.WithFields(log.Fields{
			"goroutines": numGoroutines,
			"memory_mb":  m.Alloc / 1024 / 1024,
			"sys_mb":     m.Sys / 1024 / 1024,
			"gc_runs":    m.NumGC,
		}).Info("Resource usage stats")

		// Warning thresholds
		if numGoroutines > config.HighGoroutineThreshold {
			log.WithField("goroutines", numGoroutines).Warn("High goroutine count detected")
		}

		if int(m.Alloc/1024/1024) > config.HighMemoryThresholdMB {
			log.WithField("memory_mb", m.Alloc/1024/1024).Warn("High memory usage detected")
		}
	}
}

// ListModules returns a formatted string listing all modules loaded in the bot.
// It retrieves module names from the help module, sorts them, and joins them for display.
func ListModules() string {
	modSlice := modules.HelpModule.AbleMap.LoadModules()
	sort.Strings(modSlice)
	return fmt.Sprintf("[%s]", strings.Join(modSlice, ", "))
}

// InitialChecks performs startup checks and background initializations before running the bot.
// It ensures the bot is present in the database, checks for duplicate command aliases,
// and starts resource monitoring. Cache and database initialization is now handled by lifecycle manager.
func InitialChecks(b *gotgbot.Bot) {
	// Create bot in db if not already created
	go db.EnsureBotInDb(b)
	checkDuplicateAliases()

	// Start resource monitoring
	go ResourceMonitor()
}

// checkDuplicateAliases checks for duplicate command aliases in the help module.
// If a duplicate is found, the bot logs an error and exits gracefully.
func checkDuplicateAliases() {
	var althelp []string

	for _, i := range modules.HelpModule.AltHelpOptions {
		for _, alias := range i {
			althelp = append(althelp, strings.ToLower(alias))
		}
	}

	duplicateAlias, val := string_handling.IsDuplicateInStringSlice(althelp)
	if val {
		log.Errorf("Found duplicate alias: %s", duplicateAlias)
		// Exit gracefully instead of using log.Fatalf
		// This allows for proper cleanup by the lifecycle manager
		panic(fmt.Sprintf("Found duplicate alias: %s", duplicateAlias))
	}
}

// LoadModules loads all bot modules into the dispatcher.
// Modules are loaded in a specific order, with the help module loaded last to ensure all commands are registered.
func LoadModules(dispatcher *ext.Dispatcher) {
	// Initialize Inner Map
	modules.HelpModule.AbleMap.Init()

	// Load this at last because it loads all the modules
	defer modules.LoadHelp(dispatcher)

	modules.LoadBotUpdates(dispatcher)
	modules.LoadAntispam(dispatcher)
	modules.LoadLanguage(dispatcher)
	modules.LoadAdmin(dispatcher)
	modules.LoadPin(dispatcher)
	modules.LoadMisc(dispatcher)
	modules.LoadBans(dispatcher)
	modules.LoadMutes(dispatcher)
	modules.LoadPurges(dispatcher)
	modules.LoadUsers(dispatcher)
	modules.LoadReports(dispatcher)
	modules.LoadDev(dispatcher)
	modules.LoadLocks(dispatcher)
	modules.LoadFilters(dispatcher)
	modules.LoadAntiflood(dispatcher)
	modules.LoadNotes(dispatcher)
	modules.LoadConnections(dispatcher)
	modules.LoadDisabling(dispatcher)
	modules.LoadRules(dispatcher)
	modules.LoadWarns(dispatcher)
	modules.LoadCaptcha(dispatcher)
	modules.LoadGreetings(dispatcher)
	modules.LoadBlacklists(dispatcher)
	modules.LoadMkdCmd(dispatcher)
}

// StartCaptchaScheduler initializes and starts the CAPTCHA scheduler
// DEPRECATED: This function is deprecated. Scheduler is now managed by lifecycle manager.
func StartCaptchaScheduler(_ *gotgbot.Bot) {
	log.Warn("StartCaptchaScheduler is deprecated. Scheduler is now managed by lifecycle manager.")
	// Keep for backwards compatibility but don't start scheduler
	// scheduler.StartCaptchaScheduler(bot)
}
