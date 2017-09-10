package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mopsalarm/go-pr0gramm-tags/store"
	log "github.com/sirupsen/logrus"
)

type tagInfo struct {
	Id     int    `db:"id"`
	ItemId int    `db:"item_id"`
	Tag    string `db:"tag"`
}

func queryTags(db *sqlx.DB, firstTagId, count int, consumer func(tagInfo)) error {
	var tagInfos []tagInfo
	err := db.Select(&tagInfos,
		"SELECT id, item_id, lower(tag) as tag FROM tags WHERE id >= $1 ORDER BY id ASC LIMIT $2",
		firstTagId, count)

	if err == nil {
		for _, tagInfo := range tagInfos {
			consumer(tagInfo)
		}
	}

	return err
}

type postInfo struct {
	Id            int       `db:"id"`
	Updated       time.Time `db:"updated"`
	Flags         int       `db:"flags"`
	Score         int       `db:"score"`
	CreatedEpoch  int       `db:"created"`
	Promoted      bool      `db:"promoted"`
	Username      string    `db:"username"`
	UserMark      int       `db:"mark"`
	HasText       bool      `db:"has_text"`
	HasAudio      bool      `db:"audio"`
	Width         int       `db:"width"`
	Controversial bool      `db:"is_controversial"`
}

func queryItems(db *sqlx.DB, updatedAfter time.Time, days, itemCount int, consumer func(postInfo)) error {
	var postInfos []postInfo

	err := db.Select(&postInfos, `
		SELECT
			items.id,
			items.updated,
			items.flags,
			items.created,
			items.audio,
			items.width,
			items.mark,
			items.up - items.down as score,
			items.promoted != 0 as promoted,
			lower(items.username) AS username,
			COALESCE(texts.has_text, FALSE) AS has_text,
			up>60 AND down>60 AND least(up, down)::float/greatest(up, down)>=0.7 as is_controversial
		FROM
			items
			LEFT JOIN items_text texts ON (items.id = texts.item_id)
		WHERE items.updated >= $1 ORDER BY items.updated ASC LIMIT $2`, updatedAfter, itemCount)

	if err == nil {
		for _, postInfo := range postInfos {
			consumer(postInfo)
		}
	}

	return err
}

func sizeCategories(width int) []string {
	switch {
	case width > 3800:
		return []string{"2160p", "4k"}

	case width > 1900:
		return []string{"1080p", "hd"}

	case width > 1200:
		return []string{"720p", "hd"}

	case width > 600:
		return []string{"sd"}

	default:
		return []string{"kartoffel"}
	}
}

func userMarkToString(mark int) string {
	switch mark {
	case 6:
		return "ftb"
	case 1:
		return "newfag"
	default:
		return ""
	}
}

func FetchUpdates(db *sqlx.DB, state store.StoreState) (store.IterStore, store.StoreState, bool) {
	builder := store.NewStoreBuilder(HashWord)

	itemCount := 10000
	{
		days := 3
		err := queryItems(db, state.LastItemUpdateTime, days, itemCount, func(postInfo postInfo) {
			itemId := int32(-postInfo.Id)

			// Prefixes currently in-use:
			//  d: date
			//  f: flags
			//  s: score
			//  u: user
			//  q: quality
			//  m: mark (ftb, newfag)

			builder.Push("u:"+CleanString(postInfo.Username), itemId)

			switch {
			case postInfo.Flags&1 != 0:
				builder.Push("f:sfw", itemId)
			case postInfo.Flags&2 != 0:
				builder.Push("f:nsfw", itemId)
			case postInfo.Flags&4 != 0:
				builder.Push("f:nsfl", itemId)
			case postInfo.Flags&8 != 0:
				builder.Push("f:nsfp", itemId)
			}

			if postInfo.Promoted {
				builder.Push("f:top", itemId)
			}

			if postInfo.HasText {
				builder.Push("f:text", itemId)
			}

			if postInfo.HasAudio {
				builder.Push("f:sound", itemId)
			}

			if postInfo.Controversial {
				builder.Push("f:controversial", itemId)
			}

			// mark content of ftp and newfags special.
			if mark := userMarkToString(postInfo.UserMark); mark != "" {
				builder.Push("m:"+mark, itemId)
			}

			// add quality-tag
			for _, sizeCategory := range sizeCategories(postInfo.Width) {
				builder.Push("q:"+sizeCategory, itemId)
			}

			// date tags
			created := time.Unix(int64(postInfo.CreatedEpoch), 0)
			builder.Push(fmt.Sprintf("d:%04d", created.Year()), itemId)
			builder.Push(fmt.Sprintf("d:%04d:%02d", created.Year(), created.Month()), itemId)

			// sort posts into bins (size 100) by score.
			// a post with score 1125 will be put into bins 100, 200,... and 1100
			for bin := 1; bin <= postInfo.Score/100; bin++ {
				label := fmt.Sprintf("s:%d", (100 * bin))
				builder.Push(label, itemId)
			}

			// add a label for the real shitty content.
			if postInfo.Score < -300 {
				builder.Push("s:shit", itemId)
			}

			itemCount -= 1
			state.LastItemUpdateTime = postInfo.Updated
		})

		if err != nil {
			log.WithError(err).Warn("Could not fetch the list of post items")
			metricsUpdaterError.Inc(1)
		}
	}

	tagCount := 50000
	{
		err := queryTags(db, state.LastTagId, tagCount, func(info tagInfo) {
			itemId := int32(-info.ItemId)
			for _, word := range ExtractWords(info.Tag) {
				builder.Push(word, itemId)
			}

			if strings.ToLower(info.Tag) == "repost" {
				builder.Push("f:repost", itemId)
			}

			tagCount -= 1
			state.LastTagId = info.Id
		})

		if err != nil {
			log.WithError(err).Warn("Error while streaming from postgres")
			metricsUpdaterError.Inc(1)
		}
	}

	expectMore := tagCount == 0 || itemCount == 0
	return builder.Build(), state, expectMore
}
