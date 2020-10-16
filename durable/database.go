package durable

import (
	"fmt"
	"log"
	"multisig/configs"

	"github.com/timshannon/badgerhold"
)

type Database struct {
	db *badgerhold.Store
}

func OpenDatabaseClient(c *configs.Option) *Database {
	database := c.Database
	conn := fmt.Sprintf("%s%s", database.Path, database.Name)
	options := badgerhold.DefaultOptions
	options.Dir = conn
	options.ValueDir = conn

	db, err := badgerhold.Open(options)

	if err != nil {
		// handle error
		log.Fatal(err)
	}

	return &Database{db: db}
}

func (d *Database) UpdateMatching(data interface{}, query *badgerhold.Query, update func(interface{}) error) {
	d.db.UpdateMatching(data, query, update)
}

func (d *Database) Update(key interface{}, data interface{}) error {
	err := d.db.Update(key, data)
	return err
}

func (d *Database) Insert(data interface{}) error {
	key := badgerhold.NextSequence()
	err := d.db.Insert(key, data)
	return err
}

func (d *Database) Find(data interface{}, query *badgerhold.Query) error {
	err := d.db.Find(data, query)
	return err
}
