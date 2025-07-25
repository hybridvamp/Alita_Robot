package db

import (
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/divideprojects/Alita_Robot/alita/utils/cache"
	"github.com/eko/gocache/lib/v4/store"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

// GetLanguage returns the language code for the current chat or user context.
// For private chats, it returns the user's language; for groups, it returns the group's language.
// Defaults to "en" if not set.
func GetLanguage(ctx *ext.Context) string {
	var chat gotgbot.Chat
	if ctx.CallbackQuery != nil {
		chat = ctx.CallbackQuery.Message.GetChat()
	} else {
		chat = ctx.Update.Message.Chat
	}
	if chat.Type == "private" {
		user := ctx.EffectiveSender.User
		return getUserLanguage(user.Id)
	}
	return getGroupLanguage(chat.Id)
}

func getGroupLanguage(GroupID int64) string {
	// Try cache first
	if cached, err := cache.Marshal.Get(cache.Context, GroupID, new(string)); err == nil && cached != nil {
		return *(cached.(*string))
	}
	groupc := GetChatSettings(GroupID)
	lang := "en"
	if groupc.Language != "" {
		lang = groupc.Language
	}
	_ = cache.Marshal.Set(cache.Context, GroupID, &lang, store.WithExpiration(10*time.Minute))
	return lang
}

func getUserLanguage(UserID int64) string {
	// Try cache first
	if cached, err := cache.Marshal.Get(cache.Context, UserID, new(string)); err == nil && cached != nil {
		return *(cached.(*string))
	}
	userc := checkUserInfo(UserID)
	lang := "en"
	if userc != nil && userc.Language != "" {
		lang = userc.Language
	}
	_ = cache.Marshal.Set(cache.Context, UserID, &lang, store.WithExpiration(10*time.Minute))
	return lang
}

// InvalidateUserLanguageCache clears the cached language for a user
func InvalidateUserLanguageCache(UserID int64) error {
	err := cache.Marshal.Delete(cache.Context, UserID)
	if err != nil {
		log.WithFields(log.Fields{
			"userId": UserID,
			"error":  err,
		}).Error("InvalidateUserLanguageCache: Failed to delete user language cache")
		return err
	}
	log.WithFields(log.Fields{
		"userId": UserID,
	}).Debug("InvalidateUserLanguageCache: Successfully invalidated user language cache")
	return nil
}

// InvalidateGroupLanguageCache clears the cached language for a group
func InvalidateGroupLanguageCache(GroupID int64) error {
	err := cache.Marshal.Delete(cache.Context, GroupID)
	if err != nil {
		log.WithFields(log.Fields{
			"groupId": GroupID,
			"error":   err,
		}).Error("InvalidateGroupLanguageCache: Failed to delete group language cache")
		return err
	}
	log.WithFields(log.Fields{
		"groupId": GroupID,
	}).Debug("InvalidateGroupLanguageCache: Successfully invalidated group language cache")
	return nil
}

// ChangeUserLanguage sets the language code for a specific user.
// No update is performed if the user does not exist or already has the target language.
func ChangeUserLanguage(UserID int64, lang string) {
	userc := checkUserInfo(UserID)
	if userc == nil {
		return
	} else if userc.Language == lang {
		return
	}
	userc.Language = lang // change user language
	err := updateOne(userColl, bson.M{"_id": UserID}, userc)
	if err != nil {
		log.Errorf("[Database] ChangeUserLanguage: %v - %d", err, UserID)
		return
	}
	// Invalidate old cache and update with new value
	_ = InvalidateUserLanguageCache(UserID)
	_ = cache.Marshal.Set(cache.Context, UserID, &lang, store.WithExpiration(10*time.Minute))
	log.Infof("[Database] ChangeUserLanguage: %d", UserID)
}

// ChangeGroupLanguage sets the language code for a specific group chat.
// No update is performed if the group already has the target language.
func ChangeGroupLanguage(GroupID int64, lang string) {
	groupc := GetChatSettings(GroupID)
	if groupc.Language == lang {
		return
	}
	groupc.Language = lang
	err := updateOne(chatColl, bson.M{"_id": GroupID}, groupc)
	if err != nil {
		log.Errorf("[Database] ChangeGroupLanguage: %v - %d", err, GroupID)
		return
	}
	// Invalidate old cache and update with new value
	_ = InvalidateGroupLanguageCache(GroupID)
	_ = cache.Marshal.Set(cache.Context, GroupID, &lang, store.WithExpiration(10*time.Minute))
	log.Infof("[Database] ChangeGroupLanguage: %d", GroupID)
}
