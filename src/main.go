package main

import (
	"encoding/json"
	"image/jpeg"
	"image/png"
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

	"strconv"
)

// グローバル変数なDB
var DB *gorm.DB

type Room struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Author           string `json:"author"`
	Description      string `json:"description"`
	Authorized_Users string `json:"authorized_users"`
	// Play_Time        string `json:"play_time"`
	Views int `json:"views"`
	// Comments         string `json:"commments"`
	Status string `json:"status"`
}

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GoogleSub string `json:"google_sub"`
}

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

// --------- General Functions ----------
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

func getGoogleInfo(id_token string) (*GoogleInfo, error) {

	// id_tokenのvalidation
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + id_token

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)

	google_info := &GoogleInfo{} //ポインタ型のGoogleInfo

	err = json.Unmarshal(byteArray, &google_info)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return google_info, nil
}

func isRegisteredGoogleSub(google_sub string) bool {

	var user User
	result := DB.First(&user, "google_sub = ?", google_sub)

	if result.RowsAffected != 0 {
		return true
	} else {
		return false
	}
}

func isRegisteredUserId(user_id string) bool {

	var user User
	result := DB.First(&user, "id = ?", user_id)

	if result.RowsAffected != 0 {
		return true
	} else {
		return false
	}
}

func isRegisteredRoomId(room_id string) bool {

	var room Room
	result := DB.First(&room, "id = ?", room_id)

	if result.RowsAffected != 0 {
		return true
	} else {
		return false
	}
}

func extractExt(name string) string {
	pos := strings.LastIndex(name, ".")

	// .から後ろをスライスで取得
	return name[pos:]
}

func getSession(session_id string) *Session {
	var session Session
	result := DB.First(&session, "id = ?", session_id)

	if result.RowsAffected != 0 {
		return &session
	} else {
		return nil
	}
}

// --------- Room Operate Functions ----------

// Get All Searched Rooms
func getRooms(w http.ResponseWriter, r *http.Request) {

	log.Println("getRooms")

	var rooms []Room

	var query = r.URL.Query() //クエリパラメータの取得

	if query != nil && query["search_word"] != nil {
		DB.Where("title LIKE ?", "%"+query["search_word"][0]+"%").Where("status = ?", "public").Find(&rooms) //部分一致検索
	} else {
		DB.Find(&rooms) //すべて出力
	}

	responseBody, err := json.Marshal(rooms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write((responseBody))
}

// Get Single Room
func getRoom(w http.ResponseWriter, r *http.Request) {

	log.Println("getRoom")

	var room Room
	params := mux.Vars(r)

	DB.First(&room, "id = ?", params["id"])

	log.Println(room.Description)

	responseBody, err := json.Marshal(room)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// Create a Room
func createRoom(w http.ResponseWriter, r *http.Request) {

	log.Println("createRoom")

	// 1. cookieからsession_idの確認＆user_idの取得
	cookie, err := r.Cookie("_cookie")

	if err != nil {
		http.Error(w, fmt.Sprintf("Error. You are not login: %v", err), http.StatusBadRequest)
		return
	}

	session_id := cookie.Value

	session := getSession(session_id)
	user_id := session.UserID

	log.Println(user_id)

	// 2. Roomレコードの作成

	// formでPOSTされた情報をDBにセット
	reqBody, _ := ioutil.ReadAll(r.Body)

	var room Room
	if err := json.Unmarshal(reqBody, &room); err != nil {
		http.Error(w, fmt.Sprintf("Error. Unsupported json request: %v", err), http.StatusBadRequest)
		return
	}

	log.Println(room.Author)

	// jsonでPOSTされない情報をDBにセット
	room.ID = createID()
	room.Author = user_id
	room.Views = 0
	room.Status = "private"

	// DB作成
	DB.Create(&room)

	// 3. Room自体の作成
	err = cp.Copy("uploads/template_room", "uploads/"+room.ID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error Room does not exist: %v", err), http.StatusBadRequest)
		return
	}

	// 4. 作成後の結果をjsonで返す(これでroom_idも含まれる)
	responseBody, err := json.Marshal(room)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)

}

// Update a Room
func updateRoom(w http.ResponseWriter, r *http.Request) {

	log.Println("updateRoom")

	// 1. cookieからsession_idの確認＆user_idの取得
	cookie, err := r.Cookie("_cookie")
	if err != nil {
		log.Fatal("Cookie: ", err)
	}
	session_id := cookie.Value

	session := getSession(session_id)
	user_id := session.UserID

	log.Println(user_id)

	// 2. Roomの更新(json形式でPOSTされる前提)

	// room_idをurlパラメータから取得
	vars := mux.Vars(r)
	id := vars["id"]

	log.Println(id)

	// jsonでPOSTされた情報でDB更新
	reqBody, _ := ioutil.ReadAll(r.Body)
	var room Room
	if err := json.Unmarshal(reqBody, &room); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	room.Status = "public"
	// todo: room構造体に要素が存在する場合のみアップデートするようにしたい(ただし要素が空白でもアップデート)
	DB.Model(&room).Where("id = ?", id).Updates(
		map[string]interface{}{
			"title":       room.Title,
			"description": room.Description,
			"status":      room.Status,
		},
	)

	// 3. 更新後の結果をjsonで返す
	responseBody, err := json.Marshal(room)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// --------- User Operate Functions ----------

// Get All Users
func getUsers(w http.ResponseWriter, r *http.Request) {

	var users []User
	DB.Find(&users) //見つけた結果をusersに入れよ

	responseBody, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write((responseBody))
}

// Get Single User
func getUser(w http.ResponseWriter, r *http.Request) {

	log.Println("getUser")

	var user User
	params := mux.Vars(r)
	DB.First(&user, "id = ?", params["id"])

	responseBody, err := json.Marshal(user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	DB.Create(&user)

	responseBody, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	DB.Model(&user).Where("id = ?", id).Updates(
		map[string]interface{}{
			"name":       user.Name,
			"google_sub": user.GoogleSub,
		})

	responseBody, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

// --------- Other Operate Functions ----------

// tokenから登録判定し、
// 1. 登録されていればセッション作成＆success.htmlにリダイレクト
// 2. 登録されていなければregister.htmlにリダイレクト
func signIn(w http.ResponseWriter, r *http.Request) {

	log.Println("Sign In")

	// tokenからgoogle sub取得
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id_token := r.Form.Get("credential")

	google_info, _ := getGoogleInfo(id_token)
	google_sub := google_info.Sub

	if isRegisteredGoogleSub(google_sub) {
		var user User
		DB.First(&user, "google_sub = ?", google_sub)

		// session生成
		session := createSession(user.ID)

		// cookieを生成
		cookie := http.Cookie{
			Name:     "_cookie",
			Value:    session.ID,
			HttpOnly: true,
		}

		// cookieをレスポンスに入れる
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/success.html", http.StatusMovedPermanently)

	} else {

		http.Redirect(w, r, "/register.html", http.StatusMovedPermanently)

	}
}

func register(w http.ResponseWriter, r *http.Request) {

	log.Println("register")

	err := r.ParseMultipartForm(1024 * 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user_name := r.Form.Get("user-name")
	id_token := r.Form.Get("credential")

	google_info, _ := getGoogleInfo(id_token)
	google_sub := google_info.Sub

	var user User

	user.ID = createID()
	user.Name = user_name
	user.GoogleSub = google_sub

	DB.Create(&user)

	http.Redirect(w, r, "/success.html", http.StatusMovedPermanently)
}

func upload(w http.ResponseWriter, r *http.Request) {

	log.Println("upload")

	// 1. cookieからsession_idの存在確認＆user_idの取得
	cookie, err := r.Cookie("_cookie")
	if err != nil {
		log.Fatal("Cookie: ", err)
	}
	session_id := cookie.Value

	session := getSession(session_id) // これに失敗するとHTTP ERRを返す
	user_id := session.UserID

	log.Println(user_id)

	// 2. ファイル取得(ParseForm)
	file, fileHeader, err := r.FormFile("file-input")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 3. ファイル種別＆room_id取得
	file_type := r.FormValue("file_type")
	room_id := r.FormValue("room_id") //こいつが動いてない = upload APIへのurlが間違ってる。

	log.Println("room_id is ", room_id)

	// 4. ファイル保存
	if file_type == "motion" {

		uploadedFileName := fileHeader.Filename

		// アップロードファイルのバリデーション
		ext := extractExt(uploadedFileName)
		if ext != ".vmd" {

			msg := "this is not vmd file"
			log.Println(msg)

			http.Error(w, msg, http.StatusBadRequest)

			return
		}

		// 保存実行
		dst, err := os.Create(fmt.Sprintf("./uploads/%s/static/vmds/motion.vmd", room_id))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Println("motion upload ok")

	} else if file_type == "thumbnail" {

		log.Println("thubnail save start")

		fileHandler := r.MultipartForm.File["file-input"][0]

		filename := fileHandler.Filename
		ext := extractExt(filename)

		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			msg := "not supported image type"
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		file, err := fileHandler.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dst, err := os.Create(fmt.Sprintf("./uploads/%s/thumbnail.jpg", room_id))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		opts := &jpeg.Options{Quality: 100}

		if ext == ".jpeg" || ext == ".jpg" {

			_, err = io.Copy(dst, file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

		}

		if ext == ".png" {

			img, err := png.Decode(file) //pngはjpegに変換して保存
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			jpeg.Encode(dst, img, opts)
		}

		log.Println("thumbnail upload ok")

	} else {

		http.Error(w, err.Error(), http.StatusBadRequest) // file not supported とか返したい
		return

	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(room_id))

}

func countViews(w http.ResponseWriter, r *http.Request) {

	log.Println("countViews")
	var room Room
	params := mux.Vars(r)

	room_id := params["id"]

	DB.First(&room, "id = ?", room_id)

	views_after := room.Views + 1

	log.Println("views: " + strconv.Itoa(views_after))

	DB.Model(&room).Where("id = ?", room_id).Updates(
		map[string]interface{}{
			"views": views_after,
		})
}

func main() {

	log.Println("Go is runnning")

	// DB初期化
	db, err := gorm.Open(sqlite.Open("yuri-kabe.db"), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	DB = db

	var mock_room_id = "template_room"
	var mock_user_id = "mock_user_id"
	var mock_session_id string = createID()

	// Create "rooms" table
	DB.AutoMigrate(&Room{})

	// Create Mock Data
	if !isRegisteredRoomId(mock_room_id) {
		DB.Create(&Room{
			ID:               mock_room_id,
			Title:            "template room",
			Author:           mock_user_id,
			Description:      "This is mock data.",
			Authorized_Users: "'1', '2', '3'",
			Views:            0,
			Status:           "public",
		})
	}

	// Create "users" table
	DB.AutoMigrate(&User{})

	// Create Mock Data
	if !isRegisteredUserId(mock_user_id) {
		DB.Create(&User{
			ID:        mock_user_id,
			Name:      "Gakki",
			GoogleSub: "110505856284770188621",
		})
	}

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

	r.HandleFunc("/api/signIn", signIn).Methods("POST")     // -> 廃止予定 -> 廃止せず、registrationCheckを統合
	r.HandleFunc("/api/register", register).Methods("POST") // -> /api/users の POST に統合予定(状況見て判断)

	r.HandleFunc("/api/upload", upload).Methods("POST")

	r.HandleFunc("/api/countViews/{id}", countViews).Methods("GET")

	// 静的ファイルへのルーティング
	r.Handle("/api/static/room/", http.StripPrefix("/api/static/room", http.FileServer(http.Dir("uploads"))))
	r.NotFoundHandler = http.StripPrefix("/api/static/room", http.FileServer(http.Dir("uploads")))

	log.Fatal(http.ListenAndServe(":6000", r))

}
