package controllers

import (
	"context"
	"golang-restaurant-management/database"
	"golang-restaurant-management/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tableCollection *mongo.Collection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var table models.Table

		err := c.BindJSON(&table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		result, err := tableCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		var alltables []bson.M
		err = result.All(ctx, &alltables)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, alltables[0])

	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {

		var table models.Table
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		tableId := c.Param("table_id")

		err := tableCollection.FindOne(ctx, bson.M{"table_id": tableId}).Decode(&table)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, table)

	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {

		var table models.Table

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := c.BindJSON(&table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		validationErr := validate.Struct(table)
		if validationErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
		}

		table.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		table.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		table.ID = primitive.NewObjectID()
		table.Table_id = table.ID.Hex()

		result, inserterr := tableCollection.InsertOne(ctx, table)
		if inserterr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": inserterr.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {

		var table models.Table

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		err := c.BindJSON(&table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		tableId := c.Param("table_id")

		filter := bson.M{"table_id": tableId}
		var UpdateObj primitive.D

		if table.Number_of_guests != nil {
			UpdateObj = append(UpdateObj, bson.E{"updated_at", table.Number_of_guests})
		}

		table.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		UpdateObj = append(UpdateObj, bson.E{"updated_at", table.Updated_at})

		Upsert := true

		opt := options.UpdateOptions{
			Upsert: &Upsert,
		}
		result, err := tableCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{"$set", UpdateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, result)

	}
}
