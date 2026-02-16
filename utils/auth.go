package utils

import (
	"taskmanager/constants"
	"taskmanager/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtSecret = []byte("supersecretkey")

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)
	return err == nil
}

func GenerateJWT(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(jwtSecret)
}

func CanAccessTask(
	task models.Task,
	userID uint,
	role string,
	db *gorm.DB,
) bool {

	if role == constants.RoleAdmin {
		return true
	}

	if role == constants.RoleMember {
		return task.CreatedByID == userID ||
			task.AssignedToID == userID
	}

	if role == constants.RoleManager {

		if task.CreatedByID == userID {
			return true
		}

		var member models.User
		err := db.
			Where("id = ? AND manager_id = ?",
				task.AssignedToID,
				userID,
			).
			First(&member).Error

		return err == nil
	}

	return false
}

func CanAssignTask(
	assignerID uint,
	assignerRole string,
	assigneeID uint,
	db *gorm.DB,
) (bool, error) {
	// Admin can assign to anyone
	if assignerRole == constants.RoleAdmin {
		return true, nil
	}

	// Fetch assignee role to check if they are admin
	var assignee models.User
	if err := db.First(&assignee, assigneeID).Error; err != nil {
		return false, err
	}

	// No non-admin can assign to an admin
	if assignee.Role == constants.RoleAdmin {
		return false, nil
	}

	if assignerRole == constants.RoleManager {
		// Manager can assign to their direct reports OR themselves
		if assignerID == assigneeID {
			return true, nil
		}
		return assignee.ManagerID != nil && *assignee.ManagerID == assignerID, nil
	}

	if assignerRole == constants.RoleMember {
		// Members can only assign to themselves
		return assignerID == assigneeID, nil
	}

	return false, nil
}

func JwtSecret() []byte {
	return jwtSecret
}
