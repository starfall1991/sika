package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
	UserId  string `json:"user_id"`
}

type User struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Addresses   []Address `json:"addresses" gorm:"foreignKey:UserId"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	file, err := os.Open("users_data.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)
	dec := json.NewDecoder(strings.NewReader(string(byteValue)))
	_, err = dec.Token()
	if err != nil {
		log.Fatal(err)
	}

	userChan := make(chan User)
	const maxWorkers = 10
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range userChan {
				user := User{
					Id:          u.Id,
					Name:        u.Name,
					Email:       u.Email,
					PhoneNumber: u.PhoneNumber,
				}
				tx := db.Begin()

				userCreate := tx.Create(&user)
				if userCreate.Error != nil {
					tx.Rollback()
					log.Printf("Error creating user id %s : %v", user.Id, userCreate.Error)
					continue
				}
				for _, address := range u.Addresses {
					address.UserId = u.Id
					addCreate := tx.Create(&address)
					if addCreate.Error != nil {
						tx.Rollback()
						log.Printf("Error creating address for user %s: %v\n", user.Id, err)
						break
					}
				}
				tx.Commit()
			}
		}()
	}

	for dec.More() {
		var u User
		err = dec.Decode(&u)
		if err != nil {
			log.Fatal(err)
		}
		userChan <- u
	}

	close(userChan)
	wg.Wait()
}
