package controllers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	database "github.com/Faanilo/Golang-auth/config"
	helper "github.com/Faanilo/Golang-auth/helpers"
	"github.com/Faanilo/Golang-auth/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

var validate = validator.New()

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""
	if err != nil {
		msg = "Email or password  incorrect"
		check = false
	}
	return check, msg
}
func Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		countEmail, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		defer cancel()
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the email"})
		}
		hashedPassword, err := HashPassword(*user.Password)
		if err != nil {
			log.Println("Error hashing password:", err)
		}
		user.Password = &hashedPassword
		countPhone, err := userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		defer cancel()
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the phone"})
		}
		if countEmail > 0 {
			if countPhone > 0 {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "both email and phone number are already in use"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "this email is already in use"})
			}
		} else if countPhone > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this phone number is already in use"})
		}
		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC822))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()
		token, refreshToken := helper.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_Name, *user.User_type, user.User_id)
		user.Token = &token
		user.Refresh_token = &refreshToken
		resultInsertNumber, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			msg := "User item was not created"
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		defer cancel()
		c.JSON(http.StatusAccepted, resultInsertNumber)
	}
}
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		var FindUser models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&FindUser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or password incorrect"})
			return
		}
		passwordIsvalid, msg := VerifyPassword(*user.Password, *FindUser.Password)
		defer cancel()
		if !passwordIsvalid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		if FindUser.Email == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		}
		token, refreshToken := helper.GenerateAllTokens(*FindUser.Email, *FindUser.First_name, *FindUser.Last_Name, *FindUser.User_type, FindUser.User_id)
		helper.UpdateAllTokens(token, refreshToken, FindUser.User_id)
		err = userCollection.FindOne(ctx, bson.M{"user_id": FindUser.User_id}).Decode(&FindUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, FindUser)
	}
}
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := helper.CheckUserType(c, "ADMIN")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage

		matchStage := bson.D{{Key: "$match", Value: bson.D{}}}
		groupStage := bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "null"},
				{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
			}},
		}
		projectStage := bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "total_count", Value: 1},
				{Key: "user_items", Value: bson.D{{Key: "$slice", Value: bson.A{"$data", startIndex, recordPerPage}}}},
			}},
		}

		pipeline := mongo.Pipeline{matchStage, groupStage, projectStage}
		result, err := userCollection.Aggregate(ctx, pipeline)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing user items"})
			return
		}

		var allusers []bson.M
		if err := result.All(ctx, &allusers); err != nil {
			log.Fatal(err)
			return
		}

		if len(allusers) > 0 {
			c.JSON(http.StatusOK, allusers[0])
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "No users found"})
		}
	}
}

func GetUserById() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")
		if err := helper.MatchUserTypeToUid(c, userId); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, user)
	}
}

func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")
		if err := helper.MatchUserTypeToUid(c, userId); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		err := userCollection.FindOneAndDelete(ctx, bson.M{"user_id": userId}).Decode(&user)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "User Deleted",
			"data":    user})
	}
}
