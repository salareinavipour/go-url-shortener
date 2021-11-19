package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type StorageService struct {
	redisClient map[string]*redis.Client
}

var (
	storeService = &StorageService{}
)

const CacheDuration = 24 * time.Hour

func InitializeStore() *StorageService {
	redisClient := make(map[string]*redis.Client)

	redisClient["urls"] = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	redisClient["dates"] = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	redisClient["counts"] = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       2,
	})

	for _, client := range redisClient {
		pong, err := client.Ping().Result()

		if err != nil {
			panic(fmt.Sprintf("Error init Redis: %v", err))
		}

		fmt.Printf("\nRedis started successfully: pong message = {%s}", pong)
	}

	storeService.redisClient = redisClient
	return storeService
}

func SaveUrlMapping(shortUrl string, originalUrl string) {
	err := storeService.redisClient["urls"].Set(shortUrl, originalUrl, CacheDuration).Err()

	if err != nil {
		panic(fmt.Sprintf("Failed saving key url | Error: %v - shortUrl: %s - originalUrl: %s\n", err, shortUrl, originalUrl))
	}

	err = storeService.redisClient["dates"].Set(shortUrl, time.Now().String(), CacheDuration).Err()

	if err != nil {
		panic(fmt.Sprintf("Failed saving date of url | Error: %v - shortUrl: %s - date: %s\n", err, shortUrl, time.Now().String()))
	}

	err = storeService.redisClient["counts"].Set(shortUrl, 0, CacheDuration).Err()

	if err != nil {
		panic(fmt.Sprintf("Failed saving count of url usage | Error: %v - shortUrl: %s\n", err, shortUrl))
	}
}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

func StoreColdUrl(shortUrl string, originalUrl string, userId string) {

	url, err := storeService.redisClient["dates"].Get(shortUrl).Result()

	if err != nil {
		panic(fmt.Sprintf("Failed saving date of url | Error: %v - shortUrl: %s - date: %s\n", err, shortUrl, time.Now().String()))
	}

	layout := "2006-01-02T15:04:05.000Z"
	t, err := time.Parse(layout, url)

	if err != nil {
		fmt.Println(err)
	}

	if t.Before(time.Now().Add(-time.Hour * 12)) {
		db, err := sql.Open("sqlite3", "/cold-urls.db")

		if err != nil {
			panic(fmt.Sprintf("Failed saving cold url in sqlite | Error: %v - shortUrl: %s - originalUrl: %s\n", err, shortUrl, originalUrl))
		}
		// defer close
		defer db.Close()

		stmt, _ := db.Prepare("INSERT INTO urls (id, short_url, origina_url) VALUES (?, ?, ?)")
		stmt.Exec(nil, userId, shortUrl, originalUrl)
		defer stmt.Close()

		err = storeService.redisClient["urls"].Del(shortUrl).Err()

		if err != nil {
			panic(fmt.Sprintf("Failed saving date of url | Error: %v - shortUrl: %s - date: %s\n", err, shortUrl, time.Now().String()))
		}
	}
}

func RetrieveColdUrl(shortUrl string) string {
	db, err := sql.Open("sqlite3", "/cold-urls.db")

	if err != nil {
		panic(fmt.Sprintf("Failed retrieving cold url from sqlite | Error: %v - shortUrl: %s\n", err, shortUrl))
	}

	row, err := db.Query("SELECT origina_url FROM urls WHERE short_url like " + shortUrl)

	if err != nil {
		panic(fmt.Sprintf("Failed retrieving cold url from sqlite | Error: %v - shortUrl: %s\n", err, shortUrl))
	}

	defer db.Close()

	var originalUrl string
	err = row.Scan(&originalUrl)
	if err != nil {
		panic(fmt.Sprintf("Failed retrieving cold url from sqlite | Error: %v - shortUrl: %s\n", err, shortUrl))
	}

	return originalUrl
}

func RetrieveInitialUrl(shortUrl string) string {
	result, err := storeService.redisClient["urls"].Get(shortUrl).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed RetrieveInitialUrl url | Error: %v - shortUrl: %s\n", err, shortUrl))
	}

	err = storeService.redisClient["counts"].Incr(shortUrl).Err()
	if err != nil {
		panic(fmt.Sprintf("Failed to increase url usage count | Error: %v - shortUrl: %s\n", err, shortUrl))
	}
	return result
}
