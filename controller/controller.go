package controller

import (
	"context"
	"fmt"
	"github.com/AgilAghamirzayev/simplebank/entity"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

type RecipesController struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewRecipesController(ctx context.Context, collection *mongo.Collection) *RecipesController {
	return &RecipesController{
		collection: collection,
		ctx:        ctx,
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
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while insertin a new recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (controller *RecipesController) GetAllRecipes(c *gin.Context) {
	cur, err := controller.collection.Find(controller.ctx, bson.M{})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer cur.Close(controller.ctx)
	recipes := make([]entity.Recipe, 0)

	for cur.Next(controller.ctx) {
		var recipe entity.Recipe
		cur.Decode(&recipe)
		recipes = append(recipes, recipe)
	}

	c.JSON(http.StatusOK, recipes)
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
		bson.D{{"$set",
			bson.D{
				{"name", recipe.Name},
				{"instructions", recipe.Instructions},
				{"ingredients", recipe.Ingredients},
				{"tags", recipe.Tags},
			}}})

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recipe has, been, updated, "})
}

func (controller *RecipesController) DeleteRecipeById(c *gin.Context) {
	id := c.Param("id")
	objectId, _ := primitive.ObjectIDFromHex(id)

	_, err := controller.collection.DeleteOne(controller.ctx, bson.M{"_id": objectId})

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been deleted"})
}

func (controller *RecipesController) SearchRecipesByTags(c *gin.Context) {
	tag := c.Query("tag")

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

	c.JSON(http.StatusOK, recipes)
}

func (controller *RecipesController) GetRecipeById(c *gin.Context) {
	id := c.Param("id")
	objectId, _ := primitive.ObjectIDFromHex(id)
	cur, err := controller.collection.Find(controller.ctx, bson.M{"objectId": objectId})

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}

	c.JSON(http.StatusOK, cur)
}
