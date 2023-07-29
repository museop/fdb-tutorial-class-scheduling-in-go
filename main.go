package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"
)

var courseSS subspace.Subspace
var attendSS subspace.Subspace

var classes []string

// List available classes
func availableClasses(t fdb.Transactor) (ac []string, err error) {
	r, err := t.ReadTransact(func(rtr fdb.ReadTransaction) (interface{}, error) {
		var classes []string

		ri := rtr.GetRange(courseSS, fdb.RangeOptions{}).Iterator()
		for ri.Advance() {
			kv := ri.MustGet()
			v, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				return nil, err
			}
			if v > 0 {
				t, err := courseSS.Unpack(kv.Key)
				if err != nil {
					return nil, err
				}
				classes = append(classes, t[0].(string))
			}
		}
		return classes, nil
	})
	if err == nil {
		ac = r.([]string)
	}
	return
}

// Signing up for a class
func signup(t fdb.Transactor, studentID, class string) (err error) {
	SCKey := attendSS.Pack(tuple.Tuple{studentID, class})
	classKey := courseSS.Pack(tuple.Tuple{class})

	_, err = t.Transact(func(tr fdb.Transaction) (ret interface{}, err error) {
		if tr.Get(SCKey).MustGet() != nil {
			return // already signed up
		}

		seats, err := strconv.ParseInt(string(tr.Get(classKey).MustGet()), 10, 64)
		if err != nil {
			return
		}
		if seats == 0 {
			err = errors.New("no remaining seats")
			return
		}

		classes := tr.GetRange(attendSS.Sub(studentID), fdb.RangeOptions{Mode: fdb.StreamingModeWantAll}).GetSliceOrPanic()
		if len(classes) == 5 {
			err = errors.New("too many classes")
			return
		}

		tr.Set(classKey, []byte(strconv.FormatInt(seats-1, 10)))
		tr.Set(SCKey, []byte{})

		return
	})
	return
}

// Dropping a class
func drop(t fdb.Transactor, studentID, class string) (err error) {
	SCKey := attendSS.Pack(tuple.Tuple{studentID, class})
	classKey := courseSS.Pack(tuple.Tuple{class})

	_, err = t.Transact(func(tr fdb.Transaction) (ret interface{}, err error) {
		if tr.Get(SCKey).MustGet() == nil {
			return // not taking this class
		}

		seats, err := strconv.ParseInt(string(tr.Get(classKey).MustGet()), 10, 64)
		if err != nil {
			return
		}

		tr.Set(classKey, []byte(strconv.FormatInt(seats+1, 10)))
		tr.Clear(SCKey)

		return
	})

	return
}

// Switch class
func swap(t fdb.Transactor, studentID, oldClass, newClass string) (err error) {
	_, err = t.Transact(func(tr fdb.Transaction) (ret interface{}, err error) {
		err = drop(tr, studentID, oldClass)
		if err != nil {
			return
		}
		err = signup(tr, studentID, newClass)
		return
	})
	return
}

func main() {
	// Need to specify the API verison to maintain compatibility even if the API is modified in future versions
	fdb.APIVersion(620)

	db := fdb.MustOpenDefault()
	db.Options().SetTransactionTimeout(60000) // 60,000 ms = 1 minute
	db.Options().SetTransactionRetryLimit(100)

	// Initializing the database
	schedulingDir, err := directory.CreateOrOpen(db, []string{"scheduling"}, nil)
	if err != nil {
		log.Fatal(err)
	}

	courseSS = schedulingDir.Sub("class")
	attendSS = schedulingDir.Sub("attends")

	var levels = []string{"intro", "for dummies", "remedial", "101", "201", "301", "mastery", "lab", "seminar"}
	var types = []string{"chem", "bio", "cs", "geometry", "calc", "alg", "film", "music", "art", "dance"}
	var times = []string{"2:00", "3:00", "4:00", "5:00", "6:00", "7:00", "8:00", "9:00", "10:00", "11:00",
		"12:00", "13:00", "14:00", "15:00", "16:00", "17:00", "18:00", "19:00"}

	classes := make([]string, len(levels)*len(types)*len(times))

	for i := range levels {
		for j := range types {
			for k := range times {
				classes[i*len(types)*len(times)+j*len(times)+k] = fmt.Sprintf("%s %s %s", levels[i], types[j], times[k])
			}
		}
	}

	_, err = db.Transact(func(tr fdb.Transaction) (interface{}, error) {
		tr.ClearRange(schedulingDir)

		for i := range classes {
			tr.Set(courseSS.Pack(tuple.Tuple{classes[i]}), []byte(strconv.FormatInt(100, 10)))
		}

		return nil, nil
	})

	run(db, 10, 10)
}

func indecisiveStudent(db fdb.Database, id, ops int, wg *sync.WaitGroup) {
	studentId := fmt.Sprintf("s%d", id)

	allClasses := classes

	var myClasses []string

	for i := 0; i < ops; i++ {
		var moods []string
		if len(myClasses) > 0 {
			moods = append(moods, "drop", "switch")
		}
		if len(myClasses) < 5 {
			moods = append(moods, "add")
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Need to recheck classes:", r)
					allClasses = []string{}
				}
			}()

			var err error

			if len(allClasses) == 0 {
				allClasses, err = availableClasses(db)
				if err != nil {
					panic(err)
				}
			}

			switch moods[rand.Intn(len(moods))] {
			case "add":
				class := allClasses[rand.Intn(len(allClasses))]
				err = signup(db, studentId, class)
				if err != nil {
					panic(err)
				}
				myClasses = append(myClasses, class)
			case "drop":
				classI := rand.Intn(len(myClasses))
				err = drop(db, studentId, myClasses[classI])
				if err != nil {
					panic(err)
				}
				myClasses[classI], myClasses = myClasses[len(myClasses)-1], myClasses[:len(myClasses)-1]
			case "switch":
				oldClassI := rand.Intn(len(myClasses))
				newClass := allClasses[rand.Intn(len(allClasses))]
				err = swap(db, studentId, myClasses[oldClassI], newClass)
				if err != nil {
					panic(err)
				}
				myClasses[oldClassI] = newClass
			}
		}()
	}

	wg.Done()
}

func run(db fdb.Database, students, opsPerStudent int) {
	var wg sync.WaitGroup

	wg.Add(students)

	for i := 0; i < students; i++ {
		go indecisiveStudent(db, i, opsPerStudent, &wg)
	}

	wg.Wait()

	fmt.Println("Ran", students*opsPerStudent, "transactions")
}
