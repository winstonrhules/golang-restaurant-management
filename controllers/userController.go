package controllers

import (
	"context"
	"fmt"
	"golang-restaurant-management/database"
	"golang-restaurant-management/helpers"
	"golang-restaurant-management/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		recordeperPage, err := strconv.Atoi(c.Query("recordperPage"))
		if err != nil || recordeperPage > 1 {
			recordeperPage = 10
		}

		Page, err := strconv.Atoi(c.Query("Page"))
		if err != nil || Page > 1 {
			Page = 1
		}

		startIndex, err := strconv.Atoi(c.Query("startIndex"))
		if err != nil {
			startIndex = (Page - 1) * recordeperPage
		}

		matchStage := bson.D{{"$match", bson.D{{}}}}
		groupStage := bson.D{{"$group", bson.D{{"_id", bson.D{{"_id", "null"}}}, {"total_count", 1}, {"$sum", 1}, {"$data", bson.D{{"$push", "$$ROOT"}}}}}}
		projectStage := bson.D{{"project", bson.D{
			{"_id", 0},
			{"total_count", 1},
			{"user_items", bson.D{{"$slice", []interface{}{"$data", startIndex, recordeperPage}}}},
		}}}

		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage,
			groupStage,
			projectStage,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		var Allusers []bson.M
		err = result.All(ctx, &Allusers)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, Allusers[0])
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		var user models.User

		userId := c.Param("user_id")

		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, user)

	}
}

func Signup() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
		}

		Password := HashPassword(*user.Password)
		user.Password = &Password

		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			log.Panic(err)
			msg := fmt.Sprintf("Error occured while checking Email")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		count, err = userCollection.CountDocuments(ctx, bson.M{"Phone": user.Phone})
		if err != nil {
			log.Panic(err)
			msg := fmt.Sprintf("Error occured while checking Phone")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if count > 0 {
			msg := fmt.Sprintf("Error occured while checking Phone")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		}
		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		refresh_token, token, _ := helpers.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_name, *&user.User_id)
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()
		user.Token = &token
		user.Refresh_token = &refresh_token

		result, err := userCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		var user models.User
		var founduser models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&founduser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		passwordisValid, msg := VerifyPassword(*user.Password, *founduser.Password)
		msg = fmt.Sprintf("Password invalid")
		if passwordisValid != true {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		}

		token, refresh_token, _ := helpers.GenerateAllTokens(*founduser.Email, *founduser.First_name, *founduser.Last_name, *&founduser.User_id)
		helpers.UpdateAllToken(token, refresh_token, founduser.User_id)

		c.JSON(http.StatusOK, founduser)

	}
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""
	if err != nil {
		msg = fmt.Sprint("Password Invalid")
		check = false
	}
	return check, msg
}

func HashPassword(userPassword string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(userPassword), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}
