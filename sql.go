package main

import (
	"database/sql"
	"errors"
	"log"
	_ "modernc.org/sqlite"
)

type SQLUser struct {
	Name        string
	Email       string
	UUID        string
	NodeId      string
	AccessToken string
}

type SQLVerify struct {
	UserUUID   string
	VerifyCode string
	VerifyUUID string
	BiliUid    int64
}

var db *sql.DB

func initDB() error {
	err := errors.New("")
	db, err = sql.Open("sqlite", "file:./data.db")
	if err != nil {
		log.Printf("open db failed: %v", err)
		return err
	}
	ex, err := tableExist("user")
	if err != nil {
		log.Printf("check table exist failed: %v", err)
	}
	if !ex {
		_, err = db.Exec("CREATE TABLE user (name varchar(64), email varchar(64), access_token varchar(64), uuid varchar(64), node_id varchar(64))")
		if err != nil {
			return err
		}
	}
	ex, err = tableExist("verify")
	if err != nil {
		log.Printf("check table exist failed: %v", err)
	}
	if !ex {
		_, err = db.Exec("CREATE TABLE verify (user_uuid varchar(64), verify_code varchar(64), verify_uuid varchar(64), bili_uid bigint)")
		if err != nil {
			return err
		}
	}
	return nil
}

func tableExist(name string) (bool, error) {
	res, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name = ?", name)
	if err != nil {
		return false, err
	}
	defer func(res *sql.Rows) {
		err := res.Close()
		if err != nil {
			log.Printf("close rows failed: %v", err)
		}
	}(res)
	if res.Next() {
		return true, nil
	}
	return false, nil
}

func insertUser(user SQLUser) error {
	_, err := db.Exec("INSERT INTO user (name, email, access_token, uuid, node_id) VALUES (?, ?, ?, ?, ?)", user.Name, user.Email, user.AccessToken, user.UUID, user.NodeId)
	if err != nil {
		return err
	}
	return nil
}

func getUserByNodeID(id string) (SQLUser, error) {
	rows, err := db.Query("SELECT * FROM user WHERE node_id = ?", id)
	if err != nil {
		return SQLUser{}, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("close rows failed: %v", err)
		}
	}(rows)
	var user SQLUser
	for rows.Next() {
		err := rows.Scan(&user.Name, &user.Email, &user.AccessToken, &user.UUID, &user.NodeId)
		if err != nil {
			return SQLUser{}, err
		}
	}
	return user, nil
}

func getUserByUUID(uuid string) (SQLUser, error) {
	rows, err := db.Query("SELECT * FROM user WHERE uuid = ?", uuid)
	if err != nil {
		return SQLUser{}, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("close rows failed: %v", err)
		}
	}(rows)
	var user SQLUser
	for rows.Next() {
		err := rows.Scan(&user.Name, &user.Email, &user.AccessToken, &user.UUID, &user.NodeId)
		if err != nil {
			return SQLUser{}, err
		}
	}
	return user, nil
}

func insertVerify(UUID string, verifyCode string, verifyUUID string) error {
	_, err := db.Exec("INSERT INTO verify (user_uuid, verify_code, verify_uuid, bili_uid) VALUES (?, ?, ?, -1)", UUID, verifyCode, verifyUUID)
	if err != nil {
		return err
	}
	return nil
}

func setBiliUid(verifyCode string, biliUid int64) error {
	_, err := db.Exec("UPDATE verify SET bili_uid = ? WHERE verify_code = ?  and bili_uid = -1", biliUid, verifyCode)
	if err != nil {
		return err
	}
	return nil
}
