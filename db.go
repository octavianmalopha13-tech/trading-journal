package main
import (
  "database/sql"
  _"github.com/mattn/go-sqlite3"
)

func initDB(path string) (*sql.DB, error){
  db, err := sql.Open("sqlite3",path)
  if err != nil{
    return nil, err
  }
  if _,err = db.Exec("PRAGMA foreign_keys = ON"); err !=nil{
    return nil, err
  }
  if err:=db.Ping(); err != nil{
    return nil, err
  }
  if err:=createTables(db); err != nil{
    return nil, err
  }
 
  return db,nil
}

func createTables(db *sql.DB) error{
 statements :=[]string{
  `CREATE TABLE IF NOT EXISTS users(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL
  );`,

  `CREATE TABLE IF NOT EXISTS accounts(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    broker TEXT NOT NULL,
    mt5_login TEXT NOT NULL,
    mt5_server TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
  );`,

  `CREATE TABLE IF NOT EXISTS trades(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    ticket TEXT UNIQUE NOT NULL,
    symbol TEXT NOT NULL,
    direction TEXT NOT NULL,
    lot_size REAL NOT NULL,
    entry_price REAL NOT NULL,
    exit_price REAL,
    sl REAL,
    tp REAL,
    open_time INTEGER NOT NULL,
    close_time INTEGER,
    pnl REAL,
    commission REAL DEFAULT 0,
    swap REAL DEFAULT 0,
    duration_seconds INTEGER,
    FOREIGN KEY (account_id) REFERENCES accounts(id)
  );`,

  `CREATE TABLE IF NOT EXISTS journals(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trade_id INTEGER UNIQUE NOT NULL,
    setup TEXT,
    emotions TEXT,
    mistakes TEXT,
    lessons TEXT,
    notes TEXT,
    rating INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (trade_id) REFERENCES trades(id)
  )`,
 } 

 for _,stmt:=range statements{
   if _,err:=db.Exec(stmt); err !=nil{
      return err
    }
  }
  return nil
}

func seedIfEmpty(db *sql.DB) error{
  return nil
}
