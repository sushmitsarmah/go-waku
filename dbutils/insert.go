package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver

	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

func genRandomBytes(size int) (blk []byte, err error) {
	blk = make([]byte, size)
	_, err = rand.Read(blk)
	return
}

func genRandomTimestamp(now int64, last30d int64) int64 {
	return rand.Int63n(last30d) + now
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func genRandomContentTopic(n int) string {
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5"}
	i := n % 5
	return protocol.NewContentTopic("test", 1, topics[i], "plaintext").String()
}

func newdb(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func createTable(db *sql.DB) error {
	sqlStmt := `CREATE TABLE IF NOT EXISTS message (
		id BLOB,
		receiverTimestamp INTEGER NOT NULL,
		senderTimestamp INTEGER NOT NULL,
		contentTopic BLOB NOT NULL,
		pubsubTopic BLOB NOT NULL,
		payload BLOB,
		version INTEGER NOT NULL DEFAULT 0,
		CONSTRAINT messageIndex PRIMARY KEY (senderTimestamp, id, pubsubTopic)
	) WITHOUT ROWID;
	
	CREATE INDEX IF NOT EXISTS message_senderTimestamp ON message(senderTimestamp);
	CREATE INDEX IF NOT EXISTS message_receiverTimestamp ON message(receiverTimestamp);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}

func main() {

	Ns := []int{10_000, 100_000, 1_000_000}

	for _, N := range Ns {
		dbName := fmt.Sprintf("store%d.db", N)
		fmt.Println("Inserting ", N, " records in ", dbName)

		db, err := newdb(dbName)
		query := "INSERT INTO message (id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version) VALUES (?, ?, ?, ?, ?, ?, ?)"

		err = createTable(db)
		if err != nil {
			panic(err)
		}

		trx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			panic(err)
		}

		stmt, err := trx.Prepare(query)
		if err != nil {
			panic(err)
		}

		last30d := time.Now().UnixNano() - time.Now().Add(-30*time.Hour*24).UnixNano()
		now := time.Now().Add(-1 * time.Minute).UnixNano()
		pubsubTopic := protocol.DefaultPubsubTopic().String()
		for i := 1; i <= N; i++ {

			if i%1000 == 0 && i > 1 && i < N {
				err := trx.Commit()
				if err != nil {
					panic(err)
				}

				trx, err = db.BeginTx(context.Background(), nil)
				if err != nil {
					panic(err)
				}

				stmt, err = trx.Prepare(query)
				if err != nil {
					panic(err)
				}
			}

			if i%(N/10) == 0 && i > 1 {
				fmt.Println(i, "...")
			}

			randPayload, err := genRandomBytes(100)
			if err != nil {
				panic(err)
			}

			msg := pb.WakuMessage{
				Version:      0,
				ContentTopic: genRandomContentTopic(i),
				Timestamp:    genRandomTimestamp(now, last30d),
				Payload:      randPayload,
			}

			hash, err := msg.Hash()
			if err != nil {
				panic(err)
			}

			_, err = stmt.Exec(hash, msg.Timestamp, msg.Timestamp, msg.ContentTopic, pubsubTopic, msg.Payload, msg.Version)
			if err != nil {
				panic(err)
			}
		}

		err = trx.Commit()
		if err != nil {
			panic(err)
		}
	}
}