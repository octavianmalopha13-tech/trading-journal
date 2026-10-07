package main
import (
    "database/sql"
    "encoding/json"
    "net/http"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
)

type Account struct{
  ID int64 `json:"id"`
  UserID int64 `json:"user_id"`
  Broker string `json:"broker"`
  MT5Login string `json:"mt5_login"`
  MT5Server string `json:"mt5_server"` 
  CreatedAt int64 `json:"created_at"`
}

type CreatedAccountRequest struct{
  Broker string  `json:"broker"`
  MT5Login string `json:"mt5_login"`
  MT5Server string `json:"mt5_server"`
}

func handleCreateAccount(w http.ResponseWriter, r *http.Request, db *sql.DB){
  userID := getUserID(r)

  if userID == 0{
    http.Error(w,"unauthorized",http.StatusUnauthorized)
    return
  }

  var req CreatedAccountRequest
 
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil{
    http.Error(w,"invalid JSON", http.StatusBadRequest)
    return
  }

  if req.Broker == "" || req.MT5Login == "" || req.MT5Server == ""{
    http.Error(w,"broker, mt5_login and mt5_server are required",http.StatusBadRequest)
    return
  }

  result, err := db.Exec(`INSERT INTO accounts (user_id, broker, mt5_login, mt5_server, created_at) VALUES(?, ?, ?, ?, ?)`,
    userID,
    req.Broker,
    req.MT5Login,
    req.MT5Server,
    time.Now().Unix(),
  )

  if err != nil{
    http.Error(w,"failed to create account", http.StatusInternalServerError)
    return
  }

  accountID, err := result.LastInsertId()
 
  if err != nil{
    http.Error(w,"failed to get account ID",http.StatusInternalServerError)
    return
  }

  w.Header().Set("Content-Type","application/json")

  json.NewEncoder(w).Encode(map[string]interface{}{
    "status":"ok",
    "id": accountID,
  })
}

func handleListAccounts(w http.ResponseWriter, r *http.Request, db *sql.DB){
  userID := getUserID(r)

  if userID == 0{
    http.Error(w,"unauthorized",http.StatusUnauthorized)
    return
  }

  rows, err := db.Query(`SELECT id, user_id, broker, mt5_login, mt5_server, created_at FROM accounts WHERE user_id = ? ORDER BY id DESC`, userID)

  if err != nil{
    http.Error(w,"failed to fetch accounts", http.StatusInternalServerError)
    return
  }

  defer rows.Close()

  accounts := make([]Account, 0)

  for rows.Next(){
    var account Account

    err := rows.Scan(
      &account.ID,
      &account.UserID,
      &account.Broker,
      &account.MT5Login,
      &account.MT5Server,
      &account.CreatedAt,
   )

   if err != nil{
     http.Error(w,"failed to read account",http.StatusInternalServerError)
     return
   }

   accounts = append(accounts, account)
  }

   if err := rows.Err(); err != nil{
     http.Error(w,"failed to read accounts", http.StatusInternalServerError)
     return
   }


   w.Header().Set("Content-Type","application/json")

   json.NewEncoder(w).Encode(accounts)

}

func handleDeleteAccount(w http.ResponseWriter, r *http.Request, db *sql.DB){
  userID := getUserID(r)

  if userID == 0{
    http.Error(w,"unauthorized",http.StatusUnauthorized)
    return
  }

  idStr := chi.URLParam(r, "id")

  accountID, err := strconv.ParseInt(idStr, 10, 64)
 
  if err != nil{
    http.Error(w,"invalid account ID",http.StatusBadRequest)
    return
 }

  result, err := db.Exec(`DELETE FROM accounts WHERE id=? AND user_id=?`, accountID, userID)

  if err != nil{
    http.Error(w,"failed to delete account",http.StatusInternalServerError)
    return
  }

  rowsAffected, err := result.RowsAffected()

  if err != nil{
    http.Error(w,"failed to delete result",http.StatusInternalServerError)
    return
  }

  if rowsAffected == 0{
    http.Error(w,"account not found",http.StatusNotFound)
    return
  }

  w.Header().Set("Content-Type","application/json")

  json.NewEncoder(w).Encode(map[string]string{
    "status":"deleted",
  })

}
