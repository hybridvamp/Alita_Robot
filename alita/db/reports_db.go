package db

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/divideprojects/Alita_Robot/alita/utils/cache"
	"github.com/divideprojects/Alita_Robot/alita/utils/string_handling"
	"github.com/eko/gocache/lib/v4/store"

	"go.mongodb.org/mongo-driver/bson"
)

type ChatReportSettings struct {
	ChatId      int64   `bson:"_id,omitempty" json:"_id,omitempty"`
	Status      bool    `bson:"status,omitempty" json:"status,omitempty"`
	BlockedList []int64 `bson:"blocked_list,omitempty" json:"blocked_list,omitempty"`
}

type UserReportSettings struct {
	UserId int64 `bson:"_id,omitempty" json:"_id,omitempty"`
	Status bool  `bson:"status,omitempty" json:"status,omitempty"`
}

func GetChatReportSettings(chatID int64) (reportsrc *ChatReportSettings) {
	// Try cache first
	if cached, err := cache.Marshal.Get(cache.Context, chatID, new(ChatReportSettings)); err == nil && cached != nil {
		reportsrc = cached.(*ChatReportSettings)
		return
	}

	reportsrc = &ChatReportSettings{}

	filter := bson.M{"_id": chatID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":          chatID,
			"status":       true,
			"blocked_list": make([]int64, 0),
		},
	}

	err := findOneAndUpsert(reportChatColl, filter, update, reportsrc)
	if err != nil {
		// Fallback to default values in case of error
		reportsrc = &ChatReportSettings{
			ChatId:      chatID,
			Status:      true,
			BlockedList: make([]int64, 0),
		}
		log.Error(err)
	}
	// Cache the result
	_ = cache.Marshal.Set(cache.Context, chatID, reportsrc, store.WithExpiration(10*time.Minute))
	return
}

func SetChatReportStatus(chatID int64, pref bool) {
	reportsUpdate := GetChatReportSettings(chatID)
	reportsUpdate.Status = pref
	err := updateOne(reportChatColl, bson.M{"_id": chatID}, reportsUpdate)
	if err != nil {
		log.Error(err)
	}
	// Update cache
	_ = cache.Marshal.Set(cache.Context, chatID, reportsUpdate, store.WithExpiration(10*time.Minute))
}

func BlockReportUser(chatId, userId int64) {
	reportsUpdate := GetChatReportSettings(chatId)

	if string_handling.FindInInt64Slice(reportsUpdate.BlockedList, userId) {
		return
	}

	reportsUpdate.BlockedList = append(reportsUpdate.BlockedList, userId)
	err := updateOne(reportChatColl, bson.M{"_id": chatId}, reportsUpdate)
	if err != nil {
		log.Error(err)
	}
	// Update cache
	_ = cache.Marshal.Set(cache.Context, chatId, reportsUpdate, store.WithExpiration(10*time.Minute))
}

func UnblockReportUser(chatId, userId int64) {
	reportsUpdate := GetChatReportSettings(chatId)

	if !string_handling.FindInInt64Slice(reportsUpdate.BlockedList, userId) {
		return
	}

	reportsUpdate.BlockedList = string_handling.RemoveFromInt64Slice(reportsUpdate.BlockedList, userId)
	err := updateOne(reportChatColl, bson.M{"_id": chatId}, reportsUpdate)
	if err != nil {
		log.Error(err)
	}
}

/* user settings below */

func GetUserReportSettings(userId int64) (reportsrc *UserReportSettings) {
	reportsrc = &UserReportSettings{}

	filter := bson.M{"_id": userId}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":    userId,
			"status": true,
		},
	}

	err := findOneAndUpsert(reportUserColl, filter, update, reportsrc)
	if err != nil {
		// Fallback to default values in case of error
		reportsrc = &UserReportSettings{
			UserId: userId,
			Status: true,
		}
		log.Error(err)
	}

	return
}

func SetUserReportSettings(chatID int64, pref bool) {
	reportsUpdate := &ChatReportSettings{
		ChatId: chatID,
		Status: pref,
	}
	err := updateOne(reportUserColl, bson.M{"_id": chatID}, reportsUpdate)
	if err != nil {
		log.Error(chatID)
	}
}

func LoadReportStats() (uRCount, gRCount int64) {
	uRCount, acErr := countDocs(
		reportUserColl,
		bson.M{"status": true},
	)
	if acErr != nil {
		log.Errorf("[Database] loadStats: %v", acErr)
	}
	gRCount, clErr := countDocs(
		reportChatColl,
		bson.M{"status": true},
	)
	if clErr != nil {
		log.Errorf("[Database] loadStats: %v", clErr)
	}
	return
}
