package main

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	jwtSecret = []byte("test-secret")
}

func TestPasswordHash(t *testing.T) {
	password := "mypassword123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal("hash error:", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		t.Fatal("compare should succeed:", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte("wrong")); err == nil {
		t.Fatal("compare should fail for wrong password")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	uid := primitive.NewObjectID()
	tokenStr, err := generateToken(uid, "testuser")
	if err != nil {
		t.Fatal("generate error:", err)
	}

	claims, err := parseToken(tokenStr)
	if err != nil {
		t.Fatal("parse error:", err)
	}

	if claims.UserID != uid.Hex() {
		t.Fatalf("expected user_id %s, got %s", uid.Hex(), claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Fatalf("expected username testuser, got %s", claims.Username)
	}
	if claims.ID == "" {
		t.Fatal("jti should not be empty")
	}
}

func TestExpiredToken(t *testing.T) {
	now := time.Now().Add(-2 * time.Hour)
	claims := Claims{
		UserID:   primitive.NewObjectID().Hex(),
		Username: "expired",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "test-jti",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)), // expired 1h ago
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(jwtSecret)

	_, err := parseToken(tokenStr)
	if err == nil {
		t.Fatal("expired token should fail parsing")
	}
}

func TestPredictForestEmpty(t *testing.T) {
	result := predictForest(nil, [4]float64{1, 2, 3, 4})
	if result != 0 {
		t.Fatalf("expected 0 for empty forest, got %f", result)
	}
}

func TestPredictTreeLeaf(t *testing.T) {
	tree := Tree{
		Nodes: []Node{
			{IsLeaf: true, Value: 42.5},
		},
	}
	result := predictTree(tree, [4]float64{1, 2, 3, 4})
	if result != 42.5 {
		t.Fatalf("expected 42.5, got %f", result)
	}
}

func TestPredictTreeSplit(t *testing.T) {
	// Root splits on feature 0, threshold 5.0
	// Left child (<=5): leaf with value 10
	// Right child (>5): leaf with value 20
	tree := Tree{
		Nodes: []Node{
			{IsLeaf: false, Feature: 0, Threshold: 5.0, Left: 1, Right: 2},
			{IsLeaf: true, Value: 10.0},
			{IsLeaf: true, Value: 20.0},
		},
	}

	// feature[0] = 3.0 <= 5.0 → left → 10
	r1 := predictTree(tree, [4]float64{3.0, 0, 0, 0})
	if r1 != 10.0 {
		t.Fatalf("expected 10.0, got %f", r1)
	}

	// feature[0] = 7.0 > 5.0 → right → 20
	r2 := predictTree(tree, [4]float64{7.0, 0, 0, 0})
	if r2 != 20.0 {
		t.Fatalf("expected 20.0, got %f", r2)
	}
}

func TestCalculateMetricsEmpty(t *testing.T) {
	mae, rmse, r2 := calculateMetrics(nil, nil)
	if mae != 0 || rmse != 0 || r2 != 0 {
		t.Fatalf("expected all zeros, got mae=%f rmse=%f r2=%f", mae, rmse, r2)
	}
}
