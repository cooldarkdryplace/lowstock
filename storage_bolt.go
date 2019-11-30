package lowstock

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
)

var (
	usersBucket  = []byte("Users")
	tokensBucket = []byte("TempTokens")
)

type BoltStorage struct {
	db *bolt.DB
}

func NewBoltStorage(file string) (*BoltStorage, error) {
	db, err := bolt.Open(file, 0644, nil)
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(usersBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(tokensBucket); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &BoltStorage{db: db}, nil
}

func (bs *BoltStorage) SaveUser(ctx context.Context, user User) error {
	key := []byte(strconv.FormatInt(user.EtsyUserID, 10))
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}

	if err := bs.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(usersBucket)
		if err != nil {
			return err
		}

		return bucket.Put(key, value)
	}); err != nil {
		return err
	}

	return nil
}

func (bs *BoltStorage) User(ctx context.Context, etsyUserID int64) (User, error) {
	key := []byte(strconv.FormatInt(etsyUserID, 10))
	user := User{}

	if err := bs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", usersBucket)
		}

		data := bucket.Get(key)
		if len(data) == 0 {
			return ErrNotFound
		}

		return json.Unmarshal(data, &user)
	}); err != nil {
		return User{}, err
	}

	return user, nil
}

func (bs *BoltStorage) TokenDetails(ctx context.Context, id int64) (TokenDetails, error) {
	key := []byte(strconv.FormatInt(id, 10))
	details := TokenDetails{}

	if err := bs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(tokensBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", tokensBucket)
		}

		data := bucket.Get(key)
		if len(data) == 0 {
			return ErrNotFound
		}

		return json.Unmarshal(data, &details)
	}); err != nil {
		return TokenDetails{}, err
	}

	return details, nil
}

func (bs *BoltStorage) SaveTokenDetails(ctx context.Context, td TokenDetails) error {
	key := []byte(strconv.FormatInt(td.ID, 10))
	value, err := json.Marshal(td)
	if err != nil {
		return err
	}

	if err := bs.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(tokensBucket)
		if err != nil {
			return err
		}

		return bucket.Put(key, value)
	}); err != nil {
		return err
	}

	return nil
}

func (bs *BoltStorage) Close() {
	bs.db.Close()
}
