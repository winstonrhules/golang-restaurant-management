package controllers

import (
	"context"
	"fmt"
	"golang-restaurant-management/database"
	"golang-restaurant-management/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvoiceViewFormat struct {
	invoice_id       string
	payment_method   string
	order_id         string
	payment_status   string
	table_number     interface{}
	payment_due      interface{}
	payment_due_date time.Time
	order_details    interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var invoice models.Invoice

		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		result, inserterr := invoiceCollection.Find(context.TODO(), bson.M{})
		if inserterr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": inserterr.Error()})
		}

		var allinvoice []bson.M

		err := result.All(ctx, &allinvoice)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		defer cancel()
		c.JSON(http.StatusOK, allinvoice[0])
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var invoice models.Invoice

		invoiceId := c.Param("invoice_id")

		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)

		if err != nil {
			defer cancel()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		var invoiceView InvoiceViewFormat

		allOrderItems, err := ItemsByOrder(*invoice.Order_id)
		if err != nil {
			log.Fatal(err)
		}
		invoiceView.order_id = *invoice.Order_id
		invoiceView.payment_due_date = invoice.Payment_due_date

		invoiceView.payment_method = "null"
		if invoice.Payment_method != nil {
			invoiceView.payment_method = *invoice.Payment_method
		}
		invoiceView.invoice_id = invoice.Invoice_id
		invoiceView.payment_status = *invoice.Payment_status
		invoiceView.table_number = allOrderItems[0]["table_number"]
		invoiceView.payment_due = allOrderItems[0]["payment_due"]
		invoiceView.order_details = allOrderItems[0]["order_details"]

		defer cancel()
		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var invoice models.Invoice
		var order models.Order

		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		validateErr := validate.Struct(invoice)
		if validateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": validateErr.Error()})
		}

		if invoice.Order_id != nil {
			err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
			if err != nil {
				msg := fmt.Sprintln("not able to fetch order id")
				c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			}
		}

		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
		}

		invoice.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Payment_due_date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		result, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		if insertErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": insertErr.Error()})
		}
		defer cancel()
		c.JSON(http.StatusOK, result)

	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var invoice models.Invoice
		var order models.Order

		invoiceId := c.Param("invoice_id")

		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		var UpdateInv primitive.D

		if invoice.Payment_method != nil {
			UpdateInv = append(UpdateInv, bson.E{"payment_method", invoice.Payment_method})
		}

		if invoice.Payment_status != nil {
			UpdateInv = append(UpdateInv, bson.E{"payment_status", invoice.Payment_status})
		}

		if invoice.Order_id != nil {
			err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.Order_id}).Decode(&order)
			if err != nil {
				msg := fmt.Sprintln("not able to fetch order id")
				c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			}
			UpdateInv = append(UpdateInv, bson.E{"order_id", invoice.Order_id})
		}

		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		UpdateInv = append(UpdateInv, bson.E{"update_at", invoice.Updated_at})

		Upsert := true

		filter := bson.M{"invoice_id": invoiceId}

		opt := options.UpdateOptions{
			Upsert: &Upsert,
		}

		status := "PENDING"
		if invoice.Payment_status == nil {
			invoice.Payment_status = &status
		}

		result, err := invoiceCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{"$set", UpdateInv}},
			&opt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		defer cancel()
		c.JSON(http.StatusOK, result)

	}
}
