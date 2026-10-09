package main

import (
	"database/sql"
	"log"
	"math/rand"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func seedDatabase(db *sql.DB) error {

	var count int

	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE email = ?
	`, "demo@trader.com").Scan(&count)

	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("✅ Seed data already present, skipping")
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte("demo1234"),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return err
	}

	result, err := db.Exec(`
		INSERT INTO users (email, password_hash, created_at)
		VALUES (?, ?, ?)
	`, "demo@trader.com", string(hashedPassword), time.Now().Unix())

	if err != nil {
		return err
	}

	userID, err := result.LastInsertId()

	if err != nil {
		return err
	}

	result, err = db.Exec(`
		INSERT INTO accounts (
			user_id,
			broker,
			mt5_login,
			mt5_server,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		userID,
		"IC Markets",
		"12345678",
		"ICMarkets-Demo",
		time.Now().Unix(),
	)

	if err != nil {
		return err
	}

	icMarketsID, err := result.LastInsertId()

	if err != nil {
		return err
	}

	result, err = db.Exec(`
		INSERT INTO accounts (
			user_id,
			broker,
			mt5_login,
			mt5_server,
			created_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		userID,
		"Exness",
		"87654321",
		"Exness-MT5Trial",
		time.Now().Unix(),
	)

	if err != nil {
		return err
	}

	exnessID, err := result.LastInsertId()

	if err != nil {
		return err
	}

	accountIDs := []int64{
		icMarketsID,
		exnessID,
	}

	if err := seedTrades(db, userID, accountIDs); err != nil {
		return err
	}

	log.Println("✅ Demo seed data created successfully")

	return nil
}

func seedTrades(
	db *sql.DB,
	userID int64,
	accountIDs []int64,
) error {

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	symbols := []string{
		"EURUSD",
		"GBPUSD",
		"USDJPY",
		"XAUUSD",
		"AUDUSD",
		"USDCAD",
	}

	for i := 0; i < 50; i++ {

		accountID := accountIDs[r.Intn(len(accountIDs))]

		symbol := symbols[r.Intn(len(symbols))]

		direction := "buy"

		if r.Intn(2) == 1 {
			direction = "sell"
		}

		lotSize := 0.01 + r.Float64()*1.99

		entryPrice := randomEntryPrice(r, symbol)

		slDistance := 20 + r.Float64()*80
		tpDistance := 20 + r.Float64()*80

		var sl float64
		var tp float64

		if direction == "buy" {
			sl = entryPrice - pipValue(symbol, slDistance)
			tp = entryPrice + pipValue(symbol, tpDistance)
		} else {
			sl = entryPrice + pipValue(symbol, slDistance)
			tp = entryPrice - pipValue(symbol, tpDistance)
		}

		durationSeconds := int64(
			300 + r.Intn(4*60*60-300+1),
		)

		closeTime := time.Now().Unix() -
			int64(r.Intn(30*24*60*60))

		openTime := closeTime - durationSeconds

		var pnl float64

		if r.Intn(100) < 60 {
			pnl = 10 + r.Float64()*140
		} else {
			pnl = -(5 + r.Float64()*95)
		}

		commission := -(lotSize * (2 + r.Float64()*3))
		swap := -r.Float64() * 5

		pnl += commission + swap

		result, err := db.Exec(`
			INSERT INTO trades (
				account_id,
				ticket,
				symbol,
				direction,
				lot_size,
				entry_price,
				exit_price,
				sl,
				tp,
				open_time,
				close_time,
				pnl,
				commission,
				swap,
				duration_seconds
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			accountID,
			seedTicket(r),
			symbol,
			direction,
			lotSize,
			entryPrice,
			generateExitPrice(r, symbol, entryPrice, direction),
			sl,
			tp,
			openTime,
			closeTime,
			pnl,
			commission,
			swap,
			durationSeconds,
		)

		if err != nil {
			return err
		}

		tradeID, err := result.LastInsertId()

		if err != nil {
			return err
		}

		if r.Intn(2) == 1 {
			if err := seedJournal(db, tradeID); err != nil {
				return err
			}
		}
	}

	_ = userID

	return nil
}

func seedJournal(db *sql.DB, tradeID int64) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	setups := []string{
		"Breakout of Asian range",
		"Trend continuation on H1",
		"Support bounce at key level",
		"Double top reversal",
		"News spike fade",
		"EMA crossover pullback",
		"London session breakout",
		"Fibonacci retracement entry",
	}

	emotions := []string{
		"Calm, followed plan",
		"Slightly anxious but patient",
		"Confident after previous win",
		"Hesitant at entry",
		"Disciplined, waited for confirmation",
		"FOMO, entered too early",
	}

	mistakes := []string{
		"None",
		"Entered slightly late",
		"Moved SL too early",
		"Took profit too soon",
		"Position size too large",
		"Ignored higher timeframe",
	}

	lessons := []string{
		"Wait for retest before entering",
		"Set limit order instead of market",
		"Trust the plan, don't panic close",
		"Reduce size in choppy markets",
		"Check HTF bias first",
	}

	rating := 1 + r.Intn(5)

	now := time.Now().Unix()

	_, err := db.Exec(`
		INSERT INTO journals (
			trade_id,
			setup,
			emotions,
			mistakes,
			lessons,
			notes,
			rating,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		tradeID,
		setups[r.Intn(len(setups))],
		emotions[r.Intn(len(emotions))],
		mistakes[r.Intn(len(mistakes))],
		lessons[r.Intn(len(lessons))],
		"Demo journal entry for testing.",
		rating,
		now,
		now,
	)

	return err
}

func randomEntryPrice(r *rand.Rand, symbol string) float64 {
	switch symbol {
	case "EURUSD":
		return 1.08 + (r.Float64()-0.5)*0.02

	case "GBPUSD":
		return 1.27 + (r.Float64()-0.5)*0.03

	case "USDJPY":
		return 150 + (r.Float64()-0.5)*2

	case "XAUUSD":
		return 2000 + (r.Float64()-0.5)*40

	case "AUDUSD":
		return 0.66 + (r.Float64()-0.5)*0.02

	case "USDCAD":
		return 1.36 + (r.Float64()-0.5)*0.02

	default:
		return 1.0
	}
}

func pipValue(symbol string, pips float64) float64 {
	if symbol == "USDJPY" {
		return pips * 0.01
	}

	if symbol == "XAUUSD" {
		return pips * 0.10
	}

	return pips * 0.0001
}

func generateExitPrice(
	r *rand.Rand,
	symbol string,
	entry float64,
	direction string,
) float64 {

	distance := pipValue(
		symbol,
		20+r.Float64()*80,
	)

	if direction == "buy" {
		return entry + (r.Float64()*2-1)*distance
	}

	return entry - (r.Float64()*2-1)*distance
}

func seedTicket(r *rand.Rand) string {
	return strconv.FormatInt(
		10000000+int64(r.Intn(90000000)),
		10,
	)
}
