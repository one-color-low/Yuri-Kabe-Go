package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"io/ioutil"

	"io"
	"os"

	"fmt"

	"math/rand"
	"time"

	"github.com/oklog/ulid"

	"archive/zip"
	"path/filepath"
	"strings"
)

// グローバル変数なDB
var DB *gorm.DB

type DeleteResponse struct {
	Id string `json:"id"`
}

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
}

// 汎用関数

func createID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	return id.String()
}

func unzip(zipFilePath string, outputPath string) {
	dst := outputPath
	archive, err := zip.OpenReader(zipFilePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			fmt.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}
		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}

func createSession(userID string) Session {
	var session Session
	session.ID = createID()
	session.UserID = userID
	session.CreatedAt = time.Now()

	DB.Create(&session)

	return session

}

// ----------- Room Model --------------
type Room struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Author           string `json:"author"`
	Description      string `json:"description`
	Authorized_Users string `json:"authorized_users`
}

// Get All Rooms
func getRooms(w http.ResponseWriter, r *http.Request) {

	log.Println("rooms api")

	var rooms []Room
	DB.Find(&rooms) //見つけた結果をroomsに入れよ

	responseBody, err := json.Marshal(rooms)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write((responseBody))
}

// Get Single Room
func getRoom(w http.ResponseWriter, r *http.Request) {

	var room Room
	params := mux.Vars(r)

	DB.First(&room, "id = ?", params["id"])

	responseBody, err := json.Marshal(room)

	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// Create a Room
func createRoom(w http.ResponseWriter, r *http.Request) {

	// Create ID
	var id string = createID()

	// Save file
	file, fileHeader, err := r.FormFile("file_input")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	uploadedFileName := fileHeader.Filename
	log.Println(uploadedFileName)

	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(fmt.Sprintf("./uploads/%s.zip", id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// unzip
	unzip(fmt.Sprintf("./uploads/%s.zip", id), "output")

	// Save info
	info := r.FormValue("info_input")

	var room Room

	if err := json.Unmarshal([]byte(info), &room); err != nil {
		log.Fatal(err)
	}

	// todo: 認証実装し次第、room.Author はgoogle_subから検索したuserのidにするように
	room.ID = id

	DB.Create(&room)

	responseBody, err := json.Marshal(room)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write((responseBody))
}

// Update a Room
func updateRoom(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]
	reqBody, _ := ioutil.ReadAll(r.Body)

	var room Room
	if err := json.Unmarshal(reqBody, &room); err != nil {
		log.Fatal(err)
	}

	DB.Model(&room).Where("id = ?", id).Updates(
		map[string]interface{}{
			"title":  room.Title,
			"author": room.Author,
		})

	responseBody, err := json.Marshal(room)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// Delete a Room
func deleteRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	DB.Where("id = ?", id).Delete(&Room{})

	responseBody, err := json.Marshal(DeleteResponse{Id: id})
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// ----------- User Model --------------
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GoogleSub string `json:"google_sub"`
}

// Get All Users
func getUsers(w http.ResponseWriter, r *http.Request) {

	var users []User
	DB.Find(&users) //見つけた結果をusersに入れよ

	responseBody, err := json.Marshal(users)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write((responseBody))
}

// Get Single User
func getUser(w http.ResponseWriter, r *http.Request) {

	var user User
	params := mux.Vars(r)
	DB.First(&user, params["id"])

	responseBody, err := json.Marshal(user)

	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// Create a User
func createUser(w http.ResponseWriter, r *http.Request) {

	reqBody, _ := ioutil.ReadAll(r.Body)

	var user User

	// reqBodyがUserの構造体フォーマットになっていない場合はエラーを返す
	if err := json.Unmarshal(reqBody, &user); err != nil {
		log.Fatal(err)
	}

	DB.Create(&user)

	responseBody, err := json.Marshal(user)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write((responseBody))
}

// Update a User
func updateUser(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]
	reqBody, _ := ioutil.ReadAll(r.Body)

	var user User
	if err := json.Unmarshal(reqBody, &user); err != nil {
		log.Fatal(err)
	}

	DB.Model(&user).Where("id = ?", id).Updates(
		map[string]interface{}{
			"name":       user.Name,
			"google_sub": user.GoogleSub,
		})

	responseBody, err := json.Marshal(user)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// Delete a User
func deleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	DB.Where("id = ?", id).Delete(&User{})

	responseBody, err := json.Marshal(DeleteResponse{Id: id})
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// Sign Up

// Sign In
func signIn(w http.ResponseWriter, r *http.Request) {

	log.Println("signIn trying")
	// 1. rからtokenを取得

	// 2. tokenのvalidation (not validの場合はサインイン画面にリダイレクト)

	// 3. tokenからgoogle_subを取得

	// ※ 1~3簡略化: rからgoogle_sub取得
	params := mux.Vars(r)
	google_sub := params["google_sub"]

	// google_subでUserテーブルを検索し、特定のUser構造体を取得
	var user User
	DB.First(&user, google_sub)

	// session生成
	session := createSession(user.ID)

	// cookieを生成
	cookie := http.Cookie{
		Name:     "_cookie",
		Value:    session.ID,
		HttpOnly: true,
	}

	// cookieをクライアントに返す
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", 302)

	// else{ http.Redirect(w, r, "signIn", 302) }
}

func main() {

	// DB初期化
	db, err := gorm.Open(sqlite.Open("yuri-kabe.db"), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	DB = db

	var mock_user_id string = createID()
	var mock_session_id string = createID()

	// Create "rooms" table
	DB.AutoMigrate(&Room{})

	// Create Mock Data
	DB.Create(&Room{
		ID:               createID(),
		Title:            "Room one",
		Author:           mock_user_id,
		Description:      "This is mock data.",
		Authorized_Users: "'1', '2', '3'",
	})

	// Create "users" table
	DB.AutoMigrate(&User{})

	// Create Mock Data
	DB.Create(&User{
		ID:        mock_user_id,
		Name:      "Yuta",
		GoogleSub: "xxx",
	})

	// Create "session" table
	DB.AutoMigrate(&Session{})

	// Create Mock Data
	DB.Create(&Session{
		ID:        mock_session_id,
		UserID:    mock_user_id,
		CreatedAt: time.Now(),
	})

	// Router初期化
	r := mux.NewRouter()

	// Route Hnadlers / Endpoints
	r.HandleFunc("/api/rooms", getRooms).Methods("GET")           //Roomリストを取得
	r.HandleFunc("/api/rooms/{id}", getRoom).Methods("GET")       //一つのRoomを取得
	r.HandleFunc("/api/rooms", createRoom).Methods("POST")        //Roomを作成
	r.HandleFunc("/api/rooms/{id}", updateRoom).Methods("PUT")    //Roomをアップデート
	r.HandleFunc("/api/rooms/{id}", deleteRoom).Methods("DELETE") //Roomを削除

	r.HandleFunc("/api/users", getUsers).Methods("GET")           //Userリストを取得
	r.HandleFunc("/api/users/{id}", getUser).Methods("GET")       //一つのUserを取得
	r.HandleFunc("/api/users", createUser).Methods("POST")        //Userを作成
	r.HandleFunc("/api/users/{id}", updateUser).Methods("PUT")    //Userをアップデート
	r.HandleFunc("/api/users/{id}", deleteUser).Methods("DELETE") //Userを削除

	r.HandleFunc("/api/signIn", signIn).Methods("GET") //tokenでsingInし、cookieを生成

	log.Fatal(http.ListenAndServe(":6000", r))
}
