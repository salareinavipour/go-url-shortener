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
}

func StoreColdUrl(shortUrl string, originalUrl string, userId string) {
	db, err := sql.Open("sqlite3", "/cold-urls.db")

	if err != nil {
		panic(fmt.Sprintf("Failed saving cold url in sqlite | Error: %v - shortUrl: %s - originalUrl: %s\n", err, shortUrl, originalUrl))
	}
	// defer close
	defer db.Close()

	stmt, _ := db.Prepare("INSERT INTO urls (id, short_url, origina_url) VALUES (?, ?, ?)")
	stmt.Exec(nil, userId, shortUrl, originalUrl)
	defer stmt.Close()
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
	return result
}
