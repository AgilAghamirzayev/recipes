package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AgilAghamirzayev/simplebank/controller"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
)

var (
	ctx               context.Context
	client            *mongo.Client
	collection        *mongo.Collection
	redisClient       *redis.Client
	recipesController *controller.RecipesController
)

func init() {
	ctx = context.Background()

	initMongoDB()
	initRedis()

	recipesController = controller.NewRecipesController(ctx, collection, redisClient)
}

func initMongoDB() {
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	log.Println("Connected to MongoDB")

	collection = client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")

	//recipes, err := loadRecipesFromFile("recipes.json")
	//if err != nil {
	//	log.Fatalf("Failed to load recipes: %v", err)
	//}
	//
	//populateRecipes(recipes)
}

func loadRecipesFromFile(filename string) ([]interface{}, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var recipes []interface{}
	if err := json.Unmarshal(file, &recipes); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return recipes, nil
}

func populateRecipes(recipes []interface{}) {
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
		log.Println("Recipes collection already contains data, skipping insertion.")
	}
}

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	status, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	log.Println("Redis is up and running:", status)
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
