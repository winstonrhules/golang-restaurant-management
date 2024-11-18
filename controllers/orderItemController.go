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

type orderitemPack struct {
	Table_id   *string
	orderItems []models.OrderItem
}

var orderitemCollection *mongo.Collection = database.OpenCollection(database.Client, "orderitem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderCollection.Find(context.TODO(), bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		var orderitem []bson.M

		err = result.All(ctx, &orderitem)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var orderitem models.OrderItem

		orderitemId := c.Param("orderitem_id")

		err := orderitemCollection.FindOne(ctx, bson.M{"orderitem_id": orderitemId}).Decode(&orderitem)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, orderitem)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("orderitem_id")
		allOrderItems, err := ItemsByOrder(orderId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusOK, allOrderItems)
	}
}

func ItemsByOrder(id string) (orderItems []primitive.M, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

	matchStage := bson.D{{"$match", bson.D{{"order_id", id}}}}
	lookupStage := bson.D{{"$lookup", bson.D{{"from", "food"}, {"localField", "food_id"}, {"foreignField", "food_id"}, {"as", "food"}}}}
	unwindStage := bson.D{{"$unwind", bson.D{{"path", "$food"}, {"preserveNullAndEmptyArrays", true}}}}
	lookupOrderStage := bson.D{{"$lookup", bson.D{{"from", "order"}, {"localField", "order_id"}, {"foreignField", "order_id"}, {"as", "order"}}}}
	unwindOrderStage := bson.D{{"$unwind", bson.D{{"path", "$order"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupTableStage := bson.D{{"$lookup", bson.D{{"from", "table"}, {"localField", "order.table_id"}, {"foreignField", "table_id"}, {"as", "table"}}}}
	unwindTableStage := bson.D{{"$unwind", bson.D{{"path", "$table"}, {"preserveNullAndEmptyArrays", true}}}}
	projectStage := bson.D{{"project", bson.D{
		{"id", 0},
		{"amount", "$food.price"},
		{"total_count", 1},
		{"food_name", "$food.name"},
		{"food_image", "$food.food_image"},
		{"table_number", "$table.table_number"},
		{"number_of_guest", "$table.number_of_guest"},
		{"table_id", "$table.table_id"},
		{"order_id", "$order.order_id"},
		{"price", "$food.price"},
		{"quantity", 1},
	}}}
	groupStage := bson.D{{"group", bson.D{{"_id", bson.D{{"order_id", "$order_id"}, {"table_id", "$table_id"}, {"table_number", "$table_number"}}}, {"payment_due", bson.D{{"$sum", "$amount"}}}, {"$total_count", bson.D{{"$sum", 1}}}, {"order_items", bson.D{{"$push", "$$ROOT"}}}}}}
	projectStage2 := bson.D{{"project", bson.D{
		{"id", 0},
		{"total_count", 1},
		{"table_number", "$_id.table_number"},
		{"order_items", 1},
	}}}
	results, err := orderitemCollection.Aggregate(ctx, mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage,
		groupStage,
		projectStage2,
	})
	if err != nil {
		panic(err)
	}
	if err := results.All(ctx, &orderItems); err != nil {
		panic(err)
	}
	defer cancel()
	return orderItems, err
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var orderitemPack orderitemPack
		var order models.Order

		err := c.BindJSON(&orderitemPack)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		order.Order_date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		orderItemsTobeInserted := []interface{}{}
		order.Table_id = orderitemPack.Table_id
		order_id := OrderItemOrderCreator(order)

		for _, orderItem := range orderitemPack.orderItems {
			orderItem.Order_id = &order_id

			validationErr := validate.Struct(orderItem)
			if validationErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
			}
			orderItem.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.ID = primitive.NewObjectID()
			orderItem.Order_item_id = orderItem.ID.Hex()
			var num = tofixed(*orderItem.Unit_price, 2)
			orderItem.Unit_price = &num
			orderItemsTobeInserted = append(orderItemsTobeInserted, orderItem)
		}
		result, err := orderitemCollection.InsertMany(ctx, orderItemsTobeInserted)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var orderitem models.OrderItem

		orderitemId := c.Param("orderitem_id")

		err := c.BindJSON(&orderitem)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		var updateObj primitive.D

		if orderitem.Quantity != nil {
			updateObj = append(updateObj, bson.E{"quantity", orderitem.Quantity})
		}

		if orderitem.Unit_price != nil {
			updateObj = append(updateObj, bson.E{"unit_price", orderitem.Unit_price})
		}

		if orderitem.Food_id != nil {
			updateObj = append(updateObj, bson.E{"oderitem_foodid", orderitem.Food_id})
		}

		orderitem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"update_at", orderitem.Updated_at})

		upsert := true

		filter := bson.M{"orderitem_id": orderitemId}

		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := orderitemCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{"$set", updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}
