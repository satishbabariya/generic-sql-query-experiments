package main

import (
	"fmt"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"reflect"
	"strings"
)

var schema = `
CREATE TABLE IF NOT EXISTS user (
	user_id    INTEGER PRIMARY KEY,
	email      VARCHAR(250) NOT NULL UNIQUE,
	password   VARCHAR(250) DEFAULT NULL
);
`

type User struct {
	UserId   int    `db:"user_id,primary",json:"user_id"`
	Email    string `db:"email",json:"email"`
	Password string `db:"password"`
}

// generic insert query

type Query[T any] struct {
	db    *sqlx.DB
	table string // if this not present reflect it from the model
	model T
}

func contains[T int | string](s []T, str T) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func SetPrimaryKey(in interface{}, value int64) {
	v := reflect.ValueOf(in)
	for i := 0; i < v.NumField(); i++ {
		key := v.Type().Field(i).Tag.Get("db")
		keys := strings.Split(key, ",")
		if contains(keys, "primary") {
			field := v.Field(i)
			if field.CanSet() {
				field.SetInt(value)
			} else {
				log.Fatal("ERROR: field cannot be set", field)
			}

		}
	}
}

func (q Query[T]) columns(primary bool) string {
	v := reflect.ValueOf(q.model)
	var fields []string
	for i := 0; i < v.NumField(); i++ {
		key := v.Type().Field(i).Tag.Get("db")
		keys := strings.Split(key, ",")

		// if keys contains "primary", then this is the primary key and skip it
		if contains(keys, "primary") && !primary {
			continue
		}

		fields = append(fields, keys[0])
	}
	return strings.Join(fields, ", ")
}

func (q Query[T]) values(in T, primary bool) string {
	v := reflect.ValueOf(in)
	var values []string
	for i := 0; i < v.NumField(); i++ {
		key := v.Type().Field(i).Tag.Get("db")
		keys := strings.Split(key, ",")

		// if keys contains "primary", then this is the primary key and skip it
		if contains(keys, "primary") && !primary {
			continue
		}

		// if the field is a pointer, dereference it
		if v.Type().Field(i).Type.Kind() == reflect.Ptr {
			values = append(values, fmt.Sprintf("'%v'", v.Field(i).Elem()))
		} else {
			values = append(values, fmt.Sprintf("'%v'", v.Field(i)))
		}

		//fields[i] = fmt.Sprintf("'%s'", v.Field(i).Interface())
	}
	return strings.Join(values, ", ")
}

func (q Query[T]) Insert(in T) (*int64, error) {
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.table, q.columns(false), q.values(in, false))
	log.Println("DEBUG:", query)
	res, err := q.db.Exec(query)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	// TODO: find primary key and set it on the model
	//SetPrimaryKey(in, id)

	return &id, nil
}

func (q Query[T]) Find() ([]T, error) {
	var results []T
	query := fmt.Sprintf("SELECT %s FROM %s", q.columns(true), q.table)
	log.Println("DEBUG:", query)
	err := q.db.Select(&results, query)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func main() {
	db, err := sqlx.Connect("sqlite3", "./example.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}

	user := User{
		Email:    gofakeit.Email(),
		Password: "password",
	}

	query := Query[User]{
		db:    db,
		table: "user",
	}

	res, err := query.Insert(user)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	users, err := query.Find()
	if err != nil {
		log.Fatal(err)
	}

	log.Println(users)

	return
}
