package main
//import "fmt"


type User struct{
  ID int64 `json:"id"`
  Email string `json:"email"`
  PasswordHash string `json:"-"`
  CreatedAt int64 `json:"created_at"`
}

/*type Account struct{
  ID int64 `json:"id"`
  UserID int64 `json:"user_id"`
  Broker string `json:"broker"`
  MT5Login string `json:"mt5_login"`
  MT5Server string `json:"mt5_server"`
  CreatedAt int64 `json:"created_at"`
}*/

type Trade struct{
  ID int64 `json:"id"`
  AccountID int64 `json:"account_id"`
  Ticket string `json:"ticket"`
  Symbol string `json:"symbol"`
  Direction string `json:"direction"`
  LotSize float64 `json:"lot_size"`
  EntryPrice float64 `json:"entry_price"`
  ExitPrice float64 `json:"exit_price"`
  SL *float64 `json:"sl"`
  TP *float64 `json:"tp"`
  OpenTime int64 `json:"open_time"`
  CloseTime *int64 `json:"close_time"`
  PnL *float64 `json:"pnl"`
  Commission float64 `json:"commission"`
  Swap float64 `json:"swap"`
  DurationSeconds *int64 `json:"duration_seconds"`
}

type Journal struct{
  ID int64 `json:"id"`
  TradeID int64 `json:"trade_id"`
  Setup string `json:"setup"`
  Emotions string `json:"emotions"`
  Mistakes string `json:"mistakes"`
  Lessons string `json:"lessons"`
  Notes string `json:"notes"`
  Rating int `json:"rating"`
  CreatedAt int64 `json:"created_at"`
  UpdatedAt int64 `json:"updated_at"`
}
