package helpers

import (
	"context"
	"fmt"
	"golang-restaurant-management/database"
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type signedDetails struct {
	Email      string
	First_name string
	Last_name  string
	User_id    string
	jwt.StandardClaims
}

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

var SECRETKEY string = os.Getenv("SECRETKEY")

func GenerateAllTokens(email string, first_name string, last_name string, user_id string) (signedToken string, signedRefreshToken string, err error) {
	Claims := &signedDetails{
		Email:      email,
		First_name: first_name,
		Last_name:  last_name,
		User_id:    user_id,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(30)).Unix(),
		},
	}
	refreshedClaims := &signedDetails{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(30)).Unix(),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims).SignedString([]byte(SECRETKEY))
	if err != nil {
		log.Panic(err)
	}
	refreshed_token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshedClaims).SignedString([]byte(SECRETKEY))
	if err != nil {
		log.Panic(err)
	}

	return token, refreshed_token, err
}

func UpdateAllToken(signedToken string, signedrefreshToken string, userId string) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

	var updateObj primitive.D

	updateObj = append(updateObj, bson.E{"token", signedToken})
	updateObj = append(updateObj, bson.E{"refreshed_token", signedrefreshToken})
	Updated_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	updateObj = append(updateObj, bson.E{"updated_at", Updated_at})

	filter := bson.M{"user_id": userId}

	upsert := true

	opt := options.UpdateOptions{
		Upsert: &upsert,
	}

	_, err := userCollection.UpdateOne(
		ctx,
		filter,
		bson.D{{"$set", updateObj}},
		&opt,
	)

	if err != nil {
		log.Panic(err)
		return
	}
	defer cancel()
	return

}

func ValidateAllToken(signedToken string) (claims *signedDetails, msg string) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		signedDetails{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRETKEY), nil
		},
	)

	claims, ok := token.Claims.(*signedDetails)
	if !ok {
		msg = fmt.Sprint("invlaid Token")
		msg = err.Error()
		return
	}
	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = fmt.Sprint("Token is Expired")
		msg = err.Error()
		return
	}
	return claims, msg
}
