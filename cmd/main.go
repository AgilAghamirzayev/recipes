package main

import (
	"context"
	"encoding/json"
	"github.com/AgilAghamirzayev/simplebank/controller"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
)

var ctx context.Context
var client *mongo.Client
var collection *mongo.Collection

var recipesController *controller.RecipesController

func init() {
	file, err := os.ReadFile("recipes.json")
	if err != nil {
		log.Fatalf("Failed to read recipes.json: %v", err)
	}

	var recipes []interface{}
	if err := json.Unmarshal(file, &recipes); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	ctx = context.Background()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	log.Println("Connected to MongoDB")

	collection = client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	recipesController = controller.NewRecipesController(ctx, collection)

	count, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		log.Fatalf("Failed to count documents: %v", err)
	}

	if count == 0 {
		insertManyResult, err := collection.InsertMany(ctx, recipes)
		if err != nil {
			log.Fatalf("Failed to insert recipes: %v", err)
		}
		log.Printf("Inserted %d recipes\n", len(insertManyResult.InsertedIDs))
	} else {
		log.Println("Recipes collection already has data, skipping insertion.")
	}
}

func main() {
	router := gin.Default()
	router.POST("/recipes", recipesController.CreateRecipe)
	router.GET("/recipes", recipesController.GetAllRecipes)
	router.PUT("/recipes/:id", recipesController.UpdateRecipeById)
	router.DELETE("/recipes/:id", recipesController.DeleteRecipeById)
	router.GET("/recipes/:id", recipesController.GetRecipeById)
	router.GET("/recipes/search", recipesController.SearchRecipesByTags)

	router.Run()
}
