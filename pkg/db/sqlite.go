package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type User struct {
	TelegramID int64
	UserID     int64
	CreatedAt  int64
}

type RechargeRequest struct {
	ID            int64
	TelegramID    int64
	UserID        int64
	Amount        int64
	Status        string // "pending", "approved", "rejected"
	ReceiptFileID string
	CreatedAt     int64
}

type Plan struct {
	ID          int64
	SubscribeID int
	Name        string
	Price       int64
	Description string
}

type DB struct {
	conn *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) initSchema() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		telegram_id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);`

	rechargeTable := `
	CREATE TABLE IF NOT EXISTS recharge_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		amount INTEGER NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('pending', 'approved', 'rejected')),
		receipt_file_id TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);`

	settingsTable := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`

	plansTable := `
	CREATE TABLE IF NOT EXISTS plans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		subscribe_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		price INTEGER NOT NULL,
		description TEXT NOT NULL
	);`

	if _, err := db.conn.Exec(usersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := db.conn.Exec(rechargeTable); err != nil {
		return fmt.Errorf("failed to create recharge table: %w", err)
	}

	if _, err := db.conn.Exec(settingsTable); err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}

	if _, err := db.conn.Exec(plansTable); err != nil {
		return fmt.Errorf("failed to create plans table: %w", err)
	}

	return nil
}

// User CRUD
func (db *DB) GetUser(telegramID int64) (*User, error) {
	row := db.conn.QueryRow("SELECT telegram_id, user_id, created_at FROM users WHERE telegram_id = ?", telegramID)
	var u User
	if err := row.Scan(&u.TelegramID, &u.UserID, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (db *DB) SaveUser(u *User) error {
	_, err := db.conn.Exec(`
		INSERT INTO users (telegram_id, user_id, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(telegram_id) DO UPDATE SET
			user_id = excluded.user_id;`,
		u.TelegramID, u.UserID, u.CreatedAt)
	return err
}

// Recharge Requests
func (db *DB) CreateRechargeRequest(req *RechargeRequest) (int64, error) {
	res, err := db.conn.Exec(`
		INSERT INTO recharge_requests (telegram_id, user_id, amount, status, receipt_file_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?);`,
		req.TelegramID, req.UserID, req.Amount, req.Status, req.ReceiptFileID, req.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) GetRechargeRequest(id int64) (*RechargeRequest, error) {
	row := db.conn.QueryRow(`
		SELECT id, telegram_id, user_id, amount, status, receipt_file_id, created_at
		FROM recharge_requests WHERE id = ?`, id)
	var r RechargeRequest
	if err := row.Scan(&r.ID, &r.TelegramID, &r.UserID, &r.Amount, &r.Status, &r.ReceiptFileID, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (db *DB) UpdateRechargeStatus(id int64, status string) error {
	_, err := db.conn.Exec("UPDATE recharge_requests SET status = ? WHERE id = ?", status, id)
	return err
}

// Settings KV Store
func (db *DB) GetSetting(key string) (string, error) {
	row := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key)
	var val string
	if err := row.Scan(&val); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

func (db *DB) SetSetting(key, val string) error {
	_, err := db.conn.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value;`,
		key, val)
	return err
}

// Plans DB Management
func (db *DB) GetPlans() ([]Plan, error) {
	rows, err := db.conn.Query("SELECT id, subscribe_id, name, price, description FROM plans")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.SubscribeID, &p.Name, &p.Price, &p.Description); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (db *DB) SavePlan(p *Plan) (int64, error) {
	if p.ID == 0 {
		res, err := db.conn.Exec(`
			INSERT INTO plans (subscribe_id, name, price, description)
			VALUES (?, ?, ?, ?);`,
			p.SubscribeID, p.Name, p.Price, p.Description)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	} else {
		_, err := db.conn.Exec(`
			UPDATE plans SET subscribe_id = ?, name = ?, price = ?, description = ?
			WHERE id = ?;`,
			p.SubscribeID, p.Name, p.Price, p.Description, p.ID)
		return p.ID, err
	}
}

func (db *DB) DeletePlan(id int64) error {
	_, err := db.conn.Exec("DELETE FROM plans WHERE id = ?", id)
	return err
}
