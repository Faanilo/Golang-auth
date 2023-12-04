package helper

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	database "github.com/Faanilo/Golang-auth/config"
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RegisterDetails struct {
	Email      string
	First_name string
	Last_Name  string
	Uid        string
	User_type  string
	jwt.StandardClaims
}

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var SECRET_KEY string = os.Getenv("SECRET_KEY")

func GenerateAllTokens(email string, firstName string, lastName string, userType string, uid string) (signedToken string, signedRefreshToken string) {
	accessClaims := jwt.MapClaims{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"uid":        uid,
		"user_type":  userType,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
	}
	// Create claims for refresh token
	refreshClaims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}
	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	token, err := accessToken.SignedString([]byte(SECRET_KEY))
	if err != nil {
		log.Panic(err)
	}
	// Generate refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(SECRET_KEY))
	if err != nil {
		log.Panic(err)
	}

	return token, refreshTokenString
}

func UpdateAllTokens(registerToken string, registerRefreshToken string, userId string) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	var updateObj primitive.D
	updateObj = append(updateObj, bson.E{Key: "token", Value: registerToken})
	updateObj = append(updateObj, bson.E{Key: "refresh_token", Value: registerRefreshToken})

	Updated_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	updateObj = append(updateObj, bson.E{Key: "updated_at", Value: Updated_at})
	upsert := true
	filter := bson.M{"user_id": userId}
	opt := options.UpdateOptions{
		Upsert: &upsert,
	}
	_, err := userCollection.UpdateOne(
		ctx,
		filter,
		bson.D{
			{Key: "$set", Value: updateObj},
		},
		&opt,
	)
	defer cancel()
	if err != nil {
		log.Panic(err)
		return
	}
}
func ValidateToken(signedToken string) (claims *RegisterDetails, msg string) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&RegisterDetails{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRET_KEY), nil
		},
	)
	if err != nil {
		msg = err.Error()
		return
	}
	claims, ok := token.Claims.(*RegisterDetails)
	if !ok {
		msg = fmt.Sprintf("the token is invalid")
		msg = err.Error()
		return
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = fmt.Sprintf("token is expired")
		msg = err.Error()
		return
	}
	return claims, msg

}
