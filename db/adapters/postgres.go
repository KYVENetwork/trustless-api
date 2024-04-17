package adapters

// import (
// 	"database/sql"
// 	"fmt"

// 	"github.com/KYVENetwork/trustless-rpc/files"
// 	"github.com/KYVENetwork/trustless-rpc/indexer"
// 	"github.com/uptrace/bun/driver/pgdriver"
// )

// type PostgresAdapter struct {
// 	saveDataItem files.SaveDataItem
// 	indexer      indexer.Indexer
// 	db           *sql.DB
// }

// func StartPostgres(saveDataItem files.SaveDataItem, indexer indexer.Indexer) PostgresAdapter {

// 	pgconn := pgdriver.NewConnector(
// 		pgdriver.WithAddr("localhost:5437"),
// 		pgdriver.WithUser("admin"),
// 		pgdriver.WithPassword("root"),
// 		pgdriver.WithDatabase("trustless-api"),
// 	)

// 	db := sql.OpenDB(pgconn)
// 	fmt.Println(db)

// 	logger.Debug().Msg("Connecting....")

// 	return PostgresAdapter{}

// }
