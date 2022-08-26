package main

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"os"
	"time"
)

type (
	Users struct {
		UserID    int       `json:"user_id" gorm:"primaryKey;autoIncrement"`
		Username  string    `json:"username" gorm:"unique"`
		Password  string    `json:"password"`
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		Phone     string    `json:"phone" gorm:"unique"`
		Email     string    `json:"email" gorm:"unique"`
		Birthday  time.Time `json:"birthday"`
		IsActive  bool      `json:"is_active" gorm:"default:false"`
	}
	UserRequest struct {
		Username  string `json:"username" validate:"required"`
		Password  string `json:"password" validate:"required"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone" validate:"required"`
		Email     string `json:"email" validate:"required,email"`
		Birthday  string `json:"birthday" validate:"omitempty,datetime=2006-01-02"`
	}
	UserEditRequest struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
		Email     string `json:"email" validate:"omitempty,email"`
		Birthday  string `json:"birthday" validate:"omitempty,datetime=2006-01-02"`
	}

	CustomValidator struct {
		validator *validator.Validate
	}
)

func main() {
	godotenv.Load(".env")
	e := echo.New()
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&Users{})

	e.Validator = &CustomValidator{validator: validator.New()}

	e.GET("/users", func(c echo.Context) error {
		var res []Users
		err := db.Find(&res).Error
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	})
	e.GET("/users/:id", func(c echo.Context) error {
		var res Users
		userID := c.Param("id")
		err := db.Model(&Users{}).First(&res, userID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNoContent, "")
			}
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	})
	e.POST("/users", func(c echo.Context) error {
		var request UserRequest
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		if err := c.Validate(&request); err != nil {
			return err
		}
		crypted, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		birthday, err := time.Parse("2006-01-02", request.Birthday)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		newData := Users{
			Username:  request.Username,
			Password:  string(crypted),
			FirstName: request.FirstName,
			LastName:  request.LastName,
			Phone:     request.Phone,
			Email:     request.Email,
			Birthday:  birthday,
		}
		errCreated := db.Create(&newData).Error
		if errCreated != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusCreated, newData)
	})
	e.PATCH("/users/:id", func(c echo.Context) error {
		var request UserEditRequest
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		var old Users
		userID := c.Param("id")
		err := db.Model(&Users{}).First(&old, userID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNoContent, "")
			}
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		if request.Email != "" {
			old.Email = request.Email
		}
		if request.Password != "" {
			crypted, _ := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
			old.Password = string(crypted)
		}
		if request.FirstName != "" {
			old.FirstName = request.FirstName
		}
		if request.LastName != "" {
			old.LastName = request.LastName
		}
		if request.Username != "" {
			old.Username = request.Username
		}
		if request.Birthday != "" {
			birthday, _ := time.Parse("2006-01-02", request.Birthday)
			old.Birthday = birthday
		}
		if request.Phone != "" {
			old.Phone = request.Phone
		}

		errUpdate := db.Updates(&old).Error
		if errUpdate != nil {
			return c.JSON(http.StatusInternalServerError, errUpdate.Error)
		}
		return c.JSON(http.StatusOK, old)
	})
	e.DELETE("/users/:id", func(c echo.Context) error {
		var res Users
		userID := c.Param("id")
		err := db.Model(&Users{}).First(&res, userID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNoContent, "")
			}
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		errDel := db.Delete(&res)
		if errDel != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, res)
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}
	return nil
}
