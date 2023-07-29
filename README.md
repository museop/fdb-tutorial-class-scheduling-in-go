# Class Scheduling in Go

[출처](https://apple.github.io/foundationdb/class-scheduling-go.html)


## Requirements
- `availableClasses()`: returns list of classes
- `signup(studentID, class)`: signs up a student for a class
- `drop(studentID, class)`: drops a student from a class

## Data model

Data model?
-  A method for storing our application data using **keys and values** in FoundationDB.

Two main types of data:
1. a list of classes
2. a record of which students will attend which classes

Represent the key of a key-value pairs as a **Tuple**:
- `("attends", student, class) = ""`: store the key with a *blank* value to indicate that a student is signed up for a particular class
- `("class", class_name) = seatsAvailable`: use `seatsAvilable` to record the number of seats avilable

## Directories and Subspaces

- `scheduling`: Directory(Subspace)
  - `"class"`: subspace
  - `"attends"`: subspace

## Transaction

We use `Transact()` to execute a code block transactionally.

- Using `Database`: `Transact()` automatically creates a transaction and implements a retry loop to ensure that the transaction eventually commits.
- Using `Transaction`: The caller implments appropriate retry logic for errors.

## Transaction and Database Options
- `SetRetryLimit`
- `SetTimeout`
- `SetTransactionRetryLimit`
- `SetTransactionTimeout`

## Idempotence

Idempotent transaction?
- have the some effect if committed twice as if committed once.

## Composing trasactions

Transaction property of atomicity:
- the all-or-north

Function that switch from one class to another:
- By dropping the old class and signing up for the new one inside a single transaction, we ensure that either both steps happen, or thah neigher happens.
- This can be solved with a trasaction calling two transactions.
