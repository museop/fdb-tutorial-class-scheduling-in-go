package main

import (
	"fmt"
	"log"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
)

func main() {
	// Need to specify the API verison to maintain compatibility even if the API is modified in future versions
	fdb.APIVersion(620)
	db := fdb.MustOpenDefault()

	// Write key-value pair
	_, err := db.Transact(func(tr fdb.Transaction) (ret interface{}, e error) {
		tr.Set(fdb.Key("hello"), []byte("world"))
		return
	})
	if err != nil {
		log.Fatalf("Unable to set FDB database value (%v)", err)
	}

	// Read back the data
	ret, err := db.Transact(func(tr fdb.Transaction) (ret interface{}, e error) {
		ret = tr.Get(fdb.Key("hello")).MustGet()
		return
	})
	if err != nil {
		log.Fatalf("Unable to read FDB database value (%v)", err)
	}

	v := ret.([]byte)
	fmt.Printf("hello, %s\n", string(v))
}
