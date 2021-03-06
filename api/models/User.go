package models

import (
	"bytes"
	"errors"
	"html"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/badoux/checkmail"
	"github.com/globalsign/mgo/bson"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

//Model of the user table in database
type User struct {
	ID             uint32  `gorm:"primary_key; auto_increment" json:"id"`
	Username       string  `gorm:"size:255;not null;unique" json:"username"`
	Email          string  `gorm:"size:100;not null;unique" json:"email"`
	Phone          string  `gorm:"size:25;not null;unique" json:"phone_number"`
	ImageURL       string  `gorm:"size:255;unique" json:"image_url"`
	Specialisation string  `gorm:"size:255;not null" json:"specialisation"`
	Latitude       float32 `gorm:"size:255;not null" json:"latitude"`
	Longitude      float32 `gorm:"size:255;not null" json:"longitude"`
	Address        string  `gorm:"size:255;not null" json:"address"`
	Region         string  `gorm:"size:255;not null" json:"region"`
	Country        string  `gorm:"size:255;not null" json:"country"`
	//Review         []Review  `json:"reviews"`
	Password  string    `gorm:"size:100;not null" json:"password"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

//Model the response of user-related endpoints
type ResponseUser struct {
	ID             uint32  `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone_number"`
	ImageURL       string  `json:"image_url"`
	Specialisation string  `json:"specialisation"`
	Latitude       float32 `json:"latitude"`
	Longitude      float32 `json:"longitude"`
	Address        string  `json:"address"`
	Region         string  `json:"region"`
	Country        string  `json:"country"`
	Token          string  `json:"token"`
	//Review         Review  `json:"reviews"`
}

//Encrypt password
func Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

//Hash password before saving to db
func (u *User) BeforeSave() error {
	hashedPassword, err := Hash(u.Password)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)
	return nil
}

//User input formatting
func (u *User) Prepare() {
	u.ID = 0
	u.Username = html.EscapeString(strings.TrimSpace(u.Username))
	u.Email = html.EscapeString(strings.TrimSpace(u.Email))
	u.Phone = html.EscapeString(strings.TrimSpace(u.Phone))
	u.ImageURL = html.EscapeString(strings.TrimSpace(u.ImageURL))
	u.Specialisation = html.EscapeString(strings.TrimSpace(u.Specialisation))
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
}

//User input validation
func (u *User) Validate(action string) error {
	switch strings.ToLower(action) {
	case "update":
		if u.Username == "" {
			return errors.New("Required Username")
		}
		if u.Password == "" {
			return errors.New("Required Password")
		}
		if u.Phone == "" {
			return errors.New("Required Phone Number")
		}
		if u.Email == "" {
			return errors.New("Required Email")
		}
		if err := checkmail.ValidateFormat(u.Email); err != nil {
			return errors.New("Invalid Email")
		}
		if u.Specialisation == "" {
			return errors.New("Required Specialisation")
		}
		// if u.Latitude == 0 {
		// 	return errors.New("Required Location")
		// }
		// if u.Longitude == 0 {
		// 	return errors.New("Required Longitude")
		// }

		return nil
	case "login":
		if u.Password == "" {
			return errors.New("Required Password")
		}
		if u.Email == "" {
			return errors.New("Required Email")
		}
		if err := checkmail.ValidateFormat(u.Email); err != nil {
			return errors.New("Invalid Email")
		}
		return nil

	default:
		if u.Username == "" {
			return errors.New("Required Username")
		}
		if u.Password == "" {
			return errors.New("Required Password")
		}
		if u.Phone == "" {
			return errors.New("Required Phone Number")
		}
		if u.Email == "" {
			return errors.New("Required Email")
		}
		if err := checkmail.ValidateFormat(u.Email); err != nil {
			return errors.New("Invalid Email")
		}
		return nil
	}
}

//Save user to database
func (u *User) SaveUser(db *gorm.DB) (*User, error) {
	var err error
	err = db.Debug().Create(&u).Error
	if err != nil {
		return &User{}, err
	}
	return u, nil
}

//Get all users
func (u *User) FindAllUsers(db *gorm.DB) (*[]User, error) {
	var err error

	users := []User{}

	err = db.Debug().Model(&User{}).Order("created_at desc").Limit(100).Find(&users).Error
	if err != nil {
		return &[]User{}, err
	}

	// if len(users) > 0 {
	// 	for i, _ := range users {
	// 		err := db.Debug().Model(&Review{}).Where("user_id=?", users[i].ID).Take(&users[i].Review).Error
	// 		if err != nil {
	// 			return &[]User{}, err
	// 		}
	// 	}
	// }

	return &users, err
}

//Find user based on id
func (u *User) FindUserByID(db *gorm.DB, uid uint32) (*User, error) {
	var err error
	err = db.Debug().Model(User{}).Where("id = ?", uid).Take(&u).Error
	if err != nil {
		return &User{}, err
	}
	if gorm.IsRecordNotFoundError(err) {
		return &User{}, errors.New("User Not Found")
	}
	return u, err
}

//Update user details
func (u *User) UpdateAUser(db *gorm.DB, uid uint32) (*User, error) {

	// To hash the password
	err := u.BeforeSave()
	if err != nil {
		log.Fatal(err)
	}

	db = db.Debug().Model(&User{}).Where("id = ?", uid).Take(&User{}).UpdateColumns(
		map[string]interface{}{
			"username":       u.Username,
			"email":          u.Email,
			"phone":          u.Phone,
			"image_url":      u.ImageURL,
			"specialisation": u.Specialisation,
			"latitude":       u.Latitude,
			"longitude":      u.Longitude,
			"address":        u.Address,
			"region":         u.Region,
			"country":        u.Country,
			"password":       u.Password,
			"updated_at":     time.Now(),
		},
	)
	if db.Error != nil {
		return &User{}, db.Error
	}
	// This is the display the updated user
	err = db.Debug().Model(&User{}).Where("id = ?", uid).Take(&u).Error
	if err != nil {
		return &User{}, err
	}
	return u, nil
}

//Delete user account using id
func (u *User) DeleteAUser(db *gorm.DB, uid uint32) (int64, error) {

	db = db.Debug().Model(&User{}).Where("id = ?", uid).Take(&User{}).Delete(&User{})

	if db.Error != nil {
		return 0, db.Error
	}
	return db.RowsAffected, nil
}

func UploadProfilePicToS3(path string, s *session.Session, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	urlLink := "https://vickikbt-fixit-app.s3.us-east-2.amazonaws.com/"

	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)

	// create a unique file name for the file
	tempFileName := path + "/" + bson.NewObjectId().Hex() + filepath.Ext(fileHeader.Filename)

	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String("vickikbt-fixit-app"), //Bucket name
		Key:                  aws.String(tempFileName),         //File name
		ACL:                  aws.String("public-read"),        // Access type- public
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(int64(size)),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
		StorageClass:         aws.String("INTELLIGENT_TIERING"),
	})
	if err != nil {
		return "", err
	}

	return urlLink + tempFileName, err
}
