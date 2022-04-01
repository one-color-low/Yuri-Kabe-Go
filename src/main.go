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

	cp "github.com/otiai10/copy"
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

type GoogleInfo struct {
	Iss   string `json:"iss"`
	Nbf   string `json:"nbf"`
	Aud   string `json:"aud"`
	Sub   string `json:"sub"`
	Email string `json:"email"`
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

func getGoogleInfo(id_token string) GoogleInfo {

	// id_tokenのvalidation
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + id_token

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)

	google_info := GoogleInfo{}

	err = json.Unmarshal(byteArray, &google_info)
	if err != nil {
		fmt.Println(err.Error())
	}

	return google_info
}

func isRegistered(id_token string) bool {

	google_info := getGoogleInfo(id_token)

	google_sub := google_info.Sub

	// google_subでUserテーブルを検索し、特定のUser構造体を取得
	var user User
	result := DB.First(&user, "google_sub = ?", google_sub)

	if result.RowsAffected != 0 {
		return true
	} else {
		return false
	}
}

// ----------- Room Model --------------
type Room struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Author           string `json:"author"`
	Description      string `json:"description`
	Authorized_Users string `json:"authorized_users`
}

// Get Rooms
func getRooms(w http.ResponseWriter, r *http.Request) {

	log.Println("getRooms")

	var rooms []Room

	var query = r.URL.Query() //クエリパラメータの取得

	if query != nil && query["search_word"] != nil {
		DB.Where("title LIKE ?", "%"+query["search_word"][0]+"%").Find(&rooms) //部分一致検索
	} else {
		DB.Find(&rooms) //すべて出力
	}

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

// Sign In -> 廃止予定(セッション作る場合の実装)
func signIn(w http.ResponseWriter, r *http.Request) {

	// 1. rからid_tokenを取得
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	id_token := r.Form.Get("credential")

	// 2. id_tokenのvalidation
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + id_token

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	client := new(http.Client)
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)

	google_info := GoogleInfo{}
	err = json.Unmarshal(byteArray, &google_info)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 3. id_tokenからgoogle_subを取得
	google_sub := google_info.Sub

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
	http.Redirect(w, r, "/", 200)
}

func registrationCheck(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	id_token := r.Form.Get("credential")

	if isRegistered(id_token) {
		http.Redirect(w, r, "/success.html", http.StatusMovedPermanently)
	} else {
		http.Redirect(w, r, "/register.html", http.StatusMovedPermanently)
	}
}

func register(w http.ResponseWriter, r *http.Request) {

	log.Println("register")

	err := r.ParseMultipartForm(1024 * 5)
	if err != nil {
		log.Fatal(err)
	}

	user_name := r.Form.Get("user-name")
	id_token := r.Form.Get("credential")

	google_info := getGoogleInfo(id_token)
	google_sub := google_info.Sub

	var user User

	user.ID = createID()
	user.Name = user_name
	user.GoogleSub = google_sub

	DB.Create(&user)

	http.Redirect(w, r, "/success.html", http.StatusMovedPermanently)
}

func upload(w http.ResponseWriter, r *http.Request) {
	room_id := createID()
	log.Println(room_id)

	err := cp.Copy("uploads/template_room", "uploads/"+room_id)
	fmt.Println(err) // nil

	http.Redirect(w, r, "/success.html", http.StatusMovedPermanently)

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
		GoogleSub: "110505856284770188621",
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
	r.HandleFunc("/api/rooms", getRooms).Methods("GET")           //Room情報リストを取得
	r.HandleFunc("/api/rooms/{id}", getRoom).Methods("GET")       //一つのRoom情報を取得
	r.HandleFunc("/api/rooms", createRoom).Methods("POST")        //Roomを作成
	r.HandleFunc("/api/rooms/{id}", updateRoom).Methods("PUT")    //Roomをアップデート
	r.HandleFunc("/api/rooms/{id}", deleteRoom).Methods("DELETE") //Roomを削除

	r.HandleFunc("/api/users", getUsers).Methods("GET")           //Userリストを取得
	r.HandleFunc("/api/users/{id}", getUser).Methods("GET")       //一つのUserを取得
	r.HandleFunc("/api/users", createUser).Methods("POST")        //Userを作成
	r.HandleFunc("/api/users/{id}", updateUser).Methods("PUT")    //Userをアップデート
	r.HandleFunc("/api/users/{id}", deleteUser).Methods("DELETE") //Userを削除

	r.HandleFunc("/api/signIn", signIn).Methods("POST")                       // -> 廃止予定
	r.HandleFunc("/api/registrationCheck", registrationCheck).Methods("POST") // -> 残す?
	r.HandleFunc("/api/register", register).Methods("POST")                   // -> /api/users の POST に統合予定(状況見て判断)

	r.HandleFunc("/api/upload", upload).Methods("GET")

	// 静的ファイルへのルーティング
	r.Handle("/api/static/room/", http.StripPrefix("/api/static/room", http.FileServer(http.Dir("uploads"))))
	r.NotFoundHandler = http.StripPrefix("/api/static/room", http.FileServer(http.Dir("uploads")))

	log.Fatal(http.ListenAndServe(":6000", r))

}
