package main

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type statsLogger struct {
	db *sql.DB
}

var (
	_stLogger *statsLogger
	_once     sync.Once
)

func newStatsLogger() *statsLogger {
	_once.Do(func() {
		db, err := sql.Open("sqlite3", _databasePath)
		if err != nil {
			log.Fatal(err)
		}
		stmt := `CREATE TABLE IF NOT EXISTS stat (minute INTEGER UNIQUE NOT NULL, count INTEGER NOT NULL);`
		if _, err = db.Exec(stmt); err != nil {
			log.Fatalf("failed to create table(s): %s", err)
		}
		_stLogger = &statsLogger{db: db}
	})
	return _stLogger
}

func (s *statsLogger) log(minute int64, count uint) {
	_, err := s.db.Exec(`
	INSERT INTO stat(minute, count) VALUES(?, ?)
	ON CONFLICT(minute) DO UPDATE SET count=count+excluded.count`, minute, count)
	if err != nil {
		log.Errorf("failed to insert (%d, %d) into stat: %s", minute, count, err)
		return
	}
}
