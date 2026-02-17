package utils

import (
	"os"
	"taskmanager/constants"
	"taskmanager/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtSecret = []byte(func() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	return "supersecretkey"
}())

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

func GetRecursiveReportIDs(managerID uint, db *gorm.DB) []uint {
	var reportIDs []uint
	var users []models.User

	// Find direct reports
	db.Where("manager_id = ?", managerID).Find(&users)

	for _, user := range users {
		reportIDs = append(reportIDs, user.ID)
		// Recursively find reports of reports
		reportIDs = append(reportIDs, GetRecursiveReportIDs(user.ID, db)...)
	}

	return reportIDs
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

		if task.CreatedByID == userID || task.AssignedToID == userID {
			return true
		}

		reportIDs := GetRecursiveReportIDs(userID, db)
		for _, reportID := range reportIDs {
			if task.CreatedByID == reportID || task.AssignedToID == reportID {
				return true
			}
		}
	}

	return false
}

func CanAssignTask(
	assignerID uint,
	assignerRole string,
	assigneeID uint,
	db *gorm.DB,
) (bool, error) {
	if assigneeID == 0 {
		return assignerRole == constants.RoleAdmin || assignerRole == constants.RoleManager, nil
	}

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
		// Manager can assign to themselves
		if assignerID == assigneeID {
			return true, nil
		}

		// Or to anyone in their recursive hierarchy
		reportIDs := GetRecursiveReportIDs(assignerID, db)
		for _, reportID := range reportIDs {
			if assigneeID == reportID {
				return true, nil
			}
		}
		return false, nil
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
