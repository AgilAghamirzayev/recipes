package controller

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/AgilAghamirzayev/simplebank/entity"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecipesController struct {
	collection  *mongo.Collection
	ctx         context.Context
	redisClient *redis.Client
}

func NewRecipesController(ctx context.Context, collection *mongo.Collection, redisClient *redis.Client) *RecipesController {
	return &RecipesController{
		collection:  collection,
		ctx:         ctx,
		redisClient: redisClient,
	}
}

func (controller *RecipesController) CreateRecipe(c *gin.Context) {
	var recipe entity.Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipe.ID = primitive.NewObjectID()
	recipe.PublishedAt = time.Now()
	_, err := controller.collection.InsertOne(controller.ctx, recipe)

	if err != nil {
		log.Println("MongoDB Insert Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while inserting a new recipe"})
		return
	}

	log.Println("Cache invalidated: Removing 'recipes' key from Redis")
	controller.redisClient.Del(controller.ctx, "recipes")

	c.JSON(http.StatusOK, recipe)
}

func (controller *RecipesController) GetAllRecipes(c *gin.Context) {
	val, err := controller.redisClient.Get(controller.ctx, "recipes").Result()
	if errors.Is(err, redis.Nil) {
		log.Println("Fetching recipes from MongoDB")
		cur, err := controller.collection.Find(controller.ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cur.Close(controller.ctx)

		var recipes []entity.Recipe
		if err := cur.All(controller.ctx, &recipes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		data, _ := json.Marshal(recipes)
		controller.redisClient.Set(controller.ctx, "recipes", string(data), 10*time.Minute)

		c.JSON(http.StatusOK, recipes)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	} else {
		log.Println("Fetching recipes from Redis")
		var recipes []entity.Recipe
		json.Unmarshal([]byte(val), &recipes)
		c.JSON(http.StatusOK, recipes)
	}
}

func (controller *RecipesController) GetRecipeById(c *gin.Context) {
	id := c.Param("id")
	cacheKey := "recipe:" + id

	val, err := controller.redisClient.Get(controller.ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		log.Println("Fetching recipe from MongoDB")
		objectId, _ := primitive.ObjectIDFromHex(id)
		var recipe entity.Recipe
		err := controller.collection.FindOne(controller.ctx, bson.M{"_id": objectId}).Decode(&recipe)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
			return
		}

		data, _ := json.Marshal(recipe)
		controller.redisClient.Set(controller.ctx, cacheKey, string(data), 10*time.Minute)

		c.JSON(http.StatusOK, recipe)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	} else {
		log.Println("Fetching recipe from Redis")
		var recipe entity.Recipe
		json.Unmarshal([]byte(val), &recipe)
		c.JSON(http.StatusOK, recipe)
	}
}

func (controller *RecipesController) UpdateRecipeById(c *gin.Context) {
	id := c.Param("id")
	var recipe entity.Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	objectId, _ := primitive.ObjectIDFromHex(id)
	_, err := controller.collection.UpdateOne(controller.ctx,
		bson.M{"_id": objectId},
		bson.D{{"$set", bson.D{
			{"name", recipe.Name},
			{"instructions", recipe.Instructions},
			{"ingredients", recipe.Ingredients},
			{"tags", recipe.Tags},
		}}})

	if err != nil {
		log.Println("MongoDB Update Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Cache invalidated: Removing 'recipes' and 'recipe:" + id + "' from Redis")
	controller.redisClient.Del(controller.ctx, "recipes", "recipe:"+id)

	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been updated"})
}

func (controller *RecipesController) DeleteRecipeById(c *gin.Context) {
	id := c.Param("id")
	objectId, _ := primitive.ObjectIDFromHex(id)

	_, err := controller.collection.DeleteOne(controller.ctx, bson.M{"_id": objectId})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}

	log.Println("Cache invalidated: Removing 'recipes' and 'recipe:" + id + "' from Redis")
	controller.redisClient.Del(controller.ctx, "recipes", "recipe:"+id)

	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been deleted"})
}

func (controller *RecipesController) SearchRecipesByTags(c *gin.Context) {
	tag := c.Query("tag")
	cacheKey := "recipes:tag:" + tag

	val, err := controller.redisClient.Get(controller.ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		log.Println("Fetching recipes by tag from MongoDB")
		filter := bson.M{"tags": bson.M{"$in": []string{tag}}}

		var recipes []entity.Recipe
		cursor, err := controller.collection.Find(controller.ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching recipes"})
			return
		}
		defer cursor.Close(controller.ctx)

		if err := cursor.All(controller.ctx, &recipes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding recipes"})
			return
		}

		data, _ := json.Marshal(recipes)
		controller.redisClient.Set(controller.ctx, cacheKey, string(data), 10*time.Minute)

		c.JSON(http.StatusOK, recipes)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	} else {
		log.Println("Fetching recipes by tag from Redis")
		var recipes []entity.Recipe
		json.Unmarshal([]byte(val), &recipes)
		c.JSON(http.StatusOK, recipes)
	}
}
