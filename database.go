package main

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

type Database struct {
	db                      *sql.DB
	createRecordStmt        *sql.Stmt
	updateAverageTablesStmt *sql.Stmt
}

func OpenDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

func (d *Database) Close() error {
	if d.createRecordStmt != nil {
		if err := d.createRecordStmt.Close(); err != nil {
			return err
		}
	}
	if d.updateAverageTablesStmt != nil {
		if err := d.updateAverageTablesStmt.Close(); err != nil {
			return err
		}
	}

	return d.db.Close()
}

func (d *Database) Init() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			timestamp   TEXT PRIMARY KEY,
			co2         INTEGER NOT NULL,
			temperature REAL NOT NULL,
			humidity    REAL NOT NULL
		);
		CREATE TABLE IF NOT EXISTS records_hourly_avg (
			timestamp   TEXT PRIMARY KEY,
			co2         INTEGER NOT NULL,
			temperature REAL NOT NULL,
			humidity    REAL NOT NULL
		);
		CREATE TABLE IF NOT EXISTS records_daily_avg (
			timestamp   TEXT PRIMARY KEY,
			co2         INTEGER NOT NULL,
			temperature REAL NOT NULL,
			humidity    REAL NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	d.createRecordStmt, err = d.db.Prepare(`
		INSERT OR IGNORE INTO records (timestamp, co2, temperature, humidity)
		VALUES ($timestamp, $co2, $temperature, $humidity);
	`)
	if err != nil {
		return err
	}

	d.updateAverageTablesStmt, err = d.db.Prepare(`
		-- allows indexes to be used in LIKE clauses
		PRAGMA case_sensitive_like = ON;

		INSERT
		INTO records_hourly_avg (timestamp, co2, temperature, humidity)
		SELECT $ts_hour, AVG(co2), AVG(temperature), AVG(humidity)
		FROM records
		WHERE records.timestamp LIKE $ts_pattern_hour
		ON CONFLICT (timestamp) DO UPDATE SET co2         = excluded.co2,
											  temperature = excluded.temperature,
											  humidity    = excluded.humidity;

		INSERT
		INTO records_daily_avg (timestamp, co2, temperature, humidity)
		SELECT $ts_day, AVG(co2), AVG(temperature), AVG(humidity)
		FROM records
		WHERE records.timestamp LIKE $ts_pattern_day
		ON CONFLICT (timestamp) DO UPDATE SET co2         = excluded.co2,
											  temperature = excluded.temperature,
											  humidity    = excluded.humidity;
	`)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) CreateRecord(r *Record) error {
	ts := ISO8601Time(time.Time(r.Timestamp).Truncate(time.Minute))

	res, err := d.createRecordStmt.Exec(
		sql.Named("timestamp", ts),
		sql.Named("co2", r.Co2),
		sql.Named("temperature", r.Temperature),
		sql.Named("humidity", r.Humidity),
	)
	if err != nil {
		return err
	}

	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra > 0 {
		if err := d.updateAverageTables(time.Time(ts)); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) updateAverageTables(ts time.Time) error {
	tsHour := ISO8601Time(ts.Truncate(time.Hour)).format()
	tsDay := ISO8601Time(ts.Truncate(24 * time.Hour)).format()

	_, err := d.updateAverageTablesStmt.Exec(
		// 2023-06-16T14:00:00Z
		sql.Named("ts_hour", tsHour),
		// 2023-06-16T14:%
		sql.Named("ts_pattern_hour", tsHour[:14]+"%"),
		// 2023-06-16T00:00:00Z
		sql.Named("ts_day", tsDay),
		// 2023-06-16T%
		sql.Named("ts_pattern_day", tsDay[:11]+"%"),
	)
	if err != nil {
		return err
	}

	return nil
}
